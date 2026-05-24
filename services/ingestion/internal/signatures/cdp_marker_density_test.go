package signatures

import (
	"bytes"
	"context"
	"sort"
	"testing"

	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
)

func newBrowserObservationWithCDPMarker(actorRef string, detectionCount uint32, markers []string) *eventsv1.BrowserObservation {
	return &eventsv1.BrowserObservation{
		ObservedAt:          1716120000000000000,
		ActorRef:            actorRef,
		SessionRef:          "session-test",
		CollectorRef:        "browser-sdk:v1",
		AuthenticationClass: commonv1.AuthenticationClass_AUTHENTICATION_CLASS_CLIENT_WITNESSED,
		Modality: &eventsv1.BrowserObservation_CdpMarker{
			CdpMarker: &eventsv1.BrowserCDPMarker{
				MarkersObserved: markers,
				DetectionCount:  detectionCount,
			},
		},
	}
}

func TestCDPMarkerDensityV1_BelowThreshold_NoCandidate(t *testing.T) {
	sig := &CDPMarkerDensityV1{}
	obs := newBrowserObservationWithCDPMarker("actor-a", 1, []string{"navigator.webdriver=true"})
	cands, err := sig.EvaluateBrowser(context.Background(), []*eventsv1.BrowserObservation{obs})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if len(cands) != 0 {
		t.Fatalf("expected 0 candidates, got %d", len(cands))
	}
}

func TestCDPMarkerDensityV1_AtThreshold_OneCandidate(t *testing.T) {
	sig := &CDPMarkerDensityV1{}
	obs := newBrowserObservationWithCDPMarker("actor-a", 2, []string{"navigator.webdriver=true", "$cdc_test"})
	cands, err := sig.EvaluateBrowser(context.Background(), []*eventsv1.BrowserObservation{obs})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if len(cands) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(cands))
	}
	c := cands[0]
	if c.SignatureName != "cdp_marker_density_v1" {
		t.Errorf("SignatureName: got %q want cdp_marker_density_v1", c.SignatureName)
	}
	if c.HypothesisSubtype != HypothesisSubtypeAutomationGroup {
		t.Errorf("HypothesisSubtype: got %v want AutomationGroup", c.HypothesisSubtype)
	}
	if len(c.ActorRefs) != 1 || c.ActorRefs[0] != "actor-a" {
		t.Errorf("ActorRefs: got %v want [actor-a]", c.ActorRefs)
	}
	if c.EvidenceCount != 1 {
		t.Errorf("EvidenceCount: got %d want 1", c.EvidenceCount)
	}
	if len(c.SourceHashes) != 1 || len(c.SourceHashes[0]) != 32 {
		t.Errorf("SourceHashes: got %d hashes (first len %d); want 1 hash of 32 bytes", len(c.SourceHashes), len(c.SourceHashes[0]))
	}
	if c.ConfidenceHint < 0.5 || c.ConfidenceHint > 0.9 {
		t.Errorf("ConfidenceHint: got %v; want in [0.5, 0.9]", c.ConfidenceHint)
	}
}

func TestCDPMarkerDensityV1_MultipleActorsAboveThreshold_MultipleCandidates(t *testing.T) {
	sig := &CDPMarkerDensityV1{}
	observations := []*eventsv1.BrowserObservation{
		newBrowserObservationWithCDPMarker("actor-b", 3, []string{"navigator.webdriver=true"}),
		newBrowserObservationWithCDPMarker("actor-a", 5, []string{"$cdc_test"}),
		newBrowserObservationWithCDPMarker("actor-c", 1, []string{"navigator.webdriver=true"}), // below threshold
	}
	cands, err := sig.EvaluateBrowser(context.Background(), observations)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	// Expect 2 candidates (actor-a + actor-b); actor-c below threshold.
	if len(cands) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(cands))
	}
	// Deterministic ordering: actor-a then actor-b (alphabetical).
	if cands[0].ActorRefs[0] != "actor-a" {
		t.Errorf("cands[0].ActorRefs[0]: got %q want actor-a (alphabetical order)", cands[0].ActorRefs[0])
	}
	if cands[1].ActorRefs[0] != "actor-b" {
		t.Errorf("cands[1].ActorRefs[0]: got %q want actor-b", cands[1].ActorRefs[0])
	}
	// actor-a has higher detection count → higher confidence
	if cands[0].ConfidenceHint <= cands[1].ConfidenceHint {
		t.Errorf("expected actor-a (count=5) confidence > actor-b (count=3); got %v vs %v", cands[0].ConfidenceHint, cands[1].ConfidenceHint)
	}
}

