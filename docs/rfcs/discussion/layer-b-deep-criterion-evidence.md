# Layer B Deep Criterion — substantive content discussion evidence

**Status:** in-discussion. Not authoritative. Final resolution will be recorded in decision-log and Ontology document revision.

This scratch supports the substantive content phase of [`ontology-revision-layer-b-deep-criterion`](../draft/ontology-revision-layer-b-deep-criterion.md). The RFC was opened at [`§0011`](../../charter/decision-log.md) as a placeholder when Q4 (promotion → demotion criterion) resolved to the staged-combination form (Layer A AND Layer B). All four pre-Gate dependencies are now discharged:

- §2.4 frozen v0.5 at [`§0099`](../../charter/decision-log.md)
- §2.6 frozen v0.6 at [`§0129`](../../charter/decision-log.md)
- Q3 (formal definition of `evidential_independence`) resolved at [`§0133`](../../charter/decision-log.md): Candidate α (source-count ratio over Cat I provenance roots)
- Q5 (influence propagation) resolved fully at [`§0020`](../../charter/decision-log.md) (decay) + [`§0134`](../../charter/decision-log.md) (transitivity: Candidate τ — transitive closure with β-graph storage)

The substantive content question is what [`§0011`](../../charter/decision-log.md) deferred: which combination of Candidate B family from Q4 (evidence-staleness using `evidential_independence`) and/or Candidate C family from Q4 (influence-saturation using declared `influence`) constitutes Layer B's deep criterion, and how the chosen family or families compose within Layer B itself.

This is a strictly-framing scratch: Phase 1 names the question and dependency surface; Phase 2 enumerates candidate Layer B compositions. Phases 3+ (epistemic-skill application, comparison synthesis, recommendation) are drafted in a subsequent RFC commit when the discussion advances substantively, per the Q3 / Q5 cadence precedent.

---

## Phase 1 — Scope and dependencies

### The question

[`decision-log.md` §0011](../../charter/decision-log.md) (Q4 resolution) adopted the staged-combination form:

> "DEMOTE-CANDIDATE(H) = Layer A(H) AND Layer B(H)"

where Layer A is Candidate A (time-based cadence gate, operational on substrate timestamps today) and Layer B is the deferred deep criterion on `evidential independence` and/or declared `influence`. The outer AND-composition between Layer A and Layer B is committee-resolved per [`§0011`](../../charter/decision-log.md); reversal conditions to OR-composition are recorded in [`q4-evidence.md` §Phase 5](./q4-evidence.md) for future revision if Layer A's gate proves too restrictive in practice.

Layer B itself is unresolved. The substantive content question is:

1. **Family selection** — does Layer B's deep criterion operate on (a) evidence-staleness alone (Candidate B family); (b) influence-saturation alone (Candidate C family); or (c) both?
2. **Inner composition (if both)** — disjunctive (B OR C), conjunctive (B AND C), or staged (B then C, or C then B)?
3. **Operational forms** — given Q3-α + Q5-τ, what are the specific structural formulas for the B-family freshness metric and the C-family saturation metric?
4. **Parameter values** — threshold T for B-family; ratio K for C-family. Per the [Q3 Phase 5 "What are the parameter values" open question](./q3-independence-evidence.md), parameter values are themselves deferrable to a further RFC if the structural form is the more urgent commitment.
5. **Per-subtype vs uniform parameters under [`§0010`](../../charter/decision-log.md) Q2-A.2** — Layer B's parameters may live at the abstract `Hypothesis` level or per-concrete-subtype.

This RFC's substantive content addresses questions 1 + 2 + 3 at form level. Questions 4 + 5 are sub-decisions deferrable to follow-on operational-specification RFCs per the form-vs-parameter discipline established by Q4 [`§0011`](../../charter/decision-log.md) Methodological Observation 1 (form-level convergence as a valid Ontology-RFC convergence shape).

### In scope

- The structural form of Layer B's deep criterion: which family/families and which inner composition.
- The operational forms of B-family and C-family under Q3-α + Q5-τ — how `evidential_independence` and the transitive `influenced_by` graph feed the freshness metric and the saturation metric structurally.
- The interaction with [`§0011`](../../charter/decision-log.md) outer AND-composition: Layer B's inner structure must not implicitly invert the outer AND.
- The interaction with [§2.5](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) frozen v0.3: Layer B's firing produces a `demotion` lifecycle event per §2.5 structural commitments; Layer B's structural form is the input to the firing predicate.

