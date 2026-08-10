# RFC-0001 — human-study data governance

**Status:** proposed · **Date:** 2026-08-10 · **Gates:** recruiting
participants for the capture study (roadmap 4.P2, and number 2)

> This document is a **proposal**. Nothing in it is in force. The three
> questions below are open in `experiments/PARTICIPANTS.md` today, and
> they stay open until this is accepted and the mechanism §4 describes
> is built. Accepting it is not a technical act: it commits a person to
> obligations toward other people, and it is the maintainer's to make.

## Context

Number 2 — the false-positive rate on human traffic — is `null`, and it
governs every other number in the repository. A detector that flags
every session scores 100% against every adversarial tier; until number 2
exists, number 1 is an upper bound on nothing. The README says so in its
own voice.

Getting it needs people: roughly twenty of them, five minutes each. That
is calendar-bound rather than effort-bound, which is why the roadmap
runs the capture track **alongside** a phase rather than after one.

Everything technical for that is built. The statistics report
person-level counts with an intraclass correlation estimated from the
data, and no code path anywhere emits a bare session-level rate. The
capture instrument records labelled sessions. `PARTICIPANTS.md` holds
the consent script in a file of its own, and `disclosure_test.go`
compares it against the vocabulary the SDK actually emits in both
directions, so a collected value nobody was told about fails the build.

What is missing is not code. `PARTICIPANTS.md` names it:

> - **Retention.** No retention period has been decided or promised.
> - **Deletion on request.** There is no implemented mechanism.
> - **Who holds the data.** A single maintainer, on one machine. There
>   is no institutional review, no data controller, and no third party.

Three unanswered questions, each of which becomes a promise the moment
somebody is handed the script. This RFC proposes answers.

## What the data actually is

Deciding retention without being precise about the material is how
policies end up either theatrical or negligent. A captured row holds:

- a **participant code** the volunteer was given, and can discard;
- **arm, condition, visit index** — the experiment's own labels;
- the **decision** the engine returned, its score and confidence;
- **event count and duration**;
- a **timestamp**.

The behavioural detail — pointer geometry, key timing, the coarse key
class — reaches the archive as an `Evaluation`, keyed by the same
participant code through `subject_id`. Key *content* and field *values*
are never collected anywhere, which `disclosure_test.go` and the SDK's
own structure both hold.

Two properties follow, and they pull in opposite directions:

**It is not identifying on its face.** No name, no email, no IP
retained, no cookie, no fingerprinting surface, no field contents. The
participant code is a pseudonym the volunteer chose to accept.

**It is not anonymous either.** Motor dynamics are the *subject* of the
measurement — the whole claim is that how a person moves is
distinctive enough to separate them from a machine. A corpus of
distinctive motor patterns linked by a stable pseudonym is not
identifying today and is not safely assumed to be un-identifying
forever. This is the argument for a retention *ceiling* rather than a
"delete when we remember" convention.

## 1. Retention

**Proposal.**

| what | kept for |
| --- | --- |
| raw captured rows (`human_sessions.jsonl`) and the evaluations they reference | until the figure they support is published, **plus 90 days**, and never more than **12 months** from capture |
| the published aggregate — rate, interval, person count, ICC, ± effect sizes | permanently, in `docs/results/` |

The 90 days exist so a published number can be **challenged and
re-derived** — a measurement project that deletes its evidence the
instant it publishes cannot answer "show me". The 12-month ceiling
applies regardless: if a study stalls and never publishes, that is not a
reason to keep a corpus of one person's motor patterns indefinitely.

**Raw rows are never committed.** `experiments/.gitignore` already
excludes `results/*.jsonl`, so this codifies what the tree does. A git
history is not something a retention period can be applied to, so the
only workable rule is that the rows never enter one.

*Rejected: indefinite retention.* It is the default that happens when
nobody decides, it is what makes the 12-month ceiling worth writing
down, and "we might want to re-analyse" is a reason that never expires.

*Rejected: delete-on-publish with no window.* It makes the central claim
uncheckable, which is the one thing this repository will not trade.

## 2. Deletion on request

