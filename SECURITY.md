# Security Policy

## Project status

Ghost Trace is in pre-implementation constitutional drafting. There are no production services, no deployed code paths, and no attack surface in the classical sense (no servers, no APIs, no stored content).

## Scope of security reporting

Three categories are in scope even at the current stage:

1. **Infrastructure bypass.** Mechanisms that bypass the operational discipline of `.claude/` (the hook, the CI workflow, the skills) in a way that could be used to introduce undetected constitutional drift. Examples: a way to silently edit FROZEN Charter sections; a way to suppress the hook on commits to in-scope files; a way to bypass the canonical-phrase exemption mechanism such that legitimate text is suppressed without record.

2. **Supply-chain risks in the Python helper or CI workflow.** If the `_parse_watchlists.py` helper or `.github/workflows/constitutional-check.yml` contains a vulnerability that could be exploited via a malicious PR, this is in scope.

3. **Documentation that misleads about scope.** If `SECURITY.md`, `WORKFLOW.md`, or `CONTRIBUTING.md` claim protection the infrastructure does not actually provide (the failure mode that `.claude/SELF-AUDIT.md` Rectification 2 surfaced for the pre-Gate-0b era of the hook's `in_scope` predicate), report it.

Out of scope at this stage:

- Vulnerabilities in `services/` code (no such code exists).
- Vulnerabilities in `schemas/` (no schemas committed).
- Vulnerabilities in dependencies of code that has not been written.

When implementation begins (gated by [`implementation-readiness-evaluator`](./.claude/skills/workflow/implementation-readiness-evaluator/SKILL.md)), this file will be revised to cover the implemented surface.

## How to report

For infrastructure-bypass or workflow-vulnerability reports, open a GitHub issue using the "Documentation bug" template if the issue is non-sensitive, or contact the maintainers directly (see [`README.md`](./README.md)) if the issue is sensitive (e.g., a working exploit that should not be public until patched).

A formal CVE process is not yet in place. When implementation produces code with a non-trivial attack surface, this section will be revised to reflect the CVE workflow then established.

## Acknowledgments

When a security-relevant finding is acted on, it is recorded in [`docs/charter/decision-log.md`](./docs/charter/decision-log.md) with the reporter's preferred attribution (or anonymously, on request).
