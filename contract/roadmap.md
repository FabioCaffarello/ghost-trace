# Ghost Trace — target architecture

Where the system is going. This was §10 of the architecture contract
until R1.16b, which is one genre too many for a document whose other ten
sections describe what must be true *now*: a contract and a plan age
differently, and mixing them means a reader cannot tell which sentences
they are allowed to rely on.

Nothing here is binding. The binding surface is
[`architecture.md`](architecture.md) §0–§9.

---

## Delivered

**Phase 1 — Clean Architecture inside the service.** Domain
(`session`, `feature`, `policy`, `canonical`) ← application use-cases
behind ports ← adapters. Contracts versioned with buf, generated code in
`libs/genproto`, the service and the experiment layer containerised, CI
gaining lint, vulnerability scanning, image build and breaking-change
detection.

**Phase 2 — Physical decomposition.** Four services: **collector**
(`/v1/sessions`, `/v1/telemetry`, the SDK; sole writer of session state,
snapshots to NATS KV), **decision-engine** (`/v1/decisions`,
`/v1/outcomes`; holds no session state, reads the KV), **archive**
(consumes the event stream into the substrate; idempotent by canonical
hash), **demo-web** (a customer's site on its own origin, a pure HTTP
client of the public contract). Written up in
[`../docs/the-split.md`](../docs/the-split.md).

**Phase 3 — the system can account for what it loses.** Three loss
paths made countable, an archive that writes its own position into its
own database in the same transaction as the record, and `make
loss-audit` as the gate. The measurement 3.4 proved could not be taken
from the broker was taken from that position instead. Written up in
[`../docs/counting-what-is-lost.md`](../docs/counting-what-is-lost.md);
the model is [ADR-0008](decisions/0008-what-a-zero-is-allowed-to-mean.md).

**Phase 4 — the numbers survive contention.** An open-loop load driver
that cannot hide the load it created, curves for all three services, the
archive's constraint diagnosed and tripled without a durability trade,
and `make load-gate`. The gate found a loss figure that could read
negative — shipped in PR-3.6, survived a full phase — within an hour of
existing. Written up in
[`../docs/measuring-under-load.md`](../docs/measuring-under-load.md); the
model is [ADR-0011](decisions/0011-a-latency-is-a-conditional-claim.md).

### Where the plan and the result differ

Recorded rather than quietly reconciled, because the difference is the
useful part.

- **gRPC was never built.** This document promised "internal synchronous
  calls are gRPC". Every internal call is HTTP: demo-web reaches the
  engine over the same public contract an integrator uses, which turned
  out to be the more valuable property — it dogfoods §3 instead of
  bypassing it. gRPC would buy a wire format nothing has needed yet.
- **The decision endpoints became a shared module, handlers included**
  (ADR-0005), rather than each service implementing the contract
  separately. Two hand-written HTTP layers for one contract is the drift
  this repository has already paid for twice.
- **The dual-write is gone earlier than "eventually"** (ADR-0006),
  because the parity it was waiting on was measured rather than assumed.

---

## What Phase 2 left behind

The input to the next phase. Every line is true today, and says where
the evidence is.

### Loss paths that cannot be counted

The split created three ways to lose a record, and all three are
**logged per occurrence and counted nowhere**. `/metrics` exposes HTTP
requests by route and status, and nothing about the domain.

- Best-effort writes are bounded at 250ms, so a broker that is slow or
  gone **drops** records an unbounded wait would eventually have
  published. Contract §5 permits the loss; nothing measures it.
- The stream discards by age (7 days, `DiscardOld`). An archive that
  stays behind long enough loses records silently, and nothing observes
  consumer lag outside `make kill-test`.
- Snapshot publication failures make another process's view stale. They
  produce a log line and no counter.

For a repository whose first rule is **absence is never zero**, this is
the largest standing contradiction: three known loss paths, none of them
addable.

> **Discharged by Phase 3.** All three are counted, every series is
> declared at zero before serving, and `make loss-audit` reconciles
> against a durable position rather than against process counters. The
> third path — age-out — could not be counted from broker state at all;
> see [ADR-0008](decisions/0008-what-a-zero-is-allowed-to-mean.md) rule
> 4, and the run at
> [`../docs/results/loss-audit-2026-08-05.md`](../docs/results/loss-audit-2026-08-05.md).

### Cross-service invariants held by comments

- `-session-ttl` must be identical in the collector and the decision
  engine — it is the KV bucket's TTL, and whichever service starts last
  silently rewrites it. `compose.yml` says "must match". Nothing checks.
- The tenant registry must be identical in both. Nothing checks.

> **Discharged by 3.5.** The engine reads the bucket's actual TTL and
> refuses a mismatch instead of rewriting it; both services log a
> registry fingerprint that `make shadow-http` compares.

### State durability

