package projection

import (
	"context"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// StateCounts is the aggregate count of HypothesisProjections
// grouped by their computed State value, per decision-log §0053.
// Total is the sum of all per-state counts and equals the number
// of formations in the substrate.
//
// The ByState map is fully populated — every State enum value
// appears as a key, with value 0 when no projections fall into
// that state. This keeps the JSON shape predictable for operators
// consuming the output (a missing key would be indistinguishable
// from "zero" on the wire).
type StateCounts struct {
	Total   int           `json:"total"`
	ByState map[State]int `json:"by_state"`
}

// CountByState returns the aggregate StateCounts over every
// formation in the substrate. Computed by invoking ProjectAll and
// aggregating the per-formation State values; same substrate-linear
// walk cost as ProjectAll (the aggregation step is O(N_formations)
// over the already-materialized projection map).
//
// Per the §0053 equivalence invariant (tested in counts_test.go):
// for every State value, CountByState.ByState[s] equals
// len(ListHypotheses(ctx, sub, ListOptions{StateFilter: s})). The
// invariant defends against precedence-rule drift between the
// count path and the list path.
func CountByState(ctx context.Context, sub *substrate.Substrate) (StateCounts, error) {
	all, err := ProjectAll(ctx, sub)
	if err != nil {
		return StateCounts{}, err
	}
	counts := StateCounts{
		Total: len(all),
		ByState: map[State]int{
			StateForming:    0,
			StatePromoted:   0,
			StateDemoted:    0,
			StateDissolved:  0,
			StateMergedInto: 0,
			StateSplitInto:  0,
		},
	}
	for _, proj := range all {
		counts.ByState[proj.State]++
	}
	return counts, nil
}
