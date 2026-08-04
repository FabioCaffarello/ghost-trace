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
make numbers            # ~7 minutes, needs real browsers
```

Compare against the most recent manifest in `docs/results/`:

- detection rates inside its Wilson intervals
- p99 inside the 80ms budget
- cold start still `never_blocks: true`
- the memory benchmark without a >10% regression

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
