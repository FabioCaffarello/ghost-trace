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

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/cliutil"
	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
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
	actor := flag.String("actor", "", "OPTIONAL per decision-log §0097 + §0106: when non-empty, pairs the promotion with an IngestionEvent for per-actor attribution.")
	layerB := flag.Bool("layer-b", false, "OPTIONAL per decision-log §0141 F3: when true, populate layer_b_parameters with §0138 inception-phase resolved values. Default false preserves the legacy path.")
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

	var layerBParams *commonv1.LayerBParameters
	if *layerB {
		layerBParams = cliutil.InceptionPhaseLayerBParameters(*cadenceSeconds)
	}
	report, err := hypothesis.PromoteCampaignHypothesis(ctx, sub, hypothesis.CampaignHypothesisPromoteOptions{
		FormationEventHash: hash,
		PromotedAt:         *promotedAtNs,
		CadenceSeconds:     *cadenceSeconds,
		Reason:             *reason,
		Actor:              *actor,
		LayerBParameters:   layerBParams,
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
		IngestionEventHashHex: report.IngestionEventHashHex,
		LayerBEnabled:         *layerB,
	}); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}

	layerBState := "layer_b=legacy"
	if *layerB {
		layerBState = "layer_b=inception_defaults"
	}
	if *actor != "" {
		fmt.Fprintf(os.Stderr,
			"promote-campaign-hypothesis: formation=%s promotion=%s ingestion=%s cadence_seconds=%d %s actor=%q already_promoted=%v\n",
			*formationHashHex, report.PromotionEventHashHex, report.IngestionEventHashHex, *cadenceSeconds, layerBState, *actor, report.AlreadyPromoted)
	} else {
		fmt.Fprintf(os.Stderr,
			"promote-campaign-hypothesis: formation=%s promotion=%s cadence_seconds=%d %s already_promoted=%v\n",
			*formationHashHex, report.PromotionEventHashHex, *cadenceSeconds, layerBState, report.AlreadyPromoted)
	}
	return nil
}

type payload struct {
	FormationEventHash    string `json:"formation_event_hash"`
	PromotionEventHashHex string `json:"promotion_event_hash"`
	CadenceSeconds        int64  `json:"cadence_seconds"`
	AlreadyPromoted       bool   `json:"already_promoted"`
	IngestionEventHashHex string `json:"ingestion_event_hash,omitempty"`
	LayerBEnabled         bool   `json:"layer_b_enabled"`
}
