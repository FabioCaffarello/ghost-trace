package feature

// Interaction accumulates the coarse structural evidence: scrolling,
// focus transitions, page visibility and form actions.
//
// Individually these are weak. Two of them are not:
//
//   - A programmatic scroll cannot be produced by a person. It is the
//     only signal in the system that is close to categorical rather
//     than a matter of degree.
//   - A form submitted with zero focus transitions was not filled by
//     someone tabbing or clicking through it.
//
// Both are cheap to collect and neither depends on the pointer, which
// is what makes them part of the answer to the tier-3 blind spot.
type Interaction struct {
	scrollEvents       uint32
	programmaticScroll uint32

	focusEvents  uint32
	focusTargets map[string]struct{}

	visibilityChanges uint32
	hiddenPeriods     uint32

	pastes    uint32
	autofills uint32
	submits   uint32

	// injections are field-content changes with no keystroke and no
	// paste behind them.
	injections     uint32
	injectedFields map[string]struct{}
}

// InteractionState is the extracted structural feature vector.
type InteractionState struct {
	// ProgrammaticScrollRatio is the fraction of scroll events the page
	// generated rather than the user. Above zero is already suspicious.
	ProgrammaticScrollRatio float64

	// ScrollEvents is the total scroll count.
	ScrollEvents uint32

	// FocusTransitions counts focus/blur events.
	FocusTransitions uint32

	// DistinctFocusTargets counts the fields that received focus.
	DistinctFocusTargets uint32

	// HiddenPeriods counts how often the page went to the background —
	// a strong human signal, since automation rarely tab-switches.
	HiddenPeriods uint32

	// Pastes and Autofills into fields. Neither is bot-specific — people
	// paste passwords constantly — so they inform rather than accuse.
	Pastes    uint32
	Autofills uint32

	// Injections: field contents that changed with no keystroke and no
	// paste behind them. Unlike the two above, this one IS bot-specific:
	// a person cannot change a field's contents without typing into it,
	// pasting into it, or the browser autofilling it.
	Injections uint32

	// InjectedFields is the count of distinct fields affected. One is
	// odd; every field in a form is a script.
	InjectedFields uint32
}

// AddScroll feeds one scroll event.
func (in *Interaction) AddScroll(mode string) {
	in.scrollEvents++
	if mode == "programmatic" {
		in.programmaticScroll++
	}
}

// AddFocus feeds one focus or blur event.
func (in *Interaction) AddFocus(target string) {
	in.focusEvents++
	if in.focusTargets == nil {
		in.focusTargets = make(map[string]struct{}, 4)
	}
	if target != "" {
		in.focusTargets[target] = struct{}{}
	}
}

// AddVisibility feeds one visibility change.
func (in *Interaction) AddVisibility(state string) {
	in.visibilityChanges++
	if state == "hidden" {
		in.hiddenPeriods++
	}
}

// AddForm feeds one form action against a field identity.
func (in *Interaction) AddForm(action, target string) {
	switch action {
	case "paste":
		in.pastes++
	case "autofill":
		in.autofills++
	case "submit":
		in.submits++
	case "injected":
		in.injections++
		if in.injectedFields == nil {
			in.injectedFields = make(map[string]struct{}, 4)
		}
		if target != "" {
			in.injectedFields[target] = struct{}{}
		}
	}
}

// State returns the current structural feature vector.
func (in *Interaction) State() InteractionState {
	st := InteractionState{
		ScrollEvents:         in.scrollEvents,
		FocusTransitions:     in.focusEvents,
		DistinctFocusTargets: uint32(len(in.focusTargets)),
		HiddenPeriods:        in.hiddenPeriods,
		Pastes:               in.pastes,
		Autofills:            in.autofills,
		Injections:           in.injections,
		InjectedFields:       uint32(len(in.injectedFields)),
	}
	if in.scrollEvents > 0 {
		st.ProgrammaticScrollRatio = float64(in.programmaticScroll) / float64(in.scrollEvents)
	}
	return st
}
