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

### Phase 2 sketch (to be drafted in subsequent commit)

Per candidate: enumerate Charter-impact + schemas-impact + F3-consumer-shape + retrofit-cost + § (3 N1) compatibility + backward/forward compatibility. Compare against the posture statement axes above.

---

## Phase 3+ (placeholder)

Pending Phase 2 completion.
