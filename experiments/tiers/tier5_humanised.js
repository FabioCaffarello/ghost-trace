/*
 * Tier 5 — Playwright with a humanised mouse.
 *
 * The tier that should win. Tiers 1, 2 and 4 were detected at 100%
 * because nobody had spent an evening on the pointer, not because the
 * detector is hard to beat. This one spends the evening:
 * Bézier curvature, a minimum-jerk velocity profile, overshoot with
 * correction, Gaussian tremor, and human pauses between targets.
 *
 * Deliberately no stealth plugin. Tier 2 established that fingerprint
 * evasion is irrelevant here, so including it would only confound the
 * variable under test. This is a bot that makes no attempt to hide what
 * browser it is and every attempt to move like a person — the exact
 * inverse of tier 2, and the pairing is what makes both informative.
 *
 * If this evades, pointer linearity alone is not a detector and M3's
 * additional evidence types are load-bearing rather than incremental.
 */
import { chromium } from "playwright";
import { BASE, CHROME, appendResult, decide, summarize, sleep } from "../lib/run.js";
import { moveHuman, thinkMs } from "../lib/human_mouse.js";
import { sessionLabel, sessionRand } from "../lib/prng.js";

// Curvature multiplier. The cohort name carries it so a sweep produces
// one labelled population per level rather than a single averaged blur.
const BOW = Number(process.env.GT_BOW || 1.0);
const COHORT = `tier5_humanised_bow${BOW.toFixed(1)}`;
const N = Number(process.env.GT_N || 25);

async function once(browser, i) {
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const page = await context.newPage();
  await page.goto(BASE + "/", { waitUntil: "networkidle" });
  await page.waitForFunction(() => window.GhostTrace && window.GhostTrace.token(), null, {
    timeout: 5000,
  });

  const box = async (sel) => {
    const b = await page.locator(sel).boundingBox();
    return [Math.round(b.x + b.width / 2), Math.round(b.y + b.height / 2)];
  };

  // One generator per session, derived from the run seed and the
  // session index — so session 7 draws the same numbers whether it ran
  // alone or after six others, and a flagged session can be replayed.
  const rand = sessionRand(COHORT, i);

  let at = [80 + rand() * 300, 60 + rand() * 200];

  // Wander first. A person's pointer is in motion before they commit to
  // a target, and a session that begins with a single decisive reach is
  // itself unusual.
  at = await moveHuman(page.mouse, at, [600 + rand() * 200, 300 + rand() * 150],
    { bowScale: BOW, rand });
  await sleep(thinkMs(rand));

  const u = await box("#u");
  at = await moveHuman(page.mouse, at, u, { targetWidth: 300, bowScale: BOW, rand });
  await page.mouse.click(at[0], at[1]);
  await sleep(thinkMs(rand));
  await page.type("#u", `user${i}@example.com`, { delay: 60 + rand() * 80 });

  const p = await box("#p");
  await sleep(thinkMs(rand));
  at = await moveHuman(page.mouse, at, p, { targetWidth: 300, bowScale: BOW, rand });
  await page.mouse.click(at[0], at[1]);
  await page.type("#p", "hunter2", { delay: 70 + rand() * 90 });

  const btn = await box("button[type=submit]");
  await sleep(thinkMs(rand));
  at = await moveHuman(page.mouse, at, btn, { targetWidth: 90, bowScale: BOW, rand});

  await sleep(2500);

  const token = await page.evaluate(() => window.GhostTrace.token());
  await page.evaluate(() => window.GhostTrace.flush());
  await sleep(300);

  const d = await decide(token, { subjectId: `${COHORT}_${i}` });
  await context.close();
  return {
    i,
    // The seed this session drew from. Recorded per row because a
    // seeded run nobody wrote down is exactly as unreproducible as an
    // unseeded one — and because it is what lets one flagged session be
    // replayed on its own.
    seed: sessionLabel(COHORT, i),
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
