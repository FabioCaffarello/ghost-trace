"""The ONE place the Python side of the harness builds request bodies.

The JS counterpart is lib/wire.js and the reasoning is the same: before
R1.15 every producer hand-rolled its JSON, outside all buf discipline,
and the server tolerates unknown fields by design (§5, §7) — so
renaming a wire field would have left the producers sending the old
name, the server zero-filling the new one, and the measurements quietly
degrading with everything green. That is audit finding M22.

`make contract-fixtures` emits what these functions produce into
contract/fixtures/, where a Go test validates them against the
published OpenAPI request schemas and replays them against a real
server. The request schemas set `additionalProperties: false`, so a
field the server no longer knows about fails — and so does one it
expects and never receives.

Stdlib only, like everything else on this side.
"""


def session_body(site_key, path="/login", pointer="fine", touch=False,
                 viewport=(1440, 900), tz_offset=-180, reduced_motion=False):
    """POST /v1/sessions"""
    return {
        "site_key": site_key,
        "page": {"path": path},
        "client": {
            "pointer": pointer,
            "touch": touch,
            "viewport": list(viewport),
            "tz_offset": tz_offset,
            "reduced_motion": reduced_motion,
        },
    }


def telemetry_body(session_token, seq, sent_at_ms, events,
                   path="/login", viewport=(1440, 900)):
    """POST /v1/telemetry"""
    return {
        "session_token": session_token,
        "seq": seq,
        "sent_at_ms": sent_at_ms,
        "page": {"path": path, "viewport": list(viewport)},
        "events": events,
    }


def pointer_event(t, pts, src="mouse"):
    """A pointer event. `pts` are [x, y, dt_ms] triples — the third
    element is the gap since the previous sample, not an absolute
    time, which is the detail a hand-rolled producer gets wrong."""
    return {"type": "pointer", "t": t, "src": src, "pts": pts}


def key_event(t, phase, key_class, target):
    """A keystroke event. Timing and a COARSE class only, never the key
    itself — §2 and §6. There is deliberately no parameter here that
    could carry a character."""
    return {"type": "key", "t": t, "phase": phase, "class": key_class, "target": target}


def scroll_event(t, dy, mode="wheel"):
    """A scroll event. `mode` separates a real wheel gesture from the
    page scrolling itself — a programmatic scroll is one of the
    stronger automation signals on its own."""
    return {"type": "scroll", "t": t, "dy": dy, "mode": mode}


def focus_event(t, state, target):
    """A focus event — focus or blur, with the hashed field identity.
    Never the field's value."""
    return {"type": "focus", "t": t, "state": state, "target": target}


def visibility_event(t, state):
    """A visibility event — visible or hidden, straight from
    document.visibilityState."""
    return {"type": "visibility", "t": t, "state": state}


def form_event(t, action, target):
    """A form event. `injected` is the strongest single bot signal — a
    field value that appeared with no keystroke and no paste behind it
    — and `paste` exists so a human pasting is NOT mistaken for one.
    The strongest signal in the system went without a contract fixture
    until these four families existed; that is the wrong place to be
    uncovered."""
    return {"type": "form", "t": t, "action": action, "target": target}


def decision_body(session_token, action="login", subject_id="harness"):
    """POST /v1/decisions"""
    return {
        "session_token": session_token,
        "action": action,
        "subject_id": subject_id,
    }


def outcome_body(evaluation_id, outcome, observed_at=None):
    """POST /v1/outcomes — the labels channel.

    Nothing in the harness calls this yet, which is itself a finding:
    it is the channel every future calibration depends on and it has no
    client at all. The shape is defined here so the contract covers it
    and the first caller does not have to invent it."""
    body = {"evaluation_id": evaluation_id, "outcome": outcome}
    if observed_at is not None:
        body["observed_at"] = observed_at
    return body
