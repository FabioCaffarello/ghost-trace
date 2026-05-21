// Package httpapi is the inception-phase HTTP interface for the
// ingestion service. It composes the same canonical + substrate stack
// the stdin worker uses, exposed over a minimum-viable HTTP surface:
//
//   - POST /v1/events/{type} — accepts application/x-protobuf body;
//     dispatches on {type} via ingest.LookupURLPath; returns 200 with
//     JSON confirmation on success, 400 on recoverable input failure,
//     404 on unknown type, 500 on unrecoverable substrate error (and
//     signals the service-level fatal channel for shutdown escalation).
//   - GET  /v1/hypotheses/state — single-projection projection read
//     keyed by formation event hash; per decision-log §0080. Mirrors
//     the cmd/hypothesis-state CLI's wire shape. Subtype auto-detect.
//     Requires substrate read access (injected via WithSubstrate).
//   - GET  /v1/hypotheses — multi-projection list with subtype/state/
//     time-window/limit/offset filters; per decision-log §0081.
//     Mirrors the cmd/list-hypotheses CLI's wire shape. Requires
//     substrate read access.
//   - GET  /v1/hypotheses/summary — aggregate counters + latency
//     aggregates across all four subtypes with combined section;
//     per decision-log §0082. Mirrors the cmd/summarize-hypotheses
//     CLI's wire shape. Requires substrate read access.
//   - GET  /v1/replay/operational-session — Phase 1 replay verifying
//     deterministic re-derivation of a Cat II OperationalSession; per
//     decision-log §0091. Mirrors cmd/replay-operational-session.
//   - GET  /v1/replay/operational-sessions — Phase 1 substrate-wide
//     batch replay; per §0091. Mirrors cmd/replay-all-operational-
//     sessions.
//   - GET  /v1/replay/formation — Phase 3 reconstructive replay of a
//     Cat III formation (subtype auto-detected from the row's
//     message_type); per §0091. Consolidates §0086-§0089's four
//     per-subtype CLIs into one auto-detect endpoint.
//   - GET  /v1/replay/formations — Phase 3 substrate-wide batch
//     replay across all four Cat III subtypes; per §0091. Mirrors
//     cmd/replay-all-formations.
//   - GET  /v1/verify — substrate-integrity audit; per decision-log
//     §0093. Mirrors cmd/verify (per §0039 + §0040). Walks every
//     events-table row + recomputes blob hashes; with
//     check_orphans=true, additionally walks the blob-store directory
//     to surface orphan blobs (harmless per §0033). Substrate-
//     integrity failures land as HTTP 200 with passed=false (same
//     drift semantic as §0091's replay endpoints).
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

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// eventsPathPrefix is the routing prefix for typed primary-observation
// ingestion. Trailing segment selects the message type via
// ingest.LookupURLPath.
const eventsPathPrefix = "/v1/events/"

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

// Tier identifies the operation tier of an HTTP route per decision-log
// §0094 classification. The four canonical tiers correspond to distinct
// authorization scopes: T1 producers commit Cat I observations; T2
// operator-read consumes projections + replay; T3 substrate-admin
// performs storage-layer maintenance; T4 constitutional-act commits
// Cat III lifecycle events. T0 (/healthz) is exempt — no Tier value.
//
// Tier is string-typed for forward-compatibility with the §0094
// classification's textual tier names. Unknown tier values are rejected
// at handler construction per decision-log §0098 + RFC item 6.
type Tier string

// The four canonical operation tiers per decision-log §0094.
const (
	TierProducer          Tier = "producer"
	TierOperatorRead      Tier = "operator-read"
	TierSubstrateAdmin    Tier = "substrate-admin"
	TierConstitutionalAct Tier = "constitutional-act"
)

// AllTiers returns the four canonical tiers in §0094 ordinal order.
func AllTiers() []Tier {
	return []Tier{TierProducer, TierOperatorRead, TierSubstrateAdmin, TierConstitutionalAct}
}

// validTier reports whether t is one of the four canonical tiers.
func validTier(t Tier) bool {
	switch t {
	case TierProducer, TierOperatorRead, TierSubstrateAdmin, TierConstitutionalAct:
		return true
	}
	return false
}

