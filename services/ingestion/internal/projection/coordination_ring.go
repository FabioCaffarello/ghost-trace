package projection

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"

	"google.golang.org/protobuf/proto"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// CoordinationRingProjection is the CoordinationRing-subtype
// parallel to HypothesisProjection (§0051), AutomationGroupProjection
// (§0062), and CampaignHypothesisProjection (§0069). Lands per
// decision-log §0076 — closes the projection-layer carry-forward
// opened at §0070 for the fourth (interaction-centric) Cat III
// subtype. With this entry, the projection layer covers all four
// §0010 Q2-resolved Cat III subtypes.
//
// State + LifecycleEntry are shared per §0051: subtype-agnostic
// (State enum is lifecycle-position; LifecycleEntry stores the
// message_type as a string). Only the typed lifecycle event
// pointers differ.
type CoordinationRingProjection struct {
	// FormationHash is the BLAKE3-256 content-hash of the
	// CoordinationRingFormation that defines the hypothesis's
	// identity per §0045 + §0056 + §0063 + §0070.
	FormationHash [32]byte

	// State is the projection's interpretation of where the
	// hypothesis chain currently sits in the §2.5 lifecycle.
	State State

	// LatestPromotion is the most-recent CoordinationRingPromotion
	// event observed for this formation, or nil if none.
	LatestPromotion *eventsv1.CoordinationRingPromotion

	// LatestDemotion is the most-recent CoordinationRingDemotion
	// event observed that targets LatestPromotion's event_hash.
	LatestDemotion *eventsv1.CoordinationRingDemotion

	// Dissolution is the dissolution event for this formation, or
	// nil.
	Dissolution *eventsv1.CoordinationRingDissolution

	// MergedInto is the merge event in which this formation
	// appears as an antecedent.
	MergedInto *eventsv1.CoordinationRingMerge

	// SplitInto is the split event in which this formation appears
	// as the antecedent.
	SplitInto *eventsv1.CoordinationRingSplit

	// LifecycleHistory is the full chronological list (ascending
	// by event_time) of every lifecycle event that reaches this
	// formation directly.
	LifecycleHistory []LifecycleEntry

	// FormationToFirstPromotionLatencyNs mirrors §0055 for the
	// CoordinationRing subtype.
	FormationToFirstPromotionLatencyNs *int64

	// LatestPromotionToLatestDemotionLatencyNs mirrors §0055.
	LatestPromotionToLatestDemotionLatencyNs *int64

	// FormationToDissolutionLatencyNs mirrors §0055.
	FormationToDissolutionLatencyNs *int64
}

// CoordinationRing-subtype message_type discriminators. Kept as
// package-level constants per the same layering choice as §0051 +
// §0062 + §0069.
const (
	coordinationRingFormationMessageType   = "ghosttrace.events.v1.CoordinationRingFormation"
	coordinationRingPromotionMessageType   = "ghosttrace.events.v1.CoordinationRingPromotion"
	coordinationRingDemotionMessageType    = "ghosttrace.events.v1.CoordinationRingDemotion"
	coordinationRingDissolutionMessageType = "ghosttrace.events.v1.CoordinationRingDissolution"
	coordinationRingMergeMessageType       = "ghosttrace.events.v1.CoordinationRingMerge"
	coordinationRingSplitMessageType       = "ghosttrace.events.v1.CoordinationRingSplit"
)

