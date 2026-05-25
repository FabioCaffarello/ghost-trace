// Package replay implements deterministic replay of Category II
// operational constructs from their Category I observational sources
// per docs/architecture/replay-model.md (Phase 1 replay —
// deterministic over observation alone) + decision-log §0084.
//
// Replay re-derives a committed Cat II record from the substrate by:
//
//  1. Reading the Cat II record's self-describing derivation metadata
//     (definition_version + definition_parameters + source_event_hash
//     on OperationalSession per §0043).
//  2. Resolving definition_version to a registered OperationalDefinition
//     implementation.
//  3. Verifying the resolved definition's Parameters() matches the
//     stored definition_parameters byte-for-byte (defends against
//     definition-implementation drift that the canonical-parameter
//     encoding does not catch).
//  4. Looking up the referenced Cat I source observation.
//  5. Re-running the derivation under a freshly-collected
//     DerivationContext via derivation.CollectDerivationContext.
//  6. Canonical-marshaling + hashing the re-derived record.
//  7. Comparing the recomputed hash to the substrate's committed
//     content-hash for the original record.
//
// Match = PASS; divergence = FAIL. A failure indicates that either
// (a) the operational-definition implementation has drifted since
// the original derivation (a bug to investigate), or (b) the
// substrate's Cat II record is inconsistent with its declared
// derivation inputs (a §2.1-immutability concern). Per Charter §2.1
// + §2.2, the substrate's record is authoritative — replay verifies
// the derivation logic, not the substrate.
//
// Scope: this package supports both Cat II types committed today —
// OperationalSession (per §0043) via ReplayOperationalSession, and
// DerivedActorAttribution (per §0168) via ReplayDerivedActorAttribution
// (added at §0171). Cat III hypothesis replay is Phase 3 reconstructive
// replay per replay-model.md and is out of scope here.
package replay

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"google.golang.org/protobuf/proto"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/attribution"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/derivation"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

const (
	operationalSessionMessageType       = "ghosttrace.events.v1.OperationalSession"
	declaredSessionMessageType          = "ghosttrace.events.v1.DeclaredSession"
	derivedActorAttributionMessageType  = "ghosttrace.events.v1.DerivedActorAttribution"
	networkObservationMessageType       = "ghosttrace.events.v1.NetworkObservation"
)

// Sentinels reported by ReplayOperationalSession.
var (
	// ErrTargetNotFound: the target hash does not resolve to any
	// substrate row.
	ErrTargetNotFound = errors.New("replay: target not found")

	// ErrTargetWrongType: the target hash resolves to a row whose
	// message_type is not the expected Cat II type.
	ErrTargetWrongType = errors.New("replay: target is wrong message type")

	// ErrDefinitionUnknown: the Cat II record's definition_version
	// does not resolve to a registered OperationalDefinition.
	ErrDefinitionUnknown = errors.New("replay: definition_version not registered")

	// ErrDefinitionParameterMismatch: the resolved definition's
	// Parameters() does not match the Cat II record's
	// definition_parameters byte-for-byte. Indicates the
	// definition implementation has drifted in its parameter
	// encoding since the original record was committed.
	ErrDefinitionParameterMismatch = errors.New("replay: definition parameters drifted")

	// ErrSourceNotFound: the Cat II record's source_event_hash does
	// not resolve to a substrate row.
	ErrSourceNotFound = errors.New("replay: source observation not found")

	// ErrSourceWrongType: the source_event_hash resolves to a row
	// that is not the expected Cat I type.
	ErrSourceWrongType = errors.New("replay: source observation is wrong message type")
)

// OperationalSessionReport is the per-ReplayOperationalSession
// outcome. Match is true iff the re-derived canonical-hash equals
// the substrate's committed hash for the original record. Other
// fields are diagnostic.
type OperationalSessionReport struct {
	// TargetHashHex is the hex content-hash of the OperationalSession
	// being replayed (input).
	TargetHashHex string

	// RecomputedHashHex is the hex content-hash of the re-derived
	// OperationalSession (output of replay).
	RecomputedHashHex string

	// Match is true iff TargetHashHex == RecomputedHashHex.
	Match bool

	// DefinitionVersion is the version string read from the original
	// OperationalSession.
	DefinitionVersion string

	// DefinitionParameters is the canonical-parameter string read
	// from the original OperationalSession.
	DefinitionParameters string

	// SourceEventHashHex is the hex content-hash of the source
	// DeclaredSession referenced by the original OperationalSession.
	SourceEventHashHex string
}

