# Charter Amendments

This document records formal amendments to the [Constitutional Charter](./constitutional-charter.md). It is the authoritative history of how the Charter has evolved.

## Amendment Discipline

The Charter is the constitutional surface of Ghost Trace. Changes to it are not made through ordinary commits.

An amendment is required for any change that:

1. Alters or removes an existing Constitutional Invariant.
2. Adds a new Constitutional Invariant.
3. Modifies the Thesis.
4. Adds or removes Non-Goals.
5. Modifies the Constitutional Design Rule.

Editorial fixes that do not alter meaning (typography, formatting, broken links) do not require amendment but should be noted in the [decision log](./decision-log.md) if they touch the Charter.

## Amendment Process

1. **Proposal.** A change is proposed as an RFC in [`../rfcs/`](../rfcs/), explicitly tagged as `charter-amendment`. The RFC must identify which Charter element is affected and why the change cannot be expressed in subordinate documents instead.
2. **Falsifiability review.** The proposal is evaluated against the criteria in [Section 4 of the Charter](./constitutional-charter.md#4-constitutional-design-rule). Proposals that introduce non-falsifiable language are rejected on procedural grounds.
3. **Committee redaction.** Accepted proposals are redacted in committee mode — one section at a time, with explicit defense of each word choice.
4. **Amendment record.** Adopted changes are recorded in this file with the date, the Charter version before and after, the affected sections, and a brief rationale.
5. **Version bump.** The Charter version is incremented. Patch-level changes (`v0.1.x`) for clarifications that do not alter meaning; minor (`v0.2`) for substantive changes that do not alter identity; major (`v1.0+`) reserved for the moment the Charter is declared production-ready.

## Amendment Log

### `v0.1` — Charter Inception

**Date:** _(repository creation date)_
**Sections affected:** Thesis (frozen); Invariants 2.1 and 2.2 (frozen).
**Rationale:** Initial committee redaction of the Thesis, Observational Integrity, and Epistemic Separation. Remaining invariants (2.3–2.6), Non-Goals (Section 3), and Constitutional Design Rule (Section 4) remain in committee.

No amendments have yet been recorded against `v0.1`. Future entries follow.

---

### `v0.1.1` — Clarify protected surface (§2 header frozen; hook scope explicit)

**Date:** 2026-05-15
**Originating RFC:** [`charter-amendment-v0-1-1-clarify-protected-surface`](../rfcs/draft/charter-amendment-v0-1-1-clarify-protected-surface.md)
**Sections affected:** Banner (version line; status line); §2 header (status change, explicit FROZEN).
**Summary:** The §2 (Constitutional Invariants) header — the four invariant qualification criteria — is explicitly declared FROZEN, removing ambiguity from the previous "Invariants 1–2 frozen" wording. The hook's protected file set is enumerated and extended to include `.claude/CLAUDE.md` and `.claude/README.md`. Three editorial rewrites apply the existing vocabulary discipline to formerly out-of-scope text; a new canonical-phrase exemption ("primary event log", "decision log", "event log", "historical fact") prevents legitimate canonical-vocabulary use from being reported as drift.
**Rationale:** Phase 8 SELF-AUDIT Finding 7.1 plus the empirical pre-scan during Gate 0b made two assumptions explicit that the infrastructure had previously left implicit. The Charter's substantive content is unchanged.
**Falsifiability review outcome:** Pass. The change clarifies a status table and a hook predicate, both of which are mechanically falsifiable. No non-falsifiable language introduced.

---

### `v0.2` — §4 Constitutional Design Rule (committee-mode redaction)

**Date:** 2026-05-15
**Originating RFC:** [`charter-amendment-v0-2-section-4-redaction`](../rfcs/draft/charter-amendment-v0-2-section-4-redaction.md)
**Sections affected:** Banner (version line: v0.1.1 → v0.2; status line: §4 frozen). §4 (status pending → frozen; stub body replaced with binding text in five subsections paralleling §2.1 / §2.2).
**Summary:** Pilot redaction of §4 produces binding text codifying the two surviving bullets from the working stub: the qualification criteria (Bullet 1, fulfilling §2 L41's frozen promise that the criteria "are themselves recorded formally" in §4) and the falsifiability discipline (Bullet 4, anchoring five structural infrastructure citations in `falsifiability-check`, `amendments.md`, `CLAUDE.md`, and the glossary). Bullets 2 (amendment philosophy) and 3 (precedence rule) of the prior working stub are removed as substantive duplicates of infrastructure-enforced disciplines without anchor obligations. The Charter advances to v0.2 (minor — substantive new binding text without identity change).
**Rationale:** Gate 1 of post-setup constitutional work applied the `invariant-redactor` committee-mode procedure (Steps 1.1–1.4) to the §4 stub. Step 1.2 falsifiability testing found Bullet 3 fails on observation and operationalization (its operative term "conflict" is undefined). Step 1.4 non-duplication analysis found Bullets 2 and 3 substantively duplicate infrastructure-enforced disciplines and are not cited as anchors by any infrastructure, while Bullet 1 fulfills §2 L41's frozen forward-citation and Bullet 4 anchors five structural citations. Removal of Bullets 2 and 3 does not orphan any citation; retention of Bullets 1 and 4 fulfills constitutional promises and preserves infrastructure dependencies. The full evidentiary record is preserved in [`docs/charter/in-committee/§4-constitutional-design-rule.md`](./in-committee/§4-constitutional-design-rule.md).
**Falsifiability review outcome:** Pass. Phase B of the redaction recorded an eleven-claim self-test against `falsifiability-check` §1's four-question test (V / O / Op / NC); all eleven substantive claims passed all four tests. The self-reference of §4's falsifiability claim is handled via the explicit fixed-point reading recorded in the Rationale subsection: the test procedure is defined externally to §4 (in `falsifiability-check` §1), and §4's claims reduce to procedural artifacts, so the chain bottoms out in procedure rather than in self-reference.

---

### `v0.2.1` — Extend Charter-blockquote exemption to vocabulary-drift

**Date:** 2026-05-15
**Originating RFC:** [`charter-amendment-v0-2-1-extend-blockquote-exemption-to-vocab-drift`](../rfcs/draft/charter-amendment-v0-2-1-extend-blockquote-exemption-to-vocab-drift.md)
**Sections affected:** Banner (version line `v0.2` → `v0.2.1`; status-line clause appended noting the patch). No Charter prose amended.
**Summary:** The mechanical exemption rule established in [`decision-log.md` §0005](./decision-log.md) (Gate 0a) for attributed Charter/Ontology blockquotes is extended from marketing-tell detection only to vocabulary-drift detection as well. The hook's vocabulary-drift loop gains the same `eligible_blockquote_lines` filter the marketing-tell loop has had since v0.1.1. The ambiguity advisory remains non-exempt (informational, not blocking).
**Rationale:** The original §0005 scope decision treated mis-quotation as the failure mode — a quotation containing a forbidden synonym would indicate stale Charter or misquotation. The scope was defensible when the only blockquotes in scope quoted frozen Charter text. Gate §2.5 Step 1.1 surfaced systematic stale-stub-vocabulary tripwires: the §2.5 stub uses, in ordinary-English sense, a noun that the [`glossary`](../glossary.md) records (entry for `operational construct`) as a forbidden synonym. The stub was authored in setup Phases 2–3; the glossary entry was added in Phase 4. The stub is genuinely stale relative to current canonical vocabulary, not in error. The empirical mode for committee-mode redaction pilots — including §2.3, §2.4, §2.6, §3 in future — is stale-quotation by construction. Committee review during redaction phases assumes the responsibility §0005's narrower scope assigned to the hook (catching genuine mis-quotation of frozen Charter prose with non-canonical vocabulary). The fix is mechanical replication of the marketing-tell exemption pattern in the vocabulary-drift loop; no new exemption mechanism is introduced.
**Falsifiability review outcome:** Pass. The change clarifies a mechanical predicate (the hook's vocabulary-drift filter) without introducing non-falsifiable language. The exemption is structurally falsifiable: a quotation in an attributed blockquote (regex match on a specific line: `^[[:space:]]*>[[:space:]]*—[[:space:]]*\[(Charter|Ontology)`) is the predicate that fires the exemption; the predicate is detectable mechanically without subjective judgment.

---

### `v0.3` — §2.5 Hypothesis Lifecycle Explicitness (committee-mode redaction)

**Date:** 2026-05-15
**Originating RFC:** [`charter-amendment-v0-3-section-2-5-redaction`](../rfcs/draft/charter-amendment-v0-3-section-2-5-redaction.md)
**Sections affected:** Banner (version line `v0.2.1` → `v0.3`; status line: §2.5 frozen clause added). §2.5 (status `pending` → `frozen`; stub body replaced with binding text in five subsections paralleling §2.1 / §2.2). §2 L41 (editorial — "(pending)" parenthetical removed per [`decision-log.md` §0008](./decision-log.md) queue).
**Summary:** Pilot redaction of §2.5 produces binding text codifying the six Category III lifecycle operations (formation, merge, split, promotion, demotion, dissolution) under Q2-A.2 type structure (abstract `Hypothesis` + four sibling subtypes per [`decision-log.md` §0010](./decision-log.md)) and Q4 staged-combination AND demotion criterion (Layer A operational; Layer B forward-referenced to §2.4 / §2.6 pending per [`decision-log.md` §0011](./decision-log.md)). Seven anti-patterns enumerated structurally; five boundary conditions including the lifecycle-event-as-Category-I-record vs event-content-referent distinction. §2 L41 editorial bundling enacts the queued correction from [`decision-log.md` §0008](./decision-log.md). Charter advances to v0.3 (minor — new binding text in a previously pending section).
**Rationale:** Gate §2.5 applied the `invariant-redactor` committee-mode procedure (Steps 1.1–1.5) to the §2.5 stub. Step 1.2 falsifiability testing produced 22 substantive claims passing all four tests, with one structural deferral (Layer B's structural form per Pass-by-forward-reference precedent from [§2 L41](./constitutional-charter.md#2-constitutional-invariants)). Step 1.3 epistemic-skill findings converted 6 vocabulary carry-forwards to formal Path decisions and surfaced the lifecycle-event-as-Category-I-record methodological observation (promoted to Boundary Condition 5). Step 1.4 non-duplication analysis found 6 of 8 elements both add constitutional commitment AND anchor Ontology Drafted specifications; 1 element (anti-patterns) is purely Charter-original; 1 element (type structure) inherits — no pure duplicates, path (a) full redaction supported by evidence. The full evidentiary record is preserved in [`docs/charter/in-committee/§2.5-hypothesis-lifecycle-explicitness.md`](./in-committee/§2.5-hypothesis-lifecycle-explicitness.md).
**Falsifiability review outcome:** Pass. Phase B of the redaction recorded a 22-claim self-test against [`falsifiability-check` §1](../../.claude/skills/epistemic/falsifiability-check/SKILL.md) four-question test (V/O/Op/NC). All 22 substantive claims pass all four tests, with one structural exception: SR-6 (Layer B) passes 1.4 by forward-reference to glossary-canonical terms (`evidential independence`, `influence`) and defers 1.1/1.2/1.3 per Step 1.2 Phase 2 Reading 2 (Pass-by-forward-reference precedent — committee-resolved). The forward-reference contract follows §2 L41 precedent.

---

### `v0.4` — §2.3 Provenance Integrity (committee-mode redaction)

**Date:** 2026-05-18
**Originating RFC:** [`charter-amendment-v0-4-section-2-3-redaction`](../rfcs/draft/charter-amendment-v0-4-section-2-3-redaction.md)
**Sections affected:** Banner (version line `v0.3` → `v0.4`; status line: §2.3 frozen clause added; §2.3 row dropped from pending list). §2.3 Provenance Integrity (status `pending` → `frozen`; stub body replaced with binding text in five subsections paralleling §2.1 / §2.2 / §2.5). Ontology citation refreshes ([`entity-model.md`](../ontology/entity-model.md), [`provenance-model.md`](../ontology/provenance-model.md), [`ontology.md`](../ontology/ontology.md)). CLAUDE.md §4 status table row refresh + narrative refresh. `epistemic-separator` skill: new forbidden construction citing §2.3 frozen v0.4 (Phase A Resolution 5 enactment). `charter-guardian` / `falsifiability-check` / `ontology-keeper` skill pending-list refreshes.
**Summary:** Pilot redaction of §2.3 produces binding text codifying multi-category provenance traversal as the structural commitment connecting §2.1 (substrate immutability foundation, inherited) + §2.5 v0.3 (Cat III lifecycle event commit semantics, inherited) + Q1 ([`§0015`](./decision-log.md) Session duality → DeclaredSession + OperationalSession) + Q3 ([`§0016`](./decision-log.md) typed `subject_ref_X` fields with oneOf/union exclusivity) + [`provenance-model.md`](../ontology/provenance-model.md) post-Q3 typed-by-category edges into a single provenance invariant. Seven anti-patterns enumerated (AP1 orphan typed-reference, AP2 multi-populated `subject_ref`, AP3 aggregation destroying Cat II transit, AP4 cache-as-projection inversion, AP5 multi-category chain rupture, AP6 retroactive provenance rewriting, AP7 lifecycle event reference fabrication). Six boundary conditions including the multi-category-traversal observation as BC5 (Phase A Resolution 3) and the Q2 forward-reference contract as BC4 (Phase A Resolution 4 — adapted Ontology-level marker form). Skill `epistemic-separator` §4 gains new forbidden construction citing §2.3 frozen v0.4, restoring bidirectional mutual reinforcement (Phase A Resolution 5). Charter advances to v0.4 (minor — new binding text in a previously pending section). §2 L41 fix is NOT bundled (already enacted at Gate §2.5 closure per [`decision-log.md` §0013](./decision-log.md)).
**Rationale:** Gate §2.3 applied the `invariant-redactor` committee-mode procedure (Steps 1.1–1.5) to the §2.3 stub. Step 1.1 produced 12-element inventory + scaffold (6 from stub, 6 from Ontology Q1/Q3 resolutions + §2.5 inheritance). Step 1.2 falsifiability testing produced 48 cells — 6 elements pass cleanly plus B6 under Reading 2 (Pass-by-forward-reference precedent extended to Ontology-level Open Modeling Question 1), 5 elements with vocabulary caveats. Step 1.3 epistemic-skill findings converted 5 vocabulary carry-forwards plus B6 parallel-track plus A5 fresh-scan to 7 formal Path decisions, surfaced 3 coherence findings with `epistemic-separator` skill §4 constructions, and identified the multi-category-traversal methodological observation. Step 1.4 produced 7 anti-patterns + 6 boundary conditions + 36-cell non-duplication analysis: 7 Adds (in any combination) + 3 Anchors + 8 Inherits (in any combination) — inheritance-dominant pattern empirically documented (§2.5-1.4 was 6 Both + 1 Adds + 1 Inherits; §2.3's substantial priors — §2.1 + §2.5 + Q1 + Q3 — produce qualitatively different Step 1.4 outcomes). Step 1.5 Phase A resolved 8 committee questions (Resolutions 1–8; scratch Q6 RFC-tracking-symmetry surfaced as separate Resolution 8); path (a) full redaction supported by structural-completeness argument. The full evidentiary record is preserved in [`docs/charter/in-committee/§2.3-provenance-integrity.md`](./in-committee/§2.3-provenance-integrity.md).
**Falsifiability review outcome:** Pass. Phase B per-sentence self-test recorded 24 substantive cells against [`falsifiability-check` §1](../../.claude/skills/epistemic/falsifiability-check/SKILL.md) four-question test (V/O/Op/NC): 19 Pass + 5 Pass-with-caveat + 0 Fail. Caveats are tracked structural-form properties — SR5/SR6/BC4 are forward-references per Resolution 4 (falsifiability binds at Q2 resolution per [`§0014`](./decision-log.md) lazy-pre-Gate methodology); R1/R2 are foundational prose per §2.x Rationale subsection precedent (§2.1, §2.2, §2.5 Rationales likewise carry foundational claims with operationalization in adjacent subsections). No falsifiability defects.

---

### `v0.4.1` — Fix frozen-section parser to accept amendment-qualified status cells

**Date:** 2026-05-18
**Originating RFC:** [`charter-amendment-v0-4-1-frozen-parser-fix`](../rfcs/draft/charter-amendment-v0-4-1-frozen-parser-fix.md)
**Sections affected:** Banner (version line `v0.4` → `v0.4.1`; status-line clause appended noting the patch). Hook parser (`.claude/hooks/_parse_watchlists.py:54` — regex change). CLAUDE.md §4 narrative (extension noting v0.4.1). No Charter prose amended.
**Summary:** The frozen-section parser at `_parse_watchlists.py:54` uses the regex `\|\s*frozen\s*\|` to detect frozen rows in CLAUDE.md §4 status table. The strict cell match silently drops sections whose status cell carries the `frozen — minor amendment vN.Y` qualifier introduced at amendment v0.3 (per [`decision-log §0013`](./decision-log.md)). Result: §2.5 (frozen v0.3) and §2.3 (frozen v0.4) were not recognized by the parser despite their table rows correctly marking them frozen. Hook self-test reported 5 frozen ranges instead of 7. Fix: regex `\|\s*frozen\s*\|` → `\|\s*frozen[^|]*\|` (allows arbitrary non-pipe content after `frozen`). Post-fix self-test reports `7 / 12 / 36 / 13`. Charter advances to v0.4.1.
**Rationale:** Discrepancy surfaced during [`decision-log §0017`](./decision-log.md) Gate §2.3 closure Phase F.4 final summary. Investigation traced the gap to the v0.3-introduced "minor amendment vN.Y" qualifier convention which the parser never accommodated. Functional protection was incidentally preserved by the §2 outer range (which wraps all §2.x sub-sections including §2.3 and §2.5), so the gap was cosmetic and audit-trail-relevant rather than enforcement-relevant. The fix nonetheless restores parser fidelity to the table's actual claims and removes latent fragility (future Charter structure changes could create configurations where the §2 outer range no longer wraps every frozen §2.x). Patch-level amendment per [`amendments.md` Step 5](#amendment-process); precedent is v0.2.1 (`decision-log §0012`) hook-correctness fix.
**Falsifiability review outcome:** Pass. The fix clarifies a mechanical predicate (the cell-match regex). The new predicate is structurally falsifiable: a status cell beginning with `frozen` and containing arbitrary non-pipe trailing content matches; cells with `pending` or other words do not match. Detection is mechanical, no subjective judgment. The fix passes [§4 frozen v0.2](./constitutional-charter.md#4-constitutional-design-rule) falsifiability discipline.

---

<!-- AMENDMENT TEMPLATE — copy below this line when recording an amendment -->

<!--
### `vX.Y` — Short title

**Date:** YYYY-MM-DD
**Originating RFC:** [link]
**Sections affected:** [list]
**Summary:** [one or two sentences]
**Rationale:** [why the change was necessary and why it could not be made in a subordinate document]
**Falsifiability review outcome:** [pass with reasons]
-->
