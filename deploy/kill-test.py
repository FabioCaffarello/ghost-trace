#!/usr/bin/env python3
"""Kill-test: take one service away and check what the rest promises.

The composed topology makes three degradation promises, one per
architecture decision record, and until this existed all three were
prose:

  - the ARCHIVE is not on any request path (ADR-0006). Taking it away
    must change nothing a caller sees, and nothing may be lost while it
    is gone.
  - the DECISION ENGINE degrades only /v1/decisions (ADR-0005). Sessions
    and telemetry are the collector's, and the demo host fails open
    rather than breaking (contract §5).
  - the BROKER is the collector's only durable store since ADR-0006, so
    outcomes must refuse honestly — and everything best-effort must stay
    FAST, not merely succeed. That last one is here because it did not
    hold: a JetStream publish with no broker waits five seconds for an
    ack, and /v1/telemetry does two of them, so a 202 arrived ten
    seconds late. Fail-open in status, closed in latency.

It needs the topology up. Without it the script REFUSES rather than
skipping, because a kill-test that quietly does nothing is the vacuous
green this repository keeps finding.

    docker compose --profile core up -d
    make kill-test
"""

from __future__ import annotations

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
import wire  # noqa: E402

COLLECTOR = "http://127.0.0.1:8080"
ENGINE = "http://127.0.0.1:8082"
DEMO = "http://127.0.0.1:8083"
SECRET = "sk_demo"
SITE = "pk_demo"

# Generous multiples of app.BestEffortTimeout (250ms): /v1/sessions does
# one best-effort write and /v1/telemetry two. Loose enough not to flake
# on a busy machine, tight enough that the ten-second stall this test
# was written for cannot hide under them.
SESSIONS_BUDGET_S = 1.5
TELEMETRY_BUDGET_S = 2.5

UNREACHABLE = "unreachable"


class Failures(list):
    def check(self, ok: bool, what: str) -> None:
        print(f"  {'ok  ' if ok else 'FAIL'}  {what}")
        if not ok:
            self.append(what)


def post(base, path, body, bearer=None, timeout=30):
    req = urllib.request.Request(base + path, data=json.dumps(body).encode(),
                                 headers={"Content-Type": "application/json"})
    if bearer:
        req.add_header("Authorization", "Bearer " + bearer)
    t0 = time.time()
    try:
        with urllib.request.urlopen(req, timeout=timeout) as r:
            raw = r.read()
            return r.status, time.time() - t0, (json.loads(raw) if raw else {})
    except urllib.error.HTTPError as e:
        return e.code, time.time() - t0, {}
    except Exception:
        return UNREACHABLE, time.time() - t0, {}


def session():
    st, _, out = post(COLLECTOR, "/v1/sessions",
                      wire.session_body(SITE))
    return out.get("session_token", "") if st == 200 else ""


def telemetry(token, seq=1):
    return post(COLLECTOR, "/v1/telemetry",
                wire.telemetry_body(
                    session_token=token, seq=seq, sent_at_ms=900,
                    events=[wire.key_event(t=100, phase="down",
                                           key_class="alpha", target="f")]),
                bearer=token)


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


def stream_pending():
    """Records sitting in the stream that the archive has not acked."""
    r = subprocess.run(["docker", "compose", "exec", "-T", "nats", "wget", "-qO-",
                        "http://127.0.0.1:8222/jsz?streams=true&consumers=true"],
                       capture_output=True, text=True)
    if r.returncode != 0:
        return None
    try:
        d = json.loads(r.stdout)
    except json.JSONDecodeError:
        return None
    for acc in d.get("account_details", []):
        for st in acc.get("stream_detail", []):
            for c in st.get("consumer_detail", []):
                if c["name"] == "archive":
                    return c["num_pending"]
    return None


def scenario_archive_down(f: Failures) -> None:
    print("\n== the archive is not on any request path (ADR-0006) ==")
    compose("stop", "archive")
    try:
        token = session()
        f.check(token != "", "a session opens with the archive down")
        st, _, _ = telemetry(token)
        f.check(st == 202, f"telemetry is accepted ({st})")
        st, _, dec = post(ENGINE, "/v1/decisions",
                          wire.decision_body(token), bearer=SECRET)
        f.check(st == 200, f"the engine decides ({st})")
        st, _, _ = post(ENGINE, "/v1/outcomes",
                        wire.outcome_body(dec.get("evaluation_id", "ev_x"),
                                           "login_success"), bearer=SECRET)
        f.check(st == 202, f"an outcome is accepted — the stream holds it ({st})")

        pending_while_down = stream_pending()
        f.check(pending_while_down is not None and pending_while_down > 0,
                f"records queue in the stream rather than vanishing "
                f"(pending={pending_while_down})")
    finally:
        compose("start", "archive")

    # The claim that matters: nothing was lost while it was away.
    deadline = time.time() + 60
    pending = stream_pending()
    while pending not in (0, None) and time.time() < deadline:
        time.sleep(1)
        pending = stream_pending()
    f.check(pending == 0,
            f"the archive catches up after restart — nothing lost (pending={pending})")


