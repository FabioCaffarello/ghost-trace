# Ingestion Service

## Constitutional Role

Receives observations from producers and commits them to the primary event log. The point at which the system takes responsibility for a record. After commitment, the record is governed by [Charter §2.1 Observational Integrity](../../docs/charter/constitutional-charter.md#21-observational-integrity).

## Status

Skeleton. First commit producing executable code per [`decision-log §0027`](../../docs/charter/decision-log.md) Consequences ("Service skeleton work ... proceeds under ordinary RFC/PR discipline from this point forward"). Recorded at [`decision-log §0030`](../../docs/charter/decision-log.md).

## Architecture

**Provenance**: every primary observation commits paired with an `IngestionEvent` enrichment per [`§0038`](../../docs/charter/decision-log.md). The pair captures **what** the producer reported (one of the registered Cat I primary-observation types — see §[Category I message types](#category-i-message-types) below) AND **how** the service received it (channel — `stdin`/`http`/`https`/`https+mtls` — plus, when delivered over mTLS, the verified peer-certificate's Common Name + SANs + SHA-256 fingerprint). The two events commit atomically via `substrate.AppendPair` (single SQL transaction). The pairing is by reference: each `IngestionEvent.primary_event_hash` carries the content-hash of its paired observation.

Four packages under `internal/`, each consuming a published contract:

- [`internal/canonical`](./internal/canonical) — implements [`docs/architecture/canonical-serialization-contract.md`](../../docs/architecture/canonical-serialization-contract.md). Single `Marshal` + `Hash` + `HashHex` entry points; service code MUST NOT call `proto.Marshal` directly.
- [`internal/substrate`](./internal/substrate) — implements the [`§0027`](../../docs/charter/decision-log.md) SQLite + content-addressed blob-store substrate. Single `Append` entry point + `writeMu` mutex per [`docs/architecture/concurrency-pattern.md`](../../docs/architecture/concurrency-pattern.md) §Substrate-Writer Serialization. PRAGMA `journal_mode=WAL` + `synchronous=FULL` per [`§0027`](../../docs/charter/decision-log.md) Proposal item 1. Hash-verification on every blob-read path per [`§0027`](../../docs/charter/decision-log.md) AP5.
- [`internal/ingest`](./internal/ingest) — composes `canonical` + `substrate` into a typed `Append(ctx, msg, eventTime)` boundary. The single per-process write entry point.
- [`internal/httpapi`](./internal/httpapi) — minimum-viable HTTP interface (`POST /v1/events/{type}/{type}` accepting `application/x-protobuf`; `GET /healthz`). The `{type}` segment selects a registered Cat I message type via `ingest.LookupURLPath` (per [`§0042`](../../docs/charter/decision-log.md)). Same error classification as the stdin worker: recoverable → 4xx + JSON body; unrecoverable (substrate §2.1 violations) → 500 + JSON body + signal the service-level fatal channel per [`§0032`](../../docs/charter/decision-log.md).

`main.go` orchestrates up to four goroutines via `errgroup` per [`concurrency-pattern.md`](../../docs/architecture/concurrency-pattern.md): the stdin worker (always), the HTTP server (when `--http` is set), an HTTP-graceful-shutdown coordinator, and a fatal-coordinator that propagates HTTP-side unrecoverable errors back through errgroup. Shutdown coordinated via `signal.NotifyContext`.

## `verify` CLI

Companion binary at [`cmd/verify`](./cmd/verify) that performs an up-front substrate-integrity check. Discharges the `§0033` `verify` follow-on; see [`§0039`](../../docs/charter/decision-log.md).

```sh
make verify-build                                                       # builds ./bin/verify
./bin/verify -db ./ghost-trace.db -blobs ./blobs                        # walks events table; verifies every blob
./bin/verify -db ./ghost-trace.db -blobs ./blobs -check-orphans         # also reports orphan blobs (informational)
```

Walks every events-table row in commit order, recomputes each blob's BLAKE3 hash via `substrate.ReadBlob`, surfaces hash-mismatch + missing-blob failures. With `-check-orphans` (per [`§0040`](../../docs/charter/decision-log.md)), also walks the blob-store directory + reports blobs whose content-hash does not appear in the events table — orphans are **harmless** at the substrate layer per [`§0033`](../../docs/charter/decision-log.md) (the events table is authoritative); they are reported but do NOT cause non-zero exit. Writes structured JSON to stdout + a brief human summary to stderr. Exit code: **0** on pass (including substrates with orphan blobs); **1** on any §2.1 violation (hash-mismatch or missing-blob); **2** on tool/configuration error. Intended for post-restore verification (per [`§0033` §Restoration Procedure step 3](../../docs/architecture/operational-ops.md)) and periodic substrate-integrity audits.

## `derive-operational-session` CLI

Operator-invoked tool to derive Category II [`OperationalSession`](../../schemas/events/v1/operational_session.proto) constructs from every `DeclaredSession` in the substrate under a versioned operational definition (per [`§0043`](../../docs/charter/decision-log.md) — first Cat II construct landing).

```sh
make derive-build                                                       # builds ./bin/derive-operational-session

# Run the default padded-v1 definition (pad_seconds=300)
./bin/derive-operational-session -db ./ghost-trace.db -blobs ./blobs

# Override the padding parameter — produces a NEW set of OperationalSession
# records alongside the prior derivations (versioning per entity-model.md line 45)
./bin/derive-operational-session -db ./ghost-trace.db -blobs ./blobs -pad-seconds 600

# Run the inactivity-window-v1 definition (consumes NetworkEvents per actor)
./bin/derive-operational-session -definition-version inactivity-window-v1 -inactivity-seconds 1800
```

Walks every `DeclaredSession` row in the substrate via `substrate.WalkEvents`, applies the operational definition deterministically (per [Charter §2.2](../../docs/charter/constitutional-charter.md#22-epistemic-separation) Category II requirement), and commits each `OperationalSession` to the same events table via `substrate.Append` (acquires `writeMu` per [`concurrency-pattern.md`](../../docs/architecture/concurrency-pattern.md) §Substrate-Writer Serialization). Re-running with an identical `(definition_version, definition_parameters)` tuple is a no-op (content-hash collision → `INSERT OR IGNORE`); re-running with a NEW tuple produces NEW records and preserves the prior ones per [`entity-model.md` §Category II](../../docs/ontology/entity-model.md) line 45.

Writes structured JSON to stdout (`definition_version`, `definition_parameters`, `examined`, `newly_derived`, `already_derived`) + a brief human summary to stderr. Exit code: **0** on success (including zero-newly-derived); **2** on tool/configuration error.

Registered operational definitions:

| Version | Parameters | Boundary derivation |
|---|---|---|
| `padded-v1` | `pad_seconds=<int>` (default 300) | `operational_start_at = declared_at`; `operational_end_at = declared_at + pad_seconds`. Minimal canonical example; exercises all Cat II structural mechanisms (deterministic derivation, identity-via-version, provenance, boundary divergence). |
| `inactivity-window-v1` | `inactivity_seconds=<int>` (default 1800) | `operational_start_at = declared_at`; the boundary extends past consecutive `NetworkEvent` rows for the same `actor_ref` (each within `inactivity_seconds` of the prior event) until a gap exceeds the window. `operational_end_at = lastObserved + inactivity_seconds`. Multi-Cat-I-input definition (per [`§0044`](../../docs/charter/decision-log.md)); composes the [`§0042`](../../docs/charter/decision-log.md) `NetworkEvent` Cat I type with the [`§0043`](../../docs/charter/decision-log.md) derivation pathway. Mirrors the canonical example at [`entity-model.md` line 39](../../docs/ontology/entity-model.md). |

Adding a new operational definition registers via [`internal/derivation`](./internal/derivation): implement `OperationalDefinition` (Version, Parameters, Derive) and wire it into `cmd/derive-operational-session/main.go`'s `resolveDefinition`. Definitions that need additional Cat I observations beyond the source `DeclaredSession` consult the `DerivationContext` passed to `Derive` (per [`§0044`](../../docs/charter/decision-log.md)).

## `form-hypothesis` CLI

Operator-invoked tool to form Category III hypotheses from the substrate's primary observations (per [`§0045`](../../docs/charter/decision-log.md) — first Cat III subtype landing). Lands the first Cat III lifecycle event (`BehavioralClusterFormation`) per [Charter §2.5](../../docs/charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) (frozen v0.3).

```sh
make form-hypothesis-build                                                # builds ./bin/form-hypothesis

# Run the default session-descriptor-shared-v1 pattern with min-cluster-size 2
./bin/form-hypothesis -db ./ghost-trace.db -blobs ./blobs

# Override the minimum cluster size — produces NEW formation events
# alongside the prior derivations (versioning per entity-model.md line 45)
./bin/form-hypothesis -db ./ghost-trace.db -blobs ./blobs -min-cluster-size 3
```

The CLI walks every `DeclaredSession` in the substrate, applies the formation pattern, and commits each resulting `BehavioralClusterFormation` event via `substrate.Append` (acquires `writeMu` per [`concurrency-pattern.md`](../../docs/architecture/concurrency-pattern.md) §Substrate-Writer Serialization). Re-running with an identical `(pattern_signature, pattern_parameters)` tuple is a no-op (content-hash collision → `INSERT OR IGNORE`); re-running with a NEW tuple produces NEW lifecycle events and preserves the prior ones per [Charter §2.5](../../docs/charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness).

Writes structured JSON to stdout (`pattern_signature`, `pattern_parameters`, `examined`, `newly_formed`, `already_formed`) + a brief human summary to stderr. Exit code: **0** on success (including zero-newly-formed); **2** on tool/configuration error.

Registered formation patterns:

| Signature | Parameters | Inference |
|---|---|---|
| `session-descriptor-shared-v1` | `min_cluster_size=<int>` (default 2) | Groups `DeclaredSession` rows by byte-equal `session_descriptor`; forms a `BehavioralCluster` per group with >= `min_cluster_size` distinct `actor_ref`s. Minimum-viable canonical example exercising every Cat III structural mechanism (deterministic recording, lifecycle-event-as-Cat-I per §2.5 BC5, observational provenance per §2.3, identity-via-content-hash). |

Adding a new formation pattern registers via [`internal/hypothesis`](./internal/hypothesis): implement `FormationPattern` (Signature, Parameters, Form) and wire it into `cmd/form-hypothesis/main.go`'s `resolvePattern`. Patterns that need additional Cat I observations beyond `DeclaredSession` extend the `FormationContext` interface with new typed accessors (the same incremental-extension procedure that [`§0044`](../../docs/charter/decision-log.md) established for Cat II `DerivationContext`).

**Other lifecycle operations** per [Charter §2.5](../../docs/charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness): promotion landed at [`§0046`](../../docs/charter/decision-log.md) via [`cmd/promote-hypothesis`](#promote-hypothesis-cli); demotion at [`§0047`](../../docs/charter/decision-log.md) via [`cmd/demote-hypothesis`](#demote-hypothesis-cli); dissolution at [`§0048`](../../docs/charter/decision-log.md) via [`cmd/dissolve-hypothesis`](#dissolve-hypothesis-cli); merge at [`§0049`](../../docs/charter/decision-log.md) via [`cmd/merge-hypotheses`](#merge-hypotheses-cli); split at [`§0050`](../../docs/charter/decision-log.md) via [`cmd/split-hypothesis`](#split-hypothesis-cli). **§2.5 lifecycle surface complete for BehavioralCluster** — all 6 of 6 operations structurally observable. The SECOND Cat III concrete subtype (`AutomationGroup`) lifecycle arc landed across [`§0056`](../../docs/charter/decision-log.md)–[`§0061`](../../docs/charter/decision-log.md) via `form-automation-group` / `promote-automation-group` / `demote-automation-group` / `dissolve-automation-group` / `merge-automation-groups` / `split-automation-group` — **§2.5 lifecycle surface complete for AutomationGroup too**. Per Charter §2.5 BC3, the substrate stores ONLY lifecycle events — the hypothesis's current state is a projection over the operation event chain (per [`§0051`](../../docs/charter/decision-log.md)+).

## `form-automation-group` CLI

Operator-invoked tool to form `AutomationGroup` hypotheses — the **second Category III concrete subtype** per [`§0056`](../../docs/charter/decision-log.md). Mirrors [`form-hypothesis`](#form-hypothesis-cli) for the new subtype's formation step. Distinct subtype identity per [entity-model.md §Category III](../../docs/ontology/entity-model.md): an AutomationGroup is "a set of actors whose behavioral patterns match a signature of automated (non-human) operation" — the inference is about operation CHARACTER (automated), NOT shared operatorship (which is the BehavioralCluster inference).

```sh
make form-automation-group-build                                          # builds ./bin/form-automation-group

# Default uniform-cadence-v1 pattern (5 min observations, CoV ≤ 0.15)
./bin/form-automation-group -db ./ghost-trace.db -blobs ./blobs

# Stricter threshold + larger observation requirement
./bin/form-automation-group -min-observation-count 10 -max-cov-threshold 0.05
```

The CLI walks every `DeclaredSession`, applies the formation pattern, and commits each resulting `AutomationGroupFormation` event via `substrate.Append`. Same idempotency + versioning semantics as `form-hypothesis` per §0045.

Registered formation patterns:

| Signature | Parameters | Inference |
|---|---|---|
| `uniform-cadence-v1` | `min_observation_count=<int>` (default 5), `max_cov_threshold=<float>` (default 0.15) | Groups `DeclaredSession` rows by `actor_ref`; emits one `AutomationGroupFormation` per actor whose inter-event-delta coefficient-of-variation falls within the threshold (signature of mechanical / non-human cadence). Single-actor groups at this layer; multi-actor signature-matching is a follow-on. |

Adding a new AutomationGroup formation pattern: implement `AutomationGroupFormationPattern` in [`internal/hypothesis`](./internal/hypothesis), register it in `cmd/form-automation-group/main.go`'s `resolvePattern`. Same incremental-extension pathway as §0045 + §0044.

Exit code: **0** on success (including zero-newly-formed); **2** on tool/configuration error.

## `form-campaign-hypothesis` CLI

Operator-invoked tool to form `CampaignHypothesis` hypotheses — the **third Category III concrete subtype** per [`§0063`](../../docs/charter/decision-log.md). Mirrors [`form-hypothesis`](#form-hypothesis-cli) and [`form-automation-group`](#form-automation-group-cli) for the new subtype's formation step. Distinct subtype identity per [entity-model.md §Category III](../../docs/ontology/entity-model.md): a CampaignHypothesis is "a set of EVENTS whose patterns suggest membership in a unified operation" — **event-centric** (NOT actor-centric like BC/AG).

```sh
make form-campaign-hypothesis-build                                        # builds ./bin/form-campaign-hypothesis

# Default temporal-descriptor-cohort-v1 pattern (min 3 events, max 300s gap)
./bin/form-campaign-hypothesis -db ./ghost-trace.db -blobs ./blobs

# Stricter cohort criteria
./bin/form-campaign-hypothesis -min-campaign-size 5 -max-intra-event-gap-seconds 60
```

The CLI walks every `DeclaredSession`, applies the formation pattern, and commits each resulting `CampaignHypothesisFormation` event via `substrate.Append`. Same idempotency + versioning semantics as `form-hypothesis` per §0045.

Registered formation patterns:

| Signature | Parameters | Inference |
|---|---|---|
| `temporal-descriptor-cohort-v1` | `min_campaign_size=<int>` (default 3), `max_intra_event_gap_seconds=<int>` (default 300) | Groups `DeclaredSession` rows by byte-equal `session_descriptor`; within each group, scans chronologically and emits one `CampaignHypothesisFormation` per cohort of ≥ `min_campaign_size` events where consecutive events are within `max_intra_event_gap_seconds` of each other. |

Adding a new CampaignHypothesis formation pattern: implement `CampaignHypothesisFormationPattern` in [`internal/hypothesis`](./internal/hypothesis), register it in `cmd/form-campaign-hypothesis/main.go`'s `resolvePattern`. Same incremental-extension pathway as §0045 + §0056.

Exit code: **0** on success; **2** on tool/configuration error.

## `form-coordination-ring` CLI

Operator-invoked tool to form `CoordinationRing` hypotheses — the **fourth (and final) Category III concrete subtype** per [`§0070`](../../docs/charter/decision-log.md). Mirrors the formation pathways for the prior three subtypes but with an **interaction-centric (edge-list) inference** rather than the actor-set (BC/AG) or event-set (CH) shapes. Per [entity-model.md §Category III](../../docs/ontology/entity-model.md), a CoordinationRing is "a set of actors whose patterns of INTERACTION suggest coordinated action" — the relational structure is what distinguishes it from the other three subtypes; flattening edges into a vertex set would lose the property the subtype was carved out to preserve (per [§0070 modeling choice](../../docs/charter/decision-log.md)).

```sh
make form-coordination-ring-build                                          # builds ./bin/form-coordination-ring

# Default co-occurrence-window-v1 pattern (min 3 supports, max 600s window)
./bin/form-coordination-ring -db ./ghost-trace.db -blobs ./blobs

# Tighter window + stricter support
./bin/form-coordination-ring -min-edge-support 5 -max-window-seconds 120
```

The CLI walks every `DeclaredSession`, applies the formation pattern, and commits each resulting `CoordinationRingFormation` event via `substrate.Append`. Same idempotency + versioning semantics as the prior three subtypes per §0045.

Registered formation patterns:

| Signature | Parameters | Inference |
|---|---|---|
| `co-occurrence-window-v1` | `min_edge_support=<int>` (default 3), `max_window_seconds=<int>` (default 600) | Groups `DeclaredSession` rows by byte-equal `session_descriptor`; within each group, builds undirected actor-pair edges from sessions whose `declared_at` timestamps fall within `max_window_seconds` of each other; emits one `CoordinationRingFormation` per connected component whose constituent edges all meet `min_edge_support`. Per §0070 the wire-form `CoordinationRingInteraction` is lex-canonicalized within each edge (`actor_a < actor_b`) and the repeated field is sorted ascending — content-hash stability under observation reordering. |

Adding a new CoordinationRing formation pattern: implement `CoordinationRingFormationPattern` in [`internal/hypothesis`](./internal/hypothesis), register it in `cmd/form-coordination-ring/main.go`'s `resolvePattern`. Same incremental-extension pathway as §0045 + §0056 + §0063.

Exit code: **0** on success; **2** on tool/configuration error.

## `promote-coordination-ring` CLI

Operator-invoked tool to record the **CoordinationRing promotion** lifecycle operation per [`§0071`](../../docs/charter/decision-log.md) — second lifecycle operation of the fourth Cat III subtype arc. Mirrors `promote-hypothesis` (BC), `promote-automation-group` (AG), and `promote-campaign-hypothesis` (CH).

```sh
make promote-coordination-ring-build                                       # builds ./bin/promote-coordination-ring

./bin/promote-coordination-ring \
  -formation-event-hash <64-hex-chars> \
  -cadence-seconds 86400 \
  -reason "operational pilot — coordination ring enrichment"
```

Validates the supplied `formation-event-hash` resolves to a `CoordinationRingFormation` row (otherwise exits 3 — preserves §2.5-lifecycle-integrity). Cross-subtype rejection: a BC/AG/CH formation hash returns `ErrTargetWrongType` (exit 3) — the subtype-specific message_type discriminator prevents misclassification.

| Option | Default | Notes |
|---|---|---|
| `-formation-event-hash` | (required) | Hex-encoded BLAKE3-256 of the target `CoordinationRingFormation`. |
| `-cadence-seconds` | 86400 (24h) | Layer A parameter per [`§0011`](../../docs/charter/decision-log.md). |
| `-promoted-at-ns` | 0 (= wall-clock now) | Explicit `promoted_at` for forensic replay. |
| `-reason` | empty | Operator-supplied forensic note. |

Exit codes: **0** success; **2** tool/configuration error; **3** target-not-found or target-wrong-type.

## `demote-coordination-ring` CLI

Operator-invoked tool to record the **CoordinationRing demotion** lifecycle operation per [`§0072`](../../docs/charter/decision-log.md) — third lifecycle operation of the fourth Cat III subtype arc. Mirrors `demote-hypothesis` (BC), `demote-automation-group` (AG), `demote-campaign-hypothesis` (CH).

```sh
make demote-coordination-ring-build                                        # builds ./bin/demote-coordination-ring

./bin/demote-coordination-ring \
  -promotion-event-hash <64-hex-chars> \
  -reason "operational cycle close"
```

Validates the supplied `promotion-event-hash` resolves to a `CoordinationRingPromotion` row (otherwise exits 3). Per §0011 Layer A is a CANDIDACY gate, NOT a hard barrier — the substrate accepts the demotion regardless of whether the cadence has elapsed; the structured output surfaces `cadence_satisfied` + `cadence_elapsed_seconds`.

| Option | Default | Notes |
|---|---|---|
| `-promotion-event-hash` | (required) | Hex-encoded BLAKE3-256 of the target `CoordinationRingPromotion`. |
| `-demoted-at-ns` | 0 (= wall-clock now) | Explicit `demoted_at` for forensic replay. |
| `-reason` | empty | Operator-supplied forensic note; **strongly recommended** when demoting within the cadence window. |

Exit codes: **0** success; **2** tool/configuration error; **3** target-not-found or target-wrong-type.

## `dissolve-coordination-ring` CLI

Operator-invoked tool to record the **CoordinationRing dissolution** lifecycle operation per [`§0073`](../../docs/charter/decision-log.md) — fourth lifecycle operation of the fourth Cat III subtype arc. Mirrors `dissolve-hypothesis` (BC), `dissolve-automation-group` (AG), `dissolve-campaign-hypothesis` (CH). Per glossary, dissolution is DISTINGUISHED from demotion: demotion withdraws OPERATIONAL USE; dissolution recognizes NON-EXISTENCE.

```sh
make dissolve-coordination-ring-build                                      # builds ./bin/dissolve-coordination-ring

./bin/dissolve-coordination-ring \
  -formation-event-hash <64-hex-chars> \
  -reason "interaction pattern was collection-bias artifact"
```

Validates the supplied `formation-event-hash` resolves to a `CoordinationRingFormation` row (otherwise exits 3). May be invoked regardless of whether the hypothesis was ever promoted.

| Option | Default | Notes |
|---|---|---|
| `-formation-event-hash` | (required) | Hex-encoded BLAKE3-256 of the target `CoordinationRingFormation`. |
| `-dissolved-at-ns` | 0 (= wall-clock now) | Explicit `dissolved_at` for forensic replay. |
| `-reason` | empty | **Strongly recommended** — dissolution is the terminal lifecycle operation. |

Exit codes: **0** success; **2** tool/configuration error; **3** target-not-found or target-wrong-type.

## `merge-coordination-rings` CLI

Operator-invoked tool to record the **CoordinationRing merge** lifecycle operation per [`§0074`](../../docs/charter/decision-log.md) — fifth lifecycle operation of the fourth Cat III subtype arc. Mirrors [`merge-hypotheses`](#merge-hypotheses-cli) (BC), [`merge-automation-groups`](#merge-automation-groups-cli) (AG), [`merge-campaign-hypotheses`](#merge-campaign-hypotheses-cli) (CH). Within-subtype only; cross-subtype merge deferred.

```sh
make merge-coordination-rings-build                                        # builds ./bin/merge-coordination-rings

./bin/merge-coordination-rings \
  -antecedent-a-hash <64-hex-chars> \
  -antecedent-b-hash <64-hex-chars> \
  -produced-formation-hash <64-hex-chars> \
  -reason "rings recognized as same coordinated-action phenomenon"
```

Symmetric (ascending-sort of antecedents). Validates all three hashes resolve to `CoordinationRingFormation` rows; identical antecedents return `ErrMergeAntecedentsIdentical` (exit 3). Per §0049 Option B, produced is a separately-committed `CoordinationRingFormation`.

Exit codes: **0** success; **2** tool/configuration error; **3** target-not-found, target-wrong-type, or identical-antecedents.

## `split-coordination-ring` CLI

Operator-invoked tool to record the **CoordinationRing split** lifecycle operation per [`§0075`](../../docs/charter/decision-log.md) — **sixth (final) lifecycle operation of the fourth Cat III subtype arc**. Mirrors `split-hypothesis` (BC), `split-automation-group` (AG), `split-campaign-hypothesis` (CH). Within-subtype only.

Closes the fourth Cat III subtype's §2.5 lifecycle surface — **all four subtypes (BehavioralCluster, AutomationGroup, CampaignHypothesis, CoordinationRing) now have 6 of 6 lifecycle operations structurally observable** (24 of 24 lifecycle event types across the §0010 four-subtype family).

```sh
make split-coordination-ring-build                                         # builds ./bin/split-coordination-ring

./bin/split-coordination-ring \
  -antecedent-formation-hash <64-hex-chars> \
  -successor-formation-hash <64-hex-chars> \
  -successor-formation-hash <64-hex-chars> \
  -reason "antecedent ring conflated two distinct coordinated phenomena"
```

Successors form a SET (ascending-sort idempotency per §0050). Cardinality MUST be ≥ 2; all entries MUST be byte-distinct from each other AND from the antecedent.

Exit codes: **0** success; **2** tool/configuration error; **3** target-not-found, target-wrong-type, insufficient-successors, or successors-not-distinct.

## `promote-campaign-hypothesis` CLI

Operator-invoked tool to record the **CampaignHypothesis promotion** lifecycle operation per [`§0064`](../../docs/charter/decision-log.md) — second lifecycle operation of the third Cat III subtype arc. Mirrors `promote-hypothesis` (BC) and `promote-automation-group` (AG).

```sh
make promote-campaign-hypothesis-build                                     # builds ./bin/promote-campaign-hypothesis

./bin/promote-campaign-hypothesis \
  -formation-event-hash <64-hex-chars> \
  -cadence-seconds 86400 \
  -reason "campaign promoted to operational enrichment"
```

Validates the supplied `formation-event-hash` resolves to a `CampaignHypothesisFormation` row (otherwise exits 3). Cross-subtype rejection: BC or AG formation hashes return `ErrTargetWrongType`.

Exit codes: **0** success; **2** tool/configuration error; **3** target-not-found or target-wrong-type.

## `demote-campaign-hypothesis` CLI

Operator-invoked tool to record the **CampaignHypothesis demotion** lifecycle operation per [`§0065`](../../docs/charter/decision-log.md). Mirrors `demote-hypothesis` (BC) and `demote-automation-group` (AG). Same Layer A cadence semantic per [`§0011`](../../docs/charter/decision-log.md).

```sh
make demote-campaign-hypothesis-build                                     # builds ./bin/demote-campaign-hypothesis

./bin/demote-campaign-hypothesis \
  -promotion-event-hash <64-hex-chars> \
  -reason "campaign cycle close"
```

Validates the supplied `promotion-event-hash` resolves to a `CampaignHypothesisPromotion` row. Per §0011 the cadence gate is CANDIDACY, not a hard barrier; report surfaces `cadence_satisfied` + `cadence_elapsed_seconds`.

Exit codes: **0** success; **2** tool/configuration error; **3** target-not-found or target-wrong-type.

## `dissolve-campaign-hypothesis` CLI

Operator-invoked tool to record the **CampaignHypothesis dissolution** lifecycle operation per [`§0066`](../../docs/charter/decision-log.md). Mirrors `dissolve-hypothesis` (BC) and `dissolve-automation-group` (AG). Per glossary, dissolution is DISTINGUISHED from demotion: demotion withdraws OPERATIONAL USE; dissolution recognizes NON-EXISTENCE.

```sh
make dissolve-campaign-hypothesis-build                                   # builds ./bin/dissolve-campaign-hypothesis

./bin/dissolve-campaign-hypothesis \
  -formation-event-hash <64-hex-chars> \
  -reason "campaign spurious"
```

Validates the supplied `formation-event-hash` resolves to a `CampaignHypothesisFormation` row.

Exit codes: **0** success; **2** tool/configuration error; **3** target-not-found or target-wrong-type.

## `merge-campaign-hypotheses` CLI

Operator-invoked tool to record the **CampaignHypothesis merge** lifecycle operation per [`§0067`](../../docs/charter/decision-log.md). Mirrors `merge-hypotheses` (BC) and `merge-automation-groups` (AG). Within-subtype only; cross-subtype merge deferred.

```sh
make merge-campaign-hypotheses-build                                       # builds ./bin/merge-campaign-hypotheses

./bin/merge-campaign-hypotheses \
  -antecedent-a-hash <64-hex-chars> \
  -antecedent-b-hash <64-hex-chars> \
  -produced-formation-hash <64-hex-chars> \
  -reason "campaigns recognized as same operation"
```

Symmetric (ascending-sort of antecedents). Validates all three hashes resolve to CampaignHypothesisFormation rows; identical antecedents return `ErrMergeAntecedentsIdentical` (exit 3). Per §0049 Option B, produced is a separately-committed CampaignHypothesisFormation.

Exit codes: **0** success; **2** tool/configuration error; **3** target-not-found, target-wrong-type, or identical-antecedents.

## `split-campaign-hypothesis` CLI

Operator-invoked tool to record the **CampaignHypothesis split** lifecycle operation per [`§0068`](../../docs/charter/decision-log.md) — **sixth (final) lifecycle operation of the third Cat III subtype arc**. Mirrors [`split-automation-group`](#split-automation-group-cli) and [`split-hypothesis`](#split-hypothesis-cli). Within-subtype only.

```sh
make split-campaign-hypothesis-build                                       # builds ./bin/split-campaign-hypothesis

./bin/split-campaign-hypothesis \
  -antecedent-formation-hash <64-hex-chars> \
  -successor-formation-hash <64-hex-chars> \
  -successor-formation-hash <64-hex-chars> \
  -reason "antecedent campaign conflated two distinct operations"
```

Successors form a SET (ascending-sort idempotency per §0050). Cardinality MUST be ≥ 2; all entries MUST be byte-distinct from each other AND from the antecedent. Closes the third-subtype lifecycle arc — **§2.5 lifecycle surface complete for CampaignHypothesis too** (6 of 6 ops landed: formation, promotion, demotion, dissolution, merge, split).

Exit codes: **0** success; **2** tool/configuration error; **3** target-not-found, target-wrong-type, insufficient-successors, or successors-not-distinct.

## `promote-automation-group` CLI

Operator-invoked tool to record the **AutomationGroup promotion** lifecycle operation per [`§0057`](../../docs/charter/decision-log.md) — second lifecycle operation of the second Cat III subtype arc. Mirrors [`promote-hypothesis`](#promote-hypothesis-cli) for the AutomationGroup subtype. Same Layer A cadence semantic per [`§0011`](../../docs/charter/decision-log.md).

```sh
make promote-automation-group-build                                       # builds ./bin/promote-automation-group

# Promote an AutomationGroup formation under a 24h cadence
./bin/promote-automation-group \
  -formation-event-hash <64-hex-chars> \
  -cadence-seconds 86400 \
  -reason "operational pilot — automation-signature attribution"
```

Validates the supplied `formation-event-hash` resolves to an `AutomationGroupFormation` row (otherwise exits 3 — preserves §2.5-lifecycle-integrity). Cross-subtype rejection: a `BehavioralClusterFormation` hash returns `ErrTargetWrongType` (exit 3) — the subtype-specific message_type discriminator prevents misclassification.

| Option | Default | Notes |
|---|---|---|
| `-formation-event-hash` | (required) | Hex-encoded BLAKE3-256 of the target `AutomationGroupFormation`. |
| `-cadence-seconds` | 86400 (24h) | Layer A parameter per [`§0011`](../../docs/charter/decision-log.md). |
| `-promoted-at-ns` | 0 (= wall-clock now) | Explicit `promoted_at` for forensic replay / deterministic test recording. |
| `-reason` | empty | Operator-supplied forensic note. |

Exit codes: **0** success; **2** tool/configuration error; **3** target-not-found or target-wrong-type.

## `demote-automation-group` CLI

Operator-invoked tool to record the **AutomationGroup demotion** lifecycle operation per [`§0058`](../../docs/charter/decision-log.md) — third lifecycle operation of the second Cat III subtype arc. Mirrors [`demote-hypothesis`](#demote-hypothesis-cli). Same Layer A cadence semantic per [`§0011`](../../docs/charter/decision-log.md).

```sh
make demote-automation-group-build                                        # builds ./bin/demote-automation-group

# Demote a specific AutomationGroup promotion event
./bin/demote-automation-group \
  -promotion-event-hash <64-hex-chars> \
  -reason "operational cycle close"
```

Validates the supplied `promotion-event-hash` resolves to an `AutomationGroupPromotion` row (otherwise exits 3). Per §0011 Layer A is a CANDIDACY gate, NOT a hard barrier — the substrate accepts the demotion regardless of whether the cadence has elapsed; the structured output surfaces `cadence_satisfied` + `cadence_elapsed_seconds`.

| Option | Default | Notes |
|---|---|---|
| `-promotion-event-hash` | (required) | Hex-encoded BLAKE3-256 of the target `AutomationGroupPromotion`. |
| `-demoted-at-ns` | 0 (= wall-clock now) | Explicit `demoted_at` for forensic replay. |
| `-reason` | empty | Operator-supplied forensic note; **strongly recommended** when demoting within the cadence window. |

Exit codes: **0** success; **2** tool/configuration error; **3** target-not-found or target-wrong-type.

## `dissolve-automation-group` CLI

Operator-invoked tool to record the **AutomationGroup dissolution** lifecycle operation per [`§0059`](../../docs/charter/decision-log.md) — fourth lifecycle operation of the second Cat III subtype arc. Mirrors [`dissolve-hypothesis`](#dissolve-hypothesis-cli). Per glossary, dissolution is DISTINGUISHED from demotion: demotion withdraws OPERATIONAL USE; dissolution recognizes NON-EXISTENCE.

```sh
make dissolve-automation-group-build                                      # builds ./bin/dissolve-automation-group

# Dissolve an AutomationGroup formation (signature misattributed)
./bin/dissolve-automation-group \
  -formation-event-hash <64-hex-chars> \
  -reason "signature misattributed"
```

Validates the supplied `formation-event-hash` resolves to an `AutomationGroupFormation` row (otherwise exits 3). May be invoked regardless of whether the hypothesis was ever promoted.

| Option | Default | Notes |
|---|---|---|
| `-formation-event-hash` | (required) | Hex-encoded BLAKE3-256 of the target `AutomationGroupFormation`. |
| `-dissolved-at-ns` | 0 (= wall-clock now) | Explicit `dissolved_at` for forensic replay. |
| `-reason` | empty | **Strongly recommended** — dissolution is the terminal lifecycle operation. |

Exit codes: **0** success; **2** tool/configuration error; **3** target-not-found or target-wrong-type.

## `merge-automation-groups` CLI

Operator-invoked tool to record the **AutomationGroup merge** lifecycle operation per [`§0060`](../../docs/charter/decision-log.md) — fifth lifecycle operation of the second Cat III subtype arc. Mirrors [`merge-hypotheses`](#merge-hypotheses-cli). Within-subtype only; cross-subtype merge deferred per existing entity-model.md.

```sh
make merge-automation-groups-build                                        # builds ./bin/merge-automation-groups

./bin/merge-automation-groups \
  -antecedent-a-hash <64-hex-chars> \
  -antecedent-b-hash <64-hex-chars> \
  -produced-formation-hash <64-hex-chars> \
  -reason "same automation signature"
```

Symmetric (argument order invariant): ascending-sort of antecedent hashes before recording. Validates all three hashes resolve to AutomationGroupFormation rows; identical antecedents return `ErrMergeAntecedentsIdentical` (exit 3). Mirrors §0049 Option B: the produced hypothesis identity is a separately-committed `AutomationGroupFormation`.

Exit codes: **0** success; **2** tool/configuration error; **3** target-not-found, target-wrong-type, or identical-antecedents.

## `split-automation-group` CLI

Operator-invoked tool to record the **AutomationGroup split** lifecycle operation per [`§0061`](../../docs/charter/decision-log.md) — **sixth (final) lifecycle operation of the second Cat III subtype arc**. Mirrors [`split-hypothesis`](#split-hypothesis-cli). Within-subtype only.

```sh
make split-automation-group-build                                         # builds ./bin/split-automation-group

./bin/split-automation-group \
  -antecedent-formation-hash <64-hex-chars> \
  -successor-formation-hash <64-hex-chars> \
  -successor-formation-hash <64-hex-chars> \
  -reason "antecedent conflated two automation signatures"
```

Successors form a SET (ascending-sort idempotency per §0050). Cardinality MUST be ≥ 2; all entries MUST be byte-distinct from each other AND from the antecedent. Closes the second-subtype lifecycle arc.

Exit codes: **0** success; **2** tool/configuration error; **3** target-not-found, target-wrong-type, insufficient-successors, or successors-not-distinct.

## `promote-hypothesis` CLI

Operator-invoked tool to record the second Cat III lifecycle operation (per [`§0046`](../../docs/charter/decision-log.md) — `BehavioralClusterPromotion`). Promotion transitions a hypothesis from active inference to operational use as enrichment context; per [Charter §2.5](../../docs/charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) + [`decision-log §0011`](../../docs/charter/decision-log.md), the event MUST carry the Layer A cadence parameter governing subsequent demotion-candidacy.

```sh
make promote-hypothesis-build                                          # builds ./bin/promote-hypothesis

# Promote a specific formation event (formation hash from form-hypothesis output)
./bin/promote-hypothesis \
  -formation-event-hash <64-hex-chars> \
  -cadence-seconds 86400 \
  -reason "operational pilot"

# Re-promotion under a different cadence_seconds re-records the
# cadence gate; the prior promotion event is preserved per §2.5 immutability
./bin/promote-hypothesis -formation-event-hash <hash> -cadence-seconds 3600
```

Validates the supplied `formation-event-hash` resolves to a `BehavioralClusterFormation` row in the substrate (otherwise exits 3 — preserves §2.5 lifecycle integrity: promotion references only formations, never observations or constructs). Commits the `BehavioralClusterPromotion` event via `substrate.Append` (acquires `writeMu` per [`concurrency-pattern.md`](../../docs/architecture/concurrency-pattern.md) §Substrate-Writer Serialization).

| Option | Default | Notes |
|---|---|---|
| `-formation-event-hash` | (required) | Hex-encoded BLAKE3-256 of the target `BehavioralClusterFormation`. |
| `-cadence-seconds` | 86400 (24 h) | Layer A parameter per [`§0011`](../../docs/charter/decision-log.md): elapsed time since `promoted_at` that opens demotion-candidacy. Layer B (deep criterion) remains deferred until §2.6 redacts. |
| `-promoted-at-ns` | 0 (= wall-clock now) | Explicit `promoted_at` for forensic replay / deterministic test recording. |
| `-reason` | empty | Operator-supplied forensic note; recommended at audit time. |

Exit codes: **0** success; **2** tool/configuration error; **3** target-not-found or target-wrong-type (the two §2.5-integrity errors).

## `demote-hypothesis` CLI

Operator-invoked tool to record the third Cat III lifecycle operation (per [`§0047`](../../docs/charter/decision-log.md) — `BehavioralClusterDemotion`). Demotion ends the operational use of a specific promotion event, closing the promote/demote loop per Charter §1 + §2.5.

```sh
make demote-hypothesis-build                                            # builds ./bin/demote-hypothesis

# Demote a specific promotion event (promotion hash from promote-hypothesis output)
./bin/demote-hypothesis \
  -promotion-event-hash <64-hex-chars> \
  -reason "scheduled rollover"

# Forensic-replay friendly form with explicit demoted_at
./bin/demote-hypothesis -promotion-event-hash <hash> -demoted-at-ns 1716120120000000000
```

Validates the supplied `promotion-event-hash` resolves to a `BehavioralClusterPromotion` row in the substrate (otherwise exits 3 — preserves §2.5-lifecycle-integrity: demotion references only promotions, never formations or observations). Commits the `BehavioralClusterDemotion` event via `substrate.Append`.

Per [`§0011`](../../docs/charter/decision-log.md) Layer A is a CANDIDACY gate, NOT a hard barrier. The CLI records demotion regardless of whether `cadence_seconds` has elapsed; the structured output surfaces:

- `cadence_satisfied` — `true` when `demoted_at - promotion.promoted_at >= promotion.cadence_seconds * 1e9`
- `cadence_elapsed_seconds` — actual elapsed seconds (negative if demoted_at precedes promoted_at)

Operators may demote within the cadence window (Layer A unsatisfied) when operational urgency or future Layer B criteria justify; the `reason` field is the only record of why the candidacy gate was bypassed and is strongly recommended in that case.

| Option | Default | Notes |
|---|---|---|
| `-promotion-event-hash` | (required) | Hex-encoded BLAKE3-256 of the target `BehavioralClusterPromotion`. |
| `-demoted-at-ns` | 0 (= wall-clock now) | Explicit `demoted_at` for forensic replay / deterministic test recording. |
| `-reason` | empty | Operator-supplied forensic note; **strongly recommended** when demoting within the cadence window. |

Exit codes: **0** success; **2** tool/configuration error; **3** target-not-found or target-wrong-type.

## `dissolve-hypothesis` CLI

Operator-invoked tool to record the fourth Cat III lifecycle operation (per [`§0048`](../../docs/charter/decision-log.md) — `BehavioralClusterDissolution`). Dissolution recognizes that the underlying phenomenon the hypothesis claimed to track no longer corresponds to anything — distinct from demotion per glossary + [`lifecycle-semantics.md`](../../docs/ontology/lifecycle-semantics.md) line 36: demotion withdraws OPERATIONAL USE; dissolution recognizes NON-EXISTENCE. The two operations are not interchangeable.

```sh
make dissolve-hypothesis-build                                          # builds ./bin/dissolve-hypothesis

# Dissolve a formation (may or may not have been promoted)
./bin/dissolve-hypothesis \
  -formation-event-hash <64-hex-chars> \
  -reason "phenomenon recognized as non-existent"

# Forensic-replay friendly form with explicit dissolved_at
./bin/dissolve-hypothesis -formation-event-hash <hash> -dissolved-at-ns 1716120180000000000
```

Validates the supplied `formation-event-hash` resolves to a `BehavioralClusterFormation` row in the substrate (otherwise exits 3 — preserves §2.5-lifecycle-integrity: dissolution references only formations, never promotions or demotions). Commits the `BehavioralClusterDissolution` event via `substrate.Append`.

Dissolution may be invoked regardless of whether the hypothesis was ever promoted. A formation that was never admitted to operational use can still be recognized as not corresponding to any phenomenon; the substrate accepts the discrete lifecycle event regardless of intermediate state. The projection layer reconstructs the full chain per §2.5 BC3.

| Option | Default | Notes |
|---|---|---|
| `-formation-event-hash` | (required) | Hex-encoded BLAKE3-256 of the target `BehavioralClusterFormation`. |
| `-dissolved-at-ns` | 0 (= wall-clock now) | Explicit `dissolved_at` for forensic replay / deterministic test recording. |
| `-reason` | empty | Operator-supplied forensic note; **strongly recommended** — dissolution is the terminal lifecycle operation on a hypothesis, and the absence of a reason removes the only record of the underlying judgment. |

Exit codes: **0** success; **2** tool/configuration error; **3** target-not-found or target-wrong-type.

## `merge-hypotheses` CLI

Operator-invoked tool to record the fifth Cat III lifecycle operation (per [`§0049`](../../docs/charter/decision-log.md) — `BehavioralClusterMerge`). Merge combines two hypotheses recognized as describing the same underlying phenomenon, per Charter §2.5 + [`lifecycle-semantics.md`](../../docs/ontology/lifecycle-semantics.md) line 28. Within-subtype only at this layer (both antecedents and the produced hypothesis are `BehavioralClusterFormation` events); cross-subtype merge per [`entity-model.md` §Cross-subtype operations](../../docs/ontology/entity-model.md) remains deferred to `lifecycle-semantics.md` post-Q4 redaction.

```sh
make merge-hypotheses-build                                              # builds ./bin/merge-hypotheses

# Merge two distinct BehavioralCluster formations, referencing a third
# (separately-committed) formation as the produced hypothesis
./bin/merge-hypotheses \
  -antecedent-a-hash <64-hex-chars> \
  -antecedent-b-hash <64-hex-chars> \
  -produced-formation-hash <64-hex-chars> \
  -reason "alpha and beta recognized as same phenomenon"
```

Validates all three supplied hashes resolve to `BehavioralClusterFormation` rows (otherwise exits 3 — preserves §2.5-lifecycle-integrity). Commits the `BehavioralClusterMerge` event via `substrate.Append`. The two antecedents MUST be distinct; passing identical antecedent hashes returns `ErrMergeAntecedentsIdentical` (exit 3).

**Argument-order invariance.** Merge is a symmetric relation per `lifecycle-semantics.md` line 28; the CLI sorts the two antecedent hashes ascending before recording so `-antecedent-a-hash A -antecedent-b-hash B` and `-antecedent-a-hash B -antecedent-b-hash A` produce a single substrate row (content-hash collision via normalization). The "A"/"B" labels are caller-facing only.

**Structural choice (per [`§0049`](../../docs/charter/decision-log.md)).** The merge event references both antecedent formation hashes AND a separately-committed produced formation hash — rather than collapsing the produced hypothesis identity into the merge event itself. This preserves the [`§0045`](../../docs/charter/decision-log.md) invariant ("hypothesis identity IS the formation event's content-hash") so all lifecycle operations (promote, demote, dissolve, future merge/split) continue targeting hypotheses through formation hashes uniformly. The produced formation must be created separately (typically via `form-hypothesis` against a substrate populated with the union of source observations) before the merge is recorded.

| Option | Default | Notes |
|---|---|---|
| `-antecedent-a-hash` | (required) | Hex-encoded BLAKE3-256 of the first `BehavioralClusterFormation` being merged. |
| `-antecedent-b-hash` | (required) | Hex-encoded BLAKE3-256 of the second `BehavioralClusterFormation` being merged. MUST differ from antecedent A. |
| `-produced-formation-hash` | (required) | Hex-encoded BLAKE3-256 of the separately-committed `BehavioralClusterFormation` representing the merged hypothesis. |
| `-merged-at-ns` | 0 (= wall-clock now) | Explicit `merged_at` for forensic replay / deterministic test recording. |
| `-reason` | empty | Operator-supplied forensic note; **strongly recommended** — the merge encodes a substantive epistemic claim. |

Exit codes: **0** success; **2** tool/configuration error; **3** target-not-found, target-wrong-type, or identical-antecedents (the §2.5-lifecycle-integrity errors).

## `split-hypothesis` CLI

Operator-invoked tool to record the sixth (and final) Cat III lifecycle operation (per [`§0050`](../../docs/charter/decision-log.md) — `BehavioralClusterSplit`). Split is the **structural inverse of merge** per [`§0049`](../../docs/charter/decision-log.md): merge is 2-to-1 (two antecedents → one produced), split is 1-to-N (one antecedent → multiple successors). Each successor is a separately-committed `BehavioralClusterFormation`, mirroring §0049's Option B so the [`§0045`](../../docs/charter/decision-log.md) invariant (hypothesis identity = formation content-hash) is preserved across all six lifecycle operations. Within-subtype only at this layer; cross-subtype split per [`entity-model.md` §Cross-subtype operations](../../docs/ontology/entity-model.md) remains deferred to `lifecycle-semantics.md` post-Q4 redaction.

```sh
make split-hypothesis-build                                              # builds ./bin/split-hypothesis

# Split a hypothesis into two (or more) successor hypotheses
./bin/split-hypothesis \
  -antecedent-formation-hash <64-hex-chars> \
  -successor-formation-hash <64-hex-chars> \
  -successor-formation-hash <64-hex-chars> \
  -reason "antecedent recognized as containing two distinct phenomena"
```

Validates the antecedent + every successor hash resolves to a `BehavioralClusterFormation` row (otherwise exits 3 — preserves §2.5-lifecycle-integrity). Commits the `BehavioralClusterSplit` event via `substrate.Append`. The successor set MUST contain at least 2 entries; all entries MUST be byte-distinct from each other AND from the antecedent (otherwise the operation is not a valid partition).

**Successor-order invariance.** Successors form a SET per `lifecycle-semantics.md` line 29; the CLI sorts them ascending before recording so `-successor-formation-hash A -successor-formation-hash B` and `-successor-formation-hash B -successor-formation-hash A` produce a single substrate row (content-hash collision via normalization).

| Option | Default | Notes |
|---|---|---|
| `-antecedent-formation-hash` | (required) | Hex-encoded BLAKE3-256 of the `BehavioralClusterFormation` being split. |
| `-successor-formation-hash` | (required, ≥ 2 invocations) | Hex-encoded BLAKE3-256 of a successor `BehavioralClusterFormation`; repeat the option for each successor. |
| `-split-at-ns` | 0 (= wall-clock now) | Explicit `split_at` for forensic replay / deterministic test recording. |
| `-reason` | empty | Operator-supplied forensic note; **strongly recommended** — split encodes a substantive epistemic claim (the antecedent conflated multiple phenomena). |

Exit codes: **0** success; **2** tool/configuration error; **3** target-not-found, target-wrong-type, insufficient-successors, or successors-not-distinct (the §2.5-lifecycle-integrity errors).

## `hypothesis-state` CLI

First operator-facing materializer of the **projection layer** over the §2.5 lifecycle event chain (per [`§0051`](../../docs/charter/decision-log.md)). Charter §2.5 BC3 forbids storing "current state" as a substrate row — the substrate stores immutable lifecycle events (formation, promotion, demotion, dissolution, merge, split) and the hypothesis's current state is reconstructed by replaying those events. This binary walks the substrate, builds the projection for a single formation, and emits structured JSON.

**Subtype-aware** (per [`§0062`](../../docs/charter/decision-log.md) + [`§0069`](../../docs/charter/decision-log.md) + [`§0076`](../../docs/charter/decision-log.md)): auto-detects the formation's Cat III subtype (`behavioral_cluster`, `automation_group`, `campaign_hypothesis`, or `coordination_ring`) by looking up the formation row's message_type, then dispatches to the appropriate per-subtype projection. The JSON output includes a `subtype` field. The projection layer covers **all four** §0010 Q2-resolved Cat III subtypes.

```sh
make hypothesis-state-build                                              # builds ./bin/hypothesis-state

# Project the current state of a hypothesis (formation hash from form-hypothesis output)
./bin/hypothesis-state \
  -formation-event-hash <64-hex-chars>
```

Output is structured JSON describing the projection: `formation_event_hash`, `state` (one of `forming`, `promoted`, `demoted`, `dissolved`, `merged_into`, `split_into`), optional `latest_promotion` / `latest_demotion` / `dissolution` / `merged_into` / `split_into` payloads, the full chronological `lifecycle_history` (entries sorted ascending by per-event timestamp), and `latencies` (per-projection latency derivations — see §0055).

**Latency fields (per §0055):** the `latencies` object surfaces three derived nanosecond intervals computed from the projection's lifecycle history:

| Field | When populated | Definition |
|---|---|---|
| `formation_to_first_promotion_ns` | hypothesis has been promoted at least once | EARLIEST promotion's `promoted_at` − formation's `event_time`. Answers "how long after formation did this hypothesis first move into operational use?" |
| `latest_promotion_to_latest_demotion_ns` | latest promotion has a corresponding demotion | `LatestDemotion.demoted_at` − `LatestPromotion.promoted_at` |
| `formation_to_dissolution_ns` | hypothesis has been dissolved | `Dissolution.dissolved_at` − formation's `event_time` |

Fields are absent (omitempty) when the underlying arc is incomplete. Values may be negative if the producer recorded out-of-order timestamps — the projection reports observed timestamps, not assertions about producer correctness.

**State precedence rules** (per `internal/projection.computeState`):

1. `dissolved` beats everything — dissolution recognizes non-existence per §0048 and is terminal.
2. `split_into` beats `merged_into` when both apply to the same formation (rare cross-arc; see `TestProjectSplitBeatsMerge`).
3. `merged_into` beats the promote/demote arc.
4. Within the promote/demote arc: `demoted` when a demotion targets the latest promotion; otherwise `promoted`.
5. Default: `forming` (formation row exists; no lifecycle event reaches it).

**Scope at this layer (per §0051):** single-hypothesis projection; on-demand walk; no caching or materialized indexes. Multi-hypothesis aggregate queries (e.g. "list every currently-promoted hypothesis") are deferred to a follow-on landing.

| Option | Default | Notes |
|---|---|---|
| `-formation-event-hash` | (required) | Hex-encoded BLAKE3-256 of the target `BehavioralClusterFormation`. |

Exit codes: **0** success; **2** tool/configuration error; **3** formation-not-found or target-not-formation.

## `list-hypotheses` CLI

Second binary in the **read-only** classification (per [`§0052`](../../docs/charter/decision-log.md)). Discharges the §0051 named carry-forward "Multi-hypothesis aggregate queries". Returns the projection summary for EVERY `BehavioralClusterFormation` in the substrate, with optional state filtering and paging.

```sh
make list-hypotheses-build                                               # builds ./bin/list-hypotheses

# Every hypothesis in the substrate, with its projected state
./bin/list-hypotheses -db ./ghost-trace.db -blobs ./blobs

# Filter to currently-promoted hypotheses
./bin/list-hypotheses -state promoted

# Page through results
./bin/list-hypotheses -limit 50 -offset 100
```

Output is a JSON ARRAY where each element shares the structure of `hypothesis-state`'s single-projection output (formation hash, state, optional per-lifecycle-type payloads, lifecycle event count, and `latencies` per §0055).

**Subtype-aware** (per [`§0062`](../../docs/charter/decision-log.md) + [`§0069`](../../docs/charter/decision-log.md) + [`§0076`](../../docs/charter/decision-log.md)): returns projections for ALL FOUR `behavioral_cluster`, `automation_group`, `campaign_hypothesis`, and `coordination_ring` subtypes by default. The `-subtype` option filters to one. Each entry includes a `subtype` field.

**Single linear walk per subtype over the substrate** (per `projection.ProjectAll` + `projection.ProjectAllAutomationGroups`): pass one collects formations + promotions; pass two dispatches demotions/dissolutions/merges/splits against the per-formation projections. Linear in substrate size, NOT in formation-count × substrate-size.

**Deterministic ordering**: ascending lex order of formation event hash (the content-hash). Substrate-position-independent — repeated calls against the same substrate return projections in the same order regardless of commit order.

| Option | Default | Notes |
|---|---|---|
| `-subtype` | empty | Filter by Cat III subtype: `behavioral_cluster`, `automation_group`, `campaign_hypothesis`, or `coordination_ring`. Empty = all subtypes. |
| `-state` | empty | Filter by projected state: one of `forming`, `promoted`, `demoted`, `dissolved`, `merged_into`, `split_into`. Empty = no filter. |
| `-after-ns` | 0 | Inclusive lower bound (Unix ns) on the **latest** event_time of each projection. 0 disables. |
| `-before-ns` | 0 | Inclusive upper bound (Unix ns) on the **latest** event_time. 0 disables. |
| `-limit` | 0 | Cap the number of projections returned (after subtype + state + time-window filtering). 0 = unbounded. |
| `-offset` | 0 | Skip the first N projections. |

**Time-window semantic (per §0054):** the filter matches against the projection's **latest** lifecycle event_time, not "any event in window". Operator pressure is "what's changed recently", not "what touches this window at all". Both bounds are inclusive.

Exit codes: **0** success (including empty results); **2** tool/configuration error (e.g. invalid `-state` value, negative `-after-ns`/`-before-ns`, `-after-ns > -before-ns`).

**Scope at this layer (per §0052):** Limit/Offset paging is sufficient for inception-phase substrate sizes. Cursor-based paging (resumable across substrate growth between calls) is deferred until operational pressure surfaces.

## `summarize-hypotheses` CLI

Third binary in the **read-only** classification (per [`§0053`](../../docs/charter/decision-log.md)). Discharges the §0052 named carry-forward "Aggregate counters / histograms". Returns counts of every Cat III formation grouped by subtype and projected state (per [`§0062`](../../docs/charter/decision-log.md)).

```sh
make summarize-hypotheses-build                                          # builds ./bin/summarize-hypotheses

# Per-state counts of every formation in the substrate
./bin/summarize-hypotheses -db ./ghost-trace.db -blobs ./blobs
```

Output is structured JSON with a **top-level `combined` section** (per [`§0078`](../../docs/charter/decision-log.md)) followed by four **per-subtype sections** (per [`§0062`](../../docs/charter/decision-log.md) + [`§0069`](../../docs/charter/decision-log.md) + [`§0076`](../../docs/charter/decision-log.md)). `combined`, `behavioral_cluster`, `automation_group`, `campaign_hypothesis`, and `coordination_ring` each carry `total` (formation count) + `by_state` (map from state value to count). Every State key is present (even at zero) — predictable wire shape.

The `combined` section is the per-state-aligned sum across the four per-subtype sections: `combined.total` equals the sum of subtype totals; `combined.by_state[s]` equals the sum of each subtype's `by_state[s]` for every state. Operators that want "every Cat III hypothesis, regardless of subtype, by state" read `combined` directly. The per-subtype sections remain available for subtype-specific drill-down.

**Latency aggregates per section (per [`§0079`](../../docs/charter/decision-log.md)).** Every section (the four per-subtype sections + `combined`) carries an additional `latencies` payload with three per-dimension aggregates mirroring the per-projection latency fields landed at [`§0055`](../../docs/charter/decision-log.md):

- `formation_to_first_promotion_ns`
- `latest_promotion_to_latest_demotion_ns`
- `formation_to_dissolution_ns`

Each aggregate carries:

| Field | Meaning |
|---|---|
| `sample_count` | Number of non-nil per-projection samples contributed to this aggregate. |
| `min_ns` | Smallest sample, or absent (omitempty) when `sample_count` is zero. |
| `p50_ns` | Median, nearest-rank method (no interpolation). Absent when zero samples. |
| `p90_ns` | 90th percentile, nearest-rank method. Absent when zero samples. |
| `max_ns` | Largest sample. Absent when zero samples. |

**Combined latency aggregates are exact, not approximated.** Per §0079, the combined section's `latencies` is computed from the UNION of per-subtype samples (not from per-subtype percentile values), so `combined.latencies.*.p50_ns` is the true median of all samples across the four subtypes rather than a function of the per-subtype p50s.

**Percentile method:** nearest-rank. For a sorted ascending sample of length N, P_k is the value at index `ceil(k * N / 100) - 1`. Stable + simple; no interpolation between samples.

**Equivalence invariant per §0053:** for every State value `s`, `by_state[s]` equals `len(list-hypotheses -state s)`. The equivalence is tested in `internal/projection/counts_test.go` and defends against precedence-rule drift between the count path and the list path. Both paths share `ProjectAll` (per [`§0052`](../../docs/charter/decision-log.md)).

| Option | Default | Notes |
|---|---|---|
| `-db` | `./ghost-trace.db` | SQLite primary-event-log path. |
| `-blobs` | `./blobs` | Content-addressed blob-store directory. |
| `-after-ns` | 0 | Inclusive lower bound (Unix ns) on the **latest** event_time of each projection. 0 disables. Per §0054. |
| `-before-ns` | 0 | Inclusive upper bound (Unix ns) on the **latest** event_time. 0 disables. Per §0054. |

**Time-window semantic (per §0054):** same latest-event semantic as `list-hypotheses`. The per-state-count equivalence invariant holds across the window filter: for every state `s`, `by_state[s]` equals `len(list-hypotheses -state s -after-ns X -before-ns Y)`.

Exit codes: **0** success (including empty substrate); **2** tool/configuration error (e.g. negative `-after-ns`/`-before-ns`, `-after-ns > -before-ns`).

## `replay-operational-session` CLI

Operator-invoked tool to verify deterministic **Phase 1 replay** of a Category II `OperationalSession` per [`§0084`](../../docs/charter/decision-log.md) + [`docs/architecture/replay-model.md`](../../docs/architecture/replay-model.md) L17-19. Re-derives the OperationalSession from its declared Cat I `DeclaredSession` source under the same operational definition + parameters recorded on the original record, then compares content-hashes.

```sh
make replay-operational-session-build                                      # builds ./bin/replay-operational-session

# Replay a specific OperationalSession by content-hash
./bin/replay-operational-session \
  -target-event-hash <64-hex-chars>
```

The OperationalSession self-describes its derivation rule via three fields landed at [`§0043`](../../docs/charter/decision-log.md): `definition_version`, `definition_parameters`, `source_event_hash`. Replay walks the substrate, resolves `definition_version` to a registered `OperationalDefinition` implementation (currently `padded-v1` per [`§0043`](../../docs/charter/decision-log.md) and `inactivity-window-v1` per [`§0044`](../../docs/charter/decision-log.md)), verifies the parameter-string round-trip, looks up the source `DeclaredSession`, re-runs the derivation under a freshly-collected `DerivationContext`, canonical-marshals + hashes the recomputed record, and compares to the substrate's committed hash.

| Field | Meaning |
|---|---|
| `target_event_hash` | Hex content-hash of the OperationalSession being replayed (input). |
| `recomputed_event_hash` | Hex content-hash of the freshly re-derived OperationalSession. |
| `match` | True iff `target_event_hash == recomputed_event_hash`. |
| `definition_version` | Version string read from the original. |
| `definition_parameters` | Canonical-parameter string read from the original. |
| `source_event_hash` | Hex content-hash of the source DeclaredSession. |

Exit codes:

- **0** — replay completed AND `match=true` (Phase 1 contract holds).
- **1** — replay completed AND `match=false` (**derivation drift detected**: the operational-definition implementation has changed since the original commit, OR the substrate's OperationalSession record is inconsistent with its declared derivation inputs).
- **2** — tool/configuration error.
- **3** — substrate-integrity precondition failure (`ErrTargetNotFound`, `ErrTargetWrongType`, `ErrDefinitionUnknown`, `ErrDefinitionParameterMismatch`, `ErrSourceNotFound`, `ErrSourceWrongType`).

The two failure modes are structurally distinct: exit 1 means replay ran and the contract failed; exit 3 means replay couldn't run because a precondition was missing.

**Scope:** this is Phase 1 replay over the only Cat II type committed today. Cat III hypothesis replay is Phase 3 reconstructive replay per [`replay-model.md`](../../docs/architecture/replay-model.md) L25-28 and is out of scope.

## `replay-behavioral-cluster-formation` CLI

Operator-invoked tool to verify **Phase 3 reconstructive replay** of a `BehavioralClusterFormation` per [`§0086`](../../docs/charter/decision-log.md) + [`replay-model.md`](../../docs/architecture/replay-model.md) L25-28. Walks the substrate filtered to events with `committed_at ≤ original_formation.committed_at`, re-runs the formation pattern against this substrate-at-commit-time view, and searches for a reconstructed formation whose canonical content-hash matches the original.

```sh
make replay-behavioral-cluster-formation-build                             # builds ./bin/replay-behavioral-cluster-formation

./bin/replay-behavioral-cluster-formation \
  -target-event-hash <64-hex-chars>
```

**Phase 3 vs Phase 1:** Phase 1 ([`replay-operational-session`](#replay-operational-session-cli)) verifies deterministic re-derivation of a single Cat II record from a single Cat I source. **Phase 3 reconstructs** a Cat III hypothesis from the substrate-at-commit-time view of all Cat I observations. Per [`replay-model.md`](../../docs/architecture/replay-model.md) L27 "re-deriving Phase 3 from scratch is acknowledged to potentially yield a different result" — but for our concrete patterns (session-descriptor-shared-v1) the formation IS deterministic given its FormationContext, so `match=true` is the expected outcome.

**Substrate-time filter:** the FormationContext passed to the replayed pattern contains only DeclaredSessions with `committed_at ≤ original.committed_at`. Late-arriving observations (those committed AFTER the formation was committed) are excluded — preserving the §2.1 + OMQ #3 substrate-time-authority invariant.

| Field | Meaning |
|---|---|
| `target_event_hash` | Hex content-hash of the BC formation being replayed. |
| `match` | True iff a reconstructed formation has byte-identical content-hash. |
| `recomputed_event_hash` | Hash of the matching reconstructed formation (omitted when no match). |
| `pattern_signature` | Pattern identifier read from the original. |
| `pattern_parameters` | Canonical-parameter string read from the original. |
| `reconstructed_formation_count` | Number of formations the pattern produced over the filtered context. |
| `contributing_observation_count` | DeclaredSessions visible at `committed_at ≤ max_committed_at_ns`. |
| `max_committed_at_ns` | Substrate-time bound (= original formation row's `committed_at`). |

Exit codes:

- **0** — match (Phase 3 contract holds).
- **1** — drift detected (no reconstructed formation has matching hash). Investigate pattern-implementation drift OR substrate-time vs event-time inconsistency.
- **2** — tool/configuration error.
- **3** — substrate-precondition failure (`ErrTargetNotFound`, `ErrTargetWrongType`, `ErrPatternUnknown`, `ErrPatternParameterMismatch`).

**Scope:** this entry covers BC. AutomationGroup, CampaignHypothesis, and CoordinationRing Phase 3 replay tools follow as separate landings (one-PR-per-subtype mechanical extension).

## `replay-automation-group-formation` CLI

Same shape as [`replay-behavioral-cluster-formation`](#replay-behavioral-cluster-formation-cli) for the AutomationGroup subtype per [`§0087`](../../docs/charter/decision-log.md). Currently supports the `uniform-cadence-v1` pattern.

```sh
make replay-automation-group-formation-build                               # builds ./bin/replay-automation-group-formation

./bin/replay-automation-group-formation \
  -target-event-hash <64-hex-chars>
```

Wire shape + exit codes identical to the BC replay CLI. The same `hypothesis.CollectFormationContextAt` helper backs both — by the §0056 typed-subtype-landings discipline, all four Cat III subtypes' formation contexts share the `DeclaredSessions()` surface, so one helper serves all four (the AG path performs an interface-to-interface assertion to convert `FormationContext` → `AutomationGroupFormationContext`).

## `replay-campaign-hypothesis-formation` CLI

Same shape as the BC + AG replay CLIs for the CampaignHypothesis subtype per [`§0088`](../../docs/charter/decision-log.md). Currently supports the `temporal-descriptor-cohort-v1` pattern. Third of four subtype-specific Phase 3 replay tools.

```sh
make replay-campaign-hypothesis-formation-build                            # builds ./bin/replay-campaign-hypothesis-formation

./bin/replay-campaign-hypothesis-formation \
  -target-event-hash <64-hex-chars>
```

## `replay-coordination-ring-formation` CLI

Same shape as the BC + AG + CH replay CLIs for the CoordinationRing subtype per [`§0089`](../../docs/charter/decision-log.md). Currently supports the `co-occurrence-window-v1` pattern. **Fourth (final) subtype-specific Phase 3 replay tool — closes the four-subtype Phase 3 arc opened at §0086.**

```sh
make replay-coordination-ring-formation-build                              # builds ./bin/replay-coordination-ring-formation

./bin/replay-coordination-ring-formation \
  -target-event-hash <64-hex-chars>
```

With this CLI, Phase 3 reconstructive replay covers all four §0010 Q2-resolved Cat III subtypes (BC, AG, CH, CR). Each subtype carries the same wire shape + exit-code semantic; the shared `hypothesis.CollectFormationContextAt` helper handles substrate-time filtering for all four via the interface-to-interface assertion pattern.

## `replay-all-formations` CLI

Substrate-wide Phase 3 batch replay across all four Cat III subtype formations per [`§0090`](../../docs/charter/decision-log.md). Default: replays every formation of every subtype. Optional `--subtype` filter narrows to one.

```sh
make replay-all-formations-build                                           # builds ./bin/replay-all-formations

# Audit Phase 3 reproducibility across all four Cat III subtypes
./bin/replay-all-formations -db ./ghost-trace.db -blobs ./blobs

# Single-subtype audit
./bin/replay-all-formations -subtype coordination_ring
```

Output JSON contains a section per selected subtype, each carrying the `BatchReplayReport` shape from [`§0085`](../../docs/charter/decision-log.md) (`total`, `matched`, `drifted`, `errored` + optional `drift` / `errors` arrays).

Exit codes (computed across all selected subtypes):

- **0** — every formation in every selected subtype matched.
- **1** — at least one drift detected.
- **2** — tool/configuration error.
- **3** — no drift but at least one substrate-precondition error.

Drift takes precedence over precondition error when both are present, mirroring the §0085 + §0086-§0089 convention.

## `replay-all-operational-sessions` CLI

Substrate-wide batch Phase 1 replay per [`§0085`](../../docs/charter/decision-log.md). Walks every `OperationalSession` in the substrate, re-derives each from its declared source, and reports aggregate match/drift/error counts. Pre-collects the `DerivationContext` once and reuses it across all per-target replays (cost: substrate walks = 2 + 1 lookup-per-record; vs N+1 walks if the per-target CLI were called naively in a loop).

```sh
make replay-all-operational-sessions-build                                 # builds ./bin/replay-all-operational-sessions

./bin/replay-all-operational-sessions -db ./ghost-trace.db -blobs ./blobs
```

Output is structured JSON with `total`, `matched`, `drifted`, `errored` counts + optional `drift` and `errors` arrays listing the non-matching entries:

```json
{
  "total": 42,
  "matched": 41,
  "drifted": 0,
  "errored": 1,
  "errors": [
    {
      "target_event_hash": "<hex>",
      "outcome": "error",
      "reason": "replay: source observation not found: <hex>"
    }
  ]
}
```

Invariant: `total == matched + drifted + errored`.

Exit codes:

- **0** — every OperationalSession matched (`matched == total`).
- **1** — at least one drift detected (`drifted > 0`). Phase 1 contract violated for at least one record; the originally-committed records remain authoritative per §2.1 but the derivation implementation has changed since commit. Investigate the implementation.
- **2** — tool/configuration error.
- **3** — at least one record errored (`errored > 0` AND no drift). A precondition failed (missing source, unknown definition version, etc.). Investigate substrate consistency.

When both drift and errors are non-zero, exit **1** takes precedence (drift is the stronger signal).

## `orphan-cleanup` CLI

Operator-invoked tool to delete orphan blobs (per [`§0041`](../../docs/charter/decision-log.md)). Per [`§0033` Anti-Patterns](../../docs/architecture/operational-ops.md), orphan deletion MUST be operator-invoked with explicit confirmation; this tool implements that discipline.

```sh
make orphan-cleanup-build                                                    # builds ./bin/orphan-cleanup

# Dry-run (default; safe; reports what WOULD be deleted)
./bin/orphan-cleanup -db ./ghost-trace.db -blobs ./blobs

# Confirmed deletion (requires both -dry-run=false AND -confirm)
./bin/orphan-cleanup -db ./ghost-trace.db -blobs ./blobs -dry-run=false -confirm

# With exclusion list (one hex hash per line; # comments allowed)
./bin/orphan-cleanup -dry-run=false -confirm -exclude ./preserve-these.txt

# Override default safety belts
./bin/orphan-cleanup -dry-run=false -confirm -keep-newer-than 1h -max-deletions 100
```

Safety belts (each independently configurable):

| Belt | Default | Override |
|---|---|---|
| Dry-run by default | `-dry-run=true` | explicit `-dry-run=false` required |
| Explicit confirmation | required when not dry-run | `-confirm` (or tool refuses with exit 2) |
| Age floor | `-keep-newer-than 24h` | `-keep-newer-than 0` disables; any duration |
| Per-invocation cap | `-max-deletions 1000` | `-max-deletions 0` disables |
| Exclusion list | none | `-exclude <path>` (one hash hex per line; `#` comments) |

Writes structured JSON to stdout (records of what was examined / preserved / deleted) + a brief human summary to stderr. Exit code: **0** on success (including dry-run); **2** on tool / configuration error (e.g. missing `-confirm` when not dry-run).

## Category I message types

Registered Cat I primary-observation types accepted by the ingestion pipeline (per [`§0042`](../../docs/charter/decision-log.md) — second Cat I type added). The dispatch registry at [`internal/ingest/dispatch.go`](./internal/ingest/dispatch.go) binds each type to its HTTP URL path + stdin envelope type identifier + event-time accessor. Adding a new type extends the registry, the corpus factory, the Makefile generate target, and lands a `.proto` under [`schemas/events/v1/`](../../schemas/events/v1/).

| Type | HTTP path | stdin `type` identifier | Event-time field | Producer class |
|---|---|---|---|---|
| `DeclaredSession` | `/v1/events/declared-session` | `declared_session` | `declared_at` | Client SDK (session-end report) |
| `NetworkEvent` | `/v1/events/network-event` | `network_event` | `observed_at` | Infrastructure collector (flow record / IDS event / packet summary) |

The substrate stores all types in the same events table with the `message_type` column carrying the Protobuf descriptor's full name (e.g. `ghosttrace.events.v1.DeclaredSession`, `ghosttrace.events.v1.NetworkEvent`). Verify + orphan-cleanup are type-agnostic and operate over heterogeneous-type substrates without change.

Category II `OperationalSession` records also live in the same events table — substrate immutability (Charter §2.1) applies to Cat II per [`entity-model.md` §Category II](../../docs/ontology/entity-model.md). Derivation is operator-invoked via [`cmd/derive-operational-session`](#derive-operational-session-cli); see that section for the registered operational definitions.

Category III hypothesis **lifecycle events** (per [Charter §2.5](../../docs/charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) BC5: "lifecycle events are Category I records under §2.5") also live in the same events table — `message_type` = `ghosttrace.events.v1.BehavioralClusterFormation` for the first Cat III subtype's formation event. Formation is operator-invoked via [`cmd/form-hypothesis`](#form-hypothesis-cli); see that section for the registered formation patterns. Per Charter §2.5 BC3, the substrate does NOT store a "current state" Cat III record — the hypothesis is reconstructed by replaying its lifecycle event chain (projection layer deferred).

## Build Sequence

Generated Protobuf bindings are NOT committed per [`§0024`](../../docs/charter/decision-log.md) AP3 ("Generated code is build output, not source"). First build sequence:

```sh
make tools     # installs protoc-gen-go locally (per internal/tools/tools.go pin)
make generate  # runs protoc against ../../schemas/events/v1/*.proto (every registered Cat I type + IngestionEvent)
make test      # go test -race ./...
make build     # go build -trimpath -o bin/ingestion .
```

Subsequent builds skip `make tools` unless `protoc-gen-go` is missing or out of date.

## Run

**Stdin/stdout (default):**

```sh
# Wire shape per line: {"type":"<type-id>","payload_b64":"<base64-proto>"}
echo '{"type":"declared_session","payload_b64":"<base64-Proto-DeclaredSession>"}' | ./bin/ingestion -db ./ghost-trace.db -blobs ./blobs
echo '{"type":"network_event","payload_b64":"<base64-Proto-NetworkEvent>"}'      | ./bin/ingestion -db ./ghost-trace.db -blobs ./blobs
```

Each input line is a JSON envelope where `type` selects a registered Cat I type (see §[Category I message types](#category-i-message-types) for the registered identifiers) and `payload_b64` is the base64-encoded canonical-Protobuf payload. Each input line produces one output line (JSON `confirmation` on success; `ingestError` on recoverable failure — envelope decode, unknown type, base64 decode, proto unmarshal). Unrecoverable substrate violations terminate the worker.

**HTTP (opt-in via `--http`):**

```sh
./bin/ingestion -db ./ghost-trace.db -blobs ./blobs -http :8080
```

Producers POST to `http://localhost:8080/v1/events/<type>` with `Content-Type: application/x-protobuf` and a Protobuf-marshaled body of the corresponding type:

```sh
curl -sS -X POST --data-binary @declared-session.bin \
  -H 'Content-Type: application/x-protobuf' \
  http://localhost:8080/v1/events/declared-session
curl -sS -X POST --data-binary @network-event.bin \
  -H 'Content-Type: application/x-protobuf' \
  http://localhost:8080/v1/events/network-event
```

The response is `200 OK` + JSON `confirmation` on success; `400 Bad Request` + JSON `ingestError` on recoverable input failures; `404 Not Found` + JSON `ingestError` (with the known-types list) when `<type>` is unregistered or the path is the bare `/v1/events`; `500 Internal Server Error` + JSON `ingestError` on unrecoverable substrate violations (which also trigger service shutdown). `GET /healthz` returns `200 OK` + `{"status":"ok"}`. The stdin worker runs simultaneously; both channels share the same single-writer mutex per [`concurrency-pattern.md`](../../docs/architecture/concurrency-pattern.md) §Substrate-Writer Serialization.

**HTTP with TLS termination (opt-in):**

```sh
./bin/ingestion -http :8443 \
  -http-tls-cert /etc/ghost-trace/cert.pem \
  -http-tls-key /etc/ghost-trace/key.pem
```

`--http-tls-cert` and `--http-tls-key` MUST both be set or both be empty. When set, the service serves HTTPS via `crypto/tls` with `MinVersion: TLS 1.2` (TLS 1.0/1.1 deprecated per RFC 8996). Both files are stat-checked at startup so misconfiguration fails fast rather than at first connection. ALPN auto-negotiates HTTP/2 when the client supports it (Go stdlib default). Bearer-token auth (next section) composes with TLS: the same `--http-auth-token-file` works under HTTPS. Cert reload on rotation requires a restart at inception phase; an online-reload follow-on is named in [`§0036`](../../docs/charter/decision-log.md) Out of Scope.

**HTTP with mTLS (opt-in; requires TLS):**

```sh
./bin/ingestion -http :8443 \
  -http-tls-cert /etc/ghost-trace/server-cert.pem \
  -http-tls-key /etc/ghost-trace/server-key.pem \
  -http-tls-client-ca /etc/ghost-trace/client-ca-bundle.pem
```

`--http-tls-client-ca` enables mutual-TLS authentication: every client MUST present a certificate signed by one of the CAs in the bundle. The server verifies via `tls.RequireAndVerifyClientCert` during the TLS handshake; connections without a valid client cert are rejected at the TLS layer (before any HTTP request is processed — no 401, no response body, just connection close). mTLS provides per-producer identity (the Common Name + SANs in the client cert), useful for multi-producer deployments where bearer tokens alone are insufficient. mTLS COMPOSES with bearer-token auth: when both are configured, BOTH must pass (defense in depth) — the producer presents a valid client cert AND sends `Authorization: Bearer <token>`. The client-CA file is read + parsed at startup; misconfiguration fails fast. Per-client-cert revocation (CRL / OCSP) is not exercised at inception; revoke clients by rotating the CA bundle + restarting the service.

**HTTP with bearer-token authentication (opt-in):**

```sh
# Production: token stored in a 0600-mode file (avoids process-listing leak).
echo -n "deployment-secret-token" > /etc/ghost-trace/ingestion.token
chmod 0600 /etc/ghost-trace/ingestion.token
./bin/ingestion -http :8080 -http-auth-token-file /etc/ghost-trace/ingestion.token

# Alternative (scripting/dev only): inline token.
./bin/ingestion -http :8080 -http-auth-token "dev-secret"
```

Producers MUST send `Authorization: Bearer <token>` with every `POST /v1/events/{type}`. Missing or wrong tokens return `401 Unauthorized` + JSON `ingestError` + `WWW-Authenticate: Bearer realm="ghost-trace-ingestion"`. Token comparison uses constant-time equality (`crypto/subtle.ConstantTimeCompare`); a length-mismatch leak channel exists but is acceptable at inception per [`§0035`](../../docs/charter/decision-log.md). `/healthz` is exempt from auth (orchestrator-friendly liveness probing); unknown paths return `401` (not `404`) when auth is configured, so the path structure is not leaked. Bearer tokens transmit credentials in plaintext on the wire — production deployments SHOULD also terminate TLS via reverse proxy (or a follow-on TLS RFC).

Signals (SIGINT, SIGTERM) trigger graceful shutdown via context cancellation; in-flight HTTP requests drain up to a 10-second grace window before the server returns from `Shutdown`.

### HTTP projection-read endpoints

Per [`§0080`](../../docs/charter/decision-log.md), the HTTP interface gains read-side endpoints exposing the same projection surface that the `hypothesis-state` / `list-hypotheses` / `summarize-hypotheses` CLIs serve. The first endpoint:

- **`GET /v1/hypotheses/state?formation_event_hash=<hex>`** — single-projection read, subtype auto-detected from the formation row's `message_type`. Mirrors `hypothesis-state` CLI wire shape: `subtype`, `formation_event_hash`, `state`, optional `latest_promotion` / `latest_demotion` / `dissolution` / `merged_into` / `split_into` payloads, chronological `lifecycle_history`, and `latencies` (per [`§0055`](../../docs/charter/decision-log.md)).

Response codes: **200** on success; **400** for missing parameter, invalid hex, or wrong hash length; **404** for unknown formation or cross-subtype rejection (target is not a Cat III formation event); **405** for non-GET; **503** if the handler was constructed without `WithSubstrate` (read endpoints disabled). When `WithAuthToken` is configured, the read endpoint requires `Authorization: Bearer <token>` the same way `POST /v1/events/*` does.

Example:

```sh
curl -sS \
  -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/v1/hypotheses/state?formation_event_hash=<64-hex-chars>"
```

- **`GET /v1/hypotheses?subtype=...&state=...&after_ns=...&before_ns=...&limit=...&offset=...`** (per [`§0081`](../../docs/charter/decision-log.md)) — multi-projection list mirroring `list-hypotheses` CLI. All query parameters optional; empty `subtype` / `state` disables that filter; `limit=0` is unbounded; `offset=0` starts at the first entry. Returns a JSON array of entries (each entry: `subtype`, `formation_event_hash`, `state`, optional per-lifecycle-type payloads, `lifecycle_event_count`, `latencies`). Response codes: **200** on success (including empty array); **400** for invalid parameter (unknown subtype/state, non-integer numeric, negative numeric, `after_ns > before_ns`); **405** non-GET; **503** substrate not configured.

```sh
# All subtypes, all states
curl -sS -H "Authorization: Bearer $TOKEN" "http://localhost:8080/v1/hypotheses"

# Filter to promoted CR hypotheses
curl -sS -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/v1/hypotheses?subtype=coordination_ring&state=promoted"

# Page through 50 at a time
curl -sS -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/v1/hypotheses?limit=50&offset=100"
```

- **`GET /v1/hypotheses/summary?after_ns=...&before_ns=...`** (per [`§0082`](../../docs/charter/decision-log.md)) — aggregate counters + latency aggregates mirroring `summarize-hypotheses` CLI. Output JSON has a top-level `combined` section + four per-subtype sections (`behavioral_cluster`, `automation_group`, `campaign_hypothesis`, `coordination_ring`), each carrying `total`, `by_state` (every State key present), and `latencies` (three per-dimension percentile aggregates with `sample_count` + `min_ns`/`p50_ns`/`p90_ns`/`max_ns`). Combined percentiles are exact (computed from union of per-subtype raw samples) per [`§0079`](../../docs/charter/decision-log.md). Response codes: **200** on success; **400** for invalid numeric or `after_ns > before_ns`; **405** non-GET; **503** substrate not configured.

```sh
curl -sS -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/v1/hypotheses/summary"

# Time-windowed summary
curl -sS -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/v1/hypotheses/summary?after_ns=1700000000000000000&before_ns=1800000000000000000"
```

**All three CLI surfaces now have HTTP parity.** `hypothesis-state` (§0080), `list-hypotheses` (§0081), and `summarize-hypotheses` (§0082) are reachable over HTTP with the same wire shapes as their CLI counterparts.

## Required Properties

Per the original constitutional placeholder ([decision-log §0022](../../docs/charter/decision-log.md) implementation pivot):

- **Idempotent commitment** — a producer retry produces no duplicate records in the events table. Enforced by `INSERT OR IGNORE` on `event_hash BLOB PRIMARY KEY` per [`§0027`](../../docs/charter/decision-log.md) AP6 + content-addressing.
- **Producer-time preservation** — `event_time` column records the producer-reported time, accessed per type via the dispatch registry (`DeclaredSession.declared_at`, `NetworkEvent.observed_at`); `committed_at` column records the system's commit time.
- **Source attribution** — `actor_ref` field per [`§0023`](../../docs/charter/decision-log.md) Q2 Identity tiers resolution (inception-phase single-tier); optional on collector-reported types where attribution may be absent at collection time.
- **Schema validation** — `canonical.Marshal` uses `AllowPartial: false` rejecting messages missing required fields; `proto.Unmarshal` rejects ill-formed wire bytes.

## Constitutional + Architecture Anchors

- [Charter §2.1 Observational Integrity](../../docs/charter/constitutional-charter.md#21-observational-integrity), [§2.2](../../docs/charter/constitutional-charter.md#22-epistemic-separation), [§2.3](../../docs/charter/constitutional-charter.md#23-provenance-integrity), [§2.5](../../docs/charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness).
- [`decision-log §0022`](../../docs/charter/decision-log.md) (implementation pivot), [`§0023`](../../docs/charter/decision-log.md) (Q2 actor_ref), [`§0024`](../../docs/charter/decision-log.md) (Protobuf proto3 + AP3/AP5/AP6), [`§0025`](../../docs/charter/decision-log.md) (Go), [`§0027`](../../docs/charter/decision-log.md) (SQLite + blob-store; AP4/AP5/AP6), [`§0028`](../../docs/charter/decision-log.md) (canonical-serialization-contract), [`§0029`](../../docs/charter/decision-log.md) (concurrency-pattern), [`§0030`](../../docs/charter/decision-log.md) (this skeleton).
- [`docs/architecture/canonical-serialization-contract.md`](../../docs/architecture/canonical-serialization-contract.md) — bit-stable marshal + hash.
- [`docs/architecture/concurrency-pattern.md`](../../docs/architecture/concurrency-pattern.md) — goroutine + channel + context + substrate-writer-serialization discipline.

## Out of Scope

Per skeleton-status discipline, the following are deferred to follow-on commits:

- ~~HTTP interface~~ **partially discharged at [`decision-log §0034`](../../docs/charter/decision-log.md).** `POST /v1/events/{type}` + `GET /healthz` implemented in [`internal/httpapi`](./internal/httpapi); opt-in via `--http :8080`. gRPC remains deferred per [`§0025`](../../docs/charter/decision-log.md) Open Questions; HTTP authentication, rate limiting, and TLS termination are out of scope (reverse-proxy concern at inception).
- **Backup/recovery automation.** Manual `.backup` + `rsync` per [`§0027`](../../docs/charter/decision-log.md) Proposal item 5; ordering matters (blob-store first, then SQLite).
- ~~Canonical-corpus population.~~ **Discharged at [`decision-log §0031`](../../docs/charter/decision-log.md).** Two corpus entries cover `DeclaredSession`; discovery-based test + `-update` regeneration via `make golden-corpus`; CI golden-file gate operational.
- ~~Unrecoverable-error shutdown escalation.~~ **Discharged at [`decision-log §0032`](../../docs/charter/decision-log.md).** `readLoop` classifies errors via `isUnrecoverable`; substrate §2.1-violation errors (`substrate.ErrHashMismatch`, `substrate.ErrBlobCollision`) terminate the worker, propagate through errgroup, and trigger `main()` to write a structured fatal record to stderr + exit non-zero. Recoverable errors (bad input) still emit per-message JSON entries to stdout and continue processing. Tested in `main_test.go` (6 tests; both paths exercised).
- ~~Multiple Category I message types.~~ **Partially discharged at [`decision-log §0042`](../../docs/charter/decision-log.md).** Second Cat I type (`NetworkEvent`) added; dispatch registry at [`internal/ingest/dispatch.go`](./internal/ingest/dispatch.go) makes the addition mechanical (see §[Category I message types](#category-i-message-types)). Additional types (fingerprint snapshots, external authoritative state changes) extend the registry as their schemas land.
