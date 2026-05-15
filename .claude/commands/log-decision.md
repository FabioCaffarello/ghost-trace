<!--
Slash command. No arguments. The command interviews the user for each
field; no field is invented.
-->
---
description: Append a new entry to docs/charter/decision-log.md, sequential ID determined from the file (not user-supplied), with append-only discipline.
---

# /log-decision

Append a new decision-log entry.

## Step 1 — Invoke `decision-logger`

Read `.claude/skills/workflow/decision-logger/SKILL.md` and apply its full procedure.

## Step 2 — Determine the next sequential ID

Read `docs/charter/decision-log.md`. Find the highest existing entry ID (entries are `## \`00NN\` — Title`). The new ID is the next integer, zero-padded to four digits.

**Do not trust a user-supplied ID.** The file is the source of truth.

## Step 3 — Interview the user for each required field

For each template field, ask the user. Do not invent any value.

- **Title.** Short, declarative, in canonical vocabulary. Apply `anti-marketing`.
- **Status.** Must be one of `proposed`, `accepted`, `superseded`, `rejected`. Ask.
- **Date.** Today's date in `YYYY-MM-DD`.
- **Context.** What made the decision necessary. Cite prior state, the constraint, or the originating RFC. Apply `anti-marketing` to the prose.
- **Decision.** What was decided. One paragraph, concrete. Apply `falsifiability-check` to every claim.
- **Constitutional review.** Must list every Charter invariant the decision was tested against (§2.1, §2.2, and any pending invariants whose working text the decision touches). Vague "consistent with the Charter" is rejected — demand specifics. Apply `falsifiability-check`.
- **Consequences.** Both what is enabled AND what is now constrained or excluded. If the user has only stated what is enabled, ask explicitly what is now excluded. Apply `falsifiability-check`.
- **Supersession.** `none` for original decisions; pointer to the superseding entry's ID when superseded. Ask whether this entry supersedes a prior one.

Apply `vocabulary-discipline` on every term across all fields.

## Step 4 — If this entry supersedes a prior one

Follow the supersession procedure from `decision-logger` §3:

1. The new entry is authored as a normal new entry (this step's product).
2. Edit the superseded entry's `Status` field to `superseded`. Add the new entry's ID to its `Supersession` field. **No other edit to the prior entry.**
3. Verify bidirectional pointers: new entry's `Supersession` field names the prior ID; prior entry's `Supersession` field names the new ID.

## Step 5 — Append

Append the new entry to `docs/charter/decision-log.md`. Use the template at the bottom of the file as the structural skeleton.

**Do not modify any prior entry beyond the status-line discipline in Step 4.**

## Constraints

- Refuse to backdate (use today's date).
- Refuse to insert entries between existing ones (IDs are append-only and sequential).
- Refuse to edit the text of any prior entry. Corrections to prior decisions happen by supersession (§4), not by in-place edit.
- Editorial fixes (typo in a prior entry) require their own decision-log entry that names what was fixed. The Charter's editorial-fix carve-out (`amendments.md` §Amendment Discipline) covers the Charter only, not the decision log.
- This command does not approve decisions. It records them. Approval is human committee work or RFC acceptance.
