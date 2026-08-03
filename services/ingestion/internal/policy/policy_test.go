package policy

import (
	"testing"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/feature"
)

// strong is a feature vector with a perfectly linear path and ample
// evidence: the case that should reach block under enforce.
func strong() State {
	return State{Pointer: feature.PointerState{
		Straightness: 1.0,
		Segments:     10,
		PathPx:       4000,
		Points:       800,
	}}
}

// ptr wraps a pointer-only feature vector.
func ptr(p feature.PointerState) State { return State{Pointer: p} }

const testAction = "checkout" // pointer-dominant, so pointer-only cases score

func TestColdStartNeverBlocks(t *testing.T) {
	// The contract's central cold-start commitment: a session eleven
	// events old can produce a high score and must not produce a block.
	st := feature.PointerState{
		Straightness: 1.0, // maximally bot-like
		Segments:     1,
		PathPx:       60,
		Points:       11,
	}
	j := Judge(ptr(st), testAction)

	if j.Score() < 0.99 {
		t.Fatalf("Score = %v; setup should produce a high score", j.Score())
	}
	if got := enforced(j); got == DecisionBlock {
		t.Errorf("enforced = block on %d events; cold start must never block", st.Points)
	}
}

func TestConfidenceGatesBeforeScore(t *testing.T) {
	// Same maximal score, more evidence — the decision must change.
	// If it does not, the two dimensions are not actually separate.
	thin := Judge(ptr(feature.PointerState{Straightness: 1.0, Segments: 1, PathPx: 60, Points: 11}), testAction)
	thick := Judge(ptr(feature.PointerState{Straightness: 1.0, Segments: 10, PathPx: 4000, Points: 800}), testAction)

	if thin.Score() != thick.Score() {
		t.Fatalf("scores differ (%v vs %v); this test isolates confidence", thin.Score(), thick.Score())
	}
	if enforced(thin) == enforced(thick) {
		t.Errorf("evidence volume did not change the decision: both %q", enforced(thin))
	}
}

func TestSingleChannelCannotBlock(t *testing.T) {
	// Damning pointer evidence and nothing else. Since M3 weights each
	// channel by (action weight x its own confidence), no single channel
	// can reach the block threshold alone — blocking requires
	// corroboration across independent evidence streams.
	//
	// This is v1's evidential-independence idea surviving in a new form:
	// a belief must not inflate on one source, however strong.
	out, err := Apply(Judge(strong(), testAction), ModeEnforce)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.Decision == DecisionBlock {
		t.Errorf("Decision = block on one evidence channel; blocks must require corroboration")
	}
	if out.Decision != DecisionChallenge {
		t.Errorf("Decision = %q, want %q on strong single-channel evidence", out.Decision, DecisionChallenge)
	}
	if out.Shadow != "" {
		t.Errorf("Shadow = %q, want empty under enforce", out.Shadow)
	}
}

func TestCorroboratedEvidenceBlocks(t *testing.T) {
	// Pointer AND typing both damning: now a block is warranted.
	st := State{
		Pointer: feature.PointerState{Straightness: 1.0, Segments: 10, PathPx: 4000, Points: 800},
		Keystroke: feature.KeystrokeState{
			FlightCV: 0.01, MeanDwellMs: 0.5, ExactRepeatRatio: 0.9,
			Intervals: 40, Keys: 80,
		},
		Interaction: feature.InteractionState{
			ScrollEvents: 20, ProgrammaticScrollRatio: 1.0,
		},
	}
	out, err := Apply(Judge(st, "login"), ModeEnforce)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.Decision != DecisionBlock {
		t.Errorf("Decision = %q, want %q with three corroborating channels", out.Decision, DecisionBlock)
	}
}

func TestMonitorAlwaysAllowsAndReportsShadow(t *testing.T) {
	// Every integration starts in monitor; it must never return
	// anything but allow, no matter how damning the evidence.
	out, err := Apply(Judge(strong(), testAction), ModeMonitor)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.Decision != DecisionAllow {
		t.Errorf("Decision = %q, want %q in monitor mode", out.Decision, DecisionAllow)
	}
	if out.Shadow == "" || out.Shadow == DecisionAllow {
		t.Errorf("Shadow = %q, want a non-allow enforce decision", out.Shadow)
	}
}

func TestHumanLikePathAllows(t *testing.T) {
	// Straightness below the floor is unremarkable regardless of how
	// much of it there is. This is the false-positive guard.
	j := Judge(ptr(feature.PointerState{
		Straightness: 0.88,
		Segments:     20,
		PathPx:       9000,
		Points:       1500,
	}), testAction)
	if j.Score() != 0 {
		t.Errorf("Score = %v, want 0 below the straightness floor", j.Score())
	}
	if got := enforced(j); got != DecisionAllow {
		t.Errorf("enforced = %q, want %q for a human-like path", got, DecisionAllow)
	}
}

func TestNoEvidenceYieldsZeroesAndAllow(t *testing.T) {
	j := Judge(ptr(feature.PointerState{}), testAction)
	if j.Score() != 0 || j.Confidence() != 0 {
		t.Errorf("Score=%v Confidence=%v, want 0/0 with no evidence", j.Score(), j.Confidence())
	}
	if got := enforced(j); got != DecisionAllow {
		t.Errorf("enforced = %q, want %q", got, DecisionAllow)
	}
}

func TestInsufficientEvidenceReasonExplainsAllow(t *testing.T) {
	// A high score that still allows needs to say why, or the caller
	// sees an unexplained contradiction.
	j := Judge(ptr(feature.PointerState{Straightness: 1.0, Segments: 1, PathPx: 60, Points: 11}), testAction)

	var found bool
	for _, r := range j.Reasons() {
		if r.Code == ReasonInsufficientEvidence {
			found = true
		}
	}
	if !found {
		t.Errorf("reasons %+v missing %s", j.Reasons(), ReasonInsufficientEvidence)
	}
}

func TestLinearityReasonAbsentWhenScoreIsZero(t *testing.T) {
	j := Judge(ptr(feature.PointerState{Straightness: 0.5, Segments: 8, PathPx: 3000, Points: 500}), testAction)
	for _, r := range j.Reasons() {
		if r.Code == ReasonPointerLinearity {
			t.Errorf("%s reported with score 0", ReasonPointerLinearity)
		}
	}
}

func TestScoreAndConfidenceStayInUnitRange(t *testing.T) {
	cases := []feature.PointerState{
		{},
		{Straightness: 1, Segments: 1000, PathPx: 1e9, Points: 1e6},
		{Straightness: 2, Segments: 5, PathPx: 500},  // out-of-range input
		{Straightness: -1, Segments: 5, PathPx: 500}, // out-of-range input
	}
	for i, st := range cases {
		j := Judge(ptr(st), testAction)
		if j.Score() < 0 || j.Score() > 1 {
			t.Errorf("case %d: Score = %v, out of [0,1]", i, j.Score())
		}
		if j.Confidence() < 0 || j.Confidence() > 1 {
			t.Errorf("case %d: Confidence = %v, out of [0,1]", i, j.Confidence())
		}
	}
}

func TestUnknownModeIsRejected(t *testing.T) {
	if _, err := Apply(Judge(strong(), testAction), "blocking"); err == nil {
		t.Error("expected an error for an unknown mode")
	}
}
