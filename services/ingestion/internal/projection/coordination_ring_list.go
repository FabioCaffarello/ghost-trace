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

// CoordinationRingListOptions configures filtering + paging
// behavior of ListCoordinationRings + CountCoordinationRingsByState.
// Parallel to §0052/§0062/§0069 list options for the fourth Cat
// III subtype.
type CoordinationRingListOptions struct {
	StateFilter  State
	TimeAfterNs  int64
	TimeBeforeNs int64
	Limit        int
	Offset       int
}

// withinCoordinationRingTimeWindow mirrors §0054 / §0062 / §0069.
func withinCoordinationRingTimeWindow(proj CoordinationRingProjection, opts CoordinationRingListOptions) bool {
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

// ProjectAllCoordinationRings returns the CoordinationRingProjection
// for EVERY CoordinationRingFormation in the substrate. Two-pass walk
// parallel to §0052 + §0062 + §0069.
func ProjectAllCoordinationRings(ctx context.Context, sub *substrate.Substrate) (map[[32]byte]CoordinationRingProjection, error) {
	out := map[[32]byte]CoordinationRingProjection{}
	promotionToFormation := map[[32]byte][32]byte{}

	pass1 := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		switch row.MessageType {
		case coordinationRingFormationMessageType:
			proj, ok := out[row.EventHash]
			if !ok {
				proj = CoordinationRingProjection{FormationHash: row.EventHash}
			}
			proj.LifecycleHistory = append(proj.LifecycleHistory, LifecycleEntry{
				Type:      coordinationRingFormationMessageType,
				EventHash: row.EventHash,
				EventTime: row.EventTime,
			})
			out[row.EventHash] = proj

		case coordinationRingPromotionMessageType:
			payload, err := sub.ReadBlob(ctx, row.EventHash)
			if err != nil {
				return err
			}
			ev := &eventsv1.CoordinationRingPromotion{}
			if err := proto.Unmarshal(payload, ev); err != nil {
				return err
			}
			var formationHash [32]byte
			copy(formationHash[:], ev.FormationEventHash)
			promotionToFormation[row.EventHash] = formationHash

			proj, ok := out[formationHash]
			if !ok {
				proj = CoordinationRingProjection{FormationHash: formationHash}
			}
			proj.LifecycleHistory = append(proj.LifecycleHistory, LifecycleEntry{
				Type:      coordinationRingPromotionMessageType,
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
		return nil, fmt.Errorf("projection.ProjectAllCoordinationRings: pass-one walk: %w", pass1)
	}

	pass2 := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
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
				Type:      coordinationRingDemotionMessageType,
				EventHash: row.EventHash,
				EventTime: ev.DemotedAt,
			})
			if proj.LatestPromotion != nil &&
				bytes.Equal(ev.PromotionEventHash, coordinationRingPromotionHashOfMap(proj.LatestPromotion, promotionToFormation)) {
				if proj.LatestDemotion == nil || ev.DemotedAt > proj.LatestDemotion.DemotedAt {
					proj.LatestDemotion = ev
				}
			}
			out[formationHash] = proj

		case coordinationRingDissolutionMessageType:
			payload, err := sub.ReadBlob(ctx, row.EventHash)
			if err != nil {
				return err
			}
			ev := &eventsv1.CoordinationRingDissolution{}
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
				Type:      coordinationRingDissolutionMessageType,
				EventHash: row.EventHash,
				EventTime: ev.DissolvedAt,
			})
			if proj.Dissolution == nil || ev.DissolvedAt > proj.Dissolution.DissolvedAt {
				proj.Dissolution = ev
			}
			out[formationHash] = proj

		case coordinationRingMergeMessageType:
			payload, err := sub.ReadBlob(ctx, row.EventHash)
			if err != nil {
				return err
			}
			ev := &eventsv1.CoordinationRingMerge{}
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
					Type:      coordinationRingMergeMessageType,
					EventHash: row.EventHash,
					EventTime: ev.MergedAt,
				})
				if proj.MergedInto == nil || ev.MergedAt > proj.MergedInto.MergedAt {
					proj.MergedInto = ev
				}
				out[antHash] = proj
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
			var antHash [32]byte
			copy(antHash[:], ev.AntecedentFormationEventHash)
			proj, ok := out[antHash]
			if !ok {
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
			out[antHash] = proj
		}
		return nil
	})
	if pass2 != nil {
		return nil, fmt.Errorf("projection.ProjectAllCoordinationRings: pass-two walk: %w", pass2)
	}

	for hash, proj := range out {
		sort.SliceStable(proj.LifecycleHistory, func(i, j int) bool {
			return proj.LifecycleHistory[i].EventTime < proj.LifecycleHistory[j].EventTime
		})
		proj.State = computeCoordinationRingState(&proj)
		computeCoordinationRingLatencies(&proj)
		out[hash] = proj
	}

	return out, nil
}

// coordinationRingPromotionHashOfMap — reverse-lookup over the
// promotion-to-formation index used in ProjectAllCoordinationRings.
func coordinationRingPromotionHashOfMap(promotion *eventsv1.CoordinationRingPromotion, byPromotionHash map[[32]byte][32]byte) []byte {
	for promHash, formHash := range byPromotionHash {
		if bytes.Equal(promotion.FormationEventHash, formHash[:]) {
			return promHash[:]
		}
	}
	return nil
}

// ListCoordinationRings returns a deterministically-ordered list of
// CoordinationRingProjection over every formation, filtered by state
// + time-window + paged.
func ListCoordinationRings(ctx context.Context, sub *substrate.Substrate, opts CoordinationRingListOptions) ([]CoordinationRingProjection, error) {
	all, err := ProjectAllCoordinationRings(ctx, sub)
	if err != nil {
		return nil, err
	}

	var filtered []CoordinationRingProjection
	for _, proj := range all {
		if opts.StateFilter != "" && proj.State != opts.StateFilter {
			continue
		}
		if !withinCoordinationRingTimeWindow(proj, opts) {
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

// CountCoordinationRingsByState returns the aggregate StateCounts
// over every CoordinationRingFormation in the substrate.
func CountCoordinationRingsByState(ctx context.Context, sub *substrate.Substrate, opts CoordinationRingListOptions) (StateCounts, error) {
	all, err := ProjectAllCoordinationRings(ctx, sub)
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
		if !withinCoordinationRingTimeWindow(proj, opts) {
			continue
		}
		counts.Total++
		counts.ByState[proj.State]++
	}
	return counts, nil
}
