package httpapi

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/projection"
)

// handleHypothesisList serves GET /v1/hypotheses per decision-log
// §0081. Mirrors the cmd/list-hypotheses CLI wire shape: returns a
// JSON array of projection summaries with optional subtype/state/
// time-window/limit/offset filters.
//
// Query parameters (all optional):
//   - subtype: behavioral_cluster | automation_group |
//     campaign_hypothesis | coordination_ring (empty = all subtypes).
//   - state: forming | promoted | demoted | dissolved | merged_into |
//     split_into (empty = all states).
//   - after_ns, before_ns: inclusive bounds (Unix nanoseconds) on the
//     latest event_time of each projection; 0 disables the bound.
//   - limit: cap on number of entries returned (0 = unbounded).
//   - offset: skip the first N entries (0 = start at the first).
//
// Response codes:
//   - 200 — JSON array of entries (possibly empty).
//   - 400 — invalid query parameter (unknown subtype/state, negative
//     numeric, after_ns > before_ns).
//   - 405 — non-GET method.
//   - 503 — substrate not configured.
func (h *Handler) handleHypothesisList(w http.ResponseWriter, r *http.Request) {
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
	subtypeFilter := q.Get("subtype")
	stateFilter := q.Get("state")

	if subtypeFilter != "" &&
		subtypeFilter != "behavioral_cluster" &&
		subtypeFilter != "automation_group" &&
		subtypeFilter != "campaign_hypothesis" &&
		subtypeFilter != "coordination_ring" {
		writeIngestError(w, http.StatusBadRequest,
			fmt.Sprintf("subtype %q is not one of: behavioral_cluster, automation_group, campaign_hypothesis, coordination_ring", subtypeFilter))
		return
	}
	if stateFilter != "" && !isValidState(stateFilter) {
		writeIngestError(w, http.StatusBadRequest,
			fmt.Sprintf("state %q is not one of: forming, promoted, demoted, dissolved, merged_into, split_into", stateFilter))
		return
	}

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
	limit, err := parseIntParam(q, "limit")
	if err != nil {
		writeIngestError(w, http.StatusBadRequest, err.Error())
		return
	}
	offset, err := parseIntParam(q, "offset")
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
	if limit < 0 {
		writeIngestError(w, http.StatusBadRequest, "limit must be non-negative")
		return
	}
	if offset < 0 {
		writeIngestError(w, http.StatusBadRequest, "offset must be non-negative")
		return
	}

	ctx := r.Context()
	out := []listEntry{}

	state := projection.State(stateFilter)

	if subtypeFilter == "" || subtypeFilter == "behavioral_cluster" {
		bc, err := projection.ListHypotheses(ctx, h.sub, projection.ListOptions{
			StateFilter:  state,
			TimeAfterNs:  afterNs,
			TimeBeforeNs: beforeNs,
		})
		if err != nil {
			writeIngestError(w, http.StatusInternalServerError, fmt.Sprintf("list BC: %v", err))
			return
		}
		for _, p := range bc {
			out = append(out, buildBCListEntry(p))
		}
	}
	if subtypeFilter == "" || subtypeFilter == "automation_group" {
		ag, err := projection.ListAutomationGroups(ctx, h.sub, projection.AutomationGroupListOptions{
			StateFilter:  state,
			TimeAfterNs:  afterNs,
			TimeBeforeNs: beforeNs,
		})
		if err != nil {
			writeIngestError(w, http.StatusInternalServerError, fmt.Sprintf("list AG: %v", err))
			return
		}
		for _, p := range ag {
			out = append(out, buildAGListEntry(p))
		}
	}
	if subtypeFilter == "" || subtypeFilter == "campaign_hypothesis" {
		ch, err := projection.ListCampaignHypotheses(ctx, h.sub, projection.CampaignHypothesisListOptions{
			StateFilter:  state,
			TimeAfterNs:  afterNs,
			TimeBeforeNs: beforeNs,
		})
		if err != nil {
			writeIngestError(w, http.StatusInternalServerError, fmt.Sprintf("list CH: %v", err))
			return
		}
		for _, p := range ch {
			out = append(out, buildCHListEntry(p))
		}
	}
	if subtypeFilter == "" || subtypeFilter == "coordination_ring" {
		cr, err := projection.ListCoordinationRings(ctx, h.sub, projection.CoordinationRingListOptions{
			StateFilter:  state,
			TimeAfterNs:  afterNs,
			TimeBeforeNs: beforeNs,
		})
		if err != nil {
			writeIngestError(w, http.StatusInternalServerError, fmt.Sprintf("list CR: %v", err))
			return
		}
		for _, p := range cr {
			out = append(out, buildCRListEntry(p))
		}
	}

	// Paging across the combined cross-subtype slice. Same semantic as
	// the cmd/list-hypotheses CLI: limit/offset apply after subtype +
	// state filtering, on the concatenated ordering (per-subtype
	// internal order preserved within each subtype's segment).
	if offset > 0 {
		if offset >= len(out) {
			out = nil
		} else {
			out = out[offset:]
		}
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
}

func parseInt64Param(q map[string][]string, key string) (int64, error) {
	v := ""
	if values, ok := q[key]; ok && len(values) > 0 {
		v = values[0]
	}
	if v == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a base-10 integer; got %q", key, v)
	}
	return n, nil
}

