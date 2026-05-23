# Domain Pack v0.1 — Anti-Bot Atlas — discussion evidence

**Status:** in-discussion. Not authoritative. Final resolution will be recorded in decision-log entries (multi-phase: framing acceptance + per-sub-decision resolutions as F-codes complete + final window-closure recapitulation at month 6).

This scratch supports the discussion phase of [`domain-pack-v0-1-anti-bot-atlas`](../draft/domain-pack-v0-1-anti-bot-atlas.md). The RFC opens at framing-PR phase per cross-Claude posture deliberation (D1 = 6 months free; D2 = three sources parallel; D3 = A/B/C open in main repo; D4 = discarded). Charter v0.7.2 is the substrate ground; Layer B at [`§0141`](../../charter/decision-log.md) is the operational mechanics ground; this is the first applied domain vertical.

This is a framing-phase scratch: Phase 1 names the scope, dependency surface, and posture; Phase 2 enumerates sub-decisions per F-code; Phases 3+ (epistemic-skill application, comparison synthesis, recommendations) are drafted in subsequent commits as F-codes complete.

---

## Phase 1 — Scope, dependencies, posture

### The question

Per cross-Claude triangulation closure:

> The substrate, as frozen in Charter v0.7, is the only known architecture that structurally prevents recursive belief inflation via paired-dimension commitment of §2.4 + §2.6. Build the minimum domain atlas (typed observations + inference engines + operator-grade explainability) under a six-month window. Demonstrate the substrate's structural property via a falsifiable comprovation criterion that distinguishes substrate-survival from substrate-evasion.

This RFC asks: **What is the minimum applied domain pack on the substrate that exercises §2.4 + §2.6 structural defense under realistic adversarial pressure within six months, with a falsifiable comprovation outcome at month 5–6, capturing every constitutional pressure surfaced as decision-log entries per [`§0022`](../../charter/decision-log.md) empirical-pressure methodology?**

### Posture statement (load-bearing precondition)

All sub-decisions enumerated below assume the posture closed in cross-Claude triangulation. Reversal of any of these axes invalidates the corresponding sub-decisions:

- **D1 — 6 months free of revenue pressure.** Comprovar puro is feasible; no commercial-surface deferrals required.
- **D2 — Three sources in parallel (public + synthetic + honeypot).** Public arrancam fast (4–6 weeks); synthetic covers imagined lacunas (4–8 weeks); honeypot accumulates non-curated real pressure (8–12 weeks legal + collector).
- **D3 — Signatures partitioned A (canonical aberta) / B (exploratória aberta) / C (domínio com embargo via decision-log).** All three layers in main repo (D3-arch = maximum constitutional cohesion). Quarta layer (comercial fechada) does not exist in this window by deliberate choice.
- **D4 — Estaca-zero constitucional discarded.** Three of four remaining OMQs (granularity, op-def-versioning, cross-category-lifecycle) are triggered-by-F3; resolving them in estaca-zero would be speculation. Resolution under F3 pressure follows §0022 methodology.

**Inception-phase corollary.** Per [`§0023`](../../charter/decision-log.md) Identity-tiers + [`§0141`](../../charter/decision-log.md) Layer B service-tier conservative-defaults bundle precedents: when a simpler form is structurally sufficient and the complex variant has no current empirical justification, the simpler form is adopted with reversal conditions documenting when the complex variant would be revisited. This pack defaults to the simpler implementation form for each sub-decision and documents reversal conditions.

### Dependency surface

**Discharged dependencies** (no further work required for this RFC to proceed):

