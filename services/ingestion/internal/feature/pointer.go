// Package feature computes Category II values: deterministic functions
// of raw events under a versioned definition.
//
// These are facts about what was computed, not judgements about what it
// means. Turning a feature into a score is the policy layer's job, and
// keeping the two apart is what lets a recalibrated model reinterpret an
// old evaluation record instead of silently rewriting history.
//
// M1 shipped exactly one feature; M2 corrected its segmentation.
// Still one feature. That is deliberate: a second feature
// added before there is an adversary to measure it against is
// unfalsifiable, and the whole point of M2 is to make the first one
// falsifiable before adding more.
package feature

import "math"

// SetRef versions the extractor set. It is stored on every evaluation,
// so a record computed under pointer-v1 stays readable after the
// definition changes. Bump it whenever the arithmetic below changes.
const SetRef = "pointer-v2"

const (
	// segmentGapMs splits the pointer stream into distinct movement
	// segments. Whole-session straightness is meaningless — a session is
	// many separate moves, and the straight line between the first and
	// last point of a five-minute session says nothing about any of
	// them. 300ms is roughly the pause between purposeful movements.
	segmentGapMs = 300

	// minSegmentPoints is the shortest polyline that can carry shape.
	// With fewer than four points, "straightness" is measuring sampling
	// artefacts rather than trajectory.
	minSegmentPoints = 4

	// minSegmentPathPx drops incidental jitter — a 12px twitch is
	// perfectly straight and tells us nothing. Short moves are where
	// humans look most robotic, so admitting them would inflate the
	// score for everyone.
	minSegmentPathPx = 40.0

	// turnThresholdRad splits a segment when the pointer changes
	// direction sharply, independent of timing.
	//
	// Time alone is not enough, and the M2 harness proved it. A bot
	// filling a form moves to each field back-to-back with no pause, so
	// the 300ms rule never fires and three straight legs merge into one
	// path whose direction changes look exactly like human correction:
	// path 1345px, net 424px, straightness 0.315 — a perfect evasion,
	// produced by nothing more sophisticated than not pausing.
	//
	// A human reach curves continuously and turns gently; arriving at a
	// different target is a sharp corner. 60° separates the two.
	// Uncalibrated, like every other constant here.
	turnThresholdRad = 60 * math.Pi / 180

	// minTurnStepPx is the shortest step whose direction is trusted.
	// Below this, quantisation to integer pixels dominates the angle and
	// a slow human drag would shatter into fragments.
	minTurnStepPx = 4.0
)

// Point is one pointer sample. DtMs is milliseconds since the previous
// sample in the same polyline.
type Point struct {
	X, Y int32
	DtMs uint32
}

// Pointer accumulates pointer geometry for one session.
//
// It holds running aggregates, not the raw points: state is O(1) in
// session duration, which is the property that makes maintained state
// cheap at concurrency (see REFACTOR-PLAN §8.1). Deliberately not
// safe for concurrent use — the session store serializes access.
type Pointer struct {
	// weightedStraightness is Σ(straightness_i × pathLen_i) over
	// qualifying segments; dividing by totalPathPx gives the
	// length-weighted mean. Weighting by length matters because a long
	// deliberate drag is far more informative than a short flick.
	weightedStraightness float64
	totalPathPx          float64
	segments             uint32
	points               uint32

	// Current in-progress segment.
	curPathPx         float64
	curFirst, curLast Point
	curCount          int

	// lastAbsMs is the session-relative time of the most recent sample,
	// carried across Add calls. Polylines are cut by the SDK at batch
	// and buffer boundaries, so consecutive calls may be continuous
	// motion or may be minutes apart — without absolute time the gap is
	// invisible and two unrelated moves get welded into one segment.
	lastAbsMs uint32
	started   bool

	// Direction of the most recent trusted step, for turn detection.
	dirX, dirY float64
	hasDir     bool
}

// PointerState is the extracted feature vector.
type PointerState struct {
	// Straightness is the path-length-weighted mean over qualifying
	// segments, in [0, 1]. Net displacement over path length: 1.0 is a
	// perfectly straight line, and a human reaching for a button
	// typically lands well below it because the approach corrects.
	Straightness float64

	// Segments that met the qualifying thresholds.
	Segments uint32

	// PathPx is total qualifying path length.
	PathPx float64

	// Points is every sample accumulated, including those in segments
	// that did not qualify. This is the honest evidence count — it says
	// how much we saw, not how much we used.
	Points uint32
}

