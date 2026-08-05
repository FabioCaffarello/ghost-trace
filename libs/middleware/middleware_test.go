package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/FabioCaffarello/ghost-trace/libs/metrics"
)

func TestRequestIDGeneratedAndEchoed(t *testing.T) {
	var seen string
	h := Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = RequestIDFromContext(r.Context())
	}), RequestID())

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/x", nil))

	if len(seen) != 32 {
		t.Errorf("generated id = %q, want 32 hex chars", seen)
	}
	if got := rec.Header().Get(RequestIDHeader); got != seen {
		// The client must be able to quote the id it was served.
		t.Errorf("response header id = %q, context id = %q", got, seen)
	}
}

func TestRequestIDFromUpstreamPreserved(t *testing.T) {
	// An edge-issued id is accepted verbatim so correlation spans hops.
	var seen string
	h := Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = RequestIDFromContext(r.Context())
	}), RequestID())

	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set(RequestIDHeader, "edge-issued-7")
	h.ServeHTTP(httptest.NewRecorder(), req)

	if seen != "edge-issued-7" {
		t.Errorf("id = %q, want the upstream value", seen)
	}
}

func TestRecoveryConvertsPanicTo500(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))
	h := Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}), RequestID(), Recovery(log))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/x", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	out := buf.String()
	if !strings.Contains(out, "boom") || !strings.Contains(out, "request_id") {
		t.Errorf("panic log entry missing panic value or request id:\n%s", out)
	}
}

func TestLoggingEmitsOneEntryPerRequest(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/things", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})
	h := Chain(mux, RequestID(), Logging(log))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/v1/things", nil))

	out := buf.String()
	for _, want := range []string{
		`"method":"POST"`, `"route":"POST /v1/things"`, `"status":202`, `"request_id":"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("log entry missing %s:\n%s", want, out)
		}
	}
	if n := strings.Count(out, "\n"); n != 1 {
		t.Errorf("entries = %d, want exactly 1", n)
	}
}

// scrape renders the registry the way a Prometheus server would see it.
func scrape(t *testing.T, reg *metrics.Registry) string {
	t.Helper()
	rec := httptest.NewRecorder()
	reg.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("scrape status = %d", rec.Code)
	}
	return rec.Body.String()
}

// series reads one counter's value by its labels, out of the registry
// rather than out of the text.
//
// Assertions moved off raw strings deliberately: the client library
// sorts labels, so `{route=...,le=...}` became `{le=...,route=...}`.
// Prometheus does not care about label order and neither should a test
// — a test that does would fail on a change with no observable effect,
// and pass on one with an effect it does not look at.
func counterValue(t *testing.T, reg *metrics.Registry, name string, labels map[string]string) float64 {
	t.Helper()
	families, err := reg.Gatherer().Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	for _, f := range families {
		if f.GetName() != name {
			continue
		}
		for _, m := range f.GetMetric() {
			got := map[string]string{}
			for _, l := range m.GetLabel() {
				got[l.GetName()] = l.GetValue()
			}
			match := len(got) == len(labels)
			for k, v := range labels {
				if got[k] != v {
					match = false
				}
			}
			if match {
				if m.GetCounter() != nil {
					return m.GetCounter().GetValue()
				}
				return float64(m.GetHistogram().GetSampleCount())
			}
		}
	}
	t.Fatalf("no series %s%v in the registry", name, labels)
	return 0
}

func TestMetricsCountsByRouteAndStatus(t *testing.T) {
	reg := metrics.New()
	m := NewMetrics(reg)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ok", func(w http.ResponseWriter, r *http.Request) {})
	h := Chain(mux, m.Collect())

	for i := 0; i < 3; i++ {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/ok", nil))
	}
	// A path no route claims must collapse into one bounded label, not
	// mint a series per probe.
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/scanner-probe-1", nil))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/scanner-probe-2", nil))

	for _, tc := range []struct {
		name   string
		labels map[string]string
		want   float64
	}{
		{"ghosttrace_http_requests_total", map[string]string{"route": "GET /ok", "status": "200"}, 3},
		{"ghosttrace_http_requests_total", map[string]string{"route": "unmatched", "status": "404"}, 2},
		{"ghosttrace_http_request_duration_ms", map[string]string{"route": "GET /ok"}, 3},
	} {
		if got := counterValue(t, reg, tc.name, tc.labels); got != tc.want {
			t.Errorf("%s%v = %v, want %v", tc.name, tc.labels, got, tc.want)
		}
	}

	if strings.Contains(scrape(t, reg), "scanner-probe") {
		t.Error("raw unmatched path leaked into a metric label")
	}
}

func TestExpositionDidNotChange(t *testing.T) {
	// The hand-written encoder these series used to come from produced
	// exactly these lines. Adopting a library to emit them is only free
	// if nothing downstream can tell — so this asserts the raw text,
	// which is the thing a scraper and a stored query actually see.
	reg := metrics.New()
	m := NewMetrics(reg)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ok", func(w http.ResponseWriter, r *http.Request) {})
	h := Chain(mux, m.Collect())
	for i := 0; i < 3; i++ {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/ok", nil))
	}
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/scanner-probe-1", nil))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/scanner-probe-2", nil))

	out := scrape(t, reg)
	for _, want := range []string{
		`ghosttrace_http_requests_total{route="GET /ok",status="200"} 3`,
		`ghosttrace_http_requests_total{route="unmatched",status="404"} 2`,
		`ghosttrace_http_request_duration_ms_count{route="GET /ok"} 3`,
		`ghosttrace_http_request_duration_ms_bucket{route="GET /ok",le="+Inf"} 3`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("exposition changed; missing %q:\n%s", want, out)
		}
	}
}

func TestSuccessiveScrapesOfAnUnchangedRegistryAreIdentical(t *testing.T) {
	// A scrape that reorders between reads makes every diff noise and
	// hides the one that matters.
	reg := metrics.New()
	m := NewMetrics(reg)
	mux := http.NewServeMux()
	for _, p := range []string{"GET /a", "GET /b", "GET /c"} {
		mux.HandleFunc(p, func(w http.ResponseWriter, r *http.Request) {})
	}
	h := Chain(mux, m.Collect())
	for _, p := range []string{"/a", "/b", "/c", "/a", "/b"} {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", p, nil))
	}
	first := scrape(t, reg)
	second := scrape(t, reg)
	if first != second {
		t.Error("successive scrapes of an unchanged registry differ")
	}
}

func TestMetricsHandlerServesExposition(t *testing.T) {
	reg := metrics.New()
	m := NewMetrics(reg)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /x", func(w http.ResponseWriter, r *http.Request) {})
	Chain(mux, m.Collect()).ServeHTTP(
		httptest.NewRecorder(), httptest.NewRequest("GET", "/x", nil))

	rec := httptest.NewRecorder()
	reg.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("content type = %q", ct)
	}
	if !strings.Contains(rec.Body.String(), "ghosttrace_http_requests_total") {
		t.Error("exposition body missing counter family")
	}
}
