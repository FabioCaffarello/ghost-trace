# Ingestion Service

## Constitutional Role

Receives observations from producers and commits them to the primary event log. The point at which the system takes responsibility for a record. After commitment, the record is governed by [Charter §2.1 Observational Integrity](../../docs/charter/constitutional-charter.md#21-observational-integrity).

## Status

Skeleton. First commit producing executable code per [`decision-log §0027`](../../docs/charter/decision-log.md) Consequences ("Service skeleton work ... proceeds under ordinary RFC/PR discipline from this point forward"). Recorded at [`decision-log §0030`](../../docs/charter/decision-log.md).

## Architecture

Four packages under `internal/`, each consuming a published contract:

- [`internal/canonical`](./internal/canonical) — implements [`docs/architecture/canonical-serialization-contract.md`](../../docs/architecture/canonical-serialization-contract.md). Single `Marshal` + `Hash` + `HashHex` entry points; service code MUST NOT call `proto.Marshal` directly.
- [`internal/substrate`](./internal/substrate) — implements the [`§0027`](../../docs/charter/decision-log.md) SQLite + content-addressed blob-store substrate. Single `Append` entry point + `writeMu` mutex per [`docs/architecture/concurrency-pattern.md`](../../docs/architecture/concurrency-pattern.md) §Substrate-Writer Serialization. PRAGMA `journal_mode=WAL` + `synchronous=FULL` per [`§0027`](../../docs/charter/decision-log.md) Proposal item 1. Hash-verification on every blob-read path per [`§0027`](../../docs/charter/decision-log.md) AP5.
- [`internal/ingest`](./internal/ingest) — composes `canonical` + `substrate` into a typed `Append(ctx, msg, eventTime)` boundary. The single per-process write entry point.
- [`internal/httpapi`](./internal/httpapi) — minimum-viable HTTP interface (`POST /v1/events` accepting `application/x-protobuf`; `GET /healthz`). Same error classification as the stdin worker: recoverable → 4xx + JSON body; unrecoverable (substrate §2.1 violations) → 500 + JSON body + signal the service-level fatal channel per [`§0032`](../../docs/charter/decision-log.md).

`main.go` orchestrates up to four goroutines via `errgroup` per [`concurrency-pattern.md`](../../docs/architecture/concurrency-pattern.md): the stdin worker (always), the HTTP server (when `--http` is set), an HTTP-graceful-shutdown coordinator, and a fatal-coordinator that propagates HTTP-side unrecoverable errors back through errgroup. Shutdown coordinated via `signal.NotifyContext`.

## Build Sequence

Generated Protobuf bindings are NOT committed per [`§0024`](../../docs/charter/decision-log.md) AP3 ("Generated code is build output, not source"). First build sequence:

```sh
make tools     # installs protoc-gen-go locally (per internal/tools/tools.go pin)
make generate  # runs protoc against ../../schemas/events/declared_session.proto
make test      # go test -race ./...
make build     # go build -trimpath -o bin/ingestion .
```

Subsequent builds skip `make tools` unless `protoc-gen-go` is missing or out of date.

## Run

**Stdin/stdout (default):**

```sh
echo "<base64-encoded-Protobuf-DeclaredSession>" | ./bin/ingestion -db ./ghost-trace.db -blobs ./blobs
```

Each input line produces one output line (JSON object).

**HTTP (opt-in via `--http`):**

```sh
./bin/ingestion -db ./ghost-trace.db -blobs ./blobs -http :8080
```

Then producers may POST to `http://localhost:8080/v1/events` with `Content-Type: application/x-protobuf` and a Protobuf-marshaled `DeclaredSession` body. The response is `200 OK` + JSON `confirmation` on success, `400 Bad Request` + JSON `ingestError` on recoverable input failures, `500 Internal Server Error` + JSON `ingestError` on unrecoverable substrate violations (which also trigger service shutdown). `GET /healthz` returns `200 OK` + `{"status":"ok"}`. The stdin worker runs simultaneously; both channels share the same single-writer mutex per [`concurrency-pattern.md`](../../docs/architecture/concurrency-pattern.md) §Substrate-Writer Serialization.

Signals (SIGINT, SIGTERM) trigger graceful shutdown via context cancellation; in-flight HTTP requests drain up to a 10-second grace window before the server returns from `Shutdown`.

## Required Properties

Per the original constitutional placeholder ([decision-log §0022](../../docs/charter/decision-log.md) implementation pivot):

- **Idempotent commitment** — a producer retry produces no duplicate records in the events table. Enforced by `INSERT OR IGNORE` on `event_hash BLOB PRIMARY KEY` per [`§0027`](../../docs/charter/decision-log.md) AP6 + content-addressing.
- **Producer-time preservation** — `event_time` column records the producer's `DeclaredSession.declared_at`; `committed_at` column records the system's commit time.
- **Source attribution** — `actor_ref` field per [`§0023`](../../docs/charter/decision-log.md) Q2 Identity tiers resolution (inception-phase single-tier).
- **Schema validation** — `canonical.Marshal` uses `AllowPartial: false` rejecting messages missing required fields; `proto.Unmarshal` rejects ill-formed wire bytes.

## Constitutional + Architecture Anchors

- [Charter §2.1 Observational Integrity](../../docs/charter/constitutional-charter.md#21-observational-integrity), [§2.2](../../docs/charter/constitutional-charter.md#22-epistemic-separation), [§2.3](../../docs/charter/constitutional-charter.md#23-provenance-integrity), [§2.5](../../docs/charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness).
- [`decision-log §0022`](../../docs/charter/decision-log.md) (implementation pivot), [`§0023`](../../docs/charter/decision-log.md) (Q2 actor_ref), [`§0024`](../../docs/charter/decision-log.md) (Protobuf proto3 + AP3/AP5/AP6), [`§0025`](../../docs/charter/decision-log.md) (Go), [`§0027`](../../docs/charter/decision-log.md) (SQLite + blob-store; AP4/AP5/AP6), [`§0028`](../../docs/charter/decision-log.md) (canonical-serialization-contract), [`§0029`](../../docs/charter/decision-log.md) (concurrency-pattern), [`§0030`](../../docs/charter/decision-log.md) (this skeleton).
- [`docs/architecture/canonical-serialization-contract.md`](../../docs/architecture/canonical-serialization-contract.md) — bit-stable marshal + hash.
- [`docs/architecture/concurrency-pattern.md`](../../docs/architecture/concurrency-pattern.md) — goroutine + channel + context + substrate-writer-serialization discipline.

## Out of Scope

Per skeleton-status discipline, the following are deferred to follow-on commits:

- ~~HTTP interface~~ **partially discharged at [`decision-log §0034`](../../docs/charter/decision-log.md).** `POST /v1/events` + `GET /healthz` implemented in [`internal/httpapi`](./internal/httpapi); opt-in via `--http :8080`. gRPC remains deferred per [`§0025`](../../docs/charter/decision-log.md) Open Questions; HTTP authentication, rate limiting, and TLS termination are out of scope (reverse-proxy concern at inception).
- **Backup/recovery automation.** Manual `.backup` + `rsync` per [`§0027`](../../docs/charter/decision-log.md) Proposal item 5; ordering matters (blob-store first, then SQLite).
- ~~Canonical-corpus population.~~ **Discharged at [`decision-log §0031`](../../docs/charter/decision-log.md).** Two corpus entries cover `DeclaredSession`; discovery-based test + `-update` regeneration via `make golden-corpus`; CI golden-file gate operational.
- ~~Unrecoverable-error shutdown escalation.~~ **Discharged at [`decision-log §0032`](../../docs/charter/decision-log.md).** `readLoop` classifies errors via `isUnrecoverable`; substrate §2.1-violation errors (`substrate.ErrHashMismatch`, `substrate.ErrBlobCollision`) terminate the worker, propagate through errgroup, and trigger `main()` to write a structured fatal record to stderr + exit non-zero. Recoverable errors (bad input) still emit per-message JSON entries to stdout and continue processing. Tested in `main_test.go` (6 tests; both paths exercised).
- **Multiple Category I message types.** Only `DeclaredSession` defined initially; additional types (network-level events, fingerprint snapshots) added under ordinary RFC/PR discipline as their schemas land.
