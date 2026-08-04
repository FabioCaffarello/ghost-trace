package app

import (
	"context"

	"google.golang.org/protobuf/proto"
)

// NullArchive is the EventArchive for runs without durable storage.
//
// It replaces the `archive may be nil` convention and its guard at
// every call site: call sites always append, and the two policies —
// best-effort ignores ErrArchiveUnavailable, durability-requiring
// surfaces it — are chosen by the caller's semantics, not by
// remembering a nil check.
type NullArchive struct{}

// Append reports that nothing durable exists to append to.
func (NullArchive) Append(context.Context, proto.Message, int64) error {
	return ErrArchiveUnavailable
}
