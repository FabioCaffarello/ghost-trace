// cdp_marker_density_v1 is the first F3 signature per §0152: a
// Camada A canonical-aberta signature recognizing AutomationGroup
// formation candidates via Chrome DevTools Protocol marker density
// observed at the BrowserObservation surface.
//
// Detection axis: high-confidence anti-bot indicator — automation
// frameworks (Selenium / Puppeteer / Playwright) leave artifacts in
// the browser runtime that are absent from human-operated browsers.
// Examples: `navigator.webdriver=true`, `$cdc_*` runtime variables,
// missing `chrome.runtime`, missing `chrome.csi`. F1.BrowserObservation
// CDP marker sub-modality (§0151) carries these as a list of
// detected markers per actor per observation window.
//
// MVP threshold: an actor with detection_count >= 2 across the
// observation window triggers a single-actor AutomationGroup
// formation candidate. The threshold is intentionally conservative
// for inception-phase per §0023 + §0141 simpler-form discipline;
// reversal conditions documented at §0152.
package signatures

import (
	"context"
	"fmt"
	"sort"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
)

// CDPMarkerDensityV1 implements the cdp_marker_density_v1 signature.
// Stateless; instances are reusable across invocations.
type CDPMarkerDensityV1 struct {
	// Threshold is the minimum aggregate detection_count across the
	// observation window for a candidate to be emitted. Default 2
	// at inception phase per §0152 conservative-default discipline.
	// 0 = use default.
	Threshold uint32
}

// Name identifies the signature for instrumentation + versioning.
func (s *CDPMarkerDensityV1) Name() string { return "cdp_marker_density_v1" }

// Subtype identifies the Cat III hypothesis subtype this signature
// produces candidates for: AutomationGroup.
func (s *CDPMarkerDensityV1) Subtype() HypothesisSubtype {
	return HypothesisSubtypeAutomationGroup
}

// isBrowserSignature is the BrowserSignature marker.
func (s *CDPMarkerDensityV1) isBrowserSignature() {}

// effectiveThreshold returns Threshold or the default (2) when unset.
func (s *CDPMarkerDensityV1) effectiveThreshold() uint32 {
	if s.Threshold == 0 {
		return 2
	}
	return s.Threshold
}

// EvaluateBrowser evaluates the signature against a slice of
// BrowserObservation records. Per-actor aggregation: sums
// detection_count across all CDP marker observations carried by
// records sharing the same actor_ref; emits a FormationCandidate
// per actor whose aggregate count meets the threshold.
//
// Returns EvaluationResult with candidates in actor_ref alphabetical
// order (for deterministic test fixtures + reproducible orchestrator
// behavior) + EvaluationStats with per-source + per-skip-reason
// counters per §0143 Sub-benchmark 1 instrumentation requirement.
func (s *CDPMarkerDensityV1) EvaluateBrowser(ctx context.Context, observations []*eventsv1.BrowserObservation) (*EvaluationResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	threshold := s.effectiveThreshold()
	stats := EvaluationStats{
		ObservationsScanned: uint32(len(observations)),
		PerCollector:        make(map[string]uint32),
	}

	// Per-actor aggregation. Skips observations without cdp_marker
	// modality (other modalities are out-of-scope for this
	// signature) and observations without actor_ref (unattributed
	// observations cannot anchor a hypothesis per the ontology's
	// actor-set definition for AutomationGroup).
	type aggregate struct {
		detectionCount uint32
		sourceHashes   [][]byte
	}
	perActor := make(map[string]*aggregate)
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
		cdpVariant := obs.GetCdpMarker()
		if cdpVariant == nil {
			stats.ObservationsSkippedWrongModality++
			continue
		}
		agg, ok := perActor[obs.ActorRef]
		if !ok {
			agg = &aggregate{}
			perActor[obs.ActorRef] = agg
		}
		agg.detectionCount += cdpVariant.DetectionCount

		// Hash the observation itself for source_event_hashes
		// commitment per §0139 hash-list discipline.
		_, h, err := canonical.MarshalAndHash(obs)
		if err != nil {
			return nil, fmt.Errorf("cdp_marker_density_v1: hash observation: %w", err)
		}
		hashCopy := make([]byte, len(h))
		copy(hashCopy, h[:])
		agg.sourceHashes = append(agg.sourceHashes, hashCopy)
	}
	stats.ActorsAggregated = uint32(len(perActor))

	// Sorted iteration for deterministic candidate order.
	actors := make([]string, 0, len(perActor))
	for actor := range perActor {
		actors = append(actors, actor)
	}
	sort.Strings(actors)

	out := make([]*FormationCandidate, 0)
	for _, actor := range actors {
		agg := perActor[actor]
		if agg.detectionCount < threshold {
			continue
		}
		stats.ActorsAboveThreshold++
		// Sort source hashes ascending per §0139 hash-list element-
		// shape discipline (substrate commit-time will enforce this
		// anyway; surfacing sorted at candidate-emit time eases
		// orchestrator threading).
		sort.Slice(agg.sourceHashes, func(i, j int) bool {
			return bytesLess(agg.sourceHashes[i], agg.sourceHashes[j])
		})
		out = append(out, &FormationCandidate{
			SignatureName:     s.Name(),
			HypothesisSubtype: s.Subtype(),
			ActorRefs:         []string{actor},
			SourceHashes:      agg.sourceHashes,
			EvidenceCount:     uint32(len(agg.sourceHashes)),
			// ConfidenceHint scales linearly from threshold to 5×
			// threshold, capped at 0.9 (never claims certainty per
			// §3 N1 truth-vs-structure boundary). Producer-side hint
			// only; orchestrator commits the authoritative confidence.
			ConfidenceHint: confidenceFromCount(agg.detectionCount, threshold),
		})
	}
	stats.CandidatesEmitted = uint32(len(out))
	return &EvaluationResult{Candidates: out, Stats: stats}, nil
}

// bytesLess is the canonical hash-list ordering predicate per §0139:
// ascending lexicographic order on the BLAKE3 byte sequence.
func bytesLess(a, b []byte) bool {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return len(a) < len(b)
}

// confidenceFromCount produces an advisory confidence hint in [0, 1].
// Linear from threshold (0.5) to 5×threshold (0.9), capped at 0.9
// per §3 N1 (substrate never asserts truth; confidence is bounded
// strictly below 1.0 to reflect structural inferential commitment).
func confidenceFromCount(count, threshold uint32) float64 {
	if count < threshold {
		return 0.0
	}
	upper := uint32(5) * threshold
	if count >= upper {
		return 0.9
	}
	// Linear from 0.5 at threshold to 0.9 at upper.
	span := float64(upper - threshold)
	pos := float64(count - threshold)
	return 0.5 + (pos/span)*0.4
}
