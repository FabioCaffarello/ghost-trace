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

### Step 1.2 — Falsifiability findings (per bullet × test)

Applied [`falsifiability-check`](../../../.claude/skills/epistemic/falsifiability-check/SKILL.md) §1 (violation / observation / operationalization / non-circularity) to each bullet. Verdicts: Pass / Pass with caveat / Fail. Findings record what the tests surfaced, not proposed rewrites.

#### Bullet 1 — qualification criteria (reference to §2)

- **1.1 Violation — Pass with caveat.** Falsifying state: §2 enumerates a number ≠ 4, or its criteria differ from what §4 implies. Detectable by counting numbered items at `constitutional-charter.md` L36–L39. Caveat: violation is structural (Charter-internal desync), not substrate-level — categorically different from §2.x violations.
- **1.2 Observation — Pass.** Third party counts §2 items mechanically; no cooperation needed.
- **1.3 Operationalization — Pass with caveat.** "Four" → integer; "qualification criteria" → the §2 L36–L39 items; "constitutional invariants" → the §2.x frozen sections. Caveat: "constitutional invariants" is not formally in [`CLAUDE.md` §3 canonical vocabulary](../../../.claude/CLAUDE.md) as a glossary term.
- **1.4 Non-circularity — Pass.** Bullet references; does not define.

**Overall:** Pass with caveat.

#### Bullet 2 — amendment philosophy

- **1.1 Violation — Pass.** Falsifying state: a commit diffs the Charter; no entry in `amendments.md` records that change. Detectable by cross-comparing the two files' edit histories.
- **1.2 Observation — Pass.** Edit history of both files is the substrate; mechanical detection. The hook's `check_frozen_charter` already mechanizes this for FROZEN ranges.
- **1.3 Operationalization — Pass.** "Changes to the Charter" → diff hunks; "formal amendment" → `amendments.md` entry per template; "silent edits" → Charter edit without matching entry.
- **1.4 Non-circularity — Pass.** "Amendment" is defined in [`CLAUDE.md` §3 canonical vocabulary](../../../.claude/CLAUDE.md).

**Overall:** Pass. Calibration baseline — the cleanest of the four bullets.

#### Bullet 3 — precedence rule

- **1.1 Violation — Pass with caveat.** Falsifying state: a documented conflict between subordinate doc Y and Charter §X resolved by editing §X to match Y, or persisting unresolved. Caveat: presupposes "conflict" is already determined; observability of the falsifying state hinges on prior detection.
- **1.2 Observation — Fail.** Detection of "conflict" between two prose documents requires subjective judgment about semantic compatibility. [`subordination-checker`](../../../.claude/skills/constitutional/subordination-checker/SKILL.md) *surfaces* contradictions (per [`SELF-AUDIT.md` §3](../../../.claude/SELF-AUDIT.md)) — surfacing is a reviewer action, not mechanical detection. The CI `subordination-check` job validates Charter anchors mechanically but cannot detect semantic conflict. Matches the failure mode in [`falsifiability-check` §1.2](../../../.claude/skills/epistemic/falsifiability-check/SKILL.md): detection that requires subjective judgment is not structurally falsifiable.
- **1.3 Operationalization — Fail.** "Subordinate documents" and the resolution direction ("Charter prevails / subordinate is revised") are operationalizable. "Conflict" is not — no definition in Charter, glossary, or skills. The bullet uses it as if self-defining. Same shape as the failure mode in [`falsifiability-check` §2 "Non-operationalizable"](../../../.claude/skills/epistemic/falsifiability-check/SKILL.md).
- **1.4 Non-circularity — Pass with caveat.** "Subordinate" is defined in [`CLAUDE.md` §3](../../../.claude/CLAUDE.md) as "lower-ranked documents must not conflict with higher-ranked documents". Bullet 3's "if conflict" is the negation of the definition; the non-tautological content is the resolution direction.

**Overall:** Fail.

#### Bullet 4 — falsifiability discipline

- **1.1 Violation — Pass with caveat.** Falsifying state: a Charter clause fails one or more tests when `falsifiability-check` §1 is applied. Caveat: self-referential — applying bullet 4 to bullet 4 is a recursion. Under a fixed-point reading (bullet 4 satisfies its own test procedurally), the recursion terminates with Pass. Adopting the fixed-point reading is a committee question; the test itself does not pick it.
- **1.2 Observation — Pass with caveat.** Procedurally defined; applying the test does not require subjective judgment. Caveat: same recursion as 1.1.
- **1.3 Operationalization — Pass.** "Constitutional claim" → Charter sentence; "structurally falsifiable" → passes four-question test; "violated / observed / audited" → three concrete procedures; "aspiration / aesthetic preference / research direction" → three demotion destinations in [`falsifiability-check` §3](../../../.claude/skills/epistemic/falsifiability-check/SKILL.md).
- **1.4 Non-circularity — Pass with caveat.** [`CLAUDE.md` §3](../../../.claude/CLAUDE.md) "falsifiability" entry defers to §4 itself for definition. Apparent circularity resolves because the procedural definition lives in `falsifiability-check` §1's four-question test, not in bullet 4. The chain bottoms out in a procedure.

