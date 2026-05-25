package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// substrateWithCDPMarkerObservations ingests BrowserObservation records
// carrying CDP-marker detections sufficient to trigger
// cdp_marker_density_v1 (default threshold = 2 detections aggregated
// per actor). Returns the substrate.
//
// fixtureKind:
//   - "above_threshold": single actor with 3 detection records (total
//     aggregate detection_count = 6, above threshold 2) → 1 candidate.
//   - "below_threshold": single actor with 1 detection record (count = 1
//     below threshold 2) → 0 candidates.
func substrateWithCDPMarkerObservations(t *testing.T, fixtureKind string) *substrate.Substrate {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	in := ingest.New(sub, time.Now)
	emit := func(actor string, markers []string, count uint32, observedAt int64) {
		obs := &eventsv1.BrowserObservation{
			ObservedAt:          observedAt,
			ActorRef:            actor,
			CollectorRef:        "browser-sdk:v1",
			AuthenticationClass: commonv1.AuthenticationClass_AUTHENTICATION_CLASS_CLIENT_WITNESSED,
			Modality: &eventsv1.BrowserObservation_CdpMarker{
				CdpMarker: &eventsv1.BrowserCDPMarker{
					MarkersObserved: markers,
					DetectionCount:  count,
				},
			},
		}
		if _, err := in.Append(ctx, obs, obs.ObservedAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("Append %s: %v", actor, err)
		}
	}

	switch fixtureKind {
	case "above_threshold":
		emit("actor-bot", []string{"navigator.webdriver=true"}, 2, 1)
		emit("actor-bot", []string{"$cdc_test"}, 2, 2)
		emit("actor-bot", []string{"missing-chrome.csi"}, 2, 3)
	case "below_threshold":
		emit("actor-bot", []string{"navigator.webdriver=true"}, 1, 1)
	}
	return sub
}

// TestFindCandidatesAGBrowserHTTPHappyPath verifies the GET endpoint
// against a substrate with above-threshold CDP-marker detections.
func TestFindCandidatesAGBrowserHTTPHappyPath(t *testing.T) {
	sub := substrateWithCDPMarkerObservations(t, "above_threshold")
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet, "/v1/find-candidates/automation-group-browser", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
	var out findCandidatesPayload
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.SignatureName != "cdp_marker_density_v1" {
		t.Errorf("SignatureName: got %q, want cdp_marker_density_v1", out.SignatureName)
	}
	if out.CandidateCount != 1 {
		t.Errorf("CandidateCount: got %d, want 1", out.CandidateCount)
	}
	if len(out.Candidates) != 1 {
		t.Fatalf("Candidates count: got %d, want 1", len(out.Candidates))
	}
	c := out.Candidates[0]
	if c.HypothesisSubtype != "AutomationGroup" {
		t.Errorf("HypothesisSubtype: got %q, want AutomationGroup", c.HypothesisSubtype)
	}
}

// TestFindCandidatesAGBrowserHTTPBelowThreshold verifies empty
// candidates envelope for sub-threshold input.
func TestFindCandidatesAGBrowserHTTPBelowThreshold(t *testing.T) {
	sub := substrateWithCDPMarkerObservations(t, "below_threshold")
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet, "/v1/find-candidates/automation-group-browser", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var out findCandidatesPayload
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	if out.CandidateCount != 0 {
		t.Errorf("CandidateCount: got %d, want 0", out.CandidateCount)
	}
}

// TestFindCandidatesAGBrowserHTTPMethodNotAllowed verifies non-GET methods 405.
func TestFindCandidatesAGBrowserHTTPMethodNotAllowed(t *testing.T) {
	sub := substrateWithCDPMarkerObservations(t, "below_threshold")
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodPost, "/v1/find-candidates/automation-group-browser", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("status: got %d, want 405", rr.Code)
	}
}

// TestFindCandidatesAGBrowserHTTPSubstrateNotConfigured verifies 503.
func TestFindCandidatesAGBrowserHTTPSubstrateNotConfigured(t *testing.T) {
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/find-candidates/automation-group-browser", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("status: got %d, want 503", rr.Code)
	}
}
