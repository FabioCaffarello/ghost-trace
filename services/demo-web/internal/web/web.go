// Package web serves the demo page, the SDK, and a stand-in host
// application.
//
// It is its own service because it is the only thing here that is not
// Ghost Trace: it stands in for a CUSTOMER, and a customer runs on
// their own origin. Keeping it inside the collector made the demo
// same-origin with the API, which quietly meant the browser endpoints
// never had to answer a cross-origin request — the one thing every real
// integration does.
//
// The demo backend calls /v1/decisions over HTTP with the secret_key
// exactly as a real integrator would, rather than reaching into the
// policy package in-process. That is the point of it: it exercises the
// real trust boundary (subject_id and action supplied server-side, never
// by the browser) and the real failure behaviour (client timeout, then
// fail-open) instead of a shortcut that would demonstrate neither.
package web

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/FabioCaffarello/ghost-trace/libs/wire"
)

//go:embed static/*
var static embed.FS

// Config holds what the demo handler itself needs. Everything about
// HOW to reach the engine or WHERE capture rows go lives behind the
// ports below — the handler no longer holds the secret key at all, so
// it cannot leak what it does not have.
type Config struct {
	// SiteKey is public and substituted into the page.
	SiteKey string

	// APIBase is the collector's origin, substituted into the page so
	// the SDK knows where to send sessions and telemetry. Empty means
	// same-origin, which no longer describes any deployment here but is
	// the SDK's own default and stays the honest zero value.
	APIBase string
}

// DecisionClient is the handler's port to the decision engine. The
// production implementation (HTTPDecisionClient) speaks HTTP with the
// secret key exactly as a real integrator would — that trust-boundary
// dogfooding is the demo's whole point — while tests substitute fakes
// without a network.
//
// Report is the other half of that loop, and until this port had it,
// `POST /v1/outcomes` — one of the contract's four endpoints — had no
// caller anywhere in the product. Only `deploy/kill-test.py` and
// `deploy/loss-audit.py` ever exercised it, which meant the labels
// channel every future calibration depends on was reachable exclusively
// from the gates that check it.
type DecisionClient interface {
	Decide(ctx context.Context, body []byte) (map[string]any, error)
	Report(ctx context.Context, body []byte) error
}

// CaptureSink records one labelled human session for the capture
// study. FileCaptureSink appends JSONL; NoCapture discards.
type CaptureSink interface {
	Append(row CaptureRow) error
}

// Handler serves the demo.
type Handler struct {
	cfg       Config
	log       *slog.Logger
	page      []byte
	decisions DecisionClient
	capture   CaptureSink
}

// New constructs the demo handler over its two ports. A nil capture
// means no capture study is running (NoCapture).
func New(cfg Config, decisions DecisionClient, capture CaptureSink, log *slog.Logger) (*Handler, error) {
	if log == nil {
		log = slog.Default()
	}
	if decisions == nil {
		return nil, errors.New("web: DecisionClient is required")
	}
	if capture == nil {
		capture = NoCapture{}
	}

	page, err := static.ReadFile("static/index.html")
	if err != nil {
		return nil, err
	}
	return &Handler{
		cfg: cfg,
		log: log,
		page: []byte(strings.NewReplacer(
			"SITE_KEY_PLACEHOLDER", cfg.SiteKey,
			"API_BASE_PLACEHOLDER", cfg.APIBase,
		).Replace(string(page))),
		decisions: decisions,
		capture:   capture,
	}, nil
}

// Register mounts the demo routes on mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /", h.servePage)
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

type loginRequest struct {
	SessionToken string `json:"session_token"`
	Username     string `json:"username"`

	// Capture-study labels, supplied by the participant's link. Only
	// `participant` crosses the wire, as subject_id — the pseudonymous
	// identity the host application asserts, which is exactly what
	// subject_id is for. Arm, condition and visit stay HERE: they are
	// written into the local capture row and never sent, because the
	// engine must not know which population it is looking at. (An older
	// version of this comment claimed they travelled through a request
	// `context` field; that field was removed from the contract at
	// R1.14 and this handler kept posting it into the void.)
	Participant string `json:"participant"`
	Arm         string `json:"arm"`
	Condition   string `json:"condition"`
	Visit       int    `json:"visit"`
}