func parseIntParam(q map[string][]string, key string) (int, error) {
	n, err := parseInt64Param(q, key)
	if err != nil {
		return 0, err
	}
	return int(n), nil
}

func isValidState(s string) bool {
	switch projection.State(s) {
	case projection.StateForming, projection.StatePromoted, projection.StateDemoted,
		projection.StateDissolved, projection.StateMergedInto, projection.StateSplitInto:
		return true
	}
	return false
}

// listEntry is the wire shape of every entry in the GET /v1/hypotheses
// JSON array. Mirrors the cmd/list-hypotheses CLI entry shape.
type listEntry struct {
	Subtype          string                `json:"subtype"`
	FormationHash    string                `json:"formation_event_hash"`
	State            string                `json:"state"`
	LatestPromotion  *promotionView        `json:"latest_promotion,omitempty"`
	LatestDemotion   *demotionView         `json:"latest_demotion,omitempty"`
	Dissolution      *dissolutionView      `json:"dissolution,omitempty"`
	MergedInto       *mergeView            `json:"merged_into,omitempty"`
	SplitInto        *splitView            `json:"split_into,omitempty"`
	LifecycleEntries int                   `json:"lifecycle_event_count"`
	Latencies        stateLatencyView      `json:"latencies"`
}

func buildBCListEntry(p projection.HypothesisProjection) listEntry {
	e := listEntry{
		Subtype:          "behavioral_cluster",
		FormationHash:    hex.EncodeToString(p.FormationHash[:]),
		State:            string(p.State),
		LifecycleEntries: len(p.LifecycleHistory),
	}
	if p.LatestPromotion != nil {
		e.LatestPromotion = &promotionView{
			PromotedAt:     p.LatestPromotion.PromotedAt,
			CadenceSeconds: p.LatestPromotion.CadenceSeconds,
			Reason:         p.LatestPromotion.Reason,
		}
	}
	if p.LatestDemotion != nil {
		e.LatestDemotion = &demotionView{
			DemotedAt: p.LatestDemotion.DemotedAt,
			Reason:    p.LatestDemotion.Reason,
		}
	}
	if p.Dissolution != nil {
		e.Dissolution = &dissolutionView{
			DissolvedAt: p.Dissolution.DissolvedAt,
			Reason:      p.Dissolution.Reason,
		}
	}
	if p.MergedInto != nil {
		ants := make([]string, len(p.MergedInto.AntecedentFormationEventHashes))
		for i, h := range p.MergedInto.AntecedentFormationEventHashes {
			ants[i] = hex.EncodeToString(h)
		}
		e.MergedInto = &mergeView{
			MergedAt:              p.MergedInto.MergedAt,
			ProducedFormationHash: hex.EncodeToString(p.MergedInto.ProducedFormationEventHash),
			AntecedentHashes:      ants,
			Reason:                p.MergedInto.Reason,
		}
	}
	if p.SplitInto != nil {
		succs := make([]string, len(p.SplitInto.SuccessorFormationEventHashes))
		for i, h := range p.SplitInto.SuccessorFormationEventHashes {
			succs[i] = hex.EncodeToString(h)
		}
		e.SplitInto = &splitView{
			SplitAt:         p.SplitInto.SplitAt,
			SuccessorHashes: succs,
			Reason:          p.SplitInto.Reason,
		}
	}
	e.Latencies = stateLatencyView{
		FormationToFirstPromotionNs:       p.FormationToFirstPromotionLatencyNs,
		LatestPromotionToLatestDemotionNs: p.LatestPromotionToLatestDemotionLatencyNs,
		FormationToDissolutionNs:          p.FormationToDissolutionLatencyNs,
	}
	return e
}

