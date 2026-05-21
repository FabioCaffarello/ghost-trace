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

// formCoordinationRingAndPromote populates a substrate with a CR
// formation + one promotion; returns (substrate, promotion hash).
func formCoordinationRingAndPromote(t *testing.T, promotedAt, cadenceSeconds int64) (*substrate.Substrate, [32]byte) {
	t.Helper()
	sub, formation := formCoordinationRingAndCollect(t)
	rep, err := PromoteCoordinationRing(context.Background(), sub, CoordinationRingPromoteOptions{
		FormationEventHash: formation,
		PromotedAt:         promotedAt,
		CadenceSeconds:     cadenceSeconds,
	}, nil)
	if err != nil {
		t.Fatalf("PromoteCoordinationRing: %v", err)
	}
	var promotionHash [32]byte
	raw, err := hexDecode(rep.PromotionEventHashHex)
	if err != nil {
		t.Fatalf("decode promotion hash: %v", err)
	}
	copy(promotionHash[:], raw)
	return sub, promotionHash
}

func walkCoordinationRingDemotions(t *testing.T, sub *substrate.Substrate) []*eventsv1.CoordinationRingDemotion {
	t.Helper()
	ctx := context.Background()
	var out []*eventsv1.CoordinationRingDemotion
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != "ghosttrace.events.v1.CoordinationRingDemotion" {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return err
		}
		ev := &eventsv1.CoordinationRingDemotion{}
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

func TestDemoteCoordinationRingHappyPathCadenceSatisfied(t *testing.T) {
	sub, promotionHash := formCoordinationRingAndPromote(t, 1000*int64(time.Second), 60)
	rep, err := DemoteCoordinationRing(context.Background(), sub, CoordinationRingDemoteOptions{
		PromotionEventHash: promotionHash,
		DemotedAt:          1100 * int64(time.Second),
		Reason:             "operational cycle close",
	}, nil)
	if err != nil {
		t.Fatalf("DemoteCoordinationRing: %v", err)
	}
	if !rep.CadenceSatisfied {
		t.Errorf("CadenceSatisfied: got false, want true (elapsed 100s > cadence 60s)")
	}
	if rep.CadenceElapsedSeconds != 100 {
		t.Errorf("CadenceElapsedSeconds: got %d, want 100", rep.CadenceElapsedSeconds)
	}
	if len(walkCoordinationRingDemotions(t, sub)) != 1 {
		t.Fatalf("expected 1 demotion")
	}
}

func TestDemoteCoordinationRingCadenceUnsatisfied(t *testing.T) {
	sub, promotionHash := formCoordinationRingAndPromote(t, 1000*int64(time.Second), 3600)
	rep, err := DemoteCoordinationRing(context.Background(), sub, CoordinationRingDemoteOptions{
		PromotionEventHash: promotionHash,
		DemotedAt:          1010 * int64(time.Second),
		Reason:             "early demote",
	}, nil)
	if err != nil {
		t.Fatalf("DemoteCoordinationRing: %v", err)
	}
	if rep.CadenceSatisfied {
		t.Errorf("CadenceSatisfied: got true, want false (elapsed 10s < cadence 3600s)")
	}
	if len(walkCoordinationRingDemotions(t, sub)) != 1 {
		t.Errorf("substrate must record the early demotion")
	}
}

func TestDemoteCoordinationRingIdempotent(t *testing.T) {
	sub, promotionHash := formCoordinationRingAndPromote(t, 1000*int64(time.Second), 60)
	opts := CoordinationRingDemoteOptions{
		PromotionEventHash: promotionHash,
		DemotedAt:          1100 * int64(time.Second),
	}
	ctx := context.Background()
	rep1, _ := DemoteCoordinationRing(ctx, sub, opts, nil)
	rep2, _ := DemoteCoordinationRing(ctx, sub, opts, nil)
	if !rep2.AlreadyDemoted {
		t.Errorf("second invocation should report AlreadyDemoted")
	}
	if rep1.DemotionEventHashHex != rep2.DemotionEventHashHex {
		t.Errorf("idempotency violated")
	}
}

func TestDemoteCoordinationRingUnknownTarget(t *testing.T) {
	sub, _ := formCoordinationRingAndPromote(t, 1000*int64(time.Second), 60)
	var bogus [32]byte
	for i := range bogus {
		bogus[i] = byte(0xff - i)
	}
	_, err := DemoteCoordinationRing(context.Background(), sub, CoordinationRingDemoteOptions{
		PromotionEventHash: bogus,
	}, nil)
	if !errors.Is(err, ErrTargetNotFound) {
		t.Errorf("expected ErrTargetNotFound; got %v", err)
	}
}

func TestDemoteCoordinationRingRejectsFormationHash(t *testing.T) {
	sub, formation := formCoordinationRingAndCollect(t)
	_, err := DemoteCoordinationRing(context.Background(), sub, CoordinationRingDemoteOptions{
		PromotionEventHash: formation,
	}, nil)
	if !errors.Is(err, ErrTargetWrongType) {
		t.Errorf("expected ErrTargetWrongType for formation hash; got %v", err)
	}
}

func TestDemoteCoordinationRingRejectsCrossSubtype(t *testing.T) {
	// A BehavioralClusterPromotion hash MUST be rejected.
	bcSub, bcFormation := formAndCollect(t)
	bcPromRep, _ := Promote(context.Background(), bcSub, PromoteOptions{
		FormationEventHash: bcFormation,
		PromotedAt:         1000 * int64(time.Second),
		CadenceSeconds:     60,
	}, nil)
	var bcPromHash [32]byte
	if raw, _ := hexDecode(bcPromRep.PromotionEventHashHex); true {
		copy(bcPromHash[:], raw)
	}
	_, err := DemoteCoordinationRing(context.Background(), bcSub, CoordinationRingDemoteOptions{
		PromotionEventHash: bcPromHash,
	}, nil)
	if !errors.Is(err, ErrTargetWrongType) {
		t.Errorf("expected ErrTargetWrongType for cross-subtype promotion; got %v", err)
	}
}

func TestDemoteCoordinationRingDefaultDemotedAt(t *testing.T) {
	sub, promotionHash := formCoordinationRingAndPromote(t, 1000*int64(time.Second), 60)
	fixedNow := func() time.Time { return time.Unix(0, 9999999999) }
	_, err := DemoteCoordinationRing(context.Background(), sub, CoordinationRingDemoteOptions{
		PromotionEventHash: promotionHash,
	}, fixedNow)
	if err != nil {
		t.Fatalf("DemoteCoordinationRing: %v", err)
	}
	demotions := walkCoordinationRingDemotions(t, sub)
	if len(demotions) != 1 {
		t.Fatalf("expected 1 demotion; got %d", len(demotions))
	}
	if demotions[0].DemotedAt != 9999999999 {
		t.Errorf("demoted_at: got %d, want 9999999999", demotions[0].DemotedAt)
	}
}