// ReplayOperationalSession re-derives the OperationalSession
// identified by targetHash from its declared source DeclaredSession,
// under the same operational definition + parameters recorded on the
// original record. Returns a report carrying both the original and
// recomputed content-hashes; Match is true iff they agree.
//
// Phase 1 replay per docs/architecture/replay-model.md L17–19:
// deterministic over observation alone. The replay does NOT consult
// the original OperationalSession's payload beyond its
// definition_version + definition_parameters + source_event_hash
// fields; it freshly re-derives from the Cat I observation.
func ReplayOperationalSession(ctx context.Context, sub *substrate.Substrate, targetHash [32]byte) (OperationalSessionReport, error) {
	row, err := sub.LookupRow(ctx, targetHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return OperationalSessionReport{}, fmt.Errorf("%w: %x", ErrTargetNotFound, targetHash)
		}
		return OperationalSessionReport{}, fmt.Errorf("replay.ReplayOperationalSession: lookup target: %w", err)
	}
	if row.MessageType != operationalSessionMessageType {
		return OperationalSessionReport{}, fmt.Errorf("%w: %x is %q (expected %s)",
			ErrTargetWrongType, targetHash, row.MessageType, operationalSessionMessageType)
	}

	payload, err := sub.ReadBlob(ctx, targetHash)
	if err != nil {
		return OperationalSessionReport{}, fmt.Errorf("replay.ReplayOperationalSession: read target blob: %w", err)
	}
	original := &eventsv1.OperationalSession{}
	if err := proto.Unmarshal(payload, original); err != nil {
		return OperationalSessionReport{}, fmt.Errorf("replay.ReplayOperationalSession: unmarshal target: %w", err)
	}

	def, err := ResolveOperationalDefinition(original.DefinitionVersion, original.DefinitionParameters)
	if err != nil {
		return OperationalSessionReport{}, err
	}
	if def.Parameters() != original.DefinitionParameters {
		return OperationalSessionReport{}, fmt.Errorf("%w: definition %q produced %q, original carried %q",
			ErrDefinitionParameterMismatch, def.Version(),
			def.Parameters(), original.DefinitionParameters)
	}

	var sourceHash [32]byte
	if len(original.SourceEventHash) != 32 {
		return OperationalSessionReport{}, fmt.Errorf("replay.ReplayOperationalSession: source_event_hash must be 32 bytes; got %d",
			len(original.SourceEventHash))
	}
	copy(sourceHash[:], original.SourceEventHash)

	sourceRow, err := sub.LookupRow(ctx, sourceHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return OperationalSessionReport{}, fmt.Errorf("%w: %x", ErrSourceNotFound, sourceHash)
		}
		return OperationalSessionReport{}, fmt.Errorf("replay.ReplayOperationalSession: lookup source: %w", err)
	}
	if sourceRow.MessageType != declaredSessionMessageType {
		return OperationalSessionReport{}, fmt.Errorf("%w: source %x is %q (expected %s)",
			ErrSourceWrongType, sourceHash, sourceRow.MessageType, declaredSessionMessageType)
	}

	sourcePayload, err := sub.ReadBlob(ctx, sourceHash)
	if err != nil {
		return OperationalSessionReport{}, fmt.Errorf("replay.ReplayOperationalSession: read source blob: %w", err)
	}
	source := &eventsv1.DeclaredSession{}
	if err := proto.Unmarshal(sourcePayload, source); err != nil {
		return OperationalSessionReport{}, fmt.Errorf("replay.ReplayOperationalSession: unmarshal source: %w", err)
	}

	dctx, err := derivation.CollectDerivationContext(ctx, sub)
	if err != nil {
		return OperationalSessionReport{}, fmt.Errorf("replay.ReplayOperationalSession: collect derivation context: %w", err)
	}

	rederived := def.Derive(source, sourceHash, dctx)
	rederived.DefinitionVersion = def.Version()
	rederived.DefinitionParameters = def.Parameters()
	rederived.SourceEventHash = sourceHash[:]

	_, recomputedHash, err := canonical.MarshalAndHash(rederived)
	if err != nil {
		return OperationalSessionReport{}, fmt.Errorf("replay.ReplayOperationalSession: marshal rederived: %w", err)
	}
	recomputedHex := canonical.HashHex(recomputedHash)
	targetHex := canonical.HashHex(targetHash)

	return OperationalSessionReport{
		TargetHashHex:        targetHex,
		RecomputedHashHex:    recomputedHex,
		Match:                bytes.Equal(targetHash[:], recomputedHash[:]),
		DefinitionVersion:    original.DefinitionVersion,
		DefinitionParameters: original.DefinitionParameters,
		SourceEventHashHex:   canonical.HashHex(sourceHash),
	}, nil
}

