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

func walkCampaignDissolutions(t *testing.T, sub *substrate.Substrate) []*eventsv1.CampaignHypothesisDissolution {
	t.Helper()
	ctx := context.Background()
	var out []*eventsv1.CampaignHypothesisDissolution
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != "ghosttrace.events.v1.CampaignHypothesisDissolution" {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return err
		}
		ev := &eventsv1.CampaignHypothesisDissolution{}
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

func TestDissolveCampaignHypothesisDirectFromFormation(t *testing.T) {
	sub, formation := formCampaignAndCollect(t)
	rep, err := DissolveCampaignHypothesis(context.Background(), sub, CampaignHypothesisDissolveOptions{
		FormationEventHash: formation,
		DissolvedAt:        1716120000000000000,
		Reason:             "campaign spurious",
	}, nil)
	if err != nil {
		t.Fatalf("DissolveCampaignHypothesis: %v", err)
	}
	if rep.AlreadyDissolved {
		t.Errorf("first invocation should not report AlreadyDissolved")
	}
	if len(walkCampaignDissolutions(t, sub)) != 1 {
		t.Fatalf("expected 1 dissolution")
	}
}

func TestDissolveCampaignHypothesisIdempotent(t *testing.T) {
	sub, formation := formCampaignAndCollect(t)
	opts := CampaignHypothesisDissolveOptions{
		FormationEventHash: formation,
		DissolvedAt:        1200 * int64(time.Second),
		Reason:             "terminal",
	}
	ctx := context.Background()
	rep1, _ := DissolveCampaignHypothesis(ctx, sub, opts, nil)
	rep2, _ := DissolveCampaignHypothesis(ctx, sub, opts, nil)
	if !rep2.AlreadyDissolved {
		t.Errorf("second invocation should report AlreadyDissolved")
	}
	if rep1.DissolutionEventHashHex != rep2.DissolutionEventHashHex {
		t.Errorf("idempotency violated")
	}
}

func TestDissolveCampaignHypothesisUnknownTarget(t *testing.T) {
	sub, _ := formCampaignAndCollect(t)
	var bogus [32]byte
	for i := range bogus {
		bogus[i] = byte(0xff - i)
	}
	_, err := DissolveCampaignHypothesis(context.Background(), sub, CampaignHypothesisDissolveOptions{
		FormationEventHash: bogus,
	}, nil)
	if !errors.Is(err, ErrTargetNotFound) {
		t.Errorf("expected ErrTargetNotFound; got %v", err)
	}
}

func TestDissolveCampaignHypothesisRejectsPromotionHash(t *testing.T) {
	sub, promotionHash := formCampaignAndPromote(t, 1000*int64(time.Second), 60)
	_, err := DissolveCampaignHypothesis(context.Background(), sub, CampaignHypothesisDissolveOptions{
		FormationEventHash: promotionHash,
	}, nil)
	if !errors.Is(err, ErrTargetWrongType) {
		t.Errorf("expected ErrTargetWrongType for promotion hash; got %v", err)
	}
}

func TestDissolveCampaignHypothesisRejectsCrossSubtypeBC(t *testing.T) {
	bcSub, bcFormation := formAndCollect(t)
	_, err := DissolveCampaignHypothesis(context.Background(), bcSub, CampaignHypothesisDissolveOptions{
		FormationEventHash: bcFormation,
	}, nil)
	if !errors.Is(err, ErrTargetWrongType) {
		t.Errorf("expected ErrTargetWrongType for BC formation; got %v", err)
	}
}

func TestDissolveCampaignHypothesisRejectsCrossSubtypeAG(t *testing.T) {
	agSub, agFormation := formAutomationGroupAndCollect(t)
	_, err := DissolveCampaignHypothesis(context.Background(), agSub, CampaignHypothesisDissolveOptions{
		FormationEventHash: agFormation,
	}, nil)
	if !errors.Is(err, ErrTargetWrongType) {
		t.Errorf("expected ErrTargetWrongType for AG formation; got %v", err)
	}
}

func TestDissolveCampaignHypothesisDefaultDissolvedAt(t *testing.T) {
	sub, formation := formCampaignAndCollect(t)
	fixedNow := func() time.Time { return time.Unix(0, 9999999999) }
	_, err := DissolveCampaignHypothesis(context.Background(), sub, CampaignHypothesisDissolveOptions{
		FormationEventHash: formation,
	}, fixedNow)
	if err != nil {
		t.Fatalf("DissolveCampaignHypothesis: %v", err)
	}
	dissolutions := walkCampaignDissolutions(t, sub)
	if len(dissolutions) != 1 {
		t.Fatalf("expected 1 dissolution; got %d", len(dissolutions))
	}
	if dissolutions[0].DissolvedAt != 9999999999 {
		t.Errorf("dissolved_at: got %d, want 9999999999", dissolutions[0].DissolvedAt)
	}
}
