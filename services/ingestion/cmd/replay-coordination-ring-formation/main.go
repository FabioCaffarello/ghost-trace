// Command replay-coordination-ring-formation performs Phase 3
// reconstructive replay of a CoordinationRingFormation per
// decision-log §0089. Closes the four-subtype Phase 3 arc opened at
// §0086. Mirrors §0086 (BC), §0087 (AG), §0088 (CH).
//
// Exit codes: 0 success AND match=true; 1 drift; 2 tool error;
// 3 substrate-precondition failure.
package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
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
		fmt.Fprintf(os.Stderr, "replay-coordination-ring-formation: %v\n", err)
		if errors.Is(err, replay.ErrTargetNotFound) ||
			errors.Is(err, replay.ErrTargetWrongType) ||
			errors.Is(err, replay.ErrPatternUnknown) ||
			errors.Is(err, replay.ErrPatternParameterMismatch) {
			os.Exit(exitTargetIntegrity)
		}
		os.Exit(exitToolError)
	}
}

func run() error {
	dbPath := flag.String("db", "./ghost-trace.db", "SQLite primary-event-log path")
	blobDir := flag.String("blobs", "./blobs", "content-addressed blob-store directory")
	targetHashHex := flag.String("target-event-hash", "", "REQUIRED: hex-encoded BLAKE3-256 of the CoordinationRingFormation to replay")
	flag.Parse()

	if *targetHashHex == "" {
		return fmt.Errorf("--target-event-hash is required")
	}
	raw, err := hex.DecodeString(*targetHashHex)
	if err != nil {
		return fmt.Errorf("decode --target-event-hash: %w", err)
	}
	if len(raw) != 32 {
		return fmt.Errorf("--target-event-hash must be 32 bytes (64 hex chars); got %d bytes", len(raw))
	}
	var targetHash [32]byte
	copy(targetHash[:], raw)

	ctx := context.Background()
	sub, err := substrate.Open(ctx, *dbPath, *blobDir)
	if err != nil {
		return fmt.Errorf("open substrate: %w", err)
	}
	defer func() { _ = sub.Close() }()

	report, err := replay.ReplayCoordinationRingFormation(ctx, sub, targetHash)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(payload{
		TargetEventHash:              report.TargetHashHex,
		Match:                        report.Match,
		RecomputedEventHash:          report.RecomputedHashHex,
		PatternSignature:             report.PatternSignature,
		PatternParameters:            report.PatternParameters,
		ReconstructedFormationCount:  report.ReconstructedFormationCount,
		ContributingObservationCount: report.ContributingObservationCount,
		MaxCommittedAtNs:             report.MaxCommittedAtNs,
	}); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}

	fmt.Fprintf(os.Stderr,
		"replay-coordination-ring-formation: target=%s match=%v reconstructed=%d contributors=%d max_committed_at=%d\n",
		report.TargetHashHex, report.Match, report.ReconstructedFormationCount,
		report.ContributingObservationCount, report.MaxCommittedAtNs)

	if !report.Match {
		os.Exit(exitDriftDetected)
	}
	return nil
}

type payload struct {
	TargetEventHash              string `json:"target_event_hash"`
	Match                        bool   `json:"match"`
	RecomputedEventHash          string `json:"recomputed_event_hash,omitempty"`
	PatternSignature             string `json:"pattern_signature"`
	PatternParameters            string `json:"pattern_parameters"`
	ReconstructedFormationCount  int    `json:"reconstructed_formation_count"`
	ContributingObservationCount int    `json:"contributing_observation_count"`
	MaxCommittedAtNs             int64  `json:"max_committed_at_ns"`
}
