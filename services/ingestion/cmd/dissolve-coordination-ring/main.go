// Command dissolve-coordination-ring records a CoordinationRing
// dissolution lifecycle event per Charter §2.5 + decision-log §0073 —
// fourth lifecycle operation of the fourth Cat III subtype arc
// (mirrors §0048 BC, §0059 AG, §0066 CH).
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

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/hypothesis"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

const (
	exitToolError       = 2
	exitTargetIntegrity = 3
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "dissolve-coordination-ring: %v\n", err)
		if errors.Is(err, hypothesis.ErrTargetNotFound) || errors.Is(err, hypothesis.ErrTargetWrongType) {
			os.Exit(exitTargetIntegrity)
		}
		os.Exit(exitToolError)
	}
}

func run() error {
	dbPath := flag.String("db", "./ghost-trace.db", "SQLite primary-event-log path")
	blobDir := flag.String("blobs", "./blobs", "content-addressed blob-store directory")
	formationHashHex := flag.String("formation-event-hash", "", "REQUIRED: hex-encoded BLAKE3-256 of the target CoordinationRingFormation")
	dissolvedAtNs := flag.Int64("dissolved-at-ns", 0, "explicit dissolved_at as Unix nanoseconds; 0 = wall-clock now()")
	reason := flag.String("reason", "", "operator-supplied forensic note; strongly recommended")
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

	report, err := hypothesis.DissolveCoordinationRing(ctx, sub, hypothesis.CoordinationRingDissolveOptions{
		FormationEventHash: hash,
		DissolvedAt:        *dissolvedAtNs,
		Reason:             *reason,
	}, time.Now)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(payload{
		FormationEventHash:   *formationHashHex,
		DissolutionEventHash: report.DissolutionEventHashHex,
		AlreadyDissolved:     report.AlreadyDissolved,
	}); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}

	fmt.Fprintf(os.Stderr,
		"dissolve-coordination-ring: formation=%s dissolution=%s already_dissolved=%v\n",
		*formationHashHex, report.DissolutionEventHashHex, report.AlreadyDissolved)
	return nil
}

type payload struct {
	FormationEventHash   string `json:"formation_event_hash"`
	DissolutionEventHash string `json:"dissolution_event_hash"`
	AlreadyDissolved     bool   `json:"already_dissolved"`
}
