---
type: skill
name: Bug Investigation
description: Investigate bugs systematically and perform root cause analysis. Use when Investigating reported bugs, Diagnosing unexpected behavior, or Finding the root cause of issues
skillSlug: bug-investigation
phases: [E, V]
generated: 2026-08-04
status: filled
scaffoldVersion: "2.0.0"
---
1. **Find the run.** `docs/results/` records commit, machine and seed.
2. **Replay one session.** Tiers 5 and 6 store a seed label per row
   (`ghost-trace-v1:tier6_value_injection:3`); one session reproduces on
   its own.
3. **Separate the variances.** Seeded generator vs. real browser timing.
   The second is expected and not a bug.
4. **Write the failing test first**, at the wire level if it crosses the
   wire, and confirm it is red before fixing.
5. **Ask what else shares the shape.** Most defects here were one
   instance of a class: three fabricated enums, three git-coupled sync
   gates, two swallowed-failure loops.
