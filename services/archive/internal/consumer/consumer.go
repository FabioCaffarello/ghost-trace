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

// Consumer commits arriving records.
type Consumer struct {
	store Store
	now   func() time.Time
	log   *slog.Logger

	committed atomic.Int64
	rejected  atomic.Int64
}

// New returns a consumer writing into store.
func New(store Store, now func() time.Time, log *slog.Logger) *Consumer {
	if now == nil {
		now = time.Now
	}
	if log == nil {
		log = slog.Default()
	}
	return &Consumer{store: store, now: now, log: log}
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
		c.reject(ctx, rec, "event_hash is not 32 hex bytes")
		return nil
	}
	var hash [32]byte
	copy(hash[:], raw)

	err = c.store.AppendCanonical(ctx, rec.GetCanonicalPayload(), hash,
		rec.GetEventTime(), rec.GetMessageType(), c.now().UnixNano())
	switch {
	case err == nil:
		c.committed.Add(1)
		return nil

	case errors.Is(err, substrate.ErrHashMismatch):
		// The bytes did not survive the trip. Committing them would put
		// a payload into a content-addressed store under a hash that
		// does not describe it, which is worse than losing the record —
		// every later verification would fail on it. Redelivery would
		// bring the same bytes, so drop and shout.
		c.reject(ctx, rec, "payload does not match its hash")
		return nil

	default:
		// Anything else — a full disk, a locked database — is
		// transient as far as this service can tell. Nak and let it
		// come back.
		return fmt.Errorf("consumer: commit %s: %w", rec.GetEventHash(), err)
	}
}

func (c *Consumer) reject(ctx context.Context, rec *eventsv1.ArchiveRecord, why string) {
	c.rejected.Add(1)
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
