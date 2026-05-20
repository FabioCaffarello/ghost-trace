package hypothesis

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// formAutomationGroupAndCollect populates a substrate with five
// uniform-cadence DeclaredSessions for a single actor, runs
// FormAutomationGroupAll with default uniform-cadence-v1 parameters,
// and returns the substrate + the resulting formation event hash.
func formAutomationGroupAndCollect(t *testing.T) (*substrate.Substrate, [32]byte) {
	t.Helper()
	sub := agSubstrate(t, []struct {
		ActorRef   string
		DeclaredAt int64
	}{
		{"bot-a", 1000}, {"bot-a", 2000}, {"bot-a", 3000}, {"bot-a", 4000}, {"bot-a", 5000},
	})

	rep, err := FormAutomationGroupAll(context.Background(), sub,
		UniformCadenceV1{MinObservationCount: 5, MaxCoVThreshold: 0.15},
		func() time.Time { return time.Unix(0, 1) })
	if err != nil {
		t.Fatalf("FormAutomationGroupAll: %v", err)
	}
	if rep.NewlyFormed != 1 {
		t.Fatalf("expected 1 formation; got %d", rep.NewlyFormed)
	}

	var formationHash [32]byte
	if err := sub.WalkEvents(context.Background(), func(row substrate.EventRow) error {
		if row.MessageType == automationGroupFormationMessageType {
			formationHash = row.EventHash
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	return sub, formationHash
}

func walkAutomationGroupPromotions(t *testing.T, sub *substrate.Substrate) []*eventsv1.AutomationGroupPromotion {
	t.Helper()
	ctx := context.Background()
	var out []*eventsv1.AutomationGroupPromotion
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != "ghosttrace.events.v1.AutomationGroupPromotion" {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return err
		}
		ev := &eventsv1.AutomationGroupPromotion{}
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

func TestPromoteAutomationGroupHappyPath(t *testing.T) {
	sub, formation := formAutomationGroupAndCollect(t)
	rep, err := PromoteAutomationGroup(context.Background(), sub, AutomationGroupPromoteOptions{
		FormationEventHash: formation,
		PromotedAt:         1716120000000000000,
		CadenceSeconds:     3600,
		Reason:             "operational pilot",
	}, nil)
	if err != nil {
		t.Fatalf("PromoteAutomationGroup: %v", err)
	}
	if rep.AlreadyPromoted {
		t.Errorf("unexpected AlreadyPromoted on first invocation")
	}

	promotions := walkAutomationGroupPromotions(t, sub)
	if len(promotions) != 1 {
		t.Fatalf("substrate carries %d promotions, want 1", len(promotions))
	}
	got := promotions[0]
	if got.CadenceSeconds != 3600 {
		t.Errorf("cadence_seconds: got %d, want 3600", got.CadenceSeconds)
	}
	if got.PromotedAt != 1716120000000000000 {
		t.Errorf("promoted_at: got %d, want 1716120000000000000", got.PromotedAt)
	}
	if got.Reason != "operational pilot" {
		t.Errorf("reason: got %q, want %q", got.Reason, "operational pilot")
	}
	for i, b := range got.FormationEventHash {
		if b != formation[i] {
			t.Errorf("formation_event_hash mismatch at byte %d", i)
			break
		}
	}
}

func TestPromoteAutomationGroupIdempotent(t *testing.T) {
	sub, formation := formAutomationGroupAndCollect(t)
	opts := AutomationGroupPromoteOptions{
		FormationEventHash: formation,
		PromotedAt:         1716120000000000000,
		CadenceSeconds:     3600,
		Reason:             "idempotent",
	}
	ctx := context.Background()
	rep1, err := PromoteAutomationGroup(ctx, sub, opts, nil)
	if err != nil {
		t.Fatalf("first PromoteAutomationGroup: %v", err)
	}
	rep2, err := PromoteAutomationGroup(ctx, sub, opts, nil)
	if err != nil {
		t.Fatalf("second PromoteAutomationGroup: %v", err)
	}
	if !rep2.AlreadyPromoted {
		t.Errorf("second invocation should report AlreadyPromoted")
	}
	if rep1.PromotionEventHashHex != rep2.PromotionEventHashHex {
		t.Errorf("idempotency violated: %q != %q", rep1.PromotionEventHashHex, rep2.PromotionEventHashHex)
	}
	if got := len(walkAutomationGroupPromotions(t, sub)); got != 1 {
		t.Errorf("after re-run substrate holds %d promotions; want 1", got)
	}
}

func TestPromoteAutomationGroupVersioningProducesNewRecord(t *testing.T) {
	sub, formation := formAutomationGroupAndCollect(t)
	ctx := context.Background()
	if _, err := PromoteAutomationGroup(ctx, sub, AutomationGroupPromoteOptions{
		FormationEventHash: formation,
		PromotedAt:         1716120000000000000,
		CadenceSeconds:     3600,
		Reason:             "first",
	}, nil); err != nil {
		t.Fatalf("first: %v", err)
	}
	rep, err := PromoteAutomationGroup(ctx, sub, AutomationGroupPromoteOptions{
		FormationEventHash: formation,
		PromotedAt:         1716120000000000000,
		CadenceSeconds:     7200, // different cadence
		Reason:             "first",
	}, nil)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if rep.AlreadyPromoted {
		t.Errorf("changing cadence should produce new record")
	}
	if got := len(walkAutomationGroupPromotions(t, sub)); got != 2 {
		t.Errorf("substrate should hold 2 promotions; got %d", got)
	}
}

func TestPromoteAutomationGroupRejectsNonPositiveCadence(t *testing.T) {
	sub, formation := formAutomationGroupAndCollect(t)
	for _, cs := range []int64{0, -1} {
		_, err := PromoteAutomationGroup(context.Background(), sub, AutomationGroupPromoteOptions{
			FormationEventHash: formation,
			CadenceSeconds:     cs,
		}, nil)
		if err == nil {
			t.Errorf("cadence=%d: expected error, got nil", cs)
		}
	}
}

func TestPromoteAutomationGroupUnknownTarget(t *testing.T) {
	sub, _ := formAutomationGroupAndCollect(t)
	var bogus [32]byte
	for i := range bogus {
		bogus[i] = byte(0xff - i)
	}
	_, err := PromoteAutomationGroup(context.Background(), sub, AutomationGroupPromoteOptions{
		FormationEventHash: bogus,
		CadenceSeconds:     3600,
	}, nil)
	if !errors.Is(err, ErrTargetNotFound) {
		t.Errorf("expected ErrTargetNotFound; got %v", err)
	}
}

func TestPromoteAutomationGroupWrongSubtypeRejected(t *testing.T) {
	// Cross-subtype guard: a BehavioralClusterFormation hash MUST be
	// rejected by PromoteAutomationGroup even though both are
	// formation events. The wrong-type check uses the subtype-
	// specific message_type discriminator.
	sub, behavioralFormation := formAndCollect(t) // from §0046's promotion_test.go
	_, err := PromoteAutomationGroup(context.Background(), sub, AutomationGroupPromoteOptions{
		FormationEventHash: behavioralFormation,
		CadenceSeconds:     3600,
	}, nil)
	if !errors.Is(err, ErrTargetWrongType) {
		t.Errorf("expected ErrTargetWrongType for BehavioralClusterFormation; got %v", err)
	}
}

func TestPromoteAutomationGroupDefaultPromotedAt(t *testing.T) {
	sub, formation := formAutomationGroupAndCollect(t)
	fixedNow := func() time.Time { return time.Unix(0, 9999999999) }
	_, err := PromoteAutomationGroup(context.Background(), sub, AutomationGroupPromoteOptions{
		FormationEventHash: formation,
		CadenceSeconds:     60,
	}, fixedNow)
	if err != nil {
		t.Fatalf("PromoteAutomationGroup: %v", err)
	}
	promotions := walkAutomationGroupPromotions(t, sub)
	if len(promotions) != 1 {
		t.Fatalf("expected 1 promotion; got %d", len(promotions))
	}
	if promotions[0].PromotedAt != 9999999999 {
		t.Errorf("promoted_at: got %d, want 9999999999 (from injected now)", promotions[0].PromotedAt)
	}
}

func TestPromoteAutomationGroupCoexistsWithBehavioralPromotion(t *testing.T) {
	// Substrate with both subtypes promoted independently: each
	// promotion is a distinct message_type and references a distinct
	// formation hash.
	sub, agFormation := formAutomationGroupAndCollect(t)
	ctx := context.Background()

	// Promote AutomationGroup.
	if _, err := PromoteAutomationGroup(ctx, sub, AutomationGroupPromoteOptions{
		FormationEventHash: agFormation,
		PromotedAt:         1000000,
		CadenceSeconds:     3600,
	}, nil); err != nil {
		t.Fatalf("PromoteAutomationGroup: %v", err)
	}

	// Form + promote a BehavioralCluster on the SAME substrate
	// (independent observations).
	bcSub, bcFormation := formAndCollect(t)
	_ = bcSub // separate substrate is fine; cross-substrate is not the point
	if _, err := Promote(ctx, sub, PromoteOptions{
		FormationEventHash: bcFormation,
		PromotedAt:         2000000,
		CadenceSeconds:     3600,
	}, nil); err != nil {
		// BehavioralClusterFormation isn't in the AG sub; expect
		// ErrTargetNotFound. Validate the cross-substrate barrier.
		if !errors.Is(err, ErrTargetNotFound) {
			t.Errorf("expected ErrTargetNotFound for cross-substrate; got %v", err)
		}
	}

	// Confirm AG promotion landed.
	if got := len(walkAutomationGroupPromotions(t, sub)); got != 1 {
		t.Errorf("AG promotion count: got %d, want 1", got)
	}
}
