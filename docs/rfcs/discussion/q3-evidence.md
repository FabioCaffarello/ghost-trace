# Q3 — Subject Reference Polymorphism — discussion evidence

**Status:** in-discussion. Not authoritative. Final resolution will be recorded in decision-log and Ontology document revision.

This document records the discussion-phase evidence for Ontology Open Question 2 (Subject Reference Polymorphism, post-Q1-renumbering) per the RFC [`ontology-revision-q3-subject-ref-polymorphism`](../draft/ontology-revision-q3-subject-ref-polymorphism.md) opened as cascade enactment per [`decision-log §0015`](../../charter/decision-log.md) (Q1 resolution + §0014 cascade trigger). Fourth Ontology RFC discussion phase (Q2-1, Q4-1, Q1-1 precede); first RFC opened as cascade enactment of another RFC's resolution. Five phases, structurally parallel to Q1-1, Q2-1, Q4-1.

---

## Phase 1 — Evidence per dimension per candidate

Seven dimensions evaluated. Two candidates from the RFC's Proposal section; Candidate C is rejected in the RFC and does not appear here. Dimension 7 (§2.2 compliance) is the load-bearing dimension; mirrors Q1-1's structure for the same reason — §2.2 is the frozen invariant most directly tested.

### Candidate A — Single polymorphic `subject_ref` with `subject_type` discriminator

**D1 — Schemas / validation surface.** Two fields per assertion: `subject_ref` (opaque ID type) + `subject_type` (enum). Validation surface: type discipline on `subject_ref`; enum validation on `subject_type`; schemas-level rule that subject_ref's referent's declared category equals subject_type value. Field count is minimal (2). Source: [Q3 RFC Cand A Structural claim](../draft/ontology-revision-q3-subject-ref-polymorphism.md).

**D2 — Query patterns.** "Find all assertions about hypothesis H": single query filtering by `subject_type = 'hypothesis' AND subject_ref = H`. Uniform query shape; one indexed-by-discriminator scan. "Find all assertions about any subject" reads single field. Source: Q3 RFC Cand A Pros.

**D3 — Type-system enforcement of cross-category constraints.** Constraint that "this assertion can only reference Cat I observations" lives in schemas validation rule (e.g., `subject_type ∈ {observation, declared_session}`), NOT in the type-system structure. Validation enforces; type-system permits any discriminator value structurally. Source: Q3 RFC Cand A Cons.

**D4 — Provenance graph shape.** Edges are uniform type `subject_ref`. Category determined by discriminator at the edge endpoint. Traversal: read endpoint's discriminator to determine category. Single edge type → simpler graph schemas; discriminator-aware traversal logic required.

**D5 — Forward-look to §2.3 / §2.4 binding text.** §2.3 binding text: "Every assertion declares observational provenance via `subject_ref` field with `subject_type` discriminator; Cat I references have subject_type values ∈ {`observation`, `declared_session`, ...}." §2.4: "Inferential influence references via subject_ref + subject_type." Prose must specify discriminator semantics; binding text is longer.

**D6 — §2.5 frozen v0.3 compatibility.** §2.5 does not name `subject_ref` directly (verified: §2.5 uses "antecedents", "reference-and-parameters payload", "produced hypothesis"). §2.5 BC5 calls the payload "content-as-reference (specific hypothesis, operation, antecedents, parameters)" — implementation-agnostic. **Accommodated:** under A, BC5's "specific hypothesis" reference is a subject_ref + subject_type=hypothesis pair. Source: §2.5 BC5 verbatim.

**D7 — §2.2 compliance — the load-bearing dimension.** Two sub-questions:

