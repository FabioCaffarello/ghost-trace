# The phase gate: p99 in the composed topology — 2026-08-04

Phase 2 was allowed to split the binary on one condition: that a
decision still lands inside the 80ms budget once it has to cross a
network hop and a KV read. This is that measurement.

**Result: p99 1.4ms against an 80ms budget. The gate passes, with about
fifty times the headroom it needed.**

## What was compared

Four configurations, because "the composed topology is faster" would
have been a wrong reading of the first two numbers, and it took the
middle row to see why.

| # | topology | archive on the decision path | p50 | p95 | **p99** | max |
| --- | --- | --- | ---: | ---: | ---: | ---: |
| A | monolith | local substrate | 4.994 | 6.182 | **6.765** | 12.283 |
| A′ | monolith | event stream | 0.376 | 0.484 | **0.776** | 1.042 |
| B | composed | event stream | 0.441 | 0.854 | **1.719** | 5.208 |
| C | composed, in compose | event stream | 0.394 | 0.861 | **1.393** | 1.954 |

All milliseconds, n=500 decisions on one session, idle system.
A, A′ and B are local processes with NATS in Docker; C is the deployed
compose topology measured from the host.

## The Δ that the gate is about

**A′ → B is the split: p99 +0.94ms.** Same archive backend, same
machine; the only difference is that the decision crosses a network hop
and reads the session's snapshot out of the KV store instead of holding
it in memory. That is what the phase was spending, and it is what the
budget had to absorb.

**A → A′ is not the split at all: p99 −5.99ms.** It is the archive
backend changing under ADR-0006. Committing an `Evaluation` to a local
SQLite substrate on the decision path cost about six milliseconds;
publishing it to the stream costs well under one.

Reading the published baseline (A, 5.5–6.8ms) against the composed run
(C, 1.4ms) and concluding that decomposition made decisions four times
faster would be wrong. Decomposition cost about a millisecond. Retiring
the synchronous local write paid for it several times over, and the two
changes happened in adjacent pull requests.

## What the mitigation ladder was for

The plan named an ordered response if this failed — keep-alive/pooling,
then a very short KV cache, then colocation, then rollback to the
all-in-one. **None was needed and none was applied.** They are recorded
here as unspent, not as tried and unnecessary: nothing in this
measurement says whether they would have worked.

## What this does not show

- **It is a floor, not a production figure.** The schema says so about
  number 3 already: one session, idle system, the friendliest possible
  conditions. Nothing here measures the composed topology under load,
  and the split's cost is precisely the kind that grows with contention
  — a KV read is a network round trip, and round trips queue.
- **One machine, one run each.** macOS/arm64, 10 cores, everything on
  loopback or a Docker bridge. A real deployment puts the hop across a
  network that is not a memory copy.
- **Nothing about a cold snapshot.** Every measured decision followed 20
  telemetry batches, so the snapshot was current and present. A decision
  arriving before the collector has published anything reads as a cold
  start; that is correct behaviour and a different measurement.
- **Nothing about the archive falling behind.** The stream absorbs
  publishes; whether the archive service keeps up under load is
  `make parity`'s question, not this one.

## The full composed run

`docs/results/numbers-2026-08-04-d5f66dd9.json` is a complete six-number
run against the composed topology, published as the baseline that future
composed runs are checked against. All six tiers ran, all at 100%; the
architecture benchmark (number 6) is an in-process measurement and is
unchanged by topology, as the plan expected.

`numbers-check` refuses to compare a composed run against a monolith
baseline, and picks the newest manifest **of the same topology** rather
than the newest overall. Both are guarded in its selftest: the first
time the topology guard ran for real, it caught this run being judged
against a monolith baseline.
