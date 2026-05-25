// temporal_endpoint_cohort_v1 is the first F3 signature emitting
// candidates of the CampaignHypothesis subtype per decision-log
// §0182. Third F3 emission subtype (after AutomationGroup at §0152 +
// BehavioralCluster at §0174); first event-centric signature in the
// F3 corpus.
//
// Detection axis: NetworkObservation events sharing identical
// (endpoint_ref, time_bucket) cluster as a CampaignHypothesis per
// the ontology shape: "set of events whose patterns suggest
// membership in a unified operation". The "operation" inference is
// targeting-coordinated (many events hitting the same endpoint
// within the same time window).
//
// Structurally distinct from prior F3 corpus:
//
//   - Event-centric, not actor-centric. CampaignHypothesisFormation
//     proto has NO actor_refs field (per §0063 entity-model framing);
//     actors are incidental. ActorRefs in FormationCandidate is
//     collected as enrichment metadata only.
//   - First F3 signature without sub-modality dispatch. Reads ONLY
//     envelope fields (endpoint_ref + observed_at); sub-modality is
//     irrelevant to event-set clustering.
//   - Empty actor_ref is NOT a skip reason — events without actor
//     still count in the event-set.
//
// Per §0144 discriminated-union: signature consumes NetworkObservation
// typed envelope; sub-modality untouched (signature treats all
// NetworkObservation sub-modalities identically for clustering
// purposes — endpoint targeting at time bucket is the discriminator).
//
// Per §0168 Decision A.1 (signature-aware Cat II consumption):
// EvaluateNetwork accepts AttributionLookup. Used here for actor
// enrichment when present + actor_ref empty; Cat II derivation hash
// threaded into candidate SourceHashes when consumed per §2.3 chain.
package signatures

import (
	"context"
	"fmt"
	"sort"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
)

// temporalEndpointCohortDefaultBucketNs is the default time-bucket
// width in nanoseconds (60 seconds). Each event's observed_at is
// integer-divided by this value to compute the bucket assignment.
// Per §0182: 60s balances campaign-burst observability (typical
// targeting campaigns issue requests within ~minute windows) against
// noise (sub-minute clusters often reflect normal traffic).
const temporalEndpointCohortDefaultBucketNs = uint64(60_000_000_000)

// TemporalEndpointCohortV1 implements the temporal_endpoint_cohort_v1
// signature. Stateless; reusable across invocations.
type TemporalEndpointCohortV1 struct {
	// Threshold is the minimum distinct-EVENT count per
	// (endpoint_ref, time_bucket) cluster for a candidate to be
	// emitted. Default 3 per §0182 conservative-default discipline
	// (mirrors §0161 / §0169 / §0174 cluster-cardinality discipline).
	Threshold uint32

	// BucketSeconds overrides the default time-bucket width
	// (60 seconds). 0 = default.
	BucketSeconds uint64
}

// Name identifies the signature for instrumentation + versioning.
func (s *TemporalEndpointCohortV1) Name() string { return "temporal_endpoint_cohort_v1" }

// Subtype identifies the Cat III hypothesis subtype: CampaignHypothesis
// (event-centric "set of events whose patterns suggest membership in
// a unified operation" per §0010 Q2-A.2).
func (s *TemporalEndpointCohortV1) Subtype() HypothesisSubtype {
	return HypothesisSubtypeCampaignHypothesis
}

// isNetworkSignature is the NetworkSignature marker.
func (s *TemporalEndpointCohortV1) isNetworkSignature() {}

// effectiveThreshold returns Threshold or the default (3) when unset.
func (s *TemporalEndpointCohortV1) effectiveThreshold() uint32 {
	if s.Threshold == 0 {
		return 3
	}
	return s.Threshold
}

// effectiveBucketNs returns the time-bucket width in nanoseconds:
// BucketSeconds converted, or the default 60s when unset.
func (s *TemporalEndpointCohortV1) effectiveBucketNs() uint64 {
	if s.BucketSeconds == 0 {
		return temporalEndpointCohortDefaultBucketNs
	}
	return s.BucketSeconds * 1_000_000_000
}

