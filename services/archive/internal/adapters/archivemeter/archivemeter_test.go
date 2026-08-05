package archivemeter_test

import (
	"errors"
	"testing"
	"time"

	"github.com/FabioCaffarello/ghost-trace/libs/eventstream"
	"github.com/FabioCaffarello/ghost-trace/libs/metrics"
	"github.com/FabioCaffarello/ghost-trace/libs/substrate"
	"github.com/FabioCaffarello/ghost-trace/services/archive/internal/adapters/archivemeter"
	"github.com/FabioCaffarello/ghost-trace/services/archive/internal/consumer"
)

func value(t *testing.T, reg *metrics.Registry, name string, labels map[string]string) (float64, bool) {
	t.Helper()
	families, err := reg.Gatherer().Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	for _, f := range families {
		if f.GetName() != name {
			continue
		}
		for _, m := range f.GetMetric() {
			got := map[string]string{}
			for _, l := range m.GetLabel() {
				got[l.GetName()] = l.GetValue()
			}
			if len(got) != len(labels) {
				continue
			}
			ok := true
			for k, v := range labels {
				if got[k] != v {
					ok = false
				}
			}
			if !ok {
				continue
			}
			if c := m.GetCounter(); c != nil {
				return c.GetValue(), true
			}
			return m.GetGauge().GetValue(), true
		}
	}
	return 0, false
}

func TestEveryCounterExistsBeforeAnyRecordArrives(t *testing.T) {
	reg := metrics.New()
	archivemeter.New(reg, time.Now)

	for _, mt := range eventstream.MessageTypes() {
		if _, ok := value(t, reg, "ghosttrace_archive_committed_total",
			map[string]string{"message_type": mt}); !ok {
			t.Errorf("committed{message_type=%q} is absent before traffic; a type "+
				"that has never arrived and a type that is not counted look the same", mt)
		}
	}
	for _, r := range consumer.RejectReasons {
		if _, ok := value(t, reg, "ghosttrace_archive_rejected_total",
			map[string]string{"reason": r}); !ok {
			t.Errorf("rejected{reason=%q} is absent before traffic", r)
		}
	}
}

func TestAFailedReadDoesNotOverwriteTheLagWithZero(t *testing.T) {
	// THE decision this package makes. Zero would read as "nothing is
	// pending" at the exact moment nothing is known, which is the most
	// expensive possible lie for this particular number.
	reg := metrics.New()
	clock := time.Unix(1_700_000_000, 0)
	m := archivemeter.New(reg, func() time.Time { return clock })

	m.Observe(eventstream.Stats{Pending: 42, AckPending: 7, Redelivered: 3}, nil)
	if v, _ := value(t, reg, "ghosttrace_archive_stream_pending", nil); v != 42 {
		t.Fatalf("pending = %v after a good read, want 42", v)
	}
	observedAt, _ := value(t, reg, "ghosttrace_archive_stream_observed_timestamp_seconds", nil)

	// The broker stops answering.
	clock = clock.Add(5 * time.Minute)
	m.Observe(eventstream.Stats{}, errors.New("broker unreachable"))

	if v, _ := value(t, reg, "ghosttrace_archive_stream_pending", nil); v != 42 {
		t.Errorf("pending = %v after a failed read, want the last real reading (42) "+
			"held rather than replaced", v)
	}
	if v, _ := value(t, reg, "ghosttrace_archive_stream_observed_timestamp_seconds", nil); v != observedAt {
		t.Errorf("observed timestamp moved on a failed read (%v -> %v); the whole "+
			"point is that it stops, so time() minus it is the reading's age", observedAt, v)
	}
	if v, _ := value(t, reg, "ghosttrace_archive_stream_read_failures_total", nil); v != 1 {
		t.Errorf("read failures = %v, want 1", v)
	}
}

