# RFC — Charter Amendment v0.2.1: Extend Charter-Blockquote Exemption to Vocabulary-Drift

- **Status:** accepted
- **Authors:** Ghost Trace governance (Gate §2.5 Step 1.1 hook-fix)
- **Date:** 2026-05-15
- **Type:** charter-amendment
- **Affects:** Charter banner (version line; status-line clause); `.claude/hooks/pre-commit-doc-check.sh` (vocab-drift loop exemption); `.claude/CLAUDE.md` §4 (narrative paragraph below the status table); `docs/charter/amendments.md` (new v0.2.1 entry); `docs/charter/decision-log.md` (new entry §0012)

> RFCs in Ghost Trace are proposals subject to constitutional review. The [Constitutional Charter](../../charter/constitutional-charter.md) is the authoritative reference. RFCs that conflict with the Charter must either be rejected or accompanied by a corresponding charter amendment (see [`../../charter/amendments.md`](../../charter/amendments.md)).

## Summary

The mechanical exemption rule established in [`decision-log.md` §0005](../../charter/decision-log.md) (Gate 0a) — attributed Charter/Ontology markdown blockquotes are exempt from the hook's marketing-tell check on a per-line basis — is extended from marketing-tell detection only to vocabulary-drift detection as well. The hook's vocabulary-drift loop gains the same `eligible_blockquote_lines` filter the marketing-tell loop has had since v0.1.1. The ambiguity advisory remains non-exempt (informational, not blocking).

No Charter prose is amended. The change is to mechanical enforcement of an existing rule.

## Motivation

Gate §2.5 Step 1.1 (the first object-level invariant redaction in committee mode, per [`decision-log.md` §0008](../../charter/decision-log.md)) surfaced a hook block at the anchor-scratch commit stage. The §2.5 scratch's Anchor section quotes the §2.5 stub verbatim under blockquote-attribution. The verbatim stub uses, in ordinary-English sense, an outcome-of-mutation noun the canonical vocabulary forbids:

> Operations on hypotheses — formation, merge, split, dissolution, promotion, demotion — are recorded as immutable events in the primary event log. The current state of any hypothesis is a projection over the history of operations applied to it, never the result of direct mutation.
>
> — [Charter §2.5](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness)

The noun is used in the ordinary-English outcome-of sense, not in the Category II construct sense. Per [`docs/glossary.md` L53](../../glossary.md), the noun is a forbidden synonym of `operational construct`. The glossary entry was added in setup Phase 4; the §2.5 stub was authored in setup Phases 2–3. The stub is genuinely stale relative to current canonical vocabulary, not in error.

[`decision-log.md` §0005](../../charter/decision-log.md) explicitly excluded vocabulary-drift from the blockquote-exemption scope. The rationale was that a forbidden synonym in a quotation would indicate either a stale Charter or a misquoted source. That rationale held when the only blockquotes in scope quoted frozen Charter text. The empirical reality of committee-mode redaction pilots — §2.5 here, and predicted for §2.3, §2.4, §2.6, §3 in future Gates — is systematic stale-quotation of pending stubs by construction.

The §0005 scope is now over-restrictive. The fix replicates the marketing-tell exemption pattern in the vocabulary-drift loop.

## Constitutional Review

- **Q1.** Touches Charter banner (version line and status-line clause). Does not touch §2 header, §2.1, §2.2, §4 binding text, or any other Charter section's prose. The Charter's substantive content is unchanged.
- **Q2.** Does not redefine any glossary term. References the forbidden-synonym entry for `operational construct` exactly as the glossary lists it. Does not modify the glossary.
- **Q3.** Does not resolve any of the five open Ontology questions. The §2.5 redaction itself is the locus of Gate §2.5; this RFC unblocks that redaction's Step 1.1 commit, it does not advance the redaction.
- **Q4.** Yes — this RFC is itself the charter-amendment process for v0.2.1.
- **Q5.** Does not introduce a new invariant. The change is to mechanical enforcement of the existing rule established in §0005.
- **Q6.** Does not propose ceremony. The change has a behavioral consequence: the hook's vocab-drift loop, which currently blocks committee-mode scratch documents whose verbatim stub anchors contain forbidden synonyms by construction, will no longer block them. The block is replaced by committee-review responsibility, which is the layered enforcement that catches genuine mis-quotation per §0012's trade-off record.

## Proposal

Concrete file changes (six files):

1. `.claude/hooks/pre-commit-doc-check.sh` — the vocabulary-drift loop gains an `eligible_lines=$(eligible_blockquote_lines "$added")` computation alongside `canonical_exempt=$(canonical_phrase_exemptions "$added")`, and a corresponding exemption check inside the filtered-hits inner loop before the canonical-phrase check. The check pattern mirrors the marketing-tell loop's existing exemption. Two non-blocking exemptions are documented inline. The header design-discovery count moves 6 → 7; the new seventh discovery names the v0.2.1 extension. The third discovery's text is updated to note the v0.2.1 extension by cross-reference.
2. `docs/charter/constitutional-charter.md` — banner version line `v0.2` → `v0.2.1`; status-line clause appended noting the patch and cross-referencing `decision-log.md` §0012.
3. `docs/charter/amendments.md` — append amendment `v0.2.1` record following the v0.1.1 template (Summary, Rationale, Falsifiability review outcome).
4. `docs/charter/decision-log.md` — append entry §0012 with Context / Decision / Constitutional review / Consequences / Supersession; §0005 cited as superseded-in-part.
5. `.claude/CLAUDE.md` §4 — append a clause to the narrative paragraph below the status table documenting v0.2.1 and cross-referencing `decision-log.md` §0012. The table itself is unchanged.
6. `docs/rfcs/draft/charter-amendment-v0-2-1-extend-blockquote-exemption-to-vocab-drift.md` — this RFC, status `accepted`.

