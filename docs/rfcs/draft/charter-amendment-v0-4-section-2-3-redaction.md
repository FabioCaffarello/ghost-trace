# RFC — Charter Amendment v0.4: §2.3 Provenance Integrity (committee-mode redaction)

- **Status:** accepted
- **Authors:** committee
- **Date:** 2026-05-18
- **Type:** charter-amendment
- **Affects:** Charter banner (version + status line); §2.3 Provenance Integrity (status `pending` → `frozen`; stub body replaced with binding text); CLAUDE.md §4 status table; Ontology forward-reference markers in [`entity-model.md`](../../ontology/entity-model.md), [`provenance-model.md`](../../ontology/provenance-model.md), [`ontology.md`](../../ontology/ontology.md); `epistemic-separator` skill (new forbidden construction citing §2.3 frozen v0.4); `charter-guardian` skill (move §2.3 to FROZEN list); `falsifiability-check` skill + `ontology-keeper` skill (pending-list refresh).

> RFCs in Ghost Trace are proposals subject to constitutional review. The [Constitutional Charter](../../charter/constitutional-charter.md) is the authoritative reference.

## Summary

Redaction of §2.3 (Provenance Integrity) producing binding text that inherits from §2.1 (substrate immutability foundation) and §2.5 v0.3 (Cat III lifecycle event commit semantics) and from pre-Gate Ontology RFC resolutions Q1 ([`§0015`](../../charter/decision-log.md)) and Q3 ([`§0016`](../../charter/decision-log.md)) — codifying multi-category provenance traversal as the structural commitment connecting these priors into a single provenance invariant; carrying Q2 (Identity tiers — Open Modeling Question 1) as a forward-reference contract via [`§0015`](../../charter/decision-log.md) procedural default with adapted Ontology-level marker form; promoting Step 1.3's multi-category-traversal observation to Boundary Condition 5. Charter advances to v0.4. §2 L41 fix is NOT bundled here (already enacted at Gate §2.5 closure per [`§0013`](../../charter/decision-log.md)).

## Motivation

§2.3 is the second object-level Charter invariant redacted in committee mode (§2.5 was the first, frozen v0.3). Decision-log [`§0008`](../../charter/decision-log.md) redaction order places §2.3 after §2.5. Pre-Gate work — [`§0014`](../../charter/decision-log.md) lazy-pre-Gate refinement methodology, [`§0015`](../../charter/decision-log.md) Q1 resolution (Session duality → DeclaredSession + OperationalSession), [`§0016`](../../charter/decision-log.md) Q3 resolution (per-Category typed `subject_ref_X` fields with oneOf/union exclusivity) — established the Ontology Drafted state §2.3 binding text inherits and anchors.

Without §2.3, the post-Q1+Q3 substrate commitments are present in [`entity-model.md`](../../ontology/entity-model.md) + [`provenance-model.md`](../../ontology/provenance-model.md) but lack a constitutional anchor connecting them into a single provenance invariant. §2.3 supplies that anchor: every Assertion's typed reference chain from Cat II/III back to Cat I primaries is the structural commitment on which the Charter's distinction between what was observed and what was inferred rests.

## Constitutional Review

Q1–Q6 from `rfc-author` skill:

- **Q1 — Which Charter invariants does this RFC touch?** §2.3 (this redaction's purpose). §1 Thesis (frozen — cited as rationale anchor), §2 header (frozen — qualification criteria respected), §2.1 (frozen — substrate immutability inherited), §2.2 (frozen — Cat I/II/III boundary respected), §2.5 (frozen v0.3 — lifecycle event commit semantics inherited), §4 (frozen — falsifiability discipline applied per Phase B self-test) — all respected; none amended.
- **Q2 — New glossary terms?** None. Step 1.3 backlog review confirmed empty per [§2.3 scratch L668–671](../../charter/in-committee/§2.3-provenance-integrity.md). Q3-resolved field-name literals (`subject_ref_observation` et al.) are vocabulary scaffold per Reforço 2 — live in [`entity-model.md`](../../ontology/entity-model.md), not glossary.
- **Q3 — Resolves any Ontology open question?** No. Q1 + Q3 were pre-Gate work ([`§0015`](../../charter/decision-log.md), [`§0016`](../../charter/decision-log.md)); their resolutions are anchored by §2.3 binding text but not re-resolved here. Q2 (Identity tiers — Open Modeling Question 1) remains forward-referenced per Phase A Resolution 4 marker form with [`§0014`](../../charter/decision-log.md) + [`§0015`](../../charter/decision-log.md) tracking — no new RFC opened per Phase A Resolution 8.
- **Q4 — Is this RFC the amendment itself?** Yes. The proposal is the Charter amendment.
- **Q5 — Is §2.3 a new invariant in binding form?** Yes (was pending stub since Charter inception v0.1). §4 frozen v0.2 criteria applied throughout drafting: Phase B per-sentence falsifiability self-test verified zero Fail verdicts across 24 substantive cells (19 Pass + 5 Pass-with-caveat; caveats are tracked structural-form properties — forward-reference per Resolution 4 and foundational-prose per §2.x Rationale precedent — not defects).
- **Q6 — Is the redaction ceremonial or constitutional?** Constitutional. Binding §2.3 anchors multi-category traversal as constitutional commitment via BC5; inheritance citations connect §2.1 + §2.5 + Q1 + Q3 + [`provenance-model.md`](../../ontology/provenance-model.md) into a single invariant; bidirectional mutual reinforcement with `epistemic-separator` skill §4 restored via new construction (Phase D.1 Resolution 5). The redaction is not ceremony.

## Proposal

§2.3 binding text drafted in [§2.3 scratch §Step 1.5 — Candidate binding §2.3 text](../../charter/in-committee/§2.3-provenance-integrity.md), reproduced verbatim into Charter §2.3 (heading + anchor `#23-provenance-integrity` preserved). Five subsections per the §2.x template: Definition (~75 words; A1 + A2), Structural Requirement (~190 words; B1–B6 + Resolution 4 marker), Rationale (~85 words; §1 Thesis + §2.1 inheritance anchoring), Forbidden Anti-Patterns (~200 words; AP1–AP7), Boundary Conditions (~195 words; BC1–BC6 including multi-category-traversal scope BC5 and Q2 forward-reference BC4). Total ~745 words, within calibration of §2.5's ~760-word binding text.

Banner version line: `v0.3 (draft, sections in committee mode)` → `v0.4 (draft, sections in committee mode)` (draft suffix preserved — §2.4, §2.6, §3 still pending).
Banner status line: §2.3 row added — `**Invariant 2.3 frozen — minor amendment v0.4.**`.

## Alternatives Considered

- **Path (b) — full removal.** Rejected. Inheritance count is high (8 of 12 elements involve inheritance per Step 1.4c) but Adds-count is still substantial (7 of 12 involve §2.3 adding constitutional weight). A3, A5, A6, B6 are §2.3-original; removal would orphan multi-category-traversal commitment and lose the bidirectional mutual reinforcement with skill §4. Step 1.4 Phase 3 redactor's recommendation evidence-grounded.
- **Path (c) — narrow redaction.** Rejected per Phase A Resolution 1. Inherits-only elements (B1–B5) are constitutive of chain endpoints (Cat I termini, Q3 schemas mechanism, typed-by-category edges, §2.5 BC5 lifecycle event continuation); excluding them leaves §2.3 binding text underspecified — readers cannot determine what chain shape §2.3 governs. B6 forward-reference contract load-bears on B1. Inheritance is precisely *why* §2.3 must codify at Charter level — narrow redaction inverts the argument.
- **Path (d) — method inadequate.** Rejected. Steps 1.1–1.4 produced decidable evidence (12-element inventory, 48-cell Step 1.2 matrix, 36-cell Step 1.3 matrix, 7 anti-patterns + 6 boundary conditions + 36-cell Step 1.4c matrix). The §2.5 pilot precedent (Gate §2.5 / [`§0013`](../../charter/decision-log.md)) confirmed the `invariant-redactor` procedure scales to object-level invariants; §2.3 is the second application and confirms scaling extends to inheritance-dominant content.

## Open Questions

- **Q2 (Identity tiers — [`entity-model.md` Open Modeling Question 1](../../ontology/entity-model.md#open-modeling-questions))** remains forward-referenced per Phase A Resolution 4 marker form. Per Phase A Resolution 8 (option (ii) — accept [`§0014`](../../charter/decision-log.md) + [`§0015`](../../charter/decision-log.md) as sufficient tracking), no concrete follow-on RFC is opened during §2.3 redaction. Q2 formal resolution may surface during §2.4 or §2.6 pre-Gate sequences as blocking dependency, or remain forward-referenced indefinitely if no Charter section first depends substantively on Identity-tier semantics.
- **Layer B follow-on RFC** (per [`§0011`](../../charter/decision-log.md)) remains on hold pending §2.4 + §2.6 redactions. Not affected by §2.3.

## Anti-Patterns to Avoid

- Future RFCs revisiting Q1 ([`§0015`](../../charter/decision-log.md)) or Q3 ([`§0016`](../../charter/decision-log.md)) resolutions without new evidence — those resolutions are now anchored by §2.3 binding text and inheritance via these RFCs is the load-bearing structural commitment.
- Silent generalization of multi-category-traversal scope. BC5 codifies that the *structural commitment* is §2.3 territory; runtime traversal mechanics (graph indexes, query layers, projection-rebuild paths) are below §2.3. Architecture documents must respect this scope.
- Premature Q2 parameterization in §2.3 binding text. Resolution 4 marker form defers formal Identity-tier specification to a future resolution; SR5 / SR6 / BC4 forward-references are the contract — not placeholders for unspecified parameter values.

## Migration and Backward Compatibility

No prior committed records to migrate; forward-looking. Ontology citations of "Charter §2.3 (pending)" refresh to frozen form post-merge (Phase D.3 enacts in [`entity-model.md`](../../ontology/entity-model.md) L56 + TODO L124, [`provenance-model.md`](../../ontology/provenance-model.md) TODO L60, [`ontology.md`](../../ontology/ontology.md) L43). Skill citations refresh: `charter-guardian` moves §2.3 from PENDING to FROZEN list; `falsifiability-check` + `ontology-keeper` pending-lists drop §2.3.

## References

- [Charter §2.3 Provenance Integrity](../../charter/constitutional-charter.md#23-provenance-integrity) (frozen v0.4 post-merge).
- [`decision-log.md` §0008](../../charter/decision-log.md) — strategic sequence placing §2.3 after §2.5.
- [`decision-log.md` §0013](../../charter/decision-log.md) — Gate §2.5 closure precedent.
- [`decision-log.md` §0014](../../charter/decision-log.md) — Gate §2.3 prep + lazy-pre-Gate methodology.
- [`decision-log.md` §0015](../../charter/decision-log.md) — Q1 resolution.
- [`decision-log.md` §0016](../../charter/decision-log.md) — Q3 resolution.
- [`decision-log.md` §0017](../../charter/decision-log.md) — Gate §2.3 closure (this RFC's acceptance record; 4 methodological observations).
- [§2.3 scratch](../../charter/in-committee/§2.3-provenance-integrity.md) — Steps 1.1–1.5 evidentiary record.
- [`amendments.md` v0.4 entry](../../charter/amendments.md).

## Decision Record

Accepted via [`decision-log.md` §0017](../../charter/decision-log.md) at Gate §2.3 closure merge. Charter advances to v0.4.
