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

// CampaignHypothesisDissolveOptions configures a single
// CampaignHypothesis dissolution. Mirrors §0048's DissolveOptions
// and §0059's AG variant.
type CampaignHypothesisDissolveOptions struct {
	FormationEventHash [32]byte
	DissolvedAt        int64
	Reason             string

	// Actor per §0108 — non-empty triggers AppendPair.
	Actor string
}

// CampaignHypothesisDissolveReport is the per-DissolveCampaignHypothesis
// outcome.
type CampaignHypothesisDissolveReport struct {
	DissolutionEventHashHex string
	AlreadyDissolved        bool

	// IngestionEventHashHex non-empty when Actor was supplied.
	IngestionEventHashHex string
}

// DissolveCampaignHypothesis records a CampaignHypothesisDissolution
// lifecycle event against the CampaignHypothesisFormation identified
// by opts.FormationEventHash. Per Charter §2.5 BC5 the dissolution
// event is a Cat I record committed via substrate.Append.
//
// Per glossary + lifecycle-semantics.md, dissolution is distinguished
// from demotion: demotion withdraws OPERATIONAL USE; dissolution
// recognizes NON-EXISTENCE. The distinction transfers across all
// three Cat III subtypes.
//
// Errors:
//   - ErrTargetNotFound: formation hash does not resolve to any row.
//   - ErrTargetWrongType: target hash is NOT a
//     CampaignHypothesisFormation.
func DissolveCampaignHypothesis(ctx context.Context, sub *substrate.Substrate, opts CampaignHypothesisDissolveOptions, now func() time.Time) (CampaignHypothesisDissolveReport, error) {
	if now == nil {
		now = time.Now
	}

	row, err := sub.LookupRow(ctx, opts.FormationEventHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CampaignHypothesisDissolveReport{}, fmt.Errorf("%w: %x", ErrTargetNotFound, opts.FormationEventHash)
		}
		return CampaignHypothesisDissolveReport{}, fmt.Errorf("hypothesis.DissolveCampaignHypothesis: lookup target: %w", err)
	}
	if row.MessageType != campaignHypothesisFormationMessageType {
		return CampaignHypothesisDissolveReport{}, fmt.Errorf("%w: %x is %q", ErrTargetWrongType, opts.FormationEventHash, row.MessageType)
	}

	dissolvedAt := opts.DissolvedAt
	if dissolvedAt == 0 {
		dissolvedAt = now().UnixNano()
	}

	ev := &eventsv1.CampaignHypothesisDissolution{
		FormationEventHash: opts.FormationEventHash[:],
		DissolvedAt:        dissolvedAt,
		Reason:             opts.Reason,
	}
	payload, hash, err := canonical.MarshalAndHash(ev)
	if err != nil {
		return CampaignHypothesisDissolveReport{}, fmt.Errorf("hypothesis.DissolveCampaignHypothesis: marshal dissolution: %w", err)
	}
	hex := canonical.HashHex(hash)

	_, lookupErr := sub.LookupRow(ctx, hash)
	alreadyPresent := lookupErr == nil
	if lookupErr != nil && !errors.Is(lookupErr, sql.ErrNoRows) {
		return CampaignHypothesisDissolveReport{}, fmt.Errorf("hypothesis.DissolveCampaignHypothesis: lookup dissolution %s: %w", hex, lookupErr)
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
			return CampaignHypothesisDissolveReport{}, fmt.Errorf("hypothesis.DissolveCampaignHypothesis: append dissolution %s: %w", hex, err)
		}
		return CampaignHypothesisDissolveReport{
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
		return CampaignHypothesisDissolveReport{}, fmt.Errorf("hypothesis.DissolveCampaignHypothesis: marshal ingestion event: %w", err)
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
		return CampaignHypothesisDissolveReport{}, fmt.Errorf("hypothesis.DissolveCampaignHypothesis: append pair (dissolution %s, ingestion %s): %w", hex, ingHex, err)
	}

	return CampaignHypothesisDissolveReport{
		DissolutionEventHashHex: hex,
		AlreadyDissolved:        alreadyPresent,
		IngestionEventHashHex:   ingHex,
	}, nil
}
