# RFC — Architecture: Inception-phase storage technology selection

- **Status:** draft
- **Authors:** committee
- **Date:** 2026-05-19
- **Type:** architecture
- **Affects:** [`services/`](../../../services/) (inception-phase substrate technology for ingestion service + downstream consumers); [`docs/architecture/`](../../architecture/) (future architecture documents inherit the substrate choice + the canonical-serialization contract referenced in [`§0024`](../../charter/decision-log.md) AP5); [`docs/charter/decision-log.md` §0003](../../charter/decision-log.md) (storage-technology portion of the deferral; predicate satisfied per [`§0022`](../../charter/decision-log.md)); [`docs/charter/decision-log.md` §0022](../../charter/decision-log.md) (authorizing pivot entry); [`docs/charter/decision-log.md` §0024](../../charter/decision-log.md) (schemas-technology — Protobuf bytes are the payload format the substrate stores); [`docs/charter/decision-log.md` §0025](../../charter/decision-log.md) (implementation-language — Go's database-driver ecosystem inheritance per F6); [Charter §2.1](../../charter/constitutional-charter.md#21-observational-integrity), [§2.2](../../charter/constitutional-charter.md#22-epistemic-separation), [§2.3](../../charter/constitutional-charter.md#23-provenance-integrity), [§2.5](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) (frozen — selection respects, does not modify).

> RFCs in Ghost Trace are proposals subject to constitutional review. The [Constitutional Charter](../../charter/constitutional-charter.md) is the authoritative reference. RFCs that conflict with the Charter must either be rejected or accompanied by a corresponding charter amendment (see [`../../charter/amendments.md`](../../charter/amendments.md)).

---

## Summary

**Draft position (to be tested in discussion phase, not accepted by inclusion here):** select **SQLite as the inception-phase primary event log with a content-addressed blob-store on the local filesystem** as the storage technology for Ghost Trace's substrate. SQLite supplies transactional commit guarantees + single-file substrate + no operational dependencies; the content-addressed blob-store supplies §2.1 immutability via BLAKE3 hash of canonical Protobuf serialization per [`§0024`](../../charter/decision-log.md) AP5. Application-level append-only discipline (`INSERT`-only on event rows; no `UPDATE` or `DELETE` permitted) enforces §2.1's write-once semantics.

Inception-phase selection only. Production-scale streaming substrate (NATS JetStream, Kafka) and cold-archive object stores (Parquet on S3/MinIO) are deferred to follow-on RFCs under empirical load characterization. Third of three technology RFCs authorized by [`§0022`](../../charter/decision-log.md); discharges the storage-technology portion of [`§0003`](../../charter/decision-log.md)'s deferral.

## Motivation

The substrate selection is gating for the ingestion service work under [`§0022`](../../charter/decision-log.md). The four frozen object-level invariants impose distinct storage constraints, and the [`§0024`](../../charter/decision-log.md) schemas-technology + [`§0025`](../../charter/decision-log.md) language-technology selections inherit downstream:

- **[§2.1 Observational Integrity](../../charter/constitutional-charter.md#21-observational-integrity) (frozen)** requires append-only by construction at the storage layer with content-addressable identifiers. SQLite supports application-level append-only discipline (`INSERT`-only event-row policy enforced by CI lint + runtime check + table-level write privilege boundary); content-addressing via BLAKE3 hash of canonical Protobuf bytes (per [`§0024`](../../charter/decision-log.md) AP5) supplies the mutation-detection mechanism. The blob-store on local filesystem stores each Protobuf payload as a file named by its content hash; SQLite holds the index (event-row fields + reference to blob).

- **[§2.2 Epistemic Separation](../../charter/constitutional-charter.md#22-epistemic-separation) (frozen)** is preserved at the storage layer by Charter §2.2 Boundary Conditions ("The three categories share infrastructure — storage, transport, indexing, observability tooling — but not type"). Shared storage infrastructure is permitted; the categorical separation is preserved by the Protobuf message types embedded in the payload bytes, not by a discriminator column at the storage layer. A single `events` table with a `message_type` column is structurally distinct from a unified-record-with-discriminator at the schemas layer — the column is an index field for query, not a categorical discriminator at the type layer.

- **[§2.3 Provenance Integrity](../../charter/constitutional-charter.md#23-provenance-integrity) (frozen v0.4)** requires that the typed `subject_ref_*` references per [`§0016`](../../charter/decision-log.md) Q3 resolution are reconstructible from substrate. The references are part of the Protobuf payload (oneOf-encoded per [`§0024`](../../charter/decision-log.md) Phase 1 Dimension 3); substrate stores the payload bytes; reconstruction reads + decodes. No storage-technology-specific concern.

- **[§2.5 Hypothesis Lifecycle Explicitness](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) (frozen v0.3)** requires lifecycle events committed immutably as Cat I records. Same storage requirement as §2.1 (append-only events table); no §2.5-specific concern.

- **[§2.4 + §2.6 (pending — empirical pressure phase)](../../charter/constitutional-charter.md#24-inferential-influence-disclosure)**: future `influenced_by` edges (§2.4) and confidence + independence pairing (§2.6) are additional record types that go in the same events table when their invariants redact.

The cost of not making this selection (technology dimension): no ingestion service can persist events; service work halts. The procedural cost (Charter-governance dimension): the empirical-evidence-from-implementation feedback loop that §2.4 + §2.6 redaction depends on per [`§0022`](../../charter/decision-log.md) cannot begin (this is the third and final inception-phase technology gate; all three must clear before service skeleton work proceeds).

## Constitutional Review

Verbatim output of the rfc-author §1 pre-authorship analysis (Q1–Q6).

### Q1 — Charter invariants touched

- **§2.1 Observational Integrity (frozen):** substrate must enforce write-once semantics. SQLite supports via application-level append-only discipline (`INSERT`-only on event rows; CI lint + runtime check + table-level write privilege boundary). Content-addressing via BLAKE3 hash supplies mutation-detection. Selection respects.
- **§2.2 Epistemic Separation (frozen):** shared storage infrastructure is permitted per §2.2 Boundary Conditions; categorical separation preserved by Protobuf message types in payload bytes. Single `events` table with `message_type` index column is structurally distinct from unified-record-with-discriminator at the schemas layer. Selection respects with discipline obligation surfaced in Anti-Patterns AP1.
- **§2.3 Provenance Integrity (frozen v0.4):** typed `subject_ref_*` references stored as Protobuf payload bytes; reconstructible via decode. No storage-technology-specific concern.
- **§2.5 Hypothesis Lifecycle Explicitness (frozen v0.3):** lifecycle events as additional record types in the events table; same §2.1 append-only requirement.
- **§2.4 + §2.6 (pending — empirical pressure phase):** future record types accommodated by the events table without table-structure changes (Protobuf payload + message_type column).

### Q2 — Glossary redefinition

No. SQLite, BLAKE3, content-addressed blob-store — technology vocabulary, not canonical project vocabulary. The canonical project terms `primary event log`, `substrate`, `archive` (per [`CLAUDE.md` §3](../../../.claude/CLAUDE.md) + vocabulary-discipline §4) are unchanged; this RFC's `events` table IS the inception-phase primary event log.

### Q3 — Implicit resolution of open Ontology questions

None. Storage technology does not touch the remaining open ontology questions ([`ontology.md`](../../ontology/ontology.md) Q3 independence formal definition, Q5 "transitive?" half; [`provenance-model.md`](../../ontology/provenance-model.md) OMQ #1 Granularity, OMQ #4 Cross-domain).

### Q4 — Charter amendment required

No. The pivot at [`§0022`](../../charter/decision-log.md) authorized this RFC under ordinary discipline.

### Q5 — New invariant introduced

No. Technology selection. Application-level append-only discipline is a *property of the implementation* that downstream-enforces existing §2.1 (substrate immutability), not a new invariant. Content-addressing via BLAKE3 is a *mechanism* for §2.1's mutation-detection, not a new requirement.

### Q6 — Ceremony without behavioral consequence

No. Selection is gating for all service work. Falsifiable by deletion: without it, no event can be persisted; ingestion service halts at first commit attempt.

## Proposal

**Draft position (to be tested in discussion phase):** adopt **SQLite + content-addressed blob-store on local filesystem** for the inception-phase substrate. Concrete commitments:

1. **SQLite as the inception-phase primary event log.** Single-file SQLite database at a configurable path. WAL mode enabled for concurrent-reader + single-writer semantics. Schema: single `events` table with columns `(event_hash BLOB PRIMARY KEY, event_time INTEGER NOT NULL, message_type TEXT NOT NULL, payload_ref TEXT NOT NULL, committed_at INTEGER NOT NULL)`. The `event_hash` column is the BLAKE3 hash of the canonical Protobuf serialization. The `payload_ref` column is the relative path into the blob-store directory.
2. **Content-addressed blob-store on local filesystem.** Directory tree under a configurable path. Each Protobuf payload stored as a single file named by its BLAKE3 hash (sharded by hash prefix to avoid filesystem-directory-entry limits — e.g. `blobs/ab/cd/abcd...`). Write semantics: write-once-by-content-hash; an attempted write of a hash that already exists is a no-op (idempotent commit).
3. **Application-level append-only discipline.** Service code uses an `INSERT INTO events ...` path only; no `UPDATE` or `DELETE` against the events table is permitted in service code. Enforcement: (a) Go-level repository pattern exposes only an `Append` method, no `Update` / `Delete`; (b) CI lint rule rejects `UPDATE events` / `DELETE FROM events` in service code; (c) SQLite trigger optionally rejects `UPDATE`/`DELETE` against events at the database layer (defense in depth). Same discipline applies to the blob-store: write-once-by-content-hash.
4. **Go driver: `modernc.org/sqlite` (pure-Go).** Avoids CGo dependency; preserves the [`§0025`](../../charter/decision-log.md) static-binary commitment (a CGo SQLite driver such as `mattn/go-sqlite3` requires a C toolchain and breaks the static-binary property). Driver-version pinned per [`§0025`](../../charter/decision-log.md) library-version-pinning discipline.
5. **Backup / recovery commitment.** SQLite file backed up via the `.backup` command (online consistent snapshot); blob-store directory backed up via `rsync`-equivalent with content-hash verification on restore. Recovery: replay events table in commit order; verify each `payload_ref` resolves to a blob whose content-hash matches the row's `event_hash`. Hash mismatch on replay is a §2.1 violation.
6. **Substrate-technology reversal procedure (commitment, not enactment).** Migration to a streaming substrate (NATS, Kafka) or columnar archive (Parquet on S3) involves re-reading the events table in commit order and re-writing into the target substrate. The content-addressing discipline transfers (BLAKE3 hashes are stable across substrates); the SQL-vs-streaming-API transport layer changes. No re-serialization of payload bytes required (Protobuf bytes are substrate-independent per [`§0024`](../../charter/decision-log.md) selection). Reversal cost is bounded.

## Alternatives Considered

Five alternatives evaluated. Three rejected as admissible-but-deferred (NATS JetStream, Kafka, Parquet on S3/MinIO); one rejected on reimplementation-cost grounds (plain files + custom index); one rejected as scope-mismatch (PostgreSQL).

- **NATS JetStream.** Rejected as admissible-but-deferred. JetStream is a purpose-built event-stream substrate with native append-only semantics + replication + retention policies. Mature Go driver (`nats-io/nats.go` + JetStream extensions). The rejection is operational: JetStream requires a running NATS server process, configuration management, network operations — operational dependencies inappropriate for inception phase before service topology is empirically characterized. Reversal condition R-store-1 (throughput / multi-writer pressure characterized; revisit JetStream).

- **Apache Kafka.** Rejected as admissible-but-deferred. Kafka is the canonical event-streaming substrate at scale; mature Go driver (`segmentio/kafka-go`, `IBM/sarama`). The rejection is operational: Kafka's deployment footprint (broker + ZooKeeper / KRaft + topic configuration + consumer-group management) is heavier than JetStream's and substantially heavier than SQLite's; inappropriate for inception phase. Reversal condition R-store-1 (revisit Kafka alongside JetStream if throughput/multi-writer pressure compels streaming substrate).

- **Parquet on S3 / MinIO.** Rejected as admissible-but-deferred for a different reason than NATS/Kafka: scope-mismatch. Parquet is a columnar format optimized for analytical queries over append-only datasets; S3/MinIO supplies object storage. This stack is well-suited for *cold archive* of historical events (the Tier 1 archive per [`storage-model.md`](../../architecture/storage-model.md)), NOT for the hot ingestion path. Inception phase has no cold archive yet. Reversal condition R-store-2 (cold-archive volume justifies dedicated columnar archive tier; revisit Parquet/S3 as Tier 1 archive).

- **Plain files + custom index.** Rejected on reimplementation-cost grounds. SQLite provides transactional commit + crash-recovery + indexing + WAL semantics for free; implementing these against plain files would reimplement what SQLite gives free, against substantial reliability risk (concurrent-writer durability bugs are subtle). No revisit condition; the rejection is structural-and-permanent at this scale.

- **PostgreSQL.** Rejected as scope-mismatch for inception. PostgreSQL is a robust server-tier relational database with strong append-only-enforcement primitives (table-level revoke of UPDATE/DELETE; row-level security; logical replication). However, PostgreSQL requires a running server process — the same operational-dependency concern as NATS/Kafka but in a different category. SQLite's single-file substrate has no operational dependencies. Reversal condition R-store-3 (multi-service ingestion topology requires shared database; revisit PostgreSQL).

The admissible-but-deferred registrations (NATS JetStream, Kafka, Parquet on S3/MinIO, PostgreSQL — four entries) extend the established pattern (seventh through tenth instances after the six from [`§0020`](../../charter/decision-log.md) / [`§0021`](../../charter/decision-log.md) / [`§0024`](../../charter/decision-log.md) / [`§0025`](../../charter/decision-log.md)).

## Open Questions

The RFC explicitly defers:

- **Specific SQLite version pin.** Latest stable at acceptance; exact version pinned per [`§0025`](../../charter/decision-log.md) library-version-pinning discipline.
- **Blob-store sharding scheme detail.** Two-level prefix sharding (`ab/cd/abcd...`) suggested; exact prefix length + sharding configuration deferred to follow-on operational document.
- **Backup scheduling cadence + retention.** Operational concern; deferred to follow-on operations RFC when service-tier ops are characterized.
- **SQLite trigger as defense-in-depth.** Proposal item 3 (c) names this as optional. The cost-benefit analysis (trigger overhead vs additional `UPDATE`/`DELETE` rejection layer) deferred to discussion phase.
- **Blob-in-SQLite vs blob-on-filesystem hybrid choice.** This RFC commits to hybrid (SQLite for index, filesystem for blobs). The alternative is blob-in-SQLite (BLOB column in events table; single file holds everything). Tradeoffs surfaced in discussion phase: hybrid simpler to back up incrementally (rsync); blob-in-SQLite simpler operationally (one file). Re-evaluation possible without changing substrate-technology classification.
- **Cross-RFC coupling with future architecture RFCs.** The replay-model architecture document referenced in [`replay-model.md`](../../architecture/replay-model.md) inherits this substrate selection — replay semantics must be expressible against SQLite + blob-store reads. No conflict anticipated; surfacing for completeness.

## Anti-Patterns to Avoid

By analogy to Charter [§2.1](../../charter/constitutional-charter.md#21-observational-integrity) and [§2.2](../../charter/constitutional-charter.md#22-epistemic-separation) `Forbidden Anti-Patterns` sections. Each is concrete and falsifiable.

- **`UPDATE` or `DELETE` against the events table.** §2.1 substrate-immutability prohibits mutation of committed event records. Application-level discipline (Proposal item 3) forbids these SQL operations in service code; CI lint enforces; optional SQLite trigger enforces at database layer. Detectable: grep service code for `UPDATE events`, `DELETE FROM events`, `events SET`; CI gate.

- **Using SQLite ROWID as event identity instead of `event_hash`.** SQLite assigns an integer ROWID to every row. Treating ROWID as the canonical event identity (rather than the BLAKE3 content hash) breaks content-addressing: the same Protobuf payload at two different commit times would have different ROWIDs but the same content-hash. The events table is defined with `event_hash BLOB PRIMARY KEY` to suppress the implicit ROWID; service code must reference events by `event_hash`, not by `rowid`. Detectable: grep service code for `rowid` references in events-table queries; reject in CI.

- **Blob mutation via filesystem write-through-existing-path.** §2.1 substrate-immutability prohibits mutation of committed blobs. Application-level discipline: blob writes use a write-temp-then-rename pattern (atomic on POSIX); a rename targeting an existing path is rejected via `O_EXCL`-equivalent. Detectable: grep service code for blob-write paths that do not check for existence first; runtime check.

- **Hash-mismatch on read tolerated silently.** §2.1 requires that mutation detected at read time is a violation. Service code reading a blob MUST recompute the BLAKE3 hash and compare against the SQLite row's `event_hash`; mismatch raises an immediate error and is logged for incident response. Detectable: code review checks every blob-read path includes hash verification; integration test deliberately corrupts a blob and verifies the mismatch surfaces.

- **Backup format coupled to inception-phase substrate.** Backups should serialize event records in a substrate-independent form (one Protobuf payload per file, plus an index of event-hash → ordering). Backups in SQLite native format couple recovery to the current substrate selection; substrate-technology reversal then becomes a recovery problem. Detectable: backup-format spec required as a separate document; backups not adhering to spec rejected.

## Migration and Backward Compatibility

**Inception phase.** No prior substrate state exists. Forward-looking decision. The ingestion service skeleton (gated on this RFC's acceptance) will be the first writer to the SQLite + blob-store substrate.

**Storage-technology reversal cost (forward-looking).** Per Proposal item 6: reversal to streaming substrate (NATS, Kafka) or columnar archive (Parquet on S3) involves re-reading the events table in commit order and re-writing into the target substrate. The content-addressing discipline transfers (BLAKE3 hashes are stable across substrates); Protobuf payload bytes are substrate-independent per [`§0024`](../../charter/decision-log.md). No re-serialization required. Reversal cost is bounded by re-read + re-write throughput against the target substrate; it does NOT require historical-byte re-derivation. This is structurally different from [`§0024`](../../charter/decision-log.md) F6 (schemas-technology reversal requires re-serialization of all historical Cat I records — load-bearing operation). Storage-technology reversal is read-then-write; schemas-technology reversal is re-derive-then-write.

**Substrate-tier-introduction (forward-looking).** Adding a streaming substrate (Tier 0 hot path per [`storage-model.md`](../../architecture/storage-model.md)) in front of SQLite (Tier 1 commit) preserves the SQLite substrate as the canonical primary event log; the streaming substrate becomes a write-through cache or a producer-consumer boundary. Adding a cold-archive tier (Parquet on S3 per [`storage-model.md`](../../architecture/storage-model.md) Tier 2) preserves SQLite as Tier 1 and adds Tier 2 as a downstream sink. Neither is reversal — both are additive.

## References

- [`docs/charter/constitutional-charter.md`](../../charter/constitutional-charter.md) §2.1, §2.2, §2.3 (frozen v0.4), §2.5 (frozen v0.3).
- [`docs/charter/decision-log.md`](../../charter/decision-log.md) [`§0003`](../../charter/decision-log.md) (storage-technology portion of the deferral; satisfied by this RFC's acceptance), [`§0022`](../../charter/decision-log.md) (pivot authorization), [`§0024`](../../charter/decision-log.md) (schemas-technology — Protobuf payload format; AP5 inherited; F6 reversal-cost discussion), [`§0025`](../../charter/decision-log.md) (implementation-language — Go database-driver ecosystem F6 inherited; static-binary commitment constrains driver choice to pure-Go), [`§0026`](../../charter/decision-log.md) (RFC procedural-divergence resolved; this RFC opens on clean procedural ground).
- [`docs/rfcs/draft/architecture-schemas-technology-selection.md`](./architecture-schemas-technology-selection.md) (accepted at [`§0024`](../../charter/decision-log.md)).
- [`docs/rfcs/draft/architecture-implementation-language-selection.md`](./architecture-implementation-language-selection.md) (accepted at [`§0025`](../../charter/decision-log.md)).
- [`docs/architecture/storage-model.md`](../../architecture/storage-model.md) (Tiers 0–2 architecture; this RFC selects the inception-phase Tier 1 substrate).
- [`.claude/CLAUDE.md`](../../../.claude/CLAUDE.md) §6.4 (implementation gate, cleared at [`§0022`](../../charter/decision-log.md)).

## Decision Record

Empty. Populated at status `accepted` after discussion phase completes and the resolution-phase commit records the final commitment. Per [`§0024`](../../charter/decision-log.md) / [`§0025`](../../charter/decision-log.md) pattern (status `discussion` → `accepted`; file remains in `docs/rfcs/draft/`; no renumbering applied per [`§0026`](../../charter/decision-log.md) codified convention); discussion phase is NOT bypassed.
