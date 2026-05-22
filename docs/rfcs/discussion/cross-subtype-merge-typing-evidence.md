# Cross-subtype merge produced-record typing — discussion evidence

**Status:** in-discussion. Not authoritative. Final resolution will be recorded in decision-log and Ontology document revision.

This scratch supports the discussion phase of [`ontology-revision-cross-subtype-merge-typing.md`](../draft/ontology-revision-cross-subtype-merge-typing.md). The phases below mirror the established Ontology-RFC discussion-phase procedure (per [`q2-evidence.md`](./q2-evidence.md), [`q4-evidence.md`](./q4-evidence.md)): per-dimension evidence per candidate (Phase 1), scaffold implicit-assumption surfacing (Phase 2), epistemic-skill application (Phase 3), comparison synthesis (Phase 4), conditional recommendation (Phase 5).

## Phase 1 — Evidence

Three candidates (α / β / γ) per the [draft RFC §Proposal](../draft/ontology-revision-cross-subtype-merge-typing.md). Five dimensions chosen to mirror [`q2-evidence.md`'s dimensions](./q2-evidence.md) with one substitution (dimension 5 reframed for cross-subtype-merge specifics).

| Dimension | α — Third (fifth) concrete subtype `CompositeHypothesis` | β — Abstract record with subtype-elision `MergedHypothesis` | γ — Per-pair canonical-merge typing (table-driven into the existing four) |
|---|---|---|---|
| **1. Proto / type-layer implications** | One new concrete subtype proto. Carries heterogeneous antecedent-hash list + `combined_subtypes` discriminator + union-of-subtype-specific fields. Five total Cat III concrete record types; lifecycle operations polymorphic over all five. | One new abstract-materialized proto. Carries heterogeneous antecedent-hash list + discriminator-list field. Four concrete subtypes unchanged; abstract type becomes substrate-materialized for the first time. | No new types. The existing four subtypes' protos accommodate merged records via one-line extension (heterogeneous antecedent-hash list permitted). Six (or twelve) pair→target table entries are recorded at the entity-model level only — no proto change. |
| **2. Lifecycle composition implications** | Merge-of-merge produces `Composite{Composite, X}` records, which under α's own rules produce yet another composite. Type structure grows combinatorially with composition depth. Each new composite-of-composite must be representable in the proto — either via recursive `combined_subtypes` or via a saturated "merged hypothesis" sentinel. The §2.5 lifecycle ops (promote/demote/dissolve/split) on a `CompositeHypothesis` need typed semantics that the four existing subtypes' rules do not pre-specify. | Merge-of-merge composes naturally: successive merges append to the discriminator-list on the same abstract record type. No type explosion. The §2.5 lifecycle ops on the abstract record need typed semantics, but only one set of rules vs five under α. | Merge-of-merge follows the same pair-table lookup recursively: if {BC, AG} → AG and that AG later merges with CR, the lookup is {AG, CR} → CR (per the table). The produced record is one of the existing four subtypes regardless of merge depth. The §2.5 lifecycle ops on the merged record are the existing subtype's ops — no new rules needed. |
| **3. Query and projection implications** | "All hypotheses with confidence above X" expands from a 4-union to a 5-union. "All `CompositeHypothesis` instances" is a typed query against the new type. "All hypotheses arising from cross-subtype merge" is identical to the previous (typed against `CompositeHypothesis`). Projection ([`projection-model.md`](../../architecture/projection-model.md)) materializes per-subtype views including a new `CompositeHypothesis` view. | "All hypotheses with confidence above X" remains a 4-union plus the abstract type (5 message types but topologically 4 concrete + 1 abstract). "All cross-subtype-merge hypotheses" is a typed query against the abstract. "Which subtypes were merged" requires reading the discriminator-list field — projection materializes it as a derived index. | "All hypotheses with confidence above X" remains a 4-union (no new types). "All cross-subtype-merge hypotheses" is not directly queryable — it requires inspecting each record's antecedent-list for heterogeneous-type antecedents. Projection may materialize a synthetic `IsCrossSubtypeMerge` boolean as a derived column. |
| **4. Extensibility implications** | Adding a sixth Cat III subtype (e.g., a new recognition-pattern output) interacts with α non-trivially: every existing pair {new, X} for X ∈ {BC, AG, CH, CR, Composite} produces a `Composite{new, X}` under α's own rules. Glossary surface grows (one new term — `CompositeHypothesis` — plus the new subtype). | Adding a sixth Cat III subtype interacts with β minimally: the new subtype joins as a fifth concrete; the abstract type accommodates merges of it with any existing subtype via discriminator-list append. Glossary surface grows by one term (the new subtype). | Adding a sixth Cat III subtype requires extending the pair table by C(N,2) − C(N−1,2) = N−1 new pairs (5 new pairs at N=6). Each new pair needs committee defense for its target. Glossary surface grows by one term (the new subtype). |
| **5. Constitutional implications** | §4 criterion 1 (structural enforceability): passes cleanly — `CompositeHypothesis` is a typed sibling, structurally distinct from the four. §2.3 provenance: preserved — antecedent-hash list carries the cross-subtype provenance link. §2.4 `subject_ref_hypothesis`: payload shape gains a fifth case; §0016 Q3 per-Category granularity accommodates it. §2.5 BC1: passes — the merge event itself is the lifecycle event regardless of produced-record typing. | §4 criterion 1: passes at the abstract level; subtype-specific structural constraints from the merged hypotheses are NOT carried into the abstract record's proto (per Cell 1's structural framing). §2.3: preserved — antecedent-hash list carries the link. §2.4 `subject_ref_hypothesis`: payload shape uniform across all cross-subtype merges (always the abstract); §0016 per-Category granularity accommodates the abstract via its existing wildcard semantic. §2.5 BC1: passes. | §4 criterion 1: passes cleanly — produced records are existing typed subtypes, structurally distinct. §2.3: preserved. §2.4 `subject_ref_hypothesis`: payload shape is one of the four existing; no new case. §2.5 BC1: passes. **However**: a `CoordinationRingFormation` produced from {BC, CR} merge vs. produced from BC + CR pattern-recognition is structurally indistinguishable at the type level — readers must inspect the antecedent list to discover provenance differences. This is a mild §4 criterion 1 concern (the distinction is detectable but not type-level-enforced). |

