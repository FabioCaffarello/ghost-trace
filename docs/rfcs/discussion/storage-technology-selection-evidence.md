# Inception-phase storage technology selection — discussion evidence

**Status:** in-discussion. Not authoritative. Final resolution will be recorded in decision-log and the RFC's `Decision Record` section at acceptance.

This scratch supports the discussion phase of the [storage-technology selection RFC](../draft/architecture-storage-technology-selection.md) — third of three technology RFCs authorized by [`decision-log §0022`](../../charter/decision-log.md). Final inception-phase technology gate. Follows the [`schemas`](./schemas-technology-selection-evidence.md) + [`implementation-language`](./implementation-language-selection-evidence.md) evidence-file pattern adapted for storage selection.

The RFC's draft positions: **SQLite + content-addressed blob-store on local filesystem** recommended; **NATS JetStream / Kafka / Parquet on S3-MinIO / PostgreSQL** rejected as admissible-but-deferred; **plain files + custom index** rejected on reimplementation-cost grounds. This scratch evaluates the recommended candidate across four Charter-frozen-invariant dimensions, re-tests each rejection rationale, applies the three rfc-author §3 discipline skills, surfaces non-obvious risks, and produces a recommendation with reversal conditions.

## Phase 1 — Recommended candidate (SQLite + content-addressed blob-store) per frozen invariant

Four cells. Verdict per cell: clean | conditional | violation. Source citations after each.

### Dimension 1 — §2.1 Observational Integrity (frozen)

§2.1 requires append-only by construction with content-addressable identifiers sufficient to detect mutation. SQLite supports application-level append-only via `INSERT`-only discipline; content-addressing via BLAKE3 hash of canonical Protobuf bytes per [`§0024`](../../charter/decision-log.md) AP5.

