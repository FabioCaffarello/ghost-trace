# Measuring under load

*Phase 4, eight pull requests, 2026-08-05 to 2026-08-06. A
pre-registered hypothesis that was right about the mechanism and wrong
about the constraint, a loss figure that could go negative, and one
lesson learned twice.*

---

Phase 3 made three loss paths countable. Phase 4 asked the question
that only became answerable afterwards: **do the numbers survive
contention?**

Every latency this repository published was one session on an idle
machine. The schema said so — it calls number 3 a floor — but Phase 2
had measured a split whose cost is exactly the kind that grows with
contention, and nobody had checked. A load test before Phase 3 would
have reported throughput while dropping records: the wrong thing,
measured confidently.

## The instrument had to come first

The obvious load generator is a loop — send, wait, send again — and it
has a property nobody wants and almost nobody notices. When the server
slows, the generator slows with it. Requests that would have arrived
during a stall are never sent, so they never appear in the percentiles,
and **the report gets better the worse the server behaves.**

`tools/loadgen` schedules arrivals in advance and measures each
request's latency from when it was *due*. The difference, against a
server holding a global lock for 200ms every twenty requests:

```
open-loop:   n=400  p50 1017.0ms  slow 95%   deficit_p99 0.7ms
closed-loop: n=200  p50    0.1ms  slow  5%
```

Same server, same two seconds. The closed-loop generator reports a
median of 0.1ms; the truth is 1017ms.

**The discriminating statistic is the median, not the tail** — a
closed-loop generator waits the stall out and produces a tail that reads
as a healthy server with occasional hiccups. And the stall has to be
*global*: individually-slow requests produce no coordinated omission at
all, because nothing queues.

The first version of that test passed for the wrong reason. With one
worker, the driver itself was the bottleneck — `deficit_p99` of 2033ms
against my own comment saying such a run "measured this number, not the
server". Rewritten with 256 workers, and it now fails if the deficit
exceeds 100ms.

## The hypothesis was already on record, and not ours

`session.Store` has said this since M1:

> One mutex guards the whole map. At M1 volume that is correct and
> boring; **sharding it before there is a measurement showing contention
> would be optimising against a guess.**

That measurement:

| workers | batches/s |
| --- | ---: |
| 1 | 2 879 080 |
| 8 | 2 018 435 |

**Speedup 0.70× on ten cores** — adding workers makes it *slower*, the
signature of a lock held long enough to matter. Four fifths of the
critical section is feature work rather than the map lookup, and a 16×
larger batch costs 9.3× the throughput, so batch size is a lever any one
client can pull on every other client's latency.

**And it does not matter.** 2.9M batches/s is four orders of magnitude
above what the HTTP path delivers. The mechanism was confirmed; the
significance was refuted.

The roadmap had said, before any of it was measured: *"If the bend is
somewhere else, that is the more interesting result and it gets written
down as such."*

## The bend was downstream

| | bends at |
| --- | ---: |
| decision path | ~10 000 decisions/s, p99 6.57ms |
| collector ingest | ~6 000–8 000 sessions/s |
| **archive commit** | **1 356 records/s** |

The collector at its bend publishes ~16 000 records/s into a stream the
archive drains at 1 356. A **12× mismatch**, and nothing anywhere knows:
the collector accepts indefinitely while the stream absorbs the
difference, until retention does not.

## The fix that did not need the trade

The roadmap's diagnosis named **one** fsync — SQLite's, under
`synchronous=FULL`. Reading the write path afterwards showed **two**:
the blob is written to a temp file, `tmp.Sync()`ed and renamed *before
SQLite is touched at all*.

Every lever on the table then bought speed by weakening durability, and
the decision belonged in an ADR. Then the payloads were measured:
**every record in a real run is 60 to 161 bytes**, against a 1 MiB body
cap.

So a payload that fits now lives in the `events` row and the blob write
disappears for it. Not one of the three options — it **weakens nothing**.
SQLite's own `FULL` sync covers the payload inside a transaction already
being paid for.

