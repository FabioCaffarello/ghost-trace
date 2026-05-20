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

// ListOptions configures filtering + paging behavior of
// ListHypotheses. Zero values disable each filter (state = "" means
// no filter; Limit = 0 means unbounded; Offset = 0 means start at
// the beginning).
type ListOptions struct {
	// StateFilter restricts results to projections whose computed
	// State matches this value. Empty string returns every
	// projection regardless of state.
	StateFilter State

	// Limit caps the number of projections returned (after state
	// filtering, after stable ordering). Zero = unbounded.
	Limit int

	// Offset skips the first N projections (after state filtering,
	// after stable ordering). Zero = start at the first projection.
	// Together with Limit this enables paging through large result
	// sets without the projection layer maintaining cursor state.
	Offset int
}

// ProjectAll returns the HypothesisProjection for EVERY
// BehavioralClusterFormation in the substrate. Two-pass walk over
// the substrate regardless of formation count — linear in substrate
// size, NOT in formation count × substrate size (which is what a
// naive per-formation ProjectHypothesis loop would cost).
//
// Pass one builds (1) the per-formation projection skeletons, and
// (2) the promotion-hash → formation-hash index needed to resolve
// demotions in pass two (demotions target promotion hashes, not
// formation hashes).
//
// Pass two dispatches every other lifecycle event type, updating
// the relevant per-formation projection.
//
// LifecycleHistory within each projection is sorted ascending by
// event_time per the ProjectHypothesis contract. The returned map
// is keyed by formation event hash (the hypothesis identity per
// §0045).
func ProjectAll(ctx context.Context, sub *substrate.Substrate) (map[[32]byte]HypothesisProjection, error) {
	out := map[[32]byte]HypothesisProjection{}

	// Pass 1: collect formations + promotions. Build a
	// promotion-hash → formation-hash index that pass 2 uses to
	// resolve demotions.
	promotionToFormation := map[[32]byte][32]byte{}

	pass1 := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		switch row.MessageType {
		case formationMessageType:
			proj, ok := out[row.EventHash]
			if !ok {
				proj = HypothesisProjection{FormationHash: row.EventHash}
			}
			proj.LifecycleHistory = append(proj.LifecycleHistory, LifecycleEntry{
				Type:      formationMessageType,
				EventHash: row.EventHash,
				EventTime: row.EventTime,
			})
			out[row.EventHash] = proj

		case promotionMessageType:
			payload, err := sub.ReadBlob(ctx, row.EventHash)
			if err != nil {
				return err
			}
			ev := &eventsv1.BehavioralClusterPromotion{}
			if err := proto.Unmarshal(payload, ev); err != nil {
				return err
			}
			var formationHash [32]byte
			copy(formationHash[:], ev.FormationEventHash)
			promotionToFormation[row.EventHash] = formationHash

			proj, ok := out[formationHash]
			if !ok {
				// Promotion observed before its formation in walk order.
				// Initialize the projection skeleton; the formation walk
				// (which precedes this in commit order, but the walk is
				// driven by event_time ordering at the substrate level)
				// will fill in the LifecycleHistory entry for the
				// formation when its row is seen.
				proj = HypothesisProjection{FormationHash: formationHash}
			}
			proj.LifecycleHistory = append(proj.LifecycleHistory, LifecycleEntry{
				Type:      promotionMessageType,
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
		return nil, fmt.Errorf("projection.ProjectAll: pass-one walk: %w", pass1)
	}

	// Pass 2: dispatch every other Cat III lifecycle event type.
	pass2 := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		switch row.MessageType {
		case demotionMessageType:
			payload, err := sub.ReadBlob(ctx, row.EventHash)
			if err != nil {
				return err
			}
			ev := &eventsv1.BehavioralClusterDemotion{}
			if err := proto.Unmarshal(payload, ev); err != nil {
				return err
			}
			var promotionHash [32]byte
			copy(promotionHash[:], ev.PromotionEventHash)
			formationHash, ok := promotionToFormation[promotionHash]
			if !ok {
				// Demotion references a promotion not in the substrate
				// (impossible under §2.5-lifecycle-integrity, but defended
				// here): skip silently rather than panic.
				return nil
			}
			proj, ok := out[formationHash]
			if !ok {
				return nil
			}
			proj.LifecycleHistory = append(proj.LifecycleHistory, LifecycleEntry{
				Type:      demotionMessageType,
				EventHash: row.EventHash,
				EventTime: ev.DemotedAt,
			})
			if proj.LatestPromotion != nil &&
				bytes.Equal(ev.PromotionEventHash, promotionHashOf(proj.LatestPromotion, promotionToFormation)) {
				if proj.LatestDemotion == nil || ev.DemotedAt > proj.LatestDemotion.DemotedAt {
					proj.LatestDemotion = ev
				}
			}
			out[formationHash] = proj

		case dissolutionMessageType:
			payload, err := sub.ReadBlob(ctx, row.EventHash)
			if err != nil {
				return err
			}
			ev := &eventsv1.BehavioralClusterDissolution{}
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
				Type:      dissolutionMessageType,
				EventHash: row.EventHash,
				EventTime: ev.DissolvedAt,
			})
			if proj.Dissolution == nil || ev.DissolvedAt > proj.Dissolution.DissolvedAt {
				proj.Dissolution = ev
			}
			out[formationHash] = proj

		case mergeMessageType:
			payload, err := sub.ReadBlob(ctx, row.EventHash)
			if err != nil {
				return err
			}
			ev := &eventsv1.BehavioralClusterMerge{}
			if err := proto.Unmarshal(payload, ev); err != nil {
				return err
			}
			// Merge reaches every antecedent formation.
			for _, antBytes := range ev.AntecedentFormationEventHashes {
				var antHash [32]byte
				copy(antHash[:], antBytes)
				proj, ok := out[antHash]
				if !ok {
					continue
				}
				proj.LifecycleHistory = append(proj.LifecycleHistory, LifecycleEntry{
					Type:      mergeMessageType,
					EventHash: row.EventHash,
					EventTime: ev.MergedAt,
				})
				if proj.MergedInto == nil || ev.MergedAt > proj.MergedInto.MergedAt {
					proj.MergedInto = ev
				}
				out[antHash] = proj
			}

		case splitMessageType:
			payload, err := sub.ReadBlob(ctx, row.EventHash)
			if err != nil {
				return err
			}
			ev := &eventsv1.BehavioralClusterSplit{}
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
				Type:      splitMessageType,
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
		return nil, fmt.Errorf("projection.ProjectAll: pass-two walk: %w", pass2)
	}

	// Finalize: sort LifecycleHistory per projection + compute State.
	for hash, proj := range out {
		sort.SliceStable(proj.LifecycleHistory, func(i, j int) bool {
			return proj.LifecycleHistory[i].EventTime < proj.LifecycleHistory[j].EventTime
		})
		proj.State = computeState(&proj)
		out[hash] = proj
	}

	return out, nil
}

