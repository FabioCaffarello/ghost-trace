package web

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/api"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/policy"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/session"
)

const (
	testSiteKey   = "pk_webtest"
	testSecretKey = "sk_webtest"
)

func discard() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// startAPI brings up a real API server for the demo host to call: the
// demo's whole point is exercising the HTTP trust boundary, so its
// tests do too.
func startAPI(t *testing.T) *httptest.Server {
	t.Helper()
	cfg := api.Config{
		TenantID: "t_test", SiteKey: testSiteKey, SecretKey: testSecretKey,
		Mode:          policy.ModeMonitor,
		CollectPolicy: api.CollectPolicy{PointerHz: 20, BatchMs: 2000, Types: []string{"pointer"}},
	}
	s := api.New(cfg, session.NewStore(30*time.Minute, time.Now), nil, time.Now, discard())
	srv := httptest.NewServer(s.Routes())
	t.Cleanup(srv.Close)
	return srv
}

func startDemo(t *testing.T, apiBase, captureLog string) *httptest.Server {
	t.Helper()
	h, err := New(Config{
		APIBase: apiBase, SiteKey: testSiteKey, SecretKey: testSecretKey,
		CaptureLog: captureLog,
	}, discard())
	if err != nil {
		t.Fatalf("web.New: %v", err)
	}
	mux := http.NewServeMux()
	h.Register(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func newSessionToken(t *testing.T, apiSrv *httptest.Server) string {
	t.Helper()
	body, _ := json.Marshal(map[string]any{
		"site_key": testSiteKey,
		"page":     map[string]any{"path": "/login"},
		"client":   map[string]any{"pointer": "fine", "viewport": []int{1440, 900}},
	})
	resp, err := http.Post(apiSrv.URL+"/v1/sessions", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("sessions: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	var out struct {
		SessionToken string `json:"session_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil || out.SessionToken == "" {
		t.Fatalf("no session token (err=%v)", err)
	}
	return out.SessionToken
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

func TestServeSDK(t *testing.T) {
	demo := startDemo(t, "http://127.0.0.1:1", "")
	resp, err := http.Get(demo.URL + "/sdk.js")
	if err != nil {
		t.Fatalf("GET /sdk.js: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "javascript") {
		t.Errorf("Content-Type = %q", ct)
	}
	sdk, _ := io.ReadAll(resp.Body)
	if len(sdk) == 0 {
		t.Error("empty SDK body")
	}
	if strings.Contains(string(sdk), testSecretKey) {
		t.Error("secret key leaked into the SDK")
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
	apiSrv := startAPI(t)
	demo := startDemo(t, apiSrv.URL, "")
	token := newSessionToken(t, apiSrv)

	status, out := postLogin(t, demo, map[string]any{
		"session_token": token, "username": "alice",
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

func TestLoginAppendsCaptureRow(t *testing.T) {
	apiSrv := startAPI(t)
	log := t.TempDir() + "/human_sessions.jsonl"
	demo := startDemo(t, apiSrv.URL, log)
	token := newSessionToken(t, apiSrv)

	// Participant present: the row is recorded.
	postLogin(t, demo, map[string]any{
		"session_token": token, "username": "x",
		"participant": "p07", "arm": "B", "condition": "mouse-desktop", "visit": 3,
	})
	// No participant: nothing is recorded.
	postLogin(t, demo, map[string]any{
		"session_token": token, "username": "y",
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