- **Charter fully frozen** — Amendment v0.7 at [`§0131`](../../charter/decision-log.md); all ten Charter sections frozen. ✓
- **Stale-anchor sweep complete** — Patch v0.7.2 at [`§0142`](../../charter/decision-log.md); §2.4 BC3/BC6 + §2.6 BC1/BC2/BC6 reference current resolutions. ✓
- **Implementation gate cleared** — [CLAUDE.md §6.4](../../../.claude/CLAUDE.md) gate cleared at [`§0022`](../../charter/decision-log.md). ✓
- **Layer B operational arc complete** — [`§0011`](../../charter/decision-log.md) → ... → [`§0141`](../../charter/decision-log.md) nine-decision arc; PRs #158–#161 wired Layer B through demote-* + promote-* CLIs. ✓
- **Q2/Q3/Q5 Ontology resolutions landed** — [`§0023`](../../charter/decision-log.md) Q2 + [`§0133`](../../charter/decision-log.md) Q3 + [`§0134`](../../charter/decision-log.md) Q5. ✓
- **Canonical-serialization-contract crystallized** — [`§0136`](../../charter/decision-log.md) (paired-dimension + α + τ + β-graph + L-BC-OR consolidated); [`§0139`](../../charter/decision-log.md) (hash-list discipline generalized); [`§0140`](../../charter/decision-log.md) (paired-dimension marshalling-boundary enforced). ✓
- **Cross-subtype framework closed** — [`§0122`](../../charter/decision-log.md)–[`§0127`](../../charter/decision-log.md) (typing γ + enablement B+D + attachment β + pair-table). ✓
- **Schemas + storage + auth + language technology selections accepted** — `architecture-schemas-technology-selection.md` (proto3), `architecture-storage-technology-selection.md` (substrate tiers), `architecture-http-auth-scope-model.md` (T3/T4 model), `architecture-implementation-language-selection.md` (Go). ✓
- **Tier 1/2/3 verification rule established** — Cross-Claude triangulation closed verification discipline: letter+number citations (tier 1) require grep before citation; number-only citations (tier 2) require enumeration verification; content-claim citations on entries (tier 3) require integral entry-reading before action. First two enactions: my BC5→BC3 miscite (tier 1) caught by Claude Code grep; UI Claude `subject_ref_*` claim on Cat I observations (tier 3) caught by Claude Code §2.6 BC3 grep. ✓

