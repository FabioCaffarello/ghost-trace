# Authentication-class typing — Ontology-revision discussion evidence

**Status:** in-discussion. Not authoritative. Final resolution will be recorded in a decision-log entry that simultaneously closes the OMQ in [`entity-model.md` §Open Modeling Questions](../../ontology/entity-model.md#open-modeling-questions) and revises §Category I to carry the chosen surface.

This scratch supports the discussion phase of [`ontology-revision-authentication-class-typing`](../draft/ontology-revision-authentication-class-typing.md). The RFC opened in response to F4 OMQ trigger firing at [`§0147`](../../charter/decision-log.md) (F1 reached 3/3 modalities stable: Network + Behavioral + Attestation).

Phase 1 names scope + dependency surface + candidate space; Phase 2 enumerates each candidate with per-axis trade-offs; Phases 3+ apply epistemic-skill review + synthesis + recommendation.

---

## Phase 1 — Scope, dependencies, posture

### The question

Per [`§0147`](../../charter/decision-log.md) OMQ-trigger event:

> Three F1 observation classes are empirically forced: server-authenticated (NetworkObservation gateway-witnessed), client-attested (AttestationObservation with cryptographic chain), client-witnessed (BehavioralObservation producer-attested without cryptographic anchor). The substrate's envelope does NOT structurally distinguish these classes. What structural surface should the substrate carry to make authentication-class explicit at the Cat I observation layer?

### Posture statement

The OMQ resolution must respect:

- **[§3 N1](../../charter/constitutional-charter.md#3-non-goals) — no truth at substrate.** Authentication-class is structural (which channel verified the observation) NOT truth-bearing (whether the actor is real). The resolution chooses an axis that records HOW verification happened, not WHETHER the actor is genuine. Encoding authentication-class as confidence-like dimension on Cat I would violate §3 N1.
- **[§2.1](../../charter/constitutional-charter.md#21-observational-integrity) substrate immutability.** Whatever surface carries authentication-class inherits immutability per Cat I — once committed, the class is not revised post-commit.
- **[`§0144`](../../charter/decision-log.md) Methodological observation 1 — strong typing epistemic-yield.** The resolution should pressure F3 to consume the field explicitly; operator-convention-only resolutions (Candidates γ, ε) have lower epistemic yield per the comprovar posture cravada in [`§0143`](../../charter/decision-log.md).
- **[`§0023`](../../charter/decision-log.md) Q2 Identity tiers inception-phase posture.** Defer multi-tier complexity when single-tier is structurally sufficient; document reversal conditions.
- **Backward compatibility with 3 already-landed F1 modalities** (Network at [`§0144`](../../charter/decision-log.md), Behavioral at [`§0146`](../../charter/decision-log.md), Attestation at [`§0147`](../../charter/decision-log.md)). Existing records carry no authentication-class annotation; resolution must specify migration semantics (typically: UNKNOWN class for pre-resolution records).
- **Forward compatibility with planned BrowserObservation** (F1's fourth modality). Per §0147 Methodological observation 3: discriminated-union pattern cascades; BrowserObservation will follow same envelope shape and inherit whatever authentication-class surface this RFC resolves.

### Dependency surface

**Discharged** (no further work required to proceed):

- F1 has 3/3 modalities stable per [`§0144`](../../charter/decision-log.md) + [`§0146`](../../charter/decision-log.md) + [`§0147`](../../charter/decision-log.md). Trigger condition satisfied.
- entity-model.md §Open Modeling Questions section exists with `(None remaining.)` marker; this RFC adds the F4 entry as the first new open OMQ since §0134.
- canonical-serialization-contract schemas-evolution event discipline established per [`§0136`](../../charter/decision-log.md) + [`§0139`](../../charter/decision-log.md) + [`§0140`](../../charter/decision-log.md). Resolution candidates that require proto-definition change inherit this discipline.

**Open** (this RFC's substantive deliberation):

- Choice among six candidate resolution forms (α, β, γ, δ, ε, ζ) per Phase 2 below.
- Per-candidate analysis of Charter-impact, schemas-impact, F3-consumer-shape, retrofit-cost.

**Out-of-scope for this RFC** (deferred to follow-on entries):

- BrowserObservation landing — orthogonal to this OMQ; can land before or after resolution (per [§0147 Methodological observation 1](../../charter/decision-log.md), resolution-before-landing is preferred to avoid retrofit but not strictly required).
- F3 signature engines consuming authentication-class — operational specification at service tier; opens after resolution.
- Multi-tier identity formalization (Q2 deferred per [`§0023`](../../charter/decision-log.md)) — orthogonal; multi-tier may compose with authentication-class at future revision.

### Six candidate resolution forms

Enumerated for Phase 2 substantive deliberation. Each gets its own analysis in subsequent phases.

| Candidate | Form | Proto change? | Operator effort | F3 consumer simplicity |
|---|---|---|---|---|
| α | Envelope enum field on each F1 envelope | Yes (4 protos) | Producer fills enum at observation time | High (read enum) |
| β | New Cat I subtype hierarchy (ServerAuthenticated / ClientAttested / ClientWitnessed) | Yes (entity-model + 4 protos) | Producer chooses subtype at construction | High (subtype discriminator) |
| γ | `actor_ref` prefix convention (e.g., `sa:actor-x`) | No | Operator-discipline string format | Medium (parse prefix) |
| δ | Paired enrichment record (`AuthenticationContext`) parallel to `IngestionEvent` | Yes (new proto) | Adapter commits paired record | Medium (resolve pairing) |
| ε | `collector_ref` semantic discipline | No | Operator-discipline naming | Low (parse collector_ref string) |
| ζ | Composite α + δ (envelope enum + paired record) | Yes (4 protos + new proto) | Both producer enum + paired record | Higher than α alone (carries verification anchor) |

---

## Phase 2 — Per-candidate × per-axis analysis

Six axes per candidate: Charter-impact, proto-definition cost, F3 consumer shape, retrofit cost, §3 N1 compatibility, backward/forward compatibility.

### Candidate α — Envelope enum field

**Form.** Add `authentication_class` proto3 enum field to each F1 envelope. Values: `UNKNOWN = 0`, `SERVER_AUTHENTICATED = 1`, `CLIENT_ATTESTED = 2`, `CLIENT_WITNESSED = 3`. Producer fills at observation construction time; substrate stores at commit time.

| Axis | Analysis |
|---|---|
| Charter-impact | None at framing; no Charter prose modified. Field is structural property of Cat I observation; inherits §2.1 immutability without amendment. §3 N1 satisfied: enum values name verification channel, not truth claim. |
| Proto-definition cost | 4 proto changes (NetworkObservation, BehavioralObservation, AttestationObservation, BrowserObservation). Schemas-evolution event per canonical-serialization-contract per [`§0139`](../../charter/decision-log.md) + [`§0140`](../../charter/decision-log.md). Corpus regeneration + 18+ existing golden files refreshed (Network 6 + Behavioral 5 + Attestation 6 + new). |
| F3 consumer shape | Read enum directly from envelope; pattern-match on value; weight inferences accordingly. **Simplest possible consumer surface** — one field read per observation. |
| Retrofit cost | 3 landed F1 modalities + planned BrowserObservation must support the field. Existing records (pre-resolution) carry no value; proto3 enum default = `UNKNOWN = 0`, which is structurally honest (pre-resolution records cannot retroactively claim a class). Migration is trivial: re-marshal records with new proto containing the field defaults to UNKNOWN automatically. |
| §3 N1 compatibility | High. Enum values name HOW verification happened (channel), not WHETHER the actor is real (truth). |
| Back/forward compat | Backward: existing records compatible (field defaults to UNKNOWN). Forward: future F1 modalities (e.g., BiometricObservation) inherit pattern by declaring the field. **Enum extensibility:** new values can be added at the tail (e.g., `MULTI_FACTOR_AUTHENTICATED = 4`) without breaking existing readers per proto3 enum extension semantics. |

**Net:** simplest structural form. High epistemic yield (F3 cannot consume the observation without seeing the field). Inception-phase aligned per [`§0023`](../../charter/decision-log.md) Q2 (single-tier; complex variants deferred).

### Candidate β — New Cat I subtype hierarchy

**Form.** Introduce three abstract Cat I subtypes: `ServerAuthenticatedObservation`, `ClientAttestedObservation`, `ClientWitnessedObservation`. F1 modalities extend the appropriate subtype based on producer-side context. NetworkObservation extends `ServerAuthenticatedObservation` (when collected at gateway) OR could extend others (when collected client-side); BehavioralObservation extends `ClientWitnessedObservation`; AttestationObservation extends `ClientAttestedObservation`; BrowserObservation extends `ClientWitnessedObservation`.

| Axis | Analysis |
|---|---|
| Charter-impact | Moderate. entity-model.md §Category I requires substantial revision: introduces a new abstract-subtype-hierarchy layer between Cat I and concrete modalities. Charter §2.3 BC4 (Identity tiers resolved per [`§0023`](../../charter/decision-log.md)) may interact: subtype hierarchy could be read as parallel to "Identity tiers" (currently inception-single-tier). Possible Charter amendment if the subtype hierarchy is read as introducing a new tier. **Constitutional review depth required: high.** |
| Proto-definition cost | Higher than α. Proto3 does not support inheritance natively; subtype representation would need a discriminator + nested fields OR separate top-level messages with shared envelope structure. Significant proto redesign across all 4 F1 modalities. Schemas-evolution event with corpus changes. |
| F3 consumer shape | Subtype discriminator inspection. Requires consumer to know type taxonomy. Multiple message types to handle. |
| Retrofit cost | High. Existing landed F1 records must be migrated to subtypes; either through proto-version-aware reader OR breaking change requiring corpus regeneration with new subtype discriminator. **Hardest of the six candidates to retrofit cleanly.** |
| §3 N1 compatibility | Same as α — subtypes name channel, not truth. |
| Back/forward compat | Backward: breaking unless careful versioning. Forward: new subtypes can be added but require careful coordination with existing subtype graph. |

**Net:** structurally richer than α but at significantly higher Charter + proto + retrofit cost. Constitutional-review burden high. **Not inception-phase aligned per [`§0023`](../../charter/decision-log.md) Q2 deferral discipline** (multi-tier complexity when single-tier suffices).

### Candidate γ — `actor_ref` prefix convention

**Form.** Producers encode authentication-class in `actor_ref` string via prefix convention: `sa:actor-x` (server-authenticated), `ca:actor-x` (client-attested), `cw:actor-x` (client-witnessed). No proto-definition change. Discipline lives in operator + producer documentation.

| Axis | Analysis |
|---|---|
| Charter-impact | None directly. But subverts [`§0023`](../../charter/decision-log.md) Q2 inception-phase single-tier `actor_ref` posture by overloading the field with two distinct semantics (actor identity + authentication-class). |
| Proto-definition cost | Zero. |
| F3 consumer shape | Parse prefix from string. Stringly-typed; no compile-time guarantee. F3 must implement prefix-parser per modality. |
| Retrofit cost | Zero on proto side. Existing records (with no prefix) default to "no class" via convention; F3 consumer must handle absent-prefix case. |
| §3 N1 compatibility | Same as α — prefix names channel, not truth. |
| Back/forward compat | Backward: existing records have no prefix; convention adds it forward. Forward: any new authentication-class added by extending prefix vocabulary. |

**Net: low epistemic yield per [`§0144`](../../charter/decision-log.md) Methodological observation 1.** Strong-typing pressure on F3 is absent; F3 may silently consume `actor_ref` ignoring the prefix. Operator-discipline-only candidates are explicitly downweighted in the posture statement. **Rejected for the comprovar epistemic-yield argument; documented for completeness.**

### Candidate δ — Paired enrichment record (`AuthenticationContext`)

**Form.** New Cat I record `AuthenticationContext` committed atomically alongside each F1 observation, parallel to `IngestionEvent` paired-by-reference shape per [`§0038`](../../charter/decision-log.md). Carries authentication-class + optional verification anchor (e.g., a hash of the attestation chain when applicable).

| Axis | Analysis |
|---|---|
| Charter-impact | Low. IngestionEvent precedent is established. New Cat I record inherits §2.1 immutability + §2.3 typed reference. The `AuthenticationContext` field structure parallels existing patterns. |
| Proto-definition cost | One new proto file (`authentication_context.proto`). Schemas-evolution event. Existing F1 protos unchanged (the pairing happens at substrate commit time, not in F1 proto fields). |
| F3 consumer shape | Resolve paired record via substrate query (hash → AuthenticationContext record). One indirection cost per observation. More plumbing than α. |
| Retrofit cost | Modest. Existing landed F1 records have no paired AuthenticationContext; F3 consumer handles absent-paired-record case (defaults to UNKNOWN class). New adapter writes commit the paired record going forward. |
| §3 N1 compatibility | Same as α — record names channel + (optional) verification anchor, not truth. |
| Back/forward compat | Backward: pre-resolution records lack paired AuthenticationContext; consumer handles absence. Forward: AuthenticationContext can evolve independently of F1 envelopes (new fields, new authentication-class enum values). **Strongest forward-compat of the six** — F1 envelopes don't churn; only the paired record does. |

**Net:** structurally rich, IngestionEvent-precedent-aligned, but with consumer-side indirection cost. Strong forward-compat. Loses the immediate-field-read simplicity of α.

### Candidate ε — `collector_ref` semantic discipline

**Form.** Standardize `collector_ref` naming convention to encode authentication-class: `server-collector:...`, `attested-collector:...`, `witnessed-collector:...`. No proto-definition change. Discipline lives in operator + adapter documentation.

| Axis | Analysis |
|---|---|
| Charter-impact | None directly. But conflicts with [`§0144`](../../charter/decision-log.md) (e) phenomenon-vs-record reconciliation OMQ surface — overloading `collector_ref` with two distinct semantics (adapter identity + authentication-class). |
| Proto-definition cost | Zero. |
| F3 consumer shape | Parse collector_ref prefix. Lowest epistemic-yield form of all six. |
| Retrofit cost | Zero on proto side. |
| §3 N1 compatibility | Same as α — prefix names channel. |
| Back/forward compat | Backward: existing collector_refs without prefix default to UNKNOWN. Forward: convention-based; depends on adapter compliance. |

**Net: lowest epistemic yield + conflicts with §0144 (e) OMQ surface.** Rejected for the same reasons as γ + additional conflict with phenomenon-vs-record OMQ slot. **Rejected; documented for completeness.**

### Candidate ζ — Composite α + δ

**Form.** Combine α (envelope enum field) with δ (paired AuthenticationContext record). Envelope enum provides fast lookup; paired record provides verification anchor when applicable.

| Axis | Analysis |
|---|---|
| Charter-impact | Same as α + δ combined; both are Charter-light. |
| Proto-definition cost | Higher than either alone: 4 F1 envelope changes + 1 new paired-record proto. Schemas-evolution event with broader scope. |
| F3 consumer shape | Two paths: read enum from envelope for quick filtering; resolve paired record for verification-anchor inspection when needed. **Highest expressivity of the six**, but at consumer-complexity cost. |
| Retrofit cost | α retrofit + δ retrofit (handled separately as in α + δ individual analyses). |
| §3 N1 compatibility | Same as α + δ. |
| Back/forward compat | Best of both worlds: envelope-enum-only consumers work, paired-record-aware consumers get richer surface. |

**Net:** maximum expressivity. Maximum proto + corpus cost. Consumer-complexity higher than necessary at inception phase.

---

## Phase 3 — Epistemic-skill review

Per [`§0141`](../../charter/decision-log.md) operational-spec deliberation pattern (lighter Phase 3 than constitutional ontology-revision precedents). Apply [`falsifiability-check`](../../../.claude/skills/epistemic/falsifiability-check/SKILL.md) + [`ambiguity-reducer`](../../../.claude/skills/epistemic/ambiguity-reducer/SKILL.md) + [`vocabulary-discipline`](../../../.claude/skills/ontology/vocabulary-discipline/SKILL.md) + [`anti-marketing`](../../../.claude/skills/enforcement/anti-marketing/SKILL.md) to the surviving candidates (α, β, δ, ζ — γ + ε rejected at Phase 2).

### Falsifiability-check per candidate

| Candidate | Falsifiable predicate | Operationalization |
|---|---|---|
| α | "Every F1 envelope record carries one of 4 enum values" | Substrate replay; enum field present and well-typed per proto definition; mechanical check |
| β | "Every F1 record extends exactly one of 3 abstract subtypes" | Substrate replay; subtype discriminator present; type taxonomy enforceable at proto-load time |
| δ | "Every committed F1 record has a paired AuthenticationContext record OR a `no-context` sentinel" | Substrate replay; PRIMARY-KEY-pairing checkable; pairing index reachable |
| ζ | α + δ predicates conjoined | Both substrate replays |

**All four surviving candidates are falsifiable.** Distinction is structural, not falsifiability-relative.

### Ambiguity-reducer per candidate

| Candidate | Ambiguity surface | Mitigation |
|---|---|---|
| α | "UNKNOWN" value — what does it mean operationally? Producer didn't fill OR producer chose UNKNOWN explicitly? | Document: UNKNOWN is default for pre-resolution records + for cases where producer cannot determine class. Two operational meanings collapsed into one value. |
| β | Subtype assignment criteria — when does NetworkObservation extend ServerAuthenticated vs ClientAttested? | Document per-modality table. Some F1 modalities can structurally extend multiple subtypes depending on collection context. Ambiguity higher than α. |
| δ | Pairing semantics — is paired record required or optional? | Per IngestionEvent precedent: required at commit (atomic AppendPair). Pre-resolution records: paired record absent (treated as UNKNOWN class). |
| ζ | Same as α + δ combined. Slight redundancy: envelope enum + paired record's class field. Reader must rely on one OR document precedence rule. | Document: envelope enum is canonical for quick-class-check; paired record carries fuller verification anchor; on disagreement, paired record is canonical (it represents the substrate's recorded verification context). |

**Lowest ambiguity surface: α.** β has multi-extension ambiguity per modality. ζ has redundancy + precedence ambiguity.

### Vocabulary-discipline review

All four surviving candidates introduce the same canonical terms: `authentication-class`, `server-authenticated`, `client-attested`, `client-witnessed`. No vocabulary divergence per candidate.

### Anti-marketing review

All four surviving candidates use structural-claim language; no marketing tells in proposed enum values / subtype names / field labels. **Notable risk:** any future expansion of `authentication-class` enum that introduces aspirational-noun-shaped value names (per the [`anti-marketing`](../../../.claude/skills/enforcement/anti-marketing/SKILL.md) watchlist) would breach watchlist discipline. Future enum expansion must remain on structural-channel-naming axis (e.g., `MULTI_FACTOR_AUTHENTICATED`, `HARDWARE_ATTESTED`).

---

## Phase 4 — Synthesis

### Finding 1 — γ + ε rejected at posture-axis (epistemic yield)

The [`§0144`](../../charter/decision-log.md) Methodological observation 1 (strong-typing epistemic-yield argument) is explicit in the posture statement: operator-convention-only candidates have lower yield per the "comprovar" posture. γ + ε survive proto cost analysis trivially (no change) but fail the posture axis. **Both rejected at Phase 2; not considered in Phase 3+.**

### Finding 2 — β rejected at Charter-cost + retrofit axes

β introduces a new abstract-subtype-hierarchy layer between Cat I and concrete F1 modalities. Constitutional review depth high (potential Charter amendment if read as parallel to Q2 identity tiers per [`§0023`](../../charter/decision-log.md)). Retrofit cost across 3 already-landed F1 modalities is the largest of the six. **Rejected as not inception-phase aligned** per [`§0023`](../../charter/decision-log.md) Q2 deferral discipline.

### Finding 3 — α survives all axes with smallest cost surface

α requires 4 proto changes (schemas-evolution event with corpus regeneration) but introduces no Charter complexity, satisfies §3 N1, supports forward-compat via proto3 enum extension, is lowest-ambiguity of survivors, and provides F3 with immediate field read (highest epistemic yield among falsifiable candidates).

### Finding 4 — δ is α's structural alternative with stronger forward-compat

δ has a strong forward-compat property (F1 envelopes don't churn; only the paired record does). At the cost of consumer-side indirection per observation. Useful if F3 consumers must inspect verification anchors heavily; less useful if they only need the class label.

### Finding 5 — ζ is α + δ with maximum expressivity but redundancy ambiguity

ζ combines α + δ. Pays both costs. Introduces precedence-rule ambiguity (envelope enum vs paired record class). **Justified only if both axes (quick lookup + rich verification anchor) are simultaneously required.**

### Finding 6 — Inception-phase posture favors α; reversal conditions defined

Per [`§0023`](../../charter/decision-log.md) Q2 + [`§0141`](../../charter/decision-log.md) F2 conservative-defaults bundle precedent, the inception-phase choice is the simpler form with reversal conditions documented. α is the simpler form. Reversal conditions to revisit choice:

- **Reversal to ζ:** when F3 consumers empirically need verification anchors (paired record carries hash of attestation chain, redirects, etc.) AND envelope-enum-alone is insufficient.
- **Reversal to β:** when authentication-class is structurally cross-cutting enough that subtype-discriminator pattern enforces it more cleanly than envelope field (e.g., if F3 starts having "per-class" code paths that grow large).
- **Reversal to δ-only:** when envelope enum proves insufficient AND envelope churn cost exceeds paired-record indirection cost.

---

## Phase 5 — Recommendation

**Candidate α — Envelope enum field.**

Per Finding 3 (α survives all axes with smallest cost surface) + Finding 6 (inception-phase posture aligned with explicit reversal conditions). The recommendation is for committee approval; resolution will:

1. Add `authentication_class` proto3 enum to all 4 F1 envelopes (NetworkObservation, BehavioralObservation, AttestationObservation, BrowserObservation).
2. Schemas-evolution event per canonical-serialization-contract.
3. entity-model.md §Category I revised: §Open Modeling Questions F4 entry moves to §Resolved Modeling Questions with reference to the resolution decision-log entry.
4. Corpus regeneration + golden file refresh across the 18+ affected fixtures.
5. Reversal conditions documented in resolution entry per [`§0023`](../../charter/decision-log.md) + [`§0141`](../../charter/decision-log.md) pattern.

**Adoption is contingent on committee approval at resolution decision-log entry; this evidence file's Phase 5 is recommendation only.**

### Recommended enum specification

```proto
enum AuthenticationClass {
  AUTHENTICATION_CLASS_UNKNOWN = 0;
  AUTHENTICATION_CLASS_SERVER_AUTHENTICATED = 1;
  AUTHENTICATION_CLASS_CLIENT_ATTESTED = 2;
  AUTHENTICATION_CLASS_CLIENT_WITNESSED = 3;
}
```

Per proto3 enum conventions: prefix-namespaced to avoid collision; 0 reserved for default (pre-resolution records + producer-cannot-determine).

### Recommended envelope field placement

Add `AuthenticationClass authentication_class = N` to each F1 envelope at the next available field number after `collector_ref` (typically position 5 in current envelopes, with `oneof modality` shifting up by one field number — backward-incompatible at the binary-encoding level but acceptable since no existing consumers depend on the binary format of unfilled records).

**Alternative field placement (recommended):** add at the END of envelope field numbers (position 10+, after the oneof) to avoid shifting `oneof modality` field numbers. Preserves binary backward compat for existing records.

### Recommended reversal conditions (per [`§0023`](../../charter/decision-log.md) + [`§0141`](../../charter/decision-log.md) precedent)

- **R-α-1: F3 verification-anchor pressure.** If F3 signature engines empirically need to inspect attestation chains (not just the class label), revisit ζ adoption (composite α + δ).
- **R-α-2: Per-class F3 code-path growth.** If F3 develops large per-class code paths (>20% per-modality LOC), revisit β adoption (subtype hierarchy).
- **R-α-3: Envelope churn exceeds paired-record cost.** If subsequent revisions force multiple envelope changes affecting authentication-class semantics, revisit δ adoption (paired record).
- **R-α-4: Enum value extensibility limit.** If multi-factor or hardware-attested classes emerge AND require structural carriage beyond simple enum extension, revisit β or ζ.

---

## Resolution path (post-recommendation)

After committee approval:

1. Separate PR adds `authentication_class` enum + envelope field to 4 F1 protos.
2. Decision-log entry records resolution (moves OMQ from open to resolved in entity-model.md).
3. Schemas-evolution event per canonical-serialization-contract.
4. Corpus regeneration + golden file refresh.
5. F3 signature engines may now consume `authentication_class` field; downstream F3 work is unblocked.

