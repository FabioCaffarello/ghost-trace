# RFC — Cross-subtype split: typing + enablement (consolidated)

- **Status:** accepted
- **Authors:** Ghost Trace committee
- **Date:** 2026-05-21 (opened); 2026-05-22 (resolved)
- **Type:** ontology-revision
- **Affects:** [`docs/ontology/entity-model.md`](../../ontology/entity-model.md) §Cross-subtype operations; [`docs/ontology/lifecycle-semantics.md`](../../ontology/lifecycle-semantics.md) §Split; [Charter §2.5](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) (frozen v0.3 — minor amendment surface only; this RFC anticipates no §2.5 prose change unless the resolution requires it).

> RFCs in Ghost Trace are proposals subject to constitutional review. The [Constitutional Charter](../../charter/constitutional-charter.md) is the authoritative reference. RFCs that conflict with the Charter must either be rejected or accompanied by a corresponding charter amendment (see [`../../charter/amendments.md`](../../charter/amendments.md)).

## Summary

The cross-subtype-merge framing pair ([`ontology-revision-cross-subtype-merge-typing.md`](./ontology-revision-cross-subtype-merge-typing.md) + [`ontology-revision-cross-subtype-merge-enablement.md`](./ontology-revision-cross-subtype-merge-enablement.md)) decomposed cross-subtype merge into two orthogonal questions: typing (what TYPE the produced record is) and enablement (under what criterion merge is permitted at all). This RFC opens the symmetric questions for **cross-subtype split** in a single consolidated document.

The 1-to-N structural shape of split per [`§0050`](../../charter/decision-log.md) is structurally distinct from merge's 2-to-1 shape, which permits a tighter framing here: split's antecedent is a SINGLE typed subtype; the typing question is about the SUCCESSOR SET, not about the operation's output as a unit. This asymmetry allows the typing and enablement questions to be bundled without the structural friction the merge framing avoided by splitting them.

The RFC presents three typing candidates and three enablement candidates (without picking either). The within-subtype split surface today (`§0050` for BC; `§0061` for AG; `§0068` for CH; `§0075` for CR — closed at [`§0090`](../../charter/decision-log.md) batch) validates that all successors resolve to the antecedent's OWN subtype; lifting that guard for cross-subtype split requires committee-resolved typing + enablement.

## Motivation

[`decision-log §0119`](../../charter/decision-log.md) closed the §0098 auth-scope RFC arc; within-subtype lifecycle surface is at 24/24 across helpers/CLIs/HTTP. The cross-subtype split surface is symmetrically blocked alongside cross-subtype merge: each within-subtype split helper validates that all successor formation hashes resolve to the antecedent's subtype, returning `ErrTargetWrongType` otherwise. Three failure modes appear if cross-subtype split typing + enablement are left deferred:

- **Implementation-resolved permissiveness.** Same as the merge case (ontology-keeper failure mode).
- **Asymmetric criterion across operator surfaces.** Same as the merge case.
- **Asymmetric resolution between cross-subtype merge and split.** Cross-subtype merge has typing + enablement framings at PRs #95 / #97 ([`ontology-revision-cross-subtype-merge-typing.md`](./ontology-revision-cross-subtype-merge-typing.md) / [`ontology-revision-cross-subtype-merge-enablement.md`](./ontology-revision-cross-subtype-merge-enablement.md)) and recommended candidates at PRs #96 / #98 (γ-typing; B+D-enablement, conditional). If split is framed differently or resolves differently, the merge/split lifecycle pair becomes structurally asymmetric — an operator may merge BC + CR cross-subtype but cannot split a BC into {BC, CR} successors, or vice versa. Per [`§0050`](../../charter/decision-log.md), split IS the structural inverse of merge; asymmetry between the two operations is a §2.5 BC1 channel-agnostic-symmetry concern.

## Constitutional Review

