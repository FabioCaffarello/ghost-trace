#!/usr/bin/env python3
"""Assert that every Go module is in all three lists that must name it.

A module has to be written down in three places, and two of the three
fail SILENTLY when it is not:

  go.work `use`     forgetting it breaks local builds immediately. Loud,
                    self-correcting, harmless.
  Makefile          forgetting it means `make verify` never vets, lints,
  GO_MODULES        tests or vulnerability-scans that module. Nothing
                    says so. The module is simply not covered, and the
                    gate stays green because the gate never looked.
  ci.yml modules    forgetting it means the module is never built
                    matrix    STANDALONE. That is what container builds
                    do — each image copies libs/ and one service — so a
                    missing `replace` or a workspace-only dependency
                    surfaces at image build time rather than in CI.

This exists because adding tools/loadgen required touching all three by
hand, which is the moment to notice that nothing was checking.

The filesystem is the source of truth: every directory containing a
go.mod must appear in all three lists, and no list may name a module
that does not exist.

Usage:
    check-modules.py              # check
    check-modules.py --selftest   # assert the checks catch what they claim
"""

from __future__ import annotations

import argparse
import os
import re
import sys

import yaml

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

# Directories never walked looking for modules.
SKIP = {".git", "node_modules", "vendor", ".venv", "__pycache__", ".tmp"}


def on_disk(root: str = ROOT) -> set[str]:
    """Every directory holding a go.mod, relative to the repository root."""
    found = set()
    for dirpath, dirnames, filenames in os.walk(root):
        dirnames[:] = [d for d in dirnames if d not in SKIP and not d.startswith(".")]
        if "go.mod" in filenames and os.path.abspath(dirpath) != os.path.abspath(root):
            found.add(os.path.relpath(dirpath, root))
    return found


def in_go_work(text: str) -> set[str]:
    block = re.search(r"use\s*\((.*?)\)", text, re.S)
    if not block:
        return set()
    return {line.strip().lstrip("./") for line in block.group(1).splitlines()
            if line.strip() and not line.strip().startswith("//")}


def in_makefile(text: str) -> set[str]:
    m = re.search(r"^GO_MODULES\s*:=\s*(.*)$", text, re.M)
    return set(m.group(1).split()) if m else set()


def in_ci(doc: dict) -> set[str]:
    try:
        return set(doc["jobs"]["modules"]["strategy"]["matrix"]["module"])
    except (KeyError, TypeError):
        return set()


def compare(disk: set[str], work: set[str], make: set[str], ci: set[str]) -> list[str]:
    out: list[str] = []
    for label, listed, consequence in (
        ("go.work `use`", work, "it is not in the workspace"),
        ("Makefile GO_MODULES", make,
         "`make verify` never vets, lints, tests or scans it"),
        ("ci.yml modules matrix", ci,
         "it is never built standalone, which is what container builds do"),
    ):
        for missing in sorted(disk - listed):
            out.append(f"{missing} is a module but is not in {label} — {consequence}")
        for ghost in sorted(listed - disk):
            out.append(f"{label} names {ghost}, which has no go.mod")
    return out


def check() -> int:
    disk = on_disk()
    with open(os.path.join(ROOT, "go.work")) as fh:
        work = in_go_work(fh.read())
    with open(os.path.join(ROOT, "Makefile")) as fh:
        make = in_makefile(fh.read())
    with open(os.path.join(ROOT, ".github", "workflows", "ci.yml")) as fh:
        ci = in_ci(yaml.safe_load(fh))

    problems = compare(disk, work, make, ci)
    for p in problems:
        print(f"  {p}")
    if problems:
        print(f"\n  {len(problems)} problem(s); a module missing from a list is a "
              f"module that list does not cover")
        return 1
    print(f"  {len(disk)} module(s), each named in go.work, GO_MODULES and the CI matrix")
    return 0


def selftest() -> int:
    failures = 0

    def case(ok: bool, what: str) -> None:
        nonlocal failures
        print(f"  {'ok  ' if ok else 'FAIL'}  {what}")
        failures += not ok

    full = {"libs/a", "services/b"}
    case(compare(full, full, full, full) == [], "three agreeing lists report nothing")

    for name, args, want in [
        ("a module missing from go.work",
         (full, {"libs/a"}, full, full), "not in go.work"),
        ("a module missing from GO_MODULES",
         (full, full, {"libs/a"}, full), "never vets, lints, tests or scans"),
        ("a module missing from the CI matrix",
         (full, full, full, {"libs/a"}), "never built standalone"),
        ("a list naming a module that does not exist",
         (full, full | {"libs/gone"}, full, full), "has no go.mod"),
    ]:
        got = compare(*args)
        case(any(want in p for p in got),
             f"{name} ({got[0] if got else 'NOTHING REPORTED'})"[:108])

    # The parsers must read the real files, or the comparison above is
    # checking two empty sets against each other and passing.
    with open(os.path.join(ROOT, "go.work")) as fh:
        work = in_go_work(fh.read())
    with open(os.path.join(ROOT, "Makefile")) as fh:
        make = in_makefile(fh.read())
    with open(os.path.join(ROOT, ".github", "workflows", "ci.yml")) as fh:
        ci = in_ci(yaml.safe_load(fh))
    case(len(work) > 5, f"go.work parses to {len(work)} modules")
    case(len(make) > 5, f"GO_MODULES parses to {len(make)} modules")
    case(len(ci) > 5, f"the CI matrix parses to {len(ci)} modules")
    case(len(on_disk()) > 5, f"the filesystem walk finds {len(on_disk())} modules")

    total = 1 + 4 + 4
    print(f"\n  {total - failures}/{total} module-list cases hold")
    return 1 if failures else 0


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--selftest", action="store_true")
    args = ap.parse_args()
    return selftest() if args.selftest else check()


if __name__ == "__main__":
    sys.exit(main())
