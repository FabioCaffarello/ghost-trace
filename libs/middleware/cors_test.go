package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/FabioCaffarello/ghost-trace/libs/middleware"
)

func corsServer(origins ...string) http.Handler {
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte("reached"))
	})
	return middleware.CORS(origins)(inner)
}

func do(h http.Handler, method, origin, preflightFor string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "/v1/telemetry", nil)
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	if preflightFor != "" {
		r.Header.Set("Access-Control-Request-Method", preflightFor)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestAnAllowedOriginIsEchoedBack(t *testing.T) {
	w := do(corsServer("https://shop.example"), http.MethodPost, "https://shop.example", "")
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://shop.example" {
		t.Errorf("allow-origin = %q, want the origin echoed", got)
	}
	if w.Code != http.StatusAccepted {
		t.Errorf("status = %d; the request should still have reached the handler", w.Code)
	}
}

func TestAnUnknownOriginGetsNoAllowHeader(t *testing.T) {
	w := do(corsServer("https://shop.example"), http.MethodPost, "https://evil.example", "")
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("allow-origin = %q for an origin not on the allowlist", got)
	}
	// The request is NOT rejected server-side: CORS is a browser
	// mechanism, and a non-browser client was never constrained by it.
	// Pretending otherwise would suggest a protection that is not there.
	if w.Code != http.StatusAccepted {
		t.Errorf("status = %d; CORS must not double as authorization", w.Code)
	}
}

func TestNoOriginIsNeverAllowlisted(t *testing.T) {
	// An empty allowlist entry must not turn a request with no Origin
	// into an allowed one.
	w := do(middleware.CORS([]string{"", "  "})(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {})), http.MethodPost, "", "")
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("allow-origin = %q with an all-empty allowlist", got)
	}
}

func TestWildcardIsNotAThingThatCanBeConfigured(t *testing.T) {
	// "*" is only ever matched literally: an origin is a string on the
	// list or it is not. If a future edit reintroduced wildcard
	// semantics, this passes an arbitrary origin and fails.
	w := do(corsServer("*"), http.MethodPost, "https://anything.example", "")
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("allow-origin = %q; a literal star must not allow an arbitrary origin", got)
	}
}

func TestCredentialsAreNeverAllowed(t *testing.T) {
	// The SDK holds a bearer token in memory and sends no cookies.
	// Allowing credentials would let an allowed origin drive a
	// browser's ambient session — wider than anything here needs.
	w := do(corsServer("https://shop.example"), http.MethodPost, "https://shop.example", "")
	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Errorf("allow-credentials = %q, want unset", got)
	}
}

func TestPreflightIsAnsweredAndNeverReachesTheHandler(t *testing.T) {
	w := do(corsServer("https://shop.example"), http.MethodOptions,
		"https://shop.example", http.MethodPost)
	if w.Code != http.StatusNoContent {
		t.Errorf("preflight status = %d, want 204", w.Code)
	}
	if w.Body.String() != "" {
		t.Errorf("preflight reached the handler (body %q)", w.Body.String())
	}
	if got := w.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Error("preflight carries no allow-headers; the browser cannot send Authorization")
	}
}

func TestVaryOriginIsAlwaysSet(t *testing.T) {
	// Without it a shared cache can serve one origin's response — allow
	// header and all — to another, defeating the allowlist with nobody
	// having edited it.
	for _, origin := range []string{"https://shop.example", "https://evil.example", ""} {
		w := do(corsServer("https://shop.example"), http.MethodPost, origin, "")
		if got := w.Header().Get("Vary"); got != "Origin" {
			t.Errorf("Vary = %q for origin %q, want Origin", got, origin)
		}
	}
}
