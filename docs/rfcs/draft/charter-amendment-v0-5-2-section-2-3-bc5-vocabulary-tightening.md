# RFC — Charter Amendment v0.5.2: §2.3 BC5 vocabulary tightening (editorial patch; no meaning change)

- **Status:** draft
- **Authors:** committee
- **Date:** 2026-05-21
- **Type:** charter-amendment
- **Affects:** Charter banner (version line `v0.5.1` → `v0.5.2`; status line: v0.5.2 patch clause appended). §2.3 Boundary Condition 5 (BC5) prose: scope sentence reformulated from `"§2.3 governs the structural commitment of multi-category traversal; not the runtime mechanics of traversal."` to `"§2.3 governs the structural shape of provenance chains crossing category boundaries; not the runtime mechanics of chain traversal."` — no other §2.3 text amended. CLAUDE.md §4 status table narrative paragraph (v0.5.2 chronological clause appended). `.claude/hooks/pre-commit-doc-check.sh` (amendment-in-progress exemption added: hook parses staged-diff additions to `docs/charter/amendments.md` for new `**Sections affected:**` lines, extracts §N.M markers via regex, and exempts those sections from the frozen-section-edit check in the same change set; combined with the v0.5.1 newly-frozen exemption).

> RFCs in Ghost Trace are proposals subject to constitutional review. The [Constitutional Charter](../../charter/constitutional-charter.md) is the authoritative reference.

## Summary

Editorial patch amendment closing the [`§0099`](../../charter/decision-log.md) Methodological Observation 3 carry-forward (also surfaced via [`§0100`](../../charter/decision-log.md) deferred-disposition list): §2.3 BC5 scope sentence (frozen v0.4) contains the phrase `multi-category traversal`, which the §2.4 redaction restricted from §2.4 binding text per Q3 ratification ([`§0099`](../../charter/decision-log.md)). The phrase is not on a watchlist, but the §2.4 committee judgment that it is not canonical creates an asymmetry between §2.3 (uses it) and §2.4 (restricted from it). This RFC resolves the asymmetry by reformulating §2.3 BC5's scope sentence with substantive language matching §2.4 BC7's structural pattern. Charter advances to v0.5.2 (patch — meaning preserved).

## Motivation

Two structurally distinct §2.x sections (§2.3 and §2.4) both govern typed-edge semantics over the same `subject_ref_*` substrate per Q3 ([`§0016`](../../charter/decision-log.md)): §2.3 reads the edges as observational provenance transit; §2.4 reads them as inferential influence relations. §0099 obs 3 names this the "multi-category traversal as recurring §2.x pattern" and surfaces the carry-forward: prior frozen §2.x (here, §2.3 BC5 v0.4) uses umbrella language (`multi-category traversal`) that subsequent committee discipline judges non-canonical (per §2.4 Q3 ratification of BC7-only placement of multi-category observation, restricting the phrase from §2.4 binding text).

Two dispositions catalogued by §0099 obs 3 + §0100 deferred-disposition list:
- **(A) Reformulate §2.3 BC5** with canonical vocabulary as patch/minor amendment.
- **(B) Add `multi-category traversal` to glossary** as cross-§2.x methodological term.

This RFC proposes Option A. Option B would elevate the phrase to canonical status, contradicting the §2.4 Q3 committee judgment that the phrase is not suitable for §2.x binding text. Option A removes the asymmetry by aligning §2.3 BC5 prose with the §2.4 BC7 structural pattern (named-edge-shape scope sentence + runtime-mechanics scope exclusion).

The change is semantically equivalent — both old and new scope sentences claim the same scope-vs-mechanics partition. The reformulation removes vocabulary umbrellaing while preserving the underlying §2.3 commitment.

## Constitutional Review

Q1–Q6 from `rfc-author` skill:

