// Command ingestion is the inception-phase ingestion service per
// decision-log §0022 (implementation pivot) + §0024 (Protobuf proto3
// schemas) + §0025 (Go) + §0027 (SQLite + content-addressed blob-store).
//
// Architecture: docs/architecture/canonical-serialization-contract.md +
// docs/architecture/concurrency-pattern.md.
//
// Inception-phase I/O: reads newline-delimited JSON envelope lines from
// stdin (one per primary observation) and writes one-line JSON
// confirmation per ingested message to stdout. Each envelope is
// `{"type":"<type>","payload_b64":"<base64-proto>"}` where <type> is a
// registered Cat I message type per ingest.Registry (e.g.
// "declared_session", "network_event"). gRPC interface deferred to a
// follow-on RFC (per decision-log §0025 Open Questions).
package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/httpapi"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// stdinEnvelope is the per-line wire shape on stdin. `type` selects
// the Cat I message type via ingest.LookupStdinType; `payload_b64` is
// the base64-encoded canonical-Protobuf payload.
type stdinEnvelope struct {
	Type       string `json:"type"`
	PayloadB64 string `json:"payload_b64"`
}

// exitCodeUnrecoverable is the process exit code on unrecoverable
// errors per docs/architecture/concurrency-pattern.md §Error
// Propagation step 4.
const exitCodeUnrecoverable = 1

func main() {
	if err := run(); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return // graceful shutdown via signal
		}
		// Unrecoverable-error structured shutdown per concurrency-pattern
		// §Error Propagation: write a structured-output log entry to
		// stderr identifying the error class + exit non-zero.
		emitFatal(os.Stderr, err)
		os.Exit(exitCodeUnrecoverable)
	}
}

