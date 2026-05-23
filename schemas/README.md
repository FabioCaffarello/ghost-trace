# Schemas

This directory contains the formal schemas that materialize the Ontology.

## Status

Active. Schemas technology selected per [`decision-log §0024`](../docs/charter/decision-log.md): Protocol Buffers (proto3). The Ontology has stabilized as Drafted per [`docs/architecture/README.md`](../docs/architecture/README.md). Concrete schemas exist under [`events/v1/`](./events/v1/) (30+ Category I + Category II + Category III protos including formation/merge/split/promotion/demotion/dissolution lifecycle events per [Charter §2.5](../docs/charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) frozen v0.3) and [`common/v1/`](./common/v1/) (shared types — `EvidentialIndependence` rational-pair, `LayerBParameters` per [`§0136`](../docs/charter/decision-log.md) + [`§0138`](../docs/charter/decision-log.md)). The directory structure preserves the original category-organized intent; in practice every record is materialized as an event in `events/v1/` (events are the substrate's single record form per [Charter §2.1](../docs/charter/constitutional-charter.md#21-observational-integrity)).

## Directory Layout

- [`events/`](./events/) — proto schemas for substrate event records across all three categories. Per the canonical-serialization-contract per [`§0136`](../docs/charter/decision-log.md), every substrate record is an event whose content-addressable identifier is derived from its canonical Protobuf serialization.
- [`common/`](./common/) — proto schemas for shared composite types referenced across event protos (`EvidentialIndependence`, `LayerBParameters`).
- [`assertions/`](./assertions/) — organizational placeholder; the structural commitments these protos would materialize are presently inlined in `events/v1/` Cat II + Cat III protos. See the directory's own README for details.
- [`provenance/`](./provenance/) — organizational placeholder; provenance-edge structural commitments are presently inlined in `events/v1/` per [`§0136`](../docs/charter/decision-log.md). See the directory's own README for details.

## Constitutional Anchors

Schemas are bound by:

- [Invariant 2.1 — Observational Integrity](../docs/charter/constitutional-charter.md#21-observational-integrity). Event schemas must support immutable commitment.
- [Invariant 2.2 — Epistemic Separation](../docs/charter/constitutional-charter.md#22-epistemic-separation). Schemas across categories must be structurally distinct; a single unified record schema with a discriminator field is forbidden.
- [Invariant 2.3 — Provenance Integrity](../docs/charter/constitutional-charter.md#23-provenance-integrity) (frozen v0.4 per [`decision-log §0017`](../docs/charter/decision-log.md)), [§2.4 — Inferential Influence Disclosure](../docs/charter/constitutional-charter.md#24-inferential-influence-disclosure) (frozen v0.5 per [`§0099`](../docs/charter/decision-log.md)), [§2.5 — Hypothesis Lifecycle Explicitness](../docs/charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) (frozen v0.3 per [`§0013`](../docs/charter/decision-log.md)), and [§2.6 — Evidential Independence Integrity](../docs/charter/constitutional-charter.md#26-evidential-independence-integrity) (frozen v0.6 per [`§0129`](../docs/charter/decision-log.md)). Schemas must support the structural requirements of these invariants.

## Schema Technology

Selected per [`decision-log §0024`](../docs/charter/decision-log.md): Protocol Buffers (proto3). The selection was made under the inception-phase RFC discipline that opened after the implementation-gate per [CLAUDE.md §6.4](../.claude/CLAUDE.md) cleared. Per-field conventions + canonical-serialization contract per [`docs/architecture/canonical-serialization-contract.md`](../docs/architecture/canonical-serialization-contract.md).

<!-- Schemas technology RFC accepted at `decision-log.md` §0024 (Protocol Buffers proto3). The choice is no longer an open RFC subject. Future schemas-evolution events proceed under the canonical-serialization-contract §Schemas-Evolution Events boundary. -->
