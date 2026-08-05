package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/FabioCaffarello/ghost-trace/libs/metrics"
)

// Metrics is the HTTP half of a service's instrumentation: a request
// counter by (route, status) and a duration histogram by route.
//
// It used to own a hand-written registry and exposition encoder. Both
// are gone: the series register into the process's one libs/metrics
// registry, which also carries the domain counters, and /metrics is
// served from there. Two registries behind one endpoint would need two
// encoders and would silently drop whichever the handler forgot.
//
// The exposition is UNCHANGED, and not merely compatible: the same
// series, the same label names, the same label order, byte for byte
// for these families. That was checked rather than assumed — the
// expectation going in was that the library would sort labels and
// reorder `le`, and it does not, because `le` is appended as a special
// case. TestExpositionDidNotChange holds it there.
type Metrics struct {
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

// BucketsMs are the histogram bounds in milliseconds, strictly
// ascending, +Inf implicit.
//
// Chosen rather than defaulted. They span a sub-millisecond healthz
// probe to well past the 80ms decision budget, because a histogram
// whose bounds do not straddle what is being measured puts every
// observation in one bucket and looks like it is working.
var BucketsMs = []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000}

// NewMetrics registers the HTTP series into reg.
func NewMetrics(reg *metrics.Registry) *Metrics {
	return &Metrics{
		requests: reg.Counter("http_requests_total",
			"Total HTTP requests by route and status.", "route", "status"),
		duration: reg.Histogram("http_request_duration_ms",
			"Request duration in milliseconds by route.", BucketsMs, "route"),
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
			m.requests.WithLabelValues(route, strconv.Itoa(sr.effectiveStatus())).Inc()

			ms := float64(time.Since(start).Microseconds()) / 1000.0
			if ms < 0 {
				ms = 0
			}
			m.duration.WithLabelValues(route).Observe(ms)
		})
	}
}
