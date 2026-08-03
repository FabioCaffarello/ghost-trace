package feature

import "testing"

// typeWith feeds n keystrokes with the given flight and dwell times.
func typeWith(k *Keystroke, n int, flightMs, dwellMs uint32, jitter func(i int) (uint32, uint32)) {
	t := uint32(1000)
	for i := 0; i < n; i++ {
		f, d := flightMs, dwellMs
		if jitter != nil {
			f, d = jitter(i)
		}
		t += f
		k.AddKey(t, "down", "alpha", "f_1")
		t += d
		k.AddKey(t, "up", "alpha", "f_1")
	}
}

func TestScriptedTypingHasNearZeroVariance(t *testing.T) {
	// A fixed inter-key delay is what every automation library does by
	// default, and it is not something a person can produce.
	var k Keystroke
	typeWith(&k, 30, 60, 40, nil)

	st := k.State()
	if st.FlightCV > 0.02 {
		t.Errorf("FlightCV = %.4f, want ~0 for fixed-delay typing", st.FlightCV)
	}
	if st.ExactRepeatRatio < 0.9 {
		t.Errorf("ExactRepeatRatio = %.3f, want ~1 for identical gaps", st.ExactRepeatRatio)
	}
}

func TestHumanLikeTypingHasHighVariance(t *testing.T) {
	// Irregular timing, as a person produces.
	var k Keystroke
	flights := []uint32{90, 140, 70, 210, 110, 60, 180, 95, 260, 75,
		130, 88, 175, 62, 240, 105, 150, 80, 200, 118,
		92, 165, 71, 225, 99, 143, 187, 66, 210, 128}
	dwells := []uint32{70, 95, 60, 110, 85, 55, 100, 75, 120, 65,
		90, 72, 105, 58, 115, 80, 98, 63, 108, 77,
		68, 102, 59, 118, 74, 91, 112, 61, 106, 83}
	typeWith(&k, 30, 0, 0, func(i int) (uint32, uint32) {
		return flights[i], dwells[i]
	})

	st := k.State()
	if st.FlightCV < 0.30 {
		t.Errorf("FlightCV = %.3f, want > 0.30 for irregular typing", st.FlightCV)
	}
	if st.ExactRepeatRatio > 0.1 {
		t.Errorf("ExactRepeatRatio = %.3f, want ~0", st.ExactRepeatRatio)
	}
	if st.MeanDwellMs < 50 {
		t.Errorf("MeanDwellMs = %.1f, want a realistic hold time", st.MeanDwellMs)
	}
}

func TestZeroDwellIsDetected(t *testing.T) {
	// Many automation libraries emit keydown and keyup back to back, so
	// the key is never actually held.
	var k Keystroke
	typeWith(&k, 20, 55, 0, nil)

	if st := k.State(); st.MeanDwellMs != 0 {
		// Dwell of 0 is below minPlausibleDwellMs and so is excluded
		// from the average entirely — the state reports no dwell data
		// rather than a fabricated zero.
		t.Errorf("MeanDwellMs = %.2f; sub-threshold dwells must not be averaged in", st.MeanDwellMs)
	}
}

func TestModifiersDoNotDistortRhythm(t *testing.T) {
	// Shift is held across the key it modifies. Treating it as an
	// ordinary press would inflate dwell and corrupt flight.
	var k Keystroke
	t0 := uint32(1000)
	for i := 0; i < 12; i++ {
		k.AddKey(t0, "down", "mod", "f_1")
		t0 += 5
		k.AddKey(t0, "down", "alpha", "f_1")
		t0 += 70
		k.AddKey(t0, "up", "alpha", "f_1")
		t0 += 3
		k.AddKey(t0, "up", "mod", "f_1")
		t0 += 90
	}

	st := k.State()
	if st.MeanDwellMs < 60 || st.MeanDwellMs > 80 {
		t.Errorf("MeanDwellMs = %.1f, want ~70 (the alpha hold, not the modifier)", st.MeanDwellMs)
	}
}

func TestThinkingPausesAreExcluded(t *testing.T) {
	// A four-second gap is a person looking something up, not typing.
	// Including it would make any typist look maximally variable, which
	// is the direction that hides bots.
	var k Keystroke
	typeWith(&k, 20, 60, 40, nil)
	before := k.State().FlightCV

	t0 := uint32(500000)
	k.AddKey(t0, "down", "alpha", "f_1")
	k.AddKey(t0+40, "up", "alpha", "f_1")

	if after := k.State().FlightCV; after > before+0.01 {
		t.Errorf("FlightCV moved %.4f -> %.4f; long pauses must be excluded", before, after)
	}
}

func TestKeysCountsEverythingSeen(t *testing.T) {
	var k Keystroke
	typeWith(&k, 10, 60, 40, nil)
	if st := k.State(); st.Keys != 20 {
		t.Errorf("Keys = %d, want 20 (10 down + 10 up)", st.Keys)
	}
}

func TestTooFewIntervalsReportNoRhythm(t *testing.T) {
	var k Keystroke
	typeWith(&k, 3, 60, 40, nil)
	if st := k.State(); st.FlightCV != 0 {
		t.Errorf("FlightCV = %v; below minIntervalsForRhythm it must report 0", st.FlightCV)
	}
}
