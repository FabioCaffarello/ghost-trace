---
name: implementation-readiness-evaluator
description: Replace the binary "implementation is blocked" gate with an explicit, falsifiable readiness evaluation per component. Use this skill ALWAYS when the user requests to "start implementing", "build a service", "write a schema", "add code", or "prototype something"; ALWAYS when a commit adds files under services/ or non-experimental files under schemas/. Returns ready, partially-ready (exploration only), or not-ready, with the unmet criteria named explicitly.
---

# implementation-readiness-evaluator

The project is in pre-implementation constitutional drafting ([`README.md` §Status](../../../../README.md); [`.claude/CLAUDE.md` §1](../../../CLAUDE.md)). Implementation work begins after the Charter and the relevant portions of the Ontology are stable, with the gate defined here.

The skill replaces a binary "blocked" verdict with a per-component readiness evaluation. Different components have different readiness conditions; some are ready now (Tier-0 pure-observation ingestion); most are not.

## 1. Readiness criteria

Each criterion is independently checkable. None requires subjective judgment.

### Criterion 1 — Charter completeness for the relevant slice

The Charter invariants governing the component must be FROZEN, not PENDING. Status is read from [`.claude/CLAUDE.md` §4](../../../CLAUDE.md), cross-checked against [`constitutional-charter.md`](../../../../docs/charter/constitutional-charter.md).

| Component | Required FROZEN invariants |
|---|---|
| Pure ingestion of observations (commit-only path) | §2.1, §2.2 (already FROZEN) |
| Assertion engine | §2.1, §2.2, §2.3, §2.4, §2.5, §2.6 |
| Replay (deterministic phases 1–2) | §2.1, §2.2, §2.3 |
| Replay (reconstructive phase 3) | §2.1, §2.2, §2.3, §2.4, §2.5 |
| Graph projection | §2.1, §2.2, §2.3, §2.4 |
| Promotion/demotion pipeline | §2.4, §2.5 |

The table grows as components are scoped via RFC. Entries are not removed; mappings are immutable once recorded.

### Criterion 2 — Ontology stability for the relevant slice

The Ontology sections the component depends on must be redacted, not scaffold. The five open modeling questions in [`ontology.md`](../../../../docs/ontology/ontology.md) that touch the component must be resolved or explicitly declared out of scope for this implementation. Out-of-scope declarations are recorded in the originating RFC; silent scope reduction is not allowed.

Delegate the open-question registry to [`ontology/ontology-keeper`](../../ontology/ontology-keeper/SKILL.md).

### Criterion 3 — Storage substrate decision

If the component requires a specific storage substrate, the relevant `architecture` RFC must be accepted and recorded in the decision log. Current status: open per [`decision-log.md` §0003](../../../../docs/charter/decision-log.md). Components that interact with the substrate cannot proceed until §0003 is superseded by an acceptance decision.

Components that operate purely at the schema/contract layer, without committing to a substrate, can proceed under Criterion 5.

### Criterion 4 — Schema technology decision

If the component requires concrete schemas ([`schemas/`](../../../../schemas/)), the schema-technology RFC must be accepted. Current status: open per [`schemas/README.md`](../../../../schemas/README.md). No schemas may be committed before the RFC is accepted.

### Criterion 5 — No silent open-question resolution

The implementation must not resolve, in code, any of the five open Ontology questions. Code that picks an answer to an open question is a constitutional move dressed as implementation work; surface it for resolution via `ontology-revision` RFC before proceeding.

Delegate to [`ontology/ontology-keeper`](../../ontology/ontology-keeper/SKILL.md).

## 2. Readiness states

The evaluator returns one of three states, with the unmet criteria named.

### Ready

All criteria relevant to the proposed component are met. Implementation may proceed.

### Partially ready — exploration only

Some criteria met. Only work that does not commit to the substrate or commit schemas is permitted. The permitted form is exploration in [`experiments/`](../../../../experiments/), governed by an RFC of type `experiment` per [`experiments/README.md`](../../../../experiments/README.md). The experiment must declare its hypothesis, success criteria, time bound, and the decision it informs.

