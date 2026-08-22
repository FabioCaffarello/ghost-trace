/*
 * The end-to-end gate: one real browser, one real session, the whole
 * chain, and a named failure for every link in it.
 *
 * WHY THIS EXISTS WHEN THREE GATES ALREADY INTERROGATE THE TOPOLOGY.
 * `shadow-http` asks the collector and the engine the same question and
 * compares the answers. `kill-test` takes a service away and checks what
 * the rest promises. `loss-audit` reconciles the archive's books. All
 * three enter the chain in the MIDDLE, with request bodies they build
 * themselves — none of them ever loads the page, and none of them runs
 * the SDK.
 *
 * What broke, twice, was neither the policy nor the arithmetic. It was
 * WIRING: a page naming an API origin no in-network browser could reach,
 * and a demo host dialling an engine it could not resolve. Both were
 * found by hand, days later, because a system that is wired wrong still
 * answers every request and still looks healthy. Fail-open (§5) is not a
 * bug here — it is the promise — and it means a broken decision path
 * returns `allow` forever with nothing in any log saying so.
 *
 * So this drives the product path a customer's visitor actually takes:
 *
 *   page → sdk.js → POST /v1/sessions → POST /v1/telemetry
 *        → NATS → snapshot KV
 *        → demo-web calls POST /v1/decisions server-to-server
 *        → verdict rendered → demo-web files POST /v1/outcomes
 *        → records committed to the archive
 *
 * WHAT IT IS NOT. It is not a measurement. The adversarial tiers in
 * experiments/ measure DETECTION and take about seven minutes; this
 * asserts CONNECTION and takes about one. A tier that scores badly is a
 * finding. A link that fails here is a broken deployment.
 *
 * NOTHING HERE IS SKIPPABLE. Every link either holds or fails by name.
 * A link that cannot be observed is a failure, never a pass — the same
 * rule that makes `shadow-http` refuse without a topology rather than
 * report zero tests.
 *
 * Runs INSIDE the compose network: every address below is a service
 * name, because the browser it drives has to be able to reach what the
 * page names, which is the whole point of link 2.
 */
import { chromium } from "playwright";

const DEMO = process.env.GT_DEMO_BASE || "http://demo-web-internal:8083";
const COLLECTOR = process.env.GT_BASE || "http://collector:8080";
const ENGINE = process.env.GT_ENGINE_BASE || "http://decision-engine:8082";
const ARCHIVE = process.env.GT_ARCHIVE_BASE || "http://archive:8081";
const CHROME = process.env.GT_CHROME || "";

// The mode the topology was brought up in. Both the collector and the
// engine take it, and `make e2e GT_MODE=enforce` sets it for the copy
// this gate brings up. It is here so the run can assert the CONTRACT
// that mode implies (§3) rather than the verdict, which is a detection
// question and not this gate's business (ADR-0015).
const MODE = process.env.GT_MODE || "monitor";

// The archive commits asynchronously. This is how long link 8 waits for
// the stream to drain before calling it a loss — generous, because a
// timeout here must mean "did not arrive", not "arrived slowly".
const DRAIN_TIMEOUT_MS = 60_000;

/* ------------------------------------------------------------------ *
 * Reporting
 * ------------------------------------------------------------------ */

const links = [];
let current = null;

function link(n, name) {
  current = { n, name, checks: [] };
  links.push(current);
  console.log(`\n== link ${n} — ${name} ==`);
}

function check(ok, what) {
  current.checks.push({ ok, what });
  console.log(`  ${ok ? "ok  " : "FAIL"}  ${what}`);
  return ok;
}

/* ------------------------------------------------------------------ *
 * Metrics
 * ------------------------------------------------------------------ */

/*
 * A counter, or null when the series is ABSENT.
 *
 * Absent and zero are different facts and ADR-0008 is about exactly this
 * distinction: a process that has never served a route publishes no
 * series for it, and folding that to 0.0 would let "the engine is not
 * exposing metrics at all" read as "the engine served no decisions" —
 * which is the reading link 5 depends on being able to tell apart.
 */
