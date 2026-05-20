package derivation

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// stubContext is a test-local DerivationContext returning a fixed
// per-actor NetworkEvent list. Used to unit-test InactivityWindowV1's
// boundary algorithm without spinning up a substrate.
type stubContext struct {
	byActor map[string][]*eventsv1.NetworkEvent
}

func (s *stubContext) NetworkEventsForActor(actorRef string) []*eventsv1.NetworkEvent {
	return s.byActor[actorRef]
}

func TestInactivityWindowV1Deterministic(t *testing.T) {
	source := &eventsv1.DeclaredSession{DeclaredAt: 1000 * int64(time.Second), ActorRef: "actor-A"}
	dctx := &stubContext{byActor: map[string][]*eventsv1.NetworkEvent{
		"actor-A": {
			{ActorRef: "actor-A", ObservedAt: 1500 * int64(time.Second)},
			{ActorRef: "actor-A", ObservedAt: 1800 * int64(time.Second)},
		},
	}}
	def := InactivityWindowV1{InactivitySeconds: 600}

	out1 := def.Derive(source, [32]byte{}, dctx)
	out1.DefinitionVersion = def.Version()
	out1.DefinitionParameters = def.Parameters()
	out2 := def.Derive(source, [32]byte{}, dctx)
	out2.DefinitionVersion = def.Version()
	out2.DefinitionParameters = def.Parameters()

	_, h1, err := canonical.MarshalAndHash(out1)
	if err != nil {
		t.Fatalf("MarshalAndHash: %v", err)
	}
	_, h2, err := canonical.MarshalAndHash(out2)
	if err != nil {
		t.Fatalf("MarshalAndHash: %v", err)
	}
	if h1 != h2 {
		t.Errorf("Derive non-deterministic: %x != %x", h1, h2)
	}
}

func TestInactivityWindowV1ExtendsBoundary(t *testing.T) {
	// declared_at=1000s; events at 1500s, 1800s (both within 600s
	// window); operational_end_at should be 1800s + 600s = 2400s.
	source := &eventsv1.DeclaredSession{DeclaredAt: 1000 * int64(time.Second), ActorRef: "actor-A"}
	dctx := &stubContext{byActor: map[string][]*eventsv1.NetworkEvent{
		"actor-A": {
			{ActorRef: "actor-A", ObservedAt: 1500 * int64(time.Second)},
			{ActorRef: "actor-A", ObservedAt: 1800 * int64(time.Second)},
		},
	}}
	def := InactivityWindowV1{InactivitySeconds: 600}

	out := def.Derive(source, [32]byte{}, dctx)
	wantEnd := 2400 * int64(time.Second)
	if out.OperationalEndAt != wantEnd {
		t.Errorf("operational_end_at: got %d, want %d", out.OperationalEndAt, wantEnd)
	}
	if out.OperationalStartAt != source.DeclaredAt {
		t.Errorf("operational_start_at: got %d, want %d", out.OperationalStartAt, source.DeclaredAt)
	}
}

func TestInactivityWindowV1StopsAtGap(t *testing.T) {
	// declared_at=1000s; events at 1500s (within window), then 5000s
	// (gap > 600s); operational_end_at should be 1500s + 600s = 2100s,
	// NOT 5000s + 600s.
	source := &eventsv1.DeclaredSession{DeclaredAt: 1000 * int64(time.Second), ActorRef: "actor-A"}
	dctx := &stubContext{byActor: map[string][]*eventsv1.NetworkEvent{
		"actor-A": {
			{ActorRef: "actor-A", ObservedAt: 1500 * int64(time.Second)},
			{ActorRef: "actor-A", ObservedAt: 5000 * int64(time.Second)},
		},
	}}
	def := InactivityWindowV1{InactivitySeconds: 600}

	out := def.Derive(source, [32]byte{}, dctx)
	wantEnd := 2100 * int64(time.Second)
	if out.OperationalEndAt != wantEnd {
		t.Errorf("operational_end_at after gap: got %d, want %d (boundary should not extend across > InactivitySeconds gap)", out.OperationalEndAt, wantEnd)
	}
}

func TestInactivityWindowV1NoEventsFallsBack(t *testing.T) {
	// No NetworkEvents in dctx for this actor; operational_end_at =
	// declared_at + InactivitySeconds.
	source := &eventsv1.DeclaredSession{DeclaredAt: 1000 * int64(time.Second), ActorRef: "actor-A"}
	dctx := &stubContext{byActor: nil}
	def := InactivityWindowV1{InactivitySeconds: 600}

	out := def.Derive(source, [32]byte{}, dctx)
	wantEnd := 1600 * int64(time.Second)
	if out.OperationalEndAt != wantEnd {
		t.Errorf("operational_end_at fallback: got %d, want %d", out.OperationalEndAt, wantEnd)
	}
}

func TestInactivityWindowV1IgnoresOtherActors(t *testing.T) {
	// dctx has events for both "actor-A" and "actor-B"; the
	// derivation for actor-A must not consider actor-B's events.
	source := &eventsv1.DeclaredSession{DeclaredAt: 1000 * int64(time.Second), ActorRef: "actor-A"}
	dctx := &stubContext{byActor: map[string][]*eventsv1.NetworkEvent{
		"actor-A": {{ActorRef: "actor-A", ObservedAt: 1100 * int64(time.Second)}},
		"actor-B": {{ActorRef: "actor-B", ObservedAt: 5000 * int64(time.Second)}},
	}}
	def := InactivityWindowV1{InactivitySeconds: 600}

	out := def.Derive(source, [32]byte{}, dctx)
	wantEnd := 1700 * int64(time.Second) // last actor-A event 1100 + window 600
	if out.OperationalEndAt != wantEnd {
		t.Errorf("operational_end_at: got %d, want %d (actor-B events should NOT influence actor-A boundary)", out.OperationalEndAt, wantEnd)
	}
}

