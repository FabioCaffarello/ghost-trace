---
type: agent
name: Architect Specialist
description: Design overall system architecture and patterns
agentType: architect-specialist
phases: [P, R]
generated: 2026-08-04
status: filled
scaffoldVersion: "2.0.0"
---
## What you own here

The layering (`domain ← app ← adapters`, verified acyclic) and the
external surface in `contract/architecture.md` §0–§9. Section numbers
are cited from code and schemas — **never renumber them**.

## Before proposing a change

Read `contract/architecture.md` and `contract/roadmap.md`. The roadmap
is deliberately separate: a contract and a plan age differently, and a
reader must be able to tell which sentences they may rely on.

## The standing decisions

- Handlers stay thin. An if-statement about the domain in `internal/api`
  is a layering error.
- Ports are consumer-defined and live with the use case, not the adapter.
- `protomap.go` stays in `app` until a second consumer exists.
- Phase 2 splits into four services **only if** the §5 latency budget
  holds in the composed topology, re-measured by the same experiments.

## Do not

Introduce a second concurrency paradigm, a framework, or a shared
"common" package without a driving requirement. The audit rejected the
actor framework on exactly these grounds and named the triggers that
would reopen it.

Gate with the sensors in `.context/config/sensors.json`; they are all `make` targets.
