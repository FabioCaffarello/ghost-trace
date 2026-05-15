# Ghost Trace Constitutional Charter

**Version:** v0.2.1 (draft, sections in committee mode)
**Status:** Thesis frozen. Invariant qualification criteria (§2 header) frozen. Invariants 2.1–2.2 frozen. Invariants 2.3–2.6 pending committee redaction. Non-Goals (§3) pending. Constitutional Design Rule (§4) frozen — minor amendment v0.2. Patch amendment v0.2.1 extends mechanical Charter-blockquote exemption to vocabulary-drift (no Charter prose amended; see [`decision-log.md` §0012](./decision-log.md)).

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

The criteria above are themselves recorded formally in [Section 4 — Constitutional Design Rule](#4-constitutional-design-rule) (pending). They are applied as meta-rule to all invariants in this section.

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

> **Status:** Pending committee redaction.
>
> **Working definition (non-binding):** Every assertion declares, in its structure, the observations and prior assertions that constitute its provenance. The capacity to reconstruct the path from any assertion back to underlying observations is guaranteed by the immutability properties of the primary event log.
>
> **Anti-pattern this invariant will forbid:** orphan assertions; aggregations that destroy granular traceability; caches that replace rather than supplement the provenance path.
>
> **Distinct from Inferential Influence Disclosure:** this invariant addresses *observational* provenance — where the assertion came from. The next invariant addresses *inferential* provenance — which prior beliefs influenced its formation.

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

> **Status:** Pending committee redaction.
>
> **Working definition (non-binding):** Operations on hypotheses — formation, merge, split, dissolution, promotion, demotion — are recorded as immutable events in the primary event log. The current state of any hypothesis is a projection over the history of operations applied to it, never the result of direct mutation.
>
> **Anti-pattern this invariant will forbid:** direct mutation of hypothesis state in graph or document stores; loss of evolution history; cluster merges without recorded antecedents.

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
