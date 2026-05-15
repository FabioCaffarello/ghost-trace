# Ghost Trace — Workflow

Operational guide for working in this repository with or without Claude Code. This document covers the tooling under `.claude/`. The contribution process — what kinds of changes are welcome, how to propose them, the constitutional review they undergo — lives in [`CONTRIBUTING.md`](./CONTRIBUTING.md).

## 1. Document role and precedence

This document describes the **tooling** that supports work on Ghost Trace. It is tooling-agnostic in scope (the discipline applies regardless of editor) but specific in mechanism (it describes the `.claude/` infrastructure and the CI workflow).

[`CONTRIBUTING.md`](./CONTRIBUTING.md) is the authoritative document on **process** — how to propose changes, how the amendment process works, what kinds of contributions are welcome. It is independent of any editor or tool.

**When this document conflicts with `CONTRIBUTING.md`, `CONTRIBUTING.md` wins.** Tooling is convenience; process is constitutional. This precedence rule is established in [`.claude/CLAUDE.md` §2](./.claude/CLAUDE.md) and follows from the document hierarchy in [`README.md`](./README.md).

## 2. What `.claude/` is and is not

`.claude/` contains operational infrastructure for working on Ghost Trace with Claude Code — skills, slash commands, subagents, hooks, and the `CLAUDE.md` context file loaded into every session.

**It is not part of the constitutional surface.** The Constitutional Charter (`docs/charter/constitutional-charter.md`) is authoritative; the Ontology subordinate to it; the architecture subordinate to both. `.claude/` is scaffolding around them.

If a skill says one thing and the Charter says another, the Charter wins and the skill is revised. If a hook blocks an edit the Charter would permit, the hook is buggy, not the Charter.

## 3. Three-tier hook architecture

The same documentation-discipline check runs in three contexts (per [`.claude/CLAUDE.md` §5.2](./.claude/CLAUDE.md)). All three invoke the same script: `.claude/hooks/pre-commit-doc-check.sh`.

### 3.1 Git pre-commit hook (enforcement)

The **enforcement boundary**. Runs on every commit, regardless of who or what produced the commit. Protects the repository against contributors who do not use Claude Code.

Activate in a fresh clone with two commands:

```sh
git config core.hooksPath .claude/hooks
ln -sf pre-commit-doc-check.sh .claude/hooks/pre-commit
```

The symlink names the script `pre-commit` (the filename git looks for), pointing to the actual script. Verify with `ls -l .claude/hooks/pre-commit`.

On Windows, symlinks require Developer Mode or admin privileges. As an alternative, use `cp` instead of `ln -sf` — the cost is that `pre-commit` becomes stale when the script updates and must be re-copied.

The script is already executable (`chmod +x` was set when the file was committed).

### 3.2 Claude Code event hook (feedback)

Surfaces issues during the session before commit. Wired automatically via [`.claude/settings.json`](./.claude/settings.json) (the `Stop` event invokes the same script). No operator action required.

This is convenience, not enforcement. A Claude Code user sees the script's output at the end of each turn; a non-Claude user does not.

### 3.3 CI workflow (PR-level enforcement)

Defined in [`.github/workflows/constitutional-check.yml`](./.github/workflows/constitutional-check.yml). Three jobs run on PRs touching `*.md` files:

- `doc-check` — runs the same hook script (`--self-test` first, then the diff check).
- `subordination-check` — verifies that every Charter cross-reference in changed files resolves to a real anchor.
- `glossary-coverage` — verifies every canonical-vocabulary term has an entry in `docs/glossary.md` with the four provenance fields.

The CI workflow runs the hook's `--self-test` **before** the diff check. A malformed watchlist or unparseable `CLAUDE.md` §4 table fails CI even on commits that would otherwise pass.

### 3.4 Source of truth when surfaces diverge

If the git hook, the Claude event hook, and CI disagree, **git wins**. The Claude event hook is convenience; CI is redundancy. Git is the actual enforcement boundary that runs on every commit.

## 4. How to use this without Claude Code

The skills under `.claude/skills/` are written in declarative English. They describe rules — what is forbidden, what is required, what the procedure is. A contributor using any editor can read them as authoring guidelines.

The git pre-commit hook enforces the mechanical rules (vocabulary drift, marketing tells, frozen-section edits) at commit time without any Claude Code involvement. The CI workflow enforces them again at PR time.

The skills' procedural sections (e.g., the nine-step redaction procedure in `invariant-redactor`) are followed manually by the contributor. The discipline is the same; only the assistance differs.

## 5. How to use this with Claude Code

`CLAUDE.md` is loaded automatically into every session. Read it once; you will not need to re-read it. It carries the canonical vocabulary, the Charter status table, the seven governance decisions, and the seven operational rules.

**Skills** trigger based on their descriptions. Claude consults them when working on the documents they address. They are not invoked by name.

**Slash commands** are invoked manually:

