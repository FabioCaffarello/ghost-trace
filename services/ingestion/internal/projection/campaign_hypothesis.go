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

// CampaignHypothesisProjection is the CampaignHypothesis-subtype
// parallel to HypothesisProjection (§0051) and
// AutomationGroupProjection (§0062). Lands per decision-log §0069 —
// closing the projection-layer carry-forward opened at §0063 for the
// third (event-centric) Cat III subtype. Mirrors the typed-subtype-
// landings discipline reaffirmed across §0062: parallel per-subtype
// projection structures, NOT a generic Cat III surface.
//
// State + LifecycleEntry are shared with the BC + AG projections per
// §0051: those types are subtype-agnostic (the State enum reports
// lifecycle position; LifecycleEntry stores the message_type as a
// string). Only the typed lifecycle event pointers differ.
type CampaignHypothesisProjection struct {
	// FormationHash is the BLAKE3-256 content-hash of the
	// CampaignHypothesisFormation that defines the hypothesis's
	// identity per §0045 + §0056 + §0063.
	FormationHash [32]byte

	// State is the projection's interpretation of where the
	// hypothesis chain currently sits in the §2.5 lifecycle.
	// Computed per the same precedence rules as §0051's
	// computeState; the State enum values are subtype-agnostic.
	State State

	// LatestPromotion is the most-recent CampaignHypothesisPromotion
	// event observed for this formation, or nil if none.
	LatestPromotion *eventsv1.CampaignHypothesisPromotion

	// LatestDemotion is the most-recent CampaignHypothesisDemotion
	// event observed that targets LatestPromotion's event_hash.
	LatestDemotion *eventsv1.CampaignHypothesisDemotion

	// Dissolution is the dissolution event for this formation,
	// or nil.
	Dissolution *eventsv1.CampaignHypothesisDissolution

	// MergedInto is the merge event in which this formation
	// appears as an antecedent.
	MergedInto *eventsv1.CampaignHypothesisMerge

	// SplitInto is the split event in which this formation
	// appears as the antecedent.
	SplitInto *eventsv1.CampaignHypothesisSplit

	// LifecycleHistory is the full chronological list (ascending
	// by event_time) of every lifecycle event that reaches this
	// formation directly.
	LifecycleHistory []LifecycleEntry

	// FormationToFirstPromotionLatencyNs mirrors §0055 for the
	// CampaignHypothesis subtype.
	FormationToFirstPromotionLatencyNs *int64

	// LatestPromotionToLatestDemotionLatencyNs mirrors §0055.
	LatestPromotionToLatestDemotionLatencyNs *int64

	// FormationToDissolutionLatencyNs mirrors §0055.
	FormationToDissolutionLatencyNs *int64
}

// CampaignHypothesis-subtype message_type discriminators. Kept as
// package-level constants per the same layering choice as §0051 +
// §0062 — the projection package does not depend on the hypothesis
// (writer-side) package.
const (
	campaignHypothesisFormationMessageType   = "ghosttrace.events.v1.CampaignHypothesisFormation"
	campaignHypothesisPromotionMessageType   = "ghosttrace.events.v1.CampaignHypothesisPromotion"
	campaignHypothesisDemotionMessageType    = "ghosttrace.events.v1.CampaignHypothesisDemotion"
	campaignHypothesisDissolutionMessageType = "ghosttrace.events.v1.CampaignHypothesisDissolution"
	campaignHypothesisMergeMessageType       = "ghosttrace.events.v1.CampaignHypothesisMerge"
	campaignHypothesisSplitMessageType       = "ghosttrace.events.v1.CampaignHypothesisSplit"
)

