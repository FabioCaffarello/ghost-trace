# Ghost Trace — Glossary

Source of truth for the canonical vocabulary of this project. Substituting a synonym for any canonical term introduces drift; vocabulary is constitutional ([`CONTRIBUTING.md` §Style](../CONTRIBUTING.md); [`.claude/CLAUDE.md` §3](../.claude/CLAUDE.md)).

## Status

This glossary's initial entries are derived from [`.claude/CLAUDE.md` §3 — Canonical vocabulary](../.claude/CLAUDE.md). Additions and changes follow the procedure in [`.claude/skills/ontology/vocabulary-discipline/`](../.claude/skills/ontology/vocabulary-discipline/SKILL.md). The forbidden-synonym table in `vocabulary-discipline` §4 is one view of the data carried per-term below; drift between the two views is a maintenance bug, checked by the Phase 8 SELF-AUDIT.

## Entry structure

Every entry carries five fields:

1. **Canonical definition.** One sentence.
2. **Introduction.** Document and section where the term was first introduced.
3. **Stabilization.** Decision-log entry that stabilized the term, if any. `pending` if no dedicated entry exists.
4. **Last amendment.** Charter amendment that last touched the term, if any. `pending` for terms in pending Charter sections or for non-Charter terms.
5. **Forbidden synonyms.** Terms that mean roughly this but are not this. Each carries a rewrite instruction.

Missing fields are marked `pending`, never omitted.

## Terms

### `substrate`

