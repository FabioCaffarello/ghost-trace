# 0004 — Session snapshots carry feature state, not events

**Status:** accepted · **Date:** 2026-08-04 · **Milestone:** PR-2.3a

## Context

Phase 2 moves `/v1/decisions` into its own service. That service does
not observe the session, so the collector must hand it whatever a
judgement needs.

## Decision

**The snapshot carries FEATURE STATE — the nineteen numbers the
detector actually scores — and never raw events.**

`SessionSnapshot` wraps the existing `FeatureState` message rather than
declaring a second shape, because the archived evaluation already
carries that message and two definitions of "what the detector looked
at" would be free to drift.

The collector writes a snapshot; the decision engine reads the latest
one. That is the semantics the monolith already has — decide on the
most recent state — expressed across a process boundary instead of a
mutex.

## Why feature state

The decision path already reduces a session to nineteen numbers before
scoring it. Carrying those makes the snapshot small and bounded no
matter how long the session runs, and it keeps the collector the single
writer: nothing downstream needs the event history to reproduce a
judgement.

Carrying raw events instead would make the payload grow without limit,
put event interpretation in two places, and hand the decision engine a
second chance to disagree about what the events meant.

## The precision question, measured

`FeatureState` stores the ratios as `float32`; the domain computes them
in `float64`. The round trip is therefore **not** the identity, which
audit finding M10 raised and left open.

It is now measured rather than argued. `libs/snapshot` holds both
directions of the mapping side by side, and its tests judge the same
state in process and through a snapshot:

- 2000 randomly generated states, plus hand-picked ones sitting on the
  straightness floor: **every decision identical**
- score and confidence drift stays under `1e-6` relative, consistent
  with float32's ~7 decimal digits
- a separate test asserts every field survives, using values exactly
  representable in float32 — so a dropped or crossed field fails there
  rather than hiding until a session where it moves the score

**The narrowing does not change decisions.** It is kept, because
widening the wire to double would be a breaking schema change bought
for a difference nothing can observe.

## Consequences

- Both mappings live in one package. Written apart they would each pass
  their own tests and disagree with each other; written together, the
  round trip is one assertion away.
- `ToState` does no defaulting or inference. A field the producer did
  not set arrives as zero and the judgement sees zero — guessing would
  make a snapshot judge differently from the session it describes.
- If a future feature needs precision beyond float32, that is a schema
  change with a migration, not a quiet type widening.

## Revisit

If a feature is added whose meaningful range needs more than seven
significant digits, or if the equivalence test ever fails.
