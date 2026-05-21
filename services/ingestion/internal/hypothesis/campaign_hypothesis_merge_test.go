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

// formThreeCampaignHypotheses populates a substrate with three
// distinct CampaignHypothesisFormations (alpha, beta, gamma) for
// merge-test wiring. Each campaign has a distinct
// session_descriptor with 3 events.
func formThreeCampaignHypotheses(t *testing.T) (*substrate.Substrate, [32]byte, [32]byte, [32]byte) {
	t.Helper()
	gap := int64(60 * 1e9)
	sub := campaignSubstrate(t, []struct {
		Descriptor []byte
		DeclaredAt int64
	}{
		{[]byte("alpha-camp"), 1000}, {[]byte("alpha-camp"), 1000 + gap}, {[]byte("alpha-camp"), 1000 + 2*gap},
		{[]byte("beta-camp"), 1000000}, {[]byte("beta-camp"), 1000000 + gap}, {[]byte("beta-camp"), 1000000 + 2*gap},
		{[]byte("gamma-camp"), 2000000}, {[]byte("gamma-camp"), 2000000 + gap}, {[]byte("gamma-camp"), 2000000 + 2*gap},
	})

	if _, err := FormCampaignHypothesisAll(context.Background(), sub,
		TemporalDescriptorCohortV1{MinCampaignSize: 3, MaxIntraEventGapSeconds: 300},
		nil); err != nil {
		t.Fatalf("FormCampaignHypothesisAll: %v", err)
	}

	type entry struct {
		hash [32]byte
		ev   *eventsv1.CampaignHypothesisFormation
	}
	var entries []entry
	if err := sub.WalkEvents(context.Background(), func(row substrate.EventRow) error {
		if row.MessageType != campaignHypothesisFormationMessageType {
			return nil
		}
		payload, err := sub.ReadBlob(context.Background(), row.EventHash)
		if err != nil {
			return err
		}
		ev := &eventsv1.CampaignHypothesisFormation{}
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
	// Sort entries by formation_at ascending → alpha (earliest),
	// beta (middle), gamma (latest).
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].ev.FormationAt < entries[j].ev.FormationAt
	})
	return sub, entries[0].hash, entries[1].hash, entries[2].hash
}

