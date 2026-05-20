// Command promote-hypothesis records a Category III hypothesis
// promotion lifecycle event per Charter §2.5 + decision-log §0046 —
// second Cat III lifecycle operation landing (formation was §0045).
// Wraps internal/hypothesis.Promote with command-line configuration
// + structured output.
//
// Operationally:
//
//   - Operator identifies a BehavioralClusterFormation by its
//     content-hash (e.g. from the form-hypothesis CLI output or via
//     substrate inspection) and promotes it under a specified Layer A
//     cadence parameter per decision-log §0011.
//   - Re-running with the same (formation hash, cadence, reason,
//     explicit promoted_at) is idempotent (content-hash collision →
//     INSERT OR IGNORE per §0027). With default promoted_at=now(),
//     each invocation records a fresh promotion (intentional
//     non-idempotency at wall-clock granularity).
//   - Changing cadence_seconds re-records the cadence gate without
//     mutating the prior promotion event (§2.5 immutability).
//
// Output: structured JSON to stdout + brief human summary to stderr.
// Exit codes: 0 success, 2 tool/config error, 3 target-not-found or
// target-wrong-type (the two §2.5-integrity errors).
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
		fmt.Fprintf(os.Stderr, "promote-hypothesis: %v\n", err)
		if errors.Is(err, hypothesis.ErrTargetNotFound) || errors.Is(err, hypothesis.ErrTargetWrongType) {
			os.Exit(exitTargetIntegrity)
		}
		os.Exit(exitToolError)
	}
}

func run() error {
	dbPath := flag.String("db", "./ghost-trace.db", "SQLite primary-event-log path")
	blobDir := flag.String("blobs", "./blobs", "content-addressed blob-store directory")
	formationHashHex := flag.String("formation-event-hash", "", "REQUIRED: hex-encoded BLAKE3-256 content-hash of the target BehavioralClusterFormation")
	cadenceSeconds := flag.Int64("cadence-seconds", 86400, "Layer A cadence parameter per decision-log §0011: elapsed seconds since promoted_at that opens demotion-candidacy")
	promotedAtNs := flag.Int64("promoted-at-ns", 0, "explicit promoted_at as Unix nanoseconds; 0 = use wall-clock now()")
	reason := flag.String("reason", "", "operator-supplied forensic note; optional but recommended at audit time")
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

	report, err := hypothesis.Promote(ctx, sub, hypothesis.PromoteOptions{
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
		"promote-hypothesis: formation=%s promotion=%s cadence_seconds=%d already_promoted=%v\n",
		*formationHashHex, report.PromotionEventHashHex, *cadenceSeconds, report.AlreadyPromoted)
	return nil
}

type payload struct {
	FormationEventHash    string `json:"formation_event_hash"`
	PromotionEventHashHex string `json:"promotion_event_hash"`
	CadenceSeconds        int64  `json:"cadence_seconds"`
	AlreadyPromoted       bool   `json:"already_promoted"`
}
