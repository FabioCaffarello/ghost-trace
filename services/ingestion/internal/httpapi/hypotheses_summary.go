package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/projection"
)

// handleHypothesisSummary serves GET /v1/hypotheses/summary per
// decision-log §0082. Mirrors the cmd/summarize-hypotheses CLI wire
// shape: top-level `combined` section + four per-subtype sections,
// each carrying `total` + `by_state` + `latencies`. Combined
// percentiles are computed from the union of per-subtype raw samples
// (exact, NOT approximated from per-subtype aggregates) per §0079.
//
// Query parameters (all optional):
//   - after_ns, before_ns: inclusive bounds (Unix nanoseconds) on
//     the latest event_time of each projection; 0 disables.
//
// Response codes:
//   - 200 — JSON summary.
//   - 400 — invalid numeric / after_ns > before_ns.
//   - 405 — non-GET method.
//   - 503 — substrate not configured.
func (h *Handler) handleHypothesisSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.sub == nil {
		writeIngestError(w, http.StatusServiceUnavailable,
			"projection-read endpoints not configured (handler constructed without WithSubstrate)")
		return
	}

	q := r.URL.Query()
	afterNs, err := parseInt64Param(q, "after_ns")
	if err != nil {
		writeIngestError(w, http.StatusBadRequest, err.Error())
		return
	}
	beforeNs, err := parseInt64Param(q, "before_ns")
	if err != nil {
		writeIngestError(w, http.StatusBadRequest, err.Error())
		return
	}
	if afterNs < 0 {
		writeIngestError(w, http.StatusBadRequest, "after_ns must be non-negative")
		return
	}
	if beforeNs < 0 {
		writeIngestError(w, http.StatusBadRequest, "before_ns must be non-negative")
		return
	}
	if afterNs != 0 && beforeNs != 0 && afterNs > beforeNs {
		writeIngestError(w, http.StatusBadRequest,
			fmt.Sprintf("after_ns (%d) must not exceed before_ns (%d)", afterNs, beforeNs))
		return
	}

	ctx := r.Context()

	bcProjs, err := projection.ListHypotheses(ctx, h.sub, projection.ListOptions{
		TimeAfterNs:  afterNs,
		TimeBeforeNs: beforeNs,
	})
	if err != nil {
		writeIngestError(w, http.StatusInternalServerError, fmt.Sprintf("list BC: %v", err))
		return
	}
	agProjs, err := projection.ListAutomationGroups(ctx, h.sub, projection.AutomationGroupListOptions{
		TimeAfterNs:  afterNs,
		TimeBeforeNs: beforeNs,
	})
	if err != nil {
		writeIngestError(w, http.StatusInternalServerError, fmt.Sprintf("list AG: %v", err))
		return
	}
	chProjs, err := projection.ListCampaignHypotheses(ctx, h.sub, projection.CampaignHypothesisListOptions{
		TimeAfterNs:  afterNs,
		TimeBeforeNs: beforeNs,
	})
	if err != nil {
		writeIngestError(w, http.StatusInternalServerError, fmt.Sprintf("list CH: %v", err))
		return
	}
	crProjs, err := projection.ListCoordinationRings(ctx, h.sub, projection.CoordinationRingListOptions{
		TimeAfterNs:  afterNs,
		TimeBeforeNs: beforeNs,
	})
	if err != nil {
		writeIngestError(w, http.StatusInternalServerError, fmt.Sprintf("list CR: %v", err))
		return
	}

	bc := summaryBCAggregate(bcProjs)
	ag := summaryAGAggregate(agProjs)
	ch := summaryCHAggregate(chProjs)
	cr := summaryCRAggregate(crProjs)

	combinedCounts := summaryCombineCounts(bc.counts, ag.counts, ch.counts, cr.counts)
	combinedSamples := summaryCombineSamples(bc.samples, ag.samples, ch.samples, cr.samples)

	output := summaryPerSubtype{
		Combined: summaryAggregateSection{
			StateCounts: combinedCounts,
			Latencies:   summaryAggregateAllLatencies(combinedSamples),
		},
		BehavioralCluster:  bc.section(),
		AutomationGroup:    ag.section(),
		CampaignHypothesis: ch.section(),
		CoordinationRing:   cr.section(),
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(output)
}

