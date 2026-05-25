package httpapi

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/attribution"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/observationcollector"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/signatures"
)

// handleFindCandidatesAGNetwork serves GET
// /v1/find-candidates/automation-group-network per decision-log §0194.
// FIFTH and final F3 candidate evaluation HTTP endpoint; CLOSES the
// §0188 operational HTTP gap program (3rd of 3 gaps fully discharged
// after morphology + layer-b-verdict + 5-of-5 find-candidates-*).
//
// Two-signature selection per the CLI's pattern (§0161 + §0169):
//   - signature=p0f → tcp_fingerprint_clustering_v1 (§0161; default)
//   - signature=flow-features → tcp_flow_features_clustering_v1
//     (§0169; CICFlowMeter-style adapters per §0162 gap (2) closure)
//
// Both signatures operate over NetworkObservation records + emit
// AutomationGroup FormationCandidates. Reuses §0195's shared
// observationcollector.CollectNetwork + AttributionLookup consumption.
//
// Per §3 N3 + §0152: this endpoint does NOT commit formation events.
//
// Query parameters:
//   - signature (REQUIRED): "p0f" or "flow-features".
//   - threshold (optional): signature threshold override (0 = signature
//     default = 3).
//   - limit (optional): maximum candidates to emit (0 = unlimited).
//   - with_attribution (optional): "true"/"1" to consult §0168
//     AttributionLookup.
func (h *Handler) handleFindCandidatesAGNetwork(w http.ResponseWriter, r *http.Request) {
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
	signatureSelection := q.Get("signature")
	if signatureSelection == "" {
		writeIngestError(w, http.StatusBadRequest,
			"signature query parameter is required (valid: p0f, flow-features)")
		return
	}
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
	withAttribution, ok := parseBoolQuery(w, r, "with_attribution", false)
	if !ok {
		return
	}

	sig, err := selectAGNetworkSignature(signatureSelection, uint32(threshold))
	if err != nil {
		writeIngestError(w, http.StatusBadRequest, err.Error())
		return
	}

	observations, err := observationcollector.CollectNetwork(r.Context(), h.sub)
	if err != nil {
		writeIngestError(w, http.StatusInternalServerError, fmt.Sprintf("collect: %v", err))
		return
	}

	var attributionView signatures.AttributionLookup
	if withAttribution {
		v, err := attribution.CollectAttributionView(r.Context(), h.sub)
		if err != nil {
			writeIngestError(w, http.StatusInternalServerError, fmt.Sprintf("attribution.CollectAttributionView: %v", err))
			return
		}
		attributionView = v
	}

	result, err := sig.EvaluateNetwork(r.Context(), observations, attributionView)
	if err != nil {
		writeIngestError(w, http.StatusInternalServerError, fmt.Sprintf("signature.EvaluateNetwork: %v", err))
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

// selectAGNetworkSignature dispatches the signature-selection query
// param to the corresponding NetworkSignature implementation. Mirrors
// cmd/find-automation-group-candidates-network's selectNetworkSignature
// helper.
func selectAGNetworkSignature(name string, thresholdOverride uint32) (signatures.NetworkSignature, error) {
	switch name {
	case "p0f":
		return &signatures.TCPFingerprintClusteringV1{Threshold: thresholdOverride}, nil
	case "flow-features":
		return &signatures.TCPFlowFeaturesClusteringV1{Threshold: thresholdOverride}, nil
	default:
		return nil, fmt.Errorf("unknown signature value %q (valid: p0f, flow-features)", name)
	}
}
