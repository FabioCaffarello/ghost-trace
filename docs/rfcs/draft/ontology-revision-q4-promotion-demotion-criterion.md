# RFC — Ontology Open Question 4: Promotion → Demotion Criterion

- **Status:** discussion
- **Authors:** Ghost Trace committee
- **Date:** 2026-05-15
- **Type:** ontology-revision
- **Affects:** [`docs/ontology/lifecycle-semantics.md`](../../ontology/lifecycle-semantics.md) (Promotion Mechanism section); [Charter §2.5](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) (pending)

> RFCs in Ghost Trace are proposals subject to constitutional review. The [Constitutional Charter](../../charter/constitutional-charter.md) is the authoritative reference. RFCs that conflict with the Charter must either be rejected or accompanied by a corresponding charter amendment (see [`../../charter/amendments.md`](../../charter/amendments.md)).

## Summary

Ontology Open Question 4, recorded verbatim in [`docs/ontology/ontology.md` §Open Questions for Committee Resolution](../../ontology/ontology.md): *"When does a promoted hypothesis become a candidate for demotion? Lifecycle rule."* This RFC opens structured discussion of Q4 with three candidate criterion families and two explicitly rejected alternatives. §2.5 binding text must either codify a criterion for promotion-to-demotion candidacy or explicitly defer it; redacting §2.5 without prior resolution leaves demotion either trivially available (permitting frivolous demotion) or implicitly conditioned on unstated rules (silent resolution by future implementation). The RFC does not pick a candidate.

## Motivation

The §2.5 stub names demotion as one of six lifecycle operations on hypotheses (formation, merge, split, dissolution, promotion, demotion). Demotion is the operation that withdraws a previously promoted hypothesis from operational use as enrichment context. Without a criterion for when a promoted hypothesis becomes a candidate for demotion, three failure modes appear:

