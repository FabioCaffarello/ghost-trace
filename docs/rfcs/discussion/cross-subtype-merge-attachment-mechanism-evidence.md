# Cross-subtype merge attachment mechanism — discussion evidence

**Status:** in-discussion. Not authoritative. Final resolution will be recorded in decision-log and Ontology document revision.

This scratch supports the discussion phase of [`ontology-revision-cross-subtype-merge-attachment-mechanism.md`](../draft/ontology-revision-cross-subtype-merge-attachment-mechanism.md). The phases mirror the established Ontology-RFC discussion-phase procedure (per [`q4-evidence.md`](./q4-evidence.md), [`cross-subtype-merge-typing-evidence.md`](./cross-subtype-merge-typing-evidence.md)).

## Phase 1 — Evidence

Three candidates (α / β / γ) × five dimensions. Cells cite the draft RFC where applicable.

| Dimension | α — Field-on-target-proto | β — Separately-committed attachment record | γ — Inferred-from-antecedent-structure |
|---|---|---|---|
| **1. Proto / type-layer implications** | 3 target protos gain 1–3 new fields each (AG: +1; CR: +2; CH: +2). Total: 5 new fields across 3 protos. Each addition is a [`§0024`](../../charter/decision-log.md) canonical-serialization-contract evolution event. | 1 new Cat I proto (`CrossSubtypeMergeAttachment`) introduced; existing target protos unchanged. Precedent: [`§0042`](../../charter/decision-log.md) typed-Cat-I-proto pattern + [`§0104`](../../charter/decision-log.md) OrphanCleanupAudit pattern. | Zero proto changes. Existing merge event proto already references both antecedent formation hashes per [`§0049`](../../charter/decision-log.md). |
| **2. Read-time consumer obligations** | Direct field-access: `formation.operatorship_signature_ref` returns the attached surface. No joins required. | One-record join: consumer queries substrate for `CrossSubtypeMergeAttachment` records referencing the merged formation hash; projection layer may materialize join. | Multi-record traversal: consumer follows merge event → antecedent hash → antecedent's formation record → antecedent's signature/structure field. Multiple substrate reads per consumer per cross-subtype-merge record. |
| **3. Replay-correctness implications** | Phase 3 reconstructive replay reconstructs the merged record with field-level expected values; helper-extracts-antecedent-surface deterministic if canonical-serialization preserved byte-for-byte. | Phase 3 replay reconstructs merged formation + attachment record as a pair (committed atomically via AppendPair); both must replay deterministically. The pairing IS the replay anchor. | Phase 3 replay reconstructs only the merged formation; the antecedent traversal is consumer-side, not part of replay. Replay does not check that the antecedent-implied surface is consistent with what consumers retrieve. |
| **4. Falsifiability surface (§1.3)** | Per cell: field present on target → check field-value validity. Type-system-enforced. Strongest §1.3 of the three. | Per cell: attachment record committed atomically with merged formation → check record-pair existence + attachment fields. Strong §1.3 (atomic commit guarantees pair). Surface-payload canonicalization is a sub-question (does `surface_payload` byte-match the antecedent's original payload?). | Weakest §1.3: no structural commitment to "attachment exists". A consumer that fails to traverse antecedents simply misses the surface silently. The structural commitment is "antecedents recorded" — not "non-canonical surface attached to merged". |
| **5. Lock-in / migration** | Heaviest: each proto-field addition is a [`§0024`](../../charter/decision-log.md) evolution event; removing fields later requires deprecation discipline; vestigial fields under cell reversal per §2.1. | Moderate: one new proto added once; per-cell extensions reuse the proto via `surface_type` discriminator. Forward-extensible to future cells without proto change. | Lightest: no new structure; mechanism evolves entirely in consumer projections; no §2.1 lock-in. |

## Phase 2 — Surface scaffold implicit assumptions

The relevant scaffolds are [`entity-model.md` §Cross-subtype operations](../../ontology/entity-model.md) (post-§0125+§0126; carries the resolved pair-table) and the four per-subtype formation protos at [`schemas/events/v1/`](../../../schemas/events/v1/).

### `entity-model.md` §Cross-subtype operations

**Verdict: scaffold neutral on attachment mechanism.**

The post-§0125+§0126 entity-model text records the pair-table resolutions (cells 1–6) and notes "the non-canonical-antecedent surface is carried as extension surface (implementation-RFC discipline)" — i.e., the scaffold EXPLICITLY DEFERS the mechanism question to this RFC. The scaffold does not favor any of α/β/γ.

Cost under all three candidates: minimal. The scaffold's deferral language is candidate-neutral.

### Per-subtype formation protos

**Verdict: scaffolds neutral; no implicit lean.**

The four current formation protos (`BehavioralClusterFormation`, `AutomationGroupFormation`, `CoordinationRingFormation`, `CampaignHypothesisFormation`) carry only their canonical recognition-pattern surfaces. None has placeholder fields for non-canonical surfaces. Cost under α: each gains 1–3 fields (proto-evolution event). Cost under β: protos unchanged; new proto added separately. Cost under γ: protos unchanged.

No prior structural commitment biases the choice. The proto layer is genuinely a blank slate for the attachment surface.

### Summary

Unlike the merge typing question (where scaffolds carried a slight α/β framing per [`cross-subtype-merge-typing-evidence.md` Phase 2](./cross-subtype-merge-typing-evidence.md)), the attachment-mechanism question has NO scaffold-level lean. Either α, β, or γ can be adopted without scaffold rework beyond the entity-model post-resolution revision.

## Phase 3 — Apply epistemic skills

Three skills × three candidates against the abstract structural proposition.

- **α proposition:** "Each target subtype's formation proto gains explicit field(s) for the non-canonical-antecedent surface."
- **β proposition:** "A new Cat I `CrossSubtypeMergeAttachment` proto carries the non-canonical-antecedent surface, committed via `AppendPair` alongside the merged formation."
- **γ proposition:** "Consumers retrieve non-canonical-antecedent surfaces via merge event's antecedent references at read time; merged record carries only its canonical surface."

| Skill | α | β | γ |
|---|---|---|---|
| **`falsifiability-check`** | §1.1: a target record has the expected field unpopulated when it should be populated → detectable. §1.2: third party reads the field. §1.3: field-level type check. §1.4: clean — defined in terms of existing typed proto fields. **Verdict: passes all four; strongest §1.3 of the three.** | §1.1: cross-subtype merge committed without paired attachment record → detectable via missing record in substrate. §1.2: third party queries for attachment records. §1.3: record-level type check + atomic pairing check. §1.4: clean. **Verdict: passes all four; §1.3 strong (atomic commit guarantee).** | §1.1: a consumer that misses the antecedent surface produces a different downstream artifact than one that includes it — but the difference is consumer-side, not substrate-side. The merged record itself is identical under γ regardless of consumer behavior. §1.2: structural commitment is "antecedents recorded" — already true. §1.3: WEAKEST — no field/record check operationalizes the "attachment exists" claim. §1.4: clean (no circularity; just no commitment). **Verdict: passes §1.1/§1.2/§1.4 trivially; §1.3 vacuous.** The vacuous §1.3 is the same shape as merge-enablement Candidate A per [`cross-subtype-merge-enablement-evidence.md` Phase 3](./cross-subtype-merge-enablement-evidence.md). |
| **`epistemic-separator`** | The added field is on a typed Cat I formation record; no cross-category conflation. Per-cell field naming risks per-cell semantic drift (`operatorship_signature_ref` vs `automation_signature_ref` could carry subtly different semantics if not committee-defended); risk is documented but not load-bearing. **Verdict: clean with documented per-cell-naming risk.** | The attachment record is a typed Cat I record per the [`§0042`](../../charter/decision-log.md) pattern; the `surface_type` discriminator partitions within Cat I, not across categories. **Risk: the `surface_type` discriminator is structurally analogous to [Charter §2.2's first forbidden anti-pattern](../../charter/constitutional-charter.md#22-epistemic-separation)** ("kind field" within unified record type). The Charter scopes the anti-pattern to cross-category unification; β's `surface_type` is intra-Cat-I, so it does not violate §2.2 directly. But the intra-category-flattening pattern from [`cross-subtype-merge-typing-evidence.md` Candidate β](./cross-subtype-merge-typing-evidence.md) appears here too (with diminished force). **Verdict: passes (no cross-category), with intra-category flattening risk by analogy.** | Antecedent traversal stays within Category boundaries (Cat I → Cat I). No conflation. **Risk: the "structural commitment" is implicit in consumer interpretation** — different consumers may traverse different depths or apply different derivation logic, producing different downstream artifacts from the same substrate. This is the §3 first non-goal failure mode ("does not produce truth") applied at the per-merge level: γ allows multiple consumer-side "truths" about what surface attaches to the merged record. **Verdict: passes, with consumer-interpretation-divergence risk.** |
| **`ambiguity-reducer`** | Watchlist scan: `signature` (already in canonical use); `attachment` (descriptive). Per-cell field names: `operatorship_signature_ref`, `automation_signature_ref`, `derived_pairwise_relationships` — each is a structural commitment requiring committee defense. **Verdict: one carry-forward (per-cell field naming convention).** | Watchlist scan: `surface_type` discriminator string values (`"operatorship_signature"` etc.) need committee-defended enumeration. `surface_payload` (bytes) is operationalized by the canonicalization rule (open question). **Verdict: two carry-forwards (surface_type enum; surface_payload canonicalization).** | Watchlist scan: no new terms introduced. The `consumer-side traversal logic` is itself the ambiguity — different consumers may implement differently. **Verdict: one carry-forward (consumer-traversal-logic standardization; would require a projection-layer spec).** |

### Most consequential finding

**γ's vacuous §1.3 is the disqualifying finding under Charter §4 criterion 1 discipline.** Per the [`cross-subtype-merge-enablement-evidence.md` Finding 1](./cross-subtype-merge-enablement-evidence.md) precedent: a candidate that makes no structural commitment (γ's "no attachment surface; consumers traverse") fails §4's preference for structural enforcement. The same disqualification that ruled out merge-enablement Candidate A rules out γ here.

α and β both pass §4 cleanly. The choice between them is the committee-judgment question per Phase 4.

## Phase 4 — Comparison synthesis

Findings synthesized from Phases 1–3. Numbered in order of consequence. Each classified as **asymmetry** (clear evidence-grounded preference), **apparent trade-off that resolves** (reframed by other phases), or **genuine trade-off** (substantive difference; committee judgment).

### Finding 1 — Asymmetry: structural enforceability disqualifies γ

Sources: Phase 1 cell γ4 (falsifiability surface); Phase 3 (γ, `falsifiability-check`) §1.3 vacuous; (γ, `epistemic-separator`) consumer-interpretation-divergence risk.

γ makes no structural commitment to attachment existence — the "attachment" exists only in consumer interpretation. Per [Charter §4 criterion 1 (frozen v0.2)](../../charter/constitutional-charter.md#4-constitutional-design-rule), the project prefers structural enforcement. Precedent: [`cross-subtype-merge-enablement-evidence.md` Finding 1](./cross-subtype-merge-enablement-evidence.md) disqualified Candidate A (merge enablement) on the same shape — A makes no enablement claim; γ here makes no attachment claim.

The asymmetry against γ is direct. γ is disqualified by the same §4 discipline that disqualified merge-enablement A.

### Finding 2 — Asymmetry: uniform mechanism favors β over α

Sources: Phase 1 cells α1 + β1 (proto/type-layer implications).

α requires 3 proto evolutions (AG: +1 field; CR: +2; CH: +2) plus per-cell field naming. β requires 1 proto addition (attachment record) plus uniform mechanism across all 6 cells. The cost asymmetry is real: α's per-cell field-naming requires per-cell committee defense (parallel to the pair-table per-cell defense at [`§0125`](../../charter/decision-log.md) + [`§0126`](../../charter/decision-log.md)); β's `surface_type` discriminator centralizes the per-cell semantic at a single committee-defended enum.

This favors β on operational-simplicity grounds. The committee-defense cost is centralized rather than distributed.

### Finding 3 — Apparent trade-off that resolves: β's read-time-join cost reframed as projection-layer obligation

Sources: Phase 1 cell β2 (read-time consumer obligations).

β requires consumers to query for attachment records joined with merged formations. This APPEARS to be a per-consumer cost (every reader must join). The reframing: per the [`projection-model.md`](../../architecture/projection-model.md) Phase 4 convention, projections materialize derived views; the join is a projection-layer obligation, not a per-consumer obligation. Once a `MergedFormationWithAttachments` projection materializes the join, consumers query the projection directly — no per-consumer join cost.

Under α, the join is absent (field is on-target); consumers query the target record directly. The cost difference is real but smaller than it appears in Phase 1.

### Finding 4 — Apparent trade-off that resolves: α's proto-evolution cost reframed as one-time

Sources: Phase 1 cell α5 (lock-in / migration).

α requires 3 proto evolutions, one per target subtype. This appears costly. The reframing: the §0024 canonical-serialization-contract event records the evolution; the 3 events can be bundled as a single "cross-subtype merge attachment fields" event. The cost is real but bounded — 3 fields × 3 protos = 5 field additions in a single event.

β has its own one-time cost (1 new proto). The lock-in dimension does NOT strongly differentiate α from β at the field-vs-record level; both are one-time events.

### Finding 5 — Genuine trade-off: explicit vs implicit provenance

Sources: Phase 1 cells α3 + β3 + γ3 (replay-correctness implications); Phase 3 (α/β/γ, `epistemic-separator`).

α encodes provenance via field-on-target (the field's value IS the antecedent's surface, copied). β encodes provenance via separately-committed attachment record (the record explicitly references the antecedent formation hash). γ encodes provenance via merge event's antecedent references (implicit but already-present).

Under [`§2.3 frozen v0.4 BC5`](../../charter/constitutional-charter.md#23-provenance-integrity) multi-category-traversal-shape, all three preserve §2.3. But the EXPLICITNESS of the provenance edge differs:

- α: explicit field-level provenance (the field's value IS the surface)
- β: explicit record-level provenance (the attachment IS a §2.3 BC5 edge)
- γ: implicit via merge event's antecedent-list (no dedicated structural commitment)

β has the strongest §2.3 alignment (the attachment record IS a typed provenance edge). α has field-level provenance (somewhat weaker — the field's value is a CLAIM about what the antecedent's surface was, not a record-level link). γ has implicit-only provenance.

This is a values choice for the committee: how explicit should the provenance edge be?

### Finding 6 — Genuine trade-off: precedent fit

Sources: Phase 1 cells α1 + β1.

α follows the established proto-evolution pattern; many prior canonical-serialization-contract evolution events ([`§0024`](../../charter/decision-log.md) onwards) have added fields per cell-style requirement. β follows the [`§0104`](../../charter/decision-log.md) OrphanCleanupAudit + AppendPair pattern — a Cat I record paired with another via atomic commit.

Both have strong precedent. The precedent dimension does not strongly differentiate; the committee chooses by which structural emphasis the resolution warrants — proto-evolution (α: payload carried within the canonical record) or atomic-pair (β: payload carried in a paired record).

### Summary statement

The evidence has one clear asymmetry: γ is disqualified by §4 discipline (Finding 1). Between α and β:

- β is preferred on uniform-mechanism grounds (Finding 2)
- The apparent costs (β's read-time-join, α's proto-evolution) both reframe to bounded one-time obligations (Findings 3 + 4)
- Provenance explicitness favors β slightly (Finding 5)
- Precedent fit is comparable (Finding 6)

The discussion phase converges on **β preferred over α; γ disqualified**.

## Phase 5 — Conditional recommendation

The discussion phase recommends **Candidate β — Separately-committed attachment record** as the cross-subtype merge attachment mechanism, with **α — Field-on-target-proto** as the conditional fallback.

The recommendation rests on:

- **Finding 1** disqualifies γ on §4 structural-enforceability discipline (precedent from `cross-subtype-merge-enablement-evidence.md` Finding 1).
- **Finding 2** prefers β over α on uniform-mechanism grounds; the per-cell committee-defense cost is centralized at one `surface_type` enum rather than distributed across 5 per-cell field names.
- **Finding 5** favors β slightly on §2.3 BC5 provenance-explicitness grounds (the attachment record IS a typed provenance edge).
- **Findings 3 + 4** reframe the apparent cost asymmetry; neither dimension strongly differentiates.

The third stage of convergence pattern is the same as merge-typing and merge-enablement: Stage 1 disqualifies one candidate on §4 discipline (γ here; β at merge-typing; A at merge-enablement); Stage 2 chooses among survivors by committee judgment (β vs α here, with β preferred).

### What would flip the recommendation

The recommendation flips to **α** if any of the following emerges:

- **Operationally-expensive read-time-joins.** If projection-layer joins for the attachment-record path prove costly at scale (e.g., the merged-formation-with-attachments view becomes a hot read path), α's zero-join model is preferable. Finding 3's reframing relies on projection-layer joins being a manageable cost; if not, α wins.
- **Strong committee preference for type-system explicitness over uniformity.** If the per-cell field naming under α is considered a feature (explicit per-cell semantic visibility at the proto layer) rather than a cost, α's structural explicitness wins. This is a values choice; both readings are defensible.
- **`surface_type` discriminator concern.** If the committee judges β's `surface_type` discriminator as approaching the §2.2 first forbidden anti-pattern too closely (despite the intra-Cat-I scope), α's typed-field-per-cell shape avoids the discriminator entirely.

The recommendation flips to **γ** only under a Charter §4 reinterpretation that admits consumer-interpretation as sufficient structural commitment. Per the merge-enablement A precedent, this reinterpretation is structurally unlikely.

### Methodological observation

This is the **third** Ontology RFC discussion to converge by two-stage filter (Stage 1 discipline filter → Stage 2 committee-judgment filter), and the **second** to use the merge-enablement-A precedent for Stage 1 (γ here disqualified by the same shape that disqualified merge-enablement A). The "candidate that makes no structural commitment is §4-disqualified" pattern is now established as a recurring disqualification shape.

Future implementation-RFCs with structural commitments at stake may use the same Stage 1 filter when one candidate is "let consumers handle it" — the candidate is structurally disqualified by §4 criterion 1 unless the committee reinterprets the discipline.
