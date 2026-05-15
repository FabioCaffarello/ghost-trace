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

### `hypothesis`

- **Canonical definition.** A Category III record; probabilistic inference whose boundaries, membership, and continued existence are matters of degree.
- **Introduction.** [Charter §1](./charter/constitutional-charter.md#1-thesis); [`ontology/entity-model.md` §Category III](./ontology/entity-model.md).
- **Stabilization.** [`decision-log.md` §0002 — Adopt ontological tripartition](./charter/decision-log.md).
- **Last amendment.** [`amendments.md` `v0.1` — Charter Inception](./charter/amendments.md).
- **Forbidden synonyms.** `inference` (the act, not the record); `cluster` (acceptable only for a specific subtype, and the subtype is itself an open modeling question — see [`ontology/entity-model.md` §Open Modeling Questions](./ontology/entity-model.md)).

### `assertion`

- **Canonical definition.** Any non-observation record the system produces — Category II or Category III.
- **Introduction.** [Charter §1](./charter/constitutional-charter.md#1-thesis).
- **Stabilization.** pending.
- **Last amendment.** [`amendments.md` `v0.1` — Charter Inception](./charter/amendments.md) (Thesis is frozen).
- **Forbidden synonyms.** `claim` (informal); `decision` (the Charter treats decisions as temporally extended sequences of assertions, not as one).

### `provenance`

- **Canonical definition.** Structural reference from an assertion back to the observations and prior assertions that produced it.
- **Introduction.** [Charter §1](./charter/constitutional-charter.md#1-thesis); [`ontology/provenance-model.md`](./ontology/provenance-model.md).
- **Stabilization.** pending ([Invariant 2.3](./charter/constitutional-charter.md#23-provenance-integrity) pending committee redaction).
- **Last amendment.** [`amendments.md` `v0.1` — Charter Inception](./charter/amendments.md) (Thesis introduces the term; full invariant pending).
- **Forbidden synonyms.** `metadata` (provenance is structure, not metadata, per [Charter §1](./charter/constitutional-charter.md#1-thesis)); `source` (when lineage is meant; collides with `provenance` per [`vocabulary-discipline` §4](../.claude/skills/ontology/vocabulary-discipline/SKILL.md)); `lineage` (acceptable in informal context, not as canonical replacement).

### `influence`

- **Canonical definition.** Inferential provenance; structural declaration that an assertion was formed under the influence of a prior assertion.
- **Introduction.** [Charter §1](./charter/constitutional-charter.md#1-thesis); [`ontology/provenance-model.md` §Inferential Provenance](./ontology/provenance-model.md).
- **Stabilization.** pending ([Invariant 2.4](./charter/constitutional-charter.md#24-inferential-influence-disclosure) pending committee redaction).
- **Last amendment.** pending (§2.4 not yet redacted; working text non-binding).
- **Forbidden synonyms.** `causality` (imports philosophical commitments the project has not made — see [`ambiguity-reducer`](../.claude/skills/epistemic/ambiguity-reducer/SKILL.md)); `dependency` (when inferential influence is meant).

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
- **Stabilization.** pending ([Invariant 2.6](./charter/constitutional-charter.md#26-evidential-independence-integrity) pending committee redaction).
- **Last amendment.** pending (§2.6 pending).
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