func run() error {
	dbPath := flag.String("db", "./ghost-trace.db", "SQLite primary-event-log path")
	blobDir := flag.String("blobs", "./blobs", "content-addressed blob-store directory")
	httpAddr := flag.String("http", "", "HTTP listen address (e.g. :8080); empty disables the HTTP server")
	httpAuthToken := flag.String("http-auth-token", "", "bearer token for HTTP authentication (inline; prefer --http-auth-token-file for production). Empty disables auth (default). Mutually exclusive with --http-auth-{tier}-token-file flags (multi-tier mode per decision-log §0098).")
	httpAuthTokenFile := flag.String("http-auth-token-file", "", "path to a file containing the bearer token (single line; trailing whitespace trimmed). Takes precedence over --http-auth-token. Mutually exclusive with --http-auth-{tier}-token-file flags.")
	httpAuthProducerTokenFile := flag.String("http-auth-producer-token-file", "", "path to a file containing the producer-tier bearer token (T1: POST /v1/events/{type}). Per decision-log §0098. Mutually exclusive with --http-auth-token{,-file}.")
	httpAuthOperatorReadTokenFile := flag.String("http-auth-operator-read-token-file", "", "path to a file containing the operator-read-tier bearer token (T2: GET /v1/hypotheses/*, /v1/replay/*, /v1/verify). Per decision-log §0098.")
	httpAuthSubstrateAdminTokenFile := flag.String("http-auth-substrate-admin-token-file", "", "path to a file containing the substrate-admin-tier bearer token (T3: /v1/admin/*; routes ship under named follow-on per decision-log §0098).")
	httpAuthConstitutionalActTokenFile := flag.String("http-auth-constitutional-act-token-file", "", "path to a file containing the constitutional-act-tier bearer token (T4: Cat III lifecycle HTTP endpoints; routes ship under named follow-on per decision-log §0098).")
	httpTLSCert := flag.String("http-tls-cert", "", "path to PEM-encoded TLS certificate (server cert + optional intermediate chain). Required with --http-tls-key.")
	httpTLSKey := flag.String("http-tls-key", "", "path to PEM-encoded TLS private key. Required with --http-tls-cert.")
	httpTLSClientCA := flag.String("http-tls-client-ca", "", "path to PEM-encoded CA bundle used to verify client certificates (enables mTLS). Empty disables client-cert verification. Requires --http-tls-cert + --http-tls-key.")
	flag.Parse()

	// Root context cancellable on SIGINT / SIGTERM per concurrency-pattern §Context Propagation.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	sub, err := substrate.Open(ctx, *dbPath, *blobDir)
	if err != nil {
		return fmt.Errorf("open substrate: %w", err)
	}
	defer func() { _ = sub.Close() }()

	in := ingest.New(sub, time.Now)
	reporter := &fatalReporter{}

	// errgroup orchestrates per concurrency-pattern §Goroutine Lifecycle.
	// Three goroutines may run:
	//   1. Stdin worker — always.
	//   2. HTTP server — only when --http is set.
	//   3. Fatal coordinator — propagates unrecoverable errors reported
	//      by HTTP handlers to the errgroup.
	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error { return readLoop(gctx, in.Append, os.Stdin, os.Stdout) })

	g.Go(func() error {
		select {
		case err := <-reporter.signal():
			return err
		case <-gctx.Done():
			return nil
		}
	})

	if *httpAddr != "" {
		token, err := resolveAuthToken(*httpAuthToken, *httpAuthTokenFile)
		if err != nil {
			return fmt.Errorf("resolve HTTP auth token: %w", err)
		}
		tierTokens, tierTokenIDs, err := resolveTierTokens(map[httpapi.Tier]string{
			httpapi.TierProducer:          *httpAuthProducerTokenFile,
			httpapi.TierOperatorRead:      *httpAuthOperatorReadTokenFile,
			httpapi.TierSubstrateAdmin:    *httpAuthSubstrateAdminTokenFile,
			httpapi.TierConstitutionalAct: *httpAuthConstitutionalActTokenFile,
		})
		if err != nil {
			return fmt.Errorf("resolve HTTP per-tier auth tokens: %w", err)
		}
		tlsCfg, err := resolveHTTPTLS(*httpTLSCert, *httpTLSKey, *httpTLSClientCA)
		if err != nil {
			return fmt.Errorf("resolve HTTP TLS config: %w", err)
		}
		var handlerOpts []httpapi.Option
		if token != "" {
			handlerOpts = append(handlerOpts, httpapi.WithAuthToken(token))
		}
		for tier, tk := range tierTokens {
			handlerOpts = append(handlerOpts, httpapi.WithAuthTierToken(tier, tk))
		}
		for tier, tkID := range tierTokenIDs {
			handlerOpts = append(handlerOpts, httpapi.WithAuthTierTokenID(tier, tkID))
		}
		// Per decision-log §0080: enable projection-read endpoints
		// (GET /v1/hypotheses/state and follow-ons). Same substrate
		// backs the write side.
		handlerOpts = append(handlerOpts, httpapi.WithSubstrate(sub))
		handler, err := httpapi.New(in.Append, reporter, handlerOpts...)
		if err != nil {
			return fmt.Errorf("construct HTTP handler: %w", err)
		}
		srv := &http.Server{
			Addr:              *httpAddr,
			Handler:           handler,
			ReadHeaderTimeout: 10 * time.Second,
		}
		if tlsCfg.enabled {
			// MinVersion floor at TLS 1.2 per modern best practice. TLS 1.0
			// and 1.1 are widely deprecated (RFC 8996 marks them historic).
			// Operators wanting TLS 1.3-only may add a follow-on option;
			// the floor here protects against accidental weak-protocol
			// exposure.
			srv.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
			if tlsCfg.clientCAPool != nil {
				// mTLS enabled: client MUST present a certificate; server
				// verifies it against the CA bundle. Connections without
				// a valid client cert are rejected at the TLS handshake
				// layer (before any HTTP request is processed). Composes
				// with bearer-token auth: when both are configured, both
				// MUST pass.
				srv.TLSConfig.ClientAuth = tls.RequireAndVerifyClientCert
				srv.TLSConfig.ClientCAs = tlsCfg.clientCAPool
			}
		}
		g.Go(func() error {
			var listenErr error
			if tlsCfg.enabled {
				listenErr = srv.ListenAndServeTLS(tlsCfg.certFile, tlsCfg.keyFile)
			} else {
				listenErr = srv.ListenAndServe()
			}
			if listenErr != nil && !errors.Is(listenErr, http.ErrServerClosed) {
				return fmt.Errorf("http listen: %w", listenErr)
			}
			return nil
		})
		g.Go(func() error {
			<-gctx.Done()
			// Detached shutdown context with a bounded grace period; never
			// inherits gctx because gctx is already canceled by the time
			// this fires.
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			return srv.Shutdown(shutdownCtx)
		})
	}

	if err := g.Wait(); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err // pass through to main() for clean exit
		}
		if errors.Is(err, io.EOF) {
			return nil // EOF on stdin = clean end-of-input
		}
		return fmt.Errorf("worker terminated: %w", err)
	}
	return nil
}

