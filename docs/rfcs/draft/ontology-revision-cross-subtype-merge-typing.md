# RFC — Cross-subtype merge produced-record typing

- **Status:** discussion
- **Authors:** Ghost Trace committee
- **Date:** 2026-05-21
- **Type:** ontology-revision
- **Affects:** [`docs/ontology/entity-model.md`](../../ontology/entity-model.md) §Cross-subtype operations; [`docs/ontology/lifecycle-semantics.md`](../../ontology/lifecycle-semantics.md) §Merge; [Charter §2.5](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) (frozen v0.3 — minor amendment surface only; this RFC produces no §2.5 prose change unless the resolution requires it).

> RFCs in Ghost Trace are proposals subject to constitutional review. The [Constitutional Charter](../../charter/constitutional-charter.md) is the authoritative reference. RFCs that conflict with the Charter must either be rejected or accompanied by a corresponding charter amendment (see [`../../charter/amendments.md`](../../charter/amendments.md)).

## Summary

[`entity-model.md` §Cross-subtype operations](../../ontology/entity-model.md) states: *"Cross-subtype merge (e.g., a `BehavioralCluster` and a `CoordinationRing` recognized as the same underlying phenomenon) is structurally permitted but requires a typed transformation: the merge operation produces a typed output record. Whether the produced record is a third concrete subtype or an abstract record with subtype-elision is a question whose resolution is deferred to `lifecycle-semantics.md` post-Q4 redaction."* This RFC opens structured discussion of that deferral with three candidate typings and one explicitly rejected alternative. The RFC does not pick a candidate.

The within-subtype lifecycle surface is at full coverage as of [`decision-log §0119`](../../charter/decision-log.md): all six lifecycle operations (form / promote / demote / dissolve / merge / split) operate on all four concrete subtypes (BehavioralCluster, AutomationGroup, CampaignHypothesis, CoordinationRing) across helpers, CLIs, and HTTP T4 endpoints. The cross-subtype merge surface has been blocked at the helper layer pending this resolution: each `Merge*` helper validates that both antecedent formation hashes resolve to its OWN subtype, returning `ErrTargetWrongType` otherwise. Lifting that within-subtype guard requires committee-resolved produced-record typing — without it, the helper would either silently pick a typing (resolving the entity-model deferral by implementation) or fail to commit a structurally-defensible merge record.

## Motivation

The §0119 closure of the §0098 RFC arc surfaces cross-subtype merge as the immediate next structural commitment: within-subtype coverage is at 24/24, and operators have a substrate-grounded surface for combining hypotheses *within* a subtype but no surface for combining them *across* subtypes. Three failure modes appear if the typing question is left deferred:

- **Implementation-resolved typing.** A `MergeAcrossSubtypes` helper added without RFC-level deliberation silently picks one of the candidate typings below. The choice becomes infrastructure-resolved rather than committee-resolved — the precise failure mode [`ontology-keeper`](../../../.claude/skills/ontology/ontology-keeper/SKILL.md) exists to prevent.
- **Asymmetric typing across operator surfaces.** The CLI, HTTP T4, and helper paths each pick their own typing for the produced record. Operators see different shapes on different surfaces for the same underlying merge operation; the §2.5 lifecycle event itself becomes channel-dependent, violating the §2.5 frozen v0.3 contract that lifecycle events are Cat I primary observations independent of the commit channel.
- **Cross-subtype merge effectively forbidden.** Without a committee typing, operators avoid cross-subtype merge entirely; the entity-model claim that cross-subtype merge "is structurally permitted" becomes hollow — permitted in principle, unreachable in practice.

The cost of not resolving the typing before further implementation is therefore concrete: cross-subtype merge either lands with silent typing OR remains structurally-permitted-but-operationally-unreachable.

## Constitutional Review

The Q1–Q6 impact analysis prescribed by [`rfc-author` §1](../../../.claude/skills/workflow/rfc-author/SKILL.md).

### Q1 — Which Charter invariants does this RFC touch?

