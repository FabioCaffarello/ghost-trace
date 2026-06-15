# RFC — Operational Decision Record (entity-model surface for allow / challenge / block / shadow)

- **Status:** discussion (framing PR; resolution pending committee deliberation + decision-log entry)
- **Authors:** Ghost Trace committee (opened under empirical pressure from the §0221 TLS fingerprint vertical slice)
- **Date:** 2026-06-13
- **Type:** ontology-revision
- **Affects:** [`docs/ontology/entity-model.md`](../../ontology/entity-model.md) (adds an entity-model surface); composes with Charter §2.1, §2.2, §2.3, §2.4, §2.6, §3 N3 (no amendment proposed); [`schemas/events/v1/`](../../../schemas/events/v1/) (a new proto, deferred to the chosen framing)

> RFCs in Ghost Trace are proposals subject to constitutional review. The [Constitutional Charter](../../charter/constitutional-charter.md) is the authoritative reference. RFCs that conflict with the Charter must either be rejected or accompanied by a corresponding charter amendment.

## Summary

The [`§0221`](../../charter/decision-log.md) TLS fingerprint vertical slice carried a single anti-bot observation sub-modality end to end through three constitutional layers — collection (Cat I `NetworkObservation` / `tls_ja4`), inference (Cat III `AutomationGroup` carrying paired confidence + evidential independence, with provenance), and replay (observation→inference reconstruction). It STOPPED at the fourth layer the anti-bot domain needs — a **decision** (allow / challenge / block / shadow) recorded as a substrate event — because **no decision / action / policy record type exists in any category**. `entity-model.md` §Open Modeling Questions reads "(None remaining)"; the [Domain Pack v0.1 RFC](./domain-pack-v0-1-anti-bot-atlas.md) explicitly defers real-time decision serving (F5) and Charter §3 BC (line "§3 does not govern external system behavior") carves *external* decision logic out of §3's scope. Recording a decision *as a substrate event* is therefore a new ontological commitment. This RFC opens that question, enumerates candidate framings, and **does not resolve it** — per [`.claude/CLAUDE.md` §6 rule 2 + §8](../../../.claude/CLAUDE.md), a change that picks an answer is a constitutional move and must be raised explicitly (rule 2 names the `ontology.md` Open Questions; the surface here is the sibling `entity-model.md` §Open Modeling Questions, which rule 2's intent and §8 "when in doubt, stop and ask" both cover).

## Motivation

