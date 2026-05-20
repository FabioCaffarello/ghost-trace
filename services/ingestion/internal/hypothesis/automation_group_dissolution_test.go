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

func walkAutomationGroupDissolutions(t *testing.T, sub *substrate.Substrate) []*eventsv1.AutomationGroupDissolution {
	t.Helper()
	ctx := context.Background()
	var out []*eventsv1.AutomationGroupDissolution
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != "ghosttrace.events.v1.AutomationGroupDissolution" {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return err
		}
		ev := &eventsv1.AutomationGroupDissolution{}
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

func TestDissolveAutomationGroupDirectFromFormation(t *testing.T) {
	sub, formation := formAutomationGroupAndCollect(t)
	rep, err := DissolveAutomationGroup(context.Background(), sub, AutomationGroupDissolveOptions{
		FormationEventHash: formation,
		DissolvedAt:        1716120000000000000,
		Reason:             "signature misattributed",
	}, nil)
	if err != nil {
		t.Fatalf("DissolveAutomationGroup: %v", err)
	}
	if rep.AlreadyDissolved {
		t.Errorf("first invocation should not report AlreadyDissolved")
	}
	if len(walkAutomationGroupDissolutions(t, sub)) != 1 {
		t.Fatalf("expected 1 dissolution")
	}
}

func TestDissolveAutomationGroupAfterPromoteDemote(t *testing.T) {
	sub, promotionHash := formAutomationGroupAndPromote(t, 1000*int64(time.Second), 60)
	ctx := context.Background()
	if _, err := DemoteAutomationGroup(ctx, sub, AutomationGroupDemoteOptions{
		PromotionEventHash: promotionHash,
		DemotedAt:          1100 * int64(time.Second),
	}, nil); err != nil {
		t.Fatalf("Demote: %v", err)
	}
	var formationHash [32]byte
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType == automationGroupFormationMessageType {
			formationHash = row.EventHash
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	if _, err := DissolveAutomationGroup(ctx, sub, AutomationGroupDissolveOptions{
		FormationEventHash: formationHash,
		DissolvedAt:        1200 * int64(time.Second),
		Reason:             "terminal",
	}, nil); err != nil {
		t.Fatalf("Dissolve: %v", err)
	}
	if len(walkAutomationGroupDissolutions(t, sub)) != 1 {
		t.Fatal("expected 1 dissolution")
	}
}

func TestDissolveAutomationGroupIdempotent(t *testing.T) {
	sub, formation := formAutomationGroupAndCollect(t)
	opts := AutomationGroupDissolveOptions{
		FormationEventHash: formation,
		DissolvedAt:        1200 * int64(time.Second),
		Reason:             "terminal",
	}
	ctx := context.Background()
	rep1, _ := DissolveAutomationGroup(ctx, sub, opts, nil)
	rep2, _ := DissolveAutomationGroup(ctx, sub, opts, nil)
	if !rep2.AlreadyDissolved {
		t.Errorf("second invocation should report AlreadyDissolved")
	}
	if rep1.DissolutionEventHashHex != rep2.DissolutionEventHashHex {
		t.Errorf("idempotency violated")
	}
	if got := len(walkAutomationGroupDissolutions(t, sub)); got != 1 {
		t.Errorf("substrate holds %d; want 1", got)
	}
}

func TestDissolveAutomationGroupUnknownTarget(t *testing.T) {
	sub, _ := formAutomationGroupAndCollect(t)
	var bogus [32]byte
	for i := range bogus {
		bogus[i] = byte(0xff - i)
	}
	_, err := DissolveAutomationGroup(context.Background(), sub, AutomationGroupDissolveOptions{
		FormationEventHash: bogus,
	}, nil)
	if !errors.Is(err, ErrTargetNotFound) {
		t.Errorf("expected ErrTargetNotFound; got %v", err)
	}
}

func TestDissolveAutomationGroupRejectsPromotionHash(t *testing.T) {
	sub, promotionHash := formAutomationGroupAndPromote(t, 1000*int64(time.Second), 60)
	_, err := DissolveAutomationGroup(context.Background(), sub, AutomationGroupDissolveOptions{
		FormationEventHash: promotionHash,
	}, nil)
	if !errors.Is(err, ErrTargetWrongType) {
		t.Errorf("expected ErrTargetWrongType for promotion hash; got %v", err)
	}
}

func TestDissolveAutomationGroupRejectsCrossSubtypeFormation(t *testing.T) {
	bcSub, bcFormation := formAndCollect(t)
	_, err := DissolveAutomationGroup(context.Background(), bcSub, AutomationGroupDissolveOptions{
		FormationEventHash: bcFormation,
	}, nil)
	if !errors.Is(err, ErrTargetWrongType) {
		t.Errorf("expected ErrTargetWrongType for BehavioralClusterFormation hash; got %v", err)
	}
}

func TestDissolveAutomationGroupDefaultDissolvedAt(t *testing.T) {
	sub, formation := formAutomationGroupAndCollect(t)
	fixedNow := func() time.Time { return time.Unix(0, 9999999999) }
	_, err := DissolveAutomationGroup(context.Background(), sub, AutomationGroupDissolveOptions{
		FormationEventHash: formation,
	}, fixedNow)
	if err != nil {
		t.Fatalf("DissolveAutomationGroup: %v", err)
	}
	dissolutions := walkAutomationGroupDissolutions(t, sub)
	if len(dissolutions) != 1 {
		t.Fatalf("expected 1 dissolution; got %d", len(dissolutions))
	}
	if dissolutions[0].DissolvedAt != 9999999999 {
		t.Errorf("dissolved_at: got %d, want 9999999999", dissolutions[0].DissolvedAt)
	}
}
