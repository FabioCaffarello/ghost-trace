package main

import (
	"testing"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/projection"
)

func TestCombineCountsEmpty(t *testing.T) {
	got := combineCounts()
	if got.Total != 0 {
		t.Errorf("Total: got %d, want 0", got.Total)
	}
	wantStates := []projection.State{
		projection.StateForming, projection.StatePromoted, projection.StateDemoted,
		projection.StateDissolved, projection.StateMergedInto, projection.StateSplitInto,
	}
	for _, s := range wantStates {
		if _, ok := got.ByState[s]; !ok {
			t.Errorf("by_state missing key %q (predictable-wire-shape commitment)", s)
		}
		if got.ByState[s] != 0 {
			t.Errorf("by_state[%q]: got %d, want 0", s, got.ByState[s])
		}
	}
}

func TestCombineCountsSumsTotalsAndPerStateBuckets(t *testing.T) {
	// Three subtype partial counts; combined.Total == sum(part.Total)
	// and combined.ByState[s] == sum(part.ByState[s]) for every state.
	bc := projection.StateCounts{
		Total: 5,
		ByState: map[projection.State]int{
			projection.StateForming:    3,
			projection.StatePromoted:   1,
			projection.StateDemoted:    0,
			projection.StateDissolved:  1,
			projection.StateMergedInto: 0,
			projection.StateSplitInto:  0,
		},
	}
	ag := projection.StateCounts{
		Total: 4,
		ByState: map[projection.State]int{
			projection.StateForming:    1,
			projection.StatePromoted:   2,
			projection.StateDemoted:    1,
			projection.StateDissolved:  0,
			projection.StateMergedInto: 0,
			projection.StateSplitInto:  0,
		},
	}
	ch := projection.StateCounts{
		Total: 2,
		ByState: map[projection.State]int{
			projection.StateForming:    0,
			projection.StatePromoted:   0,
			projection.StateDemoted:    0,
			projection.StateDissolved:  0,
			projection.StateMergedInto: 1,
			projection.StateSplitInto:  1,
		},
	}

	combined := combineCounts(bc, ag, ch)

	if combined.Total != 11 {
		t.Errorf("Total: got %d, want 11", combined.Total)
	}
	wantByState := map[projection.State]int{
		projection.StateForming:    4,
		projection.StatePromoted:   3,
		projection.StateDemoted:    1,
		projection.StateDissolved:  1,
		projection.StateMergedInto: 1,
		projection.StateSplitInto:  1,
	}
	for s, want := range wantByState {
		if combined.ByState[s] != want {
			t.Errorf("by_state[%q]: got %d, want %d", s, combined.ByState[s], want)
		}
	}

	// Equivalence invariant: combined.Total == sum of per-state buckets.
	bucketSum := 0
	for _, n := range combined.ByState {
		bucketSum += n
	}
	if bucketSum != combined.Total {
		t.Errorf("combined.Total (%d) != sum(by_state buckets) (%d)", combined.Total, bucketSum)
	}
}

func TestCombineCountsPopulatesAllStateKeys(t *testing.T) {
	// Combine a subtype that's missing a State key (a defensive
	// partial input). combined.ByState must still carry every State.
	partial := projection.StateCounts{
		Total: 1,
		ByState: map[projection.State]int{
			projection.StateForming: 1,
			// other states omitted
		},
	}
	combined := combineCounts(partial)
	wantStates := []projection.State{
		projection.StateForming, projection.StatePromoted, projection.StateDemoted,
		projection.StateDissolved, projection.StateMergedInto, projection.StateSplitInto,
	}
	for _, s := range wantStates {
		if _, ok := combined.ByState[s]; !ok {
			t.Errorf("by_state missing %q", s)
		}
	}
	if combined.Total != 1 {
		t.Errorf("Total: got %d, want 1", combined.Total)
	}
}
