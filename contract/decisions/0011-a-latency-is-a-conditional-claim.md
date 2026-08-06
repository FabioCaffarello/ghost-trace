# 0011 — a published latency is a conditional claim

**Status:** accepted · **Date:** 2026-08-06 · **Milestone:** PR-4.8

## Context

For three phases this repository published latency numbers taken on an
idle machine against one session. It said so — the schema calls number 3
a floor, and
[`latency-gate-2026-08-04.md`](../../docs/results/latency-gate-2026-08-04.md)
repeats it in bold. Nobody was misled on purpose.

But **the condition was prose and the number was data**, and only one of
those travels. A manifest carried `topology` because a decision answered
in-process is not the same measurement as one crossing a network hop.
It carried no equivalent for load, so an idle baseline and a run taken
under concurrency could be compared and nothing would object.

Phase 4 measured the difference: **the same decision p99, from the same
code, is 4.7× higher at 10 000 decisions/s** than it is idle. Both
readings are true.

## Decision

**A published latency claims a bound under a stated condition, not a
value.** Three consequences follow, and they are the decision:

### 1. The budget is the claim. The floor is not.

Number 3's claim is *"a decision lands inside 80ms"*. It is not *"a
decision takes 1.393ms"* — that figure describes the friendliest
possible circumstance and was always labelled as such.

`make load-gate` therefore asserts the **budget** under load. A gate
requiring the idle figure to be reproduced under concurrency would fail
for the wrong reason and be deleted for it.

### 2. The condition is recorded, not described.

`provenance.run.load` sits beside `provenance.run.topology`, and
`numbers_check` treats them identically: it refuses to compare across
conditions, and its baseline picker filters on both. A manifest with no
`load` field reads as `idle`, which is what every run before Phase 4
was.

A picker that ignored the condition would hand a loaded run an idle
baseline, and the check would then report the picker's mistake as the
run's — which is why the filter and the check have to move together.

### 3. Where a number bends is part of what it claims.

An unqualified latency invites the reader to assume it holds at any
rate. It does not, and the bend is now published rather than implied:

| | holds to | at |
| --- | ---: | ---: |
| decision path | ~10 000 decisions/s | p99 6.57ms |
| collector ingest | ~6 000–8 000 sessions/s | p99 23ms at 6 000 |
| archive commit | **4 133 records/s** | — |

## Why not simply re-measure everything under load and publish that

Because it would replace one unqualified number with another. "p99
6.57ms" is exactly as incomplete as "p99 1.393ms" without the rate it
was taken at, and a reader would have no more reason to trust it.

The floor also remains worth publishing. It is the cheapest number to
reproduce, it is stable across machines in a way loaded figures are not,
and a regression in it is a real signal. What changed is that it can no
longer be silently compared against something it is not.

## Consequences

- **Two conditions now exist in the manifest vocabulary**, `topology`
  and `load`, and the pattern is available for a third if one is ever
  needed. Each costs one field, one check and one line in the baseline
  filter.
- **`make load-gate` is the phase gate and is outside `make ci`.** It
  starts and stops containers and takes minutes. It refuses when the
  topology is down.
- **The timing measurements moved to `make measure`.** Learned twice in
  one pull request: a timing assertion on shared CI hardware encodes
  that hardware. One runner measured an inlined commit at 48/s against
  18 244/s locally. Numbers are not gates; the properties they describe
  are guarded structurally and still run in CI.
- **`make numbers` still compares the floor to the floor**, because
  both sides of that comparison are now labelled.

## What this does not fix

**There is still no backpressure.** The collector absorbs ~16 000
records/s at its bend and the archive commits 4 133/s. Nothing in the
request path knows or cares; the stream absorbs the difference until
retention does not, and Phase 3's `stream_skipped` reports it after the
fact. Recording the condition of a measurement does not change the
system being measured, and this ADR should not be read as if it had.

That mismatch is the largest thing Phase 4 leaves open.
