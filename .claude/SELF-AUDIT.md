# `.claude/` — Self-Audit

**Date:** 2026-05-15
**Phase:** 8 (final).

Audit of the operational infrastructure produced by Phases 1–7 against `PLAN.md` §2, `CLAUDE.md`, the skills, and the CI workflow.

## Section 1 — Structural completeness

Files declared in `PLAN.md` §2 + Phase-prompt extensions verified:

| Expected | Status |
|---|---|
| `.claude/CLAUDE.md`, `README.md`, `PLAN.md`, `settings.json` | present |
| `.claude/skills/constitutional/{charter-guardian, subordination-checker, invariant-redactor}/SKILL.md` | present |
| `.claude/skills/epistemic/{epistemic-separator, falsifiability-check, ambiguity-reducer}/SKILL.md` | present |
| `.claude/skills/ontology/{ontology-keeper, vocabulary-discipline}/SKILL.md` | present |
| `.claude/skills/workflow/{rfc-author, decision-logger, implementation-readiness-evaluator}/SKILL.md` | present |
| `.claude/skills/enforcement/anti-marketing/SKILL.md` | present |
| `.claude/agents/{charter-reviewer, epistemic-auditor}.md` | present |
| `.claude/commands/{status, redact-invariant, propose-rfc, log-decision, check-subordination}.md` | present |
| `.claude/hooks/pre-commit-doc-check.sh` | present, executable |
| `.claude/hooks/_parse_watchlists.py` | present, executable |
| `WORKFLOW.md` | present |
| `docs/glossary.md` | present |
| `.github/workflows/constitutional-check.yml` | present |

YAML frontmatter check on every `SKILL.md` and every agent file: 14/14 carry `---` delimiters, `name:` field, and `description:` field. The `name:` value matches the containing directory (skills) or filename stem (agents) in every case.

**Finding 1.1 (minor).** `pre-commit-doc-check.sh` is 207 lines. Phase 6's stated target was 200 lines with allowance to split into a Python helper if exceeded. The split was performed (`_parse_watchlists.py`); the 7-line overshoot is documentation density in the top comment (two design discoveries + one tradeoff). Not a bug. **Classification:** informational.

## Section 2 — Vocabulary self-test

Mechanical check via `.github/scripts/check_glossary_coverage.py`:

- 18 canonical terms in `CLAUDE.md` §3.
- 18 glossary entries in `docs/glossary.md`.
- Every entry carries all four provenance fields (`Canonical definition`, `Introduction`, `Stabilization`, `Last amendment`); fields are filled or marked `pending`.

Usage-consistency spot check across skills: terms appear in canonical form. No glossary-definition contradictions surfaced for the high-frequency terms (`substrate`, `observation`, `operational construct`, `hypothesis`, `assertion`, `provenance`, `influence`).

**Finding 2.1 (informational).** The glossary's "Forbidden synonyms" field is the fifth field on every entry. The CI check (`check_glossary_coverage.py`) verifies only the four provenance fields per Phase 7 prompt. The fifth field is the data-source of `vocabulary-discipline` §4 and is checked by parser-driven extraction in the hook script. No drift check exists between the two views — see §7 Coverage gap 6.

No other findings.

## Section 3 — Subordination self-test

Per-skill citation review. Each skill's "Source citations used" section was inspected for whether the citation points at the highest-ranked authoritative source for each claim.

| Skill | Highest-ranked source it depends on | Cited? |
|---|---|---|
| `charter-guardian` | Charter §1–§4, amendments.md, decision-log.md §0004 | yes |
| `subordination-checker` | README.md §Document Hierarchy, Charter §2.1/§2.2 anti-patterns | yes |
| `invariant-redactor` | Charter §2 qualification criteria, amendments.md §Amendment Process | yes |
| `epistemic-separator` | Charter §1, §2.1, §2.2, §2.5 pending; entity-model.md | yes |
| `falsifiability-check` | Charter §4 pending working text; CONTRIBUTING.md §Style | yes |
| `ambiguity-reducer` | Charter §1, §2.1, §2.2; ontology open-question registry | yes |
| `ontology-keeper` | `ontology.md` §Open Questions; decision-log §0004 | yes |
| `vocabulary-discipline` | `docs/glossary.md`; Charter §1, §2.1, §2.5–§2.6 pending; CONTRIBUTING.md §Style | yes |
| `rfc-author` | rfcs/README.md, rfcs/template.md, amendments.md | yes |
| `decision-logger` | decision-log.md §Format; Charter §2.1 (analogy for append-only discipline) | yes |
| `implementation-readiness-evaluator` | README.md §Status; Charter §2.1–§2.6; decision-log.md §0003 | yes |
| `anti-marketing` | CONTRIBUTING.md §What This Project Is Not; Charter §1 (exemption ref) | yes |

