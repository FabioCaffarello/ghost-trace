package signatures

import (
	"context"
	"testing"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
)

func tlsObs(actor, ja4, ja3 string) *eventsv1.NetworkObservation {
	return &eventsv1.NetworkObservation{
		ActorRef: actor,
		Modality: &eventsv1.NetworkObservation_TlsJa4{
			TlsJa4: &eventsv1.NetworkTlsJa4{Ja4: ja4, Ja3: ja3},
		},
	}
}

func TestTLSJa4Automation_MatchesKnownJA4(t *testing.T) {
	sig := &TLSJa4AutomationV1{KnownJA4: map[string]struct{}{"bot-ja4": {}}}
	obs := []*eventsv1.NetworkObservation{tlsObs("actor-1", "bot-ja4", "")}
	res, err := sig.EvaluateNetwork(context.Background(), obs, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(res.Candidates) != 1 {
		t.Fatalf("candidates: got %d want 1", len(res.Candidates))
	}
	if res.Candidates[0].ActorRefs[0] != "actor-1" || res.Candidates[0].HypothesisSubtype != HypothesisSubtypeAutomationGroup {
		t.Errorf("unexpected candidate: %+v", res.Candidates[0])
	}
	if len(res.Candidates[0].SourceHashes) != 1 {
		t.Errorf("source hashes: got %d want 1", len(res.Candidates[0].SourceHashes))
	}
}

func TestTLSJa4Automation_MatchesKnownJA3(t *testing.T) {
	sig := &TLSJa4AutomationV1{KnownJA3: map[string]struct{}{"bot-ja3": {}}}
	obs := []*eventsv1.NetworkObservation{tlsObs("actor-1", "some-ja4", "bot-ja3")}
	res, err := sig.EvaluateNetwork(context.Background(), obs, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(res.Candidates) != 1 {
		t.Fatalf("candidates: got %d want 1 (JA3 match)", len(res.Candidates))
	}
}

func TestTLSJa4Automation_NoMatchEmitsNothing(t *testing.T) {
	sig := &TLSJa4AutomationV1{KnownJA4: map[string]struct{}{"bot-ja4": {}}}
	obs := []*eventsv1.NetworkObservation{tlsObs("actor-1", "human-ja4", "human-ja3")}
	res, err := sig.EvaluateNetwork(context.Background(), obs, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(res.Candidates) != 0 {
		t.Fatalf("candidates: got %d want 0", len(res.Candidates))
	}
	// In-modality but unmatched: aggregated, not skipped-wrong-modality.
	if res.Stats.ActorsAggregated != 1 {
		t.Errorf("ActorsAggregated: got %d want 1", res.Stats.ActorsAggregated)
	}
	if res.Stats.ObservationsSkippedWrongModality != 0 {
		t.Errorf("SkippedWrongModality: got %d want 0", res.Stats.ObservationsSkippedWrongModality)
	}
}

func TestTLSJa4Automation_EmptyKnownSetEmitsNothing(t *testing.T) {
	// No reference set supplied: the signature asserts nothing.
	sig := &TLSJa4AutomationV1{}
	obs := []*eventsv1.NetworkObservation{tlsObs("actor-1", "any-ja4", "any-ja3")}
	res, err := sig.EvaluateNetwork(context.Background(), obs, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(res.Candidates) != 0 {
		t.Fatalf("candidates: got %d want 0 (empty known-set asserts nothing)", len(res.Candidates))
	}
}

func TestTLSJa4Automation_SkipNoActor(t *testing.T) {
	sig := &TLSJa4AutomationV1{KnownJA4: map[string]struct{}{"bot-ja4": {}}}
	obs := []*eventsv1.NetworkObservation{tlsObs("", "bot-ja4", "")}
	res, err := sig.EvaluateNetwork(context.Background(), obs, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if res.Stats.ObservationsSkippedNoActor != 1 {
		t.Errorf("SkippedNoActor: got %d want 1", res.Stats.ObservationsSkippedNoActor)
	}
	if len(res.Candidates) != 0 {
		t.Errorf("candidates: got %d want 0 (unattributed)", len(res.Candidates))
	}
}

func TestTLSJa4Automation_SkipWrongModality(t *testing.T) {
	sig := &TLSJa4AutomationV1{KnownJA4: map[string]struct{}{"bot-ja4": {}}}
	// tcp_fingerprint modality, not tls_ja4.
	obs := []*eventsv1.NetworkObservation{{
		ActorRef: "actor-1",
		Modality: &eventsv1.NetworkObservation_TcpFingerprint{
			TcpFingerprint: &eventsv1.NetworkTcpFingerprint{P0FSignature: "x"},
		},
	}}
	res, err := sig.EvaluateNetwork(context.Background(), obs, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if res.Stats.ObservationsSkippedWrongModality != 1 {
		t.Errorf("SkippedWrongModality: got %d want 1", res.Stats.ObservationsSkippedWrongModality)
	}
}

func TestTLSJa4Automation_ThresholdGatesSingleMatch(t *testing.T) {
	sig := &TLSJa4AutomationV1{KnownJA4: map[string]struct{}{"bot-ja4": {}}, Threshold: 2}
	// One match for actor-1 (< threshold 2) → no candidate.
	obs := []*eventsv1.NetworkObservation{tlsObs("actor-1", "bot-ja4", "")}
	res, err := sig.EvaluateNetwork(context.Background(), obs, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(res.Candidates) != 0 {
		t.Fatalf("candidates: got %d want 0 (1 match < threshold 2)", len(res.Candidates))
	}
	// Two matches for actor-1 → candidate.
	obs = append(obs, tlsObs("actor-1", "bot-ja4", ""))
	// Distinct ja4_raw to avoid identical content-hash collapsing the
	// two observations into one substrate-equal record is unnecessary
	// here (signature hashes whole obs; endpoint differs below).
	obs[1].EndpointRef = "10.0.0.9:443"
	res, err = sig.EvaluateNetwork(context.Background(), obs, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(res.Candidates) != 1 {
		t.Fatalf("candidates: got %d want 1 (2 matches >= threshold 2)", len(res.Candidates))
	}
}

func TestTLSJa4Automation_DeterministicCandidateOrder(t *testing.T) {
	sig := &TLSJa4AutomationV1{KnownJA4: map[string]struct{}{"bot-ja4": {}}}
	obs := []*eventsv1.NetworkObservation{
		tlsObs("actor-z", "bot-ja4", ""),
		tlsObs("actor-a", "bot-ja4", ""),
		tlsObs("actor-m", "bot-ja4", ""),
	}
	res, err := sig.EvaluateNetwork(context.Background(), obs, nil)
	if err != nil {
		t.Fatalf("EvaluateNetwork: %v", err)
	}
	if len(res.Candidates) != 3 {
		t.Fatalf("candidates: got %d want 3", len(res.Candidates))
	}
	want := []string{"actor-a", "actor-m", "actor-z"}
	for i, c := range res.Candidates {
		if c.ActorRefs[0] != want[i] {
			t.Errorf("candidate[%d] actor: got %q want %q", i, c.ActorRefs[0], want[i])
		}
	}
}