**Overall:** Pass with caveat.

#### Per-bullet summary

| Bullet | Verdict |
|---|---|
| 1 — qualification criteria | Pass with caveat |
| 2 — amendment philosophy | **Pass** |
| 3 — precedence rule | **Fail** |
| 4 — falsifiability discipline | Pass with caveat |

#### What the test surfaced that the Step 1.1 inventory did not

- **Bullet 1.** Inventory: forward-pointer to §2. Test: violation state is *meta* falsifiability (Charter-structural desync), categorically distinct from §2.x's *substrate* falsifiability. Whether meta-falsifiability is sufficient for a §4 claim is a question §4 itself is supposed to answer; bullet 4 does not explicitly address it.
- **Bullet 2.** Inventory: maps cleanly to Structural Requirement. Test: confirms — passes all four tests without caveat. Calibration baseline.
- **Bullet 3.** Inventory: `unclear` mapping to Structural Requirement or Boundary Conditions. Test: the unclear mapping is downstream of an unoperationalized operative term — "conflict". The template did not fail the bullet; the bullet failed itself.
- **Bullet 4.** Inventory: double-maps Definition + Forbidden Anti-Patterns. Test: self-reference is the deeper issue. Any binding redaction must explicitly handle the recursion. The double-map is a symptom; the recursion is a separate, structural question.

**Open for committee decision:**

The tests surfaced four distinct questions the committee faces, one per bullet. Bullet 1 (Pass with caveat) is a forward-pointer, not a substantive claim; the committee can keep it as a reference, restate the four criteria in §4 to make §4 self-contained, or remove it if §2 is judged sufficient on its own. The caveat about "constitutional invariants" missing from [`CLAUDE.md` §3 canonical vocabulary](../../../.claude/CLAUDE.md) may also warrant a glossary addition independent of §4's resolution. Bullet 2 (Pass) does not require a committee decision per these tests; it is the calibration baseline and may be carried into binding text as-is or with editorial improvement. Bullet 3 (Fail) failed two tests because "conflict" is not operationalized; the committee can operationalize "conflict" in the binding §4 text, demote bullet 3 to [`CONTRIBUTING.md`](../../../CONTRIBUTING.md) or [`WORKFLOW.md`](../../../WORKFLOW.md) as procedural discipline rather than a constitutional claim, or remove bullet 3 if `subordination-checker`'s surfacing capability and the document-hierarchy ranks are judged to carry the precedence rule implicitly. Bullet 4 (Pass with caveat) is self-referential; the committee can explicitly state §4's fixed-point reading (§4 satisfies its own test by procedurally specifying what passing means), exempt §4 from the falsifiability discipline (the discipline applies to claims about the system, not about the Charter's own structure), or leave the recursion implicit. Each choice has different downstream consequences for how future invariants are reviewed.

The committee is also free to combine — e.g., keep bullet 1 as a reference, accept bullet 2 unchanged, demote bullet 3, and add a fixed-point statement for bullet 4. Declaring the §2.x template inadequate for §4 entirely, and adopting a different structural template, would also be a methodological output of this pilot.

### Step 1.3 — Epistemic-skill findings

Step 1.3 applied Phase 1 of the prompt (per-bullet applicability determination for [`epistemic-separator`](../../../.claude/skills/epistemic/epistemic-separator/SKILL.md) and [`ambiguity-reducer`](../../../.claude/skills/epistemic/ambiguity-reducer/SKILL.md)). Phase 2 (mechanical application of each skill where applicable) was skipped on committee decision: the applicability matrix produced zero applicable cells for `epistemic-separator` and zero hits for `ambiguity-reducer` across all four bullets, so Phase 2 would have produced no new content; the skip is recorded as a legitimate null outcome of the step rather than as abbreviation.

#### Applicability matrix

