# RFC — Ontology revision: Authentication-class typing on Cat I observation envelopes (F4 OMQ resolution)

- **Status:** discussion (framing PR; substantive deliberation Phase 2–5 pending in evidence file; resolution pending decision-log entry)
- **Authors:** Ghost Trace committee (F4 OMQ trigger fired at [`decision-log §0147`](../../charter/decision-log.md) when F1 reached 3/3 modalities stable; opened per the §0143 anticipated-OMQs table disposition; discussion-phase deliberation in [`authentication-class-typing-evidence.md`](../discussion/authentication-class-typing-evidence.md))
- **Date:** 2026-05-23 (opened)
- **Type:** ontology-revision
- **Affects:** [`docs/ontology/entity-model.md`](../../ontology/entity-model.md) (registers F4 OMQ in §Open Modeling Questions; resolution will close the OMQ + revise §Category I to carry the chosen surface). [`schemas/events/v1/network_observation.proto`](../../../schemas/events/v1/network_observation.proto) + [`schemas/events/v1/behavioral_observation.proto`](../../../schemas/events/v1/behavioral_observation.proto) + [`schemas/events/v1/attestation_observation.proto`](../../../schemas/events/v1/attestation_observation.proto) + future [`schemas/events/v1/browser_observation.proto`](../../../schemas/events/v1/) (depending on resolution form: may add envelope field, may introduce subtype hierarchy, may modify `actor_ref` semantics). No Charter prose modification anticipated at framing; specific resolutions may require Charter amendment (assessed per resolution candidate).

