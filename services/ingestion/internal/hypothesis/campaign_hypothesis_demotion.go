package hypothesis

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

// campaignHypothesisPromotionMessageType is the substrate's
// message_type discriminator for CampaignHypothesisPromotion rows.
const campaignHypothesisPromotionMessageType = "ghosttrace.events.v1.CampaignHypothesisPromotion"

// CampaignHypothesisDemoteOptions configures a single CampaignHypothesis
// demotion. Mirrors §0047's DemoteOptions and §0058's AG variant.
type CampaignHypothesisDemoteOptions struct {
	PromotionEventHash [32]byte
	DemotedAt          int64
	Reason             string
}

// CampaignHypothesisDemoteReport is the per-DemoteCampaignHypothesis
// outcome.
type CampaignHypothesisDemoteReport struct {
	DemotionEventHashHex  string
	AlreadyDemoted        bool
	CadenceSatisfied      bool
	CadenceElapsedSeconds int64
}

// DemoteCampaignHypothesis records a CampaignHypothesisDemotion
// lifecycle event against the CampaignHypothesisPromotion identified
// by opts.PromotionEventHash. Per Charter §2.5 BC5 the demotion
// event is a Category I record committed via substrate.Append.
//
// Per §0011 Layer A is a CANDIDACY criterion, not hard barrier.
// DemoteCampaignHypothesis records demotion regardless of whether
// the cadence has elapsed; the report surfaces the gate state.
//
// Errors:
//   - ErrTargetNotFound: promotion hash does not resolve to any row.
//   - ErrTargetWrongType: target hash is NOT a
//     CampaignHypothesisPromotion.
func DemoteCampaignHypothesis(ctx context.Context, sub *substrate.Substrate, opts CampaignHypothesisDemoteOptions, now func() time.Time) (CampaignHypothesisDemoteReport, error) {
	if now == nil {
		now = time.Now
	}

	row, err := sub.LookupRow(ctx, opts.PromotionEventHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CampaignHypothesisDemoteReport{}, fmt.Errorf("%w: %x", ErrTargetNotFound, opts.PromotionEventHash)
		}
		return CampaignHypothesisDemoteReport{}, fmt.Errorf("hypothesis.DemoteCampaignHypothesis: lookup target: %w", err)
	}
	if row.MessageType != campaignHypothesisPromotionMessageType {
		return CampaignHypothesisDemoteReport{}, fmt.Errorf("%w: %x is %q", ErrTargetWrongType, opts.PromotionEventHash, row.MessageType)
	}

	payload, err := sub.ReadBlob(ctx, opts.PromotionEventHash)
	if err != nil {
		return CampaignHypothesisDemoteReport{}, fmt.Errorf("hypothesis.DemoteCampaignHypothesis: read promotion blob: %w", err)
	}
	promotion := &eventsv1.CampaignHypothesisPromotion{}
	if err := proto.Unmarshal(payload, promotion); err != nil {
		return CampaignHypothesisDemoteReport{}, fmt.Errorf("hypothesis.DemoteCampaignHypothesis: unmarshal promotion: %w", err)
	}

	demotedAt := opts.DemotedAt
	if demotedAt == 0 {
		demotedAt = now().UnixNano()
	}

	elapsedSeconds := (demotedAt - promotion.GetPromotedAt()) / int64(time.Second)
	cadenceSatisfied := elapsedSeconds >= promotion.GetCadenceSeconds()

	ev := &eventsv1.CampaignHypothesisDemotion{
		PromotionEventHash: opts.PromotionEventHash[:],
		DemotedAt:          demotedAt,
		Reason:             opts.Reason,
	}
	demotionPayload, hash, err := canonical.MarshalAndHash(ev)
	if err != nil {
		return CampaignHypothesisDemoteReport{}, fmt.Errorf("hypothesis.DemoteCampaignHypothesis: marshal demotion: %w", err)
	}
	hex := canonical.HashHex(hash)

	_, lookupErr := sub.LookupRow(ctx, hash)
	alreadyPresent := lookupErr == nil
	if lookupErr != nil && !errors.Is(lookupErr, sql.ErrNoRows) {
		return CampaignHypothesisDemoteReport{}, fmt.Errorf("hypothesis.DemoteCampaignHypothesis: lookup demotion %s: %w", hex, lookupErr)
	}

	demoRow := substrate.EventRow{
		EventHash:   hash,
		EventTime:   demotedAt,
		MessageType: string(ev.ProtoReflect().Descriptor().FullName()),
		PayloadRef:  hex[:2] + "/" + hex[2:],
		CommittedAt: now().UnixNano(),
	}
	if err := sub.Append(ctx, demoRow, demotionPayload); err != nil {
		return CampaignHypothesisDemoteReport{}, fmt.Errorf("hypothesis.DemoteCampaignHypothesis: append demotion %s: %w", hex, err)
	}

	return CampaignHypothesisDemoteReport{
		DemotionEventHashHex:  hex,
		AlreadyDemoted:        alreadyPresent,
		CadenceSatisfied:      cadenceSatisfied,
		CadenceElapsedSeconds: elapsedSeconds,
	}, nil
}
