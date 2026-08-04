# Run manifests

Each file here is one execution of `python3 experiments/numbers.py`,
copied verbatim from `experiments/results/numbers.json`. They are named
`numbers-<date>-<commit8>.json`.

Until R1.14b the six numbers had **no committed provenance at all**.
The README quoted them, a local `numbers.json` produced them, and
nothing recorded which commit, which machine, or which sample sizes
they came from — so "reproduce the six numbers" was a claim no reader
could check and no author could re-check months later.

## What a manifest is, and is not

It **is** the record of a run: the numbers, plus a `provenance` block
naming the commit, whether the tree was dirty, the machine and its cpu
count, the run mode, and the sample size used for every tier.

It is **not** a baseline that later runs must match. These numbers move
with the machine — `p99` on a busy laptop is not `p99` on an idle one,
and number 6 is a concurrency benchmark. Comparing two manifests is
only meaningful when their provenance blocks say they are comparable.

Three things a manifest deliberately does not hide:

- `false_positive_rate: null` means no human capture exists. It is the
  honest state of the project and the number that governs every other
  one; a zero there would be a claim nobody has earned.
- `architecture: null` means number 6 was **not measured** — the
  experiments container has no Go toolchain.
- `absent_tiers` names every tier that produced nothing, with the
  reason. A tier that did not run is not a tier that found nothing.

## Publishing one

```bash
make numbers            # run it
make numbers-manifest   # validate, then publish here
```

Publishing refuses a dirty tree and refuses a run whose own
`provenance.git.dirty` is true: a number produced from uncommitted code
cannot be reproduced by anyone, including its author.

Every manifest satisfies
[`experiments/schema/numbers.schema.json`](../../experiments/schema/numbers.schema.json),
which `numbers.py` checks before it writes — so a malformed measurement
is never published in the first place.
