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
| [Invariant 2.4 — Inferential Influence Disclosure](../charter/constitutional-charter.md#24-inferential-influence-disclosure) | Entities formed under influence of promoted hypotheses must declare that influence. |
| [Invariant 2.5 — Hypothesis Lifecycle Explicitness](../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) | Hypothesis-state changes must be modeled as lifecycle events, not mutations. |
| [Invariant 2.6 — Evidential Independence Integrity](../charter/constitutional-charter.md#26-evidential-independence-integrity) | Inferential assertions must carry separate confidence and independence dimensions. |

Each invariant above corresponds to a concrete structural requirement in the Ontology. As invariants are redacted in committee mode, the corresponding Ontology sections are written to match them.

## Open Questions for Committee Resolution

The following questions are deferred to committee redaction. They are recorded here so that implementation work does not silently resolve them through code:

> **Status: all five questions resolved as of [`§0134`](../charter/decision-log.md) (2026-05-22).** The list below is preserved for reference; resolutions are recorded inline. Future open questions surface as ordinary RFC discipline at their respective document anchors (e.g., [`entity-model.md` Open Modeling Questions](./entity-model.md#open-modeling-questions); [`provenance-model.md` Open Modeling Questions](./provenance-model.md#open-modeling-questions)).

1. ~~Is `Session` a single entity with reconciliation, or two entities (`DeclaredSession` and `OperationalSession`)?~~ **Resolved per [`§0015`](../charter/decision-log.md): Candidate B — `DeclaredSession` (Category I) + `OperationalSession` (Category II) as distinct entities.** RFC: [`ontology-revision-q1-session-duality`](../rfcs/draft/ontology-revision-q1-session-duality.md) (accepted).
2. ~~Are `BehavioralCluster`, `CoordinationRing`, `CampaignHypothesis`, and `AutomationGroup` distinct entity types within the hypothesis category, or are they tags on a single `Hypothesis` type?~~ **Resolved per [`§0010`](../charter/decision-log.md): Candidate A.2 — four concrete subtypes (distinct entity types) under the abstract `Hypothesis` type; subtype hierarchy via inheritance.** RFC: [`ontology-revision-q2-hypothesis-subtypes`](../rfcs/draft/ontology-revision-q2-hypothesis-subtypes.md) (accepted).
3. ~~What is the formal definition of `independence` as a measurable quantity?~~ **Resolved per [`§0133`](../charter/decision-log.md): Candidate α (source-count ratio over Cat I provenance roots) adopted under [§2.6 BC2](../charter/constitutional-charter.md#26-evidential-independence-integrity) meta-shape 1 (deterministic-from-pattern).** Formula: `evidential_independence = (count of Cat I roots NOT reachable via any influenced_by edge from a promoted hypothesis) / (total Cat I roots)`. RFC: [`ontology-revision-q3-independence`](../rfcs/draft/ontology-revision-q3-independence.md) (accepted).
4. ~~When does a promoted hypothesis become a candidate for demotion?~~ **Resolved per [`§0011`](../charter/decision-log.md): staged-combination form — Layer A (time-based cadence gate, AND-composed) plus Layer B (deep criterion on `evidential independence` and/or declared `influence`, deferred to [Layer B follow-on RFC](../rfcs/draft/ontology-revision-layer-b-deep-criterion.md) — all 4 dependencies discharged as of §0134; substantive content drafting opens).** RFC: [`ontology-revision-q4-promotion-demotion-criterion`](../rfcs/draft/ontology-revision-q4-promotion-demotion-criterion.md) (accepted).
5. ~~How does `influence` propagate through derived assertions? Transitive? Decaying? Both?~~ **Resolved fully: decay half at [`§0020`](../charter/decision-log.md) (Candidate C — via §2.5 lifecycle event supersession); transitivity half at [`§0134`](../charter/decision-log.md) (Candidate τ — transitive closure of declared direct `influenced_by` edges, with β-graph storage: substrate stores direct edges + per-record cached closures).** Committee extension: Cat II constructs structurally transmit `influenced_by` membership from their inputs per [§2.2](../charter/constitutional-charter.md#22-epistemic-separation) determinism. RFCs: [`ontology-revision-omq2-decay-of-influence`](../rfcs/draft/ontology-revision-omq2-decay-of-influence.md) (accepted) + [`ontology-revision-q5-influence-propagation-transitivity`](../rfcs/draft/ontology-revision-q5-influence-propagation-transitivity.md) (accepted).

## Status

| Section | Status |
|---|---|
| Scope and document family | Drafted |
| Constitutional anchors | Drafted |
| [Entity model](./entity-model.md) | Drafted (Cat I + II + III post-Q1/Q2/Q3 per [`§0010`](../charter/decision-log.md) + [`§0015`](../charter/decision-log.md) + [`§0016`](../charter/decision-log.md); other sections at Scaffold strength) |
| [Provenance model](./provenance-model.md) | Drafted (§Observational Provenance post-Q3 per [`§0016`](../charter/decision-log.md) + [`§0017`](../charter/decision-log.md); §Inferential Provenance post-OMQ #2 + OMQ #3 per [`§0020`](../charter/decision-log.md) + [`§0021`](../charter/decision-log.md) + [`§0099`](../charter/decision-log.md); Q5 transitivity per [`§0134`](../charter/decision-log.md)) |
| [Lifecycle semantics](./lifecycle-semantics.md) | Drafted (Cat III + §The Promotion Mechanism post-Q4 + Layer B per [`§0011`](../charter/decision-log.md) + [`§0135`](../charter/decision-log.md) + [`§0138`](../charter/decision-log.md); other sections at Scaffold strength) |
| Replay semantics | Not yet created |

<!-- Charter v0.7 fully frozen per `decision-log.md` §0131. Committee redaction of [`entity-model.md`](./entity-model.md) is complete (Drafted per `docs/architecture/README.md` Status); [`provenance-model.md`](./provenance-model.md) and [`lifecycle-semantics.md`](./lifecycle-semantics.md) likewise advanced to Drafted strength on their constitutional-anchor sections (per Status banners). -->
