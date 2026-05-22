# Q3 — Formal definition of `evidential independence` as measurable quantity — discussion evidence

**Status:** in-discussion. Not authoritative. Final resolution will be recorded in decision-log and Ontology document revision.

This scratch supports the discussion phase of [`ontology-revision-q3-independence`](../draft/ontology-revision-q3-independence.md), opened per [`decision-log.md` §0132](../../charter/decision-log.md) as Layer B follow-on RFC pre-Gate dependency (continuation of lazy pre-Gate methodology per [`§0014`](../../charter/decision-log.md), now applied at RFC-level rather than Charter-§2.x-level).

This is the second Ontology Open Question whose discussion file is named with the bare prefix `q3-` (the first is [`q3-evidence.md`](./q3-evidence.md), subject reference polymorphism, resolved at [`§0016`](../../charter/decision-log.md)). The naming clash is acknowledged in the decision-log entry: ontology.md Open Question 3 (the formal-independence question this file supports) and the §0014→§0016 Charter-cascade "Q3" (subject ref polymorphism) share a number across two distinct numbering schemes. The disambiguating suffix in this file's name (`-independence`) is the convention going forward.

This is a strictly-framing scratch: Phase 1 names the question and the dependency surface; Phase 2 enumerates candidate measurable-quantity families without selecting one. Phases 3+ (epistemic-skill application, comparison synthesis, recommendation) are drafted in a subsequent RFC commit when the discussion advances substantively.

---

## Phase 1 — Scope and dependencies

### The question

[`docs/ontology/ontology.md` Open Question 3](../../ontology/ontology.md):

> "3. What is the formal definition of `independence` as a measurable quantity? Conceptually agreed; operationally undefined."

