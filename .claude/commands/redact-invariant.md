<!--
Slash command. $ARGUMENTS substitution per Claude Code convention.
If the active version does not parse this format, the prompt body is
valid as a manual prompt with the section number stated explicitly.
-->
---
description: Initiate committee-mode redaction of a pending Charter section. Argument is the section number (§2.3, §2.4, §2.5, §2.6, §3, or §4).
argument-hint: <section-number>
---

# /redact-invariant $ARGUMENTS

Initiate redaction of Charter section `$ARGUMENTS`.

## Step 1 — Verify status

Read `.claude/CLAUDE.md` §4 — Charter status table.

- If `$ARGUMENTS` is FROZEN: refuse. Editing a frozen section is an amendment by definition. Direct the user to `docs/charter/amendments.md` for the amendment process and to the `charter-guardian` skill for the procedural envelope. Do not proceed.
- If `$ARGUMENTS` is not in the status table: refuse. The section is not a recognized element. Do not proceed.
- If `$ARGUMENTS` is PENDING: proceed.

## Step 2 — Invoke `invariant-redactor`

Read `.claude/skills/constitutional/invariant-redactor/SKILL.md` and apply its full procedure for section `$ARGUMENTS`. The skill's §2 (Redaction procedure, nine steps) is followed in order.

## Step 3 — Establish the scratch document

- Create `docs/charter/in-committee/` if it does not exist.
- Determine the scratch filename: `<section>-<short-name>.md` where `<section>` is `$ARGUMENTS` (e.g., `§2.3`) and `<short-name>` is the kebab-cased section title from `docs/charter/constitutional-charter.md`. Example: `§2.3-provenance-integrity.md`.
- If the scratch file already exists, do not overwrite. Append a new revision marker block and continue from the existing content.

## Step 4 — Walk the redaction procedure

Apply steps 1–9 of `invariant-redactor` §2 in order:

1. Anchor in the working-definition stub from the Charter.
2. Produce the candidate in the scratch document.
3. Apply `falsifiability-check` to every claim.
4. Apply `epistemic-separator` to every paragraph.
5. Apply `ambiguity-reducer` to every noun.
6. Identify forbidden anti-patterns by analogy to §2.1 and §2.2.
7. Identify boundary conditions.
8. Identify non-duplication with existing invariants.
9. Surface for human review.

**Do not edit `docs/charter/constitutional-charter.md` directly under any circumstance.** Direct edits to a PENDING element of the Charter are blocked by `charter-guardian`.

## Step 5 — Output the candidate and the merge gates

At the end, output:

1. The current candidate redaction text (full).
2. The five-item final merge checklist from `invariant-redactor` §3, with each item annotated `MET` or `UNMET` based on the current state of the change set:
   - Four qualification criteria met.
   - Section structure parallels §2.1 and §2.2.
   - Vocabulary in the glossary (every load-bearing term has an entry).
   - Decision-log entry recorded.
   - Amendments-log entry and version bump.

The candidate is not merged automatically. Surface to the human for committee approval.

## Constraints

- Do not skip Step 1.
- Do not edit `constitutional-charter.md` directly.
- Report the five-item checklist even when items are unmet — naming what is unmet is the value of the report.
- This command does not approve the candidate. Approval is human committee work.
