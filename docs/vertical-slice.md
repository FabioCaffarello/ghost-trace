# The smallest thing that can score a session

M1. Demo page → SDK → ingest → decision → visible result, with one
feature and one reason code.

Written during the work, not after. The parts worth reading are the two
things that went wrong.

---

## What it does

Open `http://127.0.0.1:8080`, move the pointer, sign in, see a score.
Press **Simulate linear bot** first and the score changes.

```
make run          # services/ingestion
```

Five files carry the whole thing: a feature extractor, a policy, a
session store, three HTTP handlers, and a page.

```
                browser                    │            server
  ┌──────────────────────────────┐         │
  │ demo page + SDK              │         │
  │  pointermove → decimate 20Hz │         │
  └──────────────┬───────────────┘         │
                 │  POST /v1/sessions      │   issue token, return collect policy
                 │  POST /v1/telemetry ────┼─→ feature.Pointer  (running accumulator)
                 │                         │            │
  ┌──────────────┴───────────────┐         │            ▼
  │ host app (demo backend)      │         │      policy.Judge → (score, confidence)
  │  POST /v1/decisions ─────────┼─────────┼─→    policy.Apply → decision
  │  ← 250ms timeout, fail-open  │         │            │
  └──────────────────────────────┘         │            ▼
                                           │      Evaluation → archive
```

The demo backend calls `/v1/decisions` over real HTTP with the
`secret_key`, rather than reaching into the policy package in-process.
That is the point of it: it exercises the actual trust boundary and the
actual failure path instead of a shortcut that would demonstrate
neither.

---

## Measured, on a loopback interface

```
                 score   confidence   shadow      events
  linear bot     1.000       1.000    block          360
  human-like     0.000       1.000    allow          360
  cold start     0.000       0.000    allow            0

  POST /v1/decisions, 300 calls, includes the archive write
    p50 5.04ms   p95 6.18ms   p99 6.38ms   max 6.98ms
```

**Read the separation with suspicion.** 1.000 against 0.000 is not a
detector working; it is a synthetic sine-curve "human" being trivially
distinguishable from a synthetic straight line. My fake human is far
more curved than a real reach. Real pointer paths will land much closer
to the bot, and the interesting question — how much closer — is exactly
what M2 exists to answer. No claim is being made here.

**p99 6.38ms against an 80ms budget.** This confirms the argument in
[`architecture.md`](architecture.md) §8.1 rather than undermining it: per-request latency was
never the hard part. An 8-second session is a few hundred points and
scoring it is microseconds. What costs is holding a hundred thousand of
them at once, over long sessions, which is why M5's sixth number is a
concurrency × duration grid and not a single latency figure.

---

## Two things that went wrong

### 1. The wire test caught a real bug the unit tests could not

`feature.Pointer.Add` originally took just a slice of points and treated
consecutive calls as continuous motion. Every unit test passed.

Then the first end-to-end test — real HTTP, real JSON, several batches —
scored a straight line at **zero**.

The SDK cuts a polyline at every batch boundary, so batch two started at
x=100 while batch one ended at x=690. With no absolute timing, that
looked like a 590px instantaneous jump backwards: the path doubled back
on itself, net displacement collapsed, and a perfectly linear bot
measured as maximally human. The feature was inverted under precisely
the conditions it exists to detect.

The fix is that `Add` now takes the polyline's session-relative start
time, so the gap between polylines comes from elapsed time rather than
from a buffer boundary that means nothing on its own.

This is v1's lesson arriving on schedule. A JSON field-name mismatch
broke v1's provenance chain in production while every in-memory test
stayed green (`docs/v1-retrospective.md`), and the response was to test
across the wire. Two milestones later that discipline caught an
inverted feature before anyone saw a number from it. The unit tests were
not wrong — they were testing one polyline, and the bug only exists at
the seam between two.

### 2. Deleting all of `.github` deadlocked the merge

M0 removed the entire `.github` directory, but branch protection
required four status checks — three of which were the constitutional
workflows being deleted. They could never run, so they could never pass.

Restoring the archived Go workflow verbatim would have been worse than
useless: every step was conditional on a `hashFiles` probe for a
Makefile that M0 had also deleted, so all steps would skip and the
required check would report **green without testing anything**. A
vacuous green on a required check is strictly more dangerous than a red
one.

---

## What is deliberately missing

- **Five of six event types.** Only `pointer`. Keys, scroll, focus,
  visibility and form are M3. A second feature added before there is an
  adversary to measure the first one against is unfalsifiable, and
  adding features is the most tempting way to avoid finding out that the
  first one does not work.
- **`POST /v1/outcomes`.** M4. A label channel with nothing durable
  behind it stores labels that cannot be joined to anything.
- **Durable session state.** The map dies with the process. M4.
- **Calibrated thresholds.** Every constant in `policy` is an inception
  guess. They are marked as such in the source, and M2 replaces them.
- **Transport, OLAP store, tenancy beyond one hardcoded tenant.** M4,
  and only because M5 needs them — not before.

The raw event archive *is* here, ahead of where the plan put it, for one
reason: M2 has to re-score recorded sessions when a threshold moves, and
you cannot re-recruit twenty people every time a constant changes.

---

## Privacy, as built

- No key events are collected at all in M1. When they arrive in M3 it is
  timing and coarse class only, never content.
- No canvas, WebGL, font enumeration or audio fingerprinting. Those
  identify the browser rather than the behaviour and contradict the
  thesis.
- Nothing persistent is written to the client. The session token lives
  in a closure and dies with the page.
- The token and the archived session id are **different values**. The
  token is a bearer credential with a 30-minute life; the archive keeps
  records for 7 days, so storing the token would put a live credential
  into storage that outlives it.
- All timestamps are session-relative. Client wall-clock is untrustworthy
  and leaks more than it gives.

---

## Next

M2: four adversarial tiers, the human capture arms, and the first
numbers that can be wrong. Recruitment opens now — the page above is the
capture instrument, and human capture is calendar-bound rather than
effort-bound, so it is the one item that more hours cannot accelerate.
