# 0010 — unaccounted counts sequences, not commit operations

**Status:** accepted · **Date:** 2026-08-05 · **Milestone:** PR-4.6

Corrects one consequence of
[ADR-0008](0008-what-a-zero-is-allowed-to-mean.md), which otherwise
stands.

## Context

ADR-0008 established the archive's accounting:

```
span        = highest_seq - first_seq + 1
accounted   = committed + rejected
unaccounted = span - accounted
```

and stated, deliberately:

> **`committed` counts commit OPERATIONS, not rows.** A record delivered
> twice commits twice and dedups to one row, so `committed` minus the
> row count is the duplicate volume rather than a discrepancy.

That reasoning is sound about *rows*. It is wrong about *sequences*, and
the two were conflated.

`span` counts sequences. `committed` counted operations. A redelivered
sequence therefore contributed **once** to the left of the subtraction
and **twice** to the right, so `committed` could exceed `span`.

The Phase 4 gate found it, under exactly the condition that produces
redelivery — the broker taken away mid-run:

```
FAIL  the archive did not drain after sustained ingest
      (pending=0.0, unaccounted=-70.0)
```

**A loss figure that can be negative is not a loss figure.** It had been
correct in every test and every clean run, because none of them
redelivered anything.

## Decision

**`committed` counts DISTINCT RECORDS. Redeliveries are counted
separately as `duplicates`, and `duplicates` is reported beside the
subtraction rather than inside it.**

```
unaccounted = span - committed - rejected
```

Distinctness is not inferred. `INSERT OR IGNORE` reports zero rows
affected on a primary-key conflict, which is precisely a delivery of a
record the substrate already holds, and that is what the position update
now reads.

## Why duplicates are not subtracted

Because that is the same error in the other direction, and it was made
first. The initial fix subtracted `duplicates` as well, on the reasoning
that a duplicate "explains its sequence". It does not — **the sequence
was already explained by the original commit**, so subtracting again
double-counts.

Twenty redeliveries of one record produced `unaccounted = -19` in a test
written for that first fix. The test was written before the fix was
trusted, which is the only reason the wrong correction did not ship.

## Consequences

- **`committed` and `rows` now agree**, where ADR-0008 expected them to
  differ by the duplicate volume. That volume is
  `archive_position_duplicates`, a series of its own.
- **`stream_position` gains a `duplicates` column**, added by an
  idempotent migration for archives that predate it.
- **The published metric set gains
  `ghosttrace_archive_position_duplicates`.** Non-zero is the
  at-least-once mechanism working, not a fault — the same reading
  `archive_stream_redelivered` already carries from the broker's side,
  now durable and survivable across restarts.
- **ADR-0008 otherwise stands.** Its four rules are unchanged; only the
  definition of `committed` moves, and rule 1 — *a zero is publishable
  only when it was measured* — is what this correction serves. A
  negative reading is worse than an unpublished one.

## What this says about the gate

The defect existed from PR-3.6 and survived a full phase, an ADR, and
`make loss-audit` — which induces outages but had never induced enough
**redelivery under load** to drive the figure past zero.

It was found within an hour of the Phase 4 gate existing, by the gate
failing for a reason nobody had predicted. That is the argument for
gates that run against a live topology rather than assertions over
fixtures, and it is worth more than the fix.
