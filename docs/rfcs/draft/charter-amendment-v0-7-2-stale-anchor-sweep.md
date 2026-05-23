# RFC — Charter Amendment v0.7.2: Q2/Q3/Q5 stale-anchor sweep across §2.3, §2.4, §2.6

- **Status:** accepted
- **Authors:** committee
- **Date:** 2026-05-23
- **Type:** charter-amendment
- **Affects:** Charter banner (version line + status line). Seven prose locations across §2.3, §2.4, §2.6 — each updating a stale Q-pending cross-reference to its actual resolution entry. CLAUDE.md §4 status table narrative paragraph (v0.7.2 chronological clause). No Charter binding-text semantics amended.

> RFCs in Ghost Trace are proposals subject to constitutional review. The [Constitutional Charter](../../charter/constitutional-charter.md) is the authoritative reference.

## Summary

At §2.3 freeze ([`decision-log §0017`](../../charter/decision-log.md), amendment v0.4), §2.4 freeze ([`§0099`](../../charter/decision-log.md), v0.5), and §2.6 freeze ([`§0129`](../../charter/decision-log.md), v0.6), each respective Boundary Condition anchored to the then-pending Ontology Open Modeling Question — Q2 (Identity tiers), Q3 (formal independence definition), Q5 (transitive scope) — with the formulation *"formal mechanism becomes structurally falsifiable when QN resolution lands"*. All three Qs subsequently resolved: Q2 at [`§0023`](../../charter/decision-log.md), Q3 at [`§0133`](../../charter/decision-log.md), Q5 at [`§0134`](../../charter/decision-log.md). Seven Charter prose locations continued to describe the formal mechanism as pending after the respective resolutions landed. Amendment v0.7.2 corrects all seven references in a single sweep, advancing the Charter banner from `v0.7.1` to `v0.7.2`. No binding-text semantics are amended — each BC's binding commitment ("§X does not govern Y formal specifics") is preserved verbatim; only the status of the formal mechanism the BC anchors to changes from "pending QN" to "resolved at §NNNN".

## Motivation

The discrepancy surfaced during pre-RFC framing review for the upcoming [Domain Pack v0.1 — anti-bot atlas](./) RFC. That RFC will cite §2.4 BC6 + §2.6 BC6 (transitive scope of `influenced_by` chains) as part of its real-time-vs-substrate-immutability framing; citing them with the stale "pending Q5" formulation would propagate the stale anchor into the new RFC.

Anchor-inventory scan against the current Q-resolution status surfaced seven instances of `pending QN` / `when QN resolution lands` patterns in Charter prose:

| Line | Section | Stale Q reference | Resolution entry |
|---|---|---|---|
| 115 | §2.3 Structural Requirement prose | Q2 pending | [`§0023`](../../charter/decision-log.md) |
| 136 | §2.3 BC4 | Q2 pending | [`§0023`](../../charter/decision-log.md) |
| 182 | §2.4 BC4 | Q2 pending | [`§0023`](../../charter/decision-log.md) |
| 186 | §2.4 BC6 | Q5 pending | [`§0134`](../../charter/decision-log.md) |
| 271 | §2.6 BC1 | Q3 pending | [`§0133`](../../charter/decision-log.md) |
| 279 | §2.6 BC5 | Q2 pending | [`§0023`](../../charter/decision-log.md) |
| 281 | §2.6 BC6 | Q5 pending | [`§0134`](../../charter/decision-log.md) |

**Functional protection was not lost.** Each BC's binding commitment ("§X does not govern Y formal specifics") is a structural rule independent of which entry the formal mechanism is anchored to. The stale anchors were status-reporting drift, not enforcement-relevant.

The fix is nonetheless required because:
- The Charter prose should reflect the current state of Ontology resolutions. A BC that says "becomes falsifiable when Q5 resolution lands" when Q5 resolved over a month ago misrepresents the structural ground.
- The [Domain Pack v0.1 anti-bot atlas](./) RFC and subsequent work will cite these BCs; citing them pre-correction propagates stale anchors into new RFCs.
- The [`§0137`](../../charter/decision-log.md) Methodological observation 3 prescribed anchor-inventory verification as a Step 1.1 augmentation for future §-section redactions; this patch is the first deliberate enaction of that prescription (prior patch-via-pressure instances all surfaced from incidental observation).

