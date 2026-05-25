package httpapi

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestRequestID_EchoedInResponseHeader verifies the X-Request-Id
// response header echoes the request-side header verbatim when
// present (operator-side upstream-edge correlation per §0198 wire
// contract).
func TestRequestID_EchoedInResponseHeader(t *testing.T) {
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil)

	const upstreamID = "operator-supplied-correlation-id-12345"
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set(requestIDHeader, upstreamID)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if got := rr.Header().Get(requestIDHeader); got != upstreamID {
		t.Errorf("X-Request-Id echo: got %q, want %q", got, upstreamID)
	}
}

// TestRequestID_GeneratedWhenAbsent verifies a fresh request_id is
// generated when no X-Request-Id header is supplied + emitted in the
// response header. The generated id matches the
// 32-hex-char format from generateRequestID.
func TestRequestID_GeneratedWhenAbsent(t *testing.T) {
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	got := rr.Header().Get(requestIDHeader)
	if got == "" {
		t.Fatal("X-Request-Id: response header is empty (expected generated id)")
	}
	if len(got) != 32 {
		t.Errorf("X-Request-Id length: got %d, want 32 (16 random bytes hex-encoded)", len(got))
	}
	if _, err := hex.DecodeString(got); err != nil {
		t.Errorf("X-Request-Id not hex: %v (got %q)", err, got)
	}
}

// TestRequestID_PropagatedToStructuredEntry verifies the request_id
// field is added to the §0197 structured entry — cross-service
// correlation contract per §0198.
func TestRequestID_PropagatedToStructuredEntry(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithLogger(logger))

	const upstreamID = "test-correlation-id-67890"
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set(requestIDHeader, upstreamID)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("decode entry: %v", err)
	}
	if entry["request_id"] != upstreamID {
		t.Errorf("request_id in structured entry: got %v, want %q", entry["request_id"], upstreamID)
	}
}

// TestRequestID_GeneratedIDsAreUniqueAcrossRequests verifies fresh-
// generated IDs differ across consecutive requests — collision-
// resistance for correlation across high-traffic windows.
func TestRequestID_GeneratedIDsAreUniqueAcrossRequests(t *testing.T) {
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil)

	seen := make(map[string]struct{})
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		id := rr.Header().Get(requestIDHeader)
		if _, dup := seen[id]; dup {
			t.Errorf("request_id %q repeated within 100-request window (collision rate too high for correlation)", id)
		}
		seen[id] = struct{}{}
	}
}

// TestRequestID_EchoedOnAuthFailure verifies the request_id header is
// set on the response BEFORE the auth gate rejects (so failed-auth
// entries can still be correlated upstream).
func TestRequestID_EchoedOnAuthFailure(t *testing.T) {
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithAuthToken("secret"))

	const upstreamID = "auth-failure-correlation-id"
	req := httptest.NewRequest(http.MethodPost, "/v1/events/declared_session", nil)
	req.Header.Set(requestIDHeader, upstreamID)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status: got %d, want 401", rr.Code)
	}
	if got := rr.Header().Get(requestIDHeader); got != upstreamID {
		t.Errorf("X-Request-Id on 401: got %q, want %q (correlation must survive auth failure)", got, upstreamID)
	}
}

// TestGenerateRequestID_FormatContract verifies the standalone
// generator's format contract: 32 hex chars (16 random bytes).
func TestGenerateRequestID_FormatContract(t *testing.T) {
	id := generateRequestID()
	if len(id) != 32 {
		t.Errorf("len: got %d, want 32", len(id))
	}
	if _, err := hex.DecodeString(id); err != nil {
		t.Errorf("not hex: %v (got %q)", err, id)
	}
}

// TestResolveRequestID_PrefersHeader verifies resolveRequestID returns
// the request header value verbatim when non-empty (no length / format
// validation — accepts any non-empty operator-supplied value).
func TestResolveRequestID_PrefersHeader(t *testing.T) {
	cases := []struct {
		name     string
		incoming string
	}{
		{"uuid_format", "550e8400-e29b-41d4-a716-446655440000"},
		{"short_id", "abc"},
		{"long_id", "a-very-long-id-with-extra-segments-for-format-tolerance-witness"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
			req.Header.Set(requestIDHeader, c.incoming)
			if got := resolveRequestID(req); got != c.incoming {
				t.Errorf("resolveRequestID: got %q, want %q", got, c.incoming)
			}
		})
	}
}

// TestResolveRequestID_GeneratesWhenHeaderEmpty verifies an empty
// header value (header absent OR header set to empty string) triggers
// generation.
func TestResolveRequestID_GeneratesWhenHeaderEmpty(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	// No X-Request-Id set.
	got := resolveRequestID(req)
	if got == "" {
		t.Fatal("resolveRequestID: got empty, want generated")
	}
	if len(got) != 32 {
		t.Errorf("len: got %d, want 32", len(got))
	}
}
