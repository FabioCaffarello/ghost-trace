#!/usr/bin/env python3
"""Every sensor is a make target, and every gate is a sensor.

`.context/README.md` states the first half as a rule — "Every sensor in
.context/config/sensors.json is a `make` target and nothing else, so
humans, CI and agents share one vocabulary" — and until this script
nothing checked it. A sensor naming a target that does not exist is
worse than no sensor: an agent reads the file, believes a gate covers
the change it is making, and runs a command that fails as "No rule to
make target", or worse, is never run at all.

The second half is the one the 2026-08-10 audit actually found broken.
`sensors.json` listed eleven sensors and not one of the four topology
gates, so the file said the project's strongest claim-checking
machinery did not exist.

Both halves are checked here. The second is a list this script owns:
adding a gate to the Makefile without deciding whether it is a sensor
is the omission being prevented, so the decision must be recorded
somewhere, and "somewhere" is GATES below.
"""

from __future__ import annotations

import json
import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parent.parent
SENSORS = ROOT / ".context/config/sensors.json"
MAKEFILE = ROOT / "Makefile"

# The gates whose absence from sensors.json is a finding. Not every make
# target belongs here — only the ones that decide whether a published
# claim holds. Adding to this list is how a new gate becomes mandatory
# for agents; leaving one off is a decision, and the failure below is
# what forces it to be a deliberate one.
GATES = {
    "numbers": "the six numbers reproduce",
    "shadow-http": "collector and engine agree over real HTTP",
    "kill-test": "each service taken away behaves as promised",
    "loss-audit": "the books balance under an outage",
    "load-gate": "the accounting holds under load",
    "e2e": "the product runs end to end, every link asserted",
}

TARGET = re.compile(r"^([A-Za-z0-9][A-Za-z0-9._-]*)\s*:(?!=)")


def targets(makefile: str) -> set[str]:
    """Target names the Makefile defines.

    Deliberately a parse and not `make -pRrq`: this must give the same
    answer on a machine where the toolchain is half-installed, and a
    checker that needs the thing it checks to be runnable is a checker
    that gets skipped.
    """
    found = set()
    for line in makefile.splitlines():
        # Recipe lines begin with a tab and are rejected twice over: the
        # pattern is anchored, and `match` starts at position zero. An
        # explicit `startswith("\t"): continue` guard stood here and was
        # dead — deleting it left the selftest green, which is how it was
        # found. Either mechanism alone suffices; the selftest turns red
        # only when both go, and then `http:` inside a recipe's URL is
        # parsed as a Makefile target.
        m = TARGET.match(line)
        if m and m.group(1) != ".PHONY":
            found.add(m.group(1))
    return found


def target_of(command: str) -> str | None:
    """The target a sensor's command invokes, or None if the command is
    not a plain `make` invocation — which is itself the violation."""
    parts = command.split()
    if len(parts) < 2 or parts[0] != "make":
        return None
    for word in parts[1:]:
        if "=" not in word and not word.startswith("-"):
            return word
    return None


def check(sensors: dict, makefile: str) -> list[str]:
    known = targets(makefile)
    problems = []
    seen = set()

    for s in sensors.get("sensors", []):
        sid = s.get("id", "<no id>")
        command = s.get("command", "")
        target = target_of(command)
        if target is None:
            problems.append(
                f"sensor {sid!r}: command {command!r} is not `make <target>` — "
                f"sensors are make targets and nothing else")
            continue
        seen.add(target)
        if target not in known:
            problems.append(
                f"sensor {sid!r}: no Makefile target {target!r} — this sensor "
                f"names a gate that does not exist")

    for gate, what in sorted(GATES.items()):
        if gate not in known:
            problems.append(
                f"gate {gate!r} ({what}) is in GATES but the Makefile has no "
                f"such target — rename the gate here, or restore it there")
        elif gate not in seen:
            problems.append(
                f"gate {gate!r} ({what}) is a Makefile target and not a "
                f"sensor — an agent reading sensors.json cannot know it "
                f"must run")

    return problems


def _selftest() -> int:
    mk = ("shadow-http: ## A/B\n\tpython3 x.py\n"
          "kill-test:\n\tpython3 y.py\n"
          "loss-audit:\n"
          "load-gate:\n"
          "numbers:\n"
          "e2e:\n"
          "VAR := notatarget\n")
    every = [{"id": g, "command": f"make {g}"} for g in GATES]

    assert check({"sensors": every}, mk) == [], check({"sensors": every}, mk)

    # A `:=` assignment is not a target, and treating it as one would
    # make this checker pass for names nobody can run.
    assert "notatarget" not in targets(mk)
    assert targets(mk) == set(GATES), targets(mk)

    # A recipe line that happens to contain a colon is not a rule.
    assert targets("a:\n\techo http://x\n") == {"a"}

    missing_target = [{"id": "x", "command": "make ghost"}]
    got = check({"sensors": missing_target + every}, mk)
    assert len(got) == 1 and "does not exist" in got[0], got

    not_make = [{"id": "x", "command": "python3 deploy/kill-test.py"}]
    got = check({"sensors": not_make + every}, mk)
    assert len(got) == 1 and "nothing else" in got[0], got

    # The audit's actual finding: the target exists, is run by hand, and
    # sensors.json does not mention it.
    got = check({"sensors": [s for s in every if s["id"] != "load-gate"]}, mk)
    assert len(got) == 1 and "cannot know it must run" in got[0], got

    # Variables and flags are not the target name.
    assert target_of("make load-gate GATE_ARGS=--duration=20s") == "load-gate"
    assert target_of("make -s numbers") == "numbers"
    assert target_of("make") is None

    # A gate renamed in the Makefile must not pass silently just because
    # no sensor mentions it either.
    got = check({"sensors": []}, "numbers:\n")
    assert any("no such target" in p for p in got), got

    print("  9 pass")
    return 0


def main() -> int:
    if "--selftest" in sys.argv:
        return _selftest()
    problems = check(json.loads(SENSORS.read_text()), MAKEFILE.read_text())
    if problems:
        print(f"  {len(problems)} problem(s):")
        for p in problems:
            print(f"    - {p}")
        return 1
    n = len(json.loads(SENSORS.read_text())["sensors"])
    print(f"  {n} sensors, every one a make target; "
          f"all {len(GATES)} gates are sensors")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
