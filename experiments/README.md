# Experiments

The validation layer: six adversarial bot tiers, the statistics that
exist to prevent false precision, and a human capture study — all run
against the live slice.

Without an adversary every metric is unfalsifiable, which is why this
exists before any further features.

---

## Read this before reading any number it prints

**The two sides of this experiment are not epistemically equal, and the
asymmetry is not a flaw to be fixed — it is the finding.**

The bot side is reproducible and unlimited. A tier runs ten thousand
sessions overnight for free, the sessions are independent, and the
confidence interval is tight. Detection rate is a real measurement.

The human side is capped by how many people will give up five minutes,
and its observations are **not independent** — sessions cluster within
people. Three separate problems follow, and the README, the experiment
output and the write-up all state them:

1. **Sample size.** A production anti-bot system operates near 0.1%
   false-positive rate. Bounding that needs roughly 3,000 independent
   observations. This study reaches two orders of magnitude short, and
   no amount of engineering closes the gap.

2. **Clustering.** If being falsely flagged is close to a property of the
   person — which is precisely what point 3 argues — then effective
   sample size approaches the **number of people**, however many sessions
   each contributes. Eight people running a thousand sessions each still
   gives an effective n of about eight.

3. **Directional bias, which no interval captures.** The humans most
   likely to be falsely flagged are those whose motor patterns differ
   most from the norm: tremor, motor impairment, switch or eye-tracking
   input, non-dominant-hand use, very old and very young users. A
   volunteer pool drawn from one person's social graph systematically
   under-samples exactly that population.

   So the number is not merely imprecise. It is **optimistic in a known
   direction**, and that direction carries the highest human cost.
   Blocking a human is far worse than admitting a bot.

Points 2 and 3 are one fact seen from two sides. If false positives were
spread evenly across people, the intraclass correlation would be low and
pooled session counts would be honest. They are not spread evenly, which
is simultaneously why the correlation is high, why the person is the
right unit of analysis, and why the sample is biased.

`analyze.py` has no code path that prints a bare session-level
false-positive rate. Everything is an interval, and the clustering
correlation is **estimated from the data**, never assumed.

Those statistics are tested before anyone runs them on people:

```bash
make experiments-check      # or: python3 experiments/analyze.py --selftest
```

The selftest pins closed-form properties of the estimators (Wilson at
zero events, the design effect collapsing effective n to the number of
people at ρ=1, the shadow-decision precedence that makes monitor-mode
detection rates non-zero) and then asserts that the committed synthetic
fixture — which plants three atypical people among twenty — is recovered
as the high-ρ structure it is. It exits non-zero when it isn't, and CI
runs it on every push. The fixture is **not data**: it exists so the
study, which can only be run once, is not the first execution of this
code.

---

## Running it

The canonical entry point is one command from the repository root:

```bash
python3 experiments/numbers.py
```

### Seeding

Tiers 5 and 6 are the only ones with randomness, and until R1.16 they
drew from `Math.random` — `humanPath` took a `rand` parameter and no
caller ever passed one (audit M13). Every published rate from those
tiers was a sample of an unrepeatable experiment.

```bash
GT_SEED=ghost-trace-v1 python3 experiments/numbers.py   # the default
```

Each session derives its own generator from `<seed>:<cohort>:<index>`,
so session 7 draws the same numbers whether it ran alone or after six
others — which is what makes one flagged session replayable. The label
is recorded in every result row and the run seed in every manifest,
because a seeded run nobody wrote down is exactly as unreproducible as
an unseeded one.

`rand` is now **required**: `humanPath` and `thinkMs` throw without it.
A default was what let four published runs go unseeded unnoticed, and
the requirement caught two call sites this change had missed.

What seeding buys is the adversary, not the measurement. A real browser
is in the loop and its event dispatch, scheduling and the SDK's sampling
clock are not ours to seed — three runs at one seed gave identical
verdicts with scores drifting 0.757–0.775. It removes the variance we
control and leaves the variance we do not.

### What the harness sends

Every request body the harness builds comes from one module per
language — [`lib/wire.js`](lib/wire.js) and [`wire.py`](wire.py). Before
R1.15 there were five hand-rolled producers, and because the service
tolerates unknown fields by design (§5, §7), renaming a wire field would
have left them all sending the old name, the server zero-filling the new
one, and every measurement quietly degrading with the suite green. That
is audit finding M22.

```bash
make contract-fixtures        # emit what those modules produce
make contract-fixtures-sync   # fail if the fixtures drift from them
```

The fixtures in [`contract/fixtures/requests/`](../contract/fixtures/requests/)
are not hand-written — they are what the wire modules produce, which is
also how the emitter can assert that the **JavaScript and Python halves
of the harness agree byte for byte**. If they disagree about the wire,
one of them is measuring something else, and no server-side test would
show it.

