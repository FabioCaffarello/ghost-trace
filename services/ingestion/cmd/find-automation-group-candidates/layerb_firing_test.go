// Integration test exercising Layer B firing against AutomationGroup
// formation events produced from F3 candidate data per §0157. Extends
// the §0156 baseline (which exercised F1 ingestion → F3 signature →
// candidate emission) with the operator-elected steps that follow:
// operator commits formation event using candidate data → operator
// promotes → Layer B evaluates per §0141 + §0136 + §0138.
//
// This is the first test exercising Layer B against AutomationGroup
// formation events derived end-to-end from F3 signature output (not
// from hand-built test fixtures). Validates that the candidate-to-
// formation mapping is structurally sound + Layer B accepts the
// resulting substrate state.
//
// Scope per §0157: structurally validate the full path. Layer B
// VERDICT semantics under various N_window + threshold configurations
// are covered by the existing layerb package test suite; this test
// validates only that the path connects without structural failure.
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

// commitFormationFromCandidate materializes an AutomationGroupFormation
// proto from an F3 signature candidate + commits it to substrate. This
// is the operator-elected step that follows candidate review per §0152
// + §0153 discipline (operator decides to commit; orchestrator does not
// auto-commit per §3 N3).
//
// Mirrors what an operator running `form-automation-group` CLI would
// do at substrate level: construct the proto, marshal, commit. The
// test path bypasses the CLI argument-parsing layer but uses the same
// substrate API (canonical.MarshalAndHash + substrate.Append).
func commitFormationFromCandidate(t *testing.T, sub *substrate.Substrate, c *signatures.FormationCandidate, formationAt int64) [32]byte {
	t.Helper()
	ctx := context.Background()

	// Build formation proto. Per §0152 + §0139 hash-list discipline:
	// source_event_hashes inherits the candidate's SourceHashes (already
	// sorted ascending by signature emit). direct_influenced_by is empty
	// (this formation is a root in the Cat III influenced-by graph —
	// the signature did not consume prior Cat III hypotheses).
	formation := &eventsv1.AutomationGroupFormation{
		PatternSignature:   c.SignatureName,
		PatternParameters:  "threshold=2",
		ActorRefs:          c.ActorRefs,
		FormationAt:        formationAt,
		SourceEventHashes:  c.SourceHashes,
		DirectInfluencedBy: nil,
		ClosureHashes:      nil,
		// Per §2.6 BC3 paired-dimension enforcement: confidence +
		// evidential_independence required at marshalling boundary.
		// Confidence inherits the signature's ConfidenceHint per §0152
		// (advisory pattern); evidential_independence computed as a
		// trivial rational pair for this integration test (1/1 =
		// "fully independent" given single Cat I source class with
		// no prior Cat III influence).
		Confidence: float32(c.ConfidenceHint),
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
// (T_B = K_C = 1/2; N_window = passed value). Mirrors halfHalf helper
// in layerb_test.go (kept here to avoid cross-test-package import).
func halfHalfParams(n uint64) *commonv1.LayerBParameters {
	return &commonv1.LayerBParameters{
		TB:                    &commonv1.EvidentialIndependence{Numerator: 1, Denominator: 2},
		KC:                    &commonv1.EvidentialIndependence{Numerator: 1, Denominator: 2},
		NWindow:               n,
		NADurationNanoseconds: 86400000000000, // 1 day
	}
}

// TestLayerBFiring_AgainstF3CandidateFormation exercises the path
// from F3 signature candidate output through operator-elected
// formation commit to Layer B evaluation. Verifies all stages connect
// structurally without failure.
//
// Scope: connectivity verification. Layer B verdict semantics (when
// freshness_B fires, when saturation_C fires, threshold-crossing
// behavior under various N_window sizes) are covered by the existing
// layerb package test suite; this test does not duplicate that
// coverage.
func TestLayerBFiring_AgainstF3CandidateFormation(t *testing.T) {
	sub, in := newLayerBFiringSubstrate(t)
	ctx := context.Background()

	// Step 1: Inject BrowserObservation with CDP markers above
	// signature threshold (5 detections aggregated → above default
	// threshold 2).
	appendBrowserObs(t, in, "actor-suspect", []string{"navigator.webdriver=true"}, 3)
	appendBrowserObs(t, in, "actor-suspect", []string{"$cdc_test"}, 2)

	// Step 2: Run F3 signature → expect 1 candidate for actor-suspect.
	observations, err := collectBrowserObservations(ctx, sub)
	if err != nil {
		t.Fatalf("collectBrowserObservations: %v", err)
	}
	if got, want := len(observations), 2; got != want {
		t.Fatalf("observations count: got %d want %d", got, want)
	}
	sig := &signatures.CDPMarkerDensityV1{}
	result, err := sig.EvaluateBrowser(ctx, observations)
	if err != nil {
		t.Fatalf("EvaluateBrowser: %v", err)
	}
	if got, want := len(result.Candidates), 1; got != want {
		t.Fatalf("candidates count: got %d want %d", got, want)
	}
	candidate := result.Candidates[0]
	if candidate.ActorRefs[0] != "actor-suspect" {
		t.Errorf("candidate actor: got %q want actor-suspect", candidate.ActorRefs[0])
	}

	// Step 3: Operator-elected formation commit. Uses candidate output
	// to build AutomationGroupFormation; commits via substrate.Append.
	formationAt := int64(1716120100000000000)
	formationHash := commitFormationFromCandidate(t, sub, candidate, formationAt)

	// Step 4: Verify the formation event lands in substrate by reading
	// it back via LookupRow.
	row, err := sub.LookupRow(ctx, formationHash)
	if err != nil {
		t.Fatalf("LookupRow formation: %v", err)
	}
	if row.MessageType != "ghosttrace.events.v1.AutomationGroupFormation" {
		t.Errorf("formation MessageType: got %q want AutomationGroupFormation", row.MessageType)
	}

	// Step 5: Unmarshal + verify formation proto matches candidate
	// values fed in. This validates the candidate-to-formation mapping
	// preserves the payload we expect Layer B to consume.
	payload, err := sub.ReadBlob(ctx, formationHash)
	if err != nil {
		t.Fatalf("ReadBlob formation: %v", err)
	}
	gotFormation := &eventsv1.AutomationGroupFormation{}
	if err := proto.Unmarshal(payload, gotFormation); err != nil {
		t.Fatalf("Unmarshal formation: %v", err)
	}
	if gotFormation.PatternSignature != "cdp_marker_density_v1" {
		t.Errorf("formation PatternSignature: got %q want cdp_marker_density_v1", gotFormation.PatternSignature)
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
	// substrate state (5 events: 2 BrowserObservation + 2 paired
	// IngestionEvent + 1 formation). FilterMatchCount may be 0 (the
	// closure_hashes filter selects events where formation's hash is
	// in the event's closure_hashes; no committed event has this
	// formation's hash in its closure_hashes yet).
	if verdict.WindowEventCount == 0 {
		t.Error("verdict WindowEventCount is 0; expected substrate events to be walked")
	}
	t.Logf("Layer B verdict: WindowEventCount=%d FilterMatchCount=%d FreshnessUndefined=%v FreshnessFired=%v SaturationFired=%v Fired=%v",
		verdict.WindowEventCount, verdict.FilterMatchCount,
		verdict.FreshnessUndefined, verdict.FreshnessFired,
		verdict.SaturationFired, verdict.Fired)
}

// TestLayerBFiring_RejectsNilParams confirms layerb.Evaluate surfaces
// a structural error when called without LayerBParameters. Documents
// the integration-path contract: the orchestrator MUST supply params
// (typically from the promotion event's bundled LayerBParameters per
// §0138 N_A bundling) before invoking Evaluate.
func TestLayerBFiring_RejectsNilParams(t *testing.T) {
	sub, _ := newLayerBFiringSubstrate(t)
	ctx := context.Background()
	var formationHash [32]byte // zero hash; Evaluate validates params before hash
	_, err := layerb.Evaluate(ctx, sub, layerb.EvaluateOptions{
		FormationEventHash: formationHash,
		Params:             nil,
	})
	if err == nil {
		t.Fatal("expected error for nil Params, got nil")
	}
}
