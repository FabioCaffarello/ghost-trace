#!/usr/bin/env python3
"""Every HTTP route a service serves is named in the contract.

`contract/architecture.md` §3 opens with "Four endpoints. This section
is the contract's core." Four is the count of endpoints that carry the
product; it is not the count of routes the binaries answer. They answer
nine, and until PR-6.4 two of the extra five appeared in no contract
section at all.

`GET /metrics` was the interesting one. `SECURITY.md` disclosed it
honestly — "every service also serves unauthenticated /healthz and
/metrics" — while the document that claims to define the external
surface did not mention it. A reader auditing the surface from the
contract would have missed an endpoint publishing the tenant-registry
fingerprint and the number of customers.

The fix for that is prose. The fix for it happening again is this: every
route registered anywhere under `services/` must appear below, with the
section that documents it. A new route fails until somebody says where a
reader would learn about it.

This is a source scan rather than a runtime one on purpose. Go's
`ServeMux` does not expose its patterns, so the alternative would be to
boot four servers and probe them — which would only find the routes the
prober already thought to ask for.
"""

from __future__ import annotations

import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parent.parent

# Where a reader learns about each route. The value is not decoration:
# it is the claim this script enforces, and "nowhere" is not a value.
DOCUMENTED = {
    "POST /v1/sessions":     "architecture §3",
    "POST /v1/telemetry":    "architecture §3",
    "POST /v1/decisions":    "architecture §3",
    "POST /v1/outcomes":     "architecture §3",
    "OPTIONS /v1/sessions":  "architecture §1 — CORS preflight for the browser endpoints",
    "OPTIONS /v1/telemetry": "architecture §1 — CORS preflight for the browser endpoints",
    "GET /healthz":          "architecture §3 — unauthenticated liveness",
    "GET /metrics":          "architecture §3 — unauthenticated, and what it exposes",
    "GET /sdk.js":           "architecture §6 — served by the collector, not the demo host",
    "GET /":                 "architecture §6 — the demo login page",
    "POST /demo/login":      "architecture §6 — the stand-in application server",
}

# A route pattern is a string literal whose first word is an HTTP method.
# Anchored at the quote so a method name inside prose does not match.
ROUTE = re.compile(r'"((?:GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS) /[^"]*)"')

# libs/ is scanned too, and not as a catch-all: ADR-0005 puts the
# decision endpoints and their HANDLERS in a shared module, because two
# services serve them and a second copy would be a second contract.
# Scanning only services/ missed POST /v1/decisions entirely.
SCAN = ("services", "libs")

COMMENT = re.compile(r"//[^\n]*|/\*.*?\*/", re.S)


def routes() -> dict[str, list[str]]:
    """Every route literal under services/, and where it was found."""
    found: dict[str, list[str]] = {}
    for top in SCAN:
        for path in sorted((ROOT / top).rglob("*.go")):
            if path.name.endswith("_test.go"):
                continue
            # Comments stripped, then the whole file scanned rather
            # than only lines containing `Handle(`. Registrations are not
            # always on the call line: the CORS preflights are literals
            # inside a `[]string` that a loop hands to the mux, and a
            # line-local filter silently missed both of them.
            text = COMMENT.sub("", path.read_text())
            for m in ROUTE.finditer(text):
                found.setdefault(m.group(1), []).append(
                    str(path.relative_to(ROOT)))
    return found


def check(found: dict[str, list[str]]) -> list[str]:
    problems = []
    for route, where in sorted(found.items()):
        if route not in DOCUMENTED:
            problems.append(
                f"{route} is served by {', '.join(sorted(set(where)))} and no "
                f"contract section documents it — add it to the contract and "
                f"to DOCUMENTED, or stop serving it")
    for route in sorted(DOCUMENTED):
        if route not in found:
            problems.append(
                f"{route} is listed as documented but nothing registers it — "
                f"the inventory outlived the route")
    return problems


def _selftest() -> int:
    every = {r: ["x.go"] for r in DOCUMENTED}
    assert check(every) == [], check(every)

    undocumented = dict(every, **{"GET /debug/pprof/": ["services/x/main.go"]})
    got = check(undocumented)
    assert len(got) == 1 and "pprof" in got[0] and "no contract section" in got[0], got

    fossil = {r: ["x.go"] for r in DOCUMENTED if r != "GET /metrics"}
    got = check(fossil)
    assert len(got) == 1 and "outlived the route" in got[0], got

    # The pattern matches a route literal and not a bare path, so a
    # string like "/v1/sessions" in a client does not register anything.
    assert ROUTE.findall('mux.Handle("GET /metrics", h)') == ["GET /metrics"]
    assert ROUTE.findall('url := "/v1/sessions"') == []
    # Nor a method name that merely appears inside a longer word.
    assert ROUTE.findall('"REPOST /v1/x"') == []
    # A method with no path is not a route.
    assert ROUTE.findall('"GET"') == []

    print("  7 pass")
    return 0


def main() -> int:
    if "--selftest" in sys.argv:
        return _selftest()
    found = routes()
    problems = check(found)
    if problems:
        print(f"  {len(problems)} problem(s):")
        for p in problems:
            print(f"    - {p}")
        return 1
    print(f"  {len(found)} route(s) registered under {'/ and '.join(SCAN)}/, "
          f"every one named in the contract")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
