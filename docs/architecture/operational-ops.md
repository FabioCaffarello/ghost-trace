# Operational Ops

**Status:** Active. Third non-scaffold architecture document. Discharges the follow-on commitment named in [`decision-log §0027`](../charter/decision-log.md) Open Questions ("backup scheduling cadence + retention"; "VACUUM operational cadence") and [`§0028`](../charter/decision-log.md) Consequences.

> This document specifies the operational discipline for running an inception-phase Ghost Trace ingestion service against the SQLite + content-addressed blob-store substrate per [`§0027`](../charter/decision-log.md). Three operations are load-bearing: backup, VACUUM, and disk-capacity monitoring. A fourth (restoration) is the procedure that backup + monitoring exist to support. Substrate migration is sketched for forward-look; concrete procedures land when [`§0027`](../charter/decision-log.md) reversal conditions fire.

## Constitutional Anchors

- [Charter §2.1 Observational Integrity](../charter/constitutional-charter.md#21-observational-integrity) — the operations specified here MUST preserve substrate-immutability under operational pressure. Backup, VACUUM, and disk-capacity responses are all evaluated against this invariant.
- [`decision-log §0027`](../charter/decision-log.md) — storage-technology selection (SQLite + blob-store); Proposal item 5 (backup/recovery commitment with blob-store-first ordering); Open Questions (backup cadence + VACUUM cadence — discharged here); AP4/AP5/AP6 (substrate-integrity anti-patterns).
- [`decision-log §0028`](../charter/decision-log.md) — canonical-serialization-contract (content-addressing makes blob-store backups verifiable on restore).
- [`decision-log §0029`](../charter/decision-log.md) — concurrency-pattern (substrate-writer serialization + error-propagation — operational responses inherit the discipline).
- [`decision-log §0032`](../charter/decision-log.md) — unrecoverable-error shutdown escalation (operations that detect §2.1 violations trigger the same escalation pathway).

## Subordination

This document is subordinate to the [Constitutional Charter](../charter/constitutional-charter.md) and the [Ontology](../ontology/ontology.md). A conflict with either resolves by revising this document.

## Scope

This document specifies:

1. **Backup procedure** — ordering, cadence, retention, hash-verification on restore.
2. **VACUUM cadence** — when to run; how to schedule; impact on substrate-immutability (none, per [`§0027`](../charter/decision-log.md) D1 analysis).
3. **Disk-capacity monitoring** — what fills first (SQLite vs blob-store); thresholds; operator notification.
4. **Restoration procedure** — step-by-step recovery from a backup pair.
5. **Substrate migration sketch** — forward-look for when [`§0027`](../charter/decision-log.md) R-store-1/2/3/4 reversal conditions fire.
6. **Minimum observability** — what to surface to operators via structured-output.

It does NOT specify:

- Automation. The procedures here are manual + scriptable; automation tooling (cron, systemd timers, etcd-coordinated leader election for backup workers) is a separate concern, deferred to operational-ops-automation work when single-process inception phase reaches its operational limit.
- Service-tier IPC operations (HTTP/gRPC health checks, readiness probes). Deferred to the service-tier IPC RFC ([`§0025`](../charter/decision-log.md) Open Questions).
- Multi-service coordination. Inception phase is single-service; multi-service operations land when [`§0027`](../charter/decision-log.md) R-store-3 fires (multi-service ingestion sharing).
- Specific monitoring infrastructure choice (Prometheus, Grafana, OpenTelemetry, etc.). The document specifies WHAT to surface; the HOW is operator preference for inception phase.

## Backup Procedure

### Ordering: blob-store first, then SQLite

Per [`§0027`](../charter/decision-log.md) Proposal item 5 (load-bearing): the blob-store directory is backed up FIRST, then the SQLite database. Rationale: SQLite `.backup` produces a point-in-time consistent snapshot; the blob-store rsync is NOT atomic with the SQLite snapshot. Backing up SQLite first would produce a snapshot referencing blobs not yet copied (recovery would fail on missing-blob errors). Backing up blob-store first produces snapshots that may include orphan blobs (blobs without index rows at backup time) — harmless at recovery, where the events table is the authoritative reference for what exists.

### Procedure (manual; scriptable)

```sh
# 1. Capture blob-store first. rsync preserves content-hash-named files;
#    no content interpretation needed.
BLOB_SRC=./blobs
BLOB_DST=./backup/blobs-$(date -u +%Y%m%dT%H%M%SZ)
mkdir -p "$BLOB_DST"
rsync -a --checksum "$BLOB_SRC/" "$BLOB_DST/"

# 2. Capture SQLite second. The .backup command produces a consistent
#    snapshot even under concurrent writes (WAL mode permits this).
DB_SRC=./ghost-trace.db
DB_DST=./backup/ghost-trace-$(date -u +%Y%m%dT%H%M%SZ).db
sqlite3 "$DB_SRC" ".backup '$DB_DST'"

# 3. Record the backup pair atomically (manifest file naming the two
#    sibling directories/files). The manifest is the operator's
#    consent that the pair is intended to be restored together.
echo "$BLOB_DST $DB_DST" > ./backup/manifest-$(date -u +%Y%m%dT%H%M%SZ).txt
```

The script captures absolute UTC timestamps in filenames (no local-time ambiguity) and writes a manifest file recording the pair. The manifest is the unit of restoration — never restore a SQLite snapshot against a blob-store snapshot from a different timestamp.

### Cadence

Inception-phase guidance:

- **Hot-path backups** — every 60 minutes during ingestion activity. Justified by the recovery-window-tolerance expectation: 60 minutes of replay-from-source rebuild is operationally acceptable at inception scale. Adjust upward (less frequent) if backup-windowing pressure surfaces; downward (more frequent) if recovery-window-tolerance tightens.
- **Idle-path backups** — every 24 hours during ingestion-idle periods. Captures the substrate state-at-rest with VACUUM coordination (see next section).

These cadences are inception-phase defaults; operators MAY adjust based on observed ingestion rate + recovery-window-tolerance. Adjustment SHOULD be recorded in a service-tier ops journal or equivalent.

### Retention

Inception-phase guidance:

- **Last 24 hot-path backups** (covering ~24 hours of recovery granularity at 60-minute cadence).
- **Last 30 idle-path backups** (covering ~30 days of point-in-time recovery).
- **Quarterly retention** — one idle-path backup per quarter retained indefinitely for forensic replay (per [`replay-model.md`](./replay-model.md) Phase 4 retrospective analytical replay obligation).

Retention is OPERATOR DISCRETION beyond these defaults; the constraint is that no retention policy MAY delete a backup that is the most recent confirmed-good restorable snapshot.

### Integrity verification on restore

Per [`§0027`](../charter/decision-log.md) Proposal item 5: recovery replays the events table in commit order; verifies each `payload_ref` resolves to a blob whose content-hash matches the row's `event_hash`. Hash mismatch on replay is a §2.1 violation per [`§0027`](../charter/decision-log.md) AP4/AP5; the recovery procedure MUST abort + escalate (do not proceed with a partially-verified substrate).

Verification is mechanical and MUST be run on every restore — not as a post-recovery validation, but as the recovery procedure's gate. See §Restoration Procedure below.

## VACUUM Cadence

### What VACUUM does

SQLite's `VACUUM` rebuilds the database file: defragments space, reclaims unused pages, applies any pending table-structure changes. The operation produces a structurally-equivalent database with potentially different on-disk byte representation. Per [`§0027`](../charter/decision-log.md) Phase 1 D1 analysis, VACUUM is NOT a §2.1 violation: content-hashes are computed over Protobuf payload bytes (blob-store content), not over SQLite file bytes. VACUUM is a storage-layer reorganization that preserves the substrate's logical commitments.

### When to run

- **Trigger condition** — disk-space utilization on the SQLite database file exceeds 70% of allocated capacity OR fragmentation (measured via `PRAGMA freelist_count` exceeding 10% of `PRAGMA page_count`) becomes significant.
- **Timing** — schedule during ingestion-idle windows. Concurrent ingestion + VACUUM is mechanically permitted (WAL mode allows it) but produces operational friction: VACUUM acquires an exclusive lock briefly at the end (the swap step), which blocks the single-writer Append path. For inception-phase services with infrequent idle windows, the trigger condition may force an under-load VACUUM — operators accept the brief Append-blocking and observe.
- **Coordination with backup** — run VACUUM BEFORE the idle-path backup, not after. Backups of a VACUUM-fresh database are smaller and have fewer freelist pages. Operational sequence: detect trigger → schedule next idle-path backup window → run VACUUM → run backup → record both events in the ops journal.

### Procedure

```sh
sqlite3 ./ghost-trace.db "VACUUM"
```

That's the entire operation. The service does not need to be stopped (WAL mode permits concurrent reads + the single ingestion writer); the exclusive lock at the swap step is brief.

### Constraints

- VACUUM MUST NOT be run inside a transaction. The SQLite driver rejects this; the `sqlite3` CLI handles it correctly.
- VACUUM resets the `auto-vacuum` mode if previously set; the substrate does NOT use auto-vacuum (the default `NONE` mode is preserved per [`§0027`](../charter/decision-log.md) Proposal item 1 — explicit PRAGMA configuration only sets `journal_mode=WAL` and `synchronous=FULL`, leaving auto-vacuum at default).

## Disk-Capacity Monitoring

### What fills first

- **Blob-store** typically fills first under sustained ingestion. Each event contributes one blob whose size equals the canonical Protobuf payload size. For inception-phase `DeclaredSession` workloads (~60 bytes per event per the [`canonical-corpus`](../../services/ingestion/internal/canonical/testdata/canonical-corpus/declared-session-typical.bin) typical entry), the blob-store grows roughly linearly with event count.
- **SQLite events table** grows more slowly (the index rows are small: ~70 bytes per event including the 32-byte hash + small integer/string columns). At inception scale, SQLite represents <50% of total substrate footprint.
- **SQLite WAL file** (`ghost-trace.db-wal`) can grow transiently between checkpoints. Under `journal_mode=WAL` + `synchronous=FULL`, the WAL is checkpointed automatically; manual `PRAGMA wal_checkpoint(TRUNCATE)` is available if the WAL grows beyond expected bounds.

### Thresholds

- **70% utilization** — schedule VACUUM (if SQLite-side); plan blob-store expansion or archival to Tier 2 (if blob-store-side per [`storage-model.md`](./storage-model.md) tiering); surface an operator notification.
- **85% utilization** — operator intervention required within hours. Either expand storage capacity OR begin substrate-migration evaluation (see §Substrate Migration below).
- **95% utilization** — emergency operator intervention. Continued ingestion at this threshold risks Append failure (SQLite returns `SQLITE_FULL`); the substrate-writer-serialization mutex prevents corruption-by-partial-write but Append rejection cascades to upstream producers.

### Operator notification mechanism

Inception phase: structured-output entries on stderr at the thresholds above. The operator runs ingestion as a foreground or systemd-supervised process; stderr is monitored. A future operational-ops-automation work cycle may add file-watcher or push-based notification.

Notification structure (informational; operator-tunable):

```json
{
  "level": "warning",
  "subsystem": "disk-capacity",
  "utilization_pct": 73,
  "subsystem_path": "/path/to/blobs",
  "threshold": 70,
  "recommendation": "schedule expansion or Tier-2 archival per docs/architecture/operational-ops.md §Disk-Capacity Monitoring"
}
```

## Restoration Procedure

Recovery from a backup pair. The procedure is gated on content-hash verification per [`§0027`](../charter/decision-log.md) Proposal item 5; failure aborts before any substrate state is published.

```sh
# 1. Locate the backup manifest naming the pair to restore.
MANIFEST=./backup/manifest-20260520T120000Z.txt
BLOB_BACKUP=$(awk '{print $1}' "$MANIFEST")
DB_BACKUP=$(awk '{print $2}' "$MANIFEST")

# 2. Stage the pair at restoration paths (NOT the live substrate yet).
STAGE_BLOBS=./restore/blobs
STAGE_DB=./restore/ghost-trace.db
mkdir -p ./restore
rsync -a --checksum "$BLOB_BACKUP/" "$STAGE_BLOBS/"
cp "$DB_BACKUP" "$STAGE_DB"

# 3. Run hash-verification: every event_hash in the SQLite snapshot
#    MUST resolve to a blob whose recomputed BLAKE3 hash matches.
#    A dedicated `verify` CLI tool lands as follow-on work; for inception
#    phase the verification is exercised via the substrate's existing
#    ReadBlob path (which performs hash recomputation on every read per
#    docs/architecture/canonical-serialization-contract.md AP "hash-
#    verification omitted from blob-read path").
#
# Until the dedicated verify tool exists, the operational verification
# is: launch the ingestion service against the restored substrate; the
# first read of any event surfaces ErrHashMismatch on §2.1 violation
# and triggers the unrecoverable-error shutdown per §0032.
# This is a coarse but mechanically-correct verification: a hash-mismatch
# substrate cannot be silently restored — the next read fails fast.

# 4. If verification passes, swap the staged substrate into the live
#    paths. The service must be stopped during the swap.
mv ./ghost-trace.db ./ghost-trace.db.pre-restore-$(date -u +%Y%m%dT%H%M%SZ)
mv ./blobs ./blobs.pre-restore-$(date -u +%Y%m%dT%H%M%SZ)
mv "$STAGE_DB" ./ghost-trace.db
mv "$STAGE_BLOBS" ./blobs
```

The pre-restore-suffixed paths are NOT removed automatically — they remain as forensic artifacts until the operator confirms the restore is good. Operator-driven cleanup of the pre-restore artifacts is a separate decision.

### Orphan-blob handling on restore

A blob present in the blob-store snapshot but NOT referenced by any events-table row is an orphan. Orphans are harmless at the substrate layer (the events table is authoritative); they consume disk space but do not affect §2.1. Cleanup is a separate operation:

```sh
# Identify orphans (CLI tool deferred to follow-on; sketch only):
for blob_path in $(find ./blobs -type f); do
  hash=$(basename "$blob_path")
  if ! sqlite3 ./ghost-trace.db "SELECT 1 FROM events WHERE hex(event_hash) = '$hash' LIMIT 1" | grep -q 1; then
    echo "orphan: $blob_path"
  fi
done
```

Orphan deletion is OPERATOR DISCRETION; never run it as part of automated restore. Always preserve orphans during the restore window for forensic inspection.

## Substrate Migration

Forward-look for when [`§0027`](../charter/decision-log.md) R-store-1/2/3/4 reversal conditions fire. The migration shape per [`§0027`](../charter/decision-log.md) Proposal item 6 + Migration section:

1. Read events table in commit order; for each row, read the referenced blob; produce a `(event_hash, message_type, payload_bytes)` triple.
2. Write the triple to the target substrate per its commit semantics (NATS JetStream + content-hash subject; Kafka topic + content-hash key; PostgreSQL events table; etc.).
3. Verify the target substrate's committed-event-hash set matches the source's.

The migration is one-time, read-then-write; no payload re-serialization required (Protobuf bytes are substrate-independent per [`§0024`](../charter/decision-log.md)). Reversal cost is bounded by read-then-write throughput against the target substrate.

Concrete migration procedure lands when an [`§0027`](../charter/decision-log.md) reversal condition fires; this section is the forward-look skeleton operators can reference at that time.

## Minimum Observability

Inception-phase observability is structured-output via stderr; the ingestion service emits JSON `fatalLog` entries on unrecoverable errors (per [`§0032`](../charter/decision-log.md)) and SHOULD emit operational notifications at the thresholds documented in §Disk-Capacity Monitoring.

Operator-visible observables (inception phase):

| Observable | Channel | Trigger |
|---|---|---|
| Successful ingest confirmation | stdout (JSON `confirmation`) | per-message; produced by `readLoop` |
| Recoverable input error | stdout (JSON `ingestError`) | per-message; bad base64 or proto bytes |
| Unrecoverable §2.1 violation | stderr (JSON `fatalLog`) + non-zero exit | per [`§0032`](../charter/decision-log.md) |
| Disk-capacity threshold crossing | stderr (JSON `disk-capacity` notification — follow-on) | 70% / 85% / 95% thresholds |
| Backup completion | ops journal (operator-maintained, free-form) | per-backup |
| VACUUM completion | ops journal | per-VACUUM |
| Restore completion | ops journal | per-restore |

The ops journal is operator-maintained free-form; integration with structured observability tooling (Prometheus metrics endpoint, OpenTelemetry traces) is deferred to operational-ops-automation work.

## Anti-Patterns

By analogy to Charter [§2.1 Forbidden Anti-Patterns](../charter/constitutional-charter.md#21-observational-integrity). Each is concrete and operationally detectable.

- **Restoring a SQLite backup against a blob-store snapshot from a different timestamp.** Pairs SQLite index rows with blob content from a different point in time; references rows that point at blobs that no longer exist (or that have been superseded). Detectable: hash-verification gate at restore time surfaces missing-blob or hash-mismatch errors. Prevention: the manifest file is the unit of restoration; never override it.

- **Skipping the hash-verification gate during restore.** Restoring a backup without verifying every `event_hash` resolves to a matching blob is restoring an unverified substrate; subsequent reads may surface §2.1 violations that should have been caught during recovery. Detectable: restore procedure documents the gate as MUST; skipping it is a process violation.

- **Running VACUUM during peak ingestion without operator awareness.** VACUUM acquires an exclusive lock briefly; under peak ingestion, the lock causes Append-rejection cascade to upstream producers. Detectable: monitor Append rejection rate during VACUUM windows; if rejection rate spikes, the operational timing was wrong. Prevention: schedule VACUUM during idle windows when available.

- **Deleting orphan blobs as part of automated restore.** Orphans are forensic artifacts at restore time; their presence indicates either a pre-backup ingest failure (legitimate; the blob was written, the index row was not) or a post-restore inspection opportunity. Automated deletion forfeits both. Detectable: orphan-deletion code paths SHOULD only run as operator-invoked separate operations.

- **Treating retention policy as a single threshold rather than a tiered set.** A retention policy that says "delete backups older than 24 hours" loses point-in-time recovery beyond 24 hours; if the operator discovers a `§2.1` violation that occurred 48 hours ago, no backup remains to restore from. The tiered retention (hot-path 24h, idle-path 30d, quarterly indefinite) preserves recovery granularity at multiple time horizons.

- **Allowing the SQLite WAL file to grow unboundedly.** Under `journal_mode=WAL`, SQLite checkpoints automatically; manual `PRAGMA wal_checkpoint(TRUNCATE)` is the operator's tool if the WAL file exceeds expected size. Unbounded WAL growth indicates either a checkpoint stall (concurrent long-running reader holding back the checkpoint) or a misconfigured `wal_autocheckpoint`. Detectable: monitor WAL file size; alert if it exceeds 10× the canonical-corpus typical-entry size × ingest rate × checkpoint interval.

## Open Questions

- **`verify` CLI tool.** Restoration currently relies on the substrate's existing ReadBlob path to surface hash mismatches at first read. A dedicated `verify` CLI that walks every events-table row + checks every blob hash up-front would surface verification failures before the service starts. Deferred to follow-on operational-ops work when the first restoration cycle is exercised.
- **Backup encryption.** Inception-phase backups are unencrypted local-filesystem copies. Production-tier backup encryption is deferred to follow-on work when remote-storage backups are introduced.
- **Backup compression.** Blob-store content is canonical Protobuf bytes; SQLite snapshots have their own representation. Whether to compress backups (and which compression tool) is operator preference at inception; standardization deferred.
- **Cross-region backup replication.** Inception-phase backups are single-host. Multi-region replication is a separate concern coupled to substrate-migration (per §Substrate Migration above) and to disaster-recovery posture; deferred.
- **Observability tooling integration.** This document specifies WHAT to surface; the HOW (Prometheus, OpenTelemetry, JSON-to-file shipping) is operator preference at inception phase. Standardization deferred to operational-ops-automation work.

## References

- [`docs/charter/constitutional-charter.md` §2.1](../charter/constitutional-charter.md#21-observational-integrity) — substrate-immutability constraint operations are evaluated against.
- [`docs/charter/decision-log.md` §0027`](../charter/decision-log.md) — storage-technology selection; Proposal item 5 (backup ordering); Open Questions discharged here.
- [`docs/charter/decision-log.md` §0028`](../charter/decision-log.md) — canonical-serialization-contract; content-addressing makes blob-store backups verifiable.
- [`docs/charter/decision-log.md` §0029`](../charter/decision-log.md) — concurrency-pattern; substrate-writer serialization (single Append entry point) the backup procedure inherits.
- [`docs/charter/decision-log.md` §0032`](../charter/decision-log.md) — unrecoverable-error shutdown escalation; operations that detect §2.1 violations trigger the same pathway.
- [`docs/charter/decision-log.md` §0033`](../charter/decision-log.md) — introduction of this document.
- [`docs/architecture/storage-model.md`](./storage-model.md) — Tier 0/1/2 reference; operations apply to Tier 1 (the inception-phase SQLite + blob-store substrate).
- [`docs/architecture/canonical-serialization-contract.md`](./canonical-serialization-contract.md) — hash-verification anti-pattern restoration relies on.
- [`docs/architecture/concurrency-pattern.md`](./concurrency-pattern.md) — error-propagation pathway operations escalate through.
- [`docs/architecture/replay-model.md`](./replay-model.md) — Phase 4 retrospective replay obligation backing the quarterly-retention default.
- [SQLite Backup API documentation](https://www.sqlite.org/backup.html) — `.backup` command semantics.
- [SQLite WAL Mode documentation](https://www.sqlite.org/wal.html) — WAL checkpoint behavior.
- [SQLite VACUUM documentation](https://www.sqlite.org/lang_vacuum.html) — VACUUM operation semantics.
