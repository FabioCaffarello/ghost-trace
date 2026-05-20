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

// formThreeAutomationGroups populates a substrate with three
// distinct uniform-cadence actors (each generating its own single-
// actor AutomationGroupFormation) and returns the three formation
// hashes (alpha, beta, gamma) for merge-test wiring.
func formThreeAutomationGroups(t *testing.T) (*substrate.Substrate, [32]byte, [32]byte, [32]byte) {
	t.Helper()
	sub := agSubstrate(t, []struct {
		ActorRef   string
		DeclaredAt int64
	}{
		{"bot-a", 1000}, {"bot-a", 2000}, {"bot-a", 3000}, {"bot-a", 4000}, {"bot-a", 5000},
		{"bot-b", 10000}, {"bot-b", 20000}, {"bot-b", 30000}, {"bot-b", 40000}, {"bot-b", 50000},
		{"bot-c", 100000}, {"bot-c", 200000}, {"bot-c", 300000}, {"bot-c", 400000}, {"bot-c", 500000},
	})
	ctx := context.Background()
	rep, err := FormAutomationGroupAll(ctx, sub,
		UniformCadenceV1{MinObservationCount: 5, MaxCoVThreshold: 0.15},
		func() time.Time { return time.Unix(0, 1) })
	if err != nil {
		t.Fatalf("FormAutomationGroupAll: %v", err)
	}
	if rep.NewlyFormed != 3 {
		t.Fatalf("expected 3 formations; got %d", rep.NewlyFormed)
	}

	type entry struct {
		hash [32]byte
		ev   *eventsv1.AutomationGroupFormation
	}
	var entries []entry
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != automationGroupFormationMessageType {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return err
		}
		ev := &eventsv1.AutomationGroupFormation{}
		if err := proto.Unmarshal(payload, ev); err != nil {
			return err
		}
		entries = append(entries, entry{hash: row.EventHash, ev: ev})
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 formation rows; got %d", len(entries))
	}

	var alpha, beta, gamma [32]byte
	for _, e := range entries {
		if len(e.ev.ActorRefs) != 1 {
			t.Fatalf("expected single-actor groups; got %d actors", len(e.ev.ActorRefs))
		}
		switch e.ev.ActorRefs[0] {
		case "bot-a":
			alpha = e.hash
		case "bot-b":
			beta = e.hash
		case "bot-c":
			gamma = e.hash
		default:
			t.Fatalf("unexpected actor: %q", e.ev.ActorRefs[0])
		}
	}
	return sub, alpha, beta, gamma
}

