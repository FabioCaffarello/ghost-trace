package projection

import (
	"context"
	"testing"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/hypothesis"
)

func TestLatencyForming(t *testing.T) {
	// A pure-forming projection has no promotion / demotion /
	// dissolution events, so all latency fields are nil.
	sub := newSubstrate(t)
	form := formCluster(t, sub, "alpha", "actor-lat-form-", 1000)

	proj, err := ProjectHypothesis(context.Background(), sub, form)
	if err != nil {
		t.Fatalf("ProjectHypothesis: %v", err)
	}
	if proj.FormationToFirstPromotionLatencyNs != nil {
		t.Errorf("FormationToFirstPromotionLatencyNs: got %v, want nil",
			*proj.FormationToFirstPromotionLatencyNs)
	}
	if proj.LatestPromotionToLatestDemotionLatencyNs != nil {
		t.Errorf("LatestPromotionToLatestDemotionLatencyNs: got %v, want nil",
			*proj.LatestPromotionToLatestDemotionLatencyNs)
	}
	if proj.FormationToDissolutionLatencyNs != nil {
		t.Errorf("FormationToDissolutionLatencyNs: got %v, want nil",
			*proj.FormationToDissolutionLatencyNs)
	}
}

func TestLatencyFormationToFirstPromotion(t *testing.T) {
	sub := newSubstrate(t)
	form := formCluster(t, sub, "alpha", "actor-lat-prom-", 1000)
	// formation_at = 1001 (max(declared_at)).
	ctx := context.Background()
	if _, err := hypothesis.Promote(ctx, sub, hypothesis.PromoteOptions{
		FormationEventHash: form,
		PromotedAt:         5001,
		CadenceSeconds:     60,
	}, nil); err != nil {
		t.Fatalf("Promote: %v", err)
	}

	proj, err := ProjectHypothesis(ctx, sub, form)
	if err != nil {
		t.Fatalf("ProjectHypothesis: %v", err)
	}
	if proj.FormationToFirstPromotionLatencyNs == nil {
		t.Fatal("FormationToFirstPromotionLatencyNs: got nil")
	}
	want := int64(5001 - 1001)
	if *proj.FormationToFirstPromotionLatencyNs != want {
		t.Errorf("FormationToFirstPromotionLatencyNs: got %d, want %d",
			*proj.FormationToFirstPromotionLatencyNs, want)
	}
	// Demote not present → its latency stays nil.
	if proj.LatestPromotionToLatestDemotionLatencyNs != nil {
		t.Errorf("LatestPromotionToLatestDemotionLatencyNs: got %v, want nil",
			*proj.LatestPromotionToLatestDemotionLatencyNs)
	}
}

func TestLatencyFormationToFirstPromotionUsesEarliest(t *testing.T) {
	// Re-promotion arc: two promotions exist. First-promotion
	// latency MUST be computed against the EARLIEST promotion
	// (not the latest), since it answers "how long after formation
	// did this hypothesis first move into operational use?".
	sub := newSubstrate(t)
	form := formCluster(t, sub, "alpha", "actor-lat-first-", 1000)
	ctx := context.Background()
	// First promotion at t=5001.
	if _, err := hypothesis.Promote(ctx, sub, hypothesis.PromoteOptions{
		FormationEventHash: form,
		PromotedAt:         5001,
		CadenceSeconds:     60,
		Reason:             "first",
	}, nil); err != nil {
		t.Fatalf("first Promote: %v", err)
	}
	// Second promotion at t=10001 (different cadence so distinct hash).
	if _, err := hypothesis.Promote(ctx, sub, hypothesis.PromoteOptions{
		FormationEventHash: form,
		PromotedAt:         10001,
		CadenceSeconds:     120,
		Reason:             "second",
	}, nil); err != nil {
		t.Fatalf("second Promote: %v", err)
	}

	proj, err := ProjectHypothesis(ctx, sub, form)
	if err != nil {
		t.Fatalf("ProjectHypothesis: %v", err)
	}
	if proj.FormationToFirstPromotionLatencyNs == nil {
		t.Fatal("FormationToFirstPromotionLatencyNs: got nil")
	}
	want := int64(5001 - 1001) // first promotion - formation
	if *proj.FormationToFirstPromotionLatencyNs != want {
		t.Errorf("FormationToFirstPromotionLatencyNs: got %d, want %d (must be EARLIEST promotion)",
			*proj.FormationToFirstPromotionLatencyNs, want)
	}
}

