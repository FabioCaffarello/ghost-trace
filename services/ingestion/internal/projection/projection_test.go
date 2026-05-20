package projection

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/hypothesis"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

func newSubstrate(t *testing.T) *substrate.Substrate {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })
	return sub
}

// formCluster populates the substrate with two actors sharing the
// supplied descriptor, runs FormAll, and returns the resulting
// formation hash.
func formCluster(t *testing.T, sub *substrate.Substrate, descriptor string, actorPrefix string, declaredAtBase int64) [32]byte {
	t.Helper()
	ctx := context.Background()
	in := ingest.New(sub, func() time.Time { return time.Unix(0, declaredAtBase) })
	for i := 0; i < 2; i++ {
		msg := &eventsv1.DeclaredSession{
			DeclaredAt:        declaredAtBase + int64(i),
			ActorRef:          actorPrefix + string(rune('a'+i)),
			SessionDescriptor: []byte(descriptor),
		}
		if _, err := in.Append(ctx, msg, msg.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("ingest %s: %v", msg.ActorRef, err)
		}
	}
	if _, err := hypothesis.FormAll(ctx, sub, hypothesis.SessionDescriptorSharedV1{MinClusterSize: 2}, func() time.Time { return time.Unix(0, declaredAtBase+10) }); err != nil {
		t.Fatalf("FormAll: %v", err)
	}
	// Walk to find the formation hash matching this descriptor.
	var formationHash [32]byte
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != "ghosttrace.events.v1.BehavioralClusterFormation" {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return err
		}
		ev := &eventsv1.BehavioralClusterFormation{}
		if err := proto.Unmarshal(payload, ev); err != nil {
			return err
		}
		for _, ar := range ev.ActorRefs {
			if bytes.HasPrefix([]byte(ar), []byte(actorPrefix)) {
				formationHash = row.EventHash
				return nil
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("walk for formation: %v", err)
	}
	if formationHash == ([32]byte{}) {
		t.Fatalf("no formation found for prefix %q", actorPrefix)
	}
	return formationHash
}

func TestProjectFormingState(t *testing.T) {
	sub := newSubstrate(t)
	formation := formCluster(t, sub, "alpha", "actor-form-", 1000)

	proj, err := ProjectHypothesis(context.Background(), sub, formation)
	if err != nil {
		t.Fatalf("ProjectHypothesis: %v", err)
	}
	if proj.State != StateForming {
		t.Errorf("state: got %q, want %q", proj.State, StateForming)
	}
	if proj.LatestPromotion != nil || proj.LatestDemotion != nil ||
		proj.Dissolution != nil || proj.MergedInto != nil || proj.SplitInto != nil {
		t.Errorf("forming projection should have no lifecycle event references: %+v", proj)
	}
	if len(proj.LifecycleHistory) != 1 {
		t.Errorf("lifecycle history: got %d, want 1 (formation only)", len(proj.LifecycleHistory))
	}
}

func TestProjectPromotedState(t *testing.T) {
	sub := newSubstrate(t)
	formation := formCluster(t, sub, "alpha", "actor-prom-", 1000)
	ctx := context.Background()
	if _, err := hypothesis.Promote(ctx, sub, hypothesis.PromoteOptions{
		FormationEventHash: formation,
		PromotedAt:         2000,
		CadenceSeconds:     60,
		Reason:             "promoted-state-test",
	}, nil); err != nil {
		t.Fatalf("Promote: %v", err)
	}

	proj, err := ProjectHypothesis(ctx, sub, formation)
	if err != nil {
		t.Fatalf("ProjectHypothesis: %v", err)
	}
	if proj.State != StatePromoted {
		t.Errorf("state: got %q, want %q", proj.State, StatePromoted)
	}
	if proj.LatestPromotion == nil || proj.LatestPromotion.Reason != "promoted-state-test" {
		t.Errorf("LatestPromotion: %+v", proj.LatestPromotion)
	}
	if proj.LatestDemotion != nil {
		t.Errorf("LatestDemotion unexpectedly populated: %+v", proj.LatestDemotion)
	}
	if len(proj.LifecycleHistory) != 2 {
		t.Errorf("lifecycle history: got %d, want 2 (formation + promotion)", len(proj.LifecycleHistory))
	}
}

func TestProjectDemotedState(t *testing.T) {
	sub := newSubstrate(t)
	formation := formCluster(t, sub, "alpha", "actor-demo-", 1000)
	ctx := context.Background()
	promRep, err := hypothesis.Promote(ctx, sub, hypothesis.PromoteOptions{
		FormationEventHash: formation,
		PromotedAt:         2000,
		CadenceSeconds:     60,
		Reason:             "to-be-demoted",
	}, nil)
	if err != nil {
		t.Fatalf("Promote: %v", err)
	}
	var promHash [32]byte
	if raw, err := hexDecode(promRep.PromotionEventHashHex); err == nil {
		copy(promHash[:], raw)
	} else {
		t.Fatalf("decode hash: %v", err)
	}
	if _, err := hypothesis.Demote(ctx, sub, hypothesis.DemoteOptions{
		PromotionEventHash: promHash,
		DemotedAt:          3000,
		Reason:             "demoted",
	}, nil); err != nil {
		t.Fatalf("Demote: %v", err)
	}

	proj, err := ProjectHypothesis(ctx, sub, formation)
	if err != nil {
		t.Fatalf("ProjectHypothesis: %v", err)
	}
	if proj.State != StateDemoted {
		t.Errorf("state: got %q, want %q", proj.State, StateDemoted)
	}
	if proj.LatestDemotion == nil || proj.LatestDemotion.Reason != "demoted" {
		t.Errorf("LatestDemotion: %+v", proj.LatestDemotion)
	}
}

func TestProjectRepromotedReturnsToPromoted(t *testing.T) {
	// Re-promotion arc: form → promote → demote → promote (second).
	// Latest promotion is the second one; no demotion targets it, so
	// state is Promoted again.
	sub := newSubstrate(t)
	formation := formCluster(t, sub, "alpha", "actor-rep-", 1000)
	ctx := context.Background()

	prom1, _ := hypothesis.Promote(ctx, sub, hypothesis.PromoteOptions{
		FormationEventHash: formation,
		PromotedAt:         2000,
		CadenceSeconds:     60,
		Reason:             "first",
	}, nil)
	var p1 [32]byte
	if raw, _ := hexDecode(prom1.PromotionEventHashHex); true {
		copy(p1[:], raw)
	}
	if _, err := hypothesis.Demote(ctx, sub, hypothesis.DemoteOptions{
		PromotionEventHash: p1,
		DemotedAt:          3000,
		Reason:             "rolling",
	}, nil); err != nil {
		t.Fatalf("Demote: %v", err)
	}
	// Second promotion — different cadence so content-hash differs.
	if _, err := hypothesis.Promote(ctx, sub, hypothesis.PromoteOptions{
		FormationEventHash: formation,
		PromotedAt:         4000,
		CadenceSeconds:     120,
		Reason:             "second",
	}, nil); err != nil {
		t.Fatalf("second Promote: %v", err)
	}

	proj, err := ProjectHypothesis(ctx, sub, formation)
	if err != nil {
		t.Fatalf("ProjectHypothesis: %v", err)
	}
	if proj.State != StatePromoted {
		t.Errorf("re-promoted state: got %q, want %q", proj.State, StatePromoted)
	}
	if proj.LatestPromotion == nil || proj.LatestPromotion.Reason != "second" {
		t.Errorf("LatestPromotion should be the second promotion: %+v", proj.LatestPromotion)
	}
	// LatestDemotion targets the first promotion, not the second; per
	// computeState, only a demotion of the latest promotion sets
	// StateDemoted. Latest promotion has no demotion, so state is Promoted.
	if proj.LatestDemotion != nil {
		t.Errorf("LatestDemotion should be nil (latest promotion has no demotion); got %+v", proj.LatestDemotion)
	}
}

func TestProjectDissolvedState(t *testing.T) {
	sub := newSubstrate(t)
	formation := formCluster(t, sub, "alpha", "actor-diss-", 1000)
	ctx := context.Background()
	if _, err := hypothesis.Dissolve(ctx, sub, hypothesis.DissolveOptions{
		FormationEventHash: formation,
		DissolvedAt:        5000,
		Reason:             "non-existent",
	}, nil); err != nil {
		t.Fatalf("Dissolve: %v", err)
	}

	proj, err := ProjectHypothesis(ctx, sub, formation)
	if err != nil {
		t.Fatalf("ProjectHypothesis: %v", err)
	}
	if proj.State != StateDissolved {
		t.Errorf("state: got %q, want %q", proj.State, StateDissolved)
	}
	if proj.Dissolution == nil || proj.Dissolution.Reason != "non-existent" {
		t.Errorf("Dissolution: %+v", proj.Dissolution)
	}
}

func TestProjectDissolutionBeatsPromotion(t *testing.T) {
	// Precedence rule: dissolution dominates even when a prior
	// promotion is present and not demoted.
	sub := newSubstrate(t)
	formation := formCluster(t, sub, "alpha", "actor-disprom-", 1000)
	ctx := context.Background()
	if _, err := hypothesis.Promote(ctx, sub, hypothesis.PromoteOptions{
		FormationEventHash: formation,
		PromotedAt:         2000,
		CadenceSeconds:     60,
		Reason:             "promoted-then-dissolved",
	}, nil); err != nil {
		t.Fatalf("Promote: %v", err)
	}
	if _, err := hypothesis.Dissolve(ctx, sub, hypothesis.DissolveOptions{
		FormationEventHash: formation,
		DissolvedAt:        3000,
		Reason:             "dissolve-overrides",
	}, nil); err != nil {
		t.Fatalf("Dissolve: %v", err)
	}

	proj, err := ProjectHypothesis(ctx, sub, formation)
	if err != nil {
		t.Fatalf("ProjectHypothesis: %v", err)
	}
	if proj.State != StateDissolved {
		t.Errorf("state: got %q, want %q (dissolution should dominate)", proj.State, StateDissolved)
	}
	// LatestPromotion still recorded for operator inspection.
	if proj.LatestPromotion == nil {
		t.Error("LatestPromotion should still be tracked even when dissolved")
	}
}

func TestProjectMergedIntoState(t *testing.T) {
	// Setup: three formations alpha/beta/gamma; merge alpha + beta → gamma.
	// Projecting alpha returns StateMergedInto with MergedInto populated.
	sub := newSubstrate(t)
	alpha := formCluster(t, sub, "alpha", "actor-merge-a-", 1000)
	beta := formCluster(t, sub, "beta", "actor-merge-b-", 2000)
	gamma := formCluster(t, sub, "gamma", "actor-merge-g-", 3000)
	ctx := context.Background()
	if _, err := hypothesis.Merge(ctx, sub, hypothesis.MergeOptions{
		AntecedentAFormationHash: alpha,
		AntecedentBFormationHash: beta,
		ProducedFormationHash:    gamma,
		MergedAt:                 4000,
		Reason:                   "merge-state-test",
	}, nil); err != nil {
		t.Fatalf("Merge: %v", err)
	}

	proj, err := ProjectHypothesis(ctx, sub, alpha)
	if err != nil {
		t.Fatalf("ProjectHypothesis(alpha): %v", err)
	}
	if proj.State != StateMergedInto {
		t.Errorf("alpha state: got %q, want %q", proj.State, StateMergedInto)
	}
	if proj.MergedInto == nil || proj.MergedInto.Reason != "merge-state-test" {
		t.Errorf("MergedInto: %+v", proj.MergedInto)
	}
	// Gamma (the produced formation) is NOT itself merged-into anything;
	// it's the survivor identity.
	projGamma, err := ProjectHypothesis(ctx, sub, gamma)
	if err != nil {
		t.Fatalf("ProjectHypothesis(gamma): %v", err)
	}
	if projGamma.State != StateForming {
		t.Errorf("gamma (produced formation) state: got %q, want %q (no lifecycle events reach it)", projGamma.State, StateForming)
	}
}

func TestProjectSplitIntoState(t *testing.T) {
	// Setup: three formations; split alpha → [beta, gamma].
	sub := newSubstrate(t)
	alpha := formCluster(t, sub, "alpha", "actor-split-a-", 1000)
	beta := formCluster(t, sub, "beta", "actor-split-b-", 2000)
	gamma := formCluster(t, sub, "gamma", "actor-split-g-", 3000)
	ctx := context.Background()
	if _, err := hypothesis.Split(ctx, sub, hypothesis.SplitOptions{
		AntecedentFormationHash:  alpha,
		SuccessorFormationHashes: [][32]byte{beta, gamma},
		SplitAt:                  4000,
		Reason:                   "split-state-test",
	}, nil); err != nil {
		t.Fatalf("Split: %v", err)
	}

	proj, err := ProjectHypothesis(ctx, sub, alpha)
	if err != nil {
		t.Fatalf("ProjectHypothesis(alpha): %v", err)
	}
	if proj.State != StateSplitInto {
		t.Errorf("alpha state: got %q, want %q", proj.State, StateSplitInto)
	}
	if proj.SplitInto == nil || proj.SplitInto.Reason != "split-state-test" {
		t.Errorf("SplitInto: %+v", proj.SplitInto)
	}
	if len(proj.SplitInto.SuccessorFormationEventHashes) != 2 {
		t.Errorf("split successors: got %d, want 2", len(proj.SplitInto.SuccessorFormationEventHashes))
	}
}

func TestProjectSplitBeatsMerge(t *testing.T) {
	// Setup: alpha + beta → gamma (merge); then gamma → [alpha, beta] (split).
	// Projecting gamma returns StateSplitInto (split beats merge per
	// precedence; merge wasn't recorded against gamma anyway since gamma
	// is the merge's produced formation, not antecedent).
	// The more interesting case: project alpha. Alpha is merged into
	// gamma AND is a successor of gamma's split. Per precedence + the
	// split-successor-targeting semantics: alpha only appears as the
	// antecedent of a split when it IS the antecedent; alpha is a
	// SUCCESSOR in this scenario, which is not what SplitInto tracks
	// (SplitInto tracks: "is this formation the antecedent of a split?").
	// So alpha remains StateMergedInto. This test verifies the
	// distinction.
	sub := newSubstrate(t)
	alpha := formCluster(t, sub, "alpha", "actor-sm-a-", 1000)
	beta := formCluster(t, sub, "beta", "actor-sm-b-", 2000)
	gamma := formCluster(t, sub, "gamma", "actor-sm-g-", 3000)
	ctx := context.Background()
	if _, err := hypothesis.Merge(ctx, sub, hypothesis.MergeOptions{
		AntecedentAFormationHash: alpha,
		AntecedentBFormationHash: beta,
		ProducedFormationHash:    gamma,
		MergedAt:                 4000,
		Reason:                   "first-merge",
	}, nil); err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if _, err := hypothesis.Split(ctx, sub, hypothesis.SplitOptions{
		AntecedentFormationHash:  gamma,
		SuccessorFormationHashes: [][32]byte{alpha, beta},
		SplitAt:                  5000,
		Reason:                   "then-split",
	}, nil); err != nil {
		t.Fatalf("Split: %v", err)
	}

	// Projecting gamma: it is the antecedent of the split.
	projGamma, err := ProjectHypothesis(ctx, sub, gamma)
	if err != nil {
		t.Fatalf("ProjectHypothesis(gamma): %v", err)
	}
	if projGamma.State != StateSplitInto {
		t.Errorf("gamma state: got %q, want %q", projGamma.State, StateSplitInto)
	}
	// Projecting alpha: it is merged-into (the antecedent of merge),
	// but NOT the antecedent of any split (it's a SUCCESSOR of gamma's
	// split). So alpha's state remains StateMergedInto.
	projAlpha, err := ProjectHypothesis(ctx, sub, alpha)
	if err != nil {
		t.Fatalf("ProjectHypothesis(alpha): %v", err)
	}
	if projAlpha.State != StateMergedInto {
		t.Errorf("alpha state: got %q, want %q (alpha is a SUCCESSOR of the split, not the antecedent)", projAlpha.State, StateMergedInto)
	}
}

func TestProjectUnknownFormation(t *testing.T) {
	sub := newSubstrate(t)
	var bogus [32]byte
	for i := range bogus {
		bogus[i] = byte(0xff)
	}
	_, err := ProjectHypothesis(context.Background(), sub, bogus)
	if !errors.Is(err, ErrFormationNotFound) {
		t.Errorf("expected ErrFormationNotFound; got %v", err)
	}
}

func TestProjectTargetNotFormation(t *testing.T) {
	// Promote a formation, then try to project the PROMOTION hash —
	// should return ErrTargetNotFormation since promotion is not a
	// formation.
	sub := newSubstrate(t)
	formation := formCluster(t, sub, "alpha", "actor-wrong-", 1000)
	ctx := context.Background()
	promRep, err := hypothesis.Promote(ctx, sub, hypothesis.PromoteOptions{
		FormationEventHash: formation,
		PromotedAt:         2000,
		CadenceSeconds:     60,
	}, nil)
	if err != nil {
		t.Fatalf("Promote: %v", err)
	}
	var promHash [32]byte
	if raw, err := hexDecode(promRep.PromotionEventHashHex); err == nil {
		copy(promHash[:], raw)
	} else {
		t.Fatalf("decode hash: %v", err)
	}

	_, err = ProjectHypothesis(ctx, sub, promHash)
	if !errors.Is(err, ErrTargetNotFormation) {
		t.Errorf("expected ErrTargetNotFormation; got %v", err)
	}
}

func TestProjectLifecycleHistoryOrderedByEventTime(t *testing.T) {
	// Full arc: form → promote → demote → re-promote → dissolve.
	// Even though events commit in promote/demote/promote/dissolve
	// order (committed_at ascending), the LifecycleHistory must be
	// ordered by event_time (promoted_at / demoted_at / dissolved_at),
	// not by commit order.
	sub := newSubstrate(t)
	formation := formCluster(t, sub, "alpha", "actor-hist-", 1000)
	ctx := context.Background()

	prom1, _ := hypothesis.Promote(ctx, sub, hypothesis.PromoteOptions{
		FormationEventHash: formation,
		PromotedAt:         2000,
		CadenceSeconds:     60,
		Reason:             "p1",
	}, nil)
	var p1 [32]byte
	if raw, _ := hexDecode(prom1.PromotionEventHashHex); true {
		copy(p1[:], raw)
	}
	if _, err := hypothesis.Demote(ctx, sub, hypothesis.DemoteOptions{
		PromotionEventHash: p1,
		DemotedAt:          3000,
		Reason:             "d1",
	}, nil); err != nil {
		t.Fatalf("Demote: %v", err)
	}
	if _, err := hypothesis.Promote(ctx, sub, hypothesis.PromoteOptions{
		FormationEventHash: formation,
		PromotedAt:         4000,
		CadenceSeconds:     120,
		Reason:             "p2",
	}, nil); err != nil {
		t.Fatalf("re-Promote: %v", err)
	}
	if _, err := hypothesis.Dissolve(ctx, sub, hypothesis.DissolveOptions{
		FormationEventHash: formation,
		DissolvedAt:        5000,
		Reason:             "diss",
	}, nil); err != nil {
		t.Fatalf("Dissolve: %v", err)
	}

	proj, err := ProjectHypothesis(ctx, sub, formation)
	if err != nil {
		t.Fatalf("ProjectHypothesis: %v", err)
	}
	if proj.State != StateDissolved {
		t.Errorf("terminal state: got %q, want %q", proj.State, StateDissolved)
	}
	if len(proj.LifecycleHistory) != 5 {
		t.Fatalf("history length: got %d, want 5", len(proj.LifecycleHistory))
	}
	// Verify ascending by EventTime.
	for i := 1; i < len(proj.LifecycleHistory); i++ {
		if proj.LifecycleHistory[i].EventTime < proj.LifecycleHistory[i-1].EventTime {
			t.Errorf("lifecycle history not sorted ascending at index %d", i)
		}
	}
}

// hexDecode is a local lowercase-hex decoder mirroring the one in
// other test files; avoiding the import cycle (the hypothesis test
// package already provides this helper but it is not exported).
func hexDecode(s string) ([]byte, error) {
	out := make([]byte, len(s)/2)
	for i := 0; i < len(out); i++ {
		var hi, lo byte
		c := s[2*i]
		switch {
		case '0' <= c && c <= '9':
			hi = c - '0'
		case 'a' <= c && c <= 'f':
			hi = c - 'a' + 10
		}
		c = s[2*i+1]
		switch {
		case '0' <= c && c <= '9':
			lo = c - '0'
		case 'a' <= c && c <= 'f':
			lo = c - 'a' + 10
		}
		out[i] = (hi << 4) | lo
	}
	return out, nil
}
