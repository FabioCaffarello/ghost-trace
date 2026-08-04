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
// constructed without both numbers.
//
// M3 makes scoring per-action, per contract §8.6. Feature state is
// per-session — there is one pointer trace and one typing rhythm — but
// which evidence is RELEVANT depends on what the application is about to
// do. A login involves typing; a checkout may involve none. Collapsing
// that into one session-level number produces the worst available
// failure: a confident-looking block on evidence that does not bear on
// the action taken.
package policy

import (
	"fmt"
	"sort"

	"github.com/FabioCaffarello/ghost-trace/services/ingestion/internal/feature"
)

// Reason codes are a stable enumeration (contract §7). Adding a code is
// non-breaking; changing what an existing code means is not.
const (
	// ReasonPointerLinearity: the pointer path is straighter than human
	// motor control usually produces.
	ReasonPointerLinearity = "POINTER_LINEARITY"

	// ReasonKeyIntervalVarianceLow: typing rhythm is too regular. Human
	// inter-key timing is irregular; scripted typing is not.
	ReasonKeyIntervalVarianceLow = "KEY_INTERVAL_VARIANCE_LOW"

	// ReasonKeyDwellAbsent: keys register no measurable hold time, so
	// they were never physically pressed.
	ReasonKeyDwellAbsent = "KEY_DWELL_ABSENT"

	// ReasonKeyIntervalsIdentical: consecutive inter-key gaps are
	// byte-identical. A person cannot do this.
	ReasonKeyIntervalsIdentical = "KEY_INTERVALS_IDENTICAL"

	// ReasonProgrammaticScroll: the page scrolled itself.
	ReasonProgrammaticScroll = "PROGRAMMATIC_SCROLL"

	// ReasonNoFocusTransitions: a form was completed without any field
	// ever receiving focus.
	ReasonNoFocusTransitions = "NO_FOCUS_TRANSITIONS"

	// ReasonValueInjected: field contents changed with no keystroke and
	// no paste behind them.
	//
	// KNOWN FALSE-POSITIVE POPULATION. "A person cannot do this" is very
	// nearly true and not quite: speech-to-text dictation, IME
	// composition (Chinese, Japanese, Korean), and some assistive input
	// devices produce input events without a preceding keydown, and will
	// look like injection under this rule.
	//
	// That is exactly the population the whole project says it is most
	// worried about mis-flagging, so this signal must not be promoted to
	// categorical without dictation and IME sessions in the capture
	// study. Recorded here rather than in a backlog because the code is
	// where someone will read it.
	ReasonValueInjected = "VALUE_INJECTED"

	// ReasonInsufficientEvidence: too little relevant evidence to
	// support any judgement, including when the score is high. It is the
	// reason a bot-looking session was still allowed.
	ReasonInsufficientEvidence = "INSUFFICIENT_EVIDENCE"
)

// ReasonCodes is the §7 enumeration as DATA.
//
// The contract has promised a stable set of reason codes since it was
// written, and never listed one — because the codes were constants,
// and a constant is something no generator and no client can
// enumerate. Publishing the set is what lets the OpenAPI schema say
// which codes exist instead of `type: string`.
//
// Order is the order above: rough order of evidential strength, which
// is how a reader should meet them.
//
// reason_codes_test.go parses this file and asserts that this slice
// and the constants are the same set, in both directions. There is
// deliberately no exception list.
// Decisions and Modes, as data, for the same reason ReasonCodes is:
// a constant is something no generator and no client can enumerate,
// and every enumeration retyped into a schema by hand in R1.14 that
// was NOT read from a Go value turned out to be wrong.
var (
	Decisions = []string{DecisionAllow, DecisionChallenge, DecisionBlock}
	Modes     = []string{ModeMonitor, ModeEnforce}
)

var ReasonCodes = []string{
	ReasonPointerLinearity,
	ReasonKeyIntervalVarianceLow,
	ReasonKeyDwellAbsent,
	ReasonKeyIntervalsIdentical,
	ReasonProgrammaticScroll,
	ReasonNoFocusTransitions,
	ReasonValueInjected,
	ReasonInsufficientEvidence,
}

// Decisions.
const (
	DecisionAllow     = "allow"
	DecisionChallenge = "challenge"
	DecisionBlock     = "block"
)

// Modes (contract §4).
const (
	// ModeMonitor always returns allow and reports what enforce would
	// have done. Every integration starts here.
	ModeMonitor = "monitor"
	ModeEnforce = "enforce"
)

