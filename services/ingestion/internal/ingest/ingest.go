// Package ingest is the single Append entry point per
// docs/architecture/concurrency-pattern.md §Substrate-Writer
// Serialization. It composes the canonical-serialization pipeline
// (canonical.MarshalAndHash) + substrate commit (substrate.AppendPair)
// into one typed boundary that service-tier code calls.
//
// Per decision-log §0038, every primary observation commits paired
// with an IngestionEvent enrichment recording the act of ingestion
// (channel + verified client identity when delivered over mTLS). The
// pairing is atomic via substrate.AppendPair.
package ingest

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// Envelope captures the ingestion-act context observed by the service
// at the moment of receiving the primary observation. Threaded into the
// paired IngestionEvent per §0038.
//
// Fields:
//   - Channel: wire channel the primary observation was delivered on.
//     Inception values: "stdin", "http", "https", "https+mtls". Empty
//     coerced to "unspecified" at commit time to enforce a non-empty
//     channel marker.
//   - ClientCommonName / ClientSubjectAltNames / ClientCertSHA256:
//     populated when the channel is "https+mtls"; otherwise empty.
//   - ReceivedAt: Unix nanoseconds when the service first received the
//     primary observation. Distinct from the producer's declared_at
//     (which is on the primary observation itself) and from the
//     IngestionEvent's ingested_at (set at commit time).
type Envelope struct {
	Channel              string
	ClientCommonName     string
	ClientSubjectAltNames []string
	ClientCertSHA256     string
	ReceivedAt           int64
}

// channelOrUnspecified returns the channel field with empty coerced to
// "unspecified" so IngestionEvent.channel is never an empty string at
// commit time (operational discipline: every IngestionEvent carries a
// non-empty channel marker for forensic provenance).
func (e Envelope) channelOrUnspecified() string {
	if e.Channel == "" {
		return "unspecified"
	}
	return e.Channel
}

// Ingester accepts Protobuf messages, canonicalizes + hashes them, and
// commits to the substrate alongside a paired IngestionEvent enrichment.
// The Append method is the single per-process write entry point; it
// inherits writeMu serialization from the underlying *substrate.Substrate
// per concurrency-pattern §Substrate-Writer Serialization.
type Ingester struct {
	sub *substrate.Substrate
	now func() time.Time // injectable for deterministic tests
}

// New constructs an Ingester over sub. `now` is used for the
// committed_at + ingested_at timestamps; pass time.Now in production,
// an injected clock in tests.
func New(sub *substrate.Substrate, now func() time.Time) *Ingester {
	if now == nil {
		now = time.Now
	}
	return &Ingester{sub: sub, now: now}
}

// AppendReport is the per-Append outcome surfaced to callers. Carries
// the hex content-hashes of both the primary observation and its
// paired IngestionEvent enrichment so logging surfaces preserve the
// full per-ingest provenance pair.
type AppendReport struct {
	EventHashHex          string // primary observation content-hash
	PayloadBytes          int    // primary observation canonical-bytes length
	IngestionEventHashHex string // paired IngestionEvent content-hash
}

// Append commits msg under eventTime + env to the substrate. Idempotent
// on re-append of canonical-bytes-identical message+envelope (both the
// primary observation and the paired IngestionEvent are content-
// addressed; identical inputs produce identical hashes; substrate
// AppendPair is idempotent on PRIMARY KEY conflict).
//
// The message_type field is derived from the Protobuf descriptor full
// name; the payload_ref is derived from the content-hash (two-character
// shard prefix per substrate's blob-path convention).
//
// Two events commit atomically via substrate.AppendPair: the primary
// observation (the msg argument) and an IngestionEvent referencing the
// primary's content-hash + carrying env's channel + verified client
// identity (when available from mTLS).
func (in *Ingester) Append(ctx context.Context, msg proto.Message, eventTime int64, env Envelope) (AppendReport, error) {
	primaryPayload, primaryHash, err := canonical.MarshalAndHash(msg)
	if err != nil {
		return AppendReport{}, fmt.Errorf("ingest.Append: marshal primary: %w", err)
	}
	committedAt := in.now().UnixNano()

	primaryHex := canonical.HashHex(primaryHash)
	primaryRow := substrate.EventRow{
		EventHash:   primaryHash,
		EventTime:   eventTime,
		MessageType: string(msg.ProtoReflect().Descriptor().FullName()),
		PayloadRef:  primaryHex[:2] + "/" + primaryHex[2:],
		CommittedAt: committedAt,
	}

	// Construct the paired IngestionEvent. ReceivedAt defaults to
	// committedAt if the caller did not populate it (zero value).
	receivedAt := env.ReceivedAt
	if receivedAt == 0 {
		receivedAt = committedAt
	}
	ingEvent := &eventsv1.IngestionEvent{
		PrimaryEventHash:      primaryHash[:],
		ReceivedAt:            receivedAt,
		IngestedAt:            committedAt,
		Channel:               env.channelOrUnspecified(),
		ClientCommonName:      env.ClientCommonName,
		ClientSubjectAltNames: env.ClientSubjectAltNames,
		ClientCertSha256:      env.ClientCertSHA256,
	}
	enrichmentPayload, enrichmentHash, err := canonical.MarshalAndHash(ingEvent)
	if err != nil {
		return AppendReport{}, fmt.Errorf("ingest.Append: marshal enrichment: %w", err)
	}
	enrichmentHex := canonical.HashHex(enrichmentHash)
	enrichmentRow := substrate.EventRow{
		EventHash:   enrichmentHash,
		EventTime:   receivedAt,
		MessageType: string(ingEvent.ProtoReflect().Descriptor().FullName()),
		PayloadRef:  enrichmentHex[:2] + "/" + enrichmentHex[2:],
		CommittedAt: committedAt,
	}

	if err := in.sub.AppendPair(ctx, primaryRow, primaryPayload, enrichmentRow, enrichmentPayload); err != nil {
		return AppendReport{}, fmt.Errorf("ingest.Append: substrate: %w", err)
	}

	return AppendReport{
		EventHashHex:          primaryHex,
		PayloadBytes:          len(primaryPayload),
		IngestionEventHashHex: enrichmentHex,
	}, nil
}
