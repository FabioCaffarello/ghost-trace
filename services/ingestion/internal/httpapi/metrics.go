package httpapi

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Metrics is the in-process httpapi metrics registry per decision-log
// §0199 third observability instrumentation advance + §0200 fourth
// advance (request-duration histogram). Tracks per-(path, status)
// counts + per-path duration histograms; exposes both in Prometheus
// text exposition format via the /metrics endpoint.
//
// Stdlib-only — no `prometheus/client_golang` dependency. Per §0199 MO1
// threshold (≤2 metric types AND ≤3 labels): 2 metric types
// (counter + histogram), counter has 2 labels (path + status),
// histogram has 1 explicit label (path) + 1 implicit (`le` for
// buckets) = 2 effective labels. Within threshold — stdlib-only
// implementation retained.
//
// Counter name: `ghosttrace_httpapi_requests_total`; labels: path + status.
// Histogram name: `ghosttrace_httpapi_request_duration_ms`; labels: path.
// Bucket boundaries: §0200 default per latencyBucketsMs (12 buckets +
// implicit +Inf) covering typical httpapi latency spectrum
// (sub-ms healthz probes → multi-second substrate-walk endpoints).
//
// Concurrency: safe for concurrent Inc + Observe calls; the encoder
// snapshot acquires the lock once + copies all state out before
// encoding, preventing reader/writer contention during scrape.
type Metrics struct {
	mu sync.Mutex
	// counters: key = "path|status" (pipe as separator since neither
	// path nor numeric status can contain `|`).
	counters map[counterKey]uint64

	// histograms: key = path; value = per-bucket cumulative counts
	// matching latencyBucketsMs index-by-index PLUS one slot for the
	// implicit +Inf bucket (totalCount + sumMs maintained separately).
	histograms map[string]*histogramState
}

// latencyBucketsMs are the histogram bucket upper-bound boundaries in
// milliseconds per §0200 default. Covers typical httpapi latency
// spectrum from sub-ms (healthz) to multi-second (substrate walks).
// Strictly ascending; +Inf is implicit (any observation greater than
// the last bucket falls into the +Inf bucket only).
//
// Operators wanting different boundaries can override via a future
// WithMetricsHistogramBuckets option (deferred — current default
// covers all currently-known operational scenarios).
var latencyBucketsMs = []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000}

type counterKey struct {
	path   string
	status int
}

// histogramState carries the per-path duration-bucket counts +
// aggregate sum/count for Prometheus histogram emission.
// bucketCounts[i] is the count of observations <= latencyBucketsMs[i]
// (NOT cumulative at the field level; cumulation happens at encode
// time per Prometheus convention). totalCount counts all observations
// including those exceeding the last bucket boundary (covers the
// implicit +Inf bucket).
type histogramState struct {
	bucketCounts []uint64
	totalCount   uint64
	sumMs        float64
}

func newHistogramState() *histogramState {
	return &histogramState{
		bucketCounts: make([]uint64, len(latencyBucketsMs)),
	}
}

// NewMetrics constructs a fresh, empty Metrics registry. Production
// main wires one Metrics per Handler via WithMetrics; tests construct
// fresh ones per-test to avoid cross-test pollution.
func NewMetrics() *Metrics {
	return &Metrics{
		counters:   make(map[counterKey]uint64),
		histograms: make(map[string]*histogramState),
	}
}

// Inc increments the per-(path, status) counter by 1.
func (m *Metrics) Inc(path string, status int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[counterKey{path: path, status: status}]++
}

