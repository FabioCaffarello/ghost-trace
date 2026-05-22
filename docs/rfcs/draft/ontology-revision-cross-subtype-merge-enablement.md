# RFC — Cross-subtype merge enablement criterion

- **Status:** discussion
- **Authors:** Ghost Trace committee
- **Date:** 2026-05-21
- **Type:** ontology-revision
- **Affects:** [`docs/ontology/entity-model.md`](../../ontology/entity-model.md) §Cross-subtype operations; [`docs/ontology/lifecycle-semantics.md`](../../ontology/lifecycle-semantics.md) §Merge; [Charter §2.5](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) (frozen v0.3 — minor amendment surface only; this RFC produces no §2.5 prose change unless the resolution requires it).

> RFCs in Ghost Trace are proposals subject to constitutional review. The [Constitutional Charter](../../charter/constitutional-charter.md) is the authoritative reference. RFCs that conflict with the Charter must either be rejected or accompanied by a corresponding charter amendment (see [`../../charter/amendments.md`](../../charter/amendments.md)).

## Summary

[`ontology-revision-cross-subtype-merge-typing.md`](./ontology-revision-cross-subtype-merge-typing.md) opens the question of what TYPE a cross-subtype merge produces. This RFC opens the orthogonal question of **when cross-subtype merge is permitted at all**: under what structural criterion does an operator gain the right to initiate the merge? Today the within-subtype merge surface validates that both antecedent formation hashes resolve to the SAME subtype (via `ErrTargetWrongType`); lifting that within-subtype guard for cross-subtype merge requires committee-resolved enablement criterion — without it, the helper would either silently permit any cross-subtype pair (resolving by implementation) or remain blocked.

The RFC presents four candidate enablement criteria and one explicitly rejected alternative. The RFC does not pick. The enablement question is orthogonal to (and resolvable independently of) the typing question per [`ontology-revision-cross-subtype-merge-typing.md` §Open Questions](./ontology-revision-cross-subtype-merge-typing.md): the typing RFC concerns the produced record's typing under any chosen enablement criterion; this RFC concerns the criterion itself under any chosen typing.

## Motivation

[`decision-log §0119`](../../charter/decision-log.md) closed the §0098 auth-scope RFC arc; within-subtype lifecycle surface is at 24/24 across helpers/CLIs/HTTP. The cross-subtype merge surface has been blocked at the helper layer since §0049 / §0060 / §0067 / §0074 introduced the per-subtype merge helpers — each helper validates that both antecedent formation hashes resolve to its OWN subtype. Two structural failure modes appear if the enablement criterion is left deferred:

- **Implementation-resolved permissiveness.** A `MergeAcrossSubtypes` helper lifts the within-subtype guard without RFC review. The choice — whether all pairs are permitted, only specific pairs, or only pairs satisfying a runtime predicate — becomes infrastructure-resolved. Falls under the [`ontology-keeper`](../../../.claude/skills/ontology/ontology-keeper/SKILL.md) failure mode.
- **Asymmetric criterion across operator surfaces.** The CLI accepts any cross-subtype pair; HTTP T4 requires shared actor membership; the helper enforces a third rule. Operators see different validation gates on different surfaces for the same operation. The §2.5 frozen v0.3 BC1 contract (lifecycle events are channel-agnostic Cat I primary observations) requires uniform criterion regardless of channel.

The cost of not resolving the criterion before further implementation is therefore concrete: cross-subtype merge either lands under silent permissiveness OR remains operationally unreachable despite the typing question's resolution.

## Constitutional Review

The Q1–Q6 impact analysis prescribed by [`rfc-author` §1](../../../.claude/skills/workflow/rfc-author/SKILL.md).

### Q1 — Which Charter invariants does this RFC touch?

