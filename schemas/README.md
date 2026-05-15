# Schemas

This directory contains the formal schemas that materialize the Ontology.

## Status

No concrete schemas exist yet. Schema definition is gated on committee redaction of the Ontology. The directory structure is established in advance to signal that schemas are first-class artifacts, not implementation detail.

## Directory Layout

- [`events/`](./events/) — schemas for observation records (Category I).
- [`assertions/`](./assertions/) — schemas for assertions over operational constructs (Category II) and hypotheses (Category III).
- [`provenance/`](./provenance/) — schemas for provenance edges and supporting structures.

## Constitutional Anchors

Schemas are bound by:

- [Invariant 2.1 — Observational Integrity](../docs/charter/constitutional-charter.md#21-observational-integrity). Event schemas must support immutable commitment.
- [Invariant 2.2 — Epistemic Separation](../docs/charter/constitutional-charter.md#22-epistemic-separation). Schemas across categories must be structurally distinct; a single unified record schema with a discriminator field is forbidden.
- Invariants 2.3–2.6 (pending). Schemas must support the structural requirements of the remaining invariants once redacted.

## Schema Technology

Not yet selected. Candidate technologies include Protocol Buffers, Avro, JSON Schema, and combinations. The choice is an RFC subject.

<!-- TODO: Open an RFC proposing schema technology after Invariants 2.3–2.6 are redacted. -->
