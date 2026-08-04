package httpmw

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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

func TestMetricsCountsByRouteAndStatus(t *testing.T) {
	m := NewMetrics()
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

	var buf bytes.Buffer
	if err := m.Encode(&buf); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	out := buf.String()

	for _, want := range []string{
		`ghosttrace_http_requests_total{route="GET /ok",status="200"} 3`,
		`ghosttrace_http_requests_total{route="unmatched",status="404"} 2`,
		`ghosttrace_http_request_duration_ms_count{route="GET /ok"} 3`,
		`ghosttrace_http_request_duration_ms_bucket{route="GET /ok",le="+Inf"} 3`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("exposition missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "scanner-probe") {
		t.Error("raw unmatched path leaked into a metric label")
	}
}

func TestMetricsEncodeIsDeterministic(t *testing.T) {
	m := NewMetrics()
	for i := 0; i < 5; i++ {
		m.inc(fmt.Sprintf("GET /r%d", i%3), 200+i)
		m.observe(fmt.Sprintf("GET /r%d", i%3), float64(i))
	}
	var a, b bytes.Buffer
	_ = m.Encode(&a)
	_ = m.Encode(&b)
	if a.String() != b.String() {
		t.Error("successive scrapes of an unchanged registry differ")
	}
}

func TestMetricsHandlerServesExposition(t *testing.T) {
	m := NewMetrics()
	m.inc("GET /x", 200)
	rec := httptest.NewRecorder()
	m.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/metrics", nil))
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("content type = %q", ct)
	}
	if !strings.Contains(rec.Body.String(), "ghosttrace_http_requests_total") {
		t.Error("exposition body missing counter family")
	}
}
