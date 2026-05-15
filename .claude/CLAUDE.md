# Ghost Trace — Operational Context for Claude Code

This file is loaded into every Claude Code session in this repository. Read it before acting.

## 1. What this repository is

> Ghost Trace is a behavioral intelligence substrate designed to preserve the epistemic integrity of operational knowledge — the continued capacity to distinguish what was observed from what was inferred.
>
> — [Charter §1](../docs/charter/constitutional-charter.md#1-thesis)

The repository is in pre-implementation constitutional drafting. The Charter is `v0.2`: §1 Thesis, §2 Invariant qualification criteria, §2.1 Observational Integrity, §2.2 Epistemic Separation, and §4 Constitutional Design Rule are frozen; §2.3–§2.6 and §3 Non-Goals are pending committee redaction. Implementation work is gated on Ontology stabilization. The gate criteria are operationalized in the `implementation-readiness-evaluator` skill (to be created in Phase 5).

## 2. Document hierarchy

Documents in this repository have explicit ranks. Conflicts resolve upward; lower-ranked documents are revised. The Charter is never silently amended via a subordinate edit.

1. **[Constitutional Charter](../docs/charter/constitutional-charter.md)** — authoritative. Changes require formal amendment per [`amendments.md`](../docs/charter/amendments.md).
2. **[Ontology](../docs/ontology/ontology.md)** — formalizes Charter concepts. Subordinate to the Charter.
3. **[Architecture](../docs/architecture/)** — translates the Ontology into operational design. Subordinate to both.
4. **[RFCs](../docs/rfcs/)** — proposals subject to constitutional review.
5. **[Schemas](../schemas/)** — materialize the Ontology.
6. **[Services](../services/)** — implementations.

A parallel rule applies to operational documents: when [`CONTRIBUTING.md`](../CONTRIBUTING.md) (process) and `WORKFLOW.md` (tooling, Phase 7) conflict, `CONTRIBUTING.md` wins. Process is closer to the Charter in function; tooling is convenience.

## 3. Canonical vocabulary

These terms are constitutional. Substituting a synonym for any of them introduces drift. Use the term as written.

- **substrate** — the immutable layer governed by Invariant 2.1; primary event log and its archive ([Charter §2.1 Boundary Conditions](../docs/charter/constitutional-charter.md#21-observational-integrity); [`storage-model.md` Tiers 0–1](../docs/architecture/storage-model.md)).
- **projection** — materialized view derived from the substrate; rebuildable, not bound by Invariant 2.1 ([`projection-model.md`](../docs/architecture/projection-model.md)).
- **observation** — Category I record; immutable historical fact ([Charter §2.2](../docs/charter/constitutional-charter.md#22-epistemic-separation); [`entity-model.md` Category I](../docs/ontology/entity-model.md)).
- **operational construct** — Category II record; derived deterministically from observations under a versioned operational definition ([`entity-model.md` Category II](../docs/ontology/entity-model.md)).
- **hypothesis** — Category III record; probabilistic inference whose boundaries are matters of degree ([Charter §2.2](../docs/charter/constitutional-charter.md#22-epistemic-separation); [`entity-model.md` Category III](../docs/ontology/entity-model.md)).
- **automation group** — Category III hypothesis subtype: set of actors whose behavioral patterns match a signature of automated (non-human) operation ([decision-log §0010](../docs/charter/decision-log.md); [`entity-model.md` Category III](../docs/ontology/entity-model.md)).
- **behavioral cluster** — Category III hypothesis subtype: set of actors whose behavioral patterns suggest operation by a common underlying entity ([decision-log §0010](../docs/charter/decision-log.md); [`entity-model.md` Category III](../docs/ontology/entity-model.md)).
- **campaign hypothesis** — Category III hypothesis subtype: set of events whose patterns suggest membership in a unified operation ([decision-log §0010](../docs/charter/decision-log.md); [`entity-model.md` Category III](../docs/ontology/entity-model.md)).
- **coordination ring** — Category III hypothesis subtype: set of actors whose patterns of interaction suggest coordinated action ([decision-log §0010](../docs/charter/decision-log.md); [`entity-model.md` Category III](../docs/ontology/entity-model.md)).
- **assertion** — any non-observation record the system produces (Category II or III) ([Charter §1](../docs/charter/constitutional-charter.md#1-thesis)).
- **provenance** — structural reference from an assertion back to the observations and prior assertions that produced it ([Charter §1](../docs/charter/constitutional-charter.md#1-thesis); [`provenance-model.md`](../docs/ontology/provenance-model.md)).
- **influence** — inferential provenance; declaration that an assertion was formed under the influence of a prior assertion ([Charter §1; §2.4 pending](../docs/charter/constitutional-charter.md#24-inferential-influence-disclosure); [`provenance-model.md`](../docs/ontology/provenance-model.md)).
- **supersession** — replacement of a record's interpretation by a new record; never mutation of the original ([Charter §2.1](../docs/charter/constitutional-charter.md#21-observational-integrity); [`decision-log.md` §Format](../docs/charter/decision-log.md)).
- **enrichment** — operational knowledge paired with observations as a separate stream of immutable events; not a mutation ([`event-flow.md` Phase 2](../docs/architecture/event-flow.md)).
- **committee mode** — redaction discipline applied to Charter sections one at a time, with explicit defense of each word choice ([`amendments.md` §Amendment Process](../docs/charter/amendments.md)).
- **frozen** — Charter section status: not editable except via formal amendment ([Charter status banner](../docs/charter/constitutional-charter.md)).
- **pending** — Charter section status: in committee, working text non-binding ([Charter §2.3–§2.6 status](../docs/charter/constitutional-charter.md)).
- **amendment** — formal modification of the Charter, recorded in `amendments.md` and version-bumped ([`amendments.md` §Amendment Discipline](../docs/charter/amendments.md)).
- **constitutional invariant** — structural rule of Ghost Trace satisfying the four qualification criteria of §2 and the falsifiability discipline of §4 ([Charter §2](../docs/charter/constitutional-charter.md#2-constitutional-invariants); [Charter §4](../docs/charter/constitutional-charter.md#4-constitutional-design-rule); [`docs/glossary.md`](../docs/glossary.md)).
- **subordination** — hierarchical relation: lower-ranked documents must not conflict with higher-ranked documents ([`README.md` §Document Hierarchy](../README.md)).
- **falsifiability** — constitutional claims must be structurally falsifiable; non-falsifiable claims are rejected on procedural grounds ([Charter §4](../docs/charter/constitutional-charter.md#4-constitutional-design-rule); [`CONTRIBUTING.md` §Style](../CONTRIBUTING.md)).
- **evidential independence** — second dimension of an inferential assertion, distinct from confidence; defends against recursive belief inflation ([Charter §2.6 pending](../docs/charter/constitutional-charter.md#26-evidential-independence-integrity); [`entity-model.md` Category III](../docs/ontology/entity-model.md)).
- **phase 1/2/3/4 replay** — the four replay contracts: deterministic (Phases 1–2), reconstructive (Phase 3), retrospective analytical (Phase 4) ([`replay-model.md`](../docs/architecture/replay-model.md)).

## 4. Charter status at a glance

| Element | Status |
|---|---|
| §1 Thesis | frozen |
| §2 Invariant qualification criteria | frozen |
| §2.1 Observational Integrity | frozen |
| §2.2 Epistemic Separation | frozen |
| §2.3 Provenance Integrity | pending |
| §2.4 Inferential Influence Disclosure | pending |
| §2.5 Hypothesis Lifecycle Explicitness | pending |
| §2.6 Evidential Independence Integrity | pending |
| §3 Non-Goals | pending |
| §4 Constitutional Design Rule | frozen |

This table is the source of truth for which sections are protected against edit. The `charter-guardian` skill and the pre-commit hook both read it. Update it only when the Charter actually changes. Keeping this table in sync with `constitutional-charter.md` is the canonical inconsistency mode of this infrastructure; the Phase 8 SELF-AUDIT checks for it. Amendment v0.1.1 added the §2 qualification criteria row, resolving SELF-AUDIT §7 Finding 7.1. Amendment v0.2 moved §4 from `pending` to `frozen` (Gate 1 pilot redaction).

## 5. Governance decisions binding this infrastructure

Resolutions to the seven open questions from `PLAN.md` §5. Binding for all subsequent phases. Not subject to silent revision in skill or command authorship.

1. **Skill packaging.** Skills are directories containing `SKILL.md`. Material exceeding ~300 lines is split into `references/` subdirectories within the skill, referenced explicitly from `SKILL.md`.

2. **Hook architecture is three-tier, single script.** The same script (`.claude/hooks/pre-commit-doc-check.sh`) runs in three contexts:
   - Git pre-commit hook (enforcement, via `core.hooksPath`). Defense-of-record; protects against any committer regardless of tooling.
   - Claude Code event hook (feedback, via `settings.json`). Surfaces issues during the session before commit.
   - CI workflow (PR-level enforcement, Phase 7). Final gate before merge.
   If these three diverge, the git pre-commit hook is the source of truth — it is the actual enforcement boundary. The Claude event hook is convenience; CI is redundancy.

3. **Hook enforcement is graded by violation class.**
   - **Blocking** (exit non-zero): edits to lines within a FROZEN Charter section without accompanying amendment; vocabulary drift (use of forbidden synonym for a canonical term); marketing language (terms on the `anti-marketing` watchlist).
   - **Advisory** (printed, exit zero): use of ambiguous terms on the `ambiguity-reducer` watchlist. The advisory is informational; the author decides.
   - `--no-verify` bypasses git enforcement; this is technical reality. The script always prints a notice stating that bypass is a registrable event and must be recorded in `decision-log.md` with justification. Silent bypass is a discipline failure; recorded bypass is not.

4. **Write permissions are gated via `ask`, not `deny`.** Edits to `docs/charter/constitutional-charter.md`, `docs/charter/amendments.md`, and anything under `docs/ontology/` pause for human confirmation in a Claude Code session. `docs/charter/in-committee/` (when it exists) is not gated — committee redaction is the expected work product. `docs/charter/decision-log.md` is not gated either: appends are legitimate and frequent. See `settings.json` for matcher syntax and the documented fallback if the matcher syntax is unavailable in the active Claude Code version.

5. **`CONTRIBUTING.md` and `WORKFLOW.md` are distinct documents.** `CONTRIBUTING.md` addresses contributors regardless of tooling. `WORKFLOW.md` addresses operators of the `.claude/` infrastructure. When they conflict, `CONTRIBUTING.md` wins by the precedence rule in §2.

6. **CI scope is bounded.** Three jobs (`doc-check`, `subordination-check`, `glossary-coverage`) run on PRs touching `*.md`. Deferred: RFC template compliance (premature — no RFC exists), Markdown link health (generic hygiene), mechanized falsifiability (judgment-bound; belongs to `epistemic-auditor`).

7. **Branch hygiene starts in Phase 2.** Phase 1 produced a single contract file on `main`. From this phase forward, each phase runs on its own branch (`claude/setup-phase-N`) and merges to `main` after review.

## 6. Operational rules for Claude in this repository

1. Do not edit any FROZEN Charter section. Edits to frozen sections are an amendment and must follow `docs/charter/amendments.md`.
2. Do not silently resolve any of the five open Ontology questions listed in `docs/ontology/ontology.md` under "Open Questions for Committee Resolution". A change that picks an answer is a constitutional move and must be raised explicitly.
3. Do not draft pending invariants outside committee mode. See the `invariant-redactor` skill.
4. Do not write implementation code (Go, Python, schemas, etc.) under any framing. Implementation is gated. See the `implementation-readiness-evaluator` skill for the gate criteria.
5. Do not introduce new vocabulary for concepts the canonical vocabulary already covers. Vocabulary drift is constitutional drift.
6. Do not write prose that cannot be checked against the source documents. If you cannot cite, do not assert.
7. Do not produce marketing language under any framing — including "vision prose", "executive summary", or "elevator pitch". The Charter is the pitch.

## 7. Constitutional minimalism

The system is required to remain compressible. Every new invariant must justify its non-redundancy with existing ones. Every new skill, command, agent, or hook must justify its non-overlap with existing infrastructure. Ceremony without behavioral consequence is rejected.

## 8. When in doubt

When in doubt, stop and ask. Do not proceed by guessing what the Charter would say.
