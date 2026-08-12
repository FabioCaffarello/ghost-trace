#!/usr/bin/env python3
"""Check a run of the six numbers against the last published manifest.

WHY THIS EXISTS. CONTRIBUTING has always said detection rates must stay
inside the Wilson intervals of the last committed manifest, p99 inside
the budget, and cold start non-blocking. Nothing checked it. The
comparison was done by eye, per pull request, by whoever remembered —
and a hand comparison picked the WRONG BASELINE twice in one day,
because `sorted(glob(...))[-1]` orders manifests by filename and
filenames start with a content hash. The older, pre-seeding manifest
sorted last. For a project whose central claim is that six numbers
reproduce, "we looked at it" is not how that claim should be checked.

So the baseline here is chosen by provenance.generated_at, and every
rule CONTRIBUTING states is a rule this file enforces.

WHAT IT REFUSES TO CALL A PASS
  - a tier in the baseline that is missing from this run. A tier that
    did not run is not a tier that found nothing; letting one vanish
    quietly makes every remaining number look better than it is.
  - a false-positive rate that HAD a value and now has none. Losing
    human data is not the same as never having had it.
  - a detection rate outside the baseline's interval, in EITHER
    direction. A detector that suddenly catches more is as much a
    change to explain as one that catches less.
  - a baseline with no adversary seed. Before seeding, tier 6 measured
    70%, 90%, 80% and 100% from unchanged code; an adversary nobody can
    replay cannot anchor a comparison, and printing "(none recorded)"
    next to the verdict was disclosure, not refusal.

Both of the first two are the same rule the repository states twice:
absence is never zero.

Usage:
    python3 experiments/numbers_check.py [--run FILE] [--baseline FILE]
"""

from __future__ import annotations

import argparse
import glob
import json
import os
import sys

# Wilson bounds are floating point; a rate of 1.0 against a ci_high of
# 0.9999999999999999 is the same number, not a regression.
EPS = 1e-9

# CONTRIBUTING's wording for the memory benchmark. Applied per
# (architecture, sessions, duration) cell rather than in aggregate,
# because an aggregate hides a regression in one cell behind an
# improvement in another.
MEMORY_REGRESSION = 0.10

# Numbers 3 and 4 are compared against the baseline with a proportional,
# TWO-SIDED band. The budget alone is no bar at all: the Phase-4 run
# moved p99 from 5.525ms to 0.739ms — 7.5x, twelve times inside the
# budget — and was published as "nothing moved" because nothing compared
# it. Two-sided because an unexplained improvement is as much a change
# to explain as a regression: a detector that got faster may be a
# detector doing less. The committed manifests put run-to-run noise at
# 5-9% for session latency and 0% for TTCD, so half is generous to
# noise and still five times smaller than the move that sailed through.
LATENCY_BAND = 0.50
TTCD_BAND = 0.50

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
DEFAULT_RUN = os.path.join(ROOT, "experiments", "results", "numbers.json")
MANIFEST_GLOB = os.path.join(ROOT, "docs", "results", "numbers-*.json")


class Report:
    """Findings, kept apart from how they are printed."""

    def __init__(self) -> None:
        self.failures: list[str] = []
        self.notes: list[str] = []

    def fail(self, msg: str) -> None:
        self.failures.append(msg)

    def note(self, msg: str) -> None:
        self.notes.append(msg)


def load(path: str) -> dict:
    with open(path, encoding="utf-8") as fh:
        return json.load(fh)


def pick_newest(paths, stamp) -> str | None:
    """The most recently GENERATED manifest, not the last filename.

    Split out from disk access so the selftest can prove the ordering
    with filenames that sort the wrong way — which is the bug this whole
    module exists because of.
    """
    if not paths:
        return None
    return max(paths, key=stamp)


