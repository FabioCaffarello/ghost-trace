// Package orphan implements operator-invoked orphan-blob cleanup per
// docs/charter/decision-log.md §0041 (this entry) + §0040 (detection)
// + §0033 §Restoration Procedure §Orphan-blob handling on restore.
//
// Per §0033 §Anti-Patterns ("Deleting orphan blobs as part of
// automated restore"): orphan deletion MUST be operator-invoked.
// This package's Cleanup function is invoked from the orphan-cleanup
// CLI; it implements dry-run-by-default, exclusion-list, age-floor,
// and bounded-deletions safety belts. The CLI additionally requires
// an explicit -confirm flag when -dry-run=false; that safety belt
// lives at the CLI layer, not in this package, so testing can drive
// the package directly without disabling the operator-confirmation
// guard.
package orphan

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// Options configures a Cleanup run.
type Options struct {
	// DryRun, when true, reports what WOULD be deleted but performs
	// no filesystem mutation. Default behavior for operator safety.
	DryRun bool

	// ExcludedHashes is a set of lowercase-hex hashes whose orphan
	// status is acknowledged + preserved. Use for orphans the
	// operator wants to retain as forensic artifacts.
	ExcludedHashes map[string]bool

	// KeepNewerThan preserves orphans whose file mtime is more
	// recent than this duration. Default 0 disables the age floor
	// (callers should pass a non-zero default in production tooling
	// to protect against deletion of recently-orphaned blobs).
	KeepNewerThan time.Duration

	// MaxDeletions caps the number of orphan deletions per
	// invocation. Excess orphans are recorded as PreservedByMax and
	// skipped. Default 0 disables the cap.
	MaxDeletions int

	// Now is injectable for deterministic tests. Defaults to
	// time.Now when nil.
	Now func() time.Time
}

func (o Options) now() time.Time {
	if o.Now != nil {
		return o.Now()
	}
	return time.Now()
}

// DeletionRecord captures the per-orphan metadata for reporting.
type DeletionRecord struct {
	Hash  string
	Path  string
	Bytes int64
}

// Report aggregates the outcome of a Cleanup run.
type Report struct {
	// Examined is the number of blobs walked, regardless of orphan
	// status.
	Examined int64

	// OrphansFound is the number of blobs whose hash does not appear
	// in the events table.
	OrphansFound int64

	// Deleted is the set of orphans whose deletion was performed
	// (or would-be-performed under DryRun). Bounded by len(Deleted)
	// <= MaxDeletions when MaxDeletions > 0.
	Deleted []DeletionRecord

	// PreservedByExclude is the set of orphans the exclusion list
	// matched + preserved.
	PreservedByExclude []DeletionRecord

	// PreservedByAge is the set of orphans newer than KeepNewerThan
	// that were preserved.
	PreservedByAge []DeletionRecord

	// PreservedByMax is the set of orphans that would have been
	// deleted but were preserved because MaxDeletions was reached.
	PreservedByMax []DeletionRecord
}

// Cleanup walks the blob-store, identifies orphans (blobs whose hash
// does NOT appear in the events table), applies the configured
// filters (exclusion list, age floor, max-deletions cap), and either
// reports the would-be-deleted set (DryRun) or performs the deletions.
//
// Returns a non-nil error only when the walk or substrate-lookup
// fails; per-orphan filesystem errors during deletion abort the run
// (a partial run is less safe than a fail-fast).
func Cleanup(ctx context.Context, sub *substrate.Substrate, opts Options) (Report, error) {
	return AuditedCleanup(ctx, sub, opts, nil)
}

// AuditedCleanup is Cleanup with an audit-callback hook invoked between
// the planning and deletion phases per RFC architecture-http-auth-
// scope-model item 4 audit-then-delete contract. The callback receives
// the planned deletion Report (with Deleted populated as the
// would-be-deleted set, identical to what DryRun=true would produce).
// Returning a non-nil error from the callback aborts before any
// deletion occurs (the substrate is unchanged).
//
// The HTTP-T3 path passes a callback that commits an OrphanCleanupAudit
// record (paired with an IngestionEvent) via substrate.AppendPair; the
// CLI path passes nil and uses the unaudited Cleanup behavior per
// decision-log §0041 local-shell-trust assumption.
//
// When opts.DryRun is true, no deletion occurs regardless of the
// audit callback's return value. When opts.DryRun is false and the
// callback returns nil (or is nil), the planned deletions are
// performed.
func AuditedCleanup(ctx context.Context, sub *substrate.Substrate, opts Options, audit func(plan Report) error) (Report, error) {
	var report Report
	now := opts.now()

	err := sub.WalkBlobs(ctx, func(hash [32]byte, path string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		report.Examined++

		_, lookupErr := sub.LookupRow(ctx, hash)
		if lookupErr == nil {
			return nil // blob has an index row; not orphan
		}
		if !errors.Is(lookupErr, sql.ErrNoRows) {
			return fmt.Errorf("orphan.Cleanup: LookupRow %s: %w", canonical.HashHex(hash), lookupErr)
		}

		report.OrphansFound++
		hex := canonical.HashHex(hash)

		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("orphan.Cleanup: stat %s: %w", path, err)
		}
		rec := DeletionRecord{Hash: hex, Path: path, Bytes: info.Size()}

		// Filter: exclusion list takes precedence over all other
		// filters (operator explicitly preserved this orphan).
		if opts.ExcludedHashes != nil && opts.ExcludedHashes[hex] {
			report.PreservedByExclude = append(report.PreservedByExclude, rec)
			return nil
		}
		// Filter: age floor.
		if opts.KeepNewerThan > 0 && now.Sub(info.ModTime()) < opts.KeepNewerThan {
			report.PreservedByAge = append(report.PreservedByAge, rec)
			return nil
		}
		// Filter: max-deletions cap.
		if opts.MaxDeletions > 0 && int64(len(report.Deleted)) >= int64(opts.MaxDeletions) {
			report.PreservedByMax = append(report.PreservedByMax, rec)
			return nil
		}

		// Planned for deletion — recorded but not yet deleted.
		report.Deleted = append(report.Deleted, rec)
		return nil
	})
	if err != nil {
		return report, err
	}

	// Audit hook between planning and deletion (RFC item 4
	// audit-then-delete contract). A non-nil error aborts before any
	// deletion; the substrate is unchanged.
	if audit != nil {
		if err := audit(report); err != nil {
			return report, fmt.Errorf("orphan.AuditedCleanup: audit hook: %w", err)
		}
	}

	// Deletion phase: perform deletions of the planned set. DryRun
	// skips deletion entirely. Per-orphan filesystem errors abort the
	// run; the audit's hash list (committed by the audit hook before
	// this point) is the recovery contract for the surviving subset.
	if !opts.DryRun {
		for _, rec := range report.Deleted {
			if err := os.Remove(rec.Path); err != nil {
				return report, fmt.Errorf("orphan.AuditedCleanup: remove %s: %w", rec.Path, err)
			}
		}
	}
	return report, nil
}
