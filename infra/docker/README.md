# Ghost Trace Sub-benchmark 1 deployment scaffold

> Per decision-log [§0205](../../docs/charter/decision-log.md). This directory holds the deployment infrastructure for running the Sub-benchmark 1 pipeline ([§0143](../../docs/charter/decision-log.md)) against an operator-supplied CIC-IDS-2017 sample. The scaffold is single-node, single-writer-serialized substrate, CLI-only by design; the HTTP observability surface is opt-in.

## Scope this revision

The scaffold currently runs three pipeline steps end-to-end:

1. **ingest** — `ingest-cic-ids` against the operator-supplied CSV (per [§0204](../../docs/charter/decision-log.md))
2. **derive-all** — `replay-all-derived-actor-attributions` (Cat II attribution per [§0168](../../docs/charter/decision-log.md))
3. **signatures** — five `find-*-candidates` invocations spanning the F3 corpus across Browser, Network, Behavioral, Event-centric (Campaign), and Interaction-centric (Coordination) modalities.

Each step's stdout + stderr are captured per RUN_ID, and a `manifest.json` is emitted at the end aggregating the §0163 signature envelopes plus run inputs + parameters + verdict.

**Lifecycle steps (form + promote + measure-chain-morphology + demotion-evaluation) are deferred to a §0206 follow-on** per [Charter §4](../../docs/charter/constitutional-charter.md#4-constitutional-design-rule) falsifiability — their bash shape depends on observing the empirical structure of signature JSON output on CIC-IDS substrate, which this scaffold's first run produces.

## Reproducibility window — honest disclosure

This scaffold reproduces a Sub-benchmark 1 run **while two external dependencies remain available**:

1. The base images pinned in `.env` (`GOLANG_IMAGE`, `DEBIAN_IMAGE`) must still be pull-able from the upstream registry by their sha256 digest. Docker Hub and other OCI registries are not obligated to retain content-addressed layers indefinitely; popular base images typically remain available for years, but this is a registry policy, not a mathematical guarantee.
2. Go module dependencies referenced by `services/ingestion/go.sum` must still be resolvable by the `proxy.golang.org` module proxy or a configured alternative.

Reproducibility is therefore **bounded in time** by registry retention. This is not a defect of the scaffold; it is a property of any container-based reproduction strategy. Operators concerned with multi-year reproducibility should mirror the base images + go module cache to durable storage of their own.

The scaffold does NOT claim:

- byte-identical output across machines (Go build embeds host CPU + OS metadata; `-trimpath` removes file paths but not all such metadata)
- deterministic substrate-row ordering across runs (substrate row order is insertion-order, and insertion-order is content-hash-ordered only within a single run)
- protection against upstream library bugs reproduced inside the digest-pinned image

## Operator quickstart

```bash
# 1. Place the CIC-IDS-2017 CSV (or any CICFlowMeter-format file) in a
#    host directory you control.
mkdir -p ./seed
cp /path/to/Monday-WorkingHours.pcap_ISCX.csv ./seed/

# 2. Copy the env template and edit it for your environment.
cp .env.example .env
$EDITOR .env          # set CIC_IDS_SEED_DIR, CIC_IDS_SAMPLE_NAME, etc.

# 3. Resolve the current base-image sha256 digests. This rewrites the
#    GOLANG_IMAGE + DEBIAN_IMAGE lines in .env with content-addressed
#    references that subsequent rebuilds will use byte-identically.
make pin-base-images

# 4. Build the runtime image (multi-stage; takes a few minutes on
#    first build, then layer-cached).
make build

# 5. Run the Sub-benchmark 1 pipeline end-to-end. The cic-ids-init
#    container runs first (compose depends_on), then the cli container
#    executes the orchestrator. Output lands in the run-artifacts
#    named volume.
make run

# 6. List runs + copy the latest manifest to the host.
make list-runs
make copy-runs           # mirrors run-artifacts volume into ./runs/
```

## Volume topology

| Named volume    | Purpose                                                  | Lifecycle                   |
|-----------------|----------------------------------------------------------|-----------------------------|
| `substrate-data`| Substrate SQLite DB + content-addressed blob store       | persisted across runs       |
| `cic-ids-input` | Operator-supplied seed, copied in by `cic-ids-init`      | refreshed on each `make run`|
| `run-artifacts` | Per-RUN_ID manifest + per-step stdout/stderr captures    | accumulates indefinitely    |

`run-artifacts` accumulates by design — auditors compare across runs, so durable accumulation is the right default per [§0205 refinement (b) cravamento](../../docs/charter/decision-log.md). Disk-growth control is operator responsibility, not automatic:

```bash
# Keep only the 10 most recent runs (default N).
make clean-old-runs

# Or specify N explicitly.
make clean-old-runs N=30
```

There is no time-based, size-based, or success-status-based automatic deletion. Cleanup is operator-invoked or it does not happen.

## Manifest structure

The manifest emitted per run conforms to `manifest_validation.json` in this directory. `manifest_version` is `f6-operational-v0.1` — an **operational placeholder** pending the F6 substrate-read contract called for at [§0143 Consequences (month 4-5 architecture artifact)](../../docs/charter/decision-log.md). Per §0205 refinement (a): when the F6 RFC lands, it may **adopt** the current operational format, **adapt** it, or **supersede** it; the operational-tier manifest does not pre-bind the contractual-tier decision.

`inputs.by_source` is prepared for multi-source ingestion ahead of need: `cic_ids` populates this scope; `synthetic` and `honeypot` reserve null slots so the §0205+ Frente 2 (synthetic) and Frente 3 (honeypot) landings do not require schema migration.

`verdict.demotion_fired` is always `null` in this scope; the §0206 follow-on populates it once lifecycle steps execute and morphology measurements drive demotion-candidacy.

`verdict.instrumented_non_firing` is `true` whenever the total candidate count across all signatures is zero — a valid epistemic outcome per [§0162 reachability-claim discipline](../../docs/charter/decision-log.md): the per-signature `EvaluationStats` sub-objects remain inspectable for diagnosis by subtype × signature × skip-reason, so non-firing is auditable, not silent.

## HTTP observability surface (optional)

```bash
make observability-up        # starts ingestion-api on 127.0.0.1:8080
curl http://127.0.0.1:8080/v1/metrics
make observability-down
```

Bound to `127.0.0.1` only — this is a single-node reproducibility scaffold, not a service exposed externally. The pipeline does not consume this surface; the daemon's role here is operator-side inspection during a long run.

## Single-writer-serialized — concurrency contract

Per [`docs/architecture/concurrency-pattern.md`](../../docs/architecture/concurrency-pattern.md) §Substrate-Writer Serialization + [§0027](../../docs/charter/decision-log.md), only one process may hold the substrate writeMu at a time. The scaffold respects this by:

- running pipeline steps **sequentially** via `docker compose run --rm cli ...`
- not starting the `ingestion-api` daemon during `make run` (it is `--profile observability` opt-in)
- placing the substrate on a **named volume** rather than a bind mount, so OS-level file-locking interactions remain predictable across compose invocations

If you start `ingestion-api` and also invoke `make run`, two processes will contend for writeMu. SQLite WAL handles this safely, but you will see contention-induced latency.

## Files in this directory

| File                          | Purpose                                                  |
|-------------------------------|----------------------------------------------------------|
| `Dockerfile`                  | multi-stage builder + runtime image                      |
| `compose.yml`                 | three services: cic-ids-init, cli, ingestion-api         |
| `run-sub-benchmark-1.sh`      | bash orchestrator for the pipeline (bundled in image)    |
| `Makefile`                    | convenience wrapper over `docker compose` invocations    |
| `pin-base-images.sh`          | resolves floating tag → sha256 digest into .env          |
| `manifest_validation.json`    | JSON Schema for the per-run manifest                     |
| `.env.example`                | environment template; copy to `.env` and edit            |
| `README.md`                   | this document                                            |

## Related decision-log entries

- [§0205](../../docs/charter/decision-log.md) — this scaffold
- [§0204](../../docs/charter/decision-log.md) — `ingest-cic-ids` CLI lift (pre-req)
- [§0143](../../docs/charter/decision-log.md) — Sub-benchmark 1 definition
- [§0162](../../docs/charter/decision-log.md) — reachability-claim discipline
- [§0163](../../docs/charter/decision-log.md) — F3 envelope JSON stable wire-contract
- [§0138](../../docs/charter/decision-log.md) — Layer B parameter inception-phase resolution
- [§0140](../../docs/charter/decision-log.md) — paired-dimension serialization contract
