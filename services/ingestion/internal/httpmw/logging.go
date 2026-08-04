package httpmw

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

// Logging emits one structured entry per completed request: method,
// route, status, duration and request id. One line per request is the
// contract — enough to reconstruct what the service did without
// per-handler printf archaeology.
func Logging(log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sr := &statusRecorder{ResponseWriter: w}
			start := time.Now()
			next.ServeHTTP(sr, r)
			log.Info("http request",
				"method", r.Method,
				"route", routeLabel(r),
				"path", r.URL.Path,
				"status", sr.effectiveStatus(),
				"duration_ms", float64(time.Since(start).Microseconds())/1000.0,
				"request_id", RequestIDFromContext(r.Context()),
			)
		})
	}
}

// Recovery converts a handler panic into a logged 500 instead of a
// killed connection. The service is fail-open by philosophy (§5); a
// panic that tears down the conn gives the caller a network error to
// diagnose, while a 500 with a request id gives them something to
// report.
func Recovery(log *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error("panic in handler",
						"panic", rec,
						"method", r.Method,
						"path", r.URL.Path,
						"request_id", RequestIDFromContext(r.Context()),
						"stack", string(debug.Stack()),
					)
					// Best-effort: if the handler already started the
					// response this write is a no-op on the status line.
					http.Error(w, "internal error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
