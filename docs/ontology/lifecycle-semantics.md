# Lifecycle Semantics

**Status:** Drafted — Category III, §The Promotion Mechanism, and §Open Modeling Questions revised per [`decision-log.md` §0011](../charter/decision-log.md) (Q4 resolution, staged-combination form) and further revised per [`decision-log.md` §0135](../charter/decision-log.md) (Layer B resolution — L-BC-OR adopted, §2.5 forward-reference fully discharged). The document advances from Scaffold to Drafted on the redaction strength of these sections; other sections (Category I, Category II) remain at scaffold strength pending future redaction.

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
- **Demotion.** A previously promoted hypothesis becomes a demotion candidate when **both** of the following hold (staged-combination form per [`decision-log.md` §0011](../charter/decision-log.md); inner Layer B form resolved per [`decision-log.md` §0135](../charter/decision-log.md); operational parameter values fixed per [`decision-log.md` §0138](../charter/decision-log.md)):
  - **(a) Cadence gate (Layer A).** The elapsed time since the promotion event exceeds `N_A`. Per [`§0138`](../charter/decision-log.md) inception-phase resolution, `N_A` = 1 day (encoded as `n_a_duration_nanoseconds: 86400000000000` in the LayerBParameters proto per [`§0136`](../charter/decision-log.md) canonical-serialization-contract). Revision per the §0138 per-parameter reversal-conditions record.
  - **(b) Deep criterion (Layer B).** Per [`decision-log.md` §0135](../charter/decision-log.md), Layer B is the **disjunctive composition (L-BC-OR)** of two structural tests:
    - **Evidence-staleness (B-family):** `freshness_B(H) < T_B`, where `freshness_B(H)` is the average `evidential_independence` (per [Charter §2.6](../charter/constitutional-charter.md#26-evidential-independence-integrity) frozen v0.6 + Q3-α per [`§0133`](../charter/decision-log.md)) over the most recent `N` assertions whose transitive `influenced_by` (per Q5-τ per [`§0134`](../charter/decision-log.md)) includes H.
    - **Influence-saturation (C-family):** `saturation_C(H) > K_C`, where `saturation_C(H)` is `(count of recent assertions with H in transitive influenced_by, EXCLUDING H's own enrichment outputs) / N`. The structural exclusion of H's own enrichment outputs is the L-C committee extension per [`§0135`](../charter/decision-log.md); the exclusion uses [Charter §2.4](../charter/constitutional-charter.md#24-inferential-influence-disclosure) v0.5 chain inspection to identify assertions formed under H's influence.

    Layer B fires when EITHER B-family OR C-family condition holds. Per [`§0138`](../charter/decision-log.md) inception-phase resolution: `T_B = K_C = 0.5` (rational `1/2`); `N = 1000`; window form is W-count (last `N` assertions by substrate-commit order); per-subtype divergence is U-uniform (single parameter set at abstract `Hypothesis` level for all four Cat III concrete subtypes). Revision per the §0138 per-parameter reversal-conditions record.

  Demotion itself, once a candidate is confirmed, is recorded as an immutable event referencing the prior promotion event. A demoted hypothesis remains in the substrate; its promotion and demotion events compose its history.
- **Dissolution.** A hypothesis is recognized as no longer corresponding to any underlying phenomenon and is marked as no longer active. Recorded as an immutable event. Distinguished from demotion: demotion withdraws operational use; dissolution recognizes non-existence of the underlying phenomenon. The two operations are not interchangeable.

All such operations are recorded as immutable lifecycle events in the primary event log per [Invariant 2.5](../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness). The current state of a hypothesis is a projection over its history, never produced by direct mutation.

## The Promotion Mechanism

The most consequential lifecycle operation in the system is **promotion** — the transition of a hypothesis from active inference to operational use as enrichment context.

The Charter's central concern about recursive belief inflation ([Charter §1](../charter/constitutional-charter.md#1-thesis)) arises specifically from how promotion is handled. The system must:

1. Subject every candidate promotion to evaluation against criteria of maturity, breadth, and `confidence`. The evaluation procedure itself is deferred to the §2.5 redaction. `Confidence` and `evidential independence` are treated as separate structural dimensions per [Charter §2.6](../charter/constitutional-charter.md#26-evidential-independence-integrity) frozen v0.6; the bare scalar `independence` is not a structural property of the substrate.
2. Record the promotion as an immutable event. The promotion event carries structural parameters that govern the promoted hypothesis's subsequent demotion-candidacy — at minimum the cadence parameter `N_A` for Layer A of the staged-combination demotion criterion per [`decision-log.md` §0011](../charter/decision-log.md); per [`decision-log.md` §0138](../charter/decision-log.md) `N_A` is bundled into the LayerBParameters proto (single source of truth for both Layer A and Layer B parameters).
3. Ensure that every assertion subsequently formed under the influence of the promoted hypothesis carries a structural declaration of that influence per [Charter §2.4](../charter/constitutional-charter.md#24-inferential-influence-disclosure) frozen v0.5.
4. Apply the staged-combination demotion criterion (per [`decision-log.md` §0011](../charter/decision-log.md)) to every promoted hypothesis: Layer A (cadence gate) evaluates against substrate timestamps with `N_A = 1 day` per [`§0138`](../charter/decision-log.md); Layer B (L-BC-OR per [`decision-log.md` §0135](../charter/decision-log.md)) evaluates the disjunction of `freshness_B(H) < T_B` (B-family evidence-staleness) and `saturation_C(H) > K_C` (C-family influence-saturation, with structural exclusion of H's own enrichment outputs per the L-C commitment). Both metrics reduce to substrate-committed queries — `freshness_B` reads Q3-α values per [Charter §2.6](../charter/constitutional-charter.md#26-evidential-independence-integrity) frozen v0.6; `saturation_C` reads Q5-τ transitive `influenced_by` chains per [Charter §2.4](../charter/constitutional-charter.md#24-inferential-influence-disclosure) frozen v0.5 + [`§0134`](../charter/decision-log.md). Parameter values per [`§0138`](../charter/decision-log.md): `T_B = K_C = 0.5`; `N = 1000`; window form W-count; per-subtype divergence U-uniform. Revision per the §0138 per-parameter reversal-conditions record (observation-based triggers per [`§0022`](../charter/decision-log.md) empirical-pressure-phase discipline).

## Open Modeling Questions

The following modeling questions remain open at this document's current revision:

1. **Operational definition versioning.** Constructs reference the definition that produced them. When a definition is revised, do existing constructs remain valid until explicitly re-derived, or are they implicitly stale?
2. **Cross-category lifecycle interactions.** When a promoted hypothesis is demoted, what happens to operational constructs that incorporated it as enrichment? Are they re-derived, marked as stale, or left intact with a note?

These questions will be answered in committee redaction. They are not resolved here.

## Resolved Modeling Questions

- **Question 1 (Hypothesis subtypes lifecycle, originally line 50 of this document's scaffold).** Resolved by [`decision-log.md` §0010 — Q2 resolution](../charter/decision-log.md) as Candidate A.2. The four concrete subtypes share the lifecycle of the abstract `Hypothesis` type; subtype-specific parameters appear where called out in the §Hypothesis (Category III) section above.
- **Question 4 (Independence-driven lifecycle, originally a narrower binary framing of canonical Ontology Q4).** Resolved by [`decision-log.md` §0011 — Q4 resolution](../charter/decision-log.md) as the staged-combination form (Layer A AND Layer B). The previously framed binary (automatic-on-threshold vs operator-driven) is superseded: the resolution is neither alternative as framed, but a staged structural form. Layer B's inner structure is further resolved by [`decision-log.md` §0135 — Layer B resolution](../charter/decision-log.md) as L-BC-OR (disjunctive — `(freshness_B < T_B) OR (saturation_C > K_C)`) with the L-C structural-exclusion commitment. Inception-phase parameter values are resolved by [`decision-log.md` §0138 — Layer B parameter-calibration resolution](../charter/decision-log.md): `T_B = K_C = 0.5`; `N = 1000`; window form W-count; per-subtype divergence U-uniform; `N_A = 1 day` (bundled). The full Q4 → Layer B operational arc (§0011 → §0099 → §0129 → §0133 → §0134 → §0135 → §0136 → §0138) is structurally complete at the operational-specification layer.

<!-- TODO: After Invariant 2.5 is redacted, formalize the hypothesis lifecycle state machine with explicit transition events. -->

<!-- TODO: Coordinate with [`provenance-model.md`](./provenance-model.md) on how lifecycle events appear in the provenance graph. -->

