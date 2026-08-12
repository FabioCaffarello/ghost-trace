#!/usr/bin/env python3
"""Assert that every Go module is in all four lists that must name it,
and that they agree about the archive-format pin.

A module has to be written down in four places, and three of the four
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
  dependabot        forgetting it means the module receives no security
                    directories updates, ever. Seventeen of nineteen sat
                    that way until PR-5.0e, including every module that
                    speaks to the network.

This exists because adding tools/loadgen required touching all of them
by hand, which is the moment to notice that nothing was checking.

THE ARCHIVE-FORMAT PIN is checked here too, because it is the same
failure in a different list. google.golang.org/protobuf produces the
canonical bytes that are hashed for content-addressed identity, so a
bump is an archive-compatibility event. It was declared at three
different versions at once — v1.36.0 in libs/canonical carrying the pin
comment, v1.36.11 in every service that actually marshals — which meant
the golden-hash tripwire verified a version no binary shipped. One
version, asserted, or the pin is decoration.

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


def in_dependabot(doc: dict) -> set[str]:
    """Every Go module directory dependabot is configured to update.

    Both spellings are read — `directory` for a single path and
    `directories` for many — because reading only one would silently
    report zero coverage as full coverage."""
    found: set[str] = set()
    for entry in (doc or {}).get("updates", []) or []:
        if entry.get("package-ecosystem") != "gomod":
            continue
        paths = entry.get("directories") or [entry.get("directory")]
        found |= {str(p).strip("/") for p in paths if p}
    return found


# The module whose version IS the archive format. Not a dependency.
PIN = "google.golang.org/protobuf"
PIN_RE = re.compile(r"^\s*(?:require\s+)?" + re.escape(PIN) + r"\s+(v[\d.]+)", re.M)


def pinned(disk: set[str], root: str = ROOT) -> dict[str, str]:
    """Every module's declared version of the archive-format pin."""
    out: dict[str, str] = {}
    for mod in sorted(disk):
        with open(os.path.join(root, mod, "go.mod")) as fh:
            m = PIN_RE.search(fh.read())
        if m:
            out[mod] = m.group(1)
    return out


def compare_pin(versions: dict[str, str]) -> list[str]:
    """One version across every module that declares it, or none."""
    distinct = sorted(set(versions.values()))
    if len(distinct) <= 1:
        return []
    where = "; ".join(f"{v}: {', '.join(m for m, mv in sorted(versions.items()) if mv == v)}"
                      for v in distinct)
    return [
        f"{PIN} is declared at {len(distinct)} different versions — {where}. "
        f"It produces the canonical bytes that are hashed for identity, so the "
        f"golden-hash tripwire verifies whichever version its OWN module "
        f"resolves; if that is not the version the services ship, the guard is "
        f"aimed at code nothing runs."
    ]


def compare(disk: set[str], work: set[str], make: set[str], ci: set[str],
            bot: set[str] | None = None) -> list[str]:
    out: list[str] = []
    lists = [
        ("go.work `use`", work, "it is not in the workspace"),
        ("Makefile GO_MODULES", make,
         "`make verify` never vets, lints, tests or scans it"),
        ("ci.yml modules matrix", ci,
         "it is never built standalone, which is what container builds do"),
    ]
    if bot is not None:
        lists.append((".github/dependabot.yml directories", bot,
                      "it receives no security updates, ever"))
    for label, listed, consequence in lists:
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
    with open(os.path.join(ROOT, ".github", "dependabot.yml")) as fh:
        bot = in_dependabot(yaml.safe_load(fh))

    versions = pinned(disk)
    problems = compare(disk, work, make, ci, bot) + compare_pin(versions)
    for p in problems:
        print(f"  {p}")
    if problems:
        print(f"\n  {len(problems)} problem(s); a module missing from a list is a "
              f"module that list does not cover")
        return 1
    print(f"  {len(disk)} module(s), each named in go.work, GO_MODULES, the CI "
          f"matrix and dependabot")
    print(f"  {PIN} declared at {sorted(set(versions.values()))[0]} "
          f"by all {len(versions)} module(s) that name it")
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
        ("a module missing from dependabot",
         (full, full, full, full, {"libs/a"}), "no security updates"),
        ("a list naming a module that does not exist",
         (full, full | {"libs/gone"}, full, full), "has no go.mod"),
    ]:
        got = compare(*args)
        case(any(want in p for p in got),
             f"{name} ({got[0] if got else 'NOTHING REPORTED'})"[:108])

    # The archive-format pin. Disagreement is the state this repository
    # was actually in: v1.36.0 where the tripwire runs, v1.36.11 where
    # the bytes are produced.
    case(compare_pin({"libs/canonical": "v1.36.11", "services/collector": "v1.36.11"}) == [],
         "one version across every module reports nothing")
    case(compare_pin({}) == [], "no module declaring the pin reports nothing")
    split = compare_pin({"libs/canonical": "v1.36.0", "services/collector": "v1.36.11"})
    case(any("2 different versions" in p for p in split),
         f"two versions of the pin is refused ({split[0][:64] if split else 'NOTHING REPORTED'})")

    # And the parser must read a real go.mod, or the three cases above
    # compare hand-written dicts and prove nothing about the tree.
    real = pinned(on_disk())
    case(len(real) > 5, f"the pin is parsed out of {len(real)} real go.mod files")

    # The parsers must read the real files, or the comparison above is
    # checking two empty sets against each other and passing.
    with open(os.path.join(ROOT, "go.work")) as fh:
        work = in_go_work(fh.read())
    with open(os.path.join(ROOT, "Makefile")) as fh:
        make = in_makefile(fh.read())
    with open(os.path.join(ROOT, ".github", "workflows", "ci.yml")) as fh:
        ci = in_ci(yaml.safe_load(fh))
    with open(os.path.join(ROOT, ".github", "dependabot.yml")) as fh:
        bot = in_dependabot(yaml.safe_load(fh))
    case(len(work) > 5, f"go.work parses to {len(work)} modules")
    case(len(make) > 5, f"GO_MODULES parses to {len(make)} modules")
    case(len(ci) > 5, f"the CI matrix parses to {len(ci)} modules")
    case(len(bot) > 5, f"dependabot parses to {len(bot)} modules")
    case(len(on_disk()) > 5, f"the filesystem walk finds {len(on_disk())} modules")

    total = 1 + 5 + 4 + 5
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
