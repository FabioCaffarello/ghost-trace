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

func walkCoordinationRingSplits(t *testing.T, sub *substrate.Substrate) []*eventsv1.CoordinationRingSplit {
	t.Helper()
	ctx := context.Background()
	var out []*eventsv1.CoordinationRingSplit
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != "ghosttrace.events.v1.CoordinationRingSplit" {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return err
		}
		ev := &eventsv1.CoordinationRingSplit{}
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

func TestSplitCoordinationRingHappyPath(t *testing.T) {
	sub, alpha, beta, gamma := formThreeCoordinationRings(t)
	rep, err := SplitCoordinationRing(context.Background(), sub, CoordinationRingSplitOptions{
		AntecedentFormationHash:  alpha,
		SuccessorFormationHashes: [][32]byte{beta, gamma},
		SplitAt:                  1716120000000000000,
		Reason:                   "alpha conflated two distinct rings",
	}, nil)
	if err != nil {
		t.Fatalf("SplitCoordinationRing: %v", err)
	}
	if rep.AlreadySplit {
		t.Errorf("unexpected AlreadySplit")
	}
	splits := walkCoordinationRingSplits(t, sub)
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

func TestSplitCoordinationRingSuccessorOrderInvariance(t *testing.T) {
	sub, alpha, beta, gamma := formThreeCoordinationRings(t)
	ctx := context.Background()
	rep1, _ := SplitCoordinationRing(ctx, sub, CoordinationRingSplitOptions{
		AntecedentFormationHash:  alpha,
		SuccessorFormationHashes: [][32]byte{beta, gamma},
		SplitAt:                  1716120000000000000,
		Reason:                   "order",
	}, nil)
	rep2, _ := SplitCoordinationRing(ctx, sub, CoordinationRingSplitOptions{
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

func TestSplitCoordinationRingInsufficientSuccessors(t *testing.T) {
	sub, alpha, beta, _ := formThreeCoordinationRings(t)
	_, err := SplitCoordinationRing(context.Background(), sub, CoordinationRingSplitOptions{
		AntecedentFormationHash:  alpha,
		SuccessorFormationHashes: [][32]byte{beta},
	}, nil)
	if !errors.Is(err, ErrSplitInsufficientSuccessors) {
		t.Errorf("expected ErrSplitInsufficientSuccessors; got %v", err)
	}
}

func TestSplitCoordinationRingDuplicateSuccessors(t *testing.T) {
	sub, alpha, beta, _ := formThreeCoordinationRings(t)
	_, err := SplitCoordinationRing(context.Background(), sub, CoordinationRingSplitOptions{
		AntecedentFormationHash:  alpha,
		SuccessorFormationHashes: [][32]byte{beta, beta},
	}, nil)
	if !errors.Is(err, ErrSplitSuccessorsNotDistinct) {
		t.Errorf("expected ErrSplitSuccessorsNotDistinct; got %v", err)
	}
}

func TestSplitCoordinationRingAntecedentEqualsSuccessor(t *testing.T) {
	sub, alpha, beta, _ := formThreeCoordinationRings(t)
	_, err := SplitCoordinationRing(context.Background(), sub, CoordinationRingSplitOptions{
		AntecedentFormationHash:  alpha,
		SuccessorFormationHashes: [][32]byte{alpha, beta},
	}, nil)
	if !errors.Is(err, ErrSplitSuccessorsNotDistinct) {
		t.Errorf("expected ErrSplitSuccessorsNotDistinct; got %v", err)
	}
}

func TestSplitCoordinationRingIdempotent(t *testing.T) {
	sub, alpha, beta, gamma := formThreeCoordinationRings(t)
	opts := CoordinationRingSplitOptions{
		AntecedentFormationHash:  alpha,
		SuccessorFormationHashes: [][32]byte{beta, gamma},
		SplitAt:                  1716120000000000000,
	}
	ctx := context.Background()
	rep1, _ := SplitCoordinationRing(ctx, sub, opts, nil)
	rep2, _ := SplitCoordinationRing(ctx, sub, opts, nil)
	if !rep2.AlreadySplit {
		t.Errorf("second invocation should report AlreadySplit")
	}
	if rep1.SplitEventHashHex != rep2.SplitEventHashHex {
		t.Errorf("idempotency violated")
	}
}

func TestSplitCoordinationRingUnknownAntecedent(t *testing.T) {
	sub, _, beta, gamma := formThreeCoordinationRings(t)
	var bogus [32]byte
	for i := range bogus {
		bogus[i] = byte(0xff - i)
	}
	_, err := SplitCoordinationRing(context.Background(), sub, CoordinationRingSplitOptions{
		AntecedentFormationHash:  bogus,
		SuccessorFormationHashes: [][32]byte{beta, gamma},
	}, nil)
	if !errors.Is(err, ErrTargetNotFound) {
		t.Errorf("expected ErrTargetNotFound; got %v", err)
	}
}

func TestSplitCoordinationRingDefaultSplitAt(t *testing.T) {
	sub, alpha, beta, gamma := formThreeCoordinationRings(t)
	fixedNow := func() time.Time { return time.Unix(0, 9999999999) }
	_, err := SplitCoordinationRing(context.Background(), sub, CoordinationRingSplitOptions{
		AntecedentFormationHash:  alpha,
		SuccessorFormationHashes: [][32]byte{beta, gamma},
	}, fixedNow)
	if err != nil {
		t.Fatalf("SplitCoordinationRing: %v", err)
	}
	splits := walkCoordinationRingSplits(t, sub)
	if len(splits) != 1 {
		t.Fatalf("expected 1 split; got %d", len(splits))
	}
	if splits[0].SplitAt != 9999999999 {
		t.Errorf("split_at: got %d", splits[0].SplitAt)
	}
}

func TestSplitCoordinationRingAllSixLifecycleOpsInSubstrate(t *testing.T) {
	// Substrate carrying ALL SIX CoordinationRing lifecycle ops:
	// formation, promotion, demotion, dissolution, merge, split.
	// Terminal end-to-end coverage for the FOURTH (and final)
	// Cat III subtype. With this test, all four Cat III subtypes
	// (BC, AG, CH, CR) have a verified all-six-ops-coexist test.
	sub, alpha, beta, gamma := formThreeCoordinationRings(t)
	ctx := context.Background()

	promRep, err := PromoteCoordinationRing(ctx, sub, CoordinationRingPromoteOptions{
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
	if _, err := DemoteCoordinationRing(ctx, sub, CoordinationRingDemoteOptions{
		PromotionEventHash: promotionHash,
		DemotedAt:          1100 * int64(time.Second),
	}, nil); err != nil {
		t.Fatalf("Demote: %v", err)
	}
	if _, err := DissolveCoordinationRing(ctx, sub, CoordinationRingDissolveOptions{
		FormationEventHash: alpha,
		DissolvedAt:        1200 * int64(time.Second),
	}, nil); err != nil {
		t.Fatalf("Dissolve: %v", err)
	}
	if _, err := MergeCoordinationRing(ctx, sub, CoordinationRingMergeOptions{
		AntecedentAFormationHash: alpha,
		AntecedentBFormationHash: beta,
		ProducedFormationHash:    gamma,
		MergedAt:                 1300 * int64(time.Second),
	}, nil); err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if _, err := SplitCoordinationRing(ctx, sub, CoordinationRingSplitOptions{
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
		"ghosttrace.events.v1.CoordinationRingFormation",
		"ghosttrace.events.v1.CoordinationRingPromotion",
		"ghosttrace.events.v1.CoordinationRingDemotion",
		"ghosttrace.events.v1.CoordinationRingDissolution",
		"ghosttrace.events.v1.CoordinationRingMerge",
		"ghosttrace.events.v1.CoordinationRingSplit",
	}
	for _, mt := range wantTypes {
		if typeCounts[mt] < 1 {
			t.Errorf("substrate missing %s", mt)
		}
	}
	if typeCounts["ghosttrace.events.v1.CoordinationRingFormation"] != 3 {
		t.Errorf("formation count: got %d, want 3", typeCounts["ghosttrace.events.v1.CoordinationRingFormation"])
	}
}
