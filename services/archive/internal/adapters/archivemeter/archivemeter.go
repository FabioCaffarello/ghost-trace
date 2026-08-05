// Package archivemeter exposes what the archive commits, refuses, and
// is behind by.
//
// TWO KINDS OF NUMBER LIVE HERE, and they fail differently.
//
// The counters are local and monotonic: this process committed a
// record, or refused one, and nothing outside can make that untrue.
// They are declared at zero for the same reason the collector's are —
// an absent series is not a zero.
//
// The lag gauges are neither. They are the BROKER's view, read over the
// network on a timer, and the interesting question is what they say
// when that read fails.
//
//   - Zero would be a lie that reads as good news: "nothing is
//     pending" at the exact moment nothing is known.
//   - Holding the last value silently is a quieter lie: "still true",
//     with no way to tell how long ago it was.
//
// So the last value is held AND `stream_observed_timestamp_seconds`
// stops advancing. `time() - observed` is then the age of the reading,
// and a stale lag figure announces itself instead of impersonating a
// fresh one. `stream_read_failures_total` counts the failed polls, so
// the condition is visible without arithmetic too.
//
// COUNTERS ARE PROCESS-LOCAL AND RESET ON RESTART. That is ordinary
// Prometheus semantics and a scraper handles it, but it matters for the
// reconciliation this phase is building: "committed" answers what THIS
// process committed, not what the archive holds. Restarting the archive
// mid-backlog was observed to take committed from 5 to 10 while fifteen
// records had in fact arrived. The reconciliation in 3.6 must read the
// substrate, or read counters over a window with no restart in it.
package archivemeter

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/FabioCaffarello/ghost-trace/libs/eventstream"
	"github.com/FabioCaffarello/ghost-trace/libs/metrics"
	"github.com/FabioCaffarello/ghost-trace/services/archive/internal/consumer"
)

// Meter implements consumer.Meter and records stream progress.
type Meter struct {
	committed *prometheus.CounterVec
	rejected  *prometheus.CounterVec

	pending     prometheus.Gauge
	ackPending  prometheus.Gauge
	redelivered prometheus.Gauge
	observed    prometheus.Gauge
	readFailed  prometheus.Counter

	now func() time.Time
}

// New registers everything and declares every series that can appear.
func New(reg *metrics.Registry, now func() time.Time) *Meter {
	if now == nil {
		now = time.Now
	}
	m := &Meter{
		committed: reg.Counter("archive_committed_total",
			"Records committed to the substrate, by message type.", "message_type"),
		rejected: reg.Counter("archive_rejected_total",
			"Records refused and not retried, by reason.", "reason"),
		pending: reg.Gauge("archive_stream_pending",
			"Records the broker says are accepted and not yet acknowledged by this "+
				"consumer. Held at its last value when the broker cannot be read; see "+
				"archive_stream_observed_timestamp_seconds.").WithLabelValues(),
		ackPending: reg.Gauge("archive_stream_ack_pending",
			"Records delivered and awaiting acknowledgement right now.").WithLabelValues(),
		redelivered: reg.Gauge("archive_stream_redelivered",
			"Messages this consumer has been sent more than once. Non-zero is the "+
				"at-least-once mechanism working; growing means commits are failing.").
			WithLabelValues(),
		observed: reg.Gauge("archive_stream_observed_timestamp_seconds",
			"When the stream gauges above were last read successfully. Subtract from "+
				"time() for their age; it stops advancing while the broker is "+
				"unreachable.").WithLabelValues(),
		readFailed: reg.Counter("archive_stream_read_failures_total",
			"Polls of the broker that did not return a reading.").WithLabelValues(),
		now: now,
	}

	// Declare. Adding zero creates a series without changing it, so a
	// scrape taken immediately after startup already carries every
	// counter this process can report.
	for _, t := range eventstream.MessageTypes() {
		m.committed.WithLabelValues(t).Add(0)
	}
	for _, r := range consumer.RejectReasons {
		m.rejected.WithLabelValues(r).Add(0)
	}
	return m
}

// Committed counts a record that reached the substrate.
func (m *Meter) Committed(messageType string) {
	m.committed.WithLabelValues(messageType).Inc()
}

// Rejected counts a record refused and not retried.
func (m *Meter) Rejected(reason string) { m.rejected.WithLabelValues(reason).Inc() }

// Observe records one poll of the broker.
//
// On failure the lag gauges are deliberately left alone: a reading that
// could not be taken is not a reading of zero, and overwriting a real
// figure with a guess would destroy the only number that says how far
// behind this service is.
func (m *Meter) Observe(s eventstream.Stats, err error) {
	if err != nil {
		m.readFailed.Inc()
		return
	}
	m.pending.Set(float64(s.Pending))
	m.ackPending.Set(float64(s.AckPending))
	m.redelivered.Set(float64(s.Redelivered))
	m.observed.Set(float64(m.now().Unix()))
}
