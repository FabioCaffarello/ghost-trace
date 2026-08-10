// Command loadgen drives the topology on a schedule and reports what
// the load actually experienced.
//
// Stdlib only, deliberately. This is a measuring instrument, and the
// fewer moving parts between the schedule and the socket the fewer
// things there are to blame when a number looks wrong.
//
// The output is JSON on stdout and a human summary on stderr, so a run
// can be piped into a results file without losing the reading a person
// needs while watching it.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/FabioCaffarello/ghost-trace/libs/wire"
	"github.com/FabioCaffarello/ghost-trace/tools/loadgen"
)

// decisionBudgetMs is the figure every published decision latency is
// measured against. Stated in the architecture contract; reproduced here
// so a run says what it was aiming at rather than leaving the reader to
// find out.
const decisionBudgetMs = 80

func main() {
	var (
		collector = flag.String("collector", envOr("GT_COLLECTOR", "http://127.0.0.1:8080"),
			"collector base URL")
		scenario = flag.String("scenario", "session",
			"healthz | session | decision — what one scheduled arrival does")
		rate = flag.Float64("rate", 50, "arrivals per second (a schedule, not a target)")
		dur  = flag.Duration("duration", 30*time.Second, "how long to keep issuing")
		// Workers bounds in-flight requests. Set it well above
		// rate x expected-latency: if it binds, the run measures this
		// flag rather than the system, which is why deficit_p99 is
		// reported and checked below.
		workers = flag.Int("workers", 512, "maximum in-flight requests")
		siteKey = flag.String("site-key", envOr("GT_SITE_KEY", "pk_demo"), "tenant site key")
		events  = flag.Int("events", 8, "events per telemetry batch")
		engine  = flag.String("engine", envOr("GT_ENGINE", "http://127.0.0.1:8082"),
			"decision-engine base URL")
		secret = flag.String("secret-key", envOr("GT_SECRET_KEY", "sk_demo"),
			"tenant secret key, for the server-to-server endpoints")
		warm = flag.Int("warm-sessions", 500,
			"sessions to open and feed before a decision run, so each arrival is "+
				"one /v1/decisions call against state that already exists")
		outPath  = flag.String("out", "", "write the JSON report here as well as stdout")
		deficitB = flag.Float64("max-deficit-ms", 50,
			"refuse the run if the driver fell further behind its own schedule than this")
	)
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        4096,
			MaxIdleConnsPerHost: 4096,
			MaxConnsPerHost:     0,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	var do loadgen.Do
	switch *scenario {
	case "healthz":
		do = healthz(client, *collector)
	case "session":
		do = sessionFlow(client, *collector, *siteKey, *events)
	case "decision":
		// ISOLATING THE DECISION. The 80ms budget is stated for a
		// decision, not for a session's whole first round trip, so an
		// arrival here must be one /v1/decisions call and nothing else.
		//
		// That needs sessions to already exist, with state in them —
		// deciding about an empty session is a different measurement,
		// and a cheaper one. They are opened and fed before the clock
		// starts, and the run then draws from the pool.
		fmt.Fprintf(os.Stderr, "warming %d sessions...\n", *warm)
		tokens, err := warmSessions(ctx, client, *collector, *siteKey, *events, *warm)
		if err != nil {
			fmt.Fprintln(os.Stderr, "warm-up:", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "  %d sessions ready\n", len(tokens))
		do = decisionOnly(client, *engine, *secret, tokens)
	default:
		fmt.Fprintf(os.Stderr, "unknown scenario %q\n", *scenario)
		os.Exit(2)
	}

	cfg := loadgen.Config{Rate: *rate, Duration: *dur, Workers: *workers}
	fmt.Fprintf(os.Stderr, "loadgen: %s at %.0f/s for %s (workers %d)\n",
		*scenario, cfg.Rate, cfg.Duration, cfg.Workers)

	results := loadgen.Run(ctx, cfg, do)
	rep := loadgen.Summarise(results, cfg)

	out := struct {
		loadgen.Report
		Scenario string `json:"scenario"`
		Target   string `json:"target"`
		Budget   int    `json:"budget_ms"`
		Workers  int    `json:"workers"`
		Events   int    `json:"events_per_batch"`
	}{rep, *scenario, *collector, decisionBudgetMs, cfg.Workers, *events}

	enc, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "encode:", err)
		os.Exit(1)
	}
	fmt.Println(string(enc))
	if *outPath != "" {
		if err := os.WriteFile(*outPath, append(enc, '\n'), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "write:", err)
			os.Exit(1)
		}
	}

	fmt.Fprintf(os.Stderr,
		"\n  requests %d (%d ok, %d failed) · achieved %.0f/s of %.0f/s\n"+
			"  service   p50 %7.2fms  p99 %7.2fms   <- what a closed-loop driver reports\n"+
			"  response  p50 %7.2fms  p99 %7.2fms   <- what the load experienced\n"+
			"  deficit                p99 %7.2fms\n"+
			"  coordinated omission       %7.2fms\n",
		rep.Requests, rep.OK, rep.Failed, rep.Achieved, rep.Intended,
		rep.ServiceP50, rep.ServiceP99, rep.ResponseP50, rep.ResponseP99,
		rep.DeficitP99, rep.Omission)

	// A run the driver could not sustain is not a measurement of the
	// system. Refusing is the same discipline as every topology gate
	// here: reporting a number taken under a bound nobody noticed is
	// worse than reporting nothing.
	if rep.DeficitP99 > *deficitB {
		fmt.Fprintf(os.Stderr,
			"\n  REFUSED: the driver fell %.1fms behind its own schedule (limit %.1fms).\n"+
				"  This run measured the -workers bound, not the system. Raise -workers,\n"+
				"  lower -rate, or run the driver somewhere with more to give.\n",
			rep.DeficitP99, *deficitB)
		os.Exit(1)
	}
}

