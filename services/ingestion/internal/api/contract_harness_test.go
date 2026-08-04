// The contract-test harness: do the clients and the server still speak
// the same language?
//
// Audit finding M22. The experiment harness posts hand-built JSON from
// several places, outside all buf discipline, and this service tolerates
// unknown fields by design — telemetry is fire-and-forget and an SDK
// must survive a server that has moved on (§5, §7). Those two facts
// together mean a renamed wire field would leave every client sending
// the old name, the server zero-filling the new one, and every
// measurement quietly degrading with the whole suite green.
//
// R1.15 collapsed the clients onto one wire module per language
// (experiments/lib/wire.js, experiments/wire.py) and made them emit
// what they produce into contract/fixtures/requests/. This file is the
// other end. Each fixture is checked twice:
//
//  1. AGAINST THE PUBLISHED CONTRACT. The request schemas set
//     additionalProperties: false, so a field the server no longer
//     knows about fails — and `required` is enforcement-truth, so a
//     field the handler refuses to work without and never receives
//     fails too. This catches a rename from either direction.
//
//  2. AGAINST A REAL SERVER. Schema agreement is not acceptance: a
//     body can satisfy the schema and still be rejected. Every fixture
//     is replayed over HTTP and must get the status the contract
//     promises.
package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"sigs.k8s.io/yaml"

	"github.com/FabioCaffarello/ghost-trace/libs/policy"
)

const fixtureDir = "../../../../contract/fixtures/requests"

// Every fixture, the schema the contract declares for its endpoint, and
// what a real server should answer.
//
// `needsSession` fixtures carry a placeholder token from the emitter;
// replaying them verbatim would exercise the unknown-session path
// instead of the one that matters, so the token is swapped for a live
// one first.
var contractFixtures = []struct {
	fixture      string
	schema       string
	path         string
	bearer       string // "" none, "secret" the secret key, "session" the live token
	wantStatus   int
	needsSession bool
}{
	{"sessions.json", "SessionsRequest", "/v1/sessions", "", http.StatusOK, false},
	{"telemetry_pointer.json", "TelemetryBatch", "/v1/telemetry", "session", http.StatusAccepted, true},
	{"telemetry_pointer_and_keys.json", "TelemetryBatch", "/v1/telemetry", "session", http.StatusAccepted, true},
	{"decisions.json", "DecisionsRequest", "/v1/decisions", "secret", http.StatusOK, true},

	// Nothing in the harness posts an outcome yet — the labels channel
	// has no client at all. Covering it here is deliberate: the shape
	// is pinned before the first caller has to invent it. NullArchive
	// answers 503, which is the contract for an archive-less run.
	{"outcomes.json", "OutcomesRequest", "/v1/outcomes", "secret", http.StatusServiceUnavailable, false},
	{"outcomes_with_observed_at.json", "OutcomesRequest", "/v1/outcomes", "secret", http.StatusServiceUnavailable, false},
}

func TestClientFixturesSatisfyThePublishedContract(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	doc := loadSpec(t)
	const specURI = "https://ghost-trace.invalid/openapi.json"
	if err := compiler.AddResource(specURI, doc); err != nil {
		t.Fatalf("add spec: %v", err)
	}

	for _, tc := range contractFixtures {
		t.Run(tc.fixture, func(t *testing.T) {
			schema, err := compiler.Compile(specURI + "#/components/schemas/" + tc.schema)
			if err != nil {
				t.Fatalf("compile %s: %v", tc.schema, err)
			}
			var body any
			if err := json.Unmarshal(readFixture(t, tc.fixture), &body); err != nil {
				t.Fatalf("fixture is not JSON: %v", err)
			}
			if err := schema.Validate(body); err != nil {
				t.Errorf("what the harness sends does not satisfy %s, the schema the "+
					"contract declares for %s.\n\nA renamed field on either side looks "+
					"exactly like this, and the server would have accepted it in "+
					"silence:\n\n%v", tc.schema, tc.path, err)
			}
		})
	}
}

func TestClientFixturesAreAcceptedByARealServer(t *testing.T) {
	srv := newTestServer(t, policy.ModeMonitor)

	for _, tc := range contractFixtures {
		t.Run(tc.fixture, func(t *testing.T) {
			raw := readFixture(t, tc.fixture)

			bearer := ""
			switch tc.bearer {
			case "secret":
				bearer = testSecretKey
			case "session":
				bearer = startSession(t, srv)
			}

			// The site key is a per-tenant value: the fixture carries the
			// demo one and this server was built with a test one. Swapping
			// it keeps the replay about the SHAPE of the request rather
			// than about which deployment produced the fixture.
			raw = swapField(t, raw, "site_key", testSiteKey)

			if tc.needsSession {
				token := bearer
				if token == "" || tc.bearer == "secret" {
					token = startSession(t, srv)
				}
				// Give the decision path something to judge, so the
				// fixture exercises a real evaluation rather than an
				// empty session.
				if tc.path == "/v1/decisions" {
					for seq := 1; seq <= 2; seq++ {
						if status, body := postRaw(t, srv.URL, "/v1/telemetry", token,
							mustJSON(t, linearBatch(token, seq, 20))); status != http.StatusAccepted {
							t.Fatalf("seeding telemetry = %d: %s", status, body)
						}
					}
				}
				raw = swapToken(t, raw, token)
				if tc.bearer == "session" {
					bearer = token
				}
			}

			status, respBody := postRaw(t, srv.URL, tc.path, bearer, raw)
			if status != tc.wantStatus {
				t.Errorf("%s answered %d, the contract promises %d\n\nsent: %s\ngot:  %s",
					tc.path, status, tc.wantStatus, raw, respBody)
			}
		})
	}
}

// Every fixture on disk must appear in the table above; otherwise
// adding one to the harness silently adds an unchecked payload.
func TestEveryFixtureIsCovered(t *testing.T) {
	listed := map[string]bool{}
	for _, tc := range contractFixtures {
		listed[tc.fixture] = true
	}

	entries, err := os.ReadDir(fixtureDir)
	if err != nil {
		t.Fatalf("read %s (generate with `make contract-fixtures`): %v", fixtureDir, err)
	}
	var seen int
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		seen++
		if !listed[e.Name()] {
			t.Errorf("contract/fixtures/requests/%s is not in contractFixtures — the "+
				"harness sends a payload nothing checks", e.Name())
		}
	}
	if seen == 0 {
		t.Fatal("no fixtures found — this harness is not testing anything")
	}
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(fixtureDir, name))
	if err != nil {
		t.Fatalf("read fixture (generate with `make contract-fixtures`): %v", err)
	}
	return raw
}

// swapToken replaces the emitter's placeholder session_token with a
// live one, leaving every other byte of the fixture alone.
func swapToken(t *testing.T, raw []byte, token string) []byte {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("fixture is not a JSON object: %v", err)
	}
	if _, ok := body["session_token"]; !ok {
		t.Fatalf("fixture has no session_token to swap")
	}
	return swapField(t, raw, "session_token", token)
}

// swapField replaces one deployment-specific value, if present, and
// leaves every other byte alone.
func swapField(t *testing.T, raw []byte, field, value string) []byte {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("fixture is not a JSON object: %v", err)
	}
	if _, ok := body[field]; !ok {
		return raw
	}
	body[field] = value
	out, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return out
}

var _ = yaml.Marshal // loadSpec lives in openapi_conformance_test.go