- **§2.5 Hypothesis Lifecycle Explicitness** (frozen v0.3): touched. Merge is one of the six §2.5 lifecycle operations. §2.5 BC1–BC6 are channel-agnostic; the cross-subtype merge produced-record typing does not amend any BC — it operationalizes within the existing BC surface. If the resolution introduces a new concrete subtype (Candidate α) or a new abstract record (Candidate β), the §2.5 surface continues to cover the new record type without amendment per BC1's "Cat I lifecycle events" framing — the merge record IS the lifecycle event regardless of its produced-record typing.
- **§2.3 Provenance Integrity** (frozen v0.4): touched. The produced record's `antecedent_formation_hashes` field is the structural provenance link back to the merged subtypes. Cross-subtype merge introduces a heterogeneous antecedent-list shape (two antecedents of potentially different `message_type` per the [§0042](../../charter/decision-log.md) typed-Cat-I-protos pattern). All three candidates preserve §2.3's provenance-chain shape; the candidates differ in how the produced record's OWN typing relates to its antecedents'.
- **§2.4 Inferential Influence Disclosure** (frozen v0.5): touched indirectly. Influence-bearing assertions formed under the merged hypothesis's influence inherit the produced record's typing as their `subject_ref_hypothesis` payload (per [`§0016`](../../charter/decision-log.md) Q3 resolution). Different candidates yield different `subject_ref_hypothesis` shapes — a `CompositeHypothesis` (α) vs. an abstract `Hypothesis` (β) vs. one of the four concrete subtypes (γ).
- **§4 Constitutional Design Rule** (frozen v0.2): consistency check, not amendment. The four falsifiability criteria are applied to each candidate; specific candidate evaluations are deferred to the discussion phase.
- No FROZEN invariant is amended.

### Q2 — Does this RFC implicitly redefine any term in the glossary?

The terms `hypothesis`, `behavioral cluster`, `automation group`, `campaign hypothesis`, `coordination ring`, `merge`, `formation`, and `subject_ref_hypothesis` are in [`docs/glossary.md`](../../glossary.md) and [`.claude/CLAUDE.md` §3 canonical vocabulary](../../../.claude/CLAUDE.md). All three candidates preserve these definitions.

Candidate α introduces a new term (`CompositeHypothesis` or analogous) — if α is chosen, the glossary gains a new entry. Candidate β introduces the operationalization of the abstract `Hypothesis` type currently described in entity-model.md but not glossary-tracked — if β is chosen, `Hypothesis` graduates from informal-document term to glossary entry. Candidate γ introduces no new term; the existing four subtypes carry the merged outputs.

### Q3 — Does this RFC implicitly resolve any of the five open Ontology questions?

This RFC opens a question NOT in the original five-OQ list in [`ontology.md` §Open Questions for Committee Resolution](../../ontology/ontology.md) — it is a sub-question of Q2 (Cat III subtypes), which resolved at [`§0010`](../../charter/decision-log.md) as Candidate A.2 (four distinct concrete subtypes). The cross-subtype merge produced-record typing is a Q2 *follow-on* per the entity-model deferral; analogous in framing to [`ontology-revision-layer-b-deep-criterion`](./ontology-revision-layer-b-deep-criterion.md) being a Q4 follow-on.

The RFC touches Q3 (subject_ref_polymorphism, resolved at [`§0016`](../../charter/decision-log.md) as Candidate B with per-Category granularity) indirectly: per-Category granularity means a `subject_ref_hypothesis` field already accommodates any subtype; the RFC's candidates differ in what shapes that field's payload can carry, not in whether the field exists.

### Q4 — Does this RFC require Charter amendment?

No. §2.5 is frozen v0.3 with a channel-agnostic BC surface that already accommodates any Cat I lifecycle event regardless of produced-record typing. The cross-subtype merge resolution operationalizes within the existing §2.5 surface; if the resolution adds prose to §2.5 it would be a minor amendment (v0.3 → v0.3.1 or v0.6), but the discussion-phase RFC anticipates no prose change.

### Q5 — Does this RFC introduce a new invariant?

