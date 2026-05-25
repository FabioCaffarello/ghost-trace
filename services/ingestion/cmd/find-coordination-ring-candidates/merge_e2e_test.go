// End-to-end integration test exercising the MergeCoordinationRing
// lifecycle operation from F3-derived candidates per §0187. Mirrors
// §0165 + §0179 + §0184 on the CoordinationRing subtype side. First
// PR-#2 axis of the §0184 MO2 bundled CoordinationRing lifecycle arc;
// extends along the BINARY CROSS-FORMATION lifecycle axis.
//
// Per §0070 + §0185 interaction-centric ontology: antecedent formations
// AND the merged "produced" formation_P all CONVERT
// candidate.Interactions [][2]string into []CoordinationRingInteraction
// protos (proto has no actor_refs field — edges are the structural
// membership shape). Structural distinction from §0165 + §0179
// (actor-centric: actor_refs) + §0184 (event-centric: source_event_hashes
// only).
package main

import (
	"bytes"
	"context"
	"path/filepath"
	"sort"
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

func newCoordinationMergeE2ESubstrate(t *testing.T) (*substrate.Substrate, *ingest.Ingester) {
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

// mergeEdges produces the union edge set from two candidate Interactions
// slices preserving per-§0070 canonicalization (within-edge lex already
// holds for each candidate's edges; across-edge ascending sort + dedup
// across union enforced here).
func mergeEdges(edgesA, edgesB [][2]string) [][2]string {
	seen := make(map[[2]string]struct{}, len(edgesA)+len(edgesB))
	for _, e := range edgesA {
		seen[e] = struct{}{}
	}
	for _, e := range edgesB {
		seen[e] = struct{}{}
	}
	out := make([][2]string, 0, len(seen))
	for e := range seen {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i][0] != out[j][0] {
			return out[i][0] < out[j][0]
		}
		return out[i][1] < out[j][1]
	})
	return out
}

