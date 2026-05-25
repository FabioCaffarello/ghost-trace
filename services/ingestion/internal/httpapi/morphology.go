package httpapi

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/morphology"
)

// handleMorphology serves GET /v1/morphology per decision-log §0188.
// HTTP exposure of the morphology.Measure substrate walk; mirrors
// cmd/measure-chain-morphology over HTTP per §0093 CLI↔HTTP wire-parity
// precedent (verify endpoint pattern).
//
// Walks every CoordinationRing/CampaignHypothesis/BehavioralCluster/
// AutomationGroup formation event, computes per-hypothesis chain
// morphology (depth/breadth/closure via direct_influenced_by edges per
// Charter §2.4 frozen v0.5 + §0134-τ closure), aggregates §0143
// Sub-benchmark 1 chains-fracas vs chains-fortes diagnostic counters.
//
// Wire shape matches cmd/measure-chain-morphology's stdout JSON
// envelope so operators get identical bytes regardless of channel.
//
// Response codes:
//   - 200 — JSON measurement envelope (always; empty hypotheses array
//     when no formations present).
//   - 405 — non-GET method.
//   - 500 — substrate walk error.
//   - 503 — substrate not configured on this handler (the
//     WithSubstrate option was not supplied at construction).
//
// Operational cost: this endpoint walks every substrate formation row.
// Bounded by N + E (formation count + influenced_by edge count) — fast
// even on large substrates (memoization caches per-hash depth). Safe
// to invoke from interactive dashboards; concurrent invocations safe
// under SQLite WAL.
func (h *Handler) handleMorphology(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.sub == nil {
		writeIngestError(w, http.StatusServiceUnavailable,
			"morphology endpoint not configured (handler constructed without WithSubstrate)")
		return
	}

	m, err := morphology.Measure(r.Context(), h.sub)
	if err != nil {
		writeIngestError(w, http.StatusInternalServerError, fmt.Sprintf("morphology.Measure: %v", err))
		return
	}

	env := morphologyPayload{
		Hypotheses: make([]morphologyHypothesisPayload, 0, len(m.Hypotheses)),
		Stats: morphologyStatsPayload{
			TotalFormations:   m.Stats.TotalFormations,
			PerSubtype:        m.Stats.PerSubtype,
			DepthHistogram:    m.Stats.DepthHistogram,
			BreadthHistogram:  m.Stats.BreadthHistogram,
			ChainsFracasCount: m.Stats.ChainsFracasCount,
			ChainsFortesCount: m.Stats.ChainsFortesCount,
		},
	}
	for _, hp := range m.Hypotheses {
		env.Hypotheses = append(env.Hypotheses, morphologyHypothesisPayload{
			HypothesisHashHex:  hex.EncodeToString(hp.HypothesisHash[:]),
			SubtypeName:        hp.SubtypeName,
			ActorRefs:          hp.ActorRefs,
			ChainDepthMax:      hp.ChainDepthMax,
			ChainBreadthAtRoot: hp.ChainBreadthAtRoot,
			ClosureCount:       hp.ClosureCount,
			SourceEventCount:   hp.SourceEventCount,
			FormationAt:        hp.FormationAt,
		})
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(env)
}

// morphologyHypothesisPayload is the per-hypothesis wire shape. Mirrors
// cmd/measure-chain-morphology's hypothesisJSON exactly per §0093 +
// §0163 CLI↔HTTP wire-parity discipline.
type morphologyHypothesisPayload struct {
	HypothesisHashHex  string   `json:"hypothesis_hash_hex"`
	SubtypeName        string   `json:"subtype_name"`
	ActorRefs          []string `json:"actor_refs"`
	ChainDepthMax      uint32   `json:"chain_depth_max"`
	ChainBreadthAtRoot uint32   `json:"chain_breadth_at_root"`
	ClosureCount       uint32   `json:"closure_count"`
	SourceEventCount   uint32   `json:"source_event_count"`
	FormationAt        int64    `json:"formation_at"`
}

// morphologyStatsPayload is the aggregate-stats wire shape. Mirrors
// cmd/measure-chain-morphology's statsJSON exactly.
type morphologyStatsPayload struct {
	TotalFormations   uint32            `json:"total_formations"`
	PerSubtype        map[string]uint32 `json:"per_subtype"`
	DepthHistogram    map[uint32]uint32 `json:"depth_histogram"`
	BreadthHistogram  map[uint32]uint32 `json:"breadth_histogram"`
	ChainsFracasCount uint32            `json:"chains_fracas_count"`
	ChainsFortesCount uint32            `json:"chains_fortes_count"`
}

// morphologyPayload is the response envelope wire shape. Mirrors
// cmd/measure-chain-morphology's emissionEnvelope exactly.
type morphologyPayload struct {
	Hypotheses []morphologyHypothesisPayload `json:"hypotheses"`
	Stats      morphologyStatsPayload        `json:"stats"`
}
