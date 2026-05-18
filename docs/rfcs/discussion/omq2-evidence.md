# OMQ #2 — Decay of Influence — discussion evidence

**Status:** in-discussion. Not authoritative. Final resolution will be recorded in decision-log and Ontology document revision.

This scratch supports the discussion phase of [OMQ #2 (Decay of influence)](../draft/ontology-revision-omq2-decay-of-influence.md), opened as §2.4 pre-Gate per [`decision-log §0019`](../../charter/decision-log.md). Five phases parallel to Q1-1 / Q2-1 / Q3-1 / Q4-1; checkpoint approval gates each phase.

The RFC's `Proposal` section enumerates three candidates (A permanent, B operational decay parameter, C decay via §2.5 lifecycle event supersession) plus Candidate D (runtime classification) explicitly rejected. This scratch evaluates A/B/C across seven dimensions, applies epistemic skills, synthesizes findings, and produces a recommendation.

## Phase 1 — Evidence per dimension per candidate

3 candidates × 7 dimensions = 21 cells. Source citations after each cell.

### Dimension 1 — Schema and validation surface

**A (Permanent).** Single edge type: `(source_assertion, influenced_hypothesis, [optional context fields])`. No decay parameter. No supersession reference. Validation: source assertion exists (Cat II or III); influenced hypothesis exists (Cat III); both resolvable to substrate records. Edge multiplicity: zero-or-more `influenced_by` edges per source assertion. Minimal field count.
- *Citation:* OMQ #2 RFC §Proposal Candidate A; [`provenance-model.md` L41-44](../../ontology/provenance-model.md) §The Provenance Graph (edges of `influenced_by` variety; bare edge type).

**B (Operational decay parameter).** Edge type with parameter fields: `(source_assertion, influenced_hypothesis, decay_function_ref, decay_parameters, [optional context fields])`. Validation: A's checks plus decay function reference resolves (likely to a Cat II operational construct namespace per §2.2 + Q1 determinism); parameters typed per the registered decay function. Edge multiplicity: same as A. Parameter field count varies by decay function form (scalar weight, half-life, threshold-and-curve, etc.); sub-decision deferred per OMQ #2 RFC §Open Questions.
- *Citation:* OMQ #2 RFC §Proposal Candidate B; [`provenance-model.md` L24](../../ontology/provenance-model.md) (typed-by-category edges post-Q3).

**C (Decay via lifecycle event supersession).** Edge type same as A: `(source_assertion, influenced_hypothesis, [optional context fields])`. Plus: supersession tracked via lifecycle event references — when the referenced hypothesis is demoted, the §2.5 BC5 demotion event (Cat I record) is consulted to determine current edge state. Validation: edge fields per A; supersession state derived from §2.5-codified lifecycle event chain at query time. No edge-local supersession field; the supersession is computed from §2.5 lifecycle event chain per [§2.5 BC5](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness).
- *Citation:* OMQ #2 RFC §Proposal Candidate C; [Charter §2.5 frozen v0.3 BC5](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness); [`lifecycle-semantics.md`](../../ontology/lifecycle-semantics.md).

### Dimension 2 — Provenance graph shape

**A.** Complete, ever-growing, uniform edge type. Every `influenced_by` edge committed to substrate persists structurally. Graph traversal is single-edge-type; no per-edge state variation. Traversal complexity is linear in edge count.
- *Citation:* OMQ #2 RFC §Proposal Candidate A "Schema implication"; [`provenance-model.md` §The Provenance Graph](../../ontology/provenance-model.md) (DAG framing).

**B.** Complete graph (all edges committed permanently per §2.1) with parameter-weighted edges. Graph shape identical to A at substrate level; weighted state is projection-layer logic. Traversal: edges retrieved from substrate, weights computed at query/projection time. Computation complexity: linear in edge count + decay function evaluation per edge.
- *Citation:* OMQ #2 RFC §Proposal Candidate B "Schema implication" + "Query pattern"; [`provenance-model.md` §The Provenance Graph](../../ontology/provenance-model.md) (projection rebuild semantics).

**C.** Complete graph plus lifecycle-event supersession overlay. Graph shape identical to A at edge level; supersession state derived from §2.5 lifecycle event chain. Traversal: edges retrieved from substrate, then for each edge the referenced hypothesis's lifecycle is consulted to determine current supersession state. Computation complexity: linear in edge count + linear in average lifecycle event chain length per referenced hypothesis.
- *Citation:* OMQ #2 RFC §Proposal Candidate C "Schema implication" + "Query pattern"; [Charter §2.5 frozen v0.3 Structural Requirement](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness).

### Dimension 3 — Query patterns: "what currently influences this assertion?"

**A.** Reduces to substrate read: enumerate `influenced_by` edges of the source assertion. Single-step, fully deterministic. Projection rebuild produces identical output every time given same substrate state.
- *Citation:* OMQ #2 RFC §Proposal Candidate A "Query pattern".

**B.** Substrate read + decay function evaluation per edge. Multi-step: enumerate edges, then for each evaluate the registered decay function against operational context (elapsed time, threshold, weighted value). Output is time-dependent — same substrate, different query outputs at different query times. Projection rebuild produces identical *edges* but decay-weighted output varies with operational time.
- *Citation:* OMQ #2 RFC §Proposal Candidate B "Query pattern".

**C.** Substrate read + lifecycle event traversal per edge. Multi-step: enumerate edges, then for each consult the referenced hypothesis's lifecycle event chain (per §2.5 BC5) to determine current supersession state. Demoted hypotheses' edges annotated as superseded. Projection rebuild consults full lifecycle event chain; rebuild is deterministic but multi-step.
- *Citation:* OMQ #2 RFC §Proposal Candidate C "Query pattern"; [Charter §2.5 frozen v0.3 Demotion subsection](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness).

### Dimension 4 — Layer B interaction (NOVEL DIMENSION)

Per [`§0011`](../../charter/decision-log.md) + [`§0017`](../../charter/decision-log.md) Methodological Observation 2: OMQ #2 resolution feeds Layer B's deep criterion specification. First Ontology RFC where Layer B feed is an explicit dimension.

**A.** Layer B reads `influenced_by` graph as time-uniform: all edges currently operative. Deep criterion (Candidate B-family per Q4: evidence-staleness; or Candidate C-family per Q4: influence-saturation) treats all edges equally. Per [Layer B RFC L60-61](../draft/ontology-revision-layer-b-deep-criterion.md): "if Layer B includes Candidate B [evidence-staleness], the binding prose must structurally subtract hypothesis-influenced assertions from the freshness denominator" — under A, ALL influences subtract (no temporal filter). For C-family (influence-saturation), saturation counts all permanent edges. Specification consequence: Layer B's deep criterion treats influence as a *current* commitment for ALL edges in substrate; specification is simpler (no temporal logic).
- *Citation:* Layer B RFC §Proposal "Two candidate families inherited from Q4 discussion" + L60-61.

**B.** Layer B reads `influenced_by` graph weighted by decay function output. Deep criterion incorporates decay-weighted influence: evidence-staleness denominator excludes hypotheses whose decay weight falls below threshold; influence-saturation count weighted by decay state. Specification consequence: Layer B's deep criterion must specify how decay weights compose with the staleness / saturation calculation. Decay function form (sub-decision under Candidate B) propagates into Layer B's parameter surface. Tight coupling between OMQ #2 resolution and Layer B specification.
- *Citation:* Layer B RFC §Proposal "if Layer B includes Candidate C [influence-saturation], the operational form depends on Q5's resolution of how `influence` propagates"; OMQ #2 RFC §Proposal Candidate B "Layer B interaction".

**C.** Layer B reads `influenced_by` graph filtered by §2.5 lifecycle supersession state. Deep criterion treats demoted hypotheses' edges as superseded — evidence-staleness denominator excludes superseded influences (effectively making demotion auto-clear the influence-staleness term); influence-saturation counts only non-superseded edges. Specification consequence: Layer B's deep criterion depends on §2.5 demotion semantics; Layer B becomes a downstream consumer of §2.5 lifecycle event chain. Tightest coupling — Layer B's specification incorporates §2.5 BC5 directly.
- *Citation:* Layer B RFC L31 "[§2.5] touched indirectly — Layer B's specification refines the binding text §2.5 redaction produces"; OMQ #2 RFC §Proposal Candidate C "Layer B interaction".

### Dimension 5 — OMQ #3 cascade-coupling probability (NOVEL DIMENSION)

OMQ #3 (Influence at projection vs substrate): *when* are `influenced_by` edges generated? Per [§0014](../../charter/decision-log.md) lazy methodology, OMQ #3 is deferred to §2.4 Step 1.1 empirical assessment; cascade trigger probability per candidate is OMQ #2-1's methodological contribution.

**A.** **Low coupling.** Projection rebuild generates same `influenced_by` edges deterministically from substrate event chain. OMQ #3's "when generated" question reduces to "when substrate events trigger influence edge commit" — independent of OMQ #2 resolution under A. OMQ #3 may remain forward-referenced per §0017 default; cascade trigger unlikely.
- *Citation:* OMQ #2 RFC §Open Questions "OMQ #3 status"; OMQ #3 statement per [`provenance-model.md` L56-57](../../ontology/provenance-model.md).

**B.** **Medium coupling.** Decay parameter on substrate-committed edge is unambiguous; decay-weighted *output* at projection introduces OMQ #3's question: is the weighted state a substrate-side commitment (edge carries decay parameter, applied at substrate event time) or a projection-time computation (edge carries decay parameter, applied at query time)? Sub-decision affects whether OMQ #3 cascades — if the decay parameter must be evaluated at substrate event time (against immutable §2.1 substrate), OMQ #3 is implicit; if at projection time only, OMQ #3 may need explicit resolution to bind the operational decay semantics. Cascade trigger probability moderate.
- *Citation:* OMQ #2 RFC §Open Questions "OMQ #3 status"; [`provenance-model.md` §Open Modeling Question 3](../../ontology/provenance-model.md).

**C.** **High coupling.** Supersession state is derived from §2.5 lifecycle event chain — entirely substrate-resident (per §2.5 BC5 lifecycle-event-as-Cat-I-record). Projection rebuild consults the lifecycle event chain to materialize current supersession; this IS the projection-vs-substrate distinction OMQ #3 asks about. OMQ #3 resolution becomes part of Candidate C's structural commitment: lifecycle event chain is substrate; supersession state is projection over substrate. Cascade trigger probability high — OMQ #3 likely opens as cascade per [§0014](../../charter/decision-log.md) methodology following §0015→§0016 pattern.
- *Citation:* OMQ #2 RFC §Open Questions "OMQ #3 status — high coupling under Candidate C"; [Charter §2.5 frozen v0.3 BC5](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness); [`§0015`](../../charter/decision-log.md) + [`§0016`](../../charter/decision-log.md) cascade pattern.

### Dimension 6 — §2.5 frozen v0.3 inheritance compatibility

**A.** **No inheritance.** `influenced_by` edges are independent of §2.5 lifecycle. Edge persists structurally regardless of hypothesis lifecycle state. No structural dependency.
- *Citation:* OMQ #2 RFC §Proposal Candidate A.

**B.** **No inheritance.** Decay parameter is independent of §2.5 lifecycle. Operational decay is governed by parameter-and-function logic, not by hypothesis lifecycle state. The decay function may *coincidentally* be parameterized by lifecycle events as inputs, but the structural dependency is on the decay function specification, not on §2.5 directly.
- *Citation:* OMQ #2 RFC §Proposal Candidate B.

**C.** **Structural inheritance — verified accommodative.** Supersession depends on §2.5 demotion semantics + lifecycle event commit guarantees per §2.5 frozen v0.3 Demotion subsection: "Demotion itself, once a candidate is confirmed, is recorded as an immutable lifecycle event referencing the prior promotion event." §2.5 codifies demotion as committed lifecycle event with structural reference to promotion. Candidate C reads this committed event and treats it as supersession annotation for `influenced_by` edges referencing the demoted hypothesis. **§2.5 frozen v0.3 supports this structurally without modification.** No retrofit required. The supersession reading is an additional consumer of the §2.5-committed event chain, not an augmentation of §2.5 binding text.
- *Citation:* [Charter §2.5 frozen v0.3 Structural Requirement L144](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness): "Demotion itself, once a candidate is confirmed, is recorded as an immutable lifecycle event referencing the prior promotion event." Plus [§2.5 BC5](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness): lifecycle events are Cat I records about Cat III entities.

### Dimension 7 — §2.2 (frozen — Epistemic Separation) compliance (LOAD-BEARING)

**A.** **Cleanest §2.2 compliance.** `influenced_by` edge is a structural reference from a Cat II/III assertion to a Cat III hypothesis. No category crossing introduced beyond the reference itself; the reference type is consistent with §2.2's Cat I/II/III separation. Edge has no decay/supersession state, so no Cat-placement question for derived state. Structurally clean, no conditional commitments.
- *Citation:* [Charter §2.2 frozen Definition L80](../../charter/constitutional-charter.md#22-epistemic-separation) — categories occupy distinct types in the system's typed structure; §2.2 codifies that each category's type and operations are not exposed through interfaces of another.

**B.** **Contested §2.2 compliance — sub-decision on Cat I vs Cat II placement.** The decay parameter on the substrate-committed edge raises a placement question:
- **Sub-decision B-substrate:** decay parameter is Cat I substrate annotation. The parameter value is committed at edge formation time (when the influence event is recorded); §2.1 immutability applies; the decay function evaluates that immutable parameter against operational time. Clean Cat I/II separation: Cat I edge + Cat I parameter; Cat II decay function evaluation at projection. **Clean.**
- **Sub-decision B-projection:** decay parameter is Cat II projection-time computation. The edge in Cat I substrate is bare; the decay parameter is recomputed at projection time from operational context. **Risk: collapses to runtime classification (Candidate D-adjacent)** — if the parameter is freely recomputable at projection, the structural commitment to "decay" lives in the projection logic, not the substrate. §2.2 separation is compromised because the influence's temporal behavior is a Cat II computation indistinguishable from runtime inference.

  Specification of Candidate B's decay parameter Cat-placement is a critical sub-decision; the discussion phase must surface this as the load-bearing question. Sub-decision B-substrate is the structurally clean reading; sub-decision B-projection is Candidate D-adjacent and should be rejected by the same rationale that rejects D.
- *Citation:* OMQ #2 RFC §Proposal Candidate B "§2.2 compliance shape"; [Charter §2.2 frozen Definition L80](../../charter/constitutional-charter.md#22-epistemic-separation); [`epistemic-separator` §4](../../../.claude/skills/epistemic/epistemic-separator/SKILL.md) Cat I/II separation.

**C.** **Structurally clean §2.2 compliance via §2.5 inheritance.** Supersession is encoded at the §2.5 lifecycle event level — the demotion event is a Cat I record per §2.5 BC5. The `influenced_by` edge remains Cat I substrate commitment (unchanged from A); the supersession reading is a *projection over §2.5 lifecycle event chain* (Cat II computation that traverses Cat I events). Cat I/II/III separation cleanly respected: Cat I edge, Cat I demotion event, Cat II projection-time supersession state. No category crossing introduced by the supersession mechanism itself.
- *Citation:* [Charter §2.2 frozen Definition L80](../../charter/constitutional-charter.md#22-epistemic-separation); [Charter §2.5 frozen v0.3 BC5](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) lifecycle-event-as-Cat-I-record; OMQ #2 RFC §Proposal Candidate C "§2.2 compliance shape".

---

## Phase 1 — Observations

### Strongest asymmetry (dimension 7 — §2.2 compliance)

Dimension 7 produces the strongest asymmetry between candidates, as predicted. Candidate B's contested §2.2 compliance sub-decision (B-substrate clean vs B-projection D-adjacent) is the load-bearing deliberation crux. Candidates A and C exhibit structurally clean §2.2 compliance; B exhibits compliance contingent on a sub-decision that may collapse it to Candidate D-adjacent shape.

### Genuinely contested vs apparent trade-offs

- **Genuinely contested:** Dimension 7 Candidate B (Cat I vs Cat II placement). This is not a trade-off resolvable by evidence alone; it requires a committee value choice about whether to admit operational decay parameters at the substrate edge layer.
- **Genuinely contested:** Dimension 4 (Layer B interaction) — all three candidates have distinct Layer B feeds; the choice depends on what Layer B is *supposed* to do, which is itself unspecified pending §2.6.
- **Apparent trade-off that resolves:** Dimension 5 (cascade-coupling) — high coupling under C is a methodological consequence (cascade will likely fire), not a strike against C. Per §0014 lazy methodology, cascade firing is the expected outcome of inheritance-dominant decisions; this is design-as-expected.
- **Apparent trade-off that resolves:** Dimension 2 (graph shape) — multi-step traversal under B/C vs single-step under A is operational complexity, not constitutional asymmetry. Substrate immutability is preserved under all three.
- **Asymmetry:** Dimension 1 (schemas surface) — A's minimalism is real but loses information vs B's parameter / C's supersession. Trade-off resolves by what information OMQ #2 wants the system to encode.

### Dimension 4 × Dimension 5 coupling (OMQ #2-specific observation)

Dimensions 4 (Layer B feed) and 5 (OMQ #3 cascade) are coupled per candidate:

- **Candidate A:** Layer B feed is simple (uniform graph); OMQ #3 cascade is low (independent of OMQ #2 outcome). Loose coupling between dimensions; both favor procedural simplicity at the cost of expressiveness.
- **Candidate B:** Layer B feed is parameter-coupled (decay propagates into Layer B's parameter surface); OMQ #3 cascade is medium (depends on B's sub-decision). Coupling is moderate; both dimensions point to the same Cat I/Cat II placement question (dimension 7 sub-decision).
- **Candidate C:** Layer B feed is lifecycle-coupled (Layer B reads §2.5 supersession); OMQ #3 cascade is high (§2.5 lifecycle event chain is substrate vs projection question itself). Tight coupling — both dimensions inherit from §2.5 BC5; Candidate C concentrates downstream decisions at the lifecycle event chain layer.

The coupling tells the committee: Candidate C concentrates *both* downstream specification and cascade activation at §2.5 inheritance; Candidate B distributes them across decay parameter + sub-decision; Candidate A defers expressiveness in exchange for minimum downstream coupling.

## Phase 2 — Scaffold check

Four findings on pre-existing scaffold language that may carry implicit lean or accommodate/reject candidates.

### F-SCAFFOLD-1 — provenance-model.md OMQ #2 framing

**Verbatim** ([`provenance-model.md` L56](../../ontology/provenance-model.md)):

> **2. Decay of influence.** Should inferential influence have a temporal decay, or does an assertion remain "influenced by" a hypothesis indefinitely? Decision pending.

The framing presents two options explicitly ("temporal decay" and "remain 'influenced by' … indefinitely"); "Decision pending" signals committee-resolved framing. Slight grammatical observation: "indefinitely" is the unmarked clause structure, which a reader could parse as the default state if decay is rejected — but this is editorial inference, not explicit lean. Neither A nor B nor C is named or favored.

**Verdict: Neutral framing.** No implicit lean.

### F-SCAFFOLD-2 — provenance-model.md §Inferential Provenance scaffold

**Verbatim** ([`provenance-model.md` L28-37](../../ontology/provenance-model.md)):

> Answers the question: **"Which prior assertions influenced the formation of this one?"**
>
> This is the form of provenance that conventional systems do not maintain. The Charter requires it because without it, the system cannot distinguish between belief grounded in independent evidence and belief inherited recursively from earlier conclusions.
>
> To be formalized:
> - The representation of influence references.
> - **The propagation rules: how influence transitively accumulates through chains of derivation.**
> - The relationship to evidential independence (Invariant 2.6 — pending): inferential provenance is the structural input to independence computation.

The bolded "to be formalized" bullet uses **"transitively accumulates"** — "transitively" addresses ontology.md Q5's transitive half; "accumulates" is implicitly additive/monotonic. An "accumulating" influence does not decay; it grows. Vocabulary tilts toward Candidate A's permanence framing.

**Verdict: Implicit lean toward Candidate A (permanence)** via "accumulates." Classified as **inherited-pressure** per §0017 Methodological Observation 2 + Q3-1 refinement — acknowledged but not dispositive. The final line ("Substantive §Inferential Provenance content remains pending §2.4 binding text") explicitly defers, partially mitigating the lean.

### F-SCAFFOLD-3 — §2.5 frozen v0.3 lifecycle subsection

**Verbatim** ([Charter §2.5 frozen v0.3 Structural Requirement L137-144](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness)) codifies demotion as "an immutable lifecycle event referencing the prior promotion event." [§2.5 BC5](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) codifies lifecycle events as Cat I records about Cat III entities. §2.5 does NOT mention `influenced_by` relationships or supersession-of-influence — it speaks to demotion-of-hypothesis only.

**Critical inference.** Candidate C's supersession reading is **external to §2.5** — it lives in §2.4 binding text's eventual codification. §2.4 binding text would codify: "when the influenced hypothesis is demoted per §2.5, the influence edge is annotated as superseded for projection purposes." §2.5's binding text does not need amendment; §2.5 supplies the committed demotion event; §2.4 supplies the supersession reading.

**Verdict: §2.5 frozen v0.3 mechanically supports Candidate C structurally — without modification.** Pattern is structurally analogous to §2.3 (frozen v0.4) inheriting §2.5 BC5 — sibling §2.x sections layer readings on the same committed substrate events without amending each other.

### F-SCAFFOLD-4 — Layer B follow-on RFC on-hold content (CRITICAL FINDING)

**Verbatim** ([Layer B RFC L13](../draft/ontology-revision-layer-b-deep-criterion.md)): the RFC names **Q5 (influence propagation)** as a dependency (alongside Q3 — formal definition of independence). [`ontology.md` §Open Questions Q5](../../ontology/ontology.md): "How does `influence` propagate through derived assertions? Transitive? Decaying? Both?" Layer B L61 mentions "decaying" as one of Q5's options.

**OMQ #2 (provenance-model.md) covers the "decaying?" half of Q5 (ontology.md). They are partially-overlapping questions in different documents.**

Consequence for OMQ #2 resolution:
- OMQ #2 resolution informs Layer B's "decaying?" axis.
- OMQ #2 resolution does NOT discharge Q5's "transitive?" half. Q5's transitive-question remains open after OMQ #2 closes.
- §2.4 binding text may need to address the transitive question separately.

Layer B's scaffold is **OMQ-#2-agnostic** at the decay-vs-permanence axis but **structurally presumes Q5's transitive half is separately tractable**. OMQ #2 resolution will not be Layer B's only Q5 input — Q5's transitive question may surface at §2.4 Step 1.1 per §0014 lazy methodology, paralleling how OMQ #3 may cascade.

**Verdict: ontology.md Q5 (transitive half) surfaces as a second cascade candidate** for §2.4 Step 1.1 (alongside OMQ #3).

### Phase 2 — Methodological summary

| Document | Scaffold state classification |
|---|---|
| provenance-model.md OMQ #2 framing | **Neutral** (F-SCAFFOLD-1) |
| provenance-model.md §Inferential Provenance scaffold | **Inherited-pressure** toward A (F-SCAFFOLD-2 — "accumulates") |
| §2.5 frozen v0.3 lifecycle subsection | **Accommodative** of C (F-SCAFFOLD-3 — supports without modification) |
| Layer B follow-on RFC | **OMQ-#2-agnostic** + surfaces ontology.md Q5 transitive cascade candidate (F-SCAFFOLD-4) |

**Methodological note on §2.5 inheritance pattern:** F-SCAFFOLD-3 establishes a structural pattern — frozen §2.x binding text codifies committed events; sibling §2.x binding texts layer additional readings on those committed events without amending each other. §2.3 BC5 inherits §2.5 BC5 for provenance chain continuation. Candidate C of OMQ #2 would be the second application — §2.4 binding text reads §2.5-committed demotion events as supersession signals. Pattern: **inter-section structural layering**, distinct from intra-section element codification.

## Phase 3 — Epistemic findings

Applied three skills (`falsifiability-check` §1, `epistemic-separator` §1+§4, `ambiguity-reducer` §1) to each candidate. 3 × 3 = 9 cells.

### Candidate A × falsifiability-check

- **1.1 Violation observability:** Pass. Falsifying state is an Assertion whose `influenced_by` edges, when read from substrate, do not include a hypothesis that the inference process declared as influencing. Detectable via substrate replay + comparison.
- **1.2 Observation:** Pass. Third party reads `influenced_by` edges from substrate without inference; pure structural check.
- **1.3 Operationalization:** Pass. "Permanent edge" reduces to "edge committed once to substrate, immutable per §2.1, no decay parameter, no supersession reference."
- **1.4 Non-circularity:** Pass. `influenced_by`, `assertion`, `hypothesis` glossary-canonical.

**Overall: Pass clean.**

### Candidate A × epistemic-separator

- **Naming:** Pass. `influenced_by`, `Assertion`, `Hypothesis` canonical.
- **Operation validity:** Pass. Commit (append-only) is valid for Cat I substrate edges per §2.1.
- **Typed crossing:** Pass. Cat II/III assertion → Cat III hypothesis; structurally typed reference per Q3 §0016.
- **Skill §4 anti-patterns:** No conflict. No mutation (#2); no write-back (#5); typed reference satisfies orphan-typed-reference shape (#6 per §2.3 v0.4 addition).

**Overall: Pass clean.**

### Candidate A × ambiguity-reducer

`state` absent; `record` canonically qualified ("Cat I substrate record"); `event` canonically qualified; others absent.

**Overall: Minimal watchlist surface — least ambiguity among A/B/C.**

### Candidate B × falsifiability-check

- **1.1 Violation observability:** Pass. Falsifying state is an `influenced_by` edge with no decay parameter populated, or an unresolvable decay function reference.
- **1.2 Observation:** Pass. Schemas-level check; mechanical.
- **1.3 Operationalization:** **Pass with caveat / partial fail.** "Decay parameter" reduces to a registered Cat II construct + edge-local parameter values per RFC §Proposal Candidate B. Specific form deferred to post-OMQ#2 specification work. Caveat: per Phase 1 Dim 7, the sub-decision (Cat I vs Cat II placement) is unresolved; if **B-projection** wins, the operationalization collapses to runtime classification (Candidate D-adjacent) and falsifiability degrades.
- **1.4 Non-circularity:** Pass. References resolve to canonical Cat II construct registration + edge parameter fields.

**Overall: Pass with caveat on 1.3 (conditional on Dim 7 sub-decision).**

### Candidate B × epistemic-separator (LOAD-BEARING CELL)

- **Naming:** Pass with caveat. `decay parameter`, `decay function` are new vocabulary candidates.
- **Operation validity:** Pass. Under B-substrate, decay parameter is committed at edge formation (Cat I substrate field). Under B-projection, decay state is computed at projection time (Cat II read-only computation).
- **Typed crossing:** Pass under B-substrate. Under B-projection, contested.

**Skill §4 critical check:**

Under **B-substrate** (decay parameter Cat I, evaluation Cat II at projection): the decay state of the edge is a *projection over Cat I substrate fields* — directly admissible per skill §4 construction #4 rewrite. **Clean.**

Under **B-projection** (decay parameter computed entirely at projection time without Cat I commitment): the decay state has no Cat I source and the "projection" is *runtime classification* — structurally equivalent to Candidate D, **forbidden** by the same rationale that rejects D. May also trigger skill §4 construction #5 if the runtime classification is committed back to substrate as derived state.

**Verdict.** Pass under B-substrate; **fails under B-projection** with §4 #4 / §4 #5 violations. **The Dim 7 sub-decision is load-bearing as Candidate B's admissibility test, not merely a sub-decision.** Without B-substrate commitment, Candidate B is structurally indistinguishable from rejected Candidate D.

### Candidate B × ambiguity-reducer

- `state` — heavy use ("decay state", "weighted state", "decay-state-dependent"). **Watchlist hit.**
- `context` — "operational context". **Watchlist hit.**
- `evidence` — implicit via Q4 framing ("evidence-staleness"); already operationalized at Q4.
- Other watchlist terms (skill §1 entries) — absent in Candidate B's claim text.

**Overall: Two watchlist hits requiring Path-decision operationalization at OMQ#2-2.** Under B-substrate, "decay state" operationalizes parallel to §2.5 Step 1.3 Path 2; under B-projection, "decay state" violates the §2.5-established Path 2 shape (computed without corresponding event-history projection). Vocabulary surface reinforces the epistemic-separator finding.

### Candidate C × falsifiability-check

- **1.1 Violation observability:** Pass. Falsifying state is an `influenced_by` edge whose target hypothesis has a committed §2.5 demotion event but where projection-time query does not annotate as superseded.
- **1.2 Observation:** Pass. Mechanical traversal of substrate edge + §2.5 BC5 lifecycle event chain.
- **1.3 Operationalization:** Pass. "Supersession" reduces to "demotion event committed per §2.5 referenced by the hypothesis target's lifecycle event chain"; "projection annotates as superseded" reduces to projection-layer computation per §2.5 BC5 reading.
- **1.4 Non-circularity:** Pass. `supersession`, `demotion`, `lifecycle event`, `influenced_by` glossary-canonical.

**Overall: Pass clean.**

### Candidate C × epistemic-separator

- **Naming:** Pass. `supersession`, `demotion`, `lifecycle event`, `Cat I record` canonical.
- **Operation validity:** Pass. §2.5 demotion event commit is valid Cat III operation per §2.5 frozen v0.3.
- **Typed crossing:** Pass. Cat I demotion event ← Cat III hypothesis; Cat II projection computes supersession state by reading Cat I event chain.

**Skill §4 anti-patterns:** No conflict with #2 (no mutation) or #5 (no write-back). **Construction #4 alignment:** the supersession state is *exactly* the skill §4 #4 rewrite pattern — "the current values of each Category II construct keyed by [referenced hypothesis identity]." **Candidate C is the canonical example of skill §4's prescribed shape for derived state.**

**Overall: Pass clean. Aligns with skill §4 #4 rewrite pattern directly.**

### Candidate C × ambiguity-reducer

- `state` — "supersession state". **Watchlist hit.** Operationalizes naturally via §2.5 Step 1.3 Path 2 precedent (state-as-projection-over-event-history).
- `record`, `event` — canonically qualified per §2.5 BC5.

**Overall: One watchlist hit operationalizing cleanly via §2.5 Step 1.3 Path 2 precedent. Lowest residual ambiguity among B/C.**

### Phase 3 — Methodological observations

**Most consequential finding:** Candidate B × epistemic-separator confirms the Dim 7 sub-decision is **Candidate B's admissibility gate**, not a sub-decision. B-substrate admissible (aligns with skill §4 #4 rewrite); B-projection forbidden (collapses to Candidate D shape). B-projection's rejection is consequence-driven, not procedural.

**§2.5 inheritance compatibility:** Candidate C × epistemic-separator confirms §2.5 lifecycle event commit semantics extend cleanly to superseding an `influenced_by` edge reading. Supersession is a *new Cat II projection* over Cat I substrate events — does not modify §2.5; consumes §2.5-committed events.

**Aggregate verdict per candidate:**
- **A:** Pass clean across all three skills. Minimal vocabulary surface; structurally simplest.
- **B:** Pass under B-substrate sub-decision only; fails under B-projection. Two watchlist hits requiring Path-decision at OMQ#2-2.
- **C:** Pass clean. One watchlist hit operationalizing via §2.5 Step 1.3 Path 2 precedent. Aligns with skill §4 #4 rewrite as canonical example.

## Phase 4 — Comparison synthesis

Findings numbered in order of consequence.

### Finding 1 — Candidate B's Dim 7 sub-decision is an admissibility gate

**(Genuine asymmetry; promoted from sub-decision to gate.)** Only **B-substrate** is admissible; B-projection collapses to Candidate D shape (rejected). Effectively B narrows to B-substrate.

### Finding 2 — Candidate C's epistemic-separator alignment is the canonical example of skill §4 #4 rewrite

**(Asymmetry — strong evidence for C.)** C's supersession-via-§2.5-lifecycle-event mechanism aligns exactly with skill §4 construction #4 rewrite. Epistemically rare: C inherits both §2.5 structural mechanism and skill §4 #4 prescriptive pattern simultaneously.

### Finding 3 — §2.5 inheritance for C is structurally clean and accommodative

**(Asymmetry — verified mechanical support.)** F-SCAFFOLD-3 verified. Sibling §2.x layering pattern §2.3 already established.

### Finding 4 — F-SCAFFOLD-2's "accumulates" lean is not dispositive

**(Apparent trade-off that resolves.)** Inherited-pressure acknowledged but not binding given Phase 3 epistemic evidence for C. "Accumulates" naturally reads as substrate-side accumulation (which all three candidates respect); projection-state decay is the OMQ #2 question.

### Finding 5 — Layer B feed shape per candidate (Dim 4 + F-SCAFFOLD-4)

**(Genuine asymmetry.)** A simplest (uniform-graph feed); B-substrate parameter-coupled (decay function form propagates); C §2.5-coupled (consumes existing mechanism rather than introducing new one).

### Finding 6 — OMQ #3 cascade probability per candidate (Dim 5)

**(Apparent trade-off — resolves via §0014 design intent.)** A low; B-substrate medium; C high. Cascade firing is design-as-expected, not a strike.

### Finding 7 — ontology.md Q5 transitive half remains open regardless of OMQ #2 outcome

**(Methodological finding from F-SCAFFOLD-4.)** Q5's "transitive?" half remains open after OMQ #2 closes. §2.4 Step 1.1 may surface Q5-transitive as second cascade candidate alongside OMQ #3. Not a candidate differentiator.

### Finding 8 — Candidate B's residual ambiguity surface requires OMQ#2-2 Path decisions

**(Genuine asymmetry — vocabulary cost.)** B carries two watchlist hits (`state`, `context`) requiring Path-decision operationalization. A and C carry less vocabulary cost.

### Finding 9 — Constitutional minimalism check (CLAUDE.md §7)

**(Asymmetry — favoring C and A over B-substrate.)** A introduces no new mechanism (bare edge); C introduces no new mechanism (consumes §2.5 events); B-substrate introduces decay parameter + Cat II decay function machinery. A and C minimal; B-substrate not.

### Finding 10 — §1 Thesis temporal-evolution support (load-bearing for A vs C)

**(Genuine trade-off — between A's structural permanence and C's structural temporal expressiveness.)** [§1 Thesis](../../charter/constitutional-charter.md#1-thesis) "evolves, and is acted upon over time" supports temporal evolution at substrate level. A captures evolution at projection only; C captures it via §2.5-committed events. Load-bearing argument for C over A.

### Per-candidate verdict summary

| Candidate | §2.2 compliance | Layer B feed | OMQ #3 cascade | §2.5 inheritance |
|---|---|---|---|---|
| **A** | Structurally clean | Simple uniform-graph | Low probability | None (independent) |
| **B-substrate** | Structurally clean (B-projection forbidden) | Parameter-coupled | Medium probability | None (independent) |
| **C** | Structurally clean via §2.5 inheritance | §2.5-coupled (lifecycle event chain) | High probability (design-as-expected) | Verified accommodative — sibling consumption |

### Convergence shape

**Candidate-level with sub-decision narrowing.** Parallel to Q3-1's per-Category granularity refinement: the discussion produces a candidate selection AND a sub-decision (B's sub-decision narrows B to B-substrate).

### Summary statement

**Evidence points toward Candidate C, with A as defensible-but-less-expressive alternative.** C is epistemically strongest (Finding 2); captures §1 Thesis temporal-evolution at substrate (Finding 10); constitutionally minimal (Finding 9). F-SCAFFOLD-2's scaffold lean toward A (Finding 4) not dispositive. A defensible as simplest commitment; loses substrate-level temporal expressiveness (Finding 10 trade-off). B-substrate admissible but strictly dominated by C on every dimension except Layer B feed independence-from-§2.5.

## Phase 5 — Recommendation

**Adopt Candidate C — Decay via §2.5 lifecycle event supersession.** Evidence base: Finding 2 (C's epistemic-separator alignment with skill §4 construction #4 rewrite as canonical example), Finding 3 (§2.5 frozen v0.3 mechanically supports C without modification, verified at F-SCAFFOLD-3), Finding 10 (C captures §1 Thesis "evolves, and is acted upon over time" at substrate level via §2.5-committed demotion events; A captures it only at projection), and Finding 9 (constitutional minimalism — C introduces no new structural mechanism, consuming §2.5-committed events via Cat II projection). C's high OMQ #3 cascade probability (Finding 6) is design-as-expected per §0014, not a strike. B-substrate admissible but strictly dominated by C on epistemic strength, constitutional minimalism, and vocabulary cost (Findings 8 + 9); B-projection rejected by the same rationale as D (Finding 1). A defensible as simplest commitment but loses substrate-level temporal expressiveness (Finding 10's trade-off); F-SCAFFOLD-2's "accumulates" framing not dispositive given Phase 3 epistemic alignment of C with skill §4 #4 (Finding 4).

### §2.5 inheritance verification

Per F-SCAFFOLD-3: [§2.5 frozen v0.3 Structural Requirement L144](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) codifies demotion as "an immutable lifecycle event referencing the prior promotion event." [§2.5 BC5](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) codifies lifecycle events as Cat I records about Cat III entities. C reads §2.5-committed demotion events as supersession signals — external to §2.5; lives in §2.4 binding text. **§2.5 frozen v0.3 supports C structurally without modification.** No retrofit required.

### What would reverse this recommendation

- **Layer B's substantive content (post-§2.6) reveals incompatibility with §2.5-mediated supersession.** If §2.6's evidential-independence specification requires operational decay weighting rather than lifecycle-event filtering, B-substrate becomes necessary. Surfaces at §2.6 redaction.
- **A demotion-decoupled decay case surfaces at §2.4 redaction.** If §2.4 binding text needs influence to decay without demotion (e.g., evidence accumulating against the hypothesis while it remains operationally promoted), C provides no mechanism. Case not identified today; would need evidence at §2.4 Step 1.1.
- **Constitutional minimalism reading inverts.** If committee at OMQ#2-2 prefers structural-permanence-at-substrate + projection-time filtering (A) on grounds that simplest substrate structure is most minimal. Values question — minimum-substrate-machinery (A) vs minimum-new-derived-state-mechanism (C). Phase 4 preferred C on §1 Thesis temporal-evolution axis.
- **F-SCAFFOLD-2's "accumulates" scaffold lean is given dispositive weight.** Under Q3-1 classification, inherited-pressure not dispositive; committee may invert.

### §2.4 binding text shape under Candidate C

- **Definition.** Every Assertion formed under influence declares the influence structurally via `influenced_by` edge to the hypothesis. Edge committed immutably per §2.1; influence-at-formation-time is permanent at substrate.
- **Structural Requirement.** Current operational state is Cat II projection over (a) committed substrate state and (b) referenced hypothesis's §2.5 lifecycle event chain. When the referenced hypothesis has a committed demotion event, edge state is annotated as superseded for projection-time queries (independence per §2.6 (pending); Layer B per §0011 follow-on RFC).
- **Forbidden Anti-Patterns.** Mutating `influenced_by` post-commit (§2.1 violation); computing supersession state without consulting §2.5 lifecycle event chain; silent promotion of hypothesis-derived assertions to enrichment context without declaring the influence edge.
- **Boundary Conditions.** §2.4 governs inferential semantics of `influenced_by` edges and supersession-reading over §2.5 events; does not codify §2.5 demotion semantics (inherited) or §2.6 independence computation (forward-referenced). §2.4 inherits §2.3 BC1 from the §2.4 side.

Binding text inheritance-dominant; substantive new content limited to §2.4-original anti-patterns and §2.4 side of §2.3 BC1's mutual scope statement.

### Cascade implications under Candidate C

**OMQ #3 cascade trigger probability: HIGH.** Per Finding 6 + Phase 1 Dim 5: §2.5 lifecycle event chain is substrate; supersession state derived from it is projection. This IS OMQ #3's question. Under C, OMQ #3 resolution becomes part of §2.4 binding text's structural commitment.

**Anticipated cascade enactment at OMQ#2-2 commit per §0014 methodology + §0015 precedent:** OMQ#2-2 resolution commit likely opens OMQ #3 RFC at `discussion`, paralleling §0015's Q1 resolution opening Q3 RFC. Estimated additional prompts: 2 (OMQ #3 discussion + resolution).

**ontology.md Q5 transitive half remains open** (Finding 7) — recognized cascade candidate for §2.4 Step 1.1 alongside OMQ #3.

### Layer B feed shape under Candidate C

Per Finding 5: Layer B reads §2.5-committed lifecycle event chain directly. Layer B's eventual deep criterion treats `influenced_by` edges referencing demoted hypotheses as superseded — evidence-staleness denominator (Q4 B-family) excludes superseded influences; influence-saturation count (Q4 C-family) counts only non-superseded edges. Layer B's specification consumes §2.5 BC5 directly. Coupling tighter than B-substrate but consumes existing mechanism rather than introducing new one. OMQ #2 resolution partially-discharges Layer B's ontology.md Q5 dependency for the "decaying?" axis; Q5's "transitive?" axis remains open.
