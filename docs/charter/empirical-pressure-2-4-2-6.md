# Empirical-Pressure Assessment — §2.4 / §2.6

> **Status:** assessment (non-binding). This document inventories what the
> implementation surface as of `2026-05-21` has structurally committed to,
> and catalogs concrete questions the pending [Charter §2.4 (Inferential
> Influence Disclosure)](./constitutional-charter.md#24-inferential-influence-disclosure)
> and [§2.6 (Evidential Independence Integrity)](./constitutional-charter.md#26-evidential-independence-integrity)
> binding text would have to answer. **This document does NOT draft binding
> text and does NOT pick answers to the questions it catalogs.** Per the
> v0.4.2 amendment ([`decision-log §0022`](./decision-log.md)), §2.4 and §2.6
> are in `pending — empirical pressure phase`; this assessment is the
> phase's deliverable. Whether the empirical pressure is sufficient to
> trigger §2.4 → §2.6 redaction-resumption is a committee call, not a
> conclusion of this document.

## 1. Status snapshot

- **Charter status:** v0.4.2. Five object-level invariants frozen (§2.1, §2.2,
  §2.3 v0.4, §2.5 v0.3, §4 v0.2). §2.4 + §2.6 pending — empirical pressure
  phase. §3 Non-Goals pending.
- **Ontology open questions:** OMQ #2 + OMQ #3 resolved
  ([`§0020`](./decision-log.md), [`§0021`](./decision-log.md)). Q1 + Q3 +
  Q2 resolved
  ([`§0015`](./decision-log.md), [`§0016`](./decision-log.md), [`§0023`](./decision-log.md)).
  Remaining: provenance-model OMQ #1 (Granularity of derivation), OMQ #4
  (Cross-domain provenance), `ontology.md` Q5 "transitive?" half, Q4
  (Cross-subtype operations).
- **Implementation surface:** 30 operational binaries; 25 substrate-write
  CLIs; 4 Cat III subtypes × 6 lifecycle operations = 24 lifecycle event
  types; projection layer covering all 4 subtypes; ~50 PRs merged since
  the implementation pivot ([`§0022`](./decision-log.md)).

## 2. Implementation surface load-bearing for §2.4 / §2.6 redaction

The following are committed structural facts. A §2.4 / §2.6 redaction must
either inherit them or amend them under [`amendments.md`](./amendments.md)
discipline.

### 2.1 Placeholder `confidence` field in every Cat III formation event

Each of the four Cat III formation protos
([BehavioralClusterFormation](../../schemas/events/v1/behavioral_cluster_formation.proto),
[AutomationGroupFormation](../../schemas/events/v1/automation_group_formation.proto),
[CampaignHypothesisFormation](../../schemas/events/v1/campaign_hypothesis_formation.proto),
[CoordinationRingFormation](../../schemas/events/v1/coordination_ring_formation.proto))
carries `float confidence = N` documented as `PLACEHOLDER pending Charter §2.6
redaction`. Implementation produces it deterministically via
`confidenceFromClusterSize`.

- **Load-bearing for §2.6:** the slot exists; the redaction decides the
  paired dimension's shape (companion field, structured payload, separate
  message).
- **Schemas-evolution implication:** adding an `evidential_independence`
  paired dimension is an additive proto change but triggers
  [canonical-serialization-contract `schemas-evolution event`](../architecture/canonical-serialization-contract.md)
  + canonical-corpus regeneration per [`§0031`](./decision-log.md). Eight
  golden corpus entries (`*-formation-minimal` + `*-formation-typical` ×
  four subtypes) would need explicit regeneration.

### 2.2 Promotion events carry Layer A `cadence_seconds`

Each of the four Cat III promotion protos carries `int64 cadence_seconds`
per the [§0011 Layer A](./decision-log.md) commitment. Demotion operators
read it and report `cadence_satisfied` + `cadence_elapsed_seconds` in their
JSON output. The cadence-gate is operationally surfaced as candidacy
(non-blocking; the substrate accepts demotions regardless).

- **Load-bearing for Layer B activation:** §0011 Layer B is the join point —
  §2.4 + §2.6 redaction makes Layer B's deep criterion specifiable. The
  demote-* operators currently report cadence-satisfaction state only;
  Layer B activation propagates new state into that JSON shape.

### 2.3 No `influenced_by` field exists in any proto

The Ontology layer has codified `influenced_by` edges as Cat I substrate
records (per [§0020](./decision-log.md) Candidate C + [§0021](./decision-log.md)
Candidate α — substrate-time generation). No wire shape has been chosen.
No event type in the substrate currently carries an inferential-influence
declaration. The system can produce promotion events (substrate record
of "this hypothesis is now in use as enrichment context"), but no
downstream consumer of enrichment exists yet — nothing is currently
formed *under influence*.

- **Load-bearing for §2.4:** the wire-shape decision is the redaction's
  first concrete commitment. The current substrate is silent on the
  question, not committed to a wrong answer.

### 2.4 Merge produces separately-committed formations (§0049 Option B)

For each of the four subtypes, merge commits the merged hypothesis as a
*separate* formation event (the produced hypothesis's identity is its own
formation hash). The merge event references three formation hashes
(antecedent A + antecedent B + produced). The produced formation's
`confidence` field is computed by the inference process producing the
merge, not by the merge event itself.

- **Load-bearing for §2.6:** when §2.6 commits to an
  `evidential_independence` dimension on formation events, the merge
  produced-formation's independence value must come from somewhere. The
  redaction must decide between (a) inherited (e.g. min of antecedents'),
  (b) computed afresh from the produced formation's `source_event_hashes`,
  or (c) operator-supplied at merge time. None of these is currently
  precluded by the substrate.

### 2.5 Hypothesis identity = formation event content-hash (§0045 invariant)

All four Cat III subtypes preserve the §0045+§0056+§0063+§0070 invariant:
the formation event's BLAKE3-256 content-hash IS the hypothesis's stable
identifier. Lifecycle events reference hypotheses by their formation
content-hash. Promotion events reference formation hashes; demotion events
reference promotion hashes; etc.

- **Load-bearing for §2.4:** an `influenced_by` edge must reference *some*
  content-hash. The redaction must decide whether the reference points at
  the formation event (the hypothesis's identity) or the promotion event
  (the substrate record that the hypothesis is in use as enrichment).
  [`§0020`](./decision-log.md) supersession-via-§2.5 supports either: a
  formation-hash reference, post-supersession-filtered against the
  formation's §2.5 lifecycle chain, recovers the same operational state
  as a promotion-hash reference.

## 3. Concrete questions §2.4 binding text must answer

The questions below have been surfaced by implementation; the Charter
working-definition does not already answer them. Numbering is not
prioritized.

### Q-§2.4-1. Wire shape of the influence declaration

**Question.** Where does the influence declaration live on the substrate?
Three substrate-compatible shapes are visible:

- (a) Field on the influenced assertion's formation event (e.g. an
  `influenced_by_formation_hashes` repeated bytes field on
  `CampaignHypothesisFormation` when a campaign is inferred *while a
  BehavioralCluster is promoted*).
- (b) Paired sibling event committed at the same `event_time` as the
  influenced formation (e.g. a separate `InferentialInfluence` message
  type referenced from the formation's content-hash).
- (c) Some hybrid — e.g. structural slot on formation events for "no
  influence" (empty list) and a separate event type for "yes, multiple
  influences" payloads.

**Why surfaced:** §0021 substrate-time generation requires the edges to
be Cat I substrate facts; the redaction chooses the wire shape they
inhabit.

**Cascade implications:** affects every Cat III formation proto's wire
shape; cascades into corpus regeneration.

### Q-§2.4-2. Reference target — formation hash or promotion hash?

**Question.** An `influenced_by` edge targets *which* content-hash? Two
candidates:

- (α) The influencing hypothesis's formation hash. Supersession-via-§2.5
  is read from the formation's lifecycle chain at projection time.
- (β) The influencing hypothesis's promotion event hash. The edge
  targets the substrate record "hypothesis is in use as enrichment".

**Why surfaced:** §0020 Candidate C resolution is silent on the targeting
question. Both candidates are §2.5-compatible.

**Cascade implications:** affects projection-time read semantics. Under
(α), `latest_promotion`/`latest_demotion` projection reads are sufficient
to compute supersession state. Under (β), the projection's `LatestPromotion`
pointer participates directly in the influence graph.

### Q-§2.4-3. Triggering condition for "formed under influence"

**Question.** When does an assertion "carry" an `influenced_by` edge? The
Charter §2.4 working-definition says "every assertion subsequently formed
under that influence" — but the operational trigger is unspecified:

- (a) Any assertion formed while ≥ 1 hypothesis is in the
  promoted-but-not-yet-demoted state for the actor/event set involved.
- (b) Only assertions whose inference process *actually consumed* the
  promoted hypothesis as input (a tighter, behaviour-conditional shape).
- (c) Some intermediate: assertions formed within the promoted
  hypothesis's *scope* (where scope itself needs definition).

**Why surfaced:** implementation has surfaced one promoted hypothesis can
be in force for the same actor set as a separately-promoted hypothesis;
the assertion-formation process operates against the substrate at a point
in time and must decide what counts as "under influence".

**Cascade implications:** (a) is operationally cheapest but most permissive
(every promoted hypothesis is structurally connected to every subsequent
formation in its potential scope); (b) requires the inference process to
make the influence claim explicitly (and the substrate to accept the
operator's claim without itself verifying); (c) needs scope semantics
(per-actor? per-event-time-window? per-descriptor?).

### Q-§2.4-4. Acyclicity guarantee

**Question.** Is the influence graph required to be acyclic by §2.4
binding text? Implementation has surfaced concrete graph-cycle risk via
merge:

- A BehavioralCluster formation H₁ is promoted.
- A new BehavioralCluster H₂ is formed under H₁'s influence:
  `H₂.influenced_by = [H₁]`.
- H₁ and H₂ are later merged → a third formation H₃ committed.
- If H₃'s `influenced_by` is *inherited from antecedents* (per
  Q-§2.6-3 below), it could contain H₂ → H₂'s `influenced_by` could
  contain H₁ → H₁'s content-hash is independent → no cycle yet.
- BUT if H₃ is then re-promoted and H₁ is re-formed (different content)
  while H₃ is in force, a cycle becomes possible.

[`provenance-model.md` §The Provenance Graph](../ontology/provenance-model.md)
asserts "Provenance in Ghost Trace forms a directed acyclic graph" — but
this is descriptive prose, not a constitutional commitment.

**Why surfaced:** §2.5 lifecycle operations (merge, split, re-promotion)
create graph structure that §2.4 binding text either constrains or
accepts at face.

**Cascade implications:** if §2.4 requires DAG-ness, the substrate-write
side needs cycle-detection at commit time (a structural check that may
need a §2.1-compatible projection lookup). If §2.4 accepts cycles, the
projection-side queries need to terminate explicitly.

### Q-§2.4-5. Q5 "transitive?" half — still open

**Question.** Per [`§0021`](./decision-log.md) Carry-forwards, [`ontology.md` Q5](../ontology/ontology.md)
"transitive?" axis remains open. Does an `influenced_by` edge accumulate
transitively (A `influenced_by` H₁; H₁ was itself formed using prior
hypotheses → does A's closure include H₁'s antecedents)?

**Why surfaced:** [`provenance-model.md` §Inferential Provenance](../ontology/provenance-model.md)
hedges this: "to the extent they affected H₁'s formation". The "extent"
is unspecified. Implementation has not yet had to decide because no
`influenced_by` edges exist.

**Cascade implications:** Q5-transitive could open as a pre-Gate
cascade RFC per [`§0014`](./decision-log.md) lazy methodology if §2.4
binding text depends on its resolution. The §2.4 Step 1.1 anchor inventory
would assess.

### Q-§2.4-6. §2.3 BC1 inheritance — the §2.4 side

**Question.** [§2.3 BC1](./constitutional-charter.md#23-provenance-integrity) codifies
`subject_ref_construct` / `subject_ref_hypothesis` edges as observational-provenance
transit. §2.4 binding text inherits these edges' inferential semantics. The
inheritance shape itself is structurally clean ([`§0019`](./decision-log.md)
Methodological Observation 2); but the codification text is the
redaction's job.

**Why surfaced:** [`§0017`](./decision-log.md) Resolution 4 marker form.

## 4. Concrete questions §2.6 binding text must answer

### Q-§2.6-1. Shape of the paired `evidential_independence` dimension

**Question.** §2.6 working-definition says "two distinct dimensions:
magnitude of confidence and degree of evidential independence". The
`confidence` field is committed as `float` per all four subtypes. The
paired dimension shape:

- (a) Symmetric float: `float evidential_independence = M+1` alongside
  `float confidence = M`.
- (b) Structured: a sub-message capturing the independence computation's
  inputs (e.g. `IndependenceWitness { repeated bytes contributing_observation_hashes; int64 observation_count; bytes inference_method_hash; }`).
- (c) Hash-typed reference: `bytes evidential_independence_witness = M+1`
  pointing at a separately-committed witness event.

**Why surfaced:** the `confidence` field's wire choice was made
([`§0045`](./decision-log.md) onward) as a placeholder; the paired
dimension's wire choice would canonically join it.

**Cascade implications:** affects all four Cat III formation protos +
corpus regeneration + canonical-serialization-contract schemas-evolution.

### Q-§2.6-2. Origin of the independence value at first commit

**Question.** Where does `evidential_independence` come from when a
formation event is first committed?

- (a) Computed deterministically by the formation pattern (like the
  current `confidence` from cluster size). Implementation has not yet
  exposed an interface where the formation pattern emits an independence
  value.
- (b) Computed by the substrate from the formation event's
  `source_event_hashes` set (e.g. some measure of overlap with prior
  formations' source sets).
- (c) Operator-supplied at formation time (escape-hatch for operator
  judgment).

**Why surfaced:** the current `confidenceFromClusterSize` interface is
the pattern's output; whether §2.6 keeps that interface (extending it
with a paired output) or moves the computation elsewhere is a redaction
question.

### Q-§2.6-3. Independence under merge

**Question.** When a merge produces a separately-committed formation,
how is its `evidential_independence` computed?

- (a) **Inherited (min):** `min(antecedent_A.indep, antecedent_B.indep)` —
  defends conservatively against recursive inflation.
- (b) **Inherited (other rule):** e.g. average, weighted by source set
  size, etc.
- (c) **Computed afresh:** independence is recomputed on the union of
  antecedents' `source_event_hashes` — treats merge as a re-derivation
  rather than a combination.
- (d) **Hybrid:** §2.6 codifies a rule; the formation pattern may
  override under named conditions.

**Why surfaced:** §0049 Option B (produced formation as separately-
committed event) means the merge's produced formation needs a concrete
content for every field at commit time. Implementation currently uses
`confidenceFromClusterSize(len(antecedents))` for the placeholder — a
clearly-not-meant-for-production choice that §2.6 redaction would
displace.

### Q-§2.6-4. Independence under split

**Question.** When a split produces multiple successor formations, how is
each successor's `evidential_independence` computed?

The same three branches as Q-§2.6-3, applied per successor. Additionally:
the partition discipline (each successor carries a *subset* of the
antecedent's `source_event_hashes`) means the successor's independence is
*derivable* from the partition — but the redaction must commit to which
derivation rule.

**Why surfaced:** §0050 split shape produces N separately-committed
successor formations; same as Q-§2.6-3.

### Q-§2.6-5. Backward-projection of independence onto already-committed formations

**Question.** When §2.6 freezes, the 8 already-committed corpus
formations (plus any production-substrate formations) have no
`evidential_independence` field. What is their value under §2.6's binding
text?

- (a) Absent (proto3 default 0.0) — §2.6 binding text accepts the
  default value semantic.
- (b) Schemas-evolution event regenerates the corpus with explicit values;
  production substrates require operator-driven backfill.
- (c) A separate "independence-backfill" lifecycle event type is added
  per §2.5 BC5 lifecycle-event-as-Cat-I-record pattern — backward
  compatibility via append-only.

**Why surfaced:** corpus regeneration is non-trivial per
[`§0031`](./decision-log.md) golden-file-gate discipline. The schemas-
evolution path has been exercised ([`§0028`](./decision-log.md)); but
the *meaning* of an absent value vs an explicit zero is unspecified.

## 5. Cross-§2.4/§2.6 questions

### Q-X1. Layer B activation

§0011 Layer B is the join point. The committee redaction sequence is:

1. §2.4 binding text freezes (codifies the OMQ #2/#3 substrate-side ground;
   activates Layer B's "declared influence" half).
2. §2.6 binding text freezes (codifies the independence dimension;
   activates Layer B's "evidential independence" half).
3. Layer B follow-on RFC ([`ontology-revision-layer-b-deep-criterion`](../rfcs/draft/ontology-revision-layer-b-deep-criterion.md))
   resumes: defines the deep criterion's structural form, consumes §2.4 +
   §2.6 + §2.5 BC5 (lifecycle-event-as-Cat-I-record), specifies what
   `demote-*` operators report as Layer B state.

**Implementation surface that's pre-committed:** the demote-* operators
already report `cadence_satisfied` (Layer A). The JSON shape would need
extension for Layer B state. The structural slot exists.

### Q-X2. Independence-confidence relationship

**Question.** §2.6 says the two dimensions are "distinct" — but the
redaction must decide whether they are *independent* (can take any
combination of values) or *constrained* (e.g. independence ≤ confidence).

**Why surfaced:** implementation's `confidenceFromClusterSize` may not
admit a corresponding `independenceFromX` that is mathematically
unconstrained relative to confidence — depending on the formation
pattern's information theory.

### Q-X3. Cross-subtype operations (Q4) — orthogonal but adjacent

[`entity-model.md` §Cross-subtype operations](../ontology/entity-model.md)
remains deferred. §2.4 / §2.6 binding text is *not* gated on Q4
resolution: cross-subtype operations affect the produced-record type
semantics ([§2.5 BC4](./constitutional-charter.md#25-hypothesis-lifecycle-explicitness)),
not the influence-edge or independence-dimension shape.

## 6. Anchor inventory pre-Gate status (Step 1.1 prep)

Applying the [`§0014`](./decision-log.md) lazy pre-Gate methodology to
§2.4 Step 1.1 (anchor inventory):

| Dependency | Status (as of 2026-05-21) | Pre-Gate disposition |
|---|---|---|
| OMQ #2 (Decay of influence) | resolved ([§0020](./decision-log.md), Candidate C) | discharged |
| OMQ #3 (Influence at projection vs substrate) | resolved ([§0021](./decision-log.md), Candidate α) | discharged |
| OMQ #1 (Granularity of derivation) | open | inherited from §2.3 v0.4; not §2.4 blocker per §0019 |
| OMQ #4 (Cross-domain provenance) | open | §3 (Non-Goals) territory per §0019; not §2.4 blocker |
| `ontology.md` Q5 "transitive?" half | open | cascade candidate per §0021 carry-forward; assess at §2.4 Step 1.1 |
| Q2 (Identity tiers) | resolved ([§0023](./decision-log.md), inception-phase single-tier) | discharged |
| Q4 (Cross-subtype operations) | deferred | orthogonal to §2.4 binding text per Q-X3 above |
| §2.3 BC1 inheritance (§2.4 side) | structural | inheritance citation in binding text; no pre-Gate RFC |
| Layer B follow-on RFC | on hold per [§0011](./decision-log.md) | post-§2.6; not §2.4 blocker |

**Single open pre-Gate candidate for §2.4 redaction:** Q5 "transitive?"
half. The committee at §2.4 Step 1.1 would assess whether the question is
load-bearing for §2.4 binding text or whether the Ontology-level prose at
[`provenance-model.md` §Inferential Provenance](../ontology/provenance-model.md)
hedge "to the extent they affected H₁'s formation" is sufficient as a
forward-reference contract.

## 7. Assessment outcome

The empirical pressure on §2.4 is **strong**: six concrete questions
(Q-§2.4-1 through Q-§2.4-6) have surfaced that the Charter working-
definition does not answer, and the implementation surface is committed
enough that the redaction has concrete material to anchor against.

The empirical pressure on §2.6 is **moderate**: five questions
(Q-§2.6-1 through Q-§2.6-5) have surfaced. The redaction would benefit
from §2.4 binding text being frozen first (per the [§0008](./decision-log.md)
order §2.5 → §2.3 → §2.4 → §2.6) because §2.6 questions are downstream of
§2.4's influence-edge wire shape.

**This is an assessment, not a trigger.** Whether to resume committee
redaction is a separate decision. Should the user direct redaction
resumption, the §2.4 Step 1.1 anchor inventory would:

1. Confirm Q5 "transitive?" pre-Gate status (open the RFC at `discussion`
   or accept the Ontology-level hedge).
2. Anchor against the six Q-§2.4-* questions enumerated above.
3. Carry forward to §2.6 redaction the Q-§2.6-* questions + the Layer B
   activation reconciliation per [§0011](./decision-log.md) +
   [§0019](./decision-log.md) Methodological Observation 3.

## References

- [Charter §2.4 (pending)](./constitutional-charter.md#24-inferential-influence-disclosure)
- [Charter §2.6 (pending)](./constitutional-charter.md#26-evidential-independence-integrity)
- [Charter §2.5 frozen v0.3](./constitutional-charter.md#25-hypothesis-lifecycle-explicitness)
- [`decision-log §0011`](./decision-log.md) — §2.5 Layer A / Layer B staged-combination
- [`decision-log §0019`](./decision-log.md) — §2.4 redaction plan (pre-implementation-pivot)
- [`decision-log §0020`](./decision-log.md) — OMQ #2 resolution (Candidate C — decay via supersession)
- [`decision-log §0021`](./decision-log.md) — OMQ #3 resolution (Candidate α — substrate-time generation)
- [`decision-log §0022`](./decision-log.md) — implementation pivot + §2.4/§2.6 posture shift
- [`provenance-model.md §Inferential Provenance`](../ontology/provenance-model.md)
- [`ontology-revision-layer-b-deep-criterion`](../rfcs/draft/ontology-revision-layer-b-deep-criterion.md) (on hold)
