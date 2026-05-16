# Q1 — Session Duality — discussion evidence

**Status:** in-discussion. Not authoritative. Final resolution will be recorded in decision-log and Ontology document revision.

This document records the discussion-phase evidence for Ontology Open Question 1 (Session Duality) per the RFC [`ontology-revision-q1-session-duality`](../draft/ontology-revision-q1-session-duality.md) opened in Gate §2.3 prep ([`decision-log §0014`](../../charter/decision-log.md)). Five phases, structurally parallel to Q2-1 ([`q2-evidence.md`](./q2-evidence.md)) and Q4-1 ([`q4-evidence.md`](./q4-evidence.md)).

---

## Phase 1 — Evidence per dimension per candidate

Six dimensions evaluated. Two candidates from the RFC's Proposal section; Candidate C is rejected in the RFC and does not appear here. Dimension 6 (§2.2 compliance) is Q1-specific; Q2-1 and Q4-1 did not require it as a separate dimension because their candidates did not directly press a frozen invariant.

### Candidate A — Single `Session` (Cat I) + Cat II `SessionReconciliation`

**D1 — Provenance chain shape.** Single Category I primary `Session` referenced. Where reconciliation has been applied, the assertion carries an additional inferential reference (per §2.4 pending) to the Cat II `SessionReconciliation`. Chain: `assertion → Session (Cat I)` + optional `→ SessionReconciliation (Cat II) → Session (Cat I)`. One Category I anchor; reconciliation is auxiliary. Source: [Q1 RFC Cand A Provenance implication](../draft/ontology-revision-q1-session-duality.md); [`provenance-model.md` §Observational Provenance](../../ontology/provenance-model.md).

**D2 — Schemas and query implications.** Two record types (Session, SessionReconciliation), only one is Cat I primary. Validation surface: Session record-type + reconciliation record-type. Query "what was happening in session X" returns Session by default; if reconciliation lookup is requested, additional indirection. Polymorphic on reconciliation outcome at the query layer; uniform at substrate. Source: [Q1 RFC Cand A Query pattern](../draft/ontology-revision-q1-session-duality.md).

**D3 — Asymmetry visibility.** Disagreement between declared and operational session boundaries lives as a relationship (Cat II construct pointing to Cat I record). Default queries return the declared form unless reconciliation is explicitly applied. Asymmetry is **traversal-visible**, not type-visible. Source: Q1 RFC Cand A Cons.

**D4 — Identity-tier interaction.** Single Session carries ownership reference to identity-tier (`ActorRef`/`Identity`/`Cluster` per [`entity-model.md` Open Modeling Question 2](../../ontology/entity-model.md) pending). SessionReconciliation, if it carries ownership info, refers to the same identity-tier — no divergence question. Identity-tier deferral inherits unchanged.

**D5 — Forward-look to §2.5 and §2.3.** §2.5 (frozen v0.3) hypothesis events referencing Sessions: `subject_ref` to Session (single type). §2.3 binding text "back to observations" anchors to Session as one Cat I primary; reconciliation appears in the inferential chain (§2.4 territory), not the observational chain.

**D6 — §2.2 compliance.** Cleanly respects §2.2 **provided SessionReconciliation is deterministic** (a §2.2-compliant Cat II operational construct requires deterministic derivation from substrate + versioned operational definition). If reconciliation is non-deterministic (judgmental, probabilistic), SessionReconciliation drifts toward Category III hypothesis, and Cand A becomes structurally `Session + hypothesis about session` not `Session + Cat II reconciliation`. **Condition: A's compliance hinges on a downstream commitment to deterministic reconciliation.** Source: §2.2 frozen; [`entity-model.md` §Category II L30](../../ontology/entity-model.md).

### Candidate B — `DeclaredSession` (Cat I) + `OperationalSession` (Cat II)

**D1 — Provenance chain shape.** Branching by type: assertion → DeclaredSession (Cat I) directly; OR assertion → OperationalSession (Cat II) → DeclaredSession (Cat I) (the operational construct derives from declared inputs plus other Cat I primaries). Each branch terminates in Cat I primaries clearly. Two distinct chain shapes; reader knows from the type which they are traversing. Source: [Q1 RFC Cand B Provenance implication](../draft/ontology-revision-q1-session-duality.md).

**D2 — Schemas and query implications.** Two entity types as distinct Cat I/Cat II surfaces. Validation surface: two schemas — larger structural commitment than A. Query "what was reported about session X" returns DeclaredSession; "what was operationally happening" returns OperationalSession; "either / both" requires explicit disjunction. Branching by type, not by relationship lookup. Source: Q1 RFC Cand B Query pattern.

**D3 — Asymmetry visibility.** First-class at the type level. Disagreement between declared and operational boundaries is immediately visible — two distinct records with potentially different content. No relationship traversal required. **Type-visible.** Source: Q1 RFC Cand B Pros.

