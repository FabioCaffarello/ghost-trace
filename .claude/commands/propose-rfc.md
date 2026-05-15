<!--
Slash command. $ARGUMENTS substitution per Claude Code convention.
If the active version does not parse this format, the prompt body is
valid as a manual prompt with the working title stated explicitly.
-->
---
description: Create an RFC draft from the canonical template, with the six-question pre-authorship impact analysis done before any section beyond the title is filled.
argument-hint: <working-title>
---

# /propose-rfc $ARGUMENTS

Create an RFC draft titled `$ARGUMENTS`.

## Step 1 — Invoke `rfc-author`

Read `.claude/skills/workflow/rfc-author/SKILL.md` and apply its full procedure.

## Step 2 — Establish the draft file

- Compute `<slugified-title>` from `$ARGUMENTS` (lowercase, hyphenated, alphanumeric only).
- Create `docs/rfcs/draft/<slugified-title>.md` from `docs/rfcs/template.md`.
- If a file with the same slug already exists, refuse and surface the conflict. Do not overwrite.

The draft file is initially populated with:

- The title from `$ARGUMENTS`.
- `Status: draft`.
- `Date: <today>`.
- `Authors: <ask the user>`.
- `Type: standard` (unless the impact analysis below reclassifies it).
- `Affects: <pending after impact analysis>`.
- All other template sections present but **empty**, marked with `<!-- pending: ... -->` placeholders.

## Step 3 — Pre-authorship impact analysis BEFORE filling sections

Apply the six questions from `rfc-author` §1 (Q1–Q6) **before** writing into any section of the RFC beyond the title block:

- **Q1.** Which Charter invariants does this RFC touch? (delegate to `charter-guardian`)
- **Q2.** Does this RFC implicitly redefine any term in `docs/glossary.md`? (delegate to `vocabulary-discipline`)
- **Q3.** Does this RFC implicitly resolve any of the five open Ontology questions? (delegate to `ontology-keeper`)
- **Q4.** Does this RFC require Charter amendment? (delegate to `charter-guardian`; if yes, change `Type` to `charter-amendment` and apply the amendment process)
- **Q5.** Does this RFC introduce a new invariant? If yes, justify non-redundancy per CLAUDE.md §7.
- **Q6.** Does this RFC propose ceremony without behavioral consequence? If yes, recommend non-adoption.

The output of Q1–Q6 becomes the seed of the RFC's `Constitutional Review` section. It is written verbatim, not paraphrased.

## Step 4 — Fill in only what the user has decided

Walk through each remaining template section: Summary, Motivation, Constitutional Review (use Q1–Q6 output), Proposal, Alternatives Considered, Open Questions, Anti-Patterns to Avoid, Migration and Backward Compatibility, References, Decision Record.

For each section, ask the user what they want recorded. Use `<!-- pending: <hint> -->` placeholders for sections the user has not decided.

**Do not invent content.** A draft with `<!-- pending -->` markers is honest about its state. A draft with manufactured prose is dishonest.

## Step 5 — Apply discipline before `status: discussion`

When the user is ready to mark the draft `status: discussion`:

1. Apply `falsifiability-check` to every claim in the draft.
2. Apply `epistemic-separator` to every paragraph.
3. Apply `ambiguity-reducer` to every noun.
4. Apply `anti-marketing` to the Summary section specifically.

A draft that fails any of these is rewritten, not marked.

## Constraints

- Do not number the RFC. Drafts in `docs/rfcs/draft/` carry working titles only. Numbering happens at acceptance per `rfc-author` §4.
- Do not invent content for sections the user has not decided.
- Do not skip the impact analysis. It is the entry point.
- The Summary section is the most marketing-prone; run `anti-marketing` on it carefully.
