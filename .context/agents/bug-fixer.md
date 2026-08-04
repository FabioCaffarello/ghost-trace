---
type: agent
name: Bug Fixer
description: Analyze bug reports and error messages
agentType: bug-fixer
phases: [E, V]
generated: 2026-08-04
status: filled
scaffoldVersion: "2.0.0"
---
## Start here

`docs/results/` says what was measured, on which commit, machine and
seed. Tiers 5 and 6 record a seed label per session
(`ghost-trace-v1:tier6_value_injection:3`) — **one session can be
replayed on its own**.

## The rule this repository keeps relearning

Every defect worth fixing lands with the test that would have caught it,
and that test must be shown **red** before it is shown green. A gate that
cannot fail is the failure mode found repeatedly here: a vacuous CI
green, a validator returning 0 unconditionally, a golden that froze one
run's random token.

## Before concluding "not reproducible"

Check whether the thing varies because a real browser is in the loop
(event dispatch and scheduling are not seeded) or because the generator
was unseeded. The first is expected; the second was audit finding M13.

Gate with the sensors in `.context/config/sensors.json`; they are all `make` targets.