// The scoring numbers live in Calibration (calibration.go), loaded
// from the embedded calibration.json or a -policy file. The formulas
// below are the scoring DEFINITION and stay in code; the numbers they
// consume are configuration, because the calibration loop must not
// require a recompile per iteration.

// State is the full per-session feature vector handed to the policy.
type State struct {
	Pointer     feature.PointerState
	Keystroke   feature.KeystrokeState
	Interaction feature.InteractionState
}

// Weights describe how much each evidence channel matters for one action.
//
// This is the per-action half of §8.6.
type Weights struct {
	Pointer     float64
	Keystroke   float64
	Interaction float64
}

// actionWeights maps an action to its evidence profile. An action nobody
// has characterised falls back to a balanced profile rather than a guess.
var actionWeights = map[string]Weights{
	"login":    {Pointer: 0.35, Keystroke: 0.50, Interaction: 0.15},
	"signup":   {Pointer: 0.30, Keystroke: 0.55, Interaction: 0.15},
	"checkout": {Pointer: 0.55, Keystroke: 0.20, Interaction: 0.25},
	"comment":  {Pointer: 0.25, Keystroke: 0.60, Interaction: 0.15},
}

var defaultWeights = Weights{Pointer: 0.40, Keystroke: 0.40, Interaction: 0.20}

// WeightsFor returns the evidence profile for an action.
func WeightsFor(action string) Weights {
	if w, ok := actionWeights[action]; ok {
		return w
	}
	return defaultWeights
}

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

	// contributing counts evidence channels that held any data. Blocks
	// require more than one; see enforced.
	contributing int
}

// Reason is one contributing factor.
type Reason struct {
	Code   string  `json:"code"`
	Weight float64 `json:"weight"`
}

// Score is how bot-like the behaviour looks, in [0, 1].
func (j Judgement) Score() float64 { return j.score }

// Confidence is how much relevant evidence supports the score, in [0, 1].
func (j Judgement) Confidence() float64 { return j.confidence }

// Reasons returns the contributing factors, most significant first.
func (j Judgement) Reasons() []Reason { return j.reasons }

// channel is one evidence stream's contribution.
type channel struct {
	score      float64 // how bot-like, [0,1]
	confidence float64 // how much evidence, [0,1]
	weight     float64 // this action's weight for the channel
	reasons    []Reason
}

// Judge maps a feature vector and an action to a judgement.
func Judge(st State, action string) Judgement {
	w := WeightsFor(action)

	channels := []channel{
		pointerChannel(st.Pointer, w.Pointer),
		keystrokeChannel(st.Keystroke, w.Keystroke),
		interactionChannel(st.Interaction, w.Interaction),
	}

	// The score is weighted by weight × CONFIDENCE, not by the action
	// weight alone. A channel holding no evidence must not drag the
	// score toward zero — that is precisely how a bot with no pointer
	// data looks innocent, which is the tier-3 evasion M2 measured.
	// Weighting by evidence makes silent channels abstain rather than
	// vote.
	var scoreNum, scoreDen, weighted, maxWeight float64
	var active int
	var reasons []Reason
	for _, c := range channels {
		if c.weight > maxWeight {
			maxWeight = c.weight
		}
		if c.confidence > 0 {
			active++
		}
		effective := c.weight * c.confidence
		scoreNum += c.score * effective
		scoreDen += effective
		weighted += effective
		reasons = append(reasons, c.reasons...)
	}

	j := Judgement{contributing: active}
	if scoreDen > 0 {
		j.score = clamp01(scoreNum / scoreDen)
	}

	// Confidence is normalised by the single most relevant channel, not
	// by the sum of all weights.
	//
	// Summing to 1.0 was the obvious formulation and it was wrong: it
	// caps single-channel confidence at that channel's own weight, so a
	// session with a perfect linear pointer path and no typing could
	// never clear the 0.40 floor for `login` — by construction, not by
	// evidence. The M3 harness caught it as tier 1 and tier 4 escaping
	// after having been detected 100% of the time.
	//
	// Normalising by maxWeight says: confidence reaches 1.0 when you
	// hold as much weighted evidence as the action's dominant channel
	// could supply on its own. Missing channels still cost confidence;
	// they no longer make a verdict unreachable.
	if maxWeight > 0 {
		j.confidence = clamp01(weighted / maxWeight)
	}

	if j.confidence < cal.MinConfidenceToChallenge {
		reasons = append(reasons, Reason{
			Code:   ReasonInsufficientEvidence,
			Weight: 1 - j.confidence,
		})
	}

	sort.SliceStable(reasons, func(a, b int) bool {
		return reasons[a].Weight > reasons[b].Weight
	})
	j.reasons = reasons
	return j
}

