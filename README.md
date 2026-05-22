# Ghost Trace

A behavioral intelligence system designed to preserve the epistemic integrity of operational knowledge — the continued capacity to distinguish what was observed from what was inferred — as that knowledge accumulates, evolves, and is acted upon over time.

> Ghost Trace is not a detector that happens to be auditable. It is a behavioral intelligence substrate within which detection is the first applied domain.
>
> — [Constitutional Charter, §1](./docs/charter/constitutional-charter.md#1-thesis)

## Status

This repository is in **implementation phase**, post the [`decision-log §0022`](./docs/charter/decision-log.md) implementation pivot (v0.4.2 amendment, 2026-05-21). Constitutional sections continue to redact in committee mode in parallel with implementation work; §2.6 and §3 remain pending under the `empirical pressure phase` posture (redaction resumes when implementation surfaces concrete questions the Charter's stub does not already answer).

Current state:

| Document | Status |
|---|---|
| [Constitutional Charter](./docs/charter/constitutional-charter.md) | `v0.5+` — §1 Thesis frozen; §2 qualification criteria frozen; §2.1 / §2.2 / §2.3 / §2.4 / §2.5 frozen; §2.6 pending — empirical pressure phase STRONG (per [`§0120`](./docs/charter/decision-log.md)); §3 pending — first empirical-pressure assessment recorded (per [`§0121`](./docs/charter/decision-log.md)); §4 frozen |
| [Ontology](./docs/ontology/ontology.md) | Scaffold + multiple revisions accepted; Q1 / Q2 / Q3 / Q4 resolved; OMQ #2 / OMQ #3 resolved; Q2-A.2 cross-subtype merge+split follow-on at discussion phase (six framing documents under [`docs/rfcs/draft/`](./docs/rfcs/draft/) + [`docs/rfcs/discussion/`](./docs/rfcs/discussion/)) |
| Architecture | [`storage-model.md`](./docs/architecture/storage-model.md), [`projection-model.md`](./docs/architecture/projection-model.md), [`replay-model.md`](./docs/architecture/replay-model.md), [`event-flow.md`](./docs/architecture/event-flow.md), [`concurrency-pattern.md`](./docs/architecture/concurrency-pattern.md) — load-bearing for the implementation |
| Schemas | [`schemas/events/v1/`](./schemas/events/v1/) — typed Cat I/II/III proto definitions (DeclaredSession, OperationalSession, NetworkEvent, IngestionEvent, OrphanCleanupAudit, four-subtype Cat III formation/promotion/demotion/dissolution/merge/split protos) |
| Services | [`services/ingestion/`](./services/ingestion/) — substrate, ingest pipeline, HTTP T1–T4 endpoints, 24 lifecycle CLIs, projection + replay layers, verify + orphan-cleanup admin tools |

## What This Project Is About

Conventional behavioral intelligence and anti-abuse systems suffer from a class of degradation that is rarely caused by individual engineering errors. They are structural consequences of treating inference and observation as ontologically equivalent.

Ghost Trace is an attempt to design, from constitutional principles, a substrate that does not have this problem. The system maintains structural separation between observation, operational construct, and inferred hypothesis. It preserves provenance — both observational and inferential — as first-class structure rather than as metadata. It distinguishes confidence from evidential independence as separate dimensions of belief.

The system is designed for engineers, researchers, and operators who recognize that intelligence systems in high-consequence environments degrade silently when their epistemology is implicit.

## Document Hierarchy

Ghost Trace is constitutionally governed. Documents in this repository have explicit ranks:

1. **[Constitutional Charter](./docs/charter/constitutional-charter.md)** — authoritative. Changes require formal amendment per [`amendments.md`](./docs/charter/amendments.md).
2. **[Ontology](./docs/ontology/ontology.md)** — formalizes Charter concepts. May evolve under implementation pressure provided it does not conflict with the Charter.
3. **[Architecture](./docs/architecture/)** — translates the Ontology into operational design. Subordinate to the Charter and the Ontology.
4. **[RFCs](./docs/rfcs/)** — proposals subject to constitutional review.
5. **[Schemas](./schemas/)** — materialize the Ontology.
6. **[Services](./services/)** — implementations.

Conflicts between layers are resolved in favor of the higher-ranked document. Lower-ranked documents are revised.

## How to Read This Repository

If this is your first time here, in order:

1. [Constitutional Charter — Thesis](./docs/charter/constitutional-charter.md#1-thesis) (5 minutes).
2. [Constitutional Charter — frozen invariants §2.1 / §2.2 / §2.3 / §2.4 / §2.5](./docs/charter/constitutional-charter.md#2-constitutional-invariants) (20 minutes).
3. [Ontology — top-level](./docs/ontology/ontology.md) (5 minutes).
4. [Decision Log](./docs/charter/decision-log.md) (10 minutes; scan the most recent entries for current state).
5. [`services/ingestion/README.md`](./services/ingestion/README.md) (15 minutes; for the implementation surface).

This is sufficient to understand the project's posture. Everything else is detail.

## How to Contribute

See [`CONTRIBUTING.md`](./CONTRIBUTING.md).

The short version: Ghost Trace is more disciplined than most open-source projects in its early stages. Contributions that conflict with the Charter are rejected on procedural grounds. Contributions that the Charter does not yet address are welcome, and may motivate Charter amendments.

## License

See [`LICENSE`](./LICENSE).
