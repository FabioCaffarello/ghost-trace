# Contributing to Ghost Trace

Ghost Trace is in **implementation phase** post the [`decision-log §0022`](./docs/charter/decision-log.md) implementation pivot. Constitutional sections continue to redact in committee mode in parallel with implementation work; §2.6 and §3 remain pending under the `empirical pressure phase` posture. Contribution norms reflect this dual-track stage.

## Before You Contribute

Read, in this order:

1. [Constitutional Charter — Thesis](./docs/charter/constitutional-charter.md#1-thesis).
2. [Constitutional Charter — Constitutional Invariants](./docs/charter/constitutional-charter.md#2-constitutional-invariants).
3. [Ontology — top-level](./docs/ontology/ontology.md).
4. [Decision Log](./docs/charter/decision-log.md).

If your contribution does not respect what you find there, it will be rejected on procedural grounds, regardless of technical merit.

## What Kinds of Contributions Are Welcome

At this stage:

- **Critical readings of the Charter.** Identification of ambiguity, drift risk, or non-falsifiable language. Constitutional drafting remains active for §2.6 and §3 under the empirical-pressure-phase posture.
- **Draft RFCs.** Especially those addressing open questions recorded in scaffold documents and the decision log.
- **Ontology refinements.** Particularly around the open modeling questions in [`docs/ontology/`](./docs/ontology/) and the Q2-A.2 cross-subtype follow-on at discussion phase.
- **Adversarial review.** Proposals describing how a hostile actor might exploit weaknesses in the architecture, or how the architecture might degrade under operational pressure.
- **Implementation contributions.** Service code, schemas evolution, CLI extensions, projection/replay handlers — all under ordinary RFC discipline against the frozen Charter sections and resolved Ontology questions.

What is not welcome:

- Implementation that conflicts with a frozen Charter invariant. Such contributions are rejected on procedural grounds regardless of technical merit; if the contributor believes the Charter is wrong, the path is `charter-amendment` RFC, not implementation that bypasses it.
- Storage technology, schemas technology, or implementation-language reselection without an RFC. The active technology selections are recorded in [`docs/charter/decision-log.md`](./docs/charter/decision-log.md); revision follows ordinary RFC discipline.
- Documentation that reads like marketing.

## How to Propose a Charter Amendment

See [`docs/charter/amendments.md`](./docs/charter/amendments.md). The short version: a charter amendment is proposed as an RFC of type `charter-amendment`, undergoes falsifiability review, and is redacted in committee mode if accepted.

## How to Propose an Ontology or Architecture Change

Open an RFC in [`docs/rfcs/draft/`](./docs/rfcs/draft/) using the [template](./docs/rfcs/template.md). Every RFC undergoes constitutional review.

## Style

- Documentation in Markdown.
- Code is Go for the ingestion service per [`docs/charter/decision-log.md`](./docs/charter/decision-log.md) implementation-language selection; the service-level Makefile + `go vet` + `go test` discipline applies to all Go contributions.
- All documentation must use the established vocabulary (substrate, projection, observation, operational construct, hypothesis, provenance, influence, supersession). The vocabulary is constitutional; using different words for the same concept introduces drift. Forbidden synonyms are checked at commit time per [`docs/glossary.md`](./docs/glossary.md) + [`.claude/skills/ontology/vocabulary-discipline`](./.claude/skills/ontology/vocabulary-discipline/).

## What This Project Is Not

This project is not a place to demonstrate technical sophistication for its own sake. The Charter rejects, by construction, contributions whose primary value is impressive complexity. A contribution must constrain or enable specific behavior in a way that survives the falsifiability test.

If you find yourself writing prose that sounds important but cannot be falsified, rewrite it or do not include it.

## Working with Claude Code (optional)

This repository carries optional Claude Code operational support under [`.claude/`](./.claude/). See [`WORKFLOW.md`](./WORKFLOW.md) for the operator-facing details — the skill layer, the slash commands, the subagents, the three-tier hook architecture, and the CI workflow.

What Claude Code is expected to enforce in this repository:

- **Vocabulary discipline.** The canonical vocabulary in [`docs/glossary.md`](./docs/glossary.md) is binding; forbidden synonyms are flagged at commit time.
- **Falsifiability.** Every constitutional claim is tested against the four-question criterion in `falsifiability-check`.
- **Subordination.** Claims in lower-ranked documents are checked against the higher-ranked documents they depend on.
- **Amendment process.** Edits to FROZEN Charter sections are blocked unless accompanied by the artifacts the amendment process requires (RFC, `amendments.md` entry, version bump).

Contributions are evaluated on their alignment with the Charter, not on whether they were produced with or without Claude Code. The tooling is convenience; the constitutional review is the same in either case. A contributor working in vim with no Claude Code involvement produces equally valid contributions as a contributor working with Claude Code.

**When this section and `WORKFLOW.md` conflict, this section wins.** `CONTRIBUTING.md` is the authoritative process document; `WORKFLOW.md` is tooling. The precedence rule is established in [`.claude/CLAUDE.md` §2](./.claude/CLAUDE.md) and §5.5.
