# The decision path under load — the floor was a floor, not a lie

**Date** 2026-08-05 · **Target** `make load LOAD_ARGS="-scenario=decision …"`
· **Topology** compose `core` profile, Docker Desktop on macOS, 10 cores
· **Driver** `tools/loadgen`, open-loop

The 80ms decision budget has only ever been measured on an idle system,
against one session. [`latency-gate-2026-08-04.md`](latency-gate-2026-08-04.md)
reported p99 **1.393ms** and said plainly that it was a floor. This is
the same number taken while the system is busy.

## Isolating the decision

The budget is stated for a *decision*, not for a session's whole first
round trip, so one scheduled arrival has to be one `/v1/decisions` call
and nothing else.

That needs sessions that already exist and already have state — deciding
about an empty session is a different measurement and a cheaper one. 400
sessions are opened and fed a telemetry batch before the clock starts,
and the run draws from that pool at random. Sequential draws would hand
the KV a predictable access pattern, and the KV is the part of this path
worth measuring.

## The curve

| offered | achieved | svc p99 | rsp p99 | deficit p99 | failed |
| ---: | ---: | ---: | ---: | ---: | ---: |
| 100 | 100 | 4.68 | 5.63 | 1.08 | 0 |
| 500 | 500 | 3.72 | 4.07 | 0.29 | 0 |
| 1 000 | 1 000 | 2.71 | 2.94 | 0.18 | 0 |
| 2 000 | 2 000 | 8.42 | 9.02 | 0.13 | 0 |
| 4 000 | 4 000 | 3.86 | 3.93 | 0.10 | 0 |
| 6 000 | 6 000 | 3.63 | 3.67 | 0.09 | 0 |
| 8 000 | 8 000 | 5.03 | 5.07 | 0.09 | 0 |
| **10 000** | **10 000** | **6.57** | **6.62** | 0.09 | 0 |
| 11 000 | 9 520 | — | **1 452.36** | 0.13 | 0 |

All milliseconds.

**It holds to 10 000 decisions per second at p99 6.57ms against an 80ms
budget** — a factor of twelve in hand. A sustained 25-second run at
8 000/s completed 200 000 decisions with none failed and p99 6.84ms.

**It saturates between 10 000 and 11 000.** At 11 000 the achieved rate
falls to 9 520 and the tail goes to 1.45 seconds while the driver's own
deficit stays at 0.13ms — so that is the system giving way, not the
instrument.

## What the idle floor was worth

| | p99 |
| --- | ---: |
| idle, one session (published) | 1.393ms |
| 10 000 decisions/s | 6.57ms |

**4.7× higher under load, and still twelve times inside the budget.**

That is the useful finding, and it is a mild one: the floor understated,
as a floor does, but not by enough to change any conclusion drawn from
it. This is worth saying because the opposite was a live possibility —
Phase 2 measured a split whose cost is exactly the kind that grows with
contention, and nobody had checked.

## Where the work goes

Sampled during a sustained 8 000/s decision run:

| | CPU |
| --- | ---: |
| decision-engine | 139.9% |
| nats | 68.6% |
| archive | 31.3% |
| collector | 0.0% |

The KV read is real work — the broker costs about half of what the
engine itself does — but it does not dominate, and it is not what gives
way first. The collector is idle because a decision-only run does not
touch it, which is the isolation working.

## The three services, ranked

Putting this beside the earlier measurements, on the same machine:

| | bends at |
| --- | ---: |
| decision path | ~10 000 decisions/s |
| collector ingest | ~6 000–8 000 sessions/s (12–16k HTTP req/s) |
| archive commit | **4 133 records/s** |

The archive remains the constraint even after [ADR-0009](../../contract/decisions/0009-small-payloads-live-in-the-row.md)
tripled it, and the decision path — the one with a published budget and
a customer waiting on it — has the most headroom of the three.

## What this does not say

- **The 12 000/s row is not in the table.** It collapsed: 61 909 of
  96 000 requests failed and the driver fell 36 seconds behind. That is
  consistent with client-side socket exhaustion on a machine also
  running four services and a broker, and the cause was **not
  established**. Publishing it as the system's behaviour would be
  asserting something not measured.
- **400 sessions is a small pool.** A real population is larger and
  colder, so more KV reads would miss whatever caching exists at any
  layer. This is the friendly case.
- **One machine, everything on it.** The driver competes with the system
  under test. The deficit column says the driver kept its schedule; it
  does not say it took nothing away.
- **Eight seconds per step**, twenty-five for the sustained run. Nothing
  here is thermal, and nothing here is a leak test.
- **Docker Desktop on macOS is a VM.** Ratios travel; magnitudes do not.

## Reproducing

```sh
make docker-build && make up
make load LOAD_ARGS="-scenario=decision -rate=8000 -duration=25s -warm-sessions=400 -workers=4096"
```

The driver refuses a run in which it fell behind its own schedule, so a
reading that survives to print is one the instrument was able to take.
