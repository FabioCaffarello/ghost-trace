---
type: skill
name: Commit Message
description: Generate commit messages that follow conventional commits and repository scope conventions. Use when Creating git commits after code changes, Writing commit messages for staged changes, or Following conventional commit format for the project
skillSlug: commit-message
phases: [E, C]
generated: 2026-08-04
status: filled
scaffoldVersion: "2.0.0"
---
Conventional Commits, milestone as scope. **The PR title is what lands** —
pull requests are squash-merged, so the title is the commit message and
the text release automation reads.

```
feat(r1.18): expose the decision contract over gRPC
fix(policy): stop rounding scores below zero to positive
ci(r1.13): pin every action by commit SHA
refactor(session)!: ports take a context
```

`feat` → minor, `fix`/`perf` → patch, `!` → major, everything else
releases nothing. The rule lives in `scripts/check-commit-message.sh`
with a 27-case selftest; the lefthook hook and CI both call it.

Bodies here explain **why**, and name what the change found or got
wrong. The diff already says what.