- **Demotion always available.** §2.5 records demotion as a lifecycle event but says nothing about when it may be invoked. The operation becomes operationally meaningless — any operator may demote any promoted hypothesis at any time, without structural defense against frivolous demotion.
- **Implicit conditioning on unstated rules.** Implementation code accumulates thresholds (confidence drops, time since promotion, observed counter-evidence) without RFC-level deliberation. The criteria become infrastructure-resolved rather than committee-resolved — the precise failure mode [`ontology-keeper`](../../../.claude/skills/ontology/ontology-keeper/SKILL.md) exists to prevent.
- **Unfalsifiable prose at the Charter level.** [`lifecycle-semantics.md` §Promotion Mechanism step 4](../../ontology/lifecycle-semantics.md) currently states that promoted hypotheses are subject to *"periodic re-evaluation against fresh, independent evidence."* The phrase has no cadence, no threshold, no operationalization. Under [Charter §4](../../charter/constitutional-charter.md#4-constitutional-design-rule) (frozen, v0.2), a constitutional claim that cannot, in principle, be violated, observed, or audited is not admissible. Any §2.5 binding text inheriting this prose would fail the [`falsifiability-check`](../../../.claude/skills/epistemic/falsifiability-check/SKILL.md) discipline applied to constitutional claims.

The cost of not resolving Q4 before §2.5 redaction is therefore concrete: §2.5 binding text either silently picks a demotion criterion or fails §4 falsifiability discipline.

## Constitutional Review

The Q1–Q6 impact analysis prescribed by [`rfc-author` §1](../../../.claude/skills/workflow/rfc-author/SKILL.md).

### Q1 — Which Charter invariants does this RFC touch?

- **§2.5 Hypothesis Lifecycle Explicitness** (pending): touched directly. The demotion operation is one of six lifecycle operations §2.5 codifies.
- **§2.6 Evidential Independence Integrity** (pending): touched indirectly. Candidate B (evidence-staleness-based) presupposes a stable measure of evidential independence, which is the dimension §2.6 introduces. Candidate B's operationalization depends partly on the formal definition of independence (ontology.md Open Question 3), which is itself pending.
- **§2.4 Inferential Influence Disclosure** (pending): touched indirectly. Candidate C (influence-saturation-based) presupposes a structural representation of declared influence, which is what §2.4 mandates.
- **§4 Constitutional Design Rule** (frozen, v0.2): consistency check, not amendment. The four falsifiability criteria (violation, observation, operationalization, non-circularity) are applied to each candidate criterion in this RFC's drafting; specific candidate evaluations are deferred to the discussion phase.
- No FROZEN invariant is amended.

### Q2 — Does this RFC implicitly redefine any term in the glossary?

The terms `hypothesis`, `enrichment`, `supersession`, and `influence` are in [`docs/glossary.md`](../../glossary.md) and [`.claude/CLAUDE.md` §3 canonical vocabulary](../../../.claude/CLAUDE.md). All candidates preserve these definitions.

The terms `promotion` and `demotion` appear in [`lifecycle-semantics.md`](../../ontology/lifecycle-semantics.md) and the [§2.5 stub](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) but are not glossary entries. Whether they should be added to the glossary as a consequence of Q4's resolution is a separate question, not addressed here.

The term `independence` is in canonical vocabulary as a component of `evidential independence`. Candidate B's use of "evidence-staleness" relative to "independence-bearing assertions" must not redefine independence; this RFC treats independence as the structural property §2.6 introduces and ontology.md Q3 will formalize.

### Q3 — Does this RFC implicitly resolve any of the five open Ontology questions?

This RFC opens structured discussion of Open Question 4. By construction it touches Q4; it does not resolve Q4.

The RFC touches ontology.md Q3 (formal definition of independence) under Candidate B and ontology.md Q5 (influence propagation) under Candidate C. Neither is resolved here: both are referenced as dependencies whose own resolution would constrain the operational form of the candidate criterion if that candidate is the resolution. The dependencies are recorded under Open Questions below.

The RFC touches ontology.md Q2 (hypothesis subtypes) indirectly: if Q2 resolves as Candidate A (distinct types), Q4's resolution may carry subtype-specific parameters; if Q2 resolves as Candidate B (single type with tags), Q4's resolution carries uniform parameters. The interaction is recorded.

### Q4 — Does this RFC require Charter amendment?

No. §2.5 is pending. Codifying the criterion within §2.5 binding text is a redaction act, not an amendment. As with [the Q2 RFC](./ontology-revision-q2-hypothesis-subtypes.md), redaction of §2.5 before Q4 resolves would either silently resolve Q4 or produce unfalsifiable §2.5 binding text — both forbidden.

### Q5 — Does this RFC introduce a new invariant?

No. The RFC codifies criteria within an existing pending invariant (§2.5). No new constitutional claim is added.

### Q6 — Does this RFC propose ceremony without behavioral consequence?

No. Different criteria produce different lifecycle decisions at runtime. A `BehavioralCluster` promoted today may become a demotion candidate at month +3 under Candidate A (time-based) but not under Candidate B (evidence-staleness-based) if independent fresh evidence continues to arrive. The candidates partition the runtime space of demotion decisions differently. Deleting this RFC would either force silent resolution (Candidate B by default if implementation chooses convenience) or produce un-redactable §2.5 binding text.

## Proposal

Three candidate criterion families, each presented with structural claim, dependency on other pending invariants, and pros and cons. The RFC does not pick. Combined criteria — applying any of A, B, or C; or applying all three; or applying them as gated stages — are an explicit option for the discussion phase.

### Candidate A — Time-based

**Structural claim.** A promoted hypothesis becomes a candidate for demotion after N units of operational time since promotion, where N is a structural parameter recorded as part of the promotion event. If [Q2](./ontology-revision-q2-hypothesis-subtypes.md) resolves as Candidate A (distinct subtypes), N may be subtype-specific; if Q2 resolves as Candidate B (single type with discriminator labels), N is uniform or discriminator-keyed.

**Dependency.** None on pending invariants other than §2.5 itself. The promotion event already records its own timestamp; demotion candidacy is a structural function of the elapsed-time field.

**Pros.**

- Simplest to operationalize. The criterion is computable against substrate timestamps without reference to projections or to other pending invariants.
- The cadence question raised by `lifecycle-semantics.md` §Promotion Mechanism step 4 ("periodic re-evaluation") is answered concretely: cadence is N, recorded structurally per promotion event.
- Falsifiability is direct: a promoted hypothesis that crosses N units without entering the demotion-candidate set is a structural anomaly detectable on the substrate.

**Cons.**

- Demotes still-valid hypotheses on a timer. A `CampaignHypothesis` whose evidence remains fresh at time N + 1 is still a demotion candidate under this criterion, even though no epistemic case for demotion exists.
- Retains invalid hypotheses until the timer fires. A `BehavioralCluster` whose evidence is shown to be stale at time N − 1 is not a demotion candidate under this criterion.
- N is a parameter, not a property of the substrate. Its committee-defensible value is unclear.

### Candidate B — Evidence-staleness-based

**Structural claim.** A promoted hypothesis becomes a candidate for demotion when the ratio of new evidence to its influence-bearing assertions falls below a threshold T. The criterion tracks the freshness of evidence supporting the hypothesis against the cumulative weight of assertions formed under its influence.

**Dependency.** Depends on a stable structural representation of independence (ontology.md Q3, pending) and on §2.6 (Evidential Independence Integrity, pending). The "fresh evidence" component requires the substrate to distinguish independent assertions from inherited-belief assertions, which is the discipline §2.6 is being redacted to enforce. Candidate B is therefore circular with §2.6 unless either §2.6 is redacted first (out of order with [decision-log §0008](../../charter/decision-log.md)) or Q3 is resolved as a pre-§2.6 dependency in the same spirit as this RFC.

**Pros.**

- Tracks epistemic freshness rather than wall-clock time. A hypothesis whose supporting evidence remains fresh is not demoted; a hypothesis whose supporting evidence is exhausted is.
- Aligns with the original `lifecycle-semantics.md` §Promotion Mechanism step 4 intent ("fresh, independent evidence").
- The structural property the criterion measures (independence ratio) is a property §2.6 is being redacted to enforce; this candidate uses §2.6's structural surface productively.

**Cons.**

- Cannot be operationalized until §2.6 is redacted and ontology.md Q3 is resolved. The criterion's binding text is currently un-falsifiable without these prerequisites.
- Computing the ratio requires traversal of provenance edges across the §2.3 surface, which is also pending.
- Threshold T is a parameter with the same committee-defensibility problem as Candidate A's N.

### Candidate C — Influence-saturation-based

**Structural claim.** A promoted hypothesis becomes a candidate for demotion when its declared influence on derived assertions exceeds a structural ratio K of all assertions in its scope. The criterion defends against runaway influence: a hypothesis whose influence has saturated the assertion stream of its scope becomes a candidate for review, on the grounds that further conclusions under its influence are no longer independent confirmations.

**Dependency.** Depends on §2.4 (Inferential Influence Disclosure, pending). The criterion presupposes that every assertion formed under a hypothesis's influence carries a structural declaration of that influence, which is exactly what §2.4 mandates.

**Pros.**

- Directly addresses the [Charter §1 Thesis](../../charter/constitutional-charter.md#1-thesis) failure mode of recursive belief inflation. Saturation is the operational expression of recursive belief.
- Computable against the substrate once §2.4 is redacted; no further pending dependencies.
- Threshold K is a structural property of the substrate (a ratio over typed records), more committee-defensible than wall-clock N.

**Cons.**

- Requires §2.4 to be redacted first or in parallel. Under [decision-log §0008](../../charter/decision-log.md) §2.4 is redacted after §2.3, after §2.5. Operationalization of Candidate C is therefore delayed.
- "Scope" of a hypothesis is itself undefined here. Whether scope is a structural property recorded at promotion or computed at evaluation time is open.
- May fail to demote a hypothesis whose influence is concentrated but whose evidence has become stale (Candidate B's failure mode covers this).

### Combined criteria

The three candidates may be combined. Three combination forms are explicit options for the discussion phase:

- **Disjunctive.** A promoted hypothesis becomes a candidate when any of A, B, or C triggers. Maximum coverage; minimum specificity.
- **Conjunctive.** A promoted hypothesis becomes a candidate only when all of A, B, and C trigger. Minimum coverage; protects against single-criterion false positives.
- **Staged.** Candidate A is the initial trigger (operationally simple); Candidate B and Candidate C are evaluated only after Candidate A has fired, narrowing the candidate set. Compromises coverage for operational simplicity.

The RFC does not recommend any combination form. The discussion phase considers them on equal footing with the three single-criterion forms.

## Alternatives Considered

### No criterion — demotion always available (REJECTED)

A promoted hypothesis is always a candidate for demotion; the demotion operation is invokable at any time. Rejected because it makes §2.5's demotion operation operationally meaningless — the §2.5 binding text would codify a non-event. The Charter's structural-enforceability criterion (§2 criterion 1) requires that the operation's structure carry constraints, not merely its existence.

### Demotion only by explicit committee decision (REJECTED)

A promoted hypothesis becomes a candidate for demotion only when a human committee records a decision to demote. Rejected because it is not falsifiable in a structural sense — the criterion depends on subjective committee judgment, not on a detectable property of the substrate. Fails [§4 falsifiability discipline](../../charter/constitutional-charter.md#4-constitutional-design-rule) and §2 criterion 4 (independence of operator interpretation).

### Threshold-form with implementation-deferred parameter values (META-PATTERN)

Rather than picking among A, B, or C, the resolution commits to "explicit threshold form" while deferring the specific parameter values (N, T, or K) to a future RFC tied to §2.6's redaction and Q3's resolution. The discussion phase may consider this as a meta-pattern alongside the three single-criterion candidates. This RFC does not recommend it; surfacing it here as one of the alternatives is procedural transparency, not a pre-decision. If discussion converges on the meta-pattern, the convergence is itself the resolution and the parameter values become a follow-on dependency.

## Open Questions

This RFC explicitly defers the following:

- **Whether the criterion is purely structural or partly procedural.** The three single-criterion candidates are structural (computable on the substrate). The meta-pattern above is partly procedural (commits to form, defers parameters to procedure). Whether the resolution must be one or may be the other is open.
- **Whether demotion is reversible.** A demoted hypothesis may, in principle, be re-promoted under a future promotion event if conditions change. Alternatively, demotion may be a terminal lifecycle transition with dissolution as the only subsequent option. The §2.5 stub does not pre-commit; this RFC does not resolve.
- **Interaction with §2.6 (Evidential Independence Integrity).** Candidate B presupposes §2.6's structural surface. The order of redaction in [decision-log §0008](../../charter/decision-log.md) is §2.5 → §2.3 → §2.4 → §2.6; Candidate B's operational form depends on the later invariant. Whether the §2.5 binding text under Candidate B should encode the dependency as a forward reference is open.
- **Interaction with §2.4 (Inferential Influence Disclosure).** Candidate C presupposes §2.4's structural surface. The same forward-reference question applies.
- **Interaction with the Q2 RFC.** If [Q2](./ontology-revision-q2-hypothesis-subtypes.md) resolves as Candidate A (distinct subtypes), Q4's resolution may carry subtype-specific parameters. The two RFCs are not strictly sequential — they may be resolved in parallel — but the redaction phase of Q4 should follow Q2's redaction or explicitly state independence.
- **Drift between canonical Q4 form and `lifecycle-semantics.md` Open Modeling Question 4.** The canonical Q4 in `ontology.md` asks the general question: when does a promoted hypothesis become a candidate for demotion? The `lifecycle-semantics.md` Open Modeling Question 4 frames a narrower variant that uses the term "independence" as the measurement basis and presents a binary alternative — automatic demotion when independence falls below a threshold, versus operator-driven demotion only. The narrower variant pre-commits implicitly to Candidate B's criterion family by naming independence as the measurement basis. The narrower variant also uses a non-canonical noun for the independence measurement: its rewrite instruction in [`vocabulary-discipline` §4](../../../.claude/skills/ontology/vocabulary-discipline/SKILL.md) picks one of `confidence` or `evidential independence` (collapsing the two dimensions is the failure mode §2.6 is being redacted to prevent). The drift in both question scope and vocabulary is surfaced here and is itself a finding for the discussion phase; the lifecycle-semantics.md text was authored before the relevant watchlist entries were enacted and was not caught by the pre-commit hook because the hook scans only added or changed lines.

## Anti-Patterns to Avoid

- **Resolving Q4 by implementation.** Hard-coded thresholds in services without RFC. The criterion belongs to the Ontology and to §2.5 binding text, not to operational code.
- **Treating demotion as a deletion operation.** Per §2.5 (working text), all lifecycle operations are recorded as immutable events; demotion is a lifecycle transition, not a removal. The demoted hypothesis's prior promotion event remains in the primary event log, and the demotion is itself a new event referencing the promotion.
- **Cascade demotion.** Demoting all hypotheses that depend on a demoted one without independent evaluation. Each demotion candidacy is evaluated against the criterion on its own terms; transitive demotion would silently violate the independence discipline §2.6 is being redacted to enforce.
- **Conflating demotion with dissolution.** Demotion withdraws operational use; dissolution recognizes the hypothesis as no longer corresponding to any underlying phenomenon. The two operations are distinct in the §2.5 stub and remain distinct under all candidates.

## Migration and Backward Compatibility

No historical Category III records exist at this point. The RFC is forward-looking.

Lock-in asymmetry across candidates:

- Candidate A (time-based) is the easiest to soften later: relaxing N or supplementing with B/C is a typed extension of the criterion record.
- Candidate B (evidence-staleness-based) carries the most coupling to other pending invariants (§2.6, ontology.md Q3) and is the most expensive to retrofit if its prerequisites resolve differently than anticipated.
- Candidate C (influence-saturation-based) is the most defensible against the Charter's central concern (recursive belief inflation) but cannot be operationalized until §2.4 is redacted.

The [Phase 3 / Phase 4 replay contracts](../../architecture/replay-model.md) require that any lifecycle event, including a demotion, is replayable. All three candidates satisfy this: demotion events are immutable records in the primary event log per §2.5 (working text), and their causes are computable from the substrate state at the time of the demotion event.

## References

- [`docs/ontology/ontology.md` §Open Questions for Committee Resolution](../../ontology/ontology.md) — canonical statement of Q4.
- [`docs/ontology/lifecycle-semantics.md` §Promotion Mechanism](../../ontology/lifecycle-semantics.md) — current scaffold whose redaction depends on Q4; §Open Modeling Question 4 (narrower-variant drift).
- [`docs/charter/constitutional-charter.md` §2.5](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) — stub whose redaction depends on Q4 resolution.
- [`docs/charter/constitutional-charter.md` §2.6](../../charter/constitutional-charter.md#26-evidential-independence-integrity) — pending; possible measurement basis for Candidate B.
- [`docs/charter/constitutional-charter.md` §2.4](../../charter/constitutional-charter.md#24-inferential-influence-disclosure) — pending; possible measurement basis for Candidate C.
- [`docs/charter/constitutional-charter.md` §4](../../charter/constitutional-charter.md#4-constitutional-design-rule) — frozen; falsifiability discipline applied to candidate criteria.
- [`docs/charter/decision-log.md` §0008](../../charter/decision-log.md) — redaction order §2.5 → §2.3 → §2.4 → §2.6.
- [`docs/charter/decision-log.md` §0009](../../charter/decision-log.md) — §2.5 redaction plan registering this RFC as a pre-§2.5 dependency.
- [`docs/rfcs/draft/ontology-revision-q2-hypothesis-subtypes.md`](./ontology-revision-q2-hypothesis-subtypes.md) — companion pre-§2.5 RFC; interaction noted under Open Questions.
- [`.claude/skills/ontology/ontology-keeper/SKILL.md`](../../../.claude/skills/ontology/ontology-keeper/SKILL.md) — registry of the five open Ontology questions.

## Decision Record

Pending. A decision-log entry will be assigned when this RFC is resolved. Current status is `discussion`.
