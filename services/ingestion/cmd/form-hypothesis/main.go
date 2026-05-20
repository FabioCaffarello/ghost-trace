// Command form-hypothesis runs the first Category III hypothesis-
// formation pathway per decision-log §0045 — first Cat III subtype
// landing (BehavioralCluster). Wraps the internal/hypothesis
// package's FormAll entry point with command-line configuration +
// structured output.
//
// Operationally:
//
//   - Run after an ingestion window when operators want to materialize
//     the current formation pattern's view of behavioral clusters.
//   - Re-run is idempotent under the same (pattern_signature,
//     pattern_parameters) tuple: zero new substrate rows produced
//     (content-addressed immutability rejects duplicates).
//   - A change to either tuple element produces NEW formation events
//     alongside the prior records per Charter §2.5 + §0011 (lifecycle
//     events are immutable; the operation history is the source-of-
//     record).
//
// Output: structured JSON to stdout + a brief human summary to stderr.
// Exit code: 0 on success; 2 on tool / configuration error.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/hypothesis"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "form-hypothesis: %v\n", err)
		os.Exit(2)
	}
}

func run() error {
	dbPath := flag.String("db", "./ghost-trace.db", "SQLite primary-event-log path")
	blobDir := flag.String("blobs", "./blobs", "content-addressed blob-store directory")
	patternSignature := flag.String("pattern-signature", hypothesis.SessionDescriptorSharedV1Signature, "formation pattern signature identifier")
	minClusterSize := flag.Int64("min-cluster-size", 2, "session-descriptor-shared-v1 parameter: minimum distinct actor_refs for a cluster to form")
	flag.Parse()

	pattern, err := resolvePattern(*patternSignature, *minClusterSize)
	if err != nil {
		return err
	}

	ctx := context.Background()
	sub, err := substrate.Open(ctx, *dbPath, *blobDir)
	if err != nil {
		return fmt.Errorf("open substrate: %w", err)
	}
	defer func() { _ = sub.Close() }()

	report, err := hypothesis.FormAll(ctx, sub, pattern, time.Now)
	if err != nil {
		return fmt.Errorf("form: %w", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(payload{
		PatternSignature:  pattern.Signature(),
		PatternParameters: pattern.Parameters(),
		Examined:          report.Examined,
		NewlyFormed:       report.NewlyFormed,
		AlreadyFormed:     report.AlreadyFormed,
	}); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}

	fmt.Fprintf(os.Stderr,
		"form-hypothesis: pattern=%s params=%q examined=%d newly_formed=%d already_formed=%d\n",
		pattern.Signature(), pattern.Parameters(), report.Examined, report.NewlyFormed, report.AlreadyFormed)
	return nil
}

// resolvePattern selects a FormationPattern by signature + binds its
// parameters. New patterns register here.
func resolvePattern(signature string, minClusterSize int64) (hypothesis.FormationPattern, error) {
	switch signature {
	case hypothesis.SessionDescriptorSharedV1Signature:
		return hypothesis.SessionDescriptorSharedV1{MinClusterSize: minClusterSize}, nil
	default:
		return nil, fmt.Errorf("unknown pattern-signature %q; known signatures: %s",
			signature, hypothesis.SessionDescriptorSharedV1Signature)
	}
}

type payload struct {
	PatternSignature  string `json:"pattern_signature"`
	PatternParameters string `json:"pattern_parameters"`
	Examined          int64  `json:"examined"`
	NewlyFormed       int64  `json:"newly_formed"`
	AlreadyFormed     int64  `json:"already_formed"`
}
