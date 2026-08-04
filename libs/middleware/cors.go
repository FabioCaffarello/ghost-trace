package middleware

import (
	"net/http"
	"strings"
)

// CORS answers cross-origin requests from an ALLOWLIST, and from
// nothing else.
//
// WHY THIS EXISTS AT ALL. The browser endpoints are meant to be called
// from a page the collector does not serve — that is the ordinary
// integration, where the customer's site is on their domain and the
// collector is on another. The demo hid that requirement by being
// same-origin with the API, so nothing here ever needed it and nothing
// described it. Separating the demo host made the requirement visible
// rather than creating it.
//
// WHAT IT DELIBERATELY DOES NOT DO:
//
//   - No "*". An allowlist is the only form of this that means
//     anything, and an empty allowlist disables the whole mechanism
//     rather than defaulting to permissive.
//   - No Access-Control-Allow-Credentials. The SDK authenticates with a
//     bearer token it holds in memory, never a cookie. Allowing
//     credentials would let any allowed origin drive a browser's
//     ambient session, which is a strictly wider surface than the SDK
//     needs — and "wider than needed" is how this becomes a finding.
//   - It is NOT applied to /v1/decisions or /v1/outcomes. Those carry
//     the secret key and are server-to-server (§1). CORS on them would
//     invite a page to try, and the only way a browser could succeed is
//     if the secret had already been shipped to it.
//
// Vary: Origin is set on every response, allowed or not, because a
// cache that stored one origin's answer for another would defeat the
// allowlist without anyone editing it.
func CORS(origins []string, methods ...string) Middleware {
	allowed := make(map[string]bool, len(origins))
	for _, o := range origins {
		if o = strings.TrimSpace(o); o != "" {
			allowed[o] = true
		}
	}
	if len(methods) == 0 {
		methods = []string{http.MethodPost}
	}
	allowMethods := strings.Join(append(methods, http.MethodOptions), ", ")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set unconditionally: the header describes how this
			// response varies, which is true whether or not the request
			// carried an Origin.
			w.Header().Add("Vary", "Origin")

			origin := r.Header.Get("Origin")
			if origin != "" && allowed[origin] {
				h := w.Header()
				h.Set("Access-Control-Allow-Origin", origin)
				h.Set("Access-Control-Allow-Methods", allowMethods)
				h.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				h.Set("Access-Control-Max-Age", "600")
			}

			// A preflight is answered here and goes no further: it is a
			// question about permission, not a request the use case
			// should ever see. An origin that is not allowed gets the
			// same 204 with no allow headers, which is what a browser
			// needs to refuse the real request.
			if r.Method == http.MethodOptions &&
				r.Header.Get("Access-Control-Request-Method") != "" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
