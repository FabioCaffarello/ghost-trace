// Command ingest-cic-ids is the operator-facing CLI for streaming a
// CIC-IDS-2017 CSV file into the substrate via the cic_ids adapter
// library. Thin wrapper around cic_ids.Ingest per decision-log §0204
// (CLI lift); zero new semantics — exposes + documents the eight
// operator workflow choices §0145 Consequences deferred to this PR.
//
// Operator workflow choices cravadas at §0204:
//
//  1. Positional file arg, stdin fallback. With path arg, the input is
//     opened with os.Open; without it, stdin is consumed.
//  2. Per-event substrate.Append — NO CLI-side batching. Single-writer-
//     serialization (concurrency-pattern.md §Substrate-Writer
//     Serialization, §0027) already serializes via WAL writeMu; batch
//     insertion under WAL does not gain throughput. §2.1 immutability
//     post-commit is the secondary alignment — per-event commit avoids
//     CLI-introduced batch-rollback semantics that would complicate
//     what content-hash idempotency already guarantees.
//  3. Error recovery: library semantics verbatim — continue-on-parse-
//     error (RowsRejected counter), abort-on-substrate-error (partial
//     Report still emitted). Exit codes mirror §0173 replay-all-*:
//     0 success / 2 tool-or-config-error / 3 substrate-error or
//     -strict && RowsRejected > 0.
//  4. -progress N stderr line every N lines read; default 10000 (≈0.1%
//     of a 1M-row dataset); 0 disables.
//  5. Timestamp fallback determinism: library falls back to row-index
//     nanoseconds when CIC-IDS Timestamp column is missing or
//     unparseable (cic_ids.go ParseTimestamp). This fallback is
//     invisible in the current Report struct; if §0205 manifest needs
//     audit-grade visibility, TimestampsRecovered counter is added to
//     library Report via separate §0022 entry.
//  6. CSV encoding: Go csv.Reader default. Non-UTF-8 surfaces as parse
//     error → RowsRejected.
//  7. -channel default contextual: "cic-ids-file" with path arg,
//     "stdin" without; operator override via flag.
//  8. Idempotency UX: silent exit 0. Operator detects re-ingest via
//     RowsParsed > 0 && ObservationsCommitted == 0.
//
// Output:
//
//	stdout — cic_ids.Report JSON (distinct from §0163 F3-envelope shape;
//	the Report struct is the CLI category's own output contract).
//	stderr — progress lines (when -progress > 0) + final summary line.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/adapters/cic_ids"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

const (
	exitToolError       = 2
	exitTargetIntegrity = 3
)

const (
	defaultCollectorRef  = "cic-ids-2017-adapter:v1"
	defaultProgressEvery = 10000
	channelFile          = "cic-ids-file"
	channelStdin         = "stdin"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// run is the testable entry point. Returns the process exit code so
// the test harness can assert exit semantics without launching a
// subprocess.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("ingest-cic-ids", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", "./ghost-trace.db", "SQLite primary-event-log path")
	blobDir := fs.String("blobs", "./blobs", "content-addressed blob-store directory")
	channelFlag := fs.String("channel", "", `ingestion channel identifier (default: "`+channelFile+`" with path arg, "`+channelStdin+`" otherwise)`)
	collectorRef := fs.String("collector", defaultCollectorRef, "collector_ref populated on emitted observations")
	progressEvery := fs.Int("progress", defaultProgressEvery, "emit stderr progress line every N CSV lines read (0 disables)")
	strict := fs.Bool("strict", false, "exit non-zero if RowsRejected > 0")

	if err := fs.Parse(args); err != nil {
		return exitToolError
	}
	if fs.NArg() > 1 {
		fmt.Fprintf(stderr, "ingest-cic-ids: too many positional arguments (expected 0 or 1, got %d)\n", fs.NArg())
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
			fmt.Fprintf(stderr, "ingest-cic-ids: open input: %v\n", err)
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
		fmt.Fprintf(stderr, "ingest-cic-ids: open substrate: %v\n", err)
		return exitToolError
	}
	defer func() { _ = sub.Close() }()

	ingester := ingest.New(sub, time.Now)

	if *progressEvery > 0 {
		reader = &progressReader{inner: reader, every: *progressEvery, w: stderr}
	}

	report, ingestErr := cic_ids.Ingest(ctx, ingester, reader, *collectorRef, ingest.Envelope{Channel: channel})

	// Emit Report JSON regardless of error path: partial Report is
	// auditable even on substrate-error abort per §0204 decision #3.
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if encErr := enc.Encode(report); encErr != nil {
		fmt.Fprintf(stderr, "ingest-cic-ids: encode json: %v\n", encErr)
		return exitToolError
	}

	fmt.Fprintf(stderr,
		"ingest-cic-ids: rows_parsed=%d rows_rejected=%d observations_committed=%d ip_asn=%d tcp_fingerprint=%d flow_statistics_dropped=%d channel=%q\n",
		report.RowsParsed, report.RowsRejected, report.ObservationsCommitted,
		report.IpAsnEmitted, report.TcpFingerprintEmitted, report.FlowStatisticsDropped, channel)

	if ingestErr != nil {
		fmt.Fprintf(stderr, "ingest-cic-ids: %v\n", ingestErr)
		return exitTargetIntegrity
	}
	if *strict && report.RowsRejected > 0 {
		return exitTargetIntegrity
	}
	return 0
}

// progressReader wraps an io.Reader and emits a stderr progress line
// every `every` newlines observed. Newline-count is a deterministic
// proxy for CSV rows; multi-line quoted fields produce mild overcount
// acceptable for operator visibility (not for auditing — audit-grade
// numbers come from the Report counters at end).
type progressReader struct {
	inner    io.Reader
	every    int
	w        io.Writer
	bytes    int64
	newlines int
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.inner.Read(b)
	p.bytes += int64(n)
	for i := 0; i < n; i++ {
		if b[i] == '\n' {
			p.newlines++
			if p.newlines%p.every == 0 {
				fmt.Fprintf(p.w, "[ingest-cic-ids] progress lines_read=%d bytes_read=%d\n", p.newlines, p.bytes)
			}
		}
	}
	return n, err
}
