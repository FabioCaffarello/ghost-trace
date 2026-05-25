// Tests for ReplayDerivedActorAttribution per §0171. Mirrors
// replay_test.go's ReplayOperationalSession test patterns on the
// attribution Cat II side.
package replay

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/attribution"
	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// substrateWithDerivedAttributions ingests two NetworkObservation
// records + runs attribution.DeriveAll, returning (substrate, slice
// of DerivedActorAttribution hashes).
func substrateWithDerivedAttributions(t *testing.T) (*substrate.Substrate, [][32]byte) {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	in := ingest.New(sub, func() time.Time { return time.Unix(0, 1000) })

	endpoints := []string{"192.0.2.10:49152", "192.0.2.11:53124"}
	for i, ep := range endpoints {
		obs := &eventsv1.NetworkObservation{
			ObservedAt:          1000 + int64(i)*1000,
			EndpointRef:         ep,
			CollectorRef:        "test-collector:v1",
			AuthenticationClass: commonv1.AuthenticationClass_AUTHENTICATION_CLASS_SERVER_AUTHENTICATED,
			Modality: &eventsv1.NetworkObservation_TcpFingerprint{
				TcpFingerprint: &eventsv1.NetworkTcpFingerprint{WindowSize: 65535},
			},
		}
		if _, err := in.Append(ctx, obs, obs.ObservedAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("ingest.Append: %v", err)
		}
	}

	if _, err := attribution.DeriveAll(ctx, sub,
		attribution.Network5TupleActorV1{},
		func() time.Time { return time.Unix(0, 2000) }); err != nil {
		t.Fatalf("attribution.DeriveAll: %v", err)
	}

	var daaHashes [][32]byte
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType == "ghosttrace.events.v1.DerivedActorAttribution" {
			daaHashes = append(daaHashes, row.EventHash)
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	if len(daaHashes) != 2 {
		t.Fatalf("expected 2 DerivedActorAttribution rows; got %d", len(daaHashes))
	}
	return sub, daaHashes
}

func TestReplayDerivedActorAttribution_HappyPath(t *testing.T) {
	sub, daaHashes := substrateWithDerivedAttributions(t)
	rep, err := ReplayDerivedActorAttribution(context.Background(), sub, daaHashes[0])
	if err != nil {
		t.Fatalf("ReplayDerivedActorAttribution: %v", err)
	}
	if !rep.Match {
		t.Errorf("Match: got false, want true (re-derivation must equal substrate)")
		t.Errorf("  target=%s\n  recomputed=%s", rep.TargetHashHex, rep.RecomputedHashHex)
	}
	if rep.DefinitionVersion != attribution.Network5TupleActorV1Version {
		t.Errorf("DefinitionVersion: got %q, want %q", rep.DefinitionVersion, attribution.Network5TupleActorV1Version)
	}
	if rep.DefinitionParameters != "" {
		t.Errorf("DefinitionParameters: got %q, want empty (v1 takes no parameters)", rep.DefinitionParameters)
	}
}

func TestReplayDerivedActorAttribution_MatchAcrossMultiple(t *testing.T) {
	sub, daaHashes := substrateWithDerivedAttributions(t)
	for i, h := range daaHashes {
		rep, err := ReplayDerivedActorAttribution(context.Background(), sub, h)
		if err != nil {
			t.Fatalf("Replay daaHashes[%d]: %v", i, err)
		}
		if !rep.Match {
			t.Errorf("daaHashes[%d]: Match got false want true", i)
		}
	}
}

func TestReplayDerivedActorAttribution_UnknownTarget(t *testing.T) {
	sub, _ := substrateWithDerivedAttributions(t)
	var bogusHash [32]byte
	for i := range bogusHash {
		bogusHash[i] = 0xAB
	}
	_, err := ReplayDerivedActorAttribution(context.Background(), sub, bogusHash)
	if err == nil {
		t.Fatal("expected ErrTargetNotFound, got nil")
	}
	if !errors.Is(err, ErrTargetNotFound) {
		t.Errorf("expected ErrTargetNotFound, got %v", err)
	}
}

func TestReplayDerivedActorAttribution_WrongMessageType(t *testing.T) {
	// Use a NetworkObservation hash (Cat I, not Cat II) as the target.
	sub, _ := substrateWithDerivedAttributions(t)
	ctx := context.Background()
	var networkObsHash [32]byte
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType == "ghosttrace.events.v1.NetworkObservation" && networkObsHash == ([32]byte{}) {
			networkObsHash = row.EventHash
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	if networkObsHash == ([32]byte{}) {
		t.Fatal("could not locate a NetworkObservation hash for the wrong-type test")
	}
	_, err := ReplayDerivedActorAttribution(ctx, sub, networkObsHash)
	if err == nil {
		t.Fatal("expected ErrTargetWrongType, got nil")
	}
	if !errors.Is(err, ErrTargetWrongType) {
		t.Errorf("expected ErrTargetWrongType, got %v", err)
	}
}

func TestResolveAttributionDefinition_Network5TupleActorV1(t *testing.T) {
	def, err := ResolveAttributionDefinition(attribution.Network5TupleActorV1Version, "")
	if err != nil {
		t.Fatalf("ResolveAttributionDefinition: %v", err)
	}
	if def.Version() != attribution.Network5TupleActorV1Version {
		t.Errorf("Version: got %q want %q", def.Version(), attribution.Network5TupleActorV1Version)
	}
	if def.Parameters() != "" {
		t.Errorf("Parameters: got %q want empty", def.Parameters())
	}
}

func TestResolveAttributionDefinition_UnknownVersion(t *testing.T) {
	_, err := ResolveAttributionDefinition("nonexistent-definition-v99", "")
	if err == nil {
		t.Fatal("expected ErrDefinitionUnknown, got nil")
	}
	if !errors.Is(err, ErrDefinitionUnknown) {
		t.Errorf("expected ErrDefinitionUnknown, got %v", err)
	}
}

func TestResolveAttributionDefinition_RejectsNonEmptyParameters(t *testing.T) {
	// v1 takes no operator-supplied parameters; non-empty parameters
	// indicate either a future version (not yet registered) or
	// implementation drift on v1. Either way, error.
	_, err := ResolveAttributionDefinition(attribution.Network5TupleActorV1Version, "some=parameter")
	if err == nil {
		t.Fatal("expected ErrDefinitionParameterMismatch, got nil")
	}
	if !errors.Is(err, ErrDefinitionParameterMismatch) {
		t.Errorf("expected ErrDefinitionParameterMismatch, got %v", err)
	}
}