func pointerChannel(st feature.PointerState, weight float64) channel {
	c := channel{weight: weight}
	if st.Segments == 0 {
		return c
	}
	c.score = ramp(st.Straightness, cal.StraightnessFloor, cal.StraightnessCeil)
	c.confidence = clamp01(0.5*min1(float64(st.Segments)/cal.ConfidenceSegments) +
		0.5*min1(st.PathPx/cal.ConfidencePathPx))
	if c.score > 0 {
		c.reasons = append(c.reasons, Reason{ReasonPointerLinearity, c.score})
	}
	return c
}

func keystrokeChannel(st feature.KeystrokeState, weight float64) channel {
	c := channel{weight: weight}
	if st.Intervals == 0 {
		return c
	}
	c.confidence = min1(float64(st.Intervals) / cal.ConfidenceIntervals)

	rhythm := ramp(st.FlightCV, cal.FlightCVFloor, cal.FlightCVCeil)
	if rhythm > 0 {
		c.reasons = append(c.reasons, Reason{ReasonKeyIntervalVarianceLow, rhythm})
	}

	dwell := 0.0
	if st.MeanDwellMs > 0 && st.MeanDwellMs < cal.DwellAbsentMs {
		dwell = 1 - st.MeanDwellMs/cal.DwellAbsentMs
		c.reasons = append(c.reasons, Reason{ReasonKeyDwellAbsent, dwell})
	}

	// Byte-identical consecutive gaps. A person cannot produce these, so
	// this saturates fast.
	identical := 0.0
	if st.ExactRepeatRatio > 0 {
		identical = min1(st.ExactRepeatRatio / cal.IdenticalIntervalRatio)
		c.reasons = append(c.reasons, Reason{ReasonKeyIntervalsIdentical, identical})
	}

	c.score = maxOf(rhythm, dwell, identical)
	return c
}

func interactionChannel(st feature.InteractionState, weight float64) channel {
	c := channel{weight: weight}

	// No structural evidence at all — abstain rather than accuse.
	if st.FocusTransitions == 0 && st.ScrollEvents == 0 && st.Pastes == 0 &&
		st.Injections == 0 {
		return c
	}

	// Value injection. Close to categorical, like programmatic scroll:
	// there is no way for a person to change a field's contents without
	// typing, pasting, or autofill. One injected field is odd; a whole
	// form is a script.
	injected := 0.0
	if st.Injections > 0 {
		injected = min1(0.55 + 0.25*float64(st.InjectedFields))
		c.reasons = append(c.reasons, Reason{ReasonValueInjected, injected})
		c.confidence = 1
	}

	// Programmatic scrolling is close to categorical: a person cannot
	// produce one.
	prog := 0.0
	if st.ProgrammaticScrollRatio > 0 {
		prog = min1(st.ProgrammaticScrollRatio * 2)
		c.reasons = append(c.reasons, Reason{ReasonProgrammaticScroll, prog})
		c.confidence = 1
	}

	noFocus := 0.0
	if st.FocusTransitions == 0 {
		noFocus = 0.6
		c.reasons = append(c.reasons, Reason{ReasonNoFocusTransitions, noFocus})
	}

	if c.confidence == 0 {
		c.confidence = clamp01(float64(st.FocusTransitions)/6 + float64(st.ScrollEvents)/20)
	}
	c.score = maxOf(prog, noFocus, injected)
	return c
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
// yields allow — that is the cold-start answer, and the reason the two
// dimensions are separate fields rather than one number.
func enforced(j Judgement) string {
	switch {
	case j.confidence >= cal.MinConfidenceToBlock && j.score >= cal.ScoreToBlock &&
		j.contributing >= cal.MinChannelsToBlock:
		return DecisionBlock
	case j.confidence >= cal.MinConfidenceToChallenge && j.score >= cal.ScoreToChallenge:
		return DecisionChallenge
	default:
		return DecisionAllow
	}
}

// ramp maps v from `zero` to `one` onto [0,1]. Works in either
// direction, so a feature where LOW values are suspicious passes
// zero > one.
func ramp(v, zero, one float64) float64 {
	if zero == one {
		return 0
	}
	return clamp01((v - zero) / (one - zero))
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func min1(v float64) float64 { return clamp01(v) }

func maxOf(vs ...float64) float64 {
	m := 0.0
	for _, v := range vs {
		if v > m {
			m = v
		}
	}
	return m
}
