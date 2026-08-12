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

	"github.com/FabioCaffarello/ghost-trace/services/collector/internal/session"
)

// SessionRepository is the port over per-session state.
//
// Access is serialized behind With because the feature accumulators are
// not safe for concurrent mutation; implementations own the locking
// strategy, callers own keeping the critical section small and copying
// out by value (the pointer must not escape fn).
//
// That rule used to be stated here and broken on the next line: Create
// returned the *State it had just put in the map. `go test -race`
// confirms it — a goroutine writing through the returned pointer and one
// writing through With report a data race on the same session. Nothing
// in production hit it, because the token has not reached the browser
// when Create returns, so no telemetry can arrive for that session yet.
// A rule that holds because of timing elsewhere is not a rule, so Create
// now returns a value.
type SessionRepository interface {
	Create(tenantID, pagePath string, c session.Client) (token string, ident session.Identity, err error)
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

// ErrNoSnapshotStore reports that the run publishes no snapshots.
//
// It exists so that "there is no store" is distinguishable from "the
// write succeeded". NullSnapshots used to return nil, which meant a run
// with snapshots disabled counted every one of them as WRITTEN — a
// counter reporting success for something that went nowhere, which is
// the failure this whole phase is about. It mirrors
// archive.ErrUnavailable exactly, and for the same reason.
var ErrNoSnapshotStore = errors.New("app: no session snapshot store configured")

// NullSnapshots is the do-nothing store: the all-in-one binary needs no
// snapshots because nothing else reads its sessions.
type NullSnapshots struct{}

// Put reports ErrNoSnapshotStore rather than success.
func (NullSnapshots) Put(context.Context, string, *eventsv1.SessionSnapshot) error {
	return ErrNoSnapshotStore
}

// Record kinds and drop reasons. Constants rather than string literals
// at the call sites, because every one of these values has to be
// declared to the meter at startup — see LossMeter — and a typo would
// mint a series nobody declared and nobody is watching.
const (
	KindSessionStart = "session_start"
	KindTelemetry    = "telemetry"
	KindSnapshot     = "snapshot"

	// ReasonDeadline: the best-effort budget expired. The record was
	// not written and will not be retried.
	ReasonDeadline = "deadline"

	// ReasonError: the store refused or failed for some other reason.
	ReasonError = "error"

	// ReasonShed: the archive is close enough to losing its backlog
	// that this record was not offered to it. Deliberate, counted, and
	// the only drop reason here that describes a CHOICE rather than a
	// failure.
	ReasonShed = "shed"
)

// Kinds and Reasons are every value that can appear, so a composition
// root can declare the whole cross product before serving anything.
var (
	Kinds   = []string{KindSessionStart, KindTelemetry, KindSnapshot}
	Reasons = []string{ReasonDeadline, ReasonError, ReasonShed}
)

// ArchivePressure reports how close the archive is to losing its
// backlog, as a fraction of the stream's retention window: 0 is empty,
// 1 means the oldest unread record is at the age limit and the next
// thing to happen is age-out.
//
// A PORT RATHER THAN A NUMBER because the collector must work without
// it. A deployment with no broker has no archive to be behind, and one
// whose watcher cannot reach the broker knows nothing — which is not
// the same as knowing there is no pressure, and NoPressure says so by
// reporting a level nothing acts on.
type ArchivePressure interface {
	// Level is 0..1, or negative when there is no reading. Negative is
	// deliberately not zero: a poll that failed must not read as calm.
	Level() float64
}

// NoPressure is the absence of a signal, not a signal of absence.
type NoPressure struct{}

// Level reports -1: unknown, act on nothing.
func (NoPressure) Level() float64 { return -1 }

// SheddingThreshold is the pressure at which the collector stops
// offering records to the archive.
//
// 0.8 of the retention window. Below it the archive is behind but the
// backlog is still recoverable, and every record handed over is one the
// archive will get to. Above it the stream is about to discard its
// OLDEST records — the ones already accepted and waiting — so adding
// newer ones to the queue trades a record that would have been stored
// for one that will not be.
//
// SHEDDING THE NEW TO SAVE THE OLD is the whole argument. Both paths
// lose records; only one of them loses records nobody counted, at a
// moment nobody chose, from the end of the queue that has been waiting
// longest. ADR-0012 §Alternatives set this up; ADR-0013 records it,
// including what 0.8 is NOT — it is argued from the shape of the
// failure, not derived from measured headroom, because the run that
// would measure the headroom has not been taken.
const SheddingThreshold = 0.8

// LossMeter counts what the application hands to a store, and what it
// fails to.
//
// The phase this belongs to exists because three loss paths were logged
// per occurrence and counted nowhere. A log line answers "did this
// happen" for someone already looking; a counter answers "how much has
// happened" for someone who is not.
//
// ErrArchiveUnavailable is deliberately NOT a drop. A run configured
// with no archive is not losing records — it never had a store to lose
// them from, and counting that would make every development run look
// catastrophic while telling nobody anything. The distinction is the
// same one contract §7 draws: absent is not zero, and neither is it
// failure.
type LossMeter interface {
	// Written: the record reached the store.
	Written(kind string)

	// Dropped: the record did not, and is gone.
	Dropped(kind, reason string)
}

// NoLossMeter counts nothing, for tests and for the composition roots
// that have not been given a registry.
type NoLossMeter struct{}

// Written discards the count.
func (NoLossMeter) Written(string) {}

// Dropped discards the count.
func (NoLossMeter) Dropped(string, string) {}

// Config is the application-level configuration.
//
// It is empty: the tenant used to live here, and it now arrives with
// each request because one collector serves several. Credentials live
// with the transport adapter and the decision-side configuration with
// libs/decision, as they always did. Kept as a type so the composition
// root still reads as configuration being passed, and so the next thing
// that genuinely IS process-wide has somewhere to go.
type Config struct{}

// App wires the use cases to their ports.
type App struct {
	cfg       Config
	sessions  SessionRepository
	archive   EventArchive
	snapshots SessionSnapshots
	loss      LossMeter
	pressure  ArchivePressure
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
		snapshots: NullSnapshots{}, loss: NoLossMeter{}, pressure: NoPressure{},
		now: now, log: log}
}

