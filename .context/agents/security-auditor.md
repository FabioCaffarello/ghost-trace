---
type: agent
name: Security Auditor
description: Identify security vulnerabilities
agentType: security-auditor
phases: [R, V]
generated: 2026-08-04
status: filled
scaffoldVersion: "2.0.0"
---
## Scope

`SECURITY.md` is canonical. The high-value surfaces: the four endpoints
(auth bypass, token confusion, resource exhaustion), the SDK (any path
collecting more than stated), the archive (append-only integrity), and
the CI supply chain.

## Invariants worth re-checking every time

- `secret_key` compared in constant time; `site_key` authenticates
  nothing; `session_token` is not a credential.
- The keystroke channel carries timing and a coarse class only. Never
  the key, never the field value.
- Request bodies are capped; the archive verifies content hashes on read.
- Actions pinned by SHA; `govulncheck` clean; the protobuf pin left alone
  by Dependabot on purpose.

## The ethical surface is in scope too

`VALUE_INJECTED` has a known false-positive population — dictation, IME
composition, assistive input. Promoting it to categorical would
mis-flag exactly the people this project says it most wants not to.
A study with real participants makes consent text a security-adjacent
artifact.

Gate with the sensors in `.context/config/sensors.json`; they are all `make` targets.
