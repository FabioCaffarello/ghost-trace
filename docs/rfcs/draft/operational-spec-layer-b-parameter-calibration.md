# RFC — Operational specification: Layer B parameter calibration

- **Status:** accepted (resolved at [`decision-log §0138`](../../charter/decision-log.md) adopting medium-zone parameter values + W-count window + U-uniform divergence + N_A=1 day bundled + per-parameter reversal-conditions)
- **Authors:** Ghost Trace committee (per [`decision-log §0135`](../../charter/decision-log.md) Layer B form-vs-parameter discipline + [`§0136`](../../charter/decision-log.md) canonical-serialization-contract crystallization; discussion-phase deliberation Phases 3–5 recorded in [`layer-b-parameter-calibration-evidence.md`](../discussion/layer-b-parameter-calibration-evidence.md); resolution at [`§0138`](../../charter/decision-log.md))
- **Date:** 2026-05-22 (opened); 2026-05-22 (resolved)
- **Type:** operational-spec
- **Affects:** [`docs/architecture/canonical-serialization-contract.md`](../../architecture/canonical-serialization-contract.md) Demotion-Candidacy Predicate section (parameter VALUES; type/range surface unchanged); [`docs/ontology/lifecycle-semantics.md`](../../ontology/lifecycle-semantics.md) §The Promotion Mechanism + §Demotion (operational parameter values become concrete); no Charter prose modification (form-level resolution already at [`§0135`](../../charter/decision-log.md))

> RFCs in Ghost Trace are proposals subject to constitutional review. The [Constitutional Charter](../../charter/constitutional-charter.md) is the authoritative reference. RFCs that conflict with the Charter must either be rejected or accompanied by a corresponding charter amendment (see [`../../charter/amendments.md`](../../charter/amendments.md)). This RFC is operational specification within the form-vs-parameter discipline established at [`§0011`](../../charter/decision-log.md) Methodological Observation 1 — the form is structural and committee-resolved; the values are operational and may be revised at separate RFC under the same form.

## Summary