def newest_manifest(want: str, want_load: str = "idle") -> str | None:
    """The newest published manifest of the same TOPOLOGY and the same
    LOAD CONDITION.

    Not simply the newest. Once a composed baseline exists, "newest"
    alone would hand a monolith run a composed one to be judged against,
    and the topology check would then reject a perfectly good run for a
    reason that is the picker's fault rather than the run's.

    Load condition joins the filter for exactly the same reason. The two
    together are what "the same kind of measurement" means here, and a
    picker that ignored either would manufacture the failure the checks
    downstream exist to report.
    """
    same = [p for p in glob.glob(MANIFEST_GLOB)
            if topology(load(p)) == want and load_condition(load(p)) == want_load]
    return pick_newest(same, lambda p: load(p)["provenance"]["generated_at"])


def check_detection(base: dict, run: dict, rep: Report) -> None:
    b, n = base["detection"], run["detection"]

    for tier in sorted(set(b) | set(n)):
        if tier not in n:
            rep.fail(
                f"{tier}: in the baseline ({b[tier]['detected']}/{b[tier]['n']}) "
                f"and ABSENT from this run. A tier that did not run is not a "
                f"tier that found nothing."
            )
            continue
        if tier not in b:
            rep.note(f"{tier}: new since the baseline "
                     f"({n[tier]['detected']}/{n[tier]['n']}) — nothing to compare against")
            continue

        rate = n[tier]["rate"]
        lo, hi = b[tier]["ci_low"], b[tier]["ci_high"]
        if not (lo - EPS <= rate <= hi + EPS):
            rep.fail(
                f"{tier}: {n[tier]['detected']}/{n[tier]['n']} = {rate:.3f}, outside "
                f"the baseline interval [{lo:.3f}, {hi:.3f}] "
                f"(baseline {b[tier]['detected']}/{b[tier]['n']})"
            )


def check_absent_tiers(base: dict, run: dict, rep: Report) -> None:
    """absent_tiers is the run's own record of what it could not do."""
    was, now = set(base.get("absent_tiers") or []), set(run.get("absent_tiers") or [])
    for tier in sorted(now - was):
        rep.fail(
            f"{tier}: this run recorded it as ABSENT and the baseline ran it. "
            f"The remaining numbers describe less than the published ones do."
        )
    for tier in sorted(was - now):
        rep.note(f"{tier}: absent in the baseline, present here")


def topology(d: dict) -> str:
    """Manifests from before the split recorded no topology; there was
    only one thing they could have been."""
    return (d.get("provenance", {}).get("run", {}) or {}).get("topology", "monolith")


def load_condition(d: dict) -> str:
    """Manifests from before Phase 4 recorded no load condition. Every
    one of them was taken against an otherwise idle system, which is
    what the harness has always done."""
    return (d.get("provenance", {}).get("run", {}) or {}).get("load", "idle")


def unseeded(base: dict) -> bool:
    """A baseline whose adversary cannot be replayed.

    The pre-seeding manifest carries `seed: null` — kept as history,
    never usable as an anchor. Pure so the selftest can hold it."""
    return not (base.get("provenance", {}).get("run", {}) or {}).get("seed")


def check_load(base: dict, run: dict, rep: Report) -> None:
    b, n = load_condition(base), load_condition(run)
    if b != n:
        rep.fail(
            f"this run is {n!r} and the baseline is {b!r}. A latency measured "
            f"against an idle system and one measured under concurrency are "
            f"different measurements wearing the same name — Phase 4 got 4.7x "
            f"the published p99 from the same code — so comparing them would "
            f"report a load change as a regression, or hide one. Publish a "
            f"baseline for {n!r} (`make numbers-manifest`) and compare against "
            f"that."
        )


def check_topology(base: dict, run: dict, rep: Report) -> None:
    b, n = topology(base), topology(run)
    if b != n:
        rep.fail(
            f"this run is {n} and the baseline is {b}. A decision answered in "
            f"process and one answered across a network hop and a KV read are "
            f"different measurements wearing the same name; comparing them "
            f"would report a topology change as a regression, or hide one. "
            f"Publish a baseline for {n} (`make numbers-manifest`) and compare "
            f"against that."
        )


