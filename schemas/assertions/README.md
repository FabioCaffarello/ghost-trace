# Assertion Schemas

Schemas for Category II (operational constructs) and Category III (hypotheses) records.

## Status

No dedicated assertion protos exist in this directory. [Invariant 2.3](../../docs/charter/constitutional-charter.md#23-provenance-integrity) (frozen v0.4 per [`decision-log §0017`](../../docs/charter/decision-log.md)), [§2.4](../../docs/charter/constitutional-charter.md#24-inferential-influence-disclosure) (frozen v0.5 per [`§0099`](../../docs/charter/decision-log.md)), [§2.5](../../docs/charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) (frozen v0.3 per [`§0013`](../../docs/charter/decision-log.md)), and [§2.6](../../docs/charter/constitutional-charter.md#26-evidential-independence-integrity) (frozen v0.6 per [`§0129`](../../docs/charter/decision-log.md)) are all frozen; the Ontology has stabilized as Drafted per [`docs/architecture/README.md`](../../docs/architecture/README.md). The structural commitments these invariants codify for Category II (operational constructs) and Category III (hypotheses) are presently materialized inline in the [`events/v1/`](../events/v1/) protos (one proto per Cat II + Cat III event variety — e.g., `operational_session.proto`, `automation_group_formation.proto`, `behavioral_cluster_promotion.proto`, etc., totalling 25 Cat II + Cat III protos as of [`§0140`](../../docs/charter/decision-log.md)). Dedicated assertion-typed protos may surface here as a future schema-evolution event if a future RFC factors them out of the inline event form; until then, this directory remains an organizational placeholder.

## Required Properties

When dedicated assertion protos surface here, they must support:

- Strict separation between Category II and Category III at the type level.
- Observational provenance references ([Invariant 2.3](../../docs/charter/constitutional-charter.md#23-provenance-integrity) frozen v0.4 per [`decision-log §0017`](../../docs/charter/decision-log.md)).
- Inferential influence declarations ([Invariant 2.4](../../docs/charter/constitutional-charter.md#24-inferential-influence-disclosure) frozen v0.5 per [`decision-log §0099`](../../docs/charter/decision-log.md)).
- For Category III specifically: lifecycle event references ([Invariant 2.5](../../docs/charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) frozen v0.3 per [`decision-log §0013`](../../docs/charter/decision-log.md)), and paired confidence + independence dimensions ([Invariant 2.6](../../docs/charter/constitutional-charter.md#26-evidential-independence-integrity) frozen v0.6 per [`decision-log §0129`](../../docs/charter/decision-log.md)).

See [`../README.md`](../README.md) for constitutional anchors.