**D4 — Identity-tier interaction.** Both DeclaredSession and OperationalSession carry ownership references. Question: do they share identity-tier references, or could OperationalSession redefine the actor relationship (e.g., where the operational definition merges sessions across declared identities)? Identity-tier deferral has **more pressure** under B because consistency-across-types must be explicit. Pending Q2 resolution; under §0014 lazy pre-Gate refinement, this would be assessed at §2.3 Step 1.1.

**D5 — Forward-look to §2.5 and §2.3.** §2.5 hypothesis events referencing Sessions: `subject_ref` must distinguish DeclaredSession from OperationalSession. **`subject_ref` polymorphism question** (entity-model Q3 pending) becomes acutely pressing under B. §2.3 binding text "back to observations" anchors only to Cat I primaries, so the phrase points to DeclaredSession (not OperationalSession, which is Cat II). The Cat II form is reached via §2.4 inferential influence (pending) or via §2.3's downstream provenance traversal.

**D6 — §2.2 compliance.** **Exemplifies §2.2.** Each form has its own category type, declared at construction and not changeable. The cross-category reference (OperationalSession derives from DeclaredSession) is a typed transformation — exactly the structure §2.2 mandates. The asymmetry between observation (Cat I) and operational construct (Cat II) is **made structural** rather than encoded in code or relationships. Source: §2.2 frozen Definition + Structural Requirement.

### Observation — strongest asymmetry and most consequential dimension

- **Strongest asymmetry between A and B: Dimension 3 (Asymmetry visibility).** B is type-visible; A is traversal-visible. The asymmetry tracks back to the central question Q1 names — "the two diverge in exactly the cases where investigation matters most" ([`entity-model.md` Open Q1](../../ontology/entity-model.md)). If divergence matters operationally, B surfaces it structurally; A surfaces it via auxiliary lookup.
- **Most consequential dimension: Dimension 6 (§2.2 compliance).** Both candidates respect §2.2, but the compliance shape differs:
  - **A:** clean compliance **conditional on** SessionReconciliation being deterministic (Cat II requirement). The compliance is real but downstream-dependent.
  - **B:** exemplifies §2.2 by making the Cat I/Cat II boundary explicit at the type level. Compliance is unconditional.
  - This is not a "B violates §2.2" finding — both candidates respect the invariant. It is a "B's compliance is more direct; A's compliance carries a downstream determinism commitment" finding.

The predicted strongest-asymmetry dimension (3) tracks evidence. The predicted most-consequential dimension (6) tracks evidence with a refinement: §2.2 is not contested for compliance/violation — it is contested for compliance *shape* (conditional vs unconditional).

---

## Phase 2 — Scaffold agnosticism check

### F-SCAFFOLD-1 — `entity-model.md` Category I + Category II Session examples

Two scaffold passages reference Session-like content, in DIFFERENT category sections with DIFFERENT capitalization:

**Cat I Examples [`entity-model.md` L16](../../ontology/entity-model.md):**

> "Session events reported by client SDKs."

- Capital "S" Session — implies a named type.
- The example is "**events**" — the Session is the referent of the events; the events themselves are the Cat I observations.
- Reading is structurally compatible with both candidates: under A, these are events about the single Session entity; under B, these are DeclaredSession-events (the Cat I form).

**Cat II Examples [`entity-model.md` L33](../../ontology/entity-model.md):**

> "A session reconstructed by an operational definition (e.g., 'events from one actor within a 30-minute inactivity window')."

- Lowercase "s" session — descriptive, not type-named.
- "Reconstructed by an operational definition" — explicitly Cat II derivation under §2.2 frozen.
- Reading is compatible with both candidates: under A, this is the operational view of the reconciled Session; under B, this IS `OperationalSession`.

**F-SCAFFOLD-1 verdict.** Scaffold uses BOTH a capitalized Session (Cat I example, events about it) AND a lowercase reconstructed session (Cat II example, operational view). The capitalization asymmetry is subtle but suggestive: Session-as-type appears in Cat I; session-as-construct appears in Cat II. This is **genuinely agnostic** on Q1 — the same prose fits both candidates — but the capitalization pattern leans slightly toward distinguishing the two roles structurally (compatible with B's typed-distinct shape, but also with A's Session + SessionReconciliation shape where reconciliation is the "reconstructed session"). The leaning is mild; neither candidate is precluded.

### F-SCAFFOLD-2 — §Open Modeling Question 1 framing

[`entity-model.md` L82](../../ontology/entity-model.md):

> "**Session duality.** Is a session a single entity with reconciliation, or two entities (`DeclaredSession` as Category I, `OperationalSession` as Category II)? The conversation that produced this Ontology recognized that the two diverge in exactly the cases where investigation matters most."

Verbatim analysis:

