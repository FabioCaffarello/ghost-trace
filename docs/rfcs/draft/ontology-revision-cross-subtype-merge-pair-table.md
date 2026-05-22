# RFC — Cross-subtype merge pair-table contents (§0122 follow-on, on hold)

- **Status:** discussion (on hold pending operational pressure or committee direction)
- **Authors:** Ghost Trace committee
- **Date:** 2026-05-22
- **Type:** ontology-revision
- **Affects:** [`docs/ontology/entity-model.md`](../../ontology/entity-model.md) §Cross-subtype operations (the §0122-deferred per-pair canonical-target table).

> RFCs in Ghost Trace are proposals subject to constitutional review. The [Constitutional Charter](../../charter/constitutional-charter.md) is the authoritative reference. RFCs that conflict with the Charter must either be rejected or accompanied by a corresponding charter amendment (see [`../../charter/amendments.md`](../../charter/amendments.md)).

## Status note

**This RFC is a placeholder.** Its substantive content — committee defense for each of the 6 unordered pair-table cells — is deferred per [`decision-log §0122`](../../charter/decision-log.md) (cross-subtype merge typing resolution, form-adopt + parameters-defer). The RFC is opened now so that the deferred question has a canonical referent in the RFC corpus; per the [`§0011`](../../charter/decision-log.md) Layer-B-placeholder precedent, the form resolution is procedurally complete without the parameters, but the parameters need a documented home to avoid silent-resolution.

## Summary

[`decision-log §0122`](../../charter/decision-log.md) resolved the cross-subtype merge produced-record typing question as **Candidate γ — per-pair canonical-merge typing**. The structural commitment is: no new typed subtype is introduced; cross-subtype merge produces a record of one of the four existing concrete subtypes (`BehavioralCluster`, `AutomationGroup`, `CampaignHypothesis`, `CoordinationRing`), determined by a per-pair entity-model-level table. The 6 unordered pair-table cells — `{BC, AG}`, `{BC, CR}`, `{BC, CH}`, `{AG, CR}`, `{AG, CH}`, `{CR, CH}` — each map to one of the four existing subtypes; this RFC is the locus of the per-cell committee defense.

The pair-table is a versioned operational-definition constant per [`§0021`](../../charter/decision-log.md) substrate-time-generation pattern; its content is structural commitment, not implementation detail.

## Motivation

The [`§0122`](../../charter/decision-log.md) form-adopt + parameters-defer resolution committed to γ without specifying the table contents in order to permit the merge typing question to close at the form level without binding the committee to specific per-pair choices that may require operational evidence to defend. Without this follow-on RFC, the table contents would lack a canonical locus and would risk silent resolution either by (a) implementation work that picks per-pair targets at first need, (b) ad-hoc decision-log entries that decide cells piecemeal without cross-cell coherence review, or (c) the [`§0122`](../../charter/decision-log.md) illustrative table being adopted by default through repeated reference.

The cost of not opening this placeholder: the 6 cells would become infrastructure-resolved one-by-one as operational pressure surfaces, precisely the [`ontology-keeper`](../../../.claude/skills/ontology/ontology-keeper/SKILL.md) failure mode the form-adopt + parameters-defer pattern is meant to forestall.

## Constitutional Review

The Q1–Q6 impact analysis prescribed by [`rfc-author` §1](../../../.claude/skills/workflow/rfc-author/SKILL.md). Compressed per the placeholder shape.

### Q1 — Which Charter invariants does this RFC touch?

None at amendment time. The pair-table is an entity-model-level structural rule, not a Charter invariant. [`§2.5 frozen v0.3`](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) BC1 channel-agnostic lifecycle surface accommodates any pair-table content; the merge event itself is the §2.5 lifecycle event regardless of cell choices. No FROZEN invariant is amended.

### Q2 — Does this RFC implicitly redefine any term in the glossary?

No new canonical vocabulary anticipated. The four concrete subtype terms (`BehavioralCluster`, `AutomationGroup`, `CampaignHypothesis`, `CoordinationRing`) are already glossary-tracked; the table maps among them without introducing new terms.

### Q3 — Does this RFC implicitly resolve any of the five open Ontology questions?

This RFC is itself a Q2-A.2 follow-on; it does not resolve any of the five canonical OQs from [`ontology.md`](../../ontology/ontology.md).

### Q4 — Does this RFC require Charter amendment?

No. The pair-table is downstream of all currently-frozen Charter sections; the resolution operationalizes within them.

### Q5 — Does this RFC introduce a new invariant?

No. Codifies parameters of an already-resolved Ontology commitment (γ form at §0122).

### Q6 — Does this RFC propose ceremony without behavioral consequence?

No. Different per-cell choices produce different operator-visible behavior: a {BC, CR} merge under `{BC, CR} → CR` produces a `CoordinationRingFormation`; under `{BC, CR} → BC` produces a `BehavioralClusterFormation`. The downstream replay handler, projection-layer surface, and §2.4 `subject_ref_hypothesis` payload all carry the choice. The cells are structural commitments, not cosmetic ones.

## Scope

Six unordered pair cells require committee defense:

