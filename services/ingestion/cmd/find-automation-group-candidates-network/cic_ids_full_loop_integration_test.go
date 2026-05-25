// End-to-end integration test demonstrating BOTH §0162 gap-closures
// per decision-log §0169. Exercises the full CIC-IDS → Cat II
// attribution → F3 signature → candidate emission loop:
//
//   1. CIC-IDS adapter (§0145) commits NetworkObservation records
//      with empty actor_ref + populated tcp_fingerprint sub-modality
//      (flags_sequence + window_size; p0f_signature EMPTY per
//      CICFlowMeter limitation).
//
//   2. attribution.DeriveAll with network_5tuple_actor_v1 (§0168)
//      synthesizes a Cat II DerivedActorAttribution per Cat I
//      record. Closes gap (1).
//
//   3. attribution.CollectAttributionView walks the substrate's
//      Cat II records into a read-only lookup.
//
//   4. tcp_flow_features_clustering_v1 (§0169) consumes the
//      NetworkObservation slice + the AttributionView. Clusters by
//      (flags_sequence, window_size) — the feature vector CIC-IDS
//      DOES populate. Closes gap (2) without fabricating p0f
//      canonical form (preserves §3 N1 truth-vs-structure).
//
//   5. The signature emits candidates whose ActorRefs are the
//      Cat II derived attributions + SourceHashes thread both
//      Cat I observation hashes AND Cat II derivation hashes per
//      §2.3 chain.
//
// This test is the structural witness that the §0162 reachability
// claim — over-projected at §0161 + corrected at §0162 — is now
// EMPIRICALLY satisfied for a CIC-IDS-shaped flow-level dataset.
// Compared with §0162's integration test (CIC-IDS-only fixture →
// 0 candidates, both gaps surfaced as skip counters), this test
// runs the SAME pipeline + the §0168 attribution layer + the §0169
// alternative signature, and the candidates DO emit.
//
// Scope: structural connectivity validation across the full
// CIC-IDS → F3 loop. Per-component semantics covered by component
// unit tests. The CIC-IDS sample is intentionally constructed
// (3 rows; same flow shape across rows is the cluster) to keep the
// test deterministic; real CIC-IDS-2017 dataset shapes are out of
// scope for this test.
package main

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/adapters/cic_ids"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/attribution"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/signatures"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// cicIDSClusterableCSV: three TCP rows constructed to share an
// identical flow-feature vector (same flag counts + same
// InitWinFwd) but distinct source IPs. After Cat II attribution
// derives per-row actor_refs (5-tuple-based), the three rows
// produce 3 distinct derived actors that cluster together on the
// shared (flags_sequence, window_size) feature.
//
// Constructed (not from real dataset) for deterministic test
// behavior; real dataset rows have more variance.
const cicIDSClusterableCSV = `Flow ID,Source IP,Source Port,Destination IP,Destination Port,Protocol,Timestamp,Flow Duration,FIN Flag Count,SYN Flag Count,RST Flag Count,PSH Flag Count,ACK Flag Count,URG Flag Count,CWE Flag Count,ECE Flag Count,Fwd Header Length,Bwd Header Length,Init_Win_bytes_forward,Init_Win_bytes_backward,Label
192.0.2.10-198.51.100.5-49152-443-6,192.0.2.10,49152,198.51.100.5,443,6,03/07/2017 09:15,12345,1,1,0,3,8,0,0,0,32,32,65535,29200,BENIGN
192.0.2.11-198.51.100.5-49152-443-6,192.0.2.11,49152,198.51.100.5,443,6,03/07/2017 09:16,12345,1,1,0,3,8,0,0,0,32,32,65535,29200,BENIGN
192.0.2.12-198.51.100.5-49152-443-6,192.0.2.12,49152,198.51.100.5,443,6,03/07/2017 09:17,12345,1,1,0,3,8,0,0,0,32,32,65535,29200,BENIGN
`