func buildAGListEntry(p projection.AutomationGroupProjection) listEntry {
	e := listEntry{
		Subtype:          "automation_group",
		FormationHash:    hex.EncodeToString(p.FormationHash[:]),
		State:            string(p.State),
		LifecycleEntries: len(p.LifecycleHistory),
	}
	if p.LatestPromotion != nil {
		e.LatestPromotion = &promotionView{
			PromotedAt:     p.LatestPromotion.PromotedAt,
			CadenceSeconds: p.LatestPromotion.CadenceSeconds,
			Reason:         p.LatestPromotion.Reason,
		}
	}
	if p.LatestDemotion != nil {
		e.LatestDemotion = &demotionView{
			DemotedAt: p.LatestDemotion.DemotedAt,
			Reason:    p.LatestDemotion.Reason,
		}
	}
	if p.Dissolution != nil {
		e.Dissolution = &dissolutionView{
			DissolvedAt: p.Dissolution.DissolvedAt,
			Reason:      p.Dissolution.Reason,
		}
	}
	if p.MergedInto != nil {
		ants := make([]string, len(p.MergedInto.AntecedentFormationEventHashes))
		for i, h := range p.MergedInto.AntecedentFormationEventHashes {
			ants[i] = hex.EncodeToString(h)
		}
		e.MergedInto = &mergeView{
			MergedAt:              p.MergedInto.MergedAt,
			ProducedFormationHash: hex.EncodeToString(p.MergedInto.ProducedFormationEventHash),
			AntecedentHashes:      ants,
			Reason:                p.MergedInto.Reason,
		}
	}
	if p.SplitInto != nil {
		succs := make([]string, len(p.SplitInto.SuccessorFormationEventHashes))
		for i, h := range p.SplitInto.SuccessorFormationEventHashes {
			succs[i] = hex.EncodeToString(h)
		}
		e.SplitInto = &splitView{
			SplitAt:         p.SplitInto.SplitAt,
			SuccessorHashes: succs,
			Reason:          p.SplitInto.Reason,
		}
	}
	e.Latencies = stateLatencyView{
		FormationToFirstPromotionNs:       p.FormationToFirstPromotionLatencyNs,
		LatestPromotionToLatestDemotionNs: p.LatestPromotionToLatestDemotionLatencyNs,
		FormationToDissolutionNs:          p.FormationToDissolutionLatencyNs,
	}
	return e
}

