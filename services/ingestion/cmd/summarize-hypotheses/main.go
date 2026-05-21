// Command summarize-hypotheses returns aggregate counters over the
// projection of every Category III BehavioralCluster hypothesis in
// the substrate, per Charter §2.5 BC3 + decision-log §0053. Wraps
// internal/projection.CountByState with structured JSON output.
//
// Discharges the §0052 named carry-forward "Aggregate counters /
// histograms". This is the third binary in the read-only
// classification (after `hypothesis-state` from §0051 and
// `list-hypotheses` from §0052) and the 12th operational binary
// overall.
//
// Output: structured JSON to stdout with two fields: `total`
// (number of formations in the substrate) and `by_state` (a map
// from State value to count; every State key is present, even at
// zero, to keep the wire shape predictable).
//
// Per the §0053 equivalence invariant: for every State value, the
// count returned here equals the length of
// `list-hypotheses -state <state>`.
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

	bcCounts, err := projection.CountByState(ctx, sub, projection.ListOptions{
		TimeAfterNs:  *afterNs,
		TimeBeforeNs: *beforeNs,
	})
	if err != nil {
		return err
	}
	agCounts, err := projection.CountAutomationGroupsByState(ctx, sub, projection.AutomationGroupListOptions{
		TimeAfterNs:  *afterNs,
		TimeBeforeNs: *beforeNs,
	})
	if err != nil {
		return err
	}
	chCounts, err := projection.CountCampaignHypothesesByState(ctx, sub, projection.CampaignHypothesisListOptions{
		TimeAfterNs:  *afterNs,
		TimeBeforeNs: *beforeNs,
	})
	if err != nil {
		return err
	}
	crCounts, err := projection.CountCoordinationRingsByState(ctx, sub, projection.CoordinationRingListOptions{
		TimeAfterNs:  *afterNs,
		TimeBeforeNs: *beforeNs,
	})
	if err != nil {
		return err
	}

	combined := combineCounts(bcCounts, agCounts, chCounts, crCounts)

	output := perSubtype{
		Combined:           combined,
		BehavioralCluster:  bcCounts,
		AutomationGroup:    agCounts,
		CampaignHypothesis: chCounts,
		CoordinationRing:   crCounts,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(output); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}

	fmt.Fprintf(os.Stderr,
		"summarize-hypotheses: combined_total=%d bc_total=%d ag_total=%d ch_total=%d cr_total=%d after_ns=%d before_ns=%d\n",
		combined.Total, bcCounts.Total, agCounts.Total, chCounts.Total, crCounts.Total,
		*afterNs, *beforeNs)
	return nil
}

// combineCounts returns a per-state-aligned sum across all four
// per-subtype StateCounts. Each by_state[s] in the combined output
// equals the sum of each subtype's by_state[s]; Total equals the
// sum of subtype Totals. Every State enum value is present (same
// predictable-wire-shape commitment as the per-subtype outputs).
func combineCounts(parts ...projection.StateCounts) projection.StateCounts {
	combined := projection.StateCounts{
		ByState: map[projection.State]int{
			projection.StateForming:    0,
			projection.StatePromoted:   0,
			projection.StateDemoted:    0,
			projection.StateDissolved:  0,
			projection.StateMergedInto: 0,
			projection.StateSplitInto:  0,
		},
	}
	for _, p := range parts {
		combined.Total += p.Total
		for state, count := range p.ByState {
			combined.ByState[state] += count
		}
	}
	return combined
}

type perSubtype struct {
	Combined           projection.StateCounts `json:"combined"`
	BehavioralCluster  projection.StateCounts `json:"behavioral_cluster"`
	AutomationGroup    projection.StateCounts `json:"automation_group"`
	CampaignHypothesis projection.StateCounts `json:"campaign_hypothesis"`
	CoordinationRing   projection.StateCounts `json:"coordination_ring"`
}
