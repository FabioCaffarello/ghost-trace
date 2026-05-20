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
	fn := func(ctx context.Context, msg proto.Message, eventTime int64) (ingest.AppendReport, error) {
		idx := calls
		calls++
		if idx < len(outcomes) && outcomes[idx] != nil {
			return ingest.AppendReport{}, outcomes[idx]
		}
		return ingest.AppendReport{
			EventHashHex: "0000000000000000000000000000000000000000000000000000000000000001",
			PayloadBytes: 8,
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
	req := httptest.NewRequest(http.MethodPost, "/v1/events", bytes.NewReader(payload))
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
	req := httptest.NewRequest(http.MethodGet, "/v1/events", nil)
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusMethodNotAllowed; got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
}

func TestPostEventsRejectsWrongContentType(t *testing.T) {
	doAppend, _ := stubAppendFunc(nil)
	h := New(doAppend, nil)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/events", strings.NewReader("plain text"))
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
	req := httptest.NewRequest(http.MethodPost, "/v1/events", strings.NewReader(""))
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
	req := httptest.NewRequest(http.MethodPost, "/v1/events", strings.NewReader("not-a-protobuf-message"))
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
	req := httptest.NewRequest(http.MethodPost, "/v1/events", bytes.NewReader(payload))
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
	req := httptest.NewRequest(http.MethodPost, "/v1/events", bytes.NewReader(payload))
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
	req := httptest.NewRequest(http.MethodPost, "/v1/events", bytes.NewReader(payload))
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
	req := httptest.NewRequest(http.MethodPost, "/v1/events", bytes.NewReader(body))
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

// sanity: handler implements http.Handler.
var _ http.Handler = (*Handler)(nil)
var _ io.Writer = (*bytes.Buffer)(nil)
