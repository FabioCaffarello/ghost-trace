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

// CollectPolicy is the server-driven collection configuration. It is
// part of the response contract, not just configuration: the SDK obeys
// it, so changing it changes what browsers send.
type CollectPolicy struct {
	PointerHz int      `json:"pointer_hz" jsonschema:"description=Pointer sampling rate in hertz"`
	BatchMs   int      `json:"batch_ms" jsonschema:"description=How often the SDK flushes a batch — milliseconds"`
	Types     []string `json:"types" jsonschema:"description=Event families the SDK should collect"`
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

// SessionsRequest opens a session. Sent by the browser SDK, so
// everything in it is hostile input (§1).
type SessionsRequest struct {
	SiteKey string      `json:"site_key" jsonschema:"description=Public tenant key embedded in the page. Identifies the tenant; it does not authenticate."`
	Page    PageRef     `json:"page"`
	Client  ClientHints `json:"client"`
}

// PageRef locates the page the session belongs to.
type PageRef struct {
	Path string `json:"path" jsonschema:"description=Path of the page — no origin and no query string"`
}

// ClientHints are self-reported environment properties. They are
// context for interpreting interaction dynamics, never identity.
type ClientHints struct {
	Pointer       string `json:"pointer" jsonschema:"description=Primary pointing device the browser reports,enum=mouse,enum=touch,enum=pen,enum=none"`
	Touch         bool   `json:"touch" jsonschema:"description=Whether the device exposes a touchscreen"`
	Viewport      []int  `json:"viewport" jsonschema:"description=Viewport as [width height] in CSS pixels,minItems=2,maxItems=2"`
	TZOffset      int    `json:"tz_offset" jsonschema:"description=Timezone offset in minutes from UTC"`
	ReducedMotion bool   `json:"reduced_motion" jsonschema:"description=Whether prefers-reduced-motion is set. Load-bearing: it legitimately changes movement dynamics."`
}

// SessionsResponse carries the session token and the collection policy
// the SDK should apply.
type SessionsResponse struct {
	SessionToken string        `json:"session_token" jsonschema:"description=Opaque token correlating telemetry to this session. Not a credential and not an identity."`
	ExpiresIn    int           `json:"expires_in" jsonschema:"description=Seconds until the token stops being accepted"`
	Collect      CollectPolicy `json:"collect"`
}

func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	var req SessionsRequest
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

	writeJSON(w, http.StatusOK, SessionsResponse{
		SessionToken: out.Token,
		ExpiresIn:    int(s.cfg.SessionTTL.Seconds()),
		Collect:      s.cfg.CollectPolicy,
	})
}

// ---------------------------------------------------------------
// POST /v1/telemetry — browser
// ---------------------------------------------------------------

// TelemetryBatch is one flush from the browser SDK. Delivery is
// fire-and-forget: loss is expected and the service is fail-open (§5).
type TelemetryBatch struct {
	SessionToken string           `json:"session_token" jsonschema:"description=Session token. May instead travel as an Authorization: Bearer header; the header wins when both are present."`
	Seq          uint32           `json:"seq" jsonschema:"description=Monotonic batch counter within the session. Used to detect loss rather than to order."`
	SentAtMs     uint32           `json:"sent_at_ms" jsonschema:"description=Milliseconds since session start when the batch was flushed"`
	Page         TelemetryPage    `json:"page"`
	Events       []TelemetryEvent `json:"events" jsonschema:"description=The interaction events in this flush. Order is the order observed."`
}

// TelemetryPage is the page state at flush time.
type TelemetryPage struct {
	Path     string `json:"path" jsonschema:"description=Path of the page — no origin and no query string"`
	Viewport []int  `json:"viewport" jsonschema:"description=Viewport as [width height] in CSS pixels,minItems=2,maxItems=2"`
}

