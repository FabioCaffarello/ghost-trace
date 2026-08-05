// The A/B, over HTTP, against both services running for real.
//
// PR-2.3a proved the snapshot MAPPING preserves decisions, in memory.
// PR-2.3b proved it through a live KV bucket, still in one process.
// This proves it through the topology a client actually meets: a
// session created and fed on the collector, then the same decision
// request put to the collector and to the decision engine, and the two
// answers compared field by field.
//
// The file also holds one assertion that is not about the shadow at
// all: that the demo host actually reaches the engine. Fail-open (§5)
// means a demo wired to an unreachable engine answers "allow" forever
// and looks perfectly healthy — which is what happened the first time
// this topology ran, because the demo was handed a host address for a
// call it makes server-side from inside the compose network.
//
// It needs the services. Without the URLs it SKIPS rather than passes,
// because a shadow test that quietly does nothing is the vacuous green
// this repository keeps finding.
//
//	make shadow-http
package shadow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

const (
	// How long to wait for a snapshot to travel from the collector to
	// the engine. Generous because this is not a latency measurement —
	// PR-2.6 is — and a flaky shadow test would get muted.
	propagationBudget = 15 * time.Second
	pollInterval      = 250 * time.Millisecond
)

type endpoints struct {
	collector, engine, siteKey, secretKey string
}

func fromEnv(t *testing.T) endpoints {
	t.Helper()
	e := endpoints{
		collector: os.Getenv("GT_COLLECTOR_URL"),
		engine:    os.Getenv("GT_ENGINE_URL"),
		siteKey:   envOr("GT_SITE_KEY", "pk_demo"),
		secretKey: envOr("GT_SECRET_KEY", "sk_demo"),
	}
	if e.collector == "" || e.engine == "" {
		t.Skip("GT_COLLECTOR_URL and GT_ENGINE_URL not set — bring both services up " +
			"(docker compose --profile core up) and run `make shadow-http`")
	}
	return e
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func post(t *testing.T, url, bearer string, body any) (int, map[string]any) {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		t.Fatalf("request %s: %v", url, err)
	}
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, _ := io.ReadAll(resp.Body)
	var out map[string]any
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	return resp.StatusCode, out
}

// linearBatch is the naive-automation shape: a straight, constant
// velocity drag with an injected field. It scores high, which is where
// a disagreement between the two services would actually show — two
// empty sessions agree trivially.
func linearBatch(token string, seq int) map[string]any {
	pts := make([][3]int, 0, 40)
	for i := 0; i < 40; i++ {
		dt := 16
		if i == 0 {
			dt = 0
		}
		pts = append(pts, [3]int{100 + i*10, 120, dt})
	}
	start := 1200 + seq*5000
	return map[string]any{
		"session_token": token,
		"seq":           seq,
		"sent_at_ms":    start + 40*16,
		"page":          map[string]any{"path": "/login", "viewport": []int{1440, 900}},
		"events": []map[string]any{
			{"type": "pointer", "t": start, "src": "mouse", "pts": pts},
			{"type": "key", "t": start + 40, "phase": "down", "class": "alpha", "target": "f_1"},
			{"type": "key", "t": start + 70, "phase": "up", "class": "alpha", "target": "f_1"},
			{"type": "form", "t": start + 90, "action": "injected", "target": "f_1"},
		},
	}
}

func TestTheEngineDecidesWhatTheCollectorDecides(t *testing.T) {
	e := fromEnv(t)

	status, body := post(t, e.collector+"/v1/sessions", "", map[string]any{
		"site_key": e.siteKey,
		"page":     map[string]any{"path": "/login"},
		"client":   map[string]any{"pointer": "fine"},
	})
	if status != http.StatusOK {
		t.Fatalf("POST /v1/sessions on the collector = %d, want 200 (body %v)", status, body)
	}
	token, _ := body["session_token"].(string)
	if token == "" {
		t.Fatal("collector issued no session_token")
	}

	for seq := 1; seq <= 3; seq++ {
		if status, body := post(t, e.collector+"/v1/telemetry", token,
			linearBatch(token, seq)); status != http.StatusAccepted {
			t.Fatalf("POST /v1/telemetry seq %d = %d, want 202 (body %v)", seq, status, body)
		}
	}

	decide := func(base string) map[string]any {
		t.Helper()
		status, body := post(t, base+"/v1/decisions", e.secretKey, map[string]any{
			"session_token": token, "action": "login", "subject_id": "u_shadow",
		})
		if status != http.StatusOK {
			t.Fatalf("POST /v1/decisions on %s = %d, want 200 (body %v)", base, status, body)
		}
		return body
	}

	// Wait for the snapshot to reach the engine. A miss here is a
	// FAILURE and not a skip: propagation is part of what this proves,
	// and an engine that never sees the session would otherwise agree
	// with nothing while looking healthy.
	deadline := time.Now().Add(propagationBudget)
	var engine map[string]any
	for {
		engine = decide(e.engine)
		if ev, _ := engine["evidence"].(map[string]any); ev != nil {
			if n, _ := ev["events"].(float64); n > 0 {
				break
			}
		}
		if time.Now().After(deadline) {
			t.Fatalf("the engine still reports no evidence after %s — the snapshot "+
				"never arrived, so every decision it makes is a cold start", propagationBudget)
		}
		time.Sleep(pollInterval)
	}

	// Asked last, so the collector's live state is at least as advanced
	// as the snapshot the engine just judged. If anything, that biases
	// AGAINST agreement, which is the right direction for a test whose
	// failure mode is a false pass.
	collector := decide(e.collector)

	for _, f := range []string{"decision", "shadow_decision", "mode"} {
		if got, want := engine[f], collector[f]; got != want {
			t.Errorf("%s: engine %v, collector %v", f, got, want)
		}
	}
	for _, f := range []string{"score", "confidence"} {
		g, _ := engine[f].(float64)
		w, _ := collector[f].(float64)
		// Both sides round to three decimals on the wire, and the
		// snapshot stores float32 (ADR-0004 measured the loss under
		// 1e-6 relative). One rounding step apart is the most the split
		// may cost; anything larger is a difference in the judgement.
		if diff := g - w; diff > 0.001 || diff < -0.001 {
			t.Errorf("%s: engine %v, collector %v", f, g, w)
		}
	}

	// Reason CODES must match; weights ride along with the scores above.
	if got, want := reasonCodes(engine), reasonCodes(collector); got != want {
		t.Errorf("reasons: engine %s, collector %s", got, want)
	}

	// The evaluation ids MUST differ — each call mints its own, and two
	// equal ids would mean something is being reused across services.
	if engine["evaluation_id"] == collector["evaluation_id"] {
		t.Errorf("both services returned evaluation_id %v; ids must be minted per call",
			engine["evaluation_id"])
	}

	// A session both sides saw as empty would agree trivially, which is
	// the false pass this whole test is arranged to avoid.
	if ev, _ := collector["evidence"].(map[string]any); ev != nil {
		if n, _ := ev["events"].(float64); n == 0 {
			t.Error("the collector reports no evidence either; the batches never landed " +
				"and the comparison above compared two empty sessions")
		}
	}
}

