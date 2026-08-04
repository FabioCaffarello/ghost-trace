# 0002 — Libraries are named for what they are

**Status:** accepted · **Date:** 2026-08-04 · **Milestone:** PR-2.1

## Context

The refactor plan called the first shared library `libs/httpkit`. The
audit pushed back on the name, preferring plain nouns and noting that
the decision should be made once rather than per library.

## Decision

**A library is named for what it contains, as a plain noun.** No `kit`,
`util`, `common`, `helpers` or `shared`.

The first library is therefore `libs/middleware` — it is the HTTP
middleware chain, and nothing else. `httpkit` was rejected as a
grab-bag name that invites unrelated things to be added to it;
`webserver` was rejected as an overclaim, since the package contains no
server.

## Consequences

- A library whose accurate plain-noun name would be `util` or `common`
  is a library that should not exist yet. The naming rule doubles as a
  cohesion check.
- Package name matches the directory: `middleware.Chain(...)` reads at
  the call site, which is where the name is actually spent.
