<!--
Slash command for Claude Code. Format: YAML frontmatter + prompt body.
$ARGUMENTS substitution per the Claude Code slash-command convention.
If the active Claude Code version does not parse this format, the prompt
body below remains valid as a manual prompt.
-->
---
description: Report the project's constitutional state — Charter status, latest decision, latest amendment, Ontology status, open Ontology questions, RFC drafts, in-committee sections.
---

# /status

Output a single-page report of the project's current constitutional state. **No prose between sections.** Structured.

## Reads (in this order)

1. `.claude/CLAUDE.md` §4 — Charter status table.
2. `docs/charter/decision-log.md` — find the highest-numbered entry; record its ID, title, status, date.
3. `docs/charter/amendments.md` — find the most recent amendment entry; record its version, date, sections affected.
4. `docs/ontology/ontology.md` — the Status table at the bottom of the document.
5. `.claude/skills/ontology/ontology-keeper/SKILL.md` §1 — the open-question registry maintained by the skill.
6. `docs/ontology/ontology.md` §Open Questions for Committee Resolution — the source of truth for the same questions.
7. `docs/rfcs/draft/` — list directory contents.
8. `docs/charter/in-committee/` — if directory exists, list contents; otherwise note absence.

## Output sections (in this order, no narrative between)

### Charter status

Mirror the table from `.claude/CLAUDE.md` §4 verbatim.

### Latest decision

`ID — Title (status, date)`.

### Latest amendment

`version — title (date) — sections affected`.

### Ontology status

Mirror the Status table from `docs/ontology/ontology.md` verbatim.

### Open Ontology questions

List the questions that remain unresolved.

If the `ontology-keeper` registry has drifted from `docs/ontology/ontology.md` (questions appear in one but not the other, or text differs), surface the drift as a finding prefixed `DRIFT:` — name which source has what.

### RFC drafts

For each file in `docs/rfcs/draft/`, list filename and `Status:` field value. If the directory is empty, state `(no drafts)`.

### In-committee sections

If `docs/charter/in-committee/` exists, list its files. If the directory does not exist, state `(no in-committee work)`.

## Constraints

- Read source files at invocation time. Do not recall cached state from prior sessions.
- Do not interpret the state. Report it.
- No marketing language. No commentary.
- If any source file is missing or unreadable, name it explicitly as a finding rather than producing a partial report silently.