def check_session(base: dict, run: dict, rep: Report) -> None:
    b, n = base.get("session"), run.get("session")

    # The schema allows session: null — a run where the measurement did
    # not happen. That is an ABSENCE, the same rule as a vanished tier,
    # and before this check existed it was a TypeError on one path and a
    # silent pass on the other.
    if n is None:
        if b is not None:
            rep.fail(
                "session: measured in the baseline and null in this run. "
                "Numbers 3, 4 and 5 did not hold; they vanished — a "
                "measurement that did not run is not a measurement that "
                "found nothing."
            )
        return
    if b is None:
        rep.note("session: measured here and null in the baseline — "
                 "nothing to compare against")

    lat = n["latency"]
    budget = lat["budget_ms"]
    if lat["p99"] > budget:
        rep.fail(f"p99 decision latency {lat['p99']}ms is over the {budget}ms budget")

    # NUMBER 3 against the BASELINE, not only the budget. This is the
    # check whose absence let a 7.5x move ship as "nothing moved".
    if b is not None:
        for q in ("p50", "p95", "p99"):
            before, after = b["latency"].get(q), lat.get(q)
            if not before or after is None:
                continue
            drift = abs(after - before) / before
            if drift > LATENCY_BAND:
                rep.fail(
                    f"latency {q}: {after}ms against {before}ms in the baseline — "
                    f"{drift:.0%} apart, and more than {LATENCY_BAND:.0%} is a "
                    f"different number, not noise. Inside the budget is not the "
                    f"same as unmoved."
                )

    cold = n["cold_start"]
    if not cold.get("never_blocks"):
        rep.fail(
            f"cold start blocks: decisions {cold.get('decisions')}. A first visit "
            f"with no history must never be blocked."
        )

    # NUMBER 4. The medians are the published claim; before this check
    # they produced at most a note, so 3.7s could become 300s and the
    # verdict would still read "reproduce".
    for gate in ("challenge", "block"):
        t = n["ttcd"].get(gate) or {}
        bt = (b["ttcd"].get(gate) or {}) if b is not None else {}
        if bt and not t:
            rep.fail(f"time-to-confident-decision ({gate}): in the baseline "
                     f"({bt.get('seconds')}s) and absent from this run")
            continue
        if bt.get("seconds") and t.get("seconds") is not None:
            drift = abs(t["seconds"] - bt["seconds"]) / bt["seconds"]
            if drift > TTCD_BAND:
                rep.fail(
                    f"time-to-confident-decision ({gate}): {t['seconds']}s against "
                    f"{bt['seconds']}s in the baseline — {drift:.0%} apart, and "
                    f"more than {TTCD_BAND:.0%} is a different number, not noise"
                )
        if t.get("of") and t.get("reached", 0) < t["of"]:
            was = (f" (baseline {bt.get('reached')}/{bt.get('of')})"
                   if bt else "")
            rep.note(f"time-to-confident-decision ({gate}): reached in "
                     f"{t['reached']}/{t['of']} sessions{was}")


def check_false_positive_rate(base: dict, run: dict, rep: Report) -> None:
    b, n = base.get("false_positive_rate"), run.get("false_positive_rate")
    if b is not None and n is None:
        rep.fail(
            "the false-positive rate had a value in the baseline and has none here. "
            "Human data was lost, which is not the same as never having had it — "
            "and number 1 means nothing without this one."
        )
    elif b is None and n is not None:
        rep.note(f"false-positive rate now has a value ({n}); the baseline had none")


def cell(row: dict) -> tuple:
    return (row["architecture"], row["sessions"], row["duration_s"])


