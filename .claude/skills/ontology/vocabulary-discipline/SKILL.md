---
name: vocabulary-discipline
description: Enforce the canonical vocabulary AND carry term-level provenance — which document introduced the term, which decision stabilized it, which amendment last touched it. Use this skill ALWAYS when writing or editing prose in any document, ALWAYS when a new term is encountered, and ALWAYS when reviewing an RFC or amendment. Vocabulary drift is constitutional drift; this skill is the blocking gate that prevents it from entering the substrate.
---

# vocabulary-discipline

Vocabulary in this project is constitutional ([`CONTRIBUTING.md` §Style](../../../../CONTRIBUTING.md); [`.claude/CLAUDE.md` §3](../../../CLAUDE.md)). Using a different word for the same concept introduces drift; drift hardens into structural divergence between documents that nominally agree.

This skill enforces the canonical vocabulary against [`docs/glossary.md`](../../../../docs/glossary.md), which is the source of truth for canonical terms and their term-level provenance.

## 1. Required structure of every glossary entry

Every term in `docs/glossary.md` carries five fields. Entries with missing fields are flagged by the Phase 8 SELF-AUDIT; missing fields are marked `pending` rather than omitted.

1. **Canonical definition.** One sentence. Reduced to substrate artifacts wherever possible.
2. **Introduction.** The document and section where the term was first introduced (Charter §, Ontology document §, architecture document §, etc.).
3. **Stabilization.** The decision-log entry that stabilized the term, if any. `pending` if no dedicated entry exists.
4. **Last amendment.** The Charter amendment that last touched the term, if any. `pending` for terms in pending Charter sections or non-Charter documents.
5. **Forbidden synonyms.** Terms that mean roughly this but are not this. Each forbidden synonym carries a rewrite instruction.

The structure mirrors the project's provenance discipline: terms carry lineage just as assertions carry provenance.

## 2. Procedure for any new term proposed in any document

Apply in order.

### Step 1 — Search

Search [`docs/glossary.md`](../../../../docs/glossary.md) for a synonym or near-synonym of the proposed term. If one exists, use the canonical term instead — do not introduce the new term.

### Step 2 — If no synonym exists

The term cannot be introduced casually in prose. Introduction requires:

- Adding the term to [`docs/glossary.md`](../../../../docs/glossary.md) with all five fields. Missing fields are marked `pending`, never omitted.
- Recording the introduction as a new entry in [`docs/charter/decision-log.md`](../../../../docs/charter/decision-log.md) following the existing format. The entry's `Decision` names the term and its canonical definition; `Consequences` lists which subordinate documents must be updated.

Exception: terms introduced in a Charter section do not require a separate decision-log entry — the Charter section itself is the introduction. The glossary entry's `Introduction` field points to the Charter section, and the `Last amendment` field references the Charter amendment that promulgated the section.

### Step 3 — If the proposed "new" term is actually a refinement of an existing one

The term is not new. Update the existing entry's `Canonical definition` only through an `ontology-revision` RFC (if non-constitutional) or a `charter-amendment` RFC (if the term is constitutional). Silent refinement of an existing term is drift.

## 3. Procedure for any use of a term in prose

### Match the canonical definition

Read the canonical definition in `docs/glossary.md`. If the use in prose matches, proceed.

### Informal or metaphorical use

If the use is informal — `event` to mean any happening rather than a Category I observation, `trust` to mean reliability rather than a non-defined term — rewrite. Vocabulary is load-bearing.

### Use suggests the canonical definition is wrong

If the use names something the canonical definition does not cover, the canonical definition may be incomplete. Do not edit the canonical definition in prose; raise an `ontology-revision` RFC (or `charter-amendment` RFC if the term is constitutional). The glossary follows the source documents, never the other way around.

## 4. Forbidden synonyms (blocking)

This table is one view of the same data carried per-term in [`docs/glossary.md`](../../../../docs/glossary.md). The two views must agree; drift between them is checked by the Phase 8 SELF-AUDIT. Items in this table are blocking — the pre-commit hook (`pre-commit-doc-check.sh`, Phase 7) rejects them.

