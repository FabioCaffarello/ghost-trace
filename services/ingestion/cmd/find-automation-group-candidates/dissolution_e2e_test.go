// End-to-end integration test exercising the AutomationGroupDissolution
// lifecycle operation from a F3-derived candidate per §0166. Extends
// the §0165 cross-formation lifecycle integration (merge) with the
// UNARY cross-formation operation: a single F3-derived formation
// recognized as not corresponding to any underlying phenomenon →
// operator-elected dissolution event referencing the formation.
//
// Continues the §0157 F3-derived-formation pattern along the
// lifecycle axis. §0160 covered linear lifecycle (formation →
// promotion → demotion); §0165 covered binary cross-formation
// lifecycle (merge: 2 antecedents + 1 produced); §0166 covers unary
// cross-formation lifecycle (dissolution: 1 formation → 1 dissolution
// event). Together they close the lifecycle-axis integration test
// coverage for the §0011 staged-combination AutomationGroup lifecycle,
// modulo split (which lands as a separate entry per §0165 MO1).
//
// Per glossary + lifecycle-semantics.md line 36: dissolution is
// DISTINGUISHED from demotion. Demotion (§0160) withdraws operational
// use of a promoted hypothesis; dissolution recognizes that the
// hypothesis no longer corresponds to any underlying phenomenon. This
// test exercises the latter — a formation may be dissolved directly
// without ever being promoted (per AutomationGroupDissolveOptions:
// "DissolveAutomationGroup does not require that the hypothesis was
// ever promoted; a formation may be dissolved directly").
//
// Scope per §0166: structural connectivity validation across the
// F3-derived → dissolution arc, NOT exhaustive per-operation
// dissolution semantics. Per-operation semantics (ErrTargetNotFound,
// ErrTargetWrongType, idempotency under reorder, AppendPair with
// Actor) are covered by the hypothesis package's own unit test suite.
package main

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/hypothesis"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/signatures"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