// warmSessions opens sessions and posts one telemetry batch into each,
// so the decisions measured afterwards run against sessions that have
// something to decide about.
func warmSessions(ctx context.Context, c *http.Client, base, siteKey string,
	events, n int) ([]string, error) {

	tokens := make([]string, 0, n)
	for range n {
		st, out := post(ctx, c, base+"/v1/sessions", string(sessionBody(siteKey)), "")
		if st != http.StatusOK {
			return nil, fmt.Errorf("open session: status %v", st)
		}
		tok := field(out, "session_token")
		if tok == "" {
			return nil, fmt.Errorf("open session: no token in response")
		}
		if _, err := send(ctx, c, http.MethodPost, base+"/v1/telemetry",
			telemetry(tok, events), tok); err != nil {
			return nil, fmt.Errorf("feed session: %w", err)
		}
		tokens = append(tokens, tok)
	}
	if len(tokens) == 0 {
		return nil, fmt.Errorf("no sessions could be opened")
	}
	return tokens, nil
}

// decisionOnly is one /v1/decisions call against a session drawn from
// the warm pool.
//
// Tokens are picked at random rather than in order so the engine is not
// handed a predictable sequence — a KV read that walks the pool in the
// same order every time is a different access pattern from a real one,
// and the KV is the part of this path worth measuring.
func decisionOnly(c *http.Client, engine, secret string, tokens []string) loadgen.Do {
	return func(ctx context.Context) (int, error) {
		tok := tokens[rand.IntN(len(tokens))]
		body, err := json.Marshal(wire.DecisionsRequest{SessionToken: tok, Action: "login"})
		if err != nil {
			return 0, err
		}
		return send(ctx, c, http.MethodPost, engine+"/v1/decisions", body, secret)
	}
}

func post(ctx context.Context, c *http.Client, url, body, bearer string) (int, string) {
	st, out, err := sendJSON(ctx, c, http.MethodPost, url, body, bearer)
	if err != nil {
		return 0, ""
	}
	return st, out
}

func healthz(c *http.Client, base string) loadgen.Do {
	return func(ctx context.Context) (int, error) {
		return send(ctx, c, http.MethodGet, base+"/healthz", nil, "")
	}
}

