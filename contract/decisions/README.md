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
