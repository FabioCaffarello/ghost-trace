// Integration test exercising the full F3 inference loop against
// real CIC-IDS-2017 adapter output per §0156: first empirical pressure
// through the complete loop validating that non-firing by
// modality-mismatch is structurally diagnosable per §0154 MO4.
//
// Scenario: CIC-IDS adapter produces NetworkObservation records;
// cdp_marker_density_v1 signature consumes BrowserObservation records.
// Expected outcome: zero candidates (signature finds zero observations
// of its required class); §0154 EvaluationStats.ObservationsScanned
// reflects this exactly (zero — no BrowserObservation in substrate).
//
// This is the first end-to-end exercise of the diagnostic surface
// landed at §0154 + §0155: non-firing for the expected reason
// (modality mismatch) shows up in instrumentation, NOT as silent
// emptiness.
package main

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/adapters/cic_ids"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/signatures"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// sampleCICIDSCSV mirrors the structure of CIC-IDS-2017 CSV output
// from CICFlowMeter (per Sharafaldin et al., ICISSP 2018). Embedded
// inline to keep the integration test self-contained — no relative
// fixture-path dependency on the cic_ids package's testdata/.
const sampleCICIDSCSV = `Flow ID,Source IP,Source Port,Destination IP,Destination Port,Protocol,Timestamp,Flow Duration,FIN Flag Count,SYN Flag Count,RST Flag Count,PSH Flag Count,ACK Flag Count,URG Flag Count,CWE Flag Count,ECE Flag Count,Fwd Header Length,Bwd Header Length,Init_Win_bytes_forward,Init_Win_bytes_backward,Label
192.0.2.10-198.51.100.5-49152-443-6,192.0.2.10,49152,198.51.100.5,443,6,03/07/2017 09:15,12345,1,1,0,3,8,0,0,0,32,32,65535,29200,BENIGN
192.0.2.11-198.51.100.6-53124-80-6,192.0.2.11,53124,198.51.100.6,80,6,03/07/2017 09:16,67890,1,1,0,0,4,0,0,0,32,32,29200,29200,BENIGN
192.0.2.13-198.51.100.8-49300-443-6,192.0.2.13,49300,198.51.100.8,443,6,03/07/2017 09:18,98765,2,3,1,5,12,0,0,0,32,32,65535,29200,DoS-Slowloris
`

func newCICIDSTestSubstrate(t *testing.T) (*substrate.Substrate, *ingest.Ingester) {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })
	clock := func() time.Time { return time.Unix(0, 1716120000000000777) }
	return sub, ingest.New(sub, clock)
}

// TestCICIDSIngestionThenSignature_NonFiringByModalityMismatch
// exercises the complete F3 loop with REAL CIC-IDS adapter output as
// the substrate input. CIC-IDS produces NetworkObservation +
// (per row) source-side + destination-side ip_asn + (per TCP row with
// flags) tcp_fingerprint; never BrowserObservation. cdp_marker_density_v1
// consumes BrowserObservation only. Expected: zero candidates +
// EvaluationStats.ObservationsScanned == 0 (no observations of the
// required class in substrate).
//
// This is the §0143 Sub-benchmark 1 diagnostic surface's first
// operational exercise: non-firing is INFORMATIVE — the operator sees
// "0 candidates" alongside "0 observations scanned" + non-zero
// substrate event count, distinguishing "signature ran but found
// nothing matching its class" from "substrate empty".
func TestCICIDSIngestionThenSignature_NonFiringByModalityMismatch(t *testing.T) {
	sub, in := newCICIDSTestSubstrate(t)
	ctx := context.Background()

	// Step 1: Ingest CIC-IDS sample via real adapter.
	report, err := cic_ids.Ingest(
		ctx,
		in,
		bytes.NewReader([]byte(sampleCICIDSCSV)),
		"cic-ids-2017-adapter:v1",
		ingest.Envelope{Channel: "test"},
	)
	if err != nil {
		t.Fatalf("cic_ids.Ingest: %v", err)
	}
	// Verify adapter produced expected output (3 TCP rows → 9 obs total:
	// 3 rows × (1 src ip_asn + 1 dst ip_asn + 1 tcp_fingerprint)).
	if report.RowsParsed != 3 {
		t.Fatalf("RowsParsed: got %d want 3", report.RowsParsed)
	}
	if report.IpAsnEmitted != 6 {
		t.Errorf("IpAsnEmitted: got %d want 6 (3 rows × 2 sides)", report.IpAsnEmitted)
	}
	if report.TcpFingerprintEmitted != 3 {
		t.Errorf("TcpFingerprintEmitted: got %d want 3 (3 TCP rows with flags)", report.TcpFingerprintEmitted)
	}
	if report.ObservationsCommitted != 9 {
		t.Fatalf("ObservationsCommitted: got %d want 9", report.ObservationsCommitted)
	}

	// Step 2: Run orchestrator's BrowserObservation walk.
	// The substrate has 9 NetworkObservation records but ZERO
	// BrowserObservation records.
	observations, err := collectBrowserObservations(ctx, sub)
	if err != nil {
		t.Fatalf("collectBrowserObservations: %v", err)
	}
	if len(observations) != 0 {
		t.Fatalf("BrowserObservation count: got %d want 0 (CIC-IDS only produces NetworkObservation)", len(observations))
	}

	// Step 3: Run cdp_marker_density_v1 signature against empty
	// observation slice. Expected: zero candidates + zero
	// observations_scanned + zero of every skip counter (no
	// observations means no skip decisions).
	sig := &signatures.CDPMarkerDensityV1{}
	result, err := sig.EvaluateBrowser(ctx, observations)
	if err != nil {
		t.Fatalf("EvaluateBrowser: %v", err)
	}

	// §0154 MO4 diagnostic invariant: non-firing is structurally
	// diagnosable. The orchestrator can distinguish three causes:
	//   (a) substrate empty                   — ObservationsScanned == 0
	//   (b) modality mismatch                 — ObservationsScanned == 0
	//                                          (covered by this test)
	//   (c) signature insensitive / threshold high
	//                                         — ObservationsScanned > 0
	//                                          (covered by existing
	//                                          TestCDPMarkerDensityV1_BelowThreshold_NoCandidate)
	//
	// CASE (b) is the cause here. From the orchestrator's perspective,
	// case (a) and case (b) are indistinguishable at the signature
	// layer — BOTH produce ObservationsScanned == 0 in the
	// BrowserObservation walk. The substrate IS non-empty (9
	// NetworkObservation records); the differentiator is at the
	// substrate-walk filter (only BrowserObservation records reach
	// the signature). This is §0154 MO4's distinction codified:
	// non-firing diagnostic surface separates evaluation-time
	// (signature) from substrate-time (orchestrator filter).
	if len(result.Candidates) != 0 {
		t.Errorf("Candidates: got %d want 0 (no BrowserObservation in substrate)", len(result.Candidates))
	}
	if result.Stats.ObservationsScanned != 0 {
		t.Errorf("ObservationsScanned: got %d want 0", result.Stats.ObservationsScanned)
	}
	if result.Stats.ObservationsSkippedNoActor != 0 {
		t.Errorf("ObservationsSkippedNoActor: got %d want 0 (no observations to skip)", result.Stats.ObservationsSkippedNoActor)
	}
	if result.Stats.ObservationsSkippedWrongModality != 0 {
		t.Errorf("ObservationsSkippedWrongModality: got %d want 0 (no observations to skip)", result.Stats.ObservationsSkippedWrongModality)
	}
	if result.Stats.ActorsAggregated != 0 {
		t.Errorf("ActorsAggregated: got %d want 0", result.Stats.ActorsAggregated)
	}
	if result.Stats.ActorsAboveThreshold != 0 {
		t.Errorf("ActorsAboveThreshold: got %d want 0", result.Stats.ActorsAboveThreshold)
	}

	// Step 4: Validate the substrate IS populated with adapter
	// output. This is the differentiator between case (a) and case (b)
	// at the orchestrator level — operator knows substrate is
	// non-empty even though BrowserObservation walk returned zero.
	count, err := sub.Count(ctx)
	if err != nil {
		t.Fatalf("sub.Count: %v", err)
	}
	// 9 NetworkObservation + 9 paired IngestionEvent per §0038.
	if got, want := count, int64(18); got != want {
		t.Errorf("substrate Count: got %d want %d (9 NetworkObservation + 9 paired IngestionEvent)", got, want)
	}
}

