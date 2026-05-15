# Provenance Model

**Status:** Scaffold. Pending committee redaction after Invariants 2.3 and 2.4 are redacted.

> This document formalizes the structure of provenance in Ghost Trace. The Charter establishes provenance as structure, not metadata ([Charter §1](../charter/constitutional-charter.md#1-thesis)). This document specifies what that structure is.

## Two Forms of Provenance

The Charter distinguishes two forms of provenance that conventional systems do not separately track:

### Observational Provenance

Answers the question: **"From which observations does this assertion derive?"**

This is the form of provenance familiar from event-sourced systems and lineage-tracking pipelines. Every non-observation assertion in Ghost Trace declares, in its structure, the set of observations from which it was computed.

To be formalized:
- The representation of observation references in assertions.
- The transitivity rules: when assertion A is derived from assertion B, and B is derived from observations O₁ and O₂, A's observational provenance includes O₁ and O₂.
- The reconstructibility guarantee: given an assertion and the immutability of the event log, the path back to observations is reconstructible.

### Inferential Provenance

Answers the question: **"Which prior assertions influenced the formation of this one?"**

This is the form of provenance that conventional systems do not maintain. The Charter requires it because without it, the system cannot distinguish between belief grounded in independent evidence and belief inherited recursively from earlier conclusions.

To be formalized:
- The representation of influence references.
- The propagation rules: how influence transitively accumulates through chains of derivation.
- The relationship to evidential independence (Invariant 2.6 — pending): inferential provenance is the structural input to independence computation.

## The Provenance Graph

Provenance in Ghost Trace forms a directed acyclic graph. Nodes are records (observations, operational constructs, hypotheses, assertions). Edges are typed and come in at least two varieties:

- **`derived_from`** — observational-provenance edges.
- **`influenced_by`** — inferential-provenance edges.

Traversal of this graph supports:
- Reconstruction of the path from any assertion back to underlying observations (used by replay).
- Independence analysis: computing how much of an assertion's evidential support is grounded in observations vs. inherited from prior assertions.
- Forensic explanation: producing narratives over the graph for operator consumption.

The graph itself is, like everything else in the substrate, derivable from the primary event log. It may be materialized in a graph database as a projection, but the projection is not the substrate.

## Open Modeling Questions

1. **Granularity of derivation.** When a signal is computed over a window of thousands of events, does its observational provenance enumerate every contributing event, or reference the window definition? The former is precise but storage-heavy; the latter is compact but adds an indirection.
2. **Decay of influence.** Should inferential influence have a temporal decay, or does an assertion remain "influenced by" a hypothesis indefinitely? Decision pending.
3. **Influence at projection vs. substrate.** When a projection is rebuilt from the substrate, does its computation introduce influence edges? Or are influence edges only generated when influence is *operationally consequential*?
4. **Cross-domain provenance.** When Ghost Trace is applied to a domain other than its first (e.g., market integrity surveillance), do provenance edges cross domains, or is each domain a separate provenance subgraph?

<!-- TODO: After Invariant 2.3 is redacted, formalize the observational-provenance section with the structural requirement language used in the invariant. -->

<!-- TODO: After Invariant 2.4 is redacted, formalize the inferential-provenance section in the same way. -->

<!-- TODO: Coordinate with [`lifecycle-semantics.md`](./lifecycle-semantics.md) on how hypothesis lifecycle operations propagate through the provenance graph. -->
