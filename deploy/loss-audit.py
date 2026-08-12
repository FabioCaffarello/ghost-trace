#!/usr/bin/env python3
"""Drive traffic through the topology, break it, and make the books balance.

This is the gate for the accounting phase. Everything before it built a
number; this decides whether the numbers add up, and refuses if they do
not.

WHAT IS BEING RECONCILED. Three separate claims, and they fail in
different ways:

  1. THE ARCHIVE'S OWN BOOKS. Every stream sequence the archive walked
     past was either committed or deliberately refused. The remainder is
     `unaccounted`, and after traffic has drained it must be zero. This
     is read from the archive's DURABLE position rather than from its
     process counters — 3.4 established that counters reset on restart,
     so a reconciliation built on them measures one process's lifetime
     and calls it the archive's contents.

  2. NOTHING LEFT THE STREAM AHEAD OF THE ARCHIVE. `stream_skipped` is
     the stream's first surviving sequence minus the archive's own mark.
     Two earlier attempts to measure this failed because both asked the
     broker where the consumer had got to, and the broker rewrites
     exactly those numbers when it discards records.

  3. THE COLLECTOR ADMITS WHAT IT LOST. With the broker down, telemetry
     cannot be published. The requirement is not that nothing is lost —
     it is that nothing is lost SILENTLY, so `records_dropped_total` must
     move by as much as the traffic that could not be written.

WHY AN OUTAGE IS PART OF THE TARGET. A clean run reconciles trivially,
and would have reconciled just as well before any of this was built.
The interesting reading is the one taken after something broke: records
queued while the archive was away must still be there when it returns,
and traffic refused while the broker was away must be counted rather
than forgotten.

REFUSES, RATHER THAN SKIPS. A topology that is not up is not a pass. The
whole phase rests on the difference between a measured zero and an
absent one, and a gate that quietly does nothing is the same mistake in
the same shape.

Usage:
    make loss-audit          # needs the core profile up
    loss-audit.py --records 40
"""

from __future__ import annotations

import argparse
import json
import subprocess
import sys
import time
import pathlib
import urllib.error
import urllib.request

# Request bodies come from the harness wire module, never hand-rolled
# dicts. The loadgen driver drifted exactly that way — pointer events
# carrying fields the wire does not have, silently dropped (PR-5.0c) —
# and these scripts were the remaining producers outside the shared
# modules. contract/fixtures/ is emitted from wire.py, so a body built
# here is a body the contract harness has already validated and
# replayed against a real server.
sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent.parent / "experiments"))
sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import wire  # noqa: E402
import provenance  # noqa: E402

COLLECTOR = "http://127.0.0.1:8080"
ARCHIVE = "http://127.0.0.1:8081"
ENGINE = "http://127.0.0.1:8082"
SECRET = "sk_demo"
SITE = "pk_demo"

DRAIN_TIMEOUT_S = 90


class Failures(list):
    def check(self, ok: bool, what: str) -> None:
        print(f"  {'ok  ' if ok else 'FAIL'}  {what}")
        if not ok:
            self.append(what)


# ---------------------------------------------------------------
# talking to the topology
# ---------------------------------------------------------------

def post(base, path, body, bearer=None, timeout=30):
    req = urllib.request.Request(base + path, data=json.dumps(body).encode(),
                                 headers={"Content-Type": "application/json"})
    if bearer:
        req.add_header("Authorization", "Bearer " + bearer)
    try:
        with urllib.request.urlopen(req, timeout=timeout) as r:
            raw = r.read()
            return r.status, (json.loads(raw) if raw else {})
    except urllib.error.HTTPError as e:
        return e.code, {}
    except Exception:
        return "unreachable", {}


def metrics(base):
    """Scrape one process. Returns {name: {labels_tuple: value}}.

    A series that is ABSENT stays absent — it is not filled in with a
    zero. That is the whole point of the exposition here: the archive
    publishes no position until it has one, and a reader that defaulted
    the missing series to zero would report a fresh archive as a
    perfectly reconciled one.
    """
    try:
        with urllib.request.urlopen(base + "/metrics", timeout=10) as r:
            body = r.read().decode()
    except Exception as e:
        raise SystemExit(f"could not scrape {base}/metrics: {e}")

    out: dict[str, dict[tuple, float]] = {}
    for line in body.splitlines():
        if not line or line.startswith("#"):
            continue
        try:
            key, value = line.rsplit(" ", 1)
            name, _, labels = key.partition("{")
            labels = tuple(sorted(labels.rstrip("}").split(","))) if labels else ()
            out.setdefault(name, {})[labels] = float(value)
        except ValueError:
            continue
    return out


