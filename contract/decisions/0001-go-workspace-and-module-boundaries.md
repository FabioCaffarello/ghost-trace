# 0001 — A Go workspace, and when a library gets extracted

**Status:** accepted · **Date:** 2026-08-04 · **Milestone:** PR-2.1

## Context

Until now two Go modules were stitched together by a filesystem
`replace` in `services/ingestion/go.mod`. The audit flagged this (M18):
a `replace` chain is manageable at two modules and drifts as they
multiply — and Phase 2 multiplies them to at least six (four services
plus shared libraries).

## Decision

**A root `go.work` lists every module.** It serves local development
and CI.

**The `replace` directives stay.** They are not redundant with the
workspace: they keep each module buildable *on its own*, which is
precisely what the container build does — it copies `libs/` and one
service directory and never sees `go.work`. CI enforces this with a
per-module matrix that runs with `GOWORK=off`.

**A package becomes its own module when a second service will consume
it, and not before.** The standing example of the other side of this
rule is `internal/app/protomap.go`, which stays inside the application
package until a second consumer exists.

`libs/middleware` is extracted now, ahead of its second consumer,
because all four Phase-2 services serve HTTP and every one of them will
want the chain. Extracting it here keeps the service PRs about their
service. That is a deliberate exception, and the bar for the next one is
the same: name the consumers.

## Consequences

- The container's module-graph layer is a hand-maintained list of
  `COPY libs/<name>/go.mod` lines. This is explicit rather than a glob
  so a missing entry fails at build time — which is exactly how
  `libs/middleware` announced itself.
- `make` targets loop over `GO_MODULES`; adding a module means adding it
  there, to `go.work`, to the CI matrix, and to the Dockerfile.
- Shared libraries are stdlib-only where possible. A shared library that
  drags dependencies into four services is a coupling nobody asked for.
