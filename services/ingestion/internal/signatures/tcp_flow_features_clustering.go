// tcp_flow_features_clustering_v1 is the third F3 signature per
// decision-log §0169. Alternative to §0161's
// tcp_fingerprint_clustering_v1: clusters actors by the partial
// fingerprint constructible from fields CICFlowMeter DOES preserve
// — TCP flag counts (canonicalized) + initial window size — rather
// than the p0f canonical form CICFlowMeter does NOT preserve.
//
// Closes §0162 gap (2) (CIC-IDS NetworkObservation records emitted
// without p0f_signature) by meeting the data where it is, NOT by
// fabricating canonical form from incomplete inputs (option (b) per
// user direction; preserves §3 N1 truth-vs-structure boundary —
// no synthesized canonical form).
//
// Detection axis: identical TCP flow-feature vector across multiple
// distinct actors indicates shared automation infrastructure (bots
// from same OS/stack, behind same NAT/proxy, or running identical
// container images). The signal is weaker than p0f canonical
// fingerprinting (TCP options carry more entropy than flag counts +
// window) but is empirically present in CIC-IDS-style flow-level
// telemetry where p0f canonical form is not preserved.
//
// Consistent with §0161's AutomationGroup subtype framing: multi-
// actor identical-stack pattern is the AutomationGroup ontology
// shape per §0010 Q2-A.2.
//
// Per §0168 Decision A.1 (signature-aware Cat II consumption):
// EvaluateNetwork consumes the AttributionLookup parameter for Cat I
// records lacking declared actor_ref + threads Cat II derivation
// hash into candidate SourceHashes alongside the Cat I observation
// hash (preserves §2.3 chain).
//
// Combined with §0168's DerivedActorAttribution + network_5tuple_
// actor_v1, this signature makes the CIC-IDS path F3-reachable
// end-to-end — gap (1) closed by §0168 (actor_ref derived from
// 5-tuple); gap (2) closed by this signature (clusters by available
// features instead of requiring p0f).
package signatures

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
)

// TCPFlowFeaturesClusteringV1 implements the
// tcp_flow_features_clustering_v1 signature. Stateless; reusable
// across invocations.
type TCPFlowFeaturesClusteringV1 struct {
	// Threshold is the minimum distinct-actor count per flow-feature
	// cluster for a candidate to be emitted. Default 3 per §0169
	// conservative-default discipline (mirrors §0161 threshold for
	// cluster-cardinality-keyed signatures).
	Threshold uint32
}

// Name identifies the signature for instrumentation + versioning.
func (s *TCPFlowFeaturesClusteringV1) Name() string { return "tcp_flow_features_clustering_v1" }

// Subtype identifies the Cat III hypothesis subtype: AutomationGroup
// (multi-actor identical-stack pattern per §0010 Q2-A.2; consistent
// with §0161's framing).
func (s *TCPFlowFeaturesClusteringV1) Subtype() HypothesisSubtype {
	return HypothesisSubtypeAutomationGroup
}

// isNetworkSignature is the NetworkSignature marker.
func (s *TCPFlowFeaturesClusteringV1) isNetworkSignature() {}

// effectiveThreshold returns Threshold or the default (3) when unset.
// Mirrors §0161's default-3 discipline for cluster-cardinality
// signatures (versus per-actor density signatures like §0152 cdp_
// marker_density_v1 which use default 2 because per-actor accumulation
// is faster).
func (s *TCPFlowFeaturesClusteringV1) effectiveThreshold() uint32 {
	if s.Threshold == 0 {
		return 3
	}
	return s.Threshold
}

