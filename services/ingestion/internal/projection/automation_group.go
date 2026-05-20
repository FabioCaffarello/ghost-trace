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

// AutomationGroupProjection is the AutomationGroup-subtype parallel
// to HypothesisProjection (per §0051's BehavioralCluster-implicit
// type). Mirrors the writer-side typed-subtype-landings discipline
// established at §0056 — each Cat III concrete subtype carries its
// own typed projection structure, rather than a generic Cat III
// surface that would lose the subtype-specific lifecycle event
// pointer types.
//
// State + LifecycleEntry are shared with BehavioralCluster
// projection per §0051: those types are subtype-agnostic (the
// State enum reports lifecycle position; LifecycleEntry stores the
// message_type as a string). Only the typed lifecycle event
// pointers differ.
type AutomationGroupProjection struct {
	// FormationHash is the BLAKE3-256 content-hash of the
	// AutomationGroupFormation that defines the hypothesis's
	// identity per §0045 + §0056.
	FormationHash [32]byte

	// State is the projection's interpretation of where the
	// hypothesis chain currently sits in the §2.5 lifecycle.
	// Computed per the same precedence rules as §0051's
	// BehavioralCluster computeState; the State enum values are
	// subtype-agnostic.
	State State

	// LatestPromotion is the most-recent AutomationGroupPromotion
	// event observed for this formation, or nil if none.
	LatestPromotion *eventsv1.AutomationGroupPromotion

	// LatestDemotion is the most-recent AutomationGroupDemotion
	// event observed that targets LatestPromotion's event_hash.
	LatestDemotion *eventsv1.AutomationGroupDemotion

	// Dissolution is the dissolution event for this formation,
	// or nil.
	Dissolution *eventsv1.AutomationGroupDissolution

	// MergedInto is the merge event in which this formation
	// appears as an antecedent.
	MergedInto *eventsv1.AutomationGroupMerge

	// SplitInto is the split event in which this formation
	// appears as the antecedent.
	SplitInto *eventsv1.AutomationGroupSplit

	// LifecycleHistory is the full chronological list (ascending
	// by event_time) of every lifecycle event that reaches this
	// formation directly.
	LifecycleHistory []LifecycleEntry

	// FormationToFirstPromotionLatencyNs mirrors §0055 for the
	// AutomationGroup subtype.
	FormationToFirstPromotionLatencyNs *int64

	// LatestPromotionToLatestDemotionLatencyNs mirrors §0055.
	LatestPromotionToLatestDemotionLatencyNs *int64

	// FormationToDissolutionLatencyNs mirrors §0055.
	FormationToDissolutionLatencyNs *int64
}

// AutomationGroup-subtype message_type discriminators. Kept as
// package-level constants per the same layering choice as §0051 —
// the projection package does not depend on the hypothesis
// (writer-side) package.
const (
	automationGroupFormationMessageType   = "ghosttrace.events.v1.AutomationGroupFormation"
	automationGroupPromotionMessageType   = "ghosttrace.events.v1.AutomationGroupPromotion"
	automationGroupDemotionMessageType    = "ghosttrace.events.v1.AutomationGroupDemotion"
	automationGroupDissolutionMessageType = "ghosttrace.events.v1.AutomationGroupDissolution"
	automationGroupMergeMessageType       = "ghosttrace.events.v1.AutomationGroupMerge"
	automationGroupSplitMessageType       = "ghosttrace.events.v1.AutomationGroupSplit"
)