- **Sub-question 7.1:** Is the discriminator schemas-level, write-time, type-determining? IF YES (schemas validates subject_ref's referent category equals subject_type at write time; subject_type immutable post-commit): §2.2 respected — discriminator IS the typed boundary §2.2 requires for cross-category references. The category is fixed at construction (per §2.2: "declared at construction and not changeable").
- **Sub-question 7.2:** Could the discriminator be implemented as a runtime-classification marker? If the discriminator-to-referent-category validation is weak or post-commit, A collapses toward rejected Candidate C (runtime classification).

**Verdict:** A's §2.2 compliance is **CONDITIONAL** on schema-technology implementation enforcing the discriminator as type-determining at write time. Source: §2.2 frozen Structural Requirement; Q3 RFC Cand A Cons.

### Candidate B — Distinct `subject_ref_X` fields per category

**D1 — Schemas / validation surface.** Three category-specific fields at minimum: `subject_ref_observation` (Cat I primaries incl. `DeclaredSession`), `subject_ref_construct` (Cat II constructs incl. `OperationalSession`), `subject_ref_hypothesis` (Cat III subtypes). Validation: per-field type discipline + exactly-one-populated constraint (oneOf at schemas-level OR type-system union). Larger field count (≥3).

**D2 — Query patterns.** "Find all assertions about hypothesis H": single field query `WHERE subject_ref_hypothesis = H`. Clean type-safety at query time. "Find all assertions about any subject" unions across 3 fields — polymorphism at query, not at field-level.

**D3 — Type-system enforcement of cross-category constraints.** Constraint "this assertion can only reference Cat I observations" is enforced at the type-system: the assertion type has ONLY `subject_ref_observation` populated; absence-of-other-fields is type-level. **Structurally enforceable.** Source: Q3 RFC Cand B Pros.

**D4 — Provenance graph shape.** Edges are typed by category (subject_ref_observation, subject_ref_construct, subject_ref_hypothesis edges). Edge type directly indicates referent category. Traversal: edge type IS the category signaling. Multiple edge types → richer graph schemas; type-aware traversal natural.

**D5 — Forward-look to §2.3 / §2.4 binding text.** §2.3 binding text: "Every assertion declares observational provenance via `subject_ref_observation` field (referencing Cat I primaries including `DeclaredSession`)." §2.4: "Inferential influence references via `subject_ref_construct` (Cat II) or `subject_ref_hypothesis` (Cat III)." Prose is structurally simpler per binding section; no discriminator semantics to specify.

**D6 — §2.5 frozen v0.3 compatibility.** §2.5 BC5 "content-as-reference" payload: under B, the "specific hypothesis" reference is `subject_ref_hypothesis` field. §2.5 AP3 ("Hypothesis-merge events without recorded antecedents"): antecedents field populated via subject_ref_hypothesis (typed). **Accommodated:** §2.5's implementation-agnostic phrasing accommodates B's typed fields directly.

**D7 — §2.2 compliance.** **Type-level structural separation.** The category boundary IS the field choice at construction. No discriminator needed; no validation question about enforcement of category. §2.2's "declared at construction and not changeable" is the typing itself; per-category fields directly exemplify §2.2's typed-boundary requirement. Conditional only on schema-level exactly-one-populated enforcement (oneOf constraint or type-system union). **Verdict:** B's §2.2 compliance is **CONDITIONAL** on schemas enforcing exactly-one-populated, but the condition is simpler than A's (one constraint vs A's discriminator-validation matrix). Source: §2.2 frozen; Q3 RFC Cand B Pros + Cons.

### Observation 1 — Strongest asymmetry

**Dimension 7 (§2.2 compliance) is the strongest asymmetry.** Specifically Sub-question 7.1:

- **Under A**, §2.2 compliance hinges on whether the discriminator is schemas-level type-determining (compliant) OR runtime-classification marker (collapses toward rejected Candidate C). The line between A and C is precisely whether the discriminator is structurally enforced at write time. This is the **deliberation crux** — A's §2.2 compliance is **mechanism-dependent**.
- **Under B**, §2.2 compliance is **type-structurally exemplified** — per-category fields ARE §2.2's typed boundary. Compliance is **mechanism-clean** but conditional on a different mechanism: exactly-one-populated enforcement. This is a single constraint, not a matrix.

Both candidates are conditional on schema-technology implementation — but the conditional shape differs. A's conditional is complex (discriminator-validation matrix per category); B's conditional is localized (oneOf/union constraint). This is the **structural-versus-procedural compliance distinction** that paralleled Q1-1's Finding 1.

### Observation 2 — Genuine trade-offs vs apparent

**Genuine trade-offs:**
- **D1 (validation surface):** A simpler (2 fields), B richer (3+ fields). Real cost trade-off.
- **D2 (query patterns):** A uniform shape, B clean type-safety per category. Real workload trade-off.
- **D4 (provenance graph):** A uniform-edge, B typed-edge. Real graph-schemas trade-off.

**Apparent trade-offs that resolve:**
- **D3 (type-system enforcement):** B's structural advantage is real and not balanced by A's discriminator validation (which lives one layer down). Resolves to B advantage.
- **D5 (binding text shape):** Both produce concrete §2.3/§2.4 binding text; A's prose is longer because it must specify discriminator semantics. Not a balance — B's is simpler per binding section.
- **D6 (§2.5 compatibility):** Both fully accommodate. §2.5's shape-agnosticism resolves this as a non-trade-off. Confirmed.

**Mechanism-dependent (the deliberation crux):**
- **D7 (§2.2 compliance):** Both conditional. A's discriminator-vs-runtime-classification distinction is the structural deliberation; B's exactly-one-populated is a simpler structural commitment.

---

## Phase 2 — Scaffold check, post-Q1 state

### F-SCAFFOLD-1 — entity-model.md §Open Modeling Question 2 (post-renumbering) framing

[`entity-model.md` L89](../../ontology/entity-model.md):

> "**Subject reference polymorphism.** Assertions carry a `subject_ref` that may point to entities of any category. Whether this is a single polymorphic field or distinct fields per category is a type-level question with ontological consequences. Resolution opened as RFC `ontology-revision-q3-subject-ref-polymorphism` (`discussion` status) per `decision-log §0015` cascade trigger."

Verbatim analysis:

- **"single polymorphic field or distinct fields per category"** — names both candidates explicitly. The binary framing is present and clean.
- **"type-level question with ontological consequences"** — neutral characterization; does not commit to which candidate the consequences favor.
- Order: "single polymorphic" first, "distinct fields" second. Purely positional; no semantic lean.
- The reference to Q3 RFC + §0015 cascade trigger is administrative parenthetical, not editorial leaning.

**F-SCAFFOLD-1 verdict: NEUTRAL.** The Open Modeling Q2 framing mirrors Q1-1's "neutral binary framing" pattern.

### F-SCAFFOLD-2 — Q1's resolution effect on Q3 scaffold lean

The post-Q1 state of `entity-model.md` includes typed-distinct `DeclaredSession` (Cat I) + `OperationalSession` (Cat II) types. Three sub-questions:

**(a) Does Q1-B's typed-distinct entity types imply typed-distinct subject_ref fields (Q3-B)?**

Argument *for* implicit pressure toward Q3-B (symmetry): if Cat I and Cat II Session forms are typed-distinct at the ENTITY level, references-to-them should also be typed-distinct at the REFERENCE level.

Argument *against* implicit pressure: Q1's type-level distinction is at the entity level (the records themselves are typed). Q3's question is at the reference level (how assertions point to records). Both Q3 candidates preserve Q1's typed entity distinction:
- Under Q3-A, the discriminator field carries the category distinction. `subject_type = declared_session` (Cat I) versus `subject_type = operational_session` (Cat II) preserves Q1's typed entity distinction at the reference level via discriminator value.
- Under Q3-B, the field choice (`subject_ref_observation` vs `subject_ref_construct`) preserves Q1's typed entity distinction at the field level.

Q1's resolution does NOT mandate Q3's shape — both candidates accommodate Q1-B equivalently.

**(b) Does the post-Q1 Category I/Cat II revision prose imply a Q3 shape?**

Category I L16 (`DeclaredSession`) and Category II L33 (`OperationalSession`) introduce these as type names. The prose does NOT reference `subject_ref` or commit to its shape.

**(c) Does §2.5 frozen v0.3's subject_ref pattern (if any) commit to Q3?** Treated in F-SCAFFOLD-4 below.

**F-SCAFFOLD-2 verdict: INHERITED-PRESSURE WITHOUT INHERITED-LEAN.** Q1's resolution created the *trigger condition* for Q3 (subject_ref to Sessions is now ambiguous without resolution per §0014 cascade) but did NOT create *shape pressure* toward either Q3 candidate. The scaffold remains shape-agnostic on Q3 post-Q1.

### F-SCAFFOLD-3 — provenance-model.md scaffold agnosticism

Empirical scan of `provenance-model.md` post-Q1:

- §Observational Provenance opening (L13–15): generic — "Every non-observation assertion in Ghost Trace declares, in its structure, the set of observations from which it was computed." No subject_ref mention.
- §Observational Provenance "To be formalized" list (L17–20): "The representation of observation references in assertions" (L18) — abstract.
- §Observational Provenance post-Q1 paragraph (added Q1-2): uses arrow notation `assertion → OperationalSession → DeclaredSession + other Cat I primaries`. The arrow is implementation-neutral.
- §Inferential Provenance (L22–31): abstract language about "prior assertions" and "influence."
- §The Provenance Graph (L33–45): names edge varieties at the relation level (`derived_from`, `influenced_by`), not at the subject_ref field level.

**F-SCAFFOLD-3 verdict: AGNOSTIC.** Fully Q3-agnostic at the prose level.

### F-SCAFFOLD-4 — §2.5 frozen v0.3 verbatim references to subject_ref

Empirical scan of §2.5 frozen v0.3 (entire section):

**Definition:** No `subject_ref` mention. Abstract reference machinery.
**Structural Requirement:** No `subject_ref` mention. Uses "Merge events reference all antecedent hypotheses"; "Promotion events carry the structural parameters"; "lifecycle event referencing the prior promotion event" — all abstract.
**Forbidden Anti-Patterns:** No `subject_ref` mention. AP3 uses "antecedent reference"; AP5 uses "antecedent subtypes differ".
**Boundary Conditions:** No `subject_ref` mention. BC5: "content-as-reference (specific hypothesis, operation, antecedents, parameters)" — implementation-agnostic.

**Total count of `subject_ref` occurrences in §2.5 frozen v0.3: 0.**

The reference machinery is described in §2.5 entirely abstractly: "antecedents", "antecedent reference", "reference-and-parameters payload", "content-as-reference", "produced hypothesis". None of these phrasings commits to Q3 shape.

**F-SCAFFOLD-4 verdict: §2.5 IS FULLY SHAPE-AGNOSTIC ON Q3.** High-consequence finding for the discussion phase: Q3-1's recommendation is NOT pre-determined by frozen Charter text.

### Methodological observation — fourth scaffold state

§0015 obs 3 documented three scaffold-finding states (implicit-lean, explicit-defer, agnostic). Q3-1 surfaces a fourth state:

**Inherited-pressure-without-inherited-lean.** A post-resolution scaffold state in which an earlier RFC resolution (here: Q1's resolution to Candidate B per §0015) creates the *trigger condition* for an open question (Q3) without committing to its *shape*. The scaffold remains shape-agnostic post-resolution; the pressure is to-resolve, not to-shape.

Methodologically distinct from:
- *Implicit-lean* (Q2-1, Q4-1): scaffold prose presupposes one answer for fluency.
- *Explicit-defer*: scaffold explicitly defers the question.
- *Agnostic* (Q1-1): pre-resolution scaffold names both candidates without leaning.

Inherited-pressure adds a post-resolution dimension. The taxonomy now distinguishes pre-resolution states from post-resolution states.

### Summary table

| Finding | Verdict | Strength |
|---|---|---|
| F-SCAFFOLD-1 (Open Modeling Q2 framing) | Neutral; binary framing names both candidates without lean | Strong |
| F-SCAFFOLD-2 (Q1's resolution effect) | Inherited-pressure WITHOUT inherited-lean | Substantive |
| F-SCAFFOLD-3 (provenance-model.md) | Fully Q3-agnostic at prose level | Strong |
| F-SCAFFOLD-4 (§2.5 frozen v0.3 subject_ref count) | §2.5 mentions `subject_ref` **0 times**; fully shape-agnostic | Strong — high-consequence |

**Overall scaffold verdict: Q3 remains genuinely open.**

---

## Phase 3 — Epistemic findings

Six cells from 2 candidates × 3 skills.

### Candidate A × `falsifiability-check`

Applying §1 four-question test to "Each assertion carries `subject_ref` + `subject_type` discriminator; the discriminator is schemas-level enforced at write time as type-determining."

- **V (Violation).** Falsifying state: an assertion exists where `subject_type`'s value disagrees with `subject_ref`'s referent's declared category. Detectable by schemas-level validation rule.
- **O (Observation).** Third party reads assertion, dereferences `subject_ref`, checks referent's category, compares against `subject_type`. Mechanical.
- **Op (Operationalization).** "`subject_ref`" → opaque ID field. "`subject_type`" → enum field. "Polymorphism" → schemas-level union with discriminator. The exact schema-technology choice is deferred per [`decision-log §0003`](../../charter/decision-log.md). **Pass with conditional deferral.**
- **NC (Non-circularity).** Reduces to existing glossary terms. Non-circular.

**Verdict:** Pass with conditional deferral.

### Candidate A × `epistemic-separator`

- **Naming:** `subject_ref` (generic), `subject_type` (category-named enum).
- **Operation validity:** Read references; valid across categories.
- **Typed crossing — load-bearing question:** A's discriminator-based crossing IS a typed transformation IF AND ONLY IF the discriminator is schemas-enforced as type-determining at write time. Compare to rejected Candidate C:
  - **A's discriminator (schemas-enforced at write time):** validation rule enforced at construction; `subject_type` immutable post-commit. Category FIXED at construction → matches §2.2's "declared at construction and not changeable". **Compliant.**
  - **C's classification (runtime, rejected):** category determined at READ time. **Non-compliant.**

  A is structurally distinct from C if and only if the discriminator's enforcement is at write time, schemas-mandated, and the discriminator value is immutable.

- **Skill §4 forbidden constructions:** A's discriminator does NOT match construction #1 ("annotate event with inferential content"). The discriminator names the *referent's category*, not the assertion's inferential content.

**Verdict:** Pass **CONDITIONAL** on schemas-level write-time discriminator enforcement. **Load-bearing finding.**

### Candidate A × `ambiguity-reducer`

Watchlist hits: `record`, `evidence`, `decision`, `context`. 4 advisory hits.

**Verdict:** 4 advisory hits (non-blocking).

### Candidate B × `falsifiability-check`

Applying §1 four-question test to "Each assertion carries category-specific subject_ref fields with exactly-one-populated enforced at schemas/type-system level."

- **V (Violation).** Falsifying states: multiple fields populated simultaneously; or no field populated.
- **O (Observation).** Third party reads assertions; checks field-population pattern.
- **Op (Operationalization).** "subject_ref_X" → typed reference fields per category. "Exactly-one-populated" → schemas `oneOf` constraint OR type-system union. Same schema-technology deferral as A. **Pass with conditional deferral.**
- **NC (Non-circularity).** Non-circular.

**Verdict:** Pass with conditional deferral.

### Candidate B × `epistemic-separator`

- **Naming:** `subject_ref_observation`, `subject_ref_construct`, `subject_ref_hypothesis` — category-typed explicitly. ✓✓
- **Operation validity:** Each field is a category-specific read reference. ✓
- **Typed crossing:** B's per-category fields **DIRECTLY EXEMPLIFY** §2.2's typed transformation requirement. The cross-category reference machinery is the field-choice itself; no discriminator needed.
- **Skill §4 forbidden constructions:** None match B's pattern.

**Verdict:** Pass cleanly and exemplary. Conditional only on schemas enforcement of exactly-one-populated.

### Candidate B × `ambiguity-reducer`

Watchlist hits: similar to A. 4 advisory hits.

**Verdict:** 4 advisory hits (non-blocking).

### 2×3 epistemic matrix summary

| Skill | Candidate A | Candidate B |
|---|---|---|
| `falsifiability-check` | Pass with conditional deferral (schema-technology per §0003) | Pass with conditional deferral (same dependency) |
| `epistemic-separator` | Pass **CONDITIONAL** on schemas-level write-time discriminator enforcement; collapses toward rejected Candidate C if not enforced | Pass cleanly and exemplary; §2.2's typed-boundary requirement met by structural construction |
| `ambiguity-reducer` | 4 advisory hits | 4 advisory hits |

### Most consequential epistemic finding

**Candidate A's `epistemic-separator` cell is the load-bearing finding.** The discriminator-vs-rejected-Candidate-C distinction.

The discriminator under A is structurally distinct from C IF AND ONLY IF schema-technology supports write-time enforced discriminator validation. The required mechanism is concrete: schemas-level rule, mandatory at construction, with immutable `subject_type` post-commit.

Schema technologies that support this: JSON Schema (`discriminator` + `oneOf`); Avro (discriminated union); Protobuf (`oneof`). Schema technologies that don't: pure key-value stores; schemas-less stores. [`decision-log §0003`](../../charter/decision-log.md) (substrate-technology selection deferred) is the locus.

**Two structural points emerge:**

1. **A's compliance is mechanism-dependent.** Without schema-technology supporting write-time discriminator enforcement, A is not §2.2-compliant; it collapses toward rejected Candidate C.

2. **B's compliance is mechanism-clean.** B requires the same schema-technology support (oneOf/union) but the structural form is simpler — single constraint, not a matrix.

---

## Phase 4 — Comparison synthesis

### Findings consolidated, numbered by consequence

**Finding 1 — §2.2 compliance shape asymmetry: mechanism-dependent (A) vs mechanism-clean (B).** Apparent trade-off that resolves. Both conditional on schema-technology choices (§0003 deferred), but the conditional shape differs structurally:
- **A's conditional is matrix-shaped:** schema-technology must support write-time enforcement, PLUS subject_type immutable post-commit, PLUS mandatory validation. Without all three, A collapses toward rejected Candidate C.
- **B's conditional is single-point:** schema-technology must support exactly-one-populated enforcement (one oneOf constraint per assertion type).

The asymmetry mirrors Q1-1's Finding 1 structurally. Sources: Phase 1 D7; Phase 3 epistemic-separator findings.

**Finding 2 — Type-system enforcement of cross-category constraints structurally exemplifies §2.2 under B.** Asymmetry favoring B. B's per-category fields ARE §2.2's typed transformation requirement at the field level. A's enforcement lives at the schemas-validation layer (one tier below the type-system). Sources: Phase 1 D3; Phase 3 epistemic-separator findings.

**Finding 3 — Scaffold is fully Q3-agnostic post-Q1.** Methodologically novel: post-resolution scaffold state with inherited-pressure-without-inherited-lean. §2.5 frozen v0.3 doesn't mention `subject_ref`; provenance-model.md is agnostic; Open Modeling Q2 framing names both candidates neutrally. **Both candidates remain genuinely open as constitutional matters.** Sources: Phase 2 F-SCAFFOLD-1/2/3/4.

**Finding 4 — Validation surface and field count asymmetry favors A.** Genuine trade-off. A: 2 fields per assertion; B: 3+ fields with oneOf constraint. The asymmetry is modest — neither candidate's surface is unworkable. Sources: Phase 1 D1.

**Finding 5 — Binding text shape asymmetry favors B's simplicity.** Apparent trade-off that resolves. §2.3 binding text under A must specify discriminator semantics; under B, simpler "per-category subject_ref_X field" prose. Sources: Phase 1 D5.

**Finding 6 — Query patterns asymmetric.** Genuine trade-off. A: uniform query shape. B: clean type-safety at query. Both work; neither dominates. Trade-off depends on query workload. Sources: Phase 1 D2.

**Finding 7 — Provenance graph shape asymmetric.** Genuine trade-off. A: uniform edges with discriminator. B: typed edges by category. Both work. Sources: Phase 1 D4.

### §2.2 compliance verdict per candidate

**Candidate A:**

**Sub-question 7.1:** Compliant **CONDITIONAL** on three concrete schema-technology requirements:
- Write-time discriminator validation (rule: `subject_type == referent's category` enforced at construction).
- `subject_type` field immutable post-commit.
- Schemas validation MANDATORY at construction.

If all three hold: A respects §2.2 by structural discriminator encoding. If any fail: A weakens toward rejected Candidate C.

**Sub-question 7.2:** Under strict A: NO (discriminator structurally fixed at construction). Under loose A: YES (weakens to C).

**Verdict for A:** Compliant in STRICT form only. Resolution phase MUST commit to strict form.

**Candidate B:**

Type-level structural separation: Compliant **STRUCTURAL** by construction. Each subject_ref_X is category-typed by definition.

Conditional on: schemas enforcement of exactly-one-populated (oneOf or union).

**Verdict for B:** Compliant by STRUCTURAL CONSTRUCTION. Conditional only on a single oneOf/union constraint.

**Asymmetry:** Both conditional. A's conditional is matrix-shaped; B's is single-point. Mirrors Q1's procedural-vs-structural compliance distinction.

### Discriminator structural form under A (if A is recommended)

If A is the recommendation, the resolution phase MUST specify:

1. **Schemas-level discriminator field with mandatory validation.** JSON Schema with `discriminator` + `oneOf`; Avro discriminated union; Protobuf `oneof`. Schema technologies without write-time discriminator support are incompatible with A's strict form.
2. **Immutability of `subject_type` post-commit.** Inherits from §2.1 substrate immutability.
3. **Discriminator validation MANDATORY at construction.** Not optional.

### Per-assertion exclusivity under B (if B is recommended)

If B is the recommendation, the resolution phase MUST specify:

1. **Schemas oneOf constraint** at assertion-type level. JSON Schema `oneOf`; Avro union; Protobuf `oneof`.
2. **Schemas validation enforces exactly-one constraint at construction.** Without enforcement, multiple subject_ref_X populated would be possible.

Schema-technology dependency (§0003) constrains both; oneOf/union is more broadly supported across substrate-technology options than discriminator-with-validation, making B's compliance more robust across §0003 outcomes.

### Q1-B interaction

Both candidates handle Q1-B's typed-distinct Session forms cleanly:

- **Under A:** Sessions are referenced via `subject_ref` + `subject_type` discriminator. `subject_type = declared_session` (Cat I) or `subject_type = operational_session` (Cat II). Q1-B preserved at reference site via discriminator validation.
- **Under B:** Sessions are referenced via category-specific fields. `subject_ref_observation` for DeclaredSession; `subject_ref_construct` for OperationalSession. Q1-B preserved at reference site via field-level typing.

**Implicit structural alignment:** Q1-B and Q3-B share a "typed-distinct" structural pattern (typed entities in Q1-B + typed reference fields in Q3-B). This alignment is NOT scaffold-leaning (per F-SCAFFOLD-2) but is a synthesis-phase observation about structural consistency.

### Convergence shape

**Candidate-level convergence (toward B).** Distinct from prior gates:

- **Q1-1 (meta-commitment-level):** Both candidates required the SAME mechanism (determinism for Cat II). Mechanism identical.
- **Q4-1 (form-level):** Convergence on form (staged combination) with parameters deferred.
- **Q2-1 (candidate-level):** Direct selection of one candidate (with sub-resolution A.2).
- **Q3-1 (candidate-level, toward B):** Both candidates require schema-technology support (§0003 dependency), but MECHANISMS differ structurally (discriminator-validation vs oneOf/union).

Q3 is candidate-level (like Q2-1) with the recommendation directional toward B.

### Summary statement

Based on evidence, the recommendation **toward Candidate B** is supported by three convergent asymmetries:

- **F1:** B's §2.2 compliance is mechanism-clean (single oneOf constraint); A's is mechanism-dependent (discriminator-validation matrix).
- **F2:** B's type-level enforcement of cross-category constraints structurally exemplifies §2.2; A's enforcement lives one tier below the type-system.
- **F5:** §2.3/§2.4 binding text under B is simpler.

Counter-evidence (F4 surface, F6/F7 trade-offs) is real but does not dominate. The committee at Q3-2 faces a values trade-off between structural compliance robustness (favors B) and surface minimality (favors A).

**Convergence:** candidate-level (toward B).

---

## Phase 5 — Recommendation

### Recommendation

**Candidate B — distinct per-category `subject_ref_X` fields — is recommended toward by the discussion-phase evidence**, based on three convergent asymmetries:

- **Finding 1 (§2.2 compliance shape):** B's compliance is mechanism-clean (single oneOf/union constraint per assertion type); A's is mechanism-dependent (matrix of discriminator-validation + immutability + mandatory-validation conditions). The asymmetry mirrors Q1-1's procedural-vs-structural compliance distinction; B exposes §2.2 compliance structurally, A requires schema-technology mechanism alignment.
- **Finding 2 (type-system enforcement):** B's per-category fields directly exemplify §2.2's typed-boundary requirement at the field level. A's enforcement lives at the schemas-validation layer, one tier below the type-system.
- **Finding 5 (binding text simplicity):** §2.3 and §2.4 binding text under B is structurally simpler — no discriminator semantics to specify at Charter level.

The required mechanism for B's compliance: schemas `oneOf` constraint OR type-system union at assertion-type level enforcing exactly-one `subject_ref_X` populated at construction. Schema-technology dependency on [`decision-log §0003`](../../charter/decision-log.md) remains shared with Candidate A; B's mechanism is more broadly supported across substrate-technology options than A's mechanism (discriminator-with-validation), making B's compliance more robust across §0003 outcomes.

### What would reverse this recommendation

1. **§0003 schema-technology incompatibility with oneOf/union enforcement.** If the eventual substrate technology pick does not support exactly-one-populated enforcement structurally — rare for production substrate-technology options but possible if a minimalist key-value substrate is selected — B's compliance fails. A's discriminator-validation, while complex, can be implemented at the application validation layer above a minimal KV substrate. Under this condition, A becomes the more pragmatic candidate despite its mechanism-dependence.

2. **Workload evidence that cross-category queries dominate.** If the operational workload is dominated by "find all assertions about any subject" queries (cross-category, requiring polymorphic resolution) rather than "find all assertions about hypothesis H" queries, A's uniform query shape is significantly more efficient. The query pattern dimension would weigh more heavily. Currently no workload evidence exists (pre-implementation).

3. **Q3-2 hybrid extension from committee deliberation.** Following Q1-2's precedent (committee extensions on determinism commitment + identity-tier consistency default; Q2-2's A.2 sub-resolution; Q4-2's AND-composition), the resolution phase may surface a hybrid form. Example: **B-with-typed-fields-AND-a-derived-discriminator-projection**, where the substrate stores typed `subject_ref_X` fields per assertion (B's structural compliance) but a query-layer projection materializes a polymorphic discriminator-tagged view for cross-category queries (A's query advantage). The hybrid preserves B's structural §2.2 compliance while gaining A's query uniformity at the projection layer. If such a hybrid emerges grounded in Phase 1 findings (particularly F6 + F4), this recommendation is superseded.

4. **Evidence that A's strict-form mechanism is robustly enforceable across §0003 candidate technologies.** If the §0003 candidate technology set is restricted to schemas uniformly supporting write-time discriminator validation + immutability + mandatory validation (e.g., Protobuf-only substrate), A's strict form becomes mechanism-clean rather than mechanism-dependent. Under that condition, A's compliance robustness parity with B is restored; the trade-off shifts back to F4 (surface) and F6 (queries).

### §2.3 binding text shape under this recommendation

Under Candidate B (recommended toward), §2.3 binding text takes the following shape:

- **"Back to observations" clause.** "Every assertion declares observational provenance via `subject_ref_observation` field referencing Cat I primaries (which includes `DeclaredSession` per [`decision-log §0015`](../../charter/decision-log.md))."
- **Cross-category references (anticipating §2.4 binding text).** Inferential influence references via `subject_ref_construct` (Cat II, including `OperationalSession`) or `subject_ref_hypothesis` (Cat III subtypes per Q2-A.2).
- **Provenance graph shape.** Typed edges by category — `subject_ref_observation` as observational-provenance edge; `subject_ref_construct` and `subject_ref_hypothesis` as inferential-or-derivational edges per §2.4 / §2.5 territory.
- **Exactly-one-populated structural commitment.** §2.3 binding text codifies that assertions populate exactly one `subject_ref_X` field per assertion. Schema-technology specifics deferred to §0003 per §0011 forward-reference precedent.
- **Q1-B Session reference handling.** `subject_ref_observation` carries `DeclaredSession` references; `subject_ref_construct` carries `OperationalSession` references. Q1-B's typed-distinct entity-level distinction is preserved structurally at the reference level via field choice.
- **§2.5 frozen v0.3 compatibility.** §2.5 binding text's abstract reference vocabulary accommodates B's per-category fields directly. No §2.5 amendment required (verified F-SCAFFOLD-4).
- **Identity tiers (Open Modeling Q1 post-renumbering).** §2.3 binding text encodes the identity-tier consistency default per §0015 (DeclaredSession and OperationalSession share identity-tier references by default). Identity tiers formal resolution remains forward-referenceable per §0011 precedent regardless of Q3 outcome.

### Cascade implications

**Q3 is the last pre-§2.3 pre-Gate** per [`decision-log §0014`](../../charter/decision-log.md)'s lazy pre-Gate methodology. No further cascade pre-Gates anticipated.

Cascade prediction:
- **Q3-2 produces decision-log §0016** (analog to Q1-2's §0015), recording Q3 resolution + any committee extensions.
- **After §0016, §2.3 redaction Step 1.1 begins.** §0014's lazy pre-Gate refinement is fully discharged.
- **Identity tiers (Open Modeling Q1 post-renumbering) remains forward-referenceable** per §0011 precedent. §2.3 Step 1.1 anchor inventory will assess Identity tiers status per §0014's empirical-assessment methodology, but no Identity-tiers pre-Gate is currently anticipated.

**No new pre-Gates opened at Q3-2.** The §0014 lazy pre-Gate cascade fully discharges with Q3-2's resolution; §2.3 redaction proceeds unblocked.
