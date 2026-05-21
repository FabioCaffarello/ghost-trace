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

// formCampaignAndCollect populates a substrate with three uniformly-
// spaced DeclaredSessions sharing a descriptor, runs
// FormCampaignHypothesisAll under default temporal-descriptor-
// cohort-v1, and returns (substrate, formation hash).
func formCampaignAndCollect(t *testing.T) (*substrate.Substrate, [32]byte) {
	t.Helper()
	gap := int64(60 * 1e9)
	sub := campaignSubstrate(t, []struct {
		Descriptor []byte
		DeclaredAt int64
	}{
		{[]byte("campaign-test"), 1000},
		{[]byte("campaign-test"), 1000 + gap},
		{[]byte("campaign-test"), 1000 + 2*gap},
	})

	if _, err := FormCampaignHypothesisAll(context.Background(), sub,
		TemporalDescriptorCohortV1{MinCampaignSize: 3, MaxIntraEventGapSeconds: 300},
		nil); err != nil {
		t.Fatalf("FormCampaignHypothesisAll: %v", err)
	}
	var formationHash [32]byte
	if err := sub.WalkEvents(context.Background(), func(row substrate.EventRow) error {
		if row.MessageType == campaignHypothesisFormationMessageType {
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

func walkCampaignPromotions(t *testing.T, sub *substrate.Substrate) []*eventsv1.CampaignHypothesisPromotion {
	t.Helper()
	ctx := context.Background()
	var out []*eventsv1.CampaignHypothesisPromotion
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != "ghosttrace.events.v1.CampaignHypothesisPromotion" {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return err
		}
		ev := &eventsv1.CampaignHypothesisPromotion{}
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

func TestPromoteCampaignHypothesisHappyPath(t *testing.T) {
	sub, formation := formCampaignAndCollect(t)
	rep, err := PromoteCampaignHypothesis(context.Background(), sub, CampaignHypothesisPromoteOptions{
		FormationEventHash: formation,
		PromotedAt:         1716120000000000000,
		CadenceSeconds:     3600,
		Reason:             "operational pilot",
	}, nil)
	if err != nil {
		t.Fatalf("PromoteCampaignHypothesis: %v", err)
	}
	if rep.AlreadyPromoted {
		t.Errorf("unexpected AlreadyPromoted on first invocation")
	}
	promotions := walkCampaignPromotions(t, sub)
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

func TestPromoteCampaignHypothesisIdempotent(t *testing.T) {
	sub, formation := formCampaignAndCollect(t)
	opts := CampaignHypothesisPromoteOptions{
		FormationEventHash: formation,
		PromotedAt:         1716120000000000000,
		CadenceSeconds:     3600,
	}
	ctx := context.Background()
	rep1, _ := PromoteCampaignHypothesis(ctx, sub, opts, nil)
	rep2, _ := PromoteCampaignHypothesis(ctx, sub, opts, nil)
	if !rep2.AlreadyPromoted {
		t.Errorf("second invocation should report AlreadyPromoted")
	}
	if rep1.PromotionEventHashHex != rep2.PromotionEventHashHex {
		t.Errorf("idempotency violated")
	}
}

func TestPromoteCampaignHypothesisVersioningProducesNewRecord(t *testing.T) {
	sub, formation := formCampaignAndCollect(t)
	ctx := context.Background()
	if _, err := PromoteCampaignHypothesis(ctx, sub, CampaignHypothesisPromoteOptions{
		FormationEventHash: formation,
		PromotedAt:         1716120000000000000,
		CadenceSeconds:     3600,
	}, nil); err != nil {
		t.Fatalf("first: %v", err)
	}
	rep, err := PromoteCampaignHypothesis(ctx, sub, CampaignHypothesisPromoteOptions{
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
	if got := len(walkCampaignPromotions(t, sub)); got != 2 {
		t.Errorf("substrate holds %d; want 2", got)
	}
}

func TestPromoteCampaignHypothesisRejectsNonPositiveCadence(t *testing.T) {
	sub, formation := formCampaignAndCollect(t)
	for _, cs := range []int64{0, -1} {
		_, err := PromoteCampaignHypothesis(context.Background(), sub, CampaignHypothesisPromoteOptions{
			FormationEventHash: formation,
			CadenceSeconds:     cs,
		}, nil)
		if err == nil {
			t.Errorf("cadence=%d: expected error", cs)
		}
	}
}

func TestPromoteCampaignHypothesisUnknownTarget(t *testing.T) {
	sub, _ := formCampaignAndCollect(t)
	var bogus [32]byte
	for i := range bogus {
		bogus[i] = byte(0xff - i)
	}
	_, err := PromoteCampaignHypothesis(context.Background(), sub, CampaignHypothesisPromoteOptions{
		FormationEventHash: bogus,
		CadenceSeconds:     3600,
	}, nil)
	if !errors.Is(err, ErrTargetNotFound) {
		t.Errorf("expected ErrTargetNotFound; got %v", err)
	}
}

func TestPromoteCampaignHypothesisRejectsCrossSubtypeBC(t *testing.T) {
	// A BehavioralClusterFormation hash MUST be rejected by
	// PromoteCampaignHypothesis. Validates subtype-specific
	// message_type discrimination.
	bcSub, bcFormation := formAndCollect(t)
	_, err := PromoteCampaignHypothesis(context.Background(), bcSub, CampaignHypothesisPromoteOptions{
		FormationEventHash: bcFormation,
		CadenceSeconds:     3600,
	}, nil)
	if !errors.Is(err, ErrTargetWrongType) {
		t.Errorf("expected ErrTargetWrongType for BC formation; got %v", err)
	}
}

func TestPromoteCampaignHypothesisRejectsCrossSubtypeAG(t *testing.T) {
	// An AutomationGroupFormation hash MUST be rejected.
	agSub, agFormation := formAutomationGroupAndCollect(t)
	_, err := PromoteCampaignHypothesis(context.Background(), agSub, CampaignHypothesisPromoteOptions{
		FormationEventHash: agFormation,
		CadenceSeconds:     3600,
	}, nil)
	if !errors.Is(err, ErrTargetWrongType) {
		t.Errorf("expected ErrTargetWrongType for AG formation; got %v", err)
	}
}

func TestPromoteCampaignHypothesisDefaultPromotedAt(t *testing.T) {
	sub, formation := formCampaignAndCollect(t)
	fixedNow := func() time.Time { return time.Unix(0, 9999999999) }
	_, err := PromoteCampaignHypothesis(context.Background(), sub, CampaignHypothesisPromoteOptions{
		FormationEventHash: formation,
		CadenceSeconds:     60,
	}, fixedNow)
	if err != nil {
		t.Fatalf("PromoteCampaignHypothesis: %v", err)
	}
	promotions := walkCampaignPromotions(t, sub)
	if len(promotions) != 1 {
		t.Fatalf("expected 1 promotion; got %d", len(promotions))
	}
	if promotions[0].PromotedAt != 9999999999 {
		t.Errorf("promoted_at: got %d, want 9999999999", promotions[0].PromotedAt)
	}
}
