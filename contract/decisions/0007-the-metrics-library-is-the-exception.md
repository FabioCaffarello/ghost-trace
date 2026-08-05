# 0007 — prometheus/client_golang is the exception to stdlib-only

**Status:** accepted · **Date:** 2026-08-05 · **Milestone:** PR-3.1

## Context

Two rules in this repository point in opposite directions here.

`libs/middleware/metrics.go` said, in its own doc comment, where its
limit was:

> Two metric types with two effective labels each is under the threshold
> where prometheus/client_golang starts paying for itself; the moment a
> third metric type or a dynamic label is wanted, adopt the library
> instead of growing this.

ADR-0001 says the opposite about shared libraries:

> Shared libraries are stdlib-only where possible. A shared library that
> drags dependencies into four services is a coupling nobody asked for.

Phase 3 crosses the first threshold on both counts. Loss accounting
needs a **gauge** (consumer lag) and a **counter labelled by drop
reason**, which is a third metric type and a dynamic label.

## Decision

**Adopt `prometheus/client_golang`, in a new `libs/metrics` that owns
the process's one registry.** `libs/middleware` registers the HTTP
series into it and no longer encodes anything.

## Why the first rule wins

Because of what this phase is *for*. It exists to make what the system
loses countable. Hand-writing the counting apparatus — label
cardinality, escaping, concurrent maps, exposition edge cases — would
leave the instrument as the least-tested part of the measurement, and
every number it produced would carry that doubt.

The threshold comment is also the more trustworthy of the two rules
here, not because it is newer but because of *when* it was written: by
someone with no stake in the outcome, before the case arose. A rule
written in advance and honoured when it becomes inconvenient is the only
kind worth writing.

ADR-0001's concern is real and is accepted rather than dismissed: four
services gain a dependency, `govulncheck` and Trivy gain a surface, and
the shared-library tree is no longer stdlib-only. That is the price.

## Consequences

- **The exposition did not change at all.** Same series, same label
  names, same label order, byte for byte for these families.

  This was expected to be false. The prediction going in was that the
  library would sort labels and turn `{route=...,le=...}` into
  `{le=...,route=...}`, and both this record and the code comment said
  so before anyone checked. `le` is appended as a special case, so the
  four exact-string assertions the old test made all still pass.
  `TestExpositionDidNotChange` keeps them, because "the swap was
  invisible to every consumer" is a stronger claim than "the swap
  preserved the names" and it costs nothing to hold.

  The series-reading assertions were kept as well, for the cases where
  label order genuinely is not the point.
- **Roughly 90 lines of hand-written exposition encoding are deleted**,
  including the label escaper and the deterministic-ordering code. Their
  behaviour is now the library's problem, which is the trade.
- `prometheus.DefaultRegisterer` is deliberately **not** used. A global
  registry makes one test's series visible to the next and makes two
  services in one process share state; every registry here is passed
  explicitly.
- Histogram buckets are never defaulted. `prometheus.DefBuckets` spans
  5ms to 10s, which describes neither a sub-millisecond decision nor a
  seven-second experiment; a histogram whose bounds do not straddle what
  is measured puts everything in one bucket and looks like it works.

## The rule that replaces the old threshold

The comment that told us to adopt the library has been honoured and is
gone. What replaces it is narrower: **`libs/metrics` is the only shared
library allowed a non-stdlib dependency, and it is allowed exactly one.**
The next library that wants a dependency does not get to cite this ADR
— it gets to write its own.