func TestLatencyLatestPromotionToLatestDemotion(t *testing.T) {
	sub := newSubstrate(t)
	form := formCluster(t, sub, "alpha", "actor-lat-pd-", 1000)
	ctx := context.Background()
	promRep, err := hypothesis.Promote(ctx, sub, hypothesis.PromoteOptions{
		FormationEventHash: form,
		PromotedAt:         5001,
		CadenceSeconds:     60,
	}, nil)
	if err != nil {
		t.Fatalf("Promote: %v", err)
	}
	var promHash [32]byte
	if raw, _ := hexDecode(promRep.PromotionEventHashHex); true {
		copy(promHash[:], raw)
	}
	if _, err := hypothesis.Demote(ctx, sub, hypothesis.DemoteOptions{
		PromotionEventHash: promHash,
		DemotedAt:          12001,
	}, nil); err != nil {
		t.Fatalf("Demote: %v", err)
	}

	proj, err := ProjectHypothesis(ctx, sub, form)
	if err != nil {
		t.Fatalf("ProjectHypothesis: %v", err)
	}
	if proj.LatestPromotionToLatestDemotionLatencyNs == nil {
		t.Fatal("LatestPromotionToLatestDemotionLatencyNs: got nil")
	}
	want := int64(12001 - 5001)
	if *proj.LatestPromotionToLatestDemotionLatencyNs != want {
		t.Errorf("LatestPromotionToLatestDemotionLatencyNs: got %d, want %d",
			*proj.LatestPromotionToLatestDemotionLatencyNs, want)
	}
}

func TestLatencyFormationToDissolution(t *testing.T) {
	sub := newSubstrate(t)
	form := formCluster(t, sub, "alpha", "actor-lat-diss-", 1000)
	ctx := context.Background()
	if _, err := hypothesis.Dissolve(ctx, sub, hypothesis.DissolveOptions{
		FormationEventHash: form,
		DissolvedAt:        30001,
	}, nil); err != nil {
		t.Fatalf("Dissolve: %v", err)
	}

	proj, err := ProjectHypothesis(ctx, sub, form)
	if err != nil {
		t.Fatalf("ProjectHypothesis: %v", err)
	}
	if proj.FormationToDissolutionLatencyNs == nil {
		t.Fatal("FormationToDissolutionLatencyNs: got nil")
	}
	want := int64(30001 - 1001)
	if *proj.FormationToDissolutionLatencyNs != want {
		t.Errorf("FormationToDissolutionLatencyNs: got %d, want %d",
			*proj.FormationToDissolutionLatencyNs, want)
	}
}

func TestLatencyAllThreeArcsPopulated(t *testing.T) {
	// Form → promote → demote → dissolve. All three latencies
	// populated.
	sub := newSubstrate(t)
	form := formCluster(t, sub, "alpha", "actor-lat-all-", 1000)
	ctx := context.Background()
	promRep, _ := hypothesis.Promote(ctx, sub, hypothesis.PromoteOptions{
		FormationEventHash: form,
		PromotedAt:         3001,
		CadenceSeconds:     60,
	}, nil)
	var promHash [32]byte
	if raw, _ := hexDecode(promRep.PromotionEventHashHex); true {
		copy(promHash[:], raw)
	}
	if _, err := hypothesis.Demote(ctx, sub, hypothesis.DemoteOptions{
		PromotionEventHash: promHash,
		DemotedAt:          7001,
	}, nil); err != nil {
		t.Fatalf("Demote: %v", err)
	}
	if _, err := hypothesis.Dissolve(ctx, sub, hypothesis.DissolveOptions{
		FormationEventHash: form,
		DissolvedAt:        9001,
	}, nil); err != nil {
		t.Fatalf("Dissolve: %v", err)
	}

	proj, err := ProjectHypothesis(ctx, sub, form)
	if err != nil {
		t.Fatalf("ProjectHypothesis: %v", err)
	}
	cases := []struct {
		name string
		got  *int64
		want int64
	}{
		{"FormationToFirstPromotionLatencyNs", proj.FormationToFirstPromotionLatencyNs, 3001 - 1001},
		{"LatestPromotionToLatestDemotionLatencyNs", proj.LatestPromotionToLatestDemotionLatencyNs, 7001 - 3001},
		{"FormationToDissolutionLatencyNs", proj.FormationToDissolutionLatencyNs, 9001 - 1001},
	}
	for _, c := range cases {
		if c.got == nil {
			t.Errorf("%s: got nil", c.name)
			continue
		}
		if *c.got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, *c.got, c.want)
		}
	}
}

