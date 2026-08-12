// Package decision serves the two server-to-server endpoints of
// contract/architecture.md §3 — POST /v1/decisions and POST /v1/outcomes
// — as one mountable unit: the use cases, the durable record mapping,
// and the HTTP handlers that carry them.
//
// WHY ALL OF IT, AND NOT JUST THE USE CASES. Two services serve these
// endpoints during the split: the collector, answering from the session
// state it holds in memory, and the decision engine, answering from the
// snapshots that state is published as. What must be identical between
// them is not merely the judgement — it is the status codes, the
// authentication, the rounding, and every field name on the way out.
// Half a contract shared and half reimplemented is the shape of
// divergence this repository has already paid for twice: libs/snapshot
// exists because two mappings written apart satisfied their own tests
// and disagreed with each other, and libs/wire exists because a server
// that tolerates unknown fields cannot fail when two definitions drift.
//
// So the only thing a host supplies is where the state comes from.
// Everything a caller can observe is decided here, once.
package decision

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/FabioCaffarello/ghost-trace/libs/archive"
	"github.com/FabioCaffarello/ghost-trace/libs/policy"
	"github.com/FabioCaffarello/ghost-trace/libs/tenant"
)

// Session is the state a decision is computed from, together with what
// the durable evaluation needs in order to be joinable later.
type Session struct {
	ID          string
	TenantID    string
	State       policy.State
	LastEventMs uint32
}

// Sessions is the port over that state, scoped to a TENANT.
//
// The tenant is a parameter rather than construction-time configuration
// because a decision request resolves it from the secret it presents,
// and an implementation must refuse a token belonging to someone else.
// Without that, a token from one tenant and another tenant's secret
// would produce a decision about a session the caller has no claim to —
// and both halves would look valid on their own.
//
// A miss is (Session{}, false, nil) and deliberately NOT an error: the
// caller is at a risk moment and needs an answer, and an unknown token
// is a cold start the confidence dimension already models (§7). An
// error means the lookup itself failed — a broken store, not a missing
// session — and those are worth surfacing because they are not
// something the policy can reason about.
type Sessions interface {
	Lookup(ctx context.Context, tenantID, token string) (Session, bool, error)
}

// Config is what the host decides: how decisions operate, and which
// tenants this process serves.
type Config struct {
	// Mode is monitor or enforce (§4).
	Mode string

	// Tenants resolves the presented secret_key to the tenant it speaks
	// for. These are the only endpoints that accept subject_id and
	// action, which is why neither is ever read from a browser request
	// — and now also why the CREDENTIAL decides who the caller is,
	// rather than a flag deciding it once for the whole process.
	Tenants *tenant.Registry
}

// Service serves the two endpoints against the ports it is given.
type Service struct {
	cfg      Config
	sessions Sessions
	archive  archive.Store
	loss     LossMeter
	now      func() time.Time
	log      *slog.Logger

	// maxBody caps request bodies. Neither of these endpoints carries a
	// large payload, so anything approaching this is an attack rather
	// than a client.
	maxBody int64
}

// New constructs the service. Pass nil for now to use time.Now, and
// archive.Null{} rather than nil for runs with no durable store.
func New(cfg Config, sessions Sessions, store archive.Store,
	now func() time.Time, log *slog.Logger) *Service {

	if now == nil {
		now = time.Now
	}
	if log == nil {
		log = slog.Default()
	}
	if store == nil {
		store = archive.Null{}
	}
	return &Service{cfg: cfg, sessions: sessions, archive: store,
		loss: NoLossMeter{}, now: now, log: log, maxBody: 1 << 20}
}

// WithLossMeter returns a copy counting into meter.
//
// A copy rather than a setter, and an option rather than a constructor
// parameter: New already takes five arguments and has eight call sites,
// six of them tests that do not care. A host that wants the counters
// asks for them; one that does not gets NoLossMeter and no series,
// which is honest — a process with no registry is not reporting zero
// drops, it is reporting nothing.
func (s *Service) WithLossMeter(meter LossMeter) *Service {
	if meter == nil {
		return s
	}
	c := *s
	c.loss = meter
	return &c
}

