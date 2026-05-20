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

// AutomationGroupDissolveOptions configures a single AutomationGroup
// dissolution. Mirrors §0048's DissolveOptions for the second-subtype
// landing.
type AutomationGroupDissolveOptions struct {
	// FormationEventHash is the content-hash of the
	// AutomationGroupFormation event identifying the hypothesis
	// being dissolved (per §0045 + §0056 — hypothesis identity IS
	// the formation event's content-hash). Dissolution targets the
	// formation directly; no two-hop indirection through promotion.
	FormationEventHash [32]byte

	// DissolvedAt is the Unix-nanoseconds time at which dissolution
	// is recorded. Zero defaults to now().UnixNano().
	DissolvedAt int64

	// Reason is an operator-supplied free-form note. Optional;
	// strongly recommended — dissolution is the terminal lifecycle
	// operation on a hypothesis.
	Reason string
}

// AutomationGroupDissolveReport is the per-DissolveAutomationGroup
// outcome.
type AutomationGroupDissolveReport struct {
	// DissolutionEventHashHex is the content-hash (hex) of the
	// recorded AutomationGroupDissolution event.
	DissolutionEventHashHex string

	// AlreadyDissolved is true when an identical dissolution event
	// was already in the substrate (content-hash collision).
	AlreadyDissolved bool
}

// DissolveAutomationGroup records an AutomationGroupDissolution
// lifecycle event against the AutomationGroupFormation identified
// by opts.FormationEventHash. Per Charter §2.5 BC5 the dissolution
// event is a Category I record committed via substrate.Append.
//
// Per glossary + lifecycle-semantics.md line 36, dissolution is
// DISTINGUISHED from demotion: demotion withdraws OPERATIONAL USE;
// dissolution recognizes NON-EXISTENCE of the underlying phenomenon.
// The distinction transfers across subtypes. DissolveAutomationGroup
// does not require that the hypothesis was ever promoted; a
// formation may be dissolved directly.
//
// Errors:
//   - ErrTargetNotFound: formation hash does not resolve to any row.
//   - ErrTargetWrongType: target hash is NOT an AutomationGroupFormation.
func DissolveAutomationGroup(ctx context.Context, sub *substrate.Substrate, opts AutomationGroupDissolveOptions, now func() time.Time) (AutomationGroupDissolveReport, error) {
	if now == nil {
		now = time.Now
	}

	row, err := sub.LookupRow(ctx, opts.FormationEventHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AutomationGroupDissolveReport{}, fmt.Errorf("%w: %x", ErrTargetNotFound, opts.FormationEventHash)
		}
		return AutomationGroupDissolveReport{}, fmt.Errorf("hypothesis.DissolveAutomationGroup: lookup target: %w", err)
	}
	if row.MessageType != automationGroupFormationMessageType {
		return AutomationGroupDissolveReport{}, fmt.Errorf("%w: %x is %q", ErrTargetWrongType, opts.FormationEventHash, row.MessageType)
	}

	dissolvedAt := opts.DissolvedAt
	if dissolvedAt == 0 {
		dissolvedAt = now().UnixNano()
	}

	ev := &eventsv1.AutomationGroupDissolution{
		FormationEventHash: opts.FormationEventHash[:],
		DissolvedAt:        dissolvedAt,
		Reason:             opts.Reason,
	}
	payload, hash, err := canonical.MarshalAndHash(ev)
	if err != nil {
		return AutomationGroupDissolveReport{}, fmt.Errorf("hypothesis.DissolveAutomationGroup: marshal dissolution: %w", err)
	}
	hex := canonical.HashHex(hash)

	_, lookupErr := sub.LookupRow(ctx, hash)
	alreadyPresent := lookupErr == nil
	if lookupErr != nil && !errors.Is(lookupErr, sql.ErrNoRows) {
		return AutomationGroupDissolveReport{}, fmt.Errorf("hypothesis.DissolveAutomationGroup: lookup dissolution %s: %w", hex, lookupErr)
	}

	dissRow := substrate.EventRow{
		EventHash:   hash,
		EventTime:   dissolvedAt,
		MessageType: string(ev.ProtoReflect().Descriptor().FullName()),
		PayloadRef:  hex[:2] + "/" + hex[2:],
		CommittedAt: now().UnixNano(),
	}
	if err := sub.Append(ctx, dissRow, payload); err != nil {
		return AutomationGroupDissolveReport{}, fmt.Errorf("hypothesis.DissolveAutomationGroup: append dissolution %s: %w", hex, err)
	}

	return AutomationGroupDissolveReport{
		DissolutionEventHashHex: hex,
		AlreadyDissolved:        alreadyPresent,
	}, nil
}
