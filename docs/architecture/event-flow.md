# Event Flow

**Status:** Scaffold. Architecture documents are subordinate to the Charter and to the Ontology. Concrete event flow design follows committee redaction of the Ontology.

> This document describes the flow of events through Ghost Trace at the architectural level. It does not specify technology choices; those are RFC subjects.

## Constitutional Anchors

Event flow design is constrained by:

- [Invariant 2.1 — Observational Integrity](../charter/constitutional-charter.md#21-observational-integrity). The primary event log is the substrate. All other components are projections.
- [Invariant 2.2 — Epistemic Separation](../charter/constitutional-charter.md#22-epistemic-separation). Events of different categories flow through structurally distinct paths.
- [Invariant 2.3 — Provenance Integrity](../charter/constitutional-charter.md#23-provenance-integrity) (frozen v0.4 per [`decision-log §0017`](../charter/decision-log.md)), [§2.4 — Inferential Influence Disclosure](../charter/constitutional-charter.md#24-inferential-influence-disclosure) (frozen v0.5 per [`§0099`](../charter/decision-log.md)), [§2.5 — Hypothesis Lifecycle Explicitness](../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) (frozen v0.3 per [`§0013`](../charter/decision-log.md)), and [§2.6 — Evidential Independence Integrity](../charter/constitutional-charter.md#26-evidential-independence-integrity) (frozen v0.6 per [`§0129`](../charter/decision-log.md)) further constrain how derived events are tagged with provenance and influence.

## High-Level Phases

Event flow in Ghost Trace proceeds through phases corresponding to the epistemic phases discussed in the Charter:

### Phase 0 — Production

Observations originate at producers: client SDKs, infrastructure collectors, authoritative external systems, and simulated adversarial agents. Producers are responsible for assigning identifiers and producer-side timestamps. Producers do not interpret.

### Phase 1 — Commitment

Observations are committed to the primary event log. Commitment is the moment the system takes responsibility for the record. After commitment, the record is governed by Invariant 2.1.

### Phase 2 — Enrichment

Observations are paired with enrichment records that capture operational knowledge available at the time of enrichment. Enrichment is a separate stream of immutable events, not a mutation of observations. The pairing is by reference.

### Phase 3 — Signal and construct derivation

Operational constructs (Category II) are derived from observations and enrichment under explicit definitions. Each derived record carries its observational provenance.

### Phase 4 — Hypothesis formation and evolution

Hypotheses (Category III) are formed and evolved through inference processes that operate over signals, constructs, and prior hypotheses. Each operation on a hypothesis is itself an event in the primary event log per [Invariant 2.5](../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) (frozen v0.3 per [`decision-log §0013`](../charter/decision-log.md)).

### Phase 5 — Projection

Derived state is materialized into projections for operational and analytical use. Projections are not part of the substrate and are not bound by Invariant 2.1.

## Open Architectural Questions

1. **Hot vs. cold substrate.** The conversation that produced this scaffold considered a split between operational streaming (hot) and object-storage archive (cold). Whether these are architecturally distinct or unified under one technology is an open question. See [`storage-model.md`](./storage-model.md).
2. **Enrichment latency handling.** Phase 3 may consume observations whose enrichment has not yet arrived. The strategies (wait, process and reprocess, process twice) have different implications.
3. **Backpressure semantics.** How does the system behave when downstream phases lag producers? Charter invariants must hold under all backpressure conditions.

<!-- TODO: After storage technology is selected via RFC, replace the abstract phase descriptions with concrete event-flow diagrams. -->

<!-- TODO: Coordinate with [`replay-model.md`](./replay-model.md) on how phases are replayed under different forensic conditions. -->