- "**Is a session a single entity with reconciliation, or two entities...**" — the question itself presents the binary cleanly. The framing names BOTH candidates explicitly. The order (A first, B second) is the only positional leaning — purely incidental.
- "**The conversation that produced this Ontology recognized that the two diverge in exactly the cases where investigation matters most.**" — this is the load-bearing scaffold sentence. Two readings:
  - *Strong reading:* divergence-matters-most-in-investigation argues for B (visible asymmetry needed for investigation).
  - *Weak reading:* divergence is a real phenomenon worth structural treatment, but the prose does NOT specify HOW investigation accesses it. Candidate A's SessionReconciliation construct *also* supports investigation of the divergence — just via traversal rather than type-level visibility.

**F-SCAFFOLD-2 verdict.** The phrase recognizes the *phenomenon* (declared/operational divergence exists and matters operationally) but does NOT commit to *how the phenomenon is typed*. Both candidates support investigation of the divergence:
- Under A, investigation traverses `Session ← SessionReconciliation` and observes the divergence in the reconciliation construct.
- Under B, investigation queries DeclaredSession and OperationalSession separately and observes the divergence between the two records.

The sentence is **evidence-of-need-for-visible-asymmetry** but **not pre-decision of B**. The scaffold reading is verified empirically: this is evidence that the asymmetry matters, NOT evidence that type-level visibility is the only way to surface it. Scaffold remains agnostic; the implicit lean (if any) is toward acknowledging the phenomenon, not toward typing it one specific way.

### F-SCAFFOLD-3 — `provenance-model.md` §Observational Provenance

Empirical scan of `provenance-model.md` (Scaffold status; pending committee redaction after §2.3 and §2.4):

- §Observational Provenance (L11–21): uses generic vocabulary throughout — "every non-observation assertion in Ghost Trace declares, in its structure, the set of observations from which it was computed." No specific reference to Sessions.
- §Inferential Provenance (L22–31): same generic vocabulary — references "observations", "operational constructs", "hypotheses", "assertions" categorically. No specific Session reference.
- §The Provenance Graph (L33–45): "Nodes are records (observations, operational constructs, hypotheses, assertions). Edges are typed..." — fully category-level abstraction; no Session-specific examples.
- §Open Modeling Questions (L47–52): four questions about granularity, decay, projection-vs-substrate, cross-domain. None Session-specific.

**F-SCAFFOLD-3 verdict.** `provenance-model.md` is **completely Session-agnostic**. The prose abstracts to category-level vocabulary; Session never appears as a specific example. The scaffold neither leans toward A nor toward B at the provenance layer. Whichever Q1 resolution wins, `provenance-model.md` requires no Session-specific revision — the generic category-level prose accommodates either form.

### Carry-along verifications

**Q3 (Subject reference polymorphism) discernibility status:**

- [`entity-model.md` L84](../../ontology/entity-model.md): "**Subject reference polymorphism.** Assertions carry a `subject_ref` that may point to entities of any category. Whether this is a single polymorphic field or distinct fields per category is a type-level question with ontological consequences."
- Q3 status today: **fully open**, with no scaffold-level commitment.
- Under Candidate B: Q3 becomes **acutely pressing** — `subject_ref` to Session needs to distinguish DeclaredSession from OperationalSession. Under Candidate A: Q3 is less pressing (one Session type).
- Discernibility under B: Q3 is **likely blocking** for §2.3 redaction under B (subject_ref shape directly affects §2.3 "back to observations" clause). This confirms the §0014 lazy-pre-Gate trigger: if B wins, Q3 becomes a pre-§2.3 dependency that may need its own resolution before §2.3 Step 1.2.
- Discernibility under A: Q3 remains forward-referenceable per §0011 precedent.

**Q2 (Identity tiers) — implicit single-ownership commitment check:**

- [`entity-model.md` L83](../../ontology/entity-model.md): "**Identity tiers.** The conversation introduced `ActorRef`, `Identity`, and `Cluster` as three tiers of identity. Their formalization is pending."
- Scaffold does NOT contain any specific Session-ownership prose. The L33 Cat II example mentions "one actor" within an inactivity window — singular "actor" reference, but in the context of operational definition (Cat II), not as a typed Session-ownership commitment.
- Scaffold makes **no implicit single-ownership commitment** for Session. Q2 is purely deferred at the scaffold level.
- Under Candidate A: Session has one ownership reference, identity-tier deferred. Single-ownership reference naturally satisfied.
- Under Candidate B: DeclaredSession and OperationalSession each carry ownership references. Consistency-across-types is a Q1-resolution-derivative question. Q2 deferral pressure is **higher** under B.

### Methodological observation

**Scaffold neutrality is a third valid scaffold-finding state**, alongside implicit-lean (Q2-1, Q4-1) and explicit-defer.

Q1's scaffold differs from Q2/Q4 scaffolds in shape:
- **Q2-1 scaffold finding:** implicit lean toward B (single-type discriminator) per the scaffold's references to "the four labels" — implied B was the assumed structure.
- **Q4-1 scaffold finding:** implicit lean toward B-family (deep criterion alone, not staged combination) per scaffold's narrower framing.
- **Q1 scaffold finding:** genuinely agnostic. Cat I/Cat II examples use different capitalizations that fit both candidates; Open Modeling Q1 names both candidates explicitly without committing; provenance-model.md is fully Session-agnostic at category-level abstraction.

