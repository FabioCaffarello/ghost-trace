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

	report, err := orphan.Cleanup(ctx, sub, orphan.Options{
		DryRun:         *dryRun,
		ExcludedHashes: excluded,
		KeepNewerThan:  *keepNewerThan,
		MaxDeletions:   *maxDeletions,
	})
	if err != nil {
		return fmt.Errorf("cleanup: %w", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(buildPayload(report, *dryRun)); err != nil {
		return fmt.Errorf("encode report: %w", err)
	}

	action := "would-delete"
	if !*dryRun {
		action = "deleted"
	}
	fmt.Fprintf(os.Stderr,
		"orphan-cleanup: examined %d blobs; %d orphans found; %s %d (preserved by exclude: %d; by age: %d; by max-cap: %d)\n",
		report.Examined, report.OrphansFound, action, len(report.Deleted),
		len(report.PreservedByExclude), len(report.PreservedByAge), len(report.PreservedByMax))
	return nil
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
// convention.
type reportPayload struct {
	Examined              int64                   `json:"examined"`
	OrphansFound          int64                   `json:"orphans_found"`
	DryRun                bool                    `json:"dry_run"`
	DeletedCount          int                     `json:"deleted_count"`
	Deleted               []deletionPayload       `json:"deleted,omitempty"`
	PreservedByExclude    []deletionPayload       `json:"preserved_by_exclude,omitempty"`
	PreservedByAge        []deletionPayload       `json:"preserved_by_age,omitempty"`
	PreservedByMax        []deletionPayload       `json:"preserved_by_max,omitempty"`
}

type deletionPayload struct {
	Hash  string `json:"hash"`
	Path  string `json:"path"`
	Bytes int64  `json:"bytes"`
}

func buildPayload(r orphan.Report, dryRun bool) reportPayload {
	return reportPayload{
		Examined:           r.Examined,
		OrphansFound:       r.OrphansFound,
		DryRun:             dryRun,
		DeletedCount:       len(r.Deleted),
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
