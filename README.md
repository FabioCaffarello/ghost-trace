# Ghost Trace

**Does this session behave like a human?**

Not *which browser is this* — that question is answered by fingerprinting,
and the standard tools for defeating fingerprinting are freely available.
Ghost Trace answers a different one, from interaction dynamics alone: how
the pointer moves, how keys are timed, how a form is actually filled.

```bash
cd services/ingestion && make run     # then open http://127.0.0.1:8080
```

---

## The six numbers

Reproduce all of them with one command:

```bash
python3 experiments/numbers.py
```

Its output satisfies a published contract
([`experiments/schema/numbers.schema.json`](experiments/schema/numbers.schema.json))
and carries a provenance block — commit, machine, and the sample size
used for every tier. Runs worth citing are committed under
[`docs/results/`](docs/results/), so the numbers below can be traced to
a specific execution rather than taken on trust.

### 1. Detection rate per adversarial tier

| Tier | What it is | n | detected | 95% CI |
| --- | --- | ---: | ---: | --- |
| 1 | Playwright, no humanisation | 12 | 100.0% | [75.8%, 100%] |
| 2 | `puppeteer-extra-stealth` | 12 | 100.0% | [75.8%, 100%] |
| 3 | `undetected-chromedriver` | 8 | 100.0% | [67.6%, 100%] |
| 4 | Synthetic linear, no browser | 100 | 100.0% | [96.3%, 100%] |
| 5 | Humanised mouse (Bézier, minimum-jerk, overshoot, tremor) | 10 | 100.0% | [72.2%, 100%] |
| 6 | Humanised mouse **+** value injection | 10 | **100.0%** | [72.2%, 100%] |

Tiers 5 and 6 are **seeded** (`GT_SEED`, default `ghost-trace-v1`, and
recorded in every run manifest). Until R1.16 they drew from
`Math.random`, and tier 6 was observed at 70%, 90%, 80% and 100% across
four runs of *unchanged* code — a published rate nobody could reproduce,
including its author. The figure above is now repeatable; the 70.0% it
replaces was never re-measured after the detector changed.

Tier 5's row is the `bow` 1.0 point of a **sweep**, not the whole
result. Curvature is a dial the adversary can turn, and detection is a
smooth function of it: at `bow` 5.0 the same tier is caught 41.7% of the
time. The curve has no human reference point on it, because the human
data does not exist yet — see
[`docs/adversary-and-uncertainty.md`](docs/adversary-and-uncertainty.md).

Tier 2 is the result the project was built to test: the standard
anti-detection plugin gives **no protection at all**, because every
evasion it implements answers *which browser is this* and none of them
touch the mouse.

Tier 6 is the most sophisticated adversary here — the only one attacking
two channels at once, and doing it by *declining to produce evidence* on
each rather than faking either. It is nonetheless caught, for the reason
the tier's own source names: declining to type is itself observable. A
field whose contents change with no keystroke and no paste behind them
was not filled by a person.

**That is a weaker result than 100% makes it look.** The catch rests on
`VALUE_INJECTED`, which is the one signal the policy explicitly refuses
to treat as categorical: dictation, IME composition and some assistive
input devices reach the same code path with no preceding keydown. Tier 6
is caught by the detector's least safe signal. And at n=10 the interval
still reaches down to 72%, while a different seed produced 9 of 10 — so
the point estimate is seed-sensitive at this sample size.

### 2. False-positive rate on human traffic

> ## NO DATA
>
> **This is the number that governs everything, and it does not exist.**

A detector that flags every session scores 100% on every tier above.
Until this row has a value, number 1 is not a measurement — it is an
upper bound on nothing.

It is also the number with the highest human cost. Blocking a person is
far worse than admitting a bot, and the people most likely to be
falsely flagged are those whose motor patterns differ most from the
norm: tremor, motor impairment, switch and eye-tracking input,
non-dominant-hand use, the very old and the very young.

The statistics to report it are built, tested against a synthetic
fixture, and waiting: person-level counts as the primary figure, an
intraclass correlation estimated from the data rather than assumed, and
no code path anywhere that emits a bare session-level rate. See
[`experiments/README.md`](experiments/README.md).