Session state is a map in the collector's memory; a restart loses every
live session. The composed topology therefore degrades **better** than
the all-in-one — the engine keeps answering from KV snapshots the
restart did not touch — and that asymmetry is neither measured nor
written down anywhere but here.

### Measurement integrity

- **Nothing has run under load.** Every published latency is one session
  on an idle system, which the schema itself calls a floor. The split's
  cost is the kind that grows with contention. Phase 3 removed the
  reason to keep deferring it — a load test can now report what it
  dropped — and added nothing to the evidence: `make loss-audit` drives
  twenty-five records per scenario, which proves the accounting is sound
  and says nothing about rate.

> **Discharged by Phase 4.** Curves for all three services, an open-loop
> driver that refuses a run it could not sustain, and `make load-gate`.
> What replaces it is narrower and worse: **the archive commits 4 133
> records/s against a collector that bends near 16 000, and there is no
> backpressure between them.** The stream absorbs the difference until
> retention does not, and `stream_skipped` reports it after the fact.
> That is now the largest standing gap.
- **Calibration is global.** Per-tenant calibration needs `libs/policy`
  to stop being a package global (13 uses of a package-level `cal`), and
  `policy.Ref` then stops being one value per run — which the manifest
  and `numbers-check` both assume.
- **Number 3's interpretation is unguarded.** The two published
  baselines differ mostly by archive backend rather than by topology.
  The comparison is guarded; the reading is not, and cannot be.
- **The demo is load-bearing for the invariant** while §6 calls it a
  stand-in. Five of six tiers drive its page.

### Process

- **Four of twenty-eight checks are required.** The image, broker,
  module-matrix and commit jobs all run on every pull request and none
  of them can block a merge.
- **There is no release workflow**, although R1.13 adopted Conventional
  Commits specifically so that semantic releases could read them.
- **The compose experiments profile is gated by nothing.** It broke
  twice in one session and was found by hand both times.
- Tier 3 records itself absent in a container: `undetected-chromedriver`
  needs real Chrome, which has no linux/arm64 build.

> **First three discharged by 3.7.** `main` requires three summary
> contexts — `ci — all checks`, `image — all checks`, `commits —
> conventional` — each of which fails unless every job beneath it
> succeeded, so a job added to a workflow is required the moment it
> exists. Releases are derived from the commit log by
> `scripts/next-release.py`; `v0.1.0` and `v0.2.0` were tagged and
> published without a hand touching them. The composed topology is
> gated by a `topology` job running `make shadow-http` and `make
> kill-test`.
>
> The summary jobs are themselves guarded: they count the results they
> read, because an expression that fails to evaluate yields an empty
> string and a loop over nothing succeeds. That is not theoretical — it
> happened, and `scripts/check-workflows.py` now asserts statically that
> every job is required. Tier 3's arm64 gap stands.

### Not-yet-true product claims

- **The false-positive rate has no data.** It governs every other
  number, and it is calendar-bound rather than effort-bound: it needs
  recruited participants, and the capture instrument already exists.
  This does not compete with a phase — it runs alongside one.
- **Telemetry replay** remains a stated limitation, not a solved
  problem.

---

## Phase 3 — the system can account for what it loses

The centre of gravity is the first section above. Phase 2 bought
decomposition and paid in **unmeasured** risk; this phase makes the
price visible.

It is deliberately not the load phase. Without loss accounting a load
test reports throughput while dropping records — it would measure the
wrong thing, confidently.

### The gate, which blocks the phase

> Under an induced broker outage and an induced archive lag, **every
> record the collector accepted is either in the archive or in a
> counter**, and the two reconcile.

Deliberately the same shape as the invariant that earned the dual-write
removal — `published + dropped == appended` — applied to a topology
where the local write no longer exists to compare against.

Ordered mitigation if it fails: count at whichever boundary is losing →
make that loss synchronous where the budget allows → widen the bound →
reintroduce a local spool as an explicit, ADR'd retreat.

**Met, 2026-08-05.** No mitigation was needed. The reconciliation is
stronger than the wording asked for: it is not "in the archive or in a
counter" but *in the archive, or refused on purpose, or named as
unaccounted* — with a fourth number, `stream_skipped`, for records that
left the stream above the archive's mark and would otherwise fall
outside the subtraction entirely.

### Pull requests

