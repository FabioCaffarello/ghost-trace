# Replay Model

**Status:** Scaffold.

> This document specifies the architectural treatment of replay in Ghost Trace. The Charter establishes that "the historical record of what the system believed, and on what basis, at any prior moment, is preserved not as audit log but as first-class data" ([Charter §1](../charter/constitutional-charter.md#1-thesis)). This document specifies how that preservation is operationalized.

## Constitutional Anchors

- [Invariant 2.1 — Observational Integrity](../charter/constitutional-charter.md#21-observational-integrity). Replay is meaningful only because the observational substrate is immutable.
- Invariant 2.3 — Provenance Integrity (pending). Replay traverses provenance to reconstruct the path from assertions to observations.
- Invariant 2.4 — Inferential Influence Disclosure (pending). Replay must preserve the distinction between belief grounded in observation and belief inherited from prior assertions.

## Replay Phases

The Charter recognizes that knowledge in Ghost Trace evolves through phases of increasing context. Replay semantics differ by phase:

### Phase 1 replay — deterministic over observation alone

Phase 1 assertions depend only on the observations that produced them and the version of the rules under which they were computed. Replay is deterministic: re-running the same rules over the same observations yields the same Phase 1 assertions.

### Phase 2 replay — deterministic given a temporal enrichment snapshot

Phase 2 assertions depend on enrichment as it existed at the time of original computation. Replay requires the enrichment-snapshot-as-of-T₁, not the current enrichment state. This is the structural reason enrichment is recorded as immutable events rather than as a mutable lookup.

### Phase 3 replay — reconstructive

Phase 3 assertions depend on global graph state (hypotheses, clusters, relationships) as it existed at the time of original computation. Replaying these assertions exactly may require versioned graph snapshots. As a pragmatic alternative, the *result* of Phase 3 analysis is preserved as an immutable assertion with explicit evidence, and re-deriving Phase 3 from scratch is acknowledged to potentially yield a different result. What is preserved is the historical truth of *what was concluded*.

**Scope.** The Phase 3 contract applies to **pattern-driven derivations** whose output depends on global graph state — Category III hypothesis formations (which carry an explicit `pattern_signature` + `pattern_parameters`) and, prospectively, Category II constructs whose derivation rule reads graph state. Category III **lifecycle events** (`promotion`, `demotion`, `dissolution`, `merge`, `split` per [Charter §2.5](../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness)) are operator-elected commits without a pattern-driven derivation; they are **outside the Phase 3 scope** per [`decision-log.md` §0092](../charter/decision-log.md). Substrate-integrity verification of their predecessor references is subsumed by the existing substrate-audit tooling (`cmd/verify`); substrate-state inspection at op commit time is a derived capability of any substrate reader applying the `committed_at ≤ T` filter — not a distinct replay contract. The reversal condition is proto-level: a future lifecycle-op proto field whose value is deterministically derived from substrate state at commit time would reopen the assessment for that op.

### Phase 4 replay — retrospective analysis

Phase 4 is analytical rather than operational. Replay in this phase means running new analyses over historical data with new models. The purpose is not reproduction; it is discovery.

## Replay Contracts

The system declares, per phase, the strength of replay guarantee it provides:

- **Deterministic replay** (Phases 1–2): exact reproduction given the substrate and the version of computation logic.
- **Reconstructive replay** (Phase 3): preservation of conclusions with their evidence at the time of decision.
- **Retrospective analysis** (Phase 4): no reproduction claim; the substrate supports new analyses, not the reproduction of prior ones.

These are distinct contracts. Implementations must not conflate them.

## Open Questions

1. **Graph snapshot frequency.** Phase 3 reconstructive replay requires snapshots of the graph state. The frequency and storage cost of these snapshots is undecided.
2. **Hot vs. cold replay.** Replay within the operational retention window is fast; replay from cold archive is slow. How these are exposed to operators is a UX question with architectural implications.
3. **Computation versioning.** Replay requires knowing the version of computation logic in effect at the time. How computation versions are recorded and referenced is undecided.

<!-- TODO: After storage decisions are made, specify the concrete mechanisms by which each replay phase is supported. -->
