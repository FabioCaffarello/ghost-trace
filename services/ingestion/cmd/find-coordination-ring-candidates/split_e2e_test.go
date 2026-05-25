// End-to-end integration test exercising the SplitCoordinationRing
// lifecycle operation from F3-derived candidates per §0187. Mirrors
// §0167 + §0181 + §0184 on the CoordinationRing subtype side. Third
// PR-#2 axis of the §0184 MO2 bundled CoordinationRing lifecycle arc;
// extends along the K-ARY CROSS-FORMATION lifecycle axis.
//
// CLOSES the CoordinationRing lifecycle integration coverage arc
// opened at §0186: bundled PR #2 of 2. With §0187 landed,
// CoordinationRing matches AutomationGroup + BehavioralCluster +
// CampaignHypothesis full §0011 lifecycle integration coverage from
// F3-derived candidates: formation + promotion (§0186 linear arc) +
// demotion (§0186) + merge (§0187) + dissolution (§0187) + split
// (this entry).
//
// **F3-derived lifecycle integration coverage reaches 4-of-4 Cat III
// subtypes** — completes the F3-loop multi-PR arc program.
//
// Per §0070 + §0185 interaction-centric ontology: antecedent + all K
// successor CoordinationRingFormation records CONVERT edges per §0070.
package main