// summaryAggregateSection is the wire shape of each section
// (combined + four per-subtype) in the GET /v1/hypotheses/summary
// JSON. Embeds StateCounts so `total` + `by_state` appear at the
// section's parent level (mirrors cmd/summarize-hypotheses wire
// shape exactly).
type summaryAggregateSection struct {
	projection.StateCounts
	Latencies summaryLatencyAggregates `json:"latencies"`
}

type summaryLatencyAggregates struct {
	FormationToFirstPromotionNs       summaryLatencyAggregate `json:"formation_to_first_promotion_ns"`
	LatestPromotionToLatestDemotionNs summaryLatencyAggregate `json:"latest_promotion_to_latest_demotion_ns"`
	FormationToDissolutionNs          summaryLatencyAggregate `json:"formation_to_dissolution_ns"`
}

type summaryLatencyAggregate struct {
	SampleCount int    `json:"sample_count"`
	MinNs       *int64 `json:"min_ns,omitempty"`
	P50Ns       *int64 `json:"p50_ns,omitempty"`
	P90Ns       *int64 `json:"p90_ns,omitempty"`
	MaxNs       *int64 `json:"max_ns,omitempty"`
}

type summaryLatencySamples struct {
	FormationToFirstPromotion       []int64
	LatestPromotionToLatestDemotion []int64
	FormationToDissolution          []int64
}

type summarySubtypeAggregate struct {
	counts  projection.StateCounts
	samples summaryLatencySamples
}

func (a summarySubtypeAggregate) section() summaryAggregateSection {
	return summaryAggregateSection{
		StateCounts: a.counts,
		Latencies:   summaryAggregateAllLatencies(a.samples),
	}
}

func summaryAggregateAllLatencies(s summaryLatencySamples) summaryLatencyAggregates {
	return summaryLatencyAggregates{
		FormationToFirstPromotionNs:       summaryAggregateLatencies(s.FormationToFirstPromotion),
		LatestPromotionToLatestDemotionNs: summaryAggregateLatencies(s.LatestPromotionToLatestDemotion),
		FormationToDissolutionNs:          summaryAggregateLatencies(s.FormationToDissolution),
	}
}

func summaryBCAggregate(projs []projection.HypothesisProjection) summarySubtypeAggregate {
	a := summarySubtypeAggregate{counts: summaryEmptyCounts()}
	for _, p := range projs {
		a.counts.Total++
		a.counts.ByState[p.State]++
		if p.FormationToFirstPromotionLatencyNs != nil {
			a.samples.FormationToFirstPromotion = append(a.samples.FormationToFirstPromotion, *p.FormationToFirstPromotionLatencyNs)
		}
		if p.LatestPromotionToLatestDemotionLatencyNs != nil {
			a.samples.LatestPromotionToLatestDemotion = append(a.samples.LatestPromotionToLatestDemotion, *p.LatestPromotionToLatestDemotionLatencyNs)
		}
		if p.FormationToDissolutionLatencyNs != nil {
			a.samples.FormationToDissolution = append(a.samples.FormationToDissolution, *p.FormationToDissolutionLatencyNs)
		}
	}
	return a
}

func summaryAGAggregate(projs []projection.AutomationGroupProjection) summarySubtypeAggregate {
	a := summarySubtypeAggregate{counts: summaryEmptyCounts()}
	for _, p := range projs {
		a.counts.Total++
		a.counts.ByState[p.State]++
		if p.FormationToFirstPromotionLatencyNs != nil {
			a.samples.FormationToFirstPromotion = append(a.samples.FormationToFirstPromotion, *p.FormationToFirstPromotionLatencyNs)
		}
		if p.LatestPromotionToLatestDemotionLatencyNs != nil {
			a.samples.LatestPromotionToLatestDemotion = append(a.samples.LatestPromotionToLatestDemotion, *p.LatestPromotionToLatestDemotionLatencyNs)
		}
		if p.FormationToDissolutionLatencyNs != nil {
			a.samples.FormationToDissolution = append(a.samples.FormationToDissolution, *p.FormationToDissolutionLatencyNs)
		}
	}
	return a
}

