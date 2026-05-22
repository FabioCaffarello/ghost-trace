# RFC — Layer B Deep Criterion (Q4 follow-on)

- **Status:** discussion (active; §2.4 v0.5 + §2.6 v0.6 surfaces present per [`§0011`](../../charter/decision-log.md) contract activation completion at [`§0129`](../../charter/decision-log.md))
- **Authors:** Ghost Trace committee
- **Date:** 2026-05-15
- **Type:** ontology-revision
- **Affects:** [`docs/ontology/lifecycle-semantics.md`](../../ontology/lifecycle-semantics.md) §The Promotion Mechanism step 4; [Charter §2.5](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) (pending — binding text refinement once Layer B is specified)

> RFCs in Ghost Trace are proposals subject to constitutional review. The [Constitutional Charter](../../charter/constitutional-charter.md) is the authoritative reference. RFCs that conflict with the Charter must either be rejected or accompanied by a corresponding charter amendment (see [`../../charter/amendments.md`](../../charter/amendments.md)).

## Status note

**This RFC is a placeholder; Charter dependencies + half of Ontology dependencies are now discharged.** Substantive content was originally gated on four conditions: §2.4 redaction, §2.6 redaction, [`docs/ontology/ontology.md` §Open Questions](../../ontology/ontology.md) Q3 (formal definition of independence), and Q5 (influence propagation). Discharge status:

