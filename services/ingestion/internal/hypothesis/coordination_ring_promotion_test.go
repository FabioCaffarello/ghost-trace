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

// formCoordinationRingAndCollect populates a substrate with three
// actors co-occurring three times within the default window, runs
// FormCoordinationRingAll under co-occurrence-window-v1, and returns
// (substrate, formation hash).
func formCoordinationRingAndCollect(t *testing.T) (*substrate.Substrate, [32]byte) {
	t.Helper()
	gap := int64(60 * 1e9)
	roundSpacing := int64(2e12)
	sub := crSubstrate(t, []struct {
		ActorRef   string
		Descriptor string
		DeclaredAt int64
	}{
		{"actor-a", "ring-test", 1000}, {"actor-b", "ring-test", 1000 + gap}, {"actor-c", "ring-test", 1000 + 2*gap},
		{"actor-a", "ring-test", 1000 + roundSpacing}, {"actor-b", "ring-test", 1000 + roundSpacing + gap}, {"actor-c", "ring-test", 1000 + roundSpacing + 2*gap},
		{"actor-a", "ring-test", 1000 + 2*roundSpacing}, {"actor-b", "ring-test", 1000 + 2*roundSpacing + gap}, {"actor-c", "ring-test", 1000 + 2*roundSpacing + 2*gap},
	})

	if _, err := FormCoordinationRingAll(context.Background(), sub,
		CoOccurrenceWindowV1{MinEdgeSupport: 3, MaxWindowSeconds: 600}, nil); err != nil {
		t.Fatalf("FormCoordinationRingAll: %v", err)
	}
	var formationHash [32]byte
	if err := sub.WalkEvents(context.Background(), func(row substrate.EventRow) error {
		if row.MessageType == coordinationRingFormationMessageType {
			formationHash = row.EventHash
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	if formationHash == ([32]byte{}) {
		t.Fatal("no formation found")
	}
	return sub, formationHash
}

func walkCoordinationRingPromotions(t *testing.T, sub *substrate.Substrate) []*eventsv1.CoordinationRingPromotion {
	t.Helper()
	ctx := context.Background()
	var out []*eventsv1.CoordinationRingPromotion
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != "ghosttrace.events.v1.CoordinationRingPromotion" {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return err
		}
		ev := &eventsv1.CoordinationRingPromotion{}
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

func TestPromoteCoordinationRingHappyPath(t *testing.T) {
	sub, formation := formCoordinationRingAndCollect(t)
	rep, err := PromoteCoordinationRing(context.Background(), sub, CoordinationRingPromoteOptions{
		FormationEventHash: formation,
		PromotedAt:         1716120000000000000,
		CadenceSeconds:     3600,
		Reason:             "operational pilot",
	}, nil)
	if err != nil {
		t.Fatalf("PromoteCoordinationRing: %v", err)
	}
	if rep.AlreadyPromoted {
		t.Errorf("unexpected AlreadyPromoted on first invocation")
	}
	promotions := walkCoordinationRingPromotions(t, sub)
	if len(promotions) != 1 {
		t.Fatalf("substrate carries %d promotions, want 1", len(promotions))
	}
	got := promotions[0]
	if got.CadenceSeconds != 3600 {
		t.Errorf("cadence_seconds: got %d, want 3600", got.CadenceSeconds)
	}
	if got.PromotedAt != 1716120000000000000 {
		t.Errorf("promoted_at: got %d", got.PromotedAt)
	}
}

func TestPromoteCoordinationRingIdempotent(t *testing.T) {
	sub, formation := formCoordinationRingAndCollect(t)
	opts := CoordinationRingPromoteOptions{
		FormationEventHash: formation,
		PromotedAt:         1716120000000000000,
		CadenceSeconds:     3600,
	}
	ctx := context.Background()
	rep1, _ := PromoteCoordinationRing(ctx, sub, opts, nil)
	rep2, _ := PromoteCoordinationRing(ctx, sub, opts, nil)
	if !rep2.AlreadyPromoted {
		t.Errorf("second invocation should report AlreadyPromoted")
	}
	if rep1.PromotionEventHashHex != rep2.PromotionEventHashHex {
		t.Errorf("idempotency violated")
	}
}

func TestPromoteCoordinationRingVersioningProducesNewRecord(t *testing.T) {
	sub, formation := formCoordinationRingAndCollect(t)
	ctx := context.Background()
	if _, err := PromoteCoordinationRing(ctx, sub, CoordinationRingPromoteOptions{
		FormationEventHash: formation,
		PromotedAt:         1716120000000000000,
		CadenceSeconds:     3600,
	}, nil); err != nil {
		t.Fatalf("first: %v", err)
	}
	rep, err := PromoteCoordinationRing(ctx, sub, CoordinationRingPromoteOptions{
		FormationEventHash: formation,
		PromotedAt:         1716120000000000000,
		CadenceSeconds:     7200,
	}, nil)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if rep.AlreadyPromoted {
		t.Errorf("changing cadence should produce new record")
	}
	if got := len(walkCoordinationRingPromotions(t, sub)); got != 2 {
		t.Errorf("substrate holds %d; want 2", got)
	}
}

func TestPromoteCoordinationRingRejectsNonPositiveCadence(t *testing.T) {
	sub, formation := formCoordinationRingAndCollect(t)
	for _, cs := range []int64{0, -1} {
		_, err := PromoteCoordinationRing(context.Background(), sub, CoordinationRingPromoteOptions{
			FormationEventHash: formation,
			CadenceSeconds:     cs,
		}, nil)
		if err == nil {
			t.Errorf("cadence=%d: expected error", cs)
		}
	}
}

func TestPromoteCoordinationRingUnknownTarget(t *testing.T) {
	sub, _ := formCoordinationRingAndCollect(t)
	var bogus [32]byte
	for i := range bogus {
		bogus[i] = byte(0xff - i)
	}
	_, err := PromoteCoordinationRing(context.Background(), sub, CoordinationRingPromoteOptions{
		FormationEventHash: bogus,
		CadenceSeconds:     3600,
	}, nil)
	if !errors.Is(err, ErrTargetNotFound) {
		t.Errorf("expected ErrTargetNotFound; got %v", err)
	}
}

func TestPromoteCoordinationRingRejectsCrossSubtypeBC(t *testing.T) {
	// A BehavioralClusterFormation hash MUST be rejected by
	// PromoteCoordinationRing. Validates subtype-specific
	// message_type discrimination.
	bcSub, bcFormation := formAndCollect(t)
	_, err := PromoteCoordinationRing(context.Background(), bcSub, CoordinationRingPromoteOptions{
		FormationEventHash: bcFormation,
		CadenceSeconds:     3600,
	}, nil)
	if !errors.Is(err, ErrTargetWrongType) {
		t.Errorf("expected ErrTargetWrongType for BC formation; got %v", err)
	}
}

func TestPromoteCoordinationRingRejectsCrossSubtypeCH(t *testing.T) {
	// A CampaignHypothesisFormation hash MUST be rejected.
	chSub, chFormation := formCampaignAndCollect(t)
	_, err := PromoteCoordinationRing(context.Background(), chSub, CoordinationRingPromoteOptions{
		FormationEventHash: chFormation,
		CadenceSeconds:     3600,
	}, nil)
	if !errors.Is(err, ErrTargetWrongType) {
		t.Errorf("expected ErrTargetWrongType for CH formation; got %v", err)
	}
}

func TestPromoteCoordinationRingDefaultPromotedAt(t *testing.T) {
	sub, formation := formCoordinationRingAndCollect(t)
	fixedNow := func() time.Time { return time.Unix(0, 9999999999) }
	_, err := PromoteCoordinationRing(context.Background(), sub, CoordinationRingPromoteOptions{
		FormationEventHash: formation,
		CadenceSeconds:     60,
	}, fixedNow)
	if err != nil {
		t.Fatalf("PromoteCoordinationRing: %v", err)
	}
	promotions := walkCoordinationRingPromotions(t, sub)
	if len(promotions) != 1 {
		t.Fatalf("expected 1 promotion; got %d", len(promotions))
	}
	if promotions[0].PromotedAt != 9999999999 {
		t.Errorf("promoted_at: got %d, want 9999999999", promotions[0].PromotedAt)
	}
}
