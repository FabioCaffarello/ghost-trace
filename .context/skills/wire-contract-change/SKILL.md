---
type: skill
name: Wire Contract Change
description: Change anything that crosses the HTTP wire. Use when renaming or adding a field, changing an enumeration, or touching sdk.js, schemas/ or libs/wire
skillSlug: wire-contract-change
phases: [P, E, V]
generated: 2026-08-04
status: filled
scaffoldVersion: "2.0.0"
---

# Changing the wire

The service **tolerates unknown fields by design** — telemetry is
fire-and-forget and an SDK must survive a server that has moved on
(§5, §7). That tolerance means a renamed field does not fail: every
client keeps sending the old name, the server zero-fills the new one,
and the measurements degrade in silence with the whole suite green.

That is audit finding M22, and it has actually happened here. Walk the
checklist; it is short because the guards do most of the work now.

## The checklist

1. **Change the Go type** in `libs/wire/wire.go`. It is a module of
   its own because more than one service serves this contract.
   That is the source of truth — the specification is reflected from it.
   - No commas inside a `jsonschema:"description=..."`. The library
     splits on them and silently truncates. Nine clause endings were
     lost that way.
   - **Never put an enumeration in a struct tag.** Enumerations live as
     Go values (`policy.ReasonCodes`, `ingest.KeyClasses`,
     `decision.ValidOutcomes`) and the generator injects them. Three
     retyped enums were published wrong before this rule existed.
2. **If it is a telemetry vocabulary or a session-start field**, edit
   `internal/ingest/vocabulary.go` AND `sdk.js` — `vocabulary_test.go`
   compares them in both directions and fails either way — AND describe
   it in `experiments/PARTICIPANTS.md`, in the section that says what
   IS recorded. `disclosure_test.go` fails until a volunteer would be
   told. This is not bookkeeping: a collected value nobody was told
   about is the one finding in this repository with a person on the
   other side of it.
3. **If it touches the archive**, edit `schemas/` and
   `make generate`. Never hand-edit `libs/genproto`.
4. **Update the harness** — `experiments/lib/wire.js` and
   `experiments/wire.py`. They are one module per language precisely so
   this is one edit rather than five.
   The wire modules feed EVERY producer, not just the tiers:
   `tools/loadgen` and `services/demo-web` build their bodies from
   `libs/wire` (Go), and `deploy/kill-test.py` / `deploy/loss-audit.py`
   import `experiments/wire.py`. This list exists because the checklist
   once said "the harness" and meant two files — and the load driver,
   outside it, spent a phase sending pointer events the server silently
   discarded (PR-5.0c). A producer not on this list is a producer that
   can drift.
5. **Regenerate**: `make openapi && make contract-fixtures`.
6. **Regenerate goldens if the response bytes really changed**:
   `go test ./internal/api -run Golden -update`. If you did not intend
   to change them, you have changed the contract by accident.
7. **`make ci`**, then **`make numbers`** — this is a services/ change.

## What each guard catches

| guard | what it sees |
| --- | --- |
| `contract-fixtures-sync` | the JS and Python halves disagreeing |
| contract harness | a fixture failing the schema, or rejected by a real server |
| `vocabulary_test.go` | SDK and contract vocabularies diverging |
| OpenAPI conformance | the spec promising something the server does not send |
| goldens | any byte a client receives moving |

## The thing to remember

`additionalProperties: false` on the request schemas is what makes a
rename fail from **either** direction. If you find yourself relaxing it,
you are removing the guard, not fixing the test.
