/*
 * Session-level measurements: decision latency, time to confident
 * decision, and cold-start behaviour.
 *
 * Numbers 3, 4 and 5 of the six. Emits JSON on stdout for numbers.py.
 */
import { BASE, decide } from "./lib/run.js";
import { keyEvent, pointerEvent, sessionBody, telemetryBody } from "./lib/wire.js";

const CHALLENGE_FLOOR = 0.40;
const BLOCK_FLOOR = 0.70;

async function startSession() {
  const r = await fetch(BASE + "/v1/sessions", {
    method: "POST", headers: { "Content-Type": "application/json" },
    body: JSON.stringify(sessionBody({ siteKey: "pk_demo" })),
  });
  return (await r.json()).session_token;
}

async function sendBatch(token, seq) {
  const start = 1200 + seq * 2000;
  const pts = [];
  for (let j = 0; j < 20; j++) pts.push([100 + j * 12, 120 + j * 5, j === 0 ? 0 : 50]);
  const events = [pointerEvent({ t: start, pts })];
  let t = start + 100;
  for (let k = 0; k < 4; k++) {
    events.push(keyEvent({ t, phase: "down", keyClass: "alpha", target: "f_1" }));
    events.push(keyEvent({ t: t + 30, phase: "up", keyClass: "alpha", target: "f_1" }));
    t += 90;
  }
  await fetch(BASE + "/v1/telemetry", {
    method: "POST",
    headers: { "Content-Type": "application/json", Authorization: "Bearer " + token },
    body: JSON.stringify(telemetryBody({ sessionToken: token, seq, sentAtMs: t, events })),
  });
  return t;
}

const median = a => { const s = [...a].sort((x, y) => x - y); return s[Math.floor(s.length / 2)]; };
const pct = (a, p) => { const s = [...a].sort((x, y) => x - y); return s[Math.floor((s.length - 1) * p)]; };

// ---- 5. cold start: a first visit with no history ----
// Asked BEFORE any telemetry, which is the condition the contract
// singles out: a session eleven events old can look bot-like and must
// not be blocked.
const cold = [];
for (let i = 0; i < 30; i++) {
  const token = await startSession();
  const d = await decide(token, { subjectId: `cold_${i}` });
  cold.push(d);
}
const coldStart = {
  n: cold.length,
  score: median(cold.map(d => d.score)),
  confidence: median(cold.map(d => d.confidence)),
  decisions: [...new Set(cold.map(d => d.shadow_decision || d.decision))],
  reasons: [...new Set(cold.flatMap(d => d.reasons.map(r => r.code)))],
  never_blocks: cold.every(d => (d.shadow_decision || d.decision) !== "block"),
};

// ---- 4. time to confident decision ----
const usable = [], blockable = [];
for (let r = 0; r < 30; r++) {
  const token = await startSession();
  let gotU = false, gotB = false;
  for (let seq = 0; seq < 14; seq++) {
    const tMs = await sendBatch(token, seq);
    const d = await decide(token, { subjectId: `ttcd_${r}_${seq}` });
    if (!gotU && d.confidence >= CHALLENGE_FLOOR) {
      usable.push({ batches: seq + 1, events: d.evidence.events, ms: tMs }); gotU = true;
    }
    if (!gotB && d.confidence >= BLOCK_FLOOR) {
      blockable.push({ batches: seq + 1, events: d.evidence.events, ms: tMs }); gotB = true;
    }
    if (gotU && gotB) break;
  }
}
const ttcd = {
  challenge: usable.length ? {
    reached: usable.length, of: 30,
    batches: median(usable.map(u => u.batches)),
    events: median(usable.map(u => u.events)),
    seconds: +(median(usable.map(u => u.ms)) / 1000).toFixed(1),
  } : null,
  block: blockable.length ? {
    reached: blockable.length, of: 30,
    batches: median(blockable.map(u => u.batches)),
    events: median(blockable.map(u => u.events)),
    seconds: +(median(blockable.map(u => u.ms)) / 1000).toFixed(1),
  } : null,
};

// ---- 3. p99 decision latency, single session, idle system ----
const token = await startSession();
for (let seq = 0; seq < 20; seq++) await sendBatch(token, seq);
const lat = [];
for (let i = 0; i < 500; i++) {
  const t0 = process.hrtime.bigint();
  await decide(token, { subjectId: "lat" });
  lat.push(Number(process.hrtime.bigint() - t0) / 1e6);
}
const latency = {
  n: lat.length,
  p50: +pct(lat, 0.50).toFixed(3),
  p95: +pct(lat, 0.95).toFixed(3),
  p99: +pct(lat, 0.99).toFixed(3),
  max: +pct(lat, 1.0).toFixed(3),
  budget_ms: 80,
};

console.log(JSON.stringify({ cold_start: coldStart, ttcd, latency }, null, 2));
