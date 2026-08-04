# 0003 — The local substrate stays authoritative during the split

**Status:** accepted · **Date:** 2026-08-04 · **Milestone:** PR-2.2b

## Context

Phase 2 moves archival out of the collector and into a separate service
fed by NATS JetStream. During the transition both paths run: every
record is committed to the local substrate *and* mirrored onto the
stream.

The question is what happens when the broker is unavailable.

## Decision

**The local write is authoritative. Publication failures are counted
and logged, never returned to the caller.**

Concretely, in `streamarchive.Append`: the local append runs first and
its error is the caller's error; the mirror runs second and its error
is recorded in a counter and a log line, and then swallowed.

## Why

Adding a broker must not make the service less reliable than it was
without one. `/v1/outcomes` is the labels channel every future
calibration depends on, and it is the one endpoint that surfaces
archive failures as 503 — so propagating a NATS outage there would
convert a broker restart into rejected outcome reports. That is a
regression bought with an architecture diagram.

The reverse asymmetry matters too: a record that failed to land locally
is **not** mirrored. Publishing it would put the stream ahead of the
disk and make the parity check meaningless in the direction that
matters.

## Consequences

- `Counts()` reports appended / published / dropped. `published +
  dropped == appended` is an invariant with a test; a gap means a record
  went to disk and neither reached the stream nor was counted as lost,
  which is the one outcome this adapter must not produce quietly.
- Those counters are the evidence PR-2.5 needs. Removing the local write
  is earned by parity, not scheduled.
- At-least-once delivery plus content-addressed idempotency is the pair
  that survives a crash between commit and ack. The consumer acks only
  after a successful commit; a redelivered record collapses in the
  substrate.
- A record whose payload does not match its hash is **dropped, not
  retried**. Redelivery brings the same bytes, and committing them would
  put a payload under a hash that does not describe it — every later
  verification would fail on it.

## Revisit

PR-2.5, when the archive service has demonstrably held everything the
collector wrote across a full experiment run. Until then, `make parity`
is the check and `docs/results/` is where the evidence for the cutover
will live.
