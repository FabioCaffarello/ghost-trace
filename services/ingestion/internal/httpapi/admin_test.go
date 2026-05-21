package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// substrateWithBlobs constructs a substrate, ingests count
// DeclaredSession primaries (with paired IngestionEvent), and returns
// the substrate. Both the SQL index and the blob-store are populated;
// none of the blobs are orphans.
func substrateWithBlobs(t *testing.T, count int) (*substrate.Substrate, string) {
	t.Helper()
	dir := t.TempDir()
	blobsDir := filepath.Join(dir, "blobs")
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), blobsDir)
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	in := ingest.New(sub, time.Now)
	for i := 0; i < count; i++ {
		msg := &eventsv1.DeclaredSession{
			DeclaredAt:        int64(1000 + i),
			ActorRef:          "actor-admin-test-" + string(rune('a'+i)),
			SessionDescriptor: []byte("session-bytes"),
		}
		if _, err := in.Append(ctx, msg, msg.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("Append %d: %v", i, err)
		}
	}
	return sub, blobsDir
}

// writeOrphanBlob writes a blob to the blob-store under the conventional
// two-char-fanout layout that orphan.Cleanup walks. Returns the blob's
// hex hash. The blob is NOT inserted into the SQL index — making it an
// orphan from the substrate's perspective.
func writeOrphanBlob(t *testing.T, blobsDir string, content []byte) string {
	t.Helper()
	hash := canonical.Hash(content)
	hex := canonical.HashHex(hash)
	subdir := filepath.Join(blobsDir, hex[:2])
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	path := filepath.Join(subdir, hex[2:])
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return hex
}

// realDoAppend constructs a doAppend that backs onto a real Ingester
// against sub. Used by T3 tests so the audit + IngestionEvent actually
// commit to the substrate (rather than the stubAppendFunc which returns
// canned hashes without writing).
func realDoAppend(t *testing.T, sub *substrate.Substrate) AppendFunc {
	t.Helper()
	in := ingest.New(sub, time.Now)
	return func(ctx context.Context, msg proto.Message, eventTime int64, env ingest.Envelope) (ingest.AppendReport, error) {
		return in.Append(ctx, msg, eventTime, env)
	}
}

func TestT3OrphanCleanupDryRunNoOrphans(t *testing.T) {
	sub, _ := substrateWithBlobs(t, 2)
	h := MustNew(realDoAppend(t, sub), nil, WithSubstrate(sub))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/orphan-cleanup", nil)
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
	var out orphanCleanupResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !out.DryRun {
		t.Errorf("DryRun: got false, want true (default)")
	}
	if out.OrphansFound != 0 {
		t.Errorf("OrphansFound: got %d, want 0 (no orphans seeded)", out.OrphansFound)
	}
	if out.AuditEventHash == "" {
		t.Errorf("AuditEventHash: empty; audit should commit even with zero orphans")
	}
	if out.IngestionEventHash == "" {
		t.Errorf("IngestionEventHash: empty; paired IngestionEvent should commit")
	}
}

