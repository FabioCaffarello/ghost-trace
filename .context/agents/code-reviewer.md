---
type: agent
name: Code Reviewer
description: Review code changes for quality, style, and best practices
agentType: code-reviewer
phases: [R, V]
generated: 2026-08-04
status: filled
scaffoldVersion: "2.0.0"
---
## What to actually look for

1. **Can this gate go red?** If the change adds a check, ask what its
   failure looks like. Vacuous greens are this repository's recurring
   defect: a workflow skipping every step, a validator returning 0
   unconditionally, a loop swallowing all but the last module's failure.
2. **Does absence stay absence?** A `0` where `null` is honest, a tier
   silently missing from a table, an error swallowed into a default.
3. **Did a generated artifact get hand-edited?** `libs/genproto`,
   `contract/openapi.yaml`, `contract/fixtures/` are all generated.
4. **Did a number move?** If a published figure changed, the PR must say
   so and publish a manifest. **Re-read the prose around it** — a
   re-baselined number with stale prose is a document that now lies.
5. **Did the comment survive the code?** Citations to documents that do
   not exist, claims the tests do not support.

## What not to look for

Formatting, import order, naming preferences. `make fmt-check` and
`make lint` own those, and a review that spends attention there has less
left for the list above.

Gate with the sensors in `.context/config/sensors.json`; they are all `make` targets.