// CaptureRow is one labelled human session, appended to the capture
// log for experiments/analyze.py. Exported because it is the value the
// CaptureSink port carries.
type CaptureRow struct {
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
	// 64KiB, deliberately far below the API's 1MiB: a telemetry batch
	// carries event payloads, a demo login form carries two short
	// strings. The two limits differ because the payloads do.
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&req); err != nil {
		http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
		return
	}

	// subject_id is asserted by the application, not read from the
	// browser. Ghost Trace cannot independently verify the binding
	// between a browser session and an account — it exists because the
	// application says so (§1).
	//
	// During the capture study it asserts NOTHING. The participant code
	// used to travel here, which put a pseudonym for a real person into
	// the append-only archive permanently — and ADR-0006 makes that
	// archive the durable record, so "delete my data" would have meant
	// deleting from a store whose whole guarantee is that nothing is
	// removed. RFC-0001 promises deletion on request; ADR-0014 keeps the
	// promise by never writing the identity in the first place.
	//
	// Nothing is lost by omitting it. The engine copies subject_id into
	// the Evaluation record and decides nothing with it, and the study's
	// join key is `evaluation_id`, which the capture row already carries.
	// Delete the row and the link is gone — which is what deletion means
	// when the remaining record can no longer be attributed to anyone.
	subject := "user_" + req.Username
	if req.Participant != "" {
		subject = ""
	}
	// Built from the wire type the engine decodes, not a hand-rolled
	// map. The map this replaces still carried the `context` object
	// removed from the contract at R1.14 — accepted, dropped, and
	// promised by a comment above to be the capture channel it never
	// was. The server tolerating unknown fields means nothing but a
	// shared type stops that from happening again.
	body, _ := json.Marshal(wire.DecisionsRequest{
		SessionToken: req.SessionToken,
		Action:       "login",
		SubjectID:    subject,
	})

	decision, err := h.decisions.Decide(r.Context(), body)
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

	// The loop closes here: the application tells Ghost Trace what its
	// own action turned out to be. `outcome_reported` is added to the
	// body the page renders, so an operator can see the label that was
	// filed rather than infer it.
	if label := h.reportOutcome(r.Context(), decision); label != "" {
		decision["outcome_reported"] = label
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(decision)
}

// reportOutcome files what the application DID, and returns the label it
// filed — empty when it filed nothing.
//
// WHICH LABEL, and why two of the seven are unreachable from here. §3's
// enumeration covers the whole life of a case; this handler knows only
// the part that has already happened:
//
//   - `allow` → the application signed the user in: login_success.
//   - `block` → it refused: login_failure.
//   - `challenge` → IT IS NOT FINISHED. The demo shows no challenge, so
//     there is no challenge_passed or challenge_failed to file, and
//     filing login_success because the request returned would be a lie
//     about a session that was never let in. `libs/decision` rejects an
//     unknown label rather than storing it, on the grounds that a wrong
//     one silently degrades every future calibration; a wrong one that
//     IS in the enum is the same damage with nothing to catch it.
//
// `fraud_confirmed` and `user_appealed` arrive days later, from a
// chargeback or a support ticket. They are the reason the labels channel
// is a separate endpoint rather than a field on the decision response,
// and nothing in a login handler can produce them.
//
// SYNCHRONOUS, inside the login. A real integrator at volume would queue
// this; the demo files it before answering because a demo whose loop
// closes somewhere off-screen demonstrates nothing, and because a
// goroutine per login is the unbounded shape this repository has spent
// two phases removing. It costs one server-to-server call inside the
// same 250ms budget as the decision.
//
// A FAILURE HERE DOES NOT FAIL THE LOGIN. The user is already signed in;
// refusing them because a label could not be filed would be the
// detector-takes-down-the-login failure §5 exists to prevent. It is
// logged, and the absent `outcome_reported` in the response is what says
// so — the page cannot then show a label that was never stored.
func (h *Handler) reportOutcome(ctx context.Context, d map[string]any) string {
	id, _ := d["evaluation_id"].(string)
	if id == "" {
		// Fail-open produced this verdict; there is no evaluation on
		// the other side to label.
		return ""
	}

	var label string
	switch v, _ := d["decision"].(string); v {
	case "allow":
		label = "login_success"
	case "block":
		label = "login_failure"
	default:
		return ""
	}

	body, _ := json.Marshal(wire.OutcomesRequest{EvaluationID: id, Outcome: label})
	if err := h.decisions.Report(ctx, body); err != nil {
		h.log.Warn("outcome not filed; the login stands", "err", err, "evaluation_id", id)
		return ""
	}
	return label
}

// appendCapture records one labelled human session through the sink.
//
// Failures are logged and swallowed: a volunteer's five minutes must not
// be wasted by a full disk, and a missing row is a missing observation
// rather than a corrupted one.
func (h *Handler) appendCapture(req loginRequest, d map[string]any) {
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

	row := CaptureRow{
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

	if err := h.capture.Append(row); err != nil {
		h.log.Error("capture append", "err", err)
	}
}
