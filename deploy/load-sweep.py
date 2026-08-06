#!/usr/bin/env python3
"""Sweep the offered rate and find where the collector stops keeping up.

Every latency this repository publishes is one session on an idle
system, which the schema itself calls a floor. This turns the floor into
a curve, and the curve's job is to say where the bend is.

WHAT COUNTS AS THE BEND. Three signals, and they do not have to agree:

  - ACHIEVED falls below OFFERED. The system stopped absorbing the rate.
  - RESPONSE p99 departs from SERVICE p99. Requests are queueing; a
    closed-loop driver would have reported the service figure and called
    it healthy.
  - Anything is DROPPED or REFUSED. The accounting phase exists so this
    is visible rather than inferred, and a run that loses records is not
    a faster run.

The first two can move without the third, and that ordering is the
useful part: a system that queues before it drops has a warning period,
and one that drops first does not.

REFUSES RATHER THAN SKIPS. A topology that is not up is not a pass, and
a step whose driver fell behind its own schedule is discarded rather
than plotted — it measured the driver.

Usage:
    make load-sweep
    load-sweep.py --rates 50,100,200,400,800 --duration 10s
"""

from __future__ import annotations

import argparse
import json
import os
import subprocess
import sys
import time
import urllib.error
import urllib.request

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
COLLECTOR = "http://127.0.0.1:8080"
ARCHIVE = "http://127.0.0.1:8081"


def metrics(base):
    with urllib.request.urlopen(base + "/metrics", timeout=10) as r:
        body = r.read().decode()
    out = {}
    for line in body.splitlines():
        if not line or line.startswith("#"):
            continue
        try:
            key, value = line.rsplit(" ", 1)
            name = key.partition("{")[0]
            out[name] = out.get(name, 0.0) + float(value)
        except ValueError:
            continue
    return out


def run_step(rate: float, duration: str, events: int, workers: int) -> dict | None:
    """One rate. Returns the report, or None when the driver refused it."""
    proc = subprocess.run(
        ["go", "run", "./cmd/loadgen",
         f"-rate={rate}", f"-duration={duration}", f"-events={events}",
         f"-workers={workers}", "-scenario=session",
         f"-collector={COLLECTOR}"],
        cwd=os.path.join(ROOT, "tools", "loadgen"),
        capture_output=True, text=True,
    )
    try:
        rep = json.loads(proc.stdout)
    except json.JSONDecodeError:
        print(f"  {rate:>6.0f}/s  driver produced no report:\n{proc.stderr[-400:]}")
        return None
    if proc.returncode != 0:
        # The driver refuses a run it could not sustain. Plotting it
        # anyway would put the driver's own worker bound on the curve
        # and label it the system's capacity.
        rep["refused"] = True
    return rep


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--rates", default="50,100,200,400,800,1600",
                    help="offered rates, requests per second")
    ap.add_argument("--duration", default="10s")
    ap.add_argument("--events", type=int, default=8)
    ap.add_argument("--workers", type=int, default=2048)
    ap.add_argument("--out", default="")
    args = ap.parse_args()

    try:
        with urllib.request.urlopen(COLLECTOR + "/healthz", timeout=3) as r:
            if r.status != 200:
                raise RuntimeError
    except Exception:
        print(f"the collector is not answering at {COLLECTOR}.")
        print("bring the topology up first:  make docker-build && make up")
        print("\nrefusing rather than skipping: a gate that quietly does nothing "
              "is the failure this project keeps removing.")
        return 1

    rates = [float(r) for r in args.rates.split(",")]
    print(f"sweeping {rates} req/s, {args.duration} each, {args.events} events/batch\n")
    print(f"  {'offered':>8} {'achieved':>9} {'svc p50':>9} {'svc p99':>9} "
          f"{'rsp p50':>9} {'rsp p99':>9} {'deficit':>9} {'failed':>7} {'dropped':>8}")
    print("  " + "-" * 88)

    steps = []
    for rate in rates:
        before = metrics(COLLECTOR).get("ghosttrace_records_dropped_total", 0.0)
        rep = run_step(rate, args.duration, args.events, args.workers)
        if rep is None:
            continue
        after = metrics(COLLECTOR).get("ghosttrace_records_dropped_total", 0.0)
        rep["dropped"] = after - before
        steps.append(rep)

        flag = "  REFUSED (driver fell behind)" if rep.get("refused") else ""
        print(f"  {rep['intended_rps']:>8.0f} {rep['achieved_rps']:>9.0f} "
              f"{rep['service_p50_ms']:>9.2f} {rep['service_p99_ms']:>9.2f} "
              f"{rep['response_p50_ms']:>9.2f} {rep['response_p99_ms']:>9.2f} "
              f"{rep['deficit_p99_ms']:>9.2f} {rep['failed']:>7} "
              f"{rep['dropped']:>8.0f}{flag}")
        time.sleep(2)

    usable = [s for s in steps if not s.get("refused")]
    if not usable:
        print("\n  every step was refused; the driver never got a clean reading")
        return 1

    print("\n== where it bends ==")
    base = usable[0]
    bends = []
    for s in usable:
        if s["achieved_rps"] < 0.95 * s["intended_rps"]:
            bends.append((s["intended_rps"], "achieved fell below offered"))
            break
    for s in usable:
        if base["response_p99_ms"] > 0 and s["response_p99_ms"] > 4 * base["response_p99_ms"]:
            bends.append((s["intended_rps"], "response p99 quadrupled from the lowest rate"))
            break
    for s in usable:
        if s["dropped"] > 0 or s["failed"] > 0:
            bends.append((s["intended_rps"],
                          f"records were lost ({s['dropped']:.0f} dropped, "
                          f"{s['failed']} failed)"))
            break

    if not bends:
        top = usable[-1]["intended_rps"]
        print(f"  no bend up to {top:.0f}/s — the curve is still flat where the "
              f"sweep stopped, so this run establishes a LOWER BOUND and not a "
              f"capacity.")
    else:
        for rate, why in bends:
            print(f"  {rate:>6.0f}/s  {why}")

    print("\n== the accounting, after the sweep ==")
    a = metrics(ARCHIVE)
    for name in ("ghosttrace_archive_position_committed",
                 "ghosttrace_archive_position_unaccounted",
                 "ghosttrace_archive_stream_skipped",
                 "ghosttrace_archive_stream_pending"):
        v = a.get(name)
        print(f"  {name.replace('ghosttrace_archive_', ''):38} "
              f"{'(absent)' if v is None else f'{v:.0f}'}")

    if args.out:
        with open(args.out, "w") as fh:
            json.dump({"steps": steps, "bends": bends}, fh, indent=2)
        print(f"\n  wrote {args.out}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
