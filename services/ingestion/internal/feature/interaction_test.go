package feature

import "testing"

func TestInteractionZeroValue(t *testing.T) {
	var in Interaction
	st := in.State()
	if st != (InteractionState{}) {
		t.Errorf("zero-value State = %+v, want all zeros", st)
	}
}

func TestScrollProgrammaticRatio(t *testing.T) {
	var in Interaction
	in.AddScroll("gesture")
	in.AddScroll("gesture")
	in.AddScroll("programmatic")
	in.AddScroll("wheel")

	st := in.State()
	if st.ScrollEvents != 4 {
		t.Errorf("ScrollEvents = %d, want 4", st.ScrollEvents)
	}
	if st.ProgrammaticScrollRatio != 0.25 {
		t.Errorf("ProgrammaticScrollRatio = %v, want 0.25", st.ProgrammaticScrollRatio)
	}
}

func TestFocusDistinctTargets(t *testing.T) {
	var in Interaction
	in.AddFocus("f_user")
	in.AddFocus("f_pass")
	in.AddFocus("f_user") // revisit: transition counts, target does not
	in.AddFocus("")       // blur without target: transition only

	st := in.State()
	if st.FocusTransitions != 4 {
		t.Errorf("FocusTransitions = %d, want 4", st.FocusTransitions)
	}
	if st.DistinctFocusTargets != 2 {
		t.Errorf("DistinctFocusTargets = %d, want 2", st.DistinctFocusTargets)
	}
}

func TestVisibilityHiddenPeriods(t *testing.T) {
	var in Interaction
	in.AddVisibility("hidden")
	in.AddVisibility("visible")
	in.AddVisibility("hidden")

	if st := in.State(); st.HiddenPeriods != 2 {
		t.Errorf("HiddenPeriods = %d, want 2", st.HiddenPeriods)
	}
}

func TestFormActions(t *testing.T) {
	var in Interaction
	in.AddForm("paste", "f_user")
	in.AddForm("paste", "f_pass")
	in.AddForm("autofill", "f_user")
	in.AddForm("submit", "")
	in.AddForm("unknown-action", "f_user") // dropped, never counted

	st := in.State()
	if st.Pastes != 2 {
		t.Errorf("Pastes = %d, want 2", st.Pastes)
	}
	if st.Autofills != 1 {
		t.Errorf("Autofills = %d, want 1", st.Autofills)
	}
	if st.Injections != 0 {
		t.Errorf("Injections = %d, want 0", st.Injections)
	}
}

// The value-injection channel is the strongest signal in the system —
// policy weighs it at confidence 1 — and until this file existed it was
// the only channel with no test at all.
func TestFormInjections(t *testing.T) {
	var in Interaction
	in.AddForm("injected", "f_user")
	in.AddForm("injected", "f_user") // same field again: count, not cardinality
	in.AddForm("injected", "f_pass")
	in.AddForm("injected", "") // no field identity: event still counts

	st := in.State()
	if st.Injections != 4 {
		t.Errorf("Injections = %d, want 4", st.Injections)
	}
	if st.InjectedFields != 2 {
		t.Errorf("InjectedFields = %d, want 2", st.InjectedFields)
	}
}
