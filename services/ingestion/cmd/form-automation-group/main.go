// Command form-automation-group runs the AutomationGroup formation
// pathway per decision-log §0056 — opens the second concrete Cat III
// subtype lifecycle arc (§0045 opened BehavioralCluster). Wraps the
// internal/hypothesis package's FormAutomationGroupAll entry point
// with command-line configuration + structured output.
//
// Subtype distinction (per entity-model.md): an AutomationGroup is
// "a set of actors whose behavioral patterns match a signature of
// automated (non-human) operation". This binary's inception pattern
// (uniform-cadence-v1) detects actors whose DeclaredSession inter-
// event intervals have a low coefficient of variation — the
// behavioral signature of mechanical (non-human) operation.
//
// Operationally:
//
//   - Run after an ingestion window when operators want to materialize
//     the current pattern's view of automated actors.
//   - Re-run is idempotent under the same (pattern_signature,
//     pattern_parameters) tuple — content-addressed immutability
//     rejects duplicates per §0027 AP6.
//   - A change to either tuple element produces NEW formation events
//     alongside the prior per Charter §2.5.
//
// Output: structured JSON to stdout + brief human summary to stderr.
// Exit code: 0 success; 2 tool / configuration error.
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
		fmt.Fprintf(os.Stderr, "form-automation-group: %v\n", err)
		os.Exit(2)
	}
}

func run() error {
	dbPath := flag.String("db", "./ghost-trace.db", "SQLite primary-event-log path")
	blobDir := flag.String("blobs", "./blobs", "content-addressed blob-store directory")
	patternSignature := flag.String("pattern-signature", hypothesis.UniformCadenceV1Signature, "formation pattern signature identifier")
	minObservationCount := flag.Int64("min-observation-count", 5, "uniform-cadence-v1: minimum DeclaredSessions per actor for cadence evaluation")
	maxCovThreshold := flag.Float64("max-cov-threshold", 0.15, "uniform-cadence-v1: coefficient-of-variation ceiling below which an actor matches the automation signature")
	actor := flag.String("actor", "", "OPTIONAL per decision-log §0097 + §0111: when non-empty, pairs each formed event with an IngestionEvent for per-actor attribution. Empty preserves the single-Append path.")
	flag.Parse()

	pattern, err := resolvePattern(*patternSignature, *minObservationCount, *maxCovThreshold)
	if err != nil {
		return err
	}

	ctx := context.Background()
	sub, err := substrate.Open(ctx, *dbPath, *blobDir)
	if err != nil {
		return fmt.Errorf("open substrate: %w", err)
	}
	defer func() { _ = sub.Close() }()

	report, err := hypothesis.FormAutomationGroupAllWithActor(ctx, sub, pattern, time.Now, *actor)
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

	if *actor != "" {
		fmt.Fprintf(os.Stderr,
			"form-automation-group: pattern=%s params=%q examined=%d newly_formed=%d already_formed=%d actor=%q\n",
			pattern.Signature(), pattern.Parameters(), report.Examined, report.NewlyFormed, report.AlreadyFormed, *actor)
	} else {
		fmt.Fprintf(os.Stderr,
			"form-automation-group: pattern=%s params=%q examined=%d newly_formed=%d already_formed=%d\n",
			pattern.Signature(), pattern.Parameters(), report.Examined, report.NewlyFormed, report.AlreadyFormed)
	}
	return nil
}

// resolvePattern selects an AutomationGroupFormationPattern by
// signature + binds its parameters. New patterns register here.
func resolvePattern(signature string, minObservationCount int64, maxCovThreshold float64) (hypothesis.AutomationGroupFormationPattern, error) {
	switch signature {
	case hypothesis.UniformCadenceV1Signature:
		return hypothesis.UniformCadenceV1{
			MinObservationCount: minObservationCount,
			MaxCoVThreshold:     maxCovThreshold,
		}, nil
	default:
		return nil, fmt.Errorf("unknown pattern-signature %q; known signatures: %s",
			signature, hypothesis.UniformCadenceV1Signature)
	}
}

type payload struct {
	PatternSignature  string `json:"pattern_signature"`
	PatternParameters string `json:"pattern_parameters"`
	Examined          int64  `json:"examined"`
	NewlyFormed       int64  `json:"newly_formed"`
	AlreadyFormed     int64  `json:"already_formed"`
}