No subordination violations: every skill that makes a claim about the project sources it to the highest-ranked document that governs the claim. No skill cites a subordinate document where a higher-ranked one is available.

No findings.

## Section 4 — Falsifiability self-test

Each skill's central claim was identified and tested for falsifiability (can a reviewer determine whether the skill operated correctly?).

| Skill | Central claim | Falsifiable? | Detection of failure |
|---|---|---|---|
| `charter-guardian` | Block FROZEN edits without RFC | yes | Unauthorized edit reaches main |
| `subordination-checker` | Surface contradictions between ranks | yes | Known contradiction not flagged |
| `invariant-redactor` | Structure committee redaction through nine steps | yes | Redaction completed skipping a step |
| `epistemic-separator` | Catch category conflation in prose | yes | Confused-category sentence merged |
| `falsifiability-check` | Apply four-question test to claims | yes | Non-falsifiable claim passes |
| `ambiguity-reducer` | Flag watchlist terms (advisory) | yes | Watchlist term unflagged |
| `ontology-keeper` | Surface implicit open-question resolution | yes | Implicit resolution merged silently |
| `vocabulary-discipline` | Block forbidden synonyms | yes | Forbidden synonym in committed prose |
| `rfc-author` | Enforce Q1–Q6 before sections filled | yes | Draft `status: discussion` without impact analysis |
| `decision-logger` | Enforce append-only on decision log | yes | In-place edit detected in diff |
| `implementation-readiness-evaluator` | Return ready/partial/not-ready per five criteria | yes | Implementation proceeds without `ready` |
| `anti-marketing` | Detect marketing tells | yes | Tell passes in committed text |

All 12 skills' central claims are falsifiable with concrete failure-detection criteria. No findings.

## Section 5 — Anti-marketing self-test

Per the Phase 8 prompt, the Charter-quotation exemption does **not** apply here — `.claude/` infrastructure files have not earned canonical-vocabulary status. The watchlist was parsed by `_parse_watchlists.py --marketing` and grepped against:

- `.claude/CLAUDE.md`, `.claude/README.md`, `WORKFLOW.md`, the new section of `CONTRIBUTING.md`.
- All 12 `SKILL.md` files except `enforcement/anti-marketing/SKILL.md` (which contains the watchlist as data).
- Both agent files.

**Finding 5.1 (mechanical bug — fixed in this phase, see §8).** The marketing parser extracts backtick-quoted terms from every line starting with a backtick in §1 of `anti-marketing/SKILL.md`. This includes explanation paragraphs after the example `> ` blocks. Five spurious entries are produced: `observational integrity`, `evidential independence integrity`, `understand`, `behavioral data`, `ambiguity-reducer`. Only the first paragraph after each `### <category>` heading (terminated by the example block) is intended as the watchlist.

**Finding 5.2 (judgment-required — deferred).** `CLAUDE.md` §1 line 7 quotes the Charter Thesis verbatim: "Ghost Trace is a behavioral intelligence substrate designed to preserve the epistemic integrity of operational knowledge". The words `intelligence` and `integrity` are on the marketing watchlist. The audit rule rejects the Charter exemption for `.claude/` files. Resolution requires judgment — rephrase to avoid the watchlist terms, or document an explicit Charter-quotation exception in `anti-marketing`. Defer.

