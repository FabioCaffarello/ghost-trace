# Schemas technology selection — discussion evidence

**Status:** in-discussion. Not authoritative. Final resolution will be recorded in decision-log and the RFC's `Decision Record` section at acceptance.

This scratch supports the discussion phase of the [schemas technology selection RFC](../draft/architecture-schemas-technology-selection.md) — first of three technology RFCs authorized by [`decision-log §0022`](../../charter/decision-log.md) implementation pivot. First non-ontology RFC discussion (six prior discussion artifacts are ontology-RFC discussions: Q2-1, Q4-1, Q1-1, Q3-1, OMQ #2-1, OMQ #3-1). The OMQ #2/3 evidence-file pattern is adapted: ontology-RFC dimensions concern Charter compliance of candidate semantics; technology-RFC dimensions concern Charter compliance of candidate primitives + operational properties.

The RFC's draft positions: **Protocol Buffers (proto3)** recommended; **Apache Avro / JSON-Schema-validated JSON / Cap'n Proto / FlatBuffers** rejected at inception. This scratch evaluates the recommended candidate across four Charter-frozen-invariant dimensions, re-tests each rejection rationale against the draft's premise, applies the three rfc-author §3 discipline skills (falsifiability-check, epistemic-separator, ambiguity-reducer) to the draft's substantive claims, surfaces non-obvious risks, and produces a recommendation with reversal conditions.

## Phase 1 — Recommended candidate (Protobuf proto3) per frozen invariant

Four cells. Verdict per cell: clean | conditional | violation. Source citations after each.

### Dimension 1 — §2.1 Observational Integrity (frozen)

§2.1 requires that observations carry content-addressable identifiers sufficient to detect mutation if attempted ([§2.1 Structural Requirement](../../charter/constitutional-charter.md#21-observational-integrity)). Content-addressing requires deterministic serialization — given a message instance, the byte sequence the technology produces must be identical across implementations sharing the same descriptors.

**Protobuf proto3 (recommended).** **Clean** — with one operational caveat. proto3 specifies canonical wire format with explicit field ordering: fields serialized in ascending tag-number order, scalar fields encoded with deterministic varint/fixed encodings, repeated fields preserved in source order, `oneof` carries exactly one branch's serialization. Hash-based content-addressing (BLAKE3 or otherwise) computed over the canonical serialization is reproducible across language bindings (Go, Rust, Python, etc.) that implement the proto3 wire format spec faithfully. The operational caveat: proto3 maps (`map<K, V>`) do NOT have a canonical iteration order under all language bindings — map field serialization can differ across implementations. Mitigation: ban `map<K, V>` in canonical-form-load-bearing message types; use `repeated` of a sub-message with `key` + `value` fields with explicit sort. This is anti-pattern AP6 candidate (added in Phase 4 below).
- *Citation:* [Protocol Buffers Encoding documentation](https://protobuf.dev/programming-guides/encoding/) (proto3 wire format spec). [`§0003`](../../charter/decision-log.md) (technology-by-property requirement: deterministic serialization).

### Dimension 2 — §2.2 Epistemic Separation (frozen)

§2.2 requires schema-level nominally-distinct categorical types — Cat I, Cat II, Cat III each expressed as distinct types with no shared discriminator field ([§2.2 Forbidden Anti-Patterns](../../charter/constitutional-charter.md#22-epistemic-separation): "Unified assertion models … Defining a single generic record type with a 'kind' field distinguishing observation from inference from operational construct").

**Protobuf proto3.** **Clean** — with one critical anti-pattern to surface. Protobuf message types are nominally distinct by construction: each `message Foo {...}` defines a separate type with its own descriptor, no shared base class or discriminator field at the protocol layer. Cat I `DeclaredSession`, Cat II `OperationalSession`, Cat III `Hypothesis` are distinct types; their structural commonality at the level of common fields (identifiers, timestamps, provenance references — per [Charter §2.2 Boundary Conditions](../../charter/constitutional-charter.md#22-epistemic-separation)) is achieved via shared field-number conventions or shared common-fields message types embedded by composition, not by inheritance. Critical anti-pattern surfaced (already in RFC AP1): `google.protobuf.Any` defeats categorical separation if used inside categorical message bodies, by allowing any serialized message to inhabit any reference. The anti-pattern is mechanically detectable: grep for `Any` field usage in `.proto` files under categorical types.
- *Citation:* [Charter §2.2 Forbidden Anti-Patterns L94](../../charter/constitutional-charter.md#22-epistemic-separation). RFC §Anti-Patterns AP1.

### Dimension 3 — §2.3 Provenance Integrity (frozen v0.4)

§2.3 requires that the Assertion entity carries a typed `subject_ref_*` reference field with oneOf/union exclusivity per [`§0016`](../../charter/decision-log.md) (Q3 resolution) — exactly one of `subject_ref_observation`, `subject_ref_construct`, `subject_ref_hypothesis` populated per Assertion ([§2.3 Structural Requirement](../../charter/constitutional-charter.md#23-provenance-integrity)).

**Protobuf proto3.** **Clean.** proto3's `oneof` primitive enforces exactly-one-populated semantics structurally at the descriptor layer: assignment to any branch of a `oneof` clears all other branches; serialization carries exactly one branch's field number and value; deserialization populates exactly one branch. No application-level discriminator check is required at the bindings layer — the wire format makes multi-population impossible at construction time. This is the cleanest cross-technology fit observed in Phase 1: Q3's oneOf requirement maps to a single proto3 primitive without translation.
- *Citation:* [Protocol Buffers `oneof` documentation](https://protobuf.dev/programming-guides/proto3/#oneof). [`§0016`](../../charter/decision-log.md) Q3 resolution Candidate B. [Charter §2.3 frozen v0.4 Structural Requirement L113](../../charter/constitutional-charter.md#23-provenance-integrity).

### Dimension 4 — §2.5 Hypothesis Lifecycle Explicitness (frozen v0.3)

§2.5 requires that lifecycle events (formation, merge, split, promotion, demotion, dissolution) are Cat I records committed immutably under §2.1, carrying antecedent references where applicable (merge events reference all antecedents; promotion events carry demotion-candidacy parameters). The schemas technology must express these events as message types with the appropriate reference structures.

**Protobuf proto3.** **Clean** — no schemas-technology-specific requirement beyond §2.1 (immutability via content-addressing) + §2.2 (nominally-distinct event types) + repeated-field support (for antecedent reference lists in merge events). All three are satisfied by Phase 1 Dimensions 1, 2, and standard Protobuf primitives. Lifecycle events as their own typed messages — `FormationEvent`, `MergeEvent`, `SplitEvent`, `PromotionEvent`, `DemotionEvent`, `DissolutionEvent` — each with a `repeated` field for antecedent references where applicable.
- *Citation:* [Charter §2.5 Structural Requirement L162](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness). [Charter §2.1 Structural Requirement L55](../../charter/constitutional-charter.md#21-observational-integrity).

## Phase 2 — Alternative rejection re-test

Four cells. The draft RFC rejected Avro, JSON-Schema-validated JSON, Cap'n Proto, FlatBuffers at inception. This phase re-tests each rejection rationale against the actual constraints — does the rejection hold under empirical scrutiny, or is it weaker than the RFC asserted?

### Cell A — Apache Avro re-test

**RFC's rejection rationale:** Schemas-registry coupling adds operational dependency; union types support oneOf but require runtime descriptor lookup, weakening static §2.2 separation; Avro's strength is dynamic schemas evolution under registry mediation, which §2.1 substrate-immutability does not benefit from.

**Re-test.** The registry-coupling claim is empirically weakened: Avro can be used without a centralized registry, by shipping schemas-descriptor headers alongside encoded payloads (the "single-object encoding" mode in the Avro spec, c. 2014). Inception-phase use without a registry IS supported. However, the static-vs-dynamic typing concern holds: Avro's typed-union mechanism (`union { Foo, Bar }`) requires descriptor consultation at decode time to identify which type a given payload carries; Protobuf's `oneof` carries the branch field number in the wire format. For §2.3 oneOf exclusivity, Protobuf's design encodes the discriminator structurally; Avro's design encodes it via descriptor reference. Both work; Protobuf's coupling between wire format and discriminator is tighter and aligns more directly with §2.2's nominally-distinct-types principle. **Verdict on RFC's rejection:** weakened on registry-coupling claim (correctable in discussion phase); preserved on static-typing alignment with §2.2.
- *Citation:* [Apache Avro single-object encoding spec](https://avro.apache.org/docs/1.11.1/specification/#single-object-encoding). [Charter §2.2 Definition L80](../../charter/constitutional-charter.md#22-epistemic-separation).

### Cell B — JSON-Schema-validated JSON re-test

**RFC's rejection rationale:** JSON has no canonical serialization; content-addressing via JSON is fragile; canonical-JSON specifications (RFC 8785 JCS) reintroduce what Protobuf provides natively, plus the validation layer is structurally separate from the serialization layer.

**Re-test.** The canonical-serialization claim holds robustly. JSON's RFC 8259 spec explicitly leaves key ordering, whitespace, and number representation implementation-defined. RFC 8785 (JSON Canonicalization Scheme, 2020) defines a deterministic canonical form, but adoption requires both producer and consumer to implement JCS faithfully — and JCS does NOT cover all JSON-Schema-validated payloads cleanly (e.g., number-format edge cases). Content-addressing via canonical-JSON is achievable but operationally fragile relative to Protobuf's native canonical wire format. The "two-tool" concern (validation layer separate from serialization layer) holds: JSON-Schema is a validation language, JSON is a serialization, and their coupling is by convention (validators run on parsed JSON), not by construction. Protobuf collapses both into one tool. **Verdict on RFC's rejection:** preserved on canonical-serialization fragility; preserved on two-tool concern; well-grounded.
- *Citation:* [RFC 8259 JSON spec](https://datatracker.ietf.org/doc/html/rfc8259) §8.1 ("JSON parsers SHOULD NOT consider… "). [RFC 8785 JSON Canonicalization Scheme](https://datatracker.ietf.org/doc/html/rfc8785).

### Cell C — Cap'n Proto re-test

**RFC's rejection rationale:** Arena allocation and zero-copy semantics are operational complexity not warranted pre-load characterization; multi-language ecosystem less mature than Protobuf's; may be revisited under future RFC if throughput evidence demands zero-copy.

**Re-test.** Cap'n Proto's wire format IS deterministic and supports content-addressing structurally (its packed encoding is canonical). Its `union` mechanism is structurally analogous to Protobuf's `oneof`. The substantive rejection rationale is operational, not Charter-constraint-driven. The maturity-gap claim is verifiable but degrades over time (Cap'n Proto's ecosystem has grown since the RFC was drafted). **Verdict on RFC's rejection:** preserved on inception-phase operational simplicity; the "revisit under throughput pressure" door explicitly held open in the RFC is procedurally correct — this is the right rejection shape (admissible-but-deferred, not structurally inadmissible). Pattern parallel to OMQ #2-2 B-substrate admissible-but-dominated registration.
- *Citation:* [Cap'n Proto schemas language documentation](https://capnproto.org/language.html). RFC §Alternatives Considered Cap'n Proto bullet.

### Cell D — FlatBuffers re-test

**RFC's rejection rationale:** Optimized for read-heavy workloads; §2.1 + content-addressing characterizes write-once-read-many, but actual access patterns not yet empirically characterized; premature optimization for a workload not yet observed.

**Re-test.** FlatBuffers' read-zero-copy design is real and well-documented; the workload-characterization concern is structurally sound (the substrate's access patterns at scale are unknown at inception). FlatBuffers' canonical serialization is deterministic (modulo offset-table alignment choices, which are spec-fixed). The maturity-gap claim is similar to Cap'n Proto's. **Verdict on RFC's rejection:** preserved on premature-optimization grounds; the rejection-shape is identical to Cap'n Proto's (admissible-but-deferred); same procedural correctness.
- *Citation:* [FlatBuffers documentation](https://flatbuffers.dev/). RFC §Alternatives Considered FlatBuffers bullet.

## Phase 3 — Discipline skills application

Per rfc-author §3, three skills apply before `status: discussion`. Findings recorded inline; rewrites surfaced for resolution-phase consideration (not applied here — discussion phase scrutinizes, resolution phase commits).

### 3.1 falsifiability-check (V/O/Op/NC four-question test)

Applied to the RFC's substantive claims in Summary, Motivation, Proposal sections.

| # | Claim (paraphrased) | V | O | Op | NC | Verdict |
|---|---|---|---|---|---|---|
| FC1 | "proto3 specifies canonical wire format with explicit field ordering; serialization is bit-stable across implementations sharing the same descriptors" (Motivation) | ✓ violation = two compliant implementations produce different bytes for same input | ✓ third party hashes both outputs, compares | ✓ "canonical wire format" = proto3 spec §3 | ✓ relies on proto3 spec definitions | **Pass** |
| FC2 | "Protobuf message types are nominally distinct by construction; no shared discriminator field exists at the protocol layer" (Motivation) | ✓ violation = descriptor parser finds shared discriminator base | ✓ inspect generated descriptors | ✓ "nominally distinct" = separate descriptor entries | ✓ Protobuf descriptor model is external standard | **Pass** |
| FC3 | "Protobuf's `oneof` primitive enforces exactly-one-populated semantics structurally at the schemas layer" (Motivation) | ✓ violation = wire payload carries two oneof branch tags | ✓ wire-format inspection | ✓ "exactly-one-populated" = proto3 wire format spec §oneof | ✓ proto3 spec external | **Pass** |
| FC4 | "Protobuf is extensible via field-number reservation and optional fields, allowing forward-compatible additions" (Motivation) | ✓ violation = adding optional field breaks old decoders | ✓ test old decoder against new payload | ✓ "forward-compatible" = proto3 evolution rules | ✓ proto3 spec | **Pass** |
| FC5 | "Avro's strength is dynamic schemas evolution under registry mediation, which §2.1 substrate-immutability does not benefit from" (Alternatives) | ⚠ partial — "does not benefit from" is comparative, harder to localize | ⚠ depends on what benefit is | ⚠ "benefit" not operationalized | ✓ vocabulary OK | **Pass-with-caveat (rewrite suggested):** rephrase as "§2.1 substrate-immutability does not require dynamic schemas evolution as a service, and Avro's registry mediation adds an operational dependency without an offsetting Charter-anchored requirement." |
| FC6 | "Cap'n Proto's arena allocation and zero-copy semantics are operational complexity not warranted pre-load" (Alternatives) | ⚠ partial — "not warranted" is judgment-bound | ⚠ depends on what counts as warrant | ⚠ "warranted" not operationalized | ✓ vocabulary OK | **Pass-with-caveat (rewrite suggested):** rephrase as "Cap'n Proto's arena-allocation and zero-copy properties are not required by any frozen invariant; inception-phase operational dependencies are minimized; revisit when throughput evidence at characterized scale demands these properties (reversal condition R-tech-1 below)." |
| FC7 | "FlatBuffers is optimized for read-heavy workloads … actual access patterns not yet empirically characterized; premature optimization" (Alternatives) | ⚠ partial — "premature" is judgment-bound | ⚠ depends on observed pattern | ⚠ "premature" not operationalized | ✓ vocabulary OK | **Pass-with-caveat (rewrite suggested):** same shape as FC6. Both FC6 + FC7 are admissible-but-deferred rejections; rewrite to operationalize via reversal conditions. |

**Falsifiability summary:** 4 pass clean (FC1–FC4) + 3 pass-with-caveat (FC5–FC7, all in Alternatives section, all rewritable via operationalization of comparative/judgment-bound modifiers). No claim fails. Rewrites are discussion-phase suggestions; resolution-phase commits them.

### 3.2 epistemic-separator (paragraph-level categorical distinctness)

Applied to RFC paragraphs in Summary, Motivation, Proposal, Anti-Patterns.

- **ES1 — Pass.** Summary paragraph carries technology-recommendation content; categorically distinct from constitutional commitment (it does not claim to be an invariant).
- **ES2 — Pass.** Motivation paragraphs cite frozen invariants explicitly (§2.1, §2.2, §2.3, §2.5) and characterize the technology's interaction with each; categorical structure clean.
- **ES3 — Pass.** Proposal paragraphs are concrete technology commitments; not mixed with Charter prose.
- **ES4 — Pass.** Anti-Patterns paragraphs cite §2.1 / §2.2 forbidden-anti-pattern templates (AP1 cites §2.2; AP2 cites §0016 / §2.2; AP3 cites §2.1; AP4 + AP5 are operational anti-patterns marked as such); categorical placement clean.
- **ES5 — Borderline / surface for discussion.** Paragraph in Motivation closing sentence: "The cost of not making this selection: no `.proto` (or `.avsc`, `.fbs`) files can exist; ingestion service work halts; the empirical-evidence-from-implementation feedback loop §2.4 / §2.6 redaction depends on cannot begin." Mixes technology commitment with §2.4 / §2.6 procedural-state claim. **Suggestion:** split into two sentences — one technology cost, one procedural-state cost — for cleaner categorical separation. Resolution-phase rewrite.

**Epistemic-separation summary:** 4 pass, 1 borderline-with-rewrite-suggestion.

### 3.3 ambiguity-reducer (advisory term flagging)

The pre-commit hook surfaced 7 ambiguity advisories at draft-commit time: `conflict`, `decision`, `evidence`, `identity`, `record`, `source`, `state`. Per the ambiguity-reducer skill, these are advisory; per CLAUDE.md §5.3, the author decides.

- **AR1 — `record`.** Appears in Constitutional Review Q5 ("Technology selection. The proposed commitments … are downstream consequences of existing §2.1 / §2.2 / §2.3 invariants, not new invariants.") and Q6 ("not records … nor projections"). Use is canonical — `record` references Cat I / II / III records per CLAUDE.md §3 canonical vocabulary. **Resolution:** acceptable as-is; canonical use.
- **AR2 — `identity`.** Appears in Q3 ("the §2.3 frozen v0.4 BC4 forward-reference contract names Q2") referencing identity-tier resolution per [`§0023`](../../charter/decision-log.md). Use is canonical — identity refers to actor identity per §0023. **Resolution:** acceptable as-is; canonical use.
- **AR3 — `state`.** Appears in Motivation "implementation-language RFC's toolchain decision before pinning" — no `state` here; advisory may be a false positive on partial-match. **Resolution:** investigate at resolution phase; likely benign.
- **AR4 — `source`.** Appears in Anti-Patterns AP4 ("Commit the `.proto` source") referring to source code. Use is operational, not Cat-I "primary observation source". **Resolution:** acceptable as-is; common-English use unambiguous in context.
- **AR5 — `conflict`, `decision`, `evidence`.** Standard prose use. **Resolution:** acceptable as-is.

**Ambiguity-reducer summary:** all 7 advisories resolve to acceptable-as-is on inspection; no rewrites required. The hook's advisory function operated correctly (it surfaced terms for inspection; the author judged them).

## Phase 4 — Non-obvious risks and findings

Findings the draft RFC does not surface but that empirical inspection raises. Numbered for discussion-phase tracking.

### F1 — Schemas-technology choice constrains implementation-language RFC

Protobuf's mature code-generation ecosystem covers Go, Rust, Python, TypeScript, Java, C++, Swift, Kotlin (and others) via standard `protoc` plugins. The implementation-language RFC (second of three technology RFCs per [`§0022`](../../charter/decision-log.md)) inherits Protobuf as a constraint: the chosen language MUST have a maintained `protoc` plugin AND a faithful proto3 wire-format implementation (per FC1 — canonical-serialization bit-stability across implementations). This constrains the implementation-language RFC's option space without itself being an explicit Charter commitment. **Action for resolution phase:** RFC's `Open Questions` section should explicitly surface this cross-RFC coupling.

### F2 — Hash-instability via tooling-version drift is real and underspecified

The draft RFC's Anti-Patterns AP5 surfaces hash-instability via tooling drift but does not specify the mitigation procedure. Concrete risk: BLAKE3 hash of canonical serialization depends on the Protobuf library version that produces the canonical bytes. A library version bump that changes default field-ordering behavior (e.g. unknown-fields handling, default-value serialization rules) silently invalidates historical content-hashes. **Action for resolution phase:** AP5 should be expanded to specify (a) pinned-library-version commitment per implementation-language RFC; (b) canonical-serialization contract document (which proto3 spec version + which library version produces the canonical bytes); (c) library-upgrade discipline as a schemas-evolution event.

### F3 — Map type (`map<K, V>`) ban needs explicit anti-pattern

Phase 1 Dimension 1 surfaced that proto3 `map<K, V>` does not have canonical iteration order across all language bindings. The draft RFC's Anti-Patterns section does not surface this. **Action for resolution phase:** add as AP6 — "Use of `map<K, V>` in canonical-form-load-bearing message types is forbidden. Mitigation: `repeated SubMessage { key, value }` with explicit sort-order convention. Detectable: grep `.proto` files for `map<` in categorical-message-type definitions."

### F4 — `google.protobuf.Timestamp` vs `int64` Unix-nanos tradeoff is load-bearing and unresolved

The draft RFC's Open Questions surface this as a deferred question. The resolution carries real consequence: `Timestamp` carries a nominal-type indicator (this field is a time) plus library-supported parsing across languages; `int64` Unix-nanos carries no nominal-type indicator but is bit-stable across all language bindings (no parsing layer involved). Recommendation for resolution phase: pick `int64` Unix-nanos for canonical-form-load-bearing time fields (content-hash stability prevails over nominal-type ergonomics), reserve `Timestamp` for non-canonical / projection-only fields. This is not in the draft; surface for committee.

### F5 — `buf` toolchain adds breaking-change detection that aligns with AP3

The draft RFC's Open Questions defer the `protoc` vs `buf` choice. `buf` (the Buf CLI, BSR ecosystem) provides linting + breaking-change detection + formatting. AP3 ("Reusing field numbers from retired fields") is mechanically detectable by `buf breaking` against a baseline. Recommendation: select `buf` for breaking-change detection regardless of whether code generation uses `protoc` directly or `buf generate`. This converts AP3 from a discipline obligation to a CI gate.

### F6 — Schemas-technology choice is reversible at higher cost than greenfield

Forward-compatibility properties (Protobuf field-number reservation) protect against intra-technology evolution but do not protect against inter-technology migration. Switching from Protobuf to (e.g.) Cap'n Proto post-substrate-commit requires re-serialization of all historical Cat I records, which is a §2.1-load-bearing operation — historical-bytes must be regenerable under the new technology, which conflicts with content-addressing if hashes are part of substrate references. Practical implication: schemas-technology reversal post-implementation is structurally expensive; the inception-phase selection has more weight than the "may be revisited" phrasing in alternatives suggests. **Action for resolution phase:** add to Migration and Backward Compatibility section.

## Phase 5 — Recommendation with reversal conditions

**Recommendation (for resolution-phase consideration):** accept the draft's selection of **Protocol Buffers (proto3)** as schemas technology, with the following modifications enacted in the resolution-phase commit:

1. **Rewrite Alternatives FC5/FC6/FC7 per Phase 3.1** — operationalize comparative/judgment-bound modifiers ("benefit", "warranted", "premature") via explicit reversal conditions.
2. **Split ES5 Motivation closing sentence** per Phase 3.2 — separate technology-cost claim from §2.4 / §2.6 procedural-state claim.
3. **Add AP6 per F3** — `map<K, V>` ban with mitigation.
4. **Expand AP5 per F2** — hash-instability mitigation procedure (pinned-library-version, canonical-serialization contract, library-upgrade-as-schemas-evolution).
5. **Add F1 to Open Questions** — schemas-technology constraint on implementation-language RFC's option space.
6. **Resolve F4 in Proposal** — pick `int64` Unix-nanos for canonical-form-load-bearing time fields.
7. **Resolve F5 in Open Questions or Proposal** — recommend `buf` for breaking-change detection.
8. **Add F6 to Migration and Backward Compatibility** — schemas-technology reversal cost characterization.

**Reversal conditions** (when to revisit this selection via subsequent RFC):

- **R-tech-1.** Throughput characterization at production-relevant scale demonstrates that proto3 serialize/deserialize cost dominates the ingest path. Threshold: serialize+deserialize >40% of single-message ingestion CPU at characterized load. (Revisit Cap'n Proto, FlatBuffers under throughput pressure.)
- **R-tech-2.** Multi-language ecosystem growth surfaces a categorical type Protobuf cannot express (e.g., sum-types with structural subtyping). Threshold: explicit RFC proposing the type and demonstrating Protobuf cannot encode it cleanly. (Revisit Avro union types.)
- **R-tech-3.** Inception-phase work surfaces canonical-serialization fragility that pinned-library-version + canonical-serialization-contract (per F2) cannot mitigate. Threshold: two empirically observed hash-divergence incidents across library versions. (Revisit JSON-Schema-validated JSON with JCS.)
- **R-tech-4.** Schemas-evolution discipline produces a breaking-change pattern Protobuf's field-number-reservation cannot accommodate. Threshold: explicit RFC documenting the pattern and the field-number-reservation failure mode. (Revisit Avro registry-mediated evolution.)

No reversal condition fires today; the recommendation stands for resolution-phase deliberation.

## Phase 6 — Carry-forwards

- The two remaining technology RFCs (implementation language, inception-phase storage technology) inherit F1 (cross-RFC coupling) and F6 (technology-reversal cost) as discussion-phase inputs. The schemas-technology selection should land first to provide structural ground for both.
- No new canonical project vocabulary introduced (Q2 of Phase 3 confirms). Glossary unchanged.
- No Ontology document revision required by this RFC's resolution.
- No Charter amendment required by this RFC's resolution (per Constitutional Review Q4).
- Layer B follow-on RFC remains on hold per [`§0011`](../../charter/decision-log.md); unaffected.
- Q2 (Identity tiers) resolution per [`§0023`](../../charter/decision-log.md) supplies the `actor_ref` string-field commitment that the eventual `.proto` files encode; no new resolution required here.