**Open dependencies** (this RFC's scope, enumerated for substantive deliberation in Phase 2):

- **F1 — Typed observation taxonomy.** Four sub-decisions: (1) order of modality introduction; (2) NetworkObservation field set; (3) AttestationObservation typing — discriminated union per protocol vs generic envelope; (4) BrowserObservation typing — single type vs sub-discrimination.
- **F2 — Cat II construct library extension.** Two sub-decisions: (5) construct-versioning surface (parallel to `uniform_cadence_v1.go` precedent? new pattern?); (6) when constructs land (front-load vs F3-responsive — already committed to F3-responsive by posture).
- **F3 — Signature engines.** Three sub-decisions: (7) signature-definition format (Go-code vs declarative vs hybrid); (8) initial subtype scope (AutomationGroup-first per F1 dependency vs BehavioralCluster vs both); (9) instrumentation surface (substrate-committed vs telemetry-only — per phenomenon-vs-record OMQ surface).
- **F4 — Trust-of-observation typing.** One sub-decision: (10) registration timing (open OMQ at F1 modality 2 vs at F1 modality 3 vs as separate ontology-revision RFC NOW). Per posture: open OMQ as separate ontology-revision RFC when F1 has three modalities stable (anticipated month 2–3).
- **F6 — Operator-grade explainability.** Three sub-decisions: (11) F6 substrate-read contract scope (full export format vs minimum hash-navigation); (12) F6 surface form (CLI-only this window vs CLI + minimum HTTP); (13) F6 interaction with existing httpapi (extend vs separate service).
- **Data sources.** Three sub-decisions: (14) public dataset order — CIC-IDS first vs ENISA first vs both adapters in parallel; (15) synthetic generator scope — own-engineering separate from F3 vs same-team-with-documented-brief vs pair-external-review-only; (16) honeypot collector architecture — own infra vs cloud-vendor sandbox vs lightweight VM.
- **Layer C embargo mechanism.** Already resolved per posture closure: config (1) — embargo via decision-log communicational discipline; code always public. Sub-decision-(17) closed.

**Anticipated OMQs** (structural pressures expected during F1–F6 build):

- Granularity of derivation (provenance-model.md OMQ #1) — surfaced under F3 windowed signatures.
- Op-def-versioning (lifecycle-semantics.md OMQ) — surfaced under F2/F3 first revision.
- Cross-category lifecycle interactions (lifecycle-semantics.md OMQ) — surfaced under F3 first demotion with Cat II downstream.
- Phenomenon-vs-record reconciliation under multi-source ingestion — NEW; propose registration in entity-model.md OMQ section + separate ontology-revision RFC when F1 has two sources stable.
- Trust-of-observation typing — NEW; propose registration in entity-model.md OMQ section + separate ontology-revision RFC when F1 has three modalities stable.

Cross-domain provenance OMQ (provenance-model.md OMQ #4) is **explicitly out-of-scope** for this RFC.

### Implementation surface inventory

Empirical audit (as of [`§0142`](../../charter/decision-log.md) merge):

- **Active services**: `services/ingestion/` only. 193+ Go files; 38 CLI commands. Internal packages: canonical, cliutil, derivation, genproto, httpapi, hypothesis (with `layerb/` subpackage), ingest, orphan, projection, replay, substrate, tools, verify. No `signatures/`, no `adapters/` package — these are new.
- **Stub services**: `assertion-engine/`, `graph/`, `projections/`, `replay/` — README-only. Not in F1-F6 scope this window.
- **Schemas surface**: `schemas/events/v1/` — current proto types include NetworkEvent (opaque event_descriptor bytes), DeclaredSession (Q1 §0015), OperationalSession (Q1 §0015), hypothesis lifecycle events for 4 subtypes × 6 operations. NetworkObservation / BehavioralObservation / AttestationObservation / BrowserObservation do not exist; F1 introduces them.
- **Cat II constructs**: Four toys in `services/ingestion/internal/hypothesis/` — `uniform_cadence_v1.go`, `co_occurrence_window_v1.go`, `session_descriptor_shared_v1.go`, `temporal_descriptor_cohort_v1.go`. All operate on DeclaredSession.
- **Cat III subtypes**: Four — BehavioralCluster, AutomationGroup, CampaignHypothesis, CoordinationRing — with full lifecycle CLIs (form/promote/demote/dissolve/merge/split) per [`§0010`](../../charter/decision-log.md) Q2-A.2.
- **Hypothesis lifecycle ops**: All wired through `services/ingestion/cmd/` CLIs. Layer A advisory + Layer B advisory surfaced through demote-* CLIs per [`§0141`](../../charter/decision-log.md) E1. Promotion CLIs accept LayerBParameters per [`§0141`](../../charter/decision-log.md) F3.
- **Canonical-serialization-contract**: `docs/architecture/canonical-serialization-contract.md` — paired-dimension at marshalling boundary per [`§0136`](../../charter/decision-log.md) + [`§0140`](../../charter/decision-log.md); β-graph storage; hash-list element-shape discipline per [`§0139`](../../charter/decision-log.md).
- **Existing httpapi**: bounded to verify + ingest paths. F6 may extend or stand parallel.

### Out-of-scope (cross-referenced from RFC)

- **F5** real-time decision serving.
- **F7** broader adversarial workflows beyond the surfaced cross-category-lifecycle OMQ.
- **F8** privacy-preserving substrate (relevant when customer-real ingestion enters).
- **F9** cross-tenant intelligence sharing.
- **F10** performance / scale architecture.
- Cross-domain provenance OMQ (provenance-model.md OMQ #4).
- Fourth signature layer (comercial fechada).
- Schema technology re-selection ([`§0024`](../../charter/decision-log.md) proto3 is accepted; F1 adds proto types within that selection).
- Storage technology re-selection ([`architecture-storage-technology-selection.md`](../draft/architecture-storage-technology-selection.md) accepted).
- Auth model re-selection ([`architecture-http-auth-scope-model.md`](../draft/architecture-http-auth-scope-model.md) accepted; F6 may consume but not modify).
- Implementation language re-selection (Go accepted).

---

## Phase 2 — Sub-decision enumeration (placeholder)

To be drafted in subsequent commits as F-codes complete. Each F-code surfaces its sub-decisions with candidate variants enumerated; Phase 3 applies epistemic-skill review (falsifiability-check, ambiguity-reducer, vocab-discipline, marketing-tell-detector); Phase 4 synthesizes comparison findings; Phase 5 produces recommendation per sub-decision.

Sub-decisions enumerated in Phase 1 ("Open dependencies" above): seventeen sub-decisions across F1 (4), F2 (2), F3 (3), F4 (1), F6 (3), ingestion sources (3), Layer C embargo (1 — already closed by posture).

---

## Phase 3+ (placeholder)

Pending Phase 2 completion.