// TestCICIDSIngestionThenSignature_WithInjectedBrowserObservation
// extends the prior scenario with two additional BrowserObservation
// commits demonstrating the diagnostic surface distinguishes
// modality-mismatch (signature filter at substrate-walk layer) from
// modality-present-but-below-threshold (signature filter at evaluation
// layer). Confirms §0154 MO4 + §0155 two-layer diagnostic separation
// holds end-to-end.
func TestCICIDSIngestionThenSignature_WithInjectedBrowserObservation(t *testing.T) {
	sub, in := newCICIDSTestSubstrate(t)
	ctx := context.Background()

	// Step 1: Ingest CIC-IDS (NetworkObservation only).
	_, err := cic_ids.Ingest(
		ctx,
		in,
		bytes.NewReader([]byte(sampleCICIDSCSV)),
		"cic-ids-2017-adapter:v1",
		ingest.Envelope{Channel: "test"},
	)
	if err != nil {
		t.Fatalf("cic_ids.Ingest: %v", err)
	}

	// Step 2: Inject 2 BrowserObservation with cdp_marker, both BELOW
	// signature default threshold (detection_count = 1 each; per-actor
	// total = 1; threshold = 2).
	appendBrowserObs(t, in, "actor-suspect", []string{"navigator.webdriver=true"}, 1)
	appendBrowserObs(t, in, "actor-other", []string{"$cdc_test"}, 1)

	// Step 3: Run orchestrator.
	observations, err := collectBrowserObservations(ctx, sub)
	if err != nil {
		t.Fatalf("collectBrowserObservations: %v", err)
	}
	if got, want := len(observations), 2; got != want {
		t.Fatalf("BrowserObservation count: got %d want %d (2 injected)", got, want)
	}

	sig := &signatures.CDPMarkerDensityV1{}
	result, err := sig.EvaluateBrowser(ctx, observations)
	if err != nil {
		t.Fatalf("EvaluateBrowser: %v", err)
	}

	// Diagnostic invariant: case (c) — signature ran against 2
	// observations, aggregated 2 distinct actors, but neither met
	// threshold. ObservationsScanned > 0 distinguishes this from
	// case (a)/case (b).
	if len(result.Candidates) != 0 {
		t.Errorf("Candidates: got %d want 0 (both actors below threshold)", len(result.Candidates))
	}
	if result.Stats.ObservationsScanned != 2 {
		t.Errorf("ObservationsScanned: got %d want 2", result.Stats.ObservationsScanned)
	}
	if result.Stats.ActorsAggregated != 2 {
		t.Errorf("ActorsAggregated: got %d want 2", result.Stats.ActorsAggregated)
	}
	if result.Stats.ActorsAboveThreshold != 0 {
		t.Errorf("ActorsAboveThreshold: got %d want 0 (both below threshold=2)", result.Stats.ActorsAboveThreshold)
	}
	if got := result.Stats.PerCollector["browser-sdk:v1"]; got != 2 {
		t.Errorf("PerCollector[browser-sdk:v1]: got %d want 2", got)
	}
}
