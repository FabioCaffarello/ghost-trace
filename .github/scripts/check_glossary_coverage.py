#!/usr/bin/env python3
# .github/scripts/check_glossary_coverage.py
#
# CI helper for the constitutional-check workflow. Verifies that every
# term in .claude/CLAUDE.md §3 (canonical vocabulary) has an entry in
# docs/glossary.md, and that each entry carries the four provenance
# fields required by .claude/skills/ontology/vocabulary-discipline/.
#
# Fail-closed discipline: if either source fails to parse, the job
# fails. Silent degradation is the failure mode the discipline exists
# to prevent.
#
# Usage: invoked from .github/workflows/constitutional-check.yml.
# Expects cwd to be the repository root.

import re
import sys
from pathlib import Path

CLAUDE_MD = ".claude/CLAUDE.md"
GLOSSARY = "docs/glossary.md"

REQUIRED_FIELDS = [
    "Canonical definition.",
    "Introduction.",
    "Stabilization.",
    "Last amendment.",
]


def fail(msg):
    print(f"ERROR: {msg}", file=sys.stderr)
    sys.exit(2)


def read(path):
    p = Path(path)
    if not p.exists():
        fail(f"missing file: {path}")
    return p.read_text(encoding="utf-8")


def parse_canonical_terms():
    """Extract terms from CLAUDE.md §3 bullet list.
    Each bullet starts with: - **<term>** — definition...
    """
    text = read(CLAUDE_MD)
    s = re.search(r'^## 3\. Canonical vocabulary', text, re.MULTILINE)
    if not s:
        fail("CLAUDE.md §3 (Canonical vocabulary) section not found")
    section_text = text[s.start():]
    end = re.search(r'^## 4\.', section_text, re.MULTILINE)
    if end:
        section_text = section_text[:end.start()]
    terms = []
    for m in re.finditer(r'^- \*\*([^*]+)\*\*', section_text, re.MULTILINE):
        terms.append(m.group(1).strip())
    if not terms:
        fail("no canonical terms extracted from CLAUDE.md §3")
    return terms


def parse_glossary_entries():
    """Parse glossary entries. Returns dict: term -> set of field labels present."""
    text = read(GLOSSARY)
    entries = {}
    current_term = None
    current_fields = set()
    for line in text.splitlines():
        # Entry heading: ### `term`
        m = re.match(r'^###\s+`([^`]+)`\s*$', line)
        if m:
            if current_term is not None:
                entries[current_term] = current_fields
            current_term = m.group(1).strip()
            current_fields = set()
            continue
        # New top-level section (## ...) ends the current entry.
        if line.startswith("## "):
            if current_term is not None:
                entries[current_term] = current_fields
                current_term = None
                current_fields = set()
            continue
        if current_term is not None:
            # Field line: - **Field name.** content...
            fm = re.match(r'^-\s+\*\*([^*]+)\*\*', line)
            if fm:
                current_fields.add(fm.group(1).strip())
    if current_term is not None:
        entries[current_term] = current_fields
    if not entries:
        fail("no glossary entries parsed")
    return entries


def main():
    terms = parse_canonical_terms()
    entries = parse_glossary_entries()
    print(f"check_glossary_coverage: {len(terms)} terms in CLAUDE.md §3, {len(entries)} glossary entries")

    missing_terms = []
    incomplete_entries = []
    for term in terms:
        if term not in entries:
            missing_terms.append(term)
            continue
        fields = entries[term]
        missing_fields = [f for f in REQUIRED_FIELDS if f not in fields]
        if missing_fields:
            incomplete_entries.append((term, missing_fields))

    if missing_terms:
        print("")
        print("MISSING glossary entries for canonical terms:")
        for t in missing_terms:
            print(f"  {t}")
    if incomplete_entries:
        print("")
        print("INCOMPLETE glossary entries (missing required fields):")
        for term, missing in incomplete_entries:
            print(f"  {term}: missing {', '.join(missing)}")

    if missing_terms or incomplete_entries:
        sys.exit(1)
    print("check_glossary_coverage: every canonical term has a complete entry")


if __name__ == "__main__":
    main()
