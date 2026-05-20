// Package projection implements read-side materialization of
// Category III hypothesis lifecycle state per Charter §2.5 BC3
// ("hypothesis state is a projection over the operation event
// history, NOT a substrate row"). The substrate stores immutable
// lifecycle events (formation, promotion, demotion, dissolution,
// merge, split — landed §0045-§0050); this package reconstructs
// the current state of a single hypothesis on demand by walking
// those events.
//
// Scope at this layer (per decision-log §0051):
//   - Single-hypothesis projection. Multi-hypothesis aggregate
//     queries (e.g. "list every currently-promoted hypothesis")
//     are deferred to a follow-on landing.
//   - On-demand walk via substrate.WalkEvents. No caching, no
//     materialized indexes. Inception-phase posture per §0027 +
//     §0051: get the read shape right first; optimize later.
//   - Within-subtype only (BehavioralCluster). The lifecycle event
//     types this package recognizes are the five
//     BehavioralCluster* lifecycle-event message types from
//     §0045-§0050 + the formation itself.
//
// The package depends only on substrate (for the walk) +
// genproto/events/v1 (for unmarshaling lifecycle payloads). It
// does NOT depend on the hypothesis package — that dependency
// would be a layering violation (projection is read-only;
// hypothesis package is the writer surface).
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

// ErrFormationNotFound is returned when the supplied formation hash
// does not resolve to a substrate row whose message_type is
// BehavioralClusterFormation.
var ErrFormationNotFound = errors.New("projection: formation hash not found in substrate")

// ErrTargetNotFormation is returned when the supplied hash resolves
// to a substrate row whose message_type is not
// BehavioralClusterFormation. Mirrors the §2.5-lifecycle-integrity
// error class used by the hypothesis package's lifecycle operators.
var ErrTargetNotFormation = errors.New("projection: target hash is not a BehavioralClusterFormation")

// State enumerates the terminal lifecycle states reachable from a
// BehavioralClusterFormation per §2.5 lifecycle semantics.
type State string

const (
	// StateForming — formation row exists; no subsequent lifecycle
	// event targets it. The hypothesis is in active inference; no
	// operational use yet.
	StateForming State = "forming"

	// StatePromoted — most recent lifecycle event affecting
	// operational use is a promotion. The hypothesis is currently
	// admitted to operational use as enrichment context per §2.5 +
	// decision-log §0046.
	StatePromoted State = "promoted"

	// StateDemoted — a promotion was recorded and subsequently a
	// demotion of that promotion. The hypothesis is no longer in
	// operational use per §0047. Re-promotion is permitted and
	// returns the projection to StatePromoted.
	StateDemoted State = "demoted"

	// StateDissolved — a dissolution event references this formation.
	// The hypothesis is recognized as not corresponding to any
	// underlying phenomenon per §0048. Terminal per the canonical
	// definition (dissolution targets the formation directly; no
	// re-formation through the same formation hash is possible
	// since formation identity IS its content-hash per §0045).
	StateDissolved State = "dissolved"

	// StateMergedInto — a merge event references this formation as
	// one of its antecedents. The hypothesis identity is preserved
	// in the substrate (the formation row is immutable) but the
	// merge declares that this hypothesis and another describe the
	// same underlying phenomenon; the produced formation referenced
	// in the merge event is the surviving identity per §0049.
	StateMergedInto State = "merged_into"

	// StateSplitInto — a split event references this formation as
	// its antecedent. The hypothesis is recognized as having
	// contained multiple distinct phenomena, divided into the
	// successor formations referenced in the split event per §0050.
	StateSplitInto State = "split_into"
)

// LifecycleEntry is a normalized record of one lifecycle event
// reaching the projected formation. The Type field is the
// Protobuf descriptor's full name (e.g.
// "ghosttrace.events.v1.BehavioralClusterPromotion"); the EventHash
// is the lifecycle event's own substrate content-hash; EventTime
// is the lifecycle event's per-event timestamp field
// (promoted_at / demoted_at / dissolved_at / merged_at / split_at /
// formation_at) for chronological ordering.
type LifecycleEntry struct {
	Type      string
	EventHash [32]byte
	EventTime int64
}

