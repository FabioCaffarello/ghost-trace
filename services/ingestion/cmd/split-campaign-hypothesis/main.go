// Command split-campaign-hypothesis records a CampaignHypothesis
// split lifecycle event per Charter §2.5 + lifecycle-semantics.md
// line 29 + decision-log §0068 — SIXTH (final) lifecycle operation
// of the third Cat III subtype arc (mirrors §0050 BC + §0061 AG).
//
// SCOPE — within-subtype only. Successors form a SET (ascending-
// sort idempotency per §0050).
//
// Output: structured JSON to stdout. Exit codes: 0 success, 2
// tool/config error, 3 target-not-found OR target-wrong-type OR
// insufficient-successors OR successors-not-distinct.
package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/hypothesis"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

const (
	exitToolError       = 2
	exitTargetIntegrity = 3
)

type successorList struct {
	hexes []string
}

func (s *successorList) String() string {
	return strings.Join(s.hexes, ",")
}

func (s *successorList) Set(v string) error {
	if v == "" {
		return fmt.Errorf("-successor-formation-hash cannot be empty")
	}
	s.hexes = append(s.hexes, v)
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "split-campaign-hypothesis: %v\n", err)
		if errors.Is(err, hypothesis.ErrTargetNotFound) ||
			errors.Is(err, hypothesis.ErrTargetWrongType) ||
			errors.Is(err, hypothesis.ErrSplitInsufficientSuccessors) ||
			errors.Is(err, hypothesis.ErrSplitSuccessorsNotDistinct) {
			os.Exit(exitTargetIntegrity)
		}
		os.Exit(exitToolError)
	}
}

func run() error {
	dbPath := flag.String("db", "./ghost-trace.db", "SQLite primary-event-log path")
	blobDir := flag.String("blobs", "./blobs", "content-addressed blob-store directory")
	antecedentHex := flag.String("antecedent-formation-hash", "", "REQUIRED: hex-encoded BLAKE3-256 of the CampaignHypothesisFormation being split")
	var successors successorList
	flag.Var(&successors, "successor-formation-hash", "REQUIRED (≥ 2 invocations): hex-encoded BLAKE3-256 of a successor CampaignHypothesisFormation; repeat the option for each successor")
	splitAtNs := flag.Int64("split-at-ns", 0, "explicit split_at as Unix nanoseconds; 0 = wall-clock now()")
	reason := flag.String("reason", "", "operator-supplied forensic note; strongly recommended")
	flag.Parse()

	if *antecedentHex == "" {
		return fmt.Errorf("--antecedent-formation-hash is required")
	}
	if len(successors.hexes) < 2 {
		return fmt.Errorf("--successor-formation-hash must be supplied at least twice (got %d)", len(successors.hexes))
	}

	ant, err := decodeHash(*antecedentHex, "antecedent-formation-hash")
	if err != nil {
		return err
	}
	succs := make([][32]byte, 0, len(successors.hexes))
	for i, h := range successors.hexes {
		dec, err := decodeHash(h, fmt.Sprintf("successor-formation-hash[%d]", i))
		if err != nil {
			return err
		}
		succs = append(succs, dec)
	}

	ctx := context.Background()
	sub, err := substrate.Open(ctx, *dbPath, *blobDir)
	if err != nil {
		return fmt.Errorf("open substrate: %w", err)
	}
	defer func() { _ = sub.Close() }()

	report, err := hypothesis.SplitCampaignHypothesis(ctx, sub, hypothesis.CampaignHypothesisSplitOptions{
		AntecedentFormationHash:  ant,
		SuccessorFormationHashes: succs,
		SplitAt:                  *splitAtNs,
		Reason:                   *reason,
	}, time.Now)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(payload{
		AntecedentFormationHash:  *antecedentHex,
		SuccessorFormationHashes: successors.hexes,
		SplitEventHash:           report.SplitEventHashHex,
		AlreadySplit:             report.AlreadySplit,
	}); err != nil {
		return fmt.Errorf("encode json: %w", err)
	}

	fmt.Fprintf(os.Stderr,
		"split-campaign-hypothesis: antecedent=%s successors=%d split=%s already_split=%v\n",
		*antecedentHex, len(successors.hexes), report.SplitEventHashHex, report.AlreadySplit)
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
	AntecedentFormationHash  string   `json:"antecedent_formation_hash"`
	SuccessorFormationHashes []string `json:"successor_formation_hashes"`
	SplitEventHash           string   `json:"split_event_hash"`
	AlreadySplit             bool     `json:"already_split"`
}
