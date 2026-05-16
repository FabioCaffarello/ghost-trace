# RFC — Charter Amendment v0.3: §2.5 Hypothesis Lifecycle Explicitness (committee-mode redaction)

- **Status:** discussion
- **Authors:** Ghost Trace governance (Gate §2.5 — first object-level invariant pilot)
- **Date:** 2026-05-15
- **Type:** charter-amendment
- **Affects:** Charter banner (version v0.2.1 → v0.3; status line); Charter §2 L41 (editorial fix per [decision-log §0008](../../charter/decision-log.md) queue); Charter §2.5 (status `pending` → `frozen`; stub replaced by binding text in five subsections); `.claude/CLAUDE.md` §3 canonical vocabulary (4 new terms); `.claude/CLAUDE.md` §4 status table (§2.5 row); `docs/glossary.md` (4 new entries); `docs/ontology/lifecycle-semantics.md` (3 citation refreshes); `docs/ontology/entity-model.md` (1 citation refresh); `docs/ontology/ontology.md` (1 status table refresh); `.claude/skills/epistemic/epistemic-separator/SKILL.md` (4 citation refreshes); `.claude/skills/ontology/vocabulary-discipline/SKILL.md` (1 citation refresh)

> RFCs in Ghost Trace are proposals subject to constitutional review. The [Constitutional Charter](../../charter/constitutional-charter.md) is the authoritative reference. RFCs that conflict with the Charter must either be rejected or accompanied by a corresponding charter amendment (see [`../../charter/amendments.md`](../../charter/amendments.md)).

## Summary

Pilot redaction of §2.5 Hypothesis Lifecycle Explicitness — the first object-level invariant redacted via committee mode (precedent: §4 in Gate 1 [decision-log §0007](../../charter/decision-log.md), meta-prose). Binding text codifies the six lifecycle operations on Category III hypotheses under Q2-A.2 type structure (abstract `Hypothesis` + four sibling subtypes) and Q4 staged-combination AND demotion criterion (Layer A operational; Layer B forward-referenced to §2.4 / §2.6 pending). Charter advances to v0.3. §2 L41 "(pending)" parenthetical correction bundled per [decision-log §0008](../../charter/decision-log.md) queue.

## Motivation

§2.5 is the first object-level invariant redacted via committee mode. The pre-Gate work (Q2 [§0010](../../charter/decision-log.md), Q4 [§0011](../../charter/decision-log.md)) established the Ontology Drafted state that §2.5 binding text anchors. Without §2.5 binding form, the Category III sections in [`entity-model.md`](../../ontology/entity-model.md) and [`lifecycle-semantics.md`](../../ontology/lifecycle-semantics.md) cite a pending source — their Drafted status is structurally orphaned (Step 1.4 non-duplication analysis verdict).

[Decision-log §0008](../../charter/decision-log.md) redaction order places §2.5 first because subsequent §2.x invariants (§2.3, §2.4, §2.6) reference Category III operations that §2.5 codifies. §2.5 redaction unblocks subsequent redactions by establishing the lifecycle vocabulary they depend on.

## Constitutional Review

- **Q1.** Touches Charter banner; Charter §2.5 (this redaction); Charter §2 L41 (editorial fix bundled per [§0008](../../charter/decision-log.md) queue). §1 Thesis frozen, §2 header frozen, §2.1 / §2.2 frozen, §4 frozen — all respected. The §2 L41 fix is an editorial parenthetical removal queued for the next §2-touching amendment; v0.3 IS that amendment.
- **Q2.** Four new glossary terms added (`formation`, `merge`, `split`, `dissolution`) — the lifecycle operations not yet glossary-canonical (`promotion`, `demotion` were added at Q4-2 per [§0011](../../charter/decision-log.md)). No existing glossary term redefined.
- **Q3.** Resolves no remaining Ontology open question. Q2 (subtypes) and Q4 (promotion-demotion criterion) were the pre-Gate work for §2.5; this redaction enacts what those resolutions concluded. Q1 (Session duality), Q3 (independence formal definition), Q5 (influence propagation) remain open as pre-Gates for §2.3, §2.6, §2.4 respectively. Layer B remains deferred to follow-on RFC.
- **Q4.** Yes — this RFC is the charter-amendment process for v0.3.
- **Q5.** §2.5 is a new invariant in binding form (was pending stub). §4 criteria applied throughout drafting; Step 1.2 falsifiability testing produced 22 substantive claims passing all four tests, with SR-6 (Layer B) passing 1.4 by forward-reference and deferring 1.1/1.2/1.3 per Pass-by-forward-reference precedent.
- **Q6.** Not ceremony. Binding §2.5 anchors Ontology Drafted specifications (6 of 8 elements are "Both" — adds constitutional commitment AND anchors Ontology citations per Step 1.4 Phase 3 non-duplication analysis). The redaction unblocks §2.3, §2.4, §2.6 pre-Gates by establishing the lifecycle vocabulary they depend on.