| Forbidden | Canonical | Rewrite |
|---|---|---|
| `log` (standalone) | `primary event log` for the substrate; `decision log` for the ADR record | Use the full canonical term. Never `log` alone. |
| `schema` (where the meaning is a type) | `type` or `category` | Reserve `schema` for the formal data definition in `schemas/`. Use `type` or `category` for ontological distinction. |
| `tag` (verb, applied to substrate) | (no canonical; the operation is forbidden) | Annotating an observation with inferential content is forbidden by [Invariant 2.1](../../../../docs/charter/constitutional-charter.md#21-observational-integrity). The rewrite is structural: produce a new assertion that references the observation. |
| `flag` (verb, applied to substrate) | (no canonical; the operation is forbidden) | See `tag`. Annotation of an observation with inferential content is forbidden. |
| `score` (used loosely) | `confidence` or `evidential independence` | Pick the dimension. [Invariant 2.6 pending](../../../../docs/charter/constitutional-charter.md#26-evidential-independence-integrity) requires both to be reported separately; the bare term `score` collapses them. |
| `fact` (about anything other than an observation) | `observation` | Only Category I records are facts in this project. Operational constructs are derived; hypotheses are inferred. |
| `result` (about anything other than a projection at a point in time) | `assertion`, `projection`, or specific category | A `result` returned to a query is a projection at a point in time. A `result` produced by inference is a Category II or III assertion. |
| `data` (about the substrate) | `observation` (Category I); `assertion` (II/III); or `record` only with the specific class noun | The substrate is composed of named categories, not undifferentiated `data`. |
| `store` (about the substrate) | `primary event log`, `archive`, or specific tier | The substrate is the event log and its archive ([`storage-model.md`](../../../../docs/architecture/storage-model.md)); call it that. |
| `metadata` (used about provenance) | `provenance` | Provenance is structure, not metadata ([Charter §1](../../../../docs/charter/constitutional-charter.md#1-thesis)). |
| `update` (applied to an observation or hypothesis state) | `supersede` (typed transformation) | Observations cannot be updated ([Invariant 2.1](../../../../docs/charter/constitutional-charter.md#21-observational-integrity)); hypothesis state is a projection over a lifecycle event history per [§2.5](../../../../docs/charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness), not mutable. |
| `signal` (used informally) | `operational construct` or `intermediate value` | See [`ambiguity-reducer`](../../epistemic/ambiguity-reducer/SKILL.md) for the disambiguation procedure. |

### Canonical-phrase exemptions

Certain multi-word canonical phrases legitimately contain forbidden-synonym substrings. The hook does not report a forbidden term when it appears within a registered canonical phrase. The whitelist is small, explicit, and grows only via decision-log entry.

Currently registered (per [`decision-log.md` §0006](../../../../docs/charter/decision-log.md)):

- **`primary event log`** — contains `log`; canonical name for the substrate per [Charter §2.1](../../../../docs/charter/constitutional-charter.md#21-observational-integrity) and [`storage-model.md`](../../../../docs/architecture/storage-model.md).
- **`decision log`** — contains `log`; canonical name for the ADR record per [`decision-log.md`](../../../../docs/charter/decision-log.md).
- **`event log`** — contains `log`; alternate canonical reference to the substrate primary tier.
- **`historical fact`** — contains `fact`; verbatim §2.2 definition of Category I.

This list does not include other multi-word phrases by pattern matching. New canonical phrases require an explicit decision-log entry referencing this section.

The table grows as new forbidden synonyms surface. Additions follow the procedure in [`ambiguity-reducer` §3](../../epistemic/ambiguity-reducer/SKILL.md) and are reflected in both this skill and `docs/glossary.md`.

## 5. Relationship to `ambiguity-reducer`

The two skills are distinct:

- `ambiguity-reducer` is **advisory** and flags terms that *may yet* have a legitimate use ([`.claude/CLAUDE.md` §5.3](../../../CLAUDE.md)). The writer decides.
- `vocabulary-discipline` is **blocking** for items in §4. Forbidden synonyms are not advisory; they are drift, and the pre-commit hook rejects them.

The watchlists are non-overlapping. A term moves from `ambiguity-reducer` to this skill only when an RFC has documented its forbidden status with a rewrite.

## 6. Source citations used

- [`docs/glossary.md`](../../../../docs/glossary.md)
- [`docs/charter/constitutional-charter.md` §1 Thesis; §2.1 Observational Integrity; §2.5, §2.6 pending](../../../../docs/charter/constitutional-charter.md)
- [`docs/architecture/storage-model.md`](../../../../docs/architecture/storage-model.md)
- [`CONTRIBUTING.md` §Style](../../../../CONTRIBUTING.md)
- [`.claude/CLAUDE.md` §3 Canonical vocabulary; §5.3 enforcement posture](../../../CLAUDE.md)
- [`.claude/skills/epistemic/ambiguity-reducer/SKILL.md`](../../epistemic/ambiguity-reducer/SKILL.md)
