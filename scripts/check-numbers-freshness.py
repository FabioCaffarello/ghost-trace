#!/usr/bin/env python3
"""Report whether the published numbers still describe the code.

WHAT THIS MEASURES. `docs/results/numbers-*.json` records a run at a
commit. If commits have landed since that touch what the numbers are a
measurement OF, the published figures describe a system that no longer
exists — and nothing says so, because the only thing that would notice
is `make numbers`, which needs seven minutes and real browsers and is
outside `make ci` on purpose.

WHY IT EXISTS. On 2026-08-12 the published p99 was 5.525ms and the
system measured 0.796ms — a 5x gap, eight days old. The check that
compares latency was added by PR-5.0a on 2026-08-11 and was correct; it
had simply never been pointed at the record it defends. Adding a check
and running it are two acts, and only the first had happened. See
`docs/results/numbers-after-phase-6-2026-08-12.md`.

This does NOT re-measure anything and cannot tell you whether a number
moved. It answers a cheaper question that turns out to be the one nobody
was asking: **has anything happened since somebody last looked?**

PER BASELINE FAMILY, not per repository. Manifests are grouped by
(topology, archive) because `numbers_check` refuses to compare across
either — a composed run and a monolith run are different measurements
wearing the same name. A single newest-overall check would report the
repository fresh the moment ANY family was republished, which is exactly
the state this was written in: the monolith baseline republished on
2026-08-12 and the composed one still sitting at 2026-08-04. Each family
answers for itself.

The grouping functions are imported from `experiments/numbers_check.py`
rather than restated. Two mappings written apart satisfy their own tests
and disagree with each other, which this repository has paid for.

WHICH PATHS COUNT. `services/`, `experiments/` and `schemas/` — the three
the numbers-invariant skill names — plus `libs/`, which it does not.
That omission is not preserved here: `libs/policy` decides every
detection rate and `libs/substrate` decides the latency, so a rule that
watches the services and ignores the libraries they are assembled from
watches the wrong thing. Named explicitly so the difference is a
decision rather than a drift.

Deliberately NOT counted: `deploy/`, `contract/`, `docs/`, `.github/`,
`tools/`. A gate script or a write-up cannot move a measurement, and a
freshness check that fires on documentation is a freshness check people
learn to ignore.

Usage:
    check-numbers-freshness.py             # report; exit 1 if stale
    check-numbers-freshness.py --warn      # report; always exit 0
    check-numbers-freshness.py --selftest  # assert the logic
"""

from __future__ import annotations

import argparse
import json
import pathlib
import subprocess
import sys

ROOT = pathlib.Path(__file__).resolve().parent.parent
RESULTS = ROOT / "docs/results"

sys.path.insert(0, str(ROOT / "experiments"))
from numbers_check import archive, topology  # noqa: E402

# What a number is a measurement of. See the module docstring for why
# libs/ is here and deploy/ is not.
MEASURED = ("services/", "experiments/", "schemas/", "libs/")


def newest_per_family(results: pathlib.Path) -> dict[tuple[str, str], tuple[pathlib.Path, dict]]:
    """The most recently GENERATED manifest of each (topology, archive).

    By generation time, never by filename: filenames carry a date and
    manifests are published by hand, so a republished older run would
    sort ahead of a newer one. The provenance block is the record; the
    filename is a label.
    """
    best: dict[tuple[str, str], tuple[pathlib.Path, dict]] = {}
    for p in sorted(results.glob("numbers-*.json")):
        try:
            d = json.loads(p.read_text())
        except (OSError, json.JSONDecodeError):
            continue
        key = (topology(d), archive(d) or "unrecorded")
        at = d.get("provenance", {}).get("generated_at", "")
        if key not in best or at > best[key][1]["provenance"]["generated_at"]:
            best[key] = (p, d)
    return best