func reasonCodes(body map[string]any) string {
	rs, _ := body["reasons"].([]any)
	out := ""
	for _, r := range rs {
		m, _ := r.(map[string]any)
		out += fmt.Sprintf("%v ", m["code"])
	}
	return out
}

func TestTheDemoHostReachesTheEngine(t *testing.T) {
	demo := os.Getenv("GT_DEMO_URL")
	if demo == "" {
		t.Skip("GT_DEMO_URL not set — bring the topology up and run `make shadow-http`")
	}
	e := fromEnv(t)

	status, body := post(t, e.collector+"/v1/sessions", "", map[string]any{
		"site_key": e.siteKey,
		"page":     map[string]any{"path": "/login"},
		"client":   map[string]any{"pointer": "fine"},
	})
	if status != http.StatusOK {
		t.Fatalf("POST /v1/sessions = %d, want 200 (body %v)", status, body)
	}
	token, _ := body["session_token"].(string)

	status, out := post(t, demo+"/demo/login", "", map[string]any{
		"session_token": token, "username": "alice",
	})
	if status != http.StatusOK {
		t.Fatalf("POST /demo/login = %d, want 200 (body %v)", status, out)
	}

	// The tell. Fail-open is CORRECT behaviour for an unreachable
	// engine; it is a misconfiguration only because the engine is right
	// there. A demo that always allows demonstrates nothing, and
	// nothing else in this repository would notice.
	if out["mode"] == "fail-open" {
		t.Fatalf("the demo host failed open: it cannot reach the decision engine. "+
			"A server-side call needs a service address, not the browser-facing one "+
			"(got %v)", out)
	}
	if id, _ := out["evaluation_id"].(string); id == "" {
		t.Error("no evaluation_id: the decision did not come from the engine")
	}
}

// fingerprint reads the tenant-set fingerprint a service publishes.
func fingerprint(t *testing.T, base string) string {
	t.Helper()
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Get(base + "/metrics")
	if err != nil {
		t.Fatalf("GET %s/metrics: %v", base, err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(resp.Body)

	for _, line := range strings.Split(string(raw), "\n") {
		if !strings.HasPrefix(line, "ghosttrace_tenant_registry_info{") {
			continue
		}
		open := strings.Index(line, `fingerprint="`)
		if open < 0 {
			continue
		}
		rest := line[open+len(`fingerprint="`):]
		if close := strings.Index(rest, `"`); close >= 0 {
			return rest[:close]
		}
	}
	t.Fatalf("%s publishes no tenant_registry_info; the registries cannot be compared", base)
	return ""
}

func TestBothServicesServeTheSameTenants(t *testing.T) {
	// Two registries that disagree about who exists each behave
	// correctly on their own and wrongly together: a session opened for
	// a tenant the engine has never heard of gets a decision attributed
	// to nobody, and NO REQUEST FAILS on the way there. Held by nothing
	// but a comment in compose.yml until this existed.
	//
	// The fingerprint covers ids and site keys, not secrets — those are
	// checked far more directly by the tests above, which present one
	// application key to both services and would get a 401.
	e := fromEnv(t)

	collector := fingerprint(t, e.collector)
	engine := fingerprint(t, e.engine)
	if collector != engine {
		t.Errorf("the collector serves tenant set %s and the engine serves %s; "+
			"a session opened on one can be judged by the other under a tenant it "+
			"does not know", collector, engine)
	}
}
