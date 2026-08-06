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
// IT COULD NOT BE COUNTED FROM WHAT THE BROKER RETAINS. Two ways were
// built and both were deleted after measurement:
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
// THE THIRD WAY WORKS, and the reason is worth stating plainly: it
// stopped asking the broker where this consumer had got to. Both failed
// attempts read a number the broker maintains, and the broker rewrites
// exactly those numbers when it discards records — the evidence is
// destroyed by the event it would evidence.
//
// The archive now writes its own position down, in its own database, in
// the same transaction as the record. archive_stream_skipped is the
// stream's first surviving sequence minus that position: sequences the
// archive walked past that the stream no longer holds and this consumer
// never took in. Nothing outside this process can move that mark.
//
// It is a GAUGE and not a counter, and it is the archive's current
// arrears rather than a running total: it is derived by subtraction on
// every poll, not accumulated. A counter would double-count the same
// skipped records on every reading.
//
// THE EARLY WARNING STAYS, because after-the-fact is the worse half of
// the answer: archive_stream_oldest_message_age_seconds against
// archive_stream_max_age_seconds says how close the stream's oldest
// content is to the retention edge while there is still time to act.
//
// COUNTERS ARE PROCESS-LOCAL AND RESET ON RESTART. That is ordinary
// Prometheus semantics and a scraper handles it, but it is why the
// reconciliation does not use them: "committed" answers what THIS
// process committed, not what the archive holds. Restarting the archive
// mid-backlog was observed to take committed from 5 to 10 while fifteen
// records had in fact arrived. archive_position_* below is read from the
// substrate and survives, which is what `make loss-audit` reconciles.
package archivemeter

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/FabioCaffarello/ghost-trace/libs/eventstream"
	"github.com/FabioCaffarello/ghost-trace/libs/metrics"
	"github.com/FabioCaffarello/ghost-trace/libs/substrate"
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

	firstSeq prometheus.Gauge
	lastSeq  prometheus.Gauge

	// UNDECLARED ON PURPOSE, and the only series here that are.
	//
	// Every counter in this file is declared at zero at construction,
	// because a counter that has never fired is indistinguishable from a
	// counter nobody wired up. For these gauges the same move produces
	// the opposite of honesty: an archive that has just started has no
	// position, and publishing `position_unaccounted 0` would say it
	// walked a stream and lost nothing when it has walked nothing at all.
	//
	// So they are GaugeVecs held unmaterialised, and a series only exists
	// once there is a reading behind it. An absent series here means "not
	// known yet"; a zero means measured.
	skipped      *prometheus.GaugeVec
	posFirst     *prometheus.GaugeVec
	posHighest   *prometheus.GaugeVec
	posCommitted *prometheus.GaugeVec
	posRejected  *prometheus.GaugeVec
	posUnaccount *prometheus.GaugeVec
	posDuplicate *prometheus.GaugeVec
	posRows      *prometheus.GaugeVec
	posReadFail  prometheus.Counter

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
		firstSeq: reg.Gauge("archive_stream_first_sequence",
			"The lowest sequence the stream still holds. Rises as records are "+
				"discarded for age.").WithLabelValues(),
		lastSeq: reg.Gauge("archive_stream_last_sequence",
			"The highest sequence the stream has accepted.").WithLabelValues(),
		skipped: reg.Gauge("archive_stream_skipped",
			"Records that left the stream before this archive took them in: the "+
				"stream's first surviving sequence minus the archive's own durable "+
				"position. Arrears rather than a running total. Absent until there "+
				"is a position to subtract from."),

		posFirst: reg.Gauge("archive_position_first_sequence",
			"The lowest stream sequence this archive has ever recorded. Durable; "+
				"survives restart. Absent on an archive that has consumed nothing."),
		posHighest: reg.Gauge("archive_position_highest_sequence",
			"The highest stream sequence this archive has recorded."),
		posCommitted: reg.Gauge("archive_position_committed",
			"Commit operations this archive has performed across its whole life, "+
				"read from the substrate rather than from a process counter. Exceeds "+
				"the row count by the number of deduplicated redeliveries."),
		posRejected: reg.Gauge("archive_position_rejected",
			"Records this archive refused on purpose, durably. Subtracted from the "+
				"span so a deliberate refusal is not reported as transport loss."),
		posUnaccount: reg.Gauge("archive_position_unaccounted",
			"Sequences inside the range this archive has walked that it neither "+
				"committed nor refused. Records in flight count here, so it is read "+
				"after traffic drains, not during."),
		posDuplicate: reg.Gauge("archive_position_duplicates",
			"Deliveries whose record the substrate already held. At-least-once "+
				"delivery working, not loss. Reported beside the subtraction that "+
				"produces unaccounted rather than inside it — a redelivery adds no "+
				"sequence to the span, so subtracting it drives the figure negative."),
		posRows: reg.Gauge("archive_position_rows",
			"Records the substrate actually holds."),
		posReadFail: reg.Counter("archive_position_read_failures_total",
			"Reads of the durable position that did not return one.").WithLabelValues(),
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
	m.firstSeq.Set(float64(s.FirstSeq))
	m.lastSeq.Set(float64(s.LastSeq))

	m.observed.Set(float64(m.now().Unix()))
}

// ObservePosition publishes the archive's durable position and what it
// implies.
//
// The bool says whether a position exists at all. A fresh archive has
// none, and reporting one full of zeros would make "never consumed
// anything" indistinguishable from "consumed everything perfectly" — the
// two readings this repository refuses to collapse. So nothing is
// published until there is something to publish.
func (m *Meter) ObservePosition(p substrate.Position, rows int64, ok bool, err error) {
	if err != nil {
		m.posReadFail.Inc()
		return
	}
	if !ok {
		return
	}
	m.posFirst.WithLabelValues().Set(float64(p.FirstSeq))
	m.posHighest.WithLabelValues().Set(float64(p.HighestSeq))
	m.posCommitted.WithLabelValues().Set(float64(p.Committed))
	m.posRejected.WithLabelValues().Set(float64(p.Rejected))
	m.posUnaccount.WithLabelValues().Set(float64(p.Unaccounted()))
	m.posDuplicate.WithLabelValues().Set(float64(p.Duplicates))
	m.posRows.WithLabelValues().Set(float64(rows))
}

// ObserveSkipped publishes how far the stream's surviving content has
// moved past this archive.
//
// Positive means records left the stream that this archive never took
// in. Reported only when a position exists: without one there is nothing
// to subtract, and a bare zero would read as "nothing was skipped".
func (m *Meter) ObserveSkipped(streamFirst uint64, p substrate.Position, ok bool) {
	if !ok || streamFirst == 0 {
		return
	}
	// The archive has walked past HighestSeq. Anything the stream has
	// already discarded above that was never delivered here.
	if streamFirst > p.HighestSeq+1 {
		m.skipped.WithLabelValues().Set(float64(streamFirst - p.HighestSeq - 1))
		return
	}
	m.skipped.WithLabelValues().Set(0)
}
