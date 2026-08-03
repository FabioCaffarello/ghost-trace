// Package web serves the demo page, the SDK, and a stand-in host
// application.
//
// The demo backend calls /v1/decisions over HTTP with the secret_key
// exactly as a real integrator would, rather than reaching into the
// policy package in-process. That is the point of it: it exercises the
// real trust boundary (subject_id and action supplied server-side, never
// by the browser) and the real failure behaviour (client timeout, then
// fail-open) instead of a shortcut that would demonstrate neither.
package web

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

//go:embed static/*
var static embed.FS

// Config holds what the demo host application needs to call the API.
type Config struct {
	// APIBase is the Ghost Trace origin, e.g. http://127.0.0.1:8080.
	APIBase string

	// SiteKey is public and substituted into the page.
	SiteKey string

	// SecretKey authenticates the server-to-server decision call and is
	// never sent to the browser.
	SecretKey string

	// DecisionTimeout is the client-side budget. Contract §5 sets it at
	// roughly 3× the p99 target; at an 80ms p99 that is 250ms.
	DecisionTimeout time.Duration
}

// Handler serves the demo.
type Handler struct {
	cfg  Config
	log  *slog.Logger
	page []byte
	sdk  []byte
	http *http.Client
}

// New constructs the demo handler.
func New(cfg Config, log *slog.Logger) (*Handler, error) {
	if log == nil {
		log = slog.Default()
	}
	if cfg.DecisionTimeout == 0 {
		cfg.DecisionTimeout = 250 * time.Millisecond
	}

	page, err := static.ReadFile("static/index.html")
	if err != nil {
		return nil, err
	}
	sdk, err := static.ReadFile("static/sdk.js")
	if err != nil {
		return nil, err
	}

	return &Handler{
		cfg:  cfg,
		log:  log,
		page: []byte(strings.Replace(string(page), "SITE_KEY_PLACEHOLDER", cfg.SiteKey, 1)),
		sdk:  sdk,
		http: &http.Client{Timeout: cfg.DecisionTimeout},
	}, nil
}

// Register mounts the demo routes on mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /", h.servePage)
	mux.HandleFunc("GET /sdk.js", h.serveSDK)
	mux.HandleFunc("POST /demo/login", h.login)
}

func (h *Handler) servePage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(h.page)
}

func (h *Handler) serveSDK(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(h.sdk)
}

type loginRequest struct {
	SessionToken string `json:"session_token"`
	Username     string `json:"username"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&req); err != nil {
		http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
		return
	}

	// subject_id is asserted by the application, not read from the
	// browser. Ghost Trace cannot independently verify the binding
	// between a browser session and an account — it exists because the
	// application says so (§1).
	body, _ := json.Marshal(map[string]any{
		"session_token": req.SessionToken,
		"action":        "login",
		"subject_id":    "user_" + req.Username,
		"context":       map[string]any{"attempt_n": 1},
	})

	decision, err := h.decide(r.Context(), body)
	if err != nil {
		// Fail-open is the default for every action (§5). A detector
		// that takes down a customer's login when it degrades is worse
		// than no detector.
		h.log.Warn("decision call failed; failing open", "err", err)
		decision = map[string]any{
			"evaluation_id": "",
			"decision":      "allow",
			"score":         0.0,
			"confidence":    0.0,
			"reasons":       []any{},
			"evidence":      map[string]any{"events": 0, "duration_ms": 0},
			"mode":          "fail-open",
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(decision)
}

func (h *Handler) decide(ctx context.Context, body []byte) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		h.cfg.APIBase+"/v1/decisions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+h.cfg.SecretKey)

	resp, err := h.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}