// Mode reports the operating mode decisions run under.
func (s *Service) Mode() string { return s.cfg.Mode }

// BestEffortTimeout bounds how long a best-effort write may hold a
// decision open.
//
// The collector learned this and this package did not. "Best effort"
// was true of the OUTCOME and false of the LATENCY: with the broker
// down, a JetStream publish waits for a server ack that never comes and
// gives up after about five seconds — so /v1/decisions, which archives
// an evaluation on the way out, took five seconds to return a fail-open
// allow. The verdict was fail-open; the request was not, and this is
// the path with a caller at a risk moment and an 80ms budget.
//
// The same 250ms the collector uses (app.BestEffortTimeout), and the
// same accepted consequence: during an outage a bounded wait DROPS
// records an unbounded one would eventually have published. §5 permits
// the loss. What §5 does not permit is losing it silently, which is why
// the drop is counted rather than only logged.
//
// Outcomes do NOT come through here. They are the labels channel every
// future calibration depends on, they require durability, and they are
// allowed to fail honestly with a 503 instead.
const BestEffortTimeout = 250 * time.Millisecond

// Record kinds and drop reasons, as constants for the same reason the
// collector keeps its own: every value has to be declared to the meter
// before serving, and a literal at a call site is how a series nobody
// declared and nobody watches gets minted.
const (
	// KindEvaluation is a judgement archived off the decision path.
	KindEvaluation = "evaluation"

	// ReasonDeadline: the best-effort budget expired. The record was
	// not written and will not be retried.
	ReasonDeadline = "deadline"

	// ReasonError: the store refused or failed for some other reason.
	ReasonError = "error"
)

// Kinds and Reasons are every value that can appear here, so a
// composition root can declare the whole cross product before serving.
var (
	Kinds   = []string{KindEvaluation}
	Reasons = []string{ReasonDeadline, ReasonError}
)

// LossMeter counts what this service hands to a store, and what it
// fails to. Defined here rather than imported so the port belongs to
// its consumer; metrics.Loss satisfies it structurally.
type LossMeter interface {
	// Written: the record reached the store.
	Written(kind string)

	// Dropped: the record did not, and is gone.
	Dropped(kind, reason string)
}

// NoLossMeter counts nothing, for tests and for hosts with no registry.
type NoLossMeter struct{}

// Written discards the count.
func (NoLossMeter) Written(string) {}

// Dropped discards the count.
func (NoLossMeter) Dropped(string, string) {}

// archiveBestEffort appends a record, counting and logging failures
// instead of surfacing them. Archival off the decision path must never
// take a user-facing request down with it — not by failing it, and not
// by holding it open.
//
// Every outcome lands in exactly one place: written, dropped with a
// reason, or neither because there is no archive configured. The third
// is not a loss and is not counted.
func (s *Service) archiveBestEffort(ctx context.Context, msg proto.Message,
	eventTime int64, what string, args ...any) {

	ctx, cancel := context.WithTimeout(ctx, BestEffortTimeout)
	defer cancel()

	err := s.archive.Append(ctx, msg, eventTime)
	switch {
	case err == nil:
		s.loss.Written(what)
	case errors.Is(err, archive.ErrUnavailable):
		// No store to lose it from. Not written, not dropped.
	default:
		s.loss.Dropped(what, reasonFor(err))
		s.log.Error("archive "+what, append([]any{"err", err}, args...)...)
	}
}

// reasonFor separates the budget expiring from everything else. A
// deadline says the dependency was too slow and the bound did its job;
// an error says it refused. Bucketing both as "failed" would hide which
// of the two a deployment is actually suffering.
func reasonFor(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return ReasonDeadline
	}
	return ReasonError
}
