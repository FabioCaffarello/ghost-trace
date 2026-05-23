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

// behavioralClusterPromotionMessageType is the substrate's
// message_type discriminator for BehavioralClusterPromotion rows.
// Demote validates the target hash points to a row of this type
// (not a formation row, not an observation) before committing —
// preserves §2.5-lifecycle-integrity: demotion references only
// promotion events.
const behavioralClusterPromotionMessageType = "ghosttrace.events.v1.BehavioralClusterPromotion"

// DemoteOptions configures a single demotion. The (PromotionEventHash,
// DemotedAt, Reason) tuple uniquely identifies a demotion event's
// content-hash — re-running with identical values is idempotent.
type DemoteOptions struct {
	// PromotionEventHash is the content-hash of the
	// BehavioralClusterPromotion event being ended. Identifies WHICH
	// cadence gate is closing per §0011 (multiple promotions per
	// formation under different cadence parameters → demotion is
	// promotion-specific, not formation-specific).
	PromotionEventHash [32]byte

	// DemotedAt is the Unix-nanoseconds time at which demotion is
	// recorded. Zero defaults to now().UnixNano(). The cadence-gate
	// satisfied state is computed as (DemotedAt - promotion.promoted_at)
	// >= promotion.cadence_seconds * 1e9.
	DemotedAt int64

	// Reason is an operator-supplied free-form note. Optional;
	// strongly recommended when demoting within the cadence window
	// (Layer A unsatisfied), as it is the only record of why the
	// candidacy gate was bypassed.
	Reason string

	// Actor is an optional per-actor attribution string. Mirrors
	// PromoteOptions.Actor per §0097 BehavioralCluster pilot — extended
	// to demote at the §0107 T4 demote landing. When non-empty, Demote
	// commits the BehavioralClusterDemotion event paired with an
	// IngestionEvent via AppendPair. Empty preserves the single-Append
	// path (backward compatible).
	Actor string
}

// DemoteReport is the per-Demote outcome.
type DemoteReport struct {
	// DemotionEventHashHex is the content-hash (hex) of the recorded
	// BehavioralClusterDemotion event.
	DemotionEventHashHex string

	// AlreadyDemoted is true when an identical demotion event was
	// already in the substrate (content-hash collision).
	AlreadyDemoted bool

	// IngestionEventHashHex is the content-hash (hex) of the paired
	// IngestionEvent committed when Actor was non-empty. Empty
	// otherwise.
	IngestionEventHashHex string

	// CadenceSatisfied reports whether Layer A of the §0011 staged-
	// combination criterion had elapsed at DemotedAt. Informational —
	// per §0011 Layer A is a CANDIDACY gate, not a hard barrier.
	// Demote records the event regardless; operators and downstream
	// projections use this flag to surface early-demotion cases.
	CadenceSatisfied bool

	// CadenceElapsedSeconds reports how many seconds elapsed between
	// the source promotion's promoted_at and this demotion's
	// demoted_at. Surfaces "how early" or "how late" the demotion
	// fired relative to the cadence parameter.
	CadenceElapsedSeconds int64

	// LayerB is the Layer B deep-criterion verdict per §0141 sub-
	// decision E1 (advisory like Layer A). Per §0011 staged-combination,
	// Layer B is a CANDIDACY criterion alongside Layer A; LayerB.Fired
	// indicates whether the demote target satisfied Layer B at evaluation
	// time. Demote records the demotion regardless of LayerB.Fired.
	// LayerB.Evaluated=false when the source promotion lacks
	// LayerBParameters (pre-§0138 legacy promotion); see LayerBReport
	// fields documentation.
	LayerB LayerBReport
}

