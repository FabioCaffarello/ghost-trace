package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/hypothesis"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/ingest"
	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/substrate"
)

// formedBehavioralCluster populates a substrate with two actors
// sharing a session descriptor + runs FormAll under
// SessionDescriptorSharedV1 with min-cluster-size=2, returning the
// substrate and the resulting formation event hash. Used by T4 promote
// tests as the precondition that a BehavioralClusterFormation exists.
func formedBehavioralCluster(t *testing.T) (*substrate.Substrate, [32]byte) {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	in := ingest.New(sub, time.Now)
	for i, actor := range []string{"actor-a-t4", "actor-b-t4"} {
		msg := &eventsv1.DeclaredSession{
			DeclaredAt:        int64(1000 + i),
			ActorRef:          actor,
			SessionDescriptor: []byte("alpha-t4"),
		}
		if _, err := in.Append(ctx, msg, msg.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("Append %d: %v", i, err)
		}
	}

	rep, err := hypothesis.FormAll(ctx, sub, hypothesis.SessionDescriptorSharedV1{MinClusterSize: 2}, func() time.Time { return time.Unix(0, 100000) })
	if err != nil {
		t.Fatalf("FormAll: %v", err)
	}
	if rep.NewlyFormed != 1 {
		t.Fatalf("expected 1 formation; got %d", rep.NewlyFormed)
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
	return sub, formationHash
}

// realT4DoAppend constructs a doAppend that backs onto a real
// Ingester against sub. Used so the audit + IngestionEvent actually
// commit. (Same shape as realDoAppend in admin_test.go but local
// to T4 lifecycle tests for clarity.)
func realT4DoAppend(t *testing.T, sub *substrate.Substrate) AppendFunc {
	t.Helper()
	in := ingest.New(sub, time.Now)
	return func(ctx context.Context, msg proto.Message, eventTime int64, env ingest.Envelope) (ingest.AppendReport, error) {
		return in.Append(ctx, msg, eventTime, env)
	}
}

func TestT4PromoteBehavioralClusterHappyPath(t *testing.T) {
	sub, formationHash := formedBehavioralCluster(t)
	h := MustNew(realT4DoAppend(t, sub), nil, WithSubstrate(sub))

	body, err := proto.Marshal(&eventsv1.BehavioralClusterPromotion{
		FormationEventHash: formationHash[:],
		PromotedAt:         200000,
		CadenceSeconds:     3600,
		Reason:             "T4 pilot test",
	})
	if err != nil {
		t.Fatalf("proto.Marshal: %v", err)
	}

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/hypotheses/behavioral-cluster/promote", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
	var out promoteResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.PromotionEventHash == "" {
		t.Errorf("PromotionEventHash: empty")
	}
	if out.IngestionEventHash == "" {
		t.Errorf("IngestionEventHash: empty (HTTP T4 must always pair an IngestionEvent)")
	}
	if out.AlreadyPromoted {
		t.Errorf("AlreadyPromoted: got true, want false (first promotion)")
	}
}

func TestT4PromoteBehavioralClusterIdempotent(t *testing.T) {
	sub, formationHash := formedBehavioralCluster(t)
	h := MustNew(realT4DoAppend(t, sub), nil, WithSubstrate(sub))

	body, _ := proto.Marshal(&eventsv1.BehavioralClusterPromotion{
		FormationEventHash: formationHash[:],
		PromotedAt:         200000,
		CadenceSeconds:     3600,
		Reason:             "idempotent test",
	})

	// First call.
	rr1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodPost, "/v1/hypotheses/behavioral-cluster/promote", bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusOK {
		t.Fatalf("first call: status %d; body=%s", rr1.Code, rr1.Body.String())
	}

	// Second call with identical body.
	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/v1/hypotheses/behavioral-cluster/promote", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("second call: status %d; body=%s", rr2.Code, rr2.Body.String())
	}

	var out2 promoteResponse
	if err := json.Unmarshal(rr2.Body.Bytes(), &out2); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !out2.AlreadyPromoted {
		t.Errorf("AlreadyPromoted: got false, want true (second call with identical body)")
	}
}

