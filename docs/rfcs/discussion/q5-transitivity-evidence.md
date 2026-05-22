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

## Phases 3+ — Deferred to substantive deliberation

The following phases are drafted in subsequent RFC commits when the committee deliberates Q5 transitivity-half substantively:

- **Phase 3 — Epistemic-skill application.** Apply [`falsifiability-check`](../../../.claude/skills/epistemic/falsifiability-check/SKILL.md), [`epistemic-separator`](../../../.claude/skills/epistemic/epistemic-separator/SKILL.md), and [`ambiguity-reducer`](../../../.claude/skills/epistemic/ambiguity-reducer/SKILL.md) to each candidate per [Q3 Phase 3 precedent](./q3-independence-evidence.md).
- **Phase 4 — Comparison synthesis.** Rank candidates against §1 Thesis defense, §4 criterion 1 cost, structural simplicity, and α-composition fidelity.
- **Phase 5 — Recommendation.** Single-candidate recommendation with explicit dissent surface + cascade implications. Resolution recorded at a future decision-log entry that closes ontology.md Open Question 5 transitivity-half and discharges the [`§0133`](../../charter/decision-log.md) Q5-cascade.

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