A Go test then checks each fixture twice: against the published OpenAPI
request schema (`additionalProperties: false`, so a renamed field fails
from either direction) and by **replaying it against a real server**,
because satisfying a schema is not the same as being accepted. Those
same fixtures are the request examples published in the contract.

Its output has a contract:
[`schema/numbers.schema.json`](schema/numbers.schema.json). `numbers.py`
validates against it **before writing** — a measurement that costs
browsers and minutes to reproduce should not be published in a shape
nothing can read, and the six numbers are quoted in the root README, so
a malformed run is a malformed claim.

The validator is stdlib (`schema/__init__.py`), because `numbers.py`
being runnable with nothing installed is the point of the one-command
promise. A hand-written validator that quietly accepts what it does not
understand would be worse than none, so two things constrain it: any
JSON Schema keyword outside its implemented subset **raises** rather
than being skipped, and the fixture corpus in `schema/testdata/` is run
through a real JSON Schema implementation in Go by
`numbers_schema_test.go`, which fails if the two disagree.

```bash
make experiments-check   # includes the schema fixture selftest
make numbers-manifest    # validate the last run and publish it to docs/results/
```

Every run carries a `provenance` block — commit, dirty flag, machine,
cpu count, run mode and the sample size actually used per tier, because
`GT_N_TIER<k>` silently changes what was measured. Published manifests
live in [`docs/results/`](../docs/results/).

It builds the service binary, starts it on a private port with a
private data directory, runs every tier, measures latency /
time-to-confident-decision / cold start, runs the two-architecture
benchmark, and writes `results/numbers.json` plus the printed table the
README's six numbers come from.

One-time setup for the browser tiers:

```bash
cd experiments
npm ci                                              # tiers 1, 2, 5, 6
python3 -m venv .venv && .venv/bin/pip install -r requirements.txt   # tier 3
```

Tier 4 needs no dependencies at all — it speaks the API directly with
Node's built-in fetch. Tiers 1, 2, 5 and 6 drive the installed Chrome;
set `GT_CHROME` if yours is somewhere unusual.

The two-terminal flow is still available for iterating on a single
tier against a long-lived server:

```bash
cd services/collector && make run          # terminal 1
cd experiments && python3 run.py && python3 analyze.py   # terminal 2
```

A tier that does not run — missing dependencies, a dead endpoint, or a
run where every session failed — exits non-zero and is recorded as
**ABSENT** (`results/absent_tiers.txt` for `run.py`, `absent_tiers` in
`numbers.json` for the canonical command), never silently skipped. Six tiers listed and five run reads as "we
tested six things", and the missing one is always the one that would
have found something.

---

## The tiers

| Tier | What it is | What it evades |
| --- | --- | --- |
| 1 | Playwright, no humanisation | nothing — the baseline |
| 2 | `puppeteer-extra-plugin-stealth` | fingerprinting: `navigator.webdriver`, plugin arrays, WebGL vendor, CDP markers |
| 3 | `undetected-chromedriver` | fingerprinting: patched chromedriver, CDP concealment, automation flags |
| 4 | Synthetic linear script, no browser | everything fingerprint-shaped — there is no browser to fingerprint |
| 5 | Humanised mouse (Bézier, minimum-jerk, overshoot, tremor) | the pointer channel, by mimicry |
| 6 | Humanised mouse **+** value injection (`page.fill`) | pointer by mimicry, keystrokes by declining to produce evidence |

**Tier 2 is the one the thesis stands or falls on.** Every evasion the
stealth plugin implements targets *which browser this is*. None of them
touch the mouse. If behavioural detection is worth building, tier 2
should be as detectable as tier 1. If tier 2 evades where tier 1 does
not, the thesis is wrong and the write-up says so.

### Two ways to evade, and they are not the same

`analyze.py` reports an evasion mode per tier, because mean score alone
cannot distinguish them:

- **By mimicry** — evidence is present and looks human. This is the hard
  case and the one the feature is meant to catch.
- **By absence** — too little evidence to judge at all. Selenium's
  `ActionChains` jumps the cursor rather than interpolating, and
  Playwright's `mouse.move` without `steps` teleports, so both can
  produce almost no `pointermove`. The confidence dimension reports this
  correctly as *insufficient evidence* and the decision is `allow`.

A tier scoring zero because it produced nothing to look at is not a
detector working. It is a gap, and it is reported as one.

---

## The human capture study

Two arms, because the design serves two questions with opposite optima.

### Arm A — condition sensitivity (depth)

~3 people × ~40 sessions, conditions crossed: input device, viewport,
familiarity, alertness, `prefers-reduced-motion`.

