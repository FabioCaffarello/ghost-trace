---
type: skill
name: Pr Review
description: Review pull requests against team standards and best practices. Use when Reviewing a pull request before merge, Providing feedback on proposed changes, or Validating PR meets project standards
skillSlug: pr-review
phases: [R, V]
generated: 2026-08-04
status: filled
scaffoldVersion: "2.0.0"
---
Read the PR **title** first: it is the commit that lands and must be
Conventional Commits.

Then, in order:

- Does the description say what it **found** or got wrong, not only what
  it did? The history here is a chain of "this is what I discovered
  while building it".
- If `services/` or `experiments/` changed: were the six numbers run?
  If one moved, is there a manifest and an explanation?
- Are generated artifacts regenerated rather than edited?
- Did goldens change? Deliberately, and called out?
- Is there a test that would have caught the thing being fixed, and was
  it shown red?

`make ci` green is necessary and not sufficient.
