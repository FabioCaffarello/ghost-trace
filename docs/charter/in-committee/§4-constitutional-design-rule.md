# §4 Constitutional Design Rule — committee draft

**Status:** in-committee draft. Not binding. Do not cite as authoritative.

## Anchor (verbatim from Charter §4 stub)

> This section formalizes the meta-rule that governs the Charter's own evolution. It declares:
>
> 1. The four qualification criteria for constitutional invariants (already enumerated in [Section 2](../constitutional-charter.md#2-constitutional-invariants)).
> 2. The amendment philosophy: changes to the Charter require formal amendment recorded in [`amendments.md`](../amendments.md), not silent edits.
> 3. The precedence rule: when subordinate documents conflict with the Charter, the Charter prevails. Subordinate documents are revised; the Charter is not.
> 4. The falsifiability discipline: any constitutional claim must be structurally falsifiable. If a property cannot, in principle, be violated, observed, or audited, it is not a constitutional property — it is an aspiration, an aesthetic preference, or a research direction, and belongs elsewhere.
>
> — [Charter §4](../constitutional-charter.md#4-constitutional-design-rule)

---

## Definition

**Bullets mapped here:** 1; 4 (positive face — the four qualification criteria define what counts as a constitutional invariant; bullet 4's affirmative formulation extends bullet 1 by adding falsifiability as a further criterion).

**Source citations from the inventory:**

- Bullet 1 → [`docs/charter/constitutional-charter.md` §2 header (L32–42; FROZEN per amendment v0.1.1)](../constitutional-charter.md#2-constitutional-invariants); echoed in [`falsifiability-check` §1.2 / §1.3](../../../.claude/skills/epistemic/falsifiability-check/SKILL.md).
- Bullet 4 (positive face) → [`falsifiability-check` SKILL.md §1](../../../.claude/skills/epistemic/falsifiability-check/SKILL.md); [`amendments.md` §Amendment Process step 2](../amendments.md); [`CONTRIBUTING.md` §What This Project Is Not](../../../CONTRIBUTING.md).

## Structural Requirement

**Bullets mapped here:** 2; 3 (tentative — the conflict-resolution mechanism: subordinate document is revised, never the Charter; mapping marked `unclear` in the inventory — see Committee notes).

**Source citations from the inventory:**

- Bullet 2 → [`docs/charter/amendments.md` §Amendment Discipline; §Amendment Process](../amendments.md); [`charter-guardian` §2 Step 3](../../../.claude/skills/constitutional/charter-guardian/SKILL.md); [`docs/charter/decision-log.md` §0004](../decision-log.md); [`pre-commit-doc-check.sh` `check_frozen_charter`](../../../.claude/hooks/pre-commit-doc-check.sh).
- Bullet 3 (tentative) → [`.claude/CLAUDE.md` §2 Document hierarchy](../../../.claude/CLAUDE.md); [`README.md` §Document Hierarchy](../../../README.md); [`subordination-checker`](../../../.claude/skills/constitutional/subordination-checker/SKILL.md); [`docs/charter/decision-log.md` §0004](../decision-log.md); [`docs/charter/constitutional-charter.md` opening blockquote (L6)](../constitutional-charter.md).

## Rationale

<!-- pending: no bullet currently maps to this subsection. Open question for committee. -->

## Forbidden Anti-Patterns

**Bullets mapped here:** 4 (negative face — the explicit "aspiration / aesthetic preference / research direction" disqualifier list); 2 (negative echo — "not silent edits").

**Source citations from the inventory:**

- Bullet 4 (negative face) → [`falsifiability-check` §2 Failure modes](../../../.claude/skills/epistemic/falsifiability-check/SKILL.md).
- Bullet 2 (negative echo) → [`charter-guardian` §2](../../../.claude/skills/constitutional/charter-guardian/SKILL.md) (silent-amendment failure mode the skill is designed to catch).

## Boundary Conditions

<!-- pending: no bullet currently maps to this subsection. Open question for committee. -->

---

## Committee notes
