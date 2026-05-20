# Canonical Corpus

Golden-file corpus per [`docs/architecture/canonical-serialization-contract.md`](../../../../../../docs/architecture/canonical-serialization-contract.md) §CI Golden-File Gate.

## Status

Active. CI golden-file gate operationalized as of [`decision-log §0031`](../../../../../../docs/charter/decision-log.md). Coverage extended to a second Cat I type ([`NetworkEvent`](../../../../../../schemas/events/v1/network_event.proto)) at [`decision-log §0042`](../../../../../../docs/charter/decision-log.md); extended again to the first Category II operational construct ([`OperationalSession`](../../../../../../schemas/events/v1/operational_session.proto)) at [`decision-log §0043`](../../../../../../docs/charter/decision-log.md); extended again to the first Category III lifecycle event ([`BehavioralClusterFormation`](../../../../../../schemas/events/v1/behavioral_cluster_formation.proto)) at [`decision-log §0045`](../../../../../../docs/charter/decision-log.md); extended again to the second Category III lifecycle event ([`BehavioralClusterPromotion`](../../../../../../schemas/events/v1/behavioral_cluster_promotion.proto)) at [`decision-log §0046`](../../../../../../docs/charter/decision-log.md); extended again to the third Category III lifecycle event ([`BehavioralClusterDemotion`](../../../../../../schemas/events/v1/behavioral_cluster_demotion.proto)) at [`decision-log §0047`](../../../../../../docs/charter/decision-log.md).

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