The Charter §1 thesis describes decisions as "temporally extended sequences of assertions … none superseded by destruction." An anti-bot deployment acts on its inferences: it allows, challenges, blocks, or shadow-flags a client. For the substrate to deliver on the §1 thesis (and the §0221 slice's stated goal of reconstructing observation→inference→**decision** for audit), the decision must be a first-class, immutable, provenance-bearing record — not an unrecorded side effect in an external gateway.

Today there is no home for it:

- It is not a Category I observation of any existing type (the F1 observation protos are network / browser / behavioral / attestation fingerprints, not actions).
- It is not a Category II construct (no operational definition produces it today).
- It is not a Category III hypothesis (a decision is an *output of* inference, not itself an inference about membership).

The cost of *not* deciding this: anti-bot decisions live outside the substrate, the obs→inference→decision chain the §1 thesis promises is unauditable at its final hop, and §3 N3's "commit an audit Cat I record BEFORE the action" has no concrete record to point at for enforcement actions.

## Constitutional Review

Per the `rfc-author` pre-authorship impact analysis (Q1–Q6):

**Q1 — Charter invariants touched.**

- **§2.1 Observational Integrity (frozen).** Any decision record committed to the substrate is immutable after commit; a decision reversal is a *new* record, never a mutation. SATISFIED by construction under all candidate framings.
- **§2.2 Epistemic Separation (frozen).** The crux. A decision has two separable parts: the *policy evaluation* ("confidence X under versioned policy P yields verdict BLOCK") is deterministic given (hypothesis state + versioned policy) — Category II shape; the *action audit* ("BLOCK was enacted at time T") is an observation (record of historical fact) — Category I shape. The framing choice (below) is precisely a §2.2 categorization decision and MUST be made explicitly, not by default.
- **§2.3 Provenance Integrity (frozen v0.4).** A decision references the observations and hypotheses that produced it. If the decision is an Assertion, it carries a typed `subject_ref_*` chain resolving to Cat I roots. SATISFIED if the chosen framing threads `source_event_hashes` / `influenced_by`.
- **§2.4 Inferential Influence Disclosure (frozen v0.5).** If the decision is formed under the influence of a promoted hypothesis (e.g. a promoted `AutomationGroup`), it MUST declare that influence via `influenced_by` at substrate-commit time per [`§0021`](../../charter/decision-log.md) OMQ #3-α. Applies to framings A and B; framing C (external) places the decision outside the substrate, so §2.4 does not reach it.
- **§2.6 Evidential Independence Integrity (frozen v0.6).** If the decision is an inferential assertion, it carries paired `confidence` + `evidential_independence` per the §0136/§0140 marshalling-boundary check. A purely *mechanical* policy application (deterministic Cat II) may instead inherit the upstream hypothesis's dimensions rather than minting new ones — a framing-dependent question.
- **§3 N3 — no autonomous irreversible action (frozen v0.7).** Directly on point. N3 forbids AUTONOMOUS substrate-side irreversible action; it explicitly PERMITS operator-initiated action via audit-on-commit (the `OrphanCleanupAudit` §0104 / CLI §0119 precedent). A decision *record* that an operator (or operator-configured policy) elected is N3-compliant; a system that auto-enacts a block with no audit record is an N3 violation. Any framing MUST preserve audit-on-commit and MUST NOT make the substrate the actor.

**Q2 — Glossary redefinition.** "decision", "action", "policy" are not canonical-vocabulary terms today. This RFC proposes ADDING an entity-model term, not redefining an existing one. The chosen term must clear `vocabulary-discipline` + `anti-marketing` at resolution time (e.g. "operational decision" / "decision audit", avoiding marketing-tell verbs). **One existing collision the resolving entry MUST reconcile:** [`docs/glossary.md`](../../glossary.md) lists `decision` as a *forbidden synonym for `assertion`* ("the Charter treats decisions as temporally extended sequences of assertions, not as one"). Adopting "decision" / "operational decision" as a new entity-model term is therefore not a clean addition — the resolving move must also narrow that glossary entry so it forbids only the single-assertion-synonym sense, not the new entity-model sense. This RFC honors the existing rule (it cites the §1 thesis "temporally extended sequences of assertions" and treats a decision as NOT reducible to one record); the reconciliation is owed at resolution, not in this framing PR.

**Q3 — Resolves an open Ontology question?** No open Ontology questions remain (`entity-model.md` §Open Modeling Questions = none). This RFC OPENS a new one rather than silently resolving an existing one.

**Q4 — Requires Charter amendment?** Assessed NO under all three candidate framings: the three categories and the §3 N3 audit-on-commit mechanism already exist frozen; this is an entity-model (Ontology) addition under ordinary RFC discipline, materialized as a schemas-evolution (additive proto) per [`§0024`](../../charter/decision-log.md) / [`§0139`](../../charter/decision-log.md). Should committee deliberation surface a framing that cannot be expressed without changing a frozen element, the RFC reclassifies to `charter-amendment` at that point.

**Q5 — Introduces a new invariant?** No. It adds a record type governed by existing invariants.

**Q6 — Ceremony without consequence?** No. The proposed record is falsifiable by deletion: without it, the obs→inference→decision chain is unauditable at its final hop and §3 N3 enforcement has no decision artifact to detect.

## Proposal

**Open the operational-decision record as an entity-model question. Do not resolve it in this framing PR.** Three candidate framings are carried for committee deliberation; the evidence file [`operational-decision-record-evidence.md`](../discussion/operational-decision-record-evidence.md) holds the substantive comparison.

- **Framing A — Decision as a Category I audit record.** A new Cat I `OperationalDecision` (or `PolicyDecisionAudit`) proto carrying `decision_type` (ALLOW / CHALLENGE / BLOCK / SHADOW), `subject_actor_ref`, `decided_at`, `source_observation_hashes`, `influencing_hypothesis_hashes`, `policy_ref`, `operator_ref`. Committed via `substrate.AppendPair` alongside the enacted action, mirroring the `OrphanCleanupAudit` ([`§0104`](../../charter/decision-log.md)) audit-on-commit precedent. The decision is framed as an observation (record of historical fact) — "the operator/policy decided X" — which fits Cat I cleanly and discharges §3 N3 directly. Open sub-question: a Cat I record carries no `confidence`/`evidential_independence` (§2.6 BC3), so the decision's relationship to the upstream hypothesis's paired dimensions is by *reference*, not by carrying its own.
- **Framing B — Decision as a Category II construct (+ Cat I audit pairing).** A `DecisionConstruct` derived deterministically from a promoted hypothesis + a versioned policy definition, carrying `influenced_by` to the hypothesis and (per §2.6) inheriting or recomputing paired dimensions; the *enactment* is still a Cat I audit record per §3 N3. Richer provenance/replay (Phase-2 deterministic given the policy version), at the cost of two record types.
- **Framing C — Decision as an external-consumer concern (no substrate record).** Per Charter §3 BC ("§3 does not govern external system behavior"), a read-only policy evaluator consumes the promoted-hypothesis projection and emits the verdict to the calling gateway, writing NO substrate record (or only a §3 N3 audit record if the action is irreversible). No ontology move; but the decision is not a substrate event, so obs→inference→decision replay reconstructs only the inputs, not a committed decision.

For each framing the evidence file evaluates: §2.2 category fit, §2.3/§2.4 provenance + influence threading, §2.6 paired-dimension handling, §3 N3 audit-on-commit, and replay-model Phase-2/3 behavior.

## Alternatives Considered

- **Resolve now in this PR.** Rejected: picks a §2.2 categorization answer unilaterally, violating CLAUDE.md §6.2. The framing PR is the correct first step under the [`§0022`](../../charter/decision-log.md) empirical-pressure methodology.
- **Force the decision into the Cat III hypothesis lifecycle** (e.g. model a block as a demotion side effect). Rejected as a category leak: a decision is not a hypothesis-membership claim; bending the lifecycle to carry it conflates §2.2 categories.
- **Do nothing; decisions are forever external.** This is Framing C made permanent. Carried as a live option (it is the most conservative and matches the Domain Pack F5 deferral), not pre-rejected.

## Open Questions

- Which framing (A / B / C)?
- Single record vs. record pair (evaluation + enactment)?
- How is the versioned policy represented and referenced (`policy_ref` shape; is the policy itself a Cat II definition)?
- Does this couple to F5 (real-time decision serving), or is recording strictly decoupled from serving latency?
- For Framing A, how does a Cat I audit (no paired dimensions per §2.6 BC3) reference the upstream hypothesis's `confidence` / `evidential_independence` for auditability?

## Anti-Patterns to Avoid

- **Autonomous enforcement without an audit record** — a §3 N3 violation; the substrate must never be the actor.
- **Truth-bearing decision record** — a decision marked definitive without declared `influenced_by` + paired `evidential_independence` where the framing makes it an inferential assertion (§3 N1 / §2.6).
- **Mutating an observation to record a decision** — a §2.1 violation; decisions are new records.
- **Silent §2.2 categorization** — choosing Cat I vs Cat II by code default instead of explicit committee resolution.

## Migration and Backward Compatibility

Additive under all framings. No existing record type changes; historical replay is unaffected. A new proto (Framings A/B) is a backward-compatible schemas-evolution per [`§0139`](../../charter/decision-log.md) / [`§0140`](../../charter/decision-log.md). Framing C adds no substrate surface at all.

## References

- [`decision-log §0221`](../../charter/decision-log.md) — TLS fingerprint vertical slice; the empirical pressure that surfaced this question.
- [`domain-pack-v0-1-anti-bot-atlas.md`](./domain-pack-v0-1-anti-bot-atlas.md) — F5 (real-time decision serving) deferral.
- [Charter §3 N3](../../charter/constitutional-charter.md#3-non-goals) — no autonomous irreversible action; audit-on-commit; external-system carve-out.
- [`decision-log §0104`](../../charter/decision-log.md) — `OrphanCleanupAudit` audit-on-commit precedent.
- [`entity-model.md`](../../ontology/entity-model.md) — the three categories + Assertion typed references.

## Decision Record

If accepted (a framing chosen), this RFC is recorded in [`../../charter/decision-log.md`](../../charter/decision-log.md) with a decision number resolving the framing. The framing PR itself is registered at [`§0221`](../../charter/decision-log.md) as the opening of the question.