No. The RFC codifies a structural rule for an existing pending operation (cross-subtype merge) within an already-frozen invariant surface (§2.5). No new constitutional claim is added.

### Q6 — Does this RFC propose ceremony without behavioral consequence?

No. Different typings produce different operator-visible surfaces (proto schemas, CLI flags, JSON shapes, HTTP routes) and different downstream replay shapes. A cross-subtype merge of {BehavioralCluster, CoordinationRing} under Candidate α produces a `CompositeHypothesis` record (distinct proto, distinct replay handler); under Candidate β produces an abstract `Hypothesis` record (single proto, multi-antecedent replay handler); under Candidate γ-{BC,CR}=CR produces a `CoordinationRingFormation` record (existing proto reused, multi-antecedent extension to the existing replay handler). The three candidates partition the operator-visible surface differently and produce different downstream replay-correctness obligations.

## Proposal

Three candidate typings, each presented with structural claim, dependency on other frozen / pending invariants, and pros and cons. The RFC does not pick. Combined candidates — e.g., γ for some pairs and α for others — are an explicit option for the discussion phase.

### Candidate α — Third (fifth) concrete subtype

**Structural claim.** Introduce a new concrete subtype `CompositeHypothesis` (working name) as a fifth sibling extending the abstract `Hypothesis`. Each cross-subtype merge produces a `CompositeHypothesis` record carrying: (a) the antecedent formation hashes (heterogeneous — antecedents may be of any two of the four existing subtypes); (b) a `combined_subtypes` field recording which two subtypes were merged (ordered pair or unordered set, per discussion); (c) the union of the merged hypotheses' subtype-specific surfaces (membership lists, signature references, etc.), structurally accommodated by the new subtype's proto. Within-subtype merge continues to produce its own subtype's formation record (BC merge → BC formation, AG merge → AG formation, ...); only CROSS-subtype merge produces `CompositeHypothesis`.

**Dependency.** None on pending invariants. The four concrete subtypes are frozen at entity-model.md §Concrete subtypes; introducing a fifth is an entity-model addition, not an amendment.

**Pros.**

- The produced record's typing is explicit and unambiguous. A reader of a `CompositeHypothesis` record knows from its type that it is a cross-subtype-merge output; the surface carries the structural commitment that within-subtype-merge outputs do not.
- The four within-subtype helper/CLI/HTTP surfaces remain unchanged. The cross-subtype surface is a separate code path (new helper, new CLI, new HTTP route) — no risk of asymmetric within-subtype regression.
- Per-subtype proto reuse preserved. Each of the four existing subtypes' formation/lifecycle protos is untouched; the fifth subtype's proto is new but mechanical.

**Cons.**

- Asymmetry in the type system. Four subtypes are *recognition-pattern outputs* (formation patterns produce them); the fifth is a *lifecycle-operation output* (merge produces it). The two roles are structurally different; mixing them at the same type-system layer obscures the distinction.
- Type explosion under composition. A cross-subtype merge of a `CompositeHypothesis` with one of the four original subtypes — operationally permitted under §2.5's BC1 — produces a `Composite{Composite, OriginalSubtype}` record; if these compose further the type structure grows combinatorially.
- The glossary gains a term whose meaning is "merger of two of the other four" — a self-referential definition that may not survive future merge-of-merge scenarios.

### Candidate β — Abstract record with subtype-elision

**Structural claim.** The cross-subtype merge produces an *abstract* `MergedHypothesis` (working name) — or, equivalently, materializes the abstract `Hypothesis` base type from entity-model.md §Category III — that is NOT bound to any of the four concrete subtypes. The four subtypes remain *sibling concrete extensions* per the existing entity-model; the abstract record sits at the parent level and represents the merged-without-collapse outcome. Fields: (a) heterogeneous antecedent formation hashes; (b) a discriminator-list field recording which subtypes were merged (open-ended set, accommodates merge-of-merge by appending). Within-subtype merge continues to produce its concrete subtype's formation record (unchanged).

