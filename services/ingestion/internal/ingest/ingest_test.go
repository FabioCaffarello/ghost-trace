package ingest

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

func newIngester(t *testing.T) (*Ingester, *substrate.Substrate) {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })
	clock := func() time.Time { return time.Unix(0, 1716120000000000777) }
	return New(sub, clock), sub
}

func TestAppendEndToEnd(t *testing.T) {
	ctx := context.Background()
	in, sub := newIngester(t)

	msg := &eventsv1.DeclaredSession{
		DeclaredAt:        1716120000000000000,
		ActorRef:          "actor-end-to-end",
		SessionDescriptor: []byte("session-bytes"),
	}

	rep, err := in.Append(ctx, msg, msg.DeclaredAt)
	if err != nil {
		t.Fatalf("Append: %v", err)
	}
	if rep.EventHashHex == "" {
		t.Fatal("AppendReport.EventHashHex empty")
	}
	if len(rep.EventHashHex) != 64 {
		t.Fatalf("hex hash length: got %d, want 64", len(rep.EventHashHex))
	}
	if rep.PayloadBytes <= 0 {
		t.Fatalf("PayloadBytes: got %d, want >0", rep.PayloadBytes)
	}

	n, err := sub.Count(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("substrate Count: got %d, want 1", n)
	}
}

func TestAppendIdempotent(t *testing.T) {
	ctx := context.Background()
	in, sub := newIngester(t)

	msg := &eventsv1.DeclaredSession{
		DeclaredAt:        1716120000000000000,
		ActorRef:          "actor-idem",
		SessionDescriptor: []byte("session-bytes"),
	}

	rep1, err := in.Append(ctx, msg, msg.DeclaredAt)
	if err != nil {
		t.Fatal(err)
	}
	rep2, err := in.Append(ctx, msg, msg.DeclaredAt)
	if err != nil {
		t.Fatalf("re-Append failed: %v", err)
	}
	if rep1.EventHashHex != rep2.EventHashHex {
		t.Errorf("hash changed on re-Append: %s != %s", rep1.EventHashHex, rep2.EventHashHex)
	}
	n, err := sub.Count(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("Count after idempotent re-Append: got %d, want 1", n)
	}
}

func TestAppendDistinctMessagesProduceDistinctHashes(t *testing.T) {
	ctx := context.Background()
	in, sub := newIngester(t)

	a := &eventsv1.DeclaredSession{DeclaredAt: 1, ActorRef: "actor-a", SessionDescriptor: []byte("a")}
	b := &eventsv1.DeclaredSession{DeclaredAt: 2, ActorRef: "actor-b", SessionDescriptor: []byte("b")}

	repA, err := in.Append(ctx, a, a.DeclaredAt)
	if err != nil {
		t.Fatal(err)
	}
	repB, err := in.Append(ctx, b, b.DeclaredAt)
	if err != nil {
		t.Fatal(err)
	}
	if repA.EventHashHex == repB.EventHashHex {
		t.Error("distinct messages produced identical hashes")
	}
	n, err := sub.Count(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("Count: got %d, want 2", n)
	}
}
