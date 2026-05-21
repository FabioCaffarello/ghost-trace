// Command merge-coordination-rings records a CoordinationRing
// merge lifecycle event per Charter §2.5 + lifecycle-semantics.md
// line 28 + decision-log §0074 — fifth lifecycle operation of the
// fourth Cat III subtype arc (mirrors §0049 BC, §0060 AG, §0067 CH).
//
// SCOPE — within-subtype only. Symmetric: ascending-sort of
// antecedent hashes before recording.
//
// Output: structured JSON. Exit codes: 0 success, 2 tool/config
// error, 3 target-not-found OR target-wrong-type OR
// identical-antecedents.
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
		fmt.Fprintf(os.Stderr, "merge-coordination-rings: %v\n", err)
		if errors.Is(err, hypothesis.ErrTargetNotFound) ||
			errors.Is(err, hypothesis.ErrTargetWrongType) ||
			errors.Is(err, hypothesis.ErrMergeAntecedentsIdentical) {
			os.Exit(exitTargetIntegrity)
		}
		os.Exit(exitToolError)
	}
}

func run() error {
	dbPath := flag.String("db", "./ghost-trace.db", "SQLite primary-event-log path")
	blobDir := flag.String("blobs", "./blobs", "content-addressed blob-store directory")
	antecedentAHex := flag.String("antecedent-a-hash", "", "REQUIRED: hex-encoded BLAKE3-256 of the FIRST CoordinationRingFormation")
	antecedentBHex := flag.String("antecedent-b-hash", "", "REQUIRED: hex-encoded BLAKE3-256 of the SECOND CoordinationRingFormation")
	producedHex := flag.String("produced-formation-hash", "", "REQUIRED: hex-encoded BLAKE3-256 of the produced CoordinationRingFormation")
	mergedAtNs := flag.Int64("merged-at-ns", 0, "explicit merged_at as Unix nanoseconds; 0 = wall-clock now()")
	reason := flag.String("reason", "", "operator-supplied forensic note")
	actor := flag.String("actor", "", "OPTIONAL per decision-log §0097 + §0109: when non-empty, pairs the merge with an IngestionEvent for per-actor attribution.")
	flag.Parse()

	if *antecedentAHex == "" {
		return fmt.Errorf("--antecedent-a-hash is required")
	}
	if *antecedentBHex == "" {
		return fmt.Errorf("--antecedent-b-hash is required")
	}
	if *producedHex == "" {
		return fmt.Errorf("--produced-formation-hash is required")
	}

	antA, err := decodeHash(*antecedentAHex, "antecedent-a-hash")
	if err != nil {
		return err
	}
	antB, err := decodeHash(*antecedentBHex, "antecedent-b-hash")
	if err != nil {
		return err
	}
	produced, err := decodeHash(*producedHex, "produced-formation-hash")
	if err != nil {
		return err
	}

	ctx := context.Background()
	sub, err := substrate.Open(ctx, *dbPath, *blobDir)
	if err != nil {
		return fmt.Errorf("open substrate: %w", err)
	}
	defer func() { _ = sub.Close() }()

	report, err := hypothesis.MergeCoordinationRing(ctx, sub, hypothesis.CoordinationRingMergeOptions{
		AntecedentAFormationHash: antA,
		AntecedentBFormationHash: antB,
		ProducedFormationHash:    produced,
		MergedAt:                 *mergedAtNs,
		Reason:                   *reason,
		Actor:                    *actor,
	}, time.Now)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(payload{
		AntecedentAHash:       *antecedentAHex,
		AntecedentBHash:       *antecedentBHex,
		ProducedFormationHash: *producedHex,
		MergeEventHash:        report.MergeEventHashHex,
		AlreadyMerged:         report.AlreadyMerged,
		IngestionEventHash:    report.IngestionEventHashHex,
	}); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}

	if *actor != "" {
		fmt.Fprintf(os.Stderr,
			"merge-coordination-rings: produced=%s merge=%s ingestion=%s actor=%q already_merged=%v\n",
			*producedHex, report.MergeEventHashHex, report.IngestionEventHashHex, *actor, report.AlreadyMerged)
	} else {
		fmt.Fprintf(os.Stderr,
			"merge-coordination-rings: produced=%s merge=%s already_merged=%v\n",
			*producedHex, report.MergeEventHashHex, report.AlreadyMerged)
	}
	return nil
}

func decodeHash(s, flagName string) ([32]byte, error) {
	raw, err := hex.DecodeString(s)
	if err != nil {
		return [32]byte{}, fmt.Errorf("decode --%s: %w", flagName, err)
	}
	if len(raw) != 32 {
		return [32]byte{}, fmt.Errorf("--%s must be 32 bytes (64 hex chars); got %d bytes", flagName, len(raw))
	}
	var h [32]byte
	copy(h[:], raw)
	return h, nil
}

type payload struct {
	AntecedentAHash       string `json:"antecedent_a_hash"`
	AntecedentBHash       string `json:"antecedent_b_hash"`
	ProducedFormationHash string `json:"produced_formation_hash"`
	MergeEventHash        string `json:"merge_event_hash"`
	AlreadyMerged         bool   `json:"already_merged"`
	IngestionEventHash    string `json:"ingestion_event_hash,omitempty"`
}
