#!/usr/bin/env python3
"""Drive synthetic participants through the capture protocol, end to end.

The false-positive rate is the one published number that is `null`, and
it is calendar-bound rather than effort-bound: it needs recruited people,
and no amount of engineering shortens the recruiting. What engineering
CAN do is make sure that recruiting is the only thing left — that on the
day a volunteer opens the link, nothing in the pipeline is discovered to
be broken.

This is that check. It drives participants who do not exist through the
same two services a real one would, and asserts three things:

1. **The protocol runs.** A session, a telemetry batch, a login, and a
   labelled row in the capture log. Before PR-4.P2 the documented
   command did not run at all — it passed `-capture-log` to the
   collector, which has no such flag.
2. **Every participant produces exactly one row**, carrying the cohort
   labels the link supplied.
3. **The cohort labels never cross the wire.** `experiments/README.md`
   claims only `participant` travels, as `subject_id`, because the engine
   must not know which population it is looking at. Nothing checked it.

It refuses rather than skips when the services are not up, like every
other gate here.
"""

from __future__ import annotations

import argparse
import json
import os
import pathlib
import re
import sqlite3
import sys
import urllib.error
import urllib.request

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import wire  # noqa: E402

COLLECTOR = os.environ.get("GT_BASE", "http://127.0.0.1:8080")
DEMO = os.environ.get("GT_DEMO_URL", "http://127.0.0.1:8083")

# Printed when the protocol ran and recorded nothing. It names the two
# ways to be in that state, because they look identical from here: this
# script talks to demo-web over HTTP and demo-web does not report
# whether it has a sink.
NO_SINK_HELP = """
  Capture is off unless demo-web is given a log to write.

  in compose:
    GT_CAPTURE_LOG=/captures/human_sessions.jsonl \\
      docker compose --profile demo up -d --force-recreate demo-web
    (./experiments/results is mounted at /captures)

  standalone:
    go run ./services/demo-web/cmd/demo-web -addr 127.0.0.1:8083 \\
      -api http://127.0.0.1:8080 -engine http://127.0.0.1:8082 \\
      -capture-log experiments/results/human_sessions.jsonl

  Refusing rather than reporting three participants and no rows: a study
  that recorded nobody must not be distinguishable only by someone
  noticing the file is empty.
"""

# Distinctive on purpose. A condition label is searched for as raw bytes
# in the archive, so it has to be a string that cannot appear there by
# accident — see `arm` in leaked() for the case where that is impossible.
PARTICIPANTS = [
    {"participant": "pdry01", "arm": "A", "condition": "dryrun-trackpad-quiet", "visit": 1},
    {"participant": "pdry02", "arm": "B", "condition": "dryrun-mouse-desktop", "visit": 1},
    {"participant": "pdry03", "arm": "A", "condition": "dryrun-touch-mobile", "visit": 7},
]


class Failures(list):
    def check(self, ok: bool, what: str) -> None:
        print(f"  {'ok  ' if ok else 'FAIL'}  {what}")
        if not ok:
            self.append(what)


def post(base, path, body, bearer=None):
    req = urllib.request.Request(base + path, data=json.dumps(body).encode(),
                                 headers={"Content-Type": "application/json"})
    if bearer:
        req.add_header("Authorization", "Bearer " + bearer)
    with urllib.request.urlopen(req, timeout=15) as r:
        raw = r.read()
        return r.status, (json.loads(raw) if raw else {})


def healthy(base) -> bool:
    try:
        with urllib.request.urlopen(base + "/healthz", timeout=3) as r:
            return r.status == 200
    except Exception:
        return False


def one_participant(p: dict) -> tuple[str, str]:
    """A whole visit: session, telemetry, login. Returns (token, decision)."""
    _, out = post(COLLECTOR, "/v1/sessions", wire.session_body("pk_demo", "/"))
    token = out["session_token"]

    events = [
        wire.pointer_event(120, [[10, 20, 0], [14, 26, 16], [19, 31, 17]]),
        wire.focus_event(80, "focus", "username"),
        wire.key_event(300, "dwell", "alpha", "username"),
        wire.key_event(390, "flight", "alpha", "username"),
        wire.form_event(700, "submit", "username"),
    ]
    post(COLLECTOR, "/v1/telemetry", wire.telemetry_body(token, 1, 900, events),
         bearer=token)

    _, out = post(DEMO, "/demo/login", {"session_token": token,
                                        "username": "volunteer", **p})
    return token, out.get("decision", "")


def schema_declares_cohort_fields(schemas: pathlib.Path) -> list[str]:
    """Cohort labels must not exist as archived fields at all.

    Checked against the schemas rather than against captured bytes: a
    field that is absent from every message cannot leak from any run,
    which is a stronger statement than one run happening not to contain
    it.
    """
    banned = ("arm", "condition", "visit", "cohort")
    found = []
    for proto in sorted(schemas.glob("*.proto")):
        for m in re.finditer(r"^\s*\w+\s+(\w+)\s*=\s*\d+;", proto.read_text(), re.M):
            if m.group(1) in banned:
                found.append(f"{proto.name}:{m.group(1)}")
    return found


