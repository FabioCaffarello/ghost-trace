// Integration test exercising Layer B firing against
// CoordinationRingFormation events produced from F3 candidate data
// per §0186. Mirrors §0157 + §0176 + §0183 patterns on the
// CoordinationRing subtype side. Opens the CoordinationRing lifecycle
// integration coverage arc + establishes the F3-derived-formation
// anchor.
//
// Per §0070 + §0185 interaction-centric ontology: the §0186 foundation
// helper commitCoordinationRingFormationFromCandidate CONVERTS
// candidate.Interactions [][2]string into CoordinationRingInteraction
// protos (actor_a=edge[0], actor_b=edge[1] per §0070 within-edge lex).
// This is structurally distinct from:
//   - §0157 commitFormationFromCandidate (AutomationGroup): commits actor_refs
//   - §0176 commitBehavioralClusterFormationFromCandidate: commits actor_refs
//   - §0183 commitCampaignHypothesisFormationFromCandidate: DROPS actor_refs
//     (event-centric)
//
// CoordinationRingFormation does NOT have an actor_refs field per §0070
// (vertex set recoverable from edge union); the helper commits ONLY
// interactions as the structural membership shape.
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

// commitCoordinationRingFormationFromCandidate materializes a
// CoordinationRingFormation proto from an F3 candidate + commits it
// to substrate. CoordinationRing equivalent of §0157's
// commitFormationFromCandidate, §0176's
// commitBehavioralClusterFormationFromCandidate, + §0183's
// commitCampaignHypothesisFormationFromCandidate.
//
// Per §0070 + §0185 interaction-centric ontology: candidate.Interactions
// [][2]string is converted into []*CoordinationRingInteraction protos
// with actor_a=edge[0] + actor_b=edge[1] (within-edge lex preserved
// from candidate). Ascending-sort across edges + no-duplicates
// preserved from candidate emission.
//
// Foundation helper for subsequent §0187 cross-formation lifecycle
// tests (merge/dissolve/split).
func commitCoordinationRingFormationFromCandidate(t *testing.T, sub *substrate.Substrate, c *signatures.FormationCandidate, formationAt int64) [32]byte {
	t.Helper()
	ctx := context.Background()

	interactions := make([]*eventsv1.CoordinationRingInteraction, len(c.Interactions))
	for i, edge := range c.Interactions {
		interactions[i] = &eventsv1.CoordinationRingInteraction{
			ActorA: edge[0],
			ActorB: edge[1],
		}
	}

	formation := &eventsv1.CoordinationRingFormation{
		PatternSignature:   c.SignatureName,
		PatternParameters:  "endpoint_window_seconds=60;min_cohort_size=3",
		FormationAt:        formationAt,
		SourceEventHashes:  c.SourceHashes,
		Interactions:       interactions,
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

func halfHalfParams(n uint64) *commonv1.LayerBParameters {
	return &commonv1.LayerBParameters{
		TB:                    &commonv1.EvidentialIndependence{Numerator: 1, Denominator: 2},
		KC:                    &commonv1.EvidentialIndependence{Numerator: 1, Denominator: 2},
		NWindow:               n,
		NADurationNanoseconds: 86400000000000,
	}
}

// TestLayerBFiring_AgainstCoordinationRingF3CandidateFormation
// exercises the F3 → CoordinationRingFormation → Layer B path.
// Mirrors §0157 + §0176 + §0183 on the CoordinationRing subtype side.
func TestLayerBFiring_AgainstCoordinationRingF3CandidateFormation(t *testing.T) {
	sub, in := newLayerBFiringSubstrate(t)
	ctx := context.Background()

	const bucketStart = int64(1716120000_000_000_000)
	appendNetworkObs(t, in, "actor-1", "10.0.0.1:443", bucketStart)
	appendNetworkObs(t, in, "actor-2", "10.0.0.1:443", bucketStart+10_000_000_000)
	appendNetworkObs(t, in, "actor-3", "10.0.0.1:443", bucketStart+20_000_000_000)

	observations, err := observationcollector.CollectNetwork(ctx, sub)
	if err != nil {
		t.Fatalf("observationcollector.CollectNetwork: %v", err)
	}
	sig := &signatures.EndpointCoVisitV1{}
	result, err := sig.EvaluateNetwork(ctx, observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if got, want := len(result.Candidates), 1; got != want {
		t.Fatalf("candidates count: got %d want %d", got, want)
	}
	candidate := result.Candidates[0]
	if candidate.HypothesisSubtype != signatures.HypothesisSubtypeCoordinationRing {
		t.Errorf("candidate subtype: got %v want CoordinationRing", candidate.HypothesisSubtype)
	}
	if len(candidate.Interactions) != 3 {
		t.Errorf("candidate Interactions: got %d want 3 (3 actors → 3 edges)", len(candidate.Interactions))
	}

	formationAt := int64(1716120100000000000)
	formationHash := commitCoordinationRingFormationFromCandidate(t, sub, candidate, formationAt)

	row, err := sub.LookupRow(ctx, formationHash)
	if err != nil {
		t.Fatalf("LookupRow formation: %v", err)
	}
	if row.MessageType != "ghosttrace.events.v1.CoordinationRingFormation" {
		t.Errorf("formation MessageType: got %q want CoordinationRingFormation", row.MessageType)
	}

	payload, err := sub.ReadBlob(ctx, formationHash)
	if err != nil {
		t.Fatalf("ReadBlob formation: %v", err)
	}
	gotFormation := &eventsv1.CoordinationRingFormation{}
	if err := proto.Unmarshal(payload, gotFormation); err != nil {
		t.Fatalf("Unmarshal formation: %v", err)
	}
	if gotFormation.PatternSignature != "endpoint_co_visit_v1" {
		t.Errorf("formation PatternSignature: got %q want endpoint_co_visit_v1", gotFormation.PatternSignature)
	}
	if len(gotFormation.Interactions) != 3 {
		t.Errorf("formation Interactions count: got %d want 3 (per §0070 edge canonicalization)", len(gotFormation.Interactions))
	}
	for i, edge := range gotFormation.Interactions {
		if edge.ActorA >= edge.ActorB {
			t.Errorf("formation Interactions[%d]: actor_a=%q NOT lex-less-than actor_b=%q (per §0070)", i, edge.ActorA, edge.ActorB)
		}
	}
	if gotFormation.GetEvidentialIndependence() == nil {
		t.Error("formation EvidentialIndependence is nil (paired-dimension violation per §2.6 BC3)")
	}

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
	t.Logf("Layer B verdict (CoordinationRing): WindowEventCount=%d FilterMatchCount=%d Fired=%v",
		verdict.WindowEventCount, verdict.FilterMatchCount, verdict.Fired)
}

func TestLayerBFiring_CoordinationRingRejectsNilParams(t *testing.T) {
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
