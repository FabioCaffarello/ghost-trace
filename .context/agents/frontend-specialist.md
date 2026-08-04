---
type: agent
name: Browser SDK Specialist
description: Design and implement user interfaces
agentType: frontend-specialist
phases: [P, E]
generated: 2026-08-04
status: filled
scaffoldVersion: "2.0.0"
---
## What you own here

`services/ingestion/internal/sdk/sdk.js` — the browser SDK, served by
the COLLECTOR because it is Ghost Trace's artefact and not the
customer's — and `services/demo-web`, the stand-in customer site on its
own origin. This is not a frontend practice: sdk.js is **the producer of
the telemetry wire**, and that is the only reason it needs its own
playbook.

The two are separate services now, and the page loads the SDK
cross-origin exactly as an integrator would. If you change where the SDK
lives, `internal/ingest/vocabulary_test.go` reads it by path and will
say so loudly rather than pass.

## The SDK defines the vocabulary

Whatever `sdk.js` emits IS the wire. `internal/ingest/vocabulary.go`
publishes those vocabularies and `vocabulary_test.go` holds the two to
each other **in both directions** — a value the SDK sends that the
contract omits, and a value the contract advertises that never arrives.
Adding an event type, a key class or a form action means editing both.

## Hard limits

- Timing and a coarse class only. **Never the key, never the value.**
  `classOf` returns a class; `targetOf` returns a stable hash of field
  identity.
- Nothing persistent is written to the client. The token lives in a
  closure and dies with the page.
- Collection is server-driven (`collect` in the session response) so it
  can be retuned without shipping a new SDK. Tolerate unknown keys.
- Telemetry is fire-and-forget. Do not add retries; loss is expected and
  a retry loop is how a stale SDK hammers a server.
