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

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/derivation"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/hypothesis"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// substrateWithOSDerivedReplay populates a substrate with one
// DeclaredSession + derives it under padded-v1; returns (substrate,
// OS formation hash hex).
func substrateWithOSDerivedReplay(t *testing.T) (*substrate.Substrate, string) {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	in := ingest.New(sub, func() time.Time { return time.Unix(0, 1000) })
	msg := &eventsv1.DeclaredSession{
		DeclaredAt:        1000,
		ActorRef:          "actor-os-http",
		SessionDescriptor: []byte("os-http"),
	}
	if _, err := in.Append(ctx, msg, msg.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if _, err := derivation.DeriveAll(ctx, sub,
		derivation.PaddedV1{PadSeconds: 60},
		func() time.Time { return time.Unix(0, 2000) }); err != nil {
		t.Fatalf("DeriveAll: %v", err)
	}

	var osHash [32]byte
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType == "ghosttrace.events.v1.OperationalSession" {
			osHash = row.EventHash
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	return sub, hex.EncodeToString(osHash[:])
}

func TestReplayOperationalSessionHTTPHappyPath(t *testing.T) {
	sub, osHex := substrateWithOSDerivedReplay(t)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet,
		"/v1/replay/operational-session?target_event_hash="+osHex, nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
	var out map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["match"] != true {
		t.Errorf("match: got %v, want true", out["match"])
	}
	if out["target_event_hash"] != osHex {
		t.Errorf("target_event_hash mismatch")
	}
}

func TestReplayOperationalSessionHTTPMissingParam(t *testing.T) {
	sub, _ := substrateWithOSDerivedReplay(t)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet, "/v1/replay/operational-session", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", rr.Code)
	}
}

func TestReplayOperationalSessionHTTPUnknownTarget(t *testing.T) {
	sub, _ := substrateWithOSDerivedReplay(t)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	bogus := strings.Repeat("ff", 32)
	req := httptest.NewRequest(http.MethodGet,
		"/v1/replay/operational-session?target_event_hash="+bogus, nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want 404; body=%s", rr.Code, rr.Body.String())
	}
}

func TestReplayAllOperationalSessionsHTTPHappyPath(t *testing.T) {
	sub, _ := substrateWithOSDerivedReplay(t)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet, "/v1/replay/operational-sessions", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var out map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["total"].(float64) != 1 {
		t.Errorf("total: got %v, want 1", out["total"])
	}
	if out["matched"].(float64) != 1 {
		t.Errorf("matched: got %v, want 1", out["matched"])
	}
}

// substrateWithBCFormationReplay populates a substrate with a BC
// formation; returns (substrate, formation hash hex).
func substrateWithBCFormationReplay(t *testing.T) (*substrate.Substrate, string) {
	t.Helper()
	sub, _, formationHex := substrateWithBCFormation(t)
	return sub, formationHex
}

func TestReplayFormationHTTPHappyPathBC(t *testing.T) {
	sub, formationHex := substrateWithBCFormationReplay(t)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet,
		"/v1/replay/formation?target_event_hash="+formationHex, nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var out map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["subtype"] != "behavioral_cluster" {
		t.Errorf("subtype: got %v, want behavioral_cluster", out["subtype"])
	}
	if out["match"] != true {
		t.Errorf("match: got %v, want true", out["match"])
	}
}

func TestReplayFormationHTTPAutoDetectsAGSubtype(t *testing.T) {
	// Set up substrate with an AG formation, hit the unified
	// /v1/replay/formation endpoint, verify subtype="automation_group".
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	in := ingest.New(sub, func() time.Time { return time.Unix(0, 1000) })
	for i := 0; i < 5; i++ {
		msg := &eventsv1.DeclaredSession{
			DeclaredAt:        1000 + int64(i)*1000,
			ActorRef:          "actor-ag-http",
			SessionDescriptor: []byte("ag-http"),
		}
		if _, err := in.Append(ctx, msg, msg.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("ingest: %v", err)
		}
	}
	if _, err := hypothesis.FormAutomationGroupAll(ctx, sub,
		hypothesis.UniformCadenceV1{MinObservationCount: 5, MaxCoVThreshold: 0.15},
		func() time.Time { return time.Unix(0, 2000) }); err != nil {
		t.Fatalf("FormAutomationGroupAll: %v", err)
	}

	var agHash [32]byte
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType == "ghosttrace.events.v1.AutomationGroupFormation" {
			agHash = row.EventHash
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}

	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet,
		"/v1/replay/formation?target_event_hash="+hex.EncodeToString(agHash[:]), nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var out map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["subtype"] != "automation_group" {
		t.Errorf("subtype: got %v, want automation_group", out["subtype"])
	}
	if out["match"] != true {
		t.Errorf("match: got %v, want true", out["match"])
	}
}

func TestReplayFormationHTTPRejectsNonFormationTarget(t *testing.T) {
	// Pass a DeclaredSession hash (Cat I, not Cat III) → 404.
	sub, _, _ := substrateWithBCFormation(t)
	var dsHash [32]byte
	if err := sub.WalkEvents(context.Background(), func(row substrate.EventRow) error {
		if row.MessageType == "ghosttrace.events.v1.DeclaredSession" && dsHash == ([32]byte{}) {
			dsHash = row.EventHash
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}

	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet,
		"/v1/replay/formation?target_event_hash="+hex.EncodeToString(dsHash[:]), nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("status: got %d, want 404; body=%s", rr.Code, rr.Body.String())
	}
}

func TestReplayAllFormationsHTTPHappyPath(t *testing.T) {
	sub, _, _ := substrateWithBCFormation(t)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet, "/v1/replay/formations", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var out map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	bc, ok := out["behavioral_cluster"].(map[string]interface{})
	if !ok {
		t.Fatalf("behavioral_cluster section missing")
	}
	if bc["total"].(float64) != 1 {
		t.Errorf("BC total: got %v, want 1", bc["total"])
	}
	if bc["matched"].(float64) != 1 {
		t.Errorf("BC matched: got %v, want 1", bc["matched"])
	}
}

func TestReplayAllFormationsHTTPSubtypeFilter(t *testing.T) {
	sub, _, _ := substrateWithBCFormation(t)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	// Filter to AG → BC section should be absent; AG section should be
	// present with total=0.
	req := httptest.NewRequest(http.MethodGet,
		"/v1/replay/formations?subtype=automation_group", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200; body=%s", rr.Code, rr.Body.String())
	}
	var out map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, present := out["behavioral_cluster"]; present {
		t.Errorf("behavioral_cluster should be absent when subtype=automation_group")
	}
	ag, ok := out["automation_group"].(map[string]interface{})
	if !ok {
		t.Fatalf("automation_group section missing")
	}
	if ag["total"].(float64) != 0 {
		t.Errorf("AG total: got %v, want 0 (no AG formations in this substrate)", ag["total"])
	}
}

func TestReplayAllFormationsHTTPInvalidSubtype(t *testing.T) {
	sub, _, _ := substrateWithBCFormation(t)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub))

	req := httptest.NewRequest(http.MethodGet,
		"/v1/replay/formations?subtype=not_a_subtype", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", rr.Code)
	}
}

func TestReplayHTTPRequiresSubstrate(t *testing.T) {
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil) // no WithSubstrate

	for _, path := range []string{
		"/v1/replay/operational-session",
		"/v1/replay/operational-sessions",
		"/v1/replay/formation",
		"/v1/replay/formations",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusServiceUnavailable {
			t.Errorf("%s without substrate: status got %d, want 503", path, rr.Code)
		}
	}
}

func TestReplayHTTPRejectsNonGet(t *testing.T) {
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil)

	for _, path := range []string{
		"/v1/replay/operational-session",
		"/v1/replay/operational-sessions",
		"/v1/replay/formation",
		"/v1/replay/formations",
	} {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s POST: status got %d, want 405", path, rr.Code)
		}
	}
}

func TestReplayHTTPAuthRequired(t *testing.T) {
	sub, _ := substrateWithOSDerivedReplay(t)
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil, WithSubstrate(sub), WithAuthToken("secret"))

	req := httptest.NewRequest(http.MethodGet, "/v1/replay/operational-sessions", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("no token: status got %d, want 401", rr.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/v1/replay/operational-sessions", nil)
	req2.Header.Set("Authorization", "Bearer secret")
	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Errorf("correct token: status got %d, want 200", rr2.Code)
	}
}
