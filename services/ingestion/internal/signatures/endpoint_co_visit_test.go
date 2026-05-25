package signatures

import (
	"bytes"
	"context"
	"testing"

	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
)

// newNetworkObsForCoVisit builds a NetworkObservation with envelope
// fields populated. Helper distinct from §0182's newNetworkObsForCohort
// to avoid in-package collision; structurally identical apart from
// telemetry labelling.
func newNetworkObsForCoVisit(actorRef, endpointRef string, observedAt int64) *eventsv1.NetworkObservation {
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

func TestEndpointCoVisitV1_NameAndSubtype(t *testing.T) {
	sig := &EndpointCoVisitV1{}
	if sig.Name() != "endpoint_co_visit_v1" {
		t.Errorf("Name: got %q want endpoint_co_visit_v1", sig.Name())
	}
	if sig.Subtype() != HypothesisSubtypeCoordinationRing {
		t.Errorf("Subtype: got %v want CoordinationRing", sig.Subtype())
	}
}

func TestEndpointCoVisitV1_SatisfiesNetworkSignatureInterface(t *testing.T) {
	var _ NetworkSignature = &EndpointCoVisitV1{}
}

func TestEndpointCoVisitV1_BelowThreshold_NoCandidate(t *testing.T) {
	sig := &EndpointCoVisitV1{}
	const bucketStart = int64(1716120000_000_000_000)
	observations := []*eventsv1.NetworkObservation{
		newNetworkObsForCoVisit("actor-a", "10.0.0.1:443", bucketStart),
		newNetworkObsForCoVisit("actor-b", "10.0.0.1:443", bucketStart+10_000_000_000),
	}
	res, err := sig.EvaluateNetwork(context.Background(), observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(res.Candidates) != 0 {
		t.Errorf("expected 0 candidates (2 actors < threshold 3), got %d", len(res.Candidates))
	}
}

func TestEndpointCoVisitV1_AtThreshold_OneCandidateWithEdges(t *testing.T) {
	sig := &EndpointCoVisitV1{}
	const bucketStart = int64(1716120000_000_000_000)
	observations := []*eventsv1.NetworkObservation{
		newNetworkObsForCoVisit("actor-a", "10.0.0.1:443", bucketStart),
		newNetworkObsForCoVisit("actor-b", "10.0.0.1:443", bucketStart+10_000_000_000),
		newNetworkObsForCoVisit("actor-c", "10.0.0.1:443", bucketStart+20_000_000_000),
	}
	res, err := sig.EvaluateNetwork(context.Background(), observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(res.Candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(res.Candidates))
	}
	c := res.Candidates[0]
	if c.SignatureName != "endpoint_co_visit_v1" {
		t.Errorf("SignatureName: got %q", c.SignatureName)
	}
	if c.HypothesisSubtype != HypothesisSubtypeCoordinationRing {
		t.Errorf("HypothesisSubtype: got %v want CoordinationRing", c.HypothesisSubtype)
	}
	if !equalStringSlices(c.ActorRefs, []string{"actor-a", "actor-b", "actor-c"}) {
		t.Errorf("ActorRefs (vertex set): got %v want [actor-a actor-b actor-c]", c.ActorRefs)
	}

	// 3 actors → 3 edges: (a,b), (a,c), (b,c) — all lex-ordered.
	wantEdges := [][2]string{
		{"actor-a", "actor-b"},
		{"actor-a", "actor-c"},
		{"actor-b", "actor-c"},
	}
	if len(c.Interactions) != len(wantEdges) {
		t.Fatalf("Interactions count: got %d want %d", len(c.Interactions), len(wantEdges))
	}
	for i, edge := range c.Interactions {
		if edge != wantEdges[i] {
			t.Errorf("Interactions[%d]: got %v want %v", i, edge, wantEdges[i])
		}
	}

	if len(c.SourceHashes) != 3 {
		t.Errorf("SourceHashes count: got %d want 3", len(c.SourceHashes))
	}
	if c.EvidenceCount != 3 {
		t.Errorf("EvidenceCount: got %d want 3", c.EvidenceCount)
	}
	for i := 1; i < len(c.SourceHashes); i++ {
		if bytes.Compare(c.SourceHashes[i-1], c.SourceHashes[i]) >= 0 {
			t.Errorf("SourceHashes not strictly ascending at index %d", i)
		}
	}
	if c.ConfidenceHint < 0.5 || c.ConfidenceHint > 0.9 {
		t.Errorf("ConfidenceHint: got %v want in [0.5, 0.9]", c.ConfidenceHint)
	}
}

// TestEndpointCoVisitV1_EdgeCanonicalization_LexOrderWithinPair
// verifies per-§0070 within-edge canonical ordering: actors arriving
// in reverse-lex order MUST produce edges with edge[0] < edge[1]
// regardless of observation order.
func TestEndpointCoVisitV1_EdgeCanonicalization_LexOrderWithinPair(t *testing.T) {
	sig := &EndpointCoVisitV1{}
	const bucketStart = int64(1716120000_000_000_000)
	// Reverse lex order at ingest: zebra, yak, alpha.
	observations := []*eventsv1.NetworkObservation{
		newNetworkObsForCoVisit("zebra", "10.0.0.1:443", bucketStart),
		newNetworkObsForCoVisit("yak", "10.0.0.1:443", bucketStart+10_000_000_000),
		newNetworkObsForCoVisit("alpha", "10.0.0.1:443", bucketStart+20_000_000_000),
	}
	res, err := sig.EvaluateNetwork(context.Background(), observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(res.Candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(res.Candidates))
	}
	c := res.Candidates[0]
	// Expected canonical edges: (alpha,yak), (alpha,zebra), (yak,zebra).
	wantEdges := [][2]string{
		{"alpha", "yak"},
		{"alpha", "zebra"},
		{"yak", "zebra"},
	}
	for i, edge := range c.Interactions {
		if edge != wantEdges[i] {
			t.Errorf("Interactions[%d]: got %v want %v (per §0070 within-edge lex)", i, edge, wantEdges[i])
		}
		if edge[0] >= edge[1] {
			t.Errorf("Interactions[%d]: edge[0]=%q NOT lex-less-than edge[1]=%q (per §0070)", i, edge[0], edge[1])
		}
	}
}

// TestEndpointCoVisitV1_InteractionsAscendingOrder verifies the
// per-§0070 ascending-sort-across-edges discipline. Edges MUST be
// sorted by (edge[0], edge[1]) lex.
func TestEndpointCoVisitV1_InteractionsAscendingOrder(t *testing.T) {
	sig := &EndpointCoVisitV1{}
	const bucketStart = int64(1716120000_000_000_000)
	observations := []*eventsv1.NetworkObservation{
		newNetworkObsForCoVisit("d", "10.0.0.1:443", bucketStart),
		newNetworkObsForCoVisit("a", "10.0.0.1:443", bucketStart+10_000_000_000),
		newNetworkObsForCoVisit("c", "10.0.0.1:443", bucketStart+20_000_000_000),
		newNetworkObsForCoVisit("b", "10.0.0.1:443", bucketStart+30_000_000_000),
	}
	res, err := sig.EvaluateNetwork(context.Background(), observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(res.Candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(res.Candidates))
	}
	c := res.Candidates[0]
	// 4 actors → 6 edges; ascending sorted:
	// (a,b), (a,c), (a,d), (b,c), (b,d), (c,d).
	for i := 1; i < len(c.Interactions); i++ {
		prev := c.Interactions[i-1]
		curr := c.Interactions[i]
		if prev[0] > curr[0] || (prev[0] == curr[0] && prev[1] >= curr[1]) {
			t.Errorf("Interactions[%d] not strictly ascending: prev=%v curr=%v (per §0070)", i, prev, curr)
		}
	}
	if len(c.Interactions) != 6 {
		t.Errorf("Interactions count: got %d want 6 (4 actors → 4*3/2 edges)", len(c.Interactions))
	}
}

// TestEndpointCoVisitV1_InteractionsNoDuplicates verifies that even
// when the same actor pair co-visits MULTIPLE times within a bucket
// (multiple observations per actor), the edge list contains no
// duplicate pairs per §0070.
func TestEndpointCoVisitV1_InteractionsNoDuplicates(t *testing.T) {
	sig := &EndpointCoVisitV1{}
	const bucketStart = int64(1716120000_000_000_000)
	// actor-a + actor-b each observed twice; actor-c observed once.
	// Single bucket; expected edges: (a,b), (a,c), (b,c) — only 3 distinct.
	observations := []*eventsv1.NetworkObservation{
		newNetworkObsForCoVisit("actor-a", "10.0.0.1:443", bucketStart),
		newNetworkObsForCoVisit("actor-a", "10.0.0.1:443", bucketStart+5_000_000_000),
		newNetworkObsForCoVisit("actor-b", "10.0.0.1:443", bucketStart+10_000_000_000),
		newNetworkObsForCoVisit("actor-b", "10.0.0.1:443", bucketStart+15_000_000_000),
		newNetworkObsForCoVisit("actor-c", "10.0.0.1:443", bucketStart+20_000_000_000),
	}
	res, err := sig.EvaluateNetwork(context.Background(), observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(res.Candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(res.Candidates))
	}
	c := res.Candidates[0]
	if len(c.Interactions) != 3 {
		t.Errorf("Interactions count: got %d want 3 (de-duplicated per §0070)", len(c.Interactions))
	}
	if len(c.ActorRefs) != 3 {
		t.Errorf("ActorRefs (vertex set): got %d want 3 (de-duplicated)", len(c.ActorRefs))
	}
}

// TestEndpointCoVisitV1_DifferentEndpoints_NoCrossCluster verifies
// that actors visiting DIFFERENT endpoints in the same time window
// do NOT cluster into a single ring.
func TestEndpointCoVisitV1_DifferentEndpoints_NoCrossCluster(t *testing.T) {
	sig := &EndpointCoVisitV1{}
	const bucketStart = int64(1716120000_000_000_000)
	observations := []*eventsv1.NetworkObservation{
		newNetworkObsForCoVisit("actor-1", "10.0.0.1:443", bucketStart+1_000_000_000),
		newNetworkObsForCoVisit("actor-2", "10.0.0.1:443", bucketStart+2_000_000_000),
		newNetworkObsForCoVisit("actor-3", "10.0.0.1:443", bucketStart+3_000_000_000),
		newNetworkObsForCoVisit("actor-4", "10.0.0.2:443", bucketStart+4_000_000_000),
		newNetworkObsForCoVisit("actor-5", "10.0.0.2:443", bucketStart+5_000_000_000),
		newNetworkObsForCoVisit("actor-6", "10.0.0.2:443", bucketStart+6_000_000_000),
	}
	res, err := sig.EvaluateNetwork(context.Background(), observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(res.Candidates) != 2 {
		t.Fatalf("expected 2 candidates (distinct endpoint rings), got %d", len(res.Candidates))
	}
}

// TestEndpointCoVisitV1_DifferentBuckets_NoCrossCluster verifies that
// actors at the SAME endpoint but DIFFERENT time buckets do NOT
// cluster into a single ring.
func TestEndpointCoVisitV1_DifferentBuckets_NoCrossCluster(t *testing.T) {
	sig := &EndpointCoVisitV1{}
	const bucketStart = int64(1716120000_000_000_000)
	observations := []*eventsv1.NetworkObservation{
		newNetworkObsForCoVisit("actor-1", "10.0.0.1:443", bucketStart+1_000_000_000),
		newNetworkObsForCoVisit("actor-2", "10.0.0.1:443", bucketStart+2_000_000_000),
		newNetworkObsForCoVisit("actor-3", "10.0.0.1:443", bucketStart+3_000_000_000),
		newNetworkObsForCoVisit("actor-4", "10.0.0.1:443", bucketStart+120_000_000_000),
		newNetworkObsForCoVisit("actor-5", "10.0.0.1:443", bucketStart+121_000_000_000),
		newNetworkObsForCoVisit("actor-6", "10.0.0.1:443", bucketStart+122_000_000_000),
	}
	res, err := sig.EvaluateNetwork(context.Background(), observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(res.Candidates) != 2 {
		t.Fatalf("expected 2 candidates (distinct time-bucket rings), got %d", len(res.Candidates))
	}
}

// TestEndpointCoVisitV1_EmptyActorRef_Skipped is the structurally
// distinctive test for the interaction-centric ontology: unlike
// §0182's CampaignHypothesis (event-centric, actor-optional), an
// observation without actor_ref CANNOT participate in a coordination
// edge — both actors of an edge MUST be named. Empty actor_ref IS a
// skip reason; ObservationsSkippedNoActor++ records the skip.
func TestEndpointCoVisitV1_EmptyActorRef_Skipped(t *testing.T) {
	sig := &EndpointCoVisitV1{}
	const bucketStart = int64(1716120000_000_000_000)
	observations := []*eventsv1.NetworkObservation{
		newNetworkObsForCoVisit("", "10.0.0.1:443", bucketStart),
		newNetworkObsForCoVisit("", "10.0.0.1:443", bucketStart+10_000_000_000),
		newNetworkObsForCoVisit("", "10.0.0.1:443", bucketStart+20_000_000_000),
	}
	res, err := sig.EvaluateNetwork(context.Background(), observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(res.Candidates) != 0 {
		t.Errorf("expected 0 candidates (all actor_refs empty), got %d", len(res.Candidates))
	}
	if res.Stats.ObservationsSkippedNoActor != 3 {
		t.Errorf("ObservationsSkippedNoActor: got %d want 3 (interaction-centric ontology — named actors required)", res.Stats.ObservationsSkippedNoActor)
	}
}

func TestEndpointCoVisitV1_EmptyEndpoint_Skipped(t *testing.T) {
	sig := &EndpointCoVisitV1{}
	const bucketStart = int64(1716120000_000_000_000)
	obs := newNetworkObsForCoVisit("actor-a", "", bucketStart)
	res, err := sig.EvaluateNetwork(context.Background(), []*eventsv1.NetworkObservation{obs}, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if res.Stats.ObservationsSkippedWrongModality != 1 {
		t.Errorf("ObservationsSkippedWrongModality: got %d want 1 (empty endpoint_ref)", res.Stats.ObservationsSkippedWrongModality)
	}
}

func TestEndpointCoVisitV1_ZeroObservedAt_Skipped(t *testing.T) {
	sig := &EndpointCoVisitV1{}
	obs := newNetworkObsForCoVisit("actor-a", "10.0.0.1:443", 0)
	res, err := sig.EvaluateNetwork(context.Background(), []*eventsv1.NetworkObservation{obs}, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if res.Stats.ObservationsSkippedWrongModality != 1 {
		t.Errorf("ObservationsSkippedWrongModality: got %d want 1 (observed_at == 0)", res.Stats.ObservationsSkippedWrongModality)
	}
}

func TestEndpointCoVisitV1_EmptyInput_NoCandidates(t *testing.T) {
	sig := &EndpointCoVisitV1{}
	res, err := sig.EvaluateNetwork(context.Background(), []*eventsv1.NetworkObservation{}, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(res.Candidates) != 0 {
		t.Errorf("expected 0 candidates for empty input, got %d", len(res.Candidates))
	}
}

// TestEndpointCoVisitV1_AttributionFillsActor_v0168 verifies the
// §0168 AttributionLookup consumption pattern: when actor_ref is
// empty AND attribution returns ok, the derived actor_ref is used
// as the effective actor (preventing what would otherwise be a
// NoActor skip). Cat II derivation hash threaded into SourceHashes
// per §2.3.
func TestEndpointCoVisitV1_AttributionFillsActor_v0168(t *testing.T) {
	sig := &EndpointCoVisitV1{}
	const bucketStart = int64(1716120000_000_000_000)
	observations := []*eventsv1.NetworkObservation{
		newNetworkObsForCoVisit("", "10.0.0.1:443", bucketStart),
		newNetworkObsForCoVisit("", "10.0.0.1:443", bucketStart+10_000_000_000),
		newNetworkObsForCoVisit("", "10.0.0.1:443", bucketStart+20_000_000_000),
	}
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
	expectedActors := []string{"actor-derived-1", "actor-derived-2", "actor-derived-3"}
	if !equalStringSlices(c.ActorRefs, expectedActors) {
		t.Errorf("ActorRefs: got %v want %v", c.ActorRefs, expectedActors)
	}
	// 3 derived actors → 3 edges.
	if len(c.Interactions) != 3 {
		t.Errorf("Interactions count: got %d want 3", len(c.Interactions))
	}
	// SourceHashes: 3 Cat I + 3 Cat II per §2.3 chain.
	if len(c.SourceHashes) != 6 {
		t.Errorf("SourceHashes count: got %d want 6 (3 Cat I + 3 Cat II per §2.3)", len(c.SourceHashes))
	}
	if c.EvidenceCount != 6 {
		t.Errorf("EvidenceCount: got %d want 6", c.EvidenceCount)
	}
	// Empty-actor skip MUST NOT have fired (attribution filled the gap).
	if res.Stats.ObservationsSkippedNoActor != 0 {
		t.Errorf("ObservationsSkippedNoActor: got %d want 0 (attribution filled actor)", res.Stats.ObservationsSkippedNoActor)
	}
}

func TestEndpointCoVisitV1_DeterministicOrder(t *testing.T) {
	sig := &EndpointCoVisitV1{}
	const bucketStart = int64(1716120000_000_000_000)
	observations := []*eventsv1.NetworkObservation{
		newNetworkObsForCoVisit("actor-1", "10.0.0.2:443", bucketStart),
		newNetworkObsForCoVisit("actor-2", "10.0.0.2:443", bucketStart+10_000_000_000),
		newNetworkObsForCoVisit("actor-3", "10.0.0.2:443", bucketStart+20_000_000_000),
		newNetworkObsForCoVisit("actor-4", "10.0.0.1:443", bucketStart+30_000_000_000),
		newNetworkObsForCoVisit("actor-5", "10.0.0.1:443", bucketStart+40_000_000_000),
		newNetworkObsForCoVisit("actor-6", "10.0.0.1:443", bucketStart+50_000_000_000),
	}
	res, err := sig.EvaluateNetwork(context.Background(), observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(res.Candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(res.Candidates))
	}
	// "endpoint=10.0.0.1:..." sorts BEFORE "endpoint=10.0.0.2:..."
	first := res.Candidates[0].ActorRefs[0]
	if first != "actor-4" {
		t.Errorf("first candidate first actor: got %q want actor-4 (10.0.0.1 endpoint sorts first lex)", first)
	}
}

func TestEndpointCoVisitV1_CustomBucketSeconds(t *testing.T) {
	sig := &EndpointCoVisitV1{BucketSeconds: 300}
	const bucketStart = int64(1716120000_000_000_000)
	observations := []*eventsv1.NetworkObservation{
		newNetworkObsForCoVisit("actor-a", "10.0.0.1:443", bucketStart),
		newNetworkObsForCoVisit("actor-b", "10.0.0.1:443", bucketStart+120_000_000_000),
		newNetworkObsForCoVisit("actor-c", "10.0.0.1:443", bucketStart+240_000_000_000),
	}
	res, err := sig.EvaluateNetwork(context.Background(), observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(res.Candidates) != 1 {
		t.Fatalf("expected 1 candidate (300s bucket widens to single ring), got %d", len(res.Candidates))
	}
}
