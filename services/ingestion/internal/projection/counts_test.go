package projection

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/hypothesis"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

func TestCountByStateEmptySubstrate(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	got, err := CountByState(ctx, sub, ListOptions{})
	if err != nil {
		t.Fatalf("CountByState: %v", err)
	}
	if got.Total != 0 {
		t.Errorf("Total: got %d, want 0", got.Total)
	}
	// Every state key MUST be present even on empty substrate.
	expected := []State{StateForming, StatePromoted, StateDemoted, StateDissolved, StateMergedInto, StateSplitInto}
	for _, s := range expected {
		v, ok := got.ByState[s]
		if !ok {
			t.Errorf("missing key %q in ByState", s)
		}
		if v != 0 {
			t.Errorf("ByState[%q]: got %d, want 0", s, v)
		}
	}
}

func TestCountByStateSingleForming(t *testing.T) {
	sub := newSubstrate(t)
	_ = formCluster(t, sub, "alpha", "actor-c1-", 1000)
	got, err := CountByState(context.Background(), sub, ListOptions{})
	if err != nil {
		t.Fatalf("CountByState: %v", err)
	}
	if got.Total != 1 {
		t.Errorf("Total: got %d, want 1", got.Total)
	}
	if got.ByState[StateForming] != 1 {
		t.Errorf("ByState[forming]: got %d, want 1", got.ByState[StateForming])
	}
	for _, s := range []State{StatePromoted, StateDemoted, StateDissolved, StateMergedInto, StateSplitInto} {
		if got.ByState[s] != 0 {
			t.Errorf("ByState[%q]: got %d, want 0", s, got.ByState[s])
		}
	}
}

func TestCountByStateMixed(t *testing.T) {
	sub := newSubstrate(t)
	ctx := context.Background()
	// Two formations forming.
	_ = formCluster(t, sub, "f1", "actor-mix-f1-", 1000)
	_ = formCluster(t, sub, "f2", "actor-mix-f2-", 2000)
	// One promoted.
	promForm := formCluster(t, sub, "f3", "actor-mix-p-", 3000)
	if _, err := hypothesis.Promote(ctx, sub, hypothesis.PromoteOptions{
		FormationEventHash: promForm,
		PromotedAt:         4000,
		CadenceSeconds:     60,
	}, nil); err != nil {
		t.Fatalf("Promote: %v", err)
	}
	// One dissolved.
	dissForm := formCluster(t, sub, "f4", "actor-mix-d-", 5000)
	if _, err := hypothesis.Dissolve(ctx, sub, hypothesis.DissolveOptions{
		FormationEventHash: dissForm,
		DissolvedAt:        6000,
	}, nil); err != nil {
		t.Fatalf("Dissolve: %v", err)
	}

	got, err := CountByState(ctx, sub, ListOptions{})
	if err != nil {
		t.Fatalf("CountByState: %v", err)
	}
	if got.Total != 4 {
		t.Errorf("Total: got %d, want 4", got.Total)
	}
	checks := map[State]int{
		StateForming:    2,
		StatePromoted:   1,
		StateDemoted:    0,
		StateDissolved:  1,
		StateMergedInto: 0,
		StateSplitInto:  0,
	}
	for s, want := range checks {
		if got.ByState[s] != want {
			t.Errorf("ByState[%q]: got %d, want %d", s, got.ByState[s], want)
		}
	}
}