| | | size |
| --- | --- | --- |
| **3.1** | A domain-metric surface beside the HTTP one. `libs/middleware` counts requests; the services need to count **records**. Counters and gauges registered by name, encoded into the existing `/metrics`. Not named "telemetry" — that word already means the browser channel. | M |
| **3.2** | Loss accounting in the collector: every best-effort drop counted **by reason** (deadline, archive error, snapshot error), not merely logged. The largest of the three paths. | M |
| **3.3** | Archive-side accounting: committed, rejected, redelivered, and **consumer lag**, exposed by the archive service rather than inferred by a kill-test. | M |
| **3.4** | Age-out becomes loud. A record leaving the stream unarchived is silent loss today; it has to be counted, and the condition that produces it detectable well before seven days pass. Shipped the **early warning**; the count was proven unbuildable from broker state and deferred to 3.6, which built it. | M |
| **3.5** | Cross-service invariants become checks. The engine reads the KV bucket's actual TTL and refuses a mismatch instead of silently rewriting it; both services log a registry fingerprint that `make shadow-http` compares. | S |
| **3.6** | **`make loss-audit`** — the gate as a repeatable target: drive traffic, induce the outage, reconcile against the archive. Refuses rather than skips, like every other topology check. Needed a **durable position** first: 3.4 established that process counters do not survive a restart and the broker rewrites its own bookkeeping when it discards records, so the archive now writes its place in the stream down in the same transaction as the record. That also closed 3.4's deferred count — see [the run](../docs/results/loss-audit-2026-08-05.md). | M |
| **3.7** | Process debt: which checks are required, whether the experiments profile is gated, and the release workflow Conventional Commits were adopted for. Three small decisions that need an owner, not a large change. | S |
| **3.8** | An ADR for the accounting model, and the write-up: what is counted, what deliberately is not, and what a counter reading zero is allowed to mean. | S |

**Dependencies.** 3.1 gates 3.2, 3.3 and 3.4; those three gate 3.6,
which is the gate itself. 3.5, 3.7 and 3.8 are independent — and 3.7 can
go first if a red required-check list is wanted before the phase rather
than after it.

### What Phase 3 is explicitly not

Load, per-tenant calibration, and the false-positive rate. The first
depends on this phase's output to mean anything. The second is a product
feature sitting behind a refactor of `libs/policy`. The third is
calendar-bound and should be started **in parallel** rather than
scheduled afterwards — the longer it waits, the longer number 1 stays
unfalsifiable.

---

## Phase 4 — the numbers survive contention

Every latency this repository publishes is **one session on an idle
system**. The schema already says so about number 3, and
[`../docs/results/latency-gate-2026-08-04.md`](../docs/results/latency-gate-2026-08-04.md)
says it again: *a floor, not a production figure*. Phase 2 measured a
split whose cost is precisely the kind that grows with contention, and
then measured it without any.

Phase 3 is what makes this phase honest rather than merely possible. A
load test on the old system would have reported throughput while
dropping records — measuring the wrong thing, confidently. It can now
report what it dropped, which is the only reason the result will mean
anything.

### The gate, which blocks the phase

> Under sustained concurrency, **the accounting still balances** —
> `unaccounted` at zero, every drop counted, every refusal explained —
> and every latency number the repository publishes is either
> **reproduced** at load or **replaced by a curve that says where it
> bends**.

Both halves are load-bearing. A run that stays fast by dropping records
fails the first; a run that stays balanced by getting slower fails the
second. Passing by degradation is not passing, it is renaming.

Ordered mitigation if it fails: find the serialization point → widen it
where the concurrency model allows → bound the queue and count the
refusals → publish the curve with the bend in it rather than a number
that hides it.

**Met, 2026-08-06,** at 2 000 sessions/s and 4 000 decisions/s. No
mitigation was needed for the gate itself — but it passes because the
offered rate is below the archive's 4 133 records/s, not because the
~4x mismatch with the collector was solved. **The absence of
backpressure is unchanged and is now the phase's largest open item.**

### Pull requests

