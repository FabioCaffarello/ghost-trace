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
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/observationcollector"
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

func appendNetworkObs(t *testing.T, in *ingest.Ingester, actorRef, endpointRef string, observedAt int64) {
	t.Helper()
	obs := &eventsv1.NetworkObservation{
		ObservedAt:          observedAt,
		ActorRef:            actorRef,
		EndpointRef:         endpointRef,
		CollectorRef:        "test-collector:v1",
		AuthenticationClass: commonv1.AuthenticationClass_AUTHENTICATION_CLASS_SERVER_AUTHENTICATED,
		Modality: &eventsv1.NetworkObservation_TcpFingerprint{
			TcpFingerprint: &eventsv1.NetworkTcpFingerprint{WindowSize: 65535},
		},
	}
	_, err := in.Append(context.Background(), obs, obs.ObservedAt, ingest.Envelope{Channel: "test"})
	if err != nil {
		t.Fatalf("Append NetworkObservation actor=%q: %v", actorRef, err)
	}
}

func TestCollectNetworkObservations_EndToEnd(t *testing.T) {
	sub, in := newTestSubstrate(t)
	const bucketStart = int64(1716120000_000_000_000)
	appendNetworkObs(t, in, "actor-a", "10.0.0.1:443", bucketStart)
	appendNetworkObs(t, in, "actor-b", "10.0.0.1:443", bucketStart+10_000_000_000)
	appendNetworkObs(t, in, "actor-c", "10.0.0.1:443", bucketStart+20_000_000_000)

	observations, err := observationcollector.CollectNetwork(context.Background(), sub)
	if err != nil {
		t.Fatalf("observationcollector.CollectNetwork: %v", err)
	}
	if got, want := len(observations), 3; got != want {
		t.Fatalf("expected %d NetworkObservation records, got %d (IngestionEvent records must be excluded)", want, got)
	}
}

// TestFullPipeline_EndToEnd verifies CampaignHypothesis subtype
// emission in the JSON envelope's hypothesis_subtype field.
func TestFullPipeline_EndToEnd(t *testing.T) {
	sub, in := newTestSubstrate(t)
	const bucketStart = int64(1716120000_000_000_000)
	appendNetworkObs(t, in, "actor-bot-1", "10.0.0.1:443", bucketStart)
	appendNetworkObs(t, in, "actor-bot-2", "10.0.0.1:443", bucketStart+10_000_000_000)
	appendNetworkObs(t, in, "actor-bot-3", "10.0.0.1:443", bucketStart+20_000_000_000)

	observations, err := observationcollector.CollectNetwork(context.Background(), sub)
	if err != nil {
		t.Fatalf("observationcollector.CollectNetwork: %v", err)
	}
	sig := &signatures.TemporalEndpointCohortV1{}
	result, err := sig.EvaluateNetwork(context.Background(), observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("candidates: got %d want 1", len(result.Candidates))
	}
	if result.Stats.ObservationsScanned != 3 {
		t.Errorf("ObservationsScanned: got %d want 3", result.Stats.ObservationsScanned)
	}
	if result.Stats.CandidatesEmitted != 1 {
		t.Errorf("CandidatesEmitted: got %d want 1", result.Stats.CandidatesEmitted)
	}

	tmp, err := os.CreateTemp("", "find-campaign-candidates-*.json")
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

	if got.SignatureName != "temporal_endpoint_cohort_v1" {
		t.Errorf("envelope SignatureName: got %q", got.SignatureName)
	}
	if got.CandidateCount != 1 {
		t.Errorf("CandidateCount: got %d want 1", got.CandidateCount)
	}
	c := got.Candidates[0]
	if c.HypothesisSubtype != "CampaignHypothesis" {
		t.Errorf("HypothesisSubtype: got %q want CampaignHypothesis (first F3 emission of this subtype per §0182)", c.HypothesisSubtype)
	}
	for i, h := range c.SourceHashesHex {
		if len(h) != 64 {
			t.Errorf("SourceHashesHex[%d]: got len %d want 64", i, len(h))
		}
		if _, err := hex.DecodeString(h); err != nil {
			t.Errorf("SourceHashesHex[%d] not hex: %v", i, err)
		}
	}
}

// TestFullPipeline_BelowThreshold_NoCandidates verifies the empty-
// candidate case emits a valid JSON envelope with populated
// diagnostic counters per §0154 MO4 + §0163 MO3 4-test contract.
func TestFullPipeline_BelowThreshold_NoCandidates(t *testing.T) {
	sub, in := newTestSubstrate(t)
	const bucketStart = int64(1716120000_000_000_000)
	appendNetworkObs(t, in, "actor-a", "10.0.0.1:443", bucketStart)
	appendNetworkObs(t, in, "actor-b", "10.0.0.1:443", bucketStart+10_000_000_000)

	observations, err := observationcollector.CollectNetwork(context.Background(), sub)
	if err != nil {
		t.Fatalf("observationcollector.CollectNetwork: %v", err)
	}
	sig := &signatures.TemporalEndpointCohortV1{}
	result, err := sig.EvaluateNetwork(context.Background(), observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(result.Candidates) != 0 {
		t.Fatalf("expected 0 candidates (below threshold), got %d", len(result.Candidates))
	}

	tmp, err := os.CreateTemp("", "find-campaign-candidates-empty-*.json")
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
}

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
	for _, c := range cases {
		if got := subtypeName(c.in); got != c.want {
			t.Errorf("subtypeName(%v): got %q want %q", c.in, got, c.want)
		}
	}
}
