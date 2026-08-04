// gen-openapi writes contract/openapi.yaml from the Go types the
// handlers actually decode into and encode from.
//
// WHY GENERATED, AND WHY ONLY PARTLY. A hand-written specification is
// a second description of the wire, free to drift from the first the
// moment anyone edits a struct — and nothing fails when it does. So
// every SHAPE here is reflected: field names, types, optionality,
// nesting, all of it derived from internal/api. Two enumerations that
// exist as Go values (policy.ReasonCodes, app.ValidOutcomes) are read
// from those values for the same reason.
//
// What reflection cannot know is SEMANTICS: which endpoint needs which
// credential, which status codes exist, what a 202 means here. That is
// written below, in Go, where it is reviewable and where the compiler
// at least holds it against real types.
//
// The output is committed and `make openapi-sync` fails on any
// difference — the same discipline libs/genproto is under.
//
// Run: make openapi
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/invopop/jsonschema"
	"sigs.k8s.io/yaml"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/api"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/app"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/policy"
)

// The spec version tracks the CONTRACT, not the build. It moves when
// the wire changes, which is what a client pins against.
const specVersion = "1.0.0"

// Where the wire types live, so their doc comments can be read. The
// generator runs from the module root (see the openapi make target).
const (
	modulePath   = "github.com/FabioCaffarello/ghost-trace/services/ingestion"
	apiSourceDir = "./internal/api"
)

func main() {
	out := flagOutput()
	doc := build()

	buf, err := marshalYAML(doc)
	if err != nil {
		fatal("marshal: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		fatal("mkdir: %v", err)
	}
	if err := os.WriteFile(out, buf, 0o644); err != nil {
		fatal("write %s: %v", out, err)
	}
	fmt.Fprintf(os.Stderr, "wrote %s\n", out)
}

func flagOutput() string {
	if len(os.Args) > 1 {
		return os.Args[1]
	}
	return filepath.Join("contract", "openapi.yaml")
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "gen-openapi: "+format+"\n", args...)
	os.Exit(1)
}

// ---------------------------------------------------------------
// document
// ---------------------------------------------------------------

func build() map[string]any {
	schemas, ref := reflectSchemas()

	return map[string]any{
		"openapi": "3.1.0",
		"info": map[string]any{
			"title":   "Ghost Trace",
			"version": specVersion,
			"summary": "Does this session behave like a human?",
			"description": strings.TrimSpace(`
Behavioural bot detection from interaction dynamics — how the pointer
moves, how keys are timed, how a form is actually filled. Not
fingerprinting: this API makes no claim about which browser is calling.

GENERATED FILE. Written by services/ingestion/cmd/gen-openapi from the
Go types the handlers decode into; CI fails if it drifts from them. Edit
the types, then run ` + "`make openapi`" + `.

Two callers, two trust levels. The BROWSER opens a session and posts
telemetry, and everything it sends is treated as hostile. The
APPLICATION SERVER asks for decisions and reports outcomes using
secret_key; it is the only caller permitted to assert who the subject
is. A session token correlates telemetry — it does not authenticate
anyone.

The service is fail-open. Telemetry is fire-and-forget and loss is
expected; a decision is a judgement under uncertainty, and score and
confidence are separate numbers precisely so that "nothing suspicious
observed" is distinguishable from "this looks human".
`),
			"license": map[string]any{
				"name":       "Apache-2.0",
				"identifier": "Apache-2.0",
			},
		},
		"servers": []any{
			map[string]any{
				"url":         "http://127.0.0.1:8080",
				"description": "Local slice (make run, or the compose core profile)",
			},
		},
		"tags": []any{
			map[string]any{"name": "browser", "description": "Called by the SDK in the page. Hostile input."},
			map[string]any{"name": "application", "description": "Called by the application server with secret_key."},
			map[string]any{"name": "operations", "description": "Liveness."},
		},
		"paths": map[string]any{
			"/v1/sessions":  map[string]any{"post": opSessions(ref)},
			"/v1/telemetry": map[string]any{"post": opTelemetry(ref)},
			"/v1/decisions": map[string]any{"post": opDecisions(ref)},
			"/v1/outcomes":  map[string]any{"post": opOutcomes(ref)},
			"/healthz":      map[string]any{"get": opHealthz()},
		},
		"components": map[string]any{
			"securitySchemes": map[string]any{
				"secretKey": map[string]any{
					"type":   "http",
					"scheme": "bearer",
					"description": "The tenant's secret_key. Authenticates the APPLICATION SERVER. " +
						"Compared in constant time, because a byte-wise comparison leaks " +
						"prefix length through response timing on an internet-facing endpoint.",
				},
				"sessionToken": map[string]any{
					"type":   "http",
					"scheme": "bearer",
					"description": "A session_token from POST /v1/sessions. Correlates telemetry to a " +
						"session; it is NOT a credential and identifies nobody.",
				},
			},
			"schemas": schemas,
		},
	}
}

