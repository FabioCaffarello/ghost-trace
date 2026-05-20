// Command ingestion is the inception-phase ingestion service per
// decision-log §0022 (implementation pivot) + §0024 (Protobuf proto3
// schemas) + §0025 (Go) + §0027 (SQLite + content-addressed blob-store).
//
// Architecture: docs/architecture/canonical-serialization-contract.md +
// docs/architecture/concurrency-pattern.md.
//
// Inception-phase I/O: reads newline-delimited base64-encoded Protobuf
// DeclaredSession messages from stdin; writes one-line JSON confirmation
// per ingested message to stdout. HTTP/gRPC interfaces deferred to a
// follow-on RFC (per decision-log §0025 Open Questions).
package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

func main() {
	if err := run(); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return // graceful shutdown via signal
		}
		fmt.Fprintf(os.Stderr, "ingestion: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	dbPath := flag.String("db", "./ghost-trace.db", "SQLite primary-event-log path")
	blobDir := flag.String("blobs", "./blobs", "content-addressed blob-store directory")
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

	// Single worker goroutine consuming stdin; structured-concurrency via
	// errgroup per concurrency-pattern §Goroutine Lifecycle.
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error { return readLoop(gctx, in, os.Stdin, os.Stdout) })

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

// confirmation is the structured per-message outcome written to stdout.
type confirmation struct {
	EventHash    string `json:"event_hash"`
	PayloadBytes int    `json:"payload_bytes"`
	CommittedAt  int64  `json:"committed_at_ns"`
}

// ingestError is the structured per-message error written to stdout.
type ingestError struct {
	Error string `json:"error"`
}

// readLoop reads base64-encoded Protobuf DeclaredSession lines from r,
// ingests each through in.Append, writes a one-line JSON confirmation
// (success) or one-line JSON error (failure) per input line to w.
//
// Per-message errors are recoverable: a failed line emits an error
// object and processing continues with the next line. Substrate-level
// or canonical-serialization-level errors that indicate §2.1 violation
// (e.g. ErrHashMismatch on read) are unrecoverable and propagated up
// per concurrency-pattern §Error Propagation; this loop currently
// surfaces them as per-message errors — the §2.1-violation-shutdown
// pathway lands when the read path is exercised end-to-end (follow-on
// commit will add the explicit unrecoverable-error escalation).
func readLoop(ctx context.Context, in *ingest.Ingester, r io.Reader, w io.Writer) error {
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

		payload, err := base64.StdEncoding.DecodeString(string(line))
		if err != nil {
			_ = enc.Encode(ingestError{Error: fmt.Sprintf("base64 decode: %v", err)})
			continue
		}

		msg := &eventsv1.DeclaredSession{}
		if err := proto.Unmarshal(payload, msg); err != nil {
			_ = enc.Encode(ingestError{Error: fmt.Sprintf("proto unmarshal: %v", err)})
			continue
		}

		rep, err := in.Append(ctx, msg, msg.DeclaredAt)
		if err != nil {
			_ = enc.Encode(ingestError{Error: fmt.Sprintf("ingest: %v", err)})
			continue
		}

		if err := enc.Encode(confirmation{
			EventHash:    rep.EventHashHex,
			PayloadBytes: rep.PayloadBytes,
			CommittedAt:  time.Now().UnixNano(),
		}); err != nil {
			return fmt.Errorf("write confirmation: %w", err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("stdin scan: %w", err)
	}
	return nil
}
