# Q5 — Transitivity of `influence` propagation — discussion evidence

**Status:** in-discussion. Not authoritative. Final resolution will be recorded in decision-log and Ontology document revision.

This scratch supports the discussion phase of [`ontology-revision-q5-influence-propagation-transitivity`](../draft/ontology-revision-q5-influence-propagation-transitivity.md), opened per [`decision-log.md` §0133](../../charter/decision-log.md) as cascade-enactment from Q3 (formal independence) resolution. Q5's other half — decay — was resolved at [`§0020`](../../charter/decision-log.md) OMQ #2-C (decay via §2.5 lifecycle event supersession). This RFC addresses the remaining transitivity half: when an assertion A is `influenced_by` hypothesis H, and assertion B is `influenced_by` A, is B structurally `influenced_by` H?

The cascade-enactment pattern is established by [`§0015`](../../charter/decision-log.md) (Q1 → Q3-subject-ref) and [`§0020`](../../charter/decision-log.md) (OMQ #2 → OMQ #3); this is the third instance.

This is a strictly-framing scratch: Phase 1 names the question and the dependency surface; Phase 2 enumerates candidate transitivity-semantic families. Phases 3+ (epistemic-skill application, comparison synthesis, recommendation) are drafted in a subsequent RFC commit when the discussion advances substantively.

---

## Phase 1 — Scope and dependencies

### The question

[`docs/ontology/ontology.md` Open Question 5](../../ontology/ontology.md):

> "5. How does `influence` propagate through derived assertions? Transitive? Decaying? Both?"

The decay half was resolved at [`§0020`](../../charter/decision-log.md): decay is via §2.5 lifecycle event supersession, NOT a runtime parameter. The transitivity half remains open and is the subject of this RFC.

Resolved under [Charter §2.4](../../charter/constitutional-charter.md#24-inferential-influence-disclosure) v0.5 vocabulary discipline: §2.4 codifies `influenced_by` edges as declared at FORMATION time by the producer of an inferential-commitment record (Cat II construct, Cat III hypothesis, or Assertion with `subject_ref_construct` / `subject_ref_hypothesis` populated). Whether `influenced_by` is structurally transitive — whether the structural `influenced_by` relation admits multi-step traversal at the substrate layer — is what Q5 transitivity-half resolves.

### Why now

[`§0133`](../../charter/decision-log.md) (Q3 resolution) adopted Candidate α — `evidential_independence` is the ratio of Cat I primary observation roots in the assertion's `subject_ref_*` chain that are NOT reachable via any `influenced_by` edge from a promoted hypothesis, divided by total Cat I roots. The "reachable via any `influenced_by` edge" predicate is structurally undefined until Q5 transitivity-half is resolved:

- Under transitive semantic, "reachable" admits multi-step traversal.
- Under direct-only semantic, "reachable" reduces to "directly listed".
- Under bounded-depth semantic, "reachable" admits multi-step up to depth K.

α's substantive computation cannot land at the canonical-serialization-contract revision until this question is resolved. The cascade fires from [`§0133`](../../charter/decision-log.md) per the established lazy pre-Gate methodology.

### In scope

- The structural semantic of `influenced_by` reachability at the substrate layer: direct edge only, transitive closure, or bounded-depth transitive.
- The interaction with [`§0021`](../../charter/decision-log.md) substrate-time generation: the chosen semantic must be evaluable at write time.
- The interaction with [`§0020`](../../charter/decision-log.md) decay-via-supersession: the chosen semantic operates on the substrate-state-at-commit snapshot; supersession is read at projection time orthogonally.
- The substrate storage shape: direct edges only vs transitive-closure-annotated edges vs hybrid.
- The interaction with [`§0133`](../../charter/decision-log.md) Q3-α resolution: the chosen semantic determines α's "reachable" predicate.

### Out of scope

- **Decay semantic.** Resolved at [`§0020`](../../charter/decision-log.md). The chosen transitivity semantic composes with the §0020 decay-via-supersession resolution: the substrate-committed transitivity-aware value is unmodified by demotion/dissolution; supersession reading at projection time applies supersession orthogonally.
- **Layer B deep criterion shape.** Layer B (per [`ontology-revision-layer-b-deep-criterion`](../draft/ontology-revision-layer-b-deep-criterion.md)) consumes Q5's resolution downstream; Q5 produces the reachability semantic, Layer B specifies how the resulting α value thresholds.
- **Per-subtype variation under [`§0010`](../../charter/decision-log.md) Q2-A.2.** The four concrete Cat III subtypes may surface per-subtype transitivity-variant if empirical pressure surfaces; the default is uniform at the abstract `Hypothesis` level per Q3 [Phase 4 Finding 9](./q3-independence-evidence.md).
- **Identity-tier specifics.** [`entity-model.md` OMQ #1](../../ontology/entity-model.md#open-modeling-questions) — inception-phase single-tier `actor_ref` per [`§0023`](../../charter/decision-log.md) — is forward-referenceable per Q3 precedent.
- **Canonical-serialization-contract revision.** Per [`§0133`](../../charter/decision-log.md) follow-on schedule: opens post-Q5 resolution as architecture-document RFC; not pre-Gate to Q5 closure.

### Resolved dependencies (structural ground present)

| Anchor | What it commits | How Q5 transitivity-half consumes it |
|---|---|---|
| [§2.4](../../charter/constitutional-charter.md#24-inferential-influence-disclosure) frozen v0.5 | `influenced_by` edges declared at formation time by the producer | Q5 governs the structural semantic of multi-step traversal over those edges |
| [`§0020`](../../charter/decision-log.md) OMQ #2-C | Decay via §2.5 lifecycle supersession; not a runtime decay parameter | Q5 transitivity operates on substrate snapshot; decay is orthogonal projection-time concern |
| [`§0021`](../../charter/decision-log.md) OMQ #3-α | Influence values committed at substrate write time | Q5 chosen semantic must be evaluable at write time |
| [`§0133`](../../charter/decision-log.md) Q3-α | `evidential_independence` is source-count ratio; "reachable via influenced_by" is the predicate | Q5 resolves the predicate's structural semantic |
| [§2.6](../../charter/constitutional-charter.md#26-evidential-independence-integrity) frozen v0.6 | `evidential_independence` paired with `confidence` at substrate | Q5 resolution lands in α's substrate-committed value |
| [§2.3](../../charter/constitutional-charter.md#23-provenance-integrity) frozen v0.4 | Typed `subject_ref_*` chains terminating at Cat I | Q5 operates over the influence subgraph, not the provenance subgraph; the two are structurally distinct but composed at α's computation |
| [`§0010`](../../charter/decision-log.md) Q2-A.2 | Four-subtype Cat III taxonomy under abstract `Hypothesis` | Per-subtype variation forward-referenced per [Q3 Phase 4 Finding 9](./q3-independence-evidence.md) |

### Open dependencies (assessed at substantive deliberation)

| Open question | Why potentially blocking | Default disposition |
|---|---|---|
| Substrate storage shape (direct edges vs transitive-closure-annotated vs hybrid) | Tightly coupled to the chosen semantic. Storage shape may itself be a sub-decision deferred to the canonical-serialization-contract revision. | Forward-referenceable per [`§0133`](../../charter/decision-log.md) follow-on schedule — canonical-serialization-contract revision is the structural follow-on, not pre-Gate. |
| [Identity-tier extension](../../ontology/entity-model.md#open-modeling-questions) | Multi-tier extension may affect what counts as "the same hypothesis" across tiers, changing transitive reachability. | Forward-referenceable per [`§0023`](../../charter/decision-log.md). Not anticipated to block. |

### Procedural posture

This RFC is at `discussion` status. Phase 1 (this section) names the dependency surface. Phase 2 (below) enumerates candidate transitivity-semantic families. Phase 3+ (epistemic-skill application, comparison synthesis, recommendation) is drafted in subsequent commits when the committee deliberates substantively. Resolution lands at a future `decision-log` entry that closes ontology.md Open Question 5 transitivity-half and discharges [`§0133`](../../charter/decision-log.md) Q5-cascade.

---

## Phase 2 — Candidate transitivity-semantic families

Four candidate families enumerated. Each candidate cites its structural inputs, the structural shape of the reachability semantic, the constraints from resolved decisions it must satisfy, and the one-line tension surfaced at framing. No candidate is selected at this phase.

### Candidate τ — Transitive closure

**Structural semantic:** an assertion B is structurally `influenced_by` hypothesis H if there exists a chain of `influenced_by` edges B → ... → H of any length ≥ 1. The substrate's `influenced_by` relation is the transitive closure of the declared direct edges.

**α composition:** under τ, α's "Cat I roots NOT reachable via any `influenced_by` edge from a promoted hypothesis" admits multi-step chain reachability. Maximally inclusive influence accounting; α's denominator's "influenced" subset is maximal.

**Storage shape options:** (a) substrate stores direct edges only; transitive closure computed at substrate write time per [`§0021`](../../charter/decision-log.md) when α is generated; (b) substrate stores transitive-closure-annotated edges per record; (c) hybrid — direct edges stored, closure cached per record. Storage shape is a sub-decision; the structural semantic is independent.

**Constraints satisfied:** [`§0021`](../../charter/decision-log.md) write-time evaluable (closure computable at commit using prior substrate state); [`§0020`](../../charter/decision-log.md) supersession-compatible (the substrate-committed value is unmodified; supersession reading at projection time orthogonally re-applies); [`§0133`](../../charter/decision-log.md) Q3-α reachability predicate well-defined.

**One-line tension:** maximally conservative for evidential-independence accounting (matches the §1 Thesis defense most directly), but write-time cost scales with chain depth and graph density — a hypothesis with deep, dense influence trees imposes O(graph-size) closure computation per new assertion.

### Candidate δ — Direct edge only

**Structural semantic:** an assertion B is structurally `influenced_by` hypothesis H only if there is a direct `influenced_by(B, H)` edge declared at B's formation. Multi-step chain reachability is NOT structurally encoded at the substrate.

**α composition:** under δ, α's "reachable via any `influenced_by` edge" reduces to "directly listed in this assertion's `influenced_by` chain". α's denominator's "influenced" subset is minimal — only Cat I roots directly listed as influenced are counted; chains are invisible.

**Storage shape:** substrate stores direct edges only; no closure computation.

**Constraints satisfied:** [`§0021`](../../charter/decision-log.md) write-time evaluable (direct edges are the producer's declaration at formation); [`§0020`](../../charter/decision-log.md) supersession-compatible; [`§0133`](../../charter/decision-log.md) Q3-α reachability predicate well-defined but minimal-reach.

**One-line tension:** structurally simplest (no closure computation, smallest substrate footprint) but admits the [§1 Thesis](../../charter/constitutional-charter.md#1-thesis) failure mode through indirect chains — a hypothesis whose influence flows through 2+ intermediate Cat II constructs is structurally invisible to α's denominator under δ; recursive belief inflation along multi-step chains is undetectable at substrate.

### Candidate κ — Bounded-depth transitive (parameter K)

**Structural semantic:** an assertion B is structurally `influenced_by` hypothesis H if there exists a chain B → ... → H of length ≤ K. Parameter K is a fixed structural constant (or per-subtype constant per [`§0010`](../../charter/decision-log.md) Q2-A.2 forward-reference).

**α composition:** under κ, α's reachability predicate admits multi-step chains up to depth K; beyond K, influence is structurally not propagated.

**Storage shape:** substrate stores direct edges + bounded-depth closure annotations (closure depth ≤ K precomputed per record at write time).

**Constraints satisfied:** [`§0021`](../../charter/decision-log.md) write-time evaluable (bounded closure computable at commit); [`§0020`](../../charter/decision-log.md) supersession-compatible; [`§0133`](../../charter/decision-log.md) Q3-α reachability predicate well-defined parameterized.

**One-line tension:** intermediate between τ and δ — captures multi-step influence within a bounded scope while bounding write-time cost. Introduces parameter K as a structural-or-operational sub-decision; K=1 collapses to δ, K=∞ collapses to τ. The choice of K's specific value (and whether it is uniform or per-subtype) is itself a sub-decision deferred to operational specification.

### Candidate β-graph — Hybrid: direct edges stored, traversal at consumer side under a structurally-published rule

**Structural semantic:** substrate stores direct `influenced_by` edges only. The transitivity rule for consumers (including α's computation) is structurally published as part of the canonical-serialization-contract per [`§0034`](../../charter/decision-log.md), but reachability traversal is performed at consumer side, not substrate side.

**α composition:** α's "reachable" predicate is computed at consumer side using the published traversal rule (e.g., "transitive closure up to depth K" or "transitive closure with supersession-aware filtering"). The α value committed at substrate write time per [`§0021`](../../charter/decision-log.md) is computed BY the producer using the same published rule, so consumer-side projection-replay byte-for-byte match holds per [§2.6](../../charter/constitutional-charter.md#26-evidential-independence-integrity) anti-pattern 2 detection.

**Storage shape:** substrate stores direct edges + the substrate-committed α value (which itself reflects the producer's application of the published traversal rule).

**Constraints satisfied:** [`§0021`](../../charter/decision-log.md) write-time evaluable (the producer applies the rule); [`§0020`](../../charter/decision-log.md) supersession-compatible; [`§0133`](../../charter/decision-log.md) Q3-α reachability predicate well-defined via published rule.

**One-line tension:** decouples storage from semantic — substrate is minimal (direct edges), semantic is published (the traversal rule). Bypasses the τ-vs-κ-vs-δ choice at storage time by deferring it to the published rule. But: the published rule's specific shape is still a Q5-substantive sub-decision; β-graph is meta-form, not a candidate semantic. **β-graph composes with τ/κ/δ rather than competing with them** — it answers "where is reachability computed" not "what is reachability".

### Asymmetries surfaced

Two asymmetries partition the candidate space and will likely organize substantive deliberation:

- **§1 Thesis-defense asymmetry:** τ defends most directly against recursive belief inflation through multi-step chains; δ admits the failure mode through indirect chains; κ defends within depth K. The asymmetry is decisive for [§1 Thesis](../../charter/constitutional-charter.md#1-thesis) discipline questions.
- **Write-time cost asymmetry:** δ is cheapest (no closure); τ is most expensive (full closure per write); κ is bounded. The asymmetry is decisive for [§4 criterion 1](../../charter/constitutional-charter.md#4-constitutional-design-rule) structural-enforceability cost questions (closure computation must be deterministic per [`§0021`](../../charter/decision-log.md); deterministic closure at write time over a deep graph is the cost surface).

These asymmetries are recorded for use by substantive deliberation; they are NOT selections. β-graph is recorded as a composable meta-form orthogonal to τ/κ/δ.

---

## Phase 3 — Apply epistemic skills

Per [Q3 Phase 3 precedent](./q3-independence-evidence.md), three skills ([`falsifiability-check`](../../../.claude/skills/epistemic/falsifiability-check/SKILL.md), [`epistemic-separator`](../../../.claude/skills/epistemic/epistemic-separator/SKILL.md), [`ambiguity-reducer`](../../../.claude/skills/epistemic/ambiguity-reducer/SKILL.md)) are applied to each candidate as an abstract structural proposition. β-graph is a meta-form orthogonal to τ/δ/κ (it answers WHERE reachability is computed, not WHAT reachability means); the matrix below focuses on the three semantic candidates and treats β-graph composability as a separate column.

### Candidate propositions

- **τ proposition:** "An assertion B is structurally `influenced_by` hypothesis H if there exists any chain of `influenced_by` edges B → ... → H of length ≥ 1. The substrate's `influenced_by` relation IS the transitive closure of declared direct edges."
- **δ proposition:** "An assertion B is structurally `influenced_by` hypothesis H if and only if there is a direct `influenced_by(B, H)` edge declared at B's formation. Multi-step chain reachability is NOT structurally encoded."
- **κ proposition:** "An assertion B is structurally `influenced_by` hypothesis H if there exists a chain B → ... → H of length ≤ K, where K is a fixed structural parameter at the canonical-serialization-contract layer."

### 3 × 3 matrix — semantic candidate × skill

| Candidate | `falsifiability-check` | `epistemic-separator` | `ambiguity-reducer` |
|---|---|---|---|
| **τ — Transitive closure** | §1.1: violation = a substrate-committed α value not matching the transitive-closure-based recomputation against the assertion's provenance + influence subgraphs. §1.2: third party reads substrate; recomputes closure; compares. §1.3: reduces to closure computation over substrate-committed edges — fully structural. §1.4: clean (no self-reference; closure is built from declared edges only). **Verdict: passes today.** Cost is the structural concern (deterministic closure per [`§0021`](../../charter/decision-log.md), but worst-case O(graph-size) per write absent caching). | Inputs: full `influenced_by` graph + transitive traversal. **Risk:** chains traverse Cat II constructs in intermediate positions. A Cat II construct C derived from a Cat-III-influenced source (per [§2.2](../../charter/constitutional-charter.md#22-epistemic-separation) deterministic derivation) under τ transitively transmits H's influence to anything formed under C's influence. **This is structurally correct under §2.2:** Cat II is deterministically a view of its inputs; influence on inputs IS influence on the Cat II output. The transitivity through Cat II is not cross-category mixing — it is the structural consequence of Cat II's determinism. **Verdict: clean per category boundary; the resolution must explicitly commit to "Cat II constructs structurally transmit `influenced_by` membership from their inputs".** This is a load-bearing structural commitment. | Terms: `transitive closure`, `chain`, `multi-step`. Watchlist scan: `closure` is mathematical and well-defined; `chain` is graph-theoretic. No watchlist hits in core terms. **Verdict: minor carry-forward.** Vocabulary clean. |
| **δ — Direct edge only** | §1.1: violation = α value not matching direct-edge-only formula. §1.2: substrate-readable; cheapest computation. §1.3: reduces to direct edge lookup. §1.4: clean. **Verdict: passes today at the structural-falsifiability level.** Falsifiability is shallow: it tests only that α matches the direct-edge formula, NOT that the direct-edge formula correctly captures structural influence. Inputs are minimal; the §1 Thesis defense surface is silent on indirect chains. | Inputs: direct edges only. No traversal at substrate. **Risk:** Cat-II-mediated indirect influence is STRUCTURALLY INVISIBLE. Consider: H is a promoted Cat III hypothesis; C is a Cat II construct derived using H as enrichment input; A is a Cat III hypothesis formed under influence of C. Under δ, A's direct `influenced_by` is `{C}`; H is not declared. α's denominator under δ counts A's Cat I roots through C without subtracting for H's transitive influence. **The §1 Thesis cycle — "promoted hypotheses re-enter as enrichment and silently reinforce themselves" — is structurally invisible under δ.** Procedural defense (producer-aware declaration of indirect influences) is required; structural defense is absent. **Verdict: clean per category boundary structurally, but admits Cat-II-mediated indirect influence as the §1 Thesis failure mode.** This is a new instance of the Q3-1 third intra-category failure-mode pattern (opacity of producer-side derivation), here surfaced as Cat-II-mediated invisibility. | Terms: `direct edge only`, `declared at formation`. Watchlist scan: clean. **Verdict: clean.** Vocabulary minimal and structurally well-defined; the structural concern (Cat-II invisibility) surfaced under `epistemic-separator` is not an ambiguity-discipline question. |
| **κ — Bounded-depth K** | §1.1: violation = α value not matching bounded-K closure. §1.2: substrate-readable; bounded traversal. §1.3: reduces to depth-≤-K traversal — fully structural conditional on K being structural. §1.4: clean. **Verdict: passes today, CONDITIONAL on K being structural at the canonical-serialization-contract layer.** If K is operator-configurable runtime, κ collapses to operator-supplied territory analogous to Q3-ε's failure (the substrate's discipline surface for the depth cutoff is at the contract; operator-configurability would invert the structural-vs-procedural balance). | Inputs: direct edges + bounded closure. **Risk:** K too small → K-mediated indirect influence invisible (analogous to δ's Cat-II invisibility, scoped to chains > K). K too large → cost approaches τ. **Risk:** the structural meaning of K — "chains longer than K are NOT influence" — admits no principled choice without empirical reference. A K-aware producer could craft chains of length K+1 to escape detection. **Verdict: clean per category boundary; the K parameter's choice is itself a sub-decision deferred to operational specification, and any specific K admits a K+1 chain-escape failure mode.** Structurally weaker than τ (which has no parameter to game) and weaker than δ (which is honestly minimal). | Terms: `bounded depth`, `parameter K`. Watchlist scan: `bounded` may need operationalization; `parameter` advisory. **Verdict: K's specific value is the central sub-decision the resolution must address — Response 3 (raise as open modeling question for the canonical-serialization-contract revision) applies.** Vocabulary clean at the structural surface; K's value is operationally-deferred. |

### β-graph composability note

β-graph is the meta-form: substrate stores direct edges only; reachability is computed at consumer side using a structurally-published rule. The published rule is itself ONE of τ / δ / κ.

- Under β-graph + τ: substrate stores direct edges; consumers compute transitive closure on read. α's substrate-committed value is computed BY the producer at write time using the same published τ rule, so per-record byte-for-byte projection-replay match holds per [§2.6](../../charter/constitutional-charter.md#26-evidential-independence-integrity) anti-pattern 2 detection. Storage cost is minimal; read-time cost is higher.
- Under β-graph + δ: substrate stores direct edges only; the "published rule" is "direct edge only". β-graph + δ is operationally identical to plain δ.
- Under β-graph + κ: substrate stores direct edges; consumers traverse to depth K. K's value is part of the published rule.

**β-graph is a storage-shape commitment, not a semantic-shape commitment.** The semantic question (τ vs δ vs κ) is logically prior to the storage question (substrate-cache the closure vs compute on read). Phase 5's recommendation addresses the semantic; β-graph is the natural storage strategy that accompanies it.

### Most consequential epistemic finding across the 9 cells

**Primary finding — Cat-II-mediated indirect-influence invisibility under δ (and partially under κ).** Under δ, the §1 Thesis failure mode — "promoted hypotheses re-enter as enrichment and silently reinforce themselves" — is structurally invisible when the cycle traverses a Cat II construct. Under κ, the same invisibility applies to chains longer than K. Under τ, the invisibility is structurally precluded by the transitive closure construction. **This is the load-bearing finding for §1 Thesis discipline.**

The finding extends the Q3-1 [intra-category failure-mode catalogue](./q3-independence-evidence.md) Methodological observation: opacity of producer-side derivation surfaced under Q3-ε; here it manifests as Cat-II-mediated structural invisibility under Q5-δ. The two are different surface forms of the same underlying pattern: when the substrate's discipline does not structurally encode some part of the inferential relationship, the producer's behavior (or the topology's accident) determines whether the §1 Thesis failure mode is detectable. **τ closes this surface entirely; δ leaves it entirely open; κ partially closes it.**

**Secondary finding — τ's structural commitment is also a structural cost.** Under τ, every commit must compute the closure of its `influenced_by` set against substrate-committed prior records. Worst-case cost is O(graph-size); typical case with closure caching at each prior record is amortized O(input-set-size). The cost is real but bounded by caching. The β-graph storage strategy + τ semantic together produce: minimal storage (direct edges) + amortized closure computation (cached per assertion at write time). **The cost surface is named, not dismissed.**

**Tertiary finding — κ's K-parameter is structurally unprincipled.** Any specific K admits a K+1 chain-escape failure mode; smaller K is more invisible; larger K approaches τ at increased cost. The "right" K has no principled structural answer — it is operationally calibrated, which makes κ's structural-falsifiability conditional on the canonical-serialization-contract committing to a specific K value (and re-opening when K is revised). κ is structurally a weaker form of τ for an operational-cost benefit that is itself bounded by τ + caching.

### Calibration carry-forward to future Ontology RFCs

Q5-1 confirms and extends the Q3-1 calibration catalogue:

- **Confirmed: falsifiability §1.3 (operationalization) does most of the work on substrate-touching propositions.** All three semantic candidates decide at §1.3 — τ + δ pass cleanly; κ passes conditional on K being structural.
- **Confirmed: ambiguity-reducer surfaces residual carry-forwards that are themselves structural deferrals.** K's value under κ is Response-3 (open modeling question for the canonical-serialization-contract revision).
- **Extended: the intra-category failure-mode catalogue now has FOUR pattern instances surfaced.** Q2-1 (flattening) + Q4-1 (circularity) + Q3-1 (opacity of producer-side derivation) + Q5-1 (Cat-II-mediated structural invisibility — a surface form of producer-side opacity, scoped to indirect chains). The catalogue continues to be structural rather than enumerative; the patterns reflect how the substrate's discipline surface composes with category discipline.

## Phase 4 — Comparison synthesis

Findings synthesized from Phase 1 (dependency surface), Phase 2 (four-candidate enumeration including β-graph meta-form), and Phase 3 (9-cell epistemic-skill matrix + β-graph composability note). Classified as **asymmetry** / **apparent trade-off that resolves** / **genuine trade-off** / **tension**. Numbered in order of consequence.

### Finding 1 — Asymmetry: τ is the only candidate that structurally precludes the §1 Thesis failure mode through indirect chains

Phase 3 primary finding: under δ and partially under κ, the §1 Thesis failure mode (promoted hypotheses re-entering as enrichment and silently reinforcing themselves) is structurally invisible when the cycle traverses Cat II constructs (or chains longer than K under κ). Under τ, the failure mode is structurally precluded by transitive closure construction. **This is the load-bearing constitutional asymmetry.** [Charter §1 Thesis](../../charter/constitutional-charter.md#1-thesis) is the central failure mode the entire §2.6 invariant exists to defend against; Q3-α was selected (per [`§0133`](../../charter/decision-log.md)) precisely to give §2.6 a substrate-falsifiable mechanism against this failure mode. Adopting δ or κ undermines that mechanism's structural completeness.

### Finding 2 — Asymmetry: τ's structural commitment is consistent with §2.2 Cat II determinism

Phase 3 τ's `epistemic-separator` finding: transitivity through Cat II is structurally correct under [§2.2](../../charter/constitutional-charter.md#22-epistemic-separation) — Cat II is deterministically derived from its inputs; influence on inputs IS influence on the Cat II output. The transitivity is not a cross-category leak; it is the structural consequence of §2.2's determinism commitment. **τ is the candidate that reads §2.2 + §2.4 together coherently.** δ treats Cat II as if it were inferentially independent of its inputs — which contradicts §2.2. κ partially treats Cat II this way (only for chains > K). τ is the only candidate fully aligned with §2.2's determinism.

### Finding 3 — Apparent trade-off that resolves: τ's closure-computation cost is bounded by caching

Phase 3 secondary finding + Phase 2 τ's tension: τ's worst-case closure cost is O(graph-size) per write. Surface reading: τ is too expensive; δ or κ is operationally preferable.

Deeper reading: closure caching at each prior assertion's substrate-committed record reduces the amortized cost to O(input-set-size) per write — the new assertion merges the closures of its direct inputs. The β-graph storage strategy + τ semantic produces minimal storage (direct edges) + amortized closure computation. **The cost surface is named, bounded, and structural — not a procedural defect.** The apparent trade-off resolves: τ's cost is acceptable under the β-graph storage strategy.

### Finding 4 — Asymmetry: κ's K parameter is structurally unprincipled

Phase 3 tertiary finding: any specific K admits a K+1 chain-escape failure mode; the "right" K has no principled structural answer — it is operationally calibrated. **κ is a structurally weaker form of τ for an operational-cost benefit that is itself bounded by τ + caching (Finding 3).** With caching, τ's cost is comparable to κ's, and τ has no parameter to game. κ is dominated by τ-with-caching on both structural discipline AND amortized cost.

### Finding 5 — Apparent trade-off that resolves: β-graph is composable, not competing

Phase 2 named β-graph as orthogonal meta-form; Phase 3 β-graph composability note confirms. **β-graph is the storage strategy that accompanies the chosen semantic, not an alternative semantic.** Phase 5 recommendation is over τ/δ/κ; β-graph's role is to clarify that τ + β-graph (substrate stores direct edges + per-record cached closures) is the operational form τ takes at the canonical-serialization-contract layer.

### Finding 6 — Genuine trade-off: τ's structural completeness vs δ's storage simplicity

The only genuine trade-off in the candidate space: τ commits to the structural completeness of the §1 Thesis defense at the cost of closure computation (bounded by Finding 3); δ commits to storage simplicity at the cost of structural completeness (Finding 1). **The trade-off resolves toward τ on constitutional grounds (Finding 1 + Finding 2), but δ's simplicity is the legitimate alternative weight the committee may consider.** This is the substantive deliberation parameter.

### Finding 7 — Carry-forward: per-subtype application under Q2-A.2 is uniform at inception

Per [Q3 Phase 4 Finding 9](./q3-independence-evidence.md) precedent: the four concrete Cat III subtypes may surface per-subtype transitivity-variant if empirical pressure surfaces; the inception-phase default is uniform at the abstract `Hypothesis` level. **No candidate is constrained by Q2-A.2 composition at inception.**

### Finding 8 — Asymmetry: τ unblocks Layer B substantive content + canonical-serialization-contract revision without further structural deferral

Under τ, α's "reachable via influenced_by" predicate is structurally complete; α's canonical-serialization-contract revision can proceed without additional sub-decisions. Under δ, α's predicate is structurally minimal but the §1 Thesis defense gap (Finding 1) is open. Under κ, α's predicate is parameterized; the canonical-serialization-contract revision must additionally commit to K's value. **τ is the only candidate that fully discharges the [`§0133`](../../charter/decision-log.md) Q3-α follow-on dependencies.** δ leaves the §1 Thesis defense gap open; κ leaves K open.

### Finding 9 — Methodological observation: storage-vs-semantic separation is a new structural pattern for Ontology RFCs

Q5-1 surfaced the storage-vs-semantic separation explicitly (β-graph as meta-form orthogonal to τ/δ/κ). This is the first Ontology RFC where the discussion phase explicitly named a meta-form during candidate enumeration. The pattern: when a question has both a "what does X mean" and a "how is X computed/stored" dimension, the meta-form is the latter and composes with the former. **Future Ontology RFCs with similar shape should surface the meta-form explicitly at Phase 2.**

## Phase 5 — Recommendation

The discussion phase recommends adopting **Candidate τ (transitive closure)** as the structural semantic of `influenced_by` propagation. The recommendation rests on Findings 1 (τ is the only candidate structurally precluding the §1 Thesis failure mode through indirect chains), 2 (τ is the only candidate fully aligned with §2.2's Cat II determinism), 4 (κ is structurally weaker than τ-with-caching on both discipline and amortized cost), and 8 (τ alone fully discharges the [`§0133`](../../charter/decision-log.md) Q3-α follow-on dependencies). Finding 3's reframing of τ's cost (bounded by closure caching under the β-graph storage strategy) eliminates the principal objection to τ. Finding 6's genuine trade-off (τ's structural completeness vs δ's storage simplicity) resolves toward τ on constitutional grounds.

The accompanying storage strategy is **β-graph + τ**: substrate stores direct `influenced_by` edges + per-record cached closures (computed at substrate write time per [`§0021`](../../charter/decision-log.md) by merging the closures of the new assertion's direct input edges). The canonical-serialization-contract revision crystallizes this storage shape per the [`§0133`](../../charter/decision-log.md) follow-on schedule.

The committee extension accompanying τ's selection:

- **Cat-II structural transmission commitment.** The resolution explicitly commits that Cat II constructs structurally transmit `influenced_by` membership from their inputs (per [§2.2](../../charter/constitutional-charter.md#22-epistemic-separation) Cat II determinism, the Cat II output's influence chain is the union of its inputs' influence chains). This is the structural reading Finding 2 surfaced; it is load-bearing for τ's correctness under §2.2 + §2.4 read together.

### What would reverse this recommendation

The recommendation flips or substantially changes if any of the following emerges:

- **Empirical implementation pressure shows closure-computation cost is prohibitive even with caching.** If the typical inception-phase substrate exhibits influence graphs deep + dense enough that even amortized O(input-set-size) per write is operationally unworkable, κ becomes the operational fallback with K chosen large enough to capture realistic chain depths. The evidence-grounded test for this reversal: concrete profiling against an inception-phase substrate with measured chain depths and write rates.
- **A new candidate emerges combining τ's structural completeness with δ's storage minimalism.** β-graph already does this for storage; if a new structural semantic surfaces (e.g., "transitive closure restricted to paths whose intermediate edges are all themselves transitively-influenced", or some other refinement), it may displace τ. Committee extension during resolution is legitimate per the Q2/Q4/Q3 precedent.
- **The committee weights storage simplicity above structural completeness.** Finding 6's trade-off resolves toward τ on constitutional grounds, but the committee may judge that δ's storage simplicity (and the procedural defense layer that compensates for its structural-invisibility) is the inception-phase preference. This requires committee evidence that the producer-aware declaration of indirect influences is reliable in practice — a values judgment the discussion phase does not pre-decide.
- **The Cat-II structural transmission commitment is found to conflict with §2.2 in an unanticipated way.** If, in implementation, the Cat II determinism commitment surfaces a structural form where transitivity through Cat II is not the natural reading (e.g., a Cat II that "summarizes" inputs without inheriting their influence chains), the commitment may need refinement. Empirical pressure would surface this during canonical-serialization-contract revision.

### Implication for Layer B substantive content unblock

τ's adoption fully discharges the two-cascade chain Q3 → Q5 → Layer B per [`§0133`](../../charter/decision-log.md) Phase 5's "Implication for Layer B follow-on RFC unblock". With Q3-α's measurable quantity + Q5-τ's transitivity semantic both structurally complete:

- α's "reachable via influenced_by" predicate is fully operational: transitive closure under τ.
- Layer B's deep criterion threshold-tests α directly — no further ontology-side dependency remains.
- Layer B's substantive content (which combination of Candidate B family from Q4 — evidence-staleness using α — and/or Candidate C family — influence-saturation using α — constitutes the deep criterion) is the next substantive RFC, opening post-Q5-resolution.

### Implication for canonical-serialization-contract revision

τ + β-graph storage strategy lands in the canonical-serialization-contract revision as: direct `influenced_by` edges stored per record + per-record closure annotations (cached at write time). The revision RFC opens post-Q5 resolution per [`§0133`](../../charter/decision-log.md) follow-on schedule. The revision is NOT pre-Gate to Q5 closure; Q5 closure is form-level under τ; storage-shape crystallization is parameter-level following the [`§0024`](../../charter/decision-log.md) + [`§0027`](../../charter/decision-log.md) AP5 step (b) precedent.

### Implication for ontology.md Open Question 5 closure

τ's resolution closes the transitivity half of Open Question 5 (the decay half was closed at [`§0020`](../../charter/decision-log.md)). The full Open Question 5 closes at the Q5 transitivity-half resolution commit; ontology.md is updated to mark Q5 fully resolved.

---

## References

- [`docs/ontology/ontology.md` Open Question 5](../../ontology/ontology.md) — Q5 source line.
- [Charter §2.4](../../charter/constitutional-charter.md#24-inferential-influence-disclosure) frozen v0.5 — `influenced_by` chain structural surface.
- [Charter §2.6](../../charter/constitutional-charter.md#26-evidential-independence-integrity) frozen v0.6 — `evidential_independence` paired-dimension surface; α's substrate-committed value lands here.
- [`decision-log.md` §0020 OMQ #2-C](../../charter/decision-log.md) — decay half resolved; transitivity half is what this RFC addresses.
- [`decision-log.md` §0021 OMQ #3-α](../../charter/decision-log.md) — substrate-time generation; Q5 chosen semantic must be write-time evaluable.
- [`decision-log.md` §0034](../../charter/decision-log.md) — canonical-serialization-contract; Q5 resolution lands in contract revision per [`§0133`](../../charter/decision-log.md) follow-on schedule.
- [`decision-log.md` §0133](../../charter/decision-log.md) Q3-α resolution — opens this RFC as cascade-enactment.
- [`ontology-revision-q5-influence-propagation-transitivity`](../draft/ontology-revision-q5-influence-propagation-transitivity.md) — the draft RFC this scratch supports.
- [`q3-independence-evidence.md`](./q3-independence-evidence.md) — Q3's discussion-phase evidence (precedent for Phase structure + α's reachability predicate).
- [`ontology-revision-layer-b-deep-criterion`](../draft/ontology-revision-layer-b-deep-criterion.md) — downstream RFC Q5 unblocks per the two-cascade chain Q3 → Q5 → Layer B.
