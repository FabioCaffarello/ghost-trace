# Storage Model

**Status:** Scaffold. Storage technology selection is intentionally deferred; see [`../charter/decision-log.md#0003`](../charter/decision-log.md).

> This document specifies the architectural treatment of storage in Ghost Trace. The Charter constrains storage by property, not by technology.

## Constitutional Anchors

- [Invariant 2.1 — Observational Integrity](../charter/constitutional-charter.md#21-observational-integrity). The primary event log must satisfy write-once semantics at the level of physical, cryptographic, or storage-enforced guarantee.

## Storage Tiers

Ghost Trace storage is conceptually organized into tiers, distinguished by their constitutional status and access characteristics:

### Tier 0 — Primary Event Log (Substrate)

The constitutional surface of storage. Bound by Invariant 2.1. Contains:

- All observations (Category I records).
- All operational construct definition events and derivation events (Category II).
- All hypothesis lifecycle events (Category III).
- All enrichment events.
- All provenance edges as first-class events.

Required properties:

- Append-only by enforced guarantee.
- Content-addressable identifiers or signed integrity proofs.
- Ordered consumption with offsets or equivalent positional addressing.
- Retention sufficient for operational replay needs.

### Tier 1 — Archive (Substrate)

The long-term home of Tier 0 content. Constitutionally equivalent to Tier 0 (still substrate, still bound by Invariant 2.1) but optimized for cost and long retention rather than for active consumption.

Required properties:

- Immutability enforced at the storage layer.
- Columnar or otherwise compressed format for efficient archival.
- Integrity verification: archive content must be verifiable against Tier 0 records.

### Tier 2 — Projections (Non-substrate)

Materialized views derived from the substrate. Not bound by Invariant 2.1. See [`projection-model.md`](./projection-model.md).

## Technology-Independence

The Charter does not name storage technologies. This is intentional. The substrate is defined by its properties, not by its vendor. Technology selection is an RFC subject. Acceptable substrates are those that satisfy the properties above; the set of acceptable substrates is not pre-decided.

Conversations leading to this document considered NATS JetStream, Kafka, object storage with Parquet, and combinations thereof. None of these is currently selected. RFCs proposing specific substrates must demonstrate satisfaction of the constitutional properties.

## Open Questions

1. **Hot/cold boundary.** Whether Tier 0 and Tier 1 are technologically distinct (e.g., streaming + object storage) or unified (e.g., a single system with tiered storage) is undecided. Both are constitutionally acceptable.
2. **Cryptographic integrity scheme.** The Charter requires "physical, cryptographic, or storage-enforced guarantee." Which of these is used in practice is a technology decision.
3. **Replay path for cold data.** How replay traverses the Tier 0 / Tier 1 boundary is a UX and architectural question.

<!-- TODO: Open an RFC proposing the primary storage substrate after Ontology v0.1 is stable. -->

<!-- TODO: Open a second RFC proposing the archive substrate. -->
