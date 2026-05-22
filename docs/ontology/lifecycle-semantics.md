# Lifecycle Semantics

**Status:** Drafted — Category III, §The Promotion Mechanism, and §Open Modeling Questions revised per [`decision-log.md` §0011](../charter/decision-log.md) (Q4 resolution, staged-combination form). The document advances from Scaffold to Drafted on the redaction strength of these sections; other sections (Category I, Category II) remain at scaffold strength pending future redaction.

> This document formalizes how entities in each ontological category are created, evolved, superseded, and (where applicable) dissolved. The Charter establishes hard rules about mutation ([Invariant 2.1](../charter/constitutional-charter.md#21-observational-integrity), [Invariant 2.2](../charter/constitutional-charter.md#22-epistemic-separation), and [Invariant 2.5](../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness)). This document specifies the lifecycle each category supports under those rules.

## Lifecycle by Category

### Observation (Category I)

- **Creation.** Observation records are committed to the primary event log. Producers are external (client SDKs, infrastructure collectors, authoritative ledgers).
- **Evolution.** None. Observations are immutable.
- **Supersession.** Not applicable. The *interpretation* of an observation may be superseded by a new assertion (in Category II or III), but the observation itself remains.
- **Dissolution.** Not applicable. Observations are preserved for the lifetime of the log.

### Operational Construct (Category II)

- **Creation.** Constructed deterministically from observations under a versioned operational definition.
- **Evolution.** None at the level of an individual construct. A new definition produces new constructs; existing constructs are not mutated.
- **Supersession.** A construct may be superseded by another construct produced under a revised definition. Both remain accessible.
- **Dissolution.** Not applicable in the substrate. A projection may stop materializing a construct, but the underlying derivation rule and its inputs remain.

### Hypothesis (Category III)

Per [`entity-model.md` §Category III](./entity-model.md) (revised per [`decision-log.md` §0010](../charter/decision-log.md)): Category III is structured as the abstract type `Hypothesis` with four concrete subtypes (`BehavioralCluster`, `CoordinationRing`, `CampaignHypothesis`, `AutomationGroup`). The lifecycle described in this section applies at the abstract-type level; subtype-specific parameters appear where called out.

- **Formation.** A hypothesis (one of the four concrete subtypes) is created when an inference process recognizes accumulated observation records crossing a structural threshold. Formation is recorded as an immutable lifecycle event in the primary event log per [Invariant 2.5](../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness).
- **Merge.** Two hypotheses (same subtype or, for cross-subtype merge, antecedents of different subtypes per [`§0122`](../charter/decision-log.md) Candidate γ resolution — produced record's type determined by the per-pair canonical-target table at [`entity-model.md` §Cross-subtype operations](./entity-model.md)) are recognized as describing the same underlying phenomenon and combined. Merge is recorded as an immutable event referencing both antecedents and the produced hypothesis. Cross-subtype merge enablement criterion per [`§0123`](../charter/decision-log.md) Candidate B+D: cross-subtype merge is permitted when antecedents share at least one `actor_ref` AND both antecedents are in promoted lifecycle-state.
- **Split.** A hypothesis is recognized as containing multiple distinct phenomena and is divided into multiple successor hypotheses. Recorded as an immutable event referencing the antecedent and each successor. Cross-subtype split typing + enablement per [`§0124`](../charter/decision-log.md) symmetric with merge: successors drawn from existing four subtypes per per-source permitted-target table; permitted when successor memberships partition the antecedent's membership AND the antecedent is in promoted lifecycle-state. See [`entity-model.md` §Cross-subtype operations](./entity-model.md) for the consolidated reference.
- **Promotion.** A hypothesis is admitted to operational use as enrichment context. Recorded as an immutable event carrying the parameters that govern the hypothesis's subsequent demotion-candidacy (per §The Promotion Mechanism below).
- **Demotion.** A previously promoted hypothesis becomes a demotion candidate when **both** of the following hold (staged-combination form per [`decision-log.md` §0011](../charter/decision-log.md)):
  - **(a) Cadence gate (Layer A).** The elapsed time since the promotion event exceeds a parameter `N` recorded on the promotion event or on the hypothesis's concrete subtype.
  - **(b) Deep criterion (Layer B).** A designated structural test on `evidential independence` (per [Charter §2.6](../charter/constitutional-charter.md#26-evidential-independence-integrity) pending) or on declared `influence` (per [Charter §2.4](../charter/constitutional-charter.md#24-inferential-influence-disclosure) pending) — the specific shape deferred to a follow-on RFC per [`decision-log.md` §0011](../charter/decision-log.md) — fires.

  Demotion itself, once a candidate is confirmed, is recorded as an immutable event referencing the prior promotion event. A demoted hypothesis remains in the substrate; its promotion and demotion events compose its history.
- **Dissolution.** A hypothesis is recognized as no longer corresponding to any underlying phenomenon and is marked as no longer active. Recorded as an immutable event. Distinguished from demotion: demotion withdraws operational use; dissolution recognizes non-existence of the underlying phenomenon. The two operations are not interchangeable.

All such operations are recorded as immutable lifecycle events in the primary event log per [Invariant 2.5](../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness). The current state of a hypothesis is a projection over its history, never produced by direct mutation.

## The Promotion Mechanism

The most consequential lifecycle operation in the system is **promotion** — the transition of a hypothesis from active inference to operational use as enrichment context.

The Charter's central concern about recursive belief inflation ([Charter §1](../charter/constitutional-charter.md#1-thesis)) arises specifically from how promotion is handled. The system must:

1. Subject every candidate promotion to evaluation against criteria of maturity, breadth, and `confidence`. The evaluation procedure itself is deferred to the §2.5 redaction. `Confidence` and `evidential independence` are treated as separate structural dimensions per [Charter §2.6](../charter/constitutional-charter.md#26-evidential-independence-integrity) (pending); the bare scalar `independence` is not a structural property of the substrate.
2. Record the promotion as an immutable event. The promotion event carries structural parameters that govern the promoted hypothesis's subsequent demotion-candidacy — at minimum the cadence parameter `N` for Layer A of the staged-combination demotion criterion per [`decision-log.md` §0011](../charter/decision-log.md).
3. Ensure that every assertion subsequently formed under the influence of the promoted hypothesis carries a structural declaration of that influence per [Charter §2.4](../charter/constitutional-charter.md#24-inferential-influence-disclosure) (pending).
4. Apply the staged-combination demotion criterion (per [`decision-log.md` §0011](../charter/decision-log.md)) to every promoted hypothesis: Layer A (cadence gate) evaluates against substrate timestamps; Layer B (deep criterion on `evidential independence` or declared `influence`) evaluates against §2.6 / §2.4 structural surfaces once those invariants are redacted and the follow-on RFC specifies the deep criterion's shape. Until Layer B is specified, demotion candidacy requires Layer A to fire AND the committee or operator to invoke the deferred deep test by procedural means recorded on the demotion event. The procedural path is a known fragility of the resolution per the [`decision-log.md` §0011](../charter/decision-log.md) reversal-condition record.

## Open Modeling Questions

The following modeling questions remain open at this document's current revision:

1. **Operational definition versioning.** Constructs reference the definition that produced them. When a definition is revised, do existing constructs remain valid until explicitly re-derived, or are they implicitly stale?
2. **Cross-category lifecycle interactions.** When a promoted hypothesis is demoted, what happens to operational constructs that incorporated it as enrichment? Are they re-derived, marked as stale, or left intact with a note?

These questions will be answered in committee redaction. They are not resolved here.

## Resolved Modeling Questions

- **Question 1 (Hypothesis subtypes lifecycle, originally line 50 of this document's scaffold).** Resolved by [`decision-log.md` §0010 — Q2 resolution](../charter/decision-log.md) as Candidate A.2. The four concrete subtypes share the lifecycle of the abstract `Hypothesis` type; subtype-specific parameters appear where called out in the §Hypothesis (Category III) section above.
- **Question 4 (Independence-driven lifecycle, originally a narrower binary framing of canonical Ontology Q4).** Resolved by [`decision-log.md` §0011 — Q4 resolution](../charter/decision-log.md) as the staged-combination form. The previously framed binary (automatic-on-threshold vs operator-driven) is superseded: the resolution is neither alternative as framed, but a staged structural form with Layer A (cadence gate) AND Layer B (deep criterion on `evidential independence` or declared `influence`, deferred to a follow-on RFC).

<!-- TODO: After Invariant 2.5 is redacted, formalize the hypothesis lifecycle state machine with explicit transition events. -->

<!-- TODO: Coordinate with [`provenance-model.md`](./provenance-model.md) on how lifecycle events appear in the provenance graph. -->

<!-- TODO: When the follow-on RFC (per decision-log §0011) specifies Layer B's deep criterion, revise §The Promotion Mechanism step 4 to make Layer B's structural test concrete; remove the "procedural path" fragility note. -->
