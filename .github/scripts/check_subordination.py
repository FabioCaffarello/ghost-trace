#!/usr/bin/env python3
# .github/scripts/check_subordination.py
#
# CI helper for the constitutional-check workflow. Mechanical check that
# every reference to constitutional-charter.md from a changed in-scope
# file resolves to a real anchor (heading) in the Charter.
#
# This is grep + anchor extraction — no judgment. The semantic check
# (does the lower doc contradict the higher one?) belongs to the
# subordination-checker skill and the human reviewer.
#
# Usage: invoked from .github/workflows/constitutional-check.yml.
# Expects:
#   - cwd is the repository root
#   - environment variable BASE_REF names the base branch (e.g., 'main')
# Exits non-zero if a broken cross-reference is found.

import os
import re
import subprocess
import sys
from pathlib import Path

CHARTER = "docs/charter/constitutional-charter.md"
SCOPE_DIRS = ("docs/ontology/", "docs/architecture/", "docs/rfcs/",
              "schemas/", "services/")


def fail(msg):
    print(f"ERROR: {msg}", file=sys.stderr)
    sys.exit(2)


def github_anchor(heading_text):
    """Compute the GitHub-style anchor for a heading."""
    # Lowercase, replace spaces with '-', strip punctuation except '-' and '_'.
    s = heading_text.lower()
    s = re.sub(r'[^\w\s-]', '', s)
    s = re.sub(r'\s+', '-', s.strip())
    return s


def charter_anchors():
    """Extract all heading anchors from the Charter."""
    p = Path(CHARTER)
    if not p.exists():
        fail(f"Charter not found at {CHARTER}")
    anchors = set()
    for line in p.read_text(encoding="utf-8").splitlines():
        m = re.match(r'^(#{1,6})\s+(.*)$', line)
        if m:
            anchors.add(github_anchor(m.group(2)))
    return anchors


def changed_files(base_ref):
    """List files changed in this PR relative to base_ref."""
    result = subprocess.run(
        ["git", "diff", "--name-only", f"origin/{base_ref}...HEAD"],
        capture_output=True, text=True, check=False
    )
    if result.returncode != 0:
        fail(f"git diff failed: {result.stderr.strip()}")
    return [f for f in result.stdout.splitlines() if f.endswith(".md")]


def in_scope(path):
    return any(path.startswith(d) for d in SCOPE_DIRS)


def extract_references(file_path):
    """Extract (line_number, anchor) tuples for all Charter cross-references."""
    refs = []
    p = Path(file_path)
    if not p.exists():
        return refs
    pattern = re.compile(r'constitutional-charter\.md#([a-z0-9\-_]+)')
    for i, line in enumerate(p.read_text(encoding="utf-8").splitlines(), 1):
        for m in pattern.finditer(line):
            refs.append((i, m.group(1)))
    return refs


def main():
    base_ref = os.environ.get("BASE_REF") or "main"
    print(f"check_subordination: base_ref={base_ref}")
    anchors = charter_anchors()
    print(f"check_subordination: {len(anchors)} Charter heading anchors loaded")
    files = changed_files(base_ref)
    in_scope_files = [f for f in files if in_scope(f)]
    print(f"check_subordination: {len(in_scope_files)} changed in-scope .md files")
    findings = []
    for f in in_scope_files:
        for line_no, anchor in extract_references(f):
            if anchor not in anchors:
                findings.append((f, line_no, anchor))
    if findings:
        print("")
        print("BROKEN cross-references to docs/charter/constitutional-charter.md:")
        for f, line_no, anchor in findings:
            print(f"  {f}:{line_no} -> #{anchor} (not a heading in the Charter)")
        sys.exit(1)
    print("check_subordination: all Charter cross-references resolve")


if __name__ == "__main__":
    main()
