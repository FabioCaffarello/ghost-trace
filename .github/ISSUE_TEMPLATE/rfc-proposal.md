---
name: RFC proposal (pre-draft)
about: Propose a structural change before drafting the formal RFC
title: "RFC proposal — <subject>"
labels: ["rfc", "discussion"]
---

## What you propose

One paragraph describing the proposed structural change. Be specific about what would be different in the substrate, its projections, the Ontology, or the Charter.

## RFC type

One of (per [`docs/rfcs/template.md`](../../docs/rfcs/template.md) Type field):

- [ ] `standard` — new capability or design within existing constitutional bounds.
- [ ] `charter-amendment` — change to the Constitutional Charter; follows amendment process per [`amendments.md`](../../docs/charter/amendments.md).
- [ ] `ontology-revision` — change to the Ontology that does not require Charter amendment.
- [ ] `architecture` — concrete architectural design (e.g., storage substrate selection).
- [ ] `experiment` — time-bounded experiment whose outcome will inform a future RFC.

## Constitutional review (preliminary)

Which Charter invariants does the proposal touch? Reference current status from [`CLAUDE.md` §4](../../.claude/CLAUDE.md):

- [ ] §1 Thesis (FROZEN)
- [ ] §2 header — qualification criteria (FROZEN)
- [ ] §2.1 Observational Integrity (FROZEN)
- [ ] §2.2 Epistemic Separation (FROZEN)
- [ ] §2.3 Provenance Integrity (pending)
- [ ] §2.4 Inferential Influence Disclosure (pending)
- [ ] §2.5 Hypothesis Lifecycle Explicitness (pending)
- [ ] §2.6 Evidential Independence Integrity (pending)
- [ ] §3 Non-Goals (pending)
- [ ] §4 Constitutional Design Rule (FROZEN)
- [ ] None

If any FROZEN section is touched, amendment is anticipated.

## Redaction-order interaction

Does the proposal touch a Charter section currently scheduled per [decision-log §0008](../../docs/charter/decision-log.md) (§2.5 → §2.3 → §2.4 → §2.6)?

- [ ] No interaction.
- [ ] Yes — the proposal defers to the schedule (waits for the section's redaction gate).
- [ ] Yes — the proposal precedes the schedule (would change order; requires a new decision-log entry justifying the departure).

## Why an issue, not a direct RFC

One sentence. Pre-draft issues are appropriate when the proposal is exploratory or when the author wants discussion before drafting. If the author already has a draft, the path is to open a PR with the draft in `docs/rfcs/draft/` and reference this issue (or skip the issue).
