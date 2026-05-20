// Command hypothesis-state materializes the current-state
// projection of a Category III BehavioralCluster hypothesis chain
// per Charter §2.5 BC3 + decision-log §0051. Wraps
// internal/projection.ProjectHypothesis with command-line input +
// structured JSON output.
//
// §2.5 BC3 forbids storing "current state" as a substrate row —
// the substrate stores immutable lifecycle events (formation,
// promotion, demotion, dissolution, merge, split) and the
// hypothesis's current state is reconstructed by replaying those
// events. This binary is the first operator-facing materializer
// of that read-side projection: given a formation hash, it walks
// the substrate, builds the projection, and emits the result.
//
// Scope per §0051: single-hypothesis projection; on-demand walk;
// no caching. Multi-hypothesis aggregate queries are a follow-on
// landing.
//
// Output: structured JSON to stdout + brief human summary to
// stderr. Exit codes: 0 success; 2 tool/config error; 3
// formation-not-found OR target-not-formation.
package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/projection"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

const (
	exitToolError       = 2
	exitTargetIntegrity = 3
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "hypothesis-state: %v\n", err)
		if errors.Is(err, projection.ErrFormationNotFound) || errors.Is(err, projection.ErrTargetNotFormation) {
			os.Exit(exitTargetIntegrity)
		}
		os.Exit(exitToolError)
	}
}

func run() error {
	dbPath := flag.String("db", "./ghost-trace.db", "SQLite primary-event-log path")
	blobDir := flag.String("blobs", "./blobs", "content-addressed blob-store directory")
	formationHex := flag.String("formation-event-hash", "", "REQUIRED: hex-encoded BLAKE3-256 content-hash of the target BehavioralClusterFormation")
	flag.Parse()

	if *formationHex == "" {
		return fmt.Errorf("--formation-event-hash is required")
	}
	raw, err := hex.DecodeString(*formationHex)
	if err != nil {
		return fmt.Errorf("decode --formation-event-hash: %w", err)
	}
	if len(raw) != 32 {
		return fmt.Errorf("--formation-event-hash must be 32 bytes (64 hex chars); got %d bytes", len(raw))
	}
	var formationHash [32]byte
	copy(formationHash[:], raw)

	ctx := context.Background()
	sub, err := substrate.Open(ctx, *dbPath, *blobDir)
	if err != nil {
		return fmt.Errorf("open substrate: %w", err)
	}
	defer func() { _ = sub.Close() }()

	proj, err := projection.ProjectHypothesis(ctx, sub, formationHash)
	if err != nil {
		return err
	}

	out := buildOutput(*formationHex, proj)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}

	fmt.Fprintf(os.Stderr,
		"hypothesis-state: formation=%s state=%s lifecycle_events=%d\n",
		*formationHex, proj.State, len(proj.LifecycleHistory))
	return nil
}

type lifecycleEntry struct {
	Type      string `json:"type"`
	EventHash string `json:"event_hash"`
	EventTime int64  `json:"event_time"`
}

type output struct {
	FormationHash    string           `json:"formation_event_hash"`
	State            string           `json:"state"`
	LatestPromotion  *promotionView   `json:"latest_promotion,omitempty"`
	LatestDemotion   *demotionView    `json:"latest_demotion,omitempty"`
	Dissolution      *dissolutionView `json:"dissolution,omitempty"`
	MergedInto       *mergeView       `json:"merged_into,omitempty"`
	SplitInto        *splitView       `json:"split_into,omitempty"`
	LifecycleHistory []lifecycleEntry `json:"lifecycle_history"`
	Latencies        latencyView      `json:"latencies"`
}

type latencyView struct {
	FormationToFirstPromotionNs        *int64 `json:"formation_to_first_promotion_ns,omitempty"`
	LatestPromotionToLatestDemotionNs  *int64 `json:"latest_promotion_to_latest_demotion_ns,omitempty"`
	FormationToDissolutionNs           *int64 `json:"formation_to_dissolution_ns,omitempty"`
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
	SplitAt        int64    `json:"split_at"`
	SuccessorHashes []string `json:"successor_formation_event_hashes"`
	Reason         string   `json:"reason,omitempty"`
}

func buildOutput(formationHex string, p projection.HypothesisProjection) output {
	o := output{
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
		o.LifecycleHistory = append(o.LifecycleHistory, lifecycleEntry{
			Type:      entry.Type,
			EventHash: hex.EncodeToString(entry.EventHash[:]),
			EventTime: entry.EventTime,
		})
	}
	o.Latencies = latencyView{
		FormationToFirstPromotionNs:       p.FormationToFirstPromotionLatencyNs,
		LatestPromotionToLatestDemotionNs: p.LatestPromotionToLatestDemotionLatencyNs,
		FormationToDissolutionNs:          p.FormationToDissolutionLatencyNs,
	}
	return o
}
