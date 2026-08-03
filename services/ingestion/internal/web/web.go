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
	"os"
	"strings"
	"sync"
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

	// CaptureLog, when set, is a JSONL file of labelled human sessions
	// for the M2 study. Empty disables capture.
	CaptureLog string
}

// Handler serves the demo.
type Handler struct {
	cfg  Config
	log  *slog.Logger
	page []byte
	sdk  []byte
	http *http.Client

	// captureMu serializes appends. Volunteers overlap in practice —
	// a link goes out to a group chat and several people open it at
	// once — and interleaved writes would corrupt the JSONL.
	captureMu sync.Mutex
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

	// Capture-study labels, supplied by the participant's link. They
	// travel through the contract's existing fields — subject_id and
	// context — rather than through any new API surface, because a
	// session's cohort is a property of the experiment, not of the
	// product. The engine must not know which population it is looking
	// at.
	Participant string `json:"participant"`
	Arm         string `json:"arm"`
	Condition   string `json:"condition"`
	Visit       int    `json:"visit"`
}

// captureRow is one labelled human session, appended to the capture log
// for harness/analyze.py.
type captureRow struct {
	Participant    string  `json:"participant"`
	Arm            string  `json:"arm"`
	Condition      string  `json:"condition"`
	Visit          int     `json:"visit"`
	EvaluationID   string  `json:"evaluation_id"`
	Decision       string  `json:"decision"`
	ShadowDecision string  `json:"shadow_decision"`
	Score          float64 `json:"score"`
	Confidence     float64 `json:"confidence"`
	Events         float64 `json:"events"`
	DurationMs     float64 `json:"duration_ms"`
	At             string  `json:"at"`
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
	//
	// During the capture study the participant code plays that role: it
	// is the pseudonymous identity the host application asserts, which
	// is exactly what subject_id is for.
	subject := "user_" + req.Username
	if req.Participant != "" {
		subject = req.Participant
	}
	body, _ := json.Marshal(map[string]any{
		"session_token": req.SessionToken,
		"action":        "login",
		"subject_id":    subject,
		"context": map[string]any{
			"attempt_n": 1,
			"arm":       req.Arm,
			"condition": req.Condition,
			"visit":     req.Visit,
		},
	})

	decision, err := h.decide(r.Context(), body)
	if err == nil && req.Participant != "" {
		h.appendCapture(req, decision)
	}
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

// appendCapture records one labelled human session.
//
// Failures are logged and swallowed: a volunteer's five minutes must not
// be wasted by a full disk, and a missing row is a missing observation
// rather than a corrupted one.
func (h *Handler) appendCapture(req loginRequest, d map[string]any) {
	if h.cfg.CaptureLog == "" {
		return
	}

	num := func(k string) float64 {
		if v, ok := d[k].(float64); ok {
			return v
		}
		return 0
	}
	str := func(k string) string {
		if v, ok := d[k].(string); ok {
			return v
		}
		return ""
	}
	evidence, _ := d["evidence"].(map[string]any)
	ev := func(k string) float64 {
		if evidence == nil {
			return 0
		}
		if v, ok := evidence[k].(float64); ok {
			return v
		}
		return 0
	}

	row := captureRow{
		Participant:    req.Participant,
		Arm:            req.Arm,
		Condition:      req.Condition,
		Visit:          req.Visit,
		EvaluationID:   str("evaluation_id"),
		Decision:       str("decision"),
		ShadowDecision: str("shadow_decision"),
		Score:          num("score"),
		Confidence:     num("confidence"),
		Events:         ev("events"),
		DurationMs:     ev("duration_ms"),
		At:             time.Now().UTC().Format(time.RFC3339),
	}

	line, err := json.Marshal(row)
	if err != nil {
		h.log.Error("capture marshal", "err", err)
		return
	}

	h.captureMu.Lock()
	defer h.captureMu.Unlock()

	f, err := os.OpenFile(h.cfg.CaptureLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		h.log.Error("capture open", "err", err, "path", h.cfg.CaptureLog)
		return
	}
	defer func() { _ = f.Close() }()
	if _, err := f.Write(append(line, '\n')); err != nil {
		h.log.Error("capture write", "err", err)
	}
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
