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

The service tier — currently the `services/ingestion/` Go module — does not evaluate this predicate. It evaluates only Layer A (cadence gate) and reports it as an advisory flag (`CadenceSatisfied`) in `DemoteReport`, recording the demotion regardless of the flag. The CLI surface is operator-elected: an operator invokes `demote-hypothesis` (or one of four subtype-specific variants) with a target promotion hash + reason + actor; the service appends the demotion event to substrate. Layer B does not enter the call chain.

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

**Conservative-defaults reading**: A1 is the simplest form respecting DRY. A2 introduces a service boundary ahead of need (cost: deployment topology, RPC schema, cross-service auth). A3 violates DRY across 5 implementation sites.

### Sub-decision B — Computation strategy

> How is the predicate's two components (`freshness_B`, `saturation_C`) computed from substrate at evaluation time?

| Candidate | Strategy | Sketch |
|---|---|---|
| B1 | On-the-fly from substrate | Per-evaluation: read promotion event + closure_hashes; scan substrate window (N=1000 events) under sub-decision C's identity; compute fresh-roots-count and saturation-ratio; return verdict. No persistent state. |
| B2 | Cat II projection per promoted hypothesis | Maintain a continuously-updated projection record per promoted hypothesis (freshness_B(H), saturation_C(H), last-updated-at, last-input-event-hash). Read projection at evaluation time; rebuild from substrate on demand per the projection-rebuildability discipline (`projection-model.md`). |
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

**Schemas-evolution-event scope**: D1 is contained within the existing schema set; no schemas-evolution event. D2 and D3 are schemas-evolution events per the canonical-serialization-contract §Schemas-Evolution Events boundary; if adopted, they may need their own follow-on schema-RFC at resolution time.

**Conservative-defaults reading**: D1 matches the existing Layer A pattern (advisory in DemoteReport, not committed as a record). D2 and D3 add audit-trail durability but cost substrate-write per evaluation. The cost-justification depends on operational scale (number of evaluations per day × storage cost per record); at inception phase, this cost is unknown.

### Sub-decision E — demote-hypothesis interaction

> How does the existing demote-hypothesis CLI (and 4 subtype variants) interact with the Layer B verdict?

| Candidate | Interaction | Sketch |
|---|---|---|
| E1 | Advisory, like Layer A | Demote CLI evaluates Layer B (via the chosen locus per A); adds `LayerBFired`, `LayerBFreshnessB`, `LayerBSaturationC` flags to DemoteReport; demote proceeds and commits regardless of the verdict. Operator sees the state post-facto. |
| E2 | Enforcing refusal with override | Demote CLI evaluates Layer B; if `Layer B(H)` is false (predicate did not fire), CLI refuses to commit the demotion unless an explicit `--force-layer-b-bypass` flag is supplied. The override is recorded in the demotion reason for audit. |
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

**Pre-existing CLI surface**: promote-hypothesis already accepts `-cadence-seconds`. F1 changes the contract (CLI no longer needs `-cadence-seconds`); F3 preserves the existing flag with a default-from-substrate fallback. F2 is the most disruptive.

---

## Cross-decision interaction matrix

The six sub-decisions interact in known ways:

- **A1 + B1 + D1 + E1 + F3**: Conservative-defaults bundle. Internal package; on-the-fly; transient; advisory; existing CLI preserved. Simplest implementation form respecting form-vs-parameter-vs-implementation discipline. No schemas-evolution event. No new substrate record types. No service-architecture changes.
- **A2 + B2 + D2 + E2 + F1**: Aggressive bundle. Separate service; projection storage; substrate audit record; enforcing refusal; bundled-only config source. Largest blast radius; schemas-evolution event(s) required; service-architecture change; backward-compatibility risk.
- **A1 + B3 + D2**: Mixed. Internal package + on-the-fly + audit record. Schemas-evolution event but no service-architecture change.
- **E3 + (any A/B/D/F)**: Decouples Layer B evaluation from demote CLI; useful if demote CLI behavior is contentious.

The Phase 3 analysis will produce a per-sub-decision recommendation; the Phase 4 synthesis will check the recommendations against the interaction matrix for consistency.

---

## Forward schedule

- **Phase 3** (next commit): Apply epistemic-skill methodology to each sub-decision in turn — operationalization clarity, falsifiability of recommendation, counterfactual robustness, vocabulary-respect. Single PR.
- **Phase 4** (subsequent commit): Synthesis findings + cross-decision interaction check. Identify single dominant configuration across the six sub-decisions OR identify the genuine trade-off space.
- **Phase 5** (subsequent commit): Single-configuration recommendation OR multi-configuration finalist comparison. Phase 5 produces the substantive deliberation outcome.
- **Resolution PR**: RFC status moves from `discussion` to `accepted`; decision-log entry recorded; implementation work proceeds under ordinary RFC discipline.
- **Downstream consequences** (post-resolution): `services/ingestion/internal/hypothesis/demotion.go:97-99` comment refresh per the chosen implementation; new code per the chosen sub-decision configuration; CLI surface changes per E variant.
