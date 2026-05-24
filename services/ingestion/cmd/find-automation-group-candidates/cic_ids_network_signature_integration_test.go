// Integration test exercising tcp_fingerprint_clustering_v1 (§0161)
// against CIC-IDS adapter output + synthetic NetworkObservation
// fixtures per §0162. Empirically documents TWO distinct §0144(e)
// phenomenon-vs-record gaps for the CIC-IDS → tcp_fingerprint path:
//
//   1. CIC-IDS adapter does NOT populate actor_ref on its
//      NetworkObservation records (CIC-IDS data is flow-level, not
//      actor-attributed). All CIC-IDS records skipped first at the
//      no-actor check → ObservationsSkippedNoActor.
//   2. EVEN IF actor_ref were populated, CIC-IDS via CICFlowMeter
//      does NOT preserve TCP options → adapter emits tcp_fingerprint
//      observations WITHOUT p0f_signature → would be skipped at the
//      empty-p0f check → ObservationsSkippedWrongModality.
//
// The integration test surfaces gap (1) empirically; gap (2) is
// surfaced via unit tests in signatures/tcp_fingerprint_clustering_test.go.
//
// Synthetic NetworkObservation records with non-empty p0f_signature
// + multiple actors sharing the same signature → the clustering
// signature emits the expected candidate, validating that the
// signature itself works correctly (the gap is at the adapter layer,
// not the signature layer).
//
// Corrects the §0161 over-optimistic claim that "CIC-IDS adversarial
// path is now F3-reachable" — empirically, CIC-IDS path is NOT F3-
// reachable for the clustering signature; reachability requires an
// adapter that preserves the canonical p0f form (e.g., a future
// adapter ingesting raw pcap + running p0f-style fingerprinting, or
// a CICFlowMeter revision that preserves TCP options).
//
// Per §0154 MO4 diagnostic discipline: the integration test surfaces
// the gap via ObservationsSkippedWrongModality count, not as silent
// emptiness — operator can distinguish "modality present but missing
// canonical form" from "modality absent" by inspecting the skip
// counter against the substrate's tcp_fingerprint record count.
package main

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/adapters/cic_ids"
	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/signatures"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

func newNetworkSignatureSubstrate(t *testing.T) (*substrate.Substrate, *ingest.Ingester) {
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

// collectNetworkObservations walks the substrate and unmarshals every
// committed NetworkObservation record. Local to this test package;
// mirrors collectBrowserObservations in main.go on the network side.
// Future advance: lift to a shared package helper when a network-
// modality orchestrator CLI lands.
func collectNetworkObservations(ctx context.Context, sub *substrate.Substrate) ([]*eventsv1.NetworkObservation, error) {
	const networkObservationMessageType = "ghosttrace.events.v1.NetworkObservation"
	var out []*eventsv1.NetworkObservation
	err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType != networkObservationMessageType {
			return nil
		}
		payload, err := sub.ReadBlob(ctx, row.EventHash)
		if err != nil {
			return err
		}
		obs := &eventsv1.NetworkObservation{}
		if err := proto.Unmarshal(payload, obs); err != nil {
			return err
		}
		out = append(out, obs)
		return nil
	})
	return out, err
}

// appendNetworkObservationWithP0F commits a NetworkObservation with a
// populated p0f_signature via the Ingester (paired with IngestionEvent
// per §0038). Helper for synthetic-fixture injection in this test.
func appendNetworkObservationWithP0F(t *testing.T, in *ingest.Ingester, actorRef string, p0fSignature string, collectorRef string) {
	t.Helper()
	obs := &eventsv1.NetworkObservation{
		ObservedAt:          1716120000000000000,
		ActorRef:            actorRef,
		EndpointRef:         "198.51.100.5:443",
		CollectorRef:        collectorRef,
		AuthenticationClass: commonv1.AuthenticationClass_AUTHENTICATION_CLASS_SERVER_AUTHENTICATED,
		Modality: &eventsv1.NetworkObservation_TcpFingerprint{
			TcpFingerprint: &eventsv1.NetworkTcpFingerprint{
				P0FSignature: p0fSignature,
				WindowSize:   65535,
				Mss:          1460,
				Ttl:          64,
			},
		},
	}
	_, err := in.Append(context.Background(), obs, obs.ObservedAt, ingest.Envelope{Channel: "test"})
	if err != nil {
		t.Fatalf("Append NetworkObservation actor=%q: %v", actorRef, err)
	}
}

