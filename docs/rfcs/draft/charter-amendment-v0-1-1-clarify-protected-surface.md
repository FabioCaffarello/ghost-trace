# RFC — Charter Amendment v0.1.1: Clarify Protected Surface

- **Status:** discussion
- **Authors:** Ghost Trace governance (post-setup work, Gate 0b)
- **Date:** 2026-05-15
- **Type:** charter-amendment
- **Affects:** Charter banner (version line; status line); Charter §2 header (status change to FROZEN); `.claude/CLAUDE.md` §4 status table (new row); `.claude/hooks/pre-commit-doc-check.sh` (`in_scope` predicate; canonical-phrase exemption; no-added-lines fix); `.claude/skills/ontology/vocabulary-discipline/SKILL.md` §4 (new subsection)

> RFCs in Ghost Trace are proposals subject to constitutional review. The [Constitutional Charter](../../charter/constitutional-charter.md) is the authoritative reference. RFCs that conflict with the Charter must either be rejected or accompanied by a corresponding charter amendment (see [`../../charter/amendments.md`](../../charter/amendments.md)).

## Summary

This RFC clarifies, in two orthogonal dimensions, the constitutional surface protected by the infrastructure. The §2 header — the four invariant qualification criteria — is explicitly declared FROZEN. The hook's protected file set is explicitly enumerated and extended to include `.claude/CLAUDE.md` and `.claude/README.md`.

## Motivation

The Phase 8 SELF-AUDIT (§7 Finding 7.1) surfaced ambiguity in the Charter banner's wording ("Invariants 1–2 frozen") about whether the §2 header is frozen. Empirical investigation during Gate 0 (post-Gate-0a) further surfaced that the hook's `in_scope` predicate never included `.claude/CLAUDE.md` or `.claude/README.md`, despite the SELF-AUDIT having treated their cleanliness as a finding. Both ambiguities are governance-adjacent and warrant explicit resolution rather than continued reliance on implicit assumption.

## Constitutional Review

- **Q1.** Touches §2 (FROZEN — being explicitly declared so), §2.1 (FROZEN — unchanged), §2.2 (FROZEN — unchanged). Does not modify the substantive text of any frozen invariant. Modifies the Charter banner (version line and status-line summary).
- **Q2.** Does not redefine any glossary term. References "primary event log", "decision log", and "historical fact" exactly as the glossary uses them. Introduces "canonical phrase" as a hook-level exemption mechanism, not a glossary term.
- **Q3.** Does not resolve any of the five open Ontology questions.
- **Q4.** Yes — this RFC is itself the charter-amendment process for v0.1.1.
- **Q5.** Does not introduce a new invariant.
- **Q6.** Does not propose ceremony. Each change has a behavioral consequence: the §2 header becomes protected by the hook; `.claude/CLAUDE.md` and `.claude/README.md` become protected; the canonical-phrase exemption prevents false-positives on legitimate canonical phrases; the no-added-lines fix prevents false-positives on whitespace-only commits.

## Proposal

Concrete file changes (eight files):

1. `docs/charter/constitutional-charter.md` — banner version line `v0.1` → `v0.1.1`; banner status line rewritten to enumerate the §2 header alongside §2.1 and §2.2 as frozen.
2. `docs/charter/amendments.md` — append amendment `v0.1.1` record.
3. `.claude/CLAUDE.md` §4 — insert new row `| §2 Invariant qualification criteria | frozen |` immediately after the §1 Thesis row; append one sentence to the paragraph after the table referencing the SELF-AUDIT finding resolved.
4. `.claude/CLAUDE.md` §5.3 — editorial rewrite at the end of the advisory-class description: the predicate now reads "The advisory is informational; the author decides."
5. `.claude/README.md` line 58 — editorial rewrite: the agents-section sentence now reads "They recommend RFCs or surface drift."
6. `.claude/README.md` line 80 — editorial rewrite: the procedures-section sentence now references "how to record a decision" rather than the prior verb form.
7. `.claude/hooks/pre-commit-doc-check.sh` — extend `in_scope` to admit `.claude/CLAUDE.md` and `.claude/README.md`; add `CANONICAL_PHRASES` whitelist and a `canonical_phrase_exemptions` helper; filter the vocabulary-drift inner loop by the exemption set; replace the empty-`added` fallback with `continue`.
8. `.claude/skills/ontology/vocabulary-discipline/SKILL.md` §4 — add subsection documenting the canonical-phrase exemption mechanism and the registered list.

A ninth file, `docs/charter/decision-log.md`, receives the append-only entry `0006` recording acceptance.

## Alternatives Considered

For Decision 1 (§2 header status):

- **(a) Treat the §2 header as pending.** Rejected: the four qualification criteria are operationally in effect via `invariant-redactor`, which structures committee redaction against them. Declaring them pending would contradict an existing operative procedure.
- **(b) Treat as editorial fix instead of amendment.** Rejected: the change adjusts what the hook protects, which is a structural consequence and warrants the amendment process even at patch level.

For Decision 2 (hook scope):

- **(a) Include `.claude/SELF-AUDIT.md` and `.claude/PLAN.md`.** Rejected: these documents register findings *about* watchlist terms; protecting them under vocabulary discipline would prevent honest observability of the infrastructure.
- **(b) Include `.claude/skills/**`.** Rejected: skills define watchlists and procedures as content; protection would generate widespread false-positives on the very terms the skills enumerate.

For Decision 5 (canonical-phrase exemption):

- **(a) Accept all forbidden hits as bypass via `--no-verify` with decision-log notes.** Rejected: establishes a routine bypass precedent for legitimate canonical vocabulary; the bypass mechanism is reserved for unusual cases per `CLAUDE.md` §5.3.
- **(b) Rewrite all canonical phrases to avoid the substrings.** Rejected: would damage canonical vocabulary — "primary event log" is the canonical name per `vocabulary-discipline` §4 and `storage-model.md`.

## Open Questions

- Whether `WORKFLOW.md` should also gain canonical-phrase coverage. It is currently in scope for the hook, but a similar pre-scan was not performed in Gate 0b. Defer.
- Whether a future `.claude/glossary.md` or comparable document would warrant scope addition. Defer until such a document exists.

## Anti-Patterns to Avoid

- **Growing the canonical-phrase whitelist by inference rather than explicit registration.** New entries require a new decision-log entry referencing `vocabulary-discipline` §4. Pattern-based expansion is rejected.
- **Expanding hook scope without enumerating the rationale per file.** The `in_scope` predicate is documented in-line; additions follow the same pattern.
- **Routine use of `--no-verify` to bypass legitimate hits.** Bypass is registrable per `CLAUDE.md` §5.3.

## Migration and Backward Compatibility

No historical content affected. The hook's behavior change is local to future commits. The Charter's substantive text is unchanged; the version bump records the clarification.

## References

- Phase 8 SELF-AUDIT (`.claude/SELF-AUDIT.md` §7 Finding 7.1).
- Decision-log entry `0005` (Gate 0a — mechanical Charter-quotation exemption).
- `vocabulary-discipline` §4 forbidden-synonym table.
- `charter-guardian` §1 protected-vs-pending element list.

## Decision Record

This RFC is recorded as accepted in decision-log entry `0006`. Acceptance is recorded by the entry; the RFC remains in `draft/` per `docs/rfcs/README.md` numbering procedure (numbering and move out of `draft/` happen at a separate, later moment, not in this commit).