// WithLossMeter returns a copy counting what it writes and drops.
// Optional rather than a constructor argument, for the same reason
// WithSnapshots is: a run without a registry must still work, and a
// required nil would be a worse API than an explicit opt-in.
func (a *App) WithLossMeter(m LossMeter) *App {
	if m == nil {
		m = NoLossMeter{}
	}
	b := *a
	b.loss = m
	return &b
}

// WithArchivePressure returns a copy that sheds when the archive is
// close to losing its backlog.
func (a *App) WithArchivePressure(p ArchivePressure) *App {
	if p == nil {
		p = NoPressure{}
	}
	b := *a
	b.pressure = p
	return &b
}

// WithSnapshots returns a copy publishing session snapshots to store.
func (a *App) WithSnapshots(store SessionSnapshots) *App {
	if store == nil {
		store = NullSnapshots{}
	}
	b := *a
	b.snapshots = store
	return &b
}

// BestEffortTimeout bounds how long a best-effort write may hold a
// user-facing request.
//
// It exists because "best effort" was true of the OUTCOME and false of
// the LATENCY. With the broker down, a JetStream publish waits for a
// server ack that never comes and gives up after five seconds — so
// /v1/telemetry, which does two of them, took ten seconds to return its
// 202. The status was fail-open; the request was not. At any real
// concurrency that exhausts the collector, which is precisely the
// failure mode the local write used to prevent and ADR-0006 retired.
//
// The consequence is accepted rather than hidden: during an outage a
// bounded wait DROPS records that an unbounded one would eventually
// have published. Contract §5 says telemetry loss is expected and the
// service is fail-open, and every drop is logged. Outcomes do not come
// through here — they require durability and are allowed to fail
// honestly instead.
const BestEffortTimeout = 250 * time.Millisecond

// bestEffort bounds ctx so a stalled dependency cannot hold a
// user-facing request open.
func bestEffort(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, BestEffortTimeout)
}

// archiveBestEffort appends a record, counting and logging failures
// instead of surfacing them. Archival off the decision path must never
// take a user-facing request down with it — not by failing it and not
// by holding it open.
//
// Every outcome lands in exactly one place: written, dropped with a
// reason, or neither because there is no archive configured. The third
// is not a loss and is not counted; see LossMeter.
func (a *App) archiveBestEffort(ctx context.Context, msg proto.Message, eventTime int64,
	kind string, args ...any) {

	// SHED BEFORE TRYING, when the archive is nearly out of runway.
	//
	// The bound below stops a slow store from holding a request open;
	// it does nothing about a store that is keeping up with the
	// REQUESTS and falling behind the STREAM. That is the standing
	// mismatch — the archive commits far slower than the collector
	// accepts — and its end state is the broker discarding the oldest
	// records to make room for these newest ones.
	//
	// So at high pressure this record is not offered. It is a drop
	// either way; this one is counted, attributed to a decision, and
	// spends the remaining runway on the backlog that has been waiting
	// longest rather than on a record that just arrived.
	if lvl := a.pressure.Level(); lvl >= SheddingThreshold {
		a.loss.Dropped(kind, ReasonShed)
		return
	}

	ctx, cancel := bestEffort(ctx)
	defer cancel()

	err := a.archive.Append(ctx, msg, eventTime)
	switch {
	case err == nil:
		a.loss.Written(kind)
	case errors.Is(err, archive.ErrUnavailable):
		// No store to lose it from. Not written, not dropped.
	default:
		a.loss.Dropped(kind, reasonFor(err))
		a.log.Error("archive "+kind, append([]any{"err", err}, args...)...)
	}
}

// reasonFor separates the budget expiring from everything else.
//
// The distinction is the one that matters when reading the counter: a
// deadline says the dependency was too slow and the bound did its job,
// while an error says it refused. Bucketing both as "failed" would hide
// which of the two a deployment is actually suffering.
func reasonFor(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return ReasonDeadline
	}
	return ReasonError
}