The archive went from 1 356 to **4 133 records/s**. The mismatch
narrowed from 12× to 4×.

Two false readings almost shipped on the way there. The first drain
measurement read 691/s — *half* the old rate — because the backlog
emptied partway through the window and the rest was idle time. The
second alternated 0 and 6 033/s because the position gauge only
publishes on a ten-second tick and I sampled at five.

## The platform inverts the answer

| | macOS | Linux |
| --- | ---: | ---: |
| blob only, with fsync | 252/s | 1 826/s |
| SQL only, `synchronous=FULL` | 19 358/s | 2 988/s |

On macOS the blob fsync is 99% of the cost. On Linux the two are the
same order of magnitude. The cause is `File.Sync()` issuing
`F_FULLFSYNC` on macOS — ~4ms *regardless of which disk*, which the
external and internal readings agreeing within 4% gives away.

**Measuring on the development machine would have produced the wrong
production answer with a 33× margin that looked conclusive.**

## The gate found what a phase of testing had not

`make load-gate` passes. Its value was failing.

Proving it red by taking the broker away mid-run produced five red
assertions — including 10 222 records dropped and **counted**, which is
the promise working. But the parenthesis said `unaccounted = -70`.

[ADR-0008](../contract/decisions/0008-what-a-zero-is-allowed-to-mean.md)
had `committed` counting commit *operations*, so a redelivered sequence
contributed once to `span` and twice to `committed`. Correct about rows;
wrong about sequences.

**That defect shipped in PR-3.6 and survived a full phase, an ADR, and
`make loss-audit`** — which induces outages but had never induced enough
redelivery *under load* to drive the figure past zero. It was found
within an hour of the gate existing, by the gate failing for a reason
nobody predicted.

A loss figure that can be negative is not a loss figure.
[ADR-0010](../contract/decisions/0010-unaccounted-must-never-be-negative.md)
corrects it — and **the first fix was also wrong**, in the other
direction: subtracting duplicates over-corrects, because a redelivery
adds no sequence to the span. Twenty redeliveries produced −19 in a test
written for that first fix. The test came before the fix was trusted,
which is the only reason the wrong correction did not ship.

A second defect surfaced the same way: the archive read its position and
its row count as two separate queries, so the published pair could claim
more rows than commits. A scraper assumes a snapshot; it was not one.

## The lesson learned twice

A timing assertion on shared CI hardware encodes that hardware.

It failed once when a runner measured the SQL half at 524/s against
1 836/s for the blob half — the reverse of the container — so an inlined
commit was legitimately slower than a bare blob write. I removed that
assertion and left another. The other one then failed when a runner
measured an inlined commit at **48/s against 18 244/s locally**.

Timing measurements produce **numbers**, which is a different job from
guarding a property. They live in `make measure` now. The properties
they describe are structural — an inlined commit writes no file, a store
keeps its content-addressing check — and those still run in CI.

## What the numbers claim now

[ADR-0011](../contract/decisions/0011-a-latency-is-a-conditional-claim.md):
a published latency claims a **bound under a stated condition**, not a
value. `provenance.run.load` records the condition beside `topology`,
and `numbers_check` refuses to compare across either.

The six numbers reproduce after the substrate rewrite. Nothing moved —
which is worth having rather than assuming, since a content-addressed
store answering with different bytes would show up as a detection change
first.

## What Phase 4 leaves open

- **There is still no backpressure**, and it is now the largest thing in
  the register. The archive commits 4 133 records/s against a collector
  that bends near 16 000.
- **One machine, minutes not hours.** Nothing here is thermal, nothing
  is a leak test, and the driver competes with the system under test.
- **Docker Desktop on macOS is a VM.** The ratios travel. The magnitudes
  do not, and this phase produced one vivid demonstration of exactly
  that.
- **The false-positive rate is still `null`.** It governs every other
  number, it is calendar-bound rather than effort-bound, and no phase
  moves it. It needs recruiting, and the instrument for it is the
  parallel track that has not been started.
