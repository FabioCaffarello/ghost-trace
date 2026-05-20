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
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// AppendFunc is the handler's dependency on the ingestion pipeline.
// Implemented in production by ingest.Ingester.Append; injectable in
// tests for unrecoverable-error path coverage.
type AppendFunc func(ctx context.Context, msg proto.Message, eventTime int64, env ingest.Envelope) (ingest.AppendReport, error)

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

	// authToken, when non-empty, requires producers to send
	// "Authorization: Bearer <authToken>" on every protected request.
	// Empty disables authentication (backward-compatible default for
	// inception phase where producers are local + trusted). /healthz
	// is exempt unconditionally for liveness probing.
	authToken string

	// authRealm is the value advertised in WWW-Authenticate on 401
	// responses. Defaults to "ghost-trace-ingestion".
	authRealm string
}

// Option configures a Handler at construction. See WithAuthToken,
// WithRequestBodyLimit.
type Option func(*Handler)

// WithAuthToken enables bearer-token authentication on protected
// endpoints. /v1/events requires `Authorization: Bearer <token>` with
// constant-time-compared <token>. /healthz remains unauthenticated for
// liveness probes. Empty token disables authentication (default).
//
// Bearer tokens are vulnerable to interception; production deployments
// SHOULD also terminate TLS (reverse proxy or follow-on TLS RFC) and
// store the token in a file with mode 0600 rather than passing it on
// the command line.
func WithAuthToken(token string) Option {
	return func(h *Handler) { h.authToken = token }
}

// WithAuthRealm overrides the WWW-Authenticate realm string. Operator
// convenience; default "ghost-trace-ingestion" suffices in most cases.
func WithAuthRealm(realm string) Option {
	return func(h *Handler) { h.authRealm = realm }
}

// WithRequestBodyLimit overrides the per-request body-size cap. Default
// 1 MiB matches the readLoop scanner buffer ceiling.
func WithRequestBodyLimit(n int64) Option {
	return func(h *Handler) { h.requestBodyLimit = n }
}

// New constructs a Handler. doAppend MUST NOT be nil. fatal MAY be nil
// in tests where unrecoverable-error escalation is not exercised; in
// production main wires a real FatalReporter. Options apply
// configuration overrides in order.
func New(doAppend AppendFunc, fatal FatalReporter, opts ...Option) *Handler {
	if doAppend == nil {
		panic("httpapi.New: doAppend must not be nil")
	}
	h := &Handler{
		doAppend:         doAppend,
		fatal:            fatal,
		requestBodyLimit: 1 << 20, // 1 MiB
		authRealm:        "ghost-trace-ingestion",
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// ServeHTTP implements http.Handler. Auth check is mux-level: if a
// bearer token is configured, every path except /healthz requires a
// valid Authorization header. Unauthenticated probes for unknown paths
// return 401 (not 404) so the path structure is not leaked to
// unauthenticated clients.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.requiresAuth(r) && !h.authorized(r) {
		h.writeUnauthorized(w)
		return
	}

	switch r.URL.Path {
	case "/v1/events":
		h.handleEvents(w, r)
	case "/healthz":
		h.handleHealthz(w, r)
	default:
		http.NotFound(w, r)
	}
}

// requiresAuth reports whether the request must carry a valid bearer
// token. Returns false when no token is configured (auth disabled) or
// when the path is the exempt /healthz liveness probe.
func (h *Handler) requiresAuth(r *http.Request) bool {
	if h.authToken == "" {
		return false
	}
	if r.URL.Path == "/healthz" {
		return false
	}
	return true
}

// authorized reports whether the request's Authorization header carries
// "Bearer <token>" matching the configured authToken under constant-time
// comparison.
func (h *Handler) authorized(r *http.Request) bool {
	const prefix = "Bearer "
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, prefix) {
		return false
	}
	provided := header[len(prefix):]
	// Pad lengths to enable constant-time comparison even when the
	// provided token has a different length than the configured one.
	// subtle.ConstantTimeCompare requires equal lengths; using fixed-
	// length comparison via SHA-equivalent would be stricter, but for
	// inception phase the length-leak channel (attacker learns token
	// length on length mismatch) is acceptable.
	if len(provided) != len(h.authToken) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(h.authToken)) == 1
}

// writeUnauthorized emits a 401 with a WWW-Authenticate header advertising
// the realm + a JSON error body matching the ingestError wire shape.
func (h *Handler) writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer realm=%q`, h.authRealm))
	writeIngestError(w, http.StatusUnauthorized, "missing or invalid Authorization header (Bearer token required)")
}

// confirmation is the structured per-message success outcome.
// Wire shape matches main.confirmation; the two channels (HTTP + stdin)
// emit the same record type so producers can rely on a single schema.
type confirmation struct {
	EventHash          string `json:"event_hash"`
	IngestionEventHash string `json:"ingestion_event_hash"`
	PayloadBytes       int    `json:"payload_bytes"`
	CommittedAt        int64  `json:"committed_at_ns"`
}

// envelopeForRequest derives the ingest.Envelope from the HTTP request,
// populating channel + client identity from the verified mTLS peer
// certificate when present. When the connection is plain HTTP the
// envelope reports channel="http"; plain HTTPS without client cert is
// "https"; HTTPS with verified client cert is "https+mtls".
func envelopeForRequest(r *http.Request) ingest.Envelope {
	env := ingest.Envelope{ReceivedAt: time.Now().UnixNano()}
	if r.TLS == nil {
		env.Channel = "http"
		return env
	}
	if len(r.TLS.VerifiedChains) == 0 || len(r.TLS.PeerCertificates) == 0 {
		// TLS handshake completed (server cert presented + verified by
		// client) but no client cert was verified — plain HTTPS.
		env.Channel = "https"
		return env
	}
	env.Channel = "https+mtls"
	peerCert := r.TLS.PeerCertificates[0]
	env.ClientCommonName = peerCert.Subject.CommonName
	env.ClientSubjectAltNames = mTLSSubjectAltNames(peerCert)
	sum := sha256.Sum256(peerCert.Raw)
	env.ClientCertSHA256 = hex.EncodeToString(sum[:])
	return env
}

// mTLSSubjectAltNames returns the SAN entries from cert in a stable
// order: DNS names, then IP addresses (as strings), then URIs, then
// email addresses. Used to populate the IngestionEvent's
// client_subject_alt_names field per §0038.
func mTLSSubjectAltNames(cert *x509.Certificate) []string {
	if cert == nil {
		return nil
	}
	var out []string
	out = append(out, cert.DNSNames...)
	for _, ip := range cert.IPAddresses {
		out = append(out, ip.String())
	}
	for _, u := range cert.URIs {
		out = append(out, u.String())
	}
	out = append(out, cert.EmailAddresses...)
	return out
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

	env := envelopeForRequest(r)
	rep, err := h.doAppend(r.Context(), msg, msg.DeclaredAt, env)
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
		EventHash:          rep.EventHashHex,
		IngestionEventHash: rep.IngestionEventHashHex,
		PayloadBytes:       rep.PayloadBytes,
		CommittedAt:        time.Now().UnixNano(),
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
