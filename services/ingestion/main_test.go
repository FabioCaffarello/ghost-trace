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

// encodeLine returns a stdin-envelope JSON line for msg, suitable for
// piping into readLoop's reader. typeTag MUST match a registered
// ingest.MessageDescriptor.StdinType.
func encodeLine(t *testing.T, typeTag string, msg proto.Message) string {
	t.Helper()
	bin, err := proto.Marshal(msg)
	if err != nil {
		t.Fatalf("proto.Marshal: %v", err)
	}
	env := stdinEnvelope{Type: typeTag, PayloadB64: base64.StdEncoding.EncodeToString(bin)}
	out, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("json.Marshal envelope: %v", err)
	}
	return string(out)
}

// encodeDeclaredSession is a convenience wrapper for the most common
// happy-path test (declared-session messages).
func encodeDeclaredSession(t *testing.T, msg *eventsv1.DeclaredSession) string {
	t.Helper()
	return encodeLine(t, "declared_session", msg)
}

// stubAppendFunc returns an appendFunc that records every call into
// calls and returns the configured per-call outcome (sequence index).
func stubAppendFunc(outcomes []error, calls *int) appendFunc {
	return func(ctx context.Context, msg proto.Message, eventTime int64, env ingest.Envelope) (ingest.AppendReport, error) {
		idx := *calls
		*calls = idx + 1
		if idx < len(outcomes) && outcomes[idx] != nil {
			return ingest.AppendReport{}, outcomes[idx]
		}
		return ingest.AppendReport{
			EventHashHex:          "0000000000000000000000000000000000000000000000000000000000000000",
			IngestionEventHashHex: "1111111111111111111111111111111111111111111111111111111111111111",
			PayloadBytes:          1,
		}, nil
	}
}

