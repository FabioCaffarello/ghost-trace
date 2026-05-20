package projection

import (
	"context"
	"testing"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/hypothesis"
)

func TestWindowFilterPureFormation(t *testing.T) {
	sub := newSubstrate(t)
	// Formation event_time is controlled by formCluster's
	// declaredAtBase argument; the formation itself uses the FormAll
	// fixed-time clock (declaredAtBase+10) for its event_time.
	formInWindow := formCluster(t, sub, "alpha", "actor-w-in-", 1000)
	formOutside := formCluster(t, sub, "beta", "actor-w-out-", 5000)

	ctx := context.Background()
	// Formation event_time = max(declared_at) per session-descriptor-shared-v1
	// (per session_descriptor_shared_v1.go — caller-supplied now is
	// intentionally ignored to preserve hypothesis-identity stability).
	// formCluster ingests two DeclaredSession messages with
	// declared_at = declaredAtBase and declaredAtBase+1; so
	// formation_at = declaredAtBase + 1.
	// formInWindow's formation_at = 1001; formOutside's formation_at = 5001.

	// Window [1000, 2000]: only formInWindow.
	got, err := ListHypotheses(ctx, sub, ListOptions{TimeAfterNs: 1000, TimeBeforeNs: 2000})
	if err != nil {
		t.Fatalf("ListHypotheses: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d, want 1", len(got))
	}
	if got[0].FormationHash != formInWindow {
		t.Errorf("got %x, want %x", got[0].FormationHash, formInWindow)
	}

	// Window [4000, 6000]: only formOutside.
	got, err = ListHypotheses(ctx, sub, ListOptions{TimeAfterNs: 4000, TimeBeforeNs: 6000})
	if err != nil {
		t.Fatalf("ListHypotheses: %v", err)
	}
	if len(got) != 1 || got[0].FormationHash != formOutside {
		t.Errorf("window [4000,6000]: got %d projections, expected only formOutside", len(got))
	}

	// Window [10000, 20000]: empty.
	got, err = ListHypotheses(ctx, sub, ListOptions{TimeAfterNs: 10000, TimeBeforeNs: 20000})
	if err != nil {
		t.Fatalf("ListHypotheses: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("empty window: got %d, want 0", len(got))
	}
}

func TestWindowFilterLatestEventSemantic(t *testing.T) {
	// A hypothesis with formation at t=1001 and dissolution at
	// t=50000 has LATEST event = 50000. The window-filter semantic
	// per §0054 matches against the latest event, NOT "any event in
	// window". So a window of [40000, 60000] INCLUDES this
	// projection even though its formation is outside the window.
	sub := newSubstrate(t)
	ctx := context.Background()
	form := formCluster(t, sub, "alpha", "actor-w-lat-", 1000)
	if _, err := hypothesis.Dissolve(ctx, sub, hypothesis.DissolveOptions{
		FormationEventHash: form,
		DissolvedAt:        50000,
	}, nil); err != nil {
		t.Fatalf("Dissolve: %v", err)
	}

	// Window matching the dissolution but excluding the formation.
	got, err := ListHypotheses(ctx, sub, ListOptions{TimeAfterNs: 40000, TimeBeforeNs: 60000})
	if err != nil {
		t.Fatalf("ListHypotheses: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d, want 1 (latest event in window)", len(got))
	}
	if got[0].State != StateDissolved {
		t.Errorf("state: got %q, want dissolved", got[0].State)
	}

	// Window matching the formation but excluding the dissolution —
	// per latest-event semantic, this projection's LATEST event is
	// the dissolution, so it falls OUTSIDE the window.
	got, err = ListHypotheses(ctx, sub, ListOptions{TimeAfterNs: 500, TimeBeforeNs: 2000})
	if err != nil {
		t.Fatalf("ListHypotheses: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("formation-only window: got %d, want 0 (latest event is dissolution, outside window)", len(got))
	}
}

func TestWindowFilterOpenLowerBound(t *testing.T) {
	// TimeAfterNs=0 disables the lower bound — everything with
	// latest event <= TimeBeforeNs returns.
	sub := newSubstrate(t)
	_ = formCluster(t, sub, "alpha", "actor-w-lo-", 1000)
	_ = formCluster(t, sub, "beta", "actor-w-lo-b-", 5000)
	_ = formCluster(t, sub, "gamma", "actor-w-lo-g-", 10000)

	ctx := context.Background()
	got, err := ListHypotheses(ctx, sub, ListOptions{TimeBeforeNs: 7000})
	if err != nil {
		t.Fatalf("ListHypotheses: %v", err)
	}
	// alpha formation_at=1001, beta formation_at=5001 (both ≤ 7000).
	// gamma formation_at=10001 > 7000.
	if len(got) != 2 {
		t.Errorf("got %d, want 2 (alpha + beta)", len(got))
	}
}

func TestWindowFilterOpenUpperBound(t *testing.T) {
	// TimeBeforeNs=0 disables the upper bound — everything with
	// latest event >= TimeAfterNs returns.
	sub := newSubstrate(t)
	_ = formCluster(t, sub, "alpha", "actor-w-up-", 1000)
	_ = formCluster(t, sub, "beta", "actor-w-up-b-", 5000)
	_ = formCluster(t, sub, "gamma", "actor-w-up-g-", 10000)

	ctx := context.Background()
	got, err := ListHypotheses(ctx, sub, ListOptions{TimeAfterNs: 4000})
	if err != nil {
		t.Fatalf("ListHypotheses: %v", err)
	}
	// alpha 1001 < 4000 (excluded). beta 5001, gamma 10001 (included).
	if len(got) != 2 {
		t.Errorf("got %d, want 2 (beta + gamma)", len(got))
	}
}

func TestWindowFilterAppliesToCountByState(t *testing.T) {
	// CountByState honors the same window filter; equivalence
	// invariant still holds: count[s] == len(list-filter).
	sub := newSubstrate(t)
	ctx := context.Background()

	// Three formations across a wide time range.
	_ = formCluster(t, sub, "f1", "actor-cw-1-", 1000)
	formProm := formCluster(t, sub, "f2", "actor-cw-2-", 5000)
	if _, err := hypothesis.Promote(ctx, sub, hypothesis.PromoteOptions{
		FormationEventHash: formProm,
		PromotedAt:         100000,
		CadenceSeconds:     60,
	}, nil); err != nil {
		t.Fatalf("Promote: %v", err)
	}
	_ = formCluster(t, sub, "f3", "actor-cw-3-", 200000)

	// Window [50000, 150000] catches formProm's promotion (latest event = 100000).
	// Excludes f1 (latest = 1001) and f3 (latest = 200001).
	opts := ListOptions{TimeAfterNs: 50000, TimeBeforeNs: 150000}
	counts, err := CountByState(ctx, sub, opts)
	if err != nil {
		t.Fatalf("CountByState: %v", err)
	}
	if counts.Total != 1 {
		t.Errorf("Total in window: got %d, want 1", counts.Total)
	}
	if counts.ByState[StatePromoted] != 1 {
		t.Errorf("ByState[promoted]: got %d, want 1", counts.ByState[StatePromoted])
	}
	if counts.ByState[StateForming] != 0 {
		t.Errorf("ByState[forming]: got %d, want 0 (filtered out by window)", counts.ByState[StateForming])
	}

	// Equivalence invariant: count[promoted] == len(list with state=promoted, same window).
	list, err := ListHypotheses(ctx, sub, ListOptions{
		StateFilter:  StatePromoted,
		TimeAfterNs:  opts.TimeAfterNs,
		TimeBeforeNs: opts.TimeBeforeNs,
	})
	if err != nil {
		t.Fatalf("ListHypotheses: %v", err)
	}
	if len(list) != counts.ByState[StatePromoted] {
		t.Errorf("equivalence violated: count[promoted]=%d, list=%d",
			counts.ByState[StatePromoted], len(list))
	}
}

func TestWindowFilterBoundsInclusive(t *testing.T) {
	// Boundary case: a projection whose latest event_time exactly
	// equals TimeAfterNs or TimeBeforeNs is INCLUDED.
	sub := newSubstrate(t)
	form := formCluster(t, sub, "alpha", "actor-w-bnd-", 1000)
	// formation_at = 1010.

	ctx := context.Background()

	// First inspect the projection to recover its actual latest event_time.
	proj, err := ProjectHypothesis(ctx, sub, form)
	if err != nil {
		t.Fatalf("ProjectHypothesis: %v", err)
	}
	if len(proj.LifecycleHistory) == 0 {
		t.Fatal("expected at least the formation entry")
	}
	latest := proj.LifecycleHistory[len(proj.LifecycleHistory)-1].EventTime
	t.Logf("formation latest event_time = %d", latest)

	// Window starts exactly at latest → included.
	got, err := ListHypotheses(ctx, sub, ListOptions{TimeAfterNs: latest})
	if err != nil {
		t.Fatalf("ListHypotheses: %v", err)
	}
	if len(got) != 1 || got[0].FormationHash != form {
		t.Errorf("TimeAfterNs=%d (boundary): got %d projections", latest, len(got))
	}

	// Window ends exactly at 1010 → included.
	got, err = ListHypotheses(ctx, sub, ListOptions{TimeBeforeNs: 1010})
	if err != nil {
		t.Fatalf("ListHypotheses: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("TimeBeforeNs=1010 (boundary): got %d projections", len(got))
	}

	// Window starts at 1011 → excluded.
	got, err = ListHypotheses(ctx, sub, ListOptions{TimeAfterNs: 1011})
	if err != nil {
		t.Fatalf("ListHypotheses: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("TimeAfterNs=1011 (above boundary): got %d, want 0", len(got))
	}
}

func TestWindowFilterEquivalenceAcrossAllStates(t *testing.T) {
	// Apply window filter; assert per-state counts equal per-state
	// list lengths for every State value. Strongest form of the
	// equivalence invariant.
	sub := newSubstrate(t)
	ctx := context.Background()

	// Populate with mixed states; some lifecycle events at varying times.
	_ = formCluster(t, sub, "f-form", "actor-eq-form-", 1000)
	formProm := formCluster(t, sub, "f-prom", "actor-eq-prom-", 2000)
	if _, err := hypothesis.Promote(ctx, sub, hypothesis.PromoteOptions{
		FormationEventHash: formProm, PromotedAt: 50000, CadenceSeconds: 60,
	}, nil); err != nil {
		t.Fatalf("Promote: %v", err)
	}
	formDiss := formCluster(t, sub, "f-diss", "actor-eq-diss-", 3000)
	if _, err := hypothesis.Dissolve(ctx, sub, hypothesis.DissolveOptions{
		FormationEventHash: formDiss, DissolvedAt: 60000,
	}, nil); err != nil {
		t.Fatalf("Dissolve: %v", err)
	}

	opts := ListOptions{TimeAfterNs: 40000, TimeBeforeNs: 70000}
	counts, err := CountByState(ctx, sub, opts)
	if err != nil {
		t.Fatalf("CountByState: %v", err)
	}
	for state := range counts.ByState {
		stateOpts := opts
		stateOpts.StateFilter = state
		list, err := ListHypotheses(ctx, sub, stateOpts)
		if err != nil {
			t.Fatalf("ListHypotheses(%q): %v", state, err)
		}
		if len(list) != counts.ByState[state] {
			t.Errorf("equivalence for state=%q: counts=%d, list=%d",
				state, counts.ByState[state], len(list))
		}
	}
}
