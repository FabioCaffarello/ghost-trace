# `.claude/` — Operational Scaffolding for Ghost Trace

This directory contains the operational infrastructure that supports work on the Ghost Trace repository when Claude Code is the working tool. It does not contain substantive project content. The Charter is the substance; everything here is scaffolding around it.

## Read order for someone unfamiliar with this directory

1. [`CLAUDE.md`](./CLAUDE.md) — always loaded into every Claude Code session in this repository. Constitutional context, canonical vocabulary, Charter status, governance decisions, operational rules.
2. [`README.md`](./README.md) — this file. How the directory is organized.
3. [`PLAN.md`](./PLAN.md) — the contract for the eight-phase setup. Project state inventory and phase mapping.
4. `SELF-AUDIT.md` — once it exists (Phase 8). Periodic check that the infrastructure remains consistent with itself.

## How the directory is organized

### `CLAUDE.md`

Always-loaded context. Loaded by Claude Code at the start of every session in this repository. Bounded so it does not displace the working task. The source of truth for canonical vocabulary, Charter status at a glance, and the seven governance decisions binding this infrastructure.

### `settings.json`

Per-project Claude Code configuration. Currently governs:

- **Permission posture.** Read tools (`Read`, `Glob`, `Grep`, `WebFetch`) are pre-approved. Writes to `docs/charter/constitutional-charter.md`, `docs/charter/amendments.md`, and `docs/ontology/**` pause for human confirmation (`ask`). `Bash` is `ask`. Other paths inherit non-`ask` behavior. See `CLAUDE.md` §5.4 for rationale.
- **Claude event hook.** The `Stop` event invokes the shared doc-check script at the end of each turn. See `CLAUDE.md` §5.2 for the three-tier hook architecture.

Deviations from the Phase 2 prompt example are documented at the top of `settings.json`.

### `skills/`

Skills are directories containing `SKILL.md` and optional `references/` material (per `CLAUDE.md` §5.1). They are grouped into five domains:

- **`constitutional/`** — `charter-guardian`, `subordination-checker`, `invariant-redactor`. Protect the Charter from drift.
- **`epistemic/`** — `epistemic-separator`, `falsifiability-check`, `ambiguity-reducer`. Preserve clarity of language.
- **`ontology/`** — `ontology-keeper`, `vocabulary-discipline`. Keep modeling work aligned with the Charter.
- **`workflow/`** — `rfc-author`, `decision-logger`, `implementation-readiness-evaluator`. Structure recurring procedural work.
- **`enforcement/`** — `anti-marketing`. Reject forbidden styles.

Skill content is authored in Phases 3–5 (per `PLAN.md` §3). The directories exist in Phase 2 as `.gitkeep` placeholders.

### `commands/`

Slash commands invokable as `/<name>` inside a Claude Code session. Each command composes one or more skills to perform a procedural task:

- `/redact-invariant` — structure committee redaction of a pending invariant.
- `/propose-rfc` — draft an RFC against `docs/rfcs/template.md`.
- `/log-decision` — append to `docs/charter/decision-log.md` following the existing format.
- `/check-subordination` — verify cross-references in subordinate documents back to the Charter.
- `/status` — report current FROZEN/PENDING status by reading source documents.

Commands are authored in a later phase per the planned decomposition. The directory exists in Phase 2 as `.gitkeep`.

### `agents/`

Specialized review roles invokable as Claude Code subagents:

- `charter-reviewer` — reads proposed changes and reports interaction with each Charter invariant.
- `epistemic-auditor` — reads documents for falsifiability, ambiguity, and inferential discipline.

Neither agent has authority to amend the Charter, the Ontology, or the decision log. They recommend RFCs or flag drift. Authored in a later phase.

### `hooks/`

Home of the shared script `pre-commit-doc-check.sh`, which runs in three contexts per `CLAUDE.md` §5.2:

- As a git `pre-commit` hook (via `git config core.hooksPath .claude/hooks/`). Enforcement-of-record; the actual boundary that protects the repository against any committer.
- As a Claude Code event hook (configured in `settings.json`). In-session feedback.
- As a CI workflow step (Phase 7). PR-level enforcement.

If the three diverge, the git hook is the source of truth. The Claude event hook is convenience; CI is redundancy. Script authored in Phase 7.

## What this directory is not

This directory does not contain the project's substantive content. The Charter is the substance. The Ontology, the architecture documents, the RFCs, the schemas, and the services are the project. This directory is operational scaffolding around them.

This directory does not have decisional authority. Skills, commands, and agents structure work; they do not approve, accept, or reject proposals. Decisions are recorded in `docs/charter/decision-log.md` by humans through the formal process.

This directory must remain compressible. Per `CLAUDE.md` §7, every new skill, command, agent, or hook must justify its non-overlap with existing infrastructure. Ceremony without behavioral consequence is rejected.

## Step-by-step procedures

For operator-facing step-by-step procedures (how to propose an amendment, how to redact an invariant, how to log a decision, what to do when a skill vetoes a change), see [`WORKFLOW.md`](../WORKFLOW.md) at the repository root.

## CI workflow

The constitutional-check workflow ([`.github/workflows/constitutional-check.yml`](../.github/workflows/constitutional-check.yml)) runs on pull requests touching `*.md` files. Three jobs, all mechanical:

- **`doc-check`** — invokes `hooks/pre-commit-doc-check.sh --self-test` then the same script as a diff check (via `git reset --soft origin/<base>` to restage PR changes). Findings are posted as a PR comment regardless of pass/fail.
- **`subordination-check`** — for each changed file under `docs/ontology/`, `docs/architecture/`, `docs/rfcs/`, `schemas/`, `services/`, verifies that every reference to `docs/charter/constitutional-charter.md#<anchor>` resolves to an actual heading in the Charter. Mechanical anchor extraction; the semantic conflict check belongs to `skills/constitutional/subordination-checker/` and the human reviewer.
- **`glossary-coverage`** — for each canonical term in `CLAUDE.md` §3, verifies an entry exists in `docs/glossary.md` with the four provenance fields (definition, introduction, stabilization, last amendment). Fields may be `pending` but must be present.

The CI runs the same script the local git pre-commit hook runs, plus the two cross-file checks (subordination cross-references, glossary coverage) that require traversal across multiple files and are not amenable to per-file grep.

Scope is bounded per `CLAUDE.md` §5.6. RFC template compliance, link-health checks, and mechanized falsifiability are out of scope.
