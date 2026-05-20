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
- [`canonical-serialization-contract.md`](./canonical-serialization-contract.md) — the bit-stable mapping from a Protobuf message to canonical bytes to a BLAKE3 content-addressable identifier. The falsifiability predicate for [Charter §2.1](../charter/constitutional-charter.md#21-observational-integrity) at the substrate.
- [`concurrency-pattern.md`](./concurrency-pattern.md) — goroutine lifecycle + channel ownership + context propagation discipline; substrate-writer serialization derived from [`§0027`](../charter/decision-log.md) F3 single-writer constraint.

## Status

The Ontology has stabilized as Drafted, and the three inception-phase technology RFCs are accepted: [`§0024`](../charter/decision-log.md) (schemas — Protocol Buffers proto3), [`§0025`](../charter/decision-log.md) (language — Go), [`§0027`](../charter/decision-log.md) (storage — SQLite + content-addressed blob-store). The implementation-gate per [CLAUDE.md §6.4](../../.claude/CLAUDE.md) is cleared. Architecture documents are transitioning from scaffold to active as concrete decisions consume the technology selections. Active documents: [`canonical-serialization-contract.md`](./canonical-serialization-contract.md) (per [`§0024`](../charter/decision-log.md) AP5 step b + [`§0028`](../charter/decision-log.md)) and [`concurrency-pattern.md`](./concurrency-pattern.md) (per [`§0025`](../charter/decision-log.md) modification 5 + [`§0029`](../charter/decision-log.md)). The four scaffold documents above remain scaffolds pending further concrete work.