Q1–Q6 impact analysis per [`rfc-author` §1](../../../.claude/skills/workflow/rfc-author/SKILL.md). Compressed by reference to the merge framing where applicable.

### Q1 — Which Charter invariants does this RFC touch?

Same surface as [`ontology-revision-cross-subtype-merge-typing.md`](./ontology-revision-cross-subtype-merge-typing.md) + [`ontology-revision-cross-subtype-merge-enablement.md`](./ontology-revision-cross-subtype-merge-enablement.md) Q1 sections: §2.5 (frozen v0.3, touched at the BC1 lifecycle-event surface); §2.3 (frozen v0.4, touched at provenance traversal); §2.4 (frozen v0.5, touched at `subject_ref_hypothesis` payload shape via successor typing); §4 (frozen v0.2, consistency-check). No FROZEN invariant is amended.

### Q2 — Does this RFC implicitly redefine any term in the glossary?

`split`, `merge`, `formation`, `hypothesis` plus the four subtypes are preserved per the merge framing's Q2 analysis. Successor-set terminology inherits from [`§0050`](../../charter/decision-log.md) ascending-sort idempotency. No new canonical term introduced unless typing Candidate α'-split (analogous to merge α — new typed successor) is the resolution.

### Q3 — Does this RFC implicitly resolve any of the five open Ontology questions?

This RFC, like its merge siblings, is a Q2-A.2 follow-on per the entity-model deferral. It does not resolve any of the five canonical OQs.

### Q4 — Does this RFC require Charter amendment?

No. Same reasoning as the merge framing: §2.5 frozen v0.3 BC1 surface accommodates any typing + enablement combination within the existing channel-agnostic lifecycle-event framing.

### Q5 — Does this RFC introduce a new invariant?

No. Codification within existing pending operation (cross-subtype split) under already-frozen invariant surface (§2.5).

### Q6 — Does this RFC propose ceremony without behavioral consequence?

No. Different combinations of typing + enablement produce different runtime behaviors: a {BC → AG + AG} cross-subtype split is permitted under split-typing-α'+split-enablement-A (split-into-new-typed) but rejected under split-typing-γ'+split-enablement-D (only promoted antecedents may split into different subtypes per the table). The candidates partition the runtime space differently.

## Proposal — Typing

Three typing candidates for the successor set. The cross-subtype split's typing question is: **may the successors be of subtypes different from the antecedent's, and if so, how is each successor's typing determined?**

### Typing α' — Successors are a new typed subtype

Mirrors merge Candidate α (`CompositeHypothesis`). Each cross-subtype split produces N successors, all of a new typed subtype (e.g., `SplitFragment` — working name) carrying: (a) the antecedent's formation-hash; (b) a `target_subtype` discriminator naming which of the four subtypes the fragment represents; (c) the fragment-specific surface (membership-subset, signature, etc.).

**Dependency.** None on pending invariants beyond Q2-A.2 itself.

**Pros.** Type-system-explicit: a reader of a `SplitFragment` knows from its type that it is a cross-subtype-split output. No type explosion (split-of-split produces more `SplitFragment` records of the same type).

