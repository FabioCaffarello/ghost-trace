package hypothesis

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// crSubstrate ingests one DeclaredSession per (actor, descriptor,
// declared_at) tuple and returns the substrate.
func crSubstrate(t *testing.T, items []struct {
	ActorRef   string
	Descriptor string
	DeclaredAt int64
}) *substrate.Substrate {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	in := ingest.New(sub, func() time.Time { return time.Unix(0, 1716120000000000777) })
	for i, it := range items {
		msg := &eventsv1.DeclaredSession{
			DeclaredAt:        it.DeclaredAt,
			ActorRef:          it.ActorRef,
			SessionDescriptor: []byte(it.Descriptor),
		}
		if _, err := in.Append(ctx, msg, msg.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("Append %d: %v", i, err)
		}
	}
	return sub
}

func walkCoordinationRingFormations(t *testing.T, sub *substrate.Substrate) []*eventsv1.CoordinationRingFormation {
	t.Helper()
	ctx := context.Background()
	var out []*eventsv1.CoordinationRingFormation
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != coordinationRingFormationMessageType {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return err
		}
		ev := &eventsv1.CoordinationRingFormation{}
		if err := proto.Unmarshal(payload, ev); err != nil {
			return err
		}
		out = append(out, ev)
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	return out
}

func TestCoOccurrenceWindowV1HappyPath(t *testing.T) {
	// Three actors actor-a, actor-b, actor-c repeatedly co-occur
	// on descriptor "shared" within a 10-minute window. Each pair
	// is observed 3 times → all three edges meet support threshold
	// → one connected component of size 3 → one ring formation.
	gap := int64(60 * 1e9)        // 60s — within window
	roundSpacing := int64(2e12) // 2000s — beyond window
	sub := crSubstrate(t, []struct {
		ActorRef   string
		Descriptor string
		DeclaredAt int64
	}{
		// round 1
		{"actor-a", "shared", 1000}, {"actor-b", "shared", 1000 + gap}, {"actor-c", "shared", 1000 + 2*gap},
		// round 2 (outside window from round 1)
		{"actor-a", "shared", 1000 + roundSpacing}, {"actor-b", "shared", 1000 + roundSpacing + gap}, {"actor-c", "shared", 1000 + roundSpacing + 2*gap},
		// round 3 (outside window from round 2)
		{"actor-a", "shared", 1000 + 2*roundSpacing}, {"actor-b", "shared", 1000 + 2*roundSpacing + gap}, {"actor-c", "shared", 1000 + 2*roundSpacing + 2*gap},
	})

	rep, err := FormCoordinationRingAll(context.Background(), sub,
		CoOccurrenceWindowV1{MinEdgeSupport: 3, MaxWindowSeconds: 600},
		func() time.Time { return time.Unix(0, 1) })
	if err != nil {
		t.Fatalf("FormCoordinationRingAll: %v", err)
	}
	if rep.NewlyFormed != 1 {
		t.Errorf("NewlyFormed: got %d, want 1", rep.NewlyFormed)
	}

	forms := walkCoordinationRingFormations(t, sub)
	if len(forms) != 1 {
		t.Fatalf("substrate carries %d formations, want 1", len(forms))
	}
	got := forms[0]
	if got.PatternSignature != CoOccurrenceWindowV1Signature {
		t.Errorf("pattern_signature: got %q, want %q", got.PatternSignature, CoOccurrenceWindowV1Signature)
	}
	if got.PatternParameters != "max_window_seconds=600;min_edge_support=3" {
		t.Errorf("pattern_parameters: got %q", got.PatternParameters)
	}
	// 3 edges: (actor-a, actor-b), (actor-a, actor-c), (actor-b, actor-c)
	if len(got.Interactions) != 3 {
		t.Fatalf("interactions: got %d, want 3", len(got.Interactions))
	}
	want := []struct{ a, b string }{
		{"actor-a", "actor-b"},
		{"actor-a", "actor-c"},
		{"actor-b", "actor-c"},
	}
	for i, w := range want {
		if got.Interactions[i].ActorA != w.a || got.Interactions[i].ActorB != w.b {
			t.Errorf("interactions[%d]: got (%q, %q), want (%q, %q)",
				i, got.Interactions[i].ActorA, got.Interactions[i].ActorB, w.a, w.b)
		}
	}
	// Per-edge canonicalization: actor_a < actor_b for every entry.
	for i, ix := range got.Interactions {
		if !(ix.ActorA < ix.ActorB) {
			t.Errorf("interactions[%d] not lex-ordered: (%q, %q)", i, ix.ActorA, ix.ActorB)
		}
	}
}

