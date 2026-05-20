package projection

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/hypothesis"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

func TestListEmptySubstrate(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	got, err := ListHypotheses(ctx, sub, ListOptions{})
	if err != nil {
		t.Fatalf("ListHypotheses: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("empty substrate: got %d projections, want 0", len(got))
	}
}

func TestListAllStatesPresent(t *testing.T) {
	sub := newSubstrate(t)
	// Build a substrate with each terminal state at least once.
	formForm := formCluster(t, sub, "form-only", "actor-forming-", 1000)
	formProm := formCluster(t, sub, "prom-only", "actor-promoted-", 2000)
	formDiss := formCluster(t, sub, "diss-only", "actor-dissolved-", 3000)

	ctx := context.Background()
	if _, err := hypothesis.Promote(ctx, sub, hypothesis.PromoteOptions{
		FormationEventHash: formProm,
		PromotedAt:         4000,
		CadenceSeconds:     60,
	}, nil); err != nil {
		t.Fatalf("Promote: %v", err)
	}
	if _, err := hypothesis.Dissolve(ctx, sub, hypothesis.DissolveOptions{
		FormationEventHash: formDiss,
		DissolvedAt:        5000,
	}, nil); err != nil {
		t.Fatalf("Dissolve: %v", err)
	}

	got, err := ListHypotheses(ctx, sub, ListOptions{})
	if err != nil {
		t.Fatalf("ListHypotheses: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d projections, want 3", len(got))
	}

	// Map by formation hash for assertion lookup.
	byHash := map[[32]byte]HypothesisProjection{}
	for _, p := range got {
		byHash[p.FormationHash] = p
	}
	cases := []struct {
		name     string
		hash     [32]byte
		wantSt   State
	}{
		{"forming", formForm, StateForming},
		{"promoted", formProm, StatePromoted},
		{"dissolved", formDiss, StateDissolved},
	}
	for _, tc := range cases {
		p, ok := byHash[tc.hash]
		if !ok {
			t.Errorf("%s: projection missing", tc.name)
			continue
		}
		if p.State != tc.wantSt {
			t.Errorf("%s: state %q, want %q", tc.name, p.State, tc.wantSt)
		}
	}
}

func TestListStateFilter(t *testing.T) {
	sub := newSubstrate(t)
	_ = formCluster(t, sub, "alpha", "actor-flt-a-", 1000)
	beta := formCluster(t, sub, "beta", "actor-flt-b-", 2000)
	_ = formCluster(t, sub, "gamma", "actor-flt-g-", 3000)
	ctx := context.Background()
	if _, err := hypothesis.Promote(ctx, sub, hypothesis.PromoteOptions{
		FormationEventHash: beta,
		PromotedAt:         4000,
		CadenceSeconds:     60,
	}, nil); err != nil {
		t.Fatalf("Promote: %v", err)
	}

	promoted, err := ListHypotheses(ctx, sub, ListOptions{StateFilter: StatePromoted})
	if err != nil {
		t.Fatalf("ListHypotheses promoted: %v", err)
	}
	if len(promoted) != 1 {
		t.Fatalf("StateFilter=promoted: got %d, want 1", len(promoted))
	}
	if promoted[0].FormationHash != beta {
		t.Errorf("promoted projection: got %x, want %x", promoted[0].FormationHash, beta)
	}

	forming, err := ListHypotheses(ctx, sub, ListOptions{StateFilter: StateForming})
	if err != nil {
		t.Fatalf("ListHypotheses forming: %v", err)
	}
	if len(forming) != 2 {
		t.Errorf("StateFilter=forming: got %d, want 2 (alpha + gamma)", len(forming))
	}
}

func TestListOrderingStable(t *testing.T) {
	// Ascending FormationHash lex order, substrate-position-
	// independent. Two repeated calls return the same order.
	sub := newSubstrate(t)
	for i, desc := range []string{"alpha", "beta", "gamma", "delta", "epsilon"} {
		_ = formCluster(t, sub, desc, "actor-ord-"+desc+"-", int64(1000*(i+1)))
	}
	ctx := context.Background()

	first, err := ListHypotheses(ctx, sub, ListOptions{})
	if err != nil {
		t.Fatalf("first ListHypotheses: %v", err)
	}
	second, err := ListHypotheses(ctx, sub, ListOptions{})
	if err != nil {
		t.Fatalf("second ListHypotheses: %v", err)
	}
	if len(first) != len(second) {
		t.Fatalf("length divergence: %d vs %d", len(first), len(second))
	}
	for i := range first {
		if first[i].FormationHash != second[i].FormationHash {
			t.Errorf("ordering divergence at %d: %x vs %x",
				i, first[i].FormationHash, second[i].FormationHash)
		}
	}
	// Ascending check.
	for i := 1; i < len(first); i++ {
		if bytes.Compare(first[i-1].FormationHash[:], first[i].FormationHash[:]) >= 0 {
			t.Errorf("not ascending at %d: %x vs %x",
				i, first[i-1].FormationHash, first[i].FormationHash)
		}
	}
}

func TestListLimitOffset(t *testing.T) {
	sub := newSubstrate(t)
	for i, desc := range []string{"a", "b", "c", "d", "e"} {
		_ = formCluster(t, sub, desc, "actor-page-"+desc+"-", int64(1000*(i+1)))
	}
	ctx := context.Background()

	all, _ := ListHypotheses(ctx, sub, ListOptions{})
	if len(all) != 5 {
		t.Fatalf("want 5 formations; got %d", len(all))
	}

	limited, _ := ListHypotheses(ctx, sub, ListOptions{Limit: 2})
	if len(limited) != 2 {
		t.Errorf("Limit=2: got %d, want 2", len(limited))
	}
	// Limit preserves the head of the canonical order.
	if limited[0].FormationHash != all[0].FormationHash {
		t.Errorf("Limit=2 head: %x, want %x", limited[0].FormationHash, all[0].FormationHash)
	}

	offset, _ := ListHypotheses(ctx, sub, ListOptions{Offset: 2})
	if len(offset) != 3 {
		t.Errorf("Offset=2: got %d, want 3", len(offset))
	}
	if offset[0].FormationHash != all[2].FormationHash {
		t.Errorf("Offset=2 head: %x, want %x", offset[0].FormationHash, all[2].FormationHash)
	}

	page, _ := ListHypotheses(ctx, sub, ListOptions{Limit: 2, Offset: 2})
	if len(page) != 2 {
		t.Errorf("Limit=2 Offset=2: got %d, want 2", len(page))
	}
	if page[0].FormationHash != all[2].FormationHash {
		t.Errorf("page[0]: %x, want %x", page[0].FormationHash, all[2].FormationHash)
	}
	if page[1].FormationHash != all[3].FormationHash {
		t.Errorf("page[1]: %x, want %x", page[1].FormationHash, all[3].FormationHash)
	}
}

func TestListOffsetBeyondLength(t *testing.T) {
	sub := newSubstrate(t)
	for i, desc := range []string{"a", "b"} {
		_ = formCluster(t, sub, desc, "actor-beyond-"+desc+"-", int64(1000*(i+1)))
	}
	ctx := context.Background()
	got, err := ListHypotheses(ctx, sub, ListOptions{Offset: 10})
	if err != nil {
		t.Fatalf("ListHypotheses: %v", err)
	}
	if got != nil {
		t.Errorf("Offset beyond length: got %d, want nil", len(got))
	}
}

func TestListEquivalentToPerFormationProject(t *testing.T) {
	// ProjectAll + ListHypotheses must produce per-formation
	// projections byte-for-byte equivalent to ProjectHypothesis for
	// the same formation. This is the multi-vs-single equivalence
	// invariant.
	sub := newSubstrate(t)
	alpha := formCluster(t, sub, "alpha", "actor-eq-a-", 1000)
	beta := formCluster(t, sub, "beta", "actor-eq-b-", 2000)
	gamma := formCluster(t, sub, "gamma", "actor-eq-g-", 3000)
	ctx := context.Background()

	promRep, err := hypothesis.Promote(ctx, sub, hypothesis.PromoteOptions{
		FormationEventHash: alpha,
		PromotedAt:         4000,
		CadenceSeconds:     60,
		Reason:             "promoted",
	}, nil)
	if err != nil {
		t.Fatalf("Promote: %v", err)
	}
	var promHash [32]byte
	if raw, err := hexDecode(promRep.PromotionEventHashHex); err == nil {
		copy(promHash[:], raw)
	}
	if _, err := hypothesis.Demote(ctx, sub, hypothesis.DemoteOptions{
		PromotionEventHash: promHash,
		DemotedAt:          5000,
	}, nil); err != nil {
		t.Fatalf("Demote: %v", err)
	}
	if _, err := hypothesis.Merge(ctx, sub, hypothesis.MergeOptions{
		AntecedentAFormationHash: beta,
		AntecedentBFormationHash: gamma,
		ProducedFormationHash:    alpha,
		MergedAt:                 6000,
	}, nil); err != nil {
		t.Fatalf("Merge: %v", err)
	}

	all, err := ProjectAll(ctx, sub)
	if err != nil {
		t.Fatalf("ProjectAll: %v", err)
	}
	for _, formHash := range [][32]byte{alpha, beta, gamma} {
		single, err := ProjectHypothesis(ctx, sub, formHash)
		if err != nil {
			t.Fatalf("ProjectHypothesis(%x): %v", formHash, err)
		}
		multi, ok := all[formHash]
		if !ok {
			t.Errorf("ProjectAll missing %x", formHash)
			continue
		}
		if single.State != multi.State {
			t.Errorf("%x: single State=%q, multi State=%q", formHash, single.State, multi.State)
		}
		if len(single.LifecycleHistory) != len(multi.LifecycleHistory) {
			t.Errorf("%x: single history len=%d, multi=%d",
				formHash, len(single.LifecycleHistory), len(multi.LifecycleHistory))
			continue
		}
		for i := range single.LifecycleHistory {
			if single.LifecycleHistory[i] != multi.LifecycleHistory[i] {
				t.Errorf("%x: history[%d] divergence: single=%+v multi=%+v",
					formHash, i, single.LifecycleHistory[i], multi.LifecycleHistory[i])
			}
		}
	}
}

func TestListIncludesAllSixLifecycleStates(t *testing.T) {
	// Stress test: substrate carrying all six possible end-states.
	sub := newSubstrate(t)
	ctx := context.Background()
	formForm := formCluster(t, sub, "f1", "actor-six-f-", 1000)
	formProm := formCluster(t, sub, "f2", "actor-six-p-", 2000)
	formDemo := formCluster(t, sub, "f3", "actor-six-d-", 3000)
	formDiss := formCluster(t, sub, "f4", "actor-six-x-", 4000)
	formMerge := formCluster(t, sub, "f5", "actor-six-m-", 5000)
	formSplit := formCluster(t, sub, "f6", "actor-six-s-", 6000)
	// Two extras for merge/split targets.
	extraGamma := formCluster(t, sub, "fg", "actor-six-g-", 7000)
	extraDelta := formCluster(t, sub, "fd", "actor-six-q-", 8000)

	// Promote formProm.
	if _, err := hypothesis.Promote(ctx, sub, hypothesis.PromoteOptions{
		FormationEventHash: formProm,
		PromotedAt:         9000,
		CadenceSeconds:     60,
	}, nil); err != nil {
		t.Fatalf("Promote formProm: %v", err)
	}
	// Promote + demote formDemo.
	promRep, _ := hypothesis.Promote(ctx, sub, hypothesis.PromoteOptions{
		FormationEventHash: formDemo,
		PromotedAt:         10000,
		CadenceSeconds:     60,
	}, nil)
	var pHash [32]byte
	if raw, _ := hexDecode(promRep.PromotionEventHashHex); true {
		copy(pHash[:], raw)
	}
	if _, err := hypothesis.Demote(ctx, sub, hypothesis.DemoteOptions{
		PromotionEventHash: pHash,
		DemotedAt:          11000,
	}, nil); err != nil {
		t.Fatalf("Demote formDemo: %v", err)
	}
	// Dissolve formDiss.
	if _, err := hypothesis.Dissolve(ctx, sub, hypothesis.DissolveOptions{
		FormationEventHash: formDiss,
		DissolvedAt:        12000,
	}, nil); err != nil {
		t.Fatalf("Dissolve formDiss: %v", err)
	}
	// Merge formMerge + extraGamma → extraDelta.
	if _, err := hypothesis.Merge(ctx, sub, hypothesis.MergeOptions{
		AntecedentAFormationHash: formMerge,
		AntecedentBFormationHash: extraGamma,
		ProducedFormationHash:    extraDelta,
		MergedAt:                 13000,
	}, nil); err != nil {
		t.Fatalf("Merge formMerge: %v", err)
	}
	// Split formSplit → [extraGamma, extraDelta]. NOTE: extraGamma is
	// also a merge antecedent above; this exercises a formation
	// reached by both a merge and a split (as successor of one,
	// antecedent of another). The split here targets formSplit's
	// projection only; the substrate accepts the discrete event
	// regardless of cross-operation ordering per §0048 carry-forward.
	if _, err := hypothesis.Split(ctx, sub, hypothesis.SplitOptions{
		AntecedentFormationHash:  formSplit,
		SuccessorFormationHashes: [][32]byte{extraGamma, extraDelta},
		SplitAt:                  14000,
	}, nil); err != nil {
		t.Fatalf("Split formSplit: %v", err)
	}

	all, err := ListHypotheses(ctx, sub, ListOptions{})
	if err != nil {
		t.Fatalf("ListHypotheses: %v", err)
	}
	if len(all) != 8 {
		t.Errorf("got %d projections, want 8 (six end-states + two extras)", len(all))
	}

	byHash := map[[32]byte]State{}
	for _, p := range all {
		byHash[p.FormationHash] = p.State
	}
	checks := []struct {
		name   string
		hash   [32]byte
		wantSt State
	}{
		{"forming", formForm, StateForming},
		{"promoted", formProm, StatePromoted},
		{"demoted", formDemo, StateDemoted},
		{"dissolved", formDiss, StateDissolved},
		{"merged_into", formMerge, StateMergedInto},
		{"split_into", formSplit, StateSplitInto},
	}
	for _, tc := range checks {
		got, ok := byHash[tc.hash]
		if !ok {
			t.Errorf("%s: projection missing", tc.name)
			continue
		}
		if got != tc.wantSt {
			t.Errorf("%s: state %q, want %q", tc.name, got, tc.wantSt)
		}
	}
}
