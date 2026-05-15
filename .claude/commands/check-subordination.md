<!--
Slash command. Optional path argument; if empty, checks all of
docs/ontology/, docs/architecture/, docs/rfcs/, schemas/, services/.
-->
---
description: Run a subordination check across the repository or a given path. Reports potential conflicts between lower-ranked and higher-ranked documents. Read-only — does not auto-fix.
argument-hint: [path]
---

# /check-subordination $ARGUMENTS

Run the directional subordination check.

## Step 1 — Invoke `subordination-checker`

Read `.claude/skills/constitutional/subordination-checker/SKILL.md` and apply its full procedure.

## Step 2 — Determine scope

- If `$ARGUMENTS` is non-empty, restrict the check to that path (single file or directory).
- If `$ARGUMENTS` is empty, check all of:
  - `docs/ontology/`
  - `docs/architecture/`
  - `docs/rfcs/`
  - `schemas/`
  - `services/`

The Charter (`docs/charter/`) is excluded — it has its own discipline via `charter-guardian`.

## Step 3 — Enumerate and classify

For each file in scope:

1. Enumerate claims. A claim is any sentence asserting a property of the system, an entity, a relation, a behavior, or a constraint. Sentences that merely repeat the higher document do not count as new claims.
2. For each claim, locate the highest-ranked governing document (search Charter → Ontology → Architecture; do not search lower than the document being checked).
3. Classify the claim against the governing document: **contradiction**, **extension**, or **repetition**.

## Step 4 — Report

For each detected potential conflict, output:

- File and line number.
- The lower-document claim (verbatim).
- The higher-document claim it conflicts with (verbatim or with section reference).
- The classification (always `contradiction` for the report).
- The recommended resolution.

The recommended resolution is **always**: revise the lower document. If the lower document names a property the higher document genuinely needs, the resolution path is upward — an RFC of type `charter-amendment` (constitutional property), `ontology-revision` (ontological property), or `architecture` (architectural property). The lower document is never edited to assert the property unilaterally.

Extensions and repetitions are not reported as conflicts. Repetitions may be flagged separately as a watchpoint — they drift on the next governing-document revision.

## Constraints

- **Do not auto-fix.** Report only. Resolution requires human review.
- Do not edit any file. The output is the report.
- Do not interpret the Charter — that's `charter-guardian`'s scope.
- Do not extrapolate intent. The higher document says what it says; the lower must align with that.