- **§2.5 Hypothesis Lifecycle Explicitness** (frozen v0.3): touched. Merge is one of the six §2.5 lifecycle operations; the enablement criterion is a structural rule on when the operation is permitted. §2.5 BC1–BC6 are channel-agnostic; the enablement criterion lives within the existing BC surface as a precondition gate. None of the candidates below amend any BC — they operationalize within the existing surface.
- **§2.3 Provenance Integrity** (frozen v0.4): touched indirectly. The enablement criterion may itself reference provenance — e.g., shared-actor-membership (Candidate B) requires inspecting the antecedents' actor-set fields, which IS provenance traversal per the §2.3 BC5 multi-category-traversal shape. All four candidates preserve §2.3's chain shape; they differ in whether they require provenance inspection at the enablement gate.
- **§4 Constitutional Design Rule** (frozen v0.2): consistency check, not amendment. The four falsifiability criteria are applied to each candidate; specific candidate evaluations are deferred to the discussion phase.
- No FROZEN invariant is amended.

### Q2 — Does this RFC implicitly redefine any term in the glossary?

The terms `hypothesis`, `merge`, `formation`, `actor_ref`, `behavioral cluster`, `automation group`, `campaign hypothesis`, `coordination ring` are in [`docs/glossary.md`](../../glossary.md) and [`.claude/CLAUDE.md` §3 canonical vocabulary](../../../.claude/CLAUDE.md). All four candidates preserve these definitions.

No candidate introduces a new canonical term. Candidate D (lifecycle-state criterion) references the §2.5 promotion / demotion / dissolution lifecycle terms but operationalizes the existing definitions; it does not redefine them.

### Q3 — Does this RFC implicitly resolve any of the five open Ontology questions?

The RFC is a sibling of [`ontology-revision-cross-subtype-merge-typing.md`](./ontology-revision-cross-subtype-merge-typing.md); both are Q2-A.2 follow-ons. Neither resolves any of the five canonical OQs from [`ontology.md`](../../ontology/ontology.md).

Candidate B (shared-actor-membership) interacts with [`entity-model.md` Open Modeling Question 2 (identity tiers)](../../ontology/entity-model.md): "shared actor_ref" presupposes the actor_ref tier is structurally addressable, which it is under the §0023 inception-phase single-tier resolution. Future identity-tier expansion (per Q2 follow-on) may extend Candidate B's operationalization to multi-tier matching; the dependency is recorded but does not block the present RFC.

### Q4 — Does this RFC require Charter amendment?

No. §2.5 is frozen v0.3 with a channel-agnostic BC surface that accommodates any enablement-criterion precondition gate. The cross-subtype merge resolution operationalizes within the existing §2.5 surface; if the resolution adds prose to §2.5 it would be a minor amendment (v0.3 → v0.3.1 or v0.6), but the discussion-phase RFC anticipates no prose change.

### Q5 — Does this RFC introduce a new invariant?

No. The RFC codifies a structural rule for an existing pending operation (cross-subtype merge enablement) within an already-frozen invariant surface (§2.5). No new constitutional claim is added.

### Q6 — Does this RFC propose ceremony without behavioral consequence?

No. Different criteria produce different runtime behaviors: a {BC, AG} pair merging actor-sets sharing no actor_refs is permitted under Candidate A, rejected under Candidate B, permitted-or-rejected per the table under Candidate C, and gated by lifecycle-state under Candidate D. The candidates partition the runtime space of merge-enablement decisions differently. Deleting this RFC would either force implementation-resolved permissiveness or produce an operationally-unreachable cross-subtype surface.

## Proposal

Four candidate criteria, each presented with structural claim, dependency on other frozen / pending invariants, and pros and cons. The RFC does not pick. Combined criteria — applying any pair as gated stages — are an explicit option for the discussion phase.

### Candidate A — Operator-discretionary (no structural gate)

