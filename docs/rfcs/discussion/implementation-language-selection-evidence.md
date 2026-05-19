# Implementation language selection — discussion evidence

**Status:** in-discussion. Not authoritative. Final resolution will be recorded in decision-log and the RFC's `Decision Record` section at acceptance.

This scratch supports the discussion phase of the [implementation-language selection RFC](../draft/architecture-implementation-language-selection.md) — second of three technology RFCs authorized by [`decision-log §0022`](../../charter/decision-log.md). Follows the [`schemas-technology-selection-evidence.md`](./schemas-technology-selection-evidence.md) pattern adapted for language selection: Charter-fit dimensions concern compile-time enforcement of the four frozen object-level invariants + Protobuf-library faithfulness inheritance per [`§0024`](../../charter/decision-log.md) F1.

The RFC's draft positions: **Go (latest stable)** recommended; **Python / TypeScript-Node** rejected on §2.2 compile-time-enforcement grounds; **Rust / JVM languages** rejected as admissible-but-deferred. This scratch evaluates the recommended candidate across four Charter-frozen-invariant dimensions, re-tests each rejection rationale against the draft's premise, applies the three rfc-author §3 discipline skills, surfaces non-obvious risks, and produces a recommendation with reversal conditions.

## Phase 1 — Recommended candidate (Go) per frozen invariant

Four cells. Verdict per cell: clean | conditional | violation. Source citations after each.

### Dimension 1 — §2.1 Observational Integrity (frozen)

§2.1 requires content-addressable identifiers via canonical Protobuf serialization. Per [`§0024`](../../charter/decision-log.md) AP5, the implementation language's Protobuf library must produce bit-stable canonical bytes; pinned-library-version + canonical-serialization-contract + library-upgrade-as-schemas-evolution + CI golden-file gate are the mitigation procedure.

