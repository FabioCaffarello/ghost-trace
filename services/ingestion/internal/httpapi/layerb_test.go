package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// layerBHappyPathURL builds the GET /v1/layer-b-verdict URL with a
// half/half threshold tuple (T_B = 1/2, K_C = 1/2) over the supplied
// formation-hash hex + N=100 window + 1-day N_A.
func layerBHappyPathURL(formationHashHex string) string {
	return fmt.Sprintf("/v1/layer-b-verdict?"+
		"formation_event_hash_hex=%s"+
		"&t_b_numerator=1&t_b_denominator=2"+
		"&k_c_numerator=1&k_c_denominator=2"+
		"&n_window=100"+
		"&n_a_duration_nanoseconds=86400000000000",
		formationHashHex)
}

// TestLayerBVerdictHTTPHappyPathBC exercises the GET /v1/layer-b-verdict
// endpoint against a substrate carrying a single BehavioralCluster
// formation. Verifies the JSON envelope shape + verdict fields
// populated correctly.
func TestLayerBVerdictHTTPHappyPathBC(t *testing.T) {
	sub, _, formationHex := substrateWithBCFormation(t)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet, layerBHappyPathURL(formationHex), nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type: got %q want application/json; charset=utf-8", ct)
	}
	var out layerBVerdictPayload
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.FormationEventHashHex != formationHex {
		t.Errorf("FormationEventHashHex echo: got %q, want %q", out.FormationEventHashHex, formationHex)
	}
	// WindowEventCount > 0 because substrate has at least the BC
	// formation event + its precursor session+ingestion events.
	if out.WindowEventCount == 0 {
		t.Error("WindowEventCount: got 0, expected substrate events to be walked")
	}
}

// TestLayerBVerdictHTTPMissingHash verifies that a missing formation
// hash query param surfaces 400.
func TestLayerBVerdictHTTPMissingHash(t *testing.T) {
	sub, _, _ := substrateWithBCFormation(t)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet,
		"/v1/layer-b-verdict?t_b_numerator=1&t_b_denominator=2&k_c_numerator=1&k_c_denominator=2&n_window=100&n_a_duration_nanoseconds=1",
		nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400; body=%s", rr.Code, rr.Body.String())
	}
}

// TestLayerBVerdictHTTPZeroDenominator verifies the §2.6 rational-pair
// invariant surfaces as 400 at the HTTP layer (NOT 500 from layerb).
func TestLayerBVerdictHTTPZeroDenominator(t *testing.T) {
	sub, _, formationHex := substrateWithBCFormation(t)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	url := fmt.Sprintf("/v1/layer-b-verdict?formation_event_hash_hex=%s"+
		"&t_b_numerator=1&t_b_denominator=0"+
		"&k_c_numerator=1&k_c_denominator=2"+
		"&n_window=100&n_a_duration_nanoseconds=1", formationHex)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400 (zero denominator); body=%s", rr.Code, rr.Body.String())
	}
}

// TestLayerBVerdictHTTPZeroNWindow verifies n_window=0 surfaces 400.
func TestLayerBVerdictHTTPZeroNWindow(t *testing.T) {
	sub, _, formationHex := substrateWithBCFormation(t)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	url := fmt.Sprintf("/v1/layer-b-verdict?formation_event_hash_hex=%s"+
		"&t_b_numerator=1&t_b_denominator=2"+
		"&k_c_numerator=1&k_c_denominator=2"+
		"&n_window=0&n_a_duration_nanoseconds=1", formationHex)
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400 (n_window=0); body=%s", rr.Code, rr.Body.String())
	}
}

// TestLayerBVerdictHTTPMalformedHash verifies invalid hex hash surfaces 400.
func TestLayerBVerdictHTTPMalformedHash(t *testing.T) {
	sub, _, _ := substrateWithBCFormation(t)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	// 63 chars (one short) — length check fires first.
	url := "/v1/layer-b-verdict?formation_event_hash_hex=" +
		"abcdef0123456789abcdef0123456789abcdef0123456789abcdef012345678" +
		"&t_b_numerator=1&t_b_denominator=2" +
		"&k_c_numerator=1&k_c_denominator=2" +
		"&n_window=100&n_a_duration_nanoseconds=1"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400 (63-char hash); body=%s", rr.Code, rr.Body.String())
	}
}

// TestLayerBVerdictHTTPMethodNotAllowed verifies non-GET methods
// surface 405 + Allow header.
func TestLayerBVerdictHTTPMethodNotAllowed(t *testing.T) {
	sub, _, _ := substrateWithBCFormation(t)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodPost, "/v1/layer-b-verdict", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusMethodNotAllowed; got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
	if allow := rr.Header().Get("Allow"); allow != http.MethodGet {
		t.Errorf("Allow header: got %q, want %q", allow, http.MethodGet)
	}
}

// TestLayerBVerdictHTTPSubstrateNotConfigured verifies 503 when handler
// was constructed without WithSubstrate.
func TestLayerBVerdictHTTPSubstrateNotConfigured(t *testing.T) {
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil) // no WithSubstrate

	req := httptest.NewRequest(http.MethodGet, "/v1/layer-b-verdict?formation_event_hash_hex=x", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusServiceUnavailable; got != want {
		t.Errorf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
}
