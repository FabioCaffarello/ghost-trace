# Ghost Trace v1 — Retrospective

Written at the moment of deletion, with the code still on screen. The full v1 tree
is preserved at the git tag `v1-archive`.

---

## What v1 set out to build

A behavioral intelligence substrate whose primary commitment was **preserving the
distinction between what was observed and what was inferred** as operational
knowledge accumulates and gets acted upon.

The diagnosis behind it is sound, and worth restating because none of what follows
retracts it: anti-abuse systems degrade in a characteristic way. An inference made
on Monday becomes an input on Tuesday and a fact by Friday. Nobody decides to
confuse a guess with an observation; it happens structurally, because the storage
layer treats them identically. Systems in this class do rot from the epistemology
outward.

v1's answer was to make the separation constitutional. Four categories —
observations (immutable), operational constructs (deterministic derivations),
hypotheses (probabilistic), decision audits — with provenance as first-class
structure, and belief split into two dimensions (confidence, evidential
independence) so that a claim could never inflate by citing itself.

It was built constitution-first, in that order: Charter, then Ontology, then
Architecture, then Schemas, then Services.

**What that produced, at the point of deletion:**

| | |
| --- | --- |
| Commits | 365 |
| Documentation | 27,651 lines |
| Decision log | 10,776 lines, 222 numbered entries |
| Charter | 397 lines, frozen v0.7 across five committee redactions |
| Proto schemas | 58 (25 of them hypothesis-lifecycle events) |
| Go source | ~22,500 lines, plus ~26,000 lines of tests |
| CLI binaries | 53 |
| Governance infrastructure | 3,356 lines under `.claude/` |
| Detection rate, false-positive rate, latency | **none** |

That last row is the retrospective in one line.

---

## Why the abstraction was wrong

Six things, ordered from the most specific to the most general. The first is a
modelling error. The last is a method error, and it is the one that mattered.

**1. The category model was right; the entity model was wrong.**

Separating observation from inference is load-bearing and survives into v2. But v1
made a hypothesis a *durable entity with a lifecycle* — formed, promoted, demoted,
merged, split, dissolved. Six operations across four subtypes: 25 protos, ~6,000
lines in `internal/hypothesis`, ~3,100 in `internal/projection`, and 53 CLIs to
drive them.

That is the right model for a claim that outlives any session — *"this set of
actors is an automation group"* is a belief you hold for months and must be able
to retire. It is the wrong model for the object a behavioural detector actually
centres on: *"does this session look human?"* is a scalar that dies when the
session does. It is never promoted, never merged, never dissolved. The entire
lifecycle apparatus was built for the wrong noun.

**2. The direction of derivation was inverted.**

There was no integration contract. The external surface — what a host application
would actually call — was never written down, so nothing constrained the internal
model. Every design question was answerable only by more deliberation, and
deliberation over an unconstrained space does not converge.

An external contract is a forcing function. Four endpoints, six event types, one
latency budget: it makes surplus *visible*. Without it, the question "is this
necessary?" had no test, and so everything passed.

**3. Deliberation had no terminating condition.**

222 numbered entries, and resolving a question reliably produced new ones: Q1 →
Q3, OMQ#2 → OMQ#3, Q3 → Q5 → Layer B. Each cascade was rigorous. Each was
*genuinely entailed* by the framework — that is the part worth sitting with. The
work was not sloppy; it was correct work inside a system with no fixed point.

A framework where answering a question generates further questions terminates only
when something outside it intervenes. Running code intervenes. Documents do not.

**4. The enforcement infrastructure outgrew the thing enforced.**

Pre-commit hooks parsing frozen-section line ranges, a `charter-guardian` skill,
an `epistemic-auditor` agent, CI jobs for subordination and glossary coverage.

And the detail that says everything: **the git hook was never installed.**
`.git/hooks/pre-commit` absent, `core.hooksPath` unset — verified again at
deletion. This was known and recorded (§0214). Of the three enforcement tiers, the
one designated as the source of truth never ran. Discipline was preserved by an
agent manually invoking the script.

The ceremony was elaborate; the mechanism was mostly absent; the gap went
unnoticed for months because nothing depended on it working.

**5. When the system finally ran, reality falsified it in hours.**

The first real deployment against CIC-IDS-2017 produced, in sequence: a CSV
header-whitespace bug (§0207), a `jq` ARG_MAX overflow (§0211), a JSON field-name
mismatch that broke the provenance chain outright (§0214).

None of these was reachable by deliberation. All were found within hours of
execution. Three years of constitutional reasoning had nothing to say about
`strings.TrimSpace`.

This is the empirical verdict on the method, and it arrived from the method's own
records.

**6. The measurable claim was never made.**

After 365 commits, the system could form, promote, demote, merge, split and
dissolve hypotheses about automation groups. It could not say whether it caught a
bot. No adversary existed, so no number was falsifiable, so no claim was made.

The single most damaging property of v1 is not any of the above. It is that
**nothing in it could have been wrong.** A system with no measurable output cannot
fail, which sounds like safety and is actually the absence of information.

---

## What survived

Kept, and load-bearing in v2:

- **`internal/canonical`** — deterministic protobuf serialization + BLAKE3 content
  hashing (336 lines, 1,897 lines of tests). A decision must be reproducible from
  its record; this is exactly that.
- **`internal/substrate`** — append-only SQLite + content-addressed blob store.
  Correct as the raw event archive. Explicitly not the online path.
- **`internal/ingest`** — idempotent append keyed by content hash. Telemetry
  batches arrive out of order and get retried; hash-idempotency is right unchanged.
- **The paired-dimension rule** — never emit a belief without its evidence
  measure, enforced at the marshalling boundary. v2 renames the axes to
  `score`/`confidence`. This is the strongest single artifact v1 produced, and it
  answers cold-start, which most detectors handle badly.
- **The observation / derivation / inference distinction** — as roughly 200 lines
  of types and a naming discipline. Not 6,000 lines of lifecycle.

The distinction survives because two v2 requirements cannot be met without it: an
evaluation record must be reinterpretable after the model is recalibrated, which
requires knowing which parts are facts and which are judgements; and
`score`/`confidence` is the paired-dimension rule restated.

**So the categories were never the problem. The apparatus built to enforce them
was.**

---

## The two rules that replace all of it

Everything above compresses into two tests, both cheap to apply and both hard to
argue with:

> **A boundary earns its keep when something on the other side of it gets
> revised.** Features get recalibrated. Policies get retuned. Observations never
> change. That is three types, not a lifecycle.

> **A document earns its keep when it describes something that has run.**

And the correction that generates both: **the contract is upstream; the internals
are downstream.** v1 derived its external surface from its internal model and
never finished. v2 starts from four endpoints and a latency budget, and anything
that does not serve them is visible as surplus on sight.

---

## A note on tone

It would be easy to read this as a repudiation. It is not.

The diagnosis was correct and remains correct — that is why the categories, the
paired dimensions, and the hashing survive intact. What went wrong was
proportionality and order: a correct diagnosis met a response scaled to a system
that did not exist yet, sequenced so that the part which could have said *"stop,
this is unnecessary"* was built last.

The failure was not rigour. It was rigour applied before there was anything to be
rigorous about.
