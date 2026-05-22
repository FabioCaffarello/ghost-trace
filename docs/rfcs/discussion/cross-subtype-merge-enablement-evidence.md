# Cross-subtype merge enablement criterion — discussion evidence

**Status:** in-discussion. Not authoritative. Final resolution will be recorded in decision-log and Ontology document revision.

This scratch supports the discussion phase of [`ontology-revision-cross-subtype-merge-enablement.md`](../draft/ontology-revision-cross-subtype-merge-enablement.md). Phases mirror the established Ontology-RFC procedure (per [`q2-evidence.md`](./q2-evidence.md), [`q4-evidence.md`](./q4-evidence.md), [`cross-subtype-merge-typing-evidence.md`](./cross-subtype-merge-typing-evidence.md)).

## Phase 1 — Evidence

Four candidates (A / B / C / D) per the [draft RFC §Proposal](../draft/ontology-revision-cross-subtype-merge-enablement.md). Five dimensions; cells cite source documents.

| Dimension | A — Operator-discretionary | B — Shared-actor-membership | C — Per-pair table | D — Lifecycle-state gate |
|---|---|---|---|---|
| **1. Helper-layer implications** | Same-subtype guard simply removed; helper accepts any pair. Implementation surface: 1-line delete in each `Merge*` validation. | Helper inspects both antecedents' actor-set fields (BC: members; AG: members; CR: participants; CH: events→actor-union) and computes intersection. Rejects with `ErrEnablementUnsatisfied` if empty. Implementation surface: per-subtype actor-set extraction + 1 intersection check. | Helper consults a per-pair table at the entity-model level (6 unordered or 12 ordered cells); rejects when cell marks pair as `disabled` or when cell's additional-criterion fails. Implementation surface: table lookup + per-cell criterion dispatch. | Helper inspects each antecedent's lifecycle-state (per §2.5 promotion-event-presence + §0011 Layer A cadence gate) and rejects when either is below the configured threshold. Implementation surface: per-subtype lifecycle-state query + 1 threshold check. |
| **2. Operator-visible semantics** | "Merge whatever you want" — operator-discretion contract identical to within-subtype merge today. Rejected merges produce 4xx with `ErrEnablementUnsatisfied` — under A, never. | "Merge only when you can point at a shared actor" — substrate-grounded structural defense. Operators see clear rejection reason ("antecedents share no actor_ref"). Rejected merges produce 4xx with `ErrEnablementUnsatisfied`. | "Merge only when the pair is enabled in the table, and only when the cell's additional-criterion is satisfied" — most expressive surface; operators consult the table to predict outcomes. | "Merge only when both antecedents are mature enough" — temporal/lifecycle-grounded gate. Operators may need to wait for promotion before merging. |
| **3. Falsifiability surface** | Trivially falsifiable at the §1.1/§1.2 level (merge succeeds when expected, fails when expected — but under A, merge ALWAYS succeeds, so falsifiability is vacuous). §1.3 operationalization: vacuous gate. §1.4 non-circularity: clean (no criterion to circular). | All four falsifiability-check rungs pass cleanly: §1.1 violation (a permitted merge whose antecedents share no actor_ref); §1.2 observation (third party reads both formation records); §1.3 operationalization (intersection of typed actor_ref sets); §1.4 non-circularity (actor_ref is a Cat I primary surface, not derived from the merge). | All four rungs pass at the per-cell level. §1.3 operationalization is strongest when the table is itself recorded structurally (versioned operational-definition constant per [`§0021`](../../charter/decision-log.md) substrate-time-generation). §1.4 non-circularity: clean per-cell; the table's collective defensibility is itself a constitutional-review surface. | All four rungs pass. §1.3 operationalization is direct: lifecycle-state is structurally recorded per §2.5. §1.4: clean — Q4's resolution per [`§0011`](../../charter/decision-log.md) makes lifecycle-state a substrate-grounded property, not a derived-from-the-merge predicate. |
| **4. Extensibility implications** | Adding a fifth subtype: zero-cost. Any pair involving the fifth is permitted automatically. No committee action required. | Adding a fifth subtype requires defining its actor-set surface. If the fifth subtype is event-centric (CH-like), the actor-set is event→actor-union; if actor-centric (BC/AG/CR-like), the field is direct. Per-subtype actor-set extraction is per-subtype proto work. | Adding a fifth subtype requires extending the pair table by N−1 new pairs at N=6 (5 new cells); each cell needs committee defense for its enablement + additional-criterion. Heaviest extension surface. | Adding a fifth subtype requires defining its lifecycle-state surface. Per §2.5 lifecycle ops apply to all Cat III subtypes; the new subtype's promotion event is per-subtype proto work. Operationally similar to B. |
| **5. Constitutional implications** | §4 criterion 1 (structural enforceability): vacuous — no structural gate to enforce. §2.5 BC1: passes — channel-agnostic (no gate is no gate everywhere). §2.3: passes — no provenance traversal at the gate. **However**: the absence of a gate is itself a constitutional choice; under A, the §2.5 BC1 surface treats all cross-subtype merges as structurally identical to within-subtype merges, which is the §0033 local-shell-trust contract extended to cross-subtype. Whether §0033's local-shell-trust contract is the right anchor for HTTP T4 cross-subtype merge is a structural question that A leaves implicit. | §4 criterion 1: passes — actor-set intersection is structurally enforced. §2.5 BC1: passes. §2.3 BC5 (multi-category traversal): touched — actor-set extraction is a Cat I/II provenance traversal at the enablement gate (the four antecedent formation records are Cat I per §2.5 BC1; their actor_ref fields are §2.3 BC5 multi-category-traversal surfaces). Falsifiability of B's gate is direct. | §4 criterion 1: passes per-cell. §2.5 BC1: passes (channel-agnostic table-lookup). §2.3: passes if the table cells reference only the antecedent's identifying surface (not deeper provenance). §0021 substrate-time-generation: the table IS the versioned operational-definition; its committee-recorded version is substrate-time-anchored. | §4 criterion 1: passes — lifecycle-state is structurally recorded. §2.5 BC1: passes — the gate inspects the §2.5 lifecycle-event substrate. §2.3: passes — lifecycle-state-query is a §2.3 BC5 multi-category-traversal but at the within-Cat-III level. §0011 Q4-Layer-A dependency: D operationalizes against Q4's already-resolved Layer A. |