| Bullet | `epistemic-separator` | `ambiguity-reducer` |
|---|---|---|
| 1 — qualification criteria | Inapplicable by domain. Bullet names the meta-criteria for what counts as a constitutional invariant; does not touch substrate, projections, or the three knowledge categories; none of the eight triggering words from the skill's description appear. | Applied per-term, no findings. None of the 12 watchlist terms ([`ambiguity-reducer` §1](../../../.claude/skills/epistemic/ambiguity-reducer/SKILL.md)) appear in the bullet. |
| 2 — amendment philosophy | Inapplicable by domain. Bullet is about Charter-governance procedure; "recorded" refers to writing an entry to a governance document, not to committing a Category I observation. | Applied per-term, no findings. No watchlist term appears. |
| 3 — precedence rule | Inapplicable by domain. Bullet governs the document-rank relationship; does not touch substrate, projections, or categories. | Applied per-term, no findings on the current watchlist. The unoperationalized operative term `conflict` surfaced in Step 1.2 above is **not** on the watchlist — see open question below. |
| 4 — falsifiability discipline | Inapplicable by domain. Bullet is about meta-criteria for Charter claims; none of the eight triggering words appear. Methodological note on Bullet 4: the verb "observed" in the bullet appears in its falsifiability-detection sense, not as a noun naming Category I observation; a reader who carries the substrate sense into the meta sense would be misframing the prose surface, not the skill's judgment. Surfaced here for record. | Applied per-term, no findings. No watchlist term appears in its noun sense. |

#### Methodological observation

The complete absence of hits across both skills (8/8 cells producing no actionable finding) is consistent with §4's meta-character established in Step 1.2 above. Both skills are authored to discipline object-level prose: `epistemic-separator` polices conflation across the three substrate categories; `ambiguity-reducer`'s 12 watchlist terms are all substrate-adjacent vocabulary. §4's vocabulary is governance-domain (`amendment`, `subordinate`, `precedence`, `falsifiability`, `constitutional`). The two domains do not overlap by construction. This itself is a finding about the §2.x template's fit for §4: §2.1 and §2.2 are disciplined by the joint application of `falsifiability-check` + `epistemic-separator` + `ambiguity-reducer`; for §4, only `falsifiability-check` reaches the prose.

**What Step 1.3 adds to Step 1.2:**

Step 1.3 contributes independent confirmation of §4's meta-character, by way of two skills whose applicability domains do not reach the bullets. Step 1.2 inferred meta-character from `falsifiability-check`'s self-application (the recursion in Bullet 4, the cross-reference desync in Bullet 1's violation state). Step 1.3 confirms by the inapplicability of the two adjacent epistemic skills, rather than by their application. Step 1.3 also contributes non-confirmation of an alternative hypothesis — that §4 might be testing the skills' edges rather than sitting outside their domain. The 8-cell inapplicability is structural, not edge-case: §4 is in a different domain, not on the boundary of the skills' domain. The distinction matters because it preserves two open committee choices, neither of which Step 1.3 resolves: (a) §4 is structurally different and may warrant a meta-prose-specific discipline analogous to the three epistemic skills but for governance prose; or (b) the existing skills are correctly scoped (object-level) and meta-prose is handled by other means (`charter-guardian` + amendment process + decision log). Step 1.3 establishes that (a) and (b) are real alternatives; it does not pick.

**Open for committee decision:**

Three items, in order of how they affect downstream redactions:

- **Watchlist-extension candidate `conflict`.** Step 1.2 above surfaced `conflict` as bullet 3's unoperationalized operative term. Step 1.3 propagates this from a Step 1.2 finding to a Step 1.3 active item: `conflict` is a watchlist-extension candidate per [`ambiguity-reducer` §3](../../../.claude/skills/epistemic/ambiguity-reducer/SKILL.md). Whether to add it is committee work; this step does not extend the watchlist.
- **(a) vs. (b) — meta-prose discipline.** The choice between "§4 warrants a meta-prose-specific discipline" and "the existing skills are correctly object-level scoped; meta-prose is handled elsewhere" is operative for §2.3–§2.6 too, since those four pending invariants are object-level and will trigger the three epistemic skills strongly. If the committee picks (a), §4 introduces an exception that must be named in §4 itself. If (b), §4 sits outside the skills' purview without contradiction, and the silence of `epistemic-separator` / `ambiguity-reducer` on §4 is by design.
- **Adjacent finding (not a §4 item).** Step 1.2 above noted "constitutional invariants" is not in [`CLAUDE.md` §3 canonical vocabulary](../../../.claude/CLAUDE.md). This is a `vocabulary-discipline` concern, not an `ambiguity-reducer` one; it is mentioned here for record but tracked as a backlog item independent of §4's redaction (candidate for a future mini-RFC), not absorbed as a §4 redaction question.
