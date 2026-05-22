# Layer B Parameter Calibration — operational-specification discussion evidence

**Status:** in-discussion. Not authoritative. Final resolution will be recorded in decision-log and operational-spec document revision.

This scratch supports the discussion phase of [`operational-spec-layer-b-parameter-calibration`](../draft/operational-spec-layer-b-parameter-calibration.md). The RFC opens per [`§0135`](../../charter/decision-log.md) Consequences ("Parameter-calibration operational-specification RFC opens as separate follow-on") + [`§0136`](../../charter/decision-log.md) carry-forward ("Parameter-calibration operational-specification RFC opens as separate follow-on per the form-vs-parameter discipline").

The form-level resolution (Layer B = L-BC-OR with L-C structural-exclusion commitment) lands at [`§0135`](../../charter/decision-log.md); the canonical-serialization-contract crystallization (α + τ + β-graph + L-BC-OR firing predicate) lands at [`§0136`](../../charter/decision-log.md). This RFC specifies the operational parameter values + window structural form + per-subtype divergence under [`§0010`](../../charter/decision-log.md) Q2-A.2 — the residual operational decisions deferred by the form-vs-parameter discipline established at [`§0011`](../../charter/decision-log.md) Methodological Observation 1.

This is a strictly-framing scratch: Phase 1 names the dependency surface and the inception-phase posture; Phase 2 enumerates parameter-by-parameter candidate values. Phases 3+ (epistemic-skill application, comparison synthesis, recommendation) are drafted in subsequent commits.

---

## Phase 1 — Scope and dependencies

### The question

Per [`§0135`](../../charter/decision-log.md) Decision + [`§0136`](../../charter/decision-log.md) Decision, the full demotion-candidacy predicate at the canonical-serialization-contract layer is:

> `DEMOTE-CANDIDATE(H) := Layer A(H) AND ((freshness_B(H) < T_B) OR (saturation_C(H) > K_C))`

where the following parameters are operationally specified (not structurally fixed):

1. **`T_B`** — B-family freshness threshold; rational in `[0, 1]`.
2. **`K_C`** — C-family saturation ratio threshold; rational in `[0, 1]`.
3. **`N`** — recent-assertions window size; positive integer.
4. **`N_A`** (Layer A) — cadence parameter; positive duration. Tracked separately at [`§0011`](../../charter/decision-log.md) Q4 resolution; not the substantive focus of this RFC but acknowledged.
5. **Window structural form** — fixed-count (window of last N assertions), fixed-time (window of last T duration), or hybrid.
6. **Per-subtype divergence** — uniform parameters at the abstract `Hypothesis` level, or per-concrete-subtype divergence under [`§0010`](../../charter/decision-log.md) Q2-A.2.

This RFC's substantive content addresses items 1, 2, 3, 5, and 6. Item 4 (`N_A`) is forward-referenced to the same operational-specification surface (cadence is itself a parameter-calibration question; bundling N_A here vs separating it is a sub-decision the resolution addresses).

### In scope

- Specific VALUES for T_B, K_C, N at inception phase.
- The structural form of the "recent assertions" window.
- The uniform-vs-per-subtype decision for the parameter set.
- The reversal-conditions record naming the empirical-pressure-phase triggers per [`§0022`](../../charter/decision-log.md) discipline.
- Whether to bundle Layer A's N_A parameter into this RFC or address it separately.

### Out of scope

- **Layer B form.** Resolved at [`§0135`](../../charter/decision-log.md): L-BC-OR + L-C structural-exclusion commitment. This RFC does NOT re-open the form question.
- **Canonical-serialization-contract structural surface.** Crystallized at [`§0136`](../../charter/decision-log.md). This RFC consumes the contract's structural commitments; it does NOT modify them.
- **α formula / τ semantics / β-graph storage.** All resolved at [`§0133`](../../charter/decision-log.md) + [`§0134`](../../charter/decision-log.md). This RFC consumes them; it does NOT modify them.
- **Implementation-level concerns.** Service-tier code adoption of the parameter values is downstream RFC discipline.

### Resolved dependencies (structural ground present)

| Anchor | What it commits | How this RFC consumes it |
|---|---|---|
| [`§0011`](../../charter/decision-log.md) | Layer A AND Layer B staged-combination | This RFC works within the staged-combination form; N_A bundling decision per §0011 form-vs-parameter discipline |
| [`§0022`](../../charter/decision-log.md) | Empirical-pressure-phase discipline | Inception-phase defaults committed conservatively; reversal-conditions record names empirical-pressure triggers |
| [`§0133`](../../charter/decision-log.md) | Q3-α source-count ratio | T_B operates on α values; rational comparison via rational pair encoding per [`§0136`](../../charter/decision-log.md) |
| [`§0134`](../../charter/decision-log.md) | Q5-τ transitive closure + β-graph storage | K_C operates on closure_hashes membership counts; closure already computed at write time |
| [`§0135`](../../charter/decision-log.md) | L-BC-OR + L-C structural-exclusion | Layer B firing predicate's form is unchanged; parameters specified within the form |
| [`§0136`](../../charter/decision-log.md) | Canonical-serialization-contract crystallizes α + τ + L-BC-OR | Parameter values land at the `LayerBParameters` proto message; contract enforces type/range; values are operational |
| [`§0010`](../../charter/decision-log.md) Q2-A.2 | Four concrete Cat III subtypes under abstract `Hypothesis` | Per-subtype divergence is the structural option this RFC addresses |
| [`§0023`](../../charter/decision-log.md) | Inception-phase posture | Conservative defaults align; per-subtype divergence deferred unless empirical pressure surfaces |

