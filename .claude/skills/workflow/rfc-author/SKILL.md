---
name: rfc-author
description: Conduct RFC authorship through the canonical template, with constitutional impact analysis embedded at the start (not bolted on). Use this skill ALWAYS when the user requests to "open an RFC", "propose a change", "document a decision about technology", or "draft an amendment"; ALWAYS when work begins in docs/rfcs/draft/. Skipping the pre-authorship impact analysis is not optional — it is the difference between an RFC and a silent amendment.
---

# rfc-author

RFCs in Ghost Trace are proposals subject to constitutional review ([`docs/rfcs/README.md`](../../../../docs/rfcs/README.md)). The canonical structure is [`docs/rfcs/template.md`](../../../../docs/rfcs/template.md). This skill ensures that authorship begins with constitutional impact analysis — not as an afterthought in the `Constitutional Review` section, but as the entry point.

## 1. Pre-authorship impact analysis

Before any section of the RFC is drafted, the author answers six questions. The answers become the substance of the RFC's `Constitutional Review` section.

### Q1 — Which Charter invariants does this RFC touch?

List the invariants the RFC interacts with, distinguishing FROZEN from PENDING. Source of truth: [`.claude/CLAUDE.md` §4](../../../CLAUDE.md), cross-checked against [`constitutional-charter.md`](../../../../docs/charter/constitutional-charter.md). For each touched invariant, state whether the RFC satisfies it (and how) or violates it (and how).

Delegate the FROZEN/PENDING reading to `charter-guardian` for canonical status.

### Q2 — Does this RFC implicitly redefine any term in the glossary?

For every load-bearing term the RFC uses, look it up in [`docs/glossary.md`](../../../../docs/glossary.md). If the RFC's use diverges from the canonical definition, the RFC is implicitly proposing a vocabulary change. That change is itself an RFC subject (`ontology-revision` if not constitutional; `charter-amendment` if constitutional). Silent redefinition is forbidden.

Delegate to `vocabulary-discipline`.

### Q3 — Does this RFC implicitly resolve any of the five open Ontology questions?

The five open questions are listed in [`docs/ontology/ontology.md` §Open Questions for Committee Resolution](../../../../docs/ontology/ontology.md). For each, ask: does the RFC's proposal, examples, or structural commitments pick an answer? If yes, the resolution must be raised explicitly; the RFC cannot silently resolve.

Delegate to `ontology-keeper`.

### Q4 — Does this RFC require Charter amendment?

If the RFC's proposal cannot be expressed without changing a frozen Charter element, the RFC is reclassified as type `charter-amendment` and the amendment process in [`docs/charter/amendments.md`](../../../../docs/charter/amendments.md) applies in full (RFC + falsifiability review + committee redaction + `amendments.md` entry + version bump).

Delegate to `charter-guardian`.

### Q5 — Does this RFC introduce a new invariant?

A new invariant is constitutional. Per the minimalism rule in [`.claude/CLAUDE.md` §7](../../../CLAUDE.md), the RFC must justify non-redundancy with every existing invariant before adoption. If the proposed invariant overlaps an existing one, recommend non-adoption.

### Q6 — Does this RFC propose ceremony without behavioral consequence?

The minimalism rule also rejects skills, commands, agents, hooks, schemas, or services whose existence does not constrain or enable specific behavior. If the proposal cannot be falsified by deleting it (i.e., nothing observable would change), recommend non-adoption.

## 2. Template walk-through

After the impact analysis is recorded, walk the author through each section of [`docs/rfcs/template.md`](../../../../docs/rfcs/template.md):

- **Summary.** One paragraph. No marketing — `anti-marketing` is applied to this section in particular. Names the proposed change in canonical vocabulary.
- **Motivation.** The cost of *not* making the change must be stated concretely. If the only cost is "the documentation does not yet say so", the proposal is premature.
- **Constitutional Review.** Verbatim output of the Q1–Q6 impact analysis. Not a paraphrase.
- **Proposal.** Concrete. Structural changes, behavioral changes, operational changes, document changes — named, not gestured at. Each claim is verifiable against the substrate or its projections.
- **Alternatives Considered.** At minimum two alternatives. If only one approach was considered, that itself is a failure mode of the analysis; surface it.
- **Open Questions.** What the RFC explicitly defers. Listing nothing here is suspicious; surface it for the author to revisit.
- **Anti-Patterns to Avoid.** By analogy to the Charter's `Forbidden Anti-Patterns` sections ([§2.1](../../../../docs/charter/constitutional-charter.md#21-observational-integrity), [§2.2](../../../../docs/charter/constitutional-charter.md#22-epistemic-separation)). Each anti-pattern is concrete and falsifiable.
- **Migration and Backward Compatibility.** For any change affecting replay ([`replay-model.md`](../../../../docs/architecture/replay-model.md)), state explicitly how historical data is preserved and what the replay contract for the transition is.

