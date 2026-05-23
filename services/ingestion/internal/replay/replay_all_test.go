package replay

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/derivation"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

func TestReplayAllOperationalSessionsEmptySubstrate(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	rep, err := ReplayAllOperationalSessions(ctx, sub)
	if err != nil {
		t.Fatalf("ReplayAllOperationalSessions: %v", err)
	}
	if rep.Total != 0 {
		t.Errorf("Total: got %d, want 0", rep.Total)
	}
	if rep.Matched != 0 || rep.Drifted != 0 || rep.Errored != 0 {
		t.Errorf("non-zero counts on empty substrate: %+v", rep)
	}
}

func TestReplayAllOperationalSessionsAllMatch(t *testing.T) {
	// Standard happy-path substrate: every OS replays cleanly.
	sub, osHashes := substrateWithDerivedSessions(t)
	rep, err := ReplayAllOperationalSessions(context.Background(), sub)
	if err != nil {
		t.Fatalf("ReplayAllOperationalSessions: %v", err)
	}
	if rep.Total != len(osHashes) {
		t.Errorf("Total: got %d, want %d", rep.Total, len(osHashes))
	}
	if rep.Matched != len(osHashes) {
		t.Errorf("Matched: got %d, want %d", rep.Matched, len(osHashes))
	}
	if rep.Drifted != 0 {
		t.Errorf("Drifted: got %d, want 0", rep.Drifted)
	}
	if rep.Errored != 0 {
		t.Errorf("Errored: got %d, want 0", rep.Errored)
	}
	if len(rep.Drift) != 0 {
		t.Errorf("Drift slice should be empty on all-match; got %v", rep.Drift)
	}
	if len(rep.Errors) != 0 {
		t.Errorf("Errors slice should be empty on all-match; got %v", rep.Errors)
	}

	// Sum invariant.
	if rep.Total != rep.Matched+rep.Drifted+rep.Errored {
		t.Errorf("Total (%d) != Matched (%d) + Drifted (%d) + Errored (%d)",
			rep.Total, rep.Matched, rep.Drifted, rep.Errored)
	}
}

func TestReplayAllOperationalSessionsCountsErrored(t *testing.T) {
	// Build a substrate with one valid OS + one hand-injected OS
	// whose definition_version is unrecognized. Total=2; Matched=1;
	// Errored=1.
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	// Source DeclaredSession for both OS records below.
	ds := &eventsv1.DeclaredSession{
		DeclaredAt:        1000,
		ActorRef:          "actor-mixed",
		SessionDescriptor: []byte("mixed"),
	}
	dsPayload, dsHash, _ := canonical.MarshalAndHash(ds)
	dsHex := canonical.HashHex(dsHash)
	if err := sub.Append(ctx, substrate.EventRow{
		EventHash: dsHash, EventTime: ds.DeclaredAt,
		MessageType: "ghosttrace.events.v1.DeclaredSession",
		PayloadRef:  dsHex[:2] + "/" + dsHex[2:], CommittedAt: 1500,
	}, dsPayload); err != nil {
		t.Fatalf("append ds: %v", err)
	}

	// Valid OS (padded-v1) — should match on replay.
	padded := derivation.PaddedV1{PadSeconds: 60}
	osValid := padded.Derive(ds, dsHash, nil)
	osValid.DefinitionVersion = padded.Version()
	osValid.DefinitionParameters = padded.Parameters()
	osValid.SourceEventHash = dsHash[:]
	osValidPayload, osValidHash, _ := canonical.MarshalAndHash(osValid)
	osValidHex := canonical.HashHex(osValidHash)
	if err := sub.Append(ctx, substrate.EventRow{
		EventHash: osValidHash, EventTime: osValid.OperationalStartAt,
		MessageType: "ghosttrace.events.v1.OperationalSession",
		PayloadRef:  osValidHex[:2] + "/" + osValidHex[2:], CommittedAt: 1500,
	}, osValidPayload); err != nil {
		t.Fatalf("append os-valid: %v", err)
	}

	// Hand-injected OS with unknown definition_version.
	osUnknown := &eventsv1.OperationalSession{
		DefinitionVersion:      "not-a-real-version",
		DefinitionParameters:   "x=y",
		SourceEventHash:        dsHash[:],
		ActorRef:               "actor-mixed",
		OperationalStartAt:     500,
		OperationalEndAt:       1500,
		EvidentialIndependence: eiOne(),
	}
	osUnknownPayload, osUnknownHash, _ := canonical.MarshalAndHash(osUnknown)
	osUnknownHex := canonical.HashHex(osUnknownHash)
	if err := sub.Append(ctx, substrate.EventRow{
		EventHash: osUnknownHash, EventTime: osUnknown.OperationalStartAt,
		MessageType: "ghosttrace.events.v1.OperationalSession",
		PayloadRef:  osUnknownHex[:2] + "/" + osUnknownHex[2:], CommittedAt: 1500,
	}, osUnknownPayload); err != nil {
		t.Fatalf("append os-unknown: %v", err)
	}

	rep, err := ReplayAllOperationalSessions(ctx, sub)
	if err != nil {
		t.Fatalf("ReplayAllOperationalSessions: %v", err)
	}
	if rep.Total != 2 {
		t.Errorf("Total: got %d, want 2", rep.Total)
	}
	if rep.Matched != 1 {
		t.Errorf("Matched: got %d, want 1", rep.Matched)
	}
	if rep.Errored != 1 {
		t.Errorf("Errored: got %d, want 1", rep.Errored)
	}
	if len(rep.Errors) != 1 {
		t.Fatalf("Errors slice: got %d, want 1", len(rep.Errors))
	}
	if rep.Errors[0].TargetHashHex != canonical.HashHex(osUnknownHash) {
		t.Errorf("Errors[0].TargetHashHex mismatch")
	}
	if rep.Errors[0].Outcome != OutcomeError {
		t.Errorf("Errors[0].Outcome: got %q, want %q", rep.Errors[0].Outcome, OutcomeError)
	}
}

func TestReplayAllOperationalSessionsSkipsNonOSRecords(t *testing.T) {
	// substrateWithDerivedSessions populates the substrate with two
	// DeclaredSessions + two IngestionEvent wrappers + two OS rows
	// (4 Cat I-ish records + 2 Cat II). Total replayed should be 2,
	// not 4 or 6.
	sub, osHashes := substrateWithDerivedSessions(t)
	rep, err := ReplayAllOperationalSessions(context.Background(), sub)
	if err != nil {
		t.Fatalf("ReplayAllOperationalSessions: %v", err)
	}
	if rep.Total != len(osHashes) {
		t.Errorf("Total: got %d, want %d (only OS records should count)", rep.Total, len(osHashes))
	}
}

func TestReplayAllOperationalSessionsSumInvariant(t *testing.T) {
	// Total == Matched + Drifted + Errored across every test substrate.
	sub, _ := substrateWithDerivedSessions(t)
	rep, err := ReplayAllOperationalSessions(context.Background(), sub)
	if err != nil {
		t.Fatalf("ReplayAllOperationalSessions: %v", err)
	}
	sum := rep.Matched + rep.Drifted + rep.Errored
	if sum != rep.Total {
		t.Errorf("Total (%d) != Matched (%d) + Drifted (%d) + Errored (%d) = %d",
			rep.Total, rep.Matched, rep.Drifted, rep.Errored, sum)
	}
}
