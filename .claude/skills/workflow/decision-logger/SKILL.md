---
name: decision-logger
description: Maintain docs/charter/decision-log.md as append-only, mirroring the Charter's treatment of observations. Use this skill ALWAYS when a decision is being made about technology, policy, or process; ALWAYS when an RFC reaches accepted; ALWAYS when a committee redaction completes; ALWAYS when a supersession is required. Silent in-place edits to recorded decisions are forbidden — corrections happen by supersession, never by rewrite.
---

# decision-logger

The decision log is the Architecture Decision Record for Ghost Trace, with constitutional awareness: every entry is tested against the Charter ([`docs/charter/decision-log.md`](../../../../docs/charter/decision-log.md)). This skill keeps the log append-only in the spirit of [Invariant 2.1](../../../../docs/charter/constitutional-charter.md#21-observational-integrity).

## 1. Append-only discipline

Decisions are recorded once. The discipline mirrors the substrate:

- Once committed, the text of a decision entry is not edited. Status updates (`proposed` → `accepted`, `accepted` → `superseded`) are the only permitted in-place edits, and they edit the status line only.
- Corrections to a prior decision happen by supersession, never by in-place rewrite. The superseded decision's text remains visible; the new decision explains the correction.
- The log preserves history. A reader inspecting any prior moment of the project can reconstruct what the project believed it had decided at that moment.

The discipline is identical to [Invariant 2.1](../../../../docs/charter/constitutional-charter.md#21-observational-integrity)'s treatment of observations. The log is, in effect, a small substrate of decisions.

## 2. Procedure for adding a decision

Apply in order. Use the template at the bottom of [`decision-log.md`](../../../../docs/charter/decision-log.md).

### Step 1 — Sequential ID

Determine the next sequential ID, zero-padded to four digits (`0001`, `0002`, ..., `00NN`). Inspect the highest existing ID in the log; the new entry uses the next integer.

### Step 2 — Fill the template fields

Use the template at the bottom of `decision-log.md`. Required fields, with discipline:

- **Title.** Short and declarative. Names the decision in canonical vocabulary.
- **Status.** `proposed`, `accepted`, `superseded`, or `rejected`.
- **Date.** When the entry is recorded (UTC date is sufficient).
- **Context.** What made the decision necessary. Cite the prior state, the constraint, or the RFC that motivated the decision.
- **Decision.** What was decided. One paragraph, concrete.
- **Constitutional review.** Lists every Charter invariant the decision was tested against — not "consistent with the Charter" in the abstract. Each invariant is named (§2.1, §2.2, etc.), with the outcome of testing against it. Pending invariants whose working text the decision touches are also listed.
- **Consequences.** Both what is enabled AND what is now constrained or excluded. A decision that has no constraints to declare is suspicious; surface it.
- **Supersession.** `none` for original decisions; pointer to the superseding entry's ID when superseded.

### Step 3 — Apply discipline to the entry

Run the entry through:

- `falsifiability-check` on every claim in the `Decision` and `Consequences` fields.
- `vocabulary-discipline` on every term.
- `anti-marketing` on the `Title` and `Context` fields (where prose-level drift typically enters).

### Step 4 — Commit

Append the entry to `decision-log.md`. No edits to prior entries in the same commit.

## 3. Procedure for supersession

A decision is corrected, replaced, or invalidated by a *new* decision, never by editing the old one.

### Step 1 — Author the new decision

The superseding decision is authored as a normal new entry (§2). Its `Title` indicates that it supersedes a prior decision. Its `Context` explains why supersession was necessary. Its `Decision` field states the new position.

### Step 2 — Update the superseded entry's status line

The only permitted edit to the prior entry: change `Status: accepted` (or whatever the prior status was) to `Status: superseded`, and add a pointer to the superseding decision's ID in the `Supersession` field.

No other field of the prior entry is touched. The text of the prior `Decision`, `Context`, `Constitutional review`, and `Consequences` remains exactly as committed.

### Step 3 — Cross-link

Verify that the superseding entry's `Supersession` field names the prior decision's ID, and the prior entry's `Supersession` field names the superseding one. Bidirectional pointers prevent the log from losing the relation.

## 4. Forbidden: silent in-place edits

If a prior decision contains an error of fact — a wrong citation, a misnamed invariant, a contradictory claim — the correction is a supersession, not an in-place fix.

Editorial corrections that do not alter meaning (formatting, broken link repair) are permitted only if recorded as their own decision-log entry that names what was fixed and where. The log itself does not have an "editorial fixes" carve-out. The Charter's editorial-fix rule ([`amendments.md` §Amendment Discipline](../../../../docs/charter/amendments.md)) covers the Charter, not the decision log.

The reasoning: the decision log is the project's record of *what it believed it had decided when*. If that record is mutable, the project loses the ability to answer questions about its own historical decisions. The same property the Charter requires of observations, the decision log requires of decisions.

## 5. What this skill does not do

This skill does not approve decisions. It records them. Approval is human committee work or, for RFC-derived decisions, the RFC acceptance process.

This skill does not write the substantive content of `Decision` or `Constitutional review` fields. The author writes; this skill enforces structure and discipline.

## 6. Delegations

| Sub-task | Delegated to |
|---|---|
| Falsifiability of claims in `Decision` and `Consequences` | [`epistemic/falsifiability-check`](../../epistemic/falsifiability-check/SKILL.md) |
| Vocabulary check on every term | [`ontology/vocabulary-discipline`](../../ontology/vocabulary-discipline/SKILL.md) |
| Marketing rejection on prose fields (`Title`, `Context`) | [`enforcement/anti-marketing`](../../enforcement/anti-marketing/SKILL.md) |
| RFC-acceptance-triggered entries (RFC-to-decision-log handoff) | [`workflow/rfc-author`](../rfc-author/SKILL.md) §4 (Numbering and acceptance) |

## 7. Source citations used

- [`docs/charter/decision-log.md` §Format; existing entries 0001–0004](../../../../docs/charter/decision-log.md)
- [`docs/charter/constitutional-charter.md` §2.1 Observational Integrity](../../../../docs/charter/constitutional-charter.md#21-observational-integrity)
- [`docs/charter/amendments.md` §Amendment Discipline](../../../../docs/charter/amendments.md)
- [`.claude/CLAUDE.md` §2 Document hierarchy](../../../CLAUDE.md)
