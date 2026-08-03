// Wire-contract tests. These drive real HTTP with the JSON shapes from
// docs/architecture.md §2–§3 rather than calling handlers with
// in-memory structs.
//
// v1 learned this the expensive way: a JSON field-name mismatch between
// a producer and a consumer broke the provenance chain in production
// while every in-memory test stayed green, because the tests never
// crossed the wire (see docs/v1-retrospective.md). If a field name here
// drifts from the contract, these fail.
package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/policy"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/session"
)

const (
	testSiteKey   = "pk_test"
	testSecretKey = "sk_test"
)

func newTestServer(t *testing.T, mode string) *httptest.Server {
	t.Helper()
	cfg := Config{
		TenantID:  "t_test",
		SiteKey:   testSiteKey,
		SecretKey: testSecretKey,
		Mode:      mode,
		CollectPolicy: CollectPolicy{
			PointerHz: 20, BatchMs: 2000, Types: []string{"pointer"},
		},
	}
	// nil archive: the decision path must not depend on storage.
	s := New(cfg, session.NewStore(30*time.Minute, time.Now), nil, time.Now,
		slog.New(slog.NewTextHandler(io.Discard, nil)))

	srv := httptest.NewServer(s.Routes())
	t.Cleanup(srv.Close)
	return srv
}

func post(t *testing.T, srv *httptest.Server, path, bearer string, body any) (int, map[string]any) {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, srv.URL+path, bytes.NewReader(b))
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

	var out map[string]any
	raw, _ := io.ReadAll(resp.Body)
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	return resp.StatusCode, out
}

// startSession performs the §3 handshake and returns the token.
func startSession(t *testing.T, srv *httptest.Server) string {
	t.Helper()
	status, body := post(t, srv, "/v1/sessions", "", map[string]any{
		"site_key": testSiteKey,
		"page":     map[string]any{"path": "/login"},
		"client": map[string]any{
			"pointer": "fine", "touch": false,
			"viewport": []int{1440, 900}, "tz_offset": -180,
			"reduced_motion": false,
		},
	})
	if status != http.StatusOK {
		t.Fatalf("POST /v1/sessions = %d, want 200", status)
	}
	tok, _ := body["session_token"].(string)
	if tok == "" {
		t.Fatal("response has no session_token")
	}
	return tok
}

// linearBatch is a straight constant-velocity polyline in wire form:
// pts is [x, y, dt] per §2.
func linearBatch(token string, seq int, n int) map[string]any {
	pts := make([][3]int, 0, n)
	for i := 0; i < n; i++ {
		dt := 16
		if i == 0 {
			dt = 0
		}
		pts = append(pts, [3]int{100 + i*10, 120, dt})
	}
	// Each batch is a distinct movement, seconds after the last.
	start := 1200 + seq*5000
	return map[string]any{
		"session_token": token,
		"seq":           seq,
		"sent_at_ms":    start + n*16,
		"page":          map[string]any{"path": "/login", "viewport": []int{1440, 900}},
		"events": []map[string]any{{
			"type": "pointer", "t": start, "src": "mouse", "pts": pts,
		}},
	}
}

func TestSessionsReturnsTokenAndCollectPolicy(t *testing.T) {
	srv := newTestServer(t, policy.ModeMonitor)
	_, body := post(t, srv, "/v1/sessions", "", map[string]any{
		"site_key": testSiteKey,
		"page":     map[string]any{"path": "/login"},
		"client":   map[string]any{"pointer": "fine", "viewport": []int{1440, 900}},
	})

	if _, ok := body["expires_in"]; !ok {
		t.Error("response missing expires_in")
	}
	collect, ok := body["collect"].(map[string]any)
	if !ok {
		t.Fatal("response missing collect policy")
	}
	// Field names are contract surface; the SDK reads these exact keys.
	for _, k := range []string{"pointer_hz", "batch_ms", "types"} {
		if _, ok := collect[k]; !ok {
			t.Errorf("collect missing %q", k)
		}
	}
}

func TestSessionsRejectsUnknownSiteKey(t *testing.T) {
	srv := newTestServer(t, policy.ModeMonitor)
	status, _ := post(t, srv, "/v1/sessions", "", map[string]any{
		"site_key": "pk_wrong",
		"page":     map[string]any{"path": "/login"},
	})
	if status != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", status)
	}
}

