package orphan

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// newPopulatedSubstrate ingests `count` test messages via the full
// ingest pipeline so the substrate has legitimate (non-orphan) blobs.
// Returns the substrate handle.
func newPopulatedSubstrate(t *testing.T, count int) *substrate.Substrate {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	in := ingest.New(sub, time.Now)
	for i := 0; i < count; i++ {
		msg := &eventsv1.DeclaredSession{
			DeclaredAt:        int64(1000 + i),
			ActorRef:          "actor-" + string(rune('a'+i)),
			SessionDescriptor: []byte("session-" + string(rune('A'+i))),
		}
		if _, err := in.Append(ctx, msg, msg.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("Append %d: %v", i, err)
		}
	}
	return sub
}

// plantOrphan writes a blob to the blob-store at the correct
// content-addressed path without committing an events-table row for
// it. Returns the orphan's hex hash + on-disk path. The optional
// mtime sets the file's modification time (zero defaults to "now").
func plantOrphan(t *testing.T, sub *substrate.Substrate, content []byte, mtime time.Time) (hex string, path string) {
	t.Helper()
	hash := canonical.Hash(content)
	hex = canonical.HashHex(hash)
	shard := filepath.Join(sub.BlobDir(), hex[:2])
	if err := os.MkdirAll(shard, 0o755); err != nil {
		t.Fatalf("mkdir shard: %v", err)
	}
	path = filepath.Join(shard, hex[2:])
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write orphan: %v", err)
	}
	if !mtime.IsZero() {
		if err := os.Chtimes(path, mtime, mtime); err != nil {
			t.Fatalf("chtimes: %v", err)
		}
	}
	return hex, path
}

func TestCleanupDryRunDoesNotDelete(t *testing.T) {
	ctx := context.Background()
	sub := newPopulatedSubstrate(t, 2)
	hex, path := plantOrphan(t, sub, []byte("orphan-dry-run"), time.Time{})

	report, err := Cleanup(ctx, sub, Options{DryRun: true})
	if err != nil {
		t.Fatalf("Cleanup: %v", err)
	}
	if report.OrphansFound != 1 {
		t.Errorf("OrphansFound: got %d, want 1", report.OrphansFound)
	}
	if len(report.Deleted) != 1 {
		t.Fatalf("Deleted len: got %d, want 1", len(report.Deleted))
	}
	if report.Deleted[0].Hash != hex {
		t.Errorf("Deleted[0].Hash: got %q, want %q", report.Deleted[0].Hash, hex)
	}
	// File must still exist after dry-run.
	if _, err := os.Stat(path); err != nil {
		t.Errorf("orphan was deleted during dry-run: %v", err)
	}
}

func TestCleanupConfirmedDeletion(t *testing.T) {
	ctx := context.Background()
	sub := newPopulatedSubstrate(t, 2)
	hex, path := plantOrphan(t, sub, []byte("orphan-confirmed"), time.Time{})

	report, err := Cleanup(ctx, sub, Options{DryRun: false})
	if err != nil {
		t.Fatalf("Cleanup: %v", err)
	}
	if len(report.Deleted) != 1 {
		t.Fatalf("Deleted len: got %d, want 1", len(report.Deleted))
	}
	if report.Deleted[0].Hash != hex {
		t.Errorf("Deleted[0].Hash: got %q, want %q", report.Deleted[0].Hash, hex)
	}
	// File must be gone.
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("expected orphan file deleted; stat err = %v", err)
	}
}

func TestCleanupExclusionListPreserves(t *testing.T) {
	ctx := context.Background()
	sub := newPopulatedSubstrate(t, 1)
	hex, path := plantOrphan(t, sub, []byte("orphan-protected"), time.Time{})

	report, err := Cleanup(ctx, sub, Options{
		DryRun:         false,
		ExcludedHashes: map[string]bool{hex: true},
	})
	if err != nil {
		t.Fatalf("Cleanup: %v", err)
	}
	if len(report.Deleted) != 0 {
		t.Errorf("Deleted: got %d, want 0 (orphan should be excluded)", len(report.Deleted))
	}
	if len(report.PreservedByExclude) != 1 {
		t.Errorf("PreservedByExclude len: got %d, want 1", len(report.PreservedByExclude))
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("excluded orphan was deleted: %v", err)
	}
}

