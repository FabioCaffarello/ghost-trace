// Command orphan-cleanup is the operator-invoked orphan-blob cleanup
// tool per decision-log §0041 + §0040 + §0033 §Restoration Procedure
// §Orphan-blob handling on restore.
//
// Safety belts (per §0033 §Anti-Patterns "Deleting orphan blobs as
// part of automated restore"):
//
//  1. -dry-run defaults to TRUE. The default invocation reports
//     what would be deleted without touching the filesystem.
//
//  2. -confirm is required when -dry-run=false. The tool refuses to
//     delete without explicit operator confirmation; running with
//     -dry-run=false alone is a startup error.
//
//  3. -keep-newer-than defaults to 24h. Orphans newer than the age
//     floor are preserved to protect against deletion of recently-
//     orphaned blobs.
//
//  4. -max-deletions defaults to 1000. Bounded per-invocation cap
//     limits blast radius of a misconfigured invocation.
//
//  5. -exclude accepts a file of orphan hashes to preserve as
//     forensic artifacts (one lowercase-hex hash per line; lines
//     beginning with # are comments).
//
// Output: structured JSON to stdout (machine-readable record of what
// was examined / preserved / deleted) + a brief human summary to
// stderr. Exit code: 0 on success (including dry-run); 2 on tool /
// configuration error.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/orphan"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "orphan-cleanup: %v\n", err)
		os.Exit(2)
	}
}

