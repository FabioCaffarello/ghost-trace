---
name: invariant-redactor
description: Support committee-mode redaction of pending Charter invariants. Use this skill ALWAYS when work begins on §2.4 Inferential Influence Disclosure, §2.6 Evidential Independence Integrity, or §3 Non-Goals. This skill is the only legitimate path to text that will become a frozen invariant — direct edits to those sections of constitutional-charter.md are blocked by charter-guardian.
---

# invariant-redactor

Pending Charter elements are redacted in committee mode ([`amendments.md` §Amendment Process](../../../../docs/charter/amendments.md)). Redaction proceeds one section at a time, with explicit defense of each word choice. A redacted section is not merged into the Charter until it passes falsifiability review. This skill structures the work.

The currently pending elements are listed in [`.claude/CLAUDE.md` §4](../../../CLAUDE.md): §2.4, §2.6, §3. The qualification criteria a candidate invariant must satisfy are stated in [Charter §2](../../../../docs/charter/constitutional-charter.md#2-constitutional-invariants). Three prior committee-mode redactions have completed and serve as procedural precedent: §4 (Gate 1, [`decision-log §0007`](../../../../docs/charter/decision-log.md)), §2.5 (Gate §2.5, [`§0013`](../../../../docs/charter/decision-log.md)), and §2.3 (Gate §2.3, [`§0017`](../../../../docs/charter/decision-log.md) — second object-level invariant, first with inheritance-dominant non-duplication shape per §0017 Methodological Observation 1).

## 1. Committee-mode discipline

Three rules. Violating any of them disqualifies the candidate.

1. **One section at a time.** A redaction work session focuses on one pending element. Parallel redaction of multiple elements is rejected; the elements interact, and parallel work conceals the interactions.
2. **Explicit defense of each word choice.** Every load-bearing word in the candidate is defended against an alternative — why this term, not another. The defense is recorded with the candidate, not separately.
3. **Falsifiability review precedes merge.** The candidate is run through `falsifiability-check` for every claim it contains. A candidate with any non-falsifiable clause does not merge.

The two existing frozen invariants ([§2.1](../../../../docs/charter/constitutional-charter.md#21-observational-integrity), [§2.2](../../../../docs/charter/constitutional-charter.md#22-epistemic-separation)) are the structural reference for what a finished redaction looks like.

## 2. Redaction procedure

Apply in order. The procedure produces a candidate redaction; it does not merge.

### Step 1 — Anchor in the working-definition stub

The pending element in the Charter already contains a working-definition stub (non-binding). Read it. The candidate either reformulates this stub into binding form or supersedes it; in either case, the stub is the entry point, not an external draft.

### Step 2 — Produce the candidate in a scratch document

The candidate lives at `docs/charter/in-committee/§NN-<short-name>.md` — e.g., `docs/charter/in-committee/§2.3-provenance-integrity.md`. The `in-committee/` directory is created the moment redaction begins, not before; preemptive creation suggests imminent work that has not been scheduled.

The candidate is not edited directly into `constitutional-charter.md`. Direct edits to a pending element of the Charter are blocked by `charter-guardian`.

### Step 3 — Apply `falsifiability-check` to every claim

Run [`falsifiability-check`](../../epistemic/falsifiability-check/SKILL.md) on every clause. A candidate that fails any of the four-question test (violation, observation, operationalization, non-circularity) in any clause is rewritten, or the failing clause is removed.

### Step 4 — Apply `epistemic-separator` to every paragraph

Run [`epistemic-separator`](../../epistemic/epistemic-separator/SKILL.md) on every paragraph. A candidate that conflates Category I / II / III anywhere is rewritten.

### Step 5 — Apply `ambiguity-reducer` to every noun

Run [`ambiguity-reducer`](../../epistemic/ambiguity-reducer/SKILL.md) on every noun. Any flagged term must be replaced with a canonical term, operationalized locally in the candidate, or raised as an open modeling question.

### Step 6 — Identify forbidden anti-patterns

Each frozen invariant carries a `Forbidden Anti-Patterns` section that names, by example, the failures it prevents. The candidate adds the same section, identifying the anti-patterns the new invariant rejects. Each anti-pattern is concrete and falsifiable; if an anti-pattern in the list is itself non-falsifiable, it is rewritten or removed.

### Step 7 — Identify boundary conditions

Each frozen invariant carries a `Boundary Conditions` section that names what the invariant does *not* govern. The candidate adds this section. Boundary conditions defend against over-application: an invariant that claims to govern everything cannot be specifically enforced.

### Step 8 — Identify non-duplication

Before merge, the candidate explicitly identifies which existing invariants it does *not* duplicate. If the candidate is redundant with an existing invariant, redaction stops and the candidate is withdrawn — the constitutional-minimalism rule ([`.claude/CLAUDE.md` §7](../../../CLAUDE.md)) rejects redundant invariants.

Non-duplication is not binary. Per [`decision-log §0017`](../../../../docs/charter/decision-log.md) Methodological Observation 1 (Gate §2.3 closure), Step 8 verdicts classify each element as **Adds** (introduces new constitutional commitment), **Anchors** (is source-of-record for what subordinate documents defer to it), **Both** (does both), or **Inherits** (references a frozen mechanism's enforcement without originating or anchoring). Inheritance is not duplication — a candidate that inherits substantially from prior frozen Charter sections (e.g., §2.3 inheriting from §2.1 + §2.5 + Ontology Q1/Q3 resolutions) still warrants Charter-level codification when the inheritance chain has no constitutional anchor connecting its priors into a single invariant. Later §2.x invariants (§2.4, §2.6) are expected to exhibit inheritance-dominant non-duplication shapes as priors accumulate; the verdict-mix is a function of priors, not procedure.

### Step 9 — Surface for human review

The completed candidate is surfaced to the human for review. The candidate is not merged automatically. Committee approval is human work; this skill structures the input.

## 3. Final merge checklist

A candidate merges into `constitutional-charter.md` only when all five items hold. Each is checked at the moment of merge, not at draft.

1. **Four qualification criteria met.** Per [Charter §2](../../../../docs/charter/constitutional-charter.md#2-constitutional-invariants): structurally enforceable; constraining of future implementation decisions; identity-defining; independent of operator interpretation.
2. **Section structure parallels §2.1 and §2.2.** The candidate has `Definition`, `Structural Requirement`, `Rationale`, `Forbidden Anti-Patterns`, and `Boundary Conditions` sections.
    - **Heading-depth transposition.** Scratch documents in `docs/charter/in-committee/` use H3 (`###`) for these five sub-subsections (parallel to §2.1/§2.2 read standalone). At Charter transposition into `constitutional-charter.md`, demote each H3 to H4 (`####`) — the §2.x sub-subsection convention places `## 2. Constitutional Invariants` at H2, `### 2.X Title` at H3, and the five sub-subsections at H4. The transposition is mechanical but easy to skip if not made explicit; surfaced by the charter-reviewer subagent R1 verdict at §2.4 v0.5 closure ([`decision-log.md` §0099](../../../../docs/charter/decision-log.md) CF1 + [`decision-log.md` §0102](../../../../docs/charter/decision-log.md) closure record).
3. **Vocabulary in the glossary.** Every load-bearing term in the candidate has an entry in [`docs/glossary.md`](../../../../docs/glossary.md) with all five provenance fields filled (or marked `pending` where appropriate). Terms introduced by this invariant are added to the glossary in the same change set.
4. **Decision-log entry.** A new entry is added to [`docs/charter/decision-log.md`](../../../../docs/charter/decision-log.md) recording the redaction — its context, the constitutional review outcome, and the consequences for subordinate documents.
5. **Amendments-log entry and version bump.** A new entry is added to [`docs/charter/amendments.md`](../../../../docs/charter/amendments.md) following the existing template, with the Charter version bumped per the rule in `amendments.md` — patch for editorial; minor for substantive non-identity; major for production-ready declaration.

A merge that skips any of these items is procedurally invalid even if the candidate text is acceptable. The procedure is part of what the invariant means.

## 4. What this skill does not do

This skill does not write invariant content for the human. It structures the work of writing, applies the discipline of the project, and identifies what the candidate is missing. The content itself is committee work.

This skill does not approve a candidate. Approval is human committee work, formally recorded.

## 5. Source citations used

- [`docs/charter/constitutional-charter.md` §2 Constitutional Invariants (qualification criteria); §2.1, §2.2 (structural reference for finished redaction)](../../../../docs/charter/constitutional-charter.md)
- [`docs/charter/amendments.md` §Amendment Process](../../../../docs/charter/amendments.md)
- [`docs/charter/decision-log.md`](../../../../docs/charter/decision-log.md)
- [`docs/glossary.md`](../../../../docs/glossary.md)
- [`.claude/CLAUDE.md` §4 Charter status at a glance; §7 Constitutional minimalism](../../../CLAUDE.md)
- [`.claude/skills/epistemic/falsifiability-check/SKILL.md`](../../epistemic/falsifiability-check/SKILL.md)
- [`.claude/skills/epistemic/epistemic-separator/SKILL.md`](../../epistemic/epistemic-separator/SKILL.md)
- [`.claude/skills/epistemic/ambiguity-reducer/SKILL.md`](../../epistemic/ambiguity-reducer/SKILL.md)
