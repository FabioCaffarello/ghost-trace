"""
Produce the six numbers. One command, from a clean checkout.

    python3 experiments/numbers.py

Builds the binary, starts it on a private port and data directory, runs
every adversarial tier, measures latency / time-to-confident-decision /
cold start, runs the two-architecture benchmark, and writes
experiments/results/numbers.json plus a printed table.

Contract §9 names five of these and calls them the difference between a
project and a stack list. The sixth is the architecture benchmark.

The false-positive rate is the one that governs everything and it is
computed from whatever real human capture exists. If none exists it
prints NO DATA — never a placeholder, never a zero, and never a number
derived from synthetic sessions. A blank in that row is the honest
state of the project; a fabricated value there would invalidate the
other five.
"""
import json
import os
import pathlib
import shutil
import signal
import subprocess
import sys
import time
import urllib.error
import urllib.request

HERE = pathlib.Path(__file__).resolve().parent
ROOT = HERE.parent
SVC = ROOT / "services" / "ingestion"
RESULTS = HERE / "results"

PORT = int(os.environ.get("GT_PORT", "18099"))

# GT_BASE in the environment points the run at an ALREADY-RUNNING slice
# — the compose experiments profile sets it to the ingestion container.
# In that mode nothing is built or started here, and the architecture
# benchmark only runs if a Go toolchain is present (it is an in-process
# measurement, not a client of the slice).
EXTERNAL_BASE = os.environ.get("GT_BASE", "")
BASE = EXTERNAL_BASE or f"http://127.0.0.1:{PORT}"

# Sample sizes. Bot tiers are free, so they are generous; the browser
# tiers cost about two seconds a session, so they are not. The defaults
# are the published run; GT_N_TIER<k> overrides one tier for smoke runs
# without changing what the canonical command measures.
def _n(tier, default):
    return int(os.environ.get(f"GT_N_TIER{tier}", str(default)))


TIER_N = {
    "tier1_playwright.js": _n(1, 12),
    "tier2_puppeteer_stealth.js": _n(2, 12),
    "tier4_synthetic_linear.js": _n(4, 100),
    "tier5_humanised.js": _n(5, 10),
    "tier6_value_injection.js": _n(6, 10),
}


def log(msg):
    print(f"  {msg}", flush=True)


def wait_healthy(timeout=30):
    deadline = time.time() + timeout
    while time.time() < deadline:
        try:
            with urllib.request.urlopen(BASE + "/healthz", timeout=2) as r:
                if r.status == 200:
                    return True
        except (urllib.error.URLError, OSError):
            time.sleep(0.4)
    return False


def run(cmd, env=None, cwd=None, timeout=2400):
    e = dict(os.environ)
    e.setdefault("SETUPTOOLS_USE_DISTUTILS", "local")
    e["GT_BASE"] = BASE
    if env:
        e.update(env)
    return subprocess.run(cmd, cwd=cwd or HERE, env=e, capture_output=True,
                          text=True, timeout=timeout)


def venv_python():
    p = HERE / ".venv" / "bin" / "python"
    return str(p) if p.exists() else sys.executable