func TestAGoodReadAdvancesTheTimestamp(t *testing.T) {
	reg := metrics.New()
	clock := time.Unix(1_700_000_000, 0)
	m := archivemeter.New(reg, func() time.Time { return clock })

	m.Observe(eventstream.Stats{Pending: 1}, nil)
	first, _ := value(t, reg, "ghosttrace_archive_stream_observed_timestamp_seconds", nil)

	clock = clock.Add(30 * time.Second)
	m.Observe(eventstream.Stats{Pending: 0}, nil)
	second, _ := value(t, reg, "ghosttrace_archive_stream_observed_timestamp_seconds", nil)

	if second != first+30 {
		t.Errorf("observed = %v then %v; a successful read must move it, or a live "+
			"reading is indistinguishable from a frozen one", first, second)
	}
	// And a genuine zero is reportable: pending really did reach 0.
	if v, _ := value(t, reg, "ghosttrace_archive_stream_pending", nil); v != 0 {
		t.Errorf("pending = %v, want 0 — a measured zero must still be publishable", v)
	}
}

func TestCommitsAndRejectsCountSeparately(t *testing.T) {
	reg := metrics.New()
	m := archivemeter.New(reg, time.Now)

	m.Committed("ghosttrace.events.v1.Outcome")
	m.Committed("ghosttrace.events.v1.Outcome")
	m.Committed("ghosttrace.events.v1.Evaluation")
	m.Rejected(consumer.ReasonHashMismatch)

	for _, tc := range []struct {
		name   string
		labels map[string]string
		want   float64
	}{
		{"ghosttrace_archive_committed_total",
			map[string]string{"message_type": "ghosttrace.events.v1.Outcome"}, 2},
		{"ghosttrace_archive_committed_total",
			map[string]string{"message_type": "ghosttrace.events.v1.Evaluation"}, 1},
		{"ghosttrace_archive_rejected_total",
			map[string]string{"reason": consumer.ReasonHashMismatch}, 1},
		{"ghosttrace_archive_rejected_total",
			map[string]string{"reason": consumer.ReasonMalformedHash}, 0},
		{"ghosttrace_archive_committed_total",
			map[string]string{"message_type": "ghosttrace.events.v1.SessionStart"}, 0},
	} {
		got, ok := value(t, reg, tc.name, tc.labels)
		if !ok {
			t.Errorf("%s%v is absent", tc.name, tc.labels)
			continue
		}
		if got != tc.want {
			t.Errorf("%s%v = %v, want %v", tc.name, tc.labels, got, tc.want)
		}
	}
}

func TestAgeOutProximityIsReadable(t *testing.T) {
	// The condition the roadmap asked to be detectable before seven days
	// pass. Pending alone cannot see age-out — a discarded record stops
	// being pending, so the backlog gauge IMPROVES at the moment of
	// loss.
	reg := metrics.New()
	m := archivemeter.New(reg, time.Now)

	m.Observe(eventstream.Stats{
		Pending:   500,
		OldestAge: 6 * 24 * time.Hour,
		MaxAge:    7 * 24 * time.Hour,
	}, nil)

	oldest, ok := value(t, reg, "ghosttrace_archive_stream_oldest_message_age_seconds", nil)
	if !ok {
		t.Fatal("oldest message age is absent")
	}
	maxAge, ok := value(t, reg, "ghosttrace_archive_stream_max_age_seconds", nil)
	if !ok {
		t.Fatal("max age is absent; the ratio cannot be computed without it")
	}
	if oldest/maxAge < 0.85 {
		t.Errorf("oldest/max = %v; a backlog one day from the retention edge must "+
			"read as close to it", oldest/maxAge)
	}
}

func TestAFailedReadLeavesTheAgeGaugesAlone(t *testing.T) {
	reg := metrics.New()
	m := archivemeter.New(reg, time.Now)
	m.Observe(eventstream.Stats{OldestAge: 3 * time.Hour, MaxAge: 7 * 24 * time.Hour}, nil)

	m.Observe(eventstream.Stats{}, errors.New("broker unreachable"))

	if v, _ := value(t, reg, "ghosttrace_archive_stream_oldest_message_age_seconds", nil); v != (3 * time.Hour).Seconds() {
		t.Errorf("oldest age = %v after a failed read, want the last real reading held", v)
	}
}

func TestAFreshArchivePublishesNoPositionAtAll(t *testing.T) {
	// The distinction the whole phase turns on. An archive that has
	// never consumed anything must not publish "committed 0, unaccounted
	// 0" — that reads as a perfect run, and is indistinguishable from
	// one. Absence is not zero.
	reg := metrics.New()
	m := archivemeter.New(reg, time.Now)

	m.ObservePosition(substrate.Position{}, 0, false, nil)

	for _, name := range []string{
		"ghosttrace_archive_position_committed",
		"ghosttrace_archive_position_unaccounted",
		"ghosttrace_archive_position_highest_sequence",
	} {
		if _, ok := value(t, reg, name, nil); ok {
			t.Errorf("%s is published by an archive with no position", name)
		}
	}
}