// Demote records a BehavioralClusterDemotion lifecycle event against
// the BehavioralClusterPromotion identified by opts.PromotionEventHash.
// Per Charter §2.5 BC5 the demotion event is a Category I record
// committed via substrate.Append (acquires writeMu per
// concurrency-pattern.md §Substrate-Writer Serialization).
//
// Per decision-log §0011 staged-combination criterion: Layer A
// (cadence gate) is a CANDIDACY criterion, not a hard barrier.
// Demote records the demotion regardless of whether
// promotion.promoted_at + cadence_seconds has elapsed; the resulting
// DemoteReport.CadenceSatisfied surfaces the gate state for
// operator-facing reporting. Per §0141 sub-decision E1 (advisory
// like Layer A), Layer B (the deep criterion under §0135 L-BC-OR +
// §0138 inception-phase parameters T_B=K_C=0.5, N=1000, W-count) is
// evaluated on-the-fly via internal/hypothesis/layerb.Evaluate and
// surfaced in DemoteReport.LayerB; Demote also records the demotion
// regardless of LayerB.Fired. Pre-§0138 promotions lack
// LayerBParameters and surface as DemoteReport.LayerB.Evaluated=false.
//
// Errors:
//   - ErrTargetNotFound: the promotion hash does not resolve to any
//     substrate row.
//   - ErrTargetWrongType: the target hash resolves to a row whose
//     message_type is NOT BehavioralClusterPromotion.
func Demote(ctx context.Context, sub *substrate.Substrate, opts DemoteOptions, now func() time.Time) (DemoteReport, error) {
	if now == nil {
		now = time.Now
	}

	row, err := sub.LookupRow(ctx, opts.PromotionEventHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return DemoteReport{}, fmt.Errorf("%w: %x", ErrTargetNotFound, opts.PromotionEventHash)
		}
		return DemoteReport{}, fmt.Errorf("hypothesis.Demote: lookup target: %w", err)
	}
	if row.MessageType != behavioralClusterPromotionMessageType {
		return DemoteReport{}, fmt.Errorf("%w: %x is %q", ErrTargetWrongType, opts.PromotionEventHash, row.MessageType)
	}

	// Recover the source promotion's promoted_at + cadence_seconds
	// so the cadence-gate state can be reported. The blob carries
	// the canonical Protobuf encoding of the promotion event.
	payload, err := sub.ReadBlob(ctx, opts.PromotionEventHash)
	if err != nil {
		return DemoteReport{}, fmt.Errorf("hypothesis.Demote: read promotion blob: %w", err)
	}
	promotion := &eventsv1.BehavioralClusterPromotion{}
	if err := proto.Unmarshal(payload, promotion); err != nil {
		return DemoteReport{}, fmt.Errorf("hypothesis.Demote: unmarshal promotion: %w", err)
	}

	demotedAt := opts.DemotedAt
	if demotedAt == 0 {
		demotedAt = now().UnixNano()
	}

	elapsedSeconds := (demotedAt - promotion.GetPromotedAt()) / int64(time.Second)
	cadenceSatisfied := elapsedSeconds >= promotion.GetCadenceSeconds()

	// Layer B evaluation per §0141 sub-decision E1 (advisory) + B1
	// (on-the-fly). Hypothesis identity per §0045: H.hash IS the
	// formation event's content-hash; the promotion event carries it
	// as formation_event_hash. LayerBParameters per §0138 bundling.
	var formationHash [32]byte
	copy(formationHash[:], promotion.GetFormationEventHash())
	layerBReport, err := evaluateLayerB(ctx, sub, formationHash, promotion.GetLayerBParameters())
	if err != nil {
		return DemoteReport{}, fmt.Errorf("hypothesis.Demote: evaluate Layer B: %w", err)
	}

	ev := &eventsv1.BehavioralClusterDemotion{
		PromotionEventHash: opts.PromotionEventHash[:],
		DemotedAt:          demotedAt,
		Reason:             opts.Reason,
	}
	demotionPayload, hash, err := canonical.MarshalAndHash(ev)
	if err != nil {
		return DemoteReport{}, fmt.Errorf("hypothesis.Demote: marshal demotion: %w", err)
	}
	hex := canonical.HashHex(hash)

	_, lookupErr := sub.LookupRow(ctx, hash)
	alreadyPresent := lookupErr == nil
	if lookupErr != nil && !errors.Is(lookupErr, sql.ErrNoRows) {
		return DemoteReport{}, fmt.Errorf("hypothesis.Demote: lookup demotion %s: %w", hex, lookupErr)
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
			return DemoteReport{}, fmt.Errorf("hypothesis.Demote: append demotion %s: %w", hex, err)
		}
		return DemoteReport{
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
		return DemoteReport{}, fmt.Errorf("hypothesis.Demote: marshal ingestion event: %w", err)
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
		return DemoteReport{}, fmt.Errorf("hypothesis.Demote: append pair (demotion %s, ingestion %s): %w", hex, ingHex, err)
	}

	return DemoteReport{
		DemotionEventHashHex:  hex,
		AlreadyDemoted:        alreadyPresent,
		CadenceSatisfied:      cadenceSatisfied,
		CadenceElapsedSeconds: elapsedSeconds,
		IngestionEventHashHex: ingHex,
		LayerB:                layerBReport,
	}, nil
}
