package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHypothesisListHappyPathReturnsBC(t *testing.T) {
	sub, _, _ := substrateWithBCFormation(t)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet, "/v1/hypotheses", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
	var out []map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode body: %v; raw=%s", err, rr.Body.String())
	}
	if len(out) != 1 {
		t.Fatalf("entries: got %d, want 1", len(out))
	}
	if out[0]["subtype"] != "behavioral_cluster" {
		t.Errorf("subtype: got %v, want behavioral_cluster", out[0]["subtype"])
	}
	if out[0]["state"] != "forming" {
		t.Errorf("state: got %v, want forming", out[0]["state"])
	}
}

func TestHypothesisListEmptySubstrate(t *testing.T) {
	sub, _, _ := substrateWithBCFormation(t)
	// Use a different empty substrate to test empty case.
	_ = sub
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	// Use a state filter that matches nothing (forming exists, but
	// promoted does not — the BC formation is in forming state).
	req := httptest.NewRequest(http.MethodGet, "/v1/hypotheses?state=promoted", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("status: got %d, want %d", got, want)
	}
	if rr.Body.String() != "[]\n" {
		t.Errorf("empty result body: got %q, want %q", rr.Body.String(), "[]\n")
	}
}

func TestHypothesisListSubtypeFilter(t *testing.T) {
	sub, _, _ := substrateWithBCFormation(t)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	// Filter to AG → no BC entries should appear → empty result.
	req := httptest.NewRequest(http.MethodGet,
		"/v1/hypotheses?subtype=automation_group", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("status: got %d, want %d", got, want)
	}
	if rr.Body.String() != "[]\n" {
		t.Errorf("filtered-to-AG body: got %q, want %q", rr.Body.String(), "[]\n")
	}
}

func TestHypothesisListInvalidSubtype(t *testing.T) {
	sub, _, _ := substrateWithBCFormation(t)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet,
		"/v1/hypotheses?subtype=not_a_subtype", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if got, want := rr.Code, http.StatusBadRequest; got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
}

func TestHypothesisListInvalidState(t *testing.T) {
	sub, _, _ := substrateWithBCFormation(t)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet,
		"/v1/hypotheses?state=not_a_state", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if got, want := rr.Code, http.StatusBadRequest; got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
}

func TestHypothesisListInvalidNumericParam(t *testing.T) {
	sub, _, _ := substrateWithBCFormation(t)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet,
		"/v1/hypotheses?limit=not-an-int", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if got, want := rr.Code, http.StatusBadRequest; got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
}

func TestHypothesisListNegativeNumericParam(t *testing.T) {
	sub, _, _ := substrateWithBCFormation(t)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet,
		"/v1/hypotheses?limit=-5", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if got, want := rr.Code, http.StatusBadRequest; got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
}

func TestHypothesisListAfterAfterBefore(t *testing.T) {
	sub, _, _ := substrateWithBCFormation(t)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet,
		"/v1/hypotheses?after_ns=2000&before_ns=1000", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if got, want := rr.Code, http.StatusBadRequest; got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
}

func TestHypothesisListRejectsNonGet(t *testing.T) {
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/hypotheses", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if got, want := rr.Code, http.StatusMethodNotAllowed; got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
}

func TestHypothesisListRequiresSubstrateConfigured(t *testing.T) {
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil) // WithSubstrate NOT supplied

	req := httptest.NewRequest(http.MethodGet, "/v1/hypotheses", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if got, want := rr.Code, http.StatusServiceUnavailable; got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
}

func TestHypothesisListPagingLimitAndOffset(t *testing.T) {
	// The single BC formation produces one entry. Test limit=0 → 1 entry;
	// limit=1 → 1 entry; offset=1 → 0 entries.
	sub, _, _ := substrateWithBCFormation(t)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	for _, tc := range []struct {
		query   string
		wantLen int
	}{
		{"?limit=0", 1},
		{"?limit=1", 1},
		{"?limit=10", 1},
		{"?offset=0", 1},
		{"?offset=1", 0},
		{"?offset=10", 0},
	} {
		t.Run(tc.query, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/v1/hypotheses"+tc.query, nil)
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, req)
			if rr.Code != http.StatusOK {
				t.Fatalf("status: got %d, want 200; body=%s", rr.Code, rr.Body.String())
			}
			var out []map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if len(out) != tc.wantLen {
				t.Errorf("len: got %d, want %d", len(out), tc.wantLen)
			}
		})
	}
}
