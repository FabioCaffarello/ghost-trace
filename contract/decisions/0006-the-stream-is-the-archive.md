# 0006 — The stream is the archive

**Status:** accepted · **Date:** 2026-08-04 · **Milestone:** PR-2.5b
· **Supersedes:** [0003](0003-dual-write-during-the-split.md)

## Context

ADR-0003 made the local substrate authoritative during the split: every
record committed locally *and* mirrored onto the stream, with
publication failures counted and swallowed. It set its own condition for
removal — *"PR-2.5, when the archive service has demonstrably held
everything the collector wrote across a full experiment run"* — and
named the counters as the evidence.

That condition is met. The measurement is in
[`docs/results/parity-cutover-2026-08-04.md`](../../docs/results/parity-cutover-2026-08-04.md).

## Decision

**`-nats` and `-data` are alternatives, and passing both is an error.**

- With `-nats`, the stream is the archive: the collector publishes and
  the archive service stores. Publication failures are **returned**.
- With `-data`, a local substrate is the archive. This is what the
  all-in-one development binary uses, and what the six-numbers run
  needs — it has no broker and should not need one.

An error rather than a precedence rule, because a precedence rule is a
thing nobody remembers under pressure and a silently ignored `-data`
looks exactly like a working local archive.

## Why this is not simply a reversal

ADR-0003's argument was that adding a broker must not make the service
less reliable than it was without one, and that argument still holds —
it is *paid for* rather than refuted. What changed is that there is now
a second durable store which has been shown to hold everything, so the
local write is redundancy rather than insurance.

The reliability cost is real and is accepted here rather than hidden:
with the stream as the only archive, a broker outage makes
`/v1/outcomes` answer 503 where it previously succeeded. That is the
correct answer — a label the caller believes recorded but which is not
poisons calibration — and it is the price of one source of truth. The
alternative, keeping both, means keeping two stores that can disagree
and a parity check nobody ever retires.

## Consequences

- `internal/adapters/streamarchive` is gone, and with it `Counts()`.
  Those counters existed to earn this decision; keeping them after it
  would be keeping a measurement of something that no longer happens.
- The collector holds **no durable store** in the composed topology. Its
  compose service has no volume, and `gt-data` is retired.
- The decision path is unaffected either way: nothing reads the archive
  to judge a session. An archive-less run still serves sessions,
  telemetry and decisions; only `/v1/outcomes` requires durability.
- `make parity` stays. It tests the archive consumer — that a record
  reaching the stream lands in the substrate, byte for byte — which is
  now the *only* path and therefore matters more, not less.
- The rollback is `-data` on one binary. That is deliberate: PR-2.6 is a
  gate that may fail, and its ordered mitigation ends in going back to
  the all-in-one, which must not require re-implementing a store.
