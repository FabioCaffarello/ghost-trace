package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestFindCandidatesCRHTTPHappyPath verifies the GET endpoint emits 1
// CoordinationRing candidate WITH populated Interactions field (3
// actors → 3 lex-canonical edges per §0070).
func TestFindCandidatesCRHTTPHappyPath(t *testing.T) {
	sub := substrateWithEndpointCohortObservations(t, 3)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet, "/v1/find-candidates/coordination-ring", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
	var out findCandidatesPayload
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.SignatureName != "endpoint_co_visit_v1" {
		t.Errorf("SignatureName: got %q, want endpoint_co_visit_v1", out.SignatureName)
	}
	if out.CandidateCount != 1 {
		t.Errorf("CandidateCount: got %d, want 1", out.CandidateCount)
	}
	if len(out.Candidates) != 1 {
		t.Fatalf("Candidates count: got %d, want 1", len(out.Candidates))
	}
	c := out.Candidates[0]
	if c.HypothesisSubtype != "CoordinationRing" {
		t.Errorf("HypothesisSubtype: got %q, want CoordinationRing", c.HypothesisSubtype)
	}
	// 3 actors → 3 edges per §0070 within-edge lex.
	if len(c.Interactions) != 3 {
		t.Errorf("Interactions count: got %d, want 3 (3-actor ring → 3 edges)", len(c.Interactions))
	}
	for i, edge := range c.Interactions {
		if edge[0] >= edge[1] {
			t.Errorf("Interactions[%d]: edge[0]=%q NOT lex-less-than edge[1]=%q (per §0070)", i, edge[0], edge[1])
		}
	}
}

// TestFindCandidatesCRHTTPInteractionsOmittedForOtherSubtypes verifies
// the `omitempty` discipline: the BC endpoint's JSON response must NOT
// contain an `interactions` field when no Interactions are populated.
// This is the structural witness that the wire-shape extension is
// backward-compatible per §0186 MO1.
func TestFindCandidatesCRHTTPInteractionsOmittedForOtherSubtypes(t *testing.T) {
	sub := substrateWithKeystrokeObservations(t, 3)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet, "/v1/find-candidates/behavioral-cluster", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200", rr.Code)
	}
	// String-level check: BC response body must NOT contain "interactions"
	// (omitempty removes the field when slice is nil/empty).
	body := rr.Body.String()
	for _, needle := range []string{`"interactions"`, `"interactions":`} {
		for _, c := range []byte(body) {
			_ = c
		}
		if found := indexAny(body, needle); found != -1 {
			t.Errorf("BC response body contains %q at offset %d; expected omitempty to suppress (per §0193 backward-compat)", needle, found)
		}
	}
}

// indexAny is a tiny local substring matcher to avoid importing strings
// just for one check. Returns -1 if not found.
func indexAny(haystack, needle string) int {
	if len(needle) == 0 || len(haystack) < len(needle) {
		return -1
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

// TestFindCandidatesCRHTTPBelowThreshold verifies sub-threshold input.
func TestFindCandidatesCRHTTPBelowThreshold(t *testing.T) {
	sub := substrateWithEndpointCohortObservations(t, 2)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet, "/v1/find-candidates/coordination-ring", nil)
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

// TestFindCandidatesCRHTTPMethodNotAllowed verifies non-GET 405.
func TestFindCandidatesCRHTTPMethodNotAllowed(t *testing.T) {
	sub := substrateWithEndpointCohortObservations(t, 0)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodPost, "/v1/find-candidates/coordination-ring", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("status: got %d, want 405", rr.Code)
	}
}

// TestFindCandidatesCRHTTPSubstrateNotConfigured verifies 503.
func TestFindCandidatesCRHTTPSubstrateNotConfigured(t *testing.T) {
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/find-candidates/coordination-ring", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("status: got %d, want 503", rr.Code)
	}
}
