# `.claude/` Setup — Phase 1 Plan

**Status:** Phase 1 deliverable. Contract for Phases 2–8.
**Date:** 2026-05-14
**Charter reference version:** `v0.1` (Thesis frozen; Invariants 2.1 and 2.2 frozen; Invariants 2.3–2.6, §3 Non-Goals, §4 Constitutional Design Rule pending committee redaction).

This file is the only artifact produced in Phase 1. No other file in the repository is created or modified by this phase.

---

## Section 1 — Project state inventory

Declarative inventory. No recommendations.

### Charter status (`docs/charter/constitutional-charter.md`, `v0.1`)

| Section | Status |
|---|---|
| §1 — Thesis | Frozen |
| §2 — Constitutional Invariants (header and the four qualification criteria) | Frozen |
| §2.1 — Observational Integrity | Frozen |
| §2.2 — Epistemic Separation | Frozen |
| §2.3 — Provenance Integrity | Pending committee redaction; working definition non-binding |
| §2.4 — Inferential Influence Disclosure | Pending committee redaction; working definition non-binding |
| §2.5 — Hypothesis Lifecycle Explicitness | Pending committee redaction; working definition non-binding |
| §2.6 — Evidential Independence Integrity | Pending committee redaction; working definition non-binding |
| §3 — Non-Goals | Pending committee redaction; anticipated non-goals non-binding |
| §4 — Constitutional Design Rule | Pending committee redaction |

### Ontology document status

| Document | Status |
|---|---|
| `docs/ontology/ontology.md` | Scaffold (scope, document family, constitutional anchors, open questions, and status table drafted) |
| `docs/ontology/entity-model.md` | Scaffold |
| `docs/ontology/provenance-model.md` | Scaffold |
| `docs/ontology/lifecycle-semantics.md` | Scaffold |
| `docs/ontology/replay-semantics.md` | Not yet created (anticipated per `ontology.md` document family) |

### Five open Ontology questions (verbatim from `docs/ontology/ontology.md` §Open Questions for Committee Resolution)

1. Is `Session` a single entity with reconciliation, or two entities (`DeclaredSession` and `OperationalSession`)? Discussed in conversation but not yet decided.
2. Are `BehavioralCluster`, `CoordinationRing`, `CampaignHypothesis`, and `AutomationGroup` distinct entity types within the hypothesis category, or are they tags on a single `Hypothesis` type? Decision affects the schema surface.
3. What is the formal definition of `independence` as a measurable quantity? Conceptually agreed; operationally undefined.
4. When does a promoted hypothesis become a candidate for demotion? Lifecycle rule.
5. How does `influence` propagate through derived assertions? Transitive? Decaying? Both?

### Decision log entries (`docs/charter/decision-log.md`)

- `0001` — Adopt event-log immutability as constitutional invariant. Status: accepted.
- `0002` — Adopt ontological tripartition (observation / construct / hypothesis). Status: accepted.
- `0003` — Defer storage technology selection. Status: accepted.
- `0004` — Charter is authoritative; subordinate documents may evolve under implementation pressure. Status: accepted.

### Implementation status — `services/`

| Service | Declared status |
|---|---|
| `services/ingestion/` | Not implemented |
| `services/assertion-engine/` | Not implemented |
| `services/replay/` | Not implemented |
| `services/graph/` | Not implemented |
| `services/projections/` | Not implemented |

`services/README.md` records that implementation language is not yet decided and that implementation work begins after the Charter and the relevant portions of the Ontology are stable.

### Implementation status — `schemas/`

`schemas/README.md` records: "No concrete schemas exist yet. Schema definition is gated on committee redaction of the Ontology." The three subdirectories (`events/`, `assertions/`, `provenance/`) each declare "Not yet defined." Schema technology is not yet selected.

---

## Section 2 — Setup target

The final structure to be established by Phases 2–8:

```
.claude/
├── CLAUDE.md
├── README.md
├── PLAN.md                  (Phase 1 — this file)
├── settings.json
├── skills/
│   ├── constitutional/
│   │   ├── charter-guardian/
│   │   ├── subordination-checker/
│   │   └── invariant-redactor/
│   ├── epistemic/
│   │   ├── epistemic-separator/
│   │   ├── falsifiability-check/
│   │   └── ambiguity-reducer/
│   ├── ontology/
│   │   ├── ontology-keeper/
│   │   └── vocabulary-discipline/
│   ├── workflow/
│   │   ├── rfc-author/
│   │   ├── decision-logger/
│   │   └── implementation-readiness-evaluator/
│   └── enforcement/
│       └── anti-marketing/
├── agents/
│   ├── charter-reviewer.md
│   └── epistemic-auditor.md
├── commands/
│   ├── redact-invariant.md
│   ├── propose-rfc.md
│   ├── log-decision.md
│   ├── check-subordination.md
│   └── status.md
└── hooks/
    └── pre-commit-doc-check.sh
```