func newDissolutionE2ESubstrate(t *testing.T) (*substrate.Substrate, *ingest.Ingester) {
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

// TestDissolveAutomationGroup_FromF3CandidateFormation exercises the
// full F3 → formation → dissolution arc (NO promotion step — per
// lifecycle-semantics.md a formation may be dissolved directly):
//
//	Step 1: Inject BrowserObservation above threshold.
//	Step 2: Run F3 signature → 1 candidate.
//	Step 3: Operator-elected formation commit.
//	Step 4: Operator-elected dissolution commit (DIRECT — no promotion
//	        path required per §2.5 BC5 + lifecycle-semantics.md).
//	Step 5: Verify formation + dissolution in substrate.
//	Step 6: Verify dissolution proto references formation event hash.
//	Step 7: Verify dissolution report fields (hex hash, idempotency
//	        marker).
func TestDissolveAutomationGroup_FromF3CandidateFormation(t *testing.T) {
	sub, in := newDissolutionE2ESubstrate(t)
	ctx := context.Background()

	// Step 1: Inject BrowserObservation above threshold.
	appendBrowserObs(t, in, "actor-suspect", []string{"navigator.webdriver=true"}, 3)
	appendBrowserObs(t, in, "actor-suspect", []string{"$cdc_test"}, 2)

	// Step 2: Run F3 signature → 1 candidate.
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
	candidate := result.Candidates[0]

	// Step 3: Operator-elected formation commit.
	formationAt := int64(1716120100000000000)
	formationHash := commitFormationFromCandidate(t, sub, candidate, formationAt)

	// Step 4: Operator-elected dissolution commit. NO promotion in
	// between — per lifecycle-semantics.md a formation may be dissolved
	// directly (operator recognizes the hypothesis as not corresponding
	// to any underlying phenomenon BEFORE it ever reaches operational
	// use).
	dissolvedAt := int64(1716120200000000000)
	now := func() time.Time { return time.Unix(0, dissolvedAt) }
	dissolveReport, err := hypothesis.DissolveAutomationGroup(ctx, sub, hypothesis.AutomationGroupDissolveOptions{
		FormationEventHash: formationHash,
		DissolvedAt:        dissolvedAt,
		Reason:             "e2e test dissolution: F3 candidate recognized as not corresponding to underlying phenomenon",
	}, now)
	if err != nil {
		t.Fatalf("DissolveAutomationGroup: %v", err)
	}
	if dissolveReport.DissolutionEventHashHex == "" {
		t.Fatal("DissolveReport.DissolutionEventHashHex is empty")
	}
	if dissolveReport.AlreadyDissolved {
		t.Error("AlreadyDissolved: got true want false (fresh dissolution)")
	}

	// Step 5: Verify both lifecycle events landed in substrate.
	formationRow, err := sub.LookupRow(ctx, formationHash)
	if err != nil {
		t.Fatalf("LookupRow formation: %v", err)
	}
	if formationRow.MessageType != "ghosttrace.events.v1.AutomationGroupFormation" {
		t.Errorf("formation MessageType: got %q want AutomationGroupFormation", formationRow.MessageType)
	}
	var dissolutionHash [32]byte
	if _, err := hexDecodeInto(dissolveReport.DissolutionEventHashHex, dissolutionHash[:]); err != nil {
		t.Fatalf("decode dissolution hash hex: %v", err)
	}
	dissolutionRow, err := sub.LookupRow(ctx, dissolutionHash)
	if err != nil {
		t.Fatalf("LookupRow dissolution: %v", err)
	}
	if dissolutionRow.MessageType != "ghosttrace.events.v1.AutomationGroupDissolution" {
		t.Errorf("dissolution MessageType: got %q want AutomationGroupDissolution", dissolutionRow.MessageType)
	}

	// Step 6: Verify dissolution proto references formation event hash.
	dissolutionPayload, err := sub.ReadBlob(ctx, dissolutionHash)
	if err != nil {
		t.Fatalf("ReadBlob dissolution: %v", err)
	}
	gotDissolution := &eventsv1.AutomationGroupDissolution{}
	if err := proto.Unmarshal(dissolutionPayload, gotDissolution); err != nil {
		t.Fatalf("Unmarshal dissolution: %v", err)
	}
	if !bytes.Equal(gotDissolution.FormationEventHash, formationHash[:]) {
		t.Error("dissolution FormationEventHash does not match formation hash")
	}
	if gotDissolution.Reason == "" {
		t.Error("dissolution Reason: got empty want non-empty")
	}
	if gotDissolution.DissolvedAt != dissolvedAt {
		t.Errorf("dissolution DissolvedAt: got %d want %d", gotDissolution.DissolvedAt, dissolvedAt)
	}

	t.Logf("dissolution E2E: formation=%x dissolution=%x", formationHash[:4], dissolutionHash[:4])
}

// TestDissolveAutomationGroup_IdempotencyUnderRepeatedCommit confirms
// the second dissolution commit with identical content surfaces
// AlreadyDissolved=true (content-hash idempotency per §0048 +
// lifecycle-semantics.md). Mirrors §0165's RejectsIdenticalAntecedents
// pattern at the F3-derived integration path layer.
func TestDissolveAutomationGroup_IdempotencyUnderRepeatedCommit(t *testing.T) {
	sub, in := newDissolutionE2ESubstrate(t)
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
	formationHash := commitFormationFromCandidate(t, sub, result.Candidates[0], 1716120100000000000)

	dissolvedAt := int64(1716120200000000000)
	now := func() time.Time { return time.Unix(0, dissolvedAt) }
	opts := hypothesis.AutomationGroupDissolveOptions{
		FormationEventHash: formationHash,
		DissolvedAt:        dissolvedAt,
		Reason:             "idempotency test",
	}

	// First call → fresh dissolution.
	first, err := hypothesis.DissolveAutomationGroup(ctx, sub, opts, now)
	if err != nil {
		t.Fatalf("first DissolveAutomationGroup: %v", err)
	}
	if first.AlreadyDissolved {
		t.Error("first call AlreadyDissolved: got true want false")
	}

	// Second call with IDENTICAL opts → content-hash collision →
	// AlreadyDissolved=true. The dissolution event's content-hash is
	// deterministic given (FormationEventHash, DissolvedAt, Reason);
	// repeating with the same inputs produces the same hash; the
	// substrate's lookup-before-append path surfaces this as
	// AlreadyDissolved without writing a duplicate row.
	second, err := hypothesis.DissolveAutomationGroup(ctx, sub, opts, now)
	if err != nil {
		t.Fatalf("second DissolveAutomationGroup: %v", err)
	}
	if !second.AlreadyDissolved {
		t.Error("second call AlreadyDissolved: got false want true (content-hash should collide)")
	}
	if first.DissolutionEventHashHex != second.DissolutionEventHashHex {
		t.Errorf("hash hex mismatch: first=%s second=%s (should be identical)",
			first.DissolutionEventHashHex, second.DissolutionEventHashHex)
	}
}
