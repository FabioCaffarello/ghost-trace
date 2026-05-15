# RFCs

This directory contains Requests for Comments — proposals for changes to Ghost Trace.

## Status of this Directory

The project is in pre-implementation constitutional drafting. No RFCs have yet been accepted. The [`template.md`](./template.md) is the canonical RFC format. The [`draft/`](./draft/) subdirectory is the home of RFCs under development.

## RFC Lifecycle

1. **Draft.** Created in [`draft/`](./draft/) with a working title and `status: draft`.
2. **Discussion.** Author marks `status: discussion` when ready for review.
3. **Decision.** Reviewed against the Charter. Accepted, rejected, or returned for revision.
4. **Accepted.** Moved out of `draft/`, renumbered, recorded in the [decision log](../charter/decision-log.md).
5. **Supersession.** A later RFC may supersede an earlier one. Both remain in the directory; status is updated.

## Numbering

RFCs are numbered when accepted, in sequence. Drafts in `draft/` carry working titles, not numbers, until accepted.

## Categories

- **`standard`** — proposes new capability or design within existing constitutional bounds.
- **`charter-amendment`** — proposes a change to the Constitutional Charter. Must follow the amendment process in [`../charter/amendments.md`](../charter/amendments.md).
- **`ontology-revision`** — proposes a change to the Ontology that does not require Charter amendment.
- **`architecture`** — proposes a concrete architectural design (e.g., storage substrate selection).
- **`experiment`** — proposes a time-bounded experiment whose outcome will inform a future RFC.

## Constitutional Review

Every RFC undergoes constitutional review. See [`template.md`](./template.md) for the required structure.
