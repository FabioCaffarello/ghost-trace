package signatures

import (
	"bytes"
	"context"
	"sort"
	"testing"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/canonical"
	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
)

func newNetworkObservationWithTCPFingerprint(actorRef string, p0fSignature string) *eventsv1.NetworkObservation {
	return &eventsv1.NetworkObservation{
		ObservedAt:          1716120000000000000,
		ActorRef:            actorRef,
		EndpointRef:         "198.51.100.5:443",
		CollectorRef:        "cic-ids-2017-adapter:v1",
		AuthenticationClass: commonv1.AuthenticationClass_AUTHENTICATION_CLASS_SERVER_AUTHENTICATED,
		Modality: &eventsv1.NetworkObservation_TcpFingerprint{
			TcpFingerprint: &eventsv1.NetworkTcpFingerprint{
				P0FSignature: p0fSignature,
				WindowSize:   65535,
				Mss:          1460,
				Ttl:          64,
			},
		},
	}
}

func TestTCPFingerprintClusteringV1_BelowThreshold_NoCandidate(t *testing.T) {
	sig := &TCPFingerprintClusteringV1{}
	observations := []*eventsv1.NetworkObservation{
		newNetworkObservationWithTCPFingerprint("actor-a", "4:64:0:1460:mss*44,7:mss,sok,ts,nop,ws:df:0"),
		newNetworkObservationWithTCPFingerprint("actor-b", "4:64:0:1460:mss*44,7:mss,sok,ts,nop,ws:df:0"),
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
	if res.Stats.ActorsAboveThreshold != 0 {
		t.Errorf("ActorsAboveThreshold: got %d want 0", res.Stats.ActorsAboveThreshold)
	}
}

func TestTCPFingerprintClusteringV1_AtThreshold_OneCandidate(t *testing.T) {
	sig := &TCPFingerprintClusteringV1{}
	p0f := "4:64:0:1460:mss*44,7:mss,sok,ts,nop,ws:df:0"
	observations := []*eventsv1.NetworkObservation{
		newNetworkObservationWithTCPFingerprint("actor-a", p0f),
		newNetworkObservationWithTCPFingerprint("actor-b", p0f),
		newNetworkObservationWithTCPFingerprint("actor-c", p0f),
	}
	res, err := sig.EvaluateNetwork(context.Background(), observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(res.Candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(res.Candidates))
	}
	c := res.Candidates[0]
	if c.SignatureName != "tcp_fingerprint_clustering_v1" {
		t.Errorf("SignatureName: got %q", c.SignatureName)
	}
	if c.HypothesisSubtype != HypothesisSubtypeAutomationGroup {
		t.Errorf("HypothesisSubtype: got %v want AutomationGroup", c.HypothesisSubtype)
	}
	expectedActors := []string{"actor-a", "actor-b", "actor-c"}
	if !equalStringSlices(c.ActorRefs, expectedActors) {
		t.Errorf("ActorRefs: got %v want %v", c.ActorRefs, expectedActors)
	}
	if c.EvidenceCount != 3 {
		t.Errorf("EvidenceCount: got %d want 3", c.EvidenceCount)
	}
	if len(c.SourceHashes) != 3 {
		t.Errorf("SourceHashes count: got %d want 3", len(c.SourceHashes))
	}
	for i, h := range c.SourceHashes {
		if len(h) != 32 {
			t.Errorf("SourceHashes[%d]: got len %d want 32", i, len(h))
		}
	}
	// Verify ascending order per §0139.
	for i := 1; i < len(c.SourceHashes); i++ {
		if bytes.Compare(c.SourceHashes[i-1], c.SourceHashes[i]) >= 0 {
			t.Errorf("SourceHashes not strictly ascending at index %d", i)
		}
	}
	if c.ConfidenceHint < 0.5 || c.ConfidenceHint > 0.9 {
		t.Errorf("ConfidenceHint: got %v want in [0.5, 0.9]", c.ConfidenceHint)
	}
	if res.Stats.ActorsAggregated != 3 {
		t.Errorf("ActorsAggregated: got %d want 3", res.Stats.ActorsAggregated)
	}
	if res.Stats.ActorsAboveThreshold != 3 {
		t.Errorf("ActorsAboveThreshold: got %d want 3", res.Stats.ActorsAboveThreshold)
	}
	if res.Stats.CandidatesEmitted != 1 {
		t.Errorf("CandidatesEmitted: got %d want 1", res.Stats.CandidatesEmitted)
	}
	if res.Stats.PerCollector["cic-ids-2017-adapter:v1"] != 3 {
		t.Errorf("PerCollector[cic-ids]: got %d want 3", res.Stats.PerCollector["cic-ids-2017-adapter:v1"])
	}
}

func TestTCPFingerprintClusteringV1_MultipleClusters_OneAboveThreshold(t *testing.T) {
	sig := &TCPFingerprintClusteringV1{}
	p0fA := "4:64:0:1460:mss*44,7:mss,sok,ts,nop,ws:df:0"
	p0fB := "4:128:0:1460:8:mss,nop,ws,nop,nop,sok:df:0"
	observations := []*eventsv1.NetworkObservation{
		newNetworkObservationWithTCPFingerprint("actor-a", p0fA),
		newNetworkObservationWithTCPFingerprint("actor-b", p0fA),
		newNetworkObservationWithTCPFingerprint("actor-c", p0fA),
		newNetworkObservationWithTCPFingerprint("actor-d", p0fB),
		newNetworkObservationWithTCPFingerprint("actor-e", p0fB),
	}
	res, err := sig.EvaluateNetwork(context.Background(), observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(res.Candidates) != 1 {
		t.Fatalf("expected 1 candidate (only p0fA cluster meets threshold), got %d", len(res.Candidates))
	}
	c := res.Candidates[0]
	if !equalStringSlices(c.ActorRefs, []string{"actor-a", "actor-b", "actor-c"}) {
		t.Errorf("ActorRefs: got %v want [actor-a actor-b actor-c]", c.ActorRefs)
	}
	if res.Stats.ActorsAggregated != 5 {
		t.Errorf("ActorsAggregated: got %d want 5", res.Stats.ActorsAggregated)
	}
	if res.Stats.ActorsAboveThreshold != 3 {
		t.Errorf("ActorsAboveThreshold: got %d want 3 (cluster size 3)", res.Stats.ActorsAboveThreshold)
	}
}

func TestTCPFingerprintClusteringV1_DuplicateActorInSameCluster_CountedOnce(t *testing.T) {
	sig := &TCPFingerprintClusteringV1{}
	p0f := "4:64:0:1460:mss*44,7:mss,sok,ts,nop,ws:df:0"
	// actor-a contributes 2 observations; cluster should have 3 distinct
	// actors (a, b, c), still meeting threshold 3.
	observations := []*eventsv1.NetworkObservation{
		newNetworkObservationWithTCPFingerprint("actor-a", p0f),
		newNetworkObservationWithTCPFingerprint("actor-a", p0f),
		newNetworkObservationWithTCPFingerprint("actor-b", p0f),
		newNetworkObservationWithTCPFingerprint("actor-c", p0f),
	}
	res, err := sig.EvaluateNetwork(context.Background(), observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(res.Candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(res.Candidates))
	}
	if res.Stats.ActorsAggregated != 3 {
		t.Errorf("ActorsAggregated: got %d want 3 (distinct actors)", res.Stats.ActorsAggregated)
	}
	c := res.Candidates[0]
	if len(c.ActorRefs) != 3 {
		t.Errorf("ActorRefs count: got %d want 3 (distinct)", len(c.ActorRefs))
	}
	if c.EvidenceCount != 4 {
		t.Errorf("EvidenceCount: got %d want 4 (4 observations)", c.EvidenceCount)
	}
}

func TestTCPFingerprintClusteringV1_EmptyActorRef_Skipped(t *testing.T) {
	sig := &TCPFingerprintClusteringV1{}
	p0f := "4:64:0:1460:mss*44,7:mss,sok,ts,nop,ws:df:0"
	observations := []*eventsv1.NetworkObservation{
		newNetworkObservationWithTCPFingerprint("", p0f),
		newNetworkObservationWithTCPFingerprint("actor-a", p0f),
	}
	res, err := sig.EvaluateNetwork(context.Background(), observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if res.Stats.ObservationsSkippedNoActor != 1 {
		t.Errorf("ObservationsSkippedNoActor: got %d want 1", res.Stats.ObservationsSkippedNoActor)
	}
	if res.Stats.ActorsAggregated != 1 {
		t.Errorf("ActorsAggregated: got %d want 1", res.Stats.ActorsAggregated)
	}
}

func TestTCPFingerprintClusteringV1_EmptyP0FSignature_Skipped(t *testing.T) {
	sig := &TCPFingerprintClusteringV1{}
	observations := []*eventsv1.NetworkObservation{
		newNetworkObservationWithTCPFingerprint("actor-a", ""),
		newNetworkObservationWithTCPFingerprint("actor-b", "4:64:0:1460:mss*44,7:mss,sok,ts,nop,ws:df:0"),
	}
	res, err := sig.EvaluateNetwork(context.Background(), observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if res.Stats.ObservationsSkippedWrongModality != 1 {
		t.Errorf("ObservationsSkippedWrongModality: got %d want 1 (empty p0f)", res.Stats.ObservationsSkippedWrongModality)
	}
}

func TestTCPFingerprintClusteringV1_WrongModality_Skipped(t *testing.T) {
	sig := &TCPFingerprintClusteringV1{}
	obs := &eventsv1.NetworkObservation{
		ObservedAt:   1716120000000000000,
		ActorRef:     "actor-a",
		CollectorRef: "test",
		Modality: &eventsv1.NetworkObservation_IpAsn{
			IpAsn: &eventsv1.NetworkIpAsn{
				IpAddress: "192.0.2.10",
				Asn:       64500,
			},
		},
	}
	res, err := sig.EvaluateNetwork(context.Background(), []*eventsv1.NetworkObservation{obs}, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if res.Stats.ObservationsSkippedWrongModality != 1 {
		t.Errorf("ObservationsSkippedWrongModality: got %d want 1 (ip_asn modality)", res.Stats.ObservationsSkippedWrongModality)
	}
	if len(res.Candidates) != 0 {
		t.Errorf("expected 0 candidates, got %d", len(res.Candidates))
	}
}

func TestTCPFingerprintClusteringV1_EmptyInput_NoCandidates(t *testing.T) {
	sig := &TCPFingerprintClusteringV1{}
	res, err := sig.EvaluateNetwork(context.Background(), []*eventsv1.NetworkObservation{}, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(res.Candidates) != 0 {
		t.Errorf("expected 0 candidates for empty input, got %d", len(res.Candidates))
	}
	if res.Stats.ObservationsScanned != 0 {
		t.Errorf("ObservationsScanned: got %d want 0", res.Stats.ObservationsScanned)
	}
}

func TestTCPFingerprintClusteringV1_DeterministicOrder(t *testing.T) {
	sig := &TCPFingerprintClusteringV1{}
	// Two clusters both meeting threshold; verify candidates emitted
	// in p0f_signature alphabetical order.
	p0fA := "4:128:0:1460:8:mss,nop,ws,nop,nop,sok:df:0" // sorts second
	p0fB := "4:64:0:1460:mss*44,7:mss,sok,ts,nop,ws:df:0" // sorts first
	observations := []*eventsv1.NetworkObservation{
		newNetworkObservationWithTCPFingerprint("actor-1", p0fA),
		newNetworkObservationWithTCPFingerprint("actor-2", p0fA),
		newNetworkObservationWithTCPFingerprint("actor-3", p0fA),
		newNetworkObservationWithTCPFingerprint("actor-4", p0fB),
		newNetworkObservationWithTCPFingerprint("actor-5", p0fB),
		newNetworkObservationWithTCPFingerprint("actor-6", p0fB),
	}
	res, err := sig.EvaluateNetwork(context.Background(), observations, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(res.Candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(res.Candidates))
	}
	// "4:128:..." sorts BEFORE "4:64:..." lexicographically ("1" < "6"
	// at position 2). p0fA cluster (actors 1-3) emits first.
	if res.Candidates[0].ActorRefs[0] != "actor-1" {
		t.Errorf("first candidate actor[0]: got %q want actor-1 (p0fA cluster sorts first lex)", res.Candidates[0].ActorRefs[0])
	}
	if res.Candidates[1].ActorRefs[0] != "actor-4" {
		t.Errorf("second candidate actor[0]: got %q want actor-4 (p0fB cluster sorts second lex)", res.Candidates[1].ActorRefs[0])
	}
}

func TestTCPFingerprintClusteringV1_NameAndSubtype(t *testing.T) {
	sig := &TCPFingerprintClusteringV1{}
	if sig.Name() != "tcp_fingerprint_clustering_v1" {
		t.Errorf("Name: got %q want tcp_fingerprint_clustering_v1", sig.Name())
	}
	if sig.Subtype() != HypothesisSubtypeAutomationGroup {
		t.Errorf("Subtype: got %v want AutomationGroup", sig.Subtype())
	}
}

func TestTCPFingerprintClusteringV1_SatisfiesNetworkSignatureInterface(t *testing.T) {
	var _ NetworkSignature = &TCPFingerprintClusteringV1{}
}

// stubAttributionLookup implements AttributionLookup over a fixed
// in-memory map. Used to drive §0168 Decision A.1 tests without
// requiring the full attribution package + substrate machinery.
type stubAttributionLookup struct {
	entries map[[32]byte]stubAttributionEntry
}

type stubAttributionEntry struct {
	derivedActorRef string
	attributionHash [32]byte
}

func (s *stubAttributionLookup) For(sourceHash [32]byte) (string, [32]byte, bool) {
	e, ok := s.entries[sourceHash]
	if !ok {
		return "", [32]byte{}, false
	}
	return e.derivedActorRef, e.attributionHash, true
}

// TestTCPFingerprintClusteringV1_AttributionFillsEmptyActor_v0168 confirms
// the §0168 Decision A.1 signature-aware Cat II consumption: when
// attribution is provided AND obs.ActorRef is empty AND For returns
// ok, the signature treats the derived actor_ref as the effective
// actor + threads BOTH source observation hash AND Cat II derivation
// hash into the candidate's SourceHashes (preserves §2.3 chain).
func TestTCPFingerprintClusteringV1_AttributionFillsEmptyActor_v0168(t *testing.T) {
	sig := &TCPFingerprintClusteringV1{}
	p0f := "4:64:0:1460:mss*44,7:mss,sok,ts,nop,ws:df:0"

	observations := []*eventsv1.NetworkObservation{
		newNetworkObservationWithTCPFingerprint("", p0f),
		newNetworkObservationWithTCPFingerprint("", p0f),
		newNetworkObservationWithTCPFingerprint("", p0f),
	}
	for i, o := range observations {
		o.ObservedAt += int64(i)
	}

	hashes := make([][32]byte, len(observations))
	for i, o := range observations {
		_, h, err := canonical.MarshalAndHash(o)
		if err != nil {
			t.Fatalf("hash obs %d: %v", i, err)
		}
		hashes[i] = h
	}

	entries := map[[32]byte]stubAttributionEntry{
		hashes[0]: {derivedActorRef: "actor-derived-1", attributionHash: [32]byte{0xa1}},
		hashes[1]: {derivedActorRef: "actor-derived-2", attributionHash: [32]byte{0xa2}},
		hashes[2]: {derivedActorRef: "actor-derived-3", attributionHash: [32]byte{0xa3}},
	}
	lookup := &stubAttributionLookup{entries: entries}

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

	if len(c.SourceHashes) != 6 {
		t.Errorf("SourceHashes count: got %d want 6 (3 Cat I + 3 Cat II per §2.3)", len(c.SourceHashes))
	}

	if res.Stats.ObservationsSkippedNoActor != 0 {
		t.Errorf("ObservationsSkippedNoActor: got %d want 0 (attribution filled all)", res.Stats.ObservationsSkippedNoActor)
	}
	if res.Stats.ActorsAggregated != 3 {
		t.Errorf("ActorsAggregated: got %d want 3 (derived attributions)", res.Stats.ActorsAggregated)
	}
}

// TestTCPFingerprintClusteringV1_DeclaredActorPrecedesDerived_v0168
// confirms that when Cat I declared actor_ref is present, Cat II
// attribution is NOT consulted (declared takes precedence; Cat II
// only fills the GAP per §0168, does not OVERRIDE).
func TestTCPFingerprintClusteringV1_DeclaredActorPrecedesDerived_v0168(t *testing.T) {
	sig := &TCPFingerprintClusteringV1{}
	p0f := "4:64:0:1460:mss*44,7:mss,sok,ts,nop,ws:df:0"

	obs := newNetworkObservationWithTCPFingerprint("actor-declared", p0f)
	_, h, err := canonical.MarshalAndHash(obs)
	if err != nil {
		t.Fatalf("hash obs: %v", err)
	}

	// Stub would return a DIFFERENT derived actor — declared takes
	// precedence so this MUST NOT be consulted.
	lookup := &stubAttributionLookup{entries: map[[32]byte]stubAttributionEntry{
		h: {derivedActorRef: "actor-derived-WRONG", attributionHash: [32]byte{0xff}},
	}}

	res, err := sig.EvaluateNetwork(context.Background(), []*eventsv1.NetworkObservation{obs}, lookup)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if res.Stats.ActorsAggregated != 1 {
		t.Errorf("ActorsAggregated: got %d want 1", res.Stats.ActorsAggregated)
	}
	if res.Stats.ObservationsSkippedNoActor != 0 {
		t.Errorf("ObservationsSkippedNoActor: got %d want 0 (declared present)", res.Stats.ObservationsSkippedNoActor)
	}
}

// TestTCPFingerprintClusteringV1_AttributionAbsent_FallbackToSkip_v0168
// confirms that when attribution is nil OR For returns ok=false, the
// signature falls back to the pre-§0168 skip-on-empty-actor behavior.
func TestTCPFingerprintClusteringV1_AttributionAbsent_FallbackToSkip_v0168(t *testing.T) {
	sig := &TCPFingerprintClusteringV1{}
	p0f := "4:64:0:1460:mss*44,7:mss,sok,ts,nop,ws:df:0"

	obs := newNetworkObservationWithTCPFingerprint("", p0f)

	res1, err := sig.EvaluateNetwork(context.Background(), []*eventsv1.NetworkObservation{obs}, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork nil attribution: %v", err)
	}
	if res1.Stats.ObservationsSkippedNoActor != 1 {
		t.Errorf("nil attribution: ObservationsSkippedNoActor got %d want 1", res1.Stats.ObservationsSkippedNoActor)
	}

	emptyLookup := &stubAttributionLookup{entries: map[[32]byte]stubAttributionEntry{}}
	res2, err := sig.EvaluateNetwork(context.Background(), []*eventsv1.NetworkObservation{obs}, emptyLookup)
	if err != nil {
		t.Fatalf("EvaluateNetwork empty lookup: %v", err)
	}
	if res2.Stats.ObservationsSkippedNoActor != 1 {
		t.Errorf("empty lookup: ObservationsSkippedNoActor got %d want 1", res2.Stats.ObservationsSkippedNoActor)
	}
}

// equalStringSlices reports whether two string slices contain the
// same values in the same order.
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	// Sort both copies and compare; tests pass pre-sorted slices but
	// this guards against ordering accidents.
	ac := append([]string(nil), a...)
	bc := append([]string(nil), b...)
	sort.Strings(ac)
	sort.Strings(bc)
	for i := range ac {
		if ac[i] != bc[i] {
			return false
		}
	}
	return true
}
