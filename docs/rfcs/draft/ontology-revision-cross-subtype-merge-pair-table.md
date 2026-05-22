# RFC — Cross-subtype merge + split pair-table contents (§0122 + §0124 follow-on)

- **Status:** discussion
- **Authors:** Ghost Trace committee
- **Date:** 2026-05-22 (placeholder opened); 2026-05-22 (substantive deliberation opened)
- **Type:** ontology-revision
- **Affects:** [`docs/ontology/entity-model.md`](../../ontology/entity-model.md) §Cross-subtype operations (the §0122-deferred per-pair canonical-merge-target table + the §0124-deferred per-source split permitted-target-set table; co-located per the merge typing RFC §Combined-table option).

> RFCs in Ghost Trace are proposals subject to constitutional review. The [Constitutional Charter](../../charter/constitutional-charter.md) is the authoritative reference. RFCs that conflict with the Charter must either be rejected or accompanied by a corresponding charter amendment (see [`../../charter/amendments.md`](../../charter/amendments.md)).

## Status note

This RFC was opened as a placeholder per [`§0011`](../../charter/decision-log.md) Layer-B precedent at the §0122 + §0124 resolutions, then promoted to **active discussion phase** on 2026-05-22 per committee direction. Substantive per-cell deliberation is below; the RFC does NOT pick targets — it surfaces candidates + structural arguments for committee resolution.

## Summary

