// End-to-end integration test exercising the SplitCampaignHypothesis
// lifecycle operation from F3-derived candidates per §0184. Mirrors
// §0167 + §0181 on the CampaignHypothesis subtype side. Third PR-#2
// axis of the §0181 MO2 bundled CampaignHypothesis lifecycle arc;
// extends along the K-ARY CROSS-FORMATION lifecycle axis (split: 1
// antecedent → K≥2 successors).
//
// CLOSES the CampaignHypothesis lifecycle integration coverage arc
// opened at §0183: bundled PR #2 of 2. With §0184 landed,
// CampaignHypothesis matches AutomationGroup + BehavioralCluster's
// full §0011 lifecycle integration coverage from F3-derived
// candidates: formation + promotion (§0183 linear arc) + demotion
// (§0183) + merge (§0184) + dissolution (§0184) + split (this entry).
//
// Per §0182 + §0063 event-centric ontology: both antecedent + all K
// successor CampaignHypothesisFormation records DROP ActorRefs (proto
// has no actor_refs field). Structural distinction from §0167
// (AutomationGroup) + §0181 (BehavioralCluster) which commit ActorRefs.
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

func newCampaignSplitE2ESubstrate(t *testing.T) (*substrate.Substrate, *ingest.Ingester) {
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

// commitCampaignHypothesisSplitSuccessor materializes a separately-
// committed CampaignHypothesisFormation representing one of K
// successor hypotheses of a split. Mirrors §0167's commitSplitSuccessor
// + §0181's commitBehavioralClusterSplitSuccessor on the
// CampaignHypothesis subtype side. Each successor carries a distinct
// pattern_signature label per §0167 MO2 (per-successor labels surface
// operator's split decision semantically).
//
// Per §0182 + §0063 event-centric ontology: ActorRefs DROPPED at
// commit (proto has no actor_refs field). Operators recover actor
// attribution by walking the source_event_hashes back to Cat I
// observations.
func commitCampaignHypothesisSplitSuccessor(t *testing.T, sub *substrate.Substrate, sourceHashes [][]byte, patternSignature string, formationAt int64) [32]byte {
	t.Helper()
	ctx := context.Background()

	sortedSources := sortHashListAscending(sourceHashes)

	formation := &eventsv1.CampaignHypothesisFormation{
		PatternSignature:   patternSignature,
		PatternParameters:  "endpoint_window_seconds=60;min_cohort_size=3;split-successor",
		FormationAt:        formationAt,
		SourceEventHashes:  sortedSources,
		DirectInfluencedBy: nil,
		ClosureHashes:      nil,
		Confidence:         0.75,
		EvidentialIndependence: &commonv1.EvidentialIndependence{
			Numerator:   1,
			Denominator: 1,
		},
		// ActorRefs INTENTIONALLY DROPPED per §0182 + §0063.
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

// TestSplitCampaignHypothesis_FromF3CandidateAntecedent exercises the
// full F3 → 1 formation → operator-elected split into 2 successors
// arc for CampaignHypothesis. Mirrors §0167 + §0181 on the
// CampaignHypothesis subtype side.
func TestSplitCampaignHypothesis_FromF3CandidateAntecedent(t *testing.T) {
	sub, in := newCampaignSplitE2ESubstrate(t)
	ctx := context.Background()

	// Step 1: Inject 3 NetworkObservation records sharing endpoint +
	// time bucket → 1 candidate with 3 source hashes available for
	// subdivision across split successors.
	const bucketStart = int64(1716120000_000_000_000)
	appendNetworkObs(t, in, "actor-suspect-1", "10.0.0.20:443", bucketStart)
	appendNetworkObs(t, in, "actor-suspect-2", "10.0.0.20:443", bucketStart+10_000_000_000)
	appendNetworkObs(t, in, "actor-suspect-3", "10.0.0.20:443", bucketStart+20_000_000_000)

	// Step 2: F3 → 1 candidate.
	observations, err := collectNetworkObservations(ctx, sub)
	if err != nil {
		t.Fatalf("collectNetworkObservations: %v", err)
	}
	sig := &signatures.TemporalEndpointCohortV1{}
	result, err := sig.EvaluateNetwork(ctx, observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("candidates count: got %d want 1", len(result.Candidates))
	}
	candidate := result.Candidates[0]
	if len(candidate.SourceHashes) != 3 {
		t.Fatalf("candidate SourceHashes count: got %d want 3", len(candidate.SourceHashes))
	}

	// Step 3: Commit antecedent formation via §0183 foundation helper.
	antecedentHash := commitCampaignHypothesisFormationFromCandidate(t, sub, candidate, 1716120100000000000)

	// Step 4: Commit 2 successor formations. Each carries a distinct
	// pattern_signature label per §0167 MO2. Source hashes subdivided
	// across successors. NO actor_refs partitioning (proto field absent).
	succAHash := commitCampaignHypothesisSplitSuccessor(t, sub,
		[][]byte{candidate.SourceHashes[0]},
		"temporal_endpoint_cohort_v1+split:campaign-a",
		1716120200000000000)
	succBHash := commitCampaignHypothesisSplitSuccessor(t, sub,
		[][]byte{candidate.SourceHashes[1], candidate.SourceHashes[2]},
		"temporal_endpoint_cohort_v1+split:campaign-b",
		1716120300000000000)

	if succAHash == succBHash {
		t.Fatal("successor hashes A and B must differ (test fixture invariant)")
	}
	if succAHash == antecedentHash || succBHash == antecedentHash {
		t.Fatal("successor hashes must differ from antecedent (test fixture invariant)")
	}

	// Step 5: Operator-elected split via SplitCampaignHypothesis
	// (subtype-suffixed per §0178 MO1).
	splitAt := int64(1716120400000000000)
	now := func() time.Time { return time.Unix(0, splitAt) }
	splitReport, err := hypothesis.SplitCampaignHypothesis(ctx, sub, hypothesis.CampaignHypothesisSplitOptions{
		AntecedentFormationHash:  antecedentHash,
		SuccessorFormationHashes: [][32]byte{succAHash, succBHash},
		SplitAt:                  splitAt,
		Reason:                   "e2e test split (CampaignHypothesis): antecedent contained two distinct sub-campaigns",
	}, now)
	if err != nil {
		t.Fatalf("hypothesis.SplitCampaignHypothesis: %v", err)
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
	if splitRow.MessageType != "ghosttrace.events.v1.CampaignHypothesisSplit" {
		t.Errorf("split MessageType: got %q want CampaignHypothesisSplit", splitRow.MessageType)
	}

	splitPayload, err := sub.ReadBlob(ctx, splitHash)
	if err != nil {
		t.Fatalf("ReadBlob split: %v", err)
	}
	gotSplit := &eventsv1.CampaignHypothesisSplit{}
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

	t.Logf("split E2E (CampaignHypothesis): antecedent=%x succA=%x succB=%x split=%x", antecedentHash[:4], succAHash[:4], succBHash[:4], splitHash[:4])
}

// TestSplitCampaignHypothesis_RejectsInsufficientSuccessors confirms
// the §0050 ErrSplitInsufficientSuccessors sentinel surfaces at the
// integration-path level for CampaignHypothesis. Mirrors §0167 +
// §0181.
func TestSplitCampaignHypothesis_RejectsInsufficientSuccessors(t *testing.T) {
	sub, in := newCampaignSplitE2ESubstrate(t)
	ctx := context.Background()

	const bucketStart = int64(1716120000_000_000_000)
	appendNetworkObs(t, in, "actor-x-1", "10.0.0.21:443", bucketStart)
	appendNetworkObs(t, in, "actor-x-2", "10.0.0.21:443", bucketStart+10_000_000_000)
	appendNetworkObs(t, in, "actor-x-3", "10.0.0.21:443", bucketStart+20_000_000_000)

	observations, err := collectNetworkObservations(ctx, sub)
	if err != nil {
		t.Fatalf("collectNetworkObservations: %v", err)
	}
	sig := &signatures.TemporalEndpointCohortV1{}
	result, err := sig.EvaluateNetwork(ctx, observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	antecedentHash := commitCampaignHypothesisFormationFromCandidate(t, sub, result.Candidates[0], 1716120100000000000)
	succHash := commitCampaignHypothesisSplitSuccessor(t, sub,
		[][]byte{result.Candidates[0].SourceHashes[0]},
		"temporal_endpoint_cohort_v1+split:lone-successor",
		1716120200000000000)

	_, err = hypothesis.SplitCampaignHypothesis(ctx, sub, hypothesis.CampaignHypothesisSplitOptions{
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

// TestSplitCampaignHypothesis_RejectsAntecedentInSuccessorSet
// confirms the §0050 ErrSplitSuccessorsNotDistinct sentinel surfaces
// at the integration-path level. Mirrors §0167 + §0181.
func TestSplitCampaignHypothesis_RejectsAntecedentInSuccessorSet(t *testing.T) {
	sub, in := newCampaignSplitE2ESubstrate(t)
	ctx := context.Background()

	const bucketStart = int64(1716120000_000_000_000)
	appendNetworkObs(t, in, "actor-y-1", "10.0.0.22:443", bucketStart)
	appendNetworkObs(t, in, "actor-y-2", "10.0.0.22:443", bucketStart+10_000_000_000)
	appendNetworkObs(t, in, "actor-y-3", "10.0.0.22:443", bucketStart+20_000_000_000)

	observations, err := collectNetworkObservations(ctx, sub)
	if err != nil {
		t.Fatalf("collectNetworkObservations: %v", err)
	}
	sig := &signatures.TemporalEndpointCohortV1{}
	result, err := sig.EvaluateNetwork(ctx, observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	antecedentHash := commitCampaignHypothesisFormationFromCandidate(t, sub, result.Candidates[0], 1716120100000000000)
	succHash := commitCampaignHypothesisSplitSuccessor(t, sub,
		[][]byte{result.Candidates[0].SourceHashes[0]},
		"temporal_endpoint_cohort_v1+split:legit-successor",
		1716120200000000000)

	_, err = hypothesis.SplitCampaignHypothesis(ctx, sub, hypothesis.CampaignHypothesisSplitOptions{
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
