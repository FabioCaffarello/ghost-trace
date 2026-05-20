# Canonical Corpus

Golden-file corpus per [`docs/architecture/canonical-serialization-contract.md`](../../../../../../docs/architecture/canonical-serialization-contract.md) §CI Golden-File Gate.

## Status

Active. CI golden-file gate operationalized as of [`decision-log §0031`](../../../../../../docs/charter/decision-log.md). Coverage extended to a second Cat I type ([`NetworkEvent`](../../../../../../schemas/events/v1/network_event.proto)) at [`decision-log §0042`](../../../../../../docs/charter/decision-log.md); extended again to the first Category II operational construct ([`OperationalSession`](../../../../../../schemas/events/v1/operational_session.proto)) at [`decision-log §0043`](../../../../../../docs/charter/decision-log.md); extended again to the first Category III lifecycle event ([`BehavioralClusterFormation`](../../../../../../schemas/events/v1/behavioral_cluster_formation.proto)) at [`decision-log §0045`](../../../../../../docs/charter/decision-log.md); extended again to the second Category III lifecycle event ([`BehavioralClusterPromotion`](../../../../../../schemas/events/v1/behavioral_cluster_promotion.proto)) at [`decision-log §0046`](../../../../../../docs/charter/decision-log.md); extended again to the third Category III lifecycle event ([`BehavioralClusterDemotion`](../../../../../../schemas/events/v1/behavioral_cluster_demotion.proto)) at [`decision-log §0047`](../../../../../../docs/charter/decision-log.md); extended again to the fourth Category III lifecycle event ([`BehavioralClusterDissolution`](../../../../../../schemas/events/v1/behavioral_cluster_dissolution.proto)) at [`decision-log §0048`](../../../../../../docs/charter/decision-log.md); extended again to the fifth Category III lifecycle event ([`BehavioralClusterMerge`](../../../../../../schemas/events/v1/behavioral_cluster_merge.proto)) at [`decision-log §0049`](../../../../../../docs/charter/decision-log.md); extended again to the sixth (and final) Category III lifecycle event ([`BehavioralClusterSplit`](../../../../../../schemas/events/v1/behavioral_cluster_split.proto)) at [`decision-log §0050`](../../../../../../docs/charter/decision-log.md), completing the §2.5 lifecycle surface for the first Cat III subtype; extended again to the SECOND Cat III concrete subtype ([`AutomationGroupFormation`](../../../../../../schemas/events/v1/automation_group_formation.proto)) at [`decision-log §0056`](../../../../../../docs/charter/decision-log.md), opening the second subtype lifecycle arc; extended again to the second-subtype promotion ([`AutomationGroupPromotion`](../../../../../../schemas/events/v1/automation_group_promotion.proto)) at [`decision-log §0057`](../../../../../../docs/charter/decision-log.md); extended again to the second-subtype demotion ([`AutomationGroupDemotion`](../../../../../../schemas/events/v1/automation_group_demotion.proto)) at [`decision-log §0058`](../../../../../../docs/charter/decision-log.md); extended again to the second-subtype dissolution ([`AutomationGroupDissolution`](../../../../../../schemas/events/v1/automation_group_dissolution.proto)) at [`decision-log §0059`](../../../../../../docs/charter/decision-log.md); extended again to the second-subtype merge ([`AutomationGroupMerge`](../../../../../../schemas/events/v1/automation_group_merge.proto)) at [`decision-log §0060`](../../../../../../docs/charter/decision-log.md); extended again to the second-subtype split ([`AutomationGroupSplit`](../../../../../../schemas/events/v1/automation_group_split.proto)) at [`decision-log §0061`](../../../../../../docs/charter/decision-log.md), closing the second-subtype lifecycle surface (6 of 6 ops landed for AutomationGroup).

## Layout

Each corpus entry is a triple sharing a stem name:

- `<name>.json`  — human-readable Protobuf-canonical-JSON form. Decoded via `protojson.Unmarshal`.
- `<name>.bin`   — expected canonical Protobuf bytes (output of `canonical.Marshal`).
- `<name>.hash`  — expected BLAKE3-256 lowercase-hex digest, single line with trailing newline.

