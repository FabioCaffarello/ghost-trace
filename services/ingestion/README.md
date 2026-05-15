# Ingestion Service

## Constitutional Role

Receives observations from producers and commits them to the primary event log. The point at which the system takes responsibility for a record. After commitment, the record is governed by [Invariant 2.1 — Observational Integrity](../../docs/charter/constitutional-charter.md#21-observational-integrity).

## Status

Not implemented.

## Required Properties

- Idempotent commitment: a producer retry must not produce duplicate records in the log.
- Producer-time preservation: the producer's view of when the event occurred is recorded alongside the system's view of when it was received.
- Source attribution: distinguishes telemetry from real users, infrastructure collectors, and adversarial simulators.
- Schema validation: rejects records that do not conform to a registered schema version.