// httpTLS captures the resolved TLS configuration for the HTTP server.
type httpTLS struct {
	certFile     string
	keyFile      string
	clientCAPool *x509.CertPool // non-nil enables mTLS (RequireAndVerifyClientCert)
	enabled      bool
}

// resolveHTTPTLS validates the TLS configuration. cert + key follow
// both-or-neither semantics; clientCAFile (if set) enables mTLS and
// requires cert + key. All file paths are eagerly stat-checked / parsed
// so misconfiguration fails at startup rather than at first connection.
//
// Outcomes:
//   - cert, key, clientCA all empty: TLS disabled.
//   - cert + key set, clientCA empty: TLS enabled, server-only auth.
//   - cert + key + clientCA all set: TLS enabled, mTLS (server verifies
//     client certs against the CA bundle).
//   - Any other combination: error at startup.
func resolveHTTPTLS(certFile, keyFile, clientCAFile string) (httpTLS, error) {
	switch {
	case certFile == "" && keyFile == "" && clientCAFile == "":
		return httpTLS{enabled: false}, nil
	case certFile == "" && keyFile == "" && clientCAFile != "":
		return httpTLS{}, fmt.Errorf("--http-tls-client-ca requires --http-tls-cert and --http-tls-key")
	case certFile == "" || keyFile == "":
		return httpTLS{}, fmt.Errorf("--http-tls-cert and --http-tls-key must both be set or both be empty (got cert=%q, key=%q)", certFile, keyFile)
	}
	// Eager-readability check: surface mis-paths at startup, not on
	// first connection.
	if _, err := os.Stat(certFile); err != nil {
		return httpTLS{}, fmt.Errorf("stat cert file %q: %w", certFile, err)
	}
	if _, err := os.Stat(keyFile); err != nil {
		return httpTLS{}, fmt.Errorf("stat key file %q: %w", keyFile, err)
	}
	out := httpTLS{certFile: certFile, keyFile: keyFile, enabled: true}
	if clientCAFile != "" {
		caBytes, err := os.ReadFile(clientCAFile)
		if err != nil {
			return httpTLS{}, fmt.Errorf("read client-CA file %q: %w", clientCAFile, err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caBytes) {
			return httpTLS{}, fmt.Errorf("client-CA file %q contains no parseable PEM certificates", clientCAFile)
		}
		out.clientCAPool = pool
	}
	return out, nil
}

// resolveAuthToken returns the effective bearer token for the HTTP
// interface. Precedence:
//  1. tokenFile (read + trim whitespace) if non-empty.
//  2. tokenInline if non-empty.
//  3. "" — auth disabled.
//
// The file path is preferred for production deployments to avoid token
// leakage via process listings.
func resolveAuthToken(tokenInline, tokenFile string) (string, error) {
	if tokenFile != "" {
		raw, err := os.ReadFile(tokenFile)
		if err != nil {
			return "", fmt.Errorf("read token file %q: %w", tokenFile, err)
		}
		token := strings.TrimSpace(string(raw))
		if token == "" {
			return "", fmt.Errorf("token file %q is empty after whitespace trim", tokenFile)
		}
		return token, nil
	}
	return tokenInline, nil
}

// resolveTierTokens reads per-tier token files and returns two maps
// keyed by tier: the bearer tokens themselves and (optionally) the
// operator-supplied token identifiers per RFC architecture-http-auth-
// scope-model item 4(b). Tiers whose file path is empty are omitted
// (operator opt-out per RFC item 1).
//
// The token-file shape is line-oriented:
//
//	<token>
//	<token_id>    OPTIONAL — RFC item 4(b)
//
// Line 1 is the bearer token (whitespace-trimmed); rejected when
// empty per the §0035 resolveAuthToken contract. Line 2, when
// present + non-empty after trim, surfaces as the matched IngestionEvent's
// token_id field + the middle precedence rung of T4 per-actor-attribution
// resolution. Legacy single-line files (no line 2) are preserved per
// RFC item 4(c) "discouraged-but-permitted" fallback to the
// `unattributed-token-<tier>` literal. Lines beyond the second are
// ignored (reserved for future per-tier annotations).
//
// The returned `tokens` map is empty when no per-tier file is
// configured (the operator is using the legacy single-token mode or
// no auth at all); httpapi.New rejects the combination of per-tier +
// single-token at construction. The returned `tokenIDs` map carries
// entries ONLY for tiers whose file has a non-empty second line —
// always a subset of `tokens`'s key set.
//
// Per decision-log §0098 + RFC item 1; line 2 extension per RFC item
// 4(b).
func resolveTierTokens(tierFiles map[httpapi.Tier]string) (map[httpapi.Tier]string, map[httpapi.Tier]string, error) {
	tokens := make(map[httpapi.Tier]string)
	tokenIDs := make(map[httpapi.Tier]string)
	for tier, file := range tierFiles {
		if file == "" {
			continue
		}
		raw, err := os.ReadFile(file)
		if err != nil {
			return nil, nil, fmt.Errorf("read %s token file %q: %w", tier, file, err)
		}
		lines := strings.Split(string(raw), "\n")
		if len(lines) == 0 {
			return nil, nil, fmt.Errorf("%s token file %q is empty after whitespace trim", tier, file)
		}
		token := strings.TrimSpace(lines[0])
		if token == "" {
			return nil, nil, fmt.Errorf("%s token file %q is empty after whitespace trim", tier, file)
		}
		tokens[tier] = token
		if len(lines) >= 2 {
			if id := strings.TrimSpace(lines[1]); id != "" {
				tokenIDs[tier] = id
			}
		}
	}
	return tokens, tokenIDs, nil
}

// fatalReporter implements httpapi.FatalReporter. The HTTP handler calls
// ReportFatal on unrecoverable errors; the service's fatal coordinator
// goroutine reads from signal() and returns the error, propagating
// shutdown through errgroup per concurrency-pattern §Error Propagation.
type fatalReporter struct {
	once sync.Once
	ch   chan error
}

func (f *fatalReporter) signal() <-chan error {
	f.once.Do(func() { f.ch = make(chan error, 1) })
	return f.ch
}

func (f *fatalReporter) ReportFatal(err error) {
	f.once.Do(func() { f.ch = make(chan error, 1) })
	select {
	case f.ch <- err:
	default: // already signaled; first fatal wins
	}
}

// appendFunc is the readLoop's dependency on the ingestion pipeline.
// Implemented in production by ingest.Ingester.Append; injectable in
// tests for unrecoverable-error path coverage.
type appendFunc func(ctx context.Context, msg proto.Message, eventTime int64, env ingest.Envelope) (ingest.AppendReport, error)

// confirmation is the structured per-message outcome written to stdout
// on successful ingest.
type confirmation struct {
	EventHash          string `json:"event_hash"`
	IngestionEventHash string `json:"ingestion_event_hash"`
	PayloadBytes       int    `json:"payload_bytes"`
	CommittedAt        int64  `json:"committed_at_ns"`
}

// ingestError is the structured per-message error written to stdout
// on recoverable failure (bad input). Recoverable errors do not
// terminate the service.
type ingestError struct {
	Error string `json:"error"`
}

// fatalLog is the structured shutdown record written to stderr when an
// unrecoverable error terminates the service per concurrency-pattern
// §Error Propagation step 3.
type fatalLog struct {
	Level string `json:"level"`
	Error string `json:"error"`
	Note  string `json:"note"`
}

// isUnrecoverable classifies an error as unrecoverable per the
// substrate's typed §2.1-violation errors. Unrecoverable errors trigger
// service-level shutdown per concurrency-pattern §Error Propagation
// ("the §2.1-violation case (hash mismatch on read per §0027 AP4 + AP5)
// is the canonical example: the read path detects the violation,
// propagates the error up the call chain, and the service exits").
//
// Extension policy: a new error joins this set when (a) it indicates a
// substrate-integrity violation that cannot be reasoned about as
// recoverable, or (b) it indicates a canonical-serialization-contract
// violation per §0024 AP5 (hash-instability). Other failure modes
// (transient I/O, malformed input, schema validation) remain recoverable.
func isUnrecoverable(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, substrate.ErrHashMismatch) ||
		errors.Is(err, substrate.ErrBlobCollision)
}

