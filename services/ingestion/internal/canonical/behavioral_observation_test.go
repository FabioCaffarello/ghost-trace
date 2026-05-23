// Focused tests for BehavioralObservation per decision-log §0146 + RFC
// Domain Pack v0.1 anti-bot atlas F1.BehavioralObservation. Follows
// §0144 NetworkObservation test pattern (round-trip per modality,
// oneof exhaustiveness, envelope-field discipline). Extends with
// session_ref discipline (new envelope field introduced for
// BehavioralObservation per §0146 (d)).
//
// Paired-dimension exclusion for BehavioralObservation + its 4
// sub-modality types is asserted by
// TestPairedDimensionSubjectSet_StructuralDetection in coverage_test.go
// (which automatically picks up the messageFactory registration in
// corpus_test.go) — no explicit assertion needed here.
package canonical

import (
	"bytes"
	"testing"

	eventsv1 "github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/genproto/events/v1"
)

func newBehavioralObservationWithEachModality() []struct {
	name string
	obs  *eventsv1.BehavioralObservation
} {
	const (
		observedAt   = int64(1716120000000000000)
		actorRef     = "actor-behavioral-test"
		sessionRef   = "session-abc-123"
		collectorRef = "synthetic-behavioral-gen:v0"
	)
	envelope := func() *eventsv1.BehavioralObservation {
		return &eventsv1.BehavioralObservation{
			ObservedAt:   observedAt,
			ActorRef:     actorRef,
			SessionRef:   sessionRef,
			CollectorRef: collectorRef,
		}
	}
	withMouse := envelope()
	withMouse.Modality = &eventsv1.BehavioralObservation_MouseTrajectory{
		MouseTrajectory: &eventsv1.BehavioralMouseTrajectory{
			Samples: []*eventsv1.MouseSample{
				{OffsetNs: 0, X: 100, Y: 100},
				{OffsetNs: 50000000, X: 150, Y: 120},
				{OffsetNs: 100000000, X: 200, Y: 140},
			},
			ClickCount:     2,
			ViewportWidth:  1920,
			ViewportHeight: 1080,
		},
	}
	withKeystroke := envelope()
	withKeystroke.Modality = &eventsv1.BehavioralObservation_KeystrokeTiming{
		KeystrokeTiming: &eventsv1.BehavioralKeystrokeTiming{
			Intervals: []*eventsv1.KeystrokeInterval{
				{FlightNs: 0, DwellNs: 80000000},
				{FlightNs: 120000000, DwellNs: 95000000},
				{FlightNs: 110000000, DwellNs: 75000000},
			},
			TotalKeystrokeCount: 3,
		},
	}
	withScroll := envelope()
	withScroll.Modality = &eventsv1.BehavioralObservation_ScrollCadence{
		ScrollCadence: &eventsv1.BehavioralScrollCadence{
			Samples: []*eventsv1.ScrollSample{
				{OffsetNs: 0, DeltaX: 0, DeltaY: 100},
				{OffsetNs: 200000000, DeltaX: 0, DeltaY: 80},
			},
			TotalScrollEvents: 2,
		},
	}
	withDwell := envelope()
	withDwell.Modality = &eventsv1.BehavioralObservation_PointerDwell{
		PointerDwell: &eventsv1.BehavioralPointerDwell{
			ElementClass:    "form-input",
			DwellNs:         1500000000,
			TransitionCount: 1,
		},
	}
	return []struct {
		name string
		obs  *eventsv1.BehavioralObservation
	}{
		{"mouse_trajectory", withMouse},
		{"keystroke_timing", withKeystroke},
		{"scroll_cadence", withScroll},
		{"pointer_dwell", withDwell},
	}
}

// TestBehavioralObservation_RoundTripPerModality asserts that for each
// of the four sub-modalities, Marshal is deterministic and hash is
// stable. Mirrors TestNetworkObservation_RoundTripPerModality.
func TestBehavioralObservation_RoundTripPerModality(t *testing.T) {
	for _, tc := range newBehavioralObservationWithEachModality() {
		t.Run(tc.name, func(t *testing.T) {
			b1, err := Marshal(tc.obs)
			if err != nil {
				t.Fatalf("first Marshal: %v", err)
			}
			b2, err := Marshal(tc.obs)
			if err != nil {
				t.Fatalf("second Marshal: %v", err)
			}
			if !bytes.Equal(b1, b2) {
				t.Fatalf("non-deterministic marshal:\n  b1=%x\n  b2=%x", b1, b2)
			}
			_, h, err := MarshalAndHash(tc.obs)
			if err != nil {
				t.Fatalf("MarshalAndHash: %v", err)
			}
			h2 := Hash(b1)
			if h != h2 {
				t.Fatalf("hash unstable: %x != %x", h, h2)
			}
		})
	}
}