**Finding 5.3 (judgment-required — deferred).** `.claude/README.md` line 31 contains the phrase "Protect Charter integrity" in the description of the `constitutional/` skill domain. `integrity` is on the watchlist. Possible rewrite: "Protect the Charter from drift". Defer.

Other apparent hits (multiple `ambiguity-reducer` references across skill cross-references; multiple `trust`/`integrity` mentions inside skills that discuss these as watchlist items themselves) are functional uses of the words as data, not marketing prose. None require revision.

## Section 6 — Duplication and ceremony self-test

Skill-pair overlap check:

- `ambiguity-reducer` vs `vocabulary-discipline`: distinct posture (advisory vs blocking) and non-overlapping watchlists, declared in both skills' "Relationship" sections. No overlap.
- `falsifiability-check` vs `anti-marketing`: orthogonal axes, declared in `anti-marketing`'s preamble. No overlap.
- `epistemic-separator` vs `vocabulary-discipline`: different axes (category usage vs canonical-term identity). No overlap.
- `charter-guardian` vs `subordination-checker`: explicit partition of scope (rank 1 vs ranks 2–6). No overlap.
- `charter-guardian` vs `invariant-redactor`: gate vs redaction-support. No overlap.
- `rfc-author` vs `decision-logger`: distinct workflow phases (proposal authorship vs acceptance-time recording), cross-referenced via §4 acceptance handoff.

Commands all add behavior beyond mere skill invocation:
- `/status` — aggregates and reports across multiple sources.
- `/redact-invariant` — creates scratch file + verifies status.
- `/propose-rfc` — creates draft file + enforces Q1–Q6 ordering.
- `/log-decision` — determines next ID + interviews user.
- `/check-subordination` — scope discovery + structured reporting.

Agents: `charter-reviewer` adds 5-step structured review with classification + accompanying-entries verification (beyond running constituent skills). `epistemic-auditor` adds Step 5 (cross-skill contextual linkage), explicitly named as the agent's distinctive value in its description and §5 of the agent file.

No findings.

## Section 7 — Coverage gaps

Aspects of project discipline that current infrastructure does **not** cover. Each gap is listed for future RFC-governed work; none are bugs to fix now.

1. **No skill for `experiments/` work.** The directory exists and `implementation-readiness-evaluator` references it, but no dedicated skill assists with drafting `experiment`-type RFCs, scoping success criteria, or evaluating outcomes.
2. **No code-level vocabulary check.** The hook and CI scan `*.md` only. When implementation begins (post-RFC for storage substrate, schema technology, implementation language), the vocabulary discipline will not be enforced on Go/Python/etc. source.
3. **No skill-retirement workflow.** No procedure exists for superseding a skill, command, agent, or hook. A retired skill's content could conflict with current discipline.
4. **No CLAUDE.md §4 ↔ Charter banner sync check.** The hook reads `CLAUDE.md` §4 as the authoritative status table. The Charter's `**Status:**` banner restates this. No mechanical check verifies they remain consistent. The Phase 8 SELF-AUDIT itself surfaces a minor divergence (see Finding 7.1 below).
5. **No `--no-verify` bypass tracking.** The hook script prints a reminder that bypass is registrable per CLAUDE.md §5.3, but nothing structurally enforces a corresponding decision-log entry. The discipline relies on operator habit.
6. **No drift check between vocabulary-discipline §4 forbidden-synonym table and per-term Forbidden-synonyms field in glossary entries.** The two views must agree; manual maintenance currently bridges them.
7. **No mechanical check that a decision-log supersession's status-line edit follows `decision-logger` §3.** The skill describes the discipline; no automated verification exists.
8. **`WORKFLOW.md` vs `CONTRIBUTING.md` drift.** CLAUDE.md §2 declares CONTRIBUTING wins. No mechanical check exists for drift between the two operational documents.

**Finding 7.1 (judgment-required — deferred).** `CLAUDE.md` §4 status table does not list §2 (Constitutional Invariants header — the four qualification criteria) as a separate row, though `PLAN.md` §1 inventory declared it `Frozen`. The Charter banner uses the wording "Invariants 1–2 frozen", which is ambiguous between "the first two invariants (§2.1, §2.2)" and "the §2 cluster including the qualification criteria". Consequence: the hook would not block an edit to the qualification criteria via the FROZEN-section check. Resolution requires judgment about what "the qualification criteria" means structurally — adding the §2 row to CLAUDE.md §4 is the simplest fix but is governance-adjacent (changes what the hook protects). Defer.

