// Package wire is the HTTP request and response surface of Ghost
// Trace, as Go types.
//
// It is a module of its own because more than one service speaks this
// contract: the collector serves /v1/sessions and /v1/telemetry, and
// the decision engine serves /v1/decisions and /v1/outcomes. A second
// definition of the wire in the second service is precisely the failure
// audit finding M22 describes — the server tolerates unknown fields by
// design, so the two would drift in silence.
//
// These types are also the SOURCE of contract/openapi.yaml.
// cmd/gen-openapi reflects them, reads the doc comments below as schema
// descriptions, and CI fails if the committed specification drifts from
// what is here. The comments are therefore documentation in the
// published sense; write them for someone holding the specification and
// not the code.
//
// One sharp edge worth repeating: the `jsonschema` tag parser splits on
// COMMAS, so a comma inside a description silently truncates it. A test
// in this package rejects that.
package wire

// CollectPolicy is the server-driven collection configuration. It is
// part of the response contract, not just configuration: the SDK obeys
// it, so changing it changes what browsers send.
type CollectPolicy struct {
	PointerHz int      `json:"pointer_hz" jsonschema:"description=Pointer sampling rate in hertz"`
	BatchMs   int      `json:"batch_ms" jsonschema:"description=How often the SDK flushes a batch — milliseconds"`
	Types     []string `json:"types" jsonschema:"description=Event families the SDK should collect"`
}

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
	Pointer       string `json:"pointer" jsonschema:"description=Primary pointing device — the CSS pointer media query as the browser reports it"`
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
	Type string `json:"type" jsonschema:"description=Event family; selects which of the fields below apply"`
	T    uint32 `json:"t" jsonschema:"description=Milliseconds since session start"`

	// pointer
	Src string     `json:"src" jsonschema:"description=pointer: input source that produced the samples"`
	Pts [][3]int32 `json:"pts" jsonschema:"description=pointer: samples as [x y t_offset_ms] triples"`

	// key — timing and coarse class only, never content (§2, §6)
	Phase    string `json:"phase" jsonschema:"description=key: keystroke phase. Dwell is down to up on one key; flight is up to down between keys"`
	KeyClass string `json:"class" jsonschema:"description=key: COARSE key class. Never the key itself — no keylogging (§2 and §6)"`
	Target   string `json:"target" jsonschema:"description=key or form: coarse field role — never the field value"`

	// scroll
	Dy   int32  `json:"dy" jsonschema:"description=scroll: vertical delta in CSS pixels"`
	Mode string `json:"mode" jsonschema:"description=scroll: whether a real wheel gesture produced it or the page scrolled itself"`

	// focus / visibility
	State string `json:"state" jsonschema:"description=focus or visibility: the state entered. focus and blur for focus events; visible and hidden for visibility"`

	// form
	Action string `json:"action" jsonschema:"description=form: what happened to the field. 'injected' is the strongest single bot signal — a value that appeared with no keystroke and no paste behind it; 'paste' exists so a human pasting is not mistaken for one"`
}

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
	Decision       string           `json:"decision" jsonschema:"description=The decision actually returned. In monitor mode this is always allow"`
	ShadowDecision string           `json:"shadow_decision,omitempty" jsonschema:"description=In monitor mode: what enforce mode WOULD have returned. Absent in enforce mode"`
	Score          float64          `json:"score" jsonschema:"description=How bot-like the interaction looks. Rounded to three decimals.,minimum=0,maximum=1"`
	Confidence     float64          `json:"confidence" jsonschema:"description=How much evidence backs the score. Low confidence gates blocking (§3).,minimum=0,maximum=1"`
	Reasons        []DecisionReason `json:"reasons" jsonschema:"description=Contributing factors. Present so a decision can be explained and contested."`
	Evidence       DecisionEvidence `json:"evidence"`
	Mode           string           `json:"mode" jsonschema:"description=Operating mode of the service"`
}

// DecisionReason is one contributing factor.
type DecisionReason struct {
	// Injected by cmd/gen-openapi from policy.ReasonCodes:
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

// OutcomesRequest labels a past evaluation with what actually
// happened. This is the channel every future calibration depends on.
type OutcomesRequest struct {
	EvaluationID string `json:"evaluation_id" jsonschema:"description=evaluation_id returned by POST /v1/decisions; required"`
	// The enum is injected by cmd/gen-openapi from app.ValidOutcomes,
	// the same map the use case rejects unknown values against.
	Outcome    string `json:"outcome" jsonschema:"description=What the action turned out to be. An unrecognised value is rejected rather than stored: a typo'd label silently degrades every future calibration."`
	ObservedAt string `json:"observed_at" jsonschema:"description=RFC 3339 timestamp of the application's observation. Optional; rejected with 400 if present and unparseable. The gap between this and the server's recorded_at is itself a signal; substituting the server clock would silently corrupt it.,format=date-time"`
}

// ErrorResponse is the single error body shape. It was an untyped
// map until R1.14, which meant the one response every client must
// handle was the one no contract could describe.
type ErrorResponse struct {
	Error string `json:"error" jsonschema:"description=Human-readable failure reason. Not a stable enumeration; do not branch on it."`
}
