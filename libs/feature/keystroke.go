package feature

import "math"

const (
	// minIntervalsForRhythm is the fewest flight times that can carry a
	// rhythm. Below this, variance is noise.
	minIntervalsForRhythm = 6

	// maxPlausibleFlightMs discards pauses that are thinking rather than
	// typing. A 4-second gap between characters is a person looking
	// something up, and including it would make any typist look
	// maximally variable — which is the direction that hides bots.
	maxPlausibleFlightMs = 2000

	// minPlausibleDwellMs / maxPlausibleDwellMs bound key-hold times.
	minPlausibleDwellMs = 1
	maxPlausibleDwellMs = 1000
)

// Keystroke accumulates typing rhythm for one session.
//
// Timing only — no key content ever reaches this package, and the coarse
// class is used solely to separate modifier keys from ordinary ones.
//
// This exists because of a measured gap, not a guess. The M2 harness
// found that undetected-chromedriver evades pointer analysis entirely by
// jumping the cursor: two pointermove events across a whole session,
// confidence 0.000, allow. Typing is the evidence that survives when
// there is no pointer path — and it is also what makes a
// keyboard-navigating human judgeable rather than merely unjudged.
type Keystroke struct {
	// Flight times: up of one key to down of the next.
	flightCount uint32
	flightSum   float64
	flightSumSq float64

	// Dwell times: down to up on the same key.
	dwellCount uint32
	dwellSum   float64
	dwellSumSq float64

	// Perfectly uniform intervals are the strongest single tell: a human
	// cannot type two identical gaps in a row, let alone twenty.
	exactRepeats uint32
	lastFlightMs int64

	// Open key-down awaiting its up, per target+class.
	pendingDown map[string]uint32

	lastUpMs   uint32
	haveLastUp bool

	keyCount uint32
}

// KeystrokeState is the extracted typing feature vector.
type KeystrokeState struct {
	// FlightCV is the coefficient of variation of flight times
	// (stddev/mean). Human typing is irregular — CV typically well above
	// 0.3. Scripted typing with a fixed or narrowly-jittered delay sits
	// near zero.
	FlightCV float64

	// DwellCV is the same for key-hold durations. Many automation
	// libraries do not model dwell at all and emit down/up back to back.
	DwellCV float64

	// MeanDwellMs is the average key-hold time. Sub-millisecond dwell
	// means the key was never physically held.
	MeanDwellMs float64

	// ExactRepeatRatio is the fraction of consecutive flight times that
	// are byte-identical to their predecessor.
	ExactRepeatRatio float64

	// Intervals is the number of usable flight times.
	Intervals uint32

	// Keys is every key event seen, the honest evidence count.
	Keys uint32
}

// AddKey feeds one key event. phase is "down" or "up"; class is the
// coarse taxonomy; target is the hashed field identity.
func (k *Keystroke) AddKey(tMs uint32, phase, class, target string) {
	k.keyCount++

	if k.pendingDown == nil {
		k.pendingDown = make(map[string]uint32, 4)
	}
	// Modifiers are held across other keys and would distort both dwell
	// and flight if treated as ordinary presses.
	if class == "mod" {
		return
	}
	key := class + "\x00" + target

	switch phase {
	case "down":
		if k.haveLastUp && tMs >= k.lastUpMs {
			flight := float64(tMs - k.lastUpMs)
			if flight <= maxPlausibleFlightMs {
				if k.flightCount > 0 && int64(flight) == k.lastFlightMs {
					k.exactRepeats++
				}
				k.lastFlightMs = int64(flight)
				k.flightCount++
				k.flightSum += flight
				k.flightSumSq += flight * flight
			}
		}
		k.pendingDown[key] = tMs

	case "up":
		if downAt, ok := k.pendingDown[key]; ok && tMs >= downAt {
			dwell := float64(tMs - downAt)
			if dwell >= minPlausibleDwellMs && dwell <= maxPlausibleDwellMs {
				k.dwellCount++
				k.dwellSum += dwell
				k.dwellSumSq += dwell * dwell
			}
			delete(k.pendingDown, key)
		}
		k.lastUpMs = tMs
		k.haveLastUp = true
	}
}

// State returns the current typing feature vector.
func (k *Keystroke) State() KeystrokeState {
	st := KeystrokeState{Intervals: k.flightCount, Keys: k.keyCount}

	if k.flightCount >= minIntervalsForRhythm {
		st.FlightCV = cv(k.flightSum, k.flightSumSq, float64(k.flightCount))
		st.ExactRepeatRatio = float64(k.exactRepeats) / float64(k.flightCount)
	}
	if k.dwellCount >= minIntervalsForRhythm {
		st.DwellCV = cv(k.dwellSum, k.dwellSumSq, float64(k.dwellCount))
		st.MeanDwellMs = k.dwellSum / float64(k.dwellCount)
	}
	return st
}

// cv returns the coefficient of variation from running sums.
//
// Running sums rather than retained samples keep session state O(1) in
// session length, which is the property the whole architecture rests on.
func cv(sum, sumSq, n float64) float64 {
	if n < 2 {
		return 0
	}
	mean := sum / n
	if mean <= 0 {
		return 0
	}
	// Population variance; the sample correction is immaterial at these
	// counts and the population form cannot go negative from rounding.
	variance := sumSq/n - mean*mean
	if variance < 0 {
		variance = 0
	}
	return math.Sqrt(variance) / mean
}
