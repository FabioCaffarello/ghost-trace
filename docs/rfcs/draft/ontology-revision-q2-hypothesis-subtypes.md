# RFC — Ontology Open Question 2: Hypothesis Subtypes

- **Status:** accepted
- **Authors:** Ghost Trace committee
- **Date:** 2026-05-15
- **Type:** ontology-revision
- **Affects:** [`docs/ontology/entity-model.md`](../../ontology/entity-model.md) (Category III section); [`docs/ontology/lifecycle-semantics.md`](../../ontology/lifecycle-semantics.md) (Category III lifecycle section); [Charter §2.5](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) (downstream binding text depends on this resolution)

> RFCs in Ghost Trace are proposals subject to constitutional review. The [Constitutional Charter](../../charter/constitutional-charter.md) is the authoritative reference. RFCs that conflict with the Charter must either be rejected or accompanied by a corresponding charter amendment (see [`../../charter/amendments.md`](../../charter/amendments.md)).

## Summary

Ontology Open Question 2, recorded in [`docs/ontology/ontology.md` §Open Questions for Committee Resolution](../../ontology/ontology.md): whether `BehavioralCluster`, `CoordinationRing`, `CampaignHypothesis`, and `AutomationGroup` are distinct entity types within the hypothesis category, or are labels on a single `Hypothesis` type. The question governs the type surface of Category III. This RFC opens structured discussion of the question with two candidate resolutions and one explicitly rejected alternative. The RFC does not pick a candidate. §2.5 binding text references operations on hypotheses; if the type surface of the hypothesis category is undecided, §2.5 redaction either presupposes a resolution silently (a constitutional move dressed as editorial work per [`ontology-keeper` §3](../../../.claude/skills/ontology/ontology-keeper/SKILL.md)) or produces vague binding text deferring to later resolution (forbidden under §4 falsifiability discipline).

## Motivation

Two distinct motivations apply.

**Why now.** Per [decision-log §0008](../../charter/decision-log.md), the redaction order for pending invariants is §2.5 → §2.3 → §2.4 → §2.6. §2.5 is next. The §2.5 stub names six lifecycle operations on hypotheses — formation, merge, split, dissolution, promotion, demotion. Each operation must be expressible against a concrete type surface. Under Candidate A, each operation may have subtype-specific parameters; under Candidate B, each operation applies uniformly across all hypothesis instances with subtype carried as a field value. The two candidates produce structurally different §2.5 binding text. Redaction without prior resolution is therefore not neutral on this question.

