# §3 Non-Goals — committee draft

**Status:** in-committee draft. Not binding. Do not cite as authoritative.

## Anchor

> **Status:** Pending committee redaction.
>
> Non-Goals are not a defensive appendix. They are the negative perimeter of the system's identity — direction explicitly rejected, not merely deprioritized. This section will receive committee treatment equal to the invariants.
>
> **Anticipated non-goals (non-binding):**
>
> - Ghost Trace does not produce truth. It maintains the distinction between observation, inference, and inherited belief.
> - Ghost Trace does not perform universal identity resolution. Identity reconciliation is a domain-specific concern subordinate to the substrate.
> - Ghost Trace does not automate irreversible operational action. Actions of consequence are taken by external systems with their own accountability.
> - Ghost Trace does not optimize for the lowest operational complexity. Where simplicity conflicts with epistemic integrity, integrity prevails.
> - Ghost Trace is not a generic event-sourcing framework. Its specificity to behavioral intelligence is constitutional, not incidental.
>
> — [Charter §3](../constitutional-charter.md#3-non-goals)

## Ontology-derived material

Ten inheritance sources contribute material the §3 stub predates. Each anticipated non-goal anchors to one or more frozen §2.x invariants or to a decision-log entry that operationalized its negative perimeter.

- [`decision-log.md` §0099 + §0129](../decision-log.md) + [Charter §2.4 frozen v0.5](../constitutional-charter.md#24-inferential-influence-disclosure) + [Charter §2.6 frozen v0.6](../constitutional-charter.md#26-evidential-independence-integrity) — anchor the first anticipated non-goal (no truth); §2.4's `influenced_by` chain + §2.6's paired-dimension structural enforcement together operationalize the observation/inference/inherited-belief distinction.
- [`decision-log.md` §0023](../decision-log.md) + [`entity-model.md` Open Modeling Question 1](../../ontology/entity-model.md) — anchor the second anticipated non-goal (no universal identity resolution); inception-phase single-tier `actor_ref` operationalizes domain-specific identity; multi-tier formalization deferred per Q2 forward-reference.
- [Charter §2.1 frozen](../constitutional-charter.md#21-observational-integrity) + [`decision-log.md` §0104 + §0119](../decision-log.md) — anchor the third anticipated non-goal (no automated irreversible action); §2.1 substrate-immutability + §0104 HTTP T3 audit-on-commit + §0119 CLI opt-in audit symmetry together codify the audit-before-act discipline.
- [Charter §4 frozen v0.2](../constitutional-charter.md#4-constitutional-design-rule) + [`decision-log.md` §0033 + §0119](../decision-log.md) — anchor the fourth anticipated non-goal (epistemic discipline over simplicity); §4 structural-enforceability discipline + §0033 local-shell-trust default preserved at §0119 operationalize the simplicity-vs-discipline boundary.
- [Charter §2.2 frozen](../constitutional-charter.md#22-epistemic-separation) + [`decision-log.md` §0010](../decision-log.md) + [`entity-model.md` Category III post-Q2](../../ontology/entity-model.md) — anchor the fifth anticipated non-goal (not generic event-sourcing); §2.2's three-category typing + §0010 Q2-A.2 four-subtype Cat III taxonomy together codify behavioral-intelligence specificity.
- [`decision-log.md` §0077 + §0121](../decision-log.md) — empirical-pressure assessments for §3 (§0077 paired §2.4/§2.6 + extended to §3 at §0121); five Q-§3-* questions + three cross-cutting questions catalogued.
- [Charter §1 Thesis (frozen)](../constitutional-charter.md#1-thesis) — anchor for the negative-perimeter framing; §1 names the failure modes Ghost Trace exists to prevent; §3 names the inverse: directions Ghost Trace explicitly rejects in pursuit of that prevention.

## Multi-carry-forward status

§3 Step 1.1 Part D assesses three carry-forwards. Aggregate verdict: **no cascade triggers fire; all three accommodate forward-reference contract continuation, clean structural inheritance, or are pre-resolved by upstream committee action.**

### Q-X1 (§3 / §4 overlap) — clean partition

[`§0121`](../decision-log.md) Q-X1 raised potential overlap between §3 N4 ("does not optimize for lowest operational complexity") and §4 ("structural-enforceability discipline"). Resolution: §3 governs the system's identity perimeter (categorical rejection of directions); §4 governs the constitutional-claim admissibility discipline (falsifiability + qualification criteria). The two are complementary, not redundant: §3 names what the system rejects; §4 names what the system requires to admit a claim as constitutional. §3 N4's binding text articulates the partition: simplicity-vs-discipline is the §3 perimeter; the discipline test itself is §4 territory.

**Verdict: clean partition; no cascade trigger.**

### Q-X2 (§3 vs §2.6 redaction order) — discharged by §2.6 closure

[`§0121`](../decision-log.md) Q-X2 raised the redaction-order question. With §2.6 closed at [`§0129`](../decision-log.md) on 2026-05-22 (PR #120), §3 follows §2.6 by default. The Q-X2 question is structurally discharged.

**Verdict: discharged by upstream closure; no cascade trigger.**

### §2.x BC mutual scope inheritance — clean structural

§3 inherits the cross-section mutual scope statements established at §2.3 BC1 (observational provenance), §2.4 BC1 (inferential influence), and §2.6 BC1 (paired dimension). §3 binding text encodes its scope as the identity perimeter; §2.x govern specific structural commitments; §4 governs the constitutional-claim discipline. Clean structural partition across all sections.

**Verdict: clean inheritance; no cascade trigger.**

## Cascade-trigger status

**No cascade triggers fire at §3 Step 1.1.** Aggregate Part D verdicts: Q-X1 clean partition; Q-X2 discharged by §2.6 closure; §2.x BC mutual scope clean inheritance.

§3 Step 1.1 demonstrates the most compact pre-Gate dependency surface of any §-section redaction (three carry-forwards vs §2.4's four or §2.6's six). The compactness reflects §3's positional advantage: as the LAST pending Charter section, §3 inherits resolved structural surfaces from every prior §2.x redaction; no cross-section dependencies remain open.

---

## Definition

§3 codifies the **negative perimeter of Ghost Trace's structural identity**: five categorical directions the system explicitly rejects. The non-goals are not deprioritized features — they are rejected by construction. A change set that would move the system toward any of N1–N5 is rejected on procedural grounds equivalent to a §2.x invariant violation per [§4](../constitutional-charter.md#4-constitutional-design-rule) qualification criteria.

The five non-goals together circumscribe the system's identity by exclusion: Ghost Trace is the system that maintains observation/inference/inherited-belief distinction (rejecting N1 truth-production); the system that does not pretend to globally reconcile identity across domains (rejecting N2 universal identity resolution); the system that records before acting (rejecting N3 autonomous irreversible action); the system that pays the structural-discipline cost (rejecting N4 simplicity-over-discipline); the system whose specificity to behavioral-domain is constitutional (rejecting N5 generic event-sourcing identity).

## Structural Requirement

Each non-goal is operationalized through structural anchors at frozen §2.x invariants or decision-log entries. A change set that proposes moving the system toward any non-goal is structurally detectable via the named anchor:

**N1 — Ghost Trace does not produce truth.** Operationalized via [§2.4 frozen v0.5](../constitutional-charter.md#24-inferential-influence-disclosure) (`influenced_by` chain declaration) + [§2.6 frozen v0.6](../constitutional-charter.md#26-evidential-independence-integrity) (paired `confidence` + `evidential_independence` structural enforcement). A change set that proposes a substrate-committed record carrying definitive-truth semantics — i.e., an inferential record marked as "true" without declared influence chain or paired independence dimension — is detectable at the canonical-serialization-contract layer per [§0034](../decision-log.md).

**N2 — Ghost Trace does not perform universal identity resolution.** Operationalized via [`§0023`](../decision-log.md) inception-phase single-tier `actor_ref` resolution + the Q2 (Identity tiers) Open Modeling Question forward-reference per [`entity-model.md` Open Modeling Question 1](../../ontology/entity-model.md#open-modeling-questions). Identity reconciliation is domain-specific, anchored at the producer layer; cross-domain identity-graph at the substrate would require formal Q2 resolution AND constitutional review of the substrate-side identity surface. Detectable by substrate query: a substrate-side identity-graph record would be a new typed Cat I record requiring §0024 canonical-serialization-contract evolution + RFC review.

**N3 — Ghost Trace does not automate irreversible operational action.** Operationalized via [§2.1 frozen](../constitutional-charter.md#21-observational-integrity) substrate-immutability + [`§0104`](../decision-log.md) (HTTP T3 audit-on-commit via `OrphanCleanupAudit`) + [`§0119`](../decision-log.md) (CLI opt-in audit symmetry). Actions of consequence — orphan-cleanup deletion, future state-mutating operations — commit an audit Cat I record BEFORE the action. Detectable via substrate replay: an irreversible action whose audit record is absent at substrate is a structural anomaly.

**N4 — Ghost Trace does not optimize for the lowest operational complexity.** Operationalized via [§4 frozen v0.2](../constitutional-charter.md#4-constitutional-design-rule) structural-enforceability discipline + [`§0033`](../decision-log.md) local-shell-trust default preserved at [`§0119`](../decision-log.md) (opt-in audit, not default-no-audit). The default behavior is the operational-simplicity choice; the opt-in path is the discipline-prevails alternative; both are structurally available. Detectable via change-set review: a proposal that would remove the opt-in discipline path, or that would default to a simpler-but-discipline-violating shape, is a procedural §3 violation.

**N5 — Ghost Trace is not a generic event-sourcing framework.** Operationalized via [§2.2 frozen](../constitutional-charter.md#22-epistemic-separation) three-category typing (Cat I observation / Cat II construct / Cat III hypothesis) + [`§0010`](../decision-log.md) Q2-A.2 four-subtype Cat III taxonomy (`BehavioralCluster` / `AutomationGroup` / `CampaignHypothesis` / `CoordinationRing`). The categorical and subtype typing IS the behavioral-intelligence specificity; the substrate-immutability + content-addressing patterns are generic event-sourcing inheritance per §2.1. Detectable via §2.2 categorical-conflation anti-pattern + §2.5 lifecycle-event-as-Cat-I-record anti-pattern (per [§2.5 frozen v0.3](../constitutional-charter.md#25-hypothesis-lifecycle-explicitness)).

## Rationale

A system per [§1 Thesis](#1-thesis) without explicit non-goals accumulates feature creep that erodes the structural-enforceability discipline of [§4](../constitutional-charter.md#4-constitutional-design-rule). The five non-goals are categorical rejections, not "out of scope" notes — implementation work that would move toward any of N1–N5 is rejected on procedural grounds equivalent to violating a §2.x invariant.

The negative perimeter complements the positive structural commitments of §2.x: §2.x defines what the system MUST DO; §3 defines what the system MUST NOT DO. Together they circumscribe the system's identity. Per [§1 Thesis](../constitutional-charter.md#1-thesis), Ghost Trace's identity rests on preserving epistemic discipline; the non-goals enumerate the categorical directions that would erode it.

N1 (no truth) is the most structurally load-bearing — the central failure mode named in §1 Thesis (collapse of observation/inference/inherited-belief distinction) is what N1 forbids. The other four non-goals are secondary perimeter: N2 (no universal identity) prevents identity-resolution scope creep that would conflict with §2.2 epistemic separation; N3 (no automated irreversible action) prevents the system from becoming an actor rather than a record-of-action; N4 (no simplicity-over-discipline) protects the §4 discipline boundary; N5 (not generic event-sourcing) preserves the categorical typing identity.

## Forbidden Anti-Patterns

- **Truth-bearing record committed to substrate.** A substrate-committed record that carries a "true" or "definitive-conclusion" semantic without declared `influenced_by` chain per §2.4 and paired `evidential_independence` dimension per §2.6. Detectable at the canonical-serialization-contract layer per [`§0034`](../decision-log.md); rejection occurs at commit time.

- **Cross-domain identity-graph at substrate.** A new typed Cat I record committed to substrate that asserts identity equivalence across heterogeneous domain producers (e.g., a `GlobalIdentityRecord` linking a session-producer's `actor_ref` to an authentication-producer's `subject_id`). Detectable by §0024 schemas-evolution-contract review: such a record would require formal Q2 resolution before the proto can be added, AND constitutional review that the resolution does not violate N2.

- **Substrate-side autonomous deletion without audit.** A substrate operation that deletes records OR mutates an immutable record without committing an audit Cat I record alongside per [§0104 + §0119](../decision-log.md) audit-on-commit discipline. Detectable by substrate replay: every irreversible action must resolve to an `OrphanCleanupAudit` (or future analogous typed audit record) committed at or before the action's commit time.

- **Operational-simplicity choice that compromises substrate-immutability.** A change set that removes structural-discipline alternatives in favor of simpler-but-discipline-violating shapes (e.g., removing the §0119 opt-in audit option to "simplify the CLI surface"). Detectable by change-set review against §3 N4 + §4 criterion 1; the procedural rejection mirrors §2.x invariant-violation rejection.

- **Generic event-sourcing surface presented as substrate identity.** A documentation surface or interface that frames Ghost Trace's substrate as "an event-sourcing system" without acknowledging the §2.2 three-category typing + §0010 four-subtype Cat III taxonomy as the identity-defining structure. The substrate-immutability + content-addressing patterns are inherited from generic event-sourcing per [`§0024`](../decision-log.md) + [`§0042`](../decision-log.md); the inheritance is structural, but the IDENTITY of Ghost Trace is the typed-record-categorical-partition layer above the generic patterns. Detectable in documentation review + §2.2 categorical-conflation anti-pattern.

## Boundary Conditions

- **§3 governs the system's identity perimeter; not specific structural rules within the perimeter.** Each non-goal anchors to one or more §2.x invariants; the structural rules ARE those invariants. §3's binding text articulates the perimeter (N1–N5 as categorical rejections); §2.x's binding text articulates the rules. A change set that violates a §2.x invariant is detected at the §2.x level; §3 is the meta-statement that the rejection is categorical, not discretionary.

- **§3 N4 does not forbid operational-simplicity choices that don't conflict with §2.x discipline.** Operational simplicity is a legitimate design objective within the boundary defined by §2.x invariants. N4 only forbids simplicity choices that would COMPROMISE structural-enforceability per §4 — e.g., removing structural-discipline alternatives or defaulting to discipline-violating shapes. The §0033 default-no-audit + §0119 opt-in audit pattern IS the N4-compliant operational-simplicity-with-discipline-availability shape.

- **§3 N3 does not forbid operator-initiated irreversible action via audit-on-commit.** N3 forbids AUTONOMOUS (substrate-side, system-initiated) irreversible action. Operator-initiated actions via the [`§0104`](../decision-log.md) HTTP T3 OR [`§0119`](../decision-log.md) CLI opt-in path are explicitly permitted; the audit record committed at substrate IS the structural evidence that the action was operator-initiated, not autonomous.

- **§3 N5 does not forbid using event-sourcing patterns; only presenting them as the substrate identity.** Generic event-sourcing patterns (substrate-immutability, content-addressing, append-only commit semantics) are inherited; this is structural reality. N5 rejects the FRAMING that these patterns constitute Ghost Trace's identity. The identity is the §2.2 three-category typing + §0010 Q2-A.2 four-subtype Cat III taxonomy layered above the generic patterns.

- **§3 N1 does not forbid recording inferential commitment; only treating inferential commitment as truth.** Cat II constructs and Cat III hypotheses ARE inferential commitment records per §2.2; recording them is required. N1 forbids treating them as definitive truth — they MUST carry declared `influenced_by` chains per §2.4 + paired `evidential_independence` dimension per §2.6. The Cat II/III records are inferential; the structural pairing makes the inferential character substrate-visible.

- **§3 N2 does not forbid identity resolution within a single domain.** Producer-layer identity resolution per [`§0023`](../decision-log.md) inception-phase single-tier `actor_ref` is permitted (it IS the operational substrate). N2 forbids UNIVERSAL identity resolution — cross-domain identity reconciliation at the substrate level. Domain-specific resolution at the producer layer remains in scope; multi-tier formalization is deferred per Q2 Open Modeling Question.

- **§3 does not govern Charter amendment process.** Amendment to §3 (adding new non-goals, removing existing ones) follows the [`amendments.md`](../amendments.md) §Amendment Process. The non-goals enumerated here are not closed — committee may add or revise them via amendment.

- **§3 does not govern external system behavior.** Ghost Trace's non-goals are constraints on Ghost Trace. An external system that consumes Ghost Trace's outputs and adds its own decision logic is outside §3's scope; §3 governs what Ghost Trace itself rejects, not what consumers of Ghost Trace do with its outputs.

---

## Status note

Steps 1.1–1.5 complete in single committed arc per the [`§0129`](../decision-log.md) §2.6 closure precedent for committee-direction-consolidated redactions. Closure at [`§0131`](../decision-log.md) (amendment v0.7).
