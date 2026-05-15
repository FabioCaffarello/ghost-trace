# RFC — Charter Amendment v0.2: §4 Constitutional Design Rule Redaction

- **Status:** discussion
- **Authors:** Ghost Trace governance (Gate 1 pilot)
- **Date:** 2026-05-15
- **Type:** charter-amendment
- **Affects:** Charter banner (version line: v0.1.1 → v0.2; status line: §4 frozen); Charter §4 (pending stub replaced with binding text); `.claude/CLAUDE.md` §4 status table (§4 row pending → frozen); `docs/glossary.md` (`subordination` entry editorial fix; `falsifiability` entry qualifier removal); `.claude/skills/epistemic/falsifiability-check/SKILL.md` (citation qualifier removals); `docs/charter/in-committee/§4-constitutional-design-rule.md` (status line: scratch closure)

> RFCs in Ghost Trace are proposals subject to constitutional review. The [Constitutional Charter](../../charter/constitutional-charter.md) is the authoritative reference. RFCs that conflict with the Charter must either be rejected or accompanied by a corresponding charter amendment (see [`../../charter/amendments.md`](../../charter/amendments.md)).

## Summary

Pilot redaction of Charter §4 produces binding text codifying the two surviving bullets from the working stub: the qualification criteria (Bullet 1, fulfilling §2 L41's frozen promise) and the falsifiability discipline (Bullet 4, anchoring five structural infrastructure citations). Bullets 2 (amendment philosophy) and 3 (precedence rule) of the prior working stub are removed as substantive duplicates of infrastructure-enforced disciplines without anchor obligations. The Charter advances to v0.2; §4 status moves from `pending` to `frozen`.

## Motivation

Two motivations, both surfaced during Gate 1 of post-setup constitutional work and recorded in the in-committee scratch ([`docs/charter/in-committee/§4-constitutional-design-rule.md`](../../charter/in-committee/§4-constitutional-design-rule.md), Steps 1.1–1.4).

First, Charter §2 L41 (FROZEN per v0.1.1) states that the four qualification criteria "are themselves recorded formally in [Section 4 — Constitutional Design Rule] (pending)." Until this redaction, §4 was a working stub and the §2 L41 promise was unfulfilled — §2's frozen text referenced a §4 that did not deliver the formal record.

Second, the falsifiability discipline used throughout the infrastructure is cited as Charter-level authority in five structural places ([`falsifiability-check` SKILL.md L8 and L118](../../../.claude/skills/epistemic/falsifiability-check/SKILL.md); [`CLAUDE.md` L45](../../../.claude/CLAUDE.md); [`amendments.md` L22](../../charter/amendments.md); [glossary `falsifiability` L146](../../glossary.md)). The §4 working stub provided a working-text anchor; binding §4 strengthens the anchor and removes the "(pending)" qualifier from the citations.

The Step 1.4 non-duplication analysis showed that Bullets 2 and 3 of the working stub neither anchor any infrastructure citation nor enforce any discipline that infrastructure does not already operationalize. Their removal closes the redaction without orphaning anything.

## Constitutional Review

- **Q1 — Charter invariants touched.** §4 (pending → frozen, this RFC). §2 L41 (FROZEN) — its forward citation to §4 becomes fulfilled. The (pending) qualifier within §2 L41 is a status snapshot that becomes inaccurate after v0.2; the committee accepts this as a known minor inconsistency in frozen text (the alternative — re-amending §2 frozen text to amend one parenthetical — is disproportionate; see Phase C.3 in the scratch). No other invariant amended.
- **Q2 — Glossary term redefinition.** No glossary term is redefined. The `falsifiability` entry's `Introduction`, `Stabilization`, and `Last amendment` fields are updated editorially to reflect that §4 is now binding rather than pending. The `subordination` entry's `Last amendment` field is updated editorially to remove a false prediction (that §4 would formalize the precedence rule).
- **Q3 — Open Ontology questions resolved.** None resolved.
- **Q4 — Charter amendment required.** Yes — this RFC is the charter-amendment process for v0.2.
- **Q5 — New invariant introduced.** No invariant beyond the two surviving bullets, which were already in the working stub. The Constitutional Design Rule is one section codifying two related disciplines (qualification + falsifiability); neither is a new invariant.
- **Q6 — Ceremony without behavioral consequence.** No. Binding §4 fulfills §2 L41's frozen promise, anchors five structural citations that previously pointed at pending working text, and introduces a Charter-level constitutional anti-pattern for the identity-defining criterion that had no enforcement before. Each change has a concrete consequence; minimalism (CLAUDE.md §7) is respected.

## Proposal

### Charter changes

The §4 stub body is replaced by binding text codifying two disciplines. The replacement preserves the `## 4. Constitutional Design Rule` heading and its `#4-constitutional-design-rule` anchor. The binding text is reproduced in full in the in-committee scratch ([`docs/charter/in-committee/§4-constitutional-design-rule.md`](../../charter/in-committee/§4-constitutional-design-rule.md), Step 1.5 `[BEGIN CANDIDATE]` / `[END CANDIDATE]` block) under the path-(b) form (with `schemas` plural for one-letter hook-tripwire normalization of §2 criterion 1's verbatim quotation). It has five subsections paralleling §2.1 and §2.2:

- **Definition** — declares qualification and falsifiability disciplines; reproduces §2's four criteria verbatim (path-b form) in an attributed blockquote for anchor purposes.
- **Structural Requirement** — names the application points: `amendments.md` Step 2 for falsifiability review; the redaction stage and `invariant-redactor` final-merge checklist for qualification.
- **Rationale** — explains why these disciplines are constitutional and addresses the self-reference of §4's falsifiability claim via an explicit fixed-point reading.
- **Forbidden Anti-Patterns** — four entries, one per qualification criterion.
- **Boundary Conditions** — three entries: §4 does not govern internal project practice; governs the form of invariants, not their content; does not govern the infrastructure that supports the Charter.

### Banner changes

- Version line: `**Version:** v0.1.1` → `**Version:** v0.2`.
- Status line: split the prior "Non-Goals and Constitutional Design Rule pending" into separate clauses, declaring §4 frozen.

### Adjacent changes

- [`.claude/CLAUDE.md` §4 status table](../../../.claude/CLAUDE.md): §4 row `pending` → `frozen`.
- [`docs/glossary.md`](../../glossary.md): editorial fixes to `subordination` and `falsifiability` entries.
- [`.claude/skills/epistemic/falsifiability-check/SKILL.md`](../../../.claude/skills/epistemic/falsifiability-check/SKILL.md): remove "(pending working text)" / "(pending)" qualifiers from L8 and L118 citations.
- [`docs/charter/in-committee/§4-constitutional-design-rule.md`](../../charter/in-committee/§4-constitutional-design-rule.md): status line updated to `merged`; rest of scratch preserved as historical record.

## Alternatives Considered

- **Path (a) — full retention of all four working-stub bullets in §4.** Rejected at Step 1.4: Bullet 3 (precedence rule) failed Step 1.2 falsifiability on observation and operationalization tests (its operative term "conflict" is undefined). Bullet 2 (amendment philosophy) passes falsifiability cleanly but adds no enforcement and is not cited as anchor by any infrastructure.
- **Path (b) — complete removal of §4.** Rejected at Step 1.4: would orphan five Bullet 4 citations and leave §2 L41's frozen promise unfulfilled.
- **Path (c) narrow-minimal — only Bullet 4 survives.** Was the pre-Additional-citations recommendation. Reversed when a second-pass citation grep surfaced §2 L41's frozen reference to §4 as the locus where the criteria "are themselves recorded formally"; this made Bullet 1's retention load-bearing.
- **Path (d) — declare the committee-mode method inadequate.** Rejected: the pilot produced decidable output and surfaced empirical evidence (citation grep, falsifiability failures) without method failure.

## Open Questions

- The §2 L41 frozen text contains a "(pending)" status qualifier on §4 that becomes inaccurate after v0.2. The committee accepts this as a known minor inconsistency rather than re-amending frozen §2 text for one parenthetical. Future amendments to §2 (if any) may editorially fix the qualifier. See decision-log entry §0007.
- Watchlist-extension candidate `conflict` (surfaced in Step 1.2 via Bullet 3's operationalization failure) is deferred to a future RFC; not enacted by this gate.
- Glossary entry for "constitutional invariant" was surfaced in Step 1.2 as missing from [`CLAUDE.md` §3 canonical vocabulary](../../../.claude/CLAUDE.md). Deferred as a future mini-RFC; not enacted here.

## Anti-Patterns to Avoid

- **Silent expansion of binding §4 scope beyond the two surviving bullets.** Future RFCs proposing additions to §4 must satisfy the four qualification criteria + falsifiability discipline that §4 itself declares.
- **Re-litigating the Step 8 non-duplication finding without new evidence.** Bullets 2 and 3 were removed on documented citation-grep evidence; their re-inclusion requires either new citation evidence or an articulated change in the project's discipline.
- **Treating §4 as governing object-level content.** The Boundary Conditions name what §4 does not govern; future RFCs must respect these boundaries.

## Migration and Backward Compatibility

No historical content affected. The five infrastructure citations to Charter §4 remain valid; their qualifier text ("(pending working text)" / "(pending)") is editorially fixed in Phase D. §2 L41's frozen "(pending)" qualifier on §4 is acknowledged as inaccurate after v0.2 and not edited (see Open Questions).

## References

- [Constitutional Charter](../../charter/constitutional-charter.md), particularly §2 header (L32–42; FROZEN per v0.1.1; L41 forward-citation to §4) and the current §4 stub.
- [`amendments.md`](../../charter/amendments.md) — v0.1.1 entry as most-recent precedent for amendment record format.
- [`decision-log.md`](../../charter/decision-log.md) — entry §0007 produced in Gate 1 closure consolidates the pilot's methodological output.
- [`docs/charter/in-committee/§4-constitutional-design-rule.md`](../../charter/in-committee/§4-constitutional-design-rule.md) — full record of Steps 1.1–1.5.
- [`.claude/skills/constitutional/invariant-redactor/SKILL.md`](../../../.claude/skills/constitutional/invariant-redactor/SKILL.md) — committee-mode procedure applied throughout Gate 1.

## Decision Record

This RFC is recorded as accepted in [`decision-log.md`](../../charter/decision-log.md) §0007 at the moment of v0.2 amendment merge. The decision-log entry consolidates the pilot's methodological output (four observations + two carried-forward items).