func TestTelemetryAccepted(t *testing.T) {
	srv := newTestServer(t, policy.ModeMonitor)
	token := startSession(t, srv)

	status, _ := post(t, srv, "/v1/telemetry", token, linearBatch(token, 0, 40))
	if status != http.StatusAccepted {
		t.Errorf("status = %d, want 202", status)
	}
}

func TestTelemetryForUnknownTokenStillReturns202(t *testing.T) {
	// Telemetry is fire-and-forget and loss is expected (§5). A 4xx
	// here would put a stale SDK into a retry loop.
	srv := newTestServer(t, policy.ModeMonitor)
	status, _ := post(t, srv, "/v1/telemetry", "st_nope", linearBatch("st_nope", 0, 10))
	if status != http.StatusAccepted {
		t.Errorf("status = %d, want 202", status)
	}
}

func TestDecisionsRequiresSecretKey(t *testing.T) {
	// subject_id and action are accepted only from the application
	// server. A browser holding a session token must not reach here.
	srv := newTestServer(t, policy.ModeMonitor)
	token := startSession(t, srv)

	status, _ := post(t, srv, "/v1/decisions", token, map[string]any{
		"session_token": token, "action": "login", "subject_id": "user_1183",
	})
	if status != http.StatusUnauthorized {
		t.Errorf("status = %d with a session token, want 401", status)
	}
}

func TestDecisionsResponseShapeMatchesContract(t *testing.T) {
	srv := newTestServer(t, policy.ModeMonitor)
	token := startSession(t, srv)
	post(t, srv, "/v1/telemetry", token, linearBatch(token, 0, 40))

	status, body := post(t, srv, "/v1/decisions", testSecretKey, map[string]any{
		"session_token": token,
		"action":        "login",
		"subject_id":    "user_1183",
		"context":       map[string]any{"ip": "203.0.113.9", "attempt_n": 2},
	})
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}

	for _, k := range []string{
		"evaluation_id", "decision", "score", "confidence", "reasons", "evidence", "mode",
	} {
		if _, ok := body[k]; !ok {
			t.Errorf("response missing %q", k)
		}
	}

	ev, ok := body["evidence"].(map[string]any)
	if !ok {
		t.Fatal("evidence is not an object")
	}
	for _, k := range []string{"events", "duration_ms"} {
		if _, ok := ev[k]; !ok {
			t.Errorf("evidence missing %q", k)
		}
	}

	// reasons[].code is a stable enumeration (§7).
	reasons, _ := body["reasons"].([]any)
	for _, r := range reasons {
		rm, ok := r.(map[string]any)
		if !ok {
			t.Fatal("reason is not an object")
		}
		if _, ok := rm["code"]; !ok {
			t.Error("reason missing code")
		}
		if _, ok := rm["weight"]; !ok {
			t.Error("reason missing weight")
		}
	}
}

func TestMonitorModeReturnsAllowWithShadow(t *testing.T) {
	srv := newTestServer(t, policy.ModeMonitor)
	token := startSession(t, srv)

	// Enough linear evidence that enforce would block.
	for seq := 0; seq < 8; seq++ {
		post(t, srv, "/v1/telemetry", token, linearBatch(token, seq, 60))
	}

	_, body := post(t, srv, "/v1/decisions", testSecretKey, map[string]any{
		"session_token": token, "action": "login", "subject_id": "u1",
	})

	if body["decision"] != policy.DecisionAllow {
		t.Errorf("decision = %v, want allow in monitor mode", body["decision"])
	}
	if body["shadow_decision"] == nil {
		t.Error("monitor mode must report shadow_decision")
	}
}

func TestUnknownSessionYieldsColdStartNotError(t *testing.T) {
	// The caller is at a risk moment. A missing session is a cold
	// start, which confidence already models — not a 4xx.
	srv := newTestServer(t, policy.ModeMonitor)

	status, body := post(t, srv, "/v1/decisions", testSecretKey, map[string]any{
		"session_token": "st_unknown", "action": "login", "subject_id": "u1",
	})
	if status != http.StatusOK {
		t.Fatalf("status = %d, want 200", status)
	}
	if body["confidence"].(float64) != 0 {
		t.Errorf("confidence = %v, want 0 for an unknown session", body["confidence"])
	}
	if body["decision"] != policy.DecisionAllow {
		t.Errorf("decision = %v, want allow", body["decision"])
	}
}

