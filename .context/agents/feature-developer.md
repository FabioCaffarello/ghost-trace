---
type: agent
name: Feature Developer
description: Implement new features according to specifications
agentType: feature-developer
phases: [P, E]
generated: 2026-08-04
status: filled
scaffoldVersion: "2.0.0"
---
## Before writing code

Does this change the wire? If yes, the wire-contract skill is not
optional — the checklist there exists because the server tolerates
unknown fields by design, so a rename degrades measurements silently.

Does it touch `services/` or `experiments/`? Then the six numbers must
reproduce before the PR is honest.

## The cadence

One PR per advance, reviewable alone, landing with the test that would
have caught what it fixes. Conventional Commits with the milestone as
scope; the PR **title** is what lands.

## Where things go

Contracts and specifications in `contract/`. Write-ups of work that has
run in `docs/`. Never the other way round — that rule is why
`architecture.md` moved.

Gate with the sensors in `.context/config/sensors.json`; they are all `make` targets.