// emitFatal writes a structured shutdown record to w. Used by main()
// before os.Exit(exitCodeUnrecoverable).
func emitFatal(w io.Writer, err error) {
	_ = json.NewEncoder(w).Encode(fatalLog{
		Level: "fatal",
		Error: err.Error(),
		Note:  "service exiting non-zero per docs/architecture/concurrency-pattern.md §Error Propagation",
	})
}

// readLoop reads JSON-envelope lines from r, ingests each primary
// observation via the appendFunc, writes a one-line JSON confirmation
// (success) or one-line JSON error (recoverable failure) per input line
// to w.
//
// Wire shape: each input line is a JSON object
// `{"type":"<type>","payload_b64":"<base64-proto>"}` where <type> is a
// registered Cat I message type per ingest.Registry (e.g.
// "declared_session", "network_event"). The base64 payload is the
// canonical-Protobuf encoding of the corresponding message.
//
// Error classification per concurrency-pattern §Error Propagation:
//   - **Recoverable** — bad input (envelope decode failure, unknown
//     type, base64 decode failure, proto unmarshal failure) — emits a
//     JSON ingestError entry to w and continues processing the next
//     line.
//   - **Unrecoverable** — substrate §2.1-violation errors
//     (substrate.ErrHashMismatch, substrate.ErrBlobCollision) —
//     terminates readLoop with the error; errgroup propagates the
//     cancellation, run() returns the error, main() writes a fatal
//     structured-output record to stderr and exits non-zero.
//
// The classifier is isUnrecoverable; the boundary is documented + tested
// (main_test.go).
func readLoop(ctx context.Context, doAppend appendFunc, r io.Reader, w io.Writer) error {
	enc := json.NewEncoder(w)
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024) // up to 1 MiB per line

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var env stdinEnvelope
		if err := json.Unmarshal(line, &env); err != nil {
			_ = enc.Encode(ingestError{Error: fmt.Sprintf("envelope decode: %v", err)})
			continue
		}
		desc, ok := ingest.LookupStdinType(env.Type)
		if !ok {
			_ = enc.Encode(ingestError{Error: fmt.Sprintf("unknown type %q; known types: %s", env.Type, strings.Join(ingest.KnownStdinTypes(), ", "))})
			continue
		}

		payload, err := base64.StdEncoding.DecodeString(env.PayloadB64)
		if err != nil {
			_ = enc.Encode(ingestError{Error: fmt.Sprintf("base64 decode: %v", err)})
			continue
		}

		msg := desc.New()
		if err := proto.Unmarshal(payload, msg); err != nil {
			_ = enc.Encode(ingestError{Error: fmt.Sprintf("proto unmarshal: %v", err)})
			continue
		}

		ingestEnv := ingest.Envelope{Channel: "stdin", ReceivedAt: time.Now().UnixNano()}
		rep, err := doAppend(ctx, msg, desc.EventTime(msg), ingestEnv)
		if err != nil {
			if isUnrecoverable(err) {
				return fmt.Errorf("unrecoverable ingest: %w", err)
			}
			_ = enc.Encode(ingestError{Error: fmt.Sprintf("ingest: %v", err)})
			continue
		}

		if err := enc.Encode(confirmation{
			EventHash:          rep.EventHashHex,
			IngestionEventHash: rep.IngestionEventHashHex,
			PayloadBytes:       rep.PayloadBytes,
			CommittedAt:        time.Now().UnixNano(),
		}); err != nil {
			return fmt.Errorf("write confirmation: %w", err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("stdin scan: %w", err)
	}
	return nil
}
