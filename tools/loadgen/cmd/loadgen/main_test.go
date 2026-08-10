package main

import (
	"encoding/json"
	"testing"

	"github.com/FabioCaffarello/ghost-trace/libs/wire"
)

// TestTelemetryEventsCarryTheirPayload pins the fix for a defect that
// invalidated measurements rather than requests: the driver's pointer
// events carried `x`/`y` — fields the wire does not have — so the
// collector accepted every batch with a 202, dropped the unknown keys,
// and did zero pointer-feature work. Every load curve published by
// Phase 4 was taken against that lighter system. A 202 is not proof of
// work done; this test is.
func TestTelemetryEventsCarryTheirPayload(t *testing.T) {
	var batch wire.TelemetryBatch
	if err := json.Unmarshal(telemetry("tok_test", 8), &batch); err != nil {
		t.Fatalf("the driver's own batch does not decode as the wire type: %v", err)
	}
	if batch.SessionToken != "tok_test" || len(batch.Events) != 8 {
		t.Fatalf("batch shape: token %q, %d events", batch.SessionToken, len(batch.Events))
	}
	pointers := 0
	for i, ev := range batch.Events {
		switch ev.Type {
		case "pointer":
			pointers++
			if len(ev.Pts) == 0 {
				t.Errorf("event %d: a pointer event with no pts does zero "+
					"feature work — the exact defect the Phase-4 curves were "+
					"measured on", i)
			}
			if ev.Src == "" {
				t.Errorf("event %d: pointer event carries no src", i)
			}
		case "key":
			if ev.Phase == "" || ev.KeyClass == "" {
				t.Errorf("event %d: key event missing phase or class", i)
			}
		default:
			t.Errorf("event %d: unexpected type %q", i, ev.Type)
		}
	}
	if pointers == 0 {
		t.Fatal("no pointer events in the batch; the load does not exercise the pointer path")
	}
}

// TestBodiesAreTheWireTypes asserts every request body the driver sends
// round-trips through the type the server decodes — hand-rolled JSON is
// how `x`/`y` shipped, and how it would ship again.
func TestBodiesAreTheWireTypes(t *testing.T) {
	var sess wire.SessionsRequest
	if err := json.Unmarshal(sessionBody("pk_test"), &sess); err != nil {
		t.Fatalf("session body: %v", err)
	}
	if sess.SiteKey != "pk_test" || sess.Page.Path == "" {
		t.Fatalf("session body shape: %+v", sess)
	}
}
