// Command summarize-hypotheses returns aggregate counters + latency
// aggregates over the projection of every Category III hypothesis in
// the substrate, per Charter §2.5 BC3 + decision-log §0053 +
// decision-log §0078 (combined cross-subtype counters) + decision-log
// §0079 (per-subtype + combined latency aggregates).
//
// Output: structured JSON to stdout with a top-level `combined`
// section and four per-subtype sections (`behavioral_cluster`,
// `automation_group`, `campaign_hypothesis`, `coordination_ring`).
// Each section carries `total`, `by_state` (every State key present;
// predictable wire shape per §0053), and `latencies` (per-dimension
// LatencyAggregate per §0079).
//
// Per the §0053 equivalence invariant: for every State value, the
// count in each section's `by_state` equals the length of
// `list-hypotheses -state <state> -subtype <subtype>` (or no subtype
// filter for the combined section).
//
// Per §0079 the combined latency aggregate is computed from the
// UNION of per-subtype samples — not approximated from the per-subtype
// aggregates — so combined percentiles are exact.
//
// Per §0083, the aggregation helpers live in internal/projection
// and are shared with internal/httpapi/hypotheses_summary.
//
// Exit codes: 0 success (including empty substrate); 2 tool/config
// error.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/projection"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

const exitToolError = 2

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "summarize-hypotheses: %v\n", err)
		os.Exit(exitToolError)
	}
}

func run() error {
	dbPath := flag.String("db", "./ghost-trace.db", "SQLite primary-event-log path")
	blobDir := flag.String("blobs", "./blobs", "content-addressed blob-store directory")
	afterNs := flag.Int64("after-ns", 0, "inclusive lower bound (Unix nanoseconds) on the latest event_time of each projection; 0 disables the lower bound")
	beforeNs := flag.Int64("before-ns", 0, "inclusive upper bound (Unix nanoseconds) on the latest event_time of each projection; 0 disables the upper bound")
	flag.Parse()

	if *afterNs < 0 {
		return fmt.Errorf("--after-ns must be non-negative; got %d", *afterNs)
	}
	if *beforeNs < 0 {
		return fmt.Errorf("--before-ns must be non-negative; got %d", *beforeNs)
	}
	if *afterNs != 0 && *beforeNs != 0 && *afterNs > *beforeNs {
		return fmt.Errorf("--after-ns (%d) must not exceed --before-ns (%d)", *afterNs, *beforeNs)
	}

	ctx := context.Background()
	sub, err := substrate.Open(ctx, *dbPath, *blobDir)
	if err != nil {
		return fmt.Errorf("open substrate: %w", err)
	}
	defer func() { _ = sub.Close() }()

	bcProjs, err := projection.ListHypotheses(ctx, sub, projection.ListOptions{
		TimeAfterNs:  *afterNs,
		TimeBeforeNs: *beforeNs,
	})
	if err != nil {
		return err
	}
	agProjs, err := projection.ListAutomationGroups(ctx, sub, projection.AutomationGroupListOptions{
		TimeAfterNs:  *afterNs,
		TimeBeforeNs: *beforeNs,
	})
	if err != nil {
		return err
	}
	chProjs, err := projection.ListCampaignHypotheses(ctx, sub, projection.CampaignHypothesisListOptions{
		TimeAfterNs:  *afterNs,
		TimeBeforeNs: *beforeNs,
	})
	if err != nil {
		return err
	}
	crProjs, err := projection.ListCoordinationRings(ctx, sub, projection.CoordinationRingListOptions{
		TimeAfterNs:  *afterNs,
		TimeBeforeNs: *beforeNs,
	})
	if err != nil {
		return err
	}

	bcAgg := projection.AggregateBC(bcProjs)
	agAgg := projection.AggregateAG(agProjs)
	chAgg := projection.AggregateCH(chProjs)
	crAgg := projection.AggregateCR(crProjs)

	combinedCounts := projection.CombineCounts(
		bcAgg.Counts, agAgg.Counts, chAgg.Counts, crAgg.Counts,
	)
	combinedSamples := projection.CombineLatencySamples(
		bcAgg.Samples, agAgg.Samples, chAgg.Samples, crAgg.Samples,
	)

	output := perSubtype{
		Combined: projection.AggregateSection{
			StateCounts: combinedCounts,
			Latencies:   projection.AggregateAllLatencies(combinedSamples),
		},
		BehavioralCluster:  bcAgg.Section(),
		AutomationGroup:    agAgg.Section(),
		CampaignHypothesis: chAgg.Section(),
		CoordinationRing:   crAgg.Section(),
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(output); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}

	fmt.Fprintf(os.Stderr,
		"summarize-hypotheses: combined_total=%d bc_total=%d ag_total=%d ch_total=%d cr_total=%d after_ns=%d before_ns=%d\n",
		combinedCounts.Total, bcAgg.Counts.Total, agAgg.Counts.Total,
		chAgg.Counts.Total, crAgg.Counts.Total,
		*afterNs, *beforeNs)
	return nil
}

type perSubtype struct {
	Combined           projection.AggregateSection `json:"combined"`
	BehavioralCluster  projection.AggregateSection `json:"behavioral_cluster"`
	AutomationGroup    projection.AggregateSection `json:"automation_group"`
	CampaignHypothesis projection.AggregateSection `json:"campaign_hypothesis"`
	CoordinationRing   projection.AggregateSection `json:"coordination_ring"`
}
