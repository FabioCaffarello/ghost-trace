package signatures

import (
	"bytes"
	"context"
	"testing"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
)

// newNetworkObsForCohort builds a NetworkObservation with envelope
// fields populated (endpoint_ref + observed_at + optional actor_ref)
// and a minimal tcp_fingerprint sub-modality. Sub-modality choice is
// irrelevant per §0182 (signature reads envelope only); tcp_fingerprint
// chosen for fixture consistency with §0161 / §0169 tests.
func newNetworkObsForCohort(actorRef, endpointRef string, observedAt int64) *eventsv1.NetworkObservation {
	return &eventsv1.NetworkObservation{
		ObservedAt:          observedAt,
		ActorRef:            actorRef,
		EndpointRef:         endpointRef,
		CollectorRef:        "test-collector:v1",
		AuthenticationClass: commonv1.AuthenticationClass_AUTHENTICATION_CLASS_SERVER_AUTHENTICATED,
		Modality: &eventsv1.NetworkObservation_TcpFingerprint{
			TcpFingerprint: &eventsv1.NetworkTcpFingerprint{WindowSize: 65535},
		},
	}
}

func TestTemporalEndpointCohortV1_NameAndSubtype(t *testing.T) {
	sig := &TemporalEndpointCohortV1{}
	if sig.Name() != "temporal_endpoint_cohort_v1" {
		t.Errorf("Name: got %q want temporal_endpoint_cohort_v1", sig.Name())
	}
	if sig.Subtype() != HypothesisSubtypeCampaignHypothesis {
		t.Errorf("Subtype: got %v want CampaignHypothesis", sig.Subtype())
	}
}

func TestTemporalEndpointCohortV1_SatisfiesNetworkSignatureInterface(t *testing.T) {
	var _ NetworkSignature = &TemporalEndpointCohortV1{}
}

