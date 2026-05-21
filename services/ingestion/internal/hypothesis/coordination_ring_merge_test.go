package hypothesis

import (
	"bytes"
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// formThreeCoordinationRings populates a substrate with three
// distinct CoordinationRingFormations (alpha, beta, gamma) for
// merge-test wiring. Each ring has a distinct descriptor + three
// rounds of three-actor co-occurrence so the formation succeeds.
// Returns formations sorted by FormationAt ascending → alpha,
// beta, gamma.
func formThreeCoordinationRings(t *testing.T) (*substrate.Substrate, [32]byte, [32]byte, [32]byte) {
	t.Helper()
	gap := int64(60 * 1e9)
	roundSpacing := int64(2e12)
	rings := []struct {
		desc        string
		base        int64
		a, b, c     string
	}{
		{"ring-alpha", 1000, "actor-a", "actor-b", "actor-c"},
		{"ring-beta", 100_000_000_000, "actor-x", "actor-y", "actor-z"},
		{"ring-gamma", 200_000_000_000_000, "actor-p", "actor-q", "actor-r"},
	}
	var items []struct {
		ActorRef   string
		Descriptor string
		DeclaredAt int64
	}
	for _, r := range rings {
		for round := 0; round < 3; round++ {
			items = append(items,
				struct {
					ActorRef   string
					Descriptor string
					DeclaredAt int64
				}{r.a, r.desc, r.base + int64(round)*roundSpacing},
				struct {
					ActorRef   string
					Descriptor string
					DeclaredAt int64
				}{r.b, r.desc, r.base + int64(round)*roundSpacing + gap},
				struct {
					ActorRef   string
					Descriptor string
					DeclaredAt int64
				}{r.c, r.desc, r.base + int64(round)*roundSpacing + 2*gap},
			)
		}
	}
	sub := crSubstrate(t, items)

	if _, err := FormCoordinationRingAll(context.Background(), sub,
		CoOccurrenceWindowV1{MinEdgeSupport: 3, MaxWindowSeconds: 600}, nil); err != nil {
		t.Fatalf("FormCoordinationRingAll: %v", err)
	}

	type entry struct {
		hash [32]byte
		ev   *eventsv1.CoordinationRingFormation
	}
	var entries []entry
	if err := sub.WalkEvents(context.Background(), func(row substrate.EventRow) error {
		if row.MessageType != coordinationRingFormationMessageType {
			return nil
		}
		payload, err := sub.ReadBlob(context.Background(), row.EventHash)
		if err != nil {
			return err
		}
		ev := &eventsv1.CoordinationRingFormation{}
		if err := proto.Unmarshal(payload, ev); err != nil {
			return err
		}
		entries = append(entries, entry{hash: row.EventHash, ev: ev})
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 formations; got %d", len(entries))
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].ev.FormationAt < entries[j].ev.FormationAt
	})
	return sub, entries[0].hash, entries[1].hash, entries[2].hash
}