func TestT3OrphanCleanupRejectsNonDryRunWithoutConfirm(t *testing.T) {
	sub, _ := substrateWithBlobs(t, 0)
	h := MustNew(realDoAppend(t, sub), nil, WithSubstrate(sub))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/orphan-cleanup?dry_run=false", nil)
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusBadRequest; got != want {
		t.Errorf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
}

func TestT3OrphanCleanupRejectsNonPost(t *testing.T) {
	sub, _ := substrateWithBlobs(t, 0)
	h := MustNew(realDoAppend(t, sub), nil, WithSubstrate(sub))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/orphan-cleanup", nil)
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusMethodNotAllowed; got != want {
		t.Errorf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
}

func TestT3OrphanCleanupReturns503WithoutSubstrate(t *testing.T) {
	// No WithSubstrate; substrate-required endpoint must return 503.
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/orphan-cleanup", nil)
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusServiceUnavailable; got != want {
		t.Errorf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
}

func TestT3OrphanCleanupDryRunIdentifiesOrphanWithoutDeleting(t *testing.T) {
	sub, blobsDir := substrateWithBlobs(t, 1)
	orphanHex := writeOrphanBlob(t, blobsDir, []byte("orphan-content"))
	h := MustNew(realDoAppend(t, sub), nil, WithSubstrate(sub))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/orphan-cleanup?dry_run=true", nil)
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
	var out orphanCleanupResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.OrphansFound != 1 {
		t.Errorf("OrphansFound: got %d, want 1", out.OrphansFound)
	}
	if out.Deleted != 1 {
		t.Errorf("Deleted (planned, under DryRun): got %d, want 1", out.Deleted)
	}
	if len(out.PlannedDeletionHashes) != 1 || out.PlannedDeletionHashes[0] != orphanHex {
		t.Errorf("PlannedDeletionHashes: got %v, want [%s]", out.PlannedDeletionHashes, orphanHex)
	}

	// Verify blob is still on disk (DryRun must not delete).
	orphanPath := filepath.Join(blobsDir, orphanHex[:2], orphanHex[2:])
	if _, err := os.Stat(orphanPath); err != nil {
		t.Errorf("orphan blob should remain on disk under DryRun: %v", err)
	}
}

func TestT3OrphanCleanupConfirmedDeletesOrphan(t *testing.T) {
	sub, blobsDir := substrateWithBlobs(t, 1)
	orphanHex := writeOrphanBlob(t, blobsDir, []byte("orphan-to-delete"))
	h := MustNew(realDoAppend(t, sub), nil, WithSubstrate(sub))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/orphan-cleanup?dry_run=false&confirm=true", nil)
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
	var out orphanCleanupResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.DryRun {
		t.Errorf("DryRun: got true, want false")
	}
	if !out.Confirm {
		t.Errorf("Confirm: got false, want true")
	}
	if out.Deleted != 1 {
		t.Errorf("Deleted: got %d, want 1", out.Deleted)
	}
	if out.AuditEventHash == "" {
		t.Errorf("AuditEventHash: empty; audit must commit before deletion")
	}

	// Verify blob has been removed.
	orphanPath := filepath.Join(blobsDir, orphanHex[:2], orphanHex[2:])
	if _, err := os.Stat(orphanPath); !os.IsNotExist(err) {
		t.Errorf("orphan blob should be deleted: stat err=%v", err)
	}
}

func TestT3OrphanCleanupRejectsBadParams(t *testing.T) {
	sub, _ := substrateWithBlobs(t, 0)
	h := MustNew(realDoAppend(t, sub), nil, WithSubstrate(sub))

	cases := []struct {
		name string
		url  string
	}{
		{"keep_newer_than_seconds non-int", "/v1/admin/orphan-cleanup?keep_newer_than_seconds=abc"},
		{"keep_newer_than_seconds negative", "/v1/admin/orphan-cleanup?keep_newer_than_seconds=-1"},
		{"max_deletions non-int", "/v1/admin/orphan-cleanup?max_deletions=xyz"},
		{"max_deletions negative", "/v1/admin/orphan-cleanup?max_deletions=-5"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, c.url, nil)
			h.ServeHTTP(rr, req)
			if got, want := rr.Code, http.StatusBadRequest; got != want {
				t.Errorf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
			}
		})
	}
}

func TestT3OrphanCleanupHonorsExcludedHashes(t *testing.T) {
	sub, blobsDir := substrateWithBlobs(t, 1)
	orphanHex := writeOrphanBlob(t, blobsDir, []byte("orphan-excluded"))
	h := MustNew(realDoAppend(t, sub), nil, WithSubstrate(sub))

	rr := httptest.NewRecorder()
	url := "/v1/admin/orphan-cleanup?dry_run=false&confirm=true&excluded_hash=" + orphanHex
	req := httptest.NewRequest(http.MethodPost, url, nil)
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
	var out orphanCleanupResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.PreservedByExclude != 1 {
		t.Errorf("PreservedByExclude: got %d, want 1", out.PreservedByExclude)
	}
	if out.Deleted != 0 {
		t.Errorf("Deleted: got %d, want 0 (excluded)", out.Deleted)
	}

	// Verify blob is still on disk (excluded from deletion).
	orphanPath := filepath.Join(blobsDir, orphanHex[:2], orphanHex[2:])
	if _, err := os.Stat(orphanPath); err != nil {
		t.Errorf("excluded orphan blob should remain on disk: %v", err)
	}
}

func TestT3MultiTierRequiresSubstrateAdminToken(t *testing.T) {
	// Multi-tier mode: T3 endpoint requires the substrate-admin tier
	// token. A request with only the producer token must 401.
	sub, _ := substrateWithBlobs(t, 0)
	h := MustNew(realDoAppend(t, sub), nil,
		WithSubstrate(sub),
		WithAuthTierToken(TierProducer, "prod-token"),
	)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/orphan-cleanup", nil)
	req.Header.Set("Authorization", "Bearer prod-token")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusUnauthorized; got != want {
		t.Errorf("status: got %d, want %d (T3 must require substrate-admin tier; body: %s)",
			got, want, rr.Body.String())
	}
}

func TestT3MultiTierSubstrateAdminTokenAuthorizes(t *testing.T) {
	sub, _ := substrateWithBlobs(t, 0)
	h := MustNew(realDoAppend(t, sub), nil,
		WithSubstrate(sub),
		WithAuthTierToken(TierSubstrateAdmin, "admin-token"),
	)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/orphan-cleanup", nil)
	req.Header.Set("Authorization", "Bearer admin-token")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Errorf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
}
