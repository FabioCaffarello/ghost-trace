# RFC — Architecture: Schemas technology selection

- **Status:** draft
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

The cost of not making this selection: no `.proto` (or `.avsc`, `.fbs`) files can exist; ingestion service work halts; the empirical-evidence-from-implementation feedback loop §2.4 / §2.6 redaction depends on cannot begin.

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

## Alternatives Considered

**Draft positions (to be tested in discussion phase):**

- **Apache Avro.** Rejected at inception. Schemas-registry coupling adds an operational dependency at inception phase. Union types support §2.3 oneOf semantics but require runtime descriptor lookup, weakening the static categorical separation §2.2 codifies. Avro's strength is dynamic schemas evolution under registry mediation, which is exactly what §2.1 substrate-immutability does not benefit from.

- **JSON-Schema-validated JSON serialization.** Rejected. JSON has no canonical serialization at the text layer; key ordering, whitespace, and number representation are implementation-defined. Content-addressing via JSON serialization is fragile across implementations and library versions. Workarounds (canonical-JSON specifications such as RFC 8785 JCS) reintroduce the canonical-form discipline Protobuf provides natively, plus the schemas-validation layer is structurally separate from the serialization layer — two tools where Protobuf is one.

- **Cap'n Proto.** Rejected at inception. Arena allocation and zero-copy semantics are operational complexity not warranted pre-load characterization. Multi-language ecosystem less mature than Protobuf's. May be revisited under a future RFC if throughput evidence demands zero-copy semantics.

- **FlatBuffers.** Rejected at inception. Optimized for read-heavy workloads; §2.1 immutability + content-addressing characterizes write-once-read-many access, but the actual access patterns are not yet characterized empirically. Premature optimization for a workload not yet observed.

If only Protobuf had been considered, that itself would be a failure mode of the analysis. The four alternatives above are surfaced so the discussion phase can test the comparison rather than ratify a foregone conclusion.

## Open Questions

The RFC explicitly defers:

- **Specific Protobuf version pin.** proto3 latest stable vs a pinned release. Discussion-phase choice.
- **Code-generation toolchain.** `protoc` directly vs `buf` (the Buf CLI provides linting, breaking-change detection, formatting). Recommend exploration during discussion.
- **Schemas-evolution policy beyond "never reuse field numbers."** A dedicated schemas-evolution-discipline RFC is likely follow-on work; not bundled here.
- **Build-time vs commit-time code generation.** Standard practice is build-time; verify against the implementation-language RFC's toolchain decision before pinning.
- **`google.protobuf.Timestamp` vs `int64` Unix nanoseconds for time-typed fields.** Both are common; tradeoffs (cross-language compatibility, content-hash stability under tooling changes) need testing in discussion.

## Anti-Patterns to Avoid

By analogy to Charter [§2.1](../../charter/constitutional-charter.md#21-observational-integrity) and [§2.2](../../charter/constitutional-charter.md#22-epistemic-separation) `Forbidden Anti-Patterns` sections. Each is concrete and falsifiable.

- **Using `google.protobuf.Any` to defeat §2.2 categorical separation.** The `Any` type carries an arbitrary serialized message plus a type URL. Using `Any` in a field that should carry a categorically-typed message collapses the §2.2 distinction by allowing any category to inhabit any reference. `Any` is permitted only at boundary layers (e.g. transport-level envelopes); never inside Category I / II / III message bodies. Detectable mechanically: grep generated descriptors for `google.protobuf.Any` field usage inside categorical message types.

- **Using `oneof` to model the rejected unified-record-with-discriminator pattern.** Q3 [`§0016`](../../charter/decision-log.md) uses `oneof` to enforce exclusivity among categorically-distinct `subject_ref_*` fields. This is structurally different from a unified record with a type discriminator (rejected per [§2.2 Forbidden Anti-Patterns](../../charter/constitutional-charter.md#22-epistemic-separation)). The branches of a permitted `oneof` are categorically-distinct reference fields; the branches of a forbidden `oneof` would be tagged variants of one record type. Detectable: review every `oneof` declaration; confirm each branch resolves to a categorically-distinct typed reference.

- **Reusing field numbers from retired fields.** Protobuf field numbers are part of the wire format. Reusing a retired field number creates silent semantic conflation across schemas versions, violating §2.1's reconstructibility guarantee for historical records. Detectable: enforce the `reserved` clause for retired numbers; CI check on every `.proto` PR.

- **Generating code into the substrate.** Generated code (e.g. `*.pb.go` files from `protoc-gen-go`) is build output, not source. Commit the `.proto` source; generate the language-specific bindings at build time. Generated code in version control creates synchronization risk between the schemas layer and the bindings. Detectable: `.gitignore` covers generated artifacts; CI check that no generated bindings are committed.

- **Hash-instability via tooling-version drift.** Content-hash computation depends on canonical serialization; canonical serialization depends on the Protobuf library implementation. Pin the implementation version per the implementation-language RFC; document the canonical-serialization contract; treat library upgrades as schemas-evolution events requiring explicit consideration.

## Migration and Backward Compatibility

No historical schemas exist. Forward-looking decision. Subsequent commits (separate from this RFC; gated on this RFC's acceptance) will create the initial `.proto` files for Cat I / Cat II / Cat III + Assertion + provenance edges per [`§0022`](../../charter/decision-log.md) Consequences. No replay-model migration applies because no prior substrate state exists.

## References

- [`docs/charter/constitutional-charter.md`](../../charter/constitutional-charter.md) §2.1, §2.2, §2.3 (frozen v0.4), §2.5 (frozen v0.3). §2.4 + §2.6 pending — empirical pressure phase.
- [`docs/charter/decision-log.md`](../../charter/decision-log.md) [`§0003`](../../charter/decision-log.md) (partial-reversal authority for schemas / language / storage technology), [`§0010`](../../charter/decision-log.md) (Q2 hypothesis subtypes), [`§0015`](../../charter/decision-log.md) (Q1 session duality), [`§0016`](../../charter/decision-log.md) (Q3 subject_ref polymorphism), [`§0020`](../../charter/decision-log.md) (OMQ #2 decay of influence), [`§0021`](../../charter/decision-log.md) (OMQ #3 influence at projection vs substrate), [`§0022`](../../charter/decision-log.md) (pivot authorization), [`§0023`](../../charter/decision-log.md) (Q2 Identity tiers resolution per pivot).
- [`docs/ontology/entity-model.md`](../../ontology/entity-model.md) Assertion entity post-§0016; Cat I / II / III sections post-§0015 / §0023.
- [`docs/ontology/provenance-model.md`](../../ontology/provenance-model.md) post-OMQ #2-C + OMQ #3-α.
- [`.claude/CLAUDE.md`](../../../.claude/CLAUDE.md) §6.4 (implementation gate, amended at v0.4.2 per [`§0022`](../../charter/decision-log.md)).

## Decision Record

Empty. Populated at status `accepted` after discussion phase completes and the resolution-phase commit records the final commitment. Per [`§0022`](../../charter/decision-log.md) Consequences, the acceptance commit advances `status: draft` → `status: discussion` → `status: accepted` via the [`§0011`](../../charter/decision-log.md)–[`§0021`](../../charter/decision-log.md) pattern; discussion phase is NOT bypassed.
