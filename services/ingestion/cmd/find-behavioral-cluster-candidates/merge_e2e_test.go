// End-to-end integration test exercising the AutomationGroupMerge
// equivalent (hypothesis.Merge for BehavioralCluster) lifecycle
// operation from F3-derived candidates per §0179. Mirrors §0165 on
// the BehavioralCluster subtype side. Fourth PR in the §0176-opened
// arc; extends along the BINARY CROSS-FORMATION lifecycle axis.
//
// Scope per §0179: structural connectivity across the F3-derived →
// merge arc + symmetric-relation sentinel. Per-operation semantics
// (per-error sentinels, idempotency, AppendPair) covered by hypothesis
// package unit tests.
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
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

func newMergeE2ESubstrate(t *testing.T) (*substrate.Substrate, *ingest.Ingester) {
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

// commitMergedBehavioralClusterFormation materializes a "produced"
// BehavioralClusterFormation representing the merged hypothesis of
// two antecedent F3 candidates. Mirrors §0165's commitMergedFormation
// on the BehavioralCluster subtype side.
func commitMergedBehavioralClusterFormation(t *testing.T, sub *substrate.Substrate, actorRefs []string, sourceHashes [][]byte, formationAt int64) [32]byte {
	t.Helper()
	ctx := context.Background()

	sortedSources := sortHashListAscending(sourceHashes)

	formation := &eventsv1.BehavioralClusterFormation{
		PatternSignature:   "keystroke_timing_clustering_v1+merged",
		PatternParameters:  "threshold=3;merged",
		ActorRefs:          actorRefs,
		FormationAt:        formationAt,
		SourceEventHashes:  sortedSources,
		DirectInfluencedBy: nil,
		ClosureHashes:      nil,
		Confidence:         0.85,
		EvidentialIndependence: &commonv1.EvidentialIndependence{
			Numerator:   1,
			Denominator: 1,
		},
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

// TestMergeBehavioralCluster_FromF3CandidateAntecedents exercises the
// full F3 → 2 formations → operator-elected merge arc for
// BehavioralCluster. Mirrors §0165 on the BehavioralCluster subtype
// side.
func TestMergeBehavioralCluster_FromF3CandidateAntecedents(t *testing.T) {
	sub, in := newMergeE2ESubstrate(t)
	ctx := context.Background()

	// Step 1: Inject for 2 distinct fingerprint clusters (3 actors
	// each), so F3 emits 2 distinct multi-actor candidates.
	ivsA := []uint64{50_000_000, 100_000_000, 0, 100_000_000, 100_000_000, 150_000_000}
	ivsB := []uint64{150_000_000, 200_000_000, 100_000_000, 200_000_000, 200_000_000, 250_000_000}
	for i, actor := range []string{"actor-a-1", "actor-a-2", "actor-a-3"} {
		appendKeystrokeObs(t, in, actor, ivsA, int64(1+i))
	}
	for i, actor := range []string{"actor-b-1", "actor-b-2", "actor-b-3"} {
		appendKeystrokeObs(t, in, actor, ivsB, int64(10+i))
	}

	// Step 2: Run F3 → 2 candidates.
	observations, err := collectBehavioralObservations(ctx, sub)
	if err != nil {
		t.Fatalf("collectBehavioralObservations: %v", err)
	}
	sig := &signatures.KeystrokeTimingClusteringV1{}
	result, err := sig.EvaluateBehavioral(ctx, observations)
	if err != nil {
		t.Fatalf("EvaluateBehavioral: %v", err)
	}
	if len(result.Candidates) != 2 {
		t.Fatalf("candidates count: got %d want 2", len(result.Candidates))
	}
	candA := findCandidateByActor(t, result.Candidates, "actor-a-1")
	candB := findCandidateByActor(t, result.Candidates, "actor-b-1")

	// Step 3: Commit two F3-derived BehavioralClusterFormation records.
	hashA := commitBehavioralClusterFormationFromCandidate(t, sub, candA, 1716120100000000000)
	hashB := commitBehavioralClusterFormationFromCandidate(t, sub, candB, 1716120200000000000)
	if hashA == hashB {
		t.Fatal("formation hashes A and B must differ for merge to be valid")
	}

	// Step 4: Operator-elected merged-hypothesis formation_P with
	// combined actor_refs + union source_hashes.
	mergedActors := []string{"actor-a-1", "actor-a-2", "actor-a-3", "actor-b-1", "actor-b-2", "actor-b-3"}
	mergedSources := make([][]byte, 0, len(candA.SourceHashes)+len(candB.SourceHashes))
	mergedSources = append(mergedSources, candA.SourceHashes...)
	mergedSources = append(mergedSources, candB.SourceHashes...)
	hashP := commitMergedBehavioralClusterFormation(t, sub, mergedActors, mergedSources, 1716120300000000000)
	if hashP == hashA || hashP == hashB {
		t.Fatal("produced formation hash P must differ from antecedents A and B")
	}

	// Step 5: Operator-elected merge commit via hypothesis.Merge
	// (BehavioralCluster-targeted per §0178 MO1).
	mergedAt := int64(1716120400000000000)
	now := func() time.Time { return time.Unix(0, mergedAt) }
	mergeReport, err := hypothesis.Merge(ctx, sub, hypothesis.MergeOptions{
		AntecedentAFormationHash: hashA,
		AntecedentBFormationHash: hashB,
		ProducedFormationHash:    hashP,
		MergedAt:                 mergedAt,
		Reason:                   "e2e test merge (BehavioralCluster): F3 candidates a + b describe same phenomenon",
	}, now)
	if err != nil {
		t.Fatalf("hypothesis.Merge: %v", err)
	}
	if mergeReport.MergeEventHashHex == "" {
		t.Fatal("MergeReport.MergeEventHashHex is empty")
	}
	if mergeReport.AlreadyMerged {
		t.Error("AlreadyMerged: got true want false (fresh merge)")
	}

	// Step 6: Verify merge event lands + references all three
	// formations + antecedents sorted ascending per §0049.
	var mergeHash [32]byte
	if _, err := hexDecodeInto(mergeReport.MergeEventHashHex, mergeHash[:]); err != nil {
		t.Fatalf("decode merge hash hex: %v", err)
	}
	mergeRow, err := sub.LookupRow(ctx, mergeHash)
	if err != nil {
		t.Fatalf("LookupRow merge: %v", err)
	}
	if mergeRow.MessageType != "ghosttrace.events.v1.BehavioralClusterMerge" {
		t.Errorf("merge MessageType: got %q want BehavioralClusterMerge", mergeRow.MessageType)
	}

	mergePayload, err := sub.ReadBlob(ctx, mergeHash)
	if err != nil {
		t.Fatalf("ReadBlob merge: %v", err)
	}
	gotMerge := &eventsv1.BehavioralClusterMerge{}
	if err := proto.Unmarshal(mergePayload, gotMerge); err != nil {
		t.Fatalf("Unmarshal merge: %v", err)
	}
	if len(gotMerge.AntecedentFormationEventHashes) != 2 {
		t.Fatalf("merge AntecedentFormationEventHashes count: got %d want 2", len(gotMerge.AntecedentFormationEventHashes))
	}

	// Verify ascending-sort per §0049 symmetric-relation discipline.
	a := gotMerge.AntecedentFormationEventHashes[0]
	b := gotMerge.AntecedentFormationEventHashes[1]
	if bytes.Compare(a, b) >= 0 {
		t.Error("merge antecedents not sorted ascending per §0049")
	}

	// Verify both antecedent hashes present (regardless of post-sort order).
	containsA := bytes.Equal(a, hashA[:]) || bytes.Equal(b, hashA[:])
	containsB := bytes.Equal(a, hashB[:]) || bytes.Equal(b, hashB[:])
	if !containsA {
		t.Error("merge antecedents do not contain formation_A hash")
	}
	if !containsB {
		t.Error("merge antecedents do not contain formation_B hash")
	}

	// Verify produced formation hash matches.
	if !bytes.Equal(gotMerge.ProducedFormationEventHash, hashP[:]) {
		t.Error("merge ProducedFormationEventHash does not match formation_P hash")
	}
	if gotMerge.Reason == "" {
		t.Error("merge Reason: got empty want non-empty")
	}
	if gotMerge.MergedAt != mergedAt {
		t.Errorf("merge MergedAt: got %d want %d", gotMerge.MergedAt, mergedAt)
	}

	// All four committed records must be present.
	for _, h := range [][32]byte{hashA, hashB, hashP, mergeHash} {
		if _, err := sub.LookupRow(ctx, h); err != nil {
			t.Errorf("LookupRow %x: %v", h[:8], err)
		}
	}

	t.Logf("merge E2E (BehavioralCluster): A=%x B=%x P=%x merge=%x", hashA[:4], hashB[:4], hashP[:4], mergeHash[:4])
}

// TestMergeBehavioralCluster_RejectsIdenticalAntecedents confirms
// ErrMergeAntecedentsIdentical surfaces at the integration path layer.
// Mirrors §0165 on the BehavioralCluster subtype side.
func TestMergeBehavioralCluster_RejectsIdenticalAntecedents(t *testing.T) {
	sub, in := newMergeE2ESubstrate(t)
	ctx := context.Background()

	ivs := []uint64{50_000_000, 100_000_000, 0, 100_000_000, 100_000_000, 150_000_000}
	appendKeystrokeObs(t, in, "actor-x-1", ivs, 1)
	appendKeystrokeObs(t, in, "actor-x-2", ivs, 2)
	appendKeystrokeObs(t, in, "actor-x-3", ivs, 3)

	observations, err := collectBehavioralObservations(ctx, sub)
	if err != nil {
		t.Fatalf("collectBehavioralObservations: %v", err)
	}
	sig := &signatures.KeystrokeTimingClusteringV1{}
	result, err := sig.EvaluateBehavioral(ctx, observations)
	if err != nil {
		t.Fatalf("EvaluateBehavioral: %v", err)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("candidates count: got %d want 1", len(result.Candidates))
	}
	hashA := commitBehavioralClusterFormationFromCandidate(t, sub, result.Candidates[0], 1716120100000000000)
	hashP := commitMergedBehavioralClusterFormation(t, sub, result.Candidates[0].ActorRefs, result.Candidates[0].SourceHashes, 1716120200000000000)

	_, err = hypothesis.Merge(ctx, sub, hypothesis.MergeOptions{
		AntecedentAFormationHash: hashA,
		AntecedentBFormationHash: hashA,
		ProducedFormationHash:    hashP,
		MergedAt:                 1716120300000000000,
	}, nil)
	if err == nil {
		t.Fatal("expected error for identical antecedents, got nil")
	}
}