func newCICFullLoopSubstrate(t *testing.T) (*substrate.Substrate, *ingest.Ingester) {
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

// TestCICIDS_FullLoop_BothGapsClosed_EmitsCandidate is the §0169
// structural witness: CIC-IDS substrate state → Cat II attribution
// → F3 alternative signature → cluster candidate emitted. Both
// §0162 gaps now closed end-to-end.
func TestCICIDS_FullLoop_BothGapsClosed_EmitsCandidate(t *testing.T) {
	sub, in := newCICFullLoopSubstrate(t)
	ctx := context.Background()
	clock := func() time.Time { return time.Unix(0, 1716120000000000777) }

	// Step 1: Ingest CIC-IDS sample. Three rows × three observations
	// per row (2 ip_asn + 1 tcp_fingerprint) = 9 NetworkObservation
	// records. All carry empty actor_ref (CIC-IDS is flow-level —
	// §0162 gap (1)). The tcp_fingerprint records carry empty
	// p0f_signature (CICFlowMeter limitation — §0162 gap (2)).
	report, err := cic_ids.Ingest(ctx, in, bytes.NewReader([]byte(cicIDSClusterableCSV)),
		"cic-ids-2017-adapter:v1", ingest.Envelope{Channel: "test"})
	if err != nil {
		t.Fatalf("cic_ids.Ingest: %v", err)
	}
	if report.TcpFingerprintEmitted != 3 {
		t.Fatalf("TcpFingerprintEmitted: got %d want 3", report.TcpFingerprintEmitted)
	}
	if report.ObservationsCommitted != 9 {
		t.Fatalf("ObservationsCommitted: got %d want 9", report.ObservationsCommitted)
	}

	// Step 2: Run attribution.DeriveAll with network_5tuple_actor_v1.
	// Closes §0162 gap (1) — each Cat I NetworkObservation gets a
	// Cat II DerivedActorAttribution pointing to it.
	derivedReport, err := attribution.DeriveAll(ctx, sub, attribution.Network5TupleActorV1{}, clock)
	if err != nil {
		t.Fatalf("attribution.DeriveAll: %v", err)
	}
	if derivedReport.Examined != 9 {
		t.Errorf("DeriveAll.Examined: got %d want 9", derivedReport.Examined)
	}
	if derivedReport.NewlyDerived != 9 {
		t.Errorf("DeriveAll.NewlyDerived: got %d want 9 (all CIC-IDS records have endpoint_ref + modality)", derivedReport.NewlyDerived)
	}
	if derivedReport.Skipped != 0 {
		t.Errorf("DeriveAll.Skipped: got %d want 0", derivedReport.Skipped)
	}

	// Step 3: CollectAttributionView reads Cat II records into a
	// read-only lookup.
	view, err := attribution.CollectAttributionView(ctx, sub)
	if err != nil {
		t.Fatalf("CollectAttributionView: %v", err)
	}

	// Step 4: Collect NetworkObservation records + run the new
	// signature (closes §0162 gap (2) via flow-feature clustering).
	observations, err := collectNetworkObservations(ctx, sub)
	if err != nil {
		t.Fatalf("collectNetworkObservations: %v", err)
	}
	if len(observations) != 9 {
		t.Fatalf("observations count: got %d want 9", len(observations))
	}

	sig := &signatures.TCPFlowFeaturesClusteringV1{}
	res, err := sig.EvaluateNetwork(ctx, observations, view)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}

	// Step 5: Verify the cluster emerged. The 3 tcp_fingerprint
	// records share (flags=[SYN,FIN,PSH,PSH,PSH,ACK,...], window=
	// 65535); the 6 ip_asn records have non-tcp_fingerprint modality
	// and skip at SkippedWrongModality.
	if len(res.Candidates) != 1 {
		t.Fatalf("Candidates: got %d want 1 (3 CIC-IDS flows share feature vector)", len(res.Candidates))
	}
	c := res.Candidates[0]
	if c.SignatureName != "tcp_flow_features_clustering_v1" {
		t.Errorf("SignatureName: got %q", c.SignatureName)
	}
	if c.HypothesisSubtype != signatures.HypothesisSubtypeAutomationGroup {
		t.Errorf("HypothesisSubtype: got %v want AutomationGroup", c.HypothesisSubtype)
	}
	if len(c.ActorRefs) != 3 {
		t.Errorf("ActorRefs count: got %d want 3 (derived actors)", len(c.ActorRefs))
	}
	// Each derived actor_ref carries the 5-tuple form.
	for _, ar := range c.ActorRefs {
		if ar == "" {
			t.Errorf("ActorRef is empty; expected derived form")
		}
	}

	// §2.3 chain: each Cat I observation hash + its Cat II derivation
	// hash threaded into SourceHashes. 3 contributing Cat I records
	// × (1 Cat I + 1 Cat II hash) = 6.
	if len(c.SourceHashes) != 6 {
		t.Errorf("SourceHashes count: got %d want 6 (3 Cat I + 3 Cat II per §2.3)", len(c.SourceHashes))
	}

	// Diagnostic counters: scanned=9, skipped 6 ip_asn obs at wrong-
	// modality, 3 tcp_fingerprint obs aggregated into 1 cluster of 3
	// derived actors.
	if res.Stats.ObservationsScanned != 9 {
		t.Errorf("ObservationsScanned: got %d want 9", res.Stats.ObservationsScanned)
	}
	if res.Stats.ObservationsSkippedNoActor != 0 {
		t.Errorf("ObservationsSkippedNoActor: got %d want 0 (Cat II filled all gaps)", res.Stats.ObservationsSkippedNoActor)
	}
	if res.Stats.ObservationsSkippedWrongModality != 6 {
		t.Errorf("ObservationsSkippedWrongModality: got %d want 6 (6 ip_asn obs)", res.Stats.ObservationsSkippedWrongModality)
	}
	if res.Stats.ActorsAggregated != 3 {
		t.Errorf("ActorsAggregated: got %d want 3", res.Stats.ActorsAggregated)
	}
	if res.Stats.ActorsAboveThreshold != 3 {
		t.Errorf("ActorsAboveThreshold: got %d want 3", res.Stats.ActorsAboveThreshold)
	}
	if res.Stats.PerCollector["cic-ids-2017-adapter:v1"] != 9 {
		t.Errorf("PerCollector[cic-ids]: got %d want 9", res.Stats.PerCollector["cic-ids-2017-adapter:v1"])
	}

	t.Logf("CIC-IDS full loop: ingested=%d obs; derived=%d Cat II; candidates=%d (cluster of %d derived actors)",
		report.ObservationsCommitted, derivedReport.NewlyDerived, len(res.Candidates), len(c.ActorRefs))
}
