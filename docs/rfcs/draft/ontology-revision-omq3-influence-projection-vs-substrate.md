# RFC — Ontology Open Question (provenance-model OMQ #3): Influence at projection vs substrate

- **Status:** discussion
- **Authors:** Ghost Trace committee (OMQ #2 cascade enactment; [`decision-log §0020`](../../charter/decision-log.md))
- **Date:** 2026-05-19
- **Type:** ontology-revision
- **Affects:** [`docs/ontology/provenance-model.md`](../../ontology/provenance-model.md) (§Inferential Provenance — influence-edge generation timing; §Open Modeling Questions OMQ #3 closed by resolution); [Charter §2.4](../../charter/constitutional-charter.md#24-inferential-influence-disclosure) (pending — binding text depends on this resolution alongside OMQ #2 outcome); [`ontology-revision-omq2-decay-of-influence`](./ontology-revision-omq2-decay-of-influence.md) (accepted at [`§0020`](../../charter/decision-log.md) — Candidate C; this RFC is its cascade enactment per [`§0015`](../../charter/decision-log.md) precedent).

> RFCs in Ghost Trace are proposals subject to constitutional review. The [Constitutional Charter](../../charter/constitutional-charter.md) is the authoritative reference. RFCs that conflict with the Charter must either be rejected or accompanied by a corresponding charter amendment (see [`../../charter/amendments.md`](../../charter/amendments.md)).

## Summary

OMQ #3, recorded in [`provenance-model.md` §Open Modeling Questions](../../ontology/provenance-model.md): when a projection is rebuilt from the substrate, does its computation introduce influence edges? Or are influence edges only generated when influence is *operationally consequential*? Cascade-triggered by OMQ #2 resolution per [`decision-log §0020`](../../charter/decision-log.md). Downstream §2.4 redaction Step 1.1 depends on resolution; under OMQ #2 Candidate C, projection rebuild consults §2.5 lifecycle event chain to materialize current supersession state — this IS OMQ #3's projection-vs-substrate distinction at the projection-state layer, but the *generation* of edges themselves is the distinct question OMQ #3 resolves.

This RFC opens structured discussion with two candidate resolutions (α substrate-time generation; β projection-time generation) and one explicitly rejected alternative (γ runtime classification). The RFC does not pick a candidate.

## Motivation

**Why now.** OMQ #2 Candidate C ([`§0020`](../../charter/decision-log.md)) established supersession as Cat II projection over §2.5 lifecycle event chain. The question OMQ #3 resolves at the substrate layer: are `influenced_by` edges committed at substrate event time (when the inference process produces the influencing relationship) or computed at projection time (from substrate events that "could have influenced")? The distinction affects §2.4 binding text's structural commitment about when influence edges enter the substrate vs the projection layer.

Per the OMQ #2 discussion phase Finding 6 cascade-probability analysis, the cascade firing under Candidate C was anticipated and design-as-expected. This RFC is the cascade enactment per [`§0014`](../../charter/decision-log.md) lazy methodology + [`§0015`](../../charter/decision-log.md)→[`§0016`](../../charter/decision-log.md) precedent.

**The cost of not resolving.** Without OMQ #3 resolution, §2.4 binding text would either silently pick substrate-time vs projection-time generation, or produce vague language deferring the decision to implementation. Both failure modes are forbidden by [`ontology-keeper`](../../../.claude/skills/ontology/ontology-keeper/SKILL.md) discipline. The pattern of committee-resolved Ontology questions ([`§0010`](../../charter/decision-log.md) Q2, [`§0011`](../../charter/decision-log.md) Q4, [`§0015`](../../charter/decision-log.md) Q1, [`§0016`](../../charter/decision-log.md) Q3, [`§0020`](../../charter/decision-log.md) OMQ #2) extends to OMQ #3.

## Constitutional Review

The Q1–Q6 impact analysis prescribed by [`rfc-author` §1](../../../.claude/skills/workflow/rfc-author/SKILL.md).

### Q1 — Which Charter invariants does this RFC touch?

- **§2.4 Inferential Influence Disclosure** (pending — this resolution constrains §2.4 binding text alongside OMQ #2 outcome). Touched directly.
- **§2.1 Observational Integrity** (frozen): touched indirectly — under candidate α (substrate-time generation), `influenced_by` edges are Cat I substrate commitments per §2.1 immutability. Under candidate β (projection-time generation), substrate commits formation events; edges derived at projection — still respecting §2.1 substrate immutability but the substrate's edge surface differs.
- **§2.2 Epistemic Separation** (frozen): touched at Cat I/II separation level — under α, edges are Cat I; under β, edges are Cat II projections from Cat I formation events.
- **§2.3 Provenance Integrity** (frozen v0.4): touched at boundary — §2.3 BC1 separates observational vs inferential provenance scope; §2.3 reads `subject_ref_construct` / `subject_ref_hypothesis` as observational transit, §2.4 reads same edges inferentially. OMQ #3 governs `influenced_by` (inferential) edges only.
- **§2.5 Hypothesis Lifecycle Explicitness** (frozen v0.3): touched via OMQ #2 Candidate C inheritance — supersession reading consumes §2.5-committed events at projection layer regardless of OMQ #3 outcome; OMQ #3 governs the *edge generation* timing, not the supersession timing.
- **Layer B follow-on RFC** ([`ontology-revision-layer-b-deep-criterion`](./ontology-revision-layer-b-deep-criterion.md), on hold per [`§0011`](../../charter/decision-log.md)): OMQ #3 resolution feeds Layer B's deep criterion specification alongside OMQ #2 outcome.

### Q2 — New glossary terms?

Depends on resolution. No new terms expected under either α or β; existing canonical vocabulary (`influenced_by`, `Assertion`, `Hypothesis`, `Category I/II/III`) suffices.

### Q3 — Resolves an Ontology open question?

**Yes.** This RFC IS the resolution of provenance-model.md OMQ #3 (Influence at projection vs substrate). Resolution closes the open question; the §Inferential Provenance section is updated to encode the chosen candidate's structural commitment alongside OMQ #2-C's supersession mechanism.

### Q4 — Charter amendment?

Not directly. The resolution constrains §2.4 binding text, but §2.4 is pending — not amendable yet. §2.4 redaction post-OMQ-#3 will encode both OMQ #2 + OMQ #3 outcomes.

### Q5 — New invariant?

No. The RFC resolves an Ontology-level modeling question that §2.4 will then codify structurally at the Charter level.

### Q6 — Ceremony or constitutional?

Constitutional, not ceremony. Substrate-time vs projection-time edge generation has materially different schemas, query patterns, and projection rebuild semantics:

- Schemas differ (edges as Cat I substrate records vs edges as Cat II projection-derived).
- Query patterns differ ("what influenced X?" reads edges directly under α; derives edges from formation events under β).
- Projection rebuild semantics differ (deterministic edge enumeration under α; deterministic edge derivation under β).
- Layer B specification consumes OMQ #3 outcome as input alongside OMQ #2-C's supersession mechanism.

## Proposal

Two candidate resolutions plus one rejection. Discussion phase may surface additional candidates or hybrid forms; this RFC does not pre-decide.

### Candidate α — Substrate-time generation

**Structural claim.** `influenced_by` edges are committed at substrate event time. When the inference process produces an Assertion under the influence of one or more hypotheses, the influence edges are committed to the primary event log as part of the formation event (or as immediately-following events referencing the formation event). The substrate captures every influence relationship structurally.

**Schemas implication.** Influence-edge commit semantics codified at the substrate layer. Edge type: `(source_assertion, influenced_hypothesis, [optional context fields])` per OMQ #2-C; commit time is bound to formation event time.

**Layer B interaction.** Layer B's deep criterion reads `influenced_by` edges directly from substrate; supersession state (per OMQ #2-C) is the projection layer; edge presence is the substrate layer. Specification consumes substrate edges + §2.5 supersession projection.

**Query pattern.** "What influenced this assertion?" reads `influenced_by` edges from substrate directly; deterministic single-step substrate read.

**§2.2 compliance shape.** Clean. Cat I edges + Cat II projection over §2.5 supersession (per OMQ #2-C). No category crossing introduced by α.

**Pros.**
- Maximal structural commitment — influence relationships captured at substrate at commit time.
- Compatible with §2.1 immutability without conditional commitments.
- Single-step substrate read for current edges (supersession state derived per OMQ #2-C).
- Projection rebuild is deterministic substrate replay.

**Cons.**
- Substrate footprint grows with every influence relationship; storage cost proportional to inference activity.
- Requires inference process to commit edges at formation time, increasing commit complexity.
- May commit edges for influences that are operationally consequential only briefly (subsequently superseded per OMQ #2-C), creating substrate fields that are inert for most of their lifetime.

### Candidate β — Projection-time generation

**Structural claim.** `influenced_by` edges are computed at projection time from substrate formation events. The substrate commits formation events (the inference process's hypothesis-context references at Assertion formation); projection derives the explicit `influenced_by` edges at query/rebuild time by traversing formation events.

**Schemas implication.** No edge type at substrate layer; edges are projection-derived. Formation events carry hypothesis-context references; projection generates edges from those references at projection time.

**Layer B interaction.** Layer B's deep criterion reads edges from projection layer (which itself reads formation events from substrate). Two-step indirection: projection derives edges; Layer B reads derived edges + applies §2.5 supersession per OMQ #2-C.

**Query pattern.** "What influenced this assertion?" computes by reading formation events from substrate, then projection generates edges + applies supersession. Multi-step but deterministic.

**§2.2 compliance shape.** Clean under deterministic-derivation reading. Cat I formation events + Cat II projection (edges + supersession). Risk: if "projection derives edges at projection time" admits inference-process-driven edge generation (rather than deterministic-from-formation-events), collapses to Candidate γ (rejected).

**Pros.**
- Minimal substrate footprint — only formation events committed; influence relationships are projection-derived.
- Commit complexity at substrate layer reduced (inference process commits formation events; edges are downstream).
- Sparse-influence cases (formation event with no influencing hypotheses) cost zero substrate fields.

**Cons.**
- Two-step projection (derive edges + apply supersession) increases query complexity.
- Projection rebuild must be fully deterministic from substrate formation events; any nondeterminism risks collapse to γ.
- Influence relationships are not directly committed — recoverable only by projection over formation events.

### Alternatives Considered

#### Candidate γ — Runtime classification of influence (REJECTED)

**Structural claim.** Influence is determined at runtime by an inference process classifying which hypotheses influenced a given Assertion based on operational context (model state, query parameters, current inference output).

**Why rejected.** Collapses OMQ #3 to runtime — the same failure mode `ontology-keeper` exists to prevent (Q1 Candidate C rejection precedent per [`§0015`](../../charter/decision-log.md); Q3 Candidate C rejection precedent per [`§0016`](../../charter/decision-log.md); OMQ #2 Candidate D rejection per [`§0020`](../../charter/decision-log.md)). Runtime classification places the structural commitment in inference code, not in schemas or types or permitted operations — failing §4 criterion 1 (structurally enforceable). Violates the project's core constitutional discipline.

## Open Questions

What does THIS RFC defer?

- **Specific schemas-technology choices.** Under α: edge commit format, indexing strategy, storage layer. Under β: formation event payload format, projection-derivation algorithm. Deferred per [`§0003`](../../charter/decision-log.md) (substrate-tech selection pending).
- **Sparse-influence representation.** Under α, formation events with zero influencing hypotheses commit no `influenced_by` edges — admissible by edge multiplicity per OMQ #2-C. Under β, formation events carry zero-element hypothesis-context lists — same admissible. No deferral needed beyond schemas-technology.
- **Q5-transitive half** ([`ontology.md` §Open Questions](../../ontology/ontology.md) Q5 "transitive?" axis) remains open regardless of OMQ #3 outcome. The transitive question is structurally distinct from substrate-vs-projection edge generation; it asks how influence propagates through derivation chains. Deferred to §2.4 Step 1.1 empirical assessment alongside Q2 (Identity tiers).

## Anti-Patterns to Avoid

- Resolving OMQ #3 by schemas-technology default ([`§0003`](../../charter/decision-log.md) pending).
- Collapsing OMQ #3 to runtime classification (Candidate γ rejection).
- Conflating OMQ #3 (edge generation timing) with OMQ #2 supersession (edge state derivation timing). The two are structurally distinct: OMQ #2-C resolved supersession as Cat II projection over §2.5 lifecycle events; OMQ #3 asks when the edges themselves are committed.

## Migration and Backward Compatibility

No prior committed records to migrate; forward-looking. Asymmetry in lock-in:

- **α → β:** requires removing committed `influenced_by` edges from substrate (non-trivial post-implementation; loses the structurally-committed influence relationships).
- **β → α:** requires adding edge-commit semantics for new influences; existing β-only edges (projection-derived) lose their substrate-committed form at the transition. Workable but information shape changes.

## References

- [`provenance-model.md` §Inferential Provenance + §Open Modeling Questions](../../ontology/provenance-model.md) — post-OMQ-#2-C state; OMQ #3 row annotated with cascade trigger.
- [`decision-log §0014`](../../charter/decision-log.md) — lazy pre-Gate refinement methodology.
- [`decision-log §0015`](../../charter/decision-log.md) — Q1 resolution + cascade trigger precedent.
- [`decision-log §0016`](../../charter/decision-log.md) — Q3 resolution + cascade discharge.
- [`decision-log §0019`](../../charter/decision-log.md) — Gate §2.4 prep + OMQ #2 opened.
- [`decision-log §0020`](../../charter/decision-log.md) — OMQ #2 resolution (Candidate C) + OMQ #3 cascade enactment.
- [`ontology-revision-omq2-decay-of-influence`](./ontology-revision-omq2-decay-of-influence.md) — OMQ #2 RFC (accepted).
- [`docs/rfcs/discussion/omq2-evidence.md`](../discussion/omq2-evidence.md) — OMQ #2 discussion phase evidence (Finding 6 cascade-probability analysis predicting this RFC's opening).
- [Charter §2.4](../../charter/constitutional-charter.md#24-inferential-influence-disclosure) — pending; binding text consumes OMQ #3 resolution.
- [`ontology-revision-layer-b-deep-criterion`](./ontology-revision-layer-b-deep-criterion.md) — on hold per [`§0011`](../../charter/decision-log.md); OMQ #3 resolution feeds Layer B specification alongside OMQ #2-C.

## Decision Record

This RFC is `discussion`. Resolution is two further prompts (OMQ #3 discussion synthesis → OMQ #3 resolution commit). At resolution, a decision-log entry records the candidate selected; `provenance-model.md` §Inferential Provenance is updated to encode the resolution alongside OMQ #2-C; OMQ #3 row in §Open Modeling Questions is moved to §Resolved Modeling Questions with a link to the resolution entry. Status moves `discussion` → `accepted` at that time.
