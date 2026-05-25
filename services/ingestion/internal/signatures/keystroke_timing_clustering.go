// keystroke_timing_clustering_v1 is the first BehavioralSignature
// per decision-log §0174. First F3 signature emitting candidates of
// the BehavioralCluster subtype per §0010 Q2-A.2 (set of actors
// operating under a common underlying entity).
//
// Detection axis: identical quantized keystroke fingerprint across
// multiple distinct actors indicates same operator typing across
// accounts. KeystrokeInterval (flight_ns, dwell_ns) per §0146
// behavioral_keystroke_timing.proto carries timing distributions
// extractable from browser-SDK / native instrumentation; per the
// privacy-preserving design, the payload carries TIMINGS, not key
// codes — anti-bot signatures inspect cadence, not content.
//
// Quantization: each (flight_ns, dwell_ns) tuple is rounded to the
// nearest 50ms bucket (50_000_000 ns). 50ms balances over-clustering
// (too tolerant; coincidental clusters) vs under-clustering (too
// strict; natural variation in human typing prevents fingerprint
// matching). Inception-phase choice per §0174; empirical pressure
// will surface if adjustment needed.
//
// Cluster key form: comma-separated "f<flight_ms>d<dwell_ms>" tuples
// in chronological order. Operator-inspectable plain string per
// §0169 plain-string discipline. Example: "f50d100,f0d100,f100d150"
// for 3 keystrokes with quantized timings.
//
// MVP threshold: cluster of >= 3 distinct actors sharing the same
// quantized fingerprint triggers a multi-actor BehavioralCluster
// formation candidate. Conservative default per §0174 mirroring
// §0161 / §0169.
//
// Minimum keystrokes per observation: 3 intervals. Records with
// fewer skip at ObservationsSkippedWrongModality — single-keystroke
// "fingerprints" coincidentally collide too easily to be meaningful.
//
// Per §0168 scope: BehavioralSignature does NOT accept AttributionLookup
// parameter. BehavioralObservation carries actor_ref from client SDK
// directly; no Cat II attribution gap exists for this modality at
// inception phase.
package signatures

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
)

// keystrokeQuantizationBucketNs is the quantization bucket size in
// nanoseconds (50ms). Each (flight_ns, dwell_ns) tuple is rounded to
// the nearest multiple of this bucket before forming the cluster key.
const keystrokeQuantizationBucketNs = uint64(50_000_000)

// keystrokeMinIntervalsForFingerprint is the minimum keystroke
// interval count required for a record to participate in clustering.
// Records with fewer intervals carry insufficient signal for
// fingerprint formation.
const keystrokeMinIntervalsForFingerprint = 3

// KeystrokeTimingClusteringV1 implements the
// keystroke_timing_clustering_v1 signature. Stateless; reusable
// across invocations.
type KeystrokeTimingClusteringV1 struct {
	// Threshold is the minimum distinct-actor count per quantized
	// keystroke-fingerprint cluster for a candidate to be emitted.
	// Default 3 per §0174 conservative-default discipline (mirrors
	// §0161 + §0169 cluster-cardinality discipline).
	Threshold uint32
}

// Name identifies the signature for instrumentation + versioning.
func (s *KeystrokeTimingClusteringV1) Name() string { return "keystroke_timing_clustering_v1" }

// Subtype identifies the Cat III hypothesis subtype: BehavioralCluster
// (multi-actor shared-fingerprint pattern is the BehavioralCluster
// ontology shape per §0010 Q2-A.2 — "set of actors operating under a
// common underlying entity").
func (s *KeystrokeTimingClusteringV1) Subtype() HypothesisSubtype {
	return HypothesisSubtypeBehavioralCluster
}

// isBehavioralSignature is the BehavioralSignature marker.
func (s *KeystrokeTimingClusteringV1) isBehavioralSignature() {}

// effectiveThreshold returns Threshold or the default (3) when unset.
func (s *KeystrokeTimingClusteringV1) effectiveThreshold() uint32 {
	if s.Threshold == 0 {
		return 3
	}
	return s.Threshold
}

