# 0014 — a study identity never enters the append-only archive

**Status:** accepted · **Date:** 2026-08-12 · **Milestone:** PR-4.P4

## Context

[ADR-0006](0006-the-stream-is-the-archive.md) makes the event archive
the durable record, and `SECURITY.md` puts "integrity of the append-only
guarantee" in scope for reports. Nothing is removed from it; that is the
point of it.

[RFC-0001 §2](../rfcs/0001-human-study-data-governance.md) proposes that
a participant can have their rows removed by asking, with no reason
given, and that the mechanism must be a target rather than a promise to
remember. It asked for two things: rewrite the capture corpus, **and
delete the matching archive records by `subject_id`**.

Those two requirements were in direct conflict, and the conflict was not
hypothetical. `demo-web` asserted the participant code as `subject_id`
on the decision call, so it was copied into every `Evaluation` record
and committed to the substrate permanently. PR-4.P2's dry run confirmed
it: three synthetic codes, three codes in the archived payloads.

So a deletion request had exactly two possible answers, and both were
bad. Punch a hole in the one guarantee the project advertises — or let
the promise quietly not hold for the part of the data that outlives
everything else.

## Decision

**The archive never receives a study identity, so there is nothing in it
to delete.**

`demo-web` sends no `subject_id` when a participant code is present. The
ordinary demo path is unchanged and still asserts `user_<name>`.

Nothing is lost by the omission:

- The engine **copies** `subject_id` into the `Evaluation` record and
  decides nothing with it. Removing it changes no decision, no score and
  no reason code.
- The study's join key is `evaluation_id`, which the capture row already
  carries. Participant ↔ evaluation lives in
  `results/human_sessions.jsonl` and nowhere else.

Deleting the row therefore severs the link permanently. What remains in
the archive is a session that **cannot be attributed to anyone** — which
is what deletion can honestly mean when the store does not forget.

`make forget P=<code>` rewrites the corpus, appends *code, date, count*
to a deletion log — never content, so the log cannot reconstitute what
it records — is idempotent, and says plainly when a code matched
nothing.

## Consequences

**The append-only guarantee is untouched**, and did not have to be
renegotiated to keep a promise to a person. That is the whole reason
this ADR exists rather than an exception clause in ADR-0006.

**RFC-0001 §2's second requirement is obsolete, not unmet.** The RFC is
still *proposed*; when it is read for acceptance, that clause should be
struck rather than implemented.

**It cannot retract a published aggregate.** If a figure under
`docs/results/` was computed from a corpus that included someone,
deleting their rows does not recall the manifest. `PARTICIPANTS.md` says
so before anyone starts, which is the only time saying it is worth
anything.

**A cohort is still invisible to the engine.** `arm`, `condition` and
`visit` never crossed the wire and still do not — a session's cohort is
a property of the experiment, not of the product. `make capture-dryrun`
asserts all four absences, and its assertion about the participant code
is now inverted from what PR-4.P2 shipped: the code must be **absent**
from the archive, and its presence is the failure.

## Alternatives considered

**Delete from the archive, as a documented exception.** An operation
that removes records deliberately, logged and counted, with ADR-0006
amended to say that "append-only" means nothing is lost *silently*
rather than nothing is ever removed. Rejected: it weakens a guarantee
the security policy advertises, for a case that a design change removes
entirely. Every future auditor would have to re-read what the guarantee
promises.

**Delete only the corpus, and say the archive rows remain.** Fast, and
partially honest. Rejected on RFC-0001's own terms — an incomplete
mechanism is indistinguishable from none at the moment it matters, and
"we kept a pseudonym for you permanently, but only over there" is not a
sentence to put in front of a volunteer.

**Hash the participant code before sending it.** A pseudonym of a
pseudonym still links visits and still cannot be removed. It buys
nothing over sending nothing.
