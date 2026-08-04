package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/FabioCaffarello/ghost-trace/libs/decision"
	"github.com/FabioCaffarello/ghost-trace/libs/policy"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/adapters/livesessions"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/app"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/session"
)

const demoOrigin = "https://demo.example"

// corsServer is the collector as the composition root builds it, with
// one origin allowed — mux and all, because WHICH ROUTES carry CORS is
// the property under test and a hand-mounted handler would not have it.
func corsServer(t *testing.T) *httptest.Server {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := session.NewStore(30*time.Minute, time.Now)
	a := app.New(app.Config{TenantID: "t_test"}, store, app.NullArchive{}, time.Now, log)
	s := New(Config{SiteKey: testSiteKey, AllowedOrigins: []string{demoOrigin}}, a, log)

	mux := s.Routes()
	decision.New(decision.Config{
		TenantID: "t_test", Mode: policy.ModeMonitor, SecretKey: testSecretKey,
	}, livesessions.New(store), app.NullArchive{}, time.Now, log).Mount(mux)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func preflight(t *testing.T, srv *httptest.Server, path string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodOptions, srv.URL+path, nil)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	req.Header.Set("Origin", demoOrigin)
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("preflight %s: %v", path, err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

func jsonBody(t *testing.T, v any) *bytes.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return bytes.NewReader(b)
}

func TestBrowserEndpointsAnswerAPreflight(t *testing.T) {
	// The demo page is served by another origin now. Without these the
	// browser refuses the real request and nothing in any log says why.
	srv := corsServer(t)
	for _, path := range []string{"/v1/sessions", "/v1/telemetry"} {
		resp := preflight(t, srv, path)
		if resp.StatusCode != http.StatusNoContent {
			t.Errorf("preflight %s = %d, want 204", path, resp.StatusCode)
		}
		if got := resp.Header.Get("Access-Control-Allow-Origin"); got != demoOrigin {
			t.Errorf("preflight %s allow-origin = %q, want %q", path, got, demoOrigin)
		}
	}
}

func TestTheSecretKeyEndpointsAreNotReachableCrossOrigin(t *testing.T) {
	// THE point of applying CORS per route rather than to the whole
	// mux. /v1/decisions and /v1/outcomes authenticate with secret_key
	// and are server-to-server (§1). Allowing a page to preflight them
	// advertises a door whose key must never be in a browser — and the
	// only way a browser could open it is if the secret had already
	// leaked into one.
	srv := corsServer(t)
	for _, path := range []string{"/v1/decisions", "/v1/outcomes"} {
		resp := preflight(t, srv, path)
		if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "" {
			t.Errorf("%s answered a preflight with allow-origin %q; the secret-key "+
				"endpoints must not be callable from a page", path, got)
		}
	}
}

func TestAPostFromTheAllowedOriginIsEchoedAndServed(t *testing.T) {
	srv := corsServer(t)
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/v1/sessions",
		jsonBody(t, map[string]any{
			"site_key": testSiteKey,
			"page":     map[string]any{"path": "/login"},
			"client":   map[string]any{"pointer": "fine"},
		}))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", demoOrigin)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != demoOrigin {
		t.Errorf("allow-origin = %q, want %q", got, demoOrigin)
	}
	if got := resp.Header.Get("Vary"); got != "Origin" {
		t.Errorf("Vary = %q, want Origin", got)
	}
}

func TestCORSIsOffWhenNoOriginIsConfigured(t *testing.T) {
	// The default. A same-origin deployment gets no CORS at all rather
	// than a permissive one.
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := session.NewStore(30*time.Minute, time.Now)
	a := app.New(app.Config{TenantID: "t_test"}, store, app.NullArchive{}, time.Now, log)
	srv := httptest.NewServer(New(Config{SiteKey: testSiteKey}, a, log).Routes())
	t.Cleanup(srv.Close)

	if got := preflight(t, srv, "/v1/sessions").
		Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("allow-origin = %q with an empty allowlist", got)
	}
}
