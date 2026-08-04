---
type: doc
name: security
description: Security policies, authentication, secrets management, and compliance requirements
category: security
generated: 2026-08-04
status: filled
scaffoldVersion: "2.0.0"
---
# Security context

`SECURITY.md` is canonical, including the private reporting channel.
What an agent should hold in mind:

## The trust boundary

Everything from the browser is hostile. `site_key` is public and
identifies a tenant — it authenticates nobody. `session_token`
correlates telemetry and is **not a credential**. Only `secret_key`
authenticates, on `/v1/decisions` and `/v1/outcomes`, and it is compared
in constant time because a byte-wise comparison leaks prefix length
through response timing.

## The collection limit is a security property

The keystroke channel carries timing and a coarse class — never the key,
never the field value. A change that widens this is not a feature, and
the SDK is the place it would happen.

## Known false-positive population

`VALUE_INJECTED` is the strongest single bot signal and the one the
policy explicitly refuses to treat as categorical: dictation, IME
composition and some assistive input devices reach the same code path
with no preceding keydown. That is the population this project says it
is most worried about mis-flagging.

## Human subjects

`experiments/` runs a study with real participants. Changes touching
consent text or what is collected belong in the PR title, not only in
the diff.

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
