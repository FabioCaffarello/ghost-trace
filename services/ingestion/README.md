# Ingestion Service

## Constitutional Role

Receives observations from producers and commits them to the primary event log. The point at which the system takes responsibility for a record. After commitment, the record is governed by [Charter §2.1 Observational Integrity](../../docs/charter/constitutional-charter.md#21-observational-integrity).

## Status

Skeleton. First commit producing executable code per [`decision-log §0027`](../../docs/charter/decision-log.md) Consequences ("Service skeleton work ... proceeds under ordinary RFC/PR discipline from this point forward"). Recorded at [`decision-log §0030`](../../docs/charter/decision-log.md).

## Architecture

**Provenance**: every primary observation commits paired with an `IngestionEvent` enrichment per [`§0038`](../../docs/charter/decision-log.md). The pair captures **what** the producer reported (one of the registered Cat I primary-observation types — see §[Category I message types](#category-i-message-types) below) AND **how** the service received it (channel — `stdin`/`http`/`https`/`https+mtls` — plus, when delivered over mTLS, the verified peer-certificate's Common Name + SANs + SHA-256 fingerprint). The two events commit atomically via `substrate.AppendPair` (single SQL transaction). The pairing is by reference: each `IngestionEvent.primary_event_hash` carries the content-hash of its paired observation.

Four packages under `internal/`, each consuming a published contract:

- [`internal/canonical`](./internal/canonical) — implements [`docs/architecture/canonical-serialization-contract.md`](../../docs/architecture/canonical-serialization-contract.md). Single `Marshal` + `Hash` + `HashHex` entry points; service code MUST NOT call `proto.Marshal` directly.
- [`internal/substrate`](./internal/substrate) — implements the [`§0027`](../../docs/charter/decision-log.md) SQLite + content-addressed blob-store substrate. Single `Append` entry point + `writeMu` mutex per [`docs/architecture/concurrency-pattern.md`](../../docs/architecture/concurrency-pattern.md) §Substrate-Writer Serialization. PRAGMA `journal_mode=WAL` + `synchronous=FULL` per [`§0027`](../../docs/charter/decision-log.md) Proposal item 1. Hash-verification on every blob-read path per [`§0027`](../../docs/charter/decision-log.md) AP5.
- [`internal/ingest`](./internal/ingest) — composes `canonical` + `substrate` into a typed `Append(ctx, msg, eventTime)` boundary. The single per-process write entry point.
- [`internal/httpapi`](./internal/httpapi) — minimum-viable HTTP interface (`POST /v1/events/{type}/{type}` accepting `application/x-protobuf`; `GET /healthz`). The `{type}` segment selects a registered Cat I message type via `ingest.LookupURLPath` (per [`§0042`](../../docs/charter/decision-log.md)). Same error classification as the stdin worker: recoverable → 4xx + JSON body; unrecoverable (substrate §2.1 violations) → 500 + JSON body + signal the service-level fatal channel per [`§0032`](../../docs/charter/decision-log.md).

`main.go` orchestrates up to four goroutines via `errgroup` per [`concurrency-pattern.md`](../../docs/architecture/concurrency-pattern.md): the stdin worker (always), the HTTP server (when `--http` is set), an HTTP-graceful-shutdown coordinator, and a fatal-coordinator that propagates HTTP-side unrecoverable errors back through errgroup. Shutdown coordinated via `signal.NotifyContext`.

## `verify` CLI

Companion binary at [`cmd/verify`](./cmd/verify) that performs an up-front substrate-integrity check. Discharges the `§0033` `verify` follow-on; see [`§0039`](../../docs/charter/decision-log.md).

```sh
make verify-build                                                       # builds ./bin/verify
./bin/verify -db ./ghost-trace.db -blobs ./blobs                        # walks events table; verifies every blob
./bin/verify -db ./ghost-trace.db -blobs ./blobs -check-orphans         # also reports orphan blobs (informational)
```

Walks every events-table row in commit order, recomputes each blob's BLAKE3 hash via `substrate.ReadBlob`, surfaces hash-mismatch + missing-blob failures. With `-check-orphans` (per [`§0040`](../../docs/charter/decision-log.md)), also walks the blob-store directory + reports blobs whose content-hash does not appear in the events table — orphans are **harmless** at the substrate layer per [`§0033`](../../docs/charter/decision-log.md) (the events table is authoritative); they are reported but do NOT cause non-zero exit. Writes structured JSON to stdout + a brief human summary to stderr. Exit code: **0** on pass (including substrates with orphan blobs); **1** on any §2.1 violation (hash-mismatch or missing-blob); **2** on tool/configuration error. Intended for post-restore verification (per [`§0033` §Restoration Procedure step 3](../../docs/architecture/operational-ops.md)) and periodic substrate-integrity audits.

## `derive-operational-session` CLI

Operator-invoked tool to derive Category II [`OperationalSession`](../../schemas/events/v1/operational_session.proto) constructs from every `DeclaredSession` in the substrate under a versioned operational definition (per [`§0043`](../../docs/charter/decision-log.md) — first Cat II construct landing).

```sh
make derive-build                                                       # builds ./bin/derive-operational-session

# Run the default padded-v1 definition (pad_seconds=300)
./bin/derive-operational-session -db ./ghost-trace.db -blobs ./blobs

# Override the padding parameter — produces a NEW set of OperationalSession
# records alongside the prior derivations (versioning per entity-model.md line 45)
./bin/derive-operational-session -db ./ghost-trace.db -blobs ./blobs -pad-seconds 600
```

Walks every `DeclaredSession` row in the substrate via `substrate.WalkEvents`, applies the operational definition deterministically (per [Charter §2.2](../../docs/charter/constitutional-charter.md#22-epistemic-separation) Category II requirement), and commits each `OperationalSession` to the same events table via `substrate.Append` (acquires `writeMu` per [`concurrency-pattern.md`](../../docs/architecture/concurrency-pattern.md) §Substrate-Writer Serialization). Re-running with an identical `(definition_version, definition_parameters)` tuple is a no-op (content-hash collision → `INSERT OR IGNORE`); re-running with a NEW tuple produces NEW records and preserves the prior ones per [`entity-model.md` §Category II](../../docs/ontology/entity-model.md) line 45.

Writes structured JSON to stdout (`definition_version`, `definition_parameters`, `examined`, `newly_derived`, `already_derived`) + a brief human summary to stderr. Exit code: **0** on success (including zero-newly-derived); **2** on tool/configuration error.

Registered operational definitions:

| Version | Parameters | Boundary derivation |
|---|---|---|
| `padded-v1` | `pad_seconds=<int>` (default 300) | `operational_start_at = declared_at`; `operational_end_at = declared_at + pad_seconds`. Minimal canonical example; exercises all Cat II structural mechanisms (deterministic derivation, identity-via-version, provenance, boundary divergence). |

Adding a new operational definition registers via [`internal/derivation`](./internal/derivation): implement `OperationalDefinition` (Version, Parameters, Derive) and wire it into `cmd/derive-operational-session/main.go`'s `resolveDefinition`.

## `orphan-cleanup` CLI

Operator-invoked tool to delete orphan blobs (per [`§0041`](../../docs/charter/decision-log.md)). Per [`§0033` Anti-Patterns](../../docs/architecture/operational-ops.md), orphan deletion MUST be operator-invoked with explicit confirmation; this tool implements that discipline.

```sh
make orphan-cleanup-build                                                    # builds ./bin/orphan-cleanup

# Dry-run (default; safe; reports what WOULD be deleted)
./bin/orphan-cleanup -db ./ghost-trace.db -blobs ./blobs

# Confirmed deletion (requires both -dry-run=false AND -confirm)
./bin/orphan-cleanup -db ./ghost-trace.db -blobs ./blobs -dry-run=false -confirm

# With exclusion list (one hex hash per line; # comments allowed)
./bin/orphan-cleanup -dry-run=false -confirm -exclude ./preserve-these.txt

# Override default safety belts
./bin/orphan-cleanup -dry-run=false -confirm -keep-newer-than 1h -max-deletions 100
```

Safety belts (each independently configurable):

| Belt | Default | Override |
|---|---|---|
| Dry-run by default | `-dry-run=true` | explicit `-dry-run=false` required |
| Explicit confirmation | required when not dry-run | `-confirm` (or tool refuses with exit 2) |
| Age floor | `-keep-newer-than 24h` | `-keep-newer-than 0` disables; any duration |
| Per-invocation cap | `-max-deletions 1000` | `-max-deletions 0` disables |
| Exclusion list | none | `-exclude <path>` (one hash hex per line; `#` comments) |

Writes structured JSON to stdout (records of what was examined / preserved / deleted) + a brief human summary to stderr. Exit code: **0** on success (including dry-run); **2** on tool / configuration error (e.g. missing `-confirm` when not dry-run).

## Category I message types

Registered Cat I primary-observation types accepted by the ingestion pipeline (per [`§0042`](../../docs/charter/decision-log.md) — second Cat I type added). The dispatch registry at [`internal/ingest/dispatch.go`](./internal/ingest/dispatch.go) binds each type to its HTTP URL path + stdin envelope type identifier + event-time accessor. Adding a new type extends the registry, the corpus factory, the Makefile generate target, and lands a `.proto` under [`schemas/events/v1/`](../../schemas/events/v1/).

| Type | HTTP path | stdin `type` identifier | Event-time field | Producer class |
|---|---|---|---|---|
| `DeclaredSession` | `/v1/events/declared-session` | `declared_session` | `declared_at` | Client SDK (session-end report) |
| `NetworkEvent` | `/v1/events/network-event` | `network_event` | `observed_at` | Infrastructure collector (flow record / IDS event / packet summary) |

The substrate stores all types in the same events table with the `message_type` column carrying the Protobuf descriptor's full name (e.g. `ghosttrace.events.v1.DeclaredSession`, `ghosttrace.events.v1.NetworkEvent`). Verify + orphan-cleanup are type-agnostic and operate over heterogeneous-type substrates without change.

Category II `OperationalSession` records also live in the same events table — substrate immutability (Charter §2.1) applies to Cat II per [`entity-model.md` §Category II](../../docs/ontology/entity-model.md). Derivation is operator-invoked via [`cmd/derive-operational-session`](#derive-operational-session-cli); see that section for the registered operational definitions.

## Build Sequence

Generated Protobuf bindings are NOT committed per [`§0024`](../../docs/charter/decision-log.md) AP3 ("Generated code is build output, not source"). First build sequence:

```sh
make tools     # installs protoc-gen-go locally (per internal/tools/tools.go pin)
make generate  # runs protoc against ../../schemas/events/v1/*.proto (every registered Cat I type + IngestionEvent)
make test      # go test -race ./...
make build     # go build -trimpath -o bin/ingestion .
```

Subsequent builds skip `make tools` unless `protoc-gen-go` is missing or out of date.

## Run

**Stdin/stdout (default):**

```sh
# Wire shape per line: {"type":"<type-id>","payload_b64":"<base64-proto>"}
echo '{"type":"declared_session","payload_b64":"<base64-Proto-DeclaredSession>"}' | ./bin/ingestion -db ./ghost-trace.db -blobs ./blobs
echo '{"type":"network_event","payload_b64":"<base64-Proto-NetworkEvent>"}'      | ./bin/ingestion -db ./ghost-trace.db -blobs ./blobs
```

Each input line is a JSON envelope where `type` selects a registered Cat I type (see §[Category I message types](#category-i-message-types) for the registered identifiers) and `payload_b64` is the base64-encoded canonical-Protobuf payload. Each input line produces one output line (JSON `confirmation` on success; `ingestError` on recoverable failure — envelope decode, unknown type, base64 decode, proto unmarshal). Unrecoverable substrate violations terminate the worker.

**HTTP (opt-in via `--http`):**

```sh
./bin/ingestion -db ./ghost-trace.db -blobs ./blobs -http :8080
```

Producers POST to `http://localhost:8080/v1/events/<type>` with `Content-Type: application/x-protobuf` and a Protobuf-marshaled body of the corresponding type:

```sh
curl -sS -X POST --data-binary @declared-session.bin \
  -H 'Content-Type: application/x-protobuf' \
  http://localhost:8080/v1/events/declared-session
curl -sS -X POST --data-binary @network-event.bin \
  -H 'Content-Type: application/x-protobuf' \
  http://localhost:8080/v1/events/network-event
```

The response is `200 OK` + JSON `confirmation` on success; `400 Bad Request` + JSON `ingestError` on recoverable input failures; `404 Not Found` + JSON `ingestError` (with the known-types list) when `<type>` is unregistered or the path is the bare `/v1/events`; `500 Internal Server Error` + JSON `ingestError` on unrecoverable substrate violations (which also trigger service shutdown). `GET /healthz` returns `200 OK` + `{"status":"ok"}`. The stdin worker runs simultaneously; both channels share the same single-writer mutex per [`concurrency-pattern.md`](../../docs/architecture/concurrency-pattern.md) §Substrate-Writer Serialization.

**HTTP with TLS termination (opt-in):**

```sh
./bin/ingestion -http :8443 \
  -http-tls-cert /etc/ghost-trace/cert.pem \
  -http-tls-key /etc/ghost-trace/key.pem
```

`--http-tls-cert` and `--http-tls-key` MUST both be set or both be empty. When set, the service serves HTTPS via `crypto/tls` with `MinVersion: TLS 1.2` (TLS 1.0/1.1 deprecated per RFC 8996). Both files are stat-checked at startup so misconfiguration fails fast rather than at first connection. ALPN auto-negotiates HTTP/2 when the client supports it (Go stdlib default). Bearer-token auth (next section) composes with TLS: the same `--http-auth-token-file` works under HTTPS. Cert reload on rotation requires a restart at inception phase; an online-reload follow-on is named in [`§0036`](../../docs/charter/decision-log.md) Out of Scope.

**HTTP with mTLS (opt-in; requires TLS):**

```sh
./bin/ingestion -http :8443 \
  -http-tls-cert /etc/ghost-trace/server-cert.pem \
  -http-tls-key /etc/ghost-trace/server-key.pem \
  -http-tls-client-ca /etc/ghost-trace/client-ca-bundle.pem
```

`--http-tls-client-ca` enables mutual-TLS authentication: every client MUST present a certificate signed by one of the CAs in the bundle. The server verifies via `tls.RequireAndVerifyClientCert` during the TLS handshake; connections without a valid client cert are rejected at the TLS layer (before any HTTP request is processed — no 401, no response body, just connection close). mTLS provides per-producer identity (the Common Name + SANs in the client cert), useful for multi-producer deployments where bearer tokens alone are insufficient. mTLS COMPOSES with bearer-token auth: when both are configured, BOTH must pass (defense in depth) — the producer presents a valid client cert AND sends `Authorization: Bearer <token>`. The client-CA file is read + parsed at startup; misconfiguration fails fast. Per-client-cert revocation (CRL / OCSP) is not exercised at inception; revoke clients by rotating the CA bundle + restarting the service.

**HTTP with bearer-token authentication (opt-in):**

```sh
# Production: token stored in a 0600-mode file (avoids process-listing leak).
echo -n "deployment-secret-token" > /etc/ghost-trace/ingestion.token
chmod 0600 /etc/ghost-trace/ingestion.token
./bin/ingestion -http :8080 -http-auth-token-file /etc/ghost-trace/ingestion.token

# Alternative (scripting/dev only): inline token.
./bin/ingestion -http :8080 -http-auth-token "dev-secret"
```

Producers MUST send `Authorization: Bearer <token>` with every `POST /v1/events/{type}`. Missing or wrong tokens return `401 Unauthorized` + JSON `ingestError` + `WWW-Authenticate: Bearer realm="ghost-trace-ingestion"`. Token comparison uses constant-time equality (`crypto/subtle.ConstantTimeCompare`); a length-mismatch leak channel exists but is acceptable at inception per [`§0035`](../../docs/charter/decision-log.md). `/healthz` is exempt from auth (orchestrator-friendly liveness probing); unknown paths return `401` (not `404`) when auth is configured, so the path structure is not leaked. Bearer tokens transmit credentials in plaintext on the wire — production deployments SHOULD also terminate TLS via reverse proxy (or a follow-on TLS RFC).

Signals (SIGINT, SIGTERM) trigger graceful shutdown via context cancellation; in-flight HTTP requests drain up to a 10-second grace window before the server returns from `Shutdown`.

## Required Properties

Per the original constitutional placeholder ([decision-log §0022](../../docs/charter/decision-log.md) implementation pivot):

- **Idempotent commitment** — a producer retry produces no duplicate records in the events table. Enforced by `INSERT OR IGNORE` on `event_hash BLOB PRIMARY KEY` per [`§0027`](../../docs/charter/decision-log.md) AP6 + content-addressing.
- **Producer-time preservation** — `event_time` column records the producer-reported time, accessed per type via the dispatch registry (`DeclaredSession.declared_at`, `NetworkEvent.observed_at`); `committed_at` column records the system's commit time.
- **Source attribution** — `actor_ref` field per [`§0023`](../../docs/charter/decision-log.md) Q2 Identity tiers resolution (inception-phase single-tier); optional on collector-reported types where attribution may be absent at collection time.
- **Schema validation** — `canonical.Marshal` uses `AllowPartial: false` rejecting messages missing required fields; `proto.Unmarshal` rejects ill-formed wire bytes.

## Constitutional + Architecture Anchors

- [Charter §2.1 Observational Integrity](../../docs/charter/constitutional-charter.md#21-observational-integrity), [§2.2](../../docs/charter/constitutional-charter.md#22-epistemic-separation), [§2.3](../../docs/charter/constitutional-charter.md#23-provenance-integrity), [§2.5](../../docs/charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness).
- [`decision-log §0022`](../../docs/charter/decision-log.md) (implementation pivot), [`§0023`](../../docs/charter/decision-log.md) (Q2 actor_ref), [`§0024`](../../docs/charter/decision-log.md) (Protobuf proto3 + AP3/AP5/AP6), [`§0025`](../../docs/charter/decision-log.md) (Go), [`§0027`](../../docs/charter/decision-log.md) (SQLite + blob-store; AP4/AP5/AP6), [`§0028`](../../docs/charter/decision-log.md) (canonical-serialization-contract), [`§0029`](../../docs/charter/decision-log.md) (concurrency-pattern), [`§0030`](../../docs/charter/decision-log.md) (this skeleton).
- [`docs/architecture/canonical-serialization-contract.md`](../../docs/architecture/canonical-serialization-contract.md) — bit-stable marshal + hash.
- [`docs/architecture/concurrency-pattern.md`](../../docs/architecture/concurrency-pattern.md) — goroutine + channel + context + substrate-writer-serialization discipline.

## Out of Scope

Per skeleton-status discipline, the following are deferred to follow-on commits:

- ~~HTTP interface~~ **partially discharged at [`decision-log §0034`](../../docs/charter/decision-log.md).** `POST /v1/events/{type}` + `GET /healthz` implemented in [`internal/httpapi`](./internal/httpapi); opt-in via `--http :8080`. gRPC remains deferred per [`§0025`](../../docs/charter/decision-log.md) Open Questions; HTTP authentication, rate limiting, and TLS termination are out of scope (reverse-proxy concern at inception).
- **Backup/recovery automation.** Manual `.backup` + `rsync` per [`§0027`](../../docs/charter/decision-log.md) Proposal item 5; ordering matters (blob-store first, then SQLite).
- ~~Canonical-corpus population.~~ **Discharged at [`decision-log §0031`](../../docs/charter/decision-log.md).** Two corpus entries cover `DeclaredSession`; discovery-based test + `-update` regeneration via `make golden-corpus`; CI golden-file gate operational.
- ~~Unrecoverable-error shutdown escalation.~~ **Discharged at [`decision-log §0032`](../../docs/charter/decision-log.md).** `readLoop` classifies errors via `isUnrecoverable`; substrate §2.1-violation errors (`substrate.ErrHashMismatch`, `substrate.ErrBlobCollision`) terminate the worker, propagate through errgroup, and trigger `main()` to write a structured fatal record to stderr + exit non-zero. Recoverable errors (bad input) still emit per-message JSON entries to stdout and continue processing. Tested in `main_test.go` (6 tests; both paths exercised).
- ~~Multiple Category I message types.~~ **Partially discharged at [`decision-log §0042`](../../docs/charter/decision-log.md).** Second Cat I type (`NetworkEvent`) added; dispatch registry at [`internal/ingest/dispatch.go`](./internal/ingest/dispatch.go) makes the addition mechanical (see §[Category I message types](#category-i-message-types)). Additional types (fingerprint snapshots, external authoritative state changes) extend the registry as their schemas land.