func TestDecisionsRequiresAction(t *testing.T) {
	srv := newTestServer(t, policy.ModeMonitor)
	status, _ := post(t, srv, "/v1/decisions", testSecretKey, map[string]any{
		"session_token": "st_x",
	})
	if status != http.StatusBadRequest {
		t.Errorf("status = %d without action, want 400", status)
	}
}

func TestLinearPathScoresHigherThanJaggedPath(t *testing.T) {
	// The headline claim of M1: the score moves when the pointer moves
	// differently. If this fails, the slice does nothing.
	srv := newTestServer(t, policy.ModeMonitor)

	linTok := startSession(t, srv)
	for seq := 0; seq < 6; seq++ {
		post(t, srv, "/v1/telemetry", linTok, linearBatch(linTok, seq, 60))
	}
	_, linBody := post(t, srv, "/v1/decisions", testSecretKey, map[string]any{
		"session_token": linTok, "action": "login", "subject_id": "u1",
	})

	humanTok := startSession(t, srv)
	for seq := 0; seq < 6; seq++ {
		post(t, srv, "/v1/telemetry", humanTok, jaggedBatch(humanTok, seq, 60))
	}
	_, humanBody := post(t, srv, "/v1/decisions", testSecretKey, map[string]any{
		"session_token": humanTok, "action": "login", "subject_id": "u2",
	})

	linScore := linBody["score"].(float64)
	humanScore := humanBody["score"].(float64)
	if linScore <= humanScore {
		t.Errorf("linear score %v not greater than jagged score %v", linScore, humanScore)
	}
}

// jaggedBatch is a path that doubles back, as human correction does.
func jaggedBatch(token string, seq, n int) map[string]any {
	pts := make([][3]int, 0, n)
	for i := 0; i < n; i++ {
		dt := 16
		if i == 0 {
			dt = 0
		}
		y := 120
		if i%2 == 0 {
			y = 190
		}
		pts = append(pts, [3]int{100 + i*4, y, dt})
	}
	start := 1200 + seq*5000
	return map[string]any{
		"session_token": token,
		"seq":           seq,
		"sent_at_ms":    start + n*16,
		"page":          map[string]any{"path": "/login", "viewport": []int{1440, 900}},
		"events": []map[string]any{{
			"type": "pointer", "t": start, "src": "mouse", "pts": pts,
		}},
	}
}

func TestUnknownEventTypesAreIgnored(t *testing.T) {
	// The collect policy is server-driven and may change at any time,
	// so an SDK sending a type this build does not know is expected
	// (§7), not a client error.
	srv := newTestServer(t, policy.ModeMonitor)
	token := startSession(t, srv)

	status, _ := post(t, srv, "/v1/telemetry", token, map[string]any{
		"session_token": token,
		"seq":           0,
		"sent_at_ms":    100,
		"page":          map[string]any{"path": "/login", "viewport": []int{1440, 900}},
		"events": []map[string]any{
			{"type": "key", "t": 3410, "phase": "down", "class": "alpha"},
			{"type": "quantum_entanglement", "t": 9000},
		},
	})
	if status != http.StatusAccepted {
		t.Errorf("status = %d, want 202", status)
	}
}