## Phase 2 — Surface scaffold implicit assumptions

The relevant scaffolds are [`entity-model.md` §Cross-subtype operations](../../ontology/entity-model.md) (lines 74–76) and [`lifecycle-semantics.md` §Merge](../../ontology/lifecycle-semantics.md) (line 28). Both are short — combined ≤ 5 lines of prose. The question for this phase: do they explicitly defer, implicitly assume one candidate, or contain a mixed pattern?

### `entity-model.md` §Cross-subtype operations (lines 74–76)

**Verdict: clean deferral, slight α/β framing.**

The prose names exactly two of the three candidates this RFC surfaces: *"Whether the produced record is a third concrete subtype or an abstract record with subtype-elision is a question whose resolution is deferred."* The phrasing reads as a binary choice. Candidate γ (per-pair canonical typing) is NOT named in the scaffold — it is a candidate this evidence document and the draft RFC contribute.

The wording *"a third concrete subtype"* (singular, definite article) reads more naturally under α than under γ (which would produce *outputs* of the four subtypes, not a single third subtype). The wording *"abstract record with subtype-elision"* maps directly to β.

Cost under α: minimal — the scaffold prose remains structurally consistent; the "third concrete subtype" is materialized as `CompositeHypothesis`.

Cost under β: minimal — the scaffold prose's second option maps directly; "abstract record with subtype-elision" becomes the chosen language.

Cost under γ: the scaffold prose needs revision — the binary deferral does not name γ. A revision would rephrase as: *"Whether the produced record is a third concrete subtype, an abstract record with subtype-elision, or a per-pair canonical from the existing four is a question whose resolution is deferred."* The cost is small (one-sentence prose supersession) but the scaffold's current omission of γ is itself a finding: γ is a candidate the original drafter did not consider when authoring the deferral.

### `lifecycle-semantics.md` §Merge (line 28)

**Verdict: implicit deferral; no candidate-preference indication.**