## Phase 2 — Surface scaffold implicit assumptions

The relevant scaffolds are [`entity-model.md` §Cross-subtype operations](../../ontology/entity-model.md) (lines 74–76) and [`lifecycle-semantics.md` §Merge](../../ontology/lifecycle-semantics.md) (line 28).

### `entity-model.md` §Cross-subtype operations

**Verdict: clean deferral on the enablement question; no candidate-preference indication.**

The prose addresses the *typing* question explicitly ("third concrete subtype" vs "abstract record with subtype-elision") but does NOT address the enablement question. The phrase *"Cross-subtype merge ... is structurally permitted"* asserts the operation's existence but does not specify under what criterion. The enablement criterion is implicitly deferred — neither permissive nor restrictive in the scaffold prose.

Cost under all four candidates: minimal. The scaffold's silence on enablement is candidate-neutral. A redaction-phase supersession would add a sentence specifying the criterion per the chosen candidate.

### `lifecycle-semantics.md` §Merge

**Verdict: implicit deferral on the enablement question; slight A-flavor by structural symmetry with within-subtype merge.**

The prose: *"Two hypotheses (same subtype or, per Q2-A.2 cross-subtype resolution deferred to a follow-on revision of `entity-model.md`, different subtypes) are recognized as describing the same underlying phenomenon and combined."*