// ProjectAutomationGroup returns the current-state projection of the
// AutomationGroup hypothesis identified by formationHash. Parallel
// to §0051's ProjectHypothesis for the second Cat III subtype.
// Same two-pass walk pattern, same precedence rules, same
// per-projection latency derivation.
//
// Errors:
//   - ErrFormationNotFound: formation hash does not resolve to any row.
//   - ErrTargetNotFormation: target hash resolves to a row whose
//     message_type is not AutomationGroupFormation.
func ProjectAutomationGroup(ctx context.Context, sub *substrate.Substrate, formationHash [32]byte) (AutomationGroupProjection, error) {
	formationRow, err := sub.LookupRow(ctx, formationHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return AutomationGroupProjection{}, fmt.Errorf("%w: %x", ErrFormationNotFound, formationHash)
		}
		return AutomationGroupProjection{}, fmt.Errorf("projection.ProjectAutomationGroup: lookup formation: %w", err)
	}
	if formationRow.MessageType != automationGroupFormationMessageType {
		return AutomationGroupProjection{}, fmt.Errorf("%w: %x is %q", ErrTargetNotFormation, formationHash, formationRow.MessageType)
	}

	proj := AutomationGroupProjection{
		FormationHash: formationHash,
		LifecycleHistory: []LifecycleEntry{{
			Type:      automationGroupFormationMessageType,
			EventHash: formationHash,
			EventTime: formationRow.EventTime,
		}},
	}

	promotionsByHash := map[[32]byte]*eventsv1.AutomationGroupPromotion{}

	walkErr := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != automationGroupPromotionMessageType {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return err
		}
		ev := &eventsv1.AutomationGroupPromotion{}
		if err := proto.Unmarshal(payload, ev); err != nil {
			return err
		}
		if !bytes.Equal(ev.FormationEventHash, formationHash[:]) {
			return nil
		}
		promotionsByHash[row.EventHash] = ev
		proj.LifecycleHistory = append(proj.LifecycleHistory, LifecycleEntry{
			Type:      automationGroupPromotionMessageType,
			EventHash: row.EventHash,
			EventTime: ev.PromotedAt,
		})
		if proj.LatestPromotion == nil || ev.PromotedAt > proj.LatestPromotion.PromotedAt {
			proj.LatestPromotion = ev
		}
		return nil
	})
	if walkErr != nil {
		return AutomationGroupProjection{}, fmt.Errorf("projection.ProjectAutomationGroup: pass-one walk: %w", walkErr)
	}

	walkErr = sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		switch row.MessageType {
		case automationGroupDemotionMessageType:
			payload, err := sub.ReadBlob(ctx, row.EventHash)
			if err != nil {
				return err
			}
			ev := &eventsv1.AutomationGroupDemotion{}
			if err := proto.Unmarshal(payload, ev); err != nil {
				return err
			}
			var target [32]byte
			copy(target[:], ev.PromotionEventHash)
			if _, ok := promotionsByHash[target]; !ok {
				return nil
			}
			proj.LifecycleHistory = append(proj.LifecycleHistory, LifecycleEntry{
				Type:      automationGroupDemotionMessageType,
				EventHash: row.EventHash,
				EventTime: ev.DemotedAt,
			})
			if proj.LatestPromotion != nil &&
				bytes.Equal(ev.PromotionEventHash, automationGroupPromotionHashOf(proj.LatestPromotion, promotionsByHash)) {
				if proj.LatestDemotion == nil || ev.DemotedAt > proj.LatestDemotion.DemotedAt {
					proj.LatestDemotion = ev
				}
			}

		case automationGroupDissolutionMessageType:
			payload, err := sub.ReadBlob(ctx, row.EventHash)
			if err != nil {
				return err
			}
			ev := &eventsv1.AutomationGroupDissolution{}
			if err := proto.Unmarshal(payload, ev); err != nil {
				return err
			}
			if !bytes.Equal(ev.FormationEventHash, formationHash[:]) {
				return nil
			}
			proj.LifecycleHistory = append(proj.LifecycleHistory, LifecycleEntry{
				Type:      automationGroupDissolutionMessageType,
				EventHash: row.EventHash,
				EventTime: ev.DissolvedAt,
			})
			if proj.Dissolution == nil || ev.DissolvedAt > proj.Dissolution.DissolvedAt {
				proj.Dissolution = ev
			}

		case automationGroupMergeMessageType:
			payload, err := sub.ReadBlob(ctx, row.EventHash)
			if err != nil {
				return err
			}
			ev := &eventsv1.AutomationGroupMerge{}
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
				Type:      automationGroupMergeMessageType,
				EventHash: row.EventHash,
				EventTime: ev.MergedAt,
			})
			if proj.MergedInto == nil || ev.MergedAt > proj.MergedInto.MergedAt {
				proj.MergedInto = ev
			}

		case automationGroupSplitMessageType:
			payload, err := sub.ReadBlob(ctx, row.EventHash)
			if err != nil {
				return err
			}
			ev := &eventsv1.AutomationGroupSplit{}
			if err := proto.Unmarshal(payload, ev); err != nil {
				return err
			}
			if !bytes.Equal(ev.AntecedentFormationEventHash, formationHash[:]) {
				return nil
			}
			proj.LifecycleHistory = append(proj.LifecycleHistory, LifecycleEntry{
				Type:      automationGroupSplitMessageType,
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
		return AutomationGroupProjection{}, fmt.Errorf("projection.ProjectAutomationGroup: pass-two walk: %w", walkErr)
	}

	sort.SliceStable(proj.LifecycleHistory, func(i, j int) bool {
		return proj.LifecycleHistory[i].EventTime < proj.LifecycleHistory[j].EventTime
	})

	proj.State = computeAutomationGroupState(&proj)
	computeAutomationGroupLatencies(&proj)
	return proj, nil
}

// computeAutomationGroupState applies the same precedence rules as
// §0051's computeState, just operating on AutomationGroupProjection
// fields. Precedence: Dissolved > SplitInto > MergedInto > Demoted >
// Promoted > Forming.
func computeAutomationGroupState(p *AutomationGroupProjection) State {
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

// computeAutomationGroupLatencies mirrors §0055's computeLatencies
// for AutomationGroupProjection. Pure function; idempotent.
func computeAutomationGroupLatencies(proj *AutomationGroupProjection) {
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
		if entry.Type == automationGroupPromotionMessageType {
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

// automationGroupPromotionHashOf — reverse-lookup helper mirroring
// §0051's promotionEventHashOf.
func automationGroupPromotionHashOf(promotion *eventsv1.AutomationGroupPromotion, byHash map[[32]byte]*eventsv1.AutomationGroupPromotion) []byte {
	for h, p := range byHash {
		if p == promotion {
			return h[:]
		}
	}
	return nil
}