def one(m, name, default=None):
    """A single-series metric, or default when the series is absent."""
    series = m.get(name)
    if not series:
        return default
    return next(iter(series.values()))


def total(m, name, contains=None):
    """Every series of a metric, summed. Absent stays None."""
    series = m.get(name)
    if series is None:
        return None
    if contains is None:
        return sum(series.values())
    return sum(v for labels, v in series.items()
               if any(contains in label for label in labels))


def compose(*args):
    return subprocess.run(["docker", "compose", "--profile", "core", *args],
                          capture_output=True, text=True)


def healthy(base, timeout=60):
    deadline = time.time() + timeout
    while time.time() < deadline:
        try:
            with urllib.request.urlopen(base + "/healthz", timeout=2) as r:
                if r.status == 200:
                    return True
        except Exception:
            time.sleep(0.5)
    return False


# ---------------------------------------------------------------
# traffic
# ---------------------------------------------------------------

def drive(n):
    """One full round trip per record: session, telemetry, decision, outcome.

    Returns how many of each the collector and engine ACCEPTED, which is
    what the archive is then expected to account for. Requests that were
    refused are not counted as owed — the audit reconciles what the
    system took responsibility for.
    """
    accepted = {"sessions": 0, "telemetry": 0, "decisions": 0, "outcomes": 0}
    for i in range(n):
        st, out = post(COLLECTOR, "/v1/sessions",
                       wire.session_body(SITE))
        token = out.get("session_token", "") if st == 200 else ""
        if not token:
            continue
        accepted["sessions"] += 1

        st, _ = post(COLLECTOR, "/v1/telemetry",
                     wire.telemetry_body(
                         session_token=token, seq=1, sent_at_ms=900,
                         events=[wire.key_event(t=100 + i, phase="down",
                                                key_class="alpha", target="f")]),
                     bearer=token)
        if st == 202:
            accepted["telemetry"] += 1

        st, dec = post(ENGINE, "/v1/decisions",
                       wire.decision_body(token), bearer=SECRET)
        if st != 200:
            continue
        accepted["decisions"] += 1

        st, _ = post(ENGINE, "/v1/outcomes",
                     wire.outcome_body(dec.get("evaluation_id", ""),
                                        "login_success"), bearer=SECRET)
        if st == 202:
            accepted["outcomes"] += 1
    return accepted


def drain(f: Failures, what: str, until_committed=None):
    """Wait until the archive has nothing pending and nothing unexplained.

    `until_committed` is a floor the durable position must reach, and it
    exists because "pending == 0" is not the same statement after a
    RESTART as it is during steady state.

    A restarted archive answers /healthz before it has bound its durable
    consumer, and until it does, it reports no backlog — because it has
    not looked yet. This function used to accept the first such reading
    and declare the archive caught up, so the caller then measured a
    position that had not moved and reported zero commits for a hundred
    records that were committed moments later. The final reconciliation
    still balanced, which is how it stayed hidden: the run said 250
    commits, 250 rows, unaccounted 0, and failed anyway.

    That is this repository's own rule turned against it — an absence
    read as a zero. Where a caller knows what the position must reach,
    it now waits for PROGRESS rather than for a silence.
    """
    deadline = time.time() + DRAIN_TIMEOUT_S
    pending = unaccounted = committed = None
    while time.time() < deadline:
        m = metrics(ARCHIVE)
        pending = one(m, "ghosttrace_archive_stream_pending")
        unaccounted = one(m, "ghosttrace_archive_position_unaccounted")
        committed = one(m, "ghosttrace_archive_position_committed")
        settled = pending == 0 and unaccounted == 0
        # None is not a number and must not compare as one: an absent
        # series means the archive has published nothing yet.
        arrived = (until_committed is None
                   or (committed is not None and committed >= until_committed))
        if settled and arrived:
            f.check(True, f"the archive drained after {what} "
                          f"(pending=0, unaccounted=0"
                          + (f", committed={committed:.0f}" if until_committed else "")
                          + ")")
            return True
        time.sleep(1)
    f.check(False, f"the archive did not drain after {what} within "
                   f"{DRAIN_TIMEOUT_S}s (pending={pending}, "
                   f"unaccounted={unaccounted}, committed={committed}"
                   + (f", needed >= {until_committed:.0f}" if until_committed else "")
                   + ")")
    return False