The prose: *"Two hypotheses (same subtype or, per Q2-A.2 cross-subtype resolution deferred to a follow-on revision of `entity-model.md`, different subtypes) are recognized as describing the same underlying phenomenon and combined. Merge is recorded as an immutable event referencing both antecedents and the produced hypothesis."*

The phrase *"the produced hypothesis"* is singular and uniformly applicable — under α, β, or γ, the produced record is "the produced hypothesis." No candidate is privileged.

Cost under all three candidates: minimal — the scaffold prose accommodates any resolution. A redaction-phase supersession of lifecycle-semantics.md would specify the produced-record typing per the chosen candidate.

### Summary — scaffold asymmetry

| Scaffold | Verdict | Cost under α | Cost under β | Cost under γ |
|---|---|---|---|---|
| `entity-model.md` §Cross-subtype operations | Clean deferral, slight α/β framing | Minimal | Minimal | One-sentence prose supersession (γ not currently named) |
| `lifecycle-semantics.md` §Merge | Implicit deferral, no preference | Minimal | Minimal | Minimal |

The scaffolds are uncommitted to a candidate. Unlike Q2 (where the scaffolds implicitly assumed B per [`q2-evidence.md` Phase 2](./q2-evidence.md)), the cross-subtype scaffolds are structurally neutral. The one finding is that γ was not on the original drafter's candidate map; this is documented but not committee-decisive.

## Phase 3 — Apply epistemic skills

Three epistemic skills ([`falsifiability-check`](../../../.claude/skills/epistemic/falsifiability-check/SKILL.md), [`epistemic-separator`](../../../.claude/skills/epistemic/epistemic-separator/SKILL.md), [`ambiguity-reducer`](../../../.claude/skills/epistemic/ambiguity-reducer/SKILL.md)) against each candidate as an abstract structural proposition.

- **Candidate α proposition:** "Cross-subtype merge produces a record of a new typed subtype `CompositeHypothesis` (a fifth sibling extending the abstract Hypothesis)."
- **Candidate β proposition:** "Cross-subtype merge produces a record of the abstract Hypothesis type, NOT bound to any of the four concrete subtypes; the abstract type becomes substrate-materialized."
- **Candidate γ proposition:** "Cross-subtype merge produces a record of one of the four existing concrete subtypes, determined by a pre-declared per-pair table at the entity-model level."

