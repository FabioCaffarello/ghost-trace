# Requests for comment

A proposal, written before the thing it proposes exists, for a decision
that cannot be made well inside a pull request.

**An RFC is not an ADR.** An [ADR](../decisions/) records a decision
already taken, and is immutable once accepted. An RFC is the argument
that comes first: it lays out the options, recommends one, and names
what it would cost. It is a document you are allowed to disagree with.

Most decisions here need neither. If the reasoning fits beside the code,
that is where it belongs; if it spans files and is settled, it is an
ADR. An RFC earns its own file only when a decision **gates work that
cannot honestly start without it**, and the deciding needs more than one
sitting.

| # | proposal | status | gates |
| --- | --- | --- | --- |
| [0001](0001-human-study-data-governance.md) | Human-study data governance: retention, deletion, custody | **proposed** | recruiting the participants number 2 needs |

## Status means what it says

- **proposed** — written, not agreed. Nothing may rely on it.
- **accepted** — agreed. The work it gates may start; anything it
  decided permanently belongs in an ADR that cites it.
- **withdrawn** — the question dissolved, or was answered elsewhere. The
  file stays, with the reason.

A proposed RFC is not a soft yes. This directory exists because the
alternative — deciding retention policy in a commit message at the
moment someone wants to start recruiting — is how a study acquires
obligations nobody chose.
