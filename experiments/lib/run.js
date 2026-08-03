/*
 * Shared harness plumbing: run one session against the slice, extract
 * the decision, append a labelled result.
 *
 * The engine has no idea a harness exists. Cohort labels live here, not
 * in the API — a session's population is a property of the experiment,
 * not of the product, and putting it in the contract would mean the
 * thing being measured knows which population it is looking at.
 */
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const HERE = path.dirname(fileURLToPath(import.meta.url));
export const RESULTS_DIR = path.join(HERE, "..", "results");

export const BASE = process.env.GT_BASE || "http://127.0.0.1:8080";
export const SECRET_KEY = process.env.GT_SECRET_KEY || "sk_demo";

// Chrome on this machine. puppeteer-core and playwright both drive the
// installed browser rather than downloading their own — a bot that
// evades detection in a bundled Chromium but not in real Chrome would
// be measuring the wrong thing.
export const CHROME =
  process.env.GT_CHROME ||
  "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome";

export function appendResult(cohort, row) {
  fs.mkdirSync(RESULTS_DIR, { recursive: true });
  const file = path.join(RESULTS_DIR, "sessions.jsonl");
  fs.appendFileSync(file, JSON.stringify({ cohort, ...row }) + "\n");
}

/**
 * Ask the engine for a decision on a session, as the host application
 * would: server-to-server, authenticated with secret_key.
 */
export async function decide(sessionToken, { action = "login", subjectId = "harness" } = {}) {
  const res = await fetch(BASE + "/v1/decisions", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: "Bearer " + SECRET_KEY,
    },
    body: JSON.stringify({
      session_token: sessionToken,
      action,
      subject_id: subjectId,
      context: { harness: true },
    }),
  });
  if (!res.ok) throw new Error(`/v1/decisions -> HTTP ${res.status}`);
  return res.json();
}

/**
 * A straight, constant-velocity interpolation between two points.
 *
 * This is what naive automation produces, and it is the single most
 * common shape in scripted browsing: no acceleration profile, no
 * correction, no overshoot.
 */
export function linearPath(from, to, steps) {
  const pts = [];
  for (let i = 0; i <= steps; i++) {
    const t = i / steps;
    pts.push([
      Math.round(from[0] + (to[0] - from[0]) * t),
      Math.round(from[1] + (to[1] - from[1]) * t),
    ]);
  }
  return pts;
}

export function summarize(cohort, rows) {
  const scored = rows.filter((r) => r.score !== undefined);
  if (scored.length === 0) {
    console.log(`  ${cohort}: no sessions completed`);
    return;
  }
  const mean = (xs) => xs.reduce((a, b) => a + b, 0) / xs.length;
  const detected = scored.filter((r) => r.shadow_decision !== "allow" || r.decision !== "allow");
  console.log(
    `  ${cohort.padEnd(26)} n=${String(scored.length).padStart(3)}  ` +
      `score=${mean(scored.map((r) => r.score)).toFixed(3)}  ` +
      `conf=${mean(scored.map((r) => r.confidence)).toFixed(3)}  ` +
      `events=${Math.round(mean(scored.map((r) => r.events)))}  ` +
      `flagged=${detected.length}/${scored.length}`
  );
}

export function sleep(ms) {
  return new Promise((r) => setTimeout(r, ms));
}
