#!/usr/bin/env bash
# .claude/hooks/pre-commit-doc-check.sh
#
# Documentation discipline check for Ghost Trace. Tripwire — not analysis.
#
# Governance source: .claude/CLAUDE.md §5.2 (three-tier hook architecture)
# and §5.3 (enforcement grading by violation class).
#
# Same script runs in three contexts:
#   - git pre-commit hook (enforcement-of-record; install via
#     core.hooksPath per WORKFLOW.md, Phase 7)
#   - Claude Code Stop event hook (in-session feedback, settings.json)
#   - CI workflow step (PR-level enforcement, Phase 7)
# Per CLAUDE.md §5.2: git is the source of truth when surfaces diverge.
#
# Usage:
#   pre-commit-doc-check.sh             check staged .md files (default)
#   pre-commit-doc-check.sh --self-test verify watchlists parse;
#                                       no diff check
#
# Watchlist parsing is delegated to .claude/hooks/_parse_watchlists.py.
# Bash is the entry point and orchestrator; Python is implementation
# detail for the parsers. The split is per the Phase 6 prompt's
# under-200-line allowance.
#
# Three BLOCKING classes (CLAUDE.md §5.3):
#   1. FROZEN-section edits to constitutional-charter.md
#   2. Vocabulary drift (forbidden synonyms from vocabulary-discipline §4)
#   3. Marketing tells (anti-marketing §1, skips charter — see §4 of skill)
# One ADVISORY class:
#   - Ambiguity terms (ambiguity-reducer §1)
#
# Fail-closed: if any watchlist source is missing or unparseable,
# the script exits non-zero. Silent degradation is the failure mode
# this script exists to prevent.
#
# Three design discoveries documented inline (see match_term, the
# eligible_blockquote_lines helper, and the vocabulary-drift check loop):
#   - Word boundary treats hyphen as word-like, so `log` is not flagged
#     inside `decision-log.md` or `event-log` identifiers.
#   - docs/glossary.md is the source of the forbidden-synonym list; it
#     is exempted from the vocabulary-drift check (circular otherwise).
#     Marketing and ambiguity checks still apply to the glossary.
#   - Charter and Ontology quotations expressed as attributed markdown
#     blockquotes (a block containing a line matching
#     `^\s*>\s*—\s*\[(Charter|Ontology)`) are exempt from the marketing
#     check on a per-line basis. Vocabulary-drift and ambiguity checks
#     are not exempted — the Charter itself uses canonical phrases, so
#     a quotation containing a forbidden synonym is still drift. See
#     docs/charter/decision-log.md §0005 and anti-marketing §4.
#
# Known tradeoff (tripwire by design):
#   The grep is literal. A forbidden term inside a canonical phrase
#   ("log" in "primary event log") or a contrastive mention ("metadata"
#   in "provenance is structure, not metadata") will be flagged. The
#   writer verifies context and either rewrites to avoid the bare word
#   or invokes the --no-verify bypass with a decision-log note per
#   CLAUDE.md §5.3. Sophistication lives in skills and agents; this
#   script reports occurrences, not intent.

set -u

REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null)"
if [ -z "$REPO_ROOT" ]; then
  echo "ERROR: not inside a git repository" >&2
  exit 2
fi
cd "$REPO_ROOT" || exit 2

BYPASS_NOTE="note: git --no-verify bypasses this check; bypass is a registrable event per CLAUDE.md §5.3 and must be noted in docs/charter/decision-log.md with justification."
trap 'echo; echo "$BYPASS_NOTE"' EXIT

CLAUDE_MD=".claude/CLAUDE.md"
VOCAB_SKILL=".claude/skills/ontology/vocabulary-discipline/SKILL.md"
MARKETING_SKILL=".claude/skills/enforcement/anti-marketing/SKILL.md"
AMBIGUITY_SKILL=".claude/skills/epistemic/ambiguity-reducer/SKILL.md"
CHARTER="docs/charter/constitutional-charter.md"
PARSER=".claude/hooks/_parse_watchlists.py"
EXIT_CODE=0

fail_closed() { echo "ERROR (fail-closed): $1" >&2; EXIT_CODE=2; }

for f in "$CLAUDE_MD" "$VOCAB_SKILL" "$MARKETING_SKILL" "$AMBIGUITY_SKILL" "$CHARTER" "$PARSER"; do
  [ -f "$f" ] || fail_closed "missing source: $f"
