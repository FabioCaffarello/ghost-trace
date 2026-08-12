"""Write contract/fixtures/requests/ from the harness's wire modules.

The fixtures are not hand-written — they are what lib/wire.js and
wire.py actually produce. That is the whole point: a hand-written
fixture is a third description of the wire, free to drift from both
the clients and the server, and drifting quietly is the failure this
harness exists to prevent (audit M22).

Two checks happen here before anything is written:

  1. JAVASCRIPT AND PYTHON MUST AGREE. The harness has a JS half (the
     browser tiers, tier 4, the session measurements) and a Python half
     (tier 3). If they disagree about the wire, one of them is
     measuring something else, and no amount of server-side testing
     would show it.

  2. The result is deterministic — fixed tokens, fixed timestamps — so
     it can be committed and `make contract-fixtures-sync` can fail on
     any difference.

A Go test then validates every fixture against the published OpenAPI
request schema AND replays it against a real server, so a fixture is
both what the contract says and what the service accepts.

    make contract-fixtures        regenerate
    make contract-fixtures-sync   fail if they drift
"""
import json
import pathlib
import subprocess
import sys

import wire

HERE = pathlib.Path(__file__).resolve().parent
ROOT = HERE.parent
OUT_DIR = ROOT / "contract" / "fixtures" / "requests"

TOKEN = "st_AufHcXG3MEt9x5F3hzVf03ZS"

POINTER = wire.pointer_event(t=1200, pts=[[412, 300, 0], [418, 305, 16], [427, 311, 17]])
KEYS = [
    wire.key_event(t=1500, phase="down", key_class="alpha", target="f_1"),
    wire.key_event(t=1530, phase="up", key_class="alpha", target="f_1"),
]

# One of every family, in one batch — so no event type the SDK can emit
# is without a fixture that is schema-validated and replayed against a
# real server. `form`/`injected` is here on purpose: the policy's
# strongest signal was the one with no contract test.
ALL_FAMILIES = [
    POINTER,
    KEYS[0],
    wire.scroll_event(t=1600, dy=240),
    wire.focus_event(t=1700, state="focus", target="f_1"),
    wire.visibility_event(t=1800, state="hidden"),
    wire.form_event(t=1900, action="injected", target="f_1"),
]


def python_fixtures():
    return {
        "sessions": wire.session_body(site_key="pk_demo"),
        "telemetry_pointer": wire.telemetry_body(
            session_token=TOKEN, seq=0, sent_at_ms=2840, events=[POINTER]),
        "telemetry_pointer_and_keys": wire.telemetry_body(
            session_token=TOKEN, seq=1, sent_at_ms=4200, events=[POINTER, *KEYS]),
        "telemetry_all_families": wire.telemetry_body(
            session_token=TOKEN, seq=2, sent_at_ms=5100, events=ALL_FAMILIES),
        "decisions": wire.decision_body(session_token=TOKEN, subject_id="user_8f21"),
        "outcomes": wire.outcome_body(
            evaluation_id="ev_5Kq2mXbT9vHs", outcome="login_success"),
        "outcomes_with_observed_at": wire.outcome_body(
            evaluation_id="ev_5Kq2mXbT9vHs", outcome="fraud_confirmed",
            observed_at="2026-08-04T09:15:00Z"),
    }


def javascript_fixtures():
    r = subprocess.run(["node", str(HERE / "lib" / "emit_fixtures.mjs")],
                       capture_output=True, text=True)
    if r.returncode != 0:
        print(r.stderr, file=sys.stderr)
        raise SystemExit("lib/emit_fixtures.mjs failed")
    return json.loads(r.stdout)


def main(argv):
    # An output directory can be passed so the drift check can emit to
    # a temporary one and diff, rather than regenerating in place and
    # asking git — git reports a newly added directory as untracked and
    # the check would call that drift on the very commit introducing it.
    out_dir = pathlib.Path(argv[0]) if argv else OUT_DIR

    py = python_fixtures()
    js = javascript_fixtures()

    if set(py) != set(js):
        only_py = sorted(set(py) - set(js))
        only_js = sorted(set(js) - set(py))
        raise SystemExit(
            "the two halves of the harness define different fixtures\n"
            f"  only in wire.py:  {only_py}\n"
            f"  only in wire.js:  {only_js}")

    disagree = [name for name in sorted(py) if py[name] != js[name]]
    if disagree:
        print("JavaScript and Python disagree about the wire:\n", file=sys.stderr)
        for name in disagree:
            print(f"  {name}:", file=sys.stderr)
            print(f"    wire.js  {json.dumps(js[name], sort_keys=True)}", file=sys.stderr)
            print(f"    wire.py  {json.dumps(py[name], sort_keys=True)}", file=sys.stderr)
        print("\none half of the harness is measuring something else.", file=sys.stderr)
        return 1

    out_dir.mkdir(parents=True, exist_ok=True)
    for name in sorted(py):
        (out_dir / f"{name}.json").write_text(json.dumps(py[name], indent=2) + "\n")

    print(f"wrote {len(py)} fixtures to {out_dir} "
          f"(JavaScript and Python agree on all of them)")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
