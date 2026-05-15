# Projections Service

## Constitutional Role

Generic infrastructure for materializing and refreshing projections from the substrate. Projections are not bound by [Invariant 2.1](../../docs/charter/constitutional-charter.md#21-observational-integrity); they are rebuildable.

## Status

Not implemented.

## Required Properties

- Rebuildability: every projection can be rebuilt from the substrate alone.
- Versioning: projection logic is versioned. Rebuilds under a new version produce new projection instances.
- Visibility: an operator querying a projection can determine which version of the projection logic produced the result.
