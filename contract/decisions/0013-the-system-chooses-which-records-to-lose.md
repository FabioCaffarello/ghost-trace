# 0013 — the system chooses which records to lose, and sheds the new to save the old

**Status:** accepted · **Date:** 2026-08-12 · **Milestone:** PR-5.7

## Context

Phase 4 passed its gate and named what it had not solved: **the archive
commits about 4 133 records/s against a collector that bends near
16 000.** PR-5.4 narrowed that with batch commits — 3.2× at a batch of
128 — but narrowing a gap is not deciding what happens in it, and the
gap has not been re-measured (see *Consequences*).

The gap itself is not the problem this ADR is about. A system whose
producer outruns its consumer must lose records, and this one has said
so since [ADR-0006](0006-the-stream-is-the-archive.md). The problem is
**who decides which ones**, and until Phase 5 the answer was: nobody.

Three separate mechanisms lost records, and not one of them was a
decision:

- The best-effort archive write on `/v1/decisions` had **no deadline**.
  With the broker away it stalled about five seconds inside an 80ms
  budget, and `libs/decision` had no metrics at all, so the loss was
  both unbounded and uncounted (PR-5.1).
- The stream had **no byte ceiling**. At the measured surplus the
  volume fills in hours, and NATS and the collector die together — a
  disk-full that takes down ingestion, which is strictly worse than the
  data loss the design already accepts
  ([ADR-0012](0012-the-stream-is-bounded-by-bytes.md), PR-5.3).
- Nothing told the collector the archive was falling behind. Local
  counters climb while a backlog builds, and **the first news of
  trouble is records ageing out** (PR-5.5).

A system without backpressure does not avoid loss. It defers the choice
of which loss to whichever component gives way first — and that
component chooses badly, silently, and all at once.

ADR-0012 set this decision up explicitly. It considered `DiscardNew` at
the broker, called it *genuinely tempting* because it converts silent
loss into backpressure, and **rejected it there and deferred it here**:
applied at the stream it makes ingest fail the moment the archive falls
behind, which is a fail-closed answer to a path
[contract §7](../architecture.md) has decided to fail open on. (ADR-0012
cited §5 there. §5 is the 80ms decision budget and its own fail-open;
the promise that governs *ingest* — telemetry accepted and dropped,
202 — is §7.)

## Decision

### 1. Every loss is counted, attributed, and bounded in time

No path may lose a record without incrementing a counter that names
**why**. Every best-effort write runs under `BestEffortTimeout`
(250ms), so no loss path can also be a latency path.

The vocabularies are close but deliberately not identical.
`libs/decision` declares `deadline` and `error`; the collector declares
those two and `shed`. The engine has no ingest path to shed from — it
answers a decision and archives a copy — so a `shed` counter there
would be a series that can never move, which
[ADR-0008](0008-what-a-zero-is-allowed-to-mean.md) treats as a lie of a
different kind than an absent one. The two constant sets are declared
separately rather than shared, because the alternative is a library
whose only purpose is to make two packages agree about strings they use
differently.

`ErrArchiveUnavailable` is deliberately **not** a drop. A run
configured with no archive never had a store to lose records from, and
counting that would make every development run look catastrophic while
telling nobody anything — the same distinction
[ADR-0008](0008-what-a-zero-is-allowed-to-mean.md) draws.

### 2. The collector learns the archive's backlog from the broker, not from the archive

`eventstream.WatchArchive` binds the archive's durable consumer
**read-only** from inside the collector and reads its position. Not an
HTTP call to the archive: the ingest path must not depend on another
service answering, and a service that is falling behind is exactly the
service least able to answer.

It never *creates* the consumer. Creating one would reset the delivery
position, so a monitoring mechanism would cause the loss it exists to
observe.

### 3. Above 80% of the retention window, the collector sheds the new to save the old

