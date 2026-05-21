package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHypothesisSummaryHappyPathBC(t *testing.T) {
	sub, _, _ := substrateWithBCFormation(t)
	doAppend, _ := stubAppendFunc(nil)
	h := New(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet, "/v1/hypotheses/summary", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
	var out map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v; raw=%s", err, rr.Body.String())
	}

	// Top-level structure: combined + four per-subtype sections.
	for _, key := range []string{
		"combined", "behavioral_cluster", "automation_group",
		"campaign_hypothesis", "coordination_ring",
	} {
		if _, ok := out[key]; !ok {
			t.Errorf("missing top-level key %q", key)
		}
	}

	// combined.total should reflect the one BC formation.
	combined, ok := out["combined"].(map[string]interface{})
	if !ok {
		t.Fatalf("combined: not an object: %v", out["combined"])
	}
	if combined["total"].(float64) != 1 {
		t.Errorf("combined.total: got %v, want 1", combined["total"])
	}

	// behavioral_cluster.total == 1.
	bc, ok := out["behavioral_cluster"].(map[string]interface{})
	if !ok {
		t.Fatalf("behavioral_cluster: not an object")
	}
	if bc["total"].(float64) != 1 {
		t.Errorf("bc.total: got %v, want 1", bc["total"])
	}

	// Other subtypes total == 0.
	for _, key := range []string{"automation_group", "campaign_hypothesis", "coordination_ring"} {
		section, ok := out[key].(map[string]interface{})
		if !ok {
			t.Fatalf("%s: not an object", key)
		}
		if section["total"].(float64) != 0 {
			t.Errorf("%s.total: got %v, want 0", key, section["total"])
		}
	}

	// Every section must have a `latencies` payload with the three
	// dimensions (predictable wire shape).
	for _, key := range []string{
		"combined", "behavioral_cluster", "automation_group",
		"campaign_hypothesis", "coordination_ring",
	} {
		section := out[key].(map[string]interface{})
		latencies, ok := section["latencies"].(map[string]interface{})
		if !ok {
			t.Errorf("%s.latencies: missing or wrong type", key)
			continue
		}
		for _, dim := range []string{
			"formation_to_first_promotion_ns",
			"latest_promotion_to_latest_demotion_ns",
			"formation_to_dissolution_ns",
		} {
			if _, ok := latencies[dim]; !ok {
				t.Errorf("%s.latencies.%s: missing", key, dim)
			}
		}
	}
}

func TestHypothesisSummaryEveryStateKeyPresent(t *testing.T) {
	// The §0053 predictable-wire-shape commitment carries through to
	// the HTTP endpoint: every State key appears in by_state, even
	// at zero.
	sub, _, _ := substrateWithBCFormation(t)
	doAppend, _ := stubAppendFunc(nil)
	h := New(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet, "/v1/hypotheses/summary", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	var out map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	wantStates := []string{
		"forming", "promoted", "demoted",
		"dissolved", "merged_into", "split_into",
	}
	for _, sectionKey := range []string{
		"combined", "behavioral_cluster", "automation_group",
		"campaign_hypothesis", "coordination_ring",
	} {
		section := out[sectionKey].(map[string]interface{})
		byState := section["by_state"].(map[string]interface{})
		for _, s := range wantStates {
			if _, ok := byState[s]; !ok {
				t.Errorf("%s.by_state[%q]: missing (predictable-wire-shape commitment)", sectionKey, s)
			}
		}
	}
}

func TestHypothesisSummaryInvalidNumericParam(t *testing.T) {
	sub, _, _ := substrateWithBCFormation(t)
	doAppend, _ := stubAppendFunc(nil)
	h := New(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet,
		"/v1/hypotheses/summary?after_ns=not-an-int", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if got, want := rr.Code, http.StatusBadRequest; got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
}

func TestHypothesisSummaryNegativeNumericParam(t *testing.T) {
	sub, _, _ := substrateWithBCFormation(t)
	doAppend, _ := stubAppendFunc(nil)
	h := New(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet,
		"/v1/hypotheses/summary?after_ns=-5", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if got, want := rr.Code, http.StatusBadRequest; got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
}

func TestHypothesisSummaryAfterAfterBefore(t *testing.T) {
	sub, _, _ := substrateWithBCFormation(t)
	doAppend, _ := stubAppendFunc(nil)
	h := New(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet,
		"/v1/hypotheses/summary?after_ns=2000&before_ns=1000", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if got, want := rr.Code, http.StatusBadRequest; got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
}

func TestHypothesisSummaryRejectsNonGet(t *testing.T) {
	doAppend, _ := stubAppendFunc(nil)
	h := New(doAppend, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/hypotheses/summary", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if got, want := rr.Code, http.StatusMethodNotAllowed; got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
}

func TestHypothesisSummaryRequiresSubstrateConfigured(t *testing.T) {
	doAppend, _ := stubAppendFunc(nil)
	h := New(doAppend, nil) // WithSubstrate NOT supplied

	req := httptest.NewRequest(http.MethodGet, "/v1/hypotheses/summary", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if got, want := rr.Code, http.StatusServiceUnavailable; got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
}

func TestHypothesisSummaryTimeWindowExcludesAll(t *testing.T) {
	// Filter to a time window that's after the BC formation's latest
	// event_time. combined.total should be 0; per-subtype totals
	// should all be 0; by_state should still have all keys with
	// zero values.
	sub, _, _ := substrateWithBCFormation(t)
	doAppend, _ := stubAppendFunc(nil)
	h := New(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet,
		"/v1/hypotheses/summary?after_ns=9999999999999", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
	var out map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["combined"].(map[string]interface{})["total"].(float64) != 0 {
		t.Errorf("combined.total under exclusive window: got %v, want 0",
			out["combined"].(map[string]interface{})["total"])
	}
}

func TestHypothesisSummaryEquivalenceWithListCount(t *testing.T) {
	// §0078 + §0053 invariants extended to HTTP: for every State,
	// combined.by_state[s] equals the sum of per-subtype by_state[s].
	sub, _, _ := substrateWithBCFormation(t)
	doAppend, _ := stubAppendFunc(nil)
	h := New(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet, "/v1/hypotheses/summary", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("status: got %d, want %d", got, want)
	}
	var out map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}

	wantStates := []string{
		"forming", "promoted", "demoted",
		"dissolved", "merged_into", "split_into",
	}
	combinedBS := out["combined"].(map[string]interface{})["by_state"].(map[string]interface{})
	for _, state := range wantStates {
		sum := 0.0
		for _, sectionKey := range []string{
			"behavioral_cluster", "automation_group",
			"campaign_hypothesis", "coordination_ring",
		} {
			section := out[sectionKey].(map[string]interface{})
			bs := section["by_state"].(map[string]interface{})
			sum += bs[state].(float64)
		}
		got := combinedBS[state].(float64)
		if got != sum {
			t.Errorf("combined.by_state[%q] (%v) != sum of per-subtype by_state[%q] (%v)",
				state, got, state, sum)
		}
	}
}
