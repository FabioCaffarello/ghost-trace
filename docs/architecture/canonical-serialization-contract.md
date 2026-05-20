# Canonical Serialization Contract

**Status:** Active. First non-scaffold architecture document. Discharges the follow-on commitment named in [`decision-log §0024`](../charter/decision-log.md) AP5 mitigation step (b) and [`§0027`](../charter/decision-log.md) Consequences.

> This document specifies the canonical-serialization contract for Ghost Trace: the bit-stable mapping from a Protobuf message instance to a byte sequence to a content-addressable identifier. The mapping is the falsifiability predicate for [Charter §2.1 Observational Integrity](../charter/constitutional-charter.md#21-observational-integrity) at the substrate; without bit-stability, content-hash recomputation on read cannot serve as the mutation-detection mechanism.

## Constitutional Anchors

- [Charter §2.1 Observational Integrity](../charter/constitutional-charter.md#21-observational-integrity) — requires content-addressable identifiers sufficient to detect mutation if attempted. The contract specified here is the operational mechanism.
- [`decision-log §0024`](../charter/decision-log.md) — schemas-technology selection: Protocol Buffers (proto3). The canonical-serialization layer this document specifies operates against proto3-generated message types.
- [`decision-log §0025`](../charter/decision-log.md) — implementation-language selection: Go. The Protobuf library binding specified here is the Go binding (`google.golang.org/protobuf` v2 module).
- [`decision-log §0027`](../charter/decision-log.md) — storage-technology selection: SQLite + content-addressed blob-store. The blob-store's content addressing operates against the canonical bytes specified here.

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

## Schemas-Evolution Events

Per [`§0024`](../charter/decision-log.md) AP5 step (c), library upgrades are schemas-evolution events. This section defines the boundary.

A change is a **schemas-evolution event** when any of the following hold:

1. The Protobuf library version (`google.golang.org/protobuf`) is changed to a version that may alter canonical serialization behavior — including default field encoding, unknown-fields handling, or `map`/`oneof` encoding. Patch-level upgrades (e.g. v1.36.1 → v1.36.2) are usually NOT schemas-evolution events; minor upgrades (e.g. v1.36 → v1.37) ARE evaluated as schemas-evolution events by inspecting the library changelog for canonical-serialization-affecting changes.
2. The BLAKE3 library version (`lukechampine.com/blake3`) is changed. Same patch-vs-minor distinction.
3. The Go toolchain version is changed in a way that affects the Protobuf or BLAKE3 library's compiled behavior (rare; usually only at major Go release boundaries).
4. The `protoc-gen-go` plugin version is changed in a way that affects generated-code marshalling behavior.
5. Any `.proto` file's structural commitments change (new required-by-discipline field; deprecation; field number reservation).

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

## Open Questions

- **Audit-log of schemas-evolution events.** A separate registry document (or section of an ops document) recording each schemas-evolution event with date + reviewer + summary may be valuable as the project's schemas-evolution history grows. Not bundled here; consider when the schemas-evolution-discipline RFC (per [`§0024`](../charter/decision-log.md) Open Questions) is opened.
- **Corpus coverage policy.** This document specifies "every canonical-form-load-bearing message type at least once" but does not codify the exhaustiveness predicate (e.g. every `oneof` branch + every field-type-variant). Concrete coverage policy deferred to the first service-tier work.
- **Hash-collision handling protocol.** Per [`§0027`](../charter/decision-log.md) AP6 (apparent-duplicate-write byte-equality verification), a BLAKE3 collision in practice indicates the algorithm is broken or the canonical-serialization contract is violated. The protocol for handling such an event (alert, freeze, investigate, recover) is operational; deferred to the operational-ops document referenced in [`§0027`](../charter/decision-log.md) Open Questions.

## References

- [`docs/charter/constitutional-charter.md` §2.1](../charter/constitutional-charter.md#21-observational-integrity) — the invariant this contract operationalizes.
- [`docs/charter/decision-log.md` §0024](../charter/decision-log.md) — schemas-technology selection (Protobuf proto3); AP5 mitigation steps (a), (b), (c), (d); AP6 (`map<K, V>` ban).
- [`docs/charter/decision-log.md` §0025](../charter/decision-log.md) — implementation-language selection (Go); library-version-pinning discipline.
- [`docs/charter/decision-log.md` §0027](../charter/decision-log.md) — storage-technology selection (SQLite + blob-store); AP6 apparent-duplicate-write byte-equality.
- [`docs/charter/decision-log.md` §0028](../charter/decision-log.md) — introduction of this document; version-pinning policy + CI gate operationalization recorded.
- [`docs/rfcs/draft/architecture-schemas-technology-selection.md`](../rfcs/draft/architecture-schemas-technology-selection.md) — accepted at [`§0024`](../charter/decision-log.md).
- [`docs/rfcs/draft/architecture-implementation-language-selection.md`](../rfcs/draft/architecture-implementation-language-selection.md) — accepted at [`§0025`](../charter/decision-log.md).
- [`docs/rfcs/draft/architecture-storage-technology-selection.md`](../rfcs/draft/architecture-storage-technology-selection.md) — accepted at [`§0027`](../charter/decision-log.md).
- [`docs/architecture/storage-model.md`](./storage-model.md) — Tier 0 substrate (which this contract's bytes inhabit) and Tier 1 archive (which inherits the contract).
