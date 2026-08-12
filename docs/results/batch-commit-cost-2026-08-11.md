# What batching a commit is worth — 2026-08-11

**Target** `GT_MEASURE=1 go test ./libs/substrate -run WhatBatchingACommitIsWorth`
· **Machine** macOS/arm64, 10 cores, native (no container) · **Store**
local substrate, `journal_mode=WAL`, `synchronous=FULL`

> **The figure this replaces was measured against code that no longer
> runs.** [`archive-commit-cost-2026-08-05.md`](archive-commit-cost-2026-08-05.md)
> reported that batching the transaction "ceilings at ~1.5x end to
> end", and the roadmap has carried that number since. It was true of
> the path as it stood: a commit then paid **two** fsyncs — the blob,
> then SQLite — and batching amortised only the second. PR-4.3 removed
> the blob write for payloads that fit, which is every real payload
> measured. On the remaining one-fsync path, batching is a different
> proposition and had never been measured.

## The measurement

| transaction holds | records/s | vs one at a time |
| --- | ---: | ---: |
| 1 record | 17 628 | — |
| 8 records | 40 256 | 2.3× |
| 32 records | 48 576 | 2.8× |
| 128 records | 55 552 | **3.2×** |
| 512 records | 61 440 | 3.5× |

`synchronous=FULL` costs one fsync per *transaction*, not per row, so
the curve is the fsync being divided by N — and it flattens once the
fsync stops dominating. Between 128 and 512 the gain is 0.3× for four
times the redelivery window, which is why **128 is the batch size that
shipped**.

## What this is not

- **Not the archive's production rate.** This ran natively on macOS;
  the archive measured 4 133 records/s *in a container*, and this same
  machine does 17 628 for the identical single-record path. The ratio
  is what travels — and PR-4.4 already demonstrated that even fsync
  *attribution* inverts between macOS and Linux, with a 33× margin. The
  container figure has to be re-measured, which is Phase 5.6's job.
- **Not 18×.** That number, sometimes quoted from 4.4, was the value of
  removing *both* fsyncs — bought by weakening a durability promise,
  and refused then. Nothing here weakens one: `synchronous=FULL` still
  covers the payload, each record is still hash-verified before it is
  written, and the durable position still moves in the same transaction
  as the rows.
- **Not the end of the mismatch.** If the ratio survives into the
  container, the archive lands near 13 000 records/s against a
  collector that bends at 16 000. Closer, still short, still no
  backpressure — which is 5.5.

## The cost, stated

The unit of acknowledgement is now the batch. A crash mid-batch
redelivers up to 128 records instead of one; the substrate's content
addressing collapses them, which is the property that made the trade
affordable and the reason at-least-once was never doing the work alone.

Reproduce with `make measure`.
