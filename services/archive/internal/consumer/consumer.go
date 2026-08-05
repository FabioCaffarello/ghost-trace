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

	eventsv1 "github.com/FabioCaffarello/ghost-trace/libs/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/libs/substrate"
)

// Store is the substrate surface this needs.
type Store interface {
	AppendCanonical(ctx context.Context, payload []byte, eventHash [32]byte,
		eventTime int64, messageType string, committedAt int64) error
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
)

// RejectReasons is every value the reason label can take.
var RejectReasons = []string{ReasonMalformedHash, ReasonHashMismatch}

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
func (c *Consumer) Handle(ctx context.Context, rec *eventsv1.ArchiveRecord) error {
	raw, err := hex.DecodeString(rec.GetEventHash())
	if err != nil || len(raw) != 32 {
		// Malformed identity. Redelivery cannot fix a hash that is not
		// a hash, so this is dropped rather than retried forever.
		c.reject(ctx, rec, ReasonMalformedHash, "event_hash is not 32 hex bytes")
		return nil
	}
	var hash [32]byte
	copy(hash[:], raw)

	err = c.store.AppendCanonical(ctx, rec.GetCanonicalPayload(), hash,
		rec.GetEventTime(), rec.GetMessageType(), c.now().UnixNano())
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
		c.reject(ctx, rec, ReasonHashMismatch, "payload does not match its hash")
		return nil

	default:
		// Anything else — a full disk, a locked database — is
		// transient as far as this service can tell. Nak and let it
		// come back.
		return fmt.Errorf("consumer: commit %s: %w", rec.GetEventHash(), err)
	}
}

func (c *Consumer) reject(ctx context.Context, rec *eventsv1.ArchiveRecord, reason, why string) {
	c.rejected.Add(1)
	c.meter.Rejected(reason)
	c.log.ErrorContext(ctx, "archive rejected a record",
		"why", why,
		"event_hash", rec.GetEventHash(),
		"message_type", rec.GetMessageType(),
		"rejected_total", c.rejected.Load())
}

// Counts reports what this process has committed and rejected.
func (c *Consumer) Counts() (committed, rejected int64) {
	return c.committed.Load(), c.rejected.Load()
}
