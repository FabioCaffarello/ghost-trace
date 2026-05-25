// End-to-end integration test exercising the AutomationGroupDissolution
// equivalent (hypothesis.Dissolve for BehavioralCluster) lifecycle
// operation from F3-derived candidates per §0180. Mirrors §0166 on
// the BehavioralCluster subtype side. Fifth PR in the §0176-opened
// arc; extends along the UNARY CROSS-FORMATION lifecycle axis.
//
// Per glossary + lifecycle-semantics.md: dissolution is DISTINGUISHED
// from demotion. Demotion (§0178) withdraws operational use; dissolution
// recognizes non-existence of the underlying phenomenon. Dissolve does
// NOT require prior promotion — formation may be dissolved directly.
// This test exercises that direct path explicitly.
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

func newDissolveE2ESubstrate(t *testing.T) (*substrate.Substrate, *ingest.Ingester) {
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

// TestDissolveBehavioralCluster_FromF3CandidateFormation exercises
// the full F3 → formation → dissolution arc (NO promotion step —
// formation may be dissolved directly per lifecycle-semantics.md).
// Mirrors §0166 on the BehavioralCluster subtype side.
func TestDissolveBehavioralCluster_FromF3CandidateFormation(t *testing.T) {
	sub, in := newDissolveE2ESubstrate(t)
	ctx := context.Background()

	// Step 1: Inject above-threshold BehavioralObservation records.
	ivs := []uint64{50_000_000, 100_000_000, 0, 100_000_000, 100_000_000, 150_000_000}
	appendKeystrokeObs(t, in, "actor-suspect-1", ivs, 1)
	appendKeystrokeObs(t, in, "actor-suspect-2", ivs, 2)
	appendKeystrokeObs(t, in, "actor-suspect-3", ivs, 3)

	// Step 2: F3 signature → 1 multi-actor candidate.
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

	// Step 3: Operator-elected formation commit via §0176 helper.
	formationAt := int64(1716120100000000000)
	formationHash := commitBehavioralClusterFormationFromCandidate(t, sub, result.Candidates[0], formationAt)

	// Step 4: Operator-elected dissolution commit. NO promotion in
	// between — formation may be dissolved directly per lifecycle-
	// semantics.md.
	dissolvedAt := int64(1716120200000000000)
	now := func() time.Time { return time.Unix(0, dissolvedAt) }
	dissolveReport, err := hypothesis.Dissolve(ctx, sub, hypothesis.DissolveOptions{
		FormationEventHash: formationHash,
		DissolvedAt:        dissolvedAt,
		Reason:             "e2e test dissolution (BehavioralCluster): F3 candidate recognized as not corresponding to underlying phenomenon",
	}, now)
	if err != nil {
		t.Fatalf("hypothesis.Dissolve: %v", err)
	}
	if dissolveReport.DissolutionEventHashHex == "" {
		t.Fatal("DissolveReport.DissolutionEventHashHex is empty")
	}
	if dissolveReport.AlreadyDissolved {
		t.Error("AlreadyDissolved: got true want false (fresh dissolution)")
	}

	// Step 5: Verify both events in substrate.
	formationRow, err := sub.LookupRow(ctx, formationHash)
	if err != nil {
		t.Fatalf("LookupRow formation: %v", err)
	}
	if formationRow.MessageType != "ghosttrace.events.v1.BehavioralClusterFormation" {
		t.Errorf("formation MessageType: got %q want BehavioralClusterFormation", formationRow.MessageType)
	}
	var dissolutionHash [32]byte
	if _, err := hexDecodeInto(dissolveReport.DissolutionEventHashHex, dissolutionHash[:]); err != nil {
		t.Fatalf("decode dissolution hash hex: %v", err)
	}
	dissolutionRow, err := sub.LookupRow(ctx, dissolutionHash)
	if err != nil {
		t.Fatalf("LookupRow dissolution: %v", err)
	}
	if dissolutionRow.MessageType != "ghosttrace.events.v1.BehavioralClusterDissolution" {
		t.Errorf("dissolution MessageType: got %q want BehavioralClusterDissolution", dissolutionRow.MessageType)
	}

	// Step 6: Verify dissolution proto references formation hash.
	dissolutionPayload, err := sub.ReadBlob(ctx, dissolutionHash)
	if err != nil {
		t.Fatalf("ReadBlob dissolution: %v", err)
	}
	gotDissolution := &eventsv1.BehavioralClusterDissolution{}
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

	t.Logf("dissolution E2E (BehavioralCluster): formation=%x dissolution=%x", formationHash[:4], dissolutionHash[:4])
}

// TestDissolveBehavioralCluster_IdempotencyUnderRepeatedCommit
// confirms content-hash idempotency: second dissolution commit with
// identical opts surfaces AlreadyDissolved=true + identical hash hex.
// Mirrors §0166 on the BehavioralCluster subtype side.
func TestDissolveBehavioralCluster_IdempotencyUnderRepeatedCommit(t *testing.T) {
	sub, in := newDissolveE2ESubstrate(t)
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
	formationHash := commitBehavioralClusterFormationFromCandidate(t, sub, result.Candidates[0], 1716120100000000000)

	dissolvedAt := int64(1716120200000000000)
	now := func() time.Time { return time.Unix(0, dissolvedAt) }
	opts := hypothesis.DissolveOptions{
		FormationEventHash: formationHash,
		DissolvedAt:        dissolvedAt,
		Reason:             "idempotency test (BehavioralCluster)",
	}

	first, err := hypothesis.Dissolve(ctx, sub, opts, now)
	if err != nil {
		t.Fatalf("first Dissolve: %v", err)
	}
	if first.AlreadyDissolved {
		t.Error("first call AlreadyDissolved: got true want false")
	}

	second, err := hypothesis.Dissolve(ctx, sub, opts, now)
	if err != nil {
		t.Fatalf("second Dissolve: %v", err)
	}
	if !second.AlreadyDissolved {
		t.Error("second call AlreadyDissolved: got false want true (content-hash should collide)")
	}
	if first.DissolutionEventHashHex != second.DissolutionEventHashHex {
		t.Errorf("hash hex mismatch: first=%s second=%s (should be identical)",
			first.DissolutionEventHashHex, second.DissolutionEventHashHex)
	}
}
