package hypothesis

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// DissolveOptions configures a single dissolution. The
// (FormationEventHash, DissolvedAt, Reason) tuple uniquely
// identifies a dissolution event's content-hash — re-running with
// identical values is idempotent.
type DissolveOptions struct {
	// FormationEventHash is the content-hash of the
	// BehavioralClusterFormation event identifying the hypothesis
	// being dissolved (per §0045 — hypothesis identity IS the
	// formation event's content-hash). Dissolution targets the
	// formation directly; no two-hop indirection through promotion,
	// unlike demotion (which targets a specific promotion event).
	FormationEventHash [32]byte

	// DissolvedAt is the Unix-nanoseconds time at which dissolution
	// is recorded. Zero defaults to now().UnixNano(). Explicit
	// non-zero values permitted for forensic replay + deterministic
	// test recording.
	DissolvedAt int64

	// Reason is an operator-supplied free-form note. Optional;
	// recorded for forensic replay. Dissolution is the terminal
	// lifecycle operation on a hypothesis; the absence of a reason
	// removes the only record of the underlying judgment.
	Reason string

	// Actor is an optional per-actor attribution string per §0108
	// T4 dissolve landing — non-empty triggers AppendPair with
	// IngestionEvent (channel="cli", client_common_name=Actor).
	Actor string
}

// DissolveReport is the per-Dissolve outcome.
type DissolveReport struct {
	// DissolutionEventHashHex is the content-hash (hex) of the
	// recorded BehavioralClusterDissolution event.
	DissolutionEventHashHex string

	// AlreadyDissolved is true when an identical dissolution event
	// was already in the substrate (content-hash collision). False
	// when this Dissolve invocation committed a new row.
	AlreadyDissolved bool

	// IngestionEventHashHex is the content-hash (hex) of the paired
	// IngestionEvent committed when Actor was non-empty.
	IngestionEventHashHex string
}

// Dissolve records a BehavioralClusterDissolution lifecycle event
// against the BehavioralClusterFormation identified by
// opts.FormationEventHash. Per Charter §2.5 BC5 the dissolution
// event is a Category I record committed via substrate.Append
// (acquires writeMu per concurrency-pattern.md §Substrate-Writer
// Serialization).
//
// Per glossary + lifecycle-semantics.md §Hypothesis (Category III)
// line 36, dissolution is DISTINGUISHED from demotion: demotion
// withdraws OPERATIONAL USE; dissolution recognizes NON-EXISTENCE
// of the underlying phenomenon. Dissolve does not require that the
// hypothesis was ever promoted; a formation may be dissolved
// directly. The substrate accepts the discrete lifecycle event
// regardless of intermediate promotion/demotion state — the
// projection layer reconstructs the full chain per §2.5 BC3.
//
// Errors:
//   - ErrTargetNotFound: the formation hash does not resolve to any
//     substrate row.
//   - ErrTargetWrongType: the target hash resolves to a row whose
//     message_type is NOT BehavioralClusterFormation.
func Dissolve(ctx context.Context, sub *substrate.Substrate, opts DissolveOptions, now func() time.Time) (DissolveReport, error) {
	if now == nil {
		now = time.Now
	}

	row, err := sub.LookupRow(ctx, opts.FormationEventHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return DissolveReport{}, fmt.Errorf("%w: %x", ErrTargetNotFound, opts.FormationEventHash)
		}
		return DissolveReport{}, fmt.Errorf("hypothesis.Dissolve: lookup target: %w", err)
	}
	if row.MessageType != behavioralClusterFormationMessageType {
		return DissolveReport{}, fmt.Errorf("%w: %x is %q", ErrTargetWrongType, opts.FormationEventHash, row.MessageType)
	}

	dissolvedAt := opts.DissolvedAt
	if dissolvedAt == 0 {
		dissolvedAt = now().UnixNano()
	}

	ev := &eventsv1.BehavioralClusterDissolution{
		FormationEventHash: opts.FormationEventHash[:],
		DissolvedAt:        dissolvedAt,
		Reason:             opts.Reason,
	}
	payload, hash, err := canonical.MarshalAndHash(ev)
	if err != nil {
		return DissolveReport{}, fmt.Errorf("hypothesis.Dissolve: marshal dissolution: %w", err)
	}
	hex := canonical.HashHex(hash)

	_, lookupErr := sub.LookupRow(ctx, hash)
	alreadyPresent := lookupErr == nil
	if lookupErr != nil && !errors.Is(lookupErr, sql.ErrNoRows) {
		return DissolveReport{}, fmt.Errorf("hypothesis.Dissolve: lookup dissolution %s: %w", hex, lookupErr)
	}

	committedAt := now().UnixNano()
	dissRow := substrate.EventRow{
		EventHash:   hash,
		EventTime:   dissolvedAt,
		MessageType: string(ev.ProtoReflect().Descriptor().FullName()),
		PayloadRef:  hex[:2] + "/" + hex[2:],
		CommittedAt: committedAt,
	}

	if opts.Actor == "" {
		if err := sub.Append(ctx, dissRow, payload); err != nil {
			return DissolveReport{}, fmt.Errorf("hypothesis.Dissolve: append dissolution %s: %w", hex, err)
		}
		return DissolveReport{
			DissolutionEventHashHex: hex,
			AlreadyDissolved:        alreadyPresent,
		}, nil
	}

	ingEv := &eventsv1.IngestionEvent{
		PrimaryEventHash: hash[:],
		ReceivedAt:       committedAt,
		IngestedAt:       committedAt,
		Channel:          "cli",
		ClientCommonName: opts.Actor,
	}
	ingPayload, ingHash, err := canonical.MarshalAndHash(ingEv)
	if err != nil {
		return DissolveReport{}, fmt.Errorf("hypothesis.Dissolve: marshal ingestion event: %w", err)
	}
	ingHex := canonical.HashHex(ingHash)
	ingRow := substrate.EventRow{
		EventHash:   ingHash,
		EventTime:   committedAt,
		MessageType: string(ingEv.ProtoReflect().Descriptor().FullName()),
		PayloadRef:  ingHex[:2] + "/" + ingHex[2:],
		CommittedAt: committedAt,
	}
	if err := sub.AppendPair(ctx, dissRow, payload, ingRow, ingPayload); err != nil {
		return DissolveReport{}, fmt.Errorf("hypothesis.Dissolve: append pair (dissolution %s, ingestion %s): %w", hex, ingHex, err)
	}

	return DissolveReport{
		DissolutionEventHashHex: hex,
		AlreadyDissolved:        alreadyPresent,
		IngestionEventHashHex:   ingHex,
	}, nil
}
