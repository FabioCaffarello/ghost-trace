package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// TestMetrics_IncAggregatesPerPathAndStatus verifies the registry
// accumulates counts per-(path, status) tuple — distinct tuples are
// tracked separately, identical tuples sum.
func TestMetrics_IncAggregatesPerPathAndStatus(t *testing.T) {
	m := NewMetrics()
	m.Inc("/healthz", 200)
	m.Inc("/healthz", 200)
	m.Inc("/healthz", 200)
	m.Inc("/v1/morphology", 200)
	m.Inc("/v1/morphology", 500)

	var buf bytes.Buffer
	if err := m.Encode(&buf); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		`ghosttrace_httpapi_requests_total{path="/healthz",status="200"} 3`,
		`ghosttrace_httpapi_requests_total{path="/v1/morphology",status="200"} 1`,
		`ghosttrace_httpapi_requests_total{path="/v1/morphology",status="500"} 1`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\nfull output:\n%s", want, out)
		}
	}
}

// TestMetrics_EncodeIncludesHelpAndType verifies the Prometheus
// text-format header lines (HELP + TYPE) precede the metric samples.
// Required for compliant consumption by the prometheus scraper.
func TestMetrics_EncodeIncludesHelpAndType(t *testing.T) {
	m := NewMetrics()
	m.Inc("/healthz", 200)
	var buf bytes.Buffer
	_ = m.Encode(&buf)
	out := buf.String()
	if !strings.Contains(out, "# HELP ghosttrace_httpapi_requests_total") {
		t.Errorf("missing HELP line:\n%s", out)
	}
	if !strings.Contains(out, "# TYPE ghosttrace_httpapi_requests_total counter") {
		t.Errorf("missing TYPE line:\n%s", out)
	}
}

// TestMetrics_EncodeDeterministicOrdering verifies the encode output
// is sorted by (path, status) ascending — successive scrapes against
// an unchanged registry produce byte-identical bytes.
func TestMetrics_EncodeDeterministicOrdering(t *testing.T) {
	m := NewMetrics()
	// Insert in non-sorted order.
	m.Inc("/zebra", 200)
	m.Inc("/alpha", 500)
	m.Inc("/alpha", 200)
	var buf1, buf2 bytes.Buffer
	_ = m.Encode(&buf1)
	_ = m.Encode(&buf2)
	if buf1.String() != buf2.String() {
		t.Errorf("Encode not deterministic:\nfirst:\n%s\nsecond:\n%s", buf1.String(), buf2.String())
	}
	// Check sort: /alpha lines come before /zebra; status 200 before 500.
	lines := strings.Split(strings.TrimRight(buf1.String(), "\n"), "\n")
	var sampleLines []string
	for _, l := range lines {
		if strings.HasPrefix(l, "ghosttrace_") {
			sampleLines = append(sampleLines, l)
		}
	}
	if len(sampleLines) != 3 {
		t.Fatalf("expected 3 sample lines, got %d:\n%s", len(sampleLines), buf1.String())
	}
	if !strings.Contains(sampleLines[0], `path="/alpha",status="200"`) {
		t.Errorf("first line not alpha+200: %s", sampleLines[0])
	}
	if !strings.Contains(sampleLines[1], `path="/alpha",status="500"`) {
		t.Errorf("second line not alpha+500: %s", sampleLines[1])
	}
	if !strings.Contains(sampleLines[2], `path="/zebra",status="200"`) {
		t.Errorf("third line not zebra+200: %s", sampleLines[2])
	}
}

// TestMetrics_EncodeEmptyRegistryStillEmitsHeader verifies an empty
// registry emits the HELP + TYPE preamble but no sample lines.
// Operators scraping a freshly-started service receive a valid (empty)
// metric set rather than 404.
func TestMetrics_EncodeEmptyRegistry(t *testing.T) {
	m := NewMetrics()
	var buf bytes.Buffer
	_ = m.Encode(&buf)
	out := buf.String()
	if !strings.Contains(out, "# HELP") || !strings.Contains(out, "# TYPE") {
		t.Errorf("missing HELP/TYPE preamble on empty registry:\n%s", out)
	}
	if strings.Contains(out, "ghosttrace_httpapi_requests_total{") {
		t.Errorf("empty registry emitted a sample line:\n%s", out)
	}
}