// ResolveOperationalDefinition maps an OperationalSession's
// (definition_version, definition_parameters) tuple back to a
// concrete OperationalDefinition implementation. Parses the
// definition_parameters string per the version's known schema and
// returns a fresh instance suitable for re-derivation.
//
// Currently supports PaddedV1 ("pad_seconds=N") and
// InactivityWindowV1 ("inactivity_seconds=N"). New operational
// definitions register here. ErrDefinitionUnknown for unrecognized
// versions; format errors for malformed parameters.
func ResolveOperationalDefinition(version, parameters string) (derivation.OperationalDefinition, error) {
	switch version {
	case derivation.PaddedV1Version:
		padSeconds, err := parseIntParam(parameters, "pad_seconds")
		if err != nil {
			return nil, fmt.Errorf("parse padded-v1 parameters %q: %w", parameters, err)
		}
		return derivation.PaddedV1{PadSeconds: padSeconds}, nil

	case derivation.InactivityWindowV1Version:
		inactivitySeconds, err := parseIntParam(parameters, "inactivity_seconds")
		if err != nil {
			return nil, fmt.Errorf("parse inactivity-window-v1 parameters %q: %w", parameters, err)
		}
		return derivation.InactivityWindowV1{InactivitySeconds: inactivitySeconds}, nil

	default:
		return nil, fmt.Errorf("%w: %q (known: %s, %s)",
			ErrDefinitionUnknown, version,
			derivation.PaddedV1Version, derivation.InactivityWindowV1Version)
	}
}

// parseIntParam extracts the value of a single-key canonical
// parameter string of the form "key=N". The canonical encoding
// rules per OperationalDefinition.Parameters() guarantee lowercase
// keys sorted alphabetically separated by semicolons; for single-
// key definitions this reduces to "key=N".
func parseIntParam(parameters, key string) (int64, error) {
	prefix := key + "="
	for _, segment := range strings.Split(parameters, ";") {
		if strings.HasPrefix(segment, prefix) {
			v, err := strconv.ParseInt(segment[len(prefix):], 10, 64)
			if err != nil {
				return 0, fmt.Errorf("parse %s value: %w", key, err)
			}
			return v, nil
		}
	}
	return 0, fmt.Errorf("parameter %s not found in %q", key, parameters)
}

// DerivedActorAttributionReport is the per-ReplayDerivedActorAttribution
// outcome per §0171. Mirrors OperationalSessionReport on the attribution
// Cat II side. Match is true iff the re-derived canonical-hash equals
// the substrate's committed hash for the original record.
type DerivedActorAttributionReport struct {
	// TargetHashHex is the hex content-hash of the
	// DerivedActorAttribution being replayed (input).
	TargetHashHex string

	// RecomputedHashHex is the hex content-hash of the re-derived
	// DerivedActorAttribution (output of replay).
	RecomputedHashHex string

	// Match is true iff TargetHashHex == RecomputedHashHex.
	Match bool

	// DefinitionVersion is the version string read from the original
	// DerivedActorAttribution.
	DefinitionVersion string

	// DefinitionParameters is the canonical-parameter string read
	// from the original DerivedActorAttribution.
	DefinitionParameters string

	// SourceEventHashHex is the hex content-hash of the source Cat I
	// observation (typically NetworkObservation) referenced by the
	// original DerivedActorAttribution.
	SourceEventHashHex string
}