**The cost of not resolving.** Silent resolution at §2.5 redaction or at the next `entity-model.md` redaction would pick an answer without committee deliberation. That outcome violates [`ontology-keeper`'s](../../../.claude/skills/ontology/ontology-keeper/SKILL.md) discipline: open Ontology questions are committee-resolved, not infrastructure-resolved. The procedural failure mode is well-documented in `ontology-keeper` §3, which lists implicit-resolution examples for each of the five open questions. Both candidates would also leave the substrate's typed surface for Category III records under-specified, which delays implementation gate readiness as evaluated by the future `implementation-readiness-evaluator`.

## Constitutional Review

The Q1–Q6 impact analysis prescribed by [`rfc-author` §1](../../../.claude/skills/workflow/rfc-author/SKILL.md).

### Q1 — Which Charter invariants does this RFC touch?

- **§2.5 Hypothesis Lifecycle Explicitness** (pending): touched directly. §2.5 binding text references operations on hypotheses; the type surface of those operations depends on this resolution.
- **§2.4 Inferential Influence Disclosure** (pending): touched indirectly. Assertions formed under the influence of a hypothesis carry a `subject_ref` to that hypothesis. Whether `subject_ref` is polymorphic across distinct subtypes (Candidate A) or against a single type (Candidate B) is a downstream consequence of this resolution. The entity-model.md Open Modeling Question 4 (`subject_ref` polymorphism) partially depends on this resolution.
- **§2.2 Epistemic Separation** (frozen): consistency check, not amendment. Both candidates preserve the structural distinction between Category III and Categories I and II. The resolution does not propose any cross-category type unification.
- No FROZEN invariant is amended.

### Q2 — Does this RFC implicitly redefine any term in the glossary?

The term `Hypothesis` is in [`docs/glossary.md`](../../glossary.md) and [`.claude/CLAUDE.md` §3 canonical vocabulary](../../../.claude/CLAUDE.md) as a Category III record. Both candidates preserve this definition.

The names `BehavioralCluster`, `CoordinationRing`, `CampaignHypothesis`, and `AutomationGroup` appear in [`entity-model.md` Category III examples](../../ontology/entity-model.md) but are not glossary entries. Under Candidate A (distinct types), each name becomes a constitutional term and would require a glossary entry. Under Candidate B (labels), the names remain illustrative and do not require glossary entries. The asymmetry is recorded here: Candidate A carries a glossary-extension obligation, Candidate B does not.

### Q3 — Does this RFC implicitly resolve any of the five open Ontology questions?

This RFC opens structured discussion of Open Question 2. By construction it touches Q2; it does not resolve Q2. Per `ontology-keeper` §3 the verdict is that this RFC is the procedure for raising Q2 to committee, not an instance of implicit resolution.

The RFC interacts with `entity-model.md` Open Modeling Question 4 (`subject_ref` polymorphism), which is a type-level question with ontological consequences. The two questions are entangled: Q4-of-entity-model collapses under Candidate B and remains open under Candidate A. This interaction is recorded under Open Questions below.

The RFC does not touch ontology.md Q1, Q3, Q5.

### Q4 — Does this RFC require Charter amendment?

No. The resolution constrains §2.5's content. §2.5 is pending — not amendable yet, only redactable for the first time. If this RFC is resolved before §2.5 redaction, no charter-amendment chain is needed; the §2.5 redaction itself remains within its normal committee-mode procedure per [`invariant-redactor`](../../../.claude/skills/constitutional/invariant-redactor/SKILL.md). If §2.5 were to be redacted before this RFC resolves, the §2.5 redaction would either silently resolve Q2 (forbidden) or produce binding text deferring to "later resolution" (also forbidden under §4 falsifiability discipline). The procedural ordering enforced by this RFC eliminates both failure modes.

### Q5 — Does this RFC introduce a new invariant?

No. The RFC resolves a modeling question internal to the Ontology. The invariant that consumes the resolution (§2.5) is a separate constitutional act with its own redaction procedure. Per the constitutional minimalism rule ([`.claude/CLAUDE.md` §7](../../../.claude/CLAUDE.md)), introducing a new invariant here would be unjustified — the existing §2.5 stub already commits the project to lifecycle explicitness.

### Q6 — Does this RFC propose ceremony without behavioral consequence?

No. The two candidates produce different type structures, different query patterns over Category III records, different migration costs for any future subtype addition, and different forms of §2.5 binding text. Deleting this RFC would either force silent resolution at §2.5 redaction or produce un-redactable binding text. The behavioral consequence is direct.

## Proposal

Two candidate resolutions, each presented with structural claim, type-surface implication, lifecycle implication, and pros and cons. The RFC does not pick. The discussion phase gathers evidence and arguments; redaction produces the resolved candidate.

### Candidate A — Distinct types

**Structural claim.** Four sibling entity types within Category III: `BehavioralCluster`, `CoordinationRing`, `CampaignHypothesis`, `AutomationGroup`. Each has its own type definition, its own identity rules, its own set of permitted fields. A common abstract type `Hypothesis` may exist for shared structure (creation timestamp, identifier scheme, provenance reference fields) but does not appear as a concrete record.

**Type-surface implication.** Concrete type structure for Category III is a discriminated union over four types. Each type carries the descriptive fields shared across Category III plus its subtype-specific payload. The discriminator is the type itself; runtime branching is by type identity, not by field-value branching.

**Lifecycle implication.** Each subtype may declare distinct parameters for lifecycle operations defined by §2.5. For instance, the threshold for promotion of a `BehavioralCluster` may differ structurally from the threshold for promotion of a `CampaignHypothesis`. The §2.5 binding text expresses lifecycle operations as polymorphic across the four types.

**Pros.**

- Type-level constraints on subtype-specific fields are enforceable structurally, not in code review. Aligns with the structural-enforceability criterion of §2.
- Migration to add a fifth subtype is a typed extension, not a redefinition of an existing discriminator set.
- Distinct subtypes carry distinct anti-pattern surfaces (e.g., `CoordinationRing` membership semantics differ from `CampaignHypothesis` membership semantics); typing protects each surface.

**Cons.**

- Glossary obligation: four new canonical terms must be defined and maintained.
- Common operations (formation, supersession of interpretation) must be polymorphic across four types; implementation complexity is non-trivial.
- If the four subtypes turn out to share substantially identical lifecycle semantics, Candidate A imposes ceremony where Candidate B would not.

### Candidate B — Single type with discriminator labels

**Structural claim.** One `Hypothesis` entity type within Category III. The four labels (`BehavioralCluster`, `CoordinationRing`, `CampaignHypothesis`, `AutomationGroup`) are values of a `kind` discriminator field carried by each `Hypothesis` instance. The type itself is uniform across all four labels.

**Type-surface implication.** Concrete type structure is a single type with a discriminator field. The substrate stores one type's records; downstream projections may filter by discriminator value for subtype-specific views.

**Lifecycle implication.** Lifecycle operations apply uniformly to all `Hypothesis` instances regardless of discriminator value. Subtype-specific parameters, if needed, live in a parameters map keyed by discriminator value or are absent entirely. The §2.5 binding text expresses lifecycle operations against the single type.

**Pros.**

- One canonical term (`Hypothesis`) covers the full Category III surface; the four labels remain illustrative and do not require glossary entries.
- Common operations are not polymorphic; implementation surface is smaller.
- Promoting a discriminator value to a type later (if needed) is easier than demoting a type to a label.

**Cons.**

- Subtype-specific constraints cannot be enforced structurally; they migrate to application code, which §2 criterion 1 (structural enforceability) discourages.
- The discriminator field forces any downstream consumer to branch on its values; the burden of correct branching is distributed, not centralized.
- If subtype semantics diverge significantly under operational pressure, Candidate B forces accumulating divergence into the single type's payload, producing the unified-record anti-pattern §2.2 forbids in spirit (even though the four labels remain within Category III).

## Alternatives Considered

### Candidate C — Polymorphic with subtype-specific extensions via a shared `extra` map (REJECTED)

**Structural claim.** One `Hypothesis` entity type with a free-form `extra` map field. Subtype-specific record content is stored in this map under conventionally agreed keys.

**Reason for rejection.** This candidate collapses Q2 by deferring it to runtime convention. The type structure does not distinguish subtypes; application code reads keys from `extra` whose presence and meaning are not constitutional. Per [`ontology-keeper` §3](../../../.claude/skills/ontology/ontology-keeper/SKILL.md), this is precisely the failure mode the skill exists to prevent: an open question resolved by infrastructure (the `extra` map convention) rather than by committee. Candidate C also fails §2 criterion 4 (independence of operator interpretation) — the meaning of subtype-specific keys depends on which operator wrote the record and which operator reads it.

## Open Questions

This RFC explicitly defers the following:

- **Names of the subtypes.** If Candidate A is the resolution, whether `BehavioralCluster`, `CoordinationRing`, `CampaignHypothesis`, and `AutomationGroup` are the right names for the four subtypes is itself an open modeling question, separate from Q2. The names are inherited from the [`entity-model.md` Category III](../../ontology/entity-model.md) examples; the examples are illustrative, not committee-ratified.
- **Whether new subtypes can be added without amendment.** Under Candidate A, adding a fifth subtype would be a typed extension. Whether this requires an `ontology-revision` RFC or only a type-level RFC is undecided here.
- **Interaction with ontology.md Q1 (Session duality).** If Q1 resolves with Session as two entities (`DeclaredSession` Category I plus `OperationalSession` Category II), hypotheses about sessions inherit the duality. Whether the inherited duality affects hypothesis subtype structure is open. Q1 resolution is queued for the pre-§2.3 gate; the dependency is recorded here so neither RFC silently resolves the other.
- **Drift between canonical Q2 form and `lifecycle-semantics.md` Open Modeling Question 1.** The canonical Q2 in `ontology.md` asks whether the four labels are distinct types or are values of a single discriminator on a single type. The `lifecycle-semantics.md` Open Modeling Question 1 asks a related but narrower question about whether the four labels have distinct lifecycle rules or share a common lifecycle with category-specific parameters. The two questions are not identical — the first is about the type surface, the second is about lifecycle parameters. Whether `lifecycle-semantics.md` Q1 reduces to Q2 (one answer covers both) or is independent (the four labels could share the type surface but diverge in lifecycle parameters, or vice versa) is itself a question for the redaction phase.
- **Drift between canonical Q2 vocabulary and `vocabulary-discipline` §4 forbidden synonyms.** The canonical Q2 in `ontology.md` uses two terms whose rewrite instructions are recorded in [`vocabulary-discipline` §4](../../../.claude/skills/ontology/vocabulary-discipline/SKILL.md). This RFC paraphrases the canonical Q2 form using `labels` (whose canonical alternative is recorded against the substrate-annotation rewrite instruction) and `type surface` (whose canonical alternative is `type` or `category` for ontological distinction). The vocabulary drift in the source document predates the relevant watchlist entries and was not caught by the pre-commit hook because the hook scans only added or changed lines. Surfacing it here is itself a finding for the Ontology revision that follows Q2's redaction.

## Anti-Patterns to Avoid

- **Resolving Q2 by code.** Pushing the decision into substrate technology selection or into projection rules — the resolution belongs to the Ontology, not to the substrate technology or the projection layer.
- **Treating Q2 as a UI or visualization question.** Whether dashboards display the four labels as separate tabs or as filters on a single view is a downstream UX choice, not a constitutional one. Q2 is a substrate-shape question.
- **Silent revision in a future `entity-model.md` redaction.** A future redaction of `entity-model.md` Category III that picks one candidate without acknowledging Q2 is being resolved is an instance of the failure mode `ontology-keeper` prevents. The redaction must reference this RFC's resolution path.
- **Combining Candidates A and B by allowing both at the type level.** A "Hypothesis with optional subtype types" hybrid presents two surfaces simultaneously and is operationally equivalent to Candidate C; reject for the same reason.

## Migration and Backward Compatibility

No historical Category III records exist at this point. The RFC is forward-looking. Two asymmetries are noted for committee consideration:

- If Candidate A is the resolution, the type structure for hypothesis-category records is non-trivial to refactor to Candidate B later: existing per-subtype types and their per-subtype constraints would need to be unified.
- If Candidate B is the resolution, promoting a discriminator value to a distinct type later is easier: an existing label becomes the discriminator of a new type. The records already carry the label.

Lock-in asymmetry: Candidate A is more committed than Candidate B at the point of resolution. The committee may weight this against the structural-enforceability advantage Candidate A claims.

The Charter's [Phase 3 / Phase 4 replay contracts](../../architecture/replay-model.md) impose no specific obligation on this RFC: both candidates support reconstructive and retrospective replay of Category III evolution, provided the §2.5 binding text records lifecycle operations as immutable events in the primary event log.

## References

- [`docs/ontology/ontology.md` §Open Questions for Committee Resolution](../../ontology/ontology.md) — canonical statement of Q2.
- [`docs/ontology/entity-model.md` §Category III](../../ontology/entity-model.md) — illustrative examples of the four labels; §Open Modeling Questions 3 and 4 (related questions).
- [`docs/ontology/lifecycle-semantics.md` §Category III](../../ontology/lifecycle-semantics.md) — current scaffold; §Open Modeling Question 1 (related-but-narrower question).
- [`docs/charter/constitutional-charter.md` §2.5](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) — stub whose redaction depends on Q2 resolution.
- [`docs/charter/decision-log.md` §0008](../../charter/decision-log.md) — redaction order §2.5 → §2.3 → §2.4 → §2.6.
- [`docs/charter/decision-log.md` §0009](../../charter/decision-log.md) — §2.5 redaction plan registering this RFC as a pre-§2.5 dependency.
- [`.claude/skills/ontology/ontology-keeper/SKILL.md`](../../../.claude/skills/ontology/ontology-keeper/SKILL.md) — registry of the five open Ontology questions.
- [`.claude/skills/ontology/vocabulary-discipline/SKILL.md`](../../../.claude/skills/ontology/vocabulary-discipline/SKILL.md) — canonical vocabulary and forbidden synonyms.
- [`.claude/skills/workflow/rfc-author/SKILL.md`](../../../.claude/skills/workflow/rfc-author/SKILL.md) — Q1–Q6 impact analysis procedure.

## Decision Record

Resolved by [`docs/charter/decision-log.md` §0010 — Q2 resolution: Hypothesis subtypes as distinct types under abstract Hypothesis](../../charter/decision-log.md). The committee adopted **Candidate A — Distinct types within Category III** with **sub-resolution A.2** (abstract type `Hypothesis` carrying shared structure; four concrete sibling subtypes `BehavioralCluster`, `CoordinationRing`, `CampaignHypothesis`, `AutomationGroup` extending `Hypothesis` and carrying subtype-specific fields). Discussion-phase evidence is preserved in [`docs/rfcs/discussion/q2-evidence.md`](../discussion/q2-evidence.md). Concrete type-definition technology is deferred to the pending substrate-technology RFC ([`decision-log §0003`](../../charter/decision-log.md)).
