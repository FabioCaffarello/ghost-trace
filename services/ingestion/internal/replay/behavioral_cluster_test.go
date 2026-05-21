package replay

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/hypothesis"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// substrateWithBCFormationForReplay ingests two DeclaredSessions
// sharing a descriptor + runs FormAll under session-descriptor-
// shared-v1, returning (substrate, formation hash).
func substrateWithBCFormationForReplay(t *testing.T) (*substrate.Substrate, [32]byte) {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	in := ingest.New(sub, func() time.Time { return time.Unix(0, 1000) })
	for _, actor := range []string{"actor-bc-replay-a", "actor-bc-replay-b"} {
		msg := &eventsv1.DeclaredSession{
			DeclaredAt:        1000,
			ActorRef:          actor,
			SessionDescriptor: []byte("bc-replay"),
		}
		if _, err := in.Append(ctx, msg, msg.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("ingest.Append: %v", err)
		}
	}
	if _, err := hypothesis.FormAll(ctx, sub,
		hypothesis.SessionDescriptorSharedV1{MinClusterSize: 2},
		func() time.Time { return time.Unix(0, 2000) }); err != nil {
		t.Fatalf("FormAll: %v", err)
	}

	var formationHash [32]byte
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType == "ghosttrace.events.v1.BehavioralClusterFormation" {
			formationHash = row.EventHash
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	if formationHash == ([32]byte{}) {
		t.Fatal("no BC formation found")
	}
	return sub, formationHash
}

func TestReplayBehavioralClusterFormationHappyPath(t *testing.T) {
	sub, formationHash := substrateWithBCFormationForReplay(t)
	rep, err := ReplayBehavioralClusterFormation(context.Background(), sub, formationHash)
	if err != nil {
		t.Fatalf("ReplayBehavioralClusterFormation: %v", err)
	}
	if !rep.Match {
		t.Errorf("Match: got false, want true")
		t.Logf("  target=%s reconstructed=%d formations contributing=%d observations",
			rep.TargetHashHex, rep.ReconstructedFormationCount, rep.ContributingObservationCount)
	}
	if rep.PatternSignature != hypothesis.SessionDescriptorSharedV1Signature {
		t.Errorf("PatternSignature: got %q, want %q",
			rep.PatternSignature, hypothesis.SessionDescriptorSharedV1Signature)
	}
	if rep.ReconstructedFormationCount < 1 {
		t.Errorf("ReconstructedFormationCount: got %d, want >= 1",
			rep.ReconstructedFormationCount)
	}
	if rep.ContributingObservationCount != 2 {
		t.Errorf("ContributingObservationCount: got %d, want 2",
			rep.ContributingObservationCount)
	}
	if rep.MaxCommittedAtNs == 0 {
		t.Errorf("MaxCommittedAtNs: got 0, want non-zero (substrate commit time)")
	}
}

func TestReplayBehavioralClusterFormationUnknownTarget(t *testing.T) {
	sub, _ := substrateWithBCFormationForReplay(t)
	var bogus [32]byte
	for i := range bogus {
		bogus[i] = byte(0xff - i)
	}
	_, err := ReplayBehavioralClusterFormation(context.Background(), sub, bogus)
	if !errors.Is(err, ErrTargetNotFound) {
		t.Errorf("expected ErrTargetNotFound; got %v", err)
	}
}

func TestReplayBehavioralClusterFormationWrongMessageType(t *testing.T) {
	// Pass a DeclaredSession hash (Cat I, not BC formation)
	// → ErrTargetWrongType.
	sub, _ := substrateWithBCFormationForReplay(t)
	var dsHash [32]byte
	if err := sub.WalkEvents(context.Background(), func(row substrate.EventRow) error {
		if row.MessageType == "ghosttrace.events.v1.DeclaredSession" && dsHash == ([32]byte{}) {
			dsHash = row.EventHash
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	_, err := ReplayBehavioralClusterFormation(context.Background(), sub, dsHash)
	if !errors.Is(err, ErrTargetWrongType) {
		t.Errorf("expected ErrTargetWrongType; got %v", err)
	}
}

func TestReplayBehavioralClusterFormationDriftDetection(t *testing.T) {
	// Inject a hand-built BC formation with a different
	// pattern_parameters string than what
	// session-descriptor-shared-v1 currently emits → expect either
	// ErrPatternUnknown (parser fails) or Match=false.
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	// Source DeclaredSessions for the formation to reference.
	for _, actor := range []string{"actor-drift-a", "actor-drift-b"} {
		ds := &eventsv1.DeclaredSession{
			DeclaredAt:        1000,
			ActorRef:          actor,
			SessionDescriptor: []byte("drift-test"),
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
	}

	// Hand-built formation with corrupted parameters
	// ("MIN_CLUSTER_SIZE=2" instead of canonical "min_cluster_size=2").
	bcDrifted := &eventsv1.BehavioralClusterFormation{
		PatternSignature:  hypothesis.SessionDescriptorSharedV1Signature,
		PatternParameters: "MIN_CLUSTER_SIZE=2", // drifted from canonical
		ActorRefs:         []string{"actor-drift-a", "actor-drift-b"},
		FormationAt:       2000,
	}
	bcPayload, bcHash, _ := canonical.MarshalAndHash(bcDrifted)
	bcHex := canonical.HashHex(bcHash)
	if err := sub.Append(ctx, substrate.EventRow{
		EventHash: bcHash, EventTime: bcDrifted.FormationAt,
		MessageType: "ghosttrace.events.v1.BehavioralClusterFormation",
		PayloadRef:  bcHex[:2] + "/" + bcHex[2:], CommittedAt: 2500,
	}, bcPayload); err != nil {
		t.Fatalf("append bc: %v", err)
	}

	_, err = ReplayBehavioralClusterFormation(ctx, sub, bcHash)
	if err == nil {
		t.Fatalf("expected error on drifted parameters; got nil")
	}
	// Parser failure is acceptable (canonical lowercase form expected).
	// What we definitely don't want is Match=true.
}

func TestReplayBehavioralClusterFormationUnknownPattern(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	bcBogus := &eventsv1.BehavioralClusterFormation{
		PatternSignature:  "not-a-real-pattern",
		PatternParameters: "x=y",
		FormationAt:       2000,
	}
	bcPayload, bcHash, _ := canonical.MarshalAndHash(bcBogus)
	bcHex := canonical.HashHex(bcHash)
	if err := sub.Append(ctx, substrate.EventRow{
		EventHash: bcHash, EventTime: bcBogus.FormationAt,
		MessageType: "ghosttrace.events.v1.BehavioralClusterFormation",
		PayloadRef:  bcHex[:2] + "/" + bcHex[2:], CommittedAt: 2500,
	}, bcPayload); err != nil {
		t.Fatalf("append bc: %v", err)
	}

	_, err = ReplayBehavioralClusterFormation(ctx, sub, bcHash)
	if !errors.Is(err, ErrPatternUnknown) {
		t.Errorf("expected ErrPatternUnknown; got %v", err)
	}
}

func TestReplayBehavioralClusterFormationSubstrateTimeFilter(t *testing.T) {
	// Ingest 2 DeclaredSessions with descriptor "early", form BC,
	// then ingest 2 more with descriptor "late" (well after the
	// formation's commit). Phase 3 replay must NOT see the "late"
	// observations — the FormationContext is filtered to events
	// with committed_at ≤ original formation's committed_at.
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	// Phase 1: ingest two "early" sessions + form BC.
	in := ingest.New(sub, func() time.Time { return time.Unix(0, 1000) })
	for _, actor := range []string{"actor-early-a", "actor-early-b"} {
		msg := &eventsv1.DeclaredSession{
			DeclaredAt:        1000,
			ActorRef:          actor,
			SessionDescriptor: []byte("early"),
		}
		if _, err := in.Append(ctx, msg, msg.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("ingest early: %v", err)
		}
	}
	if _, err := hypothesis.FormAll(ctx, sub,
		hypothesis.SessionDescriptorSharedV1{MinClusterSize: 2},
		func() time.Time { return time.Unix(0, 2000) }); err != nil {
		t.Fatalf("FormAll: %v", err)
	}

	var formationHash [32]byte
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType == "ghosttrace.events.v1.BehavioralClusterFormation" {
			formationHash = row.EventHash
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}

	// Phase 2: ingest two MORE sessions AFTER the formation was
	// committed. These should NOT influence replay.
	inLate := ingest.New(sub, func() time.Time { return time.Unix(0, 10_000_000_000) }) // committed_at well after formation
	for _, actor := range []string{"actor-late-a", "actor-late-b"} {
		msg := &eventsv1.DeclaredSession{
			DeclaredAt:        9000,
			ActorRef:          actor,
			SessionDescriptor: []byte("late"),
		}
		if _, err := inLate.Append(ctx, msg, msg.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("ingest late: %v", err)
		}
	}

	rep, err := ReplayBehavioralClusterFormation(ctx, sub, formationHash)
	if err != nil {
		t.Fatalf("ReplayBehavioralClusterFormation: %v", err)
	}
	if !rep.Match {
		t.Errorf("Match: got false, want true (substrate-time filter should exclude late observations)")
	}
	// ContributingObservationCount should reflect ONLY the 2 early
	// DeclaredSessions visible at the formation's commit time, NOT
	// the 4 currently in the substrate.
	if rep.ContributingObservationCount != 2 {
		t.Errorf("ContributingObservationCount: got %d, want 2 (only events with committed_at ≤ formation.committed_at)",
			rep.ContributingObservationCount)
	}
}

func TestResolveBCFormationPatternSessionDescriptorSharedV1(t *testing.T) {
	pat, err := ResolveBCFormationPattern(hypothesis.SessionDescriptorSharedV1Signature, "min_cluster_size=2")
	if err != nil {
		t.Fatalf("ResolveBCFormationPattern: %v", err)
	}
	if pat.Signature() != hypothesis.SessionDescriptorSharedV1Signature {
		t.Errorf("Signature: got %q, want %q",
			pat.Signature(), hypothesis.SessionDescriptorSharedV1Signature)
	}
	if pat.Parameters() != "min_cluster_size=2" {
		t.Errorf("Parameters: got %q, want %q",
			pat.Parameters(), "min_cluster_size=2")
	}
}

func TestResolveBCFormationPatternUnknownSignature(t *testing.T) {
	_, err := ResolveBCFormationPattern("not-real", "x=y")
	if !errors.Is(err, ErrPatternUnknown) {
		t.Errorf("expected ErrPatternUnknown; got %v", err)
	}
}

// Compile guard: keep proto package import alive when test refactors
// move things around. Standard idiom mirrors §0084's test file.
var _ = proto.Marshal