func TestMalformedBodyIsRejected(t *testing.T) {
	srv := newTestServer(t, policy.ModeMonitor)
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/sessions",
		bytes.NewReader([]byte("{not json")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

// ---------------------------------------------------------------
// POST /v1/outcomes
// ---------------------------------------------------------------

func TestOutcomesRequiresSecretKey(t *testing.T) {
	// Labels come from the application server, never the browser. A
	// browser that could report its own outcome could launder its way
	// out of any calibration.
	srv := newTestServer(t, policy.ModeMonitor)
	token := startSession(t, srv)

	status, _ := post(t, srv, "/v1/outcomes", token, map[string]any{
		"evaluation_id": "ev_1", "outcome": "fraud_confirmed",
	})
	if status != http.StatusUnauthorized {
		t.Errorf("status = %d with a session token, want 401", status)
	}
}

func TestOutcomesRejectsUnknownLabel(t *testing.T) {
	// A typo'd label is worse than a missing one: it silently degrades
	// the calibration everything else depends on.
	srv := newTestServer(t, policy.ModeMonitor)
	status, _ := post(t, srv, "/v1/outcomes", testSecretKey, map[string]any{
		"evaluation_id": "ev_1", "outcome": "probably_a_bot_i_think",
	})
	if status != http.StatusBadRequest {
		t.Errorf("status = %d for an unknown outcome, want 400", status)
	}
}

func TestOutcomesRequiresEvaluationID(t *testing.T) {
	srv := newTestServer(t, policy.ModeMonitor)
	status, _ := post(t, srv, "/v1/outcomes", testSecretKey, map[string]any{
		"outcome": "login_success",
	})
	if status != http.StatusBadRequest {
		t.Errorf("status = %d without evaluation_id, want 400", status)
	}
}

func TestOutcomesWithoutStorageIsRefused(t *testing.T) {
	// The test server has no archive. A label with nowhere durable to
	// live must not be accepted: the caller would believe it had
	// reported an outcome, and the loss would be invisible until
	// calibration time.
	srv := newTestServer(t, policy.ModeMonitor)
	status, _ := post(t, srv, "/v1/outcomes", testSecretKey, map[string]any{
		"evaluation_id": "ev_1", "outcome": "login_success",
	})
	if status != http.StatusServiceUnavailable {
		t.Errorf("status = %d with no archive configured, want 503", status)
	}
}

func TestAllContractOutcomeLabelsAccepted(t *testing.T) {
	// The enumeration is contract surface (§3). If one of these is
	// rejected, an integrator's labels vanish.
	for _, label := range []string{
		"login_success", "login_failure", "challenge_passed",
		"challenge_failed", "fraud_confirmed", "user_appealed", "abandoned",
	} {
		if !validOutcomes[label] {
			t.Errorf("contract outcome %q is not accepted", label)
		}
	}
}

// Telemetry keeps mutating session state under the store lock while
// decisions snapshot it. The snapshot must be copied by value inside
// the lock: an earlier version let the *session.State pointer escape
// the With callback and read LastEventMs after it returned, which this
// test catches under -race.
func TestDecisionsConcurrentWithTelemetry(t *testing.T) {
	srv := newTestServer(t, policy.ModeMonitor)
	token := startSession(t, srv)

	const iterations = 40
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for seq := 1; seq <= iterations; seq++ {
			status, _ := post(t, srv, "/v1/telemetry", token, linearBatch(token, seq, 12))
			if status != http.StatusAccepted {
				t.Errorf("POST /v1/telemetry = %d, want 202", status)
				return
			}
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			status, body := post(t, srv, "/v1/decisions", testSecretKey, map[string]any{
				"session_token": token,
				"action":        "login",
			})
			if status != http.StatusOK {
				t.Errorf("POST /v1/decisions = %d, want 200", status)
				return
			}
			if _, ok := body["evaluation_id"]; !ok {
				t.Error("decision response missing evaluation_id")
				return
			}
		}
	}()

	wg.Wait()
}

// Both secret_key endpoints must reject a wrong key regardless of its
// shape — same length, different length, empty — through the single
// constant-time helper.
func TestSecretKeyEndpointsRejectWrongKey(t *testing.T) {
	srv := newTestServer(t, policy.ModeMonitor)
	token := startSession(t, srv)

	wrongKeys := []string{"", "sk_wrong", "sk_tes", testSecretKey + "x", "sk_text"}
	for _, key := range wrongKeys {
		status, _ := post(t, srv, "/v1/decisions", key, map[string]any{
			"session_token": token, "action": "login",
		})
		if status != http.StatusUnauthorized {
			t.Errorf("decisions with key %q = %d, want 401", key, status)
		}
		status, _ = post(t, srv, "/v1/outcomes", key, map[string]any{
			"evaluation_id": "ev_x", "outcome": "login_success",
		})
		if status != http.StatusUnauthorized {
			t.Errorf("outcomes with key %q = %d, want 401", key, status)
		}
	}
}
