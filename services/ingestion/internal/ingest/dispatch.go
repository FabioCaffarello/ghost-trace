package ingest

import (
	"google.golang.org/protobuf/proto"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
)

// MessageDescriptor binds a Cat I primary-observation message type to
// the transport-level routing keys + the typed accessors the ingestion
// pipeline needs at the type-erased boundary. One descriptor per
// registered Cat I type.
//
// Adding a new Cat I message type:
//  1. Author the .proto under schemas/events/v1/.
//  2. Add a generation entry in services/ingestion/Makefile.
//  3. Append a descriptor to Registry below.
//  4. Register a canonical-corpus factory entry in
//     internal/canonical/corpus_test.go.
//  5. Author at least one .json corpus entry per the canonical-
//     serialization-contract coverage requirement.
//
// EventTime extraction is type-specific because every Cat I message
// carries its producer-reported time under a type-named field
// (DeclaredSession.DeclaredAt, NetworkEvent.ObservedAt, ...). The
// rename is intentional per entity-model.md (the field name encodes
// the producer-class distinction).
type MessageDescriptor struct {
	// URLPath is the trailing path segment for HTTP routing under
	// /v1/events/<URLPath>. Lowercase, hyphen-separated, stable across
	// schemas-evolution events (it is part of the producer contract).
	URLPath string

	// StdinType is the `type` identifier the stdin envelope carries on
	// each input line. Lowercase, underscore-separated to mirror
	// Protobuf snake_case message-name convention.
	StdinType string

	// New returns a fresh zero-value message of this type. Used by the
	// HTTP + stdin paths to allocate a target for proto.Unmarshal /
	// protojson.Unmarshal.
	New func() proto.Message

	// EventTime extracts the producer-reported event time (Unix ns)
	// from msg. msg MUST be a value returned by this descriptor's New
	// (i.e. the concrete type). Cross-type calls will panic at the
	// type assertion — this is intentional: the dispatch tables that
	// pair New + EventTime ensure the types align.
	EventTime func(msg proto.Message) int64
}

// Registry enumerates every Cat I message type the ingestion pipeline
// accepts. Order is stable; new types append.
var Registry = []MessageDescriptor{
	{
		URLPath:   "declared-session",
		StdinType: "declared_session",
		New:       func() proto.Message { return &eventsv1.DeclaredSession{} },
		EventTime: func(m proto.Message) int64 { return m.(*eventsv1.DeclaredSession).GetDeclaredAt() },
	},
	{
		URLPath:   "network-event",
		StdinType: "network_event",
		New:       func() proto.Message { return &eventsv1.NetworkEvent{} },
		EventTime: func(m proto.Message) int64 { return m.(*eventsv1.NetworkEvent).GetObservedAt() },
	},
}

// LookupURLPath returns the descriptor whose URLPath matches path, or
// (zero, false) when none matches.
func LookupURLPath(path string) (MessageDescriptor, bool) {
	for _, d := range Registry {
		if d.URLPath == path {
			return d, true
		}
	}
	return MessageDescriptor{}, false
}

// LookupStdinType returns the descriptor whose StdinType matches t, or
// (zero, false) when none matches.
func LookupStdinType(t string) (MessageDescriptor, bool) {
	for _, d := range Registry {
		if d.StdinType == t {
			return d, true
		}
	}
	return MessageDescriptor{}, false
}

// KnownURLPaths returns the registered URL paths in Registry order.
// Used by 404 + 400 error responses to enumerate the accepted types.
func KnownURLPaths() []string {
	out := make([]string, 0, len(Registry))
	for _, d := range Registry {
		out = append(out, d.URLPath)
	}
	return out
}

// KnownStdinTypes returns the registered stdin type tags in Registry
// order. Used by the stdin worker's per-line error path to enumerate
// the accepted types.
func KnownStdinTypes() []string {
	out := make([]string, 0, len(Registry))
	for _, d := range Registry {
		out = append(out, d.StdinType)
	}
	return out
}
