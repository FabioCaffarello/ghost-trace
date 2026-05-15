---
name: ambiguity-reducer
description: Flag semantically broad terms (identity, state, trust, source, causality, event, record, signal, context, behavior, evidence, decision) that depend on implicit context and have not been operationalized in this project. Use this skill ALWAYS when writing prose in docs/, ALWAYS when drafting an RFC, and ALWAYS when a commit introduces new terminology. This is preventive — it catches drift before it enters the substrate.
---

# ambiguity-reducer

Ambiguous prose hardens into ambiguous structure. This skill flags terms whose meaning in this project is not yet operationalized — terms that look meaningful but resolve only against implicit context that the document does not supply.

The skill is preventive. By the time an ambiguous term reaches a schema or an invariant, the drift is already structural.

This skill is advisory by design (per [`.claude/CLAUDE.md` §5.3](../../../CLAUDE.md)). A flag here is information; the writer chooses one of the three response patterns in §2. The blocking checks for forbidden vocabulary live in `vocabulary-discipline` and the pre-commit hook.

## 1. Watchlist

Extended over time as new high-risk terms are identified. Never trimmed without justification recorded in the decision log.

For each term: why it is risky in this project, and what specific operationalization is required when the term appears.

### `identity`

**Risk:** meaningful only against the three-tier model (`ActorRef`, `Identity`, `Cluster`) introduced in [`entity-model.md` §Open Modeling Questions](../../../../docs/ontology/entity-model.md). The model is itself an open question — using `identity` without specifying which tier silently resolves the question.

**On flag:** require the writer to (a) specify the tier explicitly, or (b) acknowledge the term operates above the tiers and motivate the abstraction.

### `state`

**Risk:** meaningful only against a specific entity's lifecycle. The three categories have very different lifecycle rules ([Charter §2.1](../../../../docs/charter/constitutional-charter.md#21-observational-integrity); [§2.2](../../../../docs/charter/constitutional-charter.md#22-epistemic-separation); §2.5 pending; [`lifecycle-semantics.md`](../../../../docs/ontology/lifecycle-semantics.md)).

**On flag:** require the writer to specify which entity's state and which lifecycle phase.

### `trust`

**Risk:** no canonical definition in this project. The Charter does not define `trust` and does not commit to a model of it.

**On flag:** demand replacement with `provenance`, `independence`, `confidence`, or another canonical term. If none fits, the claim is rejected as non-falsifiable (see `falsifiability-check`).

### `source`

**Risk:** collides with `provenance`. Common informal use covers both data origin and version-control origin; `provenance` is the canonical term for the first.

**On flag:** force the writer to pick `provenance` (if observational or inferential lineage is meant) or another specific term.

### `causality`

**Risk:** undefined in the Charter and the Ontology. Causal language imports philosophical commitments the project has not made.

**On flag:** force a rewrite as `derivation` (if deterministic, per Category II) or `influence` (if inferential, per pending §2.4) — or another canonical relation.

### `event`

**Risk:** overloaded between "observation" (Category I) and "any log entry" (which under pending §2.5 also includes hypothesis-lifecycle events).

**On flag:** demand `observation` (if Category I), `lifecycle event` (if a hypothesis-state transition), or another specific term.

### `record`

**Risk:** overloaded across all three categories.

**On flag:** demand the specific category — `observation`, `operational construct`, `hypothesis`, or `assertion`.

### `signal`

**Risk:** used informally. Often means `operational construct` but can also mean an intermediate computation result with no commitment to commit.

**On flag:** force the writer to choose `operational construct` (committed to substrate, governed by Category II) or `intermediate value` (transient, projection-only).

### `context`

**Risk:** almost always under-operationalized. "Operational context", "behavioral context", "session context" — each has a different referent.

**On flag:** require the writer to specify (a) what specific information, (b) sourced from where, (c) as of when.

### `behavior`

**Risk:** the project's central object, but not a defined entity. The Thesis treats `behavioral intelligence` as the domain, not as a primitive ([Charter §1](../../../../docs/charter/constitutional-charter.md#1-thesis)).

**On flag:** allow as a direct reference to the Thesis (`behavioral intelligence`, `behavioral telemetry`). Otherwise demand specification — `sequence of observations`, `pattern of assertions`, etc.

