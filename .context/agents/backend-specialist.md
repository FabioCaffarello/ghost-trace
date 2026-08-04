---
type: agent
name: Backend Specialist
description: Design and implement server-side architecture
agentType: backend-specialist
phases: [P, E]
generated: 2026-08-04
status: filled
scaffoldVersion: "2.0.0"
---
## What you own here

`services/ingestion` — the Go service. Two modules stitched by a
`replace` (`services/ingestion`, `libs/genproto`); no `go.work` yet.

## Non-negotiables

- **Never hand-edit `libs/genproto`.** Change `schemas/`, run
  `make generate`, commit the result.
- **Never hand-edit `contract/openapi.yaml`.** It is reflected from the
  Go wire types in `libs/wire`. Change the types, run `make openapi`.
- Enumerations live as **Go values** (`policy.ReasonCodes`,
  `ingest.KeyClasses`, `app.ValidOutcomes`) and are injected into the
  schema. Retyping one into a struct tag is how three fabricated enums
  got published; guards now prevent it.
- Wire-facing struct tags: **no commas inside a `jsonschema` description**
  — the library splits on them and silently truncates.

## The service is fail-open

Telemetry loss is expected; an unknown session gets 202; a first visit is
never blocked. If you find yourself adding an error path on the
collection side, check §5 first.

Gate with the sensors in `.context/config/sensors.json`; they are all `make` targets. Any change to the wire also needs the wire-contract skill.
