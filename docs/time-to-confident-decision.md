# Time to confident decision

M3. Five more event types, per-action scoring, and the §9 number almost
nobody publishes.

---

## The number

How long before the system knows enough to act at all?

```
  confidence >= 0.40  (challenge becomes possible)
    reached in 30/30 sessions
    median   2 batches   56 events   3.7s of session

  confidence >= 0.70  (block becomes possible)
    reached in 30/30 sessions
    median   3 batches   84 events   5.7s of session
```

Measured by asking for a decision after *every* telemetry batch rather
than once at the end. That costs nothing, because a decision is a read
of maintained session state rather than a computation over history —
which is the architecture claim, incidentally falsified-or-not by
whether this measurement was cheap to take. It was.

The practical reading: a login flow that takes a user four seconds is
right at the edge of judgeable, and one that takes under two is a cold
start no matter how bot-like it looks. That is not a defect to be tuned
away. It is the cold-start problem stated in the units that matter, and
the confidence dimension exists so that consumers see it rather than
receive a confident-sounding guess.

---

## M2 wrote this milestone's agenda

The harness found that `undetected-chromedriver` evaded completely — not
by looking human but by producing almost nothing to look at. Selenium
jumps the cursor, so the page saw two `pointermove` events across an
entire session.

The fix was never a lower confidence threshold. It was more evidence
that does not depend on the pointer.

```
  tier                            M2 (pointer only)      M3 (five channels)
  ------------------------------------------------------------------------
  tier1_playwright_naive          25/25   100%            6/12    50%
  tier2_puppeteer_stealth         25/25   100%           12/12   100%
  tier3_undetected_chromedriver    0/17     0%            8/8    100%
  tier4_synthetic_linear         200/200  100%           60/60   100%
  tier5_humanised bow1.0          16/16   100%           10/10   100%
  tier5_humanised bow5.0           5/12    42%           10/10   100%
```

**Tier 3: 0% → 100%.** Evidence per session went from 2 events to 50.
Selenium's `send_keys` emits keystrokes at a fixed cadence, and fixed
cadence is the single most legible thing a keyboard can produce.

**Tier 5 at high curvature: 42% → 100%.** The humanised-mouse adversary
spent its effort entirely on the pointer. Typing rhythm was untouched,
so the channel it did not think about is the one that caught it. That is
the argument for multiple evidence types in one line: an adversary must
defeat *all* of them, and they are cheap to add relative to the cost of
faking each.

**Tier 1: 100% → 50%, and that is a real loss.** Playwright's
`page.fill()` sets a field's value directly rather than typing it, so
the tier emits no key events at all. It is now partly evading by
absence on the keystroke channel exactly as tier 3 did on the pointer —
the same failure mode, relocated. Its pointer score also sits near the
straightness floor, so individual sessions fall on both sides of the
challenge threshold.

The honest summary is that M3 closed one blind spot, materially raised
the cost of the humanised attack, and exposed that *set a value without
typing it* is the next one.

---

## The bug this milestone produced, and what it taught

The first M3 implementation made confidence a weighted sum across
channels with weights summing to 1.0. It looked obviously right and it
broke two tiers that had previously been detected 100% of the time.

With `login` weights of pointer 0.35 / keystroke 0.50 / interaction
0.15, a session holding only pointer evidence can reach at most 0.35
confidence — below the 0.40 challenge floor **by construction rather
than by evidence**. Tier 4, with a perfect linear path and 360 pointer
events, was allowed through because it had not typed.

The mistake was conflating two different questions:

- **How much evidence is there?** → confidence
- **How many independent things agree?** → corroboration

Separating them fixed it. Confidence now normalises by the single most
relevant channel, so it reaches 1.0 when you hold as much weighted
evidence as the action's dominant channel could supply alone; missing
channels still cost confidence but no longer make a verdict
unreachable. Corroboration became a separate decision-rule requirement:
**a block needs at least two contributing channels.**

That second rule is v1's evidential-independence idea surviving into v2
in a form that earns its keep — a belief must not inflate on one source,
however strong that source looks. In v1 it was a proto-reflection walk
over hypothesis records. Here it is one integer in the decision rule,
and a test that fails if a single channel can produce a block.

---

## What the five new channels actually measure

All timing and structure. No content, ever.

