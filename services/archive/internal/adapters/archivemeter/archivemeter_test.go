package archivemeter_test

import (
	"errors"
	"testing"
	"time"

	"github.com/FabioCaffarello/ghost-trace/libs/eventstream"
	"github.com/FabioCaffarello/ghost-trace/libs/metrics"
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
