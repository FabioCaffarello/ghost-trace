// Package ingest is the single write entry point to the substrate.
//
// It exists to keep one property in one place: a record's identity is
// the BLAKE3 hash of its canonical bytes, so appending the same record
// twice is a no-op rather than a duplicate. Telemetry batches arrive out
// of order and get retried after timeouts (contract §2), so this is not
// a theoretical concern — retries are the normal case.
//
// Writing raw telemetry to an archive is not on the decision path. It
// exists because M2 must re-score recorded sessions offline when a
// threshold changes: you cannot re-recruit twenty people every time a
// constant moves, so the sessions have to be replayable.
package ingest

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// Ingester canonicalizes messages and commits them to the substrate.
type Ingester struct {
	sub *substrate.Substrate
	now func() time.Time
}

// New constructs an Ingester over sub. Pass nil for now to use
// time.Now; tests inject a fixed clock.
func New(sub *substrate.Substrate, now func() time.Time) *Ingester {
	if now == nil {
		now = time.Now
	}
	return &Ingester{sub: sub, now: now}
}

// Report describes the outcome of an Append.
type Report struct {
	// Hash is the record's content address.
	Hash [32]byte

	// HashHex is Hash rendered for logs and URLs.
	HashHex string

	// MessageType is the full protobuf name, e.g.
	// "ghosttrace.events.v1.TelemetryBatch".
	MessageType string
}

// Append canonicalizes msg, hashes it, and commits it to the substrate.
//
// eventTime is the record's own notion of when it happened, in Unix
// nanoseconds — distinct from the commit timestamp, which the substrate
// sets. Re-appending identical content is a no-op and returns the same
// hash.
func (in *Ingester) Append(ctx context.Context, msg proto.Message, eventTime int64) (Report, error) {
	payload, hash, err := canonical.MarshalAndHash(msg)
	if err != nil {
		return Report{}, fmt.Errorf("ingest.Append: canonicalize: %w", err)
	}

	hexed := canonical.HashHex(hash)
	messageType := string(msg.ProtoReflect().Descriptor().FullName())

	row := substrate.EventRow{
		EventHash:   hash,
		EventTime:   eventTime,
		MessageType: messageType,
		PayloadRef:  hexed[:2] + "/" + hexed[2:],
		CommittedAt: in.now().UnixNano(),
	}

	if err := in.sub.Append(ctx, row, payload); err != nil {
		return Report{}, fmt.Errorf("ingest.Append: commit: %w", err)
	}

	return Report{Hash: hash, HashHex: hexed, MessageType: messageType}, nil
}
