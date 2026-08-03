package ingest

import (
	"context"
	"testing"
	"time"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

func openSubstrate(t *testing.T) *substrate.Substrate {
	t.Helper()
	dir := t.TempDir()
	sub, err := substrate.Open(context.Background(), dir+"/db.sqlite", dir+"/blobs")
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })
	return sub
}

func testBatch(seq uint32) *eventsv1.TelemetryBatch {
	return &eventsv1.TelemetryBatch{
		TenantId: "t_test", SessionId: "s_1", Seq: seq, SentAtMs: 2000,
	}
}

func TestAppendCommitsAndReports(t *testing.T) {
	sub := openSubstrate(t)
	fixed := time.Unix(1_754_000_000, 0)
	in := New(sub, func() time.Time { return fixed })

	msg := testBatch(1)
	rep, err := in.Append(context.Background(), msg, 42)
	if err != nil {
		t.Fatalf("Append: %v", err)
	}

	if rep.MessageType != "ghosttrace.events.v1.TelemetryBatch" {
		t.Errorf("MessageType = %q", rep.MessageType)
	}
	if len(rep.HashHex) != 64 {
		t.Errorf("HashHex length = %d, want 64", len(rep.HashHex))
	}
	if rep.HashHex != canonical.HashHex(rep.Hash) {
		t.Error("HashHex does not render Hash")
	}

	// The row is durable and carries the injected clock and event time.
	row, err := sub.LookupRow(context.Background(), rep.Hash)
	if err != nil {
		t.Fatalf("LookupRow: %v", err)
	}
	if row.EventTime != 42 {
		t.Errorf("EventTime = %d, want 42", row.EventTime)
	}
	if row.CommittedAt != fixed.UnixNano() {
		t.Errorf("CommittedAt = %d, want injected clock %d", row.CommittedAt, fixed.UnixNano())
	}
	if row.MessageType != rep.MessageType {
		t.Errorf("row MessageType = %q", row.MessageType)
	}

	// The blob is the canonical bytes: identity is content, and the
	// payload must round-trip through the hash that names it.
	payload, err := sub.ReadBlob(context.Background(), rep.Hash)
	if err != nil {
		t.Fatalf("ReadBlob: %v", err)
	}
	wantPayload, wantHash, err := canonical.MarshalAndHash(msg)
	if err != nil {
		t.Fatalf("MarshalAndHash: %v", err)
	}
	if string(payload) != string(wantPayload) {
		t.Error("stored blob differs from canonical bytes")
	}
	if wantHash != rep.Hash {
		t.Error("reported hash differs from canonical hash")
	}
}

func TestAppendIsIdempotent(t *testing.T) {
	// Retries are the normal case (§2): batches arrive out of order and
	// get retried after timeouts. Same content must be a no-op.
	sub := openSubstrate(t)
	in := New(sub, nil)

	first, err := in.Append(context.Background(), testBatch(1), 42)
	if err != nil {
		t.Fatalf("first Append: %v", err)
	}
	second, err := in.Append(context.Background(), testBatch(1), 42)
	if err != nil {
		t.Fatalf("retried Append: %v", err)
	}
	if first.Hash != second.Hash {
		t.Error("retried append returned a different hash")
	}

	n, err := sub.Count(context.Background())
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if n != 1 {
		t.Errorf("event count after retry = %d, want 1", n)
	}
}

func TestAppendDistinctContentDistinctIdentity(t *testing.T) {
	sub := openSubstrate(t)
	in := New(sub, nil)

	a, err := in.Append(context.Background(), testBatch(1), 42)
	if err != nil {
		t.Fatalf("Append a: %v", err)
	}
	b, err := in.Append(context.Background(), testBatch(2), 42)
	if err != nil {
		t.Fatalf("Append b: %v", err)
	}
	if a.Hash == b.Hash {
		t.Error("distinct content produced identical identity")
	}

	n, err := sub.Count(context.Background())
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if n != 2 {
		t.Errorf("event count = %d, want 2", n)
	}
}
