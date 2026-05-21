package httpapi

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/projection"
)

// handleHypothesisState serves GET /v1/hypotheses/state per
// decision-log §0080. Mirrors the cmd/hypothesis-state CLI wire
// shape: auto-detects the formation's Cat III subtype via the
// substrate row's message_type, dispatches to the appropriate
// per-subtype projection, returns the projection JSON.
//
// Query parameters:
//   - formation_event_hash (REQUIRED) — hex-encoded BLAKE3-256
//     content-hash of a Cat III formation event (BehavioralCluster,
//     AutomationGroup, CampaignHypothesis, or CoordinationRing).
//
// Response codes:
//   - 200 — projection JSON in response body (subtype, formation hash,
//     state, latest_promotion/etc., lifecycle_history, latencies).
//   - 400 — missing parameter, invalid hex, wrong hash length.
//   - 404 — formation hash not in substrate, OR target hash resolves
//     to a non-formation row (cross-subtype rejection / wrong message
//     type).
//   - 405 — non-GET method.
//   - 503 — substrate not configured on this handler (the
//     WithSubstrate option was not supplied at construction).
func (h *Handler) handleHypothesisState(w http.ResponseWriter, r *http.Request) {
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

	formationHex := r.URL.Query().Get("formation_event_hash")
	if formationHex == "" {
		writeIngestError(w, http.StatusBadRequest,
			"missing required query parameter: formation_event_hash")
		return
	}
	raw, err := hex.DecodeString(formationHex)
	if err != nil {
		writeIngestError(w, http.StatusBadRequest,
			fmt.Sprintf("decode formation_event_hash: %v", err))
		return
	}
	if len(raw) != 32 {
		writeIngestError(w, http.StatusBadRequest,
			fmt.Sprintf("formation_event_hash must be 32 bytes (64 hex chars); got %d bytes", len(raw)))
		return
	}
	var formationHash [32]byte
	copy(formationHash[:], raw)

	// Auto-detect subtype by looking up the formation row's
	// message_type. Cross-subtype rejection: a row whose message_type
	// is not one of the four Cat III formation types returns
	// ErrTargetNotFormation → 404.
	ctx := r.Context()
	row, err := h.sub.LookupRow(ctx, formationHash)
	if err != nil {
		writeIngestError(w, http.StatusNotFound,
			fmt.Sprintf("formation_event_hash %s: %v", formationHex, projection.ErrFormationNotFound))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	switch row.MessageType {
	case "ghosttrace.events.v1.BehavioralClusterFormation":
		proj, err := projection.ProjectHypothesis(ctx, h.sub, formationHash)
		if err != nil {
			writeProjectionError(w, formationHex, err)
			return
		}
		out := buildBCOutput(formationHex, proj)
		_ = enc.Encode(out)

	case "ghosttrace.events.v1.AutomationGroupFormation":
		proj, err := projection.ProjectAutomationGroup(ctx, h.sub, formationHash)
		if err != nil {
			writeProjectionError(w, formationHex, err)
			return
		}
		out := buildAGOutput(formationHex, proj)
		_ = enc.Encode(out)

	case "ghosttrace.events.v1.CampaignHypothesisFormation":
		proj, err := projection.ProjectCampaignHypothesis(ctx, h.sub, formationHash)
		if err != nil {
			writeProjectionError(w, formationHex, err)
			return
		}
		out := buildCHOutput(formationHex, proj)
		_ = enc.Encode(out)

	case "ghosttrace.events.v1.CoordinationRingFormation":
		proj, err := projection.ProjectCoordinationRing(ctx, h.sub, formationHash)
		if err != nil {
			writeProjectionError(w, formationHex, err)
			return
		}
		out := buildCROutput(formationHex, proj)
		_ = enc.Encode(out)

	default:
		writeIngestError(w, http.StatusNotFound,
			fmt.Sprintf("%s: %s is %q (expected a Cat III formation event)",
				projection.ErrTargetNotFormation, formationHex, row.MessageType))
	}
}

// writeProjectionError maps a projection error to an HTTP status +
// structured body. ErrFormationNotFound + ErrTargetNotFormation → 404;
// other errors → 500.
func writeProjectionError(w http.ResponseWriter, formationHex string, err error) {
	if errors.Is(err, projection.ErrFormationNotFound) || errors.Is(err, projection.ErrTargetNotFormation) {
		writeIngestError(w, http.StatusNotFound,
			fmt.Sprintf("formation_event_hash %s: %v", formationHex, err))
		return
	}
	writeIngestError(w, http.StatusInternalServerError,
		fmt.Sprintf("project formation %s: %v", formationHex, err))
}

// stateOutput is the wire shape of the GET /v1/hypotheses/state JSON
// response. Mirrors the cmd/hypothesis-state CLI's output type, so
// operators get the same response regardless of channel (CLI or HTTP).
type stateOutput struct {
	Subtype          string                  `json:"subtype"`
	FormationHash    string                  `json:"formation_event_hash"`
	State            string                  `json:"state"`
	LatestPromotion  *promotionView          `json:"latest_promotion,omitempty"`
	LatestDemotion   *demotionView           `json:"latest_demotion,omitempty"`
	Dissolution      *dissolutionView        `json:"dissolution,omitempty"`
	MergedInto       *mergeView              `json:"merged_into,omitempty"`
	SplitInto        *splitView              `json:"split_into,omitempty"`
	LifecycleHistory []stateLifecycleEntry   `json:"lifecycle_history"`
	Latencies        stateLatencyView        `json:"latencies"`
}

type stateLifecycleEntry struct {
	Type      string `json:"type"`
	EventHash string `json:"event_hash"`
	EventTime int64  `json:"event_time"`
}

type stateLatencyView struct {
	FormationToFirstPromotionNs       *int64 `json:"formation_to_first_promotion_ns,omitempty"`
	LatestPromotionToLatestDemotionNs *int64 `json:"latest_promotion_to_latest_demotion_ns,omitempty"`
	FormationToDissolutionNs          *int64 `json:"formation_to_dissolution_ns,omitempty"`
}

type promotionView struct {
	PromotedAt     int64  `json:"promoted_at"`
	CadenceSeconds int64  `json:"cadence_seconds"`
	Reason         string `json:"reason,omitempty"`
}

type demotionView struct {
	DemotedAt int64  `json:"demoted_at"`
	Reason    string `json:"reason,omitempty"`
}

type dissolutionView struct {
	DissolvedAt int64  `json:"dissolved_at"`
	Reason      string `json:"reason,omitempty"`
}

type mergeView struct {
	MergedAt              int64    `json:"merged_at"`
	ProducedFormationHash string   `json:"produced_formation_event_hash"`
	AntecedentHashes      []string `json:"antecedent_formation_event_hashes"`
	Reason                string   `json:"reason,omitempty"`
}

type splitView struct {
	SplitAt         int64    `json:"split_at"`
	SuccessorHashes []string `json:"successor_formation_event_hashes"`
	Reason          string   `json:"reason,omitempty"`
}

func buildBCOutput(formationHex string, p projection.HypothesisProjection) stateOutput {
	o := stateOutput{
		Subtype:       "behavioral_cluster",
		FormationHash: formationHex,
		State:         string(p.State),
	}
	if p.LatestPromotion != nil {
		o.LatestPromotion = &promotionView{
			PromotedAt:     p.LatestPromotion.PromotedAt,
			CadenceSeconds: p.LatestPromotion.CadenceSeconds,
			Reason:         p.LatestPromotion.Reason,
		}
	}
	if p.LatestDemotion != nil {
		o.LatestDemotion = &demotionView{
			DemotedAt: p.LatestDemotion.DemotedAt,
			Reason:    p.LatestDemotion.Reason,
		}
	}
	if p.Dissolution != nil {
		o.Dissolution = &dissolutionView{
			DissolvedAt: p.Dissolution.DissolvedAt,
			Reason:      p.Dissolution.Reason,
		}
	}
	if p.MergedInto != nil {
		ants := make([]string, len(p.MergedInto.AntecedentFormationEventHashes))
		for i, h := range p.MergedInto.AntecedentFormationEventHashes {
			ants[i] = hex.EncodeToString(h)
		}
		o.MergedInto = &mergeView{
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
		o.SplitInto = &splitView{
			SplitAt:         p.SplitInto.SplitAt,
			SuccessorHashes: succs,
			Reason:          p.SplitInto.Reason,
		}
	}
	for _, entry := range p.LifecycleHistory {
		o.LifecycleHistory = append(o.LifecycleHistory, stateLifecycleEntry{
			Type:      entry.Type,
			EventHash: hex.EncodeToString(entry.EventHash[:]),
			EventTime: entry.EventTime,
		})
	}
	o.Latencies = stateLatencyView{
		FormationToFirstPromotionNs:       p.FormationToFirstPromotionLatencyNs,
		LatestPromotionToLatestDemotionNs: p.LatestPromotionToLatestDemotionLatencyNs,
		FormationToDissolutionNs:          p.FormationToDissolutionLatencyNs,
	}
	return o
}

func buildAGOutput(formationHex string, p projection.AutomationGroupProjection) stateOutput {
	o := stateOutput{
		Subtype:       "automation_group",
		FormationHash: formationHex,
		State:         string(p.State),
	}
	if p.LatestPromotion != nil {
		o.LatestPromotion = &promotionView{
			PromotedAt:     p.LatestPromotion.PromotedAt,
			CadenceSeconds: p.LatestPromotion.CadenceSeconds,
			Reason:         p.LatestPromotion.Reason,
		}
	}
	if p.LatestDemotion != nil {
		o.LatestDemotion = &demotionView{
			DemotedAt: p.LatestDemotion.DemotedAt,
			Reason:    p.LatestDemotion.Reason,
		}
	}
	if p.Dissolution != nil {
		o.Dissolution = &dissolutionView{
			DissolvedAt: p.Dissolution.DissolvedAt,
			Reason:      p.Dissolution.Reason,
		}
	}
	if p.MergedInto != nil {
		ants := make([]string, len(p.MergedInto.AntecedentFormationEventHashes))
		for i, h := range p.MergedInto.AntecedentFormationEventHashes {
			ants[i] = hex.EncodeToString(h)
		}
		o.MergedInto = &mergeView{
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
		o.SplitInto = &splitView{
			SplitAt:         p.SplitInto.SplitAt,
			SuccessorHashes: succs,
			Reason:          p.SplitInto.Reason,
		}
	}
	for _, entry := range p.LifecycleHistory {
		o.LifecycleHistory = append(o.LifecycleHistory, stateLifecycleEntry{
			Type:      entry.Type,
			EventHash: hex.EncodeToString(entry.EventHash[:]),
			EventTime: entry.EventTime,
		})
	}
	o.Latencies = stateLatencyView{
		FormationToFirstPromotionNs:       p.FormationToFirstPromotionLatencyNs,
		LatestPromotionToLatestDemotionNs: p.LatestPromotionToLatestDemotionLatencyNs,
		FormationToDissolutionNs:          p.FormationToDissolutionLatencyNs,
	}
	return o
}

func buildCHOutput(formationHex string, p projection.CampaignHypothesisProjection) stateOutput {
	o := stateOutput{
		Subtype:       "campaign_hypothesis",
		FormationHash: formationHex,
		State:         string(p.State),
	}
	if p.LatestPromotion != nil {
		o.LatestPromotion = &promotionView{
			PromotedAt:     p.LatestPromotion.PromotedAt,
			CadenceSeconds: p.LatestPromotion.CadenceSeconds,
			Reason:         p.LatestPromotion.Reason,
		}
	}
	if p.LatestDemotion != nil {
		o.LatestDemotion = &demotionView{
			DemotedAt: p.LatestDemotion.DemotedAt,
			Reason:    p.LatestDemotion.Reason,
		}
	}
	if p.Dissolution != nil {
		o.Dissolution = &dissolutionView{
			DissolvedAt: p.Dissolution.DissolvedAt,
			Reason:      p.Dissolution.Reason,
		}
	}
	if p.MergedInto != nil {
		ants := make([]string, len(p.MergedInto.AntecedentFormationEventHashes))
		for i, h := range p.MergedInto.AntecedentFormationEventHashes {
			ants[i] = hex.EncodeToString(h)
		}
		o.MergedInto = &mergeView{
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
		o.SplitInto = &splitView{
			SplitAt:         p.SplitInto.SplitAt,
			SuccessorHashes: succs,
			Reason:          p.SplitInto.Reason,
		}
	}
	for _, entry := range p.LifecycleHistory {
		o.LifecycleHistory = append(o.LifecycleHistory, stateLifecycleEntry{
			Type:      entry.Type,
			EventHash: hex.EncodeToString(entry.EventHash[:]),
			EventTime: entry.EventTime,
		})
	}
	o.Latencies = stateLatencyView{
		FormationToFirstPromotionNs:       p.FormationToFirstPromotionLatencyNs,
		LatestPromotionToLatestDemotionNs: p.LatestPromotionToLatestDemotionLatencyNs,
		FormationToDissolutionNs:          p.FormationToDissolutionLatencyNs,
	}
	return o
}

func buildCROutput(formationHex string, p projection.CoordinationRingProjection) stateOutput {
	o := stateOutput{
		Subtype:       "coordination_ring",
		FormationHash: formationHex,
		State:         string(p.State),
	}
	if p.LatestPromotion != nil {
		o.LatestPromotion = &promotionView{
			PromotedAt:     p.LatestPromotion.PromotedAt,
			CadenceSeconds: p.LatestPromotion.CadenceSeconds,
			Reason:         p.LatestPromotion.Reason,
		}
	}
	if p.LatestDemotion != nil {
		o.LatestDemotion = &demotionView{
			DemotedAt: p.LatestDemotion.DemotedAt,
			Reason:    p.LatestDemotion.Reason,
		}
	}
	if p.Dissolution != nil {
		o.Dissolution = &dissolutionView{
			DissolvedAt: p.Dissolution.DissolvedAt,
			Reason:      p.Dissolution.Reason,
		}
	}
	if p.MergedInto != nil {
		ants := make([]string, len(p.MergedInto.AntecedentFormationEventHashes))
		for i, h := range p.MergedInto.AntecedentFormationEventHashes {
			ants[i] = hex.EncodeToString(h)
		}
		o.MergedInto = &mergeView{
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
		o.SplitInto = &splitView{
			SplitAt:         p.SplitInto.SplitAt,
			SuccessorHashes: succs,
			Reason:          p.SplitInto.Reason,
		}
	}
	for _, entry := range p.LifecycleHistory {
		o.LifecycleHistory = append(o.LifecycleHistory, stateLifecycleEntry{
			Type:      entry.Type,
			EventHash: hex.EncodeToString(entry.EventHash[:]),
			EventTime: entry.EventTime,
		})
	}
	o.Latencies = stateLatencyView{
		FormationToFirstPromotionNs:       p.FormationToFirstPromotionLatencyNs,
		LatestPromotionToLatestDemotionNs: p.LatestPromotionToLatestDemotionLatencyNs,
		FormationToDissolutionNs:          p.FormationToDissolutionLatencyNs,
	}
	return o
}

