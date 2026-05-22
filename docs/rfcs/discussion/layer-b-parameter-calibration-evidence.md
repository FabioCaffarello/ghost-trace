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

## Phases 3+ — Deferred to substantive deliberation

The following phases are drafted in subsequent RFC commits:

- **Phase 3 — Epistemic-skill application.** Apply [`falsifiability-check`](../../../.claude/skills/epistemic/falsifiability-check/SKILL.md), [`epistemic-separator`](../../../.claude/skills/epistemic/epistemic-separator/SKILL.md), and [`ambiguity-reducer`](../../../.claude/skills/epistemic/ambiguity-reducer/SKILL.md) to each candidate per the Q3 / Q5 / Layer B Phase 3 precedent. Note: this RFC is OPERATIONAL specification, not structural; the epistemic-skill application is lighter than for ontology RFCs (the structural form is fixed at §0135).
- **Phase 4 — Comparison synthesis.** Rank candidates against §1 Thesis coverage at conservative defaults, false-positive resistance, inception-phase posture, and form-vs-parameter discipline.
- **Phase 5 — Recommendation.** Single recommendation per sub-decision; explicit reversal-conditions record for empirical-pressure-phase triggers; canonical-serialization-contract LayerBParameters proto values.

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
