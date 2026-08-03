/*
 * Tier 6 — humanised mouse plus value injection.
 *
 * The attack M3 named. It combines the two evasions the harness has
 * already proven work, each on a different channel:
 *
 *   - the pointer channel is defeated by tier 5's humanised path
 *   - the keystroke channel is defeated by NOT TYPING — page.fill()
 *     sets a field's value directly, so no key event ever fires
 *
 * This is the strongest tier so far and it costs nothing beyond
 * combining two things already written. That is the pattern worth
 * noticing: every channel closed so far was closed against adversaries
 * that attacked one at a time, and defeating a multi-channel detector
 * means declining to produce evidence on each channel rather than
 * faking any of it.
 *
 * The counter-signal is that declining to type is itself observable. A
 * field whose contents change with no keystroke and no paste behind it
 * was not filled by a person.
 */
import { chromium } from "playwright";
import { BASE, CHROME, appendResult, decide, summarize, sleep } from "../lib/run.js";
import { moveHuman, thinkMs } from "../lib/human_mouse.js";

const BOW = Number(process.env.GT_BOW || 3.0);
const COHORT = "tier6_value_injection";
const N = Number(process.env.GT_N || 15);

async function once(browser, i) {
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const page = await context.newPage();
  await page.goto(BASE + "/", { waitUntil: "networkidle" });
  await page.waitForFunction(() => window.GhostTrace && window.GhostTrace.token(), null,
    { timeout: 5000 });

  const box = async (sel) => {
    const b = await page.locator(sel).boundingBox();
    return [Math.round(b.x + b.width / 2), Math.round(b.y + b.height / 2)];
  };

  let at = [90 + Math.random() * 300, 70 + Math.random() * 200];
  at = await moveHuman(page.mouse, at, [620 + Math.random() * 180, 310 + Math.random() * 140],
    { bowScale: BOW });
  await sleep(thinkMs());

  // Move like a person, then fill like a script.
  const u = await box("#u");
  at = await moveHuman(page.mouse, at, u, { targetWidth: 300, bowScale: BOW });
  await page.fill("#u", `user${i}@example.com`);
  await sleep(thinkMs());

  const p = await box("#p");
  at = await moveHuman(page.mouse, at, p, { targetWidth: 300, bowScale: BOW });
  await page.fill("#p", "hunter2");
  await sleep(thinkMs());

  const btn = await box("button[type=submit]");
  at = await moveHuman(page.mouse, at, btn, { targetWidth: 90, bowScale: BOW });

  await sleep(2500);
  const token = await page.evaluate(() => window.GhostTrace.token());
  await page.evaluate(() => window.GhostTrace.flush());
  await sleep(300);

  const d = await decide(token, { subjectId: `${COHORT}_${i}` });
  await context.close();
  return {
    i, score: d.score, confidence: d.confidence,
    decision: d.decision, shadow_decision: d.shadow_decision,
    events: d.evidence.events, duration_ms: d.evidence.duration_ms,
    reasons: d.reasons.map((r) => r.code),
  };
}

const browser = await chromium.launch({ executablePath: CHROME, headless: true });
const rows = [];
for (let i = 0; i < N; i++) {
  try {
    const row = await once(browser, i);
    rows.push(row);
    appendResult(COHORT, row);
  } catch (e) {
    console.error(`  ${COHORT} session ${i} failed: ${e.message}`);
  }
}
await browser.close();
summarize(COHORT, rows);
