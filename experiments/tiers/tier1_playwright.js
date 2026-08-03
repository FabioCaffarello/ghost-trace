/*
 * Tier 1 — Playwright, no humanisation.
 *
 * The baseline adversary: a straightforward automation script written
 * by someone who is not trying to evade anything. Linear stepped mouse
 * moves, immediate clicks, constant-rate typing.
 *
 * `steps` on mouse.move produces evenly spaced collinear points, which
 * is the exact shape pointer linearity exists to catch. Without steps,
 * Playwright teleports the cursor and emits almost no pointermove at
 * all — a different evasion, by absence of evidence rather than by
 * mimicry, and one this tier deliberately does not use so that tier 1
 * measures the common case. The no-movement variant is reported in the
 * write-up as a known gap.
 */
import { chromium } from "playwright";
import { BASE, CHROME, appendResult, decide, summarize, sleep } from "../lib/run.js";

const COHORT = "tier1_playwright_naive";
const N = Number(process.env.GT_N || 25);

async function once(browser, i) {
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const page = await context.newPage();
  await page.goto(BASE + "/", { waitUntil: "networkidle" });

  // Let the session handshake land.
  await page.waitForFunction(() => window.GhostTrace && window.GhostTrace.token(), null, {
    timeout: 5000,
  });

  // Straight-line moves between form targets.
  await page.mouse.move(120, 140);
  await page.mouse.move(640, 300, { steps: 30 });
  await page.click("#u");
  await page.fill("#u", `bot${i}@example.com`);
  await page.mouse.move(660, 380, { steps: 30 });
  await page.click("#p");
  await page.fill("#p", "hunter2");
  await page.mouse.move(300, 520, { steps: 30 });

  // Long enough for at least one telemetry batch at the default 2s.
  await sleep(2500);

  const token = await page.evaluate(() => window.GhostTrace.token());
  await page.evaluate(() => window.GhostTrace.flush());
  await sleep(300);

  const d = await decide(token, { subjectId: `${COHORT}_${i}` });
  await context.close();
  return {
    i,
    score: d.score,
    confidence: d.confidence,
    decision: d.decision,
    shadow_decision: d.shadow_decision,
    events: d.evidence.events,
    duration_ms: d.evidence.duration_ms,
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
