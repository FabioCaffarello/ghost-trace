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

// behavioralClusterFormationMessageType is the substrate's
// message_type discriminator for BehavioralClusterFormation rows.
// Promote validates the target hash points to a row of this type
// before committing a promotion event — promoting a non-formation
// row would record a §2.5 lifecycle event against a non-hypothesis,
// which is a structural error.
const behavioralClusterFormationMessageType = "ghosttrace.events.v1.BehavioralClusterFormation"

// ErrTargetNotFound is returned by lifecycle operations (Promote,
// Demote, ...) when the target event hash does not resolve to any
// substrate row.
var ErrTargetNotFound = errors.New("hypothesis: target event hash not found in substrate")

// ErrTargetWrongType is returned by lifecycle operations when the
// target hash resolves to a row whose message_type does not match
// the expected predecessor (Promote expects BehavioralClusterFormation;
// Demote expects BehavioralClusterPromotion). Preserves
// §2.5-lifecycle-integrity: lifecycle events reference only the
// correctly-typed predecessor.
var ErrTargetWrongType = errors.New("hypothesis: target hash is not the expected lifecycle event type")

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

	// Actor is an optional CLI-channel per-actor attribution per
	// decision-log §0097. When non-empty, Promote commits the
	// BehavioralClusterPromotion event paired with an IngestionEvent
	// (channel="cli", client_common_name=Actor) via AppendPair,
	// extending the §0038 mTLS-identity-threading discipline to
	// CLI-channel lifecycle ops. Empty preserves the §0046 single-
	// Append path (backward compatible). Per the auth-scope RFC Open
	// Question 2: CLI attribution is operator opt-in, not enforced —
	// the CLI operator is trusted by virtue of local shell access.
	Actor string

	// LayerBParameters bundles the demotion-candidacy parameters per
	// §0138 (T_B, K_C, N_window, N_A_duration_nanoseconds). When non-
	// nil, the supplied LayerBParameters is written to the promotion
	// event's layer_b_parameters field, enabling subsequent Demote
	// invocations to evaluate Layer B's deep criterion per §0141 E1.
	// When nil, the promotion event's layer_b_parameters field remains
	// unset (legacy path; subsequent Demote sets LayerB.Evaluated=false).
	//
	// Per §0141 sub-decision F3, the promote-hypothesis CLI defaults
	// the parameter values from §0138 inception-phase resolution when
	// the operator does not supply them. Callers (CLIs, programmatic
	// promoters) are responsible for choosing whether to populate this
	// field; the hypothesis package does not enforce a default.
	LayerBParameters *commonv1.LayerBParameters
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

	// IngestionEventHashHex is the content-hash (hex) of the paired
	// IngestionEvent committed when PromoteOptions.Actor was non-empty.
	// Empty when no Actor was supplied (single-Append path). Stable
	// identifier for per-actor-attribution audits.
	IngestionEventHashHex string
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
		LayerBParameters:   opts.LayerBParameters,
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

	committedAt := now().UnixNano()
	promRow := substrate.EventRow{
		EventHash:   hash,
		EventTime:   promotedAt,
		MessageType: string(ev.ProtoReflect().Descriptor().FullName()),
		PayloadRef:  hex[:2] + "/" + hex[2:],
		CommittedAt: committedAt,
	}

	// Single-Append path: no CLI per-actor attribution requested.
	// Preserves §0046 backward compatibility.
	if opts.Actor == "" {
		if err := sub.Append(ctx, promRow, payload); err != nil {
			return PromoteReport{}, fmt.Errorf("hypothesis.Promote: append promotion %s: %w", hex, err)
		}
		return PromoteReport{
			PromotionEventHashHex: hex,
			AlreadyPromoted:       alreadyPresent,
		}, nil
	}

	// AppendPair path: pair the lifecycle event with an IngestionEvent
	// carrying CLI-channel actor attribution per §0097. Reuses the
	// §0038 IngestionEvent shape — channel="cli" + client_common_name=
	// opts.Actor. The IngestionEvent's primary_event_hash references
	// the lifecycle event's content-hash (the pair-by-reference shape
	// preserved per Charter §2.1 Boundary Conditions).
	ingEv := &eventsv1.IngestionEvent{
		PrimaryEventHash: hash[:],
		ReceivedAt:       committedAt,
		IngestedAt:       committedAt,
		Channel:          "cli",
		ClientCommonName: opts.Actor,
	}
	ingPayload, ingHash, err := canonical.MarshalAndHash(ingEv)
	if err != nil {
		return PromoteReport{}, fmt.Errorf("hypothesis.Promote: marshal ingestion event: %w", err)
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
		return PromoteReport{}, fmt.Errorf("hypothesis.Promote: append pair (promotion %s, ingestion %s): %w", hex, ingHex, err)
	}

	return PromoteReport{
		PromotionEventHashHex: hex,
		AlreadyPromoted:       alreadyPresent,
		IngestionEventHashHex: ingHex,
	}, nil
}
