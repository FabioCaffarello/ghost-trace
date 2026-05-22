// Command form-coordination-ring runs the CoordinationRing
// formation pathway per decision-log §0070 — opens the FOURTH (and
// final) Cat III concrete subtype lifecycle arc (§0045
// BehavioralCluster; §0056 AutomationGroup; §0063 CampaignHypothesis;
// §0070 CoordinationRing). Wraps the internal/hypothesis package's
// FormCoordinationRingAll entry point with command-line
// configuration + structured output.
//
// Subtype distinction (per entity-model.md + §0070): CoordinationRing
// is "a set of actors whose patterns of interaction suggest
// coordinated action". The inference is INTERACTION-centric (an
// undirected edge list) — neither flat-set-of-actors (BC/AG) nor
// flat-set-of-events (CH). The inception pattern
// (co-occurrence-window-v1) builds edges from actor pairs whose
// declared sessions share a descriptor within a sliding window;
// connected components meeting a minimum edge support count form
// the ring.
//
// Output: structured JSON to stdout + brief human summary to stderr.
// Exit code: 0 success; 2 tool/configuration error.
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
		fmt.Fprintf(os.Stderr, "form-coordination-ring: %v\n", err)
		os.Exit(2)
	}
}

func run() error {
	dbPath := flag.String("db", "./ghost-trace.db", "SQLite primary-event-log path")
	blobDir := flag.String("blobs", "./blobs", "content-addressed blob-store directory")
	patternSignature := flag.String("pattern-signature", hypothesis.CoOccurrenceWindowV1Signature, "formation pattern signature identifier")
	minEdgeSupport := flag.Int64("min-edge-support", 3, "co-occurrence-window-v1: minimum co-occurrence observations per actor pair for the edge to qualify")
	maxWindowSeconds := flag.Int64("max-window-seconds", 600, "co-occurrence-window-v1: max elapsed seconds for two sessions sharing a descriptor to count as a co-occurrence")
	actor := flag.String("actor", "", "OPTIONAL per decision-log §0097 + §0111: when non-empty, pairs each formed event with an IngestionEvent for per-actor attribution. Empty preserves the single-Append path.")
	flag.Parse()

	pattern, err := resolvePattern(*patternSignature, *minEdgeSupport, *maxWindowSeconds)
	if err != nil {
		return err
	}

	ctx := context.Background()
	sub, err := substrate.Open(ctx, *dbPath, *blobDir)
	if err != nil {
		return fmt.Errorf("open substrate: %w", err)
	}
	defer func() { _ = sub.Close() }()

	report, err := hypothesis.FormCoordinationRingAllWithActor(ctx, sub, pattern, time.Now, *actor)
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
			"form-coordination-ring: pattern=%s params=%q examined=%d newly_formed=%d already_formed=%d actor=%q\n",
			pattern.Signature(), pattern.Parameters(), report.Examined, report.NewlyFormed, report.AlreadyFormed, *actor)
	} else {
		fmt.Fprintf(os.Stderr,
			"form-coordination-ring: pattern=%s params=%q examined=%d newly_formed=%d already_formed=%d\n",
			pattern.Signature(), pattern.Parameters(), report.Examined, report.NewlyFormed, report.AlreadyFormed)
	}
	return nil
}

func resolvePattern(signature string, minEdgeSupport, maxWindowSeconds int64) (hypothesis.CoordinationRingFormationPattern, error) {
	switch signature {
	case hypothesis.CoOccurrenceWindowV1Signature:
		return hypothesis.CoOccurrenceWindowV1{
			MinEdgeSupport:    minEdgeSupport,
			MaxWindowSeconds: maxWindowSeconds,
		}, nil
	default:
		return nil, fmt.Errorf("unknown pattern-signature %q; known signatures: %s",
			signature, hypothesis.CoOccurrenceWindowV1Signature)
	}
}

type payload struct {
	PatternSignature  string `json:"pattern_signature"`
	PatternParameters string `json:"pattern_parameters"`
	Examined          int64  `json:"examined"`
	NewlyFormed       int64  `json:"newly_formed"`
	AlreadyFormed     int64  `json:"already_formed"`
}