`SheddingThreshold = 0.8` of the stream's retention window
(`oldest_age / max_age`). Below it the archive is behind but the
backlog is recoverable and every record handed over is one the archive
will get to. Above it the stream is about to discard its **oldest**
records — the ones already accepted and waiting — so adding newer ones
trades a record that would have been stored for one that will not be.

**Shedding the new to save the old is the whole argument.** Both paths
lose records. Only one of them loses records nobody counted, at a
moment nobody chose, from the end of the queue that has been waiting
longest.

The shed happens **before** the write is attempted, not as its failure.
A record dropped by decision and a record dropped by timeout are
different facts, and `ReasonShed` says which.

### 4. No reading is −1, and −1 is not 0

`archivepressure.Unknown = -1.0`. A poll that failed, a stream with no
retention window, and a watcher that has not yet run are all states in
which nothing is known, and **zero would mean "the archive is
completely caught up"** — the most dangerous thing to guess wrong,
because it is the reading that disables shedding.

A failed poll moves the level in neither direction: an unreachable
broker is not evidence about the archive's backlog. A reading older
than `Stale` (90s) stops counting, because acting on the past is how a
collector keeps shedding after the archive recovered.

The level is published as `archive_pressure` via `GaugeFunc`, read at
scrape time rather than mirrored, so what an operator sees cannot drift
from what the shed decision used.

## Consequences

**The loss order is now a design, and it can be stated.** Under
worsening pressure the system gives up, in order: the decision path's
archive copy (deadline, 250ms, counted); then newly-arriving records at
the collector (shed, counted, attributed); and only then — if pressure
somehow passes the ceiling anyway — the stream's oldest records
(`stream_skipped`, counted by the archive against its durable
position). Nothing in that list is silent.

**The quantitative claim is not made here, and its absence is not
zero.** How much of the ~4× gap PR-5.4's batching actually closed has
**not been re-measured**: `make load-gate` and the load curves must be
re-run with the corrected driver, and until they are, the figures this
project publishes for the archive's rate predate the batching. The
threshold of 0.8 is therefore argued from the *shape* of the failure —
what is lost when the stream wraps — and not from a measured headroom.
It is a defensible number, not a derived one, and this ADR should be
revisited when 5.6b reports.

**The SDK is not told to slow down.** The collector sheds; it does not
answer 429, and the browser goes on sending. That is a wire-contract
change and a product decision, and uniform fail-open (contract §7) is
the current answer.

**Shedding is invisible to the page, by design.** A shed telemetry
record still returns 202 — the same answer §7 already requires for an
expired token, and for the same reason: the client is not misbehaving
and nothing it could do would help. The promise to the caller is that
the page is never broken by our accounting; the record of what it cost
lives in the counters, not in the response.

## Alternatives considered

**`Discard: DiscardNew` at the stream.** The version of this idea
without a decision attached: the publish simply fails once the stream
is full. It gives the same "protect the old" ordering for free, but the
cliff is at 100% of the bound rather than at a level the collector
chose, it cannot distinguish a shed from a broker error at the call
site, and it makes the ingest path's behaviour a property of broker
configuration rather than of the service that owes the promise.

**Block the ingest path until the archive catches up.** True
backpressure, and it would preserve every record. It also converts a
storage problem into an availability problem for a service whose entire
contract is to answer quickly and never break the page. Rejected on
contract §7.

**Shed proportionally — drop a growing fraction as pressure rises.**
Smoother, and probably better under a slowly worsening backlog. It also
means that below the threshold the system is *already* losing records
that would have been stored, which cannot be justified while the
backlog is recoverable. A single threshold is the version whose
argument survives being written down; proportional shedding is worth
revisiting once 5.6b establishes real headroom.

**Shed the oldest instead — let the queue drain from the front.** This
is what happens with no decision at all, and it is what the shed exists
to prevent. The oldest records are the ones already accepted, already
counted as written, and closest to being stored.

**Rely on the alert from PR-5.2.** An operator who is awake could act.
A response that depends on someone reading a dashboard in time is not a
mechanism, the same objection ADR-0012 made to an unbounded stream.
