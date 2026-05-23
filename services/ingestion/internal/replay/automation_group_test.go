package replay

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/hypothesis"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// substrateWithAGFormationForReplay ingests five uniform-cadence
// DeclaredSessions for a single actor, runs
// FormAutomationGroupAll under uniform-cadence-v1, returns
// (substrate, formation hash).
func substrateWithAGFormationForReplay(t *testing.T) (*substrate.Substrate, [32]byte) {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	in := ingest.New(sub, func() time.Time { return time.Unix(0, 1000) })
	for i := 0; i < 5; i++ {
		msg := &eventsv1.DeclaredSession{
			DeclaredAt:        1000 + int64(i)*1000,
			ActorRef:          "actor-ag-replay",
			SessionDescriptor: []byte("ag-replay"),
		}
		if _, err := in.Append(ctx, msg, msg.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("ingest.Append: %v", err)
		}
	}
	if _, err := hypothesis.FormAutomationGroupAll(ctx, sub,
		hypothesis.UniformCadenceV1{MinObservationCount: 5, MaxCoVThreshold: 0.15},
		func() time.Time { return time.Unix(0, 2000) }); err != nil {
		t.Fatalf("FormAutomationGroupAll: %v", err)
	}

	var formationHash [32]byte
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType == "ghosttrace.events.v1.AutomationGroupFormation" {
			formationHash = row.EventHash
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	if formationHash == ([32]byte{}) {
		t.Fatal("no AG formation found")
	}
	return sub, formationHash
}

func TestReplayAutomationGroupFormationHappyPath(t *testing.T) {
	sub, formationHash := substrateWithAGFormationForReplay(t)
	rep, err := ReplayAutomationGroupFormation(context.Background(), sub, formationHash)
	if err != nil {
		t.Fatalf("ReplayAutomationGroupFormation: %v", err)
	}
	if !rep.Match {
		t.Errorf("Match: got false, want true; reconstructed=%d contributors=%d",
			rep.ReconstructedFormationCount, rep.ContributingObservationCount)
	}
	if rep.PatternSignature != hypothesis.UniformCadenceV1Signature {
		t.Errorf("PatternSignature: got %q, want %q",
			rep.PatternSignature, hypothesis.UniformCadenceV1Signature)
	}
	if rep.ContributingObservationCount != 5 {
		t.Errorf("ContributingObservationCount: got %d, want 5",
			rep.ContributingObservationCount)
	}
}

func TestReplayAutomationGroupFormationUnknownTarget(t *testing.T) {
	sub, _ := substrateWithAGFormationForReplay(t)
	var bogus [32]byte
	for i := range bogus {
		bogus[i] = byte(0xff - i)
	}
	_, err := ReplayAutomationGroupFormation(context.Background(), sub, bogus)
	if !errors.Is(err, ErrTargetNotFound) {
		t.Errorf("expected ErrTargetNotFound; got %v", err)
	}
}

func TestReplayAutomationGroupFormationWrongMessageType(t *testing.T) {
	// Passing a BC formation hash to AG replay → ErrTargetWrongType.
	sub, _, bcHash := substrateWithBCFormation(t)
	_, err := ReplayAutomationGroupFormation(context.Background(), sub, bcHash)
	if !errors.Is(err, ErrTargetWrongType) {
		t.Errorf("expected ErrTargetWrongType; got %v", err)
	}
}

func TestReplayAutomationGroupFormationUnknownPattern(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	agBogus := &eventsv1.AutomationGroupFormation{
		PatternSignature:       "not-a-real-pattern",
		PatternParameters:      "x=y",
		FormationAt:            2000,
		EvidentialIndependence: eiOne(),
	}
	agPayload, agHash, _ := canonical.MarshalAndHash(agBogus)
	agHex := canonical.HashHex(agHash)
	if err := sub.Append(ctx, substrate.EventRow{
		EventHash: agHash, EventTime: agBogus.FormationAt,
		MessageType: "ghosttrace.events.v1.AutomationGroupFormation",
		PayloadRef:  agHex[:2] + "/" + agHex[2:], CommittedAt: 2500,
	}, agPayload); err != nil {
		t.Fatalf("append ag: %v", err)
	}

	_, err = ReplayAutomationGroupFormation(ctx, sub, agHash)
	if !errors.Is(err, ErrPatternUnknown) {
		t.Errorf("expected ErrPatternUnknown; got %v", err)
	}
}

func TestReplayAutomationGroupFormationSubstrateTimeFilter(t *testing.T) {
	// Same shape as the BC time-filter test (§0086): ingest 5
	// uniform-cadence DeclaredSessions, form AG, then ingest 5 MORE
	// AFTER the formation was committed. Replay must NOT see the
	// late observations.
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	in := ingest.New(sub, func() time.Time { return time.Unix(0, 1000) })
	for i := 0; i < 5; i++ {
		msg := &eventsv1.DeclaredSession{
			DeclaredAt:        1000 + int64(i)*1000,
			ActorRef:          "actor-early",
			SessionDescriptor: []byte("early"),
		}
		if _, err := in.Append(ctx, msg, msg.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("ingest early: %v", err)
		}
	}
	if _, err := hypothesis.FormAutomationGroupAll(ctx, sub,
		hypothesis.UniformCadenceV1{MinObservationCount: 5, MaxCoVThreshold: 0.15},
		func() time.Time { return time.Unix(0, 2000) }); err != nil {
		t.Fatalf("FormAutomationGroupAll: %v", err)
	}

	var formationHash [32]byte
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType == "ghosttrace.events.v1.AutomationGroupFormation" {
			formationHash = row.EventHash
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}

	// Phase 2: late observations from a different actor with uniform
	// cadence — would form a SECOND AG if visible. Replay should not
	// see these (committed_at > formationHash.committed_at).
	inLate := ingest.New(sub, func() time.Time { return time.Unix(0, 10_000_000_000) })
	for i := 0; i < 5; i++ {
		msg := &eventsv1.DeclaredSession{
			DeclaredAt:        9000 + int64(i)*1000,
			ActorRef:          "actor-late",
			SessionDescriptor: []byte("late"),
		}
		if _, err := inLate.Append(ctx, msg, msg.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("ingest late: %v", err)
		}
	}

	rep, err := ReplayAutomationGroupFormation(ctx, sub, formationHash)
	if err != nil {
		t.Fatalf("ReplayAutomationGroupFormation: %v", err)
	}
	if !rep.Match {
		t.Errorf("Match: got false, want true (substrate-time filter must exclude late observations)")
	}
	if rep.ContributingObservationCount != 5 {
		t.Errorf("ContributingObservationCount: got %d, want 5 (only events with committed_at ≤ formation.committed_at)",
			rep.ContributingObservationCount)
	}
}

func TestResolveAGFormationPatternUniformCadenceV1(t *testing.T) {
	pat, err := ResolveAGFormationPattern(hypothesis.UniformCadenceV1Signature, "max_cov_threshold=0.15;min_observation_count=5")
	if err != nil {
		t.Fatalf("ResolveAGFormationPattern: %v", err)
	}
	if pat.Signature() != hypothesis.UniformCadenceV1Signature {
		t.Errorf("Signature: got %q, want %q",
			pat.Signature(), hypothesis.UniformCadenceV1Signature)
	}
}

func TestResolveAGFormationPatternUnknownSignature(t *testing.T) {
	_, err := ResolveAGFormationPattern("not-real", "x=y")
	if !errors.Is(err, ErrPatternUnknown) {
		t.Errorf("expected ErrPatternUnknown; got %v", err)
	}
}

// substrateWithBCFormation is a local helper that wraps
// substrateWithBCFormationForReplay's behavior — the AG cross-subtype
// rejection test needs both a BC formation in a substrate AND access
// to that formation's hash. Duplicate of substrateWithBCFormationForReplay
// but returns the hash differently (mirrors httpapi/hypotheses_test.go
// pattern).
func substrateWithBCFormation(t *testing.T) (*substrate.Substrate, [32]byte, [32]byte) {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	in := ingest.New(sub, func() time.Time { return time.Unix(0, 1000) })
	for _, actor := range []string{"actor-a", "actor-b"} {
		msg := &eventsv1.DeclaredSession{
			DeclaredAt:        1000,
			ActorRef:          actor,
			SessionDescriptor: []byte("shared"),
		}
		if _, err := in.Append(ctx, msg, msg.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("ingest: %v", err)
		}
	}
	if _, err := hypothesis.FormAll(ctx, sub,
		hypothesis.SessionDescriptorSharedV1{MinClusterSize: 2},
		func() time.Time { return time.Unix(0, 2000) }); err != nil {
		t.Fatalf("FormAll: %v", err)
	}
	var bcHash [32]byte
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType == "ghosttrace.events.v1.BehavioralClusterFormation" {
			bcHash = row.EventHash
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	return sub, [32]byte{}, bcHash
}