def check_architecture(base: dict, run: dict, rep: Report) -> None:
    # The decide_*_ms columns in these rows are deliberately NOT banded.
    # Between two committed manifests of unchanged code the same cell
    # measured 0.0005ms and 0.01775ms — 35x apart — because a
    # sub-microsecond micro-benchmark on a shared machine measures the
    # machine. Number 6 is bytes_per_session; the timing columns are
    # the bench's own context, and a guard that cries wolf on every run
    # teaches people to override it.
    b = {cell(r): r for r in base.get("architecture") or []}
    n = {cell(r): r for r in run.get("architecture") or []}

    if b and not n:
        rep.fail(
            f"architecture: measured in the baseline ({len(b)} cells) and "
            f"absent from this run — the schema allows null, and before this "
            f"check that null read as a pass. Number 6 did not hold; it "
            f"vanished."
        )
        return
    for key in sorted(set(b) - set(n)):
        a, s, d = key
        rep.fail(f"architecture: {a} at {s} sessions x {d}s is in the baseline "
                 f"and absent from this run")

    for key in sorted(n):
        row, prev = n[key], b.get(key)
        if not prev or not prev.get("bytes_per_session"):
            continue
        before, after = prev["bytes_per_session"], row["bytes_per_session"]
        growth = (after - before) / before
        if growth > MEMORY_REGRESSION:
            a, s, d = key
            rep.fail(
                f"memory: {a} at {s} sessions x {d}s uses {after} bytes/session, "
                f"up {growth:.0%} from {before} (>{MEMORY_REGRESSION:.0%} is a regression)"
            )


def no_baseline(want: str = "") -> int:
    """No manifest to check against.

    This FAILS. Nothing was compared, and a check that compared nothing
    reporting success is the vacuous green this repository keeps
    finding — the same reason parity and shadow refuse rather than skip.
    docs/results/ is committed content; an empty one is a broken
    checkout, not a state a run should pass through quietly.
    """
    which = f" for the {want} topology" if want else ""
    print(f"numbers-check: NOTHING WAS CHECKED — docs/results/ holds no manifest{which},",
          file=sys.stderr)
    print("               so the run was compared against nothing.", file=sys.stderr)
    print("fix: publish the first baseline with `make numbers-manifest`", file=sys.stderr)
    return 1


def verdict(base: dict, run: dict) -> Report:
    """Every rule CONTRIBUTING states, applied to two runs."""
    rep = Report()
    check_topology(base, run, rep)
    check_load(base, run, rep)
    check_detection(base, run, rep)
    check_absent_tiers(base, run, rep)
    check_session(base, run, rep)
    check_false_positive_rate(base, run, rep)
    check_architecture(base, run, rep)
    return rep


# ---------------------------------------------------------------
# selftest
# ---------------------------------------------------------------
#
# A checker nobody has seen reject anything is indistinguishable from a
# checker that accepts everything, and this one guards THE project
# invariant. So every rule above has a case here that it must reject,
# and the cases run in `make experiments-check` — inside `make ci`,
# where they need no browsers and no seven minutes.


def _tier(detected: int, n: int, lo: float, hi: float) -> dict:
    return {"n": n, "detected": detected, "rate": detected / n,
            "ci_low": lo, "ci_high": hi}


def _baseline() -> dict:
    return {
        "schema_version": "1.0.0",
        "detection": {"tier1": _tier(8, 10, 0.490, 0.943)},
        "absent_tiers": [],
        "false_positive_rate": None,
        "session": {
            "latency": {"p50": 2.0, "p95": 4.0, "p99": 5.0, "budget_ms": 80},
            "cold_start": {"never_blocks": True, "decisions": ["allow"]},
            "ttcd": {"challenge": {"reached": 30, "of": 30, "seconds": 3.7},
                     "block": {"reached": 30, "of": 30, "seconds": 5.7}},
        },
        "architecture": [{"architecture": "maintained", "sessions": 100,
                          "duration_s": 10, "bytes_per_session": 700}],
        "provenance": {"generated_at": "2026-01-01T00:00:00Z",
                       "run": {"seed": "s", "topology": "monolith"},
                       "git": {"commit": "0" * 40}},
    }


