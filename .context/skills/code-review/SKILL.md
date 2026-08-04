---
type: skill
name: Code Review
description: Review code quality, patterns, and best practices. Use when Reviewing code changes for quality, Checking adherence to coding standards, or Identifying potential bugs or issues
skillSlug: code-review
phases: [R, V]
generated: 2026-08-04
status: filled
scaffoldVersion: "2.0.0"
---
See `.context/agents/code-reviewer.md` — it is the fuller version.

The five questions, in order:

1. **Can this gate go red?** Vacuous greens are the recurring defect.
2. **Does absence stay absence?** A `0` where `null` is honest.
3. **Was a generated artifact hand-edited?**
4. **Did a published number move?** Then the prose around it must move.
5. **Did the comment survive the code?** Dangling citations, claims the
   tests do not support.

Formatting and naming are owned by `make fmt-check` and `make lint`.
