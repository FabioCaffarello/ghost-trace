# 0015 — the end-to-end gate asserts connection, not detection, and says what it cannot see

**Status:** accepted · **Date:** 2026-08-12 · **Milestone:** PR-E3

## Context

By Phase 6 the composed topology had three gates and every one of them
was sound. `make shadow-http` A/Bs the collector against the decision
engine. `make kill-test` takes each service away and checks the
degradation each decision record promises. `make loss-audit` reconciles
the archive's books through an induced outage. Between them they cover
equivalence, degradation and accounting.

All three enter the chain in the **middle**. Each builds its own request
bodies — from the shared wire module, correctly — and speaks to a
service directly. Not one of them loads the demo page, and not one of
them runs `sdk.js`.

So the repository had no gate for the thing a customer buys: that a
visitor's browser, on the customer's own origin, produces a verdict the
customer's server can act on. Twice that failed, and both times the
failure survived because **a mis-wired deployment answers every request
and stays healthy**:

- the page named an API origin no in-network browser could reach, so
  every browser tier reported "no sessions completed" — a symptom that
  reads as a detection result, not a configuration error;
- the demo host dialled an engine it could not resolve, and §5 fail-open
  turned that into a permanent `allow` that looked exactly like a
  working demo.

Both were found by hand, days later. Neither is a failure any of the
three gates can see, because none of them was looking at the product.

The adversarial tiers in `experiments/` do drive real browsers through
the whole path — but they are a **measurement**, seven minutes long,
answering *how well does this detect bots*. A number that moves is a
finding to investigate. A chain that is broken is a deployment that does
not work. Using the first as the second means waiting seven minutes to
learn that a container name was wrong.

## Decision

**`make e2e` drives the product path with a real browser inside the
compose network and asserts seven links by name.** The page, `sdk.js`,
the cross-origin session handshake, telemetry, the decision the host
application makes server-to-server, the snapshot that decision was
derived from, and the archive's durable position.

Three properties are the decision; the rest is implementation.

**It asserts CONNECTION, never DETECTION.** No threshold, no score
range, no rate. It checks that `events > 0` — that the engine judged
evidence rather than a cold start — and never what the judgement was.
The moment this gate has an opinion about a score it becomes a
measurement that fails on a machine, and `make numbers` already owns
that question and stays outside CI for exactly that reason.

**Every link fails by name, and no link may pass vacuously.** A link
that cannot be observed is a failure. This is [ADR-0008](0008-what-a-zero-is-allowed-to-mean.md)
applied to a browser: the counter for a route a process has never served
is *absent*, not zero, and the gate distinguishes them — link 5 reads
`absent -> 1` on the engine and `absent -> absent` on the collector, and
those are different claims. The first deliberately-broken run found two
places where the gate itself violated this: `[].every()` reported `ok`
for two of link 4's assertions in a run where nothing had been sent.

**It brings its own topology up, in a compose project of its own.**
Gates do not compose — `loss-audit` once reported a clean run as four
drops because it inherited a system `kill-test` had just taken apart.
A gate that must be handed a quiet topology will eventually be handed a
noisy one. This one cannot be, and as a consequence it runs alongside a
stack the operator already has up.

## What it deliberately does not assert

**That this session is in the archive.** The archive exposes `/healthz`
and `/metrics` and nothing else — there is no read surface, so the
strongest available claim is that the durable position advanced by at
least what the run produced and that `unaccounted` is zero. That is a
**count, not an identity**. A run that archived somebody else's records
and lost its own would pass link 7.

This is stated rather than implied because the alternative is a gate
whose users believe it proves more than it does. A read endpoint on the
archive would close it and is on the roadmap; it is a contract change
with an authorization question attached — an endpoint that returns
archived telemetry by session is a different thing from one that returns
a count — and that question is not this gate's to answer in passing.

**A rate, or a latency.** `make load-gate` measures those, encodes the
hardware it ran on, and stays out of CI for the reason recorded in the
Makefile.

**That telemetry which the browser aborted arrived.** Chromium aborts
keepalive requests it has already handed to the network; the run that
established this saw three `ERR_ABORTED` for batches the collector
logged as `202`. Whether anything was lost is not a question a browser
can answer, and link 7 answers it from the archive's books instead. The
aborts are reported, not failed on.

## Consequences

The gate is a `make` target and a sensor like every other, so agents,
CI and humans share one vocabulary for it. It runs as a step in the
existing `topology` job rather than as a job of its own: branch
protection requires three summary contexts, and a fourth job would need
a settings change to protect anything.

It costs a Playwright base image and about a minute. That is why it is
the last step in its job — its position is otherwise irrelevant, since
it cannot inherit anything, and a cheaper gate that is going to fail
should say so first.

It reuses the experiments image rather than building one of its own. The
two have nothing in common but Chromium and a lockfile; a second 2 GB
base to run one session would be a build nobody would keep current.
