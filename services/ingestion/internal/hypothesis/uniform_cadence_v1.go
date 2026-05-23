package hypothesis

import (
	"bytes"
	"fmt"
	"math"
	"sort"

	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
)

// UniformCadenceV1Signature is the stable pattern signature for the
// first canonical AutomationGroup formation pattern. Inference:
// actors whose DeclaredSessions arrive at statistically uniform
// inter-event intervals (low coefficient-of-variation) exhibit a
// behavioral signature consistent with automated (non-human)
// operation.
//
// This is the minimum-viable canonical example for AutomationGroup
// formation — analogous to session-descriptor-shared-v1 (§0045) for
// BehavioralCluster. More sophisticated patterns (multi-Cat-I-input
// signatures, machine-learning classifiers, ensemble detectors)
// extend the same AutomationGroupFormationPattern pathway
// mechanically.
const UniformCadenceV1Signature = "uniform-cadence-v1"

// UniformCadenceV1 is the first canonical AutomationGroup formation
// pattern. Algorithm:
//
//  1. Group every DeclaredSession in the substrate by actor_ref
//     (empty actor_refs are excluded — an unattributed session
//     cannot carry an automation-signature attribution).
//  2. For each actor with >= MinObservationCount sessions:
//     a. Sort the actor's sessions by declared_at ascending.
//     b. Compute the inter-event delta sequence (N-1 deltas).
//     c. Compute mean and population standard deviation of the deltas.
//     d. Compute coefficient of variation CoV = std_dev / mean.
//     e. If mean > 0 AND CoV <= MaxCoVThreshold, emit one
//        AutomationGroupFormation containing just this actor.
//
// Single-actor groups: the inception pattern emits a separate
// AutomationGroupFormation per qualifying actor. The §0056 + §0010
// definition permits sets of size >= 1 (a "set of actors" — sets
// of one are valid sets). Multi-actor grouping (e.g. by similar
// mean delta) is a §0057+ refinement; deferred per
// constitutional-minimalism (CLAUDE.md §7).
//
// Determinism: the grouping + sorting + numeric computation is
// byte-deterministic — same substrate state + same parameters
// produces content-hash-identical formation events. Floating-point
// CoV is computed identically across hosts (Go's math.Sqrt + IEEE-
// 754 single-precision is platform-stable for the precision needed
// here; the threshold comparison is a single < operator on float64).
//
// Confidence (placeholder pending Charter §2.6 redaction): computed
// as 1.0 - CoV/MaxCoVThreshold, so an actor at the threshold scores
// 0.0 and a perfectly-uniform actor (CoV = 0) scores 1.0. Same
// placeholder shape as §0045's confidenceFromClusterSize — both
// are structural-placeholders pending §2.6.
type UniformCadenceV1 struct {
	// MinObservationCount is the minimum number of DeclaredSessions
	// an actor must have for the cadence-uniformity check to apply.
	// Below this, the statistical CoV is too unstable to support
	// an automation claim. Default 5 (operator-supplied via CLI).
	MinObservationCount int64

	// MaxCoVThreshold is the coefficient-of-variation ceiling below
	// which an actor's cadence is "uniform enough" to match the
	// automation signature. Lower values are stricter. Typical
	// inception value 0.15 (15% std-dev relative to mean). Range
	// (0.0, 1.0] in practice; the pattern does not gate values
	// outside that range, but operator-supplied extreme values
	// will trivially match (1.0+) or trivially miss (0.0).
	MaxCoVThreshold float64
}

// Signature implements AutomationGroupFormationPattern.
func (p UniformCadenceV1) Signature() string { return UniformCadenceV1Signature }

// Parameters implements AutomationGroupFormationPattern. Canonical
// form: "max_cov_threshold=<float>;min_observation_count=<int>".
// Keys sorted alphabetically per the canonical-form discipline of
// §0045's Parameters().
//
// Float formatting uses %g (shortest representation that round-trips
// to the same float64) so logically-identical thresholds always
// produce byte-identical strings.
func (p UniformCadenceV1) Parameters() string {
	return fmt.Sprintf("max_cov_threshold=%g;min_observation_count=%d",
		p.MaxCoVThreshold, p.MinObservationCount)
}

