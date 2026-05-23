# RFC — Domain Pack v0.1: Anti-Bot Atlas (framing PR)

- **Status:** discussion (framing PR; substantive phase pending evidence-file Phase 2–5 deliberation; resolution pending decision-log entry)
- **Authors:** Ghost Trace committee (opened post-Layer B operational arc closure at [`§0141`](../../charter/decision-log.md); post-Charter v0.7.2 stale-anchor sweep at [`§0142`](../../charter/decision-log.md))
- **Date:** 2026-05-23 (opened)
- **Type:** domain-pack (new RFC category — first domain pack applied on the substrate; structure hybrid of `ontology-revision` constitutional framing + `operational-spec` executable criterion)
- **Affects:** This RFC opens a new category of work — domain-specific applied packs on the Charter-frozen substrate. Specific surfaces touched by accepting this framing:
  - [`schemas/events/v1/`](../../../schemas/events/v1/) — four new Cat I observation proto types (NetworkObservation, BehavioralObservation, AttestationObservation, BrowserObservation) trigger schemas-evolution events per the canonical-serialization-contract per [`§0139`](../../charter/decision-log.md) + [`§0140`](../../charter/decision-log.md) discipline.
  - [`services/ingestion/internal/`](../../../services/ingestion/internal/) — new packages for signature engines (`signatures/`) and adapter pipelines (`adapters/`) under conservative-defaults bundle precedent established by Layer B at [`§0141`](../../charter/decision-log.md) (internal-package + on-the-fly evaluation pattern).
  - [`docs/ontology/entity-model.md`](../../ontology/entity-model.md) §Open Modeling Questions — two new OMQs proposed for registration via separate ontology-revision RFCs anticipated to surface under F1+F3 pressure: phenomenon-vs-record reconciliation under multi-source ingestion; authentication-class typing (server-authenticated vs client-attested vs client-witnessed).
  - [`docs/architecture/`](../../architecture/) — new architecture artifact: F6 substrate-read contract (parallel to canonical-serialization-contract; read-semantics + reproducibility guarantees).
  - No Charter prose modification. No new Charter invariant. No structural amendment proposed. RFC consumes Charter v0.7.2 frozen state as given.

> RFCs in Ghost Trace are proposals subject to constitutional review. The [Constitutional Charter](../../charter/constitutional-charter.md) is the authoritative reference. RFCs that conflict with the Charter must either be rejected or accompanied by a corresponding charter amendment (see [`../../charter/amendments.md`](../../charter/amendments.md)). **This RFC is a domain pack** — the first applied vertical on the substrate. It is hybrid in form: constitutional framing per the `ontology-revision-*` precedent (postura + tese + Q1–Q6 review) + executable criterion per the `operational-spec-*` precedent (sub-benchmarks + falsificability discipline + milestone ordering).

## Status note — all prior preconditions discharged

This RFC could not open before today. The preconditions are:

- **Charter fully frozen.** Amendment v0.7 closed §3 at [`§0131`](../../charter/decision-log.md). All ten Charter sections frozen (Thesis, §2 header, §2.1–§2.6, §3, §4). The substrate is no longer evolving under amendment pressure that could invalidate a domain pack's structural assumptions mid-flight. ✓
- **Implementation gate cleared.** Per [CLAUDE.md §6.4](../../../.claude/CLAUDE.md), the implementation gate cleared at [`§0022`](../../charter/decision-log.md) with three structural criteria satisfied. This RFC proceeds under ordinary RFC discipline, no re-clearing needed. ✓
- **Layer A + Layer B operational arc complete.** [`§0141`](../../charter/decision-log.md) closed the nine-decision Q4 → Layer B arc (§0011 → §0099 → §0129 → §0133 → §0134 → §0135 → §0136 → §0138 → §0141) through the service-tier implementation layer. PRs #158–#161 wired Layer B through demote-* + promote-* CLIs. The hypothesis lifecycle mechanics are live; the inference layer that produces formation events from observations is the next gap. ✓
- **Stale anchor sweep complete.** Patch amendment v0.7.2 at [`§0142`](../../charter/decision-log.md) cleared seven Q-pending cross-references in Charter prose. §2.4 BC3 (substrate-time generation timing), §2.4 BC6 (transitive scope via §0134-τ), §2.6 BC1 (formal independence via §0133-α), §2.6 BC2 (substrate-time generation of EI values), §2.6 BC6 (transitive scope of EI semantics) — all referenced citably post-patch without stale-anchor drift propagating into this RFC. ✓

The four predecessor disciplines are aligned. This RFC opens the work that consumes them.

## Summary

The substrate is structurally complete and operationally exercised on four toy patterns (`uniform_cadence_v1`, `co_occurrence_window_v1`, `session_descriptor_shared_v1`, `temporal_descriptor_cohort_v1`) operating exclusively on `DeclaredSession`. The hypothesis lifecycle — formation, promotion, demotion, dissolution, merge, split, with Layer A + Layer B candidacy criteria — runs end-to-end against this toy surface. **What is missing is the inference layer**: the signature engines that detect patterns in observations and produce formation events, the typed observation schemas they consume, and the operator interface that exposes the substrate's structural honesty as audit-grade explainability.

