#!/usr/bin/env python3
"""Derive the next version and changelog from the commit log.

R1.13 adopted Conventional Commits explicitly so that releases could be
derived from them, and then no release automation was ever written. This
is it.

WHY NOT semantic-release. It would be a Node toolchain, a plugin set and
a config file, to do what the rules already written in
check-commit-message.sh describe in four lines: feat is a minor, fix and
perf are patches, `!` or a BREAKING CHANGE footer is a major, everything
else releases nothing. What it would buy is other people's edge cases;
what it would cost is a supply chain in the one workflow that can push a
tag.

THE PARSING RULE IS NOT DUPLICATED HERE. The header regex is read from
check-commit-message.sh, which is what the commit hook and the pull
request check both use. A release that read headers by its own slightly
different rule would silently skip commits those two accepted, and the
version would drift from the history it claims to describe.

Pull requests are squash-merged, so the commit that lands is the PR
TITLE. That is what this reads.

Usage:
    next-release.py                 # print the plan
    next-release.py --json          # machine-readable, for the workflow
    next-release.py --selftest      # assert the derivation rules
"""

from __future__ import annotations

import argparse
import json
import os
import re
import subprocess
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
CHECKER = os.path.join(ROOT, "scripts", "check-commit-message.sh")

# The tag this repository's semantic history starts from. Before it the
# log used a different convention entirely (§-numbered decision
# entries), so it is the honest floor rather than the repository's first
# commit.
BASE_TAG = "v1-archive"

SEMVER_TAG = re.compile(r"^v(\d+)\.(\d+)\.(\d+)$")

# What each type releases. Sourced from check-commit-message.sh's own
# doc comment, and asserted against its type list by the selftest so a
# type added there without a rule here cannot go unnoticed.
MINOR = {"feat"}
PATCH = {"fix", "perf"}


def sh(*args: str) -> str:
    return subprocess.run(args, cwd=ROOT, capture_output=True, text=True,
                          check=True).stdout.strip()


def header_re() -> re.Pattern[str]:
    return re.compile(sh("bash", CHECKER, "--header-re"))


def known_types() -> set[str]:
    return set(sh("bash", CHECKER, "--types").split("|"))


def last_release() -> str | None:
    """The newest vX.Y.Z tag reachable from HEAD."""
    try:
        tags = sh("git", "tag", "--merged", "HEAD").splitlines()
    except subprocess.CalledProcessError:
        return None
    versions = [t for t in (t.strip() for t in tags) if SEMVER_TAG.match(t)]
    if not versions:
        return None
    return max(versions, key=lambda t: tuple(int(x) for x in SEMVER_TAG.match(t).groups()))


def commits_since(ref: str | None) -> list[str]:
    span = f"{ref}..HEAD" if ref else "HEAD"
    out = sh("git", "log", "--no-merges", "--format=%s", span)
    return [line for line in out.splitlines() if line.strip()]


def classify(subjects: list[str], pattern: re.Pattern[str]) -> tuple[dict, list[str]]:
    """Group parseable headers by type; return the unparseable too.

    Unparseable subjects are RETURNED rather than dropped. They are
    commits the release cannot describe, and a changelog that silently
    omits them would claim completeness it does not have.
    """
    grouped: dict[str, list[str]] = {}
    unparsed: list[str] = []
    for s in subjects:
        m = pattern.match(s)
        if not m:
            unparsed.append(s)
            continue
        grouped.setdefault(m.group(1), []).append(s)
    return grouped, unparsed


def bump(subjects: list[str], pattern: re.Pattern[str]) -> str | None:
    """major, minor, patch — or None when nothing releases."""
    level = None
    for s in subjects:
        m = pattern.match(s)
        if not m:
            continue
        if m.group(3) == "!":
            return "major"
        t = m.group(1)
        if t in MINOR:
            level = "minor"
        elif t in PATCH and level is None:
            level = "patch"
    return level


def next_version(current: str | None, level: str) -> str:
    if current is None:
        # No prior semantic tag. Starting at 0.1.0 rather than 1.0.0 is
        # the honest reading of a repository whose central claim is
        # still that six numbers reproduce, one of which has no data.
        return "v0.1.0"
    major, minor, patch = (int(x) for x in SEMVER_TAG.match(current).groups())
    if level == "major":
        return f"v{major + 1}.0.0"
    if level == "minor":
        return f"v{major}.{minor + 1}.0"
    return f"v{major}.{minor}.{patch + 1}"


HEADINGS = {
    "feat": "Features", "fix": "Fixes", "perf": "Performance",
    "refactor": "Refactoring", "docs": "Documentation", "test": "Tests",
    "build": "Build", "ci": "CI", "chore": "Chores", "style": "Style",
    "revert": "Reverts",
}


