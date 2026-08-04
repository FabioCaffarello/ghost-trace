// Package api is the HTTP transport adapter for the endpoints of
// docs/architecture.md §3.
//
// Handlers are deliberately thin: decode the wire shape, authenticate,
// call the use case, encode the result. Orchestration lives in
// internal/app; if a handler grows an if-statement about the domain,
// it is in the wrong layer.
//
// Trust boundary, per contract §1: everything arriving from the browser
// is hostile. `subject_id` and `action` are accepted only from the
// application server authenticated with secret_key, and never read from
// a browser request — a session token correlates telemetry, it does not
// authenticate anyone.
package api

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/app"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/session"
)

// Config holds the transport-level configuration: the tenant's key
// pair and the collect policy served to the SDK. Application concerns
// (tenant id, operating mode) live in app.Config.
type Config struct {
	SiteKey   string
	SecretKey string

	// CollectPolicy is served to the SDK at session start and is
	// remotely tunable without shipping a new SDK (contract §3).
	CollectPolicy CollectPolicy

	// SessionTTL is served to the SDK as expires_in. The composition
	// root must pass the same value it gives the session store: a
	// hardcoded number here once told every browser its token lived 30
	// minutes while the store expired it on the -session-ttl flag.
	SessionTTL time.Duration
}

// CollectPolicy is the server-driven collection configuration.
type CollectPolicy struct {
	PointerHz int      `json:"pointer_hz"`
	BatchMs   int      `json:"batch_ms"`
	Types     []string `json:"types"`
}

// Server adapts HTTP to the application layer.
type Server struct {
	cfg Config
	app *app.App
	log *slog.Logger

	// maxBody caps request bodies. Telemetry batches are the only
	// large payload and a 2s batch at 20Hz is a few KB; anything
	// approaching this cap is an attack, not a client.
	maxBody int64
}

// New constructs the HTTP adapter over a.
func New(cfg Config, a *app.App, log *slog.Logger) *Server {
	if log == nil {
		log = slog.Default()
	}
	return &Server{cfg: cfg, app: a, log: log, maxBody: 1 << 20}
}