**Dependency.** Requires materializing the entity-model.md §Category III abstract `Hypothesis` type at the proto layer. The entity-model currently describes it as "an abstract type from which the four concrete subtypes inherit" without specifying whether the abstract is materializable as a substrate record. If β is the resolution, the abstract becomes materializable for cross-subtype merge outputs (and possibly extends to other future uses).

**Pros.**

- No new sibling subtype. The four concrete subtypes remain four; the abstract base accommodates the merged record without expanding the recognition-pattern-output count.
- Merge-of-merge composes naturally. Successive merges accumulate discriminator-list entries on the same abstract record type; no type explosion.
- The `subject_ref_hypothesis` payload shape is uniform across all cross-subtype-merge outputs (always the abstract type); §2.4 influence-disclosure code paths see a single shape, not five.

**Cons.**

- The abstract type is now load-bearing at the substrate layer. The entity-model's prior framing ("an abstract type from which the four concrete subtypes inherit") shifts from organizational-abstraction to substrate-materialized. Whether this drift is acceptable is itself a structural question.
- Operator surface loses information. A reader of a `MergedHypothesis` record sees the discriminator-list but does not see typed structure from the merged subtypes; the union-of-subtype-specific-surfaces is either flattened or discarded.
- Replay shape diverges. The replay handler for cross-subtype merge must reconstruct the antecedent subtypes from their formation hashes (auto-detect per the §0091 pattern); within-subtype replay reconstructs from a single typed antecedent. The two replay paths share less code than under α.

### Candidate γ — Per-pair canonical-merge typing

**Structural claim.** For each ordered (or unordered) pair of source subtypes, pre-declare a canonical merge-target from the existing four. C(4,2) = 6 unordered pairs (or 4×3 = 12 ordered pairs); each pair gets a deterministic target subtype. No new type is introduced. Within-subtype merge continues to produce its own subtype (unchanged); cross-subtype merge produces the pre-declared target.

Illustrative (NOT a recommendation — the discussion phase decides):
- {BC, AG} → AG (automation-signature dominates over shared-operatorship inference)
- {BC, CR} → CR (coordination structure dominates over shared-operatorship)
- {BC, CH} → CH (event-centric reframing of actor-centric)
- {AG, CR} → CR (coordinated automation reframed as coordination)
- {AG, CH} → CH (campaign of automated activity)
- {CR, CH} → CH (coordination-as-campaign-membership)

**Dependency.** None on pending invariants. The six (or twelve) pairs are an entity-model addition table; the existing four subtypes' protos accommodate the merged records without proto change.

**Pros.**

- No new types. The four-subtype taxonomy is preserved; the glossary surface is unchanged.
- The produced record's typing is determined at the entity-model level, not at the merge invocation. Operators cannot pick an inappropriate target — the table is structural.
- Replay shape uniform with within-subtype merge. The target subtype's existing replay handler accommodates the merged antecedent-list (with a one-line extension to accept heterogeneous antecedents).

**Cons.**

- The pre-declared table requires committee defense for each entry. Why {BC, AG} → AG and not → BC? Why {AG, CH} → CH and not → AG? Each cell of the table is a small structural commitment; defending all six (or twelve) at once is more committee surface than introducing one new type.
- Subtype-specific surface conflicts. When {BC, CR} merges into a CR, the BC's membership-list does not fit CR's pairwise-relationship surface naturally; the merge either discards BC's surface or extends CR's surface to accommodate it (per-pair extension; another committee defense).
- Operator confusion. A `CoordinationRingFormation` record may be either (a) recognition-pattern output OR (b) cross-subtype merge output with original subtypes recorded in antecedents. Reading the record requires inspecting its antecedent list to know which case applies.

### Combined-candidate forms

The discussion phase considers combinations:

- **α + γ.** Some pair-canonicals produce existing subtypes (γ); pairs without a defensible canonical produce `CompositeHypothesis` (α). Reduces type explosion while preserving operator-visible structure where defensible.
- **β + γ.** Most pairs produce abstract `MergedHypothesis` (β); selected pairs with strong structural canonicals (e.g., {BC, CH} → CH if event-centric dominates) produce the canonical concrete subtype.
- **α + β.** Cross-subtype merge produces α for first-level merges and β for merge-of-merge — preserves typing detail at the first composition layer; flattens at deeper composition.

