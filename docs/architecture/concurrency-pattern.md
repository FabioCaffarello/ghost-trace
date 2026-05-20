# Concurrency Pattern

**Status:** Active. Second non-scaffold architecture document. Discharges the follow-on commitment named in [`decision-log §0025`](../charter/decision-log.md) modification 5 and [`§0027`](../charter/decision-log.md) Consequences ("must land BEFORE the first service that uses more than a single goroutine; convention conflicts surfaced after multi-goroutine code exists are harder to refactor than conventions set before").

> This document specifies the concurrency discipline for Ghost Trace services. The discipline is enforceable mechanically where possible (lint rules, race detector, integration tests) and reviewable structurally where mechanical enforcement is not yet practical. Goroutine lifecycle, channel ownership, and context propagation are the three primary axes; substrate-writer serialization is the Ghost-Trace-specific fourth axis derived from [`§0027`](../charter/decision-log.md) F3 (SQLite WAL single-writer constraint).

## Constitutional Anchors

- [Charter §2.1 Observational Integrity](../charter/constitutional-charter.md#21-observational-integrity) — the substrate's append-only commitment must be preserved under concurrent service operation; concurrent writes to the events table that interleave incorrectly could violate commit ordering or produce non-reconstructible state.
- [Charter §2.5 Hypothesis Lifecycle Explicitness](../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) — hypothesis state is a projection over a lifecycle event chain; concurrent reads of the projection are safe, concurrent writes serialize through the substrate.
- [`decision-log §0025`](../charter/decision-log.md) — implementation-language selection (Go); standard-library concurrency primitives (goroutines + channels + `context.Context`) per Proposal item 6.
- [`decision-log §0027`](../charter/decision-log.md) — storage-technology selection (SQLite + blob-store); F3 single-writer constraint surfaced explicitly; substrate-writer serialization is the Ghost-Trace-specific concurrency property.
- [`decision-log §0028`](../charter/decision-log.md) — canonical-serialization-contract; concurrent marshalling against the same message instance is safe (the canonical marshal call is read-only on the message), but write-then-marshal-then-write race conditions on a shared message instance are NOT safe.

## Subordination

This document is subordinate to the [Constitutional Charter](../charter/constitutional-charter.md) and the [Ontology](../ontology/ontology.md). A conflict with either resolves by revising this document.

## Scope

This document specifies:

1. **Goroutine lifecycle.** Creation discipline, cancellation propagation, leak prevention, structured concurrency.
2. **Channel ownership.** Who creates, who closes, who reads, who writes.
3. **Context propagation.** `context.Context` discipline: which functions accept it, which derive it, how cancellation flows.
4. **Substrate-writer serialization.** Ghost-Trace-specific: how concurrent ingestion paths serialize writes to the SQLite events table per [`§0027`](../charter/decision-log.md) F3.
5. **Error propagation.** How errors from goroutines reach the caller; how unrecoverable errors trigger structured shutdown.
6. **Bounded concurrency.** Worker-pool patterns; the prohibition on unbounded goroutine spawn.

It does NOT specify:

- Specific service architectures (those are service-tier RFCs as needed).
- IPC concurrency (gRPC streams, NATS subscriptions) — service-to-service RPC is deferred per [`§0025`](../charter/decision-log.md) Open Questions; this document covers in-process concurrency only.
- Performance tuning (goroutine count, channel buffer sizing) — operational concern; covered in the operational-ops document anticipated per [`§0027`](../charter/decision-log.md) Open Questions.

## Goroutine Lifecycle

### Creation discipline

Every goroutine launched in service code MUST have:

1. **A named owner.** The function that launches the goroutine owns its lifecycle. The owner is responsible for cancellation propagation and leak prevention. "Fire-and-forget" goroutines launched at package scope or in `init()` are forbidden in service code.
2. **A cancellation path.** Every goroutine accepts a `context.Context` (directly or via a struct field that holds one); the goroutine MUST check `ctx.Done()` at every blocking operation that does not itself accept a context.
3. **A termination guarantee.** The owner can wait for the goroutine to terminate before returning. `sync.WaitGroup`, `errgroup.Group`, or equivalent structured-concurrency primitive enforces this. A goroutine that outlives its owner is a leak by construction.

### Structured concurrency via `errgroup`

The canonical pattern is `golang.org/x/sync/errgroup`:

```go
g, ctx := errgroup.WithContext(parentCtx)
g.Go(func() error { return worker1(ctx) })
g.Go(func() error { return worker2(ctx) })
if err := g.Wait(); err != nil {
    // structured shutdown; all goroutines have terminated by g.Wait() return
}
```

The `errgroup` primitive bundles four properties: cancellation derivation, error propagation (first error cancels the group), termination synchronization (`Wait` blocks until all workers return), and worker-error reporting. Service code that needs all four uses `errgroup`; service code that needs fewer may use `sync.WaitGroup` + manual context derivation, but the discipline obligation (named owner, cancellation path, termination guarantee) is identical.

### Leak prevention

A goroutine leak is a goroutine that runs after its owner returns. Three mechanical mitigations:

1. `go.uber.org/goleak` (or equivalent) in test setup — verifies no goroutines outlive the test function.
2. CI integration test runs every service's startup-then-shutdown cycle and asserts goleak-clean.
3. Race detector (`go test -race`) catches data-races introduced by leaked goroutines reading shared state after their owner has cleaned up.

## Channel Ownership

### Ownership rules

Channels in Go are unowned by language convention; service code adopts the following discipline:

1. **The creator owns the channel.** The function that calls `make(chan T)` owns the channel for its lifetime.
2. **The owner closes the channel.** Closing a channel is a write operation; only the owner closes. Reading from a closed channel returns the zero value + `ok = false`; closing a closed channel panics.
3. **Senders are non-owners by default.** A function that receives a channel as a parameter and sends on it does NOT own the channel and MUST NOT close it. Conversely: a function that receives a channel as a parameter and reads from it MAY close it ONLY if the API explicitly designates the receiver as the owner (uncommon).
4. **Channel direction in function signatures.** Function signatures use directional channel types where possible: `<-chan T` for receive-only, `chan<- T` for send-only. The default `chan T` is bidirectional and implies ownership.

### Buffered vs unbuffered

- **Unbuffered (synchronous handoff).** Default choice. Send blocks until receive; receive blocks until send. Use when the synchronization point matters.
- **Buffered (asynchronous handoff up to capacity).** Use when the producer should not block on slow consumer up to a known bound. Capacity must be justified in service `README.md` or comment — "buffered to avoid back-pressure" is not justification; "buffered to N to bound producer-stall under a worst-case consumer-pause of N items" IS justification.

Unbuffered channels with `select` + `ctx.Done()` is the canonical cancellation-aware send pattern:

```go
select {
case ch <- value:
case <-ctx.Done():
    return ctx.Err()
}
```

A bare `ch <- value` without cancellation-aware select is a goroutine-leak risk if the receiver disappears.

## Context Propagation

### `context.Context` discipline

Every function that performs I/O, blocking work, or spawns a goroutine accepts `context.Context` as its first parameter. Idiomatic Go convention. Service code follows it strictly:

```go
func (s *Service) Append(ctx context.Context, event Event) error { ... }
```

Functions that do NOT perform I/O, blocking work, or goroutine spawn typically do not need a context. Adding a context to a pure-computation function is over-instrumentation.

### Context derivation

- `context.WithCancel(parent)` — derives a context that the caller can explicitly cancel.
- `context.WithTimeout(parent, d)` — derives a context that cancels after `d` elapses.
- `context.WithDeadline(parent, t)` — derives a context that cancels at absolute time `t`.
- `context.WithValue(parent, key, value)` — attaches a value to the context. Use sparingly and only for request-scoped values, not for optional parameters.

Every derivation produces a `cancel` function (except `WithValue`). The cancel function MUST be called when the derived context is no longer needed, even if the context cancels for another reason. `defer cancel()` is the canonical pattern.

### Context-cancellation semantics

A canceled context affects all goroutines derived from it. Service code MUST:

1. Check `ctx.Done()` at every blocking operation that does not itself accept the context.
2. Return `ctx.Err()` (NOT a wrapped error) when a `Done()` check causes a function to abort. The caller distinguishes cancellation from other failure modes by checking `errors.Is(err, context.Canceled)` or `context.DeadlineExceeded`.
3. NOT swallow cancellation errors silently. A function that returns `nil` after `ctx.Done()` fires is hiding a control-flow indicator from its caller.

### `context.Background()` and `context.TODO()`

- `context.Background()` is the root of context derivation; used at the top-level entry point of a service (e.g. `main` function).
- `context.TODO()` is a placeholder for unfinished code. Its presence indicates "I have not yet decided what context this function should accept." CI lint rule rejects `context.TODO()` in non-test service code merged to main.

## Substrate-Writer Serialization

Ghost-Trace-specific concurrency property derived from [`§0027`](../charter/decision-log.md) F3 (SQLite WAL single-writer constraint) and [`§2.1`](../charter/constitutional-charter.md#21-observational-integrity) (substrate append-only commitment).

### The constraint

SQLite WAL mode permits unlimited concurrent readers + exactly one concurrent writer per database file. A second attempted writer either blocks (until the first writer releases the lock) or returns `SQLITE_BUSY` (depending on the busy-handler configuration). Concurrent ingestion paths in a single service process MUST serialize writes to the events table at the application layer; relying on SQLite's busy-handling for serialization is performance-fragile and exposes the application to spurious retries.

### The serialization pattern

The ingestion service exposes a single `Append(ctx, event) error` entry point. Internally, that entry point may be:

1. **Direct serialization.** The `Append` method holds a `sync.Mutex` (or equivalent) around the SQLite write. Simplest; correct; appropriate when ingestion rate is low and write latency is acceptable.
2. **Writer-goroutine serialization.** The `Append` method sends the event to a buffered channel consumed by a single writer goroutine. The writer goroutine holds the SQLite connection and serializes writes. Allows ingest-path concurrency for marshalling + hash computation; serializes the actual SQL write. Use when ingestion rate justifies the additional complexity.

Both patterns satisfy the single-writer constraint. The choice is operational, not constitutional; surface in the ingestion service's `README.md`.

### Read path

The substrate's read path (replay, projection rebuild, query) is concurrent without restriction. Multiple goroutines may open SQLite connections in read-only mode; WAL mode permits this without blocking the writer. The application-layer mutex above guards writes only; reads do not acquire it.

### The discipline boundary

This document specifies the in-process serialization discipline. Multi-process / multi-host ingestion sharing a single SQLite substrate is forbidden at inception phase per [`§0027`](../charter/decision-log.md) Open Questions (single-writer constraint surfaced explicitly); the reversal condition R-store-3 captures the trigger for revisiting (would require PostgreSQL or similar).

## Error Propagation

### Errors from goroutines

A goroutine cannot return an error to its launcher directly; the launcher must arrange a channel. Three idiomatic patterns:

1. **`errgroup.Go`.** Each `g.Go(func() error)` captures the first non-nil error; `g.Wait()` returns it. Canonical for fan-out worker patterns.
2. **Dedicated error channel.** `errCh := make(chan error, N)` where N is the number of workers; each worker sends its terminal error (or nil) on `errCh`; the launcher reads `N` values. Use when `errgroup`'s first-error-cancels semantic is wrong.
3. **Result struct via channel.** Workers send a `Result{Value, Err}` on a typed channel; the launcher dispatches on Err. Use when results and errors must both be communicated per-worker.

### Unrecoverable errors

A goroutine that encounters an unrecoverable error (e.g. corrupted substrate detected per [`§0027`](../charter/decision-log.md) AP4) MUST surface the error to its owner; the owner MUST propagate to the service-level shutdown path. The shutdown path:

1. Cancels the root context (triggers all goroutines to terminate via cancellation propagation).
2. Waits for all goroutines to terminate (via `g.Wait()` or equivalent).
3. Logs the unrecoverable error to structured-output with the substrate state at the time of detection.
4. Exits the process with a non-zero exit code.

Unrecoverable-error detection MUST NOT be swallowed or retried at the goroutine layer. The §2.1-violation case (hash mismatch on read per [`§0027`](../charter/decision-log.md) AP4 + AP5) is the canonical example: the read path detects the violation, propagates the error up the call chain, and the service exits.

## Bounded Concurrency

### The unbounded-spawn prohibition

Service code MUST NOT spawn a goroutine per incoming event, per incoming request, or per any input over which the producer has unbounded control. Such patterns expose the service to OOM under load; an attacker (or a misbehaving upstream) can cause the service to spawn arbitrary goroutines.

### Worker-pool pattern

The canonical bounded-concurrency pattern is a fixed-size worker pool consuming from a buffered channel:

```go
const workerCount = 8
jobs := make(chan Job, workerCount*4)
g, ctx := errgroup.WithContext(parentCtx)
for i := 0; i < workerCount; i++ {
    g.Go(func() error {
        for {
            select {
            case job, ok := <-jobs:
                if !ok { return nil }
                if err := process(ctx, job); err != nil { return err }
            case <-ctx.Done():
                return ctx.Err()
            }
        }
    })
}
// producer sends to jobs; closes jobs when done
// g.Wait() returns when all workers terminate
```

Worker count + channel capacity are justified in the service's `README.md` or comment. The numbers are not arbitrary; they reflect the service's load expectations + the substrate's single-writer constraint.

## Anti-Patterns

By analogy to Charter [§2.1 Forbidden Anti-Patterns](../charter/constitutional-charter.md#21-observational-integrity). Each is concrete and detectable.

- **Fire-and-forget goroutine.** A `go func() { ... }()` whose launcher does not wait for termination. Leak by construction. Detectable: lint rule on `go ` statements without a corresponding `errgroup.Go` / `WaitGroup.Add` / equivalent owner pattern.

- **Unbounded goroutine spawn per incoming event/request.** A `go process(req)` inside a handler loop. OOM exposure. Detectable: code review on every `go ` statement inside a loop; CI lint flags suspicious patterns.

- **Channel close by non-owner.** A function that receives `chan<- T` (send-only) calls `close(ch)`. Panics at runtime if the owner also closes. Detectable: directional channel types in function signatures + lint rule on `close()` calls against non-owned channels.

- **Bare `ch <- value` without cancellation-aware select.** Goroutine-leak risk if the receiver disappears. Detectable: lint rule on bare channel-send statements (not inside a `select` with `<-ctx.Done()` branch).

- **`context.TODO()` in service code merged to main.** Placeholder for unfinished design. Detectable: CI lint rule; permitted in tests, forbidden in non-test main-branch service code.

- **Swallowing cancellation errors.** A function that returns `nil` after `ctx.Done()` fires hides the control-flow indicator. Detectable: code review on `ctx.Done()` branches; `return ctx.Err()` is the required pattern, not `return nil`.

- **Concurrent SQLite write outside the application-layer serializer.** Two code paths opening separate write connections to the events table simultaneously. Violates [`§0027`](../charter/decision-log.md) F3 single-writer constraint. Detectable: code review enforces single `Append` entry point + the entry point is the only call site of the SQLite-write code path; CI integration test launches concurrent writers and verifies serialization is observed (via timing + write-order audit).

- **Goroutine reading shared mutable state without synchronization.** Race condition. Detectable: `go test -race` in CI; race detector fails the test.

## Open Questions

- **`errgroup` vs Go 1.22+ structured-concurrency primitives.** Go's standard library is gaining structured-concurrency support (e.g. `sync.WaitGroup.Go` in Go 1.25+ proposals). When standard-library equivalents are widely available, revisit the `golang.org/x/sync/errgroup` external-dependency commitment.
- **Channel capacity policy.** This document requires justification for buffered capacity but does not codify a calculation method. Concrete calculation (e.g. capacity = worst-case-consumer-pause × producer-rate) deferred to follow-on operational document when the ingestion service's load is characterized.
- **Goroutine-count budget per service.** Inception phase services are single-process; goroutine count is bounded by worker-pool sizes only. Total budget per service (a service-tier ops question) deferred to operational-ops document.
- **Cross-process coordination primitives.** Multi-process / multi-host coordination (mutex via PostgreSQL advisory locks; distributed coordination via etcd; etc.) is out of scope for inception phase. Becomes load-bearing when [`§0027`](../charter/decision-log.md) R-store-3 fires (multi-service ingestion sharing).

## References

- [`docs/charter/constitutional-charter.md` §2.1](../charter/constitutional-charter.md#21-observational-integrity), [§2.5](../charter/constitutional-charter.md#25-hypothesis-lifecycle-explicitness) — the constitutional anchors this discipline operationalizes.
- [`docs/charter/decision-log.md` §0025](../charter/decision-log.md) — implementation-language selection (Go); modification 5 names this document as a follow-on.
- [`docs/charter/decision-log.md` §0027](../charter/decision-log.md) — storage-technology selection; F3 single-writer constraint; AP4 + AP5 unrecoverable-error semantics.
- [`docs/charter/decision-log.md` §0028](../charter/decision-log.md) — canonical-serialization-contract; concurrent marshalling safety.
- [`docs/charter/decision-log.md` §0029](../charter/decision-log.md) — introduction of this document.
- [`docs/rfcs/draft/architecture-implementation-language-selection.md`](../rfcs/draft/architecture-implementation-language-selection.md) AP1 — cross-category type imports discipline (boundary functions); related to the goroutine-launch boundary-function discipline.
- [`docs/architecture/canonical-serialization-contract.md`](./canonical-serialization-contract.md) — concurrent marshalling against the same message instance is safe; race conditions on shared message state are not.
- [Go Memory Model](https://go.dev/ref/mem) — the language-level guarantees this discipline rests on.
- [Effective Go — Concurrency](https://go.dev/doc/effective_go#concurrency) — idiomatic concurrency patterns this discipline is consistent with.