// Handler is the HTTP request multiplexer.
type Handler struct {
	doAppend AppendFunc
	fatal    FatalReporter

	// sub, when non-nil, enables the projection-read endpoints landed
	// at decision-log §0080+. Nil disables them (the routes return 404
	// matching the not-configured case rather than 500). Production
	// main wires the same *substrate.Substrate that backs doAppend.
	sub *substrate.Substrate

	// requestBodyLimit bounds the body bytes the handler reads per
	// request. Defends against unbounded-input DoS per
	// concurrency-pattern §Bounded Concurrency (analogue at the HTTP
	// layer). 1 MiB matches the readLoop scanner buffer ceiling.
	requestBodyLimit int64

	// authToken, when non-empty, requires producers to send
	// "Authorization: Bearer <authToken>" on every protected request.
	// Per §0035 single-token mode; the single token authorizes every
	// non-/healthz route (treated as the union of all four tiers per
	// RFC architecture-http-auth-scope-model item 1 backward-compat).
	// Empty disables authentication (backward-compatible default for
	// inception phase where producers are local + trusted). /healthz
	// is exempt unconditionally for liveness probing. Mutually
	// exclusive with tierTokens — both configured returns an error at
	// New.
	authToken string

	// tierTokens, when non-empty, activates multi-tier dispatch per
	// decision-log §0098 + RFC architecture-http-auth-scope-model
	// item 1. Each route's tier (per §0094 classification, encoded in
	// routeTier) selects the per-tier token to compare against the
	// request's Authorization: Bearer header. Tiers without a
	// configured token are unreachable — operators can opt out of
	// exposing T3 + T4 routes by omitting the corresponding tier
	// tokens. Mutually exclusive with authToken — both configured
	// returns an error at New.
	tierTokens map[Tier]string

	// authRealm is the value advertised in WWW-Authenticate on 401
	// responses. Defaults to "ghost-trace-ingestion".
	authRealm string
}

// Option configures a Handler at construction. See WithAuthToken,
// WithAuthTierToken, WithRequestBodyLimit.
type Option func(*Handler)

// WithAuthToken enables bearer-token authentication on protected
// endpoints. /v1/events/{type} requires `Authorization: Bearer <token>` with
// constant-time-compared <token>. /healthz remains unauthenticated for
// liveness probes. Empty token disables authentication (default).
//
// Bearer tokens are vulnerable to interception; production deployments
// SHOULD also terminate TLS (reverse proxy or follow-on TLS RFC) and
// store the token in a file with mode 0600 rather than passing it on
// the command line.
//
// Per RFC architecture-http-auth-scope-model item 1 backward-compat
// clause, WithAuthToken is the §0035 single-token mode — the configured
// token is treated as the union of all four tiers. Mutually exclusive
// with WithAuthTierToken; both configured returns an error at New.
func WithAuthToken(token string) Option {
	return func(h *Handler) { h.authToken = token }
}