// Routes returns the API mux.
func (s *Server) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/sessions", s.handleSessions)
	mux.HandleFunc("POST /v1/telemetry", s.handleTelemetry)
	mux.HandleFunc("POST /v1/decisions", s.handleDecisions)
	mux.HandleFunc("POST /v1/outcomes", s.handleOutcomes)
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

	out, err := s.app.StartSession(r.Context(), app.StartSessionInput{
		PagePath: req.Page.Path,
		Client:   c,
	})
	if err != nil {
		s.log.Error("session create failed", "err", err)
		writeError(w, http.StatusInternalServerError, "could not create session")
		return
	}

	writeJSON(w, http.StatusOK, sessionsResponse{
		SessionToken: out.Token,
		ExpiresIn:    int(s.cfg.SessionTTL.Seconds()),
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
	Type string `json:"type"`
	T    uint32 `json:"t"`

	// pointer
	Src string     `json:"src"`
	Pts [][3]int32 `json:"pts"`

	// key — timing and coarse class only, never content (§2, §6)
	Phase    string `json:"phase"`
	KeyClass string `json:"class"`
	Target   string `json:"target"`

	// scroll
	Dy   int32  `json:"dy"`
	Mode string `json:"mode"`

	// focus / visibility
	State string `json:"state"`

	// form
	Action string `json:"action"`
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

	in := app.TelemetryEnvelope{
		SessionToken: token,
		Seq:          env.Seq,
		SentAtMs:     env.SentAtMs,
		PagePath:     env.Page.Path,
		Events:       make([]app.TelemetryEvent, len(env.Events)),
	}
	if len(env.Page.Viewport) == 2 {
		in.Viewport = [2]int{env.Page.Viewport[0], env.Page.Viewport[1]}
	}
	for i, ev := range env.Events {
		in.Events[i] = app.TelemetryEvent{
			Type: ev.Type, T: ev.T,
			Src: ev.Src, Pts: ev.Pts,
			Phase: ev.Phase, KeyClass: ev.KeyClass, Target: ev.Target,
			Dy: ev.Dy, Mode: ev.Mode,
			State:  ev.State,
			Action: ev.Action,
		}
	}

	if err := s.app.IngestTelemetry(r.Context(), in); err != nil {
		// An unknown token is not an error worth surfacing in detail.
		// Telemetry is fire-and-forget and loss is expected (§5), so a
		// 202 here keeps a stale SDK from retrying in a loop.
		if errors.Is(err, app.ErrSessionNotFound) {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		writeError(w, http.StatusInternalServerError, "telemetry failed")
		return
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
	EvaluationID   string         `json:"evaluation_id"`
	Decision       string         `json:"decision"`
	ShadowDecision string         `json:"shadow_decision,omitempty"`
	Score          float64        `json:"score"`
	Confidence     float64        `json:"confidence"`
	Reasons        []policyReason `json:"reasons"`
	Evidence       evidence       `json:"evidence"`
	Mode           string         `json:"mode"`
}

type policyReason struct {
	Code   string  `json:"code"`
	Weight float64 `json:"weight"`
}

type evidence struct {
	Events     uint32 `json:"events"`
	DurationMs uint32 `json:"duration_ms"`
}

func (s *Server) handleDecisions(w http.ResponseWriter, r *http.Request) {
	// secret_key authenticates the application server. This is the only
	// endpoint that accepts subject_id and action, and it is the reason
	// they are never read from a browser.
	if !s.authorizedSecret(r) {
		writeError(w, http.StatusUnauthorized, "invalid secret_key")
		return
	}

	var req decisionsRequest
	if !s.decode(w, r, &req) {
		return
	}

	out, err := s.app.Decide(r.Context(), app.DecideInput{
		SessionToken: req.SessionToken,
		Action:       req.Action,
		SubjectID:    req.SubjectID,
	})
	if err != nil {
		s.writeAppError(w, err, "decision failed")
		return
	}

	resp := decisionsResponse{
		EvaluationID:   out.EvaluationID,
		Decision:       out.Decision,
		ShadowDecision: out.ShadowDecision,
		Score:          round3(out.Score),
		Confidence:     round3(out.Confidence),
		Reasons:        make([]policyReason, 0, len(out.Reasons)),
		Evidence:       evidence{Events: out.EvidenceEvents, DurationMs: out.EvidenceMs},
		Mode:           out.Mode,
	}
	for _, rs := range out.Reasons {
		resp.Reasons = append(resp.Reasons, policyReason{Code: rs.Code, Weight: rs.Weight})
	}
	writeJSON(w, http.StatusOK, resp)
}

// ---------------------------------------------------------------
// POST /v1/outcomes — application server
// ---------------------------------------------------------------

type outcomesRequest struct {
	EvaluationID string `json:"evaluation_id"`
	Outcome      string `json:"outcome"`
	ObservedAt   string `json:"observed_at"`
}

func (s *Server) handleOutcomes(w http.ResponseWriter, r *http.Request) {
	if !s.authorizedSecret(r) {
		writeError(w, http.StatusUnauthorized, "invalid secret_key")
		return
	}

	var req outcomesRequest
	if !s.decode(w, r, &req) {
		return
	}

	in := app.RecordOutcomeInput{
		EvaluationID: req.EvaluationID,
		Outcome:      req.Outcome,
	}
	if req.ObservedAt != "" {
		t, err := time.Parse(time.RFC3339, req.ObservedAt)
		if err != nil {
			// Reject rather than fall back to the server clock:
			// observed_at is the application's claim and recorded_at the
			// server's observation, and the gap between them is itself a
			// signal. Substituting "now" on a parse failure collapses
			// that gap to zero and silently corrupts the labels channel.
			writeError(w, http.StatusBadRequest, "observed_at must be RFC 3339")
			return
		}
		in.ObservedAt = t
	}

	if err := s.app.RecordOutcome(r.Context(), in); err != nil {
		s.writeAppError(w, err, "could not record outcome")
		return
	}

	w.WriteHeader(http.StatusAccepted)
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

// authorizedSecret reports whether the request carries the secret_key.
// The comparison is constant-time: the secret authenticates the
// application server on an internet-facing endpoint, and a byte-wise
// == would leak prefix length through response timing.
func (s *Server) authorizedSecret(r *http.Request) bool {
	return subtle.ConstantTimeCompare([]byte(bearerToken(r)), []byte(s.cfg.SecretKey)) == 1
}

// writeAppError is the single translation from the application's error
// vocabulary to HTTP. Handlers pass a fallback message for errors the
// table does not name; per-endpoint improvisation is what this exists
// to prevent.
func (s *Server) writeAppError(w http.ResponseWriter, err error, fallback string) {
	switch {
	case errors.Is(err, app.ErrActionRequired):
		writeError(w, http.StatusBadRequest, "action is required")
	case errors.Is(err, app.ErrEvaluationIDRequired):
		writeError(w, http.StatusBadRequest, "evaluation_id is required")
	case errors.Is(err, app.ErrUnknownOutcome):
		writeError(w, http.StatusBadRequest, "unknown outcome")
	case errors.Is(err, app.ErrArchiveUnavailable):
		// A label with nowhere durable to live is worse than a refusal:
		// the caller would believe it had reported an outcome.
		writeError(w, http.StatusServiceUnavailable, "outcome storage not configured")
	default:
		s.log.Error(fallback, "err", err)
		writeError(w, http.StatusInternalServerError, fallback)
	}
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
