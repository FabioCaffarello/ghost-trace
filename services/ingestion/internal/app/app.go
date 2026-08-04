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

	"github.com/FabioCaffarello/ghost-trace/libs/archive"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/libs/genproto/events/v1"

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

// EventArchive is libs/archive's port, aliased so this package's
// callers keep one name for it. Everything appended is a Category I
// record: canonical bytes, content-addressed, idempotent on retry.
type EventArchive = archive.Store

// NullArchive is the archive-less implementation, aliased for the same
// reason. It reports ErrArchiveUnavailable rather than succeeding.
type NullArchive = archive.Null

// SessionSnapshots publishes the state a decision is computed from, so
// a process that did not observe the session can still judge it.
//
// Best-effort by construction, and the reason is the same asymmetry
// ADR-0003 settled for the archive: this service already holds the
// authoritative state in memory and answers decisions from it. A
// snapshot store that is unreachable must slow nothing down and fail
// nothing — it makes another process's decisions stale, which is that
// process's problem to detect, not a reason to reject telemetry here.
//
// NullSnapshots implements it for runs with no snapshot store.
type SessionSnapshots interface {
	Put(ctx context.Context, token string, snap *eventsv1.SessionSnapshot) error
}

// NullSnapshots is the do-nothing store: the all-in-one binary needs no
// snapshots because nothing else reads its sessions.
type NullSnapshots struct{}

// Put discards the snapshot.
func (NullSnapshots) Put(context.Context, string, *eventsv1.SessionSnapshot) error { return nil }

// Config is the application-level configuration: who the tenant is.
// Transport credentials (site_key, secret_key) deliberately live with
// the transport adapter, and the decision-side configuration lives with
// libs/decision.
type Config struct {
	TenantID string
}

// App wires the use cases to their ports.
type App struct {
	cfg       Config
	sessions  SessionRepository
	archive   EventArchive
	snapshots SessionSnapshots
	now       func() time.Time
	log       *slog.Logger
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
	return &App{cfg: cfg, sessions: sessions, archive: archive,
		snapshots: NullSnapshots{}, now: now, log: log}
}

// WithSnapshots returns a copy publishing session snapshots to store.
// Optional rather than a constructor argument because the all-in-one
// binary genuinely has no use for it, and a required nil would be a
// worse API than an explicit opt-in.
func (a *App) WithSnapshots(store SessionSnapshots) *App {
	if store == nil {
		store = NullSnapshots{}
	}
	b := *a
	b.snapshots = store
	return &b
}

// archiveBestEffort appends a record, logging failures instead of
// surfacing them. Archival off the decision path must never take a
// user-facing request down with it; a missing archive is not a failure
// at these call sites, so ErrArchiveUnavailable is not even logged.
func (a *App) archiveBestEffort(ctx context.Context, msg proto.Message, eventTime int64, what string, args ...any) {
	if err := a.archive.Append(ctx, msg, eventTime); err != nil && !errors.Is(err, archive.ErrUnavailable) {
		a.log.Error("archive "+what, append([]any{"err", err}, args...)...)
	}
}
