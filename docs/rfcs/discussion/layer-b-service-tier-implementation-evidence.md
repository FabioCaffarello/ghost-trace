# Layer B Service-Tier Implementation — operational-specification discussion evidence

**Status:** in-discussion. Not authoritative. Final resolution will be recorded in decision-log and operational-spec document revision.

This scratch supports the discussion phase of [`operational-spec-layer-b-service-tier-implementation`](../draft/operational-spec-layer-b-service-tier-implementation.md). The RFC opens per [`§0138`](../../charter/decision-log.md) Consequences carry-forward ("service-tier implementation now openable as ordinary RFC discipline") as the next operational-specification follow-on after the form RFC ([`ontology-revision-layer-b-deep-criterion`](../draft/ontology-revision-layer-b-deep-criterion.md), accepted at [`§0135`](../../charter/decision-log.md)) and the parameter-calibration RFC ([`operational-spec-layer-b-parameter-calibration`](../draft/operational-spec-layer-b-parameter-calibration.md), accepted at [`§0138`](../../charter/decision-log.md)).

The form-level resolution (Layer B = L-BC-OR with L-C structural-exclusion commitment) lands at [`§0135`](../../charter/decision-log.md); the canonical-serialization-contract crystallization (α + τ + β-graph + L-BC-OR firing predicate) lands at [`§0136`](../../charter/decision-log.md); the parameter values land at [`§0138`](../../charter/decision-log.md). This RFC specifies the **implementation strategy** at the service tier — where the evaluation lives in the call chain, how the predicate is computed from substrate, what artifact (if any) is produced, and how existing demote CLIs interact with the evaluation.

This is a strictly-framing scratch: Phase 1 names the dependency surface and the inception-phase posture; Phase 2 enumerates sub-decision-by-sub-decision candidate variants. Phases 3+ (epistemic-skill application, comparison synthesis, recommendation) are drafted in subsequent commits.

---

## Phase 1 — Scope and dependencies

### The question

Per [`§0135`](../../charter/decision-log.md) Decision + [`§0136`](../../charter/decision-log.md) Decision + [`§0138`](../../charter/decision-log.md) Decision, the full demotion-candidacy predicate at the canonical-serialization-contract layer is:

> `DEMOTE-CANDIDATE(H) := Layer A(H) AND ((freshness_B(H) < T_B) OR (saturation_C(H) > K_C))`

with `T_B = K_C = 0.5`; `N = 1000`; W-count window; U-uniform per-subtype; `N_A = 1 day` (bundled).

The service tier — currently the `services/ingestion/` Go module — does not evaluate this predicate. It evaluates only Layer A (cadence gate) and reports it as an advisory field (`CadenceSatisfied`) in `DemoteReport`, recording the demotion regardless of the field's value. The CLI surface is operator-elected: an operator invokes `demote-hypothesis` (or one of four subtype-specific variants) with a target promotion hash + reason + actor; the service appends the demotion event to substrate. Layer B does not enter the call chain.

This RFC asks: **How should the service tier evaluate Layer B's predicate? Where in the call chain does the evaluation live? What artifact (if any) does the evaluation produce? How does the existing demote CLI surface interact with the predicate's verdict?**

### Inception-phase posture

[`§0023`](../../charter/decision-log.md) (Q2 — Identity tiers) established the inception-phase posture: when a simpler form is structurally sufficient and the multi-tier variant has no current empirical justification, the simpler form is adopted with reversal conditions documenting when the multi-tier variant would be revisited. This RFC applies the same posture: the service is at inception; there are no promoted hypotheses in production today; the demote CLIs are operational-but-untested-at-scale; operational complexity (separate services, projection storage, scheduled jobs) is unjustified unless structurally required.

The corollary: this RFC's recommendation should default to the simpler implementation form when candidates differ only in operational complexity, AND should document the reversal conditions (what evidence would surface that justifies the more complex form).

### Dependency surface

**Discharged dependencies** (no further work required):

- **Layer B form**: [`§0135`](../../charter/decision-log.md) — L-BC-OR + L-C structural-exclusion. ✓
- **Predicate structural surface**: [`§0136`](../../charter/decision-log.md) — full firing predicate crystallized at canonical-serialization-contract; type/range enforced via `LayerBParameters` proto. ✓
- **Parameter values**: [`§0138`](../../charter/decision-log.md) — T_B = K_C = 0.5; N = 1000; W-count; U-uniform; N_A = 1 day. ✓
- **Measurable quantity `evidential_independence`**: [`§0133`](../../charter/decision-log.md) Q3-α — source-count ratio over Cat I provenance roots. ✓
- **Transitive closure storage**: [`§0134`](../../charter/decision-log.md) Q5-τ + [`§0136`](../../charter/decision-log.md) β-graph storage — substrate stores direct `influenced_by` edges + per-record cached closures via `closure_hashes` field. ✓
- **Closure marshalling-boundary discipline**: [`§0139`](../../charter/decision-log.md) — hash-list element-shape discipline at the full canonical-form set. ✓
- **Paired-dimension marshalling-boundary enforcement**: [`§0140`](../../charter/decision-log.md) — `evidential_independence` presence enforced on records subject to the paired-dimension commitment. ✓

