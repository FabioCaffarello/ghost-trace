---
name: falsifiability-check
description: Apply the Charter's falsifiability discipline to any claim that purports to be constitutional, ontological, or architectural in character. Use this skill ALWAYS when editing docs/charter/, ALWAYS when redacting a pending invariant, ALWAYS when writing or reviewing an RFC's Constitutional Review section, and ALWAYS when proposing a new claim for the Ontology. Subsumes operationalization checking.
---

# falsifiability-check

The project's falsifiability discipline is recorded in [Charter §4 — Constitutional Design Rule](../../../../docs/charter/constitutional-charter.md#4-constitutional-design-rule):

> A constitutional claim is admissible if and only if it is structurally falsifiable. A property that cannot, in principle, be violated, observed, or audited is not a constitutional property; it is an aspiration, an aesthetic preference, or a research direction, and belongs elsewhere.

The same discipline is restated procedurally in [`CONTRIBUTING.md` §What This Project Is Not](../../../../CONTRIBUTING.md): "If you find yourself writing prose that sounds important but cannot be falsified, rewrite it or do not include it."

This skill operationalizes the discipline. It applies to constitutional claims (Charter), ontological claims (Ontology), architectural commitments (architecture documents), and the `Constitutional Review` section of every RFC.

## 1. The four-question test

Apply all four to every candidate claim. A claim that fails any test does not belong in the Charter as written — it must be rewritten, demoted, or removed.

### 1.1 Violation test

Describe a system state in which this claim is false. Be concrete: name the observable artifact in the substrate or its projections that would make the claim true vs. false.

If you cannot construct such a state, the claim is not falsifiable. The most common cause: the claim is about a global property of the system that has no localized witness.

### 1.2 Observation test

Describe how a third party with access to the substrate — without the cooperation of the system that produced it — could detect the violation.

If detection requires the violator's cooperation (e.g., "the system must log all writes") with no independent verification path, the claim is not structurally falsifiable; it is a policy expecting good faith.

If detection requires subjective judgment ("the projection is accurate enough"), the claim is not structurally falsifiable; it is an aesthetic preference.

This test echoes the qualification criterion in [Charter §2](../../../../docs/charter/constitutional-charter.md#2-constitutional-invariants): "independent of operator interpretation — violation is detectable without subjective judgment."

### 1.3 Operationalization test

Reduce each term in the claim to a concrete artifact in the substrate or its projections.

A claim is operationalizable if each of its terms can be replaced by a specific record type, schema field, projection name, or event variety. A claim whose terms are `appropriate`, `sufficient`, `robust`, `integrity`, `quality`, `trust`, or `coherence` — without further reduction in the same document — fails this test.

This test echoes the qualification criterion in [Charter §2](../../../../docs/charter/constitutional-charter.md#2-constitutional-invariants): "structurally enforceable — verifiable in schema, types, or permitted operations, not merely in code review."

### 1.4 Non-circularity test

Does the claim define a term using only terms defined elsewhere — in the Charter, the Ontology, the canonical vocabulary in [`.claude/CLAUDE.md` §3](../../../CLAUDE.md), or in this skill?

A claim that defines X using X (`an observation is what is observed`), or defines X using Y while Y is defined using X, fails this test.

The Charter's existing definitions form the dependency root. New terms reduce to existing ones; existing ones do not reduce back to new ones.

## 2. Failure modes with examples

### Non-violable

> "The system respects user trust."

No procedure can show this is false. `trust` is not localized in any artifact. **Verdict:** rejected. Belongs in marketing copy, not in the Charter.

### Non-observable

> "The system maintains integrity."

`integrity` is unobservable without further specification of which property is integral and how its violation is detected. **Verdict:** rejected. Rewrite as a specific invariant whose violation is detectable. For example: "every record in the primary event log carries a content-addressable identifier whose recomputation on read must match the stored identifier."

### Non-operationalizable

> "Hypotheses are evaluated on their merits."

`evaluated` and `merits` do not reduce to concrete artifacts. **Verdict:** rejected. Rewrite by naming the evaluation procedure and the substrate artifact it inspects.

### Circular

> "An observation is what is observed and recorded."

Defines `observation` using `observed` (a variant of itself). **Verdict:** rejected. Use the Charter's existing definition: a record committed to the primary event log under Category I ([Charter §2.2](../../../../docs/charter/constitutional-charter.md#22-epistemic-separation)).

### Acceptable

> "The primary event log is append-only by enforced guarantee. A successful mutation of a committed record, detected by recomputation of its content-addressable identifier on read, is a violation of Invariant 2.1."

Each term is grounded in the Charter; the violation is concrete and observable by a third party. **Verdict:** passes all four tests.

## 3. Rewrite paths

When a claim fails the test, choose one path. Doing nothing is not a path.

### Path A — Operationalize

If the claim names a real property the system is required to have, restate it in terms that pass the four tests. Reduce each abstract term to a substrate artifact or a projection query. Re-apply the test.

### Path B — Demote

If the claim is operationally useful but not structurally falsifiable, demote it out of the Charter:

- A property the system tries to provide but cannot guarantee → move to an architecture document marked as aspirational.
- A direction the project explores but does not commit to → move to an RFC of type `experiment`.
- An aesthetic preference for documentation style → move to `CONTRIBUTING.md`.

### Path C — Remove

If the claim is neither falsifiable nor operationally useful, it is decoration. Remove it. The Charter is the project's structural identity; decoration weakens it.

## 4. Procedural integration

This skill is not optional during:

- Redaction of a pending Charter invariant (§2.6, §3). Every clause in the draft is tested before committee approval.
- Authoring of an RFC's `Constitutional Review` section. Each interaction with an invariant is tested.
- Introduction of a new claim in the Ontology that has not been explicitly delegated by the Charter as ontological discretion.

The four-test result is recorded in the RFC or amendment proposal as part of procedural review. Proposals that introduce non-falsifiable language at the Charter level are rejected on procedural grounds ([`docs/charter/amendments.md` §Amendment Process](../../../../docs/charter/amendments.md)).

## 5. Source citations used

- [`docs/charter/constitutional-charter.md` §2 Constitutional Invariants (qualification criteria)](../../../../docs/charter/constitutional-charter.md#2-constitutional-invariants)
- [`docs/charter/constitutional-charter.md` §2.2 Epistemic Separation](../../../../docs/charter/constitutional-charter.md#22-epistemic-separation)
- [`docs/charter/constitutional-charter.md` §4 Constitutional Design Rule](../../../../docs/charter/constitutional-charter.md#4-constitutional-design-rule)
- [`docs/charter/amendments.md` §Amendment Process](../../../../docs/charter/amendments.md)
- [`CONTRIBUTING.md` §What This Project Is Not](../../../../CONTRIBUTING.md)
- [`.claude/CLAUDE.md` §3 Canonical vocabulary](../../../CLAUDE.md)