What is missing is people. It is calendar-bound, not effort-bound.

### 3. p99 decision latency

```
  p50 2.34ms    p95 2.94ms    p99 5.84ms    max 6.15ms
  budget       80ms
```

Single session, idle system, over HTTP including the archive write.

### 4. Time to confident decision

| | reached | median batches | median events | median session time |
| --- | --- | ---: | ---: | ---: |
| challenge becomes possible | 30/30 | 2 | 56 | **3.7s** |
| block becomes possible | 30/30 | 3 | 84 | **5.7s** |

Almost nobody publishes this. A login flow that takes a user four
seconds is right at the edge of judgeable; under two seconds is a cold
start no matter how bot-like it looks.

### 5. Cold-start behaviour

```
  n=30   median score 0.0   median confidence 0.0
  decision:  allow  (30/30)
  reason:    INSUFFICIENT_EVIDENCE
  ever blocks a first visit:  no
```

A session eleven events old can look maximally bot-like and must not be
blocked. That is why `score` and `confidence` are separate fields rather
than one number — answering cold-start in the contract means every
consumer handles it the same way.

### 6. Memory and latency by concurrency × duration

Maintained state versus compute-on-call, same extractors, same policy.

```
  bytes per session      10s       300s      1800s
  maintained             826        825        826      ← flat
  on-call              4,153     59,066    281,273      ← 68x
```

At 10,000 concurrent sessions of 30 minutes: **8.1 MB against 2.68 GB.**

**Latency never forces this decision.** Compute-on-call's worst p99 in
the entire grid is 0.29ms against an 80ms budget — 273× inside it. The
architecture is justified by how much memory you hold while waiting to
be asked, not by how fast you answer. The tempting claim — *tight
latency forces stream processing* — is the one this measurement
disproves. See
[`docs/duration-forces-the-architecture.md`](docs/duration-forces-the-architecture.md).

---

## What it collects, and what it refuses to

**Collected:** pointer geometry, key *timing* and coarse class, scroll
displacement and mode, focus transitions, page visibility, form
paste/autofill/submit.

**Never collected:**

- Keystroke **content**. Only timing and a six-way class. M1 collected no
  key events at all.
- Canvas, WebGL, font enumeration, audio fingerprinting. These identify
  the browser rather than the behaviour and contradict the thesis.
- Any persistent client identifier. The session token lives in a closure
  and dies with the page.
- Field values. Field *identities* are hashed; values never leave the
  page.

All timestamps are session-relative — client wall-clock is untrustworthy
and leaks more than it gives.

A narrow set of device properties *is* collected — pointer type, touch,
viewport, timezone offset, reduced-motion — because they change how
behaviour should be *interpreted*. A trackpad and a mouse produce
structurally different traces; comparing them without normalising
measures the input device, not the operator. The test for admitting a
property is whether it changes interpretation, not whether it helps tell
one browser from another.

---

## Layout

```
  services/ingestion/          one Go binary
    internal/feature/          Category II: deterministic feature extraction
    internal/policy/           score / confidence / decision
    internal/session/          maintained state + the on-call comparison
    internal/api/              the four contract endpoints
    internal/web/              demo page and SDK
    cmd/bench-architecture/    the concurrency x duration grid
  experiments/                 six adversarial tiers + the statistics
  schemas/events/v1/           protobuf archive schema
  contract/                    what must be true: the architecture contract,
                               the generated OpenAPI surface, the fixtures
                               the harness actually sends, the target design
  .context/                    agent harness: playbooks, skills, sensors, policy
  experiments/schema/          the contract the six numbers satisfy
  docs/                        write-ups of work that has run
  docs/results/                committed run manifests, with provenance
```

`contract/` holds what must be true:
[`architecture.md`](contract/architecture.md) §0–§9, the
[`openapi.yaml`](contract/openapi.yaml) **generated** from the Go types
the handlers decode into, the [`fixtures/`](contract/fixtures/) the
experiment harness actually sends, and the forward-looking
[`roadmap.md`](contract/roadmap.md). CI fails if any generated artifact
drifts from its source, and every example in the specification is a
committed fixture or golden — bytes something really produced, not a
plausible invention.

