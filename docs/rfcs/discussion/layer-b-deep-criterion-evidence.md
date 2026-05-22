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

## Phase 3 — Apply epistemic skills

Per the Q3 / Q5 Phase 3 precedent, three skills ([`falsifiability-check`](../../../.claude/skills/epistemic/falsifiability-check/SKILL.md), [`epistemic-separator`](../../../.claude/skills/epistemic/epistemic-separator/SKILL.md), [`ambiguity-reducer`](../../../.claude/skills/epistemic/ambiguity-reducer/SKILL.md)) applied to each of the five candidates. Five candidates × three skills = 15 cells.

### Candidate propositions

- **L-B proposition:** "A promoted hypothesis H is demotion-candidate when `freshness_B(H) < T_B`, where `freshness_B(H)` is the average `evidential_independence` (per Q3-α) over the most recent N assertions whose transitive `influenced_by` (per Q5-τ) includes H."
- **L-C proposition:** "A promoted hypothesis H is demotion-candidate when `saturation_C(H) > K_C`, where `saturation_C(H)` is the fraction of the most recent N substrate assertions whose transitive `influenced_by` (per Q5-τ) includes H, divided by N."
- **L-BC-AND proposition:** "Layer B(H) fires iff `(freshness_B(H) < T_B) AND (saturation_C(H) > K_C)`."
- **L-BC-OR proposition:** "Layer B(H) fires iff `(freshness_B(H) < T_B) OR (saturation_C(H) > K_C)`."
- **L-BC-staged proposition:** structurally L-BC-OR's firing predicate + canonical-serialization-contract commitment to an evaluation order (one of B-then-C or C-then-B).

### 5 × 3 matrix — candidate × skill

