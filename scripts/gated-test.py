#!/usr/bin/env python3
"""Run a gated test target, and refuse a run that proved nothing.

`make parity`, `make shadow` and `make shadow-http` exist because the
tests they run skip themselves without a broker or a live topology. Each
already refuses to start when the environment is missing — that part was
built after `make shadow` skipping silently is how a broken tenant
lookup reached CI.

Two ways to pass anyway survived that fix, and this script closes both.

**The filter can select nothing.** `make parity` runs `-run Archive`,
and `go test` with a filter that matches no test prints `[no tests to
run]` and **exits 0**. Rename `TestArchiveHoldsEverythingTheCollectorWrote`
to anything not containing "Archive" and the gate becomes a no-op that
reports success. Nothing in the repository would notice.

**A test can still skip for its own reasons.** The target supplies
`GT_NATS_URL`; the test may also want `GT_DEMO_URL`, or bail on core
count. Inside a gated target that is not a neutral outcome: the entire
purpose of the target is to supply what the skip guards against, so a
skip here means the target did not do its job.

So: in a gated run, zero selected tests is a failure, and a skip is a
failure. Ordinary `make test-race` is unaffected — skipping is exactly
right there, which is why this refusal belongs to the gated targets and
not to the test files.
"""

from __future__ import annotations

import argparse
import re
import subprocess
import sys

SKIPPED = re.compile(r"^\s*--- SKIP: (\S+)", re.M)
PASSED = re.compile(r"^\s*--- PASS: (\S+)", re.M)
FAILED = re.compile(r"^\s*--- FAIL: (\S+)", re.M)
NOTHING = "no tests to run"


def verdict(output: str, code: int) -> list[str]:
    """Everything wrong with one gated run."""
    problems = []
    if code != 0:
        problems.append(f"go test exited {code}")
    if NOTHING in output:
        problems.append(
            "the -run filter selected NO tests — `go test` reports this and "
            "exits 0, so the target passed without running anything. A test "
            "was probably renamed out from under the filter")
    skipped = SKIPPED.findall(output)
    if skipped:
        problems.append(
            "skipped inside a gated target: " + ", ".join(sorted(set(skipped))) +
            " — this target exists to supply what the skip guards against, so "
            "a skip means it did not")
    if not PASSED.findall(output) and not FAILED.findall(output):
        problems.append("no test reported a result at all")
    return problems


def _selftest() -> int:
    ok = "=== RUN   TestA\n--- PASS: TestA (0.10s)\nPASS\nok  \tpkg\t0.2s\n"
    assert verdict(ok, 0) == [], verdict(ok, 0)

    nothing = "testing: warning: no tests to run\nPASS\nok  \tpkg\t0.2s [no tests to run]\n"
    got = verdict(nothing, 0)
    assert any("selected NO tests" in p for p in got), got

    skip = ("=== RUN   TestA\n--- PASS: TestA (0.01s)\n"
            "=== RUN   TestB\n    x_test.go:9: GT_DEMO_URL not set\n"
            "--- SKIP: TestB (0.00s)\nPASS\nok\tpkg\n")
    got = verdict(skip, 0)
    assert len(got) == 1 and "TestB" in got[0], got
    assert "TestA" not in got[0], "named a test that did NOT skip"

    # The marker only counts at the start of a line. Go indents it for
    # subtests, so leading space is allowed — but a test that PRINTS the
    # string, which this repository's own gate output does, must not be
    # read as a skip that never happened.
    quoted = ("=== RUN   TestA\n"
              "    gate_test.go:12: refused: --- SKIP: TestGhost (0.00s)\n"
              "--- PASS: TestA (0.01s)\nPASS\nok\tpkg\n")
    assert verdict(quoted, 0) == [], verdict(quoted, 0)

    # A failing run is a failure even though it also ran real tests.
    fail = "=== RUN   TestA\n--- FAIL: TestA (0.01s)\nFAIL\n"
    got = verdict(fail, 1)
    assert len(got) == 1 and "exited 1" in got[0], got

    # An empty package is not a pass.
    assert verdict("PASS\nok\tpkg\t0.1s\n", 0) == ["no test reported a result at all"]

    # Nor is a package whose only PASS-shaped text is inside a log line.
    # Same anchor, same reason as the SKIP case above: quoting a marker
    # must not manufacture the result it names.
    echoed = "=== RUN   TestA\n    x_test.go:3: saw --- PASS: TestGhost\nPASS\nok\tpkg\n"
    assert verdict(echoed, 0) == ["no test reported a result at all"], verdict(echoed, 0)

    # Both faults at once are both reported.
    assert len(verdict(nothing + skip, 1)) == 3

    print("  8 pass")
    return 0


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--selftest", action="store_true")
    ap.add_argument("--dir", default=".", help="module directory to run in")
    ap.add_argument("args", nargs=argparse.REMAINDER,
                    help="go test arguments, after --")
    ns = ap.parse_args()
    if ns.selftest:
        return _selftest()

    argv = [a for a in ns.args if a != "--"]
    # -v is not optional: --- SKIP and --- PASS lines only exist with it.
    cmd = ["go", "test", "-count=1", "-v", *argv]
    proc = subprocess.run(cmd, cwd=ns.dir, capture_output=True, text=True)
    out = proc.stdout + proc.stderr
    sys.stdout.write(out)

    problems = verdict(out, proc.returncode)
    if problems:
        print(f"\n  {len(problems)} reason(s) this run proves nothing:")
        for p in problems:
            print(f"    - {p}")
        return 1
    print(f"\n  {len(PASSED.findall(out))} test(s) ran against the real thing, "
          f"none skipped.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
