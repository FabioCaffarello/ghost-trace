---
name: subordination-checker
description: Verify that lower-ranked documents do not contradict higher-ranked ones, per the precedence rule. Use this skill ALWAYS when editing under docs/ontology/, docs/architecture/, docs/rfcs/, schemas/, or services/. Critical when a claim in a lower-ranked document looks like it "clarifies" something in a higher-ranked one — clarification of a higher document via a lower one is the silent-amendment failure mode this skill exists to prevent.
---

# subordination-checker

The repository has six explicit document ranks ([`README.md` §Document Hierarchy](../../../../README.md); [`.claude/CLAUDE.md` §2](../../../CLAUDE.md)). Conflicts resolve upward: the lower document is revised, never the higher one. This skill enforces the direction.

## 1. The rank order

1. **Constitutional Charter** ([`docs/charter/constitutional-charter.md`](../../../../docs/charter/constitutional-charter.md)) — authoritative; changes require amendment.
2. **Ontology** ([`docs/ontology/`](../../../../docs/ontology/)) — formalizes Charter concepts.
3. **Architecture** ([`docs/architecture/`](../../../../docs/architecture/)) — translates the Ontology into operational design.
4. **RFCs** ([`docs/rfcs/`](../../../../docs/rfcs/)) — proposals subject to constitutional review.
5. **Schemas** ([`schemas/`](../../../../schemas/)) — materialize the Ontology.
6. **Services** ([`services/`](../../../../services/)) — implementations.

A parallel rule applies to operational documents: when [`CONTRIBUTING.md`](../../../../CONTRIBUTING.md) (process) and `WORKFLOW.md` (tooling) conflict, `CONTRIBUTING.md` wins ([`.claude/CLAUDE.md` §2](../../../CLAUDE.md)).

The Charter itself is the responsibility of `charter-guardian`. This skill handles ranks 2–6 and the operational parallel.

## 2. Directional check procedure

Apply to any edit to a non-Charter document.

### Step 1 — Enumerate claims

List every claim the edit introduces or modifies. A "claim" is any sentence that asserts a property of the system, an entity, a relation, a behavior, or a constraint. Sentences that merely repeat the higher document do not count as new claims; sentences that extend, qualify, or rephrase do.

### Step 2 — Locate the governing rank

For each claim, identify the highest-ranked document that addresses the same subject. Search in order: Charter → Ontology → Architecture. Do not search lower than the document being edited.

A claim that has no governing document at any higher rank is unprecedented; surface this as a separate concern. The claim may belong in the higher document, or it may be premature.

### Step 3 — Classify

For each claim, classify against the governing document:

- **Contradiction.** The claim is incompatible with the governing document — incompatible in meaning, in structural commitment, or in the set of operations permitted. Block the edit.
- **Extension.** The claim adds detail or specificity to a topic the governing document leaves open, consistent with everything the governing document says. Permitted.
- **Repetition.** The claim restates the governing document. Acceptable but flag for review — repeated wording drifts on the next revision of the governing document.

### Step 4 — Resolve

For contradictions: the lower document is revised. The higher document is not edited to "fit." If the lower document's claim names a real property the higher document fails to express, the resolution path is upward — an RFC (§4 below).

## 3. Contradiction vs extension — examples

### Contradiction

- `schemas/events/` declares an event type with mutable fields.
  Governing: [Invariant 2.1 Observational Integrity](../../../../docs/charter/constitutional-charter.md#21-observational-integrity) requires the primary event log to be append-only. Mutable fields on an event type contradict this.
  **Verdict:** block. The schema is revised.
- `docs/architecture/storage-model.md` introduces a consolidation step that overwrites observations to save space.
  Governing: [Invariant 2.1 — Forbidden Anti-Patterns](../../../../docs/charter/constitutional-charter.md#21-observational-integrity) names "destructive deduplication or compaction" as an anti-pattern.
  **Verdict:** block.
- `services/assertion-engine/` declares a unified output type with a `kind` discriminator.
  Governing: [Invariant 2.2 — Forbidden Anti-Patterns](../../../../docs/charter/constitutional-charter.md#22-epistemic-separation) names "unified assertion models" with a `kind` field as an anti-pattern.
  **Verdict:** block.

### Extension

- `schemas/events/` declares concrete field types for an event variety whose existence is implied by [`entity-model.md` Category I](../../../../docs/ontology/entity-model.md).
  Governing: the Ontology lists examples of observations. The schema makes one of them concrete.
  **Verdict:** permitted, subject to vocabulary and falsifiability review.
- `docs/architecture/event-flow.md` Phase 3 specifies a particular ordering of derivation steps.
  Governing: the Charter is silent on derivation ordering; the Ontology specifies properties, not order.
  **Verdict:** permitted.

### Repetition

- `services/ingestion/` declares idempotent commitment.
  Governing: the property is implied by [Invariant 2.1](../../../../docs/charter/constitutional-charter.md#21-observational-integrity) (deduplication by content-address). The service declaration restates it.
  **Verdict:** acceptable; flag for review on next governing-document revision.

## 4. Conflicts that cannot be resolved by revising the lower document

If the lower document's claim names a property the system genuinely needs but the higher document does not yet express, the resolution path is upward. The lower document is not edited to assert the property unilaterally.

- If the property is constitutional — identity-defining, structurally enforceable, independent of operator interpretation per [Charter §2](../../../../docs/charter/constitutional-charter.md#2-constitutional-invariants) — open an RFC of type `charter-amendment` per [`amendments.md`](../../../../docs/charter/amendments.md).
- If the property is ontological — an entity, a relation, a lifecycle rule — open an RFC of type `ontology-revision`.
- If the property is architectural, propose the change as an `architecture` RFC.

The upward path is the only legitimate path.

## 5. What this skill does not do

This skill does not authoritatively interpret the higher document. It identifies candidate conflicts and surfaces them. Resolution requires human review.

The Charter is checked by `charter-guardian`. This skill handles ranks 2 and below; the rank-1 Charter has its own discipline.

## 6. Source citations used

- [`README.md` §Document Hierarchy](../../../../README.md)
- [`docs/charter/constitutional-charter.md` §2 Constitutional Invariants; §2.1 Observational Integrity (Forbidden Anti-Patterns); §2.2 Epistemic Separation (Forbidden Anti-Patterns)](../../../../docs/charter/constitutional-charter.md)
- [`docs/charter/amendments.md`](../../../../docs/charter/amendments.md)
- [`docs/ontology/entity-model.md`](../../../../docs/ontology/entity-model.md)
- [`docs/architecture/`](../../../../docs/architecture/)
- [`.claude/CLAUDE.md` §2 Document hierarchy](../../../CLAUDE.md)
- [`.claude/skills/constitutional/charter-guardian/SKILL.md`](../charter-guardian/SKILL.md)
