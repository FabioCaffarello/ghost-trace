# Services

This directory contains the service implementations of Ghost Trace.

## Status

No services have been implemented. The directory structure is established in advance to declare the intended service decomposition. Each service receives a placeholder README that describes its constitutional role.

## Service Roster

- [`ingestion/`](./ingestion/) — receives observations from producers; commits them to the primary event log.
- [`assertion-engine/`](./assertion-engine/) — derives operational constructs and hypothetical assertions from observations and enrichment.
- [`replay/`](./replay/) — supports phase-specific replay semantics; reconstructs historical state.
- [`graph/`](./graph/) — maintains graph projections; supports inferential analysis and independence checks.
- [`projections/`](./projections/) — generic infrastructure for materializing and refreshing projections.

## Implementation Language

Not yet decided. Conversations leading to this directory tentatively favored Go for backend services, but no RFC has formalized this choice. Implementation work begins after the Charter and the relevant portions of the Ontology are stable.
