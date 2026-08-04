---
type: skill
name: The Numbers Invariant
description: Verify the six numbers after changing services/ or experiments/. Use when a published figure might move, or before publishing a run manifest
skillSlug: numbers-invariant
phases: [V, C]
generated: 2026-08-04
status: filled
scaffoldVersion: "2.0.0"
---

# The six numbers must reproduce

They are the project's central claim. A number nobody can reproduce is
the exact failure this repository exists to avoid.

## Required after any change to `services/`, `experiments/` or `schemas/`

```bash
make numbers            # ~7 minutes, needs real browsers; measures AND checks
```

The comparison against the most recent manifest in `docs/results/` is no
longer yours to do by eye — `make numbers` ends by running it, and
`make numbers-check` re-runs it against a run already on disk. It
enforces:

- detection rates inside the baseline's Wilson intervals, in **both**
  directions. A detector that suddenly catches more is as much a change
  to explain as one that catches less
- no tier present in the baseline missing from the run
- p99 inside the 80ms budget
- cold start still `never_blocks: true`
- a false-positive rate that had a value has not lost it
- the memory benchmark without a >10% regression, per cell

Read the printed table anyway. The check enforces the rules; it does not
tell you whether the run **meant** anything.

The baseline is the newest manifest **of the same topology** —
`GT_ENGINE_BASE` set makes a run `composed`, unset makes it `monolith`,
and the two are not comparable. Within that, it is the one with the
newest `provenance.generated_at`, not the last filename — manifest names carry a content hash, so sorting
them by name picks an arbitrary one. That mistake was made twice by
hand before the check existed, which is why it exists.

## Read the run honestly

- `absent_tiers` **must be empty**, or every entry explained. A tier
  that did not run is not a tier that found nothing.
- `false_positive_rate: null` is correct and expected. It is the number
  that governs the others and it has no data.
- `architecture: null` means number 6 was **not measured** — the
  container has no Go toolchain.
- Tiers 5 and 6 are seeded (`GT_SEED`, default `ghost-trace-v1`). A
  seeded run still drifts slightly: the browser's event dispatch and
  scheduling are not ours to seed.

## If a number moved

Say so in the PR, explicitly, and publish a manifest:

```bash
make numbers-manifest   # refuses a dirty tree, and a run recorded dirty
```

**Then re-read the prose around the number.** A re-baselined figure with
stale prose is a document that now lies — re-baselining tier 6 from 70%
to 100% also required deleting the sentence calling it "the current
frontier", because at 100% detection does not fail there.

And state the weakness the number hides: n=10 intervals reach down to
72%, and tier 6 is caught by `VALUE_INJECTED`, the one signal the policy
refuses to treat as categorical.
