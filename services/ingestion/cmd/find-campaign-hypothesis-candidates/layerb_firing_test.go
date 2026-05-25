// Integration test exercising Layer B firing against
// CampaignHypothesisFormation events produced from F3 candidate data
// per §0183. Mirrors §0157 + §0176 patterns on the CampaignHypothesis
// subtype side. Opens the CampaignHypothesis lifecycle integration
// coverage arc + establishes the F3-derived-formation anchor.
//
// Per §0182 event-centric ontology: the helper
// commitCampaignHypothesisFormationFromCandidate DROPS ActorRefs from
// the candidate at formation commit (CampaignHypothesisFormation proto
// has no actor_refs field per §0063). This is structurally distinct
// from §0157's commitFormationFromCandidate (AutomationGroup) +
// §0176's commitBehavioralClusterFormationFromCandidate (BehavioralCluster)
// helpers which both commit ActorRefs.
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

// commitCampaignHypothesisFormationFromCandidate materializes a
// CampaignHypothesisFormation proto from an F3 candidate + commits
// it to substrate. CampaignHypothesis equivalent of §0157's
// commitFormationFromCandidate + §0176's
// commitBehavioralClusterFormationFromCandidate.
//
// Per §0182 + §0063 event-centric ontology: CandidateActorRefs are
// DROPPED at commit (proto has no actor_refs field). The formation
// proto only carries event memberships via source_event_hashes.
// Operators recover actor attribution by walking the referenced
// Cat I observations.
//
// Foundation helper for subsequent §0184 cross-formation lifecycle
// tests (merge/dissolve/split).
func commitCampaignHypothesisFormationFromCandidate(t *testing.T, sub *substrate.Substrate, c *signatures.FormationCandidate, formationAt int64) [32]byte {
	t.Helper()
	ctx := context.Background()

	formation := &eventsv1.CampaignHypothesisFormation{
		PatternSignature:   c.SignatureName,
		PatternParameters:  "endpoint_window_seconds=60;min_cohort_size=3",
		FormationAt:        formationAt,
		SourceEventHashes:  c.SourceHashes,
		DirectInfluencedBy: nil,
		ClosureHashes:      nil,
		Confidence:         float32(c.ConfidenceHint),
		EvidentialIndependence: &commonv1.EvidentialIndependence{
			Numerator:   1,
			Denominator: 1,
		},
		// ActorRefs INTENTIONALLY DROPPED per §0182 + §0063 event-centric
		// ontology — proto has no actor_refs field.
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

func halfHalfParams(n uint64) *commonv1.LayerBParameters {
	return &commonv1.LayerBParameters{
		TB:                    &commonv1.EvidentialIndependence{Numerator: 1, Denominator: 2},
		KC:                    &commonv1.EvidentialIndependence{Numerator: 1, Denominator: 2},
		NWindow:               n,
		NADurationNanoseconds: 86400000000000,
	}
}

// TestLayerBFiring_AgainstCampaignHypothesisF3CandidateFormation
// exercises the F3 → CampaignHypothesisFormation → Layer B path.
// Mirrors §0157 + §0176 on the CampaignHypothesis subtype side.
func TestLayerBFiring_AgainstCampaignHypothesisF3CandidateFormation(t *testing.T) {
	sub, in := newLayerBFiringSubstrate(t)
	ctx := context.Background()

	// Step 1: Inject 3 NetworkObservation records sharing endpoint +
	// time bucket above threshold 3.
	const bucketStart = int64(1716120000_000_000_000)
	appendNetworkObs(t, in, "actor-1", "10.0.0.1:443", bucketStart)
	appendNetworkObs(t, in, "actor-2", "10.0.0.1:443", bucketStart+10_000_000_000)
	appendNetworkObs(t, in, "actor-3", "10.0.0.1:443", bucketStart+20_000_000_000)

	// Step 2: Run F3 → 1 event-centric candidate.
	observations, err := collectNetworkObservations(ctx, sub)
	if err != nil {
		t.Fatalf("collectNetworkObservations: %v", err)
	}
	sig := &signatures.TemporalEndpointCohortV1{}
	result, err := sig.EvaluateNetwork(ctx, observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if got, want := len(result.Candidates), 1; got != want {
		t.Fatalf("candidates count: got %d want %d", got, want)
	}
	candidate := result.Candidates[0]
	if candidate.HypothesisSubtype != signatures.HypothesisSubtypeCampaignHypothesis {
		t.Errorf("candidate subtype: got %v want CampaignHypothesis", candidate.HypothesisSubtype)
	}

	// Step 3: Operator-elected CampaignHypothesisFormation commit
	// via new helper (drops ActorRefs per event-centric ontology).
	formationAt := int64(1716120100000000000)
	formationHash := commitCampaignHypothesisFormationFromCandidate(t, sub, candidate, formationAt)

	// Step 4: Verify formation lands in substrate.
	row, err := sub.LookupRow(ctx, formationHash)
	if err != nil {
		t.Fatalf("LookupRow formation: %v", err)
	}
	if row.MessageType != "ghosttrace.events.v1.CampaignHypothesisFormation" {
		t.Errorf("formation MessageType: got %q want CampaignHypothesisFormation", row.MessageType)
	}

	// Step 5: Unmarshal + verify formation proto.
	payload, err := sub.ReadBlob(ctx, formationHash)
	if err != nil {
		t.Fatalf("ReadBlob formation: %v", err)
	}
	gotFormation := &eventsv1.CampaignHypothesisFormation{}
	if err := proto.Unmarshal(payload, gotFormation); err != nil {
		t.Fatalf("Unmarshal formation: %v", err)
	}
	if gotFormation.PatternSignature != "temporal_endpoint_cohort_v1" {
		t.Errorf("formation PatternSignature: got %q want temporal_endpoint_cohort_v1", gotFormation.PatternSignature)
	}
	if len(gotFormation.SourceEventHashes) != int(candidate.EvidenceCount) {
		t.Errorf("formation SourceEventHashes count: got %d want %d", len(gotFormation.SourceEventHashes), candidate.EvidenceCount)
	}
	if gotFormation.GetEvidentialIndependence() == nil {
		t.Error("formation EvidentialIndependence is nil (paired-dimension violation per §2.6 BC3)")
	}

	// Step 6: Call layerb.Evaluate against the formation hash.
	verdict, err := layerb.Evaluate(ctx, sub, layerb.EvaluateOptions{
		FormationEventHash: formationHash,
		Params:             halfHalfParams(100),
	})
	if err != nil {
		t.Fatalf("layerb.Evaluate: %v", err)
	}
	if verdict.WindowEventCount == 0 {
		t.Error("verdict WindowEventCount is 0; expected substrate events to be walked")
	}
	t.Logf("Layer B verdict (CampaignHypothesis): WindowEventCount=%d FilterMatchCount=%d Fired=%v",
		verdict.WindowEventCount, verdict.FilterMatchCount, verdict.Fired)
}

func TestLayerBFiring_CampaignHypothesisRejectsNilParams(t *testing.T) {
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
