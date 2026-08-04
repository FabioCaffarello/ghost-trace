# Project Rules and Guidelines

> Auto-generated from .context/docs on 2026-08-04T19:49:52.606Z

## README

# Ghost Trace — agent context

**Does this session behave like a human?** — answered from interaction
dynamics alone. Four HTTP endpoints across three Go services, a browser
SDK, an append-only archive, and an adversarial experiment layer that
produces six published numbers.

This is a **measurement project**. Its central claim is that those six
numbers reproduce, and almost every convention here exists so the claim
can be checked rather than believed.

## Read these first

- [`.context/docs/project-overview.md`](.context/docs/project-overview.md)
  — what this is, and what is honestly unfinished
- [`.context/docs/architecture.md`](.context/docs/architecture.md)
  — the shape, in one screen
- [`.context/docs/development-workflow.md`](.context/docs/development-workflow.md)
  — `make verify`, `make ci`, the commit convention

Then as the task calls for it:
[data-flow](.context/docs/data-flow.md) ·
[testing-strategy](.context/docs/testing-strategy.md) ·
[security](.context/docs/security.md) ·
[tooling](.context/docs/tooling.md) ·
[glossary](.context/docs/glossary.md)

## Two rules that are not style

- **Absence is never zero.** A tier that did not run is not a tier that
  found nothing. A false-positive rate with no human data is `null`.
- **Key content is never collected.** The keystroke channel carries
  timing and a coarse class. Not the key, not the field value, ever.

A change that widens either is a change to what the project is.

## Before touching the wire, or a number

Two skills are not optional:

- [`wire-contract-change`](.context/skills/wire-contract-change/SKILL.md)
  — the server tolerates unknown fields **by design**, so a renamed
  field degrades every measurement in silence. That checklist exists
  because it happened.
- [`numbers-invariant`](.context/skills/numbers-invariant/SKILL.md)
  — `make numbers` after any change to `services/`, `experiments/` or
  `schemas/`. If a number moved, re-read the prose around it.

## Where the truth actually lives

This directory **indexes and enforces**; it never forks.

| what | where |
| --- | --- |
| the external surface | [`contract/architecture.md`](contract/architecture.md) §0–§9 |
| the HTTP contract | [`contract/openapi.yaml`](contract/openapi.yaml) — generated, never hand-edited |
| what the harness sends | [`contract/fixtures/`](contract/fixtures/) — generated |
| where the design is going | [`contract/roadmap.md`](contract/roadmap.md) |
| write-ups of work that has run | [`docs/`](docs/) — the repository's own, not produced by any tool |
| what was measured, when, with which seed | [`docs/results/`](docs/results/) |
| how to build, test and gate | the root [`Makefile`](Makefile) — `make help` |
| how to contribute | [`CONTRIBUTING.md`](CONTRIBUTING.md) |

The moment a copy of the architecture lives under `.context/`, the
harness recreates the drift problem it exists to solve.

## Gates

Every sensor in [`.context/config/sensors.json`](.context/config/sensors.json)
is a `make` target and nothing else, so humans, CI and agents share one
vocabulary. `make ci` is the whole gate; `make numbers` is the invariant
and is deliberately outside it — it needs real browsers and seven
minutes.

