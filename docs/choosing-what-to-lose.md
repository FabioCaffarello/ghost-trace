# Choosing what to lose

*Phase 5, eight pull requests, 2026-08-11 to 2026-08-12. A gate that
was green with a service dead, a cap rejected by the test written to
justify it, a tenant that came from a flag instead of from the record,
and four gates that pass alone and do not compose.*

---

Phase 4 passed its gate and wrote down the thing it had not fixed: the
archive commits about **4 133 records/s** against a collector that
bends near **16 000**. The gate passed because the offered rate was
below the archive's, not because the mismatch was solved.

Phase 5 is about the gap. Not closing it — that is a throughput
problem and only partly solvable — but **deciding what happens inside
it.**

The framing that made the phase tractable: *a system without
backpressure does not avoid loss. It defers the choice of which loss to
whichever component gives way first.* And that component chooses
badly — silently, at a moment nobody picked, from the end of the queue
that has been waiting longest.

## Three mechanisms, none of them a decision

The starting position had three ways to lose a record and no policy
behind any of them.

The `/v1/decisions` handler wrote to the archive best-effort with **no
deadline**. With the broker away it stalled about five seconds, on a
path with an 80ms budget. `libs/decision` had **zero domain metrics**,
so this was not merely unbounded, it was uncounted. The collector had
already solved this for its own paths in Phase 3; the engine had not
inherited the fix, because the shared module carries handlers and not
observability.

The fix (PR-5.1) was a port of the collector's `BestEffortTimeout` and
loss counting. What is worth recording is what the *test* did. A
latency assertion added to the kill-test — a decision must come back
promptly with the broker down — failed on its first run for a second
reason: the engine's snapshot KV read, also unbounded, also five
seconds. One assertion, two unbounded paths, and only one of them was
the one being fixed.

The stream had `MaxAge: 7 days` and **no byte ceiling**. Age is not a
bound on anything an operator controls: at the measured surplus the
volume fills in hours, and when the disk fills NATS dies — and compose
makes the collector `depends_on` a healthy broker, so the collector
dies with it. The worst failure mode was not data loss. It was a
disk-full that takes down ingestion.

[ADR-0012](../contract/decisions/0012-the-stream-is-bounded-by-bytes.md)
bounds it at 4 GiB. The first number written was 2 GiB, chosen because
it looked reasonable, and **the test written to justify it rejected
it** — 2 GiB is 34 minutes of backlog, and the ADR's own prose claimed
more than an hour. The number in the file is now derived from the
requirement rather than the other way round, and the episode is in the
ADR because a cap that was picked by taste once will be picked by taste
again.

Nothing told the collector the archive was falling behind. Local
counters climb while a backlog builds; the first news of trouble is
records ageing out.

## The gate was green with the archive dead

Between the second and third of those came PR-5.2, which was supposed
to be routine instrumentation and was not.

The archive was the only service without a `-healthcheck` flag. Its
image is distroless, so there is no shell to run a `curl` in, so
compose had no healthcheck for it, so CI's topology job left it out of
the list of services it waits for. **The topology gate had been passing
with the archive dead.**

Prometheus was scraping three of four services. The archive's nineteen
series — including every position metric the loss accounting depends
on — were published and collected by nobody.

There is a pattern here worth naming, because it recurs: *the absence
of a mechanism is invisible to every check that consumes the
mechanism.* Nothing was broken. A thing simply was not there, and every
consumer of it degraded quietly to not looking.

## Shedding the new to save the old

With a bound on the stream and a signal about the backlog, the decision
becomes stateable, and
[ADR-0013](../contract/decisions/0013-the-system-chooses-which-records-to-lose.md)
states it.

Above 80% of the retention window the collector stops offering records
to the archive and counts each one as `ReasonShed`. The argument is not
that shedding avoids loss — it does not. It is that above that level
the stream is about to discard its **oldest** records, the ones already
accepted and waiting, so handing it newer ones trades a record that
would have been stored for one that will not be.

Both paths lose records. Only one of them loses records nobody counted.

Most of the care in PR-5.5 went into refusing to infer. No reading is
**−1, not 0**, because zero means *the archive is completely caught up*
and that is the reading that switches shedding off. A failed poll moves
the level in neither direction — an unreachable broker is not evidence
about the archive's backlog. A reading older than 90 seconds stops
counting, because acting on a stale calm is how a collector keeps
shedding after the archive recovered. And the watcher binds the durable
consumer **read-only**: creating one would reset the delivery position,
so the monitoring would cause the loss it exists to observe.

