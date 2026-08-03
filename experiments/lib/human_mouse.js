/*
 * A humanised pointer path.
 *
 * Four properties, each cheap to implement and each targeting something
 * the detector measures:
 *
 *  1. CURVATURE. Hand movement arcs; it does not travel the straight
 *     line between two points. A quadratic Bézier with the control point
 *     offset perpendicular to the A→B axis reproduces this in one line
 *     and destroys straightness on its own.
 *
 *  2. MINIMUM-JERK VELOCITY. Human reaching follows a bell-shaped speed
 *     profile — accelerate, cruise, decelerate — not the constant
 *     velocity a linear interpolation produces. 10t³−15t⁴+6t⁵ is the
 *     standard minimum-jerk reparameterisation.
 *
 *  3. OVERSHOOT AND CORRECTION. Fast aimed movement overshoots and
 *     corrects back. This is the signature the current feature mistakes
 *     for human when a bot produces it accidentally by turning a corner,
 *     so producing it deliberately should be decisive.
 *
 *  4. TREMOR. Sub-pixel to few-pixel noise, which breaks the perfect
 *     collinearity of any interpolated path.
 *
 * Total cost: this file. That is the point of tier 5 — the three tiers
 * detected at 100% were detected because nobody had spent an evening on
 * the mouse, not because the detector is hard to beat.
 */

function minimumJerk(t) {
  return 10 * t ** 3 - 15 * t ** 4 + 6 * t ** 5;
}

function gaussian(rand) {
  // Box–Muller. Tremor is not uniform noise.
  const u = 1 - rand();
  const v = rand();
  return Math.sqrt(-2 * Math.log(u)) * Math.cos(2 * Math.PI * v);
}

/**
 * Fitts's law movement time in milliseconds: a + b·log2(2D/W).
 * Constants are the usual mouse-pointing values.
 */
export function fittsMs(distance, targetWidth = 60) {
  const id = Math.log2((2 * Math.max(distance, 1)) / Math.max(targetWidth, 1));
  return 120 + 180 * Math.max(id, 0.5);
}

/**
 * Build a humanised path from `from` to `to`.
 *
 * Returns [{x, y, dt}] where dt is milliseconds since the previous
 * point, sampled at roughly `hz` — matching what the SDK would keep
 * anyway, so the adversary wastes no events on samples that get
 * decimated away.
 */
export function humanPath(from, to, opts = {}) {
  const {
    hz = 60,
    rand = Math.random,
    tremorPx = 1.1,
    overshoot = true,
    targetWidth = 60,
    // bowScale multiplies the perpendicular curvature. 1.0 is a shallow,
    // plausible reach arc. Sweeping it answers the question a single
    // humanised tier cannot: HOW MUCH curvature does an adversary need
    // before the detector loses them?
    bowScale = 1.0,
  } = opts;

  const dx = to[0] - from[0];
  const dy = to[1] - from[1];
  const distance = Math.hypot(dx, dy);
  if (distance < 2) return [];

  const durationMs = fittsMs(distance, targetWidth);
  const steps = Math.max(6, Math.round((durationMs / 1000) * hz));

  // Control point offset perpendicular to the movement axis. Sign and
  // magnitude vary per movement; a fixed curve would be its own
  // fingerprint.
  const perpX = -dy / distance;
  const perpY = dx / distance;
  const bow = (rand() - 0.5) * 2 * distance * (0.08 + rand() * 0.12) * bowScale;
  const cx = from[0] + dx * 0.5 + perpX * bow;
  const cy = from[1] + dy * 0.5 + perpY * bow;

  // Aim slightly past the target, then correct back into it.
  const overshootPx = overshoot && distance > 120 ? 6 + rand() * 14 : 0;
  const aimX = to[0] + (dx / distance) * overshootPx;
  const aimY = to[1] + (dy / distance) * overshootPx;

  const pts = [];
  let prevT = 0;
  for (let i = 1; i <= steps; i++) {
    const raw = i / steps;
    const t = minimumJerk(raw);
    const mt = 1 - t;
    const x = mt * mt * from[0] + 2 * mt * t * cx + t * t * aimX;
    const y = mt * mt * from[1] + 2 * mt * t * cy + t * t * aimY;

    const tMs = raw * durationMs;
    pts.push({
      x: Math.round(x + gaussian(rand) * tremorPx),
      y: Math.round(y + gaussian(rand) * tremorPx),
      dt: Math.max(1, Math.round(tMs - prevT)),
    });
    prevT = tMs;
  }

  // Correction back onto the target, slower and shorter than the reach.
  if (overshootPx > 0) {
    const last = pts[pts.length - 1];
    const corrSteps = 3 + Math.floor(rand() * 4);
    for (let i = 1; i <= corrSteps; i++) {
      const t = minimumJerk(i / corrSteps);
      pts.push({
        x: Math.round(last.x + (to[0] - last.x) * t + gaussian(rand) * tremorPx),
        y: Math.round(last.y + (to[1] - last.y) * t + gaussian(rand) * tremorPx),
        dt: 16 + Math.round(rand() * 22),
      });
    }
  }

  return pts;
}

/**
 * Drive a Playwright/Puppeteer mouse along a humanised path, sleeping
 * the real inter-point delays so the arrival timing is human too.
 */
export async function moveHuman(mouse, from, to, opts = {}) {
  const pts = humanPath(from, to, opts);
  for (const p of pts) {
    await mouse.move(p.x, p.y);
    if (p.dt > 0) await new Promise((r) => setTimeout(r, p.dt));
  }
  return pts.length ? [pts[pts.length - 1].x, pts[pts.length - 1].y] : from;
}

/** A human pause: reading a label, deciding, finding the next field. */
export function thinkMs(rand = Math.random) {
  return 350 + Math.round(rand() * 900);
}