## Constitutional Review

- **Q1 — Charter invariants touched?** None in content. Each affected BC retains its binding commitment verbatim. The Charter banner version line + status line are revised (`v0.7.1` → `v0.7.2`). The seven prose locations revise cross-reference pointers + the falsifiability-status word ("becomes ... when ... lands" → "is ... per §NNNN"); the BC's logical structure is preserved.
- **Q2 — Glossary?** Not affected.
- **Q3 — Ontology open questions?** Q2, Q3, Q5 resolutions referenced (not amended). Each was previously resolved via its own decision-log entry — §0023, §0133, §0134 respectively.
- **Q4 — Is this RFC the amendment?** Yes. Patch-level amendment per [`amendments.md` Step 5](../../charter/amendments.md).
- **Q5 — Falsifiability?** Pass. The patch's correctness predicate is structurally observable: each of the seven references' post-patch target (`§0023`, `§0133`, or `§0134`) exists in the decision-log and carries content matching its Q-resolution claim. Mechanical diff inspection verifies the patch.
- **Q6 — Ceremony or constitutional?** Patch-level Charter cross-reference correctness. The fix restores prose fidelity to the current Ontology resolution status; it does not change Charter binding commitments.

## Proposal

Seven text changes in `docs/charter/constitutional-charter.md`, each replacing a stale Q-pending formulation with a resolved-at-§NNNN reference. The replacement preserves the BC's binding commitment (the `**§X does not govern Y formal specifics**` lead clause) and the surrounding contextual prose; only the formal-mechanism-status sentence is revised.

Pattern A (Q2 — four sites at L115, L136, L182, L279):
```
- Before: "...formal specification pending [Identity tiers — Open Modeling Question 1]
  (../ontology/entity-model.md#open-modeling-questions). The default-level commitment is
  binding today; the formal mechanism becomes structurally falsifiable when Q2 resolution lands."
- After:  "...inception-phase single-tier `actor_ref` adopted per [`§0023`](./decision-log.md)
  (Identity tiers — Open Modeling Question 1 resolution: single-tier at inception; multi-tier
  formalization deferred to ordinary Ontology RFC discipline). The single-tier commitment is
  binding today and structurally falsifiable; multi-tier formalization becomes structurally
  falsifiable when subsequently adopted via Ontology RFC."
```

Pattern B (Q3 — one site at L271):
```
- Before: "...the formal definition of `independence` as a measurable quantity is operational
  specification deferred to [Q3 of `ontology.md`](../ontology/ontology.md) Open Questions.
  The default-level commitment is binding today; the formal mechanism becomes structurally
  falsifiable when Q3 resolution lands."
- After:  "...the formal definition of `independence` as a measurable quantity is operational
  specification resolved per [`§0133`](./decision-log.md) (Q3 resolution — Candidate α:
  source-count ratio over Cat I provenance roots, under §2.6 BC2 meta-shape 1
  deterministic-from-pattern). The pairing commitment is binding today and structurally
  falsifiable per §0133."
```

Pattern C (Q5 — two sites at L186, L281):
```
- Before: "...formal specification of transitivity pending the 'transitive?' half of
  [`ontology.md` Q5](../ontology/ontology.md). The Ontology-level commitment is binding
  today; the formal mechanism becomes structurally falsifiable when Q5 resolution lands."
- After:  "...formal specification of transitivity resolved per [`§0134`](./decision-log.md)
  (Q5 transitivity-half — Candidate τ: transitive closure of declared direct `influenced_by`
  edges, with β-graph storage). The Ontology-level commitment is binding today and
  structurally falsifiable per §0134."
```

Charter banner revisions:
- Version line `v0.7.1` → `v0.7.2`.
- Status line: patch-amendment clause appended noting the seven-anchor sweep and reference to `decision-log §0142`.

## Alternatives Considered

