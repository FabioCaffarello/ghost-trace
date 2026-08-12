package eventstream

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

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
func Record(msg proto.Message, eventTime int64) (*eventsv1.ArchiveRecord, error) {
	tenant, err := tenantOf(msg)
	if err != nil {
		return nil, err
	}
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

// ErrNoTenant is returned for a payload carrying no tenant_id.
var ErrNoTenant = errors.New("eventstream: record has no tenant_id")

// tenantOf reads the tenant out of the RECORD, by reflection over the
// payload's own tenant_id field.
//
// IT USED TO BE A CONSTRUCTOR ARGUMENT, taken from the -tenant flag,
// and that was wrong in a way nothing caught. Every payload already
// carries the tenant the REQUEST proved — resolved from the presented
// key, per contract §1 — while the envelope carried whatever the
// process was started with. With a single-tenant deployment the two
// always agreed. With `-tenants <file>` they agree only for the tenant
// that happens to match the flag, and every other customer's records
// were archived, and subject-routed, as `t_demo`.
//
// So the envelope is now DERIVED rather than supplied. A subject that
// disagrees with its payload is no longer a thing a caller can express,
// which is the same reason SubjectFor reads the record instead of
// taking a parameter.
//
// Reflection rather than an interface assertion because the five
// payload types are generated and share no Go interface; they share a
// field, `tenant_id = 1`, which every one of them declares. A payload
// that stops declaring it fails here rather than being published to a
// subject nobody can filter.
func tenantOf(msg proto.Message) (string, error) {
	m := msg.ProtoReflect()
	fd := m.Descriptor().Fields().ByName("tenant_id")
	if fd == nil || fd.Kind() != protoreflect.StringKind {
		return "", fmt.Errorf("%w: %s declares no string tenant_id",
			ErrNoTenant, m.Descriptor().FullName())
	}
	tenant := m.Get(fd).String()
	if tenant == "" {
		// An empty tenant would route to `gt.events.v1.<kind>.` and be
		// attributed to nobody. Refusing is the only honest option: the
		// archive's whole job is saying who a record belongs to.
		return "", fmt.Errorf("%w: %s left it empty",
			ErrNoTenant, m.Descriptor().FullName())
	}
	return tenant, nil
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
	pub *Publisher
}

// NewArchive returns a store that publishes onto the stream.
func NewArchive(pub *Publisher) *Archive {
	return &Archive{pub: pub}
}

// Append publishes one record, returning any failure.
func (a *Archive) Append(ctx context.Context, msg proto.Message, eventTime int64) error {
	rec, err := Record(msg, eventTime)
	if err != nil {
		return err
	}
	return a.pub.Publish(ctx, rec)
}
