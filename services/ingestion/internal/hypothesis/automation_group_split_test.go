package hypothesis

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

func walkAutomationGroupSplits(t *testing.T, sub *substrate.Substrate) []*eventsv1.AutomationGroupSplit {
	t.Helper()
	ctx := context.Background()
	var out []*eventsv1.AutomationGroupSplit
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != "ghosttrace.events.v1.AutomationGroupSplit" {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return err
		}
		ev := &eventsv1.AutomationGroupSplit{}
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

func TestSplitAutomationGroupHappyPath(t *testing.T) {
	sub, alpha, beta, gamma := formThreeAutomationGroups(t)
	rep, err := SplitAutomationGroup(context.Background(), sub, AutomationGroupSplitOptions{
		AntecedentFormationHash:  alpha,
		SuccessorFormationHashes: [][32]byte{beta, gamma},
		SplitAt:                  1716120000000000000,
		Reason:                   "alpha conflated two automation signatures",
	}, nil)
	if err != nil {
		t.Fatalf("SplitAutomationGroup: %v", err)
	}
	if rep.AlreadySplit {
		t.Errorf("unexpected AlreadySplit on first invocation")
	}
	splits := walkAutomationGroupSplits(t, sub)
	if len(splits) != 1 {
		t.Fatalf("substrate carries %d splits; want 1", len(splits))
	}
	got := splits[0]
	if !bytes.Equal(got.AntecedentFormationEventHash, alpha[:]) {
		t.Errorf("antecedent mismatch")
	}
	if len(got.SuccessorFormationEventHashes) != 2 {
		t.Fatalf("successors: got %d; want 2", len(got.SuccessorFormationEventHashes))
	}
	if bytes.Compare(got.SuccessorFormationEventHashes[0], got.SuccessorFormationEventHashes[1]) >= 0 {
		t.Errorf("successors not sorted ascending")
	}
}

func TestSplitAutomationGroupSuccessorOrderInvariance(t *testing.T) {
	sub, alpha, beta, gamma := formThreeAutomationGroups(t)
	ctx := context.Background()
	rep1, _ := SplitAutomationGroup(ctx, sub, AutomationGroupSplitOptions{
		AntecedentFormationHash:  alpha,
		SuccessorFormationHashes: [][32]byte{beta, gamma},
		SplitAt:                  1716120000000000000,
		Reason:                   "order",
	}, nil)
	rep2, _ := SplitAutomationGroup(ctx, sub, AutomationGroupSplitOptions{
		AntecedentFormationHash:  alpha,
		SuccessorFormationHashes: [][32]byte{gamma, beta}, // swapped
		SplitAt:                  1716120000000000000,
		Reason:                   "order",
	}, nil)
	if rep1.SplitEventHashHex != rep2.SplitEventHashHex {
		t.Errorf("successor-order invariance violated")
	}
	if !rep2.AlreadySplit {
		t.Errorf("swapped invocation should report AlreadySplit")
	}
	if got := len(walkAutomationGroupSplits(t, sub)); got != 1 {
		t.Errorf("substrate holds %d splits; want 1", got)
	}
}

func TestSplitAutomationGroupInsufficientSuccessors(t *testing.T) {
	sub, alpha, beta, _ := formThreeAutomationGroups(t)
	_, err := SplitAutomationGroup(context.Background(), sub, AutomationGroupSplitOptions{
		AntecedentFormationHash:  alpha,
		SuccessorFormationHashes: [][32]byte{beta},
	}, nil)
	if !errors.Is(err, ErrSplitInsufficientSuccessors) {
		t.Errorf("expected ErrSplitInsufficientSuccessors; got %v", err)
	}
}

func TestSplitAutomationGroupDuplicateSuccessors(t *testing.T) {
	sub, alpha, beta, _ := formThreeAutomationGroups(t)
	_, err := SplitAutomationGroup(context.Background(), sub, AutomationGroupSplitOptions{
		AntecedentFormationHash:  alpha,
		SuccessorFormationHashes: [][32]byte{beta, beta},
	}, nil)
	if !errors.Is(err, ErrSplitSuccessorsNotDistinct) {
		t.Errorf("expected ErrSplitSuccessorsNotDistinct; got %v", err)
	}
}

func TestSplitAutomationGroupAntecedentEqualsSuccessor(t *testing.T) {
	sub, alpha, beta, _ := formThreeAutomationGroups(t)
	_, err := SplitAutomationGroup(context.Background(), sub, AutomationGroupSplitOptions{
		AntecedentFormationHash:  alpha,
		SuccessorFormationHashes: [][32]byte{alpha, beta},
	}, nil)
	if !errors.Is(err, ErrSplitSuccessorsNotDistinct) {
		t.Errorf("expected ErrSplitSuccessorsNotDistinct; got %v", err)
	}
}

func TestSplitAutomationGroupIdempotent(t *testing.T) {
	sub, alpha, beta, gamma := formThreeAutomationGroups(t)
	opts := AutomationGroupSplitOptions{
		AntecedentFormationHash:  alpha,
		SuccessorFormationHashes: [][32]byte{beta, gamma},
		SplitAt:                  1716120000000000000,
		Reason:                   "idempotent",
	}
	ctx := context.Background()
	rep1, _ := SplitAutomationGroup(ctx, sub, opts, nil)
	rep2, _ := SplitAutomationGroup(ctx, sub, opts, nil)
	if !rep2.AlreadySplit {
		t.Errorf("second invocation should report AlreadySplit")
	}
	if rep1.SplitEventHashHex != rep2.SplitEventHashHex {
		t.Errorf("idempotency violated")
	}
}

func TestSplitAutomationGroupUnknownAntecedent(t *testing.T) {
	sub, _, beta, gamma := formThreeAutomationGroups(t)
	var bogus [32]byte
	for i := range bogus {
		bogus[i] = byte(0xff - i)
	}
	_, err := SplitAutomationGroup(context.Background(), sub, AutomationGroupSplitOptions{
		AntecedentFormationHash:  bogus,
		SuccessorFormationHashes: [][32]byte{beta, gamma},
	}, nil)
	if !errors.Is(err, ErrTargetNotFound) {
		t.Errorf("expected ErrTargetNotFound; got %v", err)
	}
}

func TestSplitAutomationGroupUnknownSuccessor(t *testing.T) {
	sub, alpha, beta, _ := formThreeAutomationGroups(t)
	var bogus [32]byte
	for i := range bogus {
		bogus[i] = byte(0xaa)
	}
	_, err := SplitAutomationGroup(context.Background(), sub, AutomationGroupSplitOptions{
		AntecedentFormationHash:  alpha,
		SuccessorFormationHashes: [][32]byte{beta, bogus},
	}, nil)
	if !errors.Is(err, ErrTargetNotFound) {
		t.Errorf("expected ErrTargetNotFound; got %v", err)
	}
}

func TestSplitAutomationGroupWrongTypeTarget(t *testing.T) {
	// A PROMOTION hash passed as a successor → ErrTargetWrongType.
	sub, promotionHash := formAutomationGroupAndPromote(t, 1000*int64(time.Second), 60)
	var formationHash [32]byte
	if err := sub.WalkEvents(context.Background(), func(row substrate.EventRow) error {
		if row.MessageType == automationGroupFormationMessageType {
			formationHash = row.EventHash
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	var otherSuccessor [32]byte
	for i := range otherSuccessor {
		otherSuccessor[i] = byte(0x5a)
	}
	_, err := SplitAutomationGroup(context.Background(), sub, AutomationGroupSplitOptions{
		AntecedentFormationHash:  formationHash,
		SuccessorFormationHashes: [][32]byte{promotionHash, otherSuccessor},
	}, nil)
	if !errors.Is(err, ErrTargetWrongType) {
		t.Errorf("expected ErrTargetWrongType; got %v", err)
	}
}

func TestSplitAutomationGroupDefaultSplitAt(t *testing.T) {
	sub, alpha, beta, gamma := formThreeAutomationGroups(t)
	fixedNow := func() time.Time { return time.Unix(0, 9999999999) }
	_, err := SplitAutomationGroup(context.Background(), sub, AutomationGroupSplitOptions{
		AntecedentFormationHash:  alpha,
		SuccessorFormationHashes: [][32]byte{beta, gamma},
	}, fixedNow)
	if err != nil {
		t.Fatalf("SplitAutomationGroup: %v", err)
	}
	splits := walkAutomationGroupSplits(t, sub)
	if len(splits) != 1 {
		t.Fatalf("expected 1 split; got %d", len(splits))
	}
	if splits[0].SplitAt != 9999999999 {
		t.Errorf("split_at: got %d; want 9999999999", splits[0].SplitAt)
	}
}

func TestSplitAutomationGroupAllSixLifecycleOpsInSubstrate(t *testing.T) {
	// Substrate carrying ALL SIX AutomationGroup lifecycle ops:
	// formation, promotion, demotion, dissolution, merge, split.
	// Closes the AutomationGroup lifecycle arc — second subtype
	// surface complete.
	sub, alpha, beta, gamma := formThreeAutomationGroups(t)
	ctx := context.Background()

	// Promote alpha.
	promRep, err := PromoteAutomationGroup(ctx, sub, AutomationGroupPromoteOptions{
		FormationEventHash: alpha,
		PromotedAt:         1000 * int64(time.Second),
		CadenceSeconds:     60,
	}, nil)
	if err != nil {
		t.Fatalf("Promote: %v", err)
	}
	var promotionHash [32]byte
	if raw, _ := hexDecode(promRep.PromotionEventHashHex); true {
		copy(promotionHash[:], raw)
	}
	// Demote.
	if _, err := DemoteAutomationGroup(ctx, sub, AutomationGroupDemoteOptions{
		PromotionEventHash: promotionHash,
		DemotedAt:          1100 * int64(time.Second),
	}, nil); err != nil {
		t.Fatalf("Demote: %v", err)
	}
	// Dissolve alpha.
	if _, err := DissolveAutomationGroup(ctx, sub, AutomationGroupDissolveOptions{
		FormationEventHash: alpha,
		DissolvedAt:        1200 * int64(time.Second),
	}, nil); err != nil {
		t.Fatalf("Dissolve: %v", err)
	}
	// Merge alpha + beta → gamma.
	if _, err := MergeAutomationGroup(ctx, sub, AutomationGroupMergeOptions{
		AntecedentAFormationHash: alpha,
		AntecedentBFormationHash: beta,
		ProducedFormationHash:    gamma,
		MergedAt:                 1300 * int64(time.Second),
	}, nil); err != nil {
		t.Fatalf("Merge: %v", err)
	}
	// Split gamma → [alpha, beta].
	if _, err := SplitAutomationGroup(ctx, sub, AutomationGroupSplitOptions{
		AntecedentFormationHash:  gamma,
		SuccessorFormationHashes: [][32]byte{alpha, beta},
		SplitAt:                  1400 * int64(time.Second),
	}, nil); err != nil {
		t.Fatalf("Split: %v", err)
	}

	// Confirm substrate carries one of each AG lifecycle type.
	typeCounts := map[string]int{}
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		typeCounts[row.MessageType]++
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	wantTypes := []string{
		"ghosttrace.events.v1.AutomationGroupFormation",
		"ghosttrace.events.v1.AutomationGroupPromotion",
		"ghosttrace.events.v1.AutomationGroupDemotion",
		"ghosttrace.events.v1.AutomationGroupDissolution",
		"ghosttrace.events.v1.AutomationGroupMerge",
		"ghosttrace.events.v1.AutomationGroupSplit",
	}
	for _, mt := range wantTypes {
		if typeCounts[mt] < 1 {
			t.Errorf("substrate missing %s (count=%d)", mt, typeCounts[mt])
		}
	}
	// Three formations expected (alpha + beta + gamma).
	if typeCounts["ghosttrace.events.v1.AutomationGroupFormation"] != 3 {
		t.Errorf("formation count: got %d; want 3", typeCounts["ghosttrace.events.v1.AutomationGroupFormation"])
	}
}
