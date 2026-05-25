// endpoint_co_visit_v1 is the F3 signature emitting candidates of
// the CoordinationRing subtype per decision-log §0185. Fourth and
// final F3 emission subtype (after AutomationGroup at §0152 +
// BehavioralCluster at §0174 + CampaignHypothesis at §0182); first
// INTERACTION-CENTRIC signature in the F3 corpus.
//
// Detection axis: NetworkObservation events sharing identical
// (endpoint_ref, time_bucket) anchor a coordination edge between
// every distinct actor pair in that bucket. Per-bucket: when >=
// threshold distinct actors co-visit a shared endpoint, the bucket
// emits one CoordinationRing FormationCandidate whose interactions
// field carries the complete pairwise edge set (lex-canonical per
// §0070).
//
// Structurally distinct from prior F3 corpus:
//
//   - Interaction-centric, not actor-centric (AutomationGroup +
//     BehavioralCluster) and not event-centric (CampaignHypothesis).
//     The candidate's Interactions field carries edges; ActorRefs is
//     the recoverable vertex set (union of edge endpoints) preserved
//     for telemetry symmetry.
//   - CoordinationRingFormation proto's `interactions` repeated field
//     is the structural membership shape (per §0070): operators
//     committing CoordinationRingFormation convert candidate.Interactions
//     into CoordinationRingInteraction protos with actor_a=edge[0] +
//     actor_b=edge[1].
//   - Empty actor_ref IS a skip reason (signature cannot establish
//     a coordination edge without two named endpoints — actor-pair-
//     based inference requires both actors named).
//
// Per §0144 discriminated-union: signature consumes NetworkObservation
// typed envelope; sub-modality untouched (coordination inference is
// at the actor-pair level, independent of sub-modality content).
//
// Per §0168 Decision A.1 (signature-aware Cat II consumption):
// EvaluateNetwork accepts AttributionLookup. Used here for actor
// enrichment when actor_ref is empty + lookup yields a Cat II derived
// actor; the Cat II derivation hash is threaded into the candidate
// SourceHashes alongside the Cat I observation hash per §2.3 chain
// preservation.
//
// Per-bucket simplification: this MVP signature emits one candidate
// per (endpoint, time_bucket) cluster. Cross-bucket connected-component
// aggregation (where edges persist across buckets to grow rings) is
// deferred to a future signature variant — the per-bucket shape is
// structurally meaningful (coordinated targeting within a window is
// the simplest coordination signal) and mirrors §0182's per-bucket
// candidate-per-cluster shape on the interaction-centric side.
package signatures

import (
	"context"
	"fmt"
	"sort"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
)

// endpointCoVisitDefaultBucketNs is the default time-bucket width in
// nanoseconds (60 seconds). Each event's observed_at is integer-divided
// by this value to compute the bucket assignment. Per §0185: 60s
// mirrors §0182 conservative-default discipline (coordination bursts
// typically span the same minute-scale window as targeting campaigns).
const endpointCoVisitDefaultBucketNs = uint64(60_000_000_000)

// EndpointCoVisitV1 implements the endpoint_co_visit_v1 signature.
// Stateless; reusable across invocations.
type EndpointCoVisitV1 struct {
	// Threshold is the minimum distinct-ACTOR count per (endpoint_ref,
	// time_bucket) cluster for a candidate to be emitted. Default 3
	// per §0185 conservative-default discipline (mirrors §0182's
	// event-count threshold of 3; for coordination a 3-actor minimum
	// produces at least 3 edges, a structurally non-trivial ring).
	Threshold uint32

	// BucketSeconds overrides the default time-bucket width
	// (60 seconds). 0 = default. Mirrors §0182's BucketSeconds override
	// surface for operator-tunable window discipline.
	BucketSeconds uint64
}

// Name identifies the signature for instrumentation + versioning.
func (s *EndpointCoVisitV1) Name() string { return "endpoint_co_visit_v1" }

