# RFC — Architecture: Implementation language selection

- **Status:** draft
- **Authors:** committee
- **Date:** 2026-05-19
- **Type:** architecture
- **Affects:** [`services/`](../../../services/) (language-toolchain commitment for service implementations); [`docs/architecture/`](../../architecture/) (future architecture documents inherit the choice + the canonical-serialization contract referenced in [`§0024`](../../charter/decision-log.md) AP5); [`docs/charter/decision-log.md` §0003](../../charter/decision-log.md) (language-technology portion of the deferral; predicate satisfied per [`§0022`](../../charter/decision-log.md)); [`docs/charter/decision-log.md` §0022](../../charter/decision-log.md) (authorizing pivot entry); [`docs/charter/decision-log.md` §0024](../../charter/decision-log.md) (schemas-technology selection; inherited F1 cross-RFC coupling — chosen language must have maintained `protoc` plugin AND faithful proto3 wire-format implementation); [Charter §2.1](../../charter/constitutional-charter.md#21-observational-integrity), [§2.2](../../charter/constitutional-charter.md#22-epistemic-separation), [§2.3](../../charter/constitutional-charter.md#23-provenance-integrity) (frozen — selection respects, does not modify).

> RFCs in Ghost Trace are proposals subject to constitutional review. The [Constitutional Charter](../../charter/constitutional-charter.md) is the authoritative reference. RFCs that conflict with the Charter must either be rejected or accompanied by a corresponding charter amendment (see [`../../charter/amendments.md`](../../charter/amendments.md)).

---

## Summary

**Draft position (to be tested in discussion phase, not accepted by inclusion here):** select **Go (latest stable release)** as the implementation language for Ghost Trace services. Package boundaries align with Cat I / Cat II / Cat III separation at compile time; mature `protoc-gen-go` for the schemas-technology selection per [`§0024`](../../charter/decision-log.md); single static binary per service simplifies multi-service deployment topology; standard-library concurrency primitives align with event-stream substrate semantics.

Second of three technology RFCs authorized by [`decision-log §0022`](../../charter/decision-log.md). Inherits cross-RFC coupling from [`§0024`](../../charter/decision-log.md) Open Questions F1: the chosen language must have a maintained `protoc` plugin AND a faithful proto3 wire-format implementation. This RFC carries the draft position into discussion phase; resolution phase produces the final commitment.

## Motivation

The implementation-language selection is gating for any service-level work under [`§0022`](../../charter/decision-log.md) and is constrained by [`§0024`](../../charter/decision-log.md) F1 (cross-RFC coupling with schemas technology — Protobuf proto3). The four frozen object-level invariants impose distinct language constraints:

- **[§2.1 Observational Integrity](../../charter/constitutional-charter.md#21-observational-integrity) (frozen)** requires content-addressable identifiers via canonical Protobuf serialization. The implementation language must have a Protobuf library implementation that produces bit-stable canonical bytes per [`§0024`](../../charter/decision-log.md) AP5 (hash-instability mitigation: pinned-library-version + canonical-serialization-contract + library-upgrade-as-schemas-evolution + CI golden-file gate). Go's `google.golang.org/protobuf` library is mature, widely tested, and supports the proto3 spec faithfully.

- **[§2.2 Epistemic Separation](../../charter/constitutional-charter.md#22-epistemic-separation) (frozen)** requires schema-level nominally-distinct categorical types — and per the discussion-phase finding for the schemas RFC, *compile-time* enforcement is the relevant property (runtime enforcement is recoverable but does not prevent category-confusion bugs from reaching production). Go's package-private types, unexported struct fields, and absence of inheritance enforce categorical separation at compile time by default — no opt-in linting or stricter-mode configuration required. Cat I / Cat II / Cat III each gets its own Go package; cross-category reference requires an explicit typed conversion through a documented boundary function.

- **[§2.3 Provenance Integrity](../../charter/constitutional-charter.md#23-provenance-integrity) (frozen v0.4)** requires oneOf/union exclusivity on the Assertion `subject_ref_*` fields per [`§0016`](../../charter/decision-log.md). Protobuf's `oneof` generates Go interface types with exactly-one-implementing-struct semantics; the Go bindings enforce the exclusivity at the type level (assigning to one branch field clears any prior branch).

- **[§2.5 Hypothesis Lifecycle Explicitness](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) (frozen v0.3)** requires no language-specific support beyond §2.1 + §2.2 inheritance.

- **[§2.4 + §2.6 (pending — empirical pressure phase)](../../charter/constitutional-charter.md#24-inferential-influence-disclosure)**: Go's type system accommodates future additions (optional Protobuf fields generate Go-pointer-typed fields, allowing nil-checks for absence); no language-level barrier to §2.4 / §2.6 redaction.

The cost of not making this selection (technology dimension): no service code can be authored; ingestion service skeleton work halts. The procedural cost (Charter-governance dimension): the empirical-evidence-from-implementation feedback loop that §2.4 / §2.6 redaction depends on per [`§0022`](../../charter/decision-log.md) cannot begin (same shape as the schemas RFC's procedural cost — both technology RFCs are upstream of the empirical-evidence flow).

## Constitutional Review

Verbatim output of the rfc-author §1 pre-authorship analysis (Q1–Q6).

### Q1 — Charter invariants touched

- **§2.1 Observational Integrity (frozen):** implementation language must have a Protobuf library producing bit-stable canonical bytes (per [`§0024`](../../charter/decision-log.md) AP5 mitigation). Selection respects via `google.golang.org/protobuf` library commitment.
- **§2.2 Epistemic Separation (frozen):** compile-time enforcement of nominally-distinct categorical types preferred over opt-in runtime enforcement. Go's package-private types and unexported fields enforce at compile time by default. Selection respects.
- **§2.3 Provenance Integrity (frozen v0.4):** Go's Protobuf `oneof` generated bindings enforce [`§0016`](../../charter/decision-log.md) Q3 oneOf exclusivity at the type level. Selection respects.
- **§2.5 Hypothesis Lifecycle Explicitness (frozen v0.3):** no language-specific requirement beyond §2.1 + §2.2 inheritance.
- **§2.4 + §2.6 (pending — empirical pressure phase):** not directly touched. Go's type system accommodates forward-compatible additions when those invariants redact.

### Q2 — Glossary redefinition

No. Language names (Go, Rust, Python, etc.) and Go-specific terminology (`google.golang.org/protobuf`, package-private types, goroutines) are technology vocabulary, not canonical project vocabulary. Recommend do NOT add to [`docs/glossary.md`](../../glossary.md).

### Q3 — Implicit resolution of open Ontology questions

None. The five [`ontology.md`](../../ontology/ontology.md) §Open Questions remaining (Q3 independence formal definition; Q5 "transitive?" half) are not touched by language selection. [`provenance-model.md`](../../ontology/provenance-model.md) OMQ #1 (Granularity) + OMQ #4 (Cross-domain) not touched.

### Q4 — Charter amendment required

No. The pivot at [`§0022`](../../charter/decision-log.md) authorized this RFC under ordinary discipline.

### Q5 — New invariant introduced

No. Technology selection. Compile-time-enforcement-of-§2.2 is a *property of the implementation* that downstream-implements an existing invariant (§2.2 frozen), not a new invariant.

### Q6 — Ceremony without behavioral consequence

No. Selection is gating for all service-level work. Falsifiable by deletion: without it, no `.go` (or `.rs`, `.py`, `.ts`) files can exist; the ingestion service skeleton work halts.

## Proposal

**Draft position (to be tested in discussion phase):** adopt **Go (latest stable release at acceptance)** as the implementation language for services in [`services/`](../../../services/). Concrete commitments:

1. **Package-per-category discipline.** Cat I primary observations, Cat II operational constructs, Cat III hypotheses each get their own Go package. Cross-category reference uses explicit typed conversion through a documented boundary function. Unexported struct fields hide category-internal state; exported types carry the categorical commitment in their package import path. This exemplifies §2.2 at the package layer.
2. **Pinned Protobuf library version.** Use `google.golang.org/protobuf` (the v2 module, not the legacy `github.com/golang/protobuf`) at a pinned version recorded in `go.mod`. Library upgrades treated as schemas-evolution events per [`§0024`](../../charter/decision-log.md) AP5 (c).
3. **Code generation.** `protoc-gen-go` for message types; `protoc-gen-go-grpc` reserved for future RFC if/when service-to-service gRPC is introduced. Generated code is build-time output, not committed (per [`§0024`](../../charter/decision-log.md) AP3-derived anti-pattern — "Generating code into the substrate").
4. **Build-time canonical-serialization gate.** Golden-file test per [`§0024`](../../charter/decision-log.md) AP5 (d): representative message set serialized at CI, byte-compared against golden bytes; failure on library upgrade or accidental serialization-behavior change.
5. **Static binary per service.** Each service builds to a single static binary (no shared-library runtime dependencies); simplifies deployment topology and reproducibility.
6. **Standard library + minimal external dependencies.** Concurrency via goroutines + channels (standard library). External dependencies admitted only when the standard library does not suffice; each external dependency recorded with a justification in the service's `README.md`.
7. **`buf generate` integration deferred.** Per [`§0024`](../../charter/decision-log.md) Proposal item 7 — `buf` is the toolchain for linting + breaking-change detection. Code-generation invocation (`protoc` directly vs `buf generate`) deferred to follow-on tooling RFC; both are compatible with Go's protoc plugin.

## Alternatives Considered

Four alternatives evaluated. Two rejected on §2.2 compile-time-enforcement grounds (Python, TypeScript); two rejected as admissible-but-deferred (Rust, JVM languages).

- **Rust.** Rejected as admissible-but-deferred. Rust's type system is structurally *stronger* than Go's for §2.2 enforcement (the borrow checker + module-private types + exhaustive matching enforce categorical separation more rigidly than Go's package boundaries). The rejection is operational: Rust's ownership model imposes learning-curve and toolchain-complexity costs not warranted at inception phase before service topology is empirically characterized. The `prost` and `rust-protobuf` crates are mature; `protoc-gen-prost` is maintained. Revisit condition R-lang-1 (service-topology complexity demands compile-time memory-safety guarantees; or a §2.6 evidential-independence computation requires zero-cost abstractions Go cannot provide).

- **Python.** Rejected on §2.2 compile-time-enforcement grounds (not deferred). Python's structural type checking via mypy is *opt-in*: a Python module without mypy annotations passes the interpreter as valid code; category-confusion errors surface only at runtime when a Cat III value reaches a Cat I-typed function. The Charter's §2.2 forbidden anti-pattern "Unified assertion models … Defining a single generic record type with a 'kind' field distinguishing observation from inference" is a structural prohibition; Python's default duck typing does not enforce it — an enforcement RFC would have to mandate strict-mypy across all services and audit for compliance. That mandate is achievable but converts compile-time enforcement into procedural enforcement, weakening the structural property the Charter codifies. The `google.protobuf` Python library is mature; the rejection is not about Protobuf support.

- **TypeScript / Node.** Rejected on §2.2 compile-time-enforcement grounds + npm-ecosystem concerns. TypeScript's `strict` mode + `noImplicitAny` provides compile-time type checking comparable to Go's, but the enforcement is *configuration-dependent* (a `tsconfig.json` with relaxed settings disables it); additionally, the npm ecosystem's transitive-dependency surface area introduces operational dependencies (lockfile drift, vulnerability surface, supply-chain risk) disproportionate to inception-phase needs. `ts-proto` and `protobufjs` are mature. Revisit condition R-lang-2 if a service-tier emerges where Node.js-specific properties (e.g. browser interop for a future operator-facing tool) become load-bearing.

- **JVM languages (Java / Kotlin / Scala).** Rejected as admissible-but-deferred. JVM type systems enforce categorical separation at compile time (package-private + visibility modifiers; Kotlin's `internal` + sealed classes are particularly well-suited). The rejection is operational: JVM deployment weight (runtime requirement, memory footprint, startup time) is inappropriate for inception-phase services where per-service cost matters. `protoc-gen-java` is mature. Revisit condition R-lang-3 (service-tier emerges where JVM-ecosystem libraries — e.g. a specific stream-processing framework — become load-bearing).

The admissible-but-deferred registrations (Rust + JVM languages) follow the [`§0024`](../../charter/decision-log.md) committee-extension-1 pattern (third + fourth instance of admissible-but-deferred — now fifth + sixth across all RFCs counting [`§0020`](../../charter/decision-log.md) B-substrate, [`§0021`](../../charter/decision-log.md) β, [`§0024`](../../charter/decision-log.md) Cap'n Proto + FlatBuffers).

If only Go had been considered, that itself would be a failure mode of the analysis. The four alternatives above are surfaced so the discussion phase can test the comparison rather than ratify a foregone conclusion.

## Open Questions

The RFC explicitly defers:

- **Specific Go version pin.** Go 1.22+ recommended (generics + `slog`); exact minor version pinned at acceptance.
- **`golangci-lint` configuration.** Linting baseline (which lint rules enforce categorical-separation conventions, e.g. `revive`, `gocritic`, `gosec`). Follow-on tooling RFC.
- **Module organization.** Single-module vs multi-module (one Go module per service vs one module for the whole repo). Follow-on architecture RFC; not bundled here.
- **Concurrency model conventions.** Goroutine lifecycle discipline, channel ownership conventions, context propagation. Follow-on architecture document; named here as carry-forward.
- **gRPC adoption.** Service-to-service RPC technology is deferred to a separate RFC; this RFC commits only to in-process Go code, not to a service-tier IPC protocol.
- **Cross-RFC coupling with inception-phase storage-technology RFC.** Go's database-driver ecosystem (SQLite via `mattn/go-sqlite3` or `modernc.org/sqlite`; PostgreSQL via `jackc/pgx`; etc.) constrains the storage-technology RFC's option space. Surface explicitly to the storage-technology RFC's discussion phase (analogous to how [`§0024`](../../charter/decision-log.md) F1 constrained this RFC).

## Anti-Patterns to Avoid

By analogy to Charter [§2.1](../../charter/constitutional-charter.md#21-observational-integrity) and [§2.2](../../charter/constitutional-charter.md#22-epistemic-separation) `Forbidden Anti-Patterns` sections. Each is concrete and falsifiable.

- **Cross-category type imports without explicit boundary function.** A Cat II package directly importing a Cat I package's unexported types via reflection or `unsafe` to construct Cat II records bypassing the documented boundary function. Collapses §2.2 by eliminating the typed conversion. Detectable: grep service code for `reflect` + `unsafe` imports in category-package code; review every cross-category-package import in service `import` blocks.

- **Using `interface{}` (or `any` in Go 1.18+) to defeat categorical typing.** A function signature `func process(x any)` accepting Category I, II, or III values interchangeably collapses §2.2 at the type level — equivalent to the rejected unified-record-with-discriminator pattern at the Go-language layer. `any`-typed parameters are permitted only at serialization boundaries (where the value is immediately type-asserted to a specific categorical type), never as a load-bearing internal interface. Detectable: lint rule on `any` parameter types in category-package code.

- **Returning generated Protobuf types from service-internal interfaces.** Generated `*.pb.go` types are the wire-format representation; service-internal logic should operate on the Go domain types defined per Cat I/II/III packages, not on the wire types. Conflation creates implicit coupling between the wire format and the service's internal abstractions. Generated types are permitted at serialization boundary functions; never in service-internal call signatures.

- **Replacing the standard library's concurrency primitives with third-party abstractions.** Goroutines + channels + `context.Context` are the canonical concurrency vocabulary in Go; replacing them with third-party libraries (async runtimes, reactive frameworks, actor libraries) introduces operational dependencies that weaken Go's "single static binary, no shared-library dependencies" property and increase the surface area of canonical-serialization-contract regression (per [`§0024`](../../charter/decision-log.md) AP5).

## Migration and Backward Compatibility

**Inception phase.** No historical Go code exists. Forward-looking decision. Subsequent commits will create the initial service skeleton (likely the ingestion service per [`§0022`](../../charter/decision-log.md) original brief) under the package-per-category discipline.

**Language-technology reversal cost (forward-looking).** Switching from Go to a different implementation language post-service-implementation requires re-authoring of service code; this is non-trivial cost but is NOT §2.1-load-bearing (unlike schemas-technology reversal per [`§0024`](../../charter/decision-log.md) F6 — historical bytes do not need re-serialization since the wire format is language-independent). Language reversal is service-tier work, not substrate work. The four reversal conditions R-lang-1 through R-lang-3 characterize the threshold for accepting the cost.

## References

- [`docs/charter/constitutional-charter.md`](../../charter/constitutional-charter.md) §2.1, §2.2, §2.3 (frozen v0.4), §2.5 (frozen v0.3). §2.4 + §2.6 pending — empirical pressure phase.
- [`docs/charter/decision-log.md`](../../charter/decision-log.md) [`§0003`](../../charter/decision-log.md) (partial-reversal authority — language-technology portion; satisfied by this RFC's acceptance), [`§0015`](../../charter/decision-log.md) (Q1), [`§0016`](../../charter/decision-log.md) (Q3), [`§0022`](../../charter/decision-log.md) (pivot authorization), [`§0024`](../../charter/decision-log.md) (schemas-technology selection — Protobuf proto3; F1 inherited as cross-RFC coupling; AP5 inherited as pinned-library-version commitment).
- [`docs/rfcs/draft/architecture-schemas-technology-selection.md`](./architecture-schemas-technology-selection.md) (accepted at [`§0024`](../../charter/decision-log.md); F1 + AP5 inheritance).
- [`docs/ontology/entity-model.md`](../../ontology/entity-model.md) Cat I / II / III sections (package-per-category discipline maps to these).
- [`.claude/CLAUDE.md`](../../../.claude/CLAUDE.md) §6.4 (implementation gate, cleared at [`§0022`](../../charter/decision-log.md)).

## Decision Record

Empty. Populated at status `accepted` after discussion phase completes and the resolution-phase commit records the final commitment. Per [`§0024`](../../charter/decision-log.md) pattern (status `discussion` → `accepted`; file remains in `docs/rfcs/draft/`; no renumbering applied per the surfaced procedural-divergence finding); discussion phase is NOT bypassed.