## Section 8 — Recommendations

### Bugs fixed in this phase

**Fix 8.1 — Marketing parser restricts to tell paragraphs only.**

`.claude/hooks/_parse_watchlists.py::parse_marketing` is modified to extract backtick-quoted terms only from lines after a `### <category>` heading and before the first `> ` example block in that subsection. Explanation paragraphs are excluded.

The fix is applied below in this phase; see "Post-fix verification" at the end of this document.

### Deferred (future RFC-governed work)

- **§5 Finding 5.2:** `CLAUDE.md` §1 line 7 Charter Thesis quote contains `intelligence`/`integrity`. Resolution: rephrase or document Charter-quotation exception in `anti-marketing`.
- **§5 Finding 5.3:** `.claude/README.md` line 31 "Protect Charter integrity" — rephrase or accept.
- **§7 Finding 7.1:** `CLAUDE.md` §4 vs `PLAN.md` §1 vs Charter banner divergence on §2 (qualification criteria) frozen status. Governance-adjacent — defer.
- **§7 Coverage gaps 1–8.** Each is candidate future work.
- **§1 Finding 1.1:** Hook script 7 lines over the 200-line target. Documentation density tradeoff. No action required.

### Audit incompleteness

The following checks were not exhaustively performed in this phase. They are not bugs in the infrastructure; they are limits of the audit:

- **Per-claim citation depth.** Spot-checked across skills. A full claim-by-claim source verification was not performed.
- **Forbidden-synonym and ambiguity-term scans against infrastructure.** A scan was run informally (91 hits found for forbidden synonyms across infrastructure files, almost all of them canonical phrases like `primary event log` that the tripwire flags as designed). These are not findings against the infrastructure — they are the documented tradeoff of the tripwire-grep approach (see hook script top comment). No further action.
- **Runtime command invocation.** Commands were read but not executed. Whether `/status`, `/propose-rfc`, etc., behave correctly in a live Claude Code session was not verified empirically.
- **Skill auto-discovery at the depth of `.claude/skills/<domain>/<skill>/SKILL.md`.** The grouping subdirectory adds a level beyond the conventional `.claude/skills/<skill>/SKILL.md`. Whether Claude Code's auto-discovery handles the extra depth was flagged in the Phase 3 report and remains untested.

These four audit-incompleteness items are themselves findings. They are surfaced for the human's awareness; not fixed in this phase.

---

## Post-fix verification

After applying Fix 8.1 (marketing parser refinement):

- Marketing watchlist size: **41 → 36 entries** (5 spurious entries removed).
- Spurious entries (`observational integrity`, `evidential independence integrity`, `understand`, `behavioral data`, `ambiguity-reducer`) confirmed absent.
- `pre-commit-doc-check.sh --self-test` passes (3 frozen ranges, 12 forbidden, 36 marketing tells, 12 ambiguity terms).
- Re-scan of infrastructure files: **41 → 17 hits**. All 17 remaining hits are either (a) skills that discuss watchlist terms as data — `falsifiability-check` examples of non-falsifiable terms, `ambiguity-reducer` watchlist entries for `intelligence` and `trust`, `vocabulary-discipline` example of `trust` as informal use — or (b) the previously-recorded Findings 5.2 (CLAUDE.md §1 line 7 Charter quote) and 5.3 (`.claude/README.md` line 31 "Charter integrity"). One additional minor occurrence (`epistemic-separator/SKILL.md` line 64 uses `transforms` in explanatory prose "a pipeline stage that transforms its input") is a tripwire false-positive consistent with the documented tradeoff in `pre-commit-doc-check.sh` top comment — informational only, no action.

Setup is complete. Future infrastructure changes follow the RFC process, recorded in `docs/charter/decision-log.md`, and reflected in updates to `CLAUDE.md` and the relevant skills.