| Channel | Signal | Why it is hard to fake |
| --- | --- | --- |
| Keystroke flight CV | Irregularity of inter-key gaps | Human typing is erratic; a fixed `delay` is not, and jittering it convincingly means modelling a typist |
| Key dwell | How long keys are held | Most automation emits down/up back-to-back, so keys are never held at all |
| Identical intervals | Byte-identical consecutive gaps | A person cannot produce two identical gaps, let alone twenty |
| Programmatic scroll | Page scrolled itself | Close to categorical — a person cannot issue one |
| Focus transitions | Fields receiving focus | A form completed with zero focus events was not filled by someone tabbing or clicking |

Modifier keys are excluded from rhythm: Shift is held *across* the key
it modifies, and counting it would inflate dwell and corrupt flight.
Gaps beyond two seconds are excluded as thinking rather than typing —
including them makes any real typist look maximally variable, which is
the direction that hides bots.

---

## Per-action scoring

Contract §8.6, implemented: **state is per-session, scoring is
per-action.**

There is one pointer trace and one typing rhythm per session, so the
accumulators are session-scoped. But which evidence is *relevant*
depends on what the application is about to do. A `login` weights typing
at 0.50 because there was typing; a `checkout` weights it at 0.20 and
the pointer at 0.55, because there may have been none.

A session with 200 pointer points and zero keystrokes is genuinely
confident for a checkout and genuinely thin for a login, and reporting
one session-level number would erase that distinction — producing the
worst available failure, a confident-looking block on evidence that does
not bear on the action taken.

---

## Still uncalibrated, still no false-positive rate

Every constant here remains an inception guess. M2 falsified the pointer
*segmentation*; M3 added channels. Neither calibrated a threshold,
because calibration needs the human distribution and the human
distribution does not exist yet.

The tier-5 sweep made the stakes concrete: a plausibly-curved path
scores 0.942 against a 0.90 floor. If that is what real reaching looks
like, the floor is a machine for flagging people. Every number in this
document describes how the system behaves against *bots*, and says
nothing about what it does to the humans it is meant to let through.

Recruitment remains the binding constraint, and it is the only thing
here that more engineering cannot accelerate.

---

## Next

- **Human capture.** Everything else is blocked behind it.
- **Tier 6:** `page.fill()`-style value injection, which M3 just showed
  is the cheapest remaining evasion — and the sibling of the tier-3
  attack, one channel over.
- **M4:** durable session state, `/v1/outcomes`, and the two-architecture
  benchmark that decides whether maintained state was worth it.

---

## Addendum: tier 6, and the shape of every remaining attack

Tier 6 combines the two evasions the harness has already proven work,
one per channel: tier 5's humanised mouse, and `page.fill()` instead of
typing. It costs nothing beyond putting two existing things together.

```
                                  detected   score   confidence
  without the counter-signal        4 / 10    0.442       0.664
  with the counter-signal           9 / 10    0.713       0.809
```

The counter-signal is that **declining to type is itself observable**. A
field whose contents change with no keystroke and no paste behind it was
not filled by a person. It is tracked per field, so typing in one box
while another is filled programmatically is still caught.

Tiers that genuinely type were unaffected: tier 2 stayed at 8/8, tier 5
at 8/8.

### The pattern worth naming

Every channel closed so far was closed against an adversary attacking
**one channel at a time**. Tier 6 is the first that attacks two, and it
does so by *declining to produce evidence* on each rather than faking
either. Faking is expensive; abstaining is free.

So the useful generalisation is not "add more channels". It is that
**absence of evidence on a channel where evidence is expected is itself
evidence** — which is what caught tier 3 (no pointer), and now tier 6
(no keystrokes). The detector's job is less to recognise bot-like
behaviour than to notice which human traces are missing.

### A signal with a known false-positive population

`VALUE_INJECTED` says a person cannot change a field's contents without
typing, pasting, or autofill. That is very nearly true and not quite.

**Speech-to-text dictation, IME composition (Chinese, Japanese, Korean),
and some assistive input devices produce input events with no preceding
keydown.** Under this rule they look like injection.

That is precisely the population this project has said from the start it
is most worried about mis-flagging, and it is now one of the strongest
signals in the system. The signal ships because the alternative — a free
evasion — is worse, but:

- it is capped below categorical, unlike programmatic scroll;
- it cannot produce a block alone, because blocks require two channels;
- and the capture study needs dictation and IME sessions before anyone
  considers raising its weight.

Written into the source at `policy.ReasonValueInjected`, not filed in a
backlog, because the code is where the next person will read it.