// HypothesisProjection is the materialized current-state view of
// one BehavioralCluster hypothesis chain, reconstructed by
// ProjectHypothesis from the immutable lifecycle events in the
// substrate.
type HypothesisProjection struct {
	// FormationHash is the BLAKE3-256 content-hash of the
	// BehavioralClusterFormation that defines the hypothesis's
	// identity per §0045.
	FormationHash [32]byte

	// State is the projection's interpretation of where the
	// hypothesis chain currently sits in the §2.5 lifecycle.
	// Computed per the precedence rules documented in
	// computeState below.
	State State

	// LatestPromotion is the most-recent promotion event observed
	// for this formation (by event_time), or nil if no promotion
	// event references the formation.
	LatestPromotion *eventsv1.BehavioralClusterPromotion

	// LatestDemotion is the most-recent demotion event observed
	// that targets the LatestPromotion's event_hash. Nil if no
	// demotion has been recorded against the latest promotion.
	LatestDemotion *eventsv1.BehavioralClusterDemotion

	// Dissolution is the dissolution event for this formation, or
	// nil. Per §0048 there is no semantic reason to record
	// multiple dissolution events; if multiple are present (which
	// the substrate permits per the §0048 versioning carry-forward)
	// this field carries the one with the latest event_time.
	Dissolution *eventsv1.BehavioralClusterDissolution

	// MergedInto is the merge event in which this formation
	// appears as an antecedent. Nil if no such merge exists.
	MergedInto *eventsv1.BehavioralClusterMerge

	// SplitInto is the split event in which this formation
	// appears as the antecedent. Nil if no such split exists.
	SplitInto *eventsv1.BehavioralClusterSplit

	// LifecycleHistory is the full chronological list (ascending
	// by event_time) of every lifecycle event that reaches this
	// formation directly. Includes the formation event itself.
	// The list is for operator inspection; the State field
	// summarizes it.
	LifecycleHistory []LifecycleEntry
}

// Cat III lifecycle event message_type discriminators. Kept as
// package-level constants so the projection walk does not depend
// on the hypothesis package (read-only / writer-side layering).
const (
	formationMessageType   = "ghosttrace.events.v1.BehavioralClusterFormation"
	promotionMessageType   = "ghosttrace.events.v1.BehavioralClusterPromotion"
	demotionMessageType    = "ghosttrace.events.v1.BehavioralClusterDemotion"
	dissolutionMessageType = "ghosttrace.events.v1.BehavioralClusterDissolution"
	mergeMessageType       = "ghosttrace.events.v1.BehavioralClusterMerge"
	splitMessageType       = "ghosttrace.events.v1.BehavioralClusterSplit"
)

