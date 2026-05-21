package httpapi

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/replay"
)

// handleReplayOperationalSession serves GET /v1/replay/operational-session
// per decision-log §0091. Phase 1 deterministic replay of a single
// OperationalSession; mirrors cmd/replay-operational-session.
//
// Query parameters:
//   - target_event_hash (REQUIRED) — hex-encoded BLAKE3-256 of the OS.
//
// Response codes:
//   - 200 — JSON OperationalSessionReport; match=true OR match=false
//     (drift) are both 200 responses (the boolean carries the verdict).
//   - 400 — missing parameter, invalid hex, wrong hash length.
//   - 404 — target-not-found, target-wrong-type, definition-unknown,
//     definition-parameter-mismatch, source-not-found, source-wrong-type.
//   - 405 — non-GET.
//   - 503 — substrate not configured.
func (h *Handler) handleReplayOperationalSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.sub == nil {
		writeIngestError(w, http.StatusServiceUnavailable,
			"replay endpoints not configured (handler constructed without WithSubstrate)")
		return
	}

	targetHash, ok := parseTargetEventHash(w, r)
	if !ok {
		return
	}

	report, err := replay.ReplayOperationalSession(r.Context(), h.sub, targetHash)
	if err != nil {
		writeReplayError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(opSessionPayload{
		TargetEventHash:      report.TargetHashHex,
		RecomputedEventHash:  report.RecomputedHashHex,
		Match:                report.Match,
		DefinitionVersion:    report.DefinitionVersion,
		DefinitionParameters: report.DefinitionParameters,
		SourceEventHash:      report.SourceEventHashHex,
	})
}

