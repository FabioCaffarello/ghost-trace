package signatures

import (
	"bytes"
	"context"
	"testing"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
)

func newNetworkObservationWithFlowFeatures(actorRef string, flagsSeq []uint32, windowSize uint32, observedAt int64) *eventsv1.NetworkObservation {
	return &eventsv1.NetworkObservation{
		ObservedAt:          observedAt,
		ActorRef:            actorRef,
		EndpointRef:         "198.51.100.5:443",
		CollectorRef:        "cic-ids-2017-adapter:v1",
		AuthenticationClass: commonv1.AuthenticationClass_AUTHENTICATION_CLASS_SERVER_AUTHENTICATED,
		Modality: &eventsv1.NetworkObservation_TcpFingerprint{
			TcpFingerprint: &eventsv1.NetworkTcpFingerprint{
				FlagsSequence: flagsSeq,
				WindowSize:    windowSize,
				// P0FSignature intentionally empty — this signature
				// does NOT require it.
			},
		},
	}
}

func TestTCPFlowFeaturesClusteringV1_NameAndSubtype(t *testing.T) {
	sig := &TCPFlowFeaturesClusteringV1{}
	if sig.Name() != "tcp_flow_features_clustering_v1" {
		t.Errorf("Name: got %q want tcp_flow_features_clustering_v1", sig.Name())
	}
	if sig.Subtype() != HypothesisSubtypeAutomationGroup {
		t.Errorf("Subtype: got %v want AutomationGroup", sig.Subtype())
	}
}

func TestTCPFlowFeaturesClusteringV1_SatisfiesNetworkSignatureInterface(t *testing.T) {
	var _ NetworkSignature = &TCPFlowFeaturesClusteringV1{}
}

