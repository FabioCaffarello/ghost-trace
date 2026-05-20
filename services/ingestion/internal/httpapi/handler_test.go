package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"google.golang.org/protobuf/proto"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// recordingFatalReporter records every ReportFatal call for verification.
type recordingFatalReporter struct {
	mu     sync.Mutex
	errors []error
}

func (r *recordingFatalReporter) ReportFatal(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.errors = append(r.errors, err)
}

func (r *recordingFatalReporter) Calls() []error {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]error, len(r.errors))
	copy(out, r.errors)
	return out
}

// stubAppendFunc returns an AppendFunc whose ith call returns
// outcomes[i] (or success if i ≥ len(outcomes) or outcomes[i] is nil).
func stubAppendFunc(outcomes []error) (AppendFunc, *int) {
	var calls int
	fn := func(ctx context.Context, msg proto.Message, eventTime int64, env ingest.Envelope) (ingest.AppendReport, error) {
		idx := calls
		calls++
		if idx < len(outcomes) && outcomes[idx] != nil {
			return ingest.AppendReport{}, outcomes[idx]
		}
		return ingest.AppendReport{
			EventHashHex:          "0000000000000000000000000000000000000000000000000000000000000001",
			IngestionEventHashHex: "1111111111111111111111111111111111111111111111111111111111111111",
			PayloadBytes:          8,
		}, nil
	}
	return fn, &calls
}

func encodePayload(t *testing.T, msg proto.Message) []byte {
	t.Helper()
	b, err := proto.Marshal(msg)
	if err != nil {
		t.Fatalf("proto.Marshal: %v", err)
	}
	return b
}

func newTestMsg() *eventsv1.DeclaredSession {
	return &eventsv1.DeclaredSession{
		DeclaredAt:        1716120000000000000,
		ActorRef:          "actor-http-test",
		SessionDescriptor: []byte("session-bytes"),
	}
}

func TestHealthz(t *testing.T) {
	doAppend, _ := stubAppendFunc(nil)
	h := New(doAppend, nil)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `"status":"ok"`) {
		t.Errorf("body missing status:ok marker: %q", body)
	}
}

func TestHealthzRejectsNonGet(t *testing.T) {
	doAppend, _ := stubAppendFunc(nil)
	h := New(doAppend, nil)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/healthz", nil)
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusMethodNotAllowed; got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
	if got := rr.Header().Get("Allow"); got != http.MethodGet {
		t.Errorf("Allow header: got %q, want %q", got, http.MethodGet)
	}
}

func TestUnknownPath(t *testing.T) {
	doAppend, _ := stubAppendFunc(nil)
	h := New(doAppend, nil)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusNotFound; got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
}

func TestPostEventsHappyPath(t *testing.T) {
	doAppend, callCount := stubAppendFunc(nil)
	fatal := &recordingFatalReporter{}
	h := New(doAppend, fatal)

	payload := encodePayload(t, newTestMsg())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/events/declared-session", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/x-protobuf")

	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Errorf("status: got %d, want %d (body: %s)", got, want, rr.Body.String())
	}
	if *callCount != 1 {
		t.Errorf("Append calls: got %d, want 1", *callCount)
	}
	if got := len(fatal.Calls()); got != 0 {
		t.Errorf("FatalReporter calls: got %d, want 0", got)
	}

	var conf confirmation
	if err := json.Unmarshal(rr.Body.Bytes(), &conf); err != nil {
		t.Fatalf("body not parseable as confirmation: %v — %q", err, rr.Body.String())
	}
	if conf.EventHash == "" {
		t.Errorf("confirmation missing event_hash")
	}
	if conf.PayloadBytes <= 0 {
		t.Errorf("confirmation payload_bytes: got %d, want >0", conf.PayloadBytes)
	}
}

func TestPostEventsRejectsWrongMethod(t *testing.T) {
	doAppend, _ := stubAppendFunc(nil)
	h := New(doAppend, nil)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/events/declared-session", nil)
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusMethodNotAllowed; got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
}

func TestPostEventsRejectsWrongContentType(t *testing.T) {
	doAppend, _ := stubAppendFunc(nil)
	h := New(doAppend, nil)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/events/declared-session", strings.NewReader("plain text"))
	req.Header.Set("Content-Type", "text/plain")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusUnsupportedMediaType; got != want {
		t.Errorf("status: got %d, want %d (body: %s)", got, want, rr.Body.String())
	}
}

func TestPostEventsRejectsEmptyBody(t *testing.T) {
	doAppend, callCount := stubAppendFunc(nil)
	h := New(doAppend, nil)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/events/declared-session", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusBadRequest; got != want {
		t.Errorf("status: got %d, want %d (body: %s)", got, want, rr.Body.String())
	}
	if *callCount != 0 {
		t.Errorf("Append should not have been called; got %d calls", *callCount)
	}
}