def scenario_engine_down(f: Failures) -> None:
    print("\n== the decision engine degrades only decisions (ADR-0005, §5) ==")
    compose("stop", "decision-engine")
    try:
        token = session()
        f.check(token != "", "a session still opens")
        st, _, _ = telemetry(token)
        f.check(st == 202, f"telemetry is still accepted ({st})")

        st, _, _ = post(ENGINE, "/v1/decisions",
                        wire.decision_body(token), bearer=SECRET, timeout=5)
        f.check(st == UNREACHABLE, f"the engine itself is gone ({st})")

        # The all-in-one binary mounts the same package over its own
        # session store, which is the rollback the phase gate may need.
        st, _, _ = post(COLLECTOR, "/v1/decisions",
                        wire.decision_body(token), bearer=SECRET)
        f.check(st == 200, f"the collector still answers a decision ({st})")

        st, _, out = post(DEMO, "/demo/login", {"session_token": token, "username": "alice"})
        f.check(st == 200 and out.get("mode") == "fail-open",
                f"the demo host fails OPEN rather than breaking "
                f"({st}, mode={out.get('mode')})")
    finally:
        compose("start", "decision-engine")
        healthy(ENGINE)


def scenario_broker_down(f: Failures) -> None:
    print("\n== the broker is the only durable store (ADR-0006) ==")
    compose("stop", "nats")
    try:
        st, dt, out = post(COLLECTOR, "/v1/sessions", wire.session_body(SITE))
        f.check(st == 200, f"a session still opens ({st})")
        f.check(dt < SESSIONS_BUDGET_S,
                f"...and does not stall on the broker ({dt:.2f}s < {SESSIONS_BUDGET_S}s)")
        token = out.get("session_token", "")

        st, dt, _ = telemetry(token)
        f.check(st == 202, f"telemetry is still accepted ({st})")
        f.check(dt < TELEMETRY_BUDGET_S,
                f"...and does not stall on the broker ({dt:.2f}s < {TELEMETRY_BUDGET_S}s)")

        st, _, _ = post(COLLECTOR, "/v1/decisions",
                        wire.decision_body(token), bearer=SECRET)
        f.check(st == 200, f"the collector decides from memory ({st})")

        st, _, _ = post(COLLECTOR, "/v1/outcomes",
                        wire.outcome_body("ev_killtest", "login_success"),
                        bearer=SECRET)
        f.check(st in (500, 503),
                f"an outcome REFUSES rather than lying about durability ({st})")

        st, _, _ = post(ENGINE, "/v1/decisions",
                        wire.decision_body(token), bearer=SECRET)
        f.check(st == 500,
                f"a broken snapshot store is an error, not a cold start ({st}) — "
                f"scoring it innocent would fail open at the moment evidence stops")
    finally:
        compose("start", "nats")
        healthy(COLLECTOR)
        healthy(ENGINE)


def main() -> int:
    for name, base in (("collector", COLLECTOR), ("decision engine", ENGINE),
                       ("demo host", DEMO)):
        try:
            with urllib.request.urlopen(base + "/healthz", timeout=3) as r:
                if r.status != 200:
                    raise OSError
        except Exception:
            print(f"kill-test: the {name} at {base} is not answering /healthz.",
                  file=sys.stderr)
            print("           This test would report nothing, which is not a pass.",
                  file=sys.stderr)
            print("fix: docker compose --profile core up -d", file=sys.stderr)
            return 1

    f = Failures()
    scenario_archive_down(f)
    scenario_engine_down(f)
    scenario_broker_down(f)

    print()
    if f:
        for what in f:
            print(f"  FAILED: {what}", file=sys.stderr)
        print(f"\n{len(f)} degradation promise(s) did not hold.", file=sys.stderr)
        return 1
    print("  every degradation promise held")
    return 0


if __name__ == "__main__":
    sys.exit(main())
