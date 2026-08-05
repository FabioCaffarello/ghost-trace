// Package consumer commits records arriving off the event stream into
// the substrate.
//
// It never re-marshals anything. A record arrives as canonical bytes
// plus the hash the producer computed over them, and the substrate
// verifies that hash before committing. That is deliberate: protobuf's
// deterministic marshalling makes no cross-build promise, so a consumer
// that recomputed the encoding could disagree with the producer about a
// record's identity across nothing more than a version skew. Verifying
// bytes it did not produce is the strongest check available and the
// only one that stays true.
package consumer

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/FabioCaffarello/ghost-trace/libs/eventstream"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/libs/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/libs/substrate"
)

// Store is the substrate surface this needs.
//
// The two writes are separate methods rather than one because they
// answer different questions: AppendCanonicalAt says the archive HOLDS a
// record, RecordRejected says the archive DECIDED not to. Both advance
// the durable position, and that is the point — a sequence the archive
// walked past and cannot explain is loss, and one it refused on purpose
// is not.
type Store interface {
	AppendCanonicalAt(ctx context.Context, payload []byte, eventHash [32]byte,
		eventTime int64, messageType string, committedAt int64, seq uint64) error
	RecordRejected(ctx context.Context, seq uint64) error
}

// Reject reasons. Constants because every one of them is declared to
// the meter at zero before serving; a literal at a call site would mint
// an undeclared series the first time it went wrong, which is the one
// moment nobody wants to discover the counter is new.
const (
	// ReasonMalformedHash: the identity is not 32 hex bytes. Redelivery
	// cannot make it one.
	ReasonMalformedHash = "malformed_hash"

	// ReasonHashMismatch: the payload does not match the hash that
	// names it. Committing would poison a content-addressed store.
	ReasonHashMismatch = "hash_mismatch"

	// ReasonUndecodable: the bytes are not an ArchiveRecord at all, so
	// the stream library terminated the message without ever calling the
	// handler. Counted here anyway, because the sequence was consumed and
	// an unaccounted sequence reads as transport loss.
	ReasonUndecodable = "undecodable"

	// ReasonNoSequence: the broker's metadata could not be read, so the
	// record cannot be placed in the stream. It is not committed —
	// committing without a position would advance the archive's contents
	// without advancing its bookkeeping, which is the one inconsistency
	// the durable position exists to make impossible.
	ReasonNoSequence = "no_sequence"
)

// RejectReasons is every value the reason label can take.
var RejectReasons = []string{ReasonMalformedHash, ReasonHashMismatch,
	ReasonUndecodable, ReasonNoSequence}

// Meter counts what the archive commits and refuses.
//
// Committed is labelled by message type so a type that stops arriving
// is visible; rejected is labelled by reason so the two failure modes
// are not one number. Both are declared at zero by the adapter — see
// the collector's lossmeter for why that is not optional.
type Meter interface {
	Committed(messageType string)
	Rejected(reason string)
}

// NoMeter counts nothing.
type NoMeter struct{}

// Committed discards the count.
func (NoMeter) Committed(string) {}

// Rejected discards the count.
func (NoMeter) Rejected(string) {}

// Consumer commits arriving records.
type Consumer struct {
	store Store
	meter Meter
	now   func() time.Time
	log   *slog.Logger

	committed atomic.Int64
	rejected  atomic.Int64
}

// New returns a consumer writing into store, counting into meter.
//
// The meter is a constructor argument rather than a WithMeter option,
// which is the idiom everywhere else here. A Consumer holds atomic
// counters, so the copy that idiom performs would copy a lock — `go
// vet` says so, and it is right: two Consumers sharing one counter's
// address is exactly the sort of thing that reads correct and drifts.
// Pass NoMeter{} for a run that counts nothing.
func New(store Store, meter Meter, now func() time.Time, log *slog.Logger) *Consumer {
	if meter == nil {
		meter = NoMeter{}
	}
	if now == nil {
		now = time.Now
	}
	if log == nil {
		log = slog.Default()
	}
	return &Consumer{store: store, meter: meter, now: now, log: log}
}