func TestLatencyProjectAllEquivalentToProjectHypothesis(t *testing.T) {
	// Equivalence invariant family extension: latency fields from
	// ProjectAll[hash] must match the single-formation
	// ProjectHypothesis(hash) latency fields. Defends against
	// per-projection vs per-substrate latency-computation drift.
	sub := newSubstrate(t)
	form := formCluster(t, sub, "alpha", "actor-lat-eq-", 1000)
	ctx := context.Background()
	if _, err := hypothesis.Promote(ctx, sub, hypothesis.PromoteOptions{
		FormationEventHash: form,
		PromotedAt:         4001,
		CadenceSeconds:     60,
	}, nil); err != nil {
		t.Fatalf("Promote: %v", err)
	}
	if _, err := hypothesis.Dissolve(ctx, sub, hypothesis.DissolveOptions{
		FormationEventHash: form,
		DissolvedAt:        8001,
	}, nil); err != nil {
		t.Fatalf("Dissolve: %v", err)
	}

	single, err := ProjectHypothesis(ctx, sub, form)
	if err != nil {
		t.Fatalf("ProjectHypothesis: %v", err)
	}
	all, err := ProjectAll(ctx, sub)
	if err != nil {
		t.Fatalf("ProjectAll: %v", err)
	}
	multi, ok := all[form]
	if !ok {
		t.Fatal("ProjectAll missing formation")
	}
	check := func(name string, sNg, mNg *int64) {
		if (sNg == nil) != (mNg == nil) {
			t.Errorf("%s: nil divergence (single=%v multi=%v)", name, sNg, mNg)
			return
		}
		if sNg != nil && *sNg != *mNg {
			t.Errorf("%s: single=%d multi=%d", name, *sNg, *mNg)
		}
	}
	check("FormationToFirstPromotionLatencyNs",
		single.FormationToFirstPromotionLatencyNs, multi.FormationToFirstPromotionLatencyNs)
	check("LatestPromotionToLatestDemotionLatencyNs",
		single.LatestPromotionToLatestDemotionLatencyNs, multi.LatestPromotionToLatestDemotionLatencyNs)
	check("FormationToDissolutionLatencyNs",
		single.FormationToDissolutionLatencyNs, multi.FormationToDissolutionLatencyNs)
}

func TestLatencyNegativeWhenEventsOutOfOrder(t *testing.T) {
	// Producer supplies a promotion timestamp PRECEDING the
	// formation's max(declared_at). The substrate accepts (timestamps
	// are operator-supplied; substrate does not gate on temporal
	// order); the projection's latency reflects what was recorded —
	// a negative value. This is not a §2.1 invariant violation per
	// the latency-derivation contract (§0055): the field reports
	// observed timestamps, not asserts about producer correctness.
	sub := newSubstrate(t)
	form := formCluster(t, sub, "alpha", "actor-lat-neg-", 1000)
	// formation_at = 1001.
	ctx := context.Background()
	if _, err := hypothesis.Promote(ctx, sub, hypothesis.PromoteOptions{
		FormationEventHash: form,
		PromotedAt:         500, // BEFORE formation
		CadenceSeconds:     60,
	}, nil); err != nil {
		t.Fatalf("Promote: %v", err)
	}

	proj, err := ProjectHypothesis(ctx, sub, form)
	if err != nil {
		t.Fatalf("ProjectHypothesis: %v", err)
	}
	if proj.FormationToFirstPromotionLatencyNs == nil {
		t.Fatal("expected non-nil latency even for out-of-order timestamps")
	}
	want := int64(500 - 1001)
	if *proj.FormationToFirstPromotionLatencyNs != want {
		t.Errorf("FormationToFirstPromotionLatencyNs: got %d, want %d (negative reflects observed timestamps)",
			*proj.FormationToFirstPromotionLatencyNs, want)
	}
}