## Proposal

The binding §2.5 text in five subsections paralleling §2.1 / §2.2 (Definition / Structural Requirement / Rationale / Forbidden Anti-Patterns / Boundary Conditions). Full text is captured in [`docs/charter/in-committee/§2.5-hypothesis-lifecycle-explicitness.md`](../../charter/in-committee/§2.5-hypothesis-lifecycle-explicitness.md) under `### Step 1.5 — Candidate binding §2.5 text`. The candidate replaces the current §2.5 stub blockquote; the `### 2.5 Hypothesis Lifecycle Explicitness` heading and its anchor `#25-hypothesis-lifecycle-explicitness` are preserved (existing citations resolve).

Concrete file changes (12 files):

1. `docs/charter/constitutional-charter.md` — banner version v0.2.1 → v0.3; status line clause for §2.5 frozen; §2 L41 editorial parenthetical removal; §2.5 stub replaced by binding text.
2. `docs/charter/amendments.md` — append v0.3 amendment entry.
3. `docs/charter/decision-log.md` — append §0013 entry recording the Gate's methodological output.
4. `docs/charter/in-committee/§2.5-hypothesis-lifecycle-explicitness.md` — status line updated from in-committee draft to merged historical record.
5. `.claude/CLAUDE.md` §3 — four new vocabulary entries (`formation`, `merge`, `split`, `dissolution`); §4 status table row for §2.5 updated; narrative paragraph adds v0.3 clause.
6. `docs/glossary.md` — four new entries.
7. `docs/ontology/lifecycle-semantics.md` — three "(pending)" qualifiers removed (L5, L27, L38).
8. `docs/ontology/entity-model.md` — one "(pending)" qualifier removed (L51).
9. `docs/ontology/ontology.md` — status table row for Invariant 2.5 updated.
10. `.claude/skills/epistemic/epistemic-separator/SKILL.md` — four citation refreshes (L28, L58, L76, L96).
11. `.claude/skills/ontology/vocabulary-discipline/SKILL.md` — one citation refresh (L75).
12. `docs/rfcs/draft/charter-amendment-v0-3-section-2-5-redaction.md` — this RFC.

## Alternatives Considered

- **(a) Full redaction** (adopted). Step 1.4 non-duplication analysis verdict: 6 of 8 elements both add constitutional commitment AND anchor Ontology Drafted citations; 1 element (anti-patterns) is purely Charter-original; 1 element (type structure) inherits — no pure duplicates. Path (a) supported by evidence.
- **(b) Complete removal.** Rejected — would orphan multiple Ontology citations in `entity-model.md` and `lifecycle-semantics.md`; Drafted-status Ontology sections lose constitutional anchor.
- **(c) Partial redaction.** Rejected — no element is a pure duplicate of Ontology. B1 (type structure, Inherits verdict) is the only candidate for paring; but the binding text uses B1 as vocabulary, not anchors it — paring B1 does not reduce §2.5's substantive footprint.
- **(d) Method inadequate.** Rejected — Steps 1.1–1.4 produced decidable evidence across 4 dimensions (falsifiability, epistemic-skill applicability, anti-patterns, boundary conditions, non-duplication). §4 pilot precedent confirmed the procedure for meta-prose; §2.5 pilot extends it to object-level.

