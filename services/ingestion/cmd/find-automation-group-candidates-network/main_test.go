package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/signatures"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

func newTestSubstrate(t *testing.T) (*substrate.Substrate, *ingest.Ingester) {
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

func appendNetworkObsWithP0F(t *testing.T, in *ingest.Ingester, actorRef, p0fSignature string) {
	t.Helper()
	obs := &eventsv1.NetworkObservation{
		ObservedAt:          1716120000000000000,
		ActorRef:            actorRef,
		EndpointRef:         "198.51.100.5:443",
		CollectorRef:        "test-collector:v1",
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

// TestCollectNetworkObservations_EndToEnd commits multiple
// NetworkObservation records via Ingester, then verifies the
// substrate walk retrieves all of them (and only NetworkObservation,
// excluding the paired IngestionEvent records).
func TestCollectNetworkObservations_EndToEnd(t *testing.T) {
	sub, in := newTestSubstrate(t)
	p0f := "4:64:0:1460:mss*44,7:mss,sok,ts,nop,ws:df:0"
	appendNetworkObsWithP0F(t, in, "actor-a", p0f)
	appendNetworkObsWithP0F(t, in, "actor-b", p0f)
	appendNetworkObsWithP0F(t, in, "actor-c", p0f)

	observations, err := collectNetworkObservations(context.Background(), sub)
	if err != nil {
		t.Fatalf("collectNetworkObservations: %v", err)
	}
	if got, want := len(observations), 3; got != want {
		t.Fatalf("expected %d NetworkObservation records, got %d (note: IngestionEvent records must be excluded)", want, got)
	}
}

// TestFullPipeline_EndToEnd exercises the complete network-modality
// orchestrator path: commit observations → walk substrate → invoke
// tcp_fingerprint_clustering_v1 → emit JSON. Verifies the JSON
// output shape + values match expectations.
func TestFullPipeline_EndToEnd(t *testing.T) {
	sub, in := newTestSubstrate(t)
	p0f := "4:64:0:1460:mss*44,7:mss,sok,ts,nop,ws:df:0"
	// 3 actors with same p0f → cluster meets default threshold 3.
	appendNetworkObsWithP0F(t, in, "actor-bot-1", p0f)
	appendNetworkObsWithP0F(t, in, "actor-bot-2", p0f)
	appendNetworkObsWithP0F(t, in, "actor-bot-3", p0f)

	observations, err := collectNetworkObservations(context.Background(), sub)
	if err != nil {
		t.Fatalf("collectNetworkObservations: %v", err)
	}
	sig := &signatures.TCPFingerprintClusteringV1{}
	result, err := sig.EvaluateNetwork(context.Background(), observations)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("candidates: got %d want 1", len(result.Candidates))
	}
	// Stats sanity per §0143 + §0154.
	if result.Stats.ObservationsScanned != 3 {
		t.Errorf("ObservationsScanned: got %d want 3", result.Stats.ObservationsScanned)
	}
	if result.Stats.ActorsAggregated != 3 {
		t.Errorf("ActorsAggregated: got %d want 3", result.Stats.ActorsAggregated)
	}
	if result.Stats.ActorsAboveThreshold != 3 {
		t.Errorf("ActorsAboveThreshold: got %d want 3", result.Stats.ActorsAboveThreshold)
	}
	if result.Stats.CandidatesEmitted != 1 {
		t.Errorf("CandidatesEmitted: got %d want 1", result.Stats.CandidatesEmitted)
	}
	if got := result.Stats.PerCollector["test-collector:v1"]; got != 3 {
		t.Errorf("PerCollector[test-collector:v1]: got %d want 3", got)
	}

	tmp, err := os.CreateTemp("", "find-net-candidates-*.json")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer os.Remove(tmp.Name())
	if err := emitCandidatesJSON(tmp, sig.Name(), result.Candidates, result.Stats); err != nil {
		t.Fatalf("emitCandidatesJSON: %v", err)
	}
	tmp.Close()

	raw, err := os.ReadFile(tmp.Name())
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var got emissionEnvelope
	if err := json.NewDecoder(bytes.NewReader(raw)).Decode(&got); err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if got.SignatureName != "tcp_fingerprint_clustering_v1" {
		t.Errorf("envelope SignatureName: got %q want tcp_fingerprint_clustering_v1", got.SignatureName)
	}
	if got.CandidateCount != 1 {
		t.Errorf("CandidateCount: got %d want 1", got.CandidateCount)
	}
	if len(got.Candidates) != 1 {
		t.Fatalf("Candidates len: got %d want 1", len(got.Candidates))
	}
	c := got.Candidates[0]
	if c.HypothesisSubtype != "AutomationGroup" {
		t.Errorf("HypothesisSubtype: got %q want AutomationGroup", c.HypothesisSubtype)
	}
	if len(c.ActorRefs) != 3 {
		t.Errorf("ActorRefs count: got %d want 3", len(c.ActorRefs))
	}
	for i, h := range c.SourceHashesHex {
		if len(h) != 64 {
			t.Errorf("SourceHashesHex[%d]: got len %d want 64 (BLAKE3-256 hex)", i, len(h))
		}
		if _, err := hex.DecodeString(h); err != nil {
			t.Errorf("SourceHashesHex[%d] not hex: %v", i, err)
		}
	}
	if got.Stats.ActorsAboveThreshold != 3 {
		t.Errorf("Stats.ActorsAboveThreshold: got %d want 3", got.Stats.ActorsAboveThreshold)
	}
}

// TestFullPipeline_BelowThreshold_NoCandidates verifies the empty-
// candidate case still emits a valid JSON envelope with populated
// diagnostic counters. Per §0154 MO4: non-firing is INFORMATIVE.
func TestFullPipeline_BelowThreshold_NoCandidates(t *testing.T) {
	sub, in := newTestSubstrate(t)
	p0f := "4:64:0:1460:mss*44,7:mss,sok,ts,nop,ws:df:0"
	// 2 actors with same p0f → below threshold 3.
	appendNetworkObsWithP0F(t, in, "actor-a", p0f)
	appendNetworkObsWithP0F(t, in, "actor-b", p0f)

	observations, err := collectNetworkObservations(context.Background(), sub)
	if err != nil {
		t.Fatalf("collectNetworkObservations: %v", err)
	}
	sig := &signatures.TCPFingerprintClusteringV1{}
	result, err := sig.EvaluateNetwork(context.Background(), observations)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(result.Candidates) != 0 {
		t.Fatalf("expected 0 candidates (below threshold), got %d", len(result.Candidates))
	}

	tmp, err := os.CreateTemp("", "find-net-candidates-empty-*.json")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer os.Remove(tmp.Name())
	if err := emitCandidatesJSON(tmp, sig.Name(), result.Candidates, result.Stats); err != nil {
		t.Fatalf("emitCandidatesJSON: %v", err)
	}
	tmp.Close()
	raw, err := os.ReadFile(tmp.Name())
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var got emissionEnvelope
	if err := json.NewDecoder(bytes.NewReader(raw)).Decode(&got); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.CandidateCount != 0 {
		t.Errorf("CandidateCount: got %d want 0", got.CandidateCount)
	}
	if got.Stats.ActorsAggregated != 2 {
		t.Errorf("Stats.ActorsAggregated: got %d want 2 (both actors aggregated, but below threshold)", got.Stats.ActorsAggregated)
	}
	if got.Stats.ActorsAboveThreshold != 0 {
		t.Errorf("Stats.ActorsAboveThreshold: got %d want 0", got.Stats.ActorsAboveThreshold)
	}
}

// TestSubtypeName_AllValuesNamed verifies the subtypeName helper
// covers all enum values + the unknown fallback. Symmetric to the
// equivalent test in find-automation-group-candidates' main_test.go.
func TestSubtypeName_AllValuesNamed(t *testing.T) {
	cases := []struct {
		in   signatures.HypothesisSubtype
		want string
	}{
		{signatures.HypothesisSubtypeAutomationGroup, "AutomationGroup"},
		{signatures.HypothesisSubtypeBehavioralCluster, "BehavioralCluster"},
		{signatures.HypothesisSubtypeCampaignHypothesis, "CampaignHypothesis"},
		{signatures.HypothesisSubtypeCoordinationRing, "CoordinationRing"},
		{signatures.HypothesisSubtypeUnknown, "Unknown"},
	}
	for _, tc := range cases {
		if got := subtypeName(tc.in); got != tc.want {
			t.Errorf("subtypeName(%v): got %q want %q", tc.in, got, tc.want)
		}
	}
}