An experiment that demonstrates technical feasibility but violates a constitutional invariant is not eligible for promotion to production ([`experiments/README.md` §Experiment Discipline](../../../../experiments/README.md)).

### Not ready

Insufficient criteria met. Implementation work is blocked. The evaluator lists exactly which criteria are unmet and what would unblock them (typically: a specific committee redaction, a specific RFC acceptance, a specific decision-log supersession).

## 3. Evaluation procedure

Apply in order.

1. **Identify the component or change.** Name the component as it appears in [`services/`](../../../../services/) or as proposed in an RFC. If the proposed work crosses multiple components, evaluate each separately.
2. **List governing Charter invariants and Ontology sections.** Use the table in Criterion 1 if the component is listed; otherwise consult the source documents directly. Document the mapping.
3. **Check each readiness criterion** for the listed invariants and sections. Each criterion is yes/no.
4. **Return the readiness state** plus the explicit list of unmet criteria. The state is the conjunction of criteria; one unmet criterion is enough to demote the verdict.

The verdict is recorded — either in the originating RFC's `Constitutional Review` section, or in a decision-log entry if the evaluation precedes an RFC.

## 4. Exploration vs implementation

These are distinct categories of work. Conflating them defeats the gate.

| Aspect | Exploration | Implementation |
|---|---|---|
| Location | [`experiments/`](../../../../experiments/) | [`services/`](../../../../services/), [`schemas/`](../../../../schemas/) |
| Authorization | RFC of type `experiment` | Ready verdict from this skill |
| Time-bound | Required | Not applicable |
| Promotion to production | Requires `architecture` RFC | Already in production scope |
| Substrate commitment | Forbidden | Required (where applicable) |
| Schema commitment | Forbidden | Required (where applicable) |

Exploration is the legitimate path when criteria are partially met. Calling implementation "exploration" to bypass the gate is itself a failure mode; the gate checks for `experiments/` location and an `experiment`-type RFC.

## 5. Non-negotiability

The criteria are themselves subject to constitutional review; they are not negotiable conversationally. A user request to soften a criterion ("can we just ... for now?") is escalated to an RFC. The skill does not relax criteria under pressure.

If the criteria themselves are wrong, the path to changing them is the same path that changes everything else: an RFC, with constitutional review. The criteria are not edited by appeal.

## 6. What this skill does not do

This skill does not approve implementation work. It returns a readiness verdict. Approval of an actual implementation merge is the normal review process plus the verdict from this skill.

This skill does not generate code. When the verdict is `ready`, the skill returns; it does not author the implementation.

## 7. Delegations

| Sub-task | Delegated to |
|---|---|
| Charter status (FROZEN/PENDING) for Criterion 1 | [`constitutional/charter-guardian`](../../constitutional/charter-guardian/SKILL.md) |
| Open-Ontology-question registry for Criteria 2 and 5 | [`ontology/ontology-keeper`](../../ontology/ontology-keeper/SKILL.md) |
| Decision-log entries that record the verdict | [`workflow/decision-logger`](../decision-logger/SKILL.md) |
| RFC authorship when a criterion change is proposed | [`workflow/rfc-author`](../rfc-author/SKILL.md) |

## 8. Source citations used

- [`README.md` §Status; §How to Read This Repository](../../../../README.md)
- [`docs/charter/constitutional-charter.md` §2.1–§2.6](../../../../docs/charter/constitutional-charter.md)
- [`docs/charter/decision-log.md` §0003 — Defer storage technology selection](../../../../docs/charter/decision-log.md)
- [`docs/ontology/ontology.md` §Open Questions for Committee Resolution](../../../../docs/ontology/ontology.md)
- [`schemas/README.md`](../../../../schemas/README.md)
- [`services/README.md`](../../../../services/README.md)
- [`experiments/README.md`](../../../../experiments/README.md)
- [`.claude/CLAUDE.md` §1 What this repository is; §4 Charter status; §6 Operational rules (rule 4)](../../../CLAUDE.md)
