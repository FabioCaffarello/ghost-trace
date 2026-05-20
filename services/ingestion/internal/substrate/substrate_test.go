package substrate

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
)

func newTestSubstrate(t *testing.T) *Substrate {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	s, err := Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func newTestPayload(t *testing.T) ([]byte, [32]byte) {
	t.Helper()
	msg := &eventsv1.DeclaredSession{
		DeclaredAt:        1716120000000000000,
		ActorRef:          "test-actor",
		SessionDescriptor: []byte("session-descriptor-bytes"),
	}
	payload, hash, err := canonical.MarshalAndHash(msg)
	if err != nil {
		t.Fatalf("MarshalAndHash: %v", err)
	}
	return payload, hash
}

func TestOpenCloseRoundtrip(t *testing.T) {
	s := newTestSubstrate(t)
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	// Idempotent close should not error.
	if err := s.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestAppendAndLookup(t *testing.T) {
	ctx := context.Background()
	s := newTestSubstrate(t)
	payload, hash := newTestPayload(t)

	hex := canonical.HashHex(hash)
	row := EventRow{
		EventHash:   hash,
		EventTime:   1716120000000000000,
		MessageType: "ghosttrace.events.v1.DeclaredSession",
		PayloadRef:  hex[:2] + "/" + hex[2:],
		CommittedAt: 1716120000000000001,
	}

	if err := s.Append(ctx, row, payload); err != nil {
		t.Fatalf("Append: %v", err)
	}

	got, err := s.LookupRow(ctx, hash)
	if err != nil {
		t.Fatalf("LookupRow: %v", err)
	}
	if got.EventHash != hash {
		t.Errorf("EventHash mismatch")
	}
	if got.EventTime != row.EventTime {
		t.Errorf("EventTime: got %d, want %d", got.EventTime, row.EventTime)
	}
	if got.MessageType != row.MessageType {
		t.Errorf("MessageType: got %q, want %q", got.MessageType, row.MessageType)
	}
}

func TestAppendIdempotent(t *testing.T) {
	ctx := context.Background()
	s := newTestSubstrate(t)
	payload, hash := newTestPayload(t)

	hex := canonical.HashHex(hash)
	row := EventRow{
		EventHash:   hash,
		EventTime:   1,
		MessageType: "ghosttrace.events.v1.DeclaredSession",
		PayloadRef:  hex[:2] + "/" + hex[2:],
		CommittedAt: 2,
	}

	for i := 0; i < 3; i++ {
		if err := s.Append(ctx, row, payload); err != nil {
			t.Fatalf("Append iteration %d: %v", i, err)
		}
	}

	n, err := s.Count(ctx)
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if n != 1 {
		t.Errorf("Count after 3 idempotent appends: got %d, want 1", n)
	}
}

func TestAppendHashMismatchRejected(t *testing.T) {
	ctx := context.Background()
	s := newTestSubstrate(t)
	payload, hash := newTestPayload(t)

	var wrong [32]byte
	copy(wrong[:], hash[:])
	wrong[0] ^= 0xff // flip a bit so hash no longer matches payload

	row := EventRow{EventHash: wrong, EventTime: 1, MessageType: "x", PayloadRef: "x/y", CommittedAt: 1}
	err := s.Append(ctx, row, payload)
	if err == nil {
		t.Fatal("expected ErrHashMismatch, got nil")
	}
	if !errors.Is(err, ErrHashMismatch) {
		t.Fatalf("expected ErrHashMismatch, got: %v", err)
	}
}

func TestReadBlobAfterAppend(t *testing.T) {
	ctx := context.Background()
	s := newTestSubstrate(t)
	payload, hash := newTestPayload(t)

	hex := canonical.HashHex(hash)
	row := EventRow{EventHash: hash, EventTime: 1, MessageType: "x", PayloadRef: hex[:2] + "/" + hex[2:], CommittedAt: 1}
	if err := s.Append(ctx, row, payload); err != nil {
		t.Fatalf("Append: %v", err)
	}

	got, err := s.ReadBlob(ctx, hash)
	if err != nil {
		t.Fatalf("ReadBlob: %v", err)
	}
	if string(got) != string(payload) {
		t.Errorf("blob bytes differ from payload")
	}
}

func TestReadBlobHashMismatchOnCorruption(t *testing.T) {
	ctx := context.Background()
	s := newTestSubstrate(t)
	payload, hash := newTestPayload(t)

	hex := canonical.HashHex(hash)
	row := EventRow{EventHash: hash, EventTime: 1, MessageType: "x", PayloadRef: hex[:2] + "/" + hex[2:], CommittedAt: 1}
	if err := s.Append(ctx, row, payload); err != nil {
		t.Fatalf("Append: %v", err)
	}

	// Deliberately corrupt the on-disk blob to exercise §2.1 violation
	// detection on the read path per decision-log §0027 AP5.
	_, finalPath := s.blobPath(hash)
	if err := os.WriteFile(finalPath, []byte("corrupted-content"), 0o644); err != nil {
		t.Fatalf("corrupt blob: %v", err)
	}

	_, err := s.ReadBlob(ctx, hash)
	if err == nil {
		t.Fatal("expected ErrHashMismatch after corruption, got nil")
	}
	if !errors.Is(err, ErrHashMismatch) {
		t.Fatalf("expected ErrHashMismatch, got: %v", err)
	}
}

func TestLookupRowAbsent(t *testing.T) {
	ctx := context.Background()
	s := newTestSubstrate(t)

	var absent [32]byte
	for i := range absent {
		absent[i] = 0xab
	}
	_, err := s.LookupRow(ctx, absent)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows, got: %v", err)
	}
}