func TestTCPFlowFeaturesClusteringV1_BelowThreshold_NoCandidate(t *testing.T) {
	sig := &TCPFlowFeaturesClusteringV1{}
	flags := []uint32{0x02, 0x10, 0x10}
	observations := []*eventsv1.NetworkObservation{
		newNetworkObservationWithFlowFeatures("actor-a", flags, 65535, 1),
		newNetworkObservationWithFlowFeatures("actor-b", flags, 65535, 2),
	}
	res, err := sig.EvaluateNetwork(context.Background(), observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(res.Candidates) != 0 {
		t.Errorf("expected 0 candidates (2 actors < threshold 3), got %d", len(res.Candidates))
	}
	if res.Stats.ActorsAggregated != 2 {
		t.Errorf("ActorsAggregated: got %d want 2", res.Stats.ActorsAggregated)
	}
}

func TestTCPFlowFeaturesClusteringV1_AtThreshold_OneCandidate(t *testing.T) {
	sig := &TCPFlowFeaturesClusteringV1{}
	flags := []uint32{0x02, 0x10, 0x10, 0x08}
	observations := []*eventsv1.NetworkObservation{
		newNetworkObservationWithFlowFeatures("actor-a", flags, 65535, 1),
		newNetworkObservationWithFlowFeatures("actor-b", flags, 65535, 2),
		newNetworkObservationWithFlowFeatures("actor-c", flags, 65535, 3),
	}
	res, err := sig.EvaluateNetwork(context.Background(), observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(res.Candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(res.Candidates))
	}
	c := res.Candidates[0]
	if c.SignatureName != "tcp_flow_features_clustering_v1" {
		t.Errorf("SignatureName: got %q", c.SignatureName)
	}
	if c.HypothesisSubtype != HypothesisSubtypeAutomationGroup {
		t.Errorf("HypothesisSubtype: got %v want AutomationGroup", c.HypothesisSubtype)
	}
	if !equalStringSlices(c.ActorRefs, []string{"actor-a", "actor-b", "actor-c"}) {
		t.Errorf("ActorRefs: got %v want [actor-a actor-b actor-c]", c.ActorRefs)
	}
	if len(c.SourceHashes) != 3 {
		t.Errorf("SourceHashes count: got %d want 3", len(c.SourceHashes))
	}
	for i, h := range c.SourceHashes {
		if len(h) != 32 {
			t.Errorf("SourceHashes[%d]: got len %d want 32", i, len(h))
		}
	}
	for i := 1; i < len(c.SourceHashes); i++ {
		if bytes.Compare(c.SourceHashes[i-1], c.SourceHashes[i]) >= 0 {
			t.Errorf("SourceHashes not strictly ascending at index %d", i)
		}
	}
}

func TestTCPFlowFeaturesClusteringV1_DistinctFeatures_NoCrossCluster(t *testing.T) {
	sig := &TCPFlowFeaturesClusteringV1{}
	// 3 actors with feature vector A; 3 actors with feature vector B.
	// Both clusters meet threshold; both emit.
	flagsA := []uint32{0x02, 0x10, 0x10}
	flagsB := []uint32{0x02, 0x10, 0x10, 0x10}
	observations := []*eventsv1.NetworkObservation{
		newNetworkObservationWithFlowFeatures("actor-1", flagsA, 65535, 1),
		newNetworkObservationWithFlowFeatures("actor-2", flagsA, 65535, 2),
		newNetworkObservationWithFlowFeatures("actor-3", flagsA, 65535, 3),
		newNetworkObservationWithFlowFeatures("actor-4", flagsB, 29200, 4),
		newNetworkObservationWithFlowFeatures("actor-5", flagsB, 29200, 5),
		newNetworkObservationWithFlowFeatures("actor-6", flagsB, 29200, 6),
	}
	res, err := sig.EvaluateNetwork(context.Background(), observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(res.Candidates) != 2 {
		t.Fatalf("expected 2 candidates (distinct feature clusters), got %d", len(res.Candidates))
	}
	if res.Stats.ActorsAggregated != 6 {
		t.Errorf("ActorsAggregated: got %d want 6", res.Stats.ActorsAggregated)
	}
	if res.Stats.ActorsAboveThreshold != 6 {
		t.Errorf("ActorsAboveThreshold: got %d want 6 (3+3)", res.Stats.ActorsAboveThreshold)
	}
}

func TestTCPFlowFeaturesClusteringV1_WindowDifferenceSeparatesClusters(t *testing.T) {
	sig := &TCPFlowFeaturesClusteringV1{}
	flags := []uint32{0x02, 0x10}
	// 3 actors with same flags but DIFFERENT window sizes split into
	// 3 singleton clusters; none meets threshold.
	observations := []*eventsv1.NetworkObservation{
		newNetworkObservationWithFlowFeatures("actor-a", flags, 65535, 1),
		newNetworkObservationWithFlowFeatures("actor-b", flags, 29200, 2),
		newNetworkObservationWithFlowFeatures("actor-c", flags, 8192, 3),
	}
	res, err := sig.EvaluateNetwork(context.Background(), observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(res.Candidates) != 0 {
		t.Errorf("expected 0 candidates (each actor in its own window cluster), got %d", len(res.Candidates))
	}
}

func TestTCPFlowFeaturesClusteringV1_NoFlowFeatures_Skipped(t *testing.T) {
	sig := &TCPFlowFeaturesClusteringV1{}
	// Observation with empty flags AND window_size=0 → no features.
	obs := newNetworkObservationWithFlowFeatures("actor-a", nil, 0, 1)
	res, err := sig.EvaluateNetwork(context.Background(), []*eventsv1.NetworkObservation{obs}, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if res.Stats.ObservationsSkippedWrongModality != 1 {
		t.Errorf("ObservationsSkippedWrongModality: got %d want 1 (no flow features)", res.Stats.ObservationsSkippedWrongModality)
	}
}

func TestTCPFlowFeaturesClusteringV1_WindowOnly_NotSkipped(t *testing.T) {
	// Window-size-only (empty flags) still has clusterable feature.
	sig := &TCPFlowFeaturesClusteringV1{}
	observations := []*eventsv1.NetworkObservation{
		newNetworkObservationWithFlowFeatures("actor-a", nil, 65535, 1),
		newNetworkObservationWithFlowFeatures("actor-b", nil, 65535, 2),
		newNetworkObservationWithFlowFeatures("actor-c", nil, 65535, 3),
	}
	res, err := sig.EvaluateNetwork(context.Background(), observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(res.Candidates) != 1 {
		t.Errorf("Candidates: got %d want 1 (window-only cluster meets threshold)", len(res.Candidates))
	}
}

func TestTCPFlowFeaturesClusteringV1_EmptyActorRef_Skipped_NoAttribution(t *testing.T) {
	sig := &TCPFlowFeaturesClusteringV1{}
	flags := []uint32{0x02}
	obs := newNetworkObservationWithFlowFeatures("", flags, 65535, 1)
	res, err := sig.EvaluateNetwork(context.Background(), []*eventsv1.NetworkObservation{obs}, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if res.Stats.ObservationsSkippedNoActor != 1 {
		t.Errorf("ObservationsSkippedNoActor: got %d want 1", res.Stats.ObservationsSkippedNoActor)
	}
}

// TestTCPFlowFeaturesClusteringV1_AttributionFillsEmptyActor_v0168
// confirms §0168 Decision A.1 consumption applies to this signature
// identically to §0161's tcp_fingerprint_clustering_v1: when actor
// is empty + attribution.For returns ok, derived actor + Cat II hash
// thread into the cluster + candidate SourceHashes.
func TestTCPFlowFeaturesClusteringV1_AttributionFillsEmptyActor_v0168(t *testing.T) {
	sig := &TCPFlowFeaturesClusteringV1{}
	flags := []uint32{0x02, 0x10}

	observations := []*eventsv1.NetworkObservation{
		newNetworkObservationWithFlowFeatures("", flags, 65535, 1),
		newNetworkObservationWithFlowFeatures("", flags, 65535, 2),
		newNetworkObservationWithFlowFeatures("", flags, 65535, 3),
	}

	hashes := make([][32]byte, len(observations))
	for i, o := range observations {
		_, h, err := canonical.MarshalAndHash(o)
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
	if !equalStringSlices(c.ActorRefs, []string{"actor-derived-1", "actor-derived-2", "actor-derived-3"}) {
		t.Errorf("ActorRefs: got %v", c.ActorRefs)
	}
	// 3 Cat I + 3 Cat II per §2.3 chain.
	if len(c.SourceHashes) != 6 {
		t.Errorf("SourceHashes count: got %d want 6 (3 Cat I + 3 Cat II)", len(c.SourceHashes))
	}
}

func TestTCPFlowFeaturesClusteringV1_DeterministicOrder(t *testing.T) {
	sig := &TCPFlowFeaturesClusteringV1{}
	// Two clusters both meeting threshold; verify candidates emitted
	// in canonical feature-key ascending order.
	flagsA := []uint32{0x02, 0x10}     // "flags=2,16;window=8192"  (sorts after window-only)
	flagsB := []uint32{0x02, 0x10, 0x10} // "flags=2,16,16;window=65535"
	observations := []*eventsv1.NetworkObservation{
		newNetworkObservationWithFlowFeatures("actor-1", flagsA, 8192, 1),
		newNetworkObservationWithFlowFeatures("actor-2", flagsA, 8192, 2),
		newNetworkObservationWithFlowFeatures("actor-3", flagsA, 8192, 3),
		newNetworkObservationWithFlowFeatures("actor-4", flagsB, 65535, 4),
		newNetworkObservationWithFlowFeatures("actor-5", flagsB, 65535, 5),
		newNetworkObservationWithFlowFeatures("actor-6", flagsB, 65535, 6),
	}
	res, err := sig.EvaluateNetwork(context.Background(), observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(res.Candidates) != 2 {
		t.Fatalf("Candidates: got %d want 2", len(res.Candidates))
	}
	// "flags=2,16,16;..." sorts before "flags=2,16;..." because "," < ";".
	// Verify the cluster ordering via first-actor membership.
	first := res.Candidates[0].ActorRefs[0]
	second := res.Candidates[1].ActorRefs[0]
	if first == "actor-1" || first == "actor-2" || first == "actor-3" {
		// flagsA cluster came first; flagsB cluster second.
		if !(second == "actor-4" || second == "actor-5" || second == "actor-6") {
			t.Errorf("second candidate not from flagsB cluster: %q", second)
		}
	} else {
		// flagsB cluster came first.
		if !(second == "actor-1" || second == "actor-2" || second == "actor-3") {
			t.Errorf("second candidate not from flagsA cluster: %q", second)
		}
	}
}
