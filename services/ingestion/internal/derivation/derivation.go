// Package derivation implements Category II operational-construct
// derivation per Charter §2.2 + entity-model.md §Category II +
// decision-log §0015 (Q1 resolution — OperationalSession is the
// canonical Cat II construct).
//
// Derivation is **deterministic** with respect to its inputs + the
// operational definition's (version, parameters) tuple per
// entity-model.md §Category II Structural properties. Non-
// deterministic derivation would constitute a Category III
// misclassification and is rejected at this layer (the contract is
// enforceable at the OperationalDefinition interface boundary; a
// definition whose Derive method is non-deterministic violates the
// contract).
//
// Identity composition per entity-model.md line 44: an
// OperationalSession's identity composes (definition_version,
// definition_parameters, source_event_hash). Re-derivation under a NEW
// (definition_version, definition_parameters) tuple produces a NEW
// substrate record; the prior derivation is preserved. Re-derivation
// under an IDENTICAL tuple is a no-op (substrate's INSERT OR IGNORE
// rejects the duplicate by content-hash collision).
//
// Substrate-write discipline per concurrency-pattern.md §Substrate-
// Writer Serialization: derivation writes go through substrate.Append,
// which acquires writeMu. The derivation walker is allowed to run
// concurrently with the ingestion service's write path; the substrate
// serializes both into the events table.
package derivation

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

// declaredSessionMessageType is the substrate's message_type
// discriminator for DeclaredSession rows. The walker selects on this
// to skip non-source rows (IngestionEvent enrichments, NetworkEvent
// Cat I rows, OperationalSession rows from prior derivations).
const declaredSessionMessageType = "ghosttrace.events.v1.DeclaredSession"

// OperationalDefinition is the deterministic-derivation contract.
// Implementations MUST be deterministic with respect to the source +
// the Parameters they encode; the test suite at
// derivation_test.go#TestDeriveDeterminism enforces this structurally
// by recomputing twice and comparing content-hashes.
type OperationalDefinition interface {
	// Version is the stable identifier of this operational definition
	// (e.g. "padded-v1"). Part of the produced OperationalSession's
	// identity per entity-model.md line 46.
	Version() string

	// Parameters returns the canonical-form string encoding of this
	// definition's parameters. Part of identity per entity-model.md
	// line 44. Canonical form: lowercase key=value pairs separated by
	// semicolons, sorted by key. Two definitions with logically-
	// identical parameters MUST produce byte-identical Parameters
	// strings (otherwise the substrate stores spuriously-distinct
	// records).
	Parameters() string

	// Derive produces the OperationalSession for the given source
	// DeclaredSession + its source_event_hash. The output's
	// definition_version + definition_parameters fields are set by
	// the caller (DeriveAll); the implementation populates the
	// derivation-specific fields (operational_start_at,
	// operational_end_at, actor_ref).
	Derive(source *eventsv1.DeclaredSession, sourceHash [32]byte) *eventsv1.OperationalSession
}

// Report is the per-DeriveAll outcome. Examined counts every
// DeclaredSession walked; NewlyDerived counts the OperationalSessions
// committed in this run; AlreadyDerived counts the
// OperationalSessions whose content-hash collided with an existing
// substrate row (per content-addressed idempotency).
type Report struct {
	Examined       int64
	NewlyDerived   int64
	AlreadyDerived int64
}

// DeriveAll walks every DeclaredSession in the substrate, applies def
// to each, and commits the resulting OperationalSession via
// substrate.Append. The commit is idempotent: re-running with the
// same def produces zero new rows (NewlyDerived=0, AlreadyDerived
// equal to Examined) because the substrate's PRIMARY KEY constraint
// on event_hash rejects duplicate content.
//
// Concurrency: DeriveAll reads via WalkEvents (no writeMu) + writes
// via substrate.Append (acquires writeMu). Safe to run alongside the
// ingestion service's write path; the substrate serializes all
// writers per concurrency-pattern.md §Substrate-Writer Serialization.
func DeriveAll(ctx context.Context, sub *substrate.Substrate, def OperationalDefinition, now func() time.Time) (Report, error) {
	if def == nil {
		return Report{}, errors.New("derivation.DeriveAll: definition must not be nil")
	}
	if now == nil {
		now = time.Now
	}

	var rep Report
	walkErr := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != declaredSessionMessageType {
			return nil
		}
		rep.Examined++

		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return fmt.Errorf("read source blob %x: %w", row.EventHash, err)
		}
		source := &eventsv1.DeclaredSession{}
		if err := proto.Unmarshal(payload, source); err != nil {
			return fmt.Errorf("unmarshal source %x: %w", row.EventHash, err)
		}

		derived := def.Derive(source, row.EventHash)
		derived.DefinitionVersion = def.Version()
		derived.DefinitionParameters = def.Parameters()
		derived.SourceEventHash = row.EventHash[:]

		derivedPayload, derivedHash, err := canonical.MarshalAndHash(derived)
		if err != nil {
			return fmt.Errorf("marshal derived: %w", err)
		}
		derivedHex := canonical.HashHex(derivedHash)

		// Pre-flight: lookup whether a row with this content-hash
		// already exists. Used to distinguish NewlyDerived from
		// AlreadyDerived in the Report; substrate.Append is
		// idempotent regardless.
		_, lookupErr := sub.LookupRow(ctx, derivedHash)
		alreadyPresent := lookupErr == nil
		if lookupErr != nil && !errors.Is(lookupErr, sql.ErrNoRows) {
			return fmt.Errorf("lookup derived %s: %w", derivedHex, lookupErr)
		}

		derivedRow := substrate.EventRow{
			EventHash:   derivedHash,
			EventTime:   derived.OperationalStartAt,
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
		return rep, fmt.Errorf("derivation.DeriveAll: %w", walkErr)
	}
	return rep, nil
}
