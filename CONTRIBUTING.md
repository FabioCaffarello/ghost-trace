# Contributing to Ghost Trace

Ghost Trace is in pre-implementation constitutional drafting. Contribution norms reflect this stage.

## Before You Contribute

Read, in this order:

1. [Constitutional Charter — Thesis](./docs/charter/constitutional-charter.md#1-thesis).
2. [Constitutional Charter — Constitutional Invariants](./docs/charter/constitutional-charter.md#2-constitutional-invariants).
3. [Ontology — top-level](./docs/ontology/ontology.md).
4. [Decision Log](./docs/charter/decision-log.md).

If your contribution does not respect what you find there, it will be rejected on procedural grounds, regardless of technical merit.

## What Kinds of Contributions Are Welcome

At this stage:

- **Critical readings of the Charter.** Identification of ambiguity, drift risk, or non-falsifiable language. These are the most valuable contributions during constitutional drafting.
- **Draft RFCs.** Especially those addressing open questions recorded in scaffold documents and the decision log.
- **Ontology refinements.** Particularly around the open modeling questions in [`docs/ontology/`](./docs/ontology/).
- **Adversarial review.** Proposals describing how a hostile actor might exploit weaknesses in the architecture, or how the architecture might degrade under operational pressure.

What is not yet welcome:

- Implementation of services. Implementation work begins after the Ontology stabilizes.
- Storage technology selection without an RFC.
- Documentation that reads like marketing.

## How to Propose a Charter Amendment

See [`docs/charter/amendments.md`](./docs/charter/amendments.md). The short version: a charter amendment is proposed as an RFC of type `charter-amendment`, undergoes falsifiability review, and is redacted in committee mode if accepted.

## How to Propose an Ontology or Architecture Change

Open an RFC in [`docs/rfcs/draft/`](./docs/rfcs/draft/) using the [template](./docs/rfcs/template.md). Every RFC undergoes constitutional review.

## Style

- Documentation in Markdown.
- Code style decisions are deferred until implementation begins. They will be recorded as decisions when made.
- All documentation must use the established vocabulary (substrate, projection, observation, operational construct, hypothesis, provenance, influence, supersession). The vocabulary is constitutional; using different words for the same concept introduces drift.

## What This Project Is Not

This project is not a place to demonstrate technical sophistication for its own sake. The Charter rejects, by construction, contributions whose primary value is impressive complexity. A contribution must constrain or enable specific behavior in a way that survives the falsifiability test.

If you find yourself writing prose that sounds important but cannot be falsified, rewrite it or do not include it.
