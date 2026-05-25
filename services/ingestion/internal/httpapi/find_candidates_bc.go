package httpapi

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"google.golang.org/protobuf/proto"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/signatures"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// behavioralObservationMessageType is the proto message_type string
// for BehavioralObservation records. Local copy mirroring the F3 CLIs'
// equivalent constant (per-CLI scoping; no shared symbol exists today).
const behavioralObservationMessageType = "ghosttrace.events.v1.BehavioralObservation"

// handleFindCandidatesBC serves GET /v1/find-candidates/behavioral-cluster
// per decision-log §0190. Third operational-tier advance after §0188
// morphology + §0189 layer-b-verdict; first F3 candidate evaluation
// HTTP endpoint per §0188 MO1 wire-parity discipline.
//
// Invokes the keystroke_timing_clustering_v1 signature (§0174) over
// the substrate's BehavioralObservation records + emits FormationCandidates
// as JSON. Wire shape mirrors cmd/find-behavioral-cluster-candidates'
// emissionEnvelope byte-for-byte per §0163 stable wire contract +
// §0188 MO1 CLI↔HTTP wire-parity discipline.
//
// Per §3 N3 + §0152: this endpoint does NOT commit formation events
// directly. Operators consume the JSON envelope + decide whether to
// invoke the form-* HTTP endpoint downstream.
//
// Query parameters (all optional):
//   - threshold: keystroke_timing_clustering_v1 threshold override
//     (0 = signature default = 3 distinct actors per cluster).
//   - limit: maximum candidates to emit (0 = unlimited).
//
// Response codes:
//   - 200 — JSON emission envelope (always; empty candidates array
//     when threshold not met).
//   - 400 — invalid query parameter (non-numeric).
//   - 405 — non-GET method.
//   - 500 — signature evaluation internal error.
//   - 503 — substrate not configured on this handler.
//
// Operational cost: this endpoint walks every substrate BehavioralObservation
// row + invokes the signature in a single pass. Safe for interactive
// dashboard use; concurrent invocations safe under SQLite WAL.
func (h *Handler) handleFindCandidatesBC(w http.ResponseWriter, r *http.Request) {
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

	observations, err := collectBehavioralObservationsHTTP(r.Context(), h.sub)
	if err != nil {
		writeIngestError(w, http.StatusInternalServerError, fmt.Sprintf("collect: %v", err))
		return
	}

	sig := &signatures.KeystrokeTimingClusteringV1{
		Threshold: uint32(threshold),
	}
	result, err := sig.EvaluateBehavioral(r.Context(), observations)
	if err != nil {
		writeIngestError(w, http.StatusInternalServerError, fmt.Sprintf("signature.EvaluateBehavioral: %v", err))
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

// findCandidatesCandidatePayload is the per-candidate wire shape.
// Mirrors the F3 CLIs' candidateJSON exactly per §0163 + §0188 MO1
// CLI↔HTTP wire-parity discipline. Interaction-centric subtypes
// (CoordinationRing) populate the Interactions field; all other
// subtypes leave it empty (omitempty removes it from JSON) per §0186 MO1
// CLI wire-contract extension pattern landed at §0193.
type findCandidatesCandidatePayload struct {
	SignatureName     string      `json:"signature_name"`
	HypothesisSubtype string      `json:"hypothesis_subtype"`
	ActorRefs         []string    `json:"actor_refs"`
	SourceHashesHex   []string    `json:"source_event_hashes_hex"`
	EvidenceCount     uint32      `json:"evidence_count"`
	ConfidenceHint    float64     `json:"confidence_hint"`
	Interactions      [][2]string `json:"interactions,omitempty"`
}

// findCandidatesStatsPayload is the per-evaluation stats wire shape.
// Mirrors the F3 CLIs' statsJSON exactly.
type findCandidatesStatsPayload struct {
	ObservationsScanned              uint32            `json:"observations_scanned"`
	ObservationsSkippedNoActor       uint32            `json:"observations_skipped_no_actor"`
	ObservationsSkippedWrongModality uint32            `json:"observations_skipped_wrong_modality"`
	ActorsAggregated                 uint32            `json:"actors_aggregated"`
	ActorsAboveThreshold             uint32            `json:"actors_above_threshold"`
	CandidatesEmitted                uint32            `json:"candidates_emitted"`
	PerCollector                     map[string]uint32 `json:"per_collector,omitempty"`
}

// findCandidatesPayload is the response envelope wire shape. Mirrors
// the F3 CLIs' emissionEnvelope exactly.
type findCandidatesPayload struct {
	SignatureName  string                          `json:"signature_name"`
	CandidateCount int                             `json:"candidate_count"`
	Candidates     []findCandidatesCandidatePayload `json:"candidates"`
	Stats          findCandidatesStatsPayload      `json:"stats"`
}

// collectBehavioralObservationsHTTP walks the substrate and unmarshals
// every committed BehavioralObservation record. Mirrors the
// find-behavioral-cluster-candidates CLI's collectBehavioralObservations
// (local-helper duplication discipline preserved per §0190 MO1 — see
// decision-log entry for shared-collector-package deferral rationale).
func collectBehavioralObservationsHTTP(ctx context.Context, sub *substrate.Substrate) ([]*eventsv1.BehavioralObservation, error) {
	var out []*eventsv1.BehavioralObservation
	err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != behavioralObservationMessageType {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return fmt.Errorf("ReadBlob %x: %w", row.EventHash[:8], err)
		}
		obs := &eventsv1.BehavioralObservation{}
		if err := proto.Unmarshal(payload, obs); err != nil {
			return fmt.Errorf("Unmarshal %x: %w", row.EventHash[:8], err)
		}
		out = append(out, obs)
		return nil
	})
	return out, err
}

// subtypeNameHTTP maps a signatures.HypothesisSubtype to its
// human-readable name. Mirrors the F3 CLIs' subtypeName function;
// available at httpapi package scope for future find-candidates
// endpoints (per-CLI duplication today; consolidated here once).
func subtypeNameHTTP(s signatures.HypothesisSubtype) string {
	switch s {
	case signatures.HypothesisSubtypeAutomationGroup:
		return "AutomationGroup"
	case signatures.HypothesisSubtypeBehavioralCluster:
		return "BehavioralCluster"
	case signatures.HypothesisSubtypeCampaignHypothesis:
		return "CampaignHypothesis"
	case signatures.HypothesisSubtypeCoordinationRing:
		return "CoordinationRing"
	default:
		return "Unknown"
	}
}
