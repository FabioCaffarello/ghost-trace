#!/usr/bin/env python3
"""Assert that the summary jobs actually summarise their workflow.

Branch protection cannot require a matrix. It requires named contexts,
so each workflow ends in one job that succeeds only when every other job
did, and `main` requires that one name. The arrangement has a failure
mode worth more than the convenience it buys:

  - A NEW JOB IS ADDED AND NOT LISTED IN `needs`. It runs, it can fail,
    and the required check goes green anyway. Protection now says the
    branch is guarded when nothing is guarding it.
  - THE EXPRESSION STOPS EVALUATING. `join(needs.*.result, " ")` with
    double quotes is a syntax error in a GitHub expression — it took
    both workflows down at zero seconds, and had it merely returned
    empty instead, a loop over nothing would have passed.

Both are checked here, statically, so the feedback arrives before a push
rather than after a merge. The runtime count guard in the workflow stays
as well: this file can only see what is written down, and that one sees
what actually ran.

Usage:
    check-workflows.py              # check the workflows
    check-workflows.py --selftest   # assert the checks catch what they claim
"""

from __future__ import annotations

import argparse
import copy
import os
import re
import sys

import yaml

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
WORKFLOWS = os.path.join(ROOT, ".github", "workflows")

# The job each workflow ends in, and the context `main` requires.
SUMMARY = "required"

COUNT_GUARD = re.compile(r'\[ "\$count" != "(\d+)" \]')
JOIN_EXPR = re.compile(r"join\(needs\.\*\.result, (.)")


def load(path: str) -> dict:
    with open(path) as fh:
        return yaml.safe_load(fh)


def problems(wf: dict, name: str) -> list[str]:
    """Everything wrong with one workflow's summary job."""
    out: list[str] = []
    jobs = wf.get("jobs", {})

    if SUMMARY not in jobs:
        # Not every workflow has one. Only those whose name `main`
        # requires do, and that is checked by the caller's file list.
        return [f"{name}: no {SUMMARY!r} job"]

    summary = jobs[SUMMARY]
    needs = summary.get("needs") or []
    if isinstance(needs, str):
        needs = [needs]

    # 1. Every other job must be required. This is the one that matters:
    #    a job nobody needs is a job that can fail unnoticed.
    others = [j for j in jobs if j != SUMMARY]
    missing = [j for j in others if j not in needs]
    if missing:
        out.append(
            f"{name}: job(s) {', '.join(sorted(missing))} are not in "
            f"{SUMMARY}.needs, so they can fail while the required check passes"
        )

    stale = [j for j in needs if j not in jobs]
    if stale:
        out.append(f"{name}: {SUMMARY}.needs lists {', '.join(sorted(stale))}, which "
                   f"no longer exist")

    # 2. It must run even when a dependency failed. Without always(), a
    #    job that `needs` a failed job is SKIPPED — and a skipped
    #    required check blocks forever rather than reporting.
    if str(summary.get("if", "")).strip() != "always()":
        out.append(f"{name}: {SUMMARY} must be `if: always()`, or it is skipped "
                   f"exactly when a job it guards has failed")

    run = "\n".join(s.get("run", "") for s in summary.get("steps", []))

    # 3. String literals in GitHub expressions are single-quoted.
    m = JOIN_EXPR.search(run)
    if not m:
        out.append(f"{name}: {SUMMARY} does not read join(needs.*.result, ...)")
    elif m.group(1) != "'":
        out.append(f"{name}: join(needs.*.result, ...) uses {m.group(1)} — GitHub "
                   f"expressions take single quotes, and this fails the whole "
                   f"workflow file before any job runs")

    # 4. The runtime count guard must agree with the needs list.
    m = COUNT_GUARD.search(run)
    if not m:
        out.append(f"{name}: {SUMMARY} has no count guard, so an expression that "
                   f"evaluates to empty would pass")
    elif int(m.group(1)) != len(needs):
        out.append(f"{name}: count guard expects {m.group(1)} results but "
                   f"{SUMMARY} needs {len(needs)} jobs")

    return out


# The workflows whose summary job a branch protection rule names.
GUARDED = ["ci.yml", "image.yml"]


def check() -> int:
    found: list[str] = []
    for f in GUARDED:
        path = os.path.join(WORKFLOWS, f)
        if not os.path.exists(path):
            found.append(f"{f}: missing, but branch protection requires its summary")
            continue
        found += problems(load(path), f)

    for p in found:
        print(f"  {p}")
    if found:
        print(f"\n  {len(found)} problem(s); the required check would not guard what "
              f"it claims to")
        return 1
    print(f"  {len(GUARDED)} guarded workflow(s): every job is required, the summary "
          f"runs on failure, and its count agrees")
    return 0


# ---------------------------------------------------------------
# selftest
# ---------------------------------------------------------------

def selftest() -> int:
    failures = 0

    def check_case(ok: bool, what: str) -> None:
        nonlocal failures
        print(f"  {'ok  ' if ok else 'FAIL'}  {what}")
        failures += not ok

    good = {
        "jobs": {
            "a": {}, "b": {},
            SUMMARY: {
                "if": "always()",
                "needs": ["a", "b"],
                "steps": [{"run": """results='${{ join(needs.*.result, ' ') }}'
count=$(echo $results | wc -w)
if [ "$count" != "2" ]; then exit 1; fi
"""}],
            },
        }
    }
    check_case(problems(good, "w") == [], "a correct workflow reports nothing")

    def mutate(fn):
        w = copy.deepcopy(good)
        fn(w)
        return problems(w, "w")

    for name, fn, want in [
        ("an unrequired job is caught",
         lambda w: w["jobs"].update({"c": {}}), "not in required.needs"),
        ("a needs entry with no job is caught",
         lambda w: w["jobs"][SUMMARY]["needs"].append("gone"), "no longer exist"),
        ("a missing always() is caught",
         lambda w: w["jobs"][SUMMARY].pop("if"), "if: always()"),
        ("a non-always if is caught",
         lambda w: w["jobs"][SUMMARY].update({"if": "success()"}), "if: always()"),
        ("double quotes in the expression are caught",
         lambda w: w["jobs"][SUMMARY]["steps"][0].update(
             {"run": w["jobs"][SUMMARY]["steps"][0]["run"].replace(
                 "result, ' '", 'result, " "')}), "single quotes"),
        ("a missing count guard is caught",
         lambda w: w["jobs"][SUMMARY]["steps"][0].update(
             {"run": "results='${{ join(needs.*.result, ' ') }}'"}), "no count guard"),
        ("a count that disagrees with needs is caught",
         lambda w: w["jobs"][SUMMARY]["steps"][0].update(
             {"run": w["jobs"][SUMMARY]["steps"][0]["run"].replace(
                 '!= "2"', '!= "3"')}), "count guard expects 3"),
        ("a workflow with no summary at all is caught",
         lambda w: w["jobs"].pop(SUMMARY), f"no {SUMMARY!r} job"),
    ]:
        got = mutate(fn)
        check_case(any(want in p for p in got),
                   f"{name} ({got[0] if got else 'NOTHING REPORTED'})"[:110])

    total = 1 + 8
    print(f"\n  {total - failures}/{total} workflow-guard cases hold")
    return 1 if failures else 0


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--selftest", action="store_true")
    args = ap.parse_args()
    return selftest() if args.selftest else check()


if __name__ == "__main__":
    sys.exit(main())
