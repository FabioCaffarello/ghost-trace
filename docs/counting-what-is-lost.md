# Counting what is lost

*Phase 3, eight pull requests, 2026-08-05. What it found, what it cost,
and the two things it got wrong on the way.*

---

Phase 2 split one binary into four deployables and passed its gate at
p99 1.393ms against an 80ms budget. It also produced a number nobody had
asked for: the apparent 4× speedup was the **archive backend**
(−5.99ms), not the decomposition (+0.94ms). Decomposition cost
performance and bought something else.

Phase 3 started from a less comfortable observation. The system had
three ways to lose a record, and could not report on any of them.

## The starting position

- The **collector** dropped best-effort writes on a deadline and
  logged it.
- The **archive** refused records whose hash did not describe them, and
  logged it.
- A record could **age out** of the stream unarchived, and nothing at
  all happened.

None of these were bugs. Every one was a deliberate design decision with
a comment explaining it. The problem was that a deployment losing
records and a deployment losing nothing produced identical output, and
so did a deployment whose counting code had never been wired up.

That last case is not hypothetical. It is the shape of the repository's
own founding rule — *absence is never zero* — applied to a running
system rather than to a results table.

## What got built

| | |
| --- | --- |
| **3.1** | One `prometheus.Registry` per process. `libs/metrics`, and the ADR admitting it is the one shared library allowed a dependency. |
| **3.2** | The collector counts every drop by kind and reason, and declares every series at zero before serving. |
| **3.3** | The archive counts commits and refusals, and reports how far behind it is — holding the last real reading when the broker cannot be reached, with a timestamp that stops advancing so staleness announces itself. |
| **3.4** | Age-out gets an early warning. And **cannot** get a count. |
| **3.5** | Two cross-service invariants become checks rather than comments. |
| **3.6** | The archive writes down where it is, durably, and `make loss-audit` reconciles. 3.4's missing count arrives with it. |
| **3.7** | Everything blocks a merge; the topology is gated; releases exist. |
| **3.8** | [ADR-0008](../contract/decisions/0008-what-a-zero-is-allowed-to-mean.md), and this. |

## The interesting failure

3.4 was supposed to count age-out. It shipped an early warning and an
explicit admission that it could not count anything, which at the time
read like a retreat.

Two mechanisms were built and deleted, both after measurement rather
than after argument:

**The stream's first sequence against the consumer's ack floor.** This
should reveal records that left without being acknowledged, and it
survives a restart. It does not work: the broker *advances* the ack
floor when messages are removed from under a consumer, because there is
nothing left to acknowledge. Purging four unconsumed records left
`first_seq=15`, `ack_floor=14`.

**Jumps in the delivered sequence.** This should reveal the same thing
from the consumer's side. It does not either: a gap requires records to
leave without being delivered, and a consumer that keeps up never sees
one. A test built specifically to force it delivered everything before
the purge landed.

The generalisation only became visible with both failures side by side:
**both read a number the broker maintains, and the broker rewrites
exactly those numbers when it discards records.** The evidence is
destroyed by the event it would evidence.

Two months of dashboard would have shown a flat zero and been believed.

## The fix, and why it is small

The archive stopped asking the broker where it had got to, and started
writing it down.

Four numbers — first sequence, highest sequence, commit operations,
refusals — in the archive's own SQLite database, in the **same
transaction** as the record. Nothing outside the process can move them.
`stream_skipped` is then the stream's first surviving sequence minus the
archive's own mark, and it is a subtraction rather than an inference.

It fires. With the archive stopped and the stream purged before it
returned:

```
position_highest_sequence 250
stream_first_sequence     311
stream_skipped             60
```

`311 − 250 − 1 = 60`.

The same durable position makes the reconciliation possible at all:

```
span        = highest_seq - first_seq + 1
accounted   = committed + rejected
unaccounted = span - accounted
```

`make loss-audit` drives traffic, takes the archive away, takes the
broker away, and asserts that after each the books balance. Against the
real compose topology, they do.

## Two things this phase got wrong

**The gauges were declared at zero.** Every counter in the phase is
declared at zero at construction — that is the whole point, since a
counter that never fired is indistinguishable from one nobody wired up.
The position gauges were given the same treatment by analogy, and the
analogy is backwards: a gauge that has never been read is *unknown*, and
zero is a specific claim that happens to be false. A fresh archive would
have published `unaccounted 0`, which reads as a perfect run.

The tests written to assert the rule are what caught it, which is the
only reason it is a footnote rather than a finding.

**A required check that would have passed while verifying nothing.** The
summary jobs added in 3.7 read `join(needs.*.result, " ")`. Double
quotes are a syntax error in a GitHub expression, and both workflows
were rejected at zero seconds — loudly, which was luck. Had the
expression merely evaluated to *empty*, the loop would have iterated
over nothing and gone green, and branch protection would have reported a
guarded branch with nothing guarding it.

Both summary jobs now count the results they read, and
`scripts/check-workflows.py` asserts statically that every job is
required, with nine selftest cases each proven against its own mutation.

## What is still true and uncomfortable

- **Nothing here has been near load.** Twenty-five records per scenario
  proves the accounting is *sound*. Whether it holds at rate is
  untested, and is the first item on the debt register.
- **The browser end is uncounted**, by design. The SDK is
  fire-and-forget and there is no channel back; inventing one to count
  losses would be inventing something to distrust.
- **The false-positive rate still has no human data.** It remains
  `null`, and it is the number the project's central claim most needs.
  It is calendar-bound and should be running in parallel with everything
  above rather than after it.
- **`unaccounted` counts records in flight.** It is read after traffic
  drains, not during, and a reader who forgets that will see loss that
  is about to resolve itself.

## The rule that came out of it

Written down as [ADR-0008](../contract/decisions/0008-what-a-zero-is-allowed-to-mean.md),
but the version worth carrying around is shorter:

> Never measure a failure using state maintained by the component that
> fails.

Everything else in the phase is an application of the rule the front
page already had.
