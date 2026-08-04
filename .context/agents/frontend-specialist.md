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

`services/ingestion/internal/web/static/sdk.js` — the browser SDK, and
`internal/web` which serves the demo page. This is not a frontend
practice: it is **the producer of the telemetry wire**, and that is the
only reason it needs its own playbook.

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
