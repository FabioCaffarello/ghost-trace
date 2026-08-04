# Four bots and a confidence interval

M2. The first numbers that can be wrong.

Written during the work. The headline result is not the detection rate —
it is that the adversary falsified the detector within an hour of first
contact, and that one of the four tiers still walks straight through.

> [!IMPORTANT]
> **Superseded in part.** This is the M2 write-up, kept as the record of
> what was true when it was written. Two of its results have since
> reversed, and a reader following the reading order meets them here
> first:
>
> - **Tier 3 is reported at 0.0% below. It is now 100%.** The tier walks
>   straight through a detector that did not yet read the keystroke and
>   form channels; those landed in M3, and its evasion — jumping the
>   cursor instead of moving it — now shows up as *absence of evidence*
>   rather than as human-looking evidence.
> - **Tier 6 did not exist yet.** It was written in M3 and is caught at
>   100% — by `VALUE_INJECTED`, the one signal the policy refuses to
>   treat as categorical.
>
> What has NOT changed, and is the more important half: the tier 5
> sweep below still falls to **41.7% at `bow` 5.0**. The README
> publishes only the `bow` 1.0 point. Detection is a smooth function of
> how much curvature the adversary paid for, and the curve still has no
> human reference point on it.
>
> The current figures, with the commit and seed that produced them, are
> in [`docs/results/`](results/) and the README. The argument this
> document makes — that three 100% rows and no false-positive rate is
> not a result — is unchanged and is why it is still in the reading
> order.

---

## The number that matters is missing

```
  tier                              n  detected    rate            95% CI   mean score
  ---------------------------------------------------------------------------------
  tier1_playwright_naive           25        25  100.0%  [86.7%, 100.0%]        0.781
  tier2_puppeteer_stealth          25        25  100.0%  [86.7%, 100.0%]        0.846
  tier3_undetected_chromedriver    17         0    0.0%  [ 0.0%,  18.4%]        0.000
  tier4_synthetic_linear          200       200  100.0%  [98.1%, 100.0%]        1.000

  false-positive rate on human traffic:   NO DATA
```

**Three 100% rows and an unknown false-positive rate is not a result.** A
detector that flags every session scores 100% on every bot tier. Until
the bottom row has a number, the top three are unfalsifiable in exactly
the way this project exists to avoid, and they are reported here only
because the alternative — waiting to publish until the human data
arrives — is how a six-week milestone becomes a six-month one.

Recruitment is open. It is calendar-bound rather than effort-bound: it
needs people, and no amount of engineering substitutes.

---

## What the adversary found in the first hour

Before any of the rows above, the harness ran against the M1 detector as
merged. Tier 1 scored **0 out of 22**.

Not a marginal miss. The feature was inverted under precisely the
conditions it exists to detect: a real browser driven by a real
automation script, moving in perfectly straight lines, scored **0.000**
— maximally human.

### Cause one: bots do not pause

M1 split the pointer stream into movement segments on a 300ms time gap,
on the reasoning that a session is many separate movements and the
straight line between the first and last point of a five-minute session
says nothing about any of them. That part is right.

What it missed is that **the 300ms threshold encodes an assumption about
human pacing**. A person pauses between targets — to read a label, to
decide, to find the next field. A script does not. Playwright moves to
the username box, clicks, types, and moves to the password box with
nothing between the legs but the few milliseconds the CDP round trip
costs.

So three straight legs and two click-jumps arrived as one continuous
polyline, and the composite looked like this:

```
  path length      1345 px
  net displacement  424 px
  straightness      0.315      ← reads as heavy human correction
```

The direction changes at each target are geometrically indistinguishable
from the overshoot-and-correct of a human reach. The bot evaded by being
**too fast to segment**, which is the opposite of what anyone would
design against.

### Cause two: every second corner was missed

The fix was to split on sharp direction change as well as elapsed time —
a human reach curves continuously and turns gently; arriving at a
different target is a corner. 60°.

That took straightness from 0.315 to **0.798**. Still below the 0.90
floor, still scoring zero, and the reason took a second pass to find:
starting a new segment cleared the stored direction, and the turn branch
never set it back. The very next step therefore skipped its own turn
check, so **every second corner was missed and each segment swallowed one
click-jump**.

One line. Straightness went 0.798 → **0.951**, and the jump legs dropped
out of the measured path (1345px → 1228px).

```
  segmentation            straightness   tier 1 detected
  --------------------------------------------------
  time only (M1)                 0.315          0 / 22
  + turn detection               0.798          0 / 22
  + corner bug fixed             0.951         25 / 25
```

Neither bug was reachable without an adversary. Both unit tests and the
wire-contract tests passed throughout, because both were testing one
movement, and the failure only exists in the seam between three.

