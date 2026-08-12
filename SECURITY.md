# Security Policy

## Reporting

**Please do not open a public issue.** Report privately by opening a
draft advisory:

https://github.com/FabioCaffarello/ghost-trace/security/advisories/new

It is visible only to the maintainers. This policy named no channel at
all until R1.16b, which meant the only way to report anything was the
public tracker — the opposite of what it intended.

Expect an acknowledgement within a week. This is a research project
maintained by one person, not a vendor with an on-call rota, and saying
so is more useful than a response time nobody can honour.

## Project status

Ghost Trace is a research project. The repository contains four Go
services — the collector (`/v1/sessions`, `/v1/telemetry`), the decision
engine (`/v1/decisions`, `/v1/outcomes`), the archive, and a stand-in
customer site — plus a browser SDK (`sdk.js`), protobuf schemas, and an
adversarial experiment layer. Every service also serves unauthenticated
`/healthz` and `/metrics`. Traffic between them is plaintext HTTP and
the compose topology binds `127.0.0.1` only; there is no TLS story and
no rate limiting. Nothing here is deployed as a production service by
this repository; anyone operating it does so at their own risk.

## Scope

In scope:

- The HTTP API (`/v1/sessions`, `/v1/telemetry`, `/v1/decisions`,
  `/v1/outcomes`) — auth bypass, token confusion, resource exhaustion.
- The browser SDK — any path by which it could collect more than the
  README's "What it collects, and what it refuses to" section states,
  most of all keystroke content or field values.
- The event archive (SQLite + blob store) — integrity of the
  append-only guarantee.
- CI workflows and dependency supply chain.

Out of scope:

- Detection evasion. Building a bot that beats the detector is the
  project's own adversarial track, not a vulnerability — see the
  experiment tiers.

## How to report

Open a GitHub security advisory (preferred) or contact the maintainer
directly for anything sensitive. Non-sensitive issues can be filed as
ordinary GitHub issues.

There is no CVE process and no bug bounty. Reports are acknowledged in
the fix's pull request with the reporter's preferred attribution, or
anonymously on request.