// ReplayDerivedActorAttribution re-derives the DerivedActorAttribution
// identified by targetHash from its declared source Cat I observation,
// under the same attribution definition + parameters recorded on the
// original record per §0171. Returns a report carrying both the
// original and recomputed content-hashes; Match is true iff they agree.
//
// Phase 1 replay per docs/architecture/replay-model.md: deterministic
// over observation alone. Mirrors ReplayOperationalSession on the
// attribution Cat II side.
//
// The replay does NOT consult the original DerivedActorAttribution's
// payload beyond its definition_version + definition_parameters +
// source_event_hash fields; it freshly re-derives from the Cat I
// observation. Match failure indicates either (a) attribution
// definition implementation drift since original commit, or (b)
// substrate inconsistency between the Cat II record and its declared
// source observation per Charter §2.1.
func ReplayDerivedActorAttribution(ctx context.Context, sub *substrate.Substrate, targetHash [32]byte) (DerivedActorAttributionReport, error) {
	row, err := sub.LookupRow(ctx, targetHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return DerivedActorAttributionReport{}, fmt.Errorf("%w: %x", ErrTargetNotFound, targetHash)
		}
		return DerivedActorAttributionReport{}, fmt.Errorf("replay.ReplayDerivedActorAttribution: lookup target: %w", err)
	}
	if row.MessageType != derivedActorAttributionMessageType {
		return DerivedActorAttributionReport{}, fmt.Errorf("%w: %x is %q (expected %s)",
			ErrTargetWrongType, targetHash, row.MessageType, derivedActorAttributionMessageType)
	}

	payload, err := sub.ReadBlob(ctx, targetHash)
	if err != nil {
		return DerivedActorAttributionReport{}, fmt.Errorf("replay.ReplayDerivedActorAttribution: read target blob: %w", err)
	}
	original := &eventsv1.DerivedActorAttribution{}
	if err := proto.Unmarshal(payload, original); err != nil {
		return DerivedActorAttributionReport{}, fmt.Errorf("replay.ReplayDerivedActorAttribution: unmarshal target: %w", err)
	}

	def, err := ResolveAttributionDefinition(original.DefinitionVersion, original.DefinitionParameters)
	if err != nil {
		return DerivedActorAttributionReport{}, err
	}
	if def.Parameters() != original.DefinitionParameters {
		return DerivedActorAttributionReport{}, fmt.Errorf("%w: definition %q produced %q, original carried %q",
			ErrDefinitionParameterMismatch, def.Version(),
			def.Parameters(), original.DefinitionParameters)
	}

	var sourceHash [32]byte
	if len(original.SourceEventHash) != 32 {
		return DerivedActorAttributionReport{}, fmt.Errorf("replay.ReplayDerivedActorAttribution: source_event_hash must be 32 bytes; got %d",
			len(original.SourceEventHash))
	}
	copy(sourceHash[:], original.SourceEventHash)

	sourceRow, err := sub.LookupRow(ctx, sourceHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return DerivedActorAttributionReport{}, fmt.Errorf("%w: %x", ErrSourceNotFound, sourceHash)
		}
		return DerivedActorAttributionReport{}, fmt.Errorf("replay.ReplayDerivedActorAttribution: lookup source: %w", err)
	}
	if sourceRow.MessageType != networkObservationMessageType {
		return DerivedActorAttributionReport{}, fmt.Errorf("%w: source %x is %q (expected %s)",
			ErrSourceWrongType, sourceHash, sourceRow.MessageType, networkObservationMessageType)
	}

	sourcePayload, err := sub.ReadBlob(ctx, sourceHash)
	if err != nil {
		return DerivedActorAttributionReport{}, fmt.Errorf("replay.ReplayDerivedActorAttribution: read source blob: %w", err)
	}
	source := &eventsv1.NetworkObservation{}
	if err := proto.Unmarshal(sourcePayload, source); err != nil {
		return DerivedActorAttributionReport{}, fmt.Errorf("replay.ReplayDerivedActorAttribution: unmarshal source: %w", err)
	}

	rederived, ok := def.Derive(source)
	if !ok || rederived == nil {
		return DerivedActorAttributionReport{}, fmt.Errorf("replay.ReplayDerivedActorAttribution: definition %q declined to derive from source %x (Derive returned ok=false); cannot reproduce original record",
			def.Version(), sourceHash)
	}
	rederived.DefinitionVersion = def.Version()
	rederived.DefinitionParameters = def.Parameters()
	rederived.SourceEventHash = sourceHash[:]

	_, recomputedHash, err := canonical.MarshalAndHash(rederived)
	if err != nil {
		return DerivedActorAttributionReport{}, fmt.Errorf("replay.ReplayDerivedActorAttribution: marshal rederived: %w", err)
	}
	recomputedHex := canonical.HashHex(recomputedHash)
	targetHex := canonical.HashHex(targetHash)

	return DerivedActorAttributionReport{
		TargetHashHex:        targetHex,
		RecomputedHashHex:    recomputedHex,
		Match:                bytes.Equal(targetHash[:], recomputedHash[:]),
		DefinitionVersion:    original.DefinitionVersion,
		DefinitionParameters: original.DefinitionParameters,
		SourceEventHashHex:   canonical.HashHex(sourceHash),
	}, nil
}

// ResolveAttributionDefinition maps a DerivedActorAttribution's
// (definition_version, definition_parameters) tuple back to a concrete
// AttributionDefinition implementation per §0171. Mirrors
// ResolveOperationalDefinition on the attribution side.
//
// Currently supports network_5tuple_actor_v1 (no parameters; per
// §0168 v1 inception scope). New attribution definitions register
// here. ErrDefinitionUnknown for unrecognized versions.
func ResolveAttributionDefinition(version, parameters string) (attribution.AttributionDefinition, error) {
	switch version {
	case attribution.Network5TupleActorV1Version:
		// v1 takes no operator-supplied parameters; the canonical
		// parameters string is empty.
		if parameters != "" {
			return nil, fmt.Errorf("%w: network-5tuple-actor-v1 expects empty parameters, got %q",
				ErrDefinitionParameterMismatch, parameters)
		}
		return attribution.Network5TupleActorV1{}, nil

	default:
		return nil, fmt.Errorf("%w: %q (known attribution definitions: %s)",
			ErrDefinitionUnknown, version,
			attribution.Network5TupleActorV1Version)
	}
}
