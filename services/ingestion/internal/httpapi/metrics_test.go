package httpapi

import (
	"bytes"
	"fmt"
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

// ----------------------------------------------------------------------
// §0200 histogram tests
// ----------------------------------------------------------------------

// TestMetricsHistogram_ObserveAggregatesPerPath verifies per-path
// histogram accumulation: observations land in the bucket whose
// upper bound is the smallest >= observation.
func TestMetricsHistogram_ObserveAggregatesPerPath(t *testing.T) {
	m := NewMetrics()
	// /healthz: three sub-ms observations → all in bucket le=1.
	m.Observe("/healthz", 0.3)
	m.Observe("/healthz", 0.5)
	m.Observe("/healthz", 0.7)
	// /slow: one 50ms + one 500ms observation.
	m.Observe("/slow", 50)
	m.Observe("/slow", 500)

	var buf bytes.Buffer
	if err := m.Encode(&buf); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	out := buf.String()
	// /healthz: 3 in le=1; also count=3, sum=1.5.
	for _, want := range []string{
		`ghosttrace_httpapi_request_duration_ms_bucket{path="/healthz",le="1"} 3`,
		`ghosttrace_httpapi_request_duration_ms_bucket{path="/healthz",le="+Inf"} 3`,
		`ghosttrace_httpapi_request_duration_ms_sum{path="/healthz"} 1.5`,
		`ghosttrace_httpapi_request_duration_ms_count{path="/healthz"} 3`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q\nfull output:\n%s", want, out)
		}
	}
	// /slow: 1 in le=50 (cumulative), 2 in le=500 (cumulative), count=2, sum=550.
	for _, want := range []string{
		`ghosttrace_httpapi_request_duration_ms_bucket{path="/slow",le="50"} 1`,
		`ghosttrace_httpapi_request_duration_ms_bucket{path="/slow",le="500"} 2`,
		`ghosttrace_httpapi_request_duration_ms_bucket{path="/slow",le="+Inf"} 2`,
		`ghosttrace_httpapi_request_duration_ms_sum{path="/slow"} 550`,
		`ghosttrace_httpapi_request_duration_ms_count{path="/slow"} 2`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q\nfull output:\n%s", want, out)
		}
	}
}

// TestMetricsHistogram_BucketsCumulativeAscending verifies the
// Prometheus convention: each bucket count is the count of
// observations <= that upper bound — cumulative ascending. The 10s
// bucket count is GE the 100ms bucket count, etc.
func TestMetricsHistogram_BucketsCumulativeAscending(t *testing.T) {
	m := NewMetrics()
	// Observations spanning multiple buckets.
	durations := []float64{0.5, 3, 7, 20, 40, 80, 200, 400, 800, 1500, 3000, 7500}
	for _, d := range durations {
		m.Observe("/varied", d)
	}
	var buf bytes.Buffer
	_ = m.Encode(&buf)
	lines := strings.Split(buf.String(), "\n")

	var bucketCounts []uint64
	for _, l := range lines {
		if !strings.HasPrefix(l, `ghosttrace_httpapi_request_duration_ms_bucket{path="/varied"`) {
			continue
		}
		// Parse trailing number.
		parts := strings.Fields(l)
		if len(parts) < 2 {
			continue
		}
		var n uint64
		if _, err := fmt.Sscanf(parts[len(parts)-1], "%d", &n); err != nil {
			continue
		}
		bucketCounts = append(bucketCounts, n)
	}
	if len(bucketCounts) < 2 {
		t.Fatalf("expected >=2 bucket lines for /varied, got %d:\n%s", len(bucketCounts), buf.String())
	}
	// Cumulative monotone-non-decreasing.
	for i := 1; i < len(bucketCounts); i++ {
		if bucketCounts[i] < bucketCounts[i-1] {
			t.Errorf("bucket count at index %d (%d) is less than previous (%d) — buckets must be cumulative ascending", i, bucketCounts[i], bucketCounts[i-1])
		}
	}
}

