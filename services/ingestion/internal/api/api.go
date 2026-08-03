// Package api implements the endpoints of integration-contract.md §3.
//
// M1 ships three of the four: sessions, telemetry, decisions. Outcomes
// is M4, because a label channel with nothing durable behind it stores
// labels that cannot be joined to anything.
//
// Trust boundary, per contract §1: everything arriving from the browser
// is hostile. `subject_id` and `action` are accepted only from the
// application server authenticated with secret_key, and never read from
// a browser request — a session token correlates telemetry, it does not
// authenticate anyone.
package api

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/feature"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/policy"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/session"
)

func randomEvaluationID() (string, error) {
	b := make([]byte, 18)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "ev_" + base64.RawURLEncoding.EncodeToString(b), nil
}

// Config holds the server's tenant credentials and operating mode.
//
// One tenant, supplied at startup. `tenant_id` is threaded through
// every record from day one because adding the column later is a
// migration of every store and index, while carrying it now costs
// nothing (REFACTOR-PLAN §8.4).
type Config struct {
	TenantID  string
	SiteKey   string
	SecretKey string

	// Mode is monitor or enforce. Every integration starts in monitor.
	Mode string

	// CollectPolicy is served to the SDK at session start and is
	// remotely tunable without shipping a new SDK (contract §3).
	CollectPolicy CollectPolicy
}

// CollectPolicy is the server-driven collection configuration.
type CollectPolicy struct {
	PointerHz int      `json:"pointer_hz"`
	BatchMs   int      `json:"batch_ms"`
	Types     []string `json:"types"`
}

// Server holds the API's dependencies.
type Server struct {
	cfg      Config
	sessions *session.Store
	archive  *ingest.Ingester
	now      func() time.Time
	log      *slog.Logger

	// maxBody caps request bodies. Telemetry batches are the only
	// large payload and a 2s batch at 20Hz is a few KB; anything
	// approaching this cap is an attack, not a client.
	maxBody int64
}

// New constructs a Server. archive may be nil, in which case raw
// observations are scored but not persisted.
func New(cfg Config, sessions *session.Store, archive *ingest.Ingester, now func() time.Time, log *slog.Logger) *Server {
	if now == nil {
		now = time.Now
	}
	if log == nil {
		log = slog.Default()
	}
	return &Server{
		cfg:      cfg,
		sessions: sessions,
		archive:  archive,
		now:      now,
		log:      log,
		maxBody:  1 << 20,
	}
}

// Routes returns the API mux.
func (s *Server) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/sessions", s.handleSessions)
	mux.HandleFunc("POST /v1/telemetry", s.handleTelemetry)
	mux.HandleFunc("POST /v1/decisions", s.handleDecisions)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	return mux
}

// ---------------------------------------------------------------
// POST /v1/sessions — browser
// ---------------------------------------------------------------

type sessionsRequest struct {
	SiteKey string `json:"site_key"`
	Page    struct {
		Path string `json:"path"`
	} `json:"page"`
	Client struct {
		Pointer       string `json:"pointer"`
		Touch         bool   `json:"touch"`
		Viewport      []int  `json:"viewport"`
		TZOffset      int    `json:"tz_offset"`
		ReducedMotion bool   `json:"reduced_motion"`
	} `json:"client"`
}

type sessionsResponse struct {
	SessionToken string        `json:"session_token"`
	ExpiresIn    int           `json:"expires_in"`
	Collect      CollectPolicy `json:"collect"`
}

func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	var req sessionsRequest
	if !s.decode(w, r, &req) {
		return
	}

	// site_key is public and embedded in the page; it identifies the
	// tenant, it does not authenticate. Checking it stops
	// cross-tenant noise, not an adversary.
	if req.SiteKey != s.cfg.SiteKey {
		writeError(w, http.StatusUnauthorized, "unknown site_key")
		return
	}

	c := session.Client{
		PointerType:   req.Client.Pointer,
		Touch:         req.Client.Touch,
		TZOffsetMin:   req.Client.TZOffset,
		ReducedMotion: req.Client.ReducedMotion,
	}
	if len(req.Client.Viewport) == 2 {
		c.Viewport = [2]int{req.Client.Viewport[0], req.Client.Viewport[1]}
	}

	token, st, err := s.sessions.Create(s.cfg.TenantID, req.Page.Path, c)
	if err != nil {
		s.log.Error("session create failed", "err", err)
		writeError(w, http.StatusInternalServerError, "could not create session")
		return
	}

	if s.archive != nil {
		start := &eventsv1.SessionStart{
			TenantId:     st.TenantID,
			SessionId:    st.ID,
			StartedAt:    st.StartedAt.UnixNano(),
			PagePath:     st.PagePath,
			PointerType:  c.PointerType,
			Touch:        c.Touch,
			ViewportW:    uint32(c.Viewport[0]),
			ViewportH:    uint32(c.Viewport[1]),
			TzOffsetMin:  int32(c.TZOffsetMin),
			ReducedMotion: c.ReducedMotion,
		}
		if _, err := s.archive.Append(r.Context(), start, st.StartedAt.UnixNano()); err != nil {
			// Archival is not on the critical path. A session that
			// cannot be archived can still be scored, and refusing to
			// issue a token here would take down the host's page load
			// over a storage problem.
			s.log.Error("archive session start", "err", err, "session_id", st.ID)
		}
	}

	writeJSON(w, http.StatusOK, sessionsResponse{
		SessionToken: token,
		ExpiresIn:    1800,
		Collect:      s.cfg.CollectPolicy,
	})
}