The stem-name prefix selects the Protobuf message type via the `messageFactory` map in [`../../corpus_test.go`](../../corpus_test.go); the longest matching prefix wins. Adding a new message type requires registering its factory there + authoring at least one `.json` entry.

## Current entries

| Entry | Variant | Notes |
|---|---|---|
| `declared-session-minimal` | all proto3 defaults (`{}`) | Exercises proto3 default-elision; canonical bytes are zero-length. Hash is the BLAKE3 digest of empty input (`af1349b9...`). |
| `declared-session-typical` | typical production-shaped values | Exercises all three `DeclaredSession` fields: `declared_at` non-zero int64, `actor_ref` non-empty string, `session_descriptor` non-empty bytes. |
| `ingestion-event-mtls` | mTLS-enriched ingestion event | Exercises the `IngestionEvent` enrichment with verified client identity (CN + SANs + SHA-256). |
| `network-event-minimal` | all proto3 defaults (`{}`) | Mirrors `declared-session-minimal` for the second Cat I type; canonical bytes are zero-length; hash is the BLAKE3 digest of empty input. |
| `network-event-typical` | typical collector-shaped values | Exercises all four `NetworkEvent` fields: `observed_at` non-zero int64, `actor_ref` non-empty string, `endpoint_ref` non-empty string, `event_descriptor` non-empty bytes. |
| `operational-session-minimal` | all proto3 defaults (`{}`) | First Category II entry. Mirrors the all-defaults coverage shape for the Cat II type. |
| `operational-session-padded-v1` | typical padded-v1 derivation values | Exercises all six `OperationalSession` fields under the canonical `padded-v1` operational definition; `source_event_hash` is a synthetic 32-byte pattern. |
| `behavioral-cluster-formation-minimal` | all proto3 defaults (`{}`) | First Category III lifecycle-event entry. Mirrors the all-defaults coverage shape; canonical bytes zero-length. |
| `behavioral-cluster-formation-shared-descriptor` | typical session-descriptor-shared-v1 formation | Exercises all six `BehavioralClusterFormation` fields: `pattern_signature`, `pattern_parameters`, two sorted `actor_refs`, `formation_at`, placeholder `confidence`, two sorted `source_event_hashes`. |
| `behavioral-cluster-promotion-minimal` | all proto3 defaults (`{}`) | First entry covering the second Cat III lifecycle event. Mirrors the all-defaults coverage shape; canonical bytes zero-length. |
| `behavioral-cluster-promotion-typical` | typical operator-invoked promotion values | Exercises all four `BehavioralClusterPromotion` fields: `formation_event_hash` (synthetic 32-byte pattern referencing a formation), `promoted_at`, Layer A `cadence_seconds`, free-form `reason`. |
| `behavioral-cluster-demotion-minimal` | all proto3 defaults (`{}`) | First entry covering the third Cat III lifecycle event. |
| `behavioral-cluster-demotion-typical` | typical operator-invoked demotion values | Exercises all three `BehavioralClusterDemotion` fields: `promotion_event_hash` (synthetic 32-byte pattern referencing a promotion), `demoted_at`, free-form `reason`. |
| `behavioral-cluster-dissolution-minimal` | all proto3 defaults (`{}`) | First entry covering the fourth Cat III lifecycle event. |
| `behavioral-cluster-dissolution-typical` | typical operator-invoked dissolution values | Exercises all three `BehavioralClusterDissolution` fields: `formation_event_hash` (synthetic 32-byte pattern referencing a formation), `dissolved_at`, free-form `reason`. |
| `behavioral-cluster-merge-minimal` | all proto3 defaults (`{}`) | First entry covering the fifth Cat III lifecycle event. |
| `behavioral-cluster-merge-typical` | typical within-subtype merge values | Exercises all four `BehavioralClusterMerge` fields: two ascending-sorted `antecedent_formation_event_hashes` (synthetic 32-byte patterns referencing two distinct formations), `produced_formation_event_hash` (third synthetic 32-byte pattern), `merged_at`, free-form `reason`. |
| `behavioral-cluster-split-minimal` | all proto3 defaults (`{}`) | First entry covering the sixth (and final) Cat III lifecycle event; §2.5 lifecycle surface now fully covered. |
| `behavioral-cluster-split-typical` | typical within-subtype split values | Exercises all four `BehavioralClusterSplit` fields: `antecedent_formation_event_hash` (synthetic 32-byte pattern referencing the formation being split), two ascending-sorted `successor_formation_event_hashes` (synthetic 32-byte patterns referencing two distinct successor formations), `split_at`, free-form `reason`. |
| `automation-group-formation-minimal` | all proto3 defaults (`{}`) | First entry covering the SECOND Cat III concrete subtype (per [`§0056`](../../../../../../docs/charter/decision-log.md)). Mirrors the all-defaults shape; canonical bytes zero-length. |
| `automation-group-formation-typical` | typical uniform-cadence-v1 formation | Exercises all six `AutomationGroupFormation` fields: `pattern_signature` = `uniform-cadence-v1`, `pattern_parameters` with both keys, single-actor `actor_refs`, `formation_at`, placeholder `confidence`, three sorted `source_event_hashes`. |
| `automation-group-promotion-minimal` | all proto3 defaults (`{}`) | First entry covering the second-subtype promotion lifecycle event (§0057). |
| `automation-group-promotion-typical` | typical operator-invoked promotion values | Exercises all four `AutomationGroupPromotion` fields: `formation_event_hash` (synthetic 32-byte pattern referencing an AutomationGroupFormation), `promoted_at`, Layer A `cadence_seconds`, free-form `reason`. |
| `automation-group-demotion-minimal` | all proto3 defaults (`{}`) | First entry covering the second-subtype demotion lifecycle event (§0058). |
| `automation-group-demotion-typical` | typical operator-invoked demotion values | Exercises all three `AutomationGroupDemotion` fields: `promotion_event_hash` (synthetic 32-byte pattern), `demoted_at`, free-form `reason`. |
| `automation-group-dissolution-minimal` | all proto3 defaults (`{}`) | First entry covering the second-subtype dissolution lifecycle event (§0059). |
| `automation-group-dissolution-typical` | typical operator-invoked dissolution values | Exercises all three `AutomationGroupDissolution` fields: `formation_event_hash` (synthetic 32-byte pattern), `dissolved_at`, free-form `reason`. |
| `automation-group-merge-minimal` | all proto3 defaults (`{}`) | First entry covering the second-subtype merge lifecycle event (§0060). |
| `automation-group-merge-typical` | typical within-subtype merge values | Exercises all four `AutomationGroupMerge` fields: two ascending-sorted antecedent hashes, produced formation hash, `merged_at`, free-form `reason`. |
| `automation-group-split-minimal` | all proto3 defaults (`{}`) | First entry covering the second-subtype split lifecycle event (§0061) — closes the second-subtype lifecycle surface. |
| `automation-group-split-typical` | typical within-subtype split values | Exercises all four `AutomationGroupSplit` fields: antecedent formation hash, two ascending-sorted successor hashes, `split_at`, free-form `reason`. |

