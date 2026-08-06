# 0009 — small payloads live in the row

**Status:** accepted · **Date:** 2026-08-05 · **Milestone:** PR-4.3

## Context

PR-4.2 found the archive is the constraint on the whole topology: it
committed 1 356 records/s against a collector publishing ~16 000
records/s at its own bend, with no backpressure between them.

PR-4.4 decomposed a commit and found the roadmap's stated diagnosis was
wrong in a specific way. It named **one** fsync — SQLite's, under
`synchronous=FULL`. There are **two**: the blob is written to a temp
file, `tmp.Sync()`ed and renamed *before SQLite is touched at all*.

On Linux, where the archive runs, the blob half managed 1 826/s against
2 988/s for the SQL half. Neither dominated. The levers, computed from
the measured parts:

| | |
| --- | --- |
| drop the blob fsync | 2.5× |
| `synchronous=NORMAL` | 1.5× |
| both | 18.2× |

**Every one of those weakens a durability promise**, and the roadmap
said so: the decision belonged in an ADR rather than a patch.

Then the payload sizes were measured. Every record in a real run was
**between 60 and 161 bytes** — 100% under 256 — against a 1 MiB request
body cap.

## Decision

**A payload of `InlineThreshold` (16 KiB) or less is stored in the
`events` row. Larger payloads keep the content-addressed file.**

## Why this and not the three options on the table

Because it **weakens nothing**.

The blob fsync and `synchronous=FULL` both exist to make a committed
record survive power loss. Removing either buys speed by giving that up.
Inlining does not remove a guarantee — it removes a *second place to
put the bytes*. SQLite's `synchronous=FULL` then covers the payload as
well as the index, inside a transaction that was already being paid for.

The archive's delivery contract makes the alternatives look worse than
they first read. The archive does not acknowledge a message until the
commit succeeds, so a crash *before* the ack is already handled by
redelivery and content-addressed idempotency. The fsyncs protect exactly
one case: a crash **after** the ack, where the broker believes the
record is durable and the disk has not written it. That case is narrow —
but it is the one case where the archive's own accounting would say
`committed` about a record that is gone, and this phase exists to stop
numbers saying things that are not true.

So the durability trade was available, and was not taken.

## What it is worth

| | before | after | |
| --- | ---: | ---: | --- |
| commit, macOS host | 250/s | 18 244/s | 73× |
| commit, Linux container | 938/s | ~5 600/s | 4.7× |
| **the running archive, end to end** | **1 356/s** | **4 133/s** | **3.0×** |

The end-to-end figure is the one that matters and is the smallest, which
is the honest direction: the service also decodes protobuf and talks to
NATS, and those costs did not move. PR-4.4 recorded the gap between the
micro-benchmark and the running service as *bounded, not explained*;
this narrows it but does not close it.

After 180 000 sessions through the new path: 360 000 commits, 360 000
rows, zero unaccounted, zero skipped, and **zero blob files written**.

## Consequences

- **The `events` table now has a `payload` column**, declared in the DDL
  *and* added by an idempotent migration at `Open`. Both, deliberately:
  a fresh database should be correct from its schema alone rather than
  from a migration having run, and a migration that is the only
  definition is a schema nobody can read in one place.
- **A NULL payload column is the discriminator**, not `payload_ref`.
  Every row written before this change carries a `payload_ref`, so using
  it would have made old and new rows indistinguishable.
- **Records written before this change stay readable.** `ReadBlob` tries
  the row and falls back to the file. This is tested rather than
  asserted: an archive is append-only and long-lived, and a reader that
  only looked at the row would report every pre-existing record as
  missing — which, for a content-addressed store, is indistinguishable
  from having lost them.
- **The content-addressing check applies to both paths.** The hash is
  verified on read whether the bytes came from the row or the file, and
  the corruption test now runs against both. A store that answers with
  bytes not matching the name it was asked for is worse than one that
  answers nothing, and that has to stay true for the new path too.
- **The blob store is not removed.** It handles anything above the
  threshold, and it holds every record written before today.
- **The threshold is not tuned, and cannot be.** Everything measured is
  under 256 bytes; the cap on a request body is 1 MiB. Any value between
  those is equally correct for today's traffic, so 16 KiB was chosen as
  far above what the system produces and far below what makes a row
  unwieldy. If payload sizes ever approach it, that is a measurement,
  not a guess, and this ADR gets a successor.
- **`make ci` now guards the win.** `TestWhereACommitSpendsItsTime`
  fails if an inlined commit is ever dragged back down to the blob
  path's rate.

## What is still open

The archive at 4 133/s still absorbs less than the collector's ~16 000
records/s, so the mismatch is narrowed from ~12× to ~4× and **the
absence of backpressure is unchanged**. Both remain Phase 4 work.

`synchronous=NORMAL` and batched transactions are still available and
still cost durability. They are worth 1.5× and up to 16.4× on the SQL
half respectively — and now that the blob fsync is gone for real
records, the SQL half is the whole commit, so those numbers apply
directly. That is a different ADR, and it should be written only if
4 133/s turns out not to be enough.
