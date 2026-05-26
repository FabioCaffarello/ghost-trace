package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

// requestIDContextKey is the unexported type used as the context-key
// for the per-request id per decision-log §0202 third observability
// advance. Unexported type prevents collisions with other packages'
// context values (per the stdlib convention: each package defines its
// own unexported key type for context.Value storage).
type requestIDContextKey struct{}

// RequestIDFromContext returns the per-request id stored in ctx via
// WithRequestIDContext, or empty string when absent. Exported for
// downstream consumers (e.g., handlers calling into substrate / ingest
// packages) that want to thread the id into their own structured
// outputs.
//
// Per §0202 wire contract: empty-string return is the documented
// "no id" signal; downstream consumers SHOULD treat it as
// uncorrelated (rather than panicking or generating their own id).
func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(requestIDContextKey{}).(string)
	return v
}

// WithRequestIDContext returns a copy of ctx carrying the request id
// under the package's unexported context-key. Used internally by
// ServeHTTP to thread the id into the request context before
// dispatching to the per-route handler; exported for test consumers
// that want to inject a known id into a synthetic context.
func WithRequestIDContext(ctx context.Context, id string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, requestIDContextKey{}, id)
}

// requestIDHeader is the canonical wire-name for the request-correlation
// identifier per decision-log §0198 second observability advance.
// Mirrors the de-facto industry convention (`X-Request-Id`) used by
// most reverse proxies + observability stacks; operators upstream of
// the ingestion service can propagate IDs already issued at the edge.
const requestIDHeader = "X-Request-Id"

// requestIDLength is the byte-count of the random portion of generated
// request IDs (32 hex chars = 16 random bytes). Chosen to match the
// length of standard UUID-4 hex encodings without depending on the
// `google/uuid` package — stdlib-only via crypto/rand.
const requestIDLength = 16

// generateRequestID produces a fresh 32-char hex-encoded request ID
// using crypto/rand. Falls back to "unknown-request-id" if rand.Read
// surfaces an error (extremely rare on production OS-level entropy
// sources; the fallback preserves the property that the log entry
// always carries a non-empty request_id field).
func generateRequestID() string {
	b := make([]byte, requestIDLength)
	if _, err := rand.Read(b); err != nil {
		return "unknown-request-id"
	}
	return hex.EncodeToString(b)
}

// resolveRequestID returns the request ID for the incoming request:
// the X-Request-Id header value when non-empty (preserving upstream-
// edge-issued IDs for correlation across services), else a freshly-
// generated one.
//
// Per §0198 wire contract: any non-empty header value is accepted
// verbatim — operators upstream may use UUIDs, ULIDs, or their own
// scheme. The httpapi layer does NOT validate ID format; downstream
// operators consuming the structured stream choose how to filter
// invalid IDs.
func resolveRequestID(r *http.Request) string {
	if id := r.Header.Get(requestIDHeader); id != "" {
		return id
	}
	return generateRequestID()
}
