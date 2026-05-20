package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
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

// Sanity check: emitFatal accepts a nil-safe io.Writer of any kind.
var _ io.Writer = (*bytes.Buffer)(nil)
