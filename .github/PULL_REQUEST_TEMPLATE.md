<!--
The TITLE is what lands on main: pull requests are squash-merged, so the
title is the commit message and the text release automation reads. It
must be Conventional Commits — CI checks it.

    feat(r1.17): ...   fix(policy): ...   docs(r1.16b): ...
-->

## What changed, and why

<!-- The why matters more. The diff already says what. -->

## The six numbers

<!-- Required if this touches services/ or experiments/. -->

- [ ] `make numbers` run, and the rates sit inside the Wilson intervals
      of the last manifest in `docs/results/`
- [ ] Not applicable — this changes neither the service nor the harness

If a number moved on purpose, say which and why, and publish a manifest
(`make numbers-manifest`). A moved number with no explanation is
indistinguishable from a broken detector.

## Checks

- [ ] `make ci` green locally
- [ ] Generated artifacts regenerated and committed if their sources
      changed (`openapi.yaml`, `libs/genproto`, `contract/fixtures`)
- [ ] Goldens either unchanged, or regenerated deliberately and called
      out here — they freeze the bytes clients receive

## Anything a reviewer should push back on

<!-- Shortcuts taken, assumptions made, things you are unsure about.
     This section being empty is itself a claim. -->