| Skill | α | β | γ |
|---|---|---|---|
| **`falsifiability-check`** | §1.1 violation: a cross-subtype merge record exists whose type is NOT `CompositeHypothesis`. Detectable. §1.2 observation: third-party reads the record's type identifier. Detectable. §1.3 operationalization: `CompositeHypothesis` reduces to a typed proto with antecedent-list + discriminator + union-of-subtype-fields. §1.4 non-circularity: defined in terms of the four existing subtypes + type-system primitives. **Verdict: passes all four.** Strongest at §1.3. | §1.1 violation: a cross-subtype merge record exists whose type is NOT the abstract Hypothesis. Detectable. §1.2 observation: third party reads type. Detectable. §1.3 operationalization: the abstract Hypothesis reduces to a typed proto with antecedent-list + discriminator-list. **Subtype-specific constraints from the merged hypotheses do not carry into the abstract record** — they reduce to runtime / projection-query checks. §1.4: clean. **Verdict: passes §1.1, §1.2, §1.4; §1.3 partial.** Same §1.3-partial-pass shape as Q2 Candidate B per [`q2-evidence.md` Phase 3](./q2-evidence.md). | §1.1 violation: a cross-subtype merge record exists whose type is NOT the pair-table's declared target. Detectable. §1.2 observation: third party reads type, checks against pair-table. Detectable. §1.3 operationalization: each pair-table cell reduces to a typed pair → typed target mapping. §1.4: clean. **Verdict: passes all four.** §1.3 is strongest when the pair-table is itself recorded structurally (e.g., as a versioned operational-definition constant). |
| **`epistemic-separator`** | The new sibling type lives within Category III; the merge event records the cross-subtype origin in antecedents. No cross-category conflation. Operations on `CompositeHypothesis` are the six §2.5 lifecycle operations (same as the four existing subtypes). **Verdict: clean.** | The abstract type lives within Category III. **Risk: the abstract Hypothesis materializing at the substrate layer is structurally analogous to the [Charter §2.2 first forbidden anti-pattern](../../charter/constitutional-charter.md#22-epistemic-separation)** ("unified assertion models — defining a single generic record type with a 'kind' field distinguishing observation from inference from operational construct"). The Charter scopes the anti-pattern to *cross-category* unification; β does NOT cross categories (it stays within Category III). But the structural intuition — abstract type with discriminator-list substituting for typed distinction — carries over with diminished force. Same pattern surfaced under Q2 Candidate B per [`q2-evidence.md` Finding 4](./q2-evidence.md). **Verdict: passes (no cross-category conflation), with intra-category flattening risk by analogy.** | The four existing types remain typed. The pair-table is an entity-model-level structural rule, not an inferential annotation. **No conflation.** **Risk (mild): a `CoordinationRingFormation` produced from {BC, CR} merge is type-indistinguishable from a `CoordinationRingFormation` produced from pattern recognition** (per Phase 1 cell 5). Reading the record requires antecedent-list inspection to recover provenance. This is `epistemic-separator` §3 checklist 4 (typed crossing) at the projection level: type-level reads under-specify the inferential origin. **Verdict: passes, with type-vs-provenance asymmetry flagged.** |
| **`ambiguity-reducer`** | Watchlist scan of α's text. **`composite`** (the proposed type name) is not in canonical vocabulary; if α is the resolution, the term needs glossary entry. The word `composite` is descriptive and may carry connotations (composition-as-mathematical-product) the committee should accept or substitute. **`hypothesis`** is canonical. **Verdict: one substantive carry-forward (term-naming for `CompositeHypothesis` is committee-pending).** | Watchlist scan of β's text. **`abstract`** is well-established as a code-organizational term but its substrate-materialized use is non-trivial. **`merged`** (the proposed type prefix) is procedurally descriptive but doesn't capture the abstract-type-with-discriminator nature. **Verdict: one substantive carry-forward (term-naming for the materialized abstract is committee-pending).** | Watchlist scan of γ's text. **`canonical`** is used in the pair-table cell-name sense; the term is not in canonical vocabulary, but its use here is descriptive (the table cells ARE the canonical pair → target mapping by construction). No new term introduced. **Pair-table phrasing**: should the table be 6 unordered pairs (C(4,2)) or 12 ordered pairs (4×3)? The ordered/unordered question is itself a committee question. **Verdict: one carry-forward (ordered-vs-unordered table cardinality is committee-pending; no new vocabulary).** |

### Most consequential epistemic finding across the nine cells

Two findings stand out:

1. **β's §1.3 partial pass + intra-category flattening risk** is the same shape as Q2 Candidate B's finding (per [`q2-evidence.md`](./q2-evidence.md)). Subtype-specific structural commitments live below the substrate's type-enforced surface under β; they are real but not structurally enforced. Per Charter §4 criterion 1, the project prefers structural enforcement. The discipline argument that disfavored Q2 Candidate B disfavors β here for analogous reasons.

2. **γ's type-vs-provenance asymmetry**: under γ, the produced record's type does not encode that the record arose from cross-subtype merge. The information is in the antecedent list (recoverable) but not in the type (lost from the surface). Whether this asymmetry matters depends on whether downstream readers are expected to type-discriminate by recognition-pattern-output vs. merge-output. The asymmetry is real but mild — it is not a §4 criterion 1 violation; it is an information-locality preference.

α has no comparable finding under any of the three skills.

## Phase 4 — Comparison synthesis

Findings synthesized from Phase 1 (evidence matrix), Phase 2 (scaffold neutrality), and Phase 3 (epistemic skills). Numbered in order of consequence. Each is classified as **asymmetry** (clear evidence-grounded preference), **apparent trade-off that resolves** (Phase 1 trade-off reframed by Phase 2/3), or **genuine trade-off** (substantive difference neither candidate clearly wins).

### Finding 1 — Asymmetry: structural enforceability disfavors β

Sources: Phase 1 cell β5 (constitutional implications); Phase 3 (β, `falsifiability-check`) §1.3 partial pass; Phase 3 (β, `epistemic-separator`) intra-category flattening risk; precedent at [`q2-evidence.md` Finding 1](./q2-evidence.md).

Under β, subtype-specific structural constraints from the merged hypotheses do NOT carry into the abstract `MergedHypothesis` record's proto. They reduce to runtime / projection-query checks. This is the same structural compromise Q2's Candidate B made, and the same Charter §4 criterion 1 preference that disfavored Q2 Candidate B disfavors β here. The asymmetry between α and γ on this dimension is small (both preserve structural enforcement); the asymmetry between {α, γ} and β is the load-bearing one.

### Finding 2 — Asymmetry: composition behavior favors γ over α

Sources: Phase 1 cell 2 (lifecycle composition).

Under α, merge-of-merge produces `Composite{Composite, X}` records — type structure grows combinatorially with composition depth, requiring proto-level discipline for either recursive discriminators or a saturated sentinel. Under γ, merge-of-merge resolves to one of the four existing subtypes via recursive pair-table lookup — no type growth. Under β, merge-of-merge appends to the discriminator-list on the same abstract type — no growth, but inherits β's Finding-1 compromise.

The asymmetry favors γ on the composition dimension. α's type-explosion concern is bounded (committee can pick the saturated-sentinel form), but the discipline burden is non-zero.

### Finding 3 — Apparent trade-off that resolves: γ's pair-table commitment is structural, not operational

Sources: Phase 1 cell γ4 (extensibility); Phase 3 (γ, `epistemic-separator`).

γ requires committee defense of each of the 6 (or 12) pair-table entries. This appears costly compared to α (one new type, no per-pair defense) or β (no per-pair entries at all). The reframing: under α or β, the per-pair semantic differences DO NOT VANISH — they reappear as projection-layer queries (under β) or as `combined_subtypes` enumeration values (under α). γ surfaces them at the entity-model level; the alternatives push them downstream where they are less visible to committee review.

The cost is the same; γ relocates it to the more visible surface. This is analogous to Q2 Finding 2 (scaffold rework is Q2-resolution cost, not Candidate-A cost): γ's pair-table is cross-subtype-merge-resolution cost, not γ-intrinsic cost.

### Finding 4 — Genuine trade-off: type-vs-provenance information locality

Sources: Phase 3 (γ, `epistemic-separator`) type-vs-provenance asymmetry; Phase 1 cells α5 + γ5 (constitutional implications).

Under α, a record's type identifies it as "produced by cross-subtype merge" at the surface level. Under γ, the same provenance information is in the antecedent-list — recoverable but not type-level. Whether this matters depends on the dominant downstream-reader shape:

- If readers commonly type-discriminate by "is this a recognition-pattern output or a merge output?", α is preferable.
- If readers commonly type-discriminate by "what subtype is this?" without caring about lifecycle origin, γ is preferable.

This is a values choice for the committee, not an evidence-grounded asymmetry. Phase 2's scaffolds do not commit either way.

### Finding 5 — Genuine trade-off: glossary surface and gating ceremony

Sources: Phase 1 cell 4 (extensibility).

α introduces one new glossary term (`CompositeHypothesis`). β introduces one (the materialized abstract). γ introduces no new term but introduces the pair-table at the entity-model level, which is itself a structural commitment requiring committee defense. The three trade off differently:

| Candidate | New glossary term | New entity-model structural commitment |
|---|---|---|
| α | Yes (1) | No (the new type extends existing taxonomy) |
| β | Yes (1) | Yes (abstract becomes substrate-materialized) |
| γ | No | Yes (pair-table of 6 or 12 entries) |

Neither shape is universally preferable; the committee picks per its discipline preference.

### Finding 6 — Carry-forward: cross-subtype merge enablement criterion

Sources: [draft RFC §Open Questions](../draft/ontology-revision-cross-subtype-merge-typing.md).

Independent of which candidate resolves the typing question, the cross-subtype merge ENABLEMENT criterion remains open: when may an operator initiate cross-subtype merge? Under all three candidates, the same enablement-criterion follow-on is required (analogous to the within-subtype merge's `ErrTargetWrongType` guard, but inverted). This is a separate RFC; recorded here so the carry-forward is not lost.

### Summary statement

The evidence has one clear asymmetry against β (Finding 1) and one weaker asymmetry between α and γ on composition behavior (Finding 2, favoring γ). The remaining findings are either reframings (Finding 3 — γ's apparent extensibility cost is cross-subtype-merge-resolution cost in general) or genuine trade-offs (Findings 4, 5). The discussion phase converges on **{α, γ} preferred over β**; the choice between α and γ is more contested and depends on downstream-reader workload predictions and the committee's preference for type-vs-provenance information locality.

## Phase 5 — Conditional recommendation

The discussion phase recommends **Candidate γ — Per-pair canonical-merge typing**, with **Candidate α — Third (fifth) concrete subtype** as the conditional fallback.

The recommendation rests on:

- **Finding 1** disqualifies β on structural-enforceability grounds aligned with Charter §4 criterion 1 (precedent at Q2's Candidate B per [`q2-evidence.md`](./q2-evidence.md)).
- **Finding 2** favors γ over α on composition behavior — γ's recursive pair-table lookup contains type growth; α's `Composite{Composite, X}` does not.
- **Finding 3** reframes γ's apparent extensibility cost: the per-pair-defense burden is the cross-subtype-merge-resolution cost regardless of candidate; γ surfaces it at a visible committee surface (entity-model table) rather than burying it downstream.
- **Finding 4** is the conditional dimension: if the committee assesses that downstream readers commonly need to type-discriminate by "is this a merge output?", α is preferable; γ requires antecedent-list inspection for that discrimination. The recommendation is for γ on the assumption that type-discrimination by lifecycle origin is NOT the dominant downstream-reader shape — but the committee retains full latitude to weight Finding 4 differently.

### What would flip the recommendation

The recommendation flips to **α** if any of the following emerges:

- **Downstream-reader pattern of frequent merge-origin type-discrimination.** If projections, replay handlers, or operator-facing tools commonly need to discriminate between "produced by pattern recognition" and "produced by cross-subtype merge" at the type level, γ's information-locality asymmetry (Finding 4) becomes a substantive cost. α surfaces the distinction in the type itself.
- **Committee judgment that the pair-table is over-determinative.** γ pre-commits to a specific subtype-target for each pair (e.g., {BC, AG} → AG). If the committee finds that some pairs have no defensible canonical target, γ has gaps that α (one unified `CompositeHypothesis`) does not.
- **Anticipated subtype churn.** If the four subtypes are themselves expected to be revised (per [`entity-model.md` Open Modeling Question 2](../../ontology/entity-model.md) or other future Ontology work), γ's pair-table needs revision at each subtype change. α's one new type is more stable against subtype-set churn.

The recommendation flips to **β** if:

- **Evidence that the four subtypes share fully identical structure**, such that subtype-specific structural constraints are vacuous. Under such evidence, Finding 1's structural-enforceability concern collapses, and β's operational uniformity becomes preferable. Phase 2's scaffold reading is suggestive of uniform structure (the scaffolds use uniform-applicability prose), but per the Q2 precedent the scaffold uniformity is an authorship-timing artifact and is not committee-ratified evidence.

### Combined-candidate form considered

The draft RFC names **α + γ** as a combined form: γ resolves the pairs that have defensible canonical targets; α resolves pairs without a defensible canonical via `CompositeHypothesis`. This hybrid preserves γ's composition advantage where applicable and falls back to α where γ over-determines. The discussion phase notes this as a viable resolution; the committee may select it directly or use it as a fallback if γ's pair-table proves under-defensible at the redaction phase.

### Methodological observation

Unlike [`q2-evidence.md`](./q2-evidence.md), this discussion did not converge on a single decisive finding. Finding 1 disqualifies β cleanly; the choice between α and γ depends on the weighing of Findings 2, 3, and 4. The recommendation form here is therefore **two-stage**: disqualify β by structural-enforceability discipline; then choose α-vs-γ by committee assessment of downstream-reader patterns and pair-table defensibility. This two-stage pattern is the precedent for future Ontology RFCs that surface more than two candidates — the first stage applies the strongest discipline-grounded filter, the second stage applies the committee-judgment-grounded filter.
