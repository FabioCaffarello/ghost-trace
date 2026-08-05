// Package eventstream is the naming and framing contract for records in
// flight between the service that produces them and the archive that
// stores them.
//
// It exists as its own module because it has two consumers by
// construction — a publisher in the collector and a consumer in the
// archive — which is the bar ADR-0001 sets for extraction. Nothing here
// talks to a broker; the transport lives with each side. This is the
// part they must agree on, and agreement is easiest to keep when there
// is exactly one definition of it.
package eventstream

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/libs/genproto/events/v1"
)

// Stream is the JetStream stream holding every archived record.
const Stream = "GT_EVENTS"

// SubjectPrefix roots every subject this package names.
const SubjectPrefix = "gt.events.v1"

// Kind is the record family a subject carries. It is derived from the
// protobuf message name rather than chosen by the caller, so a producer
// cannot publish an Evaluation onto the telemetry subject.
type Kind string

const (
	KindSession    Kind = "session"
	KindTelemetry  Kind = "telemetry"
	KindEvaluation Kind = "evaluation"
	KindOutcome    Kind = "outcome"
)

// ErrUnknownMessageType is returned for a message this stream has no
// subject for. It is deliberately an error rather than a fallback
// subject: a record quietly published to "gt.events.v1.other" is a
// record nobody consumes, which is indistinguishable from a record
// nobody produced.
var ErrUnknownMessageType = errors.New("eventstream: no subject for message type")

// kindByMessageType maps fully-qualified protobuf names to subjects.
// Adding a Category I record means adding a line here; leaving it out
// fails loudly at publish time.
var kindByMessageType = map[string]Kind{
	"ghosttrace.events.v1.SessionStart":   KindSession,
	"ghosttrace.events.v1.TelemetryBatch": KindTelemetry,
	"ghosttrace.events.v1.Evaluation":     KindEvaluation,
	"ghosttrace.events.v1.Outcome":        KindOutcome,
}

// MessageTypes returns every protobuf name a record can carry, in a
// stable order.
//
// Exposed so a consumer can declare a per-type counter for each of them
// AT ZERO before serving. A counter that appears only when its first
// record arrives cannot distinguish "none of this type yet" from "this
// type is not being counted", and the second is a bug that hides for as
// long as traffic is quiet.
func MessageTypes() []string {
	out := make([]string, 0, len(kindByMessageType))
	for t := range kindByMessageType {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

// KindFor returns the subject kind for a protobuf message name.
func KindFor(messageType string) (Kind, error) {
	k, ok := kindByMessageType[messageType]
	if !ok {
		return "", fmt.Errorf("%w: %q", ErrUnknownMessageType, messageType)
	}
	return k, nil
}

// Kinds returns every kind, in a stable order. Used to declare the
// stream's subjects and to assert coverage in tests.
func Kinds() []Kind {
	return []Kind{KindSession, KindTelemetry, KindEvaluation, KindOutcome}
}

// Subject names where one record travels:
//
//	gt.events.v1.<kind>.<tenant>
//
// Tenant is the last token so a consumer can filter to one tenant with
// a wildcard and without decoding a payload.
func Subject(kind Kind, tenant string) string {
	return SubjectPrefix + "." + string(kind) + "." + sanitizeToken(tenant)
}

// SubjectFilter is the wildcard a consumer binds to for every tenant.
func SubjectFilter(kind Kind) string {
	return SubjectPrefix + "." + string(kind) + ".*"
}

// AllSubjects is the wildcard covering the whole stream.
func AllSubjects() string { return SubjectPrefix + ".>" }

// SubjectFor is the common path: derive the subject from the record
// itself, so the subject and the payload cannot disagree.
func SubjectFor(rec *eventsv1.ArchiveRecord) (string, error) {
	kind, err := KindFor(rec.GetMessageType())
	if err != nil {
		return "", err
	}
	return Subject(kind, rec.GetTenant()), nil
}

// sanitizeToken keeps a tenant id from breaking subject structure. NATS
// treats "." as a separator and "*"/">" as wildcards, so a tenant
// containing one would silently publish to a different subject — or to
// all of them.
func sanitizeToken(s string) string {
	if s == "" {
		return "unknown"
	}
	return strings.NewReplacer(".", "_", "*", "_", ">", "_", " ", "_").Replace(s)
}