// TestMetricsHistogram_NegativeObservationClampsToZero verifies the
// defensive clamp: a negative duration is treated as 0 (lands in the
// le=1 bucket) without panicking or producing a negative sum.
func TestMetricsHistogram_NegativeObservationClampsToZero(t *testing.T) {
	m := NewMetrics()
	m.Observe("/negative", -5)
	var buf bytes.Buffer
	_ = m.Encode(&buf)
	out := buf.String()
	if !strings.Contains(out, `ghosttrace_httpapi_request_duration_ms_sum{path="/negative"} 0`) {
		t.Errorf("negative observation should clamp sum to 0:\n%s", out)
	}
	if !strings.Contains(out, `ghosttrace_httpapi_request_duration_ms_bucket{path="/negative",le="1"} 1`) {
		t.Errorf("clamped-to-0 observation should land in le=1 bucket:\n%s", out)
	}
}

// TestMetricsHistogram_ConcurrentObserveIsSafe verifies Observe is
// concurrent-safe (mirrors the §0199 concurrent-Inc-stress test).
func TestMetricsHistogram_ConcurrentObserveIsSafe(t *testing.T) {
	m := NewMetrics()
	const goroutines = 10
	const perGoroutine = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < perGoroutine; j++ {
				m.Observe("/concurrent-hist", float64(j%50))
			}
		}()
	}
	wg.Wait()
	var buf bytes.Buffer
	_ = m.Encode(&buf)
	want := `ghosttrace_httpapi_request_duration_ms_count{path="/concurrent-hist"} 1000`
	if !strings.Contains(buf.String(), want) {
		t.Errorf("expected histogram count = 1000 after concurrent Observe:\n%s", buf.String())
	}
}

// TestMetricsHistogram_EncodeEmptyHistogramRegistry verifies an empty
// registry emits the histogram HELP + TYPE preamble but no per-path
// sample lines.
func TestMetricsHistogram_EncodeEmptyHistogramRegistry(t *testing.T) {
	m := NewMetrics()
	var buf bytes.Buffer
	_ = m.Encode(&buf)
	out := buf.String()
	if !strings.Contains(out, "# HELP ghosttrace_httpapi_request_duration_ms") {
		t.Errorf("missing histogram HELP line:\n%s", out)
	}
	if !strings.Contains(out, "# TYPE ghosttrace_httpapi_request_duration_ms histogram") {
		t.Errorf("missing histogram TYPE line:\n%s", out)
	}
	if strings.Contains(out, "ghosttrace_httpapi_request_duration_ms_bucket{") {
		t.Errorf("empty registry emitted a histogram bucket sample:\n%s", out)
	}
}

// TestMetricsHTTP_AuthExempt verifies /metrics is reachable WITHOUT
// auth even when single-token auth is configured (per §0201 production
// wiring + Prometheus convention — operator-side gating is via network
// policy on the scrape network, not per-request bearer-token auth).
func TestMetricsHTTP_AuthExempt(t *testing.T) {
	doAppend, _ := stubAppendFunc(nil)
	metrics := NewMetrics()
	h := MustNew(doAppend, nil, WithMetrics(metrics), WithAuthToken("secret"))

	// No Authorization header — would 401 on /v1/events; must succeed on /metrics.
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code == http.StatusUnauthorized {
		t.Errorf("/metrics returned 401 with auth configured; should be auth-exempt (T0 like /healthz)")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200 (auth-exempt success path); body=%s", rr.Code, rr.Body.String())
	}
}

// ----------------------------------------------------------------------
// §0203 per-instance bucket-override tests
// ----------------------------------------------------------------------

// TestMetrics_WithHistogramBucketsOverride verifies that
// WithHistogramBuckets supplies the per-instance bucket boundaries +
// the encoded output reflects the operator-supplied boundaries (not
// the §0200 default).
func TestMetrics_WithHistogramBucketsOverride(t *testing.T) {
	custom := []float64{2, 8, 32}
	m := NewMetrics(WithHistogramBuckets(custom))
	m.Observe("/custom", 1)  // → le=2
	m.Observe("/custom", 5)  // → le=8 (cumulative includes le=2)
	m.Observe("/custom", 20) // → le=32 (cumulative includes le=8)
	m.Observe("/custom", 50) // → +Inf only

	var buf bytes.Buffer
	if err := m.Encode(&buf); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		`ghosttrace_httpapi_request_duration_ms_bucket{path="/custom",le="2"} 1`,
		`ghosttrace_httpapi_request_duration_ms_bucket{path="/custom",le="8"} 2`,
		`ghosttrace_httpapi_request_duration_ms_bucket{path="/custom",le="32"} 3`,
		`ghosttrace_httpapi_request_duration_ms_bucket{path="/custom",le="+Inf"} 4`,
		`ghosttrace_httpapi_request_duration_ms_count{path="/custom"} 4`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q\nfull output:\n%s", want, out)
		}
	}
	// Default §0200 buckets must NOT appear in the output (operator
	// override fully replaces the default; not a merge).
	for _, notWant := range []string{
		`le="1"`, `le="5"`, `le="10"`, `le="25"`, `le="50"`,
	} {
		if strings.Contains(out, notWant) {
			t.Errorf("default §0200 bucket %q leaked into output despite override:\n%s", notWant, out)
		}
	}
}