func TestThePositionAndWhatItImpliesArePublished(t *testing.T) {
	reg := metrics.New()
	m := archivemeter.New(reg, time.Now)

	// Walked 10..20 — eleven sequences. Nine committed, one refused, so
	// one is unaccounted for.
	m.ObservePosition(substrate.Position{
		FirstSeq: 10, HighestSeq: 20, Committed: 9, Rejected: 1,
	}, 9, true, nil)

	for _, tc := range []struct {
		name string
		want float64
	}{
		{"ghosttrace_archive_position_first_sequence", 10},
		{"ghosttrace_archive_position_highest_sequence", 20},
		{"ghosttrace_archive_position_committed", 9},
		{"ghosttrace_archive_position_rejected", 1},
		{"ghosttrace_archive_position_unaccounted", 1},
		{"ghosttrace_archive_position_rows", 9},
	} {
		got, ok := value(t, reg, tc.name, nil)
		if !ok {
			t.Errorf("%s is absent", tc.name)
			continue
		}
		if got != tc.want {
			t.Errorf("%s = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestAFailedPositionReadIsCountedRatherThanPublishedAsZero(t *testing.T) {
	reg := metrics.New()
	m := archivemeter.New(reg, time.Now)
	m.ObservePosition(substrate.Position{
		FirstSeq: 1, HighestSeq: 5, Committed: 5,
	}, 5, true, nil)

	m.ObservePosition(substrate.Position{}, 0, false, errors.New("database is locked"))

	if v, _ := value(t, reg, "ghosttrace_archive_position_committed", nil); v != 5 {
		t.Errorf("committed = %v after a failed read, want the last real reading (5)", v)
	}
	if v, _ := value(t, reg, "ghosttrace_archive_position_read_failures_total", nil); v != 1 {
		t.Errorf("read failures = %v, want 1", v)
	}
}

func TestSkippedCountsWhatLeftTheStreamAheadOfTheArchive(t *testing.T) {
	// The measurement 3.4 could not build. Two attempts failed because
	// both read a number the broker rewrites when it discards records.
	// This one subtracts the archive's OWN durable mark, which nothing
	// outside the process can move.
	reg := metrics.New()
	m := archivemeter.New(reg, time.Now)
	pos := substrate.Position{FirstSeq: 1, HighestSeq: 100, Committed: 100}

	// The stream now begins at 106: sequences 101–105 were discarded
	// before this archive ever saw them.
	m.ObserveSkipped(106, pos, true)

	if v, ok := value(t, reg, "ghosttrace_archive_stream_skipped", nil); !ok || v != 5 {
		t.Errorf("skipped = %v (present=%v), want 5", v, ok)
	}
}

func TestAnArchiveKeepingUpReportsNothingSkipped(t *testing.T) {
	// A measured zero, and it must be reportable. The archive is at 100
	// and the stream still holds everything from 1, so nothing left
	// without being seen.
	reg := metrics.New()
	m := archivemeter.New(reg, time.Now)

	m.ObserveSkipped(1, substrate.Position{FirstSeq: 1, HighestSeq: 100, Committed: 100}, true)

	if v, ok := value(t, reg, "ghosttrace_archive_stream_skipped", nil); !ok || v != 0 {
		t.Errorf("skipped = %v (present=%v), want a measured 0", v, ok)
	}
}

func TestSkippedIsNotPublishedWithoutAPositionToSubtractFrom(t *testing.T) {
	// Without a durable mark there is nothing to subtract, and a bare
	// zero would read as "nothing was skipped" rather than "nobody
	// knows".
	reg := metrics.New()
	m := archivemeter.New(reg, time.Now)

	m.ObserveSkipped(500, substrate.Position{}, false)

	if _, ok := value(t, reg, "ghosttrace_archive_stream_skipped", nil); ok {
		t.Error("skipped is published by an archive that has no position")
	}
}
