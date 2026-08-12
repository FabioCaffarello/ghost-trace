package web

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/FabioCaffarello/ghost-trace/libs/archive"
	"github.com/FabioCaffarello/ghost-trace/libs/decision"
	"github.com/FabioCaffarello/ghost-trace/libs/policy"
	"github.com/FabioCaffarello/ghost-trace/libs/tenant"
)

const (
	testSiteKey   = "pk_webtest"
	testSecretKey = "sk_webtest"

	// The collector's origin as the BROWSER sees it. It is not this
	// service, which is the whole point of the split.
	testAPIBase = "https://collector.example"
)

func testTenants(t *testing.T) *tenant.Registry {
	t.Helper()
	r, err := tenant.New(tenant.Tenant{ID: "t_test", SiteKey: testSiteKey, SecretKey: testSecretKey})
	if err != nil {
		t.Fatalf("tenant registry: %v", err)
	}
	return r
}

func discard() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// fixedSessions answers every lookup with the same session. It stands
// in for the collector's state without this module depending on the
// collector — which it must not, now that they are separate services.
type fixedSessions struct{}

func (fixedSessions) Lookup(context.Context, string, string) (decision.Session, bool, error) {
	return decision.Session{ID: "s_webtest", TenantID: "t_test"}, true, nil
}

