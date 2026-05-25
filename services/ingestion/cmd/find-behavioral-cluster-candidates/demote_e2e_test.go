// End-to-end integration test exercising the full F3 → formation →
// promotion → demotion lifecycle arc for BehavioralCluster per §0178.
// Mirrors §0160's TestDemoteAutomationGroup_FromF3CandidateLifecycle
// on the BehavioralCluster subtype side. Third PR in the §0176-opened
// BehavioralCluster lifecycle integration coverage arc (extends along
// the LINEAR LIFECYCLE axis: formation → promotion → demotion).
//
// Scope per §0178: structural connectivity validation across the
// full §0011 staged-combination lifecycle arc at BehavioralCluster
// subtype level. Per-operation semantics (validation errors,
// cadence-gate behavior, layerb verdict math) covered by hypothesis
// + layerb package unit test suites; this test validates only that
// the full arc connects end-to-end from F3 candidate output.
package main

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/hypothesis"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/signatures"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/observationcollector"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

func newDemoteE2ESubstrate(t *testing.T) (*substrate.Substrate, *ingest.Ingester) {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })
	clock := func() time.Time { return time.Unix(0, 1716120000000000777) }
	return sub, ingest.New(sub, clock)
}

// TestDemoteBehavioralCluster_FromF3CandidateLifecycle exercises the
// full F3 → formation → promotion → demotion lifecycle arc for
// BehavioralCluster:
//
//	Step 1: Inject 3 BehavioralObservation with identical quantized
//	        keystroke fingerprint (above default threshold 3).
//	Step 2: Run F3 signature → 1 multi-actor BehavioralCluster candidate.
//	Step 3: Operator-elected formation commit via §0176 helper.
//	Step 4: Operator-elected promotion commit with LayerBParameters.
//	Step 5: Operator-elected demotion commit (advisory Layer B verdict
//	        captured in report per §0141 E1).
//	Step 6: Verify formation + promotion + demotion all in substrate.
//	Step 7: Verify demotion proto references promotion event hash.
//	Step 8: Verify Layer A cadence state + Layer B verdict captured.
//
// Mirrors §0160 on the BehavioralCluster subtype side using the
// generic-named hypothesis.Promote / hypothesis.Demote functions
// (which target BehavioralCluster per the package's generic-name
// discipline for the second subtype landing).
func TestDemoteBehavioralCluster_FromF3CandidateLifecycle(t *testing.T) {
	sub, in := newDemoteE2ESubstrate(t)
	ctx := context.Background()

	// Step 1: Inject above-threshold BehavioralObservation records.
	ivs := []uint64{50_000_000, 100_000_000, 0, 100_000_000, 100_000_000, 150_000_000}
	appendKeystrokeObs(t, in, "actor-suspect-1", ivs, 1)
	appendKeystrokeObs(t, in, "actor-suspect-2", ivs, 2)
	appendKeystrokeObs(t, in, "actor-suspect-3", ivs, 3)

	// Step 2: Run F3 signature → 1 multi-actor candidate.
	observations, err := observationcollector.CollectBehavioral(ctx, sub)
	if err != nil {
		t.Fatalf("observationcollector.CollectBehavioral: %v", err)
	}
	sig := &signatures.KeystrokeTimingClusteringV1{}
	result, err := sig.EvaluateBehavioral(ctx, observations)
	if err != nil {
		t.Fatalf("EvaluateBehavioral: %v", err)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("candidates count: got %d want 1", len(result.Candidates))
	}
	candidate := result.Candidates[0]
	if candidate.HypothesisSubtype != signatures.HypothesisSubtypeBehavioralCluster {
		t.Errorf("candidate subtype: got %v want BehavioralCluster", candidate.HypothesisSubtype)
	}

	// Step 3: Operator-elected formation commit via §0176 helper.
	formationAt := int64(1716120100000000000)
	formationHash := commitBehavioralClusterFormationFromCandidate(t, sub, candidate, formationAt)

	// Step 4: Operator-elected promotion commit. Uses the generic
	// hypothesis.Promote (BehavioralCluster-targeted; AutomationGroup
	// uses hypothesis.PromoteAutomationGroup per separate landing).
	promotedAt := int64(1716120200000000000)
	layerBParams := &commonv1.LayerBParameters{
		TB:                    &commonv1.EvidentialIndependence{Numerator: 1, Denominator: 2},
		KC:                    &commonv1.EvidentialIndependence{Numerator: 1, Denominator: 2},
		NWindow:               100,
		NADurationNanoseconds: 86400000000000,
	}
	now := func() time.Time { return time.Unix(0, promotedAt) }
	promoteReport, err := hypothesis.Promote(ctx, sub, hypothesis.PromoteOptions{
		FormationEventHash: formationHash,
		PromotedAt:         promotedAt,
		CadenceSeconds:     3600,
		Reason:             "e2e test promotion (BehavioralCluster)",
		LayerBParameters:   layerBParams,
	}, now)
	if err != nil {
		t.Fatalf("hypothesis.Promote: %v", err)
	}
	if promoteReport.PromotionEventHashHex == "" {
		t.Fatal("PromoteReport.PromotionEventHashHex is empty")
	}
	var promotionHash [32]byte
	if _, err := hexDecodeInto(promoteReport.PromotionEventHashHex, promotionHash[:]); err != nil {
		t.Fatalf("decode promotion hash hex: %v", err)
	}

	// Step 5: Operator-elected demotion commit. Layer B advisory per
	// §0141 E1; commits regardless of verdict.
	demotedAt := int64(1716120300000000000)
	nowDemote := func() time.Time { return time.Unix(0, demotedAt) }
	demoteReport, err := hypothesis.Demote(ctx, sub, hypothesis.DemoteOptions{
		PromotionEventHash: promotionHash,
		DemotedAt:          demotedAt,
		Reason:             "e2e test demotion (BehavioralCluster)",
	}, nowDemote)
	if err != nil {
		t.Fatalf("hypothesis.Demote: %v", err)
	}
	if demoteReport.DemotionEventHashHex == "" {
		t.Fatal("DemoteReport.DemotionEventHashHex is empty")
	}

	// Step 6: Verify all three lifecycle events in substrate.
	formationRow, err := sub.LookupRow(ctx, formationHash)
	if err != nil {
		t.Fatalf("LookupRow formation: %v", err)
	}
	if formationRow.MessageType != "ghosttrace.events.v1.BehavioralClusterFormation" {
		t.Errorf("formation MessageType: got %q want BehavioralClusterFormation", formationRow.MessageType)
	}
	promotionRow, err := sub.LookupRow(ctx, promotionHash)
	if err != nil {
		t.Fatalf("LookupRow promotion: %v", err)
	}
	if promotionRow.MessageType != "ghosttrace.events.v1.BehavioralClusterPromotion" {
		t.Errorf("promotion MessageType: got %q want BehavioralClusterPromotion", promotionRow.MessageType)
	}
	var demotionHash [32]byte
	if _, err := hexDecodeInto(demoteReport.DemotionEventHashHex, demotionHash[:]); err != nil {
		t.Fatalf("decode demotion hash hex: %v", err)
	}
	demotionRow, err := sub.LookupRow(ctx, demotionHash)
	if err != nil {
		t.Fatalf("LookupRow demotion: %v", err)
	}
	if demotionRow.MessageType != "ghosttrace.events.v1.BehavioralClusterDemotion" {
		t.Errorf("demotion MessageType: got %q want BehavioralClusterDemotion", demotionRow.MessageType)
	}

	// Step 7: Verify demotion proto references promotion event hash.
	demotionPayload, err := sub.ReadBlob(ctx, demotionHash)
	if err != nil {
		t.Fatalf("ReadBlob demotion: %v", err)
	}
	gotDemotion := &eventsv1.BehavioralClusterDemotion{}
	if err := proto.Unmarshal(demotionPayload, gotDemotion); err != nil {
		t.Fatalf("Unmarshal demotion: %v", err)
	}
	if !bytes.Equal(gotDemotion.PromotionEventHash, promotionHash[:]) {
		t.Error("demotion PromotionEventHash does not match promotion hash")
	}
	if gotDemotion.Reason == "" {
		t.Error("demotion Reason: got empty want non-empty")
	}

	// Step 8: Verify Layer A cadence + Layer B verdict captured.
	// CadenceSatisfied=false because demoted within cadence (100s <
	// CadenceSeconds=3600s); §0011 Layer A is CANDIDACY criterion,
	// not hard barrier.
	if demoteReport.CadenceSatisfied {
		t.Errorf("CadenceSatisfied: got true want false (demoted within cadence)")
	}
	if demoteReport.CadenceElapsedSeconds != 100 {
		t.Errorf("CadenceElapsedSeconds: got %d want 100", demoteReport.CadenceElapsedSeconds)
	}
	t.Logf("demote E2E (BehavioralCluster): cadence=%v elapsed=%ds LayerB.Fired=%v WindowEventCount=%d FilterMatchCount=%d",
		demoteReport.CadenceSatisfied, demoteReport.CadenceElapsedSeconds,
		demoteReport.LayerB.Fired, demoteReport.LayerB.WindowEventCount,
		demoteReport.LayerB.FilterMatchCount)
}

// hexDecodeInto decodes a hex string into dst (length must be
// 2*len(dst)). Local helper mirroring §0160's equivalent.
func hexDecodeInto(hexStr string, dst []byte) (int, error) {
	if len(hexStr) != 2*len(dst) {
		return 0, &hexLenErr{}
	}
	for i := 0; i < len(dst); i++ {
		hi, lo := hexNibble(hexStr[2*i]), hexNibble(hexStr[2*i+1])
		dst[i] = byte(hi<<4 | lo)
	}
	return len(dst), nil
}

type hexLenErr struct{}

func (e *hexLenErr) Error() string { return "hex length mismatch" }

func hexNibble(c byte) int {
	switch {
	case '0' <= c && c <= '9':
		return int(c - '0')
	case 'a' <= c && c <= 'f':
		return int(c-'a') + 10
	case 'A' <= c && c <= 'F':
		return int(c-'A') + 10
	}
	return 0
}