## Open Questions

- **Layer B follow-on RFC.** [`ontology-revision-layer-b-deep-criterion`](./ontology-revision-layer-b-deep-criterion.md) (opened at [§0011](../../charter/decision-log.md), `discussion` status) is the locus for Layer B's structural form deliberation. On hold pending §2.4 and §2.6 redactions; the forward-reference contract in §2.5 binding text carries the obligation until activated.
- **Cat II vs Cat III state-as-projection generalization.** Step 1.3 coherence finding 2 surfaced that `epistemic-separator` §4 construction #4 uses state-as-projection for Category II; §2.5 A3 uses it for Category III. Phase A resolution 3 of this Gate decided Cat III only (skill territory for Cat II). A future cross-category invariant or skill revision could reopen.
- **§2.3 / §2.4 / §2.6 pre-Gate planning.** Each future §2.x redaction requires pre-Gate Ontology resolution analogous to Q2/Q4 for §2.5: §2.3 → Ontology Q1 (Session duality); §2.4 → Ontology Q5 (influence propagation); §2.6 → Ontology Q3 (independence formal definition; may itself require experiment-type RFC).

## Anti-Patterns to Avoid

- **Future RFCs revisiting Q2 or Q4 without new evidence.** The Q2 and Q4 resolutions ([§0010](../../charter/decision-log.md), [§0011](../../charter/decision-log.md)) are foundational to §2.5 binding text. Revisiting without new evidence is constitutional drift.
- **Silent generalization of state-as-projection across categories.** Phase A resolution 3 scoped to Category III. Generalizing without an explicit cross-category RFC (touching §2.2 as well as §2.5) is silent amendment.
- **Premature Layer B parameterization in §2.5 binding text.** Layer B's structural form is deferred per [§0011](../../charter/decision-log.md). Embedding numerical parameter shapes in §2.5 binding text pre-decides the follow-on RFC.
- **Skipping the bundled §2 L41 fix in a future §2-touching amendment.** The fix is queued per [§0008](../../charter/decision-log.md). Future §2-touching amendments that fail to bundle the fix leave the documented inaccuracy in frozen text.

## Migration and Backward Compatibility

No historical content affected; pre-implementation. The Ontology citations of "§2.5 pending" refresh to frozen form in Phase D of this amendment. Skill citations refresh similarly. CLAUDE.md §3 vocabulary additions extend the canonical list; no existing entry is rewritten. The glossary additions follow the canonical five-field template.

## References

- [Charter §2.5 binding text](../../charter/in-committee/§2.5-hypothesis-lifecycle-explicitness.md) — `### Step 1.5 — Candidate binding §2.5 text` in the committee-draft scratch.
- [Decision-log §0007 — Gate 1 §4 pilot](../../charter/decision-log.md) — methodological precedent for committee-mode redaction.
- [Decision-log §0008 — redaction order](../../charter/decision-log.md) — places §2.5 first; queues §2 L41 fix.
- [Decision-log §0010 — Q2 resolution](../../charter/decision-log.md) — abstract `Hypothesis` + four sibling subtypes.
- [Decision-log §0011 — Q4 resolution](../../charter/decision-log.md) — staged-combination AND demotion criterion.
- [Decision-log §0012 — v0.2.1 hook fix](../../charter/decision-log.md) — vocab-drift blockquote exemption (discovered during Gate §2.5 Step 1.1).
- [Charter Amendment v0.2 RFC](./charter-amendment-v0-2-section-4-redaction.md) — template for committee-mode redaction amendment.
- [`invariant-redactor` SKILL.md](../../../.claude/skills/constitutional/invariant-redactor/SKILL.md) — the procedure.
- [`falsifiability-check` SKILL.md](../../../.claude/skills/epistemic/falsifiability-check/SKILL.md) — the four-question test applied throughout drafting.

## Decision Record

This RFC is recorded as accepted in [decision-log §0013](../../charter/decision-log.md). Acceptance is recorded by the entry; the RFC remains in `draft/` per [`docs/rfcs/README.md`](../README.md) numbering procedure.