// TelemetryEvent is one interaction event.
//
// The shape is a FLAT UNION discriminated by `type`: every field is
// declared here, and which ones carry meaning depends on the type —
// `pts` for pointer, `phase`/`class`/`target` for key, `dy`/`mode` for
// scroll, `state` for focus and visibility, `action` for form. Fields
// belonging to other types are absent or zero.
//
// This is described rather than modelled as a JSON Schema `oneOf`
// because the decoder really is one flat struct; a schema with a
// per-type variant would be a SECOND definition of the wire, free to
// drift from the one the server actually parses.
type TelemetryEvent struct {
	Type string `json:"type" jsonschema:"description=Event family; selects which of the fields below apply,enum=pointer,enum=key,enum=scroll,enum=focus,enum=visibility,enum=form"`
	T    uint32 `json:"t" jsonschema:"description=Milliseconds since session start"`

	// pointer
	Src string     `json:"src" jsonschema:"description=pointer: input source that produced the samples"`
	Pts [][3]int32 `json:"pts" jsonschema:"description=pointer: samples as [x y t_offset_ms] triples"`

	// key — timing and coarse class only, never content (§2, §6)
	Phase    string `json:"phase" jsonschema:"description=key: keystroke phase,enum=down,enum=up"`
	KeyClass string `json:"class" jsonschema:"description=key: COARSE key class. Never the key itself — no keylogging (§2 and §6).,enum=char,enum=digit,enum=nav,enum=edit,enum=mod,enum=other"`
	Target   string `json:"target" jsonschema:"description=key or form: coarse field role — never the field value"`

	// scroll
	Dy   int32  `json:"dy" jsonschema:"description=scroll: vertical delta in CSS pixels"`
	Mode string `json:"mode" jsonschema:"description=scroll: scrolling mode reported by the browser"`

	// focus / visibility
	State string `json:"state" jsonschema:"description=focus or visibility: the state entered"`

	// form
	Action string `json:"action" jsonschema:"description=form: what happened to the field. The value 'injected' is the strongest single bot signal — a field value that appeared with no keystrokes behind it.,enum=focus,enum=blur,enum=input,enum=autofill,enum=injected,enum=submit"`
}

