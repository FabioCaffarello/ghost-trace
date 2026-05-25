// End-to-end integration test exercising the AutomationGroupSplit
// equivalent (hypothesis.Split for BehavioralCluster) lifecycle
// operation from F3-derived candidates per §0181. Mirrors §0167 on
// the BehavioralCluster subtype side.
//
// CLOSES the §0176-opened BehavioralCluster lifecycle integration
// coverage arc: sixth and final PR. Extends along the K-ARY
// CROSS-FORMATION lifecycle axis (split: 1 antecedent → K≥2
// successors).
//
// Per §0176 MO1 + §0167: this is the structurally most complex
// cross-formation operation; requires K operator-synthesized
// successor formations + enforces two distinct §0050 sentinels
// (ErrSplitInsufficientSuccessors + ErrSplitSuccessorsNotDistinct).
//
// With §0181 landed, BehavioralCluster matches AutomationGroup's
// full §0011 lifecycle integration coverage from F3-derived
// candidates: formation (§0176) + promotion (§0178 linear arc) +
// demotion (§0178) + merge (§0179) + dissolution (§0180) + split
// (this entry).
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
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/observationcollector"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

func newSplitE2ESubstrate(t *testing.T) (*substrate.Substrate, *ingest.Ingester) {
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

// commitBehavioralClusterSplitSuccessor materializes a separately-
// committed BehavioralClusterFormation representing one of K
// successor hypotheses of a split. Mirrors §0167's
// commitSplitSuccessor on the BehavioralCluster subtype side. Each
// successor carries a distinct pattern_signature label per §0167 MO2
// (per-successor labels surface operator's split decision
// semantically).
func commitBehavioralClusterSplitSuccessor(t *testing.T, sub *substrate.Substrate, actorRefs []string, sourceHashes [][]byte, patternSignature string, formationAt int64) [32]byte {
	t.Helper()
	ctx := context.Background()

	sortedSources := sortHashListAscending(sourceHashes)

	formation := &eventsv1.BehavioralClusterFormation{
		PatternSignature:   patternSignature,
		PatternParameters:  "threshold=3;split-successor",
		ActorRefs:          actorRefs,
		FormationAt:        formationAt,
		SourceEventHashes:  sortedSources,
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

// TestSplitBehavioralCluster_FromF3CandidateAntecedent exercises the
// full F3 → 1 formation → operator-elected split into 2 successors
// arc for BehavioralCluster. Mirrors §0167 on the BehavioralCluster
// subtype side.
func TestSplitBehavioralCluster_FromF3CandidateAntecedent(t *testing.T) {
	sub, in := newSplitE2ESubstrate(t)
	ctx := context.Background()

	// Step 1: Inject 3 BehavioralObservation records with identical
	// fingerprint → 1 multi-actor candidate with 3 source hashes
	// available for subdivision across split successors.
	ivs := []uint64{50_000_000, 100_000_000, 0, 100_000_000, 100_000_000, 150_000_000}
	appendKeystrokeObs(t, in, "actor-suspect-1", ivs, 1)
	appendKeystrokeObs(t, in, "actor-suspect-2", ivs, 2)
	appendKeystrokeObs(t, in, "actor-suspect-3", ivs, 3)

	// Step 2: F3 → 1 candidate.
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
	if len(candidate.SourceHashes) != 3 {
		t.Fatalf("candidate SourceHashes count: got %d want 3", len(candidate.SourceHashes))
	}

	// Step 3: Commit antecedent formation via §0176 helper.
	antecedentHash := commitBehavioralClusterFormationFromCandidate(t, sub, candidate, 1716120100000000000)

	// Step 4: Commit 2 successor formations. Each carries a distinct
	// pattern_signature label per §0167 MO2 (per-successor labels
	// surface operator's split decision semantically). Source hashes
	// subdivided across successors; ActorRefs partitioned.
	succAHash := commitBehavioralClusterSplitSuccessor(t, sub, []string{"actor-suspect-1"},
		[][]byte{candidate.SourceHashes[0]},
		"keystroke_timing_clustering_v1+split:cluster-a",
		1716120200000000000)
	succBHash := commitBehavioralClusterSplitSuccessor(t, sub, []string{"actor-suspect-2", "actor-suspect-3"},
		[][]byte{candidate.SourceHashes[1], candidate.SourceHashes[2]},
		"keystroke_timing_clustering_v1+split:cluster-b",
		1716120300000000000)

	if succAHash == succBHash {
		t.Fatal("successor hashes A and B must differ (test fixture invariant)")
	}
	if succAHash == antecedentHash || succBHash == antecedentHash {
		t.Fatal("successor hashes must differ from antecedent (test fixture invariant)")
	}

	// Step 5: Operator-elected split via hypothesis.Split.
	splitAt := int64(1716120400000000000)
	now := func() time.Time { return time.Unix(0, splitAt) }
	splitReport, err := hypothesis.Split(ctx, sub, hypothesis.SplitOptions{
		AntecedentFormationHash:  antecedentHash,
		SuccessorFormationHashes: [][32]byte{succAHash, succBHash},
		SplitAt:                  splitAt,
		Reason:                   "e2e test split (BehavioralCluster): antecedent contained two distinct sub-clusters",
	}, now)
	if err != nil {
		t.Fatalf("hypothesis.Split: %v", err)
	}
	if splitReport.SplitEventHashHex == "" {
		t.Fatal("SplitReport.SplitEventHashHex is empty")
	}
	if splitReport.AlreadySplit {
		t.Error("AlreadySplit: got true want false (fresh split)")
	}

	// Step 6: Verify split event in substrate + references antecedent
	// + both successors.
	var splitHash [32]byte
	if _, err := hexDecodeInto(splitReport.SplitEventHashHex, splitHash[:]); err != nil {
		t.Fatalf("decode split hash hex: %v", err)
	}
	splitRow, err := sub.LookupRow(ctx, splitHash)
	if err != nil {
		t.Fatalf("LookupRow split: %v", err)
	}
	if splitRow.MessageType != "ghosttrace.events.v1.BehavioralClusterSplit" {
		t.Errorf("split MessageType: got %q want BehavioralClusterSplit", splitRow.MessageType)
	}

	splitPayload, err := sub.ReadBlob(ctx, splitHash)
	if err != nil {
		t.Fatalf("ReadBlob split: %v", err)
	}
	gotSplit := &eventsv1.BehavioralClusterSplit{}
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

	t.Logf("split E2E (BehavioralCluster): antecedent=%x succA=%x succB=%x split=%x", antecedentHash[:4], succAHash[:4], succBHash[:4], splitHash[:4])
}

// TestSplitBehavioralCluster_RejectsInsufficientSuccessors confirms
// the §0050 ErrSplitInsufficientSuccessors sentinel surfaces at the
// integration-path level for BehavioralCluster. Mirrors §0167 on the
// BehavioralCluster subtype side.
func TestSplitBehavioralCluster_RejectsInsufficientSuccessors(t *testing.T) {
	sub, in := newSplitE2ESubstrate(t)
	ctx := context.Background()

	ivs := []uint64{50_000_000, 100_000_000, 0, 100_000_000, 100_000_000, 150_000_000}
	appendKeystrokeObs(t, in, "actor-x-1", ivs, 1)
	appendKeystrokeObs(t, in, "actor-x-2", ivs, 2)
	appendKeystrokeObs(t, in, "actor-x-3", ivs, 3)

	observations, err := observationcollector.CollectBehavioral(ctx, sub)
	if err != nil {
		t.Fatalf("observationcollector.CollectBehavioral: %v", err)
	}
	sig := &signatures.KeystrokeTimingClusteringV1{}
	result, err := sig.EvaluateBehavioral(ctx, observations)
	if err != nil {
		t.Fatalf("EvaluateBehavioral: %v", err)
	}
	antecedentHash := commitBehavioralClusterFormationFromCandidate(t, sub, result.Candidates[0], 1716120100000000000)
	succHash := commitBehavioralClusterSplitSuccessor(t, sub, []string{"actor-x-1"},
		[][]byte{result.Candidates[0].SourceHashes[0]},
		"keystroke_timing_clustering_v1+split:lone-successor",
		1716120200000000000)

	_, err = hypothesis.Split(ctx, sub, hypothesis.SplitOptions{
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

// TestSplitBehavioralCluster_RejectsAntecedentInSuccessorSet
// confirms the §0050 ErrSplitSuccessorsNotDistinct sentinel surfaces
// at the integration-path level. Mirrors §0167 on the BehavioralCluster
// subtype side.
func TestSplitBehavioralCluster_RejectsAntecedentInSuccessorSet(t *testing.T) {
	sub, in := newSplitE2ESubstrate(t)
	ctx := context.Background()

	ivs := []uint64{50_000_000, 100_000_000, 0, 100_000_000, 100_000_000, 150_000_000}
	appendKeystrokeObs(t, in, "actor-y-1", ivs, 1)
	appendKeystrokeObs(t, in, "actor-y-2", ivs, 2)
	appendKeystrokeObs(t, in, "actor-y-3", ivs, 3)

	observations, err := observationcollector.CollectBehavioral(ctx, sub)
	if err != nil {
		t.Fatalf("observationcollector.CollectBehavioral: %v", err)
	}
	sig := &signatures.KeystrokeTimingClusteringV1{}
	result, err := sig.EvaluateBehavioral(ctx, observations)
	if err != nil {
		t.Fatalf("EvaluateBehavioral: %v", err)
	}
	antecedentHash := commitBehavioralClusterFormationFromCandidate(t, sub, result.Candidates[0], 1716120100000000000)
	succHash := commitBehavioralClusterSplitSuccessor(t, sub, []string{"actor-y-1"},
		[][]byte{result.Candidates[0].SourceHashes[0]},
		"keystroke_timing_clustering_v1+split:legit-successor",
		1716120200000000000)

	_, err = hypothesis.Split(ctx, sub, hypothesis.SplitOptions{
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
