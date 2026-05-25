// Command replay-all-derived-actor-attributions performs substrate-
// wide batch Phase 1 replay over every DerivedActorAttribution in
// the substrate per decision-log §0173. Aggregates per-target
// outcomes into a summary; lists any drifted or errored entries.
//
// Mirrors replay-all-operational-sessions on the attribution Cat II
// side; completes the §0168 → §0171 → §0172 → §0173 chain: Cat II
// construct → library replay → CLI replay → batch CLI replay.
//
// Output: structured JSON to stdout with Total, Matched, Drifted,
// Errored counts + Drift / Errors slices. Same shape as
// replay-all-operational-sessions per §0163 stable-wire-contract
// discipline.
//
// Exit codes (identical shape to replay-all-operational-sessions):
//
//	0 success AND zero drift AND zero error
//	1 success AND any drift detected (Phase 1 contract violated)
//	2 tool/config error
//	3 success AND zero drift AND any error (precondition failure on
//	  at least one record)
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
		fmt.Fprintf(os.Stderr, "replay-all-derived-actor-attributions: %v\n", err)
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

	report, err := replay.ReplayAllDerivedActorAttributions(ctx, sub)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}

	fmt.Fprintf(os.Stderr,
		"replay-all-derived-actor-attributions: total=%d matched=%d drifted=%d errored=%d\n",
		report.Total, report.Matched, report.Drifted, report.Errored)

	switch {
	case report.Drifted > 0:
		os.Exit(exitDriftDetected)
	case report.Errored > 0:
		os.Exit(exitTargetIntegrity)
	}
	return nil
}
