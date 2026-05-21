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

// substrateWithCHFormationForReplay ingests three DeclaredSessions
// sharing a descriptor within the default temporal window, runs
// FormCampaignHypothesisAll under temporal-descriptor-cohort-v1, and
// returns (substrate, formation hash).
func substrateWithCHFormationForReplay(t *testing.T) (*substrate.Substrate, [32]byte) {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	gap := int64(60 * 1e9) // 60s
	in := ingest.New(sub, func() time.Time { return time.Unix(0, 1000) })
	for i := 0; i < 3; i++ {
		msg := &eventsv1.DeclaredSession{
			DeclaredAt:        1000 + int64(i)*gap,
			ActorRef:          "actor-ch-replay",
			SessionDescriptor: []byte("ch-replay"),
		}
		if _, err := in.Append(ctx, msg, msg.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("ingest.Append: %v", err)
		}
	}
	if _, err := hypothesis.FormCampaignHypothesisAll(ctx, sub,
		hypothesis.TemporalDescriptorCohortV1{MinCampaignSize: 3, MaxIntraEventGapSeconds: 300},
		func() time.Time { return time.Unix(0, 1000+4*gap) }); err != nil {
		t.Fatalf("FormCampaignHypothesisAll: %v", err)
	}

	var formationHash [32]byte
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType == "ghosttrace.events.v1.CampaignHypothesisFormation" {
			formationHash = row.EventHash
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	if formationHash == ([32]byte{}) {
		t.Fatal("no CH formation found")
	}
	return sub, formationHash
}

func TestReplayCampaignHypothesisFormationHappyPath(t *testing.T) {
	sub, formationHash := substrateWithCHFormationForReplay(t)
	rep, err := ReplayCampaignHypothesisFormation(context.Background(), sub, formationHash)
	if err != nil {
		t.Fatalf("ReplayCampaignHypothesisFormation: %v", err)
	}
	if !rep.Match {
		t.Errorf("Match: got false, want true; reconstructed=%d contributors=%d",
			rep.ReconstructedFormationCount, rep.ContributingObservationCount)
	}
	if rep.PatternSignature != hypothesis.TemporalDescriptorCohortV1Signature {
		t.Errorf("PatternSignature: got %q, want %q",
			rep.PatternSignature, hypothesis.TemporalDescriptorCohortV1Signature)
	}
	if rep.ContributingObservationCount != 3 {
		t.Errorf("ContributingObservationCount: got %d, want 3",
			rep.ContributingObservationCount)
	}
}

func TestReplayCampaignHypothesisFormationUnknownTarget(t *testing.T) {
	sub, _ := substrateWithCHFormationForReplay(t)
	var bogus [32]byte
	for i := range bogus {
		bogus[i] = byte(0xff - i)
	}
	_, err := ReplayCampaignHypothesisFormation(context.Background(), sub, bogus)
	if !errors.Is(err, ErrTargetNotFound) {
		t.Errorf("expected ErrTargetNotFound; got %v", err)
	}
}

func TestReplayCampaignHypothesisFormationWrongMessageType(t *testing.T) {
	sub, _, bcHash := substrateWithBCFormation(t)
	_, err := ReplayCampaignHypothesisFormation(context.Background(), sub, bcHash)
	if !errors.Is(err, ErrTargetWrongType) {
		t.Errorf("expected ErrTargetWrongType; got %v", err)
	}
}

func TestReplayCampaignHypothesisFormationUnknownPattern(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	chBogus := &eventsv1.CampaignHypothesisFormation{
		PatternSignature:  "not-a-real-pattern",
		PatternParameters: "x=y",
		FormationAt:       2000,
	}
	chPayload, chHash, _ := canonical.MarshalAndHash(chBogus)
	chHex := canonical.HashHex(chHash)
	if err := sub.Append(ctx, substrate.EventRow{
		EventHash: chHash, EventTime: chBogus.FormationAt,
		MessageType: "ghosttrace.events.v1.CampaignHypothesisFormation",
		PayloadRef:  chHex[:2] + "/" + chHex[2:], CommittedAt: 2500,
	}, chPayload); err != nil {
		t.Fatalf("append ch: %v", err)
	}

	_, err = ReplayCampaignHypothesisFormation(ctx, sub, chHash)
	if !errors.Is(err, ErrPatternUnknown) {
		t.Errorf("expected ErrPatternUnknown; got %v", err)
	}
}

func TestReplayCampaignHypothesisFormationSubstrateTimeFilter(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	// Phase 1: ingest 3 early sessions + form CH.
	gap := int64(60 * 1e9)
	in := ingest.New(sub, func() time.Time { return time.Unix(0, 1000) })
	for i := 0; i < 3; i++ {
		msg := &eventsv1.DeclaredSession{
			DeclaredAt:        1000 + int64(i)*gap,
			ActorRef:          "actor-early",
			SessionDescriptor: []byte("early"),
		}
		if _, err := in.Append(ctx, msg, msg.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("ingest early: %v", err)
		}
	}
	if _, err := hypothesis.FormCampaignHypothesisAll(ctx, sub,
		hypothesis.TemporalDescriptorCohortV1{MinCampaignSize: 3, MaxIntraEventGapSeconds: 300},
		func() time.Time { return time.Unix(0, 1000+4*gap) }); err != nil {
		t.Fatalf("FormCampaignHypothesisAll: %v", err)
	}

	var formationHash [32]byte
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType == "ghosttrace.events.v1.CampaignHypothesisFormation" {
			formationHash = row.EventHash
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}

	// Phase 2: late sessions on a different descriptor — would form
	// a second CH if visible. Replay should not see them.
	inLate := ingest.New(sub, func() time.Time { return time.Unix(0, 10_000_000_000_000) })
	for i := 0; i < 3; i++ {
		msg := &eventsv1.DeclaredSession{
			DeclaredAt:        9_000_000_000_000 + int64(i)*gap,
			ActorRef:          "actor-late",
			SessionDescriptor: []byte("late"),
		}
		if _, err := inLate.Append(ctx, msg, msg.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("ingest late: %v", err)
		}
	}

	rep, err := ReplayCampaignHypothesisFormation(ctx, sub, formationHash)
	if err != nil {
		t.Fatalf("ReplayCampaignHypothesisFormation: %v", err)
	}
	if !rep.Match {
		t.Errorf("Match: got false, want true (substrate-time filter must exclude late observations)")
	}
	if rep.ContributingObservationCount != 3 {
		t.Errorf("ContributingObservationCount: got %d, want 3 (only events with committed_at ≤ formation.committed_at)",
			rep.ContributingObservationCount)
	}
}

func TestResolveCHFormationPatternTemporalDescriptorCohortV1(t *testing.T) {
	pat, err := ResolveCHFormationPattern(
		hypothesis.TemporalDescriptorCohortV1Signature,
		"max_intra_event_gap_seconds=300;min_campaign_size=3")
	if err != nil {
		t.Fatalf("ResolveCHFormationPattern: %v", err)
	}
	if pat.Signature() != hypothesis.TemporalDescriptorCohortV1Signature {
		t.Errorf("Signature: got %q, want %q",
			pat.Signature(), hypothesis.TemporalDescriptorCohortV1Signature)
	}
}

func TestResolveCHFormationPatternUnknownSignature(t *testing.T) {
	_, err := ResolveCHFormationPattern("not-real", "x=y")
	if !errors.Is(err, ErrPatternUnknown) {
		t.Errorf("expected ErrPatternUnknown; got %v", err)
	}
}
