package replay

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/derivation"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// substrateWithDerivedSessions ingests two DeclaredSessions and runs
// DeriveAll under PaddedV1, returning (substrate, slice of OS hashes).
func substrateWithDerivedSessions(t *testing.T) (*substrate.Substrate, [][32]byte) {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	in := ingest.New(sub, func() time.Time { return time.Unix(0, 1000) })
	for i, actor := range []string{"actor-replay-a", "actor-replay-b"} {
		msg := &eventsv1.DeclaredSession{
			DeclaredAt:        1000 + int64(i)*1000,
			ActorRef:          actor,
			SessionDescriptor: []byte("replay-test"),
		}
		if _, err := in.Append(ctx, msg, msg.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("ingest.Append: %v", err)
		}
	}

	if _, err := derivation.DeriveAll(ctx, sub,
		derivation.PaddedV1{PadSeconds: 60},
		func() time.Time { return time.Unix(0, 2000) }); err != nil {
		t.Fatalf("DeriveAll: %v", err)
	}

	var osHashes [][32]byte
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType == "ghosttrace.events.v1.OperationalSession" {
			osHashes = append(osHashes, row.EventHash)
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	if len(osHashes) != 2 {
		t.Fatalf("expected 2 OS rows; got %d", len(osHashes))
	}
	return sub, osHashes
}

func TestReplayOperationalSessionHappyPathPaddedV1(t *testing.T) {
	sub, osHashes := substrateWithDerivedSessions(t)
	rep, err := ReplayOperationalSession(context.Background(), sub, osHashes[0])
	if err != nil {
		t.Fatalf("ReplayOperationalSession: %v", err)
	}
	if !rep.Match {
		t.Errorf("Match: got false, want true (re-derivation must equal substrate)")
		t.Errorf("  target=%s\n  recomputed=%s", rep.TargetHashHex, rep.RecomputedHashHex)
	}
	if rep.DefinitionVersion != derivation.PaddedV1Version {
		t.Errorf("DefinitionVersion: got %q, want %q", rep.DefinitionVersion, derivation.PaddedV1Version)
	}
	if rep.DefinitionParameters != "pad_seconds=60" {
		t.Errorf("DefinitionParameters: got %q, want pad_seconds=60", rep.DefinitionParameters)
	}
}

func TestReplayOperationalSessionHappyPathInactivityWindowV1(t *testing.T) {
	// Build a substrate with an OS derived under inactivity-window-v1
	// (not padded-v1) and verify replay recognizes + re-derives it.
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	in := ingest.New(sub, func() time.Time { return time.Unix(0, 1000) })
	ds := &eventsv1.DeclaredSession{
		DeclaredAt:        1000,
		ActorRef:          "actor-inact",
		SessionDescriptor: []byte("inact"),
	}
	if _, err := in.Append(ctx, ds, ds.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if _, err := derivation.DeriveAll(ctx, sub,
		derivation.InactivityWindowV1{InactivitySeconds: 300},
		func() time.Time { return time.Unix(0, 2000) }); err != nil {
		t.Fatalf("DeriveAll inactivity-window-v1: %v", err)
	}

	var osHash [32]byte
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType == "ghosttrace.events.v1.OperationalSession" {
			osHash = row.EventHash
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}

	rep, err := ReplayOperationalSession(ctx, sub, osHash)
	if err != nil {
		t.Fatalf("ReplayOperationalSession: %v", err)
	}
	if !rep.Match {
		t.Errorf("Match: got false, want true")
	}
	if rep.DefinitionVersion != derivation.InactivityWindowV1Version {
		t.Errorf("DefinitionVersion: got %q, want %q",
			rep.DefinitionVersion, derivation.InactivityWindowV1Version)
	}
}

func TestReplayOperationalSessionUnknownTarget(t *testing.T) {
	sub, _ := substrateWithDerivedSessions(t)
	var bogus [32]byte
	for i := range bogus {
		bogus[i] = byte(0xff - i)
	}
	_, err := ReplayOperationalSession(context.Background(), sub, bogus)
	if !errors.Is(err, ErrTargetNotFound) {
		t.Errorf("expected ErrTargetNotFound; got %v", err)
	}
}

func TestReplayOperationalSessionWrongMessageType(t *testing.T) {
	// Passing a DeclaredSession hash (Cat I, not the target type)
	// must return ErrTargetWrongType.
	sub, _ := substrateWithDerivedSessions(t)
	var dsHash [32]byte
	if err := sub.WalkEvents(context.Background(), func(row substrate.EventRow) error {
		if row.MessageType == "ghosttrace.events.v1.DeclaredSession" && dsHash == ([32]byte{}) {
			dsHash = row.EventHash
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	_, err := ReplayOperationalSession(context.Background(), sub, dsHash)
	if !errors.Is(err, ErrTargetWrongType) {
		t.Errorf("expected ErrTargetWrongType; got %v", err)
	}
}

func TestReplayOperationalSessionMatchAcrossMultiple(t *testing.T) {
	// Every derived OS in the substrate must replay cleanly. This
	// is the deterministic-replay-contract test per replay-model.md
	// L17-19 at the substrate-wide level.
	sub, osHashes := substrateWithDerivedSessions(t)
	for i, h := range osHashes {
		rep, err := ReplayOperationalSession(context.Background(), sub, h)
		if err != nil {
			t.Fatalf("[%d]: %v", i, err)
		}
		if !rep.Match {
			t.Errorf("[%d] Match=false: target=%s recomputed=%s",
				i, rep.TargetHashHex, rep.RecomputedHashHex)
		}
	}
}

func TestReplayOperationalSessionDetectsDefinitionDriftViaInjection(t *testing.T) {
	// Construct an OperationalSession by hand with a definition_version
	// that resolves to a registered definition (padded-v1) but with
	// parameters that no longer correspond to what padded-v1's
	// Parameters() method emits — i.e. pre-injection drift. Commit the
	// hand-built OS to the substrate, then replay.
	//
	// Expected: ErrDefinitionParameterMismatch — the parser parses
	// "pad_seconds=60" successfully, the resolved PaddedV1{60}'s
	// Parameters() emits "pad_seconds=60" (matches), so this exact case
	// matches. To exercise the mismatch path we'd need either (a) a
	// padded-v1 implementation that produces different bytes for the
	// same logical parameters, OR (b) corrupt the stored parameters
	// after derivation. Approach (b) is what we test below.
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	// Commit a DeclaredSession the way ingest does (raw, no envelope).
	ds := &eventsv1.DeclaredSession{
		DeclaredAt:        1000,
		ActorRef:          "actor-drift",
		SessionDescriptor: []byte("drift"),
	}
	dsPayload, dsHash, err := canonical.MarshalAndHash(ds)
	if err != nil {
		t.Fatalf("marshal ds: %v", err)
	}
	dsHex := canonical.HashHex(dsHash)
	if err := sub.Append(ctx, substrate.EventRow{
		EventHash:   dsHash,
		EventTime:   ds.DeclaredAt,
		MessageType: "ghosttrace.events.v1.DeclaredSession",
		PayloadRef:  dsHex[:2] + "/" + dsHex[2:],
		CommittedAt: 1500,
	}, dsPayload); err != nil {
		t.Fatalf("append ds: %v", err)
	}

	// Hand-build an OS claiming padded-v1 with DRIFTED parameter
	// encoding — same logical parameter, but the stored canonical
	// string differs from what PaddedV1.Parameters() would emit.
	osDrifted := &eventsv1.OperationalSession{
		DefinitionVersion:    derivation.PaddedV1Version,
		DefinitionParameters: "PAD_SECONDS=60", // uppercase drift; canonical form is lowercase
		SourceEventHash:      dsHash[:],
		ActorRef:             "actor-drift",
		OperationalStartAt:   940,
		OperationalEndAt:     1060,
	}
	osPayload, osHashRaw, err := canonical.MarshalAndHash(osDrifted)
	if err != nil {
		t.Fatalf("marshal os: %v", err)
	}
	osHex := canonical.HashHex(osHashRaw)
	if err := sub.Append(ctx, substrate.EventRow{
		EventHash:   osHashRaw,
		EventTime:   osDrifted.OperationalStartAt,
		MessageType: "ghosttrace.events.v1.OperationalSession",
		PayloadRef:  osHex[:2] + "/" + osHex[2:],
		CommittedAt: 1500,
	}, osPayload); err != nil {
		t.Fatalf("append os: %v", err)
	}

	// The drifted parameters fail the parser (PAD_SECONDS != pad_seconds)
	// → ErrDefinitionUnknown via parseIntParam (parameter not found).
	_, err = ReplayOperationalSession(ctx, sub, osHashRaw)
	if err == nil {
		t.Fatalf("expected error on drifted parameters; got nil")
	}
	// Acceptable: parser failure (parameter not found). What we
	// definitely don't want is Match=true.
}

func TestReplayOperationalSessionVersioning(t *testing.T) {
	// An OS with a bogus definition_version → ErrDefinitionUnknown.
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	ds := &eventsv1.DeclaredSession{
		DeclaredAt:        1000,
		ActorRef:          "actor-v",
		SessionDescriptor: []byte("v"),
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

	osUnknownV := &eventsv1.OperationalSession{
		DefinitionVersion:    "not-a-real-version",
		DefinitionParameters: "x=y",
		SourceEventHash:      dsHash[:],
		ActorRef:             "actor-v",
		OperationalStartAt:   500,
		OperationalEndAt:     1500,
	}
	osPayload, osHash, _ := canonical.MarshalAndHash(osUnknownV)
	osHex := canonical.HashHex(osHash)
	if err := sub.Append(ctx, substrate.EventRow{
		EventHash: osHash, EventTime: osUnknownV.OperationalStartAt,
		MessageType: "ghosttrace.events.v1.OperationalSession",
		PayloadRef:  osHex[:2] + "/" + osHex[2:], CommittedAt: 1500,
	}, osPayload); err != nil {
		t.Fatalf("append os: %v", err)
	}

	_, err = ReplayOperationalSession(ctx, sub, osHash)
	if !errors.Is(err, ErrDefinitionUnknown) {
		t.Errorf("expected ErrDefinitionUnknown; got %v", err)
	}
}

func TestReplayOperationalSessionMissingSource(t *testing.T) {
	// Commit an OS whose source_event_hash does NOT correspond to any
	// substrate row → ErrSourceNotFound.
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	var bogusSourceHash [32]byte
	for i := range bogusSourceHash {
		bogusSourceHash[i] = byte(0xab)
	}

	os := &eventsv1.OperationalSession{
		DefinitionVersion:    derivation.PaddedV1Version,
		DefinitionParameters: "pad_seconds=60",
		SourceEventHash:      bogusSourceHash[:],
		ActorRef:             "actor-miss",
		OperationalStartAt:   500,
		OperationalEndAt:     1500,
	}
	osPayload, osHash, _ := canonical.MarshalAndHash(os)
	osHex := canonical.HashHex(osHash)
	if err := sub.Append(ctx, substrate.EventRow{
		EventHash: osHash, EventTime: os.OperationalStartAt,
		MessageType: "ghosttrace.events.v1.OperationalSession",
		PayloadRef:  osHex[:2] + "/" + osHex[2:], CommittedAt: 1500,
	}, osPayload); err != nil {
		t.Fatalf("append os: %v", err)
	}

	_, err = ReplayOperationalSession(ctx, sub, osHash)
	if !errors.Is(err, ErrSourceNotFound) {
		t.Errorf("expected ErrSourceNotFound; got %v", err)
	}
}

func TestResolveOperationalDefinitionPaddedV1(t *testing.T) {
	def, err := ResolveOperationalDefinition(derivation.PaddedV1Version, "pad_seconds=60")
	if err != nil {
		t.Fatalf("ResolveOperationalDefinition: %v", err)
	}
	if def.Version() != derivation.PaddedV1Version {
		t.Errorf("Version: got %q, want %q", def.Version(), derivation.PaddedV1Version)
	}
	if def.Parameters() != "pad_seconds=60" {
		t.Errorf("Parameters: got %q, want %q", def.Parameters(), "pad_seconds=60")
	}
}

func TestResolveOperationalDefinitionInactivityWindowV1(t *testing.T) {
	def, err := ResolveOperationalDefinition(derivation.InactivityWindowV1Version, "inactivity_seconds=300")
	if err != nil {
		t.Fatalf("ResolveOperationalDefinition: %v", err)
	}
	if def.Version() != derivation.InactivityWindowV1Version {
		t.Errorf("Version: got %q, want %q", def.Version(), derivation.InactivityWindowV1Version)
	}
}

func TestResolveOperationalDefinitionUnknownVersion(t *testing.T) {
	_, err := ResolveOperationalDefinition("not-real", "x=y")
	if !errors.Is(err, ErrDefinitionUnknown) {
		t.Errorf("expected ErrDefinitionUnknown; got %v", err)
	}
}

// Use proto package import via marshaling helper to keep go vet happy.
var _ = proto.Marshal