// ProjectCoordinationRing returns the current-state projection of
// the CoordinationRing identified by formationHash. Parallel to
// §0051's ProjectHypothesis + §0062's ProjectAutomationGroup +
// §0069's ProjectCampaignHypothesis. Same two-pass walk pattern,
// same precedence rules, same per-projection latency derivation.
//
// Errors:
//   - ErrFormationNotFound: formation hash does not resolve to any row.
//   - ErrTargetNotFormation: target hash resolves to a row whose
//     message_type is not CoordinationRingFormation.
func ProjectCoordinationRing(ctx context.Context, sub *substrate.Substrate, formationHash [32]byte) (CoordinationRingProjection, error) {
	formationRow, err := sub.LookupRow(ctx, formationHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CoordinationRingProjection{}, fmt.Errorf("%w: %x", ErrFormationNotFound, formationHash)
		}
		return CoordinationRingProjection{}, fmt.Errorf("projection.ProjectCoordinationRing: lookup formation: %w", err)
	}
	if formationRow.MessageType != coordinationRingFormationMessageType {
		return CoordinationRingProjection{}, fmt.Errorf("%w: %x is %q", ErrTargetNotFormation, formationHash, formationRow.MessageType)
	}

	proj := CoordinationRingProjection{
		FormationHash: formationHash,
		LifecycleHistory: []LifecycleEntry{{
			Type:      coordinationRingFormationMessageType,
			EventHash: formationHash,
			EventTime: formationRow.EventTime,
		}},
	}

	promotionsByHash := map[[32]byte]*eventsv1.CoordinationRingPromotion{}

	walkErr := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != coordinationRingPromotionMessageType {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return err
		}
		ev := &eventsv1.CoordinationRingPromotion{}
		if err := proto.Unmarshal(payload, ev); err != nil {
			return err
		}
		if !bytes.Equal(ev.FormationEventHash, formationHash[:]) {
			return nil
		}
		promotionsByHash[row.EventHash] = ev
		proj.LifecycleHistory = append(proj.LifecycleHistory, LifecycleEntry{
			Type:      coordinationRingPromotionMessageType,
			EventHash: row.EventHash,
			EventTime: ev.PromotedAt,
		})
		if proj.LatestPromotion == nil || ev.PromotedAt > proj.LatestPromotion.PromotedAt {
			proj.LatestPromotion = ev
		}
		return nil
	})
	if walkErr != nil {
		return CoordinationRingProjection{}, fmt.Errorf("projection.ProjectCoordinationRing: pass-one walk: %w", walkErr)
	}

	walkErr = sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		switch row.MessageType {
		case coordinationRingDemotionMessageType:
			payload, err := sub.ReadBlob(ctx, row.EventHash)
			if err != nil {
				return err
			}
			ev := &eventsv1.CoordinationRingDemotion{}
			if err := proto.Unmarshal(payload, ev); err != nil {
				return err
			}
			var target [32]byte
			copy(target[:], ev.PromotionEventHash)
			if _, ok := promotionsByHash[target]; !ok {
				return nil
			}
			proj.LifecycleHistory = append(proj.LifecycleHistory, LifecycleEntry{
				Type:      coordinationRingDemotionMessageType,
				EventHash: row.EventHash,
				EventTime: ev.DemotedAt,
			})
			if proj.LatestPromotion != nil &&
				bytes.Equal(ev.PromotionEventHash, coordinationRingPromotionHashOf(proj.LatestPromotion, promotionsByHash)) {
				if proj.LatestDemotion == nil || ev.DemotedAt > proj.LatestDemotion.DemotedAt {
					proj.LatestDemotion = ev
				}
			}

		case coordinationRingDissolutionMessageType:
			payload, err := sub.ReadBlob(ctx, row.EventHash)
			if err != nil {
				return err
			}
			ev := &eventsv1.CoordinationRingDissolution{}
			if err := proto.Unmarshal(payload, ev); err != nil {
				return err
			}
			if !bytes.Equal(ev.FormationEventHash, formationHash[:]) {
				return nil
			}
			proj.LifecycleHistory = append(proj.LifecycleHistory, LifecycleEntry{
				Type:      coordinationRingDissolutionMessageType,
				EventHash: row.EventHash,
				EventTime: ev.DissolvedAt,
			})
			if proj.Dissolution == nil || ev.DissolvedAt > proj.Dissolution.DissolvedAt {
				proj.Dissolution = ev
			}

		case coordinationRingMergeMessageType:
			payload, err := sub.ReadBlob(ctx, row.EventHash)
			if err != nil {
				return err
			}
			ev := &eventsv1.CoordinationRingMerge{}
			if err := proto.Unmarshal(payload, ev); err != nil {
				return err
			}
			matched := false
			for _, ant := range ev.AntecedentFormationEventHashes {
				if bytes.Equal(ant, formationHash[:]) {
					matched = true
					break
				}
			}
			if !matched {
				return nil
			}
			proj.LifecycleHistory = append(proj.LifecycleHistory, LifecycleEntry{
				Type:      coordinationRingMergeMessageType,
				EventHash: row.EventHash,
				EventTime: ev.MergedAt,
			})
			if proj.MergedInto == nil || ev.MergedAt > proj.MergedInto.MergedAt {
				proj.MergedInto = ev
			}

		case coordinationRingSplitMessageType:
			payload, err := sub.ReadBlob(ctx, row.EventHash)
			if err != nil {
				return err
			}
			ev := &eventsv1.CoordinationRingSplit{}
			if err := proto.Unmarshal(payload, ev); err != nil {
				return err
			}
			if !bytes.Equal(ev.AntecedentFormationEventHash, formationHash[:]) {
				return nil
			}
			proj.LifecycleHistory = append(proj.LifecycleHistory, LifecycleEntry{
				Type:      coordinationRingSplitMessageType,
				EventHash: row.EventHash,
				EventTime: ev.SplitAt,
			})
			if proj.SplitInto == nil || ev.SplitAt > proj.SplitInto.SplitAt {
				proj.SplitInto = ev
			}
		}
		return nil
	})
	if walkErr != nil {
		return CoordinationRingProjection{}, fmt.Errorf("projection.ProjectCoordinationRing: pass-two walk: %w", walkErr)
	}

	sort.SliceStable(proj.LifecycleHistory, func(i, j int) bool {
		return proj.LifecycleHistory[i].EventTime < proj.LifecycleHistory[j].EventTime
	})

	proj.State = computeCoordinationRingState(&proj)
	computeCoordinationRingLatencies(&proj)
	return proj, nil
}

