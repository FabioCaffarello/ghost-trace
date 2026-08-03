package feature

import (
	"math"
	"testing"
)

// line builds a perfectly straight, constant-velocity polyline — the
// tier-4 synthetic adversary in its simplest form.
func line(n int, stepX, stepY int32, dtMs uint32) []Point {
	pts := make([]Point, 0, n)
	for i := 0; i < n; i++ {
		dt := dtMs
		if i == 0 {
			dt = 0
		}
		pts = append(pts, Point{X: int32(i) * stepX, Y: int32(i) * stepY, DtMs: dt})
	}
	return pts
}

func TestStraightLineScoresOne(t *testing.T) {
	var p Pointer
	p.Add(0, line(40, 10, 0, 16))

	st := p.State()
	if st.Segments != 1 {
		t.Fatalf("Segments = %d, want 1", st.Segments)
	}
	if math.Abs(st.Straightness-1.0) > 1e-6 {
		t.Errorf("Straightness = %v, want 1.0", st.Straightness)
	}
}

func TestZigzagScoresWellBelowOne(t *testing.T) {
	// A path that doubles back covers far more distance than it
	// displaces, which is what human correction looks like.
	var p Pointer
	pts := []Point{{X: 0, Y: 0}}
	for i := 1; i <= 40; i++ {
		y := int32(0)
		if i%2 == 0 {
			y = 60
		}
		pts = append(pts, Point{X: int32(i) * 4, Y: y, DtMs: 16})
	}
	p.Add(0, pts)

	st := p.State()
	if st.Straightness > 0.5 {
		t.Errorf("Straightness = %v, want well below 0.5 for a zigzag", st.Straightness)
	}
}

func TestGapSplitsSegments(t *testing.T) {
	// Whole-session straightness is meaningless; a pause must start a
	// new segment or two unrelated moves get averaged into one line.
	var p Pointer
	p.Add(0, line(20, 10, 0, 16))

	// Second polyline starts long after the first ended. The gap is
	// carried by absolute start time, not by the point's own dt —
	// a polyline boundary alone means nothing.
	p.Add(5000, line(20, 0, 10, 16))

	if st := p.State(); st.Segments != 2 {
		t.Errorf("Segments = %d, want 2 across a 5s gap", st.Segments)
	}
}

func TestShortMovementsDoNotQualify(t *testing.T) {
	// A 12px twitch is perfectly straight and means nothing. Admitting
	// it would inflate the score for every human.
	var p Pointer
	p.Add(0, line(6, 2, 0, 16)) // 10px total path, under minSegmentPathPx

	st := p.State()
	if st.Segments != 0 {
		t.Errorf("Segments = %d, want 0 for a sub-threshold move", st.Segments)
	}
	if st.Straightness != 0 {
		t.Errorf("Straightness = %v, want 0 with no qualifying segment", st.Straightness)
	}
}

func TestTooFewPointsDoNotQualify(t *testing.T) {
	var p Pointer
	p.Add(0, []Point{{X: 0, Y: 0}, {X: 400, Y: 0, DtMs: 16}})

	if st := p.State(); st.Segments != 0 {
		t.Errorf("Segments = %d, want 0 with fewer than %d points", st.Segments, minSegmentPoints)
	}
}

func TestPointsCountsEverythingSeen(t *testing.T) {
	// Points is the honest evidence count: it reports what arrived, not
	// what qualified, so a session of nothing but twitches still shows
	// its volume.
	var p Pointer
	p.Add(0, line(6, 2, 0, 16))

	if st := p.State(); st.Points != 6 {
		t.Errorf("Points = %d, want 6 even though no segment qualified", st.Points)
	}
}

func TestOpenSegmentIsIncluded(t *testing.T) {
	// A decision can arrive mid-motion. Excluding movement in flight
	// would under-report evidence exactly when it is being asked for.
	var p Pointer
	p.Add(0, line(40, 10, 0, 16)) // never closed by a gap

	if st := p.State(); st.Segments != 1 || st.PathPx <= 0 {
		t.Errorf("open segment excluded: Segments=%d PathPx=%v", st.Segments, st.PathPx)
	}
}