func TestCleanupAgeFilterPreserves(t *testing.T) {
	ctx := context.Background()
	sub := newPopulatedSubstrate(t, 1)

	// Plant orphan with a fresh mtime (now); KeepNewerThan 1 hour
	// means this orphan is preserved.
	freshMTime := time.Now()
	_, freshPath := plantOrphan(t, sub, []byte("orphan-fresh"), freshMTime)

	// Plant orphan with an old mtime (2 hours ago); should be eligible
	// for deletion under the same 1-hour age floor.
	oldMTime := time.Now().Add(-2 * time.Hour)
	_, oldPath := plantOrphan(t, sub, []byte("orphan-old"), oldMTime)

	report, err := Cleanup(ctx, sub, Options{
		DryRun:        false,
		KeepNewerThan: 1 * time.Hour,
	})
	if err != nil {
		t.Fatalf("Cleanup: %v", err)
	}

	if len(report.Deleted) != 1 {
		t.Errorf("Deleted: got %d, want 1 (only old orphan)", len(report.Deleted))
	}
	if len(report.PreservedByAge) != 1 {
		t.Errorf("PreservedByAge: got %d, want 1 (fresh orphan)", len(report.PreservedByAge))
	}

	// Fresh file preserved.
	if _, err := os.Stat(freshPath); err != nil {
		t.Errorf("fresh orphan was deleted: %v", err)
	}
	// Old file deleted.
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Errorf("old orphan not deleted: stat err = %v", err)
	}
}

func TestCleanupMaxDeletionsCap(t *testing.T) {
	ctx := context.Background()
	sub := newPopulatedSubstrate(t, 1)

	// Plant 5 orphans with old mtimes (so age filter doesn't preserve them).
	for i := 0; i < 5; i++ {
		plantOrphan(t, sub, []byte{byte('a' + i), '-', 'o', 'r', 'p', 'h', 'a', 'n'}, time.Now().Add(-2*time.Hour))
	}

	report, err := Cleanup(ctx, sub, Options{
		DryRun:       false,
		MaxDeletions: 2,
	})
	if err != nil {
		t.Fatalf("Cleanup: %v", err)
	}
	if report.OrphansFound != 5 {
		t.Errorf("OrphansFound: got %d, want 5", report.OrphansFound)
	}
	if len(report.Deleted) != 2 {
		t.Errorf("Deleted: got %d, want 2 (capped)", len(report.Deleted))
	}
	if len(report.PreservedByMax) != 3 {
		t.Errorf("PreservedByMax: got %d, want 3 (5 - 2 cap)", len(report.PreservedByMax))
	}
}

func TestCleanupSkipsNonOrphans(t *testing.T) {
	ctx := context.Background()
	sub := newPopulatedSubstrate(t, 3)

	// No orphans planted; the 3 ingested events produced 6 blobs (primary +
	// enrichment per §0038). All have index rows; none are orphan.
	report, err := Cleanup(ctx, sub, Options{DryRun: false})
	if err != nil {
		t.Fatalf("Cleanup: %v", err)
	}
	if report.Examined != 6 {
		t.Errorf("Examined: got %d, want 6 (3 ingests × 2 per §0038 pairing)", report.Examined)
	}
	if report.OrphansFound != 0 {
		t.Errorf("OrphansFound: got %d, want 0", report.OrphansFound)
	}
	if len(report.Deleted) != 0 {
		t.Errorf("Deleted: got %d, want 0", len(report.Deleted))
	}
}

func TestCleanupEmptySubstrate(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	report, err := Cleanup(ctx, sub, Options{DryRun: false})
	if err != nil {
		t.Fatalf("Cleanup: %v", err)
	}
	if report.Examined != 0 || report.OrphansFound != 0 || len(report.Deleted) != 0 {
		t.Errorf("empty substrate: %+v", report)
	}
}

func TestCleanupDryRunRespectsAllFilters(t *testing.T) {
	// Verify that dry-run still applies exclusion + age filters
	// correctly (dry-run only suppresses the actual deletion, not the
	// filtering logic).
	ctx := context.Background()
	sub := newPopulatedSubstrate(t, 1)

	freshHex, _ := plantOrphan(t, sub, []byte("dry-fresh"), time.Now())
	oldHex, _ := plantOrphan(t, sub, []byte("dry-old"), time.Now().Add(-2*time.Hour))
	_ = freshHex
	_ = oldHex

	report, err := Cleanup(ctx, sub, Options{
		DryRun:        true,
		KeepNewerThan: 1 * time.Hour,
	})
	if err != nil {
		t.Fatalf("Cleanup: %v", err)
	}
	if len(report.Deleted) != 1 || len(report.PreservedByAge) != 1 {
		t.Errorf("dry-run filters incorrect: deleted=%d, preservedByAge=%d", len(report.Deleted), len(report.PreservedByAge))
	}
}