def _mutate(**over) -> dict:
    """A run identical to the baseline except where told otherwise."""
    run = json.loads(json.dumps(_baseline()))
    for path, value in over.items():
        node = run
        keys = path.split(".")
        for k in keys[:-1]:
            node = node[k]
        node[keys[-1]] = value
    return run


def selftest() -> int:
    cases: list[tuple[str, dict, bool]] = []

    cases.append(("an identical run", _mutate(), True))

    composed = _mutate()
    composed["provenance"]["run"]["topology"] = "composed"
    cases.append(("a composed run against a monolith baseline", composed, False))

    # Both directions. A detector that suddenly catches MORE is as much
    # a change to explain as one that catches less — and 10/10 against a
    # baseline of 8/10 is exactly the case a hand comparison waved
    # through.
    cases.append(("a rate below the interval",
                  _mutate(detection={"tier1": _tier(3, 10, 0.490, 0.943)}), False))
    cases.append(("a rate above the interval",
                  _mutate(detection={"tier1": _tier(10, 10, 0.490, 0.943)}), False))
    cases.append(("a rate inside the interval",
                  _mutate(detection={"tier1": _tier(9, 10, 0.490, 0.943)}), True))

    # Absence is never zero, stated twice in the repository and enforced
    # in two places here.
    cases.append(("a tier that vanished", _mutate(detection={}), False))
    cases.append(("a tier the run itself recorded as absent",
                  _mutate(absent_tiers=["tier1"]), False))
    cases.append(("a tier that is new", _mutate(detection={
        "tier1": _tier(8, 10, 0.490, 0.943), "tier9": _tier(5, 5, 0.5, 1.0)}), True))

    cases.append(("p99 over budget", _mutate(**{"session.latency":
                  {"p50": 2.0, "p95": 4.0, "p99": 81.0, "budget_ms": 80}}), False))
    cases.append(("a cold start that blocks", _mutate(**{"session.cold_start":
                  {"never_blocks": False, "decisions": ["block"]}}), False))

    # NUMBER 3 against the baseline. The exact mutation that shipped:
    # the Phase-4 run moved p99 from 5.525ms to 0.739ms — 7.5x, well
    # inside the budget — and the verdict read "the six numbers
    # reproduce" because only the budget was consulted.
    cases.append(("a p99 that collapsed inside the budget", _mutate(**{
        "session.latency": {"p50": 2.0, "p95": 4.0, "p99": 0.66, "budget_ms": 80}}),
        False))
    cases.append(("a p99 drifted within the band", _mutate(**{
        "session.latency": {"p50": 2.0, "p95": 4.0, "p99": 5.4, "budget_ms": 80}}),
        True))

    # NUMBER 4. 3.7s -> 300s was accepted before the medians were
    # compared; a note is not a verdict.
    cases.append(("a TTCD that ballooned", _mutate(**{
        "session.ttcd": {"challenge": {"reached": 30, "of": 30, "seconds": 300.0},
                         "block": {"reached": 30, "of": 30, "seconds": 5.7}}}),
        False))

    # The schema allows both of these nulls; before these checks one
    # crashed and the other passed silently.
    cases.append(("a session that is null against a measured baseline",
                  _mutate(session=None), False))
    cases.append(("an architecture that is null against a measured baseline",
                  _mutate(architecture=None), False))
    cases.append(("an architecture cell that vanished", _mutate(architecture=[
        {"architecture": "maintained", "sessions": 999, "duration_s": 10,
         "bytes_per_session": 700}]), False))

    cases.append(("memory up 20%", _mutate(architecture=[
        {"architecture": "maintained", "sessions": 100, "duration_s": 10,
         "bytes_per_session": 840}]), False))
    cases.append(("memory up 5%", _mutate(architecture=[
        {"architecture": "maintained", "sessions": 100, "duration_s": 10,
         "bytes_per_session": 735}]), True))

    failures = 0
    for name, run, want_pass in cases:
        got_pass = not verdict(_baseline(), run).failures
        ok = got_pass == want_pass
        failures += not ok
        print(f"  {'ok  ' if ok else 'FAIL'}  {name}: "
              f"{'accepted' if got_pass else 'rejected'}"
              f"{'' if ok else f', wanted {"accepted" if want_pass else "rejected"}'}")

    # A false-positive rate that HAD a value and lost it. Kept separate
    # because it is the one rule where the BASELINE is what differs.
    base = _baseline()
    base["false_positive_rate"] = 0.012
    lost = not verdict(base, _mutate()).failures
    failures += lost
    print(f"  {'FAIL' if lost else 'ok  '}  a false-positive rate that was lost: "
          f"{'accepted' if lost else 'rejected'}")

    # The bug this module exists because of: manifests sorted by
    # FILENAME put the older one last, because the name carries a
    # content hash rather than a date.
    paths = ["numbers-2026-08-04-2f56b63d.json", "numbers-2026-08-04-0b0af2d1.json"]
    stamps = {paths[0]: "2026-08-04T07:08:07Z", paths[1]: "2026-08-04T13:20:15Z"}
    picked = pick_newest(paths, stamps.get)
    wrong = picked != paths[1]
    failures += wrong
    print(f"  {'FAIL' if wrong else 'ok  '}  the newest manifest is chosen by date, "
          f"not by filename: picked {picked}")
    if sorted(paths)[-1] == paths[1]:
        failures += 1
        print("  FAIL  this case no longer reproduces the ordering bug it guards")

    if pick_newest([], lambda p: p) is not None:
        failures += 1
        print("  FAIL  an empty manifest list should have no newest member")
    else:
        print("  ok    an empty manifest list has no newest member")

    manifests = {"a.json": ("monolith", "2026-01-01T00:00:00Z"),
                 "b.json": ("composed", "2026-06-01T00:00:00Z"),
                 "c.json": ("monolith", "2026-03-01T00:00:00Z")}
    newest_monolith = pick_newest([p for p, (t, _) in manifests.items() if t == "monolith"],
                                  lambda p: manifests[p][1])
    if newest_monolith != "c.json":
        failures += 1
        print(f"  FAIL  a monolith run should pick the newest MONOLITH manifest, "
              f"picked {newest_monolith}")
    else:
        print("  ok    the baseline is the newest of the SAME topology, not the newest")

    if no_baseline() == 0:
        failures += 1
        print("  FAIL  no baseline was reported as a pass; nothing was compared")
    else:
        print("  ok    no baseline is a failure, not a pass")

    # Load condition, the same shape as topology.
    idle = {"provenance": {"run": {"load": "idle"}}}
    loaded = {"provenance": {"run": {"load": "sustained-2000rps"}}}

    r = Report()
    check_load(idle, loaded, r)
    if not r.failures:
        failures += 1
        print("  FAIL  an idle baseline against a loaded run was not refused")
    else:
        print("  ok    an idle baseline against a loaded run is refused")

    r = Report()
    check_load(idle, {"provenance": {"run": {}}}, r)
    if r.failures:
        failures += 1
        print("  FAIL  a manifest with no load field should be treated as idle")
    else:
        print("  ok    a manifest with no load field is treated as idle, which is "
              "what every run before Phase 4 was")

    # At the budget is not over it. Baseline and run share the value so
    # the band stays out of the way — the boundary rule and the band
    # rule are different rules and this case exercises only the first.
    at_budget = _baseline()
    at_budget["session"]["latency"] = {"p50": 2.0, "p95": 4.0, "p99": 80.0,
                                       "budget_ms": 80}
    r = Report()
    check_session(at_budget, json.loads(json.dumps(at_budget)), r)
    if r.failures:
        failures += 1
        print("  FAIL  p99 exactly at budget was rejected: " + r.failures[0])
    else:
        print("  ok    p99 exactly at budget passes; over it does not")

    # An unreplayable adversary is not an anchor — null, empty and
    # missing all read as unseeded; a real seed does not.
    seedless = _baseline()
    seedless["provenance"]["run"]["seed"] = None
    missing = _baseline()
    del missing["provenance"]["run"]["seed"]
    if unseeded(seedless) and unseeded(missing) and not unseeded(_baseline()):
        print("  ok    a baseline with no adversary seed is refused as an anchor")
    else:
        failures += 1
        print("  FAIL  an unseeded baseline was accepted as an anchor")

    total = len(cases) + 7
    print(f"\n  {total - failures}/{total} numbers-check cases hold")
    return 1 if failures else 0


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--run", default=DEFAULT_RUN, help="the run to check")
    ap.add_argument("--baseline", default=None,
                    help="manifest to check against (default: the newest published)")
    ap.add_argument("--selftest", action="store_true",
                    help="prove every rule above can reject something")
    args = ap.parse_args()

    if args.selftest:
        return selftest()

    if not os.path.exists(args.run):
        print(f"numbers-check: no run at {args.run}", file=sys.stderr)
        print("fix: make numbers", file=sys.stderr)
        return 1
    run = load(args.run)

    want = topology(run)
    want_load = load_condition(run)
    baseline_path = args.baseline or newest_manifest(want, want_load)
    if baseline_path is None:
        return no_baseline(f"{want} under {want_load} load")

    base = load(baseline_path)

    if unseeded(base):
        print(f"numbers-check: the baseline "
              f"{os.path.relpath(baseline_path, ROOT)} records no adversary "
              f"seed — it predates seeding, when tier 6 measured 70%, 90%, "
              f"80% and 100% from unchanged code. An adversary nobody can "
              f"replay cannot anchor a comparison.", file=sys.stderr)
        print("fix: compare against a seeded manifest, or publish one with "
              "`make numbers-manifest`", file=sys.stderr)
        return 1

    bmaj = str(base.get("schema_version", "")).split(".")[0]
    rmaj = str(run.get("schema_version", "")).split(".")[0]
    if bmaj != rmaj:
        print(f"numbers-check: schema {base.get('schema_version')} -> "
              f"{run.get('schema_version')} is a major change; the two runs do not "
              f"describe the same thing and cannot be compared.", file=sys.stderr)
        return 1

    rep = verdict(base, run)

    prov = base["provenance"]
    print(f"numbers-check: against {os.path.relpath(baseline_path, ROOT)}")
    print(f"               generated {prov['generated_at']}, "
          f"seed {prov['run'].get('seed', '(none recorded)')}, "
          f"commit {prov['git']['commit'][:12]}")
    print(f"               topology {topology(base)} -> {topology(run)}")
    print()

    for tier in sorted(run["detection"]):
        r = run["detection"][tier]
        was = base["detection"].get(tier)
        shown = (f"{was['detected']}/{was['n']} -> " if was else "new: ")
        print(f"  {tier:34} {shown}{r['detected']}/{r['n']}")
    if run.get("session") is not None:
        lat = run["session"]["latency"]
        print(f"  {'p99 decision latency':34} {lat['p99']}ms / {lat['budget_ms']}ms budget")
        print(f"  {'cold start never blocks':34} {run['session']['cold_start']['never_blocks']}")
    else:
        print(f"  {'session':34} not measured (null)")
    print(f"  {'false-positive rate':34} {run.get('false_positive_rate')}")
    print()

    for n in rep.notes:
        print(f"  note: {n}")
    if rep.notes:
        print()

    if rep.failures:
        for f in rep.failures:
            print(f"  FAIL: {f}", file=sys.stderr)
        print(file=sys.stderr)
        print("A number moved. If you meant it to, say so explicitly in the pull "
              "request and publish a manifest:", file=sys.stderr)
        print("    make numbers-manifest", file=sys.stderr)
        print("A moved number with no explanation is indistinguishable from a "
              "broken detector.", file=sys.stderr)
        return 1

    print("  the six numbers reproduce")
    return 0


if __name__ == "__main__":
    sys.exit(main())