done
[ "$EXIT_CODE" -ne 0 ] && exit "$EXIT_CODE"

command -v python3 >/dev/null 2>&1 || { fail_closed "python3 not found"; exit "$EXIT_CODE"; }

FROZEN_RANGES=$(python3 "$PARSER" --frozen-ranges "$CHARTER" "$CLAUDE_MD") || { fail_closed "frozen-range parser failed"; exit "$EXIT_CODE"; }
FORBIDDEN=$(python3 "$PARSER" --forbidden "$VOCAB_SKILL")                 || { fail_closed "forbidden parser failed";   exit "$EXIT_CODE"; }
MARKETING=$(python3 "$PARSER" --marketing "$MARKETING_SKILL")             || { fail_closed "marketing parser failed";    exit "$EXIT_CODE"; }
AMBIGUITY=$(python3 "$PARSER" --ambiguity "$AMBIGUITY_SKILL")             || { fail_closed "ambiguity parser failed";    exit "$EXIT_CODE"; }

if [ "${1:-}" = "--self-test" ]; then
  echo "self-test: all watchlist parsers OK"
  echo "  frozen ranges:  $(printf '%s\n' "$FROZEN_RANGES" | grep -c .) range(s)"
  echo "  forbidden:      $(printf '%s\n' "$FORBIDDEN"     | grep -c .) term(s)"
  echo "  marketing:      $(printf '%s\n' "$MARKETING"     | grep -c .) tell(s)"
  echo "  ambiguity:      $(printf '%s\n' "$AMBIGUITY"     | grep -c .) term(s)"
  exit 0
fi

