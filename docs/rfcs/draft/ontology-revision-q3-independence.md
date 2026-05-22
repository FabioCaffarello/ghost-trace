# RFC — Ontology Open Question 3: Formal definition of `evidential independence` as measurable quantity

- **Status:** accepted
- **Authors:** Ghost Trace committee (Layer B follow-on RFC pre-Gate; opened per [`decision-log §0132`](../../charter/decision-log.md); discussion-phase deliberation Phases 3–5 recorded in [`q3-independence-evidence.md`](../discussion/q3-independence-evidence.md); resolution at [`decision-log §0133`](../../charter/decision-log.md) adopting Candidate α)
- **Date:** 2026-05-22 (opened); 2026-05-22 (resolved)
- **Type:** ontology-revision
- **Affects:** [`docs/ontology/ontology.md`](../../ontology/ontology.md) (Open Question 3 closed by resolution); [Charter §2.6 BC1](../../charter/constitutional-charter.md#26-evidential-independence-integrity) (frozen v0.6 — operational specification deferred here); [Charter §2.4](../../charter/constitutional-charter.md#24-inferential-influence-disclosure) (frozen v0.5 — `influenced_by` chain is structural input to most candidate derivation rules); [`docs/architecture/canonical-serialization-contract.md`](../../architecture/canonical-serialization-contract.md) (the per-record `evidential_independence` field's type/range is contract surface per [`§0034`](../../charter/decision-log.md)); [`ontology-revision-layer-b-deep-criterion`](./ontology-revision-layer-b-deep-criterion.md) (Layer B follow-on RFC consumes Q3 resolution as the measurable quantity its deep criterion threshold-tests)

> RFCs in Ghost Trace are proposals subject to constitutional review. The [Constitutional Charter](../../charter/constitutional-charter.md) is the authoritative reference. RFCs that conflict with the Charter must either be rejected or accompanied by a corresponding charter amendment (see [`../../charter/amendments.md`](../../charter/amendments.md)).

## Summary

[`ontology.md` Open Question 3](../../ontology/ontology.md): what is the formal definition of `independence` (resolved under canonical-vocabulary discipline to `evidential_independence` per [Charter §2.6](../../charter/constitutional-charter.md#26-evidential-independence-integrity)) as a measurable quantity? §2.6 frozen v0.6 commits the structural pairing of `confidence` + `evidential_independence` at substrate per [`§0034`](../../charter/decision-log.md) canonical-serialization-contract; §2.6 BC1 explicitly defers the measurable-quantity definition to this RFC.

This RFC opens structured discussion. The candidate measurable-quantity families are enumerated in [`q3-independence-evidence.md`](../discussion/q3-independence-evidence.md) Phase 2 (six candidates partitioned by §2.6 BC2's three meta-shapes: deterministic-from-pattern, substrate-computed, operator-supplied). This RFC does not pick a candidate at this phase.

## Motivation

**Why now.** [`§0011`](../../charter/decision-log.md) opened the Layer B follow-on RFC at on-hold status pending §2.4 + §2.6 redaction. Both Charter dependencies are now satisfied: §2.4 frozen v0.5 at [`§0099`](../../charter/decision-log.md); §2.6 frozen v0.6 at [`§0129`](../../charter/decision-log.md). The Layer B RFC's Status note names Q3 + Q5 as ontology-side dependencies remaining before Layer B's substantive content can be drafted. Q3 is the most upstream of the two: without a measurable `evidential_independence` quantity, Q5's transitivity question has no measurable target to propagate, and Layer B's deep criterion has no quantity to threshold-test.

Opening Q3 at discussion now (rather than waiting for Layer B to surface it) follows the [`§0014`](../../charter/decision-log.md) lazy pre-Gate methodology: each Charter or RFC redaction opens its minimal pre-Gate at discussion status; subsequent dependencies are assessed empirically during substantive phases. Q3 is the minimal pre-Gate for Layer B's substantive content.

**The cost of not resolving.** Pattern established by Q1 [`§0015`](../../charter/decision-log.md) + Q3-subject-ref [`§0016`](../../charter/decision-log.md) + OMQ #2 [`§0020`](../../charter/decision-log.md) + OMQ #3 [`§0021`](../../charter/decision-log.md): open Ontology questions are committee-resolved, not infrastructure-resolved. Implementation work that requires reading the `evidential_independence` field — for example, the Layer B deep criterion, or any projection consuming the dimension per §2.6 anti-pattern 2 (byte-for-byte projection-replay match) — has no derivation rule to verify against. Without Q3 resolution, the §2.6 anti-pattern 2 detection mechanism (projection-replay diff) is structurally undefined.

## Constitutional Review

The Q1–Q6 impact analysis prescribed by [`rfc-author` §1](../../../.claude/skills/workflow/rfc-author/SKILL.md).

### Q1 — Which Charter invariants does this RFC touch?

- **§2.6 Evidential Independence Integrity** (frozen v0.6 — this RFC's purpose): BC1 explicitly defers the measurable-quantity definition to Q3. This RFC's resolution discharges BC1's deferral and activates the "structurally falsifiable" status §2.6 BC1 names.
- **§2.4 Inferential Influence Disclosure** (frozen v0.5): touched indirectly. Candidates α / β / γ / δ-with-graph-formula / ζ-baseline (per evidence Phase 2) consume the `influenced_by` chain as structural input. The resolution does not modify §2.4 but reads its structural surface.
- **§2.3 Provenance Integrity** (frozen v0.4): touched indirectly. The typed `subject_ref_*` chain is the provenance subgraph candidates α / β / γ traverse. The resolution does not modify §2.3 but reads its structural surface.
- **Layer B follow-on RFC** ([`ontology-revision-layer-b-deep-criterion`](./ontology-revision-layer-b-deep-criterion.md), on hold per [`§0011`](../../charter/decision-log.md) → discussion-active per [`§0129`](../../charter/decision-log.md)): Q3 resolution feeds Layer B's deep criterion specification — Layer B's threshold-test operates on the quantity Q3 defines.

### Q2 — New glossary terms?

Depends on resolution candidate. Per evidence Phase 2:

- Candidates α / β / γ may introduce terms naming the graph-traversal formulae (e.g., `source-count ratio`, `influence-edge fraction`, `topological-distance to non-influenced ancestor`). Likely formalized at resolution commit.
- Candidate δ (substrate-computed with operationally-specified formula) introduces no new glossary terms at the Charter level; the operational specification document may.
- Candidate ε (operator-supplied) may surface `operator-declared independence` as a glossary entry to distinguish from substrate-computed forms.
- Candidate ζ (hybrid) may surface `substrate-computed baseline` and `operator-declared refinement` per its two-value structure.

No glossary modifications in this discussion phase per Q1 / OMQ #2 / OMQ #3 precedent.

### Q3 — Resolves an Ontology open question?

**Yes.** This RFC IS the resolution of `ontology.md` Open Question 3. Resolution closes the open question; ontology.md is updated to mark Q3 resolved with reference to the resolution decision-log entry.

### Q4 — Charter amendment?

**No.** §2.6 BC1 explicitly anticipates this RFC: "the formal mechanism becomes structurally falsifiable when Q3 resolution lands." The resolution is procedurally a sub-Charter discharge of a deferred operational specification, not a Charter amendment. Subsequent canonical-serialization-contract revisions per [`§0034`](../../charter/decision-log.md) discipline accommodate the resolution without Charter amendment.

A Charter amendment would be required only if the resolution surfaced an invariant gap §2.6 does not anticipate — for example, a candidate whose structural form is incompatible with §2.6's "structurally independent at substrate" commitment (line 243 of [Charter §2.6](../../charter/constitutional-charter.md#26-evidential-independence-integrity)). No candidate enumerated in evidence Phase 2 has this property; the assessment is empirical and re-applied at resolution.

### Q5 — New invariant?

**No.** The RFC resolves an operational specification deferred by §2.6 BC1. The invariant that consumes the resolution is §2.6 itself; the resolution refines BC1's "deferred" status to "specified."

### Q6 — Ceremony or constitutional?

**Constitutional, not ceremony.** The choice between meta-shape 1 / 2 / 3 (per §2.6 BC2) is materially different at substrate, projection, and Layer B consumer levels:

- **Substrate layer** — meta-shape 1 freezes a formula in the canonical-serialization-contract; meta-shape 2 keeps the formula in operational specification with substrate validation of output type/range only; meta-shape 3 admits operator-declared values with substrate validation of type/range only.
- **Projection layer** — meta-shape 1 enforces byte-for-byte projection-replay match against the contract-specified formula; meta-shape 2 enforces match against the operational-specification-specified formula; meta-shape 3 enforces match against the producer-declared value verbatim.
- **Layer B consumer layer** — meta-shape 1's threshold-test reads a fully-specified structural quantity; meta-shape 2's reads a partly-deferred quantity; meta-shape 3's reads a producer-attested quantity. Layer B's deep criterion shape depends on which.
- **[Charter §4](../../charter/constitutional-charter.md#4-constitutional-design-rule) criterion 1 (structural enforceability)** — meta-shapes 1 / 2 satisfy structurally; meta-shape 3 satisfies procedurally (producer-accountable). Q4 criterion 1 questions whether procedural defense suffices for a Charter-defended dimension.

Behavioral consequences cascade through §2.6 anti-pattern 2 detection, Layer B specification, and any consumer reading `evidential_independence`. Constitutional.

## Proposal

Six candidates enumerated in [`q3-independence-evidence.md`](../discussion/q3-independence-evidence.md) Phase 2, partitioned by §2.6 BC2's three meta-shapes:

- **Meta-shape 1 — Deterministic-from-pattern:** Candidate α (source-count ratio), Candidate β (influence-edge fraction), Candidate γ (topological-distance to nearest non-influenced ancestor).
- **Meta-shape 2 — Substrate-computed with operationally-specified formula:** Candidate δ.
- **Meta-shape 3 — Operator-supplied:** Candidate ε (operator-supplied within canonical-serialization-contract validation), Candidate ζ (hybrid — substrate-computed baseline + operator-declared refinement).

This RFC does not pick a candidate at this phase. The discussion scratch records two asymmetries that will likely organize substantive deliberation:

- **Structural-enforceability asymmetry (§4 criterion 1):** α / β / γ / ζ-baseline are structurally enforceable at the substrate; δ defers part of the enforcement to operational specification; ε defers all to procedural defense.
- **Q5-transitivity dependency asymmetry:** α / β / γ / δ-with-graph-formula / ζ-baseline depend on [`ontology.md` Q5](../../ontology/ontology.md) transitivity-half resolution; ε / δ-with-non-graph-formula do not. Q5 cascades to pre-Gate status only if a chosen Q3 candidate cannot be specified without resolving transitivity (per [`§0015`](../../charter/decision-log.md) → [`§0016`](../../charter/decision-log.md) + [`§0020`](../../charter/decision-log.md) → [`§0021`](../../charter/decision-log.md) cascade precedent).

## Alternatives Considered

Out-of-scope candidate forms surfaced and explicitly rejected during framing:

- **Confidence-derived independence.** Rejected by [Charter §2.6 anti-pattern 2](../../charter/constitutional-charter.md#26-evidential-independence-integrity) — "Independence collapsed into confidence at projection." Any candidate whose derivation rule produces `evidential_independence = f(confidence)` is structurally forbidden regardless of meta-shape. Not enumerated as a candidate.
- **Runtime-classified independence.** Rejected by [`§0015`](../../charter/decision-log.md) + [`§0016`](../../charter/decision-log.md) + [`§0020`](../../charter/decision-log.md) cumulative precedent — runtime classification at projection violates [§2.2](../../charter/constitutional-charter.md#22-epistemic-separation) epistemic-separation and §2.6's "structurally independent at substrate" commitment. Not enumerated.
- **Post-commit independence mutation.** Rejected by [Charter §2.1](../../charter/constitutional-charter.md#21-observational-integrity) substrate-immutability inheritance for Cat II / Cat III records (per §2.6 anti-pattern 4 — "Per-record amendment of the independence dimension post-commit"). Not enumerated.
- **Offline-only derivation.** Rejected by §2.6 anti-pattern 5 — "Independence derived offline only." Not enumerated.

## Open Questions

The RFC's own open questions to be resolved when it advances:

- **Which meta-shape (1, 2, or 3) does the resolution adopt?** This is the load-bearing sub-decision.
- **If meta-shape 1: which specific formula (α, β, γ, or a variant not enumerated)?**
- **If meta-shape 2: what operational-specification path does the substrate consume, and what is the canonical-serialization-contract's validation surface?**
- **If meta-shape 3: what canonical-serialization-contract validation suffices for §4 criterion 1 procedural-defense adequacy?**
- **Does the resolution open Q5 transitivity-half as cascade per §0014 → §0015 → §0016 + §0019 → §0020 → §0021 precedent?** Depends on chosen candidate; cascade-trigger discipline applies only if the chosen candidate cannot be specified without Q5.
- **Per-subtype vs uniform application under [`§0010`](../../charter/decision-log.md) Q2-A.2 four-subtype Cat III taxonomy:** the derivation rule may apply uniformly to all four concrete Cat III subtypes (`BehavioralCluster`, `AutomationGroup`, `CampaignHypothesis`, `CoordinationRing`) at the abstract `Hypothesis` level, or differ per-subtype. Inherits Layer B's per-subtype-vs-uniform open question; assessed at substantive phases.

## Anti-Patterns to Avoid

Surfaced during framing for committee discipline in subsequent phases:

- **Silent collapse to confidence under simplification pressure.** A candidate whose substantive form, after deliberation, reduces to producing the same value as `confidence` under common conditions is structurally indistinguishable from the §2.6 anti-pattern 2 forbidden form even if its formal definition differs. Detection: candidate's structural form must produce values that diverge from confidence on a structurally-named class of inputs.
- **Operationally-undefined-pattern selection.** A meta-shape 2 resolution that defers the formula without naming the operational-specification document that will produce it leaves Q3 closed but BC1's "structurally falsifiable" status only partially activated. Detection: meta-shape 2 selection must name the operational-specification document or open one in the resolution commit.
- **Procedural-defense overreach.** A meta-shape 3 (ε) resolution that does not name the canonical-serialization-contract validation discipline reduces the dimension to an unvalidated free-form scalar. Detection: ε's resolution must specify the per-record validation surface and the projection-replay diff semantic.

## Migration and Backward Compatibility

No historical Category II construct or Category III hypothesis records exist at this point that carry `evidential_independence`; the dimension was introduced structurally by §2.6 frozen v0.6 at [`§0129`](../../charter/decision-log.md). Q3 resolution applies forward to all subsequent substrate commits.

Pre-§2.6-freeze records do not carry the dimension and are handled per §2.6 anti-pattern 3 (absence-of-dimension is structurally distinct from substituted-value); Q3's derivation rule applies only to records committed under §2.6 freeze and forward. Per the Charter §2.6 anti-pattern 3 explicit text: "consumers must surface the absence, not paper over it."

## References

- [`docs/ontology/ontology.md` Open Question 3](../../ontology/ontology.md) — the question this RFC resolves.
- [Charter §2.6 BC1](../../charter/constitutional-charter.md#26-evidential-independence-integrity) — the deferral this RFC discharges.
- [Charter §2.6 BC2](../../charter/constitutional-charter.md#26-evidential-independence-integrity) — the three meta-shapes Q3's candidate space partitions by.
- [Charter §2.6 forbidden anti-patterns](../../charter/constitutional-charter.md#26-evidential-independence-integrity) — the structural form constraints Q3 candidates must satisfy.
- [Charter §2.4](../../charter/constitutional-charter.md#24-inferential-influence-disclosure) — `influenced_by` chain structural surface; candidate input.
- [Charter §2.3](../../charter/constitutional-charter.md#23-provenance-integrity) — typed `subject_ref_*` chains; candidate input.
- [`decision-log.md` §0011](../../charter/decision-log.md) — Layer B follow-on RFC opening (this RFC's downstream consumer).
- [`decision-log.md` §0020 OMQ #2-C](../../charter/decision-log.md) — decay via §2.5 lifecycle event supersession; Q3 derivation rule is decay-agnostic per this resolution.
- [`decision-log.md` §0021 OMQ #3-α](../../charter/decision-log.md) — substrate-time generation; Q3 derivation rule is substrate-time evaluable.
- [`decision-log.md` §0034](../../charter/decision-log.md) — canonical-serialization-contract enforces paired-dimension requirement.
- [`decision-log.md` §0129](../../charter/decision-log.md) — §2.6 closure; opens Layer B from on-hold to discussion-active and surfaces Q3 as remaining pre-Gate.
- [`decision-log.md` §0132](../../charter/decision-log.md) — opens this RFC at discussion status.
- [`q3-independence-evidence.md`](../discussion/q3-independence-evidence.md) — discussion-phase evidence scratch (Phase 1 + Phase 2).
- [`ontology-revision-layer-b-deep-criterion`](./ontology-revision-layer-b-deep-criterion.md) — downstream RFC Q3 unblocks.

## Decision Record

Resolved at [`decision-log §0133`](../../charter/decision-log.md) (2026-05-22): **Candidate α (source-count ratio over Cat I provenance roots) adopted** under [§2.6 BC2](../../charter/constitutional-charter.md#26-evidential-independence-integrity) meta-shape 1 (deterministic-from-pattern). The formula: `evidential_independence = (count of Cat I primary observation roots in the assertion's subject_ref_* chain NOT reachable via any influenced_by edge from a promoted hypothesis) / (total Cat I roots in the chain)`. Range `[0, 1]`; type rational.

The resolution rests on [`q3-independence-evidence.md`](../discussion/q3-independence-evidence.md) Phase 4 Findings 1 (Tier 1 satisfies §4 criterion 1 structurally), 2 (ε eliminated for admitting §1 Thesis failure mode at substrate), 3 (α dominates β/γ on structural simplicity), 4 (limited resolution is feature at inception phase), 5 (ζ eliminated as over-extension per CLAUDE.md §7 constitutional minimalism).

Two committee extensions: (a) influenced-Cat-II subtraction discipline (α's "not reachable" predicate structurally subtracts Cat I roots whose only path traverses an influenced Cat II intermediate); (b) Q5-transitivity-half cascade fires in the same resolution commit per [`§0015`](../../charter/decision-log.md) + [`§0020`](../../charter/decision-log.md) precedent.

With Q3 resolved, [§2.6 BC1](../../charter/constitutional-charter.md#26-evidential-independence-integrity) "structurally falsifiable" status is activated. Q5 transitivity-half opens at discussion as [`ontology-revision-q5-influence-propagation-transitivity`](./ontology-revision-q5-influence-propagation-transitivity.md). Layer B follow-on RFC remains gated on Q5 transitivity-half resolution per the two-cascade chain Q3 → Q5 → Layer B; canonical-serialization-contract revision opens post-Q5 as structural follow-on per [`§0133`](../../charter/decision-log.md) schedule.
