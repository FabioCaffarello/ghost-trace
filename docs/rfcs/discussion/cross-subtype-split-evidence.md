# Cross-subtype split — typing + enablement — discussion evidence

**Status:** in-discussion. Not authoritative. Final resolution will be recorded in decision-log and Ontology document revision.

This scratch supports the discussion phase of [`ontology-revision-cross-subtype-split.md`](../draft/ontology-revision-cross-subtype-split.md). The split RFC is consolidated (typing + enablement in one document); this evidence document mirrors that structure. Much of the analysis **inherits** from the cross-subtype merge evidence documents per the §0050 inverse-symmetry argument; novel split-specific findings are surfaced explicitly.

## Phase 1 — Evidence (compressed by inheritance)

Three typing candidates (α'/β'/γ') + three enablement candidates (A'/B'/D'). Rather than re-tabulate every cell, this evidence document records (a) where each split candidate's evidence is **identical** to the merge sibling, (b) where it **diverges** structurally, and (c) any **new** dimension introduced by split's 1-to-N shape.

### Typing inheritance map

| Dimension | α' inherits from α? | β' inherits from β? | γ' inherits from γ? |
|---|---|---|---|
| 1. Proto / type-layer | Mostly yes — new typed subtype `SplitFragment` has the same structural pattern as `CompositeHypothesis`, but carries a `target_subtype` discriminator instead of a `combined_subtypes` list. **Diverges:** the discriminator is single-valued per fragment (one of 4), not a set. | Yes — abstract `MergedHypothesis` accommodates split fragments via the same discriminator-list shape. The list carries the antecedent's subtype rather than the merged set. **Symmetric.** | Yes — per-pair table extends to per-source-subtype `permitted_targets` set. **Diverges:** the table answers "X may split into {Y, ...}?" vs merge γ's "{X, Y} merge → Z?". Cells co-locate naturally. |
| 2. Lifecycle composition | Yes — split-of-split produces more `SplitFragment` records of the same type; no type explosion. **Diverges from merge α's combinatorial concern**: α's `Composite{Composite, X}` grows under merge-of-merge, but α'`s `SplitFragment` records do not grow under split-of-split (each fragment is independent). | Yes — same merge-of-merge composition. | Yes — recursive table-lookup. |
| 3. Query / projection | Yes — α' adds a fifth typed query target; γ' adds no new types. | Yes. | Yes. |
| 4. Extensibility | Same as merge per-candidate. | Same. | Same — table extension per added subtype. |
| 5. Constitutional | Same — §4 criterion 1 disfavors β'; α' and γ' pass. | Same. | Same — γ''s type-vs-provenance asymmetry (per [`cross-subtype-merge-typing-evidence.md` Phase 1 cell γ5](./cross-subtype-merge-typing-evidence.md)) applies symmetrically. |

**Net finding:** typing candidates inherit substantially from merge. The one structural divergence is α''s composition behavior, which is BETTER than α's (no type growth under split-of-split). The merge evidence's structural argument against α (combinatorial type growth — [`cross-subtype-merge-typing-evidence.md` Finding 2](./cross-subtype-merge-typing-evidence.md)) does NOT apply to α' symmetrically.

### Enablement inheritance map

