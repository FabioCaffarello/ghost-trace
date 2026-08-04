// Does a judgement made from a snapshot equal the judgement the
// collector would have made in process?
//
// That question is the whole of PR-2.3. If the answer is ever no, the
// split changes what the detector decides — which is a different
// product, not a different topology.
package snapshot

import (
	"math"
	"math/rand"
	"testing"

	"github.com/FabioCaffarello/ghost-trace/libs/feature"
	"github.com/FabioCaffarello/ghost-trace/libs/policy"
)

// The mapping narrows float64 ratios to float32 on the wire, so the
// round trip is not the identity. This is the size of that loss, and it
// is asserted rather than assumed: float32 carries about 7 decimal
// digits, so anything beyond ~1e-7 relative would mean a field is being
// dropped or mistyped rather than merely rounded.
const maxRelativeLoss = 1e-6

func decide(st policy.State, action string) (score, confidence float64, decision string) {
	j := policy.Judge(st, action)
	out, err := policy.Apply(j, policy.ModeEnforce)
	if err != nil {
		panic(err)
	}
	return j.Score(), j.Confidence(), out.Decision
}

func relDiff(a, b float64) float64 {
	if a == b {
		return 0
	}
	d := math.Abs(a - b)
	if m := math.Max(math.Abs(a), math.Abs(b)); m > 0 {
		return d / m
	}
	return d
}

// A spread of states, including the ones that sit near a threshold —
// which is where a narrowing would actually flip a decision.
func states() map[string]policy.State {
	return map[string]policy.State{
		"empty": {},
		"straight-line bot": {
			Pointer: feature.PointerState{Straightness: 1.0, Segments: 1, PathPx: 900, Points: 60},
		},
		"humanised": {
			Pointer:   feature.PointerState{Straightness: 0.9421, PathPx: 1183.77, Segments: 4, Points: 214},
			Keystroke: feature.KeystrokeState{FlightCV: 0.3123, DwellCV: 0.2718, MeanDwellMs: 84.31, Keys: 18, Intervals: 17},
		},
		"value injection": {
			Pointer:     feature.PointerState{Straightness: 0.88, PathPx: 640.5, Segments: 3, Points: 120},
			Interaction: feature.InteractionState{Injections: 2, InjectedFields: 2, FocusTransitions: 0, DistinctFocusTargets: 2},
		},
		"awkward precision": {
			Pointer: feature.PointerState{
				Straightness: 0.900000119, // straddles the straightness floor
				PathPx:       1.0 / 3.0,
				Segments:     7, Points: 333,
			},
			Keystroke: feature.KeystrokeState{
				FlightCV: 0.1234567890123, DwellCV: 1e-8, MeanDwellMs: 1e7 + 0.5,
				ExactRepeatRatio: 2.0 / 3.0, Keys: 41, Intervals: 40,
			},
			Interaction: feature.InteractionState{ProgrammaticScrollRatio: 1.0 / 7.0, ScrollEvents: 13},
		},
	}
}

func TestSnapshotPreservesTheDecision(t *testing.T) {
	for name, st := range states() {
		t.Run(name, func(t *testing.T) {
			for _, action := range []string{"login", "checkout"} {
				wantScore, wantConf, wantDecision := decide(st, action)
				gotScore, gotConf, gotDecision := decide(RoundTrip(st), action)

				if gotDecision != wantDecision {
					t.Errorf("%s: decision through a snapshot is %q, in process it is %q — "+
						"the split would change what the detector decides",
						action, gotDecision, wantDecision)
				}
				if d := relDiff(wantScore, gotScore); d > maxRelativeLoss {
					t.Errorf("%s: score drifts %g through the snapshot (%v -> %v)",
						action, d, wantScore, gotScore)
				}
				if d := relDiff(wantConf, gotConf); d > maxRelativeLoss {
					t.Errorf("%s: confidence drifts %g through the snapshot (%v -> %v)",
						action, d, wantConf, gotConf)
				}
			}
		})
	}
}

// Every field must survive. A mapping that silently drops one would
// pass the decision test whenever that field happens not to move the
// score — and then fail in production on the session where it does.
func TestEveryFieldSurvivesTheRoundTrip(t *testing.T) {
	st := policy.State{
		Pointer:     feature.PointerState{Straightness: 0.5, Segments: 11, PathPx: 123.5, Points: 77},
		Keystroke:   feature.KeystrokeState{FlightCV: 0.25, DwellCV: 0.75, MeanDwellMs: 42.5, ExactRepeatRatio: 0.125, Intervals: 9, Keys: 10},
		Interaction: feature.InteractionState{ProgrammaticScrollRatio: 0.375, ScrollEvents: 3, FocusTransitions: 4, HiddenPeriods: 5, Pastes: 6, Injections: 7, InjectedFields: 8, Autofills: 9, DistinctFocusTargets: 10},
	}
	got := RoundTrip(st)

	// Values chosen to be exactly representable in float32, so any
	// difference here is a dropped or crossed field rather than
	// rounding.
	if got != st {
		t.Errorf("a field did not survive the round trip:\n  in:  %+v\n  out: %+v", st, got)
	}
}

// Fuzzing the whole space, because the hand-picked states above are the
// ones I thought of.
func TestRandomStatesKeepTheirDecision(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	for i := 0; i < 2000; i++ {
		st := policy.State{
			Pointer: feature.PointerState{
				Straightness: r.Float64(), PathPx: r.Float64() * 5000,
				Segments: uint32(r.Intn(50)), Points: uint32(r.Intn(1000)),
			},
			Keystroke: feature.KeystrokeState{
				FlightCV: r.Float64(), DwellCV: r.Float64(), MeanDwellMs: r.Float64() * 500,
				ExactRepeatRatio: r.Float64(), Intervals: uint32(r.Intn(200)), Keys: uint32(r.Intn(200)),
			},
			Interaction: feature.InteractionState{
				ProgrammaticScrollRatio: r.Float64(), ScrollEvents: uint32(r.Intn(100)),
				FocusTransitions: uint32(r.Intn(20)), HiddenPeriods: uint32(r.Intn(10)),
				Pastes: uint32(r.Intn(5)), Injections: uint32(r.Intn(5)),
				InjectedFields: uint32(r.Intn(5)), Autofills: uint32(r.Intn(5)),
				DistinctFocusTargets: uint32(r.Intn(10)),
			},
		}
		_, _, want := decide(st, "login")
		_, _, got := decide(RoundTrip(st), "login")
		if got != want {
			t.Fatalf("state %d decides %q in process and %q through a snapshot:\n%+v",
				i, want, got, st)
		}
	}
}