`docs/` holds written artifacts about work that has run — never
specifications, never governance. The architecture contract lived there
as the rule's one standing exception until R1.16b; splitting the
directory preserves the rule instead of eroding it. That single rule is the difference
between this directory and the 27,651 lines deleted in
[`docs/v1-retrospective.md`](docs/v1-retrospective.md).

---

## Working on it

```bash
make bootstrap   # assert the toolchain, and name what is missing
make help        # every target, grouped
make verify      # format, vet, lint, race tests — run this before pushing
make ci          # everything CI runs, in the order CI runs it
```

Every step in the pipeline is a `make` target and nothing else, so a
green CI run and a green `make ci` are the same statement rather than
two things that resemble each other. `make hooks` installs the git hooks
that run the fast half of it on commit.

Commits follow [Conventional Commits](https://www.conventionalcommits.org),
with the milestone as the scope:

```
feat(r1.14): expose the decision contract as OpenAPI 3.1
fix(policy): stop rounding scores below zero to positive
ci(r1.13): pin every action by commit SHA
refactor(session)!: ports take a context      # ! is a breaking change
```

`feat` releases a minor, `fix` and `perf` a patch, `!` a major, and
everything else releases nothing — release automation reads the log, so
the type is a decision rather than a label. Pull requests are
squash-merged, which means **the PR title is the commit that lands**; it
is checked in CI by the same script the commit hook runs.

---

## Honest limitations

- **No false-positive rate.** Everything above describes machines.
- **Nothing is calibrated.** Every threshold is an inception guess. A
  plausibly-curved humanised path scores 0.942 against a 0.90 floor —
  if that is what real reaching looks like, the floor is a machine for
  flagging people.
- **`VALUE_INJECTED` has a known false-positive population.**
  Speech-to-text dictation, IME composition, and some assistive input
  devices produce input events with no preceding keydown and look like
  injection. The signal is capped below categorical and cannot block
  alone, but it must not gain weight before dictation and IME sessions
  are in the capture study.
- **Telemetry replay is unsolved.** An adversary who records a genuine
  human session and replays its event stream under a fresh token is the
  strongest attack against any system in this class. Partial mitigations
  raise the cost; none close it.
- **One process, one machine.** Nothing here measures a deploy
  surviving, state spanning replicas, or GC churn under real load.
- **Detection rate is a dial the adversary holds.** Tier 5's curvature
  sweep moved detection from 100% to 42% by varying one parameter.

---

## Contributing

[`CONTRIBUTING.md`](CONTRIBUTING.md) has the prerequisites, the `make`
loop, the commit convention, and the things that will fail a pull
request. Two rules there are not style: **absence is never zero**, and
**key content is never collected**.

[`CODE_OF_CONDUCT.md`](CODE_OF_CONDUCT.md) applies — this project asks
strangers for five minutes of their interaction data, and owes them a
stated standard for how strangers are treated in return.
[`SECURITY.md`](SECURITY.md) has a private reporting channel.

Licensed under [Apache 2.0](LICENSE).

---

## Reading order

1. [`contract/architecture.md`](contract/architecture.md) — the architecture
   contract: the external surface everything is derived from
2. [`docs/v1-retrospective.md`](docs/v1-retrospective.md) — what the
   first attempt got wrong
3. [`docs/vertical-slice.md`](docs/vertical-slice.md) — the smallest
   thing that scores a session
4. [`docs/adversary-and-uncertainty.md`](docs/adversary-and-uncertainty.md)
   — four bots and a confidence interval
5. [`docs/time-to-confident-decision.md`](docs/time-to-confident-decision.md)
   — five evidence channels
6. [`docs/duration-forces-the-architecture.md`](docs/duration-forces-the-architecture.md)
   — the benchmark that corrected the plan
7. [`docs/results/`](docs/results/) — run manifests: which commit, which
   machine, which seed produced the numbers above
8. [`contract/roadmap.md`](contract/roadmap.md) — where the architecture
   is going, kept separate from what it currently guarantees
