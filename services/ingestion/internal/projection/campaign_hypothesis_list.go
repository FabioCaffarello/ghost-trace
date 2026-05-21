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

// CampaignHypothesisListOptions configures filtering + paging
// behavior of ListCampaignHypotheses +
// CountCampaignHypothesesByState. Parallel to §0052's ListOptions +
// §0062's AutomationGroupListOptions for the third Cat III subtype.
// Same semantic: empty StateFilter / zero TimeAfterNs+TimeBeforeNs
// / zero Limit+Offset all disable the corresponding filter.
type CampaignHypothesisListOptions struct {
	StateFilter  State
	TimeAfterNs  int64
	TimeBeforeNs int64
	Limit        int
	Offset       int
}

// withinCampaignHypothesisTimeWindow mirrors §0054's
// withinTimeWindow for CampaignHypothesisProjection.
func withinCampaignHypothesisTimeWindow(proj CampaignHypothesisProjection, opts CampaignHypothesisListOptions) bool {
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

// ProjectAllCampaignHypotheses returns the
// CampaignHypothesisProjection for EVERY CampaignHypothesisFormation
// in the substrate. Two-pass walk parallel to §0052's ProjectAll +
// §0062's ProjectAllAutomationGroups.
func ProjectAllCampaignHypotheses(ctx context.Context, sub *substrate.Substrate) (map[[32]byte]CampaignHypothesisProjection, error) {
	out := map[[32]byte]CampaignHypothesisProjection{}
	promotionToFormation := map[[32]byte][32]byte{}

	pass1 := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		switch row.MessageType {
		case campaignHypothesisFormationMessageType:
			proj, ok := out[row.EventHash]
			if !ok {
				proj = CampaignHypothesisProjection{FormationHash: row.EventHash}
			}
			proj.LifecycleHistory = append(proj.LifecycleHistory, LifecycleEntry{
				Type:      campaignHypothesisFormationMessageType,
				EventHash: row.EventHash,
				EventTime: row.EventTime,
			})
			out[row.EventHash] = proj

		case campaignHypothesisPromotionMessageType:
			payload, err := sub.ReadBlob(ctx, row.EventHash)
			if err != nil {
				return err
			}
			ev := &eventsv1.CampaignHypothesisPromotion{}
			if err := proto.Unmarshal(payload, ev); err != nil {
				return err
			}
			var formationHash [32]byte
			copy(formationHash[:], ev.FormationEventHash)
			promotionToFormation[row.EventHash] = formationHash

			proj, ok := out[formationHash]
			if !ok {
				proj = CampaignHypothesisProjection{FormationHash: formationHash}
			}
			proj.LifecycleHistory = append(proj.LifecycleHistory, LifecycleEntry{
				Type:      campaignHypothesisPromotionMessageType,
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
		return nil, fmt.Errorf("projection.ProjectAllCampaignHypotheses: pass-one walk: %w", pass1)
	}

	pass2 := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
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
				Type:      campaignHypothesisDemotionMessageType,
				EventHash: row.EventHash,
				EventTime: ev.DemotedAt,
			})
			if proj.LatestPromotion != nil &&
				bytes.Equal(ev.PromotionEventHash, campaignHypothesisPromotionHashOfMap(proj.LatestPromotion, promotionToFormation)) {
				if proj.LatestDemotion == nil || ev.DemotedAt > proj.LatestDemotion.DemotedAt {
					proj.LatestDemotion = ev
				}
			}
			out[formationHash] = proj

		case campaignHypothesisDissolutionMessageType:
			payload, err := sub.ReadBlob(ctx, row.EventHash)
			if err != nil {
				return err
			}
			ev := &eventsv1.CampaignHypothesisDissolution{}
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
				Type:      campaignHypothesisDissolutionMessageType,
				EventHash: row.EventHash,
				EventTime: ev.DissolvedAt,
			})
			if proj.Dissolution == nil || ev.DissolvedAt > proj.Dissolution.DissolvedAt {
				proj.Dissolution = ev
			}
			out[formationHash] = proj

		case campaignHypothesisMergeMessageType:
			payload, err := sub.ReadBlob(ctx, row.EventHash)
			if err != nil {
				return err
			}
			ev := &eventsv1.CampaignHypothesisMerge{}
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
					Type:      campaignHypothesisMergeMessageType,
					EventHash: row.EventHash,
					EventTime: ev.MergedAt,
				})
				if proj.MergedInto == nil || ev.MergedAt > proj.MergedInto.MergedAt {
					proj.MergedInto = ev
				}
				out[antHash] = proj
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
			var antHash [32]byte
			copy(antHash[:], ev.AntecedentFormationEventHash)
			proj, ok := out[antHash]
			if !ok {
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
			out[antHash] = proj
		}
		return nil
	})
	if pass2 != nil {
		return nil, fmt.Errorf("projection.ProjectAllCampaignHypotheses: pass-two walk: %w", pass2)
	}

	for hash, proj := range out {
		sort.SliceStable(proj.LifecycleHistory, func(i, j int) bool {
			return proj.LifecycleHistory[i].EventTime < proj.LifecycleHistory[j].EventTime
		})
		proj.State = computeCampaignHypothesisState(&proj)
		computeCampaignHypothesisLatencies(&proj)
		out[hash] = proj
	}

	return out, nil
}

// campaignHypothesisPromotionHashOfMap — reverse-lookup over the
// promotion-to-formation index used in
// ProjectAllCampaignHypotheses.
func campaignHypothesisPromotionHashOfMap(promotion *eventsv1.CampaignHypothesisPromotion, byPromotionHash map[[32]byte][32]byte) []byte {
	for promHash, formHash := range byPromotionHash {
		if bytes.Equal(promotion.FormationEventHash, formHash[:]) {
			return promHash[:]
		}
	}
	return nil
}

// ListCampaignHypotheses returns a deterministically-ordered list
// of CampaignHypothesisProjection over every formation, filtered by
// state + time-window + paged. Parallel to §0052's ListHypotheses +
// §0062's ListAutomationGroups.
func ListCampaignHypotheses(ctx context.Context, sub *substrate.Substrate, opts CampaignHypothesisListOptions) ([]CampaignHypothesisProjection, error) {
	all, err := ProjectAllCampaignHypotheses(ctx, sub)
	if err != nil {
		return nil, err
	}

	var filtered []CampaignHypothesisProjection
	for _, proj := range all {
		if opts.StateFilter != "" && proj.State != opts.StateFilter {
			continue
		}
		if !withinCampaignHypothesisTimeWindow(proj, opts) {
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

// CountCampaignHypothesesByState returns the aggregate StateCounts
// over every CampaignHypothesisFormation in the substrate. Parallel
// to §0053's CountByState + §0062's
// CountAutomationGroupsByState. Honors TimeAfterNs + TimeBeforeNs;
// ignores StateFilter / Limit / Offset.
func CountCampaignHypothesesByState(ctx context.Context, sub *substrate.Substrate, opts CampaignHypothesisListOptions) (StateCounts, error) {
	all, err := ProjectAllCampaignHypotheses(ctx, sub)
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
		if !withinCampaignHypothesisTimeWindow(proj, opts) {
			continue
		}
		counts.Total++
		counts.ByState[proj.State]++
	}
	return counts, nil
}
