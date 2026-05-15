# Charter Amendments

This document records formal amendments to the [Constitutional Charter](./constitutional-charter.md). It is the authoritative history of how the Charter has evolved.

## Amendment Discipline

The Charter is the constitutional surface of Ghost Trace. Changes to it are not made through ordinary commits.

An amendment is required for any change that:

1. Alters or removes an existing Constitutional Invariant.
2. Adds a new Constitutional Invariant.
3. Modifies the Thesis.
4. Adds or removes Non-Goals.
5. Modifies the Constitutional Design Rule.

Editorial fixes that do not alter meaning (typography, formatting, broken links) do not require amendment but should be noted in the [decision log](./decision-log.md) if they touch the Charter.

## Amendment Process

1. **Proposal.** A change is proposed as an RFC in [`../rfcs/`](../rfcs/), explicitly tagged as `charter-amendment`. The RFC must identify which Charter element is affected and why the change cannot be expressed in subordinate documents instead.
2. **Falsifiability review.** The proposal is evaluated against the criteria in [Section 4 of the Charter](./constitutional-charter.md#4-constitutional-design-rule). Proposals that introduce non-falsifiable language are rejected on procedural grounds.
3. **Committee redaction.** Accepted proposals are redacted in committee mode — one section at a time, with explicit defense of each word choice.
4. **Amendment record.** Adopted changes are recorded in this file with the date, the Charter version before and after, the affected sections, and a brief rationale.
5. **Version bump.** The Charter version is incremented. Patch-level changes (`v0.1.x`) for clarifications that do not alter meaning; minor (`v0.2`) for substantive changes that do not alter identity; major (`v1.0+`) reserved for the moment the Charter is declared production-ready.

## Amendment Log

### `v0.1` — Charter Inception

**Date:** _(repository creation date)_
**Sections affected:** Thesis (frozen); Invariants 2.1 and 2.2 (frozen).
**Rationale:** Initial committee redaction of the Thesis, Observational Integrity, and Epistemic Separation. Remaining invariants (2.3–2.6), Non-Goals (Section 3), and Constitutional Design Rule (Section 4) remain in committee.

No amendments have yet been recorded against `v0.1`. Future entries follow.

---

<!-- AMENDMENT TEMPLATE — copy below this line when recording an amendment -->

<!--
### `vX.Y` — Short title

**Date:** YYYY-MM-DD
**Originating RFC:** [link]
**Sections affected:** [list]
**Summary:** [one or two sentences]
**Rationale:** [why the change was necessary and why it could not be made in a subordinate document]
**Falsifiability review outcome:** [pass with reasons]
-->
