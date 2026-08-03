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

	// Long, jagged, starting after a pause.
	jag := []Point{{X: 0, Y: 0}}
	for i := 1; i <= 300; i++ {
		y := int32(0)
		if i%2 == 0 {
			y = 50
		}
		jag = append(jag, Point{X: int32(i) * 4, Y: y, DtMs: 16})
	}
	p.Add(5000, jag)

	st := p.State()
	if st.Segments != 2 {
		t.Fatalf("Segments = %d, want 2", st.Segments)
	}
	if st.Straightness > 0.75 {
		t.Errorf("Straightness = %v; the long jagged segment should dominate", st.Straightness)
	}
}
