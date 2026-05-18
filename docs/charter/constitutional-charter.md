# Ghost Trace Constitutional Charter

**Version:** v0.4 (draft, sections in committee mode)
**Status:** Thesis frozen. Invariant qualification criteria (§2 header) frozen. Invariants 2.1–2.2 frozen. Invariant 2.3 frozen — minor amendment v0.4. Invariant 2.5 frozen — minor amendment v0.3. Invariants 2.4, 2.6 pending committee redaction. Non-Goals (§3) pending. Constitutional Design Rule (§4) frozen — minor amendment v0.2. Patch amendment v0.2.1 extends mechanical Charter-blockquote exemption to vocabulary-drift (no Charter prose amended; see [`decision-log.md` §0012](./decision-log.md)).

> This document is the constitutional surface of the Ghost Trace project. All other documents in this repository — Ontology, Architecture, RFCs — are subordinate to it. Changes to this document require formal amendment recorded in [`amendments.md`](./amendments.md). Changes to subordinate documents that conflict with this Charter are invalid by construction.

---

## 1. Thesis

Ghost Trace is a behavioral intelligence system designed to preserve the epistemic integrity of operational knowledge — the continued capacity to distinguish what was observed from what was inferred — as that knowledge accumulates, evolves, and is acted upon over time.

The system addresses a class of problem that conventional detection and observability platforms fail to model explicitly: intelligence systems that operate continuously over behavioral telemetry degrade in characteristic ways. Confidence in inferences inflates without proportional increase in independent evidence. Promoted hypotheses re-enter the system as enrichment and silently reinforce themselves. Historical decisions become unreplayable as the context that produced them mutates. The provenance of belief becomes indistinguishable from the provenance of observation. These degradations are rarely caused by individual engineering errors. They are structural consequences of treating inference and observation as ontologically equivalent.

Ghost Trace assumes that a behavioral intelligence system operating in environments where behavioral conclusions carry operational, financial, or regulatory consequences must treat epistemic integrity as a primary architectural property, not a desirable side effect. This is the central commitment from which all other commitments derive.

The system maintains a strict ontological separation between three categories of knowledge: observations recorded as immutable historical fact, operational constructs derived from explicit operational definitions over those observations, and hypotheses inferred probabilistically from accumulated evidence. These categories are not abstractions over the same underlying substrate. They have different lifecycles, different identity semantics, different rules for evolution, and different replay guarantees. Conflating them is the failure mode the system exists to prevent.

Provenance in Ghost Trace is therefore not metadata. It is structure. Every assertion the system produces — about a session, an actor, a cluster, a campaign — is required by construction to remain traceable to the observations from which it was derived. Each assertion is also required to remain traceable to the prior assertions whose existence influenced its formation. This second form of traceability, which conventional systems do not maintain, exists to preserve the system's capacity to distinguish between belief grounded in independent evidence and belief inherited recursively from earlier conclusions.

Decisions in Ghost Trace are not atomic events. They are temporally extended sequences of assertions, each reflecting the best available understanding at the moment it was made, none superseded by destruction. The historical record of what the system believed, and on what basis, at any prior moment, is preserved not as audit log but as first-class data. This is what makes forensic replay operationally meaningful: the system can answer not only what it concluded, but what it had grounds to conclude under the operational knowledge available at that time.

Ghost Trace is designed for engineers, researchers, and operators who recognize that intelligence systems in high-consequence environments degrade silently when their epistemology is implicit. The system accepts structural complexity where such complexity is required to preserve epistemic integrity. Where operational simplicity, throughput, or immediacy of decision conflicts with this property, epistemic integrity prevails by construction.

Ghost Trace is not a detector that happens to be auditable. It is a behavioral intelligence substrate within which detection is the first applied domain. Future domains — operational risk monitoring, market integrity surveillance, behavioral compliance — are expected to inherit the same properties without requiring redesign. The substrate is the contribution; the detectors are demonstrations.

Ghost Trace makes no claim to produce truth. It maintains, with structural rigor, the distinction between what was observed, what was inferred, and what is recursively believed. It exposes the basis of every operational conclusion. It refuses to forget the conditions under which its conclusions were drawn. These properties are not features the system provides. They are conditions of its existence as Ghost Trace. A system that violates these properties is not Ghost Trace by the criterion of this charter.

