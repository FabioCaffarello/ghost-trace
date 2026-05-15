# Architecture

Documents in this directory specify the architectural treatment of Ghost Trace at a level above implementation and below the Constitutional Charter and Ontology.

## Subordination

These documents are subordinate to the [Constitutional Charter](../charter/constitutional-charter.md) and the [Ontology](../ontology/ontology.md). When a document in this directory conflicts with either, it is the document in this directory that is revised.

Architecture documents may evolve more rapidly than the Charter or Ontology, but they may not introduce constructs that violate constitutional invariants or contradict ontological definitions.

## Document Family

- [`event-flow.md`](./event-flow.md) — high-level flow of events through the system phases.
- [`replay-model.md`](./replay-model.md) — replay semantics, including phase-specific replay contracts.
- [`projection-model.md`](./projection-model.md) — the substrate / projection distinction and its operational consequences.
- [`storage-model.md`](./storage-model.md) — storage tiers and their constitutional status.

## Status

All architecture documents are currently scaffolds. Concrete architectural decisions are deferred until the Ontology stabilizes. Technology selections will be made via RFC, recorded in the [decision log](../charter/decision-log.md), and reflected here.
