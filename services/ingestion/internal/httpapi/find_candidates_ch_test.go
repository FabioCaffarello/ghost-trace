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

// substrateWithEndpointCohortObservations ingests count NetworkObservations
// each targeting the same endpoint within the same time bucket (60s);
// cluster-shared per temporal_endpoint_cohort_v1 detection.
func substrateWithEndpointCohortObservations(t *testing.T, count int) *substrate.Substrate {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	in := ingest.New(sub, time.Now)
	const bucketStart = int64(1716120000_000_000_000)
	for i := 0; i < count; i++ {
		obs := &eventsv1.NetworkObservation{
			ObservedAt:          bucketStart + int64(i*10_000_000_000),
			ActorRef:            "actor-ch-" + string(rune('a'+i)),
			EndpointRef:         "10.0.0.1:443",
			CollectorRef:        "test-collector:v1",
			AuthenticationClass: commonv1.AuthenticationClass_AUTHENTICATION_CLASS_SERVER_AUTHENTICATED,
			Modality: &eventsv1.NetworkObservation_TcpFingerprint{
				TcpFingerprint: &eventsv1.NetworkTcpFingerprint{WindowSize: 65535},
			},
		}
		if _, err := in.Append(ctx, obs, obs.ObservedAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("Append %d: %v", i, err)
		}
	}
	return sub
}

// TestFindCandidatesCHHTTPHappyPath verifies endpoint emits 1
// CampaignHypothesis candidate for 3 events at same (endpoint, bucket).
func TestFindCandidatesCHHTTPHappyPath(t *testing.T) {
	sub := substrateWithEndpointCohortObservations(t, 3)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet, "/v1/find-candidates/campaign-hypothesis", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
	var out findCandidatesPayload
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.SignatureName != "temporal_endpoint_cohort_v1" {
		t.Errorf("SignatureName: got %q, want temporal_endpoint_cohort_v1", out.SignatureName)
	}
	if out.CandidateCount != 1 {
		t.Errorf("CandidateCount: got %d, want 1", out.CandidateCount)
	}
	if len(out.Candidates) != 1 {
		t.Fatalf("Candidates count: got %d, want 1", len(out.Candidates))
	}
	c := out.Candidates[0]
	if c.HypothesisSubtype != "CampaignHypothesis" {
		t.Errorf("HypothesisSubtype: got %q, want CampaignHypothesis", c.HypothesisSubtype)
	}
}

// TestFindCandidatesCHHTTPBelowThreshold verifies sub-threshold input.
func TestFindCandidatesCHHTTPBelowThreshold(t *testing.T) {
	sub := substrateWithEndpointCohortObservations(t, 2)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet, "/v1/find-candidates/campaign-hypothesis", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rr.Code)
	}
	var out findCandidatesPayload
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	if out.CandidateCount != 0 {
		t.Errorf("CandidateCount: got %d, want 0", out.CandidateCount)
	}
}

// TestFindCandidatesCHHTTPBucketSecondsOverride verifies the
// bucket_seconds query param widens the cluster window.
func TestFindCandidatesCHHTTPBucketSecondsOverride(t *testing.T) {
	// With default 60s bucket: 3 events at 0/120s/240s span 3 distinct buckets.
	// With 300s bucket: all 3 collapse into one cluster → 1 candidate.
	sub := func() *substrate.Substrate {
		t.Helper()
		dir := t.TempDir()
		ctx := context.Background()
		s, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
		if err != nil {
			t.Fatalf("substrate.Open: %v", err)
		}
		t.Cleanup(func() { _ = s.Close() })
		in := ingest.New(s, time.Now)
		const bucketStart = int64(1716120000_000_000_000)
		for i, dt := range []int64{0, 120_000_000_000, 240_000_000_000} {
			obs := &eventsv1.NetworkObservation{
				ObservedAt:          bucketStart + dt,
				ActorRef:            "actor-wide-" + string(rune('a'+i)),
				EndpointRef:         "10.0.0.1:443",
				CollectorRef:        "test-collector:v1",
				AuthenticationClass: commonv1.AuthenticationClass_AUTHENTICATION_CLASS_SERVER_AUTHENTICATED,
				Modality: &eventsv1.NetworkObservation_TcpFingerprint{
					TcpFingerprint: &eventsv1.NetworkTcpFingerprint{WindowSize: 65535},
				},
			}
			if _, err := in.Append(ctx, obs, obs.ObservedAt, ingest.Envelope{Channel: "test"}); err != nil {
				t.Fatalf("Append %d: %v", i, err)
			}
		}
		return s
	}()
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet, "/v1/find-candidates/campaign-hypothesis?bucket_seconds=300", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var out findCandidatesPayload
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	if out.CandidateCount != 1 {
		t.Errorf("CandidateCount with bucket_seconds=300: got %d, want 1 (300s widens to single cohort)", out.CandidateCount)
	}
}

// TestFindCandidatesCHHTTPMethodNotAllowed verifies non-GET 405.
func TestFindCandidatesCHHTTPMethodNotAllowed(t *testing.T) {
	sub := substrateWithEndpointCohortObservations(t, 0)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodPost, "/v1/find-candidates/campaign-hypothesis", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("status: got %d, want 405", rr.Code)
	}
}

// TestFindCandidatesCHHTTPSubstrateNotConfigured verifies 503.
func TestFindCandidatesCHHTTPSubstrateNotConfigured(t *testing.T) {
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/find-candidates/campaign-hypothesis", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("status: got %d, want 503", rr.Code)
	}
}
