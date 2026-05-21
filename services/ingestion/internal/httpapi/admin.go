// Package httpapi T3 substrate-admin endpoints per decision-log §0094
// classification + §0098 + §0104.
//
// HTTP T3 endpoints commit a Cat I audit record (committed via
// AppendPair paired with an IngestionEvent) BEFORE performing the
// underlying mutation. The audit + its paired IngestionEvent are the
// recovery contract per RFC architecture-http-auth-scope-model item 4:
// partial-mutation failure leaves the audit + surviving state
// recoverable — operator re-runs the mutation against the audit's
// recorded plan.
package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/orphan"
)

// orphanCleanupResponse is the wire-shape of the T3 orphan-cleanup
// HTTP response body. Mirrors the JSON shape the CLI emits on stdout
// (the CLI does not commit an audit; the HTTP path does — the audit
// hashes are the HTTP-specific addition).
type orphanCleanupResponse struct {
	// AuditEventHash is the lowercase-hex content-hash of the
	// committed OrphanCleanupAudit record. Operators reference this
	// hash to retrieve the audit's planned-deletion list (the
	// recovery contract per RFC item 4).
	AuditEventHash string `json:"audit_event_hash"`

	// IngestionEventHash is the lowercase-hex content-hash of the
	// paired IngestionEvent (channel + mTLS client identity).
	IngestionEventHash string `json:"ingestion_event_hash"`

	// InvokedAtNs is the audit-recorded invocation time (Unix
	// nanoseconds). Set at endpoint entry, before orphan
	// identification.
	InvokedAtNs int64 `json:"invoked_at_ns"`

	// DryRun reflects the operator-supplied --dry-run choice
	// (defaults to true at the HTTP layer for operator safety).
	DryRun bool `json:"dry_run"`

	// Confirm reflects the operator-supplied --confirm choice. The
	// non-dry-run path requires confirm=true; this is validated
	// before AuditedCleanup is invoked.
	Confirm bool `json:"confirm"`

	// Examined is the total blob count walked.
	Examined int64 `json:"examined"`

	// OrphansFound is the orphan count identified (i.e., blobs whose
	// hash does NOT appear in the events table).
	OrphansFound int64 `json:"orphans_found"`

	// Deleted is the count of planned deletions actually performed.
	// When DryRun is true, this is the would-be-deleted count
	// (planning-only); when DryRun is false, this is the actual
	// deletion count.
	Deleted int `json:"deleted"`

	// PreservedByExclude is the count of orphans matched + preserved
	// by the excluded_hashes filter.
	PreservedByExclude int `json:"preserved_by_exclude"`

	// PreservedByAge is the count of orphans preserved by the
	// keep_newer_than_seconds age floor.
	PreservedByAge int `json:"preserved_by_age"`

	// PreservedByMax is the count of orphans preserved because the
	// max_deletions cap was reached.
	PreservedByMax int `json:"preserved_by_max"`

	// PlannedDeletionHashes is the lowercase-hex hash list the audit
	// recorded as the planned-deletion subset. This list IS the
	// recovery contract per RFC item 4: on partial-deletion failure,
	// operator re-runs the deletion against this list.
	PlannedDeletionHashes []string `json:"planned_deletion_hashes"`
}