// EvaluateBehavioral evaluates the signature against a slice of
// BehavioralObservation records. Per-fingerprint clustering: groups
// distinct actor_refs by canonical quantized keystroke fingerprint;
// emits one multi-actor FormationCandidate per cluster meeting the
// threshold.
//
// Returns EvaluationResult with candidates in fingerprint-key
// alphabetical order (deterministic) + EvaluationStats with
// per-source + per-skip-reason counters per §0143 Sub-benchmark 1.
//
// Skip semantics:
//   - nil observation: silently skipped (no counter).
//   - empty actor_ref: ObservationsSkippedNoActor++.
//   - non-keystroke_timing modality: ObservationsSkippedWrongModality++.
//   - keystroke_timing record with fewer than
//     keystrokeMinIntervalsForFingerprint intervals:
//     ObservationsSkippedWrongModality++ (insufficient signal for
//     fingerprint formation).
func (s *KeystrokeTimingClusteringV1) EvaluateBehavioral(ctx context.Context, observations []*eventsv1.BehavioralObservation) (*EvaluationResult, error) {
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
	perFingerprint := make(map[string]*cluster)
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

		keystrokeVariant := obs.GetKeystrokeTiming()
		if keystrokeVariant == nil {
			stats.ObservationsSkippedWrongModality++
			continue
		}
		if len(keystrokeVariant.Intervals) < keystrokeMinIntervalsForFingerprint {
			stats.ObservationsSkippedWrongModality++
			continue
		}

		fingerprint := canonicalKeystrokeFingerprint(keystrokeVariant.Intervals)
		distinctActors[obs.ActorRef] = struct{}{}

		c, ok := perFingerprint[fingerprint]
		if !ok {
			c = &cluster{actors: make(map[string]struct{})}
			perFingerprint[fingerprint] = c
		}
		c.actors[obs.ActorRef] = struct{}{}

		_, h, err := canonical.MarshalAndHash(obs)
		if err != nil {
			return nil, fmt.Errorf("keystroke_timing_clustering_v1: hash observation: %w", err)
		}
		hashCopy := make([]byte, len(h))
		copy(hashCopy, h[:])
		c.sourceHashes = append(c.sourceHashes, hashCopy)
	}
	stats.ActorsAggregated = uint32(len(distinctActors))

	fingerprintKeys := make([]string, 0, len(perFingerprint))
	for fk := range perFingerprint {
		fingerprintKeys = append(fingerprintKeys, fk)
	}
	sort.Strings(fingerprintKeys)

	out := make([]*FormationCandidate, 0)
	for _, fk := range fingerprintKeys {
		c := perFingerprint[fk]
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

// canonicalKeystrokeFingerprint produces the deterministic cluster
// key from a sequence of KeystrokeInterval records. Each interval's
// (flight_ns, dwell_ns) tuple is quantized to the nearest
// keystrokeQuantizationBucketNs bucket, then rendered as "f<ms>d<ms>"
// using millisecond units for operator inspectability per §0169
// plain-string discipline.
//
// Sequence order is preserved (chronological per §0146); reordering
// would produce a different fingerprint, reflecting that typing
// rhythm is order-dependent.
func canonicalKeystrokeFingerprint(intervals []*eventsv1.KeystrokeInterval) string {
	if len(intervals) == 0 {
		return ""
	}
	parts := make([]string, len(intervals))
	for i, iv := range intervals {
		flightMs := quantizeToMs(iv.FlightNs)
		dwellMs := quantizeToMs(iv.DwellNs)
		parts[i] = fmt.Sprintf("f%dd%d", flightMs, dwellMs)
	}
	return strings.Join(parts, ",")
}

// quantizeToMs rounds a nanosecond value to the nearest
// keystrokeQuantizationBucketNs (50ms) bucket and returns the bucket
// midpoint in milliseconds. Round-half-up (the standard rounding
// convention; produces deterministic output identical to int64
// arithmetic across platforms).
func quantizeToMs(ns uint64) uint64 {
	bucket := keystrokeQuantizationBucketNs
	half := bucket / 2
	rounded := ((ns + half) / bucket) * bucket
	return rounded / 1_000_000
}
