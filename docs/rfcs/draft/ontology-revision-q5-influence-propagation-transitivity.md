# RFC — Ontology Open Question 5 (transitivity half): Transitivity semantic of `influence` propagation

- **Status:** discussion (substantive deliberation complete — recommendation: Candidate τ / transitive closure with β-graph storage strategy; formal resolution pending committee ratification)
- **Authors:** Ghost Trace committee (Q3-α cascade enactment; opened per [`decision-log §0133`](../../charter/decision-log.md); discussion-phase deliberation Phases 3–5 recorded in [`q5-transitivity-evidence.md`](../discussion/q5-transitivity-evidence.md))
- **Date:** 2026-05-22 (opened); 2026-05-22 (deliberation complete)
- **Type:** ontology-revision
- **Affects:** [`docs/ontology/ontology.md`](../../ontology/ontology.md) (Open Question 5 transitivity half closed by resolution; decay half resolved at [`§0020`](../../charter/decision-log.md)); [Charter §2.4](../../charter/constitutional-charter.md#24-inferential-influence-disclosure) (frozen v0.5 — `influenced_by` chain structural surface; Q5 governs multi-step traversal semantic); [`ontology-revision-q3-independence`](./ontology-revision-q3-independence.md) (accepted per [`§0133`](../../charter/decision-log.md) — Q3-α's "reachable via `influenced_by` edge" predicate's structural semantic); [`docs/architecture/canonical-serialization-contract.md`](../../architecture/canonical-serialization-contract.md) (Q5 resolution feeds the contract revision per [`§0133`](../../charter/decision-log.md) follow-on schedule); [`ontology-revision-layer-b-deep-criterion`](./ontology-revision-layer-b-deep-criterion.md) (Layer B follow-on RFC consumes Q5 resolution as the final ontology-side dependency before substantive content)

> RFCs in Ghost Trace are proposals subject to constitutional review. The [Constitutional Charter](../../charter/constitutional-charter.md) is the authoritative reference. RFCs that conflict with the Charter must either be rejected or accompanied by a corresponding charter amendment (see [`../../charter/amendments.md`](../../charter/amendments.md)).

## Summary

[`ontology.md` Open Question 5](../../ontology/ontology.md): "How does `influence` propagate through derived assertions? Transitive? Decaying? Both?"

The decay half was resolved at [`§0020`](../../charter/decision-log.md) OMQ #2-C — decay is via [§2.5](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) lifecycle event supersession, NOT a runtime parameter. The transitivity half remains open: when an assertion B is `influenced_by` assertion A and A is `influenced_by` hypothesis H, is B structurally `influenced_by` H?

This RFC opens structured discussion. The candidate transitivity-semantic families are enumerated in [`q5-transitivity-evidence.md`](../discussion/q5-transitivity-evidence.md) Phase 2 (four candidates: τ transitive closure; δ direct edge only; κ bounded-depth transitive; β-graph hybrid with consumer-side traversal under published rule). This RFC does not pick a candidate at this phase.

## Motivation

**Why now.** [`§0133`](../../charter/decision-log.md) (Q3-α resolution) adopted Candidate α as the formal definition of `evidential_independence` — the ratio of Cat I primary observation roots NOT reachable via any `influenced_by` edge from a promoted hypothesis. The "reachable via `influenced_by` edge" predicate is structurally undefined until Q5 transitivity-half is resolved; under different transitivity semantics, α's reachability set differs substantially (maximal under τ; minimal under δ; bounded under κ).

α's canonical-serialization-contract revision (the architecture-document follow-on per [`§0133`](../../charter/decision-log.md) schedule) cannot proceed until Q5 transitivity-half is resolved. Q5 is therefore the next pre-Gate in the two-cascade chain Q3 → Q5 → Layer B substantive content.

Opening Q5 transitivity-half at discussion now follows the cascade-enactment pattern established by [`§0015`](../../charter/decision-log.md) (Q1 → Q3-subject-ref) and [`§0020`](../../charter/decision-log.md) (OMQ #2 → OMQ #3). The pattern is fully validated at three instances per [`§0133`](../../charter/decision-log.md) Methodological Observation 3.

**The cost of not resolving.** α's substantive computation is structurally undefined until Q5 transitivity-half resolves. The canonical-serialization-contract revision RFC is blocked. Layer B's substantive content remains gated.

## Constitutional Review

The Q1–Q6 impact analysis prescribed by [`rfc-author` §1](../../../.claude/skills/workflow/rfc-author/SKILL.md).

### Q1 — Which Charter invariants does this RFC touch?

- **§2.4 Inferential Influence Disclosure** (frozen v0.5): touched directly. §2.4 codifies `influenced_by` edges; Q5 transitivity-half governs the structural semantic of multi-step traversal over those edges. The resolution does not modify §2.4 prose; it refines the structural semantic §2.4 inherits per [`§0133`](../../charter/decision-log.md) consequences.
- **§2.6 Evidential Independence Integrity** (frozen v0.6): touched indirectly. α (per [`§0133`](../../charter/decision-log.md) Q3 resolution) is the substrate-committed value §2.6 mandates the pairing of with `confidence`; Q5 resolution determines α's "reachable" predicate.
- **§2.1 Observational Integrity** (frozen): touched at substrate-immutability inheritance. The transitivity semantic operates on substrate-state-at-commit-time; per [`§0021`](../../charter/decision-log.md) the chosen semantic must be write-time evaluable.
- **Layer B follow-on RFC** ([`ontology-revision-layer-b-deep-criterion`](./ontology-revision-layer-b-deep-criterion.md)): Q5 resolution discharges the second of the two Layer B pre-Gates per [`§0133`](../../charter/decision-log.md). Post-Q5 resolution, Layer B's substantive content advances.
- **Canonical-serialization-contract** ([`§0034`](../../charter/decision-log.md)): touched at the storage-shape layer. The chosen semantic + storage shape (direct edges vs closure-annotated vs hybrid) lands in the contract revision per [`§0133`](../../charter/decision-log.md) schedule.

### Q2 — New glossary terms?

Depends on resolution candidate. Per evidence Phase 2:

- Candidate τ may introduce terms naming the transitive-closure semantic (e.g., `influence transitive closure`, `closure-annotated edge`).
- Candidate δ requires no new terms; the direct-edge semantic uses existing `influenced_by` canonical vocabulary.
- Candidate κ introduces parameter K and may surface terms like `bounded-depth transitive`, `influence depth parameter`.
- Candidate β-graph introduces meta-form terms (`published traversal rule`, `consumer-side reachability traversal`).

No glossary modifications in this discussion phase per Q1 / Q3 / OMQ #2 / OMQ #3 precedent.

### Q3 — Resolves an Ontology open question?

**Yes.** This RFC resolves the transitivity half of `ontology.md` Open Question 5. The decay half is already resolved at [`§0020`](../../charter/decision-log.md). Resolution closes Open Question 5 in its entirety.

### Q4 — Charter amendment?

**No.** Q5 resolution refines the structural semantic of `influenced_by` as inherited by §2.4 + §2.6, but does not modify Charter binding text. [`§0133`](../../charter/decision-log.md) Q3 resolution provides the immediate precedent: Q3-α landed without Charter amendment because §2.6 BC1 anticipated the resolution. Q5 transitivity-half similarly lands without Charter amendment — §2.4 codifies the edge structure; Q5 codifies the traversal semantic.

### Q5 — New invariant?

**No.** The RFC resolves an Ontology-level structural semantic that composes with existing invariants (§2.4 declared-influence; §2.6 paired-dimension; §0133 Q3-α). No new Charter invariant.

### Q6 — Ceremony or constitutional?

**Constitutional, not ceremony.** The choice among τ / δ / κ / β-graph is materially different at substrate, projection, and α-computation levels:

- **α-computation:** the reachability set of Cat I roots changes substantially across candidates. Under τ, multi-step chain reachability counts; under δ, only direct listing counts; under κ, bounded depth. The substrate-committed α value differs.
- **§1 Thesis defense:** τ defends most directly against recursive belief inflation through multi-step chains; δ admits the failure mode through indirect chains. Behavioral consequences cascade through α's value-distribution and Layer B's threshold-test.
- **§4 criterion 1 cost:** τ requires closure computation at write time (deterministic per [`§0021`](../../charter/decision-log.md), but O(graph-size) cost per write); δ is cheapest; κ is bounded.
- **Canonical-serialization-contract storage shape:** different across candidates. Substrate footprint, query patterns, and projection-rebuild semantics differ.

Constitutional.

## Proposal

Four candidates enumerated in [`q5-transitivity-evidence.md`](../discussion/q5-transitivity-evidence.md) Phase 2:

- **Candidate τ — Transitive closure.** Substrate's `influenced_by` is transitive; α's reachability admits multi-step chains.
- **Candidate δ — Direct edge only.** Substrate stores only direct edges; α's reachability reduces to direct listing.
- **Candidate κ — Bounded-depth transitive (parameter K).** Transitivity up to depth K; beyond K, structurally non-propagating.
- **Candidate β-graph — Hybrid: substrate stores direct edges, traversal at consumer side under structurally-published rule.** Meta-form orthogonal to τ/δ/κ — composes with rather than competes against them.

This RFC does not pick a candidate at this phase. The discussion scratch records two asymmetries that will likely organize substantive deliberation:

- **§1 Thesis-defense asymmetry:** τ > κ > δ on defense strength against recursive belief inflation through multi-step chains.
- **§4 criterion 1 cost asymmetry:** δ < κ < τ on write-time closure-computation cost.

## Alternatives Considered

Out-of-scope candidate forms surfaced and explicitly rejected during framing:

- **Probabilistic decay-along-chain.** Rejected by [`§0020`](../../charter/decision-log.md) OMQ #2-C resolution — decay is via §2.5 lifecycle supersession, not a runtime decay parameter along the chain. Any candidate that introduces probabilistic decay as a traversal-rule parameter conflicts with §0020. Not enumerated.
- **Runtime-classified transitivity.** Rejected by Q1/Q3-subject-ref/OMQ #2 cumulative precedent — runtime classification at projection violates [§2.2](../../charter/constitutional-charter.md#22-epistemic-separation) epistemic-separation. Not enumerated.
- **Post-commit transitivity mutation.** Rejected by [§2.1](../../charter/constitutional-charter.md#21-observational-integrity) substrate-immutability inheritance for Cat II / Cat III records. Not enumerated.

## Open Questions

The RFC's own open questions to be resolved when it advances:

- **Which candidate (τ / δ / κ / β-graph) does the resolution adopt?** This is the load-bearing sub-decision.
- **If κ: what is the K parameter value?** Uniform K across all hypothesis subtypes (default per [`§0010`](../../charter/decision-log.md) Q2-A.2 uniform-application) or per-subtype K (per [Q3 Phase 4 Finding 9](../discussion/q3-independence-evidence.md))? Parameter values may be deferrable to canonical-serialization-contract revision if structural form is more urgent.
- **If β-graph: what is the published traversal rule's structural form?** The meta-form composes with τ/κ/δ — the published rule is itself one of those three. β-graph collapses to the underlying choice + a storage-shape commitment.
- **Storage shape sub-decision:** direct edges only / closure-annotated / hybrid. Coupled to the chosen semantic but may be a follow-on canonical-serialization-contract-revision question rather than a Q5-substantive one.

## Anti-Patterns to Avoid

Surfaced during framing for committee discipline in subsequent phases:

- **Semantic-storage conflation.** Confusing the structural semantic of reachability (what `influenced_by` MEANS) with the storage shape (HOW the substrate encodes it). β-graph specifically separates these; treating storage as the primary decision risks mis-resolving the semantic. Detection: the resolution must explicitly name the semantic; storage is a follow-on.
- **K-as-procedural.** Treating Candidate κ's parameter K as an operator-configurable runtime constant. K is a structural parameter at the canonical-serialization-contract layer per [`§0021`](../../charter/decision-log.md) write-time-evaluation discipline; operator-configurability would constitute a Charter §4 criterion 1 procedural defense violation. Detection: K's resolution must name it as structural, not configurable.
- **Closure-cost dismissal.** Adopting τ without surfacing the write-time closure cost as a structural commitment. τ's cost is structural; it does not disqualify τ, but it must be acknowledged in the resolution. Detection: the resolution must name τ's cost surface if τ is adopted.

## Migration and Backward Compatibility

No historical Cat II construct or Cat III hypothesis records exist that carry `influenced_by` chains under a yet-to-be-resolved transitivity semantic. The dimension was introduced structurally by §2.4 frozen v0.5 at [`§0099`](../../charter/decision-log.md); Q5 resolution applies forward to all subsequent substrate commits. Pre-Q5-resolution records (i.e., records committed between §2.4 freeze and Q5 resolution) carry direct edges only; whether transitive-closure annotations apply retroactively is itself a Q5 closure consideration. Default per [§2.1](../../charter/constitutional-charter.md#21-observational-integrity) substrate-immutability: records are immutable; if Q5 chooses τ or κ with closure-annotation storage, the annotation applies forward only.

## References

- [`docs/ontology/ontology.md` Open Question 5](../../ontology/ontology.md) — the question this RFC's transitivity-half resolves.
- [Charter §2.4](../../charter/constitutional-charter.md#24-inferential-influence-disclosure) frozen v0.5 — `influenced_by` chain structural surface.
- [Charter §2.6](../../charter/constitutional-charter.md#26-evidential-independence-integrity) frozen v0.6 — `evidential_independence` paired-dimension surface; α's substrate-committed value depends on Q5.
- [`decision-log.md` §0020](../../charter/decision-log.md) OMQ #2-C — decay half resolved.
- [`decision-log.md` §0021](../../charter/decision-log.md) OMQ #3-α — substrate-time generation discipline.
- [`decision-log.md` §0034](../../charter/decision-log.md) — canonical-serialization-contract.
- [`decision-log.md` §0133](../../charter/decision-log.md) — Q3-α resolution; opens this RFC as cascade-enactment.
- [`q5-transitivity-evidence.md`](../discussion/q5-transitivity-evidence.md) — discussion-phase evidence scratch (Phase 1 + Phase 2).
- [`ontology-revision-q3-independence`](./ontology-revision-q3-independence.md) — upstream Q3 resolution.
- [`ontology-revision-layer-b-deep-criterion`](./ontology-revision-layer-b-deep-criterion.md) — downstream RFC Q5 unblocks.

## Decision Record

Substantive deliberation complete; formal resolution pending. The discussion-phase deliberation recorded in [`q5-transitivity-evidence.md`](../discussion/q5-transitivity-evidence.md) Phases 3–5 recommends **Candidate τ (transitive closure)** with **β-graph storage strategy** (substrate stores direct edges + per-record cached closures computed at write time). The recommendation rests on Phase 4 findings:

- **F1** — τ is the only candidate structurally precluding the §1 Thesis failure mode through indirect chains (δ admits it through Cat-II-mediated invisibility; κ partially admits it for chains > K).
- **F2** — τ is the only candidate fully aligned with §2.2 Cat II determinism (Cat II transitively transmits influence from its inputs).
- **F3** — τ's closure-computation cost is bounded by caching (amortized O(input-set-size) per write under β-graph storage).
- **F4** — κ is structurally weaker than τ-with-caching on both discipline and amortized cost.
- **F8** — τ alone fully discharges the [`§0133`](../../charter/decision-log.md) Q3-α follow-on dependencies.

One committee extension: Cat-II structural transmission commitment (per Finding 2, the resolution explicitly commits that Cat II constructs transmit `influenced_by` membership from their inputs).

Resolution lands at a future `decision-log` entry that closes ontology.md Open Question 5 transitivity-half (decay half closed at [`§0020`](../../charter/decision-log.md)), discharges the [`§0133`](../../charter/decision-log.md) Q5-cascade, and unblocks Layer B's substantive content per the two-cascade chain Q3 → Q5 → Layer B. The canonical-serialization-contract revision opens post-resolution per [`§0133`](../../charter/decision-log.md) follow-on schedule.