> RFCs in Ghost Trace are proposals subject to constitutional review. The [Constitutional Charter](../../charter/constitutional-charter.md) is the authoritative reference. RFCs that conflict with the Charter must either be rejected or accompanied by a corresponding charter amendment (see [`../../charter/amendments.md`](../../charter/amendments.md)). This RFC opens an Ontology Open Modeling Question that was anticipated at [`§0143`](../../charter/decision-log.md) and trigger-fired at [`§0147`](../../charter/decision-log.md); the OMQ is registered in [`entity-model.md` §Open Modeling Questions](../../ontology/entity-model.md#open-modeling-questions) by this PR.

## Status note — F4 OMQ trigger fired; anticipated → trigger-fired → opening-for-resolution

The OMQ's anticipated-OMQ lifecycle (per [`§0147`](../../charter/decision-log.md) Methodological observation 1):

| Stage | Decision-log anchor | Status before this RFC | Status after this RFC |
|---|---|---|---|
| Anticipated | [`§0143`](../../charter/decision-log.md) anticipated-OMQs table | named with trigger condition | unchanged |
| Trigger condition satisfied | [`§0147`](../../charter/decision-log.md) (F1 reaches 3/3 modalities stable) | not yet | satisfied at AttestationObservation landing |
| Opened in discussion | This RFC | (this RFC) | OPENS |
| Substantive deliberation | Evidence file Phase 2–5 | not begun | begins after framing PR merges |
| Resolution | Future decision-log entry | pending | pending |

## Summary

F1 introduced four typed Cat I observation modalities under Domain Pack v0.1 ([`§0143`](../../charter/decision-log.md)): NetworkObservation ([`§0144`](../../charter/decision-log.md)), BehavioralObservation ([`§0146`](../../charter/decision-log.md)), AttestationObservation ([`§0147`](../../charter/decision-log.md)), and the planned BrowserObservation. The three landed modalities expose a structural distinction the current envelope does NOT carry: observations differ in **how the observed actor's identity was verified at observation time**.

Three classes are empirically forced by the F1 modality set:

- **server-authenticated** — observation collector verified the actor identity server-side (typical of NetworkObservation collected at a gateway with mTLS termination or similar server-trusted authentication). Substrate-side: high confidence that the observation pertains to the named actor.
- **client-attested** — observation carries a cryptographic attestation chain that the substrate may re-verify (typical of AttestationObservation with WebAuthn assertion, FIDO2 attestation, or a redeemed Privacy Pass token). Substrate-side: actor identity verifiable via the attestation chain.
- **client-witnessed** — observation was reported by a producer running in untrusted territory (typical of BehavioralObservation collected by a browser SDK; the producer reports keystroke timings but the substrate has no cryptographic anchor for the actor identity claim). Substrate-side: actor identity is producer-attested; no cryptographic verification at substrate level.

The substrate's envelope (`observed_at` + `actor_ref` + `session_ref` + `collector_ref` + `oneof modality`) does NOT structurally distinguish these classes. F3 inference layer would silently consume them as equivalent unless the substrate surface forces the distinction. The OMQ asks: **what structural surface should the substrate carry to make authentication-class explicit at the Cat I observation layer?**

This RFC opens structured discussion. Candidate resolution forms enumerated in [`authentication-class-typing-evidence.md`](../discussion/authentication-class-typing-evidence.md) Phase 2. This RFC does not pick a candidate at framing.

## Motivation

### Why now

[`§0147`](../../charter/decision-log.md) closed the third F1 modality landing (AttestationObservation), satisfying the §0143 trigger condition for this OMQ (F1 has three modalities stable). The trigger fired empirically: NetworkObservation observations carry server-authenticated semantics implicitly (via the collector's gateway position), BehavioralObservation carries client-witnessed semantics (via the producer's browser-SDK position), AttestationObservation carries client-attested semantics (via the cryptographic chain). No envelope field today records the class; F3 cannot distinguish without the structural surface.

Opening this RFC immediately (rather than waiting for BrowserObservation landing) follows the anticipated-OMQ lifecycle codified at [`§0147`](../../charter/decision-log.md) Methodological observation 1: anticipated → trigger-fired → opening-for-resolution-via-separate-RFC. BrowserObservation extends the same pressure (client-witnessed at JS-collected fingerprint level) but does not change the OMQ's structural shape; resolution before BrowserObservation lands avoids retrofit work at the fourth modality.

### Cost of not resolving

F3 inference layer is the most pressing consumer. Two failure modes:

- **Silent class conflation.** F3 signatures that compute over actor cross-modal observations cannot distinguish observations of different authentication classes. A signature trusting BehavioralObservation timing patterns equivalently to AttestationObservation cryptographic chains over-trusts the former.
- **Operator-side workaround proliferation.** Without a structural envelope surface, operators may encode authentication-class in `collector_ref` strings (e.g., `browser-sdk-client-witnessed:v1`), creating an informal convention that varies across collectors and breaks the §0144 (e) phenomenon-vs-record reconciliation OMQ surface (collector_ref namespace would mix collector identity with authentication-class semantics).

Both failure modes are observable; the OMQ resolution is structurally non-optional.

### Downstream consequence

The resolution will likely require:

1. Schema change to all four F1 envelopes (Network + Behavioral + Attestation + Browser-when-it-lands) — a schemas-evolution event per the canonical-serialization-contract.
2. Entity-model.md revision of §Category I to codify the authentication-class structural distinction.
3. Possibly Charter §2.3 or §2.4 commentary on how authentication-class composes with provenance + influence semantics (TBD per resolution candidate).
4. Possibly a Charter amendment if the resolution surfaces a structural commitment the Charter does not currently anticipate.

## Constitutional Review

The Q1–Q6 impact analysis prescribed by [`rfc-author` §1](../../../.claude/skills/workflow/rfc-author/SKILL.md).

### Q1 — Which Charter invariants does this RFC touch?

- **§2.1 Observational Integrity** (frozen): touched indirectly. Authentication-class is a property of the observation; whichever structural surface carries it (envelope field, subtype, paired enrichment record) inherits §2.1 immutability per Cat I.
- **§2.3 Provenance Integrity** (frozen v0.4): touched indirectly. The authentication-class surface may participate in provenance chain semantics — e.g., a Cat II construct derived from a client-witnessed observation may carry forward the lower-confidence authentication context. Resolution candidates that introduce per-edge authentication-class annotation in the provenance chain are Charter-adjacent and assessed at resolution.
- **§2.4 Inferential Influence Disclosure** (frozen v0.5): touched indirectly. F3 signature engines reading authentication-class to gate influence chains is operational specification; not Charter modification.
- **§2.5 Hypothesis Lifecycle Explicitness** (frozen v0.3): not affected.
- **§2.6 Evidential Independence Integrity** (frozen v0.6): touched possibly. If authentication-class participates in `evidential_independence` derivation (e.g., client-witnessed roots count differently from server-authenticated roots in the [`§0133`](../../charter/decision-log.md) Candidate α source-count ratio), the resolution interacts with §2.6 BC2. Assessed per resolution candidate.
- **§3 N1 — no truth at substrate** (frozen): touched directly. Authentication-class is structural (which channel verified the observation) NOT truth-bearing (whether the actor is real). The resolution MUST preserve the structural-not-truth-bearing distinction; candidates that encode authentication-class as a confidence-like dimension on Cat I would violate §3 N1.

### Q2 — New glossary terms?

Depends on resolution candidate. Per evidence Phase 2 candidate enumeration, likely new terms: `authentication-class`, `server-authenticated`, `client-attested`, `client-witnessed`. Glossary entries added at resolution commit, not at framing.

### Q3 — Does this RFC implicitly resolve any of the five original Ontology open questions?

No. The five original `ontology.md` Open Questions (Q1–Q5) are resolved as of [`§0134`](../../charter/decision-log.md). This RFC opens a NEW OMQ at the entity-model.md Open Modeling Questions section.

### Q4 — Does this RFC require Charter amendment?

**Possibly, per resolution candidate.** Most candidates in evidence Phase 2 do NOT require Charter amendment (they revise entity-model.md + schemas only, under §3 N1 + §2.1 + §2.3 existing scope). A subset of candidates may require Charter amendment if they introduce a structural commitment the Charter does not currently anticipate (e.g., elevating authentication-class to a §2.x invariant). Assessed at resolution; not pre-empted at framing.

### Q5 — Does this RFC introduce a new invariant?

Depends on resolution. The OMQ is structurally Ontology-level; the default resolution stays at Ontology + schemas layer. A subset of candidates may motivate a new Charter invariant (assessed at resolution).

### Q6 — Does this RFC propose ceremony without behavioral consequence?

No. Behavioral consequences cascade through F3 signature consumption (per Motivation §Cost of not resolving), through F1 envelope surface (every F1 observation receives or does not receive the class field), through canonical-serialization-contract (schemas-evolution event at resolution commit), through entity-model.md (§Category I surface revision). The choice between candidates is materially different at each level.

## Proposal

Candidate resolution forms enumerated in [`authentication-class-typing-evidence.md`](../discussion/authentication-class-typing-evidence.md) Phase 2 (substantive deliberation pending). At framing, six candidate families:

- **Candidate α — Envelope enum field.** Add `authentication_class` field (proto3 enum with values `UNKNOWN = 0`, `SERVER_AUTHENTICATED = 1`, `CLIENT_ATTESTED = 2`, `CLIENT_WITNESSED = 3`) to each F1 envelope (NetworkObservation, BehavioralObservation, AttestationObservation, BrowserObservation). Producer-supplied at observation time.
- **Candidate β — New Cat I subtype hierarchy.** Introduce three abstract Cat I subtypes (`ServerAuthenticatedObservation`, `ClientAttestedObservation`, `ClientWitnessedObservation`) that the four F1 modalities extend. Subtype identifies the class structurally.
- **Candidate γ — `actor_ref` discriminator prefix.** Encode authentication-class in `actor_ref` string via prefix convention (`sa:actor-x`, `ca:actor-x`, `cw:actor-x`). No proto-definition change; producer-discipline.
- **Candidate δ — Paired enrichment record.** New Cat I record `AuthenticationContext` paired with each observation at commit time (parallel to IngestionEvent pairing per [`§0038`](../../charter/decision-log.md)). Carries authentication-class + optional verification anchor.
- **Candidate ε — `collector_ref` semantic discipline.** Standardize `collector_ref` naming convention to encode authentication-class (`server-collector:...`, `attested-collector:...`, `witnessed-collector:...`). No proto-definition change; operator-discipline.
- **Candidate ζ — Composite (envelope enum + paired enrichment record).** Combine α's quick lookup with δ's structural richness.

Each candidate has distinct properties around: proto-definition evolution cost, producer effort, F3 consumer simplicity, retrofit complexity for the three already-landed F1 modalities, §3 N1 truth-vs-structure boundary, and forward compatibility with the planned BrowserObservation. Phase 2 evidence file enumerates per-candidate analysis.

## Alternatives Considered

- **Defer until BrowserObservation lands.** Rejected — would force BrowserObservation to land under an unresolved structural distinction, multiplying retrofit work. Per §0147 Methodological observation 1: trigger-fired OMQs should open-for-resolution before further empirical pressure accumulates.
- **Resolve via decision-log entry without RFC.** Rejected — the resolution affects entity-model.md + schemas across four F1 modalities; per Ontology RFC discipline ([`§0014`](../../charter/decision-log.md) lazy pre-Gate methodology + RFC-template Q3 surface), Ontology-level structural changes require RFC.
- **Skip the OMQ; rely on operator convention.** Rejected — Candidate γ + ε attempt this; both surface as candidates for deliberation but the convention-only path violates the [`§0144`](../../charter/decision-log.md) Methodological observation 1 epistemic-yield argument (strong typing pressures F3 to consume the field explicitly).

## Open Questions

This RFC IS opening an Open Modeling Question; substantive open questions are deferred to evidence file Phase 2 candidate enumeration.

Pre-framing open questions resolved at framing:

- **Does this OMQ require entity-model.md modification?** Yes; OMQ registered in §Open Modeling Questions by this PR.
- **Does this OMQ require Charter amendment?** Possibly per resolution candidate (see Q4 above); not pre-empted at framing.
- **Does this OMQ require synthetic + honeypot collector adapters to land first?** No; the OMQ's structural shape is independent of adapter count.

## Anti-Patterns to Avoid

- **Conflating authentication-class with truth-bearing fields.** Authentication-class is structural (which channel verified at observation time) NOT truth-bearing (whether the actor is real). Resolution must preserve §3 N1 boundary; encoding authentication-class as a confidence-like dimension on Cat I would violate.
- **Pre-empting resolution at framing.** §0144 Methodological observation 2 (anticipate ≠ pre-resolve) applies: the structural slot may be reserved (per Candidate δ's paired-enrichment example), but the specific shape choice waits for evidence-file deliberation.
- **Composing candidates without enumerating them separately first.** Candidate ζ (composite) is enumerated as a distinct path because its trade-offs differ from naive sum of α + δ; substantive deliberation must compare like-vs-like before considering composites.

## Migration and Backward Compatibility

Migration considerations are candidate-specific:

- Candidates α, β, ζ require schemas-evolution event at resolution + corpus regeneration + golden file refresh per canonical-serialization-contract.
- Candidate γ, ε do NOT require schemas-evolution event (operator convention only); migration is documentation.
- Candidate δ requires schemas-evolution event for the new `AuthenticationContext` record + adapter migration to commit the paired record.

Backward compatibility: the three already-landed F1 modalities (Network, Behavioral, Attestation) and the planned BrowserObservation must all support the chosen surface. Existing substrate records committed before resolution carry no authentication-class annotation; resolution candidates must specify the migration semantics for pre-resolution records (typically: UNKNOWN class).

## References

- [`decision-log §0143`](../../charter/decision-log.md) — Domain Pack v0.1 anti-bot atlas framing PR; anticipated this OMQ with trigger condition "F1 has three modalities stable".
- [`decision-log §0144`](../../charter/decision-log.md) — F1.NetworkObservation discriminated-union typing; first F1 modality landing.
- [`decision-log §0146`](../../charter/decision-log.md) — F1.BehavioralObservation; second F1 modality.
- [`decision-log §0147`](../../charter/decision-log.md) — F1.AttestationObservation; third F1 modality; OMQ TRIGGER FIRED.
- [`decision-log §0023`](../../charter/decision-log.md) — Q2 Identity tiers resolution: inception-phase single-tier `actor_ref` adopted; multi-tier deferred to ordinary Ontology RFC discipline (this RFC is parallel structural revision at Cat I envelope layer).
- [`docs/ontology/entity-model.md`](../../ontology/entity-model.md) — §Category I subject of revision at resolution.
- [`docs/architecture/canonical-serialization-contract.md`](../../architecture/canonical-serialization-contract.md) — schemas-evolution event discipline at resolution.
- [`docs/rfcs/discussion/authentication-class-typing-evidence.md`](../discussion/authentication-class-typing-evidence.md) — evidence scratch for substantive deliberation Phase 2–5.
- [`docs/rfcs/draft/ontology-revision-q3-independence.md`](./ontology-revision-q3-independence.md) — ontology-revision template precedent (Q3 formal independence; same shape).

## Decision Record

This RFC opens at framing-PR phase. Substantive deliberation (Phase 2: candidate enumeration; Phase 3–5: candidate analysis + synthesis + recommendation) proceeds in the evidence file. Resolution will be recorded in a subsequent decision-log entry once substantive phase closes; resolution will simultaneously:

1. Close the OMQ in [`entity-model.md` §Open Modeling Questions](../../ontology/entity-model.md#open-modeling-questions) (move from open to resolved).
2. Revise [`entity-model.md` §Category I](../../ontology/entity-model.md) to carry the chosen surface.
3. Schemas-evolution event per the chosen candidate's migration semantics.
4. Schemas changes to F1 envelopes (Network + Behavioral + Attestation + Browser) per the chosen surface.
5. Charter amendment if the chosen candidate requires one (assessed at resolution).