// ---------------------------------------------------------------
// POST /v1/telemetry — browser
// ---------------------------------------------------------------

type telemetryEnvelope struct {
	SessionToken string `json:"session_token"`
	Seq          uint32 `json:"seq"`
	SentAtMs     uint32 `json:"sent_at_ms"`
	Page         struct {
		Path     string `json:"path"`
		Viewport []int  `json:"viewport"`
	} `json:"page"`
	Events []telemetryEvent `json:"events"`
}

type telemetryEvent struct {
	Type string     `json:"type"`
	T    uint32     `json:"t"`
	Src  string     `json:"src"`
	Pts  [][3]int32 `json:"pts"`
}

func (s *Server) handleTelemetry(w http.ResponseWriter, r *http.Request) {
	var env telemetryEnvelope
	if !s.decode(w, r, &env) {
		return
	}

	token := bearerToken(r)
	if token == "" {
		token = env.SessionToken
	}

	batch := &eventsv1.TelemetryBatch{
		Seq:        env.Seq,
		SentAtMs:   env.SentAtMs,
		PagePath:   env.Page.Path,
		ReceivedAt: s.now().UnixNano(),
	}
	if len(env.Page.Viewport) == 2 {
		batch.ViewportW = uint32(env.Page.Viewport[0])
		batch.ViewportH = uint32(env.Page.Viewport[1])
	}

	err := s.sessions.With(token, func(st *session.State) {
		batch.TenantId = st.TenantID
		batch.SessionId = st.ID

		st.BatchesSeen++
		if env.Seq > st.HighestSeq {
			st.HighestSeq = env.Seq
		}
		if env.SentAtMs > st.LastEventMs {
			st.LastEventMs = env.SentAtMs
		}

		for _, ev := range env.Events {
			// M1 collects pointer only. Unknown types are dropped
			// silently rather than rejected: the collect policy is
			// server-driven and may change at any time, so an SDK
			// sending a type this build does not know is expected
			// behaviour, not a client error (contract §7).
			if ev.Type != "pointer" {
				continue
			}
			if ev.T > st.LastEventMs {
				st.LastEventMs = ev.T
			}

			pts := make([]feature.Point, 0, len(ev.Pts))
			pe := &eventsv1.PointerEvent{TMs: ev.T, Src: ev.Src}
			for _, p := range ev.Pts {
				pts = append(pts, feature.Point{X: p[0], Y: p[1], DtMs: uint32(p[2])})
				pe.Pts = append(pe.Pts, &eventsv1.PointerPoint{
					X: p[0], Y: p[1], DtMs: uint32(p[2]),
				})
			}
			st.Pointer.Add(ev.T, pts)
			batch.PointerEvents = append(batch.PointerEvents, pe)
		}
	})
	if err != nil {
		// An unknown token is not an error worth surfacing in detail.
		// Telemetry is fire-and-forget and loss is expected (§5), so a
		// 202 here keeps a stale SDK from retrying in a loop.
		if errors.Is(err, session.ErrNotFound) {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		writeError(w, http.StatusInternalServerError, "telemetry failed")
		return
	}

	if s.archive != nil && len(batch.PointerEvents) > 0 {
		if _, err := s.archive.Append(r.Context(), batch, batch.ReceivedAt); err != nil {
			s.log.Error("archive telemetry", "err", err, "session_id", batch.SessionId)
		}
	}

	w.WriteHeader(http.StatusAccepted)
}

// ---------------------------------------------------------------
// POST /v1/decisions — application server
// ---------------------------------------------------------------

type decisionsRequest struct {
	SessionToken string         `json:"session_token"`
	Action       string         `json:"action"`
	SubjectID    string         `json:"subject_id"`
	Context      map[string]any `json:"context"`
}

type decisionsResponse struct {
	EvaluationID   string          `json:"evaluation_id"`
	Decision       string          `json:"decision"`
	ShadowDecision string          `json:"shadow_decision,omitempty"`
	Score          float64         `json:"score"`
	Confidence     float64         `json:"confidence"`
	Reasons        []policy.Reason `json:"reasons"`
	Evidence       evidence        `json:"evidence"`
	Mode           string          `json:"mode"`
}

type evidence struct {
	Events     uint32 `json:"events"`
	DurationMs uint32 `json:"duration_ms"`
}

func (s *Server) handleDecisions(w http.ResponseWriter, r *http.Request) {
	// secret_key authenticates the application server. This is the only
	// endpoint that accepts subject_id and action, and it is the reason
	// they are never read from a browser.
	if bearerToken(r) != s.cfg.SecretKey {
		writeError(w, http.StatusUnauthorized, "invalid secret_key")
		return
	}

	var req decisionsRequest
	if !s.decode(w, r, &req) {
		return
	}
	if req.Action == "" {
		writeError(w, http.StatusBadRequest, "action is required")
		return
	}

	var (
		st    feature.PointerState
		sess  *session.State
		found = true
	)
	if err := s.sessions.With(req.SessionToken, func(s *session.State) {
		st = s.Pointer.State()
		sess = s
	}); err != nil {
		if !errors.Is(err, session.ErrNotFound) {
			writeError(w, http.StatusInternalServerError, "decision failed")
			return
		}
		// An unknown token yields a zero-evidence judgement rather than
		// an error. The caller is at a risk moment and needs an answer;
		// a missing session is a cold start, which the confidence
		// dimension already models correctly.
		found = false
	}

	j := policy.Judge(st)
	outcome, err := policy.Apply(j, s.cfg.Mode)
	if err != nil {
		s.log.Error("policy apply", "err", err)
		writeError(w, http.StatusInternalServerError, "decision failed")
		return
	}

	evalID, err := randomEvaluationID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "decision failed")
		return
	}

	ev := evidence{Events: st.Points}
	sessionID := ""
	tenantID := s.cfg.TenantID
	if found && sess != nil {
		ev.DurationMs = sess.LastEventMs
		sessionID = sess.ID
		tenantID = sess.TenantID
	}

	if s.archive != nil {
		rec := &eventsv1.Evaluation{
			TenantId:       tenantID,
			EvaluationId:   evalID,
			SessionId:      sessionID,
			Action:         req.Action,
			SubjectId:      req.SubjectID,
			DecidedAt:      s.now().UnixNano(),
			Decision:       outcome.Decision,
			ShadowDecision: outcome.Shadow,
			Mode:           s.cfg.Mode,
			Score:          float32(j.Score()),
			Confidence:     float32(j.Confidence()),
			EvidenceEvents: ev.Events,
			EvidenceDurationMs: ev.DurationMs,
			PolicyRef:      policy.Ref,
			FeatureSetRef:  feature.SetRef,
			Features: &eventsv1.FeatureState{
				PointerStraightness: float32(st.Straightness),
				PointerSegments:     st.Segments,
				PointerPathPx:       float32(st.PathPx),
				PointerPoints:       st.Points,
			},
		}
		for _, rs := range j.Reasons() {
			rec.Reasons = append(rec.Reasons, &eventsv1.Reason{
				Code: rs.Code, Weight: float32(rs.Weight),
			})
		}
		if _, err := s.archive.Append(r.Context(), rec, rec.DecidedAt); err != nil {
			s.log.Error("archive evaluation", "err", err, "evaluation_id", evalID)
		}
	}

	writeJSON(w, http.StatusOK, decisionsResponse{
		EvaluationID:   evalID,
		Decision:       outcome.Decision,
		ShadowDecision: outcome.Shadow,
		Score:          round3(j.Score()),
		Confidence:     round3(j.Confidence()),
		Reasons:        j.Reasons(),
		Evidence:       ev,
		Mode:           s.cfg.Mode,
	})
}

// ---------------------------------------------------------------
// helpers
// ---------------------------------------------------------------

func (s *Server) decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, s.maxBody)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "malformed JSON body")
		return false
	}
	return true
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if len(h) > 7 && strings.EqualFold(h[:7], "bearer ") {
		return h[7:]
	}
	return ""
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func round3(f float64) float64 {
	return float64(int(f*1000+0.5)) / 1000
}
