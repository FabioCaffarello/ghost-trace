---
type: skill
name: Security Audit
description: Review code and infrastructure for security weaknesses. Use when Reviewing code for security vulnerabilities, Assessing authentication/authorization, or Checking for OWASP top 10 issues
skillSlug: security-audit
phases: [R, V]
generated: 2026-08-04
status: filled
scaffoldVersion: "2.0.0"
---
See `.context/agents/security-auditor.md` and `SECURITY.md`.

Check, every time:

- constant-time secret comparison; `site_key`/`session_token` are not
  credentials
- the SDK collects timing and coarse classes only — never key content
  or field values
- body caps, archive hash verification on read
- actions SHA-pinned, `make vuln` clean, the protobuf pin untouched
- consent text still true of what the SDK actually collects

The ethical surface counts: `VALUE_INJECTED` has a known false-positive
population and must not become categorical.
