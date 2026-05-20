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
	flag.Parse()

	ctx := context.Background()
	sub, err := substrate.Open(ctx, *dbPath, *blobDir)
	if err != nil {
		return fmt.Errorf("open substrate: %w", err)
	}
	defer func() { _ = sub.Close() }()

	counts, err := projection.CountByState(ctx, sub)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(counts); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}

	fmt.Fprintf(os.Stderr,
		"summarize-hypotheses: total=%d forming=%d promoted=%d demoted=%d dissolved=%d merged_into=%d split_into=%d\n",
		counts.Total,
		counts.ByState[projection.StateForming],
		counts.ByState[projection.StatePromoted],
		counts.ByState[projection.StateDemoted],
		counts.ByState[projection.StateDissolved],
		counts.ByState[projection.StateMergedInto],
		counts.ByState[projection.StateSplitInto])
	return nil
}