func TestPostEventsRejectsInvalidProtobuf(t *testing.T) {
	doAppend, callCount := stubAppendFunc(nil)
	h := New(doAppend, nil)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/events/declared-session", strings.NewReader("not-a-protobuf-message"))
	req.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusBadRequest; got != want {
		t.Errorf("status: got %d, want %d (body: %s)", got, want, rr.Body.String())
	}
	if *callCount != 0 {
		t.Errorf("Append should not have been called; got %d calls", *callCount)
	}
	var errBody ingestError
	if err := json.Unmarshal(rr.Body.Bytes(), &errBody); err != nil {
		t.Errorf("body not parseable as ingestError: %v", err)
	}
}

func TestPostEventsUnrecoverableTriggersFatal(t *testing.T) {
	doAppend, _ := stubAppendFunc([]error{
		fmt.Errorf("at /blobs/de/ad: %w", substrate.ErrBlobCollision),
	})
	fatal := &recordingFatalReporter{}
	h := New(doAppend, fatal)

	payload := encodePayload(t, newTestMsg())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/events/declared-session", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusInternalServerError; got != want {
		t.Errorf("status: got %d, want %d (body: %s)", got, want, rr.Body.String())
	}

	calls := fatal.Calls()
	if len(calls) != 1 {
		t.Fatalf("FatalReporter calls: got %d, want 1", len(calls))
	}
	if !errors.Is(calls[0], substrate.ErrBlobCollision) {
		t.Fatalf("FatalReporter did not receive ErrBlobCollision-derived error: %v", calls[0])
	}

	var errBody ingestError
	if err := json.Unmarshal(rr.Body.Bytes(), &errBody); err != nil {
		t.Errorf("body not parseable as ingestError: %v", err)
	}
	if !strings.Contains(errBody.Error, "unrecoverable") {
		t.Errorf("error body missing 'unrecoverable' marker: %q", errBody.Error)
	}
}

func TestPostEventsRecoverableErrorReturns400(t *testing.T) {
	// A non-unrecoverable error from Append (e.g. simulated transient
	// SQLite failure) returns 400 without triggering fatal.
	doAppend, _ := stubAppendFunc([]error{
		errors.New("simulated transient: database is locked"),
	})
	fatal := &recordingFatalReporter{}
	h := New(doAppend, fatal)

	payload := encodePayload(t, newTestMsg())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/events/declared-session", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusBadRequest; got != want {
		t.Errorf("status: got %d, want %d (body: %s)", got, want, rr.Body.String())
	}
	if got := len(fatal.Calls()); got != 0 {
		t.Errorf("FatalReporter calls: got %d, want 0 (recoverable should not signal fatal)", got)
	}
}

func TestPostEventsHashMismatchAlsoUnrecoverable(t *testing.T) {
	// Symmetry with the ErrBlobCollision test.
	doAppend, _ := stubAppendFunc([]error{
		fmt.Errorf("substrate.ReadBlob at /blobs/be/ef: %w", substrate.ErrHashMismatch),
	})
	fatal := &recordingFatalReporter{}
	h := New(doAppend, fatal)

	payload := encodePayload(t, newTestMsg())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/events/declared-session", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusInternalServerError; got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
	calls := fatal.Calls()
	if len(calls) != 1 {
		t.Fatalf("FatalReporter calls: got %d, want 1", len(calls))
	}
	if !errors.Is(calls[0], substrate.ErrHashMismatch) {
		t.Fatalf("FatalReporter did not receive ErrHashMismatch-derived error: %v", calls[0])
	}
}

func TestRequestBodyLimitEnforced(t *testing.T) {
	doAppend, _ := stubAppendFunc(nil)
	h := New(doAppend, nil)
	h.requestBodyLimit = 64 // tighten for this test

	// Body larger than the limit triggers a MaxBytesReader error,
	// which the handler surfaces as 400.
	body := make([]byte, 256)
	for i := range body {
		body[i] = 'A'
	}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/events/declared-session", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusBadRequest; got != want {
		// MaxBytesReader may produce a 413 directly via http.MaxBytesReader's
		// response-writer side-effect in newer Go versions. Accept either.
		if got != http.StatusRequestEntityTooLarge {
			t.Errorf("status: got %d, want 400 or 413 (body: %s)", got, rr.Body.String())
		}
	}
}

func TestPostEventsNoAuthRequiredByDefault(t *testing.T) {
	// Sanity: backward-compat check. Without WithAuthToken, requests
	// without an Authorization header succeed.
	doAppend, _ := stubAppendFunc(nil)
	h := New(doAppend, nil)

	payload := encodePayload(t, newTestMsg())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/events/declared-session", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Errorf("status: got %d, want %d (no-auth path; body: %s)", got, want, rr.Body.String())
	}
}

