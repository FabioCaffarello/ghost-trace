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

	output := perSubtype{
		BehavioralCluster: bcCounts,
		AutomationGroup:   agCounts,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(output); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}

	fmt.Fprintf(os.Stderr,
		"summarize-hypotheses: bc_total=%d ag_total=%d after_ns=%d before_ns=%d\n",
		bcCounts.Total, agCounts.Total,
		*afterNs, *beforeNs)
	return nil
}

type perSubtype struct {
	BehavioralCluster projection.StateCounts `json:"behavioral_cluster"`
	AutomationGroup   projection.StateCounts `json:"automation_group"`
}
