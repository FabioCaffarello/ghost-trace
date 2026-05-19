# OMQ #3 — Influence at projection vs substrate — discussion evidence

**Status:** in-discussion. Not authoritative. Final resolution will be recorded in decision-log and Ontology document revision.

This scratch supports the discussion phase of [OMQ #3 (Influence at projection vs substrate)](../draft/ontology-revision-omq3-influence-projection-vs-substrate.md) — cascade-triggered RFC opened at OMQ #2-2 per [`decision-log §0020`](../../charter/decision-log.md). Sixth Ontology RFC discussion (Q2-1, Q4-1, Q1-1, Q3-1, OMQ #2-1 precede); second cascade-enactment after Q3-1 (Q1→Q3 cascade per §0015→§0016).

The RFC's `Proposal` section enumerates two admissible candidates (α substrate-time generation; β projection-time generation) plus Candidate γ (runtime classification) explicitly rejected. This scratch evaluates α/β across seven dimensions, applies epistemic skills, synthesizes findings, and produces a recommendation.

## Phase 1 — Evidence per dimension per candidate

2 candidates × 7 dimensions = 14 cells. Source citations after each cell.

### Dimension 1 — Schema and substrate commit semantics

**α (Substrate-time generation).** `influenced_by` edges are committed at formation event time. When an Assertion is formed under influence of one or more hypotheses, the influence edges are part of the formation event's structural payload (or as immediately-following events referencing the formation event with the same formation-time stamp). Substrate state at time T contains all `influenced_by` edges for Assertions formed at-or-before T. Edge multiplicity per Assertion: zero-or-more, one per influencing hypothesis. Schema-level: `(source_assertion, influenced_hypothesis, [optional context fields])` per OMQ #2-C-codified structure.
- *Citation:* OMQ #3 RFC §Proposal Candidate α "Schemas implication"; [`provenance-model.md` §Inferential Provenance post-OMQ-#2-C](../../ontology/provenance-model.md) bullet 1 ("Every Assertion formed under influence carries one or more `influenced_by` edges...").

**β (Projection-time generation).** `influenced_by` edges are computed at projection time from substrate formation events. Substrate captures formation events with hypothesis-context references in the event payload (which hypotheses provided enrichment context at formation time); projection derives explicit `influenced_by` edges at query/rebuild time by traversing formation events and reading their hypothesis-context references. Substrate state at time T contains no `influenced_by` edge records; only formation events with hypothesis-context payloads. Current projection at T derives edges from substrate state at T.
- *Citation:* OMQ #3 RFC §Proposal Candidate β "Schemas implication"; [Charter §2.2 frozen](../../charter/constitutional-charter.md#22-epistemic-separation) Cat II derivation rules.

### Dimension 2 — Projection rebuild determinism

**α.** Yes — edges are read from substrate directly. Projection rebuild reads `influenced_by` edges from primary event chain; identical substrate produces identical edges every rebuild. Determinism is structural (substrate read) rather than algorithmic. No versioning required beyond schemas.
- *Citation:* OMQ #3 RFC §Proposal Candidate α "Query pattern"; [Charter §2.1 frozen](../../charter/constitutional-charter.md#21-observational-integrity) substrate immutability.

**β.** Conditionally — projection rebuild produces identical edges *if* the derivation algorithm is deterministic and versioned per Q1 ([`§0015`](../../charter/decision-log.md)) operational-definition determinism commitment. The "Q1 determinism" handle: Cat II constructs (which derived edges are, under β) must be deterministic per versioned operational definition; non-deterministic derivation would constitute Cat III misclassification rejected at validation. Practical risk: derivation algorithm evolution post-implementation creates rebuild divergence; mitigated by Q1's versioning commitment (re-derivation under new definition produces new construct; existing construct unchanged). The risk is procedural, not structural — provided the derivation algorithm is registered per §2.2 Cat II rules.
- *Citation:* OMQ #3 RFC §Proposal Candidate β "Query pattern"; [`§0015` Q1 resolution](../../charter/decision-log.md) (operational determinism + identity-tier consistency default); [Charter §2.2 frozen](../../charter/constitutional-charter.md#22-epistemic-separation) Cat II rules.

### Dimension 3 — OMQ #2-C supersession compatibility

**α.** Yes — clean. Supersession (per OMQ #2-C) reads Cat I `influenced_by` edges (per α) + §2.5 Cat I demotion events; both are Cat I records. Supersession layer (Cat II projection per OMQ #2-C) composes the two Cat I sources at projection time: for each edge, consult the referenced hypothesis's §2.5 lifecycle chain; annotate as superseded if demotion event present. Single projection layer reads from two Cat I sources.
- *Citation:* OMQ #2-C codification per [`provenance-model.md` §Inferential Provenance](../../ontology/provenance-model.md) bullet 3 ("The current operational state of an `influenced_by` edge is a Category II projection over (a) the substrate-committed edge per §2.1 and (b) the referenced hypothesis's §2.5 lifecycle event chain"); [Charter §2.5 BC5](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness).

**β.** Yes — structurally consistent but two-step projection layer. Supersession (per OMQ #2-C) reads §2.5 Cat I demotion events; β-derived edges are Cat II projection over formation events. Projection layer composes: (a) derive `influenced_by` edges from formation events per β's algorithm; (b) for each derived edge, consult referenced hypothesis's §2.5 lifecycle chain; annotate as superseded if demotion event present. Two derivations stacked at Cat II layer (edge derivation + supersession annotation); both deterministic per Q1; clean compositionally provided Q1 determinism applies to the composite.
- *Citation:* Same as α, with composition adjustment for β; OMQ #2-C codification per [`provenance-model.md` §Inferential Provenance](../../ontology/provenance-model.md); [`§0015` Q1](../../charter/decision-log.md).

### Dimension 4 — §2.1 (Cat I substrate immutability) compliance (LOAD-BEARING)

**α.** Clean. `influenced_by` edges are Cat I substrate records committed at formation event time per §2.1 immutability. Post-commit modification forbidden by §2.1 Structural Requirement (write-once semantics). The edge's permanence is structurally guaranteed; subsequent supersession (per OMQ #2-C) is projection-layer reading, not edge mutation. §2.1's scope extends naturally to `influenced_by` edges under α.
- *Citation:* [Charter §2.1 Structural Requirement L55](../../charter/constitutional-charter.md#21-observational-integrity); OMQ #3 RFC §Proposal Candidate α "§2.2 compliance shape".

**β.** Inapplicable-by-construction (Cat II projection layer); reconstructibility check via §2.3 BC1 chain extends to Cat I formation events, not to the derived edges themselves. Under β, `influenced_by` edges are not Cat I records; they are Cat II projection-derived. §2.1 immutability scope does not extend to Cat II projections by construction — projections are rebuildable per §2.1 BC. The question this raises: does β's lack of Cat I edge commitment weaken §2.3 frozen v0.4 reconstructibility guarantee? **Verification per §2.3 BC1:** §2.3's chain extends back to *Cat I primaries* including the Assertion's typed `subject_ref_*` fields (which ARE Cat I substrate records on the Assertion entity per [`entity-model.md`](../../ontology/entity-model.md) post-Q3); under β, the chain reaches the Assertion's subject_ref fields and the formation events (both Cat I) but the explicit `influenced_by` edges are Cat II derived. The edges' Cat-placement is §2.4 territory per §2.3 BC1's mutual scope statement; §2.3 is satisfied by the chain reaching Cat I formation events. **β is §2.1-compliant by way of being §2.1-inapplicable** to derived projections; §2.3 reconstructibility is satisfied via formation events.
- *Citation:* [Charter §2.1 Boundary Conditions L72](../../charter/constitutional-charter.md#21-observational-integrity) — derived projections, materialized views, and caches are not bound by §2.1; they are rebuildable from the primary event chain. [Charter §2.3 frozen v0.4 BC1 + Structural Requirement](../../charter/constitutional-charter.md#23-provenance-integrity); OMQ #3 RFC §Proposal Candidate β "§2.2 compliance shape".

### Dimension 5 — §2.3 BC1 inheritance shape (§2.4-side)

§2.3 frozen v0.4 BC1 codifies mutual scope statement: §2.3 governs observational-provenance transit through Cat II/III references; §2.4 governs inferential semantics of the same references.

**α.** §2.4-side codifies that `influenced_by` edges are Cat I substrate facts about Assertion formation; §2.4 governs their inferential semantics (which assertions/hypotheses influenced this one). Chain back to Cat I primaries: trivial — edges ARE Cat I; chain reaches edges directly + edges reference Cat I (Assertion subject_ref + Hypothesis) records.
- *Citation:* [Charter §2.3 frozen v0.4 BC1](../../charter/constitutional-charter.md#23-provenance-integrity); [`entity-model.md` Assertion entity](../../ontology/entity-model.md) post-Q3.

**β.** §2.4-side codifies that `influenced_by` edges are Cat II projections derived from formation events; §2.4 governs their inferential semantics. Chain back to Cat I primaries: via the formation events the edges reference. The projection-time computation is deterministic per Q1; supersession reading (per OMQ #2-C) composes at the same Cat II layer. §2.3 BC1 accommodates β provided §2.4-side codifies the derivation algorithm's Q1-deterministic-under-versioned-operational-definition commitment.
- *Citation:* [Charter §2.3 frozen v0.4 BC1](../../charter/constitutional-charter.md#23-provenance-integrity); [`§0015` Q1](../../charter/decision-log.md); OMQ #3 RFC §Proposal Candidate β "Layer B interaction".

### Dimension 6 — Layer B interaction

Per [`§0011`](../../charter/decision-log.md) + [`§0017`](../../charter/decision-log.md) + [`§0020`](../../charter/decision-log.md) Methodological Observation 2, OMQ #3 resolution feeds Layer B's eventual specification alongside OMQ #2-C's supersession mechanism.

**α.** Layer B reads Cat I `influenced_by` edges directly + Cat I §2.5 demotion events; supersession filter (per OMQ #2-C) applied at projection. Specification: two-source consumption (edges + supersession events), both Cat I; Layer B's deep criterion treats current-influence as "Cat I edge AND no superseding §2.5 demotion event". Clean two-source structure.
- *Citation:* Layer B RFC §Proposal "Two candidate families inherited from Q4 discussion"; OMQ #2-C codification per [`provenance-model.md` §Inferential Provenance](../../ontology/provenance-model.md) bullet 4.

**β.** Layer B reads Cat II derived edges + Cat I §2.5 demotion events; derived edges themselves consume Cat I formation events. Specification: three-source consumption (formation events + derivation algorithm version + supersession events). Layer B's deep criterion treats current-influence as "edge derivable from current substrate formation events per versioned operational definition AND no superseding §2.5 demotion event". Adds derivation-algorithm-versioning dependency to Layer B specification.
- *Citation:* OMQ #3 RFC §Proposal Candidate β "Layer B interaction"; Layer B RFC L61 ("Q5 dependency").

### Dimension 7 — §2.2 (Cat I/II/III separation) compliance

**α.** Clean. `influenced_by` edges are Cat I substrate records; supersession is Cat II projection over Cat I records (per OMQ #2-C). Cat I/II separation respected: Cat I edges + Cat I lifecycle events; Cat II reads both at projection. No Cat-mixing.
- *Citation:* [Charter §2.2 frozen Definition](../../charter/constitutional-charter.md#22-epistemic-separation); OMQ #2-C codification.

**β.** Clean *conditional on Q1 determinism*. `influenced_by` edges are Cat II projections derived from Cat I formation events; supersession is Cat II projection over Cat I lifecycle events. Both edge derivation and supersession are Cat II projections over Cat I substrate; categories remain separated. **Critical check: does β collapse to Candidate γ?** γ is runtime classification (inference process determining influence at runtime). β is *deterministic derivation per Q1 versioned operational definition* — algorithmic, registered, reproducible. β remains structurally distinct from γ **provided the derivation algorithm is registered as a Cat II operational construct per §2.2 + Q1 rules**. If β's derivation is treated as runtime inference (not registered as versioned Cat II construct), β collapses to γ; this is β's admissibility gate analogous to OMQ #2 Candidate B's Cat I/Cat II sub-decision.
- *Citation:* [Charter §2.2 frozen](../../charter/constitutional-charter.md#22-epistemic-separation); [`§0015` Q1](../../charter/decision-log.md); OMQ #3 RFC §Alternatives Considered γ rejection.

---

## Phase 1 — Observations

### Strongest asymmetry (Dimension 4 — §2.1 compliance)

Dimension 4 produces the structural asymmetry between α and β, as predicted. α's `influenced_by` edges are Cat I substrate records — §2.1 immutability applies directly. β's edges are Cat II derived projections — §2.1 inapplicable by construction. The asymmetry is not a defect in β: per §2.1 Boundary Conditions, derived projections, materialized views, and caches are not bound by §2.1; they are rebuildable from the primary event chain. β is §2.1-compliant by being §2.1-out-of-scope. The structural question Dimension 4 raises: does §2.3 frozen v0.4's reconstructibility guarantee weaken under β? **Answer: no** — §2.3's chain extends to Cat I primaries including formation events; under β the chain reaches formation events directly, and the derived edges themselves are §2.4 territory per §2.3 BC1's mutual scope statement.

### Genuinely contested vs apparent trade-offs

| Type | Dimension(s) |
|---|---|
| **Genuinely contested** | Dim 7 Candidate β admissibility gate (Q1 determinism explicit commitment required for β to remain structurally distinct from γ). Analogous to OMQ #2 Candidate B's Cat I/Cat II sub-decision but resolves more cleanly via Q1 inheritance. |
| **Genuinely contested** | Dim 6 (Layer B feed) — three-source vs two-source consumption is real specification cost difference; both feeds work, but β adds derivation-algorithm-versioning dependency. |
| **Apparent → resolves** | Dim 4 (§2.1 compliance) — apparent asymmetry resolves: §2.1 inapplicability under β is admissible per §2.1 BC, not a defect. §2.3 reconstructibility preserved via formation events. |
| **Apparent → resolves** | Dim 2 (projection rebuild determinism) — β's conditional determinism resolves via Q1 versioned operational definition (already-codified mechanism). |
| **Asymmetry** | Dim 1 (substrate commit semantics) — α commits edges at substrate; β commits only formation events. Real structural difference in substrate footprint and commit complexity. |
| **Asymmetry** | Dim 3 (OMQ #2-C compatibility) — both candidates accommodate, but β stacks two derivations (edge + supersession) at Cat II layer; α reads two Cat I sources at one Cat II layer. |

### OMQ #3-specific observation: β admissibility via Q1 inheritance

Under β, the derivation algorithm's Q1-deterministic-under-versioned-operational-definition commitment is *sufficient* to keep β structurally distinct from rejected γ — provided the §2.4 binding text (or β's resolution) explicitly references Q1 as the determinism anchor. This is **cleaner than OMQ #2 Candidate B's Cat I/Cat II sub-decision**: B's sub-decision was a values choice without a pre-existing mechanism to anchor admissibility (B-substrate was admissible because *new substrate-side commitment* could be required; B-projection collapsed because *no such commitment* existed). β inherits Q1 determinism as pre-existing mechanism; admissibility is anchored by existing §0015 commitment, not contingent on new sub-decision.

Methodological observation: β's admissibility-via-Q1-inheritance pattern mirrors OMQ #2 Candidate C's admissibility-via-§2.5-inheritance pattern. Both candidates anchor their structural admissibility by consuming existing frozen mechanisms. Pattern: **cascade-enacted RFCs may naturally favor candidates whose admissibility is anchored by prior frozen mechanisms** (because cascade-trigger contexts have richer prior commitments to consume).

## Phase 2 — Scaffold check

Five findings on pre-existing scaffold language that may carry implicit lean or accommodate/reject candidates. OMQ #3 is cascade-triggered by OMQ #2-C ([`§0020`](../../charter/decision-log.md)); F-SCAFFOLD-1 is the critical finding (cascade-inherited pressure from triggering resolution).

### F-SCAFFOLD-1 — provenance-model.md §Inferential Provenance post-OMQ-#2-C (CRITICAL CASCADE-INHERITED FINDING)

**Verbatim** ([`provenance-model.md` L32-39](../../ontology/provenance-model.md) post-§0020):

> Per [`decision-log §0020`](../charter/decision-log.md) (OMQ #2 resolution — decay via §2.5 lifecycle event supersession; Candidate C):
>
> - **Influence reference structure.** Every Assertion formed under influence carries one or more `influenced_by` edges referencing Category III hypotheses (or Category II constructs whose own provenance reaches Category III). **The edge is committed structurally at formation time and immutable per [Charter §2.1 frozen](../charter/constitutional-charter.md#21-observational-integrity).**
> - ...
> - **Decay via §2.5 lifecycle event supersession.** The current operational state of an `influenced_by` edge is a Category II projection over (a) **the substrate-committed edge per §2.1** and (b) the referenced hypothesis's §2.5 lifecycle event chain. ...

**Reading.** Bullet 1's "The edge is committed structurally at formation time and immutable per Charter §2.1 frozen" is **α's structural claim verbatim**. Bullet 3's "the substrate-committed edge per §2.1" repeats the α framing. Under β, edges are not substrate-committed; they are Cat II projection-derived. **The OMQ #2-C codification, authored at OMQ #2-2 Phase 2 before OMQ #3 was formally opened, implicitly committed to α.**

**Verdict.** **Strong α-lean — inherited from triggering resolution (cascade-inherited pressure).** This is *not* mere scaffold pressure; it is α-codified in just-merged binding-shape prose. The committee that resolved OMQ #2-C may not have intended to pre-resolve OMQ #3; but the prose did.

**Consequence for recommendation:**
- **If OMQ #3 resolves to α**: scaffold pressure aligns; no §Inferential Provenance revision needed.
- **If OMQ #3 resolves to β**: §Inferential Provenance bullet 1 and bullet 3 need revision at OMQ #3-2 to remove α-specific "committed structurally at formation time" / "substrate-committed edge" wording. The revision is a committee-extension at OMQ #3-2 analogous to OMQ #2-2's three extensions.

**Methodological observation candidate (for OMQ #3-2 §0021):** **Cascade-enactment scaffold inherits pressure from triggering resolution.** When OMQ #2-C resolution authored binding-shape prose at §Inferential Provenance, the prose used α-specific vocabulary because α was the unmarked reading (consistent with discussion phase F-SCAFFOLD-2 "accumulates" lean). The OMQ #3 RFC was then cascade-opened to formalize the question OMQ #2-C had already implicitly resolved. Pattern: **future cascade-enactment RFCs should anticipate that the triggering resolution's codification may pre-resolve the cascade question implicitly**, and the discussion phase must surface this as F-SCAFFOLD-1 finding rather than silently accept.

### F-SCAFFOLD-2 — provenance-model.md OMQ #3 verbatim

**Verbatim** ([`provenance-model.md` L58](../../ontology/provenance-model.md)):

> 3. **Influence at projection vs. substrate.** When a projection is rebuilt from the substrate, does its computation introduce influence edges? Or are influence edges only generated when influence is *operationally consequential*? **Cascade-triggered** by OMQ #2 resolution per [`decision-log §0020`](../charter/decision-log.md); RFC opened at `discussion` status at [`ontology-revision-omq3-influence-projection-vs-substrate`](../rfcs/draft/ontology-revision-omq3-influence-projection-vs-substrate.md).

**Reading.** The framing presents two options:
- Option A: "does its computation introduce influence edges" — projection rebuild *introduces* edges (β phrasing — projection generates edges at rebuild).
- Option B: "are influence edges only generated when influence is *operationally consequential*" — sparse-β phrasing (generate edges only when needed; lazy projection).

Neither option clearly states α (substrate-time commit). Under α, edges are NOT introduced at projection rebuild (they pre-exist in substrate); under α, edges are generated regardless of operational consequence (every influence is committed at formation time).

**Verdict.** **Slight β-lean** via "introduce" and "generated" vocabulary (both suggest projection-time edge creation, β-flavored). The OMQ #3 statement was authored before formal cascade enactment (at provenance-model.md scaffolding pre-§0020) and was not refreshed at the cascade-trigger annotation. The β-lean is **not dispositive** — committee may interpret "introduce" as "present in projection from substrate" (compatible with α).

### F-SCAFFOLD-3 — §2.3 frozen v0.4 BC1's §2.3-side wording

**Verbatim** ([Charter §2.3 frozen v0.4 BC1](../../charter/constitutional-charter.md#23-provenance-integrity)):

> **§2.3 governs observational provenance; not inferential influence.** The chain shape — `subject_ref_observation` edges terminating at Cat I primaries — is §2.3's scope. The inferential semantics of `subject_ref_construct` (Cat II) and `subject_ref_hypothesis` (Cat III) edges as influence relations are [§2.4](#24-inferential-influence-disclosure) territory; §2.3 reads the same edges as transit, not as influence.

**Reading.** BC1 speaks of "the inferential semantics of `subject_ref_construct` / `subject_ref_hypothesis` edges as influence relations" — these are the typed-reference fields on the Assertion entity (per Q3 §0016 + entity-model.md post-Q3, Cat I substrate fields on the Assertion record). BC1 treats them as Cat I substrate facts; whether the projection of these into `influenced_by` edges is α (substrate-time edge commit) or β (projection-derived edges) is **§2.4 territory and OMQ-#3-agnostic per §2.3 BC1**.

**Verdict.** **§2.3 BC1 is structurally agnostic on §2.4-side α vs β.** The mutual scope statement is candidate-neutral. §2.3 reconstructibility is preserved under both candidates via the Assertion's typed `subject_ref_*` fields (Cat I) — `influenced_by` edges are §2.4 territory regardless of whether they're α-substrate or β-derived. F-SCAFFOLD-3 imposes no constraint on OMQ #3 candidate selection.

### F-SCAFFOLD-4 — §2.4 stub current state

**Verbatim** ([Charter §2.4 stub L144-150](../../charter/constitutional-charter.md#24-inferential-influence-disclosure)):

> **Status:** Pending committee redaction.
>
> **Working definition (non-binding):** When a hypothesis is promoted to use as enrichment context, every assertion subsequently formed under that influence carries a structural declaration of the influence. The system preserves, by construction, the distinction between belief grounded in independent evidence and belief inherited recursively from earlier conclusions.

**Reading.** "Every assertion ... carries a structural declaration of the influence." The verb "carries" suggests substrate-side commitment (α-flavored — the assertion itself carries the declaration). Under β, the assertion carries the formation event with hypothesis-context references in payload; the explicit `influenced_by` edges are derived at projection. The stub's "carries" wording is **α-flavored vocabulary** but read against the stub's "Pending committee redaction" status (non-binding), the lean is methodologically less weighty than F-SCAFFOLD-1.

**Verdict.** **Slight α-lean via "carries" vocabulary**, but **non-binding stub status mitigates**. The §2.4 binding text will be authored at §2.4 redaction Step 1.5; this stub does not constrain OMQ #3 candidate selection. Under β, §2.4 binding text would phrase as "every assertion formation event records the hypothesis-context references; the influence relationships are derived deterministically per Q1 from these references."

### F-SCAFFOLD-5 — Layer B follow-on RFC on-hold content

**Verbatim** ([Layer B RFC L60-61](../draft/ontology-revision-layer-b-deep-criterion.md)):

> - **Q4 Phase 3 Finding 6:** if Layer B includes Candidate B, the binding prose must structurally subtract hypothesis-influenced assertions from the freshness denominator. The structural test for "formed under this hypothesis's influence" is what §2.4 must supply.
> - **Q5 dependency:** if Layer B includes Candidate C, the operational form depends on Q5's resolution of how `influence` propagates (transitive, decaying, both, or other).

**Reading.** Layer B speaks of "the `influenced_by` graph" generically — under α, the graph is substrate-resident; under β, the graph is projection-derived per Q1 determinism. Layer B's deep criterion operates on the graph either way. The L60 reference to "the structural test for 'formed under this hypothesis's influence'" — under α, the test reads Cat I edges; under β, the test reads Cat II derived edges + formation events. Both candidates admissible from Layer B's perspective.

**Verdict.** **Layer B is OMQ-#3-agnostic** at the candidate-selection level. Specification cost differs (per Phase 1 Dim 6: α two-source, β three-source consumption) but Layer B's structural shape accommodates both. **F-SCAFFOLD-5 imposes no constraint on OMQ #3 candidate selection beyond specification-cost differential.**

### Phase 2 — Methodological summary

Per [`§0017`](../../charter/decision-log.md) Methodological Observation 2 (forward-reference contract extension; scaffold-state classification refined at Q3-1 + OMQ #2-1):

| Document | Scaffold state classification |
|---|---|
| provenance-model.md §Inferential Provenance post-OMQ-#2-C | **Cascade-inherited strong α-codification** (F-SCAFFOLD-1 — critical finding) |
| provenance-model.md OMQ #3 verbatim | **Slight β-lean** via "introduce"/"generated" vocabulary (F-SCAFFOLD-2 — non-dispositive) |
| Charter §2.3 frozen v0.4 BC1 | **Structurally agnostic** (F-SCAFFOLD-3) |
| Charter §2.4 stub | **Slight α-lean** via "carries" vocabulary; non-binding stub status (F-SCAFFOLD-4) |
| Layer B follow-on RFC | **OMQ-#3-agnostic** at candidate-selection (F-SCAFFOLD-5); specification-cost differs |

### Methodological note — cascade-inherited pressure as novel scaffold-state category

**OMQ #3-1 surfaces a methodologically novel scaffold-state category: cascade-inherited pressure.** F-SCAFFOLD-1 documents that OMQ #2-C's just-merged binding-shape prose at §Inferential Provenance contains α-specific vocabulary ("committed structurally at formation time", "substrate-committed edge per §2.1"). This is not pre-existing scaffold; it is **just-merged binding-shape prose authored at the cascade-triggering resolution that implicitly pre-resolves the cascade question**.

Distinguishing features vs prior scaffold-state classifications:
- **Neutral / inherited-pressure** (Q3-1, OMQ #2-1 F-SCAFFOLD-2) — pre-existing scaffold predating the question's formal opening.
- **Cascade-inherited pressure (NEW per OMQ #3-1 F-SCAFFOLD-1)** — just-merged binding-shape prose from the triggering resolution. Higher constitutional weight than pre-existing scaffold because it was *committee-approved at the cascade-trigger merge*; weighted higher than inherited-pressure.

**Pattern for future cascade-enacted RFCs:** Phase 2 scaffold check should explicitly survey the triggering resolution's just-merged prose for candidate-specific commitments. If found, surface as F-SCAFFOLD-1 (cascade-inherited pressure) and disposition at recommendation phase: either accept the inherited resolution (recommend the implicitly-pre-resolved candidate) or contest with committee-extension at resolution phase to revise the triggering codification.

This observation is OMQ #3-1's methodological contribution — analogous to OMQ #2-1's introduction of cascade-trigger-probability dimension as Phase 1 methodological contribution. Both contributions inform future cascade RFCs at the methodology layer.

## Phase 3 — Epistemic findings

Applied three skills (`falsifiability-check` §1, `epistemic-separator` §1+§4, `ambiguity-reducer` §1) to each admissible candidate. 2 × 3 = 6 cells.

### Candidate α × falsifiability-check

- **1.1 Violation observability:** Pass. Falsifying state is an Assertion that was influenced (per inference-process declaration at formation time) but whose substrate-replay does not surface the corresponding `influenced_by` edge. Mechanical schemas-level check.
- **1.2 Observation:** Pass. Third party reads substrate event chain; verifies edge presence for declared-influenced assertions; pure structural check.
- **1.3 Operationalization:** Pass. "Cat I `influenced_by` edge committed at formation event time" reduces to schemas-level field commit per §2.1 substrate-write rules + Q3 §0016 typed reference structure.
- **1.4 Non-circularity:** Pass. `influenced_by`, `Assertion`, `Hypothesis`, `formation event` glossary-canonical.

**Overall: Pass clean.**

### Candidate α × epistemic-separator

- **Naming:** Pass. `influenced_by`, Cat I, Cat II, Cat III canonical.
- **Operation validity:** Pass. Commit (append-only) is the valid operation for Cat I substrate records per §2.1.
- **Typed crossing:** Pass. Edge from Cat II/III Assertion to Cat III Hypothesis; typed reference per Q3 §0016.
- **Skill §4 anti-patterns:** No conflicts. #2 (mutation on new evidence) N/A — edges immutable per §2.1. #4 (state about each actor) N/A — edges are not state; they are immutable structural references. #5 (writes back) N/A — edge committed at formation, not written back. #6 (orphan typed-reference per §2.3 v0.4) addressed by Q3-B oneOf at Assertion level (edges originate from Assertion subject_ref structure).

**Overall: Pass clean.**

### Candidate α × ambiguity-reducer

Scan against 13-term watchlist:
- `state` — absent in α's structural claim.
- `event` — "formation event", "lifecycle event" — canonical-qualified.
- `record` — "Cat I substrate record" — canonical-qualified.
- Other watchlist terms (per skill §1: decision, evidence, context, behavior, identity, and additional entries) — absent in α's claim text or canonical-qualified.

**Overall: Minimal watchlist surface — comparable to OMQ #2 Candidate A clean profile.**

### Candidate β × falsifiability-check (Q1-CONDITIONAL)

- **1.1 Violation observability:** Pass. Falsifying state is a derivation-algorithm execution that produces different `influenced_by` edges given identical substrate event chain (Q1 determinism violation). Or: edges derived at projection that do not have a Cat I formation event reference resolvable in substrate (orphan-derived-edge).
- **1.2 Observation:** Pass. Third party runs registered Cat II derivation algorithm against substrate snapshot; compares output across runs. Mechanical given Q1 versioning.
- **1.3 Operationalization:** **Pass via Q1 inheritance.** "Deterministic derivation per versioned operational definition" reduces to §0015 Q1 commitment for Cat II constructs. The derivation algorithm must be registered as Cat II operational construct per §2.2 + Q1; the registration is the operationalization handle. Caveat: the §2.4 binding text or OMQ #3-2 resolution must reference Q1 explicitly to make admissibility binding.
- **1.4 Non-circularity:** Pass. References resolve to canonical Cat II construct registration + Q1 §0015 + formation event structure.

**Overall: Pass via Q1 inheritance — conditional on explicit Q1 reference at resolution.**

### Candidate β × epistemic-separator (LOAD-BEARING CELL)

- **Naming:** Pass. `influenced_by`, derivation algorithm, Cat II canonical.
- **Operation validity:** Pass. Parametric re-derivation under new definition is valid for Cat II per skill §1 ("parametric re-derivation under a new definition produces a new construct; does not mutate the existing one"). β's derivation algorithm execution at projection time IS this operation.
- **Typed crossing:** Pass. Cat II projection from Cat I formation events; structurally typed.

**Skill §4 critical check — does β collapse to γ?**

Under **β with explicit Q1 commitment**: the derivation algorithm is registered as a Cat II operational construct per §0015 Q1 rules — versioned, deterministic, registered. The derived `influenced_by` edges aligning with skill §4 construction #4 rewrite pattern: "the current values of each Category II construct keyed by [Assertion identity]." **β is the canonical example of skill §4 #4 rewrite at the Cat II derivation layer**, parallel to OMQ #2-C's alignment at the Cat II supersession layer. Clean.

Under **β without explicit Q1 commitment**: the derivation algorithm becomes runtime classification — inference process determining influence at runtime without versioned operational definition. Structurally equivalent to **rejected Candidate γ**; collapses to runtime inference. Same failure mode as OMQ #2 Candidate B-projection (which collapsed to Candidate D shape).

**Verdict.** Pass under explicit-Q1-commitment; **fails under non-explicit-Q1** (collapses to γ). **The Q1 explicit commitment is β's admissibility gate**, analogous to OMQ #2 Candidate B-substrate's Cat I placement gate — but **structurally cleaner** because the gate is anchored by *pre-existing* §0015 commitment, not contingent on new sub-decision.

### Candidate β × ambiguity-reducer

Scan against 13-term watchlist:
- `state` — "current projection state", "supersession state". **Watchlist hit** (former: β-introduced; latter: OMQ #2-C-inherited).
- `event` — "formation event", "lifecycle event" — canonical-qualified.
- `record` — "Cat I substrate record" — canonical-qualified.
- `context` — "formation context", "hypothesis-context references" — β-specific terminology for the substrate event payload that derivation reads. **Watchlist hit.**
- Other watchlist terms (skill §1 entries — decision, evidence, behavior, plus the additional watchlisted terms) — absent in β's claim text.
- `derivation` — central to β but not on watchlist; reduces to Cat II construct derivation per §0015.

**Overall: Two watchlist hits (`state`, `context`).** Both operationalize via inheritance:
- `state` per §2.5 Step 1.3 Path 2 precedent (state-as-projection-over-history); under β, supersession state is projection over §2.5 lifecycle chain (OMQ #2-C inheritance); derivation state is Cat II computation over formation events (Q1 inheritance).
- `context` per OMQ #2-C "Cat II projection over formation events" framing; `formation context` is the substrate event payload field, canonical-qualified once §2.4 binding text formalizes.

Operationalization paths exist via prior precedents; **lower residual ambiguity cost than OMQ #2 Candidate B** (which lacked clear operationalization paths under B-projection). Higher cost than α (zero watchlist hits) and comparable to OMQ #2 Candidate C (one watchlist hit).

### Phase 3 — Methodological observations

**Most consequential epistemic finding:** Candidate β × epistemic-separator confirms admissibility-via-Q1-inheritance is structurally sufficient to keep β distinct from γ. β aligns with skill §4 construction #4 rewrite pattern at the Cat II derivation layer — analogous to OMQ #2-C's alignment at the Cat II supersession layer. **β is structurally clean conditional on explicit Q1 commitment.** The Q1 commitment is the admissibility gate; without it, β collapses to γ.

**β admissibility gate structurally cleaner than OMQ #2 Candidate B's sub-decision.** OMQ #2 Candidate B-substrate required *new* substrate commitment (Cat I decay parameter field); OMQ #3 Candidate β inherits *pre-existing* Q1 commitment from §0015. The structural anchor is stronger in β: §0015 is binding governance (Q1 resolution); explicit Q1 reference in §2.4 binding text or OMQ #3-2 resolution makes admissibility binding without new committee extension.

**Aggregate verdict per candidate:**
- **α:** Pass clean across all three skills. Minimal vocabulary surface; structurally simplest at substrate level. Comparable to OMQ #2 Candidate A profile.
- **β:** Pass clean conditional on explicit Q1 commitment. Two watchlist hits (`state`, `context`) operationalize via inheritance (§2.5 Step 1.3 Path 2 for `state`; OMQ #2-C for `context`). Admissibility gate (vs γ) anchored by pre-existing §0015 Q1 commitment — structurally cleaner anchor than OMQ #2 Candidate B's new-sub-decision gate.

## Phase 4 — Comparison synthesis

Synthesizing Phase 1 (14 cells) + Phase 2 (5 scaffold findings) + Phase 3 (6 epistemic cells). Findings numbered in order of consequence.

### Finding 1 — F-SCAFFOLD-1 cascade-inherited pressure is the dispositively novel constitutional consideration of OMQ #3

**(Genuine asymmetry — cascade-inherited-pressure as new constitutional category.)** OMQ #2-C's just-merged binding-shape prose at [`provenance-model.md` §Inferential Provenance](../../ontology/provenance-model.md) bullets 1 and 3 contains α-specific vocabulary verbatim: "The edge is committed structurally at formation time and immutable per Charter §2.1 frozen" and "the substrate-committed edge per §2.1." The committee that resolved OMQ #2-C at OMQ #2-2 may not have intended to pre-resolve OMQ #3; the prose did.

Under cascade-inherited pressure reading: recommending α aligns with just-merged committee-approved prose; recommending β requires a committee-extension at OMQ #3-2 to revise §Inferential Provenance bullets 1 and 3. The reversal cost is structurally novel — first cascade-enacted RFC where the triggering resolution's codification implicitly pre-resolves the cascade question.

**Methodological observation candidate (for OMQ #3-2 §0021):** Cascade-inherited pressure is a new constitutional category distinct from pre-existing scaffold pressure. Higher constitutional weight (committee-approved at cascade-trigger merge); not dispositive (discussion phase can produce evidence overturning) but requires committee-extension to reverse.

### Finding 2 — §1 Thesis "evolves, and is acted upon over time" supports α substrate-side capture

**(Genuine trade-off — α has tighter Thesis alignment; β has equivalent operational expressiveness via Q1-deterministic derivation.)** [§1 Thesis](../../charter/constitutional-charter.md#1-thesis) frames the system's purpose as preserving the continued capacity to distinguish what was observed from what was inferred, as that knowledge accumulates, evolves, and is acted upon over time.

- **α** captures every influence relationship structurally at formation; recursive belief inflation defense is maximal at substrate (every recursive influence permanently structurally visible).
- **β** captures influences derivable from formation events via Q1-deterministic derivation; recursive belief inflation defense is operationally-equivalent (same projection state) but substrate-side capture is indirect.

α's substrate-side commitment captures the recursive-inflation surface at substrate level; β captures it at projection layer. Both candidates answer the Thesis question — α via direct substrate read, β via deterministic derivation. **α has tighter Thesis alignment** at substrate-directness axis.

Parallel to OMQ #2 Finding 10 (§1 Thesis "evolves, and is acted upon over time" supports C over A). **Note the inversion:** OMQ #2-C inherited §2.5 to gain substrate-side temporal expressiveness over A; OMQ #3-β inherits Q1 to *avoid* substrate-side edge commitment, sacrificing Thesis-directness. The two inheritance directions go opposite ways structurally.

### Finding 3 — β admissibility-via-Q1-inheritance is structurally cleaner than OMQ #2 Candidate B's gate

**(Asymmetry — β admissibility anchored by pre-existing §0015 vs OMQ #2 B-substrate's new-sub-decision.)** Phase 3 × epistemic-separator confirmed: β remains structurally distinct from γ provided the derivation algorithm is registered as a Cat II operational construct per §0015 Q1 rules. Q1 commitment is pre-existing (frozen governance per §0015); explicit Q1 reference at OMQ #3-2 resolution or §2.4 binding text makes admissibility binding without new committee extension.

Structural parallel to OMQ #2-C. Pattern: **cascade-enacted RFCs naturally favor candidates whose admissibility anchors in pre-existing frozen mechanisms**. β fits the pattern; the question is whether the *direction* of inheritance use (substrate-side commitment avoidance) is structurally preferable to substrate-side commitment (α).

### Finding 4 — Layer B specification cost differs

**(Genuine trade-off — α two-source vs β three-source consumption.)**
- **α:** Layer B reads Cat I edges directly + Cat I §2.5 demotion events. Two-source consumption; specification simpler.
- **β:** Layer B reads Cat II derived edges + formation events + derivation algorithm version + Cat I §2.5 demotion events. Three-source consumption; derivation-algorithm-versioning dependency added.

Per OMQ #2-1 Finding 5 (Layer B feed as RFC dimension), α has lower Layer B specification cost. The cost differential is real but does not invalidate β; specification cost is operational complexity, not constitutional asymmetry.

### Finding 5 — Constitutional minimalism (§7) — mixed verdict

**(Mixed asymmetry — minimum-substrate-machinery vs minimum-new-derived-state-mechanism.)** Per [CLAUDE.md §7](../../../.claude/CLAUDE.md):
- **α** introduces new Cat I substrate machinery (edge-commit-at-formation-time field; schemas surface for `influenced_by` edge at substrate layer).
- **β** introduces no new Cat I substrate machinery (only formation events); requires Cat II derivation algorithm registration per existing §0015 Q1 rules — uses existing mechanism.

By **new-substrate-machinery count**, β is more constitutionally minimal. By **new-derived-state-mechanism count**, α is more minimal (no new derivation algorithm). The two axes pull opposite directions. OMQ #2-2 Phase 4 surfaced the analogous tension; resolution chose C on §1 Thesis temporal-evolution axis (Finding 10). **§7 minimalism does not dispositively favor either candidate here.**

### Finding 6 — Migration asymmetry favors β reversibility

**(Asymmetry — α stronger lock-in; β more reversible.)**
- **α→β:** requires removing committed Cat I edges from substrate. Per §2.1 immutability, substrate mutation is forbidden; only supersession via new records. α→β is structurally near-impossible without violating §2.1.
- **β→α:** requires adding edge-commit semantics for new influences post-transition. Existing β-only edges (projection-derived pre-transition) remain reconstructible from formation events. Workable.

α represents stronger constitutional lock-in. **β is the more reversible choice.**

### Finding 7 — Substrate footprint vs commit complexity (Dim 1)

**(Genuine trade-off — neither dominates.)** α: substrate grows with every influence relationship. β: minimal substrate footprint; derivation algorithm specification added at Cat II projection layer. Operational characteristic, not constitutional asymmetry.

### Per-candidate verdict summary

| Candidate | §2.1 compliance | §2.2 compliance | §2.3 BC1 fit | Layer B feed | Migration |
|---|---|---|---|---|---|
| **α (substrate-time)** | Clean — edges Cat I; §2.1 applies | Clean | Trivial (edges Cat I) | Two-source (simpler) | Stronger lock-in per §2.1 |
| **β (projection-time)** | Inapplicable-by-construction; §2.3 chain via formation events preserves reconstructibility | Clean conditional on Q1 explicit commitment (anchors β vs γ) | Via formation events; Q1-deterministic derivation | Three-source (additive cost) | More reversible (β→α workable) |

### Convergence shape

**Candidate-level with cascade-inherited-pressure as decisive evidence axis.** Distinct from prior Ontology RFC shapes — both α and β are admissible structurally; the disposition turns on whether cascade-inherited pressure (F-SCAFFOLD-1) is treated as dispositive or as input. F-SCAFFOLD-1 is unique to OMQ #3 among Ontology RFCs to date.

### Pattern observation — inheritance direction inversion vs OMQ #2

OMQ #2-C and OMQ #3-β both inherit pre-existing frozen mechanisms to anchor admissibility. **Direction inversion:**

- **OMQ #2-C** inherits §2.5 to **gain substrate-side temporal expressiveness** over the simpler candidate A. Inheritance adds capability.
- **OMQ #3-β** inherits Q1 to **avoid substrate-side edge commitment** that the simpler candidate α makes. Inheritance subtracts capability (relative to α).

Pattern candidate (for §0021): **inheritance candidates dominate when they add substrate-side capability; inheritance candidates may be dominated when they subtract substrate-side capability**. §1 Thesis "accumulates, evolves" supports the former; minimum-substrate-machinery (§7 minimalism) supports the latter — but Thesis support is stronger constitutionally than minimalism support.

### Summary statement: evidence points toward α (substrate-time generation), with β admissible-but-dominated

Three converging grounds for α:

1. **F-SCAFFOLD-1 cascade-inherited pressure** (Finding 1): OMQ #2-C codification is α-specific; reversing requires committee-extension at OMQ #3-2.
2. **§1 Thesis substrate-side directness** (Finding 2): α captures recursive-inflation surface at substrate; β captures via derivation indirection.
3. **Layer B specification cost** (Finding 4): α two-source simpler than β three-source.

Counter-considerations for β:

- **β admissibility-via-Q1-inheritance** (Finding 3): structurally clean anchor; admissible.
- **§7 constitutional minimalism axis** (Finding 5): mixed verdict.
- **Migration reversibility** (Finding 6): β→α workable; α→β near-impossible per §2.1.

**Evidence points toward α, with β registered as admissible-but-dominated alternative** — parallel structurally to OMQ #2 B-substrate's status. The "rejected dominated" vs "rejected incoherent" distinction established at OMQ #2-2 applies here for β.

## Phase 5 — Recommendation

**Adopt Candidate α — Substrate-time generation.** Evidence base: Finding 1 (F-SCAFFOLD-1 cascade-inherited pressure aligns — OMQ #2-C's just-merged §Inferential Provenance prose at bullets 1 and 3 contains α-specific vocabulary verbatim; reversing requires committee-extension at OMQ #3-2), Finding 2 (§1 Thesis "evolves, and is acted upon over time" supports α's substrate-side capture of every influence relationship; β captures via Q1-deterministic derivation indirection), and Finding 4 (Layer B specification cost favors α two-source consumption over β three-source). β is structurally admissible via Q1 inheritance (Finding 3) but dominated on the three grounds above; β registers as **admissible-but-dominated alternative** per OMQ #2-2 B-substrate precedent.

The cascade-inherited pressure (F-SCAFFOLD-1) is OMQ #3-1's distinctive constitutional consideration: just-merged binding-shape prose at §Inferential Provenance encodes α structurally ("The edge is committed structurally at formation time and immutable per Charter §2.1 frozen"; "the substrate-committed edge per §2.1"). Treating this as input rather than dispositive — but converging with §1 Thesis directness (Finding 2) and Layer B simplicity (Finding 4) — the three grounds together overdetermine α.

### Required committee extension under α recommendation

**β admissibility-but-dominated acknowledgment** (parallel to OMQ #2-2 Resolution 1 acknowledging B-substrate). Decision-log §0021 records: β is admissible structurally (Q1-inheritance anchor; Phase 3 epistemic-separator clean conditional on explicit Q1 commitment) but dominated by α on F-SCAFFOLD-1 + §1 Thesis + Layer B specification cost. Pattern preserves the "rejected dominated" vs "rejected incoherent" distinction (γ rejected as incoherent per OMQ #3 RFC Alternatives Considered; β rejected as dominated per §0021).

**No §Inferential Provenance revision needed** under α recommendation. F-SCAFFOLD-1's α-codification at bullets 1 + 3 of [`provenance-model.md`](../../ontology/provenance-model.md) is consistent with α's resolution; OMQ #3-2 commit need only formalize what is already implicit.

### What would reverse this recommendation

- **F-SCAFFOLD-1 cascade-inherited pressure is contested as illegitimate.** If committee at OMQ #3-2 reads the OMQ #2-C codification's α-specific vocabulary as an accidental over-commitment rather than a committee-approved structural commitment, F-SCAFFOLD-1's weight collapses. The reversal would require revising §Inferential Provenance bullets 1 + 3 at OMQ #3-2 to remove α-specific phrasing and committing to β.
- **Constitutional minimalism (Finding 5) is given dispositive weight.** β is more constitutionally minimal at substrate-machinery axis (no new Cat I field; inherits Q1 derivation pattern). Committee may invert on §7 minimalism grounds.
- **Migration reversibility (Finding 6) is given dispositive weight.** α is harder-to-leave per §2.1; β is more reversible. Committee may invert on optionality grounds.
- **Substrate footprint cost (Finding 7) is given dispositive weight.** α grows substrate with every influence relationship; β minimizes substrate. Committee may invert on operational-cost grounds.
- **A previously-uncited derivation-vs-commit case surfaces during §2.4 redaction.** If §2.4 Step 1.1 surfaces a structural case requiring derivation-at-projection that α cannot accommodate, β becomes structurally necessary. Case not identified today.

### §2.4 binding text shape under α

§2.4 binding text would codify:

- **Definition.** Every Assertion formed under the influence of a hypothesis declares the influence structurally via an `influenced_by` edge committed at formation event time and immutable per [§2.1](../../charter/constitutional-charter.md#21-observational-integrity). The edge references the influencing hypothesis (Cat III) or operational construct (Cat II via its provenance reach).
- **Structural Requirement.** The `influenced_by` edge is a Cat I substrate record committed at the same time as the Assertion's formation event; the edge carries no decay parameter (per OMQ #2-C, supersession is a Cat II projection over §2.5 lifecycle event chain). The current operational state of the edge is per OMQ #2-C codification at [`provenance-model.md` §Inferential Provenance](../../ontology/provenance-model.md).
- **Forbidden Anti-Patterns.** Influence-edge omission at formation (Assertion formed under influence without committing the edge); mutation of `influenced_by` edge post-commit (§2.1 violation applied to influence relationships); silent promotion of hypothesis-derived assertions to enrichment context without declaring the influence (the stub's enumerated anti-pattern).
- **Boundary Conditions.** §2.4 governs the inferential semantics of `influenced_by` edges and supersession-reading over §2.5 events; does not codify §2.5 demotion semantics (inherited per OMQ #2-C). §2.4 inherits §2.3 BC1 from the §2.4 side: §2.3 reads `subject_ref_construct` / `subject_ref_hypothesis` edges as observational-provenance transit; §2.4 reads `influenced_by` edges as inferential influence with supersession overlay. Q5-transitive half remains forward-referenced (per [`ontology.md` §Open Questions Q5](../../ontology/ontology.md)) — §2.4 Step 1.1 empirical assessment per §0014 lazy methodology.

The binding text is **inheritance-dominant** (parallel to §2.3 frozen v0.4 shape): substantive new content limited to the influence-edge-commit-at-formation-time mechanism (§2.4-original) and the §2.4 side of §2.3 BC1's mutual scope statement.

### Cascade implications

**§0020 cascade discharges.** OMQ #3-2 resolution discharges the cascade enactment initiated at OMQ #2-2. No further cascade triggered by OMQ #3 itself — α's structural commitment is self-contained at substrate.

**Carry-forwards to §2.4 Step 1.1 (per §0014 lazy methodology + §0017 forward-reference contract extension):**

- **Q2 (Identity tiers, [`entity-model.md` Open Modeling Question 1](../../ontology/entity-model.md))** remains forward-referenced per §0017 Resolution 4 marker form; §2.4 Step 1.1 empirical assessment of blocking status.
- **ontology.md Q5 "transitive?" half** remains open (per OMQ #2-1 Finding 7); §2.4 Step 1.1 empirical assessment per §0014. May surface as cascade candidate if §2.4 binding text depends substantively on transitive-influence semantics.
- **Layer B activation reconciliation** per §2.5 Layer B forward-reference contract; §2.4 binding text encodes the reconciliation at Step 1.5.
- **§2.3 BC1 inheritance for §2.4 side** — structural; §2.4 binding text codifies the §2.4 side of mutual scope statement at Step 1.5.
- **F-SCAFFOLD-1 methodological observation candidate** — cascade-inherited pressure as new scaffold-state category. Recorded at OMQ #3-2 §0021 as methodological observation; future cascade-enacted RFCs apply the pattern.

### Layer B feed shape under α (additive to OMQ #2-C feed)

Per Phase 1 Dim 6 + Finding 4: Layer B reads Cat I `influenced_by` edges directly from substrate + Cat I §2.5 demotion events (per OMQ #2-C supersession). **Two-source consumption confirmed.** Layer B's eventual deep criterion specification:

- **Evidence-staleness denominator (Q4 B-family per [Layer B RFC L60](../draft/ontology-revision-layer-b-deep-criterion.md))**: structurally subtracts hypothesis-influenced assertions from freshness denominator. Under α, the "structural test for 'formed under this hypothesis's influence'" reads `influenced_by` edges directly from substrate; OMQ #2-C supersession filter applied at projection.
- **Influence-saturation count (Q4 C-family)**: counts non-superseded `influenced_by` edges. Under α, count reads Cat I edges directly; supersession filter applied at projection per OMQ #2-C.

OMQ #2-C feed contributed the supersession mechanism (§2.5-lifecycle-event-driven); OMQ #3-α contributes the edge-source clarity (Cat I substrate read, not Cat II derivation). The Layer B specification is the simpler of the two candidates (α) and consumes both Cat I sources directly. **OMQ #2-C + OMQ #3-α composition is the cleanest possible Layer B feed under the cascade-anchored Ontology resolutions.**

ontology.md Q5 "transitive?" half remains open and may surface at Layer B's eventual specification phase post-§2.4 + §2.6.
