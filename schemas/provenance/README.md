# Provenance Schemas

Schemas for provenance edges and supporting structures.

## Status

Not yet defined. Awaiting committee redaction of Invariants 2.3 (Provenance Integrity) and 2.4 (Inferential Influence Disclosure).

## Required Properties

When defined, provenance schemas must support:

- Two distinct edge types: `derived_from` (observational) and `influenced_by` (inferential).
- References that survive replay: edges must remain navigable when the substrate is replayed from the primary event log.
- Versioning of the derivation logic that produced each edge.

See [`../README.md`](../README.md) for constitutional anchors and [`../../docs/ontology/provenance-model.md`](../../docs/ontology/provenance-model.md) for the conceptual model.
