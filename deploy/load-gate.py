#!/usr/bin/env python3
"""The Phase 4 gate: the accounting still balances under sustained load.

    Under sustained concurrency, the accounting still balances —
    unaccounted at zero, every drop counted, every refusal explained —
    and every latency number the repository publishes is either
    reproduced at load or replaced by a curve that says where it bends.

WHAT THIS TARGET CHECKS, AND WHAT IT DOES NOT. The first half is a
measurement and is checked here. The second half is partly a
documentation obligation, and pretending a `make` target can enforce it
would be the ceremony this repository keeps deleting. So the split is
stated rather than blurred:

  CHECKED HERE
    - `unaccounted` returns to zero after load drains.
    - `stream_skipped` stays zero: nothing left the stream ahead of the
      archive while it was behind.
    - Records the collector could not write are COUNTED. Loss is
      permitted; silence is not.
    - The decision p99 measured under load is inside the 80ms budget.
      The budget is the published claim; the 1.393ms idle figure never
      was one, and the schema calls it a floor.
    - The driver kept its own schedule. A run where it did not measured
      the driver, and is refused rather than reported.

  NOT CHECKED HERE
    - That a curve exists for each published latency. PR-4.2 and PR-4.5
      produced them and they live in docs/results/; a target that
      asserted the presence of a markdown file would be checking that
      somebody wrote a file, not that the number is honest.

WHY LOSS-AUDIT IS NOT ENOUGH. `make loss-audit` reconciles under an
induced OUTAGE — the archive taken away, the broker taken away. This
reconciles under sustained LOAD, which fails differently: nothing is
down, everything is answering, and the archive simply cannot keep up.
That is the condition the whole phase was opened to measure, and it is
the one an outage test never reaches.

REFUSES RATHER THAN SKIPS.

Usage:
    make load-gate
    load-gate.py --sessions-rate 2000 --decisions-rate 4000 --duration 20s
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

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import provenance  # noqa: E402

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
COLLECTOR = "http://127.0.0.1:8080"
ARCHIVE = "http://127.0.0.1:8081"
ENGINE = "http://127.0.0.1:8082"

# The published claim. Not the idle floor — the schema calls that a
# floor, and a gate that required a floor to be reproduced under load
# would fail for the wrong reason and be deleted for it.
DECISION_BUDGET_MS = 80

DRAIN_TIMEOUT_S = 240


class Failures(list):
    def check(self, ok: bool, what: str) -> None:
        print(f"  {'ok  ' if ok else 'FAIL'}  {what}")
        if not ok:
            self.append(what)


def metrics(base):
    try:
        with urllib.request.urlopen(base + "/metrics", timeout=10) as r:
            body = r.read().decode()
    except Exception as e:
        raise SystemExit(f"could not scrape {base}/metrics: {e}")
    out = {}
    for line in body.splitlines():
        if not line or line.startswith("#"):
            continue
        try:
            key, value = line.rsplit(" ", 1)
            out[key.partition("{")[0]] = out.get(key.partition("{")[0], 0.0) + float(value)
        except ValueError:
            continue
    return out


def one(m, name):
    """A metric, or None when the series is absent.

    Absent stays absent. A reader that defaulted a missing series to
    zero would report an archive that has never consumed anything as a
    perfectly reconciled one, which is the failure Phase 3 removed.
    """
    return m.get(name)


def loadgen(args: list[str]) -> tuple[dict | None, bool]:
    """Run the driver. Returns the report and whether it refused."""
    proc = subprocess.run(
        ["go", "run", "./cmd/loadgen", *args],
        cwd=os.path.join(ROOT, "tools", "loadgen"),
        capture_output=True, text=True,
    )
    try:
        return json.loads(proc.stdout), proc.returncode != 0
    except json.JSONDecodeError:
        print(proc.stderr[-600:])
        return None, True


def drain(f: Failures, what: str) -> bool:
    deadline = time.time() + DRAIN_TIMEOUT_S
    pending = unaccounted = None
    while time.time() < deadline:
        m = metrics(ARCHIVE)
        pending = one(m, "ghosttrace_archive_stream_pending")
        unaccounted = one(m, "ghosttrace_archive_position_unaccounted")
        if pending == 0 and unaccounted == 0:
            f.check(True, f"the archive drained after {what}")
            return True
        time.sleep(2)
    f.check(False, f"the archive did not drain after {what} within {DRAIN_TIMEOUT_S}s "
                   f"(pending={pending}, unaccounted={unaccounted})")
    return False


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--sessions-rate", type=float, default=2000)
    ap.add_argument("--decisions-rate", type=float, default=4000)
    ap.add_argument("--duration", default="20s")
    ap.add_argument("--workers", type=int, default=4096)
    ap.add_argument("--out", default="")
    args = ap.parse_args()

    for name, base in (("collector", COLLECTOR), ("archive", ARCHIVE),
                       ("decision-engine", ENGINE)):
        try:
            with urllib.request.urlopen(base + "/healthz", timeout=3) as r:
                if r.status != 200:
                    raise RuntimeError
        except Exception:
            print(f"the {name} is not answering at {base}.")
            print("bring the topology up first:  make docker-build && make up")
            print("\nrefusing rather than skipping: a gate that quietly does nothing "
                  "is the failure this project keeps removing.")
            return 1

    f = Failures()
    before = metrics(COLLECTOR)
    # Absent stays absent — the rule one()'s own docstring states, and
    # this file violated it here: `or 0.0` read a missing drop counter
    # as a perfect zero, so a collector whose lossmeter never came up
    # would have passed the "nothing was dropped" claim vacuously.
    drops_before = one(before, "ghosttrace_records_dropped_total")
    f.check(drops_before is not None,
            "the collector publishes its drop counter before the run "
            "(the series exists at all)")

    # ---- ingest under sustained load ----
    print(f"\n== {args.sessions_rate:.0f} sessions/s for {args.duration} ==")
    ingest, refused = loadgen([
        f"-rate={args.sessions_rate}", f"-duration={args.duration}",
        f"-workers={args.workers}", "-scenario=session", f"-collector={COLLECTOR}"])
    if ingest is None:
        f.check(False, "the ingest run produced no report")
        return report_and_exit(f, None, None, args)
    f.check(not refused,
            f"the driver kept its schedule during ingest "
            f"(deficit p99 {ingest['deficit_p99_ms']:.2f}ms)")
    f.check(ingest["failed"] == 0,
            f"every accepted request succeeded ({ingest['failed']} failed)")
    f.check(ingest["achieved_rps"] >= 0.95 * ingest["intended_rps"],
            f"the offered rate was absorbed "
            f"({ingest['achieved_rps']:.0f}/s of {ingest['intended_rps']:.0f}/s)")

    drain(f, "sustained ingest")

    after = metrics(COLLECTOR)
    a = metrics(ARCHIVE)
    drops_after = one(after, "ghosttrace_records_dropped_total")
    f.check(drops_after is not None,
            "the collector still publishes its drop counter after the run")
    drops = (None if drops_before is None or drops_after is None
             else drops_after - drops_before)
    written = one(after, "ghosttrace_records_written_total")

    f.check(written is not None,
            "the collector publishes what it wrote (the series exists at all)")
    f.check(drops == 0,
            f"nothing was dropped under load "
            f"({'the counter is absent' if drops is None else f'{drops:.0f}'}); "
            f"a drop here would be permitted but would have to be counted, and it is")
    f.check(one(a, "ghosttrace_archive_position_unaccounted") == 0,
            "every sequence the archive walked is committed or refused")
    f.check(one(a, "ghosttrace_archive_stream_skipped") == 0,
            "nothing left the stream ahead of the archive while it was behind")

    committed = one(a, "ghosttrace_archive_position_committed")
    rows = one(a, "ghosttrace_archive_position_rows")
    f.check(committed is not None and rows is not None and committed >= rows,
            f"commit operations ({committed}) cover the rows held ({rows})")

    # ---- decisions under sustained load ----
    print(f"\n== {args.decisions_rate:.0f} decisions/s for {args.duration} ==")
    dec, refused = loadgen([
        f"-rate={args.decisions_rate}", f"-duration={args.duration}",
        f"-workers={args.workers}", "-scenario=decision",
        f"-collector={COLLECTOR}", f"-engine={ENGINE}", "-warm-sessions=400"])
    if dec is None:
        f.check(False, "the decision run produced no report")
        return report_and_exit(f, ingest, None, args)

    f.check(not refused,
            f"the driver kept its schedule during decisions "
            f"(deficit p99 {dec['deficit_p99_ms']:.2f}ms)")
    f.check(dec["failed"] == 0, f"every decision succeeded ({dec['failed']} failed)")
    f.check(dec["response_p99_ms"] < DECISION_BUDGET_MS,
            f"the decision p99 under load is inside the published budget "
            f"({dec['response_p99_ms']:.2f}ms of {DECISION_BUDGET_MS}ms)")

    drain(f, "sustained decisions")
    a = metrics(ARCHIVE)
    f.check(one(a, "ghosttrace_archive_position_unaccounted") == 0,
            "the accounting still balances after the decision load")

    return report_and_exit(f, ingest, dec, args)


def report_and_exit(f: Failures, ingest, dec, args) -> int:
    print("\n== under load ==")
    a, c = metrics(ARCHIVE), metrics(COLLECTOR)
    for label, value in (
        ("collector: records written", one(c, "ghosttrace_records_written_total")),
        ("collector: records dropped", one(c, "ghosttrace_records_dropped_total")),
        ("archive: commit operations", one(a, "ghosttrace_archive_position_committed")),
        ("archive: rows held", one(a, "ghosttrace_archive_position_rows")),
        ("archive: UNACCOUNTED", one(a, "ghosttrace_archive_position_unaccounted")),
        ("archive: skipped ahead of it", one(a, "ghosttrace_archive_stream_skipped")),
    ):
        print(f"  {label:34} {'(absent — not known)' if value is None else f'{value:.0f}'}")
    if ingest:
        print(f"  {'ingest p99 (response)':34} {ingest['response_p99_ms']:.2f}ms")
    if dec:
        print(f"  {'decision p99 (response)':34} {dec['response_p99_ms']:.2f}ms "
              f"of {DECISION_BUDGET_MS}ms budget")

    st = provenance.stamp("load-gate", {
        "sessions_rate": args.sessions_rate,
        "decisions_rate": args.decisions_rate,
        "duration": args.duration,
        "workers": args.workers,
    })
    # Printed whether or not --out was given: somebody reading the
    # terminal is as capable of misciting a number as somebody reading
    # the file.
    for w in provenance.warnings(st):
        print(f"\n  NOT CITABLE: {w}")

    if args.out:
        with open(args.out, "w") as fh:
            json.dump({"provenance": st, "ingest": ingest, "decision": dec,
                       "failures": list(f)}, fh, indent=2)
        print(f"\n  wrote {args.out}")

    if f:
        print(f"\n  {len(f)} claim(s) did not hold:")
        for what in f:
            print(f"    - {what}")
        return 1
    print("\n  the accounting balances under load, and the decision path is inside "
          "its budget.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
