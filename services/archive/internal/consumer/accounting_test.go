package consumer_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/FabioCaffarello/ghost-trace/libs/canonical"
	"github.com/FabioCaffarello/ghost-trace/libs/eventstream"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/libs/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/libs/substrate"
	"github.com/FabioCaffarello/ghost-trace/services/archive/internal/consumer"
)

// fakeStore records what it was asked to do, and can be told to fail.
type fakeStore struct {
	committed []uint64
	rejected  []uint64

	appendErr error
	rejectErr error
}

// The batch write, with the same all-or-nothing semantics the real one
// has: a hash mismatch anywhere fails the whole call and records
// nothing, so the test fake cannot be more forgiving than the store.
func (f *fakeStore) AppendCanonicalBatch(_ context.Context, recs []substrate.BatchRecord,
	rejected []uint64, _ int64) error {
	if f.appendErr != nil {
		return f.appendErr
	}
	// A transaction that cannot record its refusals cannot record its
	// commits either: both halves are one write in the real store, and
	// a fake that let the commits through would be testing a store that
	// does not exist.
	if f.rejectErr != nil && len(rejected) > 0 {
		return f.rejectErr
	}
	for _, r := range recs {
		if canonical.Hash(r.Payload) != r.EventHash {
			return substrate.ErrHashMismatch
		}
	}
	for _, r := range recs {
		f.committed = append(f.committed, r.Seq)
	}
	f.rejected = append(f.rejected, rejected...)
	return nil
}

func (f *fakeStore) RecordRejected(_ context.Context, seq uint64) error {
	if f.rejectErr != nil {
		return f.rejectErr
	}
	f.rejected = append(f.rejected, seq)
	return nil
}

// countingMeter records the reason labels it was handed.
type countingMeter struct {
	committed []string
	rejected  []string
}

func (m *countingMeter) Committed(t string) { m.committed = append(m.committed, t) }
func (m *countingMeter) Rejected(r string)  { m.rejected = append(m.rejected, r) }