// EvaluateNetwork evaluates the signature against a slice of
// NetworkObservation records. Per-feature-vector clustering: groups
// distinct actor_refs by canonical (flag_counts, window_size) tuple;
// emits one multi-actor FormationCandidate per cluster meeting the
// threshold.
//
// Returns EvaluationResult with candidates in feature-key
// alphabetical order (deterministic) + EvaluationStats with
// per-source + per-skip-reason counters per §0143 Sub-benchmark 1.
//
// Skip semantics:
//   - nil observation: silently skipped (no counter).
//   - empty actor_ref AND no AttributionLookup-derived actor:
//     ObservationsSkippedNoActor++ (per §0168 A.1: lookup fills gap
//     when present).
//   - non-tcp_fingerprint modality: ObservationsSkippedWrongModality++.
//   - tcp_fingerprint with empty flags_sequence AND window_size == 0:
//     ObservationsSkippedWrongModality++ (no features to cluster
//     on; meaningfully indistinguishable from no fingerprint info).
//
// Per §0168 A.1: AttributionLookup parameter handled identically to
// §0161's tcp_fingerprint_clustering_v1 — declared-precedes-derived;
// Cat II derivation hash threaded into candidate SourceHashes when
// consumed.
func (s *TCPFlowFeaturesClusteringV1) EvaluateNetwork(ctx context.Context, observations []*eventsv1.NetworkObservation, attribution AttributionLookup) (*EvaluationResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	threshold := s.effectiveThreshold()
	stats := EvaluationStats{
		ObservationsScanned: uint32(len(observations)),
		PerCollector:        make(map[string]uint32),
	}

	type cluster struct {
		actors       map[string]struct{}
		sourceHashes [][]byte
	}
	perFeature := make(map[string]*cluster)
	distinctActors := make(map[string]struct{})

	for _, obs := range observations {
		if obs == nil {
			continue
		}
		if obs.CollectorRef != "" {
			stats.PerCollector[obs.CollectorRef]++
		}

		_, obsHash, err := canonical.MarshalAndHash(obs)
		if err != nil {
			return nil, fmt.Errorf("tcp_flow_features_clustering_v1: hash observation: %w", err)
		}

		// Resolve effective actor per §0168 A.1 (declared-precedes-
		// derived; Cat II fills gap when present).
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

		tcpVariant := obs.GetTcpFingerprint()
		if tcpVariant == nil {
			stats.ObservationsSkippedWrongModality++
			continue
		}

		// Skip records with no flow-feature signal: empty
		// flags_sequence AND window_size 0 == no clusterable features.
		if len(tcpVariant.FlagsSequence) == 0 && tcpVariant.WindowSize == 0 {
			stats.ObservationsSkippedWrongModality++
			continue
		}

		featureKey := canonicalFlowFeatureKey(tcpVariant.FlagsSequence, tcpVariant.WindowSize)
		distinctActors[effectiveActor] = struct{}{}

		c, ok := perFeature[featureKey]
		if !ok {
			c = &cluster{actors: make(map[string]struct{})}
			perFeature[featureKey] = c
		}
		c.actors[effectiveActor] = struct{}{}

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

	featureKeys := make([]string, 0, len(perFeature))
	for fk := range perFeature {
		featureKeys = append(featureKeys, fk)
	}
	sort.Strings(featureKeys)

	out := make([]*FormationCandidate, 0)
	for _, fk := range featureKeys {
		c := perFeature[fk]
		if uint32(len(c.actors)) < threshold {
			continue
		}
		stats.ActorsAboveThreshold += uint32(len(c.actors))

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
			ConfidenceHint:    confidenceFromCount(uint32(len(c.actors)), threshold),
		})
	}
	stats.CandidatesEmitted = uint32(len(out))
	return &EvaluationResult{Candidates: out, Stats: stats}, nil
}

// canonicalFlowFeatureKey produces the deterministic cluster key from
// the flow-feature vector. Format: "flags=<sequence>;window=<value>"
// where <sequence> is the comma-separated decimal flag-byte sequence
// from FlagsSequence. Operator-inspectable; not hashed.
//
// Per §0169 + §0168 plain-string-key discipline: short enough to
// display directly in operator output. The cluster key IS the
// signature of the cluster.
func canonicalFlowFeatureKey(flagsSeq []uint32, windowSize uint32) string {
	if len(flagsSeq) == 0 {
		return fmt.Sprintf("flags=;window=%d", windowSize)
	}
	parts := make([]string, len(flagsSeq))
	for i, b := range flagsSeq {
		parts[i] = fmt.Sprintf("%d", b)
	}
	return fmt.Sprintf("flags=%s;window=%d", strings.Join(parts, ","), windowSize)
}
