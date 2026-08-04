---
type: doc
name: glossary
description: Project terminology, type definitions, domain entities, and business rules
category: glossary
generated: 2026-08-04
status: filled
scaffoldVersion: "2.0.0"
---
# Glossary

Terms that mean something specific here.

**The six numbers** — detection rate per tier, false-positive rate,
p99 decision latency, time to confident decision, cold-start behaviour,
and memory/latency by concurrency × duration. The project's central
claim; they must reproduce.

**Absence** — a tier that did not run, a number not measured, a rate
with no data. Always `null` or an explicit entry in `absent_tiers`,
never `0`. This distinction is load-bearing.

**Score vs confidence** — how bot-like the interaction looks, and how
much evidence backs that. Separate fields so "nothing suspicious
observed" is distinguishable from "this looks human".

**Shadow decision** — in monitor mode the returned decision is always
`allow`; the shadow is what enforce mode would have done. A client
reading only `decision` measures nothing while the service is in
monitor mode.

**Tier** — one adversarial bot implementation, 1 through 6, ascending in
sophistication. Tiers 5 and 6 are seeded (`GT_SEED`).

**Bow** — the curvature multiplier in the humanised mouse path. Tier 5
is a sweep over it, and detection is a smooth function of it.

**Substrate** — the append-only event archive: SQLite index plus a
content-addressed blob store.

**Manifest** — a committed run of the six numbers under `docs/results/`,
carrying the commit, machine, seed and per-tier sample sizes.

**Fail-open** — the operating philosophy: telemetry loss is expected, an
unknown session still gets 202, and the detector never blocks a first
visit.
