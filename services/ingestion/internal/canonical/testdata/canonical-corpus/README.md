# Canonical Corpus

Golden-file corpus per [`docs/architecture/canonical-serialization-contract.md`](../../../../../../docs/architecture/canonical-serialization-contract.md) §CI Golden-File Gate.

## Format

Each canonical-form-load-bearing message type is represented by three files:

- `<name>.go`        — Go source constructing the message instance (or `.json` for declarative form once a regeneration script exists).
- `<name>.bin`       — expected canonical Protobuf bytes (golden output of `canonical.Marshal`).
- `<name>.hash`      — expected BLAKE3-256 lowercase-hex digest (golden output of `canonical.HashHex(canonical.Hash(<bin>))`).

## Status

This directory is structurally established but not yet populated. The first populated corpus entry lands when the ingestion service skeleton's CI golden-file gate is wired (follow-on commit after this skeleton). Per the canonical-serialization-contract, the CI gate is operationalized when at least one corpus entry exists; the gate is currently in pre-population scaffold state.

## Regeneration

Per the contract, regeneration is an explicit operation — never automatic. The regeneration script will be `services/ingestion/scripts/regenerate-canonical-corpus.sh` (or equivalent), added in the follow-on commit that populates the corpus.

## References

- [`docs/architecture/canonical-serialization-contract.md`](../../../../../../docs/architecture/canonical-serialization-contract.md) — the contract.
- [`docs/charter/decision-log.md` §0024](../../../../../../docs/charter/decision-log.md) — schemas-technology selection + AP5 mitigation (b) (this document) + (d) (the CI gate).
- [`docs/charter/decision-log.md` §0028](../../../../../../docs/charter/decision-log.md) — canonical-serialization-contract introduced.
