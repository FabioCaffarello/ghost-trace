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
)

// Session is the state a decision is computed from, together with what
// the durable evaluation needs in order to be joinable later.
type Session struct {
	ID          string
	TenantID    string
	State       policy.State
	LastEventMs uint32
}

// Sessions is the port over that state.
//
// A miss is (Session{}, false, nil) and deliberately NOT an error: the
// caller is at a risk moment and needs an answer, and an unknown token
// is a cold start the confidence dimension already models (§7). An
// error means the lookup itself failed — a broken store, not a missing
// session — and those are worth surfacing because they are not
// something the policy can reason about.
type Sessions interface {
	Lookup(ctx context.Context, token string) (Session, bool, error)
}

// Config is what the host decides: who it speaks for, how decisions
// operate, and the credential that authenticates the application
// server.
type Config struct {
	// TenantID is the fallback attribution for a session that was not
	// found. A found session carries its own.
	TenantID string

	// Mode is monitor or enforce (§4).
	Mode string

	// SecretKey authenticates the application server on both endpoints.
	// These are the only endpoints that accept subject_id and action,
	// which is why neither is ever read from a browser request.
	SecretKey string
}

// Service serves the two endpoints against the ports it is given.
type Service struct {
	cfg      Config
	sessions Sessions
	archive  archive.Store
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
		now: now, log: log, maxBody: 1 << 20}
}

// Mode reports the operating mode decisions run under.
func (s *Service) Mode() string { return s.cfg.Mode }

// archiveBestEffort appends a record, logging failures instead of
// surfacing them. Archival off the decision path must never take a
// user-facing request down with it; a missing archive is not a failure
// at these call sites, so ErrUnavailable is not even logged.
func (s *Service) archiveBestEffort(ctx context.Context, msg proto.Message,
	eventTime int64, what string, args ...any) {

	if err := s.archive.Append(ctx, msg, eventTime); err != nil &&
		!errors.Is(err, archive.ErrUnavailable) {
		s.log.Error("archive "+what, append([]any{"err", err}, args...)...)
	}
}
