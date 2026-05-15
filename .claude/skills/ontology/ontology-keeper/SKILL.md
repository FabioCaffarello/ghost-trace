---
name: ontology-keeper
description: Keep the Ontology consistent with the Charter and surface the five open modeling questions whenever a proposed change touches them. Use this skill ALWAYS when editing under docs/ontology/, ALWAYS when authoring an RFC of type ontology-revision, and ALWAYS when a Charter amendment affects entity definitions. Critical: an edit that silently picks an answer to an open question is a constitutional move — this skill exists to make those moves explicit.
---

# ontology-keeper

The [Ontology](../../../../docs/ontology/ontology.md) formalizes the concepts introduced in the [Charter](../../../../docs/charter/constitutional-charter.md). It is subordinate to the Charter; conflicts resolve in the Charter's favor and the Ontology is revised ([`decision-log.md` §0004](../../../../docs/charter/decision-log.md)).

The Ontology carries five open modeling questions not yet resolved by committee. An Ontology edit that silently picks an answer to any of them is a constitutional move dressed as editorial work. This skill exists to prevent that.

## 1. Registry of open Ontology questions

Source: [`docs/ontology/ontology.md` §Open Questions for Committee Resolution](../../../../docs/ontology/ontology.md).

The registry below is the canonical list used by this skill. If `ontology.md` is updated, this registry is updated in the same change. Drift between the two is a maintenance bug, recorded in `docs/charter/decision-log.md` when discovered.

The questions are reproduced **verbatim**. Do not paraphrase. Do not "improve."

1. Is `Session` a single entity with reconciliation, or two entities (`DeclaredSession` and `OperationalSession`)? Discussed in conversation but not yet decided.
2. Are `BehavioralCluster`, `CoordinationRing`, `CampaignHypothesis`, and `AutomationGroup` distinct entity types within the hypothesis category, or are they tags on a single `Hypothesis` type? Decision affects the schema surface.
3. What is the formal definition of `independence` as a measurable quantity? Conceptually agreed; operationally undefined.
4. When does a promoted hypothesis become a candidate for demotion? Lifecycle rule.
5. How does `influence` propagate through derived assertions? Transitive? Decaying? Both?

## 2. Procedure for any proposed Ontology edit

Apply in order. Do not skip steps.

### Step 1 — List Charter invariants the edit touches

Identify which invariants the edit interacts with — frozen and pending. Use [`.claude/CLAUDE.md` §4](../../../CLAUDE.md) as the source of truth for frozen-vs-pending status.

- For frozen invariants (§1, §2.1, §2.2): the edit must be consistent. Inconsistency with a frozen invariant is grounds for rejection, or for opening a charter-amendment RFC under [`amendments.md`](../../../../docs/charter/amendments.md).
- For pending invariants (§2.3–§2.6, §3, §4): the edit must be consistent with the working text and must not commit to a position the working text leaves open.

### Step 2 — List open Ontology questions the edit touches

For each of the five questions, ask: does the edit's text, its examples, or its structural commitments depend on a specific answer?

- If no open question is touched, proceed to Step 4.
- If any open question is touched, proceed to Step 3.

### Step 3 — STOP

The edit cannot proceed as a silent resolution. Surface the implicit resolution to the human, explicitly:

- Which open question the edit touches.
- Which answer the edit picks, verbatim from the proposed text.
- The Charter sections that constrain the choice.
- Whether the choice is consistent with all frozen invariants.

The human decides whether to:

- Raise an RFC of type `ontology-revision` to formally resolve the question.
- Revise the edit to remain neutral on the question.
- Defer the edit until committee resolution.

Do not proceed without a recorded decision.

### Step 4 — Propose

If no frozen invariant is violated and no open question is implicitly resolved, the edit is procedurally clear. It still requires the normal review for vocabulary discipline, falsifiability (`falsifiability-check`), and epistemic separation (`epistemic-separator`).

## 3. Examples of implicit resolution

These examples are illustrative of the failure mode this skill prevents. The phrasing is plausible; the resolution is silent.

### Touching Question 1

> "A session is a single entity reconciled from declared and operational sources."

Resolves Question 1 in favor of the single-entity-with-reconciliation answer. **Verdict:** stop. Surface as implicit resolution.

> "A session, regardless of whether it is later modeled as one entity or two, ..."

Neutral on Question 1. **Verdict:** acceptable.

### Touching Question 2

> "A CoordinationRing is a subtype of Hypothesis."

Resolves Question 2 in favor of distinct entity types. **Verdict:** stop.

> "A CoordinationRing is an instance of a hypothesis category whose specialization is undecided (Open Modeling Question 2)."

Neutral. **Verdict:** acceptable.

### Touching Question 3

> "Independence is computed as 1 minus the fraction of influence-edges in the provenance graph."

Resolves Question 3 in favor of a specific formula. **Verdict:** stop.

> "Independence is the structural property the system is required to expose; its formal definition is pending committee resolution (Open Modeling Question 3)."

Neutral. **Verdict:** acceptable.

### Touching Question 4

> "A promoted hypothesis becomes a candidate for demotion when its confidence drops below 0.4 for fourteen days."

Resolves Question 4 with specific demotion criteria. **Verdict:** stop.

> "Demotion criteria are committee work (Open Modeling Question 4); the system records the data that any candidate criterion would require."

Neutral. **Verdict:** acceptable.

### Touching Question 5

> "Influence decays exponentially with each derivation step."

Resolves Question 5 in favor of decaying propagation. **Verdict:** stop.

> "Influence is preserved on each derived assertion; whether its operational weight transmits transitively or decays is an open modeling question (Question 5)."

Neutral. **Verdict:** acceptable.

## 4. Procedure when committee resolves an open question

When committee resolution is made, the resolution is recorded in this order. Order matters; reordering loses provenance.

1. **Decision log.** An entry is added to [`docs/charter/decision-log.md`](../../../../docs/charter/decision-log.md) following the existing format. The entry includes the question, the resolution, the constitutional review, and the consequences for the Ontology.
2. **Ontology document.** The relevant Ontology document is updated to reflect the resolution. The question is removed from "Open Questions" and the resolution is reflected in the relevant entity, provenance, or lifecycle section.
3. **This skill.** The registry in §1 is updated. The resolved question is moved to a "Resolved Questions" section (added when the first resolution lands), with a back-reference to the decision-log entry.

If any of the three steps is skipped, the project has lost coherence between the Charter, the Ontology, and the operational scaffolding. The Phase 8 SELF-AUDIT checks for this.

## 5. Subordination

This skill does not have decisional authority over the Ontology. It cannot resolve a question; it can only surface a resolution-attempt for human decision. Committee resolution is committee work, recorded through the formal process.

## 6. Source citations used

- [`docs/ontology/ontology.md` §Open Questions for Committee Resolution](../../../../docs/ontology/ontology.md)
- [`docs/ontology/entity-model.md` §Open Modeling Questions](../../../../docs/ontology/entity-model.md)
- [`docs/ontology/provenance-model.md` §Open Modeling Questions](../../../../docs/ontology/provenance-model.md)
- [`docs/ontology/lifecycle-semantics.md` §Open Modeling Questions](../../../../docs/ontology/lifecycle-semantics.md)
- [`docs/charter/decision-log.md` §0004 — Charter is authoritative](../../../../docs/charter/decision-log.md)
- [`docs/charter/amendments.md`](../../../../docs/charter/amendments.md)
- [`.claude/CLAUDE.md` §4 Charter status at a glance](../../../CLAUDE.md)
