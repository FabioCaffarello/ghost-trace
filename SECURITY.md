# Security Policy

## Project status

Ghost Trace is a research project. The repository contains a running Go
service (`services/ingestion`) exposing four HTTP endpoints plus a demo
page, a browser SDK (`sdk.js`), protobuf schemas, and an adversarial
experiment layer. Nothing here is deployed as a production service by
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
