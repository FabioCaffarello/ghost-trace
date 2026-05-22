# RFC — Layer B Deep Criterion (Q4 follow-on)

- **Status:** discussion (substantive deliberation complete — recommendation: Candidate L-BC-OR with L-C structural-exclusion commitment; formal resolution pending committee ratification)
- **Authors:** Ghost Trace committee
- **Date:** 2026-05-15
- **Type:** ontology-revision
- **Affects:** [`docs/ontology/lifecycle-semantics.md`](../../ontology/lifecycle-semantics.md) §The Promotion Mechanism step 4; [Charter §2.5](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) (pending — binding text refinement once Layer B is specified)

> RFCs in Ghost Trace are proposals subject to constitutional review. The [Constitutional Charter](../../charter/constitutional-charter.md) is the authoritative reference. RFCs that conflict with the Charter must either be rejected or accompanied by a corresponding charter amendment (see [`../../charter/amendments.md`](../../charter/amendments.md)).

## Status note

**All 4 pre-Gate dependencies are now discharged; substantive content drafting opens.**

- **§2.4** frozen v0.5 at [`§0099`](../../charter/decision-log.md). ✓
- **§2.6** frozen v0.6 at [`§0129`](../../charter/decision-log.md). ✓
- **Q3 (formal definition of `evidential_independence`)** resolved at [`§0133`](../../charter/decision-log.md): Candidate α (source-count ratio over Cat I provenance roots) adopted under [§2.6 BC2](../../charter/constitutional-charter.md#26-evidential-independence-integrity) meta-shape 1 (deterministic-from-pattern). ✓
- **Q5 (influence propagation)** fully resolved: decay half at [`§0020`](../../charter/decision-log.md) (Candidate C — via §2.5 lifecycle supersession); transitivity half at [`§0134`](../../charter/decision-log.md) (Candidate τ — transitive closure of declared direct edges, with β-graph storage: substrate stores direct edges + per-record cached closures). Committee extension: Cat II constructs structurally transmit `influenced_by` membership from their inputs per [§2.2](../../charter/constitutional-charter.md#22-epistemic-separation) determinism. ✓

The two-cascade chain Q3 → Q5 → Layer B is fully discharged on the ontology side per [`§0134`](../../charter/decision-log.md) Methodological Observation 3. The [`§0011`](../../charter/decision-log.md) Layer B contract has reached its full activation surface. Layer B's deep criterion now has a fully-specified measurable quantity (α with transitive reachability under τ) to threshold-test. Substantive content drafting — specifying which combination of Candidate B family from Q4 (evidence-staleness using α) and/or Candidate C family from Q4 (influence-saturation using α) constitutes the deep criterion — opens as the next substantive RFC arc.

## Summary

[`decision-log.md` §0011](../../charter/decision-log.md) (Q4 resolution) adopted the staged-combination form for the demotion-candidacy criterion: Layer A (time-based cadence gate, operational today) AND Layer B (deep criterion on `evidential independence` or declared `influence`). Layer B's specific structural form — which combination of evidence-staleness (Candidate B family from Q4) and/or influence-saturation (Candidate C family from Q4) constitutes the deep criterion — is deferred to this RFC. All 4 pre-Gate dependencies are now discharged: §2.4 frozen v0.5 + §2.6 frozen v0.6 + Q3-α at [`§0133`](../../charter/decision-log.md) + Q5-τ at [`§0134`](../../charter/decision-log.md). The measurable quantity Layer B's deep criterion threshold-tests is now fully specified: `evidential_independence` per Q3-α with transitive reachability under Q5-τ.

## Motivation

The Q4 resolution committed to a structural form without specifying Layer B's deep criterion in order to permit §2.5 redaction to proceed per [`decision-log.md` §0008](../../charter/decision-log.md)'s redaction order. With all 4 pre-Gate dependencies now discharged, Layer B's substantive content opens: which combination of Candidate B family (evidence-staleness using Q3-α) and/or Candidate C family (influence-saturation using Q5-τ) constitutes Layer B's deep criterion, and how the chosen family or families compose within Layer B itself.

The cost of not resolving: the two-cascade chain Q3 → Q5 → Layer B's structural surface is complete, but Layer B's firing predicate remains structurally undefined. The §2.5 lifecycle event chain has no deep criterion to fire under, and the canonical-serialization-contract revision (per [`§0133`](../../charter/decision-log.md) + [`§0134`](../../charter/decision-log.md) follow-on schedule) cannot crystallize Layer B's structural form until this RFC resolves.

## Constitutional Review

Per [`rfc-author` §1](../../../.claude/skills/workflow/rfc-author/SKILL.md), the Q1–Q6 impact analysis applies.

### Q1 — Which Charter invariants does this RFC touch?

- **§2.5 Hypothesis Lifecycle Explicitness** (frozen v0.3): touched directly. §2.5 codifies the `demotion` lifecycle event; Layer B's firing produces it. The resolution does not modify §2.5; it specifies the firing predicate the demotion event records.
- **§2.6 Evidential Independence Integrity** (frozen v0.6): touched directly. Q3-α (per [`§0133`](../../charter/decision-log.md)) is the B-family input. Layer B's evidence-staleness metric reads `evidential_independence` per §2.6's paired-dimension commitment.
- **§2.4 Inferential Influence Disclosure** (frozen v0.5): touched directly. Q5-τ (per [`§0134`](../../charter/decision-log.md)) provides the transitive `influenced_by` semantic. Layer B's influence-saturation metric reads the transitive influence chain per §2.4 + Q5-τ.
- **§1 Thesis**: Layer B is the structural test for the two failure modes the Thesis names — "confidence in inferences inflates without proportional increase in independent evidence" (B-family) and "promoted hypotheses re-enter as enrichment and silently reinforce themselves" (C-family). The resolution determines which halves are structurally tested.

### Q2 — Does this RFC implicitly redefine any term in the glossary?

No. The RFC uses canonical vocabulary (`evidential_independence`, `influence`, `confidence`, `demotion`, `subject_ref_hypothesis`) without redefinition. The candidates' operational sketches introduce metric names (`freshness_B`, `saturation_C`) but these are operational specification terms, not constitutional vocabulary.

### Q3 — Does this RFC implicitly resolve any of the five open Ontology questions?

No. All five original ontology.md Open Questions are resolved as of [`§0134`](../../charter/decision-log.md) Methodological Observation 4. This RFC composes with Q3 (α) and Q5 (τ) resolutions; it does not re-open or implicitly modify them.

### Q4 — Does this RFC require Charter amendment?

No. Layer B's specification is a §2.5 binding-text refinement at the Ontology layer; §2.5 already forward-references Layer B explicitly. The resolution refines the forward-reference content, not the §2.5 invariant itself.

### Q5 — Does this RFC introduce a new invariant?

No. The RFC specifies Layer B's structural form within the §2.5 lifecycle framework. The invariant that consumes Layer B is §2.5; Layer B is a structural test, not a new invariant.

### Q6 — Does this RFC propose ceremony without behavioral consequence?

No. Layer B's firing produces a `demotion` lifecycle event per §2.5; the behavioral consequence is direct (a previously-promoted hypothesis exits operational use as enrichment context). The choice among candidates determines which §1 Thesis failure mode(s) the system structurally defends against.

## Proposal

Five candidate compositions enumerated in [`layer-b-deep-criterion-evidence.md`](../discussion/layer-b-deep-criterion-evidence.md) Phase 2:

- **L-B — Evidence-staleness alone:** `Layer B(H) := freshness_B(H) < T_B`. Addresses §1 Thesis first naming (loss of independent support).
- **L-C — Influence-saturation alone:** `Layer B(H) := saturation_C(H) > K_C`. Addresses §1 Thesis second naming (self-reinforcing saturation).
- **L-BC-AND — Conjunctive:** `Layer B(H) := (freshness_B < T_B) AND (saturation_C > K_C)`. Intersection of both naming halves; high bar.
- **L-BC-OR — Disjunctive:** `Layer B(H) := (freshness_B < T_B) OR (saturation_C > K_C)`. Union of both naming halves; maximum coverage.
- **L-BC-staged — Sequential:** structurally identical to L-BC-OR; commits additionally to an evaluation order at the canonical-serialization-contract layer.

This RFC does not pick a candidate at this phase. The discussion scratch records three asymmetries that will likely organize substantive deliberation:

- **§1 Thesis-coverage asymmetry:** L-B / L-C cover one half each; L-BC-AND covers the intersection; L-BC-OR / L-BC-staged cover the union. The Thesis names both halves as independent failure modes; **maximum coverage favors L-BC-OR / L-BC-staged.**
- **Single-criterion false-positive resistance asymmetry:** L-BC-AND is most resistant; L-BC-OR / L-BC-staged are least resistant. If parameter calibration is operationally uncertain, false-positive resistance is a discipline concern.
- **Structural-vs-operational separation asymmetry:** L-BC-staged blurs the form-vs-parameter discipline by committing to an evaluation order; L-BC-OR keeps the boundary clean.

The outer Layer A AND Layer B composition resolved at [`§0011`](../../charter/decision-log.md) is preserved; this RFC addresses Layer B's inner structure only.

## Alternatives Considered

Out-of-scope candidate forms surfaced and explicitly rejected during framing:

- **Layer B operating on `confidence` alone.** Rejected by [Charter §2.6](../../charter/constitutional-charter.md#26-evidential-independence-integrity) anti-pattern 2 ("Independence collapsed into confidence at projection"). Any candidate whose firing predicate reads `confidence` as a proxy for `evidential_independence` is structurally forbidden. Not enumerated.
- **Layer B operating on a runtime-derived metric not anchored at substrate.** Rejected by [`§0021`](../../charter/decision-log.md) substrate-time-generation discipline. Layer B's firing must reduce to substrate-committed queries; runtime-classified metrics are forbidden. Not enumerated.
- **Layer B operating without Layer A.** Rejected by [`§0011`](../../charter/decision-log.md) outer AND-composition (DEMOTE-CANDIDATE = Layer A AND Layer B). Reversal to OR-composition would require a separate Q4-revision RFC per the §0011 reversal-conditions record. Not enumerated as a Layer B candidate.
- **Layer B with operator-supplied threshold values.** Rejected analogously to Q3-ε's elimination — operator-configurable thresholds at runtime violate [§4 criterion 1](../../charter/constitutional-charter.md#4-constitutional-design-rule) structural enforceability. Threshold parameters T_B and K_C must be canonical-serialization-contract-fixed structural constants. (Parameter VALUES may be deferred to operational specification RFC; parameter MUTABILITY at runtime is forbidden.)

## Open Questions

The RFC's own open questions to be resolved when it advances substantively:

- **Which candidate (L-B / L-C / L-BC-AND / L-BC-OR / L-BC-staged) does the resolution adopt?** Load-bearing sub-decision.
- **If L-BC-AND / L-BC-OR / L-BC-staged: what are the operational forms of `freshness_B` and `saturation_C` under Q3-α + Q5-τ?** Structural sketches in [`layer-b-deep-criterion-evidence.md`](../discussion/layer-b-deep-criterion-evidence.md) Phase 2 are illustrative; substantive content crystallizes the structural form.
- **What are the parameter values T_B (B-family threshold) and K_C (C-family ratio)?** Deferrable to follow-on operational-specification RFC per the form-vs-parameter discipline.
- **Per-subtype vs uniform parameters under [`§0010`](../../charter/decision-log.md) Q2-A.2.** Forward-referenceable per Q3 + Q5 Phase 4 findings; inception-phase default is uniform at the abstract `Hypothesis` level.
- **Definition of "recent assertions" window.** Both B-family and C-family operate on a "recent" subset of substrate; the window's structural form (fixed time, fixed count, or hybrid) is an operational sub-decision deferrable per form-vs-parameter discipline.

## Anti-Patterns to Avoid

Surfaced during framing for committee discipline in subsequent phases:

- **Confidence/independence collapse in the freshness metric.** Layer B's `freshness_B` must read `evidential_independence` per §2.6, NOT `confidence`. The §2.6 anti-pattern 2 prohibition applies at Layer B's firing predicate just as it does at α's substrate-commit predicate. Detection: the resolution must specify that `freshness_B` reads `evidential_independence` explicitly.
- **Inner-vs-outer composition conflation.** The outer Layer A AND Layer B composition is committee-resolved at [`§0011`](../../charter/decision-log.md). Layer B's inner composition (AND, OR, staged among B-family and C-family) is an independent structural question. The resolution must NOT implicitly modify the outer composition. Detection: the resolution names the outer composition as unchanged AND specifies the inner composition explicitly.
- **Parameter-vs-form conflation.** Threshold values T_B and K_C are operational parameters; Layer B's form (which candidate) is structural. The form-vs-parameter discipline established at [`§0011`](../../charter/decision-log.md) Methodological Observation 1 separates the two; the resolution must commit to the form without crystallizing parameter values (unless the committee chooses to bundle, in which case the bundling is explicit).
- **§1 Thesis half-defense.** Adopting L-B alone or L-C alone leaves one §1 Thesis failure mode structurally undefended. If the resolution chooses a single-family candidate, the procedural defense for the unaddressed half must be named (typically: operator vigilance + RFC reversal-condition record).

## Migration and Backward Compatibility

No historical Category III records exist at this point. Layer B's specification is forward-looking. The placeholder records that Layer B's eventual binding text must be expressible without retroactive substrate revision: a §2.5 binding text initially redacted with Layer B forward-referenced as "deferred to this RFC" must remain valid prose; this RFC's resolution refines the forward reference, not the §2.5 invariant itself.

## References

- [Charter §1 Thesis](../../charter/constitutional-charter.md#1-thesis) — names both failure modes Layer B defends against.
- [Charter §2.4](../../charter/constitutional-charter.md#24-inferential-influence-disclosure) v0.5 — `influenced_by` chain structural surface; C-family input via Q5-τ.
- [Charter §2.5](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) v0.3 — `demotion` lifecycle event Layer B's firing produces.
- [Charter §2.6](../../charter/constitutional-charter.md#26-evidential-independence-integrity) v0.6 — `evidential_independence` structural surface; B-family input via Q3-α.
- [`decision-log.md` §0011](../../charter/decision-log.md) — Q4 resolution; opens this RFC as placeholder.
- [`decision-log.md` §0099](../../charter/decision-log.md) — §2.4 closure.
- [`decision-log.md` §0129](../../charter/decision-log.md) — §2.6 closure.
- [`decision-log.md` §0133](../../charter/decision-log.md) — Q3-α resolution. α is the B-family freshness metric input.
- [`decision-log.md` §0134](../../charter/decision-log.md) — Q5-τ resolution. τ is the C-family saturation metric input. Layer B substantive content opens per consequences.
- [`layer-b-deep-criterion-evidence.md`](../discussion/layer-b-deep-criterion-evidence.md) — discussion-phase evidence scratch (Phase 1 + Phase 2).
- [`q3-independence-evidence.md`](../discussion/q3-independence-evidence.md) — Q3 discussion-phase evidence.
- [`q4-evidence.md`](../discussion/q4-evidence.md) — Q4 discussion-phase evidence (original B/C family enumeration).
- [`q5-transitivity-evidence.md`](../discussion/q5-transitivity-evidence.md) — Q5 discussion-phase evidence.
- [`ontology-revision-q4-promotion-demotion-criterion`](./ontology-revision-q4-promotion-demotion-criterion.md) — upstream Q4 RFC.
- [`docs/ontology/lifecycle-semantics.md` §The Promotion Mechanism](../../ontology/lifecycle-semantics.md) — current binding home of Layer B's forward reference.

## Decision Record

Substantive deliberation complete; formal resolution pending. The discussion-phase deliberation recorded in [`layer-b-deep-criterion-evidence.md`](../discussion/layer-b-deep-criterion-evidence.md) Phases 3–5 recommends **Candidate L-BC-OR (disjunctive — `Layer B(H) := (freshness_B(H) < T_B) OR (saturation_C(H) > K_C)`)** with the **L-C structural-exclusion commitment** (saturation denominator excludes H's own enrichment outputs per §2.4 v0.5 chain inspection).

Full demotion-candidacy predicate:

> `DEMOTE-CANDIDATE(H) := Layer A(H) AND ((freshness_B(H) < T_B) OR (saturation_C(H) > K_C))`

The recommendation rests on Phase 4 Findings 1 (only L-BC-OR / L-BC-staged structurally cover both §1 Thesis failure modes), 2 (L-BC-AND systematically under-defends — admits single-half violations as structurally-undefended states), 3 (L-BC-staged blurs form-vs-parameter discipline; L-BC-OR preserves it), 4 (false-positive resistance concern is parameter-calibration-addressable; structural-coverage gap is not), 5 (L-C structural-exclusion commitment is constitutional, not operational), 6 (L-B's transitive-reading requirement pre-satisfied by §0134).

Two committee extensions: (a) L-C structural-exclusion commitment — saturation denominator excludes H's own enrichment outputs (structural mirror of Q4 Phase 3 Finding 6 for the C-family); (b) form-vs-parameter discipline preserved — T_B / K_C / N values open as operational-specification follow-on RFC.

Resolution lands at a future `decision-log` entry that fully discharges the §2.5 binding-text Layer B forward-reference per [`§0011`](../../charter/decision-log.md) contract, completes the two-cascade chain Q3 → Q5 → Layer B, and feeds the canonical-serialization-contract revision (consolidated architecture-document RFC crystallizing α + τ + L-BC-OR per [`§0133`](../../charter/decision-log.md) + [`§0134`](../../charter/decision-log.md) follow-on schedule). Parameter-calibration operational-specification RFC opens as separate follow-on.