// ProjectHypothesis returns the current-state projection of the
// BehavioralCluster hypothesis identified by formationHash. Walks
// the substrate once, filtering Cat III lifecycle events that
// reach the formation either directly (promotion / dissolution /
// merge-antecedent / split-antecedent) or indirectly via the
// formation's promotion chain (demotion targets a promotion hash,
// not the formation directly).
//
// Per §0051 Option (A) (chosen over Option B's per-formation
// index): on-demand walk, no caching. The substrate's
// content-addressed primary-key already provides O(1) formation
// lookup; the linear walk is the read-side cost paid in exchange
// for not maintaining a materialized index. This is acceptable at
// inception-phase substrate sizes; a follow-on landing may
// introduce projection-side indexing if the linear walk becomes a
// bottleneck.
//
// Errors:
//   - ErrFormationNotFound: the formation hash does not resolve to
//     any substrate row.
//   - ErrTargetNotFormation: the target hash resolves to a row
//     whose message_type is not BehavioralClusterFormation.
func ProjectHypothesis(ctx context.Context, sub *substrate.Substrate, formationHash [32]byte) (HypothesisProjection, error) {
	formationRow, err := sub.LookupRow(ctx, formationHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return HypothesisProjection{}, fmt.Errorf("%w: %x", ErrFormationNotFound, formationHash)
		}
		return HypothesisProjection{}, fmt.Errorf("projection.ProjectHypothesis: lookup formation: %w", err)
	}
	if formationRow.MessageType != formationMessageType {
		return HypothesisProjection{}, fmt.Errorf("%w: %x is %q", ErrTargetNotFormation, formationHash, formationRow.MessageType)
	}

	proj := HypothesisProjection{
		FormationHash: formationHash,
		LifecycleHistory: []LifecycleEntry{{
			Type:      formationMessageType,
			EventHash: formationHash,
			EventTime: formationRow.EventTime,
		}},
	}

	// First pass: collect every promotion that targets this
	// formation. Promotion hashes are needed in pass-two to resolve
	// demotions (which target a promotion, not a formation).
	promotionsByHash := map[[32]byte]*eventsv1.BehavioralClusterPromotion{}

	walkErr := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != promotionMessageType {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return err
		}
		ev := &eventsv1.BehavioralClusterPromotion{}
		if err := proto.Unmarshal(payload, ev); err != nil {
			return err
		}
		if !bytes.Equal(ev.FormationEventHash, formationHash[:]) {
			return nil
		}
		promotionsByHash[row.EventHash] = ev
		proj.LifecycleHistory = append(proj.LifecycleHistory, LifecycleEntry{
			Type:      promotionMessageType,
			EventHash: row.EventHash,
			EventTime: ev.PromotedAt,
		})
		// Track latest by PromotedAt.
		if proj.LatestPromotion == nil || ev.PromotedAt > proj.LatestPromotion.PromotedAt {
			proj.LatestPromotion = ev
		}
		return nil
	})
	if walkErr != nil {
		return HypothesisProjection{}, fmt.Errorf("projection.ProjectHypothesis: pass-one walk: %w", walkErr)
	}

	// Second pass: every other Cat III lifecycle event type. Walk
	// once more so the demotion/dissolution/merge/split filters can
	// each consult the promotion map collected above.
	walkErr = sub.WalkEvents(ctx, func(row substrate.EventRow) error {
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
			var target [32]byte
			copy(target[:], ev.PromotionEventHash)
			if _, ok := promotionsByHash[target]; !ok {
				return nil // demotion targets a promotion of another formation
			}
			proj.LifecycleHistory = append(proj.LifecycleHistory, LifecycleEntry{
				Type:      demotionMessageType,
				EventHash: row.EventHash,
				EventTime: ev.DemotedAt,
			})
			// Track latest-demotion-targeting-latest-promotion.
			if proj.LatestPromotion != nil &&
				bytes.Equal(ev.PromotionEventHash, promotionEventHashOf(proj.LatestPromotion, promotionsByHash)) {
				if proj.LatestDemotion == nil || ev.DemotedAt > proj.LatestDemotion.DemotedAt {
					proj.LatestDemotion = ev
				}
			}

		case dissolutionMessageType:
			payload, err := sub.ReadBlob(ctx, row.EventHash)
			if err != nil {
				return err
			}
			ev := &eventsv1.BehavioralClusterDissolution{}
			if err := proto.Unmarshal(payload, ev); err != nil {
				return err
			}
			if !bytes.Equal(ev.FormationEventHash, formationHash[:]) {
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

		case mergeMessageType:
			payload, err := sub.ReadBlob(ctx, row.EventHash)
			if err != nil {
				return err
			}
			ev := &eventsv1.BehavioralClusterMerge{}
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
				Type:      mergeMessageType,
				EventHash: row.EventHash,
				EventTime: ev.MergedAt,
			})
			if proj.MergedInto == nil || ev.MergedAt > proj.MergedInto.MergedAt {
				proj.MergedInto = ev
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
			if !bytes.Equal(ev.AntecedentFormationEventHash, formationHash[:]) {
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
		}
		return nil
	})
	if walkErr != nil {
		return HypothesisProjection{}, fmt.Errorf("projection.ProjectHypothesis: pass-two walk: %w", walkErr)
	}

	// Stable chronological ordering of the lifecycle history.
	sort.SliceStable(proj.LifecycleHistory, func(i, j int) bool {
		return proj.LifecycleHistory[i].EventTime < proj.LifecycleHistory[j].EventTime
	})

	proj.State = computeState(&proj)
	return proj, nil
}

// computeState applies the precedence rules to a fully-populated
// projection. Precedence (highest → lowest):
//
//  1. Dissolution beats everything. Per §0048 dissolution recognizes
//     non-existence; once recorded, the chain reads as dissolved
//     regardless of any prior promotion/demotion/merge/split arc.
//  2. SplitInto beats MergedInto when both are present against the
//     same formation. Operational reading: a hypothesis split into
//     successors is the terminal partition; the prior merge
//     (whose produced formation became the split antecedent — see
//     §0050 + tests) is captured in LifecycleHistory but doesn't
//     dominate the current-state read.
//  3. MergedInto beats Promoted/Demoted/Forming.
//  4. Within the promote/demote arc: latest promotion vs latest
//     demotion-of-latest-promotion. If a demotion targets the latest
//     promotion, the state is Demoted; otherwise Promoted.
//  5. Default: Forming.
//
// The precedence rules are operational conventions of this
// projection, not Charter invariants. A future revision may
// redact them; the precedence chosen here matches the
// projection-time read shape operators are most likely to want
// (terminal states surface, in-flight states are visible via
// LifecycleHistory).
func computeState(p *HypothesisProjection) State {
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

// promotionEventHashOf finds the substrate event_hash of the
// supplied promotion within promotionsByHash by content-equality.
// Returns the zero hash if not found (caller's bytes.Equal then
// returns false naturally).
func promotionEventHashOf(promotion *eventsv1.BehavioralClusterPromotion, byHash map[[32]byte]*eventsv1.BehavioralClusterPromotion) []byte {
	for h, p := range byHash {
		if p == promotion {
			return h[:]
		}
	}
	return nil
}
