#!/usr/bin/env python3
"""The end-to-end gate: bring the chain up, drive it with a browser, tear it down.

WHAT IS BEING ASSERTED lives in deploy/e2e/flow.mjs, which runs inside
the network and checks seven links: the page, the SDK, the session
handshake, telemetry, the decision, the snapshot the decision was
derived from, and the archive's durable position. This file is the
orchestration around it, and it exists because three of its decisions
are not the runner's to make.

A PROJECT OF ITS OWN, not the default one. Two reasons, and the second
is the one that has already cost a CI run:

  1. An operator with `--profile core` up loses nothing by running this.
     No container name collides, no published port collides, no volume
     is shared — the e2e profile publishes nothing at all.

  2. Gates do not compose. `make loss-audit` first ran in CI directly
     after `make kill-test`, which stops and restarts three services and
     returns without waiting for them to settle; the accounting gate
     then reported four drops in a run it believed to be clean. The fix
     was not to teach it tolerance, it was to give it the quiet system
     its assertions describe. This gate gets that by construction rather
     than by remembering to bring one up.

IT REFUSES, IT DOES NOT SKIP. A missing docker, an image that will not
build, a service that never becomes healthy, a runner that dies — every
one of them exits non-zero. The distinction matters more here than
anywhere: the whole reason for an end-to-end gate is that a broken
deployment still answers every request, and a gate that quietly does
nothing is that same failure wearing a green tick.

THE IMAGE IS ALWAYS REBUILT. `--profile core up --build` does not
rebuild the experiments image, so a stale one runs an old copy of the
driver and fails in ways that look like a topology problem. That cost an
afternoon once, and this gate would inherit it verbatim.

    make e2e
    make e2e E2E_ARGS=--keep-up        # leave it running to look at
"""

from __future__ import annotations

import argparse
import os
import subprocess
import sys
import time

PROJECT = "ghost-trace-e2e"
PROFILE = ["--profile", "e2e"]

# Line-buffered, because this script's own prints and its subprocesses'
# output share a file descriptor. Redirected to a file — which is what CI
# and any `make e2e | tee` do — Python's block buffering held every
# progress line until exit and then dumped them AFTER the compose output
# they were introducing. The run read as though the gate ran backwards.
sys.stdout.reconfigure(line_buffering=True)

# The copy of the topology this gate brings up must not fight one the
# operator already has. compose.yml publishes those three host ports as
# `${GT_*_PORT-<default>}` — bare dash — so an EMPTY value here means an
# ephemeral host port rather than the default. Nothing in this gate reads
# them: every address the runner uses is a service name on the compose
# network, and publishing at all is only so a stuck run can be poked at.
ENV = {**os.environ, "GT_COLLECTOR_PORT": "", "GT_ENGINE_PORT": "", "GT_ARCHIVE_PORT": ""}

# Everything the runner talks to. demo-web-internal is the page a browser
# INSIDE the network can use; the host-facing demo-web is deliberately
# not in this profile, because one page cannot name an origin that is
# right for both.
SERVICES = ["nats", "collector", "decision-engine", "archive", "demo-web-internal"]

HEALTH_TIMEOUT_S = 180
RUN_TIMEOUT_S = 600


def compose(*args: str, **kw) -> subprocess.CompletedProcess:
    return subprocess.run(["docker", "compose", "-p", PROJECT, *PROFILE, *args],
                          env=ENV, **kw)


def up(build: bool) -> None:
    print(f"== bringing the chain up in project {PROJECT} ==")
    # `up --no-start` would be enough to create them, but the images have
    # to exist first and a build failure must surface here rather than as
    # a container that never becomes healthy.
    args = ["up", "-d", *(["--build"] if build else []), *SERVICES]
    if compose(*args).returncode != 0:
        die("the topology would not come up — see the build output above")


def health() -> None:
    print("== waiting for every service to report healthy ==")
    deadline = time.time() + HEALTH_TIMEOUT_S
    for svc in SERVICES:
        state = ""
        while time.time() < deadline:
            r = compose("ps", "--format", "{{.Name}}\t{{.Health}}", svc,
                        capture_output=True, text=True)
            state = (r.stdout.strip().split("\t")[-1] if r.stdout.strip() else "").strip()
            if state == "healthy":
                break
            # A service with no healthcheck reports an empty string. The
            # archive was in exactly that position until PR-5.2, and CI's
            # topology job left it out of its wait list and went green
            # with the archive dead. Treating "no healthcheck" as
            # "healthy" would rebuild that hole here.
            if state in ("unhealthy", "exited", "dead"):
                break
            time.sleep(2)
        print(f"  {svc}: {state or 'no health reported'}")
        if state != "healthy":
            logs()
            die(f"{svc} never became healthy — the gate has nothing to assert against")


def run_driver() -> int:
    print("== driving the chain with a real browser ==")
    # Streams rather than captures: the run takes about a minute and the
    # per-link table is the thing an operator reads while waiting.
    r = compose("run", "--rm", "--build", "e2e", timeout=RUN_TIMEOUT_S)
    return r.returncode


def logs() -> None:
    print("\n== logs ==")
    compose("logs", "--tail=60")


def down() -> None:
    print("== tearing down ==")
    compose("down", "-v", "--remove-orphans", capture_output=True, text=True)


def die(msg: str) -> None:
    print(f"\n  {msg}.")
    print("  refusing rather than skipping: an e2e that quietly does nothing "
          "is the failure it exists to remove.")
    down()
    raise SystemExit(1)


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--keep-up", action="store_true",
                    help="leave the topology running after the run, to look at it")
    ap.add_argument("--no-build", action="store_true",
                    help="reuse the images as they are (faster; wrong after any source change)")
    args = ap.parse_args()

    if subprocess.run(["docker", "version"], capture_output=True).returncode != 0:
        print("docker is not answering — this gate needs it, and cannot stand in for it.")
        return 1

    # A previous run that was interrupted leaves containers behind, and
    # `up` would then reuse them: the gate would assert against a
    # topology whose lifetime it does not know.
    down()

    up(build=not args.no_build)
    health()
    code = run_driver()

    if code != 0:
        logs()
    if not args.keep_up:
        down()
    else:
        print(f"\n  left up. `docker compose -p {PROJECT} {' '.join(PROFILE)} down -v` when done.")

    return code


if __name__ == "__main__":
    sys.exit(main())
