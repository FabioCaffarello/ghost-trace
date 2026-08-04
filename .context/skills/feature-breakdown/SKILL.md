---
type: skill
name: Feature Breakdown
description: Break down features into implementable tasks. Use when Planning new feature implementation, Breaking large tasks into smaller pieces, or Creating implementation roadmap
skillSlug: feature-breakdown
phases: [P]
generated: 2026-08-04
status: filled
scaffoldVersion: "2.0.0"
---
One PR per advance, reviewable alone.

A good slice here: lands with the test that would have caught what it
fixes; touches one concern; regenerates whatever it invalidates; and
says in the PR what it **found**.

Sequence anything that changes the wire as: types → vocabularies →
harness modules → regenerate → goldens → `make ci` → `make numbers`.
That order is the wire-contract skill and skipping a step is how a
silent degradation ships.