**Structural claim.** Cross-subtype merge is permitted whenever an operator initiates it; no structural precondition is enforced. The helper accepts any two formation hashes resolving to ANY of the four subtypes (or any cross-subtype pair under the typing RFC's α/β/γ resolution). Mirrors the within-subtype merge contract today (the within-subtype helper accepts ANY two same-subtype antecedents without further structural check).

**Dependency.** None on pending invariants. The §0033 local-shell-trust assumption applies to CLI invocations; HTTP T4 invocations are gated by the §0098 per-actor authentication only.

**Pros.**

- Simplest to operationalize. The helper's existing same-subtype guard is replaced by NO guard; no new structural check.
- Symmetric with within-subtype merge. The within-subtype helper today does not check whether the two antecedents are semantically compatible; cross-subtype merge under A inherits the same operator-trust contract.
- Falsifiability is direct: an operator initiates a structurally-permitted merge; the helper commits the merge event without further check.

**Cons.**

- No structural defense against frivolous cross-subtype merge. An operator may merge an unrelated {BC, CH} pair (no shared actors, no temporal overlap, no semantic relationship); the substrate records the merge as a lifecycle event regardless.
- Asymmetry with the typing question: γ's pair-table approach (per the typing RFC) pre-commits the produced type per pair; A leaves the merge act itself ungated. The two RFCs' resolutions may interact in unexpected ways under A (any pair → table lookup → typed output; but no defense at the merge act itself).
- Operationally, "the operator can do anything" is the §0033 contract — but §0033 applies to the local-shell-trust CLI surface, not to HTTP T4. Under A, HTTP T4 cross-subtype merge is permitted regardless of cross-tier-authentication defense beyond the existing per-actor attribution.

### Candidate B — Shared-actor-membership criterion

**Structural claim.** Cross-subtype merge is permitted only when the antecedents share at least one `actor_ref` in their respective membership / participant / actor fields. For subtypes whose membership is event-centric (CampaignHypothesis), "membership" is interpreted as the union of actor_refs across the events. The helper inspects both antecedents' resolved formation records and rejects (returns `ErrEnablementUnsatisfied` or similar) when the intersection is empty.

**Dependency.** Mild. Requires that each subtype's formation record exposes its actor-set in a uniformly-traversable shape. The four existing subtypes (BC actor-set; AG actor-set; CR pairwise-relationship actor-set; CH event-actor-union) each have an actor-set surface; the criterion operationalizes against the union of all four. Future identity-tier expansion (per Q2 follow-on) may broaden "shared actor_ref" to "shared identity-tier-N reference"; the dependency is recorded but does not block.

**Pros.**

- Structural defense against frivolous cross-subtype merge. An operator may not merge an {BC, CH} pair whose actor-sets are disjoint; the substrate-grounded check rejects the operation.
- Falsifiability is direct: the intersection of antecedent actor-sets is computable on the substrate without operator interpretation.
- Aligns with the entity-model claim that cross-subtype merge represents "recognition of the same underlying phenomenon" ([entity-model.md §Cross-subtype operations](../../ontology/entity-model.md)): the shared-actor-set IS structural evidence of a common underlying entity.

**Cons.**

- Rejects merges that may be epistemically valid but actor-disjoint. Two `CampaignHypothesis` records representing distinct campaign-stages by different operators may be the same underlying campaign even with no shared actor_ref; B rejects.
- Forces a single notion of "shared". Tier-1 actor_ref equality is the inception-phase definition; future identity-tier work may complicate (when actor_ref A and actor_ref B resolve to the same identity but were observed as distinct refs).
- For CoordinationRing pairs, "shared actor_ref" may be present but on a non-load-bearing axis (a single shared participant in a 50-actor ring is structurally weak evidence of common phenomenon).

### Candidate C — Pre-declared per-pair enablement table

**Structural claim.** For each of the C(4,2) = 6 unordered pairs (or 4×3 = 12 ordered pairs) of cross-subtype combinations, pre-declare at the entity-model level whether merge is enabled and under what additional criterion. Operationally a table:

| Pair | Enabled? | Additional criterion |
|---|---|---|
| {BC, AG} | yes | none — both actor-set hypotheses |
| {BC, CR} | conditional | shared-actor-membership ≥ 1 |
| {BC, CH} | conditional | shared-actor-membership ≥ 1 |
| {AG, CR} | yes | none |
| {AG, CH} | conditional | shared-actor-membership ≥ 1 |
| {CR, CH} | conditional | shared-actor-membership ≥ 1 |

(Illustrative — the actual table content is the committee's deliberation.)

**Dependency.** None on pending invariants. The table is an entity-model-level structural rule. Naturally pairs with the typing RFC's γ candidate (per-pair canonical-target typing) — the two tables may be co-located, with each cell carrying both "enabled?" and "produced typing".

**Pros.**

- Per-pair structural deliberation. The committee defends each cell explicitly; the table IS the deliberation surface.
- Composes naturally with the typing RFC's γ candidate. A single per-pair table carries both questions; operator surface is consistent.
- Avoids B's rigid single-criterion limitation: pairs with weak structural similarity may require shared-actor-membership; pairs with strong similarity (BC and AG are both actor-set hypotheses by structural surface) may not.

**Cons.**

- Committee defense for each of 6 (or 12) cells. Same per-cell-defense burden as the typing RFC's γ — but here doubled (each cell carries both enablement and typing under γ+C combined).
- The table couples to the four subtypes; subtype churn (per Q2 follow-on) requires table revision.
- Operator surface gains complexity. CLI / HTTP / helper paths each need to read the table at runtime; mis-implementation may produce asymmetric criterion across surfaces (the precise §2.5 BC1 channel-agnosticism risk the motivation cited).

### Candidate D — Lifecycle-state criterion

**Structural claim.** Cross-subtype merge is permitted only when both antecedents are in lifecycle-state `promoted` (or above some configurable threshold per Q4-resolution's Layer A time-based-cadence gate, [`§0011`](../../charter/decision-log.md)). Within-subtype merge today accepts antecedents in any lifecycle-state; D imposes a higher bar for cross-subtype on the rationale that cross-subtype recognition warrants higher confidence than within-subtype recognition.

**Dependency.** [`§0011`](../../charter/decision-log.md) Q4 resolution (promotion-demotion criterion). The Layer A cadence gate is already structurally available; the Layer B deep criterion is on hold per [`ontology-revision-layer-b-deep-criterion.md`](./ontology-revision-layer-b-deep-criterion.md). D operationalizes against Layer A today; Layer B may sharpen later.

**Pros.**

- Defends against premature cross-subtype merge of unevolved hypotheses. A newly-formed hypothesis (lifecycle-state `formed`, not yet promoted) is not a candidate for cross-subtype merge under D; only promoted hypotheses qualify.
- Substrate-grounded: lifecycle-state is structurally recorded per §2.5; the gate is computable on the substrate.
- Composes orthogonally with A, B, or C — D may be applied as an additional gate on top of any of those (e.g., A+D: any pair permitted but only when both are promoted; B+D: shared-actor-membership AND both promoted).

**Cons.**

- Restrictive in scenarios where cross-subtype merge of `formed`-state antecedents is structurally defensible (e.g., two newly-formed but actor-overlapping hypotheses recognized at formation as the same phenomenon).
- The lifecycle-state surface is per-subtype; D's gate must inspect both antecedents' subtype-specific state representations. Operationally similar to B's actor-set inspection.
- Couples enablement to Q4's resolution. If Q4's Layer B sharpens the demotion criterion later, D's enablement gate inherits the sharpening — the criterion becomes a moving target unless the RFC pins it to a specific Q4-Layer-A form.

### Combined-candidate forms

The discussion phase considers combinations:

- **A + D** — Operator-discretionary among the four pairs, but only when both antecedents are promoted. Lightweight gate; protects against unevolved-hypothesis merge.
- **B + D** — Shared-actor-membership AND both promoted. Strongest structural defense; rejects both frivolous-pair AND unevolved-state merges.
- **C + D** — Per-pair table + promoted-state gate. Maximum committee surface but maximum structural deliberation.
- **B + C** — Shared-actor-membership as fallback for table cells marked "conditional"; explicit "enabled" cells skip the actor-check. Reduces B's rigidity by allowing per-pair exemption.

The RFC does not recommend any combination form. The discussion phase considers them on equal footing with the four single-criterion forms.

## Alternatives Considered

### Forbid cross-subtype merge entirely (REJECTED)

Cross-subtype merge is structurally forbidden at the entity-model level; operators recognizing the same underlying phenomenon across subtypes record only an informational Assertion. Rejected because it conflicts with [`entity-model.md` §Cross-subtype operations](../../ontology/entity-model.md) frozen prose: *"Cross-subtype merge ... is structurally permitted."* Closing the surface here requires amending entity-model.md, not this RFC.

### Operator-specified criterion per invocation (META-ANTI-PATTERN)

The cross-subtype merge helper takes an `enablement_criterion` parameter; the operator picks per-invocation. Rejected on the same grounds as the typing RFC's per-invocation-typing alternative: violates [`§4`](../../charter/constitutional-charter.md#4-constitutional-design-rule) criterion 4 (independence of operator interpretation). Two operators initiating cross-subtype merge on the same antecedents under different self-declared criteria would produce structurally-different validation outcomes. Not a candidate; surfaced for procedural transparency.

## Open Questions

This RFC explicitly defers the following:

- **Symmetric enablement for cross-subtype split.** The cross-subtype split surface has a sibling enablement question (when may an operator initiate cross-subtype split?). Opened at [`ontology-revision-cross-subtype-split.md`](./ontology-revision-cross-subtype-split.md) (consolidated typing + enablement; Status: discussion). Per §0050 split-as-inverse-of-merge, the §Symmetric resolution preference of the split RFC raises whether the merge enablement and split enablement should resolve symmetrically (B+D for merge → B'+D' for split per the [`cross-subtype-split-evidence.md`](../discussion/cross-subtype-split-evidence.md) recommendation).
- **Asymmetric enablement on ordered pairs.** Under C, the 6-unordered vs 12-ordered choice has structural consequences: is {BC, AG} merge the same as {AG, BC} merge? Under within-subtype merge today, merge is symmetric (the two antecedents form a SET per [`§0049`](../../charter/decision-log.md) Option B). Whether cross-subtype merge inherits the symmetric-antecedent contract is itself a sub-question of the typing RFC + this RFC interaction.
- **Multi-antecedent merge.** Today's merge helpers accept exactly two antecedents per [`§0049`](../../charter/decision-log.md). Whether cross-subtype merge may accept three or more antecedents (a {BC, AG, CR} 3-way merge) is open; the helper's two-antecedent shape is preserved across all four candidates here.
- **Interaction with the typing RFC.** Some combinations (C+γ — per-pair enablement table co-located with per-pair typing table) are natural; others (A+α — operator-discretionary enablement producing `CompositeHypothesis`) are less so. Whether the two RFCs should be co-resolved or independently is open.
- **Reversibility / supersession.** A rejected cross-subtype merge (under B, C, D) leaves no substrate trace. Whether the rejection itself should be recorded as a Cat I observation (analogous to [`OrphanCleanupAudit` per `§0104`](../../charter/decision-log.md)) for forensic record is open. The within-subtype merge does NOT currently record rejected attempts; whether cross-subtype merge SHOULD differ is a separate question.

## Anti-Patterns to Avoid

- **Resolving the criterion by implementation.** Hard-coded enablement rules in a `MergeAcrossSubtypes` helper without RFC. The criterion belongs to the Ontology and to entity-model.md, not to operational code.
- **Asymmetric criterion across operator surfaces.** The CLI, HTTP T4, and helper paths each enforce their own criterion. The §2.5 BC1 frozen surface requires lifecycle events to be channel-agnostic Cat I primary observations; channel-dependent enablement violates this.
- **Conflating enablement with typing.** The two RFCs (this one + [`ontology-revision-cross-subtype-merge-typing.md`](./ontology-revision-cross-subtype-merge-typing.md)) are orthogonal. The enablement criterion gates WHEN merge is permitted; the typing criterion specifies WHAT TYPE the produced record is. Resolution of one does not bind the other.
- **Treating rejected merges as missing observations.** A merge rejected under B/C/D is a non-event — no Cat I record is committed (unless the Open Questions §4 path is chosen). Recording a non-event AS a record would violate §2.1 substrate-immutability discipline (the substrate records what happened, not what failed to happen).

## Migration and Backward Compatibility

No historical cross-subtype merge records exist. The RFC is forward-looking.

Lock-in asymmetry across candidates:

- Candidate A (operator-discretionary) is the easiest to tighten later: adding any of B/C/D as a gate on top of A is a typed extension; prior cross-subtype merges under A remain valid lifecycle events (they were permitted at the time of commit).
- Candidate B (shared-actor-membership) is moderate: relaxing to A means lifting the gate (prior B-passing merges remain valid); strengthening to D adds an orthogonal gate (prior B-passing merges may have been pre-promotion and need not be retroactively invalidated per §2.1).
- Candidate C (per-pair table) is the hardest to revise: changing a cell from "enabled" to "disabled" creates an asymmetry — prior merges under the cell remain valid (per §2.1) but new merges of the same pair are rejected. The table revision IS a committee action that needs supersession-style discipline.
- Candidate D (lifecycle-state) is moderate: D's gate is computable per-merge from substrate state at commit time; relaxing or strengthening D does not retroactively invalidate prior merges.

The [Phase 3 / Phase 4 replay contracts](../../architecture/replay-model.md) require that any lifecycle event, including cross-subtype merge, is replayable. All four candidates satisfy this: the merge event records its antecedents + the produced record; the enablement criterion is computable from the substrate state at the merge event's commit time. Whether to record the enablement criterion's specific values (e.g., the shared-actor-ref under B; the lifecycle-state under D) in the merge event itself for replay-determinism is a follow-on operational-construct question.

## References

- [`docs/ontology/entity-model.md` §Cross-subtype operations](../../ontology/entity-model.md) — canonical statement of the cross-subtype merge deferral.
- [`docs/ontology/lifecycle-semantics.md` §Merge](../../ontology/lifecycle-semantics.md) — current scaffold; cross-subtype Q2-A.2 deferral.
- [`docs/rfcs/draft/ontology-revision-cross-subtype-merge-typing.md`](./ontology-revision-cross-subtype-merge-typing.md) — sibling RFC; orthogonal to this one. Both are Q2-A.2 follow-ons.
- [`docs/rfcs/discussion/cross-subtype-merge-typing-evidence.md`](../discussion/cross-subtype-merge-typing-evidence.md) — phased evidence for the typing RFC. Pattern reusable for this RFC's discussion phase.
- [`docs/charter/constitutional-charter.md` §2.5](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) — frozen v0.3; channel-agnostic lifecycle surface that accommodates any enablement criterion as a precondition gate.
- [`docs/charter/constitutional-charter.md` §2.3](../../charter/constitutional-charter.md#23-provenance-integrity) — frozen v0.4; provenance traversal shape touched by Candidate B's actor-set inspection.
- [`docs/charter/decision-log.md` §0011](../../charter/decision-log.md) — Q4 resolution (staged Layer A + Layer B). Candidate D operationalizes against Layer A.
- [`docs/charter/decision-log.md` §0023](../../charter/decision-log.md) — Q2 (identity tiers) inception-phase single-tier resolution; Candidate B depends on this for actor_ref equality semantics.
- [`docs/charter/decision-log.md` §0049 + §0060 + §0067 + §0074 + §0119](../../charter/decision-log.md) — per-subtype merge helpers + their within-subtype guard; §0119 within-subtype 24/24 closure.
- [`.claude/skills/ontology/ontology-keeper/SKILL.md`](../../../.claude/skills/ontology/ontology-keeper/SKILL.md) — registry of open Ontology questions + Q2-follow-on framing precedent.

## Decision Record

Pending. The discussion phase opens with this RFC at `Status: discussion`; resolution will be recorded in `docs/charter/decision-log.md` and reflected back here on acceptance.
