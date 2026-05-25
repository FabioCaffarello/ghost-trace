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

// Metrics is the in-process httpapi request counter registry per
// decision-log §0199 third observability instrumentation advance.
// Tracks per-(path, status) counts; exposes them in Prometheus text
// exposition format via the /metrics endpoint.
//
// Stdlib-only — no `prometheus/client_golang` dependency. The narrow
// surface (one counter, no histograms/gauges) covers the §0199 scope;
// future advances may extend with histograms (request duration buckets)
// or per-tier label without depending on the full prometheus client.
//
// Counter name: `ghosttrace_httpapi_requests_total` (singular metric,
// no per-route definition burden); labels: `path` + `status`.
//
// Concurrency: Metrics is safe for concurrent Inc calls; the encoder
// snapshot acquires a lock + copies counters out before encoding,
// preventing reader/writer contention during scrape.
type Metrics struct {
	mu sync.Mutex
	// counters: key = "path|status" (pipe as separator since neither
	// path nor numeric status can contain `|`).
	counters map[counterKey]uint64
}

type counterKey struct {
	path   string
	status int
}

// NewMetrics constructs a fresh, empty Metrics registry. Production
// main wires one Metrics per Handler via WithMetrics; tests construct
// fresh ones per-test to avoid cross-test counter pollution.
func NewMetrics() *Metrics {
	return &Metrics{counters: make(map[counterKey]uint64)}
}

// Inc increments the per-(path, status) counter by 1.
func (m *Metrics) Inc(path string, status int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[counterKey{path: path, status: status}]++
}

// snapshot returns a stable copy of the counters map for encoding.
// Holding the lock only across the copy minimizes contention.
func (m *Metrics) snapshot() map[counterKey]uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[counterKey]uint64, len(m.counters))
	for k, v := range m.counters {
		out[k] = v
	}
	return out
}

// Encode writes the registry to w in Prometheus text exposition format
// (per the de-facto convention used by `prometheus/client_golang`).
// Output is deterministic: lines sorted by (path, status) ascending so
// successive scrapes against an unchanged registry produce byte-identical
// output (helpful for downstream diff-aware tooling).
//
// Format example:
//
//	# HELP ghosttrace_httpapi_requests_total Total httpapi requests by route + status.
//	# TYPE ghosttrace_httpapi_requests_total counter
//	ghosttrace_httpapi_requests_total{path="/healthz",status="200"} 5
//	ghosttrace_httpapi_requests_total{path="/v1/morphology",status="200"} 12
func (m *Metrics) Encode(w io.Writer) error {
	snap := m.snapshot()
	keys := make([]counterKey, 0, len(snap))
	for k := range snap {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].path != keys[j].path {
			return keys[i].path < keys[j].path
		}
		return keys[i].status < keys[j].status
	})

	var sb strings.Builder
	sb.WriteString("# HELP ghosttrace_httpapi_requests_total Total httpapi requests by route + status.\n")
	sb.WriteString("# TYPE ghosttrace_httpapi_requests_total counter\n")
	for _, k := range keys {
		sb.WriteString(`ghosttrace_httpapi_requests_total{path="`)
		sb.WriteString(escapeLabelValue(k.path))
		sb.WriteString(`",status="`)
		sb.WriteString(strconv.Itoa(k.status))
		sb.WriteString(`"} `)
		sb.WriteString(strconv.FormatUint(snap[k], 10))
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
