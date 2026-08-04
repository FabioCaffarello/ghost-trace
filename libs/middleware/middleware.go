// Package middleware is the HTTP middleware chain every Ghost Trace
// service shares: request id, panic recovery, structured request
// logging, and Prometheus-style metrics.
//
// It is its own module rather than a package inside one service because
// Phase 2 splits the binary into four (collector, decision-engine,
// archive, demo-web) and every one of them serves HTTP. Extracting it
// here keeps the service PRs about their service.
//
// Stdlib only, deliberately. A shared library that drags dependencies
// into four services is a coupling nobody asked for.
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

// Middleware wraps an http.Handler.
type Middleware func(http.Handler) http.Handler

// Chain applies mws around h, first argument outermost. Chain(h, a, b)
// serves a(b(h)).
func Chain(h http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// requestIDKey is unexported so no other package can collide with the
// context entry (the stdlib convention for context.Value keys).
type requestIDKey struct{}

// RequestIDHeader is the wire name for the correlation id — the
// de-facto convention, so ids issued at an upstream edge propagate.
const RequestIDHeader = "X-Request-Id"

// RequestIDFromContext returns the request id, or "" when the request
// did not pass through RequestID. Empty is the documented "no id"
// signal — treat it as uncorrelated, don't invent one downstream.
func RequestIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(requestIDKey{}).(string)
	return v
}

// RequestID resolves the request's correlation id — the incoming
// X-Request-Id verbatim when present (upstream schemes are accepted,
// not validated), a fresh 32-hex-char id otherwise — then stores it in
// the context and echoes it on the response so a client can quote the
// id when reporting a failure.
func RequestID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := r.Header.Get(RequestIDHeader)
			if id == "" {
				id = newRequestID()
			}
			w.Header().Set(RequestIDHeader, id)
			next.ServeHTTP(w, r.WithContext(
				context.WithValue(r.Context(), requestIDKey{}, id)))
		})
	}
}

// newRequestID returns 16 random bytes as 32 hex chars — UUID-length
// without a uuid dependency. The fallback value keeps the field
// non-empty if the entropy source ever fails, which is the property
// the log line relies on.
func newRequestID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "unknown-request-id"
	}
	return hex.EncodeToString(b)
}

// statusRecorder captures the status code the inner handler wrote,
// which is otherwise unretrievable once the response has started.
type statusRecorder struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (sr *statusRecorder) WriteHeader(code int) {
	if !sr.wrote {
		sr.status = code
		sr.wrote = true
	}
	sr.ResponseWriter.WriteHeader(code)
}

func (sr *statusRecorder) Write(b []byte) (int, error) {
	// A body written without WriteHeader is an implicit 200, matching
	// net/http's own behavior.
	if !sr.wrote {
		sr.status = http.StatusOK
		sr.wrote = true
	}
	return sr.ResponseWriter.Write(b)
}

func (sr *statusRecorder) effectiveStatus() int {
	if !sr.wrote {
		return http.StatusOK
	}
	return sr.status
}

// routeLabel returns the bounded metrics/log label for a request: the
// ServeMux pattern that matched (available on the request once the mux
// has routed it, which is true by the time the deferred observation
// runs), or "unmatched" for requests no route claimed. Using the
// pattern rather than the raw path keeps label cardinality at the
// route count — a scanner probing random URLs must not mint a metric
// series per probe.
func routeLabel(r *http.Request) string {
	if r.Pattern != "" {
		return r.Pattern
	}
	return "unmatched"
}