| Dimension | A' inherits from A? | B' inherits from B? | D' inherits from D? |
|---|---|---|---|
| 1. Helper-layer | Yes — same 1-line guard delete. | **Diverges:** B' inspects **membership partition** across N successors, not pairwise intersection. The partition check is N-ary: union of successor memberships = antecedent membership. More structural surface than merge B's binary intersection. | Yes — single antecedent lifecycle-state inspection (simpler than merge D's two-antecedent inspection). |
| 2. Operator-visible | Yes. | **Diverges:** "your successors must partition the antecedent's membership" — operator must structure the split's outputs to satisfy the partition gate, not just point at a shared element. | Yes — "antecedent must be promoted before split". |
| 3. Falsifiability | A' vacuous (same as A). | Passes all four — **stronger than B in §1.3 sense** because partition is more structurally constraining than non-empty intersection. | Same as D. |
| 4. Extensibility | Yes. | Yes — per-subtype membership surface required (same as merge B's actor-set extraction). | Yes. |
| 5. Constitutional | Same — A' makes no enablement claim. | Same — §4 criterion 1 cleanly enforced. | Same — Q4-Layer-A dependency. |

**Net finding:** enablement candidates inherit substantially from merge. The one structural divergence is B''s partition check (N-ary vs merge B's binary intersection). The partition shape is MORE constraining than intersection — under merge, antecedents may share one actor; under split, all of antecedent membership must be accounted for across successors. This makes B' a tighter structural gate than B.

## Phase 2 — Surface scaffold implicit assumptions (compressed)

The scaffold surfaces are [`entity-model.md` §Cross-subtype operations](../../ontology/entity-model.md) and [`lifecycle-semantics.md` §Split](../../ontology/lifecycle-semantics.md) (line 29 — *"Split. One hypothesis is recognized as containing multiple distinct phenomena ..."*).

**Verdict for both scaffolds: neutral on typing + enablement.** Same as the merge evidence per [`cross-subtype-merge-typing-evidence.md` Phase 2](./cross-subtype-merge-typing-evidence.md) + [`cross-subtype-merge-enablement-evidence.md` Phase 2](./cross-subtype-merge-enablement-evidence.md). The scaffolds describe split's 1-to-N shape (multiple successors per [`§0050`](../../charter/decision-log.md)) without committing to typing or enablement candidates.

**One scaffold-level divergence from merge:** the split scaffold's "multiple distinct phenomena" framing is candidate-neutral but reads slightly more naturally under α' or γ' than under β' (the "multiple distinct" phrasing presumes a type-level distinction across successors, which α' encodes via `target_subtype` and γ' encodes via the table; β' flattens the distinction into a single abstract type). The framing is suggestive, not committee-decisive — same Phase 2 caveat as the merge typing evidence.

## Phase 3 — Apply epistemic skills (compressed)

Three skills × three typing candidates × three enablement candidates = 18 cells if fully expanded. Compressed by inheritance from the merge evidence's twelve cells (per [`cross-subtype-merge-typing-evidence.md` Phase 3](./cross-subtype-merge-typing-evidence.md) + [`cross-subtype-merge-enablement-evidence.md` Phase 3](./cross-subtype-merge-enablement-evidence.md)).

### Inheritance findings (typing skills)

- **α' `falsifiability-check`:** passes all four; identical to merge α.
- **α' `epistemic-separator`:** clean; identical to merge α. **Divergence (positive):** α''s `SplitFragment` does NOT have merge α's combinatorial-growth concern.
- **α' `ambiguity-reducer`:** one carry-forward — `SplitFragment` term-naming (committee-pending), parallel to merge α's `CompositeHypothesis`.
- **β' three skills:** identical to merge β per [`cross-subtype-merge-typing-evidence.md` Phase 3 column β](./cross-subtype-merge-typing-evidence.md). §1.3 partial pass + intra-category flattening risk + identity-tier carry-forward all inherited.
- **γ' three skills:** identical to merge γ per [`cross-subtype-merge-typing-evidence.md` Phase 3 column γ](./cross-subtype-merge-typing-evidence.md). All four falsifiability rungs pass; type-vs-provenance asymmetry recorded.

### Inheritance findings (enablement skills)

- **A' three skills:** identical to merge A per [`cross-subtype-merge-enablement-evidence.md` Phase 3 column A](./cross-subtype-merge-enablement-evidence.md). Vacuous gate-falsifiability; operator-discretion extension recorded.
- **B' three skills:** identical to merge B in structure; **B''s partition gate is structurally stricter than B's intersection gate**, so the falsifiability `§1.3 operationalization` rung is even cleaner under B' (partition is a tighter typed-set check than intersection-non-empty).
- **D' three skills:** identical to merge D. Q4-Layer-A coupling inherited.

### Most consequential split-specific finding

**B''s partition check is structurally stricter than B's intersection check.** The merge evidence's Finding 2 noted B's rigidity (rejects actor-disjoint same-phenomenon merges). The split symmetric concern is even tighter: under B', the operator must construct successors whose memberships exactly partition the antecedent's — no actor created, no actor lost, no overlap between successors. This is a substantial operational discipline; whether it is the right shape for cross-subtype split is committee-pending.

## Phase 4 — Comparison synthesis (compressed)

The findings from the merge evidence's Phase 4 inherit symmetrically:

- **Finding 1 (merge):** §4 discipline disfavors β. **Symmetric:** disfavors β' for split.
- **Finding 2 (merge):** composition favors γ over α. **Asymmetric divergence:** for split, α' has no combinatorial-growth concern. γ' and α' are closer on composition than γ and α were for merge.
- **Finding 3 (merge enablement):** D's Q4-coupling is structural alignment. **Symmetric:** applies to D'.
- **Finding 4 (merge):** type-vs-provenance information locality (γ vs α). **Symmetric:** applies to γ' vs α'.
- **Finding 5 (merge):** A's symmetric-simplicity vs §4 discipline. **Symmetric:** applies to A'.

### New finding: symmetric-resolution structural argument (split-specific)

Per [the split RFC §Symmetric resolution preference](../draft/ontology-revision-cross-subtype-split.md), [`§0050`](../../charter/decision-log.md) names split as the structural inverse of merge. The discussion phase weighs symmetric-resolution as a meta-question:

- **Symmetric resolution is structurally preferable** when the §0050 inverse-symmetry argument is treated as load-bearing. Under symmetric resolution, the cross-subtype merge and split surfaces are co-evaluated by the committee; both resolve to the same candidate family. The shared per-pair table per [`§Combined-table option`](../draft/ontology-revision-cross-subtype-split.md) is the structural artifact.
- **Independent resolution is permitted** when split-specific evidence warrants. The merge evidence recommended γ + B+D; the split evidence (this document) finds the same recommendation symmetrically (γ' + B'+D'), with the partition-gate strictness noted as a structural-defense advantage of B' over B.

**Symmetric-resolution lean:** the split-specific evidence does not surface findings that warrant divergence from the merge resolution. The composition concern (Finding 2 divergence — α' has no growth) is a *softening* of the merge argument against α, not a *flip* in favor of α' — γ' remains preferable per Findings 1 + 4. The partition-gate strictness of B' (relative to B) is a confirmation that structural-defense scales under split's 1-to-N shape, not a reason to prefer a different enablement candidate.

## Phase 5 — Conditional recommendation

The discussion phase recommends **γ'-typing + B'+D'-enablement** as the resolution for cross-subtype split, **conditional on symmetric resolution with the cross-subtype merge framing** (γ-typing + B+D-enablement per the merge evidence's recommendations).

The recommendation rests on:

- **Symmetric inheritance from merge.** The merge evidence recommendations stand; split inherits via §0050 inverse-symmetry. The shared per-pair table is the structural artifact that makes symmetric resolution operationally tight.
- **Split-specific confirmations.** B''s partition gate is structurally stricter than B's intersection gate; α''s composition is better than α's. Neither finding warrants divergence from the merge recommendation.

### What would warrant asymmetric resolution

Asymmetric resolution (merge γ + B+D; split α' + B'+D', or any non-symmetric pair) is justified if the committee surfaces:

- **Split-specific operator workload requiring α' for type-level discrimination of split outputs.** If projections, replay handlers, or operator-facing tools commonly need to type-discriminate "this is a split fragment" at the surface level, α' becomes preferable for split (parallel to merge Finding 4's flip-to-α condition). γ''s antecedent-list-inspection cost is lighter for merge (one antecedent-pair lookup) than for split (N-successor-list scan), so the cost asymmetry may favor α' for split even when γ is preferred for merge.
- **Split-specific enablement requiring D'-alone (no partition gate).** If partition-strictness rejects too many epistemically-valid splits (e.g., a CR antecedent splitting into {CR, BC} where the BC fragment includes actors discovered ONLY at the split moment, not in the antecedent's pre-split membership), B' becomes too restrictive. D'-alone (lifecycle-state gate only) is the looser alternative.

### Methodological observation

This is the **third** Ontology RFC discussion to converge by two-stage filter — but with a new structural property: **convergence by inheritance**. The split evidence inherits substantially from the merge evidence; the convergence is preserved across the inverse-symmetric operation pair. This is structurally distinct from:

- Q4's staged-combination convergence ([`§0011`](../../charter/decision-log.md)) — single-question convergence.
- Q2's binary by-asymmetry convergence ([`q2-evidence.md`](./q2-evidence.md)) — single-question convergence.
- Merge typing's two-stage convergence ([`cross-subtype-merge-typing-evidence.md`](./cross-subtype-merge-typing-evidence.md)) — single-question convergence.
- Merge enablement's two-stage convergence ([`cross-subtype-merge-enablement-evidence.md`](./cross-subtype-merge-enablement-evidence.md)) — single-question convergence.

**Convergence-by-inheritance** is a new pattern: a structurally-paired question (split's inverse of merge) inherits its resolution from the paired question via §0050 inverse-symmetry. The pattern requires (a) a structural argument for symmetry (§0050 here); (b) explicit surfacing of any divergence; (c) explicit reversal conditions for the symmetric default. All three are present in this evidence document.

The precedent: future structurally-paired questions (e.g., merge-of-merge / split-of-split composition; cross-subtype demotion as inverse of cross-subtype promotion) may use convergence-by-inheritance as the discussion-phase pattern when one half of the pair has already been resolved. The pattern is NOT applicable when no structurally-paired prior resolution exists.
