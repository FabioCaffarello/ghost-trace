# Empirical-Pressure Assessment — §2.6 (post-§2.4 refresh)

**Status:** assessment-only. Non-binding. Does not draft §2.6 prose, does not pick answers to catalogued questions, does not itself trigger redaction resumption. The decision whether to resume committee redaction is reserved.

This document refreshes the §2.6 portion of [`empirical-pressure-2-4-2-6.md`](./empirical-pressure-2-4-2-6.md) (recorded at [`decision-log §0077`](./decision-log.md)) in the post-§2.4-frozen context. §2.4 closed at frozen v0.5 per [`decision-log §0099`](./decision-log.md); the §2.6 questions catalogued at §0077 are reassessed here against §2.4's binding text and against the implementation work that has accumulated since (§0098 auth-scope arc closure; Q2-A.2 cross-subtype merge+split framing arc).

## 1. Status snapshot

| Section | Status at §0077 | Status now |
|---|---|---|
| §2.4 Inferential Influence Disclosure | pending — empirical pressure phase | **frozen v0.5** per [`§0099`](./decision-log.md) |
| §2.6 Evidential Independence Integrity | pending — empirical pressure phase | pending — empirical pressure phase |
| §3 Non-Goals | pending committee redaction | pending committee redaction |

The §0077 assessment ranked §2.6 pressure as "moderate, downstream of §2.4 per [`§0008`](./decision-log.md) order". With §2.4 frozen, the downstream-of-§2.4 condition is discharged.

## 2. Implementation surface since §0077

Material change since §0077 (recorded 2026-05-21 alongside §2.4 closure):

