// Integration test exercising Layer B firing against BehavioralClusterFormation
// events produced from F3 candidate data per §0176. Mirrors §0157's
// equivalent for AutomationGroup on the BehavioralCluster subtype side.
//
// Establishes the F3-derived-formation anchor for BehavioralCluster:
// the helper commitBehavioralClusterFormationFromCandidate is the
// BehavioralCluster equivalent of §0157's commitFormationFromCandidate
// and is the foundation for subsequent BehavioralCluster lifecycle
// integration tests (demote/promote/merge/dissolve/split E2E mirroring
// the §0157–§0167 AutomationGroup arc).
//
// Scope per §0176: structurally validate the F3 → BehavioralClusterFormation
// → Layer B path. Layer B verdict semantics under various N_window +
// threshold configurations are covered by the existing layerb package
// test suite; this test validates only that the path connects without
// structural failure.
package main

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/hypothesis/layerb"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/signatures"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/observationcollector"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

func newLayerBFiringSubstrate(t *testing.T) (*substrate.Substrate, *ingest.Ingester) {
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

// commitBehavioralClusterFormationFromCandidate materializes a
// BehavioralClusterFormation proto from an F3 signature candidate +
// commits it to substrate. The BehavioralCluster equivalent of
// §0157's commitFormationFromCandidate; foundation for subsequent
// BehavioralCluster lifecycle integration tests per §0176.
//
// Mirrors what an operator running `form-behavioral-cluster` CLI
// would do at substrate level: construct the proto, marshal, commit.
// The test path bypasses the CLI argument-parsing layer but uses
// the same substrate API (canonical.MarshalAndHash + substrate.Append).
//
// Per §0174 BehavioralCluster ontology: actor_refs is the multi-actor
// cluster membership (set of actors operating under a common underlying
// entity). source_event_hashes inherits the candidate's SourceHashes
// (already sorted ascending by signature emit per §0139 hash-list
// element-shape discipline).
func commitBehavioralClusterFormationFromCandidate(t *testing.T, sub *substrate.Substrate, c *signatures.FormationCandidate, formationAt int64) [32]byte {
	t.Helper()
	ctx := context.Background()

	formation := &eventsv1.BehavioralClusterFormation{
		PatternSignature:   c.SignatureName,
		PatternParameters:  "threshold=3",
		ActorRefs:          c.ActorRefs,
		FormationAt:        formationAt,
		SourceEventHashes:  c.SourceHashes,
		DirectInfluencedBy: nil,
		ClosureHashes:      nil,
		Confidence:         float32(c.ConfidenceHint),
		EvidentialIndependence: &commonv1.EvidentialIndependence{
			Numerator:   1,
			Denominator: 1,
		},
	}

	payload, hash, err := canonical.MarshalAndHash(formation)
	if err != nil {
		t.Fatalf("MarshalAndHash formation: %v", err)
	}
	hex := canonical.HashHex(hash)
	row := substrate.EventRow{
		EventHash:   hash,
		EventTime:   formationAt,
		MessageType: string(formation.ProtoReflect().Descriptor().FullName()),
		PayloadRef:  hex[:2] + "/" + hex[2:],
		CommittedAt: formationAt,
	}
	if err := sub.Append(ctx, row, payload); err != nil {
		t.Fatalf("substrate.Append formation: %v", err)
	}
	return hash
}

// halfHalfParams returns §0138 inception-phase Layer B parameters
// (T_B = K_C = 1/2; N_window = passed value). Local copy mirroring
// §0157's helper to keep the test self-contained.
func halfHalfParams(n uint64) *commonv1.LayerBParameters {
	return &commonv1.LayerBParameters{
		TB:                    &commonv1.EvidentialIndependence{Numerator: 1, Denominator: 2},
		KC:                    &commonv1.EvidentialIndependence{Numerator: 1, Denominator: 2},
		NWindow:               n,
		NADurationNanoseconds: 86400000000000, // 1 day
	}
}

// TestLayerBFiring_AgainstBehavioralClusterF3CandidateFormation
// exercises the path from F3 keystroke_timing_clustering_v1 signature
// candidate output through operator-elected BehavioralClusterFormation
// commit to Layer B evaluation. Verifies all stages connect
// structurally without failure.
//
// Mirrors §0157's TestLayerBFiring_AgainstF3CandidateFormation on
// the BehavioralCluster subtype side.
func TestLayerBFiring_AgainstBehavioralClusterF3CandidateFormation(t *testing.T) {
	sub, in := newLayerBFiringSubstrate(t)
	ctx := context.Background()

	// Step 1: Inject 3 BehavioralObservation records for distinct
	// actors with identical quantized keystroke fingerprint. Above
	// default threshold 3.
	ivs := []uint64{50_000_000, 100_000_000, 0, 100_000_000, 100_000_000, 150_000_000}
	appendKeystrokeObs(t, in, "actor-suspect-1", ivs, 1)
	appendKeystrokeObs(t, in, "actor-suspect-2", ivs, 2)
	appendKeystrokeObs(t, in, "actor-suspect-3", ivs, 3)

	// Step 2: Run F3 signature → expect 1 multi-actor candidate.
	observations, err := observationcollector.CollectBehavioral(ctx, sub)
	if err != nil {
		t.Fatalf("observationcollector.CollectBehavioral: %v", err)
	}
	if got, want := len(observations), 3; got != want {
		t.Fatalf("observations count: got %d want %d", got, want)
	}
	sig := &signatures.KeystrokeTimingClusteringV1{}
	result, err := sig.EvaluateBehavioral(ctx, observations)
	if err != nil {
		t.Fatalf("EvaluateBehavioral: %v", err)
	}
	if got, want := len(result.Candidates), 1; got != want {
		t.Fatalf("candidates count: got %d want %d", got, want)
	}
	candidate := result.Candidates[0]
	if candidate.HypothesisSubtype != signatures.HypothesisSubtypeBehavioralCluster {
		t.Errorf("candidate subtype: got %v want BehavioralCluster", candidate.HypothesisSubtype)
	}
	if len(candidate.ActorRefs) != 3 {
		t.Fatalf("candidate ActorRefs count: got %d want 3", len(candidate.ActorRefs))
	}

	// Step 3: Operator-elected BehavioralClusterFormation commit.
	formationAt := int64(1716120100000000000)
	formationHash := commitBehavioralClusterFormationFromCandidate(t, sub, candidate, formationAt)

	// Step 4: Verify the formation event lands in substrate via
	// LookupRow.
	row, err := sub.LookupRow(ctx, formationHash)
	if err != nil {
		t.Fatalf("LookupRow formation: %v", err)
	}
	if row.MessageType != "ghosttrace.events.v1.BehavioralClusterFormation" {
		t.Errorf("formation MessageType: got %q want BehavioralClusterFormation", row.MessageType)
	}

	// Step 5: Unmarshal + verify formation proto matches candidate
	// values. Validates the candidate-to-formation mapping preserves
	// the values we expect Layer B to consume.
	payload, err := sub.ReadBlob(ctx, formationHash)
	if err != nil {
		t.Fatalf("ReadBlob formation: %v", err)
	}
	gotFormation := &eventsv1.BehavioralClusterFormation{}
	if err := proto.Unmarshal(payload, gotFormation); err != nil {
		t.Fatalf("Unmarshal formation: %v", err)
	}
	if gotFormation.PatternSignature != "keystroke_timing_clustering_v1" {
		t.Errorf("formation PatternSignature: got %q want keystroke_timing_clustering_v1", gotFormation.PatternSignature)
	}
	if len(gotFormation.ActorRefs) != 3 {
		t.Errorf("formation ActorRefs count: got %d want 3", len(gotFormation.ActorRefs))
	}
	if len(gotFormation.SourceEventHashes) != int(candidate.EvidenceCount) {
		t.Errorf("formation SourceEventHashes count: got %d want %d", len(gotFormation.SourceEventHashes), candidate.EvidenceCount)
	}
	if gotFormation.GetEvidentialIndependence() == nil {
		t.Error("formation EvidentialIndependence is nil (paired-dimension violation per §2.6 BC3)")
	}

	// Step 6: Call layerb.Evaluate against the formation hash.
	// N_window = 100 (larger than substrate to walk all events).
	verdict, err := layerb.Evaluate(ctx, sub, layerb.EvaluateOptions{
		FormationEventHash: formationHash,
		Params:             halfHalfParams(100),
	})
	if err != nil {
		t.Fatalf("layerb.Evaluate: %v", err)
	}

	// Verdict shape validation: WindowEventCount should reflect
	// substrate state.
	if verdict.WindowEventCount == 0 {
		t.Error("verdict WindowEventCount is 0; expected substrate events to be walked")
	}
	t.Logf("Layer B verdict (BehavioralCluster): WindowEventCount=%d FilterMatchCount=%d FreshnessUndefined=%v FreshnessFired=%v SaturationFired=%v Fired=%v",
		verdict.WindowEventCount, verdict.FilterMatchCount,
		verdict.FreshnessUndefined, verdict.FreshnessFired,
		verdict.SaturationFired, verdict.Fired)
}

// TestLayerBFiring_BehavioralClusterRejectsNilParams confirms
// layerb.Evaluate rejects nil LayerBParameters at the integration
// path for BehavioralCluster (mirrors §0157's equivalent for
// AutomationGroup).
func TestLayerBFiring_BehavioralClusterRejectsNilParams(t *testing.T) {
	sub, _ := newLayerBFiringSubstrate(t)
	ctx := context.Background()
	var formationHash [32]byte
	_, err := layerb.Evaluate(ctx, sub, layerb.EvaluateOptions{
		FormationEventHash: formationHash,
		Params:             nil,
	})
	if err == nil {
		t.Fatal("expected error for nil Params, got nil")
	}
}
