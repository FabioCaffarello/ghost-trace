// Byte-level goldens for every endpoint and every error body.
//
// The previous golden decoded the response into map[string]any,
// re-marshalled it with indentation, and compared THAT. It froze the
// data, not the wire: JSON number formatting, key order, the trailing
// newline and whether a field is emitted at all were all normalised
// away before the comparison. `"score": 1` and `"score": 1.0` are the
// same map and different bytes, and a client parses bytes.
//
// These compare what the server actually wrote. Only the two genuinely
// random values — the session token and the evaluation id — are
// replaced, and they are replaced in the RAW BYTES so the surrounding
// formatting is still under test.
//
// Regenerate deliberately:  go test ./internal/api -run Golden -update
package api

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/policy"
)

var updateGolden = flag.Bool("update", false, "rewrite golden files from current output")

const goldenDir = "testdata/golden"

// postRaw sends bytes and returns the status and the bytes that came
// back, with nothing decoded on the way in or out.
func postRaw(t *testing.T, url, path, bearer string, body []byte) (int, []byte) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url+path, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return resp.StatusCode, raw
}

// session.NewID is a prefix plus 18 random bytes in base64url, so a
// token looks like st_AufHcXG3MEt9x5F3hzVf03ZS — NOT hex. An earlier
// version of this pattern assumed hex, matched nothing, and committed a
// golden containing one run's real token: a file that could only ever
// match the run that produced it.
var (
	reToken  = regexp.MustCompile(`"session_token":"[A-Za-z]+_[A-Za-z0-9_-]{20,}"`)
	reEvalID = regexp.MustCompile(`"evaluation_id":"[^"]+"`)
)

func normalize(raw []byte) []byte {
	out := reToken.ReplaceAll(raw, []byte(`"session_token":"TOKEN_NORMALIZED"`))
	out = reEvalID.ReplaceAll(out, []byte(`"evaluation_id":"ev_NORMALIZED"`))
	return out
}

// assertGolden compares raw response bytes against a committed file.
func assertGolden(t *testing.T, name string, raw []byte) {
	t.Helper()
	got := normalize(raw)
	path := filepath.Join(goldenDir, name)

	if *updateGolden {
		if err := os.MkdirAll(goldenDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s (create with -update): %v", path, err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("%s drifted.\n--- got  (%d bytes) ---\n%s\n--- want (%d bytes) ---\n%s",
			name, len(got), got, len(want), want)
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

// ---------------------------------------------------------------
// success bodies
// ---------------------------------------------------------------

func TestGoldenSessions(t *testing.T) {
	srv := newTestServer(t, policy.ModeMonitor)
	status, raw := postRaw(t, srv.URL, "/v1/sessions", "", mustJSON(t, map[string]any{
		"site_key": testSiteKey,
		"page":     map[string]any{"path": "/login"},
		"client": map[string]any{
			"pointer": "mouse", "touch": false,
			"viewport": []int{1440, 900}, "tz_offset": -180,
			"reduced_motion": false,
		},
	}))
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", status, raw)
	}
	assertGolden(t, "sessions_200.json", raw)
}

// Telemetry and outcomes answer with a status and NOTHING. That is
// part of the contract too — a client that expects a body would break
// on the day one appeared.
func TestGoldenTelemetryHasNoBody(t *testing.T) {
	srv := newTestServer(t, policy.ModeMonitor)
	token := startSession(t, srv)
	status, raw := postRaw(t, srv.URL, "/v1/telemetry", token, mustJSON(t, linearBatch(token, 1, 40)))
	if status != http.StatusAccepted {
		t.Fatalf("status = %d, want 202: %s", status, raw)
	}
	if len(raw) != 0 {
		t.Errorf("202 carried a body of %d bytes: %s", len(raw), raw)
	}
}

// An unknown token is also 202 with no body: telemetry is
// fire-and-forget and an error would only make a stale SDK retry.
func TestGoldenTelemetryUnknownSessionStillAccepted(t *testing.T) {
	srv := newTestServer(t, policy.ModeMonitor)
	status, raw := postRaw(t, srv.URL, "/v1/telemetry", "",
		mustJSON(t, linearBatch("no_such_token", 1, 4)))
	if status != http.StatusAccepted {
		t.Fatalf("status = %d, want 202: %s", status, raw)
	}
	if len(raw) != 0 {
		t.Errorf("202 carried a body of %d bytes: %s", len(raw), raw)
	}
}

func TestGoldenDecisions(t *testing.T) {
	srv := newTestServer(t, policy.ModeMonitor)
	token := startSession(t, srv)
	for seq := 1; seq <= 3; seq++ {
		if status, raw := postRaw(t, srv.URL, "/v1/telemetry", token,
			mustJSON(t, linearBatch(token, seq, 40))); status != http.StatusAccepted {
			t.Fatalf("telemetry = %d: %s", status, raw)
		}
	}

	status, raw := postRaw(t, srv.URL, "/v1/decisions", testSecretKey, mustJSON(t, map[string]any{
		"session_token": token,
		"action":        "login",
		"subject_id":    "user_golden",
	}))
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", status, raw)
	}
	assertGolden(t, "decisions_200.json", raw)
}

