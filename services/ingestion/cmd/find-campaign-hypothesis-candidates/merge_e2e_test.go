// End-to-end integration test exercising the MergeCampaignHypothesis
// lifecycle operation from F3-derived candidates per §0184. Mirrors
// §0165 + §0179 on the CampaignHypothesis subtype side. First PR-#2
// axis of the §0181 MO2 bundled CampaignHypothesis lifecycle arc;
// extends along the BINARY CROSS-FORMATION lifecycle axis.
//
// Per §0182 + §0063 event-centric ontology: both antecedent
// CampaignHypothesisFormation records AND the merged "produced"
// formation_P record DROP ActorRefs (proto has no actor_refs field).
// This is the structural distinction from §0165 (AutomationGroup) +
// §0179 (BehavioralCluster) which both commit ActorRefs at every
// formation site.
package main

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/hypothesis"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/signatures"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/observationcollector"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

func newCampaignMergeE2ESubstrate(t *testing.T) (*substrate.Substrate, *ingest.Ingester) {
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

// commitMergedCampaignHypothesisFormation materializes a "produced"
// CampaignHypothesisFormation representing the merged hypothesis of
// two antecedent F3 candidates. Mirrors §0165's commitMergedFormation
// + §0179's commitMergedBehavioralClusterFormation on the
// CampaignHypothesis subtype side.
//
// Per §0182 + §0063 event-centric ontology: ActorRefs DROPPED at
// commit (proto has no actor_refs field). Operators recover actor
// attribution by walking the union source_event_hashes back to
// Cat I observations.
func commitMergedCampaignHypothesisFormation(t *testing.T, sub *substrate.Substrate, sourceHashes [][]byte, formationAt int64) [32]byte {
	t.Helper()
	ctx := context.Background()

	sortedSources := sortHashListAscending(sourceHashes)

	formation := &eventsv1.CampaignHypothesisFormation{
		PatternSignature:   "temporal_endpoint_cohort_v1+merged",
		PatternParameters:  "endpoint_window_seconds=60;min_cohort_size=3;merged",
		FormationAt:        formationAt,
		SourceEventHashes:  sortedSources,
		DirectInfluencedBy: nil,
		ClosureHashes:      nil,
		Confidence:         0.85,
		EvidentialIndependence: &commonv1.EvidentialIndependence{
			Numerator:   1,
			Denominator: 1,
		},
		// ActorRefs INTENTIONALLY DROPPED per §0182 + §0063.
	}

	payload, hash, err := canonical.MarshalAndHash(formation)
	if err != nil {
		t.Fatalf("MarshalAndHash merged formation: %v", err)
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
		t.Fatalf("substrate.Append merged formation: %v", err)
	}
	return hash
}

// TestMergeCampaignHypothesis_FromF3CandidateAntecedents exercises
// the full F3 → 2 formations → operator-elected merge arc for
// CampaignHypothesis. Mirrors §0165 + §0179 on the CampaignHypothesis
// subtype side.
func TestMergeCampaignHypothesis_FromF3CandidateAntecedents(t *testing.T) {
	sub, in := newCampaignMergeE2ESubstrate(t)
	ctx := context.Background()

	// Step 1: Inject 2 distinct (endpoint, time-bucket) clusters via
	// separate endpoint addresses → F3 emits 2 distinct candidates.
	const bucketStart = int64(1716120000_000_000_000)
	for i, actor := range []string{"actor-a-1", "actor-a-2", "actor-a-3"} {
		appendNetworkObs(t, in, actor, "10.0.0.1:443", bucketStart+int64(i*1_000_000_000))
	}
	for i, actor := range []string{"actor-b-1", "actor-b-2", "actor-b-3"} {
		appendNetworkObs(t, in, actor, "10.0.0.2:443", bucketStart+60_000_000_000+int64(i*1_000_000_000))
	}

	// Step 2: Run F3 → 2 candidates.
	observations, err := observationcollector.CollectNetwork(ctx, sub)
	if err != nil {
		t.Fatalf("observationcollector.CollectNetwork: %v", err)
	}
	sig := &signatures.TemporalEndpointCohortV1{}
	result, err := sig.EvaluateNetwork(ctx, observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(result.Candidates) != 2 {
		t.Fatalf("candidates count: got %d want 2", len(result.Candidates))
	}
	candA, candB := result.Candidates[0], result.Candidates[1]

	// Step 3: Commit two F3-derived CampaignHypothesisFormation records
	// (ActorRefs dropped per §0182 + §0063).
	hashA := commitCampaignHypothesisFormationFromCandidate(t, sub, candA, 1716120100000000000)
	hashB := commitCampaignHypothesisFormationFromCandidate(t, sub, candB, 1716120200000000000)
	if hashA == hashB {
		t.Fatal("formation hashes A and B must differ for merge to be valid")
	}

	// Step 4: Operator-elected merged-hypothesis formation_P with
	// union source_hashes. NO actor_refs combined (proto field absent).
	mergedSources := make([][]byte, 0, len(candA.SourceHashes)+len(candB.SourceHashes))
	mergedSources = append(mergedSources, candA.SourceHashes...)
	mergedSources = append(mergedSources, candB.SourceHashes...)
	hashP := commitMergedCampaignHypothesisFormation(t, sub, mergedSources, 1716120300000000000)
	if hashP == hashA || hashP == hashB {
		t.Fatal("produced formation hash P must differ from antecedents A and B")
	}

	// Step 5: Operator-elected merge commit via MergeCampaignHypothesis
	// (subtype-suffixed per §0178 MO1).
	mergedAt := int64(1716120400000000000)
	now := func() time.Time { return time.Unix(0, mergedAt) }
	mergeReport, err := hypothesis.MergeCampaignHypothesis(ctx, sub, hypothesis.CampaignHypothesisMergeOptions{
		AntecedentAFormationHash: hashA,
		AntecedentBFormationHash: hashB,
		ProducedFormationHash:    hashP,
		MergedAt:                 mergedAt,
		Reason:                   "e2e test merge (CampaignHypothesis): F3 candidates a + b describe same campaign",
	}, now)
	if err != nil {
		t.Fatalf("hypothesis.MergeCampaignHypothesis: %v", err)
	}
	if mergeReport.MergeEventHashHex == "" {
		t.Fatal("MergeReport.MergeEventHashHex is empty")
	}
	if mergeReport.AlreadyMerged {
		t.Error("AlreadyMerged: got true want false (fresh merge)")
	}

	// Step 6: Verify merge event lands + references both antecedents
	// sorted ascending per §0049 symmetric-relation discipline.
	var mergeHash [32]byte
	if _, err := hexDecodeInto(mergeReport.MergeEventHashHex, mergeHash[:]); err != nil {
		t.Fatalf("decode merge hash hex: %v", err)
	}
	mergeRow, err := sub.LookupRow(ctx, mergeHash)
	if err != nil {
		t.Fatalf("LookupRow merge: %v", err)
	}
	if mergeRow.MessageType != "ghosttrace.events.v1.CampaignHypothesisMerge" {
		t.Errorf("merge MessageType: got %q want CampaignHypothesisMerge", mergeRow.MessageType)
	}

	mergePayload, err := sub.ReadBlob(ctx, mergeHash)
	if err != nil {
		t.Fatalf("ReadBlob merge: %v", err)
	}
	gotMerge := &eventsv1.CampaignHypothesisMerge{}
	if err := proto.Unmarshal(mergePayload, gotMerge); err != nil {
		t.Fatalf("Unmarshal merge: %v", err)
	}
	if len(gotMerge.AntecedentFormationEventHashes) != 2 {
		t.Fatalf("merge AntecedentFormationEventHashes count: got %d want 2", len(gotMerge.AntecedentFormationEventHashes))
	}

	a := gotMerge.AntecedentFormationEventHashes[0]
	b := gotMerge.AntecedentFormationEventHashes[1]
	if bytes.Compare(a, b) >= 0 {
		t.Error("merge antecedents not sorted ascending per §0049")
	}

	containsA := bytes.Equal(a, hashA[:]) || bytes.Equal(b, hashA[:])
	containsB := bytes.Equal(a, hashB[:]) || bytes.Equal(b, hashB[:])
	if !containsA {
		t.Error("merge antecedents do not contain formation_A hash")
	}
	if !containsB {
		t.Error("merge antecedents do not contain formation_B hash")
	}

	if !bytes.Equal(gotMerge.ProducedFormationEventHash, hashP[:]) {
		t.Error("merge ProducedFormationEventHash does not match formation_P hash")
	}
	if gotMerge.Reason == "" {
		t.Error("merge Reason: got empty want non-empty")
	}
	if gotMerge.MergedAt != mergedAt {
		t.Errorf("merge MergedAt: got %d want %d", gotMerge.MergedAt, mergedAt)
	}

	for _, h := range [][32]byte{hashA, hashB, hashP, mergeHash} {
		if _, err := sub.LookupRow(ctx, h); err != nil {
			t.Errorf("LookupRow %x: %v", h[:8], err)
		}
	}

	t.Logf("merge E2E (CampaignHypothesis): A=%x B=%x P=%x merge=%x", hashA[:4], hashB[:4], hashP[:4], mergeHash[:4])
}

// TestMergeCampaignHypothesis_RejectsIdenticalAntecedents confirms
// ErrMergeAntecedentsIdentical surfaces at the integration path layer
// for CampaignHypothesis. Mirrors §0165 + §0179 sentinel-coverage
// pattern.
func TestMergeCampaignHypothesis_RejectsIdenticalAntecedents(t *testing.T) {
	sub, in := newCampaignMergeE2ESubstrate(t)
	ctx := context.Background()

	const bucketStart = int64(1716120000_000_000_000)
	appendNetworkObs(t, in, "actor-x-1", "10.0.0.5:443", bucketStart)
	appendNetworkObs(t, in, "actor-x-2", "10.0.0.5:443", bucketStart+10_000_000_000)
	appendNetworkObs(t, in, "actor-x-3", "10.0.0.5:443", bucketStart+20_000_000_000)

	observations, err := observationcollector.CollectNetwork(ctx, sub)
	if err != nil {
		t.Fatalf("observationcollector.CollectNetwork: %v", err)
	}
	sig := &signatures.TemporalEndpointCohortV1{}
	result, err := sig.EvaluateNetwork(ctx, observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("candidates count: got %d want 1", len(result.Candidates))
	}
	hashA := commitCampaignHypothesisFormationFromCandidate(t, sub, result.Candidates[0], 1716120100000000000)
	hashP := commitMergedCampaignHypothesisFormation(t, sub, result.Candidates[0].SourceHashes, 1716120200000000000)

	_, err = hypothesis.MergeCampaignHypothesis(ctx, sub, hypothesis.CampaignHypothesisMergeOptions{
		AntecedentAFormationHash: hashA,
		AntecedentBFormationHash: hashA,
		ProducedFormationHash:    hashP,
		MergedAt:                 1716120300000000000,
	}, nil)
	if err == nil {
		t.Fatal("expected error for identical antecedents, got nil")
	}
}