Per [`decision-log.md` §0135`](../../charter/decision-log.md), Layer B's structural form is **L-BC-OR with L-C structural-exclusion commitment**:

> `Layer B(H) := (freshness_B(H) < T_B) OR (saturation_C(H) > K_C)`

Per [`§0136`](../../charter/decision-log.md), the canonical-serialization-contract crystallizes the form's structural surface but enforces type/range only for the parameters; parameter VALUES are operationally specified. This RFC produces the parameter values at inception phase: `T_B`, `K_C`, `N` (recent-window size), the window structural form, and the per-subtype divergence decision.

This RFC opens structured discussion. Candidate value sets are enumerated in [`layer-b-parameter-calibration-evidence.md`](../discussion/layer-b-parameter-calibration-evidence.md) Phase 2 (five sub-decisions, three to four candidates each). This RFC does not pick values at this phase.

## Motivation

**Why now.** [`§0135`](../../charter/decision-log.md) resolved Layer B's form (L-BC-OR + L-C exclusion); [`§0136`](../../charter/decision-log.md) crystallized the canonical-serialization-contract structural surface. The `LayerBParameters` proto in the contract declares fields for T_B, K_C, N but enforces type/range only. Without operational-specification values, the contract is consumable but does not yet specify behavior. **The next ingestion-service implementation step that touches Layer B firing requires the values.** Opening this RFC at discussion now per the form-vs-parameter discipline + the [`§0135`](../../charter/decision-log.md) follow-on schedule.

**The cost of not resolving.** Implementation work consuming the contract has two options absent this RFC: (a) commit values inline without committee review (which violates form-vs-parameter discipline by elevating implementation choice to constitutional weight); (b) defer Layer B firing entirely (which leaves the demotion-candidacy gate operational only via Layer A). Both are forbidden by [`§0135`](../../charter/decision-log.md) consequences ("Parameter-calibration operational-specification RFC opens as separate follow-on") and [`§0011`](../../charter/decision-log.md) Methodological Observation 1.

## Constitutional Review

The Q1–Q6 impact analysis prescribed by [`rfc-author` §1](../../../.claude/skills/workflow/rfc-author/SKILL.md). Note: this RFC is OPERATIONAL specification, not constitutional or structural; several Q-answers are trivially "no" because the form is committee-resolved elsewhere.

### Q1 — Which Charter invariants does this RFC touch?

Indirectly via parameter values, no Charter prose is modified or referenced operatively:

- **§2.5 Hypothesis Lifecycle Explicitness** (frozen v0.3): Layer B firing produces a `demotion` lifecycle event. This RFC specifies WHEN the predicate fires (parameter values) without modifying §2.5 prose.
- **§2.6 Evidential Independence Integrity** (frozen v0.6): T_B operates on `evidential_independence` values per [`§0133`](../../charter/decision-log.md) Q3-α. This RFC specifies the threshold value; §2.6 paired-dimension commitment is structural and is consumed, not modified.
- **§2.4 Inferential Influence Disclosure** (frozen v0.5): K_C operates on transitive `influenced_by` closures per [`§0134`](../../charter/decision-log.md) Q5-τ. This RFC specifies the threshold value; §2.4 chain declaration is structural and is consumed, not modified.
- **§1 Thesis**: Parameter calibration determines how aggressively the system structurally defends against the two failure modes. Conservative defaults preserve defense surface while bounding false positives at inception phase.

### Q2 — Does this RFC implicitly redefine any term in the glossary?

No. The RFC uses canonical vocabulary (`evidential_independence`, `influenced_by`, `closure_hashes`, `demotion`, `Hypothesis`) without redefinition. Parameter names (`T_B`, `K_C`, `N`) are operational specification terms introduced at [`§0135`](../../charter/decision-log.md) + [`§0136`](../../charter/decision-log.md); not glossary candidates.

### Q3 — Does this RFC implicitly resolve any of the five open Ontology questions?

No. All five original ontology.md Open Questions are resolved as of [`§0134`](../../charter/decision-log.md) Methodological Observation 4. This RFC composes with the resolutions; it does not re-open them.

### Q4 — Does this RFC require Charter amendment?

No. The RFC is operational-specification at the canonical-serialization-contract operational layer; it does NOT modify Charter prose. The form-vs-parameter discipline established at [`§0011`](../../charter/decision-log.md) Methodological Observation 1 explicitly authorizes this separation: form-level RFCs are structural and may require Charter amendment; parameter-value RFCs are operational and do not.

### Q5 — Does this RFC introduce a new invariant?

No. The RFC specifies operational values within an existing structural commitment (L-BC-OR per [`§0135`](../../charter/decision-log.md)). No new Charter invariant.

### Q6 — Does this RFC propose ceremony without behavioral consequence?

No. Parameter values determine when Layer B fires; firing produces `demotion` lifecycle events per [§2.5](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness). Behavioral consequences are direct and material — different value choices produce materially different demotion rates and substrate evolution dynamics. The values are operational; their behavioral consequences are constitutional via [§1 Thesis](../../charter/constitutional-charter.md#1-thesis) defense surface.

## Proposal

Five sub-decisions enumerated in [`layer-b-parameter-calibration-evidence.md`](../discussion/layer-b-parameter-calibration-evidence.md) Phase 2:

| Sub-decision | Candidates |
|---|---|
| 1. `T_B` (freshness threshold) | T_B-loose (0.3) / T_B-medium (0.5) / T_B-strict (0.7) / T_B-derived (REJECTED — violates §0136 parameter-mutability) |
| 2. `K_C` (saturation ratio) | K_C-very-low (0.1) / K_C-low (0.25) / K_C-medium (0.5) / K_C-high (0.75) |
| 3. `N` (window size) | N-small (100) / N-medium (1000) / N-large (10000) |
| 4. Window structural form | W-count (fixed-count) / W-time (fixed-time) / W-hybrid |
| 5. Per-subtype divergence | U-uniform (single set at abstract Hypothesis level) / P-per-subtype (separate sets per concrete subtype) |

This RFC does not pick at this phase. The evidence file records two asymmetries that will likely organize substantive deliberation:

- **Conservative-defaults asymmetry**: medium-zone candidates (T_B-medium, K_C-medium, N-medium, W-count, U-uniform) align with [`§0022`](../../charter/decision-log.md) empirical-pressure-phase + [`§0023`](../../charter/decision-log.md) inception-phase posture. Aggressive variants require empirical justification not currently available.
- **Form-vs-parameter respect asymmetry**: T_B-derived candidate (runtime-derived threshold) violates [`§0136`](../../charter/decision-log.md) parameter-mutability prohibition — structurally precluded for runtime mutation; can only be applied at operational-specification revision time.

## Alternatives Considered

Out-of-scope candidate forms surfaced and explicitly rejected during framing:

- **Runtime-configurable parameter values.** Rejected by [`§0136`](../../charter/decision-log.md) parameter-mutability prohibition — parameter values are structural constants at the canonical-serialization-contract layer, not operator-configurable at runtime. Operational-specification revision is the legitimate path for value changes.
- **Layer B firing without Layer A composition.** Rejected by [`§0011`](../../charter/decision-log.md) outer AND-composition. Layer A `N_A` cadence parameter is acknowledged as a sibling operational parameter; bundling decision is a sub-question of this RFC.
- **Per-record parameter overrides.** Rejected by structural-uniformity discipline at the contract layer; parameters are abstract-Hypothesis-level OR concrete-subtype-level (the per-subtype divergence sub-decision), never per-individual-record.
- **Empirically-driven calibration at inception.** Rejected by [`§0022`](../../charter/decision-log.md) empirical-pressure-phase discipline + [`§0023`](../../charter/decision-log.md) inception-phase posture — NO empirical substrate observations exist at inception; calibration is committee-judgment-over-conservative-defaults, not observation-driven.

## Open Questions

The RFC's own open questions to be resolved when it advances substantively:

- **Which T_B candidate?** T_B-loose / T_B-medium / T_B-strict. Load-bearing sub-decision.
- **Which K_C candidate?** K_C-very-low / K_C-low / K_C-medium / K_C-high. Load-bearing sub-decision.
- **Which N candidate?** N-small / N-medium / N-large. Load-bearing sub-decision.
- **Which window structural form?** W-count / W-time / W-hybrid. Structural form choice.
- **Uniform or per-subtype?** U-uniform / P-per-subtype. Per-subtype divergence question.
- **N_A bundling decision.** Layer A's `N_A` cadence parameter (per [`§0011`](../../charter/decision-log.md)) is operationally adjacent. The resolution either bundles N_A into this RFC's scope OR defers it to a sibling RFC. Sub-decision the resolution addresses.
- **Reversal-conditions record granularity.** The empirical-pressure-phase triggers per [`§0022`](../../charter/decision-log.md) may be enumerated at per-parameter granularity OR at parameter-set granularity. Sub-decision the resolution addresses.

## Anti-Patterns to Avoid

Surfaced during framing for committee discipline in subsequent phases:

- **Optimization theater.** Picking values based on aesthetic preference or hypothetical scenarios without grounding in conservative-defaults posture. The inception-phase calibration is committee-judgment-over-conservative-defaults per [`§0022`](../../charter/decision-log.md); any value committed must have explicit rationale grounded in §1 Thesis defense or false-positive minimization.
- **False precision.** Committing parameter values with implied precision the inception-phase posture does not support. E.g., `T_B = 0.47` would suggest empirical calibration the substrate cannot yet support; `T_B = 0.5` is honest about the committee-judgment nature.
- **Empirical-pressure deferral without trigger record.** Committing inception defaults without explicitly naming the empirical-pressure-phase reversal triggers per [`§0022`](../../charter/decision-log.md). The reversal-conditions record IS the discipline that justifies the conservative-defaults posture; absent it, the committee defaults become "permanent" by procedural inertia.
- **Per-subtype premature optimization.** Adopting per-subtype divergence (P-per-subtype) without empirical evidence that subtypes' behavior diverges meaningfully. [CLAUDE.md §7](../../../.claude/CLAUDE.md) constitutional minimalism forbids premature optimization at this phase.
- **N_A bundling silence.** Failing to name the N_A bundling decision in the resolution — either bundle Layer A's cadence parameter explicitly OR defer to a named sibling RFC. Silent treatment leaves operational ambiguity.

## Migration and Backward Compatibility

No historical Cat III hypothesis records carry promotion events at this point. The Layer B parameter calibration is forward-looking; values apply to hypothesis-promotion events committed under the calibrated parameters. Subsequent operational-specification revisions per [`§0022`](../../charter/decision-log.md) reversal-conditions are themselves separate RFCs; backward compatibility is the resolution's structural-form preservation, not value preservation.

## References

- [`docs/architecture/canonical-serialization-contract.md`](../../architecture/canonical-serialization-contract.md) — §0136 contract revision; LayerBParameters proto + parameter-mutability prohibition.
- [Charter §1 Thesis](../../charter/constitutional-charter.md#1-thesis) — failure modes Layer B parameter calibration preserves defense against.
- [Charter §2.4](../../charter/constitutional-charter.md#24-inferential-influence-disclosure) v0.5 — `influenced_by` chain structural surface K_C operates on.
- [Charter §2.5](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) v0.3 — `demotion` lifecycle event Layer B firing produces.
- [Charter §2.6](../../charter/constitutional-charter.md#26-evidential-independence-integrity) v0.6 (patched v0.7.1) — `evidential_independence` paired-dimension; T_B operates on α values.
- [`decision-log.md` §0010](../../charter/decision-log.md) — Q2-A.2 four-subtype Cat III taxonomy (per-subtype divergence question).
- [`decision-log.md` §0011](../../charter/decision-log.md) — Q4 staged-combination form; form-vs-parameter discipline established.
- [`decision-log.md` §0022](../../charter/decision-log.md) — empirical-pressure-phase discipline; inception-phase posture.
- [`decision-log.md` §0023](../../charter/decision-log.md) — inception-phase single-tier `actor_ref` precedent.
- [`decision-log.md` §0133](../../charter/decision-log.md) — Q3-α resolution.
- [`decision-log.md` §0134](../../charter/decision-log.md) — Q5-τ + β-graph resolution.
- [`decision-log.md` §0135](../../charter/decision-log.md) — Layer B L-BC-OR resolution; parameter-calibration follow-on opening.
- [`decision-log.md` §0136](../../charter/decision-log.md) — canonical-serialization-contract revision; LayerBParameters proto.
- [`layer-b-parameter-calibration-evidence.md`](../discussion/layer-b-parameter-calibration-evidence.md) — discussion-phase evidence scratch (Phase 1 + Phase 2).
- [`layer-b-deep-criterion-evidence.md`](../discussion/layer-b-deep-criterion-evidence.md) — upstream Layer B form deliberation.
- [`q3-independence-evidence.md`](../discussion/q3-independence-evidence.md) — Q3 discussion phase (precedent structure).
- [`q5-transitivity-evidence.md`](../discussion/q5-transitivity-evidence.md) — Q5 discussion phase (precedent structure).
- [CLAUDE.md §7](../../../.claude/CLAUDE.md) — constitutional minimalism (relevant to per-subtype divergence anti-pattern).

## Decision Record

Resolved at [`decision-log §0138`](../../charter/decision-log.md) (2026-05-22): inception-phase parameter values adopted per Phase 5 recommendation.

| Sub-decision | Resolved value |
|---|---|
| `T_B` | **`{numerator: 1, denominator: 2}`** (0.5) |
| `K_C` | **`{numerator: 1, denominator: 2}`** (0.5) |
| `N` | **1000** |
| Window form | **W-count** (fixed-count: last N assertions by substrate-commit order) |
| Per-subtype divergence | **U-uniform** (single parameter set at abstract `Hypothesis` level) |
| N_A bundling | **Bundled** in this RFC |
| `N_A` (Layer A cadence) | **`n_a_duration_nanoseconds: 86400000000000`** (1 day) |
| Reversal-conditions granularity | **Per-parameter** |

Full demotion-candidacy predicate under this resolution:

> `DEMOTE-CANDIDATE(H) := (elapsed_time_since_H.promotion > 1 day) AND ((freshness_B(H) < 0.5) OR (saturation_C(H) > 0.5))`

The resolution rests on [`layer-b-parameter-calibration-evidence.md`](../discussion/layer-b-parameter-calibration-evidence.md) Phase 4 findings F1–F8.

LayerBParameters proto extension committed at the canonical-serialization-contract revision: `n_a_duration_nanoseconds` field added at field number 4. Per-parameter reversal-conditions record committed with the resolution names empirical-pressure-phase triggers per [`§0022`](../../charter/decision-log.md) discipline.

With this resolution, the full Q4 → Layer B operational arc ([`§0011`](../../charter/decision-log.md) → [`§0099`](../../charter/decision-log.md) → [`§0129`](../../charter/decision-log.md) → [`§0133`](../../charter/decision-log.md) → [`§0134`](../../charter/decision-log.md) → [`§0135`](../../charter/decision-log.md) → [`§0136`](../../charter/decision-log.md) → [`§0138`](../../charter/decision-log.md)) is structurally complete at the operational-specification layer; service-tier Layer B firing implementation can now adopt the demotion-candidacy predicate as ordinary RFC discipline.
