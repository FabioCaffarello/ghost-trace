---
name: epistemic-separator
description: Force structural separation between observation (Category I), operational construct (Category II), and hypothesis (Category III) in any prose that touches the substrate or its projections. Use this skill ALWAYS when writing or editing under docs/charter/, docs/ontology/, docs/architecture/, or any RFC; ALWAYS when reviewing prose containing the words "event", "record", "entity", "data", "signal", "detection", "cluster", or "session"; and BEFORE finalizing any paragraph that mentions more than one knowledge category.
---

# epistemic-separator

The Charter establishes three structurally distinct categories of knowledge ([Charter §2.2](../../../../docs/charter/constitutional-charter.md#22-epistemic-separation)). Prose that conflates them is the failure mode the system exists to prevent. This skill keeps the conflation out of the text before it becomes structure.

## 1. The three categories

### Category I — observation

Immutable record of fact. Answers *what happened*, never *what does it mean*. Once committed to the primary event log, an observation cannot be modified, deleted, or annotated with inferential content ([Charter §2.1](../../../../docs/charter/constitutional-charter.md#21-observational-integrity); [`entity-model.md` Category I](../../../../docs/ontology/entity-model.md)).

Valid operations: commit (append-only). Not valid: mutation, annotation with inferential content, soft deletion, retroactive correction.

### Category II — operational construct

Entity derived deterministically from observations under a versioned operational definition. Identity composes the definition reference, its parameters, and its time scope ([`entity-model.md` Category II](../../../../docs/ontology/entity-model.md)).

Valid operations: parametric re-derivation under a new definition (produces a new construct; does not mutate the existing one). Not valid: in-place mutation of an existing construct.

### Category III — hypothesis

Probabilistic inference whose boundaries, membership, and continued existence are matters of degree, not of fact ([Charter §1](../../../../docs/charter/constitutional-charter.md#1-thesis); [`entity-model.md` Category III](../../../../docs/ontology/entity-model.md)).

Valid operations: formation, merge, split, promotion, demotion, dissolution — each recorded as an immutable lifecycle event ([`lifecycle-semantics.md` §Category III](../../../../docs/ontology/lifecycle-semantics.md); [Charter §2.5](../../../../docs/charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness)). Not valid: direct mutation of hypothesis state.

## 2. Adjacent concepts

These terms have specific structural meaning. Substituting them is also drift.

- **Enrichment.** Operational knowledge paired with observations as a separate stream of immutable events; not a mutation of the observation ([`event-flow.md` Phase 2](../../../../docs/architecture/event-flow.md)).
- **Assertion.** Any non-observation record the system produces — Category II or Category III ([Charter §1](../../../../docs/charter/constitutional-charter.md#1-thesis)).
- **Projection.** A materialized view derived from the substrate. Rebuildable; not bound by Invariant 2.1 ([`projection-model.md`](../../../../docs/architecture/projection-model.md)). "The substrate's projection" is correct; "the projection's substrate" inverts the relation.

## 3. Checklist for multi-category paragraphs

Apply when a paragraph mentions more than one category, or contains any of the triggering words listed in the description.

1. **Naming.** Is each category named explicitly — `observation`, `operational construct`, `hypothesis`, `assertion`, `projection` — and not by a generic word (`data`, `record`, `entity`, `event`, `thing`)? If a generic word is used, replace it.
2. **Operation validity.** For each operation mentioned, is it on the list of operations valid for that category? An invalid operation paired with a category is a constitutional drift signal, not a typo.
3. **Typed crossing.** Where a sentence crosses categories, is the crossing marked as a typed transformation (`promote`, `derive`, `commit`, `supersede`) rather than an implicit reclassification (`update`, `tag`, `flag`)?

A paragraph that fails any of these is rewritten, not patched.

## 4. Forbidden constructions and their rewrites

### "We tag the event with its detection score."

**Violation.** Annotation of an observation with inferential content. Collapses observation and hypothesis into a single mutable record. Forbidden by [Charter §2.1 — Forbidden Anti-Patterns](../../../../docs/charter/constitutional-charter.md#21-observational-integrity).

**Rewrite.** "Detection produces an assertion that references the observation. The observation itself is not modified."

### "We update the cluster when new evidence arrives."

**Violation.** Direct mutation of hypothesis state. Forbidden by [Charter §2.5](../../../../docs/charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) and inconsistent with the structural commitment in the Thesis.

**Rewrite.** "When new observations enter the substrate, the assertion engine emits a hypothesis-evolution event that supersedes the prior state. The current state of the hypothesis is a projection over its history of events."

### "Records flow through enrichment."

**Violation.** Collapses Category I, Category II, and Category III into the generic word `records`. Also misframes enrichment (a separate stream of immutable events) as a pipeline stage that transforms its input.

**Rewrite.** "Observations are paired by reference with enrichment records as they pass through Phase 2 of the event flow ([`event-flow.md`](../../../../docs/architecture/event-flow.md)). Neither the observation nor the enrichment is mutated; the pairing is a new record."

### "The system maintains state about each actor."

**Violation.** `state` is undefined here. Without specifying which entity's lifecycle, the claim is unfalsifiable.

**Rewrite.** "The system maintains, as projection, the current values of each Category II construct keyed by actor identity. The substrate retains the full event history from which the projection is derived."

### "The detector reads from the event log and writes the cluster back."

**Violation.** "Writes back" implies mutation of substrate (Invariant 2.1) and implicit reclassification — a hypothesis cannot be "written to the event log" in the same sense as an observation; it must be committed as a hypothesis-lifecycle event with its own provenance ([Charter §2.5](../../../../docs/charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness)).

**Rewrite.** "The assertion engine reads observations from the primary event log, derives a hypothesis with observational provenance, and commits a hypothesis-formation event back to the log."

### "The assertion's source is contextually clear from the surrounding text."

**Violation.** Orphan-typed-reference failure mode. Forbidden by [Charter §2.3](../../../../docs/charter/constitutional-charter.md#23-provenance-integrity) (Provenance Integrity, frozen v0.4): every Assertion declares its observational provenance *structurally* via a populated typed reference field (`subject_ref_observation`, `subject_ref_construct`, or `subject_ref_hypothesis`) per the Q3 schemas commitment ([`decision-log §0016`](../../../../docs/charter/decision-log.md)). "Contextually clear", "implicit", "inferable at query time", and "obvious from the surrounding text" all describe an Assertion whose typed reference is unpopulated or unresolvable — schemas-level oneOf-exclusivity rejects this at validation time.

**Rewrite.** "The Assertion declares its subject via a populated `subject_ref_observation` (Category I primary), `subject_ref_construct` (Category II construct), or `subject_ref_hypothesis` (Category III hypothesis) field; the reference chain is structurally reconstructible back to Category I substrate per [Charter §2.3](../../../../docs/charter/constitutional-charter.md#23-provenance-integrity)."

### "The Assertion's inferential origin is evident from the surrounding analysis."

**Violation.** Loss-of-evidence-belief-distinction failure mode (prose-side manifestation). Forbidden by [Charter §2.4](../../../../docs/charter/constitutional-charter.md#24-inferential-influence-disclosure) (Inferential Influence Disclosure, frozen v0.5): every Assertion that was formed under inferential influence declares it *structurally* via a populated typed `subject_ref_construct` or `subject_ref_hypothesis` field, with the corresponding `influenced_by` chain to substrate per [`decision-log §0021`](../../../../docs/charter/decision-log.md) (OMQ #3-α substrate-time generation). "Evident from the surrounding analysis", "implicit in the inference context", "follows from prior reasoning", and "consistent with what was already established" all describe an Assertion whose inferential origin is claimed in prose but whose `influenced_by` chain to substrate is absent — committee-mode review of downstream prose for observational-vs-inferential conflation fails this construction per AP3 of [Charter §2.4](../../../../docs/charter/constitutional-charter.md#24-inferential-influence-disclosure).

**Rewrite.** "The Assertion declares its inferential origin via a populated `subject_ref_construct` (referencing the Category II construct) or `subject_ref_hypothesis` (referencing the Category III hypothesis whose promotion influenced this formation), with the corresponding `influenced_by` chain reconstructible from substrate per [Charter §2.4](../../../../docs/charter/constitutional-charter.md#24-inferential-influence-disclosure)."

## 5. Self-application test

At the end of writing a paragraph that touches any category:

1. Re-read every noun and noun phrase. For each, identify which category it refers to.
2. If any noun is ambiguous — could plausibly refer to more than one category — rewrite it with the category named.
3. If a verb in the paragraph is not on the list of operations valid for the category of its subject, then the verb is wrong, the subject is wrong, or the categorization is wrong. Identify which and fix.
4. If a sentence describes a cross-category operation, verify it is named as a typed transformation and not as an implicit conversion.

Failure to apply this test before commit is itself a form of drift — the discipline is in the loop, not in the document.

## 6. Source citations used

- [`docs/charter/constitutional-charter.md` §1 Thesis](../../../../docs/charter/constitutional-charter.md#1-thesis)
- [`docs/charter/constitutional-charter.md` §2.1 Observational Integrity (Forbidden Anti-Patterns)](../../../../docs/charter/constitutional-charter.md#21-observational-integrity)
- [`docs/charter/constitutional-charter.md` §2.2 Epistemic Separation](../../../../docs/charter/constitutional-charter.md#22-epistemic-separation)
- [`docs/charter/constitutional-charter.md` §2.3 Provenance Integrity](../../../../docs/charter/constitutional-charter.md#23-provenance-integrity)
- [`docs/charter/constitutional-charter.md` §2.4 Inferential Influence Disclosure](../../../../docs/charter/constitutional-charter.md#24-inferential-influence-disclosure)
- [`docs/charter/constitutional-charter.md` §2.5 Hypothesis Lifecycle Explicitness](../../../../docs/charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness)
- [`docs/ontology/entity-model.md` Categories I, II, III](../../../../docs/ontology/entity-model.md)
- [`docs/ontology/lifecycle-semantics.md` §Hypothesis (Category III)](../../../../docs/ontology/lifecycle-semantics.md)
- [`docs/architecture/event-flow.md` Phase 2 — Enrichment](../../../../docs/architecture/event-flow.md)
- [`docs/architecture/projection-model.md`](../../../../docs/architecture/projection-model.md)
