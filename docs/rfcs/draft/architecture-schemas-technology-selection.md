# RFC — Architecture: Schemas technology selection

- **Status:** accepted
- **Authors:** committee
- **Date:** 2026-05-19
- **Type:** architecture
- **Affects:** [`schemas/`](../../../schemas/) (technology commitment for substrate type definitions); [`services/`](../../../services/) (downstream consumers of generated bindings); [`docs/architecture/`](../../architecture/) (future architecture documents inherit the choice); [`docs/charter/decision-log.md` §0003](../../charter/decision-log.md) (deferral predicate for schemas-technology selection is satisfied per §0022); [`docs/charter/decision-log.md` §0022](../../charter/decision-log.md) (authorizing pivot entry); [Charter §2.1](../../charter/constitutional-charter.md#21-observational-integrity), [§2.2](../../charter/constitutional-charter.md#22-epistemic-separation), [§2.3](../../charter/constitutional-charter.md#23-provenance-integrity) (frozen — selection respects, does not modify).

> RFCs in Ghost Trace are proposals subject to constitutional review. The [Constitutional Charter](../../charter/constitutional-charter.md) is the authoritative reference. RFCs that conflict with the Charter must either be rejected or accompanied by a corresponding charter amendment (see [`../../charter/amendments.md`](../../charter/amendments.md)).

---

## Summary

**Draft position (to be tested in discussion phase, not accepted by inclusion here):** select **Protocol Buffers (proto3)** as the schemas technology for Ghost Trace's substrate. Binary serialization with deterministic canonical wire format under proto3's explicit field ordering rules; native `oneof` primitive supporting the §2.3 / Q3 §0016 subject_ref exclusivity contract; multi-language code generation via `protoc`; mature tooling ecosystem.

Selection authorized by [`decision-log.md` §0022](../../charter/decision-log.md) (storage-technology deferral [`§0003`](../../charter/decision-log.md) partially reversed; empirical threshold reached — four frozen object-level invariants + four resolved Ontology full-cycle questions). This RFC carries the draft position into discussion phase; resolution phase produces the final commitment.

## Motivation

The schemas-technology selection is gating for any concrete schema-level work under [`§0022`](../../charter/decision-log.md). Without selection, the ingestion service skeleton and the Cat I / Cat II / Cat III type definitions cannot proceed. The four frozen object-level invariants impose distinct technology constraints:

- **[§2.1 Observational Integrity](../../charter/constitutional-charter.md#21-observational-integrity) (frozen)** requires content-addressable identifiers, which require deterministic serialization. Protobuf proto3 specifies a canonical wire format with explicit field ordering; serialization is bit-stable across implementations sharing the same descriptors. Hash-based content-addressing is structurally supported.

- **[§2.2 Epistemic Separation](../../charter/constitutional-charter.md#22-epistemic-separation) (frozen)** requires schema-level nominally-distinct categorical types. Protobuf message types are nominally distinct by construction; no shared discriminator field exists at the protocol layer. Each Category I, Category II, and Category III gets its own message type.

- **[§2.3 Provenance Integrity](../../charter/constitutional-charter.md#23-provenance-integrity) (frozen v0.4)** requires oneOf/union exclusivity on the Assertion `subject_ref_*` fields per [`§0016`](../../charter/decision-log.md) (Q3 resolution). Protobuf's `oneof` primitive enforces exactly-one-populated semantics at the schemas layer. No application-level discriminator-check workaround is required.

- **[§2.5 Hypothesis Lifecycle Explicitness](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) (frozen v0.3)** requires no schemas-technology-specific support beyond §2.1 / §2.2 inheritance.

- **[§2.4 Inferential Influence Disclosure](../../charter/constitutional-charter.md#24-inferential-influence-disclosure) + [§2.6 Evidential Independence Integrity](../../charter/constitutional-charter.md#26-evidential-independence-integrity) (pending — empirical pressure phase per [`§0022`](../../charter/decision-log.md))**: Protobuf is extensible via field-number reservation and optional fields, allowing forward-compatible additions for `influenced_by` edges (§2.4) and confidence / independence pairing (§2.6) when those invariants redact.

The cost of not making this selection (technology dimension): no `.proto` (or `.avsc`, `.fbs`) files can exist, and ingestion service work halts. The procedural cost (Charter-governance dimension): the empirical-evidence-from-implementation feedback loop that §2.4 + §2.6 redaction depends on per [`§0022`](../../charter/decision-log.md) cannot begin.

## Constitutional Review

Verbatim output of the rfc-author §1 pre-authorship analysis (Q1–Q6).

### Q1 — Charter invariants touched

- **§2.1 Observational Integrity (frozen):** schemas technology must support content-addressable identifiers via deterministic serialization. Selection respects, does not violate.
- **§2.2 Epistemic Separation (frozen):** schemas technology must support nominally-distinct categorical types (no shared discriminator). Selection respects, does not violate. Anti-pattern surfaced (see §Anti-Patterns below): technology-native escape hatches (e.g. Protobuf `Any`) that collapse the categorical distinction.
- **§2.3 Provenance Integrity (frozen v0.4):** schemas technology must support `subject_ref_*` oneOf/union exclusivity per [`§0016`](../../charter/decision-log.md). Selection respects, does not violate.
- **§2.5 Hypothesis Lifecycle Explicitness (frozen v0.3):** no schemas-technology-specific requirement beyond §2.1 / §2.2 inheritance.
- **§2.4 + §2.6 (pending — empirical pressure phase):** not directly touched. Schemas technology must be extensible enough to accommodate future `influenced_by` edges (§2.4) and confidence / independence pairing (§2.6) when those sections redact.

### Q2 — Glossary redefinition

No. Technology-specific terminology (Protobuf, proto3, `oneof`, content-addressing, hash function) is technology vocabulary, not canonical project vocabulary. Recommend do NOT add tech terms to [`docs/glossary.md`](../../glossary.md). The canonical project vocabulary (substrate, observation, operational construct, hypothesis, provenance, etc. per [`CLAUDE.md` §3](../../../.claude/CLAUDE.md)) is unchanged.

### Q3 — Implicit resolution of open Ontology questions

None. The five [`ontology.md`](../../ontology/ontology.md) §Open Questions for Committee Resolution remaining post-§0023: Q1, Q2, Q4 resolved per [`§0015`](../../charter/decision-log.md) / [`§0010`](../../charter/decision-log.md) / [`§0011`](../../charter/decision-log.md); Q3 (independence formal definition — distinct from §2.3 Q3 subject_ref polymorphism) and Q5 (transitive / decaying influence — "decaying" half closed at [`§0020`](../../charter/decision-log.md) / [`§0021`](../../charter/decision-log.md), "transitive" half open) are not touched by schemas-technology selection. [`entity-model.md`](../../ontology/entity-model.md) §Open Modeling Questions is drained post-[`§0023`](../../charter/decision-log.md). [`provenance-model.md`](../../ontology/provenance-model.md) OMQ #1 (Granularity) and OMQ #4 (Cross-domain) not touched.

### Q4 — Charter amendment required

No. The pivot at [`§0022`](../../charter/decision-log.md) explicitly authorized this RFC under ordinary discipline. No frozen Charter prose requires modification.

### Q5 — New invariant introduced

No. Technology selection. The proposed commitments — deterministic serialization, content-addressing, schemas-versioning discipline — are downstream consequences of existing §2.1 / §2.2 / §2.3 invariants, not new invariants.

### Q6 — Ceremony without behavioral consequence

No. Selection is gating for all concrete schema-level work. Falsifiable by deletion: without it, no `.proto` / `.avsc` / `.fbs` files can exist; downstream service work halts.

## Proposal

**Draft position (to be tested in discussion phase):** adopt **Protocol Buffers (proto3, latest stable release)** as the schemas technology for the [`schemas/`](../../../schemas/) subtree. Concrete commitments:

1. **All categorical types** (Category I primary observations, Category II operational constructs, Category III hypotheses, Assertion subject references, provenance edges) defined as distinct Protobuf message types — one type per category-instance, no shared discriminator.
2. **Content-addressing** computed via BLAKE3 hash of canonical Protobuf serialization. Hash is part of message identity established at serialization time, not derived at read time.
3. **`oneof`** used for [`§0016`](../../charter/decision-log.md) (Q3 resolution) `subject_ref_*` exclusivity on the Assertion entity.
4. **Code generation:** `protoc-gen-go` for Go bindings (subject to the implementation-language RFC); other languages added via standard `protoc` plugins as the topology grows.
5. **Evolution discipline:** Protobuf field-number reservation enforced; never reuse retired field numbers; never change field types in incompatible ways. Evolution policy detail deferred to a follow-on schemas-evolution-discipline RFC.
6. **Time-typed fields.** Canonical-form-load-bearing time fields use `int64` Unix nanoseconds, not `google.protobuf.Timestamp`. Rationale: `int64` is bit-stable across all language bindings (no parsing layer between wire bytes and value); `Timestamp` carries nominal-type ergonomics at the cost of cross-binding parsing variability that can affect content-hash stability. `Timestamp` is reserved for non-canonical-form / projection-only fields.
7. **Toolchain.** Adopt `buf` (the Buf CLI) for linting, breaking-change detection, and formatting. AP3 (field-number reuse forbidden) becomes a CI gate via `buf breaking` against a baseline ref, converting AP3 from a discipline obligation to mechanically enforced. Code-generation invocation (`protoc` directly vs `buf generate`) is deferred to the implementation-language RFC's toolchain decision.

## Alternatives Considered

Four alternatives evaluated. Three rejected as admissible-but-deferred (revisit conditions registered in Decision Record below as R-tech-1 through R-tech-4); one rejected on canonical-serialization-fragility grounds. Discussion-phase re-test recorded in [`schemas-technology-selection-evidence.md`](../discussion/schemas-technology-selection-evidence.md) Phase 2.

- **Apache Avro.** Rejected. The single-object encoding mode (Avro spec, c. 2014) allows registry-free use, weakening the original "registry coupling" framing the draft asserted. The structural objection that holds: Avro's typed-union mechanism (`union { Foo, Bar }`) requires descriptor consultation at decode time to identify which type a given payload carries, where Protobuf's `oneof` encodes the discriminator in the wire format. For §2.3 oneOf exclusivity, Protobuf's coupling between wire format and discriminator aligns more directly with §2.2's nominally-distinct-types principle. Revisit conditions: R-tech-2 (categorical type Protobuf cannot express) or R-tech-4 (schemas-evolution pattern field-number-reservation cannot accommodate).

- **JSON-Schema-validated JSON serialization.** Rejected on canonical-serialization-fragility grounds (not deferred). JSON has no canonical serialization at the text layer per RFC 8259; key ordering, whitespace, and number representation are implementation-defined. RFC 8785 (JSON Canonicalization Scheme, 2020) defines a deterministic canonical form but adoption requires both producer and consumer to implement JCS faithfully and JCS does not cover all JSON-Schema-validated payloads cleanly (number-format edge cases). The schemas-validation layer is structurally separate from the serialization layer (validators run on parsed JSON; coupling is by convention). Protobuf collapses both layers into one canonical-by-construction tool. Revisit condition: R-tech-3 (canonical-serialization fragility under pinned-library-version + canonical-serialization-contract mitigation per AP5 — two empirically observed hash-divergence incidents required).

- **Cap'n Proto.** Rejected as admissible-but-deferred. Cap'n Proto's wire format is deterministic and its packed encoding is canonical; its `union` mechanism is structurally analogous to Protobuf's `oneof`. The arena-allocation and zero-copy properties are not required by any frozen invariant; inception-phase operational dependencies are minimized in favor of mature toolchain ecosystem. Revisit condition: R-tech-1 (throughput characterization at production-relevant scale demonstrates serialize+deserialize cost dominates ingestion path beyond a threshold).

- **FlatBuffers.** Rejected as admissible-but-deferred (same rejection-shape as Cap'n Proto). FlatBuffers' read-zero-copy design is real and well-documented; canonical serialization is deterministic modulo offset-table alignment choices. The read-zero-copy properties are not required by any frozen invariant; inception-phase work prioritizes operational simplicity over speculative read-path optimization for access patterns not yet characterized. Revisit condition: R-tech-1 (throughput characterization demands zero-copy reads).

The admissible-but-deferred registrations (Cap'n Proto + FlatBuffers) follow the OMQ #2-2 B-substrate pattern per [`§0020`](../../charter/decision-log.md) — admissibility-preserved-with-revisit-threshold rather than structural rejection.

## Open Questions

The RFC explicitly defers:

- **Specific Protobuf library-version pin.** The library version that produces canonical bytes is pinned per the implementation-language RFC's toolchain commitment; the exact version is decided there, not here.
- **Schemas-evolution policy beyond "never reuse field numbers."** A dedicated schemas-evolution-discipline RFC is follow-on work; not bundled here. The follow-on RFC will codify (a) breaking-change classification, (b) deprecation procedure, (c) the canonical-serialization contract maintenance (per AP5 mitigation).
- **Build-time vs commit-time code generation.** Standard practice is build-time; the final pinning is in the implementation-language RFC's toolchain decision.
- **Cross-RFC coupling with implementation-language RFC.** Protobuf's mature code-generation ecosystem covers many languages but not all; the implementation-language RFC (second of three technology RFCs per [`§0022`](../../charter/decision-log.md)) inherits Protobuf as a constraint — the chosen language must have a maintained `protoc` plugin AND a faithful proto3 wire-format implementation (per AP5 — canonical-serialization bit-stability requirement). The implementation-language RFC's option space is constrained by this selection without itself being an explicit Charter commitment. Surface explicitly to the implementation-language RFC's discussion phase.

## Anti-Patterns to Avoid

By analogy to Charter [§2.1](../../charter/constitutional-charter.md#21-observational-integrity) and [§2.2](../../charter/constitutional-charter.md#22-epistemic-separation) `Forbidden Anti-Patterns` sections. Each is concrete and falsifiable.

- **Using `google.protobuf.Any` to defeat §2.2 categorical separation.** The `Any` type carries an arbitrary serialized message plus a type URL. Using `Any` in a field that should carry a categorically-typed message collapses the §2.2 distinction by allowing any category to inhabit any reference. `Any` is permitted only at boundary layers (e.g. transport-level envelopes); never inside Category I / II / III message bodies. Detectable mechanically: grep generated descriptors for `google.protobuf.Any` field usage inside categorical message types.

- **Using `oneof` to model the rejected unified-record-with-discriminator pattern.** Q3 [`§0016`](../../charter/decision-log.md) uses `oneof` to enforce exclusivity among categorically-distinct `subject_ref_*` fields. This is structurally different from a unified record with a type discriminator (rejected per [§2.2 Forbidden Anti-Patterns](../../charter/constitutional-charter.md#22-epistemic-separation)). The branches of a permitted `oneof` are categorically-distinct reference fields; the branches of a forbidden `oneof` would be tagged variants of one record type. Detectable: review every `oneof` declaration; confirm each branch resolves to a categorically-distinct typed reference.

- **Reusing field numbers from retired fields.** Protobuf field numbers are part of the wire format. Reusing a retired field number creates silent semantic conflation across schemas versions, violating §2.1's reconstructibility guarantee for historical records. Detectable: enforce the `reserved` clause for retired numbers; CI check on every `.proto` PR.

- **Generating code into the substrate.** Generated code (e.g. `*.pb.go` files from `protoc-gen-go`) is build output, not source. Commit the `.proto` source; generate the language-specific bindings at build time. Generated code in version control creates synchronization risk between the schemas layer and the bindings. Detectable: `.gitignore` covers generated artifacts; CI check that no generated bindings are committed.

- **Hash-instability via tooling-version drift.** Content-hash computation depends on canonical serialization; canonical serialization depends on the Protobuf library implementation. A library version bump that changes default field-ordering behavior, unknown-fields handling, or default-value serialization rules silently invalidates historical content-hashes and violates §2.1 reconstructibility. Mitigation procedure: (a) pin the Protobuf library version per the implementation-language RFC's toolchain commitment; (b) document the canonical-serialization contract in a separate architecture document specifying the proto3 spec version + the library version that produces the canonical bytes; (c) treat library upgrades as schemas-evolution events governed by the follow-on schemas-evolution-discipline RFC; (d) CI gate on canonical-serialization regression — golden-file test comparing serialized bytes for a representative message set across versions. Detectable: golden-file test diverges; content-hash recomputation on read fails for historical records.

- **Use of `map<K, V>` in canonical-form-load-bearing message types.** proto3's `map<K, V>` field type does not have canonical iteration order across all language bindings; serialization of a map field can produce different byte sequences for the same logical content across implementations or even across runs of the same implementation. Use of `map<K, V>` in canonical-form-load-bearing message types breaks content-hash stability and violates §2.1 reconstructibility. Mitigation: encode key-value collections as `repeated SubMessage { key, value }` with explicit sort-order convention (ascending by key field, ties broken by value). Detectable: grep `.proto` files for `map<` declarations in categorical-message-type definitions; reject in CI gate.

## Migration and Backward Compatibility

**Inception phase.** No historical schemas exist. Forward-looking decision. Subsequent commits (separate from this RFC; gated on this RFC's acceptance) will create the initial `.proto` files for Cat I / Cat II / Cat III + Assertion + provenance edges per [`§0022`](../../charter/decision-log.md) Consequences. No replay-model migration applies because no prior substrate state exists.

**Schemas-technology reversal cost (forward-looking).** Forward-compatibility properties (Protobuf field-number reservation) protect against intra-technology evolution but do not protect against inter-technology migration. Switching from Protobuf to a different schemas technology (Cap'n Proto, FlatBuffers, Avro, JSON-Schema-validated JSON) post-substrate-commit requires re-serialization of all historical Cat I records, which is a §2.1-load-bearing operation: historical bytes must be regenerable under the new technology, which conflicts with content-addressing if hashes are part of substrate references. Practical implication: the inception-phase selection has more weight than the "admissible-but-deferred" framing in Alternatives suggests. The reversal conditions R-tech-1 through R-tech-4 in the Decision Record characterize the threshold for accepting the migration cost.

## References

- [`docs/charter/constitutional-charter.md`](../../charter/constitutional-charter.md) §2.1, §2.2, §2.3 (frozen v0.4), §2.5 (frozen v0.3). §2.4 + §2.6 pending — empirical pressure phase.
- [`docs/charter/decision-log.md`](../../charter/decision-log.md) [`§0003`](../../charter/decision-log.md) (partial-reversal authority for schemas / language / storage technology), [`§0010`](../../charter/decision-log.md) (Q2 hypothesis subtypes), [`§0015`](../../charter/decision-log.md) (Q1 session duality), [`§0016`](../../charter/decision-log.md) (Q3 subject_ref polymorphism), [`§0020`](../../charter/decision-log.md) (OMQ #2 decay of influence), [`§0021`](../../charter/decision-log.md) (OMQ #3 influence at projection vs substrate), [`§0022`](../../charter/decision-log.md) (pivot authorization), [`§0023`](../../charter/decision-log.md) (Q2 Identity tiers resolution per pivot).
- [`docs/ontology/entity-model.md`](../../ontology/entity-model.md) Assertion entity post-§0016; Cat I / II / III sections post-§0015 / §0023.
- [`docs/ontology/provenance-model.md`](../../ontology/provenance-model.md) post-OMQ #2-C + OMQ #3-α.
- [`.claude/CLAUDE.md`](../../../.claude/CLAUDE.md) §6.4 (implementation gate, amended at v0.4.2 per [`§0022`](../../charter/decision-log.md)).
- [`docs/rfcs/discussion/schemas-technology-selection-evidence.md`](../discussion/schemas-technology-selection-evidence.md) — discussion-phase evidence file (six numbered findings, recommendation with eight proposed modifications, four reversal conditions).

## Decision Record

Resolved at [`decision-log §0024`](../../charter/decision-log.md): **Protocol Buffers (proto3)** adopted as schemas technology for the [`schemas/`](../../../schemas/) subtree. The committee adopted the discussion-phase recommendation with the eight resolution-phase modifications enacted in this commit (Alternatives FC5/FC6/FC7 operationalized; ES5 Motivation split; AP5 expanded with concrete mitigation procedure; AP6 added — `map<K, V>` ban; F1 added to Open Questions — implementation-language RFC coupling; F4 resolved in Proposal — `int64` Unix nanoseconds for canonical time fields; F5 resolved in Proposal — `buf` adopted as toolchain; F6 added to Migration — schemas-technology reversal cost) plus two committee extensions.

### Committee extensions

1. **Cap'n Proto + FlatBuffers admissible-but-deferred registration.** Per [`§0020`](../../charter/decision-log.md) OMQ #2-2 B-substrate precedent and [`§0021`](../../charter/decision-log.md) OMQ #3-2 β precedent. Both alternatives are structurally admissible (deterministic canonical serialization; union mechanisms structurally analogous to Protobuf's `oneof`); dominated at inception phase by Protobuf on mature-toolchain ecosystem, operational-simplicity, and §2.3 oneOf-as-wire-format-primitive alignment. Pattern: third + fourth instance of admissible-but-deferred registration after OMQ #2-2 / OMQ #3-2; the pattern is now established for non-ontology RFCs (technology selection) in addition to ontology resolutions. Methodologically distinct from JSON-Schema-validated-JSON rejection (rejected on canonical-serialization-fragility — structural concern, not deferred).

2. **RFC procedural-divergence finding (informational, not gated on this RFC's acceptance).** Discussion-phase work surfaced two divergences between [`docs/rfcs/README.md`](../README.md) + [`rfc-author` skill §4](../../../.claude/skills/workflow/rfc-author/SKILL.md) and actual practice: (a) README claims "No RFCs have yet been accepted" which is stale ([`§0007`](../../charter/decision-log.md), [`§0010`](../../charter/decision-log.md), [`§0015`](../../charter/decision-log.md), [`§0016`](../../charter/decision-log.md), [`§0020`](../../charter/decision-log.md), [`§0021`](../../charter/decision-log.md) reference accepted RFCs); (b) rfc-author §4 specifies that accepted RFCs are renumbered and moved out of `docs/rfcs/draft/`, but actual practice leaves accepted RFCs in `draft/` with only the `Status:` field updated. This RFC follows actual practice (status `discussion` → `accepted`; file remains in `docs/rfcs/draft/`; no renumbering applied). The README + rfc-author §4 divergences are surfaced for committee resolution as separate follow-on work; not bundled with this RFC's acceptance.

### Reversal conditions

The selection stands subject to four named reversal conditions per [`schemas-technology-selection-evidence.md`](../discussion/schemas-technology-selection-evidence.md) Phase 5. Any single condition firing triggers a follow-on RFC reconsidering the selection.

- **R-tech-1 — Throughput pressure.** Characterization at production-relevant scale demonstrates that proto3 serialize+deserialize cost dominates the ingest path. Threshold: serialize+deserialize >40% of single-message ingestion CPU at characterized load. (Reconsider Cap'n Proto, FlatBuffers.)
- **R-tech-2 — Inexpressible categorical type.** A categorical type emerges (e.g. sum-types with structural subtyping) that Protobuf cannot encode cleanly. Threshold: explicit RFC proposing the type and demonstrating Protobuf cannot encode it cleanly. (Reconsider Avro union types.)
- **R-tech-3 — Canonical-serialization fragility.** Pinned-library-version + canonical-serialization-contract (per AP5 mitigation) prove insufficient — two empirically observed hash-divergence incidents across library versions. (Reconsider JSON-Schema-validated JSON with JCS.)
- **R-tech-4 — Schemas-evolution pattern.** A breaking-change pattern emerges that Protobuf's field-number-reservation discipline cannot accommodate. Threshold: explicit RFC documenting the pattern and the field-number-reservation failure mode. (Reconsider Avro registry-mediated evolution.)

No reversal condition fires at acceptance.