async function counter(base, series, labels = {}) {
  const r = await fetch(base + "/metrics", { signal: AbortSignal.timeout(10_000) });
  if (!r.ok) throw new Error(`${base}/metrics answered ${r.status}`);
  const body = await r.text();
  const wanted = Object.entries(labels)
    .map(([k, v]) => `${k}="${v}"`)
    .sort();
  for (const line of body.split("\n")) {
    if (line.startsWith("#") || !line.startsWith("ghosttrace_" + series)) continue;
    const m = line.match(/^[a-z_]+(?:\{([^}]*)\})?\s+(-?[\d.eE+]+)$/);
    if (!m) continue;
    const got = (m[1] || "").split(",").filter(Boolean).sort();
    if (wanted.every((w) => got.includes(w))) return Number(m[2]);
  }
  return null;
}

const moved = (before, after) => (after ?? 0) - (before ?? 0);

/* ------------------------------------------------------------------ *
 * The run
 * ------------------------------------------------------------------ */

const seen = { sdk: null, sessions: null, telemetry: [], failures: [], aborted: [], login: null };

async function run() {
  // Read before anything is driven. The archive is empty in this
  // project — the gate brings its own topology up — but a baseline read
  // is what makes link 8 a DELTA rather than an absolute, and a delta
  // survives a rerun against a topology that was already carrying rows.
  const archiveBefore = await counter(ARCHIVE, "archive_position_committed");
  const engineBefore = await counter(ENGINE, "http_requests_total", {
    route: "POST /v1/decisions",
    status: "200",
  });
  const collectorBefore = await counter(COLLECTOR, "http_requests_total", {
    route: "POST /v1/decisions",
    status: "200",
  });
  const outcomeBefore = await counter(ENGINE, "http_requests_total", {
    route: "POST /v1/outcomes",
    status: "202",
  });

  /* ---- link 1: the page ---- */
  link(1, "the page the visitor opens");
  const pageRes = await fetch(DEMO + "/", { signal: AbortSignal.timeout(10_000) });
  const html = await pageRes.text();
  check(pageRes.status === 200, `demo-web serves the page (${pageRes.status})`);
  // A page still carrying its placeholders renders identically, scores
  // identically in a screenshot, and authenticates nothing: the SDK gets
  // the literal string "SITE_KEY_PLACEHOLDER" as a site key.
  check(
    !html.includes("PLACEHOLDER"),
    "the site key and API origin were substituted into it " +
      "(a page still carrying PLACEHOLDER renders exactly the same)",
  );
  const named = (html.match(/window\.GHOST_TRACE_API\s*=\s*"([^"]*)"/) || [])[1];
  check(
    named === COLLECTOR,
    `the origin it names is the one this browser can reach ` +
      `(page says ${named || "nothing"}, collector is at ${COLLECTOR})`,
  );

  if (!CHROME) {
    check(false, "GT_CHROME names a Chromium — without a browser there is no e2e, and a skip is not a pass");
    return;
  }

  const browser = await chromium.launch({ executablePath: CHROME, headless: true });
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const page = await context.newPage();

  // Every observation below comes from the NETWORK, not from the page's
  // status line. The SDK swallows a failed telemetry POST on purpose
  // (§5: loss is expected, retrying would amplify an outage), so a CORS
  // refusal leaves the page saying "sent batch #0" with nothing sent.
  page.on("response", async (res) => {
    const url = res.url();
    // Method-filtered: a CORS preflight answers on the SAME url, and an
    // OPTIONS 204 recorded as the session response reads as "the
    // collector opened a session and issued no token" — a true statement
    // about the wrong request.
    const post = res.request().method() === "POST";
    // The host application's own response, as the SERVER sent it. Every
    // other decision assertion here reads the DOM, which is the page's
    // account of itself; this is the account it was given. They are
    // different sources and link 5 needs both.
    if (post && url.endsWith("/demo/login")) {
      try {
        seen.login = await res.json();
      } catch {
        seen.login = null;
      }
    }
    if (url.endsWith("/sdk.js")) seen.sdk = { url, status: res.status() };
    else if (post && url.endsWith("/v1/sessions")) {
      let token = "";
      try {
        token = (await res.json()).session_token || "";
      } catch {
        /* a non-JSON body is itself the finding; recorded as an empty token */
      }
      seen.sessions = { status: res.status(), token };
    } else if (post && url.endsWith("/v1/telemetry")) {
      let authorized = false;
      try {
        authorized = /^Bearer\s+\S/.test((await res.request().allHeaders()).authorization || "");
      } catch {
        // Reading headers is itself async, so a context that closes
        // first makes this throw — inside an async event handler, where
        // the rejection is unhandled and takes the process with it.
        // An unreadable header is recorded as absent and fails its own
        // check, which is a finding rather than a crash.
      }
      seen.telemetry.push({ status: res.status(), authorized });
    }
  });
  // Split by whether a failure is SILENT.
  //
  // sdk.js and /v1/sessions failing outright leaves nothing else to
  // notice it: the page renders, the form submits, and the decision
  // comes back as a cold start that looks like a lenient verdict.
  //
  // A telemetry batch is different. The SDK sends it with keepalive and
  // swallows the rejection by design (§5: loss is expected, retrying
  // would amplify an outage), and Chromium does abort keepalive requests
  // it has already handed to the network — this run saw three ERR_ABORTED
  // for batches the collector logged as 202. Whether anything was
  // actually lost is not a question the browser can answer, and it is
  // the question link 8 exists to answer from the archive's books. So
  // these are reported, not failed on.
  page.on("requestfailed", (req) => {
    const rec = `${req.url()} — ${req.failure()?.errorText}`;
    if (req.url().endsWith("/v1/telemetry")) seen.aborted.push(rec);
    else seen.failures.push(rec);
  });

  try {
    await page.goto(DEMO + "/", { waitUntil: "domcontentloaded", timeout: 20_000 });

    /* ---- link 2: the SDK ---- */
    link(2, "the SDK, fetched from the collector");
    // Cross-origin by construction: the page is the customer's site and
    // the collector is not. Same-origin is what the demo used to be, and
    // it hid every one of these failures.
    check(!!seen.sdk, `the browser requested ${COLLECTOR}/sdk.js from the page's own markup`);
    check(seen.sdk?.status === 200, `the collector served it (${seen.sdk?.status ?? "no response"})`);
    check(
      await page.evaluate(() => typeof window.GhostTrace?.start === "function"),
      "it defined window.GhostTrace — a 200 carrying an error page would not",
    );

    /* ---- link 3: the session ---- */
    link(3, "the session handshake, cross-origin");
    await page
      .waitForFunction(() => window.GhostTrace && window.GhostTrace.token(), null, { timeout: 15_000 })
      .catch(() => {});
    check(!!seen.sessions, "POST /v1/sessions crossed the origin boundary at all");
    check(seen.sessions?.status === 200, `the collector opened a session (${seen.sessions?.status ?? "no response"})`);
    check(
      !!seen.sessions?.token,
      "it issued a session token — a CORS refusal is invisible to the SDK, " +
        "which reports the handshake as failed only in the page's own console",
    );

    /* ---- link 4: telemetry ---- */
    link(4, "telemetry, accepted and attributed");
    // Real pointer input, not _injectLinear: the injection hook writes
    // straight into the SDK's queue and would prove the transport while
    // saying nothing about whether the SDK's own listeners are attached.
    // flush() sends nothing when the queue is empty, so a gate that
    // skipped the movement would pass on silence.
    // `.catch` attached HERE, at creation, not at the await below.
    //
    // This promise is armed several statements before it is awaited, and
    // anything that throws in between — a page with no SDK makes the
    // flush() call throw — unwinds to the report while this is still
    // pending. It then rejects when the context closes, as an UNHANDLED
    // rejection, and node kills the process before a single line of the
    // report is printed. A gate that dies instead of reporting turns a
    // three-link failure into a stack trace, which is how the first
    // deliberately-broken run of this file behaved.
    const telemetry = page
      .waitForResponse((r) => r.url().endsWith("/v1/telemetry"), { timeout: 20_000 })
      .catch(() => null);
    for (let i = 0; i < 40; i++) {
      await page.mouse.move(120 + i * 14, 160 + Math.sin(i / 4) * 90, { steps: 3 });
      await page.waitForTimeout(20);
    }
    // Non-fatal: a page whose SDK never loaded has no flush to call, and
    // that is links 2 and 3's finding to report, not an exception that
    // replaces the whole report with a stack trace.
    await page.evaluate(() => window.GhostTrace?.flush()).catch(() => {});
    await telemetry;
    // The response handler reads request headers, which is itself async,
    // so the record it appends can land after waitForResponse resolves.
    // Asserting on the array without this settle is a race that fails
    // once in a while and reads as a broken deployment.
    await page.waitForTimeout(500);
    // `.every()` on an empty array is TRUE, so both checks below would
    // report ok for a run in which nothing was ever sent — an absence
    // read as a zero, in the one repository whose first rule is that
    // those are different facts. The length guard is not belt and
    // braces; without it a page whose SDK never loaded passes two of
    // link 4's three assertions.
    const sent = seen.telemetry.length > 0;
    check(sent, `at least one batch was sent (${seen.telemetry.length})`);
    check(
      sent && seen.telemetry.every((t) => t.status >= 200 && t.status < 300),
      `every batch was accepted (${seen.telemetry.map((t) => t.status).join(", ") || "none sent"})`,
    );
    check(
      sent && seen.telemetry.every((t) => t.authorized),
      "each carried the session's bearer token — telemetry that is accepted " +
        "but unattributed reaches the archive and no snapshot",
    );

    /* ---- links 5 and 6: the decision ---- */
    link(5, "the decision, answered by the engine");
    // The page's own form: GhostTrace.flush(), then demo-web calls
    // /v1/decisions server-to-server with the secret key. Driving the
    // engine directly — which every adversarial tier does — would skip
    // the one hop that has broken twice.
    await page.click("#login button[type=submit]");
    await page
      .waitForFunction(() => document.getElementById("evalid")?.textContent?.trim().length > 0, null, {
        timeout: 20_000,
      })
      .catch(() => {});

    const shown = await page.evaluate(() => ({
      decision: document.getElementById("decision")?.textContent?.trim(),
      mode: document.getElementById("modeline")?.textContent?.trim(),
      score: document.getElementById("score")?.textContent?.trim(),
      confidence: document.getElementById("confidence")?.textContent?.trim(),
      events: document.getElementById("events")?.textContent?.trim(),
      evalid: document.getElementById("evalid")?.textContent?.trim(),
      status: document.getElementById("status")?.textContent?.trim(),
      outcome: document.getElementById("outcomeline")?.textContent?.trim(),
    }));

    check(!!shown.evalid, `the host application rendered a verdict (evaluation ${shown.evalid || "none"})`);
    check(shown.decision && shown.decision !== "—", `the verdict is a decision, not the placeholder (${shown.decision})`);
    // The part the DOM cannot tell you. A demo wired to an unreachable
    // engine renders `allow` with mode=fail-open and looks like this one;
    // a demo wired to the COLLECTOR renders a correct verdict from the
    // wrong service, which is how a run once recorded `topology:
    // monolith` while measuring the composed deployment.
    const engineAfter = await counter(ENGINE, "http_requests_total", {
      route: "POST /v1/decisions",
      status: "200",
    });
    const collectorAfter = await counter(COLLECTOR, "http_requests_total", {
      route: "POST /v1/decisions",
      status: "200",
    });
    check(
      moved(engineBefore, engineAfter) >= 1,
      `the decision engine served it (${engineBefore ?? "absent"} -> ${engineAfter ?? "absent"}) — ` +
        "the rendered verdict alone cannot tell fail-open from a real answer",
    );
    check(
      moved(collectorBefore, collectorAfter) === 0,
      `and the collector did not (${collectorBefore ?? "absent"} -> ${collectorAfter ?? "absent"}) — ` +
        "it mounts the same endpoints, so a misdialled demo answers correctly from the wrong service",
    );
    // A page that rendered no verdict has no mode line, and "no mode
    // line does not contain fail-open" is true and worthless. The mode
    // must be there to be judged.
    check(
      !!shown.mode && !/fail-open/.test(shown.mode),
      `the verdict is not fail-open (${shown.mode || "no mode line rendered"})`,
    );

    // THE MODE CONTRACT (§3), which is a property of the RESPONSE SHAPE
    // and not of the verdict.
    //
    // In monitor mode `decision` is always allow and `shadow_decision`
    // carries what enforce would have returned. In enforce mode
    // `shadow_decision` is absent and `decision` is the real answer. The
    // contract says a client that reads only `decision` measures nothing
    // while the service is in monitor — so the difference between the
    // two shapes is the difference between a detector that is watching
    // and one that is acting, and nothing has ever driven it end to end.
    //
    // WHAT IS NOT ASSERTED: which verdict came back. That is detection,
    // it moves with calibration, and a gate that pins it becomes a
    // measurement that fails on a Tuesday. `make numbers` owns it.
    const got = seen.login;
    check(!!got, "the host application's own response was observed, not just its rendering");
    if (got) {
      const shadow = Object.prototype.hasOwnProperty.call(got, "shadow_decision");
      check(got.mode === MODE, `it answered in the mode the topology was started in (${got.mode} / expected ${MODE})`);
      if (MODE === "monitor") {
        check(got.decision === "allow",
          `monitor returns allow whatever it thinks (${got.decision}) — §3, and why reading only ` +
            "`decision` in monitor measures nothing");
        check(shadow && !!got.shadow_decision,
          `and carries what enforce would have said (${got.shadow_decision ?? "absent"})`);
      } else {
        check(!shadow,
          `enforce carries no shadow_decision (${shadow ? "present: " + got.shadow_decision : "absent"}) — ` +
            "its `decision` IS the answer");
        check(["allow", "challenge", "block"].includes(got.decision),
          `and its decision is a real verdict (${got.decision})`);
      }
    }

    link(6, "the snapshot — the verdict was derived from this session");
    // The one link that proves collector → stream → KV → engine. A cold
    // start returns a well-formed verdict from no evidence at all: the
    // engine has never heard of the token, so it answers from nothing
    // and the page shows events 0. That is indistinguishable from a
    // healthy allow unless someone reads the evidence.
    const events = Number(shown.events);
    check(
      Number.isFinite(events) && events > 0,
      `the engine judged evidence rather than a cold start (events=${shown.events}) — ` +
        "a token the engine never received scores identically and shows 0",
    );
    check(Number(shown.confidence) > 0, `and had some confidence in it (confidence=${shown.confidence})`);
    check(
      seen.failures.length === 0,
      "no request the SDK depends on failed silently" +
        (seen.failures.length ? ": " + seen.failures.join("; ") : ""),
    );
    if (seen.aborted.length) {
      console.log(
        `  note  ${seen.aborted.length} telemetry batch(es) were aborted by the browser ` +
          "(keepalive); link 8 is what decides whether anything was lost",
      );
    }

    /* ---- link 7: the loop closes ---- */
    link(7, "the loop closes — the application says what happened");
    // `POST /v1/outcomes` is one of the contract's four endpoints and,
    // until the demo host filed a label, no product code called it at
    // all: `kill-test` and `loss-audit` reached it and nothing else, so
    // the channel every future calibration depends on was exercised
    // solely by the gates that check it.
    //
    // The COUNTER is the assertion; the page's line is a cross-check.
    // Asking the page whether the label was filed and believing the page
    // is the shape this repository keeps finding — a guard that checks
    // what the claim already said. These are two independent sources and
    // they have to agree.
    // WHICH label, derived from the verdict rather than assumed. The
    // first version hardcoded login_success, which is only ever right in
    // monitor mode — where the verdict is always allow by §3. In enforce
    // a block files login_failure and a CHALLENGE FILES NOTHING, because
    // the application has not finished. Asserting the implication covers
    // all three without holding an opinion about which should happen,
    // which is the line ADR-0015 draws.
    const verdict = seen.login?.decision;
    const wanted = { allow: "login_success", block: "login_failure" }[verdict] ?? "";

    const outcomeAfter = await counter(ENGINE, "http_requests_total", {
      route: "POST /v1/outcomes",
      status: "202",
    });
    check(
      wanted ? moved(outcomeBefore, outcomeAfter) >= 1 : moved(outcomeBefore, outcomeAfter) === 0,
      wanted
        ? `the engine accepted a label for this evaluation ` +
          `(${outcomeBefore ?? "absent"} -> ${outcomeAfter ?? "absent"})`
        : `the engine was sent no label, which is what a '${verdict}' means ` +
          `(${outcomeBefore ?? "absent"} -> ${outcomeAfter ?? "absent"})`,
    );
    check(
      wanted
        ? new RegExp(`outcome filed: ${wanted}`).test(shown.outcome || "")
        : /no outcome filed/.test(shown.outcome || ""),
      wanted
        ? `and the page reports ${wanted}, the label a '${verdict}' implies ` +
          `(${shown.outcome || "no outcome line rendered"})`
        : `and files nothing for '${verdict}', which has not turned out yet ` +
          `(${shown.outcome || "no outcome line rendered"})`,
    );
  } finally {
    await context.close();
    await browser.close();
  }

  /* ---- link 8: the archive ---- */
  link(8, "the records reach the durable store");
  // Read from the archive's DURABLE position, never its process
  // counters: 3.4 established that counters reset on restart, so a
  // reconciliation built on them measures one process's lifetime.
  //
  // AN ACCOUNTING IDENTITY, not a floor. What the two services say they
  // ACCEPTED must equal what the archive says it made durable:
  //
  //   sessions + telemetry batches + evaluations + labels == commits
  //
  // Every accepted request produces exactly one record (loss-audit rests
  // on the same 1:1), so the sum is exact and an equality catches both
  // directions — a record lost, and a record that appeared from
  // somewhere this run cannot account for. The floor this replaces
  // caught only the first.
  //
  // COUNTED FROM THE SERVERS, never from the browser's view of its own
  // requests. Chromium aborts keepalive fetches it has already handed to
  // the network, so a batch the page never saw a response for may have
  // been accepted anyway; `seen.telemetry` is therefore a lower bound on
  // what the collector took, and building the expectation from it made
  // the floor the only safe assertion available.
  //
  // WHAT IT STILL DOES NOT PROVE, stated rather than implied: that THIS
  // session's records are the ones in there. The archive has no read
  // surface — the events table is keyed by content hash and carries no
  // session column — so this is arithmetic, not identity. In THIS gate
  // the distinction is nearly empty: the topology is brought up fresh in
  // a project of its own, nothing else sends anything to it, and passing
  // while losing this session's records would mean the archive committed
  // exactly as many records from a source that does not exist. Against a
  // shared archive it would be a real gap, which is the one the roadmap
  // keeps.
  const deadline = Date.now() + DRAIN_TIMEOUT_MS;
  let pending = null;
  let unaccounted = null;
  let committed = null;
  let expected = null;
  let unreachable = "";
  while (Date.now() < deadline) {
    // An archive that cannot be SCRAPED is the condition this link
    // measures, so it must not be an exception. The first version let
    // the fetch throw, which unwound past every check here — and the
    // summary then reported link 8 fine, because a link with no checks
    // scored as one with no failures. The outage became a silent pass in
    // the assertion written to catch outages.
    try {
      // Re-read the sources every pass, not once. A request in flight
      // when the browser closed can still be accepted, which moves the
      // expectation as well as the archive; freezing the expectation
      // first would turn that race into a permanent failure.
      const [s, t, d, o] = await Promise.all([
        counter(COLLECTOR, "http_requests_total", { route: "POST /v1/sessions", status: "200" }),
        counter(COLLECTOR, "http_requests_total", { route: "POST /v1/telemetry", status: "202" }),
        counter(ENGINE, "http_requests_total", { route: "POST /v1/decisions", status: "200" }),
        counter(ENGINE, "http_requests_total", { route: "POST /v1/outcomes", status: "202" }),
      ]);
      expected = (s ?? 0) + (t ?? 0) + (d ?? 0) + (o ?? 0);

      pending = await counter(ARCHIVE, "archive_stream_pending");
      unaccounted = await counter(ARCHIVE, "archive_position_unaccounted");
      committed = await counter(ARCHIVE, "archive_position_committed");
      unreachable = "";
    } catch (e) {
      unreachable = e.message;
      pending = unaccounted = committed = null;
    }
    if (pending === 0 && unaccounted === 0 && moved(archiveBefore, committed) === expected) break;
    await new Promise((r) => setTimeout(r, 1000));
  }
  check(
    committed !== null,
    "the archive publishes a durable position at all" +
      (unreachable ? ` — it did not answer at all: ${unreachable}` : ""),
  );
  check(
    moved(archiveBefore, committed) === expected,
    `it made durable exactly what the services accepted ` +
      `(${moved(archiveBefore, committed)} committed against ${expected} accepted: ` +
      `sessions + batches + evaluations + labels)`,
  );
  check(pending === 0, `the stream drained (pending=${pending ?? "absent"})`);
  check(unaccounted === 0, `and nothing is unaccounted (unaccounted=${unaccounted ?? "absent"})`);
}