// Observe records a per-path request-duration observation in ms.
// Per §0200 first histogram-advance: durations are observed in
// milliseconds (matching the §0197 structured entry's duration_ms
// field). Negative observations are treated as 0 (defensive — caller
// SHOULD pass non-negative; the clamp prevents bucket misclassification).
func (m *Metrics) Observe(path string, durationMs float64) {
	if durationMs < 0 {
		durationMs = 0
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	h, ok := m.histograms[path]
	if !ok {
		h = newHistogramState()
		m.histograms[path] = h
	}
	for i, upper := range latencyBucketsMs {
		if durationMs <= upper {
			h.bucketCounts[i]++
		}
	}
	h.totalCount++
	h.sumMs += durationMs
}

// snapshot returns a stable copy of the counters + histograms for
// encoding. Holding the lock only across the copy minimizes contention.
func (m *Metrics) snapshot() (map[counterKey]uint64, map[string]*histogramState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	counters := make(map[counterKey]uint64, len(m.counters))
	for k, v := range m.counters {
		counters[k] = v
	}
	histograms := make(map[string]*histogramState, len(m.histograms))
	for path, h := range m.histograms {
		hc := newHistogramState()
		copy(hc.bucketCounts, h.bucketCounts)
		hc.totalCount = h.totalCount
		hc.sumMs = h.sumMs
		histograms[path] = hc
	}
	return counters, histograms
}

// Encode writes the registry to w in Prometheus text exposition format
// (per the de-facto convention used by `prometheus/client_golang`).
// Output is deterministic: counter samples sorted by (path, status)
// ascending; histogram samples grouped per-path with buckets in
// ascending boundary order. Successive scrapes against an unchanged
// registry produce byte-identical output (helpful for downstream
// diff-aware tooling).
//
// Format example:
//
//	# HELP ghosttrace_httpapi_requests_total Total httpapi requests by route + status.
//	# TYPE ghosttrace_httpapi_requests_total counter
//	ghosttrace_httpapi_requests_total{path="/healthz",status="200"} 5
//	# HELP ghosttrace_httpapi_request_duration_ms Request duration in ms by route.
//	# TYPE ghosttrace_httpapi_request_duration_ms histogram
//	ghosttrace_httpapi_request_duration_ms_bucket{path="/healthz",le="1"} 5
//	ghosttrace_httpapi_request_duration_ms_bucket{path="/healthz",le="+Inf"} 5
//	ghosttrace_httpapi_request_duration_ms_sum{path="/healthz"} 2.5
//	ghosttrace_httpapi_request_duration_ms_count{path="/healthz"} 5
func (m *Metrics) Encode(w io.Writer) error {
	counters, histograms := m.snapshot()

	var sb strings.Builder

	// Counter emission.
	counterKeys := make([]counterKey, 0, len(counters))
	for k := range counters {
		counterKeys = append(counterKeys, k)
	}
	sort.Slice(counterKeys, func(i, j int) bool {
		if counterKeys[i].path != counterKeys[j].path {
			return counterKeys[i].path < counterKeys[j].path
		}
		return counterKeys[i].status < counterKeys[j].status
	})
	sb.WriteString("# HELP ghosttrace_httpapi_requests_total Total httpapi requests by route + status.\n")
	sb.WriteString("# TYPE ghosttrace_httpapi_requests_total counter\n")
	for _, k := range counterKeys {
		sb.WriteString(`ghosttrace_httpapi_requests_total{path="`)
		sb.WriteString(escapeLabelValue(k.path))
		sb.WriteString(`",status="`)
		sb.WriteString(strconv.Itoa(k.status))
		sb.WriteString(`"} `)
		sb.WriteString(strconv.FormatUint(counters[k], 10))
		sb.WriteString("\n")
	}

	// Histogram emission.
	histPaths := make([]string, 0, len(histograms))
	for p := range histograms {
		histPaths = append(histPaths, p)
	}
	sort.Strings(histPaths)
	sb.WriteString("# HELP ghosttrace_httpapi_request_duration_ms Request duration in ms by route.\n")
	sb.WriteString("# TYPE ghosttrace_httpapi_request_duration_ms histogram\n")
	for _, p := range histPaths {
		h := histograms[p]
		escPath := escapeLabelValue(p)
		// Buckets per Prometheus convention: cumulative counts ascending
		// by upper bound. Each bucket line carries the count of
		// observations <= that upper bound.
		var cumulative uint64
		for i, upper := range latencyBucketsMs {
			cumulative = h.bucketCounts[i] // bucket-counts already <= upper-bound, but the index-wise count is exclusive vs lower.
			// Note: our Observe loop increments h.bucketCounts[i] for every
			// observation <= latencyBucketsMs[i] — so bucketCounts[i] is
			// ALREADY the cumulative count <= upper. No further cumulation
			// needed.
			_ = cumulative
			sb.WriteString(`ghosttrace_httpapi_request_duration_ms_bucket{path="`)
			sb.WriteString(escPath)
			sb.WriteString(`",le="`)
			sb.WriteString(strconv.FormatFloat(upper, 'f', -1, 64))
			sb.WriteString(`"} `)
			sb.WriteString(strconv.FormatUint(h.bucketCounts[i], 10))
			sb.WriteString("\n")
		}
		// +Inf bucket: total count (catches observations exceeding the
		// last bucket boundary).
		sb.WriteString(`ghosttrace_httpapi_request_duration_ms_bucket{path="`)
		sb.WriteString(escPath)
		sb.WriteString(`",le="+Inf"} `)
		sb.WriteString(strconv.FormatUint(h.totalCount, 10))
		sb.WriteString("\n")
		// _sum and _count required per Prometheus histogram convention.
		sb.WriteString(`ghosttrace_httpapi_request_duration_ms_sum{path="`)
		sb.WriteString(escPath)
		sb.WriteString(`"} `)
		sb.WriteString(strconv.FormatFloat(h.sumMs, 'f', -1, 64))
		sb.WriteString("\n")
		sb.WriteString(`ghosttrace_httpapi_request_duration_ms_count{path="`)
		sb.WriteString(escPath)
		sb.WriteString(`"} `)
		sb.WriteString(strconv.FormatUint(h.totalCount, 10))
		sb.WriteString("\n")
	}

	_, err := io.WriteString(w, sb.String())
	return err
}

// escapeLabelValue applies the Prometheus text-format label-value
// escaping rules: backslash → `\\`, double-quote → `\"`, newline →
// `\n`. Per the exposition-format spec at
// https://prometheus.io/docs/instrumenting/exposition_formats/
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

// handleMetrics serves GET /metrics per §0199 — Prometheus text-format
// scrape endpoint. Content-Type follows the de-facto Prometheus
// convention (`text/plain; version=0.0.4; charset=utf-8`); operator
// scrapers consume per the standard exposition format.
//
// Returns 503 when metrics are not configured on the handler
// (WithMetrics not supplied); 405 on non-GET.
func (h *Handler) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.metrics == nil {
		writeIngestError(w, http.StatusServiceUnavailable,
			"metrics endpoint not configured (handler constructed without WithMetrics)")
		return
	}
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	if err := h.metrics.Encode(w); err != nil {
		writeIngestError(w, http.StatusInternalServerError, fmt.Sprintf("metrics.Encode: %v", err))
		return
	}
}
