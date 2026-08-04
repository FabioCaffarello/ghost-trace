# Parity at cutover — 2026-08-04

The measurement ADR-0003 required before the dual-write could be
removed: *"when the archive service has demonstrably held everything the
collector wrote across a full experiment run."*

Taken on the composed topology with the dual-write still in place, so
both stores existed and could be compared. ADR-0006 is the decision this
evidence bought.

## What was run

`docker compose --profile core --profile experiments run --rm experiments`
— the adversarial tiers against the running topology: collector,
decision engine, archive, NATS and the in-network demo host.

Five of six tiers ran. Tier 3 records itself **absent** in a container:
`undetected-chromedriver` needs real Chrome, which has no linux/arm64
build. That absence is loud rather than silent, and it does not weaken
this measurement — parity is about records crossing the stream, and the
other tiers produced 3419 of them.

## What was compared

Every record in the collector's local substrate against every record in
the archive service's substrate, **by content hash** — not by count. A
matching total with different contents would pass a count check and fail
this one.

| message type | collector | archive |
| --- | ---: | ---: |
| `ghosttrace.events.v1.SessionStart` | 373 | 373 |
| `ghosttrace.events.v1.TelemetryBatch` | 1560 | 1560 |
| `ghosttrace.events.v1.Evaluation` | 1486 | 1486 |
| **total** | **3419** | **3419** |

- in the collector and not in the archive: **0**
- in the archive and not in the collector: **0**

The collector's own mirror counters, logged at shutdown:

```
event stream mirror  appended=3419  published=3419  dropped=0
```

`published + dropped == appended` is the invariant ADR-0003 named as the
evidence this cutover needed. It holds with `dropped` at zero.

## The fourth record type

An experiment run produces no `Outcome` records — the tiers do not
report labels. A separate drive of 40 sessions through the collector's
HTTP surface (sessions → telemetry → decisions → outcomes) covered it:

| message type | collector | archive |
| --- | ---: | ---: |
| `ghosttrace.events.v1.SessionStart` | 40 | 40 |
| `ghosttrace.events.v1.TelemetryBatch` | 120 | 120 |
| `ghosttrace.events.v1.Evaluation` | 40 | 40 |
| `ghosttrace.events.v1.Outcome` | 40 | 40 |
| **total** | **240** | **240** |

`appended=240 published=240 dropped=0`. Between the two runs every
archived message type has been compared.

## What this does not show

- **Nothing about behaviour under a broker outage.** Both runs had a
  healthy NATS throughout. What the counters would show during an outage
  is `dropped > 0`, and the dual-write existed precisely so that case
  cost nothing; after ADR-0006 it costs a 503 on `/v1/outcomes`. That
  trade is argued in the ADR, not measured here.
- **Nothing about scale beyond this run.** 3419 records over a few
  minutes on one machine. The claim is that the path is lossless, not
  that it is fast or that it stays lossless under load — PR-2.6 is the
  latency gate.
- **Nothing about a partitioned archive service.** The consumer acks
  only after committing, so a crash between the two redelivers; that is
  covered by `make parity`, not by this.

## Reproducing it

The comparison tool was written for this measurement and deliberately
not committed: after ADR-0006 there is no local write left to compare
against, and a tool that dies on the same pull request is waste. It
walked both substrates with `substrate.WalkEvents` and diffed the
`EventRow.EventHash` sets — about eighty lines.

What survives is `make parity`, which asserts the same property at the
consumer: a record reaching the stream lands in the substrate, byte for
byte.
