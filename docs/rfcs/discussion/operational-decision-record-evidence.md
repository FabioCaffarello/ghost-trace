# Evidence — Operational Decision Record

Discussion-phase evidence companion to [`ontology-revision-operational-decision-record.md`](../draft/ontology-revision-operational-decision-record.md). Records the deliberation framing that the resolving decision-log entry will draw on. No resolution is asserted here; this file structures the comparison the committee will converge over.

## 1. The empirical pressure (how the question surfaced)

Per the [`§0022`](../../charter/decision-log.md) empirical-pressure methodology, the question was surfaced by implementation, not anticipated in the abstract. The [`§0221`](../../charter/decision-log.md) TLS fingerprint vertical slice was scoped to carry one anti-bot observation sub-modality (JA3 + JA4) end to end through every existing layer. Three of the four requested layers landed cleanly on existing structure:

| Layer | Landed on | Constitutional status |
|---|---|---|
| Collection | Cat I `NetworkObservation` / `tls_ja4` (+ JA3 schemas-evolution) | existing |
| Inference | Cat III `AutomationGroup` (paired confidence + evidential independence), with `source_event_hashes` provenance | existing |
| Replay | `ReconstructAutomationGroupProvenance` (obs→inference chain) | existing |
| **Decision** | **— nothing —** | **greenfield** |

The Decision layer had no landing site. A grep of `entity-model.md`, the schemas, and the Charter for "decision / action / policy / block / challenge / shadow / enforcement" returns no record type. The Domain Pack v0.1 RFC had already, independently, deferred F5 (real-time decision serving) and parked decisioning outside the comprovation window. The slice therefore stopped and opened this RFC rather than inventing a record type against the constitution.

## 2. The §2.2 decomposition

"Decision" conflates two separable things, which is why a single category answer is non-obvious:

1. **Policy evaluation** — `verdict = P_v(hypothesis_state, thresholds)`. Deterministic given a *versioned* policy `P_v` and the hypothesis state read at evaluation time. This is the Category II shape (deterministic derivation under a versioned operational definition — exactly `OperationalSession`'s shape).
2. **Action enactment / audit** — "verdict V was enacted on actor A at time T." An observation (record of historical fact). This is the Category I shape, and it is the artifact §3 N3 ("commit an audit Cat I record BEFORE the action") already presupposes.

The framing choice is whether to model (1), (2), both, or neither in the substrate.

## 3. Candidate framings compared

| Dimension | A — Cat I audit | B — Cat II construct (+ Cat I audit) | C — external consumer |
|---|---|---|---|
| §2.2 category | I (observation) | II (eval) + I (enactment) | none (outside substrate) |
| New protos | 1 | 2 | 0 |
| §2.3 provenance | by `source_observation_hashes` + `influencing_hypothesis_hashes` | full typed `subject_ref_*` chain | n/a (no substrate record) |
| §2.4 influence | by reference (Cat I carries no `influenced_by`) | native `influenced_by` at commit | out of §2.4 scope |
| §2.6 paired dims | not carried (Cat I excluded per BC3); referenced upstream | carried/inherited on the Cat II eval | n/a |
| §3 N3 audit-on-commit | directly satisfied (IS the audit record) | satisfied via the Cat I enactment pairing | satisfied only if the external action commits an audit record |
| Replay | Phase-3 reconstructive over the audit record's references | Phase-2 deterministic (re-evaluate `P_v`) + reconstructive | reconstructs inputs only; no committed decision |
| obs→inference→**decision** auditable end to end | yes | yes (richest) | no (final hop external) |
| Cost / surface | low | high (two record types + policy-as-Cat-II question) | lowest |
| Precedent in repo | `OrphanCleanupAudit` ([`§0104`](../../charter/decision-log.md)) | `OperationalSession` / `DerivedActorAttribution` Cat II derivations | Charter §3 BC external-system carve-out |

### Reading

- **A** is the smallest constitutional step that still puts the decision in the substrate and discharges §3 N3 by construction (the audit record IS the §3 N3 artifact). Its open edge is §2.6: a Cat I record cannot carry paired dimensions, so the decision's evidential posture is *referenced*, not *embodied*.
- **B** is the richest for replay and influence disclosure (the decision becomes a first-class inferential assertion with native `influenced_by` + paired dimensions), at the cost of two record types and a second sub-question — whether the *policy* is itself a versioned Cat II operational definition.
- **C** is the most conservative and aligns with the Domain Pack's F5 deferral and the Charter §3 external-system carve-out. It keeps the substrate purely epistemic (observe + infer) and leaves acting to consumers. Its cost is exactly the §0221 goal it cannot meet: the obs→inference→**decision** chain is not reconstructible from the substrate, because there is no committed decision.

## 4. Convergence criteria (for the resolving entry)

The committee entry that resolves this should state, in canonical vocabulary:

1. The chosen framing (A / B / C) and the explicit §2.2 categorization it commits.
2. Whether decision = single record or evaluation+enactment pair.
3. The `policy_ref` representation (and whether policy is a Cat II definition).
4. The §3 N3 audit-on-commit mechanism for the enforcement action.
5. The replay contract (Phase-2 vs Phase-3) the framing yields.
6. For A: how the Cat I audit references upstream `confidence` / `evidential_independence` without carrying them.

Until that entry lands, no decision-layer code is written — the §0221 slice ships obs→inference→replay only, and this file plus the draft RFC are the constitutional placeholder.

## 5. References

- [`ontology-revision-operational-decision-record.md`](../draft/ontology-revision-operational-decision-record.md) — the draft RFC.
- [`decision-log §0221`](../../charter/decision-log.md) — slice + question opening.
- [`domain-pack-v0-1-anti-bot-atlas.md`](../draft/domain-pack-v0-1-anti-bot-atlas.md) — F5 deferral.
- [Charter §3 N3 + BCs](../../charter/constitutional-charter.md#3-non-goals).