This RFC opens a domain pack — the first applied vertical — that fills these gaps for anti-bot specifically, under a six-month window with a falsifiable comprovation criterion. The thesis: the substrate, as frozen in Charter v0.7, is the only known architecture that structurally prevents recursive belief inflation via the paired-dimension commitment of §2.4 + §2.6. This pack stress-tests that property by building the minimum domain atlas (typed observations + inference engines + operator-grade explainability) under realistic adversarial pressure, and recording every constitutional pressure surfaced into decision-log entries per the §0022 empirical-pressure methodology.

The RFC does not pick implementation choices in framing. Sub-decisions are enumerated for substantive deliberation in [`domain-pack-v0-1-evidence.md`](../discussion/domain-pack-v0-1-evidence.md) Phase 2.

## Motivation

### Why now

Three orthogonal preconditions landed within the past three weeks:

- **Constitutional ground.** Charter v0.7 frozen + v0.7.2 stale-anchor sweep means every BC + structural claim this pack will cite is current.
- **Operational mechanics.** Layer B at §0141 brought the staged-combination demotion criterion from form (§0135) + parameters (§0138) into live service-tier code (PRs #158–#161). The substrate now has both candidacy gates — Layer A advisory + Layer B advisory — wired through CLIs. The mechanics are ready to receive real signatures.
- **Postura strategic resolution.** The chat-Claude + Claude-Code triangulation closed posture-question deliberation at: *comprovar puro, six months free of revenue pressure, three-source-parallel ingestion strategy, three-layer signature openness (A/B/C all open in main repo)*. The strategic axes are decided; what remains is execution discipline.

Per [`§0011`](../../charter/decision-log.md) Methodological Observation 1 (form-vs-parameter-vs-implementation tripartite extended through Q4 → Layer B arc), domain packs are the natural fourth layer above implementation: form (Charter + Ontology) + parameters (operational-spec) + implementation (services) + **applied domain** (this RFC). The tripartite that closed Layer B now extends to a four-layer methodology.

### Cost of not resolving

Implementation work consuming the substrate has three options absent this framing:

- **(a) Build signatures inline without framing.** Each signature becomes a discretionary commit choice; instrumentação per subtype × fonte × morfologia-de-chain is decided post-hoc; comprovation criterion is implicit. Violates §0022 empirical-pressure discipline (which demands deliberate capture of structural pressures, not retrospective reconstruction).
- **(b) Defer signature work entirely.** Substrate continues to exercise toy patterns; no real demotion ever fires; the §2.4 + §2.6 paired-dimension defense remains unstressed; the comprovation thesis remains untested. The six-month window is spent without evidentiary yield.
- **(c) Pivot to commercial work before comprovation.** Reverses the posture decision; introduces commercial-surface pressure that distorts F1–F6 priorities. Already considered and rejected in posture deliberation per D1 = 6 months free.

The framing PR is the lowest-ceremony way to make the comprovation contract explicit before execution begins.

### Downstream consequence

The hypothesis-lifecycle plumbing — particularly the demote-* CLIs that surface Layer A + Layer B verdicts (PRs #159–#161) — currently exercises against toy patterns. Once F3 signature engines produce real formation events, the same CLIs run against real hypotheses; the same Layer A + Layer B advisory verdicts compute against real evidence chains; the same instrumentation captures real demotion candidacies. **No new CLI is needed for the comprovation criterion**; the existing lifecycle CLI surface is the comprovation surface.

## Postura constitucional

This RFC opens the first applied domain pack on the Ghost Trace substrate. It assumes Charter v0.7.2 frozen as given. It proposes no Charter amendments. It anticipates that domain work will surface constitutional pressures that are captured as decision-log entries per the [`§0022`](../../charter/decision-log.md) empirical-pressure methodology; some of those entries may motivate subsequent Charter amendments or new Ontology surface, which would proceed under ordinary amendment / RFC discipline.

The domain pack is hybrid in structure:

- **Constitutional framing** per `ontology-revision-*` precedent — postura + tese + Q1–Q6 review + anticipated OMQs.
- **Executable criterion** per `operational-spec-*` precedent — falsifiable sub-benchmarks + instrumentation specification + milestone ordering + assumptions named explicitly.

The hybrid structure follows from this RFC's dual nature: it both opens structural pressure on the substrate (constitutional surface) and commits to a six-month execution plan (operational surface). Neither precedent alone covers the scope.

## Tese de comprovação

> The Ghost Trace substrate, as frozen in Charter v0.7, is the only known architecture that structurally prevents recursive belief inflation via the paired-dimension commitment of [§2.4](../../charter/constitutional-charter.md#24-inferential-influence-disclosure) (declared influence) + [§2.6](../../charter/constitutional-charter.md#26-evidential-independence-integrity) (evidential independence at substrate-write time). Every Assertion under inferential commitment carries both `confidence` and `evidential_independence` at commit per [`§0136`](../../charter/decision-log.md) canonical-serialization-contract; offline-only EI derivation is structurally forbidden per §2.6 Forbidden Anti-Patterns; supersession reads against §2.5 lifecycle events per [`§0020`](../../charter/decision-log.md) OMQ #2-C ensure demoted/dissolved hypotheses do not silently continue inflating downstream EI. No conventional anti-bot vendor — black-box ML, fingerprinting-as-service, attestation-only — has this property. The property is not an implementation detail; it is a constitutional invariant.

This RFC stress-tests the property by building the minimum domain atlas necessary to generate realistic adversarial pressure on the substrate, under a six-month window, with a falsifiable comprovation criterion that distinguishes substrate-survival from substrate-evasion.

## Critério de comprovação (falsificável)

Two sub-benchmarks gate the comprovation outcome. Both are mechanically observable via substrate replay; both produce non-ambiguous veredicts per [§4 criterion 1](../../charter/constitutional-charter.md#4-constitutional-design-rule) structural enforceability.

### Sub-benchmark 1 — Public + synthetic sources (month 3–4)

**Trigger:** F3 minimum viable inference produces first non-trivial demotion under datasets-públicos + synthetic-controlled pressure.

**Non-trivial** is defined as:

- **Non-tautological.** The signature that formed the hypothesis did not encode its own demotion criterion. Layer A advisory (cadence elapsed) + Layer B advisory (deep criterion fired) are read from substrate-grounded evidence, not from operator annotation at formation time.
- **Provenance reconstructible.** The hypothesis's `subject_ref_construct` / `subject_ref_hypothesis` chain (per [§0016](../../charter/decision-log.md) Q3 typed reference) resolves to Cat I primary observations through Cat II constructs without external state.
- **Layer A + Layer B substrate-grounded.** Layer A predicate evaluates per [`§0011`](../../charter/decision-log.md) staged-combination cadence gate; Layer B predicate evaluates per [`§0136`](../../charter/decision-log.md) `freshness_B(H) < T_B OR saturation_C(H) > K_C` against substrate-committed `evidential_independence` (per [`§0133`](../../charter/decision-log.md) Candidate α: source-count ratio over Cat I provenance roots) and transitive `influenced_by` closure (per [`§0134`](../../charter/decision-log.md) Candidate τ: β-graph storage).

### Sub-benchmark 2 — Honeypot source (month 5–6)

**Trigger:** F3 produces first non-trivial demotion under honeypot pressure (non-curated adversarial traffic).

Same non-triviality predicate as Sub-benchmark 1, but the source is non-curated by the team. This sub-benchmark has higher epistemic yield: it tests whether F3 + substrate handle phenomena not imagined by the team.

### Mandatory instrumentation

For every signature in F3, the substrate-grounded test harness must record, **per subtype × fonte × morfologia-de-chain**:

| Metric | Per subtype × fonte × morfologia-de-chain |
|---|---|
| Formation count | Number of formation events emitted by this signature for this subtype under this source class. |
| Promotion count | Number that advanced to promotion lifecycle event. |
| EI distribution at formation | Distribution of `evidential_independence` values committed at formation time per [`§0133`](../../charter/decision-log.md). |
| Demotion candidacy count | Number of times Layer A + Layer B candidacy predicate fired. |
| Committed demotion count | Number of operator-elected demotions that materialized as substrate events. |
| Chain morphology | `chain_depth_max` (longest `influenced_by` path) + `chain_breadth_at_root` (number of influencing hypotheses at the root). |

**Without this instrumentation, non-firing of benchmarks is non-informative.** The diagnostic distinction between (i) chains-fracas (depth ≤ 2 OR breadth-at-root ≤ 3) indicating F3 insufficient, and (ii) chains-fortes with EI stable indicating constitutional thesis confirmed (domain rotates evidence faster than decay) — requires the morphology dimension at instrumentation, not at analysis. Instrumentation is **marshalling-boundary constraint at build time**, not retroactive add-on; per the cross-Claude triangulation: instrumentation cost of design is trivial, instrumentation cost of retrofit is impossible.

### Falsifiability outcomes

| Outcome | Reading |
|---|---|
| Both sub-benchmarks pass with reconstructible provenance | Comprovation thesis confirmed under exercised conditions. Six-month yield: substrate-with-pedigree + decision-log entries documenting surfaced pressures. |
| Sub-benchmark 1 passes, Sub-benchmark 2 fails | F3 + substrate handle curated/imagined adversarial conditions but not non-curated. Reading: domain atlas insufficient for real-world deployment; substrate property unconfirmed for non-curated pressure. |
| Sub-benchmark 1 fails | F3 not minimum viable. Reading: signature engines are weaker than required to exercise the substrate. Pre-commit revision of F3 scope. |
| Both fail with chain-morphology weak (depth ≤ 2 OR breadth ≤ 3) | F3 not recursive enough. Reading: signature engines produce shallow chains; substrate's EI computation receives insufficient input to defend. |
| Both fail with chain-morphology strong, EI stable | Substrate property confirmed: domain rotates evidence faster than influence decay. Comprovation thesis **stronger** than originally framed — substrate defends preemptively. |
| Both fail with chain-morphology strong, EI decays but no demotion candidacy | Layer B parameter calibration ([`§0138`](../../charter/decision-log.md) T_B = K_C = 0.5) may be miscalibrated for this domain. Triggers parameter re-evaluation RFC per [`§0138`](../../charter/decision-log.md) per-parameter reversal-conditions surface. |

Every outcome is informative — the criterion is genuinely falsifiable per §4 criterion 1, not a checkbox.

## Escopo — F-codes from cross-Claude landscape mapping

Notation: **F1–F10** are the front codes established in the cross-Claude posture deliberation (10 fronts grouped into 3 maturity layers). This RFC's scope is **Camada 1 + F6 from Camada 2**; F5, F7–F10 are out-of-scope for this window.

### F1 — Typed observation taxonomy (Cat I extensions)

The current substrate accepts generic `NetworkEvent` with opaque `event_descriptor` bytes. Anti-bot requires semantic typing per modality. Four new Cat I observation proto types:

- **NetworkObservation** (month 1) — IP/ASN/geo, TLS JA4 fingerprint, HTTP/2 frame patterns, request timing distributions. Lowest friction modality (well-defined in literature; opaque NetworkEvent already gesticulates the surface).
- **BehavioralObservation** (month 2) — mouse trajectory, keyboard timing distribution, scroll cadence, dwell time, navigation graph. Required by any realistic signature beyond network-only.
- **AttestationObservation** (month 2–3) — Apple Private Access Tokens, Google Privacy Pass, Cloudflare Turnstile, WebAuthn / FIDO2 attestation. **Not simple per superficial reading**; requires discriminated union per protocol or generic envelope with semantic loss. Likely warrants a sub-RFC for the typing decision.
- **BrowserObservation** (month 2–3) — canvas/WebGL fingerprint, font enumeration, CDP automation markers, header order. May itself require discriminated union (the five sub-modalities are semantically distinct).

Each new Cat I observation proto triggers a schemas-evolution event per the canonical-serialization-contract per [`§0139`](../../charter/decision-log.md) (BLAKE3-hash-list discipline) + [`§0140`](../../charter/decision-log.md) (marshalling-boundary enforcement). Corpus regeneration + golden file refresh per existing discipline.

**Structural commitment:** each Cat I observation record carries `actor_ref` per [`§0023`](../../charter/decision-log.md) Q2 (inception-phase single-tier `actor_ref`) and a content-addressable identifier (BLAKE3) such that each record is independently hash-reconstructible without consulting external state. **Cat I observations do NOT carry `subject_ref_construct` / `subject_ref_hypothesis`** — those fields are §0016 Q3 typed reference edges on Assertions referencing Cat II / Cat III, per [§2.6 BC3](../../charter/constitutional-charter.md#26-evidential-independence-integrity) ("§2.6 governs the paired-dimension commitment for inferential-assertion records; not for observation records").

### F2 — Cat II construct library (extended sob pressão de F3)

The current library has four toy constructs operating on `DeclaredSession`. Anti-bot inference will require richer Cat II surface — likely `humanness_indicator`, `automation_signature`, `navigation_regularity`, `cohort_anomaly_distance`, `attestation_chain_trust`, `cross_modal_coherence`. **Not front-loaded.** Each construct formalizes when F3 needs it; the existing `uniform_cadence_v1.go` pattern (versioned Cat II definition) is the precedent.

Each new construct is itself an **operational definition** per [§2.2](../../charter/constitutional-charter.md#22-epistemic-separation) Q1 determinism; the versioning of operational definitions is one of the anticipated OMQs surfaced below.

### F3 — Signature engines (Cat III inference logic)

The largest gap. The substrate has Cat III lifecycle plumbing (formation, promotion, demotion, dissolution, merge, split events per [`§0010`](../../charter/decision-log.md) Q2-A.2 + [`§0011`](../../charter/decision-log.md) staged-combination) but no inference processes that produce formation events from observations + constructs.

**Subtype ordering** (revised from the cross-Claude initial framing per F1 dependency analysis):

- **AutomationGroup** (month 4–5) — canonical patterns are well-published (uniform cadence, automation framework markers). Starts when F1 has 2 stable modalities (Network + Behavioral). Camada A signatures + Camada B signatures.
- **BehavioralCluster** (month 5–6) — requires BehavioralObservation stable + cross-modal coherence Cat II construct. Camada B signatures primarily.
- **CoordinationRing + CampaignHypothesis** — when F1 has all 4 modalities (likely beyond month 6 window). Out-of-scope for this RFC's comprovation.

**Inference contract** for every signature:

- Formation event is a Cat III hypothesis-lifecycle Cat I record per [§2.5 BC at line ~40](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) (lifecycle events are Cat I).
- `influenced_by` chain declared at formation event time per [§2.4 BC3](../../charter/constitutional-charter.md#24-inferential-influence-disclosure) (substrate-time generation per [`§0021`](../../charter/decision-log.md) OMQ #3-α) when the signature reads from prior promoted hypotheses.
- `evidential_independence` computed at formation per [`§0133`](../../charter/decision-log.md) Candidate α (source-count ratio over Cat I provenance roots).
- Instrumentation per subtype × fonte × morfologia-de-chain wired from build time.
- Versioned definition format (parallel to `uniform_cadence_v1.go` precedent). Signature versioning surfaces the op-def-versioning OMQ.

### F4 — Authentication-class typing (OMQ-shaped surface)

Anti-bot-specific: not all observations carry the same authentication-class weight. Server-authenticated NetworkObservation (gateway-verified TLS handshake) is structurally different from client-attested AttestationObservation (browser-supplied attestation chain) is structurally different from client-witnessed BrowserObservation (JS-collected fingerprint, no attestation). The current `actor_ref` does not capture this dimension.

F4 opens as a formal OMQ surfaced under F1 pressure (likely month 4–5 when F1 has 3+ modalities forcing the distinction). **Proposed for separate ontology-revision RFC**; not resolved in this RFC. Likely candidates for the OMQ's structural surface: (a) new field on Cat I observations; (b) new Cat I subtype hierarchy; (c) discriminator on existing `actor_ref` semantics. Canonical-vocabulary note: the OMQ name `authentication-class typing` replaces the colloquial term often used in the anti-bot literature; the literature term is on the [`anti-marketing`](../../../.claude/skills/enforcement/anti-marketing/SKILL.md) watchlist and is replaced here under [`vocabulary-discipline`](../../../.claude/skills/ontology/vocabulary-discipline/SKILL.md) per [§4 of the discipline](../../../.claude/skills/ontology/vocabulary-discipline/SKILL.md) — same structural meaning, watchlist-compliant surface.

### F6 — Operator-grade explainability (architecture, not UX)

F6 is the **commercializable surface of the substrate's structural property**. Not a dashboard; an interface against contract. Audit-grade navigability by hash, statelessness over substrate, exportable to audit-grade format. The regulator must be able to reproduce, from F6's export, the exact provenance chain the operator saw.

**Designed alongside F3, not after.** F6's reading requirements (statelessness, hash-navigability, exportability) constrain F3's commit-time materialization. If F6 is treated as fim-de-fase product, F3's proto-definition choices may foreclose hash-navigability and no quantity of UI corrects it.

**New architecture artifact: F6 substrate-read contract.** Parallel to the canonical-serialization-contract. Defines:

- Read semantics against substrate at instant T₀.
- Reproducibility guarantee in T₀+Δ given no superseding §2.5 lifecycle events between T₀ and T₀+Δ.
- Export format for audit-grade navigation (hash-rooted provenance subgraph).
- Statelessness invariant (no F6-side cache that diverges from substrate truth).

F6 in this window is CLI-grade (no GUI). The substrate-read contract is the artifact that survives beyond CLI to any future GUI / API / regulator-facing surface.

### Out-of-scope for this window

- **F5** — Real-time decision serving. Latency budget is a comercializar concern; under comprovar posture, latency does not gate the comprovation criterion. Deferred.
- **F7** — Adversarial lifecycle workflows beyond the substrate's existing surface (cross-category-lifecycle OMQ surfaces naturally under F3 pressure; broader work deferred).
- **F8** — Privacy-preserving substrate (GDPR / CCPA tension with §2.1 immutability). Relevant when customer-real datasets enter; not under comprovar.
- **F9** — Cross-tenant inference sharing. Platform-tier; deferred.
- **F10** — Performance / scale architecture. Production volume pressure; not under comprovar.

## Origem de dados (per D2 resolution)

Three sources in parallel, with distinct temporal profiles. Decision recorded in cross-Claude posture deliberation: combined yields maximum epistemic yield within the six-month window.

| Source | Time-to-first-pressure | Coverage | Bias profile |
|---|---|---|---|
| **Public datasets** (CIC-IDS, ENISA, academic) | 4–6 weeks (adapter + parser) | Known scenarios | Diagnosable bias (years of literature) |
| **Synthetic generator** (own engineering, designed-by-adversarial-engineer) | 4–8 weeks (gen design + coverage justification) | Coverage of lacunas in public datasets | Bounded by generator's imagination; circular if designed by F3 team |
| **Honeypot** (own infra, legal-reviewed) | 8–12 weeks (legal + collector + curation) | Real non-curated traffic | Unknown initially; observable with lag |

**Synthetic discipline (commitment):** the generator is a sub-project with its own README and justification of coverage. Each category of synthetic pattern must be justifiable against either a public real-world example or an articulated adversarial hypothesis — **never against an assumption of what F3 detects**. Without this discipline, synthetic collapses into circular validation. Recommended that the synthetic generator engineer be separate from the F3 engineer; pair external review of coverage claims.

**Honeypot legal-review timing:** legal review of jurisdiction, retention, anonymization must begin **month 1**, in parallel with F1, not later. Under brand jurisdictions (Brazil, US with counsel): 4–6 weeks. Under EU post-GDPR: 8–12 weeks. Subdimensioning this critical path foreshortens the honeypot yield window.

## Estrutura de signatures (per D3 + D3-arch resolution)

Three-layer signature partition, all in main repo (D3-arch = maximum constitutional cohesion).

- **Camada A — Canonical aberta.** Classical patterns described in literature. Repo principal. Critério: exists in public paper or vendor documentation; openness demonstrates the substrate over known patterns, not IP claims.
- **Camada B — Exploratória aberta.** Multi-modal combinations without established canon. Repo principal. Critério: not for sale; used to generate honest substrate pressure; openness maximizes scientific value and positive externality to the community.
- **Camada C — Domínio com embargo.** Signatures whose immediate openness reduces their adversarial utility (publish a detection rule; adversary adapts the next day). Repo principal. **Embargo via decision-log entry, not segregation física.** Signature C enters the open repo from the first commit; its USE in public demonstration / paper / external talk waits for the decision-log entry of release. The code is never private; the COMMUNICATION about the code has a window.

The fourth layer (signature comercial fechada) **does not exist in this window** by deliberate posture choice. Under comprovar puro, creating commercial-closed signatures yields work whose epistemic value cannot be captured (substrate is being proven, requires signatures inspecionáveis) and whose commercial value is not being pursued. Double-waste; explicitly excluded.

## Assumptions explicitas

These assumptions are **load-bearing** for the comprovation criterion. If they fail, the criterion's falsifiability degrades or breaks.

- **Capacity assumption.** Ordering and milestone dates assume **single-developer baseline**. Capacity ≥ 2 developers reduces timeline proportionally; capacity < 1 (interrupted attention, context-switching above 50%) extends it. The criterion is not falsifiable independent of capacity — non-firing of sub-benchmarks under capacity < baseline is uninformative about substrate.
- **Jurisdiction assumption.** Honeypot legal-review window assumes Brazil or US-with-counsel (4–6 weeks). EU-post-GDPR pushes Sub-benchmark 2 timing by ~4 weeks; under that jurisdiction, the 6-month window may slip Sub-benchmark 2 to month 7.
- **Charter stability assumption.** No Charter amendments during the window that invalidate F1–F6 structural assumptions. Patch amendments are tolerated (precedent: v0.7.2 stale-anchor sweep mid-RFC-framing was beneficial); §-section amendments would require RFC re-baseline.
- **Decision-log empirical-pressure discipline.** Every constitutional pressure surfaced during F1–F6 build is captured as decision-log entry per [`§0022`](../../charter/decision-log.md) methodology. The window's secondary yield (alongside the primary comprovation outcome) is the corpus of decision-log entries documenting how the substrate held under domain pressure.

## Anticipated OMQs (structural pressures the RFC expects to surface)

Per the §0137 + §0142 anchor-inventory pattern: declare anticipated pressures before work begins so that surfacing is recognized as expected, not surprise. Five OMQs anticipated; two warrant separate ontology registration (proposed for opening as discrete ontology-revision RFCs):

| OMQ | Surface | Trigger | Disposition |
|---|---|---|---|
| **Granularity of derivation** (provenance-model.md OMQ #1) | Existing open OMQ in [`provenance-model.md`](../../ontology/provenance-model.md#open-modeling-questions) | F3 windowed signatures (e.g., uniform cadence over N events) — enumerate every contributing event, or reference the window definition | Resolve under F3 pressure; existing OMQ surface; no new registration needed. |
| **Operational definition versioning** (lifecycle-semantics.md OMQ) | Existing open OMQ in [`lifecycle-semantics.md`](../../ontology/lifecycle-semantics.md#open-modeling-questions) | First signature revision in flight (signature v1 → v2; constructs already in substrate under v1 definition — re-derive, mark stale, or leave intact?) | Resolve under F2/F3 versioning pressure; existing OMQ surface; no new registration needed. |
| **Cross-category lifecycle interactions** (lifecycle-semantics.md OMQ) | Existing open OMQ in [`lifecycle-semantics.md`](../../ontology/lifecycle-semantics.md#open-modeling-questions) | First demotion of a hypothesis that was previously consumed as enrichment by a Cat II construct | Resolve under F3 first-demotion pressure; existing OMQ surface; no new registration needed. |
| **Phenomenon-vs-record reconciliation under multi-source ingestion** | **NEW** — proposed for registration in [`entity-model.md` Open Modeling Questions](../../ontology/entity-model.md#open-modeling-questions) | Same observational phenomenon (e.g., a TLS fingerprint) ingested from CIC-IDS + synthetic + honeypot produces three distinct substrate Cat I records (different `actor_ref` per fonte, different content-addressable identifiers). F3 inference computing over union vs filtered-to-unique faces reconciliation question. | **Propose opening separate `ontology-revision-phenomenon-vs-record-reconciliation` RFC** when F1 has two sources stable. |
| **Authentication-class typing** | **NEW** — proposed for registration in [`entity-model.md` Open Modeling Questions](../../ontology/entity-model.md#open-modeling-questions) | F1 multi-modality forces distinction between server-authenticated, client-attested, client-witnessed observations. `actor_ref` does not capture authentication-class dimension today. | **Propose opening separate `ontology-revision-authentication-class-typing` RFC** when F1 has three modalities stable. Likely affects entity-model.md Cat I structure. |

Cross-domain provenance OMQ (provenance-model.md OMQ #4) is **explicitly out-of-scope** for this RFC; anti-bot is the only domain considered.

## Constitutional Review

Per [`rfc-author` §1](../../../.claude/skills/workflow/rfc-author/SKILL.md), the Q1–Q6 impact analysis.

### Q1 — Which Charter invariants does this RFC touch?

**No Charter prose modified.** The RFC consumes Charter v0.7.2 as given:

- **§2.1** (frozen) — Cat I observation records committed by F1 adapters are subject to substrate immutability per §2.1. F3 formation events are Cat I substrate records per [§2.5 BC](../../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) and inherit §2.1's immutability.
- **§2.2** (frozen) — Cat II constructs derived from F1 observations under versioned operational definitions per §2.2 Q1 determinism. F2 library extends under §2.2 discipline.
- **§2.3** (frozen v0.4 + v0.7.2 sweep) — provenance chains terminate at Cat I primaries per §2.3 Structural Requirement. F1 observations are the chain termini; F2 constructs are typed transit; F3 hypotheses are typed reference targets.
- **§2.4** (frozen v0.5 + v0.7.2 sweep) — F3 declares `influenced_by` chain at formation event time per §2.4 BC3 (substrate-time generation per [`§0021`](../../charter/decision-log.md)). Transitive scope is closure per [`§0134`](../../charter/decision-log.md) Candidate τ with β-graph storage per §2.4 BC6 post-sweep.
- **§2.5** (frozen v0.3) — Lifecycle events for F3-formed hypotheses are Cat I records subject to §2.5. Layer A + Layer B candidacy predicates already wired through demote-* CLIs per [`§0141`](../../charter/decision-log.md).
- **§2.6** (frozen v0.6 + v0.7.2 sweep) — F3 commits `evidential_independence` at formation per §2.6 paired-dimension commitment + [`§0136`](../../charter/decision-log.md) canonical-serialization-contract + [`§0140`](../../charter/decision-log.md) marshalling-boundary enforcement. EI computation rule is [`§0133`](../../charter/decision-log.md) Candidate α (source-count ratio) per §2.6 BC1 post-sweep.
- **§3 N1 — no truth.** Signatures of Camada A/B/C produce hypotheses (Cat III), not definitive claims. Demotion is structural per §2.5; the substrate never asserts truth, only records typed claims under explicit category.
- **§3 N3 — no autonomous irreversible action.** Demote-* CLIs are operator-elected per [`§0119`](../../charter/decision-log.md) audit-symmetry posture. Layer A + Layer B verdicts are advisory per [`§0141`](../../charter/decision-log.md) E1; signatures never auto-commit demotions.
- **§4 criterion 1** — sub-benchmarks are mechanically observable (substrate replay produces non-ambiguous verdict). Instrumentation by subtype × fonte × morfologia-de-chain is structurally-checkable at the marshalling boundary.

### Q2 — Does this RFC implicitly redefine any term in the glossary?

No. Canonical vocabulary used per glossary discipline. Implementation-locus terms (`signature engine`, `adapter`, `honeypot`, `morfologia-de-chain`) are domain-pack vocabulary, not Charter / Ontology vocabulary; they do not require glossary entries.

### Q3 — Does this RFC implicitly resolve any of the five open Ontology questions?

No. All five original `ontology.md` Open Questions are resolved as of [`§0134`](../../charter/decision-log.md). This RFC composes with the resolutions; does not re-open them. Five **NEW** OMQs are anticipated, three on existing OMQ surfaces (provenance-model OMQ #1; lifecycle-semantics OMQ × 2) and two requiring registration (phenomenon-vs-record; authentication-class), with proposed separate ontology-revision RFCs.

### Q4 — Does this RFC require Charter amendment?

No. The RFC consumes Charter v0.7.2 frozen state. It anticipates that domain work may surface pressures that motivate future Charter amendments — but no amendment is proposed in this framing. Per §0022 empirical-pressure methodology, amendments surface as decision-log entries first, then advance to formal amendment if structurally required.

### Q5 — Does this RFC introduce a new invariant?

No. Domain-pack-level. No new Charter invariant. F4 (authentication-class typing) may require entity-model.md structural amendment when its OMQ resolves; that amendment would be Ontology-level, not Charter-level, and would proceed under ordinary RFC discipline. Canonical-vocabulary note: see Q3 above on watchlist-compliant naming.

### Q6 — Does this RFC propose ceremony without behavioral consequence?

No. The comprovation criterion produces a falsifiable outcome at month 5–6: either sub-benchmark 1 + sub-benchmark 2 pass (substrate property confirmed under exercised conditions) or they fail (substrate property unconfirmed; specific outcome reading per the falsifiability outcomes table determines next move). The criterion is structurally observable, not interpretation-bound. Either outcome is informative per §4 criterion 1.

## Ordering (under capacity assumption — single-developer baseline)

Per cross-Claude triangulation. F-code references map to the scope section above.

| Month | P0 immediate | P0 sequential | P1 paralelo | Notes |
|---|---|---|---|---|
| **1** | F1.NetworkObservation (lowest-friction modality) | Adapter público #1 (CIC-IDS) begins | Honeypot legal review begins (critical path) | Single-developer: F1 alone fills the month. |
| **2** | F1.BehavioralObservation | Synthetic generator v0 engineering (separate engineer ideally) | Adapter público #1 complete | F1+synthetic+legal-review parallelism stretches single-developer. |
| **2–3** | F1.AttestationObservation + F1.BrowserObservation (may need sub-RFC for discriminated-union decision) | F4 OMQ formal opens (authentication-class surface pressure from F1) | Honeypot infra provisioning | F4 surfaces naturally; propose ontology-revision RFC opening. |
| **4–5** | F3 AutomationGroup minimum viable (canonical aberta signatures) | F6 substrate-read contract drafted in parallel with F3 | F2 Cat II library extends under F3 pressure | F3 starts only when F1 has 2 stable modalities. F2 responsive, not front-loaded. |
| **4–5** | Phenomenon-vs-record OMQ surfaces (multi-source reconciliation pressure) | Propose ontology-revision RFC opening | Sub-benchmark 1 attempt (public + synthetic sources) | First comprovation gate. |
| **5–6** | F3 BehavioralCluster extends (camada B) | F6 audit-grade navigability over F3 output | Sub-benchmark 2 attempt (honeypot non-curated) | Second comprovation gate; capture pressures as decision-log entries. |
| **6** | §0022 final capture | Possible Charter amendment candidates identified | OMQs new for next window opened | Window closure with decision-log corpus. |

**Out-of-scope per posture:** F5 (real-time decisioning), F7 (broader adversarial workflows beyond cross-category-lifecycle OMQ), F8 (privacy), F9 (cross-tenant), F10 (performance/scale).

## References

- [`decision-log §0022`](../../charter/decision-log.md) — implementation gate cleared; empirical-pressure methodology.
- [`decision-log §0023`](../../charter/decision-log.md) — Q2 Identity tiers resolved; inception-phase single-tier `actor_ref`.
- [`decision-log §0133`](../../charter/decision-log.md) — Q3 formal independence Candidate α (source-count ratio).
- [`decision-log §0134`](../../charter/decision-log.md) — Q5 transitivity Candidate τ (transitive closure with β-graph storage).
- [`decision-log §0136`](../../charter/decision-log.md) — canonical-serialization-contract paired-dimension consolidation.
- [`decision-log §0140`](../../charter/decision-log.md) — paired-dimension marshalling-boundary enforcement.
- [`decision-log §0141`](../../charter/decision-log.md) — Layer B service-tier implementation (conservative-defaults bundle).
- [`decision-log §0142`](../../charter/decision-log.md) — Charter v0.7.2 stale-anchor sweep; first deliberate enaction of anchor-inventory checklist.
- [`amendments.md` v0.7.2`](../../charter/amendments.md) — Q2/Q3/Q5 stale-anchor sweep precedent for this RFC's clean citations.
- [`charter-amendment-v0-7-2-stale-anchor-sweep`](./charter-amendment-v0-7-2-stale-anchor-sweep.md) — RFC for the prior patch amendment.
- [`ontology-revision-q3-independence`](./ontology-revision-q3-independence.md) — Q3 resolution RFC; template precedent for ontology-revision constitutional framing.
- [`operational-spec-layer-b-service-tier-implementation`](./operational-spec-layer-b-service-tier-implementation.md) — service-tier implementation RFC; template precedent for operational-spec executable criterion.
- [`docs/rfcs/discussion/domain-pack-v0-1-evidence.md`](../discussion/domain-pack-v0-1-evidence.md) — evidence scratch for substantive deliberation Phase 2–5.

## Decision Record

This RFC opens at framing-PR phase. Substantive deliberation (Phase 2: sub-decisions enumeration; Phase 3–5: candidate analysis + synthesis + recommendation) proceeds in the evidence file. Resolution will be recorded in a subsequent decision-log entry once substantive phase closes.

Cross-Claude posture deliberation (D1 = 6 months free; D2 = three sources parallel; D3 = A/B/C in main repo; D4 = discarded) is **load-bearing context** for this RFC; the framing assumes that resolution and does not re-open it.