func TestCoOccurrenceWindowV1InsufficientSupport(t *testing.T) {
	// Two actors co-occur only twice (each round spaced beyond
	// window so cross-round pairs don't double-count); support
	// threshold = 3 → no edges qualify → no ring formation.
	gap := int64(60 * 1e9)
	roundSpacing := int64(2e12)
	sub := crSubstrate(t, []struct {
		ActorRef   string
		Descriptor string
		DeclaredAt int64
	}{
		{"actor-a", "shared", 1000}, {"actor-b", "shared", 1000 + gap},
		{"actor-a", "shared", 1000 + roundSpacing}, {"actor-b", "shared", 1000 + roundSpacing + gap},
	})

	rep, err := FormCoordinationRingAll(context.Background(), sub,
		CoOccurrenceWindowV1{MinEdgeSupport: 3, MaxWindowSeconds: 600},
		func() time.Time { return time.Unix(0, 1) })
	if err != nil {
		t.Fatalf("FormCoordinationRingAll: %v", err)
	}
	if rep.NewlyFormed != 0 {
		t.Errorf("NewlyFormed: got %d, want 0", rep.NewlyFormed)
	}
}

func TestCoOccurrenceWindowV1OutsideWindowExcluded(t *testing.T) {
	// Actor-a + actor-b appear three times but the gap each round
	// exceeds the max window. No edge accrues support → no ring.
	gap := int64(1200 * 1e9) // 20 min > 10 min window
	sub := crSubstrate(t, []struct {
		ActorRef   string
		Descriptor string
		DeclaredAt int64
	}{
		{"actor-a", "shared", 1000}, {"actor-b", "shared", 1000 + gap},
		{"actor-a", "shared", 1000 + 2*gap}, {"actor-b", "shared", 1000 + 3*gap},
		{"actor-a", "shared", 1000 + 4*gap}, {"actor-b", "shared", 1000 + 5*gap},
	})

	rep, err := FormCoordinationRingAll(context.Background(), sub,
		CoOccurrenceWindowV1{MinEdgeSupport: 3, MaxWindowSeconds: 600},
		func() time.Time { return time.Unix(0, 1) })
	if err != nil {
		t.Fatalf("FormCoordinationRingAll: %v", err)
	}
	if rep.NewlyFormed != 0 {
		t.Errorf("NewlyFormed: got %d, want 0", rep.NewlyFormed)
	}
}

func TestCoOccurrenceWindowV1Idempotent(t *testing.T) {
	// Re-running the formation should produce zero new rows.
	gap := int64(60 * 1e9)
	roundSpacing := int64(2e12)
	sub := crSubstrate(t, []struct {
		ActorRef   string
		Descriptor string
		DeclaredAt int64
	}{
		{"actor-a", "shared", 1000}, {"actor-b", "shared", 1000 + gap}, {"actor-c", "shared", 1000 + 2*gap},
		{"actor-a", "shared", 1000 + roundSpacing}, {"actor-b", "shared", 1000 + roundSpacing + gap}, {"actor-c", "shared", 1000 + roundSpacing + 2*gap},
		{"actor-a", "shared", 1000 + 2*roundSpacing}, {"actor-b", "shared", 1000 + 2*roundSpacing + gap}, {"actor-c", "shared", 1000 + 2*roundSpacing + 2*gap},
	})

	pat := CoOccurrenceWindowV1{MinEdgeSupport: 3, MaxWindowSeconds: 600}
	if _, err := FormCoordinationRingAll(context.Background(), sub, pat, func() time.Time { return time.Unix(0, 1) }); err != nil {
		t.Fatalf("FormCoordinationRingAll (first): %v", err)
	}
	rep2, err := FormCoordinationRingAll(context.Background(), sub, pat, func() time.Time { return time.Unix(0, 2) })
	if err != nil {
		t.Fatalf("FormCoordinationRingAll (second): %v", err)
	}
	if rep2.NewlyFormed != 0 {
		t.Errorf("idempotency violated: NewlyFormed=%d, want 0", rep2.NewlyFormed)
	}
	if rep2.AlreadyFormed != 1 {
		t.Errorf("AlreadyFormed: got %d, want 1", rep2.AlreadyFormed)
	}
}