// handleAdminOrphanCleanup implements POST /v1/admin/orphan-cleanup
// per decision-log §0104. T3 substrate-admin tier per §0094 + §0098.
//
// Query parameters (all optional except as noted):
//   - dry_run=false        — defaults to true for operator safety;
//     must be explicitly set to false to perform deletion.
//   - confirm=true         — required when dry_run=false; the
//     CLI-mirror -confirm safety belt.
//   - keep_newer_than_seconds=<int>  — preserves orphans whose file
//     mtime is newer than (invoked_at - this * 1e9 ns). 0 disables.
//   - max_deletions=<int>  — caps the deletion count per invocation.
//     0 disables the cap.
//   - excluded_hash=<hex>  — repeatable; lowercase-hex BLAKE3-256;
//     marks the orphan as preserved-by-exclude regardless of other
//     filters.
//
// The audit + its paired IngestionEvent commit BEFORE any blob
// deletion (RFC item 4 audit-then-delete contract). On audit-commit
// failure: no deletion occurs and the substrate is unchanged. On
// per-orphan deletion failure mid-flight: the audit + the surviving
// blobs are recoverable; operator re-runs against the audit's
// PlannedDeletionHashes.
//
// Response: 200 + orphanCleanupResponse JSON. 400 on parameter
// validation failure. 405 on non-POST. 503 when the handler was
// constructed without WithSubstrate (read access disabled). 500 on
// audit-commit or deletion failure (recoverable per RFC item 4: the
// audit's hash list is the recovery contract).
func (h *Handler) handleAdminOrphanCleanup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeIngestError(w, http.StatusMethodNotAllowed, "method not allowed; POST required")
		return
	}
	if h.sub == nil {
		writeIngestError(w, http.StatusServiceUnavailable, "substrate access not configured (WithSubstrate)")
		return
	}

	q := r.URL.Query()
	dryRun := q.Get("dry_run") != "false" // default true for operator safety
	confirm := q.Get("confirm") == "true"

	keepNewerThanSeconds, err := parseInt64QueryParam(q, "keep_newer_than_seconds")
	if err != nil {
		writeIngestError(w, http.StatusBadRequest, fmt.Sprintf("keep_newer_than_seconds: %v", err))
		return
	}
	if keepNewerThanSeconds < 0 {
		writeIngestError(w, http.StatusBadRequest, "keep_newer_than_seconds must be >= 0")
		return
	}

	maxDeletions, err := parseInt64QueryParam(q, "max_deletions")
	if err != nil {
		writeIngestError(w, http.StatusBadRequest, fmt.Sprintf("max_deletions: %v", err))
		return
	}
	if maxDeletions < 0 {
		writeIngestError(w, http.StatusBadRequest, "max_deletions must be >= 0")
		return
	}

	excludedHashes := q["excluded_hash"]
	excludedHashMap := make(map[string]bool, len(excludedHashes))
	for _, hash := range excludedHashes {
		excludedHashMap[strings.ToLower(hash)] = true
	}

	if !dryRun && !confirm {
		writeIngestError(w, http.StatusBadRequest, "confirm=true required when dry_run=false")
		return
	}

	invokedAt := time.Now().UnixNano()
	env := envelopeForRequest(r)

	opts := orphan.Options{
		DryRun:         dryRun,
		ExcludedHashes: excludedHashMap,
		KeepNewerThan:  time.Duration(keepNewerThanSeconds) * time.Second,
		MaxDeletions:   int(maxDeletions),
		Now:            func() time.Time { return time.Unix(0, invokedAt) },
	}

	var auditEventHash, auditIngestionHash string
	auditCallback := func(plan orphan.Report) error {
		audit := &eventsv1.OrphanCleanupAudit{
			InvokedAt:                invokedAt,
			DryRun:                   dryRun,
			Confirm:                  confirm,
			KeepNewerThanSeconds:     keepNewerThanSeconds,
			MaxDeletions:             maxDeletions,
			ExcludedHashes:           excludedHashes,
			ExaminedCount:            plan.Examined,
			OrphansFound:             plan.OrphansFound,
			PlannedDeletionHashes:    hashesOf(plan.Deleted),
			PreservedByExcludeHashes: hashesOf(plan.PreservedByExclude),
			PreservedByAgeHashes:     hashesOf(plan.PreservedByAge),
			PreservedByMaxHashes:     hashesOf(plan.PreservedByMax),
		}
		report, err := h.doAppend(r.Context(), audit, invokedAt, env)
		if err != nil {
			return fmt.Errorf("commit audit: %w", err)
		}
		auditEventHash = report.EventHashHex
		auditIngestionHash = report.IngestionEventHashHex
		return nil
	}

	report, err := orphan.AuditedCleanup(r.Context(), h.sub, opts, auditCallback)
	if err != nil {
		// The audit may have committed before the failure (audit-then-
		// delete contract): the audit's hash list is the recovery
		// contract per RFC item 4. Surface 500 + structured body so the
		// operator can retrieve the audit by hash and re-run.
		if h.fatal != nil {
			h.fatal.ReportFatal(fmt.Errorf("admin orphan-cleanup: %w", err))
		}
		writeIngestError(w, http.StatusInternalServerError, fmt.Sprintf("orphan-cleanup failure: %v (audit committed at %s; replay against audit's planned-deletion hash list)", err, auditEventHash))
		return
	}

	out := orphanCleanupResponse{
		AuditEventHash:        auditEventHash,
		IngestionEventHash:    auditIngestionHash,
		InvokedAtNs:           invokedAt,
		DryRun:                dryRun,
		Confirm:               confirm,
		Examined:              report.Examined,
		OrphansFound:          report.OrphansFound,
		Deleted:               len(report.Deleted),
		PreservedByExclude:    len(report.PreservedByExclude),
		PreservedByAge:        len(report.PreservedByAge),
		PreservedByMax:        len(report.PreservedByMax),
		PlannedDeletionHashes: hashesOf(report.Deleted),
	}
	writeJSON(w, http.StatusOK, out)
}

// parseInt64QueryParam parses an int64 query parameter from q. Empty
// or absent returns (0, nil) — the caller's default. Non-empty
// non-integer values return an error with the offending raw value
// embedded.
func parseInt64QueryParam(q map[string][]string, key string) (int64, error) {
	raw := ""
	if vs, ok := q[key]; ok && len(vs) > 0 {
		raw = vs[0]
	}
	if raw == "" {
		return 0, nil
	}
	return strconv.ParseInt(raw, 10, 64)
}

// hashesOf extracts the Hash field from each DeletionRecord into a
// slice of lowercase-hex strings.
func hashesOf(recs []orphan.DeletionRecord) []string {
	out := make([]string, 0, len(recs))
	for _, r := range recs {
		out = append(out, r.Hash)
	}
	return out
}

// writeJSON emits a JSON response with the given status code and body.
// Marshalling errors fall back to a generic 500 + plain-text message.
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		// Best-effort: the response headers are already written. Log
		// via the fatal channel would require plumbing; at this point
		// the connection write has likely failed too.
		_ = err
	}
}

// ensure context-parameter linter does not flag handleAdminOrphanCleanup
// for unused-context — it threads r.Context() into AuditedCleanup and
// doAppend.
var _ = context.Background