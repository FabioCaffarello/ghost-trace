// Command demote-campaign-hypothesis records a CampaignHypothesis
// demotion lifecycle event per Charter §2.5 + decision-log §0065 —
// third lifecycle operation of the third Cat III subtype arc
// (mirrors §0047 and §0058).
//
// Output: structured JSON. Exit codes: 0 success, 2 tool/config
// error, 3 target-not-found or target-wrong-type.
package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/cliutil"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/hypothesis"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

const (
	exitToolError       = 2
	exitTargetIntegrity = 3
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "demote-campaign-hypothesis: %v\n", err)
		if errors.Is(err, hypothesis.ErrTargetNotFound) || errors.Is(err, hypothesis.ErrTargetWrongType) {
			os.Exit(exitTargetIntegrity)
		}
		os.Exit(exitToolError)
	}
}

func run() error {
	dbPath := flag.String("db", "./ghost-trace.db", "SQLite primary-event-log path")
	blobDir := flag.String("blobs", "./blobs", "content-addressed blob-store directory")
	promotionHashHex := flag.String("promotion-event-hash", "", "REQUIRED: hex-encoded BLAKE3-256 of the target CampaignHypothesisPromotion")
	demotedAtNs := flag.Int64("demoted-at-ns", 0, "explicit demoted_at as Unix nanoseconds; 0 = wall-clock now()")
	reason := flag.String("reason", "", "operator-supplied forensic note")
	actor := flag.String("actor", "", "OPTIONAL per decision-log §0097 + §0107: when non-empty, pairs the demotion with an IngestionEvent for per-actor attribution.")
	flag.Parse()

	if *promotionHashHex == "" {
		return fmt.Errorf("--promotion-event-hash is required")
	}
	raw, err := hex.DecodeString(*promotionHashHex)
	if err != nil {
		return fmt.Errorf("decode --promotion-event-hash: %w", err)
	}
	if len(raw) != 32 {
		return fmt.Errorf("--promotion-event-hash must be 32 bytes (64 hex chars); got %d bytes", len(raw))
	}
	var hash [32]byte
	copy(hash[:], raw)

	ctx := context.Background()
	sub, err := substrate.Open(ctx, *dbPath, *blobDir)
	if err != nil {
		return fmt.Errorf("open substrate: %w", err)
	}
	defer func() { _ = sub.Close() }()

	report, err := hypothesis.DemoteCampaignHypothesis(ctx, sub, hypothesis.CampaignHypothesisDemoteOptions{
		PromotionEventHash: hash,
		DemotedAt:          *demotedAtNs,
		Reason:             *reason,
		Actor:              *actor,
	}, time.Now)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(payload{
		PromotionEventHash:    *promotionHashHex,
		DemotionEventHashHex:  report.DemotionEventHashHex,
		AlreadyDemoted:        report.AlreadyDemoted,
		CadenceSatisfied:      report.CadenceSatisfied,
		CadenceElapsedSeconds: report.CadenceElapsedSeconds,
		IngestionEventHashHex: report.IngestionEventHashHex,
		LayerB:                cliutil.LayerBPayloadFromReport(report.LayerB),
	}); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}

	layerBSummary := cliutil.LayerBSummary(report.LayerB)
	if *actor != "" {
		fmt.Fprintf(os.Stderr,
			"demote-campaign-hypothesis: promotion=%s demotion=%s ingestion=%s cadence_satisfied=%v elapsed=%ds %s actor=%q\n",
			*promotionHashHex, report.DemotionEventHashHex, report.IngestionEventHashHex, report.CadenceSatisfied, report.CadenceElapsedSeconds, layerBSummary, *actor)
	} else {
		fmt.Fprintf(os.Stderr,
			"demote-campaign-hypothesis: promotion=%s demotion=%s cadence_satisfied=%v elapsed=%ds %s\n",
			*promotionHashHex, report.DemotionEventHashHex, report.CadenceSatisfied, report.CadenceElapsedSeconds, layerBSummary)
	}
	return nil
}

type payload struct {
	PromotionEventHash    string                `json:"promotion_event_hash"`
	DemotionEventHashHex  string                `json:"demotion_event_hash"`
	AlreadyDemoted        bool                  `json:"already_demoted"`
	CadenceSatisfied      bool                  `json:"cadence_satisfied"`
	CadenceElapsedSeconds int64                 `json:"cadence_elapsed_seconds"`
	IngestionEventHashHex string                `json:"ingestion_event_hash,omitempty"`
	LayerB                cliutil.LayerBPayload `json:"layer_b"`
}
