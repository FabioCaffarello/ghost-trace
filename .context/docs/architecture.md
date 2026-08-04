---
type: doc
name: architecture
description: System architecture, layers, patterns, and design decisions
category: architecture
generated: 2026-08-04
status: filled
scaffoldVersion: "2.0.0"
---
# Architecture — the index

**Read `contract/architecture.md` §0–§9.** It is the binding surface and
this file will not repeat it.

## The shape, in one screen

```
browser ──sdk.js──> /v1/sessions, /v1/telemetry ──> internal/api (thin transport)
                                                          │
app server ────────> /v1/decisions, /v1/outcomes ──> libs/decision (shared: the
                                                     collector AND the decision
                                                     engine mount it — ADR-0005)
                                                          │
                                                          v
                                            internal/app (use cases, ports)
                                                          │
                          ┌───────────────────────────────┼──────────────────┐
                          v                               v                  v
                   internal/session              internal/feature      internal/policy
                   (aggregate, invariants)       (extraction)          (scoring)
                                                          │
                                                          v
                                     internal/adapters/substratearchive → internal/substrate
```

Dependencies point inward: `domain ← app ← adapters`, verified acyclic.
Handlers are deliberately thin — if one grows an if-statement about the
domain, it is in the wrong layer.

## Two things that surprise people

- **`internal/app/protomap.go` lives in `app`, not `adapters`.** It maps
  domain values into the durable record type the port speaks. Its
  evaluation half has already left for `libs/decision`, which is what
  ADR-0001 said would happen once a second consumer existed; the
  telemetry half stayed.
- **The service is fail-open.** Telemetry loss is expected, unknown
  sessions get 202, and a decision is a judgement under uncertainty.
  `score` and `confidence` are separate fields precisely so "nothing
  suspicious observed" is distinguishable from "this looks human".

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
