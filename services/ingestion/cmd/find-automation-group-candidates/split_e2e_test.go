// End-to-end integration test exercising the AutomationGroupSplit
// lifecycle operation from a F3-derived candidate per §0167. Extends
// the §0166 unary cross-formation lifecycle (dissolution: 1 formation
// → 1 dissolution event) with the K-ARY cross-formation operation:
// one F3-derived formation recognized as containing multiple distinct
// phenomena → operator-elected split event referencing the antecedent
// + K≥2 separately-committed successor formations.
//
// Closes the §0011 staged-combination AutomationGroup lifecycle
// integration coverage from F3-derived candidates. Combined with
// §0157 (formation) + §0160 (promotion + demotion) + §0165 (binary
// cross-formation: merge) + §0166 (unary cross-formation: dissolution)
// + §0167 (k-ary cross-formation: split, this entry), every §0011
// lifecycle operation now has integration coverage from F3-derived
// candidate substrate state.
//
// Per §0165 MO1: cross-formation lifecycle operations differ in how
// many operator-synthesized successor formations they need. Merge
// needs ONE (the produced hypothesis); dissolution needs NONE; split
// needs K≥2 (the successor hypotheses). §0167 introduces
// commitSplitSuccessor as the per-successor synthesizer helper.
//
// Scope per §0167: structural connectivity validation across the
// F3-derived → split arc + two §0050 sentinel checks
// (insufficient-successors + antecedent-in-successor-set). Per-
// operation semantics (ErrTargetNotFound for missing hashes,
// ErrTargetWrongType for wrong message_type, AppendPair with Actor,
// idempotency under reorder) covered by the hypothesis package's
// own unit test suite.
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
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/observationcollector"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/signatures"
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