# ---------------------------------------------------------------
# the scenarios
# ---------------------------------------------------------------

def scenario_clean(f: Failures, n: int) -> None:
    print(f"\n== {n} records through an intact topology ==")
    before = metrics(ARCHIVE)
    accepted = drive(n)
    print(f"  accepted: {accepted}")
    drain(f, "a clean run")

    after = metrics(ARCHIVE)
    # An empty scrape and a scrape without this series are different
    # facts, and only one of them is a zero. A freshly created archive
    # genuinely has committed nothing, so an absent position among other
    # series IS zero; an empty scrape means the archive answered nothing
    # and the baseline is unknown. Defaulting both to 0.0 inflated the
    # delta, and an inflated delta makes `committed >= owed` EASIER to
    # satisfy — the failure direction that hides a real loss.
    if not before:
        f.check(False, "the archive was scrapeable before the run — an empty "
                       "scrape is not a position of zero")
        return
    pos_before = one(before, "ghosttrace_archive_position_committed", 0.0)
    pos_after = one(after, "ghosttrace_archive_position_committed")

    f.check(pos_after is not None,
            "the archive publishes a durable position at all")
    if pos_after is None:
        return

    committed = pos_after - pos_before
    owed = accepted["telemetry"] + accepted["sessions"] + \
        accepted["decisions"] + accepted["outcomes"]
    f.check(committed >= owed,
            f"the archive committed at least what was accepted "
            f"({committed:.0f} commits for {owed} accepted records)")

    f.check(one(after, "ghosttrace_archive_position_unaccounted") == 0,
            "nothing is unaccounted after a clean run")
    f.check(one(after, "ghosttrace_archive_stream_skipped") == 0,
            "nothing left the stream ahead of the archive")

    dropped = total(metrics(COLLECTOR), "ghosttrace_records_dropped_total")
    f.check(dropped == 0,
            f"the collector dropped nothing with everything up (dropped={dropped})")


def scenario_archive_outage(f: Failures, n: int) -> None:
    print("\n== the archive is taken away mid-traffic, then brought back ==")
    before = metrics(ARCHIVE)
    pos_before = one(before, "ghosttrace_archive_position_committed")
    high_before = one(before, "ghosttrace_archive_position_highest_sequence")
    # No default. These used to fall back to 0.0, which would have made a
    # delta out of a reading nobody took — the same "absence is zero" the
    # whole project refuses. If the archive has published no position, the
    # scenario cannot measure one and says so.
    if pos_before is None or high_before is None:
        f.check(False, "the archive published a position before the outage "
                       f"(committed={pos_before}, highest={high_before}) — "
                       "without a baseline there is nothing to compare against")
        return

    compose("stop", "archive")
    try:
        accepted = drive(n)
        print(f"  accepted while the archive was away: {accepted}")
        f.check(accepted["telemetry"] > 0,
                "traffic is still accepted with the archive down (ADR-0006)")
    finally:
        compose("start", "archive")

    f.check(healthy(ARCHIVE), "the archive comes back")
    # Four records per driven session — session, telemetry, decision,
    # outcome — and the floor is deliberately ONE, not that: the claim
    # here is that the queue was drained, not how the archive batches.
    if not drain(f, "the archive returned", until_committed=pos_before + 1):
        return

    after = metrics(ARCHIVE)
    pos_after = one(after, "ghosttrace_archive_position_committed")
    high_after = one(after, "ghosttrace_archive_position_highest_sequence")
    committed = (pos_after or 0) - pos_before
    f.check(pos_after is not None and committed > 0,
            f"the queued records were committed on return "
            f"({pos_before:.0f} -> {pos_after} = {committed:.0f})")
    f.check(high_after is not None and high_after > high_before,
            f"the durable position advanced past where it was before the "
            f"outage ({high_before:.0f} -> {high_after})")
    f.check(one(after, "ghosttrace_archive_position_unaccounted") == 0,
            "an outage the archive recovered from leaves nothing unaccounted")
    f.check(one(after, "ghosttrace_archive_stream_skipped") == 0,
            "nothing aged out while the archive was away")