**Open dependencies** (this RFC's scope):

- **Implementation locus** — where does the evaluation function live as Go code?
- **Computation strategy** — how is the predicate computed from substrate at evaluation time?
- **Window stream identity** — N=1000 of WHAT events?
- **Output shape** — does evaluation produce a substrate record?
- **CLI integration** — how does demote-hypothesis (and 4 subtype variants) interact with Layer B?
- **N_A=1 day source** — where does the Layer A cadence parameter come from at the service tier?

These six form the sub-decision surface enumerated in Phase 2.

### Implementation surface inventory

Empirical audit confirms the following implementation surface (as of [`§0140`](../../charter/decision-log.md) + PR #150 merge):

- **Active services**: `services/ingestion/` only. 193 Go files; 38 CLI commands; internal packages for canonical serialization, derivation, hypothesis lifecycle ops, ingestion, orphan-cleanup, projection helpers, replay helpers, substrate.
- **README-only services**: `services/assertion-engine/`, `services/graph/`, `services/projections/`, `services/replay/`. Zero Go code.
- **Existing demote surface**: `services/ingestion/internal/hypothesis/demotion.go` carries the generic Demote function (BehavioralCluster-targeted at the implementation surface; the four subtype-specific variants — automation_group_demotion.go, behavioral_cluster_demotion.go, campaign_hypothesis_demotion.go, coordination_ring_demotion.go — mirror this shape per Q2-A.2). CLI surface: `cmd/demote-hypothesis/main.go` + four subtype-specific variants.
- **Layer B-related proto types**: `schemas/common/v1/layer_b_parameters.proto` defines `LayerBParameters` with `t_b`, `k_c`, `window_size_n` (N), and `inactivity_window_seconds` (N_A). `schemas/common/v1/evidential_independence.proto` defines the rational-pair structural surface. Both are imported into the four formation protos + OperationalSession per [`§0136`](../../charter/decision-log.md).
- **Closure-storage shape**: All four formation protos (`automation_group_formation.proto`, `behavioral_cluster_formation.proto`, `campaign_hypothesis_formation.proto`, `coordination_ring_formation.proto`) declare `closure_hashes` field per [`§0136`](../../charter/decision-log.md) β-graph storage. Marshalling-boundary discipline enforced per [`§0139`](../../charter/decision-log.md) + [`§0140`](../../charter/decision-log.md).

### Out-of-scope

Three candidate forms surfaced during framing and explicitly rejected as out-of-scope per the form-vs-parameter-vs-implementation discipline:

1. **Stand-alone scheduled-job-only evaluation.** Cron-style scheduler that periodically scans all promoted hypotheses and emits `LayerBEvaluation` records, with no CLI integration. Rejected: no scheduling primitive in `services/ingestion/` today; introducing one ahead of need is premature.
2. **GraphQL-style query layer over Layer B state.** Exposing Layer B evaluation via a query API. Rejected: no query layer in `services/ingestion/` today; introducing one is an architecture decision (services/graph or services/projections), not an operational-spec decision.
3. **Multi-criterion composition beyond §0135 L-BC-OR.** Per-subtype OR per-deployment overrides on the L-BC-OR firing predicate. Rejected: [`§0135`](../../charter/decision-log.md) committed to L-C structural-exclusion; firing predicate is structurally fixed at L-BC-OR; per-subtype divergence was explicitly resolved as U-uniform at [`§0138`](../../charter/decision-log.md). Re-opening either would require form-level RFC.

---

## Phase 2 — Sub-decision candidate enumeration

### Sub-decision A — Evaluation locus

> Where does the Layer B predicate evaluation function live in the Go codebase?

| Candidate | Locus | Sketch |
|---|---|---|
| A1 | `services/ingestion/internal/hypothesis/layerb/` | New internal package, sibling to the lifecycle-operation files. Pure functions: `EvaluateLayerB(ctx, sub, promotionEventHash, params) (LayerBVerdict, error)`. Called by demote-hypothesis (and 4 subtype variants) via the existing Demote function family; called by any future candidate-finder CLI. |
| A2 | `services/assertion-engine/` | Promote the README-only service to a real service with Layer B as its first feature. Service boundary at the network; ingestion calls assertion-engine via RPC or in-process import. |
| A3 | Inline in each demote CLI | Each of 5 demote CLIs (demote-hypothesis + 4 subtype variants) carries the Layer B computation inline in `main.go`. No shared evaluator. |

**Conservative-defaults reading**: A1 is the simplest form respecting DRY. A2 introduces a service boundary ahead of need (cost: deployment topology, RPC type definitions, cross-service auth). A3 violates DRY across 5 implementation sites.

### Sub-decision B — Computation strategy

> How is the predicate's two components (`freshness_B`, `saturation_C`) computed from substrate at evaluation time?

| Candidate | Strategy | Sketch |
|---|---|---|
| B1 | On-the-fly from substrate | Per-evaluation: read promotion event + closure_hashes; scan substrate window (N=1000 events) under sub-decision C's identity; compute fresh-roots-count and saturation-ratio; return verdict. No persistent state. |
| B2 | Cat II projection per promoted hypothesis | Maintain a continuously-refreshed projection record per promoted hypothesis (freshness_B(H), saturation_C(H), last-refreshed-at, last-input-event-hash). Read projection at evaluation time; rebuild from substrate on demand per the projection-rebuildability discipline (`projection-model.md`). |
| B3 | On-the-fly with substrate audit record | Compute on-the-fly per evaluation (as B1); additionally emit a `LayerBEvaluation` Cat II substrate record per evaluation cycle, capturing the computed values + window range. Audit trail without per-hypothesis projection. |

**Conservative-defaults reading**: B1 is the simplest form, matching the existing demote-hypothesis pattern (no persistent state; computation is transient). B2 introduces projection-storage cost ahead of need. B3 introduces substrate-write cost per evaluation (potentially per demote attempt) — schemas-evolution event required if adopted.

### Sub-decision C — W-count N=1000 stream identity

> Per [`§0138`](../../charter/decision-log.md), N=1000 substrate events under W-count form. N=1000 of WHAT?

| Candidate | Stream identity | Sketch |
|---|---|---|
| C1 | Substrate global | Last 1000 substrate events in commit order across all categories + subtypes + identities. Predicate is computed against a global window. |
| C2 | Per-hypothesis closure | Last 1000 events in this hypothesis's `closure_hashes` set (transitive `influenced_by` closure under τ). Predicate is per-hypothesis-localized. |
| C3 | Per-subtype | Last 1000 events of this hypothesis's concrete subtype (e.g., last 1000 BehavioralCluster events). Predicate is subtype-localized. |
| C4 | Since-promotion | All substrate events committed after this hypothesis's promotion event up to capped N=1000. Predicate is post-promotion-window-localized. |

**Form-vs-parameter respect**: All four candidates respect the W-count form fixed at [`§0138`](../../charter/decision-log.md). The choice is implementation-level — how to interpret N=1000's stream identity in code. A future implementation RFC may revise this without re-opening the form RFC.

**Operational-semantics asymmetry**: C2 (per-hypothesis closure) most directly matches the predicate's semantic intent — `freshness_B` asks "how fresh are the Cat I roots underlying this hypothesis's evidential support", which is a per-hypothesis question. C1 (substrate global) is operationally simpler but semantically diffuse. C3 (per-subtype) is between the two. C4 (since-promotion) is well-defined but conflates "freshness" with "time-since-promotion" and may behave erratically for old promoted hypotheses (window could be enormous).

### Sub-decision D — Output shape

> What artifact does Layer B evaluation produce?

| Candidate | Artifact | Sketch |
|---|---|---|
| D1 | Transient DemoteReport extension | No substrate record. Layer B verdict surfaces only in the in-memory `DemoteReport` returned to the CLI invocation, mirroring the existing `CadenceSatisfied` field. Operator sees the verdict in CLI output; no audit trail. |
| D2 | `LayerBEvaluation` Cat II substrate record | New Cat II proto: `LayerBEvaluation` with fields (target_promotion_event_hash, freshness_b, saturation_c, t_b_threshold, k_c_threshold, window_size_n, window_form, layer_b_fired, evaluation_event_hash). Committed to substrate per evaluation. Audit trail; substrate-grounded. |
| D3 | `DemotionCandidacyEvaluation` Cat II composite | New Cat II proto combining Layer A + Layer B verdicts in one substrate record per candidacy evaluation. Audit trail; composite. |

**Schemas-evolution-event scope**: D1 is contained within the existing proto set under `schemas/`; no schemas-evolution event. D2 and D3 are schemas-evolution events per the canonical-serialization-contract §Schemas-Evolution Events boundary; if adopted, they may need their own follow-on schemas-evolution RFC at resolution time.

**Conservative-defaults reading**: D1 matches the existing Layer A pattern (advisory in DemoteReport, not committed as a record). D2 and D3 add audit-trail durability but cost substrate-write per evaluation. The cost-justification depends on operational scale (number of evaluations per day × storage cost per record); at inception phase, this cost is unknown.

### Sub-decision E — demote-hypothesis interaction

> How does the existing demote-hypothesis CLI (and 4 subtype variants) interact with the Layer B verdict?

| Candidate | Interaction | Sketch |
|---|---|---|
| E1 | Advisory, like Layer A | Demote CLI evaluates Layer B (via the chosen locus per A); adds `LayerBFired`, `LayerBFreshnessB`, `LayerBSaturationC` flags to DemoteReport; demote proceeds and commits regardless of the verdict. Operator sees the state post-facto. |
| E2 | Enforcing refusal with override | Demote CLI evaluates Layer B; if `Layer B(H)` is false (predicate did not fire), CLI refuses to commit the demotion unless an explicit `--force-layer-b-bypass` option is supplied. The override is recorded in the demotion reason for audit. |
| E3 | Untouched; separate candidate-finder | demote CLIs remain untouched (pure operator-elected substrate commit). A new `cmd/find-demotion-candidates/` CLI consults Layer B across all promoted hypotheses and produces a candidate list. Operators use the candidate-finder to identify demotion candidates, then invoke demote-hypothesis manually. |

**§0011 Layer A pattern precedent**: Layer A is advisory per [`§0011`](../../charter/decision-log.md) staged-combination — it's a CANDIDACY criterion, not a barrier. E1 directly mirrors this pattern for Layer B. E2 elevates Layer B from candidacy to barrier, which is a structural commitment beyond §0011. E3 leaves the existing CLIs untouched and adds Layer B as a separate query tool.

**Operator-authority asymmetry**: Per [`§1 Thesis`](../../charter/constitutional-charter.md#1-thesis) and the substrate-writer authority pattern, operator decisions to commit substrate records are the source-of-truth. E1 and E3 preserve this fully; E2 introduces a service-side commitment that conditions operator action on a service-computed verdict — operationally appropriate only with the override mechanism.

### Sub-decision F — N_A=1 day Layer A cadence source

> Per [`§0138`](../../charter/decision-log.md), N_A=1 day is bundled into LayerBParameters. The existing promote-hypothesis CLI accepts `-cadence-seconds` (Layer A advisory). Where does the service tier read N_A from?

| Candidate | Source | Sketch |
|---|---|---|
| F1 | Bundled in LayerBParameters | Service reads N_A directly from the promoted hypothesis's `LayerBParameters.inactivity_window_seconds` field. promote-hypothesis CLI defaults `-cadence-seconds` from this field. Substrate-grounded. |
| F2 | Separate Layer A config | LayerBParameters carries Layer B parameters only (T_B, K_C, N). Layer A cadence is a separate operator-supplied config (per CLI invocation) OR a separate substrate-committed Layer A config record. |
| F3 | CLI operator-supplied with bundled defaults | promote-hypothesis CLI continues to accept `-cadence-seconds`; defaults from `LayerBParameters.inactivity_window_seconds` if not supplied. Backward-compatible; respects [`§0138`](../../charter/decision-log.md) bundling. |

**§0138 bundling reading**: [`§0138`](../../charter/decision-log.md) explicitly bundled `N_A = 1 day` into `LayerBParameters` rather than maintaining a separate Layer A config, on inception-phase simplicity grounds. F1 honors this bundling fully; F2 reverses it. F3 honors the bundling while preserving the existing CLI surface backward-compatibly.

**Pre-existing CLI surface**: promote-hypothesis already accepts `-cadence-seconds`. F1 changes the contract (CLI no longer needs `-cadence-seconds`); F3 preserves the existing option with a default-from-substrate fallback. F2 is the most disruptive.

---

## Cross-decision interaction matrix

The six sub-decisions interact in known ways:

- **A1 + B1 + D1 + E1 + F3**: Conservative-defaults bundle. Internal package; on-the-fly; transient; advisory; existing CLI preserved. Simplest implementation form respecting form-vs-parameter-vs-implementation discipline. No schemas-evolution event. No new substrate record types. No service-architecture changes.
- **A2 + B2 + D2 + E2 + F1**: Aggressive bundle. Separate service; projection storage; substrate audit record; enforcing refusal; bundled-only config source. Largest blast radius; schemas-evolution event(s) required; service-architecture change; backward-compatibility risk.
- **A1 + B3 + D2**: Mixed. Internal package + on-the-fly + audit record. Schemas-evolution event but no service-architecture change.
- **E3 + (any A/B/D/F)**: Decouples Layer B evaluation from demote CLI; useful if demote CLI behavior is contentious.

The Phase 3 analysis will produce a per-sub-decision recommendation; the Phase 4 synthesis will check the recommendations against the interaction matrix for consistency.

---

## Phase 3 — Apply epistemic skills

Per the operational-spec Phase 3 precedent from [`layer-b-parameter-calibration-evidence.md`](./layer-b-parameter-calibration-evidence.md) Phase 3 (the "lighter Phase 3" methodological observation 9): three skills ([`falsifiability-check`](../../../.claude/skills/epistemic/falsifiability-check/SKILL.md), [`epistemic-separator`](../../../.claude/skills/epistemic/epistemic-separator/SKILL.md), [`ambiguity-reducer`](../../../.claude/skills/epistemic/ambiguity-reducer/SKILL.md)) applied. The structural form is fixed at [`§0135`](../../charter/decision-log.md); the parameter values are fixed at [`§0138`](../../charter/decision-log.md); the canonical-serialization-contract enforces type/range at the marshalling boundary per [`§0136`](../../charter/decision-log.md) + [`§0140`](../../charter/decision-log.md). The epistemic-skill questions focus on posture-fit at inception phase rather than on structural admissibility.

### Sub-decision A — Evaluation locus

| Candidate | `falsifiability-check` | `epistemic-separator` | `ambiguity-reducer` |
|---|---|---|---|
| **A1 internal package** (`internal/hypothesis/layerb`) | Pass. The evaluation function's behavior is mechanically observable via unit tests over fixture substrate state; outputs (`LayerBVerdict` shape) are deterministic given inputs. | Pass per category boundary. The function operates over `closure_hashes` (Cat I substrate facts per [`§0021`](../../charter/decision-log.md)) + `evidential_independence` (Cat III paired-dimension per [`§0140`](../../charter/decision-log.md)); the read crosses categories but does not mix them at the assertion level. | Pass. Package locus and function signature are mechanically specified. |
| **A2 separate service** (`assertion-engine`) | Pass at form-level. Service-boundary call is observable but introduces RPC-call testability burden. | **Risk:** introduces a service-boundary separation between substrate read (ingestion) and Layer B evaluation (assertion-engine); the read-and-evaluate path is split across two services. Operationally admissible but the boundary requires structural justification not currently available. | **Ambiguity surfaced on service-boundary semantics.** Inception-phase has zero service-to-service calls in `services/`; introducing the pattern requires its own design decision (sync vs async, in-process import vs out-of-process RPC, type-definition versioning across services). Response 3 — raise as a structural decision under a separate architecture RFC if A2 is preferred. |
| **A3 inline per CLI** | Pass at form-level. Each CLI's behavior is observable independently. | **Risk:** the same Layer B predicate is computed in 5+ CLI bodies; per [CLAUDE.md §7](../../../.claude/CLAUDE.md) constitutional minimalism, this is ceremony without behavioral consequence — five sites with the same logic offer no separation-of-concerns benefit. | **Ambiguity surfaced on DRY violation.** The inlining is operationally arbitrary; no per-CLI reason exists for the duplicated logic. |

### Sub-decision B — Computation strategy

| Candidate | `falsifiability-check` | `epistemic-separator` | `ambiguity-reducer` |
|---|---|---|---|
| **B1 on-the-fly from substrate** | Pass. Per-evaluation computation is deterministic given (a) the promoted hypothesis's `closure_hashes` set + (b) the recent-N substrate window. Falsifiability via substrate-replay over the same window. | Pass. Computation reads substrate facts (per [`§0021`](../../charter/decision-log.md)) without committing new substrate records; the read does not invert the Cat I / Cat II / Cat III layering. | Pass. The computation is well-defined; the recent-N window is per sub-decision C. |
| **B2 Cat II projection per promoted hypothesis** | Pass at form-level. The projection is deterministically rebuildable from substrate per [`projection-model.md`](../../architecture/projection-model.md). | Pass per category boundary. The projection is Cat II derived from Cat I + Cat III substrate facts; the projection-vs-substrate distinction per [§2.1 Boundary Conditions](../../charter/constitutional-charter.md#21-observational-integrity) is respected. | **Ambiguity surfaced on refresh cadence.** A continuously-refreshed projection requires a cadence specification (every event, every N events, every time slice). The cadence is itself a sub-sub-decision the projection variant would have to specify. Response 3 — raise as deferred sub-question. |
| **B3 on-the-fly with substrate audit record** | Pass at form-level. The audit record is a Cat II substrate event committed alongside evaluation. | Pass per category boundary. The audit record (Cat II) records the evaluation's inputs + outputs deterministically. | **Ambiguity surfaced on audit-record emission cadence.** The audit record is emitted "per evaluation cycle" — but the cycle definition (per CLI invocation, per scheduled job tick, per user query) is undefined. Response 3 — raise as deferred sub-question. |

### Sub-decision C — W-count N=1000 stream identity

| Candidate | `falsifiability-check` | `epistemic-separator` | `ambiguity-reducer` |
|---|---|---|---|
| **C1 substrate-global** | Pass. Matches the canonical-serialization-contract's [`§Demotion-Candidacy Predicate`](../../architecture/canonical-serialization-contract.md) explicit form: `freshness_B(H) = avg(evidential_independence(r) OVER recent N assertions r WHERE H.hash ∈ r.closure_hashes)` — "recent N assertions" is the global universe; the `WHERE` clause is a filter, not a window-identity redefinition. `saturation_C(H)`'s divisor is `N`, confirming the global count of recent assertions, not the filtered count. **Structural reading from the contract.** | Pass. Substrate-commit-order is the substrate's native ordering per [`§0024`](../../charter/decision-log.md) + [`§0027`](../../charter/decision-log.md); no parallel temporal axis introduced. | Pass. Window is deterministic. |
| **C2 per-hypothesis closure** | **FAIL — violates canonical-serialization-contract `saturation_C` definition.** With per-hypothesis-closure window, every assertion in the window is by definition in H's closure; `saturation_C` would always be 1.0 (or its L-C-excluded equivalent), defeating the predicate's purpose. The contract's divisor `N` is the global count, not the filtered count. | Fail. Recasts the predicate's semantic from "fraction of recent global assertions in H's downstream" to "fraction of H's downstream that's recent" — a structurally different question. | N/A — rejected at falsifiability. |
| **C3 per-subtype** | **FAIL — violates canonical-serialization-contract `recent N` reading.** The contract's `recent N assertions` is unfiltered at the window-selection step; per-subtype filtering would be an additional `WHERE` clause not present in the contract's predicate definition. | Fail. Introduces per-subtype window-identity not motivated by the abstract `Hypothesis` lifecycle per [`§0010`](../../charter/decision-log.md) Q2-A.2. | N/A — rejected at falsifiability. |
| **C4 since-promotion** | **FAIL — violates W-count form.** The contract's W-count window per [`§0138`](../../charter/decision-log.md) is "the last N assertions by substrate-commit order" — a fixed-count window, NOT a since-event window. C4 conflates W-count with a since-promotion-bounded selection that is structurally different. | Fail. Introduces an event-anchored window not supported by W-count semantics. | N/A — rejected at falsifiability. |

**Methodological observation surfaced.** Sub-decision C was over-framed in Phase 2 — the canonical-serialization-contract [`§Demotion-Candidacy Predicate`](../../architecture/canonical-serialization-contract.md) section EXPLICITLY defines the window stream as substrate-global with `WHERE H.hash ∈ r.closure_hashes` as a downstream filter, not as a window-identity definition. C2/C3/C4 are structurally precluded, parallel to the T_B-derived structural-preclusion finding from [`layer-b-parameter-calibration-evidence.md`](./layer-b-parameter-calibration-evidence.md) Phase 3 sub-decision 1. **Future operational-spec RFCs should re-check the canonical-serialization-contract for structural determination before enumerating implementation-tier candidates.**

### Sub-decision D — Output shape

| Candidate | `falsifiability-check` | `epistemic-separator` | `ambiguity-reducer` |
|---|---|---|---|
| **D1 transient DemoteReport extension** | Pass. The verdict is observable in CLI output; falsifiability via per-invocation behavior. | Pass. Matches the existing Layer A pattern: `CadenceSatisfied` is a struct-field in DemoteReport, transient, not substrate-committed. | Pass. The DemoteReport extension is a mechanical Go struct addition. |
| **D2 LayerBEvaluation Cat II substrate record** | Pass at form-level. The substrate record is observable per [§2.1 frozen](../../charter/constitutional-charter.md#21-observational-integrity) post-commit. | Pass per category boundary. Cat II derived-from-Cat-I-and-Cat-III evaluation is structurally well-defined. | **Ambiguity surfaced on schemas-evolution event scope.** Adding `LayerBEvaluation` proto requires its own schemas-evolution event RFC per the canonical-serialization-contract §Schemas-Evolution Events boundary; the schemas-evolution RFC must specify the proto's fields, version, and corpus impact. Response 3 — surface as triggering follow-on RFC. |
| **D3 DemotionCandidacyEvaluation Cat II composite** | Same form-level pass as D2. | Same as D2 (composite Cat II record). | Same schemas-evolution-event surfacing; additionally, the composite-vs-separate choice (Layer A + Layer B combined record vs separate records) is itself a sub-sub-decision. |

### Sub-decision E — demote-hypothesis interaction

| Candidate | `falsifiability-check` | `epistemic-separator` | `ambiguity-reducer` |
|---|---|---|---|
| **E1 advisory like Layer A** | Pass. The advisory pattern is observable in DemoteReport; the demote commit semantic is unchanged. | Pass. Matches the [`§0011`](../../charter/decision-log.md) staged-combination Layer A precedent: Layer A is advisory in DemoteReport, not enforcing. Operator authority over substrate commit preserved per the substrate-writer pattern. | Pass. The advisory field semantic is mechanical. |
| **E2 enforcing refusal with override** | Pass at form-level. The refusal path is observable; the override is observable in the demotion reason. | **Risk:** elevates Layer B from CANDIDACY (per [`§0011`](../../charter/decision-log.md) staged-combination) to BARRIER. Per [`§0011`](../../charter/decision-log.md) Q4 resolution, Layer A + Layer B is a CANDIDACY criterion, not a barrier; operator-elected demotion is the substrate-commit primitive. E2 requires an override mechanism to preserve operator authority — admissible but requires a structural commitment beyond [`§0011`](../../charter/decision-log.md). | **Ambiguity surfaced on override semantics.** The `--force-layer-b-bypass` option must be specified: does it require additional justification text? Is it audit-logged separately? Does it require a higher operator-permission level? Response 3 — raise as deferred sub-question. |
| **E3 untouched separate candidate-finder** | Pass. New CLI is observable independently; existing CLIs unchanged. | Pass per category boundary. The candidate-finder is a Cat III-reading utility; demote CLIs remain pure substrate-commit. | Pass. Operationally clean separation — Layer B is a query tool, demote-hypothesis is a substrate-commit tool. |

### Sub-decision F — N_A=1 day Layer A cadence source

| Candidate | `falsifiability-check` | `epistemic-separator` | `ambiguity-reducer` |
|---|---|---|---|
| **F1 bundled in LayerBParameters** | Pass. Reading from `LayerBParameters.inactivity_window_seconds` per [`§0138`](../../charter/decision-log.md) bundling is mechanically observable. | Pass. Substrate-grounded read; no parallel config source. | **Ambiguity surfaced on CLI surface change.** Removing `-cadence-seconds` from promote-hypothesis is a breaking change to the existing CLI contract; backward-compatibility burden is non-trivial. Response 3 — raise as deferred sub-question. |
| **F2 separate Layer A config** | Pass at form-level. | **Risk:** reverses the [`§0138`](../../charter/decision-log.md) N_A bundling decision. The bundling was committee-decided on inception-phase simplicity grounds; unbundling without empirical justification is committee-revisited. | N/A — rejected at epistemic-separator. |
| **F3 CLI operator-supplied with bundled defaults** | Pass. CLI default-from-substrate is mechanically observable; operator override preserved. | Pass. Honors [`§0138`](../../charter/decision-log.md) bundling at the substrate layer while preserving CLI surface compatibility. | Pass. Default-from-substrate with operator override is a standard CLI pattern. |

### Most consequential epistemic finding across the matrix

**Primary finding — sub-decision C is structurally determined by the canonical-serialization-contract, not by implementation choice.** The contract's [`§Demotion-Candidacy Predicate`](../../architecture/canonical-serialization-contract.md) section explicitly defines `recent N assertions` as the substrate-global universe and `WHERE H.hash ∈ r.closure_hashes` as a downstream filter; the divisor in `saturation_C` is `N` (global count). C2/C3/C4 are structurally precluded; C1 is the only admissible candidate. **Sub-decision C resolves to C1 by structural determination from the upstream contract — not by inception-phase posture.**

**Secondary finding — E2 enforcing refusal requires structural commitment beyond [`§0011`](../../charter/decision-log.md).** Per [`§0011`](../../charter/decision-log.md) Q4 resolution, Layer A + Layer B is a CANDIDACY criterion; E2 elevates it to BARRIER. Admissible only with an override mechanism preserving operator authority. The override mechanism is a sub-sub-decision not in framing scope — E2's adoption would require its own follow-on specification.

**Tertiary finding — F2 (separate Layer A config) reverses [`§0138`](../../charter/decision-log.md) N_A bundling.** Bundling was committee-decided on inception-phase grounds; reversing it requires committee-revisit, not implementation choice. F2 is structurally precluded at this RFC's scope; revisit requires a separate operational-spec RFC.

**Quaternary finding — D2/D3 require schemas-evolution event follow-on RFC.** Adding `LayerBEvaluation` or `DemotionCandidacyEvaluation` proto types is a schemas-evolution event per the canonical-serialization-contract boundary item 9. Adoption of D2 or D3 triggers a follow-on schemas-evolution RFC; D1 is contained within the existing proto set.

### Calibration carry-forward to future operational-spec RFCs

Layer-B-service-tier-implementation Phase 3 confirms and extends the parameter-calibration Phase 3 methodological observations:

- **Confirmed: operational-spec RFCs admit a lighter Phase 3** (per parameter-calibration Phase 3 MO-9). Three skills (falsifiability-check + epistemic-separator + ambiguity-reducer) are sufficient; the structural form is upstream-fixed; the implementation-tier focus is posture-fit + inheritance-conflict surfacing.
- **New observation — Sub-decision over-framing is detectable at Phase 3 via canonical-serialization-contract re-reading.** Sub-decision C was over-framed in Phase 2 (4 candidates); the contract's explicit predicate definition reduces it to 1 admissible candidate at Phase 3. **Future operational-spec RFCs should re-check the canonical-serialization-contract for structural determination of each sub-decision before Phase 3 begins.** Pattern parallel to T_B-derived structural-preclusion from parameter-calibration Phase 3 sub-decision 1, but at implementation tier rather than parameter tier.
- **New observation — Inheritance conflict between framing-PR enumeration and committee-decision precedents is detectable at Phase 3 via epistemic-separator.** F2 (separate Layer A config) reverses [`§0138`](../../charter/decision-log.md) bundling — visible only by reading the predecessor decision-log entry. **Future operational-spec RFCs should explicitly inventory predecessor committee decisions during Phase 1 to avoid framing candidates that revisit them.**

## Phase 4 — Comparison synthesis

Findings synthesized from Phase 1 (dependency surface + inception-phase posture + implementation surface inventory at §0140 commit), Phase 2 (sub-decision candidate enumeration with sketches and conservative-defaults readings), and Phase 3 (epistemic-skill matrix). Classified as **asymmetry** / **apparent trade-off that resolves** / **genuine trade-off** / **structural determination** / **tension**. Numbered in order of consequence.

### Finding 1 — Structural determination: sub-decision C resolves by canonical-serialization-contract reading, not by implementation choice

Per Phase 3 primary finding: the canonical-serialization-contract's [`§Demotion-Candidacy Predicate`](../../architecture/canonical-serialization-contract.md) section explicitly defines `recent N assertions` as the substrate-global universe; C1 is the only admissible candidate. **Sub-decision C exits the open-decision space at this Phase 3 finding; the framing PR's 4-candidate enumeration was over-framed.**

### Finding 2 — Asymmetry: inception-phase posture favors conservative-defaults across the remaining sub-decisions

Per Phase 1's inception-phase posture commitment + [`§0022`](../../charter/decision-log.md) empirical-pressure-phase discipline + [`§0023`](../../charter/decision-log.md) inception-phase precedent + [`§0138`](../../charter/decision-log.md) inception-phase parameter values: the system has zero promoted hypotheses in production, zero observed Layer B evaluations, and zero empirical pressure on the implementation tier. The conservative-defaults bundle across the remaining 5 sub-decisions is A1 + B1 + D1 + E1 + F3 — internal package; on-the-fly computation; transient DemoteReport extension; advisory-like-Layer-A; CLI operator-supplied with bundled defaults. Each conservative-default minimizes the operational surface added at this RFC's resolution.

### Finding 3 — Asymmetry: A1 + B1 + D1 + E1 align with [`§0011`](../../charter/decision-log.md) staged-combination Layer A pattern precedent

Per Phase 3 sub-decision A (A1), sub-decision B (B1), sub-decision D (D1), and sub-decision E (E1): the existing Layer A handling at the service tier is exactly the conservative-defaults bundle. Layer A is computed on-the-fly (no projection); the verdict is transient (in DemoteReport, not substrate-committed); it's advisory (demote records regardless); it lives in an internal package (the cadence helper functions in `internal/hypothesis/`). **Adopting the same pattern for Layer B inherits the established service-tier discipline from [`§0011`](../../charter/decision-log.md) Layer A — no new pattern is introduced.**

### Finding 4 — Apparent trade-off that resolves: E2 (enforcing) elevates Layer B to barrier, requires beyond-§0011 commitment

Per Phase 3 sub-decision E secondary finding: E2 elevates Layer B from CANDIDACY (per [`§0011`](../../charter/decision-log.md)) to BARRIER. The apparent benefit (stronger structural defense) does not survive the [`§0011`](../../charter/decision-log.md) staged-combination reading — Layer A + Layer B is candidacy, not barrier; operator-elected demotion is the substrate-commit primitive. **E2's adoption would require a separate structural commitment beyond [`§0011`](../../charter/decision-log.md), which is not in this RFC's scope. The trade-off resolves toward E1 at inception, with E2 available as empirical-pressure-phase reversal option if operator-elected demotion patterns surface that warrant barrier-mode.**

### Finding 5 — Structural precluusion: F2 (separate Layer A config) reverses [`§0138`](../../charter/decision-log.md) bundling

Per Phase 3 sub-decision F tertiary finding: F2 reverses the [`§0138`](../../charter/decision-log.md) N_A bundling decision. The bundling was committee-decided on inception-phase simplicity grounds; reversing it requires committee-revisit, not implementation choice. **F2 exits the admissible-candidate space; F1 vs F3 is the remaining choice.**

### Finding 6 — Genuine trade-off: F1 vs F3 — CLI surface compatibility vs strict bundling honor

Per Phase 2 sub-decision F + Phase 3 sub-decision F: F1 (bundled in LayerBParameters) honors [`§0138`](../../charter/decision-log.md) bundling most directly but removes the existing `-cadence-seconds` CLI option from promote-hypothesis (breaking change). F3 (CLI operator-supplied with bundled defaults) honors the bundling at the substrate layer while preserving CLI surface compatibility. **The trade-off is committee-judgment on whether strict bundling honor or CLI surface compatibility matters more at inception**. The conservative-defaults reading favors F3 — backward compatibility is the more conservative choice; F1's break is justifiable only with operator-experience evidence that the existing option is harmful, which is not available at inception.

### Finding 7 — Schemas-evolution event scope: D1 contained, D2/D3 triggers follow-on RFC

Per Phase 3 sub-decision D quaternary finding: D1 is contained within the existing proto set; D2/D3 are schemas-evolution events per the canonical-serialization-contract §Schemas-Evolution Events boundary. Adopting D2 or D3 triggers a follow-on schemas-evolution RFC (proto field specification, version bumping, corpus impact, golden regeneration). **At inception phase, the schemas-evolution surface is unjustified — there is no audit-trail requirement yet that D1's transient form fails to meet; D1 is the conservative choice. D2/D3 reversal triggers: audit-trail requirement surfaces (e.g., regulatory, operational forensics); operator workflow demands persistent Layer B evaluation history.**

### Finding 8 — Methodological observation: phase 3 sub-decision over-framing detection

Per Phase 3 calibration carry-forward observation 2: the framing PR's sub-decision C over-enumeration was detectable only at Phase 3 via canonical-serialization-contract re-reading. The phase-3-detection pattern parallels the T_B-derived structural-preclusion from [`layer-b-parameter-calibration-evidence.md`](./layer-b-parameter-calibration-evidence.md) Phase 3 sub-decision 1; both cases share the structure "framing enumerated structurally-precluded candidates that Phase 3's contract-re-reading surfaces." **Recommendation for future operational-spec RFCs: explicit canonical-serialization-contract re-reading as a Phase 0 step (between framing PR and Phase 3) catches structural-preclusion sooner.**

### Finding 9 — Cross-decision interaction matrix: conservative-defaults bundle has no internal conflict

Per Phase 2's cross-decision interaction matrix + Phase 3's per-sub-decision findings: the conservative-defaults bundle A1 + B1 + C1 + D1 + E1 + F3 has no internal conflict. Each sub-decision's recommendation is compatible with every other sub-decision's recommendation in the bundle. The aggressive bundle (A2 + B2 + C-NA + D2 + E2 + F1) introduces multiple downstream dependencies (service architecture RFC, projection-rebuild RFC, schemas-evolution RFC, override-semantics specification, CLI breaking change) — adoption would multiply the RFC arc rather than complete it.

## Phase 5 — Recommendation

The discussion phase recommends the following service-tier implementation specification:

| Sub-decision | Recommendation | Reversal trigger |
|---|---|---|
| **A. Evaluation locus** | **A1 — internal package** (`services/ingestion/internal/hypothesis/layerb/`) | Revise to A2 (separate service) if operational evidence shows Layer B evaluation requires independent scaling/deployment from ingestion (e.g., evaluation latency dominates ingestion CLI runtime; resource consumption competes with substrate writes). |
| **B. Computation strategy** | **B1 — on-the-fly from substrate** | Revise to B2 (Cat II projection) if observed evaluation latency proves prohibitive at scale (e.g., closure-read time exceeds operator-tolerable threshold). Revise to B3 (substrate audit record) if audit-trail requirements become non-optional (regulatory, operational forensics, or post-incident analysis demands persistent evaluation history). |
| **C. Window stream identity** | **C1 — substrate-global** | No reversal. Structurally determined by the canonical-serialization-contract [`§Demotion-Candidacy Predicate`](../../architecture/canonical-serialization-contract.md); only contract revision can revisit. |
| **D. Output shape** | **D1 — transient DemoteReport extension** | Revise to D2 (`LayerBEvaluation` Cat II record) if audit-trail requirements surface; revise to D3 (`DemotionCandidacyEvaluation` composite) if operator workflow demands Layer A + Layer B combined audit per evaluation. Both reversals trigger a follow-on schemas-evolution RFC. |
| **E. demote-hypothesis interaction** | **E1 — advisory like Layer A** | Revise to E2 (enforcing refusal) ONLY if operator-elected demotion patterns surface that warrant barrier-mode (e.g., empirical evidence of demotion-without-criterion creating substrate noise); E2 adoption requires concurrent structural commitment beyond [`§0011`](../../charter/decision-log.md). Revise to E3 (untouched, separate candidate-finder) if operator-workflow evidence shows the advisory field is overlooked in CLI output. |
| **F. N_A=1 day Layer A cadence source** | **F3 — CLI operator-supplied with bundled defaults** | Revise to F1 (bundled-only, no CLI option) if operator-experience evidence accumulates that the existing `-cadence-seconds` option is harmful (e.g., overrides defeat the [`§0138`](../../charter/decision-log.md) inception-phase calibration). |

The bundle **A1 + B1 + C1 + D1 + E1 + F3** is the conservative-defaults recommendation across the six sub-decisions. It honors:

- [`§0011`](../../charter/decision-log.md) staged-combination Layer A pattern precedent (advisory in DemoteReport, no barrier);
- [`§0022`](../../charter/decision-log.md) + [`§0023`](../../charter/decision-log.md) inception-phase posture (no service-architecture changes ahead of need; no projection storage ahead of need; no schemas-evolution events ahead of need);
- [`§0136`](../../charter/decision-log.md) canonical-serialization-contract structural determination (sub-decision C resolves by contract reading);
- [`§0138`](../../charter/decision-log.md) N_A bundling at the substrate layer (F3 reads default from LayerBParameters);
- [CLAUDE.md §7](../../../.claude/CLAUDE.md) constitutional minimalism (no new patterns, no new service boundaries, no new substrate-record types at this RFC's resolution).

**Per-sub-decision reversal triggers are observable empirical signals, not predictions.** Per [`layer-b-parameter-calibration-evidence.md`](./layer-b-parameter-calibration-evidence.md) Phase 4 Finding 8 carry-forward: triggers are observation-based ("if we observe X, revise"); the structural commitment is what would be observed, not what is predicted.

### Implementation surface at resolution

If accepted, the resolution PR will land:

- New internal package `services/ingestion/internal/hypothesis/layerb/` with pure-function `Evaluate(ctx, sub, promotionEventHash, params) (Verdict, error)` (sub-decision A1).
- The function reads substrate on-the-fly (sub-decision B1) using `closure_hashes` per [`§0136`](../../charter/decision-log.md) + `evidential_independence` per [`§0140`](../../charter/decision-log.md); window per the contract's substrate-global reading (sub-decision C1); returns a `Verdict` struct in memory (sub-decision D1).
- Existing demote-hypothesis + 4 subtype variants are extended to invoke `layerb.Evaluate` and surface the verdict in `DemoteReport`'s new `LayerB*` fields (sub-decision E1). Demote behavior unchanged — records the demotion regardless of the verdict.
- promote-hypothesis is extended to default `-cadence-seconds` from the hypothesis's `LayerBParameters.n_a_duration_nanoseconds` field when the option is not supplied (sub-decision F3); existing CLI semantic preserved.
- `services/ingestion/internal/hypothesis/demotion.go:97-99` comment refreshed to acknowledge [`§0129`](../../charter/decision-log.md) §2.6 freeze + [`§0138`](../../charter/decision-log.md) Layer B specification + the new evaluation function.
- Unit tests over fixture substrate for the evaluation function; integration tests for CLI extension.
- No new proto types. No corpus changes. No schemas-evolution event.

### Forward schedule

- **Resolution PR**: RFC status moves from `discussion` to `accepted`; decision-log entry recorded; service-tier implementation work proceeds under ordinary RFC discipline.
- **Implementation PR(s)**: post-resolution; one or more PRs land the implementation surface specified above. Coordination with existing CI hooks (Go build/test, doc-check) is structural — no new infrastructure required.
- **Downstream consequences** (post-resolution): no Charter prose modification; no Ontology binding-text change; the `demotion.go:97-99` comment refresh is bundled with the implementation PR(s).
