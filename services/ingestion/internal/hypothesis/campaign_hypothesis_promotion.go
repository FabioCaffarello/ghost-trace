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

// campaignHypothesisFormationMessageType is the substrate's
// message_type discriminator for CampaignHypothesisFormation rows.
// PromoteCampaignHypothesis validates the target hash points to a
// row of this type — preserves §2.5-lifecycle-integrity for the
// third subtype's arc (mirrors §0046 + §0057 + §0058 pattern).
const campaignHypothesisFormationMessageType = "ghosttrace.events.v1.CampaignHypothesisFormation"

// CampaignHypothesisPromoteOptions configures a single
// CampaignHypothesis promotion. Mirrors §0046's PromoteOptions and
// §0057's AutomationGroupPromoteOptions for the third-subtype
// landing.
type CampaignHypothesisPromoteOptions struct {
	// FormationEventHash is the content-hash of the originating
	// CampaignHypothesisFormation event.
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
	// PromoteOptions.Actor per §0097 BehavioralCluster pilot — extended
	// to CampaignHypothesis at the §0105 mechanical-replication
	// landing. When non-empty, PromoteCampaignHypothesis commits the
	// CampaignHypothesisPromotion event paired with an IngestionEvent
	// via AppendPair. Empty preserves the single-Append path.
	Actor string

	// LayerBParameters bundles the demotion-candidacy parameters per
	// §0138; mirrors PromoteOptions.LayerBParameters. When non-nil,
	// written to the promotion event's layer_b_parameters field;
	// when nil, the field remains unset (legacy path).
	LayerBParameters *commonv1.LayerBParameters
}

// CampaignHypothesisPromoteReport is the per-PromoteCampaignHypothesis
// outcome.
type CampaignHypothesisPromoteReport struct {
	// PromotionEventHashHex is the content-hash (hex) of the
	// recorded CampaignHypothesisPromotion event.
	PromotionEventHashHex string

	// AlreadyPromoted is true when an identical promotion event
	// was already in the substrate.
	AlreadyPromoted bool

	// IngestionEventHashHex is the content-hash (hex) of the paired
	// IngestionEvent committed when Actor was non-empty. Empty
	// otherwise.
	IngestionEventHashHex string
}

// PromoteCampaignHypothesis records a CampaignHypothesisPromotion
// lifecycle event against the CampaignHypothesisFormation
// identified by opts.FormationEventHash. Per Charter §2.5 BC5 the
// promotion event is a Category I record committed via
// substrate.Append.
//
// Errors:
//   - ErrTargetNotFound: formation hash does not resolve to any row.
//   - ErrTargetWrongType: target hash is NOT a
//     CampaignHypothesisFormation.
//   - validation errors when opts.CadenceSeconds <= 0.
//
// Sentinels are SHARED across all three subtypes per the §0046+§0047
// design pattern; the wrong-type check uses the subtype-specific
// campaignHypothesisFormationMessageType constant. A BC or AG
// formation hash passed here will correctly trigger
// ErrTargetWrongType.
func PromoteCampaignHypothesis(ctx context.Context, sub *substrate.Substrate, opts CampaignHypothesisPromoteOptions, now func() time.Time) (CampaignHypothesisPromoteReport, error) {
	if opts.CadenceSeconds <= 0 {
		return CampaignHypothesisPromoteReport{}, fmt.Errorf("hypothesis.PromoteCampaignHypothesis: cadence_seconds must be positive (got %d)", opts.CadenceSeconds)
	}
	if now == nil {
		now = time.Now
	}

	row, err := sub.LookupRow(ctx, opts.FormationEventHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CampaignHypothesisPromoteReport{}, fmt.Errorf("%w: %x", ErrTargetNotFound, opts.FormationEventHash)
		}
		return CampaignHypothesisPromoteReport{}, fmt.Errorf("hypothesis.PromoteCampaignHypothesis: lookup target: %w", err)
	}
	if row.MessageType != campaignHypothesisFormationMessageType {
		return CampaignHypothesisPromoteReport{}, fmt.Errorf("%w: %x is %q", ErrTargetWrongType, opts.FormationEventHash, row.MessageType)
	}

	promotedAt := opts.PromotedAt
	if promotedAt == 0 {
		promotedAt = now().UnixNano()
	}

	ev := &eventsv1.CampaignHypothesisPromotion{
		FormationEventHash: opts.FormationEventHash[:],
		PromotedAt:         promotedAt,
		CadenceSeconds:     opts.CadenceSeconds,
		Reason:             opts.Reason,
		LayerBParameters:   opts.LayerBParameters,
	}
	payload, hash, err := canonical.MarshalAndHash(ev)
	if err != nil {
		return CampaignHypothesisPromoteReport{}, fmt.Errorf("hypothesis.PromoteCampaignHypothesis: marshal promotion: %w", err)
	}
	hex := canonical.HashHex(hash)

	_, lookupErr := sub.LookupRow(ctx, hash)
	alreadyPresent := lookupErr == nil
	if lookupErr != nil && !errors.Is(lookupErr, sql.ErrNoRows) {
		return CampaignHypothesisPromoteReport{}, fmt.Errorf("hypothesis.PromoteCampaignHypothesis: lookup promotion %s: %w", hex, lookupErr)
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
			return CampaignHypothesisPromoteReport{}, fmt.Errorf("hypothesis.PromoteCampaignHypothesis: append promotion %s: %w", hex, err)
		}
		return CampaignHypothesisPromoteReport{
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
		return CampaignHypothesisPromoteReport{}, fmt.Errorf("hypothesis.PromoteCampaignHypothesis: marshal ingestion event: %w", err)
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
		return CampaignHypothesisPromoteReport{}, fmt.Errorf("hypothesis.PromoteCampaignHypothesis: append pair (promotion %s, ingestion %s): %w", hex, ingHex, err)
	}

	return CampaignHypothesisPromoteReport{
		PromotionEventHashHex: hex,
		AlreadyPromoted:       alreadyPresent,
		IngestionEventHashHex: ingHex,
	}, nil
}
