# 0005 — The decision endpoints are a shared module, handlers included

**Status:** accepted · **Date:** 2026-08-04 · **Milestone:** PR-2.3d

## Context

Two services serve `/v1/decisions` and `/v1/outcomes` during the split.
The collector answers from the session state it holds in memory; the
decision engine answers from the snapshots that state is published as
(ADR-0004). They are meant to agree, and PR-2.3b's shadow test exists to
check that they do.

The obvious split — share the use cases, let each service write its own
HTTP layer — is not enough. What a caller observes is not only the
judgement: it is the status codes, the authentication, the rounding, and
every field name on the way out. Sharing half the contract and
reimplementing the other half is how this repository has already been
bitten twice. `libs/snapshot` exists because two mappings written apart
satisfied their own tests and disagreed with each other. `libs/wire`
exists because a server that tolerates unknown fields by design cannot
fail when two definitions drift.

## Decision

**`libs/decision` owns those two endpoints end to end** — the use cases,
the durable evaluation mapping, and the HTTP handlers — and a host
supplies only where session state comes from.

The port is one method:

```go
Lookup(ctx, token) (Session, found bool, err error)
```

A miss is `found == false` and **not** an error, because an unknown
token is a cold start the confidence dimension already models (§7). An
error means the lookup itself failed, which the policy cannot reason
about and which must not be scored as an innocent cold start.

Two supporting modules fall out of this, each satisfying ADR-0001's bar
by naming its consumers:

- **`libs/archive`** — the archive port and its sentinel. The sentinel
  is the reason it is a module rather than an interface declared twice:
  outcomes refuse when there is nowhere durable to put a label, and
  that refusal is `errors.Is(err, ErrUnavailable)`. Two packages each
  declaring their own "archive unavailable" compare unequal, and the
  refusal silently becomes a 500 in one service and a 503 in the other.
- **`libs/id`** — identifier minting. The collector issues session
  tokens and the decision engine mints evaluation ids; an identifier
  space that is 144 bits in one process and something else in another
  is a property nobody would notice until it mattered.

## Consequences

- ADR-0001's standing example is discharged. That ADR named
  `internal/app/protomap.go` as the thing that *stays* inside the
  application "until a second consumer exists". The second consumer now
  exists, so the evaluation half of protomap moved and the telemetry
  half stayed. The rule was followed, not bent.

- **A duplicated feature mapping collapsed on the way.** The evaluation
  record and the session snapshot each built a `FeatureState` from
  nineteen identical assignments, in two packages, and nothing compared
  them. `TestFeatureStateProtoCoversFeatureVector` checks that every
  feature field maps to a proto field — not that both builders set it —
  so a field added to one and forgotten in the other would have passed.
  There is now one builder, `snapshot.FromState`. Had this survived,
  a decision engine judging from snapshots while the archive stored
  evaluations would have decided on one feature vector and recorded
  another.

- The collector's `api.Config` no longer carries `SecretKey`. It
  authenticates endpoints this adapter no longer serves, and a
  credential kept where nothing reads it outlives its purpose.

- The collector's HTTP suite — goldens, OpenAPI conformance, the
  contract-fixture harness — is unchanged and still passes. That is the
  evidence the move is inert: the same tests drive the same bytes,
  served from somewhere else.