**SQLite + blob-store (recommended).** **Clean with discipline obligations.** Three substrate-relevant operational properties surface:
- **WAL mode + fsync behavior.** SQLite WAL mode is the standard configuration for concurrent-reader + single-writer durability. The default `PRAGMA synchronous=NORMAL` has a small window on power loss; `PRAGMA synchronous=FULL` is required for §2.1 commit guarantee. RFC Proposal item 1 does not currently pin this. Add to resolution-phase modifications.
- **`VACUUM` operation.** SQLite's `VACUUM` rebuilds the file, preserving logical content but changing on-disk byte representation. This is NOT a §2.1 violation: content-hashes are computed over Protobuf payload bytes (the blob-store's content), not over SQLite file bytes. VACUUM is a storage-layer reorganization that preserves the substrate's logical commitments. Surface as Open Question for operational cadence (VACUUM during ingestion may affect performance + crash semantics).
- **Schema migration restriction.** `ALTER TABLE events` would be a §2.1 concern (the events table is the substrate). Mitigation: events table structure is frozen at v0.1; future event categories surface via new `message_type` values, not new columns. Lifecycle-event additions per §2.5 inherit the same pattern.
- *Citation:* [SQLite WAL documentation](https://www.sqlite.org/wal.html). [SQLite PRAGMA synchronous documentation](https://www.sqlite.org/pragma.html#pragma_synchronous). [`§0024`](../../charter/decision-log.md) AP5 hash-stability commitment.

### Dimension 2 — §2.2 Epistemic Separation (frozen)

§2.2 requires schema-level nominally-distinct categorical types. Charter §2.2 Boundary Conditions: "The three categories share infrastructure — storage, transport, indexing, observability tooling — but not type." Shared storage infrastructure is permitted; categorical separation is preserved by the Protobuf message types embedded in the payload bytes.

**SQLite + blob-store.** **Clean with discipline obligation surfaced to §0025 AP1 inheritance.** The events table with `message_type` column is structurally distinct from a unified-record-with-discriminator at the schemas layer (the Charter §2.2 forbidden anti-pattern). The `message_type` column is an index field for query at the storage layer, NOT a categorical discriminator at the type layer. Critical discipline obligation: at the application code reading FROM SQLite, the Go-level decoding dispatch on `message_type` to the correct Protobuf message type IS a cross-category boundary function in the [`§0025`](../../charter/decision-log.md) AP1 sense (cross-category type imports without explicit boundary function). A bug in the decode dispatch (decode a Cat I observation as Cat II construct) would be a runtime §2.2 violation. The decode dispatch must be a documented + tested boundary function; [`§0025`](../../charter/decision-log.md) AP1 lint rule applies to the storage-decode path explicitly.
- *Citation:* [Charter §2.2 Boundary Conditions L101](../../charter/constitutional-charter.md#22-epistemic-separation). [`§0025`](../../charter/decision-log.md) AP1 cross-category-import discipline.

### Dimension 3 — §2.3 Provenance Integrity (frozen v0.4)

§2.3 requires that typed `subject_ref_*` references per [`§0016`](../../charter/decision-log.md) Q3 resolution are reconstructible from substrate. References are part of the Protobuf payload (oneOf-encoded per [`§0024`](../../charter/decision-log.md) Phase 1 D3); substrate stores the payload bytes; reconstruction reads + decodes.

**SQLite + blob-store.** **Clean.** No storage-technology-specific concern. The substrate stores Protobuf bytes; oneOf-exclusivity is enforced at the schemas layer per [`§0024`](../../charter/decision-log.md); reconstruction at read time is a decode operation. The blob-store's content-addressing supplies the reference-resolvability guarantee (a `subject_ref_observation` whose hash does not resolve to any blob is a chain-rupture per §2.3 AP6 or AP7).
- *Citation:* [Charter §2.3 frozen v0.4 Structural Requirement L113](../../charter/constitutional-charter.md#23-provenance-integrity). [`§0016`](../../charter/decision-log.md) Q3 resolution. [`§0024`](../../charter/decision-log.md) Phase 1 D3.

### Dimension 4 — §2.5 Hypothesis Lifecycle Explicitness (frozen v0.3)

§2.5 requires lifecycle events committed immutably as Cat I records. Same storage requirement as §2.1 (append-only events table); no §2.5-specific concern.

**SQLite + blob-store.** **Clean.** Lifecycle events (formation, merge, split, promotion, demotion, dissolution) are additional `message_type` values in the events table; same append-only discipline + content-addressing applies. Merge events' antecedent references (per §2.5 Structural Requirement) are in the Protobuf payload; substrate-reconstructibility from the events table reads + decodes.
- *Citation:* [Charter §2.5 Structural Requirement L162](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness). [Charter §2.1 Structural Requirement L55](../../charter/constitutional-charter.md#21-observational-integrity).

## Phase 2 — Alternative rejection re-test

Five cells. Four admissible-but-deferred (NATS JetStream, Kafka, Parquet/S3, PostgreSQL); one structural-permanent (plain files + custom index).

### Cell A — NATS JetStream re-test

**RFC's rejection rationale:** admissible-but-deferred. JetStream is purpose-built event-stream substrate with native append-only + replication + retention; rejection is operational (running server process + configuration + network operations).

**Re-test.** JetStream IS structurally well-suited for the substrate (append-only by design; content-addressing via deterministic ID generation; durability via replication). The operational concern holds at inception (single-process services, no need for multi-writer or replication). The admissible-but-deferred shape is correct. **Verdict:** preserved.
- *Citation:* [NATS JetStream documentation](https://docs.nats.io/nats-concepts/jetstream). RFC Alternatives NATS bullet.

### Cell B — Kafka re-test

**RFC's rejection rationale:** admissible-but-deferred. Kafka is canonical event-streaming substrate at scale; rejection is operational (broker + ZooKeeper/KRaft + topic configuration + consumer-group management).

**Re-test.** Kafka topics are append-only by design; partitioning + replication supply durability beyond SQLite's single-node guarantee. The operational dependency is heavier than JetStream's at inception. Reversal R-store-1 captures both NATS + Kafka under the same throughput/multi-writer pressure trigger; the within-streaming-substrate choice (NATS vs Kafka) is itself a follow-on RFC concern when reversal fires. **Verdict:** preserved.
- *Citation:* [Apache Kafka documentation](https://kafka.apache.org/documentation/). RFC Alternatives Kafka bullet.

### Cell C — Parquet on S3 / MinIO re-test

**RFC's rejection rationale:** scope-mismatch. Parquet is columnar, optimized for analytical queries; S3/MinIO supplies object storage. Suited for cold archive (Tier 2 per [`storage-model.md`](../../architecture/storage-model.md)), NOT hot ingestion. Reversal R-store-2 captures cold-archive volume threshold.

**Re-test.** The scope-mismatch framing is structurally correct: Parquet's strength (columnar batched write + analytical read) is orthogonal to inception-phase needs (row-at-a-time write + immediate read for replay). The "admissible-when-cold-archive-tier-is-introduced" reversal shape is methodologically distinct from NATS/Kafka's "admissible-when-throughput-pressure-demands" shape — both are admissible-but-deferred but trigger on different conditions. **Verdict:** preserved. Worth noting: this is the first non-substrate-tier alternative (it's a Tier 2 candidate, not a Tier 1 substitute); the rejection-shape correctly reflects that.
- *Citation:* [Apache Parquet documentation](https://parquet.apache.org/docs/). RFC Alternatives Parquet bullet.

### Cell D — Plain files + custom index re-test

**RFC's rejection rationale:** reimplementation-cost grounds (structural, not deferred). SQLite provides transactional commit + crash-recovery + indexing + WAL semantics; implementing these against plain files reimplements substantial reliability-critical code.

**Re-test.** The cleanest structural rejection in the evaluated set. Concurrent-writer durability against power loss is famously subtle; commercial databases (SQLite included) have decades of testing against this category of bug. Building a custom equivalent at inception phase trades empirical-iteration speed for reimplementation work. **Verdict:** preserved. The "no revisit condition" classification is correct — this is structural-and-permanent rejection, methodologically distinct from the four admissible-but-deferred alternatives.
- *Citation:* [SQLite "How SQLite Is Tested" documentation](https://www.sqlite.org/testing.html). RFC Alternatives plain-files bullet.

### Cell E — PostgreSQL re-test

**RFC's rejection rationale:** admissible-but-deferred. PostgreSQL has stronger append-only-enforcement primitives than SQLite (REVOKE UPDATE/DELETE at SQL level + row-level security + logical replication); rejection is operational (running server process). Reversal R-store-3 captures multi-service ingestion sharing.

**Re-test.** PostgreSQL's enforcement primitives ARE structurally stronger than SQLite's (the database-level REVOKE is harder to bypass than an application-level discipline + trigger). The operational concern is the same shape as NATS/Kafka but a different category: PostgreSQL is a relational substrate option, not a streaming option. At inception with single-process ingestion, SQLite's no-server-process property dominates. **Verdict:** preserved with one consideration: PostgreSQL might be the right reversal target when multi-service ingestion arrives (R-store-3) but NOT for throughput pressure (R-store-1, which favors streaming substrate). The reversal-condition-to-target mapping is correctly differentiated.
- *Citation:* [PostgreSQL REVOKE documentation](https://www.postgresql.org/docs/current/sql-revoke.html). RFC Alternatives PostgreSQL bullet.

## Phase 3 — Discipline skills application

### 3.1 falsifiability-check (V/O/Op/NC four-question test)

Applied to RFC substantive claims in Summary, Motivation, Proposal, Migration sections.

| # | Claim (paraphrased) | V | O | Op | NC | Verdict |
|---|---|---|---|---|---|---|
| FC1 | "SQLite supports application-level append-only discipline (INSERT-only on event rows; no UPDATE or DELETE permitted)" (Motivation) | ✓ violation = UPDATE statement succeeds against events table | ✓ inspect SQL execution outcome | ✓ "append-only" = SQL operation set restricted to INSERT | ✓ SQL spec external | **Pass** |
| FC2 | "Content-addressing via BLAKE3 hash supplies the mutation-detection mechanism" (Motivation) | ✓ violation = mutated content produces different hash that does not match stored hash | ✓ recompute hash on read, compare | ✓ "content-addressing" = BLAKE3 spec | ✓ BLAKE3 spec external | **Pass** |
| FC3 | "modernc.org/sqlite preserves the §0025 static-binary commitment" (Proposal item 4) | ✓ violation = `ldd binary` reports dynamic library dependency | ✓ run `ldd`, inspect output | ✓ "static-binary commitment" = §0025 Proposal item 5 | ✓ §0025 spec internal | **Pass** |
| FC4 | "Single-file substrate has no operational dependencies" (Summary) | ⚠ "no operational dependencies" comparative | ⚠ depends on what counts as dependency | ⚠ "dependencies" not fully operationalized | ✓ vocabulary OK | **Pass-with-caveat (rewrite suggested):** rephrase as "Single-file substrate requires no external server process, no network configuration, and no out-of-process coordination to operate; the SQLite library is statically linked per Proposal item 4." |
| FC5 | "Reversal cost is bounded" (Migration) | ⚠ "bounded" comparative | ⚠ depends on bound | ⚠ "bounded" not operationalized | ✓ vocabulary OK | **Pass-with-caveat (rewrite suggested):** rephrase as "Reversal cost is bounded by re-read + re-write throughput against the target substrate — a one-time migration cost; no re-derivation of historical Protobuf bytes is required (in contrast to §0024 F6 schemas-technology reversal, which requires re-serialization of all historical records)." |
| FC6 | "An attempted write of a hash that already exists is a no-op (idempotent commit)" (Proposal item 2) | ✓ violation = duplicate write triggers an error or overwrites | ✓ test deliberately, inspect outcome | ✓ "idempotent commit" = same input produces same outcome | ✓ standard CS terminology | **Pass** |
| FC7 | "VACUUM is a storage-layer reorganization that preserves the substrate's logical commitments" (Phase 1 D1; if recorded in RFC) | ✓ violation = post-VACUUM events table missing rows or with mutated content-hash references | ✓ replay before + after VACUUM, compare | ✓ "logical commitments" = events-table row preservation | ✓ SQLite spec external | **Pass** (note: this claim from Phase 1; consider promoting to RFC Anti-Patterns boundary discussion) |

**Falsifiability summary:** 5 pass clean (FC1–FC3, FC6, FC7) + 2 pass-with-caveat (FC4, FC5 — both rewritable via operationalization). No claim fails.

### 3.2 epistemic-separator (paragraph-level categorical distinctness)

- **ES1 — Pass.** Summary categorically distinct from constitutional commitment.
- **ES2 — Pass.** Motivation paragraphs cite frozen invariants explicitly + technology-fit-to-invariant; categorical structure clean.
- **ES3 — Pass.** Proposal items are concrete technology commitments.
- **ES4 — Borderline / surface for discussion.** Anti-Patterns AP4 ("Hash-mismatch on read tolerated silently") mixes a §2.1 invariant claim (mutation detected = violation) with an operational discipline claim (service MUST recompute + compare + report). Suggestion: split into two anti-patterns — one §2.1-invariant-restatement (mutation-detected-is-violation per §2.1 inheritance), one operational-discipline (read-path-must-verify-hash). Resolution-phase rewrite consideration.
- **ES5 — Pass.** Alternatives section cleanly separates admissible-but-deferred (4) from structural-permanent (1) rejection categories.

**Epistemic-separation summary:** 4 pass, 1 borderline-with-rewrite-suggestion.

### 3.3 ambiguity-reducer (advisory term flagging)

Pre-commit hook surfaced 6 ambiguity advisories at draft-commit time on standard project terms (event, record, source, identity, state, context). All acceptable on inspection — canonical use throughout, no informal reuse of canonical vocabulary.

**Ambiguity-reducer summary:** 6 advisories acceptable as-is.

## Phase 4 — Non-obvious risks and findings

### F1 — `PRAGMA synchronous=FULL` commitment

SQLite WAL mode's default `synchronous=NORMAL` has a small window on power loss that can lose recent writes. §2.1's commit guarantee requires `synchronous=FULL` (slower; durable). RFC Proposal item 1 does not pin this. **Action for resolution phase:** Proposal item 1 expanded to commit to `PRAGMA synchronous=FULL` + `PRAGMA journal_mode=WAL` as the canonical configuration; CI integration test verifies these settings.

### F2 — POSIX-only inception-phase constraint

Blob-store anti-pattern (RFC AP3) commits to write-temp-then-rename for blob atomicity; POSIX `rename()` is atomic on overwriting existing path. On Windows, `MoveFileEx` with `MOVEFILE_REPLACE_EXISTING` is the equivalent but has different semantics for in-use files. **Action for resolution phase:** Open Question added — inception-phase substrate is POSIX-only (Linux, macOS); Windows support requires either substrate-technology selection re-evaluation or a different blob-store backend. Surface for committee deliberation.

### F3 — Single-writer constraint surfaced explicitly

SQLite WAL mode supports concurrent readers but only single writer per database file. At inception with single ingestion service, not load-bearing; surfaced explicitly so the constraint is visible at substrate-tier-introduction (R-store-3 reversal target — multi-service ingestion sharing). **Action for resolution phase:** add to Open Questions as inception-phase-explicit constraint.

### F4 — BLAKE3 collision-probability discussion in AP4

AP4 (hash-mismatch tolerated silently) covers the case where a stored blob's content has been corrupted (hash recomputation diverges from stored hash). It does NOT explicitly cover the case where an attempted duplicate write produces a hash collision — same hash, different content. BLAKE3's 256-bit output makes collision probability structurally negligible at any scale (~2^-128 for birthday bound); a collision in practice indicates BLAKE3 is broken or the canonical-serialization contract is violated. **Action for resolution phase:** AP4 expanded to address both cases (corruption detection AND collision detection); idempotent-commit check (Proposal item 2) must verify byte-equality on apparent duplicate writes, not just hash-equality.

### F5 — VACUUM operational discussion

SQLite's `VACUUM` rebuilds the file; preserves logical content; changes on-disk byte representation. Not a §2.1 violation per Phase 1 D1 analysis. Operational concern: VACUUM during ingestion may affect performance + crash semantics. **Action for resolution phase:** Open Question added — VACUUM cadence + operational guidelines; recommendation likely "schedule during ingestion-idle periods" deferred to operational ops document.

### F6 — Backup ordering (blob-store first, then SQLite)

SQLite `.backup` produces a point-in-time consistent snapshot; rsync against the blob-store directory is NOT atomic with respect to the SQLite snapshot. Backup ordering matters: if SQLite is backed up first, the snapshot can reference blobs not yet copied (recovery would fail on missing blob). If blob-store is backed up first, the snapshot may include blobs without index rows (harmless: orphan blobs at backup time become referenced rows at recovery). **Action for resolution phase:** Proposal item 5 (backup/recovery commitment) updated to specify ordering — blob-store first, then SQLite. Recovery verifies all `payload_ref` resolve to blobs; orphan blobs surface but do not block recovery.

## Phase 5 — Recommendation with reversal conditions

**Recommendation (for resolution-phase consideration):** accept the draft's selection of **SQLite + content-addressed blob-store on local filesystem** as the inception-phase substrate, with the following modifications enacted in the resolution-phase commit:

1. **Rewrite FC4/FC5 per Phase 3.1** — operationalize "no operational dependencies" and "bounded" comparative modifiers.
2. **Expand Proposal item 1 per F1** — commit to `PRAGMA synchronous=FULL` + `PRAGMA journal_mode=WAL`; CI integration test verifies.
3. **Add POSIX-only constraint to Open Questions per F2** — inception-phase substrate is POSIX-only; Windows requires re-evaluation.
4. **Add single-writer constraint to Open Questions per F3** — explicit at inception; reversal target for R-store-3.
5. **Expand AP4 per F4** — collision-detection + corruption-detection both addressed; idempotent-commit (Proposal item 2) verifies byte-equality on apparent duplicate writes.
6. **Add VACUUM operational discussion to Open Questions per F5** — cadence + operational guidelines deferred to ops document.
7. **Update Proposal item 5 per F6** — backup ordering: blob-store first, then SQLite; recovery surfaces orphan blobs but does not block.
8. **Split AP4 per ES4** — separate §2.1-invariant-restatement (mutation-detected-is-violation) from operational-discipline (read-path-must-verify-hash).

**Reversal conditions** (when to revisit via subsequent RFC):

- **R-store-1.** Throughput / multi-writer pressure characterized: single-writer SQLite cannot keep up with ingestion rate, OR multi-process / multi-host ingestion required for reliability. Threshold: ingestion rate sustained >10K events/sec at characterized load OR explicit RFC requiring multi-writer topology. (Revisit NATS JetStream or Kafka as Tier 0/1 streaming substrate; the within-streaming choice is itself a follow-on RFC.)
- **R-store-2.** Cold-archive volume justifies dedicated columnar tier. Threshold: SQLite + blob-store cumulative size >100GB OR sustained query-latency degradation on analytical queries (full-substrate replay). (Revisit Parquet on S3/MinIO as Tier 2 archive; SQLite + blob-store remain Tier 1.)
- **R-store-3.** Multi-service ingestion topology requires shared database. Threshold: explicit RFC introducing second service that ingests into the same substrate. (Revisit PostgreSQL — server-process operational dependency justified by multi-service sharing requirement.)
- **R-store-4 (new — surfaced via F2).** Filesystem-portability requirement: Windows or other non-POSIX substrate host required. Threshold: explicit RFC requiring Windows substrate host. (Revisit blob-in-SQLite or a different blob-store backend with portable atomicity.)

No reversal condition fires today.

## Phase 6 — Carry-forwards

- **This is the third and final inception-phase technology RFC per [`§0022`](../../charter/decision-log.md).** Nothing inherits beyond this. The implementation-gate per [CLAUDE.md §6.4](../../../.claude/CLAUDE.md) is fully cleared on acceptance: all three technology selections made under ordinary RFC discipline; no procedural-divergence carry-forward per [`§0026`](../../charter/decision-log.md).
- **Service skeleton work proceeds under ordinary RFC/PR discipline** after acceptance. The first concrete commit producing executable code (e.g. an ingestion service skeleton) becomes possible.
- Cross-RFC discipline obligation: [`§0025`](../../charter/decision-log.md) AP1 (cross-category type imports without explicit boundary function) lint rule applies to the storage-decode path (Phase 1 D2 finding); explicit in service code review.
- No new canonical project vocabulary introduced. Glossary unchanged.
- No Ontology document revision required.
- No Charter amendment required.
- The five frozen object-level invariants (§2.1, §2.2, §2.3, §2.5 — and §2.4 / §2.6 pending in empirical-pressure phase) are all respected; selection enacts §0003's "technology selection is subordinate to constitutional properties" principle empirically a third time (after [`§0024`](../../charter/decision-log.md) + [`§0025`](../../charter/decision-log.md)).
