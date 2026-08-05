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
	"strings"
	"syscall"
	"time"

	"github.com/FabioCaffarello/ghost-trace/tools/loadgen"
)

func main() {
	var (
		collector = flag.String("collector", envOr("GT_COLLECTOR", "http://127.0.0.1:8080"),
			"collector base URL")
		scenario = flag.String("scenario", "session",
			"healthz | session — what one scheduled arrival does")
		rate = flag.Float64("rate", 50, "arrivals per second (a schedule, not a target)")
		dur  = flag.Duration("duration", 30*time.Second, "how long to keep issuing")
		// Workers bounds in-flight requests. Set it well above
		// rate x expected-latency: if it binds, the run measures this
		// flag rather than the system, which is why deficit_p99 is
		// reported and checked below.
		workers  = flag.Int("workers", 512, "maximum in-flight requests")
		siteKey  = flag.String("site-key", envOr("GT_SITE_KEY", "pk_demo"), "tenant site key")
		events   = flag.Int("events", 8, "events per telemetry batch")
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
		Workers  int    `json:"workers"`
		Events   int    `json:"events_per_batch"`
	}{rep, *scenario, *collector, cfg.Workers, *events}

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
		body := fmt.Sprintf(
			`{"site_key":%q,"page":{"path":"/login"},"client":{"pointer":"fine"}}`, siteKey)
		status, token, err := sendJSON(ctx, c, http.MethodPost, base+"/v1/sessions", body, "")
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
			[]byte(telemetry(tok, events)), tok)
	}
}

// telemetry builds a batch. The event times jitter so that every session
// does not present the collector with byte-identical work — a load test
// whose payloads are all the same measures a cache it did not mean to
// build.
func telemetry(token string, events int) string {
	var b strings.Builder
	fmt.Fprintf(&b, `{"session_token":%q,"seq":1,"sent_at_ms":900,`, token)
	b.WriteString(`"page":{"path":"/login"},"events":[`)
	for i := range events {
		if i > 0 {
			b.WriteByte(',')
		}
		t := 100 + i*17 + rand.IntN(7)
		if i%2 == 0 {
			fmt.Fprintf(&b, `{"type":"pointer","t":%d,"x":%d,"y":%d}`,
				t, 300+rand.IntN(200), 200+rand.IntN(120))
			continue
		}
		fmt.Fprintf(&b, `{"type":"key","t":%d,"phase":"down","class":"alpha","target":"f"}`, t)
	}
	b.WriteString(`]}`)
	return b.String()
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
