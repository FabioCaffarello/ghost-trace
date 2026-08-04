// Package archive is the port over the durable, append-only record
// store, and the one sentinel error that says there isn't one.
//
// It is a module of its own because the error IDENTITY has to cross a
// service boundary. Outcomes refuse when the archive is unavailable —
// a label the caller believes recorded but which is not silently
// poisons calibration — and that refusal is expressed as
// errors.Is(err, ErrUnavailable). Two packages each declaring their own
// "archive unavailable" would compare unequal, and the refusal would
// quietly become a 500 in one service and a 503 in the other.
//
// Everything appended is a Category I record: canonical bytes,
// content-addressed, idempotent on retry. Implementations decide where
// the bytes land; callers decide whether a failure is fatal.
package archive

import (
	"context"
	"errors"

	"google.golang.org/protobuf/proto"
)

// ErrUnavailable reports that the run has no durable archive
// configured. Best-effort call sites ignore it; durability-requiring
// ones surface it.
var ErrUnavailable = errors.New("archive: event archive unavailable")

// Store appends a record, timestamped with the event's own time rather
// than the write's.
type Store interface {
	Append(ctx context.Context, msg proto.Message, eventTime int64) error
}

// Null is the archive-less implementation. It reports ErrUnavailable
// rather than succeeding, so a run without durable storage cannot be
// mistaken for one with it — the distinction the whole outcomes
// endpoint rests on.
type Null struct{}

// Append always reports ErrUnavailable.
func (Null) Append(context.Context, proto.Message, int64) error { return ErrUnavailable }
