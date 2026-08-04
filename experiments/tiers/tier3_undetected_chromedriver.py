"""
Tier 3 — undetected-chromedriver.

Selenium plus undetected-chromedriver, the standard answer for evading
Cloudflare/Distil-style bot walls. Like tier 2's stealth plugin, every
evasion it implements targets the browser's identity: a patched
chromedriver binary, CDP-runtime concealment, and the automation flags
that give Selenium away.

Its mouse is also its most interesting property. Selenium's ActionChains
does not interpolate — move_to_element jumps the cursor to the target in
a single step, so almost no pointermove fires at all. That is a real
evasion of pointer-based detection, but it works by producing no
evidence rather than by producing human-looking evidence, and the
confidence dimension is what reports it. Expect low confidence and
therefore `allow`: a true negative that is not a true negative, and the
most useful thing this tier can tell us.

Optional. If undetected-chromedriver cannot install (it lags new Python
releases), run.py records the tier as ABSENT rather than skipping it
silently — a tier that did not run must never read as a tier that
produced nothing to find.
"""
import json
import os
import pathlib
import subprocess
import sys
import time
import urllib.request

BASE = os.environ.get("GT_BASE", "http://127.0.0.1:8080")
SECRET_KEY = os.environ.get("GT_SECRET_KEY", "sk_demo")
COHORT = "tier3_undetected_chromedriver"
N = int(os.environ.get("GT_N", "15"))

HERE = pathlib.Path(__file__).resolve().parent
RESULTS = HERE.parent / "results"

# The wire shapes live one level up, in the single module that defines
# them for the Python side of the harness (R1.15 / audit M22).
sys.path.insert(0, str(HERE.parent))
from wire import decision_body  # noqa: E402


def decide(token, subject):
    req = urllib.request.Request(
        BASE + "/v1/decisions",
        data=json.dumps(decision_body(token, subject_id=subject)).encode(),
        headers={
            "Content-Type": "application/json",
            "Authorization": "Bearer " + SECRET_KEY,
        },
    )
    with urllib.request.urlopen(req) as r:
        return json.loads(r.read())


def chrome_major():
    """Major version of the installed Chrome, or None to let uc guess."""
    if os.environ.get("GT_CHROME_MAJOR"):
        return int(os.environ["GT_CHROME_MAJOR"])
    chrome = os.environ.get(
        "GT_CHROME",
        "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
    )
    try:
        out = subprocess.run([chrome, "--version"], capture_output=True, text=True, timeout=20)
        for tok in (out.stdout or "").split():
            if tok[:1].isdigit() and "." in tok:
                return int(tok.split(".")[0])
    except Exception:  # noqa: BLE001
        pass
    return None


def append(row):
    RESULTS.mkdir(parents=True, exist_ok=True)
    with open(RESULTS / "sessions.jsonl", "a") as f:
        f.write(json.dumps({"cohort": COHORT, **row}) + "\n")


def main():
    try:
        import undetected_chromedriver as uc
        from selenium.webdriver.common.by import By
        from selenium.webdriver.common.action_chains import ActionChains
    except Exception as e:  # noqa: BLE001
        print(f"  {COHORT}: ABSENT — {type(e).__name__}: {e}", file=sys.stderr)
        print(f"  {COHORT}: install with `pip install undetected-chromedriver selenium`", file=sys.stderr)
        return 2

    opts = uc.ChromeOptions()
    opts.add_argument("--headless=new")
    opts.add_argument("--window-size=1440,900")

    # undetected-chromedriver fetches the newest ChromeDriver by default,
    # which refuses to attach when it outruns the installed Chrome. Pin
    # it to whatever this machine actually has.
    driver = uc.Chrome(options=opts, version_main=chrome_major())

    rows = []
    try:
        for i in range(N):
            try:
                driver.get(BASE + "/")
                time.sleep(1.0)

                user = driver.find_element(By.ID, "u")
                pwd = driver.find_element(By.ID, "p")

                # ActionChains jumps rather than interpolating. This is
                # the evasion-by-absence described above; it is not a
                # humanised path.
                ActionChains(driver).move_to_element(user).click().perform()
                user.clear()
                user.send_keys(f"bot{i}@example.com")
                ActionChains(driver).move_to_element(pwd).click().perform()
                pwd.send_keys("hunter2")

                time.sleep(2.5)

                token = driver.execute_script("return window.GhostTrace.token();")
                driver.execute_script("window.GhostTrace.flush();")
                time.sleep(0.3)

                d = decide(token, f"{COHORT}_{i}")
                row = {
                    "i": i,
                    "score": d["score"],
                    "confidence": d["confidence"],
                    "decision": d["decision"],
                    "shadow_decision": d.get("shadow_decision"),
                    "events": d["evidence"]["events"],
                    "duration_ms": d["evidence"]["duration_ms"],
                    "reasons": [r["code"] for r in d["reasons"]],
                }
                rows.append(row)
                append(row)
            except Exception as e:  # noqa: BLE001
                print(f"  {COHORT} session {i} failed: {e}", file=sys.stderr)
    finally:
        try:
            driver.quit()
        except Exception:  # noqa: BLE001
            pass

    if not rows:
        # Fail loudly: per-session errors are caught inside the loop, so a
        # run where every session failed would otherwise exit 0 and vanish
        # from the results as neither present nor absent.
        print(f"  {COHORT}: no sessions completed", file=sys.stderr)
        return 1

    n = len(rows)
    flagged = sum(1 for r in rows if (r["shadow_decision"] or r["decision"]) != "allow")
    print(
        f"  {COHORT:<26} n={n:>3}  "
        f"score={sum(r['score'] for r in rows)/n:.3f}  "
        f"conf={sum(r['confidence'] for r in rows)/n:.3f}  "
        f"events={round(sum(r['events'] for r in rows)/n)}  "
        f"flagged={flagged}/{n}"
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
