# Ghost Trace

A behavioral intelligence system designed to preserve the epistemic integrity of operational knowledge — the continued capacity to distinguish what was observed from what was inferred — as that knowledge accumulates, evolves, and is acted upon over time.

> Ghost Trace is not a detector that happens to be auditable. It is a behavioral intelligence substrate within which detection is the first applied domain.
>
> — [Constitutional Charter, §1](./docs/charter/constitutional-charter.md#1-thesis)

## Status

This repository is in **pre-implementation constitutional drafting**. The Constitutional Charter is being redacted in committee mode, one section at a time. Implementation work begins after the Charter and the relevant portions of the Ontology are stable.

Current state:

| Document | Status |
|---|---|
| [Constitutional Charter](./docs/charter/constitutional-charter.md) | `v0.1` — Thesis and Invariants 2.1, 2.2 frozen; remainder in committee |
| [Ontology](./docs/ontology/ontology.md) | Scaffold; awaits Charter completion |
| Architecture | Scaffolds; await Ontology |
| Schemas | Placeholders; await Ontology and schema-technology RFC |
| Services | Placeholders; await all of the above |

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
2. [Constitutional Charter — Invariants 2.1 and 2.2](./docs/charter/constitutional-charter.md#2-constitutional-invariants) (10 minutes).
3. [Ontology — top-level](./docs/ontology/ontology.md) (5 minutes).
4. [Decision Log](./docs/charter/decision-log.md) (5 minutes).

This is sufficient to understand the project's posture. Everything else is detail.

## How to Contribute

See [`CONTRIBUTING.md`](./CONTRIBUTING.md).

The short version: Ghost Trace is more disciplined than most open-source projects in its early stages. Contributions that conflict with the Charter are rejected on procedural grounds. Contributions that the Charter does not yet address are welcome, and may motivate Charter amendments.

## License

See [`LICENSE`](./LICENSE).
