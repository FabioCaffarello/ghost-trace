package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/adapters/cic_ids"
)

// miniCSV is a 3-data-row CIC-IDS-2017 sample with: row 1 TCP w/ flags
// → 3 observations (2 ip_asn + 1 tcp_fingerprint); row 2 UDP → 2
// observations (2 ip_asn); row 3 Source Port empty → rejected.
//
// Header preserved verbatim from CICFlowMeter output format per
// cic_ids/columns.go; inlining is acceptable for CLI surface tests
// because the column-name registry is library-owned + already covered
// by cic_ids_test.go.
const miniCSV = `Flow ID,Source IP,Source Port,Destination IP,Destination Port,Protocol,Timestamp,Flow Duration,FIN Flag Count,SYN Flag Count,RST Flag Count,PSH Flag Count,ACK Flag Count,URG Flag Count,CWE Flag Count,ECE Flag Count,Fwd Header Length,Bwd Header Length,Init_Win_bytes_forward,Init_Win_bytes_backward,Label
flow-1,192.0.2.10,49152,198.51.100.5,443,6,03/07/2017 09:15,12345,1,1,0,3,8,0,0,0,32,32,65535,29200,BENIGN
flow-2,192.0.2.12,49200,198.51.100.7,53,17,03/07/2017 09:17,5000,0,0,0,0,0,0,0,0,0,0,0,0,BENIGN
flow-3,192.0.2.14,,198.51.100.9,8080,6,03/07/2017 09:19,1000,0,0,0,0,0,0,0,0,0,0,0,0,BENIGN
`