The phrase *"recognized as describing the same underlying phenomenon"* names the epistemic basis for merge (operator recognition) without specifying a structural criterion. Within-subtype merge today operationalizes this without a structural gate (Candidate A's contract); cross-subtype merge inherits the same "recognized" framing. The scaffold prose is consistent with A, but the scaffold's silence on the cross-subtype enablement question is itself a deferral, not a commitment.

Cost under A: minimal. Symmetric with within-subtype merge. Cost under B/C/D: minor — the prose adds a clause specifying the criterion. The scaffold's current "recognized" framing remains compatible with B/C/D (the criterion gates which recognitions are structurally defensible).

### Summary — scaffold asymmetry

| Scaffold | Verdict | Cost under A | Cost under B | Cost under C | Cost under D |
|---|---|---|---|---|---|
| `entity-model.md` §Cross-subtype operations | Clean deferral on enablement | Minimal | Minimal | Minimal (sentence + table reference) | Minimal |
| `lifecycle-semantics.md` §Merge | Implicit deferral; slight A-flavor (symmetry with within-subtype merge) | Minimal | Minor (clause supersession) | Minor | Minor |

The scaffolds are structurally neutral on enablement — unlike the typing question (where [`cross-subtype-merge-typing-evidence.md` Phase 2](./cross-subtype-merge-typing-evidence.md) surfaced an α/β framing), the enablement question has no scaffold-level candidate framing. This is a structural difference from the typing question: the enablement question is more "blank slate" at the scaffold level.

## Phase 3 — Apply epistemic skills

Three epistemic skills ([`falsifiability-check`](../../../.claude/skills/epistemic/falsifiability-check/SKILL.md), [`epistemic-separator`](../../../.claude/skills/epistemic/epistemic-separator/SKILL.md), [`ambiguity-reducer`](../../../.claude/skills/epistemic/ambiguity-reducer/SKILL.md)) against each candidate as an abstract structural proposition.

- **A:** "Cross-subtype merge is permitted whenever operator initiates it; no structural precondition is enforced."
- **B:** "Cross-subtype merge is permitted only when antecedents share at least one actor_ref."
- **C:** "Cross-subtype merge is permitted per a pre-declared per-pair entity-model table; each cell carries enabled-flag and additional-criterion."
- **D:** "Cross-subtype merge is permitted only when both antecedents are in promoted lifecycle-state (or above a Q4-Layer-A threshold)."

| Skill | A | B | C | D |
|---|---|---|---|---|
| **`falsifiability-check`** | §1.1 violation: a merge rejected as cross-subtype-unenabled. Under A, this never occurs — A asserts no enablement gate; falsifiability is vacuous on the gate-question itself. §1.2: vacuous. §1.3: gate is "true" tautologically; not operationalizable as a check. §1.4: clean (no criterion). **Verdict: vacuous on the gate's own falsifiability — A's claim IS "no claim".** The §4 criterion 1 preference for structural enforcement is not violated under A (there is nothing to enforce); it is simply not exercised. | §1.1: clean — a cross-subtype merge committed despite disjoint actor-sets is a structural anomaly. §1.2: third party inspects both formation records' actor-sets. §1.3: intersection of typed sets reduces to type-system check (`len(A ∩ B) > 0`). §1.4: actor_ref is Cat I per §0023; non-circular. **Verdict: passes all four cleanly.** | §1.1 per-cell: clean. §1.2 per-cell: clean. §1.3 per-cell: depends on the cell's additional-criterion shape; if the cell defers to B (shared-actor) or D (lifecycle-state), the per-cell operationalization is direct. §1.4: the table itself is non-circular (committee-defended); the cells defending against each other for non-circularity is a meta-question. **Verdict: passes per-cell; table-level cohesion is a committee surface.** | §1.1: a cross-subtype merge committed with pre-promotion antecedents is a structural anomaly. §1.2: lifecycle-state queryable per §2.5. §1.3: lifecycle-state is structurally recorded per §0011 Layer A. §1.4: Q4-Layer-A is non-circular per [`§0011`](../../charter/decision-log.md). **Verdict: passes all four cleanly.** |
| **`epistemic-separator`** | No structural enforcement → no cross-category conflation at the gate. **Risk: the absence of a gate is a deferral of structural-check to operator interpretation.** Per epistemic-separator §3 checklist 5 (operator-interpretation independence): under A, two operators initiating cross-subtype merge on the same antecedents under different self-narratives produce structurally-identical commits. The gate-less surface relies on operator discretion, which IS the §0033 contract — but §0033 is explicit about its scope (CLI local-shell-trust); A extends it implicitly to HTTP T4 cross-subtype merge. **Verdict: passes (no conflation), with §0033-scope-extension recorded.** | The actor-set intersection check operates on Cat I primary observations (formation records' actor_ref fields). No cross-category conflation. **No risk.** The check is structurally typed; downstream consumers reading a B-permitted merge record may inspect the antecedents' actor-set intersection deterministically. **Verdict: clean.** | The per-pair table is an entity-model structural rule; not an inferential commitment. **Risk (mild): the table's per-cell additional-criterion may itself defer to operator interpretation** if a cell's text is non-operationalized. The committee defense of each cell IS the §4 falsifiability discipline applied to the table. **Verdict: passes when each cell is operationalized; flagged where cells lack operationalization.** | Lifecycle-state inspection is structurally typed (§2.5 promotion-event substrate-recorded). No conflation. **Risk: the threshold (which promotion-event timestamp counts) couples to Q4-Layer-A's pending sharpening per [`ontology-revision-layer-b-deep-criterion.md`](../draft/ontology-revision-layer-b-deep-criterion.md).** Under D, the enablement gate inherits Layer B's evolution; the gate is a moving target unless the RFC pins it to Layer A's specific form. **Verdict: passes, with Q4-pending-sharpening dependency flagged.** |
| **`ambiguity-reducer`** | Watchlist scan: "operator-discretionary" is descriptive; no new vocabulary. The §0033 contract is canonical. **Verdict: clean.** | **`identity`** appears under B's actor_ref equality semantics; depends on §0023 single-tier resolution. Q2 follow-on (multi-tier identity) may extend B's operationalization. **Verdict: one substantive carry-forward (`identity`-tier evolution affects B's gate semantics).** | **`canonical`** appears in "pre-declared per-pair canonical-enablement table" — descriptive, not new vocabulary. The ordered-vs-unordered table question is itself an `ambiguity-reducer` advisory (the table's cardinality is committee-pending). **Verdict: one carry-forward (table cardinality is committee-pending).** | **`state`** appears in "lifecycle-state criterion" — well-anchored to §2.5. **`threshold`** appears in "Q4-Layer-A threshold" — anchored to [`§0011`](../../charter/decision-log.md). **Verdict: clean — both terms are operationalized at their canonical references.** |

### Most consequential epistemic finding across the twelve cells

Three findings stand out:

1. **A's vacuous falsifiability.** Under A, the gate IS no-gate; falsifiability of the gate-question itself is vacuous. Per Charter §4's preference for structural enforcement (criterion 1), A is the candidate that explicitly DECLINES to enforce. This is not a Charter violation (the Charter requires falsifiability when claims are made; A makes no enablement claim), but it is an absence-of-structure that the committee should weigh against the discipline preference.

2. **B's clean four-rung pass.** B is the most direct match for §4's structural-enforceability discipline: actor-set intersection is typed, computable, and falsifiable without operator interpretation. The only nuance is the Q2-follow-on identity-tier evolution (multi-tier shared-actor matching may extend B later).

3. **D's Q4-pending-sharpening dependency.** D operationalizes against Q4-Layer-A today; Layer B is on hold per [`ontology-revision-layer-b-deep-criterion.md`](../draft/ontology-revision-layer-b-deep-criterion.md). If Layer B resolves with a different sharpening, D's gate inherits the change. This is a *coupling*, not a defect — but it is a coupling the committee should explicitly accept under D.

## Phase 4 — Comparison synthesis

Findings synthesized. Numbered in order of consequence; classified as **asymmetry**, **apparent trade-off that resolves**, or **genuine trade-off**.

### Finding 1 — Asymmetry: structural-enforceability favors B over A

Sources: Phase 1 cells A5 + B5; Phase 3 (A, `falsifiability-check`) vacuous; (B, `falsifiability-check`) clean four-rung pass.

A makes no enablement claim; B makes the strongest structurally-enforceable claim. Per Charter §4 criterion 1 (frozen v0.2), the project prefers structural enforcement over operator-trust. The asymmetry between A and B on this dimension is direct. C and D both pass §4 criterion 1 but with caveats (C's table-cohesion question; D's Q4-pending-sharpening coupling).

The asymmetry weights against A. C and D are intermediate: stronger than A on structural-enforceability but with secondary trade-offs.

### Finding 2 — Asymmetry: operational reach of B vs C

Sources: Phase 1 cell 4 (extensibility); Phase 1 cells B3 + C3 (falsifiability surface).

B operationalizes as a single check (set intersection). C operationalizes as a table-lookup + per-cell criterion dispatch. The two surfaces have different ergonomic properties:

- B is simpler to implement, simpler to test, simpler to extend (one rule); but it may reject epistemically-valid merges where actor-overlap is absent.
- C is heavier to implement (table + per-cell logic) but more expressive (per-pair specificity); it can permit pairs B would reject.

The asymmetry is **conditional on whether per-pair specificity is needed**. If the committee judges that "shared actor_ref" is a uniformly-defensible criterion across all pairs, B is the right candidate. If specific pairs need different criteria, C is the right candidate.

### Finding 3 — Apparent trade-off that resolves: D's Q4-coupling is structural alignment, not defect

Sources: Phase 1 cell D5 (constitutional implications); Phase 3 (D, `epistemic-separator`) Q4-pending-sharpening dependency.

D couples to Q4's Layer A; Layer B is on hold. The apparent concern: D is unstable because Q4 is still evolving. The reframing: D's coupling to Q4 is *structurally aligned* — both questions concern hypothesis maturity for downstream operations; both operationalize against the same §2.5 lifecycle substrate; both share the §0011 Layer A foundation today. D inheriting Layer B's future sharpening is feature, not bug — D's gate sharpens in lockstep with the broader project's notion of hypothesis maturity.

The reframing does not eliminate D's coupling cost (it is real); it relocates the cost from "D-specific instability" to "shared Q4 + D evolution".

### Finding 4 — Genuine trade-off: A's symmetric simplicity vs §4 discipline

Sources: Phase 1 cell A5; Phase 2 (scaffold A-flavor by symmetry with within-subtype merge).

The within-subtype merge today is gate-less per §0033 local-shell-trust. A extends this contract to cross-subtype. The simplicity argument is real: symmetric gateless treatment of all merge ops minimizes implementation surface and matches the scaffolds' implicit framing. The §4 discipline argument is also real: cross-subtype merge crosses a structural boundary (different subtypes) that within-subtype does not, and the operator-trust contract may not extend cleanly to the cross-subtype surface.

Neither side is universally preferable. This is a values choice for the committee: simplicity (extend §0033 contract to cross-subtype) vs discipline (add a structural gate where the within-subtype surface lacks one).

### Finding 5 — Genuine trade-off: HTTP T4 cross-tier obligation

Sources: Phase 1 cell A5 (constitutional implications, second sentence).

Under [`§0094`](../../charter/decision-log.md), the HTTP T4 surface inherits per-actor attribution as a cross-tier obligation. Within-subtype merge satisfies this via the §0119 per-actor IngestionEvent pairing. Under A, cross-subtype merge inherits the per-actor pairing but adds NO further enablement gate; under B/C/D, the gate is structurally defended.

The trade-off: A's "per-actor attribution suffices" framing treats per-actor authentication as the cross-subtype defense (the operator's CN or token_id IS the structural anchor); B/C/D add a substrate-grounded gate on top. Whether per-actor attribution alone suffices is a question about §0094's cross-tier obligation scope; this RFC does not resolve it.

### Finding 6 — Carry-forward: rejection-record question

Sources: Phase 1 cell 1 (helper-layer implications).

Under B/C/D, the gate rejects some merges. Whether the rejection is recorded as a Cat I observation (analogous to `OrphanCleanupAudit` per [`§0104`](../../charter/decision-log.md)) for forensic record OR is silent is a separate question. Within-subtype merge does NOT record rejected attempts today. Recorded for follow-on; does not affect candidate selection.

### Summary statement

The evidence has one clear asymmetry against A (Finding 1: A makes no enablement claim, in tension with §4 discipline). The remaining candidates split:

- B is the cleanest structural-enforceability candidate (Finding 1, Phase 3) but uniform (Finding 2 — may reject epistemically-valid actor-disjoint merges).
- C is the most expressive (per-pair specificity) but heaviest committee surface.
- D is structurally aligned with Q4 (Finding 3) but couples to Q4's pending sharpening.

The discussion phase converges on **{B, C, D} preferred over A**; the choice among B/C/D is more contested.

## Phase 5 — Conditional recommendation

The discussion phase recommends a **two-stage selection**:

**Stage 1 (discipline filter):** disqualify Candidate A on structural-enforceability grounds aligned with Charter §4 criterion 1. The within-subtype merge's gate-less surface is anchored to §0033 local-shell-trust; cross-subtype merge crosses a structural boundary that §0033's local-shell-trust contract was not authored to span. The asymmetry between within-subtype and cross-subtype merge IS the entity-model's structural commitment ([`entity-model.md` §Concrete subtypes](../../ontology/entity-model.md): the four subtypes are *sibling concrete extensions* with structurally-distinct semantic surfaces); cross-subtype merge crosses this commitment and warrants an enablement gate.

**Stage 2 (committee judgment among B/C/D):** the recommendation is **B (shared-actor-membership) with D (lifecycle-state) as a composable additional gate (B+D)**, conditional on committee assessment.

The recommendation rests on:

- **Finding 1** removes A from consideration.
- **Finding 2** prefers B over C absent evidence of per-pair specificity needs. The scaffold reading (Phase 2) does not surface per-pair semantic distinctions; B's uniform criterion is the simpler structural baseline. C is the right candidate IF the committee surfaces specific pair-asymmetries.
- **Finding 3** suggests D is a clean additional gate (Q4-coupling is structural alignment), but D alone is conservative — pre-promotion cross-subtype merge may be legitimate when actor-overlap is strong. D as a SECONDARY gate on top of B preserves B's substrate-grounded structural defense while adding D's hypothesis-maturity gate.

The combined B+D form is the recommendation: cross-subtype merge permitted when (a) antecedents share at least one actor_ref AND (b) both antecedents are in promoted state per §0011 Layer A. This:

- Aligns with §4 discipline (Finding 1).
- Operationalizes uniformly across all four pairs (no per-cell defense burden — Finding 2 deferred).
- Couples cleanly to Q4 (Finding 3) without inheriting Layer B's pending sharpening (B's actor-set check is independent).

