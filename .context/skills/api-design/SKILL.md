---
type: skill
name: Api Design
description: Design RESTful APIs following best practices. Use when Designing new API endpoints, Restructuring existing APIs, or Planning API versioning strategy
skillSlug: api-design
phases: [P, R]
generated: 2026-08-04
status: filled
scaffoldVersion: "2.0.0"
---
The HTTP surface is `contract/architecture.md` §3 and the generated
`contract/openapi.yaml`. Any change to it goes through the
**wire-contract-change** skill.

Principles already decided:

- Two callers, two trust levels. Only `secret_key` authenticates; only
  it may assert `subject_id`.
- Fail-open. Telemetry loss is expected; unknown sessions get 202.
- `required` in the request schemas means **the handler refuses without
  it** — not "the Go field is non-pointer". Reflection gets this wrong;
  `requiredOverrides` in the generator states the truth.
- A field the service accepts and discards does not belong in the
  contract. `context` was removed for exactly that.
