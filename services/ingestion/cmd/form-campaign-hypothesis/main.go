// Command form-campaign-hypothesis runs the CampaignHypothesis
// formation pathway per decision-log §0063 — opens the THIRD Cat
// III concrete subtype lifecycle arc (§0045 opened BehavioralCluster;
// §0056 opened AutomationGroup). Wraps the internal/hypothesis
// package's FormCampaignHypothesisAll entry point with command-line
// configuration + structured output.
//
// Subtype distinction (per entity-model.md): CampaignHypothesis is
// "a set of events whose patterns suggest membership in a unified
// operation". The inference is event-centric (not actor-centric);
// the inception pattern (temporal-descriptor-cohort-v1) groups
// events by shared session_descriptor + temporal proximity.
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
		fmt.Fprintf(os.Stderr, "form-campaign-hypothesis: %v\n", err)
		os.Exit(2)
	}
}

func run() error {
	dbPath := flag.String("db", "./ghost-trace.db", "SQLite primary-event-log path")
	blobDir := flag.String("blobs", "./blobs", "content-addressed blob-store directory")
	patternSignature := flag.String("pattern-signature", hypothesis.TemporalDescriptorCohortV1Signature, "formation pattern signature identifier")
	minCampaignSize := flag.Int64("min-campaign-size", 3, "temporal-descriptor-cohort-v1: minimum events in a cohort")
	maxIntraEventGapSeconds := flag.Int64("max-intra-event-gap-seconds", 300, "temporal-descriptor-cohort-v1: max elapsed seconds between consecutive events to keep them in the same cohort")
	flag.Parse()

	pattern, err := resolvePattern(*patternSignature, *minCampaignSize, *maxIntraEventGapSeconds)
	if err != nil {
		return err
	}

	ctx := context.Background()
	sub, err := substrate.Open(ctx, *dbPath, *blobDir)
	if err != nil {
		return fmt.Errorf("open substrate: %w", err)
	}
	defer func() { _ = sub.Close() }()

	report, err := hypothesis.FormCampaignHypothesisAll(ctx, sub, pattern, time.Now)
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
		"form-campaign-hypothesis: pattern=%s params=%q examined=%d newly_formed=%d already_formed=%d\n",
		pattern.Signature(), pattern.Parameters(), report.Examined, report.NewlyFormed, report.AlreadyFormed)
	return nil
}

func resolvePattern(signature string, minCampaignSize, maxIntraEventGapSeconds int64) (hypothesis.CampaignHypothesisFormationPattern, error) {
	switch signature {
	case hypothesis.TemporalDescriptorCohortV1Signature:
		return hypothesis.TemporalDescriptorCohortV1{
			MinCampaignSize:         minCampaignSize,
			MaxIntraEventGapSeconds: maxIntraEventGapSeconds,
		}, nil
	default:
		return nil, fmt.Errorf("unknown pattern-signature %q; known signatures: %s",
			signature, hypothesis.TemporalDescriptorCohortV1Signature)
	}
}

type payload struct {
	PatternSignature  string `json:"pattern_signature"`
	PatternParameters string `json:"pattern_parameters"`
	Examined          int64  `json:"examined"`
	NewlyFormed       int64  `json:"newly_formed"`
	AlreadyFormed     int64  `json:"already_formed"`
}