// TestMetrics_WithHistogramBucketsEmptyPreservesDefault verifies the
// nil/empty-buckets ergonomic — passing nil or [] does NOT override;
// the §0200 default applies. Matches the WithLogger(nil) /
// WithMetrics(nil) nil-preserving precedent.
func TestMetrics_WithHistogramBucketsEmptyPreservesDefault(t *testing.T) {
	cases := [][]float64{nil, {}}
	for i, c := range cases {
		m := NewMetrics(WithHistogramBuckets(c))
		m.Observe("/preserve", 0.5)
		var buf bytes.Buffer
		_ = m.Encode(&buf)
		out := buf.String()
		// Must contain the §0200 default first bucket (le=1).
		if !strings.Contains(out, `ghosttrace_httpapi_request_duration_ms_bucket{path="/preserve",le="1"} 1`) {
			t.Errorf("case %d (nil/empty): default §0200 buckets missing:\n%s", i, out)
		}
	}
}

// TestMetrics_WithHistogramBucketsDefensiveCopy verifies that mutating
// the operator-supplied slice AFTER NewMetrics does NOT affect the
// registry's bucket boundaries. Defensive-copy contract.
func TestMetrics_WithHistogramBucketsDefensiveCopy(t *testing.T) {
	custom := []float64{1, 10, 100}
	m := NewMetrics(WithHistogramBuckets(custom))
	// Mutate the caller's slice.
	custom[0] = 999
	m.Observe("/defensive", 0.5)
	var buf bytes.Buffer
	_ = m.Encode(&buf)
	out := buf.String()
	if !strings.Contains(out, `ghosttrace_httpapi_request_duration_ms_bucket{path="/defensive",le="1"} 1`) {
		t.Errorf("post-NewMetrics caller mutation leaked into registry buckets; defensive-copy contract broken:\n%s", out)
	}
}

// TestMetrics_DefaultBucketsWhenNoOption verifies the no-option path:
// NewMetrics() with zero MetricsOption args produces the §0200 default
// buckets.
func TestMetrics_DefaultBucketsWhenNoOption(t *testing.T) {
	m := NewMetrics()
	m.Observe("/default", 0.5)
	var buf bytes.Buffer
	_ = m.Encode(&buf)
	out := buf.String()
	if !strings.Contains(out, `ghosttrace_httpapi_request_duration_ms_bucket{path="/default",le="1"} 1`) {
		t.Errorf("default-buckets path broken (no MetricsOption supplied):\n%s", out)
	}
}

// TestMetricsHTTP_HistogramObservedOnServeHTTP exercises end-to-end:
// requests against the /healthz endpoint populate the histogram
// observable via /metrics scrape.
func TestMetricsHTTP_HistogramObservedOnServeHTTP(t *testing.T) {
	doAppend, _ := stubAppendFunc(nil)
	metrics := NewMetrics()
	h := MustNew(doAppend, nil, WithMetrics(metrics))

	// Three /healthz requests should yield count=3 + le=+Inf bucket=3.
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	body := rr.Body.String()
	if !strings.Contains(body, `ghosttrace_httpapi_request_duration_ms_count{path="/healthz"} 3`) {
		t.Errorf("expected histogram count=3 for /healthz:\n%s", body)
	}
	if !strings.Contains(body, `ghosttrace_httpapi_request_duration_ms_bucket{path="/healthz",le="+Inf"} 3`) {
		t.Errorf("expected +Inf bucket=3 for /healthz:\n%s", body)
	}
}