- **§0098 auth-scope RFC arc closed** (PRs #74–#94 + §0117/§0118/§0119). Within-subtype lifecycle at full 24/24 surface across helpers, CLIs, HTTP T4. Per-actor attribution end-to-end. No new Cat III field added — `confidence` remains placeholder; no `evidential_independence` field exists in any proto.
- **Cross-subtype merge+split framing arc opened at discussion phase** (PRs #95–#100). Six framing documents covering typing + enablement for merge and split. Recommendations recorded conditional; resolution committee-pending. **Not yet implemented**; the cross-subtype helpers remain blocked at `ErrTargetWrongType`. The §2.6 questions catalogued at §0077 inherit the same within-subtype assumptions they were authored against (single-subtype formations; merge produces same-subtype formation per §0049 Option B).
- **§2.4 frozen v0.5** — substrate-time `influenced_by` edge generation (per [§0021](./decision-log.md) OMQ #3-α); decay via §2.5 lifecycle event supersession (per [§0020](./decision-log.md) OMQ #2-C); Layer B forward-reference activation pending §2.6 redaction.

No new proto field has been added to any Cat III formation between §0077 and now. No `evidential_independence`-bearing wire shape has been committed. The implementation surface load-bearing for §2.6 redaction is structurally unchanged at the proto layer; it has accumulated at the lifecycle-operation surface (HTTP T4 + CLI + projection layer coverage).

## 3. Re-evaluation of Q-§2.6-* questions in the post-§2.4 context

For each question catalogued at [`empirical-pressure-2-4-2-6.md` §4](./empirical-pressure-2-4-2-6.md), the post-§2.4 status:

### Q-§2.6-1 — Shape of the paired `evidential_independence` dimension

**Status:** unchanged. §2.4 binding text does not commit to a shape for the paired independence dimension; the question remains exactly as catalogued. §2.4 BC frozen-v0.5 binding text governs the timing of `influenced_by` edges (substrate-time) and the inferential semantics of typed reference edges, NOT the structural shape of the §2.6-paired dimension.

**Pressure increment:** none from §2.4 closure. The cross-subtype framing arc surfaces a sub-question: under typing α (third concrete subtype `CompositeHypothesis`), the new subtype's proto must carry both `confidence` AND the §2.6-paired field; under typing β (abstract `MergedHypothesis`), the abstract type's proto must carry both. The cross-subtype framing's typing question is upstream of how Q-§2.6-1's wire shape propagates across produced records — but the §2.6-paired field's BASE shape is independent of cross-subtype typing.

### Q-§2.6-2 — Origin of the independence value at first commit

**Status:** unchanged in core; §2.4 closure adds a constraint. §2.4 BC frozen-v0.5 specifies that `influenced_by` edges are generated at substrate-write time per OMQ #3-α. If Q-§2.6-2's resolution is (b) "computed by the substrate from the formation event's `source_event_hashes` set", the computation can naturally co-locate with the substrate-time `influenced_by` derivation — both are substrate-write-time computations. If (a) "computed deterministically by the formation pattern" or (c) "operator-supplied", co-location does not naturally follow.

**Pressure increment:** moderate. §2.4's substrate-time generation creates a natural sibling location for §2.6 substrate-time independence derivation; the structural alignment makes (b) the operationally cheapest choice at implementation. Whether this alignment is the *committee-defensible* choice is itself a redaction question — but the pressure is now stronger to address it explicitly rather than leaving the wire-time question implicit.

### Q-§2.6-3 — Independence under merge

**Status:** changed by the cross-subtype framing arc. The §0077 framing assumed within-subtype merge per §0049 Option B (produced formation is same subtype). The cross-subtype framing arc raises three new sub-questions:

- Under merge typing α (`CompositeHypothesis`), how is the new subtype's independence computed from heterogeneous antecedents (a BC with independence X₁ + a CR with independence X₂)? Are X₁ and X₂ semantically comparable across subtypes?
- Under merge typing β (abstract `MergedHypothesis`), independence is on the abstract; the subtype-specific independence semantics of the antecedents are flattened. Is this flattening structurally defensible?
- Under merge typing γ (per-pair canonical-target), the produced subtype is one of the existing four; the produced record's independence is computable per the existing same-subtype rule, but the antecedents may have used subtype-specific independence-derivation rules — does the produced record carry forward those rules?

**Pressure increment:** material. Q-§2.6-3 was a within-subtype question at §0077; it is now also a cross-subtype question conditioned on the typing resolution. The two RFCs (cross-subtype merge typing + this §2.6 question) compose: §2.6 redaction either pre-commits to a cross-subtype-typing assumption (binds the typing RFC's resolution) or carries forward all three branches (binds nothing, defers operationalization).

### Q-§2.6-4 — Independence under split

**Status:** changed analogously to Q-§2.6-3 via the cross-subtype split framing. The split RFC's typing candidates (α'/β'/γ') interact with §2.6 in the same way as merge typing. Additionally, the split enablement RFC's Candidate B' (membership-partition criterion) introduces a new question: if the antecedent's `source_event_hashes` partition across successors, the per-successor independence is *derivable* from the partition — but the redaction must commit to which derivation rule (per [`empirical-pressure-2-4-2-6.md` Q-§2.6-4](./empirical-pressure-2-4-2-6.md)). Under B', the partition is itself a structural commitment, which constrains the derivation rule's domain.

**Pressure increment:** material. Same structural shape as Q-§2.6-3.

### Q-§2.6-5 — Backward-projection onto already-committed formations

**Status:** unchanged. The §2.4 closure does not affect this question; the corpus formations committed under the §0045-era placeholder confidence remain without an `evidential_independence` field regardless of §2.4's resolution. The three branches (a/b/c) catalogued at §0077 stand.

**Pressure increment:** none from §2.4. The cross-subtype framing arc has not added new corpus formations; the corpus count is unchanged from §0077.

## 4. New questions surfaced since §0077

### Q-§2.6-6 — `influenced_by` edge independence contribution

**Question (new).** §2.4 frozen v0.5 commits that `influenced_by` edges are Cat I substrate records generated at substrate-write time. When an Assertion A's `influenced_by` chain reads {H₁, H₂, H₃}, A's own `evidential_independence` (if A is itself a hypothesis OR an assertion subject to §2.6) must reflect the independence-vs-inheritance distinction the §2.6 stub names. The wire shape question:

- (a) A's `evidential_independence` is a scalar derived from the chain's accumulated independence — §2.6 binding text specifies the derivation function.
- (b) A's `evidential_independence` is structurally a pair (own-independence, inherited-independence) — the two are reported separately.
- (c) A's `evidential_independence` is computed at projection time from the chain — substrate carries only the chain edges (per OMQ #3-α reasoning); the scalar is a projection product.

**Why surfaced:** §2.4's substrate-time edge generation makes the chain structurally inspectable; §2.6's binding text must specify how the chain's independence reads compose into A's own independence. The §0077 questions assumed independence is a per-formation field; §2.4 closure introduces the chain-traversal alternative.

**Pressure:** moderate. The question is structurally load-bearing for §2.6 binding text — without it, §2.6 redaction either silently picks (a/b/c) or carries the ambiguity into pending operational definitions.

### Q-§2.6-7 — Layer B activation predicate

**Question (new — refinement of §0077 Q-X1).** §2.5 Layer B forward-reference contract per [`§0011`](./decision-log.md) requires §2.4 + §2.6 BOTH frozen before [`ontology-revision-layer-b-deep-criterion`](../rfcs/draft/ontology-revision-layer-b-deep-criterion.md) activates. §2.4 is now frozen v0.5; §2.6 is the remaining gate. The activation-pending state means Layer B's structural form is held open by §2.6's pending status.

**Pressure increment:** strong. The activation-pending state IS structural pressure — the Charter has a known-pending forward-reference contract that resolves only when §2.6 freezes. Per [`§0011`](./decision-log.md), §2.5 frozen v0.3's Layer B reference is binding for the demotion candidacy criterion; the criterion's operational shape is gated by §2.6 redaction.

**Why now:** §0077 Q-X1 catalogued Layer B activation as cross-§2.4/§2.6 question. With §2.4 closed, the question collapses to a §2.6-only question: §2.6 redaction activates Layer B's deferred falsifiability obligation per the §2.5 contract.

## 5. Updated pressure assessment

The §0077 assessment ranked §2.6 pressure as **moderate**, **downstream of §2.4**. The refresh changes this on two axes:

- **Downstream condition discharged.** §2.4 is frozen; §2.6 is no longer waiting on §2.4's wire-shape commitments.
- **New questions accumulated.** Q-§2.6-3 / Q-§2.6-4 gained material cross-subtype framing-arc dependencies; Q-§2.6-6 (new) and Q-§2.6-7 (Layer B activation) accumulated since §0077.

**Updated pressure: strong, gated by cross-subtype-framing-arc resolution.** §2.6 binding text needs to either (a) pre-commit to cross-subtype-typing assumptions for Q-§2.6-3/4 OR (b) carry forward all three branches and operationalize at follow-on. Either path is a structural choice for the redaction phase.

**Per the §0022 posture:** redaction resumes when implementation surfaces concrete questions the Charter does not already answer. The refresh confirms §2.6 pressure has accumulated since §0077 — particularly the Q-§2.6-6 chain-traversal question (new since §2.4 closure) and the Layer B activation pressure (Q-§2.6-7). Whether the accumulation is sufficient to trigger redaction resumption is a separate committee decision.

## 6. Anchor inventory pre-Gate status

Per [`§0019`](./decision-log.md) lazy methodology + [`§0014`](./decision-log.md) pre-Gate dependency assessment:

- **§2.4 BC inheritance** — clean. §2.4 frozen v0.5 governs `influenced_by` edge timing + inferential-edge semantics; §2.6 governs the independence-dimension structural shape. The two are structurally complementary per [`§2.4 frozen v0.5 BC mutual-scope statement`](./constitutional-charter.md#24-inferential-influence-disclosure).
- **Q5 "transitive?" half** — open carry-forward from [`§0021`](./decision-log.md). Per §2.4 binding text's Boundary Condition: *"§2.4 does not govern the transitive scope of `influenced_by` chains."* §2.6 redaction inherits this carry-forward: independence-along-chain semantics (Q-§2.6-6) interact with the transitive-scope question. Whether §2.6 redaction blocks on the Q5 follow-on RFC OR uses the forward-reference contract pattern (per §2.3 BC4, §2.4 multi-forward-reference structure) is a §2.6 Step 1.1 question.
- **Q2 (Identity tiers)** — forward-referenceable continued. §2.4 binding text records the forward-reference marker; §2.6 inherits.
- **Layer B activation reconciliation** — §2.4 activated the §2.5 Layer B forward-reference contract per [`§2.4 frozen v0.5`](./constitutional-charter.md#24-inferential-influence-disclosure). §2.6 redaction completes the activation: Layer B's deep-criterion RFC becomes fully unblocked when §2.6 freezes.

**No new pre-Gate dependency surfaced** since §0077. The cross-subtype framing arc (PRs #95–#100) is a Q2-A.2 follow-on; it is upstream of the cross-subtype implementation work but not upstream of §2.6 binding text. §2.6 redaction may either pre-commit on cross-subtype typing (binding the typing RFC) or carry-forward (deferring operationalization) without blocking on the cross-subtype RFCs' committee resolution.

## 7. References

- [`empirical-pressure-2-4-2-6.md`](./empirical-pressure-2-4-2-6.md) — §0077 paired §2.4/§2.6 assessment; this document refreshes its §2.6 portion.
- [`decision-log §0077`](./decision-log.md) — original empirical-pressure assessment recording.
- [`decision-log §0099`](./decision-log.md) — §2.4 redaction closure (amendment v0.5).
- [`decision-log §0022`](./decision-log.md) — implementation-pivot + §2.4/§2.6 empirical-pressure-phase posture.
- [`decision-log §0008`](./decision-log.md) — redaction order §2.5 → §2.3 → §2.4 → §2.6.
- [`decision-log §0011`](./decision-log.md) — Q4 resolution; §2.5 Layer B forward-reference contract.
- [`decision-log §0020`](./decision-log.md) — OMQ #2-C (decay via §2.5 lifecycle event supersession).
- [`decision-log §0021`](./decision-log.md) — OMQ #3-α (substrate-time `influenced_by` edge generation).
- [Charter §2.4 frozen v0.5](./constitutional-charter.md#24-inferential-influence-disclosure)
- [Charter §2.6 (pending)](./constitutional-charter.md#26-evidential-independence-integrity)
- [`ontology-revision-layer-b-deep-criterion`](../rfcs/draft/ontology-revision-layer-b-deep-criterion.md) (on hold pending §2.6).
- [`ontology-revision-cross-subtype-merge-typing.md`](../rfcs/draft/ontology-revision-cross-subtype-merge-typing.md) + [`-enablement.md`](../rfcs/draft/ontology-revision-cross-subtype-merge-enablement.md) + [`-split.md`](../rfcs/draft/ontology-revision-cross-subtype-split.md) — cross-subtype framing arc; Q-§2.6-3/4 reference these for cross-subtype-typing-dependent independence rules.
