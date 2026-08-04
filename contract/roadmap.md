# Ghost Trace — target architecture

Where the system is going. This was §10 of the architecture contract
until R1.16b, which is one genre too many for a document whose other ten
sections describe what must be true *now*: a contract and a plan age
differently, and mixing them means a reader cannot tell which sentences
they are allowed to rely on.

Nothing here is binding. The binding surface is
[`architecture.md`](architecture.md) §0–§9.

---

## Two phases, one external surface

The refactor program in progress evolves the single binary in two phases
without changing the §3 surface:

**Phase 1 — Clean Architecture inside the service.** Domain
(`session`, `feature`, `policy`, `canonical`) ← application use-cases
behind ports (`SessionRepository`, `EventArchive`, `Clock`) ← adapters
(HTTP, in-memory store, substrate archive, proto mapping). Contracts
versioned with buf; generated code moves to an importable module
(`libs/genproto`); the service and the experiment layer become
containerized (`docker compose`); CI gains lint, vulnerability scanning,
image build/publish, and schema breaking-change detection.

**Phase 2 — Physical decomposition.** Four services with single
responsibilities: **collector** (`/v1/sessions`, `/v1/telemetry`; sole
writer of session state, snapshots to NATS KV), **decision-engine**
(`/v1/decisions`, `/v1/outcomes`; stateless, reads KV), **archive**
(consumes the event stream from NATS JetStream into the substrate;
idempotent by canonical hash), **demo-web** (a pure HTTP client of the
public contract — dogfooding it). Internal synchronous calls are gRPC;
the event flow is JetStream subjects partitioned by tenant. The split
ships only if the §5 budget holds in the composed topology, re-measured
by the same experiments that produced §9; the all-in-one binary remains
as the development composition and the cheap rollback.