Outside `.claude/`:

- `WORKFLOW.md` (repository root)
- Updates to `CONTRIBUTING.md`
- `.github/workflows/constitutional-check.yml`
- `docs/glossary.md`

---

## Section 3 — Phase mapping

Phases are sequential. Each merges to `main` before the next begins. The decomposition below is proposed by Phase 1 and is the contract that the prompts driving Phases 2–8 are expected to execute against.

| Phase | Files created or modified | Files NOT to be touched | Acceptance criteria |
|---|---|---|---|
| **1 — Inventory and plan** | `.claude/PLAN.md` | Every other path. No file under `docs/`, `schemas/`, `services/`, `infra/`, `experiments/`, the repository root, or any other location inside `.claude/`. | The file exists. Section 1 contains no value-laden adjectives. Section 4 lists at least the five non-goals stated in the prompt. Section 5 questions are each answerable in one sentence. |
| **2 — Foundation** | `.claude/CLAUDE.md`, `.claude/README.md`, `.claude/settings.json`, `docs/glossary.md` | Anything under `.claude/skills/`, `.claude/agents/`, `.claude/commands/`, `.claude/hooks/`; the Charter; the Ontology; service and schema READMEs; existing root files (`README.md`, `CONTRIBUTING.md`, `.gitignore`). | `CLAUDE.md` declares subordination to the Charter and the Ontology in canonical vocabulary. `settings.json` permission scope is the minimum required for Phases 3–8. `docs/glossary.md` entries cite the Charter or the Ontology as source for every term. |
| **3 — Constitutional skills** | `.claude/skills/constitutional/charter-guardian/`, `.claude/skills/constitutional/subordination-checker/`, `.claude/skills/constitutional/invariant-redactor/` | All other skill directories; agents; commands; hooks; non-`.claude/` files. | Each skill declares its inputs and outputs. Each refers only to the frozen Charter sections (§1, §2 header, §2.1, §2.2) when citing authority. `invariant-redactor` cannot author content for §2.3–§2.6; it structures the committee redaction process. |
| **4 — Epistemic, ontology, and enforcement skills** | `.claude/skills/epistemic/epistemic-separator/`, `.claude/skills/epistemic/falsifiability-check/`, `.claude/skills/epistemic/ambiguity-reducer/`, `.claude/skills/ontology/ontology-keeper/`, `.claude/skills/ontology/vocabulary-discipline/`, `.claude/skills/enforcement/anti-marketing/` | Workflow skills; agents; commands; hooks; non-`.claude/` files. | `falsifiability-check` operationalizes the four qualification criteria in Charter §2 plus the falsifiability discipline in §4 (using §4's pending working text only as procedural reference, not as authority). `vocabulary-discipline` uses the canonical vocabulary list from `CONTRIBUTING.md` §Style and `docs/glossary.md`. `anti-marketing` operationalizes `CONTRIBUTING.md` §What This Project Is Not. |
| **5 — Workflow skills** | `.claude/skills/workflow/rfc-author/`, `.claude/skills/workflow/decision-logger/`, `.claude/skills/workflow/implementation-readiness-evaluator/` | Agents; commands; hooks; non-`.claude/` files. | `rfc-author` uses `docs/rfcs/template.md` as the authoritative template. `decision-logger` uses the decision-log format from `docs/charter/decision-log.md`. `implementation-readiness-evaluator` reads but does not advance the FROZEN/PENDING state of any Charter section. |
| **6 — Agents** | `.claude/agents/charter-reviewer.md`, `.claude/agents/epistemic-auditor.md` | Commands; hooks; non-`.claude/` files. Skills are read-only inputs at this phase. | Each agent composes skills established in Phases 3–5. Neither agent has authority to amend the Charter, the Ontology, or the decision log; both may only recommend RFCs or flag drift. |
| **7 — Commands** | `.claude/commands/redact-invariant.md`, `.claude/commands/propose-rfc.md`, `.claude/commands/log-decision.md`, `.claude/commands/check-subordination.md`, `.claude/commands/status.md` | Hooks; non-`.claude/` files. | Each command is invocable as `/<name>`. Each command's behavior is described in terms of which skill(s) and/or agent(s) it composes. `/status` reports current FROZEN/PENDING state by reading source documents at invocation time, not by recalling cached state. |
| **8 — Automation and human-facing integration** | `.claude/hooks/pre-commit-doc-check.sh`, `.github/workflows/constitutional-check.yml`, `WORKFLOW.md`, updates to `CONTRIBUTING.md` | The Charter; the Ontology; the decision log; the amendments log; `.claude/` artifacts created in Phases 1–7. | The hook and the CI workflow run the same checks. `WORKFLOW.md` cross-references `.claude/commands/` rather than duplicating their content. The `CONTRIBUTING.md` update is additive. Hook and CI enforcement posture (advisory vs. blocking) is set per the human's resolution of Section 5 question 3. |

---

## Section 4 — Non-goals of this setup

This `.claude/` infrastructure will not:

1. Resolve any of the five open Ontology questions listed in Section 1.
2. Select any technology — storage substrate, schema technology, projection technology, infrastructure orchestration, or implementation language.
3. Draft Charter Invariants §2.3, §2.4, §2.5, or §2.6; or §3 (Non-Goals); or §4 (Constitutional Design Rule). The working definitions in the Charter remain non-binding until committee redaction produces them.
4. Generate Ontology entities, concrete schemas, or service code.
5. Produce marketing copy under any framing. Per `CONTRIBUTING.md` §What This Project Is Not, prose that "sounds important but cannot be falsified" is rejected.
6. Modify the Charter, the amendments log, or the decision log by any path other than the formal amendment process defined in `docs/charter/amendments.md`.
7. Advance the status of any document from "scaffold" to "redacted." Status transitions are committee work; they are not setup work.
8. Confer decisional authority on any skill, agent, or command. These artifacts structure work; they do not approve, accept, or reject proposals.

---

## Section 5 — Open questions for the human

Questions that affect the setup itself and are not resolvable from the source documents read for Phase 1. Each is answerable in one sentence.

1. **Skill packaging convention.** Each skill in Section 2's target structure is shown as a directory; should it contain a `SKILL.md` (with optional supporting files), or should the skill be a single file named `<skill-name>.md` placed directly under the group directory?
2. **Hook scope.** Is `.claude/hooks/pre-commit-doc-check.sh` intended as a git `pre-commit` hook (installed via `core.hooksPath`) or as a Claude Code event hook configured in `settings.json`?
3. **Hook enforcement posture.** Should the documentation check block commits and CI when it detects vocabulary or subordination drift, or should it report advisory-only?
4. **`settings.json` permission scope.** Should the project require approval for writes to `docs/charter/`, `docs/ontology/`, `docs/charter/amendments.md`, and `docs/charter/decision-log.md`, or inherit the user's global Claude Code settings without project-level restriction?
5. **`WORKFLOW.md` boundary.** What is the distinct purpose of `WORKFLOW.md` relative to the existing `CONTRIBUTING.md` — operational workflows for maintainers using Claude Code in this repository, or something narrower?
6. **CI workflow scope.** Should `.github/workflows/constitutional-check.yml` run only documentation checks, or also vocabulary discipline, RFC template conformance, and link integrity?
7. **Branch hygiene for Phase 1.** The Rules of Operation instruct creating `claude/setup-phase-N` branches before each prompt; Phase 1 currently sits on `main` (clean) — should the PLAN be moved to `claude/setup-phase-1` before commit, or is `main` acceptable for this single-file deliverable?

These questions are to be resolved by the human in a commit message or as an addendum to this PLAN before Phase 2 begins.

---

## Section 6 — Resolutions to Section 5 (recorded 2026-05-14)

Resolved by the human in response to Section 5. Each resolution is binding for Phases 2–8.

### 6.1 — Skill packaging convention (resolves Q1)

Each skill is a directory containing `SKILL.md` plus optional `references/` material. Structural justification: several skills will accumulate content that does not fit in a single readable Markdown file — `vocabulary-discipline` carries an expanding forbidden-synonym table; `ambiguity-reducer` an extensible watchlist; `invariant-redactor` per-invariant redaction templates (one per pending invariant in §2.3–§2.6). A directory is also a more legible diff target than a monolithic file as content accretes through subsequent corrections. The reference pattern documented in skill-creator (`SKILL.md` + `references/`) is adopted as the project convention. Cost accepted: more files in the tree.

### 6.2 — Hook scope (resolves Q2)

Both, with distinct roles. They are not alternatives.

- **Git pre-commit hook** (via `git config core.hooksPath .claude/hooks/` plus a `pre-commit` entry that invokes the shared script) is the enforcement surface. It triggers on every commit, regardless of author or tool, and is the only surface that protects the repository against contributors who are not running Claude Code.
- **Claude Code event hook** (configured in `settings.json`) is the feedback surface. It triggers in-session for early correction, inside the working loop.

The script `.claude/hooks/pre-commit-doc-check.sh` is the shared core; both invocations and the CI workflow run the same script. When the surfaces drift, the git side is authoritative — it is what protects the repository at PR review.

### 6.3 — Hook enforcement posture (resolves Q3)

Tiered by category. The distinction is constitutional.

| Category | Behavior |
|---|---|
| Edit to lines inside a FROZEN Charter section without an accompanying amendment | **Block.** Frozen content is amendment by definition. |
| Vocabulary drift (canonical term replaced by a non-canonical synonym from the declarative watchlist) | **Block** with explanatory message. Watchlist revisions go through RFC, not casual bypass. |
| Marketing language (matches against an explicit tells list) | **Block** with the matched tells shown to the author for reformulation. |
| Ambiguity flag (terms such as `context`, `state`, `trust` used without local operationalization) | **Advisory.** The flag is informational; the author decides. |

`git commit --no-verify` cannot be technically prevented but is treated as an event: every run of the hook prints, on every path, a notice instructing the author that bypass must be recorded as a note in the decision log with justification. Silent bypass is a discipline failure; recorded bypass is not.

Explicit rejection: a "warning-only everywhere to avoid friction" posture is refused. The Charter enforces observational integrity by structural guarantee, not by convention; the infrastructure that sustains the Charter carries the same discipline.

### 6.4 — `settings.json` permission scope (resolves Q4)

Gate via `ask`, not `deny`. Writes and edits to `docs/charter/constitutional-charter.md`, `docs/charter/amendments.md`, and the entirety of `docs/ontology/**` pause the session for human confirmation. `Bash` is `ask` because it bypasses path matchers. Read-only tools (`Read`, `Glob`, `Grep`, `WebFetch`) are `allow`. The decision log is intentionally not gated; legitimate appends to it are frequent.

When `docs/charter/in-committee/` is created, it inherits the global `allow` for write tools — edits there are committee redaction, not amendment.

Phase 2 verification requirement: the `Write(path)` matcher syntax and glob support may vary by Claude Code version. If the proposed syntax fails, the fallback is `ask: ["Write", "Edit"]` globally, with the loss of granularity documented in a comment in `settings.json`.

### 6.5 — `WORKFLOW.md` boundary (resolves Q5)

Distinct audience, distinct scope.

| Document | Audience | Question answered |
|---|---|---|
| `CONTRIBUTING.md` | External contributor (PR author) | "How do I propose a change?" — process, independent of tooling. |
| `WORKFLOW.md` | Operator of the `.claude/` infrastructure | "How does the tooling enforce what `CONTRIBUTING.md` describes?" |

`CONTRIBUTING.md` remains authoritative on process and is closer to the Charter in the document hierarchy. When the two conflict, `CONTRIBUTING.md` prevails and `WORKFLOW.md` is revised. `WORKFLOW.md` describes infrastructure (`.claude/commands/`, hook behavior, settings posture, what to do when a skill vetoes a change) and must not duplicate process content.

The human will send a `WORKFLOW.md`-specific adjustment to the prompt that produces it, making this boundary explicit beyond what is stated here.

### 6.6 — CI workflow scope (resolves Q6)

Three jobs, each detecting a class of drift the others do not. Each is mechanical (no judgment).

1. **`doc-check`** — invokes the shared hook script.
2. **`subordination-check`** — verifies cross-references from Ontology and Architecture documents back to the Charter.
3. **`glossary-coverage`** — verifies every canonical term cited in `CLAUDE.md` §3 (and elsewhere in `.claude/`) has an entry in `docs/glossary.md` with a Charter or Ontology source citation.

Explicitly deferred, with reason:

- **RFC template compliance check.** Premature; no RFC exists. A check that never fires is a check whose first failure no one will understand. Added when the first RFC is proposed.
- **Broken Markdown link check.** Useful but generic and outside the constitutional scope. May be added later without RFC because it does not touch the substrate.
- **Falsifiability check in CI.** Not mechanical; depends on judgment. Belongs to `epistemic-auditor` (agent) with a human in the loop. Mechanizing it would be tooling-as-pretense.

Principle: CI checks what `grep` and list comparison can verify. Everything else is an agent or a skill.

### 6.7 — Branch hygiene for Phase 1 (resolves Q7)

`main` is accepted for Phase 1. The deliverable is a single contract file that does not touch infrastructure or code, and the artifact does not need to be reviewed against itself. Phase 1 commits to `main` with the message:

```
chore(claude): add operational setup plan (Phase 1)
```

From Phase 2 onward, each phase branches as `claude/setup-phase-N` and merges to `main` after review. This is not a precedent for "main for small things"; it is specific to Phase 1's meta status.

---

### Note on phase-numbering alignment

The human's prompt-numbering (referenced as "Prompt 7", "Prompt 8" in earlier messages) may differ from the phase-numbering proposed in Section 3 of this PLAN. References to "Phase N" in Section 6 use this PLAN's numbering. When the Phase 2 prompt arrives, mismatches between the human's decomposition and Section 3 will be reconciled — the human's authoritative decomposition prevails.