| Cell | Antecedent A | Antecedent B | Proposed target | Defense status |
|---|---|---|---|---|
| 1 | BehavioralCluster | AutomationGroup | — | pending |
| 2 | BehavioralCluster | CoordinationRing | — | pending |
| 3 | BehavioralCluster | CampaignHypothesis | — | pending |
| 4 | AutomationGroup | CoordinationRing | — | pending |
| 5 | AutomationGroup | CampaignHypothesis | — | pending |
| 6 | CoordinationRing | CampaignHypothesis | — | pending |

Per [`§0122`](../../charter/decision-log.md), the [`ontology-revision-cross-subtype-merge-typing.md` §Candidate γ](./ontology-revision-cross-subtype-merge-typing.md) illustrative table is **NOT** automatically adopted — it was explicitly framed as discussion-phase illustrative, not committee recommendation. Each cell requires independent defense.

### Defense framework per cell

Each cell's committee defense should record:

1. **Structural argument.** Why does the proposed target subtype's surface best accommodate the merged hypothesis? E.g., for `{BC, AG} → AG`: automation-signature pattern dominates over shared-operatorship inference because the AG surface's signature-reference field is the strictly-stronger structural commitment.
2. **Subtype-specific surface accommodation.** How does the antecedent that does NOT match the target's type carry its surface into the merged record? E.g., the BC's membership-list is structurally a subset of the AG's actor-set surface; no information is discarded.
3. **Inverse-merge consideration.** What happens under an `{X, Target} → Target` merge where X is itself a prior cross-subtype-merge output? Recursive table-lookup per [`§0122`](../../charter/decision-log.md) Decision 4; the cell-defense should articulate the recursive interaction.
4. **Falsifiability check.** Per [`falsifiability-check` §1.3](../../../.claude/skills/epistemic/falsifiability-check/SKILL.md), is the cell's target structurally enforceable? A `{X, Y} → Z` cell is structurally enforced if the helper-layer code can reject a merge that produces a non-Z output.

## Open Questions

This placeholder explicitly defers:

- **Per-cell target choice.** Each of the 6 cells is open. The illustrative table at [`§Candidate γ`](./ontology-revision-cross-subtype-merge-typing.md) is NOT a default.
- **Cross-cell coherence constraints.** Are there structural constraints across cells? E.g., should `{BC, CR}` and `{CR, BC}` map to the same target (already guaranteed by the unordered-pair shape per [`§0049`](../../charter/decision-log.md))? Should `{BC, CR} → CR` imply `{BC, AG} → AG` and `{BC, CH} → CH` (consistent "BC defers to the other")? Or is each cell evaluated independently?
- **Subtype-specific surface accommodation rule.** Per the per-cell defense framework item 2 above: is there a uniform rule for carrying the non-target antecedent's surface into the merged record, or per-cell?

## Anti-Patterns to Avoid

- **Adopting the illustrative table by default.** The [`§Candidate γ` illustrative table](./ontology-revision-cross-subtype-merge-typing.md) was framed as discussion-phase only. Defaulting to it without per-cell defense would re-introduce the silent-resolution failure mode the form-adopt + parameters-defer pattern is meant to prevent.
- **Resolving cells piecemeal in decision-log without cross-cell coherence review.** Each cell affects the cross-cell coherence question; resolving them one-by-one without an explicit cross-cell pass risks structural incoherence across cells.
- **Treating the table as implementation detail.** The pair-table is a structural commitment per [`§0021`](../../charter/decision-log.md) substrate-time-generation pattern; its content lives at the entity-model level.

## Migration and Backward Compatibility

No historical cross-subtype merge records exist (the within-subtype guard at the helper layer has blocked the surface since §0049 / §0060 / §0067 / §0074). The pair-table is forward-looking.

Lock-in asymmetry: per [`§0122`](../../charter/decision-log.md), the pair-table is harder to revise than the form itself — once a cell has produced records, supersession is the only path per §2.1 substrate-immutability. The cells should be defended with this lock-in in mind.

## References

- [`docs/rfcs/draft/ontology-revision-cross-subtype-merge-typing.md`](./ontology-revision-cross-subtype-merge-typing.md) — the resolved typing RFC; γ form adopted per [`§0122`](../../charter/decision-log.md).
- [`docs/rfcs/discussion/cross-subtype-merge-typing-evidence.md`](../discussion/cross-subtype-merge-typing-evidence.md) — discussion-phase evidence for the typing question.
- [`docs/charter/decision-log.md` §0122](../../charter/decision-log.md) — cross-subtype merge typing resolution.
- [`docs/charter/decision-log.md` §0011](../../charter/decision-log.md) — Q4 staged-combination form-adopt + Layer B parameters-defer; precedent for this placeholder.
- [`docs/charter/decision-log.md` §0021](../../charter/decision-log.md) — substrate-time-generation pattern; relevant to pair-table-as-versioned-operational-definition framing.
- [`docs/charter/decision-log.md` §0049](../../charter/decision-log.md) — Option B merge-as-separately-committed-formation + symmetric-antecedent set-shape.
- [`docs/ontology/entity-model.md` §Cross-subtype operations](../../ontology/entity-model.md) — anchor section for the pair-table.

## Decision Record

Pending. The RFC is on hold per the form-adopt + parameters-defer pattern at [`§0122`](../../charter/decision-log.md). Substantive deliberation will be initiated when operational pressure requires cross-subtype merge implementation OR when the committee directs proactive resolution of the cells.