// EvaluateNetwork evaluates the signature against a slice of
// NetworkObservation records. Per-(endpoint_ref, time_bucket)
// clustering: groups EVENTS (not actors) into temporal cohorts;
// emits one CampaignHypothesis FormationCandidate per cluster whose
// event-count meets the threshold.
//
// Returns EvaluationResult with candidates in cluster-key
// alphabetical order (deterministic) + EvaluationStats per §0143
// Sub-benchmark 1.
//
// Skip semantics (structurally different from prior F3 signatures):
//   - nil obs: silently skipped (no counter).
//   - empty endpoint_ref: ObservationsSkippedWrongModality++ (no
//     targeting context — event cannot anchor to a campaign).
//   - observed_at == 0: ObservationsSkippedWrongModality++ (no
//     temporal context — event cannot anchor to a time bucket).
//   - EMPTY actor_ref is NOT a skip reason. Events without actor
//     still count in the event-set; per §0063 CampaignHypothesis is
//     event-centric, not actor-centric.
//
// Per §0168 Decision A.1: when attribution is provided AND actor_ref
// is empty AND For returns ok, the derived actor_ref is collected as
// enrichment metadata + Cat II derivation hash threaded into the
// candidate SourceHashes alongside the Cat I observation hash. Per-
// §2.3 chain preservation.
//
// ActorRefs in candidate: distinct actor_refs across the cluster
// (deduplicated; may be empty when all events lack actor_ref + no
// AttributionLookup matches). NOT structurally committed to the
// CampaignHypothesisFormation proto (proto has no actor_refs field);
// emitted at candidate layer for operator-review-time visibility.
func (s *TemporalEndpointCohortV1) EvaluateNetwork(ctx context.Context, observations []*eventsv1.NetworkObservation, attribution AttributionLookup) (*EvaluationResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	threshold := s.effectiveThreshold()
	bucketNs := s.effectiveBucketNs()
	stats := EvaluationStats{
		ObservationsScanned: uint32(len(observations)),
		PerCollector:        make(map[string]uint32),
	}

	// Per-cluster aggregation. eventCount tracks Cat I observations
	// (separate from sourceHashes which may include Cat II attribution
	// hashes); threshold check is against eventCount.
	type cluster struct {
		actors       map[string]struct{}
		sourceHashes [][]byte
		eventCount   uint32
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

		// Envelope-only checks. Sub-modality untouched.
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
			return nil, fmt.Errorf("temporal_endpoint_cohort_v1: hash observation: %w", err)
		}

		// Actor enrichment: declared takes precedence; Cat II fills
		// gap if attribution provided + actor empty. Empty actor is
		// NOT a skip (event still counts in cluster).
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

		// Compute time bucket: integer-divide observed_at by bucket
		// width. Negative observed_at not handled (int64 > 0 in
		// practice; production substrates timestamp post-Unix-epoch).
		bucketIndex := uint64(obs.ObservedAt) / bucketNs
		bucketKey := fmt.Sprintf("endpoint=%s;bucket=%d", obs.EndpointRef, bucketIndex)

		c, ok := perBucket[bucketKey]
		if !ok {
			c = &cluster{actors: make(map[string]struct{})}
			perBucket[bucketKey] = c
		}

		if effectiveActor != "" {
			c.actors[effectiveActor] = struct{}{}
			distinctActors[effectiveActor] = struct{}{}
		}

		// Cat I observation hash always added; eventCount tracks
		// Cat I obs separately from Cat II attribution hashes.
		obsHashCopy := make([]byte, 32)
		copy(obsHashCopy, obsHash[:])
		c.sourceHashes = append(c.sourceHashes, obsHashCopy)
		c.eventCount++

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
		// Threshold check: EVENT count (excludes Cat II attribution
		// hashes), not sourceHashes count.
		if c.eventCount < threshold {
			continue
		}
		// ActorsAboveThreshold semantics: track distinct actors
		// across emitted clusters (may include 0 for actor-less
		// clusters; consistent with event-centric ontology).
		stats.ActorsAboveThreshold += uint32(len(c.actors))

		// ActorRefs in candidate: deduplicated sorted list of actors
		// observed across the cluster's events. Empty when all
		// events lacked actor_ref.
		actorRefs := make([]string, 0, len(c.actors))
		for a := range c.actors {
			actorRefs = append(actorRefs, a)
		}
		sort.Strings(actorRefs)

		sort.Slice(c.sourceHashes, func(i, j int) bool {
			return bytesLess(c.sourceHashes[i], c.sourceHashes[j])
		})

		out = append(out, &FormationCandidate{
			SignatureName:     s.Name(),
			HypothesisSubtype: s.Subtype(),
			ActorRefs:         actorRefs,
			SourceHashes:      c.sourceHashes,
			EvidenceCount:     uint32(len(c.sourceHashes)),
			// ConfidenceHint scales with eventCount (not actor count)
			// per event-centric ontology. Reuses shared
			// confidenceFromCount helper; caps at 0.9 per §3 N1.
			ConfidenceHint: confidenceFromCount(c.eventCount, threshold),
		})
	}
	stats.CandidatesEmitted = uint32(len(out))
	return &EvaluationResult{Candidates: out, Stats: stats}, nil
}