// TestBehavioralObservation_OneofExhaustiveness asserts that all four
// sub-modalities are constructable + marshalable + produce structurally
// distinct canonical bytes from the same envelope-level field values.
// Discriminator (oneof field number) must propagate into canonical
// bytes such that two BehavioralObservation records with the same
// envelope but different sub-modality variants do not collide.
func TestBehavioralObservation_OneofExhaustiveness(t *testing.T) {
	hashes := map[string][32]byte{}
	for _, tc := range newBehavioralObservationWithEachModality() {
		_, h, err := MarshalAndHash(tc.obs)
		if err != nil {
			t.Fatalf("MarshalAndHash %s: %v", tc.name, err)
		}
		for prior, priorHash := range hashes {
			if h == priorHash {
				t.Fatalf("hash collision between sub-modalities %s and %s: %x", prior, tc.name, h)
			}
		}
		hashes[tc.name] = h
	}
	if got, want := len(hashes), 4; got != want {
		t.Fatalf("expected %d distinct sub-modality hashes, got %d", want, got)
	}
}

// TestBehavioralObservation_ActorRefDiscipline asserts that
// BehavioralObservation can be marshaled with an empty actor_ref
// (unattributed observation per §0023 single-tier discipline) and
// with a populated actor_ref; both produce stable distinct hashes.
func TestBehavioralObservation_ActorRefDiscipline(t *testing.T) {
	base := func() *eventsv1.BehavioralObservation {
		return &eventsv1.BehavioralObservation{
			ObservedAt:   1716120000000000000,
			SessionRef:   "session-abc-123",
			CollectorRef: "synthetic-behavioral-gen:v0",
			Modality: &eventsv1.BehavioralObservation_PointerDwell{
				PointerDwell: &eventsv1.BehavioralPointerDwell{
					ElementClass: "form-input",
					DwellNs:      1000000000,
				},
			},
		}
	}
	unattributed := base()
	attributed := base()
	attributed.ActorRef = "actor-x"

	_, hU, err := MarshalAndHash(unattributed)
	if err != nil {
		t.Fatalf("MarshalAndHash unattributed: %v", err)
	}
	_, hA, err := MarshalAndHash(attributed)
	if err != nil {
		t.Fatalf("MarshalAndHash attributed: %v", err)
	}
	if hU == hA {
		t.Fatalf("empty vs populated actor_ref produced same hash: %x", hU)
	}
}

// TestBehavioralObservation_SessionRefDiscipline asserts that the
// session_ref envelope field (new in BehavioralObservation per §0146
// (d)) participates in hash derivation — two otherwise-identical
// observations under different session contexts produce distinct
// hashes. This is the substrate-level guarantee that session-context
// anchoring is content-addressed even though session_ref is a
// producer-supplied string (parallel to actor_ref pattern, not a
// substrate-verifiable BLAKE3 reference).
func TestBehavioralObservation_SessionRefDiscipline(t *testing.T) {
	base := func() *eventsv1.BehavioralObservation {
		return &eventsv1.BehavioralObservation{
			ObservedAt:   1716120000000000000,
			ActorRef:     "actor-x",
			CollectorRef: "synthetic-behavioral-gen:v0",
			Modality: &eventsv1.BehavioralObservation_PointerDwell{
				PointerDwell: &eventsv1.BehavioralPointerDwell{
					ElementClass: "form-input",
					DwellNs:      1000000000,
				},
			},
		}
	}
	sessionA := base()
	sessionA.SessionRef = "session-abc-123"
	sessionB := base()
	sessionB.SessionRef = "session-xyz-456"

	_, hA, err := MarshalAndHash(sessionA)
	if err != nil {
		t.Fatalf("MarshalAndHash sessionA: %v", err)
	}
	_, hB, err := MarshalAndHash(sessionB)
	if err != nil {
		t.Fatalf("MarshalAndHash sessionB: %v", err)
	}
	if hA == hB {
		t.Fatalf("different session_ref produced same hash: %x", hA)
	}
}

// TestBehavioralObservation_CollectorRefDiscipline asserts that
// collector_ref participates in hash derivation for BehavioralObservation
// (parallel to TestNetworkObservation_CollectorRefDiscipline). Two
// otherwise-identical observations from different collectors produce
// distinct hashes — the substrate-level guarantee that the
// phenomenon-vs-record reconciliation OMQ resolution can rest on.
func TestBehavioralObservation_CollectorRefDiscipline(t *testing.T) {
	base := func() *eventsv1.BehavioralObservation {
		return &eventsv1.BehavioralObservation{
			ObservedAt: 1716120000000000000,
			ActorRef:   "actor-x",
			SessionRef: "session-abc-123",
			Modality: &eventsv1.BehavioralObservation_PointerDwell{
				PointerDwell: &eventsv1.BehavioralPointerDwell{
					ElementClass: "form-input",
					DwellNs:      1000000000,
				},
			},
		}
	}
	fromSynthetic := base()
	fromSynthetic.CollectorRef = "synthetic-behavioral-gen:v0"
	fromHoneypot := base()
	fromHoneypot.CollectorRef = "honeypot-behavioral-collector:hp-01"

	_, hS, err := MarshalAndHash(fromSynthetic)
	if err != nil {
		t.Fatalf("MarshalAndHash synthetic: %v", err)
	}
	_, hH, err := MarshalAndHash(fromHoneypot)
	if err != nil {
		t.Fatalf("MarshalAndHash honeypot: %v", err)
	}
	if hS == hH {
		t.Fatalf("different collector_ref produced same hash: %x", hS)
	}
}
