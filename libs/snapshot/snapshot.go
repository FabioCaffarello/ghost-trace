// Package snapshot converts between the in-memory state a decision is
// computed from and the SessionSnapshot that carries it across a
// process boundary.
//
// Both directions live here, next to each other, deliberately. The
// collector writes snapshots and the decision engine reads them, and
// the one property that matters is that a judgement made from a
// snapshot equals the judgement the collector would have made in
// process. Two mappings written apart would satisfy their own tests and
// disagree with each other; written together, the round trip is one
// assertion away.
//
// It is NOT lossless, and the loss is documented rather than hidden:
// FeatureState stores the ratios as float32 while the domain computes
// them in float64. See LossOf and the round-trip test.
package snapshot

import (
	"github.com/FabioCaffarello/ghost-trace/libs/feature"
	eventsv1 "github.com/FabioCaffarello/ghost-trace/libs/genproto/events/v1"
	"github.com/FabioCaffarello/ghost-trace/libs/policy"
)

// FromState builds the wire feature state from a policy state.
func FromState(st policy.State) *eventsv1.FeatureState {
	return &eventsv1.FeatureState{
		PointerStraightness:     float32(st.Pointer.Straightness),
		PointerSegments:         st.Pointer.Segments,
		PointerPathPx:           float32(st.Pointer.PathPx),
		PointerPoints:           st.Pointer.Points,
		KeyFlightCv:             float32(st.Keystroke.FlightCV),
		KeyDwellCv:              float32(st.Keystroke.DwellCV),
		KeyMeanDwellMs:          float32(st.Keystroke.MeanDwellMs),
		KeyExactRepeatRatio:     float32(st.Keystroke.ExactRepeatRatio),
		KeyIntervals:            st.Keystroke.Intervals,
		ProgrammaticScrollRatio: float32(st.Interaction.ProgrammaticScrollRatio),
		ScrollEvents:            st.Interaction.ScrollEvents,
		FocusTransitions:        st.Interaction.FocusTransitions,
		HiddenPeriods:           st.Interaction.HiddenPeriods,
		Pastes:                  st.Interaction.Pastes,
		Injections:              st.Interaction.Injections,
		InjectedFields:          st.Interaction.InjectedFields,
		Autofills:               st.Interaction.Autofills,
		DistinctFocusTargets:    st.Interaction.DistinctFocusTargets,
		KeyEvents:               st.Keystroke.Keys,
	}
}

// ToState rebuilds the policy state from the wire feature state.
//
// The inverse of FromState, field for field. There is deliberately no
// defaulting or inference here: a field the producer did not set
// arrives as zero, and zero is what the judgement then sees. Guessing
// at a missing value would make a snapshot judge differently from the
// session it describes.
func ToState(fs *eventsv1.FeatureState) policy.State {
	if fs == nil {
		return policy.State{}
	}
	return policy.State{
		Pointer: feature.PointerState{
			Straightness: float64(fs.GetPointerStraightness()),
			Segments:     fs.GetPointerSegments(),
			PathPx:       float64(fs.GetPointerPathPx()),
			Points:       fs.GetPointerPoints(),
		},
		Keystroke: feature.KeystrokeState{
			FlightCV:         float64(fs.GetKeyFlightCv()),
			DwellCV:          float64(fs.GetKeyDwellCv()),
			MeanDwellMs:      float64(fs.GetKeyMeanDwellMs()),
			ExactRepeatRatio: float64(fs.GetKeyExactRepeatRatio()),
			Intervals:        fs.GetKeyIntervals(),
			Keys:             fs.GetKeyEvents(),
		},
		Interaction: feature.InteractionState{
			ProgrammaticScrollRatio: float64(fs.GetProgrammaticScrollRatio()),
			ScrollEvents:            fs.GetScrollEvents(),
			FocusTransitions:        fs.GetFocusTransitions(),
			HiddenPeriods:           fs.GetHiddenPeriods(),
			Pastes:                  fs.GetPastes(),
			Injections:              fs.GetInjections(),
			InjectedFields:          fs.GetInjectedFields(),
			Autofills:               fs.GetAutofills(),
			DistinctFocusTargets:    fs.GetDistinctFocusTargets(),
		},
	}
}

// RoundTrip is FromState followed by ToState — the state a decision
// engine would actually judge, given what the collector would actually
// write. Exposed because the difference between it and the input is
// the cost of the split, and a cost nobody can measure is a cost nobody
// will notice.
func RoundTrip(st policy.State) policy.State { return ToState(FromState(st)) }
