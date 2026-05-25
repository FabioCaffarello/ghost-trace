package httpapi

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"google.golang.org/protobuf/proto"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/attribution"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/signatures"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// networkObservationMessageType is the proto message_type string for
// NetworkObservation records. Local copy mirroring the F3 CLIs' equivalent
// constant per §0190 MO1 pilot-first replication discipline. Shared
// across §0192 + §0193 + §0194 network-modality endpoints.
const networkObservationMessageType = "ghosttrace.events.v1.NetworkObservation"

// handleFindCandidatesCH serves GET /v1/find-candidates/campaign-hypothesis
// per decision-log §0192. Third F3 candidate evaluation HTTP endpoint;
// extends the per-subtype pattern along the network-modality CampaignHypothesis
// axis. Invokes the temporal_endpoint_cohort_v1 signature (§0182) over the
// substrate's NetworkObservation records.
//
// Per §3 N3 + §0152: this endpoint does NOT commit formation events.
// Per §0182 + §0063 event-centric ontology: emitted candidates carry
// ActorRefs as enrichment-only — operators committing
// CampaignHypothesisFormation downstream MUST drop ActorRefs (proto has
// no actor_refs field).
//
// Wire shape mirrors cmd/find-campaign-hypothesis-candidates'
// emissionEnvelope byte-for-byte per §0163 + §0188 MO1.
//
// Query parameters (all optional):
//   - threshold: temporal_endpoint_cohort_v1 threshold override
//     (0 = signature default = 3 events per cluster).
//   - bucket_seconds: time-bucket width in seconds (0 = signature
//     default = 60s).
//   - limit: maximum candidates to emit (0 = unlimited).
//   - with_attribution: "true"/"1" to consult Cat II DerivedActorAttribution
//     records (§0168) for actor enrichment.
func (h *Handler) handleFindCandidatesCH(w http.ResponseWriter, r *http.Request) {
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

	observations, err := collectNetworkObservationsHTTP(r.Context(), h.sub)
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

	sig := &signatures.TemporalEndpointCohortV1{
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
		})
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(env)
}

// collectNetworkObservationsHTTP walks the substrate and unmarshals
// every committed NetworkObservation record. Shared across §0192 +
// §0193 + §0194 network-modality endpoints per §0190 MO1 pilot-first
// replication discipline.
func collectNetworkObservationsHTTP(ctx context.Context, sub *substrate.Substrate) ([]*eventsv1.NetworkObservation, error) {
	var out []*eventsv1.NetworkObservation
	err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != networkObservationMessageType {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return fmt.Errorf("ReadBlob %x: %w", row.EventHash[:8], err)
		}
		obs := &eventsv1.NetworkObservation{}
		if err := proto.Unmarshal(payload, obs); err != nil {
			return fmt.Errorf("Unmarshal %x: %w", row.EventHash[:8], err)
		}
		out = append(out, obs)
		return nil
	})
	return out, err
}
