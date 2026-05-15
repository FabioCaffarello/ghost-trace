# Decision Log

This document records architectural decisions made during the development of Ghost Trace. It functions as an Architecture Decision Record (ADR) log, but with constitutional awareness: every decision is evaluated against the [Constitutional Charter](./constitutional-charter.md).

## Format

Each decision is recorded as a separate entry with the following structure:

- **ID** — sequential, zero-padded (`0001`, `0002`, ...).
- **Title** — short, declarative.
- **Status** — `proposed`, `accepted`, `superseded`, `rejected`.
- **Date** — when the decision was recorded.
- **Context** — what made the decision necessary.
- **Decision** — what was decided.
- **Constitutional review** — which invariants the decision was tested against, and the outcome.
- **Consequences** — what the decision constrains, enables, or excludes.
- **Supersession** — if applicable, which decision this one replaces.

Decisions are recorded once and never edited except for status changes. A decision that proves wrong is superseded by a new decision, not rewritten. This mirrors the Charter's treatment of observations and assertions: history is preserved, not overwritten.

---

## `0001` — Adopt event-log immutability as constitutional invariant

- **Status:** accepted
- **Date:** _(charter inception)_
- **Context:** Initial architectural framing required a decision on whether the primary event log would be append-only by enforced guarantee, by procedural convention, or by aspirational policy.
- **Decision:** Append-only by enforced guarantee. Codified as [Constitutional Invariant 2.1 — Observational Integrity](./constitutional-charter.md#21-observational-integrity).
- **Constitutional review:** Pre-dates and partly defines the Charter itself. The decision is the substrate on which all other invariants rest.
- **Consequences:** Storage substrate selection is constrained to systems capable of physical, cryptographic, or storage-enforced write-once guarantees. Operational pressure to "fix" or "clean" historical records cannot be satisfied through mutation; only through superseding records.
- **Supersession:** none.

---

## `0002` — Adopt ontological tripartition (observation / construct / hypothesis)

- **Status:** accepted
- **Date:** _(charter inception)_
- **Context:** Multiple framings of the system data model were considered, including a unified assertion model with a discriminator field. The unified model would have simplified short-term implementation but conflated categories that the Thesis treats as ontologically distinct.
- **Decision:** Three structurally distinct categories of knowledge, each with its own type and operations. Codified as [Constitutional Invariant 2.2 — Epistemic Separation](./constitutional-charter.md#22-epistemic-separation).
- **Constitutional review:** This decision is itself a constitutional invariant. It is the structural mechanism by which the Thesis's central distinction is preserved.
- **Consequences:** Schema cannot be unified across categories. Cross-category operations require typed transformations. Promotion of hypotheses to operational use cannot be implicit.
- **Supersession:** none.

---

## `0003` — Defer storage technology selection

- **Status:** accepted
- **Date:** _(charter inception)_
- **Context:** Initial conversations considered NATS JetStream and Kafka as candidate backbones, with object storage (Parquet on S3 or MinIO) for cold archive. The decision pressure was to select one before constitutional work was complete.
- **Decision:** Defer technology selection until after the Charter and the relevant portions of the Ontology are stabilized. The Charter constrains storage technology by property (immutability, content-addressing, reconstructibility), not by name.
- **Constitutional review:** Decision is consistent with the precedence rule: technology choices are subordinate to constitutional properties. Recording the deferral is itself an exercise of that precedence.
- **Consequences:** Implementation work that requires a concrete storage substrate must either propose a candidate via RFC (subject to constitutional review) or remain at the schema/contract layer until selection is made.
- **Supersession:** none. Selection itself will be a separate decision.

---

## `0004` — Charter is authoritative; subordinate documents may evolve under implementation pressure

- **Status:** accepted
- **Date:** _(charter inception)_
- **Context:** The project needs a clear rule about what is constitutional (slow to change, requires amendment) and what is subordinate (may evolve as implementation reveals ambiguity).
- **Decision:** `docs/charter/` is the constitutional surface; changes require formal amendment per [`amendments.md`](./amendments.md). `docs/ontology/`, `docs/architecture/`, and `docs/rfcs/` may evolve under implementation pressure provided they do not conflict with the Charter. Conflicts are resolved in favor of the Charter; subordinate documents are revised.
- **Constitutional review:** Pre-figures Section 4 of the Charter (Constitutional Design Rule), which formalizes the precedence rule.
- **Consequences:** Implementation work cannot silently amend the Charter through subordinate document edits. Constitutional drift is detectable as conflict.
- **Supersession:** none.

---

## `0005` — Charter quotation by attributed blockquote — mechanical exemption rule

- **Status:** accepted
- **Date:** 2026-05-15
- **Context:** SELF-AUDIT.md Finding 5.2 surfaced that `.claude/CLAUDE.md` §1 line 7 quotes the Charter Thesis verbatim, but the hook's marketing-tell check flagged Charter watchlist terms because the parenthetical attribution at line end was not recognizable as a quotation by mechanical means. The `anti-marketing` skill §4 already declared a Charter-quotation exemption at the prose level; the gap was mechanical.
- **Decision:** Charter and Ontology quotations are recognized mechanically when expressed as markdown blockquotes (lines beginning with `>`) attributed by a line matching `>\s*—\s*\[(Charter|Ontology)`. The hook's marketing check skips watchlist hits on lines within such attributed blockquote blocks. The exemption is scoped to marketing-tell detection only; vocabulary-drift and ambiguity advisories continue to apply.
- **Constitutional review:** No invariant amended. The exemption operationalizes a rule already declared in `anti-marketing` §4 (prose-level). It does not weaken enforcement; it makes the existing exemption mechanically detectable. The vocabulary-drift check is not exempted, because the Charter itself uses canonical phrases — a quotation containing a forbidden synonym would indicate either a stale Charter or a misquoted source.
- **Consequences:** Faithful blockquote quotation of Charter or Ontology text in any document is no longer flagged by the marketing check. Paraphrase of Charter prose without blockquote form remains flagged. New convention: when `CLAUDE.md`, `WORKFLOW.md`, `README.md`, or any skill needs to invoke Charter prose, use blockquote + attribution rather than parenthetical reference.
- **Supersession:** none.

---

## `0006` — Clarify constitutional surface protected by the infrastructure (v0.1.1)

- **Status:** accepted
- **Date:** 2026-05-15
- **Context:** Two ambiguities surfaced during Gate 0 of post-setup work. Phase 8 SELF-AUDIT Finding 7.1 noted that the Charter banner's wording ("Invariants 1–2 frozen") is ambiguous about the status of the §2 header — the four invariant qualification criteria, which `invariant-redactor` already treats as operative. Empirical pre-scan during Gate 0b further showed that the hook's `in_scope` predicate never included `.claude/CLAUDE.md` or `.claude/README.md`, despite the SELF-AUDIT having treated their cleanliness as a finding. Both ambiguities were governance-adjacent.
- **Decision:** Charter amendment v0.1.1, recorded in [`amendments.md`](./amendments.md) and originated as RFC [`charter-amendment-v0-1-1-clarify-protected-surface`](../rfcs/draft/charter-amendment-v0-1-1-clarify-protected-surface.md). Explicitly declares the §2 header FROZEN (`.claude/CLAUDE.md` §4 status table updated). Extends the hook `in_scope` predicate to include `.claude/CLAUDE.md` and `.claude/README.md`. Three editorial rewrites bring previously out-of-scope text into compliance. New canonical-phrase exemption mechanism — registered phrases "primary event log", "decision log", "event log", and "historical fact" — prevents legitimate canonical vocabulary from being reported as drift. Hook fallback fix (no-added-lines case) prevents whitespace-only commits from scanning pre-existing content.
- **Constitutional review:** §2 header (status change, FROZEN). §2.1, §2.2 unchanged. No new invariant introduced. No glossary term redefined. No open Ontology question resolved. No ceremony without behavioral consequence: each clause of the amendment changes hook behavior or eliminates a documented latent bug.
- **Consequences:** Future edits to lines 32–42 of `constitutional-charter.md` (the §2 header) require formal amendment. Future edits to `.claude/CLAUDE.md` and `.claude/README.md` are subject to vocabulary, marketing, and ambiguity discipline. The canonical-phrase whitelist does not grow by inference; each addition requires a new decision-log entry. Whitespace-only commits to in-scope files no longer trigger scans of pre-existing content. `.claude/SELF-AUDIT.md` and `.claude/PLAN.md` remain explicitly out of scope; this exclusion is documented in the `in_scope` function.
- **Supersession:** none. Successor amendments may further clarify the protected surface; this decision establishes the canonical pattern for such clarifications (explicit, amendment-backed, decision-log-recorded).

---

## `0007` — Gate 1 pilot: §4 committee-mode redaction (path c narrow-but-not-minimal)

- **Status:** accepted
- **Date:** 2026-05-15
- **Context:** Gate 1 was the pilot application of the committee-mode redaction procedure (`invariant-redactor` Steps 1.1–1.5) to a pending Charter element. §4 was chosen as pilot because of its meta-character — rules about the Charter itself, not about the substrate — permitting orthogonal testing of the procedure before §2.3–§2.6 redactions which exercise object-level skills. The pilot proceeded through five steps across five prior commits and one hook-fix commit, all merged to main: anchor + scaffold (`2ec5f79`), falsifiability findings (`3abcfc2`), hook-fix for non-ASCII filenames surfaced during pilot work (`9577520`), epistemic-skill findings (`0e0be10`), Steps 6/7/8 + Synthesis (`18ac6a6`), and this closure commit (Phase F of Step 1.5). The full evidentiary record is preserved in [`docs/charter/in-committee/§4-constitutional-design-rule.md`](./in-committee/§4-constitutional-design-rule.md).
- **Decision:** Charter amendment v0.2 (see [`amendments.md`](./amendments.md)) enacts the pilot's substantive output. §4 binding text retains the two surviving bullets — qualification criteria (Bullet 1, fulfilling §2 L41's frozen promise that the criteria "are themselves recorded formally" in §4) and falsifiability discipline (Bullet 4, anchoring five structural infrastructure citations across `falsifiability-check`, `amendments.md`, `CLAUDE.md`, and the glossary). Bullets 2 (amendment philosophy) and 3 (precedence rule) of the working stub are removed as substantive duplicates without anchor obligations. §4 status moves `pending` → `frozen`.
- **Constitutional review:** No invariant amended outside §4 itself. The §2 L41 frozen forward-citation to §4 is fulfilled (made true) by the new binding §4. Five existing structural citations to §4 are validated in Phase C.3 of Step 1.5; all anchors resolve, four require editorial qualifier removal (enacted in Phase D). The §2 L41 frozen text contains a "(pending)" qualifier on its citation to §4 that becomes inaccurate after v0.2; the committee accepts this as a known minor inconsistency in frozen text per the Phase C.3 guidance (editing a single parenthetical in frozen text is disproportionate). No open Ontology question resolved. No new vocabulary introduced. No new invariant introduced.
- **Consequences:**
  - **First Charter section moved from `pending` to `frozen` since v0.1 inception.** The committee-mode procedure produced its first binding-text output; the precedent for §2.3–§2.6 redactions is established.
  - **The committee-mode procedure is validated.** It produced decidable output, surfaced empirical evidence (citation grep, falsifiability failures), and arrived at a recommendation different from the naive reading (narrow-but-not-minimal — Bullets 1 and 4 — rather than minimal — Bullet 4 only — or full retention).
  - **Methodological observation 1 — epistemic-skill applicability is domain-typed.** [`epistemic-separator`](../../.claude/skills/epistemic/epistemic-separator/SKILL.md) and [`ambiguity-reducer`](../../.claude/skills/epistemic/ambiguity-reducer/SKILL.md) have natural inapplicability to meta-prose: their domain is substrate-touching prose, and §4 is about Charter governance, not substrate. Step 1.3 found 8/8 cells inapplicable or zero-finding. Future redactions of §2.3–§2.6 (object-level) will see full applicability; the expectation is encoded as precedent here. The committee should not interpret §4's null finding as a skill failure.
  - **Methodological observation 2 — non-duplication check is the operative step for late-redacted invariants.** When the infrastructure has been built before a Charter section's redaction, Step 8 (non-duplication) surfaces whether the section is content-bearing or anchor-bearing. §4 turned out to be anchor-bearing (Bullet 4) plus promise-fulfilling (Bullet 1). Without Step 8's bidirectional analysis (strong reading + weak/anchor reading), the right answer would have been missed. Future redactions should treat Step 8 as a first-class step, not as a final box-tick.
  - **Methodological observation 3 — citation grep is decisive.** Step 8's verdict was empirical, not opinion. The initial 4-citation grep produced one recommendation (narrow-minimal); a second-pass wider grep surfaced three additional citations (most consequent: §2 L41 in frozen text), reversing the recommendation to narrow-but-not-minimal. Future redactions should perform citation grep early in Step 8, with explicit wider-criteria re-runs, and should treat citation chain analysis as first-output not afterthought.
  - **Methodological observation 4 — pre-staging hook rephrasings are documented tradeoff responses, not findings softenings.** Step 1.4 surfaced three pre-staging rephrasings for tripwire false-positives: a singular data-definition noun pluralized to escape the boundary match; a document-management verb substituted with one of equivalent meaning; an importance-by-assertion adverb omitted via rephrase. The hook's top comment documents this exact response path. Step 1.5 Phase B-to-Phase F applied the same approach — verbatim §2 criteria rendered with the pluralized form to escape the bare-singular boundary match. The discipline distinguishes "rewrite to pass" (forbidden — softens findings) from "rewrite to avoid documented tripwire false-positive" (permitted — preserves content with single-letter normalization).
  - **Methodological observation 5 — Phase F bypass for amendments that move FROZEN status while editing text.** The Phase F commit uses `git commit --no-verify` because amending §4 — moving its CLAUDE.md §4 status-table row to `frozen` while replacing its body in the same commit — triggers the hook's `check_frozen_charter` block on §4's now-frozen range. The four amendment artifacts (RFC `charter-amendment-v0-2-section-4-redaction`, falsifiability review recorded in Step 1.5 Phase B's per-claim self-test, `amendments.md` v0.2 entry, version bump v0.1.1 → v0.2) are the constitutional justification per [`charter-guardian` §2 Step 3](../../.claude/skills/constitutional/charter-guardian/SKILL.md). The bypass is registered here per CLAUDE.md §5.3. Future amendments that move a section's FROZEN status while editing its body will face the same hook-block; the bypass with decision-log note is the documented path.
  - **Watchlist extension candidate carried forward — `conflict`.** Surfaced in Step 1.2 via Bullet 3's operationalization failure (the term is undefined in Charter, glossary, or skills). [`ambiguity-reducer` §3](../../.claude/skills/epistemic/ambiguity-reducer/SKILL.md) names the procedure for watchlist additions. Disposition: deferred; not enacted by this gate. Future RFC may add `conflict` to the watchlist.
  - **Glossary entry candidate carried forward — "constitutional invariant".** Surfaced in Step 1.2 as missing from [`CLAUDE.md` §3 canonical vocabulary](../../.claude/CLAUDE.md). Belongs to `vocabulary-discipline`, not `ambiguity-reducer`. Disposition: deferred; not enacted by this gate. Future mini-RFC may add the entry.
  - **§2 L41 "(pending)" qualifier acknowledged as inaccurate, not edited.** §2 is FROZEN per v0.1.1. The parenthetical "(pending)" on §2 L41's forward-citation to §4 is structurally minor; editing it would require bypass-of-frozen-section for a single parenthetical, which is disproportionate. Recorded here as a known acknowledged drift; if a future §2-touching amendment occurs, the editorial fix can be bundled.
- **Supersession:** none.

The five methodological observations and the two carried-forward items are the pilot's contribution to procedure beyond the §4 redaction itself. They are recorded here, not in a separate document, because the decision log is the project's source-of-record for methodological precedent.

---

## `0008` — Redaction order for pending invariants (§2.5 → §2.3 → §2.4 → §2.6); Gate 1 carry-forwards enacted

- **Status:** accepted
- **Date:** 2026-05-15
- **Context:** Gate 1 closed with §4 frozen as v0.2. Four invariants remain pending: §2.3 (Provenance Integrity), §2.4 (Inferential Influence Disclosure), §2.5 (Hypothesis Lifecycle Explicitness), §2.6 (Evidential Independence Integrity). The order in which they are redacted matters: each invariant references concepts the others define, and redaction without dependency respect would either produce binding text with forward references to pending material or force later redactions to revise earlier ones. Gate 1's decision-log entry §0007 carried forward two items requiring enactment (`conflict` watchlist candidate; "constitutional invariant" glossary entry) and one acknowledged drift (§2 L41 parenthetical). This entry binds the redaction order, enacts the two watchlist/glossary carry-forwards, and formally queues the parenthetical correction.
- **Decision:** Redaction order is **§2.5 → §2.3 → §2.4 → §2.6**. The justification is dependency-driven: §2.5 has the broadest dependency surface (defines Category III operations — formation, merge, split, promotion, demotion, dissolution — that §2.3, §2.4, and §2.6 all reference); §2.3 depends on §2.5 (provenance records reference Category III operations); §2.4 depends on §2.3 (inferential influence is a structural attribute of provenance records); §2.6 depends on §2.4 (independence presupposes declared influence, and §2.6 also depends on [Ontology Open Question 3](../ontology/entity-model.md) which may require an experiment-type RFC). Each invariant follows the same committee-mode procedure piloted on §4 in Gate 1 (`invariant-redactor` Steps 1.1–1.5). Adjacent ontology work may proceed in parallel where the supporting Ontology Open Question requires empirical or experimental resolution.
- **Constitutional review:** No invariant amended. This entry establishes calendar discipline and enacts two non-substantive carry-forward registrations (a watchlist addition and a glossary entry). The precedent for committee-mode redaction is §4 v0.2 (decision-log §0007). No new vocabulary introduced at the Charter level; the glossary entry for "constitutional invariant" formalizes vocabulary already in operational use across the Charter and skills.
- **Consequences:**
  - **Future redactions begin with §2.5.** Subsequent gates apply `invariant-redactor` Steps 1.1–1.5 to §2.5, then §2.3, then §2.4, then §2.6.
  - **§3 Non-Goals is not in this ordering.** Its dependencies are not within §2.x; it can be redacted in parallel or after §2.6 at committee discretion. A separate decision-log entry will register §3's redaction timing when it is scheduled.
  - **Departures from this order require a new decision-log entry justifying the departure.** Silent reordering is not permitted; the redaction calendar is itself a constitutional artifact.
  - **`conflict` added to `ambiguity-reducer` watchlist** (Subject B of Gate 2). The skill's §1 now lists 13 terms (was 12). Watchlist extension follows `ambiguity-reducer` §3 procedure. Carry-forward from §0007 enacted.
  - **`constitutional invariant` added to `docs/glossary.md` and `.claude/CLAUDE.md` §3 canonical vocabulary** (Subject C of Gate 2). The five-field glossary entry follows the template established by `subordination` and `falsifiability` entries. Carry-forward from §0007 enacted.
  - **§2 L41 parenthetical correction queued.** The phrase "(pending)" qualifying §2 L41's forward-reference to §4 is inaccurate after v0.2 (§4 is frozen). Editorial correction is deferred until a §2-touching amendment occurs, at which point this fix is bundled. Until then, the inaccuracy is acknowledged. This formalizes the §0007 acknowledgment.
- **Supersession:** none.

---

## `0009` — §2.5 redaction plan: Ontology RFCs Q2 + Q4 before charter redaction

- **Status:** accepted
- **Date:** 2026-05-15
- **Context:** Per [`decision-log §0008`](#0008--redaction-order-for-pending-invariants-25--23--24--26-gate-1-carry-forwards-enacted), the redaction order for pending invariants is §2.5 → §2.3 → §2.4 → §2.6. Strategic deliberation before §2.5 redaction surfaced that §2.5 binding text depends on two Ontology Open Questions: Q2 (Hypothesis subtypes — distinct types vs tags on a single type) and Q4 (Promotion → demotion criterion). Both questions appear verbatim in [`docs/ontology/ontology.md`](../ontology/ontology.md) §Open Questions for Committee Resolution. Q2 reappears in [`docs/ontology/entity-model.md`](../ontology/entity-model.md) §Open Modeling Questions in slightly rephrased form; Q4 reappears in [`docs/ontology/lifecycle-semantics.md`](../ontology/lifecycle-semantics.md) §Open Modeling Questions in a narrower variant (the lifecycle-semantics.md framing implicitly pre-commits to an independence-based measurement basis, partial drift from the canonical Q4 form). Redacting §2.5 without resolving Q2 and Q4 would either silently pick a resolution (violating [`ontology-keeper`'s](../../.claude/skills/ontology/ontology-keeper/SKILL.md) discipline — open questions are committee-resolved, not infrastructure-resolved) or produce vague binding text deferring to "later resolution" (failing [§4 falsifiability discipline](./constitutional-charter.md#4-constitutional-design-rule), frozen v0.2).
- **Decision:** §2.5 redaction is preceded by resolution of two Ontology RFCs in [`docs/rfcs/draft/`](../rfcs/draft/):
  - [`ontology-revision-q2-hypothesis-subtypes.md`](../rfcs/draft/ontology-revision-q2-hypothesis-subtypes.md)
  - [`ontology-revision-q4-promotion-demotion-criterion.md`](../rfcs/draft/ontology-revision-q4-promotion-demotion-criterion.md)

  Both are opened at status `discussion`. Resolution path for each: discussion → committee redaction → decision-log entry registering the resolved candidate → corresponding Ontology document revision (Scaffold → Drafted for `entity-model.md` Category III upon Q2 resolution, and for `lifecycle-semantics.md` Category III + Promotion Mechanism upon Q4 resolution). The two RFCs are independent of each other and may be redacted in parallel; their interaction is recorded in each RFC's Open Questions section. The §2.5 redaction itself follows only after both decision-log entries are recorded.
- **Constitutional review:** No Charter invariant amended by this entry. The decision sets calendar discipline (sequencing Ontology resolution before Charter redaction). The precedent for this sequencing is the §4 redaction in Gate 1 ([`decision-log §0007`](#0007--gate-1-pilot-4-committee-mode-redaction-path-c-narrow-but-not-minimal)): §4 surfaced citation chain dependencies that reversed the redaction recommendation; the pre-§2.5 work surfaces Ontology dependencies in the same spirit. Both are operational clarifications of what [`invariant-redactor`](../../.claude/skills/constitutional/invariant-redactor/SKILL.md) Step 8 (non-duplication / dependency check) means in practice. No new vocabulary introduced. No open Ontology question resolved by this entry (the entry plans their resolution; resolution happens in the RFCs' redaction phases).
- **Consequences:**
  - Two RFCs are now in `discussion` status awaiting committee redaction. Their drafts include three-candidate (Q4) and two-candidate-plus-rejected-alternative (Q2) structures; neither RFC pre-decides its own question per the constraint stated in each RFC's Summary.
  - §2.5 redaction begins only after both RFCs are accepted (their decision-log entries assigned). Departure from this sequencing requires a new decision-log entry justifying the departure, per the same principle articulated in [§0008](#0008--redaction-order-for-pending-invariants-25--23--24--26-gate-1-carry-forwards-enacted).
  - [`docs/ontology/lifecycle-semantics.md`](../ontology/lifecycle-semantics.md) Category III and Promotion Mechanism sections transition from `Scaffold` to `Drafted` upon Q4 resolution; [`docs/ontology/entity-model.md`](../ontology/entity-model.md) Category III transitions similarly upon Q2 resolution. The [`docs/ontology/ontology.md` §Status](../ontology/ontology.md) table is revised as part of each resolution's decision-log enactment; the `ontology-keeper` registry §1 is revised in the same change per `ontology-keeper` §4 procedure.
  - **Methodological observation for future redactions (§2.3, §2.4, §2.6, §3):** before opening a charter-redaction Gate, run a dependency scan against Ontology Open Questions and against any Scaffold-status Ontology documents that the target invariant references. Surface the dependencies as pre-redaction RFCs in the same spirit as this entry. This is the §4-pilot-revealed generalization captured by §0007's methodological observation 2 (non-duplication check is the operative step for late-redacted invariants) and observation 3 (citation grep is decisive): redaction Gates have pre-Gates, and the pre-Gates are where dependency surface is mapped.
  - The third Ontology open question relevant to §2.5 — Q1 (Session duality) — is NOT in this plan's scope. Q1 affects §2.3 (provenance of sessions) more than §2.5 (lifecycle of hypotheses) directly. Q1 will be addressed before §2.3, not now.
  - Q3 (independence formal definition) is referenced by Q4's Candidate B but is not resolved by Q4's RFC. Q3 is the dependency for §2.6; it remains open and will be addressed in §2.6's pre-Gate.
  - Q5 (influence propagation) is referenced by Q4's Candidate C but is similarly not resolved by Q4's RFC. Q5 defers to §2.4's pre-Gate.
  - The drift between canonical Q4 in `ontology.md` and the narrower variant in `lifecycle-semantics.md` §Open Modeling Question 4 is surfaced in Q4's RFC under Open Questions and is itself a finding for the Q4 redaction phase, not a fix bundled with this entry.
- **Supersession:** none.

---

<!-- DECISION TEMPLATE — copy below this line when recording a decision -->

<!--
## `00NN` — Title

- **Status:** proposed | accepted | superseded | rejected
- **Date:** YYYY-MM-DD
- **Context:**
- **Decision:**
- **Constitutional review:**
- **Consequences:**
- **Supersession:**
-->