No skill, agent, command, settings, CI, glossary, Ontology document, or other RFC is modified.

## Alternatives Considered

For each occurrence of the stale-stub-vocabulary tripwire, three resolution paths were considered:

- **(a) Bypass with `--no-verify` and decision-log note per CLAUDE.md §5.3.** Acceptable for one-off blocks; not acceptable as the routine path. The §0007 Methodological observation 4 (Gate 1 §4 pilot, Phase F bypass) documents the bypass mechanism for a single amendment commit. Generalizing it to every committee-mode scratch commit across §2.3, §2.4, §2.5, §2.6, §3 establishes a routine bypass precedent inconsistent with CLAUDE.md §5.3's reservation of bypass for unusual cases. Rejected.
- **(c) Localized stub-anchor adaptation in the scratch.** Substitute the forbidden synonym with a lexically-different but semantically-identical word in the anchor, recording the divergence with §0007 Methodological observation 4 as precedent. The §4 pilot Phase F established this path for a Charter-binding-text rendering of a §2 verbatim criterion (the singular-vs-plural noun normalization). Applying it to the in-committee scratch's anchor weakens the anchor's traceability — the anchor's purpose is to record the stub as written, against which binding text is later defended. Rejected at the scratch level; preserved as available at the binding-text level if specific binding text faces an analogous tripwire on a forbidden synonym not exempted by the blockquote rule.
- **(b) Extend the §0005 exemption mechanism to vocabulary-drift.** The chosen path. The fix is mechanical (replicates the marketing-tell loop's existing pattern in the vocab-drift loop), the behavioral consequence is well-bounded (committee review during redaction phases catches genuine mis-quotation), and the empirical mode of committee-mode redaction pilots motivates the change (systematic stale-quotation, not error).

## Open Questions

- Whether the §0005 ambiguity-non-exemption rationale should also be revisited. The ambiguity advisory is non-blocking (informational only), so the case for extension is weaker. The watchlist terms (`state`, `event`, `record`, etc.) appearing in attributed blockquote lines fire advisory NOTEs that the committee reviews during redaction; the advisory output is exactly the audit trail Step 1.4's vocabulary-discipline pass uses. Defer; no extension proposed.
- Whether a future Gate's redaction will surface a stub whose forbidden synonym IS used in the canonical sense (i.e., the binding text intends to inherit the stub's wording at face value). Such a case is not stale-vocabulary but rather material the committee accepts. Step 1.4's vocabulary-discipline pass is the locus for catching this; the hook's blockquote exemption is upstream of that pass and does not pre-decide it. Defer.

## Anti-Patterns to Avoid

- **Treating the blockquote exemption as a routine substitute for committee vocabulary-discipline review.** The exemption unblocks the hook; it does not absolve the committee. Binding text drafted during Steps 1.4–1.5 is still subject to `vocabulary-discipline` and `falsifiability-check`. Forbidden synonyms that survive past the anchor stage are findings, not exemptions.
- **Adding new exemption mechanisms by inference rather than explicit registration.** This RFC extends an existing mechanism (`eligible_blockquote_lines`) to a second loop. It does not introduce a new mechanism. Future similar extensions (e.g., to a hypothetical third blocking check) would require their own RFCs and decision-log entries; pattern-based growth without registration is rejected.
- **Conflating the exemption with the source-of-truth.** The glossary is the source of canonical vocabulary; the hook is a tripwire. The exemption changes which tripwires fire on a per-line basis; it does not change canonical vocabulary. If the §2.5 stub's outcome-of-mutation noun were used in the Category II construct sense (it is not), the canonical vocabulary would still mandate the correction; the exemption would still apply mechanically, and the committee would catch the misuse during Step 1.4 vocabulary review.

## Migration and Backward Compatibility

No historical content affected. No prior commits are re-evaluated. The hook's behavior change is local to future commits scanning attributed Charter/Ontology blockquote lines for forbidden-synonym substrings. Pre-implementation; no operational systems depend on the prior behavior.

## References

- [`decision-log.md` §0005](../../charter/decision-log.md) — original mechanical Charter-quotation exemption (marketing-tell only).
- [`decision-log.md` §0006](../../charter/decision-log.md) — v0.1.1 protected-surface clarification, precedent for patch-level amendments to the hook predicate.
- [`decision-log.md` §0007](../../charter/decision-log.md) Methodological observation 4 — Gate 1 §4 pilot's analogous tripwire on a different forbidden synonym in a §2 verbatim criterion, resolved by pluralization in Phase F.
- [`decision-log.md` §0012](../../charter/decision-log.md) — this amendment's decision-log entry, recording the trade-off and the supersession-in-part of §0005's marketing-only scope.
- [`amendments.md` v0.1.1](../../charter/amendments.md) — template for patch-level mechanical amendment.
- [`anti-marketing` §4](../../../.claude/skills/enforcement/anti-marketing/SKILL.md) — prose-level Charter-quotation exemption (this RFC operationalizes the parallel exemption at the vocabulary-drift level).

## Decision Record

This RFC is recorded as accepted in [`decision-log.md` §0012](../../charter/decision-log.md). Acceptance is recorded by the entry; the RFC remains in `draft/` per [`docs/rfcs/README.md`](../README.md) numbering procedure (numbering and move out of `draft/` happen at a separate, later moment, not in this commit).
