# The books balance — first reconciled run

**Date** 2026-08-05 · **Target** `make loss-audit` · **Topology** compose
`core` profile, four services and a broker on one machine ·
**Records** 25 per scenario

The gate for the accounting phase. Three phases built numbers; this one
decides whether they add up, and it is the first run where the archive
could answer *what it holds* rather than *what this process did*.

## What changed underneath

Before this, the archive's accounting was three process counters. 3.4
established why that is not enough, by measurement rather than by
argument:

- **Counters reset.** Restarting the archive mid-backlog took
  `committed` from 5 to 10 while fifteen records had in fact arrived.
- **The broker cannot be asked.** It advances a consumer's ack floor
  when records are removed from under it, because there is nothing left
  to acknowledge. Purging four unconsumed records left `first_seq=15`,
  `ack_floor=14` — the evidence destroyed by the event it would
  evidence.

So the archive now writes its own position down, in its own database, in
the **same transaction** as the record. Four durable numbers make one
subtraction possible:

```
span        = highest_seq - first_seq + 1
accounted   = committed + rejected
unaccounted = span - accounted
```

## The run

```
== 25 records through an intact topology ==
  accepted: {'sessions': 25, 'telemetry': 25, 'decisions': 25, 'outcomes': 25}
  ok    the archive drained after a clean run (pending=0, unaccounted=0)
  ok    the archive publishes a durable position at all
  ok    the archive committed at least what was accepted (100 commits for 100 accepted records)
  ok    nothing is unaccounted after a clean run
  ok    nothing left the stream ahead of the archive
  ok    the collector dropped nothing with everything up (dropped=0.0)

== the archive is taken away mid-traffic, then brought back ==
  accepted while the archive was away: {'sessions': 25, 'telemetry': 25, 'decisions': 25, 'outcomes': 25}
  ok    traffic is still accepted with the archive down (ADR-0006)
  ok    the archive comes back
  ok    the archive drained after the archive returned (pending=0, unaccounted=0)
  ok    the queued records were committed on return (100)
  ok    the durable position advanced past where it was before the outage
  ok    an outage the archive recovered from leaves nothing unaccounted
  ok    nothing aged out while the archive was away

== the broker is taken away: loss is allowed, silence is not ==
  accepted while the broker was away: {'sessions': 25, 'telemetry': 25, 'decisions': 0, 'outcomes': 0}
  ok    the collector counted what it could not write (75 drops)
  ok    telemetry is still ACCEPTED with the broker down
  ok    the collector is still serving
  ok    the archive is still serving

== the books ==
  archive position: first sequence           1
  archive position: highest sequence         200
  archive position: commit operations        200
  archive position: refused on purpose       0
  archive position: UNACCOUNTED              0
  archive: rows actually held                200
  stream: skipped ahead of the archive       0
  stream: pending                            0
  collector: records written                 150
  collector: records dropped                 75

  the books balance.
```

## Three things the run says that were not known before

**A fresh archive publishes no position at all.** Scraping the archive
seconds after `make up` returned exactly one series —
`position_read_failures_total 0` — and none of the position gauges. That
is the repository's rule holding in the service rather than only in a
test: an archive that has consumed nothing must not report
`unaccounted 0`, because that reads as a perfect run and is
indistinguishable from one. The gauges are deliberately left
unmaterialised until there is a reading behind them.

**The decision engine refuses when the broker is down; the collector
does not.** With NATS stopped, 25 sessions and 25 telemetry posts were
still accepted while 0 decisions and 0 outcomes were. That is the
fail-open promise applying where it was designed to (the request path a
page sits on) and not applying where a wrong answer would be worse than
no answer.

**Loss is permitted; silence is not.** 75 drops were counted with the
broker away. The requirement was never that nothing is lost — it is that
nothing is lost without a number moving.

## The age-out measurement fires

3.4 shipped an early warning and explicitly could not ship a count,
because the case that needs catching — a consumer far enough behind that
discard overtakes it — "needs load this repository has not built". With
a durable position it needs no load at all, only a stream that moves on
without the archive.

Forced by hand: the archive was stopped, 15 records driven (60 stream
messages), and the stream purged before the archive returned.

```
ghosttrace_archive_position_highest_sequence 250
ghosttrace_archive_stream_first_sequence     311
ghosttrace_archive_stream_skipped             60
ghosttrace_archive_position_unaccounted        0
```

`311 − 250 − 1 = 60`. Sixty records entered the stream and left it
without ever being delivered, and the archive says so.

**`skipped` and `unaccounted` are both correct at once, and they are not
the same number.** `unaccounted` is the hole *inside* the range the
archive has walked; `skipped` is what left *above* its mark. A record
lost while the archive was reading past it lands in the first; a record
discarded before the archive ever got near it lands in the second.
Reading either alone would miss a whole class of loss.

The third attempt works for a reason worth stating plainly: **it stopped
asking the broker where the consumer had got to.** Both failed attempts
read a number the broker maintains, and the broker rewrites exactly
those numbers when it discards records.

## What this does not measure

- **A single machine, no load.** 25 records per scenario is enough to
  prove the accounting is *sound*, and nothing at all about whether it
  holds at rate. Load testing stays on the debt register.
- **Duplicates outside the broker's dedup window** would read as extra
  commit operations, visible as `committed` exceeding `rows`. In this
  run the two were equal, so the case is untested here.
- **A crash between the blob write and the transaction** leaves a blob
  with no row. That is pre-existing and harmless — content-addressed, so
  it is either collected or rewritten identically — but it is reasoned
  about rather than measured.
- **Nothing here says the six numbers reproduce.** That is `make
  numbers`, and it is a different claim.

## Reproducing

```sh
make docker-build && make up
make loss-audit
```

It refuses when the topology is down. A gate that skips is the failure
this phase was opened to remove: `make shadow` skipping without
`GT_NATS_URL` is how a broken tenant lookup reached CI.
