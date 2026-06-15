// Command replay-formation-provenance reconstructs the observational-
// provenance chain of an AutomationGroupFormation per decision-log
// §0221: given a formation event hash, it resolves the committed
// source_event_hashes (§2.3) back to the Cat I observations that
// grounded the hypothesis, surfacing each observation's observed TLS
// fingerprint (JA3/JA4). This is the REPLAY / audit layer of the §0221
// vertical slice — it reconstructs the observation→inference chain for
// a hypothesis so an auditor can confirm what was observed vs inferred.
//
// Distinct from replay-automation-group-formation (Phase-3 deterministic
// replay against a registered formation pattern): F3-candidate-derived
// formations carry a signature name (e.g. tls_ja4_automation_v1) as
// their pattern_signature, which is not a registered substrate-walk
// pattern, so deterministic replay reports pattern-unknown for them.
// This command covers the complementary reconstructive guarantee by
// resolving the provenance chain directly.
//
// Read-only over the substrate; no AppendPair.
//
// Exit codes: 0 success; 2 tool/config error; 3 substrate-precondition
// failure (target not found / wrong type).
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
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "replay-formation-provenance: %v\n", err)
		if errors.Is(err, replay.ErrTargetNotFound) || errors.Is(err, replay.ErrTargetWrongType) {
			os.Exit(exitTargetIntegrity)
		}
		os.Exit(exitToolError)
	}
}

func run() error {
	dbPath := flag.String("db", "./ghost-trace.db", "SQLite primary-event-log path")
	blobDir := flag.String("blobs", "./blobs", "content-addressed blob-store directory")
	targetHashHex := flag.String("target-event-hash", "", "REQUIRED: hex-encoded BLAKE3-256 of the AutomationGroupFormation to reconstruct")
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
		return fmt.Errorf("substrate.Open: %w", err)
	}
	defer sub.Close()

	report, err := replay.ReconstructAutomationGroupProvenance(ctx, sub, targetHash)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}

	resolved := 0
	for _, rs := range report.ResolvedSources {
		if rs.Found {
			resolved++
		}
	}
	fmt.Fprintf(os.Stderr, "replay-formation-provenance: formation=%s pattern=%s confidence=%.3f ei=%d/%d sources=%d resolved=%d\n",
		report.FormationHashHex, report.PatternSignature, report.Confidence,
		report.EvidentialNum, report.EvidentialDen, report.SourceCount, resolved)
	return nil
}