// Subtype identifies the Cat III hypothesis subtype: CoordinationRing
// (interaction-centric "set of actors whose patterns of interaction
// suggest coordinated action" per §0010 Q2-A.2 + entity-model.md
// §Category III).
func (s *EndpointCoVisitV1) Subtype() HypothesisSubtype {
	return HypothesisSubtypeCoordinationRing
}

// isNetworkSignature is the NetworkSignature marker.
func (s *EndpointCoVisitV1) isNetworkSignature() {}

// effectiveThreshold returns Threshold or the default (3) when unset.
func (s *EndpointCoVisitV1) effectiveThreshold() uint32 {
	if s.Threshold == 0 {
		return 3
	}
	return s.Threshold
}

// effectiveBucketNs returns the time-bucket width in nanoseconds:
// BucketSeconds converted, or the default 60s when unset.
func (s *EndpointCoVisitV1) effectiveBucketNs() uint64 {
	if s.BucketSeconds == 0 {
		return endpointCoVisitDefaultBucketNs
	}
	return s.BucketSeconds * 1_000_000_000
}

// EvaluateNetwork evaluates the signature against a slice of
// NetworkObservation records. Per-(endpoint_ref, time_bucket)
// clustering: groups ACTORS (deduplicated) into temporal cohorts;
// emits one CoordinationRing FormationCandidate per cluster whose
// distinct-actor count meets the threshold. Candidate Interactions
// field carries the complete pairwise edge set (lex-canonical per
// §0070).
//
// Returns EvaluationResult with candidates in cluster-key alphabetical
// order (deterministic) + EvaluationStats per §0143 Sub-benchmark 1.
//
// Skip semantics:
//   - nil obs: silently skipped (no counter).
//   - empty endpoint_ref: ObservationsSkippedWrongModality++ (no
//     targeting context — cannot anchor a coordination edge).
//   - observed_at == 0: ObservationsSkippedWrongModality++ (no
//     temporal context — cannot anchor to a time bucket).
//   - EMPTY actor_ref IS a skip reason: ObservationsSkippedNoActor++.
//     Unlike §0182 CampaignHypothesis (event-centric, actor-optional),
//     coordination inference REQUIRES two named actors per edge; an
//     unnamed observation cannot participate in a coordination edge.
//   - Single-actor bucket (threshold not met): silently skipped.
//
// Per §0168 Decision A.1: when attribution is provided AND actor_ref
// is empty AND For returns ok, the derived actor_ref is used as the
// effective actor + Cat II derivation hash threaded into the candidate
// SourceHashes alongside the Cat I observation hash. Per-§2.3 chain
// preservation.
//
// ActorRefs in candidate: deduplicated sorted union of edge endpoints
// (recoverable vertex set per §0070 "operators that want set-shaped
// membership compute the union locally"). Carried at the candidate
// layer for telemetry symmetry with prior actor-centric subtypes.
func (s *EndpointCoVisitV1) EvaluateNetwork(ctx context.Context, observations []*eventsv1.NetworkObservation, attribution AttributionLookup) (*EvaluationResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	threshold := s.effectiveThreshold()
	bucketNs := s.effectiveBucketNs()
	stats := EvaluationStats{
		ObservationsScanned: uint32(len(observations)),
		PerCollector:        make(map[string]uint32),
	}

	// Per-cluster aggregation. actorSourceHashes preserves the source-
	// hash list per actor so the candidate's SourceHashes reflects all
	// contributing observations + Cat II attribution hashes when
	// consumed.
	type cluster struct {
		actors       map[string]struct{}
		sourceHashes [][]byte
	}
	perBucket := make(map[string]*cluster)
	distinctActors := make(map[string]struct{})

	for _, obs := range observations {
		if obs == nil {
			continue
		}
		if obs.CollectorRef != "" {
			stats.PerCollector[obs.CollectorRef]++
		}

		if obs.EndpointRef == "" {
			stats.ObservationsSkippedWrongModality++
			continue
		}
		if obs.ObservedAt == 0 {
			stats.ObservationsSkippedWrongModality++
			continue
		}

		_, obsHash, err := canonical.MarshalAndHash(obs)
		if err != nil {
			return nil, fmt.Errorf("endpoint_co_visit_v1: hash observation: %w", err)
		}

		// Actor resolution: declared takes precedence; Cat II fills
		// gap if attribution provided + actor empty. Skip when no
		// effective actor (coordination requires named actors).
		effectiveActor := obs.ActorRef
		var attributionHash [32]byte
		var attributionPresent bool
		if effectiveActor == "" && attribution != nil {
			if derived, attHash, ok := attribution.For(obsHash); ok {
				effectiveActor = derived
				attributionHash = attHash
				attributionPresent = true
			}
		}
		if effectiveActor == "" {
			stats.ObservationsSkippedNoActor++
			continue
		}

		bucketIndex := uint64(obs.ObservedAt) / bucketNs
		bucketKey := fmt.Sprintf("endpoint=%s;bucket=%d", obs.EndpointRef, bucketIndex)

		c, ok := perBucket[bucketKey]
		if !ok {
			c = &cluster{actors: make(map[string]struct{})}
			perBucket[bucketKey] = c
		}
		c.actors[effectiveActor] = struct{}{}
		distinctActors[effectiveActor] = struct{}{}

		obsHashCopy := make([]byte, 32)
		copy(obsHashCopy, obsHash[:])
		c.sourceHashes = append(c.sourceHashes, obsHashCopy)

		if attributionPresent {
			attHashCopy := make([]byte, 32)
			copy(attHashCopy, attributionHash[:])
			c.sourceHashes = append(c.sourceHashes, attHashCopy)
		}
	}
	stats.ActorsAggregated = uint32(len(distinctActors))

	bucketKeys := make([]string, 0, len(perBucket))
	for fk := range perBucket {
		bucketKeys = append(bucketKeys, fk)
	}
	sort.Strings(bucketKeys)

	out := make([]*FormationCandidate, 0)
	for _, fk := range bucketKeys {
		c := perBucket[fk]
		actorCount := uint32(len(c.actors))
		if actorCount < threshold {
			continue
		}
		stats.ActorsAboveThreshold += actorCount

		// Vertex set: deduplicated sorted union of cluster actors.
		actorRefs := make([]string, 0, len(c.actors))
		for a := range c.actors {
			actorRefs = append(actorRefs, a)
		}
		sort.Strings(actorRefs)

		// Edge set: complete pairwise enumeration (i < j ensures each
		// pair appears once + edge[0] < edge[1] lex per §0070). Per-
		// edge canonicalization preserved because actorRefs is already
		// sorted ascending.
		edgeCount := len(actorRefs) * (len(actorRefs) - 1) / 2
		interactions := make([][2]string, 0, edgeCount)
		for i := 0; i < len(actorRefs); i++ {
			for j := i + 1; j < len(actorRefs); j++ {
				interactions = append(interactions, [2]string{actorRefs[i], actorRefs[j]})
			}
		}
		// interactions is already ASCENDING-sorted by (edge[0], edge[1])
		// because actorRefs was sorted + the nested i<j loop preserves
		// lex order. No additional sort step required.

		sort.Slice(c.sourceHashes, func(i, j int) bool {
			return bytesLess(c.sourceHashes[i], c.sourceHashes[j])
		})

		out = append(out, &FormationCandidate{
			SignatureName:     s.Name(),
			HypothesisSubtype: s.Subtype(),
			ActorRefs:         actorRefs,
			SourceHashes:      c.sourceHashes,
			EvidenceCount:     uint32(len(c.sourceHashes)),
			// ConfidenceHint scales with distinct actor count (the
			// vertex-set cardinality is the primary coordination
			// signal; edge count is derivable). Caps at 0.9 per §3 N1.
			ConfidenceHint: confidenceFromCount(actorCount, threshold),
			Interactions:   interactions,
		})
	}
	stats.CandidatesEmitted = uint32(len(out))
	return &EvaluationResult{Candidates: out, Stats: stats}, nil
}