func TestInactivityWindowV1IgnoresPreSessionEvents(t *testing.T) {
	// declared_at=1000s; an event at 500s (BEFORE declared_at) MUST
	// NOT contribute to lastObserved. Events at 1100s (within window
	// from declared_at) does contribute.
	source := &eventsv1.DeclaredSession{DeclaredAt: 1000 * int64(time.Second), ActorRef: "actor-A"}
	dctx := &stubContext{byActor: map[string][]*eventsv1.NetworkEvent{
		"actor-A": {
			{ActorRef: "actor-A", ObservedAt: 500 * int64(time.Second)},
			{ActorRef: "actor-A", ObservedAt: 1100 * int64(time.Second)},
		},
	}}
	def := InactivityWindowV1{InactivitySeconds: 600}

	out := def.Derive(source, [32]byte{}, dctx)
	wantEnd := 1700 * int64(time.Second) // 1100 + 600
	if out.OperationalEndAt != wantEnd {
		t.Errorf("operational_end_at: got %d, want %d (pre-session events should be ignored)", out.OperationalEndAt, wantEnd)
	}
}

func TestInactivityWindowV1VersioningProducesNewRecord(t *testing.T) {
	source := &eventsv1.DeclaredSession{DeclaredAt: 1000 * int64(time.Second), ActorRef: "actor-A"}
	dctx := &stubContext{byActor: map[string][]*eventsv1.NetworkEvent{
		"actor-A": {{ActorRef: "actor-A", ObservedAt: 1100 * int64(time.Second)}},
	}}
	defA := InactivityWindowV1{InactivitySeconds: 600}
	defB := InactivityWindowV1{InactivitySeconds: 1800}

	outA := defA.Derive(source, [32]byte{}, dctx)
	outA.DefinitionVersion = defA.Version()
	outA.DefinitionParameters = defA.Parameters()
	outB := defB.Derive(source, [32]byte{}, dctx)
	outB.DefinitionVersion = defB.Version()
	outB.DefinitionParameters = defB.Parameters()

	_, hA, err := canonical.MarshalAndHash(outA)
	if err != nil {
		t.Fatalf("MarshalAndHash A: %v", err)
	}
	_, hB, err := canonical.MarshalAndHash(outB)
	if err != nil {
		t.Fatalf("MarshalAndHash B: %v", err)
	}
	if hA == hB {
		t.Errorf("different parameters produced same hash %x (versioning broken)", hA)
	}
}

// TestDeriveAllInactivityWindowV1Integration is the end-to-end proof
// that the dispatch + DerivationContext pre-collection + the
// inactivity-window-v1 boundary algorithm compose. Substrate contains
// 1 DeclaredSession + 2 NetworkEvents for the same actor; one within
// the window, one beyond. Expected: 1 OperationalSession committed
// with operational_end_at reflecting only the first event.
func TestDeriveAllInactivityWindowV1Integration(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	in := ingest.New(sub, time.Now)
	decl := &eventsv1.DeclaredSession{
		DeclaredAt: 1000 * int64(time.Second),
		ActorRef:   "actor-integration",
	}
	if _, err := in.Append(ctx, decl, decl.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
		t.Fatalf("Append DeclaredSession: %v", err)
	}
	for _, obs := range []int64{1100, 5000} {
		ne := &eventsv1.NetworkEvent{
			ObservedAt: obs * int64(time.Second),
			ActorRef:   "actor-integration",
		}
		if _, err := in.Append(ctx, ne, ne.ObservedAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("Append NetworkEvent %d: %v", obs, err)
		}
	}

	def := InactivityWindowV1{InactivitySeconds: 600}
	rep, err := DeriveAll(ctx, sub, def, func() time.Time { return time.Unix(0, 1) })
	if err != nil {
		t.Fatalf("DeriveAll: %v", err)
	}
	if rep.Examined != 1 {
		t.Errorf("Examined: got %d, want 1 (1 DeclaredSession in substrate)", rep.Examined)
	}
	if rep.NewlyDerived != 1 {
		t.Errorf("NewlyDerived: got %d, want 1", rep.NewlyDerived)
	}

	// Walk substrate, find the OperationalSession, verify boundary.
	var found *eventsv1.OperationalSession
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != "ghosttrace.events.v1.OperationalSession" {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return err
		}
		os := &eventsv1.OperationalSession{}
		if err := proto.Unmarshal(payload, os); err != nil {
			return err
		}
		found = os
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	if found == nil {
		t.Fatal("no OperationalSession found in substrate")
	}
	if got, want := found.OperationalStartAt, 1000*int64(time.Second); got != want {
		t.Errorf("operational_start_at: got %d, want %d", got, want)
	}
	// First event at 1100s within window → lastObserved=1100s.
	// Second event at 5000s > 1100+600=1700s → break.
	// operational_end_at = 1100 + 600 = 1700s.
	if got, want := found.OperationalEndAt, 1700*int64(time.Second); got != want {
		t.Errorf("operational_end_at: got %d, want %d (boundary should reflect only the first event)", got, want)
	}
	if found.DefinitionVersion != InactivityWindowV1Version {
		t.Errorf("definition_version: got %q, want %q", found.DefinitionVersion, InactivityWindowV1Version)
	}
}