**Go (recommended).** **Clean** — with one library-history caveat. Go has two Protobuf libraries: `github.com/golang/protobuf` (v1, legacy) and `google.golang.org/protobuf` (v2, current). The v2 module is the maintained implementation; v1 is in maintenance-only mode with internal type aliases pointing to v2. The v2 library's `proto.MarshalOptions{Deterministic: true}.Marshal(...)` produces byte-stable canonical output across processes provided the same library version. The caveat: code that mixes v1 + v2 imports (common in transitional projects) can produce different serialization behavior across packages — strict v2-only commitment per RFC Proposal item 2 mitigates. Library upgrades governed by §0024 AP5 (c) (treat as schemas-evolution events).
- *Citation:* [`google.golang.org/protobuf` MarshalOptions documentation](https://pkg.go.dev/google.golang.org/protobuf/proto#MarshalOptions). [`§0024`](../../charter/decision-log.md) AP5 mitigation procedure.

### Dimension 2 — §2.2 Epistemic Separation (frozen)

§2.2 requires nominally-distinct categorical types. The discussion-phase finding worth emphasizing: *compile-time* enforcement is structurally distinct from *runtime* enforcement. The Charter's "Forbidden Anti-Patterns" subsection prohibits unified-record-with-discriminator at the schemas layer; the implementation language inherits the obligation to PREVENT category-confusion bugs from reaching production, not merely DETECT them at runtime.

**Go.** **Clean** — with the strongest by-default enforcement among the four alternatives evaluated. Go's enforcement properties:
- Package-private types (unexported identifiers, lowercase initial letter) are inaccessible from other packages at compile time. No reflection escape hatch at runtime without explicit `reflect` use (anti-pattern AP1).
- Unexported struct fields are similarly inaccessible. Cross-package mutation requires a documented boundary function.
- Go has no inheritance; struct embedding does not create subtype relationships at the type level. Categorical separation is preserved.
- Type assertions on interface types fail at compile time when types are incompatible; at runtime when the assertion is incorrect. Both surface the error before production-data-corruption.

Compared to alternatives: Python's mypy is opt-in (procedural enforcement); TypeScript's strict mode is configuration-dependent (procedural); Rust's enforcement is structurally stronger than Go's (the borrow checker + exhaustive matching) at higher operational cost; JVM languages enforce similarly to Go via visibility modifiers + Kotlin `internal` + sealed classes at higher deployment-weight cost. Go's enforcement is the *minimal default-strong* option in the evaluated set.
- *Citation:* [Charter §2.2 Forbidden Anti-Patterns L94](../../charter/constitutional-charter.md#22-epistemic-separation). RFC Constitutional Review Q1 §2.2 bullet.

### Dimension 3 — §2.3 Provenance Integrity (frozen v0.4)

§2.3 requires `oneof` exclusivity per [`§0016`](../../charter/decision-log.md) on the Assertion `subject_ref_*` fields. The implementation language's Protobuf bindings must preserve the wire-format exclusivity at the language type level.

**Go.** **Clean with discipline caveat.** Protobuf `oneof` generates a Go interface type (e.g. `isAssertion_SubjectRef`) and one struct type per branch (e.g. `Assertion_SubjectRefObservation`, `Assertion_SubjectRefConstruct`, `Assertion_SubjectRefHypothesis`). Assignment of a new branch struct replaces any prior branch; the wire format exclusivity is preserved structurally. **Discipline caveat:** the generated interface field can be `nil` (no branch set) — valid Protobuf semantics for an absent oneOf. Application code must handle the "no branch set" case explicitly when reading `subject_ref` from an Assertion (e.g. type-switch with default branch covering the nil case). Not a §2.3 violation (the wire format permits absent oneOf), but a service-tier discipline obligation worth surfacing in the canonical-serialization-contract follow-on document.
- *Citation:* [Protocol Buffers Go Generated Code Guide §oneof](https://protobuf.dev/reference/go/go-generated/#oneof). [Charter §2.3 frozen v0.4 Structural Requirement L113](../../charter/constitutional-charter.md#23-provenance-integrity). RFC Proposal item 1 (package-per-category) + item 3 (`protoc-gen-go`).

### Dimension 4 — §2.5 Hypothesis Lifecycle Explicitness (frozen v0.3)

§2.5 requires that lifecycle events are Cat I records with antecedent references. No language-specific support requirement beyond §2.1 + §2.2 inheritance.

**Go.** **Clean.** Lifecycle event types (`FormationEvent`, `MergeEvent`, `SplitEvent`, `PromotionEvent`, `DemotionEvent`, `DissolutionEvent`) as Go struct types in a `cat3lifecycle` (or similarly-named) package. Repeated antecedent references generated as Go slices from proto `repeated` fields. No language-specific concern.
- *Citation:* [Charter §2.5 Structural Requirement L162](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness). RFC Constitutional Review Q1 §2.5 bullet.

## Phase 2 — Alternative rejection re-test

Four cells. The draft RFC rejected Python + TypeScript on §2.2 compile-time-enforcement grounds; Rust + JVM as admissible-but-deferred.

### Cell A — Python re-test

**RFC's rejection rationale:** Python's structural type checking via mypy is opt-in; default duck typing does not enforce categorical separation; converting to procedural enforcement (mandatory strict-mypy + audit) weakens the structural property the Charter codifies.

**Re-test.** Holds robustly. Even with mandatory mypy strict mode + `mypy-protobuf` plugin for generated bindings, Python's typing has documented gaps: third-party libraries without type stubs propagate `Any` through call chains; `# type: ignore` comments are silently permitted; runtime `__class__` mutation can change a value's nominal type. The `mypy-protobuf` plugin produces typed Protobuf bindings but does not eliminate the surrounding-library typing gaps. The structural concern is Python-specific and not addressable by configuration alone. **Verdict on RFC's rejection:** preserved; structural rejection holds.
- *Citation:* [`mypy-protobuf` documentation](https://github.com/nipunn1313/mypy-protobuf). [PEP 484 — Type Hints](https://peps.python.org/pep-0484/) (gradual typing semantics).

### Cell B — TypeScript / Node re-test

**RFC's rejection rationale:** strict mode is configuration-dependent + npm ecosystem operational concerns (lockfile drift, transitive-dependency surface area, supply-chain risk).

**Re-test.** TypeScript's `strict: true` + `noImplicitAny: true` provides robust compile-time type checking; the configuration-dependency concern is procedural (similar to Python's mypy concern, but TypeScript's strict mode is more commonly the default in modern projects). The npm ecosystem operational concern is real and harder to bound: even with `npm ci` + lockfile pinning, the transitive-dependency graph for service-tier code can reach hundreds of packages; canonical-serialization-contract auditability (per [`§0024`](../../charter/decision-log.md) AP5) becomes harder when the surrounding ecosystem moves quickly. `ts-proto` produces TypeScript bindings with strict typing. **Verdict on RFC's rejection:** preserved on both grounds; the procedural-enforcement concern is slightly weaker than Python's, but the npm operational concern compounds.
- *Citation:* [TypeScript Strict Mode documentation](https://www.typescriptlang.org/tsconfig#strict). [`ts-proto` documentation](https://github.com/stephenh/ts-proto).

### Cell C — Rust re-test

**RFC's rejection rationale:** admissible-but-deferred. Rust's type system is structurally stronger than Go's; rejection is operational (ownership-model learning-curve, toolchain-complexity, not warranted pre-load).

**Re-test.** The structural-strength claim is correct and worth recording explicitly: Rust's borrow checker + module-private types + exhaustive `match` enforce §2.2 separation more strictly than Go (no `reflect` or `unsafe` escape hatch in safe code; `unsafe` blocks are explicit and reviewable). The `prost` crate produces deterministic canonical bytes; `protoc-gen-prost` is maintained. The operational concern (learning curve, build time, toolchain dependencies) is real at inception phase but degrades over time as the project's complexity grows. **Verdict on RFC's rejection:** preserved on operational grounds at inception; the admissible-but-deferred registration is the right rejection shape. Reversal condition R-lang-1 captures throughput/memory-safety pressure as the re-evaluation trigger.
- *Citation:* [`prost` crate documentation](https://docs.rs/prost/latest/prost/). [Rust ownership documentation](https://doc.rust-lang.org/book/ch04-00-understanding-ownership.html).

### Cell D — JVM languages re-test

**RFC's rejection rationale:** admissible-but-deferred. Compile-time enforcement strong via visibility modifiers + Kotlin `internal` + sealed classes; rejection is operational (deployment weight inappropriate at inception).

**Re-test.** Compile-time enforcement is real and Kotlin's sealed classes + `internal` modifier offer particularly clean §2.2 expression (sealed classes mirror Protobuf `oneof` semantically). Deployment weight: traditional JVM (JIT, GC tuning, multi-second startup) is a real concern for multi-service topology at inception; GraalVM native-image reduces deployment weight but adds toolchain complexity (image builds slow; reflective code requires explicit configuration). Native-image's reflection limitations could interact awkwardly with Protobuf generated code, requiring per-message-type registration. **Verdict on RFC's rejection:** preserved at inception; admissible-but-deferred is the right rejection shape. Reversal condition R-lang-3 captures JVM-ecosystem-library pressure.
- *Citation:* [Kotlin sealed classes documentation](https://kotlinlang.org/docs/sealed-classes.html). [GraalVM native-image documentation](https://www.graalvm.org/latest/reference-manual/native-image/).

## Phase 3 — Discipline skills application

Per rfc-author §3, three skills apply before `status: discussion`. Findings recorded inline.

### 3.1 falsifiability-check (V/O/Op/NC four-question test)

Applied to RFC substantive claims in Summary, Motivation, Proposal sections.

| # | Claim (paraphrased) | V | O | Op | NC | Verdict |
|---|---|---|---|---|---|---|
| FC1 | "Go's package-private types and unexported struct fields enforce categorical separation at compile time by default" (Motivation, Constitutional Review) | ✓ violation = compilation succeeds when accessing unexported field from another package | ✓ try the compile, observe `cannot refer to unexported field` error | ✓ "package-private" = Go visibility spec | ✓ Go language spec external | **Pass** |
| FC2 | "Go's `google.golang.org/protobuf` library is mature, widely tested, and supports the proto3 spec faithfully" (Motivation) | ⚠ "mature/widely tested" judgment-bound | ⚠ no localized witness | ⚠ "mature" not operationalized | ✓ vocabulary OK | **Pass-with-caveat (rewrite suggested):** rephrase as "Go's `google.golang.org/protobuf` v2 library produces byte-stable proto3 canonical serialization across processes at a pinned version, verifiable via the CI golden-file gate per §0024 AP5 (d)." |
| FC3 | "Single static binary per service simplifies deployment topology" (Motivation, Proposal item 5) | ⚠ "simplifies" comparative | ⚠ depends on comparison baseline | ⚠ "simplifies" not operationalized | ✓ vocabulary OK | **Pass-with-caveat (rewrite suggested):** rephrase as "Each service builds to a single self-contained executable with no shared-library runtime dependencies (verified by `ldd service-binary` reporting `not a dynamic executable`); deployment requires only the binary + any external configuration files explicitly named in the service `README.md`." |
| FC4 | "Python's structural type checking via mypy is opt-in" (Alternatives) | ✓ violation = Python interpreter rejects un-annotated code (it does not) | ✓ run interpreter, observe acceptance | ✓ "opt-in" = mypy is separate tool from CPython | ✓ Python language spec external | **Pass** |
| FC5 | "TypeScript's strict mode is configuration-dependent" (Alternatives) | ✓ violation = strict mode is unconditionally enforced (it is not) | ✓ inspect `tsconfig.json` setting | ✓ "configuration-dependent" = `tsconfig.json` format | ✓ TypeScript spec external | **Pass** |
| FC6 | "Rust's type system is structurally stronger than Go's for §2.2 enforcement" (Alternatives) | ⚠ "stronger" comparative | ⚠ depends on what counts as stronger | ⚠ "stronger" not fully operationalized | ✓ vocabulary OK | **Pass-with-caveat (rewrite suggested):** rephrase as "Rust enforces module-private type access without runtime escape hatches in safe code (no analog of Go's `reflect.Value.UnsafePointer()` available without `unsafe` blocks, which are explicit and reviewable); §2.2 separation has fewer non-mechanical bypass paths than in Go." |
| FC7 | "JVM deployment weight is inappropriate for inception-phase services" (Alternatives) | ⚠ "inappropriate" judgment-bound | ⚠ depends on appropriateness criterion | ⚠ "inappropriate" not operationalized | ✓ vocabulary OK | **Pass-with-caveat (rewrite suggested):** rephrase as "JVM services exhibit multi-second startup times and resident-memory baselines >100MB even for trivial services; inception-phase optimizes per-service cost (startup latency + memory footprint) over runtime-ecosystem-library availability. Revisit per R-lang-3." |

**Falsifiability summary:** 3 pass clean (FC1, FC4, FC5) + 4 pass-with-caveat (FC2, FC3, FC6, FC7 — all rewritable via operationalization of comparative/judgment-bound modifiers). No claim fails. Rewrites are discussion-phase suggestions; resolution-phase commits them.

### 3.2 epistemic-separator (paragraph-level categorical distinctness)

Applied to RFC paragraphs in Summary, Motivation, Proposal, Anti-Patterns.

- **ES1 — Pass.** Summary categorically distinct from constitutional commitment.
- **ES2 — Pass.** Motivation paragraphs cite frozen invariants explicitly; categorical structure clean.
- **ES3 — Pass.** Proposal items are concrete technology commitments; not mixed with Charter prose.
- **ES4 — Pass.** Anti-Patterns paragraphs cite §2.2 + AP-derivation; categorical placement clean. The RFC already pre-applies the schemas-RFC ES5 fix (Motivation closing sentence split into technology-cost / procedural-cost components).
- **ES5 — Pass.** Alternatives section's "rejected on §2.2 compile-time-enforcement grounds (not deferred)" vs "rejected as admissible-but-deferred" framing keeps the two rejection categories distinct.

**Epistemic-separation summary:** 5 pass; no rewrites required.

### 3.3 ambiguity-reducer (advisory term flagging)

Pre-commit hook surfaced 5 ambiguity advisories at draft-commit time. Per the ambiguity-reducer skill, advisory; per CLAUDE.md §5.3, author decides.

- All 5 are standard project terms used in canonical or common-English senses (no informal reuse of canonical vocabulary). Resolution: acceptable as-is.

**Ambiguity-reducer summary:** all 5 advisories resolve to acceptable-as-is on inspection.

## Phase 4 — Non-obvious risks and findings

Numbered for discussion-phase tracking.

### F1 — `unsafe` package as load-bearing anti-pattern

The RFC's AP1 names `reflect` + `unsafe` together as cross-category-import bypass paths. The actual mechanism worth surfacing explicitly: Go's `unsafe.Pointer` permits constructing a value of any type from raw bytes; `reflect.Value.UnsafePointer()` returns a `unsafe.Pointer` to a reflected value. Either mechanism can construct a categorically-typed struct value with arbitrary internal state, bypassing the package-private boundary. **Action for resolution phase:** AP1 expanded to enumerate both `unsafe.Pointer` and `reflect.Value.UnsafePointer()` (+ analogous `reflect.NewAt`); CI lint rule on both in category-package service code.

### F2 — Build reproducibility commitment

Go's `go build` produces binary output that includes path information (build directory, GOPATH); two builds of identical source in different directories produce different binaries. Reproducible builds matter for canonical-serialization-contract auditability per [`§0024`](../../charter/decision-log.md) AP5: a CI golden-file test against a non-reproducible build cannot distinguish library-version-regression from build-environment-drift. **Action for resolution phase:** Proposal commits to `go build -trimpath` + `GOFLAGS="-trimpath"` for service binaries; reproducible-build verification step in CI.

### F3 — Module proxy commitment for supply-chain hygiene

Go modules are fetched from `proxy.golang.org` by default; transitive dependencies can be added without explicit consent. **Action for resolution phase:** add Open Question — should the project commit to a private module proxy (e.g. self-hosted Athens, JFrog, or vendored modules via `go mod vendor`)? At inception phase, the public proxy is operationally sufficient; the question becomes load-bearing as the project's dependency surface area grows.

### F4 — Concurrency-pattern document timing

The RFC's Open Questions name "concurrency model conventions" as a follow-on architecture document. **Finding:** the document should land BEFORE the first non-trivial service (i.e. before any service that uses more than a single goroutine), not after. Reason: convention conflicts surfaced after multi-goroutine code exists are harder to refactor than conventions set before. **Action for resolution phase:** add to Open Questions with timing clarification.

### F5 — `go.work` vs single `go.mod` for multi-service topology

The RFC's Open Questions name module organization (single-module vs multi-module) as a follow-on. **Finding:** the choice has empirical consequence — single-module simplifies cross-service refactoring but couples release cadence; multi-module decouples release but requires `go.work` or explicit replace-directive management. At inception with a single ingestion service, single-module is operationally simplest; the question becomes load-bearing as the second service is added. **Action for resolution phase:** the follow-on RFC should be triggered by the second-service decision, not by procedural cadence.

### F6 — Cross-RFC coupling to inception-phase storage-technology RFC

The RFC's Open Questions surface this explicitly (Go's database-driver ecosystem constrains storage-RFC option space). **Finding:** the constraint is asymmetric — Go has mature drivers for SQLite (`modernc.org/sqlite` pure-Go; `mattn/go-sqlite3` CGo), PostgreSQL (`jackc/pgx`), and most other common embedded + server stores. The constraint is *which kinds of storage* (relational, key-value, log-structured, columnar) have well-maintained Go drivers, not which specific product within a category. The storage-RFC discussion phase inherits this finding as input. **Action:** no immediate action; documented for storage-RFC inheritance.

## Phase 5 — Recommendation with reversal conditions

**Recommendation (for resolution-phase consideration):** accept the draft's selection of **Go (latest stable)** as implementation language, with the following modifications enacted in the resolution-phase commit:

1. **Rewrite FC2/FC3/FC6/FC7 per Phase 3.1** — operationalize comparative/judgment-bound modifiers ("mature", "widely tested", "simplifies", "stronger", "inappropriate").
2. **Expand AP1 per F1** — enumerate `unsafe.Pointer`, `reflect.Value.UnsafePointer()`, `reflect.NewAt` as named bypass mechanisms; CI lint rule.
3. **Add build-reproducibility commitment per F2** — Proposal item: `go build -trimpath` + `GOFLAGS="-trimpath"`; CI reproducible-build verification.
4. **Add module-proxy Open Question per F3** — surface for committee deliberation at inception phase vs follow-on.
5. **Clarify concurrency-pattern document timing per F4** — document lands before first multi-goroutine service.
6. **Clarify module-organization trigger per F5** — follow-on RFC triggered by second-service decision, not procedural cadence.

**Reversal conditions** (when to revisit this selection via subsequent RFC):

- **R-lang-1.** Throughput characterization at production-relevant scale demonstrates that Go's GC pause time or runtime overhead dominates the ingest path. Threshold: GC pauses >5ms p99 at characterized load OR runtime allocation rate >50% of single-message CPU. (Revisit Rust under throughput/safety pressure.)
- **R-lang-2.** A service-tier emerges where Node.js-specific properties (e.g. browser interop for an operator-facing tool, V8 runtime properties for a specific use case) become load-bearing. Threshold: explicit RFC proposing the service-tier and demonstrating the property is load-bearing. (Revisit TypeScript/Node for that service-tier.)
- **R-lang-3.** A service-tier emerges where JVM-ecosystem libraries (specific stream-processing framework, ML library) become load-bearing. Threshold: explicit RFC proposing the service-tier and demonstrating the library is load-bearing. (Revisit JVM languages for that service-tier.)
- **R-lang-4.** §2.2 compile-time enforcement proves insufficient at scale — recurring category-confusion bugs surface despite the package-per-category discipline + AP1 lint rules. Threshold: two empirically observed §2.2 enforcement failures within a six-month window. (Revisit Rust or another stricter-by-default language.)

No reversal condition fires today; the recommendation stands for resolution-phase deliberation.

## Phase 6 — Carry-forwards

- Inception-phase storage-technology RFC (third of three per [`§0022`](../../charter/decision-log.md)) inherits F6 — cross-RFC coupling to Go's database-driver ecosystem; discussion phase opens with this finding as input.
- No new canonical project vocabulary introduced. Glossary unchanged.
- No Ontology document revision required.
- No Charter amendment required (per Constitutional Review Q4).
- The follow-on tooling RFC (`buf generate` vs `protoc`, `golangci-lint` configuration) may bundle the concurrency-pattern document (F4) and module-organization decision (F5), or those may be separate; that scoping decision is itself a discussion-phase question for the tooling RFC.
- The procedural-divergence finding from [`§0024`](../../charter/decision-log.md) committee extension 2 (README staleness + rfc-author §4 vs actual practice) remains carried forward unresolved; this RFC follows the same actual-practice pattern (file remains in `draft/` after acceptance; no renumbering).