func (s *Server) handleTelemetry(w http.ResponseWriter, r *http.Request) {
	var env TelemetryBatch
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

// DecisionsRequest asks for a judgement. Sent by the APPLICATION
// SERVER with secret_key — this is the only endpoint that accepts
// subject_id and action, which is why they are never read from a
// browser (§1).
//
// There was a `context` object here until R1.14. It was accepted,
// never reached the use case, and never reached the archive: a field
// the contract appeared to offer and the service silently discarded.
// Removing it changes no behaviour (unknown JSON fields were already
// ignored) — it changes what is promised. When something consumes
// request context, it comes back with a consumer.
type DecisionsRequest struct {
	SessionToken string `json:"session_token" jsonschema:"description=Token returned by POST /v1/sessions"`
	Action       string `json:"action" jsonschema:"description=Action being judged; required. Free-form — for example login or checkout."`
	SubjectID    string `json:"subject_id" jsonschema:"description=Application's identifier for the actor. Never accepted from a browser."`
}

// DecisionsResponse is the judgement.
//
// Note score and confidence are SEPARATE: a low score with low
// confidence means "nothing suspicious observed yet", which is not the
// same claim as "this looks human" (§3).
type DecisionsResponse struct {
	EvaluationID   string           `json:"evaluation_id" jsonschema:"description=Identifier of this evaluation. Pass it to POST /v1/outcomes to label the decision later."`
	Decision       string           `json:"decision" jsonschema:"description=The decision actually returned. In monitor mode this is always allow.,enum=allow,enum=challenge,enum=block"`
	ShadowDecision string           `json:"shadow_decision,omitempty" jsonschema:"description=In monitor mode: what enforce mode WOULD have returned. Absent in enforce mode.,enum=allow,enum=challenge,enum=block"`
	Score          float64          `json:"score" jsonschema:"description=How bot-like the interaction looks. Rounded to three decimals.,minimum=0,maximum=1"`
	Confidence     float64          `json:"confidence" jsonschema:"description=How much evidence backs the score. Low confidence gates blocking (§3).,minimum=0,maximum=1"`
	Reasons        []DecisionReason `json:"reasons" jsonschema:"description=Contributing factors. Present so a decision can be explained and contested."`
	Evidence       DecisionEvidence `json:"evidence"`
	Mode           string           `json:"mode" jsonschema:"description=Operating mode of the service,enum=monitor,enum=enforce"`
}

// DecisionReason is one contributing factor.
type DecisionReason struct {
	// The enum is injected by cmd/gen-openapi from policy.ReasonCodes:
	// restating it here would be a second list free to drift from the
	// one the service actually emits.
	Code   string  `json:"code" jsonschema:"description=Stable reason code. Adding a code is non-breaking; changing what an existing code means is not (§7)."`
	Weight float64 `json:"weight" jsonschema:"description=Contribution of this factor to the score,minimum=0,maximum=1"`
}

// DecisionEvidence is how much was actually observed. It is the
// difference between "looks human" and "produced nothing to look at".
type DecisionEvidence struct {
	Events     uint32 `json:"events" jsonschema:"description=Interaction events observed in the session"`
	DurationMs uint32 `json:"duration_ms" jsonschema:"description=Wall time those events span — milliseconds"`
}

func (s *Server) handleDecisions(w http.ResponseWriter, r *http.Request) {
	// secret_key authenticates the application server. This is the only
	// endpoint that accepts subject_id and action, and it is the reason
	// they are never read from a browser.
	if !s.authorizedSecret(r) {
		writeError(w, http.StatusUnauthorized, "invalid secret_key")
		return
	}

	var req DecisionsRequest
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

	resp := DecisionsResponse{
		EvaluationID:   out.EvaluationID,
		Decision:       out.Decision,
		ShadowDecision: out.ShadowDecision,
		Score:          round3(out.Score),
		Confidence:     round3(out.Confidence),
		Reasons:        make([]DecisionReason, 0, len(out.Reasons)),
		Evidence:       DecisionEvidence{Events: out.EvidenceEvents, DurationMs: out.EvidenceMs},
		Mode:           out.Mode,
	}
	for _, rs := range out.Reasons {
		resp.Reasons = append(resp.Reasons, DecisionReason{Code: rs.Code, Weight: rs.Weight})
	}
	writeJSON(w, http.StatusOK, resp)
}

// ---------------------------------------------------------------
// POST /v1/outcomes — application server
// ---------------------------------------------------------------

// OutcomesRequest labels a past evaluation with what actually
// happened. This is the channel every future calibration depends on.
type OutcomesRequest struct {
	EvaluationID string `json:"evaluation_id" jsonschema:"description=evaluation_id returned by POST /v1/decisions; required"`
	// The enum is injected by cmd/gen-openapi from app.ValidOutcomes,
	// the same map the use case rejects unknown values against.
	Outcome    string `json:"outcome" jsonschema:"description=What the action turned out to be. An unrecognised value is rejected rather than stored: a typo'd label silently degrades every future calibration."`
	ObservedAt string `json:"observed_at" jsonschema:"description=RFC 3339 timestamp of the application's observation. Optional; rejected with 400 if present and unparseable. The gap between this and the server's recorded_at is itself a signal; substituting the server clock would silently corrupt it.,format=date-time"`
}

func (s *Server) handleOutcomes(w http.ResponseWriter, r *http.Request) {
	if !s.authorizedSecret(r) {
		writeError(w, http.StatusUnauthorized, "invalid secret_key")
		return
	}

	var req OutcomesRequest
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

// ErrorResponse is the single error body shape. It was an untyped
// map until R1.14, which meant the one response every client must
// handle was the one no contract could describe.
type ErrorResponse struct {
	Error string `json:"error" jsonschema:"description=Human-readable failure reason. Not a stable enumeration; do not branch on it."`
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, ErrorResponse{Error: msg})
}

func round3(f float64) float64 {
	return float64(int(f*1000+0.5)) / 1000
}
