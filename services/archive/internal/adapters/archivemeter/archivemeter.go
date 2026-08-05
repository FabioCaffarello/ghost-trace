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
// AGE-OUT IS THE THIRD LOSS PATH, and it is the one that improves the
// other numbers as it happens: a record discarded for age stops being
// pending, so archive_stream_pending goes DOWN at the moment of loss. A
// backlog that vanished looks exactly like a backlog that drained.
//
// IT CANNOT BE COUNTED AFTER THE FACT with what the broker retains.
// Two ways were built and both were deleted after measurement:
//
//   - stream first sequence against consumer ack floor. The broker
//     advances the ack floor when messages are removed from under a
//     consumer, because there is nothing left to acknowledge. Purging
//     four unconsumed records left first_seq=15, ack_floor=14.
//   - jumps in delivered sequence. A jump needs records to leave
//     without being delivered, and a keeping-up consumer never sees
//     one; a test built to force it delivered everything before the
//     purge. The case it would catch — a consumer far enough behind
//     that discard overtakes it — needs load nobody here has built.
//
// Neither ships. A counter that reads zero through real loss is worse
// than no counter, which is the principle this phase rests on.
//
// WHAT SHIPS IS THE EARLY WARNING, which is the half of the requirement
// that can be honestly met: archive_stream_oldest_message_age_seconds
// against archive_stream_max_age_seconds says how close the stream's
// oldest content is to the retention edge, while there is still time to
// act. Read with archive_stream_pending it answers the question age-out
// poses, before rather than after.
//
// Closing the after-the-fact half needs the archive to remember its own
// high-water mark durably and compare on startup. That is a design with
// a hot-path cost and belongs with 3.6, which needs a durable position
// for reconciliation anyway.
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

	oldestAge prometheus.Gauge
	maxAge    prometheus.Gauge

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
		oldestAge: reg.Gauge("archive_stream_oldest_message_age_seconds",
			"Age of the oldest message still in the stream. An upper bound on the age "+
				"of the oldest record this consumer has not acknowledged; read with "+
				"archive_stream_pending and archive_stream_max_age_seconds to see how "+
				"close a backlog is to being discarded.").WithLabelValues(),
		maxAge: reg.Gauge("archive_stream_max_age_seconds",
			"The stream's retention window, so the ratio that matters can be computed "+
				"rather than hardcoded into a dashboard.").WithLabelValues(),
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
	m.oldestAge.Set(s.OldestAge.Seconds())
	m.maxAge.Set(s.MaxAge.Seconds())

	m.observed.Set(float64(m.now().Unix()))
}
