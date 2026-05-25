package httpapi

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/observationcollector"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/signatures"
)

// handleFindCandidatesAGBrowser serves GET
// /v1/find-candidates/automation-group-browser per decision-log §0191.
// Second F3 candidate evaluation HTTP endpoint after the §0190 BC pilot;
// extends the per-subtype endpoint pattern along the browser-modality
// AutomationGroup axis.
//
// Invokes the cdp_marker_density_v1 signature (§0152) over the
// substrate's BrowserObservation records + emits FormationCandidates as
// JSON. Wire shape mirrors cmd/find-automation-group-candidates'
// emissionEnvelope byte-for-byte per §0163 + §0188 MO1 CLI↔HTTP
// wire-parity discipline.
//
// Per §3 N3 + §0152: this endpoint does NOT commit formation events.
//
// Query parameters (all optional):
//   - threshold: cdp_marker_density_v1 threshold override (0 = signature
//     default = 2 detections per actor).
//   - limit: maximum candidates to emit (0 = unlimited).
//
// Response codes mirror §0190.
func (h *Handler) handleFindCandidatesAGBrowser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.sub == nil {
		writeIngestError(w, http.StatusServiceUnavailable,
			"find-candidates endpoint not configured (handler constructed without WithSubstrate)")
		return
	}

	q := r.URL.Query()
	threshold, err := parseInt64Param(q, "threshold")
	if err != nil {
		writeIngestError(w, http.StatusBadRequest, err.Error())
		return
	}
	if threshold < 0 {
		writeIngestError(w, http.StatusBadRequest, "threshold must be non-negative")
		return
	}
	limit, err := parseIntParam(q, "limit")
	if err != nil {
		writeIngestError(w, http.StatusBadRequest, err.Error())
		return
	}
	if limit < 0 {
		writeIngestError(w, http.StatusBadRequest, "limit must be non-negative")
		return
	}

	observations, err := observationcollector.CollectBrowser(r.Context(), h.sub)
	if err != nil {
		writeIngestError(w, http.StatusInternalServerError, fmt.Sprintf("collect: %v", err))
		return
	}

	sig := &signatures.CDPMarkerDensityV1{
		Threshold: uint32(threshold),
	}
	result, err := sig.EvaluateBrowser(r.Context(), observations)
	if err != nil {
		writeIngestError(w, http.StatusInternalServerError, fmt.Sprintf("signature.EvaluateBrowser: %v", err))
		return
	}

	candidates := result.Candidates
	if limit > 0 && len(candidates) > limit {
		candidates = candidates[:limit]
	}

	env := findCandidatesPayload{
		SignatureName:  sig.Name(),
		CandidateCount: len(candidates),
		Candidates:     make([]findCandidatesCandidatePayload, 0, len(candidates)),
		Stats: findCandidatesStatsPayload{
			ObservationsScanned:              result.Stats.ObservationsScanned,
			ObservationsSkippedNoActor:       result.Stats.ObservationsSkippedNoActor,
			ObservationsSkippedWrongModality: result.Stats.ObservationsSkippedWrongModality,
			ActorsAggregated:                 result.Stats.ActorsAggregated,
			ActorsAboveThreshold:             result.Stats.ActorsAboveThreshold,
			CandidatesEmitted:                result.Stats.CandidatesEmitted,
			PerCollector:                     result.Stats.PerCollector,
		},
	}
	for _, c := range candidates {
		hashesHex := make([]string, len(c.SourceHashes))
		for i, hh := range c.SourceHashes {
			hashesHex[i] = hex.EncodeToString(hh)
		}
		env.Candidates = append(env.Candidates, findCandidatesCandidatePayload{
			SignatureName:     c.SignatureName,
			HypothesisSubtype: subtypeNameHTTP(c.HypothesisSubtype),
			ActorRefs:         c.ActorRefs,
			SourceHashesHex:   hashesHex,
			EvidenceCount:     c.EvidenceCount,
			ConfidenceHint:    c.ConfidenceHint,
		})
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(env)
}

