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

	bucketSum := 0
	for _, n := range combined.ByState {
		bucketSum += n
	}
	if bucketSum != combined.Total {
		t.Errorf("combined.Total (%d) != sum(by_state buckets) (%d)", combined.Total, bucketSum)
	}
}

func TestCombineCountsPopulatesAllStateKeys(t *testing.T) {
	partial := projection.StateCounts{
		Total: 1,
		ByState: map[projection.State]int{
			projection.StateForming: 1,
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
}

func TestAggregateLatenciesEmpty(t *testing.T) {
	got := aggregateLatencies(nil)
	if got.SampleCount != 0 {
		t.Errorf("SampleCount: got %d, want 0", got.SampleCount)
	}
	if got.MinNs != nil || got.P50Ns != nil || got.P90Ns != nil || got.MaxNs != nil {
		t.Errorf("zero-sample aggregate must have nil percentile fields; got %+v", got)
	}
}

func TestAggregateLatenciesSingleSample(t *testing.T) {
	got := aggregateLatencies([]int64{42})
	if got.SampleCount != 1 {
		t.Errorf("SampleCount: got %d, want 1", got.SampleCount)
	}
	for label, p := range map[string]*int64{"min": got.MinNs, "p50": got.P50Ns, "p90": got.P90Ns, "max": got.MaxNs} {
		if p == nil {
			t.Errorf("%s: got nil, want non-nil", label)
			continue
		}
		if *p != 42 {
			t.Errorf("%s: got %d, want 42", label, *p)
		}
	}
}

func TestAggregateLatenciesNearestRank(t *testing.T) {
	// Sample [10, 20, 30, 40, 50]; n=5.
	// P50: ceil(50*5/100) - 1 = ceil(2.5) - 1 = 3 - 1 = 2 → sorted[2] = 30
	// P90: ceil(90*5/100) - 1 = ceil(4.5) - 1 = 5 - 1 = 4 → sorted[4] = 50
	got := aggregateLatencies([]int64{30, 10, 50, 40, 20}) // unsorted input
	if got.SampleCount != 5 {
		t.Errorf("SampleCount: got %d, want 5", got.SampleCount)
	}
	if *got.MinNs != 10 {
		t.Errorf("Min: got %d, want 10", *got.MinNs)
	}
	if *got.MaxNs != 50 {
		t.Errorf("Max: got %d, want 50", *got.MaxNs)
	}
	if *got.P50Ns != 30 {
		t.Errorf("P50: got %d, want 30", *got.P50Ns)
	}
	if *got.P90Ns != 50 {
		t.Errorf("P90: got %d, want 50", *got.P90Ns)
	}
}

func TestAggregateLatenciesInputNotMutated(t *testing.T) {
	// Caller's sample slice must not be reordered by aggregation.
	input := []int64{30, 10, 50, 40, 20}
	want := []int64{30, 10, 50, 40, 20}
	_ = aggregateLatencies(input)
	for i, v := range input {
		if v != want[i] {
			t.Errorf("input mutated at index %d: got %d, want %d", i, v, want[i])
		}
	}
}

func TestCombineSamplesUnionsAcrossSubtypes(t *testing.T) {
	bc := latencySamples{
		FormationToFirstPromotion:       []int64{10, 20},
		LatestPromotionToLatestDemotion: []int64{5},
		FormationToDissolution:          nil,
	}
	ag := latencySamples{
		FormationToFirstPromotion:       []int64{30},
		LatestPromotionToLatestDemotion: nil,
		FormationToDissolution:          []int64{100, 200},
	}
	combined := combineSamples(bc, ag)
	if len(combined.FormationToFirstPromotion) != 3 {
		t.Errorf("FormationToFirstPromotion: got %d, want 3", len(combined.FormationToFirstPromotion))
	}
	if len(combined.LatestPromotionToLatestDemotion) != 1 {
		t.Errorf("LatestPromotionToLatestDemotion: got %d, want 1", len(combined.LatestPromotionToLatestDemotion))
	}
	if len(combined.FormationToDissolution) != 2 {
		t.Errorf("FormationToDissolution: got %d, want 2", len(combined.FormationToDissolution))
	}
}

func TestCombinedPercentilesAreExactNotApproximated(t *testing.T) {
	// Per §0079: combined percentiles are computed from the UNION of
	// per-subtype samples, NOT from per-subtype aggregates. This test
	// catches accidental regression to a max-of-percentiles
	// approximation.
	bcSamples := []int64{10, 20, 30}     // p50 over bc alone = 20
	agSamples := []int64{40, 50, 60, 70} // p50 over ag alone = 50
	combinedSamples := append([]int64(nil), bcSamples...)
	combinedSamples = append(combinedSamples, agSamples...)
	// Union [10,20,30,40,50,60,70]; n=7; p50 index = ceil(7*0.5)-1 = 4-1 = 3 → 40
	got := aggregateLatencies(combinedSamples)
	if *got.P50Ns != 40 {
		t.Errorf("combined p50: got %d, want 40 (exact from union, not max-of-parts which would be 50)", *got.P50Ns)
	}
}
