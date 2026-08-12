"""What a gate run can honestly say about itself.

The topology gates (`make loss-audit`, `make load-gate`) produce numbers
that end up quoted in `docs/results/` and in the roadmap. Until this
module they produced them anonymously: a JSON file of measurements with
nothing in it saying which code was measured, on what, or when. The
evidence chain for the load figures terminated at "some checkout, some
afternoon" — which is not a chain.

The subtlety, and the reason this is not four lines of `git rev-parse`:
a topology gate does not measure the working tree. It measures four
containers that were built at some earlier moment. If the tree is dirty,
or `make docker-build` has not run since the last edit, the checkout SHA
names code that did NOT produce the number. So the stamp records both —
the SHA the tree claims, and the image IDs that actually served the
requests — and says plainly when it could not read the second one.

Absence is written as null, never as a plausible-looking default. A
missing image ID means "not known", and a reader of the manifest must be
able to tell that apart from a match.
"""

from __future__ import annotations

import datetime
import json
import os
import platform
import subprocess

SERVICES = ("collector", "decision-engine", "archive", "demo-web")


def _cmd(args: list[str]) -> str | None:
    """Run a command for its stdout, or None if it cannot be run at all.

    None and "" mean different things here and both occur: None is "the
    question could not be asked" (no git, no docker, non-zero exit), ""
    is "asked, and the answer was empty" — which is how a clean tree
    reports itself to `git status --porcelain`.
    """
    try:
        p = subprocess.run(args, capture_output=True, text=True, timeout=15)
    except (OSError, subprocess.SubprocessError):
        return None
    return p.stdout.strip() if p.returncode == 0 else None


def _images() -> dict[str, str | None]:
    """The image ID behind each running service.

    This is the part that says what actually ran. `docker compose ps`
    reports one JSON object per line; a service that is not up simply
    has no line, and stays None.
    """
    out = _cmd(["docker", "compose", "--profile", "core", "ps",
                "--format", "json"])
    found: dict[str, str | None] = {name: None for name in SERVICES}
    if not out:
        return found
    for line in out.splitlines():
        try:
            row = json.loads(line)
        except json.JSONDecodeError:
            continue
        service = row.get("Service")
        if service in found:
            # Image is the tag; ImageID (when present) pins the build.
            found[service] = row.get("ImageID") or row.get("Image") or None
    return found


def stamp(gate: str, parameters: dict | None = None) -> dict:
    """Provenance for one gate run, in the shape `experiments/numbers.py`
    already uses, so a reader meets one format and not two."""
    sha = _cmd(["git", "rev-parse", "HEAD"])
    status = _cmd(["git", "status", "--porcelain"])
    return {
        "gate": gate,
        "generated_at": datetime.datetime.now(datetime.timezone.utc)
                                .replace(microsecond=0).isoformat()
                                .replace("+00:00", "Z"),
        "git": {
            "commit": sha,
            "dirty": bool(status) if status is not None else None,
        },
        # What served the requests, which is the claim a reader actually
        # needs and the one the SHA above cannot make on its own.
        "images": _images(),
        "machine": {
            "platform": platform.system(),
            "arch": platform.machine(),
            "cpus": os.cpu_count(),
            # A gate run inside CI is a different measurement from one on
            # a workstation, and the difference is why `load-gate` is not
            # in CI at all. Record it rather than leave it to be inferred.
            "ci": os.environ.get("CI") == "true",
        },
        "parameters": parameters or {},
    }


def warnings(st: dict) -> list[str]:
    """Reasons a reader should not cite this run as evidence about the
    commit it names. Returned rather than printed, so the caller decides
    whether they are fatal (they are not — an uncitable run is still a
    passing gate)."""
    out = []
    if st["git"]["commit"] is None:
        out.append("no commit could be read — this run names no code")
    elif st["git"]["dirty"]:
        out.append(f"the tree is dirty: {st['git']['commit'][:12]} is not what ran")
    missing = [s for s, v in st["images"].items() if v is None]
    if missing:
        out.append("no image ID for " + ", ".join(missing) +
                   " — cannot confirm the containers were built from this tree")
    return out


def _selftest() -> int:
    """Asserted, because `warnings` is the function that decides whether a
    published number may be cited, and a silently-permissive version of it
    would let an uncitable run into `docs/results/` looking exactly like a
    citable one."""
    ok = {"git": {"commit": "a" * 40, "dirty": False},
          "images": {s: "sha256:" + "b" * 8 for s in SERVICES}}
    assert warnings(ok) == [], warnings(ok)

    dirty = {"git": {"commit": "a" * 40, "dirty": True}, "images": ok["images"]}
    assert len(warnings(dirty)) == 1 and "dirty" in warnings(dirty)[0]

    # Unknown is not clean. `git status` that could not run at all reports
    # dirty=None, and None is falsy — the bug this case exists to catch is
    # a `warnings` that treats "could not ask" as "asked, and it was fine".
    unknown = {"git": {"commit": None, "dirty": None}, "images": ok["images"]}
    assert len(warnings(unknown)) == 1 and "names no code" in warnings(unknown)[0]

    half = {"git": {"commit": "a" * 40, "dirty": False},
            "images": {**ok["images"], "archive": None}}
    assert len(warnings(half)) == 1 and "archive" in warnings(half)[0]
    assert "collector" not in warnings(half)[0], "named a service that WAS known"

    both = {"git": {"commit": "a" * 40, "dirty": True},
            "images": {s: None for s in SERVICES}}
    assert len(warnings(both)) == 2

    # The stamp must be JSON-serialisable, or --out fails at the end of a
    # twenty-minute gate run rather than at the start.
    st = stamp("selftest", {"n": 1})
    json.loads(json.dumps(st))
    assert set(st["images"]) == set(SERVICES)
    assert st["parameters"] == {"n": 1}

    print("  8 pass")
    return 0


if __name__ == "__main__":
    import sys
    raise SystemExit(_selftest() if "--selftest" in sys.argv else
                     (print(json.dumps(stamp("ad-hoc"), indent=2)) or 0))
