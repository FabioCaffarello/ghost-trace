# The chain holds

The first time anything in this repository ran the product end to end
and failed when it did not.

Written during the work. The parts worth reading are the two runs that
were supposed to be red, because both of them were — and the second one
found two bugs in the gate itself.

---

## What was missing

Four services, a browser SDK, an append-only archive, six published
numbers, and three gates interrogating the composed topology. `make
shadow-http` compares the collector's and the engine's answers. `make
kill-test` takes each service away. `make loss-audit` reconciles the
archive's books through an induced outage. All three are sound and all
three pass.

Every one of them enters the chain in the middle.

They build their own request bodies — from the shared wire module,
correctly — and speak to a service directly. None of them loads the demo
page. None of them runs `sdk.js`. So the question *does the product
work* had no answer that a machine could give, and twice it was answered
by hand, days late:

- the page named an API origin no browser inside the network could
  reach. Every browser tier reported "no sessions completed", which
  reads as a detection result rather than a configuration error.
- the demo host dialled an engine it could not resolve. §5 fail-open
  turned that into a permanent `allow` that looked exactly like a
  working demo.

That is the shape of the whole problem. **A mis-wired deployment answers
every request and stays healthy.** Every container reports healthy,
every endpoint returns 200, and the verdict on the screen is a plausible
verdict. Nothing is red anywhere.

## The chain, and what asserts it

```
   demo-web-internal          collector              nats           archive
  ┌────────────────┐    ┌──────────────────┐    ┌────────┐    ┌─────────────┐
  │  GET /         │ 1  │                  │    │        │    │             │
  │  page, no      │◄───┤                  │    │        │    │             │
  │  PLACEHOLDER   │    │                  │    │        │    │             │
  └───────┬────────┘    │                  │    │        │    │             │
          │  browser    │                  │    │        │    │             │
          ▼             │                  │    │        │    │             │
   ┌─────────────┐   2  │  GET  /sdk.js    │    │        │    │             │
   │  Chromium   ├─────►│                  │    │        │    │             │
   │  (in the    │   3  │  POST /v1/       │    │        │    │             │
   │   network)  ├─────►│       sessions   │    │        │    │             │
   │             │   4  │  POST /v1/       │    │        │    │             │
   │             ├─────►│       telemetry  ├───►│ stream ├───►│  committed  │
   └──────┬──────┘      └──────────────────┘    │  + KV  │  7 │ unaccounted │
          │ submits                              └───┬────┘    │     = 0     │
          ▼                                          │ 6       └─────────────┘
   ┌─────────────┐                          ┌────────▼─────────┐
   │  demo-web,  │            5             │ decision-engine  │
   │ server-side ├─────────────────────────►│ judges the       │
   │ secret key  │  POST /v1/decisions      │ SNAPSHOT         │
   └─────────────┘                          └──────────────────┘
```

Seven links, each failing by name:

| # | what must hold | what it catches |
| --- | --- | --- |
| 1 | the page is served and carries no `PLACEHOLDER`, and names an origin this browser can reach | a page that renders identically and authenticates nothing |
| 2 | the browser fetched `sdk.js` from the collector and `window.GhostTrace` exists | the origin bug, and a 200 carrying an error page |
| 3 | `POST /v1/sessions` returned a token across the origin boundary | a CORS refusal, which is invisible to the SDK |
| 4 | every batch was accepted and carried its bearer token | wire drift, unattributed telemetry |
| 5 | a verdict rendered, **and the engine's `/v1/decisions` counter moved while the collector's did not** | fail-open; and the collector answering instead of the engine |
| 6 | `events > 0` and `confidence > 0` | a cold start, which scores identically and looks healthy |
| 7 | the committed position advanced and `unaccounted == 0` | records accepted and never made durable |

Link 5 is the one that carries the design. A verdict on a screen cannot
tell you which service produced it, and both wrong answers — fail-open
and the-collector-answered — render something correct-looking. Only the
two counters, read before and after and compared, separate them.

## The two runs that were supposed to be red

A gate nobody has seen fail is a gate nobody has tested.

**The demo dialled the collector instead of the engine.** The page
rendered `allow`, with `events=36`, `confidence=0.292`, mode `monitor`,
and six of seven links green. Every human-visible signal said the system
worked. Link 5 read `absent -> absent` on the engine and `absent -> 1`
on the collector, and failed on both.

**The page named `127.0.0.1:8080`.** Links 1, 2 and 3 failed —
`ERR_CONNECTION_REFUSED` on `sdk.js`, no session, no telemetry. The
historical bug, caught in the first link that could see it.

That second run also found two defects in the gate:

- a `waitForResponse` promise was armed several statements before it was
  awaited. The broken page made the flush call throw, the run unwound to
  the report, and the pending promise rejected during teardown — as an
  *unhandled* rejection, which killed node before a single line of the
  report printed. A gate that dies instead of reporting turns a
  three-link failure into a stack trace.
- `[].every()` is `true`, so two of link 4's three assertions reported
  `ok` in a run where nothing had ever been sent. An absence read as a
  zero, in the repository whose first rule is that those are different
  facts.

Neither would have been found by a green run. That is what the red ones
are for.

## What it deliberately does not say

**That this session is in the archive.** The archive has no read
surface, so link 7 asserts a count and not an identity: a run that
archived somebody else's records and lost its own would pass it. Named
in [ADR-0015](../contract/decisions/0015-the-e2e-gate-asserts-connection-not-detection.md),
roadmapped, and not quietly implied to be more than it is.

**Anything about detection.** No threshold, no score range, no rate.
`events > 0` is a claim that the engine judged evidence, never a claim
about the judgement. The six numbers own that question and `make
numbers` stays outside CI because a rate measured on a shared runner
encodes the runner.

## The other thing that changed

`demo-web-capture` was `demo-web` plus one flag: two services, one
image, and a fight over `127.0.0.1:8083` that the compose file had to
warn about. It folded into `-capture-log=${GT_CAPTURE_LOG:-}`.

What made the duplicate defensible was that the flag could not be
absent — `--profile demo` *was* capture, by construction. A variable can
simply be left unset, and a `demo-web` with no sink runs the human study
perfectly while recording nobody. So the promise moved from the topology
to a gate: `make capture-dryrun` now fails a run whose capture log did
not grow. Checked in both directions — with the variable unset the run
refuses while all three synthetic participants still reach `allow`.

That trade is the same one this whole document is about. Structure that
makes a mistake impossible is better than a check that catches it. When
the structure has to go, the check is not optional.

## Running it

```bash
make e2e                      # brings its own topology up, and drops it
make e2e E2E_ARGS=--keep-up   # leave it up to look at
```

It needs nothing running first — and that is deliberate. Gates do not
compose: `make loss-audit` once reported a clean run as four drops
because it inherited a topology `kill-test` had just taken apart. This
one brings up its own copy in a compose project of its own, on ephemeral
host ports, so it can neither inherit a broken scene nor collide with a
stack you already have up. It runs as the last step of CI's `topology`
job.
