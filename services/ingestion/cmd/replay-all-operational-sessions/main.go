// Command replay-all-operational-sessions performs substrate-wide
// batch Phase 1 replay over every OperationalSession in the substrate
// per decision-log §0085. Aggregates per-target outcomes into a
// summary; lists any drifted or errored entries.
//
// Output: structured JSON to stdout with Total, Matched, Drifted,
// Errored counts + Drift / Errors slices.
//
// Exit codes:
//   0 success AND zero drift AND zero error (every OS replays cleanly)
//   1 success AND any drift detected (Phase 1 contract violated)
//   2 tool/config error
//   3 success AND zero drift AND any error (precondition failure on
//     at least one record)
//
// Distinguishing exit 1 from exit 3:
//   - exit 1: replays completed, hashes diverged → operational-
//     definition drift since commit. Action: investigate the
//     definition implementation.
//   - exit 3: replays could NOT complete for at least one record
//     (missing source, unknown definition version, etc.). Action:
//     investigate substrate consistency.
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
		fmt.Fprintf(os.Stderr, "replay-all-operational-sessions: %v\n", err)
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

	report, err := replay.ReplayAllOperationalSessions(ctx, sub)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}

	fmt.Fprintf(os.Stderr,
		"replay-all-operational-sessions: total=%d matched=%d drifted=%d errored=%d\n",
		report.Total, report.Matched, report.Drifted, report.Errored)

	switch {
	case report.Drifted > 0:
		os.Exit(exitDriftDetected)
	case report.Errored > 0:
		os.Exit(exitTargetIntegrity)
	}
	return nil
}