// startEngine brings up a REAL /v1/decisions for the demo host to call.
//
// It mounts libs/decision — the same package the collector and the
// decision engine mount — rather than a hand-written stub, because the
// demo's whole point is exercising the HTTP trust boundary and a stub
// would exercise the test author's idea of it. What is faked here is
// only where the session state comes from.
func startEngine(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	decision.New(decision.Config{
		Mode: policy.ModeMonitor, Tenants: testTenants(t),
	}, fixedSessions{}, archive.Null{}, time.Now, discard()).Mount(mux)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func startDemo(t *testing.T, apiBase, captureLog string) *httptest.Server {
	t.Helper()
	var sink CaptureSink = NoCapture{}
	if captureLog != "" {
		sink = NewFileCaptureSink(captureLog)
	}
	h, err := New(Config{SiteKey: testSiteKey, APIBase: testAPIBase},
		NewHTTPDecisionClient(apiBase, testSecretKey, 0), sink, discard())
	if err != nil {
		t.Fatalf("web.New: %v", err)
	}
	mux := http.NewServeMux()
	h.Register(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

// stubDecisions substitutes the engine without a network — what the
// DecisionClient port exists to allow.
type stubDecisions struct {
	out map[string]any
	err error

	// filed collects the bodies passed to Report. A pointer because the
	// stub is passed by value and a value receiver cannot otherwise
	// record what it was asked to file — which is the whole assertion
	// for the labels channel.
	filed     *[]string
	reportErr error
}

func (s stubDecisions) Decide(context.Context, []byte) (map[string]any, error) {
	return s.out, s.err
}

func (s stubDecisions) Report(_ context.Context, body []byte) error {
	if s.reportErr != nil {
		return s.reportErr
	}
	if s.filed != nil {
		*s.filed = append(*s.filed, string(body))
	}
	return nil
}

// failingSink proves a capture failure costs a log line, not the
// volunteer's session.
type failingSink struct{}

func (failingSink) Append(CaptureRow) error { return errors.New("disk full") }

func startDemoWithPorts(t *testing.T, d DecisionClient, c CaptureSink) *httptest.Server {
	t.Helper()
	h, err := New(Config{SiteKey: testSiteKey}, d, c, discard())
	if err != nil {
		t.Fatalf("web.New: %v", err)
	}
	mux := http.NewServeMux()
	h.Register(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func postLogin(t *testing.T, demo *httptest.Server, payload map[string]any) (int, map[string]any) {
	t.Helper()
	b, _ := json.Marshal(payload)
	resp, err := http.Post(demo.URL+"/demo/login", "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("demo login: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

func TestServePageSubstitutesSiteKey(t *testing.T) {
	demo := startDemo(t, "http://127.0.0.1:1", "")
	resp, err := http.Get(demo.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	page, _ := io.ReadAll(resp.Body)

	if !strings.Contains(string(page), testSiteKey) {
		t.Error("page does not embed the site key")
	}
	if strings.Contains(string(page), "SITE_KEY_PLACEHOLDER") {
		t.Error("placeholder survived substitution")
	}
	if strings.Contains(string(page), testSecretKey) {
		t.Error("secret key leaked into the page")
	}
}

func TestTheDemoHostDoesNotServeTheSDK(t *testing.T) {
	// The SDK is Ghost Trace's artefact, served from Ghost Trace's
	// origin. A customer embeds it; they do not vendor a copy, and a
	// vendored copy is how every deployment ends up running a different
	// version of the thing that decides what the wire carries.
	demo := startDemo(t, "http://127.0.0.1:1", "")
	resp, err := http.Get(demo.URL + "/sdk.js")
	if err != nil {
		t.Fatalf("GET /sdk.js: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404: this service must not ship the SDK", resp.StatusCode)
	}
}

func TestThePageLoadsTheSDKFromTheCollector(t *testing.T) {
	demo := startDemo(t, "http://127.0.0.1:1", "")
	resp, err := http.Get(demo.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	page, _ := io.ReadAll(resp.Body)

	if !strings.Contains(string(page), `src="`+testAPIBase+`/sdk.js"`) {
		t.Errorf("the page does not load the SDK from %s", testAPIBase)
	}
	// And the SDK is told where to POST, since it is no longer
	// same-origin with the collector.
	if !strings.Contains(string(page), `window.GHOST_TRACE_API = "`+testAPIBase+`"`) {
		t.Errorf("the page does not point the SDK at %s", testAPIBase)
	}
	if strings.Contains(string(page), "API_BASE_PLACEHOLDER") {
		t.Error("placeholder survived substitution")
	}
	if strings.Contains(string(page), testSecretKey) {
		t.Error("secret key leaked into the page")
	}
}
func TestUnknownPathIs404(t *testing.T) {
	demo := startDemo(t, "http://127.0.0.1:1", "")
	resp, err := http.Get(demo.URL + "/nope")
	if err != nil {
		t.Fatalf("GET /nope: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

func TestLoginReturnsRealDecision(t *testing.T) {
	engine := startEngine(t)
	demo := startDemo(t, engine.URL, "")

	status, out := postLogin(t, demo, map[string]any{
		"session_token": "st_webtest", "username": "alice",
	})
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	for _, k := range []string{"evaluation_id", "decision", "score", "confidence", "mode"} {
		if _, ok := out[k]; !ok {
			t.Errorf("response missing %q", k)
		}
	}
	if out["mode"] == "fail-open" {
		t.Error("real backend answered but the demo failed open")
	}
}

func TestLoginFailsOpenWhenAPIUnreachable(t *testing.T) {
	// Fail-open is the §5 commitment: a degraded detector must not take
	// down the host's login.
	demo := startDemo(t, "http://127.0.0.1:1", "")

	status, out := postLogin(t, demo, map[string]any{
		"session_token": "st_whatever", "username": "alice",
	})
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200 (fail-open)", status)
	}
	if out["decision"] != "allow" {
		t.Errorf("decision = %v, want allow", out["decision"])
	}
	if out["mode"] != "fail-open" {
		t.Errorf("mode = %v, want fail-open", out["mode"])
	}
}

func TestLoginPassesStubbedDecisionThrough(t *testing.T) {
	// The port in action: no API server anywhere, the decision comes
	// from the stub, and the handler serves it untouched.
	demo := startDemoWithPorts(t, stubDecisions{out: map[string]any{
		"evaluation_id": "ev_stub", "decision": "challenge",
		"score": 0.9, "confidence": 0.8, "mode": "enforce",
	}}, NoCapture{})

	status, out := postLogin(t, demo, map[string]any{
		"session_token": "st_x", "username": "alice",
	})
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if out["decision"] != "challenge" || out["evaluation_id"] != "ev_stub" {
		t.Errorf("decision not passed through: %v", out)
	}
}

func TestLoginFailsOpenOnPortError(t *testing.T) {
	demo := startDemoWithPorts(t, stubDecisions{err: errors.New("engine down")}, NoCapture{})
	status, out := postLogin(t, demo, map[string]any{
		"session_token": "st_x", "username": "alice",
	})
	if status != http.StatusOK || out["mode"] != "fail-open" {
		t.Errorf("status = %d, mode = %v; want 200 + fail-open", status, out["mode"])
	}
}

func TestCaptureFailureDoesNotCostTheSession(t *testing.T) {
	// A volunteer's five minutes must not be wasted on a full disk:
	// the sink error is logged and swallowed, the decision still lands.
	demo := startDemoWithPorts(t, stubDecisions{out: map[string]any{
		"evaluation_id": "ev_1", "decision": "allow",
	}}, failingSink{})

	status, out := postLogin(t, demo, map[string]any{
		"session_token": "st_x", "username": "x",
		"participant": "p01", "arm": "B", "condition": "c", "visit": 1,
	})
	if status != http.StatusOK || out["decision"] != "allow" {
		t.Errorf("status = %d, out = %v; capture failure leaked to the volunteer", status, out)
	}
}

func TestLoginAppendsCaptureRow(t *testing.T) {
	engine := startEngine(t)
	log := t.TempDir() + "/human_sessions.jsonl"
	demo := startDemo(t, engine.URL, log)

	// Participant present: the row is recorded.
	postLogin(t, demo, map[string]any{
		"session_token": "st_webtest", "username": "x",
		"participant": "p07", "arm": "B", "condition": "mouse-desktop", "visit": 3,
	})
	// No participant: nothing is recorded.
	postLogin(t, demo, map[string]any{
		"session_token": "st_webtest", "username": "y",
	})

	raw, err := os.ReadFile(log)
	if err != nil {
		t.Fatalf("capture log: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
	if len(lines) != 1 {
		t.Fatalf("capture rows = %d, want exactly 1", len(lines))
	}

	var row map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &row); err != nil {
		t.Fatalf("row is not JSON: %v", err)
	}
	if row["participant"] != "p07" || row["arm"] != "B" || row["visit"] != float64(3) {
		t.Errorf("row labels = %v", row)
	}
	if _, ok := row["evaluation_id"]; !ok {
		t.Error("row missing evaluation_id")
	}
}

// The labels channel. Until PR-E6 `POST /v1/outcomes` had no caller
// anywhere in the product — only the topology gates reached it — so the
// endpoint every future calibration depends on was exercised solely by
// the scripts that check it.
func TestTheApplicationFilesWhatItsOwnActionTurnedOutToBe(t *testing.T) {
	for _, tc := range []struct {
		name     string
		decision map[string]any
		want     string
	}{
		{
			name:     "allow — the user was signed in",
			decision: map[string]any{"evaluation_id": "ev_1", "decision": "allow"},
			want:     "login_success",
		},
		{
			name:     "block — the user was refused",
			decision: map[string]any{"evaluation_id": "ev_2", "decision": "block"},
			want:     "login_failure",
		},
		{
			// The case worth having a test for. A challenge means the
			// application is NOT finished: there is no challenge_passed
			// to file yet, and filing login_success because the request
			// returned would label a session that was never let in.
			// libs/decision rejects a label outside the enumeration; a
			// wrong label INSIDE it is the same damage with nothing left
			// to catch it.
			name:     "challenge — nothing has turned out yet",
			decision: map[string]any{"evaluation_id": "ev_3", "decision": "challenge"},
			want:     "",
		},
		{
			// Fail-open synthesises a verdict with no evaluation behind
			// it. There is nothing on the other side to label.
			name:     "fail-open — there is no evaluation to label",
			decision: map[string]any{"evaluation_id": "", "decision": "allow", "mode": "fail-open"},
			want:     "",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var filed []string
			demo := startDemoWithPorts(t,
				stubDecisions{out: tc.decision, filed: &filed}, NoCapture{})

			_, body := postLogin(t, demo, map[string]any{
				"session_token": "st_webtest", "username": "x",
			})

			if tc.want == "" {
				if len(filed) != 0 {
					t.Errorf("filed %v, want nothing", filed)
				}
				if _, ok := body["outcome_reported"]; ok {
					t.Errorf("response claims a label was filed: %v", body["outcome_reported"])
				}
				return
			}

			if len(filed) != 1 {
				t.Fatalf("filed %d labels, want 1", len(filed))
			}
			var got map[string]any
			if err := json.Unmarshal([]byte(filed[0]), &got); err != nil {
				t.Fatalf("the body is not JSON: %v", err)
			}
			if got["outcome"] != tc.want {
				t.Errorf("outcome = %v, want %q", got["outcome"], tc.want)
			}
			if got["evaluation_id"] != tc.decision["evaluation_id"] {
				t.Errorf("evaluation_id = %v, want %v", got["evaluation_id"], tc.decision["evaluation_id"])
			}
			// The response tells the page what was filed, so an operator
			// reads the label rather than inferring it.
			if body["outcome_reported"] != tc.want {
				t.Errorf("outcome_reported = %v, want %q", body["outcome_reported"], tc.want)
			}
		})
	}
}

// A label that could not be filed must not cost the user their login —
// they are already signed in, and refusing them because a record failed
// is the detector-takes-down-the-login failure §5 exists to prevent. But
// the response must not then claim a label that was never stored.
func TestAnUnfiledLabelCostsALogLineAndNotTheLogin(t *testing.T) {
	demo := startDemoWithPorts(t, stubDecisions{
		out:       map[string]any{"evaluation_id": "ev_9", "decision": "allow"},
		reportErr: errors.New("outcomes: 503 Service Unavailable"),
	}, NoCapture{})

	status, body := postLogin(t, demo, map[string]any{
		"session_token": "st_webtest", "username": "x",
	})

	if status != http.StatusOK {
		t.Errorf("status = %d, want 200: the login stands", status)
	}
	if body["decision"] != "allow" {
		t.Errorf("decision = %v, want allow", body["decision"])
	}
	if v, ok := body["outcome_reported"]; ok {
		t.Errorf("response claims %v was filed, and it was not", v)
	}
}
