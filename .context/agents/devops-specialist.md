---
type: agent
name: Devops Specialist
description: Design and maintain CI/CD pipelines
agentType: devops-specialist
phases: [E, C]
generated: 2026-08-04
status: filled
scaffoldVersion: "2.0.0"
---
## What you own here

The root `Makefile`, `.github/workflows/`, `deploy/docker/`, `compose.yml`.

## The contract between them

**Every CI step is `run: make <target>` and nothing else.** Workflows
decide when and where; the Makefile decides what. Adding a step that
inlines a command breaks the equivalence that makes `make ci` meaningful.

## Traps already hit here

- `.SHELLFLAGS` needs GNU Make 3.82; macOS ships 3.81 and ignores it
  silently. Loops end in explicit `|| exit 1`.
- **No sync gate may use `git status`** — it reports change relative to
  HEAD, not drift. Generate to a temp dir and diff.
- `mktemp -t name` works on BSD and fails on GNU. Use an explicit
  `XXXXXX` template.
- Actions are pinned by commit SHA; a tag is a third party's promise.
- Job names are required status checks. **Renaming one silently removes
  the protection.**

## Containers

Base images are ARG-parameterised to invalid all-zero digests, so a build
that skips pinning fails fast. `CGO_ENABLED=0` is load-bearing, not an
optimisation: it is what allows distroless.

Gate with the sensors in `.context/config/sensors.json`; they are all `make` targets.
