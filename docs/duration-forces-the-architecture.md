# Duration forces the architecture, not concurrency

M4. The two-architecture benchmark, and a correction to the plan that
commissioned it.

**This file was going to be called `concurrency-forces-the-architecture.md`.
The measurement says concurrency does not force it. Duration does.**

---

## The grid

Maintained state versus compute-on-call, same feature extractors, same
policy, same locking. The only difference is *when* the arithmetic
happens.

```
architecture    sessions  duration    heap MB   bytes/sess    p99 ms
------------------------------------------------------------------------
maintained           100       10s        0.3          733     0.001
on-call              100       10s        0.6        4,059     0.007
maintained         1,000       10s        1.0          851     0.002
on-call            1,000       10s        4.2        4,190     0.005
maintained        10,000       10s        8.1          826     0.001
on-call           10,000       10s       39.9        4,153     0.007

maintained           100      300s        0.3          732     0.001
on-call              100      300s        5.9       58,973     0.040
maintained         1,000      300s        1.0          851     0.001
on-call            1,000      300s       56.6       59,097     0.038
maintained        10,000      300s        8.1          825     0.001
on-call           10,000      300s      563.5       59,066     0.040

maintained           100     1800s        0.3          732     0.001
on-call              100     1800s       27.1      281,180     0.302
maintained         1,000     1800s        1.1          852     0.000
on-call            1,000     1800s      268.5      281,299     0.193
maintained        10,000     1800s        8.1          826     0.001
on-call           10,000     1800s    2,682.7      281,273     0.293
```

## The claim, confirmed

```
  bytes per session      10s       300s      1800s
  maintained             826        825        826      ← flat
  on-call              4,153     59,066    281,273      ← 68x
```

**Maintained state is constant in session duration.** 826 bytes at ten
seconds, 826 bytes at thirty minutes. Its cost is `O(sessions)`, exactly
as predicted, and the entire 10,000-session heap is 8.1 MB whether the
sessions last ten seconds or half an hour.

**Compute-on-call is not.** It grows 68× across the duration axis
because it must retain the raw event stream — it cannot know in advance
which features the decision will want, which is the whole point of
deferring the computation.

At the headline cell — **10,000 concurrent sessions of 30 minutes** —
the two designs sit at **8.1 MB against 2.68 GB**, a factor of 331.

---

## Three things the measurement got right, and three it got wrong

The prediction was written down before the run, which is what makes it a
prediction. Scoring it honestly:

**Right: the diagonal.** Divergence appears and grows along it, and the
shape is exactly `O(sessions)` versus `O(sessions × duration)`.

**Right: the axis.** Duration, not concurrency, is what separates them.
Read the columns: at fixed duration, both architectures scale linearly
in sessions and the *ratio* between them never moves. Concurrency
multiplies whatever per-session cost you have; it does not create the
gap.

**Right: memory, not CPU.** Confirmed below, and more strongly than
intended.

**Wrong: "bottom-left, no difference."** Even at 100 sessions of 10
seconds there is already a 5.5× per-session gap (733 vs 4,059 bytes). It
is invisible in absolute terms — 0.3 MB against 0.6 MB, which is nothing
— but the ratio is present from the first cell. The gap does not appear
along the diagonal; it is there immediately and only becomes *material*
along the diagonal.

**Wrong: "top-right fails outright."** It did not fail. It allocated
2.68 GB and kept answering, with a p99 of 0.293 ms. A large box has
64 GB, so compute-on-call at this load is expensive rather than
impossible.

**Wrong by 2×: the estimate.** REFACTOR-PLAN §8.1 predicted ~5.5 GB at
this cell from back-of-envelope arithmetic. The measured figure is
2.68 GB. The estimate was the right order of magnitude and the wrong
number, which is the ordinary fate of estimates and the reason the
benchmark exists.

---

## The finding that matters most

**Latency never forces this decision. Not at any cell in the grid.**

```
  compute-on-call p99, worst cell (10,000 x 1800s):   0.293 ms
  the budget (contract §5, REFACTOR-PLAN §8.1):      80     ms
```

Compute-on-call is **273× inside the latency budget** at the worst point
measured. Replaying half an hour of retained events through the feature
extractors costs a third of a millisecond, because the extractors are
running sums over a few thousand points and that is simply cheap.

This closes an argument that has run through the whole project. The
original plan justified maintained state on p99. That was corrected once
— to concurrency — and the correction was accepted. The measurement now
says both framings were wrong:

> Maintained state is not justified by how fast a decision is. It is
> justified by how much memory you are holding while you wait to be
> asked.

An honest article says this plainly, because the tempting version —
*"tight latency forces stream processing"* — is the version the
measurement disproves, and it is the version most articles in this genre
publish.

---

## What this does not license

- **It does not justify maintained state for short sessions.** At 10
  seconds, the difference at 10,000 concurrent sessions is 40 MB against
  8 MB. Nobody should build a streaming architecture for 32 MB. A
  login-only deployment should compute on call and skip all of this.
- **It only matters because §8.5 says collection continues.** The
  architecture is earned by the decision to keep scoring after the
  decision point, which is what makes 30-minute sessions real. Those two
  choices stand or fall together.
- **It is one process on one machine.** Nothing here measures a deploy
  surviving, state spanning replicas, or GC behaviour under real
  allocation churn. Those are the reasons the maintained-state design is
  *hard*, and this benchmark says nothing about them.
- **The on-call variant is not a strawman, but it is naive.** A real
  implementation would cap retention, downsample old events, or
  checkpoint partial aggregates — at which point it becomes maintained
  state with extra steps, which is arguably the actual conclusion.

---

## Also in M4: the label channel

`POST /v1/outcomes` completes the contract's four endpoints.

Without it there are no labels; without labels there is no calibration;
without calibration the false-positive rate is a guess. It is the
difference between a detector and a platform.

Design commitments, each with a reason:

- **Unknown labels are rejected, not stored.** A typo'd outcome is worse
  than a missing one, because it silently degrades the calibration
  everything else depends on.
- **A label with nowhere durable to live is refused with 503.** Accepting
  it would let the caller believe it had reported an outcome, and the
  loss would stay invisible until calibration time.
- **Outcomes are Category I observations** — immutable, never revised. A
  later, better outcome for the same evaluation is a *new* record, and
  the join resolves by time order rather than by mutation.
- **`observed_at` and `recorded_at` are separate fields.** The first is
  the application's clock and the second is the server's. The gap between
  them is worth watching.

---

## Still missing, four milestones in

No human sessions. Every number in this document describes machines.

The benchmark measured the architecture; it did not measure whether the
system is fair to the people it is meant to let through. That remains
the binding constraint, and it is the only item in the project that more
engineering cannot accelerate.

---

## Next

- **Human capture.** Everything else is blocked behind it.
- **Durable session state across restarts and replicas**, which is where
  the maintained-state design gets genuinely hard and where the transport
  choice finally earns its place. Deliberately not done here: nothing in
  this grid needed it, and installing infrastructure to satisfy a plan
  rather than a measurement is how v1 went wrong.
- **M5**: the README with six reproducible numbers.
