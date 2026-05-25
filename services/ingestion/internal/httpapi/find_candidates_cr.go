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

// handleFindCandidatesCR serves GET /v1/find-candidates/coordination-ring
// per decision-log §0193. Fourth F3 candidate evaluation HTTP endpoint;
// extends the per-subtype pattern along the network-modality
// CoordinationRing axis. Invokes the endpoint_co_visit_v1 signature
// (§0185) over the substrate's NetworkObservation records.
//
// **First HTTP endpoint populating the Interactions wire-shape field**
// per §0186 MO1 CLI wire-contract extension pattern. Other endpoints
// leave Interactions empty (omitempty removes it from JSON); CR
// populates the edge set per §0070 interaction-centric ontology.
//
// Per §3 N3 + §0152: this endpoint does NOT commit formation events.
// Per §0070 + §0185 interaction-centric ontology: operators committing
// CoordinationRingFormation downstream CONVERT Interactions into
// CoordinationRingInteraction protos (actor_a=edge[0], actor_b=edge[1]
// per §0070 within-edge lex).
//
// Reuses §0195's shared observationcollector.CollectNetwork +
// (when with_attribution=true) §0168 AttributionLookup consumption.
//
// Query parameters (all optional):
//   - threshold: endpoint_co_visit_v1 threshold override (0 = signature
//     default = 3 distinct actors per cluster).
//   - bucket_seconds: time-bucket width in seconds (0 = signature
//     default = 60s).
//   - limit: maximum candidates to emit (0 = unlimited).
//   - with_attribution: "true"/"1" to consult §0168 AttributionLookup.
func (h *Handler) handleFindCandidatesCR(w http.ResponseWriter, r *http.Request) {
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
	bucketSeconds, err := parseInt64Param(q, "bucket_seconds")
	if err != nil {
		writeIngestError(w, http.StatusBadRequest, err.Error())
		return
	}
	if bucketSeconds < 0 {
		writeIngestError(w, http.StatusBadRequest, "bucket_seconds must be non-negative")
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

	sig := &signatures.EndpointCoVisitV1{
		Threshold:     uint32(threshold),
		BucketSeconds: uint64(bucketSeconds),
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
			Interactions:      c.Interactions,
		})
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(env)
}