// runWith invokes run() against a freshly-isolated substrate under dir.
// Returns (exit code, stdout, stderr).
func runWith(t *testing.T, dir string, extraArgs []string, stdin string) (int, string, string) {
	t.Helper()
	args := []string{
		"-db", filepath.Join(dir, "test.db"),
		"-blobs", filepath.Join(dir, "blobs"),
	}
	args = append(args, extraArgs...)
	var stdout, stderr bytes.Buffer
	code := run(args, strings.NewReader(stdin), &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func decodeReport(t *testing.T, stdout string) cic_ids.Report {
	t.Helper()
	var r cic_ids.Report
	if err := json.NewDecoder(strings.NewReader(stdout)).Decode(&r); err != nil {
		t.Fatalf("decode report json: %v\nstdout=%q", err, stdout)
	}
	return r
}

func TestRun_StdinPath(t *testing.T) {
	dir := t.TempDir()
	code, stdout, stderr := runWith(t, dir, nil, miniCSV)

	if code != 0 {
		t.Fatalf("exit code: got %d want 0 (stderr=%q)", code, stderr)
	}
	r := decodeReport(t, stdout)
	if r.RowsParsed != 2 {
		t.Errorf("RowsParsed: got %d want 2", r.RowsParsed)
	}
	if r.RowsRejected != 1 {
		t.Errorf("RowsRejected: got %d want 1", r.RowsRejected)
	}
	// row 1 TCP w/ flags → 3 obs; row 2 UDP → 2 obs.
	if r.ObservationsCommitted != 5 {
		t.Errorf("ObservationsCommitted: got %d want 5", r.ObservationsCommitted)
	}
	if r.IpAsnEmitted != 4 {
		t.Errorf("IpAsnEmitted: got %d want 4", r.IpAsnEmitted)
	}
	if r.TcpFingerprintEmitted != 1 {
		t.Errorf("TcpFingerprintEmitted: got %d want 1", r.TcpFingerprintEmitted)
	}
	if !strings.Contains(stderr, `channel="stdin"`) {
		t.Errorf("stderr should report channel=stdin; got %q", stderr)
	}
}

func TestRun_FilePath(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "input.csv")
	if err := os.WriteFile(csvPath, []byte(miniCSV), 0o644); err != nil {
		t.Fatalf("write csv: %v", err)
	}
	code, stdout, stderr := runWith(t, dir, []string{csvPath}, "")

	if code != 0 {
		t.Fatalf("exit code: got %d want 0 (stderr=%q)", code, stderr)
	}
	r := decodeReport(t, stdout)
	if r.RowsParsed != 2 {
		t.Errorf("RowsParsed: got %d want 2", r.RowsParsed)
	}
	if !strings.Contains(stderr, `channel="cic-ids-file"`) {
		t.Errorf("stderr should report channel=cic-ids-file; got %q", stderr)
	}
}

func TestRun_ChannelOverride(t *testing.T) {
	dir := t.TempDir()
	code, _, stderr := runWith(t, dir, []string{"-channel", "cic-ids-honeypot-v1"}, miniCSV)
	if code != 0 {
		t.Fatalf("exit code: got %d want 0 (stderr=%q)", code, stderr)
	}
	if !strings.Contains(stderr, `channel="cic-ids-honeypot-v1"`) {
		t.Errorf("stderr should report overridden channel; got %q", stderr)
	}
}

func TestRun_Idempotent(t *testing.T) {
	dir := t.TempDir()

	code1, stdout1, stderr1 := runWith(t, dir, nil, miniCSV)
	if code1 != 0 {
		t.Fatalf("first run exit code: got %d want 0 (stderr=%q)", code1, stderr1)
	}
	r1 := decodeReport(t, stdout1)

	code2, stdout2, stderr2 := runWith(t, dir, nil, miniCSV)
	if code2 != 0 {
		t.Fatalf("second run exit code: got %d want 0 (stderr=%q)", code2, stderr2)
	}
	r2 := decodeReport(t, stdout2)

	// Both runs parse identically — library is content-addressed.
	// The Report counters mirror per-run library work; substrate
	// idempotency means re-ingest does NOT add observations even
	// though the counters report identical totals (the counters
	// reflect Append CALLS, not net new rows). Operator detects re-
	// ingest by observing that the substrate row count did not grow,
	// not by the Report differing — silent exit 0 per §0204 dec. #8.
	if r1.RowsParsed != r2.RowsParsed {
		t.Errorf("RowsParsed should be identical across runs: got %d then %d", r1.RowsParsed, r2.RowsParsed)
	}
	if r1.ObservationsCommitted != r2.ObservationsCommitted {
		t.Errorf("ObservationsCommitted should be identical across runs: got %d then %d", r1.ObservationsCommitted, r2.ObservationsCommitted)
	}
}

func TestRun_StrictFlagExits3OnRowsRejected(t *testing.T) {
	dir := t.TempDir()
	code, stdout, stderr := runWith(t, dir, []string{"-strict"}, miniCSV)

	if code != exitTargetIntegrity {
		t.Fatalf("exit code: got %d want %d (stderr=%q)", code, exitTargetIntegrity, stderr)
	}
	// Report should still be emitted to stdout under strict-failure.
	r := decodeReport(t, stdout)
	if r.RowsRejected == 0 {
		t.Errorf("expected RowsRejected > 0 to trigger -strict exit, got 0")
	}
}

func TestRun_StrictNoEffectWhenZeroRejected(t *testing.T) {
	// CSV with zero rejected rows: header + one valid TCP row.
	cleanCSV := `Flow ID,Source IP,Source Port,Destination IP,Destination Port,Protocol,Timestamp,Flow Duration,FIN Flag Count,SYN Flag Count,RST Flag Count,PSH Flag Count,ACK Flag Count,URG Flag Count,CWE Flag Count,ECE Flag Count,Fwd Header Length,Bwd Header Length,Init_Win_bytes_forward,Init_Win_bytes_backward,Label
flow-1,192.0.2.10,49152,198.51.100.5,443,6,03/07/2017 09:15,12345,1,1,0,3,8,0,0,0,32,32,65535,29200,BENIGN
`
	dir := t.TempDir()
	code, _, stderr := runWith(t, dir, []string{"-strict"}, cleanCSV)
	if code != 0 {
		t.Fatalf("exit code with zero rejected + -strict: got %d want 0 (stderr=%q)", code, stderr)
	}
}

func TestRun_ProgressEmission(t *testing.T) {
	dir := t.TempDir()
	// miniCSV has 4 newlines total (1 header + 3 rows + trailing
	// newline on the last row). With -progress 2 the wrapper emits
	// at newlines 2 and 4 → exactly 2 progress lines.
	code, _, stderr := runWith(t, dir, []string{"-progress", "2"}, miniCSV)
	if code != 0 {
		t.Fatalf("exit code: got %d want 0 (stderr=%q)", code, stderr)
	}
	got := strings.Count(stderr, "[ingest-cic-ids] progress lines_read=")
	if got != 2 {
		t.Errorf("progress lines: got %d want 2; stderr=%q", got, stderr)
	}
}

func TestRun_ProgressDisabledZero(t *testing.T) {
	dir := t.TempDir()
	code, _, stderr := runWith(t, dir, []string{"-progress", "0"}, miniCSV)
	if code != 0 {
		t.Fatalf("exit code: got %d want 0 (stderr=%q)", code, stderr)
	}
	if strings.Contains(stderr, "[ingest-cic-ids] progress lines_read=") {
		t.Errorf("progress disabled should suppress progress lines; stderr=%q", stderr)
	}
}

func TestRun_TooManyPositionalArgs(t *testing.T) {
	dir := t.TempDir()
	code, _, stderr := runWith(t, dir, []string{"a", "b"}, "")
	if code != exitToolError {
		t.Fatalf("exit code: got %d want %d (stderr=%q)", code, exitToolError, stderr)
	}
	if !strings.Contains(stderr, "too many positional arguments") {
		t.Errorf("stderr should explain arg count; got %q", stderr)
	}
}

func TestRun_BadFilePath(t *testing.T) {
	dir := t.TempDir()
	code, _, stderr := runWith(t, dir, []string{filepath.Join(dir, "does-not-exist.csv")}, "")
	if code != exitToolError {
		t.Fatalf("exit code: got %d want %d (stderr=%q)", code, exitToolError, stderr)
	}
	if !strings.Contains(stderr, "open input") {
		t.Errorf("stderr should explain file open failure; got %q", stderr)
	}
}