### Out of scope

- **Layer A.** Resolved at [`§0011`](../../charter/decision-log.md) — time-based cadence gate. Layer A's parameter N is itself deferrable to operational specification; not within Layer B's scope.
- **Outer AND-vs-OR composition between Layer A and Layer B.** Resolved at [`§0011`](../../charter/decision-log.md) as AND with five enumerated reversal conditions. Layer B's substantive content does not re-open this composition.
- **Specific parameter values T (B-family) and K (C-family).** Deferrable to follow-on operational-specification RFC per the form-vs-parameter discipline.
- **Per-subtype vs uniform parameters under Q2-A.2.** Forward-referenceable per [Q3 Phase 4 Finding 9](./q3-independence-evidence.md) + [Q5 Phase 4 Finding 7](./q5-transitivity-evidence.md) — inception-phase default is uniform at the abstract `Hypothesis` level; per-subtype variant is operational-specification follow-on.
- **Canonical-serialization-contract revision.** Opens as architecture-document RFC follow-on per [`§0133`](../../charter/decision-log.md) + [`§0134`](../../charter/decision-log.md) schedule; not pre-Gate to Layer B substantive content.

### Resolved dependencies (structural ground present)

| Anchor | What it commits | How Layer B consumes it |
|---|---|---|
| [§2.4](../../charter/constitutional-charter.md#24-inferential-influence-disclosure) v0.5 | `influenced_by` chain declared at formation time | C-family saturation metric reads the chain |
| [§2.5](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) v0.3 | Hypothesis lifecycle events including `demotion` | Layer B's firing produces a demotion event |
| [§2.6](../../charter/constitutional-charter.md#26-evidential-independence-integrity) v0.6 | `evidential_independence` paired with `confidence`; substrate-time-generated per [`§0021`](../../charter/decision-log.md) | B-family freshness metric reads `evidential_independence` |
| [`§0011`](../../charter/decision-log.md) | Staged-combination form: Layer A AND Layer B | Layer B is the second layer; this RFC fills in its structure |
| [`§0020`](../../charter/decision-log.md) OMQ #2-C | Decay via §2.5 lifecycle supersession | Layer B operates on substrate-state-at-evaluation-time; supersession reading is orthogonal |
| [`§0021`](../../charter/decision-log.md) OMQ #3-α | Substrate-time generation | Both B-family freshness and C-family saturation reduce to substrate-committed values + queries over the substrate state |
| [`§0133`](../../charter/decision-log.md) Q3-α | `evidential_independence = (Cat I roots NOT reachable via influenced_by from promoted H) / (total Cat I roots)` | B-family freshness metric IS the α value over recent assertions referencing the hypothesis |
| [`§0134`](../../charter/decision-log.md) Q5-τ | Transitive closure of `influenced_by` with β-graph storage; Cat II structural transmission commitment | C-family saturation metric IS a fraction over recent assertions whose transitive `influenced_by` includes the hypothesis |

### Open dependencies (not blocking; assessed at substantive deliberation)

| Open question | Why potentially relevant | Default disposition |
|---|---|---|
| Specific operational form of "recent assertions" — a fixed time window, a fixed count window, or a hybrid | Both B-family and C-family operate on a "recent" subset of substrate-committed assertions; the specific window definition is operational | Forward-referenceable per the form-vs-parameter discipline; substantive content can resolve the form-level commitment without crystallizing the window's specific shape |
| Specific operational form of "promoted hypothesis" enumeration | The criteria evaluate against the set of currently-promoted hypotheses; the enumeration is operational | Forward-referenceable; §2.5 lifecycle supersession reading per [`§0020`](../../charter/decision-log.md) governs the enumeration's projection-time form |

### Procedural posture

This RFC's substantive content is at `discussion` status (advanced from placeholder per [`§0134`](../../charter/decision-log.md) consequences). Phase 1 (this section) names the dependency surface. Phase 2 (below) enumerates candidate Layer B compositions. Phase 3+ is drafted in subsequent commits per the Q3 / Q5 cadence precedent. Resolution lands at a future `decision-log` entry that records Layer B's deep criterion form.

---

## Phase 2 — Candidate Layer B compositions

Five candidate compositions enumerated. Each is structurally a Layer B internal form; the outer Layer A AND Layer B composition is unchanged per [`§0011`](../../charter/decision-log.md).

The operational sketches use Q3-α + Q5-τ as the structural ground. The B-family freshness metric is the α-value-trend over recent assertions referencing the hypothesis; the C-family saturation metric is the fraction of recent substrate assertions with the hypothesis in their transitive `influenced_by` chain (per Q5-τ).

### Candidate L-B — Evidence-staleness alone

**Structural form:** `Layer B(H) := freshness_B(H) < T_B`

where `freshness_B(H)` is the average `evidential_independence` (per Q3-α) over the most recent N assertions whose `subject_ref_hypothesis` is H (or whose transitive `influenced_by` per Q5-τ includes H — the specific window subset is an operational sub-decision).

**Failure mode addressed:** the [§1 Thesis](../../charter/constitutional-charter.md#1-thesis)' first naming — "confidence in inferences inflates without proportional increase in independent evidence". When the hypothesis's recent supporting assertions have low α (their own evidence is dominated by influence chains, often back to H itself), the hypothesis has lost independent support.

**§2.4/§2.6/§4 consistency:** B-family reads `evidential_independence` via §2.6; reads recency via §2.5 lifecycle events; reads provenance via §2.3 + §2.4. All inputs are substrate artifacts. §4 criterion 1 satisfied.

**One-line tension:** B addresses the loss-of-independent-support failure mode directly, but is silent on the saturation failure mode — a hypothesis with stable α on its few recent supporters can still dominate the substrate at large.

### Candidate L-C — Influence-saturation alone

**Structural form:** `Layer B(H) := saturation_C(H) > K_C`

where `saturation_C(H)` is the fraction of the most recent N substrate assertions whose transitive `influenced_by` chain (per Q5-τ) includes H, over N. The denominator is the recent-substrate window; the numerator is the count where H appears in the transitive influence chain.

**Failure mode addressed:** the [§1 Thesis](../../charter/constitutional-charter.md#1-thesis)' second naming — "promoted hypotheses re-enter the system as enrichment and silently reinforce themselves". When H's transitive influence dominates a structurally-bounded fraction of recent substrate activity, H has become a fixed point of inferential gravity.

**§2.4/§2.6/§4 consistency:** C-family reads transitive `influenced_by` via §2.4 + Q5-τ; reads recency via §2.5 lifecycle events. All substrate artifacts. §4 criterion 1 satisfied.

**One-line tension:** C addresses the saturation failure mode directly, but is silent on the staleness failure mode — a hypothesis that is supported by recent fresh-and-independent evidence can still be saturating if it's widely used as enrichment.

### Candidate L-BC-AND — Conjunctive (B AND C)

**Structural form:** `Layer B(H) := (freshness_B(H) < T_B) AND (saturation_C(H) > K_C)`

**Failure mode addressed:** the intersection of both §1 Thesis naming halves — Layer B fires only when BOTH the staleness AND the saturation failure modes are present. High bar; protects against single-criterion false positives.

**§2.4/§2.6/§4 consistency:** inherits from L-B and L-C.

**One-line tension:** the AND composition admits a known failure mode: a hypothesis exhibiting strong staleness BUT moderate saturation (or vice versa) does not fire Layer B even though one §1 Thesis half is violated. The AND requires both halves to fire simultaneously; the §1 Thesis names them as independent failure modes either of which is structurally adequate to demote.

### Candidate L-BC-OR — Disjunctive (B OR C)

**Structural form:** `Layer B(H) := (freshness_B(H) < T_B) OR (saturation_C(H) > K_C)`

**Failure mode addressed:** the union of both §1 Thesis naming halves — Layer B fires when EITHER staleness OR saturation is present. Maximum coverage of the structurally-named failure modes.

**§2.4/§2.6/§4 consistency:** inherits from L-B and L-C.

**One-line tension:** L-BC-OR is maximally inclusive at the cost of evaluating both criteria on every Layer A firing. The cost is bounded — both freshness_B and saturation_C reduce to substrate queries; neither requires graph closure beyond what Q5-τ already commits. The trade-off relative to L-B / L-C alone is "evaluate two metrics vs one".

### Candidate L-BC-staged — Sequential (B then C, or C then B)

**Structural form:** `Layer B(H) := if freshness_B(H) < T_B then TRUE; else if saturation_C(H) > K_C then TRUE; else FALSE` (or the reverse order — evaluate C first, then B).

**Failure mode addressed:** same as L-BC-OR — the union of both naming halves. The structural firing condition is identical to L-BC-OR; the staging is a cost-optimization variant (evaluate the cheaper metric first; short-circuit on firing).

**§2.4/§2.6/§4 consistency:** inherits from L-BC-OR.

**One-line tension:** L-BC-staged is operationally identical to L-BC-OR on the firing predicate, but commits to an evaluation order at the canonical-serialization-contract layer. The order itself is an operational sub-decision (which metric is cheaper depends on substrate topology), and committing to an order may foreclose future cost-optimization. Distinct from L-BC-OR primarily at the operational-specification layer, not the structural layer.

### Asymmetries surfaced

Three asymmetries partition the candidate space and will likely organize substantive deliberation:

- **§1 Thesis-coverage asymmetry:** L-B covers only the staleness half; L-C covers only the saturation half; L-BC-AND covers the intersection; L-BC-OR and L-BC-staged cover the union. The §1 Thesis names BOTH halves as independent failure modes; **maximum coverage favors L-BC-OR / L-BC-staged.**
- **Single-criterion false-positive resistance asymmetry:** L-BC-AND is most resistant (requires both criteria to fire); L-BC-OR / L-BC-staged are least resistant (either alone fires). If parameter calibration (T_B, K_C) is operationally uncertain at inception, false-positive resistance is a discipline concern.
- **Structural-vs-operational separation asymmetry:** L-B / L-C / L-BC-AND / L-BC-OR are structural form commitments — Layer B's firing predicate is committee-fixed at this RFC's resolution. L-BC-staged additionally commits to an operational evaluation order — partly structural, partly operational. **L-BC-staged blurs the form-vs-parameter discipline established at [`§0011`](../../charter/decision-log.md) Methodological Observation 1; L-BC-OR keeps the boundary clean.**

These asymmetries are recorded for use by substantive deliberation; they are NOT selections.

---

## Phases 3+ — Deferred to substantive deliberation

The following phases are drafted in subsequent RFC commits when the committee deliberates Layer B substantively:

- **Phase 3 — Epistemic-skill application.** Apply [`falsifiability-check`](../../../.claude/skills/epistemic/falsifiability-check/SKILL.md), [`epistemic-separator`](../../../.claude/skills/epistemic/epistemic-separator/SKILL.md), and [`ambiguity-reducer`](../../../.claude/skills/epistemic/ambiguity-reducer/SKILL.md) to each candidate per the Q3 / Q5 Phase 3 precedent.
- **Phase 4 — Comparison synthesis.** Rank candidates against §1 Thesis coverage, single-criterion false-positive resistance, structural-vs-operational discipline, and parameter-deferral compatibility.
- **Phase 5 — Recommendation.** Single-candidate recommendation with explicit dissent surface + parameter-deferral implication + canonical-serialization-contract revision contribution. Resolution recorded at a future decision-log entry that closes Layer B's substantive content question and unblocks the final Layer B canonical-serialization-contract crystallization.

---

## References

- [Charter §1 Thesis](../../charter/constitutional-charter.md#1-thesis) — names both failure modes Layer B defends against.
- [Charter §2.4](../../charter/constitutional-charter.md#24-inferential-influence-disclosure) v0.5 — `influenced_by` structural surface.
- [Charter §2.5](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) v0.3 — `demotion` lifecycle event; Layer B firing produces it.
- [Charter §2.6](../../charter/constitutional-charter.md#26-evidential-independence-integrity) v0.6 — `evidential_independence` paired-dimension surface.
- [`decision-log.md` §0011](../../charter/decision-log.md) — Q4 resolution; opens Layer B placeholder.
- [`decision-log.md` §0020](../../charter/decision-log.md) — Q5 decay half resolved.
- [`decision-log.md` §0021](../../charter/decision-log.md) — substrate-time generation.
- [`decision-log.md` §0133](../../charter/decision-log.md) — Q3-α resolution; α is the B-family input.
- [`decision-log.md` §0134](../../charter/decision-log.md) — Q5-τ transitivity-half resolution; τ is the C-family input. Layer B substantive content opens per consequences.
- [`q3-independence-evidence.md`](./q3-independence-evidence.md) — Q3 discussion-phase evidence (precedent for Phase structure).
- [`q4-evidence.md`](./q4-evidence.md) — Q4 discussion-phase evidence (original B-family and C-family enumeration).
- [`q5-transitivity-evidence.md`](./q5-transitivity-evidence.md) — Q5 discussion-phase evidence.
- [`ontology-revision-layer-b-deep-criterion`](../draft/ontology-revision-layer-b-deep-criterion.md) — the draft RFC this scratch supports.
