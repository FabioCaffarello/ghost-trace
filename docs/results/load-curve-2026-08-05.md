# The floor becomes a curve — and the bend is not where it was predicted

**Date** 2026-08-05 · **Targets** `make load-sweep`, `go test
./internal/session -run Cores` · **Topology** compose `core` profile,
Docker Desktop on macOS, 10 cores · **Driver** `tools/loadgen`,
open-loop

> **Correction, 2026-08-10 (PR-5.0c).** The driver behind this run sent
> pointer events as `x`/`y` — fields the wire does not have — so the
> collector dropped them and did **zero pointer-feature work** for half
> of every batch. The shape of this curve and its comparisons stand;
> the absolute throughput figures describe a system doing less
> per-batch work than production does, and read **high**. The driver
> now builds batches from `libs/wire` with real `pts` polylines;
> re-measurement is scheduled as Phase 5.6.

The first measurement of this system under load. It confirms the
pre-registered hypothesis in its mechanism, **refutes it in its
significance**, and finds the real constraint somewhere else entirely.

## What was predicted, in writing, before it was measured

`session.Store` has said this in its own comment since M1:

> One mutex guards the whole map. At M1 volume that is correct and
> boring; **sharding it before there is a measurement showing contention
> would be optimising against a guess.**

[`contract/roadmap.md`](../../contract/roadmap.md) §4.2 sharpened it
before any number existed: `Store.With` holds that process-global mutex
across its whole callback, and the callback performs the full per-event
feature update for every event in a batch — so **telemetry throughput
should barely improve with cores**.

## The mechanism: confirmed, and worse than predicted

| workers | batches/s |
| --- | --- |
| 1 | 2 879 080 |
| 8 | 2 018 435 |

**Speedup 0.70× on ten cores.** Not "barely improves" — adding seven
workers made it *slower*. Contention costs more than the parallelism
buys, which is the classic signature of a lock held long enough to
matter.

Held for what, specifically:

| callback | ops/s |
| --- | --- |
| map lookup only | 14 935 043 |
| lookup + 8 events | 2 858 297 |

So roughly **four fifths of the critical section is feature work**, not
the map lookup a mutex on a map would suggest. And the consequence
follows directly: a 16× larger batch costs 9.3× the batch throughput
(3 178 773 → 341 820 batches/s). **Batch size is a lever any one client
can pull on every other client's latency.**

## The significance: refuted

2.9 million batches per second is four orders of magnitude above
anything the HTTP path delivers. The mutex is real, it does not scale,
and **it is not the constraint**.

The roadmap said, before the measurement: *"If the bend is somewhere
else, that is the more interesting result and it gets written down as
such."* It is, and this is that.

## Where the collector actually bends

Offered rate is *sessions* per second; each is two HTTP requests
(`/v1/sessions` then `/v1/telemetry`, 8 events).

| offered | achieved | svc p50 | svc p99 | rsp p99 | deficit p99 | dropped |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 100 | 100 | 1.40 | 6.76 | 7.76 | 1.05 | 0 |
| 250 | 250 | 0.91 | 10.92 | 13.37 | 0.59 | 0 |
| 500 | 500 | 0.78 | 5.41 | 5.66 | 0.26 | 0 |
| 1 000 | 1 000 | 0.62 | 10.93 | 11.20 | 0.17 | 0 |
| 2 000 | 2 000 | 1.00 | 15.20 | 15.49 | 0.32 | 0 |
| 4 000 | 4 000 | 1.76 | 14.33 | 14.35 | 0.11 | 0 |
| 6 000 | 5 999 | 2.63 | 22.96 | 23.18 | 0.13 | 0 |
| 8 000 | 7 994 | 3.83 | **132.42** | 132.46 | 0.15 | 0 |

All milliseconds. **The bend is between 6 000 and 8 000 sessions/s** —
12 000 to 16 000 HTTP requests per second — where p99 goes from 23ms to
132ms while the median barely moves. Nothing was dropped at any rate.

`deficit p99` stays under 1.1ms throughout, so the driver was never the
bottleneck and none of these rows is a measurement of `-workers`.

## The real constraint is downstream

At 2 000 sessions/s the collector was completely unbothered — and the
archive was **33 965 records behind**. At 8 000 sessions/s it reached
**300 716**.

Measured with no traffic at all, over a clean 20-second window:

> **The archive commits 1 356 records per second.**

The collector at its own bend publishes roughly **16 000 records per
second** into the stream. That is a **~12× mismatch**, and it is the
number this phase was actually looking for.

### There is no backpressure

The collector will accept 8 000 sessions/s indefinitely while the
archive falls further behind every second. Nothing in the request path
knows or cares. The stream absorbs the difference until the retention
window does not, at which point records age out — and Phase 3's
`stream_skipped` reports it **after** the fact.

That is not a bug in either service; it is what "the archive is not on
any request path" (ADR-0006) buys and costs. It has simply never been
quantified before.

## Response ≈ Service, and that is a result too

The open-loop driver exists because a closed-loop one hides queueing.
Across this entire sweep, response and service times agree within about
1ms until the very last row. **This system, at these rates, does not
exhibit the pathology the driver was built to catch.**

That is worth stating plainly: the instrument was necessary to establish
the *absence* of the effect. Had the sweep been run with a closed-loop
generator, the numbers would have looked similar and nobody could have
known whether that was the system or the instrument.

## The accounting held throughout

After ~74 000 sessions across two sweeps:

```
position_committed    462 429
position_rows         462 429
position_rejected           0
position_unaccounted        0
stream_skipped              0
collector drops             0
```

Committed equals rows exactly — no redelivery was needed. The first half
of the phase gate (*the accounting still balances under sustained
concurrency*) is met at every rate measured.

## What this does not say

- **Docker Desktop on macOS is a virtual machine.** Every absolute
  figure here is depressed by a hypervisor and a virtualised network
  stack. **The ratios are the result; the magnitudes are not.**
- **Ten seconds per step.** Long enough to fill a queue, far too short
  for anything thermal, for GC steady state, or for a leak.
- **One machine runs the driver, all four services and the broker.**
  Above ~6 000 sessions/s the driver is competing with the system under
  test for the same cores, and some of the bend at 8 000 may be that.
  The deficit figure says the driver kept its schedule; it does not say
  it stole nothing.
- **Sessions are created and never revisited.** A real population
  revisits, so `Store.With` here contends less on individual entries and
  more on the map than production would.
- **Nothing about memory.** `bench-architecture` covers the memory
  claim; this run does not touch it.

## What follows

**4.3 was deliberately left unspecified** until this existed, and it
should now be re-scoped: sharding `session.Store` would fix a mechanism
that is not the constraint. The candidate that matches the measurement
is the archive's commit rate.

A hypothesis for 4.4, on record before that measurement: the substrate
opens with `PRAGMA synchronous=FULL` and serialises every write through
one `writeMu`, so 1 356 commits/s is close to what one fsync per commit
on this hardware would allow. If that is right, the lever is batching
commits into one transaction — and the cost is a clearly-stated
weakening of the durability promise, which is an ADR, not a patch.
