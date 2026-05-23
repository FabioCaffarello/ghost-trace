package hypothesis

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// CoordinationRingPromoteOptions configures a single CoordinationRing
// promotion. Mirrors §0046/§0057/§0064 promote options for the
// fourth-subtype landing.
type CoordinationRingPromoteOptions struct {
	// FormationEventHash is the content-hash of the originating
	// CoordinationRingFormation event.
	FormationEventHash [32]byte

	// PromotedAt is the Unix-nanoseconds time from which the Layer A
	// cadence gate is measured. Zero defaults to now().UnixNano().
	PromotedAt int64

	// CadenceSeconds is the Layer A parameter (per §0011);
	// mandatory. Values <= 0 are rejected.
	CadenceSeconds int64

	// Reason is an operator-supplied free-form note.
	Reason string

	// Actor is an optional per-actor attribution string. Mirrors
	// PromoteOptions.Actor per §0097 BehavioralCluster pilot —
	// extended to CoordinationRing at the §0105 mechanical-replication
	// landing. When non-empty, PromoteCoordinationRing commits the
	// CoordinationRingPromotion event paired with an IngestionEvent
	// via AppendPair. Empty preserves the single-Append path.
	Actor string

	// LayerBParameters bundles the demotion-candidacy parameters per
	// §0138; mirrors PromoteOptions.LayerBParameters. When non-nil,
	// written to the promotion event's layer_b_parameters field;
	// when nil, the field remains unset (legacy path).
	LayerBParameters *commonv1.LayerBParameters
}

// CoordinationRingPromoteReport is the per-PromoteCoordinationRing
// outcome.
type CoordinationRingPromoteReport struct {
	PromotionEventHashHex string
	AlreadyPromoted       bool

	// IngestionEventHashHex is the content-hash (hex) of the paired
	// IngestionEvent committed when Actor was non-empty.
	IngestionEventHashHex string
}

// PromoteCoordinationRing records a CoordinationRingPromotion
// lifecycle event against the CoordinationRingFormation identified
// by opts.FormationEventHash. Per Charter §2.5 BC5 the promotion
// event is a Cat I record committed via substrate.Append.
//
// Errors:
//   - ErrTargetNotFound: formation hash does not resolve to any row.
//   - ErrTargetWrongType: target hash is NOT a
//     CoordinationRingFormation.
//   - validation errors when opts.CadenceSeconds <= 0.
//
// Sentinels are SHARED across all four subtypes per the §0046+§0047
// design pattern carried forward to §0057/§0064. The wrong-type
// check uses coordinationRingFormationMessageType; a BC/AG/CH
// formation hash passed here returns ErrTargetWrongType.
func PromoteCoordinationRing(ctx context.Context, sub *substrate.Substrate, opts CoordinationRingPromoteOptions, now func() time.Time) (CoordinationRingPromoteReport, error) {
	if opts.CadenceSeconds <= 0 {
		return CoordinationRingPromoteReport{}, fmt.Errorf("hypothesis.PromoteCoordinationRing: cadence_seconds must be positive (got %d)", opts.CadenceSeconds)
	}
	if now == nil {
		now = time.Now
	}

	row, err := sub.LookupRow(ctx, opts.FormationEventHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CoordinationRingPromoteReport{}, fmt.Errorf("%w: %x", ErrTargetNotFound, opts.FormationEventHash)
		}
		return CoordinationRingPromoteReport{}, fmt.Errorf("hypothesis.PromoteCoordinationRing: lookup target: %w", err)
	}
	if row.MessageType != coordinationRingFormationMessageType {
		return CoordinationRingPromoteReport{}, fmt.Errorf("%w: %x is %q", ErrTargetWrongType, opts.FormationEventHash, row.MessageType)
	}

	promotedAt := opts.PromotedAt
	if promotedAt == 0 {
		promotedAt = now().UnixNano()
	}

	ev := &eventsv1.CoordinationRingPromotion{
		FormationEventHash: opts.FormationEventHash[:],
		PromotedAt:         promotedAt,
		CadenceSeconds:     opts.CadenceSeconds,
		Reason:             opts.Reason,
		LayerBParameters:   opts.LayerBParameters,
	}
	payload, hash, err := canonical.MarshalAndHash(ev)
	if err != nil {
		return CoordinationRingPromoteReport{}, fmt.Errorf("hypothesis.PromoteCoordinationRing: marshal promotion: %w", err)
	}
	hex := canonical.HashHex(hash)

	_, lookupErr := sub.LookupRow(ctx, hash)
	alreadyPresent := lookupErr == nil
	if lookupErr != nil && !errors.Is(lookupErr, sql.ErrNoRows) {
		return CoordinationRingPromoteReport{}, fmt.Errorf("hypothesis.PromoteCoordinationRing: lookup promotion %s: %w", hex, lookupErr)
	}

	committedAt := now().UnixNano()
	promRow := substrate.EventRow{
		EventHash:   hash,
		EventTime:   promotedAt,
		MessageType: string(ev.ProtoReflect().Descriptor().FullName()),
		PayloadRef:  hex[:2] + "/" + hex[2:],
		CommittedAt: committedAt,
	}

	if opts.Actor == "" {
		if err := sub.Append(ctx, promRow, payload); err != nil {
			return CoordinationRingPromoteReport{}, fmt.Errorf("hypothesis.PromoteCoordinationRing: append promotion %s: %w", hex, err)
		}
		return CoordinationRingPromoteReport{
			PromotionEventHashHex: hex,
			AlreadyPromoted:       alreadyPresent,
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
		return CoordinationRingPromoteReport{}, fmt.Errorf("hypothesis.PromoteCoordinationRing: marshal ingestion event: %w", err)
	}
	ingHex := canonical.HashHex(ingHash)
	ingRow := substrate.EventRow{
		EventHash:   ingHash,
		EventTime:   committedAt,
		MessageType: string(ingEv.ProtoReflect().Descriptor().FullName()),
		PayloadRef:  ingHex[:2] + "/" + ingHex[2:],
		CommittedAt: committedAt,
	}
	if err := sub.AppendPair(ctx, promRow, payload, ingRow, ingPayload); err != nil {
		return CoordinationRingPromoteReport{}, fmt.Errorf("hypothesis.PromoteCoordinationRing: append pair (promotion %s, ingestion %s): %w", hex, ingHex, err)
	}

	return CoordinationRingPromoteReport{
		PromotionEventHashHex: hex,
		AlreadyPromoted:       alreadyPresent,
		IngestionEventHashHex: ingHex,
	}, nil
}