func summaryCHAggregate(projs []projection.CampaignHypothesisProjection) summarySubtypeAggregate {
	a := summarySubtypeAggregate{counts: summaryEmptyCounts()}
	for _, p := range projs {
		a.counts.Total++
		a.counts.ByState[p.State]++
		if p.FormationToFirstPromotionLatencyNs != nil {
			a.samples.FormationToFirstPromotion = append(a.samples.FormationToFirstPromotion, *p.FormationToFirstPromotionLatencyNs)
		}
		if p.LatestPromotionToLatestDemotionLatencyNs != nil {
			a.samples.LatestPromotionToLatestDemotion = append(a.samples.LatestPromotionToLatestDemotion, *p.LatestPromotionToLatestDemotionLatencyNs)
		}
		if p.FormationToDissolutionLatencyNs != nil {
			a.samples.FormationToDissolution = append(a.samples.FormationToDissolution, *p.FormationToDissolutionLatencyNs)
		}
	}
	return a
}

func summaryCRAggregate(projs []projection.CoordinationRingProjection) summarySubtypeAggregate {
	a := summarySubtypeAggregate{counts: summaryEmptyCounts()}
	for _, p := range projs {
		a.counts.Total++
		a.counts.ByState[p.State]++
		if p.FormationToFirstPromotionLatencyNs != nil {
			a.samples.FormationToFirstPromotion = append(a.samples.FormationToFirstPromotion, *p.FormationToFirstPromotionLatencyNs)
		}
		if p.LatestPromotionToLatestDemotionLatencyNs != nil {
			a.samples.LatestPromotionToLatestDemotion = append(a.samples.LatestPromotionToLatestDemotion, *p.LatestPromotionToLatestDemotionLatencyNs)
		}
		if p.FormationToDissolutionLatencyNs != nil {
			a.samples.FormationToDissolution = append(a.samples.FormationToDissolution, *p.FormationToDissolutionLatencyNs)
		}
	}
	return a
}

func summaryEmptyCounts() projection.StateCounts {
	return projection.StateCounts{
		ByState: map[projection.State]int{
			projection.StateForming:    0,
			projection.StatePromoted:   0,
			projection.StateDemoted:    0,
			projection.StateDissolved:  0,
			projection.StateMergedInto: 0,
			projection.StateSplitInto:  0,
		},
	}
}

func summaryAggregateLatencies(samples []int64) summaryLatencyAggregate {
	if len(samples) == 0 {
		return summaryLatencyAggregate{}
	}
	cp := append([]int64(nil), samples...)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })

	min := cp[0]
	max := cp[len(cp)-1]
	p50 := summaryPercentileNearestRank(cp, 50)
	p90 := summaryPercentileNearestRank(cp, 90)

	return summaryLatencyAggregate{
		SampleCount: len(cp),
		MinNs:       &min,
		P50Ns:       &p50,
		P90Ns:       &p90,
		MaxNs:       &max,
	}
}

func summaryPercentileNearestRank(sorted []int64, k int) int64 {
	n := len(sorted)
	idx := (k*n + 99) / 100 // ceil(k*n/100)
	idx--
	if idx < 0 {
		idx = 0
	}
	if idx >= n {
		idx = n - 1
	}
	return sorted[idx]
}

func summaryCombineCounts(parts ...projection.StateCounts) projection.StateCounts {
	combined := summaryEmptyCounts()
	for _, p := range parts {
		combined.Total += p.Total
		for state, count := range p.ByState {
			combined.ByState[state] += count
		}
	}
	return combined
}

// summaryCombineSamples returns the union of per-dimension samples
// across per-subtype summaryLatencySamples. The combined latency
// aggregate is computed from this union (NOT approximated from
// per-subtype aggregates) per §0079 — combined percentiles are exact.
func summaryCombineSamples(parts ...summaryLatencySamples) summaryLatencySamples {
	combined := summaryLatencySamples{}
	for _, p := range parts {
		combined.FormationToFirstPromotion = append(combined.FormationToFirstPromotion, p.FormationToFirstPromotion...)
		combined.LatestPromotionToLatestDemotion = append(combined.LatestPromotionToLatestDemotion, p.LatestPromotionToLatestDemotion...)
		combined.FormationToDissolution = append(combined.FormationToDissolution, p.FormationToDissolution...)
	}
	return combined
}

type summaryPerSubtype struct {
	Combined           summaryAggregateSection `json:"combined"`
	BehavioralCluster  summaryAggregateSection `json:"behavioral_cluster"`
	AutomationGroup    summaryAggregateSection `json:"automation_group"`
	CampaignHypothesis summaryAggregateSection `json:"campaign_hypothesis"`
	CoordinationRing   summaryAggregateSection `json:"coordination_ring"`
}
