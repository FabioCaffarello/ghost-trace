// tcp_fingerprint_clustering_v1 is the second F3 signature per §0161:
// a Camada A canonical-aberta signature recognizing AutomationGroup
// formation candidates via clustering actors that share an identical
// TCP-layer fingerprint observed at the NetworkObservation surface.
// First NetworkSignature implementation; mirrors §0152's
// cdp_marker_density_v1 pattern on the NetworkObservation modality
// side per §0144 discriminated-union framing.
//
// Detection axis: multi-actor TCP-fingerprint identity is a high-
// confidence indicator of shared automation infrastructure —
// independent human-operated clients running varied OS / stack
// combinations produce a HETEROGENEOUS p0f signature distribution,
// while automated infrastructure (bots from the same host, behind
// the same NAT/proxy, or running identical container images) emits
// IDENTICAL TCP fingerprints. F1.NetworkObservation TcpFingerprint
// sub-modality (§0144) carries the p0f_signature string for direct
// equality-based clustering.
//
// MVP threshold: a cluster of >= 3 distinct actors sharing the same
// p0f_signature triggers a multi-actor AutomationGroup formation
// candidate. The threshold is intentionally conservative for
// inception-phase per §0023 + §0141 simpler-form discipline.
package signatures

import (
	"context"
	"fmt"
	"sort"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
)

// TCPFingerprintClusteringV1 implements the tcp_fingerprint_clustering_v1
// signature. Stateless; instances are reusable across invocations.
type TCPFingerprintClusteringV1 struct {
	// Threshold is the minimum distinct-actor count per p0f_signature
	// cluster for a candidate to be emitted. Default 3 at inception
	// phase per §0161 conservative-default discipline. 0 = use
	// default.
	Threshold uint32
}

// Name identifies the signature for instrumentation + versioning.
func (s *TCPFingerprintClusteringV1) Name() string { return "tcp_fingerprint_clustering_v1" }

// Subtype identifies the Cat III hypothesis subtype this signature
// produces candidates for: AutomationGroup (multi-actor identical-
// stack pattern is the AutomationGroup ontology shape per §0010
// Q2-A.2).
func (s *TCPFingerprintClusteringV1) Subtype() HypothesisSubtype {
	return HypothesisSubtypeAutomationGroup
}

// isNetworkSignature is the NetworkSignature marker.
func (s *TCPFingerprintClusteringV1) isNetworkSignature() {}

// effectiveThreshold returns Threshold or the default (3) when unset.
// Default 3 is higher than cdp_marker_density_v1's default 2 because
// the AutomationGroup ontology shape for this signature is multi-
// actor (cluster), not single-actor (per-actor density) — the
// threshold gates cluster size, not detection count.
func (s *TCPFingerprintClusteringV1) effectiveThreshold() uint32 {
	if s.Threshold == 0 {
		return 3
	}
	return s.Threshold
}

// EvaluateNetwork evaluates the signature against a slice of
// NetworkObservation records. Per-p0f_signature clustering: groups
// distinct actor_refs by identical p0f_signature string; emits a
// single multi-actor FormationCandidate per cluster whose distinct-
// actor count meets the threshold.
//
// Returns EvaluationResult with candidates in p0f_signature
// alphabetical order (for deterministic test fixtures + reproducible
// orchestrator behavior) + EvaluationStats with per-source + per-
// skip-reason counters per §0143 Sub-benchmark 1 instrumentation
// requirement.
//
// Skip semantics:
//   - nil observation: skipped (no counter increment).
//   - empty actor_ref: ObservationsSkippedNoActor++.
//   - non-tcp_fingerprint modality: ObservationsSkippedWrongModality++.
//   - empty p0f_signature string: ObservationsSkippedWrongModality++
//     (signature requires the canonical p0f form for clustering;
//     records lacking it cannot participate in equality-based
//     clustering).
func (s *TCPFingerprintClusteringV1) EvaluateNetwork(ctx context.Context, observations []*eventsv1.NetworkObservation) (*EvaluationResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	threshold := s.effectiveThreshold()
	stats := EvaluationStats{
		ObservationsScanned: uint32(len(observations)),
		PerCollector:        make(map[string]uint32),
	}

	// Per-p0f_signature cluster aggregation. Tracks distinct actors
	// (set semantics) + source hashes (list semantics, sorted at
	// emit time per §0139).
	type cluster struct {
		actors       map[string]struct{}
		sourceHashes [][]byte
	}
	perSignature := make(map[string]*cluster)
	distinctActors := make(map[string]struct{})

	for _, obs := range observations {
		if obs == nil {
			continue
		}
		if obs.CollectorRef != "" {
			stats.PerCollector[obs.CollectorRef]++
		}
		if obs.ActorRef == "" {
			stats.ObservationsSkippedNoActor++
			continue
		}
		tcpVariant := obs.GetTcpFingerprint()
		if tcpVariant == nil {
			stats.ObservationsSkippedWrongModality++
			continue
		}
		if tcpVariant.P0FSignature == "" {
			stats.ObservationsSkippedWrongModality++
			continue
		}
		distinctActors[obs.ActorRef] = struct{}{}

		c, ok := perSignature[tcpVariant.P0FSignature]
		if !ok {
			c = &cluster{actors: make(map[string]struct{})}
			perSignature[tcpVariant.P0FSignature] = c
		}
		c.actors[obs.ActorRef] = struct{}{}

		_, h, err := canonical.MarshalAndHash(obs)
		if err != nil {
			return nil, fmt.Errorf("tcp_fingerprint_clustering_v1: hash observation: %w", err)
		}
		hashCopy := make([]byte, len(h))
		copy(hashCopy, h[:])
		c.sourceHashes = append(c.sourceHashes, hashCopy)
	}
	stats.ActorsAggregated = uint32(len(distinctActors))

	// Sorted iteration for deterministic candidate order.
	signatureKeys := make([]string, 0, len(perSignature))
	for sig := range perSignature {
		signatureKeys = append(signatureKeys, sig)
	}
	sort.Strings(signatureKeys)

	out := make([]*FormationCandidate, 0)
	for _, sigKey := range signatureKeys {
		c := perSignature[sigKey]
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
			// ConfidenceHint scales linearly from threshold cluster
			// size to 5×threshold, capped at 0.9 per §3 N1 truth-
			// vs-structure boundary. Mirrors cdp_marker_density_v1's
			// confidenceFromCount discipline; input is distinct-actor
			// count (not detection count).
			ConfidenceHint: confidenceFromCount(uint32(len(c.actors)), threshold),
		})
	}
	stats.CandidatesEmitted = uint32(len(out))
	return &EvaluationResult{Candidates: out, Stats: stats}, nil
}
