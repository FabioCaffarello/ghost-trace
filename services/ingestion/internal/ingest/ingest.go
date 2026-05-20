// Package ingest is the single Append entry point per
// docs/architecture/concurrency-pattern.md §Substrate-Writer
// Serialization. It composes the canonical-serialization pipeline
// (canonical.MarshalAndHash) + substrate commit (substrate.Append) into
// one typed boundary that service-tier code calls.
package ingest

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// Ingester accepts Protobuf messages, canonicalizes + hashes them, and
// commits to the substrate. The Append method is the single per-process
// write entry point; it inherits writeMu serialization from the
// underlying *substrate.Substrate per concurrency-pattern §Substrate-
// Writer Serialization.
type Ingester struct {
	sub *substrate.Substrate
	now func() time.Time // injectable for deterministic tests
}

// New constructs an Ingester over sub. `now` is used for the
// committed_at timestamp; pass time.Now in production, an injected
// clock in tests.
func New(sub *substrate.Substrate, now func() time.Time) *Ingester {
	if now == nil {
		now = time.Now
	}
	return &Ingester{sub: sub, now: now}
}

// AppendReport is the per-Append outcome surfaced to callers.
type AppendReport struct {
	EventHashHex string
	PayloadBytes int
}

// Append commits msg under eventTime to the substrate. Idempotent on
// re-append of canonical-bytes-identical message.
//
// The message_type field is derived from the Protobuf descriptor full
// name; the payload_ref is derived from the content-hash (two-character
// shard prefix per substrate's blob-path convention).
func (in *Ingester) Append(ctx context.Context, msg proto.Message, eventTime int64) (AppendReport, error) {
	payload, hash, err := canonical.MarshalAndHash(msg)
	if err != nil {
		return AppendReport{}, fmt.Errorf("ingest.Append: marshal: %w", err)
	}

	hex := canonical.HashHex(hash)
	row := substrate.EventRow{
		EventHash:   hash,
		EventTime:   eventTime,
		MessageType: string(msg.ProtoReflect().Descriptor().FullName()),
		PayloadRef:  hex[:2] + "/" + hex[2:],
		CommittedAt: in.now().UnixNano(),
	}

	if err := in.sub.Append(ctx, row, payload); err != nil {
		return AppendReport{}, fmt.Errorf("ingest.Append: substrate: %w", err)
	}

	return AppendReport{
		EventHashHex: hex,
		PayloadBytes: len(payload),
	}, nil
}
