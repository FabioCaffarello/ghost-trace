# The published latency was eight days stale, and the check that says so was built five days before it ran

**Date** 2026-08-12 · **Target** `make numbers` · **Topology** local,
monolith mode · **Archive** substrate · **Load** idle

Run to discharge the repository's own rule — `make numbers` after any
change to `services/` — for the demo host's new outcome call. It found
something older than that change.

## What reproduced, and what did not

```
numbers-check: against docs/results/numbers-2026-08-04-0b0af2d1.json

  tier1_playwright_naive             12/12 -> 12/12
  tier2_puppeteer_stealth            12/12 -> 12/12
  tier3_undetected_chromedriver      8/8 -> 8/8
  tier4_synthetic_linear             100/100 -> 100/100
  tier5_humanised_bow1.0             10/10 -> 10/10
  tier6_value_injection              10/10 -> 10/10
  cold start never blocks            True

  FAIL: latency p50: 0.435ms against 2.189ms in the baseline — 80% apart
  FAIL: latency p95: 0.524ms against 2.725ms in the baseline — 81% apart
  FAIL: latency p99: 0.834ms against 5.525ms in the baseline — 85% apart
```

Every detection rate held exactly. The latency had moved by 5×, and
**downward**, which is the direction that most deserves suspicion: a
number that improves can mean the system stopped doing the work.

## It was not this change

The first thing to rule out was the change that prompted the run. At
`ee5f810` — the commit before any of this session's work — the same
measurement gives `p50 0.428 / p95 0.506 / p99 1.035`. Three independent
runs across two commits land at 0.80, 0.83 and 1.04ms. Nothing recent
moved it.

## What actually happened

The first guess was PR-5.4's batch commits, landed 2026-08-11, which
amortise one fsync across a batch and were measured at 3.2×. The
arithmetic fits, which is exactly why it was worth checking rather than
publishing.

It is wrong, and the correction is the point of writing the guess down.
[The Phase 4 run](numbers-after-phase-4-2026-08-06.md)
already printed **`p99 0.739ms`** on 2026-08-06 — five days before
batching existed — and reported that the six numbers reproduced. It was
telling the truth: `numbers_check` gained the ability to reject numbers
3 and 4 in **PR-5.0a on 2026-08-11** (#328). Before that, latency was
printed and never compared.

So the timeline is:

| when | what |
| --- | --- |
| 2026-08-04 | monolith baseline published at p99 **5.525ms** |
| ≤ 2026-08-06 | latency is already ~0.74ms |
| 2026-08-06 | the Phase 4 run prints 0.739ms and passes, because nothing compares latency |
| 2026-08-11 | PR-5.0a teaches the checker to reject numbers 3 and 4 |
| 2026-08-12 | the checker is run for the first time since, and rejects |

What moved it is **not established here**, and replacing one confident
guess with another would be the same mistake twice. The window is Phase
4's storage rewrite: ADR-0009 put small payloads in the `events` row and
took one of a commit's two fsyncs off the path, the Phase 4 write-up
says "the storage layer under every archived record was rewritten", and
both the timing and the mechanism fit. It was not bisected. Establishing
it costs one more eight-minute run per candidate commit, and nothing
downstream depends on the answer.

**The baseline the new check was built to defend was already stale on
the day it was built.** The gap is not that the check was wrong; it is
that adding a check and running it are two acts, and only the first one
happened.

## Why nothing caught it

`make numbers` is a gate, is in `sensors.json`, and is deliberately
outside `make ci` — seven minutes and real browsers. Nothing forces it.
Phases 5 and 6 both changed `services/` and neither ran it. That
deliberate exclusion is defensible on cost and it has a cost of its own,
now measured: **eight days, and a published figure 5× off.**

This is the repository's own recurring shape from Phase 6, one step
sideways: there, guards checked what their authors' claims already said.
Here the guard was correct, arrived late, and was never pointed at the
record it existed to check.

## What was done

- A fresh manifest, [`numbers-2026-08-12-8b5e5c96.json`](numbers-2026-08-12-8b5e5c96.json),
  published from a clean tree at `8b5e5c9`. `make numbers-check` passes
  against it.
- [`the-split.md`](../the-split.md) cited `5.525ms` as a current fact in
  the paragraph explaining why that figure would be read backwards. It
  now carries the correction, and keeps the **+0.94ms** split cost, which
  was measured stream-against-stream and is untouched by the substrate
  path that changed.

## What was not done

**The composed baseline was not re-measured.** `numbers-2026-08-04-d5f66dd9.json`
(p99 1.515ms, topology `composed`) is from the same day and probably
carries the same staleness, but a composed run needs the topology up and
its own seven minutes, and guessing at it here would put a number on
record that nobody measured. `numbers_check` refuses to compare across
topologies, so nothing silently reads this monolith manifest as that
one.

**No mechanism was added to stop this recurring.** The obvious one — fail
when the newest manifest predates the last commit touching `services/` —
is a design question rather than a patch, and inventing it inside a
write-up is how a gate gets designed by accident.
