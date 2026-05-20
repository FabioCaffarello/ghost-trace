# Event Schemas

Schemas for Category I records — observations committed to the primary event log.

## Status

Active. Schemas technology selected per [`decision-log §0024`](../../docs/charter/decision-log.md): Protocol Buffers (proto3). Schemas versioned per Protobuf convention (`v1/` subdirectory). First schema landed: [`v1/declared_session.proto`](./v1/declared_session.proto) per [`decision-log §0030`](../../docs/charter/decision-log.md) ingestion service skeleton.

## Required Properties

When defined, event schemas must support:

- Producer-generated identifiers (content-addressable preferred).
- Distinction between producer time (`occurred_at`) and receipt time (`received_at`).
- Source attribution sufficient to distinguish telemetry from real users, infrastructure collectors, and adversarial simulators.
- Strong typing per event variety; no unified-record-with-discriminator pattern.
- Schema versioning with a registry.

See [`../README.md`](../README.md) for constitutional anchors.
