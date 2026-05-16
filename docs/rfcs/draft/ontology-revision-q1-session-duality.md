# RFC — Ontology Open Question 1: Session Duality

- **Status:** accepted
- **Authors:** Ghost Trace committee
- **Date:** 2026-05-15
- **Type:** ontology-revision
- **Affects:** [`docs/ontology/entity-model.md`](../../ontology/entity-model.md) (Category I section, particularly Session-related structural properties and examples; Open Modeling Question 1); [`docs/ontology/provenance-model.md`](../../ontology/provenance-model.md) (§Observational Provenance — Session events are foundational Category I primaries); [Charter §2.3](../../charter/constitutional-charter.md#23-provenance-integrity) (pending — binding text depends on this resolution)

> RFCs in Ghost Trace are proposals subject to constitutional review. The [Constitutional Charter](../../charter/constitutional-charter.md) is the authoritative reference. RFCs that conflict with the Charter must either be rejected or accompanied by a corresponding charter amendment (see [`../../charter/amendments.md`](../../charter/amendments.md)).

## Summary

Ontology Open Question 1, recorded in [`docs/ontology/ontology.md` §Open Questions for Committee Resolution](../../ontology/ontology.md) and elaborated in [`docs/ontology/entity-model.md` §Open Modeling Questions](../../ontology/entity-model.md): is `Session` a single Category I entity with reconciliation against operational interpretation, or two entities (`DeclaredSession` as Category I plus `OperationalSession` as Category II)? The question governs the granularity of what counts as a primary observation when Session-related claims arise — declared session boundaries (client SDK reports or authoritative external events) and operational session boundaries (inferred from behavior patterns over the substrate) often disagree precisely in cases where investigation matters most. This RFC opens structured discussion with two candidate resolutions and one explicitly rejected alternative. The RFC does not pick a candidate.

## Motivation

**Why now.** Per [decision-log §0008](../../charter/decision-log.md), the redaction order for pending invariants is §2.5 → §2.3 → §2.4 → §2.6. §2.5 closed at [decision-log §0013](../../charter/decision-log.md). §2.3 (Provenance Integrity) is next. The §2.3 stub commits the system to "every assertion declares, in its structure, the observations and prior assertions that constitute its provenance" — and Session events are a canonical Category I primary. Under Candidate A (single entity), §2.3 references a single Session type when assertions trace back to session-related observations. Under Candidate B (dual entity), §2.3 references `DeclaredSession` (Category I substrate) and `OperationalSession` (Category II construct) distinctly. The two candidates produce structurally different §2.3 binding text. Redaction without prior resolution is therefore not neutral on this question.

**The cost of not resolving.** Silent resolution at §2.3 redaction would pick an answer without committee deliberation. That outcome violates [`ontology-keeper`'s](../../../.claude/skills/ontology/ontology-keeper/SKILL.md) discipline: open Ontology questions are committee-resolved, not infrastructure-resolved. The procedural failure mode is documented in `ontology-keeper` §3 and the precedent for this pre-Gate pattern is established by Q2 ([§0010](../../charter/decision-log.md)) and Q4 ([§0011](../../charter/decision-log.md)) resolutions. Both candidates also leave the substrate's Category I primary surface under-specified for Session-related observations, which has downstream consequences for provenance graph reconstructibility ([`provenance-model.md`](../../ontology/provenance-model.md) Scaffold).

## Constitutional Review

The Q1–Q6 impact analysis prescribed by [`rfc-author` §1](../../../.claude/skills/workflow/rfc-author/SKILL.md).

### Q1 — Which Charter invariants does this RFC touch?

- **§2.3 Provenance Integrity** (pending): touched directly. §2.3 binding text references provenance back to "observations"; the granularity of what counts as a primary observation when Sessions appear depends on this resolution.
- **§2.4 Inferential Influence Disclosure** (pending): touched indirectly. Assertions formed under inference may reference Sessions through `subject_ref` (whose polymorphism is itself an open question — `entity-model.md` Open Modeling Question 3). Under Candidate B, `subject_ref` to a Session may need to distinguish `DeclaredSession` vs `OperationalSession`.
- **§2.1 Observational Integrity** (frozen): consistency check, not amendment. Both candidates preserve Category I substrate immutability for the records that are Category I (Session in Candidate A; `DeclaredSession` in Candidate B). The resolution does not propose any mutation of Category I substrate; it concerns which records ARE Category I.
- **§2.2 Epistemic Separation** (frozen): consistency check. Both candidates preserve the structural distinction between Category I and Category II. Candidate B explicitly assigns one Session form to each category; Candidate A keeps Session as Category I and assigns reconciliation records to Category II.
- **§2.5 Hypothesis Lifecycle Explicitness** (frozen v0.3): not amended. Hypotheses about sessions could still be formed under either candidate; the Q2-A.2 abstract-plus-sibling structure for Category III is independent of how Sessions themselves are typed.
- No FROZEN invariant is amended.

### Q2 — Does this RFC implicitly redefine any term in the glossary?

The term `Session` does not have a current glossary entry. It appears informally in [`entity-model.md` Category I examples](../../ontology/entity-model.md) and is referenced in [Open Modeling Question 1](../../ontology/entity-model.md). Both candidates may require glossary additions:

- Under Candidate A, a single `session` glossary entry establishes `Session` as a Category I primary observation type with reconciliation against operational interpretation handled by Category II constructs (e.g., `SessionReconciliation`).
- Under Candidate B, two glossary entries are needed: `declared session` (Category I primary) and `operational session` (Category II construct).

Either candidate carries a glossary-extension obligation; the asymmetry is in count (one vs two entries) and in the structural classification each entry establishes.

### Q3 — Does this RFC implicitly resolve any of the five open Ontology questions?

This RFC opens structured discussion of Open Question 1. By construction it touches Q1; it does not resolve Q1. Per `ontology-keeper` §3 the verdict is that this RFC is the procedure for raising Q1 to committee, not an instance of implicit resolution.

The RFC interacts with:
- `entity-model.md` Open Modeling Question 3 (`subject_ref` polymorphism). Under Candidate B, `subject_ref` to a Session needs to distinguish two types, making the polymorphism question more pressing. Under Candidate A, the question is less acute (one Session type either way). The interaction is recorded under Open Questions below; resolution is empirically assessed at §2.3 Step 1.1 per [§0014](../../charter/decision-log.md).
- `entity-model.md` Open Modeling Question 2 (Identity tiers — `ActorRef`/`Identity`/`Cluster`). Sessions are typically owned by an identity tier. Under either candidate, the identity-tier formalization remains pending; the Q1 resolution does not depend on it.

The RFC does not touch ontology.md Q3 (independence formal definition) or Q5 (influence propagation).

### Q4 — Does this RFC require Charter amendment?

No. The resolution constrains §2.3's content. §2.3 is pending — not amendable yet, only redactable for the first time. If this RFC is resolved before §2.3 redaction, no charter-amendment chain is needed; the §2.3 redaction itself proceeds within its normal committee-mode procedure per [`invariant-redactor`](../../../.claude/skills/constitutional/invariant-redactor/SKILL.md). The procedural ordering enforced by this RFC eliminates the silent-resolution and vague-binding-text failure modes.

### Q5 — Does this RFC introduce a new invariant?

No. The RFC resolves a modeling question internal to the Ontology. §2.3 is a separate constitutional act with its own redaction procedure. Per the constitutional minimalism rule ([`.claude/CLAUDE.md` §7](../../../.claude/CLAUDE.md)), introducing a new invariant here would be unjustified.

### Q6 — Does this RFC propose ceremony without behavioral consequence?

No. The two candidates produce different Category I primary type surfaces, different provenance-chain shapes when Session-related claims arise, different query patterns over Session-related observations, and different forms of §2.3 binding text. Deleting this RFC would either force silent resolution at §2.3 redaction or produce un-redactable binding text. The behavioral consequence is direct — and reflects the documented disagreement between declared and operational session boundaries in cases where investigation matters.

## Proposal

Two candidate resolutions, each with structural claim, provenance implication, category boundary check, query pattern, and pros/cons. The RFC does not pick. The discussion phase gathers evidence and arguments; resolution produces the final answer.

### Candidate A — Single `Session` entity with reconciliation

**Structural claim.** One Category I entity type `Session`. The substrate records `Session` events as they arrive (client SDK reports, authoritative external session-boundary signals, etc.). Discrepancies between the declared session boundaries and the system's later operational reading of where a session "really was" are recorded as additional Category II constructs (e.g., `SessionReconciliation`) referring to the canonical `Session`. The reconciliation construct is a Category II derivation under a versioned operational definition.

**Provenance implication.** Under §2.3 binding text, assertions referencing a Session trace observational provenance to one Category I primary type: `Session`. Where a reconciliation has been applied, the assertion also carries an inferential reference (per §2.4) to the `SessionReconciliation` construct that altered the operational reading. The two layers (Category I observation; Category II reconciliation) are visible in the provenance graph; the substrate record itself remains the declared form.

**Category boundary check.** Respects §2.2 separation: `Session` is Category I (immutable, primary); `SessionReconciliation` is Category II (deterministic derivation under a definition). No category violation.

**Query pattern.** A query asking "what was happening in session X" returns the `Session` record. A query asking "what was happening in session X under the current operational reading" first looks up applicable `SessionReconciliation` constructs and applies them. Polymorphic-on-reconciliation-state at the query layer; uniform-on-type at the substrate layer.

**Pros.**

- One canonical Category I type for Session-related observations; the substrate surface is uniform.
- Reconciliation as Category II construct preserves the asymmetry between "what was reported" (immutable) and "what is operationally read" (deterministic, re-derivable).
- One glossary entry suffices for Session-related vocabulary at the substrate level.
- Forward-compatible with future reconciliation strategies: a new reconciliation definition produces new Category II constructs without altering the substrate.

**Cons.**

- The structural distinction between declared and operational session boundaries lives as a relationship (Category II referring to Category I), not as a type-level commitment. Readers and operators must traverse the relationship to see the asymmetry.
- Queries that need the operational reading by default have an extra indirection (lookup of applicable reconciliation).
- If reconciliation becomes the dominant operational form, Candidate A leaves the operational reading without first-class type representation.

### Candidate B — `DeclaredSession` (Category I) + `OperationalSession` (Category II) as distinct entities

**Structural claim.** Two entity types in different categories. `DeclaredSession` is a Category I primary observation type: the substrate records what client SDKs or authoritative external sources reported about session boundaries. `OperationalSession` is a Category II operational construct: the system's reading of where a session "really was," derived deterministically from observations under a versioned operational definition. The two types are distinct at the substrate level.

**Provenance implication.** Under §2.3 binding text, assertions referencing a Session distinguish their target by type: provenance to `DeclaredSession` traces to Category I substrate primaries; provenance to `OperationalSession` traces through the operational construct to its derivation inputs (which may include `DeclaredSession` records and other Category I primaries). The provenance graph shows the asymmetry at the type level, not as a relationship requiring traversal.

**Category boundary check.** Respects §2.2 separation: `DeclaredSession` is Category I; `OperationalSession` is Category II. The boundary is explicit at the type level. Cross-category references between the two are typed transformations.

**Query pattern.** A query asking "what was reported about session X" returns `DeclaredSession` records. A query asking "what was operationally happening" returns `OperationalSession` constructs. Disagreement between the two is visible by structural type, not by reconciliation lookup.

**Pros.**

- The asymmetry between declared and operational session boundaries is first-class at the type level. Investigation patterns that depend on the disagreement (e.g., forensic analysis of session-boundary attacks) get type-level visibility.
- §2.2 separation is explicit: each form of Session is unambiguously categorized.
- Provenance chains are clearer: the type of the Session reference indicates which category the chain terminates in.
- Future evolution: adding alternate operational session definitions becomes another Category II construct alongside `OperationalSession` (e.g., versioned definitions); the Category I `DeclaredSession` substrate remains stable.

**Cons.**

- Two canonical terms must be defined and maintained in the glossary.
- Common operations (queries that don't care about the distinction) must handle both types explicitly or rely on a projection that unifies them at the read layer.
- If the disagreement between declared and operational session boundaries turns out to be rare, Candidate B imposes ceremony where Candidate A would not.

## Alternatives Considered

### Candidate C — Polymorphic `Session` with dynamic category classification (REJECTED)

**Structural claim.** One `Session` entity type with a `category` discriminator field. The same record can be classified at runtime as either Category I or Category II depending on context (declared-source marker, operational-derivation marker, etc.).

**Reason for rejection.** This candidate collapses Q1 by deferring it to runtime. The category-boundary commitment of [§2.2 Epistemic Separation](../../charter/constitutional-charter.md#22-epistemic-separation) is structural — Category I and Category II have different substrate semantics, different lifecycle rules, and different operations permitted upon them. A record whose category is determined at runtime is, by construction, in violation of §2.2's "declared at construction and not changeable" requirement. Per [`ontology-keeper` §3](../../../.claude/skills/ontology/ontology-keeper/SKILL.md), this is precisely the failure mode the skill exists to prevent: an open question resolved by infrastructure (runtime classification) rather than by committee. Candidate C also fails §4 criterion 1 (structural enforceability) — the category boundary becomes a runtime check, not a structural property.

## Open Questions

This RFC explicitly defers the following:

- **Reconciliation rules under Candidate A.** What counts as a reconciliation event, how often it fires, what triggers it, how multiple reconciliations interact — all deferred to post-Q1 specification work. The RFC commits to "reconciliation as Category II construct" structurally; the specific reconciliation semantics are downstream.
- **Identity tiers question** ([`entity-model.md` Open Modeling Question 2](../../ontology/entity-model.md)). How `ActorRef`, `Identity`, and `Cluster` relate to Sessions — and whether a Session is owned by an `ActorRef` or by an `Identity` — touches Q1 tangentially but does not depend on it. Remains open; may require its own pre-Gate later (likely before §2.4 or as part of §2.6 prep depending on resolution).
- **Subject reference polymorphism** ([`entity-model.md` Open Modeling Question 3](../../ontology/entity-model.md)). Whether `subject_ref` is a single polymorphic field or per-category fields. Under Candidate B, the question becomes more pressing because `subject_ref` to a Session needs to distinguish `DeclaredSession` from `OperationalSession`. Per [decision-log §0014](../../charter/decision-log.md), this question's blocking status will be assessed empirically at §2.3 Step 1.1 anchor inventory rather than resolved beforehand.
- **Drift between canonical Q1 form and `entity-model.md` Open Modeling Question 1.** The canonical Q1 in `ontology.md` asks the question in single-sentence form; `entity-model.md` elaborates with the explicit Candidate B form ("`DeclaredSession` as Category I, `OperationalSession` as Category II"). The elaboration is helpful framing but predates this RFC's structured candidates; whether the elaboration should be amended in light of this RFC's Candidate A formulation is a question for the redaction phase.
- **Cross-domain Session semantics.** Future Ghost Trace applications beyond the first applied domain (per [Charter §1](../../charter/constitutional-charter.md#1-thesis)) may have different Session-equivalent concepts. Whether Q1's resolution generalizes across domains or is scoped to the first domain is out of scope for this RFC.

## Anti-Patterns to Avoid

- **Resolving Q1 by code.** Pushing the decision into substrate technology selection, projection rules, or schemas-level discriminators picked at implementation time. The resolution belongs to the Ontology, not to substrate technology or projection layer.
- **Treating Q1 as merely a naming issue.** The question is a category-boundary question with substantive consequences for §2.3 binding text shape, provenance graph traversal patterns, and query semantics. Naming follows the structural decision; it does not precede it.
- **Silent revision in a future `entity-model.md` redaction.** A future redaction of `entity-model.md` Category I that picks one candidate without acknowledging Q1 being resolved is an instance of the failure mode `ontology-keeper` prevents. The redaction must reference this RFC's resolution path.
- **Combining Candidates A and B by allowing both at the type level.** A "Session with optional operational subtype" hybrid presents two surfaces simultaneously and is operationally equivalent to Candidate C; reject for the same reason.
- **Premature reconciliation rule specification under Candidate A.** The Open Questions section explicitly defers the specifics of reconciliation semantics. Embedding reconciliation rules in §2.3 binding text would over-commit the Charter to operational details that belong in Ontology or Architecture.

## Migration and Backward Compatibility

No historical Session records exist at this point. The RFC is forward-looking. Two asymmetries are noted for committee consideration:

- **Lock-in asymmetry.** Candidate A → Candidate B migration would require retrospectively classifying Category II reconciliation records and possibly promoting some to a new Category II `OperationalSession` type, with corresponding refactor of provenance references. Candidate B → Candidate A collapse would require merging two types into one and converting type-level distinctions to relationship-level distinctions; potentially less disruptive depending on how downstream consumers branched on type.
- **Provenance asymmetry.** Existing assertions (in future, post-implementation) that reference Sessions via `subject_ref` carry typed references. Under Candidate A, all such references resolve to `Session`; under Candidate B, references distinguish `DeclaredSession` and `OperationalSession`. Migration in either direction requires re-typing existing provenance chains.

The Charter's [Phase 3 / Phase 4 replay contracts](../../architecture/replay-model.md) impose no specific obligation on this RFC: both candidates support reconstructive and retrospective replay of Session-related events, provided the Category I substrate is immutable per §2.1.

## References

- [`docs/ontology/ontology.md` §Open Questions for Committee Resolution](../../ontology/ontology.md) — canonical statement of Q1.
- [`docs/ontology/entity-model.md` §Category I](../../ontology/entity-model.md) — Session examples and Category I structural properties; §Open Modeling Question 1 (Q1 elaboration); Open Modeling Question 2 (Identity tiers); Open Modeling Question 3 (subject_ref polymorphism).
- [`docs/ontology/provenance-model.md`](../../ontology/provenance-model.md) — Scaffold; §Observational Provenance form referenced by §2.3.
- [`docs/charter/constitutional-charter.md` §2.3](../../charter/constitutional-charter.md#23-provenance-integrity) — stub whose redaction depends on Q1 resolution.
- [`docs/charter/decision-log.md` §0008](../../charter/decision-log.md) — redaction order §2.5 → §2.3 → §2.4 → §2.6.
- [`docs/charter/decision-log.md` §0009](../../charter/decision-log.md) — §2.5 prep precedent (Q2 + Q4 pre-Gate pattern).
- [`docs/charter/decision-log.md` §0010](../../charter/decision-log.md) — Q2 resolution methodological precedent.
- [`docs/charter/decision-log.md` §0011](../../charter/decision-log.md) — Q4 resolution; established the §2 L41 forward-reference precedent admitting pending-target forward-references in Charter binding text.
- [`docs/charter/decision-log.md` §0013](../../charter/decision-log.md) — §2.5 closure with four methodological observations including the pre-Gate Ontology RFC pattern (observation 2) and the forward-reference contract as structural commitment (observation 3).
- [`docs/charter/decision-log.md` §0014](../../charter/decision-log.md) — §2.3 redaction plan; this RFC's enclosing pre-Gate.
- [`.claude/skills/ontology/ontology-keeper/SKILL.md`](../../../.claude/skills/ontology/ontology-keeper/SKILL.md) — registry of the five open Ontology questions.
- [`.claude/skills/workflow/rfc-author/SKILL.md`](../../../.claude/skills/workflow/rfc-author/SKILL.md) — Q1–Q6 impact analysis procedure.

## Decision Record

Resolved by [`docs/charter/decision-log.md` §0015 — Q1 resolution](../../charter/decision-log.md). The committee adopted **Candidate B — `DeclaredSession` (Category I) + `OperationalSession` (Category II) as distinct entities**, with two committee extensions:

1. **Determinism commitment.** `OperationalSession` is deterministically derived from `DeclaredSession` plus other Category I inputs under a versioned operational definition per [§2.2 Category II](../../charter/constitutional-charter.md#22-epistemic-separation) requirement. Encoded structurally in `entity-model.md` Category II's `OperationalSession` paragraph.
2. **Identity-tier consistency default.** `DeclaredSession` and `OperationalSession` share identity-tier references by default unless the operational definition explicitly overrides them. Procedural default until Identity tiers is formally resolved.

The §0014 lazy pre-Gate cascade trigger fires with this resolution: Q3 (subject_ref polymorphism) is now blocking for §2.3 redaction Step 1.2 under Candidate B's selection. Q3 RFC opened at `discussion` status as cascade enactment ([`ontology-revision-q3-subject-ref-polymorphism`](./ontology-revision-q3-subject-ref-polymorphism.md)).

Discussion-phase evidence is preserved in [`docs/rfcs/discussion/q1-evidence.md`](../discussion/q1-evidence.md).