---

## Tier 2 is the whole thesis, and it holds

`puppeteer-extra-plugin-stealth` is the standard answer to bot
detection. It patches `navigator.webdriver`, spoofs the plugin and
mimeType arrays, repairs `iframe.contentWindow`, fixes the WebGL vendor
strings, hides the CDP marker, and a dozen more.

Every one of those evasions answers the question *which browser is
this?*

None of them touch the mouse.

```
  tier1_playwright_naive     100% detected   mean score 0.781
  tier2_puppeteer_stealth    100% detected   mean score 0.846
```

Stealth provided **no protection whatsoever**. It scored marginally
*higher* than the tier with no evasion at all — noise at these sample
sizes, but the direction is not the point. The point is that the
industry-standard tool for defeating bot detection is invisible to a
detector that never asks what the browser is.

That is the argument the project was built to test, and it is the
cleanest result here.

---

## Tier 3 walks straight through

```
  tier3_undetected_chromedriver    0 / 17 detected
    mean events      2
    mean confidence  0.000
```

`undetected-chromedriver` evades completely. It is important to be
precise about *how*, because the summary row looks like the detector
failing and it is something worse.

Selenium's `ActionChains.move_to_element` does not interpolate. It jumps
the cursor to the target in one step, so across an entire session the
page sees **two `pointermove` events**. There is no path to measure.

The system reports this correctly: confidence 0.000, reason
`INSUFFICIENT_EVIDENCE`, decision `allow`. That is the cold-start
machinery working exactly as designed — it refuses to judge on no
evidence rather than guessing.

But a correct `allow` on a bot is still a bot getting through, and the
honest reading is:

> **This is not a true negative. It is a blind spot the confidence
> dimension is honest about.**

Two consequences follow, and the second is worse than the first.

1. **Evasion by absence is cheaper than evasion by mimicry.** Making a
   mouse path look human is hard. Not moving the mouse is free, and
   `pip install undetected-chromedriver` is the entire attack.

2. **Humans navigate by keyboard.** Tab, tab, type, Enter — a person who
   fills a login form without touching the pointer is indistinguishable
   from tier 3 under this feature set. Any policy that treats *low
   pointer evidence* as suspicious would flag them, which is the
   false-positive population that matters most. The only correct response
   is more evidence types, not a lower confidence threshold. That is M3.

---

## How the false-positive side will be reported

No human sessions exist yet, so the statistics were validated against a
synthetic fixture (`experiments/testdata/`, run with `analyze.py
--selftest`). Shipping untested statistics into a study you can only run
once would be careless; the fixture is explicitly not data and lives
outside the results directory so it cannot be mistaken for it.

The fixture encodes the structure the design expects — a few people who
flag often, most who never do — and the estimator recovers it:

```
  PRIMARY  5 of 20 people were falsely flagged at least once
           95% CI on the proportion of people: [11.2%, 46.9%]

  SECONDARY  session-level, adjusted for clustering within people
             rho (estimated from data) = 0.308
             design effect = 1.92   effective n = 41.6 of 80
             95% CI, cluster-adjusted: [4.8%, 24.3%]
             (unadjusted would report [6.0%, 20.0%] — too narrow)
```

Three commitments, enforced in code rather than intention:

**The person is the unit of analysis.** *"k of 20 people were falsely
flagged at least once"* sidesteps the clustering problem instead of
correcting for it, is directly interpretable, and cannot be inflated by
running more sessions per person.

**The clustering correlation is estimated, never assumed.** An earlier
draft of the plan put it at 0.2. That contradicted the argument sitting
next to it: if false positives concentrate in people with atypical motor
patterns, then being flagged is close to a property of the *person*, and
the correlation is high. As it approaches 1, effective sample size
approaches the number of people no matter how many sessions each
contributes — eight people running a thousand sessions each still gives
an effective n of about eight.

**There is no code path that prints a bare false-positive rate.**
Everything is an interval.

### Why it will still be the weak metric

Even run perfectly, three limits remain, and the third is not
statistical:

1. **Sample size.** Production systems operate near 0.1% FPR, which needs
   roughly 3,000 independent observations to bound. Twenty people at four
   sessions each supports an upper bound two orders of magnitude looser.

2. **Clustering**, above.

3. **Directional bias, which no interval captures.** The humans most
   likely to be falsely flagged are those whose motor patterns differ
   most from the norm: tremor, motor impairment, switch or eye-tracking
   input, non-dominant-hand use, very old and very young users. A
   volunteer pool drawn from one person's social graph systematically
   under-samples exactly that population.

   The figure will therefore be **optimistic in a known direction**, and
   that direction carries the highest human cost — blocking a human is
   far worse than admitting a bot.