Resolved under canonical vocabulary discipline (per [`vocabulary-discipline` §4](../../../.claude/skills/ontology/vocabulary-discipline/SKILL.md)), the question reads: **what is the measurable-quantity formalization of the `evidential_independence` dimension that [Charter §2.6](../../charter/constitutional-charter.md#26-evidential-independence-integrity) frozen v0.6 defers to operational specification at its BC1?**

Charter §2.6 BC1 (verbatim):

> "§2.6 governs the structural pairing of `confidence` + `evidential_independence`; not the formal measurable-quantity definition of independence. The pairing's structural enforcement is binding today; the formal definition of `independence` as a measurable quantity is operational specification deferred to Q3 of `ontology.md` Open Questions. The default-level commitment is binding today; the formal mechanism becomes structurally falsifiable when Q3 resolution lands."

The deferred specification names two coupled sub-questions:

- **Sub-question 3.a — Derivation meta-shape.** Per §2.6 BC2, three meta-shapes are admitted: deterministic-from-pattern, substrate-computed, or operator-supplied. Q3 must select one (or a hybrid).
- **Sub-question 3.b — Structural formula or constraint.** Given a meta-shape, what is the specific structural formula (for meta-shape 1 or 2) or validation constraint (for meta-shape 3) that produces the per-record `evidential_independence` value?

Q3 closure requires both sub-questions answered.

### In scope

- The derivation rule for `evidential_independence` at substrate write time per [`§0021`](../../charter/decision-log.md) OMQ #3-α (substrate-time generation).
- The structural inputs the derivation rule consumes (provenance subgraph per [`§2.3`](../../charter/constitutional-charter.md#23-provenance-integrity) v0.4; influence subgraph per [`§2.4`](../../charter/constitutional-charter.md#24-inferential-influence-disclosure) v0.5).
- The structural admissibility of the derived value (range, type, validation discipline) at the canonical-serialization-contract layer per [`§0034`](../../charter/decision-log.md).
- The interaction with [§2.6](../../charter/constitutional-charter.md#26-evidential-independence-integrity) forbidden anti-patterns — specifically the "independence collapsed into confidence at projection" prohibition (a derivation rule that produces `evidential_independence = f(confidence)` is structurally forbidden by §2.6 anti-pattern 2, regardless of which meta-shape it falls under).

### Out of scope

- **Layer B deep criterion.** Layer B's specification (per [`ontology-revision-layer-b-deep-criterion`](../draft/ontology-revision-layer-b-deep-criterion.md)) consumes Q3's resolution; Q3 produces the measurable quantity, Layer B specifies the threshold-test that uses it. Distinct RFCs.
- **Transitive propagation semantics.** [`ontology.md` Q5](../../ontology/ontology.md) (How does `influence` propagate through derived assertions? Transitive? Decaying? Both?) is partially-resolved — decay was resolved at [`§0020`](../../charter/decision-log.md) OMQ #2-C (via §2.5 lifecycle event supersession). The transitivity half remains open and is forward-referenceable from Q3 per the §0014 + §0017 lazy pre-Gate / forward-reference methodology. Whether Q5 transitivity becomes blocking is assessed at Q3 Step 1.1-equivalent during substantive deliberation.
- **Identity-tier specifics.** [`entity-model.md` Open Modeling Question 1 (Identity tiers)](../../ontology/entity-model.md#open-modeling-questions) — inception-phase single-tier `actor_ref` per [`§0023`](../../charter/decision-log.md) — is forward-referenceable. Multi-tier extension is ordinary Ontology RFC discipline post-§0023.
- **The per-record runtime mechanics of independence reading.** Graph indexes, projection-rebuild paths, runtime traversal mechanics are architecture-document territory per [§2.6 BC8](../../charter/constitutional-charter.md#26-evidential-independence-integrity). Q3 governs the substrate-time derivation; the projection-time reading is downstream.

### Resolved dependencies (structural ground present)

The following Charter and decision-log resolutions are in place and load-bearing for Q3:

| Anchor | What it commits | How Q3 consumes it |
|---|---|---|
| [`§2.4`](../../charter/constitutional-charter.md#24-inferential-influence-disclosure) frozen v0.5 | `influenced_by` chain is a structural attribute of every inferential-commitment record | Q3 derivation rule may read the influence chain as input |
| [`§2.6`](../../charter/constitutional-charter.md#26-evidential-independence-integrity) frozen v0.6 | `evidential_independence` is a structurally-distinct dimension paired with `confidence`; both required at substrate commit per [`§0034`](../../charter/decision-log.md); structurally independent at substrate (neither derivable from the other) | Q3 specifies the derivation rule for the `evidential_independence` value — the structural shape into which Q3's resolution lands |
| [`§0020`](../../charter/decision-log.md) OMQ #2-C | Decay of influence is via §2.5 lifecycle event supersession, NOT a runtime derivation parameter | Q3 derivation rule operates on the substrate-state-at-commit-time snapshot; decay is read at projection time via supersession check, not baked into the derivation rule |
| [`§0021`](../../charter/decision-log.md) OMQ #3-α | Independence values are committed at substrate write time (formation event time of the influenced Assertion) | Q3 derivation rule must be evaluable at substrate write time using only substrate-committed records available at that time |
| [`§0034`](../../charter/decision-log.md) | Canonical-serialization-contract enforces paired-dimension requirement at commit | Q3 resolution surfaces in canonical-serialization-contract as a per-record value with declared type/range |
| [`§2.3`](../../charter/constitutional-charter.md#23-provenance-integrity) frozen v0.4 | Provenance chain via typed `subject_ref_*` fields terminates at Cat I primaries | Q3 derivation rule may traverse the provenance subgraph as input; termination guarantee is structural |
| [`§0023`](../../charter/decision-log.md) Q2 inception | Single-tier `actor_ref` for inception phase | Q3 operates over single-tier actor identities at the inception phase; multi-tier extension is forward-referenceable per §0023 |

### Open dependencies (assessed at substantive deliberation, not pre-Gate)

| Open question | Why potentially blocking | Default disposition |
|---|---|---|
| [`ontology.md` Q5](../../ontology/ontology.md) transitivity half | If Q3's derivation rule traverses the provenance graph transitively (e.g., Candidates α, β, γ below), the transitivity rule of `influence` propagation along the graph affects what counts as "influenced" at each edge. Decay half resolved at §0020; transitivity half pending. | Forward-referenceable per §0014 + §0017 lazy pre-Gate methodology. Assessed empirically when Q3 advances past Phase 2 (candidate enumeration) to Phase 3+ (substantive evidence). Promotion to cascade-trigger status (per §0015 → §0016 + §0020 → §0021 precedent) only if a chosen Q3 candidate cannot be specified without resolving transitivity. |
| [`entity-model.md` OMQ #1 (Identity tiers)](../../ontology/entity-model.md#open-modeling-questions) | If Q3's derivation rule counts distinct provenance roots and "distinct" depends on identity-tier specifics, multi-tier extension may affect the formula. | Forward-referenceable per [`§0023`](../../charter/decision-log.md) — inception-phase `actor_ref` is the operational default; multi-tier is ordinary Ontology RFC. Not anticipated to block. |

### Why now

[`§0011`](../../charter/decision-log.md) opened the Layer B follow-on RFC at on-hold status pending §2.4 + §2.6 redaction. Both Charter dependencies are now satisfied (§2.4 frozen v0.5; §2.6 frozen v0.6 at [`§0129`](../../charter/decision-log.md)). The Layer B RFC's [Status note](../draft/ontology-revision-layer-b-deep-criterion.md) names Q3 + Q5 as ontology-side dependencies remaining before Layer B's substantive content can be drafted. Q3 is the most upstream of the two: without a measurable `evidential_independence` quantity, Q5's transitivity question has no measurable target to propagate, and Layer B's deep criterion has no quantity to threshold-test.

Opening Q3 at discussion now (rather than waiting for Layer B to surface it) follows the §0014 lazy pre-Gate methodology: each Charter or RFC redaction opens its minimal pre-Gate at discussion status; subsequent dependencies are assessed empirically during substantive phases. Q3 is the minimal pre-Gate for Layer B.

### Procedural posture

This RFC is at `discussion` status. Phase 1 (this section) names the dependency surface. Phase 2 (below) enumerates candidate measurable-quantity families. Phase 3+ (epistemic-skill application, comparison synthesis, recommendation) is drafted in subsequent commits when the committee deliberates substantively. The resolution lands at a future `decision-log` entry; ontology.md Open Question 3's line is closed at that entry.

---

## Phase 2 — Candidate measurable-quantity families

Six candidate families enumerated, partitioned by §2.6 BC2's three meta-shapes. Each candidate cites its structural inputs, the structural shape of the derived value, the constraints from resolved decisions it must satisfy, and its dependency on open questions. No candidate is selected at this phase.

### Meta-shape 1 — Deterministic-from-pattern

Per §2.6 BC2 meta-shape 1: the value is fully determined by the substrate state at commit time, computed by a published formula. Producer cannot influence the value; the formula is part of the canonical-serialization-contract.

#### Candidate α — Source-count ratio over Cat I provenance roots

**Structural input:** the typed `subject_ref_*` chain (per §2.3 v0.4) traced from the assertion to its Cat I primary observation roots; cross-referenced against the `influenced_by` chain (per §2.4 v0.5).

**Derived-value shape:** `evidential_independence = (count of Cat I roots NOT reachable through any influenced_by edge from any promoted hypothesis) / (total count of Cat I roots)`. Range [0, 1]; type rational (canonical-serialization-contract specifies precision).

**Constraints satisfied:** §0021 substrate-time evaluable (provenance + influence chains are both substrate-committed before the assertion in question); §0020 supersession-compatible (counts read substrate-committed values; demotion supersession applied at projection time per §2.6 line 245); §2.6 anti-pattern 2 satisfied (not a function of confidence).

**Open-question dependency:** depends on Q5 transitivity half if `influenced_by` is transitive across the assertion graph (a Cat I root reachable only via a chain of influences must be counted as "influenced"). Decay is supersession-only per §0020; α's substrate-committed value is decay-immune by §2.1.

**One-line tension:** structurally minimal and verifiable, but the ratio's resolution is bounded by the cardinality of the Cat I root set — a hypothesis with 3 supporting observations admits only 4 distinct independence values {0, 1/3, 2/3, 1}.

#### Candidate β — Influence-edge fraction

**Structural input:** all edges in the assertion's provenance subgraph; the subset of those edges that pass through at least one `influenced_by` edge from any promoted hypothesis.

**Derived-value shape:** `evidential_independence = 1 − (count of influenced edges) / (count of total edges)`. Range [0, 1]; type rational.

**Constraints satisfied:** §0021 substrate-time evaluable; §0020 supersession-compatible at projection; §2.6 anti-pattern 2 satisfied.

**Open-question dependency:** Q5 transitivity-half determines what "passes through" means — direct edge only, or transitive closure? Distinct from α: α counts roots, β counts edges; both depend on transitivity.

**One-line tension:** higher resolution than α (admits finer-grained values) but conflates short-and-deep with long-and-shallow provenance topologies that may carry different epistemic weight.

#### Candidate γ — Topological-distance to nearest non-influenced ancestor

**Structural input:** the assertion's provenance subgraph; computation of the shortest path from the assertion to any non-influenced ancestor.

**Derived-value shape:** `evidential_independence = f(distance)` where `f` is a monotone-decreasing function specified by the canonical-serialization-contract (e.g., `1 / (1 + distance)`, or piecewise constants). Range [0, 1]; type rational.

**Constraints satisfied:** §0021 substrate-time evaluable; §0020 supersession-compatible at projection; §2.6 anti-pattern 2 satisfied.

**Open-question dependency:** Q5 transitivity-half determines graph traversal direction and what constitutes a "non-influenced" edge. Q2 identity-tier extension may affect whether distinct ancestors are merged across tiers, changing the distance metric.

**One-line tension:** topology-sensitive rather than count-sensitive — measures structural distance from the assertion to independent ancestry rather than the proportion of independent roots, but requires choosing a function `f` whose shape is itself a sub-decision deferred to operational specification.

### Meta-shape 2 — Substrate-computed (formula deferred to operational specification)

Per §2.6 BC2 meta-shape 2: the value is computed by the substrate at commit time using a derivation rule that is operationally specified rather than fully crystallized in the canonical-serialization-contract. Distinguished from meta-shape 1 by the locus of the formula's definition: meta-shape 1 freezes the formula at the contract; meta-shape 2 keeps the formula in operational specification, with the contract enforcing only the type/range of the output.

#### Candidate δ — Substrate-computed with operationally-specified formula (e.g., parameterized variant of α/β/γ)

**Structural input:** any combination of the structural inputs to α, β, γ, plus possible parameters supplied by the operational specification (e.g., transitivity depth K; root-set weighting scheme).

**Derived-value shape:** range [0, 1]; type rational; specific formula NOT specified in the canonical-serialization-contract — specified in an operational document subject to ordinary Ontology RFC discipline.

**Constraints satisfied:** §0021 substrate-time evaluable (the substrate has the operational specification at commit time); §0020 supersession-compatible at projection; §2.6 anti-pattern 2 satisfied.

**Open-question dependency:** the operational specification document may itself surface Q5 / Q2 dependencies. The meta-shape commits to the existence of an operational specification; the specification's content is a follow-on RFC.

**One-line tension:** flexibility at the cost of the structural-falsifiability guarantee §2.6 BC1 commits to — Q3 resolution under δ leaves part of the falsifiability deferred to the follow-on operational RFC. Distinguished from a meta-pattern selection because δ commits to the substrate-computed branch (excluding meta-shapes 1 and 3); the operational RFC fills in the formula.

### Meta-shape 3 — Operator-supplied

Per §2.6 BC2 meta-shape 3: the producer declares the value at commit time. Substrate validates the value's type and range but does not derive it. Distinguished from meta-shape 1 and 2 by the absence of a substrate-side derivation rule.

#### Candidate ε — Operator-supplied within canonical-serialization-contract validation

**Structural input:** producer-declared value at commit time.

**Derived-value shape:** range [0, 1]; type rational; substrate validates range and type only.

**Constraints satisfied:** §0021 substrate-time committed (the value is present at write time); §0020 supersession-compatible at projection (the substrate-committed value is read at projection; supersession affects whether/how it is consumed, not the value itself); §2.6 anti-pattern 2 satisfied at structural level (the substrate cannot detect a producer who internally computes the value from confidence — anti-pattern 2 is enforced via canonical-serialization-contract output equality at projection-replay per §2.6 forbidden anti-pattern 2 detection).

**Open-question dependency:** none directly — operator-supplied bypasses Q5 / Q2 at the substrate layer (the producer's internal logic may consume them, but the substrate does not).

**One-line tension:** structurally simplest at the substrate layer but raises [Charter §4](../../charter/constitutional-charter.md#4-constitutional-design-rule) criterion 1 (structural enforceability) concerns — the substrate cannot enforce that the value reflects independence in any specific structural sense. ε's defense is procedural (declared value + producer-accountable provenance per §2.4); §4 criterion 1 questions whether procedural defense suffices for a Charter-defended dimension.

#### Candidate ζ — Hybrid (substrate-computed baseline + operator-declared refinement)

**Structural input:** substrate-computed value via one of α/β/γ at commit time, PLUS optional operator-declared refinement value (constrained to a bounded range around the substrate-computed baseline).

**Derived-value shape:** the recorded `evidential_independence` is the operator-declared refinement when present, defaulting to the substrate-computed baseline. Range [0, 1]; type rational. Canonical-serialization-contract carries both values (or the difference) per [`§0034`](../../charter/decision-log.md) discipline.

**Constraints satisfied:** §0021 substrate-time evaluable for the baseline; substrate-validation of the refinement at commit; §0020 supersession-compatible at projection; §2.6 anti-pattern 2 satisfied (the substrate-computed baseline is not a function of confidence; the refinement is bounded around the baseline).

**Open-question dependency:** inherits the open-question dependency of the chosen baseline (α / β / γ).

**One-line tension:** combines the structural-enforceability of meta-shape 1 with the operator-flexibility of meta-shape 3 at the cost of carrying two values per record (substrate-baseline + operator-refinement) and a more elaborate canonical-serialization-contract. Whether the additional structural surface justifies the flexibility is the operative cost-benefit question.

### Asymmetries surfaced

Two asymmetries partition the candidate space and will likely organize substantive deliberation:

- **Structural-enforceability asymmetry (§4 criterion 1):** α / β / γ / ζ-baseline are structurally enforceable at the substrate; δ defers part of the enforcement to operational specification; ε defers all of it to procedural defense. The asymmetry is decisive for [§4](../../charter/constitutional-charter.md#4-constitutional-design-rule) discipline questions.
- **Q5-transitivity dependency asymmetry:** α / β / γ / δ-with-graph-formula / ζ-baseline all depend on Q5's transitivity-half resolution; ε / δ-with-non-graph-formula do not. The asymmetry is decisive for whether Q5 must be opened as cascade-trigger before Q3 can advance to Phase 3 — under candidates that depend on Q5, Q5 likely cascades per §0015 → §0016 + §0020 → §0021 precedent; under candidates that do not, Q5 remains forward-referenceable.

These asymmetries are recorded for use by substantive deliberation; they are NOT selections.

---

## Phase 3 — Apply epistemic skills

Each candidate is restated as an abstract structural proposition; three epistemic skills ([`falsifiability-check`](../../../.claude/skills/epistemic/falsifiability-check/SKILL.md), [`epistemic-separator`](../../../.claude/skills/epistemic/epistemic-separator/SKILL.md), [`ambiguity-reducer`](../../../.claude/skills/epistemic/ambiguity-reducer/SKILL.md)) are applied to each per the Q2-1 + Q4-1 precedent. Six candidates × three skills = 18 cells. The skills are non-redundant: falsifiability tests the criterion's reducibility to substrate; epistemic-separator tests category-boundary discipline and intra-category circularity risks; ambiguity-reducer scans for watchlist hits and operationalization deficits.

The vocabulary-discipline skill is not run as a separate column because all candidate propositions use canonical vocabulary (`evidential_independence`, `influenced_by`, `subject_ref_*`, `Category I/II/III`) by construction in Phase 2; vocabulary-discipline is a write-time discipline rather than a comparative-evaluation skill.

### Candidate propositions

- **α proposition:** "A record's `evidential_independence` is the ratio of Cat I primary observation roots in its `subject_ref_*` chain that are NOT reachable via any `influenced_by` edge from a promoted hypothesis, divided by the total Cat I roots in the chain."
- **β proposition:** "A record's `evidential_independence` is one minus the fraction of provenance subgraph edges that pass through at least one `influenced_by` edge from any promoted hypothesis."
- **γ proposition:** "A record's `evidential_independence` is a monotone-decreasing function `f` of the shortest graph distance from the record to its nearest non-influenced ancestor in the provenance subgraph."
- **δ proposition:** "A record's `evidential_independence` is computed by the substrate at commit time using a derivation formula specified in an operational-specification document subject to ordinary Ontology RFC discipline; the canonical-serialization-contract validates the output's type/range only."
- **ε proposition:** "A record's `evidential_independence` is the value declared by the producer at commit time; the substrate validates the value's type and range but does not derive it."
- **ζ proposition:** "A record's `evidential_independence` is the substrate-computed baseline (via α, β, or γ) optionally refined by a producer-declared value bounded around the baseline; the recorded value is the refinement if present, defaulting to the baseline."

### 6 × 3 matrix — candidate × skill

| Candidate | `falsifiability-check` | `epistemic-separator` | `ambiguity-reducer` |
|---|---|---|---|
| **α — Source-count ratio** | §1.1: violation detectable by recomputing the ratio against substrate-committed provenance + influence chains. §1.2: third party reads `subject_ref_*` + `influenced_by` chains from substrate; no producer cooperation needed. §1.3: reduces to Cat I root count + influenced-chain reachability — both substrate artifacts post-§2.3 v0.4 + §2.4 v0.5. §1.4: clean (no self-reference; the ratio operates on prior records). **Verdict: pass-with-Q5-dependency.** "Reachable via any `influenced_by` edge" reduces fully to substrate only when Q5 transitivity-half is resolved (direct edge vs transitive closure). Today: pass at §1.1 + §1.2; partial at §1.3. Post-Q5: full pass. | Inputs: Cat I roots (Cat I record class) + `influenced_by` chain membership (§2.4 v0.5 structural attribute of Cat II/III records). No cross-category mixing in the criterion itself. **Risk:** if "Cat I roots" in the chain are reached through a Cat II construct that was Cat-III-influenced, the chain crosses inferential influence en route to the Cat I root. §2.3 v0.4 mandates chain termination at Cat I; α must structurally subtract Cat I roots reached only through influenced Cat II from the "non-influenced" count. **Verdict: clean per category boundary, with documented requirement that α's "not reachable via influenced_by" predicate exclude Cat I roots whose only path traverses an influenced intermediate.** The exclusion is structurally available via §2.4 chain inspection; no new structural surface required. | Terms in α's prose: `ratio`, `Cat I primary observation roots`, `reachable`, `subject_ref_*`, `influenced_by`. Watchlist scan: none of these are on the [`ambiguity-reducer`](../../../.claude/skills/epistemic/ambiguity-reducer/SKILL.md) watchlist. `ratio` is structurally well-defined (count/count). `reachable` is graph-theoretic; depends on Q5 transitivity for full operationalization — the skill's Response 3 path applies (raise as open modeling question; Q5 is exactly that question). **Verdict: minor carry-forward.** Vocabulary is structurally clean; the `reachable` operationalization is the same Q5-dependency surfaced under falsifiability. |
| **β — Influence-edge fraction** | §1.1: violation detectable by recomputing the edge-fraction against the substrate-committed graph. §1.2: third party reads provenance + influence subgraphs from substrate. §1.3: reduces to edge counting — structural. §1.4: clean. **Verdict: pass-with-Q5-dependency.** "Passes through at least one `influenced_by` edge" admits ambiguity between direct-edge-only and transitive-closure interpretations; resolution requires Q5 transitivity-half. Today: pass at §1.1 + §1.2; partial at §1.3. Post-Q5: full pass. | Inputs: provenance graph edges + their influenced_by membership. **Risk:** the "denominator" (total edges) treats Cat-I-to-Cat-II edges and Cat-II-to-Cat-III edges symmetrically — but a chain from a Cat I observation through two Cat II constructs to a Cat III hypothesis has 2 Cat-II-to-Cat-II edges that may carry different epistemic weight than a single Cat-I-to-Cat-II edge. The skill's category-boundary discipline questions whether all edges should count equally; β's structural form is silent on edge-weighting. **Verdict: clean per category boundary, with documented residual question on edge-weighting that is itself a sub-decision deferred to operational specification.** | Terms in β's prose: `fraction`, `provenance edges`, `influenced_by`, `passes through`. Watchlist scan: `passes through` is informal — depends on Q5 transitivity for operationalization (same Response-3 path as α's `reachable`). `fraction` is structurally well-defined. **Verdict: minor carry-forward — `passes through` operationalization is the Q5-dependency surfaced under falsifiability + the edge-weighting question surfaced under epistemic-separator.** Vocabulary clean. |
| **γ — Topological-distance to non-influenced ancestor** | §1.1: violation detectable by recomputing the distance + applying `f`. §1.2: substrate-readable graph distance. §1.3: reduces to shortest-path computation over the provenance subgraph + applying a contract-specified function `f` — substrate artifacts post-§2.3 + §2.4. §1.4: clean. **Verdict: pass-with-Q5-and-function-specification-dependency.** Two sub-dependencies: (a) Q5 transitivity-half for "non-influenced ancestor" reachability semantics; (b) the function `f` (e.g., `1/(1+distance)` vs piecewise constants) is itself a sub-decision deferred to operational specification per Phase 2 Candidate γ's tension. Today: pass at §1.1 + §1.2; partial at §1.3 (depends on `f` resolution). | Inputs: graph distance + ancestor inspection. **Risk:** "ancestor" must be structurally defined to traverse only typed `subject_ref_*` edges (not arbitrary edges). §2.3 v0.4 provides the typed chain. **Risk:** under Q2 identity-tier extension, "distinct ancestors" across tiers may merge, altering the distance metric. **Verdict: clean per category boundary, with documented dependency on Q5 transitivity for ancestor-reachability semantics + Q2 forward-reference for identity-tier handling.** | Terms in γ's prose: `topological distance`, `non-influenced ancestor`, `function f`, `monotone-decreasing`. Watchlist scan: `function` may be advisory but is structurally well-defined here. `f` is parametric — its specific shape is operationally-deferred (Response 3 — raise as open modeling question for the operational spec RFC). **Verdict: ambiguity-flagged on `f` specification.** `f`'s shape is a sub-decision the resolution must either crystallize at the canonical-serialization-contract or open as a follow-on RFC. |
| **δ — Substrate-computed with operationally-specified formula** | §1.1: violation detectable only after the operational-specification document is in place. §1.2: substrate-readable IF the operational specification is itself substrate-published. §1.3: PARTIAL reduction — the validation rule (type/range) is contract-enforced, but the formula content is operationally-specified, not contract-frozen. §1.4: clean structurally (no circularity in the meta-shape itself). **Verdict: pass-with-operational-specification-RFC-dependency.** Today even more partial than α/β/γ at §1.3: half of the falsifiability surface is deferred to a yet-to-be-opened operational RFC. Post-operational-RFC, falsifiable IF that RFC produces a structurally-falsifiable formula; if the operational spec admits multiple producer-side interpretations, δ collapses toward ε. | Inputs: any combination of α/β/γ structural inputs + operationally-specified parameters. **Risk:** if the operational specification mixes Cat II and Cat III inputs without explicit category-discipline checks, δ admits intra-category drift that the substrate cannot detect at commit. The skill cannot verify δ at the Charter level; verification is deferred to the operational spec RFC, which itself is subject to the skill. **Verdict: clean only conditionally — depends on the operational spec RFC adhering to category discipline.** The conditional verdict is itself a structural fragility relative to α/β/γ. | Terms in δ's prose: `operationally-specified formula`, `substrate-computed`, `validation`. Watchlist scan: `formula` advisory; `specified` informal. The operational specification document is named but its content is unspecified — Response 3 (raise as open modeling question) applies to the meta-shape as a whole. **Verdict: ambiguity-flagged on the unspecified-content surface.** The candidate's structural form is named but its operational content is not yet enumerated; the skill demands the resolution either name the operational RFC at closure or commit to a specific formula. |
| **ε — Operator-supplied** | §1.1: violation = value outside the contract-validated range, detectable by substrate. §1.2: substrate-readable (the declared value). §1.3: reduces to per-record value validation — substrate artifact at the type/range level only. §1.4: clean structurally (no circularity in the value declaration itself). **Verdict: passes at §1.1 + §1.2 + §1.3 at the validation-range level only.** No pending dependencies — ε bypasses Q5 entirely. **But:** the falsifiability is shallow — a producer's declared 0.7 is structurally indistinguishable from another producer's declared 0.7 even if they used different internal logic. ε admits silent collapse-to-confidence (producer's internal logic may compute `independence = confidence` and declare it; §2.6 anti-pattern 2 detection at projection-replay does NOT catch this because the substrate value matches the declared value byte-for-byte). | Inputs: producer-declared value (opaque to substrate). **Risk:** the producer's internal logic is opaque — the substrate cannot enforce category discipline on whatever inputs the producer consults. A producer could base its declared independence on Cat III inputs that should have been excluded per epistemic separation; the substrate cannot detect this. **Verdict: clean at the substrate layer (the value itself is not category-mixed), but producer-side rationale is opaque to category-discipline enforcement.** The procedural defense (producer accountability + §2.4 chain disclosure) is the only enforcement surface; structural enforcement is structurally absent. | Terms in ε's prose: `operator-supplied`, `producer-declared`, `validation`. Watchlist scan: `declared` is structurally clean. `producer` is advisory. **Verdict: minor carry-forward.** Vocabulary clean at the structural surface; the semantic content of the declared value is opaque (which is the substantive concern surfaced under falsifiability + epistemic-separator, not under ambiguity-reducer). |
| **ζ — Hybrid (baseline + refinement)** | Inherits the chosen baseline's verdict (α/β/γ pass-with-Q5; δ pass-with-operational-RFC). Adds a validation-range surface for the refinement that is falsifiable independently (refinement violates bounds → substrate rejection). §1.4: clean. **Verdict: pass-with-baseline-dependency + additional validation-range falsifiability.** Strongest combined falsifiability surface among operator-touched candidates: the baseline structural falsifiability + the refinement's bounds-validation. Today: same operational readiness as the chosen baseline. | Inputs: baseline's inputs + producer's refinement. **Risk:** the refinement, even bounded around the baseline, can be biased systematically upward by the producer (toward higher independence values). The bounds-validation catches per-record violations but not population-level drift. **Verdict: clean per category boundary at the structural level; the bounded-refinement carries a population-drift risk that the substrate cannot detect per-record.** Stronger than ε (the baseline anchors at a structural value); weaker than α/β/γ alone (the refinement admits operator influence). | Terms in ζ's prose: `substrate-computed baseline`, `operator-declared refinement`, `bounded range`, `default`. Watchlist scan: `default` advisory; `refinement` is operationally informal. The two-value structure (baseline + refinement, or refinement-with-baseline-fallback) is a structural sub-decision the canonical-serialization-contract must crystallize — Response 3 applies. **Verdict: ambiguity-flagged on the two-value canonical structure.** The resolution must specify whether both values are committed or only one (and which); the contract surface is not yet enumerated. |

### Most consequential epistemic finding across the 18 cells

**Primary finding — Tier-partitioning by structural falsifiability.** The 18 cells partition Q3's candidate space into three operational-readiness tiers along the `falsifiability-check` axis:

- **Tier 1 (Q5-dependent, structurally falsifiable at substrate):** α, β, γ. Today pass-with-Q5-dependency at §1.3; post-Q5 transitivity-half resolution, full pass. The Q5-dependency is a single, named, already-identified upstream open question — falsifiability becomes fully operational at a known structural milestone.
- **Tier 2 (operational-RFC-dependent or population-level-vulnerable):** δ, ζ. δ defers half of falsifiability to a yet-to-be-opened operational RFC; ζ adds population-level drift risk that substrate per-record validation cannot catch. Falsifiability is partially structural + partially procedural.
- **Tier 3 (validation-only, opaque meaning):** ε. Passes shallowly — the validation-range surface is falsifiable, but the meaning of the declared value is opaque. §2.6 anti-pattern 2 detection (projection-replay byte-for-byte) does NOT catch a producer who internally computes `independence = confidence` and declares it — the byte-for-byte match holds against the producer's declaration, not against any structural criterion. **ε admits the §2.6 anti-pattern 2 failure mode in a way the anti-pattern's detection mechanism cannot discover.**

This tier-partitioning is decisive for [Charter §4 criterion 1](../../charter/constitutional-charter.md#4-constitutional-design-rule) (structural enforceability discipline): Tier 1 satisfies the criterion fully (post-Q5); Tier 2 satisfies it partially; Tier 3 satisfies it only procedurally.

**Secondary finding — ε's silent-collapse fragility.** ε's failure mode (producer internally computes `independence = confidence` and declares it) is the most direct manifestation of the [§1 Thesis](../../charter/constitutional-charter.md#1-thesis) failure mode that the entire §2.6 invariant exists to defend against. The structural pairing per §2.6 — confidence + evidential_independence as structurally distinct dimensions — is rendered substantively equivalent to a confidence-only commitment under ε. **The substrate cannot tell the difference.** The §2.6 anti-pattern 2 detection mechanism (byte-for-byte projection-replay) catches a *projection-time* re-derivation but not a *commit-time* producer-internal collapse — the latter is undetectable at the substrate layer.

**Tertiary finding — `epistemic-separator` calibration: opacity of producer-internal logic is a Cat-discipline failure mode.** [Q2-1's calibration carry-forward](./q2-evidence.md) named intra-category flattening; [Q4-1's](./q4-evidence.md) refined to intra-category circularity. Q3-1 adds a third intra-category failure mode: **opacity of producer-side derivation**. When the substrate accepts a producer-declared value without structural verification (ε; δ if the operational spec is weakly enforced), the substrate-layer category discipline is decoupled from the actual derivation, and intra-category drift can occur entirely below the discipline's enforcement surface. This is a new structural pattern not surfaced in prior Q's discussions.

### Calibration carry-forward to future Ontology RFCs

Q3-1 confirms two and extends one of the prior Q's calibrations:

- **Confirmed: falsifiability §1.3 (operationalization) does most of the work on substrate-touching propositions.** Q3-1 confirms — all six candidates decide at §1.3, with the §1.3 verdict structuring the Tier partitioning.
- **Confirmed: ambiguity-reducer surfaces residual carry-forwards that are themselves structural deferrals.** Q3-1 confirms — `f` under γ, the operational spec under δ, the two-value contract under ζ all surface as Response-3 cases.
- **Extended: epistemic-separator's intra-category failure modes now form a three-element catalogue.** Q2-1 (flattening) + Q4-1 (circularity) + Q3-1 (opacity of producer-side derivation). Future Ontology RFCs should scan for all three patterns. The catalogue is structural rather than enumerative — it reflects how producer/substrate boundary discipline composes with category discipline.

## Phase 4 — Comparison synthesis

Findings synthesized from Phase 1 (resolved-dependency surface), Phase 2 (six-candidate enumeration), and Phase 3 (18-cell epistemic-skill matrix). Classified as **asymmetry** (one candidate or candidate-class clearly stronger by evidence-grounded argument), **apparent trade-off that resolves** (a finding whose appearance reframes under deeper analysis), **genuine trade-off** (a substantive difference where no candidate clearly wins), or **tension** (a structural feature of the evidence space that does not resolve under any candidate alone). Numbered in order of consequence.

### Finding 1 — Asymmetry: Tier 1 (α/β/γ) is the only tier that satisfies §4 criterion 1 structurally

Phase 3's tier-partitioning is decisive for [Charter §4 criterion 1](../../charter/constitutional-charter.md#4-constitutional-design-rule). §4 criterion 1 mandates structural enforceability — discipline at the schemas/substrate layer, not procedural defense at the producer layer. Tier 1 (α/β/γ) satisfies the criterion fully (post-Q5). Tier 2 (δ, ζ) satisfies it partially: δ defers half to an operational RFC; ζ accepts a population-drift surface that per-record validation cannot catch. Tier 3 (ε) satisfies the criterion only procedurally. **The §4 criterion 1 alignment alone partitions Tier 1 above Tier 2 above Tier 3.**

### Finding 2 — Asymmetry: ε admits the §1 Thesis failure mode at substrate; structural candidates structurally cannot

Phase 3's secondary finding: ε admits silent collapse-to-confidence undetectably at substrate. The substrate accepts the producer's declared value; the §2.6 anti-pattern 2 detection mechanism (projection-replay byte-for-byte) does not catch a commit-time producer-internal collapse. **ε's structural surface is substantively equivalent to a confidence-only commitment under simplification pressure.** This is exactly the [§1 Thesis](../../charter/constitutional-charter.md#1-thesis) failure mode the entire §2.6 invariant exists to defend against. **ε is constitutionally inadequate as the sole resolution.**

ε remains structurally available as the *refinement* layer within ζ (where it is bounded around a structural baseline), but not as the sole derivation rule.

### Finding 3 — Asymmetry: α has the simplest structural form; β + γ pay additional structural costs without compensating discipline gain

Within Tier 1, α, β, γ are all Q5-dependent and structurally falsifiable post-Q5. The asymmetry is in structural simplicity:

- **α** counts Cat I roots — a count over a finite set of named record-class instances. The denominator is the chain's cardinality; the numerator is a subset of the same set. No additional sub-decisions.
- **β** counts edges in the provenance subgraph. The edge-weighting question (Phase 3 β's epistemic-separator finding) is a structural sub-decision deferred to operational specification. β's "fraction" is higher-resolution than α's "ratio" only at the cost of conflating short-and-deep with long-and-shallow topologies — Phase 2 β's tension.
- **γ** requires choosing a function `f` (Phase 3 γ's ambiguity-reducer finding) — a structural sub-decision deferred to operational specification.

α is the only Tier 1 candidate without additional sub-decisions deferred. β and γ admit structurally-deferred sub-decisions that compound the Q5-dependency. **α dominates β and γ on structural simplicity within Tier 1.**

### Finding 4 — Apparent trade-off that resolves: α's "limited resolution" is a feature at inception phase, not a defect

Phase 2 α's tension named α's limited resolution (a 3-Cat-I-root hypothesis admits only {0, 1/3, 2/3, 1} as possible values). Surface reading: β/γ admit finer-grained resolution and are therefore preferable.

Deeper reading: α's resolution is bounded by the cardinality of the supporting evidence set. **The "limited resolution" is honest about the limited evidence basis.** A hypothesis with 3 supporting observations *should* admit only 4 distinct independence values — claiming finer-grained resolution under β or γ is structurally over-claiming. Per the [inception-phase posture established at `§0023`](../../charter/decision-log.md) + [`§0027`](../../charter/decision-log.md): the system is maturing; structural commitments at this phase should NOT pre-commit to resolution finer than the underlying evidence supports.

The apparent β/γ-favoring trade-off resolves toward α: limited resolution is a feature, not a defect, at inception. **The trade-off resolves once the inception-phase posture is brought into the analysis.**

### Finding 5 — Apparent trade-off that resolves: ζ's "bridging" structural-vs-flexible is asymmetric — the substrate cannot validate the refinement's derivation

Phase 2 ζ's framing: hybrid baseline+refinement reconciles structural discipline (baseline) with operator flexibility (refinement). Surface reading: ζ is the best of both worlds.

Deeper reading: the refinement's "boundedness around the baseline" is a per-record substrate validation, not a population-level discipline. A producer who consistently refines upward (Phase 3 ζ's epistemic-separator finding) introduces population-level drift that the substrate cannot detect at the per-record commit boundary. **The substrate's discipline surface for the refinement layer is bounded-range only.** ζ's refinement layer has the same opacity-of-producer-derivation failure mode as ε, scoped to a bounded delta from the baseline. **The "bridging" framing overstates ζ's structural surface.**

This does not eliminate ζ as a candidate — the *baseline* layer carries Tier 1 structural discipline; the refinement adds a bounded operator surface. But Finding 5 demotes ζ relative to α-alone: ζ's additional structural surface (two values per record, refinement validation) carries cost without commensurate discipline gain at inception.

### Finding 6 — Asymmetry: δ vs ζ — δ's operational RFC dependency is open-ended; ζ's refinement surface is at least bounded

Within Tier 2, δ and ζ partition the partial-structural space differently:

- δ defers the formula's content to a yet-to-be-opened operational RFC. The operational RFC's existence, scope, and discipline are all unknown at Q3 closure. The operational RFC may itself surface further sub-decisions, compounding the partial-structurality.
- ζ inherits a known structural baseline (α/β/γ); the refinement layer is bounded-range only. The structural surface is enumerable at Q3 closure.

ζ's partial-structurality is **bounded and named**; δ's is **open-ended and procedurally promissory.** Within Tier 2, ζ-with-α-baseline dominates δ on structural discipline at Q3-closure-time.

### Finding 7 — Genuine trade-off: Q5 transitivity-half cascade vs operational-flexibility candidates

Tier 1 candidates require Q5 transitivity-half resolution to become fully operational. The cascade-trigger discipline per [`§0015 → §0016`](../../charter/decision-log.md) + [`§0020 → §0021`](../../charter/decision-log.md) applies: opening Q5 transitivity-half as discussion is a known structural commitment.

Tier 2 + Tier 3 candidates do NOT require Q5 (δ if operationally-specified non-graph; ε entirely). They permit Q3 closure without Q5 cascade.

The trade-off is structural-vs-procedural readiness: Tier 1 is more structurally disciplined but requires an additional pre-Gate; Tier 2/3 close faster but accept structural compromises. **This is a genuine trade-off — the committee weights structural discipline vs procedural simplicity.**

At inception phase, the precedent established at [`§0014`](../../charter/decision-log.md) + [`§0019`](../../charter/decision-log.md) is that lazy pre-Gates are not avoided; they are opened minimally and discharged in sequence. **The precedent favors Tier 1 + Q5 cascade.**

### Finding 8 — Carry-forward: Q5 transitivity-half cascade trigger fires under Tier 1 resolution

If Q3 closure adopts α (or β or γ), Q5 transitivity-half cascade-triggers per [`§0015 → §0016`](../../charter/decision-log.md) + [`§0020 → §0021`](../../charter/decision-log.md) precedent. The cascade is anticipated in [`§0132`](../../charter/decision-log.md) Methodological Observation 1; Phase 4 confirms it fires under any Tier 1 selection. **Q5 transitivity-half cascade is the structural consequence of the recommendation.**

If Q3 closure adopts ε (Tier 3) or δ-with-non-graph-formula (Tier 2 sub-variant), Q5 transitivity-half does NOT cascade — but the §4 criterion 1 discipline gap (Finding 1 + Finding 2) is not addressed.

### Finding 9 — Composition with Q2-A.2 four-subtype Cat III taxonomy is clean for all Tier 1 candidates

[`§0010`](../../charter/decision-log.md) Q2-A.2 established four concrete Cat III subtypes (`BehavioralCluster`, `AutomationGroup`, `CampaignHypothesis`, `CoordinationRing`) under the abstract `Hypothesis`. Q3 resolution applies at the abstract level; per-subtype divergence (different formulas per subtype) is structurally available but not required at inception. **Per-subtype divergence is forward-referenceable as a sub-RFC if empirical pressure surfaces it.** No candidate is constrained by Q2-A.2 composition.

## Phase 5 — Recommendation

The discussion phase recommends adopting **Candidate α (source-count ratio over Cat I provenance roots)** as the formal definition of `evidential_independence` as a measurable quantity. The recommendation rests primarily on Finding 1 (Tier 1 is the only tier satisfying §4 criterion 1 structurally), supported by Finding 3 (α dominates β and γ on structural simplicity within Tier 1), Finding 4 (α's limited resolution is a feature, not a defect, at inception phase), and Finding 5 (ζ's bridging framing overstates its structural surface). Finding 2 (ε's substrate-level admittance of the §1 Thesis failure mode) eliminates ε from the resolution space. Finding 6 (δ's open-ended operational-RFC dependency) demotes δ relative to ζ; Finding 5 demotes ζ relative to α-alone at inception. The recommendation is **α alone**, not ζ-with-α-baseline, on grounds of constitutional minimalism per [CLAUDE.md §7](../../../.claude/CLAUDE.md): "Every new invariant must justify its non-redundancy with existing ones. ... Ceremony without behavioral consequence is rejected." ζ's refinement layer adds canonical-serialization-contract complexity without commensurate inception-phase discipline gain.

Per [§2.6 BC2](../../charter/constitutional-charter.md#26-evidential-independence-integrity), the recommendation lands in **meta-shape 1 (deterministic-from-pattern)**. The canonical-serialization-contract freezes α's formula directly; producer cannot influence the value; consumer-side projection-replay diff verifies byte-for-byte match against the substrate-committed value.

### What would reverse this recommendation

The recommendation flips or substantially changes if any of the following emerges:

- **The committee finds α's resolution bound (limited to {0, 1/n, 2/n, ..., 1} for n roots) operationally unworkable.** If consumers of `evidential_independence` consistently require finer resolution than α admits, β (edge-fraction) becomes preferable. The evidence-grounded test for this reversal: are there concrete inception-phase consumers whose threshold-tests under Layer B cannot operate on α's bounded value set?
- **Q5 transitivity-half resolves to non-transitive in a way that further constrains α's value distribution.** Under non-transitive Q5 ("only direct `influenced_by` edges count, not transitive closure"), α's "not reachable via influenced_by" predicate becomes coarser — more Cat I roots qualify as non-influenced. The resolution may flatten further; if flattening is severe, γ (which uses graph distance rather than root counts) becomes preferable.
- **Empirical implementation pressure surfaces operator refinement as required.** If producers consistently produce hypotheses where α's substrate-computed value does not match the producer's own assessment AND the discrepancy is structurally informative, ζ-with-α-baseline becomes the operational extension. The empirical pressure must be evidenced, not anticipated — per the [`§0022`](../../charter/decision-log.md) empirical-pressure-phase discipline.
- **The committee identifies an operational specification surface that fully restores δ's structural enforceability.** If a parallel RFC produces an operational specification document with §4 criterion 1-grade structural discipline, δ may overcome Finding 6's open-endedness and become competitive. The empirical evidence required: the operational RFC must exist at Q3 closure time.
- **A new candidate emerges in committee deliberation.** Committee extension during resolution is legitimate per [Q2-2's precedent (sub-resolution A.2 was a committee extension)](../../charter/decision-log.md). If a seventh candidate is proposed and grounded in Findings 1–9, the recommendation may be displaced.

### Implication for Q5 transitivity-half cascade

The recommended resolution **triggers Q5 transitivity-half cascade** per [`§0015 → §0016`](../../charter/decision-log.md) + [`§0020 → §0021`](../../charter/decision-log.md) precedent. α's "Cat I roots NOT reachable via any `influenced_by` edge" requires Q5's transitivity-half resolution to be operationally complete. The cascade was anticipated at [`§0132`](../../charter/decision-log.md) Methodological Observation 1; Phase 4 Finding 8 confirms it fires under the α recommendation.

Q5 transitivity-half opens as `discussion` at the Q3-resolution commit. The cascade-enactment-in-single-commit pattern per [`§0015`](../../charter/decision-log.md) (Q1 resolution + Q3 pre-Gate opening) and [`§0020`](../../charter/decision-log.md) (OMQ #2 resolution + OMQ #3 pre-Gate opening) is the precedent.

### Implication for Layer B follow-on RFC unblock

α's adoption produces the measurable quantity Layer B's deep criterion threshold-tests. Layer B's substantive content advances post-Q3-resolution + post-Q5-transitivity-resolution. **The two-cascade chain Q3 → Q5 → Layer B is the structural path forward.** Layer B's RFC status note will be updated at the Q5 transitivity-half resolution to reflect "Q3 + Q5 discharged; Layer B substantive content drafting opens".

The Layer B deep criterion form (Candidate B family from Q4 — evidence-staleness using α; Candidate C family — influence-saturation using α — or both) is a sub-decision at Layer B's substantive phase, not at Q3's resolution. α produces the quantity; Layer B selects how the quantity gates demotion.

### Implication for canonical-serialization-contract revision

α's adoption requires a canonical-serialization-contract revision per [`§0028`](../../charter/decision-log.md) + [`§0034`](../../charter/decision-log.md) — the contract must crystallize α's formula at the substrate-commit layer. The revision is a follow-on RFC at architecture-document layer, opening post-Q3-resolution and tracked separately from Q5 transitivity-half. The revision is not pre-Gate to Q3 closure; it is a structural follow-on per the [`§0024`](../../charter/decision-log.md) schemas-evolution-contract discipline.

---

## References

- [`docs/ontology/ontology.md` Open Questions](../../ontology/ontology.md) — Q3 source line.
- [Charter §2.6](../../charter/constitutional-charter.md#26-evidential-independence-integrity) frozen v0.6 — BC1 + BC2 are the structural surface Q3 lands in.
- [Charter §2.4](../../charter/constitutional-charter.md#24-inferential-influence-disclosure) frozen v0.5 — `influenced_by` chain structural surface; Q3 derivation rule may consume.
- [Charter §2.3](../../charter/constitutional-charter.md#23-provenance-integrity) frozen v0.4 — typed `subject_ref_*` chains terminating at Cat I; Q3 derivation rule may traverse.
- [`decision-log.md` §0020 OMQ #2-C](../../charter/decision-log.md) — decay via §2.5 lifecycle supersession; Q3 derivation rule operates on substrate snapshot.
- [`decision-log.md` §0021 OMQ #3-α](../../charter/decision-log.md) — substrate-time generation; Q3 derivation rule evaluated at commit.
- [`decision-log.md` §0034](../../charter/decision-log.md) — canonical-serialization-contract enforces paired-dimension requirement.
- [`decision-log.md` §0011 + §0129](../../charter/decision-log.md) — Layer B follow-on RFC contract; Q3 unblocks Layer B substantive content jointly with Q5.
- [`decision-log.md` §0132](../../charter/decision-log.md) — opens this RFC at discussion.
- [`ontology-revision-q3-independence`](../draft/ontology-revision-q3-independence.md) — the draft RFC this scratch supports.
- [`ontology-revision-layer-b-deep-criterion`](../draft/ontology-revision-layer-b-deep-criterion.md) — the downstream RFC Q3 unblocks.
- [`q3-evidence.md`](./q3-evidence.md) — the OTHER Q3 (subject reference polymorphism, §0016); naming-clash disambiguated by this file's `-independence` suffix.