// ProjectCampaignHypothesis returns the current-state projection of
// the CampaignHypothesis identified by formationHash. Parallel to
// §0051's ProjectHypothesis + §0062's ProjectAutomationGroup. Same
// two-pass walk pattern, same precedence rules, same per-projection
// latency derivation.
//
// Errors:
//   - ErrFormationNotFound: formation hash does not resolve to any row.
//   - ErrTargetNotFormation: target hash resolves to a row whose
//     message_type is not CampaignHypothesisFormation.
func ProjectCampaignHypothesis(ctx context.Context, sub *substrate.Substrate, formationHash [32]byte) (CampaignHypothesisProjection, error) {
	formationRow, err := sub.LookupRow(ctx, formationHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return CampaignHypothesisProjection{}, fmt.Errorf("%w: %x", ErrFormationNotFound, formationHash)
		}
		return CampaignHypothesisProjection{}, fmt.Errorf("projection.ProjectCampaignHypothesis: lookup formation: %w", err)
	}
	if formationRow.MessageType != campaignHypothesisFormationMessageType {
		return CampaignHypothesisProjection{}, fmt.Errorf("%w: %x is %q", ErrTargetNotFormation, formationHash, formationRow.MessageType)
	}

	proj := CampaignHypothesisProjection{
		FormationHash: formationHash,
		LifecycleHistory: []LifecycleEntry{{
			Type:      campaignHypothesisFormationMessageType,
			EventHash: formationHash,
			EventTime: formationRow.EventTime,
		}},
	}

	promotionsByHash := map[[32]byte]*eventsv1.CampaignHypothesisPromotion{}

	walkErr := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != campaignHypothesisPromotionMessageType {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return err
		}
		ev := &eventsv1.CampaignHypothesisPromotion{}
		if err := proto.Unmarshal(payload, ev); err != nil {
			return err
		}
		if !bytes.Equal(ev.FormationEventHash, formationHash[:]) {
			return nil
		}
		promotionsByHash[row.EventHash] = ev
		proj.LifecycleHistory = append(proj.LifecycleHistory, LifecycleEntry{
			Type:      campaignHypothesisPromotionMessageType,
			EventHash: row.EventHash,
			EventTime: ev.PromotedAt,
		})
		if proj.LatestPromotion == nil || ev.PromotedAt > proj.LatestPromotion.PromotedAt {
			proj.LatestPromotion = ev
		}
		return nil
	})
	if walkErr != nil {
		return CampaignHypothesisProjection{}, fmt.Errorf("projection.ProjectCampaignHypothesis: pass-one walk: %w", walkErr)
	}

	walkErr = sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		switch row.MessageType {
		case campaignHypothesisDemotionMessageType:
			payload, err := sub.ReadBlob(ctx, row.EventHash)
			if err != nil {
				return err
			}
			ev := &eventsv1.CampaignHypothesisDemotion{}
			if err := proto.Unmarshal(payload, ev); err != nil {
				return err
			}
			var target [32]byte
			copy(target[:], ev.PromotionEventHash)
			if _, ok := promotionsByHash[target]; !ok {
				return nil
			}
			proj.LifecycleHistory = append(proj.LifecycleHistory, LifecycleEntry{
				Type:      campaignHypothesisDemotionMessageType,
				EventHash: row.EventHash,
				EventTime: ev.DemotedAt,
			})
			if proj.LatestPromotion != nil &&
				bytes.Equal(ev.PromotionEventHash, campaignHypothesisPromotionHashOf(proj.LatestPromotion, promotionsByHash)) {
				if proj.LatestDemotion == nil || ev.DemotedAt > proj.LatestDemotion.DemotedAt {
					proj.LatestDemotion = ev
				}
			}

		case campaignHypothesisDissolutionMessageType:
			payload, err := sub.ReadBlob(ctx, row.EventHash)
			if err != nil {
				return err
			}
			ev := &eventsv1.CampaignHypothesisDissolution{}
			if err := proto.Unmarshal(payload, ev); err != nil {
				return err
			}
			if !bytes.Equal(ev.FormationEventHash, formationHash[:]) {
				return nil
			}
			proj.LifecycleHistory = append(proj.LifecycleHistory, LifecycleEntry{
				Type:      campaignHypothesisDissolutionMessageType,
				EventHash: row.EventHash,
				EventTime: ev.DissolvedAt,
			})
			if proj.Dissolution == nil || ev.DissolvedAt > proj.Dissolution.DissolvedAt {
				proj.Dissolution = ev
			}

		case campaignHypothesisMergeMessageType:
			payload, err := sub.ReadBlob(ctx, row.EventHash)
			if err != nil {
				return err
			}
			ev := &eventsv1.CampaignHypothesisMerge{}
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
				Type:      campaignHypothesisMergeMessageType,
				EventHash: row.EventHash,
				EventTime: ev.MergedAt,
			})
			if proj.MergedInto == nil || ev.MergedAt > proj.MergedInto.MergedAt {
				proj.MergedInto = ev
			}

		case campaignHypothesisSplitMessageType:
			payload, err := sub.ReadBlob(ctx, row.EventHash)
			if err != nil {
				return err
			}
			ev := &eventsv1.CampaignHypothesisSplit{}
			if err := proto.Unmarshal(payload, ev); err != nil {
				return err
			}
			if !bytes.Equal(ev.AntecedentFormationEventHash, formationHash[:]) {
				return nil
			}
			proj.LifecycleHistory = append(proj.LifecycleHistory, LifecycleEntry{
				Type:      campaignHypothesisSplitMessageType,
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
		return CampaignHypothesisProjection{}, fmt.Errorf("projection.ProjectCampaignHypothesis: pass-two walk: %w", walkErr)
	}

	sort.SliceStable(proj.LifecycleHistory, func(i, j int) bool {
		return proj.LifecycleHistory[i].EventTime < proj.LifecycleHistory[j].EventTime
	})

	proj.State = computeCampaignHypothesisState(&proj)
	computeCampaignHypothesisLatencies(&proj)
	return proj, nil
}

// computeCampaignHypothesisState applies the same precedence rules
// as §0051's computeState + §0062's computeAutomationGroupState.
// Precedence: Dissolved > SplitInto > MergedInto > Demoted >
// Promoted > Forming.
func computeCampaignHypothesisState(p *CampaignHypothesisProjection) State {
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

// computeCampaignHypothesisLatencies mirrors §0055's
// computeLatencies for CampaignHypothesisProjection. Pure function;
// idempotent.
func computeCampaignHypothesisLatencies(proj *CampaignHypothesisProjection) {
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
		if entry.Type == campaignHypothesisPromotionMessageType {
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

// campaignHypothesisPromotionHashOf — reverse-lookup helper
// mirroring §0051's promotionEventHashOf + §0062's variant.
func campaignHypothesisPromotionHashOf(promotion *eventsv1.CampaignHypothesisPromotion, byHash map[[32]byte]*eventsv1.CampaignHypothesisPromotion) []byte {
	for h, p := range byHash {
		if p == promotion {
			return h[:]
		}
	}
	return nil
}
