# Q4 — Promotion → Demotion Criterion — discussion evidence

**Status:** in-discussion. Not authoritative. Final resolution will be recorded in decision-log and Ontology document revision.

This scratch supports the discussion phase of [`ontology-revision-q4-promotion-demotion-criterion.md`](../draft/ontology-revision-q4-promotion-demotion-criterion.md). The phases below mirror the discussion-phase procedure established by Q2: evidence per dimension per candidate (Phase 1, expanded to six dimensions and three candidates plus combinations), scaffold drift surfacing (Phase 2, particular to Q4 — lifecycle-semantics.md preexisting drift), epistemic-skill application (Phase 3), comparison synthesis (Phase 4), recommendation (Phase 5).

## Phase 1 — Evidence

Three candidate criterion families (A: time-based, B: evidence-staleness-based, C: influence-saturation-based) across six dimensions. Cells cite source documents. Cells whose answer depends on a yet-unresolved decision (e.g., §2.6, §2.4, ontology.md Q3 or Q5) name the dependency rather than guess.

### Base matrix — 3 candidates × 6 dimensions

| Dimension | Candidate A — Time-based | Candidate B — Evidence-staleness-based | Candidate C — Influence-saturation-based |
|---|---|---|---|
| **1. Falsifiability of the criterion** | Passes [`falsifiability-check`](../../../.claude/skills/epistemic/falsifiability-check/SKILL.md) §1.1–§1.4 today. §1.1 violation: a hypothesis past elapsed-time N since promotion that is not in the demotion-candidate set. §1.2 observation: third party reads the promotion event's timestamp (recorded per §2.5 working text) and compares to wall-clock. §1.3 operationalization: N reduces to a structural parameter; time reduces to substrate timestamps. §1.4: clean. | **Pass-with-dependency.** §1.1 and §1.2 are conditional on the "independence ratio" being computable — which requires §2.6 (Evidential Independence Integrity, pending) to structurally define independence and §2.4 (Inferential Influence Disclosure, pending) to identify influence-bearing assertions. §1.3 operationalization: the criterion's terms reduce only to other pending invariants today; post-§2.6+§2.4, B reduces to substrate artifacts. §1.4: clean as form; non-circular. | **Pass-with-dependency.** §1.1 and §1.2 require §2.4 (pending) for declared influence to be a structural attribute of assertions. §1.3 operationalization: "scope" is undefined here ([`ontology.md` Q5](../../ontology/ontology.md): how influence propagates — pending). Today, C reduces to one pending invariant plus one open Ontology question; post-§2.4 and post-Q5, C reduces to substrate artifacts. §1.4: clean. |
| **2. Dependency on pending invariants** | **None.** Promotion events already record timestamps per §2.5 working text; the promotion event itself is the structural prerequisite (also pending but the same invariant this RFC composes into). | **§2.6 + §2.4 + [`ontology.md` Q3](../../ontology/ontology.md).** Independence requires §2.6's structural definition and Q3's operationalization; influence-bearing assertions require §2.4's structural disclosure. Three pending dependencies. | **§2.4 + [`ontology.md` Q5](../../ontology/ontology.md).** Declared influence requires §2.4; "scope" requires Q5 (influence propagation). Two pending dependencies. |
| **3. Interaction with §2.5** | §2.5 binding text declares N as a structural parameter at promotion-event level (or at the abstract-`Hypothesis` level per Q2-A.2). The criterion is direct: "demotion candidacy fires when elapsed time since promotion exceeds N". §2.5 carries the criterion as structural property, not as forward reference. | §2.5 binding text either (a) encodes B with explicit forward references to §2.6/§2.4 (precedent in [Charter §2 L41](../../charter/constitutional-charter.md#2-constitutional-invariants), where §2 forward-references §4) or (b) defers the criterion to post-§2.6 redaction. Direct operational text is feasible only after §2.6 is also redacted. | §2.5 binding text encodes C with forward reference to §2.4 and to scope. Same structural pattern as B but one fewer pending dependency. Direct operational text feasible after §2.4 is redacted. |
| **4. Composition with Q2-A.2** | N can live at the abstract-`Hypothesis` level (uniform across subtypes) or at the concrete-subtype level (per-`BehavioralCluster`, per-`CoordinationRing`, etc.). Q2-A.2's abstract+sibling structure does not constrain A; the composition is clean. Per-subtype N permits the structural-enforceability argument that drove Q2's resolution (Finding 1) to extend to demotion parameters. | T can live at the abstract level or per-subtype. The ratio is computed per-hypothesis-record from its provenance neighborhood, independent of subtype. Composition is clean; per-subtype T offers limited additional structural enforceability beyond uniform T, because the input record types (Category III observations of independence) are uniform across subtypes. | K and scope live at the abstract level or per-subtype. Scope's per-subtype definition may differ meaningfully (a `CampaignHypothesis`'s scope is the event-stream around it; a `BehavioralCluster`'s scope is the actor-stream around it) — Q2-A.2's distinct subtypes carry distinct natural scopes. Composition is clean and benefits from per-subtype expression. |
| **5. Defense against §1 Thesis failure mode (recursive belief inflation)** | **Weak.** A timer fires regardless of whether the hypothesis has accumulated independent observations or has become a recursive self-confirmation loop. A demotes a still-valid hypothesis on a timer; A retains an invalid (self-confirming) hypothesis until the timer fires. The [Charter §1 Thesis](../../charter/constitutional-charter.md#1-thesis) failure mode ("confidence in inferences inflates without proportional increase in independent evidence") is exactly the case A is poorly suited to detect. | **Strongest.** B is the closest direct match to the Thesis failure mode. The Thesis names "confidence ... inflates without proportional increase in independent evidence"; B operationalizes a ratio of fresh-and-independent inputs versus influence-bearing inputs. A hypothesis whose support is exhausted is precisely what B's criterion fires on. | **Strong.** C addresses runaway influence directly: if a hypothesis is the dominant influence on a large fraction of derived assertions, further use is no longer independent. The Thesis names "promoted hypotheses re-enter the system as enrichment and silently reinforce themselves" — C's saturation criterion is the structural defense against this. |
| **6. Operational cost / detection latency** | Substrate-only. Latency is the polling interval (or event-driven on time triggers). Cheapest. | Projection-traversal. Computing the ratio requires walking the provenance graph (per §2.3 pending) to count independent vs influence-bearing assertions. More expensive; may lag substrate. | Projection-traversal. Similar to B: requires walking the influence graph (per §2.4 pending) over the hypothesis's scope. May lag substrate. |

### Combination sub-matrix — combination form × consequential dimensions

Combinations are evaluated only on dimensions where the combination form genuinely changes the answer (dimensions 1, 5, 6); dimensions 2, 3, 4 inherit from the component candidates and are not duplicated here.

| Combination form | Falsifiability (dim 1) | Defense (dim 5) | Cost (dim 6) |
|---|---|---|---|
| **Disjunctive (A OR B OR C)** | Each criterion individually falsifiable; the union is falsifiable (any criterion firing produces demotion candidacy). Pass-with-dependency inherited from B and C: the union is operational today for A's branch, conditional for B's and C's branches. | Maximum coverage — A's timer, B's freshness, and C's saturation each catch the cases the others miss. Strongest defense. | Highest — must evaluate all three on every promoted hypothesis. Can be partially mitigated by evaluating B and C only on a schedule and A on every change. |
| **Conjunctive (A AND B AND C)** | Intersection falsifiable; high bar to fire (all three must agree). Same pass-with-dependency for B and C. | Protects against single-criterion false positives but allows obviously-invalid hypotheses to persist if any one criterion is silent. The defense achieved is the minimum of the three components, not the maximum. | High but short-circuitable: evaluate cheap A first; only if A fires, evaluate B and C. |
| **Staged (A then B then C)** | A's first stage fully operational today. B and C stages remain pass-with-dependency. Stage-order falsifiability is intermediate: the stages are sequentially falsifiable but the overall criterion's behavior depends on which stage fired. | Combined — A's timer is necessary but not sufficient; B's freshness OR C's saturation must also fire to demote. Avoids A's weak-defense problem (Candidate A alone) by gating with B/C. | Moderate. A first (cheap, frequent), B and C only when A fires (expensive, rare). Latency dominated by A's interval. |

### Meta-pattern column — threshold-form with deferred parameters

The meta-pattern commits to "criteria of the form *a structural property of the hypothesis crosses a threshold*" without specifying which property or threshold. §2.5 binding text encodes the form; specific parameters are deferred to post-§2.6 (or post-§2.4) RFCs.

| Dimension | Meta-pattern (threshold-form, parameters deferred) |
|---|---|
| **1. Falsifiability** | Form-level falsifiability is achievable (§2.5 binding text declares the criterion's form structurally). Specific evaluation requires the parameters; today, the meta-pattern's binding text is operational at the form level but the runtime check is undefined until parameters are set. |
| **2. Dependency** | Logical dependency on §2.6 OR §2.4 (depending on which family the eventual parameters land in). Dependency is on form-completion, not on a specific invariant. |
| **3. Interaction with §2.5** | §2.5 binding text declares the form (e.g., "demotion candidacy fires when a designated structural threshold criterion is violated") without specifying which criterion. The criterion-specification is a follow-on RFC. |
| **4. Composition with Q2-A.2** | Form is uniform at the abstract-`Hypothesis` level; per-subtype parameters are available when parameters are specified. |
| **5. Defense against recursive belief inflation** | Indirect — the defense achieved depends entirely on which parameters are eventually filled in. As form alone, meta-pattern is silent on the failure mode. |
| **6. Operational cost** | Indeterminate until parameters are set. The form does not commit to substrate-only or projection-traversal. |

### Observation — strongest asymmetries

Two dimensions produce the strongest asymmetries between candidates:

- **Dimension 2 (dependency on pending invariants)** partitions B and C from A. A has zero pending dependencies; B has three; C has two. The asymmetry is decisive for "can this candidate be operationalized today" but is not by itself decisive for "is this candidate right".
- **Dimension 5 (defense against recursive belief inflation)** partitions A from B and C. A is weak; B is strongest; C is strong. The asymmetry is decisive for "does this candidate address the [Charter §1 Thesis](../../charter/constitutional-charter.md#1-thesis) failure mode this RFC defends against".

The two strongest asymmetries point in opposite directions on the candidate ranking: dimension 2 favors A; dimension 5 favors B. The tension between the two is the central feature of Q4's evidence space and is what makes the meta-pattern and the staged-combination form natural candidates for the discussion to consider.

Two further observations:

- **Dimension 4 (composition with Q2-A.2)** does not partition meaningfully — all three candidates compose cleanly with Q2-A.2, though C benefits most from per-subtype expression (each subtype has a natural distinct scope).
- **Dimension 6 (cost)** partitions A from B and C — A is cheap and substrate-only; B and C are projection-traversal. The cost asymmetry favors A but is less decisive than dimensions 2 and 5.

## Phase 2 — Surface scaffold drift

[`docs/ontology/lifecycle-semantics.md`](../../ontology/lifecycle-semantics.md) is at `Scaffold` status; its prose was authored before Q4 was formally posed in [`ontology.md` §Open Questions](../../ontology/ontology.md) and before [`vocabulary-discipline` §4](../../../.claude/skills/ontology/vocabulary-discipline/SKILL.md) added the noun-collapsing-to-single-dimension term as a forbidden synonym. Q4 discussion must surface the drift before deliberation; otherwise the scaffold's implicit candidate preference and its vocabulary collapse would silently bias the committee.

### Finding F-DRIFT-1 — Open Modeling Question 4 narrows Q4's question and pre-commits

**Source line:** [`lifecycle-semantics.md` line 53](../../ontology/lifecycle-semantics.md), reproduced here in paraphrase to avoid propagating its non-canonical noun into this scratch (see F-DRIFT-2 below). The lifecycle-semantics question, labeled "Independence-driven lifecycle", asks whether a hypothesis whose independence value (using a non-canonical noun whose rewrite per [`vocabulary-discipline` §4](../../../.claude/skills/ontology/vocabulary-discipline/SKILL.md) picks one of `confidence` or `evidential independence`) falls below a threshold should be automatically considered for demotion, or whether demotion is always operator-driven.

**Compared to canonical Q4** in [`ontology.md` line 57](../../ontology/ontology.md):

> "4. When does a promoted hypothesis become a candidate for demotion? Lifecycle rule."

**Drift type:** scope narrowing plus binary pre-commitment. The canonical Q4 is open-ended on the criterion family (the candidate space Phase 1 names: A, B, C, combinations, meta-pattern). The lifecycle-semantics narrower Q4 frames the question as a binary within Candidate B's family — its first alternative ("automatic demotion when independence falls below a threshold") is structurally Candidate B; its second alternative ("demotion always operator-driven") collapses the criterion question with the execution-mode question (criterion-when-to-fire and human-or-automated-trigger are independent axes that the binary conflates).

Candidates A and C are not framed in the lifecycle-semantics question. A reader encountering the scaffold's Q4 without the canonical Q4 in hand would reasonably believe Candidate B-or-operator-only is the resolution space.

**Authorship-timing artifact, not deliberate narrowing.** The lifecycle-semantics scaffold predates the Q4 RFC and predates [`decision-log §0009`](../../charter/decision-log.md) (the §2.5 prep plan that opened the canonical Q4 RFC). The scaffold's narrower phrasing is the kind of pre-question-formalization drift that the [`ontology-keeper`](../../../.claude/skills/ontology/ontology-keeper/SKILL.md) skill exists to detect; in this case the skill was created after the scaffold, so detection only became possible retrospectively. The same pattern was surfaced in Q2-1's Phase 2 for the entity-model.md prose.

### Finding F-DRIFT-2 — Vocabulary collapse on a two-dimension term

**Source line:** [`lifecycle-semantics.md` line 53](../../ontology/lifecycle-semantics.md), the phrase that pairs the bare noun `independence` with the loose-scalar noun whose rewrite is recorded in [`vocabulary-discipline` §4](../../../.claude/skills/ontology/vocabulary-discipline/SKILL.md). Both terms are non-canonical in this project; together they compose a single drift.

**Drift type:** vocabulary collapse, two layers:

1. **Term-level drift.** The bare term `independence` is overloaded. The canonical vocabulary ([`CLAUDE.md` §3](../../../.claude/CLAUDE.md); [`glossary.md`](../../glossary.md)) carries `evidential independence` as a full term, defined as *"the second dimension of an inferential assertion, distinct from confidence"*. The bare `independence` standalone is named as a forbidden synonym in the `evidential independence` glossary entry. The lifecycle-semantics line uses the bare term as if it were a single scalar.

2. **Loose-scalar drift.** [`vocabulary-discipline` §4](../../../.claude/skills/ontology/vocabulary-discipline/SKILL.md) lists the loose-scalar noun as a forbidden synonym whose rewrite explicitly addresses this scaffold's failure mode: *"Pick the dimension. [Invariant 2.6 pending] requires both [confidence and evidential independence] to be reported separately; the bare term collapses them."*

The composite phrase in the scaffold is a textbook instance of the exact failure mode §2.6 is being redacted to prevent — recorded in the scaffold prose itself. The drift is consequential for Q4's resolution because **Candidate B's operational form depends on the two-dimension structure being preserved**, not collapsed. A Candidate B that computes a single value as the scaffold prose suggests is silently a different criterion than a Candidate B that operates over a structurally preserved two-dimension `evidential independence` value.

The drift is also a forward-blocker for §2.6's redaction: the scaffold prose will need vocabulary revision when §2.6 is redacted, in addition to any structural updates §2.6 induces.

### Finding F-DRIFT-3 — Implicit candidate preference in §Promotion Mechanism

**Source lines:** [`lifecycle-semantics.md` lines 41 and 44](../../ontology/lifecycle-semantics.md), within the §Promotion Mechanism prose.

> "1. Subject every candidate promotion to evaluation against criteria of maturity, breadth, and confidence." (line 41)
>
> "4. Periodically re-evaluate promoted hypotheses against fresh, independent evidence. Promotion is not permanent." (line 44)

**Drift type:** implicit candidate preference toward staged-combination form (A cadence + B measurement basis).

- **Line 41** is about promotion criteria, not demotion. It names three criteria (maturity, breadth, confidence) which are not mapped to Q4's A/B/C candidate families. The line is neutral on Q4 directly.
- **Line 44** contains two implicit preferences for the *demotion-candidacy* question:
  - **"Periodically"** suggests time-based cadence (A-flavor). The word does not commit to A as the criterion but does commit to A-flavor scheduling.
  - **"fresh, independent evidence"** suggests B's measurement basis (independence as the criterion's input). The adjective form `independent` is acceptable vocabulary (it does not collapse to a scalar the way the F-DRIFT-2 phrase does); the structural preference, however, points at Candidate B's family.

**Composite reading:** §Promotion Mechanism's prose implicitly favors a staged-combination form — A-flavor cadence (when to re-evaluate) gates B-flavor measurement (what to evaluate). The staged form is one of the combination forms surfaced in Phase 1. The scaffold does not name the combination; it just reads as if it is the resolution. Line 46's explicit deferral *"The exact mechanics of promotion criteria, evaluation cadence, and demotion rules are deferred until committee redaction of the relevant invariants"* is procedurally clean, but the surrounding prose has already structurally voted.

### Implication for Q4 resolution

The three drifts compound:

- **F-DRIFT-1** narrows the candidate space to Candidate B or operator-only, hiding A, C, combinations, and the meta-pattern from a reader who consults only the scaffold.
- **F-DRIFT-2** uses vocabulary that operationally collapses the two-dimension structure §2.6 is being redacted to preserve, undermining Candidate B's own operational requirement.
- **F-DRIFT-3** reads as if the staged-combination form is already the resolution, providing implicit answer-shape pressure on the committee.

**The committee should evaluate Phase 1 evidence under the discipline that the scaffold's implicit preferences are authorship-timing artifacts, not committee-ratified positions** — the same reframing principle Q2-1 surfaced as Finding 2 (scaffold rework cost is Q-resolution cost, not Candidate-A penalty). Phase 1 ranking should not be biased by the scaffold's apparent endorsement of staged A+B; nor should Candidate B's apparent inevitability (per F-DRIFT-1's narrowing) be taken as evidence for Candidate B's correctness.

The scaffold revision that follows Q4's resolution (in the Q4-2 enactment commit) will:

- Replace F-DRIFT-1's narrower Question 4 phrasing with whatever the committee resolves.
- Replace F-DRIFT-2's collapsing phrase with vocabulary that preserves the two-dimension separation (`evidential independence` and `confidence` as separate references, in line with §2.6's structural commitment).
- Make F-DRIFT-3's implicit staged-form preference explicit if that is the resolution, or remove its implicit framing if a different candidate is the resolution.

## Phase 3 — Apply epistemic skills

Apply each skill to each candidate as an abstract structural proposition. Q2-1's Phase 3 established the precedent that the three skills are non-redundant; Q4 reuses the methodology against object-level lifecycle propositions. Combinations and meta-pattern inherit from their component candidates and are not given separate cells unless they introduce findings not present in the components — none arose in Q4's pass.

- **Candidate A proposition:** "A promoted hypothesis becomes a demotion candidate when elapsed time since its promotion event exceeds parameter N."
- **Candidate B proposition:** "A promoted hypothesis becomes a demotion candidate when the ratio of new structurally-independent input records to influence-bearing assertions falls below threshold T."
- **Candidate C proposition:** "A promoted hypothesis becomes a demotion candidate when its declared influence on derived assertions within its scope exceeds ratio K."

### 3 × 3 matrix — candidate × skill

| Skill | Candidate A | Candidate B | Candidate C |
|---|---|---|---|
| **`falsifiability-check`** | §1.1: a promoted hypothesis past elapsed-time N not in the demotion-candidate set. §1.2: third party reads the promotion-event timestamp and compares to wall-clock; no producer cooperation needed. §1.3: N reduces to a structural parameter at promotion-event level (or per-subtype per Q2-A.2); time reduces to substrate timestamps. §1.4 clean. **Verdict: passes all four today.** Strongest at §1.3 — A is reducible to substrate artifacts now. | §1.1 + §1.2 conditional on independence and influence being structurally defined. §1.3: the criterion's terms (`structurally-independent input records`, `influence-bearing assertions`, `ratio`) reduce only to other pending invariants — §2.6 ([`Charter §2.6`](../../charter/constitutional-charter.md#26-evidential-independence-integrity) pending) for independence; §2.4 ([`Charter §2.4`](../../charter/constitutional-charter.md#24-inferential-influence-disclosure) pending) for influence-bearing assertions; [`ontology.md` Q3](../../ontology/ontology.md) for operational form of independence. Post-§2.6 + §2.4, reduces to substrate artifacts. §1.4 clean. **Verdict: pass-with-dependency.** Today not operationalizable as constitutional text; post-redaction operationalizable. | §1.1 + §1.2 conditional on §2.4 (declared influence as a structural attribute of derived assertions) and on [`ontology.md` Q5](../../ontology/ontology.md) (operational definition of `scope`). §1.3: terms reduce only to pending invariants and one open Ontology question today. §1.4 clean. **Verdict: pass-with-dependency.** Today not operationalizable; post-§2.4 + Q5, operationalizable. |
| **`epistemic-separator`** | A operates on promotion-event timestamps (a structural attribute of Category III lifecycle events). Time itself is not a Category I record — it is a structural attribute of every record. No cross-category mixing. A's demotion-candidacy firing produces a new lifecycle event ([`lifecycle-semantics.md` line 33](../../ontology/lifecycle-semantics.md): "All such operations are recorded as immutable events"), not a mutation of the hypothesis. **Verdict: clean.** | B's input is the per-hypothesis provenance neighborhood. **Particular risk per the prompt's named hazard:** if "evidence" in B's prose includes Category III assertions formed under the hypothesis's influence, B is circular — the hypothesis's own influence would count as "evidence" supporting it. **Verdict: passes with documented requirement that "evidence" be strictly Category I observations and/or Category II constructs not formed under this hypothesis's influence.** B's criterion must structurally exclude hypothesis-influenced assertions from the freshness denominator. This requirement is the structural-clarity surface §2.4 (Inferential Influence Disclosure) must provide; B's operationalization depends on §2.4 producing a structural test for "formed under this hypothesis's influence" that the criterion can subtract from. The requirement compounds F-DRIFT-2 from Phase 2: the scaffold's F-DRIFT-2 collapsing phrase vocabulary collapse already loses the structural distinction this skill demands. | C operates on the influence graph — Category III assertions that declare this hypothesis as an influence per §2.4. No cross-category mixing in the criterion's inputs. C's demotion-candidacy firing produces a new lifecycle event, not a mutation. **Risk:** `scope` is structurally undefined today; whether scope is a substrate-level set of assertions or a projection-level construct must be resolved. **Verdict: clean per category boundary, with documented dependency on `scope` being structurally defined (deferred to Q5).** |
| **`ambiguity-reducer`** | Scan A's prose for watchlist hits. `state` may appear in surrounding prose discussing lifecycle state but is not core to A's criterion. `time` and `parameter` are not on the watchlist. `decision` (when discussing operator-driven demotion) is on the watchlist. **Verdict: minor carry-forward.** A's core criterion is the cleanest of the three on this skill — its terms are structural and on no watchlist. | Multiple watchlist hits in B's prose. **`evidence`** (heavy use): the skill's §1 entry forces the writer to pick `observations` (the records themselves) or `observational provenance`. Under B, "fresh evidence" must be operationalized as fresh Category I observations and/or non-influenced Category II/III provenance references. **`independence`** standalone: forbidden synonym per the [`glossary.md` `evidential independence`](../../glossary.md) entry. B's binding prose must use the full `evidential independence` term. **`state`**, **`context`**, **`decision`** advisory. **Verdict: multiple carry-forwards.** `evidence` and `independence` standalone are operationalizability blockers — they must be resolved before B's binding text passes ambiguity discipline. The resolution is structural, not editorial: it cannot be fixed by word substitution; the underlying two-dimension structure §2.6 introduces must be available. | Scan C's prose. **`influence`** is canonical vocabulary; not on watchlist. **`scope`** is not on the watchlist but is structurally undefined today; the legitimate response is the skill's Response 3 (raise as open modeling question) — Q5 is exactly that question. **`context`**, **`state`** may appear in surrounding prose; advisory. **Verdict: minor carry-forward.** C's core criterion vocabulary is clean; its structural undefined surface (`scope`) is already an open Ontology question with its own resolution path. |

### Most consequential epistemic finding across the 9 cells

**Primary:** **Candidate B's `falsifiability-check` is pass-with-dependency on §2.6 + §2.4 redactions.** Today B is un-falsifiable as constitutional text — its terms reduce only to other pending invariants, not to substrate artifacts. Post-§2.6 + post-§2.4, B becomes falsifiable. This is the predicted "operationalizable later but not now" case. The finding does not disqualify B as an Ontology direction; it constrains how §2.5 binding text can encode B: either (a) §2.5 forward-references §2.6 and §2.4 explicitly (precedent in [Charter §2 L41](../../charter/constitutional-charter.md#2-constitutional-invariants) which forward-references §4), or (b) §2.5 defers the criterion specification to post-§2.6 redaction. Both are procedurally available; the choice is itself a meta-recommendation Phase 4 must surface.

**Secondary, related but distinct:** **Candidate B's `epistemic-separator` finding** — B's criterion requires structural exclusion of hypothesis-influenced assertions from the freshness denominator, otherwise B is circular. This is a definitional finding, not a sequencing finding: even after §2.6 is redacted, B's binding text must structurally subtract hypothesis-influenced assertions from "evidence". The requirement compounds F-DRIFT-2's vocabulary collapse — the scaffold's F-DRIFT-2 collapsing phrase phrasing loses precisely the structural distinction this finding demands. Without §2.4 providing a structural test for "formed under this hypothesis's influence", B cannot non-circularly state what counts as fresh evidence.

The primary and secondary findings together establish that **Candidate B is dependent on §2.4 and §2.6 in two distinct ways**: (1) for falsifiability (operationalization), and (2) for non-circularity (definitional). Both bars must be cleared before B's binding text is admissible per [Charter §4](../../charter/constitutional-charter.md#4-constitutional-design-rule).

### Calibration carry-forward from Q2-1

Q2-1's Phase 3 established three calibration observations for future Ontology RFCs. Q4-1 confirms two and refines one:

- **Confirmed: falsifiability §1.3 (operationalization) does most of the work on substrate-touching propositions.** Q4 confirms — A passes today, B and C are pass-with-dependency, all decided at §1.3.
- **Confirmed: ambiguity-reducer surfaces residual carry-forwards.** Q4 confirms — multiple carry-forwards on B's vocabulary; minor on A and C. The Response-3 path (raise as open modeling question) is invoked twice (for `scope` under C, already an open question; for the operational form of `evidence` under B, which composes into §2.6's redaction).
- **Refined: epistemic-separator's most common finding is intra-category structural risk.** Q4-1 refines: the risk extends to *circularity* within a category — B's potential circularity within Category III (hypothesis-influenced assertions counting as evidence for the hypothesis itself) is a category-internal failure mode, not a cross-category conflation. Q2-1 surfaced intra-category flattening; Q4-1 surfaces intra-category circularity. The skill exists to catch both shapes of within-category structural drift, not only cross-category conflation. This is the refined calibration for future Ontology RFCs (Q1 next via §2.3 prep; Q3 via §2.6 prep; Q5 via §2.4 prep).