// sessionFlow is one browser's worth of work: open a session, then post
// a telemetry batch against it.
//
// Both halves are inside ONE scheduled arrival on purpose. They are not
// independent — the telemetry cannot exist without the session — and
// scheduling them separately would let the driver post telemetry for
// sessions it had not opened, measuring an error path and calling it
// throughput.
func sessionFlow(c *http.Client, base, siteKey string, events int) loadgen.Do {
	return func(ctx context.Context) (int, error) {
		status, token, err := sendJSON(ctx, c, http.MethodPost, base+"/v1/sessions",
			string(sessionBody(siteKey)), "")
		if err != nil {
			return status, err
		}
		if status != http.StatusOK {
			return status, nil
		}
		tok := field(token, "session_token")
		if tok == "" {
			return status, fmt.Errorf("no session_token in response")
		}
		return send(ctx, c, http.MethodPost, base+"/v1/telemetry",
			telemetry(tok, events), tok)
	}
}

// sessionBody builds the handshake from the SAME Go types the servers
// decode. This driver used to hand-roll its JSON with fmt.Sprintf, and
// paid for it in the worst currency available to a measuring
// instrument: its pointer events carried `x`/`y` — fields the wire
// does not have — so the collector silently dropped them, `pts` stayed
// empty, and every published load curve measured batches doing zero
// pointer-feature work while looking exactly like batches that did it
// all. Building from libs/wire makes that drift a compile error.
func sessionBody(siteKey string) []byte {
	b, err := json.Marshal(wire.SessionsRequest{
		SiteKey: siteKey,
		Page:    wire.PageRef{Path: "/login"},
		Client: wire.ClientHints{
			Pointer:  "fine",
			Viewport: []int{1440, 900},
		},
	})
	if err != nil {
		panic(err) // a struct literal that cannot marshal is a programming error
	}
	return b
}

// telemetry builds a batch. The event times and pointer paths jitter so
// that every session does not present the collector with byte-identical
// work — a load test whose payloads are all the same measures a cache
// it did not mean to build.
//
// Pointer events carry a real polyline in `pts`, sized like the ones
// the SDK flushes, because the per-event feature update is the work the
// collector does under the session lock — a batch without it measures
// a lighter system than the one in production.
func telemetry(token string, events int) []byte {
	evs := make([]wire.TelemetryEvent, 0, events)
	for i := range events {
		t := uint32(100 + i*17 + rand.IntN(7))
		if i%2 == 0 {
			x := int32(300 + rand.IntN(200))
			y := int32(200 + rand.IntN(120))
			pts := make([][3]int32, 6)
			for j := range pts {
				var dt int32
				if j > 0 {
					x += int32(rand.IntN(9) - 4)
					y += int32(rand.IntN(9) - 4)
					dt = int32(45 + rand.IntN(20))
				}
				pts[j] = [3]int32{x, y, dt}
			}
			evs = append(evs, wire.TelemetryEvent{Type: "pointer", T: t, Src: "mouse", Pts: pts})
			continue
		}
		evs = append(evs, wire.TelemetryEvent{
			Type: "key", T: t, Phase: "down", KeyClass: "alpha", Target: "f"})
	}
	b, err := json.Marshal(wire.TelemetryBatch{
		SessionToken: token,
		Seq:          1,
		SentAtMs:     900,
		Page:         wire.TelemetryPage{Path: "/login", Viewport: []int{1440, 900}},
		Events:       evs,
	})
	if err != nil {
		panic(err)
	}
	return b
}

func send(ctx context.Context, c *http.Client, method, url string, body []byte, bearer string) (int, error) {
	status, _, err := sendJSON(ctx, c, method, url, string(body), bearer)
	return status, err
}

func sendJSON(ctx context.Context, c *http.Client, method, url, body, bearer string) (int, string, error) {
	var rdr io.Reader
	if body != "" {
		rdr = bytes.NewReader([]byte(body))
	}
	req, err := http.NewRequestWithContext(ctx, method, url, rdr)
	if err != nil {
		return 0, "", err
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := c.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	return resp.StatusCode, string(raw), nil
}

// field pulls one string field out of a JSON object without modelling
// the whole response. The tool deliberately does not import libs/wire:
// a load driver that shares types with the server under test cannot
// detect a contract the server broke.
func field(doc, name string) string {
	var m map[string]any
	if err := json.Unmarshal([]byte(doc), &m); err != nil {
		return ""
	}
	s, _ := m[name].(string)
	return s
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
