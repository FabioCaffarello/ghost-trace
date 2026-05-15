# Projection Model

**Status:** Scaffold.

> This document specifies the architectural treatment of projections in Ghost Trace. The Charter draws a sharp line between substrate (governed by Invariant 2.1) and projection (rebuildable, disposable). This document specifies how projections relate to the substrate.

## Constitutional Anchors

- [Invariant 2.1, Boundary Conditions](../charter/constitutional-charter.md#21-observational-integrity). Projections are rebuildable from the substrate and may be recomputed, truncated, or replaced without violating Invariant 2.1.
- [Invariant 2.2 — Epistemic Separation](../charter/constitutional-charter.md#22-epistemic-separation). Projections may join across categories for query convenience, but the substrate they project from preserves category separation.

## What Is a Projection

A projection is any materialized view derived from the substrate. The substrate is the source of truth; the projection is convenience.

Projections in Ghost Trace include, at minimum:

- **Analytical projections.** Time-series and aggregate views over events, assertions, and signals. Used for operational dashboards and historical analysis.
- **Graph projections.** Relationship views over entities and their connections. Used for cluster analysis, coordination detection, and investigation traversal.
- **Index projections.** Lookup structures that accelerate access to specific records or paths in the provenance graph.
- **Narrative projections.** Synthesized summaries of assertion histories, derived from the provenance graph for operator consumption.

## Properties of Projections

Projections share the following architectural properties:

1. **Rebuildability.** A projection that is lost or corrupted is recomputed from the substrate. The substrate is sufficient.
2. **Disposability.** A projection may be truncated, recomputed, or replaced without affecting any constitutional property. The cost is operational (downtime, recomputation expense), not epistemic.
3. **Subordinate consistency.** Projections are eventually consistent with the substrate. The substrate is strongly consistent and ordered within itself.
4. **Versioned computation.** The logic that produces a projection is versioned. Recomputation under a new version produces a new projection; older versions may be retained for comparison.

## Open Questions

1. **Projection technology selection.** Candidate technologies for analytical (column-oriented analytical stores) and graph (graph databases) projections are deferred to RFCs.
2. **Concurrent projection versions.** When projection logic changes, does the system run old and new in parallel until validated, or does it cut over? Operational decision with architectural implications.
3. **Operator-visible projection identity.** When an operator queries a projection, do they see a version reference? Important for honest interpretation of results.

<!-- TODO: After analytical and graph technology choices are made via RFC, specify the concrete projection architecture. -->
