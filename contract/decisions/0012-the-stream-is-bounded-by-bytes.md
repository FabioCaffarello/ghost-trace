# 0012 — the stream is bounded by bytes, and the collector owns the bound

**Status:** accepted · **Date:** 2026-08-11 · **Milestone:** PR-5.3

## Context

Phase 4 measured the mismatch this phase exists to survive: **the
archive commits about 4 133 records/s against a collector that bends
near 16 000**, with nothing between them slowing the producer down. The
stream absorbs the difference.

Until now it absorbed it into a stream bounded only by
`MaxAge: 7 days`, on `FileStorage`, with no `MaxBytes`, no `MaxMsgs`,
and no resource limit on the container holding the volume.

Age is not a bound on anything an operator controls. At the measured
surplus of roughly 11 900 records/s the volume fills in **hours**, not
seven days — and when the disk fills, NATS dies. Compose makes the
collector `depends_on` a healthy broker, so the collector dies with it.

**The system's worst failure mode was a disk-full that took down
ingestion, not a data loss.** That is a strictly worse outcome than the
one the design already accounts for, and it arrives without warning
because nothing was measuring the thing that would fill.

A second problem sat beside it. `EnsureStream` used
`CreateOrUpdateStream` and **all three services called it**, so
whichever started last silently rewrote the limits. That was tolerable
while the only limit was a constant every binary shared. It stops being
tolerable the moment the limits encode a policy — how much backlog the
archive is allowed to accumulate — because a policy that depends on
container start order is not a policy.

## Decision

### 1. The stream is bounded by bytes as well as by age

`MaxBytes` is set (4 GiB) beside `MaxAge`. With `Discard: DiscardOld`,
exceeding it drops the oldest unread records — which is a loss this
system already knows how to describe: `stream_skipped` counts records
that left the stream above the archive's durable mark, the reconciliation
subtracts them, and PR-5.2's `ArchiveFallingBehind` alert fires long
before either.

**We prefer bounded, counted, survivable loss to unbounded, silent,
fatal loss.** That is the whole decision. Both outcomes lose records
under sustained overload; only one of them leaves a running system and
a number to read afterwards.

**The requirement is the decision; the number follows from it.** The
requirement is that the cap hold more than an hour of backlog at the
archive's measured drain rate — long enough that a restart, a redeploy,
or an outage somebody is actively fixing costs no records. At 4 133
records/s and a conservative 256 bytes each (real ones measured 60–161),
that is **4 GiB, about 68 minutes**.

This was written down as 2 GiB first, picked by taste, and the test
below it rejected that at 34 minutes — under the hour the ADR's own
prose was claiming. Recorded because it is the argument for deriving
the number rather than choosing it: a deployment with a different disk
or a different drain rate should recompute, not copy.

### 2. The collector owns the declaration; readers bind and refuse

`EnsureStream` is the producer's call. `OpenStream` binds to the
existing stream and returns `ErrStreamLimitsMismatch` if `MaxAge` or
`MaxBytes` differ from what the calling process was built with. The
decision engine and the archive call `OpenStream`.

This is exactly the rule [ADR-0004](0004-session-snapshots-carry-feature-state.md)'s
snapshot bucket already follows, and PR-3.5 already paid for learning:
`EnsureSessions` / `OpenSessions` exist because two services calling
CreateOrUpdate meant the last one to start rewrote the other's TTL.

The cost is ordering: the archive now `depends_on` a healthy collector
in compose, where before either could start first. That is a real
constraint and it is the right one — a consumer that declared its own
retention would be choosing the backlog bound it is then measured
against.

### 3. Containers have ceilings

`mem_limit` and `pids_limit` on every service, for the same reason as
§1: the failure was not a bad number, it was an unbounded one. A
collector holding sessions in a map and a broker spooling to disk both
grow until something outside them says stop.

Development ceilings, not capacity planning — high enough that the load
gate's 2 000 sessions/s does not touch them, low enough that a runaway
takes its own container down rather than the host.

## Consequences

- A sustained overload now **ages records out of the stream sooner**
  than it used to. That is the intended trade and it is visible:
  `stream_skipped` rises, `unaccounted` stays zero because the
  subtraction accounts for it, and the alert fires at half the window.
- **The archive cannot start before the collector.** A reader that
  starts first fails with a message that says why. In compose that is
  expressed as a dependency; anywhere else it is an operator's problem
  to order, and the error text says so.
- **`archive_stream_max_age_seconds` now describes a limit somebody
  chose**, which matters because PR-5.2's alert rules divide by it.
- This does **not** solve the mismatch. The archive is still four times
  slower than the collector's bend and there is still no backpressure;
  5.4 (batching) and 5.5 (an admission signal) are what address that.
  This decision only ensures that until they land, the overload is
  survivable and legible instead of terminal.

## Alternatives considered

**Leave it unbounded and rely on monitoring.** The alert added in
PR-5.2 would fire, and an operator who is awake could act. But the
failure it precedes is the loss of the whole topology, and a bound that
depends on someone reading a dashboard in time is not a bound.

**`MaxMsgs` instead of `MaxBytes`.** Records vary in size only slightly
today (60–161 bytes), so the two are nearly interchangeable — but the
resource actually at risk is disk, and a message cap stops describing
it the moment a payload grows. Bound the thing that runs out.

**`Discard: DiscardNew` — refuse the publish instead of dropping the
oldest.** This is genuinely tempting, because it turns silent loss into
backpressure: the publish fails, the collector counts a drop by reason,
and the producer learns. It is rejected *here* and deliberately left to
5.5: applied now it would make the collector's ingest fail as soon as
the archive fell behind, which is a fail-closed answer to a problem
this project has decided to fail open on (§5). The admission signal in
5.5 is the same idea with a decision attached rather than a cliff.
