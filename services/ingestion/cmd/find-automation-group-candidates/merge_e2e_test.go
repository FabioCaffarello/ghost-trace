// End-to-end integration test exercising the AutomationGroupMerge
// lifecycle operation from F3-derived candidates per §0165. Extends
// the §0160 demote E2E baseline (lifecycle axis: formation →
// promotion → demotion) with the merge operation: two F3-derived
// formations recognized as the same underlying phenomenon →
// operator-elected merge commit referencing both antecedents + a
// produced formation.
//
// Continues the §0157 F3-derived-formation pattern along the
// lifecycle axis. §0160 covered the linear lifecycle (formation →
// promotion → demotion); §0165 extends to the cross-formation
// lifecycle operation (merge). Together they cover the full §0011
// staged-combination AutomationGroup lifecycle from F3-derived
// candidate output.
//
// Scope per §0165: structural connectivity validation across the
// full merge arc, NOT exhaustive merge-semantic coverage. Per-
// operation merge semantics (validation errors, antecedent
// symmetry, idempotency under reorder) are covered by the
// hypothesis package's own unit test suite; this test validates
// only the F3-derived → merge integration path.
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

// commitMergedFormation materializes a "produced" AutomationGroupFormation
// representing the merged hypothesis of two antecedent F3 candidates.
// Mirrors operator-elected commit at substrate level: actor_refs is the
// merged union; source_event_hashes is the sorted union of antecedents'
// sources; pattern_signature carries a synthetic identifier indicating
// the merged origin. Confidence + EvidentialIndependence are operator-
// supplied per §2.6 BC3.
func commitMergedFormation(t *testing.T, sub *substrate.Substrate, actorRefs []string, sourceHashes [][]byte, formationAt int64) [32]byte {
	t.Helper()
	ctx := context.Background()

	// Sort source hashes ascending per §0139 hash-list discipline.
	sortedSources := sortHashListAscending(sourceHashes)

	formation := &eventsv1.AutomationGroupFormation{
		PatternSignature:   "cdp_marker_density_v1+merged",
		PatternParameters:  "threshold=2;merged",
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

// TestMergeAutomationGroup_FromF3CandidateAntecedents exercises the
// full F3 → 2 formations → operator-elected merge arc:
//
//	Step 1: Inject BrowserObservation for 2 distinct actors above threshold.
//	Step 2: Run F3 signature → 2 candidates.
//	Step 3: Commit formation_A (from candidate_a) + formation_B (from
//	        candidate_b) — both AutomationGroupFormation.
//	Step 4: Commit formation_P (produced merged hypothesis with combined
//	        actor_refs + union source_hashes).
//	Step 5: Operator-elected merge via hypothesis.MergeAutomationGroup
//	        with (formation_A_hash, formation_B_hash, formation_P_hash).
//	Step 6: Verify merge event committed + references all three formations
//	        + antecedents are sorted ascending per §0049 symmetric-
//	        relation discipline.
func TestMergeAutomationGroup_FromF3CandidateAntecedents(t *testing.T) {
	sub, in := newMergeE2ESubstrate(t)
	ctx := context.Background()

	// Step 1: Inject for 2 distinct actors above threshold.
	appendBrowserObs(t, in, "actor-suspect-a", []string{"navigator.webdriver=true"}, 3)
	appendBrowserObs(t, in, "actor-suspect-b", []string{"$cdc_test"}, 3)

	// Step 2: Run F3 signature → 2 candidates.
	observations, err := collectBrowserObservations(ctx, sub)
	if err != nil {
		t.Fatalf("collectBrowserObservations: %v", err)
	}
	sig := &signatures.CDPMarkerDensityV1{}
	result, err := sig.EvaluateBrowser(ctx, observations)
	if err != nil {
		t.Fatalf("EvaluateBrowser: %v", err)
	}
	if len(result.Candidates) != 2 {
		t.Fatalf("candidates count: got %d want 2", len(result.Candidates))
	}
	candA := findCandidateByActor(t, result.Candidates, "actor-suspect-a")
	candB := findCandidateByActor(t, result.Candidates, "actor-suspect-b")

	// Step 3: Commit two F3-derived formations. Distinct formationAt
	// ensures unique content-hashes.
	hashA := commitFormationFromCandidate(t, sub, candA, 1716120100000000000)
	hashB := commitFormationFromCandidate(t, sub, candB, 1716120200000000000)
	if hashA == hashB {
		t.Fatal("formation hashes A and B must differ for merge to be valid")
	}

	// Step 4: Operator constructs merged-hypothesis formation_P with
	// combined actor_refs + union source_hashes. This is the operator-
	// elected representation of "these two F3 candidates describe the
	// same underlying phenomenon" per §2.5 BC4 merge semantics.
	mergedActorRefs := []string{"actor-suspect-a", "actor-suspect-b"}
	mergedSources := make([][]byte, 0, len(candA.SourceHashes)+len(candB.SourceHashes))
	mergedSources = append(mergedSources, candA.SourceHashes...)
	mergedSources = append(mergedSources, candB.SourceHashes...)
	hashP := commitMergedFormation(t, sub, mergedActorRefs, mergedSources, 1716120300000000000)
	if hashP == hashA || hashP == hashB {
		t.Fatal("produced formation hash P must differ from antecedents A and B")
	}

	// Step 5: Operator-elected merge commit.
	mergedAt := int64(1716120400000000000)
	now := func() time.Time { return time.Unix(0, mergedAt) }
	mergeReport, err := hypothesis.MergeAutomationGroup(ctx, sub, hypothesis.AutomationGroupMergeOptions{
		AntecedentAFormationHash: hashA,
		AntecedentBFormationHash: hashB,
		ProducedFormationHash:    hashP,
		MergedAt:                 mergedAt,
		Reason:                   "e2e test merge: F3 candidates a + b describe same phenomenon",
	}, now)
	if err != nil {
		t.Fatalf("MergeAutomationGroup: %v", err)
	}
	if mergeReport.MergeEventHashHex == "" {
		t.Fatal("MergeReport.MergeEventHashHex is empty")
	}
	if mergeReport.AlreadyMerged {
		t.Error("AlreadyMerged: got true want false (fresh merge)")
	}

	// Step 6: Verify merge event landed in substrate + references all
	// three formations.
	var mergeHash [32]byte
	if _, err := hexDecodeInto(mergeReport.MergeEventHashHex, mergeHash[:]); err != nil {
		t.Fatalf("decode merge hash hex: %v", err)
	}
	mergeRow, err := sub.LookupRow(ctx, mergeHash)
	if err != nil {
		t.Fatalf("LookupRow merge: %v", err)
	}
	if mergeRow.MessageType != "ghosttrace.events.v1.AutomationGroupMerge" {
		t.Errorf("merge MessageType: got %q want AutomationGroupMerge", mergeRow.MessageType)
	}

	mergePayload, err := sub.ReadBlob(ctx, mergeHash)
	if err != nil {
		t.Fatalf("ReadBlob merge: %v", err)
	}
	gotMerge := &eventsv1.AutomationGroupMerge{}
	if err := proto.Unmarshal(mergePayload, gotMerge); err != nil {
		t.Fatalf("Unmarshal merge: %v", err)
	}
	if len(gotMerge.AntecedentFormationEventHashes) != 2 {
		t.Fatalf("merge AntecedentFormationEventHashes count: got %d want 2", len(gotMerge.AntecedentFormationEventHashes))
	}

	// Verify antecedents are ascending-sorted per §0049 symmetric-relation
	// discipline. The merge committed (hashA, hashB); the recorded order
	// MUST be sorted ascending regardless of caller order.
	a := gotMerge.AntecedentFormationEventHashes[0]
	b := gotMerge.AntecedentFormationEventHashes[1]
	if bytes.Compare(a, b) >= 0 {
		t.Error("merge antecedents not sorted ascending per §0049")
	}

	// Verify both antecedent hashes correspond to formation_A and
	// formation_B (regardless of which is first after sort).
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

	// Step 7: Verify substrate has formation_A, formation_B, formation_P,
	// merge event, plus paired ingestion events from observation commits.
	// This is the structural connectivity assertion for the full arc.
	for _, h := range [][32]byte{hashA, hashB, hashP, mergeHash} {
		if _, err := sub.LookupRow(ctx, h); err != nil {
			t.Errorf("LookupRow for committed hash %x: %v", h[:8], err)
		}
	}

	t.Logf("merge E2E: A=%x B=%x P=%x merge=%x", hashA[:4], hashB[:4], hashP[:4], mergeHash[:4])
}

// TestMergeAutomationGroup_RejectsIdenticalAntecedents confirms the
// merge layer rejects self-merge per ErrMergeAntecedentsIdentical
// sentinel. Mirrors the §0049 symmetric-relation merge constraint
// at the integration-test layer (the unit test in the hypothesis
// package covers the error sentinel directly; this confirms the
// constraint surfaces at the F3-derived integration path too).
func TestMergeAutomationGroup_RejectsIdenticalAntecedents(t *testing.T) {
	sub, in := newMergeE2ESubstrate(t)
	ctx := context.Background()

	appendBrowserObs(t, in, "actor-x", []string{"navigator.webdriver=true"}, 3)

	observations, err := collectBrowserObservations(ctx, sub)
	if err != nil {
		t.Fatalf("collectBrowserObservations: %v", err)
	}
	sig := &signatures.CDPMarkerDensityV1{}
	result, err := sig.EvaluateBrowser(ctx, observations)
	if err != nil {
		t.Fatalf("EvaluateBrowser: %v", err)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("candidates count: got %d want 1", len(result.Candidates))
	}
	hashA := commitFormationFromCandidate(t, sub, result.Candidates[0], 1716120100000000000)
	hashP := commitMergedFormation(t, sub, []string{"actor-x"}, result.Candidates[0].SourceHashes, 1716120200000000000)

	// Call merge with A == A; must error.
	_, err = hypothesis.MergeAutomationGroup(ctx, sub, hypothesis.AutomationGroupMergeOptions{
		AntecedentAFormationHash: hashA,
		AntecedentBFormationHash: hashA,
		ProducedFormationHash:    hashP,
		MergedAt:                 1716120300000000000,
	}, nil)
	if err == nil {
		t.Fatal("expected error for identical antecedents, got nil")
	}
}
