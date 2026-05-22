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

## Phases 3+ — Deferred to substantive deliberation

The following phases are drafted in subsequent RFC commits when the committee deliberates Q3 substantively:

- **Phase 3 — Epistemic-skill application.** Apply [`falsifiability-check`](../../../.claude/skills/epistemic/falsifiability-check/SKILL.md), [`category-separation`](../../../.claude/skills/epistemic/category-separation/SKILL.md), [`vocabulary-discipline`](../../../.claude/skills/ontology/vocabulary-discipline/SKILL.md), and [`ambiguity-control`](../../../.claude/skills/epistemic/ambiguity-control/SKILL.md) to each candidate.
- **Phase 4 — Comparison synthesis.** Rank candidates against constitutional discipline (§4 criterion 1 + §1 Thesis defense), operational cost, and structural simplicity.
- **Phase 5 — Recommendation.** Single-candidate recommendation with explicit dissent surface. Resolution recorded at a future decision-log entry that closes ontology.md Open Question 3.

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