func TestCountByStateAllSix(t *testing.T) {
	// Substrate carrying one of each terminal state.
	sub := newSubstrate(t)
	ctx := context.Background()
	formForm := formCluster(t, sub, "f1", "actor-six-f-", 1000)
	formProm := formCluster(t, sub, "f2", "actor-six-p-", 2000)
	formDemo := formCluster(t, sub, "f3", "actor-six-d-", 3000)
	formDiss := formCluster(t, sub, "f4", "actor-six-x-", 4000)
	formMerge := formCluster(t, sub, "f5", "actor-six-m-", 5000)
	formSplit := formCluster(t, sub, "f6", "actor-six-s-", 6000)
	extraG := formCluster(t, sub, "fg", "actor-six-g-", 7000)
	extraD := formCluster(t, sub, "fd", "actor-six-q-", 8000)

	_ = formForm
	if _, err := hypothesis.Promote(ctx, sub, hypothesis.PromoteOptions{
		FormationEventHash: formProm, PromotedAt: 9000, CadenceSeconds: 60,
	}, nil); err != nil {
		t.Fatalf("Promote formProm: %v", err)
	}
	promRep, _ := hypothesis.Promote(ctx, sub, hypothesis.PromoteOptions{
		FormationEventHash: formDemo, PromotedAt: 10000, CadenceSeconds: 60,
	}, nil)
	var p [32]byte
	if raw, _ := hexDecode(promRep.PromotionEventHashHex); true {
		copy(p[:], raw)
	}
	if _, err := hypothesis.Demote(ctx, sub, hypothesis.DemoteOptions{
		PromotionEventHash: p, DemotedAt: 11000,
	}, nil); err != nil {
		t.Fatalf("Demote: %v", err)
	}
	if _, err := hypothesis.Dissolve(ctx, sub, hypothesis.DissolveOptions{
		FormationEventHash: formDiss, DissolvedAt: 12000,
	}, nil); err != nil {
		t.Fatalf("Dissolve: %v", err)
	}
	if _, err := hypothesis.Merge(ctx, sub, hypothesis.MergeOptions{
		AntecedentAFormationHash: formMerge, AntecedentBFormationHash: extraG,
		ProducedFormationHash: extraD, MergedAt: 13000,
	}, nil); err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if _, err := hypothesis.Split(ctx, sub, hypothesis.SplitOptions{
		AntecedentFormationHash:  formSplit,
		SuccessorFormationHashes: [][32]byte{extraG, extraD},
		SplitAt:                  14000,
	}, nil); err != nil {
		t.Fatalf("Split: %v", err)
	}

	got, err := CountByState(ctx, sub, ListOptions{})
	if err != nil {
		t.Fatalf("CountByState: %v", err)
	}
	if got.Total != 8 {
		t.Errorf("Total: got %d, want 8", got.Total)
	}
	// formForm + extraG + extraD = 3 forming (extraG/extraD are still
	// forming — being referenced as merge antecedent or split successor
	// does NOT change their state per the projection's
	// State-precedence rules; only being a merge ANTECEDENT or split
	// ANTECEDENT-targeted-FROM changes the state of the referenced
	// formation. extraG appears as merge antecedent so it is
	// merged_into. extraD is never an antecedent of anything, so it
	// remains forming. Wait — extraG IS merge antecedent (alongside
	// formMerge); so extraG is merged_into. extraD is split successor
	// (never antecedent) so remains forming. Total: formForm forming,
	// extraD forming = 2 forming.
	checks := map[State]int{
		StateForming:    2, // formForm + extraD
		StatePromoted:   1, // formProm
		StateDemoted:    1, // formDemo
		StateDissolved:  1, // formDiss
		StateMergedInto: 2, // formMerge + extraG (both are merge antecedents)
		StateSplitInto:  1, // formSplit
	}
	for s, want := range checks {
		if got.ByState[s] != want {
			t.Errorf("ByState[%q]: got %d, want %d", s, got.ByState[s], want)
		}
	}
}

func TestCountByStateEquivalentToListFilter(t *testing.T) {
	// Equivalence invariant per §0053: for every State value,
	// CountByState.ByState[s] equals
	// len(ListHypotheses(ctx, sub, ListOptions{StateFilter: s})).
	sub := newSubstrate(t)
	ctx := context.Background()
	for i, desc := range []string{"a", "b", "c", "d", "e"} {
		_ = formCluster(t, sub, desc, "actor-eq-"+desc+"-", int64(1000*(i+1)))
	}
	// Promote two of them so we have a mix.
	promForm1 := formCluster(t, sub, "p1", "actor-eq-p1-", 6000)
	promForm2 := formCluster(t, sub, "p2", "actor-eq-p2-", 7000)
	if _, err := hypothesis.Promote(ctx, sub, hypothesis.PromoteOptions{
		FormationEventHash: promForm1, PromotedAt: 8000, CadenceSeconds: 60,
	}, nil); err != nil {
		t.Fatalf("Promote 1: %v", err)
	}
	if _, err := hypothesis.Promote(ctx, sub, hypothesis.PromoteOptions{
		FormationEventHash: promForm2, PromotedAt: 9000, CadenceSeconds: 60,
	}, nil); err != nil {
		t.Fatalf("Promote 2: %v", err)
	}

	counts, err := CountByState(ctx, sub, ListOptions{})
	if err != nil {
		t.Fatalf("CountByState: %v", err)
	}
	for state, count := range counts.ByState {
		list, err := ListHypotheses(ctx, sub, ListOptions{StateFilter: state})
		if err != nil {
			t.Fatalf("ListHypotheses(state=%q): %v", state, err)
		}
		if len(list) != count {
			t.Errorf("equivalence violated for state=%q: counts=%d, list=%d",
				state, count, len(list))
		}
	}
}

func TestCountByStateTotalEqualsSumOfByState(t *testing.T) {
	// Structural invariant: sum of ByState values equals Total.
	sub := newSubstrate(t)
	ctx := context.Background()
	for i, desc := range []string{"a", "b", "c"} {
		_ = formCluster(t, sub, desc, "actor-sum-"+desc+"-", int64(1000*(i+1)))
	}
	got, err := CountByState(ctx, sub, ListOptions{})
	if err != nil {
		t.Fatalf("CountByState: %v", err)
	}
	sum := 0
	for _, v := range got.ByState {
		sum += v
	}
	if sum != got.Total {
		t.Errorf("sum of ByState (%d) != Total (%d)", sum, got.Total)
	}
}