### `evidence`

**Risk:** ambiguous between "the observations supporting an assertion" and "the substrate's structural reference to those observations".

**On flag:** demand `observations` (the records themselves) or `observational provenance` ([`provenance-model.md`](../../../../docs/ontology/provenance-model.md)).

### `decision`

**Risk:** confused with `assertion`. The Charter treats decisions as "temporally extended sequences of assertions, each reflecting the best available understanding at the moment it was made" ([Charter §1](../../../../docs/charter/constitutional-charter.md#1-thesis)), not as atomic events.

**On flag:** specify whether the writer means an individual `assertion`, a sequence of assertions, or a non-system actor's action.

### `conflict`

**Risk:** Semantic incompatibility between two documents or claims. Undefined in the Charter, glossary, or skills as of Gate 1; surfaced in §4 Step 1.2 falsifiability analysis of the precedence rule (the "conflict" condition in the working stub's Bullet 3 lacked operationalization, contributing to Bullet 3's removal from binding §4).

**On flag:** require the writer to (a) operationalize locally (state which artifact's contradiction with which other artifact constitutes the conflict, by what detection procedure), (b) replace with a more specific term (e.g., `cross-reference broken`, `claim contradiction`, `version mismatch`), or (c) raise as an open modeling question.

Watchlist addition authorized by [`decision-log.md` §0008](../../../../docs/charter/decision-log.md) (Gate 2 carry-forward from §0007).

## 2. Response patterns

When the skill flags a term, the writer chooses one of three responses. Doing nothing is not a response.

### Response 1 — Replace

Substitute a canonical term that fits. Example: `record` (when meaning Category III) → `hypothesis`.

### Response 2 — Operationalize locally

Define the term in the same document, naming which substrate artifact or projection it refers to.

Example: "In this section, `context` refers to the enrichment record paired with the observation by reference, as of the enrichment's commit time."

A local operationalization is binding within its document. Future use of the term outside the document is flagged again — the skill does not learn across documents.

### Response 3 — Raise as an open modeling question

If neither replacement nor local operationalization is possible, the term names an unresolved modeling question. Raise it explicitly in the relevant Ontology document, in its "Open Modeling Questions" section.

This is the legitimate path. Silent use of an ambiguous term in lieu of the question is the drift this skill exists to prevent.

## 3. Adding to the watchlist

When a new high-risk term surfaces in review:

1. Verify the term is genuinely ambiguous in this project — not merely unfamiliar to one reader. A term with a canonical Charter or Ontology definition is not added.
2. Identify the specific risk (overloading, undefined in this project, philosophical import).
3. Specify the operationalization required at the moment of use.
4. Add the term to this skill's watchlist in a commit dedicated to the addition. The commit message names the term, the document where it surfaced, and the risk.
5. If the term affects an already-redacted document, reference the watchlist update in [`docs/charter/decision-log.md`](../../../../docs/charter/decision-log.md).

The watchlist grows. It is not trimmed without justification.

## 4. What this skill does not do

This skill does not block work. A term flagged here is informational. The writer decides.

The blocking checks for forbidden vocabulary live in `vocabulary-discipline` (Phase 4) and the pre-commit hook (`pre-commit-doc-check.sh`, Phase 7). This skill catches terms that may yet have a legitimate use.

## 5. Source citations used

- [`docs/charter/constitutional-charter.md` §1 Thesis](../../../../docs/charter/constitutional-charter.md#1-thesis)
- [`docs/charter/constitutional-charter.md` §2.1 Observational Integrity](../../../../docs/charter/constitutional-charter.md#21-observational-integrity)
- [`docs/charter/constitutional-charter.md` §2.2 Epistemic Separation](../../../../docs/charter/constitutional-charter.md#22-epistemic-separation)
- [`docs/ontology/entity-model.md` §Open Modeling Questions](../../../../docs/ontology/entity-model.md)
- [`docs/ontology/provenance-model.md`](../../../../docs/ontology/provenance-model.md)
- [`docs/ontology/lifecycle-semantics.md`](../../../../docs/ontology/lifecycle-semantics.md)
- [`.claude/CLAUDE.md` §3 Canonical vocabulary, §5.3 enforcement posture](../../../CLAUDE.md)
