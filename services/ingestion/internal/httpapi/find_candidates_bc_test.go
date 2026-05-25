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

// substrateWithKeystrokeObservations ingests count BehavioralObservation
// records each carrying an identical keystroke-timing fingerprint
// (cluster-shared per keystroke_timing_clustering_v1 detection). Returns
// the substrate.
func substrateWithKeystrokeObservations(t *testing.T, count int) *substrate.Substrate {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	in := ingest.New(sub, time.Now)
	// Shared keystroke-timing fingerprint per cluster.
	pairsNs := []uint64{50_000_000, 100_000_000, 0, 100_000_000, 100_000_000, 150_000_000}
	intervals := make([]*eventsv1.KeystrokeInterval, 0, len(pairsNs)/2)
	for i := 0; i < len(pairsNs); i += 2 {
		intervals = append(intervals, &eventsv1.KeystrokeInterval{
			FlightNs: pairsNs[i],
			DwellNs:  pairsNs[i+1],
		})
	}
	for i := 0; i < count; i++ {
		actor := "actor-bc-" + string(rune('a'+i))
		obs := &eventsv1.BehavioralObservation{
			ObservedAt:          int64(1000 + i),
			ActorRef:            actor,
			CollectorRef:        "browser-sdk:v1",
			AuthenticationClass: commonv1.AuthenticationClass_AUTHENTICATION_CLASS_CLIENT_WITNESSED,
			Modality: &eventsv1.BehavioralObservation_KeystrokeTiming{
				KeystrokeTiming: &eventsv1.BehavioralKeystrokeTiming{
					Intervals:           intervals,
					TotalKeystrokeCount: uint32(len(intervals)),
				},
			},
		}
		if _, err := in.Append(ctx, obs, obs.ObservedAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("Append %d: %v", i, err)
		}
	}
	return sub
}

// TestFindCandidatesBCHTTPHappyPath exercises GET
// /v1/find-candidates/behavioral-cluster against a substrate with 3
// keystroke-fingerprint-sharing actors. Verifies the JSON envelope shape +
// CoordinationRing-like wire-parity with the CLI's emissionEnvelope.
func TestFindCandidatesBCHTTPHappyPath(t *testing.T) {
	sub := substrateWithKeystrokeObservations(t, 3)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet, "/v1/find-candidates/behavioral-cluster", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type: got %q want application/json; charset=utf-8", ct)
	}
	var out findCandidatesPayload
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.SignatureName != "keystroke_timing_clustering_v1" {
		t.Errorf("SignatureName: got %q, want keystroke_timing_clustering_v1", out.SignatureName)
	}
	if out.CandidateCount != 1 {
		t.Errorf("CandidateCount: got %d, want 1", out.CandidateCount)
	}
	if len(out.Candidates) != 1 {
		t.Fatalf("Candidates count: got %d, want 1", len(out.Candidates))
	}
	c := out.Candidates[0]
	if c.HypothesisSubtype != "BehavioralCluster" {
		t.Errorf("HypothesisSubtype: got %q, want BehavioralCluster", c.HypothesisSubtype)
	}
	if len(c.ActorRefs) != 3 {
		t.Errorf("ActorRefs: got %d, want 3 actors clustered", len(c.ActorRefs))
	}
	for i, hh := range c.SourceHashesHex {
		if len(hh) != 64 {
			t.Errorf("SourceHashesHex[%d]: got len %d, want 64", i, len(hh))
		}
	}
}

// TestFindCandidatesBCHTTPBelowThreshold verifies that a 2-actor cluster
// (below threshold 3) returns a valid envelope with empty candidates.
func TestFindCandidatesBCHTTPBelowThreshold(t *testing.T) {
	sub := substrateWithKeystrokeObservations(t, 2)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet, "/v1/find-candidates/behavioral-cluster", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var out findCandidatesPayload
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	if out.CandidateCount != 0 {
		t.Errorf("CandidateCount: got %d, want 0 (below threshold)", out.CandidateCount)
	}
	if out.Stats.ObservationsScanned != 2 {
		t.Errorf("ObservationsScanned: got %d, want 2", out.Stats.ObservationsScanned)
	}
}

// TestFindCandidatesBCHTTPThresholdOverride verifies the threshold
// query param suppresses candidate emission when set above the cluster size.
func TestFindCandidatesBCHTTPThresholdOverride(t *testing.T) {
	sub := substrateWithKeystrokeObservations(t, 3)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	// Threshold 5 > cluster size 3 → no candidates.
	req := httptest.NewRequest(http.MethodGet, "/v1/find-candidates/behavioral-cluster?threshold=5", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var out findCandidatesPayload
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	if out.CandidateCount != 0 {
		t.Errorf("CandidateCount with threshold=5: got %d, want 0", out.CandidateCount)
	}
}

// TestFindCandidatesBCHTTPNegativeThreshold verifies negative threshold
// surfaces 400.
func TestFindCandidatesBCHTTPNegativeThreshold(t *testing.T) {
	sub := substrateWithKeystrokeObservations(t, 0)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet, "/v1/find-candidates/behavioral-cluster?threshold=-1", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", rr.Code)
	}
}

// TestFindCandidatesBCHTTPMethodNotAllowed verifies non-GET methods
// surface 405 + Allow header.
func TestFindCandidatesBCHTTPMethodNotAllowed(t *testing.T) {
	sub := substrateWithKeystrokeObservations(t, 0)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodPost, "/v1/find-candidates/behavioral-cluster", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusMethodNotAllowed; got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
	if allow := rr.Header().Get("Allow"); allow != http.MethodGet {
		t.Errorf("Allow header: got %q, want %q", allow, http.MethodGet)
	}
}

// TestFindCandidatesBCHTTPSubstrateNotConfigured verifies 503 when
// handler lacks WithSubstrate.
func TestFindCandidatesBCHTTPSubstrateNotConfigured(t *testing.T) {
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil) // no WithSubstrate

	req := httptest.NewRequest(http.MethodGet, "/v1/find-candidates/behavioral-cluster", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusServiceUnavailable; got != want {
		t.Errorf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
}
