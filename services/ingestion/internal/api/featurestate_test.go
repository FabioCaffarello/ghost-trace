// The §3 reinterpretation invariant, made executable: the evaluation
// record "must carry enough feature state to be reinterpreted without
// the original events". These tests exist because the invariant was
// broken once — the value-injection channel was scored for a milestone
// without being persisted — and a drift that silent must never be
// possible again.
package api

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/libs/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/adapters/substratearchive"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/app"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/feature"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/policy"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/session"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// featureFieldToProto maps every exported field of the three feature
// state structs to its FeatureState proto field. This table IS the
// guard: adding a field to a state struct without extending it fails
// the coverage assertion below, and naming a proto field that does not
// exist fails the descriptor assertion. There is deliberately no
// exception list.
var featureFieldToProto = map[string]string{
	"PointerState.Straightness": "pointer_straightness",
	"PointerState.Segments":     "pointer_segments",
	"PointerState.PathPx":       "pointer_path_px",
	"PointerState.Points":       "pointer_points",

	"KeystrokeState.FlightCV":         "key_flight_cv",
	"KeystrokeState.DwellCV":          "key_dwell_cv",
	"KeystrokeState.MeanDwellMs":      "key_mean_dwell_ms",
	"KeystrokeState.ExactRepeatRatio": "key_exact_repeat_ratio",
	"KeystrokeState.Intervals":        "key_intervals",
	"KeystrokeState.Keys":             "key_events",

	"InteractionState.ProgrammaticScrollRatio": "programmatic_scroll_ratio",
	"InteractionState.ScrollEvents":            "scroll_events",
	"InteractionState.FocusTransitions":        "focus_transitions",
	"InteractionState.DistinctFocusTargets":    "distinct_focus_targets",
	"InteractionState.HiddenPeriods":           "hidden_periods",
	"InteractionState.Pastes":                  "pastes",
	"InteractionState.Autofills":               "autofills",
	"InteractionState.Injections":              "injections",
	"InteractionState.InjectedFields":          "injected_fields",
}

func TestFeatureStateProtoCoversFeatureVector(t *testing.T) {
	// Every struct field must be in the table.
	for _, st := range []any{
		feature.PointerState{}, feature.KeystrokeState{}, feature.InteractionState{},
	} {
		rt := reflect.TypeOf(st)
		for i := 0; i < rt.NumField(); i++ {
			key := rt.Name() + "." + rt.Field(i).Name
			if _, ok := featureFieldToProto[key]; !ok {
				t.Errorf("%s is not mapped to a FeatureState proto field; "+
					"a feature the record cannot carry cannot be scored (§3)", key)
			}
		}
	}

	// Every mapped target must exist in the proto descriptor.
	desc := (&eventsv1.FeatureState{}).ProtoReflect().Descriptor()
	for key, protoName := range featureFieldToProto {
		if desc.Fields().ByName(protoreflect.Name(protoName)) == nil {
			t.Errorf("%s maps to proto field %q, which does not exist in FeatureState", key, protoName)
		}
	}
}

// TestEvaluationRecordCarriesFullFeatureState drives the real path —
// HTTP telemetry into feature accumulators, decision, archive — and
// re-reads the durable Evaluation from the substrate, asserting that
// the channels the policy weighs (and the ones it will be recalibrated
// against) survived the write.
func TestEvaluationRecordCarriesFullFeatureState(t *testing.T) {
	dir := t.TempDir()
	sub, err := substrate.Open(context.Background(), dir+"/db.sqlite", dir+"/blobs")
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	a := app.New(app.Config{TenantID: "t_test", Mode: policy.ModeMonitor},
		session.NewStore(30*time.Minute, time.Now),
		substratearchive.New(ingest.New(sub, time.Now)), time.Now, log)
	s := New(Config{
		SiteKey: testSiteKey, SecretKey: testSecretKey,
		CollectPolicy: CollectPolicy{PointerHz: 20, BatchMs: 2000, Types: []string{"pointer"}},
	}, a, log)
	srv := httptest.NewServer(s.Routes())
	t.Cleanup(srv.Close)
	token := startSession(t, srv)

	// One batch exercising every accumulator the new fields come from:
	// keystrokes, focus on two distinct fields, an autofill, and a
	// value injection into two fields.
	events := []map[string]any{
		{"type": "key", "t": 1000, "phase": "down", "class": "alpha", "target": "f_user"},
		{"type": "key", "t": 1080, "phase": "up", "class": "alpha", "target": "f_user"},
		{"type": "key", "t": 1200, "phase": "down", "class": "alpha", "target": "f_user"},
		{"type": "key", "t": 1270, "phase": "up", "class": "alpha", "target": "f_user"},
		{"type": "focus", "t": 1300, "state": "focus", "target": "f_user"},
		{"type": "focus", "t": 1400, "state": "focus", "target": "f_pass"},
		{"type": "form", "t": 1500, "action": "autofill", "target": "f_user"},
		{"type": "form", "t": 1600, "action": "injected", "target": "f_user"},
		{"type": "form", "t": 1700, "action": "injected", "target": "f_pass"},
	}
	status, _ := post(t, srv, "/v1/telemetry", token, map[string]any{
		"session_token": token, "seq": 1, "sent_at_ms": 2000,
		"page":   map[string]any{"path": "/login", "viewport": []int{1440, 900}},
		"events": events,
	})
	if status != http.StatusAccepted {
		t.Fatalf("telemetry = %d, want 202", status)
	}

	status, body := post(t, srv, "/v1/decisions", testSecretKey, map[string]any{
		"session_token": token, "action": "login",
	})
	if status != http.StatusOK {
		t.Fatalf("decisions = %d, want 200", status)
	}
	evalID, _ := body["evaluation_id"].(string)

	var rec *eventsv1.Evaluation
	err = sub.WalkEvents(context.Background(), func(row substrate.EventRow) error {
		if row.MessageType != "ghosttrace.events.v1.Evaluation" {
			return nil
		}
		payload, err := sub.ReadBlob(context.Background(), row.EventHash)
		if err != nil {
			return err
		}
		var ev eventsv1.Evaluation
		if err := proto.Unmarshal(payload, &ev); err != nil {
			return err
		}
		if ev.EvaluationId == evalID {
			rec = &ev
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk substrate: %v", err)
	}
	if rec == nil {
		t.Fatalf("evaluation %s not found in the archive", evalID)
	}

	fs := rec.Features
	if fs == nil {
		t.Fatal("evaluation record has no FeatureState")
	}
	checks := []struct {
		name string
		got  uint32
		want uint32
	}{
		{"injections", fs.Injections, 2},
		{"injected_fields", fs.InjectedFields, 2},
		{"autofills", fs.Autofills, 1},
		{"distinct_focus_targets", fs.DistinctFocusTargets, 2},
		{"key_events", fs.KeyEvents, 4},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("FeatureState.%s = %d, want %d", c.name, c.got, c.want)
		}
	}
}
