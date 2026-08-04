/*
 * Tier 2 — puppeteer-extra-stealth.
 *
 * The interesting tier, and the one the project's thesis stands or falls
 * on.
 *
 * puppeteer-extra-plugin-stealth is the standard answer to detection: it
 * patches navigator.webdriver, spoofs the plugin and mimeType arrays,
 * fixes WebGL vendor strings, repairs iframe.contentWindow, hides the
 * CDP marker, and about a dozen more. Every one of those evasions
 * targets *fingerprinting* — the question of which browser this is.
 *
 * None of them touch the mouse. The pointer path is identical to tier 1.
 *
 * If Ghost Trace's thesis is right, this tier should be as detectable as
 * the naive one: stealth defeats the question "which browser is this?"
 * and says nothing about "does this session behave like a human?". If
 * tier 2 evades where tier 1 does not, the thesis is wrong and the
 * write-up says so.
 */
import puppeteer from "puppeteer-extra";
import StealthPlugin from "puppeteer-extra-plugin-stealth";
import { DEMO_BASE, CHROME, appendResult, decide, summarize, sleep } from "../lib/run.js";

puppeteer.use(StealthPlugin());

const COHORT = "tier2_puppeteer_stealth";
const N = Number(process.env.GT_N || 25);

async function once(browser, i) {
  const page = await browser.newPage();
  await page.setViewport({ width: 1440, height: 900 });
  await page.goto(DEMO_BASE + "/", { waitUntil: "networkidle0" });
  await page.waitForFunction(() => window.GhostTrace && window.GhostTrace.token(), {
    timeout: 5000,
  });

  await page.mouse.move(120, 140);
  await page.mouse.move(640, 300, { steps: 30 });
  await page.click("#u");
  await page.type("#u", `bot${i}@example.com`);
  await page.mouse.move(660, 380, { steps: 30 });
  await page.click("#p");
  await page.type("#p", "hunter2");
  await page.mouse.move(300, 520, { steps: 30 });

  await sleep(2500);

  const token = await page.evaluate(() => window.GhostTrace.token());
  await page.evaluate(() => window.GhostTrace.flush());
  await sleep(300);

  const d = await decide(token, { subjectId: `${COHORT}_${i}` });
  await page.close();
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

const browser = await puppeteer.launch({
  executablePath: CHROME,
  headless: "new",
  args: ["--no-sandbox"],
});
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
