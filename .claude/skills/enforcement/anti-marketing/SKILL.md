---
name: anti-marketing
description: Detect and reject marketing language in all project documents. Use this skill ALWAYS when reviewing prose for README.md, CONTRIBUTING.md, repository description, RFC summaries, decision log entries, glossary entries, or any new document. Orthogonal to falsifiability — a sentence can be falsifiable AND marketing, or non-marketing AND non-falsifiable. Both axes are enforced.
---

# anti-marketing

[`CONTRIBUTING.md` §What This Project Is Not](../../../../CONTRIBUTING.md) records the rule this skill operationalizes: "If you find yourself writing prose that sounds important but cannot be falsified, rewrite it or do not include it." Marketing language is a particular failure mode of sounding important. This skill catches it before commit.

Marketing is orthogonal to falsifiability ([`epistemic/falsifiability-check`](../../epistemic/falsifiability-check/SKILL.md)): a claim can pass the falsifiability test and still be marketing (true but pitched), or fail the falsifiability test without being marketing (vague but earnest). Both axes are enforced separately.

## 1. Marketing tells

A non-exhaustive list. Each category is illustrated with examples drawn from prose that would plausibly appear in a draft.

### Empty superlatives

`world-class`, `state-of-the-art`, `best-in-class`, `industry-leading`, `cutting-edge`, `revolutionary`, `next-generation`, `enterprise-grade`.

> "Ghost Trace provides best-in-class observational integrity."

The superlative substitutes for a structural claim. Cut the superlative; the underlying claim ("Ghost Trace provides observational integrity") is in the Charter and is already verifiable. The superlative adds nothing the substrate can express.

### Vision-prose verbs

`empowers`, `unlocks`, `transforms`, `reimagines`, `redefines`, `delivers`, `enables` (when used without naming what is enabled).

> "This RFC unlocks new capabilities for behavioral intelligence."

The verb gestures at consequence without naming it. Replace with a behavioral claim about the substrate: "This RFC permits the assertion engine to emit hypothesis-formation events that reference the source observations."

### Aspirational nouns deployed as standalone goods

`excellence`, `innovation`, `synergy`, `intelligence`, `quality`, `integrity`, `trust` — when used as ends in themselves rather than as terms whose structural meaning is given.

> "The system prioritizes integrity."

`integrity` is not a defined good in this project except where it is paired with a specific dimension (`observational integrity`, `evidential independence integrity`). Standalone `integrity` is marketing. Either name the specific Charter invariant or rewrite.

### Audience flattery

`for forward-thinking teams`, `for engineers who care`, `for serious operators`, `for the discerning`.

> "Ghost Trace is for teams who take epistemics seriously."

The sentence selects a flattered reader instead of describing the system. Delete; the Charter describes the system.

### Importance-by-assertion

`critically`, `essentially`, `fundamentally`, `at its core`, `at heart` — used as rhetorical signal rather than as logical connector.

> "At its core, Ghost Trace is about preserving epistemic integrity."

Strip the modifier: "Ghost Trace preserves epistemic integrity." If the underlying claim is no longer interesting without the modifier, the modifier *was* the claim. The Charter does not need "at its core" because the Charter is the core.

### Vague enabling claims

`helps you`, `lets you`, `makes it easy to`, `provides a way to`, `offers the ability to` — used without operational specifics.

> "Ghost Trace lets you understand your behavioral data."

`understand` is not operational; `behavioral data` is not canonical (compare [`ambiguity-reducer` watchlist](../../epistemic/ambiguity-reducer/SKILL.md)). Rewrite as a behavioral claim about the substrate, or delete.

### Founder-voice prose

Sentences that would fit on a pitch deck. The tell is rhetorical register; the failure is the same — substance is substituted by tone.

> "We believe behavioral intelligence is the future of compliance."

Belief statements about future trends are not Charter material. They belong in a personal essay, not in a constitutional document.

## 2. Positive criterion

A sentence is acceptable if a reader could verify or falsify its main claim against:

- The source documents (Charter, Ontology, architecture), or
- The eventual substrate (a specific record type, a specific projection, a specific event variety).

If verification requires trusting the author's sincerity or intent, the sentence is marketing. If verification requires inspecting concrete artifacts or behavior, the sentence is acceptable.

## 3. Rewrite paths

| Tell | Path |
|---|---|
| Empty superlative | Delete the superlative. If the residual claim is also empty, delete the sentence. Otherwise specify the structural property the superlative was gesturing at. |
| Vision-prose verb | Replace with a behavioral claim about the substrate. If no such claim exists, delete. |
| Aspirational noun | Replace with the canonical term that names the structural property. If no such term exists, the noun was marketing; delete. |
| Audience flattery | Delete. Replace with a description of the system, not the reader. |
| Importance-by-assertion | Delete the modifier. Re-read the residual claim cold. |
| Vague enabling claim | Replace with the specific operation the system performs. If no such operation exists, the enabling claim was aspirational; delete. |
| Founder-voice | Delete. The Charter is the project's voice; new prose does not need a second one. |

## 4. Special-case exemption — the Charter

Quotations from the Charter or the Ontology that contain words on the watchlist are not violations. The Charter has earned its vocabulary by surviving committee redaction; the words are loaded with structural meaning. Words like `integrity` and `epistemic integrity` in the Thesis are not aspirational nouns — they are terms with operational consequences elsewhere in the Charter.

The exemption applies only to quotation. New prose that uses the same words without the Charter's earned context is flagged. The test: would a reader who has not read the Charter understand the term as a specific structural property, or as elevated language? If the latter, the exemption does not apply.

## 5. What this skill does not do

This skill does not approve prose. It rejects marketing. Approval is the normal review of substance, structure, and citations.

This skill does not check falsifiability. A sentence can be non-marketing and still non-falsifiable (vague but earnest); `falsifiability-check` is the orthogonal axis.

This skill does not flag the Charter itself. The Charter's vocabulary is canonical even where it sounds elevated, per §4 above.

## 6. Delegations

| Sub-task | Delegated to |
|---|---|
| Falsifiability of any non-marketing claim that survives this skill | [`epistemic/falsifiability-check`](../../epistemic/falsifiability-check/SKILL.md) |
| Vocabulary check on the canonical substitutes proposed in §3 rewrites | [`ontology/vocabulary-discipline`](../../ontology/vocabulary-discipline/SKILL.md) |
| Open-Ontology-question detection in proposed rewrites | [`ontology/ontology-keeper`](../../ontology/ontology-keeper/SKILL.md) |

No skill delegates *into* this one for marketing detection. This skill is the canonical home for the marketing axis.

## 7. Source citations used

- [`CONTRIBUTING.md` §What This Project Is Not; §What Kinds of Contributions Are Welcome](../../../../CONTRIBUTING.md)
- [`docs/charter/constitutional-charter.md` §1 Thesis (for the Charter-quotation exemption reference)](../../../../docs/charter/constitutional-charter.md#1-thesis)
- [`.claude/CLAUDE.md` §6 Operational rules (rule 7: no marketing); §3 Canonical vocabulary](../../../CLAUDE.md)
- [`.claude/skills/epistemic/falsifiability-check/SKILL.md`](../../epistemic/falsifiability-check/SKILL.md)
- [`.claude/skills/epistemic/ambiguity-reducer/SKILL.md`](../../epistemic/ambiguity-reducer/SKILL.md)
