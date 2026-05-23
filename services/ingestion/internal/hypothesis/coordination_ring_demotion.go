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

// coordinationRingPromotionMessageType is the substrate's
// message_type discriminator for CoordinationRingPromotion rows.
const coordinationRingPromotionMessageType = "ghosttrace.events.v1.CoordinationRingPromotion"

// CoordinationRingDemoteOptions configures a single CoordinationRing
// demotion. Mirrors §0047+§0058+§0065 demote options for the
// fourth-subtype landing.
type CoordinationRingDemoteOptions struct {
	PromotionEventHash [32]byte
	DemotedAt          int64
	Reason             string

	// Actor per §0107 T4 demote landing — non-empty triggers
	// AppendPair with IngestionEvent.
	Actor string
}

// CoordinationRingDemoteReport is the per-DemoteCoordinationRing
// outcome.
type CoordinationRingDemoteReport struct {
	DemotionEventHashHex  string
	AlreadyDemoted        bool
	CadenceSatisfied      bool
	CadenceElapsedSeconds int64

	// IngestionEventHashHex non-empty when Actor was supplied.
	IngestionEventHashHex string

	// LayerB is the Layer B deep-criterion verdict per §0141
	// sub-decision E1 (advisory like Layer A). Demote records the
	// demotion regardless of LayerB.Fired.
	LayerB LayerBReport
}

// DemoteCoordinationRing records a CoordinationRingDemotion lifecycle
// event against the CoordinationRingPromotion identified by
// opts.PromotionEventHash. Per Charter §2.5 BC5 the demotion event
// is a Cat I record committed via substrate.Append.
//
// Per §0011 Layer A is a CANDIDACY criterion, not hard barrier.
// DemoteCoordinationRing records demotion regardless of whether the
// cadence has elapsed; the report surfaces the gate state.
//
// Errors:
//   - ErrTargetNotFound: promotion hash does not resolve to any row.
//   - ErrTargetWrongType: target hash is NOT a
//     CoordinationRingPromotion.
func DemoteCoordinationRing(ctx context.Context, sub *substrate.Substrate, opts CoordinationRingDemoteOptions, now func() time.Time) (CoordinationRingDemoteReport, error) {
	if now == nil {
		now = time.Now
	}

	row, err := sub.LookupRow(ctx, opts.PromotionEventHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CoordinationRingDemoteReport{}, fmt.Errorf("%w: %x", ErrTargetNotFound, opts.PromotionEventHash)
		}
		return CoordinationRingDemoteReport{}, fmt.Errorf("hypothesis.DemoteCoordinationRing: lookup target: %w", err)
	}
	if row.MessageType != coordinationRingPromotionMessageType {
		return CoordinationRingDemoteReport{}, fmt.Errorf("%w: %x is %q", ErrTargetWrongType, opts.PromotionEventHash, row.MessageType)
	}

	payload, err := sub.ReadBlob(ctx, opts.PromotionEventHash)
	if err != nil {
		return CoordinationRingDemoteReport{}, fmt.Errorf("hypothesis.DemoteCoordinationRing: read promotion blob: %w", err)
	}
	promotion := &eventsv1.CoordinationRingPromotion{}
	if err := proto.Unmarshal(payload, promotion); err != nil {
		return CoordinationRingDemoteReport{}, fmt.Errorf("hypothesis.DemoteCoordinationRing: unmarshal promotion: %w", err)
	}

	demotedAt := opts.DemotedAt
	if demotedAt == 0 {
		demotedAt = now().UnixNano()
	}

	elapsedSeconds := (demotedAt - promotion.GetPromotedAt()) / int64(time.Second)
	cadenceSatisfied := elapsedSeconds >= promotion.GetCadenceSeconds()

	// Layer B evaluation per §0141 sub-decision E1 (advisory) + B1
	// (on-the-fly).
	var formationHash [32]byte
	copy(formationHash[:], promotion.GetFormationEventHash())
	layerBReport, err := evaluateLayerB(ctx, sub, formationHash, promotion.GetLayerBParameters())
	if err != nil {
		return CoordinationRingDemoteReport{}, fmt.Errorf("hypothesis.DemoteCoordinationRing: evaluate Layer B: %w", err)
	}

	ev := &eventsv1.CoordinationRingDemotion{
		PromotionEventHash: opts.PromotionEventHash[:],
		DemotedAt:          demotedAt,
		Reason:             opts.Reason,
	}
	demotionPayload, hash, err := canonical.MarshalAndHash(ev)
	if err != nil {
		return CoordinationRingDemoteReport{}, fmt.Errorf("hypothesis.DemoteCoordinationRing: marshal demotion: %w", err)
	}
	hex := canonical.HashHex(hash)

	_, lookupErr := sub.LookupRow(ctx, hash)
	alreadyPresent := lookupErr == nil
	if lookupErr != nil && !errors.Is(lookupErr, sql.ErrNoRows) {
		return CoordinationRingDemoteReport{}, fmt.Errorf("hypothesis.DemoteCoordinationRing: lookup demotion %s: %w", hex, lookupErr)
	}

	committedAt := now().UnixNano()
	demoRow := substrate.EventRow{
		EventHash:   hash,
		EventTime:   demotedAt,
		MessageType: string(ev.ProtoReflect().Descriptor().FullName()),
		PayloadRef:  hex[:2] + "/" + hex[2:],
		CommittedAt: committedAt,
	}

	if opts.Actor == "" {
		if err := sub.Append(ctx, demoRow, demotionPayload); err != nil {
			return CoordinationRingDemoteReport{}, fmt.Errorf("hypothesis.DemoteCoordinationRing: append demotion %s: %w", hex, err)
		}
		return CoordinationRingDemoteReport{
			DemotionEventHashHex:  hex,
			AlreadyDemoted:        alreadyPresent,
			CadenceSatisfied:      cadenceSatisfied,
			CadenceElapsedSeconds: elapsedSeconds,
			LayerB:                layerBReport,
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
		return CoordinationRingDemoteReport{}, fmt.Errorf("hypothesis.DemoteCoordinationRing: marshal ingestion event: %w", err)
	}
	ingHex := canonical.HashHex(ingHash)
	ingRow := substrate.EventRow{
		EventHash:   ingHash,
		EventTime:   committedAt,
		MessageType: string(ingEv.ProtoReflect().Descriptor().FullName()),
		PayloadRef:  ingHex[:2] + "/" + ingHex[2:],
		CommittedAt: committedAt,
	}
	if err := sub.AppendPair(ctx, demoRow, demotionPayload, ingRow, ingPayload); err != nil {
		return CoordinationRingDemoteReport{}, fmt.Errorf("hypothesis.DemoteCoordinationRing: append pair (demotion %s, ingestion %s): %w", hex, ingHex, err)
	}

	return CoordinationRingDemoteReport{
		DemotionEventHashHex:  hex,
		AlreadyDemoted:        alreadyPresent,
		CadenceSatisfied:      cadenceSatisfied,
		CadenceElapsedSeconds: elapsedSeconds,
		IngestionEventHashHex: ingHex,
		LayerB:                layerBReport,
	}, nil
}
