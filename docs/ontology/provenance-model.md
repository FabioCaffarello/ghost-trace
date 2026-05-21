# Provenance Model

**Status:** §Observational Provenance — Drafted post-Q3 ([`decision-log §0016`](../charter/decision-log.md)); anchored by Charter §2.3 frozen v0.4 ([`decision-log §0017`](../charter/decision-log.md)). §Inferential Provenance — Drafted post-OMQ #2 ([`decision-log §0020`](../charter/decision-log.md) — decay via §2.5 lifecycle event supersession; Candidate C) and post-OMQ #3 ([`decision-log §0021`](../charter/decision-log.md) — substrate-time generation; Candidate α); anchored by Charter §2.4 frozen v0.5 ([`decision-log §0099`](../charter/decision-log.md)). §Open Modeling Questions: rows 2 and 3 closed (moved to §Resolved Modeling Questions); row 1 (Granularity of derivation) and row 4 (Cross-domain provenance) remain open. §0020 cascade fully discharged.

> This document formalizes the structure of provenance in Ghost Trace. The Charter establishes provenance as structure, not metadata ([Charter §1](../charter/constitutional-charter.md#1-thesis)). This document specifies what that structure is.

## Two Forms of Provenance

The Charter distinguishes two forms of provenance that conventional systems do not separately track:

### Observational Provenance

Answers the question: **"From which observations does this assertion derive?"**

This is the form of provenance familiar from event-sourced systems and lineage-tracking pipelines. Every non-observation assertion in Ghost Trace declares, in its structure, the set of observations from which it was computed.

To be formalized:
- The representation of observation references in assertions.
- The transitivity rules: when assertion A is derived from assertion B, and B is derived from observations O₁ and O₂, A's observational provenance includes O₁ and O₂.
- The reconstructibility guarantee: given an assertion and the immutability of the event log, the path back to observations is reconstructible.

Per [`decision-log.md` §0015](../charter/decision-log.md) (Q1 resolution), `DeclaredSession` is a canonical Category I primary observation type; `OperationalSession` is a Category II operational construct derived from it. Provenance chains anchored to `DeclaredSession` are observational (back to substrate at commit time). Provenance chains anchored to `OperationalSession` are observational *through* the typed transformation that derives `OperationalSession` from its Category I inputs — chain shape: `assertion → OperationalSession → DeclaredSession + other Cat I primaries`. The branching is structural and visible at the type level.

Per [`decision-log.md` §0016](../charter/decision-log.md) (Q3 resolution), the Assertion entity carries one of three typed reference fields (`subject_ref_observation`, `subject_ref_construct`, `subject_ref_hypothesis`). Observational provenance chains traverse `subject_ref_observation` edges to Category I primaries; the chain shape is type-uniform at the reference level (one field type, one target Category). Chains crossing into Category II (operational constructs) traverse `subject_ref_construct` edges; chains crossing into Category III (hypotheses) traverse `subject_ref_hypothesis` edges. The graph of provenance edges is typed-by-category, mirroring the [§2.2](../charter/constitutional-charter.md#22-epistemic-separation) structural separation at the reference level.

### Inferential Provenance

Answers the question: **"Which prior assertions influenced the formation of this one?"**

This is the form of provenance that conventional systems do not maintain. The Charter requires it because without it, the system cannot distinguish between belief grounded in independent evidence and belief inherited recursively from earlier conclusions.

Per [`decision-log §0020`](../charter/decision-log.md) (OMQ #2 resolution — decay via §2.5 lifecycle event supersession; Candidate C):

- **Influence reference structure.** Every Assertion formed under influence carries one or more `influenced_by` edges referencing Category III hypotheses (or Category II constructs whose own provenance reaches Category III). The edge is committed structurally at formation time and immutable per [Charter §2.1 frozen](../charter/constitutional-charter.md#21-observational-integrity).
- **Propagation rules.** Influence accumulates transitively through chains of derivation: when Assertion A is `influenced_by` Hypothesis H₁, and H₁ was formed using observations and prior hypotheses, A's inferential provenance closure includes H₁'s antecedents to the extent they affected H₁'s formation. (Specific transitive scope per §Open Modeling Question 3 — pending; cascade-triggered per [`§0020`](../charter/decision-log.md).)
- **Decay via §2.5 lifecycle event supersession.** The current operational state of an `influenced_by` edge is a Category II projection over (a) the substrate-committed edge per §2.1 and (b) the referenced hypothesis's §2.5 lifecycle event chain. When the referenced hypothesis has a committed demotion event per [Charter §2.5 frozen v0.3 Demotion subsection](../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness), the edge state is annotated as superseded for projection-time queries. The substrate-committed edge itself is never mutated; supersession is a derived projection state per [§2.5 BC5](../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) (lifecycle events are Category I records about Category III entities).
- **Relationship to evidential independence (Invariant 2.6 — pending).** Inferential provenance, post-supersession-filtering, is the structural input to independence computation. The evidence-staleness or influence-saturation computation per Layer B's deep criterion (per [`ontology-revision-layer-b-deep-criterion`](../rfcs/draft/ontology-revision-layer-b-deep-criterion.md), follow-on RFC on hold pending §2.6; §2.4 declared-influence half discharged at frozen v0.5) consumes the superseded-state-annotated graph directly per §2.5 BC5.

Per [`decision-log.md` §0016](../charter/decision-log.md) (Q3 resolution), inferential influence references via Assertion's `subject_ref_construct` (Cat II) or `subject_ref_hypothesis` (Cat III) fields are typed-by-category; the typed-edge structure carries through the inferential provenance graph as well. Substantive §Inferential Provenance content is now codified above per OMQ #2 Candidate C; Charter-level constraints are anchored by [Charter §2.4](../charter/constitutional-charter.md#24-inferential-influence-disclosure) frozen v0.5 ([`decision-log §0099`](../charter/decision-log.md)).

Per [`decision-log §0021`](../charter/decision-log.md) (OMQ #3 resolution — substrate-time generation; Candidate α): the `influenced_by` edge is committed at formation event time as a Category I substrate record (not derived at projection time from formation events). §2.1 immutability applies to the edge directly. This commitment is consistent with the §Inferential Provenance prose above ("Every Assertion formed under influence carries one or more `influenced_by` edges...") which already encodes substrate-side commitment per OMQ #2 Candidate C codification; OMQ #3 resolution ratifies the substrate-side reading explicitly.

## The Provenance Graph

Provenance in Ghost Trace forms a directed acyclic graph. Nodes are records (observations, operational constructs, hypotheses, assertions). Edges are typed and come in at least two varieties:

- **`derived_from`** — observational-provenance edges.
- **`influenced_by`** — inferential-provenance edges.

Traversal of this graph supports:
- Reconstruction of the path from any assertion back to underlying observations (used by replay).
- Independence analysis: computing how much of an assertion's evidential support is grounded in observations vs. inherited from prior assertions.
- Forensic explanation: producing narratives over the graph for operator consumption.

The graph itself is, like everything else in the substrate, derivable from the primary event log. It may be materialized in a graph database as a projection, but the projection is not the substrate.

## Open Modeling Questions

1. **Granularity of derivation.** When a signal is computed over a window of thousands of events, does its observational provenance enumerate every contributing event, or reference the window definition? The former is precise but storage-heavy; the latter is compact but adds an indirection.
4. **Cross-domain provenance.** When Ghost Trace is applied to a domain other than its first (e.g., market integrity surveillance), do provenance edges cross domains, or is each domain a separate provenance subgraph?

## Resolved Modeling Questions

The following questions were recorded as open and have since been resolved by committee. Each entry preserves the question and links to the decision-log entry that records the resolution.

- **Decay of influence** (formerly Open Modeling Question 2). Whether inferential influence has a temporal decay, or an assertion remains "influenced by" a hypothesis indefinitely. **Resolved** by [`decision-log §0020`](../charter/decision-log.md): Candidate C — decay via §2.5 lifecycle event supersession. The §Inferential Provenance subsection above reflects the resolution; the substrate-committed `influenced_by` edge is never mutated, and supersession is a derived Category II projection over §2.5 lifecycle events per §2.5 BC5.
- **Influence at projection vs. substrate** (formerly Open Modeling Question 3, cascade-triggered by OMQ #2 resolution). Whether `influenced_by` edges are generated at substrate event time or computed at projection time from substrate event chain. **Resolved** by [`decision-log §0021`](../charter/decision-log.md): Candidate α — substrate-time generation. `influenced_by` edges are committed at formation event time as Category I substrate facts; projection rebuild reads edges from substrate deterministically. Candidate β (projection-time generation) is admissible under Q1 determinism explicit commitment but dominated by α on cascade-inherited pressure alignment (OMQ #3-1 Finding 1), §1 Thesis substrate-directness (Finding 2), Layer B specification cost (Finding 4), and §2.1 compliance clarity. The §Inferential Provenance subsection above reflects the resolution.

<!-- Invariant 2.3 frozen v0.4 per `decision-log.md` §0017. The §Observational Provenance section above carries the structural requirement language (typed-by-category edges; chain termination at Cat I primaries; transit through Cat II constructs and Cat III hypotheses); §2.3 binding text anchors it constitutionally. -->

<!-- TODO: After Invariant 2.4 is redacted, formalize the inferential-provenance section in the same way. -->

<!-- TODO: Coordinate with [`lifecycle-semantics.md`](./lifecycle-semantics.md) on how hypothesis lifecycle operations propagate through the provenance graph. -->