// cicIDSNetworkSampleCSV mirrors the CIC-IDS CSV columns used by the
// adapter; 3 TCP rows ensures 3 tcp_fingerprint observations land in
// substrate. Identical structure to sampleCICIDSCSV in
// cic_ids_integration_test.go (kept inline to avoid cross-test-file
// coupling).
const cicIDSNetworkSampleCSV = `Flow ID,Source IP,Source Port,Destination IP,Destination Port,Protocol,Timestamp,Flow Duration,FIN Flag Count,SYN Flag Count,RST Flag Count,PSH Flag Count,ACK Flag Count,URG Flag Count,CWE Flag Count,ECE Flag Count,Fwd Header Length,Bwd Header Length,Init_Win_bytes_forward,Init_Win_bytes_backward,Label
192.0.2.10-198.51.100.5-49152-443-6,192.0.2.10,49152,198.51.100.5,443,6,03/07/2017 09:15,12345,1,1,0,3,8,0,0,0,32,32,65535,29200,BENIGN
192.0.2.11-198.51.100.6-53124-80-6,192.0.2.11,53124,198.51.100.6,80,6,03/07/2017 09:16,67890,1,1,0,0,4,0,0,0,32,32,29200,29200,BENIGN
192.0.2.13-198.51.100.8-49300-443-6,192.0.2.13,49300,198.51.100.8,443,6,03/07/2017 09:18,98765,2,3,1,5,12,0,0,0,32,32,65535,29200,DoS-Slowloris
`

// TestTCPFingerprintClustering_AgainstCICIDS_NoActorSkipped exercises
// tcp_fingerprint_clustering_v1 against substrate populated by the
// CIC-IDS adapter. CIC-IDS adapter does NOT populate actor_ref on its
// NetworkObservation records (the dataset is flow-level, not actor-
// attributed). The signature skips ALL CIC-IDS records at the
// no-actor check before reaching the modality dispatch.
//
// Documents the §0144(e) phenomenon-vs-record gap empirically:
// the substrate IS populated (9 NetworkObservation records present),
// but the signature emits 0 candidates because the records lack
// actor_ref (required for any per-actor hypothesis formation).
func TestTCPFingerprintClustering_AgainstCICIDS_NoActorSkipped(t *testing.T) {
	sub, in := newNetworkSignatureSubstrate(t)
	ctx := context.Background()

	// Step 1: Ingest CIC-IDS sample → produces 9 NetworkObservation
	// records (3 rows × 3 obs per row: 2 ip_asn + 1 tcp_fingerprint).
	report, err := cic_ids.Ingest(ctx, in, bytes.NewReader([]byte(cicIDSNetworkSampleCSV)),
		"cic-ids-2017-adapter:v1", ingest.Envelope{Channel: "test"})
	if err != nil {
		t.Fatalf("cic_ids.Ingest: %v", err)
	}
	if report.TcpFingerprintEmitted != 3 {
		t.Fatalf("TcpFingerprintEmitted: got %d want 3", report.TcpFingerprintEmitted)
	}

	// Step 2: Collect NetworkObservation records from substrate.
	observations, err := collectNetworkObservations(ctx, sub)
	if err != nil {
		t.Fatalf("collectNetworkObservations: %v", err)
	}
	if len(observations) != 9 {
		t.Fatalf("NetworkObservation count: got %d want 9 (6 ip_asn + 3 tcp_fingerprint)", len(observations))
	}

	// Step 3: Run tcp_fingerprint_clustering_v1 against the
	// CIC-IDS-derived observations.
	sig := &signatures.TCPFingerprintClusteringV1{}
	res, err := sig.EvaluateNetwork(ctx, observations)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}

	// §0144(e) phenomenon-vs-record gap empirical witness (gap #1 —
	// no actor_ref):
	//   - ObservationsScanned == 9 (all NetworkObservations seen).
	//   - ObservationsSkippedNoActor == 9 (CIC-IDS adapter does NOT
	//     populate actor_ref; all records skipped at the no-actor
	//     check before reaching modality dispatch).
	//   - ObservationsSkippedWrongModality == 0 (the no-actor skip
	//     happens earlier in the loop).
	//   - Candidates == 0.
	//
	// The CIC-IDS adapter (cic_ids.go:185-228) explicitly emits
	// NetworkObservation without setting ActorRef — the dataset is
	// flow-level, not actor-attributed. This is gap (1) per §0162.
	// Gap (2) — empty p0f_signature — is independently observable
	// via the signatures package unit tests; both gaps would need to
	// close before CIC-IDS path is genuinely F3-reachable for this
	// signature.
	if len(res.Candidates) != 0 {
		t.Errorf("Candidates: got %d want 0 (no actor_ref → no per-actor aggregation)", len(res.Candidates))
	}
	if res.Stats.ObservationsScanned != 9 {
		t.Errorf("ObservationsScanned: got %d want 9", res.Stats.ObservationsScanned)
	}
	if res.Stats.ObservationsSkippedNoActor != 9 {
		t.Errorf("ObservationsSkippedNoActor: got %d want 9 (CIC-IDS records lack actor_ref)", res.Stats.ObservationsSkippedNoActor)
	}
	if res.Stats.ObservationsSkippedWrongModality != 0 {
		t.Errorf("ObservationsSkippedWrongModality: got %d want 0 (no-actor skip happens first)", res.Stats.ObservationsSkippedWrongModality)
	}
	if res.Stats.ActorsAggregated != 0 {
		t.Errorf("ActorsAggregated: got %d want 0", res.Stats.ActorsAggregated)
	}
	if res.Stats.PerCollector["cic-ids-2017-adapter:v1"] != 9 {
		t.Errorf("PerCollector[cic-ids]: got %d want 9", res.Stats.PerCollector["cic-ids-2017-adapter:v1"])
	}
}