func newConsumer(store consumer.Store, meter consumer.Meter) *consumer.Consumer {
	return consumer.New(store, meter, time.Now,
		slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func record(t *testing.T, body string) *eventsv1.ArchiveRecord {
	t.Helper()
	msg := &eventsv1.Outcome{EvaluationId: body, Outcome: "login_success"}
	payload, hash, err := canonical.MarshalAndHash(msg)
	if err != nil {
		t.Fatal(err)
	}
	return &eventsv1.ArchiveRecord{
		CanonicalPayload: payload,
		EventHash:        canonical.HashHex(hash),
		MessageType:      "ghosttrace.events.v1.Outcome",
	}
}

func TestARefusalIsWrittenDownAndNotJustLogged(t *testing.T) {
	// The change this file exists for. Before the durable position, a
	// refusal incremented a process counter and vanished on restart —
	// and the sequence it consumed then read as a record the transport
	// had lost. A deliberate decision must not be reported as loss.
	store, meter := &fakeStore{}, &countingMeter{}
	c := newConsumer(store, meter)

	rec := record(t, "ev_1")
	rec.EventHash = "not-hex"

	if err := c.HandleBatch(context.Background(), []eventstream.Item{{Record: rec, Delivery: eventstream.Delivery{Sequence: 7}}}); err != nil {
		t.Fatalf("a malformed hash must be dropped, not retried: %v", err)
	}
	if len(store.rejected) != 1 || store.rejected[0] != 7 {
		t.Errorf("store.rejected = %v, want [7] — the sequence must be accounted for", store.rejected)
	}
	if len(meter.rejected) != 1 || meter.rejected[0] != consumer.ReasonMalformedHash {
		t.Errorf("meter.rejected = %v, want [%s]", meter.rejected, consumer.ReasonMalformedHash)
	}
}

func TestARefusalThatCannotBeRecordedComesBack(t *testing.T) {
	// If the refusal cannot be written down, acking would consume the
	// sequence with nothing to explain it, and the audit would blame the
	// transport for a decision this service made. Naking gets the
	// accounting another chance; the record is still refused either way.
	store, meter := &fakeStore{rejectErr: errors.New("disk full")}, &countingMeter{}
	c := newConsumer(store, meter)

	rec := record(t, "ev_2")
	rec.EventHash = "not-hex"

	if err := c.HandleBatch(context.Background(), []eventstream.Item{{Record: rec, Delivery: eventstream.Delivery{Sequence: 9}}}); err == nil {
		t.Error("a refusal that could not be recorded was acknowledged anyway; its " +
			"sequence would read as transport loss forever")
	}
	if len(meter.rejected) != 0 {
		t.Errorf("meter counted %v for a refusal that was not durably recorded",
			meter.rejected)
	}
}

func TestARecordWithNoSequenceIsNotCommitted(t *testing.T) {
	// Committing without a position would leave the archive holding
	// content its own bookkeeping does not know about — the one
	// inconsistency the shared transaction exists to prevent.
	store, meter := &fakeStore{}, &countingMeter{}
	c := newConsumer(store, meter)

	if err := c.HandleBatch(context.Background(), []eventstream.Item{{Record: record(t, "ev_3"), Delivery: eventstream.Delivery{Sequence: 0}}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.committed) != 0 {
		t.Errorf("committed %v without a sequence", store.committed)
	}
	if len(meter.rejected) != 1 || meter.rejected[0] != consumer.ReasonNoSequence {
		t.Errorf("meter.rejected = %v, want [%s]", meter.rejected, consumer.ReasonNoSequence)
	}
}

func TestAnUndecodableMessageIsAccountedForToo(t *testing.T) {
	// The stream library terminates these before the handler ever runs.
	// The sequence is consumed regardless, so without this path it would
	// surface as unexplained loss.
	store, meter := &fakeStore{}, &countingMeter{}
	c := newConsumer(store, meter)

	c.Undecodable(eventstream.Delivery{Sequence: 12})

	if len(store.rejected) != 1 || store.rejected[0] != 12 {
		t.Errorf("store.rejected = %v, want [12]", store.rejected)
	}
	if len(meter.rejected) != 1 || meter.rejected[0] != consumer.ReasonUndecodable {
		t.Errorf("meter.rejected = %v, want [%s]", meter.rejected, consumer.ReasonUndecodable)
	}
}

func TestATransientFailureIsRetriedRatherThanAccounted(t *testing.T) {
	// A full disk is not a refusal. Recording it as one would explain
	// away a record the archive is supposed to keep trying for.
	store, meter := &fakeStore{appendErr: errors.New("database is locked")}, &countingMeter{}
	c := newConsumer(store, meter)

	if err := c.HandleBatch(context.Background(), []eventstream.Item{{Record: record(t, "ev_4"), Delivery: eventstream.Delivery{Sequence: 3}}}); err == nil {
		t.Error("a transient failure was acknowledged; the record would be lost")
	}
	if len(store.rejected) != 0 {
		t.Errorf("a transient failure was recorded as a deliberate refusal: %v",
			store.rejected)
	}
}

func TestEveryReasonTheConsumerEmitsIsDeclarable(t *testing.T) {
	// RejectReasons is what the meter declares at zero before serving. A
	// reason emitted but not listed would mint an undeclared series the
	// first time it went wrong — the one moment nobody wants to discover
	// the counter is new.
	emitted := []string{
		consumer.ReasonMalformedHash, consumer.ReasonHashMismatch,
		consumer.ReasonUndecodable, consumer.ReasonNoSequence,
	}
	for _, r := range emitted {
		found := false
		for _, known := range consumer.RejectReasons {
			if known == r {
				found = true
			}
		}
		if !found {
			t.Errorf("reason %q is emitted but not in RejectReasons", r)
		}
	}
	if len(consumer.RejectReasons) != len(emitted) {
		t.Errorf("RejectReasons has %d entries, %d are emitted; a listed reason that "+
			"nothing emits is a series that can never leave zero",
			len(consumer.RejectReasons), len(emitted))
	}
}

func TestABatchCommitsWhatItKeepsAndAccountsWhatItRefuses(t *testing.T) {
	// The partition, which is what HandleBatch is for. Good records
	// become rows, a malformed identity becomes an accounted sequence,
	// and both travel in one call so a rollback un-decides both.
	store := &fakeStore{}
	meter := &countingMeter{}
	c := newConsumer(store, meter)

	batch := []eventstream.Item{
		{Record: record(t, "ev_a"), Delivery: eventstream.Delivery{Sequence: 1}},
		{Record: &eventsv1.ArchiveRecord{EventHash: "not-hex", MessageType: "t"},
			Delivery: eventstream.Delivery{Sequence: 2}},
		{Record: record(t, "ev_b"), Delivery: eventstream.Delivery{Sequence: 3}},
	}
	if err := c.HandleBatch(context.Background(), batch); err != nil {
		t.Fatalf("HandleBatch: %v", err)
	}

	if len(store.committed) != 2 || store.committed[0] != 1 || store.committed[1] != 3 {
		t.Errorf("committed = %v, want [1 3]", store.committed)
	}
	if len(store.rejected) != 1 || store.rejected[0] != 2 {
		t.Errorf("rejected = %v, want [2] — a sequence the archive refused on "+
			"purpose must still advance the position, or it reads as loss",
			store.rejected)
	}
	if len(meter.committed) != 2 || len(meter.rejected) != 1 {
		t.Errorf("meter saw %d committed and %d rejected, want 2 and 1",
			len(meter.committed), len(meter.rejected))
	}
}

func TestAFailedBatchAnnouncesNothing(t *testing.T) {
	// Counting or logging before the transaction commits would report
	// records the archive does not have: the batch is naked and
	// redelivered whole, and the same records would be counted again.
	store := &fakeStore{appendErr: errors.New("disk is full")}
	meter := &countingMeter{}
	c := newConsumer(store, meter)

	err := c.HandleBatch(context.Background(), []eventstream.Item{
		{Record: record(t, "ev_a"), Delivery: eventstream.Delivery{Sequence: 1}},
		{Record: &eventsv1.ArchiveRecord{EventHash: "not-hex", MessageType: "t"},
			Delivery: eventstream.Delivery{Sequence: 2}},
	})
	if err == nil {
		t.Fatal("a batch that could not be written reported success; it would be " +
			"acknowledged and the records lost")
	}
	if len(meter.committed) != 0 || len(meter.rejected) != 0 {
		t.Errorf("a failed batch counted %d commits and %d rejections; nothing "+
			"happened, so nothing may be announced",
			len(meter.committed), len(meter.rejected))
	}
}

func TestOnePoisonedRecordDoesNotStopTheRestOfTheBatch(t *testing.T) {
	// The slow path. A payload that does not match its hash fails the
	// whole all-or-nothing batch, so the batch is re-driven singly: the
	// bad one is refused and the good ones still land. The first
	// version of this recursed until the stack ran out.
	store := &fakeStore{}
	meter := &countingMeter{}
	c := newConsumer(store, meter)

	good := record(t, "ev_good")
	bad := record(t, "ev_bad")
	bad.CanonicalPayload = append([]byte(nil), bad.CanonicalPayload...)
	bad.CanonicalPayload[0] ^= 0xff // same hash, different bytes

	if err := c.HandleBatch(context.Background(), []eventstream.Item{
		{Record: good, Delivery: eventstream.Delivery{Sequence: 1}},
		{Record: bad, Delivery: eventstream.Delivery{Sequence: 2}},
	}); err != nil {
		t.Fatalf("HandleBatch: %v", err)
	}

	if len(store.committed) != 1 || store.committed[0] != 1 {
		t.Errorf("committed = %v, want [1] — the healthy record must survive a "+
			"neighbour that does not", store.committed)
	}
	if len(store.rejected) != 1 || store.rejected[0] != 2 {
		t.Errorf("rejected = %v, want [2]", store.rejected)
	}
}