// ---------------------------------------------------------------
// operations
// ---------------------------------------------------------------

func opSessions(ref refFunc) map[string]any {
	return map[string]any{
		"operationId": "startSession",
		"summary":     "Open a session",
		"description": "Returns a session token and the collection policy the SDK should apply. " +
			"The collect block is server-driven so collection can be retuned without shipping a new SDK.\n\n" +
			"No security scheme: site_key travels in the body. It identifies the tenant and stops " +
			"cross-tenant noise — it does not authenticate, and it is public in the page source.",
		"tags":     []any{"browser"},
		"security": []any{},
		"requestBody": body(ref("SessionsRequest"),
			"The tenant's public key plus what the browser reports about itself.",
			readFixture("sessions.json")),
		"responses": responses(map[string]any{
			"200": jsonResponseFromGolden("Session opened", ref("SessionsResponse"), "sessions_200.json"),
			"400": errorResponseFromGolden(ref, "Malformed JSON body, or a body over 1 MiB", "error_400_malformed.json"),
			"401": errorResponseFromGolden(ref, "Unknown site_key", "error_401_site_key.json"),
			"500": errorResponse(ref, "Session could not be created"),
		}),
	}
}

func opTelemetry(ref refFunc) map[string]any {
	return map[string]any{
		"operationId": "ingestTelemetry",
		"summary":     "Post a telemetry batch",
		"description": "Fire-and-forget. Always 202 on an accepted request, INCLUDING when the session " +
			"token is unknown or expired: telemetry loss is expected (§5), and an error there would " +
			"only teach a stale SDK to retry in a loop. Ingestion is idempotent on (session, seq).\n\n" +
			"The token may travel as an Authorization: Bearer header or in the body; the header wins.",
		"tags":     []any{"browser"},
		"security": []any{map[string]any{"sessionToken": []any{}}, map[string]any{}},
		"requestBody": body(ref("TelemetryBatch"),
			"One flush of interaction events. The pts triples are [x, y, milliseconds since the previous sample].",
			readFixture("telemetry_pointer_and_keys.json")),
		"responses": responses(map[string]any{
			"202": map[string]any{"description": "Accepted. No body — including for an unknown session."},
			"400": errorResponseFromGolden(ref, "Malformed JSON body, or a body over 1 MiB", "error_400_telemetry_malformed.json"),
			"500": errorResponse(ref, "Telemetry could not be ingested"),
		}),
	}
}

func opDecisions(ref refFunc) map[string]any {
	return map[string]any{
		"operationId": "decide",
		"summary":     "Ask for a judgement",
		"description": "The only endpoint that accepts subject_id and action, which is exactly why " +
			"neither is ever read from a browser request.\n\n" +
			"In monitor mode `decision` is always allow and `shadow_decision` carries what enforce " +
			"mode would have returned; in enforce mode `shadow_decision` is absent. A client that " +
			"reads only `decision` therefore measures nothing while the service is in monitor mode.",
		"tags":     []any{"application"},
		"security": []any{map[string]any{"secretKey": []any{}}},
		"requestBody": body(ref("DecisionsRequest"),
			"Which session, and what it is trying to do.",
			readFixture("decisions.json")),
		"responses": responses(map[string]any{
			"200": jsonResponseFromGolden("A judgement", ref("DecisionsResponse"), "decisions_200.json"),
			"400": errorResponseFromGolden(ref, "Malformed JSON body, or action missing", "error_400_action_required.json"),
			"401": errorResponseFromGolden(ref, "Missing or invalid secret_key", "error_401_secret_key.json"),
			"500": errorResponse(ref, "Decision could not be produced"),
		}),
	}
}

