package projection

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/hypothesis"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// formCoordinationRing populates the substrate with three actors
// repeatedly co-occurring on a shared descriptor across three rounds
// spaced beyond the window, runs FormCoordinationRingAll, and
// returns the hash of the newly-created CoordinationRingFormation.
// Pre-walks the substrate to record the existing formation set; the
// new formation is the one that's added.
func formCoordinationRing(t *testing.T, sub *substrate.Substrate, descriptor string, declaredAtBase int64) [32]byte {
	t.Helper()
	ctx := context.Background()

	existing := map[[32]byte]bool{}
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType == coordinationRingFormationMessageType {
			existing[row.EventHash] = true
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents (pre): %v", err)
	}

	in := ingest.New(sub, func() time.Time { return time.Unix(0, declaredAtBase) })
	gap := int64(60 * 1e9)
	roundSpacing := int64(2e12)
	actors := []string{
		"actor-" + descriptor + "-a",
		"actor-" + descriptor + "-b",
		"actor-" + descriptor + "-c",
	}
	for round := 0; round < 3; round++ {
		for i, actor := range actors {
			msg := &eventsv1.DeclaredSession{
				DeclaredAt:        declaredAtBase + int64(round)*roundSpacing + int64(i)*gap,
				ActorRef:          actor,
				SessionDescriptor: []byte(descriptor),
			}
			if _, err := in.Append(ctx, msg, msg.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
				t.Fatalf("ingest: %v", err)
			}
		}
	}
	if _, err := hypothesis.FormCoordinationRingAll(ctx, sub,
		hypothesis.CoOccurrenceWindowV1{MinEdgeSupport: 3, MaxWindowSeconds: 600},
		func() time.Time { return time.Unix(0, declaredAtBase+10000) }); err != nil {
		t.Fatalf("FormCoordinationRingAll: %v", err)
	}
	var formationHash [32]byte
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != coordinationRingFormationMessageType {
			return nil
		}
		if existing[row.EventHash] {
			return nil
		}
		formationHash = row.EventHash
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents (post): %v", err)
	}
	if formationHash == ([32]byte{}) {
		t.Fatalf("no new formation found for descriptor %q", descriptor)
	}
	return formationHash
}

func TestProjectCoordinationRingForming(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	formation := formCoordinationRing(t, sub, "ring-form", 1000)
	proj, err := ProjectCoordinationRing(ctx, sub, formation)
	if err != nil {
		t.Fatalf("ProjectCoordinationRing: %v", err)
	}
	if proj.State != StateForming {
		t.Errorf("state: got %q, want %q", proj.State, StateForming)
	}
	if proj.LatestPromotion != nil || proj.LatestDemotion != nil ||
		proj.Dissolution != nil || proj.MergedInto != nil || proj.SplitInto != nil {
		t.Errorf("forming projection should have no lifecycle pointers")
	}
	if len(proj.LifecycleHistory) != 1 {
		t.Errorf("lifecycle history: got %d, want 1 (formation only)", len(proj.LifecycleHistory))
	}
}

func TestProjectCoordinationRingPromoted(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	formation := formCoordinationRing(t, sub, "ring-prom", 1000)
	if _, err := hypothesis.PromoteCoordinationRing(ctx, sub, hypothesis.CoordinationRingPromoteOptions{
		FormationEventHash: formation,
		PromotedAt:         50_000_000_000_000,
		CadenceSeconds:     60,
		Reason:             "operational pilot",
	}, nil); err != nil {
		t.Fatalf("PromoteCoordinationRing: %v", err)
	}

	proj, err := ProjectCoordinationRing(ctx, sub, formation)
	if err != nil {
		t.Fatalf("ProjectCoordinationRing: %v", err)
	}
	if proj.State != StatePromoted {
		t.Errorf("state: got %q, want %q", proj.State, StatePromoted)
	}
	if proj.LatestPromotion == nil || proj.LatestPromotion.Reason != "operational pilot" {
		t.Errorf("LatestPromotion: %+v", proj.LatestPromotion)
	}
}

