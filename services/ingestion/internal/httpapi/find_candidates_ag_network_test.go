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

// substrateWithP0FObservations ingests count NetworkObservations sharing
// an identical p0f_signature fingerprint string — cluster-shared per
// tcp_fingerprint_clustering_v1 detection.
func substrateWithP0FObservations(t *testing.T, count int) *substrate.Substrate {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	in := ingest.New(sub, time.Now)
	for i := 0; i < count; i++ {
		obs := &eventsv1.NetworkObservation{
			ObservedAt:          int64(1716120000000000000 + i),
			ActorRef:            "actor-p0f-" + string(rune('a'+i)),
			EndpointRef:         "198.51.100.5:443",
			CollectorRef:        "test-collector:v1",
			AuthenticationClass: commonv1.AuthenticationClass_AUTHENTICATION_CLASS_SERVER_AUTHENTICATED,
			Modality: &eventsv1.NetworkObservation_TcpFingerprint{
				TcpFingerprint: &eventsv1.NetworkTcpFingerprint{
					P0FSignature: "4:64:0:1460:mss*4,0:.",
					WindowSize:   65535,
					Mss:          1460,
					Ttl:          64,
				},
			},
		}
		if _, err := in.Append(ctx, obs, obs.ObservedAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("Append %d: %v", i, err)
		}
	}
	return sub
}

// TestFindCandidatesAGNetworkHTTPHappyPathP0F verifies the p0f
// signature selection emits 1 AutomationGroup candidate.
func TestFindCandidatesAGNetworkHTTPHappyPathP0F(t *testing.T) {
	sub := substrateWithP0FObservations(t, 3)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet, "/v1/find-candidates/automation-group-network?signature=p0f", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
	var out findCandidatesPayload
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.SignatureName != "tcp_fingerprint_clustering_v1" {
		t.Errorf("SignatureName: got %q, want tcp_fingerprint_clustering_v1", out.SignatureName)
	}
	if out.CandidateCount != 1 {
		t.Errorf("CandidateCount: got %d, want 1", out.CandidateCount)
	}
	if len(out.Candidates) != 1 {
		t.Fatalf("Candidates count: got %d, want 1", len(out.Candidates))
	}
	if out.Candidates[0].HypothesisSubtype != "AutomationGroup" {
		t.Errorf("HypothesisSubtype: got %q, want AutomationGroup", out.Candidates[0].HypothesisSubtype)
	}
}

// TestFindCandidatesAGNetworkHTTPMissingSignature verifies the required
// signature query param surfaces 400 when missing.
func TestFindCandidatesAGNetworkHTTPMissingSignature(t *testing.T) {
	sub := substrateWithP0FObservations(t, 0)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet, "/v1/find-candidates/automation-group-network", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400 (missing signature); body=%s", rr.Code, rr.Body.String())
	}
}

// TestFindCandidatesAGNetworkHTTPUnknownSignature verifies unknown
// signature value surfaces 400.
func TestFindCandidatesAGNetworkHTTPUnknownSignature(t *testing.T) {
	sub := substrateWithP0FObservations(t, 0)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet, "/v1/find-candidates/automation-group-network?signature=nonexistent", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400 (unknown signature); body=%s", rr.Code, rr.Body.String())
	}
}

// TestFindCandidatesAGNetworkHTTPFlowFeaturesAcceptance verifies that
// signature=flow-features is accepted (the test fixtures don't trigger
// flow-features clustering, so we expect 200 + 0 candidates; the
// structural witness is that the signature selection path resolved).
func TestFindCandidatesAGNetworkHTTPFlowFeaturesAcceptance(t *testing.T) {
	sub := substrateWithP0FObservations(t, 0)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet, "/v1/find-candidates/automation-group-network?signature=flow-features", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var out findCandidatesPayload
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	if out.SignatureName != "tcp_flow_features_clustering_v1" {
		t.Errorf("SignatureName: got %q, want tcp_flow_features_clustering_v1", out.SignatureName)
	}
}

// TestFindCandidatesAGNetworkHTTPMethodNotAllowed verifies non-GET 405.
func TestFindCandidatesAGNetworkHTTPMethodNotAllowed(t *testing.T) {
	sub := substrateWithP0FObservations(t, 0)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodPost, "/v1/find-candidates/automation-group-network?signature=p0f", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("status: got %d, want 405", rr.Code)
	}
}

// TestFindCandidatesAGNetworkHTTPSubstrateNotConfigured verifies 503.
func TestFindCandidatesAGNetworkHTTPSubstrateNotConfigured(t *testing.T) {
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/find-candidates/automation-group-network?signature=p0f", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("status: got %d, want 503", rr.Code)
	}
}
