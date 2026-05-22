// Tests for the orphan-cleanup CLI's audit-on-commit extension per
// decision-log §0119 (auth-scope RFC Open Question 4 discharge). The
// CLI defaults to NO substrate write (§0033 local-shell-trust); when
// --actor=<id> is supplied, the CLI mirrors the HTTP T3 audit-then-
// delete contract from §0104.
//
// The tests exercise the helper functions (sortedExclusionHashes,
// hashesOfDeletions) + the end-to-end commitAuditPair → substrate
// pairing against a real substrate.
package main

import (
	"context"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/orphan"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

func decodeHexHash(t *testing.T, s string) [32]byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("decode hex %q: %v", s, err)
	}
	if len(b) != 32 {
		t.Fatalf("hex %q decoded to %d bytes, want 32", s, len(b))
	}
	var out [32]byte
	copy(out[:], b)
	return out
}

func TestSortedExclusionHashesEmpty(t *testing.T) {
	if got := sortedExclusionHashes(nil); got != nil {
		t.Errorf("nil input: got %v, want nil", got)
	}
	if got := sortedExclusionHashes(map[string]bool{}); got != nil {
		t.Errorf("empty input: got %v, want nil", got)
	}
}

func TestSortedExclusionHashesAscending(t *testing.T) {
	in := map[string]bool{
		"ccccc": true,
		"aaaaa": true,
		"bbbbb": true,
	}
	got := sortedExclusionHashes(in)
	want := []string{"aaaaa", "bbbbb", "ccccc"}
	if len(got) != len(want) {
		t.Fatalf("len: got %d, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("index %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestHashesOfDeletions(t *testing.T) {
	in := []orphan.DeletionRecord{
		{Hash: "h1", Path: "/p1", Bytes: 10},
		{Hash: "h2", Path: "/p2", Bytes: 20},
	}
	got := hashesOfDeletions(in)
	if len(got) != 2 || got[0] != "h1" || got[1] != "h2" {
		t.Errorf("got %v, want [h1 h2]", got)
	}
	if hashesOfDeletions(nil) != nil {
		t.Error("nil input should return nil")
	}
}

// TestCommitAuditPairCommitsBothRows verifies commitAuditPair commits
// both the OrphanCleanupAudit + the paired IngestionEvent atomically
// via substrate.AppendPair. Mirrors the HTTP T3 admin.go contract.
func TestCommitAuditPairCommitsBothRows(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	invokedAt := time.Now().UnixNano()
	audit := &eventsv1.OrphanCleanupAudit{
		InvokedAt:             invokedAt,
		DryRun:                false,
		Confirm:               true,
		KeepNewerThanSeconds:  86400,
		MaxDeletions:          100,
		ExaminedCount:         5,
		OrphansFound:          1,
		PlannedDeletionHashes: []string{"deadbeef"},
	}
	hashes, err := commitAuditPair(ctx, sub, audit, "operator-bob", invokedAt)
	if err != nil {
		t.Fatalf("commitAuditPair: %v", err)
	}
	if hashes.audit == "" || hashes.ingestion == "" {
		t.Fatalf("hashes empty: audit=%q ingestion=%q", hashes.audit, hashes.ingestion)
	}

	// Verify the audit row is in the substrate via LookupRow.
	auditHashRaw := decodeHexHash(t, hashes.audit)
	row, err := sub.LookupRow(ctx, auditHashRaw)
	if err != nil {
		t.Fatalf("LookupRow audit: %v", err)
	}
	if want := "ghosttrace.events.v1.OrphanCleanupAudit"; row.MessageType != want {
		t.Errorf("audit MessageType: got %q, want %q", row.MessageType, want)
	}

	// Verify the IngestionEvent row carries channel="cli" + actor.
	ingHashRaw := decodeHexHash(t, hashes.ingestion)
	ingRow, err := sub.LookupRow(ctx, ingHashRaw)
	if err != nil {
		t.Fatalf("LookupRow ingestion: %v", err)
	}
	if want := "ghosttrace.events.v1.IngestionEvent"; ingRow.MessageType != want {
		t.Errorf("ingestion MessageType: got %q, want %q", ingRow.MessageType, want)
	}

	// Load the IngestionEvent payload + verify its fields.
	payload, err := os.ReadFile(filepath.Join(sub.BlobDir(), hashes.ingestion[:2], hashes.ingestion[2:]))
	if err != nil {
		t.Fatalf("read ingestion blob: %v", err)
	}
	var ing eventsv1.IngestionEvent
	if err := proto.Unmarshal(payload, &ing); err != nil {
		t.Fatalf("unmarshal IngestionEvent: %v", err)
	}
	if ing.Channel != "cli" {
		t.Errorf("Channel: got %q, want %q", ing.Channel, "cli")
	}
	if ing.ClientCommonName != "operator-bob" {
		t.Errorf("ClientCommonName: got %q, want %q", ing.ClientCommonName, "operator-bob")
	}
	if string(ing.PrimaryEventHash) != string(auditHashRaw[:]) {
		t.Errorf("PrimaryEventHash should reference the audit hash; got %x", ing.PrimaryEventHash)
	}
}
