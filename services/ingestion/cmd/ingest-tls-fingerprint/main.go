// Command ingest-tls-fingerprint is the operator-facing CLI that
// streams newline-delimited JSON TLS fingerprint records (JA3 + JA4)
// into the substrate as Category I NetworkObservation records, via the
// tls_fingerprint adapter library. Thin wrapper around
// tls_fingerprint.Ingest per decision-log §0221 (TLS fingerprint
// vertical slice); zero new semantics beyond the adapter.
//
// This is the COLLECTION layer of the §0221 slice. It commits the
// observed fingerprint verbatim as an immutable Cat I observation —
// never a verdict. Whether the fingerprint indicates automation is a
// Category III question answered downstream by
// find-automation-group-candidates-tls (§2.2 epistemic separation).
//
// Positional file arg, stdin fallback (mirrors ingest-cic-ids §0204
// operator-workflow choice 1). Per-event substrate.Append; no batching.
//
// Output:
//   stdout — tls_fingerprint.Report JSON.
//   stderr — final summary line.
//
// Exit codes: 0 success; 2 tool/config error; 3 substrate error or
// (-strict && RowsRejected > 0).
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/adapters/tls_fingerprint"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

const (
	exitToolError       = 2
	exitTargetIntegrity = 3
)

const (
	defaultCollectorRef = "tls-fingerprint-adapter:v1"
	channelFile         = "tls-fingerprint-file"
	channelStdin        = "stdin"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// run is the testable entry point; returns the process exit code.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("ingest-tls-fingerprint", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "./ghost-trace.db", "SQLite primary-event-log path")
	blobDir := fs.String("blobs", "./blobs", "content-addressed blob-store directory")
	channelFlag := fs.String("channel", "", `ingestion channel identifier (default: "`+channelFile+`" with path arg, "`+channelStdin+`" otherwise)`)
	collectorRef := fs.String("collector", defaultCollectorRef, "collector_ref populated on emitted observations (per-record collector_ref overrides)")
	strict := fs.Bool("strict", false, "exit non-zero if RowsRejected > 0")

	if err := fs.Parse(args); err != nil {
		return exitToolError
	}
	if fs.NArg() > 1 {
		fmt.Fprintf(stderr, "ingest-tls-fingerprint: too many positional arguments (expected 0 or 1, got %d)\n", fs.NArg())
		return exitToolError
	}

	var (
		reader  io.Reader
		channel string
	)
	if fs.NArg() == 1 {
		path := fs.Arg(0)
		f, err := os.Open(path)
		if err != nil {
			fmt.Fprintf(stderr, "ingest-tls-fingerprint: open input: %v\n", err)
			return exitToolError
		}
		defer func() { _ = f.Close() }()
		reader = f
		channel = channelFile
	} else {
		reader = stdin
		channel = channelStdin
	}
	if *channelFlag != "" {
		channel = *channelFlag
	}

	ctx := context.Background()
	sub, err := substrate.Open(ctx, *dbPath, *blobDir)
	if err != nil {
		fmt.Fprintf(stderr, "ingest-tls-fingerprint: open substrate: %v\n", err)
		return exitToolError
	}
	defer func() { _ = sub.Close() }()

	ingester := ingest.New(sub, time.Now)

	report, ingestErr := tls_fingerprint.Ingest(ctx, ingester, reader, *collectorRef, ingest.Envelope{Channel: channel})

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if encErr := enc.Encode(report); encErr != nil {
		fmt.Fprintf(stderr, "ingest-tls-fingerprint: encode json: %v\n", encErr)
		return exitToolError
	}

	fmt.Fprintf(stderr,
		"ingest-tls-fingerprint: rows_parsed=%d rows_rejected=%d observations_committed=%d elapsed_ns=%d channel=%q\n",
		report.RowsParsed, report.RowsRejected, report.ObservationsCommitted, report.ElapsedNanos, channel)

	if ingestErr != nil {
		fmt.Fprintf(stderr, "ingest-tls-fingerprint: %v\n", ingestErr)
		return exitTargetIntegrity
	}
	if *strict && report.RowsRejected > 0 {
		return exitTargetIntegrity
	}
	return 0
}