- **One patch amendment per Q-resolution** (three separate amendments: v0.7.2 for Q2, v0.7.3 for Q3, v0.7.4 for Q5) — rejected. Each Q's stale anchors share the same pattern shape and correction shape; bundling per [v0.7.1](../../charter/amendments.md) Methodological observation 2 (bundle-by-shape) precedent is the lower-ceremony option for the same audit-trail surface.
- **Inline correction without amendment** — rejected. Patch-amendment ceremony per [v0.7.1](../../charter/amendments.md) precedent makes the audit trail explicit; Charter prose corrections require amendment per [`amendments.md`](../../charter/amendments.md) §Amendment Discipline.
- **Defer until next §-section redaction** — rejected. No further §-section redactions are planned (Charter is fully frozen as of v0.7); waiting for the next downstream RFC to surface stale anchors loses the deliberate-inventory discipline that this patch demonstrates.
- **Retain "becomes falsifiable when QN resolution lands" prose with §NNNN footnote** — rejected. The future-tense construction is semantically incorrect once QN is resolved; the surface form should reflect the actual structural state.

## Open Questions

None.

## Anti-Patterns to Avoid

- Adding new BC formulations of the form "...pending [QN of `ontology.md`](../ontology/...)..." without a corresponding decision-log carry-forward to revise the BC when QN resolves. Future §-section redactions or future BC additions to existing §-sections should either (a) cite the resolution entry directly if the resolution has landed, or (b) anchor to a specific resolution-watcher decision-log entry that explicitly carries forward the revision obligation.
- Treating Charter prose status-reporting drift as cosmetic-only. The [`§0137`](../../charter/decision-log.md) precedent (six stale `§0034` references that pointed to an entry containing unrelated content) demonstrates that stale anchors can mislead readers materially even when the binding commitment is preserved. Anchor inventory is part of constitutional discipline.
- Bundling structurally-unrelated patches into a single amendment. This patch bundles seven Q-stale anchors because they share pattern shape; bundling them with, e.g., a hook fix or a typography correction would dilute the audit trail.

## Migration and Backward Compatibility

No migration. The Charter binding commitments are unchanged; subordinate documents (Ontology, Architecture, RFCs) that cite §2.3 BC4 / §2.4 BC4 / §2.4 BC6 / §2.6 BC1 / §2.6 BC5 / §2.6 BC6 continue to cite the same BCs with the same binding commitments — only the formal-mechanism anchors are revised.

Existing RFCs that quoted the pre-patch BC text (e.g., [`ontology-revision-q3-independence`](./ontology-revision-q3-independence.md) at line 53) preserve historical fidelity to the state at proposal time; no retroactive revision to accepted-status RFCs.

## References

- [`decision-log §0017`](../../charter/decision-log.md) — Gate §2.3 closure (introduced Q2 forward-reference at §2.3 BC4).
- [`decision-log §0023`](../../charter/decision-log.md) — Q2 (Identity tiers) resolution.
- [`decision-log §0099`](../../charter/decision-log.md) — Gate §2.4 closure (introduced Q2 + Q5 forward-references at §2.4 BC4 + BC6).
- [`decision-log §0129`](../../charter/decision-log.md) — Gate §2.6 closure (introduced Q3 + Q2 + Q5 forward-references at §2.6 BC1 + BC5 + BC6).
- [`decision-log §0133`](../../charter/decision-log.md) — Q3 (formal independence) resolution.
- [`decision-log §0134`](../../charter/decision-log.md) — Q5 (transitivity-half) resolution.
- [`decision-log §0137`](../../charter/decision-log.md) — v0.7.1 patch amendment + Methodological observation 3 (anchor-inventory checklist).
- [`decision-log §0142`](../../charter/decision-log.md) — this RFC's acceptance record.
- [`amendments.md` v0.7.2 entry](../../charter/amendments.md).
- [`amendments.md` v0.7.1 entry](../../charter/amendments.md) — patch-via-pressure precedent.

## Decision Record

Accepted via [`decision-log §0142`](../../charter/decision-log.md). Charter advances to v0.7.2.