// WithAuthTierToken enables per-tier bearer-token authentication on
// protected endpoints. Each call adds the token for one tier; multiple
// calls compose. Each protected route is annotated with its tier per
// the §0094 classification (encoded in routeTier); incoming requests
// must present the per-tier token for the route's tier or receive 401.
//
// Tiers without a configured token are unreachable — operators opt out
// of exposing T3 + T4 routes by omitting the corresponding tier
// tokens. /healthz remains unauthenticated regardless of multi-tier
// configuration (T0 exemption per §0035 + §0094).
//
// Mutually exclusive with WithAuthToken; both configured returns an
// error at New. Unknown tier values (not in AllTiers) return an error
// at New. Per decision-log §0098 + RFC architecture-http-auth-scope-
// model item 1.
func WithAuthTierToken(tier Tier, token string) Option {
	return func(h *Handler) {
		if h.tierTokens == nil {
			h.tierTokens = make(map[Tier]string)
		}
		h.tierTokens[tier] = token
	}
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

// WithSubstrate enables the projection-read endpoints (per §0080+).
// When nil/unset, the read routes return 404 (the endpoints are
// effectively not registered). Production main wires the same
// *substrate.Substrate that backs doAppend.
func WithSubstrate(sub *substrate.Substrate) Option {
	return func(h *Handler) { h.sub = sub }
}

// New constructs a Handler. doAppend MUST NOT be nil. fatal MAY be nil
// in tests where unrecoverable-error escalation is not exercised; in
// production main wires a real FatalReporter. Options apply
// configuration overrides in order.
//
// Returns an error when configuration is invalid: (a) doAppend is nil;
// (b) both WithAuthToken and WithAuthTierToken are configured (single-
// token and multi-tier modes are mutually exclusive per RFC
// architecture-http-auth-scope-model item 1); (c) WithAuthTierToken
// names a tier outside AllTiers (forward-compat guard against typos in
// future tier names). Per decision-log §0098 + RFC item 6.
func New(doAppend AppendFunc, fatal FatalReporter, opts ...Option) (*Handler, error) {
	if doAppend == nil {
		return nil, errors.New("httpapi.New: doAppend must not be nil")
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
	if h.authToken != "" && len(h.tierTokens) > 0 {
		return nil, errors.New("httpapi.New: WithAuthToken and WithAuthTierToken are mutually exclusive (configure single-token OR per-tier, not both)")
	}
	for t := range h.tierTokens {
		if !validTier(t) {
			return nil, fmt.Errorf("httpapi.New: unknown tier %q (valid: %v)", t, AllTiers())
		}
	}
	return h, nil
}

// MustNew is a New variant that panics on validation error. Intended
// for test setup where configuration is statically known-valid; the
// validation panic surfaces a test misconfiguration immediately.
// Production code calls New and handles the returned error.
func MustNew(doAppend AppendFunc, fatal FatalReporter, opts ...Option) *Handler {
	h, err := New(doAppend, fatal, opts...)
	if err != nil {
		panic(err)
	}
	return h
}

// routeTier returns the operation tier of a route per the §0094
// classification, or empty Tier for T0 routes (/healthz exemption) and
// unknown paths (returns 404 below).
//
// The classification is wire-format-agnostic (per §0094): the same
// tiers apply to a future gRPC interface (§0095 reframing). Routes not
// yet implemented (T3 admin endpoints, T4 lifecycle endpoints) are
// listed below; their tier annotations are pre-positioned for the
// named follow-on landings.
func routeTier(r *http.Request) Tier {
	p := r.URL.Path
	if p == "/healthz" {
		return ""
	}
	if strings.HasPrefix(p, eventsPathPrefix) {
		return TierProducer
	}
	if p == "/v1/hypotheses" ||
		p == "/v1/hypotheses/state" ||
		p == "/v1/hypotheses/summary" ||
		p == "/v1/replay/operational-session" ||
		p == "/v1/replay/operational-sessions" ||
		p == "/v1/replay/formation" ||
		p == "/v1/replay/formations" ||
		p == "/v1/verify" {
		return TierOperatorRead
	}
	if p == "/v1/admin/orphan-cleanup" {
		return TierSubstrateAdmin
	}
	if p == "/v1/hypotheses/behavioral-cluster/promote" ||
		p == "/v1/hypotheses/automation-group/promote" ||
		p == "/v1/hypotheses/campaign-hypothesis/promote" ||
		p == "/v1/hypotheses/coordination-ring/promote" {
		return TierConstitutionalAct
	}
	// Remaining T4 constitutional-act routes (named follow-on per
	// §0098 + §0105 pilot): the other 20 endpoints — promote is
	// done across all four subtypes; form/demote/dissolve/merge/split
	// × 4 subtypes remain pre-positioned for follow-on landings.
	return ""
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

	switch {
	case r.URL.Path == "/healthz":
		h.handleHealthz(w, r)
	case r.URL.Path == "/v1/events" || r.URL.Path == "/v1/events/":
		// Untyped path: surface the dispatch contract rather than 404
		// so producers running against pre-dispatch wire shape see a
		// clear migration hint.
		writeIngestError(w, http.StatusNotFound,
			fmt.Sprintf("message type required; POST to /v1/events/<type> where <type> is one of: %s",
				strings.Join(ingest.KnownURLPaths(), ", ")))
	case strings.HasPrefix(r.URL.Path, eventsPathPrefix):
		h.handleEvents(w, r)
	case r.URL.Path == "/v1/hypotheses/state":
		h.handleHypothesisState(w, r)
	case r.URL.Path == "/v1/hypotheses":
		h.handleHypothesisList(w, r)
	case r.URL.Path == "/v1/hypotheses/summary":
		h.handleHypothesisSummary(w, r)
	case r.URL.Path == "/v1/replay/operational-session":
		h.handleReplayOperationalSession(w, r)
	case r.URL.Path == "/v1/replay/operational-sessions":
		h.handleReplayAllOperationalSessions(w, r)
	case r.URL.Path == "/v1/replay/formation":
		h.handleReplayFormation(w, r)
	case r.URL.Path == "/v1/replay/formations":
		h.handleReplayAllFormations(w, r)
	case r.URL.Path == "/v1/verify":
		h.handleVerify(w, r)
	case r.URL.Path == "/v1/admin/orphan-cleanup":
		h.handleAdminOrphanCleanup(w, r)
	case r.URL.Path == "/v1/hypotheses/behavioral-cluster/promote":
		h.handlePromoteBehavioralCluster(w, r)
	case r.URL.Path == "/v1/hypotheses/automation-group/promote":
		h.handlePromoteAutomationGroup(w, r)
	case r.URL.Path == "/v1/hypotheses/campaign-hypothesis/promote":
		h.handlePromoteCampaignHypothesis(w, r)
	case r.URL.Path == "/v1/hypotheses/coordination-ring/promote":
		h.handlePromoteCoordinationRing(w, r)
	default:
		http.NotFound(w, r)
	}
}

// requiresAuth reports whether the request must carry a valid bearer
// token. Returns false when no token is configured (auth disabled —
// single-token AND tier-tokens both empty) or when the path is the
// exempt /healthz liveness probe.
func (h *Handler) requiresAuth(r *http.Request) bool {
	if r.URL.Path == "/healthz" {
		return false
	}
	if h.authToken == "" && len(h.tierTokens) == 0 {
		return false
	}
	return true
}

// authorized reports whether the request's Authorization header carries
// "Bearer <token>" matching the route's expected token under constant-
// time comparison.
//
// In single-token mode (h.authToken non-empty, h.tierTokens empty), the
// configured token is the union of all four tiers per RFC architecture-
// http-auth-scope-model item 1 backward-compat clause.
//
// In multi-tier mode (h.tierTokens non-empty, h.authToken empty), the
// route's tier per routeTier selects the per-tier token; tiers without
// a configured token are unreachable (operator opt-out).
//
// Routes not in routeTier's classification (T0 /healthz exempt; unknown
// paths reach ServeHTTP's 404 path) do not exercise this function.
func (h *Handler) authorized(r *http.Request) bool {
	const prefix = "Bearer "
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, prefix) {
		return false
	}
	provided := header[len(prefix):]

	if len(h.tierTokens) > 0 {
		// Multi-tier dispatch: lookup route's tier; reject if route is
		// unclassified under multi-tier mode (defense per RFC AP1 tier
		// conflation — unknown routes must not silently pass).
		tier := routeTier(r)
		if tier == "" {
			return false
		}
		token, ok := h.tierTokens[tier]
		if !ok {
			return false
		}
		return constantTimeMatch(provided, token)
	}

	// Single-token mode: legacy §0035 behavior.
	return constantTimeMatch(provided, h.authToken)
}

// constantTimeMatch compares two strings under constant-time semantics.
// Returns true iff lengths match AND bytes match. Length mismatch
// short-circuits (subtle.ConstantTimeCompare requires equal-length
// inputs). The length-leak channel (attacker learns token length on
// length mismatch) is acceptable at inception phase.
func constantTimeMatch(provided, expected string) bool {
	if len(provided) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
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

	typePath := strings.TrimPrefix(r.URL.Path, eventsPathPrefix)
	if typePath == "" || strings.ContainsRune(typePath, '/') {
		writeIngestError(w, http.StatusNotFound,
			fmt.Sprintf("message type %q not registered; POST to /v1/events/<type> where <type> is one of: %s",
				typePath, strings.Join(ingest.KnownURLPaths(), ", ")))
		return
	}
	desc, ok := ingest.LookupURLPath(typePath)
	if !ok {
		writeIngestError(w, http.StatusNotFound,
			fmt.Sprintf("message type %q not registered; known types: %s",
				typePath, strings.Join(ingest.KnownURLPaths(), ", ")))
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

	msg := desc.New()
	if err := proto.Unmarshal(payload, msg); err != nil {
		writeIngestError(w, http.StatusBadRequest, fmt.Sprintf("proto unmarshal: %v", err))
		return
	}

	env := envelopeForRequest(r)
	rep, err := h.doAppend(r.Context(), msg, desc.EventTime(msg), env)
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
