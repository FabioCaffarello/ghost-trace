/*
 * A seeded, deterministic pseudo-random generator for the adversarial
 * tiers.
 *
 * Audit finding M13: humanPath already took a `rand` parameter and no
 * caller ever passed one, so tiers 5 and 6 ran on bare Math.random.
 * Every published detection rate from those tiers was therefore a
 * sample of an unrepeatable experiment — and tier 6, at n=10, has now
 * been observed at 70%, 90%, 80% and 100% across four runs of the same
 * code. A number that moves 30 points between runs of an unchanged
 * detector is not a measurement anyone can check.
 *
 * WHAT SEEDING BUYS, AND WHAT IT DOES NOT. It makes the ADVERSARY
 * reproducible: the same seed produces the same intended path, the same
 * pauses, the same target points. It does not make the MEASUREMENT
 * deterministic, because a real browser is in the loop — event
 * dispatch, main-thread scheduling and the SDK's sampling clock are not
 * ours to seed. So this removes the variance we control and leaves the
 * variance we do not, which is the honest half of the problem and the
 * one worth removing first.
 *
 * sfc32 seeded through xmur3: both are small, well-known, integer-only,
 * and therefore identical on every Node version. Math.random is
 * explicitly NOT a fallback anywhere — a tier that silently reverted to
 * it would look seeded and reproduce nothing.
 */

/** xmur3: string -> a sequence of 32-bit seeds. */
function xmur3(str) {
  let h = 1779033703 ^ str.length;
  for (let i = 0; i < str.length; i++) {
    h = Math.imul(h ^ str.charCodeAt(i), 3432918353);
    h = (h << 13) | (h >>> 19);
  }
  return function () {
    h = Math.imul(h ^ (h >>> 16), 2246822507);
    h = Math.imul(h ^ (h >>> 13), 3266489909);
    h ^= h >>> 16;
    return h >>> 0;
  };
}

/** sfc32: fast, small-state, well-distributed. Returns [0, 1). */
function sfc32(a, b, c, d) {
  return function () {
    a |= 0; b |= 0; c |= 0; d |= 0;
    const t = (((a + b) | 0) + d) | 0;
    d = (d + 1) | 0;
    a = b ^ (b >>> 9);
    b = (c + (c << 3)) | 0;
    c = (c << 21) | (c >>> 11);
    c = (c + t) | 0;
    return (t >>> 0) / 4294967296;
  };
}

/**
 * A generator for one session.
 *
 * The label is hashed rather than the index added to a base, so session
 * 7 of tier 6 draws the same numbers whether it ran alone or after six
 * others — which is what makes a single flagged session replayable.
 */
export function seeded(label) {
  const h = xmur3(String(label));
  return sfc32(h(), h(), h(), h());
}

/**
 * The run seed. Fixed by default so the canonical command measures the
 * same adversary every time; override to sample a different one.
 *
 * It is recorded in every result row and in the run manifest, because a
 * seeded number nobody wrote down is exactly as unreproducible as an
 * unseeded one.
 */
export const RUN_SEED = process.env.GT_SEED || "ghost-trace-v1";

/** The label for one session of one cohort under the run seed. */
export function sessionLabel(cohort, i) {
  return `${RUN_SEED}:${cohort}:${i}`;
}

/** Convenience: the generator for one session of one cohort. */
export function sessionRand(cohort, i) {
  return seeded(sessionLabel(cohort, i));
}
