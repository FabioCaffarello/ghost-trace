# RFCs

This directory contains Requests for Comments — proposals for changes to Ghost Trace.

## Status of this Directory

The RFC discipline is active. The [`template.md`](./template.md) is the canonical RFC format. [`draft/`](./draft/) is the working directory for RFCs at any non-rejected status (including `accepted`); [`discussion/`](./discussion/) contains evidence files supporting RFCs at `status: discussion` (preserved as historical record after acceptance per [`discussion/README.md`](./discussion/README.md)).

## RFC Lifecycle

1. **Draft.** Created in [`draft/`](./draft/) with a working title and `status: draft`.
2. **Discussion.** Author marks `status: discussion` when ready for review. An evidence file may be created in [`discussion/`](./discussion/) to support the discussion phase.
3. **Decision.** Reviewed against the Charter. Accepted, rejected, or returned for revision.
4. **Accepted.** `Status:` field updated to `accepted`; `Decision Record` section populated with the resolution + any committee extensions + reversal conditions; recorded in the [decision log](../charter/decision-log.md). File remains in [`draft/`](./draft/) under its working-title filename — see [`decision-log §0026`](../charter/decision-log.md) for the actual-practice convention. Charter amendments are additionally recorded in [`amendments.md`](../charter/amendments.md) per the amendment process.
5. **Supersession.** A later RFC may supersede an earlier one. Both remain in the directory; status is updated.

## Referencing

RFCs are referenced by filename across their lifecycle. Decision-log entries and amendments.md entries link to RFCs by relative filename. The numbering-with-rename procedure described in earlier versions of this document was never followed in practice; per [`decision-log §0026`](../charter/decision-log.md), filename-based referencing is codified as the actual-practice convention across the [`§0007`](../charter/decision-log.md)–[`§0025`](../charter/decision-log.md) RFC acceptance cycles.

## Categories

- **`standard`** — proposes new capability or design within existing constitutional bounds.
- **`charter-amendment`** — proposes a change to the Constitutional Charter. Must follow the amendment process in [`../charter/amendments.md`](../charter/amendments.md).
- **`ontology-revision`** — proposes a change to the Ontology that does not require Charter amendment.
- **`architecture`** — proposes a concrete architectural design (e.g., storage substrate selection).
- **`experiment`** — proposes a time-bounded experiment whose outcome will inform a future RFC.

## Constitutional Review

Every RFC undergoes constitutional review. See [`template.md`](./template.md) for the required structure.
