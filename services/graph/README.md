# Graph Service

## Constitutional Role

Maintains graph projections derived from the substrate. Supports inferential analysis, cluster detection, and independence checks. The graph is a projection, not the substrate; it is rebuildable from the primary event log.

## Status

Not implemented.

## Required Properties

- Projection discipline: the graph reflects the substrate. Direct mutation of graph state outside the projection rebuild path is forbidden (per the [projection-model.md](../../docs/architecture/projection-model.md) projection-rebuildability discipline; the graph is a projection of the substrate event log, not a mutable store).
- Independence support: graph traversal must be capable of filtering by edge type (`derived_from` vs. `influenced_by`) to support independence analysis.
- Versioned projection logic.