**Proposal.** A participant can have their rows removed by asking, with
no reason given and no negotiation, and the mechanism is a target rather
than a promise to remember:

```
make forget P=p07
```

It must:

1. rewrite `results/human_sessions.jsonl` without that participant's
   rows, and delete the matching archive records by `subject_id`;
2. append to a deletion log the **participant code, the date, and the
   number of rows removed** — never their content, so the log does not
   reconstitute what it records;
3. be idempotent, and say plainly when a code matched nothing.

**It cannot retract a published aggregate, and the script must say so.**
If a number was computed from a corpus including that person and is
already committed under `docs/results/`, deleting the rows does not
recall the manifest. A person deciding whether to take part is entitled
to know that before they start, not after they ask. This is a real limit
and dressing it up would be worse than stating it.

**Implementation is a precondition of recruitment, not a follow-up.**
An unimplemented deletion mechanism is indistinguishable from no
deletion mechanism at the only moment it matters.

## 3. Custody

**Proposal, stated as a limitation rather than a structure.**

The data controller is the single maintainer, on one machine, with no
cloud storage, no third-party processor and **no institutional review
board**. There is nobody to appeal to and nobody independent reviewing
the design. That is the honest description, and it constrains what the
study is permitted to be:

- **Adults who can consent for themselves.** No minors, and nobody whose
  circumstances make declining awkward — which specifically includes
  anyone in a reporting relationship to the maintainer.
- **Recruitment through personal networks and public invitation only.**
  No panels, no incentives, no recruitment inside an organisation whose
  members might read participation as expected.
- **No deception of any kind**, including by omission. The script is the
  whole disclosure, and `disclosure_test.go` is what keeps it whole.
- **No secondary use.** These rows exist to produce number 2 and the
  arm-A effect sizes. They are not a training set, and if that ever
  changes it is a new consent, not a new paragraph.
- **A breach is disclosed** to every participant reachable by the
  channel they were recruited through, and recorded in `docs/`.

If the study ever wants to be more than this — more people, an
institution, publication beyond this repository — that is a different
governance question and a successor RFC, not an amendment.

## 4. What must be true before anyone is recruited

The gate, as a checklist, so that "are we ready" has an answer that is
not a judgement call:

- [ ] this RFC is **accepted**, and `contract/rfcs/README.md` says so;
- [ ] `make forget` exists, is tested, and has been run once against
      synthetic rows;
- [ ] the consent script names the retention period, the deletion right,
      and the limit in §2 — that a published number cannot be recalled;
- [ ] `disclosure_test.go` is green, so what the script describes and
      what the SDK collects still agree in both directions;
- [ ] roadmap 4.P2 has run: the capture protocol end to end against
      synthetic participants, so recruiting is the only step left that
      needs a human.

Until every box is ticked, `PARTICIPANTS.md` keeps saying the script
should not be handed to anyone, and the test keeps failing if it stops
saying so.

## 5. A change to what is collected invalidates consent for what follows

The disclosure tests bind the script to the SDK's vocabulary, which
keeps the script *true*. They cannot keep it *agreed*: a volunteer
consented to the script as it read on the day they read it.

**Proposal.** Widening the collection surface mid-study — a new event
family, a new session-start field, a new class — requires that
participants be shown the changed script before any further session is
captured from them. Sessions already captured are unaffected; they were
taken under a disclosure that was accurate at the time.

The `consent-and-collection-surface` rule in
`.context/config/policy.json` already routes such a change through
approval. This says what the approval is *for*.

## Open questions this RFC does not answer

- **Where the deletion log lives** so that it survives the machine but
  does not become a second corpus. A file beside the rows is the obvious
  answer and inherits their fragility.
- **What happens to arm A's habituation contrast** if a participant with
  forty sessions withdraws — a per-person effect size is not
  recomputable without that person, and the aggregate may become
  unpublishable rather than merely smaller.
- **Whether 20 people is enough to publish a rate at all**, which is a
  statistics question the analysis layer already answers conservatively
  (person-level counts, ICC estimated from data) and not a governance
  one. Named here so it is not mistaken for settled by this document's
  silence.