Answers: does the same person's score drift as they habituate? Does
switching trackpad→mouse move it more than alert→tired?

**Arm A reports effect sizes and no false-positive rate.** Three people
cannot support a population rate, and printing one would be exactly the
false precision this experiment layer exists to prevent.

The habituation contrast is the one most likely to matter: the 30th
visit to a form produces fast, ballistic, low-correction movement that
looks markedly more synthetic than the 1st. If that effect is large, the
system degrades against its most loyal users — a finding worth more than
any single rate.

### Arm B — false-positive breadth

As many distinct people as possible × 3–5 sessions each. Target ~20.

Precision here is governed by **person count alone**, so the recruitment
ask is *"five minutes each from as many people as possible"*, not
"thirty sessions from a friend". The first is something you can send to
a group chat; the second is something you can ask of one person.

### Participant links

Each volunteer gets a link carrying their pseudonymous code:

```
http://<host>:8083/?p=p07&arm=B&c=mouse-desktop&v=1
http://<host>:8083/?p=p02&arm=A&c=trackpad-tired&v=17
```

Port **8083**, not 8080. The page a volunteer opens is served by
`demo-web`; 8080 is the collector, which serves the SDK and the API and
has no page to show. This line said 8080 until PR-4.P2.

- `p` participant code — pseudonymous, never a name
- `arm` `A` or `B`
- `c` condition label
- `v` visit index, used for the habituation contrast

`-capture-log` is a **`demo-web`** flag, not a collector one. The
collector never sees the labels — it sees a session like any other. Run
both, from the repository root:

```bash
# the collector: sessions, telemetry, the SDK
go run ./services/collector/cmd/ghost-trace -data .run-data -addr 127.0.0.1:8080

# the page the volunteer opens, and the only thing that writes the log
go run ./services/demo-web/cmd/demo-web -addr 0.0.0.0:8083 \
  -api http://127.0.0.1:8080 \
  -capture-log experiments/results/human_sessions.jsonl
```

This block ran `./cmd/ghost-trace -capture-log` until PR-4.P2, which the
collector has no flag for: an operator following it verbatim got
`flag provided but not defined` and no capture at all.

Under compose the same flag is `GT_CAPTURE_LOG`, a path **inside** the
container — `./experiments/results` is mounted at `/captures`:

```bash
GT_CAPTURE_LOG=/captures/human_sessions.jsonl \
  docker compose --profile demo up -d
```

There is no separate capture service any more. There was one, and what
made it safe was that it could not be started without the flag; a
variable can simply be left unset, and a `demo-web` with no sink serves
the study perfectly while recording nobody. `make capture-dryrun` is
what closes that: it fails a run whose capture log did not grow, and
prints both ways to turn capture on.

**Only `participant` crosses the wire**, as `subject_id` — the
pseudonymous identity the host application asserts, which is exactly what
`subject_id` is for. `arm`, `condition` and `visit` stay in `demo-web`
and go straight into the capture row: a session's cohort is a property of
the experiment, not of the product, and the engine must not know which
population it is looking at.

Once both are up, check the pipeline before anyone is invited:

```bash
make capture-dryrun DRYRUN_ARGS="--data .run-data"
```

It drives three participants who do not exist through the same path a
real one takes, and asserts that each produces one labelled row, that the
labels survive, and that `arm`, `condition` and `visit` reach neither the
collector nor the archive. Without `--data` it says so rather than
reporting the archive clean — an absent check is not a passing one.

The older wording here said the labels travelled through `subject_id`
**and `context`**. That was wrong twice over. `context` was removed from
the contract at R1.14 — `libs/wire` records that it "was accepted, never
reached the use case, and never reached the archive" — and the three
cohort labels were never meant to cross at all.

### What volunteers should be told

The script lives in **[`PARTICIPANTS.md`](PARTICIPANTS.md)**, in a file
of its own, and it is not duplicated here.

That is deliberate. It used to live in this README, where a change to
what the SDK collects produced no diff a volunteer would ever be shown —
and the wording went stale exactly that way, claiming no key events were
collected after the keystroke channel was added. A second copy would
recreate the problem it was moved to solve.

`disclosure_test.go` compares that file against the vocabulary the SDK
actually emits, in both directions, on every `make ci`. Adding a
collected value without describing it fails the build; describing a
channel that does not exist fails it too.

**Recruitment is gated** on a data-governance RFC that has not been
written. `PARTICIPANTS.md` says so, and a test fails if it stops saying
so.

---

## Output

- `results/sessions.jsonl` — bot sessions, labelled by cohort
- `results/human_sessions.jsonl` — human sessions, labelled by
  participant / arm / condition / visit
- `results/absent_tiers.txt` — tiers that did not run and why
