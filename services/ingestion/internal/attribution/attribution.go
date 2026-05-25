// Package attribution implements Category II derived-actor-attribution
// per Charter §2.2 + entity-model.md §Category II + decision-log §0168.
// Parallel to the derivation package (which produces OperationalSession
// per §0015) — this package produces DerivedActorAttribution records.
//
// Per §0168 Decision A.1 (signature-aware consumption): F3 signatures
// consume Cat II derived attribution via an explicit AttributionView
// parameter passed by the orchestrator, NOT via silent back-fill of
// Cat I records' actor_ref field. The epistemic separation between
// observed (Cat I actor_ref) and derived (Cat II derived_actor_ref)
// attribution is preserved at the consumption point per §2.2 BC1.
//
// Per §0168 per-record derivation: one DerivedActorAttribution record
// per Cat I source observation. Same flow → same derived_actor_ref →
// content-hash collision under substrate INSERT OR IGNORE if the
// source_event_hash is identical; distinct Cat I sources within the
// same flow produce distinct Cat II records (one per source) sharing
// the derived_actor_ref but pointing to distinct sources.
//
// Per §2.2 Cat II determinism: re-derivation under (definition_version,
// definition_parameters, source_event_hash) IDENTICAL inputs produces
// content-hash-identical output (substrate idempotency).
//
// Per §2.2 Forbidden Anti-Pattern 4 (Reclassification): this package
// NEVER mutates Cat I records. Cat I source records' actor_ref fields
// remain unchanged (typically empty for the derivation targets — Cat I
// records with declared actor_ref do not need Cat II attribution).
package attribution

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// networkObservationMessageType is the substrate message_type for
// NetworkObservation rows — the Cat I source type for inception-phase
// AttributionDefinitions. Future definitions targeting BehavioralObservation
// or AttestationObservation would add additional message-type discriminators.
const networkObservationMessageType = "ghosttrace.events.v1.NetworkObservation"

// derivedActorAttributionMessageType is the substrate message_type
// produced by this package. Used by AttributionView's substrate walk
// to select Cat II records.
const derivedActorAttributionMessageType = "ghosttrace.events.v1.DerivedActorAttribution"

// AttributionDefinition is the deterministic-derivation contract for
// Category II actor-attribution constructs. Mirrors derivation.
// OperationalDefinition on the per-observation-attribution axis.
//
// Implementations MUST be deterministic with respect to (source,
// Parameters); a definition whose Derive method is non-deterministic
// violates the contract + would be rejected at structural validation.
type AttributionDefinition interface {
	// Version is the stable identifier of this attribution definition
	// (e.g., "network-5tuple-actor-v1"). Part of the produced
	// DerivedActorAttribution's identity per entity-model.md line 46.
	Version() string

	// Parameters returns the canonical-form string encoding of this
	// definition's parameters. Part of identity per entity-model.md
	// line 44. Canonical form: lowercase key=value pairs separated by
	// semicolons, sorted by key. Empty when the definition has no
	// operator-supplied parameters (e.g., network_5tuple_actor_v1
	// reads inputs from the source observation only).
	Parameters() string

	// Derive produces the DerivedActorAttribution for the given source
	// NetworkObservation. The output's definition_version,
	// definition_parameters, and source_event_hash fields are set by
	// the caller (DeriveAll); the implementation populates the
	// derivation-specific fields (derived_actor_ref, confidence,
	// evidential_independence).
	//
	// Returns nil + the second return value when the source is not
	// derivable by this definition (e.g., NetworkObservation lacks
	// the modality this definition requires); DeriveAll skips nil
	// outputs without committing.
	Derive(source *eventsv1.NetworkObservation) (*eventsv1.DerivedActorAttribution, bool)
}

// Report is the per-DeriveAll outcome.
type Report struct {
	Examined       int64
	Skipped        int64
	NewlyDerived   int64
	AlreadyDerived int64
}