func TestTemporalEndpointCohortV1_BelowThreshold_NoCandidate(t *testing.T) {
	sig := &TemporalEndpointCohortV1{}
	// 2 events in same bucket; below threshold 3.
	const bucketStart = int64(1716120000_000_000_000) // 2024-05-19 12:00:00 UTC (within minute bucket)
	observations := []*eventsv1.NetworkObservation{
		newNetworkObsForCohort("actor-a", "10.0.0.1:443", bucketStart),
		newNetworkObsForCohort("actor-b", "10.0.0.1:443", bucketStart+10_000_000_000), // +10s, same bucket
	}
	res, err := sig.EvaluateNetwork(context.Background(), observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(res.Candidates) != 0 {
		t.Errorf("expected 0 candidates (2 events < threshold 3), got %d", len(res.Candidates))
	}
}

func TestTemporalEndpointCohortV1_AtThreshold_OneCandidate(t *testing.T) {
	sig := &TemporalEndpointCohortV1{}
	const bucketStart = int64(1716120000_000_000_000)
	observations := []*eventsv1.NetworkObservation{
		newNetworkObsForCohort("actor-a", "10.0.0.1:443", bucketStart),
		newNetworkObsForCohort("actor-b", "10.0.0.1:443", bucketStart+10_000_000_000),
		newNetworkObsForCohort("actor-c", "10.0.0.1:443", bucketStart+20_000_000_000),
	}
	res, err := sig.EvaluateNetwork(context.Background(), observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(res.Candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(res.Candidates))
	}
	c := res.Candidates[0]
	if c.SignatureName != "temporal_endpoint_cohort_v1" {
		t.Errorf("SignatureName: got %q", c.SignatureName)
	}
	if c.HypothesisSubtype != HypothesisSubtypeCampaignHypothesis {
		t.Errorf("HypothesisSubtype: got %v want CampaignHypothesis", c.HypothesisSubtype)
	}
	if !equalStringSlices(c.ActorRefs, []string{"actor-a", "actor-b", "actor-c"}) {
		t.Errorf("ActorRefs: got %v want [actor-a actor-b actor-c] (enrichment metadata)", c.ActorRefs)
	}
	if len(c.SourceHashes) != 3 {
		t.Errorf("SourceHashes count: got %d want 3", len(c.SourceHashes))
	}
	if c.EvidenceCount != 3 {
		t.Errorf("EvidenceCount: got %d want 3", c.EvidenceCount)
	}
	// Hashes sorted ascending per §0139.
	for i := 1; i < len(c.SourceHashes); i++ {
		if bytes.Compare(c.SourceHashes[i-1], c.SourceHashes[i]) >= 0 {
			t.Errorf("SourceHashes not strictly ascending at index %d", i)
		}
	}
	if c.ConfidenceHint < 0.5 || c.ConfidenceHint > 0.9 {
		t.Errorf("ConfidenceHint: got %v want in [0.5, 0.9]", c.ConfidenceHint)
	}
}

// TestTemporalEndpointCohortV1_DifferentEndpoints_NoCrossCluster
// verifies that events targeting DIFFERENT endpoints in the SAME
// time window do NOT cluster.
func TestTemporalEndpointCohortV1_DifferentEndpoints_NoCrossCluster(t *testing.T) {
	sig := &TemporalEndpointCohortV1{}
	const bucketStart = int64(1716120000_000_000_000)
	// 3 events targeting endpoint A; 3 events targeting endpoint B;
	// all within same time bucket. Should produce 2 distinct clusters.
	observations := []*eventsv1.NetworkObservation{
		newNetworkObsForCohort("actor-1", "10.0.0.1:443", bucketStart+1_000_000_000),
		newNetworkObsForCohort("actor-2", "10.0.0.1:443", bucketStart+2_000_000_000),
		newNetworkObsForCohort("actor-3", "10.0.0.1:443", bucketStart+3_000_000_000),
		newNetworkObsForCohort("actor-4", "10.0.0.2:443", bucketStart+4_000_000_000),
		newNetworkObsForCohort("actor-5", "10.0.0.2:443", bucketStart+5_000_000_000),
		newNetworkObsForCohort("actor-6", "10.0.0.2:443", bucketStart+6_000_000_000),
	}
	res, err := sig.EvaluateNetwork(context.Background(), observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(res.Candidates) != 2 {
		t.Fatalf("expected 2 candidates (distinct endpoint clusters), got %d", len(res.Candidates))
	}
}

// TestTemporalEndpointCohortV1_DifferentBuckets_NoCrossCluster
// verifies that events targeting the SAME endpoint in DIFFERENT
// time buckets do NOT cluster.
func TestTemporalEndpointCohortV1_DifferentBuckets_NoCrossCluster(t *testing.T) {
	sig := &TemporalEndpointCohortV1{}
	const bucketStart = int64(1716120000_000_000_000)
	// 3 events in bucket 1, 3 events in bucket 2 (≥ 60s apart);
	// same endpoint.
	observations := []*eventsv1.NetworkObservation{
		newNetworkObsForCohort("actor-1", "10.0.0.1:443", bucketStart+1_000_000_000),
		newNetworkObsForCohort("actor-2", "10.0.0.1:443", bucketStart+2_000_000_000),
		newNetworkObsForCohort("actor-3", "10.0.0.1:443", bucketStart+3_000_000_000),
		// +120s = bucket index increases by 2 (different bucket).
		newNetworkObsForCohort("actor-4", "10.0.0.1:443", bucketStart+120_000_000_000),
		newNetworkObsForCohort("actor-5", "10.0.0.1:443", bucketStart+121_000_000_000),
		newNetworkObsForCohort("actor-6", "10.0.0.1:443", bucketStart+122_000_000_000),
	}
	res, err := sig.EvaluateNetwork(context.Background(), observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(res.Candidates) != 2 {
		t.Fatalf("expected 2 candidates (distinct time bucket clusters), got %d", len(res.Candidates))
	}
}

// TestTemporalEndpointCohortV1_EventCentric_EmptyActorRefs_NotSkipped
// is the structurally distinctive test: events with empty actor_ref
// are NOT skipped (unlike prior F3 signatures); they still count
// toward the event-set cluster.
func TestTemporalEndpointCohortV1_EventCentric_EmptyActorRefs_NotSkipped(t *testing.T) {
	sig := &TemporalEndpointCohortV1{}
	const bucketStart = int64(1716120000_000_000_000)
	// 3 events ALL with empty actor_ref but identical (endpoint, bucket).
	observations := []*eventsv1.NetworkObservation{
		newNetworkObsForCohort("", "10.0.0.1:443", bucketStart),
		newNetworkObsForCohort("", "10.0.0.1:443", bucketStart+10_000_000_000),
		newNetworkObsForCohort("", "10.0.0.1:443", bucketStart+20_000_000_000),
	}
	res, err := sig.EvaluateNetwork(context.Background(), observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	// Should emit candidate (event-centric — actor irrelevant to
	// membership).
	if len(res.Candidates) != 1 {
		t.Fatalf("expected 1 candidate (event-centric clustering), got %d", len(res.Candidates))
	}
	c := res.Candidates[0]
	if len(c.ActorRefs) != 0 {
		t.Errorf("ActorRefs: got %v want empty (no actors observed; enrichment metadata empty)", c.ActorRefs)
	}
	if c.EvidenceCount != 3 {
		t.Errorf("EvidenceCount: got %d want 3", c.EvidenceCount)
	}
	// Confirm NoActor was NOT incremented (event-centric ontology).
	if res.Stats.ObservationsSkippedNoActor != 0 {
		t.Errorf("ObservationsSkippedNoActor: got %d want 0 (event-centric ontology — actor optional)", res.Stats.ObservationsSkippedNoActor)
	}
}

func TestTemporalEndpointCohortV1_EmptyEndpoint_Skipped(t *testing.T) {
	sig := &TemporalEndpointCohortV1{}
	const bucketStart = int64(1716120000_000_000_000)
	obs := newNetworkObsForCohort("actor-a", "", bucketStart)
	res, err := sig.EvaluateNetwork(context.Background(), []*eventsv1.NetworkObservation{obs}, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if res.Stats.ObservationsSkippedWrongModality != 1 {
		t.Errorf("ObservationsSkippedWrongModality: got %d want 1 (empty endpoint_ref)", res.Stats.ObservationsSkippedWrongModality)
	}
}

func TestTemporalEndpointCohortV1_ZeroObservedAt_Skipped(t *testing.T) {
	sig := &TemporalEndpointCohortV1{}
	obs := newNetworkObsForCohort("actor-a", "10.0.0.1:443", 0)
	res, err := sig.EvaluateNetwork(context.Background(), []*eventsv1.NetworkObservation{obs}, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if res.Stats.ObservationsSkippedWrongModality != 1 {
		t.Errorf("ObservationsSkippedWrongModality: got %d want 1 (observed_at == 0)", res.Stats.ObservationsSkippedWrongModality)
	}
}

func TestTemporalEndpointCohortV1_EmptyInput_NoCandidates(t *testing.T) {
	sig := &TemporalEndpointCohortV1{}
	res, err := sig.EvaluateNetwork(context.Background(), []*eventsv1.NetworkObservation{}, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(res.Candidates) != 0 {
		t.Errorf("expected 0 candidates for empty input, got %d", len(res.Candidates))
	}
}

// TestTemporalEndpointCohortV1_AttributionEnriches_v0168 verifies
// the §0168 AttributionLookup consumption pattern: when actor_ref is
// empty AND attribution returns ok, the derived actor_ref is
// collected as enrichment metadata (NOT a skip avoidance — event
// would have counted regardless per the event-centric ontology).
// Cat II derivation hash threaded into SourceHashes per §2.3.
func TestTemporalEndpointCohortV1_AttributionEnriches_v0168(t *testing.T) {
	sig := &TemporalEndpointCohortV1{}
	const bucketStart = int64(1716120000_000_000_000)
	observations := []*eventsv1.NetworkObservation{
		newNetworkObsForCohort("", "10.0.0.1:443", bucketStart),
		newNetworkObsForCohort("", "10.0.0.1:443", bucketStart+10_000_000_000),
		newNetworkObsForCohort("", "10.0.0.1:443", bucketStart+20_000_000_000),
	}
	// Compute observation hashes for stub lookup keys.
	hashes := make([][32]byte, len(observations))
	for i, o := range observations {
		_, h, err := canonicalHashObsAlt(o)
		if err != nil {
			t.Fatalf("hash obs %d: %v", i, err)
		}
		hashes[i] = h
	}
	lookup := &stubAttributionLookup{entries: map[[32]byte]stubAttributionEntry{
		hashes[0]: {derivedActorRef: "actor-derived-1", attributionHash: [32]byte{0xa1}},
		hashes[1]: {derivedActorRef: "actor-derived-2", attributionHash: [32]byte{0xa2}},
		hashes[2]: {derivedActorRef: "actor-derived-3", attributionHash: [32]byte{0xa3}},
	}}
	res, err := sig.EvaluateNetwork(context.Background(), observations, lookup)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(res.Candidates) != 1 {
		t.Fatalf("Candidates: got %d want 1", len(res.Candidates))
	}
	c := res.Candidates[0]
	// Enrichment populated.
	expectedActors := []string{"actor-derived-1", "actor-derived-2", "actor-derived-3"}
	if !equalStringSlices(c.ActorRefs, expectedActors) {
		t.Errorf("ActorRefs: got %v want %v", c.ActorRefs, expectedActors)
	}
	// SourceHashes: 3 Cat I + 3 Cat II per §2.3 chain.
	if len(c.SourceHashes) != 6 {
		t.Errorf("SourceHashes count: got %d want 6 (3 Cat I + 3 Cat II per §2.3)", len(c.SourceHashes))
	}
	// EvidenceCount = sourceHashes count (Cat I + Cat II).
	if c.EvidenceCount != 6 {
		t.Errorf("EvidenceCount: got %d want 6", c.EvidenceCount)
	}
}

func TestTemporalEndpointCohortV1_DeterministicOrder(t *testing.T) {
	sig := &TemporalEndpointCohortV1{}
	const bucketStart = int64(1716120000_000_000_000)
	// Two clusters both meeting threshold; verify candidates emitted
	// in canonical bucket-key alphabetical order.
	observations := []*eventsv1.NetworkObservation{
		newNetworkObsForCohort("actor-1", "10.0.0.2:443", bucketStart),
		newNetworkObsForCohort("actor-2", "10.0.0.2:443", bucketStart+10_000_000_000),
		newNetworkObsForCohort("actor-3", "10.0.0.2:443", bucketStart+20_000_000_000),
		newNetworkObsForCohort("actor-4", "10.0.0.1:443", bucketStart+30_000_000_000),
		newNetworkObsForCohort("actor-5", "10.0.0.1:443", bucketStart+40_000_000_000),
		newNetworkObsForCohort("actor-6", "10.0.0.1:443", bucketStart+50_000_000_000),
	}
	res, err := sig.EvaluateNetwork(context.Background(), observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(res.Candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(res.Candidates))
	}
	// "endpoint=10.0.0.1:443;..." sorts BEFORE "endpoint=10.0.0.2:443;..."
	// because "1" < "2" at position 14.
	first := res.Candidates[0].ActorRefs[0]
	if first != "actor-4" {
		t.Errorf("first candidate first actor: got %q want actor-4 (10.0.0.1 endpoint sorts first lex)", first)
	}
}

func TestTemporalEndpointCohortV1_CustomBucketSeconds(t *testing.T) {
	// 5-minute (300s) bucket override; verify it widens the cluster
	// window. Three events spread across 4 minutes — single bucket
	// under 300s override; would be three buckets under default 60s.
	sig := &TemporalEndpointCohortV1{BucketSeconds: 300}
	const bucketStart = int64(1716120000_000_000_000)
	observations := []*eventsv1.NetworkObservation{
		newNetworkObsForCohort("actor-a", "10.0.0.1:443", bucketStart),
		newNetworkObsForCohort("actor-b", "10.0.0.1:443", bucketStart+120_000_000_000), // +2min
		newNetworkObsForCohort("actor-c", "10.0.0.1:443", bucketStart+240_000_000_000), // +4min
	}
	res, err := sig.EvaluateNetwork(context.Background(), observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(res.Candidates) != 1 {
		t.Fatalf("expected 1 candidate (300s bucket widens to single cohort), got %d", len(res.Candidates))
	}
}

// canonicalHashObsAlt is a local helper that hashes a NetworkObservation
// via the canonical pipeline — used for stub AttributionLookup keys.
// Distinct name from §0161's canonicalHashObs (same package; would
// collide if both files exist with same function name).
func canonicalHashObsAlt(obs *eventsv1.NetworkObservation) ([]byte, [32]byte, error) {
	return canonical.MarshalAndHash(obs)
}