- **Q1 — Which Charter invariants does this RFC touch?** §2.3 BC5 (this patch's scope). §2 header (frozen — qualification criteria respected; no qualification-criteria amendment). §1 Thesis (frozen — provenance commitment unchanged). §2.1 (frozen — substrate immutability inherited unchanged). §2.4 v0.5 (frozen — vocabulary discipline asymmetry resolved; no §2.4 prose amended). §2.5 v0.3 (frozen — no interaction). §4 v0.2 (frozen — falsifiability discipline applied to reformulation). All other §2.x respected.
- **Q2 — New glossary terms?** No. The reformulation removes umbrella language; does NOT add new canonical vocabulary. Plain English substituted for the restricted phrase.
- **Q3 — Resolves any Ontology open question?** No. Closes §0099 obs 3 + §0100 deferred-disposition carry-forward; both internal to Charter discipline, not Ontology.
- **Q4 — Is this RFC the amendment itself?** Yes. The proposal is the Charter patch amendment v0.5.2.
- **Q5 — Is meaning preserved?** Yes. Both old and new sentences claim the same scope partition: §2.3 governs the structural shape of typed-edge structure across category boundaries terminating at Cat I primaries; §2.3 does not govern runtime mechanics of chain traversal. S2 of BC5 already articulates the substantive content ("every chain has typed structure from Assertion to Category I primary observations across category boundaries (Cat II constructs and Cat III hypotheses as transit)") — S2 is unchanged. The reformulation is to the scope sentence (S1) only.
- **Q6 — Is the patch editorial or substantive?** Editorial. The scope sentence's substantive claim is unchanged; only the umbrella language `multi-category traversal` is replaced with substantive-named-shape language. Per [`amendments.md` §Amendment Discipline](../../charter/amendments.md), "patch for clarifications that do not alter meaning" — meaning-preserving prose tightening qualifies as patch.

## Proposal

§2.3 BC5 (frozen v0.4) scope sentence reformulation. Single-clause replacement; no other §2.3 text amended.

**Current text (frozen v0.4):**

> **§2.3 governs the structural commitment of multi-category traversal; not the runtime mechanics of traversal.** The Charter-level commitment is that every chain has typed structure from Assertion to Category I primary observations across category boundaries (Cat II constructs and Cat III hypotheses as transit). Graph indexes, query layers, and projection-rebuild paths are architecture-document territory below §2.3.

**Proposed text (frozen v0.5.2 — patch):**

> **§2.3 governs the structural shape of provenance chains crossing category boundaries; not the runtime mechanics of chain traversal.** The Charter-level commitment is that every chain has typed structure from Assertion to Category I primary observations across category boundaries (Cat II constructs and Cat III hypotheses as transit). Graph indexes, query layers, and projection-rebuild paths are architecture-document territory below §2.3.

**Diff:**
- S1: `"the structural commitment of multi-category traversal"` → `"the structural shape of provenance chains crossing category boundaries"`
- S1: `"the runtime mechanics of traversal"` → `"the runtime mechanics of chain traversal"` (verb-anchored; matches §2.4 BC7's `"the runtime mechanics of traversal"` pattern with clarifying subject)
- S2 + S3: unchanged.

Banner version line: `v0.5.1 (draft, sections in committee mode)` → `v0.5.2 (draft, sections in committee mode)`.
Banner status line: v0.5.2 clause appended.

## Alternatives Considered

- **Option B — Add `multi-category traversal` to glossary as cross-§2.x methodological term.** Rejected. Adding the phrase to glossary elevates it to canonical vocabulary status, contradicting the §2.4 Q3 committee judgment ([`§0099`](../../charter/decision-log.md)) that the phrase is not suitable for §2.x binding text. The Q3 restriction was made AFTER §2.3 v0.4 froze; consistency requires retroactive alignment of §2.3 BC5 to the post-§2.4 vocabulary discipline.
- **Mark §2.3 BC5 as minor amendment v0.6 instead of patch v0.5.2.** Rejected. The reformulation is semantically equivalent (Q5 above); minor amendments are reserved for "substantive changes that do not alter identity" per [`amendments.md`](../../charter/amendments.md). Meaning-preserving prose tightening qualifies as patch per `amendments.md` §Amendment Discipline. The patch classification is also consistent with v0.4.1 + v0.5.1 hook patches (no Charter prose) extended to this case (Charter prose, meaning preserved).
- **No amendment — leave §2.3 BC5 as v0.4 prose.** Rejected. §0099 obs 3 explicit closing decision was either Option A or Option B; "no amendment" leaves the asymmetry permanent. The asymmetry surfaces every time a §2.x reader compares §2.3 BC5 to §2.4 BC7 and notices that §2.4 avoids the umbrella while §2.3 uses it. Maintaining the asymmetry is corrosive of the vocabulary discipline.
- **Defer until §2.6 redaction.** Rejected. §0100 obs deferred-disposition already deferred a related pre-§2.6 vocabulary sweep (CF4 from §0099 charter-reviewer subagent). Bundling this single-clause patch with that future sweep would consume committee bandwidth at §2.6 pre-Gate time on a trivially closeable carry-forward. Closing now reduces §2.6 pre-Gate load.

## Open Questions

None. The reformulation is a single-clause replacement; no committee questions remain after the disposition between Option A and Option B is made.

## Anti-Patterns to Avoid

- Future RFCs adding `multi-category traversal` (or similar umbrella phrases) to Charter binding text. §0099 obs 3 + this v0.5.2 reformulation are the committee record that the phrase is not canonical for §2.x binding text. Future RFCs needing to refer to the concept should use substantive-named-shape language (e.g., "provenance chains crossing category boundaries", "the typed reference edges connecting an Assertion to a Cat II construct or to a Cat III hypothesis").
- Treating this patch as a precedent for substantive §2.3 amendment. The reformulation is editorial only; the §2.3 substantive commitment (S2 of BC5) is unchanged. Future §2.3 amendments altering meaning require minor or major amendment classification per `amendments.md` discipline.
- Silent generalization to non-Charter prose. The patch applies to Charter binding text only. Skill, RFC, decision-log, and informal-prose use of `multi-category traversal` is acceptable as informal vocabulary. Glossary still does NOT contain the term as canonical.

## Migration and Backward Compatibility

No prior implementation references §2.3 BC5's scope sentence by exact wording. Ontology + architecture documents reference §2.3 frozen v0.4 by section number; the BC5 scope sentence is not quoted. Forward-looking RFCs and prose will use the v0.5.2 reformulation as the citable text.

The substantive §2.3 commitment is unchanged; no implementation rework is required.

## References

- [Charter §2.3 Provenance Integrity](../../charter/constitutional-charter.md#23-provenance-integrity) (frozen v0.4 pre-patch; frozen v0.4 with v0.5.2 patch post-merge).
- [Charter §2.4 Inferential Influence Disclosure](../../charter/constitutional-charter.md#24-inferential-influence-disclosure) (frozen v0.5; BC7 structural pattern referenced).
- [`decision-log.md` §0017](../../charter/decision-log.md) — Gate §2.3 closure (BC5 v0.4 origin).
- [`decision-log.md` §0099](../../charter/decision-log.md) — Gate §2.4 closure; obs 3 surfaces the carry-forward.
- [`decision-log.md` §0100](../../charter/decision-log.md) — Patch amendment v0.5.1 (hook fix); deferred-disposition list cites §0099 obs 3 carry-forward.
- [`amendments.md` §Amendment Discipline](../../charter/amendments.md) — patch-vs-minor-vs-major classification.

## Decision Record

Pending. On acceptance via charter-reviewer subagent invocation + Charter prose patch, this RFC is recorded in [`docs/charter/decision-log.md`](../../charter/decision-log.md) and Charter advances to v0.5.2.
