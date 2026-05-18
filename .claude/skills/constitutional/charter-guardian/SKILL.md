---
name: charter-guardian
description: Protect the Constitutional Charter from edits that bypass the formal amendment process. Use this skill ALWAYS when reading or writing any file under docs/charter/, ALWAYS when a commit modifies constitutional-charter.md, and ALWAYS when a proposal is phrased as "let's clarify section X" of the Charter or invokes wording from a frozen section. Apparently-editorial edits that change meaning are the most common silent-amendment failure mode.
---

# charter-guardian

The Constitutional Charter is authoritative ([`README.md` §Document Hierarchy](../../../../README.md); [`docs/charter/decision-log.md` §0004](../../../../docs/charter/decision-log.md)). The Charter is changed only through the formal amendment process recorded in [`amendments.md`](../../../../docs/charter/amendments.md). This skill keeps casual edits from becoming silent amendments.

## 1. Protected vs in-committee elements

The status of each Charter element is canonicalized in [`.claude/CLAUDE.md` §4](../../../CLAUDE.md), which mirrors the status banner in `constitutional-charter.md`. The two must remain in sync; drift between them is the canonical inconsistency mode of this infrastructure and is checked by the Phase 8 SELF-AUDIT.

### Protected (FROZEN)

These elements may not be edited except through the formal amendment process:

- §1 Thesis
- §2 Constitutional Invariants (header and the four qualification criteria)
- §2.1 Observational Integrity
- §2.2 Epistemic Separation
- §2.3 Provenance Integrity (frozen v0.4 — `decision-log.md` §0017)
- §2.5 Hypothesis Lifecycle Explicitness (frozen v0.3 — `decision-log.md` §0013)
- §4 Constitutional Design Rule (frozen v0.2 — `decision-log.md` §0007)

### In-committee (PENDING)

These elements are under committee redaction. Edits to them are managed by `invariant-redactor`, not by direct write:

- §2.4 Inferential Influence Disclosure
- §2.6 Evidential Independence Integrity
- §3 Non-Goals

## 2. Procedure for any proposed edit to constitutional-charter.md

### Step 1 — Identify the element

Locate the specific element (section number, anti-pattern entry, boundary condition, qualification criterion) being edited. Identification is by structural location, not by line number; line numbers are unstable across edits.

### Step 2 — Classify by status

Read [`.claude/CLAUDE.md` §4](../../../CLAUDE.md) to determine whether the element is FROZEN or PENDING.

### Step 3 — Apply the rule for that status

#### If FROZEN

The edit is an amendment by definition ([`amendments.md` §Amendment Discipline](../../../../docs/charter/amendments.md)). Block the edit unless **all** of the following hold in the same change set:

1. An RFC of type `charter-amendment` exists in `docs/rfcs/` proposing the change.
2. The RFC has passed falsifiability review (`falsifiability-check`).
3. An entry is being added to `amendments.md` recording the date, the affected sections, the originating RFC, the summary, the rationale, and the falsifiability-review outcome.
4. The amendment includes a version bump per the rule in `amendments.md` — patch for clarifications that do not alter meaning; minor for substantive changes that do not alter identity; major reserved for the moment the Charter is declared production-ready.

If any of these is missing, surface the requirement to the human and do not proceed.

#### If PENDING

The edit is committee redaction. Defer to `invariant-redactor`, which structures redaction in a scratch document under `docs/charter/in-committee/§NN-<name>.md` and applies the redaction procedure. Direct edits to a PENDING element in `constitutional-charter.md` are not the path; pending sections are merged from the scratch document only at the moment they pass the redaction merge checklist.

#### If neither (typo, link repair, formatting)

Permitted as an editorial change. The change must still be recorded as an editorial note in [`decision-log.md`](../../../../docs/charter/decision-log.md) per [`amendments.md` §Amendment Discipline](../../../../docs/charter/amendments.md): "Editorial fixes that do not alter meaning ... do not require amendment but should be noted in the decision log if they touch the Charter."

## 3. Checklist for every edit

Apply regardless of category, after the status rule above:

1. **Meaning vs editorial.** Does the edit change meaning? An apparently editorial edit — renaming, reordering, restating — often changes meaning. If meaning changes, the edit is substantive and follows the rule for its status. Re-read the before-and-after as a hostile reader.
2. **Cross-references.** Does the edit affect cross-references in subordinate documents (`docs/ontology/`, `docs/architecture/`, `docs/rfcs/`, `schemas/`, `services/`)? If yes, those documents are updated in the same change set. The `subordination-checker` skill verifies this.
3. **Status changes.** Does the edit change the FROZEN/PENDING status of any element? Status changes are themselves amendments — moving an element from PENDING to FROZEN, or relaxing FROZEN to PENDING, is constitutional. The same change must update [`.claude/CLAUDE.md` §4](../../../CLAUDE.md).

A change that satisfies the status rule but fails any item in this checklist is incomplete, not permitted.

## 4. What this skill does not do

This skill does not approve amendments. It enforces the procedural envelope: that the amendment process exists, that it is followed, and that the change set contains the artifacts the process requires. Approval is committee work, recorded through the formal process.

## 5. Source citations used

- [`docs/charter/constitutional-charter.md` §1 through §4](../../../../docs/charter/constitutional-charter.md)
- [`docs/charter/amendments.md` §Amendment Discipline; §Amendment Process](../../../../docs/charter/amendments.md)
- [`docs/charter/decision-log.md` §0004 — Charter is authoritative](../../../../docs/charter/decision-log.md)
- [`README.md` §Document Hierarchy](../../../../README.md)
- [`.claude/CLAUDE.md` §4 Charter status at a glance](../../../CLAUDE.md)
- [`.claude/skills/constitutional/invariant-redactor/SKILL.md`](../invariant-redactor/SKILL.md)
- [`.claude/skills/constitutional/subordination-checker/SKILL.md`](../subordination-checker/SKILL.md)
- [`.claude/skills/epistemic/falsifiability-check/SKILL.md`](../../epistemic/falsifiability-check/SKILL.md)