def main():
    RESULTS.mkdir(parents=True, exist_ok=True)
    data_dir = RESULTS / "numbers-run"
    if data_dir.exists():
        shutil.rmtree(data_dir)
    data_dir.mkdir(parents=True)

    sessions_file = RESULTS / "sessions.jsonl"
    if sessions_file.exists():
        sessions_file.unlink()

    print("\nGhost Trace — producing the six numbers\n")

    # ---- build + start (or target an external slice) ----
    proc = None
    if EXTERNAL_BASE:
        log(f"targeting external slice at {BASE}")
    else:
        log("building")
        b = subprocess.run(["go", "build", "-o", str(data_dir / "ghost-trace"),
                            "./cmd/ghost-trace"], cwd=SVC, capture_output=True, text=True)
        if b.returncode != 0:
            print(b.stderr)
            return 1

        log(f"starting slice on {BASE}")
        proc = subprocess.Popen(
            [str(data_dir / "ghost-trace"), "-addr", f"127.0.0.1:{PORT}",
             "-data", str(data_dir / "substrate")],
            stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
        )
    try:
        if not wait_healthy():
            log("slice did not become healthy")
            return 1

        # ---- 1. detection rate per tier ----
        absent = []
        for script, n in TIER_N.items():
            log(f"tier: {script} (n={n})")
            r = run(["node", str(HERE / "tiers" / script)], env={"GT_N": str(n)})
            if r.returncode != 0:
                lines = (r.stderr or "").strip().splitlines()
                absent.append(f"{script}: {lines[-1] if lines else 'failed'}")

        n3 = _n(3, 8)
        log(f"tier: tier3_undetected_chromedriver (n={n3})")
        r = run([venv_python(), str(HERE / "tiers" / "tier3_undetected_chromedriver.py")],
                env={"GT_N": str(n3)})
        if r.returncode != 0:
            lines = (r.stderr or "").strip().splitlines()
            absent.append(
                f"tier3_undetected_chromedriver.py: {lines[-1] if lines else 'failed'}")

        # ---- 3, 4, 5 ----
        log("latency, time-to-confident-decision, cold start")
        r = run(["node", str(HERE / "measure_session.mjs")])
        session_numbers = json.loads(r.stdout) if r.returncode == 0 else None
        if session_numbers is None:
            log(f"session measurement failed: {(r.stderr or '')[-300:]}")
    finally:
        if proc is not None:
            proc.send_signal(signal.SIGTERM)
            try:
                proc.wait(timeout=10)
            except subprocess.TimeoutExpired:
                proc.kill()

    # ---- 6. two-architecture benchmark ----
    bench = None
    if shutil.which("go"):
        log("architecture benchmark (concurrency x duration)")
        bench_out = RESULTS / "bench-architecture.json"
        subprocess.run(
            ["go", "run", "./cmd/bench-architecture",
             "-sessions", "100,1000,10000", "-durations", "10,300,1800",
             "-sample", "200", "-out", str(bench_out)],
            cwd=SVC, capture_output=True, text=True, timeout=3600,
        )
        bench = json.loads(bench_out.read_text()) if bench_out.exists() else None
    else:
        # Absence must be explicit, never a silently thinner report: the
        # containerised run has no Go toolchain, and number 6 is an
        # in-process measurement that belongs to the local run.
        log("architecture benchmark SKIPPED — no Go toolchain on PATH")

    # ---- assemble ----
    detection = summarize_tiers()

    # Absence is derived from the data, not only from exit codes. The
    # tiers now exit non-zero when zero sessions complete, but that is
    # the first line of defence; this is the second. A cohort with no
    # rows in sessions.jsonl is recorded ABSENT — a tier reported as
    # neither present nor absent would silently shrink the detection
    # table, which is exactly the failure this layer exists to prevent.
    for script in [*TIER_N, "tier3_undetected_chromedriver.py"]:
        stem = script.rsplit(".", 1)[0]        # e.g. tier5_humanised
        prefix = stem.split("_", 1)[0] + "_"   # e.g. tier5_ (cohort names may carry suffixes)
        if any(c.startswith(prefix) for c in detection):
            continue
        if any(a.startswith(stem) for a in absent):
            continue
        absent.append(f"{script}: completed with zero sessions")

    numbers = {
        "detection": detection,
        "false_positive_rate": human_fpr(),
        "session": session_numbers,
        "architecture": bench,
        "absent_tiers": absent,
    }
    (RESULTS / "numbers.json").write_text(json.dumps(numbers, indent=2))
    report(numbers)
    return 0


def summarize_tiers():
    p = RESULTS / "sessions.jsonl"
    if not p.exists():
        return {}
    from collections import defaultdict
    rows = [json.loads(l) for l in p.read_text().splitlines() if l.strip()]
    by = defaultdict(list)
    for r in rows:
        by[r["cohort"]].append(r)

    sys.path.insert(0, str(HERE))
    from analyze import flagged, wilson  # noqa: E402

    out = {}
    for cohort, rs in by.items():
        n = len(rs)
        det = sum(1 for r in rs if flagged(r))
        lo, hi = wilson(det, n)
        out[cohort] = {
            "n": n, "detected": det, "rate": det / n,
            "ci_low": lo, "ci_high": hi,
            "mean_score": sum(r["score"] for r in rs) / n,
            "mean_confidence": sum(r["confidence"] for r in rs) / n,
        }
    return out


def human_fpr():
    """The governing number, computed only from real capture.

    Returns None when no human sessions exist. That blank is the honest
    state of the project — a placeholder here would quietly invalidate
    every other number, because a detection rate without a
    false-positive rate is not a measurement."""
    p = RESULTS / "human_sessions.jsonl"
    if not p.exists():
        return None
    rows = [json.loads(l) for l in p.read_text().splitlines() if l.strip()]
    arm_b = [r for r in rows if r.get("arm") == "B"]
    if not arm_b:
        return None

    from collections import defaultdict
    sys.path.insert(0, str(HERE))
    from analyze import design_effect, flagged, icc_anova, wilson  # noqa: E402

    by_person = defaultdict(list)
    for r in arm_b:
        by_person[r["participant"]].append(r)

    people = len(by_person)
    sessions = len(arm_b)
    people_flagged = sum(1 for rs in by_person.values() if any(flagged(r) for r in rs))
    lo, hi = wilson(people_flagged, people)

    binary = {p_: [1 if flagged(r) else 0 for r in rs] for p_, rs in by_person.items()}
    rho = icc_anova(binary)
    deff = design_effect(rho, sessions / people) if rho is not None else None
    n_eff = sessions / deff if deff else people

    return {
        "people": people, "sessions": sessions,
        "people_flagged": people_flagged,
        "person_ci_low": lo, "person_ci_high": hi,
        "rho": rho, "design_effect": deff, "effective_n": n_eff,
    }