func opOutcomes(ref refFunc) map[string]any {
	return map[string]any{
		"operationId": "recordOutcome",
		"summary":     "Label a past evaluation",
		"description": "The labels channel every future calibration depends on.\n\n" +
			"observed_at is the application's claim about when it saw the outcome; the server " +
			"separately records its own observation time. The gap between them is itself a signal, " +
			"so an unparseable observed_at is rejected rather than quietly replaced by the server " +
			"clock — that substitution would collapse the gap to zero and corrupt the channel " +
			"invisibly.",
		"tags":     []any{"application"},
		"security": []any{map[string]any{"secretKey": []any{}}},
		"requestBody": body(ref("OutcomesRequest"),
			"What actually happened to a past evaluation.",
			readFixture("outcomes_with_observed_at.json")),
		"responses": responses(map[string]any{
			"202": map[string]any{"description": "Outcome recorded. No body."},
			"400": errorResponseFromGolden(ref, "Malformed body, missing evaluation_id, unknown outcome, or observed_at that is not RFC 3339", "error_400_unknown_outcome.json"),
			"401": errorResponseFromGolden(ref, "Missing or invalid secret_key", "error_401_outcomes_secret_key.json"),
			"503": errorResponseFromGolden(ref, "No durable archive is configured. Refusing is deliberate: a label with nowhere to live is worse than a refusal, because the caller would believe it had reported one.", "error_503_no_archive.json"),
			"500": errorResponse(ref, "Outcome could not be recorded"),
		}),
	}
}

func opHealthz() map[string]any {
	return map[string]any{
		"operationId": "healthz",
		"summary":     "Liveness",
		"description": "Returns 200 while the process is serving. Used by the container health check, " +
			"which execs the binary itself because the runtime image is distroless and has no shell.",
		"tags":     []any{"operations"},
		"security": []any{},
		"responses": responses(map[string]any{
			"200": map[string]any{
				"description": "Serving",
				"content": map[string]any{
					"text/plain": map[string]any{
						"schema":  map[string]any{"type": "string"},
						"example": "ok\n",
					},
				},
			},
		}),
	}
}

// ---------------------------------------------------------------
// building blocks
// ---------------------------------------------------------------

type refFunc func(name string) map[string]any

func body(schema map[string]any, desc string, example any) map[string]any {
	content := map[string]any{"schema": schema}
	if example != nil {
		content["example"] = example
	}
	return map[string]any{
		"required":    true,
		"description": desc,
		"content":     map[string]any{"application/json": content},
	}
}

func jsonResponse(desc string, schema map[string]any) map[string]any {
	return map[string]any{
		"description": desc,
		"content":     map[string]any{"application/json": map[string]any{"schema": schema}},
	}
}

// jsonResponseFromGolden is the same, with the EXAMPLE taken from the
// committed byte-level golden — the literal bytes the server wrote in
// a test. A hand-written example is a claim; this one has already been
// produced by the code it documents, and internal/api validates it
// against this very schema.
func jsonResponseFromGolden(desc string, schema map[string]any, golden string) map[string]any {
	resp := jsonResponse(desc, schema)
	content := resp["content"].(map[string]any)["application/json"].(map[string]any)
	content["example"] = readGolden(golden)
	return resp
}

// Both relative to the module root, where the generator runs.
const (
	goldenDir  = "internal/api/testdata/golden"
	fixtureDir = "../../contract/fixtures/requests"
)

// readFixture returns a committed request fixture — what the harness's
// wire modules actually produce. Using it as the published example
// means the request examples are as real as the response ones: a
// hand-written example is a claim, and the first version of this file
// made three of them that the wire vocabulary did not allow.
func readFixture(name string) any {
	raw, err := os.ReadFile(filepath.Join(fixtureDir, name))
	if err != nil {
		fatal("read fixture %s for the published example "+
			"(generate with `make contract-fixtures`): %v", name, err)
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		fatal("fixture %s is not JSON: %v", name, err)
	}
	return v
}

func readGolden(name string) any {
	raw, err := os.ReadFile(filepath.Join(goldenDir, name))
	if err != nil {
		fatal("read golden %s for the published example: %v", name, err)
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		fatal("golden %s is not JSON: %v", name, err)
	}
	return v
}

