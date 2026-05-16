# Entity Model

**Status:** Drafted — Category III revised post-Q2 ([`decision-log.md` §0010](../charter/decision-log.md)); Category I and Category II revised post-Q1 ([`decision-log.md` §0015](../charter/decision-log.md)); other sections remain Scaffold.

> This document formalizes the three categories of knowledge introduced in [Charter §1](../charter/constitutional-charter.md#1-thesis) and structurally separated by [Invariant 2.2](../charter/constitutional-charter.md#22-epistemic-separation). It does not specify schemas; concrete schema definitions live in [`../../schemas/`](../../schemas/).

## The Three Categories

The Charter establishes three structurally distinct categories. The Ontology names and characterizes them; the schemas materialize them.

### Category I — Observation

Records of fact, recorded as immutable historical events. Observations are not interpreted; they are committed. An observation answers the question *what happened*, not *what does it mean*.

Examples (illustrative, not exhaustive):
- **`DeclaredSession`**: session boundaries as reported by client SDKs (primary observation at the moment of session-end reporting). Canonical Category I type for the declared form of a session per [`decision-log.md` §0015](../charter/decision-log.md) (Q1 resolution).
- Network-level events recorded by infrastructure collectors.
- Transactions or state changes recorded by external authoritative systems.
- Fingerprint snapshots collected at well-defined moments.

Per [`decision-log.md` §0015](../charter/decision-log.md) (Q1 resolution), `Session` as a domain concept resolves to two distinct entity-model types: `DeclaredSession` (Category I, defined here) and `OperationalSession` (Category II, defined below). The two diverge in the cases where investigation matters most; the type-level visibility of the divergence is intentional and structural.

Structural properties (to be formalized):
- Immutable after commit (Charter Invariant 2.1).
- Carries source attribution and timing metadata sufficient to distinguish producer time from receipt time.
- Identifier is producer-generated, content-addressable, or otherwise capable of supporting deduplication without mutation.

> _Detailed schema lives in [`../../schemas/events/`](../../schemas/events/). This document specifies category semantics; the schema specifies field-level structure._

### Category II — Operational Construct

Entities derived from explicit operational definitions over observations. Operational constructs are deterministic with respect to their input observations and their definitional parameters, but their *boundaries* are operational conventions, not facts about the world.

**`OperationalSession`** is a canonical Category II operational construct per [`decision-log.md` §0015](../charter/decision-log.md) (Q1 resolution): the system's reading of where a session operationally was, derived from `DeclaredSession` (Category I) + other Category I inputs (typically network-level events and fingerprint snapshots) under a versioned operational definition. The derivation is deterministic per §2.2's Category II requirement; non-deterministic derivation would constitute a Category III misclassification and would be rejected at validation. `OperationalSession` may diverge from its source `DeclaredSession` in boundary timing, in attributed identity (subject to identity-tier consistency rules per [`decision-log.md` §0015](../charter/decision-log.md)), or in scope; the divergences are themselves first-class structural signals.

Identity-tier references on `DeclaredSession` and `OperationalSession` default to the same `ActorRef` / `Identity` / `Cluster` references (per Q1 of §Open Modeling Questions below — Identity tiers, pending); the operational definition may explicitly override identity-tier attribution where the operational reading differs from the declared. The default is procedural until Identity tiers is formally resolved.

Examples (illustrative):
- **`OperationalSession`**: per the canonical definition above (e.g., operational definition "events from one actor within a 30-minute inactivity window").
- A rate-limit bucket defined over a window and a key.
- A daily-actor projection defined over a time slice and an identity reference.

Structural properties (to be formalized):
- Identity composes the definition reference, its parameters, and its time scope.
- Re-derivation under a new definition produces a new construct, never a mutation of the existing one.
- Definitions are versioned. The definition that produced a construct is part of its identity.

### Category III — Hypothesis

Entities whose very existence is an inference. Hypotheses are probabilistic constructions whose boundaries, membership, and continued existence are matters of degree, not of historical fact. Category III is structured as an abstract type `Hypothesis` with four concrete subtypes, resolved by [`decision-log.md` §0010 — Q2 resolution](../charter/decision-log.md).

#### Abstract type — `Hypothesis`

The abstract type carries the structure shared by all hypothesis subtypes. It is not itself a concrete record type — no record exists whose declared type is `Hypothesis`. The abstract type captures shared structure that the four concrete subtypes inherit:

- Stable identifier under provenance discipline ([Charter §2.3](../charter/constitutional-charter.md#23-provenance-integrity) pending; reference shape stabilizes when §2.3 is redacted).
- Lifecycle position under [Charter §2.5](../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness). The six lifecycle operations (formation, merge, split, promotion, demotion, dissolution) are defined at the abstract-type level; subtype-specific composition rules surface in the subtype sections below.
- Confidence and evidential independence ([Charter §2.6](../charter/constitutional-charter.md#26-evidential-independence-integrity) pending) — paired dimensions; see §2.6's redaction.
- Inferential influence ([Charter §2.4](../charter/constitutional-charter.md#24-inferential-influence-disclosure) pending) — every assertion formed under a hypothesis's influence carries a structural declaration of that influence.

#### Concrete subtypes

The four subtypes are sibling concrete types extending `Hypothesis`. Each carries subtype-specific structure beyond the shared abstract structure. Subtype-specific structural constraints are enforceable in the type system per [Charter §4 criterion 1](../charter/constitutional-charter.md#4-constitutional-design-rule) (structural enforceability); they do not migrate to runtime checks.

- **`BehavioralCluster`** — a set of actors whose behavioral patterns suggest operation by a common underlying entity. Subtype-specific surface includes actor membership representation and pattern-signature reference. The inference is about shared operatorship.
- **`CoordinationRing`** — a set of actors whose patterns of interaction with one another suggest coordinated action. Subtype-specific surface includes pairwise relationship records and temporal alignment references. Distinguished from `BehavioralCluster` by the relational nature of the inference: ring is about coordination among potentially distinct operators, not shared operatorship.
- **`CampaignHypothesis`** — a set of events whose patterns suggest membership in a unified operation. Subtype-specific surface includes event membership representation and thematic-coherence reference. The inference is over events, not actors.
- **`AutomationGroup`** — a set of actors whose behavioral patterns match a signature of automated (non-human) operation. Subtype-specific surface includes signature reference and detected-pattern records. The inference is about the operation's character (automated), not about shared operatorship.

The subtype distinction is structurally recorded: a hypothesis record's concrete type determines its subtype, and subtype-specific constraints (membership shapes, required references, validation rules) are type-system-enforced. Subtype membership is not a label on a uniform type; it is a structural commitment.

> _Detailed concrete-type definitions live in [`../../schemas/`](../../schemas/) once the substrate-technology selection ([`decision-log.md` §0003](../charter/decision-log.md)) is resolved. This document specifies subtype semantics; the type definitions specify field-level structure._

#### Cross-subtype operations

Cross-subtype merge (e.g., a `BehavioralCluster` and a `CoordinationRing` recognized as the same underlying phenomenon) is structurally permitted but requires a typed transformation: the merge operation produces a typed output record. Whether the produced record is a third concrete subtype or an abstract record with subtype-elision is a question whose resolution is deferred to [`lifecycle-semantics.md`](./lifecycle-semantics.md) post-Q4 redaction.

## The Distinction Between Substrate and Projection

The categories above describe records in the **substrate**. Projections — analytical stores, graph indexes, dashboards — are not bound by the same rules. The substrate is governed by Charter invariants; projections are rebuildable from the substrate and may be discarded or recomputed.

This distinction is constitutional, not merely architectural. See [Charter Invariant 2.1, Boundary Conditions](../charter/constitutional-charter.md#21-observational-integrity).

## Open Modeling Questions

The following questions are recorded as open and intentionally not resolved here:

1. **Identity tiers.** The conversation introduced `ActorRef`, `Identity`, and `Cluster` as three tiers of identity. Their formalization is pending.
2. **Subject reference polymorphism.** Assertions carry a `subject_ref` that may point to entities of any category. Whether this is a single polymorphic field or distinct fields per category is a type-level question with ontological consequences. Resolution opened as RFC [`ontology-revision-q3-subject-ref-polymorphism`](../rfcs/draft/ontology-revision-q3-subject-ref-polymorphism.md) (`discussion` status) per [`decision-log.md` §0015](../charter/decision-log.md) cascade trigger.

These questions will be answered in committee redaction. They are not resolved here.

## Resolved Modeling Questions

The following questions were recorded as open and have since been resolved by committee. Each entry preserves the question and links to the decision-log entry that records the resolution.

- **Subtypes of hypothesis** (formerly Open Modeling Question 3). Whether `BehavioralCluster`, `CoordinationRing`, `CampaignHypothesis`, and `AutomationGroup` are distinct types within Category III or are values of a single discriminator. **Resolved** by [`decision-log.md` §0010 — Q2 resolution](../charter/decision-log.md): Candidate A.2 (abstract type `Hypothesis` with four concrete sibling subtypes). The Category III section above reflects the resolution.
- **Session duality** (formerly Open Modeling Question 1). Whether a session is a single entity with reconciliation, or two entities (`DeclaredSession` as Category I, `OperationalSession` as Category II). **Resolved** by [`decision-log.md` §0015 — Q1 resolution](../charter/decision-log.md): Candidate B (distinct entities). The Category I and Category II sections above reflect the resolution; subject_ref polymorphism implications are addressed in the cascade Q3 RFC (`ontology-revision-q3-subject-ref-polymorphism`, `discussion` status per §0015).

<!-- TODO: After Invariant 2.3 (Provenance Integrity) is redacted, expand the section on identifier semantics to specify how provenance references compose across categories. -->

<!-- TODO: After Invariant 2.5 (Hypothesis Lifecycle Explicitness) is redacted, expand the Category III section with the formal lifecycle state machine. -->

<!-- TODO: After Invariant 2.6 (Evidential Independence Integrity) is redacted, specify the structural representation of confidence and independence as paired dimensions. -->
