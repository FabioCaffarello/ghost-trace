# Architecture decision records

One file per decision that was expensive to make and would be expensive
to re-litigate. Numbered, immutable once accepted: a superseded ADR gets
a successor and a banner, never an edit.

Not everything is an ADR. If the reasoning fits in a code comment next
to the thing it explains, that is where it belongs — this repository
has more rationale in comments than in documents, on purpose. An ADR is
for decisions that span files, or that a future reader would otherwise
reverse without knowing what it cost.

| # | decision | status |
| --- | --- | --- |
| [0001](0001-go-workspace-and-module-boundaries.md) | A Go workspace, and when a library gets extracted | accepted |
| [0002](0002-library-naming.md) | Libraries are named for what they are | accepted |
| [0003](0003-dual-write-during-the-split.md) | The local substrate stays authoritative during the split | superseded by [0006](0006-the-stream-is-the-archive.md) |
| [0004](0004-session-snapshots-carry-feature-state.md) | Session snapshots carry feature state, not events | accepted |
| [0005](0005-the-decision-endpoints-are-a-shared-module.md) | The decision endpoints are a shared module, handlers included | accepted |
| [0006](0006-the-stream-is-the-archive.md) | The stream is the archive; `-nats` and `-data` are alternatives | accepted |
| [0007](0007-the-metrics-library-is-the-exception.md) | prometheus/client_golang is the one exception to stdlib-only | accepted |
| [0008](0008-what-a-zero-is-allowed-to-mean.md) | What a zero is allowed to mean, and never measuring a failure with the failing component's state | accepted |
| [0009](0009-small-payloads-live-in-the-row.md) | Small payloads live in the row — 3x the archive's rate without weakening a durability promise | accepted |
| [0010](0010-unaccounted-must-never-be-negative.md) | `unaccounted` counts sequences, not commit operations — corrects one consequence of 0008 | accepted |
| [0011](0011-a-latency-is-a-conditional-claim.md) | A published latency claims a bound under a stated condition, not a value | accepted |
| [0012](0012-the-stream-is-bounded-by-bytes.md) | The stream is bounded by bytes, and the collector owns the bound | accepted |
