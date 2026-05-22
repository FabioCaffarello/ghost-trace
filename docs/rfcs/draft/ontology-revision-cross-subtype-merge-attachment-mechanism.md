# RFC — Cross-subtype merge extension-field attachment mechanism (§0126 follow-on)

- **Status:** discussion
- **Authors:** Ghost Trace committee
- **Date:** 2026-05-22
- **Type:** ontology-revision
- **Affects:** [`docs/ontology/entity-model.md`](../../ontology/entity-model.md) §Cross-subtype operations; [`schemas/events/v1/`](../../../schemas/events/v1/) per-subtype formation protos (potentially); [`docs/architecture/replay-model.md`](../../architecture/replay-model.md) (replay-traversal implications).

> RFCs in Ghost Trace are proposals subject to constitutional review. The [Constitutional Charter](../../charter/constitutional-charter.md) is the authoritative reference. RFCs that conflict with the Charter must either be rejected or accompanied by a corresponding charter amendment (see [`../../charter/amendments.md`](../../charter/amendments.md)).

## Summary

[`§0126`](../../charter/decision-log.md) closed the cross-subtype merge γ pair-table at 6/6 cells. Five cells (1, 2, 3, 4, 5) have a NON-CANONICAL ANTECEDENT — an antecedent whose subtype differs from the merged target — whose structural surface must be carried into the merged record per the per-cell rationale. Cell 6 has a similar (weaker) case for CR's pairwise structure on CH.

This RFC opens the question: **what is the structural mechanism for the non-canonical-antecedent surface attachment on the merged record?** Three candidate mechanisms are presented + combined forms; the RFC does not pick.

## Motivation

Per [`§0126`](../../charter/decision-log.md) Decision + carry-forward: each cell's resolution names a structural attachment for the non-canonical-antecedent surface:

| Cell | Target | Non-canonical antecedent surface needing attachment |
|---|---|---|
| 1 `{BC, AG} → AG` | AG | BC's pattern-signature (operatorship-signature reference) |
| 2 `{BC, CR} → CR` | CR | BC's pattern-signature (operatorship-signature reference) |
| 3 `{BC, CH} → CH` | CH | BC's pattern-signature + actor-set attribution |
| 4 `{AG, CR} → CR` | CR | AG's automation-signature reference (per [`§0125`](../../charter/decision-log.md)) |
| 5 `{AG, CH} → CH` | CH | AG's automation-signature + actor-set attribution |
| 6 `{CR, CH} → CH` | CH | CR's pairwise actor-relationship records |

Without a committee-defended mechanism, the attachment surface becomes infrastructure-resolved at first implementation. Three failure modes appear:

- **Per-cell ad-hoc attachment.** Each cell's helper implements its own mechanism (e.g., Cell 1 uses a proto field on AG; Cell 2 uses a separate record; Cell 6 derives implicitly from antecedents). Asymmetric mechanisms across cells violate the §2.5 BC1 channel-agnosticism analog at the cell layer.
- **Silent surface loss.** A helper implementation that "drops" the non-canonical surface (e.g., Cell 6's CR pairwise structure not carried) produces a merged record that the per-cell rationale explicitly required to preserve. Structurally undetectable without the mechanism specified.
- **Replay non-determinism.** Phase 3 reconstructive replay needs to reproduce the merged record from antecedents; without a specified mechanism, the replay path is ambiguous (does the merged record include the non-canonical surface? if yes, where?).

The cost of not resolving the mechanism: §0126's commitment to "non-canonical surface attachment" remains structurally hollow until implementation. The §0049 Option B substrate-immutability + Phase 3 replay surfaces require the mechanism BEFORE the first cross-subtype merge commit.

## Constitutional Review

The Q1–Q6 impact analysis prescribed by [`rfc-author` §1](../../../.claude/skills/workflow/rfc-author/SKILL.md).

### Q1 — Which Charter invariants does this RFC touch?

- **§2.1 Observational Integrity** (frozen): touched at the substrate-immutability layer. All three candidates respect §2.1 — the merged record + any attachment are committed once and not mutated. Candidate β adds a new Cat I record type (the attachment record); this is an extension of the existing typed-protos pattern per [`§0042`](../../charter/decision-log.md), not a §2.1 amendment.
- **§2.3 Provenance Integrity** (frozen v0.4): touched at the provenance-chain layer. The non-canonical-antecedent surface IS provenance per [`§0016`](../../charter/decision-log.md) Q3 typed-reference resolution — the merged record's "carries the BC's signature" claim is a §2.3 BC5 multi-category-traversal-shape commitment. Each candidate carries provenance differently (α via proto field; β via separately-committed record; γ via antecedent traversal).
- **§2.5 Hypothesis Lifecycle Explicitness** (frozen v0.3): touched at BC1. The merged record IS the lifecycle event per §2.5; the attachment mechanism does not change the event's category (Cat I per §2.5 BC1). Each candidate operationalizes within §2.5 unchanged.
- **§4 Constitutional Design Rule** (frozen v0.2): consistency check, not amendment. Falsifiability discipline applied per candidate.
- No FROZEN invariant is amended.

### Q2 — Does this RFC implicitly redefine any term in the glossary?

No new canonical vocabulary. Candidate β introduces a new Cat I proto name (`CrossSubtypeMergeAttachment` or analogous); this is per-proto naming, not canonical-glossary-term introduction (analogous to the [`§0104`](../../charter/decision-log.md) `OrphanCleanupAudit` proto naming — not in glossary).

### Q3 — Does this RFC implicitly resolve any of the five open Ontology questions?

This RFC is a §0126 follow-on; it does not resolve any of the five canonical OQs. It interacts indirectly with Q2 (Identity Tiers Open Modeling Question) — the BC pattern-signature reference (Cells 1, 2, 3) carries actor_ref identity; identity-tier evolution may extend the signature's payload but does not change the attachment-mechanism question.

### Q4 — Does this RFC require Charter amendment?

No. All three candidates operationalize within frozen Charter sections.

### Q5 — Does this RFC introduce a new invariant?

No. Codifies an implementation mechanism for an already-resolved Ontology commitment (§0126 cells).

### Q6 — Does this RFC propose ceremony without behavioral consequence?

No. Different mechanisms produce different operator-visible surfaces (proto layer, projection-layer queries, replay-handler shape) and different downstream consumer obligations (read-time joins under β vs direct field-access under α vs antecedent-traversal under γ). The mechanisms partition the operator-visible surface differently.

## Proposal

Three candidate mechanisms, each presented with structural claim, dependency, pros and cons. Combined forms considered.

### Candidate α — Field-on-target-proto

**Structural claim.** Each target subtype's formation proto gains explicit field(s) for the non-canonical-antecedent surface(s). Per the §0126 cell-table:

- `BehavioralClusterFormation`: no changes (BC is never a target).
- `AutomationGroupFormation`: + `operatorship_signature_ref: bytes` (Cell 1's BC signature).
- `CoordinationRingFormation`: + `operatorship_signature_ref` (Cell 2) + `automation_signature_ref` (Cell 4).
- `CampaignHypothesisFormation`: + `actor_signature_attachments` (Cells 3 + 5) + `derived_pairwise_relationships` (Cell 6).

Field values are populated at cross-subtype merge time (the merge helper extracts the non-canonical antecedent's surface and copies into the merged record); for within-subtype merges + recognition-pattern-output formations, the fields are unset (proto3 default).

**Dependency.** Proto-evolution event per the schemas-evolution Cat I record convention (per [`§0034`](../../charter/decision-log.md) canonical-serialization-contract). The field additions are forward-compatible per proto3 optional semantics.

**Pros.**

- Type-system-explicit. Each attachment is a typed field on a typed record; consumers see the attachment surface at the same locality as the canonical record.
- §1.3 falsifiability direct: a cross-subtype merge whose merged record has the expected target type AND the expected attachment field populated is structurally checkable.
- Zero read-time joins. Consumer reading the merged record sees the full surface in one record fetch.

**Cons.**

- Proto-evolution churn: 4 protos gain 1–4 new fields total. Each addition is a proto-version commit + corpus regeneration per the [`§0024`](../../charter/decision-log.md) canonical-serialization-contract event.
- Per-cell field naming: each attachment needs a per-cell field name (`operatorship_signature_ref` vs `automation_signature_ref` etc.). The field names are committee-defended; the naming convention is itself a sub-question.
- Forward-incompatible with future cells: if Cell 4 were ever re-resolved to AG, the `automation_signature_ref` on CR becomes vestigial. Per §2.1 immutability, prior records retain the field; new records leave it default.

### Candidate β — Separately-committed attachment record

**Structural claim.** Introduce a new Cat I proto `CrossSubtypeMergeAttachment` carrying the non-canonical-antecedent surface as a typed payload referencing the merged record. Committed via `substrate.AppendPair` alongside the merged formation (analogous to the [`§0104`](../../charter/decision-log.md) audit-then-delete pattern's audit + IngestionEvent pairing).

Schema:

```
message CrossSubtypeMergeAttachment {
  bytes merged_formation_hash = 1;    // the §0126-resolved target
  bytes antecedent_formation_hash = 2; // which antecedent's surface this attaches
  string surface_type = 3;             // "operatorship_signature" / "automation_signature" / "pairwise_relationships"
  bytes surface_payload = 4;           // canonicalized payload from the antecedent
}
```

The target protos (`AutomationGroupFormation`, `CoordinationRingFormation`, `CampaignHypothesisFormation`) are UNCHANGED. Consumers needing the non-canonical surface query the substrate for `CrossSubtypeMergeAttachment` records referencing the merged formation hash.

**Dependency.** Schemas-evolution event for the new proto. Mirrors the [`§0042`](../../charter/decision-log.md) typed-Cat-I-protos pattern (NetworkEvent, ClientStateEvent) + [`§0104`](../../charter/decision-log.md) (OrphanCleanupAudit) — adding a new Cat I record type is established discipline.

**Pros.**

- Target protos unchanged. The four formation protos remain at their current shape; no proto-evolution churn beyond the new attachment proto itself.
- Uniform mechanism across all cells. Cells 1–6 all use the same attachment record type; no per-cell field naming.
- Forward-extensible: future cells (e.g., from hypothetical new subtypes) add new `surface_type` values without proto change.
- Provenance is structurally explicit: each attachment record IS a §2.3 provenance edge from the merged record back to the non-canonical antecedent.

**Cons.**

- Read-time joins required: consumers needing the non-canonical surface query for attachment records. Projection layer needs to materialize the joined view.
- §1.3 falsifiability indirect: the helper-layer structural check is "merged record committed AND attachment record committed atomically"; the attachment's surface_payload validity (is it the actual antecedent's signature?) is harder to check structurally.
- Three new Cat I record types under the pattern, OR one polymorphic `CrossSubtypeMergeAttachment` with a `surface_type` discriminator. The discriminator path repeats the [`§0042`](../../charter/decision-log.md) "kind field" concern at the attachment layer.

### Candidate γ — Inferred-from-antecedent-structure

**Structural claim.** No explicit attachment. The merge event records the antecedent formation hashes; consumers retrieve non-canonical-antecedent surfaces by following the antecedent references at read time. The merged formation record contains ONLY its canonical surface (matching its target subtype's existing proto). The "BC signature attached to AG" claim is structurally encoded by the merge event's antecedent-list (the BC's formation hash is one antecedent; reading that record provides the signature).

**Dependency.** None on proto changes. The existing merge event proto already references both antecedent formation hashes per [`§0049`](../../charter/decision-log.md) Option B.

**Pros.**

- Zero new structural surface. No proto additions; no new Cat I record type.
- Maximally substrate-minimal: the merged record carries only what its canonical surface natively carries.
- Forward-extensible without any new structure: future cells inherit the same "follow antecedents" mechanism.

**Cons.**

- Read-time traversal required for every cross-subtype-merge consumer: to know "what signature does this CR-from-{BC, CR}-merge carry?", consumer must traverse the merge event → BC antecedent → BC's signature.
- §1.3 falsifiability weakest: there's no structural commitment to attach; the "attachment" exists only in consumer interpretation. A consumer that fails to traverse antecedents misses the surface silently.
- Cell 6 specifically problematic: CR's pairwise structure under CH target is itself derivable from event co-occurrence per the §0126 rationale; under γ, the derivation is consumer-side. Multiple consumers may derive differently.
- Projection-layer obligation: projections wanting the merged-with-surface view must implement the traversal + derivation logic uniformly across consumers.

### Combined-candidate forms

- **α + β.** High-frequency cells use α (proto field); low-frequency cells use β (attachment record). The committee chooses the boundary; per-cell defense for each path.
- **α + γ.** Frequently-accessed surfaces use α (field on target); less-load-bearing surfaces use γ (antecedent traversal). Cell 6's CR-pairwise-on-CH might naturally fall to γ (derivable from events anyway).
- **β + γ.** Surfaces that need structural commitment use β (typed attachment); surfaces that are derivable (Cell 6) use γ.

The discussion phase considers combinations.

## Alternatives Considered

### Cell-by-cell mechanism choice (META-PATTERN, NOT REJECTED)

Each cell picks its own mechanism (Cell 1 uses α; Cell 4 uses β; Cell 6 uses γ). Operationally tempting (per-cell optimization); structurally surfaces the §0126 "asymmetric per-cell criterion" anti-pattern at the implementation layer. The committee may choose this if per-cell defense is committee-defensible (combined-candidate α+β+γ is the same shape framed differently). Recorded as the legitimate "combined" framing.

### Skip the attachment entirely (REJECTED)

The merged record carries only its canonical surface; the non-canonical antecedent's surface is structurally discarded at merge time. Rejected because it conflicts with the §0126 per-cell rationale — each cell explicitly required the non-canonical surface to be preserved per the "carry as extension surface" framing. Skipping would be silent surface loss, the failure mode the RFC exists to prevent.

## Open Questions

- **Read-time projection-layer shape.** Under β, projections must materialize the merged-record + attachments view. Whether projections eagerly materialize the join OR lazily query on consumer request is itself a follow-on question.
- **Surface payload canonicalization.** Per [`§0034`](../../charter/decision-log.md), all substrate records use canonical-serialization. Under β, the `surface_payload` field's canonicalization must match the antecedent's original payload byte-for-byte (otherwise the attachment carries a re-serialized version that drifts from the antecedent). Open: enforce byte-for-byte equality, or accept canonical-form equality?
- **Per-cell vs uniform mechanism.** The combined-candidate form raises this; the committee may resolve uniformly OR per-cell.
- **Cell 6's derivation-vs-explicit-attachment.** CR's pairwise structure under CH target is derivable from event co-occurrence; under α + β, the merged record carries it explicitly; under γ, the derivation is consumer-side. Whether the structural commitment is "the merged record carries the pairwise structure" OR "the merged record carries enough to derive the pairwise structure" is itself a sub-question.

## Anti-Patterns to Avoid

- **Resolving the mechanism by implementation.** Hard-coded attachment mechanism in a `MergeAcrossSubtypes` helper without RFC. The mechanism belongs to the Ontology + proto layers, not to operational code.
- **Asymmetric mechanisms across cells without committee defense.** Even under combined-candidate forms, each per-cell mechanism choice needs the same defense as the unified-mechanism path.
- **Silent surface loss.** A helper implementation that drops the non-canonical surface produces a merged record violating the §0126 per-cell rationale. The mechanism must structurally prevent silent loss (under α: field-on-target ensures the field's absence is detectable; under β: missing attachment record is detectable; under γ: the consumer must traverse antecedents).
- **Re-serializing antecedent surface under β.** If the `surface_payload` is re-canonicalized rather than copied byte-for-byte from the antecedent's original payload, the attachment claim "this IS the antecedent's signature" becomes structurally weaker (consumer must accept the helper's re-canonicalization as faithful). The canonicalization question above raises this.

## Migration and Backward Compatibility

No historical cross-subtype merge records exist; the mechanism is forward-looking.

Lock-in asymmetry:

- **α** has the heaviest forward lock-in: each proto-field addition is a [`§0024`](../../charter/decision-log.md) canonical-serialization-contract evolution event; removing a field later requires deprecation discipline.
- **β** is moderate: a new proto is added once; per-cell extensions reuse the proto.
- **γ** has the lightest lock-in: no new structure; the mechanism evolves entirely in consumer projections.

[Phase 3 / Phase 4 replay contracts](../../architecture/replay-model.md):

- Under α: replay reconstructs the merged record with field-level expected values; deterministic if the helper extracts antecedent surface byte-for-byte.
- Under β: replay reconstructs the merged formation + the attachment record(s); the attachment + the merge form a unit; both must replay deterministically.
- Under γ: replay reconstructs only the merged formation; the antecedent traversal is consumer-side, not part of replay.

## References

- [`docs/charter/decision-log.md` §0122](../../charter/decision-log.md) — cross-subtype merge typing γ resolution.
- [`docs/charter/decision-log.md` §0125 + §0126](../../charter/decision-log.md) — pair-table cells resolved; this RFC discharges the unified-attachment-mechanism carry-forward.
- [`docs/charter/decision-log.md` §0042](../../charter/decision-log.md) — initial deferred types rendered into Cat I protos (NetworkEvent, ClientStateEvent); typed-Cat-I-protos pattern precedent for Candidate β.
- [`docs/charter/decision-log.md` §0104](../../charter/decision-log.md) — T3 OrphanCleanupAudit + AppendPair pairing; precedent for Candidate β.
- [`docs/charter/decision-log.md` §0034](../../charter/decision-log.md) — canonical-serialization-contract; relevant to all three candidates' payload-canonicalization question.
- [`docs/charter/decision-log.md` §0049](../../charter/decision-log.md) — Option B merge-as-separately-committed-formation; merge event proto already carries antecedent references for Candidate γ.
- [`docs/architecture/replay-model.md`](../../architecture/replay-model.md) — Phase 3 reconstructive replay; differs per candidate.
- [`docs/rfcs/draft/ontology-revision-cross-subtype-merge-pair-table.md`](./ontology-revision-cross-subtype-merge-pair-table.md) — pair-table RFC; §0126 carry-forward named this mechanism question.

## Decision Record

Pending. The discussion phase opens with this RFC at `Status: discussion`; resolution will be recorded in `docs/charter/decision-log.md` and reflected back here on acceptance. The natural follow-on is a paired evidence document per the established RFC + evidence cadence (q4-evidence / cross-subtype-merge-typing-evidence pattern).
