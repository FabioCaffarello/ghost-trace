// Command replay-behavioral-cluster-formation performs Phase 3
// reconstructive replay of a BehavioralCluster formation per
// docs/architecture/replay-model.md L25-28 + decision-log §0086.
//
// Phase 3 vs Phase 1: Phase 1 (replay-operational-session §0084)
// verifies deterministic re-derivation of a single Cat II record
// from a single Cat I source. Phase 3 reconstructs a Cat III
// hypothesis from the substrate-at-commit-time view of all Cat I
// observations. The BC formation pattern IS deterministic given its
// FormationContext, so for our concrete patterns Match=true is the
// expected outcome; a divergence indicates pattern-implementation
// drift or substrate-time vs event-time inconsistency.
//
// Output: structured JSON. Exit codes:
//   0 success AND match=true (Phase 3 contract holds)
//   1 success AND match=false (drift detected)
//   2 tool/config error
//   3 target-not-found / target-wrong-type / pattern-unknown /
//     pattern-parameter-mismatch (substrate-precondition errors)
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
		fmt.Fprintf(os.Stderr, "replay-behavioral-cluster-formation: %v\n", err)
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
	targetHashHex := flag.String("target-event-hash", "", "REQUIRED: hex-encoded BLAKE3-256 of the BehavioralClusterFormation to replay")
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

	report, err := replay.ReplayBehavioralClusterFormation(ctx, sub, targetHash)
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
		"replay-behavioral-cluster-formation: target=%s match=%v reconstructed=%d contributors=%d max_committed_at=%d\n",
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
