package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/httpapi"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

func TestIsUnrecoverable(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"recoverable plain", errors.New("transient i/o"), false},
		{"recoverable wrapped", fmt.Errorf("wrapping: %w", errors.New("decode failed")), false},
		{"unrecoverable ErrHashMismatch", substrate.ErrHashMismatch, true},
		{"unrecoverable wrapped ErrHashMismatch", fmt.Errorf("substrate.ReadBlob at /x/y: %w", substrate.ErrHashMismatch), true},
		{"unrecoverable ErrBlobCollision", substrate.ErrBlobCollision, true},
		{"unrecoverable wrapped ErrBlobCollision", fmt.Errorf("at /x/y: %w", substrate.ErrBlobCollision), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isUnrecoverable(tc.err); got != tc.want {
				t.Fatalf("isUnrecoverable(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

// encodeLine returns a base64-encoded canonical Protobuf line for msg,
// suitable for piping into readLoop's reader.
func encodeLine(t *testing.T, msg proto.Message) string {
	t.Helper()
	bin, err := proto.Marshal(msg)
	if err != nil {
		t.Fatalf("proto.Marshal: %v", err)
	}
	return base64.StdEncoding.EncodeToString(bin)
}

// stubAppendFunc returns an appendFunc that records every call into
// calls and returns the configured per-call outcome (sequence index).
func stubAppendFunc(outcomes []error, calls *int) appendFunc {
	return func(ctx context.Context, msg proto.Message, eventTime int64) (ingest.AppendReport, error) {
		idx := *calls
		*calls = idx + 1
		if idx < len(outcomes) && outcomes[idx] != nil {
			return ingest.AppendReport{}, outcomes[idx]
		}
		return ingest.AppendReport{EventHashHex: "0000000000000000000000000000000000000000000000000000000000000000", PayloadBytes: 1}, nil
	}
}

func TestReadLoopHappyPath(t *testing.T) {
	ctx := context.Background()
	msg := &eventsv1.DeclaredSession{DeclaredAt: 1, ActorRef: "happy", SessionDescriptor: []byte("x")}

	input := strings.NewReader(encodeLine(t, msg) + "\n")
	var output bytes.Buffer

	calls := 0
	if err := readLoop(ctx, stubAppendFunc(nil, &calls), input, &output); err != nil {
		t.Fatalf("readLoop: %v", err)
	}

	if calls != 1 {
		t.Errorf("Append calls: got %d, want 1", calls)
	}
	if !strings.Contains(output.String(), `"event_hash"`) {
		t.Errorf("expected confirmation JSON in output, got: %s", output.String())
	}
}

func TestReadLoopRecoverableErrorContinues(t *testing.T) {
	ctx := context.Background()
	msg := &eventsv1.DeclaredSession{DeclaredAt: 1, ActorRef: "follow-up", SessionDescriptor: []byte("y")}

	// Three input lines:
	//   1. Invalid base64 → recoverable; emits ingestError, continues.
	//   2. Valid base64 but invalid Protobuf → recoverable; emits ingestError, continues.
	//   3. Valid message → succeeds.
	input := strings.NewReader("!!!not-base64!!!\n" + base64.StdEncoding.EncodeToString([]byte("not-a-protobuf")) + "\n" + encodeLine(t, msg) + "\n")
	var output bytes.Buffer

	calls := 0
	if err := readLoop(ctx, stubAppendFunc(nil, &calls), input, &output); err != nil {
		t.Fatalf("readLoop returned error on recoverable inputs: %v", err)
	}

	if calls != 1 {
		t.Errorf("Append calls: got %d, want 1 (only the third line should reach Append)", calls)
	}

	// Output should contain three lines: two errors + one confirmation.
	lines := strings.Split(strings.TrimRight(output.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("output line count: got %d, want 3 — %q", len(lines), output.String())
	}
	for i, line := range lines[:2] {
		var entry ingestError
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Errorf("line %d not parseable as ingestError: %v", i, err)
		}
		if entry.Error == "" {
			t.Errorf("line %d: empty error message", i)
		}
	}
	var conf confirmation
	if err := json.Unmarshal([]byte(lines[2]), &conf); err != nil {
		t.Fatalf("line 2 not parseable as confirmation: %v", err)
	}
	if conf.EventHash == "" {
		t.Error("confirmation missing event_hash")
	}
}

func TestReadLoopUnrecoverableErrorTerminates(t *testing.T) {
	ctx := context.Background()
	msg := &eventsv1.DeclaredSession{DeclaredAt: 1, ActorRef: "doomed", SessionDescriptor: []byte("z")}

	// Two input lines:
	//   1. First Append returns substrate.ErrBlobCollision (unrecoverable).
	//   2. Second line never reached — loop terminates on line 1's unrecoverable.
	input := strings.NewReader(encodeLine(t, msg) + "\n" + encodeLine(t, msg) + "\n")
	var output bytes.Buffer

	calls := 0
	err := readLoop(ctx, stubAppendFunc([]error{
		fmt.Errorf("at /blobs/de/ad: %w", substrate.ErrBlobCollision),
	}, &calls), input, &output)

	if err == nil {
		t.Fatal("readLoop returned nil; expected unrecoverable error to terminate")
	}
	if !isUnrecoverable(err) {
		t.Fatalf("returned error not classified unrecoverable: %v", err)
	}
	if !errors.Is(err, substrate.ErrBlobCollision) {
		t.Fatalf("expected errors.Is(err, substrate.ErrBlobCollision); got: %v", err)
	}
	if calls != 1 {
		t.Errorf("Append calls: got %d, want 1 (loop should terminate after first unrecoverable)", calls)
	}
	// Output should contain zero entries on stdout — unrecoverable does
	// not write a per-message JSON entry; the structured-shutdown record
	// goes to stderr at main() level via emitFatal.
	if strings.TrimSpace(output.String()) != "" {
		t.Errorf("expected empty stdout on unrecoverable path, got: %q", output.String())
	}
}

func TestReadLoopHashMismatchAlsoUnrecoverable(t *testing.T) {
	// Symmetry with the ErrBlobCollision test; documents that
	// ErrHashMismatch also triggers shutdown.
	ctx := context.Background()
	msg := &eventsv1.DeclaredSession{DeclaredAt: 1, ActorRef: "doomed-2", SessionDescriptor: []byte("z")}

	input := strings.NewReader(encodeLine(t, msg) + "\n")
	var output bytes.Buffer

	calls := 0
	err := readLoop(ctx, stubAppendFunc([]error{
		fmt.Errorf("substrate.ReadBlob at /blobs/be/ef: %w", substrate.ErrHashMismatch),
	}, &calls), input, &output)
	if err == nil {
		t.Fatal("readLoop returned nil; expected unrecoverable error")
	}
	if !errors.Is(err, substrate.ErrHashMismatch) {
		t.Fatalf("expected errors.Is(err, substrate.ErrHashMismatch); got: %v", err)
	}
}

func TestEmitFatalStructure(t *testing.T) {
	var buf bytes.Buffer
	emitFatal(&buf, fmt.Errorf("test: %w", substrate.ErrHashMismatch))

	var got fatalLog
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &got); err != nil {
		t.Fatalf("fatalLog not parseable as JSON: %v — output: %q", err, buf.String())
	}
	if got.Level != "fatal" {
		t.Errorf("Level: got %q, want %q", got.Level, "fatal")
	}
	if !strings.Contains(got.Error, substrate.ErrHashMismatch.Error()) {
		t.Errorf("Error field missing substrate.ErrHashMismatch text: %q", got.Error)
	}
	if !strings.Contains(got.Note, "concurrency-pattern.md") {
		t.Errorf("Note field missing concurrency-pattern.md reference: %q", got.Note)
	}
}

func TestResolveAuthToken(t *testing.T) {
	t.Run("both empty returns empty (auth disabled)", func(t *testing.T) {
		got, err := resolveAuthToken("", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})

	t.Run("inline token returns as-is", func(t *testing.T) {
		got, err := resolveAuthToken("inline-secret", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "inline-secret" {
			t.Errorf("got %q, want %q", got, "inline-secret")
		}
	})

	t.Run("file takes precedence over inline", func(t *testing.T) {
		dir := t.TempDir()
		path := dir + "/token"
		if err := os.WriteFile(path, []byte("file-secret\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		got, err := resolveAuthToken("inline-secret", path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "file-secret" {
			t.Errorf("got %q, want %q (file should win)", got, "file-secret")
		}
	})

	t.Run("file trims surrounding whitespace", func(t *testing.T) {
		dir := t.TempDir()
		path := dir + "/token"
		if err := os.WriteFile(path, []byte("  \t spaced-secret  \n\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		got, err := resolveAuthToken("", path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "spaced-secret" {
			t.Errorf("got %q, want %q", got, "spaced-secret")
		}
	})

	t.Run("missing file returns error", func(t *testing.T) {
		_, err := resolveAuthToken("", "/nonexistent/path/to/token")
		if err == nil {
			t.Fatal("expected error for missing file, got nil")
		}
	})

	t.Run("empty file returns error", func(t *testing.T) {
		dir := t.TempDir()
		path := dir + "/empty"
		if err := os.WriteFile(path, []byte("   \n\n  "), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := resolveAuthToken("inline-secret", path)
		if err == nil {
			t.Fatal("expected error for whitespace-only file, got nil")
		}
	})
}

func TestResolveHTTPTLS(t *testing.T) {
	t.Run("both empty returns disabled", func(t *testing.T) {
		cfg, err := resolveHTTPTLS("", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.enabled {
			t.Error("got enabled=true, want false")
		}
	})

	t.Run("cert without key returns error", func(t *testing.T) {
		_, err := resolveHTTPTLS("/some/cert.pem", "")
		if err == nil {
			t.Fatal("expected error for cert without key, got nil")
		}
	})

	t.Run("key without cert returns error", func(t *testing.T) {
		_, err := resolveHTTPTLS("", "/some/key.pem")
		if err == nil {
			t.Fatal("expected error for key without cert, got nil")
		}
	})

	t.Run("missing cert file returns error", func(t *testing.T) {
		dir := t.TempDir()
		keyPath := filepath.Join(dir, "key.pem")
		if err := os.WriteFile(keyPath, []byte("dummy"), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := resolveHTTPTLS("/nonexistent/cert.pem", keyPath)
		if err == nil {
			t.Fatal("expected error for missing cert, got nil")
		}
	})

	t.Run("missing key file returns error", func(t *testing.T) {
		dir := t.TempDir()
		certPath := filepath.Join(dir, "cert.pem")
		if err := os.WriteFile(certPath, []byte("dummy"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := resolveHTTPTLS(certPath, "/nonexistent/key.pem")
		if err == nil {
			t.Fatal("expected error for missing key, got nil")
		}
	})

	t.Run("both present returns enabled", func(t *testing.T) {
		certPath, keyPath := writeEphemeralTLSCert(t)
		cfg, err := resolveHTTPTLS(certPath, keyPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !cfg.enabled {
			t.Error("got enabled=false, want true")
		}
		if cfg.certFile != certPath || cfg.keyFile != keyPath {
			t.Errorf("paths not preserved: got cert=%q key=%q, want cert=%q key=%q",
				cfg.certFile, cfg.keyFile, certPath, keyPath)
		}
	})
}

func TestHTTPTLSEndToEnd(t *testing.T) {
	// Integration test: write ephemeral TLS cert + key files, construct
	// an http.Server with the same TLS config production uses, exercise
	// a real HTTPS round-trip through the httpapi handler against the
	// canonical+substrate stack.
	certPath, keyPath := writeEphemeralTLSCert(t)
	_, certPEM := readBack(t, certPath)
	cfg, err := resolveHTTPTLS(certPath, keyPath)
	if err != nil {
		t.Fatalf("resolveHTTPTLS: %v", err)
	}
	if !cfg.enabled {
		t.Fatal("TLS not enabled")
	}

	// Bind to an ephemeral port; capture the chosen port via Listener.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	defer func() { _ = sub.Close() }()

	in := ingest.New(sub, time.Now)
	handler := httpapi.New(in.Append, nil)

	srv := &http.Server{
		Handler:           handler,
		TLSConfig:         &tls.Config{MinVersion: tls.VersionTLS12},
		ReadHeaderTimeout: 10 * time.Second,
	}
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- srv.ServeTLS(listener, cfg.certFile, cfg.keyFile)
	}()
	t.Cleanup(func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		<-serveErr
	})

	// Build a TLS client that trusts the ephemeral cert.
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(certPEM) {
		t.Fatal("AppendCertsFromPEM: failed to parse cert")
	}
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:    roots,
				MinVersion: tls.VersionTLS12,
			},
		},
		Timeout: 5 * time.Second,
	}

	addr := listener.Addr().(*net.TCPAddr)
	url := fmt.Sprintf("https://localhost:%d", addr.Port)

	// Healthz round-trip.
	resp, err := client.Get(url + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz over TLS: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("healthz status: got %d, want 200 (body: %s)", resp.StatusCode, body)
	}

	// Verify TLS version actually negotiated ≥ 1.2.
	if resp.TLS == nil {
		t.Fatal("response missing TLS state — handshake did not happen?")
	}
	if resp.TLS.Version < tls.VersionTLS12 {
		t.Errorf("negotiated TLS version: got 0x%04x, want ≥ 0x%04x (TLS 1.2)", resp.TLS.Version, tls.VersionTLS12)
	}

	// POST /v1/events round-trip with a valid Protobuf payload.
	msg := &eventsv1.DeclaredSession{
		DeclaredAt:        1716120000000000000,
		ActorRef:          "actor-tls-test",
		SessionDescriptor: []byte("session-bytes"),
	}
	payload, err := proto.Marshal(msg)
	if err != nil {
		t.Fatalf("proto.Marshal: %v", err)
	}
	postResp, err := client.Post(url+"/v1/events", "application/x-protobuf", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("POST /v1/events over TLS: %v", err)
	}
	postBody, _ := io.ReadAll(postResp.Body)
	_ = postResp.Body.Close()
	if postResp.StatusCode != http.StatusOK {
		t.Errorf("POST status: got %d, want 200 (body: %s)", postResp.StatusCode, postBody)
	}
	if !strings.Contains(string(postBody), `"event_hash"`) {
		t.Errorf("POST response missing event_hash: %s", postBody)
	}
}

// writeEphemeralTLSCert generates a self-signed ECDSA cert valid for one
// hour with localhost + 127.0.0.1 SANs, writes the cert + key as PEM to
// a per-test temp directory, and returns the paths. Used by the TLS
// integration tests; not safe for production.
func writeEphemeralTLSCert(t *testing.T) (certPath, keyPath string) {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("ecdsa.GenerateKey: %v", err)
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "ingestion-test"},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("x509.CreateCertificate: %v", err)
	}
	dir := t.TempDir()
	certPath = filepath.Join(dir, "cert.pem")
	keyPath = filepath.Join(dir, "key.pem")

	certOut, err := os.Create(certPath)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		t.Fatalf("pem encode cert: %v", err)
	}
	if err := certOut.Close(); err != nil {
		t.Fatalf("close cert: %v", err)
	}

	keyBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		t.Fatalf("MarshalECPrivateKey: %v", err)
	}
	keyOut, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		t.Fatalf("create key: %v", err)
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes}); err != nil {
		t.Fatalf("pem encode key: %v", err)
	}
	if err := keyOut.Close(); err != nil {
		t.Fatalf("close key: %v", err)
	}
	return certPath, keyPath
}

func readBack(t *testing.T, path string) (string, []byte) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return path, b
}

// Sanity check: emitFatal accepts a nil-safe io.Writer of any kind.
var _ io.Writer = (*bytes.Buffer)(nil)
