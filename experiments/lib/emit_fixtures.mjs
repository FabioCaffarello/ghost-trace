/*
 * Print the request bodies lib/wire.js produces, as JSON on stdout.
 *
 * Driven by emit_fixtures.py, which asks the same of wire.py and
 * refuses to write anything unless the two agree byte for byte. The
 * harness has a JavaScript half and a Python half; if they disagree
 * about the wire, one of them is measuring something else.
 *
 * The values are fixed, not sampled — a fixture that changed every
 * run could not be committed, and a drift gate needs something stable
 * to compare against.
 */
import {
  decisionBody,
  keyEvent,
  outcomeBody,
  pointerEvent,
  sessionBody,
  telemetryBody,
} from "./wire.js";

const TOKEN = "st_AufHcXG3MEt9x5F3hzVf03ZS";

const pointer = pointerEvent({
  t: 1200,
  pts: [
    [412, 300, 0],
    [418, 305, 16],
    [427, 311, 17],
  ],
});

const keys = [
  keyEvent({ t: 1500, phase: "down", keyClass: "alpha", target: "f_1" }),
  keyEvent({ t: 1530, phase: "up", keyClass: "alpha", target: "f_1" }),
];

const fixtures = {
  sessions: sessionBody({ siteKey: "pk_demo" }),

  // The naive-automation shape: one straight constant-velocity drag.
  telemetry_pointer: telemetryBody({
    sessionToken: TOKEN,
    seq: 0,
    sentAtMs: 2840,
    events: [pointer],
  }),

  // Pointer plus keystrokes, which is what a real form fill produces
  // and what the key-timing signals need in order to exist at all.
  telemetry_pointer_and_keys: telemetryBody({
    sessionToken: TOKEN,
    seq: 1,
    sentAtMs: 4200,
    events: [pointer, ...keys],
  }),

  decisions: decisionBody({ sessionToken: TOKEN, subjectId: "user_8f21" }),

  // The labels channel has no client yet. Covering it here means the
  // first one will not have to invent the shape.
  outcomes: outcomeBody({ evaluationId: "ev_5Kq2mXbT9vHs", outcome: "login_success" }),
  outcomes_with_observed_at: outcomeBody({
    evaluationId: "ev_5Kq2mXbT9vHs",
    outcome: "fraud_confirmed",
    observedAt: "2026-08-04T09:15:00Z",
  }),
};

process.stdout.write(JSON.stringify(fixtures, null, 2) + "\n");
