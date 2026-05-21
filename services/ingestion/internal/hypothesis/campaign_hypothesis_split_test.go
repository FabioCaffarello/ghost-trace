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

func walkCampaignSplits(t *testing.T, sub *substrate.Substrate) []*eventsv1.CampaignHypothesisSplit {
	t.Helper()
	ctx := context.Background()
	var out []*eventsv1.CampaignHypothesisSplit
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != "ghosttrace.events.v1.CampaignHypothesisSplit" {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return err
		}
		ev := &eventsv1.CampaignHypothesisSplit{}
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

func TestSplitCampaignHypothesisHappyPath(t *testing.T) {
	sub, alpha, beta, gamma := formThreeCampaignHypotheses(t)
	rep, err := SplitCampaignHypothesis(context.Background(), sub, CampaignHypothesisSplitOptions{
		AntecedentFormationHash:  alpha,
		SuccessorFormationHashes: [][32]byte{beta, gamma},
		SplitAt:                  1716120000000000000,
		Reason:                   "alpha conflated two distinct operations",
	}, nil)
	if err != nil {
		t.Fatalf("SplitCampaignHypothesis: %v", err)
	}
	if rep.AlreadySplit {
		t.Errorf("unexpected AlreadySplit")
	}
	splits := walkCampaignSplits(t, sub)
	if len(splits) != 1 {
		t.Fatalf("expected 1 split; got %d", len(splits))
	}
	got := splits[0]
	if !bytes.Equal(got.AntecedentFormationEventHash, alpha[:]) {
		t.Errorf("antecedent mismatch")
	}
	if len(got.SuccessorFormationEventHashes) != 2 {
		t.Fatalf("successors: got %d, want 2", len(got.SuccessorFormationEventHashes))
	}
	if bytes.Compare(got.SuccessorFormationEventHashes[0], got.SuccessorFormationEventHashes[1]) >= 0 {
		t.Errorf("successors not sorted ascending")
	}
}

func TestSplitCampaignHypothesisSuccessorOrderInvariance(t *testing.T) {
	sub, alpha, beta, gamma := formThreeCampaignHypotheses(t)
	ctx := context.Background()
	rep1, _ := SplitCampaignHypothesis(ctx, sub, CampaignHypothesisSplitOptions{
		AntecedentFormationHash:  alpha,
		SuccessorFormationHashes: [][32]byte{beta, gamma},
		SplitAt:                  1716120000000000000,
		Reason:                   "order",
	}, nil)
	rep2, _ := SplitCampaignHypothesis(ctx, sub, CampaignHypothesisSplitOptions{
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
}

func TestSplitCampaignHypothesisInsufficientSuccessors(t *testing.T) {
	sub, alpha, beta, _ := formThreeCampaignHypotheses(t)
	_, err := SplitCampaignHypothesis(context.Background(), sub, CampaignHypothesisSplitOptions{
		AntecedentFormationHash:  alpha,
		SuccessorFormationHashes: [][32]byte{beta},
	}, nil)
	if !errors.Is(err, ErrSplitInsufficientSuccessors) {
		t.Errorf("expected ErrSplitInsufficientSuccessors; got %v", err)
	}
}

func TestSplitCampaignHypothesisDuplicateSuccessors(t *testing.T) {
	sub, alpha, beta, _ := formThreeCampaignHypotheses(t)
	_, err := SplitCampaignHypothesis(context.Background(), sub, CampaignHypothesisSplitOptions{
		AntecedentFormationHash:  alpha,
		SuccessorFormationHashes: [][32]byte{beta, beta},
	}, nil)
	if !errors.Is(err, ErrSplitSuccessorsNotDistinct) {
		t.Errorf("expected ErrSplitSuccessorsNotDistinct; got %v", err)
	}
}

func TestSplitCampaignHypothesisAntecedentEqualsSuccessor(t *testing.T) {
	sub, alpha, beta, _ := formThreeCampaignHypotheses(t)
	_, err := SplitCampaignHypothesis(context.Background(), sub, CampaignHypothesisSplitOptions{
		AntecedentFormationHash:  alpha,
		SuccessorFormationHashes: [][32]byte{alpha, beta},
	}, nil)
	if !errors.Is(err, ErrSplitSuccessorsNotDistinct) {
		t.Errorf("expected ErrSplitSuccessorsNotDistinct; got %v", err)
	}
}

func TestSplitCampaignHypothesisIdempotent(t *testing.T) {
	sub, alpha, beta, gamma := formThreeCampaignHypotheses(t)
	opts := CampaignHypothesisSplitOptions{
		AntecedentFormationHash:  alpha,
		SuccessorFormationHashes: [][32]byte{beta, gamma},
		SplitAt:                  1716120000000000000,
	}
	ctx := context.Background()
	rep1, _ := SplitCampaignHypothesis(ctx, sub, opts, nil)
	rep2, _ := SplitCampaignHypothesis(ctx, sub, opts, nil)
	if !rep2.AlreadySplit {
		t.Errorf("second invocation should report AlreadySplit")
	}
	if rep1.SplitEventHashHex != rep2.SplitEventHashHex {
		t.Errorf("idempotency violated")
	}
}

func TestSplitCampaignHypothesisUnknownAntecedent(t *testing.T) {
	sub, _, beta, gamma := formThreeCampaignHypotheses(t)
	var bogus [32]byte
	for i := range bogus {
		bogus[i] = byte(0xff - i)
	}
	_, err := SplitCampaignHypothesis(context.Background(), sub, CampaignHypothesisSplitOptions{
		AntecedentFormationHash:  bogus,
		SuccessorFormationHashes: [][32]byte{beta, gamma},
	}, nil)
	if !errors.Is(err, ErrTargetNotFound) {
		t.Errorf("expected ErrTargetNotFound; got %v", err)
	}
}

func TestSplitCampaignHypothesisDefaultSplitAt(t *testing.T) {
	sub, alpha, beta, gamma := formThreeCampaignHypotheses(t)
	fixedNow := func() time.Time { return time.Unix(0, 9999999999) }
	_, err := SplitCampaignHypothesis(context.Background(), sub, CampaignHypothesisSplitOptions{
		AntecedentFormationHash:  alpha,
		SuccessorFormationHashes: [][32]byte{beta, gamma},
	}, fixedNow)
	if err != nil {
		t.Fatalf("SplitCampaignHypothesis: %v", err)
	}
	splits := walkCampaignSplits(t, sub)
	if len(splits) != 1 {
		t.Fatalf("expected 1 split; got %d", len(splits))
	}
	if splits[0].SplitAt != 9999999999 {
		t.Errorf("split_at: got %d", splits[0].SplitAt)
	}
}

func TestSplitCampaignHypothesisAllSixLifecycleOpsInSubstrate(t *testing.T) {
	// Substrate carrying ALL SIX CampaignHypothesis lifecycle ops:
	// formation, promotion, demotion, dissolution, merge, split.
	// Terminal end-to-end coverage for the third subtype.
	sub, alpha, beta, gamma := formThreeCampaignHypotheses(t)
	ctx := context.Background()

	promRep, err := PromoteCampaignHypothesis(ctx, sub, CampaignHypothesisPromoteOptions{
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
	if _, err := DemoteCampaignHypothesis(ctx, sub, CampaignHypothesisDemoteOptions{
		PromotionEventHash: promotionHash,
		DemotedAt:          1100 * int64(time.Second),
	}, nil); err != nil {
		t.Fatalf("Demote: %v", err)
	}
	if _, err := DissolveCampaignHypothesis(ctx, sub, CampaignHypothesisDissolveOptions{
		FormationEventHash: alpha,
		DissolvedAt:        1200 * int64(time.Second),
	}, nil); err != nil {
		t.Fatalf("Dissolve: %v", err)
	}
	if _, err := MergeCampaignHypothesis(ctx, sub, CampaignHypothesisMergeOptions{
		AntecedentAFormationHash: alpha,
		AntecedentBFormationHash: beta,
		ProducedFormationHash:    gamma,
		MergedAt:                 1300 * int64(time.Second),
	}, nil); err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if _, err := SplitCampaignHypothesis(ctx, sub, CampaignHypothesisSplitOptions{
		AntecedentFormationHash:  gamma,
		SuccessorFormationHashes: [][32]byte{alpha, beta},
		SplitAt:                  1400 * int64(time.Second),
	}, nil); err != nil {
		t.Fatalf("Split: %v", err)
	}

	typeCounts := map[string]int{}
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		typeCounts[row.MessageType]++
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	wantTypes := []string{
		"ghosttrace.events.v1.CampaignHypothesisFormation",
		"ghosttrace.events.v1.CampaignHypothesisPromotion",
		"ghosttrace.events.v1.CampaignHypothesisDemotion",
		"ghosttrace.events.v1.CampaignHypothesisDissolution",
		"ghosttrace.events.v1.CampaignHypothesisMerge",
		"ghosttrace.events.v1.CampaignHypothesisSplit",
	}
	for _, mt := range wantTypes {
		if typeCounts[mt] < 1 {
			t.Errorf("substrate missing %s", mt)
		}
	}
	if typeCounts["ghosttrace.events.v1.CampaignHypothesisFormation"] != 3 {
		t.Errorf("formation count: got %d, want 3", typeCounts["ghosttrace.events.v1.CampaignHypothesisFormation"])
	}
}
