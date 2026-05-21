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

// substrateWithCRFormationForReplay ingests three actors co-occurring
// across three rounds (each round inside the window, rounds spaced
// beyond the window), runs FormCoordinationRingAll under
// co-occurrence-window-v1, returns (substrate, formation hash).
func substrateWithCRFormationForReplay(t *testing.T) (*substrate.Substrate, [32]byte) {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	gap := int64(60 * 1e9)
	roundSpacing := int64(2e12)
	in := ingest.New(sub, func() time.Time { return time.Unix(0, 1000) })
	actors := []string{"actor-cr-a", "actor-cr-b", "actor-cr-c"}
	for round := 0; round < 3; round++ {
		for i, actor := range actors {
			msg := &eventsv1.DeclaredSession{
				DeclaredAt:        1000 + int64(round)*roundSpacing + int64(i)*gap,
				ActorRef:          actor,
				SessionDescriptor: []byte("cr-replay"),
			}
			if _, err := in.Append(ctx, msg, msg.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
				t.Fatalf("ingest.Append: %v", err)
			}
		}
	}
	if _, err := hypothesis.FormCoordinationRingAll(ctx, sub,
		hypothesis.CoOccurrenceWindowV1{MinEdgeSupport: 3, MaxWindowSeconds: 600},
		func() time.Time { return time.Unix(0, 1000+3*roundSpacing) }); err != nil {
		t.Fatalf("FormCoordinationRingAll: %v", err)
	}

	var formationHash [32]byte
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType == "ghosttrace.events.v1.CoordinationRingFormation" {
			formationHash = row.EventHash
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	if formationHash == ([32]byte{}) {
		t.Fatal("no CR formation found")
	}
	return sub, formationHash
}

func TestReplayCoordinationRingFormationHappyPath(t *testing.T) {
	sub, formationHash := substrateWithCRFormationForReplay(t)
	rep, err := ReplayCoordinationRingFormation(context.Background(), sub, formationHash)
	if err != nil {
		t.Fatalf("ReplayCoordinationRingFormation: %v", err)
	}
	if !rep.Match {
		t.Errorf("Match: got false, want true; reconstructed=%d contributors=%d",
			rep.ReconstructedFormationCount, rep.ContributingObservationCount)
	}
	if rep.PatternSignature != hypothesis.CoOccurrenceWindowV1Signature {
		t.Errorf("PatternSignature: got %q, want %q",
			rep.PatternSignature, hypothesis.CoOccurrenceWindowV1Signature)
	}
	if rep.ContributingObservationCount != 9 { // 3 actors × 3 rounds
		t.Errorf("ContributingObservationCount: got %d, want 9",
			rep.ContributingObservationCount)
	}
}

func TestReplayCoordinationRingFormationUnknownTarget(t *testing.T) {
	sub, _ := substrateWithCRFormationForReplay(t)
	var bogus [32]byte
	for i := range bogus {
		bogus[i] = byte(0xff - i)
	}
	_, err := ReplayCoordinationRingFormation(context.Background(), sub, bogus)
	if !errors.Is(err, ErrTargetNotFound) {
		t.Errorf("expected ErrTargetNotFound; got %v", err)
	}
}

func TestReplayCoordinationRingFormationWrongMessageType(t *testing.T) {
	sub, _, bcHash := substrateWithBCFormation(t)
	_, err := ReplayCoordinationRingFormation(context.Background(), sub, bcHash)
	if !errors.Is(err, ErrTargetWrongType) {
		t.Errorf("expected ErrTargetWrongType; got %v", err)
	}
}

func TestReplayCoordinationRingFormationUnknownPattern(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	crBogus := &eventsv1.CoordinationRingFormation{
		PatternSignature:  "not-a-real-pattern",
		PatternParameters: "x=y",
		FormationAt:       2000,
	}
	crPayload, crHash, _ := canonical.MarshalAndHash(crBogus)
	crHex := canonical.HashHex(crHash)
	if err := sub.Append(ctx, substrate.EventRow{
		EventHash: crHash, EventTime: crBogus.FormationAt,
		MessageType: "ghosttrace.events.v1.CoordinationRingFormation",
		PayloadRef:  crHex[:2] + "/" + crHex[2:], CommittedAt: 2500,
	}, crPayload); err != nil {
		t.Fatalf("append cr: %v", err)
	}

	_, err = ReplayCoordinationRingFormation(ctx, sub, crHash)
	if !errors.Is(err, ErrPatternUnknown) {
		t.Errorf("expected ErrPatternUnknown; got %v", err)
	}
}

func TestReplayCoordinationRingFormationSubstrateTimeFilter(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	gap := int64(60 * 1e9)
	roundSpacing := int64(2e12)
	in := ingest.New(sub, func() time.Time { return time.Unix(0, 1000) })
	actors := []string{"actor-early-a", "actor-early-b", "actor-early-c"}
	for round := 0; round < 3; round++ {
		for i, actor := range actors {
			msg := &eventsv1.DeclaredSession{
				DeclaredAt:        1000 + int64(round)*roundSpacing + int64(i)*gap,
				ActorRef:          actor,
				SessionDescriptor: []byte("early-shared"),
			}
			if _, err := in.Append(ctx, msg, msg.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
				t.Fatalf("ingest early: %v", err)
			}
		}
	}
	if _, err := hypothesis.FormCoordinationRingAll(ctx, sub,
		hypothesis.CoOccurrenceWindowV1{MinEdgeSupport: 3, MaxWindowSeconds: 600},
		func() time.Time { return time.Unix(0, 1000+3*roundSpacing) }); err != nil {
		t.Fatalf("FormCoordinationRingAll: %v", err)
	}

	var formationHash [32]byte
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType == "ghosttrace.events.v1.CoordinationRingFormation" {
			formationHash = row.EventHash
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}

	// Phase 2: late observations with a different actor set + descriptor.
	inLate := ingest.New(sub, func() time.Time { return time.Unix(0, 100_000_000_000_000) })
	lateActors := []string{"actor-late-x", "actor-late-y", "actor-late-z"}
	for round := 0; round < 3; round++ {
		for i, actor := range lateActors {
			msg := &eventsv1.DeclaredSession{
				DeclaredAt:        90_000_000_000_000 + int64(round)*roundSpacing + int64(i)*gap,
				ActorRef:          actor,
				SessionDescriptor: []byte("late-shared"),
			}
			if _, err := inLate.Append(ctx, msg, msg.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
				t.Fatalf("ingest late: %v", err)
			}
		}
	}

	rep, err := ReplayCoordinationRingFormation(ctx, sub, formationHash)
	if err != nil {
		t.Fatalf("ReplayCoordinationRingFormation: %v", err)
	}
	if !rep.Match {
		t.Errorf("Match: got false, want true (substrate-time filter must exclude late observations)")
	}
	if rep.ContributingObservationCount != 9 {
		t.Errorf("ContributingObservationCount: got %d, want 9 (only events with committed_at ≤ formation.committed_at)",
			rep.ContributingObservationCount)
	}
}

func TestResolveCRFormationPatternCoOccurrenceWindowV1(t *testing.T) {
	pat, err := ResolveCRFormationPattern(
		hypothesis.CoOccurrenceWindowV1Signature,
		"max_window_seconds=600;min_edge_support=3")
	if err != nil {
		t.Fatalf("ResolveCRFormationPattern: %v", err)
	}
	if pat.Signature() != hypothesis.CoOccurrenceWindowV1Signature {
		t.Errorf("Signature: got %q, want %q",
			pat.Signature(), hypothesis.CoOccurrenceWindowV1Signature)
	}
}

func TestResolveCRFormationPatternUnknownSignature(t *testing.T) {
	_, err := ResolveCRFormationPattern("not-real", "x=y")
	if !errors.Is(err, ErrPatternUnknown) {
		t.Errorf("expected ErrPatternUnknown; got %v", err)
	}
}