def changelog(grouped: dict, unparsed: list[str], pattern: re.Pattern[str]) -> str:
    lines: list[str] = []

    breaking = [s for subs in grouped.values() for s in subs
                if (m := pattern.match(s)) and m.group(3) == "!"]
    if breaking:
        lines.append("### Breaking")
        lines += [f"- {s}" for s in breaking]
        lines.append("")

    for t in ("feat", "fix", "perf", "refactor", "docs", "test", "build", "ci",
              "chore", "style", "revert"):
        if t in grouped:
            lines.append(f"### {HEADINGS[t]}")
            lines += [f"- {s}" for s in grouped[t]]
            lines.append("")

    if unparsed:
        # Named, not hidden. These are commits the release cannot
        # describe, and pretending otherwise makes the changelog a
        # summary of what happened to parse.
        lines.append("### Not classified")
        lines.append("")
        lines.append("Headers that do not match Conventional Commits, listed so the "
                     "changelog does not claim to be complete when it is not:")
        lines += [f"- {s}" for s in unparsed]
        lines.append("")

    return "\n".join(lines).strip()


def plan() -> dict:
    pattern = header_re()
    current = last_release()
    base = current or (BASE_TAG if tag_exists(BASE_TAG) else None)
    subjects = commits_since(base)
    level = bump(subjects, pattern)
    grouped, unparsed = classify(subjects, pattern)
    return {
        "previous": current,
        "since": base,
        "commits": len(subjects),
        "bump": level,
        "version": next_version(current, level) if level else None,
        "changelog": changelog(grouped, unparsed, pattern) if level else "",
        "unclassified": len(unparsed),
    }


def tag_exists(tag: str) -> bool:
    return subprocess.run(["git", "rev-parse", "-q", "--verify", f"refs/tags/{tag}"],
                          cwd=ROOT, capture_output=True).returncode == 0


# ---------------------------------------------------------------
# selftest
# ---------------------------------------------------------------

def selftest() -> int:
    pattern = header_re()
    failures = 0

    def check(ok: bool, what: str) -> None:
        nonlocal failures
        print(f"  {'ok  ' if ok else 'FAIL'}  {what}")
        failures += not ok

    for subjects, want, name in [
        (["feat(a): x"], "minor", "feat releases a minor"),
        (["fix(a): x"], "patch", "fix releases a patch"),
        (["perf(a): x"], "patch", "perf releases a patch"),
        (["docs(a): x"], None, "docs releases nothing"),
        (["chore(a): x", "ci(b): y"], None, "chores and ci release nothing"),
        (["feat(a)!: x"], "major", "a bang releases a major"),
        (["fix(a)!: x"], "major", "a bang beats the type"),
        (["docs(a): x", "feat(b): y"], "minor", "the highest bump wins"),
        (["feat(a): x", "fix(b): y"], "minor", "minor beats patch"),
        (["feat(a): x", "fix(b)!: y"], "major", "major beats everything"),
        (["not a conventional header"], None, "an unparseable header releases nothing"),
    ]:
        check(bump(subjects, pattern) == want, name)

    for current, level, want in [
        (None, "minor", "v0.1.0"),
        (None, "major", "v0.1.0"),
        ("v0.1.0", "patch", "v0.1.1"),
        ("v0.1.9", "minor", "v0.2.0"),
        ("v0.2.3", "major", "v1.0.0"),
        ("v1.4.2", "patch", "v1.4.3"),
    ]:
        got = next_version(current, level)
        check(got == want, f"{current or 'no tag'} + {level} -> {got}")

    # Every type the checker accepts must have a release rule and a
    # changelog heading. A type added there and forgotten here would
    # silently never release and never appear.
    for t in sorted(known_types()):
        check(t in HEADINGS, f"type {t!r} has a changelog heading")
    check(MINOR | PATCH <= known_types(),
          "every type with a release rule is one the checker accepts")

    # Unparseable commits are listed, not dropped.
    grouped, unparsed = classify(["feat(a): x", "garbage"], pattern)
    log = changelog(grouped, unparsed, pattern)
    check("garbage" in log, "an unparseable commit still appears in the changelog")

    total = 11 + 6 + len(known_types()) + 2
    print(f"\n  {total - failures}/{total} release-derivation cases hold")
    return 1 if failures else 0


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--json", action="store_true")
    ap.add_argument("--selftest", action="store_true")
    args = ap.parse_args()

    if args.selftest:
        return selftest()

    p = plan()
    if args.json:
        print(json.dumps(p))
        return 0

    print(f"  previous release: {p['previous'] or '(none)'}")
    print(f"  commits since {p['since'] or 'the beginning'}: {p['commits']}")
    if p["previous"] is None and p["bump"]:
        # Saying "bump: major" beside "version: v0.1.0" reads as a bug.
        # The first semantic release is a floor, not a computation.
        print(f"  bump: {p['bump']} (not applied — the first semantic release "
              f"starts the series)")
    else:
        print(f"  bump: {p['bump'] or '(nothing releases)'}")
    print(f"  next version: {p['version'] or '(no release)'}")
    if p["unclassified"]:
        print(f"  unclassified headers: {p['unclassified']} (listed in the changelog)")
    if p["changelog"]:
        print()
        print(p["changelog"])
    return 0


if __name__ == "__main__":
    sys.exit(main())