This produces a **new methodological observation**: a scaffold that has NOT implicitly resolved the open question. The pattern suggests two scaffold authorship modes:
- *Implicit-lean scaffolds* (Q2, Q4): the scaffold prose presupposes one answer for fluency, leaving the open question to be surfaced by careful reading.
- *Agnostic scaffolds* (Q1): the scaffold prose names both candidates explicitly or abstracts to category-level vocabulary that does not pre-decide.

Q1's agnostic scaffold likely owes to the explicit binary framing in `entity-model.md` Open Q1 ("Is a session a single entity with reconciliation, or two entities") — naming both candidates in the question itself forces scaffold authorship to avoid pre-decision. Q2's and Q4's scaffolds had less explicit binary framing in their parent questions, which permitted implicit lean.

### Summary table

| Finding | Verdict | Strength |
|---|---|---|
| F-SCAFFOLD-1 (entity-model Cat I/II Session examples) | Genuinely agnostic; mild capitalization asymmetry leans subtly toward structural distinction (compatible with both A and B) | Mild |
| F-SCAFFOLD-2 (Open Modeling Q1 framing) | Evidence-of-need-for-visible-asymmetry; NOT pre-decision of B | Substantive non-leaning |
| F-SCAFFOLD-3 (provenance-model.md) | Completely Session-agnostic; category-level abstraction throughout | Strong |
| Q3 discernibility | Open today; **likely blocking under B** (subject_ref polymorphism becomes acute); forward-referenceable under A | Asymmetric |
| Q2 (Identity tiers) implicit commitment | None; scaffold makes no single-ownership pre-commitment. Q2 deferral pressure higher under B | Asymmetric |

**Overall scaffold verdict: agnostic on Q1.** No implicit candidate-lean detected.

---

## Phase 3 — Epistemic findings

Six cells from 2 candidates × 3 skills.

### Candidate A × `falsifiability-check`

Applying §1 four-question test to "SessionReconciliation is a Cat II construct that captures the operational reading of a Session."