// handleReplayAllOperationalSessions serves GET /v1/replay/operational-sessions
// per §0091. Phase 1 substrate-wide batch; mirrors
// cmd/replay-all-operational-sessions.
//
// No query parameters. Response codes: 200 (always for completed walks);
// 405 non-GET; 503 substrate not configured.
func (h *Handler) handleReplayAllOperationalSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.sub == nil {
		writeIngestError(w, http.StatusServiceUnavailable,
			"replay endpoints not configured (handler constructed without WithSubstrate)")
		return
	}

	report, err := replay.ReplayAllOperationalSessions(r.Context(), h.sub)
	if err != nil {
		writeIngestError(w, http.StatusInternalServerError, fmt.Sprintf("batch replay: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(report)
}

// handleReplayFormation serves GET /v1/replay/formation per §0091.
// Phase 3 reconstructive replay of a Cat III formation. Auto-detects
// the subtype from the substrate row's message_type and dispatches to
// the appropriate per-subtype replay function. Consolidates the four
// per-subtype Phase 3 CLIs (§0086-§0089) into one endpoint.
//
// Query parameters:
//   - target_event_hash (REQUIRED) — hex-encoded BLAKE3-256.
//
// Response codes:
//   - 200 — JSON with subtype + match boolean + diagnostic fields.
//   - 400 — missing/invalid parameter.
//   - 404 — target-not-found OR target is not a Cat III formation OR
//     pattern-unknown OR pattern-parameter-mismatch.
//   - 405 — non-GET.
//   - 503 — substrate not configured.
func (h *Handler) handleReplayFormation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.sub == nil {
		writeIngestError(w, http.StatusServiceUnavailable,
			"replay endpoints not configured (handler constructed without WithSubstrate)")
		return
	}

	targetHash, ok := parseTargetEventHash(w, r)
	if !ok {
		return
	}

	ctx := r.Context()
	row, err := h.sub.LookupRow(ctx, targetHash)
	if err != nil {
		writeIngestError(w, http.StatusNotFound,
			fmt.Sprintf("target_event_hash %x: not found", targetHash))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	switch row.MessageType {
	case "ghosttrace.events.v1.BehavioralClusterFormation":
		report, err := replay.ReplayBehavioralClusterFormation(ctx, h.sub, targetHash)
		if err != nil {
			writeReplayError(w, err)
			return
		}
		_ = enc.Encode(formationPayload{
			Subtype:                      "behavioral_cluster",
			TargetEventHash:              report.TargetHashHex,
			Match:                        report.Match,
			RecomputedEventHash:          report.RecomputedHashHex,
			PatternSignature:             report.PatternSignature,
			PatternParameters:            report.PatternParameters,
			ReconstructedFormationCount:  report.ReconstructedFormationCount,
			ContributingObservationCount: report.ContributingObservationCount,
			MaxCommittedAtNs:             report.MaxCommittedAtNs,
		})

	case "ghosttrace.events.v1.AutomationGroupFormation":
		report, err := replay.ReplayAutomationGroupFormation(ctx, h.sub, targetHash)
		if err != nil {
			writeReplayError(w, err)
			return
		}
		_ = enc.Encode(formationPayload{
			Subtype:                      "automation_group",
			TargetEventHash:              report.TargetHashHex,
			Match:                        report.Match,
			RecomputedEventHash:          report.RecomputedHashHex,
			PatternSignature:             report.PatternSignature,
			PatternParameters:            report.PatternParameters,
			ReconstructedFormationCount:  report.ReconstructedFormationCount,
			ContributingObservationCount: report.ContributingObservationCount,
			MaxCommittedAtNs:             report.MaxCommittedAtNs,
		})

	case "ghosttrace.events.v1.CampaignHypothesisFormation":
		report, err := replay.ReplayCampaignHypothesisFormation(ctx, h.sub, targetHash)
		if err != nil {
			writeReplayError(w, err)
			return
		}
		_ = enc.Encode(formationPayload{
			Subtype:                      "campaign_hypothesis",
			TargetEventHash:              report.TargetHashHex,
			Match:                        report.Match,
			RecomputedEventHash:          report.RecomputedHashHex,
			PatternSignature:             report.PatternSignature,
			PatternParameters:            report.PatternParameters,
			ReconstructedFormationCount:  report.ReconstructedFormationCount,
			ContributingObservationCount: report.ContributingObservationCount,
			MaxCommittedAtNs:             report.MaxCommittedAtNs,
		})

	case "ghosttrace.events.v1.CoordinationRingFormation":
		report, err := replay.ReplayCoordinationRingFormation(ctx, h.sub, targetHash)
		if err != nil {
			writeReplayError(w, err)
			return
		}
		_ = enc.Encode(formationPayload{
			Subtype:                      "coordination_ring",
			TargetEventHash:              report.TargetHashHex,
			Match:                        report.Match,
			RecomputedEventHash:          report.RecomputedHashHex,
			PatternSignature:             report.PatternSignature,
			PatternParameters:            report.PatternParameters,
			ReconstructedFormationCount:  report.ReconstructedFormationCount,
			ContributingObservationCount: report.ContributingObservationCount,
			MaxCommittedAtNs:             report.MaxCommittedAtNs,
		})

	default:
		writeIngestError(w, http.StatusNotFound,
			fmt.Sprintf("target_event_hash %x is %q (expected a Cat III formation event)",
				targetHash, row.MessageType))
	}
}

// handleReplayAllFormations serves GET /v1/replay/formations per §0091.
// Phase 3 substrate-wide batch across all four Cat III subtypes;
// mirrors cmd/replay-all-formations.
//
// Query parameters:
//   - subtype (optional): behavioral_cluster | automation_group |
//     campaign_hypothesis | coordination_ring. Empty = all.
//
// Response: per-subtype BatchReplayReport sections.
func (h *Handler) handleReplayAllFormations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.sub == nil {
		writeIngestError(w, http.StatusServiceUnavailable,
			"replay endpoints not configured (handler constructed without WithSubstrate)")
		return
	}

	subtype := r.URL.Query().Get("subtype")
	if subtype != "" &&
		subtype != "behavioral_cluster" &&
		subtype != "automation_group" &&
		subtype != "campaign_hypothesis" &&
		subtype != "coordination_ring" {
		writeIngestError(w, http.StatusBadRequest,
			fmt.Sprintf("subtype %q is not one of: behavioral_cluster, automation_group, campaign_hypothesis, coordination_ring", subtype))
		return
	}

	ctx := r.Context()
	out := allFormationsPayload{}

	if subtype == "" || subtype == "behavioral_cluster" {
		rep, err := replay.ReplayAllBehavioralClusterFormations(ctx, h.sub)
		if err != nil {
			writeIngestError(w, http.StatusInternalServerError, fmt.Sprintf("BC batch: %v", err))
			return
		}
		out.BehavioralCluster = &rep
	}
	if subtype == "" || subtype == "automation_group" {
		rep, err := replay.ReplayAllAutomationGroupFormations(ctx, h.sub)
		if err != nil {
			writeIngestError(w, http.StatusInternalServerError, fmt.Sprintf("AG batch: %v", err))
			return
		}
		out.AutomationGroup = &rep
	}
	if subtype == "" || subtype == "campaign_hypothesis" {
		rep, err := replay.ReplayAllCampaignHypothesisFormations(ctx, h.sub)
		if err != nil {
			writeIngestError(w, http.StatusInternalServerError, fmt.Sprintf("CH batch: %v", err))
			return
		}
		out.CampaignHypothesis = &rep
	}
	if subtype == "" || subtype == "coordination_ring" {
		rep, err := replay.ReplayAllCoordinationRingFormations(ctx, h.sub)
		if err != nil {
			writeIngestError(w, http.StatusInternalServerError, fmt.Sprintf("CR batch: %v", err))
			return
		}
		out.CoordinationRing = &rep
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
}

// parseTargetEventHash extracts + validates the target_event_hash
// query parameter. Writes a 400 response on failure and returns ok=false.
func parseTargetEventHash(w http.ResponseWriter, r *http.Request) ([32]byte, bool) {
	targetHex := r.URL.Query().Get("target_event_hash")
	if targetHex == "" {
		writeIngestError(w, http.StatusBadRequest,
			"missing required query parameter: target_event_hash")
		return [32]byte{}, false
	}
	raw, err := hex.DecodeString(targetHex)
	if err != nil {
		writeIngestError(w, http.StatusBadRequest,
			fmt.Sprintf("decode target_event_hash: %v", err))
		return [32]byte{}, false
	}
	if len(raw) != 32 {
		writeIngestError(w, http.StatusBadRequest,
			fmt.Sprintf("target_event_hash must be 32 bytes (64 hex chars); got %d bytes", len(raw)))
		return [32]byte{}, false
	}
	var h [32]byte
	copy(h[:], raw)
	return h, true
}

// writeReplayError maps a replay package error to an HTTP status +
// structured body. The replay sentinels are precondition-failure
// errors; map all of them to 404 (mirrors the per-target CLI exit-3
// semantic — target/source integrity issue).
func writeReplayError(w http.ResponseWriter, err error) {
	if errors.Is(err, replay.ErrTargetNotFound) ||
		errors.Is(err, replay.ErrTargetWrongType) ||
		errors.Is(err, replay.ErrDefinitionUnknown) ||
		errors.Is(err, replay.ErrDefinitionParameterMismatch) ||
		errors.Is(err, replay.ErrSourceNotFound) ||
		errors.Is(err, replay.ErrSourceWrongType) ||
		errors.Is(err, replay.ErrPatternUnknown) ||
		errors.Is(err, replay.ErrPatternParameterMismatch) {
		writeIngestError(w, http.StatusNotFound, err.Error())
		return
	}
	writeIngestError(w, http.StatusInternalServerError, fmt.Sprintf("replay: %v", err))
}

// opSessionPayload mirrors cmd/replay-operational-session's JSON shape.
type opSessionPayload struct {
	TargetEventHash      string `json:"target_event_hash"`
	RecomputedEventHash  string `json:"recomputed_event_hash"`
	Match                bool   `json:"match"`
	DefinitionVersion    string `json:"definition_version"`
	DefinitionParameters string `json:"definition_parameters"`
	SourceEventHash      string `json:"source_event_hash"`
}

// formationPayload mirrors the four per-subtype Phase 3 CLIs' JSON
// shape — unified into a single output type with a `subtype` field
// identifying which Cat III subtype produced the response.
type formationPayload struct {
	Subtype                      string `json:"subtype"`
	TargetEventHash              string `json:"target_event_hash"`
	Match                        bool   `json:"match"`
	RecomputedEventHash          string `json:"recomputed_event_hash,omitempty"`
	PatternSignature             string `json:"pattern_signature"`
	PatternParameters            string `json:"pattern_parameters"`
	ReconstructedFormationCount  int    `json:"reconstructed_formation_count"`
	ContributingObservationCount int    `json:"contributing_observation_count"`
	MaxCommittedAtNs             int64  `json:"max_committed_at_ns"`
}

// allFormationsPayload mirrors cmd/replay-all-formations's JSON shape.
type allFormationsPayload struct {
	BehavioralCluster  *replay.BatchReplayReport `json:"behavioral_cluster,omitempty"`
	AutomationGroup    *replay.BatchReplayReport `json:"automation_group,omitempty"`
	CampaignHypothesis *replay.BatchReplayReport `json:"campaign_hypothesis,omitempty"`
	CoordinationRing   *replay.BatchReplayReport `json:"coordination_ring,omitempty"`
}