// commitMergedCoordinationRingFormation materializes a "produced"
// CoordinationRingFormation representing the merged hypothesis of two
// antecedent F3 candidates. Mirrors §0165 / §0179 / §0184 on the
// CoordinationRing subtype side.
//
// Per §0070 + §0185 interaction-centric ontology: edges CONVERTED from
// [][2]string union into []CoordinationRingInteraction protos
// preserving within-edge lex + ascending across-edge sort + no duplicates.
func commitMergedCoordinationRingFormation(t *testing.T, sub *substrate.Substrate, mergedEdges [][2]string, sourceHashes [][]byte, formationAt int64) [32]byte {
	t.Helper()
	ctx := context.Background()

	sortedSources := sortHashListAscending(sourceHashes)

	interactions := make([]*eventsv1.CoordinationRingInteraction, len(mergedEdges))
	for i, edge := range mergedEdges {
		interactions[i] = &eventsv1.CoordinationRingInteraction{
			ActorA: edge[0],
			ActorB: edge[1],
		}
	}

	formation := &eventsv1.CoordinationRingFormation{
		PatternSignature:   "endpoint_co_visit_v1+merged",
		PatternParameters:  "endpoint_window_seconds=60;min_cohort_size=3;merged",
		FormationAt:        formationAt,
		SourceEventHashes:  sortedSources,
		Interactions:       interactions,
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

// TestMergeCoordinationRing_FromF3CandidateAntecedents exercises the
// full F3 → 2 formations → operator-elected merge arc for
// CoordinationRing. Mirrors §0165 + §0179 + §0184 on the
// CoordinationRing subtype side.
func TestMergeCoordinationRing_FromF3CandidateAntecedents(t *testing.T) {
	sub, in := newCoordinationMergeE2ESubstrate(t)
	ctx := context.Background()

	// Step 1: Inject 2 distinct (endpoint, time-bucket) actor cohorts
	// via separate endpoint addresses → F3 emits 2 distinct candidates.
	const bucketStart = int64(1716120000_000_000_000)
	for i, actor := range []string{"actor-a-1", "actor-a-2", "actor-a-3"} {
		appendNetworkObs(t, in, actor, "10.0.0.1:443", bucketStart+int64(i*1_000_000_000))
	}
	for i, actor := range []string{"actor-b-1", "actor-b-2", "actor-b-3"} {
		appendNetworkObs(t, in, actor, "10.0.0.2:443", bucketStart+60_000_000_000+int64(i*1_000_000_000))
	}

	observations, err := observationcollector.CollectNetwork(ctx, sub)
	if err != nil {
		t.Fatalf("observationcollector.CollectNetwork: %v", err)
	}
	sig := &signatures.EndpointCoVisitV1{}
	result, err := sig.EvaluateNetwork(ctx, observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(result.Candidates) != 2 {
		t.Fatalf("candidates count: got %d want 2", len(result.Candidates))
	}
	candA, candB := result.Candidates[0], result.Candidates[1]

	hashA := commitCoordinationRingFormationFromCandidate(t, sub, candA, 1716120100000000000)
	hashB := commitCoordinationRingFormationFromCandidate(t, sub, candB, 1716120200000000000)
	if hashA == hashB {
		t.Fatal("formation hashes A and B must differ for merge to be valid")
	}

	// Step 4: Operator-elected merged-hypothesis formation_P with
	// union edge set + union source_hashes. Edge union preserves §0070
	// canonicalization across antecedents.
	mergedEdges := mergeEdges(candA.Interactions, candB.Interactions)
	mergedSources := make([][]byte, 0, len(candA.SourceHashes)+len(candB.SourceHashes))
	mergedSources = append(mergedSources, candA.SourceHashes...)
	mergedSources = append(mergedSources, candB.SourceHashes...)
	hashP := commitMergedCoordinationRingFormation(t, sub, mergedEdges, mergedSources, 1716120300000000000)
	if hashP == hashA || hashP == hashB {
		t.Fatal("produced formation hash P must differ from antecedents A and B")
	}

	// Step 5: Operator-elected merge commit via MergeCoordinationRing
	// (subtype-suffixed per §0178 MO1).
	mergedAt := int64(1716120400000000000)
	now := func() time.Time { return time.Unix(0, mergedAt) }
	mergeReport, err := hypothesis.MergeCoordinationRing(ctx, sub, hypothesis.CoordinationRingMergeOptions{
		AntecedentAFormationHash: hashA,
		AntecedentBFormationHash: hashB,
		ProducedFormationHash:    hashP,
		MergedAt:                 mergedAt,
		Reason:                   "e2e test merge (CoordinationRing): F3 candidates a + b describe coordinated rings",
	}, now)
	if err != nil {
		t.Fatalf("hypothesis.MergeCoordinationRing: %v", err)
	}
	if mergeReport.MergeEventHashHex == "" {
		t.Fatal("MergeReport.MergeEventHashHex is empty")
	}
	if mergeReport.AlreadyMerged {
		t.Error("AlreadyMerged: got true want false (fresh merge)")
	}

	// Step 6: Verify merge event lands + references both antecedents
	// sorted ascending per §0049.
	var mergeHash [32]byte
	if _, err := hexDecodeInto(mergeReport.MergeEventHashHex, mergeHash[:]); err != nil {
		t.Fatalf("decode merge hash hex: %v", err)
	}
	mergeRow, err := sub.LookupRow(ctx, mergeHash)
	if err != nil {
		t.Fatalf("LookupRow merge: %v", err)
	}
	if mergeRow.MessageType != "ghosttrace.events.v1.CoordinationRingMerge" {
		t.Errorf("merge MessageType: got %q want CoordinationRingMerge", mergeRow.MessageType)
	}

	mergePayload, err := sub.ReadBlob(ctx, mergeHash)
	if err != nil {
		t.Fatalf("ReadBlob merge: %v", err)
	}
	gotMerge := &eventsv1.CoordinationRingMerge{}
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

	// Verify merged formation_P edges round-trip with §0070 canonicalization.
	mergedPPayload, err := sub.ReadBlob(ctx, hashP)
	if err != nil {
		t.Fatalf("ReadBlob merged formation_P: %v", err)
	}
	gotMergedP := &eventsv1.CoordinationRingFormation{}
	if err := proto.Unmarshal(mergedPPayload, gotMergedP); err != nil {
		t.Fatalf("Unmarshal merged formation_P: %v", err)
	}
	for i, edge := range gotMergedP.Interactions {
		if edge.ActorA >= edge.ActorB {
			t.Errorf("merged formation_P Interactions[%d]: actor_a=%q NOT lex-less-than actor_b=%q (per §0070)", i, edge.ActorA, edge.ActorB)
		}
	}

	for _, h := range [][32]byte{hashA, hashB, hashP, mergeHash} {
		if _, err := sub.LookupRow(ctx, h); err != nil {
			t.Errorf("LookupRow %x: %v", h[:8], err)
		}
	}

	t.Logf("merge E2E (CoordinationRing): A=%x B=%x P=%x merge=%x edges_merged=%d", hashA[:4], hashB[:4], hashP[:4], mergeHash[:4], len(gotMergedP.Interactions))
}

// TestMergeCoordinationRing_RejectsIdenticalAntecedents confirms
// ErrMergeAntecedentsIdentical surfaces at the integration path layer
// for CoordinationRing. Mirrors §0165 + §0179 + §0184 sentinel pattern.
func TestMergeCoordinationRing_RejectsIdenticalAntecedents(t *testing.T) {
	sub, in := newCoordinationMergeE2ESubstrate(t)
	ctx := context.Background()

	const bucketStart = int64(1716120000_000_000_000)
	appendNetworkObs(t, in, "actor-x-1", "10.0.0.5:443", bucketStart)
	appendNetworkObs(t, in, "actor-x-2", "10.0.0.5:443", bucketStart+10_000_000_000)
	appendNetworkObs(t, in, "actor-x-3", "10.0.0.5:443", bucketStart+20_000_000_000)

	observations, err := observationcollector.CollectNetwork(ctx, sub)
	if err != nil {
		t.Fatalf("observationcollector.CollectNetwork: %v", err)
	}
	sig := &signatures.EndpointCoVisitV1{}
	result, err := sig.EvaluateNetwork(ctx, observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("candidates count: got %d want 1", len(result.Candidates))
	}
	hashA := commitCoordinationRingFormationFromCandidate(t, sub, result.Candidates[0], 1716120100000000000)
	hashP := commitMergedCoordinationRingFormation(t, sub, result.Candidates[0].Interactions, result.Candidates[0].SourceHashes, 1716120200000000000)

	_, err = hypothesis.MergeCoordinationRing(ctx, sub, hypothesis.CoordinationRingMergeOptions{
		AntecedentAFormationHash: hashA,
		AntecedentBFormationHash: hashA,
		ProducedFormationHash:    hashP,
		MergedAt:                 1716120300000000000,
	}, nil)
	if err == nil {
		t.Fatal("expected error for identical antecedents, got nil")
	}
}
