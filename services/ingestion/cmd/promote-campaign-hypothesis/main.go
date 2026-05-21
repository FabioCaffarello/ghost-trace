// Command promote-campaign-hypothesis records a CampaignHypothesis
// promotion lifecycle event per Charter §2.5 + decision-log §0064 —
// second lifecycle operation of the third Cat III subtype arc
// (mirrors §0046 + §0057).
//
// Output: structured JSON to stdout. Exit codes: 0 success, 2
// tool/config error, 3 target-not-found OR target-wrong-type.
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

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/hypothesis"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

const (
	exitToolError       = 2
	exitTargetIntegrity = 3
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "promote-campaign-hypothesis: %v\n", err)
		if errors.Is(err, hypothesis.ErrTargetNotFound) || errors.Is(err, hypothesis.ErrTargetWrongType) {
			os.Exit(exitTargetIntegrity)
		}
		os.Exit(exitToolError)
	}
}

func run() error {
	dbPath := flag.String("db", "./ghost-trace.db", "SQLite primary-event-log path")
	blobDir := flag.String("blobs", "./blobs", "content-addressed blob-store directory")
	formationHashHex := flag.String("formation-event-hash", "", "REQUIRED: hex-encoded BLAKE3-256 of the target CampaignHypothesisFormation")
	cadenceSeconds := flag.Int64("cadence-seconds", 86400, "Layer A cadence parameter per decision-log §0011")
	promotedAtNs := flag.Int64("promoted-at-ns", 0, "explicit promoted_at as Unix nanoseconds; 0 = wall-clock now()")
	reason := flag.String("reason", "", "operator-supplied forensic note; optional")
	flag.Parse()

	if *formationHashHex == "" {
		return fmt.Errorf("--formation-event-hash is required")
	}
	raw, err := hex.DecodeString(*formationHashHex)
	if err != nil {
		return fmt.Errorf("decode --formation-event-hash: %w", err)
	}
	if len(raw) != 32 {
		return fmt.Errorf("--formation-event-hash must be 32 bytes (64 hex chars); got %d bytes", len(raw))
	}
	var hash [32]byte
	copy(hash[:], raw)

	ctx := context.Background()
	sub, err := substrate.Open(ctx, *dbPath, *blobDir)
	if err != nil {
		return fmt.Errorf("open substrate: %w", err)
	}
	defer func() { _ = sub.Close() }()

	report, err := hypothesis.PromoteCampaignHypothesis(ctx, sub, hypothesis.CampaignHypothesisPromoteOptions{
		FormationEventHash: hash,
		PromotedAt:         *promotedAtNs,
		CadenceSeconds:     *cadenceSeconds,
		Reason:             *reason,
	}, time.Now)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(payload{
		FormationEventHash:    *formationHashHex,
		PromotionEventHashHex: report.PromotionEventHashHex,
		CadenceSeconds:        *cadenceSeconds,
		AlreadyPromoted:       report.AlreadyPromoted,
	}); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}

	fmt.Fprintf(os.Stderr,
		"promote-campaign-hypothesis: formation=%s promotion=%s cadence_seconds=%d already_promoted=%v\n",
		*formationHashHex, report.PromotionEventHashHex, *cadenceSeconds, report.AlreadyPromoted)
	return nil
}

type payload struct {
	FormationEventHash    string `json:"formation_event_hash"`
	PromotionEventHashHex string `json:"promotion_event_hash"`
	CadenceSeconds        int64  `json:"cadence_seconds"`
	AlreadyPromoted       bool   `json:"already_promoted"`
}