in_scope() {
  case "$1" in
    docs/charter/*|docs/ontology/*|docs/architecture/*|docs/rfcs/*)
      [ "${1##*.}" = "md" ] && return 0 ;;
    docs/glossary.md|CONTRIBUTING.md|README.md|WORKFLOW.md) return 0 ;;
  esac
  return 1
}

STAGED=$(git diff --cached --name-only --diff-filter=AM 2>/dev/null | grep '\.md$' || true)
[ -z "$STAGED" ] && STAGED=$(git diff --name-only --diff-filter=AM 2>/dev/null | grep '\.md$' || true)
if [ -z "$STAGED" ]; then
  echo "doc-check: no .md changes in scope"
  exit 0
fi

# Match a term against content with appropriate boundary discipline.
# Simple alphanumeric word: boundary treats hyphen as word-like, so
# `log` does not match inside `decision-log.md` or `event-log`.
# Hyphenated/multi-word terms: substring grep (specific enough).
# This is a tripwire, not analysis.
match_term() {
  local term="$1" content="$2" pat
  pat=$(printf '%s' "$term" | sed 's/[][\.*^$(){}?+|/]/\\&/g')
  if printf '%s' "$term" | grep -qE '^[a-zA-Z][a-zA-Z0-9]*$'; then
    pat="(^|[^a-zA-Z-])${pat}([^a-zA-Z-]|\$)"
  fi
  printf '%s\n' "$content" | grep -nE "$pat" 2>/dev/null || true
}

# Compute line numbers in $content that belong to attributed blockquote
# blocks. A block is "attributed" if any of its lines matches
# ^[[:space:]]*>[[:space:]]*—[[:space:]]*\[(Charter|Ontology). All
# consecutive lines starting with optional whitespace then `>` form one
# block (a blank or non-blockquote line breaks the block). Per
# decision-log §0005 and anti-marketing §4, the marketing check skips
# watchlist hits on these lines. Per-line scope. Line numbers are 1-based
# and refer to positions within $content (added-lines view or full file
# fallback) — the same base match_term reports against.
eligible_blockquote_lines() {
  local content="$1"
  printf '%s\n' "$content" | awk '
    BEGIN { block_start = 0 }
    {
      if ($0 ~ /^[[:space:]]*>/) {
        if (block_start == 0) block_start = NR
        block[NR] = block_start
        if ($0 ~ /^[[:space:]]*>[[:space:]]*—[[:space:]]*\[(Charter|Ontology)/) {
          attributed[block_start] = 1
        }
      } else {
        block_start = 0
      }
    }
    END {
      for (n in block) {
        if (attributed[block[n]]) print n
      }
    }
  ' | sort -n
}

# Check that any diff hunk touching the charter file intersects no
# FROZEN range. Returns 0 if clean, 1 if a violation is found.
check_frozen_charter() {
  local file="$1" hunk found=0
  while IFS= read -r hunk; do
    [ -z "$hunk" ] && continue
    local old_start old_count old_end
    old_start=$(printf '%s' "$hunk" | sed -E 's/^@@ -([0-9]+).*$/\1/')
    old_count=$(printf '%s' "$hunk" | sed -nE 's/^@@ -[0-9]+,([0-9]+).*$/\1/p')
    [ -z "$old_count" ] && old_count=1
    old_end=$((old_start + old_count - 1))
    [ "$old_count" -eq 0 ] && old_end=$old_start
    local range fs fe ms me
    for range in $FROZEN_RANGES; do
      fs=$(printf '%s' "$range" | cut -d: -f1)
      fe=$(printf '%s' "$range" | cut -d: -f2)
      ms=$old_start; [ "$fs" -gt "$ms" ] && ms=$fs
      me=$old_end;   [ "$fe" -lt "$me" ] && me=$fe
      if [ "$ms" -le "$me" ]; then
        echo "BLOCK [frozen-section-edit] $file lines $old_start-$old_end intersect FROZEN range $range"
        echo "  → Edits to FROZEN Charter sections require RFC (charter-amendment) + amendments.md entry + version bump."
        found=1
      fi
    done
  done < <(git diff --cached --unified=0 -- "$file" 2>/dev/null | grep -E '^@@')
  return $found
}

for file in $STAGED; do
  in_scope "$file" || continue

  added=$(git diff --cached -- "$file" 2>/dev/null | grep -E '^\+[^+]' | sed 's/^+//')
  [ -z "$added" ] && added=$(cat "$file" 2>/dev/null || true)

  if [ "$file" = "$CHARTER" ]; then
    check_frozen_charter "$file" || EXIT_CODE=1
  fi

  # docs/glossary.md is the SOURCE of the forbidden-synonym list (every
  # entry's "Forbidden synonyms" field lists the terms verbatim). Checking
  # the glossary against itself is circular — every entry would trip. The
  # marketing and ambiguity checks still apply to the glossary; only
  # vocabulary-drift is exempted here.
  if [ "$file" != "docs/glossary.md" ]; then
    while IFS= read -r term; do
      [ -z "$term" ] && continue
      bareterm=$(printf '%s' "$term" | sed -E 's/[[:space:]]*\([^)]*\).*$//')
      hits=$(match_term "$bareterm" "$added")
      if [ -n "$hits" ]; then
        echo "BLOCK [vocabulary-drift] $file: forbidden synonym '$bareterm'"
        printf '%s\n' "$hits" | sed 's/^/    /'
        echo "  → see docs/glossary.md and vocabulary-discipline §4 for canonical replacement."
        EXIT_CODE=1
      fi
    done <<< "$FORBIDDEN"
  fi

  if [ "$file" != "$CHARTER" ]; then
    eligible_lines=$(eligible_blockquote_lines "$added")
    while IFS= read -r tell; do
      [ -z "$tell" ] && continue
      raw_hits=$(match_term "$tell" "$added")
      [ -z "$raw_hits" ] && continue
      filtered_hits=$(printf '%s\n' "$raw_hits" | while IFS= read -r hit; do
        [ -z "$hit" ] && continue
        ln=${hit%%:*}
        if [ -n "$eligible_lines" ] && printf '%s\n' "$eligible_lines" | grep -qx -- "$ln"; then
          continue
        fi
        printf '%s\n' "$hit"
      done)
      if [ -n "$filtered_hits" ]; then
        echo "BLOCK [marketing] $file: marketing tell '$tell'"
        printf '%s\n' "$filtered_hits" | sed 's/^/    /'
        echo "  → see anti-marketing §3 for rewrite paths."
        EXIT_CODE=1
      fi
    done <<< "$MARKETING"
  fi

  while IFS= read -r term; do
    [ -z "$term" ] && continue
    hits=$(match_term "$term" "$added")
    if [ -n "$hits" ]; then
      echo "NOTE [ambiguity] $file: term '$term' may need operationalization (advisory)"
      printf '%s\n' "$hits" | sed 's/^/    /'
    fi
  done <<< "$AMBIGUITY"
done

exit "$EXIT_CODE"
