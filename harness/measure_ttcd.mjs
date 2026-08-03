/*
 * Time to confident decision: how many events and how many seconds of
 * session before `confidence` crosses the usable threshold.
 *
 * Contract §9 calls this the most interesting number here and notes
 * almost nobody publishes it. It is measured by asking for a decision
 * after every batch rather than only at the end — the decision endpoint
 * is a read of current state, so sampling it costs nothing.
 */
import { BASE, decide } from "./lib/run.js";

const CHALLENGE_FLOOR = 0.40;
const BLOCK_FLOOR = 0.70;

async function startSession() {
  const r = await fetch(BASE + "/v1/sessions", {
    method: "POST", headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ site_key: "pk_demo", page: { path: "/login" },
      client: { pointer: "fine", viewport: [1440, 900] } }),
  });
  return (await r.json()).session_token;
}

// One batch: a straight drag plus a few scripted keystrokes, which is
// what a bot filling a login form emits.
async function batch(token, seq) {
  const start = 1200 + seq * 2000;
  const pts = [];
  for (let j = 0; j < 20; j++) pts.push([100 + j * 12, 120 + j * 5, j === 0 ? 0 : 50]);
  const events = [{ type: "pointer", t: start, src: "mouse", pts }];
  let t = start + 100;
  for (let k = 0; k < 4; k++) {
    events.push({ type: "key", t, phase: "down", class: "alpha", target: "f_1" });
    events.push({ type: "key", t: t + 30, phase: "up", class: "alpha", target: "f_1" });
    t += 90; // fixed cadence — the scripted tell
  }
  await fetch(BASE + "/v1/telemetry", {
    method: "POST",
    headers: { "Content-Type": "application/json", Authorization: "Bearer " + token },
    body: JSON.stringify({ session_token: token, seq, sent_at_ms: t,
      page: { path: "/login", viewport: [1440, 900] }, events }),
  });
  return t;
}

const runs = [];
for (let r = 0; r < 30; r++) {
  const token = await startSession();
  let firstUsable = null, firstBlockable = null;
  for (let seq = 0; seq < 14; seq++) {
    const tMs = await batch(token, seq);
    const d = await decide(token, { subjectId: `ttcd_${r}_${seq}` });
    if (!firstUsable && d.confidence >= CHALLENGE_FLOOR) {
      firstUsable = { batches: seq + 1, events: d.evidence.events, ms: tMs, conf: d.confidence };
    }
    if (!firstBlockable && d.confidence >= BLOCK_FLOOR) {
      firstBlockable = { batches: seq + 1, events: d.evidence.events, ms: tMs, conf: d.confidence };
    }
    if (firstUsable && firstBlockable) break;
  }
  runs.push({ firstUsable, firstBlockable });
}

function report(label, key) {
  const hit = runs.map(r => r[key]).filter(Boolean);
  if (!hit.length) { console.log(`${label}: never reached`); return; }
  const med = a => { const s = [...a].sort((x, y) => x - y); return s[Math.floor(s.length / 2)]; };
  console.log(`${label}`);
  console.log(`  reached in ${hit.length}/${runs.length} sessions`);
  console.log(`  median batches: ${med(hit.map(h => h.batches))}`);
  console.log(`  median events : ${med(hit.map(h => h.events))}`);
  console.log(`  median session time: ${(med(hit.map(h => h.ms)) / 1000).toFixed(1)}s`);
}
report(`confidence >= ${CHALLENGE_FLOOR} (challenge is possible)`, "firstUsable");
report(`confidence >= ${BLOCK_FLOOR} (block is possible)`, "firstBlockable");
