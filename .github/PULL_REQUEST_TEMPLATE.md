## Summary

One paragraph: what this PR changes and why.

## Document rank touched

Which rank of document does this PR modify? (See [README.md §Document Hierarchy](../README.md).)

- [ ] Constitutional Charter (rank 1) — requires amendment per [`amendments.md`](../docs/charter/amendments.md)
- [ ] Ontology (rank 2)
- [ ] Architecture (rank 3)
- [ ] RFC (rank 4)
- [ ] Schemas (rank 5)
- [ ] Services (rank 6)
- [ ] Project infrastructure (CLAUDE.md, hooks, CI, skills, glossary, etc. — not part of the rank hierarchy but subject to its own discipline)

## Constitutional gates crossed

- [ ] No Charter section was modified.
- [ ] A FROZEN Charter section was modified, accompanied by an `amendments.md` entry and the required RFC.
- [ ] A `pending` Charter section was redacted in committee mode (per [`invariant-redactor`](../.claude/skills/constitutional/invariant-redactor/SKILL.md)).
- [ ] A subordinate document conflict was resolved by revising the subordinate document (never the higher-ranked one).
- [ ] None of the above (mechanical fix, editorial, etc.).

## Falsifiability review

For PRs that touch the Charter, the Ontology, or an RFC's `Constitutional Review` section: state the outcome of applying [`falsifiability-check`](../.claude/skills/epistemic/falsifiability-check/SKILL.md) to each new or modified claim.

For other PRs: write `n/a`.

## Decision-log entry

If this PR enacts a decision (accepted RFC, amendment, calendar change, watchlist extension, governance clarification), reference the decision-log entry ID. If none, write `none`.

## Hook output

Run `bash .claude/hooks/pre-commit-doc-check.sh --self-test` and paste the counts. If the counts changed from the baseline (5 / 12 / 36 / 13 as of 2026-05-15), explain why.

## Reviewer notes

Anything the reviewer should know but that does not fit in the above sections.

---

Reviews evaluate alignment with the Charter, not with this template. The template surfaces what reviews check; the Charter is what is checked against.
