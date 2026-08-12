package metrics

// Records written, and records lost — the same two counters wherever a
// service hands something to a store it does not control.
//
// WHAT A ZERO IS ALLOWED TO MEAN. This is the decision the type exists
// to carry, and it is the repository's own rule in metric form: an
// absent series is not a zero.
//
// A Prometheus counter created with WithLabelValues only appears once
// something increments it. So a deployment that has never dropped a
// record and a deployment whose drop counter was never wired both
// expose nothing at all — and the second is exactly the failure Phase 3
// was opened to remove. Every label combination that CAN occur is
// declared at construction, at zero, before the service serves
// anything. After that a zero means measured-zero, and a missing series
// means somebody added a kind or a reason and did not declare it.
//
// It lives here rather than beside one service because there are two
// now. The collector counts what it publishes; the decision engine
// counts the evaluations it archives off the decision path, and that
// path went a phase with no counter at all — /v1/decisions could lose
// every record it produced and `/metrics` would look identical.
// A second copy of the declared-at-zero doctrine is a second chance to
// get it wrong.

import "github.com/prometheus/client_golang/prometheus"

// Loss counts records handed to a store and records that never got
// there. It satisfies any consumer-defined port with these two methods,
// which is how one implementation serves both services without either
// importing the other.
type Loss struct {
	written *prometheus.CounterVec
	dropped *prometheus.CounterVec
}

// NewLoss registers the counters and declares the full cross product of
// kinds and reasons at zero.
//
// Both lists are the CALLER's, because what a record is and how it can
// be lost are properties of the application, not of the meter. Passing
// a short list is how a series goes undeclared, which is the one
// mistake this type is built to prevent — so keep them as exported
// package-level values, not literals at the call site.
func NewLoss(reg *Registry, kinds, reasons []string) *Loss {
	m := &Loss{
		written: reg.Counter("records_written_total",
			"Records handed to a store, by kind.", "kind"),
		dropped: reg.Counter("records_dropped_total",
			"Records that could not be handed to a store and will not be retried, "+
				"by kind and reason.", "kind", "reason"),
	}
	// Adding zero creates the series without changing it, so a scrape
	// taken one second after startup already shows every counter this
	// process can ever report.
	for _, kind := range kinds {
		m.written.WithLabelValues(kind).Add(0)
		for _, reason := range reasons {
			m.dropped.WithLabelValues(kind, reason).Add(0)
		}
	}
	return m
}

// Written counts a record that reached its store.
func (m *Loss) Written(kind string) { m.written.WithLabelValues(kind).Inc() }

// Dropped counts a record that did not, and is gone.
func (m *Loss) Dropped(kind, reason string) {
	m.dropped.WithLabelValues(kind, reason).Inc()
}
