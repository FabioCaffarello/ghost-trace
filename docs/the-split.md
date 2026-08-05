# One binary becomes four

Phase 2. The vertical slice that produced the six numbers was a single
process; it is now a collector, a decision engine, an archive and a demo
host, with an event stream between them.

Written during the work. The parts worth reading are the four things the
split found, none of which were the things it was expected to find.

---

## What was actually being bought

Not scale — there is no traffic. Not availability — there are no users.
The split buys **the ability to answer a question separately from the
ability to observe one**, which is the shape a detector has to have if
the thing deciding is ever to be replaced, re-scored or run somewhere
else.

It was allowed to proceed on one condition: that a decision still lands
inside 80ms once it has to cross a network hop and a KV read. That gate
is [`results/latency-gate-2026-08-04.md`](results/latency-gate-2026-08-04.md),
and it passed: p99 1.393ms in the deployed topology, against a budget of
80.

## The shape

```
  browser ──sdk.js──> collector          (sessions, telemetry, the SDK)
                          │
                          ├── snapshot ──> KV bucket ──> decision engine
                          │                                    │
                          └── record ────> event stream ──> archive
                                                               │
  app server ─────────────────────────> decision engine        v
                                          /v1/decisions      substrate
                                          /v1/outcomes

  demo-web: a customer's site, on its own origin, calling the engine
            server-to-server exactly as an integrator would
```

Four deployables and NATS. The collector holds session state in memory
and is the only thing that observes a session; everything else works
from what it publishes.

`cmd/ghost-trace` remains an **all-in-one binary**: the same collector
process also mounts the decision endpoints over its own session store.
That is what keeps `experiments/numbers.py` free of Docker, and it is
the rollback the phase gate might have needed.

## Two mechanisms worth understanding

**Session snapshots (ADR-0004).** The collector writes the nineteen
numbers a judgement is computed from into a KV bucket keyed by
`(tenant, token)` — never raw events. Single writer, many readers: the
collector owns a session and is the only thing that writes it, which is
why there is no compare-and-swap. The TTL is the session TTL, so an
abandoned session expires rather than accumulating, and there is no
second place that decides when a session is over.

The mapping lives in one package with both directions next to each
other, because two mappings written apart would satisfy their own tests
and disagree with each other. It is **not lossless** — `float32` on the
wire against `float64` in the domain — and the loss is measured rather
than asserted.

**At-least-once plus content addressing (ADR-0006).** The archive
consumer acknowledges only after committing. A crash between the two
redelivers the record, and the redelivered copy collapses in the
substrate because identity is the hash of the canonical bytes. Those
bytes are produced exactly once, at publish, and travel as bytes:
deterministic marshalling makes no promise across builds, so a record
re-encoded downstream could hash differently and stop being the same
record.

A payload that does not match its hash is **dropped, not retried** —
redelivery brings the same bytes, and committing them would put a
payload under a hash that does not describe it.

## What the split found

None of these were on the plan. Each was found by a check that had to be
built before it could find anything.

**A fail-open promise that was only half true.** With the broker down,
`/v1/telemetry` returned its 202 — after **ten seconds**. A JetStream
publish waits five seconds for an ack that is not coming and telemetry
does two of them. Fail-open in status, closed in latency; at any real
concurrency it exhausts the collector. Best-effort writes are bounded at
250ms now, and the cost is that a bounded wait drops records an
unbounded one would eventually have published. `make kill-test`.

**A cross-tenant read.** A token from one tenant presented with another
tenant's secret returned a real decision about a session the caller had
no claim to. Both halves authenticated on their own, which is why
nothing caught it. Lookups are scoped to the tenant the caller proved,
and somebody else's session reads as a cold start rather than a refusal
— refusing would confirm the token exists.

**A requirement the demo had been hiding.** While the collector served
the demo page, the browser endpoints were same-origin with the API and
never had to answer a cross-origin request — the one thing every real
integration does. Nothing in the contract, the specification or the code
mentioned CORS, because nothing had needed it.

**A number that would have been read backwards.** The published
composed baseline (p99 1.515ms) is far below the published monolith ones
(5.525ms and 5.807ms), which looks like decomposition making decisions
nearly four times faster. It did not.

Measuring a monolith with the *stream* as its archive isolates it: the
split cost **+0.94ms**, and the other six milliseconds were a
synchronous SQLite append on the decision path that ADR-0006 retired one
pull request earlier.

## Two tensions this phase leaves standing

**The demo is load-bearing for the invariant.** Contract §6 calls the
demo surface "non-contract, but load-bearing for the project". Five of
the six adversarial tiers now load its page and drive it, so the number
that the whole repository is an argument about depends on a service
whose own documentation presents it as a stand-in. Nothing is wrong
today; the tension is that §6's wording and the tiers' dependency point
in opposite directions, and a future change to the demo will be reviewed
under the wrong assumption.

**Number 3 measures two different paths.** A manifest now records
`topology: monolith|composed`, and `numbers-check` refuses to compare
across them. But the two published baselines differ mostly by *archive
backend* rather than by topology, and a reader who compares them without
reading the latency gate will conclude the wrong thing. The manifests
are honest and the comparison is guarded; the interpretation is not
guarded and cannot be.

## What is still not true

- **Nothing here has run under load.** Every latency figure is one
  session on an idle system, which the schema calls a floor rather than
  a production figure. The split's cost is precisely the kind that grows
  with contention: a KV read is a round trip, and round trips queue.
- **Calibration is still global.** Tenants are isolated in state, keys
  and attribution, and judged by the same numbers. Per-tenant
  calibration needs `libs/policy` to stop being a package global, and
  `policy.Ref` then stops being one value per run — which the manifest
  and the invariant check both assume.
- **The false-positive rate still has no data**, which is the number
  that governs every other one. Nothing in this phase changed that, and
  no amount of architecture will.

## Where the pieces are recorded

| | |
| --- | --- |
| the external surface, including origins and tenants | [`contract/architecture.md`](../contract/architecture.md) §1, §6 |
| why each structural decision was made | [`contract/decisions/`](../contract/decisions/) — ADR-0001 to ADR-0006 |
| what the cutover from the dual-write cost | [`results/parity-cutover-2026-08-04.md`](results/parity-cutover-2026-08-04.md) |
| what the split cost in latency | [`results/latency-gate-2026-08-04.md`](results/latency-gate-2026-08-04.md) |
| the checks that need the topology up | `make shadow-http`, `make kill-test` |
| the checks that need a broker | `make parity`, `make shadow` |

Those four are different checks. Running two of them is not running the
topology's tests — a lesson this phase learned by pushing a change whose
only failing gate was the one that had been skipped.