### Open dependencies (not blocking; assessed at substantive deliberation)

| Open question | Why relevant | Default disposition |
|---|---|---|
| Empirical substrate observations — α distribution, closure-membership counts, hypothesis promotion/demotion rates | Optimal parameter values depend on substrate characteristics | At inception phase, NO empirical observations exist per [`§0022`](../../charter/decision-log.md); inception defaults are committed conservatively with reversal triggers |
| Layer A `N_A` parameter | Layer A cadence affects Layer B firing frequency (Layer A AND Layer B) | Bundling decision is itself a sub-decision; this RFC may resolve N_A within scope or defer to a sibling RFC |

### Inception-phase posture

Per [`§0022`](../../charter/decision-log.md) empirical-pressure-phase discipline + [`§0023`](../../charter/decision-log.md) inception-phase single-tier `actor_ref` + [`§0027`](../../charter/decision-log.md) inception-phase SQLite + content-addressed blob-store: the system is at inception phase. NO empirical substrate observations exist yet for hypothesis promotion/demotion rates, α distributions, or closure-membership statistics. Parameter calibration at this phase is **conservative-defaults-with-explicit-reversal-triggers**: commit values that minimize false positives while preserving §1 Thesis defense surface, name the empirical-pressure conditions under which the defaults are revisited.

This posture is structurally distinct from "empirically-calibrated values" — at inception, the calibration is *committee-judgment over conservative-defaults* rather than *observation-driven optimization*. The structural commitment is to revisit the defaults once empirical pressure surfaces per [`§0022`](../../charter/decision-log.md) discipline.

### Procedural posture

This RFC is at `discussion` status. Phase 1 (this section) names the dependency surface + inception-phase posture. Phase 2 (below) enumerates parameter-by-parameter candidate values. Phase 3+ (epistemic-skill application, comparison synthesis, recommendation) drafted in subsequent commits per the Q3 / Q5 / Layer B cadence precedent. Resolution lands at a future `decision-log` entry that closes parameter values.

---

## Phase 2 — Parameter-by-parameter candidate enumeration

Five sub-decisions, each with multiple candidates. No selection at this phase.

### Sub-decision 1: T_B (B-family freshness threshold)

`T_B ∈ [0, 1]`. Layer B fires when `freshness_B(H) < T_B`, i.e., the average `evidential_independence` over recent assertions referencing H falls below T_B.

#### Candidate T_B-loose — T_B = 0.3

**Rationale:** Fires only when the recent supporting evidence is severely depleted — average α below 0.3 means the typical recent assertion is reachable via influenced_by from a promoted hypothesis for ≥70% of its Cat I roots. This is a strong indicator of evidence staleness.

**Pros:** Low false-positive rate at inception; demotion-candidacy fires only on clear-cut staleness; aligns with conservative-defaults posture.

**Cons:** May miss moderate staleness; a hypothesis with α-trend declining toward 0.4 persists longer than ideal.

#### Candidate T_B-medium — T_B = 0.5

**Rationale:** Fires when majority of recent supporting evidence is influenced — average α below 0.5 means more than half of Cat I roots in recent assertions are reachable via influenced_by. Aligns with the §1 Thesis naming of "confidence inflates without proportional independent evidence" — when independence drops below the midpoint, the failure mode is unambiguous.

**Pros:** Catches moderate staleness; midpoint is a natural structural threshold; matches Thesis half (a) framing.

**Cons:** Higher false-positive rate than T_B-loose; may demote productive hypotheses temporarily off-cycle.

#### Candidate T_B-strict — T_B = 0.7

**Rationale:** Fires when minority of recent supporting evidence is independent — average α below 0.7 means more than 30% of Cat I roots are reachable via influenced_by. Aggressive demotion-candidacy.

**Pros:** Catches early-stage staleness; protects §1 Thesis surface strongly.

**Cons:** High false-positive rate; many hypotheses fire on Layer B even when productive; effectively shortens promotion lifetime substantially.

#### Candidate T_B-derived — T_B computed from initial substrate statistics

**Rationale:** Rather than committing a fixed value, T_B is derived at startup or periodically from the substrate's actual α-distribution — e.g., T_B = mean(α) - k*stddev(α) for some k.

**Pros:** Adapts to substrate; principled empirical calibration.

**Cons:** Introduces a runtime-computed parameter that violates [`§0136`](../../charter/decision-log.md) parameter-mutability prohibition (T_B is a structural constant at the canonical-serialization-contract layer, not operator-configurable); rejected at the substrate layer; can be applied only at operational-specification revision time, not runtime.

