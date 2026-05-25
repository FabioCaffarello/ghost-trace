package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestMorphologyHTTPHappyPathBC exercises the GET /v1/morphology
// endpoint against a substrate carrying a single BehavioralCluster
// formation. Verifies the JSON envelope shape + per-hypothesis
// payload fields populated correctly.
func TestMorphologyHTTPHappyPathBC(t *testing.T) {
	sub, formationHash, formationHex := substrateWithBCFormation(t)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet, "/v1/morphology", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type: got %q want application/json; charset=utf-8", ct)
	}
	var out morphologyPayload
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Stats.TotalFormations != 1 {
		t.Errorf("TotalFormations: got %d, want 1", out.Stats.TotalFormations)
	}
	if got := out.Stats.PerSubtype["ghosttrace.events.v1.BehavioralClusterFormation"]; got != 1 {
		t.Errorf("PerSubtype[BehavioralClusterFormation]: got %d, want 1", got)
	}
	if len(out.Hypotheses) != 1 {
		t.Fatalf("Hypotheses count: got %d, want 1", len(out.Hypotheses))
	}
	hp := out.Hypotheses[0]
	if hp.HypothesisHashHex != formationHex {
		t.Errorf("HypothesisHashHex: got %q, want %q", hp.HypothesisHashHex, formationHex)
	}
	if hp.SubtypeName != "ghosttrace.events.v1.BehavioralClusterFormation" {
		t.Errorf("SubtypeName: got %q, want BehavioralClusterFormation", hp.SubtypeName)
	}
	// Root formation: depth=0, breadth=0, closure=0.
	if hp.ChainDepthMax != 0 {
		t.Errorf("ChainDepthMax: got %d, want 0 (root formation)", hp.ChainDepthMax)
	}
	if hp.ChainBreadthAtRoot != 0 {
		t.Errorf("ChainBreadthAtRoot: got %d, want 0 (root formation)", hp.ChainBreadthAtRoot)
	}
	if hp.ClosureCount != 0 {
		t.Errorf("ClosureCount: got %d, want 0 (root formation)", hp.ClosureCount)
	}
	_ = formationHash
}

// TestMorphologyHTTPEmptySubstrate verifies the endpoint returns a
// valid envelope (empty hypotheses array + zero stats) when no
// formations exist.
func TestMorphologyHTTPEmptySubstrate(t *testing.T) {
	sub := substrateWithDeclaredSessionsHTTPVerify(t, 0)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet, "/v1/morphology", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var out morphologyPayload
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Stats.TotalFormations != 0 {
		t.Errorf("TotalFormations: got %d, want 0 (empty substrate)", out.Stats.TotalFormations)
	}
	if len(out.Hypotheses) != 0 {
		t.Errorf("Hypotheses count: got %d, want 0 (empty substrate)", len(out.Hypotheses))
	}
}

// TestMorphologyHTTPMethodNotAllowed verifies that non-GET methods
// surface 405 + Allow header.
func TestMorphologyHTTPMethodNotAllowed(t *testing.T) {
	sub := substrateWithDeclaredSessionsHTTPVerify(t, 0)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodPost, "/v1/morphology", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusMethodNotAllowed; got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
	if allow := rr.Header().Get("Allow"); allow != http.MethodGet {
		t.Errorf("Allow header: got %q, want %q", allow, http.MethodGet)
	}
}

// TestMorphologyHTTPSubstrateNotConfigured verifies that the endpoint
// returns 503 when the handler was constructed without WithSubstrate.
func TestMorphologyHTTPSubstrateNotConfigured(t *testing.T) {
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil) // no WithSubstrate

	req := httptest.NewRequest(http.MethodGet, "/v1/morphology", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusServiceUnavailable; got != want {
		t.Errorf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
}