// TestTCPFingerprintClustering_AgainstSyntheticP0F_ClusterEmitted
// exercises tcp_fingerprint_clustering_v1 against synthetic
// NetworkObservation fixtures with populated p0f_signature. Validates
// the signature itself works correctly when the canonical form is
// present — the §0162 gap is at the adapter layer (CIC-IDS), NOT at
// the signature layer.
func TestTCPFingerprintClustering_AgainstSyntheticP0F_ClusterEmitted(t *testing.T) {
	sub, in := newNetworkSignatureSubstrate(t)
	ctx := context.Background()

	// Step 1: Inject 3 synthetic NetworkObservation records with same
	// p0f_signature (meets default threshold = 3).
	p0f := "4:64:0:1460:mss*44,7:mss,sok,ts,nop,ws:df:0"
	appendNetworkObservationWithP0F(t, in, "actor-bot-1", p0f, "synthetic-collector:v1")
	appendNetworkObservationWithP0F(t, in, "actor-bot-2", p0f, "synthetic-collector:v1")
	appendNetworkObservationWithP0F(t, in, "actor-bot-3", p0f, "synthetic-collector:v1")

	// Step 2: Collect + evaluate.
	observations, err := collectNetworkObservations(ctx, sub)
	if err != nil {
		t.Fatalf("collectNetworkObservations: %v", err)
	}
	if len(observations) != 3 {
		t.Fatalf("NetworkObservation count: got %d want 3", len(observations))
	}
	sig := &signatures.TCPFingerprintClusteringV1{}
	res, err := sig.EvaluateNetwork(ctx, observations)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}

	// Verify the cluster was detected.
	if len(res.Candidates) != 1 {
		t.Fatalf("Candidates: got %d want 1", len(res.Candidates))
	}
	c := res.Candidates[0]
	if c.SignatureName != "tcp_fingerprint_clustering_v1" {
		t.Errorf("SignatureName: got %q", c.SignatureName)
	}
	if len(c.ActorRefs) != 3 {
		t.Errorf("ActorRefs count: got %d want 3", len(c.ActorRefs))
	}
	if res.Stats.ObservationsSkippedWrongModality != 0 {
		t.Errorf("ObservationsSkippedWrongModality: got %d want 0", res.Stats.ObservationsSkippedWrongModality)
	}
	if res.Stats.ActorsAboveThreshold != 3 {
		t.Errorf("ActorsAboveThreshold: got %d want 3", res.Stats.ActorsAboveThreshold)
	}
}