## Regeneration

Per the contract, regeneration is an **explicit** operation. Run only when a [schemas-evolution event](../../../../../../docs/architecture/canonical-serialization-contract.md#schemas-evolution-events) is contemplated:

```sh
make golden-corpus     # or: go test ./internal/canonical/ -run TestCanonicalCorpus -update
```

The command rewrites every `<name>.bin` + `<name>.hash` from the current `<name>.json` source under the current canonical-serialization-contract library versions. Commit the regenerated files alongside the schemas-evolution commit that produces them, with a commit message naming the event explicitly.

Per [`docs/architecture/canonical-serialization-contract.md`](../../../../../../docs/architecture/canonical-serialization-contract.md) §Upgrade Discipline, the regeneration step is part of a five-step procedure (survey → predict → run → reconcile → commit → inform downstream). Running `-update` without the surrounding discipline defeats the gate.

## References

- [`docs/architecture/canonical-serialization-contract.md`](../../../../../../docs/architecture/canonical-serialization-contract.md) — the contract.
- [`docs/charter/decision-log.md` §0024](../../../../../../docs/charter/decision-log.md) — schemas-technology selection + AP5 mitigation (d) (the gate this corpus implements).
- [`docs/charter/decision-log.md` §0028](../../../../../../docs/charter/decision-log.md) — canonical-serialization-contract introduced.
- [`docs/charter/decision-log.md` §0031](../../../../../../docs/charter/decision-log.md) — corpus populated + CI gate operationalized.