// commitSplitSuccessor materializes a separately-committed
// AutomationGroupFormation representing one of K successor hypotheses
// of a split. Distinct from §0165's commitMergedFormation in that
// each successor carries a distinct pattern_signature label
// indicating which sub-phenomenon it represents (the operator's
// split decision separates the antecedent into K distinct
// sub-phenomena).
//
// Per §0165 MO1: cross-formation integration tests must include a
// synthesizer helper for each post-operation record. Split needs K
// synthesizers (or one parameterized helper called K times); this
// helper is the parameterized form.
func commitSplitSuccessor(t *testing.T, sub *substrate.Substrate, actorRefs []string, sourceHashes [][]byte, patternSignature string, formationAt int64) [32]byte {
	t.Helper()
	ctx := context.Background()

	sortedSources := sortHashListAscending(sourceHashes)

	formation := &eventsv1.AutomationGroupFormation{
		PatternSignature:   patternSignature,
		PatternParameters:  "threshold=2;split-successor",
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

// TestSplitAutomationGroup_FromF3CandidateAntecedent exercises the
// full F3 → 1 formation → operator-elected split into 2 successors arc:
//
//	Step 1: Inject 2 BrowserObservation for a single actor (5 detections
//	        aggregate, above threshold).
//	Step 2: Run F3 signature → 1 antecedent candidate.
//	Step 3: Commit antecedent formation via §0157's commitFormationFromCandidate.
//	Step 4: Operator commits 2 successor formations via commitSplitSuccessor
//	        (each with distinct pattern_signature representing a
//	        distinct sub-phenomenon).
//	Step 5: Operator-elected split via hypothesis.SplitAutomationGroup
//	        with (antecedent, [succ_A, succ_B]).
//	Step 6: Verify split event committed + references antecedent + both
//	        successors exactly.
func TestSplitAutomationGroup_FromF3CandidateAntecedent(t *testing.T) {
	sub, in := newSplitE2ESubstrate(t)
	ctx := context.Background()

	// Step 1: Inject 2 observations for single actor.
	appendBrowserObs(t, in, "actor-suspect", []string{"navigator.webdriver=true"}, 3)
	appendBrowserObs(t, in, "actor-suspect", []string{"$cdc_test"}, 2)

	// Step 2: F3 signature → 1 candidate.
	observations, err := observationcollector.CollectBrowser(ctx, sub)
	if err != nil {
		t.Fatalf("observationcollector.CollectBrowser: %v", err)
	}
	sig := &signatures.CDPMarkerDensityV1{}
	result, err := sig.EvaluateBrowser(ctx, observations)
	if err != nil {
		t.Fatalf("EvaluateBrowser: %v", err)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("candidates count: got %d want 1", len(result.Candidates))
	}
	candidate := result.Candidates[0]
	if len(candidate.SourceHashes) != 2 {
		t.Fatalf("candidate SourceHashes count: got %d want 2 (two committed observations)", len(candidate.SourceHashes))
	}

	// Step 3: Commit antecedent formation.
	antecedentHash := commitFormationFromCandidate(t, sub, candidate, 1716120100000000000)

	// Step 4: Operator commits 2 split-successor formations. Each
	// carries a distinct pattern_signature label representing a
	// distinct sub-phenomenon ("webdriver-signal" vs "cdc-signal").
	// Source hashes are subdivided: successor A gets the first source,
	// successor B gets the second.
	succAHash := commitSplitSuccessor(t, sub, []string{"actor-suspect"},
		[][]byte{candidate.SourceHashes[0]},
		"cdp_marker_density_v1+split:webdriver-signal",
		1716120200000000000)
	succBHash := commitSplitSuccessor(t, sub, []string{"actor-suspect"},
		[][]byte{candidate.SourceHashes[1]},
		"cdp_marker_density_v1+split:cdc-signal",
		1716120300000000000)

	// Distinctness check (also enforced by SplitAutomationGroup but
	// asserted here for test self-clarity).
	if succAHash == succBHash {
		t.Fatal("successor hashes A and B must differ (test fixture invariant)")
	}
	if succAHash == antecedentHash || succBHash == antecedentHash {
		t.Fatal("successor hashes must differ from antecedent (test fixture invariant)")
	}

	// Step 5: Operator-elected split.
	splitAt := int64(1716120400000000000)
	now := func() time.Time { return time.Unix(0, splitAt) }
	splitReport, err := hypothesis.SplitAutomationGroup(ctx, sub, hypothesis.AutomationGroupSplitOptions{
		AntecedentFormationHash:  antecedentHash,
		SuccessorFormationHashes: [][32]byte{succAHash, succBHash},
		SplitAt:                  splitAt,
		Reason:                   "e2e test split: antecedent contained two distinct CDP sub-signals",
	}, now)
	if err != nil {
		t.Fatalf("SplitAutomationGroup: %v", err)
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
	if splitRow.MessageType != "ghosttrace.events.v1.AutomationGroupSplit" {
		t.Errorf("split MessageType: got %q want AutomationGroupSplit", splitRow.MessageType)
	}

	splitPayload, err := sub.ReadBlob(ctx, splitHash)
	if err != nil {
		t.Fatalf("ReadBlob split: %v", err)
	}
	gotSplit := &eventsv1.AutomationGroupSplit{}
	if err := proto.Unmarshal(splitPayload, gotSplit); err != nil {
		t.Fatalf("Unmarshal split: %v", err)
	}

	// Verify antecedent reference.
	if !bytes.Equal(gotSplit.AntecedentFormationEventHash, antecedentHash[:]) {
		t.Error("split AntecedentFormationEventHash does not match committed antecedent")
	}

	// Verify successor count + membership.
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

	// All four committed records must be present.
	for _, h := range [][32]byte{antecedentHash, succAHash, succBHash, splitHash} {
		if _, err := sub.LookupRow(ctx, h); err != nil {
			t.Errorf("LookupRow %x: %v", h[:8], err)
		}
	}

	t.Logf("split E2E: antecedent=%x succA=%x succB=%x split=%x", antecedentHash[:4], succAHash[:4], succBHash[:4], splitHash[:4])
}

// TestSplitAutomationGroup_RejectsInsufficientSuccessors confirms the
// §0050 ErrSplitInsufficientSuccessors sentinel surfaces at the
// integration-path level: split call with fewer than 2 successors →
// error. Mirrors the merge-side §0165 RejectsIdenticalAntecedents
// falsifiability check at the split-side sentinel.
func TestSplitAutomationGroup_RejectsInsufficientSuccessors(t *testing.T) {
	sub, in := newSplitE2ESubstrate(t)
	ctx := context.Background()

	appendBrowserObs(t, in, "actor-x", []string{"navigator.webdriver=true"}, 3)
	observations, err := observationcollector.CollectBrowser(ctx, sub)
	if err != nil {
		t.Fatalf("observationcollector.CollectBrowser: %v", err)
	}
	sig := &signatures.CDPMarkerDensityV1{}
	result, err := sig.EvaluateBrowser(ctx, observations)
	if err != nil {
		t.Fatalf("EvaluateBrowser: %v", err)
	}
	antecedentHash := commitFormationFromCandidate(t, sub, result.Candidates[0], 1716120100000000000)
	succHash := commitSplitSuccessor(t, sub, []string{"actor-x"},
		[][]byte{result.Candidates[0].SourceHashes[0]},
		"cdp_marker_density_v1+split:lone-successor",
		1716120200000000000)

	// Call split with ONLY 1 successor → must error per §0050.
	_, err = hypothesis.SplitAutomationGroup(ctx, sub, hypothesis.AutomationGroupSplitOptions{
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

// TestSplitAutomationGroup_RejectsAntecedentInSuccessorSet confirms
// the §0050 ErrSplitSuccessorsNotDistinct sentinel surfaces when the
// antecedent hash appears in the successor set. Mirrors the merge-
// side §0165 antecedents-distinct discipline on the split-side.
func TestSplitAutomationGroup_RejectsAntecedentInSuccessorSet(t *testing.T) {
	sub, in := newSplitE2ESubstrate(t)
	ctx := context.Background()

	appendBrowserObs(t, in, "actor-y", []string{"navigator.webdriver=true"}, 3)
	observations, err := observationcollector.CollectBrowser(ctx, sub)
	if err != nil {
		t.Fatalf("observationcollector.CollectBrowser: %v", err)
	}
	sig := &signatures.CDPMarkerDensityV1{}
	result, err := sig.EvaluateBrowser(ctx, observations)
	if err != nil {
		t.Fatalf("EvaluateBrowser: %v", err)
	}
	antecedentHash := commitFormationFromCandidate(t, sub, result.Candidates[0], 1716120100000000000)
	succHash := commitSplitSuccessor(t, sub, []string{"actor-y"},
		[][]byte{result.Candidates[0].SourceHashes[0]},
		"cdp_marker_density_v1+split:legit-successor",
		1716120200000000000)

	// Call split with antecedent hash also in successor set → must
	// error per §0050.
	_, err = hypothesis.SplitAutomationGroup(ctx, sub, hypothesis.AutomationGroupSplitOptions{
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
