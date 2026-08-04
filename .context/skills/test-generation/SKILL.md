---
type: skill
name: Test Generation
description: Generate comprehensive test cases for code. Use when Writing tests for new functionality, Adding tests for bug fixes (regression tests), or Improving test coverage for existing code
skillSlug: test-generation
phases: [E, V]
generated: 2026-08-04
status: filled
scaffoldVersion: "2.0.0"
---
See `.context/agents/test-writer.md`.

- Drive the wire: real HTTP, real JSON, for anything crossing it.
- Show the test **red** before green. Reintroduce the defect it guards.
- Two-way drift guards, no exception lists.
- Goldens freeze **bytes**, not re-marshalled maps. Regenerate with
  `-update` deliberately.
- Python selftests (`analyze.py --selftest`, `python3 -m schema
  --selftest`) assert and exit non-zero. A selftest that always returns
  0 is what R1.13 replaced.
