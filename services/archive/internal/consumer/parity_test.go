// The parity test: what went onto the stream, the archive holds — byte
// for byte, by content hash.
//
// It was written as the evidence for removing the collector's local
// write, and that removal happened (ADR-0006). What it asserts now
// matters MORE rather than less: the stream is the only path a record
// takes to durable storage, so a record that reaches the stream and not
// the substrate is simply lost.
//
// The reference store below is a stand-in for "what was published", not
// a second production store — there is no local write left to mirror.
//
// It needs a real broker. GT_NATS_URL points at one; without it the
// test SKIPS RATHER THAN PASSES, because a parity test that quietly
// does nothing is exactly the vacuous green this repository keeps
// finding.
package consumer_test

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/FabioCaffarello/ghost-trace/libs/canonical"
	"github.com/FabioCaffarello/ghost-trace/libs/eventstream"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/libs/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/libs/substrate"
	"google.golang.org/protobuf/proto"

	"github.com/FabioCaffarello/ghost-trace/services/archive/internal/consumer"
)

func natsURL(t *testing.T) string {
	t.Helper()
	url := os.Getenv("GT_NATS_URL")
	if url == "" {
		t.Skip("GT_NATS_URL not set — start a broker to run the parity test " +
			"(docker run -p 4222:4222 nats:alpine -js)")
	}
	return url
}

func openStore(t *testing.T, name string) *substrate.Substrate {
	t.Helper()
	dir := t.TempDir()
	s, err := substrate.Open(context.Background(),
		filepath.Join(dir, name+".db"), filepath.Join(dir, name+"-blobs"))
	if err != nil {
		t.Fatalf("open %s substrate: %v", name, err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// records builds a spread across every archived type, so parity is
// asserted over the whole subject space rather than one convenient
// message.
//
// Contents are UNIQUE PER RUN. The broker deduplicates by message id,
// which is the record's content hash, over a five-minute window — so
// fixed contents make a second run within that window publish nothing
// and the test then fails claiming the archive received nothing. That
// happened, and the spurious failure masked the assertion this test
// exists for.
func records(t *testing.T, run string) []proto.Message {
	t.Helper()
	return []proto.Message{
		&eventsv1.SessionStart{SessionId: "s_" + run, TenantId: "t_parity"},
		&eventsv1.TelemetryBatch{SessionId: "s_" + run, Seq: 1},
		&eventsv1.Evaluation{EvaluationId: "ev_" + run, SessionId: "s_" + run},
		&eventsv1.Outcome{EvaluationId: "ev_" + run, Outcome: "login_success"},
	}
}

func TestArchiveHoldsEverythingTheCollectorWrote(t *testing.T) {
	url := natsURL(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	nc, js, err := eventstream.Connect(url, "parity-test")
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer nc.Close()
	if err := eventstream.EnsureStream(ctx, js); err != nil {
		t.Fatalf("ensure stream: %v", err)
	}

	run := strconv.FormatInt(time.Now().UnixNano(), 36)

	local := openStore(t, "collector")
	remote := openStore(t, "archive")

	// The archive side, running as it does in production.
	cons := consumer.New(remote, time.Now, nil)
	consumeCtx, stopConsume := context.WithCancel(ctx)
	consumed := make(chan error, 1)
	go func() { consumed <- eventstream.Consume(consumeCtx, js, cons.Handle) }()

	// The producer side: record what is published, and publish it. The
	// reference store is the test's own bookkeeping.
	pub := eventstream.NewPublisher(js)
	want := map[string]string{}
	for i, msg := range records(t, run) {
		payload, hash, err := canonical.MarshalAndHash(msg)
		if err != nil {
			t.Fatalf("canonicalize: %v", err)
		}
		hexed := canonical.HashHex(hash)
		messageType := string(msg.ProtoReflect().Descriptor().FullName())
		eventTime := int64(1_700_000_000_000_000_000 + i)

		if err := local.AppendCanonical(ctx, payload, hash, eventTime,
			messageType, time.Now().UnixNano()); err != nil {
			t.Fatalf("local commit: %v", err)
		}
		want[hexed] = messageType

		if err := pub.Publish(ctx, &eventsv1.ArchiveRecord{
			CanonicalPayload: payload,
			EventHash:        hexed,
			EventTime:        eventTime,
			MessageType:      messageType,
			Tenant:           "t_parity",
		}); err != nil {
			t.Fatalf("publish: %v", err)
		}
	}

	// Wait for the archive to catch up rather than sleeping a guess.
	deadline := time.Now().Add(30 * time.Second)
	for {
		committed, rejected := cons.Counts()
		if rejected > 0 {
			t.Fatalf("the archive rejected %d records", rejected)
		}
		if committed >= int64(len(want)) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("archive committed %d of %d records before the deadline",
				committed, len(want))
		}
		time.Sleep(100 * time.Millisecond)
	}
	stopConsume()
	<-consumed

	// The assertion: every hash the collector wrote is in the archive,
	// carrying the same message type.
	got := map[string]string{}
	if err := remote.WalkEvents(ctx, func(row substrate.EventRow) error {
		got[canonical.HashHex(row.EventHash)] = row.MessageType
		return nil
	}); err != nil {
		t.Fatalf("walk archive: %v", err)
	}

	for hash, msgType := range want {
		archived, ok := got[hash]
		if !ok {
			t.Errorf("the collector wrote %s (%s) and the archive does not have it",
				hash[:12], msgType)
			continue
		}
		if archived != msgType {
			t.Errorf("%s: collector stored %s, archive stored %s", hash[:12], msgType, archived)
		}
	}
	if len(got) != len(want) {
		t.Errorf("archive holds %d records, the collector wrote %d", len(got), len(want))
	}
}

// A record whose bytes were corrupted in flight must be refused, not
// stored. Committing it would put a payload under a hash that does not
// describe it, and every later verification would fail on it.
func TestCorruptedPayloadIsRefused(t *testing.T) {
	store := openStore(t, "archive")
	cons := consumer.New(store, time.Now, nil)

	msg := &eventsv1.Outcome{EvaluationId: "ev_x", Outcome: "login_success"}
	payload, hash, err := canonical.MarshalAndHash(msg)
	if err != nil {
		t.Fatal(err)
	}
	corrupted := append([]byte(nil), payload...)
	corrupted[len(corrupted)-1] ^= 0xff

	err = cons.Handle(context.Background(), &eventsv1.ArchiveRecord{
		CanonicalPayload: corrupted,
		EventHash:        canonical.HashHex(hash),
		MessageType:      "ghosttrace.events.v1.Outcome",
	})
	if err != nil {
		t.Fatalf("a corrupted record should be dropped, not retried forever: %v", err)
	}
	if committed, rejected := cons.Counts(); committed != 0 || rejected != 1 {
		t.Errorf("counts = (%d committed, %d rejected), want (0, 1)", committed, rejected)
	}

	var n int
	_ = store.WalkEvents(context.Background(), func(substrate.EventRow) error { n++; return nil })
	if n != 0 {
		t.Errorf("archive stored %d records after refusing one, want 0", n)
	}
}
