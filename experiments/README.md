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

---

## Running it

The canonical entry point is one command from the repository root:

```bash
python3 experiments/numbers.py
```

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
cd services/ingestion && make run          # terminal 1
cd experiments && python3 run.py && python3 analyze.py   # terminal 2
```

A tier whose dependencies are missing is recorded in
`results/absent_tiers.txt` and reported by `analyze.py` as **ABSENT**,
never silently skipped. Six tiers listed and five run reads as "we
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
http://<host>:8080/?p=p07&arm=B&c=mouse-desktop&v=1
http://<host>:8080/?p=p02&arm=A&c=trackpad-tired&v=17
```

- `p` participant code — pseudonymous, never a name
- `arm` `A` or `B`
- `c` condition label
- `v` visit index, used for the habituation contrast

Run the slice with `-capture-log` to record them:

```bash
go run ./cmd/ghost-trace -data .run-data \
  -capture-log ../../experiments/results/human_sessions.jsonl \
  -addr 0.0.0.0:8080
```

The labels travel through the contract's existing `subject_id` and
`context` fields rather than any new API surface. A session's cohort is a
property of the experiment, not of the product — the engine must not know
which population it is looking at.

### What volunteers should be told

- Only **how the pointer moves** is recorded. Not what is typed — M1
  collects no key events at all.
- No canvas, WebGL, font or audio fingerprinting.
- Nothing persistent is written to their browser.
- The participant code is a pseudonym they can discard.
- They can use a throwaway value in the form fields; the content is
  never read.

---

## Output

- `results/sessions.jsonl` — bot sessions, labelled by cohort
- `results/human_sessions.jsonl` — human sessions, labelled by
  participant / arm / condition / visit
- `results/absent_tiers.txt` — tiers that did not run and why
