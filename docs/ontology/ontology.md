# Ghost Trace Ontology

**Status:** Scaffold. Pending committee redaction after Charter `v0.1` is complete.

> This document formalizes the concepts introduced in the [Constitutional Charter](../charter/constitutional-charter.md). It is subordinate to the Charter; where this document conflicts with the Charter, the Charter prevails and this document is revised.
>
> The Ontology is the model against which every implementation feature is checked. It must remain compatible with the Charter at all times, but is permitted to evolve under implementation pressure in ways the Charter is not.

## Scope of this Document

The Ontology formalizes:

1. The three categories of knowledge introduced in [Charter §1](../charter/constitutional-charter.md#1-thesis) and structurally separated by [Invariant 2.2](../charter/constitutional-charter.md#22-epistemic-separation).
2. The assertion taxonomy that operates over those categories.
3. The semantics of provenance — both observational and inferential.
4. The lifecycle rules for each category, with particular attention to the lifecycle of hypotheses.
5. The semantics of replay — what reconstruction means at each epistemic phase.
6. The structural distinction between confidence and evidential independence as dimensions of inferential assertions.
7. The boundary between substrate (governed by Charter invariants) and projection (rebuildable from substrate).

The Ontology does not specify:

- Concrete schemas (those live in [`../../schemas/`](../../schemas/), subordinate to this document).
- Concrete technology choices (those are RFC subjects, not Ontology concerns).
- Operational tooling (subordinate to architecture documents).

## Document Family

- [`entity-model.md`](./entity-model.md) — the three categories of knowledge, formalized as entity types.
- [`provenance-model.md`](./provenance-model.md) — observational and inferential provenance as a graph structure.
- [`lifecycle-semantics.md`](./lifecycle-semantics.md) — how entities are created, evolved, superseded, and (where applicable) dissolved.

A future document, `replay-semantics.md`, is anticipated to formalize the phase-specific replay guarantees alluded to in [Charter §1](../charter/constitutional-charter.md#1-thesis).

## Constitutional Anchors

The Ontology must, at minimum, satisfy the following Charter requirements:

| Charter element | Ontology requirement |
|---|---|
| [Invariant 2.1 — Observational Integrity](../charter/constitutional-charter.md#21-observational-integrity) | The observation category must be modeled as immutable. |
| [Invariant 2.2 — Epistemic Separation](../charter/constitutional-charter.md#22-epistemic-separation) | The three categories must occupy distinct types with distinct operations. |
| [Invariant 2.3 — Provenance Integrity](../charter/constitutional-charter.md#23-provenance-integrity) | Every Assertion declares observational provenance via the typed `subject_ref_*` field; the reference chain terminates at Category I primary observations. |
| Invariant 2.4 — Inferential Influence Disclosure (pending) | Entities formed under influence of promoted hypotheses must declare that influence. |
| [Invariant 2.5 — Hypothesis Lifecycle Explicitness](../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) | Hypothesis-state changes must be modeled as lifecycle events, not mutations. |
| Invariant 2.6 — Evidential Independence Integrity (pending) | Inferential assertions must carry separate confidence and independence dimensions. |

Each invariant above corresponds to a concrete structural requirement in the Ontology. As invariants are redacted in committee mode, the corresponding Ontology sections are written to match them.

## Open Questions for Committee Resolution

The following questions are deferred to committee redaction. They are recorded here so that implementation work does not silently resolve them through code:

1. Is `Session` a single entity with reconciliation, or two entities (`DeclaredSession` and `OperationalSession`)? Discussed in conversation but not yet decided.
2. Are `BehavioralCluster`, `CoordinationRing`, `CampaignHypothesis`, and `AutomationGroup` distinct entity types within the hypothesis category, or are they tags on a single `Hypothesis` type? Decision affects the schema surface.
3. What is the formal definition of `independence` as a measurable quantity? Conceptually agreed; operationally undefined. Opened for discussion as RFC [`ontology-revision-q3-independence`](../rfcs/draft/ontology-revision-q3-independence.md) per [`decision-log.md` §0132](../charter/decision-log.md); six candidate measurable-quantity families enumerated in [`q3-independence-evidence.md`](../rfcs/discussion/q3-independence-evidence.md) Phase 2.
4. When does a promoted hypothesis become a candidate for demotion? Lifecycle rule.
5. How does `influence` propagate through derived assertions? Transitive? Decaying? Both?

These questions are intentionally not answered here. They will be resolved as the corresponding sections of the Ontology are written in committee mode.

## Status

| Section | Status |
|---|---|
| Scope and document family | Drafted |
| Constitutional anchors | Drafted |
| [Entity model](./entity-model.md) | Scaffold |
| [Provenance model](./provenance-model.md) | Scaffold |
| [Lifecycle semantics](./lifecycle-semantics.md) | Scaffold |
| Replay semantics | Not yet created |

<!-- TODO: After Charter v0.1 is frozen, begin committee redaction of entity-model.md as the first concrete Ontology document. -->
