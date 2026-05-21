package httpapi

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/hypothesis"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// substrateWithBCFormation populates a fresh substrate with two
// DeclaredSessions sharing a descriptor + runs FormAll under the
// session-descriptor-shared-v1 pattern. Returns the substrate + the
// resulting BehavioralClusterFormation hash (32 bytes) + its hex
// encoding.
func substrateWithBCFormation(t *testing.T) (*substrate.Substrate, [32]byte, string) {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	in := ingest.New(sub, func() time.Time { return time.Unix(0, 1000) })
	for _, actor := range []string{"actor-a", "actor-b"} {
		msg := &eventsv1.DeclaredSession{
			DeclaredAt:        1000,
			ActorRef:          actor,
			SessionDescriptor: []byte("shared-descriptor"),
		}
		if _, err := in.Append(ctx, msg, msg.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("ingest.Append: %v", err)
		}
	}
	if _, err := hypothesis.FormAll(ctx, sub,
		hypothesis.SessionDescriptorSharedV1{MinClusterSize: 2},
		func() time.Time { return time.Unix(0, 2000) }); err != nil {
		t.Fatalf("FormAll: %v", err)
	}

	var formationHash [32]byte
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType == "ghosttrace.events.v1.BehavioralClusterFormation" {
			formationHash = row.EventHash
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	if formationHash == ([32]byte{}) {
		t.Fatalf("no BC formation found")
	}
	return sub, formationHash, hex.EncodeToString(formationHash[:])
}

func TestHypothesisStateHappyPathBC(t *testing.T) {
	sub, _, formationHex := substrateWithBCFormation(t)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet,
		"/v1/hypotheses/state?formation_event_hash="+formationHex, nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
	var out map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode body: %v; raw=%s", err, rr.Body.String())
	}
	if out["subtype"] != "behavioral_cluster" {
		t.Errorf("subtype: got %v, want behavioral_cluster", out["subtype"])
	}
	if out["formation_event_hash"] != formationHex {
		t.Errorf("formation_event_hash: got %v, want %s", out["formation_event_hash"], formationHex)
	}
	if out["state"] != "forming" {
		t.Errorf("state: got %v, want forming", out["state"])
	}
}

func TestHypothesisStateRejectsNonGet(t *testing.T) {
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil)

	req := httptest.NewRequest(http.MethodPost, "/v1/hypotheses/state", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if got, want := rr.Code, http.StatusMethodNotAllowed; got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
}

func TestHypothesisStateRequiresSubstrateConfigured(t *testing.T) {
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil) // WithSubstrate NOT supplied

	req := httptest.NewRequest(http.MethodGet,
		"/v1/hypotheses/state?formation_event_hash="+strings.Repeat("0", 64), nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if got, want := rr.Code, http.StatusServiceUnavailable; got != want {
		t.Errorf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
}

func TestHypothesisStateMissingFormationHash(t *testing.T) {
	sub, _, _ := substrateWithBCFormation(t)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet, "/v1/hypotheses/state", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if got, want := rr.Code, http.StatusBadRequest; got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
}

func TestHypothesisStateInvalidHex(t *testing.T) {
	sub, _, _ := substrateWithBCFormation(t)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet,
		"/v1/hypotheses/state?formation_event_hash=xyz-not-hex", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if got, want := rr.Code, http.StatusBadRequest; got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
}

func TestHypothesisStateWrongHashLength(t *testing.T) {
	sub, _, _ := substrateWithBCFormation(t)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet,
		"/v1/hypotheses/state?formation_event_hash=abcd1234", nil) // 4 bytes, not 32
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if got, want := rr.Code, http.StatusBadRequest; got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
}

func TestHypothesisStateUnknownFormation(t *testing.T) {
	sub, _, _ := substrateWithBCFormation(t)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	bogus := strings.Repeat("ff", 32)
	req := httptest.NewRequest(http.MethodGet,
		"/v1/hypotheses/state?formation_event_hash="+bogus, nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if got, want := rr.Code, http.StatusNotFound; got != want {
		t.Errorf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
}

func TestHypothesisStateNotAFormation(t *testing.T) {
	// Pass an IngestionEvent hash (the wrapper event written for every
	// ingested DeclaredSession) — it's in the substrate but it's not a
	// Cat III formation. Must return 404 with target-not-formation.
	sub, _, _ := substrateWithBCFormation(t)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	var ingHash [32]byte
	if err := sub.WalkEvents(context.Background(), func(row substrate.EventRow) error {
		if row.MessageType == "ghosttrace.events.v1.IngestionEvent" && ingHash == ([32]byte{}) {
			ingHash = row.EventHash
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	if ingHash == ([32]byte{}) {
		t.Fatalf("no IngestionEvent found in test substrate")
	}

	req := httptest.NewRequest(http.MethodGet,
		"/v1/hypotheses/state?formation_event_hash="+hex.EncodeToString(ingHash[:]), nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if got, want := rr.Code, http.StatusNotFound; got != want {
		t.Errorf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "ErrTargetNotFormation") &&
		!strings.Contains(rr.Body.String(), "not a formation") &&
		!strings.Contains(rr.Body.String(), "expected a Cat III formation") {
		t.Errorf("body should indicate not-a-formation reason; got: %s", rr.Body.String())
	}
}

func TestHypothesisStateAuthRequired(t *testing.T) {
	// When WithAuthToken is set, the projection-read endpoint requires
	// a Bearer token (same as /v1/events/*). Probes the auth path.
	sub, _, formationHex := substrateWithBCFormation(t)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub), WithAuthToken("secret"))

	// No Authorization header → 401.
	req := httptest.NewRequest(http.MethodGet,
		"/v1/hypotheses/state?formation_event_hash="+formationHex, nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if got, want := rr.Code, http.StatusUnauthorized; got != want {
		t.Errorf("no auth: status got %d, want %d", got, want)
	}

	// Wrong token → 401.
	req2 := httptest.NewRequest(http.MethodGet,
		"/v1/hypotheses/state?formation_event_hash="+formationHex, nil)
	req2.Header.Set("Authorization", "Bearer wrong-token-of-correct-length-here-")
	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, req2)
	if got, want := rr2.Code, http.StatusUnauthorized; got != want {
		t.Errorf("wrong token: status got %d, want %d", got, want)
	}

	// Correct token → 200.
	req3 := httptest.NewRequest(http.MethodGet,
		"/v1/hypotheses/state?formation_event_hash="+formationHex, nil)
	req3.Header.Set("Authorization", "Bearer secret")
	rr3 := httptest.NewRecorder()
	h.ServeHTTP(rr3, req3)
	if got, want := rr3.Code, http.StatusOK; got != want {
		t.Errorf("correct token: status got %d, want %d; body=%s", got, want, rr3.Body.String())
	}
}