func TestCoOccurrenceWindowV1ActorOrderInvariance(t *testing.T) {
	// Re-ingest the same actor pairs but with each pair swapped:
	// content-hash MUST be byte-identical because edges are
	// canonicalized as (lex-smaller, lex-larger) regardless of
	// observation order.
	gap := int64(60 * 1e9)
	roundSpacing := int64(2e12)
	subA := crSubstrate(t, []struct {
		ActorRef   string
		Descriptor string
		DeclaredAt int64
	}{
		{"actor-a", "shared", 1000}, {"actor-b", "shared", 1000 + gap}, {"actor-c", "shared", 1000 + 2*gap},
		{"actor-a", "shared", 1000 + roundSpacing}, {"actor-b", "shared", 1000 + roundSpacing + gap}, {"actor-c", "shared", 1000 + roundSpacing + 2*gap},
		{"actor-a", "shared", 1000 + 2*roundSpacing}, {"actor-b", "shared", 1000 + 2*roundSpacing + gap}, {"actor-c", "shared", 1000 + 2*roundSpacing + 2*gap},
	})
	subB := crSubstrate(t, []struct {
		ActorRef   string
		Descriptor string
		DeclaredAt int64
	}{
		// observation order swapped — content-hash must still match
		{"actor-c", "shared", 1000 + 2*gap}, {"actor-b", "shared", 1000 + gap}, {"actor-a", "shared", 1000},
		{"actor-c", "shared", 1000 + roundSpacing + 2*gap}, {"actor-b", "shared", 1000 + roundSpacing + gap}, {"actor-a", "shared", 1000 + roundSpacing},
		{"actor-c", "shared", 1000 + 2*roundSpacing + 2*gap}, {"actor-b", "shared", 1000 + 2*roundSpacing + gap}, {"actor-a", "shared", 1000 + 2*roundSpacing},
	})

	pat := CoOccurrenceWindowV1{MinEdgeSupport: 3, MaxWindowSeconds: 600}
	if _, err := FormCoordinationRingAll(context.Background(), subA, pat, func() time.Time { return time.Unix(0, 1) }); err != nil {
		t.Fatalf("FormCoordinationRingAll subA: %v", err)
	}
	if _, err := FormCoordinationRingAll(context.Background(), subB, pat, func() time.Time { return time.Unix(0, 1) }); err != nil {
		t.Fatalf("FormCoordinationRingAll subB: %v", err)
	}
	a := walkCoordinationRingFormations(t, subA)
	b := walkCoordinationRingFormations(t, subB)
	if len(a) != 1 || len(b) != 1 {
		t.Fatalf("expected 1 formation each; got a=%d b=%d", len(a), len(b))
	}
	if !proto.Equal(a[0], b[0]) {
		t.Errorf("formations diverged under observation-order swap")
	}
}

func TestCoOccurrenceWindowV1DisconnectedComponents(t *testing.T) {
	// Two disjoint triangles on different descriptors → two
	// separate ring formations.
	gap := int64(60 * 1e9)
	roundSpacing := int64(2e12)
	sub := crSubstrate(t, []struct {
		ActorRef   string
		Descriptor string
		DeclaredAt int64
	}{
		{"actor-a", "d1", 1000}, {"actor-b", "d1", 1000 + gap}, {"actor-c", "d1", 1000 + 2*gap},
		{"actor-a", "d1", 1000 + roundSpacing}, {"actor-b", "d1", 1000 + roundSpacing + gap}, {"actor-c", "d1", 1000 + roundSpacing + 2*gap},
		{"actor-a", "d1", 1000 + 2*roundSpacing}, {"actor-b", "d1", 1000 + 2*roundSpacing + gap}, {"actor-c", "d1", 1000 + 2*roundSpacing + 2*gap},

		{"actor-x", "d2", 1000}, {"actor-y", "d2", 1000 + gap}, {"actor-z", "d2", 1000 + 2*gap},
		{"actor-x", "d2", 1000 + roundSpacing}, {"actor-y", "d2", 1000 + roundSpacing + gap}, {"actor-z", "d2", 1000 + roundSpacing + 2*gap},
		{"actor-x", "d2", 1000 + 2*roundSpacing}, {"actor-y", "d2", 1000 + 2*roundSpacing + gap}, {"actor-z", "d2", 1000 + 2*roundSpacing + 2*gap},
	})

	rep, err := FormCoordinationRingAll(context.Background(), sub,
		CoOccurrenceWindowV1{MinEdgeSupport: 3, MaxWindowSeconds: 600},
		func() time.Time { return time.Unix(0, 1) })
	if err != nil {
		t.Fatalf("FormCoordinationRingAll: %v", err)
	}
	if rep.NewlyFormed != 2 {
		t.Errorf("NewlyFormed: got %d, want 2 (two disjoint components)", rep.NewlyFormed)
	}
}
