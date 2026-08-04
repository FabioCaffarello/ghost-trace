package eventstream

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	"github.com/FabioCaffarello/ghost-trace/libs/canonical"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/libs/genproto/events/v1"
)

// Record canonicalizes a message into the envelope the stream carries.
//
// The bytes are produced HERE, once, and travel as bytes. Nothing
// downstream re-marshals the message to recover them: deterministic
// marshalling makes no promise across builds, so a record re-encoded by
// a different binary could hash differently and stop being the same
// record. Content addressing only means something if the content is
// fixed at exactly one point.
func Record(msg proto.Message, eventTime int64, tenant string) (*eventsv1.ArchiveRecord, error) {
	payload, hash, err := canonical.MarshalAndHash(msg)
	if err != nil {
		return nil, fmt.Errorf("eventstream: canonicalize: %w", err)
	}
	return &eventsv1.ArchiveRecord{
		CanonicalPayload: payload,
		EventHash:        canonical.HashHex(hash),
		EventTime:        eventTime,
		MessageType:      string(msg.ProtoReflect().Descriptor().FullName()),
		Tenant:           tenant,
	}, nil
}

// Archive is the stream as the durable store: canonicalize, publish,
// and REPORT what went wrong.
//
// This is the opposite policy to the collector's mirror, and
// deliberately so. The collector writes to a local substrate first and
// treats publication as best effort, because it has somewhere else to
// put the record (ADR-0003). A service whose only durable store is the
// stream has no such fallback — swallowing a publish failure here would
// answer 202 to an outcome that was never stored, which is precisely
// the silent calibration poisoning /v1/outcomes refuses to risk.
//
// The mechanism is shared with that mirror; only the policy differs.
type Archive struct {
	pub    *Publisher
	tenant string
}

// NewArchive returns a store that publishes onto the stream.
func NewArchive(pub *Publisher, tenant string) *Archive {
	return &Archive{pub: pub, tenant: tenant}
}

// Append publishes one record, returning any failure.
func (a *Archive) Append(ctx context.Context, msg proto.Message, eventTime int64) error {
	rec, err := Record(msg, eventTime, a.tenant)
	if err != nil {
		return err
	}
	return a.pub.Publish(ctx, rec)
}
