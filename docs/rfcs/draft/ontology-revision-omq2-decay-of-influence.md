# RFC — Ontology Open Question (provenance-model OMQ #2): Decay of influence

- **Status:** accepted
- **Authors:** Ghost Trace committee (§2.4 pre-Gate; resolution at [`decision-log §0020`](../../charter/decision-log.md))
- **Date:** 2026-05-18 (opened); 2026-05-19 (resolved)
- **Type:** ontology-revision
- **Affects:** [`docs/ontology/provenance-model.md`](../../ontology/provenance-model.md) (§Inferential Provenance — influence propagation rules; §Open Modeling Questions OMQ #2 closed by resolution); [Charter §2.4](../../charter/constitutional-charter.md#24-inferential-influence-disclosure) (pending — binding text depends on this resolution); [Charter §2.5](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) (frozen v0.3 — Cat III hypothesis lifecycle events are influencers; their demotion semantics intersect with OMQ #2 candidate C); [`ontology-revision-layer-b-deep-criterion`](./ontology-revision-layer-b-deep-criterion.md) (on hold per [`§0011`](../../charter/decision-log.md); OMQ #2 resolution feeds Layer B's deep criterion specification per [`§0017`](../../charter/decision-log.md) Methodological Observation 2)

> RFCs in Ghost Trace are proposals subject to constitutional review. The [Constitutional Charter](../../charter/constitutional-charter.md) is the authoritative reference. RFCs that conflict with the Charter must either be rejected or accompanied by a corresponding charter amendment (see [`../../charter/amendments.md`](../../charter/amendments.md)).

## Summary

OMQ #2, recorded in [`provenance-model.md` §Open Modeling Questions](../../ontology/provenance-model.md): should inferential influence have a temporal decay, or does an assertion remain "influenced by" a hypothesis indefinitely? Downstream §2.4 redaction depends on resolution because §2.4 binding text governs the structural form of `influenced_by` edges; whether decay is part of the structural commitment or a runtime concern shapes the binding text directly.

This RFC opens structured discussion with three candidate resolutions (A permanent, B operational decay parameter, C decay via lifecycle event supersession) and one explicitly rejected alternative (D runtime classification). The RFC does not pick a candidate.

## Motivation

**Why now.** §2.4 (Inferential Influence Disclosure) is next per [`§0008`](../../charter/decision-log.md) redaction order; [`§0019`](../../charter/decision-log.md) opens this RFC as §2.4 pre-Gate dependency. §2.4 binding text codifies the structural form of `influenced_by` edges. The temporal behavior of those edges — permanently recorded or decaying over time — is operative for the binding text:

- Under permanence, §2.4 binding text codifies edges as immutable facts; independence computation traverses without temporal filtering.
- Under operational decay, §2.4 binding text codifies edges as carrying a decay parameter; projection-time logic applies the decay.
- Under decay via lifecycle event supersession, §2.4 binding text references §2.5 Cat III demotion semantics to bound when influence ceases to be operative.

Without OMQ #2 resolution, §2.4 binding text either silently picks decay-vs-permanent or produces vague language that defers the decision to implementation. Both failure modes are forbidden by [`ontology-keeper`](../../../.claude/skills/ontology/ontology-keeper/SKILL.md) discipline.

**The cost of not resolving.** Pattern established by Q1 [`§0015`](../../charter/decision-log.md) + Q3 [`§0016`](../../charter/decision-log.md) + Q2/Q4 [`§0010`](../../charter/decision-log.md)/[`§0011`](../../charter/decision-log.md): open Ontology questions are committee-resolved, not infrastructure-resolved. The OMQ #2 pre-Gate is the expected cascade per [`§0014`](../../charter/decision-log.md) methodology — opened minimally, with other open questions (OMQ #3, Q2 Identity tiers) deferred to §2.4 Step 1.1 empirical assessment.

## Constitutional Review

The Q1–Q6 impact analysis prescribed by [`rfc-author` §1](../../../.claude/skills/workflow/rfc-author/SKILL.md).

### Q1 — Which Charter invariants does this RFC touch?

- **§2.4 Inferential Influence Disclosure** (pending — this resolution's purpose): touched directly. §2.4 binding text encodes `influenced_by` edge structure; this RFC's resolution determines whether decay is part of the structural commitment.
- **§2.5 Hypothesis Lifecycle Explicitness** (frozen v0.3): touched indirectly. Cat III hypotheses are the influencers; their lifecycle events (formation through dissolution per §2.5 abstract-type operations) bound when influence is operative. Candidate C explicitly reads §2.5 demotion semantics into OMQ #2 resolution.
- **Layer B follow-on RFC** ([`ontology-revision-layer-b-deep-criterion`](./ontology-revision-layer-b-deep-criterion.md), on hold per [`§0011`](../../charter/decision-log.md)): OMQ #2 resolution feeds Layer B's deep criterion specification per [`§0017`](../../charter/decision-log.md) Methodological Observation 2 (forward-reference contract accommodates pending Ontology RFCs). The deep criterion's evidential-independence test under §2.6 (pending) depends on when `influenced_by` edges are operative — which is what OMQ #2 resolves.
- **§2.3 Provenance Integrity** (frozen v0.4): touched at the boundary level only. §2.3 BC1 codified `subject_ref_construct` / `subject_ref_hypothesis` edges as observational-provenance transit; §2.4's `influenced_by` is the inferential-semantic reading of the same edge structure. OMQ #2 governs `influenced_by` temporal behavior; §2.3 is unaffected.

### Q2 — New glossary terms?

Depends on resolution.

- Candidate A (permanent): no new terms. The bare `influenced_by` edge is permanent by construction.
- Candidate B (operational decay parameter): likely candidates — `influence decay`, `decay parameter`, `decay function`, `decay state`. Terms added with resolution commit if Candidate B wins.
- Candidate C (decay via lifecycle event supersession): likely candidates — `influence supersession`, `superseding event` (or refinement of existing `demotion` glossary entry to encompass influence-supersession semantics). Terms added with resolution commit if Candidate C wins.

No glossary modifications in this discussion phase per Q1/Q2/Q3/Q4 precedent.

### Q3 — Resolves an Ontology open question?

**Yes.** This RFC IS the resolution of provenance-model.md OMQ #2 (Decay of influence). Resolution closes the open question; the §Inferential Provenance section is updated to encode the chosen candidate's structural commitment.

### Q4 — Charter amendment?

Not directly. The resolution constrains §2.4 binding text, but §2.4 is pending — not amendable yet. §2.4 redaction is the subsequent step; OMQ #2 resolution is its precondition.

### Q5 — New invariant?

No. The RFC does not introduce a new Charter invariant. It resolves an Ontology-level modeling question that §2.4 will then codify structurally at the Charter level.

### Q6 — Ceremony or constitutional?

Constitutional, not ceremony. Permanence vs decay has materially different schemas, query patterns, projection rebuild semantics, and Layer B-input consequences:

- Schema differs (permanent edges vs edges-with-decay-parameter vs edges-plus-supersession-references).
- Query patterns differ ("what currently influences X?" computes differently under each candidate).
- Projection rebuild semantics differ (deterministic vs decay-time-dependent vs lifecycle-event-dependent).
- Layer B deep criterion specification consumes OMQ #2 outcome as input; behavioral consequences cascade through §2.4 + §2.6 + Layer B.

## Proposal

Three candidate resolutions plus one rejection.

### Candidate A — Permanent influence

**Structural claim.** `influenced_by` edges are permanent structural facts. Once an assertion's formation is influenced by a hypothesis, the edge exists for all time in the substrate.

**Schema implication.** Edge type is simple: `(source_assertion, influenced_hypothesis, [optional context fields])`. No decay parameter. No supersession reference. Edge multiplicity is structural — multiple `influenced_by` edges may exist per source assertion (one per influencing hypothesis).

**Layer B interaction.** Layer B's deep criterion specification operates over the full `influenced_by` graph without temporal filtering. Evidential-independence computation per §2.6 (pending) reads all edges as currently-operative.

**Query pattern.** "What currently influences this assertion?" reduces to "what `influenced_by` edges does this assertion have in the substrate?" — a single substrate read. Projection rebuild is fully deterministic from substrate; rebuild produces identical edges.

**§2.2 compliance shape.** Clean. Cat I/II/III separation is respected at the edge level — `influenced_by` references a Cat III hypothesis from a Cat II/III assertion. No category crossing introduced by the edge structure itself.

**Pros.**
- Maximal simplicity.
- Projection rebuild is fully deterministic from substrate.
- Independence computation is structurally clean — no decay state to track.
- Compatible with §2.1 immutability without conditional commitments.
- Layer B specification can proceed with a complete, time-stable graph.

**Cons.**
- "Influence" persists even after the influencing hypothesis is demoted per §2.5 (Cat III demotion withdraws operational use as enrichment context, but the lifecycle event is recorded in substrate per §2.5 frozen v0.3). The structural commitment is that influence-at-formation-time is structurally permanent; the substrate does not encode "influence has ceased."
- Operational systems may want to filter out influence from demoted hypotheses for current-decision purposes. Under Candidate A, this is a projection-time concern, not a structural commitment.
- May misrepresent the epistemic intent of "influence" — colloquial usage often implies "currently shaping belief", which permanence does not capture.

### Candidate B — Permanent edges with operational decay parameter

**Structural claim.** `influenced_by` edges are permanent in the substrate (preserving §2.1 immutability), but each edge carries a decay parameter (or decay function reference) that operational code applies at projection/query time.

**Schema implication.** Edge type includes decay parameter field(s): `(source_assertion, influenced_hypothesis, decay_function_ref, decay_parameters, [optional context fields])`. The decay function is registered separately (likely in a Cat II operational construct namespace per §2.2 + Q1 determinism); the parameters are edge-local.

**Layer B interaction.** Layer B's deep criterion specification operates over edges weighted by decay state. Evidential-independence per §2.6 may filter or weight edges by elapsed time, decay-function output, or other operational context. Layer B's specification absorbs the decay-weighted graph as input.

**Query pattern.** "What currently influences this assertion?" computes by reading edges from substrate, then applying decay function to each at query time. Projection rebuild produces identical *edges* (deterministic from substrate); decay-weighted state is projection-layer logic. Independence computation is decay-state-dependent.

**§2.2 compliance shape.** Clean, with one conditional commitment: the decay function itself is a Cat II operational construct (deterministic from observation timestamps + parameters); its registration follows §2.2 Cat II rules. The decay-weighted edge state is a Cat III-adjacent inference (it's an operational reading of decay, not a hypothesis per se).

**Pros.**
- Captures the operational intent of "decay" without violating §2.1 substrate immutability.
- Decay function is versioned and registered (Cat II semantics); changes to decay logic produce new construct versions, not retroactive substrate edits.
- Independence computation under §2.6 has a principled handle on temporal weighting.
- Compatible with Layer B's deep criterion specification — provides the weighted graph Layer B can consume.

**Cons.**
- Schema complexity increases (decay parameter fields per edge).
- Independence computation becomes decay-state-dependent — same substrate, different query outputs at different times. Falsifiability of §2.6 evidential-independence claims must accommodate decay-state at query time.
- Decay function specification itself becomes a deferred concern (what form? what parameters? scope of application?) — may surface as cascade RFC depending on resolution.
- "Operational decay" is a runtime concern collapsing into a structural commitment — the boundary between operational and structural is blurred.

### Candidate C — Decay via lifecycle event supersession

**Structural claim.** `influenced_by` edges are augmented over time by Cat III hypothesis lifecycle events (demotion per §2.5 frozen v0.3). When an influencing hypothesis is demoted, the demotion event supersedes the influence relationship structurally — the edge persists per §2.1 immutability but is structurally annotated as superseded by reference to the demotion event.

**Schema implication.** Edge type: `(source_assertion, influenced_hypothesis, [optional context fields])`. Plus: edge supersession is tracked via lifecycle event references — when a referenced hypothesis is demoted, the demotion event (per §2.5 BC5 as Cat I record) is consulted to determine current edge state. The edge's "current state" is a projection over substrate + lifecycle event history of the referenced hypothesis.

**Layer B interaction.** Layer B's deep criterion specification operates over edges filtered by current lifecycle state of the referenced hypothesis. Demoted hypotheses' edges are structurally annotated as superseded; Layer B reads the annotation. The deep criterion may treat superseded edges differently from active edges per §2.6 (pending) specification.

**Query pattern.** "What currently influences this assertion?" computes by reading edges from substrate, then traversing each referenced hypothesis's lifecycle event chain (per §2.5 BC5) to determine current state. Projection rebuild consults the full lifecycle event log; rebuild is deterministic but multi-step.

**§2.2 compliance shape.** Clean. The lifecycle events are Cat I records per §2.5 BC5 (frozen v0.3); the supersession is structurally encoded at the lifecycle event level, not at the edge level. Cat I/II/III separation respected.

**Pros.**
- Decay is structurally tied to §2.5 lifecycle events — the system's existing mechanism for hypothesis state changes.
- No new schemas concept (no decay parameter); reuses lifecycle event infrastructure.
- Decay timing is principled — tied to demotion events that have their own committee-scrutinized semantics per §2.5 frozen v0.3.
- Compatible with Layer B's deep criterion — provides a clear "edge is currently superseded" annotation.

**Cons.**
- Coupling between §2.4 (this RFC's downstream consumer) and §2.5 (frozen) is high. §2.4 binding text would reference §2.5 demotion semantics directly.
- Query pattern is multi-step (substrate read + lifecycle traversal); projection rebuild semantics depend on lifecycle event log completeness.
- The "decay" framing is conflated with "demotion" — they may not be the same epistemic concept. Demotion withdraws operational use; influence-decay may have other triggers (e.g., refuted-by-new-observation, not equivalent to operational-demotion).
- Influences from hypotheses that have not yet been demoted (but should be) remain operative; the structural commitment is to lifecycle-event-driven decay, not to "any reasonable decay trigger."

## Alternatives Considered

### Candidate D — Decay via runtime classification (REJECTED)

**Structural claim.** Influence is determined at query time by an inference process classifying edges as "active" or "decayed" based on operational signals (e.g., time elapsed, evidence accumulated against the influencing hypothesis, current confidence).

**Why rejected.** Collapses OMQ #2 to runtime — the same failure mode `ontology-keeper` exists to prevent (Q1 Candidate C rejection precedent per [`§0015`](../../charter/decision-log.md); Q3 Candidate C rejection precedent per [`§0016`](../../charter/decision-log.md)). Runtime classification places the structural commitment in inference code, not in schemas or types or permitted operations — failing §4 criterion 1 (structurally enforceable). Independence computation becomes inference-driven (recursive belief-inflation risk in the test for evidential independence itself). Violates the project's core constitutional discipline.

## Open Questions

What does THIS RFC defer?

- **provenance-model.md OMQ #3 (Influence at projection vs substrate).** Touches OMQ #2 by way of:
  - Under Candidate A (permanent), projection rebuild generates the same `influenced_by` edges deterministically — low coupling to OMQ #3.
  - Under Candidate B (operational decay parameter), projection-time decay computation creates ambiguity about when edges are generated — high coupling to OMQ #3.
  - Under Candidate C (supersession), projection consults lifecycle event log — high coupling to OMQ #3.

  Status: deferred to §2.4 Step 1.1 empirical assessment per [`§0014`](../../charter/decision-log.md) lazy methodology. If blocking under OMQ #2 outcome, opens as cascade per §0015→§0016 cascade pattern.

- **Specific parameters under Candidate B.** Decay function form, parameter scope, query-layer weighting rules — deferred to post-OMQ-#2 specification work (likely a follow-on Cat II operational definition RFC).

- **Layer B deep criterion specification.** Layer B follow-on RFC ([`ontology-revision-layer-b-deep-criterion`](./ontology-revision-layer-b-deep-criterion.md)) remains the locus per [`§0011`](../../charter/decision-log.md). OMQ #2 resolution provides input but does not complete Layer B's specification. Layer B's eventual specification reconciles OMQ #2 outcome + §2.4 binding text + §2.6 (pending) — per [`§0017`](../../charter/decision-log.md) Methodological Observation 2.

- **Q2 (Identity tiers — entity-model.md Open Modeling Question 1).** Remains forward-referenced per [`§0017`](../../charter/decision-log.md) Resolution 4 marker form. §2.4 Step 1.1 may surface Q2 as blocking depending on §2.4 binding text dependence on identity-tier semantics for influence-source references. Not in scope for OMQ #2.

## Anti-Patterns to Avoid

- Resolving OMQ #2 by code (schemas technology pick or runtime classification without RFC). The discipline mirrors `ontology-keeper` Q1/Q2/Q3/Q4 framing.
- Treating OMQ #2 as a "tuning parameter" rather than a structural commitment. The candidate choice is constitutional — independence computation semantics differ materially.
- Silent revision in a future `provenance-model.md` redaction that picks one candidate without acknowledging OMQ #2 resolution.
- Conflating Candidate B's operational decay with Candidate C's lifecycle-event supersession. Both are "decay" framings but encode the concept structurally differently — Candidate B at the edge level, Candidate C at the lifecycle level.

## Migration and Backward Compatibility

No prior committed records to migrate; forward-looking. Asymmetry in lock-in:

- **Candidate A → Candidate B:** structural addition (add decay parameter field to edges; existing edges grandfathered as "decay = none"). Workable post-implementation.
- **Candidate A → Candidate C:** requires lifecycle event semantics to encompass influence supersession. Cat III demotion events per §2.5 already record demotion structurally; the addition is the supersession-of-influence reading at query time. Moderate refactor.
- **Candidate B / C → Candidate A:** structural simplification. Loses decay/supersession information captured pre-simplification — workable but information loss.
- **Candidate B → Candidate C:** moderate refactor; decay parameter fields removed, lifecycle event traversal added.
- **Candidate C → Candidate B:** moderate refactor; lifecycle traversal removed, decay parameters added.

## References

- [`provenance-model.md` §Inferential Provenance](../../ontology/provenance-model.md) — `influenced_by` edge definition.
- [`provenance-model.md` §Open Modeling Question 2](../../ontology/provenance-model.md) — OMQ #2 statement.
- [`entity-model.md`](../../ontology/entity-model.md) — Category III hypothesis structure (post-Q2 + post-Q4); Q2 (Identity tiers, post-Q1-renumbering) carry-forward context.
- [Charter §2.4](../../charter/constitutional-charter.md#24-inferential-influence-disclosure) — pending; binding text consumes OMQ #2 resolution.
- [Charter §2.5](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) — frozen v0.3; demotion semantics intersect Candidate C.
- [`ontology-revision-layer-b-deep-criterion`](./ontology-revision-layer-b-deep-criterion.md) — on hold per [`§0011`](../../charter/decision-log.md); OMQ #2 resolution feeds Layer B specification.
- [`decision-log §0011`](../../charter/decision-log.md) — Q4 resolution + Layer B follow-on RFC opened on hold.
- [`decision-log §0014`](../../charter/decision-log.md) — lazy pre-Gate refinement methodology.
- [`decision-log §0015`](../../charter/decision-log.md) — Q1 resolution + cascade trigger pattern.
- [`decision-log §0016`](../../charter/decision-log.md) — Q3 resolution + cascade discharge.
- [`decision-log §0017`](../../charter/decision-log.md) — Gate §2.3 closure; Methodological Observation 2 (forward-reference contract extension to Ontology-level pending content).
- [`decision-log §0019`](../../charter/decision-log.md) — §2.4 redaction plan opening this RFC.

## Decision Record

Resolved at [`decision-log §0020`](../../charter/decision-log.md): **Candidate C — decay via §2.5 lifecycle event supersession**. The committee adopted Candidate C with three committee extensions:

1. **B-substrate admissibility acknowledged but dominated.** B-projection collapsed to rejected Candidate D shape per discussion Finding 1; B-substrate registered as admissible-but-dominated (methodologically distinct from D's structural-incoherence rejection). Pattern preserves the "rejected dominated" vs "rejected incoherent" distinction for future RFCs.
2. **No supersession-encoding glossary entry.** Existing canonical terms (`demotion`, `hypothesis lifecycle event`) suffice; supersession is read from §2.5-committed lifecycle events, not §2.4-original vocabulary. Aligns with Q3-2 (structural mechanism, no new entries); contrasts Q1-2 (entity types added).
3. **OMQ #3 cascade-triggered.** Per discussion Finding 6 (high cascade probability under Candidate C) + [`§0014`](../../charter/decision-log.md) lazy methodology + [`§0015`](../../charter/decision-log.md)→[`§0016`](../../charter/decision-log.md) cascade precedent, [`ontology-revision-omq3-influence-projection-vs-substrate`](./ontology-revision-omq3-influence-projection-vs-substrate.md) opened at `discussion` status in the §0020 commit. OMQ #3 becomes pre-Gate dependency for §2.4 redaction Step 1.1 alongside Q2 (Identity tiers) forward-reference and Layer B activation reconciliation.

Provenance-model.md §Inferential Provenance updated to encode Candidate C per §0020 Phase 2; OMQ #2 row moved from §Open Modeling Questions to new §Resolved Modeling Questions section; row 3 (OMQ #3) annotated with cascade trigger; Status banner refreshed. CLAUDE.md §4 status table narrative updated. No glossary changes. Layer B follow-on RFC remains on hold per [`§0011`](../../charter/decision-log.md); OMQ #2 partially-discharges its Q5 dependency for the "decaying?" axis (Q5 "transitive?" axis remains open per discussion Finding 7).
