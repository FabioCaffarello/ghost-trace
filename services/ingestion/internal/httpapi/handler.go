// Package httpapi is the inception-phase HTTP interface for the
// ingestion service. It composes the same canonical + substrate stack
// the stdin worker uses, exposed over a minimum-viable HTTP surface:
//
//   - POST /v1/events  — accepts application/x-protobuf body; returns
//     200 with JSON confirmation on success, 400 on recoverable input
//     failure, 500 on unrecoverable substrate error (and signals the
//     service-level fatal channel for shutdown escalation).
//   - GET  /healthz    — liveness probe; returns 200 + {"status":"ok"}.
//
// All other paths return 404; non-matching methods return 405.
//
// Error classification mirrors readLoop's discipline per
// docs/architecture/concurrency-pattern.md §Error Propagation +
// decision-log §0032 (unrecoverable-error shutdown escalation):
// recoverable errors return 4xx + JSON body; unrecoverable errors
// return 500 + JSON body AND signal the fatal channel.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"google.golang.org/protobuf/proto"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// AppendFunc is the handler's dependency on the ingestion pipeline.
// Implemented in production by ingest.Ingester.Append; injectable in
// tests for unrecoverable-error path coverage.
type AppendFunc func(ctx context.Context, msg proto.Message, eventTime int64) (ingest.AppendReport, error)

// FatalReporter is the service-level escalation channel. Handlers call
// ReportFatal on unrecoverable errors; the service's errgroup
// coordinator returns the error from its goroutine, propagating
// shutdown per concurrency-pattern §Error Propagation.
type FatalReporter interface {
	ReportFatal(err error)
}

// Handler is the HTTP request multiplexer.
type Handler struct {
	doAppend AppendFunc
	fatal    FatalReporter

	// requestBodyLimit bounds the body bytes the handler reads per
	// request. Defends against unbounded-input DoS per
	// concurrency-pattern §Bounded Concurrency (analogue at the HTTP
	// layer). 1 MiB matches the readLoop scanner buffer ceiling.
	requestBodyLimit int64
}

// New constructs a Handler. doAppend MUST NOT be nil. fatal MAY be nil
// in tests where unrecoverable-error escalation is not exercised; in
// production main wires a real FatalReporter.
func New(doAppend AppendFunc, fatal FatalReporter) *Handler {
	if doAppend == nil {
		panic("httpapi.New: doAppend must not be nil")
	}
	return &Handler{
		doAppend:         doAppend,
		fatal:            fatal,
		requestBodyLimit: 1 << 20, // 1 MiB
	}
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/v1/events":
		h.handleEvents(w, r)
	case "/healthz":
		h.handleHealthz(w, r)
	default:
		http.NotFound(w, r)
	}
}

// confirmation is the structured per-message success outcome.
// Wire shape matches main.confirmation; the two channels (HTTP + stdin)
// emit the same record type so producers can rely on a single schema.
type confirmation struct {
	EventHash    string `json:"event_hash"`
	PayloadBytes int    `json:"payload_bytes"`
	CommittedAt  int64  `json:"committed_at_ns"`
}

// ingestError is the structured per-message error outcome. Wire shape
// matches main.ingestError.
type ingestError struct {
	Error string `json:"error"`
}

func (h *Handler) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, `{"status":"ok"}`+"\n")
}

func (h *Handler) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Content-Type validation. Inception phase supports only
	// application/x-protobuf to match the canonical-serialization-
	// contract wire shape. JSON-wrapped base64 input is the stdin
	// worker's I/O; the HTTP channel keeps the wire format binary.
	contentType := r.Header.Get("Content-Type")
	if contentType != "" && contentType != "application/x-protobuf" {
		writeIngestError(w, http.StatusUnsupportedMediaType,
			fmt.Sprintf("content-type %q not supported; use application/x-protobuf", contentType))
		return
	}

	bodyReader := http.MaxBytesReader(w, r.Body, h.requestBodyLimit)
	defer func() { _ = bodyReader.Close() }()

	payload, err := io.ReadAll(bodyReader)
	if err != nil {
		writeIngestError(w, http.StatusBadRequest, fmt.Sprintf("read body: %v", err))
		return
	}
	if len(payload) == 0 {
		writeIngestError(w, http.StatusBadRequest, "empty body")
		return
	}

	msg := &eventsv1.DeclaredSession{}
	if err := proto.Unmarshal(payload, msg); err != nil {
		writeIngestError(w, http.StatusBadRequest, fmt.Sprintf("proto unmarshal: %v", err))
		return
	}

	rep, err := h.doAppend(r.Context(), msg, msg.DeclaredAt)
	if err != nil {
		if isUnrecoverable(err) {
			// Write a 500 with a structured error body, then signal
			// the service-level fatal channel. The producer sees the
			// 500; the service shuts down asynchronously.
			writeIngestError(w, http.StatusInternalServerError, fmt.Sprintf("unrecoverable: %v", err))
			if h.fatal != nil {
				h.fatal.ReportFatal(err)
			}
			return
		}
		// All other Append errors are treated as recoverable: 400 with
		// a structured error body. This matches readLoop's discipline
		// (per main.go) where non-unrecoverable Append errors emit a
		// per-message ingestError to stdout and continue.
		writeIngestError(w, http.StatusBadRequest, fmt.Sprintf("ingest: %v", err))
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(confirmation{
		EventHash:    rep.EventHashHex,
		PayloadBytes: rep.PayloadBytes,
		CommittedAt:  time.Now().UnixNano(),
	})
}

// isUnrecoverable mirrors main.isUnrecoverable: substrate §2.1-violation
// errors trigger service-level shutdown per concurrency-pattern §Error
// Propagation. Duplicated rather than imported to keep the httpapi
// package's dependency surface limited to substrate + ingest +
// genproto + Go stdlib (no dep on main).
func isUnrecoverable(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, substrate.ErrHashMismatch) ||
		errors.Is(err, substrate.ErrBlobCollision)
}

// writeIngestError writes a structured ingestError JSON body with the
// given HTTP status code. Defensive: if the encoder fails (e.g.
// connection drop), the error is swallowed — the HTTP transport's
// failure is itself observable through other channels (Go's http
// package logs writer failures to its ErrorLog).
func writeIngestError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ingestError{Error: msg})
}
