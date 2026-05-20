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

// behavioralClusterFormationMessageType is the substrate's
// message_type discriminator for BehavioralClusterFormation rows.
// Promote validates the target hash points to a row of this type
// before committing a promotion event — promoting a non-formation
// row would record a §2.5 lifecycle event against a non-hypothesis,
// which is a structural error.
const behavioralClusterFormationMessageType = "ghosttrace.events.v1.BehavioralClusterFormation"

// ErrTargetNotFound is returned by Promote when the formation_event_hash
// does not resolve to any substrate row.
var ErrTargetNotFound = errors.New("hypothesis.Promote: formation event hash not found in substrate")

// ErrTargetWrongType is returned by Promote when the target hash
// resolves to a row whose message_type is NOT
// BehavioralClusterFormation — preserves §2.5 lifecycle integrity
// (promotion references only formation events, never observations or
// constructs).
var ErrTargetWrongType = errors.New("hypothesis.Promote: target hash is not a BehavioralClusterFormation")

// PromoteOptions configures a single promotion. The (FormationEventHash,
// CadenceSeconds, PromotedAt, Reason) tuple uniquely identifies a
// promotion event's content-hash — re-running Promote with identical
// values is idempotent via §0027 AP6.
type PromoteOptions struct {
	// FormationEventHash is the content-hash of the
	// BehavioralClusterFormation event identifying the hypothesis
	// being promoted (per §0045 — hypothesis identity IS the
	// formation event's content-hash).
	FormationEventHash [32]byte

	// PromotedAt is the Unix-nanoseconds time from which the Layer A
	// cadence gate is measured. Zero defaults to now().UnixNano()
	// per the caller's now function. Explicit non-zero values
	// permitted for forensic replay + deterministic test recording.
	PromotedAt int64

	// CadenceSeconds is the Layer A parameter per decision-log §0011:
	// elapsed time since PromotedAt that opens demotion-candidacy.
	// Mandatory per Charter §2.5 Structural Requirement; values <= 0
	// are rejected.
	CadenceSeconds int64

	// Reason is an operator-supplied free-form note. Optional;
	// recorded for forensic replay.
	Reason string
}

// PromoteReport is the per-Promote outcome.
type PromoteReport struct {
	// PromotionEventHashHex is the content-hash (hex) of the recorded
	// BehavioralClusterPromotion event. Stable identifier for
	// subsequent demotion / merge / split references.
	PromotionEventHashHex string

	// AlreadyPromoted is true when an identical promotion event was
	// already in the substrate (content-hash collision). False when
	// this Promote invocation committed a new row.
	AlreadyPromoted bool
}

// Promote records a BehavioralClusterPromotion lifecycle event
// against the BehavioralClusterFormation identified by
// opts.FormationEventHash. Per Charter §2.5 BC5 the promotion event
// is a Category I record committed via substrate.Append (acquires
// writeMu per concurrency-pattern.md §Substrate-Writer Serialization).
//
// Errors:
//   - ErrTargetNotFound: the formation hash does not resolve to any
//     substrate row.
//   - ErrTargetWrongType: the target hash resolves to a row whose
//     message_type is NOT BehavioralClusterFormation.
//   - validation errors when opts.CadenceSeconds <= 0.
func Promote(ctx context.Context, sub *substrate.Substrate, opts PromoteOptions, now func() time.Time) (PromoteReport, error) {
	if opts.CadenceSeconds <= 0 {
		return PromoteReport{}, fmt.Errorf("hypothesis.Promote: cadence_seconds must be positive (got %d)", opts.CadenceSeconds)
	}
	if now == nil {
		now = time.Now
	}

	row, err := sub.LookupRow(ctx, opts.FormationEventHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PromoteReport{}, fmt.Errorf("%w: %x", ErrTargetNotFound, opts.FormationEventHash)
		}
		return PromoteReport{}, fmt.Errorf("hypothesis.Promote: lookup target: %w", err)
	}
	if row.MessageType != behavioralClusterFormationMessageType {
		return PromoteReport{}, fmt.Errorf("%w: %x is %q", ErrTargetWrongType, opts.FormationEventHash, row.MessageType)
	}

	promotedAt := opts.PromotedAt
	if promotedAt == 0 {
		promotedAt = now().UnixNano()
	}

	ev := &eventsv1.BehavioralClusterPromotion{
		FormationEventHash: opts.FormationEventHash[:],
		PromotedAt:         promotedAt,
		CadenceSeconds:     opts.CadenceSeconds,
		Reason:             opts.Reason,
	}
	payload, hash, err := canonical.MarshalAndHash(ev)
	if err != nil {
		return PromoteReport{}, fmt.Errorf("hypothesis.Promote: marshal promotion: %w", err)
	}
	hex := canonical.HashHex(hash)

	_, lookupErr := sub.LookupRow(ctx, hash)
	alreadyPresent := lookupErr == nil
	if lookupErr != nil && !errors.Is(lookupErr, sql.ErrNoRows) {
		return PromoteReport{}, fmt.Errorf("hypothesis.Promote: lookup promotion %s: %w", hex, lookupErr)
	}

	promRow := substrate.EventRow{
		EventHash:   hash,
		EventTime:   promotedAt,
		MessageType: string(ev.ProtoReflect().Descriptor().FullName()),
		PayloadRef:  hex[:2] + "/" + hex[2:],
		CommittedAt: now().UnixNano(),
	}
	if err := sub.Append(ctx, promRow, payload); err != nil {
		return PromoteReport{}, fmt.Errorf("hypothesis.Promote: append promotion %s: %w", hex, err)
	}

	return PromoteReport{
		PromotionEventHashHex: hex,
		AlreadyPromoted:       alreadyPresent,
	}, nil
}