func TestStraightnessNeverExceedsOne(t *testing.T) {
	// Float accumulation over a long path can drift; a value above 1.0
	// would silently poison the length-weighted mean.
	var p Pointer
	for i := 0; i < 50; i++ {
		p.Add(uint32(i)*10000, line(200, 7, 3, 16))
	}
	if st := p.State(); st.Straightness > 1.0 {
		t.Errorf("Straightness = %v, must never exceed 1.0", st.Straightness)
	}
}

func TestLongerSegmentDominatesWeighting(t *testing.T) {
	// A long deliberate drag is more informative than a short flick, so
	// the mean is weighted by path length.
	var p Pointer

	// Short, straight.
	p.Add(0, line(8, 8, 0, 16))

	// Long and curved, starting after a pause. A smooth arc rather than
	// a zigzag: consecutive direction changes stay well under the turn
	// threshold, so it remains one segment — which is what a human
	// reach looks like.
	arc := []Point{{X: 0, Y: 0}}
	for i := 1; i <= 200; i++ {
		t := float64(i) / 200
		x := int32(1200 * t)
		y := int32(500 * math.Sin(t*math.Pi))
		arc = append(arc, Point{X: x, Y: y, DtMs: 16})
	}
	p.Add(5000, arc)

	st := p.State()
	if st.Segments != 2 {
		t.Fatalf("Segments = %d, want 2", st.Segments)
	}
	// The short leg is perfectly straight (1.0) and the long arc is
	// around 0.75. An unweighted mean would land near 0.88; weighting by
	// path length must pull the result down toward the long segment.
	if st.Straightness > 0.80 {
		t.Errorf("Straightness = %.3f; the long curved segment should dominate", st.Straightness)
	}
}

func TestSharpTurnSplitsSegmentsWithoutAPause(t *testing.T) {
	// The M2 finding, as a regression test. Three straight legs to
	// different targets, back-to-back with no pause — a bot filling a
	// form. Before turn detection these merged into one path whose
	// direction changes read as human correction (straightness 0.315)
	// and the bot evaded completely.
	var p Pointer
	legs := [][2][2]int32{
		{{100, 120}, {620, 300}},
		{{620, 300}, {660, 380}},
		{{660, 380}, {300, 520}},
	}
	pts := []Point{}
	for _, leg := range legs {
		a, b := leg[0], leg[1]
		for j := 0; j < 12; j++ {
			f := float64(j) / 11
			dt := uint32(50)
			if len(pts) == 0 {
				dt = 0
			}
			pts = append(pts, Point{
				X:    a[0] + int32(float64(b[0]-a[0])*f),
				Y:    a[1] + int32(float64(b[1]-a[1])*f),
				DtMs: dt,
			})
		}
	}
	p.Add(0, pts)

	st := p.State()
	if st.Segments < 2 {
		t.Fatalf("Segments = %d; sharp turns must split without a time gap", st.Segments)
	}
	if st.Straightness < 0.9 {
		t.Errorf("Straightness = %.3f; straight legs must survive segmentation", st.Straightness)
	}
}

func TestSmoothCurveStaysOneSegment(t *testing.T) {
	// The other side of the same threshold: a human reach curves
	// continuously and must not shatter into fragments, which would
	// drop the evidence below the qualifying thresholds and hand every
	// human a free pass.
	var p Pointer
	pts := []Point{{X: 0, Y: 0}}
	for i := 1; i <= 60; i++ {
		f := float64(i) / 60
		pts = append(pts, Point{
			X:    int32(700 * f),
			Y:    int32(260 * math.Sin(f*math.Pi/2)),
			DtMs: 50,
		})
	}
	p.Add(0, pts)

	if st := p.State(); st.Segments != 1 {
		t.Errorf("Segments = %d, want 1 for a smooth curve", st.Segments)
	}
}