/* ------------------------------------------------------------------ *
 * Verdict
 * ------------------------------------------------------------------ */

let crashed = null;
try {
  await run();
} catch (e) {
  crashed = e;
}

const failed = links.flatMap((l) => l.checks.filter((c) => !c.ok).map((c) => `link ${l.n}: ${c.what}`));

console.log("\n== the chain ==");
for (const l of links) {
  const bad = l.checks.filter((c) => !c.ok).length;
  // A link that ran NO checks is not a link that held. It reads `ok`
  // under `bad === 0` alone, and this file has now produced that bug
  // three times in three different shapes: `[].every()` twice, and a
  // link whose first metrics read threw before a single check ran, so
  // the summary reported the archive fine while the archive was down.
  // Same mistake each time — an absence scored as a pass.
  const state = l.checks.length === 0 ? "NOCHECK" : bad ? "BROKEN " : "  ok   ";
  console.log(`  ${state} link ${l.n} — ${l.name}`);
}
const silent = links.filter((l) => l.checks.length === 0);
if (silent.length) {
  console.log(
    `\n  ${silent.length} link(s) asserted nothing at all — a link that could not be ` +
      `observed is a failure, not a pass`,
  );
}
// A link that was never reached is reported as such rather than left
// off. Seven links minus the ones that ran is the number that matters
// when the run died halfway.
for (let n = links.length + 1; n <= 8; n++) console.log(`  ------  link ${n} — not reached`);

if (crashed) {
  console.log(`\n  the run did not finish: ${crashed.stack || crashed.message}`);
  process.exit(1);
}
if (silent.length) process.exit(1);
if (failed.length) {
  console.log(`\n  ${failed.length} link assertion(s) did not hold:`);
  for (const f of failed) console.log(`    - ${f}`);
  process.exit(1);
}
console.log("\n  the chain holds end to end: a real browser opened the page, the SDK");
console.log("  reached the collector, the engine judged the evidence it published,");
console.log("  and the archive accounted for every record.");
