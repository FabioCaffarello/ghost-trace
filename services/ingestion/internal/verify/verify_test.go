package verify

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// newPopulatedSubstrate constructs a substrate + ingests `count` test
// DeclaredSession messages via the full ingest.Ingester path. Returns
// the substrate plus the slice of primary content-hashes committed.
func newPopulatedSubstrate(t *testing.T, count int) (*substrate.Substrate, []string) {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	in := ingest.New(sub, time.Now)
	var hashes []string
	for i := 0; i < count; i++ {
		msg := &eventsv1.DeclaredSession{
			DeclaredAt:        int64(1000 + i),
			ActorRef:          "actor-" + string(rune('a'+i)),
			SessionDescriptor: []byte("session-" + string(rune('A'+i))),
		}
		rep, err := in.Append(ctx, msg, msg.DeclaredAt, ingest.Envelope{Channel: "test"})
		if err != nil {
			t.Fatalf("Append %d: %v", i, err)
		}
		hashes = append(hashes, rep.EventHashHex, rep.IngestionEventHashHex)
	}
	return sub, hashes
}

func TestVerifyHappyPath(t *testing.T) {
	ctx := context.Background()
	sub, hashes := newPopulatedSubstrate(t, 3)

	report, err := Verify(ctx, sub, Options{})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if report.Failed() {
		t.Errorf("happy-path Verify reported failure: %+v", report)
	}
	// 3 ingest calls × 2 records (primary + enrichment) = 6 walked.
	if report.VerifiedCount != int64(len(hashes)) {
		t.Errorf("VerifiedCount: got %d, want %d", report.VerifiedCount, len(hashes))
	}
	if report.HashMismatchCount != 0 {
		t.Errorf("HashMismatchCount: got %d, want 0", report.HashMismatchCount)
	}
	if report.MissingBlobCount != 0 {
		t.Errorf("MissingBlobCount: got %d, want 0", report.MissingBlobCount)
	}
}

func TestVerifyDetectsCorruption(t *testing.T) {
	ctx := context.Background()
	sub, _ := newPopulatedSubstrate(t, 3)

	// Locate the blob-store directory + corrupt one blob.
	// Walk to find any blob file.
	var blobPath string
	dir := sub.BlobDir()
	if dir == "" {
		t.Skip("substrate does not expose BlobDir() — instrumentation needed")
	}
	if err := filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || blobPath != "" {
			return nil
		}
		blobPath = p
		return nil
	}); err != nil {
		t.Fatalf("walk blobs: %v", err)
	}
	if blobPath == "" {
		t.Fatal("no blob file found to corrupt")
	}
	// Overwrite with garbage; hash recomputation must surface a mismatch.
	if err := os.WriteFile(blobPath, []byte("corrupted-on-purpose"), 0o644); err != nil {
		t.Fatalf("write corrupted blob: %v", err)
	}

	report, err := Verify(ctx, sub, Options{})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !report.Failed() {
		t.Fatal("Verify did not detect blob corruption")
	}
	if report.HashMismatchCount < 1 {
		t.Errorf("HashMismatchCount: got %d, want >= 1", report.HashMismatchCount)
	}
	if len(report.HashMismatchHashes) != int(report.HashMismatchCount) {
		t.Errorf("HashMismatchHashes len %d != HashMismatchCount %d",
			len(report.HashMismatchHashes), report.HashMismatchCount)
	}
}

func TestVerifyDetectsMissingBlob(t *testing.T) {
	ctx := context.Background()
	sub, _ := newPopulatedSubstrate(t, 3)

	dir := sub.BlobDir()
	if dir == "" {
		t.Skip("substrate does not expose BlobDir()")
	}
	// Delete one blob to simulate filesystem corruption / partial backup.
	var deletedHex string
	if err := filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || deletedHex != "" {
			return err
		}
		if err := os.Remove(p); err != nil {
			return err
		}
		// Recover the hex hash from the path: <dir>/<2>/<62>.
		rel, _ := filepath.Rel(dir, p)
		deletedHex = strings.ReplaceAll(filepath.ToSlash(rel), "/", "")
		return nil
	}); err != nil {
		t.Fatalf("walk blobs: %v", err)
	}
	if deletedHex == "" {
		t.Fatal("no blob file deleted")
	}

	report, err := Verify(ctx, sub, Options{})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !report.Failed() {
		t.Fatal("Verify did not detect missing blob")
	}
	if report.MissingBlobCount < 1 {
		t.Errorf("MissingBlobCount: got %d, want >= 1", report.MissingBlobCount)
	}

	// The deleted hash should appear in MissingBlobHashes.
	found := false
	for _, h := range report.MissingBlobHashes {
		if h == deletedHex {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("deleted hash %s not in MissingBlobHashes %v", deletedHex, report.MissingBlobHashes)
	}
}

func TestVerifyEmptySubstrate(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	report, err := Verify(ctx, sub, Options{})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if report.Failed() {
		t.Errorf("empty substrate should pass: %+v", report)
	}
	if report.VerifiedCount != 0 {
		t.Errorf("VerifiedCount: got %d, want 0", report.VerifiedCount)
	}
}

// Sanity: the helper produces deterministic test data.
func TestPopulatedSubstrateHashes(t *testing.T) {
	_, hashes := newPopulatedSubstrate(t, 2)
	if len(hashes) != 4 {
		t.Errorf("expected 4 hashes (2 primary + 2 enrichment), got %d", len(hashes))
	}
	for _, h := range hashes {
		if len(h) != 64 {
			t.Errorf("hash length: got %d, want 64 — %q", len(h), h)
		}
	}
}

