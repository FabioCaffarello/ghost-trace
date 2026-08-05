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

### Cross-service invariants held by comments

- `-session-ttl` must be identical in the collector and the decision
  engine — it is the KV bucket's TTL, and whichever service starts last
  silently rewrites it. `compose.yml` says "must match". Nothing checks.
- The tenant registry must be identical in both. Nothing checks.

### State durability

Session state is a map in the collector's memory; a restart loses every
live session. The composed topology therefore degrades **better** than
the all-in-one — the engine keeps answering from KV snapshots the
restart did not touch — and that asymmetry is neither measured nor
written down anywhere but here.

### Measurement integrity

- **Nothing has run under load.** Every published latency is one session
  on an idle system, which the schema itself calls a floor. The split's
  cost is the kind that grows with contention.
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

### Pull requests

| | | size |
| --- | --- | --- |
| **3.1** | A domain-metric surface beside the HTTP one. `libs/middleware` counts requests; the services need to count **records**. Counters and gauges registered by name, encoded into the existing `/metrics`. Not named "telemetry" — that word already means the browser channel. | M |
| **3.2** | Loss accounting in the collector: every best-effort drop counted **by reason** (deadline, archive error, snapshot error), not merely logged. The largest of the three paths. | M |
| **3.3** | Archive-side accounting: committed, rejected, redelivered, and **consumer lag**, exposed by the archive service rather than inferred by a kill-test. | M |
| **3.4** | Age-out becomes loud. A record leaving the stream unarchived is silent loss today; it has to be counted, and the condition that produces it detectable well before seven days pass. | M |
| **3.5** | Cross-service invariants become checks. The engine reads the KV bucket's actual TTL and refuses a mismatch instead of silently rewriting it; both services log a registry fingerprint that `make shadow-http` compares. | S |
| **3.6** | **`make loss-audit`** — the gate as a repeatable target: drive traffic, induce the outage, reconcile counters against the archive. Refuses rather than skips, like every other topology check. | M |
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
