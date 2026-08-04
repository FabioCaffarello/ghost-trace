// Package substratearchive adapts the substrate-backed ingester to the
// application's EventArchive port.
//
// The port hides everything below it: callers cannot reach the blob
// store, the SQLite handle, or the ingest report — the archive is an
// append of canonical bytes and nothing else.
package substratearchive

import (
	"context"

	"google.golang.org/protobuf/proto"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
)

// Archive implements app.EventArchive over an ingest.Ingester.
type Archive struct {
	ing *ingest.Ingester
}

// New wraps ing.
func New(ing *ingest.Ingester) *Archive {
	return &Archive{ing: ing}
}

// Append canonicalizes and commits msg. Idempotent on retry by content
// hash.
func (a *Archive) Append(ctx context.Context, msg proto.Message, eventTime int64) error {
	_, err := a.ing.Append(ctx, msg, eventTime)
	return err
}
