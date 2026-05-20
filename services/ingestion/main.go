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
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/httpapi"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

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
		handler := httpapi.New(in.Append, reporter)
		srv := &http.Server{
			Addr:              *httpAddr,
			Handler:           handler,
			ReadHeaderTimeout: 10 * time.Second,
		}
		g.Go(func() error {
			if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				return fmt.Errorf("http listen: %w", err)
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
type appendFunc func(ctx context.Context, msg proto.Message, eventTime int64) (ingest.AppendReport, error)

// confirmation is the structured per-message outcome written to stdout
// on successful ingest.
type confirmation struct {
	EventHash    string `json:"event_hash"`
	PayloadBytes int    `json:"payload_bytes"`
	CommittedAt  int64  `json:"committed_at_ns"`
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

// readLoop reads base64-encoded Protobuf DeclaredSession lines from r,
// ingests each via the appendFunc, writes a one-line JSON confirmation
// (success) or one-line JSON error (recoverable failure) per input line
// to w.
//
// Error classification per concurrency-pattern §Error Propagation:
//   - **Recoverable** — bad input (base64 decode failure, proto
//     unmarshal failure) — emits a JSON ingestError entry to w and
//     continues processing the next line.
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

		rep, err := doAppend(ctx, msg, msg.DeclaredAt)
		if err != nil {
			if isUnrecoverable(err) {
				return fmt.Errorf("unrecoverable ingest: %w", err)
			}
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