// computeCoordinationRingState applies the same precedence rules
// as §0051+§0062+§0069: Dissolved > SplitInto > MergedInto >
// Demoted > Promoted > Forming.
func computeCoordinationRingState(p *CoordinationRingProjection) State {
	if p.Dissolution != nil {
		return StateDissolved
	}
	if p.SplitInto != nil {
		return StateSplitInto
	}
	if p.MergedInto != nil {
		return StateMergedInto
	}
	if p.LatestPromotion != nil {
		if p.LatestDemotion != nil {
			return StateDemoted
		}
		return StatePromoted
	}
	return StateForming
}

// computeCoordinationRingLatencies mirrors §0055's computeLatencies
// for CoordinationRingProjection. Pure function; idempotent.
func computeCoordinationRingLatencies(proj *CoordinationRingProjection) {
	if len(proj.LifecycleHistory) == 0 {
		return
	}
	formationEventTime := proj.LifecycleHistory[0].EventTime
	for _, entry := range proj.LifecycleHistory {
		if entry.EventHash == proj.FormationHash {
			formationEventTime = entry.EventTime
			break
		}
	}

	for _, entry := range proj.LifecycleHistory {
		if entry.Type == coordinationRingPromotionMessageType {
			latency := entry.EventTime - formationEventTime
			proj.FormationToFirstPromotionLatencyNs = &latency
			break
		}
	}

	if proj.LatestPromotion != nil && proj.LatestDemotion != nil {
		latency := proj.LatestDemotion.DemotedAt - proj.LatestPromotion.PromotedAt
		proj.LatestPromotionToLatestDemotionLatencyNs = &latency
	}

	if proj.Dissolution != nil {
		latency := proj.Dissolution.DissolvedAt - formationEventTime
		proj.FormationToDissolutionLatencyNs = &latency
	}
}

// coordinationRingPromotionHashOf — reverse-lookup helper mirroring
// §0051 + §0062 + §0069.
func coordinationRingPromotionHashOf(promotion *eventsv1.CoordinationRingPromotion, byHash map[[32]byte]*eventsv1.CoordinationRingPromotion) []byte {
	for h, p := range byHash {
		if p == promotion {
			return h[:]
		}
	}
	return nil
}