**Cons.** Information loss at the surface level (the fragment's `target_subtype` discriminator carries what α encoded in `combined_subtypes`; downstream consumers must inspect the discriminator). Asymmetric with merge α: merge α's `CompositeHypothesis` is multi-typed in antecedents and unitary in output; split α' is unitary in antecedent and multi-typed in successors via discriminator — the two framings don't compose cleanly (a `CompositeHypothesis` cross-subtype-split produces `SplitFragment` successors with `target_subtype` referencing the `combined_subtypes` source pairs?).

### Typing β' — Successors are abstract `MergedHypothesis` records

Mirrors merge Candidate β. Each successor is an abstract Hypothesis record with subtype-elision; the discriminator-list field records the original subtype + the fragment-specific structure flattens.

**Dependency.** Requires materializing the abstract Hypothesis type, same as merge β.

**Pros.** Symmetric with merge β. Merge-of-split and split-of-merge compose naturally under the abstract type.

**Cons.** Same Charter §4 criterion 1 structural-enforceability compromise as merge β per [`cross-subtype-merge-typing-evidence.md` Finding 1](../discussion/cross-subtype-merge-typing-evidence.md) — subtype-specific structural constraints reduce to runtime / projection-query checks under β.

### Typing γ' — Each successor's type is determined by a per-target-subtype-table

Mirrors merge Candidate γ. The antecedent of subtype X may split into successors of subtypes drawn from a pre-declared per-X "permitted target set". E.g., a BC antecedent may split into {BC, BC} (within-subtype, default), {BC, AG}, {BC, CR}, or {BC, CH}; an AG antecedent may split into {AG, AG}, {AG, CR}, etc. The 6 (or 12) pair-cells map source-subtype → set-of-permitted-target-subtypes.

**Dependency.** None on pending invariants. Co-located naturally with the merge γ table if both resolve to per-pair canonicals.

**Pros.** No new types; existing four subtypes' protos accommodate split successors via heterogeneous successor-list. Composes with merge γ symmetrically (the two operations share a single pair-table per [§Combined-table option below]).

**Cons.** Per-pair committee defense; same burden as merge γ.

### Combined-table option for typing γ' + merge γ

A single per-pair entity-model table carries BOTH merge γ's canonical-merge-target AND split γ's permitted-split-targets. The merge cell answers "{X, Y} merge → Z?"; the split cell answers "X split → {Y, ...}?". Co-location preserves the §0050 structural-inverse symmetry: merge and split share the same structural commitments by construction.

## Proposal — Enablement

Three enablement candidates for the operator's right to initiate cross-subtype split. Inherits from the merge enablement framing per [`ontology-revision-cross-subtype-merge-enablement.md`](./ontology-revision-cross-subtype-merge-enablement.md); candidate letters retained for ease of cross-reference.

### Enablement A' — Operator-discretionary

Mirrors merge A. No structural gate beyond the within-subtype same-subtype guard being lifted.

**Pros / cons.** Same as merge A per [`cross-subtype-merge-enablement-evidence.md` Finding 1 + Finding 4](../discussion/cross-subtype-merge-enablement-evidence.md). The merge evidence disqualified A on §4 discipline grounds; the same disqualification applies symmetrically to split A'.

### Enablement B' — Successor-membership-partition criterion

Cross-subtype split is permitted only when the successors' combined membership is a partition of the antecedent's membership (modulo subtype-specific reinterpretation). For BC antecedent splitting into {BC, AG}: the BC fragment's members ∪ AG fragment's members = the antecedent's members (no actor created or lost in the split). For CH antecedent: the events partition; for CR: the participant set partitions.

**Dependency.** Requires that each subtype's formation record exposes its membership in a uniformly-partitionable shape. The four existing subtypes each have this surface; the criterion operationalizes against the union.

**Pros.** Structural defense against frivolous cross-subtype split. Symmetric with merge B's shared-actor-membership criterion (merge requires shared-actors; split requires actor-partition). The two criteria together form a closed structural shape: merge unions antecedent memberships into one; split partitions one membership across successors.

**Cons.** Rejects splits where the membership partition is structurally valid but semantically asymmetric (e.g., an AG splitting into {AG, BC} where the AG fragment is a strict actor-subset of the antecedent but the BC fragment is a different membership extracted from a different source). Forces a single notion of "partition"; future identity-tier work may complicate (members that are distinct actor_refs but resolve to the same identity).

### Enablement D' — Lifecycle-state criterion

Mirrors merge D. Cross-subtype split is permitted only when the antecedent is in promoted state per §0011 Q4-Layer-A.

**Pros / cons.** Same as merge D per [`cross-subtype-merge-enablement-evidence.md` Finding 3](../discussion/cross-subtype-merge-enablement-evidence.md). D' couples to Q4-Layer-A symmetrically with merge D.

### Combined enablement form

The merge evidence recommended B+D (shared-actor-membership AND lifecycle-state). The symmetric recommendation for split is **B'+D'** (membership-partition AND lifecycle-state). The committee may diverge on the symmetric form if split-specific evidence warrants.

## Symmetric resolution preference

The discussion phase explicitly raises a meta-question: **should cross-subtype merge and cross-subtype split resolve SYMMETRICALLY?** Per [`§0050`](../../charter/decision-log.md), split IS the structural inverse of merge; asymmetric resolution creates a lifecycle-pair gap. Two positions:

- **Symmetric resolution.** Whatever the merge framing resolves to, split inherits the symmetric form. Concretely: if merge resolves to γ-typing + B+D-enablement (the merge evidence's recommendation), split resolves to γ'-typing + B'+D'-enablement. The shared per-pair table from the [§Combined-table option above](#combined-table-option-for-typing-γ--merge-γ) is the structural artifact.
- **Independent resolution.** Merge and split are evaluated separately; either may resolve to any combination. Concrete asymmetry permitted (e.g., merge γ + split β').

The RFC does not pre-commit. The symmetric-resolution argument is structural (§0050 inverse-symmetry); the independent-resolution argument is operator-discretion. The discussion phase considers both.

## Alternatives Considered

### Forbid cross-subtype split entirely (REJECTED)

Same reasoning as merge: the entity-model frozen prose permits the operation structurally. Forbidding requires entity-model amendment, not this RFC.

### Asymmetric resolution between merge and split (META-PATTERN)

Surfaced for procedural transparency. The RFC does not pre-commit to symmetry; the committee may resolve independently.

## Open Questions

This RFC explicitly defers:

- **Successor cardinality.** Today's within-subtype split helpers accept N ≥ 2 successors per [`§0050`](../../charter/decision-log.md). Cross-subtype split inherits the N ≥ 2 contract under all six candidate combinations. Whether some candidates warrant N=2-only (binary split) is open.
- **Split-of-merge / merge-of-split composition.** A cross-subtype-merge output may be subsequently split; a cross-subtype-split fragment may be subsequently merged with another. The two lifecycle operations compose; whether the composition is bounded or unrestricted is open.
- **Multi-stage split** (1 → N₁ → N₂). The within-subtype split today produces N successors at once; multi-stage split (split a successor of an earlier split) requires no special semantics. Cross-subtype multi-stage split inherits this — but interactions with typing γ' + enablement B' may produce surprising trajectories (a {BC → BC, CR} split's CR fragment may further split into {CR, CH} under γ'). Recorded.
- **Rejection-record question** (same as merge).

## Anti-Patterns to Avoid

- **Resolving the criterion by implementation.** Hard-coded typing or enablement rules in a `SplitAcrossSubtypes` helper without RFC.
- **Asymmetric criterion across operator surfaces.** Same as merge.
- **Treating cross-subtype split as reclassification.** Per [`lifecycle-semantics.md`](../../ontology/lifecycle-semantics.md), reclassification is a Cat II operation; cross-subtype split is a Cat III lifecycle event.
- **Forcing symmetric resolution by default.** The symmetric-resolution argument is structurally compelling but is itself a committee choice (per §Symmetric resolution preference above). Pre-committing to symmetry before committee evaluation IS the [`ontology-keeper`](../../../.claude/skills/ontology/ontology-keeper/SKILL.md) failure mode this RFC exists to prevent.

## Migration and Backward Compatibility

No historical cross-subtype split records exist. Lock-in asymmetry mirrors merge:

- α' / γ' / D' are easiest to revise later.
- β' carries the heaviest substrate-immutability conflict if it needs to be revised.
- B' couples to identity-tier evolution (Q2 follow-on).

[Phase 3 / Phase 4 replay contracts](../../architecture/replay-model.md) preserved per the same arguments as merge.

## References

- [`docs/ontology/entity-model.md` §Cross-subtype operations](../../ontology/entity-model.md) — canonical deferral.
- [`docs/ontology/lifecycle-semantics.md` §Split](../../ontology/lifecycle-semantics.md) — current scaffold.
- [`docs/rfcs/draft/ontology-revision-cross-subtype-merge-typing.md`](./ontology-revision-cross-subtype-merge-typing.md) — merge typing framing; sibling.
- [`docs/rfcs/draft/ontology-revision-cross-subtype-merge-enablement.md`](./ontology-revision-cross-subtype-merge-enablement.md) — merge enablement framing; sibling.
- [`docs/rfcs/discussion/cross-subtype-merge-typing-evidence.md`](../discussion/cross-subtype-merge-typing-evidence.md) — merge typing evidence; recommends γ.
- [`docs/rfcs/discussion/cross-subtype-merge-enablement-evidence.md`](../discussion/cross-subtype-merge-enablement-evidence.md) — merge enablement evidence; recommends B+D.
- [`docs/charter/constitutional-charter.md` §2.5](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) — frozen v0.3; channel-agnostic lifecycle surface.
- [`docs/charter/decision-log.md` §0050](../../charter/decision-log.md) — split is structural inverse of merge; basis for symmetric-resolution argument.
- [`docs/charter/decision-log.md` §0011](../../charter/decision-log.md) — Q4 Layer A; enablement D' operationalizes against.

## Decision Record

Resolved at [`decision-log §0124`](../../charter/decision-log.md): **symmetric resolution with the cross-subtype merge framing** ([`§0122`](../../charter/decision-log.md) merge typing γ + [`§0123`](../../charter/decision-log.md) merge enablement B+D). Per the §Symmetric resolution preference framing above, option (b) — symmetric inheritance — adopted.

Concretely:

- **Typing: Candidate γ' — per-source-subtype permitted-target-set table.** Cross-subtype split produces successors of one of the four existing concrete subtypes; no new typed subtype introduced. Form adopted; specific per-source permitted-target contents deferred to the [`ontology-revision-cross-subtype-merge-pair-table.md`](./ontology-revision-cross-subtype-merge-pair-table.md) placeholder, which is extended per §Combined-table option above to carry BOTH merge γ's canonical-merge-target AND split γ' permitted-split-targets.

- **Enablement: Candidate B'+D' combined form.** Cross-subtype split is permitted when both: (a) **B' (successor-membership-partition):** successors' combined membership partitions the antecedent's membership (union equals antecedent's set; disjoint successor sets); (b) **D' (lifecycle-state gate):** the antecedent is in promoted lifecycle-state per §2.5 + [`§0011`](../../charter/decision-log.md) Q4 Layer A.

The §0050 inverse-symmetry with merge is preserved by construction: merge requires shared-actor intersection (non-empty); split requires actor partition (union equals antecedent + disjoint successors). The two enablement gates are duals under set-theoretic operations.

### Reversal conditions

Per [`cross-subtype-split-evidence.md` §What would warrant asymmetric resolution](../discussion/cross-subtype-split-evidence.md), asymmetric resolution is justified if any of the following emerges:

- **Split-specific operator workload requiring α' for type-level discrimination of split outputs.** If type-discrimination by "is this a split fragment?" becomes load-bearing, α' becomes preferable for split independent of merge's γ.
- **Split-specific enablement requiring D'-alone (no partition gate).** If partition-strictness rejects too many epistemically-valid splits, B'-without-D' or D'-alone is the looser alternative.
- **Symmetric-resolution reversal at the merge framing.** If [`§0122`](../../charter/decision-log.md) or [`§0123`](../../charter/decision-log.md) flips, the symmetric-resolution adoption here may flip in tandem.

No reversal condition fires at acceptance.
