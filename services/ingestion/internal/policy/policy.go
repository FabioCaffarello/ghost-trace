// Package policy maps features to a judgement, and a judgement to a
// decision.
//
// The split between the two is contract §3's most important commitment:
// `score` is how bot-like the behaviour looks, `confidence` is how much
// evidence supports that judgement. A session eleven events old can
// produce a high score and must not produce a block. Answering
// cold-start here rather than inside a model means every consumer
// handles it identically.
//
// This is the surviving form of v1's paired-dimension rule (see
// docs/v1-retrospective.md). v1 enforced it by walking proto fields at
// the marshalling boundary; here a Judgement simply cannot be
// constructed without both numbers, which is stronger and about
// forty lines shorter.
package policy

import (
	"fmt"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/feature"
)

// Ref versions the scoring definition. Stored on every evaluation so a
// record stays interpretable after the constants below move.
const Ref = "pointer-linearity-v1"

// Reason codes are a stable enumeration (contract §7). Adding one is
// non-breaking; changing what one means is not.
const (
	// ReasonPointerLinearity: the pointer path is straighter than human
	// motor control usually produces.
	ReasonPointerLinearity = "POINTER_LINEARITY"

	// ReasonInsufficientEvidence: too little qualifying movement to
	// support any judgement. Present whenever confidence is below the
	// challenge threshold, including when the score is high — it is the
	// reason a bot-looking session was still allowed.
	ReasonInsufficientEvidence = "INSUFFICIENT_EVIDENCE"
)

// Decisions.
const (
	DecisionAllow     = "allow"
	DecisionChallenge = "challenge"
	DecisionBlock     = "block"
)

// Modes (contract §4).
const (
	// ModeMonitor always returns allow and reports what enforce would
	// have done. Every integration starts here — nobody puts an
	// untested blocker in front of paying users.
	ModeMonitor = "monitor"
	ModeEnforce = "enforce"
)

// Scoring constants.
//
// UNCALIBRATED. These are inception-phase guesses, not measurements.
// There is no adversary and no human sample yet, so no number here is
// evidence of anything — M2 exists to replace them, and the M2 write-up
// reports what they moved to. Treating them as tuned before that would
// be exactly the unfalsifiable-metric failure the project is trying to
// avoid.
const (
	// Below this, straightness is unremarkable for a human.
	straightnessFloor = 0.90

	// At or above this, the path is straighter than hand motion
	// plausibly produces.
	straightnessCeil = 0.995

	// Evidence saturation points for confidence.
	confidenceSegments = 6.0
	confidencePathPx   = 1500.0

	// Cold-start floors. A block is a serious act and requires strong
	// evidence; the asymmetry is deliberate, because blocking a human
	// is far worse than admitting a bot (contract §9).
	minConfidenceToChallenge = 0.40
	minConfidenceToBlock     = 0.70

	// Score thresholds, applied only once the confidence floors above
	// are met.
	scoreToChallenge = 0.50
	scoreToBlock     = 0.80
)

// Judgement pairs a score with the confidence supporting it.
//
// The fields are unexported and there is no literal construction path:
// the only way to obtain one is Judge, which always sets both. That is
// the entire enforcement mechanism, and it is not bypassable the way a
// validated struct literal is.
type Judgement struct {
	score      float64
	confidence float64
	reasons    []Reason
}

// Reason is one contributing factor.
type Reason struct {
	Code   string  `json:"code"`
	Weight float64 `json:"weight"`
}

// Score is how bot-like the behaviour looks, in [0, 1].
func (j Judgement) Score() float64 { return j.score }

// Confidence is how much evidence supports the score, in [0, 1].
func (j Judgement) Confidence() float64 { return j.confidence }

// Reasons returns the contributing factors, most significant first.
func (j Judgement) Reasons() []Reason { return j.reasons }

// Judge maps a feature vector to a judgement.
func Judge(st feature.PointerState) Judgement {
	score := scoreFromStraightness(st.Straightness)
	conf := confidenceFromEvidence(st)

	// With no qualifying movement at all there is nothing to score.
	// Reporting score 0 here would be a claim ("looks human") rather
	// than an absence of evidence; the confidence field is what carries
	// the distinction, so score stays 0 and confidence stays 0.
	reasons := make([]Reason, 0, 2)
	if st.Segments > 0 && score > 0 {
		reasons = append(reasons, Reason{Code: ReasonPointerLinearity, Weight: score})
	}
	if conf < minConfidenceToChallenge {
		reasons = append(reasons, Reason{Code: ReasonInsufficientEvidence, Weight: 1 - conf})
	}

	return Judgement{score: score, confidence: conf, reasons: reasons}
}

// scoreFromStraightness maps straightness onto [0, 1].
//
// The band is narrow — a purposeful human move often lands around 0.90
// and a linear script sits at 1.0 — which is precisely why this feature
// is weak on its own and why M2 must measure it before anything is
// built on top of it.
func scoreFromStraightness(s float64) float64 {
	if s <= straightnessFloor {
		return 0
	}
	if s >= straightnessCeil {
		return 1
	}
	return (s - straightnessFloor) / (straightnessCeil - straightnessFloor)
}

// confidenceFromEvidence combines segment count and path length.
//
// Both terms are needed: six flicks across a button is not the same
// evidence as one long deliberate drag, and either alone can be gamed
// by an adversary who knows the threshold.
func confidenceFromEvidence(st feature.PointerState) float64 {
	segs := float64(st.Segments) / confidenceSegments
	if segs > 1 {
		segs = 1
	}
	path := st.PathPx / confidencePathPx
	if path > 1 {
		path = 1
	}
	return 0.5*segs + 0.5*path
}

// Outcome is the result of applying a mode to a judgement.
type Outcome struct {
	// Decision is what the caller must act on.
	Decision string

	// Shadow is what enforce mode would have returned. Empty unless
	// mode is monitor.
	Shadow string
}

// Apply maps a judgement to a decision under the given mode.
func Apply(j Judgement, mode string) (Outcome, error) {
	switch mode {
	case ModeMonitor:
		return Outcome{Decision: DecisionAllow, Shadow: enforced(j)}, nil
	case ModeEnforce:
		return Outcome{Decision: enforced(j)}, nil
	default:
		return Outcome{}, fmt.Errorf("policy.Apply: unknown mode %q (want %q or %q)", mode, ModeMonitor, ModeEnforce)
	}
}

// enforced is the decision enforce mode would return.
//
// Confidence gates first, then score. A high score on thin evidence
// yields allow — that is the cold-start answer, and it is the reason
// the two dimensions are separate fields rather than one number.
func enforced(j Judgement) string {
	switch {
	case j.confidence >= minConfidenceToBlock && j.score >= scoreToBlock:
		return DecisionBlock
	case j.confidence >= minConfidenceToChallenge && j.score >= scoreToChallenge:
		return DecisionChallenge
	default:
		return DecisionAllow
	}
}