- **V (Violation).** Falsifying state: a SessionReconciliation record exists without a corresponding versioned operational definition reference. Detectable structurally via schema-level field-presence check on `definition_ref`.
- **O (Observation).** Third party reads substrate; checks reconciliation records for `definition_ref` field; verifies determinism by recomputing output from inputs. Mechanical.
- **Op (Operationalization).** "Reconciliation" reduces to "deterministic output of versioned operational definition over substrate inputs" — **conditional pass.** The Q1 RFC explicitly defers "Reconciliation rules under Candidate A"; the operationalization passes ONLY if SessionReconciliation is committed to be deterministic in the resolution phase.
- **NC (Non-circularity).** `Session` glossary-canonical (under A's resolution path). `SessionReconciliation` reduces to Cat II construct over Session inputs. Non-circular by reference.

**Verdict:** Conditional pass. Op test hinges on resolution-phase commitment to determinism.

### Candidate A × `epistemic-separator`

Applying §3 checklist + §4 forbidden constructions.

- **Naming:** `Session` (Cat I) and `SessionReconciliation` (Cat II) — category-named explicitly. ✓
- **Operation validity:** Session commits (Cat I append-only — valid). SessionReconciliation is re-derived under operational definition revision (Cat II parametric re-derivation — valid). ✓
- **Typed crossing:** SessionReconciliation references Session via a typed read (Cat II referring to Cat I). ✓

**Particular risk surfaced — the load-bearing question:** Is `SessionReconciliation` genuinely Category II, or implicit Category III (hypothesis about the session)?

Cat II requires *deterministic* derivation under a versioned operational definition (per [`entity-model.md` Cat II L30](../../ontology/entity-model.md): "deterministic with respect to their input observations and their definitional parameters"). If the reconciliation procedure is judgmental, probabilistic, or non-deterministic, SessionReconciliation drifts toward Cat III (hypothesis), violating its declared Cat II type.

Examples:
- Deterministic rule like "events from one actor within a 30-minute inactivity window operationally constitute one session" → SessionReconciliation Cat II ✓
- Stochastic clustering algorithm with non-deterministic boundaries → would be Cat III, not Cat II ✗ — and under §2.5 frozen v0.3, Cat III is `Hypothesis` (one of four sibling subtypes). Reconciliation as `BehavioralCluster`-equivalent would change Q1's structural shape substantially.

**A's epistemic-separator verdict:** Clean **IF** reconciliation is deterministic. Resolution phase must commit explicitly.

### Candidate A × `ambiguity-reducer`

Watchlist scan against A's vocabulary surface:

- `state` — "Session state" is implied by reconciliation language (what changes when reconciliation is applied?). Advisory.
- `identity` — Session ownership references identity-tier per Q2 (pending). Advisory.
- `context` — "session context" is a common phrase under A's reconciliation framing. Advisory.
- `decision` — reconciliation invokes "decisions" about which boundary to apply. Advisory.

**Verdict:** 4 advisory hits (non-blocking). Each is a vocabulary-surface finding for resolution-phase binding text.

### Candidate B × `falsifiability-check`

Applying §1 four-question test to "OperationalSession is a Cat II construct typed-derived from DeclaredSession (and other Cat I primaries) under a versioned operational definition."

- **V (Violation).** Falsifying state: an OperationalSession record exists without a typed-derivation chain back to Cat I inputs. Detectable structurally via schema-level `derivation_inputs` field presence.
- **O (Observation).** Third party reads substrate; checks OperationalSession records for derivation inputs; verifies determinism by recomputing. Mechanical.
- **Op (Operationalization).** "OperationalSession" reduces to "deterministic output of versioned operational definition over Cat I inputs" — **same conditional as A**, but B exposes the conditional at the type level. The determinism requirement is structurally explicit (the type is Cat II by declaration; Cat II requires determinism per §2.2).
- **NC (Non-circularity).** `DeclaredSession` and `OperationalSession` would be new glossary terms under B. Both reduce to Cat I / Cat II definitions. Non-circular by reference.

**Verdict:** Conditional pass on the same determinism requirement as A — but the conditional is structurally exposed under B (Cat II type declaration forces the determinism question to be addressed).

### Candidate B × `epistemic-separator`

- **Naming:** `DeclaredSession` (Cat I), `OperationalSession` (Cat II) — both category-named explicitly. ✓✓
- **Operation validity:** DeclaredSession commits (Cat I append-only — valid). OperationalSession is re-derived under definition revision (Cat II — valid). ✓✓
- **Typed crossing:** OperationalSession derives from DeclaredSession (and other Cat I primaries) — explicit typed transformation producing a new Cat II record. ✓

**Particular risk surfaced:** Does the typed transformation respect §2.2's per-category immutability and per-category operations?

[§2.2 Structural Requirement (frozen)](../../charter/constitutional-charter.md#22-epistemic-separation): "Promotion of a hypothesis into operational use, or the use of an observation as input to inferential computation, requires explicit transformation through a typed boundary that produces a new record of the destination category, never reclassification of the original."

Under B, the typed transformation produces an OperationalSession (new Cat II record); DeclaredSession is NOT reclassified. ✓ Respects §2.2.

**B's epistemic-separator verdict:** Clean and exemplary. The typed transformation makes the Cat I → Cat II boundary visible structurally. §2.2's "typed boundary" requirement is met by virtue of separate type declarations.

### Candidate B × `ambiguity-reducer`

Watchlist scan against B's vocabulary surface:

- `state` — under B, "session state" is more ambiguous because two types coexist (which type's state?). Advisory; possibly more pressing than under A.
- `identity` — both DeclaredSession and OperationalSession reference identity-tier; consistency-across-types question (Q2 deferral). Advisory.
- `context` — "session context" under B is more ambiguous. Advisory.
- `decision` — operational definition application invokes "decisions." Advisory.

**Verdict:** 4 advisory hits, similar surface to A. But: B introduces TWO new canonical glossary terms (DeclaredSession, OperationalSession); A introduces ONE new canonical term (SessionReconciliation). Vocabulary footprint is larger under B at the glossary layer.

### 2×3 epistemic matrix summary

| Skill | Candidate A | Candidate B |
|---|---|---|
| `falsifiability-check` | Conditional pass (Op hinges on determinism commitment) | Conditional pass on same determinism (Cat II declaration exposes it structurally) |
| `epistemic-separator` | Clean **IF** SessionReconciliation is deterministic Cat II; risks Cat III drift if non-deterministic | Clean and exemplary; typed transformation makes Cat I→Cat II boundary structural |
| `ambiguity-reducer` | 4 advisory hits (state, identity, context, decision); ONE new glossary term | 4 advisory hits; TWO new glossary terms |

### Most consequential epistemic finding

**The determinism requirement is structural to §2.2 Cat II, and applies equally to both candidates.** This dissolves part of the perceived asymmetry from D6 of Phase 1:

- Under A, the Cat II construct is `SessionReconciliation`; its determinism must be committed in the resolution phase. If non-deterministic, A's "SessionReconciliation as Cat II" claim fails and the construct drifts to Cat III hypothesis — which would change Q1's structural shape from "Session + Cat II reconciliation" to "Session + Cat III hypothesis about session boundary."
- Under B, the Cat II construct is `OperationalSession`; same determinism requirement applies, but B exposes it at the type level (the Cat II declaration is the structural commitment).

**The real structural commitment Q1 resolution must make (regardless of candidate)** is: the operational Cat II form's deterministic derivation under a versioned operational definition. Q1's two candidates differ in HOW the commitment is exposed:
- **A:** commitment is procedural (resolution phase must commit to deterministic reconciliation; otherwise drift to Cat III).
- **B:** commitment is structural (Cat II type declaration forces the determinism question to be addressed at the type level; non-determinism would be visible as a Cat III misclassification).

This refines D6 (§2.2 compliance) further:
- A's compliance is conditional because A's structural form *allows* the determinism question to remain unclear.
- B's compliance is structural because B's structural form *forces* the determinism question to be addressed at the type level.

---

## Phase 4 — Comparison synthesis

### Findings consolidated, numbered by consequence

**Finding 1 — Determinism is the real structural commitment, regardless of candidate.** Apparent trade-off that resolves. The Cat II form (SessionReconciliation under A; OperationalSession under B) must be deterministic under a versioned operational definition per §2.2 frozen. This requirement applies equally to both candidates. The asymmetry from D6 (§2.2 compliance) refines to: **A's compliance is procedurally conditional** (resolution phase must commit explicitly; structural form ALLOWS the determinism question to remain unclear); **B's compliance is structurally exposed** (Cat II type declaration FORCES the determinism question at the type level). Sources: Phase 1 D6; Phase 3 most-consequential finding.

**Finding 2 — Asymmetry visibility is the most direct candidate trade-off.** Genuine trade-off. Under B, the declared/operational divergence is type-visible (two distinct records). Under A, it is traversal-visible (Cat II reconciliation pointing to Cat I Session). The phenomenon Q1's open-question prose names — "the two diverge in exactly the cases where investigation matters most" — is supported by both candidates; they differ in HOW investigation accesses the divergence. Sources: Phase 1 D3; Phase 2 F-SCAFFOLD-2.

**Finding 3 — Q3 (subject_ref polymorphism) deferral asymmetric.** Asymmetry. Under B, Q3 becomes likely-blocking for §2.3 redaction (subject_ref to Session needs to distinguish DeclaredSession from OperationalSession; the polymorphism question directly affects §2.3 binding text shape). Under A, Q3 remains forward-referenceable per §0011 / §2 L41 precedent. This is **exactly the §0014 lazy-pre-Gate trigger condition**: if B wins, Q3 is the next pre-Gate dependency before §2.3 Step 1.2; if A wins, §0014's deferral-to-Step-1.1 holds. Sources: Phase 1 D5; Phase 2 F-SCAFFOLD Q3 status check.

**Finding 4 — Q2 (Identity tiers) deferral pressure asymmetric.** Asymmetry, lower-consequence than Finding 3. Under B, two types each carry identity-tier references; consistency-across-types is a Q1-derivative question (Q2 deferral has more pressure). Under A, single Session has one ownership reference; Q2 forward-reference works naturally. Both candidates remain Q2-forward-referenceable per §0011 precedent; only the pressure differs. Sources: Phase 1 D4; Phase 2 Q2 implicit-commitment check.

**Finding 5 — Schemas and glossary footprint asymmetric, modestly favoring A.** Genuine trade-off. Under B, two new canonical glossary terms (DeclaredSession, OperationalSession) plus two record-type validation surfaces. Under A, one new canonical term (SessionReconciliation, if scoped as Cat II construct) plus two record-type validation surfaces (Session + SessionReconciliation). Footprint difference is modest at glossary layer (1 vs 2 entries); validation-surface count is identical (2 schemas). Sources: Phase 1 D2; Phase 3 vocabulary surfaces.

**Finding 6 — Provenance chain shape asymmetric.** Asymmetry. Under A, single Cat I anchor (Session) with optional inferential reference to SessionReconciliation (Cat II) via §2.4 channel. Under B, branching by type — chain terminates in DeclaredSession (Cat I) directly, or in OperationalSession (Cat II) which traces back to its Cat I derivation inputs. Both shapes are concrete and falsifiable; B's branching is structurally explicit. Sources: Phase 1 D1; Phase 3 falsifiability findings.

**Finding 7 — Scaffold neutrality verified empirically.** Methodological observation. Q1's scaffold differs from Q2/Q4 scaffolds: it is genuinely agnostic, with mild capitalization asymmetry in entity-model.md Cat I/Cat II examples (capital Session in Cat I; lowercase session in Cat II) and complete Session-agnosticism in provenance-model.md. Open Modeling Q1's framing recognizes the phenomenon but does not commit to typing. This produces a third valid scaffold-finding state (alongside Q2/Q4's implicit-lean and the explicit-defer pattern): **agnostic scaffold**. Sources: Phase 2 F-SCAFFOLD-1, F-SCAFFOLD-2, F-SCAFFOLD-3.

### §2.2 compliance verdict

| Candidate | §2.2 compliance | Compliance shape | Risk if compliance fails |
|---|---|---|---|
| **Candidate A** | **Compliant CONDITIONAL** on explicit determinism commitment for SessionReconciliation in Q1-2 resolution phase. | Procedurally exposed. Structural form ALLOWS the determinism question to remain unclear at the Q1-1 stage. | Without the commitment, SessionReconciliation drifts toward Cat III hypothesis (becomes a `BehavioralCluster`-class entity per §2.5 frozen vocabulary), changing Q1's resolution shape from "Session + Cat II reconciliation" to "Session + Cat III hypothesis about session boundary." |
| **Candidate B** | **Compliant STRUCTURAL**. | Structurally exposed. The Cat II type declaration of OperationalSession forces the determinism question at the type level. | Non-determinism would be visible as a Cat III misclassification at schemas/validation time; the structural form prevents implicit drift. |

**Verdict:** Both candidates respect §2.2. Neither violates. The asymmetry is in compliance *shape*: A's compliance is procedurally conditional; B's is structurally exposed. **Refined from Phase 1's "genuinely contested" prediction:** the contest is over compliance shape, not compliance vs violation.

### Convergence shape

**Meta-commitment-level convergence**, a third shape alongside Q2's candidate-level (§0010 — Candidate A.2) and Q4's form-level (§0011 — staged-combination AND). The convergence is on the structural commitment that the operational Cat II form must be deterministic (regardless of A/B choice); the candidate choice is a deferred values deliberation parameter.

This is methodologically significant. Q4-1 converged on a *form* (staged combination) with the *parameter* deferred (AND vs OR). Q1-1 converges on a *meta-commitment* (deterministic Cat II form) with the *candidate* deferred (single+reconciliation vs dual-entity). Both leave the resolution-phase committee a deliberation parameter to set; the difference is what kind of parameter (form-level for Q4; candidate-level for Q1).

The discussion phase has done its work: it has surfaced the real structural commitment (Finding 1) and identified the genuine trade-off space (Findings 2, 3, 5, 6 as asymmetries; Finding 2 as the central values dimension).

### §2.3 binding text impact

| Aspect | Under Candidate A | Under Candidate B |
|---|---|---|
| "Back to observations" clause | Single Cat I primary anchor: `Session`. | Multiple Cat I primaries; assertions referencing Session-as-declared anchor to `DeclaredSession`; assertions involving Session-as-operational trace through `OperationalSession` (Cat II) back to its Cat I inputs. |
| Provenance chain shape | One Cat I anchor + optional §2.4 inferential reference to SessionReconciliation | Type-branching; two distinct shapes by which Session form is referenced |
| Subject_ref polymorphism (Q3) | Forward-referenceable per §0011; binding text uses canonical subject_ref form without committing to polymorphism shape | Likely blocking; binding text needs Q3 resolution OR explicit subject_ref-per-category accommodation |
| Vocabulary surface | "observation" includes Session as one canonical Cat I primary type | "observation" includes DeclaredSession; OperationalSession explicitly named as Cat II (not "observation" per §2.2 separation) |

### Identity-tier deferral status

| Candidate | Identity-tier deferral feasibility | Pressure level |
|---|---|---|
| **Candidate A** | Forward-referenceable per §0011 precedent. Single Session ownership reference; Q2 resolution affects HOW ownership is structurally encoded (ActorRef/Identity/Cluster), not WHETHER it multiplies. | **Low pressure** — Q2 deferral fits the §0011 pattern cleanly. |
| **Candidate B** | Forward-referenceable per §0011 precedent. Two types each carry ownership references; Q2 resolution must address consistency-across-types. | **Higher pressure** — Q2 deferral still feasible, but resolution-phase decisions about Cat II `OperationalSession`'s identity-tier flexibility raise more downstream questions. |

Both candidates retain Q2 forward-reference under the §0011 precedent. Pressure differs; neither is blocked by Q2.

### Summary statement

Based on evidence, **no clear winner on dimensions alone**. The asymmetries cut both ways:

- **Candidate B advantages:** type-level visibility of declared/operational asymmetry (F2 most directly tracks Q1's named operational concern), structural exposure of the determinism requirement (F1 — §2.2 exemplified rather than conditional), no risk of Cat II→Cat III drift.
- **Candidate A advantages:** smaller schemas and glossary footprint (F5), simpler identity-tier deferral (F4 lower pressure), Q3 subject_ref polymorphism remains forward-referenceable per §0011 (F3 — A keeps §0014 lazy pre-Gate deferral intact).

The committee at Q1-2 faces a values trade-off between:
- **Type-level visibility** of the declared/operational divergence (favors B; the central Q1 question of "where investigation matters most" is most directly served by type-level visibility).
- **Minimum-surface-versus-need** (favors A; if divergence is rare in practice, A's smaller surface is justified; if frequent, B's larger surface is justified).

**Convergence:** meta-commitment-level. Both candidates require the same Cat II determinism commitment; the candidate choice is the resolution-phase deliberation parameter.

---

## Phase 5 — Recommendation

### Recommendation

**Candidate B — `DeclaredSession` (Category I) + `OperationalSession` (Category II) — is recommended toward by the discussion-phase evidence**, primarily on the strength of Finding 2 (asymmetry visibility) matching Q1's named central design driver most directly. Q1's open-question prose explicitly identifies "investigation" as the use case where the declared/operational divergence matters; B's type-level visibility serves this concern structurally, while A's traversal-visible asymmetry serves it indirectly. Finding 1's refinement reinforces — B's §2.2 compliance is structurally exposed (the Cat II type declaration forces the determinism question to be addressed at the type level), whereas A's compliance is procedurally conditional and carries Cat III drift risk. The cost of B (Finding 3 — Q3 subject_ref polymorphism becomes acutely pressing and is likely to flip to "blocking" at §2.3 Step 1.1; Finding 5 — two new canonical glossary terms versus A's one; Finding 4 — modestly higher Q2 deferral pressure) is the operational economy trade-off. Crucially, the §0014 lazy pre-Gate methodology was designed precisely for this scenario: if §2.3 Step 1.1 assesses Q3 as blocking, the response is procedurally well-defined (open a Q3 pre-Gate RFC between Step 1.1 and Step 1.2, calendar cost ~2 prompts modeled on Q2/Q4 paths). The Q3 pre-Gate is the anticipated cost, not a surprise.

### What would reverse this recommendation

1. **Evidence that declared/operational divergence is rare in practice.** If divergence is the exceptional case rather than the operationally consequential one, B's type-level visibility imposes ceremony that A's traversal pattern avoids. Q1's open-question prose ("diverge in exactly the cases where investigation matters most") states the divergence matters operationally but does not quantify frequency. If operational telemetry (when collected post-implementation) showed divergence below a meaningful threshold, A's smaller footprint would dominate.

2. **Evidence that the Q3 pre-Gate is more costly than the §0014 methodology anticipated.** §0014 budgets the Q3 pre-Gate as a sequenced 2-prompt resolution analogous to Q2/Q4 paths. If Q3 turns out to be entangled with §2.6 (evidential independence, pending) — for instance, if subject_ref polymorphism affects how evidential independence dimensions are recorded per-category — Q3 may not resolve cleanly without §2.6 prerequisites. Under that condition, B's selection adds calendar cost beyond the §0014 budget; A's Q3-forward-referenceable status becomes more attractive.

3. **Evidence that A's determinism commitment is robust in practice.** If domain-specific reconciliation procedures are reliably deterministic across the first applied domain AND projected future expansion domains, A's procedural-conditional compliance is empirically no weaker than B's structural-exposed compliance. F-SCAFFOLD-2's reading hinges on type-level visibility being NEEDED for investigation; if traversal-visible suffices in practice, A's economy advantages dominate. The committee at Q1-2 may have domain expertise to weigh this directly.

4. **A specific Q1-2 hybrid emerging from committee deliberation.** Q2-1 recommended Candidate A; the committee at Q2-2 extended with sub-resolution A.2 (abstract + sibling subtypes). Q4-1 recommended staged-combination form; the committee at Q4-2 extended with AND-composition. If Q1-2 surfaces a third structural form — for example, "B with deferred OperationalSession instantiation" (only materialize OperationalSession when its output differs from DeclaredSession; otherwise reduce to a reference) — that emerges from values deliberation, this recommendation is superseded by the hybrid. The committee-extension precedent from §0010 / §0011 is operative.

### §2.3 binding text shape under this recommendation

Under Candidate B (recommended toward), §2.3 binding text takes the following shape:

- **"Back to observations" clause.** Anchors to Category I primaries explicitly. Assertions referencing Session-as-declared anchor to `DeclaredSession` (Cat I primary). Assertions involving Session-as-operational trace through `OperationalSession` (Cat II) back to its Cat I derivation inputs (which include DeclaredSession plus other Cat I primaries like network events).
- **Provenance chain shape.** Type-branching — two distinct chain shapes depending on which Session-form is referenced. The branching is structural and visible at the type level.
- **Subject_ref polymorphism (Q3) handling.** Acutely pressing under B. §2.3 Step 1.1 anchor inventory will assess Q3 status per §0014's empirical-assessment methodology; expected outcome is Q3-blocking; expected response is opening a Q3 pre-Gate RFC between §2.3 Step 1.1 and §2.3 Step 1.2. This is the documented trigger condition for §0014's lazy pre-Gate refinement.
- **Identity-tier (Q2) consistency-across-types.** Q1-2 resolution must include a default commitment: DeclaredSession and OperationalSession share identity-tier references unless the operational definition explicitly overrides them. The commitment is procedural; Q2 itself remains forward-referenceable per §0011 precedent.
- **§2.5 hypothesis subject_ref interaction.** Hypothesis lifecycle events (per §2.5 frozen v0.3) that reference Sessions via subject_ref need to distinguish DeclaredSession from OperationalSession. This is part of the Q3 pre-Gate scope.
- **Determinism commitment for OperationalSession.** Q1-2 resolution must include explicit commitment that OperationalSession is deterministic per §2.2 Cat II requirements. Finding 1's structural requirement is the same under both candidates; under B, the commitment is structurally exposed via the Cat II type declaration.
