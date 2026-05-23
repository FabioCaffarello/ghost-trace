# RFC — Operational specification: Layer B service-tier implementation

- **Status:** discussion
- **Authors:** Ghost Trace committee (per [`decision-log §0138`](../../charter/decision-log.md) Consequences carry-forward — "service-tier implementation now openable as ordinary RFC discipline" — opening this RFC at discussion phase as the next operational-specification follow-on after [`§0136`](../../charter/decision-log.md) canonical-serialization-contract crystallization + [`§0138`](../../charter/decision-log.md) parameter calibration)
- **Date:** 2026-05-23 (opened)
- **Type:** operational-spec
- **Affects:** [`services/ingestion/internal/hypothesis/`](../../../services/ingestion/internal/hypothesis/) (new evaluation surface; affects `demotion.go` + per-subtype demotion implementations); [`services/ingestion/cmd/demote-hypothesis/`](../../../services/ingestion/cmd/demote-hypothesis/) + four subtype-specific demote commands (CLI surface for Layer B reporting); potentially new `cmd/find-demotion-candidates/` (depends on sub-decision E); [`docs/architecture/canonical-serialization-contract.md`](../../architecture/canonical-serialization-contract.md) §Demotion-Candidacy Predicate (operational discharge of the contract's structural surface — values + computation strategy + locus); no Charter prose modification (form-level + parameter-level resolutions already at [`§0135`](../../charter/decision-log.md) + [`§0138`](../../charter/decision-log.md))

> RFCs in Ghost Trace are proposals subject to constitutional review. The [Constitutional Charter](../../charter/constitutional-charter.md) is the authoritative reference. RFCs that conflict with the Charter must either be rejected or accompanied by a corresponding charter amendment (see [`../../charter/amendments.md`](../../charter/amendments.md)). This RFC is operational specification within the form-vs-parameter discipline established at [`§0011`](../../charter/decision-log.md) Methodological Observation 1 — the form is structural and committee-resolved at [`§0135`](../../charter/decision-log.md); the values are operational and were resolved at [`§0138`](../../charter/decision-log.md); the implementation strategy is operational and is resolved here.

## Status note

**All prior dependencies are discharged; service-tier implementation drafting opens.**

- **Form** — [`§0135`](../../charter/decision-log.md) Layer B = L-BC-OR with L-C structural-exclusion commitment. ✓
- **Canonical-serialization-contract crystallization** — [`§0136`](../../charter/decision-log.md) full predicate surface at contract layer; [`§0139`](../../charter/decision-log.md) hash-list discipline generalization; [`§0140`](../../charter/decision-log.md) paired-dimension marshalling-boundary enforcement. ✓
- **Parameters** — [`§0138`](../../charter/decision-log.md) inception-phase values: `T_B = K_C = 0.5`; `N = 1000`; W-count window; U-uniform per-subtype divergence; `N_A = 1 day` (bundled). ✓
- **Implementation surface clearance** — empirical audit confirms `services/ingestion/` is the sole actively-implemented service (193 Go files; 4 subtypes × 6 lifecycle operations = 24 CLIs plus utilities); `assertion-engine`, `graph`, `projections`, `replay` are README-only; the existing demote-hypothesis + 4 subtype variants explicitly defer Layer B evaluation at [`services/ingestion/internal/hypothesis/demotion.go:97-99`](../../../services/ingestion/internal/hypothesis/demotion.go) (comment text stale post-[`§0129`](../../charter/decision-log.md) §2.6 freeze + [`§0138`](../../charter/decision-log.md) Layer B specification — comment refresh is a downstream consequence of this RFC's resolution, not a framing-phase action). ✓

The implementation surface is consumable; the structural commitments are operational; the parameters are concrete. **The remaining open decisions are about how Layer B's predicate is COMPUTED and PRESENTED at the service tier** — the structural shape and parameter values are committee-fixed.

## Summary

Per [`§0135`](../../charter/decision-log.md) Decision + [`§0136`](../../charter/decision-log.md) Decision + [`§0138`](../../charter/decision-log.md) Decision, the full demotion-candidacy predicate at the canonical-serialization-contract layer is:

> `DEMOTE-CANDIDATE(H) := Layer A(H) AND ((freshness_B(H) < T_B) OR (saturation_C(H) > K_C))`

with `T_B = K_C = 0.5`; `N = 1000`; W-count window; U-uniform per-subtype.

The service tier does not currently evaluate this predicate. The existing demote-hypothesis CLI (plus four subtype-specific variants) is pure operator-elected substrate-commit: it records the demotion regardless of any criterion, reporting `CadenceSatisfied` (Layer A advisory) post-facto. Layer B is named in proto types (`LayerBParameters` per [`schemas/common/v1/layer_b_parameters.proto`](../../../schemas/common/v1/layer_b_parameters.proto)) but not evaluated. This RFC opens discussion of how the service tier should compute the predicate, where the computation lives in the call chain, what artifact (if any) it produces, and how the existing demote CLIs interact with the evaluation.

This RFC opens structured discussion. Candidate decisions are enumerated in [`layer-b-service-tier-implementation-evidence.md`](../discussion/layer-b-service-tier-implementation-evidence.md) Phase 2 (six sub-decisions). This RFC does not pick at this phase.

## Motivation

**Why now.** The two predecessor RFCs in the Layer B arc — [`ontology-revision-layer-b-deep-criterion`](./ontology-revision-layer-b-deep-criterion.md) (form, accepted at [`§0135`](../../charter/decision-log.md)) and [`operational-spec-layer-b-parameter-calibration`](./operational-spec-layer-b-parameter-calibration.md) (parameters, accepted at [`§0138`](../../charter/decision-log.md)) — fully discharged the structural and parametric commitments. [`§0138`](../../charter/decision-log.md) Consequences explicitly carry forward to this RFC: "service-tier implementation now openable as ordinary RFC discipline." Per the [`§0011`](../../charter/decision-log.md) form-vs-parameter-vs-implementation tripartite established in Methodological Observation 1 and extended through the Q4 → Layer B arc, the implementation step is the third tier of the same staged discipline.

**The cost of not resolving.** Implementation work consuming the contract has two options absent this RFC: (a) commit implementation choices inline without committee review (which violates the form-vs-parameter-vs-implementation discipline by elevating implementation choices to structural weight); (b) defer Layer B firing entirely (which leaves the demotion-candidacy gate operational only via Layer A, as today, and renders [`§0138`](../../charter/decision-log.md) parameter values unconsulted runtime). Both are forbidden by [`§0138`](../../charter/decision-log.md) Consequences carry-forward + [`§0011`](../../charter/decision-log.md) Methodological Observation 1.

**The downstream consequence**. The [`services/ingestion/internal/hypothesis/demotion.go:97-99`](../../../services/ingestion/internal/hypothesis/demotion.go) comment text — *"Layer B (deep criterion on evidential independence) remains deferred until §2.6 operationalization; Demote does not evaluate it"* — is stale post-[`§0129`](../../charter/decision-log.md) §2.6 freeze + [`§0138`](../../charter/decision-log.md) Layer B specification. The comment refresh is a follow-on consequence of this RFC's resolution; not a framing-phase action.

## Constitutional Review

The Q1–Q6 impact analysis prescribed by [`rfc-author` §1](../../../.claude/skills/workflow/rfc-author/SKILL.md). Note: this RFC is OPERATIONAL specification at the implementation tier, not constitutional or structural; several Q-answers are trivially "no" because the form is committee-resolved at [`§0135`](../../charter/decision-log.md) and the parameters are committee-resolved at [`§0138`](../../charter/decision-log.md).

### Q1 — Which Charter invariants does this RFC touch?

Indirectly via implementation behavior, no Charter prose is modified or referenced operatively:

- **§2.5 Hypothesis Lifecycle Explicitness** (frozen v0.3): Layer B firing produces a `demotion` lifecycle event per [`§0011`](../../charter/decision-log.md) staged-combination. This RFC specifies HOW the predicate is computed and whether it BLOCKS, ADVISES, or is consulted separately — without modifying §2.5 prose.
- **§2.6 Evidential Independence Integrity** (frozen v0.6): `freshness_B` operates on `evidential_independence` values per [`§0133`](../../charter/decision-log.md) Q3-α. This RFC specifies the implementation-tier computation strategy (on-the-fly vs cached projection vs committed-event); §2.6 paired-dimension commitment is structural and is consumed, not modified.
- **§2.4 Inferential Influence Disclosure** (frozen v0.5): `saturation_C` operates on transitive `influenced_by` closures per [`§0134`](../../charter/decision-log.md) Q5-τ. This RFC specifies the implementation-tier closure-read strategy; §2.4 chain declaration is structural and is consumed, not modified.
- **§1 Thesis**: Implementation choices determine how aggressively and visibly the system structurally defends against the two failure modes at the service-CLI surface. Conservative defaults preserve defense surface; advisory-rather-than-enforcing demote behavior preserves operator authority over substrate commits per [`§0011`](../../charter/decision-log.md) Layer A pattern precedent.
- **§2.1 Observational Integrity**: If sub-decision D resolves toward emitting a `LayerBEvaluation` substrate record (D2 or D3 in Phase 2), the record is itself a Category I substrate event subject to §2.1 immutability post-commit. The current substrate-writer serialization discipline per [`docs/architecture/concurrency-pattern.md`](../../architecture/concurrency-pattern.md) §Substrate-Writer Serialization is consumed, not modified.

### Q2 — Does this RFC implicitly redefine any term in the glossary?

No. The RFC uses canonical vocabulary (`evidential_independence`, `influenced_by`, `closure_hashes`, `demotion`, `Hypothesis`, `Category I/II/III`, `substrate`, `projection`) without redefinition. Implementation-locus terms (`evaluator`, `candidate-finder`, `on-the-fly`, `cached projection`) are introduced at this RFC for the sub-decision enumeration; they are implementation vocabulary, not Charter or Ontology vocabulary, and do not require glossary entries.

### Q3 — Does this RFC implicitly resolve any of the five open Ontology questions?

No. All five original ontology.md Open Questions are resolved as of [`§0134`](../../charter/decision-log.md) Methodological Observation 4. This RFC composes with the resolutions; it does not re-open them.

### Q4 — Does this RFC require Charter amendment?

No. The RFC is operational-specification at the implementation tier; it does NOT modify Charter prose. The form-vs-parameter-vs-implementation discipline established at [`§0011`](../../charter/decision-log.md) Methodological Observation 1 + extended at [`§0135`](../../charter/decision-log.md) + [`§0138`](../../charter/decision-log.md) explicitly authorizes this separation: form-level RFCs are structural; parameter-value RFCs are operational at the canonical-serialization-contract layer; implementation RFCs are operational at the service-tier layer.

### Q5 — Does this RFC introduce a new invariant?

No. The RFC specifies operational implementation within an existing structural commitment (L-BC-OR per [`§0135`](../../charter/decision-log.md)) and parameter set (per [`§0138`](../../charter/decision-log.md)). No new Charter invariant. Sub-decision D (substrate-record emission) MIGHT introduce a new Cat II proto type if a `LayerBEvaluation` record is adopted; that is a schemas-evolution event per the canonical-serialization-contract's §Schemas-Evolution Events boundary item, not a Charter invariant.

### Q6 — Does this RFC propose ceremony without behavioral consequence?

No. Implementation choices determine when, how, and where Layer B's predicate is computed; whether its output is durable substrate evidence or transient; whether the existing demote-hypothesis CLIs run the predicate or remain pure operator-elected. The behavioral consequences are direct and material — different implementation choices produce materially different operator experiences, different audit-trail completeness, different substrate-storage costs, and different service-architecture footprints. The implementation choices are operational; their behavioral consequences are constitutional via [`§1 Thesis`](../../charter/constitutional-charter.md#1-thesis) defense surface presentation.

## Proposal

Six sub-decisions enumerated in [`layer-b-service-tier-implementation-evidence.md`](../discussion/layer-b-service-tier-implementation-evidence.md) Phase 2:

| Sub-decision | Candidates |
|---|---|
| A. Evaluation locus | A1-internal-package (`internal/hypothesis/layerb`) / A2-separate-service (`assertion-engine`) / A3-inline-per-CLI |
| B. Computation strategy | B1-on-the-fly-from-substrate / B2-Cat-II-projection / B3-on-the-fly-with-substrate-audit-record |
| C. W-count N=1000 stream identity | C1-substrate-global / C2-per-hypothesis-closure / C3-per-subtype / C4-since-promotion |
| D. Output shape | D1-transient-DemoteReport / D2-LayerBEvaluation-Cat-II-record / D3-DemotionCandidacyEvaluation-composite |
| E. demote-hypothesis interaction | E1-advisory-like-Layer-A / E2-enforcing-refuse / E3-untouched-separate-candidate-finder |
| F. N_A=1 day Layer A cadence source | F1-bundled-in-LayerBParameters / F2-separate-Layer-A-config / F3-CLI-operator-supplied-with-defaults |

This RFC does not pick at this phase. The evidence file records the inception-phase posture asymmetries that will likely organize substantive deliberation:

- **Conservative-defaults asymmetry**: A1 (internal package, no new service) + B1 (on-the-fly) + D1 (transient) + E1 (advisory) align with [`§0023`](../../charter/decision-log.md) inception-phase posture (deferring multi-tier formalization where simpler form is sufficient) and [`§0011`](../../charter/decision-log.md) Layer A pattern precedent (Layer A is advisory in DemoteReport, not enforcing). Aggressive variants (A2 separate service; B2 projection; D2/D3 substrate records; E2 enforcing) require empirical justification not currently available — the service has zero promoted hypotheses in production today; cost-of-projection-storage and cost-of-extra-substrate-events are unknowns.
- **Form-vs-implementation respect asymmetry**: Sub-decision C (window stream identity) operates within the W-count form fixed at [`§0138`](../../charter/decision-log.md). C-variants are implementation readings of N=1000, not parameter-form revisions. A future implementation RFC may revise C without re-opening the form-level RFC. The same form-vs-parameter-vs-implementation discipline pattern repeats.
- **Substrate-record-emission asymmetry**: Sub-decisions D2 + D3 (emit Cat II `LayerBEvaluation` or `DemotionCandidacyEvaluation` substrate record) are schemas-evolution events per the canonical-serialization-contract §Schemas-Evolution Events boundary. They MAY trigger a separate schemas-evolution event RFC if adopted; D1 (transient) is contained within the existing schema set and triggers no further RFC.

## Alternatives Considered

Out-of-scope candidate forms surfaced and explicitly rejected during framing:

- **Stand-alone scheduled-job-only evaluation.** Considered: a cron-style scheduled job that periodically scans all promoted hypotheses and emits `LayerBEvaluation` records, with no CLI integration. Rejected as inception-phase: there are no scheduled jobs in `services/ingestion/` today; introducing the scheduling primitive ahead of need is premature. Layer B evaluation can be on-demand via CLI (sub-decision E variants); periodic-scan is a future operational consideration, not a service-tier-MVP requirement.
- **GraphQL-style query layer over Layer B state.** Considered: exposing Layer B evaluation via a query API. Rejected: there is no query layer in `services/ingestion/` today; the service is CLI-only. Introducing a query layer is an architecture decision (services/graph or services/projections), not an operational-spec decision. Future operational-spec RFCs may revise this.
- **Multi-criterion composition beyond §0135 L-BC-OR.** Considered: introducing per-subtype OR per-deployment overrides on the L-BC-OR firing predicate. Rejected: [`§0135`](../../charter/decision-log.md) committed to L-C structural-exclusion, meaning the firing predicate is structurally fixed at L-BC-OR. Per-subtype divergence was explicitly resolved as U-uniform at [`§0138`](../../charter/decision-log.md). Re-opening either would require form-level RFC, not implementation-tier.

## Open Questions

This RFC's open questions are the six sub-decisions in §Proposal. They are framed for discussion phase; substantive deliberation in Phases 3-5 of [`layer-b-service-tier-implementation-evidence.md`](../discussion/layer-b-service-tier-implementation-evidence.md) will produce a recommendation; resolution lands at a subsequent RFC-acceptance commit + decision-log entry.

Cross-cutting question raised during framing: **What is the unit of Layer B's per-hypothesis closure read at evaluation time?** Per [`§0136`](../../charter/decision-log.md), formation events store `closure_hashes` (transitive `influenced_by` closure under τ committed at substrate event time). At Layer B evaluation, the implementation reads the promoted hypothesis's `closure_hashes` set and computes `freshness_B` (fresh Cat I roots in window) + `saturation_C` (closure roots reachable via promoted-hypothesis-influenced chains in window). The unit-of-read is well-defined; the window-identity question (sub-decision C) is what's open. This cross-cutting clarification is not itself a sub-decision; it's a framing constraint that all candidates must respect.

## Anti-Patterns to Avoid

- **Inline criterion evaluation in CLI command bodies.** Each of the 5 demote CLIs (demote-hypothesis + 4 subtype variants) currently has a tight body that calls `internal/hypothesis.Demote` or its subtype-specific variant. Inlining Layer B computation in each CLI (sub-decision A3) duplicates the predicate logic five times and is the anti-pattern this RFC's resolution should explicitly reject in favor of internal-package centralization (A1) or service-split (A2).
- **Silent substrate emission without operator awareness.** If sub-decision D resolves toward D2 or D3 (Cat II substrate record), the demote CLI's reporting MUST surface that a substrate-side record was committed alongside the operator's demote action. Silent emission is a substrate-immutability surprise.
- **Reading mutable parameters from non-substrate sources at evaluation time.** Per [`§0136`](../../charter/decision-log.md) + [`§0138`](../../charter/decision-log.md), `LayerBParameters` carries the operational values committed to substrate via per-record `layer_b_parameters` field. The implementation MUST read parameters from the hypothesis's `LayerBParameters` field, not from environment variables or runtime config — this would conflate operational-specification-time mutability with runtime mutability, violating [`§0136`](../../charter/decision-log.md) parameter-mutability prohibition.
- **Layer B as a hard barrier on demote without operator override.** Per [`§0011`](../../charter/decision-log.md) staged-combination, Layer A + Layer B is a CANDIDACY criterion. Operator-elected demotion is the substrate-commit primitive. If sub-decision E resolves toward E2 (enforcing-refuse), it MUST be paired with an explicit operator override mechanism — substrate writers retain authority over substrate commits per the Layer A pattern.

## Migration and Backward Compatibility

The existing demote-hypothesis + 4 subtype demote CLIs are operational today and produce demotion lifecycle events. This RFC's resolution will affect their behavior in proportion to the chosen sub-decisions:

- If E3 (untouched-separate-candidate-finder) is adopted: zero behavior change for existing CLIs. New `find-demotion-candidates` CLI surfaces Layer B state separately; demote CLIs remain pure operator-elected.
- If E1 (advisory-like-Layer-A) is adopted: existing CLIs gain Layer B state in DemoteReport, same shape as the existing CadenceSatisfied flag. No commit semantics change; the report surface expands.
- If E2 (enforcing-refuse) is adopted: existing CLIs gain a refusal path when Layer B(H) is false. Operator override required for backward-compatible operation; the override mechanism is a design point under E2.

The `services/ingestion/internal/hypothesis/demotion.go` Layer B deferred-comment is stale and will be refreshed at the RFC-resolution PR (not at this framing PR).

No corpus or canonical-byte changes from this RFC. No schemas-evolution event from this framing PR; sub-decisions D2/D3 may trigger a schemas-evolution event at resolution time per the canonical-serialization-contract §Schemas-Evolution Events boundary.

## References

- [`decision-log §0011`](../../charter/decision-log.md) — Q4 resolution: staged-combination demotion criterion (Layer A + Layer B).
- [`decision-log §0099`](../../charter/decision-log.md) — Gate §2.4 closure (frozen v0.5).
- [`decision-log §0129`](../../charter/decision-log.md) — Gate §2.6 closure (frozen v0.6).
- [`decision-log §0133`](../../charter/decision-log.md) — Q3 resolution: Candidate α (source-count ratio) — the measurable quantity `freshness_B` operates on.
- [`decision-log §0134`](../../charter/decision-log.md) — Q5 transitivity-half resolution: Candidate τ (transitive closure) — the closure that `saturation_C` reads.
- [`decision-log §0135`](../../charter/decision-log.md) — Layer B form: L-BC-OR with L-C structural-exclusion commitment.
- [`decision-log §0136`](../../charter/decision-log.md) — Canonical-serialization-contract revision: α + τ + β-graph + L-BC-OR firing-predicate crystallization.
- [`decision-log §0138`](../../charter/decision-log.md) — Layer B parameter calibration: T_B = K_C = 0.5; N = 1000; W-count; U-uniform; N_A = 1 day bundled.
- [`decision-log §0139`](../../charter/decision-log.md) — Canonical-serialization-contract revision: BLAKE3-hash-list element-shape discipline generalization.
- [`decision-log §0140`](../../charter/decision-log.md) — Canonical-serialization-contract revision: paired-dimension marshalling-boundary enforcement.
- [`ontology-revision-layer-b-deep-criterion`](./ontology-revision-layer-b-deep-criterion.md) — Layer B form RFC (accepted at §0135).
- [`operational-spec-layer-b-parameter-calibration`](./operational-spec-layer-b-parameter-calibration.md) — Layer B parameter calibration RFC (accepted at §0138).
- [`layer-b-service-tier-implementation-evidence.md`](../discussion/layer-b-service-tier-implementation-evidence.md) — Discussion-phase evidence scratch for this RFC.
- [`docs/architecture/canonical-serialization-contract.md`](../../architecture/canonical-serialization-contract.md) §Demotion-Candidacy Predicate — the contract layer this RFC's implementation operationally discharges.
- [`docs/ontology/lifecycle-semantics.md`](../../ontology/lifecycle-semantics.md) §The Promotion Mechanism — the staged-combination semantic this RFC implements at the service tier.
- [`services/ingestion/internal/hypothesis/demotion.go:97-99`](../../../services/ingestion/internal/hypothesis/demotion.go) — the stale comment whose refresh is a downstream consequence.

## Decision Record

Status: discussion. Resolution will be recorded here at acceptance time, with reference to the decision-log entry that adopts a final implementation specification across the six sub-decisions.