// DeriveAll walks every NetworkObservation in the substrate, applies
// def to each, and commits the resulting DerivedActorAttribution via
// substrate.Append. The commit is idempotent per content-hash
// collision: re-running with the same def produces zero new rows.
//
// Sources for which def.Derive returns (nil, false) are counted as
// Skipped (the definition declined; not an error). All other outcomes
// (committed or already-present) count toward NewlyDerived /
// AlreadyDerived.
//
// Concurrency: same discipline as derivation.DeriveAll — read via
// WalkEvents (no writeMu) + write via substrate.Append (acquires
// writeMu). Safe to run alongside the ingestion service's write path.
func DeriveAll(ctx context.Context, sub *substrate.Substrate, def AttributionDefinition, now func() time.Time) (Report, error) {
	if def == nil {
		return Report{}, errors.New("attribution.DeriveAll: definition must not be nil")
	}
	if now == nil {
		now = time.Now
	}

	var rep Report
	walkErr := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != networkObservationMessageType {
			return nil
		}
		rep.Examined++

		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return fmt.Errorf("read source blob %x: %w", row.EventHash, err)
		}
		source := &eventsv1.NetworkObservation{}
		if err := proto.Unmarshal(payload, source); err != nil {
			return fmt.Errorf("unmarshal source %x: %w", row.EventHash, err)
		}

		derived, ok := def.Derive(source)
		if !ok || derived == nil {
			rep.Skipped++
			return nil
		}
		derived.DefinitionVersion = def.Version()
		derived.DefinitionParameters = def.Parameters()
		derived.SourceEventHash = row.EventHash[:]

		derivedPayload, derivedHash, err := canonical.MarshalAndHash(derived)
		if err != nil {
			return fmt.Errorf("marshal derived: %w", err)
		}
		derivedHex := canonical.HashHex(derivedHash)

		_, lookupErr := sub.LookupRow(ctx, derivedHash)
		alreadyPresent := lookupErr == nil
		if lookupErr != nil && !errors.Is(lookupErr, sql.ErrNoRows) {
			return fmt.Errorf("lookup derived %s: %w", derivedHex, lookupErr)
		}

		derivedRow := substrate.EventRow{
			EventHash:   derivedHash,
			EventTime:   row.EventTime,
			MessageType: string(derived.ProtoReflect().Descriptor().FullName()),
			PayloadRef:  derivedHex[:2] + "/" + derivedHex[2:],
			CommittedAt: now().UnixNano(),
		}
		if err := sub.Append(ctx, derivedRow, derivedPayload); err != nil {
			return fmt.Errorf("append derived %s: %w", derivedHex, err)
		}

		if alreadyPresent {
			rep.AlreadyDerived++
		} else {
			rep.NewlyDerived++
		}
		return nil
	})
	if walkErr != nil {
		return rep, fmt.Errorf("attribution.DeriveAll: %w", walkErr)
	}
	return rep, nil
}

// AttributionView provides read-only lookup from Cat I source hash to
// derived attribution. F3 signatures consume this view to obtain Cat II
// derived attribution for Cat I observations that lack declared actor_ref.
// Per §0168 Decision A.1: explicit parameter (not silent back-fill).
type AttributionView interface {
	// For returns the derived actor_ref and the Cat II derivation
	// record's content-hash for the given Cat I source hash, plus ok
	// = true. When no Cat II derivation exists for the source, returns
	// "", [32]byte{}, false. The Cat II hash is returned so the
	// consuming signature can thread it through to the resulting
	// hypothesis's source_event_hashes (preserves §2.3 chain).
	For(sourceHash [32]byte) (derivedActorRef string, attributionHash [32]byte, ok bool)
}

// substrateView implements AttributionView over a substrate snapshot.
// Constructed via CollectAttributionView (one substrate walk).
type substrateView struct {
	bySource map[[32]byte]attributionEntry
}

type attributionEntry struct {
	derivedActorRef string
	attributionHash [32]byte
}

// For implements AttributionView.
func (v *substrateView) For(sourceHash [32]byte) (string, [32]byte, bool) {
	e, ok := v.bySource[sourceHash]
	if !ok {
		return "", [32]byte{}, false
	}
	return e.derivedActorRef, e.attributionHash, true
}

// CollectAttributionView walks the substrate once and collects every
// DerivedActorAttribution row into an in-memory map keyed by Cat I
// source hash. Returns an AttributionView suitable for passing to F3
// signature Evaluate methods.
//
// When the substrate carries multiple DerivedActorAttribution records
// pointing to the SAME source_event_hash (e.g., two definition versions
// derived the same Cat I source), the LAST one walked wins. The order
// of substrate.WalkEvents is deterministic per substrate implementation
// (event_hash ascending); the "last wins" outcome is therefore
// deterministic per substrate state. Future definitions resolving the
// multi-derivation case may add a definition-version selector parameter.
func CollectAttributionView(ctx context.Context, sub *substrate.Substrate) (AttributionView, error) {
	v := &substrateView{bySource: map[[32]byte]attributionEntry{}}
	err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != derivedActorAttributionMessageType {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return fmt.Errorf("read attribution blob %x: %w", row.EventHash, err)
		}
		att := &eventsv1.DerivedActorAttribution{}
		if err := proto.Unmarshal(payload, att); err != nil {
			return fmt.Errorf("unmarshal attribution %x: %w", row.EventHash, err)
		}
		if len(att.SourceEventHash) != 32 {
			return nil
		}
		var src [32]byte
		copy(src[:], att.SourceEventHash)
		v.bySource[src] = attributionEntry{
			derivedActorRef: att.DerivedActorRef,
			attributionHash: row.EventHash,
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("attribution.CollectAttributionView: %w", err)
	}
	return v, nil
}

// EmptyAttributionView is a fixed-empty AttributionView. Use when the
// caller does not have a substrate to walk + needs to invoke an F3
// signature with no Cat II attribution available (the backward-compat
// pre-§0168 behavior; signature treats all unattributed Cat I records
// as skip).
type EmptyAttributionView struct{}

// For implements AttributionView with empty result.
func (EmptyAttributionView) For(sourceHash [32]byte) (string, [32]byte, bool) {
	return "", [32]byte{}, false
}