## What batching was actually worth

PR-5.4 made the archive commit in batches: **2.3× at 8, 3.2× at 128,
3.5× at 512** ([the measurement](results/batch-commit-cost-2026-08-11.md)).

The roadmap had this at "~1.5× ceiling", from Phase 4's decomposition.
That figure was correct when it was measured and wrong when it was
quoted: it measured the **pre-inlining two-fsync path**, and PR-4.3 had
since removed one of the two fsyncs. A number that goes stale because
the thing beneath it improved is the most quotable kind of wrong
number, since nothing about it looks out of date.

Two bugs surfaced from tests written in the same PR. The slow path that
isolates a poisoned record called back into the fast path, so a batch
of one recursed forever — the test *panicked* rather than failed, which
is its own lesson about what a stack overflow does to a test report.
And the test fake let commits through while refusing to record
rejections, which no real transaction can do; the fake was more
forgiving than the thing it stood in for.

## A record says which customer it belongs to

PR-5.7a began as a subject-routing bug from the strategic audit and was
worse than the audit said.

The archive envelope's tenant came from the archive's `-tenant` **flag**
while every payload already carried the tenant the request had proven.
With a multi-tenant registry the two agree only for the single customer
matching the flag. Every other customer's records were archived and
subject-routed as `t_demo` — so this was not a filtering bug on top of
correct data. **The durable attribution was wrong.** The stored answer
to *whose record is this* was the operator's default.

The tenant now comes from the payload, read by reflection over the
`tenant_id = 1` field that all five event types declare, and a payload
without one is refused rather than defaulted.

## The gates pass alone and do not compose

The phase's last finding came from its most clerical change.

`.context/config/sensors.json` is the file an agent reads to learn
which gates apply to a change it is making. It listed eleven sensors
and **not one of the four topology gates**. The project's strongest
claim-checking machinery did not exist as far as that file was
concerned, and `.context/README.md` states "every sensor is a `make`
target and nothing else" as a rule that nothing enforced.

So PR-5.6a added the four, made the rule enforceable in both
directions, and moved `make loss-audit` into CI's topology job — which
already brings the topology up and, until then, took it away again
without ever reconciling the books.

Its first CI run failed, and the failure was the useful kind. The gate
ran directly after `make kill-test`, which stops and restarts three
services and returns without waiting for them to settle. So
loss-audit's *25 records through an intact topology* scenario inherited
a topology that had just been taken apart, and reported four
unexplained drops and no commit delta in a run it believes to be clean.

Each gate is correct. Each gate was only ever run against a topology
someone had just brought up. **Composing them is a case nobody had
tested**, and the discipline the repository already had — refuse rather
than skip — cannot help here, because the gate did not skip. It ran,
and measured a scene that was not the scene it describes.

The fix gives the gate a freshly recreated topology, rather than
teaching it to tolerate a noisy one. *The collector dropped nothing
with everything up* is exactly the claim worth keeping strict; a gate
that shrugs at four unexplained drops is the failure the accounting
phase was opened to remove.

## What Phase 5 does not know

**How much of the 4× gap the batching closed.** `make load-gate` and
the load curves have **not** been re-run with the corrected driver, so
every archive-rate figure this repository publishes predates PR-5.4.
The shed threshold of 0.8 is argued from the *shape* of the failure —
what is lost when the stream wraps — and not from measured headroom. It
is a defensible number, not a derived one, and it is the first thing
that should move when the measurement lands.

That measurement needs a machine that can run the topology, and it must
not be a development Mac: Phase 4 established that the fsync
attribution **inverts** between macOS and Linux, with a 33× margin.
Publishing a native-macOS figure as a deployment number is the specific
error that decomposition exists to prevent.

**Whether the SDK should be told to slow down.** The collector sheds; it
does not answer 429. The browser goes on sending, and fail-open remains
the contract's answer.

---

## What the phase is

Eight pull requests, and the thing they add up to is not throughput.
Ingest is roughly where it was. What changed is that **every way this
system can lose a record is now a decision somebody made, bounded in
time, counted by reason, and visible from outside the process** — and
where a number is missing, the system says so rather than reporting a
zero.

The one claim it cannot yet make is how much room the decisions have to
work in. That is 5.6b, and it is honest to leave it open.
