// Package lossmeter counts records the collector wrote and records it
// lost, into the process's metrics registry.
//
// WHAT A ZERO IS ALLOWED TO MEAN. This is the decision the package
// exists to make, and it is the repository's own rule in metric form:
// an absent series is not a zero.
//
// A Prometheus counter created with WithLabelValues only appears once
// something increments it. So a deployment that has never dropped a
// record and a deployment whose drop counter was never wired both
// expose nothing at all — and the second is exactly the failure this
// phase was opened to remove. Every label combination that CAN occur is
// therefore declared at construction, at zero, before the service
// serves anything.
//
// After that, a zero means measured-zero. A missing series means
// somebody added a kind or a reason and did not declare it, which is a
// bug in this file rather than an unremarkable quiet day.
package lossmeter

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/FabioCaffarello/ghost-trace/libs/metrics"
	"github.com/FabioCaffarello/ghost-trace/services/collector/internal/app"
)

// Meter implements app.LossMeter.
type Meter struct {
	written *prometheus.CounterVec
	dropped *prometheus.CounterVec
}

// New registers the counters and declares every series they can carry.
func New(reg *metrics.Registry) *Meter {
	m := &Meter{
		written: reg.Counter("records_written_total",
			"Records the collector handed to a store, by kind.", "kind"),
		dropped: reg.Counter("records_dropped_total",
			"Records the collector could not hand to a store and will not retry, "+
				"by kind and reason.", "kind", "reason"),
	}

	// The declaration. Adding zero creates the series without changing
	// it, so a scrape taken one second after startup already shows every
	// counter this process can ever report.
	for _, kind := range app.Kinds {
		m.written.WithLabelValues(kind).Add(0)
		for _, reason := range app.Reasons {
			m.dropped.WithLabelValues(kind, reason).Add(0)
		}
	}
	return m
}

// Written counts a record that reached its store.
func (m *Meter) Written(kind string) { m.written.WithLabelValues(kind).Inc() }

// Dropped counts a record that did not, and is gone.
func (m *Meter) Dropped(kind, reason string) {
	m.dropped.WithLabelValues(kind, reason).Inc()
}