// Handle commits one record. Returning an error means the message is
// negatively acknowledged and redelivered, so this returns nil only
// when the record is durable — or when redelivering it could never
// help.
//
// EVERY PATH OUT OF HERE ACCOUNTS FOR ITS SEQUENCE, or returns an error
// so the message comes back. That is the invariant the audit rests on:
// a sequence the archive walked past and did not explain is loss, so a
// silent return would manufacture loss that never happened — and, worse,
// would look exactly like the real thing.
func (c *Consumer) Handle(ctx context.Context, rec *eventsv1.ArchiveRecord,
	d eventstream.Delivery) error {

	if d.Sequence == 0 {
		// No position, no commit. Writing the record while failing to
		// record where it came from would leave the archive holding
		// content its own bookkeeping does not know about, which is the
		// single inconsistency the durable position exists to prevent.
		// Nothing to account against either, so this is counted in the
		// process and shouted about rather than written down.
		c.rejected.Add(1)
		c.meter.Rejected(ReasonNoSequence)
		c.log.ErrorContext(ctx, "archive could not place a record in the stream",
			"why", "the broker returned no sequence for this delivery",
			"event_hash", rec.GetEventHash(),
			"message_type", rec.GetMessageType())
		return nil
	}

	raw, err := hex.DecodeString(rec.GetEventHash())
	if err != nil || len(raw) != 32 {
		// Malformed identity. Redelivery cannot fix a hash that is not
		// a hash, so this is dropped rather than retried forever.
		return c.reject(ctx, rec, d, ReasonMalformedHash, "event_hash is not 32 hex bytes")
	}
	var hash [32]byte
	copy(hash[:], raw)

	err = c.store.AppendCanonicalAt(ctx, rec.GetCanonicalPayload(), hash,
		rec.GetEventTime(), rec.GetMessageType(), c.now().UnixNano(), d.Sequence)
	switch {
	case err == nil:
		c.committed.Add(1)
		c.meter.Committed(rec.GetMessageType())
		return nil

	case errors.Is(err, substrate.ErrHashMismatch):
		// The bytes did not survive the trip. Committing them would put
		// a payload into a content-addressed store under a hash that
		// does not describe it, which is worse than losing the record —
		// every later verification would fail on it. Redelivery would
		// bring the same bytes, so drop and shout.
		return c.reject(ctx, rec, d, ReasonHashMismatch, "payload does not match its hash")

	default:
		// Anything else — a full disk, a locked database — is
		// transient as far as this service can tell. Nak and let it
		// come back.
		return fmt.Errorf("consumer: commit %s: %w", rec.GetEventHash(), err)
	}
}

// Undecodable accounts for a message the stream library terminated
// before this consumer ever saw it.
func (c *Consumer) Undecodable(d eventstream.Delivery) {
	ctx := context.Background()
	if d.Sequence == 0 {
		c.rejected.Add(1)
		c.meter.Rejected(ReasonNoSequence)
		return
	}
	if err := c.store.RecordRejected(ctx, d.Sequence); err != nil {
		// The refusal cannot be written down, so the sequence will read
		// as unexplained loss. Say so rather than letting the audit
		// quietly blame the transport.
		c.log.ErrorContext(ctx, "archive could not account for an undecodable message",
			"error", err, "sequence", d.Sequence)
	}
	c.rejected.Add(1)
	c.meter.Rejected(ReasonUndecodable)
	c.log.ErrorContext(ctx, "archive terminated an undecodable message",
		"sequence", d.Sequence, "rejected_total", c.rejected.Load())
}

// reject records a deliberate refusal, durably.
//
// It returns an error only when the refusal could NOT be written down.
// That is on purpose: a refusal nobody recorded leaves its sequence
// unaccounted, and the audit would report it as a record the stream
// lost. Naking sends the message back so the refusal can be recorded on
// the next attempt — the record is still refused, but the accounting
// gets another chance to be true.
func (c *Consumer) reject(ctx context.Context, rec *eventsv1.ArchiveRecord,
	d eventstream.Delivery, reason, why string) error {

	if err := c.store.RecordRejected(ctx, d.Sequence); err != nil {
		return fmt.Errorf("consumer: account for refused %s: %w", rec.GetEventHash(), err)
	}
	c.rejected.Add(1)
	c.meter.Rejected(reason)
	c.log.ErrorContext(ctx, "archive rejected a record",
		"why", why,
		"sequence", d.Sequence,
		"event_hash", rec.GetEventHash(),
		"message_type", rec.GetMessageType(),
		"rejected_total", c.rejected.Load())
	return nil
}

// Counts reports what this process has committed and rejected.
func (c *Consumer) Counts() (committed, rejected int64) {
	return c.committed.Load(), c.rejected.Load()
}
