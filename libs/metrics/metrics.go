// Package metrics is the one registry a service exposes at /metrics.
//
// WHY A LIBRARY AND NOT MORE HAND-ROLLED EXPOSITION. The stdlib
// encoder this replaces said, in its own doc comment, where its limit
// was: "two metric types with two effective labels each is under the
// threshold where prometheus/client_golang starts paying for itself;
// the moment a third metric type or a dynamic label is wanted, adopt
// the library instead of growing this." Phase 3 crosses that threshold
// on both counts — it needs gauges for consumer lag and a counter
// labelled by drop reason.
//
// The deciding argument is what the phase is FOR. It exists to make
// what the system loses countable, and hand-writing the counting
// apparatus — label cardinality, escaping, concurrent maps, exposition
// edge cases — would leave the instrument as the least-tested part of
// the measurement. See ADR-0007, which records this as a deliberate
// exception to ADR-0001's stdlib-only preference for shared libraries.
//
// Naming: every series is prefixed `ghosttrace_`. The prefix is applied
// here rather than trusted to each call site, because a series that
// escapes with the wrong name is invisible to every dashboard and query
// that was written expecting it.
package metrics

import (
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Prefix is on every series this repository publishes.
const Prefix = "ghosttrace_"

// Registry is a service's metrics, HTTP and domain alike.
//
// One registry per process, exposed once. Two registries behind one
// endpoint would need two encoders and would silently drop whichever
// one the handler forgot.
type Registry struct {
	reg *prometheus.Registry
}

// New returns an empty registry.
//
// Deliberately NOT prometheus.DefaultRegisterer: a global registry
// makes a test's metrics visible to the next test and makes two
// services in one process share series. Everything here is passed
// explicitly.
func New() *Registry {
	return &Registry{reg: prometheus.NewRegistry()}
}

// Handler serves the exposition format.
func (r *Registry) Handler() http.Handler {
	return promhttp.HandlerFor(r.reg, promhttp.HandlerOpts{})
}

// Gatherer exposes the underlying registry for tests that want to read
// series back rather than parse text.
func (r *Registry) Gatherer() prometheus.Gatherer { return r.reg }

func name(n string) string {
	if strings.HasPrefix(n, Prefix) {
		return n
	}
	return Prefix + n
}

// Counter registers a counter, or returns the one already registered
// under the same name and labels.
//
// Re-registration returns the existing collector rather than panicking,
// which is what prometheus.MustRegister would do. A duplicate
// registration is a composition mistake, and taking a service down at
// boot for one is a worse outcome than handing back the collector the
// caller was going to use anyway.
func (r *Registry) Counter(n, help string, labels ...string) *prometheus.CounterVec {
	c := prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: name(n), Help: help}, labels)
	if err := r.reg.Register(c); err != nil {
		var already prometheus.AlreadyRegisteredError
		if ok := asAlreadyRegistered(err, &already); ok {
			if existing, ok := already.ExistingCollector.(*prometheus.CounterVec); ok {
				return existing
			}
		}
		panic(err)
	}
	return c
}

// Gauge registers a gauge, with the same re-registration behaviour.
func (r *Registry) Gauge(n, help string, labels ...string) *prometheus.GaugeVec {
	g := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Name: name(n), Help: help}, labels)
	if err := r.reg.Register(g); err != nil {
		var already prometheus.AlreadyRegisteredError
		if ok := asAlreadyRegistered(err, &already); ok {
			if existing, ok := already.ExistingCollector.(*prometheus.GaugeVec); ok {
				return existing
			}
		}
		panic(err)
	}
	return g
}

// Histogram registers a histogram with explicit bucket bounds.
//
// Buckets are never defaulted here. prometheus.DefBuckets spans 5ms to
// 10s, which describes neither a sub-millisecond decision nor a
// seven-second experiment run; a histogram with the wrong bounds
// reports every observation in one bucket and looks like it is working.
func (r *Registry) Histogram(n, help string, buckets []float64, labels ...string) *prometheus.HistogramVec {
	h := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: name(n), Help: help, Buckets: buckets}, labels)
	if err := r.reg.Register(h); err != nil {
		var already prometheus.AlreadyRegisteredError
		if ok := asAlreadyRegistered(err, &already); ok {
			if existing, ok := already.ExistingCollector.(*prometheus.HistogramVec); ok {
				return existing
			}
		}
		panic(err)
	}
	return h
}

func asAlreadyRegistered(err error, target *prometheus.AlreadyRegisteredError) bool {
	if e, ok := err.(prometheus.AlreadyRegisteredError); ok {
		*target = e
		return true
	}
	return false
}