| Candidate | `falsifiability-check` | `epistemic-separator` | `ambiguity-reducer` |
|---|---|---|---|
| **L-B — Evidence-staleness alone** | §1.1: violation = promoted H with `freshness_B(H) < T_B` not in demotion-candidate set. §1.2: third party recomputes freshness_B from substrate (α values per [`§0133`](../../charter/decision-log.md) + recent-window query). §1.3: reduces to averaging substrate-committed α values over a structurally-named window — fully substrate. §1.4: clean. **Verdict: passes today at form level; parameter form (window N, threshold T_B) deferrable to operational specification per form-vs-parameter discipline.** | Inputs: α values + recent-window subset. **Risk:** the "most recent N assertions referencing H" must use the transitive reading per Q5-τ (per [`§0134`](../../charter/decision-log.md)) to avoid Cat-II-mediated invisibility — an assertion whose Cat II input was H-influenced is structurally referencing H. Under Q5-τ this is automatic; the structural reading is already in place. **Verdict: clean per category boundary; Q5-τ resolution structurally prevents the Cat-II-mediated invisibility failure mode.** No new commitment needed — §0134 already supplies. | Terms: `freshness`, `recent`, `threshold T_B`. Watchlist scan: `freshness` is advisory (the metric is operationally well-defined as average α; the prose may use the bare noun acceptably). `recent` requires operationalization (window form — fixed time, fixed count, hybrid) — Response 3 (raise as operational-specification open question). **Verdict: minor carry-forward on `recent` window form.** Vocabulary clean at structural surface. |
| **L-C — Influence-saturation alone** | §1.1: violation = promoted H with `saturation_C(H) > K_C` not in demotion-candidate set. §1.2: third party recomputes saturation_C from substrate (transitive `influenced_by` per Q5-τ + β-graph cached closures + recent-window query). §1.3: reduces to counting substrate-committed transitive closures over a structurally-named window — fully substrate (β-graph storage from [`§0134`](../../charter/decision-log.md) makes the closure queries amortized O(N) for window size N). §1.4: clean. **Verdict: passes today at form level; parameter form (window N, ratio K_C) deferrable per form-vs-parameter discipline.** | Inputs: transitive `influenced_by` counts + recent-window subset. **Risk:** if the count includes H's OWN enrichment outputs (assertions formed BY H as enrichment context), H's own activity inflates its saturation — a self-reinforcing loop the §1 Thesis names. The recent-window subset must structurally exclude H's own enrichment outputs from the saturation denominator. Under §2.4 v0.5, assertions formed under H's influence are structurally identifiable via the `influenced_by` declaration; the exclusion is structurally available. **Verdict: clean per category boundary, with documented STRUCTURAL EXCLUSION COMMITMENT — L-C's saturation denominator must exclude H's own enrichment outputs.** This is the mirror of [`q4-evidence.md` Phase 3 Finding 6](./q4-evidence.md) (B-family's circularity-exclusion requirement); under Q3-α + Q5-τ, the analogous C-family commitment surfaces here. | Terms: `saturation`, `recent`, `fraction`, `ratio K_C`. Watchlist scan: `saturation` advisory (the metric is operationally well-defined as a ratio). `recent` same as L-B. **Verdict: minor carry-forward on `recent` + the `saturation` denominator's structural-exclusion specification.** Vocabulary clean at structural surface; the structural-exclusion commitment surfaced under `epistemic-separator` is the load-bearing operationalization detail. |
| **L-BC-AND — Conjunctive** | §1.1: violation = promoted H with both predicates firing + H not in demotion set. §1.2 + §1.3: inherit from L-B + L-C. §1.4: clean. **Verdict: passes — falsifiability is STRICTLY stronger than either L-B or L-C alone (requires both predicates to be tested).** Note: AND-conjunction makes the *violation* test harder to satisfy (a violation requires both metrics to fire AND H to not be demoted), making the falsification mechanism less sensitive to single-criterion failures. | Inherits both L-B and L-C risk surfaces — Q5-τ transitive reading (already satisfied) + L-C structural-exclusion commitment (new commitment required). **Verdict: clean conditionally on both commitments.** No new structural risk beyond L-B + L-C composition. | Inherits both L-B and L-C terminology. **Verdict: same carry-forwards as L-B + L-C; no new ambiguity-discipline concerns at AND-composition.** |
| **L-BC-OR — Disjunctive** | §1.1: violation = promoted H with either predicate firing + H not in demotion set. §1.2 + §1.3: inherit. §1.4: clean. **Verdict: passes — falsifiability is STRICTLY weaker than L-BC-AND (single-metric firing suffices for violation test), but matches L-B / L-C single-family candidates on per-metric falsification sensitivity.** | Inherits both L-B and L-C risk surfaces — same commitments required as L-BC-AND. **Verdict: clean conditionally on both commitments.** | Inherits both L-B and L-C terminology. **Verdict: same carry-forwards.** No new ambiguity-discipline concerns at OR-composition. |
| **L-BC-staged — Sequential (B-then-C or C-then-B)** | Structurally identical to L-BC-OR on the firing predicate; §1.1–§1.4 all inherit from L-BC-OR. **Verdict: passes — structurally equivalent to L-BC-OR.** | Inherits L-BC-OR's risk surface + adds the evaluation-order question (which metric is cheaper depends on substrate topology). **Verdict: clean per category boundary; the staging order is an operational sub-decision that does NOT add structural risk beyond L-BC-OR.** | Inherits L-BC-OR terminology + adds the staging order specification. The order itself is a structural commitment at the canonical-serialization-contract layer per L-BC-staged's proposition — Response 3 (the order is itself a sub-decision the resolution must commit to, or defer to operational specification with an explicit deferral marker). **Verdict: ambiguity-flagged on the evaluation-order specification.** The structural sub-decision is in scope at this RFC's resolution; deferring it would partially blur the form-vs-parameter discipline. |

### Most consequential epistemic finding across the 15 cells

**Primary finding — L-C structural-exclusion commitment is the mirror of Q4 Phase 3 Finding 6 (B-family's circularity exclusion).** Phase 3 L-C's `epistemic-separator` finding surfaces a new structural commitment: L-C's saturation denominator must structurally exclude H's own enrichment outputs from the recent-window subset. Under §2.4 v0.5, the exclusion is structurally available — assertions formed under H's influence are structurally identifiable via the `influenced_by` declaration. **Q4 Phase 3 Finding 6 surfaced the analogous requirement for the B-family at Q4-1 deliberation; that finding was structurally resolved by §2.4's `influenced_by` chain commitment + Q3-α + Q5-τ giving α full transitive reach. The L-C commitment is the C-family's analogue: §2.4 supplies the exclusion mechanism; the resolution must commit to applying it.**

**Secondary finding — L-B's transitive-reading requirement is already satisfied by §0134.** Phase 3 L-B's `epistemic-separator` finding surfaces that L-B's freshness window must use the Q5-τ transitive reading to avoid Cat-II-mediated invisibility. Per [`§0134`](../../charter/decision-log.md) resolution (Candidate τ + Cat II structural transmission commitment), the transitive reading is structural; no new commitment is needed. The secondary finding is anti-finding — it records a structural commitment that does NOT need to be added because §0134 already supplies it. This validates the two-cascade chain Q3 → Q5 → Layer B's discharge: Q5-τ's selection at §0134 was load-bearing for Layer B's structural completeness.

**Tertiary finding — falsifiability sensitivity asymmetry between L-BC-AND and L-BC-OR.** Per Phase 3 L-BC-AND / L-BC-OR `falsifiability-check`: AND-conjunction makes the violation test STRICTLY stronger (requires both metrics to fire) but reduces falsification sensitivity to single-metric failures. OR-disjunction makes the violation test STRICTLY weaker but matches the per-metric falsification sensitivity of single-family candidates. **The falsifiability asymmetry is not decisive in either direction — both pass — but it surfaces a structural feature relevant to parameter calibration discipline at the operational specification phase: under L-BC-AND, conservative T_B + K_C calibration may produce few-to-no violations; under L-BC-OR, even loose calibration produces violations on either metric independently.**

### Calibration carry-forward to future Ontology RFCs

Layer B-1 confirms and extends prior Q's calibrations:

- **Confirmed: falsifiability §1.3 (operationalization) does most of the work on substrate-touching propositions.** All five candidates decide at §1.3 cleanly post-Q3-α + Q5-τ; the form-vs-parameter discipline means parameter form is deferred without violating §1.3.
- **Confirmed: ambiguity-reducer surfaces residual carry-forwards that are themselves structural deferrals.** The `recent` window form is a Response-3 case (deferred to operational specification). L-BC-staged's evaluation order is a structural sub-decision.
- **Extended: intra-category failure-mode catalogue now FIVE patterns deep.** Q2-1 (flattening) + Q4-1 (B-family circularity) + Q3-1 (opacity of producer-side derivation) + Q5-1 (Cat-II-mediated structural invisibility) + Layer-B-1 (**C-family enrichment-output inclusion** — the structural mirror of Q4-1's B-family circularity, surfaced for C). The catalogue continues to be structural rather than enumerative; each new instance reflects a different surface form of the producer/substrate boundary discipline.
- **New observation: structural-mirror discipline.** Q4-1's Finding 6 (B-family exclusion) and Layer-B-1's Finding 1 (L-C exclusion commitment) are structural mirrors — each candidate family within a multi-family composition admits its own structural-exclusion question. **Future multi-family Ontology RFCs should anticipate per-family structural-exclusion requirements and surface them explicitly at Phase 3.**

## Phase 4 — Comparison synthesis

Findings synthesized from Phase 1 (dependency surface), Phase 2 (five-candidate enumeration), and Phase 3 (15-cell epistemic-skill matrix). Classified as **asymmetry** / **apparent trade-off that resolves** / **genuine trade-off** / **tension**. Numbered in order of consequence.

### Finding 1 — Asymmetry: maximum §1 Thesis coverage favors L-BC-OR and L-BC-staged

[Charter §1 Thesis](../../charter/constitutional-charter.md#1-thesis) names TWO failure modes: (a) "confidence in inferences inflates without proportional increase in independent evidence" (the B-family failure mode); (b) "promoted hypotheses re-enter the system as enrichment and silently reinforce themselves" (the C-family failure mode). The Thesis treats them as INDEPENDENT — either alone is sufficient to violate the epistemic-integrity commitment.

- L-B alone: defends against (a); silent on (b). A hypothesis dominating the substrate but supported by fresh independent evidence is structurally invisible to L-B.
- L-C alone: defends against (b); silent on (a). A hypothesis with run-out independent support but limited downstream saturation is structurally invisible to L-C.
- L-BC-AND: defends only the INTERSECTION — a hypothesis violating (a) but not (b) (or vice versa) is structurally undefended.
- L-BC-OR / L-BC-staged: defend the UNION — either failure mode alone fires Layer B.

**The Thesis names independent failure modes; structurally-complete defense requires UNION coverage.** L-BC-OR / L-BC-staged are the only candidates that fully cover both Thesis halves. **This is the load-bearing constitutional asymmetry.**

### Finding 2 — Asymmetry: L-BC-AND systematically under-defends

Per Finding 1, L-BC-AND requires both halves to fire simultaneously. Concretely: a hypothesis exhibiting strong saturation (clearly violating Thesis half (b)) but with moderate per-assertion α values (not yet violating Thesis half (a)) is NOT demotion-candidate under L-BC-AND. The hypothesis remains promoted despite the (b) violation. **L-BC-AND admits both single-half violations as structurally-undefended states.** This is not a hypothetical concern: the Thesis explicitly treats the two halves as independent, and L-BC-AND's structural form contradicts that independence.

### Finding 3 — Apparent trade-off that resolves: L-BC-staged is structurally identical to L-BC-OR

Per Phase 3 L-BC-staged `falsifiability-check`: the firing predicate is identical to L-BC-OR; only the evaluation order is committed. The staging is an OPERATIONAL sub-decision (which metric is cheaper to evaluate first depends on substrate topology), not a STRUCTURAL one.

Per the form-vs-parameter discipline established at [`§0011`](../../charter/decision-log.md) Methodological Observation 1, the evaluation order belongs at the operational-specification layer, NOT at the structural-form layer. **L-BC-staged elevates an operational decision to the form layer; L-BC-OR keeps the form/parameter boundary clean.** The apparent operational benefit of L-BC-staged is achievable under L-BC-OR via canonical-serialization-contract or operational-specification follow-on RFC without committing to staging at the form layer.

### Finding 4 — Genuine trade-off: L-BC-OR false-positive resistance vs §1 Thesis coverage

The principal genuine trade-off in the candidate space: L-BC-AND offers stronger false-positive resistance (requires both metrics to fire); L-BC-OR offers stronger §1 Thesis coverage (either metric firing suffices). Finding 1 + Finding 2 resolve the trade-off toward L-BC-OR on constitutional grounds.

The false-positive resistance concern is addressable through parameter calibration: under L-BC-OR with conservative T_B (low — only fire when freshness is severely depleted) and K_C (high — only fire when saturation is overwhelming), false positives are bounded. The form-vs-parameter discipline permits empirical calibration at the operational-specification phase per [`§0022`](../../charter/decision-log.md) empirical-pressure-phase discipline. **L-BC-OR with parameter calibration is the path that satisfies both §1 Thesis coverage AND false-positive resistance; L-BC-AND structurally forecloses Thesis coverage on grounds that parameter calibration cannot recover.**

### Finding 5 — L-C structural-exclusion commitment is constitutional, not operational

Per Phase 3 L-C `epistemic-separator` finding: L-C's saturation denominator must structurally exclude H's own enrichment outputs from the recent-window subset. This is the STRUCTURAL mirror of Q4 Phase 3 Finding 6 (B-family's circularity exclusion). The exclusion is structurally available via §2.4 v0.5's `influenced_by` chain commitment; the resolution must explicitly commit to applying it.

The commitment is constitutional, not operational: without it, L-C systematically inflates saturation_C through H's own activity, structurally mis-firing Layer B on a hypothesis that is merely PRODUCTIVE rather than over-saturating. **Any candidate including L-C (L-C, L-BC-AND, L-BC-OR, L-BC-staged) inherits this commitment.**

### Finding 6 — L-B's transitive-reading requirement is pre-satisfied by §0134

Per Phase 3 L-B `epistemic-separator` finding: L-B's freshness window requires the transitive reading per Q5-τ. Per [`§0134`](../../charter/decision-log.md), the transitive reading is structural — Cat II structural transmission commitment makes Cat-II-mediated reference transitive automatically. **No new commitment is needed.** The finding records that the two-cascade chain Q3 → Q5 → Layer B's structural discharge was load-bearing: had Q5 resolved to δ (rejected per §0134), L-B would have admitted Cat-II-mediated invisibility as an open failure mode. Q5-τ closes this surface.

### Finding 7 — Carry-forward: parameter values T_B, K_C deferrable to operational-specification RFC

Per the form-vs-parameter discipline + Phase 2 candidate enumeration: the structural form (which candidate) is THIS RFC's resolution; the parameter values (T_B, K_C, window size N) are operational-specification follow-on. **The resolution commits to the form; the operational-specification RFC commits to parameter values.** This preserves the form-vs-parameter discipline established at [`§0011`](../../charter/decision-log.md) Methodological Observation 1.

### Finding 8 — Carry-forward: per-subtype application under Q2-A.2 forward-referenced

Per [Q3 Phase 4 Finding 9](./q3-independence-evidence.md) + [Q5 Phase 4 Finding 7](./q5-transitivity-evidence.md): the four concrete Cat III subtypes may surface per-subtype parameter divergence if empirical pressure surfaces; inception-phase default is uniform at the abstract `Hypothesis` level. **No candidate is constrained by Q2-A.2 composition at inception.** Layer B's per-subtype divergence is a downstream operational concern, not a structural-form concern.

### Finding 9 — Carry-forward: "recent assertions" window form deferrable

Per Phase 3 ambiguity-reducer findings across all candidates: the structural form of "recent assertions" (fixed time, fixed count, hybrid) is operational-specification follow-on. **All five candidates share this carry-forward; the choice is independent of the form decision this RFC resolves.**

### Finding 10 — Methodological observation: structural-mirror discipline for multi-family compositions

Q4-1's Finding 6 (B-family exclusion) and Layer-B-1's Finding 5 (L-C exclusion commitment) are STRUCTURAL MIRRORS — each family within a multi-family composition admits its own structural-exclusion question. The mirrors share a common form: the candidate family's metric must structurally exclude the hypothesis's own activity from the metric's input set to avoid self-reinforcing circularity. **Future multi-family Ontology RFCs should anticipate per-family structural-exclusion requirements and surface them explicitly at Phase 3.** This is the first explicit codification of the mirror-discipline pattern.

## Phase 5 — Recommendation

The discussion phase recommends adopting **Candidate L-BC-OR (disjunctive — `Layer B(H) := (freshness_B(H) < T_B) OR (saturation_C(H) > K_C)`)** as Layer B's deep criterion structural form. The recommendation rests on Findings 1 (only L-BC-OR / L-BC-staged structurally cover both §1 Thesis failure modes), 2 (L-BC-AND systematically under-defends), 3 (L-BC-staged blurs the form-vs-parameter discipline; L-BC-OR preserves it), and 4 (false-positive resistance concern is parameter-calibration-addressable; structural-coverage gap is not parameter-addressable).

The accompanying full demotion-candidacy predicate is:

> `DEMOTE-CANDIDATE(H) := Layer A(H) AND ((freshness_B(H) < T_B) OR (saturation_C(H) > K_C))`

where:
- `Layer A(H)` is the time-based cadence gate per [`§0011`](../../charter/decision-log.md).
- `freshness_B(H)` is the average `evidential_independence` (per Q3-α) over the most recent N assertions whose transitive `influenced_by` (per Q5-τ) includes H. Q5-τ transitive reading is structurally automatic per [`§0134`](../../charter/decision-log.md) Cat II structural transmission commitment.
- `saturation_C(H)` is the fraction `(count of recent assertions with H in transitive influenced_by, EXCLUDING H's own enrichment outputs) / N`. The structural exclusion is the L-C committee extension below.
- `T_B`, `K_C`, `N` are operational parameters deferred to follow-on operational-specification RFC per the form-vs-parameter discipline.

Two committee extensions accompanying the L-BC-OR selection:

- **L-C structural-exclusion commitment.** Per Finding 5: L-C's saturation denominator structurally excludes H's own enrichment outputs (assertions formed under H's influence per §2.4 v0.5) from the recent-window subset. This is the structural mirror of [Q4 Phase 3 Finding 6](./q4-evidence.md) for the C-family. The exclusion is structurally available via §2.4 chain inspection; the commitment records that the resolution applies it.
- **Form-vs-parameter discipline preserved.** Per Finding 7: the resolution commits to L-BC-OR as form; T_B / K_C / N values open as operational-specification follow-on RFC. This preserves the [`§0011`](../../charter/decision-log.md) Methodological Observation 1 discipline.

### What would reverse this recommendation

The recommendation flips or substantially changes if any of the following emerges:

- **Empirical implementation pressure shows L-BC-OR's false-positive rate is operationally unworkable even with conservative T_B / K_C calibration.** If, at the operational-specification phase, no parameter calibration produces acceptable false-positive rates, L-BC-AND becomes the operational fallback at the cost of Thesis-coverage incompleteness. The evidence-grounded test for this reversal: concrete profiling against an inception-phase substrate with measured violation rates under varying parameter values.
- **The committee weights single-criterion false-positive resistance above §1 Thesis coverage completeness.** Finding 4's trade-off resolves toward L-BC-OR on constitutional grounds, but the committee may judge that L-BC-AND's stronger resistance is the inception-phase preference. This requires committee evidence that L-BC-AND's structural under-coverage (Finding 2) is operationally non-problematic — a values judgment the discussion phase does not pre-decide.
- **L-BC-staged's evaluation-order benefit proves operationally substantial.** If profiling shows L-BC-OR's per-candidate evaluation cost is prohibitive AND L-BC-staged's order-commitment provides material reduction, L-BC-staged becomes preferred — but the staging order itself opens as a separate sub-RFC at the operational-specification phase, NOT bundled with L-BC-OR's form-level resolution.
- **A sixth candidate emerges in committee deliberation.** Committee extension during resolution is legitimate per Q2/Q4/Q3/Q5 precedent. If a sixth candidate is grounded in Findings 1–10 and surfaces a constitutional alignment L-BC-OR does not, the recommendation may be displaced.
- **L-C structural-exclusion commitment proves operationally insufficient.** If implementation pressure shows the exclusion mechanism (§2.4 chain inspection) admits residual self-reinforcement at the operational layer, an additional structural commitment may be required. The recommendation in that case shifts toward a strengthened L-C or toward L-B-alone (which sidesteps the C-family exclusion question).

### Implication for the §2.5 binding-text forward-reference discharge

The recommended resolution **fully discharges the [§2.5 binding text's Layer B forward-reference](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness)** per the [`§0011`](../../charter/decision-log.md) contract. With Layer B's structural form resolved to L-BC-OR + L-C structural-exclusion commitment, §2.5's reference to "a designated structural test on evidential independence and/or declared influence" gains a fully-specified referent. **The full two-cascade chain Q3 → Q5 → Layer B is structurally discharged.**

### Implication for canonical-serialization-contract revision

The resolution feeds the canonical-serialization-contract revision opening per [`§0133`](../../charter/decision-log.md) + [`§0134`](../../charter/decision-log.md) follow-on schedule. The revision now crystallizes:

- Q3-α's source-count-ratio formula (per §0133).
- Q5-τ's transitive closure with β-graph storage (per §0134).
- Layer B's L-BC-OR firing predicate + L-C structural-exclusion mechanism (per this resolution).

The canonical-serialization-contract revision is a single architecture-document RFC consolidating all three structural commitments. It opens as ordinary architecture-document RFC discipline post-Layer-B-resolution; not pre-Gate to Layer B closure.

### Implication for operational-specification follow-on

Parameter values (T_B, K_C, N), the "recent assertions" window form, and per-subtype divergence open as a separate operational-specification RFC. The RFC may itself surface empirical-pressure-phase reversal conditions for L-BC-OR per [`§0022`](../../charter/decision-log.md) discipline.

### Implication for Q4 fragility record discharge

Per [`q4-evidence.md` §Phase 5 reversal conditions](./q4-evidence.md), Q4's AND-composition between Layer A and Layer B was committee-chosen with five reversal conditions recorded. Layer B's resolution does NOT touch the outer AND-vs-OR composition; the Q4 reversal record remains operative. **However**, Layer B's L-BC-OR inner form (UNION coverage of Thesis halves) addresses the underlying concern that motivated several Q4 reversal conditions — particularly the "Layer A's necessary gate proves too restrictive" condition. Under L-BC-OR, the deep criterion fires on either failure mode independently, mitigating the case where Layer A's timer is the bottleneck on a hypothesis already exhibiting one Thesis half. **The Q4 fragility surface is reduced (but not eliminated) by L-BC-OR's adoption.**

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