func TestPostEventsRequiresAuthWhenConfigured(t *testing.T) {
	doAppend, callCount := stubAppendFunc(nil)
	h := New(doAppend, nil, WithAuthToken("secret-token"))

	payload := encodePayload(t, newTestMsg())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/events/declared-session", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/x-protobuf")
	// No Authorization header.
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusUnauthorized; got != want {
		t.Errorf("status: got %d, want %d (auth required, no header; body: %s)", got, want, rr.Body.String())
	}
	if *callCount != 0 {
		t.Errorf("Append should not have been called; got %d calls", *callCount)
	}
	if got := rr.Header().Get("WWW-Authenticate"); !strings.HasPrefix(got, "Bearer realm=") {
		t.Errorf("WWW-Authenticate header missing or wrong shape: %q", got)
	}
}

func TestPostEventsAuthWrongToken(t *testing.T) {
	doAppend, callCount := stubAppendFunc(nil)
	h := New(doAppend, nil, WithAuthToken("secret-token"))

	payload := encodePayload(t, newTestMsg())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/events/declared-session", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Authorization", "Bearer wrong-token-same-length")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusUnauthorized; got != want {
		t.Errorf("status: got %d, want %d (wrong token)", got, want)
	}
	if *callCount != 0 {
		t.Errorf("Append should not have been called on wrong-token; got %d calls", *callCount)
	}
}

func TestPostEventsAuthWrongTokenDifferentLength(t *testing.T) {
	doAppend, _ := stubAppendFunc(nil)
	h := New(doAppend, nil, WithAuthToken("secret-token"))

	payload := encodePayload(t, newTestMsg())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/events/declared-session", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Authorization", "Bearer short")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusUnauthorized; got != want {
		t.Errorf("status: got %d, want %d (wrong-length token)", got, want)
	}
}

func TestPostEventsAuthCorrectToken(t *testing.T) {
	doAppend, callCount := stubAppendFunc(nil)
	h := New(doAppend, nil, WithAuthToken("secret-token"))

	payload := encodePayload(t, newTestMsg())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/events/declared-session", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Authorization", "Bearer secret-token")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Errorf("status: got %d, want %d (correct token; body: %s)", got, want, rr.Body.String())
	}
	if *callCount != 1 {
		t.Errorf("Append calls: got %d, want 1 (correct token should reach Append)", *callCount)
	}
}

func TestPostEventsAuthBadHeaderFormat(t *testing.T) {
	// Authorization header present but not "Bearer <token>" — e.g.,
	// "Basic ..." or raw token without scheme.
	doAppend, _ := stubAppendFunc(nil)
	h := New(doAppend, nil, WithAuthToken("secret-token"))

	cases := []struct {
		name   string
		header string
	}{
		{"basic-scheme", "Basic dXNlcjpwYXNz"},
		{"raw-token-no-scheme", "secret-token"},
		{"bearer-lowercase", "bearer secret-token"},
		{"empty", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			payload := encodePayload(t, newTestMsg())
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/v1/events/declared-session", bytes.NewReader(payload))
			req.Header.Set("Content-Type", "application/x-protobuf")
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			h.ServeHTTP(rr, req)

			if got, want := rr.Code, http.StatusUnauthorized; got != want {
				t.Errorf("status: got %d, want %d (header: %q)", got, want, tc.header)
			}
		})
	}
}

func TestHealthzExemptFromAuth(t *testing.T) {
	// /healthz must NOT require auth even when a token is configured,
	// so that orchestrators (Kubernetes, etc.) can liveness-probe
	// without credentials.
	doAppend, _ := stubAppendFunc(nil)
	h := New(doAppend, nil, WithAuthToken("secret-token"))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	// No Authorization header.
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Errorf("status: got %d, want %d (healthz must be auth-exempt; body: %s)", got, want, rr.Body.String())
	}
}

func TestUnknownPathReturns401WhenAuthConfigured(t *testing.T) {
	// Unauthenticated probes for unknown paths return 401 (not 404)
	// when auth is configured, so the path structure is not leaked
	// to unauthenticated clients.
	doAppend, _ := stubAppendFunc(nil)
	h := New(doAppend, nil, WithAuthToken("secret-token"))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/some-random-path", nil)
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusUnauthorized; got != want {
		t.Errorf("status: got %d, want %d (unknown path under auth; body: %s)", got, want, rr.Body.String())
	}
}

func TestEnvelopeForRequestPlainHTTP(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/events/declared-session", nil)
	env := envelopeForRequest(req)
	if env.Channel != "http" {
		t.Errorf("Channel: got %q, want %q", env.Channel, "http")
	}
	if env.ClientCommonName != "" || env.ClientCertSHA256 != "" || len(env.ClientSubjectAltNames) != 0 {
		t.Errorf("client identity fields should be empty for plain HTTP: %+v", env)
	}
	if env.ReceivedAt == 0 {
		t.Errorf("ReceivedAt should be set")
	}
}

