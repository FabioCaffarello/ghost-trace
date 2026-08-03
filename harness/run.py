"""
Run the adversarial tiers against a running slice.

    cd services/ingestion && make run          # terminal 1
    python3 harness/run.py                     # terminal 2
    python3 harness/analyze.py

A tier whose dependencies are missing is recorded as ABSENT, not
skipped. Silent omission is the failure mode that makes a harness lie:
four tiers listed and three run reads as "we tested four things", and
the missing one is invariably the one that would have found something.
"""
import json
import os
import pathlib
import shutil
import subprocess
import sys
import urllib.error
import urllib.request

HERE = pathlib.Path(__file__).resolve().parent
RESULTS = HERE / "results"
BASE = os.environ.get("GT_BASE", "http://127.0.0.1:8080")

TIERS = [
    ("tier1_playwright_naive", "node", ["node", str(HERE / "tiers" / "tier1_playwright.js")],
     "playwright"),
    ("tier2_puppeteer_stealth", "node", ["node", str(HERE / "tiers" / "tier2_puppeteer_stealth.js")],
     "puppeteer-extra + stealth"),
    ("tier3_undetected_chromedriver", "python", [sys.executable, str(HERE / "tiers" / "tier3_undetected_chromedriver.py")],
     "undetected-chromedriver + selenium"),
    ("tier4_synthetic_linear", "node", ["node", str(HERE / "tiers" / "tier4_synthetic_linear.js")],
     "none (stdlib fetch)"),
    ("tier5_humanised_mouse", "node", ["node", str(HERE / "tiers" / "tier5_humanised.js")],
     "playwright"),
    ("tier6_value_injection", "node", ["node", str(HERE / "tiers" / "tier6_value_injection.js")],
     "playwright"),
]


def slice_is_up():
    try:
        with urllib.request.urlopen(BASE + "/healthz", timeout=3) as r:
            return r.status == 200
    except (urllib.error.URLError, OSError):
        return False


def main():
    if not slice_is_up():
        print(f"The slice is not reachable at {BASE}.")
        print("Start it first:  cd services/ingestion && make run")
        return 1

    RESULTS.mkdir(parents=True, exist_ok=True)
    absent = []

    print(f"\nGhost Trace — adversarial harness against {BASE}\n")

    for name, runtime, cmd, deps in TIERS:
        if runtime == "node" and shutil.which("node") is None:
            absent.append(f"{name} — node not installed")
            print(f"  {name}: ABSENT (node not installed)")
            continue

        print(f"  running {name} …")
        env = dict(os.environ)
        # undetected-chromedriver still imports distutils, removed in
        # Python 3.12+. setuptools ships a shim that this switch enables.
        env.setdefault("SETUPTOOLS_USE_DISTUTILS", "local")
        try:
            proc = subprocess.run(cmd, cwd=HERE, capture_output=True, text=True,
                                  timeout=1800, env=env)
        except subprocess.TimeoutExpired:
            absent.append(f"{name} — timed out after 30 minutes")
            print(f"  {name}: ABSENT (timed out)")
            continue

        out = (proc.stdout or "").strip()
        err = (proc.stderr or "").strip()
        if out:
            print(out)
        if proc.returncode != 0:
            reason = err.splitlines()[-1] if err else f"exit {proc.returncode}"
            absent.append(f"{name} — did not run ({deps}): {reason}")
            print(f"  {name}: ABSENT — {reason}")
        elif err:
            # Non-fatal per-session failures still matter.
            print("  " + err.splitlines()[-1])

    (RESULTS / "absent_tiers.txt").write_text("\n".join(absent) + ("\n" if absent else ""))

    if absent:
        print("\n  TIERS THAT DID NOT RUN:")
        for a in absent:
            print(f"    {a}")
        print("\n  A tier that did not run is not a tier that found nothing.")

    print("\n  Now: python3 harness/analyze.py")
    return 0


if __name__ == "__main__":
    sys.exit(main())