// Add appends one polyline that began at startMs, session-relative.
//
// Within a polyline, each point's DtMs is the gap from its predecessor.
// Between polylines the gap is computed from startMs, because the SDK
// cuts polylines at batch boundaries regardless of whether the pointer
// was still moving — so a polyline boundary means nothing on its own,
// and only elapsed time distinguishes continued motion from a new one.
func (p *Pointer) Add(startMs uint32, pts []Point) {
	for i, pt := range pts {
		p.points++

		// Gap since the previous sample. For the first point of a
		// polyline that is the inter-polyline gap; within one, it is
		// the point's own dt.
		gap := pt.DtMs
		if i == 0 {
			gap = 0
			if p.started && startMs > p.lastAbsMs {
				gap = startMs - p.lastAbsMs
			}
			p.lastAbsMs = startMs
		} else {
			p.lastAbsMs += pt.DtMs
		}
		p.started = true

		if p.curCount == 0 {
			p.startSegment(pt)
			continue
		}
		if gap > segmentGapMs {
			p.closeSegment()
			p.startSegment(pt)
			continue
		}

		// Direction change, independent of timing.
		stepLen := dist(p.curLast, pt)
		if p.hasDir && stepLen >= minTurnStepPx {
			dx, dy := float64(pt.X-p.curLast.X), float64(pt.Y-p.curLast.Y)
			if turnAngle(p.dirX, p.dirY, dx, dy) > turnThresholdRad {
				p.closeSegment()
				p.startSegment(p.curLast) // the corner belongs to both legs
				p.curPathPx = stepLen
				p.curLast = pt
				p.curCount = 2
				p.dirX, p.dirY = dx, dy
				// startSegment cleared this; without restoring it the
				// very next step skips its turn check, so every second
				// corner is missed and each segment swallows one jump.
				p.hasDir = true
				continue
			}
		}
		if stepLen >= minTurnStepPx {
			p.dirX, p.dirY = float64(pt.X-p.curLast.X), float64(pt.Y-p.curLast.Y)
			p.hasDir = true
		}

		p.curPathPx += stepLen
		p.curLast = pt
		p.curCount++
	}
}

// turnAngle returns the angle between two direction vectors, in radians.
func turnAngle(ax, ay, bx, by float64) float64 {
	na, nb := math.Hypot(ax, ay), math.Hypot(bx, by)
	if na == 0 || nb == 0 {
		return 0
	}
	cos := (ax*bx + ay*by) / (na * nb)
	if cos > 1 {
		cos = 1
	} else if cos < -1 {
		cos = -1
	}
	return math.Acos(cos)
}

// State returns the current feature vector, including any in-progress
// segment.
//
// Including the open segment matters: a decision can arrive mid-motion,
// and discarding the movement in flight would systematically
// under-report evidence at exactly the moment it is being asked for.
func (p *Pointer) State() PointerState {
	weighted, total, segs := p.weightedStraightness, p.totalPathPx, p.segments

	if s, l, ok := p.currentSegment(); ok {
		weighted += s * l
		total += l
		segs++
	}

	st := PointerState{Segments: segs, PathPx: total, Points: p.points}
	if total > 0 {
		st.Straightness = weighted / total
	}
	return st
}

func (p *Pointer) startSegment(pt Point) {
	p.curFirst, p.curLast = pt, pt
	p.curPathPx = 0
	p.curCount = 1
	p.hasDir = false
}

func (p *Pointer) closeSegment() {
	if s, l, ok := p.currentSegment(); ok {
		p.weightedStraightness += s * l
		p.totalPathPx += l
		p.segments++
	}
	p.curCount = 0
	p.curPathPx = 0
}

// currentSegment returns the straightness and path length of the
// in-progress segment, and whether it qualifies.
func (p *Pointer) currentSegment() (straightness, pathPx float64, ok bool) {
	if p.curCount < minSegmentPoints || p.curPathPx < minSegmentPathPx {
		return 0, 0, false
	}
	net := dist(p.curFirst, p.curLast)

	// Path length is a sum of segment lengths and net displacement is a
	// straight line between endpoints, so net ≤ path by the triangle
	// inequality. Clamp anyway: float accumulation over a long path can
	// drift, and a straightness above 1.0 would silently poison the
	// weighted mean.
	s := net / p.curPathPx
	if s > 1 {
		s = 1
	}
	return s, p.curPathPx, true
}

func dist(a, b Point) float64 {
	dx := float64(b.X - a.X)
	dy := float64(b.Y - a.Y)
	return math.Hypot(dx, dy)
}