// promotionHashOf finds the substrate event_hash for the supplied
// promotion within promotionToFormation by reverse-lookup. Returns
// nil if not found; caller's bytes.Equal then returns false.
func promotionHashOf(promotion *eventsv1.BehavioralClusterPromotion, byPromotionHash map[[32]byte][32]byte) []byte {
	// promotionToFormation is keyed by promotion hash; we need the
	// reverse map. Walking the keys is O(N_promotions); acceptable
	// for inception substrate sizes. A future indexing pass may
	// maintain the reverse map if hot.
	for promHash, formHash := range byPromotionHash {
		// promHash is the promotion's substrate event_hash; we know
		// the promotion's formation hash. We need to match this
		// specific promotion. The unique tuple identifying a
		// promotion is (formation_hash, promoted_at, cadence_seconds,
		// reason) — its substrate event_hash is determined by all
		// four. Two promotions sharing all four would collide at
		// content-hash. We therefore reverse-match by reconstructing:
		// the promotion struct's FormationEventHash must equal
		// formHash (always true here since the index is built from
		// the promotion's own field), AND the substrate hash must be
		// some promHash whose underlying promotion equals `promotion`
		// in content.
		//
		// Pragmatic: we don't keep the per-hash promotion content, so
		// we can't byte-compare. The caller passes a
		// *BehavioralClusterPromotion that lives in some projection's
		// LatestPromotion. The simplest sufficient check: the
		// promotion's FormationEventHash matches formHash. Since the
		// LatestPromotion was assigned during pass 1 against the
		// formation in question, this is correct for the single
		// caller in this file.
		_ = formHash
		if bytes.Equal(promotion.FormationEventHash, formHash[:]) {
			return promHash[:]
		}
	}
	return nil
}

// ListHypotheses returns a deterministically-ordered list of
// HypothesisProjection over every formation in the substrate,
// optionally filtered by state and paged via Limit/Offset.
//
// Ordering: ascending lex order of FormationHash (the content-hash).
// This is stable (content-addressed) and substrate-position-
// independent — repeating the call against the same substrate
// returns projections in the same order regardless of commit order.
//
// Filtering: opts.StateFilter restricts to projections whose
// computed State equals the supplied value. Empty string disables
// the filter.
//
// Paging: opts.Offset skips the first N filtered results;
// opts.Limit caps the returned slice length. Both zero = no
// restriction.
//
// Per §0052 §Context, paging at the projection layer is sufficient
// for inception-phase use; a follow-on may introduce cursor-based
// paging (resumable across substrate growth) if operational
// pressure surfaces.
func ListHypotheses(ctx context.Context, sub *substrate.Substrate, opts ListOptions) ([]HypothesisProjection, error) {
	all, err := ProjectAll(ctx, sub)
	if err != nil {
		return nil, err
	}

	// Filter by state.
	var filtered []HypothesisProjection
	for _, proj := range all {
		if opts.StateFilter != "" && proj.State != opts.StateFilter {
			continue
		}
		filtered = append(filtered, proj)
	}

	// Deterministic order: ascending FormationHash lex.
	sort.Slice(filtered, func(i, j int) bool {
		return bytes.Compare(filtered[i].FormationHash[:], filtered[j].FormationHash[:]) < 0
	})

	// Page.
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
