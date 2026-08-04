// Package app holds the application layer: one use case per contract
// endpoint, expressed against ports instead of concrete adapters.
//
// The dependency rule is strict and pointed inward — domain packages
// (session, feature, policy) know nothing of this package, and this
// package knows nothing of HTTP. Transport adapters decode their wire
// format into the input types here and encode the outputs back; they
// contain no orchestration. That split is what the M0–M5 slice traded
// away for speed, and what Phase 2's decomposition needs back: a use
// case that does not know it is being served over HTTP can be served
// over anything.
package app

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/session"
)

// SessionRepository is the port over per-session state.
//
// Access is serialized behind With because the feature accumulators are
// not safe for concurrent mutation; implementations own the locking
// strategy, callers own keeping the critical section small and copying
// out by value (the pointer must not escape fn).
type SessionRepository interface {
	Create(tenantID, pagePath string, c session.Client) (token string, st *session.State, err error)
	With(token string, fn func(*session.State)) error
	Sweep() int
}

// EventArchive is the port over the durable, append-only record store.
//
// Everything appended is a Category I record: canonical bytes,
// content-addressed, idempotent on retry. NullArchive implements the
// port for archive-less runs and reports ErrArchiveUnavailable, which
// best-effort call sites ignore and durability-requiring call sites
// surface.
type EventArchive interface {
	Append(ctx context.Context, msg proto.Message, eventTime int64) error
}

// Config is the application-level configuration: who the tenant is and
// how decisions operate. Transport credentials (site_key, secret_key)
// deliberately live with the transport adapter, not here.
type Config struct {
	TenantID string

	// Mode is monitor or enforce (§4).
	Mode string
}

// App wires the use cases to their ports.
type App struct {
	cfg      Config
	sessions SessionRepository
	archive  EventArchive
	now      func() time.Time
	log      *slog.Logger
}

// New constructs the application. Pass nil for now to use time.Now.
// archive must not be nil — use NullArchive for archive-less runs.
func New(cfg Config, sessions SessionRepository, archive EventArchive, now func() time.Time, log *slog.Logger) *App {
	if now == nil {
		now = time.Now
	}
	if log == nil {
		log = slog.Default()
	}
	return &App{cfg: cfg, sessions: sessions, archive: archive, now: now, log: log}
}

// Mode reports the operating mode decisions run under.
func (a *App) Mode() string { return a.cfg.Mode }

// archiveBestEffort appends a record, logging failures instead of
// surfacing them. Archival off the decision path must never take a
// user-facing request down with it; a missing archive is not a failure
// at these call sites, so ErrArchiveUnavailable is not even logged.
func (a *App) archiveBestEffort(ctx context.Context, msg proto.Message, eventTime int64, what string, args ...any) {
	if err := a.archive.Append(ctx, msg, eventTime); err != nil && !errors.Is(err, ErrArchiveUnavailable) {
		a.log.Error("archive "+what, append([]any{"err", err}, args...)...)
	}
}
