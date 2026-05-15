# Entity Model

**Status:** Scaffold. Pending committee redaction.

> This document formalizes the three categories of knowledge introduced in [Charter §1](../charter/constitutional-charter.md#1-thesis) and structurally separated by [Invariant 2.2](../charter/constitutional-charter.md#22-epistemic-separation). It does not specify schemas; concrete schema definitions live in [`../../schemas/`](../../schemas/).

## The Three Categories

The Charter establishes three structurally distinct categories. The Ontology names and characterizes them; the schemas materialize them.

### Category I — Observation

Records of fact, recorded as immutable historical events. Observations are not interpreted; they are committed. An observation answers the question *what happened*, not *what does it mean*.

Examples (illustrative, not exhaustive):
- Session events reported by client SDKs.
- Network-level events recorded by infrastructure collectors.
- Transactions or state changes recorded by external authoritative systems.
- Fingerprint snapshots collected at well-defined moments.

Structural properties (to be formalized):
- Immutable after commit (Charter Invariant 2.1).
- Carries source attribution and timing metadata sufficient to distinguish producer time from receipt time.
- Identifier is producer-generated, content-addressable, or otherwise capable of supporting deduplication without mutation.

> _Detailed schema lives in [`../../schemas/events/`](../../schemas/events/). This document specifies category semantics; the schema specifies field-level structure._

### Category II — Operational Construct

Entities derived from explicit operational definitions over observations. Operational constructs are deterministic with respect to their input observations and their definitional parameters, but their *boundaries* are operational conventions, not facts about the world.

Examples (illustrative):
- A session reconstructed by an operational definition (e.g., "events from one actor within a 30-minute inactivity window").
- A rate-limit bucket defined over a window and a key.
- A daily-actor projection defined over a time slice and an identity reference.

Structural properties (to be formalized):
- Identity composes the definition reference, its parameters, and its time scope.
- Re-derivation under a new definition produces a new construct, never a mutation of the existing one.
- Definitions are versioned. The definition that produced a construct is part of its identity.

### Category III — Hypothesis

Entities whose very existence is an inference. Hypotheses are probabilistic constructions whose boundaries, membership, and continued existence are matters of degree, not of fact.

Examples (illustrative):
- A behavioral cluster: "these actors appear to be operated by the same entity."
- A coordination ring: "this group appears to act in concert."
- A campaign hypothesis: "these events appear to be part of a unified operation."

Structural properties (to be formalized):
- Identity is probabilistic and may evolve through merge, split, and dissolution operations.
- Every claim about a hypothesis carries both confidence and evidential independence (Charter Invariant 2.6 — pending).
- Lifecycle transitions are modeled as immutable events (Charter Invariant 2.5 — pending), never as direct state mutations.

## The Distinction Between Substrate and Projection

The categories above describe records in the **substrate**. Projections — analytical stores, graph indexes, dashboards — are not bound by the same rules. The substrate is governed by Charter invariants; projections are rebuildable from the substrate and may be discarded or recomputed.

This distinction is constitutional, not merely architectural. See [Charter Invariant 2.1, Boundary Conditions](../charter/constitutional-charter.md#21-observational-integrity).

## Open Modeling Questions

The following questions are recorded as open and intentionally not resolved here:

1. **Session duality.** Is a session a single entity with reconciliation, or two entities (`DeclaredSession` as Category I, `OperationalSession` as Category II)? The conversation that produced this Ontology recognized that the two diverge in exactly the cases where investigation matters most.
2. **Identity tiers.** The conversation introduced `ActorRef`, `Identity`, and `Cluster` as three tiers of identity. Their formalization is pending.
3. **Subtypes of hypothesis.** Whether `BehavioralCluster`, `CoordinationRing`, `CampaignHypothesis`, and `AutomationGroup` are distinct types within Category III or are tags on a single type is undecided.
4. **Subject reference polymorphism.** Assertions carry a `subject_ref` that may point to entities of any category. Whether this is a single polymorphic field or distinct fields per category is a schema-level question with ontological consequences.

These questions will be answered in committee redaction. They are not resolved here.

<!-- TODO: After Invariant 2.3 (Provenance Integrity) is redacted, expand the section on identifier semantics to specify how provenance references compose across categories. -->

<!-- TODO: After Invariant 2.5 (Hypothesis Lifecycle Explicitness) is redacted, expand the Category III section with the formal lifecycle state machine. -->

<!-- TODO: After Invariant 2.6 (Evidential Independence Integrity) is redacted, specify the structural representation of confidence and independence as paired dimensions. -->
