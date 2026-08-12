#!/usr/bin/env python3
"""Remove one participant's rows, on request, with no reason given.

RFC-0001 §2 asks for a target rather than a promise to remember, because
"an unimplemented deletion mechanism is indistinguishable from no
deletion mechanism at the only moment it matters" — the moment a person
asks.

    make forget P=p07

The RFC also asked this to delete matching records from the archive by
`subject_id`. It no longer has to, and that is the better outcome:
ADR-0014 stops the participant code from ever reaching the archive, so
there is nothing there to remove. The study's join key is
`evaluation_id`, which lives only in the capture row — deleting the row
severs the link, and what remains in the append-only archive is a
session that cannot be attributed to anyone.

That matters because the alternative was worse in both directions.
Deleting from the archive would have meant punching a hole in the one
guarantee `SECURITY.md` puts in scope; not deleting would have meant a
promise that quietly did not hold.

## What this cannot do

**It cannot retract a published aggregate.** If a number under
`docs/results/` was computed from a corpus that included this person,
removing their rows does not recall the manifest. A volunteer is
entitled to know that before they start rather than after they ask, and
`PARTICIPANTS.md` says so in the script they are read.
"""

from __future__ import annotations

import argparse
import datetime
import json
import pathlib
import sys

HERE = pathlib.Path(__file__).resolve().parent
CORPUS = HERE / "results/human_sessions.jsonl"
DELETION_LOG = HERE / "results/deletions.log"


def utc_today() -> str:
    return datetime.datetime.now(datetime.timezone.utc).strftime("%Y-%m-%d")


def partition(lines: list[str], code: str) -> tuple[list[str], int]:
    """The rows that stay, and how many went.

    A line that does not parse is KEPT. Deleting on request must not
    become a way to lose a corpus to a malformed line, and a row nobody
    can read is a row nobody can attribute either.
    """
    keep, removed = [], 0
    for line in lines:
        if not line.strip():
            continue
        try:
            row = json.loads(line)
        except json.JSONDecodeError:
            keep.append(line)
            continue
        if row.get("participant") == code:
            removed += 1
        else:
            keep.append(line)
    return keep, removed


def log_entry(code: str, removed: int, day: str) -> str:
    """The deletion log records that it happened, never what was in it.

    Code, date, count. Nothing else — a log that reconstituted the rows
    it records would be a copy of the thing the person asked to have
    deleted.
    """
    return f"{day}\t{code}\t{removed}\n"


def forget(corpus: pathlib.Path, log: pathlib.Path, code: str, day: str) -> int:
    """Returns rows removed. Idempotent: a second run removes zero."""
    if not corpus.exists():
        print(f"  no corpus at {corpus} — nothing has been captured yet.")
        print(f"  recording the request anyway, so that a code asked for before "
              f"capture is not silently forgotten.")
        removed = 0
    else:
        keep, removed = partition(corpus.read_text().splitlines(), code)
        if removed:
            corpus.write_text("".join(l + "\n" for l in keep))

    log.parent.mkdir(parents=True, exist_ok=True)
    with log.open("a") as fh:
        fh.write(log_entry(code, removed, day))

    if removed:
        print(f"  removed {removed} row(s) for {code} from {corpus.name}")
    else:
        print(f"  {code} matched no rows. That is not an error — a code asked "
              f"for twice, or never captured, both land here.")
    print(f"  recorded in {log.name}: date, code, count. Never content.")
    print(f"\n  This does NOT retract a published aggregate. A figure under "
          f"docs/results/ computed from a corpus that included {code} stands "
          f"as published; see RFC-0001 §2.")
    return removed


def _selftest() -> int:
    rows = [
        json.dumps({"participant": "p07", "arm": "A", "events": 5}),
        json.dumps({"participant": "p02", "arm": "B", "events": 9}),
        json.dumps({"participant": "p07", "arm": "A", "events": 7}),
    ]
    keep, removed = partition(rows, "p07")
    assert removed == 2, removed
    assert len(keep) == 1 and json.loads(keep[0])["participant"] == "p02"

    # Idempotent: the second pass finds nothing and says so.
    keep2, removed2 = partition(keep, "p07")
    assert removed2 == 0 and keep2 == keep

    # A code nobody has is not an error, and removes nothing.
    _, none = partition(rows, "p99")
    assert none == 0

    # An unreadable line is KEPT. Losing a corpus to a bad line would be
    # a deletion nobody asked for.
    broken = rows + ["{not json at all"]
    keep3, removed3 = partition(broken, "p07")
    assert removed3 == 2 and "{not json at all" in keep3, keep3

    # A code that is a substring of another must not match it.
    near = [json.dumps({"participant": "p0"}), json.dumps({"participant": "p07"})]
    _, exact = partition(near, "p0")
    assert exact == 1, "matched on a prefix rather than the whole code"

    # The log records that it happened and nothing about what.
    line = log_entry("p07", 2, "2026-08-12")
    assert line == "2026-08-12\tp07\t2\n", line
    assert "arm" not in line and "events" not in line

    print("  6 pass")
    return 0


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--selftest", action="store_true")
    ap.add_argument("-P", "--participant", default="",
                    help="the participant code to forget")
    ap.add_argument("--corpus", default=str(CORPUS))
    ap.add_argument("--log", default=str(DELETION_LOG))
    ns = ap.parse_args()

    if ns.selftest:
        return _selftest()
    if not ns.participant:
        print("no participant code given.\n\n  make forget P=p07\n")
        print("refusing rather than guessing: deleting the wrong person's rows "
              "is not recoverable from here.")
        return 1

    forget(pathlib.Path(ns.corpus), pathlib.Path(ns.log),
           ns.participant, utc_today())
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
