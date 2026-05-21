// Command replay-all-formations performs substrate-wide Phase 3
// reconstructive replay across the four Cat III subtype formations
// per decision-log §0090. Default: replays every formation of every
// subtype. Optional --subtype filter narrows to one subtype.
//
// Output: JSON with per-subtype BatchReplayReport sections.
//
// Exit codes:
//   0 every formation across the selected subtype(s) matched.
//   1 at least one drift detected.
//   2 tool/config error.
//   3 no drift but at least one substrate-precondition error.
//
// Distinguishing drift from precondition error mirrors the per-target
// and Phase-1-batch (§0085) CLI conventions.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/replay"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

const (
	exitToolError       = 2
	exitTargetIntegrity = 3
	exitDriftDetected   = 1
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "replay-all-formations: %v\n", err)
		os.Exit(exitToolError)
	}
}

func run() error {
	dbPath := flag.String("db", "./ghost-trace.db", "SQLite primary-event-log path")
	blobDir := flag.String("blobs", "./blobs", "content-addressed blob-store directory")
	subtypeFilter := flag.String("subtype", "", "filter by Cat III subtype: behavioral_cluster|automation_group|campaign_hypothesis|coordination_ring (empty = all)")
	flag.Parse()

	if *subtypeFilter != "" &&
		*subtypeFilter != "behavioral_cluster" &&
		*subtypeFilter != "automation_group" &&
		*subtypeFilter != "campaign_hypothesis" &&
		*subtypeFilter != "coordination_ring" {
		return fmt.Errorf("--subtype %q is not one of: behavioral_cluster, automation_group, campaign_hypothesis, coordination_ring", *subtypeFilter)
	}

	ctx := context.Background()
	sub, err := substrate.Open(ctx, *dbPath, *blobDir)
	if err != nil {
		return fmt.Errorf("open substrate: %w", err)
	}
	defer func() { _ = sub.Close() }()

	out := output{}

	if *subtypeFilter == "" || *subtypeFilter == "behavioral_cluster" {
		rep, err := replay.ReplayAllBehavioralClusterFormations(ctx, sub)
		if err != nil {
			return fmt.Errorf("BC: %w", err)
		}
		out.BehavioralCluster = &rep
	}
	if *subtypeFilter == "" || *subtypeFilter == "automation_group" {
		rep, err := replay.ReplayAllAutomationGroupFormations(ctx, sub)
		if err != nil {
			return fmt.Errorf("AG: %w", err)
		}
		out.AutomationGroup = &rep
	}
	if *subtypeFilter == "" || *subtypeFilter == "campaign_hypothesis" {
		rep, err := replay.ReplayAllCampaignHypothesisFormations(ctx, sub)
		if err != nil {
			return fmt.Errorf("CH: %w", err)
		}
		out.CampaignHypothesis = &rep
	}
	if *subtypeFilter == "" || *subtypeFilter == "coordination_ring" {
		rep, err := replay.ReplayAllCoordinationRingFormations(ctx, sub)
		if err != nil {
			return fmt.Errorf("CR: %w", err)
		}
		out.CoordinationRing = &rep
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}

	totalDrift, totalErrored := 0, 0
	totalMatched, totalCount := 0, 0
	for _, r := range []*replay.BatchReplayReport{out.BehavioralCluster, out.AutomationGroup, out.CampaignHypothesis, out.CoordinationRing} {
		if r == nil {
			continue
		}
		totalCount += r.Total
		totalMatched += r.Matched
		totalDrift += r.Drifted
		totalErrored += r.Errored
	}

	fmt.Fprintf(os.Stderr,
		"replay-all-formations: subtype_filter=%q total=%d matched=%d drifted=%d errored=%d\n",
		*subtypeFilter, totalCount, totalMatched, totalDrift, totalErrored)

	switch {
	case totalDrift > 0:
		os.Exit(exitDriftDetected)
	case totalErrored > 0:
		os.Exit(exitTargetIntegrity)
	}
	return nil
}

type output struct {
	BehavioralCluster  *replay.BatchReplayReport `json:"behavioral_cluster,omitempty"`
	AutomationGroup    *replay.BatchReplayReport `json:"automation_group,omitempty"`
	CampaignHypothesis *replay.BatchReplayReport `json:"campaign_hypothesis,omitempty"`
	CoordinationRing   *replay.BatchReplayReport `json:"coordination_ring,omitempty"`
}