// TestMetrics_EncodeEscapesLabelValues verifies the label-value
// escaping rules (backslash, double-quote, newline) per Prometheus
// exposition spec.
func TestMetrics_EncodeEscapesLabelValues(t *testing.T) {
	m := NewMetrics()
	m.Inc(`/contains "quote"`, 200)
	m.Inc("/contains\\backslash", 200)
	m.Inc("/contains\nnewline", 200)

	var buf bytes.Buffer
	_ = m.Encode(&buf)
	out := buf.String()
	if !strings.Contains(out, `path="/contains \"quote\""`) {
		t.Errorf("double-quote not escaped:\n%s", out)
	}
	if !strings.Contains(out, `path="/contains\\backslash"`) {
		t.Errorf("backslash not escaped:\n%s", out)
	}
	if !strings.Contains(out, `path="/contains\nnewline"`) {
		t.Errorf("newline not escaped:\n%s", out)
	}
}

// TestMetrics_ConcurrentIncIsSafe verifies Inc is safe under
// concurrent invocation — required for high-traffic httpapi
// deployments where many goroutines may increment simultaneously.
// Test discipline: 1000 increments across 10 goroutines; the final
// count MUST equal 1000.
func TestMetrics_ConcurrentIncIsSafe(t *testing.T) {
	m := NewMetrics()
	const goroutines = 10
	const perGoroutine = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < perGoroutine; j++ {
				m.Inc("/concurrent", 200)
			}
		}()
	}
	wg.Wait()
	var buf bytes.Buffer
	_ = m.Encode(&buf)
	want := `ghosttrace_httpapi_requests_total{path="/concurrent",status="200"} 1000`
	if !strings.Contains(buf.String(), want) {
		t.Errorf("expected counter = 1000 after concurrent Inc:\n%s", buf.String())
	}
}

// TestMetricsHTTP_ServesPrometheusFormat exercises the /metrics
// endpoint end-to-end: requests increment the counter; the response
// body contains the Prometheus-format counter.
func TestMetricsHTTP_ServesPrometheusFormat(t *testing.T) {
	doAppend, _ := stubAppendFunc(nil)
	metrics := NewMetrics()
	h := MustNew(doAppend, nil, WithMetrics(metrics))

	// Two requests to /healthz to bump the counter to 2.
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
	}

	// Scrape /metrics.
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("Content-Type: got %q, want text/plain prefix", ct)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `ghosttrace_httpapi_requests_total{path="/healthz",status="200"} 2`) {
		t.Errorf("missing /healthz counter (should be 2 after 2 requests):\n%s", body)
	}
	// /metrics scrape itself increments by 1 — but the snapshot is taken
	// BEFORE the deferred Inc fires (deferred Inc happens after Encode
	// writes to w), so the scrape sees /metrics count = 0 (not yet
	// incremented for the current request). The current /metrics
	// request's Inc fires AFTER Encode returns.
}

// TestMetricsHTTP_NotConfiguredSurfaces503 verifies the /metrics
// endpoint returns 503 when the handler was constructed without
// WithMetrics. Default behavior — metrics opt-in via Option.
func TestMetricsHTTP_NotConfiguredSurfaces503(t *testing.T) {
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil) // no WithMetrics

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("status: got %d, want 503; body=%s", rr.Code, rr.Body.String())
	}
}

// TestMetricsHTTP_MethodNotAllowed verifies non-GET methods surface
// 405 + Allow header.
func TestMetricsHTTP_MethodNotAllowed(t *testing.T) {
	doAppend, _ := stubAppendFunc(nil)
	metrics := NewMetrics()
	h := MustNew(doAppend, nil, WithMetrics(metrics))

	req := httptest.NewRequest(http.MethodPost, "/metrics", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("status: got %d, want 405", rr.Code)
	}
	if allow := rr.Header().Get("Allow"); allow != http.MethodGet {
		t.Errorf("Allow header: got %q, want %q", allow, http.MethodGet)
	}
}

// TestMetricsHTTP_NilMetricsOptionPreservesDefault verifies the
// WithMetrics(nil) ergonomic — passing nil does NOT clobber the
// default (no metrics).
func TestMetricsHTTP_NilMetricsOptionPreservesDefault(t *testing.T) {
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithMetrics(nil))

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("status with WithMetrics(nil): got %d, want 503", rr.Code)
	}
}

// TestMetricsHTTP_NoIncWhenMetricsNotConfigured verifies that when
// WithMetrics is NOT supplied, ServeHTTP's deferred Inc is a no-op
// (no panic from nil-pointer-deref + per-request counter never
// increments). Structural witness via successful /healthz dispatch.
func TestMetricsHTTP_NoIncWhenMetricsNotConfigured(t *testing.T) {
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil) // no WithMetrics

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", rr.Code)
	}
}
