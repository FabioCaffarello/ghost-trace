# CIC-IDS-2017 → NetworkObservation mapping

**Status:** Drafted at the landing of `services/ingestion/internal/adapters/cic_ids/` per [`decision-log §0145`](../charter/decision-log.md). First public adversarial ingestion source integrated under Domain Pack v0.1 [`§0143`](../charter/decision-log.md) D2 (three-source-parallel ingestion strategy).

## Source

- **Dataset:** CIC-IDS-2017 / CIC-IDS-2018 (Canadian Institute for Cybersecurity).
- **Paper:** Sharafaldin, Lashkari, Ghorbani. *Toward Generating a New Intrusion Detection Dataset and Intrusion Traffic Characterization.* ICISSP 2018.
- **Feature extractor:** [CICFlowMeter](https://www.unb.ca/cic/research/applications.html). Produces ~80 features per network flow as CSV rows.
- **Format consumed by adapter:** CIC-IDS-2017 CSV format (header row + flow rows). CIC-IDS-2018 / CSE-CIC-IDS2018 column variants are deferred to a subsequent revision when empirically integrated.

## Coverage estimate (empirically confirmed at §0145)

[`§0143`](../charter/decision-log.md) RFC Domain Pack v0.1 CIC-IDS preview estimated **~18% aggregate dataset coverage** across the five NetworkObservation sub-modalities. This adapter empirically confirms the estimate within the ENDPOINT + TCP-FLAGS classes; the remaining ~82% (FLOW-STATISTICS class) is the empirical pressure surface for the `flow_record_summary` OMQ candidate per [`§0145`](../charter/decision-log.md).

| Sub-modality | Estimate (§0143) | Empirical (§0145) | Notes |
|---|---|---|---|
| `ip_asn` | ~50% | **clean** (IP + port mapped; ASN/geo deferred to `lookup_source = ""` enrichment) | Two observations emitted per flow row (src side + dst side). |
| `tcp_fingerprint` | ~30% | **partial with semantic loss** | Flag counts → flag-byte sequence approximation; TCP options absent (CICFlowMeter does not preserve); window from `Init_Win_bytes_forward` when present. |
| `tls_ja4` | 0% | **0%** | Dataset does not expose TLS ClientHello fingerprints. |
| `http2_frame_pattern` | 0% | **0%** | HTTP-layer frame patterns not in flow-level features. |
| `dns_pattern` | ~10% | **0% in this adapter version** | DNS flows present as Protocol = 17 + DstPort = 53; QTYPE/RCODE not preserved in CICFlowMeter output. Future revision may add anonymized QNAME-pattern derivation from packet-capture sources. |

## Column-by-column mapping table

Three classes per [`services/ingestion/internal/adapters/cic_ids/columns.go`](../../services/ingestion/internal/adapters/cic_ids/columns.go) classification:

### ENDPOINT class → `ip_asn`

| CIC-IDS Column | NetworkObservation field | Transformation |
|---|---|---|
| `Flow ID` | (none; CICFlowMeter-internal) | Discarded. The substrate's content-addressable identifier per [§2.1](../charter/constitutional-charter.md#21-observational-integrity) supersedes any flow-level identifier. |
| `Source IP` | `ip_asn.ip_address` (src observation) | Verbatim. |
| `Source Port` | `endpoint_ref` (src observation) | Concatenated as `<src_ip>:<src_port>`. |
| `Destination IP` | `ip_asn.ip_address` (dst observation) | Verbatim. |
| `Destination Port` | `endpoint_ref` (dst observation) | Concatenated as `<dst_ip>:<dst_port>`. |
| `Protocol` | (none directly; gates `tcp_fingerprint` emission) | Protocol = 6 (TCP) triggers `tcp_fingerprint` observation; other protocols emit only `ip_asn` pair. |
| `Timestamp` | envelope `observed_at` | Parsed as `DD/MM/YYYY HH:MM` → Unix nanoseconds. Minute-precision per dataset; sub-minute precision irrecoverable. Parse failure → row-index × 60s substitute (preserves substrate hash-stability across ingest runs). |

### TCP-FLAGS class → `tcp_fingerprint` (partial)

| CIC-IDS Column | NetworkObservation field | Transformation + semantic loss notes |
|---|---|---|
| `FIN Flag Count` | `tcp_fingerprint.flags_sequence` (FIN byte 0x01 repeated by count) | Semantic loss: COUNT, not per-packet sequence. |
| `SYN Flag Count` | `tcp_fingerprint.flags_sequence` (SYN byte 0x02 repeated by count) | Same. |
| `RST Flag Count` | `tcp_fingerprint.flags_sequence` (RST byte 0x04 repeated by count) | Same. |
| `PSH Flag Count` | `tcp_fingerprint.flags_sequence` (PSH byte 0x08 repeated by count) | Same. |
| `ACK Flag Count` | `tcp_fingerprint.flags_sequence` (ACK byte 0x10 repeated by count) | Same. |
| `URG Flag Count` | `tcp_fingerprint.flags_sequence` (URG byte 0x20 repeated by count) | Same. |
| `CWE Flag Count` | `tcp_fingerprint.flags_sequence` (CWE byte 0x80 repeated by count) | Same. |
| `ECE Flag Count` | `tcp_fingerprint.flags_sequence` (ECE byte 0x40 repeated by count) | Same. |
| `Init_Win_bytes_forward` | `tcp_fingerprint.window_size` | Verbatim when present. 0 = absent. |
| `Init_Win_bytes_backward` | (none; not currently consumed) | Single window per observation; forward side is canonical (matches SYN-from-client direction). |
| `Fwd Header Length` / `Bwd Header Length` | (none; not currently consumed) | Reserved for future revision; aggregate header lengths do not directly correspond to TCP options. |

**`tcp_fingerprint` observations are gated:** emitted only when Protocol = 6 (TCP) AND at least one TCP flag count is non-zero. TCP flows with all-zero flag counts (aggregated summary rows where flag preservation was lost upstream) produce only `ip_asn` pair, no `tcp_fingerprint`.

**TCP options absent:** CICFlowMeter does not preserve TCP option kind sequences (MSS, WSCALE, SACK-OK, TIMESTAMP). `tcp_fingerprint.tcp_options_order` is left empty; `tcp_fingerprint.mss` and `tcp_fingerprint.ttl` are left at zero. Downstream p0f-style signature comparison is degraded relative to packet-capture-source observations.

### FLOW-STATISTICS class → `flow_record_summary` OMQ candidate (NOT mapped)

The remaining CICFlowMeter columns — durations, packet length statistics, IAT statistics, byte/packet rates, segment sizes, active/idle stats, down/up ratios — do NOT map to any current NetworkObservation sub-modality. These are flow-level statistical aggregations over the lifetime of a flow, not per-observation network-layer features.

| CIC-IDS Column (representative subset) | Class | Empirical pressure |
|---|---|---|
| `Flow Duration` | flow-level scalar | OMQ candidate |
| `Total Fwd Packets` / `Total Backward Packets` | flow-level counts | OMQ candidate |
| `Total Length of Fwd Packets` / `Total Length of Bwd Packets` | flow-level byte sums | OMQ candidate |
| `Fwd Packet Length Max/Min/Mean/Std` | per-direction packet-length statistics | OMQ candidate |
| `Bwd Packet Length Max/Min/Mean/Std` | per-direction packet-length statistics | OMQ candidate |
| `Flow Bytes/s` / `Flow Packets/s` | flow rates | OMQ candidate |
| `Flow IAT Mean/Std/Max/Min` | inter-arrival-time statistics | OMQ candidate |
| `Fwd IAT Total/Mean/Std/Max/Min` | per-direction IAT statistics | OMQ candidate |
| `Bwd IAT Total/Mean/Std/Max/Min` | per-direction IAT statistics | OMQ candidate |
| `Min Packet Length` / `Max Packet Length` | flow-level extremes | OMQ candidate |
| `Packet Length Mean/Std/Variance` | flow-level statistics | OMQ candidate |
| `Down/Up Ratio` | derived ratio | OMQ candidate |
| `Average Packet Size` | flow-level mean | OMQ candidate |
| `Avg Fwd Segment Size` / `Avg Bwd Segment Size` | per-direction segment statistics | OMQ candidate |
| `Active Mean/Std/Max/Min` / `Idle Mean/Std/Max/Min` | flow-state-duration statistics | OMQ candidate |

**OMQ candidate to be opened separately:** `flow_record_summary` as a sixth NetworkObservation sub-modality OR as a distinct Cat I observation type. Per §0145, the OMQ formally opens via a separate `ontology-revision-flow-record-summary-modality` RFC; not pre-empted in this adapter PR (compatible with §0144 Methodological observation 2: anticipate ≠ pre-resolve).

### `Label` column

CIC-IDS includes a `Label` column carrying the attack-class annotation (`BENIGN`, `DoS-Slowloris`, `Brute-Force`, etc.). **NOT consumed by the adapter.** Per [§3 N1](../charter/constitutional-charter.md#3-non-goals) (no truth at substrate): the substrate is purely observational; ground-truth labels are operator annotation for downstream F3 signature evaluation, not Cat I observations. Labels are preserved out-of-band (operator may retain the original CSV) for evaluation purposes.

## OMQ candidates surfaced by this adapter

Two OMQs surface empirically through CIC-IDS integration:

1. **`flow_record_summary` modality** (NEW; primary surface from this adapter). The bulk of CICFlowMeter features are flow-level statistical aggregations that don't fit any current sub-modality of NetworkObservation. Proposed for a separate ontology-revision RFC when the second ingestion source (synthetic or honeypot) confirms the gap is structural (not specific to CICFlowMeter).
2. **Phenomenon-vs-record reconciliation** (existing; introduced as anticipated in §0143 + §0144). The `collector_ref` field on NetworkObservation envelope is exercised here: every observation emitted by this adapter carries `collector_ref = "cic-ids-2017-adapter:v1"`. When synthetic / honeypot sources also emit observations, the same phenomenon (e.g., same IP appearing in multiple sources) produces distinct substrate records with distinct `collector_ref` values. The OMQ then formally opens for committee resolution per §0143 trigger ("when F1 has two ingestion sources stable").

## Adapter discipline

- **Stream-friendly.** `Ingest(ctx, ingester, reader, collectorRef, env)` processes the CSV row-by-row; memory footprint is O(1) per row, not O(N) per file. Suitable for the full CIC-IDS dataset without materializing all records in memory.
- **Idempotent.** Substrate commit per row produces content-addressed records; re-ingest of the same CSV produces identical hashes and substrate's PRIMARY KEY conflict makes re-commit a no-op.
- **Counter discipline.** `Report` carries `RowsParsed`, `RowsRejected`, `ObservationsCommitted`, `IpAsnEmitted`, `TcpFingerprintEmitted`, `FlowStatisticsDropped`. The `FlowStatisticsDropped` counter is the **empirical pressure surface** for the `flow_record_summary` OMQ: it increments per row, surfacing the volume of unmapped features.
- **Operator opt-in for labels.** The adapter ignores the `Label` column; operators that wish to retain labels for evaluation must preserve them out-of-band per the §3 N1 discipline.

## CICFlowMeter header convention (real-world distribution)

The CIC-IDS-2017 GeneratedLabelledFlows distribution (Sharafaldin et al. ICISSP 2018) emits CSV files whose header row contains **leading whitespace on every column following each comma separator** — i.e. `Flow ID, Source IP, Source Port, ...` with a literal space character after each comma. Some Windows-produced variants additionally carry a trailing `\r` on the last column under CRLF line endings.

Per [`decision-log §0207`](../charter/decision-log.md), the adapter normalizes column names via `strings.TrimSpace` at `indexHeader` time, accepting both the CICFlowMeter convention and the canonical no-whitespace form. The normalization is silent — no operator-facing surface, no `Report` counter — because the resulting index keys are identical across both inputs. This was the first substrate-emergent §0022 pressure surfaced within the Domain Pack v0.1 program: the §0205 deployment scaffold's first real run against GeneratedLabelledFlows failed with `MissingColumnError: Source IP`, and the discrepancy between handcrafted canonical fixture and real-world distribution surfaced mechanically through the existing error path.

## CLI usage

The operator-facing CLI lift `cmd/ingest-cic-ids` (per [`decision-log §0204`](../charter/decision-log.md)) is a thin wrapper around `cic_ids.Ingest`. The eight operator workflow choices §0145 Consequences deferred are documented and tested at §0204; this section is the operator-facing summary.

```
ingest-cic-ids [flags] [csv-path]

Reads from csv-path when provided; from stdin otherwise.

Flags:
  -db        path to SQLite substrate              (default "./ghost-trace.db")
  -blobs     path to blob-store directory          (default "./blobs")
  -channel   ingestion channel identifier          (default: "cic-ids-file" with path arg, "stdin" without)
  -collector collector_ref on emitted observations (default "cic-ids-2017-adapter:v1")
  -progress  stderr progress line every N lines    (default 10000; 0 disables)
  -strict    exit non-zero if RowsRejected > 0     (default false)

Output:
  stdout — cic_ids.Report JSON (RowsParsed, RowsRejected, ObservationsCommitted, IpAsnEmitted, TcpFingerprintEmitted, FlowStatisticsDropped)
  stderr — progress lines + final one-line summary

Exit codes (mirror replay-all-* precedent per §0173):
  0  success
  2  tool/config error (bad flag, cannot open input or substrate)
  3  substrate error mid-run OR -strict && RowsRejected > 0
```

**Distinction from §0163 F3-envelope shape.** The Report JSON emitted by `ingest-cic-ids` is the ingest-tier output contract; it is **structurally distinct** from the signature-tier `{signature_name, candidate_count, candidates[], stats{...}}` envelope §0163 cravou for F3 CLIs. Both are stable wire-contracts for their respective CLI categories; a downstream consumer aggregating both (e.g., the §0205 deployment scaffold manifest) embeds them side-by-side under separate pipeline-step entries — no coercion of one shape into the other.

**Re-ingest is silent.** Substrate commits are content-addressed via BLAKE3; a second invocation against the same CSV produces identical hashes and `INSERT OR IGNORE` returns no-op rows. The Report still reflects RowsParsed > 0 for the second run (it counts library Append CALL counts, not net new rows); operators detecting re-ingest should observe substrate row counts directly rather than rely on Report variance.

**Timestamp fallback (audit-grade caveat).** When a CIC-IDS row's Timestamp column is missing or unparseable, the library falls back to row-index nanoseconds for substrate hash-stability. This fallback is **not currently surfaced** in Report counters; §0204 names the explicit trigger for adding a `TimestampsRecovered` counter — when §0205 manifest needs audit-row visibility into the substitute count, a separate library-tier PR adds the field.

## References

- [`decision-log §0143`](../charter/decision-log.md) — Domain Pack v0.1 anti-bot atlas framing PR; D2 three-source-parallel ingestion strategy.
- [`decision-log §0144`](../charter/decision-log.md) — F1.NetworkObservation discriminated-union typing; first proto-definition landing.
- [`decision-log §0145`](../charter/decision-log.md) — CIC-IDS adapter landing; this mapping document; first public adversarial source integrated.
- [`decision-log §0204`](../charter/decision-log.md) — `cmd/ingest-cic-ids` CLI lift; eight operator workflow choices cravadas.
- [`decision-log §0207`](../charter/decision-log.md) — CICFlowMeter header-whitespace normalization; first substrate-emergent §0022 of Domain Pack v0.1.
- [Charter §2.1 Observational Integrity](../charter/constitutional-charter.md#21-observational-integrity)
- [Charter §3 N1 — no truth at substrate](../charter/constitutional-charter.md#3-non-goals)
- [`services/ingestion/internal/adapters/cic_ids/`](../../services/ingestion/internal/adapters/cic_ids/) — adapter implementation.
- [Sharafaldin, Lashkari, Ghorbani. ICISSP 2018](https://www.scitepress.org/Papers/2018/66398/) — CIC-IDS-2017 dataset paper.
- [CICFlowMeter](https://www.unb.ca/cic/research/applications.html) — feature extractor.
