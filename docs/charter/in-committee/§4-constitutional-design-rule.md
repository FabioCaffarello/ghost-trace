# §4 Constitutional Design Rule — committee draft

**Status:** merged. Binding form is now Charter §4 v0.2. This scratch is preserved as historical record of the committee-mode pilot. See [decision-log entry §0007](../decision-log.md).

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

### Step 1.4a — Forbidden Anti-Patterns analysis

Per the redactor's Step 6 (`Forbidden Anti-Patterns` identification). Anti-patterns must be concrete, falsifiable, and drawn from real failure modes — not speculative.

#### Bullet 1 — qualification criteria

Bullet 1 is a forward-pointer to [Charter §2 header (L32–42, FROZEN)](../constitutional-charter.md#2-constitutional-invariants), which enumerates the four criteria but has no `Forbidden Anti-Patterns` subsection of its own. The anti-patterns of *failing* the criteria therefore have no Charter-level home today; if Bullet 1 is retained in binding §4 (see Step 1.4c verdict), these anti-patterns become its substantive content.

1. **Adopting an invariant whose violation requires subjective judgment to detect** — fails criterion 4 ("independent of operator interpretation"). Caught by [`falsifiability-check` §1.2](../../../.claude/skills/epistemic/falsifiability-check/SKILL.md). **Enforcement: Yes** (mechanical via the four-question test).
2. **Adopting an invariant not structurally enforceable in schemas, types, or permitted operations** — fails criterion 1. Caught by [`falsifiability-check` §1.3](../../../.claude/skills/epistemic/falsifiability-check/SKILL.md) (operationalization test). **Enforcement: Yes**.
3. **Adopting an invariant that does not constrain future implementation decisions** — fails criterion 2; the invariant is decorative. Checked procedurally by [`invariant-redactor` §3 Final merge checklist item 1](../../../.claude/skills/constitutional/invariant-redactor/SKILL.md). **Enforcement: Partial** (no mechanical per-claim check).
4. **Adopting an invariant whose absence would not change what the system is** — fails criterion 3 ("identity-defining"). Same redactor merge checklist; the criterion is itself a judgment call. **Enforcement: No** (no skill operationalizes "identity-defining").

#### Bullet 2 — amendment philosophy

1. **Silent edits to FROZEN Charter sections.** Editing `constitutional-charter.md` without a corresponding `amendments.md` entry. Caught by [`charter-guardian` §3](../../../.claude/skills/constitutional/charter-guardian/SKILL.md) (explicitly names this as "the most common silent-amendment failure mode") and mechanically by [`pre-commit-doc-check.sh` `check_frozen_charter`](../../../.claude/hooks/pre-commit-doc-check.sh). **Enforcement: Yes** (mechanically blocking).
2. **Amendment adopted without falsifiability review.** [`amendments.md` §Amendment Process Step 2](../amendments.md) mandates it; `charter-guardian` §2 Step 3 lists it as a required artifact. **Enforcement: Yes** (procedurally required; no check that the review *was performed*, only that artifacts exist).
3. **Editorial change treated as substantive (or vice versa).** Using "editorial fix" as cover for meaning change, or burdening typo fixes with full ceremony. [`charter-guardian` §2 Step 3](../../../.claude/skills/constitutional/charter-guardian/SKILL.md); [`amendments.md` §Amendment Discipline](../amendments.md). **Enforcement: No** (relies on meaning judgment).
4. **Charter edited to match a subordinate document on conflict** — reverse-direction failure of Bullet 3. [`subordination-checker` §2 Step 4](../../../.claude/skills/constitutional/subordination-checker/SKILL.md). **Enforcement: Partial**.

#### Bullet 3 — precedence rule

1. **Lower-rank document edits a higher-rank document on conflict.** [`subordination-checker` §4](../../../.claude/skills/constitutional/subordination-checker/SKILL.md). **Enforcement: Partial** — FROZEN-section edits are blocked regardless of motivation; pending and subordinate-direction-reversal not mechanically blocked.
2. **Conflicting claim persists unresolved between subordinate and Charter.** The Step 1.2 finding: requires subjective conflict detection. **Enforcement: No**.
3. **Cross-reference to a deleted or moved Charter section.** [`.github/workflows/constitutional-check.yml` `subordination-check` job](../../../.github/workflows/constitutional-check.yml) validates anchors mechanically. **Enforcement: Yes** (CI-blocking).

#### Bullet 4 — falsifiability discipline

The skill [`falsifiability-check` §2](../../../.claude/skills/epistemic/falsifiability-check/SKILL.md) already enumerates four failure modes with examples (non-violable, non-observable, non-operationalizable, circular). Restating them in §4 binding text would be literal duplication. **Bullet 4 does not need its own anti-patterns subsection; it delegates to the skill.** Enforcement of the four failure modes: Yes, mechanically via the four-question test at every committee redaction and every RFC Constitutional Review.

#### Enforcement-coverage summary

| Verdict | Count |
|---|---|
| Yes (mechanical / required-by-procedure) | 6 |
| Partial | 3 |
| No | 2 |

Of the 10 anti-patterns across Bullets 1–3, 6 are already mechanically enforced. The Bullet 4 cluster (4 skill-level failure modes) is fully covered by `falsifiability-check`. Two anti-patterns are unenforced: Bullet 1 criterion 3 (identity-defining), Bullet 2 anti-pattern 3 (editorial vs substantive); plus Bullet 3 anti-pattern 2 (conflict-detection, already failed Step 1.2 falsifiability).

### Step 1.4b — Boundary Conditions analysis

Per the redactor's Step 7. The §2.x substrate-vs-projection pattern does not translate to meta-prose; §4's boundaries are governance-domain.

#### Candidate 1 — Internal project practice outside Charter governance

Commit conventions, branch naming, code style, README phrasing: not Charter content. **Applies.** Already implicitly drawn by [`CLAUDE.md` §2](../../../.claude/CLAUDE.md) (operational hierarchy) and the Charter's opening blockquote (L6); not stated as a Charter §-boundary today.

#### Candidate 2 — The content of invariants, only their form

§4 disciplines what *qualifies* as an invariant; it does not dictate *which* invariants the project adopts. **Applies.** Implicit in the §2-header / §2.x split; not stated explicitly anywhere.

#### Candidate 3 — Infrastructure supporting governance but not the Charter

Skills, hooks, CI, agents, settings. Modifiable without Charter amendment. **Applies.** Concrete evidence: Gates 0a, 0b, hook-fix all modified `pre-commit-doc-check.sh` without Charter amendment. Partially stated in [`.claude/README.md`](../../../.claude/README.md); not at Charter level.

#### Additional boundaries surfaced

None at the level of structural relevance. Two near-candidates rejected: "§4 does not govern pending-section redaction pace" (too obvious; not a real boundary) and "§4 does not govern external documents" (outside the project entirely). Honest finding: the three candidates are the natural boundaries.

#### Overall verdict on Boundary Conditions

Two options for the committee at Step 9:

- **Option α — include Boundary Conditions in binding §4.** All three boundaries are real, none currently stated at Charter level. Cost: ~3 short paragraphs. Path of self-contained reading.
- **Option β — omit Boundary Conditions.** Meta-prose has self-evident boundaries: §4 governs the Charter; what is not the Charter is outside §4. Path of constitutional minimalism ([`CLAUDE.md` §7](../../../.claude/CLAUDE.md)). The §2.x Boundary Conditions pattern does not translate to meta, and declaring this is itself a methodological finding.

Both defensible. Not a Step 1.4 decision.

### Step 1.4c — Non-duplication check (Step 8)

Per the redactor's Step 8. Both readings applied: strong (does §4 add new enforcement?) and weak (does the infrastructure cite §4 as anchor?).

**Citation evidence summary.** Most §4 references across infrastructure point to *other things* (CLAUDE.md §4 status table; or "this-skill's-own-§4"). Four structural places cite **Charter §4** specifically as authority, all concerning the falsifiability discipline:

1. [`falsifiability-check` SKILL.md L8](../../../.claude/skills/epistemic/falsifiability-check/SKILL.md) — quotes §4 working text verbatim.
2. [`falsifiability-check` SKILL.md L118](../../../.claude/skills/epistemic/falsifiability-check/SKILL.md) — Source citations.
3. [`CLAUDE.md` L45](../../../.claude/CLAUDE.md) — canonical-vocabulary entry for `falsifiability` defers to §4 pending.
4. [`amendments.md` L22](../amendments.md) — §Amendment Process Step 2 cites "Section 4 of the Charter" as authority for falsifiability review.

#### 8-cell analysis

| Bullet | Strong reading (does §4 add new enforcement?) | Anchor reading (is §4 cited as authority?) |
|---|---|---|
| 1 — qualification criteria | **No new enforcement.** Criteria are in [§2 header](../constitutional-charter.md#2-constitutional-invariants). Failure-mode detection: 3 of 4 criteria caught by `falsifiability-check`; the fourth (identity-defining) has only the redactor procedure. | **Substantive duplication; but see Additional citations below — §2 L41 (FROZEN) cites §4 as the formal locus where the criteria "are themselves recorded."** Initial assessment of "no anchor role" is revised by the §2 L41 finding below. |
| 2 — amendment philosophy | **No new enforcement.** `check_frozen_charter` + `charter-guardian` + `amendments.md` together fully cover the discipline. Step 1.2 found Bullet 2 passes all four falsifiability tests cleanly — as a restatement of infrastructure. | **No direct anchor role.** `charter-guardian` cites `amendments.md` and CLAUDE.md §4 (status table), not Charter §4. `amendments.md` self-justifies in its opening; if §4 had Bullet 2 verbatim, `amendments.md` would point to it, but the citation does not currently exist. (See structural note below on `amendments.md`'s implicit dependency on Bullet 2's principle.) |
| 3 — precedence rule | **No new enforcement.** Hierarchy declared in [`CLAUDE.md` §2](../../../.claude/CLAUDE.md) and [`README.md` §Document Hierarchy](../../../README.md); direction-reversal policed by `subordination-checker`; anchor validity by CI. Step 1.2 *failed* Bullet 3 on observation and operationalization. | **No direct anchor role.** `subordination-checker` cites CLAUDE.md §2 (hierarchy) and Charter §2.1/§2.2 (anti-patterns), not §4. [`decision-log.md` §0004](../decision-log.md) is the source-of-record for the precedence rule; §4 only "pre-figures" it. |
| 4 — falsifiability discipline | **No new enforcement.** The four-question test is in [`falsifiability-check` §1](../../../.claude/skills/epistemic/falsifiability-check/SKILL.md); the procedural mandate is in [`amendments.md` Step 2](../amendments.md); restated in [`CONTRIBUTING.md`](../../../CONTRIBUTING.md). | **Strong anchor role.** Bullet 4 is the source-of-record cited in five structural places (four above + glossary `falsifiability` Introduction). Removing it would orphan all five. |

**Additional citations surfaced after CHECKPOINT 1.4.C:**

A second-pass grep with wider criteria (`charter.*#4`, `Charter Section 4`, `#4-constitutional-design-rule`) surfaced three citations the initial analysis missed. The first is the most consequent because it is in FROZEN text:

- **[Charter §2 L41 (FROZEN per v0.1.1)](../constitutional-charter.md#2-constitutional-invariants):** *"The criteria above are themselves recorded formally in [Section 4 — Constitutional Design Rule](#4-constitutional-design-rule) (pending). They are applied as meta-rule to all invariants in this section."* This is the strongest citation in the repository: §2 (frozen) commits §4 to be the formal locus of the criteria. If Bullet 1 is removed, §2's L41 claim becomes unfulfilled — §2 references a §4 that does not deliver. Editing §2 L41 to remove the reference would itself require amendment (§2 is FROZEN per v0.1.1). The amendment would be "remove §2's reference to §4 because §4 was emptied of the criteria reference" — circular.
- **[Glossary `falsifiability` entry L146](../../glossary.md):** *"Introduction. [Charter §4 Constitutional Design Rule (pending working text)]; CONTRIBUTING.md §Style."* Fifth anchor citation for Bullet 4; reinforces Bullet 4's verdict.
- **[Glossary `subordination` entry L140](../../glossary.md):** *"Last amendment. pending (full formalization is in pending §4 Constitutional Design Rule)."* Forward-looking — glossary expects §4 to formalize subordination. If Bullet 3 does not survive in binding §4, this entry needs an editorial fix (not citation orphanage; the entry stays valid, but the prediction becomes false).

**Structural note on Bullet 2 (amendments.md dependency).** `amendments.md` opens with *"Changes to it [the Charter] are not made through ordinary commits"* and derives the entire Amendment Process from that principle. This is the substance of Bullet 2 in operational form. The dependency is not a direct citation (`amendments.md` does not say "per Charter §4 Bullet 2"), but it is structural: if §4 had Bullet 2 verbatim, the file would have an explicit anchor; without §4, `amendments.md` self-justifies by convention. Less fatal than the §2 L41 citation (no FROZEN claim is unfulfilled), but worth recording.

#### Per-bullet verdicts (adjusted after Additional citations)

- **Bullet 1 — Substantively duplicates §2, but §2 L41 (FROZEN) cites §4 as the locus where the criteria "are recorded formally." Bullet 1 fulfills this promise. Removing Bullet 1 leaves §2's L41 claim unfulfilled, creating a constitutional gap unresolvable without amending §2 (which is frozen).**
- **Bullet 2 — Duplicates substantively, no direct anchor citation.** Note: `amendments.md` is the operational expression of Bullet 2's principle; the relationship is structural, not citational.
- **Bullet 3 — Duplicates substantively, no anchor.** Failed Step 1.2 falsifiability. Glossary `subordination` entry L140 expects §4 to formalize this; if Bullet 3 does not survive, that glossary entry needs editorial fix.
- **Bullet 4 — Both: substantively duplicates skill content, AND is the load-bearing anchor across five structural citations.** Removing Bullet 4 would orphan all five.

#### Overall verdict for §4 as a section

Two bullets are load-bearing — Bullet 1 (fulfills §2 L41's frozen promise) and Bullet 4 (anchors five citations). Two bullets are not — Bullet 2 (substantive duplicate, no direct citation) and Bullet 3 (substantive duplicate, failed Step 1.2 falsifiability, no anchor). The evidence points to **path (c) — partial redaction**, specifically a **narrow-but-not-minimal** form in which Bullets 1 and 4 survive and Bullets 2 and 3 are removed. The narrow-minimal form (Bullet 4 only) was the pre-additional-citations recommendation; the §2 L41 finding revises it.

### Step 1.4 — Synthesis for Step 9 (Surface for review)

#### Where §4 stands after Steps 1.1–1.4

After four redactor steps, the four §4 bullets stand as follows. Bullet 1 (qualification criteria) is substantively a forward-pointer to §2 — it adds nothing on its own — but [§2 L41 (FROZEN)](../constitutional-charter.md#2-constitutional-invariants) cites §4 as the locus where the criteria "are themselves recorded formally," making Bullet 1 the fulfillment of a frozen promise rather than redundant text. Bullet 2 (amendment philosophy) passes all four falsifiability tests cleanly but adds no new enforcement and is not cited as anchor by any infrastructure; the amendment process operationalized in `amendments.md` derives from the principle without citing it. Bullet 3 (precedence rule) failed Step 1.2 falsifiability on observation and operationalization because "conflict" is undefined; it adds no enforcement and is not cited as anchor by `subordination-checker` (which cites CLAUDE.md §2 and decision-log §0004 instead). Bullet 4 (falsifiability discipline) passes falsifiability with the self-reference caveat and is the load-bearing anchor for the discipline across five structural citations (`falsifiability-check` L8 + L118, CLAUDE.md L45, `amendments.md` L22, glossary `falsifiability` L146).

#### Candidate paths

- **(a) Full successful redaction** — bind all four bullets as §4 text. Requires operationalizing "conflict" in Bullet 3 (Step 1.2 fail) and addressing Bullet 4's self-reference. Evidence support: weak — Bullet 3 has an unresolved falsifiability failure that the committee would need to repair.
- **(b) Complete removal** — emptied §4. Evidence support: not supported — would orphan five Bullet 4 citations and leave §2 L41's frozen promise unfulfilled.
- **(c) Partial redaction** — some bullets survive, others do not. Evidence support: strong, in the **narrow-but-not-minimal form** (Bullets 1 and 4 survive; Bullets 2 and 3 are removed). The narrow-minimal form (Bullet 4 only) was the pre-Additional-citations recommendation; it is contraindicated by the §2 L41 finding.
- **(d) Method inadequate** — committee-mode redaction declared an unfit instrument for §4. Evidence support: weak — Steps 1.1–1.4 produced usable evidence and surfaced the §2 L41 dependency without method failure.

#### Open questions for Step 9

1. **Bullet 1 binding form.** If retained to fulfill §2 L41's promise, does its binding text restate the four criteria verbatim from §2 (creating intentional duplication for anchor purposes), or reformulate them at a higher level of abstraction (preserving the spirit, risking drift)? The §2 L41 phrase "recorded formally" admits both readings.
2. **Bullet 4 binding form.** How is the self-reference (Step 1.2 caveat) handled? Three options: explicit fixed-point statement, explicit exception for §4 from its own discipline, or implicit/silent. Each has different downstream consequences for future invariant review.
3. **Anti-patterns under Bullet 1.** Step 1.4a found four anti-patterns for Bullet 1 (one per criterion); three are mechanically enforced by `falsifiability-check`, one (identity-defining) is unenforced. Does binding §4 include these as `Forbidden Anti-Patterns` content under Bullet 1, or rely on the skill?
4. **Boundary Conditions.** Option α (include explicitly) vs Option β (omit on minimalism grounds). Not constrained by other evidence; pure committee choice.
5. **Glossary `subordination` entry editorial fix.** If Bullet 3 is removed, glossary L140's "Last amendment. pending (full formalization is in pending §4 Constitutional Design Rule)" becomes a false prediction. Editorial fix required (not citation orphanage; entry stays valid).
6. **Amendments.md dependency.** Does the implicit Bullet-2-as-principle dependency in `amendments.md`'s opening warrant any clarifying edit, or remain operational-by-convention?

#### Redactor's recommendation

**Path (c), narrow-but-not-minimal form: §4 binding text retains Bullets 1 and 4 only — Bullet 1 because §2 L41 cites §4 as the formal locus of the four criteria (removing Bullet 1 would orphan a frozen citation), and Bullet 4 because it is the load-bearing anchor for the falsifiability discipline (five structural citations). Bullets 2 and 3 are removed as substantive duplicates without anchor obligations.**

### Step 1.5 — Candidate binding §4 text

Per Phase A resolutions of the six open questions (Q1 hybrid verbatim+role, Q2 fixed-point in Rationale, Q3 include four anti-patterns, Q4 α three boundaries, Q5 glossary subordination editorial fix, Q6 no amendments.md edit), the following candidate replaces the current §4 stub body in [`constitutional-charter.md`](../constitutional-charter.md#4-constitutional-design-rule) at amendment v0.2. The `## 4. Constitutional Design Rule` heading and its anchor `#4-constitutional-design-rule` are preserved unchanged; only the section body is replaced.

---

**[BEGIN CANDIDATE — text below this divider is the proposed §4 binding text. Subsection headings shown here at `####` for scratch hygiene; in `constitutional-charter.md` they become `###` (the level used inside §2.1 / §2.2).]**

#### Definition

This section governs two disciplines applied to every candidate constitutional invariant of Ghost Trace.

**Qualification.** A claim qualifies as a constitutional invariant if and only if it satisfies the four criteria stated in [Section 2 — Constitutional Invariants](#2-constitutional-invariants), reproduced here for anchor purposes (canonical statement remains in §2):

> 1. **Structurally enforceable** — verifiable in schemas, types, or permitted operations, not merely in code review.
> 2. **Constraining of future implementation decisions** — capable of rejecting proposals that violate it.
> 3. **Identity-defining** — its absence changes what the system fundamentally is, not merely what it does.
> 4. **Independent of operator interpretation** — violation is detectable without subjective judgment.
>
> — [Charter §2](#2-constitutional-invariants)

**Falsifiability.** A constitutional claim is admissible if and only if it is structurally falsifiable. A property that cannot, in principle, be violated, observed, or audited is not a constitutional property; it is an aspiration, an aesthetic preference, or a research direction, and belongs elsewhere.

#### Structural Requirement

The two disciplines are applied at amendment time. The [`amendments.md` §Amendment Process](./amendments.md) procedure requires falsifiability review (Step 2) before any proposal advances to redaction (Step 3). The four qualification criteria are tested at the redaction stage and again at the final-merge checklist of the [`invariant-redactor`](../../.claude/skills/constitutional/invariant-redactor/SKILL.md) skill. The four-question falsifiability test — violation, observation, operationalization, non-circularity — is operationalized in the [`falsifiability-check`](../../.claude/skills/epistemic/falsifiability-check/SKILL.md) skill.

#### Rationale

The Charter constrains the system; this section constrains the Charter. Without qualification and falsifiability disciplines, any prose declared "constitutional" would carry the same weight as the structurally-enforceable invariants of §2.1 and §2.2, and the meaning of "constitutional" would collapse into the meaning of "important to someone."

The falsifiability discipline applies to all constitutional claims, including the claims of this section. The recursion is not vicious: the test procedure is defined externally to §4, in `falsifiability-check` §1, and §4's claims reduce to procedural artifacts (qualification testing at amendment time; falsifiability review at amendment time). The chain bottoms out in procedure, not in self-reference. This is the fixed-point reading.

#### Forbidden Anti-Patterns

- **Adopting an invariant whose violation requires subjective judgment.** Fails criterion 4. Detected by the observation test of [`falsifiability-check` §1.2](../../.claude/skills/epistemic/falsifiability-check/SKILL.md).
- **Adopting an invariant not structurally enforceable in schemas, types, or permitted operations.** Fails criterion 1. Detected by the operationalization test of [`falsifiability-check` §1.3](../../.claude/skills/epistemic/falsifiability-check/SKILL.md).
- **Adopting an invariant that does not constrain future implementation decisions.** Fails criterion 2. Surfaced at the `invariant-redactor` final-merge checklist; no per-claim mechanical check.
- **Adopting an invariant whose absence would not change what the system is.** Fails criterion 3. Surfaced at the `invariant-redactor` final-merge checklist; no per-claim mechanical check.

#### Boundary Conditions

- §4 does not govern internal project practice outside Charter governance. Commit message conventions, branch naming, code style, and README phrasing are operational and belong to [`CONTRIBUTING.md`](../../CONTRIBUTING.md) (process) and [`WORKFLOW.md`](../../WORKFLOW.md) (tooling).
- §4 governs the form of invariants, not their content. The committee chooses which invariants the project adopts; §4 filters candidate invariants into qualified versus non-qualified.
- §4 does not govern the infrastructure that supports Charter governance. Skills, hooks, CI workflows, agents, slash-commands, and per-project settings can be modified, replaced, or extended without Charter amendment, subject to RFC and decision-log discipline.

**[END CANDIDATE]**

---

#### Per-claim self-test (`falsifiability-check` §1: V = violation, O = observation, Op = operationalization, NC = non-circularity)

| # | Claim | V | O | Op | NC |
|---|---|---|---|---|---|
| 1 | Def-Q: A claim qualifies as a constitutional invariant iff it satisfies the four §2 criteria. | ✓ | ✓ | ✓ | ✓ |
| 2 | Def-F: A constitutional claim is admissible iff it is structurally falsifiable. | ✓ | ✓ | ✓ | ✓ |
| 3 | SR: The disciplines are applied at amendment time per `amendments.md`. | ✓ | ✓ | ✓ | ✓ |
| 4 | R-fix: §4 satisfies its own discipline; the recursion bottoms out in procedure. | ✓ | ✓ | ✓ | ✓ |
| 5 | AP1: Adopting an invariant whose violation requires subjective judgment fails criterion 4. | ✓ | ✓ | ✓ | ✓ |
| 6 | AP2: Adopting an invariant not structurally enforceable fails criterion 1. | ✓ | ✓ | ✓ | ✓ |
| 7 | AP3: Adopting an invariant that does not constrain future implementation decisions fails criterion 2. | ✓ | ✓ | ✓ | ✓ |
| 8 | AP4: Adopting an invariant whose absence would not change what the system is fails criterion 3. | ✓ | ✓ | ✓ | ✓ |
| 9 | BC1: §4 does not govern internal project practice. | ✓ | ✓ | ✓ | ✓ |
| 10 | BC2: §4 governs the form of invariants, not their content. | ✓ | ✓ | ✓ | ✓ |
| 11 | BC3: §4 does not govern the infrastructure that supports Charter governance. | ✓ | ✓ | ✓ | ✓ |

All eleven substantive claims pass all four tests. Note: claims 2 and 4 hinge on the fixed-point reading of §4's self-reference (Q2 resolution); the chain bottoms out in `falsifiability-check` §1's procedure, not in §4 itself. The criteria reproduced in the Definition blockquote (§2's four criteria, frozen) are inherited; they passed falsifiability at §2 v0.1 inception.

#### Hook-tripwire note (surface for Checkpoint 1.5.B)

Strict verbatim of §2's four criteria (per Q1 resolution) triggers two known-tradeoff false-positives in the binding text:

- **The importance-by-assertion adverb in §2 criterion 3** (the adverb between "what the system" and "is") — marketing tell. This is **exempted** by Gate 0a's blockquote-attribution mechanism: the criteria sit inside an attributed blockquote, which the hook's `eligible_blockquote_lines` helper marks as eligible. No bypass needed for this trip.
- **The singular data-definition noun in §2 criterion 1** (the noun preceding "types, or permitted operations") — vocabulary-drift forbidden synonym. This is **not** exempted: Gate 0a covers marketing only, not vocabulary-drift. Gate 0b's canonical-phrase whitelist does not include any phrase matching the §2 wording. §2's use of the noun is the legitimate case (the formal-definition sense, not the type sense) that `vocabulary-discipline` §4 reserves as canonical — but the hook does literal grep and cannot disambiguate. This is the canonical tripwire false-positive the hook's top comment names.

Two paths for Phase F commit (and likewise for the current scratch commit — both blockquote occurrences are scanned):

- **Path (a) — bypass with `--no-verify` and decision-log note.** Strict verbatim of §2 preserved; the bypass is registered in decision-log entry 0007 per CLAUDE.md §5.3, with justification that the hit is on a §2 verbatim quotation (a documented tripwire false-positive). Preferred under a strict reading of Q1 ("verbatim").
- **Path (b) — pluralize the data-definition noun in the blockquote.** The criterion then reads as "verifiable in [plural form], types, or permitted operations" — semantically identical, lexically one letter different. A reader comparing §4 to §2 sees one minor normalization; no bypass needed. An honest-but-not-strict reading of Q1.

I drafted Path (a) — strict verbatim — above. The committee picks at Checkpoint 1.5.B. Either path keeps §2 L41's promise fulfilled (the criteria are recorded formally in §4); they differ only on whether one letter of one criterion is normalized for hook hygiene.

#### Citation chain pre-verification (will be re-verified in Phase C.3)

| Citation | Current location | After v0.2 | Status |
|---|---|---|---|
| `falsifiability-check` SKILL.md L8 (quote of working text) | Charter §4 "pending working text" | Binding §4 Definition / Falsifiability paragraph | Quote text may need editorial fix in Phase D.3 — the L8 quote currently says "(pending working text)"; the qualifier becomes wrong. Otherwise pointer is correct. |
| `falsifiability-check` SKILL.md L118 (Source citations) | "Charter §4 Constitutional Design Rule (pending)" | Binding §4 | "(pending)" qualifier removed in Phase D.3. |
| `CLAUDE.md` L45 (`falsifiability` canonical vocabulary) | "Charter §4 pending" | Binding §4 | "pending" qualifier removed (not required immediately — CLAUDE.md is updated in Phase D.5 for the status table; this canonical-vocabulary line may be left or edited.) |
| `amendments.md` L22 (cites §4 for falsifiability review authority) | "Section 4 of the Charter" | Same. Now binding instead of pending. | No textual change required; citation now resolves to stronger source. |
| Glossary `falsifiability` Introduction L146 | "Charter §4 Constitutional Design Rule (pending working text)" | Binding §4 | "(pending working text)" qualifier removed in Phase D.2. |
| §2 L41 (FROZEN promise) | "recorded formally in §4 (pending)" | Promise fulfilled by binding §4 | "(pending)" qualifier in §2 L41 would become wrong, but §2 is FROZEN — no edit possible without re-amending §2. **Surface in Phase C.3:** is the "(pending)" qualifier on the §2 L41 phrase itself ("Section 4 — Constitutional Design Rule (pending)") considered Charter text whose semantic content has drifted, or is the parenthetical "(pending)" an editorial marker tied to §4's status that updates implicitly? Likely the latter — but worth explicit verification at Phase C.3. |
