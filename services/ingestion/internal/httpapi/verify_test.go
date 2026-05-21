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

	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// substrateWithDeclaredSessionsHTTPVerify constructs a substrate +
// ingests `count` DeclaredSession primary observations via the full
// ingest.Ingester path. Returns the substrate.
func substrateWithDeclaredSessionsHTTPVerify(t *testing.T, count int) *substrate.Substrate {
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
			ActorRef:          "actor-verify-http-" + string(rune('a'+i)),
			SessionDescriptor: []byte("session-verify-http"),
		}
		if _, err := in.Append(ctx, msg, msg.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("Append %d: %v", i, err)
		}
	}
	return sub
}

func TestVerifyHTTPHappyPath(t *testing.T) {
	sub := substrateWithDeclaredSessionsHTTPVerify(t, 3)
	doAppend, _ := stubAppendFunc(nil)
	h := New(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet, "/v1/verify", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
	var out verifyPayload
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !out.Passed {
		t.Errorf("passed: got false, want true; body=%s", rr.Body.String())
	}
	// 3 DeclaredSession + 3 paired IngestionEvent = 6 substrate rows.
	if out.Verified != 6 {
		t.Errorf("verified: got %d, want 6", out.Verified)
	}
	if out.HashMismatch != 0 || out.MissingBlob != 0 {
		t.Errorf("violations on clean substrate: hash_mismatch=%d missing_blob=%d",
			out.HashMismatch, out.MissingBlob)
	}
	if out.OrphanBlob != 0 {
		t.Errorf("orphan_blob: got %d, want 0 (check_orphans not requested)", out.OrphanBlob)
	}
}

func TestVerifyHTTPEmptySubstrate(t *testing.T) {
	sub := substrateWithDeclaredSessionsHTTPVerify(t, 0)
	doAppend, _ := stubAppendFunc(nil)
	h := New(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet, "/v1/verify", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rr.Code)
	}
	var out verifyPayload
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	if out.Verified != 0 {
		t.Errorf("verified: got %d, want 0", out.Verified)
	}
	if !out.Passed {
		t.Errorf("passed: got false, want true (empty substrate is clean by definition)")
	}
}

func TestVerifyHTTPCheckOrphansClean(t *testing.T) {
	sub := substrateWithDeclaredSessionsHTTPVerify(t, 2)
	doAppend, _ := stubAppendFunc(nil)
	h := New(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet, "/v1/verify?check_orphans=true", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var out verifyPayload
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	if !out.Passed {
		t.Errorf("passed: got false, want true")
	}
	if out.OrphanBlob != 0 {
		t.Errorf("orphan_blob: got %d, want 0 (no orphans created)", out.OrphanBlob)
	}
}

func TestVerifyHTTPCheckOrphansDetectsOrphan(t *testing.T) {
	sub := substrateWithDeclaredSessionsHTTPVerify(t, 1)
	// Plant an orphan blob: a file in the blob-store directory whose
	// content-hash does not appear in the events table. The verify
	// blob-walk discovers it under -check-orphans.
	blobDir := sub.BlobDir()
	if blobDir == "" {
		t.Skip("substrate does not expose BlobDir()")
	}
	// Use a hex hash unrelated to any committed blob; the path layout
	// is <dir>/<first-2-hex>/<rest-62-hex> per substrate convention.
	orphanHex := "deadbeef" + "00000000000000000000000000000000000000000000000000000000"
	subdir := filepath.Join(blobDir, orphanHex[:2])
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("mkdir orphan subdir: %v", err)
	}
	orphanPath := filepath.Join(subdir, orphanHex[2:])
	if err := os.WriteFile(orphanPath, []byte("orphan-content"), 0o644); err != nil {
		t.Fatalf("write orphan blob: %v", err)
	}

	doAppend, _ := stubAppendFunc(nil)
	h := New(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet, "/v1/verify?check_orphans=true", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var out verifyPayload
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	if !out.Passed {
		t.Errorf("passed: got false, want true (orphans are NOT a §2.1 violation)")
	}
	if out.OrphanBlob < 1 {
		t.Errorf("orphan_blob: got %d, want >= 1", out.OrphanBlob)
	}
	if len(out.OrphanBlobPaths) < 1 {
		t.Errorf("orphan_blob_paths: got %v, want non-empty", out.OrphanBlobPaths)
	}
}

func TestVerifyHTTPDetectsCorruption(t *testing.T) {
	sub := substrateWithDeclaredSessionsHTTPVerify(t, 2)
	// Corrupt one blob in place; verify must surface a hash-mismatch.
	blobDir := sub.BlobDir()
	if blobDir == "" {
		t.Skip("substrate does not expose BlobDir()")
	}
	var blobPath string
	if err := filepath.Walk(blobDir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || blobPath != "" {
			return err
		}
		blobPath = p
		return nil
	}); err != nil {
		t.Fatalf("walk blobs: %v", err)
	}
	if blobPath == "" {
		t.Fatal("no blob to corrupt")
	}
	if err := os.WriteFile(blobPath, []byte("corrupted-on-purpose"), 0o644); err != nil {
		t.Fatalf("corrupt blob: %v", err)
	}

	doAppend, _ := stubAppendFunc(nil)
	h := New(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet, "/v1/verify", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200 (substrate-integrity failure is a 200 + passed=false, not an HTTP error)", rr.Code)
	}
	var out verifyPayload
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	if out.Passed {
		t.Error("passed: got true, want false (corrupted blob should fail verification)")
	}
	if out.HashMismatch < 1 {
		t.Errorf("hash_mismatch: got %d, want >= 1", out.HashMismatch)
	}
	if len(out.HashMismatchHashes) != int(out.HashMismatch) {
		t.Errorf("hash_mismatch_hashes len %d != hash_mismatch %d",
			len(out.HashMismatchHashes), out.HashMismatch)
	}
}

