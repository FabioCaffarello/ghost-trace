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

// automationGroupPromotionMessageType is the substrate's message_type
// discriminator for AutomationGroupPromotion rows. DemoteAutomationGroup
// validates the target hash points to a row of this type before
// committing — preserves §2.5-lifecycle-integrity for the second
// subtype's arc (mirrors the §0047 + §0057 pattern).
const automationGroupPromotionMessageType = "ghosttrace.events.v1.AutomationGroupPromotion"

// AutomationGroupDemoteOptions configures a single AutomationGroup
// demotion. Mirrors §0047's DemoteOptions for the second-subtype
// landing.
type AutomationGroupDemoteOptions struct {
	// PromotionEventHash is the content-hash of the AutomationGroup
	// promotion being ended (per §0011 demotion is promotion-
	// specific, not formation-specific).
	PromotionEventHash [32]byte

	// DemotedAt is the Unix-nanoseconds time at which demotion is
	// recorded. Zero defaults to now().UnixNano().
	DemotedAt int64

	// Reason is an operator-supplied free-form note. Optional;
	// strongly recommended when demoting within the cadence window.
	Reason string
}

// AutomationGroupDemoteReport is the per-DemoteAutomationGroup outcome.
type AutomationGroupDemoteReport struct {
	// DemotionEventHashHex is the content-hash (hex) of the recorded
	// AutomationGroupDemotion event.
	DemotionEventHashHex string

	// AlreadyDemoted is true when an identical demotion event was
	// already in the substrate.
	AlreadyDemoted bool

	// CadenceSatisfied reports whether Layer A of the §0011 staged-
	// combination criterion had elapsed at DemotedAt. Per §0011 Layer
	// A is a CANDIDACY gate, not a hard barrier.
	CadenceSatisfied bool

	// CadenceElapsedSeconds reports how many seconds elapsed between
	// the source promotion's promoted_at and this demotion's
	// demoted_at.
	CadenceElapsedSeconds int64
}

// DemoteAutomationGroup records an AutomationGroupDemotion lifecycle
// event against the AutomationGroupPromotion identified by
// opts.PromotionEventHash. Per Charter §2.5 BC5 the demotion event
// is a Category I record committed via substrate.Append.
//
// Per §0011 staged-combination criterion: Layer A (cadence gate) is
// CANDIDACY criterion, not hard barrier. DemoteAutomationGroup
// records demotion regardless of whether the cadence has elapsed;
// the report surfaces the gate state.
//
// Errors:
//   - ErrTargetNotFound: promotion hash does not resolve to any row.
//   - ErrTargetWrongType: target hash is NOT an AutomationGroupPromotion.
func DemoteAutomationGroup(ctx context.Context, sub *substrate.Substrate, opts AutomationGroupDemoteOptions, now func() time.Time) (AutomationGroupDemoteReport, error) {
	if now == nil {
		now = time.Now
	}

	row, err := sub.LookupRow(ctx, opts.PromotionEventHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AutomationGroupDemoteReport{}, fmt.Errorf("%w: %x", ErrTargetNotFound, opts.PromotionEventHash)
		}
		return AutomationGroupDemoteReport{}, fmt.Errorf("hypothesis.DemoteAutomationGroup: lookup target: %w", err)
	}
	if row.MessageType != automationGroupPromotionMessageType {
		return AutomationGroupDemoteReport{}, fmt.Errorf("%w: %x is %q", ErrTargetWrongType, opts.PromotionEventHash, row.MessageType)
	}

	// Recover source promotion's promoted_at + cadence_seconds.
	payload, err := sub.ReadBlob(ctx, opts.PromotionEventHash)
	if err != nil {
		return AutomationGroupDemoteReport{}, fmt.Errorf("hypothesis.DemoteAutomationGroup: read promotion blob: %w", err)
	}
	promotion := &eventsv1.AutomationGroupPromotion{}
	if err := proto.Unmarshal(payload, promotion); err != nil {
		return AutomationGroupDemoteReport{}, fmt.Errorf("hypothesis.DemoteAutomationGroup: unmarshal promotion: %w", err)
	}

	demotedAt := opts.DemotedAt
	if demotedAt == 0 {
		demotedAt = now().UnixNano()
	}

	elapsedSeconds := (demotedAt - promotion.GetPromotedAt()) / int64(time.Second)
	cadenceSatisfied := elapsedSeconds >= promotion.GetCadenceSeconds()

	ev := &eventsv1.AutomationGroupDemotion{
		PromotionEventHash: opts.PromotionEventHash[:],
		DemotedAt:          demotedAt,
		Reason:             opts.Reason,
	}
	demotionPayload, hash, err := canonical.MarshalAndHash(ev)
	if err != nil {
		return AutomationGroupDemoteReport{}, fmt.Errorf("hypothesis.DemoteAutomationGroup: marshal demotion: %w", err)
	}
	hex := canonical.HashHex(hash)

	_, lookupErr := sub.LookupRow(ctx, hash)
	alreadyPresent := lookupErr == nil
	if lookupErr != nil && !errors.Is(lookupErr, sql.ErrNoRows) {
		return AutomationGroupDemoteReport{}, fmt.Errorf("hypothesis.DemoteAutomationGroup: lookup demotion %s: %w", hex, lookupErr)
	}

	demoRow := substrate.EventRow{
		EventHash:   hash,
		EventTime:   demotedAt,
		MessageType: string(ev.ProtoReflect().Descriptor().FullName()),
		PayloadRef:  hex[:2] + "/" + hex[2:],
		CommittedAt: now().UnixNano(),
	}
	if err := sub.Append(ctx, demoRow, demotionPayload); err != nil {
		return AutomationGroupDemoteReport{}, fmt.Errorf("hypothesis.DemoteAutomationGroup: append demotion %s: %w", hex, err)
	}

	return AutomationGroupDemoteReport{
		DemotionEventHashHex:  hex,
		AlreadyDemoted:        alreadyPresent,
		CadenceSatisfied:      cadenceSatisfied,
		CadenceElapsedSeconds: elapsedSeconds,
	}, nil
}
