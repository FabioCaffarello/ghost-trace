---
type: doc
name: data-flow
description: How data moves through the system and external integrations
category: data-flow
generated: 2026-08-04
status: filled
scaffoldVersion: "2.0.0"
---
# Data flow

## Collection → decision

1. `sdk.js` opens a session (`POST /v1/sessions`) and receives a
   server-driven `collect` policy — sampling rate and event families are
   tunable without shipping a new SDK.
2. It batches interaction events and flushes them (`POST /v1/telemetry`).
   Fire-and-forget: loss is expected, an unknown token still gets 202.
3. The application server asks for a judgement
   (`POST /v1/decisions`, `secret_key`). This is the ONLY endpoint that
   accepts `subject_id` and `action` — which is why neither is ever read
   from a browser.
4. Later, the application labels what happened
   (`POST /v1/outcomes`). **Nothing currently calls this.** The labels
   channel every future calibration depends on has no client.

## The wire is a flat union

`TelemetryEvent` declares every field and `type` selects which apply.
It is described that way rather than modelled as a JSON Schema `oneOf`
because the decoder really is one flat struct — a per-type variant would
be a second definition of the wire, free to drift from the one the
server parses.

## Archive

Evaluations are canonicalised, hashed with BLAKE3, and appended to a
SQLite index plus a content-addressed blob store
(`internal/substrate`). Writes serialise through a single `Append`, so
the append-only guarantee has one enforcement site.

The protobuf runtime version is an **archive-format pin**, not a
dependency: canonical bytes are hashed for identity, so upgrading it is
an archive-compatibility event. Dependabot is configured to leave it
alone.

## Canonical sources — never restate them here

| what | where |
| --- | --- |
| the external surface | `contract/architecture.md` §0–§9 |
| the HTTP contract | `contract/openapi.yaml` (generated; never hand-edit) |
| what the harness sends | `contract/fixtures/requests/` (generated) |
| how to build, test, gate | the root `Makefile` — `make help` |
| how to contribute | `CONTRIBUTING.md` |
| what was measured, when | `docs/results/` |
| where the design is going | `contract/roadmap.md` |

This directory **indexes and enforces**; it does not fork. The moment a
copy of the architecture lives here, the harness recreates the drift
problem it exists to solve.
