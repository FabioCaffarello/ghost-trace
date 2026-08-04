package httpmw

import (
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Metrics is the in-process registry: a request counter by (route,
// status) and a duration histogram by route, exposed in Prometheus
// text exposition format.
//
// Stdlib-only on purpose. Two metric types with two effective labels
// each is under the threshold where prometheus/client_golang starts
// paying for itself; the moment a third metric type or a dynamic label
// is wanted, adopt the library instead of growing this.
type Metrics struct {
	mu         sync.Mutex
	counters   map[counterKey]uint64
	histograms map[string]*histogramState

	// buckets are the histogram upper bounds in milliseconds, strictly
	// ascending, +Inf implicit. The defaults span sub-ms healthz probes
	// to the 80ms decision budget and beyond.
	buckets []float64
}

var defaultBucketsMs = []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000}

type counterKey struct {
	route  string
	status int
}

// histogramState: bucketCounts[i] is the count of observations
// <= buckets[i] — cumulative by construction, matching what the
// Prometheus bucket lines want. totalCount includes observations past
// the last bound (the implicit +Inf bucket).
type histogramState struct {
	bucketCounts []uint64
	totalCount   uint64
	sumMs        float64
}

// NewMetrics constructs an empty registry.
func NewMetrics() *Metrics {
	return &Metrics{
		counters:   make(map[counterKey]uint64),
		histograms: make(map[string]*histogramState),
		buckets:    defaultBucketsMs,
	}
}

// Collect is the middleware: it counts every completed request by
// (route, status) and observes its duration. Innermost in the chain so
// it measures the handler, not the logging around it.
func (m *Metrics) Collect() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sr := &statusRecorder{ResponseWriter: w}
			start := time.Now()
			next.ServeHTTP(sr, r)
			route := routeLabel(r)
			m.inc(route, sr.effectiveStatus())
			m.observe(route, float64(time.Since(start).Microseconds())/1000.0)
		})
	}
}

func (m *Metrics) inc(route string, status int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[counterKey{route: route, status: status}]++
}

func (m *Metrics) observe(route string, durationMs float64) {
	if durationMs < 0 {
		durationMs = 0
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	h, ok := m.histograms[route]
	if !ok {
		h = &histogramState{bucketCounts: make([]uint64, len(m.buckets))}
		m.histograms[route] = h
	}
	for i, upper := range m.buckets {
		if durationMs <= upper {
			h.bucketCounts[i]++
		}
	}
	h.totalCount++
	h.sumMs += durationMs
}

// snapshot copies state out under the lock so encoding never contends
// with request traffic.
func (m *Metrics) snapshot() (map[counterKey]uint64, map[string]*histogramState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	counters := make(map[counterKey]uint64, len(m.counters))
	for k, v := range m.counters {
		counters[k] = v
	}
	histograms := make(map[string]*histogramState, len(m.histograms))
	for route, h := range m.histograms {
		hc := &histogramState{
			bucketCounts: make([]uint64, len(h.bucketCounts)),
			totalCount:   h.totalCount,
			sumMs:        h.sumMs,
		}
		copy(hc.bucketCounts, h.bucketCounts)
		histograms[route] = hc
	}
	return counters, histograms
}

// Encode writes the registry in Prometheus text exposition format.
// Output is deterministic (sorted by route, then status) so successive
// scrapes of an unchanged registry are byte-identical.
func (m *Metrics) Encode(w io.Writer) error {
	counters, histograms := m.snapshot()

	var sb strings.Builder

	keys := make([]counterKey, 0, len(counters))
	for k := range counters {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].route != keys[j].route {
			return keys[i].route < keys[j].route
		}
		return keys[i].status < keys[j].status
	})
	sb.WriteString("# HELP ghosttrace_http_requests_total Total HTTP requests by route and status.\n")
	sb.WriteString("# TYPE ghosttrace_http_requests_total counter\n")
	for _, k := range keys {
		sb.WriteString(`ghosttrace_http_requests_total{route="` + escapeLabelValue(k.route) +
			`",status="` + strconv.Itoa(k.status) + `"} `)
		sb.WriteString(strconv.FormatUint(counters[k], 10))
		sb.WriteString("\n")
	}

	routes := make([]string, 0, len(histograms))
	for p := range histograms {
		routes = append(routes, p)
	}
	sort.Strings(routes)
	sb.WriteString("# HELP ghosttrace_http_request_duration_ms Request duration in milliseconds by route.\n")
	sb.WriteString("# TYPE ghosttrace_http_request_duration_ms histogram\n")
	for _, route := range routes {
		h := histograms[route]
		esc := escapeLabelValue(route)
		for i, upper := range m.buckets {
			sb.WriteString(`ghosttrace_http_request_duration_ms_bucket{route="` + esc +
				`",le="` + strconv.FormatFloat(upper, 'f', -1, 64) + `"} `)
			sb.WriteString(strconv.FormatUint(h.bucketCounts[i], 10))
			sb.WriteString("\n")
		}
		sb.WriteString(`ghosttrace_http_request_duration_ms_bucket{route="` + esc + `",le="+Inf"} `)
		sb.WriteString(strconv.FormatUint(h.totalCount, 10))
		sb.WriteString("\n")
		sb.WriteString(`ghosttrace_http_request_duration_ms_sum{route="` + esc + `"} `)
		sb.WriteString(strconv.FormatFloat(h.sumMs, 'f', -1, 64))
		sb.WriteString("\n")
		sb.WriteString(`ghosttrace_http_request_duration_ms_count{route="` + esc + `"} `)
		sb.WriteString(strconv.FormatUint(h.totalCount, 10))
		sb.WriteString("\n")
	}

	_, err := io.WriteString(w, sb.String())
	return err
}

// Handler serves the scrape endpoint.
func (m *Metrics) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		if err := m.Encode(w); err != nil {
			http.Error(w, "metrics encode failed", http.StatusInternalServerError)
		}
	})
}

// escapeLabelValue applies the exposition-format escaping rules:
// backslash, double-quote and newline.
func escapeLabelValue(s string) string {
	if !strings.ContainsAny(s, `\"`+"\n") {
		return s
	}
	var sb strings.Builder
	sb.Grow(len(s))
	for _, r := range s {
		switch r {
		case '\\':
			sb.WriteString(`\\`)
		case '"':
			sb.WriteString(`\"`)
		case '\n':
			sb.WriteString(`\n`)
		default:
			sb.WriteRune(r)
		}
	}
	return sb.String()
}
