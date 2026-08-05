# 0008 — what a zero is allowed to mean

**Status:** accepted · **Date:** 2026-08-05 · **Milestone:** PR-3.8

## Context

This repository already has the rule. It is the first of the two on the
front page:

> **Absence is never zero.** A tier that did not run is not a tier that
> found nothing. A false-positive rate with no human data is `null`.

Phase 3 applied it to a system rather than a table, and the application
turned out to be less obvious than the rule. Three loss paths exist —
the collector dropping a best-effort write, the archive refusing a
record, and a record ageing out of the stream — and at the start of the
phase all three were invisible. Not under-reported: **absent**. A
deployment that had never dropped anything and a deployment whose drop
counter was never wired exposed exactly the same thing, which is
nothing.

Six pull requests later there are numbers. This records the model they
share, because the model is what a future reader would otherwise reverse
one metric at a time.

## Decision

**Four rules govern every number the services publish about loss.**

### 1. A zero is publishable only when it was measured

Counters are **declared at zero** at construction, for every label
combination that can occur. A Prometheus counter created with
`WithLabelValues` does not appear until something increments it, so an
undeclared counter is silent exactly when it matters. After declaration,
a zero means measured-zero and a *missing* series means somebody added a
kind or a reason and forgot to list it — a bug in one file rather than a
quiet day.

Gauges are **not declared**, and the asymmetry is the point:

- A counter that has never fired is indistinguishable from a counter
  nobody wired up. Declaring removes the ambiguity.
- A gauge that has never been read is genuinely **unknown**, and zero is
  a specific claim about the world that happens to be false. Declaring
  creates the ambiguity.

So the archive's position gauges do not exist until there is a reading
behind them. An archive that has consumed nothing publishes no position
at all, rather than `unaccounted 0` — which reads as a perfect run and
is indistinguishable from one.

This was got wrong first. The gauges were declared at zero along with
the counters, by analogy, and the tests written to assert the rule are
what caught it.

### 2. Accounting that must survive a restart lives in the substrate

Process counters answer "what did **this process** do". That is a useful
question and they still answer it. It is not the question a
reconciliation asks.

Measured, not assumed: restarting the archive mid-backlog took
`committed` from 5 to 10 while fifteen records had in fact arrived.

So the archive writes its place in the stream — first sequence, highest
sequence, commits, refusals — into its own database, **in the same
transaction as the record**. A crash cannot leave it holding a record it
does not believe it holds, or believing it holds one it does not.

`make loss-audit` reconciles against that position and never against the
counters.

### 3. A refusal is an outcome, not a loss

```
span        = highest_seq - first_seq + 1
accounted   = committed + rejected
unaccounted = span - accounted
```

A record the archive refused on purpose consumed a sequence. If the
refusal is not recorded, that sequence has no explanation, and the audit
reports it as a record the transport lost — blaming the transport for a
decision this service made and logged.

Which is why a refusal that **cannot** be recorded returns an error
rather than acknowledging: the record is refused either way, but the
accounting gets another attempt. And why every path out of the
consumer's `Handle` accounts for its sequence or returns an error,
including two that previously had nowhere to go — an undecodable message
the stream library terminates before the handler runs, and a delivery
whose metadata carries no sequence.

### 4. Never measure a failure with state the failing component maintains

The rule that cost the most to learn, and the only one here that
generalises beyond this repository.

Age-out could not be counted from what the broker retains. Two ways were
built and both were deleted after measurement:

- **Stream first sequence against consumer ack floor.** The broker
  advances the ack floor when records are removed from under a consumer,
  because there is nothing left to acknowledge. Purging four unconsumed
  records left `first_seq=15`, `ack_floor=14`.
- **Jumps in delivered sequence.** A gap needs records to leave without
  being delivered, and a consumer that keeps up never sees one.

Both read a number the broker maintains, and the broker rewrites exactly
those numbers when it discards records. **The evidence is destroyed by
the event it would evidence.**

The third way works because it stopped asking. `stream_skipped` is the
stream's first surviving sequence minus the archive's **own** durable
mark, and nothing outside the archive can move that mark. Verified: with
the archive stopped and the stream purged, position 250 against stream
first 311 reported 60 skipped.

## What is deliberately not counted

Naming these is part of the model. A gap nobody wrote down becomes a
number somebody assumes.

- **What the browser never sent.** The SDK is fire-and-forget by design
  (§5, §7) and there is no channel back. Counting client-side loss would
  mean inventing one, and the first thing that channel would carry is a
  reason to trust it less.
- **Which producer a missing record came from.** The audit reconciles
  the stream as a whole. The collector and the decision engine both
  publish into it, so `unaccounted` says a record is missing and not
  whose it was.
- **A blob written without its row.** Possible if a crash lands between
  the blob write and the transaction. It is harmless — content-addressed
  content is either collected or rewritten identically — and it is
  reasoned about rather than measured.
- **Duplicates outside the broker's dedup window.** They appear as
  `committed` exceeding `rows`, which is the right shape, but nothing
  attributes them.
- **Loss per tenant.** The registry exists; per-tenant calibration is
  deferred, and so is per-tenant accounting.
- **Whether any of this holds at rate.** Twenty-five records per
  scenario proves the accounting is sound and says nothing about load.

## Consequences

- **`make loss-audit` is the phase gate and is outside `make ci`.** It
  stops and starts containers and takes minutes. It **refuses** when the
  topology is down, because a gate that skips is the failure this phase
  was opened to remove — `make shadow` skipping without `GT_NATS_URL` is
  how a broken tenant lookup reached CI.
- **`skipped` and `unaccounted` are both published, and they are not the
  same number.** One is the hole inside the range the archive walked;
  the other is what left above its mark. A run was observed with
  `skipped 60` and `unaccounted 0` simultaneously, both correct.
- **`committed` counts commit operations, not rows.** A redelivered
  record dedups to one row; counting rows would make every redelivery
  look like a vanished record. `committed - rows` is duplicate volume.
- **The archive's substrate now has a second table**, and one more
  reason for the archive and the collector not to share a data
  directory.
- **Absence stays absent through the whole pipeline.** The audit's
  scraper does not default a missing series to zero, because a reader
  that did would report a fresh archive as a perfectly reconciled one.

## What would reverse this

If per-record durability ever costs more than the accounting is worth —
the position write is one extra statement inside a transaction the
commit already pays for, so this is a claim about load, not about
design. Measure before assuming; the write is currently invisible beside
the blob write it accompanies.

If a broker is adopted that maintains consumer position in a way the
broker does **not** rewrite on discard, rule 4's implementation could
move back to it. Rule 4 itself would still hold.
