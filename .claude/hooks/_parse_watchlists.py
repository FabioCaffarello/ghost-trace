#!/usr/bin/env python3
# .claude/hooks/_parse_watchlists.py
#
# Watchlist parsers for pre-commit-doc-check.sh. Single-purpose helper.
# Bash is the entry point and orchestrator; this script does only the
# parsing that bash + sed/awk would do awkwardly.
#
# Governance source: .claude/CLAUDE.md §5.2 (three-tier hook architecture).
# Per the prompt's split-allowance: bash stays the entry point and this
# is implementation detail.
#
# Each invocation does one job and prints one item per line on stdout,
# or fails non-zero on parse error.
#
# Watchlist source assumption: skill markdown files keep their watchlists
# in the sections referenced below. If a skill's table later moves to
# `references/`, the corresponding parser here must be updated to follow
# the indirection. Until then, watchlists are inline in the SKILL.md.

import argparse
import re
import sys
from pathlib import Path


def fail(msg):
    print(f"ERROR (parser): {msg}", file=sys.stderr)
    sys.exit(2)


def read(path):
    p = Path(path)
    if not p.exists():
        fail(f"missing file: {path}")
    return p.read_text(encoding="utf-8")


def section(text, start_re, end_re):
    """Return the substring from the start_re match to the end_re match."""
    start = re.search(start_re, text, re.MULTILINE)
    if not start:
        return None
    rest = text[start.start():]
    end = re.search(end_re, rest, re.MULTILINE)
    return rest[:end.start()] if end else rest


def parse_frozen_sections(claude_md):
    text = read(claude_md)
    s = section(text, r'^## 4\. Charter status', r'^## 5\.')
    if s is None:
        fail("CLAUDE.md §4 (Charter status) section not found")
    markers = []
    for m in re.finditer(r'^\|\s*(§[0-9.]+)[^|]*\|\s*frozen\s*\|', s, re.MULTILINE):
        markers.append(m.group(1))
    if not markers:
        fail("no frozen sections in CLAUDE.md §4")
    return markers


def parse_frozen_ranges(charter_path, claude_md):
    markers = parse_frozen_sections(claude_md)
    charter_lines = read(charter_path).splitlines()
    headings = []
    for i, line in enumerate(charter_lines):
        m = re.match(r'^(#{2,3})\s+([0-9]+(?:\.[0-9]+)?)[.\s]', line)
        if m:
            headings.append((i, len(m.group(1)), m.group(2)))
    ranges = []
    for marker in markers:
        num = marker.lstrip('§')
        start_idx = start_lvl = None
        for idx, lvl, n in headings:
            if n == num:
                start_idx, start_lvl = idx, lvl
                break
        if start_idx is None:
            print(f"WARN: {marker} listed FROZEN in CLAUDE.md but not found as heading in {charter_path}", file=sys.stderr)
            continue
        end_idx = len(charter_lines) - 1
        for idx, lvl, n in headings:
            if idx > start_idx and lvl <= start_lvl:
                end_idx = idx - 1
                break
        ranges.append(f"{start_idx + 1}:{end_idx + 1}")
    if not ranges:
        fail("no frozen line ranges computed")
    return ranges


def parse_forbidden(skill_path):
    text = read(skill_path)
    s = section(text, r'^## 4\. Forbidden synonyms', r'^## 5\.')
    if s is None:
        fail(f"vocabulary-discipline §4 not found in {skill_path}")
    terms = []
    for m in re.finditer(r'^\|\s*`([^`]+)`', s, re.MULTILINE):
        terms.append(m.group(1))
    if not terms:
        fail("no forbidden synonyms extracted")
    return terms


def parse_marketing(skill_path):
    """Extract marketing tells from §1 of anti-marketing/SKILL.md.

    Per the Phase 8 SELF-AUDIT Fix 8.1: only the first paragraph after each
    `### <category>` heading is treated as the tells paragraph (terminated
    by the first `> ` example block). Explanation paragraphs after the
    example are excluded — they routinely mention watchlist terms or
    skill names as data, which the previous parser captured as spurious
    tells.
    """
    text = read(skill_path)
    s = section(text, r'^## 1\. Marketing tells', r'^## 2\.')
    if s is None:
        fail(f"anti-marketing §1 not found in {skill_path}")
    tells = []
    seen = set()
    in_tells_paragraph = False
    for line in s.splitlines():
        stripped = line.lstrip()
        if stripped.startswith('### '):
            in_tells_paragraph = True
            continue
        if stripped.startswith('> '):
            in_tells_paragraph = False
            continue
        if in_tells_paragraph and stripped.startswith('`'):
            for m in re.finditer(r'`([^`]+)`', stripped):
                t = m.group(1)
                if t not in seen:
                    seen.add(t)
                    tells.append(t)
    if not tells:
        fail("no marketing tells extracted")
    return tells


def parse_ambiguity(skill_path):
    text = read(skill_path)
    s = section(text, r'^## 1\. Watchlist', r'^## 2\.')
    if s is None:
        fail(f"ambiguity-reducer §1 not found in {skill_path}")
    terms = []
    for m in re.finditer(r'^###\s+`([^`]+)`', s, re.MULTILINE):
        terms.append(m.group(1))
    if not terms:
        fail("no ambiguity terms extracted")
    return terms


def main():
    ap = argparse.ArgumentParser()
    g = ap.add_mutually_exclusive_group(required=True)
    g.add_argument('--frozen-sections', metavar='CLAUDE_MD')
    g.add_argument('--frozen-ranges', nargs=2, metavar=('CHARTER', 'CLAUDE_MD'))
    g.add_argument('--forbidden', metavar='SKILL_MD')
    g.add_argument('--marketing', metavar='SKILL_MD')
    g.add_argument('--ambiguity', metavar='SKILL_MD')
    args = ap.parse_args()
    items = []
    if args.frozen_sections:
        items = parse_frozen_sections(args.frozen_sections)
    elif args.frozen_ranges:
        items = parse_frozen_ranges(args.frozen_ranges[0], args.frozen_ranges[1])
    elif args.forbidden:
        items = parse_forbidden(args.forbidden)
    elif args.marketing:
        items = parse_marketing(args.marketing)
    elif args.ambiguity:
        items = parse_ambiguity(args.ambiguity)
    for it in items:
        print(it)


if __name__ == "__main__":
    main()
