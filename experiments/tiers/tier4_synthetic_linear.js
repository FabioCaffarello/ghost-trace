/*
 * Tier 4 — synthetic linear script, no browser.
 *
 * Speaks the telemetry API directly with a perfectly straight,
 * constant-velocity path. No Chrome, no CDP, no fingerprint surface at
 * all — nothing for a fingerprint-based detector to look at, because
 * there is no browser to fingerprint.
 *
 * This is the floor of the experiment. If pointer linearity cannot
 * detect this, it cannot detect anything, and the feature is dead.
 *
 * It is also the cheapest tier by a wide margin: no browser process,
 * so thousands of sessions run in seconds. That asymmetry is worth
 * stating plainly — the bot side of this experiment is unlimited and
 * its intervals are tight, while the human side is capped by how many
 * people will give up five minutes.
 */
import { BASE, appendResult, decide, summarize } from "../lib/run.js";

const COHORT = "tier4_synthetic_linear";
const N = Number(process.env.GT_N || 200);
const SITE_KEY = process.env.GT_SITE_KEY || "pk_demo";

async function startSession() {
  const res = await fetch(BASE + "/v1/sessions", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      site_key: SITE_KEY,
      page: { path: "/login" },
      client: {
        pointer: "fine",
        touch: false,
        viewport: [1440, 900],
        tz_offset: -180,
        reduced_motion: false,
      },
    }),
  });
  if (!res.ok) throw new Error(`/v1/sessions -> HTTP ${res.status}`);
  return (await res.json()).session_token;
}

async function sendBatch(token, seq) {
  // One straight drag per batch, each starting seconds after the last
  // so they read as distinct movements rather than one long path.
  const start = 1200 + seq * 5000;
  const pts = [];
  for (let j = 0; j < 60; j++) {
    pts.push([100 + j * 10, 120 + j * 4, j === 0 ? 0 : 16]);
  }
  await fetch(BASE + "/v1/telemetry", {
    method: "POST",
    headers: { "Content-Type": "application/json", Authorization: "Bearer " + token },
    body: JSON.stringify({
      session_token: token,
      seq,
      sent_at_ms: start + 60 * 16,
      page: { path: "/login", viewport: [1440, 900] },
      events: [{ type: "pointer", t: start, src: "mouse", pts }],
    }),
  });
}

const rows = [];
for (let i = 0; i < N; i++) {
  try {
    const token = await startSession();
    for (let seq = 0; seq < 6; seq++) await sendBatch(token, seq);
    const d = await decide(token, { subjectId: `${COHORT}_${i}` });
    const row = {
      i,
      score: d.score,
      confidence: d.confidence,
      decision: d.decision,
      shadow_decision: d.shadow_decision,
      events: d.evidence.events,
      duration_ms: d.evidence.duration_ms,
      reasons: d.reasons.map((r) => r.code),
    };
    rows.push(row);
    appendResult(COHORT, row);
  } catch (e) {
    console.error(`  ${COHORT} session ${i} failed: ${e.message}`);
  }
}
summarize(COHORT, rows);
