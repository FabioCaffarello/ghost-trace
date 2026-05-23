# Provenance Schemas

Schemas for provenance edges and supporting structures.

## Status

No dedicated provenance-edge protos exist in this directory. [Invariant 2.3 — Provenance Integrity](../../docs/charter/constitutional-charter.md#23-provenance-integrity) is frozen v0.4 per [`decision-log §0017`](../../docs/charter/decision-log.md); [§2.4 — Inferential Influence Disclosure](../../docs/charter/constitutional-charter.md#24-inferential-influence-disclosure) is frozen v0.5 per [`§0099`](../../docs/charter/decision-log.md). The structural requirements both invariants codify (`derived_from` observational edges via typed `subject_ref_*` references per [`§0016`](../../docs/charter/decision-log.md); `influenced_by` inferential edges committed at substrate event time per [`§0021`](../../docs/charter/decision-log.md); transitive scope per [`§0134`](../../docs/charter/decision-log.md); decay via §2.5 lifecycle event supersession per [`§0020`](../../docs/charter/decision-log.md)) are presently materialized inline in the [`events/v1/`](../events/v1/) Category III formation protos and the canonical-serialization-contract closure-storage shape per [`decision-log §0136`](../../docs/charter/decision-log.md). Dedicated provenance-edge protos may surface here as a future schema-evolution event if a future RFC factors them out of the inline form; until then, this directory remains an organizational placeholder.

## Required Properties

When defined, provenance schemas must support:

- Two distinct edge types: `derived_from` (observational) and `influenced_by` (inferential).
- References that survive replay: edges must remain navigable when the substrate is replayed from the primary event log.
- Versioning of the derivation logic that produced each edge.

See [`../README.md`](../README.md) for constitutional anchors and [`../../docs/ontology/provenance-model.md`](../../docs/ontology/provenance-model.md) for the conceptual model.
