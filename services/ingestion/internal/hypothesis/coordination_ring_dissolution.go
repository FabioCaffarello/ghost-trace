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

// CoordinationRingDissolveOptions configures a single CoordinationRing
// dissolution. Mirrors §0048+§0059+§0066 dissolve options for the
// fourth-subtype landing.
type CoordinationRingDissolveOptions struct {
	FormationEventHash [32]byte
	DissolvedAt        int64
	Reason             string

	// Actor per §0108 — non-empty triggers AppendPair.
	Actor string
}

// CoordinationRingDissolveReport is the per-DissolveCoordinationRing
// outcome.
type CoordinationRingDissolveReport struct {
	DissolutionEventHashHex string
	AlreadyDissolved        bool

	// IngestionEventHashHex non-empty when Actor was supplied.
	IngestionEventHashHex string
}

// DissolveCoordinationRing records a CoordinationRingDissolution
// lifecycle event against the CoordinationRingFormation identified
// by opts.FormationEventHash. Per Charter §2.5 BC5 the dissolution
// event is a Cat I record committed via substrate.Append.
//
// Per glossary + lifecycle-semantics.md, dissolution is
// distinguished from demotion: demotion withdraws OPERATIONAL USE;
// dissolution recognizes NON-EXISTENCE. The distinction transfers
// across all four Cat III subtypes.
//
// Errors:
//   - ErrTargetNotFound: formation hash does not resolve to any row.
//   - ErrTargetWrongType: target hash is NOT a
//     CoordinationRingFormation.
func DissolveCoordinationRing(ctx context.Context, sub *substrate.Substrate, opts CoordinationRingDissolveOptions, now func() time.Time) (CoordinationRingDissolveReport, error) {
	if now == nil {
		now = time.Now
	}

	row, err := sub.LookupRow(ctx, opts.FormationEventHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CoordinationRingDissolveReport{}, fmt.Errorf("%w: %x", ErrTargetNotFound, opts.FormationEventHash)
		}
		return CoordinationRingDissolveReport{}, fmt.Errorf("hypothesis.DissolveCoordinationRing: lookup target: %w", err)
	}
	if row.MessageType != coordinationRingFormationMessageType {
		return CoordinationRingDissolveReport{}, fmt.Errorf("%w: %x is %q", ErrTargetWrongType, opts.FormationEventHash, row.MessageType)
	}

	dissolvedAt := opts.DissolvedAt
	if dissolvedAt == 0 {
		dissolvedAt = now().UnixNano()
	}

	ev := &eventsv1.CoordinationRingDissolution{
		FormationEventHash: opts.FormationEventHash[:],
		DissolvedAt:        dissolvedAt,
		Reason:             opts.Reason,
	}
	payload, hash, err := canonical.MarshalAndHash(ev)
	if err != nil {
		return CoordinationRingDissolveReport{}, fmt.Errorf("hypothesis.DissolveCoordinationRing: marshal dissolution: %w", err)
	}
	hex := canonical.HashHex(hash)

	_, lookupErr := sub.LookupRow(ctx, hash)
	alreadyPresent := lookupErr == nil
	if lookupErr != nil && !errors.Is(lookupErr, sql.ErrNoRows) {
		return CoordinationRingDissolveReport{}, fmt.Errorf("hypothesis.DissolveCoordinationRing: lookup dissolution %s: %w", hex, lookupErr)
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
			return CoordinationRingDissolveReport{}, fmt.Errorf("hypothesis.DissolveCoordinationRing: append dissolution %s: %w", hex, err)
		}
		return CoordinationRingDissolveReport{
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
		return CoordinationRingDissolveReport{}, fmt.Errorf("hypothesis.DissolveCoordinationRing: marshal ingestion event: %w", err)
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
		return CoordinationRingDissolveReport{}, fmt.Errorf("hypothesis.DissolveCoordinationRing: append pair (dissolution %s, ingestion %s): %w", hex, ingHex, err)
	}

	return CoordinationRingDissolveReport{
		DissolutionEventHashHex: hex,
		AlreadyDissolved:        alreadyPresent,
		IngestionEventHashHex:   ingHex,
	}, nil
}
