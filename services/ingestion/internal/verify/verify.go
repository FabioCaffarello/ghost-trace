// Package verify implements the up-front substrate-integrity check
// referenced in docs/charter/decision-log.md §0033 §Restoration
// Procedure step 3 + §0039 (the dedicated CLI) + §0040 (orphan-blob
// detection). It walks every events-table row, recomputes each blob's
// BLAKE3 hash via the substrate's ReadBlob path (which performs hash
// verification per canonical-serialization-contract anti-pattern
// "hash-verification omitted from blob-read path"), and reports any
// mismatch / missing-blob failures. With Options.CheckOrphans, it
// additionally walks the blob-store directory and reports blobs whose
// content-hash does not appear in the events table (orphan blobs are
// harmless at the substrate layer per §0033 but operator-visible).
//
// The package surfaces the verify logic as a callable function so the
// cmd/verify binary is a thin wrapper + the same logic is exercised
// in tests. Operationally, the binary is intended for post-restore
// verification (per §0033) and for periodic substrate-integrity
// audits.
package verify

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// Options configures a Verify run.
type Options struct {
	// CheckOrphans enables the second-pass walk of the blob-store
	// directory to identify blobs whose content-hash does not appear
	// in the events table. Per §0033 / §0040, orphans are harmless at
	// the substrate layer (the events table is authoritative); they
	// are surfaced for operator inspection but do NOT cause the
	// Report to return Failed() == true.
	CheckOrphans bool
}

// Report aggregates the outcome of a verify run.
type Report struct {
	// VerifiedCount is the number of events-table rows walked,
	// regardless of pass / fail status.
	VerifiedCount int64
	// HashMismatchCount is the number of rows whose blob's recomputed
	// hash did not match the stored event_hash. Each indicates a
	// §2.1 violation per §0027 AP4/AP5.
	HashMismatchCount int64
	// MissingBlobCount is the number of rows whose payload_ref does
	// not resolve to a readable blob (file missing, I/O error, etc.).
	MissingBlobCount int64
	// HashMismatchHashes is the set of event_hash hex strings whose
	// blob recomputation failed. Bounded by HashMismatchCount.
	HashMismatchHashes []string
	// MissingBlobHashes is the set of event_hash hex strings whose
	// blob could not be read at all. Bounded by MissingBlobCount.
	MissingBlobHashes []string
	// OrphanBlobCount is the number of blob files in the blob-store
	// whose content-hash does not appear in the events table. Per
	// §0033, orphans are harmless and informational. Zero unless
	// Options.CheckOrphans was set.
	OrphanBlobCount int64
	// OrphanBlobPaths is the set of orphan blob file paths discovered
	// during the blob-store walk. Bounded by OrphanBlobCount.
	OrphanBlobPaths []string
}

// Failed reports whether the verify run surfaced any substrate-integrity
// violation. A failed report means the substrate must not be brought
// into service before operator inspection. Orphan blobs are NOT a
// failure (per §0033 / §0040 they are harmless + informational).
func (r Report) Failed() bool {
	return r.HashMismatchCount > 0 || r.MissingBlobCount > 0
}

// Verify walks every events-table row and verifies its blob via
// substrate.ReadBlob (which recomputes the BLAKE3 hash on read).
// Continues past individual failures to surface ALL violations in one
// run (rather than aborting at the first); the caller decides what to
// do with the Report.
//
// When opts.CheckOrphans is true, performs a second-pass walk of the
// blob-store directory to surface orphan blobs (blobs without a
// matching index row). Per §0033 / §0040, orphans do NOT cause
// Report.Failed() to return true.
//
// Returns a non-nil error only when the walk itself fails (e.g.
// database I/O); individual per-row failures populate the Report but
// do not return an error.
func Verify(ctx context.Context, sub *substrate.Substrate, opts Options) (Report, error) {
	var report Report
	err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		report.VerifiedCount++
		if _, err := sub.ReadBlob(ctx, row.EventHash); err != nil {
			hex := canonical.HashHex(row.EventHash)
			if errors.Is(err, substrate.ErrHashMismatch) {
				report.HashMismatchCount++
				report.HashMismatchHashes = append(report.HashMismatchHashes, hex)
			} else {
				report.MissingBlobCount++
				report.MissingBlobHashes = append(report.MissingBlobHashes, hex)
			}
		}
		return nil
	})
	if err != nil {
		return report, fmt.Errorf("verify: walk events: %w", err)
	}

	if opts.CheckOrphans {
		err := sub.WalkBlobs(ctx, func(hash [32]byte, path string) error {
			if _, lookupErr := sub.LookupRow(ctx, hash); lookupErr != nil {
				if errors.Is(lookupErr, sql.ErrNoRows) {
					report.OrphanBlobCount++
					report.OrphanBlobPaths = append(report.OrphanBlobPaths, path)
					return nil
				}
				return fmt.Errorf("LookupRow during orphan check: %w", lookupErr)
			}
			return nil
		})
		if err != nil {
			return report, fmt.Errorf("verify: walk blobs: %w", err)
		}
	}
	return report, nil
}
