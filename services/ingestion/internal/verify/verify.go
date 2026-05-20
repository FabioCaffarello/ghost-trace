// Package verify implements the up-front substrate-integrity check
// referenced in docs/charter/decision-log.md §0033 §Restoration
// Procedure step 3 + §Open Questions ("verify CLI tool"). It walks
// every events-table row, recomputes each blob's BLAKE3 hash via the
// substrate's ReadBlob path (which performs hash verification per
// canonical-serialization-contract anti-pattern "hash-verification
// omitted from blob-read path"), and reports any mismatch /
// missing-blob failures.
//
// The package surfaces the verify logic as a callable function so the
// cmd/verify binary is a thin wrapper + the same logic is exercised
// in tests. Operationally, the binary is intended for post-restore
// verification (per §0033) and for periodic substrate-integrity
// audits.
package verify

import (
	"context"
	"errors"
	"fmt"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

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
}

// Failed reports whether the verify run surfaced any substrate-integrity
// violation. A failed report means the substrate must not be brought
// into service before operator inspection.
func (r Report) Failed() bool {
	return r.HashMismatchCount > 0 || r.MissingBlobCount > 0
}

// Verify walks every events-table row and verifies its blob via
// substrate.ReadBlob (which recomputes the BLAKE3 hash on read).
// Continues past individual failures to surface ALL violations in one
// run (rather than aborting at the first); the caller decides what to
// do with the Report.
//
// Returns a non-nil error only when the walk itself fails (e.g.
// database I/O); individual per-row failures populate the Report but
// do not return an error.
func Verify(ctx context.Context, sub *substrate.Substrate) (Report, error) {
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
	return report, nil
}
