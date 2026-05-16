# RFC — Ontology Open Question 3: Subject Reference Polymorphism

- **Status:** accepted
- **Authors:** Ghost Trace committee (Q3 pre-Gate; §0014 cascade enactment)
- **Date:** 2026-05-16
- **Type:** ontology-revision
- **Affects:** [`docs/ontology/entity-model.md`](../../ontology/entity-model.md) (Open Modeling Question 2 post-Q1-renumbering — `subject_ref` structural definition); [Charter §2.3](../../charter/constitutional-charter.md#23-provenance-integrity) (pending — binding text depends on this resolution; blocking per §0014 cascade); [Charter §2.4](../../charter/constitutional-charter.md#24-inferential-influence-disclosure) (pending — inferential influence references via `subject_ref`); [Charter §2.5](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) (frozen v0.3 — hypothesis events reference Sessions via `subject_ref`; Q1 Candidate B selection forces typed reference distinction)

> RFCs in Ghost Trace are proposals subject to constitutional review. The [Constitutional Charter](../../charter/constitutional-charter.md) is the authoritative reference. RFCs that conflict with the Charter must either be rejected or accompanied by a corresponding charter amendment (see [`../../charter/amendments.md`](../../charter/amendments.md)).

## Summary

Q3, recorded in [`docs/ontology/entity-model.md` Open Modeling Question 2 post-Q1-renumbering](../../ontology/entity-model.md) and originally framed in [`ontology.md` §Open Questions](../../ontology/ontology.md): assertions carry a `subject_ref` field that may point to entities of any category. Is this a single polymorphic field or distinct fields per category? Post-Q1 resolution ([decision-log §0015](../../charter/decision-log.md), Candidate B — Session has typed duality), the polymorphism question is acute: `subject_ref` to Session must distinguish `DeclaredSession` (Cat I) from `OperationalSession` (Cat II); the polymorphism scope extends beyond Sessions to all cross-category references in the entity model.

This RFC opens structured discussion with two candidate resolutions and one explicitly rejected alternative. The RFC does not pick a candidate.

## Motivation

**Why now.** §2.3 Step 1.2 requires Q3 resolution per [§0014's lazy pre-Gate refinement](../../charter/decision-log.md) triggered by [§0015](../../charter/decision-log.md) (Q1 resolution to Candidate B). §2.5 hypothesis events already reference Sessions via `subject_ref` (per §2.5 v0.3 Boundary Condition 5 forward-reference). Under Q1-B, the reference shape needs Q3 resolved to distinguish `DeclaredSession` from `OperationalSession` structurally; the same polymorphism question extends across all cross-category subject references.

**The cost of not resolving.** Silent resolution at §2.3 redaction would pick an answer without committee deliberation. Per [`ontology-keeper` discipline](../../../.claude/skills/ontology/ontology-keeper/SKILL.md), open Ontology questions are committee-resolved, not infrastructure-resolved. The Q3 pre-Gate is the expected cascade per §0014 methodology — its opening is the documented trigger condition firing as designed.

## Constitutional Review

The Q1–Q6 impact analysis prescribed by [`rfc-author` §1](../../../.claude/skills/workflow/rfc-author/SKILL.md).

### Q1 — Which Charter invariants does this RFC touch?

- **§2.3 Provenance Integrity** (pending): touched directly. §2.3 binding text encodes `subject_ref` shape; this RFC's resolution determines whether the field is polymorphic or split per category.
- **§2.4 Inferential Influence Disclosure** (pending): touched indirectly. Assertions formed under inferential influence reference subjects via `subject_ref`. Q3 resolution constrains §2.4 binding text similarly.
- **§2.5 Hypothesis Lifecycle Explicitness** (frozen v0.3): consistency check. §2.5 binding text references Sessions via `subject_ref` (the forward-reference per §2.5 Boundary Condition 5). Resolution must preserve §2.5 frozen text's compatibility with the chosen polymorphism form.
- **§2.2 Epistemic Separation** (frozen): consistency check. Both candidates preserve the structural Category I/II/III separation; the resolution concerns how cross-category references are encoded structurally.
- No FROZEN invariant amended.

### Q2 — Does this RFC implicitly redefine any term in the glossary?

The term `subject_ref` is not currently a glossary entry; it is referenced in `entity-model.md` Open Modeling Question 2 (post-Q1-renumbering). Both candidates may require glossary entries:
- Under Candidate A (single polymorphic field), one glossary entry for `subject_ref` defining the polymorphic type.
- Under Candidate B (split per-category fields), multiple glossary entries (e.g., `subject_ref_observation`, `subject_ref_construct`, `subject_ref_hypothesis`).

Either candidate carries glossary-extension obligations; the asymmetry is in count.

### Q3 — Does this RFC implicitly resolve any of the five open Ontology questions?

This RFC opens structured discussion of Open Modeling Question 2 (post-Q1-renumbering). It does not resolve Identity tiers (the remaining Open Modeling Question 1). The RFC interacts with Identity tiers tangentially: under Candidate B (split per-category fields), the Cat I reference field shape may need to align with how identity-tier references are encoded; under Candidate A (single polymorphic field), the alignment is one layer of indirection. Identity tiers remains forward-referenceable per §0011 precedent regardless of Q3 outcome.

### Q4 — Does this RFC require Charter amendment?

No. The resolution constrains §2.3 (pending) and §2.4 (pending) binding text. §2.5 frozen v0.3 references `subject_ref` via a forward-reference that accommodates either candidate (the forward-reference mechanism per §2 L41 precedent does not pre-decide the polymorphism shape).

### Q5 — Does this RFC introduce a new invariant?

No. The RFC resolves a modeling question that §2.3 and §2.4 will codify in their binding text. Per the constitutional minimalism rule, introducing a new invariant here would be unjustified.

### Q6 — Does this RFC propose ceremony without behavioral consequence?

No. The two candidates produce structurally different schemas, validation surfaces, query patterns, and provenance graph shapes. Q1 Candidate B's structural commitment to typed-distinct Session forms forces this RFC; silent resolution at §2.3 Step 1.2 would either contradict Q1-B or produce un-redactable binding text.

## Proposal

Two candidate resolutions. The RFC does not pick.

### Candidate A — Single polymorphic `subject_ref`

**Structural claim.** Every assertion carries one `subject_ref` field of polymorphic type. The referent type is discriminated either by an accompanying `subject_type` discriminator field (enum-valued — `observation`, `operational_construct`, `hypothesis`, plus subtype refinements like `declared_session`, `operational_session`, `behavioral_cluster`, etc.) or by schemas-level union/oneOf (depending on substrate technology selection per §0003).

**Schemas implication.** One field, polymorphic surface. Validation surface: one field's polymorphic type definition. Query patterns: type-discriminator branching at query layer to resolve the referent's category and subtype.

**Q1-B interaction.** `subject_ref` to a Session resolves to `subject_type = declared_session` (Cat I) OR `subject_type = operational_session` (Cat II). The distinction lives in the discriminator field; the `subject_ref` field itself is monomorphic in form.

**Pros.**

- Single field shape; smaller validation surface; easier to evolve when new subtypes are added.
- Polymorphism is centralized in one field's type definition; queries that do not care about subject type can ignore the discriminator.
- §2.5 frozen v0.3's forward-reference to `subject_ref` accommodates this candidate without amendment.

**Cons.**

- Subtype-specific constraints (e.g., "this assertion can only reference Category I subjects") cannot be enforced at the field-level type definition; they migrate to validation rules.
- Type-discriminator mismatch (where the discriminator and the actual referent disagree) becomes a category of runtime error rather than a structural impossibility.
- Reads need to branch on the discriminator; the discriminator-field becomes a critical path.

### Candidate B — Distinct `subject_ref` fields per category

**Structural claim.** Every assertion carries category-specific subject_ref fields: `subject_ref_observation` (references Cat I primaries including `DeclaredSession`), `subject_ref_construct` (references Cat II constructs including `OperationalSession`), `subject_ref_hypothesis` (references Cat III subtypes). Exactly one is populated per assertion (or, depending on assertion semantics, more than one with one-per-category cardinality).

**Schemas implication.** Three category-specific fields. Validation surface: per-field type discipline; exactly-one-populated constraint at the assertion level. Type-system can enforce "this assertion references a Cat I subject" by populating only `subject_ref_observation`.

**Q1-B interaction.** `subject_ref` to a Session resolves to one of two category-specific fields: `subject_ref_observation` for `DeclaredSession`; `subject_ref_construct` for `OperationalSession`. The distinction is type-level structural.

**Pros.**

- Subtype-specific constraints on which category an assertion can reference are structurally enforceable (population pattern enforced by type system).
- No type-discriminator mismatch possible — the field choice IS the type commitment.
- Q1-B's structural-exposure pattern is preserved: declared vs operational session reference is visible by field, not by discriminator-lookup.

**Cons.**

- Three fields per assertion (vs one); validation surface is larger.
- Adding a new category would require adding a new field and migrating existing schemas.
- Queries that scan all references must read three fields.

## Alternatives Considered

### Candidate C — Polymorphic with runtime classification (REJECTED)

**Structural claim.** One `subject_ref` field with the category of the referent determined at runtime (e.g., by looking up the referent's record type at read time).

**Reason for rejection.** Collapses Q3 to runtime — the category boundary becomes a runtime resolution, not a structural property. Per [`ontology-keeper` §3](../../../.claude/skills/ontology/ontology-keeper/SKILL.md), this is the failure mode the skill prevents. Also fails §4 criterion 1 (structural enforceability) — the category boundary on cross-category references becomes implicit rather than declared.

## Open Questions

This RFC explicitly defers the following:

- **Schemas technology selection.** Whether discrimination under Candidate A is by enum discriminator, by type-system union, or by schemas-level oneOf is implementation/schemas concern, deferred to the substrate-technology RFC ([`decision-log §0003`](../../charter/decision-log.md)).
- **Identity-tier interaction.** Whether Cat I `DeclaredSession` references go through identity-tier indirection (referencing an identity-tier owner, with Session resolved through the identity) or carry direct references is partially an Identity-tiers question. Forward-referenceable per §0011 precedent.
- **Cardinality.** Whether assertions carry exactly-one `subject_ref` or potentially multiple (e.g., an assertion that derives from observations of multiple subjects) is independent of polymorphism shape; deferred.

## Anti-Patterns to Avoid

- **Resolving Q3 by code.** Pushing the decision into schemas technology pick or projection rules without RFC.
- **Treating Q3 as a schemas-level concern.** It is an ontology-level concern about how cross-category references are typed at the substrate.
- **Silent revision in a future `entity-model.md` redaction.** A future redaction that picks one candidate without acknowledging Q3 being resolved is the failure mode `ontology-keeper` prevents.

## Migration and Backward Compatibility

No historical content; forward-looking. Two asymmetries noted:

- Lock-in: Candidate B → Candidate A migration would collapse three fields into one polymorphic field with associated type discriminator; substantively a refactor. Candidate A → Candidate B migration would expand one field into three category-specific fields; substantively a structural commitment broadening.
- §2.5 v0.3 compatibility: §2.5 binding text references `subject_ref` via forward-reference per §2 L41 precedent. Either candidate is compatible without §2.5 amendment.

## References

- [`docs/ontology/entity-model.md`](../../ontology/entity-model.md) — Open Modeling Question 2 (post-Q1-renumbering); Category I + II revisions post-Q1.
- [`docs/charter/decision-log.md` §0015](../../charter/decision-log.md) — Q1 resolution that triggered this RFC via §0014 cascade.
- [`docs/charter/decision-log.md` §0014](../../charter/decision-log.md) — lazy pre-Gate refinement; this RFC enacts §0014's anticipated trigger.
- [`docs/charter/decision-log.md` §0008](../../charter/decision-log.md) — redaction order §2.5 → §2.3 → §2.4 → §2.6.
- [`docs/charter/decision-log.md` §0011](../../charter/decision-log.md) — Q4 resolution + forward-reference precedent (applies to Identity-tier deferral).
- [`.claude/skills/ontology/ontology-keeper/SKILL.md`](../../../.claude/skills/ontology/ontology-keeper/SKILL.md) — registry of open Ontology questions.
- [`.claude/skills/workflow/rfc-author/SKILL.md`](../../../.claude/skills/workflow/rfc-author/SKILL.md) — Q1–Q6 impact analysis procedure.

## Decision Record

Resolved by [`docs/charter/decision-log.md` §0016 — Q3 resolution](../../charter/decision-log.md). The committee adopted **Candidate B — distinct per-Category `subject_ref` fields on the Assertion entity**, with three committee extensions:

1. **Granularity: per-Category coarse (3 fields).** `subject_ref_observation` (Cat I), `subject_ref_construct` (Cat II), `subject_ref_hypothesis` (Cat III). Rationale: [§2.2](../../charter/constitutional-charter.md#22-epistemic-separation) separation is at Category level, not subtype level; sub-Category granularity is not warranted at the Assertion entity layer. Encoded structurally in `entity-model.md` Assertion entity section.
2. **Exclusivity enforcement: schemas-level oneOf/union with mandatory population.** Exactly one `subject_ref_X` field per Assertion; schemas validation rejects zero or more-than-one populated. Structural commitment encoded in `entity-model.md`.
3. **No per-field glossary entries.** The three fields are structural mechanism, not canonical entities. Contrast §0015 (Q1 resolution) which added `declared session` and `operational session` as canonical entities.

The §0014 lazy pre-Gate cascade fully discharges with this resolution: Q1 (§0015) triggered Q3; Q3 (§0016) resolves; no further cascades anticipated before §2.3 redaction. §2.3 redaction begins Step 1.1 onward.

Discussion-phase evidence is preserved in [`docs/rfcs/discussion/q3-evidence.md`](../discussion/q3-evidence.md).
