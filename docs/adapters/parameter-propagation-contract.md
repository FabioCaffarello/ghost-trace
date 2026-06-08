# Parameter-Propagation Contract — Sub-Benchmark 1 Orchestrator

> Per [`decision-log.md` §0220](../charter/decision-log.md). Established post-§0219
> stale-scope-boundary diagnostic + §0217 MO2 binding (a)+(b) discipline
> (full evaluation-path source inspection + source-grounded markers).

This document carries the parameter-propagation contract for CLIs invoked in
`infra/docker/run-sub-benchmark-1.sh`. Each row documents one CLI opt-in option
whose state at orchestrator-invocation affects downstream evaluation in the
pipeline. The contract serves three purposes:

1. **Audit trail** — operator reading the orchestrator can cross-reference
   each invocation against the contract to verify wiring matches downstream
   expectations.
2. **Pre-flight checklist** — when a new CLI is added to the orchestrator or
   an existing CLI's downstream surface changes, the contract is consulted to
   verify wiring obligations.
3. **Scope-boundary registry** — for each non-wired opt-in option, the
   contract distinguishes intentional-still-valid (current scope deliberately
   defers; downstream does not depend on the option) vs stale-from-prior-scope
   (prior scope deferred; subsequent scope expanded the dependency; comment
   not propagated). Per [`decision-log.md` §0219](../charter/decision-log.md)
   Finding 1 stale-scope-boundary subclass.

## Lazy-population discipline (§0220 MO)

The table is **lazy-populated**: rows are added when a CLI opt-in option is
empirically surfaced as relevant to downstream evaluation. This entry does NOT
catalog every option of every CLI invoked in the orchestrator preventively
(rejected per §0143 D2 observation-precedes-refactor discipline + §0220 MO
lazy-population-over-exhaustive-audit).

**Maintenance:** add a new row when (a) a §-entry surfaces an opt-in option
whose wiring matters, or (b) an audit during scope-expansion identifies a
stale-scope-boundary. Do NOT add rows speculatively.

## Stale-scope-boundary maintenance discipline (§0220 inline note)

When an orchestrator step is added or modified, all `Per §NNNN:` scope
comments within `run-sub-benchmark-1.sh` MUST be re-validated against the
current step set. A scope comment that was correct at its original entry can
become stale when a subsequent entry expands the step set in a way that
depends on what the comment defers. The §0213 → §0216 → §0219 → §0220 cycle
is the canonical empirical example (§0213 scoped Layer B parameters out at
promotion; §0216 expanded scope to add demote-formations dependent on those
parameters; §0219 Finding 1 surfaced the stale comment; §0220 revised it +
wired `-layer-b`).

## Contract table

Columns:
- **CLI** — the cmd/<X>/ directory; the operator-invoked binary
- **Option** — exact CLI flag string (e.g., `-layer-b`)
- **Default** — what happens when not passed at invocation
- **Downstream effect** — what field/evaluation the option controls
- **Read by** — downstream CLI/component that consumes the field
- **Guard** — file:line of early-return condition that gates downstream
  evaluation (per §0217 MO2 binding (a)+(b) full evaluation-path discipline)
- **Wired in orchestrator?** — current state YES / NO / N/A (option exists
  but no downstream dependency)
- **Scope decision origin** — §-entry that established the scope decision +
  current validity classification
- **Status** — INTENTIONAL / STALE-FROM-PRIOR-SCOPE / REMEDIATED

| CLI | Option | Default | Downstream effect | Read by | Guard | Wired in orchestrator? | Scope decision origin | Status |
|---|---|---|---|---|---|---|---|---|
| `promote-automation-group` | `-layer-b` | false | Populates promotion event's `layer_b_parameters` field with §0138 inception-phase resolved values (T_B=K_C=1/2; N_window=1000; N_A=cadence_seconds) | `demote-automation-group` → `hypothesis.DemoteAutomationGroup` → `layerb.Evaluate` | `layerb.go:132` returns error if `opts.Params == nil` | **YES** (post-§0220) | §0213 deliberately scoped out under candidacy-materialization split; §0216 expanded scope to add demote-formations Step 8 (depends on parameters at promotion-time); §0219 Finding 1 surfaced stale-scope-boundary; §0220 wires + revises comment | **REMEDIATED** (post-§0220) |
| `find-coordination-ring-candidates` | `-with-attribution` | (option exists per §0214 Finding 4 ratification of hypothesis (a)) | Cat II AttributionView consumption at signature evaluation; without it, `observations_skipped_no_actor = total` | `find-coordination-ring-candidates` signature evaluation logic | (audit deferred until §0222+ remediation) | **NO** (per §0214 Finding 4) | §0212 named hypothesis (a)/(b); §0214 ratified (a) but deferred remediation to §0215 (renumbered §0222+ through §0214→§0217→§0219 reflows); §0222+ binding scope pending | **STALE-FROM-PRIOR-SCOPE** (pending §0222+ remediation) |
| `ingest-cic-ids` | `-strict` | false | Exits 3 if RowsRejected > 0 (CLI-internal exit-code semantic; no downstream pipeline field affected) | (CLI-internal only) | none (CLI-internal exit-code path) | N/A (no downstream dependency) | §0204 design; default-false correct for orchestrator (rejections logged in Report; non-strict exit preserves pipeline) | INTENTIONAL |
| `derive-actor-attribution` | `-definition-version` | `network-5tuple-actor-v1` | Selects attribution definition for derivation | (CLI-internal; chooses derivation impl) | `resolveDefinition` switch in main.go | YES (default matches §0168) | §0209 design | INTENTIONAL |
| `find-automation-group-candidates-network` | `-with-attribution` | (option exists) | Cat II AttributionView consumption for tcp_flow_features_clustering_v1 signature; populates derived_actor_ref on observations missing declared actor_ref | tcp_flow_features_clustering_v1 evaluation logic | (audit grounds at §0169 integration test) | YES (orchestrator passes per Step 4) | §0209 + §0169 wire | INTENTIONAL |
| `find-automation-group-candidates-network` | `-signature` | (default TBD) | Selects signature variant (flow-features vs alternative) | network signature CLI dispatch | (audit grounds at §0169) | YES (orchestrator passes `flow-features` per Step 4) | §0169 design | INTENTIONAL |

## Future additions

Empirical pressure adds rows. Examples of triggers:

- A §-entry surfaces a new opt-in option whose wiring matters → add row with
  full evaluation-path citations per §0217 MO2 binding discipline.
- A multi-run methodology cycle per §0228+ exercises CLIs not in current table
  → add rows as evidence surfaces.
- A Frente 2 (synthetic) or Frente 3 (honeypot) bridge per §0227+ introduces
  new opt-in options → add rows when the bridge lands.
- An audit during scope-expansion (per the stale-scope-boundary maintenance
  discipline above) identifies a new instance → add row with STALE-FROM-
  PRIOR-SCOPE classification + remediation §-entry citation.

The contract is a forward-only record; rows are not deleted retroactively
even when REMEDIATED status is reached (the row preserves the history of the
gap + its closure for audit-trail completeness).