func TestProjectCoordinationRingDissolved(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	formation := formCoordinationRing(t, sub, "ring-diss", 1000)
	if _, err := hypothesis.DissolveCoordinationRing(ctx, sub, hypothesis.CoordinationRingDissolveOptions{
		FormationEventHash: formation,
		DissolvedAt:        50_000_000_000_000,
		Reason:             "interaction pattern was collection-bias artifact",
	}, nil); err != nil {
		t.Fatalf("DissolveCoordinationRing: %v", err)
	}

	proj, err := ProjectCoordinationRing(ctx, sub, formation)
	if err != nil {
		t.Fatalf("ProjectCoordinationRing: %v", err)
	}
	if proj.State != StateDissolved {
		t.Errorf("state: got %q, want %q", proj.State, StateDissolved)
	}
}

func TestProjectCoordinationRingUnknownFormation(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	var bogus [32]byte
	for i := range bogus {
		bogus[i] = byte(0xff)
	}
	_, err = ProjectCoordinationRing(ctx, sub, bogus)
	if !errors.Is(err, ErrFormationNotFound) {
		t.Errorf("expected ErrFormationNotFound; got %v", err)
	}
}

func TestProjectCoordinationRingCrossSubtypeRejected(t *testing.T) {
	// A BehavioralClusterFormation hash passed to
	// ProjectCoordinationRing MUST return ErrTargetNotFormation —
	// the CR projection's wrong-type check is subtype-specific.
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	// Form a BehavioralCluster.
	in := ingest.New(sub, func() time.Time { return time.Unix(0, 1000) })
	for _, actor := range []string{"actor-a", "actor-b"} {
		msg := &eventsv1.DeclaredSession{
			DeclaredAt:        1000,
			ActorRef:          actor,
			SessionDescriptor: []byte("shared"),
		}
		if _, err := in.Append(ctx, msg, msg.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}
	if _, err := hypothesis.FormAll(ctx, sub,
		hypothesis.SessionDescriptorSharedV1{MinClusterSize: 2}, func() time.Time { return time.Unix(0, 2000) }); err != nil {
		t.Fatalf("FormAll: %v", err)
	}
	var bcFormation [32]byte
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType == "ghosttrace.events.v1.BehavioralClusterFormation" {
			bcFormation = row.EventHash
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}

	_, err = ProjectCoordinationRing(ctx, sub, bcFormation)
	if !errors.Is(err, ErrTargetNotFormation) {
		t.Errorf("expected ErrTargetNotFormation for BC formation; got %v", err)
	}
}

func TestListCoordinationRingsEmptyAndPopulated(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	got, err := ListCoordinationRings(ctx, sub, CoordinationRingListOptions{})
	if err != nil {
		t.Fatalf("ListCoordinationRings: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("empty substrate: got %d", len(got))
	}

	_ = formCoordinationRing(t, sub, "ring-list-a", 1000)
	_ = formCoordinationRing(t, sub, "ring-list-b", 100_000_000_000)
	_ = formCoordinationRing(t, sub, "ring-list-c", 200_000_000_000_000)

	got, err = ListCoordinationRings(ctx, sub, CoordinationRingListOptions{})
	if err != nil {
		t.Fatalf("ListCoordinationRings: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("populated: got %d, want 3", len(got))
	}
	// Ascending lex order by FormationHash.
	for i := 1; i < len(got); i++ {
		if bytes.Compare(got[i-1].FormationHash[:], got[i].FormationHash[:]) >= 0 {
			t.Errorf("not ascending at %d", i)
		}
	}
}

func TestCountCoordinationRingsByStateEquivalentToListFilter(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	_ = formCoordinationRing(t, sub, "ring-c-a", 1000)
	r2 := formCoordinationRing(t, sub, "ring-c-b", 100_000_000_000)
	r3 := formCoordinationRing(t, sub, "ring-c-c", 200_000_000_000_000)

	if _, err := hypothesis.PromoteCoordinationRing(ctx, sub, hypothesis.CoordinationRingPromoteOptions{
		FormationEventHash: r2,
		PromotedAt:         150_000_000_000,
		CadenceSeconds:     60,
	}, nil); err != nil {
		t.Fatalf("Promote: %v", err)
	}
	if _, err := hypothesis.DissolveCoordinationRing(ctx, sub, hypothesis.CoordinationRingDissolveOptions{
		FormationEventHash: r3,
		DissolvedAt:        250_000_000_000_000,
	}, nil); err != nil {
		t.Fatalf("Dissolve: %v", err)
	}

	counts, err := CountCoordinationRingsByState(ctx, sub, CoordinationRingListOptions{})
	if err != nil {
		t.Fatalf("CountCoordinationRingsByState: %v", err)
	}
	if counts.Total != 3 {
		t.Errorf("Total: got %d, want 3", counts.Total)
	}
	for state := range counts.ByState {
		list, err := ListCoordinationRings(ctx, sub, CoordinationRingListOptions{StateFilter: state})
		if err != nil {
			t.Fatalf("ListCoordinationRings(%q): %v", state, err)
		}
		if len(list) != counts.ByState[state] {
			t.Errorf("equivalence violated for state=%q: counts=%d, list=%d",
				state, counts.ByState[state], len(list))
		}
	}
}

func TestProjectAllCoordinationRingsEquivalentToProjectCoordinationRing(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	alpha := formCoordinationRing(t, sub, "ring-eq-a", 1000)
	beta := formCoordinationRing(t, sub, "ring-eq-b", 100_000_000_000)

	if _, err := hypothesis.PromoteCoordinationRing(ctx, sub, hypothesis.CoordinationRingPromoteOptions{
		FormationEventHash: alpha,
		PromotedAt:         50_000_000_000_000,
		CadenceSeconds:     60,
	}, nil); err != nil {
		t.Fatalf("Promote: %v", err)
	}
	if _, err := hypothesis.DissolveCoordinationRing(ctx, sub, hypothesis.CoordinationRingDissolveOptions{
		FormationEventHash: beta,
		DissolvedAt:        150_000_000_000_000,
	}, nil); err != nil {
		t.Fatalf("Dissolve: %v", err)
	}

	all, err := ProjectAllCoordinationRings(ctx, sub)
	if err != nil {
		t.Fatalf("ProjectAllCoordinationRings: %v", err)
	}
	for _, formHash := range [][32]byte{alpha, beta} {
		single, err := ProjectCoordinationRing(ctx, sub, formHash)
		if err != nil {
			t.Fatalf("ProjectCoordinationRing(%x): %v", formHash, err)
		}
		multi, ok := all[formHash]
		if !ok {
			t.Errorf("ProjectAll missing %x", formHash)
			continue
		}
		if single.State != multi.State {
			t.Errorf("%x: single=%q multi=%q", formHash, single.State, multi.State)
		}
		if len(single.LifecycleHistory) != len(multi.LifecycleHistory) {
			t.Errorf("%x: single history=%d, multi=%d",
				formHash, len(single.LifecycleHistory), len(multi.LifecycleHistory))
		}
	}
}

func TestProjectCoordinationRingLatencies(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	formation := formCoordinationRing(t, sub, "ring-lat", 1000)

	formationRow, err := sub.LookupRow(ctx, formation)
	if err != nil {
		t.Fatalf("LookupRow: %v", err)
	}
	formationAt := formationRow.EventTime

	if _, err := hypothesis.PromoteCoordinationRing(ctx, sub, hypothesis.CoordinationRingPromoteOptions{
		FormationEventHash: formation,
		PromotedAt:         50_000_000_000_000,
		CadenceSeconds:     60,
	}, nil); err != nil {
		t.Fatalf("Promote: %v", err)
	}
	if _, err := hypothesis.DissolveCoordinationRing(ctx, sub, hypothesis.CoordinationRingDissolveOptions{
		FormationEventHash: formation,
		DissolvedAt:        80_000_000_000_000,
	}, nil); err != nil {
		t.Fatalf("Dissolve: %v", err)
	}

	proj, err := ProjectCoordinationRing(ctx, sub, formation)
	if err != nil {
		t.Fatalf("ProjectCoordinationRing: %v", err)
	}
	if proj.FormationToFirstPromotionLatencyNs == nil {
		t.Fatal("FormationToFirstPromotionLatencyNs: got nil")
	}
	wantPromo := int64(50_000_000_000_000) - formationAt
	if *proj.FormationToFirstPromotionLatencyNs != wantPromo {
		t.Errorf("FormationToFirstPromotionLatencyNs: got %d, want %d",
			*proj.FormationToFirstPromotionLatencyNs, wantPromo)
	}
	if proj.FormationToDissolutionLatencyNs == nil {
		t.Fatal("FormationToDissolutionLatencyNs: got nil")
	}
	wantDiss := int64(80_000_000_000_000) - formationAt
	if *proj.FormationToDissolutionLatencyNs != wantDiss {
		t.Errorf("FormationToDissolutionLatencyNs: got %d, want %d",
			*proj.FormationToDissolutionLatencyNs, wantDiss)
	}
}
