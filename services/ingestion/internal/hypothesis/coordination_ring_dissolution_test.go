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

func walkCoordinationRingDissolutions(t *testing.T, sub *substrate.Substrate) []*eventsv1.CoordinationRingDissolution {
	t.Helper()
	ctx := context.Background()
	var out []*eventsv1.CoordinationRingDissolution
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != "ghosttrace.events.v1.CoordinationRingDissolution" {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return err
		}
		ev := &eventsv1.CoordinationRingDissolution{}
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

func TestDissolveCoordinationRingDirectFromFormation(t *testing.T) {
	sub, formation := formCoordinationRingAndCollect(t)
	rep, err := DissolveCoordinationRing(context.Background(), sub, CoordinationRingDissolveOptions{
		FormationEventHash: formation,
		DissolvedAt:        1716120000000000000,
		Reason:             "coordinated-action pattern was collection-bias artifact",
	}, nil)
	if err != nil {
		t.Fatalf("DissolveCoordinationRing: %v", err)
	}
	if rep.AlreadyDissolved {
		t.Errorf("first invocation should not report AlreadyDissolved")
	}
	if len(walkCoordinationRingDissolutions(t, sub)) != 1 {
		t.Fatalf("expected 1 dissolution")
	}
}

func TestDissolveCoordinationRingIdempotent(t *testing.T) {
	sub, formation := formCoordinationRingAndCollect(t)
	opts := CoordinationRingDissolveOptions{
		FormationEventHash: formation,
		DissolvedAt:        1200 * int64(time.Second),
		Reason:             "terminal",
	}
	ctx := context.Background()
	rep1, _ := DissolveCoordinationRing(ctx, sub, opts, nil)
	rep2, _ := DissolveCoordinationRing(ctx, sub, opts, nil)
	if !rep2.AlreadyDissolved {
		t.Errorf("second invocation should report AlreadyDissolved")
	}
	if rep1.DissolutionEventHashHex != rep2.DissolutionEventHashHex {
		t.Errorf("idempotency violated")
	}
}

func TestDissolveCoordinationRingUnknownTarget(t *testing.T) {
	sub, _ := formCoordinationRingAndCollect(t)
	var bogus [32]byte
	for i := range bogus {
		bogus[i] = byte(0xff - i)
	}
	_, err := DissolveCoordinationRing(context.Background(), sub, CoordinationRingDissolveOptions{
		FormationEventHash: bogus,
	}, nil)
	if !errors.Is(err, ErrTargetNotFound) {
		t.Errorf("expected ErrTargetNotFound; got %v", err)
	}
}

func TestDissolveCoordinationRingRejectsPromotionHash(t *testing.T) {
	sub, promotionHash := formCoordinationRingAndPromote(t, 1000*int64(time.Second), 60)
	_, err := DissolveCoordinationRing(context.Background(), sub, CoordinationRingDissolveOptions{
		FormationEventHash: promotionHash,
	}, nil)
	if !errors.Is(err, ErrTargetWrongType) {
		t.Errorf("expected ErrTargetWrongType for promotion hash; got %v", err)
	}
}

func TestDissolveCoordinationRingRejectsCrossSubtypeBC(t *testing.T) {
	bcSub, bcFormation := formAndCollect(t)
	_, err := DissolveCoordinationRing(context.Background(), bcSub, CoordinationRingDissolveOptions{
		FormationEventHash: bcFormation,
	}, nil)
	if !errors.Is(err, ErrTargetWrongType) {
		t.Errorf("expected ErrTargetWrongType for BC formation; got %v", err)
	}
}

func TestDissolveCoordinationRingRejectsCrossSubtypeCH(t *testing.T) {
	chSub, chFormation := formCampaignAndCollect(t)
	_, err := DissolveCoordinationRing(context.Background(), chSub, CoordinationRingDissolveOptions{
		FormationEventHash: chFormation,
	}, nil)
	if !errors.Is(err, ErrTargetWrongType) {
		t.Errorf("expected ErrTargetWrongType for CH formation; got %v", err)
	}
}

func TestDissolveCoordinationRingDefaultDissolvedAt(t *testing.T) {
	sub, formation := formCoordinationRingAndCollect(t)
	fixedNow := func() time.Time { return time.Unix(0, 9999999999) }
	_, err := DissolveCoordinationRing(context.Background(), sub, CoordinationRingDissolveOptions{
		FormationEventHash: formation,
	}, fixedNow)
	if err != nil {
		t.Fatalf("DissolveCoordinationRing: %v", err)
	}
	dissolutions := walkCoordinationRingDissolutions(t, sub)
	if len(dissolutions) != 1 {
		t.Fatalf("expected 1 dissolution; got %d", len(dissolutions))
	}
	if dissolutions[0].DissolvedAt != 9999999999 {
		t.Errorf("dissolved_at: got %d, want 9999999999", dissolutions[0].DissolvedAt)
	}
}
