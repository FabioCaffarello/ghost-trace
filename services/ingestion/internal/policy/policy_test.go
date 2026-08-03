package policy

import (
	"testing"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/feature"
)

// strong is a feature vector with a perfectly linear path and ample
// evidence: the case that should reach block under enforce.
func strong() feature.PointerState {
	return feature.PointerState{
		Straightness: 1.0,
		Segments:     10,
		PathPx:       4000,
		Points:       800,
	}
}

func TestColdStartNeverBlocks(t *testing.T) {
	// The contract's central cold-start commitment: a session eleven
	// events old can produce a high score and must not produce a block.
	st := feature.PointerState{
		Straightness: 1.0, // maximally bot-like
		Segments:     1,
		PathPx:       60,
		Points:       11,
	}
	j := Judge(st)

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
	thin := Judge(feature.PointerState{Straightness: 1.0, Segments: 1, PathPx: 60, Points: 11})
	thick := Judge(feature.PointerState{Straightness: 1.0, Segments: 10, PathPx: 4000, Points: 800})

	if thin.Score() != thick.Score() {
		t.Fatalf("scores differ (%v vs %v); this test isolates confidence", thin.Score(), thick.Score())
	}
	if enforced(thin) == enforced(thick) {
		t.Errorf("evidence volume did not change the decision: both %q", enforced(thin))
	}
}

func TestStrongLinearEvidenceBlocksUnderEnforce(t *testing.T) {
	out, err := Apply(Judge(strong()), ModeEnforce)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.Decision != DecisionBlock {
		t.Errorf("Decision = %q, want %q", out.Decision, DecisionBlock)
	}
	if out.Shadow != "" {
		t.Errorf("Shadow = %q, want empty under enforce", out.Shadow)
	}
}

func TestMonitorAlwaysAllowsAndReportsShadow(t *testing.T) {
	// Every integration starts in monitor; it must never return
	// anything but allow, no matter how damning the evidence.
	out, err := Apply(Judge(strong()), ModeMonitor)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.Decision != DecisionAllow {
		t.Errorf("Decision = %q, want %q in monitor mode", out.Decision, DecisionAllow)
	}
	if out.Shadow != DecisionBlock {
		t.Errorf("Shadow = %q, want %q", out.Shadow, DecisionBlock)
	}
}

func TestHumanLikePathAllows(t *testing.T) {
	// Straightness below the floor is unremarkable regardless of how
	// much of it there is. This is the false-positive guard.
	j := Judge(feature.PointerState{
		Straightness: 0.88,
		Segments:     20,
		PathPx:       9000,
		Points:       1500,
	})
	if j.Score() != 0 {
		t.Errorf("Score = %v, want 0 below the straightness floor", j.Score())
	}
	if got := enforced(j); got != DecisionAllow {
		t.Errorf("enforced = %q, want %q for a human-like path", got, DecisionAllow)
	}
}

func TestNoEvidenceYieldsZeroesAndAllow(t *testing.T) {
	j := Judge(feature.PointerState{})
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
	j := Judge(feature.PointerState{Straightness: 1.0, Segments: 1, PathPx: 60, Points: 11})

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
	j := Judge(feature.PointerState{Straightness: 0.5, Segments: 8, PathPx: 3000, Points: 500})
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
		j := Judge(st)
		if j.Score() < 0 || j.Score() > 1 {
			t.Errorf("case %d: Score = %v, out of [0,1]", i, j.Score())
		}
		if j.Confidence() < 0 || j.Confidence() > 1 {
			t.Errorf("case %d: Confidence = %v, out of [0,1]", i, j.Confidence())
		}
	}
}

func TestUnknownModeIsRejected(t *testing.T) {
	if _, err := Apply(Judge(strong()), "blocking"); err == nil {
		t.Error("expected an error for an unknown mode")
	}
}
