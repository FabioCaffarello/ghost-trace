package signatures

import (
	"bytes"
	"context"
	"testing"

	commonv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/common/v1"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
)

// newBehavioralObservationWithKeystrokes builds a BehavioralObservation
// with the given actor + keystroke intervals (raw nanoseconds; the
// signature quantizes to 50ms buckets at fingerprint formation).
func newBehavioralObservationWithKeystrokes(actorRef string, intervals []*eventsv1.KeystrokeInterval, observedAt int64) *eventsv1.BehavioralObservation {
	return &eventsv1.BehavioralObservation{
		ObservedAt:          observedAt,
		ActorRef:            actorRef,
		CollectorRef:        "browser-sdk:v1",
		AuthenticationClass: commonv1.AuthenticationClass_AUTHENTICATION_CLASS_CLIENT_WITNESSED,
		Modality: &eventsv1.BehavioralObservation_KeystrokeTiming{
			KeystrokeTiming: &eventsv1.BehavioralKeystrokeTiming{
				Intervals:           intervals,
				TotalKeystrokeCount: uint32(len(intervals)),
			},
		},
	}
}

// makeIntervals constructs a slice of KeystrokeInterval from
// flat (flight_ns, dwell_ns) pairs. Helper to keep test fixtures
// terse.
func makeIntervals(pairsNs ...uint64) []*eventsv1.KeystrokeInterval {
	if len(pairsNs)%2 != 0 {
		panic("makeIntervals: pairsNs length must be even")
	}
	out := make([]*eventsv1.KeystrokeInterval, 0, len(pairsNs)/2)
	for i := 0; i < len(pairsNs); i += 2 {
		out = append(out, &eventsv1.KeystrokeInterval{
			FlightNs: pairsNs[i],
			DwellNs:  pairsNs[i+1],
		})
	}
	return out
}

func TestKeystrokeTimingClusteringV1_NameAndSubtype(t *testing.T) {
	sig := &KeystrokeTimingClusteringV1{}
	if sig.Name() != "keystroke_timing_clustering_v1" {
		t.Errorf("Name: got %q want keystroke_timing_clustering_v1", sig.Name())
	}
	if sig.Subtype() != HypothesisSubtypeBehavioralCluster {
		t.Errorf("Subtype: got %v want BehavioralCluster", sig.Subtype())
	}
}

func TestKeystrokeTimingClusteringV1_SatisfiesBehavioralSignatureInterface(t *testing.T) {
	var _ BehavioralSignature = &KeystrokeTimingClusteringV1{}
}

func TestKeystrokeTimingClusteringV1_BelowThreshold_NoCandidate(t *testing.T) {
	sig := &KeystrokeTimingClusteringV1{}
	ivs := makeIntervals(50_000_000, 100_000_000, 0, 100_000_000, 100_000_000, 150_000_000)
	observations := []*eventsv1.BehavioralObservation{
		newBehavioralObservationWithKeystrokes("actor-a", ivs, 1),
		newBehavioralObservationWithKeystrokes("actor-b", ivs, 2),
	}
	res, err := sig.EvaluateBehavioral(context.Background(), observations)
	if err != nil {
		t.Fatalf("EvaluateBehavioral: %v", err)
	}
	if len(res.Candidates) != 0 {
		t.Errorf("expected 0 candidates (2 actors < threshold 3), got %d", len(res.Candidates))
	}
	if res.Stats.ActorsAggregated != 2 {
		t.Errorf("ActorsAggregated: got %d want 2", res.Stats.ActorsAggregated)
	}
}