The RFC does not recommend any combination form. The discussion phase considers them on equal footing with the three single-candidate forms.

## Alternatives Considered

### Reject cross-subtype merge entirely (REJECTED)

Cross-subtype merge is structurally forbidden; operators recognizing two cross-subtype hypotheses as the same underlying phenomenon must record the recognition as an informational Assertion only, without producing a §2.5 merge record. Rejected because it conflicts with the [entity-model.md §Cross-subtype operations](../../ontology/entity-model.md) frozen-by-redaction statement that *"Cross-subtype merge ... is structurally permitted."* Closing the surface here requires amending the entity-model, not closing it at this RFC.

### Implementation-resolved typing — first-implemented helper picks (REJECTED)

The first cross-subtype merge helper to land picks its produced-record typing; subsequent helpers inherit the choice. Rejected on `ontology-keeper` grounds: this is precisely the implementation-resolves-ontology failure mode the gate at [CLAUDE.md §6.2](../../../.claude/CLAUDE.md) forbids. The decision belongs to the committee, not to the first operator who needs the feature.

### Per-invocation operator choice (META-ANTI-PATTERN)

The cross-subtype merge helper takes a `produced_record_type` parameter; the operator picks at invocation time. Operationally tempting (flexibility); structurally rejected because it makes the produced-record typing operator-discretionary, not substrate-deterministic — violates [`§4`](../../charter/constitutional-charter.md#4-constitutional-design-rule) criterion 4 (independence of operator interpretation). Two operators recognizing the same underlying phenomenon would produce different typed records on the same substrate. Not a candidate; surfaced here for procedural transparency.

## Open Questions

This RFC explicitly defers the following:

- **Cross-subtype merge enablement criterion.** When may an operator initiate cross-subtype merge? The within-subtype merge surface uses the same criterion as within-subtype operations (two antecedent formation hashes of the SAME subtype). Cross-subtype merge needs an analogue: are both antecedents always permitted to merge, OR is there a structural compatibility check (e.g., shared actor membership)? The criterion question is orthogonal to the produced-record typing question and may be a follow-on RFC.
- **Merge-of-merge composition.** If a `CompositeHypothesis` (α) merges with a concrete subtype, what is the typing of the third-level record? Same applies to β and γ. Each candidate has a default; whether the default is the right answer is open.
- **Cross-subtype split.** Symmetric to merge: if a hypothesis is recognized as containing multiple distinct phenomena that span subtypes, what are the successors' typings? The split surface for cross-subtype is a separate question; this RFC does not address it.
- **Cross-subtype promote / demote / dissolve.** The three other lifecycle operations on a cross-subtype-merge output need their own typing analyses (especially under α where a new subtype's promotion semantics need definition). Out of scope for this RFC; named follow-ons.
- **Interaction with §2.4 declared-influence chains.** A cross-subtype-merge output is potentially the basis for downstream influence-bearing assertions per §2.4. Whether the influence-chain semantics differ for cross-subtype merge outputs vs. within-subtype merge outputs is open.
- **Hypothesis (abstract) glossary entry.** If β is the resolution, `Hypothesis` becomes a load-bearing substrate-materialized type and needs a canonical glossary definition. If α or γ is the resolution, `Hypothesis` remains an organizational abstraction. Whether to pre-emptively glossarize is open.

## Anti-Patterns to Avoid

- **Resolving the typing by implementation.** Hard-coded produced-record typing in a `MergeAcrossSubtypes` helper without RFC. The typing belongs to the Ontology and to entity-model.md, not to operational code.
- **Asymmetric typing across operator surfaces.** The CLI, HTTP T4, and helper paths each pick their own typing. The §2.5 BC1 frozen surface requires lifecycle events to be channel-agnostic Cat I primary observations; channel-dependent typing violates this.
- **Cross-subtype merge with discarded subtype-specific surfaces.** When merging across subtypes, naively discarding the merged hypotheses' subtype-specific structures (membership lists, signature references, pairwise relations, event memberships) without explicit committee-resolved discipline. The merged record's structure must structurally accommodate or explicitly elide the antecedents' surfaces; silent discarding is structure loss disguised as merge.
- **Conflating cross-subtype merge with reclassification.** Reclassification per [`lifecycle-semantics.md`](../../ontology/lifecycle-semantics.md) is a Cat II operation (operational construct under a versioned definition); cross-subtype merge is a Cat III lifecycle event. The two are distinct in shape, semantics, and §2.5 frozen-text classification.

## Migration and Backward Compatibility

No historical cross-subtype merge records exist (the helper layer has blocked the surface since §0049 / §0060 / §0067 / §0074 introduced the per-subtype merge helpers). The RFC is forward-looking.

Lock-in asymmetry across candidates:

- Candidate α (third concrete subtype) is the easiest to extend later: introducing additional cross-subtype-only types (e.g., one per pair) is a typed extension of the entity-model.
- Candidate β (abstract record) carries the most coupling to the entity-model's abstract-vs-concrete distinction; retrofitting to α or γ requires re-materializing every prior `MergedHypothesis` record as one of the four concrete subtypes (a §2.1 substrate-immutability conflict — supersession is the only legal path).
- Candidate γ (per-pair canonical) is the most defensible against committee-redaction discipline (the table is bounded and finite) but the hardest to revisit if a pair's canonical proves wrong in practice; per-pair re-typing IS a committee action under γ, but the prior records are already produced under the old typing and cannot be retyped per §2.1.

The [Phase 3 / Phase 4 replay contracts](../../architecture/replay-model.md) require that any lifecycle event, including cross-subtype merge, is replayable. All three candidates satisfy this in principle, but the replay-handler shape differs (per Q6 evaluation above). Replay-handler implementation is a follow-on PR regardless of which candidate is chosen.

## References

- [`docs/ontology/entity-model.md` §Cross-subtype operations](../../ontology/entity-model.md) — canonical statement of the deferral.
- [`docs/ontology/lifecycle-semantics.md` §Merge](../../ontology/lifecycle-semantics.md) — current scaffold; cross-subtype Q2-A.2 deferral noted.
- [`docs/ontology/ontology.md` §Open Questions for Committee Resolution](../../ontology/ontology.md) — the five canonical OQs; cross-subtype merge typing is a Q2 follow-on, not in the five.
- [`docs/charter/constitutional-charter.md` §2.5](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) — frozen v0.3; channel-agnostic lifecycle surface that accommodates any produced-record typing.
- [`docs/charter/constitutional-charter.md` §2.3](../../charter/constitutional-charter.md#23-provenance-integrity) — frozen v0.4; antecedent-link provenance shape.
- [`docs/charter/constitutional-charter.md` §2.4](../../charter/constitutional-charter.md#24-inferential-influence-disclosure) — frozen v0.5; subject_ref_hypothesis payload shape.
- [`docs/charter/decision-log.md` §0010](../../charter/decision-log.md) — Q2-A.2 resolution: four distinct concrete subtypes.
- [`docs/charter/decision-log.md` §0016](../../charter/decision-log.md) — Q3 resolution: per-Category subject_ref granularity.
- [`docs/charter/decision-log.md` §0049 + §0060 + §0067 + §0074](../../charter/decision-log.md) — per-subtype within-subtype merge helpers (within-subtype guard via `ErrTargetWrongType`).
- [`docs/charter/decision-log.md` §0119](../../charter/decision-log.md) — §0098 auth-scope RFC arc closure; within-subtype lifecycle surface now at full 24/24 coverage.
- [`.claude/skills/ontology/ontology-keeper/SKILL.md`](../../../.claude/skills/ontology/ontology-keeper/SKILL.md) — registry of open Ontology questions + Q2-follow-on framing precedent.

## Decision Record

Pending. The discussion phase opens with this RFC at `Status: discussion`; resolution will be recorded in `docs/charter/decision-log.md` and reflected back here on acceptance.