// TestPostEventsSuccessIncludesIngestionEventHash verifies the success
// response carries both hashes (primary + paired enrichment) per §0038.
func TestPostEventsSuccessIncludesIngestionEventHash(t *testing.T) {
	doAppend, _ := stubAppendFunc(nil)
	h := New(doAppend, nil)

	payload := encodePayload(t, newTestMsg())
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/events/declared-session", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200 (body: %s)", rr.Code, rr.Body.String())
	}
	var conf confirmation
	if err := json.Unmarshal(rr.Body.Bytes(), &conf); err != nil {
		t.Fatalf("body not parseable as confirmation: %v", err)
	}
	if conf.EventHash == "" {
		t.Error("confirmation missing event_hash")
	}
	if conf.IngestionEventHash == "" {
		t.Error("confirmation missing ingestion_event_hash (provenance pair not surfaced)")
	}
	if conf.EventHash == conf.IngestionEventHash {
		t.Errorf("event_hash and ingestion_event_hash should differ: %s", conf.EventHash)
	}
}

// TestPostNetworkEventHappyPath proves the dispatch routes the
// second Cat I message type (NetworkEvent) end-to-end through the same
// pipeline as DeclaredSession, with the type-specific event-time
// accessor reading from the right field.
func TestPostNetworkEventHappyPath(t *testing.T) {
	var capturedType string
	var capturedEventTime int64
	doAppend := func(ctx context.Context, msg proto.Message, eventTime int64, env ingest.Envelope) (ingest.AppendReport, error) {
		capturedType = string(msg.ProtoReflect().Descriptor().FullName())
		capturedEventTime = eventTime
		return ingest.AppendReport{
			EventHashHex:          "00000000000000000000000000000000000000000000000000000000000000aa",
			IngestionEventHashHex: "00000000000000000000000000000000000000000000000000000000000000bb",
			PayloadBytes:          16,
		}, nil
	}
	h := New(doAppend, nil)

	netEvt := &eventsv1.NetworkEvent{
		ObservedAt:      1716120000000000777,
		ActorRef:        "actor-network-test",
		EndpointRef:     "10.0.0.42:443",
		EventDescriptor: []byte("flow-record"),
	}
	payload, err := proto.Marshal(netEvt)
	if err != nil {
		t.Fatalf("proto.Marshal: %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/events/network-event", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("status: got %d, want %d (body: %s)", got, want, rr.Body.String())
	}
	if capturedType != "ghosttrace.events.v1.NetworkEvent" {
		t.Errorf("dispatched type: got %q, want ghosttrace.events.v1.NetworkEvent", capturedType)
	}
	if capturedEventTime != netEvt.ObservedAt {
		t.Errorf("event time: got %d, want %d (NetworkEvent.observed_at)", capturedEventTime, netEvt.ObservedAt)
	}
}

// TestPostEventsUnknownTypeReturns404 proves the dispatch rejects
// unregistered types with a structured 404 that enumerates the known
// types — the operator-facing migration hint.
func TestPostEventsUnknownTypeReturns404(t *testing.T) {
	doAppend, callCount := stubAppendFunc(nil)
	h := New(doAppend, nil)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/events/fingerprint-snapshot", bytes.NewReader([]byte("anything")))
	req.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusNotFound; got != want {
		t.Fatalf("status: got %d, want %d (body: %s)", got, want, rr.Body.String())
	}
	if *callCount != 0 {
		t.Errorf("Append should not have been called; got %d calls", *callCount)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "fingerprint-snapshot") {
		t.Errorf("error body should echo the unknown type: %q", body)
	}
	if !strings.Contains(body, "declared-session") || !strings.Contains(body, "network-event") {
		t.Errorf("error body should enumerate known types: %q", body)
	}
}

// TestPostEventsUntypedPathReturns404WithMigrationHint proves the bare
// /v1/events path is rejected with a structured error that points
// producers at the new typed path layout.
func TestPostEventsUntypedPathReturns404WithMigrationHint(t *testing.T) {
	doAppend, _ := stubAppendFunc(nil)
	h := New(doAppend, nil)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/events", bytes.NewReader([]byte("anything")))
	req.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusNotFound; got != want {
		t.Fatalf("status: got %d, want %d (body: %s)", got, want, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "/v1/events/") {
		t.Errorf("error body should point at /v1/events/<type>: %q", body)
	}
}

// sanity: handler implements http.Handler.
var _ http.Handler = (*Handler)(nil)
var _ io.Writer = (*bytes.Buffer)(nil)