Points 2 and 3 are one fact from two sides. If false positives were
spread evenly across people the correlation would be low and pooled
session counts would be honest. They are not spread evenly, which is
simultaneously why the correlation is high, why the person is the right
unit, and why the sample is biased.

---

## Tier 5: the detector's accuracy is a dial the adversary holds

The first four tiers use their library's default mouse. Tier 5 spends an
evening on it: Bézier curvature, a minimum-jerk velocity profile,
overshoot with correction, Gaussian tremor, Fitts's-law timing, and human
pauses between targets. About 140 lines.

Rather than ship it as one point, the curvature is a parameter and the
tier is run as a sweep. `bow` multiplies the perpendicular arc; 1.0 is a
shallow, plausible reach.

```
  cohort                     n  detected    rate   mean score   median   mean conf
  --------------------------------------------------------------------------------
  tier5_humanised_bow0.5     6         6  100.0%        0.944    0.954       0.729
  tier5_humanised_bow1.0    16        16  100.0%        0.942    0.950       0.732
  tier5_humanised_bow2.0    12        12  100.0%        0.784    0.822       0.718
  tier5_humanised_bow3.0    12        10   83.3%        0.663    0.771       0.732
  tier5_humanised_bow5.0    12         5   41.7%        0.325    0.491       0.744
```

Confidence sits at 0.72–0.74 across every level. Evidence volume is
constant and only the score moves, so the sweep isolates exactly one
variable.

**Detection rate is a smooth function of how much effort the adversary
spent on curvature.** It is not a property of the detector. Between
`bow` 2 and 5 — the difference between one plausible-looking arc and a
more exaggerated one — detection falls from 100% to 42%.

### And this is the part that matters

The sweep produces a curve with **no reference point on it.**

Nobody knows which `bow` value corresponds to a real human hand, because
the human data does not exist yet. A shallow arc at `bow` 1.0 scores
0.942 and is flagged every time. If that is what real reaching looks
like, the 0.90 straightness floor is not a detector — it is a machine for
flagging people.

So the honest reading of tier 5 being *caught* at low curvature is not
reassurance. It is evidence that **the threshold is unanchored**, and
that the same measurement which produces a 100% detection rate could be
producing an intolerable false-positive rate. Those two numbers are the
same number seen from opposite sides, and this project currently has one
of them.

That is what the human capture study is for, and it is why no threshold
should be calibrated before it lands.

---

## What these numbers do not say

- **Nothing about false positives.** See above, twice.
- **Nothing about a determined adversary.** Tier 5 stops at curvature.
  It does not replay recorded human paths — the telemetry-replay attack
  the contract names as the strongest against any system in this class
  and does not claim to solve.
- **Nothing about production traffic.** Loopback, one machine, headless
  Chrome, one page.
- **Nothing calibrated.** Every constant — the 0.90 straightness floor,
  the 60° turn, the confidence thresholds — is still an inception guess.
  M2 falsified the *segmentation*, not the *thresholds*; separating those
  needs the human distribution.

---

## Reproducing it

```bash
cd services/ingestion && make run          # terminal 1
cd experiments
npm install --registry=https://registry.npmjs.org
python3 -m venv .venv && .venv/bin/pip install undetected-chromedriver selenium setuptools
.venv/bin/python run.py
python3 analyze.py
```

Three environment obstacles cost more time than the experiment and are
recorded so they cost the next person none:

- **npm** was configured against a local Verdaccio proxy at
  `localhost:4874`. With it down, npm hangs for fifteen minutes and then
  fails with `ECONNREFUSED`. The `--registry` flag overrides it per
  command.
- **`undetected-chromedriver` imports `distutils`**, removed in Python
  3.12+. `setuptools` ships a shim, enabled with
  `SETUPTOOLS_USE_DISTUTILS=local`; `run.py` sets it automatically.
- **ChromeDriver outran Chrome** (151 vs 150) and refused to attach. The
  tier now pins the major version to the installed browser.

A tier whose dependencies are missing is recorded as **ABSENT** rather
than skipped. Four tiers listed and three run reads as "we tested four
things", and the missing one is invariably the one that would have found
something — which is precisely what tier 3 turned out to be.

---

## Next

**Human capture**, which is now the only thing standing between this
project and a real number. Tier 5 turned the missing false-positive rate
from an incompleteness into the blocking dependency: without it the
straightness floor cannot be calibrated, and an uncalibrated floor makes
every detection rate above meaningless.

**M3.** The adversary has written its agenda. Keystroke timing, scroll
dynamics and focus events exist primarily to close the tier-3 blind
spot: a session with no pointer data must still be judgeable. The same
evidence makes keyboard-navigating humans judgeable, which is the
strongest signal it is the right fix — one change serving both the
detection gap and the fairness gap.
