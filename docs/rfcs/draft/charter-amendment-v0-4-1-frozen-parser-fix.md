# RFC — Charter Amendment v0.4.1: Fix frozen-section parser to accept amendment-qualified status cells

- **Status:** accepted
- **Authors:** committee
- **Date:** 2026-05-18
- **Type:** charter-amendment
- **Affects:** Charter banner (version line + status line). Hook parser (`.claude/hooks/_parse_watchlists.py`). No Charter prose amended.

> RFCs in Ghost Trace are proposals subject to constitutional review. The [Constitutional Charter](../../charter/constitutional-charter.md) is the authoritative reference.

## Summary

The frozen-section parser at `_parse_watchlists.py:54` uses the regex `^\|\s*(§[0-9.]+)[^|]*\|\s*frozen\s*\|` to extract frozen-section markers from the CLAUDE.md §4 status table. The strict cell match `\|\s*frozen\s*\|` requires the status cell to contain only `frozen` with optional surrounding whitespace. Cells with the `frozen — minor amendment vN.Y` qualifier — introduced by amendment v0.3 ([`decision-log §0013`](../../charter/decision-log.md)) for newly-frozen sections — silently fail to match. Consequence: §2.5 (frozen v0.3) and §2.3 (frozen v0.4) were not recognized by the parser despite their CLAUDE.md table rows correctly marking them frozen. Amendment v0.4.1 fixes the regex to `\|\s*frozen[^|]*\|` (allowing arbitrary non-pipe content after `frozen`), and bumps the Charter banner to v0.4.1. No Charter prose is amended.

## Motivation

The discrepancy was surfaced during Gate §2.3 closure ([`decision-log §0017`](../../charter/decision-log.md) Phase F.4 final summary). Post-merge hook self-test reported `frozen ranges: 5 range(s)` when the expected count was 7 (§1 + §2 + §2.1 + §2.2 + §2.3 + §2.5 + §4). Investigation traced the gap to the strict regex.

**Functional protection was not lost.** The parser produces both the per-section ranges AND the §2 outer range (`32:206` in current Charter), which wraps all §2.x sub-sections. Direct edits to §2.3 or §2.5 binding text still BLOCK via the §2 outer range (which is correctly recognized because the `§2` row in CLAUDE.md uses bare `frozen` without qualifier). So §2.3 and §2.5 are incidentally protected — the gap is **cosmetic and audit-trail-relevant**, not enforcement-relevant.

The fix is nonetheless required because:
- The table-driven parser should reflect the table's actual claims. §2.5's "frozen — minor amendment v0.3" row claims §2.5 is frozen; the parser must agree.
- Future Charter structure changes (e.g., adding §2.7 pending while keeping §2.3/§2.5 frozen) could create a configuration where the §2 outer range no longer wraps every frozen §2.x. The fix prevents that latent fragility.
- Self-test output is part of the hook's audit surface. A persistently undercounted self-test undermines confidence in the hook's enforcement claims.

## Constitutional Review

- **Q1 — Charter invariants touched?** None in content. The Charter banner version line and status line are updated (Charter v0.4 → v0.4.1) per the amendment-versioning rule established in [`amendments.md`](../../charter/amendments.md). No frozen invariant prose is amended.
- **Q2 — Glossary?** Not affected.
- **Q3 — Ontology open questions?** Not affected.
- **Q4 — Is this RFC the amendment?** Yes. Patch-level amendment per [`amendments.md` Step 5](../../charter/amendments.md).
- **Q5 — Falsifiability?** Pass. The fix clarifies a mechanical predicate (the cell-match regex). The new predicate is structurally falsifiable: a status cell beginning with `frozen` and containing arbitrary non-pipe trailing content matches; a cell containing `pending` or any other word does not match. Detection is mechanical, no subjective judgment.
- **Q6 — Ceremony or constitutional?** Patch-level hook correctness. The fix restores parser fidelity to the table's claims; it does not change Charter enforcement semantics (the §2 outer range already provided incidental §2.x protection).

## Proposal

Single-character regex change in `.claude/hooks/_parse_watchlists.py`:

```diff
-    for m in re.finditer(r'^\|\s*(§[0-9.]+)[^|]*\|\s*frozen\s*\|', s, re.MULTILINE):
+    for m in re.finditer(r'^\|\s*(§[0-9.]+)[^|]*\|\s*frozen[^|]*\|', s, re.MULTILINE):
```

`\s*` → `[^|]*` in the status-cell side of the alternation. Matches `frozen` followed by any non-pipe content (including the qualifier `— minor amendment vN.Y`).

Charter banner updates:
- Version line `v0.4` → `v0.4.1`.
- Status line: patch-amendment clause appended noting the parser fix and reference to `decision-log §0018`.

## Alternatives Considered

- **Tighter regex** (`frozen(\s+—\s+minor amendment v[0-9.]+)?`) — rejected. Couples the parser to the specific qualifier wording; future qualifier variants (e.g., "frozen — pending revision" or "frozen — patch amendment vN.Y.Z") would need additional parser updates. The chosen `frozen[^|]*` is permissive enough to admit any human-readable qualifier without future parser changes.
- **Restructure the table to use separate columns for status and qualifier** — rejected. Larger surface change; touches CLAUDE.md narrative + every consumer of the table; not justified for the cosmetic fix.
- **Hook-maintenance commit without amendment** — rejected per user direction. Patch-amendment ceremony per v0.2.1 precedent makes the audit trail explicit.

## Open Questions

None.

## Anti-Patterns to Avoid

- Adding more strict-match regexes elsewhere in the parser without considering qualifier-tolerance. Future patch amendments may introduce additional qualifier conventions; the parser's other status-detection logic (e.g., `pending`-cell detection if added) should follow the same permissive shape.
- Treating self-test count discrepancies as cosmetic-only without investigating root cause. The Gate §2.3 closure observation that surfaced this bug demonstrates the value of treating self-test output as authoritative — if it disagrees with expectation, investigate before dismissing.

## Migration and Backward Compatibility

No migration. Hook behavior change is forward-only: post-merge, self-test reports 7 frozen ranges (was 5). Existing committed §2.3 and §2.5 binding text remains protected — the §2 outer range continues to wrap them; the per-section ranges are now also recognized (additive, no semantics change).

## References

- [`decision-log §0007`](../../charter/decision-log.md) — Gate 1 §4 redaction (first to use "frozen" status without qualifier).
- [`decision-log §0012`](../../charter/decision-log.md) — v0.2.1 patch amendment (hook fix precedent).
- [`decision-log §0013`](../../charter/decision-log.md) — Gate §2.5 closure (introduced the "frozen — minor amendment vN.Y" qualifier convention).
- [`decision-log §0017`](../../charter/decision-log.md) — Gate §2.3 closure (Phase F.4 surfaced the parser discrepancy).
- [`decision-log §0018`](../../charter/decision-log.md) — this RFC's acceptance record.
- [`amendments.md` v0.2.1 entry](../../charter/amendments.md) — patch-amendment ceremony template.
- [`amendments.md` v0.4.1 entry](../../charter/amendments.md).

## Decision Record

Accepted via [`decision-log §0018`](../../charter/decision-log.md). Charter advances to v0.4.1.
