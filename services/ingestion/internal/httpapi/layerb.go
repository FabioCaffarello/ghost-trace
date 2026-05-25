package httpapi

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/hypothesis/layerb"
)

// handleLayerBVerdict serves GET /v1/layer-b-verdict per decision-log
// §0189. HTTP exposure of layerb.Evaluate; second operational-tier
// advance after §0188's GET /v1/morphology.
//
// Evaluates the Layer B demotion-candidacy verdict (L-BC-OR per §0135
// + §0138) for a given formation event hash against a caller-supplied
// LayerBParameters tuple. Read-only, idempotent — no events committed.
//
// Per §0188 MO1 CLI↔HTTP wire-parity discipline: there is no CLI
// counterpart for layerb.Evaluate today (Layer B is consumed in-process
// by demote-* CLIs); §0189 establishes the wire shape from scratch.
// The JSON response mirrors the layerb.Verdict struct snake-case-
// converted; the formation_event_hash_hex echo enables operator-side
// correlation with subsequent demote requests.
//
// Query parameters (all required):
//   - formation_event_hash_hex: 64-char hex-encoded BLAKE3 hash of the
//     formation event.
//   - t_b_numerator, t_b_denominator: freshness threshold T_B as
//     rational pair (denominator > 0 per §2.6).
//   - k_c_numerator, k_c_denominator: saturation threshold K_C as
//     rational pair (denominator > 0 per §2.6).
//   - n_window: substrate-event window count N (> 0 per §0138 W-count form).
//   - n_a_duration_nanoseconds: N_A duration bound (currently advisory;
//     reserved for future per-§0138 N_A-driven window).
//
// Response codes:
//   - 200 — JSON verdict envelope (verdict always returned; the
//     `fired` boolean carries the L-BC-OR disjunction).
//   - 400 — invalid query parameter (missing required, non-numeric,
//     zero denominator, malformed hex, non-32-byte hash).
//   - 405 — non-GET method.
//   - 500 — layerb.Evaluate internal error (rare; substrate walk failure).
//   - 503 — substrate not configured on this handler.
//
// Operational cost: this endpoint walks the substrate to capture the
// last N events (N from query param). Bounded by min(substrate_event_count, N).
// Safe for interactive dashboard use; concurrent invocations safe under
// SQLite WAL.
func (h *Handler) handleLayerBVerdict(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.sub == nil {
		writeIngestError(w, http.StatusServiceUnavailable,
			"layer-b-verdict endpoint not configured (handler constructed without WithSubstrate)")
		return
	}

	q := r.URL.Query()

	hashHex := q.Get("formation_event_hash_hex")
	if hashHex == "" {
		writeIngestError(w, http.StatusBadRequest, "formation_event_hash_hex is required")
		return
	}
	if len(hashHex) != 64 {
		writeIngestError(w, http.StatusBadRequest,
			fmt.Sprintf("formation_event_hash_hex must be 64 hex chars (got %d)", len(hashHex)))
		return
	}
	hashBytes, err := hex.DecodeString(hashHex)
	if err != nil {
		writeIngestError(w, http.StatusBadRequest,
			fmt.Sprintf("formation_event_hash_hex must be valid hex: %v", err))
		return
	}
	var formationHash [32]byte
	copy(formationHash[:], hashBytes)

	tbNum, ok := parseRequiredUint64Query(w, q, "t_b_numerator")
	if !ok {
		return
	}
	tbDen, ok := parseRequiredUint64Query(w, q, "t_b_denominator")
	if !ok {
		return
	}
	kcNum, ok := parseRequiredUint64Query(w, q, "k_c_numerator")
	if !ok {
		return
	}
	kcDen, ok := parseRequiredUint64Query(w, q, "k_c_denominator")
	if !ok {
		return
	}
	nWindow, ok := parseRequiredUint64Query(w, q, "n_window")
	if !ok {
		return
	}
	naDurationNs, ok := parseRequiredUint64Query(w, q, "n_a_duration_nanoseconds")
	if !ok {
		return
	}

	if tbDen == 0 {
		writeIngestError(w, http.StatusBadRequest, "t_b_denominator must be positive per §2.6 rational-pair invariant")
		return
	}
	if kcDen == 0 {
		writeIngestError(w, http.StatusBadRequest, "k_c_denominator must be positive per §2.6 rational-pair invariant")
		return
	}
	if nWindow == 0 {
		writeIngestError(w, http.StatusBadRequest, "n_window must be positive per §0138 W-count window form")
		return
	}

	params := &commonv1.LayerBParameters{
		TB:                    &commonv1.EvidentialIndependence{Numerator: tbNum, Denominator: tbDen},
		KC:                    &commonv1.EvidentialIndependence{Numerator: kcNum, Denominator: kcDen},
		NWindow:               nWindow,
		NADurationNanoseconds: naDurationNs,
	}

	verdict, err := layerb.Evaluate(r.Context(), h.sub, layerb.EvaluateOptions{
		FormationEventHash: formationHash,
		Params:             params,
	})
	if err != nil {
		writeIngestError(w, http.StatusInternalServerError, fmt.Sprintf("layerb.Evaluate: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(layerBVerdictPayload{
		FormationEventHashHex: hashHex,
		FreshnessBNumerator:   verdict.FreshnessBNumerator,
		FreshnessBDenominator: verdict.FreshnessBDenominator,
		SaturationCNumerator:  verdict.SaturationCNumerator,
		SaturationCDenominator: verdict.SaturationCDenominator,
		FreshnessUndefined:    verdict.FreshnessUndefined,
		FreshnessFired:        verdict.FreshnessFired,
		SaturationFired:       verdict.SaturationFired,
		Fired:                 verdict.Fired,
		WindowEventCount:      verdict.WindowEventCount,
		FilterMatchCount:      verdict.FilterMatchCount,
	})
}

// layerBVerdictPayload is the wire shape of the GET /v1/layer-b-verdict
// JSON response. Mirrors layerb.Verdict snake-case-converted + echoes
// the formation_event_hash_hex for operator-side correlation with
// subsequent demote requests.
type layerBVerdictPayload struct {
	FormationEventHashHex  string `json:"formation_event_hash_hex"`
	FreshnessBNumerator    uint64 `json:"freshness_b_numerator"`
	FreshnessBDenominator  uint64 `json:"freshness_b_denominator"`
	SaturationCNumerator   uint64 `json:"saturation_c_numerator"`
	SaturationCDenominator uint64 `json:"saturation_c_denominator"`
	FreshnessUndefined     bool   `json:"freshness_undefined"`
	FreshnessFired         bool   `json:"freshness_fired"`
	SaturationFired        bool   `json:"saturation_fired"`
	Fired                  bool   `json:"fired"`
	WindowEventCount       uint64 `json:"window_event_count"`
	FilterMatchCount       uint64 `json:"filter_match_count"`
}

// parseRequiredUint64Query extracts a REQUIRED uint64 query parameter.
// Missing or invalid → writes 400 + returns ok=false. Used for required
// numeric params on read-only diagnostic endpoints.
func parseRequiredUint64Query(w http.ResponseWriter, q map[string][]string, name string) (uint64, bool) {
	raw := ""
	if values, ok := q[name]; ok && len(values) > 0 {
		raw = values[0]
	}
	if raw == "" {
		writeIngestError(w, http.StatusBadRequest, fmt.Sprintf("%s is required", name))
		return 0, false
	}
	var v uint64
	if _, err := fmt.Sscanf(raw, "%d", &v); err != nil {
		writeIngestError(w, http.StatusBadRequest,
			fmt.Sprintf("%s must be a base-10 unsigned integer; got %q", name, raw))
		return 0, false
	}
	return v, true
}

