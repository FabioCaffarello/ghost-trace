# Where a commit's time goes — two fsyncs, not one, and it depends on the platform

**Date** 2026-08-05 · **Target** `go test ./libs/substrate -run
CommitSpends` · **Platforms** macOS host (10 cores, external and
internal storage) and `golang:1.26-alpine` container

PR-4.2 found the archive is the constraint: 1 356 commits/s against a
collector publishing ~16 000 records/s. It put a hypothesis on record
before this measurement existed, and this is the correction.

## What was predicted

> the substrate opens with `PRAGMA synchronous=FULL` and serialises
> every write through one `writeMu`, so 1 356 commits/s is close to what
> **one fsync per commit** on this hardware would allow. If that is
> right, the lever is batching commits into one transaction.

**One** fsync. Reading the write path afterwards shows **two**: the blob
is written to a temp file, `tmp.Sync()`ed and renamed *before SQLite is
touched at all*.

Two explanations, one fingerprint. A fix aimed at the wrong one would be
a well-tested change that buys nothing.

## The decomposition

Each variant removes exactly one thing, so the difference between two
rows is that thing's cost rather than a guess about it.

| | macOS (external) | macOS (internal) | **Linux container** |
| --- | ---: | ---: | ---: |
| full `AppendCanonicalAt` | 250 | 234 | **938** |
| blob only, with fsync | 252 | 260 | 1 826 |
| blob only, no fsync | 8 414 | 8 556 | 59 142 |
| SQL only, `synchronous=FULL` | 19 358 | 20 972 | **2 988** |
| SQL only, `synchronous=NORMAL` | 36 032 | 36 262 | 31 756 |

Operations per second. The halves run one after the other inside one
`writeMu`, so their reciprocals should add to the whole — they predict
249/s against 250 measured on macOS, 1 133 against 938 on Linux.
**Between 83% and 100% of the commit is accounted for**, which is what
makes the attribution below usable rather than suggestive.

## The finding: the platform inverts the answer

**On macOS the blob fsync is essentially the entire cost** — 252/s with
it, 8 414/s without, a 33× difference — while SQLite at
`synchronous=FULL` runs at 19 358/s and contributes about 1% of the
time.

**On Linux the two are the same order of magnitude.** The blob half
manages 1 826/s and the SQL half 2 988/s. Neither is *the* constraint;
both are.

The inversion is Go's `File.Sync()` issuing `F_FULLFSYNC` on macOS,
which forces a full drive-cache flush and costs about 4ms regardless of
which disk it is — the external and internal readings are within 4% of
each other, which is itself the giveaway. SQLite's own sync does not do
that.

**Measuring this on the development machine would have produced the
wrong answer for production**, and produced it with a 33× margin that
would have looked conclusive.

## What each lever is actually worth

Computed from the measured parts, on Linux, where the archive runs:

| | commits/s | |
| --- | ---: | --- |
| today | 1 133 | |
| drop the blob fsync only | 2 844 | 2.5× |
| `synchronous=NORMAL` only | 1 727 | 1.5× |
| **both** | **20 662** | **18.2×** |

Amdahl, plainly: **fixing either one alone is disappointing.** The
roadmap named the smaller of the two.

And the named fix specifically — batching rows into one transaction — is
worth **16.4×** on Linux *on the SQL half*, which is roughly 40% of the
commit. End to end that ceiling is about 1.5×.

## The durability question this hands to 4.3

Neither fsync can simply be deleted; both exist for a reason and the
reason is stated in the code. But the archive's delivery contract
narrows what they are protecting against, and that is worth writing down
before anyone argues about it:

**The archive does not acknowledge a message until the commit
succeeds.** A crash *before* the ack means redelivery — the stream still
holds the record, and at-least-once plus content-addressed idempotency
handles it. That is already the design.

So the fsyncs protect exactly one case: a crash **after** the ack, where
the broker believes the record is durable and the disk has not yet
written it. Weakening them means acked records can be lost to power
loss, while the stream still holds them for its retention window.

That is a real trade with a real name, and it is an **ADR**, not a
patch. 4.3 should make it explicitly, or find a third option — writing
blobs without a per-record sync and fsyncing the shard directory once
per batch is one; so is not writing blobs for payloads small enough to
live in the row.

## What this does not say

- **Both hosts are virtualised or laptop-class.** The Linux figures come
  from a container on Docker Desktop, which is a VM on the same macOS
  host. A real Linux server with a battery-backed controller would
  change every absolute number here and could change the ratio too.
- **The container measured 938/s here against 1 356/s for the running
  archive in PR-4.2.** Same order, different code path — the real
  archive also decodes protobuf and talks to NATS. The gap is not
  explained, only bounded.
- **Single-threaded.** Every figure is one writer, because `writeMu`
  makes it one writer. Whether concurrent callers would change the fsync
  economics is not measured.
- **Payloads are ~260 bytes.** Blob cost is dominated by syscalls at
  this size; larger payloads would shift the balance toward bandwidth.

## Reproducing

```sh
go test ./libs/substrate -run CommitSpends -v                       # host
docker run --rm -v "$PWD":/src -w /src/libs/substrate -e GOWORK=off \
  golang:1.26-alpine go test ./ -run CommitSpends -v                # linux
```

The assertions in `cost_test.go` are deliberately loose — they run on
shared CI hardware, where a tight timing band is a flake generator
rather than a gate. They still fail if a third cost appears, if the two
halves stop being serial, or if batching stops helping.
