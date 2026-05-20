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

**Other lifecycle operations** per [Charter §2.5](../../docs/charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness): promotion landed at [`§0046`](../../docs/charter/decision-log.md) via [`cmd/promote-hypothesis`](#promote-hypothesis-cli); demotion at [`§0047`](../../docs/charter/decision-log.md) via [`cmd/demote-hypothesis`](#demote-hypothesis-cli); dissolution at [`§0048`](../../docs/charter/decision-log.md) via [`cmd/dissolve-hypothesis`](#dissolve-hypothesis-cli); merge at [`§0049`](../../docs/charter/decision-log.md) via [`cmd/merge-hypotheses`](#merge-hypotheses-cli); split at [`§0050`](../../docs/charter/decision-log.md) via [`cmd/split-hypothesis`](#split-hypothesis-cli). **§2.5 lifecycle surface complete — all 6 of 6 operations now structurally observable.** Per Charter §2.5 BC3, the substrate stores ONLY lifecycle events — the hypothesis's current state is a projection over the operation event chain (projection layer deferred).

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

```sh
make hypothesis-state-build                                              # builds ./bin/hypothesis-state

# Project the current state of a hypothesis (formation hash from form-hypothesis output)
./bin/hypothesis-state \
  -formation-event-hash <64-hex-chars>
```

Output is structured JSON describing the projection: `formation_event_hash`, `state` (one of `forming`, `promoted`, `demoted`, `dissolved`, `merged_into`, `split_into`), optional `latest_promotion` / `latest_demotion` / `dissolution` / `merged_into` / `split_into` payloads, and the full chronological `lifecycle_history` (entries sorted ascending by per-event timestamp).

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

Output is a JSON ARRAY where each element shares the structure of `hypothesis-state`'s single-projection output (formation hash, state, optional per-lifecycle-type payloads, lifecycle event count).

**Single linear walk over the substrate** (per `projection.ProjectAll`) regardless of formation count: pass one collects formations + promotions; pass two dispatches demotions/dissolutions/merges/splits against the per-formation projections. Linear in substrate size, NOT in formation-count × substrate-size.

**Deterministic ordering**: ascending lex order of formation event hash (the content-hash). Substrate-position-independent — repeated calls against the same substrate return projections in the same order regardless of commit order.

| Option | Default | Notes |
|---|---|---|
| `-state` | empty | Filter by projected state: one of `forming`, `promoted`, `demoted`, `dissolved`, `merged_into`, `split_into`. Empty = no filter. |
| `-limit` | 0 | Cap the number of projections returned. 0 = unbounded. |
| `-offset` | 0 | Skip the first N projections (after state filtering, after ordering). |

Exit codes: **0** success (including empty results); **2** tool/configuration error (e.g. invalid `-state` value).

**Scope at this layer (per §0052):** Limit/Offset paging is sufficient for inception-phase substrate sizes. Cursor-based paging (resumable across substrate growth between calls) is deferred until operational pressure surfaces.

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
