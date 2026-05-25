package httpapi

import (
	"io"
	"log/slog"
	"net/http"
)

// defaultLogger is the no-op slog.Logger used when WithLogger is not
// configured. Emits nothing — preserves the pre-§0197 behavior of
// silent handler operation. Tests that exercise log-emission paths
// SHOULD construct their own logger via WithLogger.
var defaultLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

// loggingResponseWriter wraps http.ResponseWriter to capture the status
// code written by the inner handler. Mirrors the standard middleware
// idiom; needed because ServeHTTP must emit a per-request structured
// log entry AFTER the handler completes + the status is otherwise
// not retrievable from the writer.
//
// Per §0197 first observability advance: only the status field is
// captured here; future advances may extend with response body size,
// content type, etc. as operator pressure surfaces.
type loggingResponseWriter struct {
	http.ResponseWriter
	status int
	wrote  bool
}

// WriteHeader records the status code + delegates to the wrapped writer.
// Idempotent across multiple calls (only the first WriteHeader takes
// effect per net/http contract; the captured status reflects the first
// call).
func (lw *loggingResponseWriter) WriteHeader(code int) {
	if !lw.wrote {
		lw.status = code
		lw.wrote = true
	}
	lw.ResponseWriter.WriteHeader(code)
}

// Write delegates to the wrapped writer + implicitly records status
// 200 when the handler writes the body without calling WriteHeader
// (matches net/http's implicit-200 behavior at the writer layer).
func (lw *loggingResponseWriter) Write(b []byte) (int, error) {
	if !lw.wrote {
		lw.status = http.StatusOK
		lw.wrote = true
	}
	return lw.ResponseWriter.Write(b)
}

// effectiveStatus returns the captured status, defaulting to 200 when
// the handler emitted no body + no explicit WriteHeader (rare; included
// for completeness so the log always carries a status value).
func (lw *loggingResponseWriter) effectiveStatus() int {
	if !lw.wrote {
		return http.StatusOK
	}
	return lw.status
}
