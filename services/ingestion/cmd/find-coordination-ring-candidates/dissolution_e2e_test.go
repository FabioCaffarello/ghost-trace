// End-to-end integration test exercising the DissolveCoordinationRing
// lifecycle operation from F3-derived candidates per §0187. Mirrors
// §0166 + §0180 + §0184 on the CoordinationRing subtype side. Second
// PR-#2 axis of the §0184 MO2 bundled CoordinationRing lifecycle arc;
// extends along the UNARY CROSS-FORMATION lifecycle axis.
//
// Per glossary + lifecycle-semantics.md: dissolution is DISTINGUISHED
// from demotion. Demotion (§0186) withdraws operational use;
// dissolution recognizes non-existence of the underlying phenomenon.
// Dissolve does NOT require prior promotion — formation may be
// dissolved directly. This test exercises that direct path explicitly.
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

func newCoordinationDissolveE2ESubstrate(t *testing.T) (*substrate.Substrate, *ingest.Ingester) {
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

// TestDissolveCoordinationRing_FromF3CandidateFormation exercises
// the full F3 → formation → dissolution arc (NO promotion step —
// formation may be dissolved directly per lifecycle-semantics.md).
// Mirrors §0166 + §0180 + §0184 on the CoordinationRing subtype side.
func TestDissolveCoordinationRing_FromF3CandidateFormation(t *testing.T) {
	sub, in := newCoordinationDissolveE2ESubstrate(t)
	ctx := context.Background()

	const bucketStart = int64(1716120000_000_000_000)
	appendNetworkObs(t, in, "actor-suspect-1", "10.0.0.10:443", bucketStart)
	appendNetworkObs(t, in, "actor-suspect-2", "10.0.0.10:443", bucketStart+10_000_000_000)
	appendNetworkObs(t, in, "actor-suspect-3", "10.0.0.10:443", bucketStart+20_000_000_000)

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

	formationAt := int64(1716120100000000000)
	formationHash := commitCoordinationRingFormationFromCandidate(t, sub, result.Candidates[0], formationAt)

	dissolvedAt := int64(1716120200000000000)
	now := func() time.Time { return time.Unix(0, dissolvedAt) }
	dissolveReport, err := hypothesis.DissolveCoordinationRing(ctx, sub, hypothesis.CoordinationRingDissolveOptions{
		FormationEventHash: formationHash,
		DissolvedAt:        dissolvedAt,
		Reason:             "e2e test dissolution (CoordinationRing): F3 candidate recognized as not corresponding to underlying coordinated ring",
	}, now)
	if err != nil {
		t.Fatalf("hypothesis.DissolveCoordinationRing: %v", err)
	}
	if dissolveReport.DissolutionEventHashHex == "" {
		t.Fatal("DissolveReport.DissolutionEventHashHex is empty")
	}
	if dissolveReport.AlreadyDissolved {
		t.Error("AlreadyDissolved: got true want false (fresh dissolution)")
	}

	formationRow, err := sub.LookupRow(ctx, formationHash)
	if err != nil {
		t.Fatalf("LookupRow formation: %v", err)
	}
	if formationRow.MessageType != "ghosttrace.events.v1.CoordinationRingFormation" {
		t.Errorf("formation MessageType: got %q want CoordinationRingFormation", formationRow.MessageType)
	}
	var dissolutionHash [32]byte
	if _, err := hexDecodeInto(dissolveReport.DissolutionEventHashHex, dissolutionHash[:]); err != nil {
		t.Fatalf("decode dissolution hash hex: %v", err)
	}
	dissolutionRow, err := sub.LookupRow(ctx, dissolutionHash)
	if err != nil {
		t.Fatalf("LookupRow dissolution: %v", err)
	}
	if dissolutionRow.MessageType != "ghosttrace.events.v1.CoordinationRingDissolution" {
		t.Errorf("dissolution MessageType: got %q want CoordinationRingDissolution", dissolutionRow.MessageType)
	}

	dissolutionPayload, err := sub.ReadBlob(ctx, dissolutionHash)
	if err != nil {
		t.Fatalf("ReadBlob dissolution: %v", err)
	}
	gotDissolution := &eventsv1.CoordinationRingDissolution{}
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

	t.Logf("dissolution E2E (CoordinationRing): formation=%x dissolution=%x", formationHash[:4], dissolutionHash[:4])
}

// TestDissolveCoordinationRing_IdempotencyUnderRepeatedCommit confirms
// content-hash idempotency: second dissolution commit with identical
// opts surfaces AlreadyDissolved=true + identical hash hex. Mirrors
// §0166 + §0180 + §0184 on the CoordinationRing subtype side.
func TestDissolveCoordinationRing_IdempotencyUnderRepeatedCommit(t *testing.T) {
	sub, in := newCoordinationDissolveE2ESubstrate(t)
	ctx := context.Background()

	const bucketStart = int64(1716120000_000_000_000)
	appendNetworkObs(t, in, "actor-x-1", "10.0.0.11:443", bucketStart)
	appendNetworkObs(t, in, "actor-x-2", "10.0.0.11:443", bucketStart+10_000_000_000)
	appendNetworkObs(t, in, "actor-x-3", "10.0.0.11:443", bucketStart+20_000_000_000)

	observations, err := collectNetworkObservations(ctx, sub)
	if err != nil {
		t.Fatalf("collectNetworkObservations: %v", err)
	}
	sig := &signatures.EndpointCoVisitV1{}
	result, err := sig.EvaluateNetwork(ctx, observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	formationHash := commitCoordinationRingFormationFromCandidate(t, sub, result.Candidates[0], 1716120100000000000)

	dissolvedAt := int64(1716120200000000000)
	now := func() time.Time { return time.Unix(0, dissolvedAt) }
	opts := hypothesis.CoordinationRingDissolveOptions{
		FormationEventHash: formationHash,
		DissolvedAt:        dissolvedAt,
		Reason:             "idempotency test (CoordinationRing)",
	}

	first, err := hypothesis.DissolveCoordinationRing(ctx, sub, opts, now)
	if err != nil {
		t.Fatalf("first DissolveCoordinationRing: %v", err)
	}
	if first.AlreadyDissolved {
		t.Error("first call AlreadyDissolved: got true want false")
	}

	second, err := hypothesis.DissolveCoordinationRing(ctx, sub, opts, now)
	if err != nil {
		t.Fatalf("second DissolveCoordinationRing: %v", err)
	}
	if !second.AlreadyDissolved {
		t.Error("second call AlreadyDissolved: got false want true (content-hash should collide)")
	}
	if first.DissolutionEventHashHex != second.DissolutionEventHashHex {
		t.Errorf("hash hex mismatch: first=%s second=%s (should be identical)",
			first.DissolutionEventHashHex, second.DissolutionEventHashHex)
	}
}