---

## 2. Constitutional Invariants

The following invariants define the structural identity of Ghost Trace. Each is required to be:

1. **Structurally enforceable** — verifiable in schema, types, or permitted operations, not merely in code review.
2. **Constraining of future implementation decisions** — capable of rejecting proposals that violate it.
3. **Identity-defining** — its absence changes what the system fundamentally is, not merely what it does.
4. **Independent of operator interpretation** — violation is detectable without subjective judgment.

The criteria above are themselves recorded formally in [Section 4 — Constitutional Design Rule](#4-constitutional-design-rule). They are applied as meta-rule to all invariants in this section.

The invariants appear in order of conceptual dependency: each rests on the ones preceding it.

---

### 2.1 Observational Integrity

#### Definition

Observations recorded in the primary event log are immutable. Once committed, an observation cannot be modified, deleted, or annotated with inferential content. Corrections, reinterpretations, and refinements are expressed as new records that reference the original, never as alterations of it.

#### Structural Requirement

The primary event log is append-only by construction. The storage substrate — whether operational stream or cold archive — enforces write-once semantics at the level of physical, cryptographic, or storage-enforced guarantee, not procedural convention. Records carry content-addressable identifiers or signed integrity proofs sufficient to detect mutation if attempted. No production code path is permitted write access to historical records; supersession of an observation's interpretation occurs through new assertion records, not through mutation of the observation itself.

#### Rationale

Every other property the system claims — replayability, provenance reconstruction, forensic auditability, separation of observation from inference — rests on the assumption that observations, once recorded, remain what they were. If observations can be modified, the historical record becomes a moving target, and no claim about past system behavior survives scrutiny. The integrity of inference depends on the integrity of the substrate over which inference operates.

This invariant is therefore not a storage policy. It is the precondition for the system to have an epistemology at all.

#### Forbidden Anti-Patterns

- **Annotation of raw events with inferential conclusions.** Writing detection outcomes, scores, or hypothesis tags directly onto the observation record. Even when intended as performance optimization or query convenience, this collapses the separation the Charter is built to maintain.
- **Retroactive correction of observations.** Modifying timestamps, payloads, or attribution fields based on later understanding, even when the later understanding is correct. The correction belongs in a new record that supersedes the prior interpretation; the original observation remains.
- **Destructive deduplication or compaction.** Removing observations from the log to reclaim storage or to "clean" the historical record. Compaction policies that lose individual records violate the invariant regardless of operational pressure. Physical reorganization that preserves reconstructibility of individual observations does not violate this invariant.
- **Soft deletion with hidden tombstones.** Marking records as deleted in a way that makes them invisible to replay or audit. If a record is recorded, it remains visible to the systems that depend on it.

#### Boundary Conditions

This invariant governs the primary event log only. Derived projections, materialized views, and caches are not bound by it; they are rebuildable from the log and may be recomputed, truncated, or replaced without violating this invariant. The invariant draws the line between substrate (immutable) and projection (disposable). Confusing the two is itself an anti-pattern.

---

### 2.2 Epistemic Separation

#### Definition

Ghost Trace maintains three structurally distinct categories of knowledge: **observations**, recorded as immutable historical fact; **operational constructs**, derived from explicit operational definitions over those observations; and **hypotheses**, inferred probabilistically from accumulated evidence. Each category occupies a distinct type in the system schema, with distinct identity semantics, distinct lifecycle rules, and distinct operations permitted upon it. No record exists in the system without belonging to exactly one category.

#### Structural Requirement

The category of every assertion is declared at construction and is not changeable. The schema prevents instances of one category from being read or written through the interfaces of another. Operations valid for one category — probabilistic merge for hypotheses, parametric re-derivation for operational constructs, append-only commitment for observations — are not exposed on records of other categories. Promotion of a hypothesis into operational use, or the use of an observation as input to inferential computation, requires explicit transformation through a typed boundary that produces a new record of the destination category, never reclassification of the original.

#### Rationale

The failure modes the Charter exists to prevent — recursive belief inflation, provenance collapse, irreplayable historical decisions — all originate in a single architectural error: treating inference and observation as interchangeable within a unified data model. Once that conflation is permitted, every downstream property degrades. Observations begin carrying inferential weight; hypotheses begin being treated as facts; the system loses the capacity to answer the most basic epistemic question it must answer, which is whether a given claim was seen or believed.

Epistemic Separation is the structural mechanism that makes this conflation impossible by construction, rather than discouraged by convention.

#### Forbidden Anti-Patterns

- **Unified assertion models.** Defining a single generic record type with a "kind" field distinguishing observation from inference from operational construct. This pattern surfaces routinely under simplification pressure and is precisely the failure the invariant prevents. Distinction must be carried by type, not by tag.
- **Implicit promotion of hypotheses to observations.** Using inferred entities (clusters, campaigns, behavioral groups) as inputs to systems that treat their inputs as observational. Promotion is a typed transformation with explicit provenance, never an interpretive shift.
- **Cross-category mutation interfaces.** Exposing operations such as "update assertion" that accept records of any category. Each category exposes only the mutations valid for it.
- **Reclassification of existing records.** Changing the declared category of an assertion after creation, for any reason. Misclassification is corrected by superseding the record, not by editing its category.

#### Boundary Conditions

The three categories share infrastructure — storage, transport, indexing, observability tooling — but not type. Shared infrastructure does not violate the invariant. Shared schema does. Structural commonality at the level of metadata fields (identifiers, timestamps, provenance references) is permitted and expected; structural commonality at the level of payload semantics is the boundary this invariant defends.

---

### 2.3 Provenance Integrity

#### Definition

Every Assertion declares its observational provenance structurally, via the typed reference field populated under the Q3-resolved schemas (per [`entity-model.md` Assertion entity](../ontology/entity-model.md#the-assertion-type-and-cross-category-references) post-[`§0016`](./decision-log.md)). The reference chain — transiting Category II operational constructs and Category III hypotheses as needed — terminates at underlying Category I primary observations. Given an Assertion and the immutability of the primary event log (per [§2.1](#21-observational-integrity)), the path from the Assertion back to those Category I primaries is structurally reconstructible.

#### Structural Requirement

The provenance chain's structural form is the typed `subject_ref_*` reference field on each Assertion, with oneOf-exclusivity enforced at the schemas layer (per [`§0016` Q3 resolution](./decision-log.md) — exactly one of `subject_ref_observation`, `subject_ref_construct`, or `subject_ref_hypothesis` is populated per Assertion). Category I primary observations — `DeclaredSession`, network-level events, fingerprint snapshots — are the chain termini (per [`§0015` Q1 resolution](./decision-log.md)). Chains crossing Category III continue to the hypothesis's lifecycle events, which are Category I records under [§2.5](#25-hypothesis-lifecycle-explicitness) Boundary Condition 5. The graph of provenance edges is typed-by-category (per [`provenance-model.md` §Observational Provenance](../ontology/provenance-model.md#observational-provenance) post-§0016), mirroring the [§2.2](#22-epistemic-separation) structural separation at the reference level.

Chains crossing identity boundaries — for example, `DeclaredSession` → `OperationalSession` — carry the identity-tier consistency-across-types default per [`§0015`](./decision-log.md); formal specification pending [Identity tiers — Open Modeling Question 1](../ontology/entity-model.md#open-modeling-questions). The default-level commitment is binding today; the formal mechanism becomes structurally falsifiable when Q2 resolution lands per [`§0014`](./decision-log.md) lazy-pre-Gate refinement methodology.

#### Rationale

Without structural reconstructibility from any Assertion back to its underlying Category I primary observations, the system loses the capacity to distinguish what was observed from what was inferred — the central failure mode the Charter exists to prevent (per [§1 Thesis](#1-thesis)). The reconstructibility guarantee rests on the substrate-immutability foundation of [§2.1](#21-observational-integrity): without immutable substrate, the chain has no stable terminus to reconstruct against. §2.3 inherits §2.1's mechanism and applies it to the provenance chain structurally, codifying that every Assertion's chain is anchored at Category I primaries rather than at projection-resident derivatives.

#### Forbidden Anti-Patterns

- **Orphan Assertion without typed reference.** An Assertion committed with zero populated typed `subject_ref_*` fields, or with a populated `subject_ref_X` that does not resolve to an existing substrate record. Schemas-level oneOf check plus reference-resolution check applied to each Assertion record.
- **Multi-populated subject_ref violation.** An Assertion committed with two or more `subject_ref_X` fields populated. Schemas-level oneOf check.
- **Aggregation destroying the chain through Cat II transit.** A Category II operational construct whose `subject_ref_observation` chain fails to resolve to the Category I primary observations from which the construct was derived. Substrate replay through typed `subject_ref_observation` edges; chain failure to terminate at Cat I primaries is the violation.
- **Cache-as-projection inversion.** Stores or projections that replace rather than supplement the provenance chain — a chain reconstructible from a projection but unreconstructible from the primary event log. Substrate-failure-with-projection-success is the falsifying signature.
- **Multi-category chain rupture.** A chain that terminates at a Category II construct or Category III hypothesis without continuing via typed edges to Category I primary observations. Substrate replay yields a chain whose terminal node is non-Cat-I.
- **Retroactive provenance rewriting.** Modification of a recorded `subject_ref_X` reference on an Assertion post-commit. [§2.1](#21-observational-integrity) substrate-immutability check applied to Assertion records: content-addressable identifier recomputation on read fails to match stored identifier.
- **Lifecycle event reference fabrication.** An Assertion's `subject_ref_hypothesis` resolving to a Hypothesis whose substrate lifecycle event chain is incomplete or unresolvable. Substrate replay from the target Hypothesis through [§2.5](#25-hypothesis-lifecycle-explicitness) Boundary Condition 5 lifecycle events; chain reconstruction failure is the violation.

#### Boundary Conditions

- **§2.3 governs observational provenance; not inferential influence.** The chain shape — `subject_ref_observation` edges terminating at Cat I primaries — is §2.3's scope. The inferential semantics of `subject_ref_construct` (Cat II) and `subject_ref_hypothesis` (Cat III) edges as influence relations are [§2.4](#24-inferential-influence-disclosure) territory; §2.3 reads the same edges as transit, not as influence.
- **§2.3 governs structural reconstructibility; not specific implementation parameters.** Schema technology selection (per [`§0003`](./decision-log.md)), index strategy, query mechanics, and projection-rebuild cadence live below §2.3.
- **§2.3 governs assertion-to-substrate provenance; not the substrate's own commit semantics.** Cat I substrate immutability is governed by [§2.1](#21-observational-integrity); Cat III lifecycle event commit semantics are governed by [§2.5](#25-hypothesis-lifecycle-explicitness). §2.3 inherits both; does not re-codify.
- **§2.3 does not govern Identity-tier formal specifics.** The identity-tier consistency-across-types default per [`§0015`](./decision-log.md) carries through provenance chains today; formal specification pending [Identity tiers — Open Modeling Question 1](../ontology/entity-model.md#open-modeling-questions). The default-level commitment is binding today; the formal mechanism becomes structurally falsifiable when Q2 resolution lands.
- **§2.3 governs the structural commitment of multi-category traversal; not the runtime mechanics of traversal.** The Charter-level commitment is that every chain has typed structure from Assertion to Category I primary observations across category boundaries (Cat II constructs and Cat III hypotheses as transit). Graph indexes, query layers, and projection-rebuild paths are architecture-document territory below §2.3.
- **The referenced entity's own structural commitments are not governed by §2.3.** A Category I observation referenced via `subject_ref_observation` has commitments under [§2.1](#21-observational-integrity) and [`entity-model.md`](../ontology/entity-model.md) Cat I; a Category II construct referenced via `subject_ref_construct` has commitments under [§2.2](#22-epistemic-separation) and Q1 determinism; a Category III hypothesis referenced via `subject_ref_hypothesis` has commitments under [§2.5](#25-hypothesis-lifecycle-explicitness). §2.3 requires the typed reference and the chain shape only.

---

### 2.4 Inferential Influence Disclosure

> **Status:** Pending committee redaction.
>
> **Working definition (non-binding):** When a hypothesis is promoted to use as enrichment context, every assertion subsequently formed under that influence carries a structural declaration of the influence. The system preserves, by construction, the distinction between belief grounded in independent evidence and belief inherited recursively from earlier conclusions.
>
> **Anti-pattern this invariant will forbid:** silent promotion of hypotheses to enrichment; invisible recursive reinforcement; loss of the distinction between fresh evidence and inherited belief.
>
> **This is the invariant that most directly defends against the central failure mode described in the Thesis.**

---

### 2.5 Hypothesis Lifecycle Explicitness

#### Definition

Operations on hypothesis records — recorded as immutable lifecycle events in the primary event log — constitute the operation history from which the current state of any hypothesis is reconstructed. The hypothesis type system is structured per [Charter §2.2](#22-epistemic-separation) and the Q2-A.2 resolution ([decision-log §0010](./decision-log.md)) as an abstract type `Hypothesis` with four concrete sibling subtypes (`BehavioralCluster`, `CoordinationRing`, `CampaignHypothesis`, `AutomationGroup`); the six lifecycle operations — formation, merge, split, promotion, demotion, dissolution — are defined at the abstract-type level. The current state of any hypothesis is a projection over the operation history applied to it; no record exists whose declared type is "current hypothesis state" outside of this projection.

#### Structural Requirement

Every operation on a hypothesis — formation, merge, split, promotion, demotion, dissolution — is recorded as an immutable lifecycle event in the primary event log under the substrate-immutability guarantee of [§2.1](#21-observational-integrity). Merge events reference all antecedent hypotheses; cross-subtype merge produces a typed output record (specific produced-type semantics deferred to [`lifecycle-semantics.md`](../ontology/lifecycle-semantics.md) per [decision-log §0011](./decision-log.md)). Promotion events carry the structural parameters governing the promoted hypothesis's subsequent demotion-candidacy.

A promoted hypothesis becomes a demotion candidate when both of the following hold (staged-combination AND form per [decision-log §0011](./decision-log.md)):

1. **Layer A.** The elapsed time since the promotion event exceeds a parameter recorded on the promotion event or on the hypothesis's concrete subtype.
2. **Layer B.** A designated structural test on `evidential independence` (per [Charter §2.6](#26-evidential-independence-integrity) (pending)) or on declared `influence` (per [Charter §2.4](#24-inferential-influence-disclosure) (pending)) fires.

The structural form of Layer B is deferred to the follow-on RFC tied to §2.4 and §2.6 redactions; the falsifiability obligation becomes binding when those siblings are redacted. Demotion itself, once a candidate is confirmed, is recorded as an immutable lifecycle event referencing the prior promotion event.

#### Rationale

Without explicit recording of each lifecycle event in immutable form, the system cannot answer the question of what it concluded — or had grounds to conclude — at any prior moment. The substrate-immutability guarantee of [§2.1](#21-observational-integrity) ensures the operation history, once recorded, remains reconstructible; §2.5 inherits this guarantee for the Category I subclass of records that lifecycle events constitute.

Hypothesis-state-as-projection rather than as a mutable record is the structural mechanism by which the operation history retains its primacy. A current-state record stored as if it were primary would invert the substrate-projection relation, degrading lifecycle events to audit trail. The invariant prevents this inversion by construction.

#### Forbidden Anti-Patterns

- **Direct mutation of hypothesis state outside append-only commit semantics per [§2.1](#21-observational-integrity).** A projection write to a hypothesis record's state field with no corresponding substrate lifecycle event. Detectable by comparison of projection output against substrate replay.
- **Loss of any lifecycle event from a hypothesis's operation history.** A hypothesis present in projections whose substrate operation-event chain is incomplete. Compaction policies that lose individual lifecycle events violate this anti-pattern; physical reorganization that preserves reconstructibility does not.
- **Hypothesis-merge events without recorded antecedents.** A merge event whose antecedent reference is empty or fails to resolve to existing Hypothesis records.
- **Modification or annotation of a committed lifecycle event with revised content.** Lifecycle events are Category I substrate records under [§2.1](#21-observational-integrity); the substrate write-once guarantee applies. Detectable by content-addressable identifier recomputation.
- **Cross-subtype Hypothesis-merge committed without recording the typed-output rule.** A merge event whose antecedent subtypes differ AND whose produced-type derivation is unspecified or unresolvable.
- **Demotion candidacy recorded as satisfied while either Layer A or Layer B alone holds (not both).** The staged-combination AND form structurally rejects single-layer satisfaction.
- **Storing current hypothesis state as a primary substrate record rather than as a projection over the operation event history.** The substrate-projection relation is inverted by such direct commit; the operation event chain ceases to be the source-of-record.

#### Boundary Conditions

- **Categories I and II are not governed by §2.5.** Category I observation lifecycle is governed by [§2.1](#21-observational-integrity); Category II construct supersession is governed by [§2.2](#22-epistemic-separation). §2.5 governs Category III hypothesis lifecycle specifically.
- **Parameter values are not governed by §2.5.** Cadence parameter values for Layer A and the eventual Layer B parameters live in promotion event payloads or in concrete subtype declarations. §2.5 governs the form of the staged-combination criterion (AND of Layer A and Layer B); it does not codify specific numerical values.
- **Layer B operationalization is not governed by §2.5.** The structural form of Layer B's deep criterion is deferred to the follow-on RFC tied to [§2.4](#24-inferential-influence-disclosure) (pending) and [§2.6](#26-evidential-independence-integrity) (pending) redactions. §2.5 carries the forward-reference contract; the operationalization is the follow-on RFC's territory.
- **Cross-subtype merge produced-type semantics are not governed by §2.5.** §2.5 codifies the typed-transformation requirement. Whether the produced record is a third concrete subtype or an abstract record with subtype-elision is deferred to [`lifecycle-semantics.md`](../ontology/lifecycle-semantics.md) post-Q4 follow-on.
- **The referenced Category III entity's own structural commitments are not governed by §2.5.** A lifecycle event is a Category I record carrying a reference-and-parameters payload about a Category III hypothesis. §2.5 governs (a) the lifecycle event's category — Category I commit semantics inherited from §2.1 — and (b) the event's content-as-reference (specific hypothesis, operation, antecedents, parameters). The referenced hypothesis's own structural commitments — provenance, confidence, evidential independence, declared influence — are [§2.3](#23-provenance-integrity), [§2.6](#26-evidential-independence-integrity), [§2.4](#24-inferential-influence-disclosure) territory respectively.

---

### 2.6 Evidential Independence Integrity

> **Status:** Pending committee redaction.
>
> **Working definition (non-binding):** Inferential assertions carry, structurally, two distinct dimensions: magnitude of confidence and degree of evidential independence. Reporting only one is insufficient. The schema requires both. Their separation is the structural defense against recursive belief inflation.
>
> **Anti-pattern this invariant will forbid:** assertions reported with confidence alone; independence calculated only in offline analyses; collapse of independence into confidence under simplification pressure.

---

## 3. Non-Goals

> **Status:** Pending committee redaction.
>
> Non-Goals are not a defensive appendix. They are the negative perimeter of the system's identity — direction explicitly rejected, not merely deprioritized. This section will receive committee treatment equal to the invariants.
>
> **Anticipated non-goals (non-binding):**
>
> - Ghost Trace does not produce truth. It maintains the distinction between observation, inference, and inherited belief.
> - Ghost Trace does not perform universal identity resolution. Identity reconciliation is a domain-specific concern subordinate to the substrate.
> - Ghost Trace does not automate irreversible operational action. Actions of consequence are taken by external systems with their own accountability.
> - Ghost Trace does not optimize for the lowest operational complexity. Where simplicity conflicts with epistemic integrity, integrity prevails.
> - Ghost Trace is not a generic event-sourcing framework. Its specificity to behavioral intelligence is constitutional, not incidental.

---

## 4. Constitutional Design Rule

### Definition

This section governs two disciplines applied to every candidate constitutional invariant of Ghost Trace.

**Qualification.** A claim qualifies as a constitutional invariant if and only if it satisfies the four criteria stated in [Section 2 — Constitutional Invariants](#2-constitutional-invariants), reproduced here for anchor purposes (canonical statement remains in §2):

> 1. **Structurally enforceable** — verifiable in schemas, types, or permitted operations, not merely in code review.
> 2. **Constraining of future implementation decisions** — capable of rejecting proposals that violate it.
> 3. **Identity-defining** — its absence changes what the system fundamentally is, not merely what it does.
> 4. **Independent of operator interpretation** — violation is detectable without subjective judgment.
>
> — [Charter §2](#2-constitutional-invariants)

**Falsifiability.** A constitutional claim is admissible if and only if it is structurally falsifiable. A property that cannot, in principle, be violated, observed, or audited is not a constitutional property; it is an aspiration, an aesthetic preference, or a research direction, and belongs elsewhere.

### Structural Requirement

The two disciplines are applied at amendment time. The [`amendments.md` §Amendment Process](./amendments.md) procedure requires falsifiability review (Step 2) before any proposal advances to redaction (Step 3). The four qualification criteria are tested at the redaction stage and again at the final-merge checklist of the [`invariant-redactor`](../../.claude/skills/constitutional/invariant-redactor/SKILL.md) skill. The four-question falsifiability test — violation, observation, operationalization, non-circularity — is operationalized in the [`falsifiability-check`](../../.claude/skills/epistemic/falsifiability-check/SKILL.md) skill.

### Rationale

The Charter constrains the system; this section constrains the Charter. Without qualification and falsifiability disciplines, any prose declared "constitutional" would carry the same weight as the structurally-enforceable invariants of §2.1 and §2.2, and the meaning of "constitutional" would collapse into the meaning of "important to someone."

The falsifiability discipline applies to all constitutional claims, including the claims of this section. The recursion is not vicious: the test procedure is defined externally to §4, in `falsifiability-check` §1, and §4's claims reduce to procedural artifacts (qualification testing at amendment time; falsifiability review at amendment time). The chain bottoms out in procedure, not in self-reference. This is the fixed-point reading.

### Forbidden Anti-Patterns

- **Adopting an invariant whose violation requires subjective judgment.** Fails criterion 4. Detected by the observation test of [`falsifiability-check` §1.2](../../.claude/skills/epistemic/falsifiability-check/SKILL.md).
- **Adopting an invariant not structurally enforceable in schemas, types, or permitted operations.** Fails criterion 1. Detected by the operationalization test of [`falsifiability-check` §1.3](../../.claude/skills/epistemic/falsifiability-check/SKILL.md).
- **Adopting an invariant that does not constrain future implementation decisions.** Fails criterion 2. Surfaced at the `invariant-redactor` final-merge checklist; no per-claim mechanical check.
- **Adopting an invariant whose absence would not change what the system is.** Fails criterion 3. Surfaced at the `invariant-redactor` final-merge checklist; no per-claim mechanical check.

### Boundary Conditions

- §4 does not govern internal project practice outside Charter governance. Commit message conventions, branch naming, code style, and README phrasing are operational and belong to [`CONTRIBUTING.md`](../../CONTRIBUTING.md) (process) and [`WORKFLOW.md`](../../WORKFLOW.md) (tooling).
- §4 governs the form of invariants, not their content. The committee chooses which invariants the project adopts; §4 filters candidate invariants into qualified versus non-qualified.
- §4 does not govern the infrastructure that supports Charter governance. Skills, hooks, CI workflows, agents, slash-commands, and per-project settings can be modified, replaced, or extended without Charter amendment, subject to RFC and decision-log discipline.

---

## Related Documents

- [`amendments.md`](./amendments.md) — formal amendments to this Charter
- [`decision-log.md`](./decision-log.md) — architectural decision record
- [`../ontology/ontology.md`](../ontology/ontology.md) — formalization of concepts introduced here
- [`../architecture/`](../architecture/) — architectural treatments derived from this Charter
- [`../rfcs/`](../rfcs/) — proposals subject to constitutional review