def commits_since(commit: str, paths: tuple[str, ...]) -> list[str] | None:
    """Commits touching `paths` after `commit`, newest first.

    None means the question could not be asked — the commit is not in
    this history, which happens to anyone who published from a branch
    that was later rebased away. An unanswerable question is reported as
    unanswerable, never as a clean answer.
    """
    r = subprocess.run(
        ["git", "log", "--format=%h %ad %s", "--date=short", f"{commit}..HEAD", "--", *paths],
        cwd=ROOT, capture_output=True, text=True,
    )
    if r.returncode != 0:
        return None
    return [line for line in r.stdout.splitlines() if line.strip()]


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--warn", action="store_true",
                    help="report and exit 0 — for contexts that surface drift without blocking")
    ap.add_argument("--selftest", action="store_true")
    args = ap.parse_args()

    if args.selftest:
        return _selftest()

    families = newest_per_family(RESULTS)
    if not families:
        print("no manifest in docs/results/ — nothing has ever been published.")
        return 0 if args.warn else 1

    stale = 0
    for (topo, arch), (path, manifest) in sorted(families.items()):
        commit = manifest["provenance"]["git"]["commit"]
        when = manifest["provenance"]["generated_at"]
        since = commits_since(commit, MEASURED)

        print(f"\n{topo} / {arch}")
        print(f"  {path.name}")
        print(f"  measured {when} at {commit[:12]}")

        if since is None:
            stale += 1
            print(f"  UNANSWERABLE — {commit[:12]} is not in this history, so nothing can")
            print("                 be said about what landed after it. Republish, or")
            print("                 fetch the commit it names.")
            continue

        if not since:
            print("  fresh — nothing measured has changed since")
            continue

        stale += 1
        print(f"  STALE — {len(since)} commit(s) have touched "
              f"{', '.join(MEASURED)} since:")
        for line in since[:5]:
            print(f"    {line}")
        if len(since) > 5:
            print(f"    ... and {len(since) - 5} more")

    if not stale:
        print("\n  every published baseline still describes the code it was measured on.")
        return 0

    print(f"\n  {stale} of {len(families)} baseline(s) describe a system that has since")
    print("  changed. That is not a claim any number moved — it is a claim that")
    print("  nobody has looked. Looking is:")
    print("\n    make numbers            # ~7 minutes, needs real browsers")
    print("    make numbers-manifest   # if it moved on purpose, publish the new record")
    print("\n  A composed baseline needs the topology up and GT_BASE pointed at it;")
    print("  see experiments/README.md.")
    return 0 if args.warn else 1


def _selftest() -> int:
    import tempfile

    checks = []

    def ok(cond, what):
        checks.append((bool(cond), what))
        print(f"  {'ok  ' if cond else 'FAIL'}  {what}")

    with tempfile.TemporaryDirectory() as d:
        tmp = pathlib.Path(d)

        def write(name, at, commit, topo=None, arch=None):
            run = {}
            if topo is not None:
                run["topology"] = topo
            if arch is not None:
                run["archive"] = arch
            (tmp / name).write_text(json.dumps({
                "provenance": {"generated_at": at, "git": {"commit": commit}, "run": run},
            }))

        ok(newest_per_family(tmp) == {}, "an empty directory yields no families")

        write("numbers-2026-08-04-aaaaaaaa.json", "2026-08-04T13:20:15Z", "a" * 40)
        write("numbers-2026-08-12-bbbbbbbb.json", "2026-08-12T23:52:45Z", "b" * 40)
        fam = newest_per_family(tmp)
        ok(len(fam) == 1, "manifests with no run block fall into one family")
        ok(next(iter(fam.values()))[1]["provenance"]["git"]["commit"].startswith("b"),
           "the newest GENERATED manifest of a family wins")

        # The trap this picker exists to avoid: a run measured earlier
        # but published later sorts ahead by filename and behind by
        # provenance. Provenance is the record.
        write("numbers-2026-08-31-cccccccc.json", "2026-08-01T00:00:00Z", "c" * 40)
        fam = newest_per_family(tmp)
        ok(next(iter(fam.values()))[1]["provenance"]["git"]["commit"].startswith("b"),
           "a later FILENAME with an earlier measurement does not win")

        (tmp / "numbers-broken.json").write_text("{not json")
        ok(len(newest_per_family(tmp)) == 1,
           "an unreadable manifest is skipped rather than crashing the check")

        # THE HOLE THIS GROUPING EXISTS TO CLOSE. Republishing one family
        # must not vouch for another: a newest-overall check would have
        # gone green here while the composed baseline stayed weeks old.
        write("numbers-composed.json", "2026-08-04T00:00:00Z", "d" * 40, "composed", "stream")
        write("numbers-mono.json", "2026-08-12T00:00:00Z", "e" * 40, "monolith", "substrate")
        fam = newest_per_family(tmp)
        ok(("composed", "stream") in fam and ("monolith", "substrate") in fam,
           "a composed and a monolith baseline are separate families")
        ok(fam[("composed", "stream")][1]["provenance"]["git"]["commit"].startswith("d"),
           "republishing the monolith does not refresh the composed baseline")

    ok(commits_since("0" * 40, MEASURED) is None,
       "a commit this history does not contain reports UNANSWERABLE, not clean")

    head = subprocess.run(["git", "rev-parse", "HEAD"], cwd=ROOT,
                          capture_output=True, text=True).stdout.strip()
    ok(commits_since(head, MEASURED) == [],
       "HEAD against itself reports nothing since")

    # The paths are the decision, so they are asserted rather than left
    # to a reader of the tuple.
    ok("libs/" in MEASURED, "libs/ counts — policy decides the rates, substrate the latency")
    ok("deploy/" not in MEASURED, "deploy/ does not — a gate script cannot move a measurement")
    ok("docs/" not in MEASURED, "docs/ does not — a check that fires on prose gets ignored")

    bad = [w for good, w in checks if not good]
    print(f"\n  {len(checks) - len(bad)}/{len(checks)} freshness cases hold")
    return 1 if bad else 0


if __name__ == "__main__":
    sys.exit(main())