- **§2.4** frozen v0.5 at [`§0099`](../../charter/decision-log.md). ✓
- **§2.6** frozen v0.6 at [`§0129`](../../charter/decision-log.md). ✓
- **Q3 (formal definition of `evidential_independence`)** resolved at [`§0133`](../../charter/decision-log.md): Candidate α (source-count ratio over Cat I provenance roots) adopted under [§2.6 BC2](../../charter/constitutional-charter.md#26-evidential-independence-integrity) meta-shape 1 (deterministic-from-pattern). ✓
- **Q5 (influence propagation)** partially discharged: decay half resolved at [`§0020`](../../charter/decision-log.md); transitivity half opened at `discussion` status as RFC [`ontology-revision-q5-influence-propagation-transitivity`](./ontology-revision-q5-influence-propagation-transitivity.md) per [`§0133`](../../charter/decision-log.md) cascade-enactment from Q3, with companion discussion-phase scratch [`q5-transitivity-evidence.md`](../discussion/q5-transitivity-evidence.md). Resolution pending.

Layer B substantive content advances after Q5 transitivity-half resolves per the two-cascade chain Q3 → Q5 → Layer B. With Q3-α adopted, Layer B's deep criterion now has a measurable quantity to threshold-test; Q5's transitivity-half resolution determines α's "reachable" predicate's structural semantic, which is the last ontology-side dependency before Layer B's substantive content can be drafted.

## Summary

[`decision-log.md` §0011](../../charter/decision-log.md) (Q4 resolution) adopted the staged-combination form for the demotion-candidacy criterion: Layer A (time-based cadence gate, operational today) AND Layer B (deep criterion on `evidential independence` or declared `influence`). Layer B's specific structural form — which combination of evidence-staleness (Candidate B family from Q4) and/or influence-saturation (Candidate C family from Q4) constitutes the deep criterion — is deferred to this RFC. Charter-side surfaces present: §2.4 frozen v0.5; §2.6 frozen v0.6. Ontology-side: Q3 resolved at [`§0133`](../../charter/decision-log.md) (Candidate α — source-count ratio); Q5 transitivity-half opened at [`ontology-revision-q5-influence-propagation-transitivity`](./ontology-revision-q5-influence-propagation-transitivity.md) per [`§0133`](../../charter/decision-log.md) cascade-enactment. Layer B's substantive content emerges once Q5 transitivity-half resolves.

## Motivation

The Q4 resolution committed to a structural form without specifying Layer B's deep criterion in order to permit §2.5 redaction to proceed per [`decision-log.md` §0008](../../charter/decision-log.md)'s redaction order. Without this follow-on RFC, the deferred question would lack a canonical locus and would risk silent resolution by future implementation work or by a §2.5 binding-text refinement post-§2.4. The placeholder preserves the question's procedural visibility and anchors future deliberation.

The cost of not opening this placeholder: §2.5 binding text would forward-reference a question with no documented home; the home would emerge implicitly as part of §2.4 or §2.6 redaction without independent committee deliberation. The placeholder is the procedural defense against that implicit-home failure mode.

## Constitutional Review

Per [`rfc-author` §1](../../../.claude/skills/workflow/rfc-author/SKILL.md), the Q1–Q6 impact analysis applies. Each question's answer is provisional at the placeholder stage; the analysis is re-applied substantively when the RFC advances post-§2.4 + post-§2.6.

### Q1 — Which Charter invariants does this RFC touch?

- **§2.5 Hypothesis Lifecycle Explicitness** (pending): touched indirectly — Layer B's specification refines the binding text §2.5 redaction produces.
- **§2.6 Evidential Independence Integrity** (pending): touched if the deep criterion lands in Candidate B's family. The structural form of `evidential independence` per §2.6 is Layer B's input.
- **§2.4 Inferential Influence Disclosure** (pending): touched if the deep criterion lands in Candidate C's family. Declared `influence` per §2.4 is Layer B's input. Also touched by the structural exclusion mechanism Q4 Phase 3 Finding 6 named (B's non-circularity requires §2.4's structural test for "formed under this hypothesis's influence").

### Q2 — Does this RFC implicitly redefine any term in the glossary?

To be re-applied substantively post-§2.4 + post-§2.6. The placeholder uses canonical vocabulary (`evidential independence`, `influence`, `confidence`, `promotion`, `demotion`); no redefinition.

### Q3 — Does this RFC implicitly resolve any of the five open Ontology questions?

This RFC's eventual content resolves no Ontology open question by itself, but composes with the resolutions of Q3 (formal definition of independence) and Q5 (influence propagation). The composition is named explicitly here so the RFC's substantive form is anchored to those resolutions, not silently resolving them.

### Q4 — Does this RFC require Charter amendment?

No. Layer B's specification refines §2.5 binding text after §2.5 is initially redacted; refinement is procedurally an Ontology revision, not a Charter amendment, provided the §2.5 binding text already forward-references Layer B in its initial redaction.

### Q5 — Does this RFC introduce a new invariant?

No. The RFC specifies the deep criterion that Layer B references; the invariant that consumes Layer B is §2.5.

### Q6 — Does this RFC propose ceremony without behavioral consequence?

No. Layer B's specification is the structural test that determines whether a hypothesis whose Layer A gate has fired actually becomes a demotion candidate. The behavioral consequence is direct.

## Proposal

To be filled when the RFC advances. The structural shape of the eventual proposal is constrained by:

- **Two candidate families inherited from Q4 discussion:** Candidate B (evidence-staleness) and Candidate C (influence-saturation), per [`q4-evidence.md` Phase 1](../discussion/q4-evidence.md). Layer B may adopt B alone, C alone, or both (disjunctive or conjunctive within Layer B itself, independently of the outer Layer A AND Layer B composition).
- **Q4 Phase 3 Finding 6:** if Layer B includes Candidate B, the binding prose must structurally subtract hypothesis-influenced assertions from the freshness denominator. The structural test for "formed under this hypothesis's influence" is what §2.4 must supply.
- **Q5 dependency:** if Layer B includes Candidate C, the operational form depends on Q5's resolution of how `influence` propagates (transitive, decaying, both, or other).
- **AND-versus-OR composition between Layer A and Layer B:** the outer composition is currently AND per [`decision-log.md` §0011](../../charter/decision-log.md). If a future RFC reverses to OR (per §0011's fragility record), Layer B's role shifts from "deep gate after Layer A fires" to "independent demotion trigger". This RFC's eventual specification must accommodate either composition without requiring a separate revision.

## Alternatives Considered

To be enumerated substantively when the RFC advances. The Q4 discussion phase enumerated Candidates A, B, C plus combinations plus a meta-pattern; the Q4 resolution selected the staged-combination form with Layer B deferred to this RFC. The alternatives this RFC explicitly considers in its substantive phase will inherit from Q4's Phase 1 enumeration and may add candidates that surface post-§2.4 + post-§2.6.

## Open Questions

The placeholder records the following as the RFC's own open questions to be resolved when it advances:

- **Which family (B, C, or both) constitutes Layer B?**
- **If both, what composition (disjunctive, conjunctive, staged) within Layer B itself?**
- **What are the parameter values (threshold T for B-family; ratio K and scope for C-family)?** Parameter values are themselves deferrable to a further RFC if the structural form is the more urgent commitment.
- **Per-subtype vs uniform parameters under Q2-A.2:** Layer B's parameters may live at the abstract `Hypothesis` level or per-concrete-subtype. The choice depends on whether the four concrete subtypes' provenance neighborhoods diverge meaningfully.

## Anti-Patterns to Avoid

To be enumerated substantively when the RFC advances. Q4 discussion phase F-DRIFT-2 surfaced the vocabulary-collapse anti-pattern; this RFC's eventual content must preserve the two-dimension separation of `confidence` and `evidential independence`.

## Migration and Backward Compatibility

No historical Category III records exist at this point. Layer B's specification is forward-looking. The placeholder records that Layer B's eventual binding text must be expressible without retroactive substrate revision: a §2.5 binding text initially redacted with Layer B forward-referenced as "deferred to this RFC" must remain valid prose; this RFC's resolution refines the forward reference, not the §2.5 invariant itself.

## References

- [`docs/charter/decision-log.md` §0011 — Q4 resolution](../../charter/decision-log.md) — the resolution that opened this RFC.
- [`docs/rfcs/discussion/q4-evidence.md`](../discussion/q4-evidence.md) — Q4 discussion phase evidence.
- [`docs/rfcs/draft/ontology-revision-q4-promotion-demotion-criterion.md`](./ontology-revision-q4-promotion-demotion-criterion.md) — Q4 RFC (accepted, decision-log §0011).
- [`docs/ontology/lifecycle-semantics.md` §The Promotion Mechanism](../../ontology/lifecycle-semantics.md) — current binding home of Layer B's forward reference.
- [`docs/charter/constitutional-charter.md` §2.4](../../charter/constitutional-charter.md#24-inferential-influence-disclosure), [§2.5](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness), [§2.6](../../charter/constitutional-charter.md#26-evidential-independence-integrity).
- [`docs/ontology/ontology.md` §Open Questions](../../ontology/ontology.md) — Q3, Q5 (pending Ontology questions Layer B depends on).

## Decision Record

Pending. This RFC is on hold; its substantive content will be drafted when §2.4 and §2.6 are redacted and ontology.md Q3 and Q5 resolved. A decision-log entry will be assigned when the RFC advances and the committee resolves Layer B's deep criterion.
