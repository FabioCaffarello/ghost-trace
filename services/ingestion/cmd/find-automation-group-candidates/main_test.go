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

// newTestSubstrate opens a temp substrate + Ingester for end-to-end
// test scenarios. Mirrors the pattern from services/ingestion/internal/
// ingest/ingest_test.go newIngester().
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

func appendBrowserObs(t *testing.T, in *ingest.Ingester, actorRef string, markers []string, detectionCount uint32) {
	t.Helper()
	obs := &eventsv1.BrowserObservation{
		ObservedAt:          1716120000000000000,
		ActorRef:            actorRef,
		CollectorRef:        "browser-sdk:v1",
		AuthenticationClass: commonv1.AuthenticationClass_AUTHENTICATION_CLASS_CLIENT_WITNESSED,
		Modality: &eventsv1.BrowserObservation_CdpMarker{
			CdpMarker: &eventsv1.BrowserCDPMarker{
				MarkersObserved: markers,
				DetectionCount:  detectionCount,
			},
		},
	}
	_, err := in.Append(context.Background(), obs, obs.ObservedAt, ingest.Envelope{Channel: "test"})
	if err != nil {
		t.Fatalf("Append BrowserObservation actor=%q: %v", actorRef, err)
	}
}

// TestCollectBrowserObservations_EndToEnd commits multiple
// BrowserObservation records to a temp substrate via the Ingester
// path, then verifies collectBrowserObservations walks the substrate
// + retrieves all of them (and only BrowserObservation records,
// not the paired IngestionEvent records the Ingester also commits).
func TestCollectBrowserObservations_EndToEnd(t *testing.T) {
	sub, in := newTestSubstrate(t)
	appendBrowserObs(t, in, "actor-a", []string{"navigator.webdriver=true"}, 3)
	appendBrowserObs(t, in, "actor-b", []string{"$cdc_test"}, 1)
	appendBrowserObs(t, in, "actor-a", []string{"missing-chrome.csi"}, 2)

	observations, err := collectBrowserObservations(context.Background(), sub)
	if err != nil {
		t.Fatalf("collectBrowserObservations: %v", err)
	}
	if got, want := len(observations), 3; got != want {
		t.Fatalf("expected %d BrowserObservation records, got %d (note: IngestionEvent records must be excluded)", want, got)
	}
}

// TestFullPipeline_EndToEnd exercises the complete orchestrator path:
// commit observations → walk substrate → invoke signature → emit JSON.
// Verifies the JSON output shape + values match expectations across
// the threshold boundary.
func TestFullPipeline_EndToEnd(t *testing.T) {
	sub, in := newTestSubstrate(t)
	// actor-a: 3 + 2 = 5 detections → above threshold → 1 candidate
	appendBrowserObs(t, in, "actor-a", []string{"navigator.webdriver=true"}, 3)
	appendBrowserObs(t, in, "actor-a", []string{"missing-chrome.csi"}, 2)
	// actor-b: 1 detection → below threshold → 0 candidates
	appendBrowserObs(t, in, "actor-b", []string{"$cdc_test"}, 1)

	observations, err := collectBrowserObservations(context.Background(), sub)
	if err != nil {
		t.Fatalf("collectBrowserObservations: %v", err)
	}

	sig := &signatures.CDPMarkerDensityV1{}
	result, err := sig.EvaluateBrowser(context.Background(), observations)
	if err != nil {
		t.Fatalf("EvaluateBrowser: %v", err)
	}
	candidates := result.Candidates
	if got, want := len(candidates), 1; got != want {
		t.Fatalf("candidates: got %d want %d", got, want)
	}
	// Stats sanity checks (telemetry surface per §0154).
	if result.Stats.ObservationsScanned != uint32(len(observations)) {
		t.Errorf("Stats.ObservationsScanned: got %d want %d", result.Stats.ObservationsScanned, len(observations))
	}
	if result.Stats.ActorsAboveThreshold != 1 {
		t.Errorf("Stats.ActorsAboveThreshold: got %d want 1", result.Stats.ActorsAboveThreshold)
	}
	if result.Stats.CandidatesEmitted != 1 {
		t.Errorf("Stats.CandidatesEmitted: got %d want 1", result.Stats.CandidatesEmitted)
	}
	if got := result.Stats.PerCollector["browser-sdk:v1"]; got != uint32(len(observations)) {
		t.Errorf("Stats.PerCollector[browser-sdk:v1]: got %d want %d", got, len(observations))
	}

	tmp, err := os.CreateTemp("", "find-candidates-*.json")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer os.Remove(tmp.Name())
	if err := emitCandidatesJSON(tmp, sig.Name(), candidates, result.Stats); err != nil {
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

	if got.SignatureName != "cdp_marker_density_v1" {
		t.Errorf("envelope SignatureName: got %q want cdp_marker_density_v1", got.SignatureName)
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
	if len(c.ActorRefs) != 1 || c.ActorRefs[0] != "actor-a" {
		t.Errorf("ActorRefs: got %v want [actor-a]", c.ActorRefs)
	}
	if c.EvidenceCount != 2 {
		t.Errorf("EvidenceCount: got %d want 2 (two BrowserObservation records for actor-a)", c.EvidenceCount)
	}
	if len(c.SourceHashesHex) != 2 {
		t.Errorf("SourceHashesHex count: got %d want 2", len(c.SourceHashesHex))
	}
	for i, h := range c.SourceHashesHex {
		if len(h) != 64 {
			t.Errorf("SourceHashesHex[%d]: got len %d want 64 (BLAKE3-256 hex)", i, len(h))
		}
		if _, err := hex.DecodeString(h); err != nil {
			t.Errorf("SourceHashesHex[%d] not hex: %v", i, err)
		}
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