func run() error {
	dbPath := flag.String("db", "./ghost-trace.db", "SQLite primary-event-log path")
	blobDir := flag.String("blobs", "./blobs", "content-addressed blob-store directory")
	dryRun := flag.Bool("dry-run", true, "report only; do not delete. Default true. Must be explicitly set to false to delete (and -confirm is required).")
	confirm := flag.Bool("confirm", false, "REQUIRED when -dry-run=false. Explicit operator confirmation per §0033 §Anti-Patterns.")
	excludeFile := flag.String("exclude", "", "path to a file containing one orphan-hash hex per line; matching orphans preserved. Lines beginning with # are comments.")
	keepNewerThan := flag.Duration("keep-newer-than", 24*time.Hour, "orphans newer than this duration are preserved (protection against recently-orphaned blobs).")
	maxDeletions := flag.Int("max-deletions", 1000, "upper bound on number of orphan deletions per invocation.")
	actor := flag.String("actor", "", "OPTIONAL per decision-log §0119 (auth-scope RFC Open Question 4 discharge): when non-empty, commits an OrphanCleanupAudit Cat I record + paired IngestionEvent (channel=\"cli\", client_common_name=<actor>) BEFORE deletion — mirrors the HTTP T3 audit-then-delete contract per §0104. When empty, preserves the §0033 local-shell-trust no-audit behavior (no substrate write).")
	flag.Parse()

	if !*dryRun && !*confirm {
		return fmt.Errorf("refusing to delete without -confirm. Re-run with -dry-run (default) to preview, or add -confirm to perform deletion.")
	}

	excluded, err := loadExclusions(*excludeFile)
	if err != nil {
		return fmt.Errorf("load exclusions: %w", err)
	}

	ctx := context.Background()
	sub, err := substrate.Open(ctx, *dbPath, *blobDir)
	if err != nil {
		return fmt.Errorf("open substrate: %w", err)
	}
	defer func() { _ = sub.Close() }()

	opts := orphan.Options{
		DryRun:         *dryRun,
		ExcludedHashes: excluded,
		KeepNewerThan:  *keepNewerThan,
		MaxDeletions:   *maxDeletions,
	}

	invokedAt := time.Now().UnixNano()
	excludedHashList := sortedExclusionHashes(excluded)
	keepNewerThanSeconds := int64((*keepNewerThan).Seconds())

	var auditEventHash, ingestionEventHash string
	var auditCallback func(plan orphan.Report) error
	if *actor != "" {
		auditCallback = func(plan orphan.Report) error {
			audit := &eventsv1.OrphanCleanupAudit{
				InvokedAt:                invokedAt,
				DryRun:                   *dryRun,
				Confirm:                  *confirm,
				KeepNewerThanSeconds:     keepNewerThanSeconds,
				MaxDeletions:             int64(*maxDeletions),
				ExcludedHashes:           excludedHashList,
				ExaminedCount:            plan.Examined,
				OrphansFound:             plan.OrphansFound,
				PlannedDeletionHashes:    hashesOfDeletions(plan.Deleted),
				PreservedByExcludeHashes: hashesOfDeletions(plan.PreservedByExclude),
				PreservedByAgeHashes:     hashesOfDeletions(plan.PreservedByAge),
				PreservedByMaxHashes:     hashesOfDeletions(plan.PreservedByMax),
			}
			hashes, err := commitAuditPair(ctx, sub, audit, *actor, invokedAt)
			if err != nil {
				return err
			}
			auditEventHash, ingestionEventHash = hashes.audit, hashes.ingestion
			return nil
		}
	}

	report, err := orphan.AuditedCleanup(ctx, sub, opts, auditCallback)
	if err != nil {
		return fmt.Errorf("cleanup: %w", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(buildPayload(report, *dryRun, auditEventHash, ingestionEventHash)); err != nil {
		return fmt.Errorf("encode report: %w", err)
	}

	action := "would-delete"
	if !*dryRun {
		action = "deleted"
	}
	if *actor != "" {
		fmt.Fprintf(os.Stderr,
			"orphan-cleanup: examined %d blobs; %d orphans found; %s %d (preserved by exclude: %d; by age: %d; by max-cap: %d) actor=%q audit=%s ingestion=%s\n",
			report.Examined, report.OrphansFound, action, len(report.Deleted),
			len(report.PreservedByExclude), len(report.PreservedByAge), len(report.PreservedByMax),
			*actor, auditEventHash, ingestionEventHash)
	} else {
		fmt.Fprintf(os.Stderr,
			"orphan-cleanup: examined %d blobs; %d orphans found; %s %d (preserved by exclude: %d; by age: %d; by max-cap: %d)\n",
			report.Examined, report.OrphansFound, action, len(report.Deleted),
			len(report.PreservedByExclude), len(report.PreservedByAge), len(report.PreservedByMax))
	}
	return nil
}

// auditHashes is the per-commit hash pair returned by the audit
// callback. Both are lowercase-hex content-hashes (32 bytes raw).
type auditHashes struct {
	audit     string
	ingestion string
}

// commitAuditPair commits the OrphanCleanupAudit + a paired IngestionEvent
// via substrate.AppendPair. Mirrors the HTTP T3 admin.go contract: the
// audit is the recovery contract per RFC item 4; deletion follows only
// when the audit lands. Channel is "cli" + client_common_name = actor
// per the §0097 CLI per-actor-attribution shape.
func commitAuditPair(ctx context.Context, sub *substrate.Substrate, audit *eventsv1.OrphanCleanupAudit, actor string, committedAt int64) (auditHashes, error) {
	auditPayload, auditHash, err := canonical.MarshalAndHash(audit)
	if err != nil {
		return auditHashes{}, fmt.Errorf("marshal audit: %w", err)
	}
	auditHex := canonical.HashHex(auditHash)
	auditRow := substrate.EventRow{
		EventHash:   auditHash,
		EventTime:   audit.InvokedAt,
		MessageType: string(audit.ProtoReflect().Descriptor().FullName()),
		PayloadRef:  auditHex[:2] + "/" + auditHex[2:],
		CommittedAt: committedAt,
	}

	ingEv := &eventsv1.IngestionEvent{
		PrimaryEventHash: auditHash[:],
		ReceivedAt:       committedAt,
		IngestedAt:       committedAt,
		Channel:          "cli",
		ClientCommonName: actor,
	}
	ingPayload, ingHash, err := canonical.MarshalAndHash(ingEv)
	if err != nil {
		return auditHashes{}, fmt.Errorf("marshal ingestion event: %w", err)
	}
	ingHex := canonical.HashHex(ingHash)
	ingRow := substrate.EventRow{
		EventHash:   ingHash,
		EventTime:   committedAt,
		MessageType: string(ingEv.ProtoReflect().Descriptor().FullName()),
		PayloadRef:  ingHex[:2] + "/" + ingHex[2:],
		CommittedAt: committedAt,
	}
	if err := sub.AppendPair(ctx, auditRow, auditPayload, ingRow, ingPayload); err != nil {
		return auditHashes{}, fmt.Errorf("append pair (audit %s, ingestion %s): %w", auditHex, ingHex, err)
	}
	return auditHashes{audit: auditHex, ingestion: ingHex}, nil
}

// sortedExclusionHashes returns the keys of excluded sorted ascending —
// used to build the audit record's excluded_hashes field deterministically
// so the audit hash is invariant under map-iteration order.
func sortedExclusionHashes(excluded map[string]bool) []string {
	if len(excluded) == 0 {
		return nil
	}
	out := make([]string, 0, len(excluded))
	for h := range excluded {
		out = append(out, h)
	}
	sortStrings(out)
	return out
}

// sortStrings is a tiny non-import-burden ascending sort. Avoids pulling
// "sort" purely for this single use; mirrors the §0050 "successor-set
// ascending-sort" idempotency pattern.
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}

// hashesOfDeletions returns the lowercase-hex hash field of each record
// in the order recorded by orphan.AuditedCleanup. Used to populate the
// audit's planned_deletion_hashes + preserved_by_* fields.
func hashesOfDeletions(records []orphan.DeletionRecord) []string {
	if len(records) == 0 {
		return nil
	}
	out := make([]string, len(records))
	for i, r := range records {
		out[i] = r.Hash
	}
	return out
}

// loadExclusions parses the exclusion file (one lowercase-hex hash
// per line; # comments + blank lines ignored) into a lookup map.
// Empty path returns nil (no exclusions).
func loadExclusions(path string) (map[string]bool, error) {
	if path == "" {
		return nil, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %q: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	out := map[string]bool{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out[line] = true
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %q: %w", path, err)
	}
	return out, nil
}

// reportPayload is the JSON shape written to stdout. Snake-case keys
// for parity with the ingestion service's structured-output
// convention. AuditEventHash + IngestionEventHash are surfaced ONLY
// when --actor was non-empty (the audit-on-commit path per decision-log
// §0119); absent under the §0033 local-shell-trust default.
type reportPayload struct {
	Examined           int64             `json:"examined"`
	OrphansFound       int64             `json:"orphans_found"`
	DryRun             bool              `json:"dry_run"`
	DeletedCount       int               `json:"deleted_count"`
	AuditEventHash     string            `json:"audit_event_hash,omitempty"`
	IngestionEventHash string            `json:"ingestion_event_hash,omitempty"`
	Deleted            []deletionPayload `json:"deleted,omitempty"`
	PreservedByExclude []deletionPayload `json:"preserved_by_exclude,omitempty"`
	PreservedByAge     []deletionPayload `json:"preserved_by_age,omitempty"`
	PreservedByMax     []deletionPayload `json:"preserved_by_max,omitempty"`
}

type deletionPayload struct {
	Hash  string `json:"hash"`
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
}

func buildPayload(r orphan.Report, dryRun bool, auditEventHash, ingestionEventHash string) reportPayload {
	return reportPayload{
		Examined:           r.Examined,
		OrphansFound:       r.OrphansFound,
		DryRun:             dryRun,
		DeletedCount:       len(r.Deleted),
		AuditEventHash:     auditEventHash,
		IngestionEventHash: ingestionEventHash,
		Deleted:            toPayload(r.Deleted),
		PreservedByExclude: toPayload(r.PreservedByExclude),
		PreservedByAge:     toPayload(r.PreservedByAge),
		PreservedByMax:     toPayload(r.PreservedByMax),
	}
}

func toPayload(records []orphan.DeletionRecord) []deletionPayload {
	if len(records) == 0 {
		return nil
	}
	out := make([]deletionPayload, len(records))
	for i, r := range records {
		out[i] = deletionPayload{Hash: r.Hash, Path: r.Path, Bytes: r.Bytes}
	}
	return out
}