// TestTCPFingerprintClustering_MixedSubstrate_OnlySyntheticContributes
// validates the diagnostic distinction at substrate-level: when a
// substrate contains BOTH CIC-IDS-derived observations (empty p0f) AND
// synthetic observations (populated p0f), the signature isolates the
// p0f-populated cluster correctly + the skip counters reflect the
// CIC-IDS observations exactly.
//
// This is the §0154 MO4 + §0155 two-layer diagnostic separation
// applied to the tcp_fingerprint modality: substrate-walk produces
// the full observation set; signature-layer skip counters distinguish
// the contributing records from the skipped ones.
func TestTCPFingerprintClustering_MixedSubstrate_OnlySyntheticContributes(t *testing.T) {
	sub, in := newNetworkSignatureSubstrate(t)
	ctx := context.Background()

	// Step 1: Ingest CIC-IDS → 9 NetworkObservation (3 tcp_fingerprint
	// without p0f + 6 ip_asn).
	_, err := cic_ids.Ingest(ctx, in, bytes.NewReader([]byte(cicIDSNetworkSampleCSV)),
		"cic-ids-2017-adapter:v1", ingest.Envelope{Channel: "test"})
	if err != nil {
		t.Fatalf("cic_ids.Ingest: %v", err)
	}

	// Step 2: Inject 3 synthetic NetworkObservation with populated p0f.
	p0f := "4:64:0:1460:mss*44,7:mss,sok,ts,nop,ws:df:0"
	appendNetworkObservationWithP0F(t, in, "actor-x", p0f, "synthetic-collector:v1")
	appendNetworkObservationWithP0F(t, in, "actor-y", p0f, "synthetic-collector:v1")
	appendNetworkObservationWithP0F(t, in, "actor-z", p0f, "synthetic-collector:v1")

	// Step 3: Collect + evaluate.
	observations, err := collectNetworkObservations(ctx, sub)
	if err != nil {
		t.Fatalf("collectNetworkObservations: %v", err)
	}
	if len(observations) != 12 {
		t.Fatalf("NetworkObservation count: got %d want 12 (9 CIC-IDS + 3 synthetic)", len(observations))
	}
	sig := &signatures.TCPFingerprintClusteringV1{}
	res, err := sig.EvaluateNetwork(ctx, observations)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}

	// Diagnostic invariant: signature isolates the contributing
	// records (3 synthetic with p0f) from the skipped ones (9 CIC-IDS
	// without p0f).
	if len(res.Candidates) != 1 {
		t.Fatalf("Candidates: got %d want 1 (synthetic cluster only)", len(res.Candidates))
	}
	if res.Stats.ObservationsScanned != 12 {
		t.Errorf("ObservationsScanned: got %d want 12", res.Stats.ObservationsScanned)
	}
	if res.Stats.ObservationsSkippedNoActor != 9 {
		t.Errorf("ObservationsSkippedNoActor: got %d want 9 (CIC-IDS records lack actor_ref)", res.Stats.ObservationsSkippedNoActor)
	}
	if res.Stats.ObservationsSkippedWrongModality != 0 {
		t.Errorf("ObservationsSkippedWrongModality: got %d want 0 (no-actor skip happens first)", res.Stats.ObservationsSkippedWrongModality)
	}
	if res.Stats.ActorsAggregated != 3 {
		t.Errorf("ActorsAggregated: got %d want 3 (synthetic actors)", res.Stats.ActorsAggregated)
	}
	if res.Stats.ActorsAboveThreshold != 3 {
		t.Errorf("ActorsAboveThreshold: got %d want 3", res.Stats.ActorsAboveThreshold)
	}
	// Per-collector breakdown surfaces both sources.
	if res.Stats.PerCollector["cic-ids-2017-adapter:v1"] != 9 {
		t.Errorf("PerCollector[cic-ids]: got %d want 9", res.Stats.PerCollector["cic-ids-2017-adapter:v1"])
	}
	if res.Stats.PerCollector["synthetic-collector:v1"] != 3 {
		t.Errorf("PerCollector[synthetic]: got %d want 3", res.Stats.PerCollector["synthetic-collector:v1"])
	}

	c := res.Candidates[0]
	if len(c.ActorRefs) != 3 {
		t.Errorf("candidate ActorRefs count: got %d want 3", len(c.ActorRefs))
	}
	for _, a := range c.ActorRefs {
		if a != "actor-x" && a != "actor-y" && a != "actor-z" {
			t.Errorf("unexpected actor %q in candidate (CIC-IDS actors should not appear; their tcp_fingerprint lacks p0f)", a)
		}
	}
}