## 3. Discipline applied before `status: discussion`

The author does not mark the draft `status: discussion` until three skills have been applied to the full text:

- [`falsifiability-check`](../../epistemic/falsifiability-check/SKILL.md) — every claim runs through the four-question test.
- [`epistemic-separator`](../../epistemic/epistemic-separator/SKILL.md) — every paragraph carries categorical distinctness.
- [`ambiguity-reducer`](../../epistemic/ambiguity-reducer/SKILL.md) — every flagged term is replaced, operationalized, or raised as an open modeling question.

A draft that fails any of these is rewritten before review.

## 4. Numbering procedure

Drafts in [`docs/rfcs/draft/`](../../../../docs/rfcs/) carry working titles, not numbers ([`docs/rfcs/README.md` §Numbering](../../../../docs/rfcs/README.md)).

Numbering happens at acceptance. The procedure:

1. The next sequential RFC number is determined by inspecting accepted RFCs already moved out of `draft/`.
2. The accepted RFC's file is renamed to include the number.
3. The RFC's status field is updated to `accepted`.
4. A decision-log entry is added per [`decision-logger`](../decision-logger/SKILL.md). The entry references the RFC by its new number.
5. The Charter amendments log is updated only if the RFC is type `charter-amendment`.

A draft is never numbered. Numbering before acceptance creates a fiction of decision and is rejected.

## 5. What this skill does not do

This skill does not approve RFCs. It structures authorship. Acceptance is human committee work, recorded through the formal process.

This skill does not write content for the author. It applies the discipline of the project to draft text the author produces. The proposal itself is the author's argument.

## 6. Delegations

This skill composes work that lives in other skills. Composition is explicit so duplication is visible:

| Sub-task | Delegated to |
|---|---|
| Charter status (FROZEN/PENDING); amendment classification (Q1, Q4) | [`constitutional/charter-guardian`](../../constitutional/charter-guardian/SKILL.md) |
| Glossary lookup; canonical-vs-divergent term use (Q2) | [`ontology/vocabulary-discipline`](../../ontology/vocabulary-discipline/SKILL.md) |
| Open-Ontology-question registry; implicit-resolution detection (Q3) | [`ontology/ontology-keeper`](../../ontology/ontology-keeper/SKILL.md) |
| Falsifiability of every claim (§3) | [`epistemic/falsifiability-check`](../../epistemic/falsifiability-check/SKILL.md) |
| Category separation per paragraph (§3) | [`epistemic/epistemic-separator`](../../epistemic/epistemic-separator/SKILL.md) |
| High-risk-term flagging (§3) | [`epistemic/ambiguity-reducer`](../../epistemic/ambiguity-reducer/SKILL.md) |
| Marketing-prose rejection (Summary section in §2) | [`enforcement/anti-marketing`](../../enforcement/anti-marketing/SKILL.md) |
| Subordinate-rank conflict detection on the proposal | [`constitutional/subordination-checker`](../../constitutional/subordination-checker/SKILL.md) |
| Decision-log entry on acceptance (§4) | [`workflow/decision-logger`](../decision-logger/SKILL.md) |

Q5 and Q6 are handled directly here because they apply to the *proposed* invariant or proposed ceremony, which does not yet have a home in any other skill.

## 7. Source citations used

- [`docs/rfcs/README.md`](../../../../docs/rfcs/README.md)
- [`docs/rfcs/template.md`](../../../../docs/rfcs/template.md)
- [`docs/charter/constitutional-charter.md` §2.1, §2.2 (anti-pattern reference)](../../../../docs/charter/constitutional-charter.md)
- [`docs/charter/amendments.md` §Amendment Process](../../../../docs/charter/amendments.md)
- [`docs/charter/decision-log.md`](../../../../docs/charter/decision-log.md)
- [`docs/ontology/ontology.md` §Open Questions for Committee Resolution](../../../../docs/ontology/ontology.md)
- [`docs/architecture/replay-model.md`](../../../../docs/architecture/replay-model.md)
- [`docs/glossary.md`](../../../../docs/glossary.md)
- [`.claude/CLAUDE.md` §4 Charter status; §7 Constitutional minimalism](../../../CLAUDE.md)