func buildCHListEntry(p projection.CampaignHypothesisProjection) listEntry {
	e := listEntry{
		Subtype:          "campaign_hypothesis",
		FormationHash:    hex.EncodeToString(p.FormationHash[:]),
		State:            string(p.State),
		LifecycleEntries: len(p.LifecycleHistory),
	}
	if p.LatestPromotion != nil {
		e.LatestPromotion = &promotionView{
			PromotedAt:     p.LatestPromotion.PromotedAt,
			CadenceSeconds: p.LatestPromotion.CadenceSeconds,
			Reason:         p.LatestPromotion.Reason,
		}
	}
	if p.LatestDemotion != nil {
		e.LatestDemotion = &demotionView{
			DemotedAt: p.LatestDemotion.DemotedAt,
			Reason:    p.LatestDemotion.Reason,
		}
	}
	if p.Dissolution != nil {
		e.Dissolution = &dissolutionView{
			DissolvedAt: p.Dissolution.DissolvedAt,
			Reason:      p.Dissolution.Reason,
		}
	}
	if p.MergedInto != nil {
		ants := make([]string, len(p.MergedInto.AntecedentFormationEventHashes))
		for i, h := range p.MergedInto.AntecedentFormationEventHashes {
			ants[i] = hex.EncodeToString(h)
		}
		e.MergedInto = &mergeView{
			MergedAt:              p.MergedInto.MergedAt,
			ProducedFormationHash: hex.EncodeToString(p.MergedInto.ProducedFormationEventHash),
			AntecedentHashes:      ants,
			Reason:                p.MergedInto.Reason,
		}
	}
	if p.SplitInto != nil {
		succs := make([]string, len(p.SplitInto.SuccessorFormationEventHashes))
		for i, h := range p.SplitInto.SuccessorFormationEventHashes {
			succs[i] = hex.EncodeToString(h)
		}
		e.SplitInto = &splitView{
			SplitAt:         p.SplitInto.SplitAt,
			SuccessorHashes: succs,
			Reason:          p.SplitInto.Reason,
		}
	}
	e.Latencies = stateLatencyView{
		FormationToFirstPromotionNs:       p.FormationToFirstPromotionLatencyNs,
		LatestPromotionToLatestDemotionNs: p.LatestPromotionToLatestDemotionLatencyNs,
		FormationToDissolutionNs:          p.FormationToDissolutionLatencyNs,
	}
	return e
}

func buildCRListEntry(p projection.CoordinationRingProjection) listEntry {
	e := listEntry{
		Subtype:          "coordination_ring",
		FormationHash:    hex.EncodeToString(p.FormationHash[:]),
		State:            string(p.State),
		LifecycleEntries: len(p.LifecycleHistory),
	}
	if p.LatestPromotion != nil {
		e.LatestPromotion = &promotionView{
			PromotedAt:     p.LatestPromotion.PromotedAt,
			CadenceSeconds: p.LatestPromotion.CadenceSeconds,
			Reason:         p.LatestPromotion.Reason,
		}
	}
	if p.LatestDemotion != nil {
		e.LatestDemotion = &demotionView{
			DemotedAt: p.LatestDemotion.DemotedAt,
			Reason:    p.LatestDemotion.Reason,
		}
	}
	if p.Dissolution != nil {
		e.Dissolution = &dissolutionView{
			DissolvedAt: p.Dissolution.DissolvedAt,
			Reason:      p.Dissolution.Reason,
		}
	}
	if p.MergedInto != nil {
		ants := make([]string, len(p.MergedInto.AntecedentFormationEventHashes))
		for i, h := range p.MergedInto.AntecedentFormationEventHashes {
			ants[i] = hex.EncodeToString(h)
		}
		e.MergedInto = &mergeView{
			MergedAt:              p.MergedInto.MergedAt,
			ProducedFormationHash: hex.EncodeToString(p.MergedInto.ProducedFormationEventHash),
			AntecedentHashes:      ants,
			Reason:                p.MergedInto.Reason,
		}
	}
	if p.SplitInto != nil {
		succs := make([]string, len(p.SplitInto.SuccessorFormationEventHashes))
		for i, h := range p.SplitInto.SuccessorFormationEventHashes {
			succs[i] = hex.EncodeToString(h)
		}
		e.SplitInto = &splitView{
			SplitAt:         p.SplitInto.SplitAt,
			SuccessorHashes: succs,
			Reason:          p.SplitInto.Reason,
		}
	}
	e.Latencies = stateLatencyView{
		FormationToFirstPromotionNs:       p.FormationToFirstPromotionLatencyNs,
		LatestPromotionToLatestDemotionNs: p.LatestPromotionToLatestDemotionLatencyNs,
		FormationToDissolutionNs:          p.FormationToDissolutionLatencyNs,
	}
	return e
}
