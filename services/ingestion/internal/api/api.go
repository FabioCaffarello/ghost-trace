// Package api is the HTTP transport adapter for the endpoints of
// contract/architecture.md §3.
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
	"encoding/json"
	"errors"
	"github.com/FabioCaffarello/ghost-trace/libs/middleware"
	"github.com/FabioCaffarello/ghost-trace/libs/wire"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/app"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/sdk"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/session"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// Config holds the transport-level configuration for the endpoints this
// adapter serves: the public site key and the collect policy handed to
// the SDK. The secret key is NOT here — it authenticates /v1/decisions
// and /v1/outcomes, which libs/decision serves, and a credential kept
// where nothing reads it is a credential that outlives its purpose.
type Config struct {
	SiteKey string

	// AllowedOrigins are the page origins permitted to call the browser
	// endpoints cross-origin. Empty disables CORS entirely, which is
	// correct for a same-origin deployment and is the default: a
	// mechanism that defaults to permissive is not an allowlist.
	AllowedOrigins []string

	// CollectPolicy is served to the SDK at session start and is
	// remotely tunable without shipping a new SDK (contract §3).
	CollectPolicy wire.CollectPolicy

	// SessionTTL is served to the SDK as expires_in. The composition
	// root must pass the same value it gives the session store: a
	// hardcoded number here once told every browser its token lived 30
	// minutes while the store expired it on the -session-ttl flag.
	SessionTTL time.Duration
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

// Routes returns the mux carrying the browser-facing endpoints and the
// health probe.
//
// /v1/decisions and /v1/outcomes are NOT here: libs/decision serves
// them, and the composition root mounts it on this mux. They are the
// two endpoints a second service also serves, so a copy of them in this
// adapter would be a copy of the contract.
func (s *Server) Routes() *http.ServeMux {
	mux := http.NewServeMux()

	// CORS wraps these two and NOTHING else. They are the endpoints a
	// page calls; /v1/decisions and /v1/outcomes carry the secret key
	// and are server-to-server (§1), so allowing a browser to preflight
	// them would advertise a door whose key must never be in a browser.
	//
	// The OPTIONS routes are registered explicitly because Go's mux
	// matches on method: without them a preflight would get 405 and the
	// browser would refuse the real request, with nothing in any log
	// saying why.
	browser := middleware.CORS(s.cfg.AllowedOrigins)
	for path, h := range map[string]http.HandlerFunc{
		"POST /v1/sessions":  s.handleSessions,
		"POST /v1/telemetry": s.handleTelemetry,
	} {
		mux.Handle(path, browser(h))
	}
	for _, path := range []string{"OPTIONS /v1/sessions", "OPTIONS /v1/telemetry"} {
		mux.Handle(path, browser(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			// Reached only when CORS did not treat it as a preflight —
			// an OPTIONS with no Access-Control-Request-Method. Nothing
			// to allow, nothing to refuse.
			w.WriteHeader(http.StatusNoContent)
		})))
	}

	// Ghost Trace's own artefact, served from Ghost Trace's origin. A
	// classic script tag is not CORS-restricted, so a page anywhere can
	// load it.
	sdk.Register(mux)

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	return mux
}

// ---------------------------------------------------------------
// POST /v1/sessions — browser
// ---------------------------------------------------------------

func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	var req wire.SessionsRequest
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

	writeJSON(w, http.StatusOK, wire.SessionsResponse{
		SessionToken: out.Token,
		ExpiresIn:    int(s.cfg.SessionTTL.Seconds()),
		Collect:      s.cfg.CollectPolicy,
	})
}

// ---------------------------------------------------------------
// POST /v1/telemetry — browser
// ---------------------------------------------------------------

func (s *Server) handleTelemetry(w http.ResponseWriter, r *http.Request) {
	var env wire.TelemetryBatch
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
	writeJSON(w, status, wire.ErrorResponse{Error: msg})
}
