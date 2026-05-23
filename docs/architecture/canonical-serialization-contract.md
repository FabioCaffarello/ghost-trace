# Canonical Serialization Contract

**Status:** Active. First non-scaffold architecture document. Discharges the follow-on commitment named in [`decision-log §0024`](../charter/decision-log.md) AP5 mitigation step (b) and [`§0027`](../charter/decision-log.md) Consequences. Extended at [`decision-log §0136`](../charter/decision-log.md) to consolidate the paired-dimension commitment ([§2.6](../charter/constitutional-charter.md#26-evidential-independence-integrity) operational discharge), the α derivation rule (per [`§0133`](../charter/decision-log.md) Q3 resolution), the τ + β-graph influence storage (per [`§0134`](../charter/decision-log.md) Q5 resolution), and the Layer B L-BC-OR firing predicate (per [`§0135`](../charter/decision-log.md) Layer B resolution). Further extended at [`decision-log §0138`](../charter/decision-log.md) to bundle Layer A's `N_A` cadence parameter into the LayerBParameters proto + fix inception-phase parameter values (T_B = K_C = 0.5; N = 1000; window form = W-count; per-subtype divergence = U-uniform; N_A = 1 day; per-parameter reversal-conditions record). Further extended at [`decision-log §0139`](../charter/decision-log.md) to generalize the BLAKE3-hash-list element-shape discipline from the two influence-storage fields (`closure_hashes`, `direct_influenced_by`) to the full set of canonical-form-load-bearing 32-byte BLAKE3 `repeated bytes` fields (adding `source_event_hashes`, `antecedent_formation_event_hashes`, `successor_formation_event_hashes`) — the uniform structural commitment (32-byte length + ascending lexicographic order + no duplicates) was already in the proto comments; this revision crystallizes it at the contract layer and extends marshalling-boundary rejection accordingly.

> This document specifies the canonical-serialization contract for Ghost Trace: the bit-stable mapping from a Protobuf message instance to a byte sequence to a content-addressable identifier. The mapping is the falsifiability predicate for [Charter §2.1 Observational Integrity](../charter/constitutional-charter.md#21-observational-integrity) at the substrate; without bit-stability, content-hash recomputation on read cannot serve as the mutation-detection mechanism.

## Constitutional Anchors

- [Charter §2.1 Observational Integrity](../charter/constitutional-charter.md#21-observational-integrity) — requires content-addressable identifiers sufficient to detect mutation if attempted. The contract specified here is the operational mechanism.
- [Charter §2.4 Inferential Influence Disclosure](../charter/constitutional-charter.md#24-inferential-influence-disclosure) v0.5 — requires `influenced_by` chain declaration for inferential-commitment records. The Influence Storage section below specifies the substrate encoding.
- [Charter §2.5 Hypothesis Lifecycle Explicitness](../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) v0.3 — requires `demotion` lifecycle events with a designated structural test. The Demotion-Candidacy Predicate section below specifies the test's substrate-enforceable form.
- [Charter §2.6 Evidential Independence Integrity](../charter/constitutional-charter.md#26-evidential-independence-integrity) v0.6 — requires paired `confidence` + `evidential_independence` dimensions on inferential-commitment records. The Paired-Dimension Commitment section below specifies the substrate enforcement. (Note: §2.6 prose references `§0034` as the paired-dimension enforcement entry — this is a stale anchor per [`§0136`](../charter/decision-log.md) Anchor-fidelity observation; the actual operational discharge is [`§0136`](../charter/decision-log.md). Patch amendment to §2.6 deferred.)
- [`decision-log §0024`](../charter/decision-log.md) — schemas-technology selection: Protocol Buffers (proto3). The canonical-serialization layer this document specifies operates against proto3-generated message types.
- [`decision-log §0025`](../charter/decision-log.md) — implementation-language selection: Go. The Protobuf library binding specified here is the Go binding (`google.golang.org/protobuf` v2 module).
- [`decision-log §0027`](../charter/decision-log.md) — storage-technology selection: SQLite + content-addressed blob-store. The blob-store's content addressing operates against the canonical bytes specified here.
- [`decision-log §0133`](../charter/decision-log.md) — Q3-α resolution: `evidential_independence` = source-count ratio over Cat I provenance roots. The Evidential Independence section below specifies the substrate encoding and validation.
- [`decision-log §0134`](../charter/decision-log.md) — Q5-τ resolution: `influenced_by` is the transitive closure of declared direct edges; storage strategy is β-graph (direct edges + per-record cached closures). The Influence Storage section below specifies the substrate encoding.
- [`decision-log §0135`](../charter/decision-log.md) — Layer B L-BC-OR resolution: demotion-candidacy predicate is the disjunction of evidence-staleness and influence-saturation tests. The Demotion-Candidacy Predicate section below specifies the substrate-enforceable form.
- [`decision-log §0136`](../charter/decision-log.md) — this consolidation entry.

## Subordination

This document is subordinate to the [Constitutional Charter](../charter/constitutional-charter.md) and the [Ontology](../ontology/ontology.md). A conflict with either resolves by revising this document.

## Scope

This document specifies, with the discipline of a contract that downstream code is held to:

1. The serialization stack: which library, which call, which options.
2. The hash stack: which algorithm, which library, which output form.
3. What constitutes a schemas-evolution event (per [`§0024`](../charter/decision-log.md) AP5 step c).
4. The CI golden-file gate (per [`§0024`](../charter/decision-log.md) AP5 step d).
5. Upgrade discipline for library versions.

It does NOT specify:

- The concrete `.proto` files (those are content created when the ingestion service skeleton work begins).
- The downstream service architecture (service-tier RFCs as needed).
- The blob-store filesystem layout details (those are operational concerns per [`§0027`](../charter/decision-log.md) Open Questions).

## Serialization Stack

### Library binding

The canonical Protobuf binding for Go is **`google.golang.org/protobuf`** (the v2 module). The legacy `github.com/golang/protobuf` (v1) is in maintenance mode and uses internal aliases to v2; service code MUST NOT import v1 directly. Per [`§0025`](../charter/decision-log.md) Proposal item 2.

### Marshal call

Canonical bytes are produced via:

```go
proto.MarshalOptions{
    Deterministic: true,
    AllowPartial:  false,
    UseCachedSize: false,
}.Marshal(msg)
```

The three options are load-bearing:

- `Deterministic: true` — instructs the marshaller to order fields by ascending field-number and to use canonical encoding for map fields (per [`§0024`](../charter/decision-log.md) modifications enacted at resolution). Without this option, the library reserves the right to produce non-deterministic output.
- `AllowPartial: false` — rejects messages with missing required fields; surfaces schemas violations at write time rather than at read time. proto3 has no required fields per se, but `AllowPartial: false` preserves the discipline boundary in case proto2-derived types ever appear in the codebase.
- `UseCachedSize: false` — disables the library's size-caching optimization. The cached-size path can interact awkwardly with deterministic encoding under specific message-type combinations; disabling preserves bit-stability across reentrant marshalling.

### Constraint: `map<K, V>` ban

Per [`§0024`](../charter/decision-log.md) AP6, `map<K, V>` field types are forbidden in canonical-form-load-bearing message types. Key-value collections use `repeated SubMessage { key, value }` with ascending-key sort enforced at construction. The deterministic-marshalling option above does specify canonical map encoding, but the AP6 ban is the stronger commitment — substrate-load-bearing types avoid the construct entirely.

### Type construction discipline

Generated Go types from `protoc-gen-go` may have fields populated in any order; the deterministic marshaller produces canonical bytes regardless. However: construction-time validation (per `AllowPartial: false`) requires that all `oneof` branches resolve to exactly one populated branch when the field is meant to be present, and that all sub-message fields are non-nil when they are meant to be present. The construction-time discipline is enforced by the service-tier typed-boundary functions per [`§0025`](../charter/decision-log.md) AP1.

## Hash Stack

### Algorithm

**BLAKE3.** 256-bit output. The full 32-byte digest is the content-addressable identifier.

Rationale: BLAKE3 is the canonical modern cryptographic hash (RFC-quality spec; widely-implemented; explicit performance + parallelism design; well-characterized collision resistance). SHA-256 would also work; BLAKE3 is selected because of its explicit support for tree-mode parallelism (useful for future large-payload hashing) and its predictable performance characteristics. SHA-3 considered and rejected at this layer because its design tradeoffs (Keccak's permutation cost) are not well-matched to the high-message-rate-with-small-payloads workload Ghost Trace anticipates.

### Library binding

The canonical Go binding for BLAKE3 is **`lukechampine.com/blake3`**. It is a maintained Go-native implementation. Pinned per [`§0025`](../charter/decision-log.md) library-version-pinning discipline.

### Hash call

Content-hash is produced via:

```go
func ContentHash(canonicalBytes []byte) [32]byte {
    return blake3.Sum256(canonicalBytes)
}
```

The 32-byte array is the canonical identifier form. Encodings for non-binary contexts (filesystem paths, structured-output messages, error reports) use lowercase hex (`fmt.Sprintf("%x", h)` — 64-character string). The lowercase-hex convention is fixed; uppercase-hex or base64 encoding are NOT permitted in canonical-form-load-bearing contexts (a hash recorded as base64 in one path and as hex in another path would not compare equal as strings, which has subtle downstream effects).

## Paired-Dimension Commitment

Per [Charter §2.6](../charter/constitutional-charter.md#26-evidential-independence-integrity) v0.6 + [`§0136`](../charter/decision-log.md) (operational discharge): every record that is structurally inferential carries two structurally-distinct dimensions at substrate commit. The two dimensions are required by the substrate; commitment with only one is structurally precluded at the canonical-serialization-contract layer.

### Records subject to the commitment

The paired-dimension requirement applies to:

- Category II construct records (the `Construct` message family).
- Category III hypothesis records (the four concrete subtypes per [`§0010`](../charter/decision-log.md) Q2-A.2: `BehavioralCluster`, `CoordinationRing`, `CampaignHypothesis`, `AutomationGroup`).
- Any Assertion record whose subject reference is `subject_ref_construct` or `subject_ref_hypothesis` per [`§0016`](../charter/decision-log.md) Q3-subject-ref resolution.

Category I observation records are NOT subject to the commitment — they carry no inferential commitment per [§2.1](../charter/constitutional-charter.md#21-observational-integrity).

### Required fields

For records subject to the commitment, two paired fields are required:

```proto
message InferentialCommitment {
    Confidence confidence = 1;             // Required; magnitude
    EvidentialIndependence evidential_independence = 2; // Required; degree
}
```

Both fields are required (`AllowPartial: false` in the canonical Marshal call rejects records missing either). Confidence and evidential_independence are structurally independent at substrate per [Charter §2.6](../charter/constitutional-charter.md#26-evidential-independence-integrity) — neither is derivable from the other at commit time.

### Validation discipline

Substrate-commit fails at the canonical-marshalling boundary when a record subject to the commitment is missing either dimension. The failure mode is deterministic and not bypassable at the consumer layer.

## Evidential Independence (Q3-α)

Per [`§0133`](../charter/decision-log.md) Q3-α resolution + [`§0136`](../charter/decision-log.md) (operational discharge): the `evidential_independence` value is computed at substrate write time per the source-count-ratio formula.

### Formula

```
evidential_independence(record) =
    (count of Cat I primary observation roots in record.subject_ref_* chain
     NOT reachable via any influenced_by edge from a promoted hypothesis)
    /
    (total Cat I roots in record.subject_ref_* chain)
```

Range: `[0, 1]`. The reachability predicate uses the transitive `influenced_by` semantic per the Influence Storage section below (Q5-τ).

### Encoding

The `evidential_independence` field encodes as a rational number — a numerator/denominator pair — to preserve the bounded-resolution structure α commits to per [`§0133`](../charter/decision-log.md) Phase 4 Finding 4. A floating-point encoding is forbidden (it obscures the structural-resolution commitment).

```proto
message EvidentialIndependence {
    uint64 numerator = 1;    // Cat I roots NOT reachable via influenced_by
    uint64 denominator = 2;  // Total Cat I roots; must be > 0
}
```

The denominator is positive; a denominator of zero (no Cat I roots in the chain) is a structural error and indicates the record's provenance chain does not terminate at Cat I per [§2.3](../charter/constitutional-charter.md#23-provenance-integrity) v0.4. Such records are rejected at the marshalling boundary.

### Validation discipline

Substrate recomputes the formula from the substrate-committed provenance + influence subgraphs at commit time and compares against the committed value byte-for-byte. Mismatch is rejected. Producer-side derivation is permitted (the producer must apply the same formula), but the substrate verifies the value; consumer-side acceptance of the producer's value is not required.

### Cat-II structural transmission

Per [`§0134`](../charter/decision-log.md) Cat II structural transmission commitment: Cat II constructs deterministically transmit `influenced_by` membership from their inputs. The α formula's reachability predicate honors this transmission — a Cat I root reachable only through an influenced Cat II intermediate is counted as influenced (excluded from the numerator).

## Influence Storage (Q5-τ + β-graph)

Per [`§0134`](../charter/decision-log.md) Q5-τ resolution + [`§0136`](../charter/decision-log.md) (operational discharge): the `influenced_by` relation is the transitive closure of declared direct edges. Storage strategy is β-graph: substrate stores direct edges per record + per-record cached closures.

### Direct edges

Each inferential-commitment record carries a `direct_influenced_by` field listing the Cat III hypothesis records whose influence the producer directly declares at formation:

```proto
message InferentialRecord {
    // ... other fields ...
    repeated bytes direct_influenced_by = 10; // 32-byte BLAKE3 hashes of Cat III hypothesis records
    repeated bytes closure_hashes = 11;       // 32-byte BLAKE3 hashes of all transitively-reachable Cat III hypotheses
}
```

Each element of `direct_influenced_by` is the 32-byte BLAKE3 content-hash of a Cat III hypothesis record. Elements are stored in ascending lexicographic order; out-of-order or duplicate entries are rejected at marshalling.

### Closure encoding

The `closure_hashes` field stores the transitive closure of `direct_influenced_by` — every Cat III hypothesis reachable through any chain of `influenced_by` edges. Encoding is `repeated bytes` (not `map<K,V>` per [`§0024`](../charter/decision-log.md) AP6); elements are 32-byte BLAKE3 hashes in ascending lexicographic order.

### Computation algorithm

Per [`§0021`](../charter/decision-log.md) substrate-time generation: at write time, the substrate computes `closure_hashes` for the new record by:

1. Initializing `closure_hashes` as the union of `direct_influenced_by` elements with the `closure_hashes` of each record referenced by `subject_ref_*` AND each Cat III hypothesis in `direct_influenced_by`.
2. Sorting the union in ascending lexicographic order.
3. Deduplicating (closure is a set).

Amortized cost is O(input-set-size) per write — the substrate reads input records' already-committed `closure_hashes` rather than recomputing the transitive traversal from scratch. The recursion terminates because `closure_hashes` is committed before the record that consumes it (§0021 substrate-time discipline).

### Cat-II structural transmission

Per [`§0134`](../charter/decision-log.md) Cat II structural transmission commitment + [`§0136`](../charter/decision-log.md): a Cat II construct's `closure_hashes` IS the union of the `closure_hashes` of the records it is deterministically derived from. This is structural, not optional — Cat II constructs are deterministic views of their inputs per [§2.2](../charter/constitutional-charter.md#22-epistemic-separation), so their influence chain is the union of their inputs' chains.

### Validation discipline

Substrate recomputes `closure_hashes` from the substrate-committed direct edges + input records' closures at commit time and compares element-by-element against the committed value. Mismatch is rejected. Out-of-order or duplicate elements in either `direct_influenced_by` or `closure_hashes` are rejected. The element-shape discipline (32-byte BLAKE3 length + ascending lexicographic order + no duplicates) is the influence-storage-specific instance of the uniform hash-list discipline specified in the next section; the substrate-tier closure-recomputation check is independent of and supplementary to the marshalling-boundary element-shape check.

## Hash-List Field Discipline

Per [`§0139`](../charter/decision-log.md): every canonical-form-load-bearing `repeated bytes` field whose elements are 32-byte BLAKE3 content-hashes — whether the semantic surface is influence storage, observational provenance, or lifecycle antecedent/successor reference — carries the same uniform structural commitment:

1. **Element length.** Every element is exactly 32 bytes (BLAKE3-256 per the Hash Stack section above).
2. **Ascending lexicographic order.** Elements are stored in ascending byte-lexicographic order.
3. **No duplicates.** No two elements are byte-equal.

### Fields subject to the discipline

| Field | Proto sites | Semantic surface |
|---|---|---|
| `closure_hashes` | 4 formation protos + `OperationalSession` | Influence storage closure (§0134 Q5-τ) |
| `direct_influenced_by` | 4 formation protos + `OperationalSession` | Influence storage direct edges (§0134 Q5-τ + §0136 β-graph) |
| `source_event_hashes` | 4 formation protos | Observational provenance roots ([§2.3](../charter/constitutional-charter.md#23-provenance-integrity) v0.4) |
| `antecedent_formation_event_hashes` | 4 merge protos | Lifecycle merge antecedents ([§2.5](../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) v0.3 + §0045) |
| `successor_formation_event_hashes` | 4 split protos | Lifecycle split successors ([§2.5](../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) v0.3 + §0045) |

The fields belong to different sections of the constitutional surface — influence-storage (§2.4 + §0134), provenance-chain (§2.3), lifecycle-event antecedence (§2.5) — but they share the same structural shape because the underlying construct is the same: a stable, set-shaped reference to other substrate-committed records by their BLAKE3 content-hash. The semantic distinction is real and remains; the structural discipline is uniform.

### Validation discipline

Substrate-commit fails at the canonical-marshalling boundary when any field in the table above contains an element of length ≠ 32 bytes, OR an out-of-order pair, OR a duplicate pair. The failure mode is deterministic and not bypassable at the consumer layer. The check is field-shape-only — it does NOT verify that elements are substrate-resident, nor that closure_hashes equals the recomputed union per the Cat II structural-transmission commitment, nor that source_event_hashes terminate at Cat I roots per [§2.3](../charter/constitutional-charter.md#23-provenance-integrity); those are substrate-tier validations performed against substrate-committed state, supplementary to the field-shape check at the marshalling boundary.

### Scope and future extension

The check is scoped to top-level `repeated bytes` fields of the canonical-form-load-bearing types named above. A future canonical-form-load-bearing message type that introduces a new 32-byte BLAKE3 hash-list field inherits this discipline structurally and must be added to the marshalling-boundary check at the same commit as the proto change (per the Schemas-Evolution Events boundary item 5 — `.proto` file structural commitments change). Singular `bytes` content-hash fields (e.g., `produced_formation_event_hash`, `source_event_hash`) are NOT subject to this discipline at the field-list level — their structural commitment is length-only (32 bytes) and is enforced by the same proto-field-shape mechanism at substrate-commit time, not by the ordered/deduplicated list-shape rules.

## Demotion-Candidacy Predicate (Layer B L-BC-OR)

Per [`§0135`](../charter/decision-log.md) Layer B L-BC-OR resolution + [`§0136`](../charter/decision-log.md) (operational discharge): the demotion-candidacy predicate for a promoted hypothesis is the disjunction of evidence-staleness and influence-saturation tests, composed with the outer Layer A cadence gate.

### Predicate

```
DEMOTE-CANDIDATE(H) :=
    Layer A(H)
    AND
    ((freshness_B(H) < T_B) OR (saturation_C(H) > K_C))
```

### Layer A — Cadence gate

Per [`§0011`](../charter/decision-log.md) Q4 resolution:

```
Layer A(H) := (current_substrate_time - H.promotion_event.committed_at) > N_A
```

`N_A` is recorded on the promotion event or on the hypothesis's concrete subtype per [`§0010`](../charter/decision-log.md) Q2-A.2; operational specification per follow-on RFC.

### Layer B — freshness_B (evidence-staleness)

```
freshness_B(H) =
    avg(
        evidential_independence(r)
        OVER recent N assertions r WHERE H.hash ∈ r.closure_hashes
    )
```

The recent-N window's structural form (fixed time, fixed count, hybrid) is deferred to operational specification. The `closure_hashes` field per the Influence Storage section above is what `H.hash ∈ r.closure_hashes` queries against.

### Layer B — saturation_C (influence-saturation)

```
saturation_C(H) =
    (count of recent N assertions r WHERE H.hash ∈ r.closure_hashes
       AND H.hash ∉ r.direct_influenced_by)
    /
    N
```

The `H.hash ∉ r.direct_influenced_by` clause is the L-C structural-exclusion commitment per [`§0135`](../charter/decision-log.md): assertions whose direct `influenced_by` lists H are H's OWN enrichment outputs (H is the proximate influence); excluding them prevents self-reinforcing saturation. Assertions where H appears in the transitive closure but NOT in the direct edges are downstream consumers — those are what saturation_C counts.

### Parameter ranges

The contract enforces type/range at marshalling:

```proto
message LayerBParameters {
    EvidentialIndependence t_b = 1;                  // 0 ≤ T_B ≤ 1; freshness threshold
    EvidentialIndependence k_c = 2;                  // 0 ≤ K_C ≤ 1; saturation ratio
    uint64 n_window = 3;                             // N > 0; recent-window size (assertion count)
    uint64 n_a_duration_nanoseconds = 4;             // N_A > 0; Layer A cadence (nanoseconds since promotion event commit)
}
```

`T_B` and `K_C` use the same rational encoding as `evidential_independence` (numerator/denominator pair). `N` is a positive integer (count of recent assertions per the W-count window form). `n_a_duration_nanoseconds` is a positive integer encoding the Layer A cadence as nanoseconds elapsed since the promotion event's `committed_at` substrate timestamp (per [`§0138`](../charter/decision-log.md) N_A bundling).

### Values (§0138 inception-phase)

Per [`§0138`](../charter/decision-log.md) Layer B parameter-calibration resolution, the inception-phase values are:

```proto
LayerBParameters {
    t_b: EvidentialIndependence { numerator: 1, denominator: 2 }   // 0.5
    k_c: EvidentialIndependence { numerator: 1, denominator: 2 }   // 0.5
    n_window: 1000
    n_a_duration_nanoseconds: 86400000000000                        // 1 day
}
```

Window structural form: W-count (fixed-count — last `n_window` assertions by substrate-commit order; clock-time-independent per [§2.1](../charter/constitutional-charter.md#21-observational-integrity) substrate-immutability). Per-subtype divergence: U-uniform (single LayerBParameters message at the abstract `Hypothesis` level; the four concrete Cat III subtypes — `BehavioralCluster`, `AutomationGroup`, `CampaignHypothesis`, `CoordinationRing` — share the same parameter set per [`§0010`](../charter/decision-log.md) Q2-A.2 + [`§0138`](../charter/decision-log.md) F4).

Per-parameter reversal-conditions record per [`§0138`](../charter/decision-log.md) F7: each parameter carries its own empirical-pressure-phase trigger; revision of one parameter does not require revisiting the others. The reversal-conditions record is the canonical reference for value revisions per [`§0022`](../charter/decision-log.md) discipline — observation-based, not hypothesis-based.

### Validation discipline

The predicate's firing is computed at projection time (projection-replay path), not at substrate-commit time — `DEMOTE-CANDIDATE(H)` is a structural test, not a stored value. The substrate stores the components (α values + closures + direct edges + promotion timestamps); projection replay computes the predicate. The §2.6 anti-pattern 2 detection (byte-for-byte projection-replay match) applies to each component independently.

When the predicate fires, the operator commits a `demotion` lifecycle event per [§2.5](../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) v0.3; the event references the promotion event and records the predicate's firing values for audit.

## Schemas-Evolution Events

Per [`§0024`](../charter/decision-log.md) AP5 step (c), library upgrades are schemas-evolution events. This section defines the boundary.

A change is a **schemas-evolution event** when any of the following hold:

1. The Protobuf library version (`google.golang.org/protobuf`) is changed to a version that may alter canonical serialization behavior — including default field encoding, unknown-fields handling, or `map`/`oneof` encoding. Patch-level upgrades (e.g. v1.36.1 → v1.36.2) are usually NOT schemas-evolution events; minor upgrades (e.g. v1.36 → v1.37) ARE evaluated as schemas-evolution events by inspecting the library changelog for canonical-serialization-affecting changes.
2. The BLAKE3 library version (`lukechampine.com/blake3`) is changed. Same patch-vs-minor distinction.
3. The Go toolchain version is changed in a way that affects the Protobuf or BLAKE3 library's compiled behavior (rare; usually only at major Go release boundaries).
4. The `protoc-gen-go` plugin version is changed in a way that affects generated-code marshalling behavior.
5. Any `.proto` file's structural commitments change (new required-by-discipline field; deprecation; field number reservation).
6. The α formula (Evidential Independence section) changes — numerator/denominator definition, reachability semantic, or rational encoding shape.
7. The closure_hashes encoding or computation algorithm (Influence Storage section) changes — hash element shape, sort order, deduplication, or merge algorithm.
8. The L-BC-OR predicate's structural form (Demotion-Candidacy Predicate section) changes — Layer A composition, Layer B inner predicate, or L-C structural-exclusion semantic.
9. The paired-dimension required-fields shape (Paired-Dimension Commitment section) changes — which record types are subject to the commitment, which fields are required, validation discipline.
10. The LayerBParameters proto field shape changes — adding/removing fields, changing type/range, or modifying the bundling discipline (e.g., un-bundling N_A from LayerBParameters back to a separate proto). Parameter VALUES per [`§0138`](../charter/decision-log.md) are operational; their revision per the per-parameter reversal-conditions record does NOT trigger a schemas-evolution event unless the proto field shape itself changes.

A change is NOT a schemas-evolution event when:

- A patch-level library upgrade has no canonical-serialization-affecting changes per its changelog.
- Service-tier code changes that do not touch the canonical-serialization or hashing call sites.
- Configuration changes unrelated to the marshalling pipeline.

The boundary distinction is enforceable mechanically by the CI golden-file gate (next section): a change that breaks the golden-file test is, by definition, a schemas-evolution event.

## CI Golden-File Gate

Per [`§0024`](../charter/decision-log.md) AP5 step (d), the CI golden-file gate is the mechanical predicate for detecting canonical-serialization regression.

### Specification

1. A **golden corpus** of representative message instances is maintained at `services/<service>/testdata/canonical-corpus/`. Each instance:
   - Has a stable name (e.g. `declared-session-minimal.json` for the human-readable form).
   - Has a paired canonical-bytes golden file (e.g. `declared-session-minimal.bin`) containing the expected canonical Protobuf serialization.
   - Has a paired content-hash golden file (e.g. `declared-session-minimal.hash`) containing the expected BLAKE3 32-byte digest in lowercase-hex form.

2. The corpus covers every canonical-form-load-bearing message type at least once, with at least one variant per `oneof` branch and at least one variant exercising every non-trivial field type.

3. CI runs a test that, for each corpus entry:
   - Constructs the Go message from the human-readable form.
   - Marshals via the canonical procedure (Serialization Stack section above).
   - Compares byte-for-byte against the canonical-bytes golden file.
   - Computes the BLAKE3 hash and compares hex-for-hex against the hash golden file.
   - Any mismatch FAILS the test.

4. A test failure indicates one of:
   - A library upgrade silently changed canonical serialization (schemas-evolution event per the boundary above).
   - A code change accidentally altered marshalling-pipeline behavior.
   - A golden file is stale relative to a deliberate schemas-evolution change (the golden files are regenerated as part of the schemas-evolution commit, with explicit reviewer attention).

5. Golden-file regeneration is an explicit operation, not an automatic one. The regeneration script is `services/<service>/testdata/regenerate-canonical-corpus.sh` (or equivalent); running it requires the change author to acknowledge that the canonical bytes are changing. The regenerated files are committed alongside the change that produces them, with a commit message naming the schemas-evolution event.

## Upgrade Discipline

When a schemas-evolution event is contemplated:

1. **Survey the change.** Read the upstream changelog (library or `.proto` file diff). Identify whether canonical serialization is affected.
2. **Predict golden-file divergence.** State explicitly which golden corpus entries are expected to change bytes and which are expected to remain stable.
3. **Run the upgrade locally.** Regenerate the corpus; inspect the diff against prediction.
4. **Reconcile prediction vs reality.** Unexpected divergence (a corpus entry's bytes changed when prediction said they wouldn't) is an indicator of either a library-changelog omission or an unexamined call-site dependency. Resolve before committing.
5. **Commit the change with the regenerated corpus.** Commit message identifies the schemas-evolution event explicitly. Service version markers (e.g. `go.mod`) are committed in the same commit as the regenerated golden files.
6. **Inform downstream consumers.** If the upgrade affects content-hash stability for historical records, downstream consumers of those hashes (replay paths; backup-recovery procedures per [`§0027`](../charter/decision-log.md) Proposal item 5) must be re-validated against the new hashes. In practice for inception phase, this is internal — there is one consumer (the ingestion service) and one operator (the project committee).

## Anti-Patterns

By analogy to Charter [§2.1 Forbidden Anti-Patterns](../charter/constitutional-charter.md#21-observational-integrity). Each is concrete and falsifiable.

- **Marshalling outside the canonical procedure.** Service code that calls `proto.Marshal(msg)` (without the `MarshalOptions` block) produces bytes that may not be canonical and whose hash may not be stable. Detectable: lint rule on `proto.Marshal(` call sites in canonical-form-load-bearing service code; only `(MarshalOptions{...}).Marshal(msg)` form permitted.
- **Hash computation against non-canonical bytes.** Computing a BLAKE3 hash against a byte slice produced by anything other than the canonical procedure produces a hash that is not the content-addressable identifier. Detectable: code review on all `blake3.Sum256(` call sites; bytes argument must trace to canonical marshal output.
- **Golden-file mismatch tolerated.** A CI golden-file failure that is dismissed (e.g. by commenting out the test, or by regenerating without inspection) defeats the gate. Detectable: code review on golden-corpus regeneration commits; every regeneration commit identifies the schemas-evolution event explicitly.
- **Encoding the hash in a non-canonical form in canonical-load-bearing contexts.** Mixing hex and base64 encodings of the same hash across the codebase introduces string-comparison bugs that do not surface until the comparison occurs. Detectable: lint rule on hex/base64 encoding call sites against hash values.
- **Paired-dimension bypass.** Committing a Cat II construct, Cat III hypothesis, or Assertion with `subject_ref_construct` / `subject_ref_hypothesis` populated WITHOUT both `confidence` AND `evidential_independence` fields. Forbidden by [§2.6](../charter/constitutional-charter.md#26-evidential-independence-integrity) v0.6 anti-pattern 1. Detectable at the canonical-marshalling boundary via `AllowPartial: false` rejection on missing required fields.
- **α computed offline / α not byte-stable.** Computing `evidential_independence` in a process other than the substrate at write time, or producing an α value that does not byte-for-byte match the recomputed value from substrate-committed provenance + influence subgraphs. Forbidden by [§2.6](../charter/constitutional-charter.md#26-evidential-independence-integrity) anti-pattern 5 (offline-only derivation) + anti-pattern 2 (projection-replay byte-for-byte match). Detectable at write-time substrate validation: substrate recomputes α and compares.
- **BLAKE3-hash-list element mis-encoded.** Any field subject to the Hash-List Field Discipline section above (`closure_hashes`, `direct_influenced_by`, `source_event_hashes`, `antecedent_formation_event_hashes`, `successor_formation_event_hashes`) containing elements that are not 32-byte BLAKE3 hashes, OR not in ascending lexicographic order, OR containing duplicates. Forbidden by [`§0139`](../charter/decision-log.md) (uniform discipline) + the proto comments at each field's site (each documents the ascending-order + content-hash-stability commitment). Detectable at the canonical-marshalling boundary via field-shape validation. Generalization of the inception-revision form named for `closure_hashes` only at [`§0136`](../charter/decision-log.md); supersedes that narrower form per [`§0139`](../charter/decision-log.md).
- **Cat II without closure-union transmission.** A Cat II construct whose `closure_hashes` is NOT the union of its inputs' `closure_hashes`. Forbidden by [`§0134`](../charter/decision-log.md) Cat II structural transmission commitment + [`§0136`](../charter/decision-log.md) operational discharge. Detectable at write-time substrate validation: substrate recomputes the union and compares.
- **α encoded as float.** Encoding `evidential_independence` as a floating-point value rather than as a `EvidentialIndependence` rational-pair message. Forbidden by [`§0136`](../charter/decision-log.md) (the rational encoding preserves the bounded-resolution structural commitment per [`§0133`](../charter/decision-log.md) Phase 4 Finding 4). Detectable at proto type definition review + code review.
- **Layer B saturation_C without L-C structural-exclusion.** A `saturation_C(H)` computation that does NOT exclude assertions where H appears in `direct_influenced_by` from the numerator. Forbidden by [`§0135`](../charter/decision-log.md) L-C structural-exclusion committee extension + [`§0136`](../charter/decision-log.md) operational discharge. Detectable at projection-replay validation of the demotion-event firing conditions.

## Open Questions

- **Audit-log of schemas-evolution events.** A separate registry document (or section of an ops document) recording each schemas-evolution event with date + reviewer + summary may be valuable as the project's schemas-evolution history grows. Not bundled here; consider when the schemas-evolution-discipline RFC (per [`§0024`](../charter/decision-log.md) Open Questions) is opened.
- **Corpus coverage policy.** This document specifies "every canonical-form-load-bearing message type at least once" but does not codify the exhaustiveness predicate (e.g. every `oneof` branch + every field-type-variant). Concrete coverage policy deferred to the first service-tier work.
- **Hash-collision handling protocol.** Per [`§0027`](../charter/decision-log.md) AP6 (apparent-duplicate-write byte-equality verification), a BLAKE3 collision in practice indicates the algorithm is broken or the canonical-serialization contract is violated. The protocol for handling such an event (alert, freeze, investigate, recover) is operational; deferred to the operational-ops document referenced in [`§0027`](../charter/decision-log.md) Open Questions.

## References

- [`docs/charter/constitutional-charter.md` §2.1](../charter/constitutional-charter.md#21-observational-integrity) — the invariant this contract operationalizes for substrate-immutability.
- [`docs/charter/constitutional-charter.md` §2.4](../charter/constitutional-charter.md#24-inferential-influence-disclosure) v0.5 — `influenced_by` chain declaration; this contract's Influence Storage section operationalizes.
- [`docs/charter/constitutional-charter.md` §2.5](../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) v0.3 — `demotion` lifecycle event with designated structural test; this contract's Demotion-Candidacy Predicate section operationalizes.
- [`docs/charter/constitutional-charter.md` §2.6](../charter/constitutional-charter.md#26-evidential-independence-integrity) v0.6 — paired-dimension commitment; this contract's Paired-Dimension Commitment + Evidential Independence sections operationalize.
- [`docs/charter/decision-log.md` §0024](../charter/decision-log.md) — schemas-technology selection (Protobuf proto3); AP5 mitigation steps (a), (b), (c), (d); AP6 (`map<K, V>` ban).
- [`docs/charter/decision-log.md` §0025](../charter/decision-log.md) — implementation-language selection (Go); library-version-pinning discipline.
- [`docs/charter/decision-log.md` §0027](../charter/decision-log.md) — storage-technology selection (SQLite + blob-store); AP6 apparent-duplicate-write byte-equality.
- [`docs/charter/decision-log.md` §0028](../charter/decision-log.md) — introduction of this document; version-pinning policy + CI gate operationalization recorded.
- [`docs/charter/decision-log.md` §0133](../charter/decision-log.md) — Q3-α resolution (source-count ratio); this contract's Evidential Independence section operationalizes.
- [`docs/charter/decision-log.md` §0134](../charter/decision-log.md) — Q5-τ resolution (transitive closure + β-graph storage + Cat II structural transmission); this contract's Influence Storage section operationalizes.
- [`docs/charter/decision-log.md` §0135](../charter/decision-log.md) — Layer B L-BC-OR resolution (disjunctive + L-C structural-exclusion); this contract's Demotion-Candidacy Predicate section operationalizes.
- [`docs/charter/decision-log.md` §0136](../charter/decision-log.md) — first contract revision consolidating §0133 + §0134 + §0135 at the contract layer.
- [`docs/charter/decision-log.md` §0137](../charter/decision-log.md) — Charter v0.7.1 patch amendment correcting §2.6 + §3 `§0034` → `§0136` stale anchors (closes §0136 Anchor-fidelity observation).
- [`docs/charter/decision-log.md` §0138](../charter/decision-log.md) — Layer B parameter-calibration resolution; LayerBParameters proto extension (n_a_duration_nanoseconds field) + inception-phase values fixed.
- [`docs/charter/decision-log.md` §0139](../charter/decision-log.md) — Hash-list field discipline generalization; the inception-revision element-shape rule for `closure_hashes` + `direct_influenced_by` at [`§0136`](../charter/decision-log.md) extends to the full set of canonical-form-load-bearing 32-byte BLAKE3 `repeated bytes` fields (adding `source_event_hashes`, `antecedent_formation_event_hashes`, `successor_formation_event_hashes`). This contract's Hash-List Field Discipline + Anti-Patterns sections operationalize.
- [`docs/rfcs/draft/architecture-schemas-technology-selection.md`](../rfcs/draft/architecture-schemas-technology-selection.md) — accepted at [`§0024`](../charter/decision-log.md).
- [`docs/rfcs/draft/architecture-implementation-language-selection.md`](../rfcs/draft/architecture-implementation-language-selection.md) — accepted at [`§0025`](../charter/decision-log.md).
- [`docs/rfcs/draft/architecture-storage-technology-selection.md`](../rfcs/draft/architecture-storage-technology-selection.md) — accepted at [`§0027`](../charter/decision-log.md).
- [`docs/rfcs/draft/ontology-revision-q3-independence.md`](../rfcs/draft/ontology-revision-q3-independence.md) — Q3-α RFC; accepted at [`§0133`](../charter/decision-log.md).
- [`docs/rfcs/draft/ontology-revision-q5-influence-propagation-transitivity.md`](../rfcs/draft/ontology-revision-q5-influence-propagation-transitivity.md) — Q5-τ RFC; accepted at [`§0134`](../charter/decision-log.md).
- [`docs/rfcs/draft/ontology-revision-layer-b-deep-criterion.md`](../rfcs/draft/ontology-revision-layer-b-deep-criterion.md) — Layer B RFC; accepted at [`§0135`](../charter/decision-log.md).
- [`docs/rfcs/draft/operational-spec-layer-b-parameter-calibration.md`](../rfcs/draft/operational-spec-layer-b-parameter-calibration.md) — Layer B parameter-calibration RFC; accepted at [`§0138`](../charter/decision-log.md). First operational-spec RFC.
- [`docs/architecture/storage-model.md`](./storage-model.md) — Tier 0 substrate (which this contract's bytes inhabit) and Tier 1 archive (which inherits the contract).
