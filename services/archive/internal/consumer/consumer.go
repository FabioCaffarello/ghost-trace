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
	// AppendCanonicalBatch commits what the archive keeps and accounts
	// for what it refuses, in ONE transaction. Both halves travel
	// together because a reject recorded outside the transaction would
	// survive a rollback and be recorded again on redelivery — ADR-0010
	// arriving through a different door.
	AppendCanonicalBatch(ctx context.Context, recs []substrate.BatchRecord,
		rejected []uint64, committedAt int64) error

	// RecordRejected accounts for a single sequence outside any batch.
	// Only the undecodable path uses it, and only because that message
	// was TERMINATED rather than naked: it will never be redelivered,
	// so there is no second recording to guard against.
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
// HandleBatch commits a whole fetch.
//
// It partitions rather than loops: records the archive will KEEP become
// substrate rows, records it REFUSES become accounted sequences, and
// both go into one transaction. Returning an error naks the entire
// batch, so nothing here may have written anything durable of its own
// first — which is why rejections are collected rather than recorded as
// they are found.
func (c *Consumer) HandleBatch(ctx context.Context, batch []eventstream.Item) error {
	recs := make([]substrate.BatchRecord, 0, len(batch))
	var rejected []uint64

	// Deferred logging: a batch that fails to commit is retried whole,
	// and shouting about rejections that got rolled back would fill the
	// log with events that did not happen.
	var notes []rejectNote

	for _, it := range batch {
		rec, d := it.Record, it.Delivery

		if d.Sequence == 0 {
			// No position, no commit. Writing the record while failing
			// to record where it came from would leave the archive
			// holding content its own bookkeeping does not know about.
			// Nothing to account against either, so this is counted in
			// the process and shouted about rather than written down.
			c.rejected.Add(1)
			c.meter.Rejected(ReasonNoSequence)
			c.log.ErrorContext(ctx, "archive could not place a record in the stream",
				"why", "the broker returned no sequence for this delivery",
				"event_hash", rec.GetEventHash(),
				"message_type", rec.GetMessageType())
			continue
		}

		raw, err := hex.DecodeString(rec.GetEventHash())
		if err != nil || len(raw) != 32 {
			// Malformed identity. Redelivery cannot fix a hash that is
			// not a hash, so this is refused rather than retried.
			rejected = append(rejected, d.Sequence)
			notes = append(notes, rejectNote{seq: d.Sequence,
				reason: ReasonMalformedHash,
				why:    "event_hash is not 32 hex bytes",
				hash:   rec.GetEventHash()})
			continue
		}
		var hash [32]byte
		copy(hash[:], raw)

		// The hash is verified again inside the substrate, over the
		// same bytes, before anything is written. Checking it here as
		// well would be a second implementation of the one check that
		// must not drift.
		recs = append(recs, substrate.BatchRecord{
			Payload:     rec.GetCanonicalPayload(),
			EventHash:   hash,
			EventTime:   rec.GetEventTime(),
			MessageType: rec.GetMessageType(),
			Seq:         d.Sequence,
		})
	}

	err := c.store.AppendCanonicalBatch(ctx, recs, rejected, c.now().UnixNano())
	switch {
	case err == nil:
		// nothing to do; counted below

	case errors.Is(err, substrate.ErrHashMismatch):
		// One record's bytes did not survive the trip, and an
		// all-or-nothing batch cannot tell which from here. Re-drive it
		// one at a time: the fast path stays free of per-record error
		// handling and the slow path is exact.
		return c.isolate(ctx, recs, notes)

	default:
		// Anything else — a full disk, a locked database — is transient
		// as far as this service can tell. Nak the batch and let it
		// come back.
		return fmt.Errorf("consumer: commit batch of %d: %w", len(recs), err)
	}

	c.report(ctx, recs, notes)
	return nil
}

// rejectNote is a refusal waiting to be announced, once the transaction
// that recorded it has actually committed.
type rejectNote struct {
	seq    uint64
	reason string
	why    string
	hash   string
}

// isolate is the slow path: a batch that failed on a hash mismatch,
// re-driven one record at a time so the poisoned record is found and
// the rest still land.
//
// It does NOT call HandleBatch again. The first version did, and a
// batch of one whose single record was the bad one recursed until the
// stack ran out — caught by TestCorruptedPayloadIsRefused, which
// panicked instead of failing. A slow path that can re-enter the fast
// path is a slow path that can re-enter itself.
func (c *Consumer) isolate(ctx context.Context, recs []substrate.BatchRecord,
	notes []rejectNote) error {

	var poisoned []uint64
	committed := make([]substrate.BatchRecord, 0, len(recs))

	for _, r := range recs {
		err := c.store.AppendCanonicalBatch(ctx, []substrate.BatchRecord{r}, nil,
			c.now().UnixNano())
		switch {
		case err == nil:
			committed = append(committed, r)
		case errors.Is(err, substrate.ErrHashMismatch):
			// The bytes did not survive the trip. Committing them would
			// put a payload into a content-addressed store under a hash
			// that does not describe it — worse than losing the record,
			// because every later verification would fail on it.
			// Redelivery brings the same bytes, so refuse it.
			poisoned = append(poisoned, r.Seq)
			notes = append(notes, rejectNote{seq: r.Seq, reason: ReasonHashMismatch,
				why: "payload does not match its hash", hash: hexOf(r.EventHash)})
		default:
			return fmt.Errorf("consumer: isolate commit at %d: %w", r.Seq, err)
		}
	}

	// The refusals, accounted together. A separate transaction from the
	// commits above is safe here and only here: if it fails the batch is
	// naked, redelivery re-commits the good records as duplicates and
	// re-refuses the bad ones, and the position lands in the same place.
	if len(poisoned) > 0 {
		if err := c.store.AppendCanonicalBatch(ctx, nil, poisoned, c.now().UnixNano()); err != nil {
			return fmt.Errorf("consumer: account for %d refused: %w", len(poisoned), err)
		}
	}

	c.report(ctx, committed, notes)
	return nil
}

// report counts and logs, once a batch is durable. Kept apart from the
// writing so that nothing is announced before it is true — a batch that
// is naked and retried must not have shouted about rejections that got
// rolled back.
func (c *Consumer) report(ctx context.Context, recs []substrate.BatchRecord, notes []rejectNote) {
	for _, n := range notes {
		c.rejected.Add(1)
		c.meter.Rejected(n.reason)
		c.log.ErrorContext(ctx, "archive rejected a record",
			"why", n.why, "sequence", n.seq, "event_hash", n.hash,
			"rejected_total", c.rejected.Load())
	}
	for _, r := range recs {
		c.committed.Add(1)
		c.meter.Committed(r.MessageType)
	}
}

func hexOf(h [32]byte) string { return hex.EncodeToString(h[:]) }

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

// Counts reports what this process has committed and rejected.
func (c *Consumer) Counts() (committed, rejected int64) {
	return c.committed.Load(), c.rejected.Load()
}