func TestReadLoopHappyPath(t *testing.T) {
	ctx := context.Background()
	msg := &eventsv1.DeclaredSession{DeclaredAt: 1, ActorRef: "happy", SessionDescriptor: []byte("x")}

	input := strings.NewReader(encodeDeclaredSession(t, msg) + "\n")
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
	//   1. Malformed envelope JSON → recoverable; emits ingestError, continues.
	//   2. Valid envelope but invalid Protobuf payload → recoverable; emits ingestError, continues.
	//   3. Valid message → succeeds.
	badProtoEnv := stdinEnvelope{Type: "declared_session", PayloadB64: base64.StdEncoding.EncodeToString([]byte("not-a-protobuf"))}
	badProtoLine, err := json.Marshal(badProtoEnv)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	input := strings.NewReader("!!!not-json!!!\n" + string(badProtoLine) + "\n" + encodeDeclaredSession(t, msg) + "\n")
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
	input := strings.NewReader(encodeDeclaredSession(t, msg) + "\n" + encodeDeclaredSession(t, msg) + "\n")
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

	input := strings.NewReader(encodeDeclaredSession(t, msg) + "\n")
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

// TestReadLoopDispatchesNetworkEvent proves stdin routes the second
// Cat I type through the dispatch registry: the envelope's `type` tag
// "network_event" selects ingest.MessageDescriptor's NetworkEvent
// factory, the payload unmarshals into a *NetworkEvent, and the
// observed_at field reaches the appendFunc as the event-time.
func TestReadLoopDispatchesNetworkEvent(t *testing.T) {
	ctx := context.Background()
	netEvt := &eventsv1.NetworkEvent{
		ObservedAt:      1716120000000000777,
		ActorRef:        "actor-stdin-network",
		EndpointRef:     "192.0.2.10:8080",
		EventDescriptor: []byte("flow"),
	}
	line := encodeLine(t, "network_event", netEvt)
	input := strings.NewReader(line + "\n")

	var capturedType string
	var capturedEventTime int64
	doAppend := func(ctx context.Context, msg proto.Message, eventTime int64, env ingest.Envelope) (ingest.AppendReport, error) {
		capturedType = string(msg.ProtoReflect().Descriptor().FullName())
		capturedEventTime = eventTime
		return ingest.AppendReport{
			EventHashHex:          "00000000000000000000000000000000000000000000000000000000000000aa",
			IngestionEventHashHex: "00000000000000000000000000000000000000000000000000000000000000bb",
			PayloadBytes:          16,
		}, nil
	}

	var output bytes.Buffer
	if err := readLoop(ctx, doAppend, input, &output); err != nil {
		t.Fatalf("readLoop: %v", err)
	}
	if capturedType != "ghosttrace.events.v1.NetworkEvent" {
		t.Errorf("dispatched type: got %q, want ghosttrace.events.v1.NetworkEvent", capturedType)
	}
	if capturedEventTime != netEvt.ObservedAt {
		t.Errorf("event time: got %d, want %d (NetworkEvent.observed_at)", capturedEventTime, netEvt.ObservedAt)
	}
	if !strings.Contains(output.String(), `"event_hash"`) {
		t.Errorf("expected confirmation JSON in output, got: %s", output.String())
	}
}

// TestReadLoopUnknownTypeIsRecoverable proves an envelope with an
// unregistered type tag does not terminate the loop: it emits a
// structured ingestError and continues.
func TestReadLoopUnknownTypeIsRecoverable(t *testing.T) {
	ctx := context.Background()
	unknownEnv := stdinEnvelope{Type: "fingerprint_snapshot", PayloadB64: base64.StdEncoding.EncodeToString([]byte("anything"))}
	unknownLine, err := json.Marshal(unknownEnv)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	goodMsg := &eventsv1.DeclaredSession{DeclaredAt: 1, ActorRef: "after-unknown", SessionDescriptor: []byte("k")}
	input := strings.NewReader(string(unknownLine) + "\n" + encodeDeclaredSession(t, goodMsg) + "\n")

	calls := 0
	var output bytes.Buffer
	if err := readLoop(ctx, stubAppendFunc(nil, &calls), input, &output); err != nil {
		t.Fatalf("readLoop returned error on unknown-type input: %v", err)
	}
	if calls != 1 {
		t.Errorf("Append calls: got %d, want 1 (only the second line should reach Append)", calls)
	}
	lines := strings.Split(strings.TrimRight(output.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("output line count: got %d, want 2 — %q", len(lines), output.String())
	}
	var firstErr ingestError
	if err := json.Unmarshal([]byte(lines[0]), &firstErr); err != nil {
		t.Fatalf("line 0 not parseable as ingestError: %v", err)
	}
	if !strings.Contains(firstErr.Error, "fingerprint_snapshot") {
		t.Errorf("error message should echo the unknown type: %q", firstErr.Error)
	}
	if !strings.Contains(firstErr.Error, "declared_session") || !strings.Contains(firstErr.Error, "network_event") {
		t.Errorf("error message should enumerate known types: %q", firstErr.Error)
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

// TestResolveLogger covers the §0201 production-wiring helper that
// constructs the structured-request logger from the --http-log-format
// value.
func TestResolveLogger(t *testing.T) {
	t.Run("none returns nil logger (no-op-default discipline)", func(t *testing.T) {
		got, err := resolveLogger("none", os.Stderr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Errorf("got non-nil logger, want nil (none should preserve §0197 MO1 no-op-default)")
		}
	})

	t.Run("empty string equivalent to none", func(t *testing.T) {
		got, err := resolveLogger("", os.Stderr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Errorf("got non-nil logger for empty format, want nil")
		}
	})

	t.Run("text format returns text-handler logger", func(t *testing.T) {
		var buf bytes.Buffer
		got, err := resolveLogger("text", &buf)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil {
			t.Fatal("got nil logger, want non-nil text-handler logger")
		}
		got.Info("witness", "key", "value")
		out := buf.String()
		if !strings.Contains(out, "witness") || !strings.Contains(out, "key=value") {
			t.Errorf("text-handler output missing expected content: %q", out)
		}
	})

	t.Run("json format returns json-handler logger", func(t *testing.T) {
		var buf bytes.Buffer
		got, err := resolveLogger("json", &buf)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil {
			t.Fatal("got nil logger, want non-nil json-handler logger")
		}
		got.Info("witness", "key", "value")
		out := buf.String()
		if !strings.Contains(out, `"msg":"witness"`) || !strings.Contains(out, `"key":"value"`) {
			t.Errorf("json-handler output missing expected content: %q", out)
		}
	})

	t.Run("unknown format errors", func(t *testing.T) {
		got, err := resolveLogger("nonsense", os.Stderr)
		if err == nil {
			t.Fatal("expected error for unknown format, got nil")
		}
		if got != nil {
			t.Errorf("expected nil logger on error, got non-nil")
		}
		if !strings.Contains(err.Error(), "valid: none, text, json") {
			t.Errorf("error message should list valid values: %v", err)
		}
	})
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

func TestResolveTierTokens(t *testing.T) {
	// resolveTierTokens reads per-tier token files; empty paths are
	// skipped (operator opt-out); read errors + empty-after-trim
	// surface as errors. Per decision-log §0098.

	t.Run("all empty paths returns empty map", func(t *testing.T) {
		tokens, tokenIDs, err := resolveTierTokens(map[httpapi.Tier]string{
			httpapi.TierProducer:          "",
			httpapi.TierOperatorRead:      "",
			httpapi.TierSubstrateAdmin:    "",
			httpapi.TierConstitutionalAct: "",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(tokens) != 0 {
			t.Errorf("tokens: got %v, want empty map", tokens)
		}
		if len(tokenIDs) != 0 {
			t.Errorf("tokenIDs: got %v, want empty map", tokenIDs)
		}
	})

	t.Run("subset of tiers configured", func(t *testing.T) {
		dir := t.TempDir()
		prodFile := dir + "/prod"
		opFile := dir + "/op"
		if err := os.WriteFile(prodFile, []byte("prod-secret\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(opFile, []byte("op-secret\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		tokens, tokenIDs, err := resolveTierTokens(map[httpapi.Tier]string{
			httpapi.TierProducer:     prodFile,
			httpapi.TierOperatorRead: opFile,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tokens[httpapi.TierProducer] != "prod-secret" {
			t.Errorf("producer: got %q, want %q", tokens[httpapi.TierProducer], "prod-secret")
		}
		if tokens[httpapi.TierOperatorRead] != "op-secret" {
			t.Errorf("operator-read: got %q, want %q", tokens[httpapi.TierOperatorRead], "op-secret")
		}
		if _, ok := tokens[httpapi.TierSubstrateAdmin]; ok {
			t.Errorf("substrate-admin should not be present (file path empty)")
		}
		// Legacy single-line files contribute no token_ids.
		if len(tokenIDs) != 0 {
			t.Errorf("tokenIDs: got %v, want empty map (single-line files)", tokenIDs)
		}
	})

	t.Run("two-line files yield token_id per RFC item 4(b)", func(t *testing.T) {
		dir := t.TempDir()
		prodFile := dir + "/prod"
		opFile := dir + "/op"
		// Producer has two-line token file: line 1 token, line 2 token_id.
		if err := os.WriteFile(prodFile, []byte("prod-secret\nprod-token-alpha\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		// Operator-read is single-line — preserves §0035 backward compat.
		if err := os.WriteFile(opFile, []byte("op-secret\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		tokens, tokenIDs, err := resolveTierTokens(map[httpapi.Tier]string{
			httpapi.TierProducer:     prodFile,
			httpapi.TierOperatorRead: opFile,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tokens[httpapi.TierProducer] != "prod-secret" {
			t.Errorf("producer token: got %q, want %q", tokens[httpapi.TierProducer], "prod-secret")
		}
		if tokenIDs[httpapi.TierProducer] != "prod-token-alpha" {
			t.Errorf("producer token_id: got %q, want %q", tokenIDs[httpapi.TierProducer], "prod-token-alpha")
		}
		if _, ok := tokenIDs[httpapi.TierOperatorRead]; ok {
			t.Errorf("operator-read token_id should be absent (single-line file)")
		}
	})

	t.Run("whitespace-only second line yields no token_id", func(t *testing.T) {
		dir := t.TempDir()
		path := dir + "/tok"
		if err := os.WriteFile(path, []byte("the-token\n   \n"), 0o600); err != nil {
			t.Fatal(err)
		}
		tokens, tokenIDs, err := resolveTierTokens(map[httpapi.Tier]string{
			httpapi.TierProducer: path,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tokens[httpapi.TierProducer] != "the-token" {
			t.Errorf("token: got %q, want %q", tokens[httpapi.TierProducer], "the-token")
		}
		if _, ok := tokenIDs[httpapi.TierProducer]; ok {
			t.Errorf("token_id should be absent (whitespace-only second line)")
		}
	})

	t.Run("missing file returns error", func(t *testing.T) {
		_, _, err := resolveTierTokens(map[httpapi.Tier]string{
			httpapi.TierProducer: "/nonexistent/path/to/token",
		})
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
		_, _, err := resolveTierTokens(map[httpapi.Tier]string{
			httpapi.TierProducer: path,
		})
		if err == nil {
			t.Fatal("expected error for whitespace-only file, got nil")
		}
	})
}

func TestResolveHTTPTLS(t *testing.T) {
	t.Run("both empty returns disabled", func(t *testing.T) {
		cfg, err := resolveHTTPTLS("", "", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.enabled {
			t.Error("got enabled=true, want false")
		}
	})

	t.Run("cert without key returns error", func(t *testing.T) {
		_, err := resolveHTTPTLS("/some/cert.pem", "", "")
		if err == nil {
			t.Fatal("expected error for cert without key, got nil")
		}
	})

	t.Run("key without cert returns error", func(t *testing.T) {
		_, err := resolveHTTPTLS("", "/some/key.pem", "")
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
		_, err := resolveHTTPTLS("/nonexistent/cert.pem", keyPath, "")
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
		_, err := resolveHTTPTLS(certPath, "/nonexistent/key.pem", "")
		if err == nil {
			t.Fatal("expected error for missing key, got nil")
		}
	})

	t.Run("both present returns enabled", func(t *testing.T) {
		certPath, keyPath := writeEphemeralTLSCert(t)
		cfg, err := resolveHTTPTLS(certPath, keyPath, "")
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
		if cfg.clientCAPool != nil {
			t.Error("clientCAPool should be nil when --http-tls-client-ca not set")
		}
	})

	t.Run("clientCA without cert+key returns error", func(t *testing.T) {
		dir := t.TempDir()
		caPath := filepath.Join(dir, "ca.pem")
		if err := os.WriteFile(caPath, []byte("dummy"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := resolveHTTPTLS("", "", caPath)
		if err == nil {
			t.Fatal("expected error for clientCA without cert+key, got nil")
		}
	})

	t.Run("clientCA + cert + key returns mTLS-enabled", func(t *testing.T) {
		pki := generateTestPKI(t)
		serverCertPath, serverKeyPath := pki.signServerCert(t)
		cfg, err := resolveHTTPTLS(serverCertPath, serverKeyPath, pki.caCertPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !cfg.enabled {
			t.Error("got enabled=false, want true")
		}
		if cfg.clientCAPool == nil {
			t.Error("clientCAPool should be non-nil when --http-tls-client-ca is set")
		}
	})

	t.Run("clientCA file unparseable returns error", func(t *testing.T) {
		dir := t.TempDir()
		caPath := filepath.Join(dir, "ca.pem")
		if err := os.WriteFile(caPath, []byte("not a PEM block"), 0o644); err != nil {
			t.Fatal(err)
		}
		certPath, keyPath := writeEphemeralTLSCert(t)
		_, err := resolveHTTPTLS(certPath, keyPath, caPath)
		if err == nil {
			t.Fatal("expected error for unparseable clientCA file, got nil")
		}
	})

	t.Run("clientCA missing file returns error", func(t *testing.T) {
		certPath, keyPath := writeEphemeralTLSCert(t)
		_, err := resolveHTTPTLS(certPath, keyPath, "/nonexistent/ca.pem")
		if err == nil {
			t.Fatal("expected error for missing clientCA file, got nil")
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
	cfg, err := resolveHTTPTLS(certPath, keyPath, "")
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
	handler := httpapi.MustNew(in.Append, nil)

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

	// POST /v1/events/declared-session round-trip with a valid Protobuf payload.
	msg := &eventsv1.DeclaredSession{
		DeclaredAt:        1716120000000000000,
		ActorRef:          "actor-tls-test",
		SessionDescriptor: []byte("session-bytes"),
	}
	payload, err := proto.Marshal(msg)
	if err != nil {
		t.Fatalf("proto.Marshal: %v", err)
	}
	postResp, err := client.Post(url+"/v1/events/declared-session", "application/x-protobuf", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("POST /v1/events/declared-session over TLS: %v", err)
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

func TestHTTPmTLSEndToEnd(t *testing.T) {
	// Integration test: full PKI dance.
	//   1. Generate a CA.
	//   2. Sign a server cert with the CA.
	//   3. Sign two client certs with the CA: one trusted (for the
	//      success case), one self-signed by a DIFFERENT CA (for the
	//      rejection case).
	//   4. Spin up the server with --http-tls-cert (server cert) +
	//      --http-tls-key (server key) + --http-tls-client-ca (CA cert).
	//   5. Verify: trusted client succeeds; untrusted client is rejected
	//      at the TLS layer; no-client-cert is rejected at the TLS layer.

	pki := generateTestPKI(t)
	serverCertPath, serverKeyPath := pki.signServerCert(t)
	trustedClientCert, trustedClientKey := pki.signClientCert(t, "trusted-producer")

	// Untrusted client: signed by a different CA.
	otherPKI := generateTestPKI(t)
	untrustedClientCert, untrustedClientKey := otherPKI.signClientCert(t, "untrusted-producer")

	cfg, err := resolveHTTPTLS(serverCertPath, serverKeyPath, pki.caCertPath)
	if err != nil {
		t.Fatalf("resolveHTTPTLS: %v", err)
	}
	if !cfg.enabled || cfg.clientCAPool == nil {
		t.Fatal("mTLS not enabled")
	}

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
	handler := httpapi.MustNew(in.Append, nil)

	srv := &http.Server{
		Handler: handler,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			ClientAuth: tls.RequireAndVerifyClientCert,
			ClientCAs:  cfg.clientCAPool,
		},
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

	addr := listener.Addr().(*net.TCPAddr)
	url := fmt.Sprintf("https://localhost:%d", addr.Port)

	// Trust pool for server cert verification (clients verify the server's CA).
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(pki.caCertPEM) {
		t.Fatal("AppendCertsFromPEM: failed to parse server CA")
	}

	t.Run("trusted client succeeds", func(t *testing.T) {
		clientCertPair, err := tls.LoadX509KeyPair(trustedClientCert, trustedClientKey)
		if err != nil {
			t.Fatalf("LoadX509KeyPair (trusted client): %v", err)
		}
		client := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs:      roots,
					Certificates: []tls.Certificate{clientCertPair},
					MinVersion:   tls.VersionTLS12,
				},
			},
			Timeout: 5 * time.Second,
		}
		resp, err := client.Get(url + "/healthz")
		if err != nil {
			t.Fatalf("GET /healthz with trusted client cert: %v", err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("status: got %d, want 200", resp.StatusCode)
		}

		// Verify the server saw the client cert by checking the PeerCertificates
		// indirectly via the response.TLS state.
		if resp.TLS == nil {
			t.Fatal("response missing TLS state")
		}
	})

	t.Run("untrusted client cert rejected at TLS handshake", func(t *testing.T) {
		clientCertPair, err := tls.LoadX509KeyPair(untrustedClientCert, untrustedClientKey)
		if err != nil {
			t.Fatalf("LoadX509KeyPair (untrusted client): %v", err)
		}
		client := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs:      roots,
					Certificates: []tls.Certificate{clientCertPair},
					MinVersion:   tls.VersionTLS12,
				},
			},
			Timeout: 5 * time.Second,
		}
		resp, err := client.Get(url + "/healthz")
		if err == nil {
			_ = resp.Body.Close()
			t.Fatalf("untrusted client cert: expected TLS handshake failure, got HTTP %d", resp.StatusCode)
		}
		// Acceptable error patterns: "tls: ..." or "bad certificate" or
		// "unknown certificate authority". We assert "tls" appears in the
		// error string for a baseline check.
		if !strings.Contains(err.Error(), "tls") && !strings.Contains(err.Error(), "certificate") {
			t.Errorf("expected TLS / certificate error, got: %v", err)
		}
	})

	t.Run("no client cert rejected at TLS handshake", func(t *testing.T) {
		client := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs:    roots,
					MinVersion: tls.VersionTLS12,
				},
			},
			Timeout: 5 * time.Second,
		}
		resp, err := client.Get(url + "/healthz")
		if err == nil {
			_ = resp.Body.Close()
			t.Fatalf("no client cert: expected TLS handshake failure, got HTTP %d", resp.StatusCode)
		}
	})
}

// testPKI holds an ephemeral certificate authority for mTLS testing.
type testPKI struct {
	caCertPath string
	caKeyPath  string
	caCertPEM  []byte // raw PEM bytes (handy for AppendCertsFromPEM in tests)
	caCert     *x509.Certificate
	caKey      *ecdsa.PrivateKey
}

// generateTestPKI creates a self-signed CA in a per-test temp directory
// and returns paths + parsed structures sufficient to sign server and
// client certs.
func generateTestPKI(t *testing.T) *testPKI {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("ecdsa.GenerateKey (ca): %v", err)
	}
	template := x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().UnixNano()),
		Subject:               pkix.Name{CommonName: "ingestion-test-ca"},
		NotBefore:             time.Now().Add(-time.Minute),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("CreateCertificate (ca): %v", err)
	}
	cert, err := x509.ParseCertificate(derBytes)
	if err != nil {
		t.Fatalf("ParseCertificate (ca): %v", err)
	}

	dir := t.TempDir()
	caCertPath := filepath.Join(dir, "ca-cert.pem")
	caKeyPath := filepath.Join(dir, "ca-key.pem")

	var certPEM bytes.Buffer
	if err := pem.Encode(&certPEM, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		t.Fatalf("pem encode ca cert: %v", err)
	}
	if err := os.WriteFile(caCertPath, certPEM.Bytes(), 0o644); err != nil {
		t.Fatalf("write ca cert: %v", err)
	}

	keyBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		t.Fatalf("MarshalECPrivateKey (ca): %v", err)
	}
	var keyPEM bytes.Buffer
	if err := pem.Encode(&keyPEM, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes}); err != nil {
		t.Fatalf("pem encode ca key: %v", err)
	}
	if err := os.WriteFile(caKeyPath, keyPEM.Bytes(), 0o600); err != nil {
		t.Fatalf("write ca key: %v", err)
	}

	return &testPKI{
		caCertPath: caCertPath,
		caKeyPath:  caKeyPath,
		caCertPEM:  certPEM.Bytes(),
		caCert:     cert,
		caKey:      priv,
	}
}

// signServerCert issues a leaf cert with ServerAuth EKU + localhost SANs,
// signed by the testPKI's CA. Returns paths to the PEM-encoded cert and
// private key.
func (p *testPKI) signServerCert(t *testing.T) (certPath, keyPath string) {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("ecdsa.GenerateKey (server): %v", err)
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: "ingestion-test-server"},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, p.caCert, &priv.PublicKey, p.caKey)
	if err != nil {
		t.Fatalf("CreateCertificate (server): %v", err)
	}
	return writeCertKey(t, derBytes, priv, "server")
}

// signClientCert issues a leaf cert with ClientAuth EKU + the given
// CommonName, signed by the testPKI's CA. Returns paths to the PEM-
// encoded cert and private key.
func (p *testPKI) signClientCert(t *testing.T, commonName string) (certPath, keyPath string) {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("ecdsa.GenerateKey (client): %v", err)
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: commonName},
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, p.caCert, &priv.PublicKey, p.caKey)
	if err != nil {
		t.Fatalf("CreateCertificate (client): %v", err)
	}
	return writeCertKey(t, derBytes, priv, commonName)
}

// writeCertKey writes a (cert DER, ECDSA priv key) pair as PEM files in
// a per-test temp directory and returns the paths.
func writeCertKey(t *testing.T, derBytes []byte, priv *ecdsa.PrivateKey, nameHint string) (certPath, keyPath string) {
	t.Helper()
	dir := t.TempDir()
	certPath = filepath.Join(dir, nameHint+"-cert.pem")
	keyPath = filepath.Join(dir, nameHint+"-key.pem")

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
