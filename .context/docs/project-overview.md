---
type: doc
name: project-overview
description: High-level overview of the project, its purpose, and key components
category: overview
generated: 2026-08-04
status: filled
scaffoldVersion: "2.0.0"
---
# Ghost Trace, for an agent picking this up cold

**Does this session behave like a human?** — answered from interaction
dynamics alone: how the pointer moves, how keys are timed, how a form is
actually filled. Explicitly NOT *which browser is this*; fingerprinting
is out of scope and the tools that defeat it are freely available.

Four Go services — the collector (`services/collector`), the decision
engine, the archive, and a stand-in customer site on its own origin —
serving four HTTP endpoints between them, plus a browser SDK, a protobuf
archive, and an adversarial experiment layer that produces six published
numbers. It was one binary through M5; Phase 2 split it, Phase 3 made
what the split can lose countable, Phase 4 measured it under load.
`make shadow-http` is what holds the split honest, and the all-in-one
binary is kept permanently as the no-broker path `make numbers` uses.

## The thing to understand first

This is a **measurement project**. Its central claim is that six numbers
reproduce. Almost every convention here — committed generated artifacts,
drift gates, seeded adversaries, run manifests with provenance — exists
so that claim can be checked rather than believed.

Two rules are not style, and a change that widens either is a change to
what the project is:

- **Absence is never zero.** A tier that did not run is not a tier that
  found nothing. A false-positive rate with no human data is `null`.
- **Key content is never collected.** The keystroke channel carries
  timing and a coarse class. Not the key, not the field value, ever.

## What is honestly unfinished

The false-positive rate has **no data**. It is the number that governs
every other one, it is calendar-bound rather than effort-bound, and
until it exists the detection rates are unfalsifiable in exactly the way
this project was built to avoid. Anything you write should preserve that
admission rather than smooth it over. Recruitment is gated on a
data-governance RFC that has not been written, and
`disclosure_test.go` fails if `experiments/PARTICIPANTS.md` stops
saying so.

The other standing gap is **backpressure**: the archive commits about
4 133 records/s against a collector that bends near 16 000, with a
stream between them bounded only by a seven-day retention window.
Nothing slows the producer down, and `stream_skipped` reports the loss
after the fact.

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