func TestKeystrokeTimingClusteringV1_AtThreshold_OneCandidate(t *testing.T) {
	sig := &KeystrokeTimingClusteringV1{}
	ivs := makeIntervals(50_000_000, 100_000_000, 0, 100_000_000, 100_000_000, 150_000_000)
	observations := []*eventsv1.BehavioralObservation{
		newBehavioralObservationWithKeystrokes("actor-a", ivs, 1),
		newBehavioralObservationWithKeystrokes("actor-b", ivs, 2),
		newBehavioralObservationWithKeystrokes("actor-c", ivs, 3),
	}
	res, err := sig.EvaluateBehavioral(context.Background(), observations)
	if err != nil {
		t.Fatalf("EvaluateBehavioral: %v", err)
	}
	if len(res.Candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d", len(res.Candidates))
	}
	c := res.Candidates[0]
	if c.SignatureName != "keystroke_timing_clustering_v1" {
		t.Errorf("SignatureName: got %q", c.SignatureName)
	}
	if c.HypothesisSubtype != HypothesisSubtypeBehavioralCluster {
		t.Errorf("HypothesisSubtype: got %v want BehavioralCluster", c.HypothesisSubtype)
	}
	if !equalStringSlices(c.ActorRefs, []string{"actor-a", "actor-b", "actor-c"}) {
		t.Errorf("ActorRefs: got %v want [actor-a actor-b actor-c]", c.ActorRefs)
	}
	if len(c.SourceHashes) != 3 {
		t.Errorf("SourceHashes count: got %d want 3", len(c.SourceHashes))
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

// TestKeystrokeTimingClusteringV1_QuantizationClustersSimilarTimings
// validates the 50ms quantization: three actors with timings that
// quantize to the same bucket cluster together, even though raw
// timings differ slightly.
func TestKeystrokeTimingClusteringV1_QuantizationClustersSimilarTimings(t *testing.T) {
	sig := &KeystrokeTimingClusteringV1{}
	// All three pass quantize to the same buckets:
	// flight: ~50ms (round-half-up at 25ms boundary)
	// dwell: ~100ms (round-half-up at 75ms boundary)
	a := makeIntervals(45_000_000, 95_000_000, 0, 100_000_000, 100_000_000, 150_000_000)
	b := makeIntervals(50_000_000, 100_000_000, 5_000_000, 105_000_000, 95_000_000, 145_000_000)
	c := makeIntervals(55_000_000, 105_000_000, 10_000_000, 110_000_000, 110_000_000, 155_000_000)
	observations := []*eventsv1.BehavioralObservation{
		newBehavioralObservationWithKeystrokes("actor-a", a, 1),
		newBehavioralObservationWithKeystrokes("actor-b", b, 2),
		newBehavioralObservationWithKeystrokes("actor-c", c, 3),
	}
	res, err := sig.EvaluateBehavioral(context.Background(), observations)
	if err != nil {
		t.Fatalf("EvaluateBehavioral: %v", err)
	}
	if len(res.Candidates) != 1 {
		t.Fatalf("expected 1 candidate (quantization should cluster similar timings), got %d", len(res.Candidates))
	}
}

// TestKeystrokeTimingClusteringV1_QuantizationSeparatesDifferentRhythms
// validates the converse: actors with timings that quantize to
// DIFFERENT buckets do NOT cluster.
func TestKeystrokeTimingClusteringV1_QuantizationSeparatesDifferentRhythms(t *testing.T) {
	sig := &KeystrokeTimingClusteringV1{}
	// Three distinct rhythms (different bucket assignments):
	fast := makeIntervals(40_000_000, 60_000_000, 0, 80_000_000, 0, 70_000_000)
	medium := makeIntervals(150_000_000, 200_000_000, 100_000_000, 220_000_000, 130_000_000, 250_000_000)
	slow := makeIntervals(400_000_000, 500_000_000, 300_000_000, 550_000_000, 350_000_000, 600_000_000)
	observations := []*eventsv1.BehavioralObservation{
		newBehavioralObservationWithKeystrokes("actor-fast-1", fast, 1),
		newBehavioralObservationWithKeystrokes("actor-fast-2", fast, 2),
		newBehavioralObservationWithKeystrokes("actor-fast-3", fast, 3),
		newBehavioralObservationWithKeystrokes("actor-medium", medium, 4),
		newBehavioralObservationWithKeystrokes("actor-slow", slow, 5),
	}
	res, err := sig.EvaluateBehavioral(context.Background(), observations)
	if err != nil {
		t.Fatalf("EvaluateBehavioral: %v", err)
	}
	// Only the fast-rhythm cluster meets threshold 3.
	if len(res.Candidates) != 1 {
		t.Fatalf("expected 1 candidate (only fast cluster meets threshold), got %d", len(res.Candidates))
	}
	c := res.Candidates[0]
	if len(c.ActorRefs) != 3 {
		t.Errorf("cluster ActorRefs: got %d want 3", len(c.ActorRefs))
	}
}

func TestKeystrokeTimingClusteringV1_EmptyActorRef_Skipped(t *testing.T) {
	sig := &KeystrokeTimingClusteringV1{}
	ivs := makeIntervals(50_000_000, 100_000_000, 0, 100_000_000, 100_000_000, 150_000_000)
	obs := newBehavioralObservationWithKeystrokes("", ivs, 1)
	res, err := sig.EvaluateBehavioral(context.Background(), []*eventsv1.BehavioralObservation{obs})
	if err != nil {
		t.Fatalf("EvaluateBehavioral: %v", err)
	}
	if res.Stats.ObservationsSkippedNoActor != 1 {
		t.Errorf("ObservationsSkippedNoActor: got %d want 1", res.Stats.ObservationsSkippedNoActor)
	}
}

func TestKeystrokeTimingClusteringV1_InsufficientIntervals_Skipped(t *testing.T) {
	sig := &KeystrokeTimingClusteringV1{}
	// Only 2 intervals; below the 3-minimum.
	ivs := makeIntervals(50_000_000, 100_000_000, 0, 100_000_000)
	obs := newBehavioralObservationWithKeystrokes("actor-a", ivs, 1)
	res, err := sig.EvaluateBehavioral(context.Background(), []*eventsv1.BehavioralObservation{obs})
	if err != nil {
		t.Fatalf("EvaluateBehavioral: %v", err)
	}
	if res.Stats.ObservationsSkippedWrongModality != 1 {
		t.Errorf("ObservationsSkippedWrongModality: got %d want 1 (insufficient intervals)", res.Stats.ObservationsSkippedWrongModality)
	}
}

func TestKeystrokeTimingClusteringV1_WrongModality_Skipped(t *testing.T) {
	sig := &KeystrokeTimingClusteringV1{}
	// Mouse trajectory modality (not keystroke_timing) → skipped.
	obs := &eventsv1.BehavioralObservation{
		ObservedAt:   1,
		ActorRef:     "actor-a",
		CollectorRef: "browser-sdk:v1",
		Modality: &eventsv1.BehavioralObservation_MouseTrajectory{
			MouseTrajectory: &eventsv1.BehavioralMouseTrajectory{},
		},
	}
	res, err := sig.EvaluateBehavioral(context.Background(), []*eventsv1.BehavioralObservation{obs})
	if err != nil {
		t.Fatalf("EvaluateBehavioral: %v", err)
	}
	if res.Stats.ObservationsSkippedWrongModality != 1 {
		t.Errorf("ObservationsSkippedWrongModality: got %d want 1", res.Stats.ObservationsSkippedWrongModality)
	}
}

func TestKeystrokeTimingClusteringV1_EmptyInput_NoCandidates(t *testing.T) {
	sig := &KeystrokeTimingClusteringV1{}
	res, err := sig.EvaluateBehavioral(context.Background(), []*eventsv1.BehavioralObservation{})
	if err != nil {
		t.Fatalf("EvaluateBehavioral: %v", err)
	}
	if len(res.Candidates) != 0 {
		t.Errorf("expected 0 candidates for empty input, got %d", len(res.Candidates))
	}
	if res.Stats.ObservationsScanned != 0 {
		t.Errorf("ObservationsScanned: got %d want 0", res.Stats.ObservationsScanned)
	}
}

func TestKeystrokeTimingClusteringV1_DeterministicOrder(t *testing.T) {
	sig := &KeystrokeTimingClusteringV1{}
	// Two clusters both meeting threshold; verify candidates emitted
	// in canonical fingerprint-key ascending order.
	rhythm1 := makeIntervals(50_000_000, 100_000_000, 0, 100_000_000, 100_000_000, 150_000_000)
	rhythm2 := makeIntervals(150_000_000, 200_000_000, 100_000_000, 200_000_000, 200_000_000, 250_000_000)
	observations := []*eventsv1.BehavioralObservation{
		newBehavioralObservationWithKeystrokes("actor-r1-1", rhythm1, 1),
		newBehavioralObservationWithKeystrokes("actor-r1-2", rhythm1, 2),
		newBehavioralObservationWithKeystrokes("actor-r1-3", rhythm1, 3),
		newBehavioralObservationWithKeystrokes("actor-r2-1", rhythm2, 4),
		newBehavioralObservationWithKeystrokes("actor-r2-2", rhythm2, 5),
		newBehavioralObservationWithKeystrokes("actor-r2-3", rhythm2, 6),
	}
	res, err := sig.EvaluateBehavioral(context.Background(), observations)
	if err != nil {
		t.Fatalf("EvaluateBehavioral: %v", err)
	}
	if len(res.Candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(res.Candidates))
	}
	// Lexical ordering: rhythm2 "f150d200,..." sorts BEFORE rhythm1
	// "f50d100,..." because "1" < "5" at position 1 of the fingerprint
	// string (after "f"). String sort is character-by-character, not
	// numeric.
	first := res.Candidates[0].ActorRefs[0]
	if first != "actor-r2-1" {
		t.Errorf("first candidate first actor: got %q want actor-r2-1 (rhythm2 'f150...' sorts before rhythm1 'f50...' lex)", first)
	}
}

func TestCanonicalKeystrokeFingerprint_Quantization(t *testing.T) {
	cases := []struct {
		name      string
		intervals []*eventsv1.KeystrokeInterval
		want      string
	}{
		{
			name:      "exact bucket boundaries",
			intervals: makeIntervals(50_000_000, 100_000_000),
			want:      "f50d100",
		},
		{
			name: "round half-up at midpoint",
			// 25_000_000 is exactly halfway between bucket 0 and bucket 50;
			// round-half-up → bucket 50; 75_000_000 → bucket 100.
			intervals: makeIntervals(25_000_000, 75_000_000),
			want:      "f50d100",
		},
		{
			name:      "round down below midpoint",
			intervals: makeIntervals(24_999_999, 74_999_999),
			want:      "f0d50",
		},
		{
			name:      "empty intervals",
			intervals: makeIntervals(),
			want:      "",
		},
	}
	for _, tc := range cases {
		got := canonicalKeystrokeFingerprint(tc.intervals)
		if got != tc.want {
			t.Errorf("%s: got %q want %q", tc.name, got, tc.want)
		}
	}
}
