package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/projection"
)

// handleHypothesisSummary serves GET /v1/hypotheses/summary per
// decision-log §0082. Mirrors the cmd/summarize-hypotheses CLI wire
// shape: top-level `combined` section + four per-subtype sections,
// each carrying `total` + `by_state` + `latencies`. Combined
// percentiles are computed from the union of per-subtype raw samples
// (exact, NOT approximated from per-subtype aggregates) per §0079.
//
// Per §0083, the aggregation helpers live in internal/projection
// and are shared with cmd/summarize-hypotheses.
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

	bcAgg := projection.AggregateBC(bcProjs)
	agAgg := projection.AggregateAG(agProjs)
	chAgg := projection.AggregateCH(chProjs)
	crAgg := projection.AggregateCR(crProjs)

	combinedCounts := projection.CombineCounts(
		bcAgg.Counts, agAgg.Counts, chAgg.Counts, crAgg.Counts,
	)
	combinedSamples := projection.CombineLatencySamples(
		bcAgg.Samples, agAgg.Samples, chAgg.Samples, crAgg.Samples,
	)

	output := summaryPerSubtype{
		Combined: projection.AggregateSection{
			StateCounts: combinedCounts,
			Latencies:   projection.AggregateAllLatencies(combinedSamples),
		},
		BehavioralCluster:  bcAgg.Section(),
		AutomationGroup:    agAgg.Section(),
		CampaignHypothesis: chAgg.Section(),
		CoordinationRing:   crAgg.Section(),
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(output)
}

type summaryPerSubtype struct {
	Combined           projection.AggregateSection `json:"combined"`
	BehavioralCluster  projection.AggregateSection `json:"behavioral_cluster"`
	AutomationGroup    projection.AggregateSection `json:"automation_group"`
	CampaignHypothesis projection.AggregateSection `json:"campaign_hypothesis"`
	CoordinationRing   projection.AggregateSection `json:"coordination_ring"`
}