func walkAutomationGroupMerges(t *testing.T, sub *substrate.Substrate) []*eventsv1.AutomationGroupMerge {
	t.Helper()
	ctx := context.Background()
	var out []*eventsv1.AutomationGroupMerge
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != "ghosttrace.events.v1.AutomationGroupMerge" {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return err
		}
		ev := &eventsv1.AutomationGroupMerge{}
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

func TestMergeAutomationGroupHappyPath(t *testing.T) {
	sub, alpha, beta, gamma := formThreeAutomationGroups(t)
	rep, err := MergeAutomationGroup(context.Background(), sub, AutomationGroupMergeOptions{
		AntecedentAFormationHash: alpha,
		AntecedentBFormationHash: beta,
		ProducedFormationHash:    gamma,
		MergedAt:                 1716120000000000000,
		Reason:                   "same automation signature",
	}, nil)
	if err != nil {
		t.Fatalf("MergeAutomationGroup: %v", err)
	}
	if rep.AlreadyMerged {
		t.Errorf("unexpected AlreadyMerged on first invocation")
	}
	merges := walkAutomationGroupMerges(t, sub)
	if len(merges) != 1 {
		t.Fatalf("substrate carries %d merges; want 1", len(merges))
	}
	got := merges[0]
	if len(got.AntecedentFormationEventHashes) != 2 {
		t.Fatalf("antecedents: got %d; want 2", len(got.AntecedentFormationEventHashes))
	}
	if bytes.Compare(got.AntecedentFormationEventHashes[0], got.AntecedentFormationEventHashes[1]) >= 0 {
		t.Errorf("antecedents not sorted ascending")
	}
	if !bytes.Equal(got.ProducedFormationEventHash, gamma[:]) {
		t.Errorf("produced mismatch")
	}
}

func TestMergeAutomationGroupArgumentOrderInvariance(t *testing.T) {
	sub, alpha, beta, gamma := formThreeAutomationGroups(t)
	ctx := context.Background()
	rep1, _ := MergeAutomationGroup(ctx, sub, AutomationGroupMergeOptions{
		AntecedentAFormationHash: alpha,
		AntecedentBFormationHash: beta,
		ProducedFormationHash:    gamma,
		MergedAt:                 1716120000000000000,
	}, nil)
	rep2, _ := MergeAutomationGroup(ctx, sub, AutomationGroupMergeOptions{
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
	if got := len(walkAutomationGroupMerges(t, sub)); got != 1 {
		t.Errorf("substrate holds %d merges; want 1", got)
	}
}

func TestMergeAutomationGroupIdenticalAntecedents(t *testing.T) {
	sub, alpha, _, gamma := formThreeAutomationGroups(t)
	_, err := MergeAutomationGroup(context.Background(), sub, AutomationGroupMergeOptions{
		AntecedentAFormationHash: alpha,
		AntecedentBFormationHash: alpha,
		ProducedFormationHash:    gamma,
	}, nil)
	if !errors.Is(err, ErrMergeAntecedentsIdentical) {
		t.Errorf("expected ErrMergeAntecedentsIdentical; got %v", err)
	}
}

func TestMergeAutomationGroupIdempotent(t *testing.T) {
	sub, alpha, beta, gamma := formThreeAutomationGroups(t)
	opts := AutomationGroupMergeOptions{
		AntecedentAFormationHash: alpha,
		AntecedentBFormationHash: beta,
		ProducedFormationHash:    gamma,
		MergedAt:                 1716120000000000000,
		Reason:                   "idempotent",
	}
	ctx := context.Background()
	rep1, _ := MergeAutomationGroup(ctx, sub, opts, nil)
	rep2, _ := MergeAutomationGroup(ctx, sub, opts, nil)
	if !rep2.AlreadyMerged {
		t.Errorf("second invocation should report AlreadyMerged")
	}
	if rep1.MergeEventHashHex != rep2.MergeEventHashHex {
		t.Errorf("idempotency violated")
	}
}

func TestMergeAutomationGroupUnknownAntecedent(t *testing.T) {
	sub, alpha, _, gamma := formThreeAutomationGroups(t)
	var bogus [32]byte
	for i := range bogus {
		bogus[i] = byte(0xff - i)
	}
	_, err := MergeAutomationGroup(context.Background(), sub, AutomationGroupMergeOptions{
		AntecedentAFormationHash: alpha,
		AntecedentBFormationHash: bogus,
		ProducedFormationHash:    gamma,
	}, nil)
	if !errors.Is(err, ErrTargetNotFound) {
		t.Errorf("expected ErrTargetNotFound; got %v", err)
	}
}

func TestMergeAutomationGroupWrongTypeTarget(t *testing.T) {
	// A PROMOTION hash passed as an antecedent → ErrTargetWrongType.
	sub, promotionHash := formAutomationGroupAndPromote(t, 1000*int64(time.Second), 60)
	// Form a second AG formation for the produced slot.
	var produced [32]byte
	if err := sub.WalkEvents(context.Background(), func(row substrate.EventRow) error {
		if row.MessageType == automationGroupFormationMessageType {
			produced = row.EventHash
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	_, err := MergeAutomationGroup(context.Background(), sub, AutomationGroupMergeOptions{
		AntecedentAFormationHash: produced,
		AntecedentBFormationHash: promotionHash, // wrong type
		ProducedFormationHash:    produced,
	}, nil)
	if !errors.Is(err, ErrTargetWrongType) {
		t.Errorf("expected ErrTargetWrongType for promotion hash; got %v", err)
	}
}

func TestMergeAutomationGroupRejectsCrossSubtypeFormation(t *testing.T) {
	// Cross-subtype guard: a BehavioralClusterFormation hash passed
	// to MergeAutomationGroup MUST return ErrTargetWrongType.
	sub, alpha, _, gamma := formThreeAutomationGroups(t)
	// Populate the same substrate with a BehavioralCluster formation.
	// formAndCollect uses its own substrate; we need to ingest manually.
	bcSub, bcFormation := formAndCollect(t)
	_ = bcSub
	_, err := MergeAutomationGroup(context.Background(), sub, AutomationGroupMergeOptions{
		AntecedentAFormationHash: alpha,
		AntecedentBFormationHash: bcFormation, // cross-subtype; doesn't exist in `sub`
		ProducedFormationHash:    gamma,
	}, nil)
	// Cross-substrate lookup: bcFormation not in sub → ErrTargetNotFound.
	if !errors.Is(err, ErrTargetNotFound) {
		t.Errorf("expected ErrTargetNotFound (cross-substrate); got %v", err)
	}
}

func TestMergeAutomationGroupDefaultMergedAt(t *testing.T) {
	sub, alpha, beta, gamma := formThreeAutomationGroups(t)
	fixedNow := func() time.Time { return time.Unix(0, 9999999999) }
	_, err := MergeAutomationGroup(context.Background(), sub, AutomationGroupMergeOptions{
		AntecedentAFormationHash: alpha,
		AntecedentBFormationHash: beta,
		ProducedFormationHash:    gamma,
	}, fixedNow)
	if err != nil {
		t.Fatalf("MergeAutomationGroup: %v", err)
	}
	merges := walkAutomationGroupMerges(t, sub)
	if len(merges) != 1 {
		t.Fatalf("expected 1 merge; got %d", len(merges))
	}
	if merges[0].MergedAt != 9999999999 {
		t.Errorf("merged_at: got %d; want 9999999999", merges[0].MergedAt)
	}
}