- **Canonical definition.** The immutable layer governed by Invariant 2.1; the primary event log and its archive.
- **Introduction.** [Charter §2.1 Boundary Conditions](./charter/constitutional-charter.md#21-observational-integrity); [`architecture/storage-model.md` Tiers 0–1](./architecture/storage-model.md).
- **Stabilization.** [`decision-log.md` §0001 — Adopt event-log immutability as constitutional invariant](./charter/decision-log.md).
- **Last amendment.** [`amendments.md` `v0.1` — Charter Inception](./charter/amendments.md).
- **Forbidden synonyms.** `store` (use `primary event log` or the specific tier); `log` standalone (use `primary event log` for the substrate or `decision log` for the ADR record); `data layer`.

### `projection`

- **Canonical definition.** A materialized view derived from the substrate; rebuildable, not bound by Invariant 2.1.
- **Introduction.** [Charter §2.1 Boundary Conditions](./charter/constitutional-charter.md#21-observational-integrity); [`architecture/projection-model.md`](./architecture/projection-model.md).
- **Stabilization.** pending.
- **Last amendment.** [`amendments.md` `v0.1` — Charter Inception](./charter/amendments.md) (the substrate/projection boundary is in frozen §2.1).
- **Forbidden synonyms.** `view` (when the substrate is meant; `projection` is the structural term); `cache` (overloaded — use `projection` when the structural meaning is intended).

### `observation`

- **Canonical definition.** A Category I record committed to the primary event log; immutable historical fact.
- **Introduction.** [Charter §1 Thesis](./charter/constitutional-charter.md#1-thesis), [§2.2 Epistemic Separation](./charter/constitutional-charter.md#22-epistemic-separation); [`ontology/entity-model.md` §Category I](./ontology/entity-model.md).
- **Stabilization.** [`decision-log.md` §0002 — Adopt ontological tripartition](./charter/decision-log.md).
- **Last amendment.** [`amendments.md` `v0.1` — Charter Inception](./charter/amendments.md).
- **Forbidden synonyms.** `fact` (only acceptable for an explicit Category I record; never about a derived assertion); `event` (overloaded — use `observation` for Category I, `lifecycle event` for hypothesis transitions); `record` (when a specific category is meant); `data` (undifferentiated).

### `operational construct`

- **Canonical definition.** A Category II record derived deterministically from observations under a versioned operational definition.
- **Introduction.** [Charter §1](./charter/constitutional-charter.md#1-thesis); [`ontology/entity-model.md` §Category II](./ontology/entity-model.md).
- **Stabilization.** [`decision-log.md` §0002 — Adopt ontological tripartition](./charter/decision-log.md).
- **Last amendment.** [`amendments.md` `v0.1` — Charter Inception](./charter/amendments.md).
- **Forbidden synonyms.** `result` (when the construct is meant; use `operational construct`); `derivation` (the act of derivation, not the record it produces); `aggregate` (acceptable only when the term is fully reduced to a specific operational construct).

### `operational session`

- **Canonical definition.** Category II operational construct: the system's reading of where a session operationally was, derived deterministically from `declared session` plus other Category I inputs under a versioned operational definition. The derivation is required to be deterministic per [Charter §2.2](./charter/constitutional-charter.md#22-epistemic-separation) (frozen Category II requirement); non-deterministic derivation would constitute a Category III misclassification.
- **Introduction.** [`decision-log.md` §0015 — Q1 resolution](./charter/decision-log.md); [`ontology/entity-model.md` §Category II post-Q1](./ontology/entity-model.md).
- **Stabilization.** Ontology v(post-Q1) — Drafted (Category II revision).
- **Last amendment.** [`decision-log.md` §0015](./charter/decision-log.md).
- **Forbidden synonyms.** `session` as a bare reference to the operational form (use `operational session`); `inferred session` (description rather than canonical term); `reconstructed session` (engineering vocabulary, not Ontology); `derived session` (acceptable in informal context; `operational session` is canonical).

### `hypothesis`

- **Canonical definition.** A Category III record; probabilistic inference whose boundaries, membership, and continued existence are matters of degree.
- **Introduction.** [Charter §1](./charter/constitutional-charter.md#1-thesis); [`ontology/entity-model.md` §Category III](./ontology/entity-model.md).
- **Stabilization.** [`decision-log.md` §0002 — Adopt ontological tripartition](./charter/decision-log.md).
- **Last amendment.** [`amendments.md` `v0.1` — Charter Inception](./charter/amendments.md).
- **Forbidden synonyms.** `inference` (the act, not the record); `cluster` (acceptable only for a specific subtype, and the subtype is itself an open modeling question — see [`ontology/entity-model.md` §Open Modeling Questions](./ontology/entity-model.md)).

### `automation group`

- **Canonical definition.** Category III hypothesis subtype: a set of actors whose behavioral patterns match a signature of automated (non-human) operation. The hypothesis asserts that the set is operated automatically; membership and signature evolve through hypothesis lifecycle operations.
- **Introduction.** [`decision-log.md` §0010 — Q2 resolution](./charter/decision-log.md); [`ontology/entity-model.md` §Category III, post-Q2](./ontology/entity-model.md).
- **Stabilization.** Ontology v(post-Q2) — Drafted.
- **Last amendment.** [`decision-log.md` §0010](./charter/decision-log.md).
- **Forbidden synonyms.** `bot cluster` (informal; ambiguous with classification rather than inference); `automated actors` (description, not the inferential entity).

### `behavioral cluster`

- **Canonical definition.** Category III hypothesis subtype: a set of actors whose behavioral patterns suggest operation by a common underlying entity. The hypothesis is about shared operatorship, not shared activity. Membership evolves through hypothesis lifecycle operations.
- **Introduction.** [`decision-log.md` §0010 — Q2 resolution](./charter/decision-log.md); [`ontology/entity-model.md` §Category III, post-Q2](./ontology/entity-model.md).
- **Stabilization.** Ontology v(post-Q2) — Drafted.
- **Last amendment.** [`decision-log.md` §0010](./charter/decision-log.md).
- **Forbidden synonyms.** `actor group` (does not specify the inferential character — an actor group is a description; `behavioral cluster` is a hypothesis); `account cluster` (collapses identity tier — `behavioral cluster` operates over actors, not accounts).

### `campaign hypothesis`

- **Canonical definition.** Category III hypothesis subtype: a set of events whose patterns suggest membership in a unified operation with thematic, temporal, or actor-level coherence. The hypothesis asserts that the events are part of one operation, not several coincident ones.
- **Introduction.** [`decision-log.md` §0010 — Q2 resolution](./charter/decision-log.md); [`ontology/entity-model.md` §Category III, post-Q2](./ontology/entity-model.md).
- **Stabilization.** Ontology v(post-Q2) — Drafted.
- **Last amendment.** [`decision-log.md` §0010](./charter/decision-log.md).
- **Forbidden synonyms.** `campaign` (the bare word denotes the operation itself; `campaign hypothesis` is the inferential entity about it); `event cluster` (collapses with `behavioral cluster` — `behavioral cluster` is over actors, `campaign hypothesis` is over events).

### `coordination ring`

- **Canonical definition.** Category III hypothesis subtype: a set of actors whose patterns of interaction with one another suggest coordinated action. The hypothesis is about relational structure among actors, not shared operatorship. Distinguished from `behavioral cluster`: cluster is about same-operator inference; ring is about coordinated-action inference among different operators.
- **Introduction.** [`decision-log.md` §0010 — Q2 resolution](./charter/decision-log.md); [`ontology/entity-model.md` §Category III, post-Q2](./ontology/entity-model.md).
- **Stabilization.** Ontology v(post-Q2) — Drafted.
- **Last amendment.** [`decision-log.md` §0010](./charter/decision-log.md).
- **Forbidden synonyms.** `coordination group` (description rather than hypothesis); `network` (overloaded — used elsewhere for graph structure, not the inferential entity).

### `session`

- **Canonical definition.** Domain concept distinguishing two typed entity-model forms: `declared session` (Category I primary observation) and `operational session` (Category II operational construct derived from declared inputs). Per [`decision-log.md` §0015](./charter/decision-log.md) (Q1 resolution), the typed duality is structural; bare `session` references in informal context should resolve to one of the two typed forms in canonical prose.
- **Introduction.** [`decision-log.md` §0015 — Q1 resolution](./charter/decision-log.md); [`ontology/entity-model.md` §Category I + §Category II post-Q1](./ontology/entity-model.md).
- **Stabilization.** Ontology v(post-Q1) — Drafted (Categories I + II revisions).
- **Last amendment.** [`decision-log.md` §0015](./charter/decision-log.md).
- **Forbidden synonyms.** None at the umbrella level; specific typed forms have their own forbidden-synonym entries. The bare `session` reference is acceptable as a domain-concept umbrella but in structural prose should resolve to `declared session` or `operational session`.

### `declared session`

- **Canonical definition.** Category I primary observation: the session as reported by a client SDK, network-level infrastructure collector, or other authoritative source. Carries source attribution and timing metadata. Immutable after commit per [Charter §2.1](./charter/constitutional-charter.md#21-observational-integrity) (frozen).
- **Introduction.** [`decision-log.md` §0015 — Q1 resolution](./charter/decision-log.md); [`ontology/entity-model.md` §Category I post-Q1](./ontology/entity-model.md).
- **Stabilization.** Ontology v(post-Q1) — Drafted (Category I revision).
- **Last amendment.** [`decision-log.md` §0015](./charter/decision-log.md).
- **Forbidden synonyms.** `session` as a bare reference to the declared form (use `declared session` for precision); `client-reported session` (description, not canonical term); `raw session` (suggests partial or processed; not canonical).

### `demotion`

- **Canonical definition.** Category III hypothesis lifecycle operation: the transition of a previously promoted hypothesis out of operational use as enrichment context. Demotion is recorded as an immutable lifecycle event in the primary event log per Invariant 2.5 (frozen v0.3). A demoted hypothesis remains in the substrate; demotion does not delete.
- **Introduction.** [`decision-log.md` §0011 — Q4 resolution](./charter/decision-log.md); [`ontology/lifecycle-semantics.md` §Category III, post-Q4](./ontology/lifecycle-semantics.md).
- **Stabilization.** Charter §2.5 v0.3.
- **Last amendment.** v0.3.
- **Forbidden synonyms.** `demote` (verb form is acceptable in informal use; the canonical noun is `demotion`); `unpromotion` (the lifecycle event is `demotion`, not its negation); `retraction` (this term may be used in a different sense for assertions; `demotion` is for hypotheses); `decommission` (engineering-systems vocabulary, not Ontology).

### `dissolution`

- **Canonical definition.** Category III hypothesis lifecycle operation: the recognition that a hypothesis no longer corresponds to any underlying phenomenon, recorded as an immutable lifecycle event in the primary event log per Invariant 2.5 (frozen v0.3). Distinguished from `demotion`: demotion withdraws operational use; dissolution recognizes non-existence of the underlying phenomenon. The two operations are not interchangeable.
- **Introduction.** [`decision-log.md` §0013 — Gate §2.5 redaction](./charter/decision-log.md); [`ontology/lifecycle-semantics.md` §Hypothesis (Category III)](./ontology/lifecycle-semantics.md).
- **Stabilization.** Charter §2.5 v0.3.
- **Last amendment.** v0.3.
- **Forbidden synonyms.** `dissolve` (verb form is acceptable in informal use; the canonical noun is `dissolution`); `removal` (substrate immutability per [Invariant 2.1](./charter/constitutional-charter.md#21-observational-integrity) prevents removal; dissolution is a lifecycle event, not a record deletion); `archival` (operational concept; dissolution is the constitutional lifecycle event).

### `formation`

- **Canonical definition.** Category III hypothesis lifecycle operation: the creation of a hypothesis (one of the four concrete subtypes per Q2-A.2) when an inference process recognizes accumulated observations crossing a structural threshold, recorded as an immutable lifecycle event in the primary event log per Invariant 2.5 (frozen v0.3).
- **Introduction.** [`decision-log.md` §0013 — Gate §2.5 redaction](./charter/decision-log.md); [`ontology/lifecycle-semantics.md` §Hypothesis (Category III)](./ontology/lifecycle-semantics.md).
- **Stabilization.** Charter §2.5 v0.3.
- **Last amendment.** v0.3.
- **Forbidden synonyms.** `creation` (informal; `formation` is canonical for the structural lifecycle event); `instantiation` (engineering vocabulary); `emergence` (philosophical import the project has not made; use `formation`).

### `merge`

- **Canonical definition.** Category III hypothesis lifecycle operation: the combination of two hypotheses recognized as describing the same underlying phenomenon, recorded as an immutable lifecycle event referencing both antecedents and the produced hypothesis per Invariant 2.5 (frozen v0.3). Cross-subtype merge produces a typed output record per [`ontology/entity-model.md` §Cross-subtype operations](./ontology/entity-model.md).
- **Introduction.** [`decision-log.md` §0013 — Gate §2.5 redaction](./charter/decision-log.md); [`ontology/lifecycle-semantics.md` §Hypothesis (Category III)](./ontology/lifecycle-semantics.md).
- **Stabilization.** Charter §2.5 v0.3.
- **Last amendment.** v0.3.
- **Forbidden synonyms.** `combine` (informal); `unify` (overloaded; `merge` is the canonical operation name); `consolidate` (engineering vocabulary).

### `split`

- **Canonical definition.** Category III hypothesis lifecycle operation: the division of a hypothesis recognized as containing multiple distinct phenomena into multiple successor hypotheses, recorded as an immutable lifecycle event referencing the antecedent and each successor per Invariant 2.5 (frozen v0.3).
- **Introduction.** [`decision-log.md` §0013 — Gate §2.5 redaction](./charter/decision-log.md); [`ontology/lifecycle-semantics.md` §Hypothesis (Category III)](./ontology/lifecycle-semantics.md).
- **Stabilization.** Charter §2.5 v0.3.
- **Last amendment.** v0.3.
- **Forbidden synonyms.** `divide` (informal); `partition` (mathematical import; `split` is the canonical operation name); `fork` (engineering vocabulary).

### `promotion`

- **Canonical definition.** Category III hypothesis lifecycle operation: the transition of a hypothesis from active inference to operational use as enrichment context. Promotion is recorded as an immutable lifecycle event in the primary event log per Invariant 2.5 (frozen v0.3). The promotion event carries the structural parameters governing the hypothesis's subsequent demotion-candidacy timing (Layer A of the staged-combination criterion per [`decision-log.md` §0011](./charter/decision-log.md)).
- **Introduction.** [`decision-log.md` §0011 — Q4 resolution](./charter/decision-log.md); [`ontology/lifecycle-semantics.md` §The Promotion Mechanism, post-Q4](./ontology/lifecycle-semantics.md).
- **Stabilization.** Charter §2.5 v0.3.
- **Last amendment.** v0.3.
- **Forbidden synonyms.** `promote` (verb form is acceptable in informal use; the canonical noun is `promotion`); `activation` (engineering-systems vocabulary); `enrichment promotion` (the bare term `promotion` is canonical; `enrichment` describes the use the promoted hypothesis is admitted to, not the operation).

### `assertion`

- **Canonical definition.** Any non-observation record the system produces — Category II or Category III.
- **Introduction.** [Charter §1](./charter/constitutional-charter.md#1-thesis).
- **Stabilization.** pending.
- **Last amendment.** [`amendments.md` `v0.1` — Charter Inception](./charter/amendments.md) (Thesis is frozen).
- **Forbidden synonyms.** `claim` (informal); `decision` (the Charter treats decisions as temporally extended sequences of assertions, not as one).

### `provenance`

- **Canonical definition.** Structural reference from an assertion back to the observations and prior assertions that produced it.
- **Introduction.** [Charter §1](./charter/constitutional-charter.md#1-thesis); [`ontology/provenance-model.md`](./ontology/provenance-model.md).
- **Stabilization.** [`decision-log.md` §0017 — Gate §2.3 closure](./charter/decision-log.md) ([Invariant 2.3](./charter/constitutional-charter.md#23-provenance-integrity) frozen v0.4).
- **Last amendment.** v0.4 (§2.3 pending → frozen at amendment v0.4).
- **Forbidden synonyms.** `metadata` (provenance is structure, not metadata, per [Charter §1](./charter/constitutional-charter.md#1-thesis)); `source` (when lineage is meant; collides with `provenance` per [`vocabulary-discipline` §4](../.claude/skills/ontology/vocabulary-discipline/SKILL.md)); `lineage` (acceptable in informal context, not as canonical replacement).

### `influence`

- **Canonical definition.** Inferential provenance; structural declaration that an assertion was formed under the influence of a prior assertion.
- **Introduction.** [Charter §1](./charter/constitutional-charter.md#1-thesis); [`ontology/provenance-model.md` §Inferential Provenance](./ontology/provenance-model.md).
- **Stabilization.** [Charter §2.4](./charter/constitutional-charter.md#24-inferential-influence-disclosure) frozen v0.5.
- **Last amendment.** v0.5.
- **Forbidden synonyms.** `causality` (imports philosophical commitments the project has not made — see [`ambiguity-reducer`](../.claude/skills/epistemic/ambiguity-reducer/SKILL.md)); `dependency` (when inferential influence is meant).

### `observational evidence`

- **Canonical definition.** The structural surface against which an Assertion's observational provenance claim is reconstructible: the substrate-grounded subject reference (`subject_ref_observation`, `subject_ref_construct`, or `subject_ref_hypothesis`) and the corresponding provenance chain back to Category I records. Operationalization of the informal vocabulary item `evidence` under [Charter §2.3 §Observational Provenance](./charter/constitutional-charter.md#23-provenance-integrity), introduced through Step 1.3 Path 1 of the §2.4 redaction per [`decision-log.md` §0099](./charter/decision-log.md).
- **Introduction.** [`decision-log.md` §0099 — Gate §2.4 closure](./charter/decision-log.md); [Charter §2.4](./charter/constitutional-charter.md#24-inferential-influence-disclosure) (frozen v0.5).
- **Stabilization.** Charter §2.4 v0.5.
- **Last amendment.** v0.5.
- **Forbidden synonyms.** `evidence` (bare term remains informal; `observational evidence` is canonical when the structural reference chain to Category I substrate is meant); `proof` (imports closure semantics the substrate does not provide); `support` (collapses observational and inferential grounds — use `observational evidence` or `inferential commitment` per the distinction `evidential independence` exists to preserve).

### `inferential commitment`

- **Canonical definition.** The structural declaration that an Assertion is held under inferential influence from a prior Assertion: the populated typed `subject_ref_construct` or `subject_ref_hypothesis` field on the Assertion together with the `influenced_by` chain reconstructible from substrate per [Charter §2.4](./charter/constitutional-charter.md#24-inferential-influence-disclosure). Operationalization of the informal vocabulary item `belief` under §2.4, introduced through Step 1.3 Path 1 of the §2.4 redaction per [`decision-log.md` §0099](./charter/decision-log.md).
- **Introduction.** [`decision-log.md` §0099 — Gate §2.4 closure](./charter/decision-log.md); [Charter §2.4](./charter/constitutional-charter.md#24-inferential-influence-disclosure) (frozen v0.5).
- **Stabilization.** Charter §2.4 v0.5.
- **Last amendment.** v0.5.
- **Forbidden synonyms.** `belief` (informal; the structural commitment is what is canonical, not the mental state the word suggests); `assumption` (philosophical import the project has not made; use `inferential commitment` when the structural declaration is meant); `stance` (overloaded with engineering and discourse vocabulary).

### `supersession`

- **Canonical definition.** Replacement of a record's interpretation by a new record; never mutation of the original.
- **Introduction.** [Charter §2.1](./charter/constitutional-charter.md#21-observational-integrity) (Forbidden Anti-Patterns refers to "supersession of an observation's interpretation"); [`decision-log.md` §Format](./charter/decision-log.md).
- **Stabilization.** pending.
- **Last amendment.** [`amendments.md` `v0.1` — Charter Inception](./charter/amendments.md).
- **Forbidden synonyms.** `update` (the original is preserved under supersession, not updated); `overwrite` (forbidden by Invariant 2.1); `replace` (when the original is preserved, `supersede` is the canonical term).

### `enrichment`

- **Canonical definition.** Operational knowledge paired with observations as a separate stream of immutable events; not a mutation of the observation.
- **Introduction.** [Charter §1](./charter/constitutional-charter.md#1-thesis) (mentions the failure mode in which promoted hypotheses re-enter as enrichment); [`architecture/event-flow.md` Phase 2](./architecture/event-flow.md).
- **Stabilization.** pending.
- **Last amendment.** [`amendments.md` `v0.1` — Charter Inception](./charter/amendments.md) (Thesis introduces the term; structural detail pending).
- **Forbidden synonyms.** `decoration`; `tagging` (annotation of observations is forbidden by [Invariant 2.1](./charter/constitutional-charter.md#21-observational-integrity)); `lookup` (the enrichment stream is event-sourced, not a mutable lookup).

### `committee mode`

- **Canonical definition.** The redaction discipline applied to Charter sections one at a time, with explicit defense of each word choice.
- **Introduction.** [Charter status banner](./charter/constitutional-charter.md); [`amendments.md` §Amendment Process](./charter/amendments.md).
- **Stabilization.** pending.
- **Last amendment.** [`amendments.md` `v0.1` — Charter Inception](./charter/amendments.md).
- **Forbidden synonyms.** `working group` (committee mode is a discipline, not a body); `editing` (committee mode is not ordinary editing).

### `frozen`

- **Canonical definition.** Status of a Charter section that may not be edited except through formal amendment.
- **Introduction.** [Charter status banner](./charter/constitutional-charter.md).
- **Stabilization.** pending.
- **Last amendment.** [`amendments.md` `v0.1` — Charter Inception](./charter/amendments.md).
- **Forbidden synonyms.** `locked`; `final` (frozen sections may still be amended; `final` implies they cannot).

### `pending`

- **Canonical definition.** Status of a Charter section under committee redaction; the working text is non-binding.
- **Introduction.** [Charter §2.3–§2.6 status notes](./charter/constitutional-charter.md).
- **Stabilization.** pending.
- **Last amendment.** [`amendments.md` `v0.1` — Charter Inception](./charter/amendments.md).
- **Forbidden synonyms.** `draft` (the working text constrains what the final invariant may say more than a draft does); `placeholder`.

### `amendment`

- **Canonical definition.** A formal modification of the Charter, recorded in `amendments.md` with a version bump.
- **Introduction.** [`amendments.md` §Amendment Discipline](./charter/amendments.md).
- **Stabilization.** pending.
- **Last amendment.** [`amendments.md` `v0.1` — Charter Inception](./charter/amendments.md) (the amendment process is itself recorded in that entry).
- **Forbidden synonyms.** `edit` (an edit to a frozen section is an amendment by definition; collapsing the two is the failure mode `charter-guardian` prevents); `revision` (acceptable for subordinate documents, not for the Charter).

### `constitutional invariant`

- **Canonical definition.** A structural rule of Ghost Trace that satisfies all four qualification criteria stated in [Charter §2 header](./charter/constitutional-charter.md#2-constitutional-invariants) and the falsifiability discipline of [Charter §4](./charter/constitutional-charter.md#4-constitutional-design-rule): structurally enforceable; constraining of implementation; identity-defining; independent of operator interpretation; and structurally falsifiable.
- **Introduction.** [Charter §2 — Constitutional Invariants](./charter/constitutional-charter.md#2-constitutional-invariants); [Charter §4 — Constitutional Design Rule](./charter/constitutional-charter.md#4-constitutional-design-rule).
- **Stabilization.** Charter v0.1 (§2 header + §2.1 + §2.2 frozen); refined in v0.1.1 (§2 header explicitly frozen) and v0.2 (§4 redacted as formal locus).
- **Last amendment.** v0.2.
- **Forbidden synonyms.** `principle` (broader; not necessarily structurally enforceable); `rule` (acceptable in informal use; `invariant` is canonical for structural rules); `axiom` (acceptable in mathematical contexts; not Ghost Trace's term).

### `subordination`

- **Canonical definition.** Hierarchical relation in which lower-ranked documents must not conflict with higher-ranked documents; conflicts resolve upward and the lower document is revised.
- **Introduction.** [`README.md` §Document Hierarchy](../README.md); [`decision-log.md` §0004](./charter/decision-log.md).
- **Stabilization.** [`decision-log.md` §0004 — Charter is authoritative; subordinate documents may evolve under implementation pressure](./charter/decision-log.md).
- **Last amendment.** Not amended; the precedence rule was considered for Charter §4 in the Gate 1 redaction (amendment v0.2) and excluded as a substantive duplicate of CLAUDE.md §2 and `subordination-checker`. The rule remains operationally enforced (`subordination-checker` + CLAUDE.md §2 + CI `subordination-check` job). See [`decision-log.md` §0007](./charter/decision-log.md).
- **Forbidden synonyms.** `precedence` (acceptable in informal use; `subordination` is canonical for the relation); `hierarchy` (names the structure; `subordination` names the relation between layers).

### `falsifiability`

- **Canonical definition.** The requirement that constitutional claims be structurally falsifiable; non-falsifiable claims are rejected on procedural grounds.
- **Introduction.** [Charter §4 Constitutional Design Rule](./charter/constitutional-charter.md#4-constitutional-design-rule); [`CONTRIBUTING.md` §Style](../CONTRIBUTING.md).
- **Stabilization.** Charter §4 v0.2 (this redaction).
- **Last amendment.** v0.2.
- **Forbidden synonyms.** `testability` (overloaded with engineering tests); `verifiability` (acceptable in some contexts but not as a constitutional term).

### `evidential independence`

- **Canonical definition.** The second dimension of an inferential assertion, distinct from confidence; defends against recursive belief inflation.
- **Introduction.** [Charter §1](./charter/constitutional-charter.md#1-thesis) (mentioned); [`ontology/entity-model.md` §Category III](./ontology/entity-model.md).
- **Stabilization.** [Invariant 2.6](./charter/constitutional-charter.md#26-evidential-independence-integrity) frozen v0.6 (Gate §2.6 closure per [`decision-log.md` §0129](./charter/decision-log.md)).
- **Last amendment.** v0.6 (§2.6 pending → frozen at amendment v0.6).
- **Forbidden synonyms.** `confidence` (the two are paired but distinct; collapsing them is the failure mode §2.6 prevents); `independence` standalone (overloaded — `evidential independence` is the canonical full term in this project).

### `phase 1/2/3/4 replay`

- **Canonical definition.** The four replay contracts: deterministic (Phases 1–2), reconstructive (Phase 3), retrospective analytical (Phase 4).
- **Introduction.** [`architecture/replay-model.md`](./architecture/replay-model.md).
- **Stabilization.** pending (replay model is a scaffold).
- **Last amendment.** pending (architecture is not a Charter element).
- **Forbidden synonyms.** `replay` standalone (acceptable informally; `phase N replay` is canonical when contract is meant); `reconstruction` (the act, not the contract).

## Open lineage gaps

Glossary entries with `pending` in any field document gaps in lineage at the current Charter version. The gaps are normal — many terms await committee redaction of the relevant Charter section. Each gap closes when:

- A decision-log entry stabilizes the term, or
- A Charter amendment introduces or modifies the term, or
- The corresponding committee redaction completes.

The Phase 8 SELF-AUDIT verifies that pending fields close in step with the underlying Charter changes.
