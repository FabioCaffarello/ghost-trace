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

// formAutomationGroupAndPromote populates a substrate with an
// AutomationGroup formation + one promotion under the supplied
// (promotedAt, cadenceSeconds) and returns (substrate, promotion hash).
func formAutomationGroupAndPromote(t *testing.T, promotedAt int64, cadenceSeconds int64) (*substrate.Substrate, [32]byte) {
	t.Helper()
	sub, formation := formAutomationGroupAndCollect(t)
	rep, err := PromoteAutomationGroup(context.Background(), sub, AutomationGroupPromoteOptions{
		FormationEventHash: formation,
		PromotedAt:         promotedAt,
		CadenceSeconds:     cadenceSeconds,
	}, nil)
	if err != nil {
		t.Fatalf("PromoteAutomationGroup: %v", err)
	}
	var promotionHash [32]byte
	raw, err := hexDecode(rep.PromotionEventHashHex)
	if err != nil {
		t.Fatalf("decode promotion hash: %v", err)
	}
	copy(promotionHash[:], raw)
	return sub, promotionHash
}

func walkAutomationGroupDemotions(t *testing.T, sub *substrate.Substrate) []*eventsv1.AutomationGroupDemotion {
	t.Helper()
	ctx := context.Background()
	var out []*eventsv1.AutomationGroupDemotion
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != "ghosttrace.events.v1.AutomationGroupDemotion" {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return err
		}
		ev := &eventsv1.AutomationGroupDemotion{}
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

func TestDemoteAutomationGroupHappyPathCadenceSatisfied(t *testing.T) {
	sub, promotionHash := formAutomationGroupAndPromote(t, 1000*int64(time.Second), 60)
	rep, err := DemoteAutomationGroup(context.Background(), sub, AutomationGroupDemoteOptions{
		PromotionEventHash: promotionHash,
		DemotedAt:          1100 * int64(time.Second),
		Reason:             "operational cycle close",
	}, nil)
	if err != nil {
		t.Fatalf("DemoteAutomationGroup: %v", err)
	}
	if !rep.CadenceSatisfied {
		t.Errorf("CadenceSatisfied: got false, want true (elapsed 100s > cadence 60s)")
	}
	if rep.CadenceElapsedSeconds != 100 {
		t.Errorf("CadenceElapsedSeconds: got %d, want 100", rep.CadenceElapsedSeconds)
	}
	if len(walkAutomationGroupDemotions(t, sub)) != 1 {
		t.Fatalf("expected 1 demotion")
	}
}

func TestDemoteAutomationGroupCadenceUnsatisfied(t *testing.T) {
	sub, promotionHash := formAutomationGroupAndPromote(t, 1000*int64(time.Second), 3600)
	rep, err := DemoteAutomationGroup(context.Background(), sub, AutomationGroupDemoteOptions{
		PromotionEventHash: promotionHash,
		DemotedAt:          1010 * int64(time.Second),
		Reason:             "early demote",
	}, nil)
	if err != nil {
		t.Fatalf("DemoteAutomationGroup: %v", err)
	}
	if rep.CadenceSatisfied {
		t.Errorf("CadenceSatisfied: got true, want false (elapsed 10s < cadence 3600s)")
	}
	if rep.CadenceElapsedSeconds != 10 {
		t.Errorf("CadenceElapsedSeconds: got %d, want 10", rep.CadenceElapsedSeconds)
	}
	if len(walkAutomationGroupDemotions(t, sub)) != 1 {
		t.Errorf("substrate must record the early demotion regardless of gate state")
	}
}

func TestDemoteAutomationGroupIdempotent(t *testing.T) {
	sub, promotionHash := formAutomationGroupAndPromote(t, 1000*int64(time.Second), 60)
	opts := AutomationGroupDemoteOptions{
		PromotionEventHash: promotionHash,
		DemotedAt:          1100 * int64(time.Second),
		Reason:             "idempotent",
	}
	ctx := context.Background()
	rep1, _ := DemoteAutomationGroup(ctx, sub, opts, nil)
	rep2, _ := DemoteAutomationGroup(ctx, sub, opts, nil)
	if !rep2.AlreadyDemoted {
		t.Errorf("second invocation should report AlreadyDemoted")
	}
	if rep1.DemotionEventHashHex != rep2.DemotionEventHashHex {
		t.Errorf("idempotency violated")
	}
	if got := len(walkAutomationGroupDemotions(t, sub)); got != 1 {
		t.Errorf("substrate holds %d; want 1", got)
	}
}

func TestDemoteAutomationGroupUnknownTarget(t *testing.T) {
	sub, _ := formAutomationGroupAndPromote(t, 1000*int64(time.Second), 60)
	var bogus [32]byte
	for i := range bogus {
		bogus[i] = byte(0xff - i)
	}
	_, err := DemoteAutomationGroup(context.Background(), sub, AutomationGroupDemoteOptions{
		PromotionEventHash: bogus,
	}, nil)
	if !errors.Is(err, ErrTargetNotFound) {
		t.Errorf("expected ErrTargetNotFound; got %v", err)
	}
}

func TestDemoteAutomationGroupRejectsFormationHash(t *testing.T) {
	sub, formation := formAutomationGroupAndCollect(t)
	_, err := DemoteAutomationGroup(context.Background(), sub, AutomationGroupDemoteOptions{
		PromotionEventHash: formation,
	}, nil)
	if !errors.Is(err, ErrTargetWrongType) {
		t.Errorf("expected ErrTargetWrongType for formation hash; got %v", err)
	}
}

func TestDemoteAutomationGroupRejectsCrossSubtypePromotion(t *testing.T) {
	bcSub, bcFormation := formAndCollect(t)
	bcPromRep, err := Promote(context.Background(), bcSub, PromoteOptions{
		FormationEventHash: bcFormation,
		PromotedAt:         1000 * int64(time.Second),
		CadenceSeconds:     60,
	}, nil)
	if err != nil {
		t.Fatalf("BC Promote: %v", err)
	}
	var bcPromHash [32]byte
	if raw, _ := hexDecode(bcPromRep.PromotionEventHashHex); true {
		copy(bcPromHash[:], raw)
	}

	_, err = DemoteAutomationGroup(context.Background(), bcSub, AutomationGroupDemoteOptions{
		PromotionEventHash: bcPromHash,
	}, nil)
	if !errors.Is(err, ErrTargetWrongType) {
		t.Errorf("expected ErrTargetWrongType for cross-subtype promotion; got %v", err)
	}
}

func TestDemoteAutomationGroupDefaultDemotedAt(t *testing.T) {
	sub, promotionHash := formAutomationGroupAndPromote(t, 1000*int64(time.Second), 60)
	fixedNow := func() time.Time { return time.Unix(0, 9999999999) }
	rep, err := DemoteAutomationGroup(context.Background(), sub, AutomationGroupDemoteOptions{
		PromotionEventHash: promotionHash,
	}, fixedNow)
	if err != nil {
		t.Fatalf("DemoteAutomationGroup: %v", err)
	}
	demotions := walkAutomationGroupDemotions(t, sub)
	if len(demotions) != 1 {
		t.Fatalf("expected 1 demotion; got %d", len(demotions))
	}
	if demotions[0].DemotedAt != 9999999999 {
		t.Errorf("demoted_at: got %d, want 9999999999", demotions[0].DemotedAt)
	}
	if rep.CadenceElapsedSeconds >= 0 {
		t.Errorf("expected negative elapsed (demoted_at < promoted_at); got %d", rep.CadenceElapsedSeconds)
	}
}