def report(nums):
    line = "=" * 78
    print(f"\n{line}\nTHE SIX NUMBERS\n{line}\n")

    print("1. DETECTION RATE PER ADVERSARIAL TIER\n")
    det = nums["detection"]
    if det:
        print(f"   {'tier':<32} {'n':>4} {'detected':>9} {'rate':>7}  {'95% CI':>16}")
        for cohort in sorted(det):
            d = det[cohort]
            print(f"   {cohort:<32} {d['n']:>4} {d['detected']:>9} {d['rate']:>6.1%}  "
                  f"[{d['ci_low']:>5.1%}, {d['ci_high']:>5.1%}]")
    else:
        print("   no bot sessions recorded")

    print("\n2. FALSE-POSITIVE RATE ON HUMAN TRAFFIC\n")
    fpr = nums["false_positive_rate"]
    if fpr is None:
        print("   NO DATA — no human sessions have been captured.\n")
        print("   This is the number that governs everything: a detector that")
        print("   flags every session scores 100% on every tier above. Until")
        print("   this row has a value, number 1 is not a measurement.\n")
        print("   It is calendar-bound, not effort-bound. See experiments/README.md.")
    else:
        print(f"   {fpr['people_flagged']} of {fpr['people']} people falsely flagged "
              f"at least once")
        print(f"   95% CI on the proportion of people: "
              f"[{fpr['person_ci_low']:.1%}, {fpr['person_ci_high']:.1%}]")
        if fpr["rho"] is not None:
            print(f"   rho (estimated) = {fpr['rho']:.3f}   effective n = "
                  f"{fpr['effective_n']:.1f} of {fpr['sessions']}")

    s = nums.get("session") or {}
    print("\n3. p99 DECISION LATENCY (single session, idle system)\n")
    lat = s.get("latency")
    if lat:
        print(f"   p50 {lat['p50']}ms   p95 {lat['p95']}ms   p99 {lat['p99']}ms   "
              f"max {lat['max']}ms")
        print(f"   budget: {lat['budget_ms']}ms")

    print("\n4. TIME TO CONFIDENT DECISION\n")
    t = s.get("ttcd") or {}
    for key, label in (("challenge", "challenge possible (confidence >= 0.40)"),
                       ("block", "block possible (confidence >= 0.70)")):
        v = t.get(key)
        if v:
            print(f"   {label}")
            print(f"     reached in {v['reached']}/{v['of']} sessions — median "
                  f"{v['batches']} batches, {v['events']} events, {v['seconds']}s")

    print("\n5. COLD-START BEHAVIOUR (first visit, no history)\n")
    c = s.get("cold_start")
    if c:
        print(f"   n={c['n']}   median score {c['score']}   median confidence "
              f"{c['confidence']}")
        print(f"   decisions: {', '.join(c['decisions'])}")
        print(f"   reasons:   {', '.join(c['reasons'])}")
        print(f"   never blocks: {c['never_blocks']}")

    print("\n6. MEMORY AND LATENCY BY CONCURRENCY x DURATION\n")
    b = nums.get("architecture")
    if not b:
        print("   not measured in this run (requires the Go toolchain;")
        print("   python3 experiments/numbers.py locally produces it)")
    if b:
        print(f"   {'architecture':<14} {'sessions':>9} {'duration':>9} {'heap MB':>10} "
              f"{'bytes/sess':>12} {'p99 ms':>8}")
        for c_ in b:
            print(f"   {c_['architecture']:<14} {c_['sessions']:>9} "
                  f"{c_['duration_s']:>8}s {c_['heap_mb']:>10.1f} "
                  f"{c_['bytes_per_session']:>12} {c_['decide_p99_ms']:>8.3f}")

    if nums.get("absent_tiers"):
        print("\nTIERS THAT DID NOT RUN")
        for a in nums["absent_tiers"]:
            print(f"   {a}")
        print("   A tier that did not run is not a tier that found nothing.")

    print(f"\n{line}")
    print("wrote experiments/results/numbers.json")


if __name__ == "__main__":
    sys.exit(main())