func walkCampaignMerges(t *testing.T, sub *substrate.Substrate) []*eventsv1.CampaignHypothesisMerge {
	t.Helper()
	ctx := context.Background()
	var out []*eventsv1.CampaignHypothesisMerge
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != "ghosttrace.events.v1.CampaignHypothesisMerge" {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return err
		}
		ev := &eventsv1.CampaignHypothesisMerge{}
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

func TestMergeCampaignHypothesisHappyPath(t *testing.T) {
	sub, alpha, beta, gamma := formThreeCampaignHypotheses(t)
	rep, err := MergeCampaignHypothesis(context.Background(), sub, CampaignHypothesisMergeOptions{
		AntecedentAFormationHash: alpha,
		AntecedentBFormationHash: beta,
		ProducedFormationHash:    gamma,
		MergedAt:                 1716120000000000000,
		Reason:                   "same campaign",
	}, nil)
	if err != nil {
		t.Fatalf("MergeCampaignHypothesis: %v", err)
	}
	if rep.AlreadyMerged {
		t.Errorf("unexpected AlreadyMerged")
	}
	merges := walkCampaignMerges(t, sub)
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

func TestMergeCampaignHypothesisArgumentOrderInvariance(t *testing.T) {
	sub, alpha, beta, gamma := formThreeCampaignHypotheses(t)
	ctx := context.Background()
	rep1, _ := MergeCampaignHypothesis(ctx, sub, CampaignHypothesisMergeOptions{
		AntecedentAFormationHash: alpha,
		AntecedentBFormationHash: beta,
		ProducedFormationHash:    gamma,
		MergedAt:                 1716120000000000000,
	}, nil)
	rep2, _ := MergeCampaignHypothesis(ctx, sub, CampaignHypothesisMergeOptions{
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

func TestMergeCampaignHypothesisIdenticalAntecedents(t *testing.T) {
	sub, alpha, _, gamma := formThreeCampaignHypotheses(t)
	_, err := MergeCampaignHypothesis(context.Background(), sub, CampaignHypothesisMergeOptions{
		AntecedentAFormationHash: alpha,
		AntecedentBFormationHash: alpha,
		ProducedFormationHash:    gamma,
	}, nil)
	if !errors.Is(err, ErrMergeAntecedentsIdentical) {
		t.Errorf("expected ErrMergeAntecedentsIdentical; got %v", err)
	}
}

func TestMergeCampaignHypothesisIdempotent(t *testing.T) {
	sub, alpha, beta, gamma := formThreeCampaignHypotheses(t)
	opts := CampaignHypothesisMergeOptions{
		AntecedentAFormationHash: alpha,
		AntecedentBFormationHash: beta,
		ProducedFormationHash:    gamma,
		MergedAt:                 1716120000000000000,
		Reason:                   "idempotent",
	}
	ctx := context.Background()
	rep1, _ := MergeCampaignHypothesis(ctx, sub, opts, nil)
	rep2, _ := MergeCampaignHypothesis(ctx, sub, opts, nil)
	if !rep2.AlreadyMerged {
		t.Errorf("second invocation should report AlreadyMerged")
	}
	if rep1.MergeEventHashHex != rep2.MergeEventHashHex {
		t.Errorf("idempotency violated")
	}
}

func TestMergeCampaignHypothesisUnknownAntecedent(t *testing.T) {
	sub, alpha, _, gamma := formThreeCampaignHypotheses(t)
	var bogus [32]byte
	for i := range bogus {
		bogus[i] = byte(0xff - i)
	}
	_, err := MergeCampaignHypothesis(context.Background(), sub, CampaignHypothesisMergeOptions{
		AntecedentAFormationHash: alpha,
		AntecedentBFormationHash: bogus,
		ProducedFormationHash:    gamma,
	}, nil)
	if !errors.Is(err, ErrTargetNotFound) {
		t.Errorf("expected ErrTargetNotFound; got %v", err)
	}
}

func TestMergeCampaignHypothesisRejectsCrossSubtype(t *testing.T) {
	// A BehavioralClusterFormation hash MUST be rejected as
	// produced (or antecedent) by MergeCampaignHypothesis. Tests
	// cross-substrate behavior (BC formation not in CH substrate).
	chSub, alpha, _, gamma := formThreeCampaignHypotheses(t)
	bcSub, bcFormation := formAndCollect(t)
	_ = bcSub
	_, err := MergeCampaignHypothesis(context.Background(), chSub, CampaignHypothesisMergeOptions{
		AntecedentAFormationHash: alpha,
		AntecedentBFormationHash: bcFormation, // not in chSub
		ProducedFormationHash:    gamma,
	}, nil)
	if !errors.Is(err, ErrTargetNotFound) {
		t.Errorf("expected ErrTargetNotFound (cross-substrate); got %v", err)
	}
}

func TestMergeCampaignHypothesisDefaultMergedAt(t *testing.T) {
	sub, alpha, beta, gamma := formThreeCampaignHypotheses(t)
	fixedNow := func() time.Time { return time.Unix(0, 9999999999) }
	_, err := MergeCampaignHypothesis(context.Background(), sub, CampaignHypothesisMergeOptions{
		AntecedentAFormationHash: alpha,
		AntecedentBFormationHash: beta,
		ProducedFormationHash:    gamma,
	}, fixedNow)
	if err != nil {
		t.Fatalf("MergeCampaignHypothesis: %v", err)
	}
	merges := walkCampaignMerges(t, sub)
	if len(merges) != 1 {
		t.Fatalf("expected 1 merge; got %d", len(merges))
	}
	if merges[0].MergedAt != 9999999999 {
		t.Errorf("merged_at: got %d", merges[0].MergedAt)
	}
}