## Phase 4 — Comparison synthesis

Findings synthesized from Phase 1 (evidence matrices), Phase 2 (scaffold drift surfacing), and Phase 3 (epistemic-skill application). Classified as **asymmetry** (one candidate clearly stronger by evidence-grounded argument), **apparent trade-off that resolves** (a finding whose appearance reframes under Phase 2 or Phase 3), **genuine trade-off** (a substantive difference where no candidate clearly wins), or **tension** (a structural feature of the evidence space that does not resolve under any candidate alone). Numbered in order of consequence.

### Finding 1 — Tension: the two strongest asymmetries point in opposite directions

Sources: Phase 1 dimensions 2 and 5.

Dimension 2 (dependency on pending invariants) partitions Candidate A (zero pending dependencies) from Candidates B (three) and C (two). The asymmetry favors A and is decisive for "can this candidate be operationalized today".

Dimension 5 (defense against the [Charter §1 Thesis](../../charter/constitutional-charter.md#1-thesis) failure mode — recursive belief inflation) partitions Candidate A (weak — timer-only, mismatched with the failure mode) from Candidates B (strongest — direct operational match) and C (strong — addresses runaway influence). The asymmetry favors B and C and is decisive for "does this candidate fulfill the constitutional purpose this RFC defends against".

**The two strongest asymmetries point in opposite directions on the candidate ranking.** Dimension 2 favors A; dimension 5 favors B (and C). This tension is the central feature of Q4's evidence space. It does not resolve under any candidate alone. Subsequent findings — particularly Findings 4 and 7 — explore the structural options for reconciling the tension.

### Finding 2 — Asymmetry: A is fully falsifiable today; B and C are pass-with-dependency

Sources: Phase 3 (`falsifiability-check`) cells A, B, C.

Candidate A passes [`falsifiability-check`](../../../.claude/skills/epistemic/falsifiability-check/SKILL.md) §1.1–§1.4 today; its terms reduce to substrate artifacts (timestamps + a structural parameter). Candidate B is pass-with-dependency on §2.6 + §2.4 + Q3; Candidate C is pass-with-dependency on §2.4 + Q5. The asymmetry is decisive for "can this candidate serve as §2.5 binding text today" — A can; B and C cannot without explicit forward references or deferral.

The asymmetry is not a values judgment; it is a procedural matter of sequencing. B and C are not wrong; they are not yet operationalizable. The dependency-on-pending property recorded in Phase 1 dimension 2 is honored here as a procedural finding, not as a strike.

### Finding 3 — Asymmetry: B and C address the Thesis failure mode; A does not

Sources: Phase 1 dimension 5.

The [Charter §1 Thesis](../../charter/constitutional-charter.md#1-thesis) names recursive belief inflation as the central failure mode this RFC defends against: *"confidence in inferences inflates without proportional increase in independent evidence. Promoted hypotheses re-enter the system as enrichment and silently reinforce themselves."*

Candidate B is the closest direct match — it measures the ratio of fresh-and-independent input records to influence-bearing assertions, exactly the structure the Thesis names. Candidate C addresses the second half of the Thesis quote directly — saturation of influence is recursive reinforcement made structural. Candidate A is timer-only; it is mismatched with both halves of the Thesis quote.

The asymmetry is decisive for "is this candidate adequate to the constitutional purpose". A standalone is structurally inadequate to defend against the Thesis failure mode. B or C is adequate; the choice between them depends on which half of the Thesis the committee weights more heavily, or whether both are required.

### Finding 4 — Apparent trade-off that resolves: staged-combination reconciles Findings 1, 2, 3

Sources: Phase 1 combination sub-matrix (staged row); Phase 2 F-DRIFT-3.

The staged-combination form (A first, then B or C) reconciles the Finding 1 tension:

- **A's substrate-only test is the first gate.** Fully falsifiable today (Finding 2); cheap; substrate-only. §2.5 binding text can declare A as the cadence criterion structurally now.
- **B or C is the deep criterion gated by A.** Evaluated only when A fires. The deep criterion's binding text either forward-references §2.6/§2.4 explicitly or is deferred to post-redaction RFCs.
- **The form addresses the Thesis failure mode** (Finding 3) by ensuring A is necessary but not sufficient — a hypothesis demoted under staged-form must also fail B or C, which are the candidates that actually detect recursive belief inflation.

The staged-combination is what Phase 2's F-DRIFT-3 surfaced as the scaffold's implicit preference (line 44's "periodically re-evaluate ... against fresh, independent evidence" structurally describes A-cadence-gating-B-measurement). Finding 5 (next) addresses why the scaffold's implicit preference is not a committee endorsement, but the structural form's merits stand independently of the scaffold's accidental endorsement.

### Finding 5 — Apparent trade-off that resolves: scaffold's F-DRIFT-3 implicit preference is not committee evidence

Sources: Phase 2 F-DRIFT-3; Q2-1's Phase 4 Finding 2 (precedent).

Phase 2 surfaced that [`lifecycle-semantics.md` line 44](../../ontology/lifecycle-semantics.md)'s prose implicitly favors the staged-combination form. Without surfacing, Finding 4's recommendation could be misread as "the scaffold already says so, so we just ratify the scaffold". The reframing: the scaffold's implicit preference is an authorship-timing artifact (scaffold predates the Q4 RFC and predates [`decision-log §0008`](../../charter/decision-log.md)'s redaction-order plan); it is not committee-ratified evidence for the staged form.

The principle is the same one Q2-1 Finding 2 surfaced: scaffold rework is Q-resolution cost, not Candidate-X-specific cost. Q4-1's analogue: scaffold's implicit preference is Q-resolution drift, not committee-ratified support for any candidate.

Finding 4's recommendation must therefore stand on its own structural merits (Findings 1, 2, 3 reconciled), not on the scaffold's accidental endorsement. If the committee adopts the staged-combination form, the Q4-2 enactment commit revises F-DRIFT-3's implicit framing into explicit committee adoption; if the committee adopts a different form, the Q4-2 enactment removes F-DRIFT-3's implicit framing entirely.

### Finding 6 — Definitional asymmetry: B requires structural test for "formed under this hypothesis's influence"

Source: Phase 3 (`epistemic-separator`) cell B.

Phase 3 surfaced that Candidate B is potentially circular within Category III — if "evidence" in B's prose includes Category III assertions formed under the hypothesis's influence, the hypothesis's own influence would count as evidence supporting it. Even after §2.6 is redacted, B's binding text must structurally subtract hypothesis-influenced assertions from "evidence". The structural test for "formed under this hypothesis's influence" is what §2.4 (Inferential Influence Disclosure) must provide.

The finding compounds F-DRIFT-2: the scaffold's F-DRIFT-2 collapsing phrase vocabulary collapse already loses the structural distinction Finding 6 demands. Without §2.4's structural test, B cannot non-circularly state what counts as fresh evidence.

Implication for §2.5 sequencing: under staged-combination (Finding 4) with B as the deep criterion, §2.5 must wait for §2.4 to specify the influence-test before B's deep criterion is non-circularly statable. Under staged-combination with C as the deep criterion, §2.4 is still required (declared influence is C's input), but C's binding text does not face the same intra-category circularity surface that B does.

### Finding 7 — Genuine trade-off: meta-pattern vs explicit candidate (or combination)

Sources: Phase 1 meta-pattern column.

The meta-pattern (threshold-form with deferred parameters) keeps §2.5 binding text uncommitted to which family eventually fills in the parameters. The explicit candidate (A, B, C, or staged-combination) commits structurally.

Trade-off:

- **Meta-pattern** preserves committee latitude at the cost of §2.5 binding text being silent on which structural property the criterion is checking. The form-level falsifiability is achievable but the runtime check is undefined until a follow-on RFC sets the parameters.
- **Explicit candidate (or combination)** commits the §2.5 binding text to a specific structural surface. The criterion is concrete (or partially concrete, under staged-combination); the committee retains less latitude post-resolution.

This trade-off is genuine: it is a values choice about how much §2.5 binding text should commit at this redaction phase. The discussion phase does not pre-decide.

### Finding 8 — Carry-forward: Q5 (influence propagation) becomes pre-§2.4 dependency under Candidate C

Sources: Phase 1 dimension 2; Phase 3 (`ambiguity-reducer`) cell C.

[`ontology.md` Q5](../../ontology/ontology.md) (how `influence` propagates through derived assertions — transitive, decaying, or both) is already open. Under Candidate C as the deep criterion of staged-combination (or under C standalone), Q5's resolution becomes a pre-§2.4 dependency in the same procedural shape Q2 and Q4 are pre-§2.5 dependencies. Recorded for [`decision-log §0008`](../../charter/decision-log.md)'s redaction-order tracking; not candidate-decisive at the Q4 stage.

### Finding 9 — Composition with Q2-A.2 is clean for all candidates; C benefits most from per-subtype expression

Sources: Phase 1 dimension 4.

All three candidates compose cleanly with [Q2-A.2's abstract+sibling structure](../../charter/decision-log.md): parameters (N for A, T for B, K + scope for C) can live at the abstract `Hypothesis` level (uniform across subtypes) or at the concrete-subtype level (per-`BehavioralCluster`, etc.). Candidate C benefits most from per-subtype expression because each subtype carries a natural distinct `scope` (a `CampaignHypothesis`'s scope is event-stream-around-it; a `BehavioralCluster`'s scope is actor-stream-around-it).

Not candidate-decisive at the Q4 stage but informs how §2.5 binding text expresses the chosen criterion under Q2-A.2's structure.

### Sequencing finding

Q4 has three procedural options for how its resolution composes with §2.5 redaction sequencing:

- **(i) Resolve fully now.** Pick A, B, C, or a combination as a complete criterion. If B or C is picked alone, §2.5 binding text must forward-reference §2.6/§2.4 (precedent: [Charter §2 L41 forward-references §4](../../charter/constitutional-charter.md#2-constitutional-invariants)). If A is picked alone, Finding 3 (defense inadequacy) disqualifies the resolution constitutionally.
- **(ii) Defer to post-§2.6 + §2.4.** §2.5 redaction is blocked until the dependency invariants are redacted. This inverts [`decision-log §0008`](../../charter/decision-log.md)'s redaction order; departing requires a new decision-log entry justifying the reordering.
- **(iii) Split: resolve the form now, defer parameters.** §2.5 binding text declares the form structurally (e.g., staged-combination — A as cadence gate, plus a designated structural test on evidential independence and/or declared influence as the deep criterion); specific parameters and the choice between B and C (or both) is deferred to post-§2.6 + §2.4 RFCs.

**Option (iii) is the recommended sequencing option** under the evidence. It permits §2.5 redaction to proceed (consistent with §0008's order), preserves Finding 3's defense requirement (the form's structural commitment includes a deep criterion gated by A), and respects Finding 2's "B and C cannot operationalize today" without disqualifying them as the eventual deep criteria. The choice between option (i) with explicit B-or-C and option (iii) with deferred-deep-criterion is the meta-recommendation Phase 5 makes explicit.

### Composition finding

Under option (iii) staged-combination, §2.5 binding text under Q2-A.2 reads (illustrative form, not committee text):

> *"A promoted hypothesis becomes a demotion candidate when both of the following hold: (a) the elapsed time since its promotion event exceeds a parameter declared on the promotion event or on the hypothesis's concrete subtype (Candidate A as cadence); and (b) a designated structural test on evidential independence (Charter §2.6 pending) or declared influence (Charter §2.4 pending) — to be specified by a follow-on RFC — fires."*

Under Q2-A.2:

- Both layers (a) and (b) admit per-subtype parameters. The abstract `Hypothesis` carries the structural commitment to the form; concrete subtypes carry subtype-specific parameter values.
- The form is operationalizable today at layer (a); layer (b) is operationalizable after §2.6 or §2.4 (whichever supplies the eventual deep criterion).
- The form does not pre-decide whether the deep criterion is B (evidence-staleness) or C (influence-saturation) or both; that decision lives in the follow-on RFC.

This composition is the structural shape Findings 1–6 collectively point to. It is implementable today and remains stable across §2.6 / §2.4 redactions.

### Summary statement

The evidence does not converge on a single Q4 candidate as Q2 did on A.2. Q4's evidence converges instead on a **structural form — staged-combination — with the deep criterion deferred to a follow-on RFC**. The form encodes A's substrate-only cadence (Finding 2, today-operational) as a necessary gate; the deep criterion's specific shape (B's evidence-staleness, C's influence-saturation, or a combination) is deferred to post-§2.6 + §2.4 RFCs (Findings 2, 6, sequencing option (iii)).

The shape of this recommendation differs from Q2-1's precedent — Q2-1 converged on a candidate (A.2 — the committee extension); Q4-1 converges on a form. Both are legitimate discussion-phase outputs under the methodology established by Q2-1. The methodological observation that "convergence by one finding alone would indicate insufficient deliberation" still holds — Q4-1's convergence rests on multiple findings (1 tension + 2-3 falsifiability/defense asymmetries + 4 reconciliation + 5-6 reframings).

## Phase 5 — Recommendation

The discussion phase recommends adopting **the staged-combination form (sequencing option (iii) per Phase 4 — split form-and-parameters)**: §2.5 binding text declares the demotion-candidacy criterion as a staged combination — Candidate A as the substrate-only cadence gate, plus a designated structural test on evidential independence and/or declared influence as the deep criterion; the deep criterion's specific shape (Candidate B alone, Candidate C alone, or both) is deferred to a follow-on RFC tied to §2.6 and §2.4 redactions. The recommendation rests primarily on Finding 4 (staged-combination structurally reconciles the Finding 1 tension between dim-2 dependency asymmetry and dim-5 defense asymmetry), supported by Finding 2 (A's today-operational falsifiability is the structural property that permits §2.5 redaction to proceed under [`decision-log §0008`](../../charter/decision-log.md)'s redaction order without inverting it) and Finding 3 (the deep criterion gated by A is what addresses the [Charter §1 Thesis](../../charter/constitutional-charter.md#1-thesis) failure mode; A alone is constitutionally inadequate). Finding 5's reframing is load-bearing for justification — the recommendation must stand on the staged form's structural merits, not on Phase 2 F-DRIFT-3's accidental scaffold endorsement. Finding 6 sharpens what B's eventual operationalization must commit to (a structural exclusion mechanism for hypothesis-influenced assertions via §2.4); it does not pre-decide between B and C as the deep criterion. Finding 7 (meta-pattern vs explicit candidate) is acknowledged as a genuine trade-off the committee may weight in either direction — the recommended staged-combination is more committed than the meta-pattern (it names A as the cadence gate structurally) and less committed than a single-candidate resolution (the deep criterion remains open). The convergence shape — a structural form rather than a single candidate — differs from Q2-1's precedent; Q4-1's evidence supports form-level convergence rather than candidate-level convergence, and the methodological observation that "convergence by one finding alone would indicate insufficient deliberation" is preserved (the form-level convergence rests on Findings 1–6 collectively).

### What would reverse this recommendation

The recommendation flips or substantially changes if any of the following emerges:

- **The committee finds A's "necessary gate" structure too restrictive.** If some hypotheses should be demotable on evidence-staleness or influence-saturation alone — before A's timer fires — then the staged form's "both must hold" structure (A AND deep) is wrong, and the disjunctive combination (A OR B OR C) becomes preferable. The evidence-grounded test for this reversal: are there concrete cases where a hypothesis's evidence is exhausted or influence is saturated long before any reasonable A-timer fires, and where waiting for A is itself a Thesis-failure-mode contributor?
- **The committee weights operational simplicity above structural defense.** If the committee judges that A standalone is constitutionally adequate (operator vigilance handles cases A's timer misses), Finding 3's "A is constitutionally inadequate" is overstated and A alone becomes the resolution. This requires committee evidence that the Thesis failure mode is actually addressable by operator vigilance, not only by structural design — a values judgment the discussion phase does not pre-decide.
- **The committee identifies a §2.4 structural surface that cannot supply Finding 6's exclusion mechanism.** If §2.4's eventual redaction does not provide a structural test for "formed under this hypothesis's influence", Candidate B is permanently un-falsifiable (intra-category circularity per Phase 3). The recommendation would flip toward C alone as the deep criterion (C does not face B's circularity surface); the staged-form structure stands but its (b) layer commits to C.
- **The committee adopts the meta-pattern (Finding 7) for committee-latitude reasons.** If the committee judges that §2.5 binding text should be uncommitted to A-as-cadence-gate as well as to the deep criterion, the meta-pattern (threshold-form with deferred parameters) replaces the staged-combination form. The recommendation would soften from "staged-combination form" to "threshold-form, parameters deferred including which family the parameters live in".
- **A new candidate emerges in committee deliberation that the discussion phase did not consider.** Committee extension during resolution is legitimate per Q2-2's precedent (sub-resolution A.2 was a committee extension). If a fourth candidate family is proposed and grounded in Findings 1–9, the recommendation may be displaced.

### Implication for §2.5 redaction sequencing

The recommended resolution **unblocks §2.5 redaction.** Specifically:

- §2.5 binding text can declare the staged-combination form today. Layer (a) — A as cadence gate — is falsifiable today; its structural commitment can be encoded in §2.5 binding text without forward references.
- Layer (b) — the deep criterion — is encoded as a structural reference to "a designated test on evidential independence (Charter §2.6 pending) or declared influence (Charter §2.4 pending)". The forward reference is the precedent established by [Charter §2 L41](../../charter/constitutional-charter.md#2-constitutional-invariants) (which forward-references §4); §2.5's forward-reference is procedurally identical.
- [`decision-log §0008`](../../charter/decision-log.md)'s redaction order (§2.5 → §2.3 → §2.4 → §2.6) **is preserved.** §2.5 redaction proceeds with the staged form; §2.3, §2.4, §2.6 redactions follow in their planned order; a follow-on RFC opens post-§2.4 and/or post-§2.6 to specify the deep criterion.
- The deep-criterion follow-on RFC will itself be a downstream pre-Gate exercise analogous to Q2/Q4 being pre-§2.5 dependencies. The pre-Gate-of-pre-Gate pattern (decision-log §0009's methodological observation 1) extends one level deeper for Q4's deep criterion.
- The Q4-2 enactment commit (next prompt) revises [`lifecycle-semantics.md`](../../ontology/lifecycle-semantics.md) to:
  - Replace F-DRIFT-1's narrower Open Modeling Question 4 with prose reflecting the staged-form resolution and naming the deferred deep criterion as a tracked open question.
  - Replace F-DRIFT-2's F-DRIFT-2 collapsing phrase vocabulary with `evidential independence` and `confidence` as separate references (preserving the two-dimension structure §2.6 mandates).
  - Make F-DRIFT-3's implicit staged-form preference explicit as committee-ratified resolution, with the deep-criterion deferral noted.
  - Revise §Promotion Mechanism step 4's *"Periodically re-evaluate ... against fresh, independent evidence"* to be operational (cadence parameter N referenced; evidence operationalized as Category I observations or non-influenced Category II/III provenance references).
  - Address the carry-forward Q2 inconsistency (Q2-A.2 sub-resolution made the scaffold's uniform-singular lifecycle prose formally inconsistent — see [`decision-log §0010`](../../charter/decision-log.md) Consequences); the Q4 enactment is the natural commit for that revision.
