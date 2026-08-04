/*
 * The ONE place the experiment harness builds request bodies.
 *
 * Until R1.15 there were three: tier4 wrote its own session and
 * telemetry envelopes, measure_session.mjs wrote a second pair, and
 * lib/run.js wrote the decision body. Each was hand-rolled JSON,
 * outside all buf discipline, and the server tolerates unknown fields
 * by design (§5, §7) — so renaming a wire field would have left every
 * producer sending the old name, the server zero-filling the new one,
 * and every measurement quietly degrading with the suite still green.
 * That is audit finding M22, and it is the exact accident
 * docs/architecture.md §2 forbids.
 *
 * Two things follow from putting the shapes here. A rename is now one
 * edit rather than four. And `make contract-fixtures` emits the
 * payloads THESE functions produce into contract/fixtures/, where a Go
 * test validates them against the published OpenAPI request schemas —
 * which set `additionalProperties: false`, so a field the server no
 * longer knows about fails, and so does one it expects and never
 * receives.
 *
 * Nothing here decides anything. Behaviour — what a tier does with a
 * pointer — stays in the tier; this file only says what the wire looks
 * like.
 */

/** POST /v1/sessions */
export function sessionBody({
  siteKey,
  path = "/login",
  pointer = "fine",
  touch = false,
  viewport = [1440, 900],
  tzOffset = -180,
  reducedMotion = false,
} = {}) {
  return {
    site_key: siteKey,
    page: { path },
    client: {
      pointer,
      touch,
      viewport,
      tz_offset: tzOffset,
      reduced_motion: reducedMotion,
    },
  };
}

/** POST /v1/telemetry */
export function telemetryBody({
  sessionToken,
  seq,
  sentAtMs,
  path = "/login",
  viewport = [1440, 900],
  events,
} = {}) {
  return {
    session_token: sessionToken,
    seq,
    sent_at_ms: sentAtMs,
    page: { path, viewport },
    events,
  };
}

/**
 * A pointer event. `pts` are [x, y, dt_ms] triples — the third element
 * is the gap since the previous sample, not an absolute time, which is
 * the detail a hand-rolled producer gets wrong.
 */
export function pointerEvent({ t, src = "mouse", pts }) {
  return { type: "pointer", t, src, pts };
}

/**
 * A keystroke event. Timing and a COARSE class only, never the key
 * itself — §2 and §6. There is deliberately no parameter here that
 * could carry a character.
 */
export function keyEvent({ t, phase, keyClass, target }) {
  return { type: "key", t, phase, class: keyClass, target };
}

/** POST /v1/decisions */
export function decisionBody({ sessionToken, action = "login", subjectId = "harness" } = {}) {
  return {
    session_token: sessionToken,
    action,
    subject_id: subjectId,
  };
}

/**
 * POST /v1/outcomes — the labels channel.
 *
 * Nothing in the harness calls this yet, which is itself a finding: it
 * is the channel every future calibration depends on and it has no
 * client at all. The shape is defined here so the contract covers it
 * and so the first caller does not have to invent it.
 */
export function outcomeBody({ evaluationId, outcome, observedAt = null } = {}) {
  const body = { evaluation_id: evaluationId, outcome };
  if (observedAt !== null) body.observed_at = observedAt;
  return body;
}