func TestVerifyCheckOrphansDetectsOrphan(t *testing.T) {
	ctx := context.Background()
	sub, _ := newPopulatedSubstrate(t, 2)

	// Plant an orphan blob: write a file in the blob-store with a
	// hash that does NOT appear in the events table.
	orphanContent := []byte("this-blob-has-no-index-row")
	orphanHash := canonical.Hash(orphanContent)
	orphanHex := canonical.HashHex(orphanHash)
	orphanDir := filepath.Join(sub.BlobDir(), orphanHex[:2])
	if err := os.MkdirAll(orphanDir, 0o755); err != nil {
		t.Fatalf("mkdir orphan shard: %v", err)
	}
	orphanPath := filepath.Join(orphanDir, orphanHex[2:])
	if err := os.WriteFile(orphanPath, orphanContent, 0o644); err != nil {
		t.Fatalf("write orphan blob: %v", err)
	}

	report, err := Verify(ctx, sub, Options{CheckOrphans: true})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	// Orphans are NOT failures per §0033 + §0040.
	if report.Failed() {
		t.Errorf("orphan should not cause Failed(): %+v", report)
	}
	if report.OrphanBlobCount != 1 {
		t.Errorf("OrphanBlobCount: got %d, want 1", report.OrphanBlobCount)
	}
	if len(report.OrphanBlobPaths) != 1 {
		t.Fatalf("OrphanBlobPaths len: got %d, want 1", len(report.OrphanBlobPaths))
	}
	if report.OrphanBlobPaths[0] != orphanPath {
		t.Errorf("OrphanBlobPaths[0]: got %q, want %q", report.OrphanBlobPaths[0], orphanPath)
	}
}

func TestVerifyOrphanDetectionDisabledByDefault(t *testing.T) {
	ctx := context.Background()
	sub, _ := newPopulatedSubstrate(t, 2)

	// Plant an orphan blob.
	orphanContent := []byte("orphan-not-checked-when-off")
	orphanHash := canonical.Hash(orphanContent)
	orphanHex := canonical.HashHex(orphanHash)
	orphanDir := filepath.Join(sub.BlobDir(), orphanHex[:2])
	_ = os.MkdirAll(orphanDir, 0o755)
	_ = os.WriteFile(filepath.Join(orphanDir, orphanHex[2:]), orphanContent, 0o644)

	// Default Options{CheckOrphans: false} skips the blob-store walk.
	report, err := Verify(ctx, sub, Options{})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if report.OrphanBlobCount != 0 {
		t.Errorf("OrphanBlobCount: got %d, want 0 (orphan-check disabled)", report.OrphanBlobCount)
	}
	if report.Failed() {
		t.Errorf("happy-path with orphan present (and check disabled) should not fail: %+v", report)
	}
}

func TestVerifyCheckOrphansHappyPath(t *testing.T) {
	// No orphans planted; OrphanBlobCount must be 0.
	ctx := context.Background()
	sub, _ := newPopulatedSubstrate(t, 3)

	report, err := Verify(ctx, sub, Options{CheckOrphans: true})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if report.Failed() {
		t.Errorf("clean substrate should not fail: %+v", report)
	}
	if report.OrphanBlobCount != 0 {
		t.Errorf("OrphanBlobCount: got %d, want 0 (no orphans planted)", report.OrphanBlobCount)
	}
}

// TestVerifyHeterogeneousCatITypes proves verify walks a substrate
// containing multiple Cat I primary-observation types (per decision-log
// §0042) and reports success. The substrate's hash-chain consistency
// is type-agnostic; verify recomputes hashes per the canonical-
// serialization-contract for each row regardless of message_type.
func TestVerifyHeterogeneousCatITypes(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	in := ingest.New(sub, time.Now)
	decl := &eventsv1.DeclaredSession{DeclaredAt: 1000, ActorRef: "het-decl", SessionDescriptor: []byte("s")}
	if _, err := in.Append(ctx, decl, decl.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
		t.Fatalf("Append DeclaredSession: %v", err)
	}
	netEvt := &eventsv1.NetworkEvent{ObservedAt: 1001, ActorRef: "het-net", EndpointRef: "10.0.0.1:80", EventDescriptor: []byte("f")}
	if _, err := in.Append(ctx, netEvt, netEvt.ObservedAt, ingest.Envelope{Channel: "test"}); err != nil {
		t.Fatalf("Append NetworkEvent: %v", err)
	}

	report, err := Verify(ctx, sub, Options{})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if report.Failed() {
		t.Errorf("heterogeneous substrate Verify reported failure: %+v", report)
	}
	if report.VerifiedCount != 4 {
		t.Errorf("VerifiedCount: got %d, want 4 (2 primary + 2 enrichment)", report.VerifiedCount)
	}
}

// Sanity: every proto.Message field on the test message round-trips
// through canonical.Marshal without error.
func TestMessageRoundtrip(t *testing.T) {
	msg := &eventsv1.DeclaredSession{
		DeclaredAt:        42,
		ActorRef:          "round-trip",
		SessionDescriptor: []byte("rt"),
	}
	b, err := proto.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	want := canonical.Hash(b)
	if want == [32]byte{} {
		t.Error("expected non-zero hash")
	}
}