func TestVerifyHTTPDetectsMissingBlob(t *testing.T) {
	sub := substrateWithDeclaredSessionsHTTPVerify(t, 2)
	blobDir := sub.BlobDir()
	if blobDir == "" {
		t.Skip("substrate does not expose BlobDir()")
	}
	// Delete one blob to simulate filesystem corruption.
	var deleted bool
	if err := filepath.Walk(blobDir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || deleted {
			return err
		}
		if err := os.Remove(p); err != nil {
			return err
		}
		deleted = true
		return nil
	}); err != nil {
		t.Fatalf("walk blobs: %v", err)
	}
	if !deleted {
		t.Fatal("no blob deleted")
	}

	doAppend, _ := stubAppendFunc(nil)
	h := New(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet, "/v1/verify", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rr.Code)
	}
	var out verifyPayload
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	if out.Passed {
		t.Error("passed: got true, want false (missing blob should fail verification)")
	}
	if out.MissingBlob < 1 {
		t.Errorf("missing_blob: got %d, want >= 1", out.MissingBlob)
	}
}

func TestVerifyHTTPInvalidCheckOrphans(t *testing.T) {
	sub := substrateWithDeclaredSessionsHTTPVerify(t, 1)
	doAppend, _ := stubAppendFunc(nil)
	h := New(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet, "/v1/verify?check_orphans=garbage", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", rr.Code)
	}
}

func TestVerifyHTTPNonGetRejected(t *testing.T) {
	sub := substrateWithDeclaredSessionsHTTPVerify(t, 1)
	doAppend, _ := stubAppendFunc(nil)
	h := New(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodPost, "/v1/verify", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST: status got %d, want 405", rr.Code)
	}
}

func TestVerifyHTTPSubstrateNotConfigured(t *testing.T) {
	// Construct handler WITHOUT WithSubstrate. The endpoint must return
	// 503 — the route remains registered but is structurally
	// unavailable on this handler.
	doAppend, _ := stubAppendFunc(nil)
	h := New(doAppend, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/verify", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("status: got %d, want 503", rr.Code)
	}
}

func TestVerifyHTTPAuthRequired(t *testing.T) {
	sub := substrateWithDeclaredSessionsHTTPVerify(t, 1)
	doAppend, _ := stubAppendFunc(nil)
	h := New(doAppend, nil, WithSubstrate(sub), WithAuthToken("secret"))

	req := httptest.NewRequest(http.MethodGet, "/v1/verify", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("no token: status got %d, want 401", rr.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/v1/verify", nil)
	req2.Header.Set("Authorization", "Bearer secret")
	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Errorf("correct token: status got %d, want 200", rr2.Code)
	}
}