### What would reverse the recommendation

The recommendation flips to **C** if any of the following emerges:

- **Specific per-pair asymmetries.** If the committee surfaces evidence that some cross-subtype pairs warrant different criteria (e.g., {AG, CR} permits without actor-overlap because coordinated automation may be detected before actor-attribution; but {BC, CH} requires actor-overlap), C's per-pair table becomes the right candidate. The expressiveness cost is justified by the structural-defensibility benefit.

The recommendation flips to **D-alone** (without B) if:

- **Evidence that actor-set intersection is too restrictive in practice.** Two campaign-stages by different operator-cells may be the same campaign with no shared actor_ref (per the typing RFC's Open Question). If actor-disjoint same-phenomenon recognition is operationally common, B is too restrictive; D alone (with operator-trust on actor-overlap) is the right balance.

The recommendation flips to **A** only if:

- **The committee's reading of §4 criterion 1 is that operator-trust extends cleanly across the within-subtype/cross-subtype boundary.** This is a discipline-interpretation choice (Finding 1 + Finding 4). The scaffolds' implicit A-flavor (Phase 2) is suggestive but not decisive per the Q2-precedent (scaffold uniformity is authorship-timing artifact).

### Combined-candidate form recorded

The draft RFC names B+D as a combined form. This recommendation adopts B+D as the recommended single form. The committee may select B alone (drop the D gate) or D alone (drop the B gate) without contradicting the evidence; both are weaker but coherent alternatives.

### Methodological observation

This is the **second** Ontology RFC discussion to converge by two-stage filter (the first being [`cross-subtype-merge-typing-evidence.md` Phase 5](./cross-subtype-merge-typing-evidence.md)). The pattern is now established as the precedent for Ontology RFCs that surface 3+ candidates: **stage 1 applies the discipline-grounded filter (Charter §4 criterion 1); stage 2 applies the committee-judgment filter among the survivors**. Two-stage convergence is structurally different from single-finding convergence (Q4's staged-combination per [`§0011`](../../charter/decision-log.md)) and from binary convergence (Q2's Candidate-A-by-asymmetry per [`q2-evidence.md`](./q2-evidence.md)); it suits questions where one candidate is clearly disqualified but the survivors require values-grounded selection.
