// Command replay-derived-actor-attribution re-derives a Cat II
// DerivedActorAttribution from its declared Cat I source NetworkObservation
// under the same attribution definition the original record carries,
// per decision-log §0171 + docs/architecture/replay-model.md L17-19
// (Phase 1 replay — deterministic over observation alone).
//
// Mirrors replay-operational-session on the attribution Cat II side
// per §0172 — both Cat II types committed today (per §0043
// OperationalSession + §0168 DerivedActorAttribution) now have
// parallel operator-invocable replay CLIs.
//
// Output: structured JSON to stdout with target hash, recomputed
// hash, match boolean, and diagnostic fields. Exit codes (identical
// shape to replay-operational-session):
//
//	0 success AND match=true (Phase 1 contract holds)
//	1 success AND match=false (derivation drift detected)
//	2 tool/config error
//	3 target-not-found OR target-wrong-type OR definition-unknown
//	  OR definition-parameter-mismatch OR source-not-found OR
//	  source-wrong-type (substrate-integrity concerns)
//
// The two failure modes are distinct:
//   - exit 1 means replay completed but the recomputed hash differs
//     from the substrate's committed hash. Indicates derivation-
//     implementation drift since the original commit, OR a §2.1
//     immutability concern with the substrate record itself.
//   - exit 3 means replay could not complete due to a substrate-
//     integrity precondition failure (e.g. the referenced source is
//     missing).
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
		fmt.Fprintf(os.Stderr, "replay-derived-actor-attribution: %v\n", err)
		if errors.Is(err, replay.ErrTargetNotFound) ||
			errors.Is(err, replay.ErrTargetWrongType) ||
			errors.Is(err, replay.ErrDefinitionUnknown) ||
			errors.Is(err, replay.ErrDefinitionParameterMismatch) ||
			errors.Is(err, replay.ErrSourceNotFound) ||
			errors.Is(err, replay.ErrSourceWrongType) {
			os.Exit(exitTargetIntegrity)
		}
		os.Exit(exitToolError)
	}
}

func run() error {
	dbPath := flag.String("db", "./ghost-trace.db", "SQLite primary-event-log path")
	blobDir := flag.String("blobs", "./blobs", "content-addressed blob-store directory")
	targetHashHex := flag.String("target-event-hash", "", "REQUIRED: hex-encoded BLAKE3-256 of the DerivedActorAttribution to replay")
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

	report, err := replay.ReplayDerivedActorAttribution(ctx, sub, targetHash)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(payload{
		TargetEventHash:      report.TargetHashHex,
		RecomputedEventHash:  report.RecomputedHashHex,
		Match:                report.Match,
		DefinitionVersion:    report.DefinitionVersion,
		DefinitionParameters: report.DefinitionParameters,
		SourceEventHash:      report.SourceEventHashHex,
	}); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}

	fmt.Fprintf(os.Stderr,
		"replay-derived-actor-attribution: target=%s recomputed=%s match=%v definition=%s parameters=%q\n",
		report.TargetHashHex, report.RecomputedHashHex, report.Match,
		report.DefinitionVersion, report.DefinitionParameters)

	if !report.Match {
		os.Exit(exitDriftDetected)
	}
	return nil
}

// payload mirrors replay-operational-session's payload shape per
// §0172 stable-wire-contract discipline (§0163): both replay CLIs
// emit identical JSON envelopes; only the target type differs.
type payload struct {
	TargetEventHash      string `json:"target_event_hash"`
	RecomputedEventHash  string `json:"recomputed_event_hash"`
	Match                bool   `json:"match"`
	DefinitionVersion    string `json:"definition_version"`
	DefinitionParameters string `json:"definition_parameters"`
	SourceEventHash      string `json:"source_event_hash"`
}
