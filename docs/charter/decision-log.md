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
