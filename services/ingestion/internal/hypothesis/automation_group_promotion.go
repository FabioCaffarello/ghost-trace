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

// automationGroupFormationMessageType is the substrate's
// message_type discriminator for AutomationGroupFormation rows.
// PromoteAutomationGroup validates the target hash points to a row
// of this type before committing a promotion event — promoting a
// non-formation row would record a §2.5 lifecycle event against a
// non-hypothesis, a structural error.
const automationGroupFormationMessageType = "ghosttrace.events.v1.AutomationGroupFormation"

// AutomationGroupPromoteOptions configures a single
// AutomationGroup promotion. Mirrors §0046's PromoteOptions for the
// second-subtype landing.
type AutomationGroupPromoteOptions struct {
	// FormationEventHash is the content-hash of the originating
	// AutomationGroupFormation event identifying the hypothesis
	// being promoted (per §0045 + §0056 — hypothesis identity IS
	// the formation event's content-hash, invariant transferring
	// across subtypes).
	FormationEventHash [32]byte

	// PromotedAt is the Unix-nanoseconds time from which the Layer A
	// cadence gate is measured (per §0011). Zero defaults to
	// now().UnixNano().
	PromotedAt int64

	// CadenceSeconds is the Layer A parameter (per §0011);
	// mandatory per Charter §2.5 Structural Requirement. Values <= 0
	// are rejected.
	CadenceSeconds int64

	// Reason is an operator-supplied free-form note. Optional.
	Reason string
}

// AutomationGroupPromoteReport is the per-PromoteAutomationGroup
// outcome.
type AutomationGroupPromoteReport struct {
	// PromotionEventHashHex is the content-hash (hex) of the
	// recorded AutomationGroupPromotion event.
	PromotionEventHashHex string

	// AlreadyPromoted is true when an identical promotion event was
	// already in the substrate (content-hash collision).
	AlreadyPromoted bool
}

// PromoteAutomationGroup records an AutomationGroupPromotion
// lifecycle event against the AutomationGroupFormation identified
// by opts.FormationEventHash. Per Charter §2.5 BC5 the promotion
// event is a Category I record committed via substrate.Append.
//
// Errors:
//   - ErrTargetNotFound: the formation hash does not resolve to any
//     substrate row.
//   - ErrTargetWrongType: the target hash resolves to a row whose
//     message_type is NOT AutomationGroupFormation.
//   - validation errors when opts.CadenceSeconds <= 0.
//
// Sentinels are SHARED with the BehavioralCluster lifecycle
// operators per §0046+§0047 generalization — the operation-agnostic
// message form accommodates both subtypes; the wrong-type check
// uses the subtype-specific message_type constant
// (automationGroupFormationMessageType) so a BehavioralClusterFormation
// hash passed here will correctly trigger ErrTargetWrongType.
func PromoteAutomationGroup(ctx context.Context, sub *substrate.Substrate, opts AutomationGroupPromoteOptions, now func() time.Time) (AutomationGroupPromoteReport, error) {
	if opts.CadenceSeconds <= 0 {
		return AutomationGroupPromoteReport{}, fmt.Errorf("hypothesis.PromoteAutomationGroup: cadence_seconds must be positive (got %d)", opts.CadenceSeconds)
	}
	if now == nil {
		now = time.Now
	}

	row, err := sub.LookupRow(ctx, opts.FormationEventHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AutomationGroupPromoteReport{}, fmt.Errorf("%w: %x", ErrTargetNotFound, opts.FormationEventHash)
		}
		return AutomationGroupPromoteReport{}, fmt.Errorf("hypothesis.PromoteAutomationGroup: lookup target: %w", err)
	}
	if row.MessageType != automationGroupFormationMessageType {
		return AutomationGroupPromoteReport{}, fmt.Errorf("%w: %x is %q", ErrTargetWrongType, opts.FormationEventHash, row.MessageType)
	}

	promotedAt := opts.PromotedAt
	if promotedAt == 0 {
		promotedAt = now().UnixNano()
	}

	ev := &eventsv1.AutomationGroupPromotion{
		FormationEventHash: opts.FormationEventHash[:],
		PromotedAt:         promotedAt,
		CadenceSeconds:     opts.CadenceSeconds,
		Reason:             opts.Reason,
	}
	payload, hash, err := canonical.MarshalAndHash(ev)
	if err != nil {
		return AutomationGroupPromoteReport{}, fmt.Errorf("hypothesis.PromoteAutomationGroup: marshal promotion: %w", err)
	}
	hex := canonical.HashHex(hash)

	_, lookupErr := sub.LookupRow(ctx, hash)
	alreadyPresent := lookupErr == nil
	if lookupErr != nil && !errors.Is(lookupErr, sql.ErrNoRows) {
		return AutomationGroupPromoteReport{}, fmt.Errorf("hypothesis.PromoteAutomationGroup: lookup promotion %s: %w", hex, lookupErr)
	}

	promRow := substrate.EventRow{
		EventHash:   hash,
		EventTime:   promotedAt,
		MessageType: string(ev.ProtoReflect().Descriptor().FullName()),
		PayloadRef:  hex[:2] + "/" + hex[2:],
		CommittedAt: now().UnixNano(),
	}
	if err := sub.Append(ctx, promRow, payload); err != nil {
		return AutomationGroupPromoteReport{}, fmt.Errorf("hypothesis.PromoteAutomationGroup: append promotion %s: %w", hex, err)
	}

	return AutomationGroupPromoteReport{
		PromotionEventHashHex: hex,
		AlreadyPromoted:       alreadyPresent,
	}, nil
}
