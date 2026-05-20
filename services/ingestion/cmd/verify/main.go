// Command verify performs an up-front substrate-integrity check
// against the ingestion service's SQLite + content-addressed blob-
// store substrate per decision-log §0039 + §0033 §Restoration
// Procedure step 3.
//
// Operationally:
//
//   - Run after restoring a backup pair (per §0033) to surface any
//     §2.1 violations before the ingestion service starts. Hash-
//     mismatch or missing-blob errors abort recovery before live
//     traffic resumes.
//
//   - Run periodically as a substrate-integrity audit on a running
//     substrate (the read path is concurrent-safe under SQLite WAL).
//
// Output is structured JSON to stdout (machine-readable) + a brief
// human summary to stderr. Exit code: 0 on pass, 1 on any failure
// (hash-mismatch or missing-blob).
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/verify"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "verify: %v\n", err)
		os.Exit(2) // 2 = tool/configuration error (distinct from 1 = verification failed)
	}
}

func run() error {
	dbPath := flag.String("db", "./ghost-trace.db", "SQLite primary-event-log path")
	blobDir := flag.String("blobs", "./blobs", "content-addressed blob-store directory")
	flag.Parse()

	ctx := context.Background()
	sub, err := substrate.Open(ctx, *dbPath, *blobDir)
	if err != nil {
		return fmt.Errorf("open substrate: %w", err)
	}
	defer func() { _ = sub.Close() }()

	report, err := verify.Verify(ctx, sub)
	if err != nil {
		return fmt.Errorf("verify: %w", err)
	}

	// Structured JSON to stdout.
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(buildPayload(report)); err != nil {
		return fmt.Errorf("encode report: %w", err)
	}

	// Human summary to stderr.
	if report.Failed() {
		fmt.Fprintf(os.Stderr,
			"verify FAILED: %d rows walked; %d hash-mismatch; %d missing-blob\n",
			report.VerifiedCount, report.HashMismatchCount, report.MissingBlobCount)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "verify OK: %d rows walked; substrate integrity confirmed\n", report.VerifiedCount)
	return nil
}

// reportPayload is the JSON shape written to stdout. Mirrors the
// verify.Report struct with snake_case keys for consumer parity with
// the ingestion service's structured-output convention.
type reportPayload struct {
	Verified           int64    `json:"verified"`
	HashMismatch       int64    `json:"hash_mismatch"`
	MissingBlob        int64    `json:"missing_blob"`
	Passed             bool     `json:"passed"`
	HashMismatchHashes []string `json:"hash_mismatch_hashes,omitempty"`
	MissingBlobHashes  []string `json:"missing_blob_hashes,omitempty"`
}

func buildPayload(r verify.Report) reportPayload {
	return reportPayload{
		Verified:           r.VerifiedCount,
		HashMismatch:       r.HashMismatchCount,
		MissingBlob:        r.MissingBlobCount,
		Passed:             !r.Failed(),
		HashMismatchHashes: r.HashMismatchHashes,
		MissingBlobHashes:  r.MissingBlobHashes,
	}
}
