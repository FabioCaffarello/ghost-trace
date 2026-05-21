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

// formCampaignAndPromote populates a substrate with a campaign
// formation + one promotion under the supplied parameters; returns
// (substrate, promotion hash).
func formCampaignAndPromote(t *testing.T, promotedAt, cadenceSeconds int64) (*substrate.Substrate, [32]byte) {
	t.Helper()
	sub, formation := formCampaignAndCollect(t)
	rep, err := PromoteCampaignHypothesis(context.Background(), sub, CampaignHypothesisPromoteOptions{
		FormationEventHash: formation,
		PromotedAt:         promotedAt,
		CadenceSeconds:     cadenceSeconds,
	}, nil)
	if err != nil {
		t.Fatalf("PromoteCampaignHypothesis: %v", err)
	}
	var promotionHash [32]byte
	raw, err := hexDecode(rep.PromotionEventHashHex)
	if err != nil {
		t.Fatalf("decode promotion hash: %v", err)
	}
	copy(promotionHash[:], raw)
	return sub, promotionHash
}

func walkCampaignDemotions(t *testing.T, sub *substrate.Substrate) []*eventsv1.CampaignHypothesisDemotion {
	t.Helper()
	ctx := context.Background()
	var out []*eventsv1.CampaignHypothesisDemotion
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != "ghosttrace.events.v1.CampaignHypothesisDemotion" {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return err
		}
		ev := &eventsv1.CampaignHypothesisDemotion{}
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

func TestDemoteCampaignHypothesisHappyPathCadenceSatisfied(t *testing.T) {
	sub, promotionHash := formCampaignAndPromote(t, 1000*int64(time.Second), 60)
	rep, err := DemoteCampaignHypothesis(context.Background(), sub, CampaignHypothesisDemoteOptions{
		PromotionEventHash: promotionHash,
		DemotedAt:          1100 * int64(time.Second),
		Reason:             "operational cycle close",
	}, nil)
	if err != nil {
		t.Fatalf("DemoteCampaignHypothesis: %v", err)
	}
	if !rep.CadenceSatisfied {
		t.Errorf("CadenceSatisfied: got false, want true (elapsed 100s > cadence 60s)")
	}
	if rep.CadenceElapsedSeconds != 100 {
		t.Errorf("CadenceElapsedSeconds: got %d, want 100", rep.CadenceElapsedSeconds)
	}
	if len(walkCampaignDemotions(t, sub)) != 1 {
		t.Fatalf("expected 1 demotion")
	}
}

func TestDemoteCampaignHypothesisCadenceUnsatisfied(t *testing.T) {
	sub, promotionHash := formCampaignAndPromote(t, 1000*int64(time.Second), 3600)
	rep, err := DemoteCampaignHypothesis(context.Background(), sub, CampaignHypothesisDemoteOptions{
		PromotionEventHash: promotionHash,
		DemotedAt:          1010 * int64(time.Second),
		Reason:             "early demote",
	}, nil)
	if err != nil {
		t.Fatalf("DemoteCampaignHypothesis: %v", err)
	}
	if rep.CadenceSatisfied {
		t.Errorf("CadenceSatisfied: got true, want false (elapsed 10s < cadence 3600s)")
	}
	if len(walkCampaignDemotions(t, sub)) != 1 {
		t.Errorf("substrate must record the early demotion")
	}
}

func TestDemoteCampaignHypothesisIdempotent(t *testing.T) {
	sub, promotionHash := formCampaignAndPromote(t, 1000*int64(time.Second), 60)
	opts := CampaignHypothesisDemoteOptions{
		PromotionEventHash: promotionHash,
		DemotedAt:          1100 * int64(time.Second),
	}
	ctx := context.Background()
	rep1, _ := DemoteCampaignHypothesis(ctx, sub, opts, nil)
	rep2, _ := DemoteCampaignHypothesis(ctx, sub, opts, nil)
	if !rep2.AlreadyDemoted {
		t.Errorf("second invocation should report AlreadyDemoted")
	}
	if rep1.DemotionEventHashHex != rep2.DemotionEventHashHex {
		t.Errorf("idempotency violated")
	}
}

func TestDemoteCampaignHypothesisUnknownTarget(t *testing.T) {
	sub, _ := formCampaignAndPromote(t, 1000*int64(time.Second), 60)
	var bogus [32]byte
	for i := range bogus {
		bogus[i] = byte(0xff - i)
	}
	_, err := DemoteCampaignHypothesis(context.Background(), sub, CampaignHypothesisDemoteOptions{
		PromotionEventHash: bogus,
	}, nil)
	if !errors.Is(err, ErrTargetNotFound) {
		t.Errorf("expected ErrTargetNotFound; got %v", err)
	}
}

func TestDemoteCampaignHypothesisRejectsFormationHash(t *testing.T) {
	sub, formation := formCampaignAndCollect(t)
	_, err := DemoteCampaignHypothesis(context.Background(), sub, CampaignHypothesisDemoteOptions{
		PromotionEventHash: formation,
	}, nil)
	if !errors.Is(err, ErrTargetWrongType) {
		t.Errorf("expected ErrTargetWrongType for formation hash; got %v", err)
	}
}

func TestDemoteCampaignHypothesisRejectsCrossSubtype(t *testing.T) {
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
	_, err := DemoteCampaignHypothesis(context.Background(), bcSub, CampaignHypothesisDemoteOptions{
		PromotionEventHash: bcPromHash,
	}, nil)
	if !errors.Is(err, ErrTargetWrongType) {
		t.Errorf("expected ErrTargetWrongType for cross-subtype promotion; got %v", err)
	}
}

func TestDemoteCampaignHypothesisDefaultDemotedAt(t *testing.T) {
	sub, promotionHash := formCampaignAndPromote(t, 1000*int64(time.Second), 60)
	fixedNow := func() time.Time { return time.Unix(0, 9999999999) }
	_, err := DemoteCampaignHypothesis(context.Background(), sub, CampaignHypothesisDemoteOptions{
		PromotionEventHash: promotionHash,
	}, fixedNow)
	if err != nil {
		t.Fatalf("DemoteCampaignHypothesis: %v", err)
	}
	demotions := walkCampaignDemotions(t, sub)
	if len(demotions) != 1 {
		t.Fatalf("expected 1 demotion; got %d", len(demotions))
	}
	if demotions[0].DemotedAt != 9999999999 {
		t.Errorf("demoted_at: got %d, want 9999999999", demotions[0].DemotedAt)
	}
}
