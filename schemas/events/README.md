# Event Schemas

Schemas for Category I records — observations committed to the primary event log.

## Status

Not yet defined. Awaiting Ontology stabilization and schema technology selection.

## Required Properties

When defined, event schemas must support:

- Producer-generated identifiers (content-addressable preferred).
- Distinction between producer time (`occurred_at`) and receipt time (`received_at`).
- Source attribution sufficient to distinguish telemetry from real users, infrastructure collectors, and adversarial simulators.
- Strong typing per event variety; no unified-record-with-discriminator pattern.
- Schema versioning with a registry.

See [`../README.md`](../README.md) for constitutional anchors.