func errorResponse(ref refFunc, desc string) map[string]any {
	return jsonResponse(desc, ref("ErrorResponse"))
}

func errorResponseFromGolden(ref refFunc, desc, golden string) map[string]any {
	return jsonResponseFromGolden(desc, ref("ErrorResponse"), golden)
}

func responses(m map[string]any) map[string]any { return m }

// ---------------------------------------------------------------
// reflection
// ---------------------------------------------------------------

// wireTypes is every type that appears on the wire. A type reachable
// from one of these is pulled in automatically as a $ref.
var wireTypes = []any{
	&api.SessionsRequest{},
	&api.SessionsResponse{},
	&api.TelemetryBatch{},
	&api.DecisionsRequest{},
	&api.DecisionsResponse{},
	&api.OutcomesRequest{},
	&api.ErrorResponse{},
}

// reflectSchemas turns the wire types into OpenAPI component schemas.
func reflectSchemas() (map[string]any, refFunc) {
	r := &jsonschema.Reflector{
		// Definitions are shared components, so nothing is inlined and
		// every named type appears once under components/schemas.
		ExpandedStruct: false,
		DoNotReference: false,
	}

	// Go doc comments on the wire types become the schema descriptions.
	// The alternative is writing the same paragraphs twice — once for
	// whoever reads the code and once for whoever reads the contract —
	// and the second copy is the one that goes stale, because nothing
	// is looking at it.
	if err := r.AddGoComments(modulePath, apiSourceDir); err != nil {
		fatal("read Go doc comments from %s: %v", apiSourceDir, err)
	}

	schemas := map[string]any{}
	for _, t := range wireTypes {
		s := r.Reflect(t)
		// OpenAPI 3.1 schemas carry no $schema and no root $ref: the
		// document itself declares the dialect.
		s.Version = ""

		raw, err := json.Marshal(s)
		if err != nil {
			fatal("marshal schema for %T: %v", t, err)
		}
		var doc map[string]any
		if err := json.Unmarshal(raw, &doc); err != nil {
			fatal("unmarshal schema for %T: %v", t, err)
		}
		defs, _ := doc["$defs"].(map[string]any)
		for name, def := range defs {
			schemas[name] = def
		}
	}

	// $defs is the JSON Schema location; OpenAPI keeps components
	// elsewhere, so every reference is rewritten to point at the one
	// place the schemas actually live.
	rewriteRefs(schemas)
	injectEnums(schemas)
	applyRequired(schemas)

	return schemas, func(name string) map[string]any {
		if _, ok := schemas[name]; !ok {
			fatal("no schema named %q was reflected", name)
		}
		return map[string]any{"$ref": "#/components/schemas/" + name}
	}
}

func rewriteRefs(node any) {
	switch v := node.(type) {
	case map[string]any:
		for key, val := range v {
			if key == "$ref" {
				if s, ok := val.(string); ok {
					v[key] = strings.Replace(s, "#/$defs/", "#/components/schemas/", 1)
				}
				continue
			}
			rewriteRefs(val)
		}
	case []any:
		for _, item := range v {
			rewriteRefs(item)
		}
	}
}

// injectEnums fills in the two enumerations that exist as Go values.
// Restating them in a struct tag would be a second list, free to drift
// from the one the service enforces — which is the whole failure this
// generator exists to make impossible.
func injectEnums(schemas map[string]any) {
	setEnum(schemas, "DecisionReason", "code", policy.ReasonCodes)
	setEnum(schemas, "DecisionsResponse", "decision", policy.Decisions)
	setEnum(schemas, "DecisionsResponse", "shadow_decision", policy.Decisions)
	setEnum(schemas, "DecisionsResponse", "mode", policy.Modes)

	// The telemetry vocabularies. Every one of these was retyped into a
	// struct tag in R1.14 and three of them were WRONG — `char` and
	// `edit` as key classes that nothing sends, pointer types invented
	// as mouse/touch/pen when the SDK reports the CSS media query
	// values, and form actions that omitted `paste`. The server
	// tolerates unknown values, so the specification was free to be
	// wrong about the wire until something checked. Reading them from
	// internal/ingest, which ingest/vocabulary_test.go holds sdk.js to,
	// is what makes that impossible.
	setEnum(schemas, "ClientHints", "pointer", ingest.PointerTypes)
	setEnum(schemas, "TelemetryEvent", "type", ingest.EventTypes)
	setEnum(schemas, "TelemetryEvent", "phase", ingest.KeyPhases)
	setEnum(schemas, "TelemetryEvent", "class", ingest.KeyClasses)
	setEnum(schemas, "TelemetryEvent", "mode", ingest.ScrollModes)
	setEnum(schemas, "TelemetryEvent", "action", ingest.FormActions)

	// `state` is shared by two event families in the flat union, so its
	// vocabulary is the union of both.
	setEnum(schemas, "TelemetryEvent", "state",
		append(append([]string{}, ingest.FocusStates...), ingest.VisibilityStates...))

	outcomes := make([]string, 0, len(app.ValidOutcomes))
	for name := range app.ValidOutcomes {
		outcomes = append(outcomes, name)
	}
	sort.Strings(outcomes) // a map has no order; the file must have one
	setEnum(schemas, "OutcomesRequest", "outcome", outcomes)
}

