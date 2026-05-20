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

func testEnvelope() Envelope {
	return Envelope{Channel: "test"}
}

func TestAppendEndToEnd(t *testing.T) {
	ctx := context.Background()
	in, sub := newIngester(t)

	msg := &eventsv1.DeclaredSession{
		DeclaredAt:        1716120000000000000,
		ActorRef:          "actor-end-to-end",
		SessionDescriptor: []byte("session-bytes"),
	}

	rep, err := in.Append(ctx, msg, msg.DeclaredAt, testEnvelope())
	if err != nil {
		t.Fatalf("Append: %v", err)
	}
	if rep.EventHashHex == "" {
		t.Fatal("AppendReport.EventHashHex empty")
	}
	if len(rep.EventHashHex) != 64 {
		t.Fatalf("primary hex hash length: got %d, want 64", len(rep.EventHashHex))
	}
	if rep.IngestionEventHashHex == "" {
		t.Fatal("AppendReport.IngestionEventHashHex empty (pairing not surfaced)")
	}
	if len(rep.IngestionEventHashHex) != 64 {
		t.Fatalf("enrichment hex hash length: got %d, want 64", len(rep.IngestionEventHashHex))
	}
	if rep.EventHashHex == rep.IngestionEventHashHex {
		t.Errorf("primary and enrichment hashes should differ; both = %s", rep.EventHashHex)
	}
	if rep.PayloadBytes <= 0 {
		t.Fatalf("PayloadBytes: got %d, want >0", rep.PayloadBytes)
	}

	n, err := sub.Count(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("substrate Count: got %d, want 2 (one primary + one enrichment per Append)", n)
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

	rep1, err := in.Append(ctx, msg, msg.DeclaredAt, testEnvelope())
	if err != nil {
		t.Fatal(err)
	}
	rep2, err := in.Append(ctx, msg, msg.DeclaredAt, testEnvelope())
	if err != nil {
		t.Fatalf("re-Append failed: %v", err)
	}
	if rep1.EventHashHex != rep2.EventHashHex {
		t.Errorf("primary hash changed on re-Append: %s != %s", rep1.EventHashHex, rep2.EventHashHex)
	}
	if rep1.IngestionEventHashHex != rep2.IngestionEventHashHex {
		t.Errorf("enrichment hash changed on re-Append: %s != %s", rep1.IngestionEventHashHex, rep2.IngestionEventHashHex)
	}
	n, err := sub.Count(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("Count after idempotent re-Append: got %d, want 2 (one primary + one enrichment)", n)
	}
}

func TestAppendDistinctMessagesProduceDistinctHashes(t *testing.T) {
	ctx := context.Background()
	in, sub := newIngester(t)

	a := &eventsv1.DeclaredSession{DeclaredAt: 1, ActorRef: "actor-a", SessionDescriptor: []byte("a")}
	b := &eventsv1.DeclaredSession{DeclaredAt: 2, ActorRef: "actor-b", SessionDescriptor: []byte("b")}

	repA, err := in.Append(ctx, a, a.DeclaredAt, testEnvelope())
	if err != nil {
		t.Fatal(err)
	}
	repB, err := in.Append(ctx, b, b.DeclaredAt, testEnvelope())
	if err != nil {
		t.Fatal(err)
	}
	if repA.EventHashHex == repB.EventHashHex {
		t.Error("distinct messages produced identical primary hashes")
	}
	if repA.IngestionEventHashHex == repB.IngestionEventHashHex {
		t.Error("distinct primaries produced identical enrichment hashes (the IngestionEvent references the primary by hash, so should differ)")
	}
	n, err := sub.Count(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 4 {
		t.Errorf("Count: got %d, want 4 (two primaries + two enrichments)", n)
	}
}

func TestAppendDistinctEnvelopesProduceDistinctEnrichmentHashes(t *testing.T) {
	// Same primary observation under two distinct envelopes (e.g.
	// same message ingested over stdin then over https+mtls) produces
	// the SAME primary hash (idempotent on the primary) and DIFFERENT
	// enrichment hashes (distinct ingestion-act observations).
	ctx := context.Background()
	in, sub := newIngester(t)

	msg := &eventsv1.DeclaredSession{
		DeclaredAt:        1716120000000000000,
		ActorRef:          "actor-multi-channel",
		SessionDescriptor: []byte("session-bytes"),
	}

	repStdin, err := in.Append(ctx, msg, msg.DeclaredAt, Envelope{Channel: "stdin"})
	if err != nil {
		t.Fatal(err)
	}
	repHTTPS, err := in.Append(ctx, msg, msg.DeclaredAt, Envelope{
		Channel:          "https+mtls",
		ClientCommonName: "trusted-producer",
		ClientCertSHA256: "deadbeefcafef00d",
	})
	if err != nil {
		t.Fatal(err)
	}
	if repStdin.EventHashHex != repHTTPS.EventHashHex {
		t.Errorf("primary hash should be identical across channels: stdin=%s https=%s",
			repStdin.EventHashHex, repHTTPS.EventHashHex)
	}
	if repStdin.IngestionEventHashHex == repHTTPS.IngestionEventHashHex {
		t.Errorf("enrichment hashes should differ across channels: both=%s",
			repStdin.IngestionEventHashHex)
	}

	n, err := sub.Count(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Errorf("Count: got %d, want 3 (one primary + two enrichments)", n)
	}
}

func TestEnvelopeChannelDefaultsToUnspecified(t *testing.T) {
	// Empty channel coerces to "unspecified" at commit time so the
	// IngestionEvent always carries a non-empty channel marker.
	got := Envelope{}.channelOrUnspecified()
	if got != "unspecified" {
		t.Errorf("empty channel coercion: got %q, want %q", got, "unspecified")
	}
	got = Envelope{Channel: "explicit"}.channelOrUnspecified()
	if got != "explicit" {
		t.Errorf("explicit channel preservation: got %q, want %q", got, "explicit")
	}
}
