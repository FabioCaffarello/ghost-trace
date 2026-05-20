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

// TestAppendHeterogeneousCatITypes proves the substrate ingests
// distinct Cat I message types in the same events table with the
// message_type discriminator distinguishing them — the structural
// proof that the ingestion service is a substrate rather than a
// DeclaredSession-special-case (per decision-log §0042).
func TestAppendHeterogeneousCatITypes(t *testing.T) {
	ctx := context.Background()
	in, sub := newIngester(t)

	decl := &eventsv1.DeclaredSession{
		DeclaredAt:        1716120000000000000,
		ActorRef:          "actor-hetero-test",
		SessionDescriptor: []byte("session"),
	}
	if _, err := in.Append(ctx, decl, decl.DeclaredAt, testEnvelope()); err != nil {
		t.Fatalf("Append DeclaredSession: %v", err)
	}

	netEvt := &eventsv1.NetworkEvent{
		ObservedAt:      1716120000000000999,
		ActorRef:        "actor-hetero-test",
		EndpointRef:     "10.0.0.1:443",
		EventDescriptor: []byte("flow"),
	}
	if _, err := in.Append(ctx, netEvt, netEvt.ObservedAt, testEnvelope()); err != nil {
		t.Fatalf("Append NetworkEvent: %v", err)
	}

	n, err := sub.Count(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 4 {
		t.Errorf("substrate Count: got %d, want 4 (2 primary + 2 enrichment)", n)
	}

	typeCounts := map[string]int{}
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		typeCounts[row.MessageType]++
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	if got := typeCounts["ghosttrace.events.v1.DeclaredSession"]; got != 1 {
		t.Errorf("DeclaredSession row count: got %d, want 1", got)
	}
	if got := typeCounts["ghosttrace.events.v1.NetworkEvent"]; got != 1 {
		t.Errorf("NetworkEvent row count: got %d, want 1", got)
	}
	if got := typeCounts["ghosttrace.events.v1.IngestionEvent"]; got != 2 {
		t.Errorf("IngestionEvent row count: got %d, want 2 (one per primary)", got)
	}
}

// TestRegistryLookupRoundTrip is a sanity check that every registered
// descriptor is retrievable via both lookup helpers.
func TestRegistryLookupRoundTrip(t *testing.T) {
	if len(Registry) < 2 {
		t.Fatalf("Registry holds fewer than 2 descriptors (got %d); the second-Cat-I-type invariant (decision-log §0042) is not satisfied", len(Registry))
	}
	for _, want := range Registry {
		gotByURL, ok := LookupURLPath(want.URLPath)
		if !ok {
			t.Errorf("LookupURLPath(%q) missing", want.URLPath)
			continue
		}
		if gotByURL.StdinType != want.StdinType {
			t.Errorf("LookupURLPath(%q).StdinType: got %q, want %q", want.URLPath, gotByURL.StdinType, want.StdinType)
		}
		gotByStdin, ok := LookupStdinType(want.StdinType)
		if !ok {
			t.Errorf("LookupStdinType(%q) missing", want.StdinType)
			continue
		}
		if gotByStdin.URLPath != want.URLPath {
			t.Errorf("LookupStdinType(%q).URLPath: got %q, want %q", want.StdinType, gotByStdin.URLPath, want.URLPath)
		}
		// New() must return a non-nil zero message.
		if msg := want.New(); msg == nil {
			t.Errorf("descriptor %q New() returned nil", want.URLPath)
		}
	}

	if _, ok := LookupURLPath("nonexistent-type"); ok {
		t.Errorf("LookupURLPath returned ok=true for unregistered type")
	}
	if _, ok := LookupStdinType("nonexistent_type"); ok {
		t.Errorf("LookupStdinType returned ok=true for unregistered type")
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