func TestGoldenOutcomesHasNoBody(t *testing.T) {
	srv := newTestServer(t, policy.ModeMonitor)
	// NullArchive answers ErrArchiveUnavailable, so a 202 here needs a
	// real archive; the archive-less path is asserted below as 503.
	status, raw := postRaw(t, srv.URL, "/v1/outcomes", testSecretKey, mustJSON(t, map[string]any{
		"evaluation_id": "ev_whatever",
		"outcome":       "login_success",
	}))
	if status != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 with NullArchive: %s", status, raw)
	}
	assertGolden(t, "error_503_no_archive.json", raw)
}

// ---------------------------------------------------------------
// error bodies — the response every client must handle, and the one
// that had no contract at all until it became a typed struct
// ---------------------------------------------------------------

func TestGoldenErrorBodies(t *testing.T) {
	srv := newTestServer(t, policy.ModeMonitor)

	cases := []struct {
		name   string
		path   string
		bearer string
		body   []byte
		status int
		golden string
	}{
		{
			name:   "malformed JSON",
			path:   "/v1/sessions",
			body:   []byte("{not json"),
			status: http.StatusBadRequest,
			golden: "error_400_malformed.json",
		},
		{
			name:   "unknown site_key",
			path:   "/v1/sessions",
			body:   mustJSON(t, map[string]any{"site_key": "pk_wrong"}),
			status: http.StatusUnauthorized,
			golden: "error_401_site_key.json",
		},
		{
			name:   "missing secret_key",
			path:   "/v1/decisions",
			body:   mustJSON(t, map[string]any{"action": "login"}),
			status: http.StatusUnauthorized,
			golden: "error_401_secret_key.json",
		},
		{
			name:   "action required",
			path:   "/v1/decisions",
			bearer: testSecretKey,
			body:   mustJSON(t, map[string]any{"session_token": "t", "subject_id": "s"}),
			status: http.StatusBadRequest,
			golden: "error_400_action_required.json",
		},
		{
			name:   "evaluation_id required",
			path:   "/v1/outcomes",
			bearer: testSecretKey,
			body:   mustJSON(t, map[string]any{"outcome": "login_success"}),
			status: http.StatusBadRequest,
			golden: "error_400_evaluation_id_required.json",
		},
		{
			name:   "unknown outcome",
			path:   "/v1/outcomes",
			bearer: testSecretKey,
			body:   mustJSON(t, map[string]any{"evaluation_id": "ev_x", "outcome": "definitely_not_valid"}),
			status: http.StatusBadRequest,
			golden: "error_400_unknown_outcome.json",
		},
		{
			name:   "malformed telemetry",
			path:   "/v1/telemetry",
			body:   []byte("{not json"),
			status: http.StatusBadRequest,
			golden: "error_400_telemetry_malformed.json",
		},
		{
			name:   "outcomes without secret_key",
			path:   "/v1/outcomes",
			body:   mustJSON(t, map[string]any{"evaluation_id": "ev_x", "outcome": "login_success"}),
			status: http.StatusUnauthorized,
			golden: "error_401_outcomes_secret_key.json",
		},
		{
			name:   "observed_at not RFC 3339",
			path:   "/v1/outcomes",
			bearer: testSecretKey,
			body: mustJSON(t, map[string]any{
				"evaluation_id": "ev_x", "outcome": "login_success", "observed_at": "yesterday",
			}),
			status: http.StatusBadRequest,
			golden: "error_400_observed_at.json",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, raw := postRaw(t, srv.URL, tc.path, tc.bearer, tc.body)
			if status != tc.status {
				t.Fatalf("status = %d, want %d: %s", status, tc.status, raw)
			}
			assertGolden(t, tc.golden, raw)
		})
	}
}

// healthz is served as text, not JSON, and the container health check
// depends on it staying that way.
func TestGoldenHealthz(t *testing.T) {
	srv := newTestServer(t, policy.ModeMonitor)
	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if string(raw) != "ok\n" {
		t.Errorf("body = %q, want %q", raw, "ok\n")
	}
}