// requiredOverrides says which REQUEST fields the service actually
// refuses to work without.
//
// Reflection cannot know this. It marks every non-omitempty Go field
// required, because a Go struct field is always present — which for a
// request schema is a lie in two directions. TelemetryEvent is a flat
// union discriminated by `type`: a pointer event carries no `phase` and
// no `dy`, and the reflected schema demanded all of them. And the
// service is fail-open by design (§5), so most absent fields are
// tolerated rather than rejected.
//
// The rule applied here is: a field is required when the handler
// REFUSES the request without it. Everything else is optional, however
// pointless omitting it may be. Response schemas keep the reflected
// required list, which is correct — the server really does emit every
// one of those fields.
var requiredOverrides = map[string][]string{
	// A missing site_key cannot match the tenant's, so it is a 401.
	"SessionsRequest": {"site_key"},
	"PageRef":         {},
	"ClientHints":     {},

	// Telemetry rejects nothing: an unknown token, an empty batch and a
	// missing page all answer 202. Loss is expected, and an error here
	// would only teach a stale SDK to retry.
	"TelemetryBatch": {},
	"TelemetryPage":  {},
	"TelemetryEvent": {},

	// Decide refuses without an action (ErrActionRequired -> 400).
	"DecisionsRequest": {"action"},

	// RecordOutcome refuses without either (-> 400). observed_at is
	// optional, and invalid rather than absent is what earns the 400.
	"OutcomesRequest": {"evaluation_id", "outcome"},
}

func applyRequired(schemas map[string]any) {
	for typeName, required := range requiredOverrides {
		schema, ok := schemas[typeName].(map[string]any)
		if !ok {
			fatal("no schema named %q to set required on", typeName)
		}
		if len(required) == 0 {
			delete(schema, "required")
			continue
		}
		list := make([]any, len(required))
		for i, f := range required {
			list[i] = f
		}
		schema["required"] = list
	}
}

func setEnum(schemas map[string]any, typeName, field string, values []string) {
	schema, ok := schemas[typeName].(map[string]any)
	if !ok {
		fatal("no schema named %q to inject %q into", typeName, field)
	}
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		fatal("schema %q has no properties", typeName)
	}
	prop, ok := props[field].(map[string]any)
	if !ok {
		fatal("schema %q has no property %q", typeName, field)
	}
	enum := make([]any, len(values))
	for i, v := range values {
		enum[i] = v
	}
	prop["enum"] = enum
}

// ---------------------------------------------------------------
// output
// ---------------------------------------------------------------

func marshalYAML(doc map[string]any) ([]byte, error) {
	raw, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	out, err := yaml.JSONToYAML(raw)
	if err != nil {
		return nil, err
	}

	header := strings.TrimSpace(`
# GENERATED FILE — DO NOT EDIT.
#
# Written by services/ingestion/cmd/gen-openapi from the Go types in
# internal/api that the handlers actually decode into, plus the two
# enumerations that live as Go values (policy.ReasonCodes and
# app.ValidOutcomes).
#
# To change this file, change those types and run:
#
#     make openapi
#
# `+"`make openapi-sync`"+` fails if this file differs from a fresh
# generation, so the specification cannot describe an API the service
# does not serve.
`) + "\n\n"

	var buf bytes.Buffer
	buf.WriteString(header)
	buf.Write(out)
	return buf.Bytes(), nil
}
