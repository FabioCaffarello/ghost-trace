---
type: doc
name: development-workflow
description: Day-to-day engineering processes, branching, and contribution guidelines
category: workflow
generated: 2026-08-04
status: filled
scaffoldVersion: "2.0.0"
---
# Development workflow

```bash
make bootstrap   # assert the toolchain, name what is missing
make verify      # format, vet, lint, race tests — before pushing
make ci          # everything CI runs, in the order CI runs it
```

**Every CI step is a `make` target and nothing else.** The workflows
decide when things run; the Makefile decides what they do. If `make ci`
is green and CI is not, that is a bug in the split.

## Commits and pull requests

Conventional Commits with the milestone as scope
(`feat(r1.18): ...`). Pull requests are **squash-merged**, so the PR
**title** is the commit that lands and the text release automation
reads. CI checks it with the same script the commit hook runs.

## The cadence

One PR per advance. Each is reviewable alone and lands with the test
that would have caught the thing it fixes.

## What will fail a PR

Mostly generated-artifact drift, each with a fix command in the failure:
`contract/openapi.yaml`, `libs/genproto`, `contract/fixtures` — all
generated, committed and gated. Plus the byte-level goldens under
`services/ingestion/internal/api/testdata/golden/`, which freeze the
bytes clients receive.

See `CONTRIBUTING.md` for the full list; it is the canonical version.

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
