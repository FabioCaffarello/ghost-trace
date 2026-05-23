package hypothesis

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// formAndPromote sets up the full prerequisite chain: populates a
// substrate with two shared-descriptor DeclaredSessions, runs FormAll
// to land a BehavioralClusterFormation, then Promote to land a
// BehavioralClusterPromotion under a specified cadence. Returns the
// substrate + the promotion event hash for subsequent Demote calls.
func formAndPromote(t *testing.T, promotedAt int64, cadenceSeconds int64) (*substrate.Substrate, [32]byte) {
	t.Helper()
	sub, formationHash := formAndCollect(t)
	rep, err := Promote(context.Background(), sub, PromoteOptions{
		FormationEventHash: formationHash,
		PromotedAt:         promotedAt,
		CadenceSeconds:     cadenceSeconds,
		Reason:             "test promotion",
	}, nil)
	if err != nil {
		t.Fatalf("Promote: %v", err)
	}
	// Recover the promotion hash from substrate walk (the report has
	// the hex form; we need the bytes for downstream Demote).
	rawHex := rep.PromotionEventHashHex
	var promotionHash [32]byte
	if err := sub.WalkEvents(context.Background(), func(row substrate.EventRow) error {
		if row.MessageType == "ghosttrace.events.v1.BehavioralClusterPromotion" {
			promotionHash = row.EventHash
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	if promotionHash == [32]byte{} {
		t.Fatalf("no BehavioralClusterPromotion found in substrate (expected hex %s)", rawHex)
	}
	return sub, promotionHash
}

func walkDemotions(t *testing.T, sub *substrate.Substrate) []*eventsv1.BehavioralClusterDemotion {
	t.Helper()
	ctx := context.Background()
	var out []*eventsv1.BehavioralClusterDemotion
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != "ghosttrace.events.v1.BehavioralClusterDemotion" {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return err
		}
		ev := &eventsv1.BehavioralClusterDemotion{}
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

func TestDemoteFullLoop(t *testing.T) {
	// Full lifecycle chain: form → promote → demote. Substrate ends
	// with all three lifecycle events present.
	sub, promotionHash := formAndPromote(t, 1716120000000000000, 3600)
	rep, err := Demote(context.Background(), sub, DemoteOptions{
		PromotionEventHash: promotionHash,
		DemotedAt:          1716120004000000000, // 4 sec after promote → cadence NOT satisfied
		Reason:             "early demotion",
	}, nil)
	if err != nil {
		t.Fatalf("Demote: %v", err)
	}
	if rep.AlreadyDemoted {
		t.Errorf("unexpected AlreadyDemoted on first invocation")
	}
	if rep.CadenceSatisfied {
		t.Errorf("cadence_satisfied: got true, want false (demoted 4s after promote, cadence=3600s)")
	}
	if rep.CadenceElapsedSeconds != 4 {
		t.Errorf("cadence_elapsed_seconds: got %d, want 4", rep.CadenceElapsedSeconds)
	}

	demotions := walkDemotions(t, sub)
	if len(demotions) != 1 {
		t.Fatalf("substrate carries %d demotions, want 1", len(demotions))
	}
	got := demotions[0]
	if got.Reason != "early demotion" {
		t.Errorf("reason: got %q, want %q", got.Reason, "early demotion")
	}
	// promotion_event_hash must reference the promotion event's hash.
	for i, b := range got.PromotionEventHash {
		if b != promotionHash[i] {
			t.Errorf("promotion_event_hash mismatch at byte %d: got %x, want %x", i, b, promotionHash[i])
			break
		}
	}
}

func TestDemoteCadenceSatisfied(t *testing.T) {
	// Demoted AFTER cadence_seconds has elapsed → cadence_satisfied=true.
	sub, promotionHash := formAndPromote(t, 1000*int64(time.Second), 60)
	demotedAt := (1000 + 120) * int64(time.Second) // 120s after promote, cadence=60s
	rep, err := Demote(context.Background(), sub, DemoteOptions{
		PromotionEventHash: promotionHash,
		DemotedAt:          demotedAt,
	}, nil)
	if err != nil {
		t.Fatalf("Demote: %v", err)
	}
	if !rep.CadenceSatisfied {
		t.Errorf("cadence_satisfied: got false, want true (120s elapsed, cadence=60s)")
	}
	if rep.CadenceElapsedSeconds != 120 {
		t.Errorf("cadence_elapsed_seconds: got %d, want 120", rep.CadenceElapsedSeconds)
	}
}

func TestDemoteCadenceBoundaryExact(t *testing.T) {
	// Demoted EXACTLY at the cadence boundary → cadence_satisfied=true.
	sub, promotionHash := formAndPromote(t, 1000*int64(time.Second), 60)
	demotedAt := (1000 + 60) * int64(time.Second)
	rep, err := Demote(context.Background(), sub, DemoteOptions{
		PromotionEventHash: promotionHash,
		DemotedAt:          demotedAt,
	}, nil)
	if err != nil {
		t.Fatalf("Demote: %v", err)
	}
	if !rep.CadenceSatisfied {
		t.Errorf("cadence_satisfied at exact boundary: got false, want true")
	}
}

func TestDemoteIdempotent(t *testing.T) {
	sub, promotionHash := formAndPromote(t, 1000*int64(time.Second), 60)
	opts := DemoteOptions{
		PromotionEventHash: promotionHash,
		DemotedAt:          1200 * int64(time.Second),
		Reason:             "scheduled rollover",
	}

	ctx := context.Background()
	rep1, err := Demote(ctx, sub, opts, nil)
	if err != nil {
		t.Fatalf("first Demote: %v", err)
	}
	if rep1.AlreadyDemoted {
		t.Errorf("first invocation should not report AlreadyDemoted")
	}
	rep2, err := Demote(ctx, sub, opts, nil)
	if err != nil {
		t.Fatalf("second Demote: %v", err)
	}
	if !rep2.AlreadyDemoted {
		t.Errorf("second invocation should report AlreadyDemoted (content-hash collision)")
	}
	if rep1.DemotionEventHashHex != rep2.DemotionEventHashHex {
		t.Errorf("idempotency violated: hashes differ %q != %q", rep1.DemotionEventHashHex, rep2.DemotionEventHashHex)
	}
	if got := len(walkDemotions(t, sub)); got != 1 {
		t.Errorf("substrate holds %d demotions after re-run; want 1", got)
	}
}

func TestDemoteVersioningProducesNewRecord(t *testing.T) {
	// Re-demoting with a DIFFERENT demoted_at OR different reason
	// produces a NEW demotion event alongside the prior (§2.5
	// immutability — operation history records every parameter).
	sub, promotionHash := formAndPromote(t, 1000*int64(time.Second), 60)
	ctx := context.Background()
	if _, err := Demote(ctx, sub, DemoteOptions{
		PromotionEventHash: promotionHash,
		DemotedAt:          1100 * int64(time.Second),
		Reason:             "first",
	}, nil); err != nil {
		t.Fatalf("first Demote: %v", err)
	}
	rep, err := Demote(ctx, sub, DemoteOptions{
		PromotionEventHash: promotionHash,
		DemotedAt:          1100 * int64(time.Second),
		Reason:             "second",
	}, nil)
	if err != nil {
		t.Fatalf("second Demote: %v", err)
	}
	if rep.AlreadyDemoted {
		t.Errorf("changing reason should produce new record; got AlreadyDemoted")
	}
	if got := len(walkDemotions(t, sub)); got != 2 {
		t.Errorf("after two demotions with distinct reasons, substrate holds %d demotions; want 2", got)
	}
}

func TestDemoteUnknownTarget(t *testing.T) {
	sub, _ := formAndPromote(t, 1000*int64(time.Second), 60)
	var bogus [32]byte
	for i := range bogus {
		bogus[i] = byte(0xff - i)
	}
	_, err := Demote(context.Background(), sub, DemoteOptions{
		PromotionEventHash: bogus,
		DemotedAt:          1100 * int64(time.Second),
	}, nil)
	if !errors.Is(err, ErrTargetNotFound) {
		t.Errorf("expected ErrTargetNotFound; got %v", err)
	}
}

func TestDemoteWrongTypeTarget(t *testing.T) {
	// Pointing the promotion hash at a FORMATION row (not a promotion)
	// should return ErrTargetWrongType — preserves §2.5-lifecycle-integrity
	// (demotion references only promotion events).
	sub, formationHash := formAndCollect(t)
	_, err := Demote(context.Background(), sub, DemoteOptions{
		PromotionEventHash: formationHash, // formation hash; wrong type
		DemotedAt:          1100 * int64(time.Second),
	}, nil)
	if !errors.Is(err, ErrTargetWrongType) {
		t.Errorf("expected ErrTargetWrongType; got %v", err)
	}
}

func TestDemoteDefaultDemotedAt(t *testing.T) {
	sub, promotionHash := formAndPromote(t, 1000*int64(time.Second), 60)
	fixedNow := func() time.Time { return time.Unix(0, 9999999999) }
	rep, err := Demote(context.Background(), sub, DemoteOptions{
		PromotionEventHash: promotionHash,
	}, fixedNow)
	if err != nil {
		t.Fatalf("Demote: %v", err)
	}
	demotions := walkDemotions(t, sub)
	if len(demotions) != 1 {
		t.Fatalf("expected 1 demotion; got %d", len(demotions))
	}
	if demotions[0].DemotedAt != 9999999999 {
		t.Errorf("demoted_at: got %d, want 9999999999 (from injected now)", demotions[0].DemotedAt)
	}
	// 9999999999 ns = ~10 seconds; promoted_at = 1000s; elapsed ≈ -990s.
	// Cadence_satisfied should still be reported correctly (false here).
	if rep.CadenceSatisfied {
		t.Errorf("cadence_satisfied: got true; want false (negative elapsed)")
	}
}

func TestDemoteFullChainInSubstrate(t *testing.T) {
	// After form → promote → demote, the substrate carries:
	// 2 DeclaredSession + 2 IngestionEvent + 1 Formation + 1 Promotion + 1 Demotion = 7 rows.
	sub, promotionHash := formAndPromote(t, 1000*int64(time.Second), 60)
	if _, err := Demote(context.Background(), sub, DemoteOptions{
		PromotionEventHash: promotionHash,
		DemotedAt:          1100 * int64(time.Second),
		Reason:             "lifecycle test",
	}, nil); err != nil {
		t.Fatalf("Demote: %v", err)
	}

	n, err := sub.Count(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if n != 7 {
		t.Errorf("substrate Count: got %d, want 7 (2 DeclaredSession + 2 IngestionEvent + 1 Formation + 1 Promotion + 1 Demotion)", n)
	}

	typeCounts := map[string]int{}
	if err := sub.WalkEvents(context.Background(), func(row substrate.EventRow) error {
		typeCounts[row.MessageType]++
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	wantCounts := map[string]int{
		"ghosttrace.events.v1.DeclaredSession":             2,
		"ghosttrace.events.v1.IngestionEvent":              2,
		"ghosttrace.events.v1.BehavioralClusterFormation":  1,
		"ghosttrace.events.v1.BehavioralClusterPromotion":  1,
		"ghosttrace.events.v1.BehavioralClusterDemotion":   1,
	}
	for mt, want := range wantCounts {
		if got := typeCounts[mt]; got != want {
			t.Errorf("%s count: got %d, want %d", mt, got, want)
		}
	}
}

// TestDemoteLayerBLegacyPromotion verifies that demotions targeting
// pre-§0138 promotions (which lack populated LayerBParameters) surface
// DemoteReport.LayerB.Evaluated=false. This is the documented
// fallback per evaluateLayerB + LayerBReport: when the source
// promotion's layer_b_parameters field is unset, Layer B's predicate
// is not runnable and the report's other LayerB fields are zero.
//
// The current Promote() implementation does not populate
// layer_b_parameters; all promotions produced via Promote() therefore
// fall into the legacy path until promote-hypothesis is extended per
// §0141 sub-decision F3 follow-on PR.
func TestDemoteLayerBLegacyPromotion(t *testing.T) {
	sub, promotionHash := formAndPromote(t, 1716120000000000000, 3600)
	rep, err := Demote(context.Background(), sub, DemoteOptions{
		PromotionEventHash: promotionHash,
		DemotedAt:          1716120004000000000,
		Reason:             "test",
	}, nil)
	if err != nil {
		t.Fatalf("Demote: %v", err)
	}
	if rep.LayerB.Evaluated {
		t.Errorf("LayerB.Evaluated: got true, want false (Promote() does not populate layer_b_parameters; legacy path)")
	}
	if rep.LayerB.Fired {
		t.Errorf("LayerB.Fired: got true, want false on Evaluated=false")
	}
	if rep.LayerB.FreshnessFired || rep.LayerB.SaturationFired {
		t.Errorf("LayerB.{Freshness,Saturation}Fired: got true, want false on Evaluated=false")
	}
	// Demote itself succeeded — Layer B advisory per §0141 E1 does not block.
	if rep.DemotionEventHashHex == "" {
		t.Error("DemotionEventHashHex empty; demote should record regardless of LayerB state per §0141 E1")
	}
}

// TestDemoteLayerBEvaluatedPath verifies the Evaluated=true end-to-end
// path: a promotion produced via Promote() with LayerBParameters set
// produces a substrate state where the subsequent Demote can run
// Layer B's predicate. This exercises the full layerb.Evaluate path
// through the hypothesis package boundary.
//
// The window contains a single formation event (the substrate carries
// formation + promotion only at demote time; the formation is itself
// in the window and is its own closure root in the §0136 β-graph
// shape). Because the formation does not carry itself in
// closure_hashes (a hypothesis's own formation event is not in its
// own influence closure), the FilterMatchCount is 0 — yielding
// FreshnessUndefined=true. Saturation_C numerator is also 0 → ratio
// 0/1000 = 0, which is NOT > K_C=0.5; saturation does not fire either.
// Fired=false; demote records the demotion regardless per E1.
//
// The test asserts the structural invariants: Evaluated=true; both
// Fired flags false; FreshnessUndefined=true; demote completes.
func TestDemoteLayerBEvaluatedPath(t *testing.T) {
	sub, formationHash := formAndCollect(t)
	ctx := context.Background()
	params := &commonv1.LayerBParameters{
		TB:                    &commonv1.EvidentialIndependence{Numerator: 1, Denominator: 2},
		KC:                    &commonv1.EvidentialIndependence{Numerator: 1, Denominator: 2},
		NWindow:               1000,
		NADurationNanoseconds: 86400000000000,
	}
	if _, err := Promote(ctx, sub, PromoteOptions{
		FormationEventHash: formationHash,
		PromotedAt:         1716120000000000000,
		CadenceSeconds:     3600,
		Reason:             "test promotion with LayerBParameters",
		LayerBParameters:   params,
	}, nil); err != nil {
		t.Fatalf("Promote: %v", err)
	}
	var promotionHash [32]byte
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType == "ghosttrace.events.v1.BehavioralClusterPromotion" {
			promotionHash = row.EventHash
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}

	rep, err := Demote(ctx, sub, DemoteOptions{
		PromotionEventHash: promotionHash,
		DemotedAt:          1716120004000000000,
		Reason:             "test",
	}, nil)
	if err != nil {
		t.Fatalf("Demote: %v", err)
	}
	if !rep.LayerB.Evaluated {
		t.Errorf("LayerB.Evaluated: got false, want true (promotion carries LayerBParameters)")
	}
	if rep.LayerB.Fired {
		t.Errorf("LayerB.Fired: got true, want false (FilterMatchCount=0 → freshness undefined; saturation=0/1000=0 not > 0.5)")
	}
	if !rep.LayerB.FreshnessUndefined {
		t.Errorf("LayerB.FreshnessUndefined: got false, want true (no events in window match the formation hash filter)")
	}
	if rep.LayerB.FilterMatchCount != 0 {
		t.Errorf("LayerB.FilterMatchCount: got %d, want 0 (formation is not its own closure root)", rep.LayerB.FilterMatchCount)
	}
	if rep.DemotionEventHashHex == "" {
		t.Error("DemotionEventHashHex empty; demote should record regardless of LayerB state per §0141 E1")
	}
}
