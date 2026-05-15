# Lifecycle Semantics

**Status:** Scaffold. Pending committee redaction after Invariant 2.5 is redacted.

> This document formalizes how entities in each ontological category are created, evolved, superseded, and (where applicable) dissolved. The Charter establishes hard rules about mutation ([Invariant 2.1](../charter/constitutional-charter.md#21-observational-integrity), [Invariant 2.2](../charter/constitutional-charter.md#22-epistemic-separation), and Invariant 2.5 — pending). This document specifies the lifecycle each category supports under those rules.

## Lifecycle by Category

### Observation (Category I)

- **Creation.** Observation records are committed to the primary event log. Producers are external (client SDKs, infrastructure collectors, authoritative ledgers).
- **Evolution.** None. Observations are immutable.
- **Supersession.** Not applicable. The *interpretation* of an observation may be superseded by a new assertion (in Category II or III), but the observation itself remains.
- **Dissolution.** Not applicable. Observations are preserved for the lifetime of the log.

### Operational Construct (Category II)

- **Creation.** Constructed deterministically from observations under a versioned operational definition.
- **Evolution.** None at the level of an individual construct. A new definition produces new constructs; existing constructs are not mutated.
- **Supersession.** A construct may be superseded by another construct produced under a revised definition. Both remain accessible.
- **Dissolution.** Not applicable in the substrate. A projection may stop materializing a construct, but the underlying derivation rule and its inputs remain.

### Hypothesis (Category III)

- **Creation.** A hypothesis is formed when accumulated evidence crosses a threshold defined by an inference process. Formation is itself an event in the primary log.
- **Evolution.** A hypothesis evolves through events: new evidence updates its confidence, its membership, or its scope.
- **Lifecycle operations.** Hypotheses support operations that simpler entities do not:
  - **Merge.** Two hypotheses are recognized as describing the same underlying phenomenon and are combined into one.
  - **Split.** A hypothesis is recognized as containing multiple distinct phenomena and is divided.
  - **Promotion.** A hypothesis with sufficient maturity is admitted to operational use as enrichment context.
  - **Demotion.** A previously promoted hypothesis is withdrawn from operational use.
  - **Dissolution.** A hypothesis is recognized as no longer corresponding to any underlying phenomenon and is marked as no longer active.
- **Per Invariant 2.5 (pending):** All such operations are recorded as immutable events in the primary log. The current state of a hypothesis is a projection over its history, never the result of direct mutation.

## The Promotion Mechanism

The most consequential lifecycle operation in the system is **promotion** — the transition of a hypothesis from "active inference" to "operational intelligence used as enrichment context."

The Charter's central concern about recursive belief inflation ([Charter §1](../charter/constitutional-charter.md#1-thesis)) arises specifically from how promotion is handled. The system must:

1. Subject every candidate promotion to evaluation against criteria of maturity, breadth, and confidence.
2. Record the promotion as an immutable event.
3. Ensure that every assertion subsequently formed under the influence of the promoted hypothesis carries a structural declaration of that influence (Invariant 2.4 — pending).
4. Periodically re-evaluate promoted hypotheses against fresh, independent evidence. Promotion is not permanent.

The exact mechanics of promotion criteria, evaluation cadence, and demotion rules are deferred until committee redaction of the relevant invariants.

## Open Modeling Questions

1. **Hypothesis subtypes.** Do `BehavioralCluster`, `CoordinationRing`, `CampaignHypothesis`, etc., have distinct lifecycle rules, or share a common lifecycle with category-specific parameters?
2. **Operational definition versioning.** Constructs reference the definition that produced them. When a definition is updated, do existing constructs remain valid until explicitly re-derived, or are they implicitly stale?
3. **Cross-category lifecycle interactions.** When a promoted hypothesis is demoted, what happens to operational constructs that incorporated it as enrichment? Are they re-derived, marked as stale, or left intact with a note?
4. **Independence-driven lifecycle.** Should a hypothesis whose independence score falls below a threshold be automatically considered for demotion, or is demotion always operator-driven?

<!-- TODO: After Invariant 2.5 is redacted, formalize the hypothesis lifecycle state machine with explicit transition events. -->

<!-- TODO: Coordinate with [`provenance-model.md`](./provenance-model.md) on how lifecycle events appear in the provenance graph. -->