### Sub-decision 2: K_C (C-family saturation ratio threshold)

`K_C ∈ [0, 1]`. Layer B fires when `saturation_C(H) > K_C`, i.e., the fraction of recent assertions with H in `closure_hashes` (excluding H's own direct enrichment outputs per L-C exclusion) exceeds K_C.

#### Candidate K_C-very-low — K_C = 0.1

**Rationale:** Fires when H transitively influences ≥10% of recent assertions (excluding H's direct outputs). Aggressive saturation detection.

**Pros:** Catches early-stage saturation; protects §1 Thesis half (b) strongly.

**Cons:** High false-positive rate; productive hypotheses with broad transitive influence trees fire frequently.

#### Candidate K_C-low — K_C = 0.25

**Rationale:** Quartile-saturation threshold. Fires when H influences ≥25% of recent assertions.

**Pros:** Catches moderate saturation; quartile is a natural empirical threshold.

**Cons:** May miss high-but-not-dominant saturation patterns.

#### Candidate K_C-medium — K_C = 0.5

**Rationale:** Fires when H influences majority of recent assertions (>50%). Matches the §1 Thesis "promoted hypotheses re-enter as enrichment and silently reinforce themselves" framing — when more than half of recent activity is downstream of H, H has become a fixed point of inferential gravity.

**Pros:** Matches Thesis framing; midpoint is a natural structural threshold; conservative-defaults posture.

**Cons:** Hypotheses with substantial-but-not-majority transitive influence persist longer than ideal.

#### Candidate K_C-high — K_C = 0.75

**Rationale:** Fires only on near-overwhelming saturation (>75%). Very conservative.

**Pros:** Almost no false positives; only demotes hypotheses that clearly dominate substrate.

**Cons:** Catches saturation very late — the §1 Thesis failure mode is well-developed before Layer B fires.

### Sub-decision 3: N (recent-assertions window size)

`N` is a positive integer (under fixed-count window form) or a duration (under fixed-time window form).

#### Candidate N-small — N = 100 assertions

**Rationale:** Small window. Layer B fires quickly when recent trend turns adverse.

**Pros:** Reactive; demotion-candidacy fires within ~100 assertion-cycles of staleness/saturation onset.

**Cons:** Statistically thin; small window admits high variance; spurious firings on transient distribution shifts.

#### Candidate N-medium — N = 1000 assertions

**Rationale:** Medium window. Balances reactivity with statistical robustness.

**Pros:** Statistically meaningful; moderate reactivity; matches typical inception-phase substrate scale per [`§0027`](../../charter/decision-log.md).

**Cons:** Slower reaction time than N-small; may persist staleness/saturation for ~1000 cycles before firing.

#### Candidate N-large — N = 10000 assertions

**Rationale:** Large window. Maximum statistical robustness.

**Pros:** Stable; demotion-candidacy fires only on persistent trends.

**Cons:** Very slow reaction; may take substantial time before Layer B catches a clear failure mode.

### Sub-decision 4: Window structural form

#### Candidate W-count — Fixed-count (last N assertions)

**Rationale:** The window is the most recent N assertions, regardless of when they were committed.

**Pros:** Simple; statistically uniform; matches the rational comparison structure α uses per [`§0136`](../../charter/decision-log.md); no temporal-arithmetic semantics needed.

**Cons:** Insensitive to time — a hypothesis with N recent assertions over 10 years vs over 1 day are treated identically.

#### Candidate W-time — Fixed-time (last T duration)

**Rationale:** The window is all assertions committed in the last T duration.

**Pros:** Time-aware; temporal patterns (e.g., burst activity) are honored.

**Cons:** Statistically variable — window size varies; high-throughput periods produce many-N windows, quiet periods produce few-N windows. Substantially harder to calibrate. Introduces clock-time semantics that interact awkwardly with substrate immutability per [§2.1](../../charter/constitutional-charter.md#21-observational-integrity).

#### Candidate W-hybrid — Fixed-count OR fixed-time, whichever produces a larger window

**Rationale:** Window is max(last N assertions, all assertions in last T duration). Ensures statistical floor + temporal coverage.

**Pros:** Robust to both dimensions; floors prevent both undersized and overly-old windows.

**Cons:** Most complex; two parameters (N + T); calibration is harder.

### Sub-decision 5: Per-subtype divergence

Per [`§0010`](../../charter/decision-log.md) Q2-A.2, four concrete Cat III subtypes exist: `BehavioralCluster`, `AutomationGroup`, `CampaignHypothesis`, `CoordinationRing`. Parameter sets may be uniform at the abstract `Hypothesis` level or per-subtype.

#### Candidate U-uniform — Single parameter set at abstract Hypothesis level

**Rationale:** All four concrete subtypes share T_B, K_C, N, window form.

**Pros:** Simplest; matches inception-phase posture; no per-subtype empirical calibration needed; aligns with [Q3 Phase 4 Finding 9](./q3-independence-evidence.md) + [Q5 Phase 4 Finding 7](./q5-transitivity-evidence.md) + [Layer-B Phase 4 Finding 8](./layer-b-deep-criterion-evidence.md) consistent "uniform at inception" pattern.

**Cons:** May miss subtype-specific dynamics (e.g., `CampaignHypothesis` may have natively different saturation patterns than `BehavioralCluster`).

#### Candidate P-per-subtype — Separate parameter set per concrete subtype

**Rationale:** Each subtype gets its own T_B, K_C, N, window form.

**Pros:** Maximally adaptable; subtype-specific empirical calibration possible.

**Cons:** 4× the calibration surface; no inception-phase empirical evidence supports the divergence; premature optimization per [CLAUDE.md §7](../../../.claude/CLAUDE.md) constitutional minimalism.

### Asymmetries surfaced

Two cross-sub-decision asymmetries will likely organize substantive deliberation:

- **Conservative-defaults asymmetry:** loose/medium T_B + medium/high K_C + medium N + W-count + U-uniform align with the [`§0022`](../../charter/decision-log.md) empirical-pressure-phase + [`§0023`](../../charter/decision-log.md) inception-phase posture. Aggressive variants (strict T_B, low K_C, large N, W-time/W-hybrid, P-per-subtype) require empirical justification not currently available.
- **Form-vs-parameter respect asymmetry:** T_B-derived violates the [`§0136`](../../charter/decision-log.md) parameter-mutability prohibition (parameters are structural at the canonical-serialization-contract layer, not runtime-configurable). The candidate is structurally precluded for runtime mutation; can only be applied at operational-specification revision time.

---

## Phase 3 — Apply epistemic skills

Per the Q3 / Q5 / Layer B Phase 3 precedent, three skills ([`falsifiability-check`](../../../.claude/skills/epistemic/falsifiability-check/SKILL.md), [`epistemic-separator`](../../../.claude/skills/epistemic/epistemic-separator/SKILL.md), [`ambiguity-reducer`](../../../.claude/skills/epistemic/ambiguity-reducer/SKILL.md)) applied. Operational-spec RFCs admit a lighter Phase 3 than structural RFCs because the structural form is fixed at [`§0135`](../../charter/decision-log.md) and the canonical-serialization-contract enforces type/range at the marshalling boundary per [`§0136`](../../charter/decision-log.md). The epistemic-skill questions therefore focus on the candidate's posture-fit at inception phase rather than on structural admissibility.

### Sub-decision 1 — T_B (freshness threshold)

| Candidate | `falsifiability-check` | `epistemic-separator` | `ambiguity-reducer` |
|---|---|---|---|
| **T_B-loose (0.3)** | Pass at form-level. The threshold is structurally observable (firing predicate per [`§0136`](../../charter/decision-log.md)); a hypothesis with freshness_B(H) < 0.3 not in demotion-candidate set is detectable via projection-replay. Falsifiability is independent of value choice. | Pass per category boundary. α values per [`§0133`](../../charter/decision-log.md) operate at the Cat I root layer; no cross-category mixing introduced by threshold value. | Pass. `0.3` is a deterministic rational value; no ambiguity. |
| **T_B-medium (0.5)** | Same form-level pass as T_B-loose. | Same as T_B-loose. | Same as T_B-loose. Additionally: the midpoint matches §1 Thesis half (a) framing — "majority of recent supporting evidence is influenced" is structurally meaningful at the 0.5 boundary. |
| **T_B-strict (0.7)** | Same form-level pass. | Same as T_B-loose. | Pass. |
| **T_B-derived (rejected)** | FAIL — falsifiability at the canonical-serialization-contract layer requires the value to be a structural constant per [`§0136`](../../charter/decision-log.md) parameter-mutability prohibition. A runtime-derived value is not contract-falsifiable; projection-replay byte-for-byte match against a moving target is structurally undefined. | Fail. Runtime derivation introduces a cross-layer dependency (substrate → runtime metric → substrate) that the contract layer cannot enforce. | N/A — rejected at falsifiability + epistemic-separator. |

### Sub-decision 2 — K_C (saturation ratio)

| Candidate | `falsifiability-check` | `epistemic-separator` | `ambiguity-reducer` |
|---|---|---|---|
| **K_C-very-low (0.1)** | Pass at form-level. | Pass. saturation_C reads closure_hashes per [`§0134`](../../charter/decision-log.md); the structural exclusion of H's direct enrichment outputs is the L-C commitment per [`§0135`](../../charter/decision-log.md); both invariant under value choice. | Pass. |
| **K_C-low (0.25)** | Pass. | Pass. | Pass. Quartile boundary is a natural empirical threshold but does not encode constitutional meaning. |
| **K_C-medium (0.5)** | Pass. | Pass. | Pass. Midpoint matches §1 Thesis half (b) framing — "promoted hypotheses re-enter as enrichment and silently reinforce themselves" is structurally meaningful at the majority boundary. |
| **K_C-high (0.75)** | Pass. | Pass. | Pass. |

### Sub-decision 3 — N (window size)

| Candidate | `falsifiability-check` | `epistemic-separator` | `ambiguity-reducer` |
|---|---|---|---|
| **N-small (100)** | Pass at form-level. Window-size choice does not affect structural falsifiability. | Pass. Window scope is the recent-assertion set; no cross-category mixing. | Pass. `100` is deterministic. |
| **N-medium (1000)** | Pass. | Pass. | Pass. |
| **N-large (10000)** | Pass. | Pass. | Pass. |

### Sub-decision 4 — Window structural form

| Candidate | `falsifiability-check` | `epistemic-separator` | `ambiguity-reducer` |
|---|---|---|---|
| **W-count (fixed-count)** | Pass at form-level. The window is deterministically the last N assertions per substrate-commit order; falsifiability via projection-replay over the same set. | Pass. No clock-time semantics; no §2.1 substrate-immutability interaction. | Pass. Window semantics are deterministic. |
| **W-time (fixed-time)** | Pass-with-concern. Window is the assertions committed in the last T duration; falsifiability requires deterministic clock-time semantics at projection-replay. **Risk:** clock-time interpretation across replay environments may diverge (timezone, NTP drift); the contract would need to specify clock-time semantics explicitly. | **Risk:** clock-time introduces a temporal-arithmetic dependency that interacts with [§2.1](../../charter/constitutional-charter.md#21-observational-integrity) substrate-immutability. The substrate's content-addressable identifier per [`§0021`](../../charter/decision-log.md) does not include clock-time; introducing clock-time-dependent windows creates a parallel temporal axis the substrate does not natively carry. | **Ambiguity-flagged on `T` operationalization.** Window duration's semantic (wall-clock vs substrate-commit time) is itself a structural sub-decision — Response 3 (raise as open modeling question). |
| **W-hybrid (max of count + time)** | Pass-with-concern (inherits W-time concerns). | Inherits W-time concerns. | Inherits W-time ambiguities; additionally introduces two-parameter calibration (N + T). |

### Sub-decision 5 — Per-subtype divergence

| Candidate | `falsifiability-check` | `epistemic-separator` | `ambiguity-reducer` |
|---|---|---|---|
| **U-uniform** | Pass at form-level. Single parameter set is contract-structural per [`§0136`](../../charter/decision-log.md) LayerBParameters proto; falsifiability via single value comparison. | Pass. Uniform parameters operate at the abstract `Hypothesis` level; no per-subtype category-boundary commitment introduced. | Pass. |
| **P-per-subtype** | Pass at form-level. Per-subtype parameters extend the LayerBParameters proto with subtype-indexed fields. | **Risk:** per-subtype divergence at inception phase requires per-subtype rationale not currently available; without empirical evidence, the divergence introduces structural-asymmetry across the four concrete subtypes that the abstract `Hypothesis` type's uniform lifecycle (per [`§0010`](../../charter/decision-log.md) Q2-A.2) does not motivate. | **Ambiguity-flagged on per-subtype rationale.** Each subtype's parameter set requires justification; without justification, the values are operationally arbitrary. Response 3 — raise as open modeling question for empirical-pressure-phase reversal. |

### Most consequential epistemic finding across the matrix

**Primary finding — W-time / W-hybrid carry clock-time semantic risk that W-count avoids.** Per Phase 3 sub-decision 4: W-time and W-hybrid introduce a temporal-arithmetic dependency that interacts with §2.1 substrate-immutability (the substrate's content-addressable identifier does not include clock-time per [`§0021`](../../charter/decision-log.md); introducing clock-time-dependent windows creates a parallel temporal axis). W-count uses substrate-commit-order, which is the substrate's native ordering per [`§0024`](../../charter/decision-log.md) + [`§0027`](../../charter/decision-log.md). **W-count is the only candidate without a structural risk surface.**

**Secondary finding — P-per-subtype carries an empirical-justification burden U-uniform does not.** Per Phase 3 sub-decision 5: per-subtype divergence at inception phase requires per-subtype rationale not available. The candidate is not structurally rejected (the LayerBParameters proto admits per-subtype extension via additional fields), but the operational discipline (rationale-per-subtype) is unmet at inception. U-uniform inherits the Q3 / Q5 / Layer B precedent of uniform-at-inception per their respective Phase 4 findings.

**Tertiary finding — T_B-derived is structurally precluded, not just operationally suboptimal.** Per Phase 3 sub-decision 1: T_B-derived violates [`§0136`](../../charter/decision-log.md) parameter-mutability prohibition. The candidate is not a value choice; it's a structural-layer violation. The committee's rejection at framing per the form-vs-parameter respect asymmetry is upheld at Phase 3.

### Calibration carry-forward to future operational-spec RFCs

Layer-B-parameter-calibration-1 confirms and extends prior calibrations:

- **Confirmed: falsifiability §1.3 (operationalization) does most of the work on substrate-touching propositions.** All operational candidates decide cleanly at §1.3; the value choices are independent of structural falsifiability, which lands at the form layer.
- **Confirmed: ambiguity-reducer surfaces residual carry-forwards that are themselves structural deferrals.** W-time / W-hybrid surface the clock-time operationalization as Response-3.
- **New observation — Operational-spec Phase 3 is lighter than ontology-revision Phase 3.** The structural form is fixed; the epistemic-skill matrix focuses on posture-fit at inception phase rather than on structural admissibility. **Future operational-spec RFCs may apply this lighter Phase 3 pattern.**

## Phase 4 — Comparison synthesis

Findings synthesized from Phase 1 (dependency surface + inception-phase posture), Phase 2 (parameter-by-parameter candidate enumeration), and Phase 3 (epistemic-skill matrix). Classified as **asymmetry** / **apparent trade-off that resolves** / **genuine trade-off** / **tension**. Numbered in order of consequence.

### Finding 1 — Asymmetry: inception-phase posture favors medium-zone candidates across all parameter sub-decisions

Per Phase 1's inception-phase posture commitment + [`§0022`](../../charter/decision-log.md) empirical-pressure-phase discipline + [`§0023`](../../charter/decision-log.md) inception-phase precedent: the system has NO empirical observations of α distributions, closure-membership counts, or hypothesis lifecycle dynamics. Parameter calibration is **committee-judgment-over-conservative-defaults**, not observation-driven. Medium-zone candidates (T_B-medium = 0.5, K_C-medium = 0.5, N-medium = 1000) minimize the asymmetry between false-positive and false-negative rates — both extremes (T_B-strict / K_C-very-low / large N for aggressive; T_B-loose / K_C-high / small N for permissive) require empirical justification not available. **Medium-zone is the structurally-disciplined inception-phase default.**

### Finding 2 — Asymmetry: T_B-medium and K_C-medium align with §1 Thesis framing

Per Phase 3 sub-decision 1 (T_B-medium) + sub-decision 2 (K_C-medium): the midpoint thresholds match the §1 Thesis framing of the two failure modes. T_B = 0.5 fires when "majority of recent supporting evidence is influenced" (Thesis half (a)); K_C = 0.5 fires when "promoted hypotheses influence majority of recent assertions" (Thesis half (b)). The midpoints are not arbitrary defaults — they encode the structural-meaningfulness of the boundary in [§1 Thesis](../../charter/constitutional-charter.md#1-thesis) terms. **Defaulting to medium-zone in this case carries constitutional alignment beyond mere conservatism.**

### Finding 3 — Apparent trade-off that resolves: W-time / W-hybrid clock-time concerns eliminate them at inception

Per Phase 3 sub-decision 4: W-time and W-hybrid introduce a clock-time semantic risk (parallel temporal axis the substrate does not natively carry per [`§0021`](../../charter/decision-log.md)). The apparent benefit (time-awareness, temporal pattern handling) does not survive the inception-phase posture — at inception, the substrate's commit-order is the only temporal axis structurally defined. **W-count is the only candidate without a structural risk surface; the trade-off resolves toward W-count at inception, with W-time / W-hybrid available as empirical-pressure-phase reversal options if temporal patterns matter operationally.**

### Finding 4 — Asymmetry: P-per-subtype carries empirical-justification burden U-uniform does not

Per Phase 3 sub-decision 5 + [CLAUDE.md §7](../../../.claude/CLAUDE.md) constitutional minimalism: per-subtype divergence at inception phase requires per-subtype rationale not available. The four concrete Cat III subtypes (per [`§0010`](../../charter/decision-log.md) Q2-A.2) inherit the abstract `Hypothesis` lifecycle uniformly; per-subtype parameter sets would impose structural asymmetry without empirical justification. **U-uniform inherits the established inception-phase pattern from Q3 / Q5 / Layer B Phase 4 Findings 9 / 7 / 8 respectively.**

### Finding 5 — Genuine trade-off: N-medium vs N-small/N-large requires committee-judgment on reactivity-vs-stability balance

Per Phase 2 sub-decision 3: N-small (100) is more reactive but statistically thin; N-large (10000) is more stable but slow to react; N-medium (1000) balances. At inception, there's no empirical guidance on the right balance — it's a values judgment about whether faster reaction or higher confidence matters more. **Defaulting to N-medium per the conservative-defaults posture is committee-judgment, with reversal-conditions naming the empirical signals that would trigger N-small (false-negative on staleness that develops faster than 1000 cycles allows detection) or N-large (false-positive on transient variance that 1000 cycles is too sensitive to).**

### Finding 6 — N_A bundling: include in this RFC for operational coherence

Per Phase 1's N_A bundling open question: Layer A's `N_A` cadence parameter is the third operational parameter alongside T_B and K_C in the full demotion-candidacy predicate. Bundling N_A into this RFC's resolution produces a single source of truth for the LayerBParameters proto VALUES; deferring N_A to a separate RFC creates split operational ownership for the same predicate. **Bundling is structurally coherent.** N_A candidates inherit the same medium-zone-default pattern:

- N_A-short (1 hour) — high reactivity; risks Layer A firing constantly
- N_A-medium (1 day) — balanced; aligns with inception-phase observable substrate scale
- N_A-long (1 week) — slow cadence; matches stable-operations posture
- N_A-very-long (1 month) — quarterly-rotation cadence

**N_A-medium (1 day) is the conservative-defaults choice** — long enough to avoid Layer A firing on minor cadence variations, short enough to permit reaction within a working week.

### Finding 7 — Reversal-conditions per-parameter granularity is more rigorous

Per Phase 1's reversal-conditions record granularity open question: per-parameter granularity (one reversal condition per parameter) is more rigorous than parameter-set granularity (one reversal condition for the entire set). The per-parameter form admits independent revision — if empirical pressure surfaces on T_B but not on K_C, the resolution can revise T_B without re-opening K_C. **Per-parameter is structurally aligned with the form-vs-parameter discipline — each parameter is independent at the contract layer, so the reversal-conditions record should be independent at the operational-specification layer.**

### Finding 8 — Carry-forward: empirical-pressure-phase triggers must be observable, not predictive

Per [`§0022`](../../charter/decision-log.md) empirical-pressure-phase discipline: reversal triggers are empirical signals, not predictions. The reversal-conditions record names what would be observed in operation to trigger a revision RFC; it does NOT predict what those observations will show. **The triggers are observation-based, not hypothesis-based** — the structural commitment is "if we observe X, revise"; not "we predict X will be observed".

### Finding 9 — Methodological observation: operational-spec RFCs admit lighter Phase 3

Per Phase 3's calibration carry-forward (third bullet): operational-spec RFCs admit a lighter Phase 3 than ontology-revision RFCs because the structural form is fixed at upstream resolutions. The epistemic-skill matrix focuses on posture-fit at inception phase rather than on structural admissibility. **This is the first operational-spec RFC to instantiate the pattern; the methodology is recorded for future operational-spec RFCs.**

## Phase 5 — Recommendation

The discussion phase recommends the following parameter values:

| Sub-decision | Recommendation | Reversal trigger (per Finding 7) |
|---|---|---|
| **T_B** | **0.5** (T_B-medium) | Revise if observed α-distribution median over substantial promoted-hypothesis substrate exhibits systematic bias above 0.7 (false-negative on staleness) or below 0.3 (false-positive on productive hypotheses) |
| **K_C** | **0.5** (K_C-medium) | Revise if observed saturation distribution shows productive hypotheses firing Layer B (false-positive) or clearly-saturating hypotheses persisting beyond multiple Layer A cycles without firing (false-negative) |
| **N** | **1000** (N-medium) | Revise if Layer B's reactivity proves observably misaligned — false-negatives on staleness developing faster than 1000-assertion cycles allow detection (reduce N), or false-positives on transient variance (increase N) |
| **Window form** | **W-count** (fixed-count) | Revise to W-time or W-hybrid if substrate-commit-order proves inadequate for capturing operationally-significant temporal patterns (e.g., burst activity, periodic dormancy) |
| **Per-subtype divergence** | **U-uniform** (single set at abstract Hypothesis level) | Revise to P-per-subtype if observed α / closure / lifecycle distributions diverge substantially across the four Cat III subtypes |
| **N_A bundling** | **Bundle into this RFC** for operational coherence | (Bundling decision; no per-parameter reversal) |
| **N_A (cadence parameter)** | **1 day** (N_A-medium) | Revise if Layer A's firing rate proves observably misaligned — false-negatives on stale hypotheses persisting beyond a working week without Layer A firing (reduce N_A), or false-positives on Layer A firing more frequently than operational team can review (increase N_A) |
| **Reversal-conditions granularity** | **Per-parameter** (one condition per parameter) | (Structural choice; no reversal-of-reversal-conditions) |

Full demotion-candidacy predicate under the recommendation:

> `DEMOTE-CANDIDATE(H) := (elapsed_time_since_H.promotion > 1 day) AND ((freshness_B(H) < 0.5) OR (saturation_C(H) > 0.5))`

with `freshness_B` and `saturation_C` computed over the last 1000 assertions referencing H (transitively per Q5-τ for freshness_B; transitively excluding H's direct enrichment outputs per L-C commitment for saturation_C). Uniform across all four Cat III concrete subtypes.

### Rationale by Phase 4 finding

- **F1 (medium-zone)** — T_B-medium, K_C-medium, N-medium, N_A-medium all default to the medium-zone per inception-phase posture.
- **F2 (Thesis framing)** — T_B-medium and K_C-medium specifically encode §1 Thesis half-boundary structural meaning.
- **F3 (W-count)** — W-count avoids clock-time semantic risk; W-time / W-hybrid retained as empirical-pressure-phase reversal options.
- **F4 (U-uniform)** — U-uniform inherits the established Q3 / Q5 / Layer B inception-phase pattern.
- **F5 (N-medium)** — N-medium is the conservative-defaults balance between reactivity and stability.
- **F6 (N_A bundled)** — Bundling produces a single source of truth for LayerBParameters proto VALUES.
- **F7 (per-parameter granularity)** — Per-parameter reversal-conditions admit independent revision.
- **F8 (observation-based triggers)** — All reversal triggers are observation-based, not hypothesis-based.

### What would reverse this set of recommendations

The recommendations flip or substantially change if any of the following emerges:

- **Empirical observations on substrate-state distributions** — once the substrate has accumulated promoted hypotheses, α distributions, closure-membership counts, and lifecycle event rates may show patterns that argue for non-medium values. The reversal-conditions record per-parameter names specific triggers.
- **Operational tolerance feedback** — operator review of Layer B firings may show systematic mis-firings or persistent failures-to-fire; per-parameter reversal conditions admit targeted revision.
- **Temporal-pattern dominance** — if substrate exhibits substantial temporal patterns (burst, periodic), W-count may prove insufficient and W-time or W-hybrid may need to be revisited.
- **Per-subtype divergence empirically observed** — if measured α / closure / lifecycle distributions diverge substantially across the four concrete subtypes, P-per-subtype becomes preferred.
- **Form-level reversal** — if a future RFC reverses [`§0011`](../../charter/decision-log.md) Layer A AND Layer B → Layer A OR Layer B, the entire parameter set may need re-calibration. The Q4 reversal-conditions record is operative; this RFC's parameter choices remain valid under the AND composition.

### Implication for canonical-serialization-contract LayerBParameters proto

Per [`§0136`](../../charter/decision-log.md): the `LayerBParameters` proto fields receive the recommended values:

```proto
LayerBParameters {
    t_b: EvidentialIndependence { numerator: 1, denominator: 2 }      # 0.5
    k_c: EvidentialIndependence { numerator: 1, denominator: 2 }      # 0.5
    n_window: 1000
    // N_A bundling adds:
    n_a_duration_nanoseconds: 86400000000000                          # 1 day in nanoseconds
}
```

The contract revision to add `n_a_duration_nanoseconds` is a sub-decision of this resolution; the proto change is committed alongside this RFC's resolution at the canonical-serialization-contract layer per the [`§0028`](../../charter/decision-log.md) + [`§0034`](../../charter/decision-log.md) AP5 step (b) precedent.

### Implication for §0022 empirical-pressure-phase discipline

The reversal-conditions record committed with this resolution names the observable signals that would trigger a revision RFC. Per [`§0022`](../../charter/decision-log.md) discipline: implementation work proceeds under the recommended values; substrate-state observations are collected; if any reversal trigger fires, a revision RFC opens at the operational-specification layer.

---

## References

- [`docs/architecture/canonical-serialization-contract.md`](../../architecture/canonical-serialization-contract.md) — §0136 contract revision; LayerBParameters proto.
- [Charter §1 Thesis](../../charter/constitutional-charter.md#1-thesis) — names the two failure modes Layer B parameter calibration must preserve defense against.
- [Charter §2.4](../../charter/constitutional-charter.md#24-inferential-influence-disclosure) v0.5 + [`§0134`](../../charter/decision-log.md) Q5-τ — closure_hashes structural surface K_C operates on.
- [Charter §2.6](../../charter/constitutional-charter.md#26-evidential-independence-integrity) v0.6 — evidential_independence paired-dimension; T_B operates on α values.
- [`decision-log.md` §0010](../../charter/decision-log.md) Q2-A.2 — four concrete Cat III subtypes (per-subtype divergence question).
- [`decision-log.md` §0011](../../charter/decision-log.md) — Q4 Layer A AND Layer B staged-combination; form-vs-parameter discipline.
- [`decision-log.md` §0022](../../charter/decision-log.md) — empirical-pressure-phase discipline; inception-phase posture.
- [`decision-log.md` §0023](../../charter/decision-log.md) — inception-phase single-tier `actor_ref`; conservative-defaults posture.
- [`decision-log.md` §0027](../../charter/decision-log.md) — inception-phase storage; substrate scale.
- [`decision-log.md` §0133](../../charter/decision-log.md) — Q3-α resolution.
- [`decision-log.md` §0134](../../charter/decision-log.md) — Q5-τ + β-graph storage resolution.
- [`decision-log.md` §0135](../../charter/decision-log.md) — Layer B L-BC-OR resolution; parameter-calibration follow-on opening.
- [`decision-log.md` §0136](../../charter/decision-log.md) — canonical-serialization-contract revision; LayerBParameters proto + parameter-mutability prohibition.
- [`operational-spec-layer-b-parameter-calibration`](../draft/operational-spec-layer-b-parameter-calibration.md) — the draft RFC this scratch supports.
- [`q3-independence-evidence.md`](./q3-independence-evidence.md) — Q3 discussion-phase evidence (precedent for Phase structure).
- [`q5-transitivity-evidence.md`](./q5-transitivity-evidence.md) — Q5 discussion-phase evidence.
- [`layer-b-deep-criterion-evidence.md`](./layer-b-deep-criterion-evidence.md) — Layer B discussion-phase evidence (form-level resolution).