// Form implements AutomationGroupFormationPattern. Groups
// DeclaredSessions by actor_ref; emits one AutomationGroupFormation
// per actor whose inter-event-delta CoV satisfies the threshold.
//
// formation_at is derived deterministically as the max declared_at
// across the actor's contributing DeclaredSessions — same "inference
// window closes at the last contributing observation" semantic as
// §0045's session-descriptor-shared-v1. The caller-supplied
// formationAt argument is IGNORED on purpose: the formation event's
// content-hash is the hypothesis's identity per Charter §2.3, and
// re-running with the same substrate state MUST produce the same
// hypothesis identity.
func (p UniformCadenceV1) Form(fctx AutomationGroupFormationContext, _ int64) []*eventsv1.AutomationGroupFormation {
	type member struct {
		hash       [32]byte
		declaredAt int64
	}

	// Group by actor_ref.
	byActor := map[string][]member{}
	for _, src := range fctx.DeclaredSessions() {
		actor := src.Session.GetActorRef()
		if actor == "" {
			continue
		}
		byActor[actor] = append(byActor[actor], member{
			hash:       src.Hash,
			declaredAt: src.Session.GetDeclaredAt(),
		})
	}

	// Iterate actors in deterministic order (sorted by actor_ref
	// ascending) so the output slice is deterministic.
	actors := make([]string, 0, len(byActor))
	for a := range byActor {
		actors = append(actors, a)
	}
	sort.Strings(actors)

	var formations []*eventsv1.AutomationGroupFormation
	for _, actor := range actors {
		members := byActor[actor]
		if int64(len(members)) < p.MinObservationCount {
			continue
		}

		// Sort by declared_at ascending; compute inter-event deltas.
		sort.Slice(members, func(i, j int) bool {
			return members[i].declaredAt < members[j].declaredAt
		})
		deltas := make([]float64, 0, len(members)-1)
		for i := 1; i < len(members); i++ {
			deltas = append(deltas, float64(members[i].declaredAt-members[i-1].declaredAt))
		}

		// Need at least 1 delta to compute statistics.
		if len(deltas) == 0 {
			continue
		}

		mean := 0.0
		for _, d := range deltas {
			mean += d
		}
		mean /= float64(len(deltas))
		if mean <= 0 {
			// Pathological: all observations at identical declared_at.
			// Mean of 0 makes CoV undefined; skip rather than emit a
			// degenerate formation.
			continue
		}

		variance := 0.0
		for _, d := range deltas {
			diff := d - mean
			variance += diff * diff
		}
		variance /= float64(len(deltas))
		stdDev := math.Sqrt(variance)
		cov := stdDev / mean

		if cov > p.MaxCoVThreshold {
			continue
		}

		// Compute formation_at = max(declared_at) across members.
		maxDeclaredAt := int64(0)
		hashes := make([][32]byte, 0, len(members))
		for _, m := range members {
			if m.declaredAt > maxDeclaredAt {
				maxDeclaredAt = m.declaredAt
			}
			hashes = append(hashes, m.hash)
		}
		sort.Slice(hashes, func(i, j int) bool {
			return bytes.Compare(hashes[i][:], hashes[j][:]) < 0
		})
		hashBytes := make([][]byte, 0, len(hashes))
		for _, h := range hashes {
			cp := make([]byte, 32)
			copy(cp, h[:])
			hashBytes = append(hashBytes, cp)
		}

		// Placeholder confidence: 1.0 - CoV/MaxCoVThreshold. Bounded
		// to [0, 1] by the threshold check above (CoV <= threshold).
		confidence := float32(1.0 - cov/p.MaxCoVThreshold)
		if confidence < 0 {
			confidence = 0
		}
		if confidence > 1 {
			confidence = 1
		}

		formations = append(formations, &eventsv1.AutomationGroupFormation{
			ActorRefs:   []string{actor},
			FormationAt: maxDeclaredAt,
			Confidence:  confidence,
			// EvidentialIndependence per §0140 — α = 1/1. This
			// formation reads only from Cat I DeclaredSessions
			// (cadence analysis over declared_at timestamps); no
			// promoted-hypothesis influence is consumed.
			EvidentialIndependence: &commonv1.EvidentialIndependence{Numerator: 1, Denominator: 1},
			SourceEventHashes:      hashBytes,
		})
	}

	return formations
}

// Ensure UniformCadenceV1 satisfies AutomationGroupFormationPattern
// at compile time.
var _ AutomationGroupFormationPattern = UniformCadenceV1{}
