package lossmeter_test

import (
	"strings"
	"testing"

	"github.com/FabioCaffarello/ghost-trace/libs/metrics"
	"github.com/FabioCaffarello/ghost-trace/services/collector/internal/adapters/lossmeter"
	"github.com/FabioCaffarello/ghost-trace/services/collector/internal/app"
)

func series(t *testing.T, reg *metrics.Registry, name string, labels map[string]string) (float64, bool) {
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
			match := true
			for k, v := range labels {
				if got[k] != v {
					match = false
				}
			}
			if match {
				return m.GetCounter().GetValue(), true
			}
		}
	}
	return 0, false
}

func TestEverySeriesExistsBeforeAnythingHappens(t *testing.T) {
	// THE decision this package exists to make. A counter that has
	// never been incremented does not appear in a scrape, so "no drops"
	// and "the drop counter was never wired" look identical — which is
	// the failure this phase was opened to remove.
	//
	// Declared at zero, a zero means measured-zero.
	reg := metrics.New()
	lossmeter.New(reg)

	for _, kind := range app.Kinds {
		if _, ok := series(t, reg, "ghosttrace_records_written_total",
			map[string]string{"kind": kind}); !ok {
			t.Errorf("written{kind=%q} is absent before any traffic; an absent "+
				"series is indistinguishable from a zero one", kind)
		}
		for _, reason := range app.Reasons {
			if _, ok := series(t, reg, "ghosttrace_records_dropped_total",
				map[string]string{"kind": kind, "reason": reason}); !ok {
				t.Errorf("dropped{kind=%q,reason=%q} is absent before any traffic",
					kind, reason)
			}
		}
	}
}

func TestDeclaredSeriesStartAtZeroRatherThanAtOne(t *testing.T) {
	// Declaring must not count. If Add(0) were ever changed to Inc(),
	// every deployment would report one phantom loss of every kind and
	// the reconciliation this phase is building would never balance.
	reg := metrics.New()
	lossmeter.New(reg)

	for _, kind := range app.Kinds {
		if v, _ := series(t, reg, "ghosttrace_records_written_total",
			map[string]string{"kind": kind}); v != 0 {
			t.Errorf("written{kind=%q} = %v before any traffic, want 0", kind, v)
		}
	}
}

func TestCountingIsPerKindAndPerReason(t *testing.T) {
	reg := metrics.New()
	m := lossmeter.New(reg)

	m.Written(app.KindTelemetry)
	m.Written(app.KindTelemetry)
	m.Dropped(app.KindTelemetry, app.ReasonDeadline)
	m.Dropped(app.KindSnapshot, app.ReasonError)

	for _, tc := range []struct {
		name   string
		labels map[string]string
		want   float64
	}{
		{"ghosttrace_records_written_total", map[string]string{"kind": "telemetry"}, 2},
		{"ghosttrace_records_dropped_total",
			map[string]string{"kind": "telemetry", "reason": "deadline"}, 1},
		{"ghosttrace_records_dropped_total",
			map[string]string{"kind": "snapshot", "reason": "error"}, 1},
		// The neighbours must not move: a counter that increments the
		// wrong label is worse than one that does not increment.
		{"ghosttrace_records_dropped_total",
			map[string]string{"kind": "telemetry", "reason": "error"}, 0},
		{"ghosttrace_records_written_total", map[string]string{"kind": "session_start"}, 0},
	} {
		got, ok := series(t, reg, tc.name, tc.labels)
		if !ok {
			t.Errorf("%s%v is absent", tc.name, tc.labels)
			continue
		}
		if got != tc.want {
			t.Errorf("%s%v = %v, want %v", tc.name, tc.labels, got, tc.want)
		}
	}
}

func TestKindsAndReasonsCoverTheConstants(t *testing.T) {
	// The declaration loop reads app.Kinds and app.Reasons. A constant
	// added without being listed there would mint an undeclared series
	// at first use — present only after it has already been wrong once.
	declared := strings.Join(app.Kinds, ",") + "|" + strings.Join(app.Reasons, ",")
	for _, v := range []string{app.KindSessionStart, app.KindTelemetry, app.KindSnapshot,
		app.ReasonDeadline, app.ReasonError} {
		if !strings.Contains(declared, v) {
			t.Errorf("%q is a constant but is not in Kinds or Reasons, so it would "+
				"never be declared at zero", v)
		}
	}
}