def scenario_broker_outage(f: Failures, n: int) -> None:
    print("\n== the broker is taken away: loss is allowed, silence is not ==")
    before = metrics(COLLECTOR)
    dropped_before = total(before, "ghosttrace_records_dropped_total", ) or 0.0

    compose("stop", "nats")
    try:
        accepted = drive(n)
        print(f"  accepted while the broker was away: {accepted}")
    finally:
        compose("start", "nats")

    after = metrics(COLLECTOR)
    dropped = (total(after, "ghosttrace_records_dropped_total") or 0.0) - dropped_before

    f.check(dropped > 0,
            f"the collector counted what it could not write ({dropped:.0f} drops); "
            f"a zero here would be loss with no record of it")
    f.check(accepted["telemetry"] > 0,
            "telemetry is still ACCEPTED with the broker down — the promise is "
            "fail-open, and the drop is counted rather than surfaced to the page")

    f.check(healthy(COLLECTOR), "the collector is still serving")
    # The archive must survive the broker's absence too, and resume.
    f.check(healthy(ARCHIVE), "the archive is still serving")


def report(f: Failures) -> None:
    print("\n== the books ==")
    a, c = metrics(ARCHIVE), metrics(COLLECTOR)
    rows = [
        ("archive position: first sequence",
         one(a, "ghosttrace_archive_position_first_sequence")),
        ("archive position: highest sequence",
         one(a, "ghosttrace_archive_position_highest_sequence")),
        ("archive position: commit operations",
         one(a, "ghosttrace_archive_position_committed")),
        ("archive position: refused on purpose",
         one(a, "ghosttrace_archive_position_rejected")),
        ("archive position: UNACCOUNTED",
         one(a, "ghosttrace_archive_position_unaccounted")),
        ("archive: rows actually held", one(a, "ghosttrace_archive_position_rows")),
        ("stream: skipped ahead of the archive",
         one(a, "ghosttrace_archive_stream_skipped")),
        ("stream: pending", one(a, "ghosttrace_archive_stream_pending")),
        ("collector: records written", total(c, "ghosttrace_records_written_total")),
        ("collector: records dropped", total(c, "ghosttrace_records_dropped_total")),
    ]
    for label, v in rows:
        shown = "(absent — not known)" if v is None else f"{v:.0f}"
        print(f"  {label:42} {shown}")

    commits = one(a, "ghosttrace_archive_position_committed")
    held = one(a, "ghosttrace_archive_position_rows")
    if commits is not None and held is not None and commits > held:
        print(f"\n  {commits - held:.0f} commit(s) were deduplicated redeliveries — "
              f"at-least-once delivery working, not a discrepancy.")


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--records", type=int, default=25,
                    help="records to drive per scenario")
    ap.add_argument("--out", default="",
                    help="write the reconciliation, and what produced it, "
                         "as JSON")
    args = ap.parse_args()

    # `healthy` and not a single 3s probe. In CI this runs immediately
    # after `make kill-test`, which restores the services it stopped in a
    # `finally` and returns without waiting for them to reconnect to the
    # broker. A one-shot probe there refuses for a reason that is not a
    # defect — the refusal must mean "the topology is not coming up", not
    # "the topology is not up yet".
    for name, base in (("collector", COLLECTOR), ("archive", ARCHIVE),
                       ("decision-engine", ENGINE)):
        if not healthy(base):
            print(f"the {name} is not answering at {base}.")
            print("bring the topology up first:  make docker-build && make up")
            print("\nrefusing rather than skipping: a gate that quietly does nothing "
                  "is the failure this phase exists to remove.")
            return 1

    f = Failures()
    scenario_clean(f, args.records)
    scenario_archive_outage(f, args.records)
    scenario_broker_outage(f, args.records)

    # The broker came back; let the topology settle and reconcile once
    # more. The final reading is the one that matters: it is taken after
    # everything that was going to break has broken.
    print("\n== after everything, the archive reconciles again ==")
    drain(f, "the broker returned")
    final = metrics(ARCHIVE)
    f.check(one(final, "ghosttrace_archive_position_unaccounted") == 0,
            "the final position accounts for every sequence it walked")

    report(f)

    st = provenance.stamp("loss-audit", {"records": args.records})
    for w in provenance.warnings(st):
        print(f"\n  NOT CITABLE: {w}")
    if args.out:
        a, c = metrics(ARCHIVE), metrics(COLLECTOR)
        with open(args.out, "w") as fh:
            json.dump({
                "provenance": st,
                "archive": {k: one(a, k) for k in sorted(a)},
                "collector": {k: total(c, k) for k in sorted(c)},
                "failures": list(f),
            }, fh, indent=2)
        print(f"\n  wrote {args.out}")

    if f:
        print(f"\n  {len(f)} claim(s) did not hold:")
        for what in f:
            print(f"    - {what}")
        return 1
    print("\n  the books balance.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
