package projection

import (
	"bytes"
	"context"
	"fmt"
	"sort"

	"google.golang.org/protobuf/proto"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// AutomationGroupListOptions configures filtering + paging behavior
// of ListAutomationGroups + CountAutomationGroupsByState. Parallel
// to §0052's ListOptions for the second Cat III subtype. Same
// semantic: empty StateFilter / zero TimeAfterNs+TimeBeforeNs /
// zero Limit+Offset all disable the corresponding filter.
type AutomationGroupListOptions struct {
	StateFilter  State
	TimeAfterNs  int64
	TimeBeforeNs int64
	Limit        int
	Offset       int
}

// withinAutomationGroupTimeWindow mirrors §0054's withinTimeWindow
// for AutomationGroupProjection.
func withinAutomationGroupTimeWindow(proj AutomationGroupProjection, opts AutomationGroupListOptions) bool {
	if opts.TimeAfterNs == 0 && opts.TimeBeforeNs == 0 {
		return true
	}
	if len(proj.LifecycleHistory) == 0 {
		return false
	}
	latest := proj.LifecycleHistory[len(proj.LifecycleHistory)-1].EventTime
	if opts.TimeAfterNs != 0 && latest < opts.TimeAfterNs {
		return false
	}
	if opts.TimeBeforeNs != 0 && latest > opts.TimeBeforeNs {
		return false
	}
	return true
}

// ProjectAllAutomationGroups returns the AutomationGroupProjection
// for EVERY AutomationGroupFormation in the substrate. Two-pass
// walk parallel to §0052's ProjectAll.
func ProjectAllAutomationGroups(ctx context.Context, sub *substrate.Substrate) (map[[32]byte]AutomationGroupProjection, error) {
	out := map[[32]byte]AutomationGroupProjection{}
	promotionToFormation := map[[32]byte][32]byte{}

	pass1 := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		switch row.MessageType {
		case automationGroupFormationMessageType:
			proj, ok := out[row.EventHash]
			if !ok {
				proj = AutomationGroupProjection{FormationHash: row.EventHash}
			}
			proj.LifecycleHistory = append(proj.LifecycleHistory, LifecycleEntry{
				Type:      automationGroupFormationMessageType,
				EventHash: row.EventHash,
				EventTime: row.EventTime,
			})
			out[row.EventHash] = proj

		case automationGroupPromotionMessageType:
			payload, err := sub.ReadBlob(ctx, row.EventHash)
			if err != nil {
				return err
			}
			ev := &eventsv1.AutomationGroupPromotion{}
			if err := proto.Unmarshal(payload, ev); err != nil {
				return err
			}
			var formationHash [32]byte
			copy(formationHash[:], ev.FormationEventHash)
			promotionToFormation[row.EventHash] = formationHash

			proj, ok := out[formationHash]
			if !ok {
				proj = AutomationGroupProjection{FormationHash: formationHash}
			}
			proj.LifecycleHistory = append(proj.LifecycleHistory, LifecycleEntry{
				Type:      automationGroupPromotionMessageType,
				EventHash: row.EventHash,
				EventTime: ev.PromotedAt,
			})
			if proj.LatestPromotion == nil || ev.PromotedAt > proj.LatestPromotion.PromotedAt {
				proj.LatestPromotion = ev
			}
			out[formationHash] = proj
		}
		return nil
	})
	if pass1 != nil {
		return nil, fmt.Errorf("projection.ProjectAllAutomationGroups: pass-one walk: %w", pass1)
	}

	pass2 := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
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
			var promotionHash [32]byte
			copy(promotionHash[:], ev.PromotionEventHash)
			formationHash, ok := promotionToFormation[promotionHash]
			if !ok {
				return nil
			}
			proj, ok := out[formationHash]
			if !ok {
				return nil
			}
			proj.LifecycleHistory = append(proj.LifecycleHistory, LifecycleEntry{
				Type:      automationGroupDemotionMessageType,
				EventHash: row.EventHash,
				EventTime: ev.DemotedAt,
			})
			if proj.LatestPromotion != nil &&
				bytes.Equal(ev.PromotionEventHash, automationGroupPromotionHashOfMap(proj.LatestPromotion, promotionToFormation)) {
				if proj.LatestDemotion == nil || ev.DemotedAt > proj.LatestDemotion.DemotedAt {
					proj.LatestDemotion = ev
				}
			}
			out[formationHash] = proj

		case automationGroupDissolutionMessageType:
			payload, err := sub.ReadBlob(ctx, row.EventHash)
			if err != nil {
				return err
			}
			ev := &eventsv1.AutomationGroupDissolution{}
			if err := proto.Unmarshal(payload, ev); err != nil {
				return err
			}
			var formationHash [32]byte
			copy(formationHash[:], ev.FormationEventHash)
			proj, ok := out[formationHash]
			if !ok {
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
			out[formationHash] = proj

		case automationGroupMergeMessageType:
			payload, err := sub.ReadBlob(ctx, row.EventHash)
			if err != nil {
				return err
			}
			ev := &eventsv1.AutomationGroupMerge{}
			if err := proto.Unmarshal(payload, ev); err != nil {
				return err
			}
			for _, antBytes := range ev.AntecedentFormationEventHashes {
				var antHash [32]byte
				copy(antHash[:], antBytes)
				proj, ok := out[antHash]
				if !ok {
					continue
				}
				proj.LifecycleHistory = append(proj.LifecycleHistory, LifecycleEntry{
					Type:      automationGroupMergeMessageType,
					EventHash: row.EventHash,
					EventTime: ev.MergedAt,
				})
				if proj.MergedInto == nil || ev.MergedAt > proj.MergedInto.MergedAt {
					proj.MergedInto = ev
				}
				out[antHash] = proj
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
			var antHash [32]byte
			copy(antHash[:], ev.AntecedentFormationEventHash)
			proj, ok := out[antHash]
			if !ok {
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
			out[antHash] = proj
		}
		return nil
	})
	if pass2 != nil {
		return nil, fmt.Errorf("projection.ProjectAllAutomationGroups: pass-two walk: %w", pass2)
	}

	for hash, proj := range out {
		sort.SliceStable(proj.LifecycleHistory, func(i, j int) bool {
			return proj.LifecycleHistory[i].EventTime < proj.LifecycleHistory[j].EventTime
		})
		proj.State = computeAutomationGroupState(&proj)
		computeAutomationGroupLatencies(&proj)
		out[hash] = proj
	}

	return out, nil
}

