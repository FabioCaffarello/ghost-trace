// Command measure-chain-morphology is the substrate-layer
// instrumentation companion to §0154's signature-layer instrumentation
// per §0155. Reads Cat III formation events from substrate, computes
// per-hypothesis chain morphology (chain_depth_max +
// chain_breadth_at_root per §0143 Sub-benchmark 1 definition), emits
// per-hypothesis morphology + aggregate stats as JSON for operator +
// Sub-benchmark 1 diagnostic use.
//
// Per §0143 Sub-benchmark 1 diagnostic distinction:
//   - chains-fracas (depth_max <= 2 OR breadth_at_root <= 3) →
//     indicates F3 insufficient
//   - chains-fortes (depth_max > 2 AND breadth_at_root > 3) →
//     indicates constitutional thesis confirmed (domain rotates
//     evidence faster than decay; substrate-level)
//
// Read-only over substrate; no AppendPair invocation; safe concurrent
// with ingestion paths.
//
// Exit codes: 0 success (including empty hypothesis set); 2
// tool/config error.
package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/morphology"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

const exitToolError = 2

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "measure-chain-morphology: %v\n", err)
		os.Exit(exitToolError)
	}
}

func run() error {
	dbPath := flag.String("db", "./ghost-trace.db", "SQLite primary-event-log path")
	blobDir := flag.String("blobs", "./blobs", "content-addressed blob-store directory")
	flag.Parse()

	ctx := context.Background()
	sub, err := substrate.Open(ctx, *dbPath, *blobDir)
	if err != nil {
		return fmt.Errorf("substrate.Open: %w", err)
	}
	defer sub.Close()

	m, err := morphology.Measure(ctx, sub)
	if err != nil {
		return fmt.Errorf("morphology.Measure: %w", err)
	}
	if err := emitJSON(os.Stdout, m); err != nil {
		return fmt.Errorf("emit JSON: %w", err)
	}
	fmt.Fprintf(os.Stderr, "measure-chain-morphology: total=%d formations; chains_fracas=%d chains_fortes=%d (depth_max<=2 OR breadth<=3 = fracas)\n",
		m.Stats.TotalFormations, m.Stats.ChainsFracasCount, m.Stats.ChainsFortesCount)
	return nil
}

// hypothesisJSON is the operator-facing per-hypothesis serialization.
// Distinct from morphology.ChainMorphology per §0153 MO4 (decoupled
// CLI-package-local types from package-internal representation).
type hypothesisJSON struct {
	HypothesisHashHex  string   `json:"hypothesis_hash_hex"`
	SubtypeName        string   `json:"subtype_name"`
	ActorRefs          []string `json:"actor_refs"`
	ChainDepthMax      uint32   `json:"chain_depth_max"`
	ChainBreadthAtRoot uint32   `json:"chain_breadth_at_root"`
	ClosureCount       uint32   `json:"closure_count"`
	SourceEventCount   uint32   `json:"source_event_count"`
	FormationAt        int64    `json:"formation_at"`
}

type statsJSON struct {
	TotalFormations   uint32            `json:"total_formations"`
	PerSubtype        map[string]uint32 `json:"per_subtype"`
	DepthHistogram    map[uint32]uint32 `json:"depth_histogram"`
	BreadthHistogram  map[uint32]uint32 `json:"breadth_histogram"`
	ChainsFracasCount uint32            `json:"chains_fracas_count"`
	ChainsFortesCount uint32            `json:"chains_fortes_count"`
}

type emissionEnvelope struct {
	Hypotheses []hypothesisJSON `json:"hypotheses"`
	Stats      statsJSON        `json:"stats"`
}

func emitJSON(w *os.File, m *morphology.Measurement) error {
	env := emissionEnvelope{
		Hypotheses: make([]hypothesisJSON, 0, len(m.Hypotheses)),
		Stats: statsJSON{
			TotalFormations:   m.Stats.TotalFormations,
			PerSubtype:        m.Stats.PerSubtype,
			DepthHistogram:    m.Stats.DepthHistogram,
			BreadthHistogram:  m.Stats.BreadthHistogram,
			ChainsFracasCount: m.Stats.ChainsFracasCount,
			ChainsFortesCount: m.Stats.ChainsFortesCount,
		},
	}
	for _, h := range m.Hypotheses {
		env.Hypotheses = append(env.Hypotheses, hypothesisJSON{
			HypothesisHashHex:  hex.EncodeToString(h.HypothesisHash[:]),
			SubtypeName:        h.SubtypeName,
			ActorRefs:          h.ActorRefs,
			ChainDepthMax:      h.ChainDepthMax,
			ChainBreadthAtRoot: h.ChainBreadthAtRoot,
			ClosureCount:       h.ClosureCount,
			SourceEventCount:   h.SourceEventCount,
			FormationAt:        h.FormationAt,
		})
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(env)
}