- `/status` — report current Charter state, latest decision, latest amendment, Ontology status, open Ontology questions, RFC drafts, in-committee sections.
- `/redact-invariant <§>` — initiate committee-mode redaction of a pending Charter section.
- `/propose-rfc <title>` — create an RFC draft from the canonical template.
- `/log-decision` — append a new entry to `docs/charter/decision-log.md`.
- `/check-subordination [path]` — directional rank-order conflict check.

**Subagents** are invoked for second-pair-of-eyes review of significant changes:

- `charter-reviewer` — structured review of any change touching `docs/charter/`.
- `epistemic-auditor` — prose audit across the four epistemic skills with cross-skill linkage.

**Hooks** run at commit time per §3 above.

## 6. Common workflows

Each procedure ends by pointing at the relevant skill or agent for deeper detail. Do not duplicate skill content; consult the source.

### 6.1 Drafting an RFC

1. Decide the working title.
2. Run `/propose-rfc <working-title>`. The command creates `docs/rfcs/draft/<slug>.md` and runs the six-question impact analysis (Q1–Q6) before any section beyond the title is filled.
3. Fill sections the user has decided on; leave `<!-- pending -->` placeholders for the rest. Do not invent content.
4. When the draft is ready for discussion, apply `falsifiability-check`, `epistemic-separator`, `ambiguity-reducer`, and `anti-marketing` to the Summary. Mark `status: discussion` only after all four pass.
5. RFC numbering happens at acceptance, not at draft. See [`rfc-author`](./.claude/skills/workflow/rfc-author/SKILL.md) §4.

### 6.2 Redacting a pending invariant

1. Verify the section is PENDING (check [`.claude/CLAUDE.md` §4](./.claude/CLAUDE.md)).
2. Run `/redact-invariant <§>` (e.g., `/redact-invariant §2.3`).
3. The command creates `docs/charter/in-committee/<section>-<name>.md` and walks the nine-step redaction procedure from [`invariant-redactor`](./.claude/skills/constitutional/invariant-redactor/SKILL.md) §2.
4. Do not edit `docs/charter/constitutional-charter.md` directly. The Charter is updated only at the final merge of the redaction.
5. The merge requires all five items in the final checklist (qualification criteria; structural parallel; glossary entries; decision-log entry; amendments-log entry + version bump).

### 6.3 Logging a decision

1. Run `/log-decision`. The command determines the next sequential ID from the file (not from a user-supplied value).
2. Provide each field when prompted. `Constitutional review` must list every Charter invariant the decision was tested against; `Consequences` must include both what is enabled and what is now constrained.
3. If the decision supersedes a prior entry, the command performs the bidirectional cross-link in the existing entry's status line (no other edit to the prior entry).
4. Editorial fixes to prior decision-log entries require their own new entry; in-place edits are not permitted. See [`decision-logger`](./.claude/skills/workflow/decision-logger/SKILL.md) §4.

### 6.4 Updating the glossary

1. Determine whether the term is new (no existing entry) or refined (existing entry's definition is being changed).
2. **New term:** add the entry to [`docs/glossary.md`](./docs/glossary.md) with all five fields filled or marked `pending`. Record the introduction as a new decision-log entry unless the term is introduced by a Charter section (in which case the Charter section is the introduction).
3. **Refined term:** silent refinement is forbidden. Raise an `ontology-revision` RFC (or `charter-amendment` if the term is constitutional). See [`vocabulary-discipline`](./.claude/skills/ontology/vocabulary-discipline/SKILL.md) §2.
4. After the glossary is updated, the forbidden-synonym table in `vocabulary-discipline` §4 may need a parallel update. Both views must agree.

### 6.5 Reviewing a Charter change

1. Invoke the `charter-reviewer` subagent on the PR.
2. The agent classifies the change (editorial / redaction / amendment), applies `charter-guardian` for the procedural envelope, applies the three epistemic skills for content, and verifies accompanying entries (RFC, decision log, amendments log, glossary, status table).
3. The agent's verdict is one of `PASS`, `PASS-WITH-REVISIONS`, or `BLOCK`. The agent does not approve in its own name; it produces a structured review for human committee approval. See [`agents/charter-reviewer.md`](./.claude/agents/charter-reviewer.md).

## 7. When Claude Code blocks you

Skills are designed to block silent constitutional drift. If a skill blocks a change you believe is correct, the disagreement is itself a constitutional signal: **escalate it as an RFC, not as a bypass.**

The path to changing what skills block is the same path that changes everything else in this project — a proposal, reviewed against the Charter. A skill's behavior cannot be revised by conversation, and the criteria the skill enforces cannot be softened by appeal. See [`implementation-readiness-evaluator`](./.claude/skills/workflow/implementation-readiness-evaluator/SKILL.md) §5 for the non-negotiability principle.

Mechanical bypass (`git commit --no-verify`) is technically possible — git cannot prevent it. It is a **registrable event** per [`.claude/CLAUDE.md` §5.3](./.claude/CLAUDE.md): the bypass must be noted in [`docs/charter/decision-log.md`](./docs/charter/decision-log.md) with justification. The pre-commit hook prints this reminder on every run, regardless of pass or fail.

Silent bypass is a discipline failure. Recorded bypass is not.