def leaked(db: pathlib.Path, needles: list[str]) -> list[str]:
    """Which needles appear in archived payload bytes.

    Byte search rather than protobuf decoding, because the archive is
    content-addressed bytes and this repository generates no Python
    bindings. That is fine for a distinctive label and useless for a
    short one: `arm` is "A" or "B", one byte, which appears in binary
    payloads by chance — an early version of this check "found" a leak of
    arm B that was a length prefix. Single-character labels are covered
    by the schema check above instead, which does not depend on luck.
    """
    if not db.exists():
        return []
    c = sqlite3.connect(f"file:{db}?mode=ro", uri=True)
    try:
        blob = b"".join(r[0] or b"" for r in c.execute("SELECT payload FROM events"))
    finally:
        c.close()
    return [n for n in needles if n.encode() in blob]


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--capture-log", default="experiments/results/human_sessions.jsonl",
                    help="the log demo-web was started with")
    ap.add_argument("--data", default="",
                    help="the collector's -data directory, to check for label leaks")
    args = ap.parse_args()

    for name, base in (("collector", COLLECTOR), ("demo-web", DEMO)):
        if not healthy(base):
            print(f"the {name} is not answering at {base}.")
            print("\nbring both up (see experiments/README.md — the capture protocol):")
            print("  go run ./services/collector/cmd/ghost-trace -data .run-data "
                  "-addr 127.0.0.1:8080")
            print("  go run ./services/demo-web/cmd/demo-web -addr 127.0.0.1:8083 "
                  "-api http://127.0.0.1:8080 \\\n    -capture-log "
                  "experiments/results/human_sessions.jsonl")
            print("\nrefusing rather than skipping: a dry run that quietly does "
                  "nothing is the failure this check exists to remove.")
            return 1

    log = pathlib.Path(args.capture_log)
    before = len(log.read_text().splitlines()) if log.exists() else 0

    f = Failures()
    print(f"\n== {len(PARTICIPANTS)} synthetic participants through the real protocol ==")
    for p in PARTICIPANTS:
        _, decision = one_participant(p)
        f.check(decision != "", f"{p['participant']} reached a decision ({decision or 'none'})")

    rows = ([json.loads(l) for l in log.read_text().splitlines()[before:] if l.strip()]
            if log.exists() else [])

    # A demo-web with no capture sink answers every request exactly as
    # one with a sink does: the participants reach decisions, the page
    # works, and nothing anywhere says the study recorded nobody. That is
    # the whole failure mode, and it is why this check is separate from
    # the row-count one below and reports first.
    #
    # It used to be impossible to reach — capture was a second service
    # that existed only with the flag, so `--profile demo` WAS capture.
    # Folding that service away made the flag a variable, and a variable
    # can be forgotten. This is the guard that came with the fold.
    if not rows:
        f.check(False,
                f"the capture log grew during the run ({log}) — it did not, so "
                f"demo-web is running with NO capture sink and the study "
                f"recorded nobody")
        print(NO_SINK_HELP)
        return report(f)

    f.check(len(rows) == len(PARTICIPANTS),
            f"one row per participant ({len(rows)} of {len(PARTICIPANTS)})")

    by_code = {r.get("participant"): r for r in rows}
    for p in PARTICIPANTS:
        r = by_code.get(p["participant"])
        if r is None:
            f.check(False, f"{p['participant']} has a row")
            continue
        f.check(all(r.get(k) == v for k, v in p.items()),
                f"{p['participant']} kept its labels "
                f"(arm={r.get('arm')} condition={r.get('condition')} visit={r.get('visit')})")
        f.check(isinstance(r.get("confidence"), (int, float)) and r.get("events", 0) > 0,
                f"{p['participant']} carries evidence, not just a verdict "
                f"(events={r.get('events')} confidence={r.get('confidence')})")

    print("\n== the cohort labels do not cross the wire ==")
    schemas = pathlib.Path(__file__).resolve().parent.parent / "schemas/events/v1"
    declared = schema_declares_cohort_fields(schemas)
    f.check(not declared,
            "no archived message declares arm, condition, visit or cohort"
            + (f" — found {declared}" if declared else ""))

    if args.data:
        db = pathlib.Path(args.data) / "events.db"
        conditions = [p["condition"] for p in PARTICIPANTS]
        found = leaked(db, conditions)
        f.check(not found,
                "no condition label reached the archive"
                + (f" — found {found}" if found else ""))
        # And the participant code must NOT be there either. It used to
        # be, as subject_id, until ADR-0014: a pseudonym for a real
        # person, in an append-only store, permanently. The study's join
        # key is evaluation_id, which lives in the capture row — so
        # deleting the row severs the link, which is what deletion can
        # mean when the archive does not forget.
        codes = leaked(db, [p["participant"] for p in PARTICIPANTS])
        f.check(not codes,
                "no participant code reached the archive"
                + (f" — found {codes}, which `make forget` could not remove"
                   if codes else ""))
    else:
        print("  --data not given; the archive was not inspected. Absence of a "
              "check is not a passing check.")

    return report(f)


def report(f: Failures) -> int:
    if f:
        print(f"\n  {len(f)} claim(s) did not hold:")
        for what in f:
            print(f"    - {what}")
        return 1
    print("\n  the capture protocol runs end to end. Recruiting is what is left.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