// automationGroupPromotionHashOfMap — reverse-lookup over the
// promotion-to-formation index used in ProjectAllAutomationGroups.
func automationGroupPromotionHashOfMap(promotion *eventsv1.AutomationGroupPromotion, byPromotionHash map[[32]byte][32]byte) []byte {
	for promHash, formHash := range byPromotionHash {
		if bytes.Equal(promotion.FormationEventHash, formHash[:]) {
			return promHash[:]
		}
	}
	return nil
}

// ListAutomationGroups returns a deterministically-ordered list of
// AutomationGroupProjection over every formation, filtered by state
// + time-window + paged. Parallel to §0052's ListHypotheses.
func ListAutomationGroups(ctx context.Context, sub *substrate.Substrate, opts AutomationGroupListOptions) ([]AutomationGroupProjection, error) {
	all, err := ProjectAllAutomationGroups(ctx, sub)
	if err != nil {
		return nil, err
	}

	var filtered []AutomationGroupProjection
	for _, proj := range all {
		if opts.StateFilter != "" && proj.State != opts.StateFilter {
			continue
		}
		if !withinAutomationGroupTimeWindow(proj, opts) {
			continue
		}
		filtered = append(filtered, proj)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return bytes.Compare(filtered[i].FormationHash[:], filtered[j].FormationHash[:]) < 0
	})

	if opts.Offset > 0 {
		if opts.Offset >= len(filtered) {
			return nil, nil
		}
		filtered = filtered[opts.Offset:]
	}
	if opts.Limit > 0 && len(filtered) > opts.Limit {
		filtered = filtered[:opts.Limit]
	}

	return filtered, nil
}

// CountAutomationGroupsByState returns the aggregate StateCounts
// over every AutomationGroupFormation in the substrate. Parallel
// to §0053's CountByState. Honors TimeAfterNs + TimeBeforeNs from
// opts; ignores StateFilter / Limit / Offset (counts reflect the
// full filtered population).
func CountAutomationGroupsByState(ctx context.Context, sub *substrate.Substrate, opts AutomationGroupListOptions) (StateCounts, error) {
	all, err := ProjectAllAutomationGroups(ctx, sub)
	if err != nil {
		return StateCounts{}, err
	}
	counts := StateCounts{
		ByState: map[State]int{
			StateForming:    0,
			StatePromoted:   0,
			StateDemoted:    0,
			StateDissolved:  0,
			StateMergedInto: 0,
			StateSplitInto:  0,
		},
	}
	for _, proj := range all {
		if !withinAutomationGroupTimeWindow(proj, opts) {
			continue
		}
		counts.Total++
		counts.ByState[proj.State]++
	}
	return counts, nil
}