| | | size |
| --- | --- | --- |
| **4.1** | An **open-loop** load driver. Coordinated omission is the trap this whole phase can fall into: a closed-loop generator waits for each response before sending the next, so when the server slows the generator slows with it and the latency never appears in the numbers. The driver issues on a schedule and records intended-versus-actual send times, so queueing is visible rather than absorbed. | M |
| **4.2** | The collector under contention — where the floor stops being a floor. **The hypothesis is already on record, and not by us.** `session.Store` says in its own comment: *"One mutex guards the whole map. At M1 volume that is correct and boring; sharding it before there is a measurement showing contention would be optimising against a guess."* This is that measurement. The prediction is sharper than the comment: `Store.With` holds the process-global mutex across its whole callback, and the callback performs the full per-event feature update for every event in a telemetry batch — so telemetry throughput should be bounded by batch feature-computation time and should barely improve with cores. If the bend is somewhere else, that is the more interesting result and it gets written down as such. | M |
| **4.3** | Fix what the curve found — **done, and it turned out not to need the durability trade at all.** Every real payload measured between 60 and 161 bytes against a 1 MiB body cap, so a payload that fits now lives in the `events` row and the blob write disappears for it entirely. SQLite's `synchronous=FULL` covers the payload in the transaction that was already being paid for, so nothing is given up. The running archive went from 1 356 to **4 133 records/s (3.0x)**; the alternatives on the table — `synchronous=NORMAL`, dropping the blob fsync — all bought speed by weakening durability and were not taken. [ADR-0009](decisions/0009-small-payloads-live-in-the-row.md). | M |
| **4.4** | The archive under a real backlog — promoted from "one of three independent measurements" to **the phase's centre**, because 4.2 found it is the constraint. **Measured, and the hypothesis was wrong in a way worth keeping visible: it named ONE fsync (SQLite) and there are TWO — the blob is fsynced before SQLite is touched. On Linux the two cost the same order of magnitude, so neither alone is the constraint; the named lever, batching the transaction, ceilings at ~1.5x end to end while removing both fsyncs is worth 18x. And the attribution INVERTS between macOS and Linux, so measuring on the development machine would have given the wrong production answer with a 33x margin.** See [the decomposition](../docs/results/archive-commit-cost-2026-08-05.md). | M |
| **4.5** | The decision path at concurrency, against the 80ms budget it has only ever been measured against while idle. **Measured: it holds to 10 000 decisions/s at p99 6.57ms — twelve times inside the budget — and saturates between 10 000 and 11 000. The published idle floor of 1.393ms understated by 4.7x, which is what a floor does and not enough to change any conclusion drawn from it. The KV read costs about half what the engine itself does and is not what gives way first.** The decision path has the most headroom of the three services; the archive is still the constraint. See [the curve](../docs/results/decision-under-load-2026-08-05.md). | S |
| **4.6** | **`make load-gate`** — the phase gate as a repeatable target, refusing when the topology is down. **Passes**, and found two defects doing it: a skewed metric snapshot (position and row count read as two queries, so the published pair could say the archive held more rows than it performed commits), and — under a mid-run broker disruption — **`unaccounted = -70`**. A loss figure that can be negative is not a loss figure; [ADR-0010](decisions/0010-unaccounted-must-never-be-negative.md) corrects ADR-0008's definition of `committed`. See [the run](../docs/results/load-gate-2026-08-06.md). | M |
| **4.7** | Republish what moved, and teach `numbers-check` the difference. **The six numbers reproduce after Phase 4's substrate rewrite — nothing moved, which is worth having rather than assuming.** What was missing was the CONDITION: every run this harness has produced was idle, that was stated but never recorded, and 4.5 measured the same p99 at 4.7x under load. `provenance.run.load` now records it; `numbers_check` refuses to compare across conditions and picks a baseline of the same one, exactly as `check_topology` already did. See [the run](../docs/results/numbers-after-phase-4-2026-08-06.md). | M |
| **4.8** | An ADR for what the numbers now claim, and the write-up. [ADR-0011](decisions/0011-a-latency-is-a-conditional-claim.md) · [the write-up](../docs/measuring-under-load.md). | S |

**Dependencies.** 4.1 gates everything else. 4.2, 4.4 and 4.5 are
independent of one another. 4.3 depends on what 4.2 finds and cannot be
scheduled before it. 4.7 depends on whatever actually moved.

### The parallel track — the false-positive rate

Not part of the phase, and deliberately not scheduled after it. The
false-positive rate is **calendar-bound rather than effort-bound**: it
needs recruited participants, and no amount of engineering shortens the
recruiting. It governs every other number and it is `null`.

| | | size |
| --- | --- | --- |
| **4.P1** | `experiments/PARTICIPANTS.md` — **done.** The consent script in its own reviewable file, and `disclosure_test.go` comparing it against the vocabulary the SDK emits in both directions: a collected value nobody was told about fails the build, and so does a described channel that does not exist. Two sentences are asserted directly because no enumeration implies them — that key content is never collected, and that the data-governance RFC gating recruitment **has not been written**. Proven red three ways. | S |
| **4.P2** | The capture protocol end to end, dry-run against synthetic participants, so that recruiting is the only thing left that needs a human. | M |
| **4.P3** | **The data-governance RFC — drafted, not accepted.** Retention, deletion on request, and custody were three open questions that become promises the moment somebody is handed the script. [RFC-0001](rfcs/0001-human-study-data-governance.md) proposes a 12-month retention ceiling with a 90-day post-publication window, a `make forget P=<code>` deletion target whose implementation is a *precondition* of recruiting rather than a follow-up, and a custody section written as a limitation — no institutional review, so adults only, personal networks only, no secondary use. Accepting it is not a technical act and is deliberately left to a person. | S |

### What Phase 4 is explicitly not

- **State durability.** Session state is a map in the collector's
  memory and a restart loses every live session. Load will make that
  more visible, not less, and it is a product decision as much as a
  technical one.
- **Per-tenant calibration.** Still sitting behind a refactor of
  `libs/policy`.
- **Telemetry replay.** Still a stated limitation.