import (
	"bytes"
	"context"
	"errors"
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

func newCoordinationSplitE2ESubstrate(t *testing.T) (*substrate.Substrate, *ingest.Ingester) {
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

// commitCoordinationRingSplitSuccessor materializes a separately-
// committed CoordinationRingFormation representing one of K successor
// rings of a split. Mirrors §0167 / §0181 / §0184 on the
// CoordinationRing subtype side. Each successor carries a distinct
// pattern_signature label per §0167 MO2 (per-successor labels surface
// operator's split decision semantically).
//
// Per §0070 + §0185 interaction-centric ontology: successorEdges
// CONVERTED into []CoordinationRingInteraction protos preserving
// within-edge lex.
func commitCoordinationRingSplitSuccessor(t *testing.T, sub *substrate.Substrate, successorEdges [][2]string, sourceHashes [][]byte, patternSignature string, formationAt int64) [32]byte {
	t.Helper()
	ctx := context.Background()

	sortedSources := sortHashListAscending(sourceHashes)

	interactions := make([]*eventsv1.CoordinationRingInteraction, len(successorEdges))
	for i, edge := range successorEdges {
		interactions[i] = &eventsv1.CoordinationRingInteraction{
			ActorA: edge[0],
			ActorB: edge[1],
		}
	}

	formation := &eventsv1.CoordinationRingFormation{
		PatternSignature:   patternSignature,
		PatternParameters:  "endpoint_window_seconds=60;min_cohort_size=3;split-successor",
		FormationAt:        formationAt,
		SourceEventHashes:  sortedSources,
		Interactions:       interactions,
		DirectInfluencedBy: nil,
		ClosureHashes:      nil,
		Confidence:         0.75,
		EvidentialIndependence: &commonv1.EvidentialIndependence{
			Numerator:   1,
			Denominator: 1,
		},
	}

	payload, hash, err := canonical.MarshalAndHash(formation)
	if err != nil {
		t.Fatalf("MarshalAndHash split successor: %v", err)
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
		t.Fatalf("substrate.Append split successor: %v", err)
	}
	return hash
}

// TestSplitCoordinationRing_FromF3CandidateAntecedent exercises the
// full F3 → 1 formation → operator-elected split into 2 successors
// arc for CoordinationRing. Mirrors §0167 + §0181 + §0184 on the
// CoordinationRing subtype side.
func TestSplitCoordinationRing_FromF3CandidateAntecedent(t *testing.T) {
	sub, in := newCoordinationSplitE2ESubstrate(t)
	ctx := context.Background()

	// Step 1: Inject 4 NetworkObservation records sharing endpoint +
	// time bucket → 1 candidate with 4 actors + 6 edges (4*3/2).
	const bucketStart = int64(1716120000_000_000_000)
	appendNetworkObs(t, in, "actor-suspect-1", "10.0.0.20:443", bucketStart)
	appendNetworkObs(t, in, "actor-suspect-2", "10.0.0.20:443", bucketStart+10_000_000_000)
	appendNetworkObs(t, in, "actor-suspect-3", "10.0.0.20:443", bucketStart+20_000_000_000)
	appendNetworkObs(t, in, "actor-suspect-4", "10.0.0.20:443", bucketStart+30_000_000_000)

	observations, err := collectNetworkObservations(ctx, sub)
	if err != nil {
		t.Fatalf("collectNetworkObservations: %v", err)
	}
	sig := &signatures.EndpointCoVisitV1{}
	result, err := sig.EvaluateNetwork(ctx, observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("candidates count: got %d want 1", len(result.Candidates))
	}
	candidate := result.Candidates[0]
	if len(candidate.SourceHashes) != 4 {
		t.Fatalf("candidate SourceHashes count: got %d want 4", len(candidate.SourceHashes))
	}
	if len(candidate.Interactions) != 6 {
		t.Fatalf("candidate Interactions count: got %d want 6 (4 actors → 6 edges)", len(candidate.Interactions))
	}

	antecedentHash := commitCoordinationRingFormationFromCandidate(t, sub, candidate, 1716120100000000000)

	// Step 4: Commit 2 successor formations. Edge set + source hashes
	// partitioned across successors. Sub-rings each preserve §0070
	// canonicalization internally.
	succAEdges := [][2]string{{"actor-suspect-1", "actor-suspect-2"}}
	succBEdges := [][2]string{
		{"actor-suspect-3", "actor-suspect-4"},
	}
	succAHash := commitCoordinationRingSplitSuccessor(t, sub, succAEdges,
		[][]byte{candidate.SourceHashes[0], candidate.SourceHashes[1]},
		"endpoint_co_visit_v1+split:ring-a",
		1716120200000000000)
	succBHash := commitCoordinationRingSplitSuccessor(t, sub, succBEdges,
		[][]byte{candidate.SourceHashes[2], candidate.SourceHashes[3]},
		"endpoint_co_visit_v1+split:ring-b",
		1716120300000000000)

	if succAHash == succBHash {
		t.Fatal("successor hashes A and B must differ (test fixture invariant)")
	}
	if succAHash == antecedentHash || succBHash == antecedentHash {
		t.Fatal("successor hashes must differ from antecedent (test fixture invariant)")
	}

	splitAt := int64(1716120400000000000)
	now := func() time.Time { return time.Unix(0, splitAt) }
	splitReport, err := hypothesis.SplitCoordinationRing(ctx, sub, hypothesis.CoordinationRingSplitOptions{
		AntecedentFormationHash:  antecedentHash,
		SuccessorFormationHashes: [][32]byte{succAHash, succBHash},
		SplitAt:                  splitAt,
		Reason:                   "e2e test split (CoordinationRing): antecedent contained two distinct sub-rings",
	}, now)
	if err != nil {
		t.Fatalf("hypothesis.SplitCoordinationRing: %v", err)
	}
	if splitReport.SplitEventHashHex == "" {
		t.Fatal("SplitReport.SplitEventHashHex is empty")
	}
	if splitReport.AlreadySplit {
		t.Error("AlreadySplit: got true want false (fresh split)")
	}

	var splitHash [32]byte
	if _, err := hexDecodeInto(splitReport.SplitEventHashHex, splitHash[:]); err != nil {
		t.Fatalf("decode split hash hex: %v", err)
	}
	splitRow, err := sub.LookupRow(ctx, splitHash)
	if err != nil {
		t.Fatalf("LookupRow split: %v", err)
	}
	if splitRow.MessageType != "ghosttrace.events.v1.CoordinationRingSplit" {
		t.Errorf("split MessageType: got %q want CoordinationRingSplit", splitRow.MessageType)
	}

	splitPayload, err := sub.ReadBlob(ctx, splitHash)
	if err != nil {
		t.Fatalf("ReadBlob split: %v", err)
	}
	gotSplit := &eventsv1.CoordinationRingSplit{}
	if err := proto.Unmarshal(splitPayload, gotSplit); err != nil {
		t.Fatalf("Unmarshal split: %v", err)
	}

	if !bytes.Equal(gotSplit.AntecedentFormationEventHash, antecedentHash[:]) {
		t.Error("split AntecedentFormationEventHash does not match committed antecedent")
	}

	if len(gotSplit.SuccessorFormationEventHashes) != 2 {
		t.Fatalf("split SuccessorFormationEventHashes count: got %d want 2", len(gotSplit.SuccessorFormationEventHashes))
	}
	containsA := false
	containsB := false
	for _, succ := range gotSplit.SuccessorFormationEventHashes {
		if bytes.Equal(succ, succAHash[:]) {
			containsA = true
		}
		if bytes.Equal(succ, succBHash[:]) {
			containsB = true
		}
	}
	if !containsA {
		t.Error("split successors do not contain succA hash")
	}
	if !containsB {
		t.Error("split successors do not contain succB hash")
	}

	if gotSplit.Reason == "" {
		t.Error("split Reason: got empty want non-empty")
	}
	if gotSplit.SplitAt != splitAt {
		t.Errorf("split SplitAt: got %d want %d", gotSplit.SplitAt, splitAt)
	}

	for _, h := range [][32]byte{antecedentHash, succAHash, succBHash, splitHash} {
		if _, err := sub.LookupRow(ctx, h); err != nil {
			t.Errorf("LookupRow %x: %v", h[:8], err)
		}
	}

	t.Logf("split E2E (CoordinationRing): antecedent=%x succA=%x succB=%x split=%x", antecedentHash[:4], succAHash[:4], succBHash[:4], splitHash[:4])
}

// TestSplitCoordinationRing_RejectsInsufficientSuccessors confirms
// the §0050 ErrSplitInsufficientSuccessors sentinel surfaces at the
// integration-path level for CoordinationRing. Mirrors §0167 + §0181 +
// §0184.
func TestSplitCoordinationRing_RejectsInsufficientSuccessors(t *testing.T) {
	sub, in := newCoordinationSplitE2ESubstrate(t)
	ctx := context.Background()

	const bucketStart = int64(1716120000_000_000_000)
	appendNetworkObs(t, in, "actor-x-1", "10.0.0.21:443", bucketStart)
	appendNetworkObs(t, in, "actor-x-2", "10.0.0.21:443", bucketStart+10_000_000_000)
	appendNetworkObs(t, in, "actor-x-3", "10.0.0.21:443", bucketStart+20_000_000_000)

	observations, err := collectNetworkObservations(ctx, sub)
	if err != nil {
		t.Fatalf("collectNetworkObservations: %v", err)
	}
	sig := &signatures.EndpointCoVisitV1{}
	result, err := sig.EvaluateNetwork(ctx, observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	antecedentHash := commitCoordinationRingFormationFromCandidate(t, sub, result.Candidates[0], 1716120100000000000)
	succHash := commitCoordinationRingSplitSuccessor(t, sub,
		[][2]string{{"actor-x-1", "actor-x-2"}},
		[][]byte{result.Candidates[0].SourceHashes[0]},
		"endpoint_co_visit_v1+split:lone-successor",
		1716120200000000000)

	_, err = hypothesis.SplitCoordinationRing(ctx, sub, hypothesis.CoordinationRingSplitOptions{
		AntecedentFormationHash:  antecedentHash,
		SuccessorFormationHashes: [][32]byte{succHash},
		SplitAt:                  1716120300000000000,
	}, nil)
	if err == nil {
		t.Fatal("expected ErrSplitInsufficientSuccessors, got nil")
	}
	if !errors.Is(err, hypothesis.ErrSplitInsufficientSuccessors) {
		t.Errorf("expected ErrSplitInsufficientSuccessors, got %v", err)
	}
}

// TestSplitCoordinationRing_RejectsAntecedentInSuccessorSet confirms
// the §0050 ErrSplitSuccessorsNotDistinct sentinel surfaces at the
// integration-path level. Mirrors §0167 + §0181 + §0184.
func TestSplitCoordinationRing_RejectsAntecedentInSuccessorSet(t *testing.T) {
	sub, in := newCoordinationSplitE2ESubstrate(t)
	ctx := context.Background()

	const bucketStart = int64(1716120000_000_000_000)
	appendNetworkObs(t, in, "actor-y-1", "10.0.0.22:443", bucketStart)
	appendNetworkObs(t, in, "actor-y-2", "10.0.0.22:443", bucketStart+10_000_000_000)
	appendNetworkObs(t, in, "actor-y-3", "10.0.0.22:443", bucketStart+20_000_000_000)

	observations, err := collectNetworkObservations(ctx, sub)
	if err != nil {
		t.Fatalf("collectNetworkObservations: %v", err)
	}
	sig := &signatures.EndpointCoVisitV1{}
	result, err := sig.EvaluateNetwork(ctx, observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	antecedentHash := commitCoordinationRingFormationFromCandidate(t, sub, result.Candidates[0], 1716120100000000000)
	succHash := commitCoordinationRingSplitSuccessor(t, sub,
		[][2]string{{"actor-y-1", "actor-y-2"}},
		[][]byte{result.Candidates[0].SourceHashes[0]},
		"endpoint_co_visit_v1+split:legit-successor",
		1716120200000000000)

	_, err = hypothesis.SplitCoordinationRing(ctx, sub, hypothesis.CoordinationRingSplitOptions{
		AntecedentFormationHash:  antecedentHash,
		SuccessorFormationHashes: [][32]byte{succHash, antecedentHash},
		SplitAt:                  1716120300000000000,
	}, nil)
	if err == nil {
		t.Fatal("expected ErrSplitSuccessorsNotDistinct, got nil")
	}
	if !errors.Is(err, hypothesis.ErrSplitSuccessorsNotDistinct) {
		t.Errorf("expected ErrSplitSuccessorsNotDistinct, got %v", err)
	}
}
