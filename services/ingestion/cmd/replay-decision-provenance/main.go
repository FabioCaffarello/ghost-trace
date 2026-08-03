// Command replay-decision-provenance reconstructs the full
// observation→inference→decision chain for an OperationalDecisionAudit
// per decision-log §0222. Given a decision audit hash, it resolves the
// influencing Cat III hypotheses and, through each, the Cat I
// observations that grounded the inference — answering "why was this
// verdict reached, on what evidence?" entirely from the substrate. This
// completes the §0221 replay capability across the fourth (decision) hop.
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
		fmt.Fprintf(os.Stderr, "replay-decision-provenance: %v\n", err)
		if errors.Is(err, replay.ErrTargetNotFound) || errors.Is(err, replay.ErrTargetWrongType) {
			os.Exit(exitTargetIntegrity)
		}
		os.Exit(exitToolError)
	}
}

func run() error {
	dbPath := flag.String("db", "./ghost-trace.db", "SQLite primary-event-log path")
	blobDir := flag.String("blobs", "./blobs", "content-addressed blob-store directory")
	targetHashHex := flag.String("target-event-hash", "", "REQUIRED: hex BLAKE3-256 of the OperationalDecisionAudit to reconstruct")
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

	report, err := replay.ReconstructDecisionProvenance(ctx, sub, targetHash)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	fmt.Fprintf(os.Stderr, "replay-decision-provenance: decision=%s verdict=%s subject=%s policy=%s operator=%s hypotheses=%d\n",
		report.DecisionHashHex, report.Verdict, report.SubjectActorRef, report.PolicyRef, report.OperatorRef, len(report.InfluencingHypotheses))
	return nil
}
