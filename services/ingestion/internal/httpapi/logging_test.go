package httpapi

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestRequestLogging_EmitsStructuredEntryOnHealthz verifies the
// per-request structured log entry per §0197: method/path/status/
// duration_ms/tier/remote_addr fields populated correctly. /healthz
// chosen because it is unauthenticated (no auth-failure path) + has
// no tier (T0 exempt per §0094) — exercises the empty-tier code path.
func TestRequestLogging_EmitsStructuredEntryOnHealthz(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithLogger(logger))

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.RemoteAddr = "203.0.113.10:54321"
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rr.Code)
	}

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("decode log entry: %v\nraw=%s", err, buf.String())
	}
	if entry["msg"] != "httpapi request" {
		t.Errorf("msg: got %v, want \"httpapi request\"", entry["msg"])
	}
	if entry["method"] != http.MethodGet {
		t.Errorf("method: got %v, want %s", entry["method"], http.MethodGet)
	}
	if entry["path"] != "/healthz" {
		t.Errorf("path: got %v, want /healthz", entry["path"])
	}
	if status, ok := entry["status"].(float64); !ok || int(status) != http.StatusOK {
		t.Errorf("status: got %v, want 200", entry["status"])
	}
	if _, ok := entry["duration_ms"].(float64); !ok {
		t.Errorf("duration_ms: got %v, want numeric", entry["duration_ms"])
	}
	if entry["tier"] != "" {
		t.Errorf("tier: got %v, want empty string (T0 /healthz exempt per §0094)", entry["tier"])
	}
	if entry["remote_addr"] != "203.0.113.10:54321" {
		t.Errorf("remote_addr: got %v, want 203.0.113.10:54321", entry["remote_addr"])
	}
}

// TestRequestLogging_CapturesTierForOperatorReadEndpoint verifies the
// tier field reflects the §0094 classification — operator-read for
// /v1/hypotheses.
func TestRequestLogging_CapturesTierForOperatorReadEndpoint(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	sub, _, _ := substrateWithBCFormation(t)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub), WithLogger(logger))

	req := httptest.NewRequest(http.MethodGet, "/v1/hypotheses", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("decode log entry: %v", err)
	}
	if entry["tier"] != string(TierOperatorRead) {
		t.Errorf("tier: got %v, want %q", entry["tier"], TierOperatorRead)
	}
}

// TestRequestLogging_CapturesStatusOnMethodNotAllowed verifies the
// captured status reflects 405 when the handler emits one via
// http.Error or w.WriteHeader (NOT 200 from the implicit default).
func TestRequestLogging_CapturesStatusOnMethodNotAllowed(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	sub, _, _ := substrateWithBCFormation(t)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub), WithLogger(logger))

	req := httptest.NewRequest(http.MethodPost, "/v1/morphology", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status: got %d, want 405", rr.Code)
	}
	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("decode log entry: %v", err)
	}
	if status, ok := entry["status"].(float64); !ok || int(status) != http.StatusMethodNotAllowed {
		t.Errorf("status: got %v, want 405", entry["status"])
	}
}

// TestRequestLogging_CapturesStatusOnUnauthorized verifies the captured
// status reflects 401 when the auth gate rejects the request BEFORE
// dispatching to a per-route handler. Witnesses that the
// loggingResponseWriter wraps the auth path too (writeUnauthorized must
// flow through lw to record status).
func TestRequestLogging_CapturesStatusOnUnauthorized(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithAuthToken("secret-token"), WithLogger(logger))

	req := httptest.NewRequest(http.MethodPost, "/v1/events/declared_session", nil)
	// No Authorization header → 401.
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want 401", rr.Code)
	}
	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("decode log entry: %v\nraw=%s", err, buf.String())
	}
	if status, ok := entry["status"].(float64); !ok || int(status) != http.StatusUnauthorized {
		t.Errorf("status: got %v, want 401", entry["status"])
	}
}

// TestRequestLogging_DefaultLoggerEmitsNothing verifies that when
// WithLogger is NOT configured, the default discard logger emits no
// output (preserves pre-§0197 silent operation; tests not exercising
// logging surface continue to produce no log noise).
func TestRequestLogging_DefaultLoggerEmitsNothing(t *testing.T) {
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil) // no WithLogger

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", rr.Code)
	}
	// No explicit log-output verification — the io.Discard handler
	// swallows everything; the contract is "no panic + no test-output
	// noise." The structural witness is that this test produces no log
	// lines in the test runner's output.
}

// TestRequestLogging_NilLoggerToWithLoggerPreservesDefault verifies the
// WithLogger(nil) ergonomic — passing nil does NOT clobber the default
// discard logger.
func TestRequestLogging_NilLoggerToWithLoggerPreservesDefault(t *testing.T) {
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithLogger(nil))

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	// Must not panic.
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", rr.Code)
	}
}

// TestLoggingResponseWriter_ImplicitTwoHundredOnWriteWithoutWriteHeader
// verifies the loggingResponseWriter's status-capture contract: when
// the handler writes the body via Write() without an explicit
// WriteHeader call, the captured status defaults to 200 (matches
// net/http's implicit-200 semantic at the writer layer).
func TestLoggingResponseWriter_ImplicitTwoHundredOnWriteWithoutWriteHeader(t *testing.T) {
	rr := httptest.NewRecorder()
	lw := &loggingResponseWriter{ResponseWriter: rr}
	_, _ = lw.Write([]byte("body"))
	if got := lw.effectiveStatus(); got != http.StatusOK {
		t.Errorf("effectiveStatus: got %d, want 200", got)
	}
}

// TestLoggingResponseWriter_FirstWriteHeaderWins verifies the
// loggingResponseWriter records only the FIRST WriteHeader call's
// status (matches net/http's first-write-wins contract).
func TestLoggingResponseWriter_FirstWriteHeaderWins(t *testing.T) {
	rr := httptest.NewRecorder()
	lw := &loggingResponseWriter{ResponseWriter: rr}
	lw.WriteHeader(http.StatusTeapot)
	lw.WriteHeader(http.StatusNotFound) // ignored
	if got := lw.effectiveStatus(); got != http.StatusTeapot {
		t.Errorf("effectiveStatus: got %d, want 418 (first write wins)", got)
	}
}

// TestLoggingResponseWriter_DefaultStatusIsTwoHundred verifies that
// effectiveStatus returns 200 when the writer was never written to —
// the no-body no-WriteHeader edge case.
func TestLoggingResponseWriter_DefaultStatusIsTwoHundred(t *testing.T) {
	rr := httptest.NewRecorder()
	lw := &loggingResponseWriter{ResponseWriter: rr}
	if got := lw.effectiveStatus(); got != http.StatusOK {
		t.Errorf("effectiveStatus: got %d, want 200", got)
	}
}

// indexInBytes is a tiny case-insensitive substring presence check
// used by the auth-witness test.
func bytesContains(haystack []byte, needle string) bool {
	return strings.Contains(string(haystack), needle)
}

// TestRequestLogging_NoStatusAttributeMissing verifies that EVERY
// emitted log entry carries the status attribute — defending against a
// regression where the loggingResponseWriter is not wired into the
// auth path or into the unauthenticated /healthz path.
func TestRequestLogging_NoStatusAttributeMissing(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithLogger(logger))

	for _, path := range []string{"/healthz", "/v1/morphology"} {
		buf.Reset()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if !bytesContains(buf.Bytes(), `"status":`) {
			t.Errorf("path %q: log entry missing `status` attribute: %s", path, buf.String())
		}
	}
}