func TestT4PromoteBehavioralClusterRejectsNonPost(t *testing.T) {
	sub, _ := formedBehavioralCluster(t)
	h := MustNew(realT4DoAppend(t, sub), nil, WithSubstrate(sub))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/hypotheses/behavioral-cluster/promote", nil)
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusMethodNotAllowed; got != want {
		t.Errorf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
}

func TestT4PromoteBehavioralClusterRejectsWrongContentType(t *testing.T) {
	sub, formationHash := formedBehavioralCluster(t)
	h := MustNew(realT4DoAppend(t, sub), nil, WithSubstrate(sub))

	body, _ := proto.Marshal(&eventsv1.BehavioralClusterPromotion{
		FormationEventHash: formationHash[:],
		PromotedAt:         200000,
		CadenceSeconds:     3600,
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/hypotheses/behavioral-cluster/promote", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusUnsupportedMediaType; got != want {
		t.Errorf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
}

func TestT4PromoteBehavioralClusterRejectsBadHashLength(t *testing.T) {
	sub, _ := formedBehavioralCluster(t)
	h := MustNew(realT4DoAppend(t, sub), nil, WithSubstrate(sub))

	body, _ := proto.Marshal(&eventsv1.BehavioralClusterPromotion{
		FormationEventHash: []byte("too-short"),
		PromotedAt:         200000,
		CadenceSeconds:     3600,
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/hypotheses/behavioral-cluster/promote", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusBadRequest; got != want {
		t.Errorf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
}

func TestT4PromoteBehavioralClusterRejectsNonPositiveCadence(t *testing.T) {
	sub, formationHash := formedBehavioralCluster(t)
	h := MustNew(realT4DoAppend(t, sub), nil, WithSubstrate(sub))

	body, _ := proto.Marshal(&eventsv1.BehavioralClusterPromotion{
		FormationEventHash: formationHash[:],
		PromotedAt:         200000,
		CadenceSeconds:     0,
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/hypotheses/behavioral-cluster/promote", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusBadRequest; got != want {
		t.Errorf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
}

func TestT4PromoteBehavioralClusterRejectsUnknownFormation(t *testing.T) {
	sub, _ := formedBehavioralCluster(t)
	h := MustNew(realT4DoAppend(t, sub), nil, WithSubstrate(sub))

	// All-zeros hash never matches a row.
	var nilHash [32]byte
	body, _ := proto.Marshal(&eventsv1.BehavioralClusterPromotion{
		FormationEventHash: nilHash[:],
		PromotedAt:         200000,
		CadenceSeconds:     3600,
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/hypotheses/behavioral-cluster/promote", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusNotFound; got != want {
		t.Errorf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
}

func TestT4PromoteBehavioralClusterRejectsWrongTypeFormationHash(t *testing.T) {
	// Use a substrate row that exists but is NOT a
	// BehavioralClusterFormation — e.g., a DeclaredSession hash.
	sub, _ := formedBehavioralCluster(t)
	var declaredHash [32]byte
	ctx := context.Background()
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType == "ghosttrace.events.v1.DeclaredSession" && declaredHash == ([32]byte{}) {
			declaredHash = row.EventHash
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	if declaredHash == ([32]byte{}) {
		t.Fatal("no DeclaredSession row found for wrong-type test")
	}

	h := MustNew(realT4DoAppend(t, sub), nil, WithSubstrate(sub))

	body, _ := proto.Marshal(&eventsv1.BehavioralClusterPromotion{
		FormationEventHash: declaredHash[:],
		PromotedAt:         200000,
		CadenceSeconds:     3600,
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/hypotheses/behavioral-cluster/promote", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusNotFound; got != want {
		t.Errorf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
}

func TestT4PromoteBehavioralClusterReturns503WithoutSubstrate(t *testing.T) {
	doAppend, _ := stubAppendFunc(nil)
	h := MustNew(doAppend, nil)

	body, _ := proto.Marshal(&eventsv1.BehavioralClusterPromotion{
		PromotedAt:     200000,
		CadenceSeconds: 3600,
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/hypotheses/behavioral-cluster/promote", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusServiceUnavailable; got != want {
		t.Errorf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
}

func TestT4PromoteMultiTierRequiresConstitutionalActToken(t *testing.T) {
	sub, formationHash := formedBehavioralCluster(t)
	h := MustNew(realT4DoAppend(t, sub), nil,
		WithSubstrate(sub),
		WithAuthTierToken(TierProducer, "prod-token"),
	)

	body, _ := proto.Marshal(&eventsv1.BehavioralClusterPromotion{
		FormationEventHash: formationHash[:],
		PromotedAt:         200000,
		CadenceSeconds:     3600,
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/hypotheses/behavioral-cluster/promote", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Authorization", "Bearer prod-token")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusUnauthorized; got != want {
		t.Errorf("status: got %d, want %d (T4 must require constitutional-act tier; body: %s)",
			got, want, rr.Body.String())
	}
}

func TestT4PromoteMultiTierConstitutionalActTokenAuthorizes(t *testing.T) {
	sub, formationHash := formedBehavioralCluster(t)
	h := MustNew(realT4DoAppend(t, sub), nil,
		WithSubstrate(sub),
		WithAuthTierToken(TierConstitutionalAct, "ca-token"),
	)

	body, _ := proto.Marshal(&eventsv1.BehavioralClusterPromotion{
		FormationEventHash: formationHash[:],
		PromotedAt:         200000,
		CadenceSeconds:     3600,
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/hypotheses/behavioral-cluster/promote", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Authorization", "Bearer ca-token")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Errorf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
}

// formedAutomationGroup populates a substrate with 5 uniform-cadence
// NetworkEvents from a single actor; FormAutomationGroupAll matches
// UniformCadenceV1 with MinObservationCount=5; returns the formation
// hash.
func formedAutomationGroup(t *testing.T) (*substrate.Substrate, [32]byte) {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	in := ingest.New(sub, time.Now)
	gap := int64(60 * 1e9)
	for i := int64(0); i < 5; i++ {
		msg := &eventsv1.DeclaredSession{
			DeclaredAt:        1000 + i*gap,
			ActorRef:          "bot-ag-t4",
			SessionDescriptor: []byte("ag-t4-descriptor"),
		}
		if _, err := in.Append(ctx, msg, msg.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("Append %d: %v", i, err)
		}
	}

	rep, err := hypothesis.FormAutomationGroupAll(ctx, sub,
		hypothesis.UniformCadenceV1{MinObservationCount: 5, MaxCoVThreshold: 0.5},
		func() time.Time { return time.Unix(0, 1) })
	if err != nil {
		t.Fatalf("FormAutomationGroupAll: %v", err)
	}
	if rep.NewlyFormed != 1 {
		t.Fatalf("expected 1 AG formation; got %d", rep.NewlyFormed)
	}

	var formationHash [32]byte
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType == "ghosttrace.events.v1.AutomationGroupFormation" {
			formationHash = row.EventHash
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	return sub, formationHash
}

// formedCampaignHypothesis populates a substrate with 3 DeclaredSessions
// sharing a campaign descriptor within MaxIntraEventGapSeconds; returns
// the formation hash.
func formedCampaignHypothesis(t *testing.T) (*substrate.Substrate, [32]byte) {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	in := ingest.New(sub, time.Now)
	gap := int64(60 * 1e9)
	for i := int64(0); i < 3; i++ {
		msg := &eventsv1.DeclaredSession{
			DeclaredAt:        1000 + i*gap,
			ActorRef:          "actor-campaign-t4",
			SessionDescriptor: []byte("campaign-t4"),
		}
		if _, err := in.Append(ctx, msg, msg.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("Append %d: %v", i, err)
		}
	}

	rep, err := hypothesis.FormCampaignHypothesisAll(ctx, sub,
		hypothesis.TemporalDescriptorCohortV1{MinCampaignSize: 3, MaxIntraEventGapSeconds: 300},
		func() time.Time { return time.Unix(0, 1) })
	if err != nil {
		t.Fatalf("FormCampaignHypothesisAll: %v", err)
	}
	if rep.NewlyFormed != 1 {
		t.Fatalf("expected 1 CH formation; got %d", rep.NewlyFormed)
	}

	var formationHash [32]byte
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType == "ghosttrace.events.v1.CampaignHypothesisFormation" {
			formationHash = row.EventHash
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	return sub, formationHash
}

// formedCoordinationRing populates a substrate with three actors
// repeatedly co-occurring on the same descriptor within the window
// → triangle of edges → one ring formation.
func formedCoordinationRing(t *testing.T) (*substrate.Substrate, [32]byte) {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	in := ingest.New(sub, time.Now)
	gap := int64(60 * 1e9)
	for round := int64(0); round < 3; round++ {
		base := 1000 + round*5*gap
		for k, actor := range []string{"actor-cr-a", "actor-cr-b", "actor-cr-c"} {
			msg := &eventsv1.DeclaredSession{
				DeclaredAt:        base + int64(k)*gap,
				ActorRef:          actor,
				SessionDescriptor: []byte("shared-cr-t4"),
			}
			if _, err := in.Append(ctx, msg, msg.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
				t.Fatalf("Append: %v", err)
			}
		}
	}

	rep, err := hypothesis.FormCoordinationRingAll(ctx, sub,
		hypothesis.CoOccurrenceWindowV1{MinEdgeSupport: 3, MaxWindowSeconds: 600},
		func() time.Time { return time.Unix(0, 1) })
	if err != nil {
		t.Fatalf("FormCoordinationRingAll: %v", err)
	}
	if rep.NewlyFormed != 1 {
		t.Fatalf("expected 1 CR formation; got %d", rep.NewlyFormed)
	}

	var formationHash [32]byte
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType == "ghosttrace.events.v1.CoordinationRingFormation" {
			formationHash = row.EventHash
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	return sub, formationHash
}

func TestT4PromoteAutomationGroupHappyPath(t *testing.T) {
	sub, formationHash := formedAutomationGroup(t)
	h := MustNew(realT4DoAppend(t, sub), nil, WithSubstrate(sub))

	body, _ := proto.Marshal(&eventsv1.AutomationGroupPromotion{
		FormationEventHash: formationHash[:],
		PromotedAt:         200000,
		CadenceSeconds:     3600,
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/hypotheses/automation-group/promote", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
	var out promoteResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.PromotionEventHash == "" || out.IngestionEventHash == "" {
		t.Errorf("hashes: promotion=%q ingestion=%q (both should be non-empty)", out.PromotionEventHash, out.IngestionEventHash)
	}
}

func TestT4PromoteAutomationGroupIdempotent(t *testing.T) {
	sub, formationHash := formedAutomationGroup(t)
	h := MustNew(realT4DoAppend(t, sub), nil, WithSubstrate(sub))

	body, _ := proto.Marshal(&eventsv1.AutomationGroupPromotion{
		FormationEventHash: formationHash[:],
		PromotedAt:         200000,
		CadenceSeconds:     3600,
	})

	rr1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodPost, "/v1/hypotheses/automation-group/promote", bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusOK {
		t.Fatalf("first call: status %d; body=%s", rr1.Code, rr1.Body.String())
	}

	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/v1/hypotheses/automation-group/promote", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr2, req2)

	var out2 promoteResponse
	_ = json.Unmarshal(rr2.Body.Bytes(), &out2)
	if !out2.AlreadyPromoted {
		t.Errorf("second call AlreadyPromoted: got false, want true")
	}
}

func TestT4PromoteCampaignHypothesisHappyPath(t *testing.T) {
	sub, formationHash := formedCampaignHypothesis(t)
	h := MustNew(realT4DoAppend(t, sub), nil, WithSubstrate(sub))

	body, _ := proto.Marshal(&eventsv1.CampaignHypothesisPromotion{
		FormationEventHash: formationHash[:],
		PromotedAt:         200000,
		CadenceSeconds:     3600,
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/hypotheses/campaign-hypothesis/promote", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
	var out promoteResponse
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	if out.PromotionEventHash == "" || out.IngestionEventHash == "" {
		t.Errorf("hashes: promotion=%q ingestion=%q", out.PromotionEventHash, out.IngestionEventHash)
	}
}

func TestT4PromoteCoordinationRingHappyPath(t *testing.T) {
	sub, formationHash := formedCoordinationRing(t)
	h := MustNew(realT4DoAppend(t, sub), nil, WithSubstrate(sub))

	body, _ := proto.Marshal(&eventsv1.CoordinationRingPromotion{
		FormationEventHash: formationHash[:],
		PromotedAt:         200000,
		CadenceSeconds:     3600,
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/hypotheses/coordination-ring/promote", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
	var out promoteResponse
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	if out.PromotionEventHash == "" || out.IngestionEventHash == "" {
		t.Errorf("hashes: promotion=%q ingestion=%q", out.PromotionEventHash, out.IngestionEventHash)
	}
}

func TestT4PromoteWrongSubtypeFormationHashRejected(t *testing.T) {
	// AG promote endpoint with a BC formation hash → 404 (cross-subtype
	// integrity per §2.5 BC5).
	sub, bcFormationHash := formedBehavioralCluster(t)
	h := MustNew(realT4DoAppend(t, sub), nil, WithSubstrate(sub))

	body, _ := proto.Marshal(&eventsv1.AutomationGroupPromotion{
		FormationEventHash: bcFormationHash[:],
		PromotedAt:         200000,
		CadenceSeconds:     3600,
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/hypotheses/automation-group/promote", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusNotFound; got != want {
		t.Errorf("status: got %d, want %d (BC formation in AG endpoint must 404; body: %s)",
			got, want, rr.Body.String())
	}
}

// promotedBehavioralCluster: forms a BC + promotes it; returns the
// promotion event hash for use as the demote target.
func promotedBehavioralCluster(t *testing.T) (*substrate.Substrate, [32]byte) {
	t.Helper()
	sub, formationHash := formedBehavioralCluster(t)
	rep, err := hypothesis.Promote(context.Background(), sub,
		hypothesis.PromoteOptions{
			FormationEventHash: formationHash,
			PromotedAt:         300000,
			CadenceSeconds:     3600,
			Reason:             "demote-test",
		},
		func() time.Time { return time.Unix(0, 400000) })
	if err != nil {
		t.Fatalf("Promote: %v", err)
	}
	var promotionHash [32]byte
	hexBytes := []byte(rep.PromotionEventHashHex)
	if len(hexBytes) != 64 {
		t.Fatalf("unexpected promotion hex length: %d", len(hexBytes))
	}
	for i := 0; i < 32; i++ {
		hi := hexNibble(hexBytes[2*i])
		lo := hexNibble(hexBytes[2*i+1])
		promotionHash[i] = byte(hi<<4 | lo)
	}
	return sub, promotionHash
}

// promotedAutomationGroup forms + promotes an AG; returns promotion hash.
func promotedAutomationGroup(t *testing.T) (*substrate.Substrate, [32]byte) {
	t.Helper()
	sub, formationHash := formedAutomationGroup(t)
	rep, err := hypothesis.PromoteAutomationGroup(context.Background(), sub,
		hypothesis.AutomationGroupPromoteOptions{
			FormationEventHash: formationHash,
			PromotedAt:         300000,
			CadenceSeconds:     3600,
		},
		func() time.Time { return time.Unix(0, 400000) })
	if err != nil {
		t.Fatalf("PromoteAutomationGroup: %v", err)
	}
	return sub, hexToHash(t, rep.PromotionEventHashHex)
}

// promotedCampaignHypothesis forms + promotes a CH; returns promotion hash.
func promotedCampaignHypothesis(t *testing.T) (*substrate.Substrate, [32]byte) {
	t.Helper()
	sub, formationHash := formedCampaignHypothesis(t)
	rep, err := hypothesis.PromoteCampaignHypothesis(context.Background(), sub,
		hypothesis.CampaignHypothesisPromoteOptions{
			FormationEventHash: formationHash,
			PromotedAt:         300000,
			CadenceSeconds:     3600,
		},
		func() time.Time { return time.Unix(0, 400000) })
	if err != nil {
		t.Fatalf("PromoteCampaignHypothesis: %v", err)
	}
	return sub, hexToHash(t, rep.PromotionEventHashHex)
}

// promotedCoordinationRing forms + promotes a CR; returns promotion hash.
func promotedCoordinationRing(t *testing.T) (*substrate.Substrate, [32]byte) {
	t.Helper()
	sub, formationHash := formedCoordinationRing(t)
	rep, err := hypothesis.PromoteCoordinationRing(context.Background(), sub,
		hypothesis.CoordinationRingPromoteOptions{
			FormationEventHash: formationHash,
			PromotedAt:         300000,
			CadenceSeconds:     3600,
		},
		func() time.Time { return time.Unix(0, 400000) })
	if err != nil {
		t.Fatalf("PromoteCoordinationRing: %v", err)
	}
	return sub, hexToHash(t, rep.PromotionEventHashHex)
}

func hexNibble(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return 0
}

func hexToHash(t *testing.T, hex string) [32]byte {
	t.Helper()
	if len(hex) != 64 {
		t.Fatalf("unexpected hex length: %d", len(hex))
	}
	var out [32]byte
	for i := 0; i < 32; i++ {
		hi := hexNibble(hex[2*i])
		lo := hexNibble(hex[2*i+1])
		out[i] = byte(hi<<4 | lo)
	}
	return out
}

func TestT4DemoteBehavioralClusterHappyPath(t *testing.T) {
	sub, promotionHash := promotedBehavioralCluster(t)
	h := MustNew(realT4DoAppend(t, sub), nil, WithSubstrate(sub))

	body, _ := proto.Marshal(&eventsv1.BehavioralClusterDemotion{
		PromotionEventHash: promotionHash[:],
		DemotedAt:          5000000000000,
		Reason:             "T4 demote test",
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/hypotheses/behavioral-cluster/demote", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
	var out demoteResponse
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	if out.DemotionEventHash == "" || out.IngestionEventHash == "" {
		t.Errorf("hashes: demotion=%q ingestion=%q (both should be non-empty)", out.DemotionEventHash, out.IngestionEventHash)
	}
	// CadenceElapsedSeconds = (5000000000000 - 300000) / 1e9 ≈ 4999s,
	// which is > 3600 cadence_seconds → CadenceSatisfied=true.
	if !out.CadenceSatisfied {
		t.Errorf("CadenceSatisfied: got false, want true (5000s > 3600s)")
	}
}

func TestT4DemoteAutomationGroupHappyPath(t *testing.T) {
	sub, promotionHash := promotedAutomationGroup(t)
	h := MustNew(realT4DoAppend(t, sub), nil, WithSubstrate(sub))

	body, _ := proto.Marshal(&eventsv1.AutomationGroupDemotion{
		PromotionEventHash: promotionHash[:],
		DemotedAt:          5000000000000,
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/hypotheses/automation-group/demote", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
	var out demoteResponse
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	if out.DemotionEventHash == "" || out.IngestionEventHash == "" {
		t.Errorf("hashes: demotion=%q ingestion=%q", out.DemotionEventHash, out.IngestionEventHash)
	}
}

func TestT4DemoteCampaignHypothesisHappyPath(t *testing.T) {
	sub, promotionHash := promotedCampaignHypothesis(t)
	h := MustNew(realT4DoAppend(t, sub), nil, WithSubstrate(sub))

	body, _ := proto.Marshal(&eventsv1.CampaignHypothesisDemotion{
		PromotionEventHash: promotionHash[:],
		DemotedAt:          5000000000000,
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/hypotheses/campaign-hypothesis/demote", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
	var out demoteResponse
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	if out.DemotionEventHash == "" || out.IngestionEventHash == "" {
		t.Errorf("hashes: demotion=%q ingestion=%q", out.DemotionEventHash, out.IngestionEventHash)
	}
}

func TestT4DemoteCoordinationRingHappyPath(t *testing.T) {
	sub, promotionHash := promotedCoordinationRing(t)
	h := MustNew(realT4DoAppend(t, sub), nil, WithSubstrate(sub))

	body, _ := proto.Marshal(&eventsv1.CoordinationRingDemotion{
		PromotionEventHash: promotionHash[:],
		DemotedAt:          5000000000000,
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/hypotheses/coordination-ring/demote", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
	var out demoteResponse
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	if out.DemotionEventHash == "" || out.IngestionEventHash == "" {
		t.Errorf("hashes: demotion=%q ingestion=%q", out.DemotionEventHash, out.IngestionEventHash)
	}
}

func TestT4DemoteRejectsFormationHashAsPromotion(t *testing.T) {
	// Demote endpoint with a formation hash (not promotion) → 404
	// (§2.5 BC5 wrong-type gate).
	sub, formationHash := formedBehavioralCluster(t)
	h := MustNew(realT4DoAppend(t, sub), nil, WithSubstrate(sub))

	body, _ := proto.Marshal(&eventsv1.BehavioralClusterDemotion{
		PromotionEventHash: formationHash[:],
		DemotedAt:          5000000000000,
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/hypotheses/behavioral-cluster/demote", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusNotFound; got != want {
		t.Errorf("status: got %d, want %d (formation in demote endpoint must 404; body: %s)",
			got, want, rr.Body.String())
	}
}

func TestT4DemoteRejectsBadHashLength(t *testing.T) {
	sub, _ := formedBehavioralCluster(t)
	h := MustNew(realT4DoAppend(t, sub), nil, WithSubstrate(sub))

	body, _ := proto.Marshal(&eventsv1.BehavioralClusterDemotion{
		PromotionEventHash: []byte("too-short"),
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/hypotheses/behavioral-cluster/demote", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusBadRequest; got != want {
		t.Errorf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
}

// T4 form tests per §0111.

// formSubstrateBC populates a substrate with two actors sharing a
// descriptor so SessionDescriptorSharedV1{MinClusterSize:2} forms 1.
func formSubstrateBC(t *testing.T) *substrate.Substrate {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })
	in := ingest.New(sub, time.Now)
	for i, actor := range []string{"actor-a-form", "actor-b-form"} {
		msg := &eventsv1.DeclaredSession{
			DeclaredAt:        int64(1000 + i),
			ActorRef:          actor,
			SessionDescriptor: []byte("form-test"),
		}
		if _, err := in.Append(ctx, msg, msg.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
			t.Fatalf("Append %d: %v", i, err)
		}
	}
	return sub
}

func TestT4FormBehavioralClusterHappyPath(t *testing.T) {
	sub := formSubstrateBC(t)
	h := MustNew(realT4DoAppend(t, sub), nil, WithSubstrate(sub))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/hypotheses/behavioral-cluster/form?min_cluster_size=2", nil)
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
	var out formResponse
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	if out.NewlyFormed != 1 {
		t.Errorf("NewlyFormed: got %d, want 1", out.NewlyFormed)
	}
}

func TestT4FormBehavioralClusterIdempotent(t *testing.T) {
	sub := formSubstrateBC(t)
	h := MustNew(realT4DoAppend(t, sub), nil, WithSubstrate(sub))

	// First call: forms 1.
	rr1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodPost, "/v1/hypotheses/behavioral-cluster/form?min_cluster_size=2", nil)
	h.ServeHTTP(rr1, req1)
	if rr1.Code != http.StatusOK {
		t.Fatalf("first call: status %d; body=%s", rr1.Code, rr1.Body.String())
	}

	// Second call: idempotent (already formed).
	rr2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/v1/hypotheses/behavioral-cluster/form?min_cluster_size=2", nil)
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("second call: status %d", rr2.Code)
	}
	var out2 formResponse
	_ = json.Unmarshal(rr2.Body.Bytes(), &out2)
	if out2.NewlyFormed != 0 || out2.AlreadyFormed != 1 {
		t.Errorf("second call: NewlyFormed=%d AlreadyFormed=%d, want 0 + 1", out2.NewlyFormed, out2.AlreadyFormed)
	}
}

func TestT4FormBehavioralClusterRejectsMissingParam(t *testing.T) {
	sub := formSubstrateBC(t)
	h := MustNew(realT4DoAppend(t, sub), nil, WithSubstrate(sub))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/hypotheses/behavioral-cluster/form", nil)
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusBadRequest; got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
}

func TestT4FormBehavioralClusterRejectsBadParam(t *testing.T) {
	sub := formSubstrateBC(t)
	h := MustNew(realT4DoAppend(t, sub), nil, WithSubstrate(sub))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/hypotheses/behavioral-cluster/form?min_cluster_size=abc", nil)
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusBadRequest; got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
}

// T4 split tests per §0110.

func TestT4SplitBehavioralClusterHappyPath(t *testing.T) {
	sub, antecedent, succ1, succ2 := threeFormedBC(t)
	h := MustNew(realT4DoAppend(t, sub), nil, WithSubstrate(sub))

	body, _ := proto.Marshal(&eventsv1.BehavioralClusterSplit{
		AntecedentFormationEventHash:  antecedent[:],
		SuccessorFormationEventHashes: [][]byte{succ1[:], succ2[:]},
		SplitAt:                       600000,
		Reason:                        "T4 split test",
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/hypotheses/behavioral-cluster/split", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
	var out splitResponse
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	if out.SplitEventHash == "" || out.IngestionEventHash == "" {
		t.Errorf("hashes: split=%q ingestion=%q", out.SplitEventHash, out.IngestionEventHash)
	}
}

func TestT4SplitRejectsInsufficientSuccessors(t *testing.T) {
	sub, antecedent, succ1, _ := threeFormedBC(t)
	h := MustNew(realT4DoAppend(t, sub), nil, WithSubstrate(sub))

	body, _ := proto.Marshal(&eventsv1.BehavioralClusterSplit{
		AntecedentFormationEventHash:  antecedent[:],
		SuccessorFormationEventHashes: [][]byte{succ1[:]}, // only 1
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/hypotheses/behavioral-cluster/split", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusBadRequest; got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
}

func TestT4SplitRejectsAntecedentInSuccessors(t *testing.T) {
	sub, antecedent, succ1, _ := threeFormedBC(t)
	h := MustNew(realT4DoAppend(t, sub), nil, WithSubstrate(sub))

	body, _ := proto.Marshal(&eventsv1.BehavioralClusterSplit{
		AntecedentFormationEventHash:  antecedent[:],
		SuccessorFormationEventHashes: [][]byte{antecedent[:], succ1[:]}, // antecedent reused
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/hypotheses/behavioral-cluster/split", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusBadRequest; got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
}

// T4 merge tests per §0109.

// threeFormedBC forms three distinct BehavioralCluster hypotheses
// (different descriptors) in the same substrate; returns their three
// formation hashes for use as merge antecedents + produced.
func threeFormedBC(t *testing.T) (*substrate.Substrate, [32]byte, [32]byte, [32]byte) {
	t.Helper()
	dir := t.TempDir()
	ctx := context.Background()
	sub, err := substrate.Open(ctx, filepath.Join(dir, "test.db"), filepath.Join(dir, "blobs"))
	if err != nil {
		t.Fatalf("substrate.Open: %v", err)
	}
	t.Cleanup(func() { _ = sub.Close() })

	in := ingest.New(sub, time.Now)
	for i, descriptor := range []string{"merge-a", "merge-b", "merge-produced"} {
		for k, actor := range []string{"actor-a-" + descriptor, "actor-b-" + descriptor} {
			msg := &eventsv1.DeclaredSession{
				DeclaredAt:        int64(1000+i*100) + int64(k),
				ActorRef:          actor,
				SessionDescriptor: []byte(descriptor),
			}
			if _, err := in.Append(ctx, msg, msg.DeclaredAt, ingest.Envelope{Channel: "test"}); err != nil {
				t.Fatalf("Append: %v", err)
			}
		}
	}

	rep, err := hypothesis.FormAll(ctx, sub, hypothesis.SessionDescriptorSharedV1{MinClusterSize: 2}, func() time.Time { return time.Unix(0, 100000) })
	if err != nil {
		t.Fatalf("FormAll: %v", err)
	}
	if rep.NewlyFormed != 3 {
		t.Fatalf("expected 3 formations; got %d", rep.NewlyFormed)
	}

	var hashes [3][32]byte
	idx := 0
	if err := sub.WalkEvents(ctx, func(row substrate.EventRow) error {
		if row.MessageType == "ghosttrace.events.v1.BehavioralClusterFormation" && idx < 3 {
			hashes[idx] = row.EventHash
			idx++
		}
		return nil
	}); err != nil {
		t.Fatalf("WalkEvents: %v", err)
	}
	return sub, hashes[0], hashes[1], hashes[2]
}

func TestT4MergeBehavioralClusterHappyPath(t *testing.T) {
	sub, a, b, produced := threeFormedBC(t)
	h := MustNew(realT4DoAppend(t, sub), nil, WithSubstrate(sub))

	body, _ := proto.Marshal(&eventsv1.BehavioralClusterMerge{
		AntecedentFormationEventHashes: [][]byte{a[:], b[:]},
		ProducedFormationEventHash:     produced[:],
		MergedAt:                       500000,
		Reason:                         "T4 merge test",
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/hypotheses/behavioral-cluster/merge", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
	var out mergeResponse
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	if out.MergeEventHash == "" || out.IngestionEventHash == "" {
		t.Errorf("hashes: merge=%q ingestion=%q", out.MergeEventHash, out.IngestionEventHash)
	}
}

func TestT4MergeRejectsIdenticalAntecedents(t *testing.T) {
	sub, a, _, produced := threeFormedBC(t)
	h := MustNew(realT4DoAppend(t, sub), nil, WithSubstrate(sub))

	body, _ := proto.Marshal(&eventsv1.BehavioralClusterMerge{
		AntecedentFormationEventHashes: [][]byte{a[:], a[:]}, // identical
		ProducedFormationEventHash:     produced[:],
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/hypotheses/behavioral-cluster/merge", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusBadRequest; got != want {
		t.Errorf("status: got %d, want %d (identical antecedents must 400; body: %s)",
			got, want, rr.Body.String())
	}
}

func TestT4MergeRejectsWrongAntecedentCount(t *testing.T) {
	sub, a, _, produced := threeFormedBC(t)
	h := MustNew(realT4DoAppend(t, sub), nil, WithSubstrate(sub))

	body, _ := proto.Marshal(&eventsv1.BehavioralClusterMerge{
		AntecedentFormationEventHashes: [][]byte{a[:]}, // only 1
		ProducedFormationEventHash:     produced[:],
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/hypotheses/behavioral-cluster/merge", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusBadRequest; got != want {
		t.Errorf("status: got %d, want %d", got, want)
	}
}

// T4 dissolve tests per §0108.

func TestT4DissolveBehavioralClusterHappyPath(t *testing.T) {
	sub, formationHash := formedBehavioralCluster(t)
	h := MustNew(realT4DoAppend(t, sub), nil, WithSubstrate(sub))

	body, _ := proto.Marshal(&eventsv1.BehavioralClusterDissolution{
		FormationEventHash: formationHash[:],
		DissolvedAt:        9000000000000,
		Reason:             "T4 dissolve test",
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/hypotheses/behavioral-cluster/dissolve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
	var out dissolveResponse
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	if out.DissolutionEventHash == "" || out.IngestionEventHash == "" {
		t.Errorf("hashes: dissolution=%q ingestion=%q", out.DissolutionEventHash, out.IngestionEventHash)
	}
}

func TestT4DissolveAutomationGroupHappyPath(t *testing.T) {
	sub, formationHash := formedAutomationGroup(t)
	h := MustNew(realT4DoAppend(t, sub), nil, WithSubstrate(sub))

	body, _ := proto.Marshal(&eventsv1.AutomationGroupDissolution{
		FormationEventHash: formationHash[:],
		DissolvedAt:        9000000000000,
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/hypotheses/automation-group/dissolve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
	var out dissolveResponse
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	if out.DissolutionEventHash == "" || out.IngestionEventHash == "" {
		t.Errorf("hashes: dissolution=%q ingestion=%q", out.DissolutionEventHash, out.IngestionEventHash)
	}
}

func TestT4DissolveCampaignHypothesisHappyPath(t *testing.T) {
	sub, formationHash := formedCampaignHypothesis(t)
	h := MustNew(realT4DoAppend(t, sub), nil, WithSubstrate(sub))

	body, _ := proto.Marshal(&eventsv1.CampaignHypothesisDissolution{
		FormationEventHash: formationHash[:],
		DissolvedAt:        9000000000000,
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/hypotheses/campaign-hypothesis/dissolve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
}

func TestT4DissolveCoordinationRingHappyPath(t *testing.T) {
	sub, formationHash := formedCoordinationRing(t)
	h := MustNew(realT4DoAppend(t, sub), nil, WithSubstrate(sub))

	body, _ := proto.Marshal(&eventsv1.CoordinationRingDissolution{
		FormationEventHash: formationHash[:],
		DissolvedAt:        9000000000000,
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/hypotheses/coordination-ring/dissolve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
}

func TestT4DissolveDirectFromFormation(t *testing.T) {
	// Dissolution does NOT require a prior promotion (lifecycle-
	// semantics distinction: dissolution recognizes NON-EXISTENCE;
	// formation can be dissolved directly). Test that a freshly-
	// formed BC can be dissolved without intermediate promote/demote.
	sub, formationHash := formedBehavioralCluster(t)
	h := MustNew(realT4DoAppend(t, sub), nil, WithSubstrate(sub))

	body, _ := proto.Marshal(&eventsv1.BehavioralClusterDissolution{
		FormationEventHash: formationHash[:],
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/hypotheses/behavioral-cluster/dissolve", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/x-protobuf")
	h.ServeHTTP(rr, req)

	if got, want := rr.Code, http.StatusOK; got != want {
		t.Fatalf("status: got %d, want %d; body=%s", got, want, rr.Body.String())
	}
}

func TestResolveT4ActorFromMTLS(t *testing.T) {
	got := resolveT4Actor(ingest.Envelope{ClientCommonName: "operator-alice"}, TierConstitutionalAct)
	if want := "operator-alice"; got != want {
		t.Errorf("resolveT4Actor with CN: got %q, want %q", got, want)
	}
}

func TestResolveT4ActorFallback(t *testing.T) {
	got := resolveT4Actor(ingest.Envelope{}, TierConstitutionalAct)
	if want := "unattributed-token-constitutional-act"; got != want {
		t.Errorf("resolveT4Actor fallback: got %q, want %q", got, want)
	}
}
