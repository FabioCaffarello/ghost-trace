---
type: doc
name: testing-strategy
description: Test frameworks, patterns, coverage requirements, and quality gates
category: testing
generated: 2026-08-04
status: filled
scaffoldVersion: "2.0.0"
---
# Testing strategy

## Wire-level, not in-memory

The `internal/api` tests drive **real HTTP with real JSON**. v1 lost a
provenance chain to a field-name mismatch between a producer and a
consumer while every in-memory test stayed green — that is why the
characterization tests cross the wire.

## The layers

| layer | what it proves |
| --- | --- |
| byte-level goldens | the exact bytes clients receive, all four endpoints and every error body |
| OpenAPI conformance | each golden satisfies the schema the **spec declares** for that path/method/status |
| contract harness | the fixtures the experiment clients produce validate AND replay against a real server |
| vocabulary guards | `sdk.js` and the published enumerations agree, in both directions |
| drift guards | published reason codes vs the constants; wire tags vs swallowed text; feature state vs proto |
| statistics selftest | 22 assertions on the estimators and the planted fixture structure |
| the six numbers | the invariant — expensive, gated, run with `make numbers` |

## The recurring pattern

Nearly every guard here exists because something was silently wrong and
nothing could contradict it. When adding a check, ask what it would look
like **red** — a gate that cannot fail is the failure mode this
repository keeps rediscovering.

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