[`decision-log §0122`](../../charter/decision-log.md) resolved cross-subtype merge typing as Candidate γ (per-pair canonical-merge typing) at the FORM level; [`§0124`](../../charter/decision-log.md) extended this to cross-subtype split via symmetric resolution (γ' per-source permitted-target-set table). This RFC carries the substantive per-cell deliberation for both tables, co-located per the merge typing RFC §Combined-table option to preserve §0050 inverse-symmetry by construction.

Two table surfaces:

- **Merge canonical-target table.** 6 unordered pair cells (`{X, Y} → Z` where `Z ∈ {BC, AG, CH, CR}`).
- **Split permitted-target-set table.** 4 per-source rows (for each source subtype X, the set of permitted target multisets `{Y₁, Y₂, ...}` that an X may split into across successors).

The two tables interact under §0050 inverse-symmetry: a `{X, Y} → Z` merge cell implies a `Z → {X, Y}` split permission (and vice versa). Committee deliberation should preserve this interaction.

## Constitutional Review

(Inherited from the placeholder; unchanged at promotion.)

- **Q1:** None — pair-table is entity-model-level structural rule, not Charter invariant.
- **Q2:** No new canonical vocabulary at Charter level.
- **Q3:** Q2-A.2 follow-on; does not resolve any of the five canonical OQs.
- **Q4:** No Charter amendment required.
- **Q5:** No new invariant.
- **Q6:** Different per-cell choices produce different operator-visible behavior; cells are structural commitments, not cosmetic.

## Subtype semantic surfaces

To defend each pair-table cell, the per-subtype structural surfaces must be in view:

| Subtype | Semantic claim | Structural surface |
|---|---|---|
| **`BehavioralCluster` (BC)** | Set of actors operated by common entity (shared operatorship) | actor membership list + pattern-signature reference (operatorship-pattern) |
| **`AutomationGroup` (AG)** | Set of actors whose behavior matches automation signature (operation character) | actor membership list + pattern-signature reference (automation-signature) |
| **`CoordinationRing` (CR)** | Set of actors with patterns of interaction suggesting coordination (relational structure) | pairwise actor-relationship records + temporal alignment references |
| **`CampaignHypothesis` (CH)** | Set of events whose patterns suggest membership in unified operation (event-centric) | event membership list + thematic-coherence reference + (derived) actor-set via event-actor index |

Structural observations:

- **BC + AG share the actor-membership surface** (both are actor-set hypotheses with a pattern-signature). They differ in WHAT the signature represents (operatorship vs automation).
- **CR's pairwise surface is richer than BC/AG's flat actor-set** (a pairwise relationship implies set membership; the converse is not true).
- **CH's event-set surface is richer than the other three's actor-set** (events imply actors via the §0023 actor_ref; the converse is not true — an actor doesn't imply a specific event).
- **CR and CH represent orthogonal richness dimensions** — CR adds relational structure to actors; CH adds event-centric framing. Neither subsumes the other structurally.

These observations are the basis for per-cell defense in §Merge per-cell analysis below.

## Merge per-cell analysis

For each of the 6 unordered pair cells, this section presents the candidate targets, structural arguments, and the cell's open status. The RFC does NOT pick; the candidates are the deliberation surface.

### Cell 1 — `{BC, AG}`

**Candidates:** AG (strict-claim absorption), BC (operatorship surface preserved).

**Structural argument for `→ AG`.** Both BC and AG share the actor-membership surface; both have a pattern-signature reference. The difference is in the signature's content (operatorship vs automation). An "automated operation" IS a kind of shared operatorship — the automation IS the operator. AG is therefore a structurally-stricter claim than BC; the merge `{BC, AG}` represents convergent evidence that the BC's shared operator is specifically the automation pattern. The stricter claim absorbs the weaker.

**Structural argument for `→ BC`.** BC's operatorship-pattern signature can accept "automation-signature" as a specialization of operatorship; under this framing, BC is the broader category and AG is a refinement within it. The merge produces BC because the broader category subsumes the narrower.

**Recommendation:** `→ AG` is the leading candidate — the stricter-claim-absorbs-weaker framing aligns with §4 falsifiability (the AG target is more constrained and therefore more falsifiable). `→ BC` is the conservative alternative if the committee weighs hierarchy-by-broadness over hierarchy-by-specificity.

**Open status:** open.

### Cell 2 — `{BC, CR}`

**Candidates:** CR (richer relational structure), BC (simpler actor-set baseline).

**Structural argument for `→ CR`.** BC's surface is a flat actor-set; CR's surface is pairwise actor-relationships + temporal alignment. CR's surface STRUCTURALLY ACCOMMODATES BC's claim (a coordination ring's participants form a set; the BC's "common operator" inference attaches as a property of the relationship graph). The converse is not true — BC's flat actor-set cannot carry CR's pairwise structure without surface extension. The richer surface wins.

**Structural argument for `→ BC`.** A merge that produces BC discards the pairwise-relationship structure as an inference artifact, preserving only the actor-set + the shared-operator claim. Under this framing, the BC is the "core finding" and the CR is an investigative path that led to it. Useful when the relationship structure is not load-bearing for downstream use.

**Recommendation:** `→ CR` is the leading candidate — preserves more structural information; aligns with the "merge produces the richer record" intuition. `→ BC` is the alternative if the committee weighs flat-set simplicity over relational richness.

**Open status:** open.

### Cell 3 — `{BC, CH}`

**Candidates:** CH (event-centric richer than actor-centric), BC (actor-centric simpler).

**Structural argument for `→ CH`.** CH's event-set surface implies an actor-set (every event carries `actor_ref`); the BC's actor-membership claim attaches as a property of the event-set's derived actor index. CH structurally accommodates BC. The converse is not true — BC's flat actor-set does not imply an event-set.

**Structural argument for `→ BC`.** A merge that produces BC discards the event-centric framing, preserving only the actor-set + shared-operator claim. Under this framing, the campaign events are evidence FOR the BC, not the merged record's content.

**Recommendation:** `→ CH` is the leading candidate per the same "richer surface wins" framing as Cell 2.

**Open status:** open.

### Cell 4 — `{AG, CR}` (GENUINELY AMBIGUOUS)

**Candidates:** AG (automation-signature preserved), CR (relational structure preserved), neither-strictly-dominates.

**Structural argument for `→ AG`.** AG's automation-signature is the load-bearing structural commitment of the merge (automation evidence is what distinguishes AG from a generic BC). CR's pairwise-relationship surface can be retained as an attached field (`pairwise_relationships`) on the AG, preserving information. The automation-signature is structurally weakened if the target is CR (CR's surface does not natively carry a signature reference).

**Structural argument for `→ CR`.** CR's pairwise-relationship surface is structurally richer than AG's flat actor-set. The automation-signature can be retained as an attached field (`automation_signature_ref`) on the CR. The relational structure is structurally weakened if the target is AG.

**Net.** **Neither candidate strictly dominates.** Both load-bearing surfaces require extension fields under the non-canonical target. This is the cell where the merge typing RFC's "subtype-specific surface conflicts" concern surfaces most acutely.

**Recommendation:** GENUINELY OPEN. The committee may resolve either way, OR may decide that {AG, CR} represents a phenomenon that warrants a third concrete subtype (which would partially reverse [`§0122`](../../charter/decision-log.md) γ — see §Cross-cell coherence below).

**Open status:** open with explicit ambiguity.

### Cell 5 — `{AG, CH}`

**Candidates:** CH (event-centric richer), AG (automation-signature preserved).

**Structural argument for `→ CH`.** CH's event-set implies an actor-set; AG's automation-signature attaches as a property of the actor-set derived from events (`actor_signature_attachments` field). CH accommodates AG without losing the signature reference.

**Structural argument for `→ AG`.** AG's automation-signature is the distinguishing claim; CH's event-set is evidence for the signature, not the merged record's content. Under this framing, CH is investigative and AG is the finding.

**Recommendation:** `→ CH` is the leading candidate per the "richer surface wins" pattern. Less ambiguous than Cell 4 because AG's surface is simpler than CR's.

**Open status:** open.

### Cell 6 — `{CR, CH}`

**Candidates:** CH (event-centric richer; pairwise derivable from event co-occurrence), CR (pairwise preserved as primary).

**Structural argument for `→ CH`.** CH's event-set implies actor co-occurrence; pairwise relationships between actors are derivable from "which actors co-occur in which events". CR's pairwise structure is a DERIVED projection over the event-set under this framing — the CH is the underlying structure. The converse is not true — pairwise relationships do not imply specific events.

**Structural argument for `→ CR`.** The pairwise actor-relationship is structurally specific (it asserts WHICH actors relate to WHICH others); the event-set is more abstract. Under this framing, CR is the specific finding and CH is the broader frame.

**Recommendation:** `→ CH` is the leading candidate per "richer surface wins". This is the strongest candidate-recommendation in the table because the event-actor derivation is direct.

**Open status:** open.

### Aggregated leading recommendations

If the leading candidate in each cell is adopted:

| Cell | Leading target | Conservative alternative | Ambiguity |
|---|---|---|---|
| `{BC, AG}` | AG | BC | low |
| `{BC, CR}` | CR | BC | low |
| `{BC, CH}` | CH | BC | low |
| `{AG, CR}` | CR | AG | **HIGH** |
| `{AG, CH}` | CH | AG | low |
| `{CR, CH}` | CH | CR | very low |

This is the same shape as the merge typing RFC's illustrative table, with one important difference: Cell 4 is flagged as genuinely ambiguous and may warrant a different resolution (including reversing the §0122 γ commitment for that pair specifically).

## Split permitted-target-set analysis (§0124)

Under §0124 split γ' symmetric resolution, the per-source-subtype permitted-target-set table records which multiset-partitions a source X may split into across N successors (N ≥ 2 per §0050).

Symmetric inheritance from merge cells (§0050 inverse-of-merge): a merge cell `{X, Y} → Z` implies the inverse split `Z → {X, Y}` is permitted. Under the leading recommendations above:

- AG → permits {BC, AG} (inverse of Cell 1)
- CR → permits {BC, CR}, {AG, CR} (inverses of Cells 2 + 4 if Cell 4 → CR)
- CH → permits {BC, CH}, {AG, CH}, {CR, CH} (inverses of Cells 3 + 5 + 6)
- BC → permits no cross-subtype split (BC is not the target of any merge cell under the leading recommendations)

**Observation.** Under the leading recommendations, BC is structurally "terminal" for split (no cross-subtype split produces a BC successor). This is a CONSEQUENCE, not a separately-defended commitment — the symmetric inheritance forces it.

**Open question:** is BC-as-split-target-terminal a desired structural commitment? If a BC may legitimately split into {BC, AG} (recognizing a sub-pattern of automation within a BC), the merge inverse {BC, AG} → BC would need to be the Cell 1 target — reversing the leading recommendation for Cell 1.

This is the kind of cross-cell coherence question §Cross-cell coherence (below) raises.

## Cross-cell coherence

Several structural patterns emerge if the leading recommendations are adopted as a whole:

1. **Hierarchy of richness.** BC < AG ≈ BC + signature; CR ≈ BC + pairwise; CH ≈ events + derived actors. Cells follow "richer wins" except Cell 4 (where AG and CR are non-comparable). The hierarchy is partial, not total.

2. **CH is a sink.** Three of six cells (Cells 3, 5, 6) point to CH. CH is the "most-receiving" subtype under the leading recommendations. Combined with the split inverse: CH is the most "outgoing" (most cross-subtype splits produce CH).

3. **BC is terminal.** Per §Split permitted-target-set analysis above, BC is not the target of any cell (under leading recommendations); thus BC cross-subtype split is structurally absent.

4. **Cell 4's ambiguity is the table's coherence anchor.** Resolving Cell 4 → CR aligns with the "pairwise structure preserved" pattern of Cells 2 + 6; resolving Cell 4 → AG breaks the pattern but preserves the automation-signature surface. The committee's resolution of Cell 4 determines whether the table has a single coherent shape (Cell 4 → CR) or a "mostly-coherent-with-Cell-4-exception" shape.

**Coherence question:** should the committee resolve all cells in a single deliberation (preserving cross-cell coherence) OR cell-by-cell (allowing per-cell judgment to override coherence)? Recorded for deliberation; no recommendation here.

## Falsifiability per cell

Per [`falsifiability-check` §1.3](../../../.claude/skills/epistemic/falsifiability-check/SKILL.md), each cell's resolution must operationalize to a type-system check:

- **All cells under leading recommendations** pass §1.3: the helper-layer can mechanically check that a cross-subtype merge of `{X, Y}` produces a record of the table-declared type Z. Mismatch is detectable structurally.
- **Cell 4 (ambiguous)** passes §1.3 under EITHER candidate; the choice is not gated on falsifiability but on which surface-extension cost the committee is willing to absorb.

§1.4 (non-circularity): no cell is defended in terms of another cell; cross-cell coherence patterns are observed, not used as primary defense. Clean.

## Open Questions

This RFC explicitly defers:

- **Cell 4 ambiguity resolution.** The committee may adopt CR, AG, or a third option (e.g., introducing a new typed subtype that combines automation + coordination — would partially reverse §0122 γ for this cell). Recorded as the table's primary open question.
- **BC-as-split-target-terminal.** Whether this is a deliberate structural commitment (BC's flat actor-set is "weaker than" the other three; BC cross-subtype split is structurally absent) OR a consequence to be revisited. Linked to Cell 1 resolution.
- **Per-cell defense vs aggregate-table defense.** Whether the committee resolves cells independently or as a coherent table. The cross-cell coherence section raises this.
- **Cross-cell exception cells.** If Cell 4 resolves contra the leading recommendation, are there other cells where the operationally-stronger argument differs from the structural-richness argument? Recorded.

## Anti-Patterns to Avoid

- **Adopting the illustrative table by default.** The §Candidate γ illustrative table at [`ontology-revision-cross-subtype-merge-typing.md`](./ontology-revision-cross-subtype-merge-typing.md) coincides with the leading recommendations above. This is NOT coincidence — both reflect the "richer surface wins" pattern. But the substantive per-cell defense above is what the committee evaluates, not the illustrative table.
- **Resolving cells piecemeal in decision-log without cross-cell coherence review.** Per §Cross-cell coherence, the cells interact. Piecemeal resolution risks structural incoherence.
- **Treating Cell 4's ambiguity as a "tiebreaker" question.** Cell 4 is GENUINELY structural-ambiguous (neither candidate dominates); arbitrary tiebreaker resolution would be the ontology-keeper failure mode. Per `falsifiability-check` §1.4, the resolution should defend WHY one surface-extension cost is preferable.

## Migration and Backward Compatibility

No historical cross-subtype merge records exist; the pair-table is forward-looking. Per [`§0122`](../../charter/decision-log.md) Consequences, once a cell's records exist, supersession is the only path per §2.1 substrate-immutability.

Cross-cell defense lock-in: the committee may revise cells later only via supersession of prior cross-subtype-merge records produced under the prior cell value. The first cell that produces records becomes the hardest-to-revise.

The [Phase 3 / Phase 4 replay contracts](../../architecture/replay-model.md) require that any cross-subtype merge record is replayable. All leading-recommendation choices satisfy this (the produced subtype's existing replay handler operates on the merged record). Cell 4 → CR or → AG both satisfy; the replay handler attached fields (per the structural arguments above) become the load-bearing surfaces.

## References

- [`docs/rfcs/draft/ontology-revision-cross-subtype-merge-typing.md`](./ontology-revision-cross-subtype-merge-typing.md) — Status: accepted per §0122; γ form adopted.
- [`docs/rfcs/draft/ontology-revision-cross-subtype-merge-enablement.md`](./ontology-revision-cross-subtype-merge-enablement.md) — Status: accepted per §0123; B+D enablement.
- [`docs/rfcs/draft/ontology-revision-cross-subtype-split.md`](./ontology-revision-cross-subtype-split.md) — Status: accepted per §0124; γ' + B'+D' symmetric.
- [`docs/rfcs/discussion/cross-subtype-merge-typing-evidence.md`](../discussion/cross-subtype-merge-typing-evidence.md) — discussion-phase evidence for merge typing.
- [`docs/rfcs/discussion/cross-subtype-merge-enablement-evidence.md`](../discussion/cross-subtype-merge-enablement-evidence.md) — discussion-phase evidence for merge enablement.
- [`docs/rfcs/discussion/cross-subtype-split-evidence.md`](../discussion/cross-subtype-split-evidence.md) — discussion-phase evidence for split (convergence-by-inheritance).
- [`docs/charter/decision-log.md` §0122](../../charter/decision-log.md) — merge typing γ resolution; this RFC carries the deferred table contents.
- [`docs/charter/decision-log.md` §0123](../../charter/decision-log.md) — merge enablement B+D resolution.
- [`docs/charter/decision-log.md` §0124](../../charter/decision-log.md) — split symmetric γ' + B'+D' resolution; extends this RFC's scope to cover split permitted-target sets.
- [`docs/charter/decision-log.md` §0011](../../charter/decision-log.md) — Layer B placeholder precedent for form-adopt + parameters-defer.
- [`docs/charter/decision-log.md` §0021](../../charter/decision-log.md) — substrate-time-generation pattern; pair-table is a versioned operational-definition constant.
- [`docs/charter/decision-log.md` §0049](../../charter/decision-log.md) — Option B merge-as-separately-committed-formation + symmetric-antecedent set-shape.
- [`docs/ontology/entity-model.md` §Cross-subtype operations](../../ontology/entity-model.md) — anchor section.

## Decision Record

Pending. The substantive discussion phase is now open per committee direction. Resolution will record per-cell targets + split permitted-target-set table + an explicit position on the Cell 4 ambiguity + the BC-as-split-target-terminal question.