func TestCDPMarkerDensityV1_AggregatesAcrossWindow(t *testing.T) {
	sig := &CDPMarkerDensityV1{}
	// Same actor across 3 observations; counts sum: 1 + 1 + 1 = 3, above threshold.
	observations := []*eventsv1.BrowserObservation{
		newBrowserObservationWithCDPMarker("actor-x", 1, []string{"m1"}),
		newBrowserObservationWithCDPMarker("actor-x", 1, []string{"m2"}),
		newBrowserObservationWithCDPMarker("actor-x", 1, []string{"m3"}),
	}
	cands, err := sig.EvaluateBrowser(context.Background(), observations)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if len(cands) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(cands))
	}
	if cands[0].EvidenceCount != 3 {
		t.Errorf("EvidenceCount: got %d want 3 (aggregated across window)", cands[0].EvidenceCount)
	}
	if len(cands[0].SourceHashes) != 3 {
		t.Errorf("SourceHashes count: got %d want 3", len(cands[0].SourceHashes))
	}
}

func TestCDPMarkerDensityV1_SourceHashesAscendingOrder(t *testing.T) {
	sig := &CDPMarkerDensityV1{}
	// Multiple observations for one actor; verify SourceHashes are
	// sorted ascending per §0139 hash-list element-shape discipline.
	observations := []*eventsv1.BrowserObservation{
		newBrowserObservationWithCDPMarker("actor-x", 2, []string{"a"}),
		newBrowserObservationWithCDPMarker("actor-x", 2, []string{"b"}),
		newBrowserObservationWithCDPMarker("actor-x", 2, []string{"c"}),
		newBrowserObservationWithCDPMarker("actor-x", 2, []string{"d"}),
	}
	cands, err := sig.EvaluateBrowser(context.Background(), observations)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if len(cands) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(cands))
	}
	hashes := cands[0].SourceHashes
	if !sort.SliceIsSorted(hashes, func(i, j int) bool { return bytes.Compare(hashes[i], hashes[j]) < 0 }) {
		t.Errorf("SourceHashes not sorted ascending per §0139")
	}
}

func TestCDPMarkerDensityV1_SkipsUnattributedObservations(t *testing.T) {
	sig := &CDPMarkerDensityV1{}
	// Empty actor_ref → no anchor for AutomationGroup; observation skipped.
	obs := newBrowserObservationWithCDPMarker("", 5, []string{"m1"})
	cands, err := sig.EvaluateBrowser(context.Background(), []*eventsv1.BrowserObservation{obs})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if len(cands) != 0 {
		t.Fatalf("expected 0 candidates (unattributed observation), got %d", len(cands))
	}
}

func TestCDPMarkerDensityV1_SkipsNonCDPModalities(t *testing.T) {
	sig := &CDPMarkerDensityV1{}
	// canvas_fingerprint modality — out of scope for this signature.
	obs := &eventsv1.BrowserObservation{
		ObservedAt:   1716120000000000000,
		ActorRef:     "actor-x",
		CollectorRef: "browser-sdk:v1",
		Modality: &eventsv1.BrowserObservation_CanvasFingerprint{
			CanvasFingerprint: &eventsv1.BrowserCanvasFingerprint{
				CanvasHash: bytes.Repeat([]byte{0x01}, 32),
			},
		},
	}
	cands, err := sig.EvaluateBrowser(context.Background(), []*eventsv1.BrowserObservation{obs})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if len(cands) != 0 {
		t.Fatalf("expected 0 candidates (canvas_fingerprint out of scope), got %d", len(cands))
	}
}

func TestCDPMarkerDensityV1_CustomThreshold(t *testing.T) {
	sig := &CDPMarkerDensityV1{Threshold: 10}
	// Below custom threshold (default would emit since 5 >= 2).
	obs := newBrowserObservationWithCDPMarker("actor-x", 5, []string{"m1"})
	cands, err := sig.EvaluateBrowser(context.Background(), []*eventsv1.BrowserObservation{obs})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if len(cands) != 0 {
		t.Fatalf("expected 0 candidates (5 < custom threshold 10), got %d", len(cands))
	}
}

func TestCDPMarkerDensityV1_ContextCancellation(t *testing.T) {
	sig := &CDPMarkerDensityV1{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := sig.EvaluateBrowser(ctx, nil)
	if err == nil {
		t.Fatalf("expected ctx.Err() on canceled context, got nil")
	}
}

func TestCDPMarkerDensityV1_ConfidenceHintCappedAt09(t *testing.T) {
	sig := &CDPMarkerDensityV1{}
	// Very high detection count → confidence should be capped at 0.9
	// per §3 N1 (substrate never asserts certainty).
	obs := newBrowserObservationWithCDPMarker("actor-x", 100, []string{"m1"})
	cands, err := sig.EvaluateBrowser(context.Background(), []*eventsv1.BrowserObservation{obs})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if len(cands) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(cands))
	}
	if cands[0].ConfidenceHint != 0.9 {
		t.Errorf("ConfidenceHint not capped at 0.9: got %v", cands[0].ConfidenceHint)
	}
}