func walkCoordinationRingMerges(t *testing.T, sub *substrate.Substrate) []*eventsv1.CoordinationRingMerge {
	t.Helper()
	ctx := context.Background()
	var out []*eventsv1.CoordinationRingMerge
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != "ghosttrace.events.v1.CoordinationRingMerge" {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return err
		}
		ev := &eventsv1.CoordinationRingMerge{}
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

func TestMergeCoordinationRingHappyPath(t *testing.T) {
	sub, alpha, beta, gamma := formThreeCoordinationRings(t)
	rep, err := MergeCoordinationRing(context.Background(), sub, CoordinationRingMergeOptions{
		AntecedentAFormationHash: alpha,
		AntecedentBFormationHash: beta,
		ProducedFormationHash:    gamma,
		MergedAt:                 1716120000000000000,
		Reason:                   "same coordinated-action phenomenon",
	}, nil)
	if err != nil {
		t.Fatalf("MergeCoordinationRing: %v", err)
	}
	if rep.AlreadyMerged {
		t.Errorf("unexpected AlreadyMerged")
	}
	merges := walkCoordinationRingMerges(t, sub)
	if len(merges) != 1 {
		t.Fatalf("expected 1 merge; got %d", len(merges))
	}
	got := merges[0]
	if bytes.Compare(got.AntecedentFormationEventHashes[0], got.AntecedentFormationEventHashes[1]) >= 0 {
		t.Errorf("antecedents not sorted ascending")
	}
	if !bytes.Equal(got.ProducedFormationEventHash, gamma[:]) {
		t.Errorf("produced mismatch")
	}
}

func TestMergeCoordinationRingArgumentOrderInvariance(t *testing.T) {
	sub, alpha, beta, gamma := formThreeCoordinationRings(t)
	ctx := context.Background()
	rep1, _ := MergeCoordinationRing(ctx, sub, CoordinationRingMergeOptions{
		AntecedentAFormationHash: alpha,
		AntecedentBFormationHash: beta,
		ProducedFormationHash:    gamma,
		MergedAt:                 1716120000000000000,
	}, nil)
	rep2, _ := MergeCoordinationRing(ctx, sub, CoordinationRingMergeOptions{
		AntecedentAFormationHash: beta, // swapped
		AntecedentBFormationHash: alpha,
		ProducedFormationHash:    gamma,
		MergedAt:                 1716120000000000000,
	}, nil)
	if rep1.MergeEventHashHex != rep2.MergeEventHashHex {
		t.Errorf("argument-order invariance violated")
	}
	if !rep2.AlreadyMerged {
		t.Errorf("swapped-args invocation should report AlreadyMerged")
	}
}

func TestMergeCoordinationRingIdenticalAntecedents(t *testing.T) {
	sub, alpha, _, gamma := formThreeCoordinationRings(t)
	_, err := MergeCoordinationRing(context.Background(), sub, CoordinationRingMergeOptions{
		AntecedentAFormationHash: alpha,
		AntecedentBFormationHash: alpha,
		ProducedFormationHash:    gamma,
	}, nil)
	if !errors.Is(err, ErrMergeAntecedentsIdentical) {
		t.Errorf("expected ErrMergeAntecedentsIdentical; got %v", err)
	}
}

func TestMergeCoordinationRingIdempotent(t *testing.T) {
	sub, alpha, beta, gamma := formThreeCoordinationRings(t)
	opts := CoordinationRingMergeOptions{
		AntecedentAFormationHash: alpha,
		AntecedentBFormationHash: beta,
		ProducedFormationHash:    gamma,
		MergedAt:                 1716120000000000000,
		Reason:                   "idempotent",
	}
	ctx := context.Background()
	rep1, _ := MergeCoordinationRing(ctx, sub, opts, nil)
	rep2, _ := MergeCoordinationRing(ctx, sub, opts, nil)
	if !rep2.AlreadyMerged {
		t.Errorf("second invocation should report AlreadyMerged")
	}
	if rep1.MergeEventHashHex != rep2.MergeEventHashHex {
		t.Errorf("idempotency violated")
	}
}

func TestMergeCoordinationRingUnknownAntecedent(t *testing.T) {
	sub, alpha, _, gamma := formThreeCoordinationRings(t)
	var bogus [32]byte
	for i := range bogus {
		bogus[i] = byte(0xff - i)
	}
	_, err := MergeCoordinationRing(context.Background(), sub, CoordinationRingMergeOptions{
		AntecedentAFormationHash: alpha,
		AntecedentBFormationHash: bogus,
		ProducedFormationHash:    gamma,
	}, nil)
	if !errors.Is(err, ErrTargetNotFound) {
		t.Errorf("expected ErrTargetNotFound; got %v", err)
	}
}

func TestMergeCoordinationRingRejectsCrossSubtype(t *testing.T) {
	// A BehavioralClusterFormation hash MUST be rejected as
	// produced (or antecedent) by MergeCoordinationRing. Tests
	// cross-substrate behavior (BC formation not in CR substrate).
	crSub, alpha, _, gamma := formThreeCoordinationRings(t)
	bcSub, bcFormation := formAndCollect(t)
	_ = bcSub
	_, err := MergeCoordinationRing(context.Background(), crSub, CoordinationRingMergeOptions{
		AntecedentAFormationHash: alpha,
		AntecedentBFormationHash: bcFormation, // not in crSub
		ProducedFormationHash:    gamma,
	}, nil)
	if !errors.Is(err, ErrTargetNotFound) {
		t.Errorf("expected ErrTargetNotFound (cross-substrate); got %v", err)
	}
}

func TestMergeCoordinationRingDefaultMergedAt(t *testing.T) {
	sub, alpha, beta, gamma := formThreeCoordinationRings(t)
	fixedNow := func() time.Time { return time.Unix(0, 9999999999) }
	_, err := MergeCoordinationRing(context.Background(), sub, CoordinationRingMergeOptions{
		AntecedentAFormationHash: alpha,
		AntecedentBFormationHash: beta,
		ProducedFormationHash:    gamma,
	}, fixedNow)
	if err != nil {
		t.Fatalf("MergeCoordinationRing: %v", err)
	}
	merges := walkCoordinationRingMerges(t, sub)
	if len(merges) != 1 {
		t.Fatalf("expected 1 merge; got %d", len(merges))
	}
	if merges[0].MergedAt != 9999999999 {
		t.Errorf("merged_at: got %d", merges[0].MergedAt)
	}
}
